package ui

import (
	"fmt"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 爆弾の操作と表示（DES-NG-05、NG-FR-04）。

// usableItem は、アイテムの使用を行動の一覧に出せるかを返す（DES-NG-05）。
// 爆弾があれば、HP が最大でも出す。回復薬だけなら、今までどおり HP が最大でないときだけ出す。
func usableItem(u tactics.Unit) bool {
	for _, s := range u.Items {
		if s.Item.Bomb {
			return true
		}
	}
	return len(u.Items) > 0 && u.HP < u.Stats.HP
}

// bombItem は、爆心を選んでいる爆弾を返す。BombTargeting でなければ ok が偽。
func (c *Controller) bombItem() (tactics.Item, bool) {
	if c.state != BombTargeting {
		return tactics.Item{}, false
	}
	items := c.battle.Units()[c.unit].Items
	if c.item < 0 || c.item >= len(items) || !items[c.item].Item.Bomb {
		return tactics.Item{}, false
	}
	return items[c.item].Item, true
}

// throwCells は、移動先から爆弾を投げられるマス（マップの中で、移動先から ThrowRange 以内）を返す（DES-NG-05）。
// 地形は問わない。範囲の計算は core の CellsWithin に任せる。
func (c *Controller) throwCells() []tactics.Pos {
	it, ok := c.bombItem()
	if !ok {
		return nil
	}
	return c.battle.CellsWithin(c.dest, it.ThrowRange)
}

// blastCommand は、仮選択した爆心で爆弾を使うコマンドを返す。
func (c *Controller) blastCommand(center tactics.Pos) tactics.Command {
	return tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.UseItem, Item: c.item, Blast: &center}
}

// clickBlast は、爆心を選んでいるときのマス p のクリックの処理（DES-NG-05）。
// 投げられる範囲の外は理由を知らせて受け付けない。範囲の中なら、1 回目で仮選択し、同じマスをもう一度クリックすると決定する。
func (c *Controller) clickBlast(p tactics.Pos) {
	it, ok := c.bombItem()
	if !ok {
		return
	}
	if distance(c.dest, p) > it.ThrowRange {
		c.message = fmt.Sprintf("そのマスは投げられる範囲（%d マス）の外です", it.ThrowRange)
		return
	}
	if c.blastPicked && c.blast == p {
		c.act(c.blastCommand(p))
		return
	}
	c.blast, c.blastPicked, c.message = p, true, ""
}

// BlastOverlay は、仮選択した爆心の爆発の範囲のマスを返す（DES-NG-05）。選んでいなければ nil。
func (c *Controller) BlastOverlay() map[tactics.Pos]bool {
	f, ok := c.BombForecast()
	if !ok {
		return nil
	}
	out := map[tactics.Pos]bool{}
	for _, p := range c.battle.CellsWithin(f.Center, f.Radius) {
		out[p] = true
	}
	return out
}

// BombForecast は、仮選択した爆心で使ったときの爆発の予想を返す（DES-NG-03、DES-NG-05）。
// 爆心を選んでいない、または core が拒むときは ok が偽。
func (c *Controller) BombForecast() (tactics.BlastForecast, bool) {
	if c.state != BombTargeting || !c.blastPicked || c.rewind.open {
		return tactics.BlastForecast{}, false
	}
	f, err := c.battle.BlastForecast(c.blastCommand(c.blast))
	return f, err == nil
}

// BombLabel は、爆発に巻き込まれるユニットの上に出す予想の文字（DES-NG-05）。
type BombLabel struct {
	Unit  int
	Pos   tactics.Pos
	Text  string
	Style InfoStyle
	// Ally は自軍（使う人を含む）を巻き込むこと。枠を赤で描く。
	Ally bool
}

// bombLabelText は、巻き込まれる 1 体の予想の文を返す。「敵」「味方」「自分」と、与えるダメージと、倒れるかを書く。
// 例: 「敵 -12 撃破」「味方 -12 HP 20→8」「自分 -12 倒れる」。
func bombLabelText(t tactics.BlastTarget, self bool, damage int) string {
	who := "敵"
	switch {
	case self:
		who = "自分"
	case t.Side == tactics.Player:
		who = "味方"
	}
	if t.Falls {
		verb := "撃破"
		if t.Side == tactics.Player {
			verb = "倒れる"
		}
		return fmt.Sprintf("%s -%d %s", who, damage, verb)
	}
	return fmt.Sprintf("%s -%d HP %d→%d", who, damage, t.HP, t.After)
}

// BombLabels は、爆発の予想を、巻き込まれるユニットごとに返す（DES-NG-05）。使う人は移動先の位置に出す。
// 自軍（味方・自分）は赤（StyleBad）、敵は緑（StyleGood）にして、味方を巻き込むことが一目で分かるようにする。
func (c *Controller) BombLabels() []BombLabel {
	f, ok := c.BombForecast()
	if !ok {
		return nil
	}
	units := c.Units()
	var out []BombLabel
	for _, t := range f.Targets {
		ally := t.Side == tactics.Player
		style := StyleGood
		if ally {
			style = StyleBad
		}
		out = append(out, BombLabel{Unit: t.Unit, Pos: units[t.Unit].Pos, Text: bombLabelText(t, t.Unit == c.unit, f.Damage), Style: style, Ally: ally})
	}
	return out
}

// bombSummary は、爆発の予想の要約（敵・味方・自分の数と、倒れる数）を返す（DES-NG-05）。
func bombSummary(f tactics.BlastForecast, user int) (line string, allies int) {
	enemies, fallEnemies, fallAllies, self := 0, 0, 0, false
	for _, t := range f.Targets {
		switch {
		case t.Unit == user:
			self = true
			allies++
			if t.Falls {
				fallAllies++
			}
		case t.Side == tactics.Player:
			allies++
			if t.Falls {
				fallAllies++
			}
		default:
			enemies++
			if t.Falls {
				fallEnemies++
			}
		}
	}
	line = fmt.Sprintf("敵 %d 体（撃破 %d）", enemies, fallEnemies)
	if allies > 0 {
		who := fmt.Sprintf("味方 %d 体", allies)
		if self {
			who = fmt.Sprintf("自分を含む味方 %d 体", allies)
		}
		line += fmt.Sprintf("　%sを巻き込む（倒れる %d）", who, fallAllies)
	}
	return line, allies
}

// bombGuide は爆心を選んでいる間の案内欄の行を返す（DES-NG-05）。
func (c *Controller) bombGuide() []string {
	f, ok := c.BombForecast()
	if !ok {
		return []string{"左クリック: 爆心を選ぶ（赤いマスまで投げられる）　右クリック・Esc: 戻る"}
	}
	line, _ := bombSummary(f, c.unit)
	return []string{line, "もう一度クリックで投げる　右クリック・Esc: 爆心を選び直す"}
}

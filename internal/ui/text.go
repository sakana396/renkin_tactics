package ui

import (
	"fmt"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面に出す文はすべて日本語にする（IF-02）。

// sideName は陣営の名前を返す。
func sideName(s tactics.Side) string {
	if s == tactics.Player {
		return "自軍"
	}
	return "敵軍"
}

// actionName は行動の一覧に出す名前を返す（DES-UI-02）。
func actionName(a tactics.Action) string {
	switch a {
	case tactics.Attack:
		return "攻撃"
	case tactics.Heal:
		return "回復"
	case tactics.UseItem:
		return "アイテム"
	default:
		return "待機"
	}
}

// itemLabel はアイテムの一覧に出す文を返す（DES-UI-02）。
func itemLabel(s tactics.ItemStack) string {
	return fmt.Sprintf("%s ×%d（回復 %d）", s.Item.Name, s.Count, s.Item.Heal)
}

// weaponDetail は武器・杖の詳しい欄の 1 行（DES-UI-26、DES-IT06 12.1 節）。
// 名前・物理／魔法・威力・命中・必殺・重さ・射程・特効の対象。杖は回復量・射程・残りの回数。
func weaponDetail(w tactics.Weapon) string {
	if w.Type == tactics.Staff {
		return fmt.Sprintf("%s　回復 %d　射程 %s　残り %d/%d", w.Name, w.Heal, rangeText(w), w.Uses, w.MaxUses)
	}
	kind := "物理"
	if w.DamageType == tactics.Magical {
		kind = "魔法"
	}
	eff := "なし"
	if len(w.EffectiveTags) > 0 {
		eff = strings.Join(w.EffectiveTags, "・")
	}
	return fmt.Sprintf("%s　%s　威力 %d　命中 %d　必殺 %d　重さ %d　射程 %s　特効 %s", w.Name, kind, w.Might, w.Hit, w.Crit, w.Weight, rangeText(w), eff)
}

// staffsEmpty は、杖を持ち、そのすべての回数が残っていないかを返す（DES-UI-18）。
func staffsEmpty(u tactics.Unit) bool {
	has := false
	for _, w := range u.Weapons {
		if w.Type != tactics.Staff {
			continue
		}
		if w.Uses > 0 {
			return false
		}
		has = true
	}
	return has
}

// BandText は番の帯に出す文を返す（DES-UI-03）。PhaseStart でなければ空。
func BandText(e tactics.Event) string {
	if e.Kind != tactics.EventPhaseStart {
		return ""
	}
	return fmt.Sprintf("ターン %d　%sの番", e.Turn, sideName(e.Side))
}

// StrikeText は、1 撃の結果として相手の上に出す文を返す（DES-UI-04）。
// 当たり・外れ・ダメージ・必殺・大盾を出す。Strike でなければ空。
func StrikeText(e tactics.Event) string {
	if e.Kind != tactics.EventStrike {
		return ""
	}
	if !e.Hit {
		return "外れ"
	}
	s := fmt.Sprintf("%d ダメージ", e.Amount)
	if e.Crit {
		s = "必殺！ " + s
	}
	if e.Pavise {
		s = "大盾！ " + s
	}
	return s
}

// EventText は、出来事を案内欄に出す 1 行の文にする（DES-UI-04）。名前は出来事の写しから引く。
func EventText(e tactics.Event) string {
	name := func(i int) string {
		if i >= 0 && i < len(e.Units) {
			return e.Units[i].Name
		}
		return "？"
	}
	switch e.Kind {
	case tactics.EventPhaseStart:
		return BandText(e)
	case tactics.EventReinforce:
		return fmt.Sprintf("%s が増援で現れた", name(e.Unit))
	case tactics.EventHeal:
		where := "砦"
		if e.Item != "" {
			where = e.Item
		}
		return fmt.Sprintf("%s は%sで HP が %d 回復した", name(e.Unit), where, e.Amount)
	case tactics.EventMove:
		return fmt.Sprintf("%s が移動した", name(e.Unit))
	case tactics.EventStrike:
		if !e.Hit {
			return fmt.Sprintf("%s の攻撃 → %s：外れ", name(e.Unit), name(e.Target))
		}
		return fmt.Sprintf("%s の攻撃 → %s：%s", name(e.Unit), name(e.Target), StrikeText(e))
	case tactics.EventFall:
		return fmt.Sprintf("%s は倒れた", name(e.Unit))
	case tactics.EventWait:
		return fmt.Sprintf("%s は待機した", name(e.Unit))
	case tactics.EventExp:
		return fmt.Sprintf("%s は経験を %d 得た", name(e.Unit), e.Amount)
	case tactics.EventLevelUp:
		gains := growthText(e.Gains)
		if gains == "" {
			gains = "上がった能力なし"
		}
		return fmt.Sprintf("%s　レベル %d　%s", name(e.Unit), e.Level, gains)
	case tactics.EventTrigger:
		// 出来事の ID は戦闘の中で比べるだけの識別子なので、画面には出さない（DES-MAP-07）。
		return "出来事が起きた"
	case tactics.EventGroupCharge:
		// 一団の名前は識別子で、表示名を持たない（DES-MAP-07）。
		return "敵の一団が動き出した"
	case tactics.EventWon:
		return "ボスを倒した。勝ち！"
	case tactics.EventLost:
		return "主人公が倒れた。負け……"
	}
	return ""
}

// ResultText は勝ち負けの画面の見出しを返す（DES-UI-02、UC-07）。
func ResultText(r tactics.Result) string {
	switch r {
	case tactics.Won:
		return "勝利"
	case tactics.Lost:
		return "敗北"
	}
	return ""
}

// 案内欄の敵の範囲の操作の文（DES-UI-08）。
const (
	guideEnemyRange = "敵にカーソル: その敵の範囲　Q: 敵の危険範囲"
	guideDanger     = "Q: 敵の危険範囲"
)

// Guide は案内欄に出す行を返す（DES-UI-01、DES-UI-08）。今の番とターン、できる操作、敵の範囲の操作、知らせの順に並べる。
func (c *Controller) Guide() []string {
	b := c.battle
	lines := []string{fmt.Sprintf("ターン %d　%sの番", c.shownTurn, sideName(c.shownSide))}
	switch c.state {
	case Idle:
		lines = append(lines, "左クリック: 自軍を選ぶ・敵の範囲を出し続ける（もう一度で消す）　E: 番を終える", guideEnemyRange)
	case Selected:
		lines = append(lines, "左クリック: 青いマスに移動する　右クリック・Esc: 取り消す", guideEnemyRange)
	case Moved:
		lines = append(lines, "左クリック: 行動を選ぶ　右クリック・Esc: 移動を取り消す", guideEnemyRange)
	case Targeting:
		if c.target >= 0 {
			lines = append(lines, fmt.Sprintf("%s を攻撃する。もう一度クリックで決定　右クリック・Esc: 戻る", b.Units()[c.target].Name))
		} else {
			lines = append(lines, "左クリック: 攻撃する敵を選ぶ　右クリック・Esc: 戻る")
		}
		lines = append(lines, guideDanger)
	case HealTargeting:
		if c.target >= 0 {
			lines = append(lines, fmt.Sprintf("%s を回復する。もう一度クリックで決定　右クリック・Esc: 戻る", b.Units()[c.target].Name))
		} else {
			lines = append(lines, "左クリック: 回復する味方を選ぶ　右クリック・Esc: 戻る")
		}
		lines = append(lines, guideDanger)
	case ItemMenu:
		lines = append(lines, "左クリック: 使うアイテムを選ぶ　右クリック・Esc: 戻る", guideDanger)
	case WeaponMenu, StaffMenu:
		what := "攻撃に使う武器"
		if c.state == StaffMenu {
			what = "回復に使う杖"
		}
		lines = append(lines, fmt.Sprintf("左クリック: %sを選ぶ　右クリック・Esc: 戻る", what))
		// 詳しい欄: カーソルの下の行の武器の値（DES-IT06 12.1 節）。
		if w := c.hoveredWeapon(); w >= 0 {
			lines = append(lines, weaponDetail(b.Units()[c.unit].Weapons[w]))
		}
	case Playing:
		// PhaseStart は 1 行目と同じ文になるので出さない。
		if e, _, ok := c.Event(); ok && e.Kind != tactics.EventPhaseStart {
			lines = append(lines, EventText(e))
		}
		lines = append(lines, "Space・Enter（押している間）: 早送り")
		return lines
	case Over:
		lines = append(lines, fmt.Sprintf("%s　左クリック: 次へ", ResultText(b.Result)))
	}
	// 杖を使い切った理由を、回復が一覧にないときも伝える（DES-UI-18）。
	if c.state == Moved && staffsEmpty(b.Units()[c.unit]) && c.message == "" {
		lines = append(lines, tactics.ErrNoUses.Error())
	}
	if c.message != "" {
		lines = append(lines, c.message)
	}
	return lines
}

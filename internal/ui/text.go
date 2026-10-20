package ui

import (
	"fmt"

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
		return "勝ち！"
	case tactics.Lost:
		return "負け……"
	}
	return ""
}

// Guide は案内欄に出す行を返す（DES-UI-01）。今の番とターン、できる操作、知らせの順に並べる。
func (c *Controller) Guide() []string {
	b := c.battle
	lines := []string{fmt.Sprintf("ターン %d　%sの番", c.shownTurn, sideName(c.shownSide))}
	switch c.state {
	case Idle:
		lines = append(lines, "左クリック: 自軍を選ぶ（敵なら範囲を出す・消す）　E: 番を終える")
	case Selected:
		lines = append(lines, "左クリック: 青いマスに移動する　右クリック・Esc: 取り消す")
	case Moved:
		lines = append(lines, "左クリック: 行動を選ぶ　右クリック・Esc: 移動を取り消す")
	case Targeting:
		if c.target >= 0 {
			lines = append(lines, fmt.Sprintf("%s を攻撃する。もう一度クリックで決定　右クリック・Esc: 戻る", b.Units()[c.target].Name))
		} else {
			lines = append(lines, "左クリック: 攻撃する敵を選ぶ　右クリック・Esc: 戻る")
		}
	case ItemMenu:
		lines = append(lines, "左クリック: 使うアイテムを選ぶ　右クリック・Esc: 戻る")
	case Playing:
		// PhaseStart は 1 行目と同じ文になるので出さない。
		if e, _, ok := c.Event(); ok && e.Kind != tactics.EventPhaseStart {
			lines = append(lines, EventText(e))
		}
		lines = append(lines, "Space・Enter: 早送り")
		return lines
	case Over:
		lines = append(lines, fmt.Sprintf("%s　R: やり直す　Esc: 終了", ResultText(b.Result)))
	}
	if c.message != "" {
		lines = append(lines, c.message)
	}
	return lines
}

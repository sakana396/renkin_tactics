package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面に出す文が日本語であることを確かめる（IF-02、DES-UI-03、DES-UI-04）。
func TestTexts(t *testing.T) {
	units := []tactics.Unit{{Name: "レオン"}, {Name: "ソルジャー 3"}}
	ev := func(e tactics.Event) tactics.Event {
		e.Units = units
		return e
	}

	t.Run("TC-IF-02-01 出来事の文: 番の帯、当たり・外れ・ダメージ・必殺・大盾、回復、増援、倒れた、勝ち負け", func(t *testing.T) {
		cases := []struct {
			e          tactics.Event
			text, pop  string
			band       string
			resultText string
		}{
			{e: ev(tactics.Event{Kind: tactics.EventPhaseStart, Side: tactics.Player, Turn: 1}), text: "ターン 1　自軍の番", band: "ターン 1　自軍の番"},
			{e: ev(tactics.Event{Kind: tactics.EventPhaseStart, Side: tactics.Enemy, Turn: 3}), text: "ターン 3　敵軍の番", band: "ターン 3　敵軍の番"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 1, Target: 0, Hit: true, Amount: 9}), text: "ソルジャー 3 の攻撃 → レオン：9 ダメージ", pop: "9 ダメージ"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 0, Target: 1, Hit: false}), text: "レオン の攻撃 → ソルジャー 3：外れ", pop: "外れ"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 0, Target: 1, Hit: true, Amount: 39, Crit: true}), text: "レオン の攻撃 → ソルジャー 3：必殺！ 39 ダメージ", pop: "必殺！ 39 ダメージ"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 1, Target: 0, Hit: true, Amount: 4, Pavise: true}), text: "ソルジャー 3 の攻撃 → レオン：大盾！ 4 ダメージ", pop: "大盾！ 4 ダメージ"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 0, Target: 1, Hit: true, Amount: 19, Crit: true, Pavise: true}), text: "レオン の攻撃 → ソルジャー 3：大盾！ 必殺！ 19 ダメージ", pop: "大盾！ 必殺！ 19 ダメージ"},
			{e: ev(tactics.Event{Kind: tactics.EventStrike, Unit: 1, Target: 0, Hit: true, Amount: 0}), text: "ソルジャー 3 の攻撃 → レオン：0 ダメージ", pop: "0 ダメージ"},
			{e: ev(tactics.Event{Kind: tactics.EventHeal, Unit: 1, Amount: 5}), text: "ソルジャー 3 は砦で HP が 5 回復した"},
			{e: ev(tactics.Event{Kind: tactics.EventHeal, Unit: 0, Amount: 10, Item: "回復薬"}), text: "レオン は回復薬で HP が 10 回復した"},
			{e: ev(tactics.Event{Kind: tactics.EventReinforce, Unit: 1}), text: "ソルジャー 3 が増援で現れた"},
			{e: ev(tactics.Event{Kind: tactics.EventMove, Unit: 0}), text: "レオン が移動した"},
			{e: ev(tactics.Event{Kind: tactics.EventWait, Unit: 0}), text: "レオン は待機した"},
			{e: ev(tactics.Event{Kind: tactics.EventFall, Unit: 1}), text: "ソルジャー 3 は倒れた"},
			{e: ev(tactics.Event{Kind: tactics.EventWon}), text: "ボスを倒した。勝ち！"},
			{e: ev(tactics.Event{Kind: tactics.EventLost}), text: "主人公が倒れた。負け……"},
		}
		for _, tc := range cases {
			if got := EventText(tc.e); got != tc.text {
				t.Errorf("EventText(%v) = %q、%q のはず", tc.e.Kind, got, tc.text)
			}
			if got := StrikeText(tc.e); got != tc.pop {
				t.Errorf("StrikeText(%v) = %q、%q のはず", tc.e.Kind, got, tc.pop)
			}
			if got := BandText(tc.e); got != tc.band {
				t.Errorf("BandText(%v) = %q、%q のはず", tc.e.Kind, got, tc.band)
			}
		}
		if ResultText(tactics.Won) != "勝利" || ResultText(tactics.Lost) != "敗北" || ResultText(tactics.Ongoing) != "" {
			t.Errorf("ResultText が違う")
		}
		// 写しにないユニットを指していても止まらない。
		if got := EventText(tactics.Event{Kind: tactics.EventFall, Unit: 5}); got != "？ は倒れた" {
			t.Errorf("写しにないユニット: %q", got)
		}
	})

	t.Run("TC-IF-02-02 案内欄: 今の番とターン、状態ごとの操作、敵の範囲の操作、知らせを日本語で出す", func(t *testing.T) {
		// 案内欄の行は、盤面の幅から左右の余白 8 px を除いた幅に収まる（DES-UI-01）。
		fits := func(lines []string) {
			t.Helper()
			for _, l := range lines {
				if w := textWidth(l); w > MapSize-16 {
					t.Errorf("%q の幅 %d px が案内欄に収まらない", l, w)
				}
			}
			if len(lines) > GuideHeight/lineHeight {
				t.Errorf("行の数 %d が案内欄に収まらない: %q", len(lines), lines)
			}
		}
		const enemyRange = "敵にカーソル: その敵の範囲　Q: 敵の危険範囲"
		c := newIdle(t)
		want := []string{"ターン 1　自軍の番", "左クリック: 自軍を選ぶ・敵の範囲を出し続ける（もう一度で消す）　E: 番を終える", enemyRange}
		if got := c.Guide(); !reflect.DeepEqual(got, want) {
			t.Errorf("Idle = %q", got)
		}
		c.EndPhase()
		if got := c.Guide(); len(got) != 4 || got[2] != enemyRange || got[3] != "まだ行動していない自軍がいます。もう一度 E を押すと番を終えます" {
			t.Errorf("確認 = %q", got)
		}
		fits(c.Guide())
		c.Cancel()
		c.Click(tactics.Pos{X: 0, Y: 0})
		if got := c.Guide(); !reflect.DeepEqual(got[1:], []string{"左クリック: 青いマスに移動する　右クリック・Esc: 取り消す", enemyRange}) {
			t.Errorf("Selected = %q", got)
		}
		c.Click(tactics.Pos{X: 0, Y: 0})
		if got := c.Guide(); !reflect.DeepEqual(got[1:], []string{"左クリック: 行動を選ぶ　右クリック・Esc: 移動を取り消す", enemyRange}) {
			t.Errorf("Moved = %q", got)
		}
		c.ChooseAction(tactics.Attack)
		if got := c.Guide(); !reflect.DeepEqual(got[1:], []string{"左クリック: 攻撃する敵を選ぶ　右クリック・Esc: 戻る", "Q: 敵の危険範囲"}) {
			t.Errorf("Targeting = %q", got)
		}
		c.Click(tactics.Pos{X: 1, Y: 0})
		if got := c.Guide(); !reflect.DeepEqual(got[1:], []string{"ファイター を攻撃する。もう一度クリックで決定　右クリック・Esc: 戻る", "Q: 敵の危険範囲"}) {
			t.Errorf("Targeting（相手を選んだ） = %q", got)
		}
		fits(c.Guide())
		c.Click(tactics.Pos{X: 1, Y: 0})
		if got := c.Guide(); len(got) != 3 || got[0] != "ターン 1　自軍の番" || got[2] != "Space・Enter（押している間）: 早送り" {
			t.Errorf("Playing = %q", got)
		}
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if got := d.Guide(); !reflect.DeepEqual(got[1:], []string{"左クリック: 使うアイテムを選ぶ　右クリック・Esc: 戻る", "Q: 敵の危険範囲"}) {
			t.Errorf("ItemMenu = %q", got)
		}
		if got := d.MenuLabels(); !reflect.DeepEqual(got, []string{"回復薬 ×1（回復 10）"}) {
			t.Errorf("アイテムの一覧 = %q", got)
		}
	})
}

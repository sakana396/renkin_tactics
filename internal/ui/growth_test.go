package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestGrowthReplay(t *testing.T) {
	t.Run("TC-FR-UNT-04-15 経験20フレームとレベルアップ60フレームを勝利より先に再生する", func(t *testing.T) {
		c := newIdleWith(t, func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Level, s.Units[0].Exp = 1, 80
			s.Units[0].Growth = tactics.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}
			s.Units[3].Stats.HP, s.Units[3].Boss = 1, true
			s.Units[2].Boss = false
			return s
		})
		c.Click(tactics.Pos{})
		c.Click(tactics.Pos{})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1})
		c.Click(tactics.Pos{X: 1})
		// 剣と斧の相性により1ダメージで撃破。80+30=110、成長率100で各+1。
		for _, tc := range []struct {
			kind                   tactics.EventKind
			frames, level, exp, hp int
		}{
			{tactics.EventStrike, 20, 1, 80, 20}, {tactics.EventFall, 20, 1, 80, 20},
			{tactics.EventExp, 20, 2, 10, 21}, {tactics.EventLevelUp, 60, 2, 10, 21}, {tactics.EventWon, 20, 2, 10, 21},
		} {
			for frame := 0; frame < tc.frames; frame++ {
				e, gotFrame, ok := c.Event()
				if !ok || e.Kind != tc.kind || gotFrame != frame {
					t.Fatalf("出来事=%v frame=%d ok=%v、%v frame=%d のはず", e.Kind, gotFrame, ok, tc.kind, frame)
				}
				u := c.Panel(tactics.Pos{}, true).Unit
				if u == nil || u.Level != tc.level || u.Exp != tc.exp || u.Stats.HP != tc.hp || u.HP != tc.hp {
					t.Fatalf("再生の写し=%+v", u)
				}
				if tc.kind == tactics.EventExp || tc.kind == tactics.EventLevelUp {
					textOf(t, c.Panel(tactics.Pos{}, true).View(), "Lv 2　経験 10")
					if c.Guide()[1] != EventText(e) || c.Guide()[1] == "" {
						t.Fatal("案内欄に成長の文がない")
					}
				}
				c.Tick()
			}
		}
		if c.State() != Over {
			t.Fatalf("再生後=%v", c.State())
		}
	})
	t.Run("TC-FR-UNT-04-16 経験と成長の文・盤面の位置・能力なし", func(t *testing.T) {
		units := []tactics.Unit{{Name: "ブラン", Pos: tactics.Pos{X: 2, Y: 3}}}
		for _, tc := range []struct {
			e           tactics.Event
			text, popup string
			band        bool
		}{
			{tactics.Event{Kind: tactics.EventExp, Amount: 30}, "ブラン は経験を 30 得た", "+30 経験", false},
			{tactics.Event{Kind: tactics.EventExp, Amount: 1}, "ブラン は経験を 1 得た", "+1 経験", false},
			{tactics.Event{Kind: tactics.EventExp, Amount: 10}, "ブラン は経験を 10 得た", "+10 経験", false},
			{tactics.Event{Kind: tactics.EventLevelUp, Level: 2, Gains: tactics.Stats{HP: 1, Skl: 1, Def: 1}}, "ブラン　レベル 2　HP+1 技+1 守備+1", "ブラン　レベル 2　HP+1 技+1 守備+1", true},
			{tactics.Event{Kind: tactics.EventLevelUp, Level: 20}, "ブラン　レベル 20　上がった能力なし", "ブラン　レベル 20　上がった能力なし", true},
			{tactics.Event{Kind: tactics.EventLevelUp, Level: 2, Gains: tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1}}, "ブラン　レベル 2　HP+1 力+1 技+1 速さ+1 守備+1", "ブラン　レベル 2　HP+1 力+1 技+1 速さ+1 守備+1", true},
		} {
			tc.e.Units = units
			if got := EventText(tc.e); got != tc.text {
				t.Errorf("文=%q、%q のはず", got, tc.text)
			}
			v := GrowthEventView(tc.e)
			if !v.Visible || v.Text != tc.popup || v.Band != tc.band || (!v.Band && v.Pos != units[0].Pos) {
				t.Errorf("描画=%+v", v)
			}
			if textWidth(v.Text) > MapSize-16 {
				t.Errorf("帯の幅が盤面を超える: %q", v.Text)
			}
		}
		for _, e := range []tactics.Event{{Kind: tactics.EventWait}, {Kind: tactics.EventExp, Unit: 9}, {Kind: tactics.EventLevelUp, Unit: -1}} {
			if v := GrowthEventView(e); v.Visible {
				t.Errorf("不正または対象外の出来事に描画=%+v", v)
			}
		}
	})
	t.Run("TC-FR-UNT-04-17 早送りと再生途中の切り替えで出来事を飛ばさない", func(t *testing.T) {
		c := newIdle(t)
		es := []tactics.Event{{Kind: tactics.EventExp}, {Kind: tactics.EventLevelUp}, {Kind: tactics.EventWait}}
		c.play(es)
		c.SetFastForward(true)
		for i := range es {
			for f := 0; f < 4; f++ {
				e, frame, ok := c.Event()
				if !ok || e.Kind != es[i].Kind || frame != f {
					t.Fatalf("早送り=%+v frame=%d", e, frame)
				}
				c.Tick()
			}
		}
		if c.State() != Idle {
			t.Fatal("早送りが終わらない")
		}
		c.play(es[1:])
		c.SetFastForward(false)
		for i := 0; i < 30; i++ {
			c.Tick()
		}
		c.SetFastForward(true)
		c.Tick()
		e, frame, _ := c.Event()
		if e.Kind != tactics.EventWait || frame != 0 {
			t.Fatal("途中からの早送りが効かない")
		}
		c.play(es[1:])
		c.SetFastForward(true)
		c.Tick()
		c.Tick()
		c.SetFastForward(false)
		for i := 2; i < 59; i++ {
			c.Tick()
		}
		e, frame, _ = c.Event()
		if e.Kind != tactics.EventLevelUp || frame != 59 {
			t.Fatal("早送りを離したあと60フレームまで表示されない")
		}
		c.Tick()
		e, _, _ = c.Event()
		if e.Kind != tactics.EventWait {
			t.Fatal("通常速度に戻らない")
		}
	})
}

func TestLevelInfo(t *testing.T) {
	t.Run("TC-FR-UNT-09-14 名前とクラスの下に自軍のレベルと経験を表示する", func(t *testing.T) {
		for _, tc := range []struct {
			side       tactics.Side
			level, exp int
			want       string
		}{
			{tactics.Player, 2, 40, "Lv 2　経験 40"}, {tactics.Player, 1, 0, "Lv 1　経験 0"},
			{tactics.Player, 19, 99, "Lv 19　経験 99"}, {tactics.Player, 20, 0, "Lv 20　経験 ―"},
			{tactics.Enemy, 0, 0, ""}, {tactics.Player, 0, 0, ""},
		} {
			u := tactics.Unit{Name: "レオン", ClassName: "ロード", Side: tc.side, Level: tc.level, Exp: tc.exp, Stats: leonStats, HP: 20, Weapon: ironSword}
			v := (Panel{Unit: &u}).View()
			var got []string
			for _, tx := range v.Texts {
				if strings.HasPrefix(tx.Text, "Lv ") {
					got = append(got, tx.Text)
				}
			}
			var want []string
			if tc.want != "" {
				want = []string{tc.want}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("レベル行=%q、%q のはず", got, want)
			}
			if tc.want != "" {
				level, class, hp := textOf(t, v, tc.want), textOf(t, v, "クラス: ロード"), textOf(t, v, "HP 20/20")
				pic := portraitOf(t, v)
				if level.X != class.X || level.Y != class.Y+16 || level.Y+16 > pic.Y+64 || hp.Y < pic.Y+64 {
					t.Errorf("位置: level=%+v class=%+v hp=%+v", level, class, hp)
				}
			}
		}
	})
}

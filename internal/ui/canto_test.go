package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 再移動の画面の状態（DES-UI-26、DES-IT06 12.3 節、#124）。

// cantoController は 6 × 6 の平地で、騎兵 (0, 0)（移動 5・再移動）が (1, 0) へ動いて敵 (2, 0) を攻撃し、
// 結果の再生を終えて再移動を待つところまで進めた Controller を返す。費用 1 なので残りは 4。
// ダメージはどちらも 0 で、戦死しない。主人公 (0, 5) とボス (5, 5) は離れている。
func cantoController(t *testing.T, lordActed bool) *Controller {
	t.Helper()
	c := newIdleWith(t, func() tactics.Setup {
		return tactics.Setup{Terrain: field(6, 6), Units: []tactics.UnitSetup{
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 1}, Pos: tactics.Pos{X: 0, Y: 5}, Weapon: testSword, Lord: true},
			{Name: "エルザ", ClassName: "騎兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 5}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testLance,
				ClassSkills: []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}},
			{Name: "ソルジャー", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 2, Y: 0}, Weapon: testLance, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 5, Y: 5}, Weapon: testSword, Boss: true, AI: tactics.Hold},
		}}
	})
	if lordActed {
		c.Click(tactics.Pos{X: 0, Y: 5})
		c.Click(tactics.Pos{X: 0, Y: 5})
		c.ChooseAction(tactics.Wait)
		playOut(c)
	}
	c.Click(tactics.Pos{X: 0, Y: 0})
	c.Click(tactics.Pos{X: 1, Y: 0})
	c.ChooseAction(tactics.Attack)
	c.Click(tactics.Pos{X: 2, Y: 0})
	c.Click(tactics.Pos{X: 2, Y: 0})
	playOut(c)
	if c.State() != Canto {
		t.Fatalf("状態 = %v、Canto になるはず", c.State())
	}
	return c
}

func TestCantoUI(t *testing.T) {
	t.Run("TC-IF-01-40 行動の再生のあとに再移動の状態に入り、残りと行けるマスと案内を出す", func(t *testing.T) {
		c := cantoController(t, false)
		unit, dest, _ := c.Selection()
		if unit != 1 || dest != (tactics.Pos{X: 1, Y: 0}) {
			t.Errorf("選択 = %d・%v", unit, dest)
		}
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"終える"}) {
			t.Errorf("ボタン = %v", got)
		}
		blue, red := c.Overlay()
		want := map[tactics.Pos]bool{}
		for p := range c.Battle().CantoReachable() {
			want[p] = true
		}
		if len(want) == 0 || !reflect.DeepEqual(blue, want) || len(red) != 0 {
			t.Errorf("範囲 = 青 %v・赤 %v（青は %v）", blue, red, want)
		}
		guide := strings.Join(c.Guide(), "\n")
		for _, s := range []string{"再移動（残りの移動 4）", "その場で終える", "ほかのユニットの操作と番の終わりはできません"} {
			if !strings.Contains(guide, s) {
				t.Errorf("案内に %q がない: %s", s, guide)
			}
		}
		// 番を終える・ほかの自軍を選ぶ・行けないマスは受け付けない。
		c.EndPhase()
		c.Click(tactics.Pos{X: 0, Y: 5})
		c.Click(tactics.Pos{X: 5, Y: 0})
		if c.State() != Canto || c.Battle().Phase != tactics.Player {
			t.Errorf("状態 = %v・番 %v", c.State(), c.Battle().Phase)
		}
		// 行けるマスをクリックすると再移動する。
		c.Click(tactics.Pos{X: 1, Y: 3})
		playOut(c)
		if c.State() != Idle || c.Battle().Units()[1].Pos != (tactics.Pos{X: 1, Y: 3}) {
			t.Errorf("再移動のあと: 状態 %v・位置 %v", c.State(), c.Battle().Units()[1].Pos)
		}
		if _, ok := c.Battle().PendingCanto(); ok {
			t.Error("再移動のあとも待っている")
		}
	})

	t.Run("TC-IF-01-40 今の位置のクリック・「終える」・右クリック/Esc はその場で終え、攻撃は取り消さない", func(t *testing.T) {
		for name, end := range map[string]func(c *Controller){
			"今の位置": func(c *Controller) { c.Click(tactics.Pos{X: 1, Y: 0}) },
			"終える":  func(c *Controller) { c.ClickMenu(0) },
			"Esc":  func(c *Controller) { c.Escape() },
		} {
			t.Run(name, func(t *testing.T) {
				c := cantoController(t, false)
				history := len(c.Battle().History())
				end(c)
				playOut(c)
				u := c.Battle().Units()[1]
				if c.State() != Idle || u.Pos != (tactics.Pos{X: 1, Y: 0}) || !u.Acted {
					t.Errorf("状態 %v・位置 %v・行動済み %v", c.State(), u.Pos, u.Acted)
				}
				if _, ok := c.Battle().PendingCanto(); ok || len(c.Battle().History()) != history {
					t.Errorf("待っている %v・履歴 %d（前 %d）", ok, len(c.Battle().History()), history)
				}
			})
		}
	})

	t.Run("TC-IF-01-40 最後の自軍の再移動を終えてから敵軍の番を再生する", func(t *testing.T) {
		c := cantoController(t, true)
		if turn, side := c.Phase(); turn != 1 || side != tactics.Player {
			t.Fatalf("表示 = %d・%v", turn, side)
		}
		c.Escape()
		if c.State() != Playing {
			t.Fatalf("状態 = %v、敵軍の番の再生になるはず", c.State())
		}
		playOut(c)
		if c.State() != Idle || c.Battle().Turn != 2 {
			t.Errorf("状態 %v・ターン %d", c.State(), c.Battle().Turn)
		}
	})

	t.Run("TC-IF-01-40 待っている間の巻き戻しは、ふつうと同じ入口から移動の前へ戻す", func(t *testing.T) {
		c := cantoController(t, false)
		c.rewindKey("T")
		if !c.rewind.open {
			t.Fatal("一覧が開かない")
		}
		c.RewindEnter()
		c.RewindEnter()
		u := c.Battle().Units()[1]
		if c.State() != Idle || u.Pos != (tactics.Pos{X: 0, Y: 0}) || u.Acted {
			t.Errorf("巻き戻したあと: 状態 %v・位置 %v・行動済み %v", c.State(), u.Pos, u.Acted)
		}
		if _, ok := c.Battle().PendingCanto(); ok {
			t.Error("巻き戻しても待っている")
		}
	})

	t.Run("TC-IF-01-40 情報欄にキャラのスキルと兵種のスキルを並べ、説明はスキルごとに欄の幅に収める", func(t *testing.T) {
		canto := tactics.Skill{Name: "再移動", Effect: tactics.Canto}
		wrath := tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}
		for _, tc := range []struct {
			name        string
			skill       tactics.Skill
			class       []tactics.Skill
			names, desc string
		}{
			{"兵種のスキルだけ", tactics.Skill{}, []tactics.Skill{canto}, "再移動", "　攻撃・回復・道具のあと、残りの移動で動ける"},
			{"キャラと兵種", wrath, []tactics.Skill{canto}, "怒り、再移動", "　HP 半分以下で必殺 ＋50\n　攻撃・回復・道具のあと、残りの移動で動ける"},
			{"なし", tactics.Skill{}, nil, "なし", ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				u := tactics.Unit{Name: "エルザ", ClassName: "騎兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, HP: 20, Weapons: []tactics.Weapon{testLance}, Skill: tc.skill, ClassSkills: tc.class, Alive: true}
				v := (Panel{Unit: &u}).View()
				var texts []string
				for _, x := range v.Texts {
					texts = append(texts, x.Text)
				}
				all := strings.Join(texts, "\n")
				if !strings.Contains(all, "スキル:\n"+tc.names) || (tc.desc != "" && !strings.Contains(all, tc.desc)) {
					t.Errorf("情報欄に %q・%q がない:\n%s", tc.names, tc.desc, all)
				}
				// どの文も情報欄（左右の余白 8 を除く）の幅に収まる（REV-073 F-01）。
				for _, x := range v.Texts {
					if x.X < MapSize+8 || x.X+textWidth(x.Text) > ScreenWidth-8 {
						t.Errorf("%q（x %d・幅 %d）が情報欄 %d〜%d からはみ出す", x.Text, x.X, textWidth(x.Text), MapSize+8, ScreenWidth-8)
					}
				}
			})
		}
	})
}

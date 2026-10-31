package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// healController は設計書 7.10 節の回復例を、敵の攻撃で HP 12/19 にして用意する。
func healController(t *testing.T, uses int, damage ...int) *Controller {
	hit := 7
	if len(damage) > 0 {
		hit = damage[0]
	}
	t.Helper()
	c := newIdleWith(t, func() tactics.Setup {
		return tactics.Setup{Terrain: field(7, 7), Units: []tactics.UnitSetup{
			{Name: "リナ", ClassName: "僧侶", Side: tactics.Player, Stats: tactics.Stats{HP: 16, Mov: 3, Def: 2}, Pos: tactics.Pos{X: 2, Y: 1}, Weapon: tactics.Weapon{Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, MaxUses: uses, MinRange: 1, MaxRange: 1}, Items: []tactics.ItemStack{{Item: potion, Count: 1}}},
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 19}, Pos: tactics.Pos{X: 3, Y: 1}, Weapon: testSword, Lord: true},
			{Name: "ガルド", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 2, Y: 2}, Weapon: testLance},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Str: hit}, Pos: tactics.Pos{X: 4, Y: 1}, Weapon: testSword, Boss: true, AI: tactics.Hold},
		}}
	})
	c.EndPhase()
	c.EndPhase()
	playOut(c)
	if c.Battle().Units()[1].HP != 19-hit {
		t.Fatal("前提の HP が違う", c.Battle().Units())
	}
	return c
}

func moveHealer(c *Controller, dest tactics.Pos) {
	c.Click(tactics.Pos{X: 2, Y: 1})
	c.Click(dest)
}

func TestHealUI(t *testing.T) {
	t.Run("TC-FR-BTL-03-16 杖の行動と回復対象を移動先から判定する", func(t *testing.T) {
		c := healController(t, 15)
		moveHealer(c, tactics.Pos{X: 3, Y: 2})
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"回復", "待機"}) {
			t.Fatal(got)
		}
		if got := c.Targets(); len(got) != 0 {
			t.Fatal("杖に攻撃対象", got)
		}
		if got := c.HealTargets(); !reflect.DeepEqual(got, []int{1}) {
			t.Fatal(got)
		}
		c.ChooseAction(tactics.Attack)
		if c.State() != Moved {
			t.Fatal("杖で攻撃を選べた")
		}
		c.Cancel()
		c.Click(tactics.Pos{X: 0, Y: 1})
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"待機"}) {
			t.Fatal(got)
		}
	})
	t.Run("TC-IF-01-36 回復の選択・二度クリック・取り消し", func(t *testing.T) {
		c := healController(t, 15)
		moveHealer(c, tactics.Pos{X: 3, Y: 2})
		before := c.Battle().Units()
		c.ChooseAction(tactics.Heal)
		if c.State() != HealTargeting || !strings.Contains(strings.Join(c.Guide(), "\n"), "回復する味方を選ぶ") {
			t.Fatal(c.State(), c.Guide())
		}
		for _, p := range []tactics.Pos{{X: 2, Y: 2}, {X: 4, Y: 1}, {X: 3, Y: 2}, {X: 6, Y: 6}} {
			c.Click(p)
		}
		if c.target != -1 {
			t.Fatal("回復できない相手を選べた")
		}
		if got := c.targetFrameColor(1); got != goodColor {
			t.Fatal("未選択の回復候補枠が緑でない", got)
		}
		c.Click(tactics.Pos{X: 3, Y: 1})
		if got := c.targetFrameColor(1); got != cursorColor {
			t.Fatal("選んだ味方の枠がカーソル色でない", got)
		}
		if got := c.targetFrameColor(2); got != goodColor {
			t.Fatal("選んだ味方以外の枠の色が変わった", got)
		}
		if c.State() != HealTargeting || c.target != 1 || !reflect.DeepEqual(before, c.Battle().Units()) || c.Units()[0].Pos != (tactics.Pos{X: 3, Y: 2}) {
			t.Fatal("予測前に状態が変わった")
		}
		c.Escape()
		if c.State() != Moved || c.target != -1 {
			t.Fatal("取り消せない")
		}
		c.ChooseAction(tactics.Heal)
		c.Click(tactics.Pos{X: 3, Y: 1})
		c.Cancel()
		if c.State() != Moved {
			t.Fatal("右クリック相当の取り消し")
		}
		c.ChooseAction(tactics.Heal)
		c.Click(tactics.Pos{X: 3, Y: 1})
		c.Click(tactics.Pos{X: 3, Y: 1})
		if c.State() != Playing || c.Battle().Units()[1].HP != 19 || c.Battle().Units()[0].Weapon.Uses != 14 {
			t.Fatal("回復の確定に失敗")
		}
		found := false
		for _, e := range c.events {
			if e.Kind == tactics.EventHeal {
				found = true
				if e.Unit != 1 || e.Amount != 7 || e.Item != "癒やしの杖" || e.Units[0].Weapon.Uses != 14 || !strings.Contains(EventText(e), "7 回復した") {
					t.Fatal(e)
				}
			}
		}
		if !found {
			t.Fatal("回復の出来事がない")
		}
		c.Tick()
		playOut(c)
	})
	t.Run("TC-FR-BTL-08-22 回復の予測と盤面の文字は core の値を使う", func(t *testing.T) {
		c := healController(t, 15)
		moveHealer(c, tactics.Pos{X: 3, Y: 2})
		c.ChooseAction(tactics.Heal)
		if c.Panel(tactics.Pos{}, false).Heal != nil || c.TargetView().Visible {
			t.Fatal("選ぶ前に予測がある")
		}
		c.Click(tactics.Pos{X: 3, Y: 1})
		p := c.Panel(tactics.Pos{}, false)
		v := p.View()
		for _, s := range []string{"回復", "リナ → レオン", "HP 12 → 19", "回復 +7", "杖 残り 15 → 14/15"} {
			textOf(t, v, s)
		}
		if textOf(t, v, "回復").Style != StyleGood || len(p.Forecast) != 0 || v.Banner != nil || len(v.Icons) != 0 {
			t.Fatal("攻撃の表示が混ざる", v)
		}
		tv := c.TargetView()
		if tv.Text != "+7" || tv.Style != StyleGood || tv.Danger || !tv.Visible || tv.Pos != (tactics.Pos{X: 3, Y: 1}) {
			t.Fatal(tv)
		}
		for _, tc := range []struct{ damage, hp, amount, after int }{{14, 5, 10, 15}, {1, 18, 1, 19}} {
			d := healController(t, 15, tc.damage)
			moveHealer(d, tactics.Pos{X: 3, Y: 2})
			d.ChooseAction(tactics.Heal)
			d.Click(tactics.Pos{X: 3, Y: 1})
			f := d.Panel(tactics.Pos{}, false).Heal
			if f == nil || f.HP != tc.hp || f.Amount != tc.amount || f.After != tc.after {
				t.Fatal(f)
			}
		}
		c.Cancel()
		if c.TargetView().Visible || c.Panel(tactics.Pos{}, false).Heal != nil {
			t.Fatal("取り消しても予測が残る")
		}
	})
	t.Run("TC-FR-BTL-03-17 最後の一回の後は理由を表示し待機を許す", func(t *testing.T) {
		c := healController(t, 1)
		moveHealer(c, tactics.Pos{X: 2, Y: 1})
		c.ChooseAction(tactics.Heal)
		c.Click(tactics.Pos{X: 3, Y: 1})
		textOf(t, c.Panel(tactics.Pos{}, false).View(), "杖 残り 1 → 0/1")
		c.Click(tactics.Pos{X: 3, Y: 1})
		playOut(c)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		moveHealer(c, tactics.Pos{X: 2, Y: 1})
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"待機"}) {
			t.Fatal(got)
		}
		if !strings.Contains(strings.Join(c.Guide(), "\n"), tactics.ErrNoUses.Error()) {
			t.Fatal(c.Guide())
		}
		before := c.Battle().Units()
		c.ChooseAction(tactics.Heal)
		if c.State() != Moved || !reflect.DeepEqual(before, c.Battle().Units()) {
			t.Fatal("使い切って回復できた")
		}
		c.ChooseAction(tactics.Wait)
		if c.State() != Playing {
			t.Fatal("待機できない")
		}

	})
	t.Run("TC-FR-UNT-09-15 杖の情報と僧侶の絵", func(t *testing.T) {
		c := healController(t, 15)
		u := c.Units()[0]
		u.Weapon.Uses = 12
		v := (Panel{Unit: &u}).View()
		textOf(t, v, "武器: 癒やしの杖（杖）")
		textOf(t, v, "回復 10　射程 1　残り 12/15")
		if portraitOf(t, v).Tile != (assets.Tile{Sheet: assets.Dungeon, N: 108}) {
			t.Fatal(v.Icons)
		}
		for _, tx := range v.Texts {
			if strings.Contains(tx.Text, "威力") || strings.Contains(tx.Text, "命中") {
				t.Fatal(tx)
			}
		}
	})
	t.Run("TC-FR-BTL-08-23 杖の範囲と対象枠は緑で敵は赤のまま", func(t *testing.T) {
		c := healController(t, 15)
		c.Click(tactics.Pos{X: 2, Y: 1})
		if c.RangeStyle() != StyleGood {
			t.Fatal("選択中の杖が緑でない")
		}
		blue, reach := c.Overlay()
		if len(blue) == 0 || len(reach) == 0 {
			t.Fatal("範囲がない")
		}
		c.Click(tactics.Pos{X: 3, Y: 2})
		c.ChooseAction(tactics.Heal)
		_, reach = c.Overlay()
		if !reach[tactics.Pos{X: 3, Y: 1}] || c.RangeStyle() != StyleGood || c.TargetStyle() != StyleGood {
			t.Fatal(reach)
		}
		c.Cancel()
		c.SetCursor(4*CellSize+1, CellSize+1)
		if c.RangeStyle() != StyleBad {
			t.Fatal("敵の範囲の色が変わった")
		}
	})
}

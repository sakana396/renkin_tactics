package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面からの予測中の武器の操作を確かめる（DES-UI-34）。
func TestIT07ForecastWeapon(t *testing.T) {
	click := func(g *Game, p tactics.Pos) {
		pt := g.ctrl.Camera().Screen(p)
		g.leftClick(pt.X+2, pt.Y+2)
	}
	t.Run("TC-FR-BTL-08-27 相手選択で武器一覧を省き届く武器を仮選択", func(t *testing.T) {
		c := weaponMenuController(t)
		g := &Game{ctrl: c}
		click(g, tactics.Pos{X: 1, Y: 1})
		click(g, tactics.Pos{X: 1, Y: 1})
		_, dest, _ := c.Selection()
		r := c.Camera().MenuRects(dest, len(c.MenuLabels()))[0]
		g.leftClick(r.Min.X+2, r.Min.Y+2)
		if c.State() != Targeting || c.target != -1 || !reflect.DeepEqual(c.Targets(), []int{2}) {
			t.Fatalf("攻撃から相手: %v %d %v", c.State(), c.target, c.Targets())
		}
		click(g, tactics.Pos{X: 1, Y: 3})
		if c.weapon != 1 || c.Panel(tactics.Pos{}, false).ForecastWeapon != "ファイア" {
			t.Fatalf("予測の仮選択 %d", c.weapon)
		}
		setup := quickSetup()
		setup.Units[0].Weapons = []tactics.Weapon{quickSword, menuFire}
		setup.Units[0].AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword, tactics.Magic}
		setup.Units[0].Equipped = 1
		c = newIdleWith(t, func() tactics.Setup { return setup })
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 2})
		if c.weapon != 1 {
			t.Fatalf("届く装備を優先しない: %d", c.weapon)
		}
	})
	t.Run("TC-FR-BTL-08-28 窓の矢印とホイールで循環し窓外は送らない", func(t *testing.T) {
		c := newIdleWith(t, quickSetup)
		g := &Game{ctrl: c}
		click(g, tactics.Pos{X: 1, Y: 1})
		click(g, tactics.Pos{X: 1, Y: 2})
		if c.weapon != 0 || c.dest != (tactics.Pos{X: 1, Y: 1}) {
			t.Fatalf("初期 %d %v", c.weapon, c.dest)
		}
		row := c.Panel(c.dest, true).View().Bottom - lineHeight
		g.leftClick(MapSize+infoMargin+5, row+2)
		if c.weapon != 1 || c.dest != (tactics.Pos{X: 1, Y: 0}) {
			t.Fatalf("左 %d %v", c.weapon, c.dest)
		}
		c.Wheel(MapSize+10, 10, -1, false)
		if c.weapon != 0 {
			t.Fatalf("ホイール %d", c.weapon)
		}
		c.Wheel(10, 10, -1, false)
		if c.weapon != 0 {
			t.Fatal("窓外で武器が送られた")
		}
		g.leftClick(MapSize+infoMargin+100, row+2)
		if c.weapon != 1 || c.Panel(tactics.Pos{}, false).ForecastWeapons[0] != tactics.Bow {
			t.Fatalf("右 %d", c.weapon)
		}
	})
	t.Run("TC-FR-BTL-08-29 取消しと決定の実クリック経路", func(t *testing.T) {
		c := weaponMenuController(t)
		g := &Game{ctrl: c}
		click(g, tactics.Pos{X: 1, Y: 1})
		click(g, tactics.Pos{X: 1, Y: 1})
		c.ChooseAction(tactics.Attack)
		click(g, tactics.Pos{X: 1, Y: 3})
		for _, state := range []State{Targeting, Moved, Selected} {
			c.Cancel()
			if c.State() != state {
				t.Fatalf("取消し %v、期待 %v", c.State(), state)
			}
		}
		c = newIdleWith(t, quickSetup)
		g.ctrl = c
		click(g, tactics.Pos{X: 1, Y: 1})
		click(g, tactics.Pos{X: 1, Y: 2})
		if c.State() != Targeting {
			t.Fatalf("直接クリック: %v", c.State())
		}
		c.Cancel()
		if c.State() != Selected {
			t.Fatalf("直接クリックの取消し: %v", c.State())
		}
		click(g, tactics.Pos{X: 1, Y: 2})
		row := c.Panel(c.dest, true).View().Bottom - lineHeight
		g.leftClick(MapSize+infoMargin+230, row+2)
		if c.State() != Playing {
			t.Fatalf("窓の決定: %v", c.State())
		}
	})
	t.Run("TC-FR-ITM-01-07 仮選択と失敗時は装備と履歴を保つ", func(t *testing.T) {
		c := newIdleWith(t, quickSetup)
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 2})
		before := c.Battle().Units()
		history := len(c.Battle().History())
		c.ShiftForecastWeapon(1)
		if c.weapon != 1 || !reflect.DeepEqual(c.Battle().Units(), before) || len(c.Battle().History()) != history {
			t.Fatal("仮選択で戦闘が変わった")
		}
		good := c.dest
		control := newIdleWith(t, quickSetup)
		control.Click(tactics.Pos{X: 1, Y: 1})
		control.Click(tactics.Pos{X: 1, Y: 2})
		control.ShiftForecastWeapon(1)
		c.dest = tactics.Pos{X: 6, Y: 6}
		c.ConfirmForecast()
		if c.State() != Targeting || c.weapon != 1 || !reflect.DeepEqual(c.Battle().Units(), before) || len(c.Battle().History()) != history {
			t.Fatal("失敗時に仮選択か戦闘が変わった")
		}
		c.dest = good
		c.ConfirmForecast()
		control.ConfirmForecast()
		if !reflect.DeepEqual(c.events, control.events) || !reflect.DeepEqual(c.Battle().Units(), control.Battle().Units()) {
			t.Fatal("失敗後の決定で乱数か結果が変わった")
		}
		if c.State() != Playing || c.Battle().Units()[0].Equipped != 1 {
			t.Fatalf("決定: %v %d", c.State(), c.Battle().Units()[0].Equipped)
		}
	})
}

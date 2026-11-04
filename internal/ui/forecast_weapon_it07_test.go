package ui

import (
	"image"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面からの予測中の武器の操作を確かめる（DES-UI-34）。
func TestIT07ForecastWeapon(t *testing.T) {
	window := func(c *Controller) image.Rectangle { w, _ := c.ForecastWindow(); return w.Rect }
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
		r := window(c)
		g.leftClick(r.Min.X+16, r.Max.Y-20)
		if c.weapon != 1 || c.dest != (tactics.Pos{X: 1, Y: 0}) {
			t.Fatalf("左 %d %v", c.weapon, c.dest)
		}
		r = window(c)
		c.Wheel(r.Min.X+10, r.Min.Y+10, -1, false)
		if c.weapon != 0 {
			t.Fatalf("ホイール %d", c.weapon)
		}
		c.Wheel(10, 10, -1, false)
		if c.weapon != 0 {
			t.Fatal("窓外で武器が送られた")
		}
		r = window(c)
		g.leftClick(r.Min.X+100, r.Max.Y-20)
		if c.weapon != 1 || c.Panel(tactics.Pos{}, false).ForecastWeapons[0] != tactics.Bow {
			t.Fatalf("右 %d", c.weapon)
		}
	})
	t.Run("TC-FR-BTL-08-28 武器 3 本の入力順と戻る・巻き戻し", func(t *testing.T) {
		setup := quickSetup()
		setup.Units[0].Weapons = []tactics.Weapon{quickSword, menuBlade, quickBow}
		c := newIdleWith(t, func() tactics.Setup { return setup })
		g := &Game{ctrl: c}
		click(g, tactics.Pos{X: 1, Y: 1})
		click(g, tactics.Pos{X: 1, Y: 2})
		if c.weapon != 0 {
			t.Fatalf("初期武器 %d", c.weapon)
		}
		r := window(c)
		row := r.Max.Y - 20
		left, right := r.Min.X+16, r.Min.X+100
		for _, step := range []struct {
			name  string
			input func()
			want  int
		}{
			{"◀", func() { g.leftClick(left, row+2) }, 2},
			{"▶", func() { g.leftClick(right, row+2) }, 0},
			{"A", func() { c.battleKey("A") }, 2},
			{"D", func() { c.battleKey("D") }, 0},
			{"ホイール上", func() { c.Wheel(r.Min.X+10, r.Min.Y+10, 1, false) }, 2},
			{"ホイール下", func() { c.Wheel(r.Min.X+10, r.Min.Y+10, -1, false) }, 0},
		} {
			step.input()
			if c.weapon != step.want {
				t.Errorf("%s: 武器 %d、期待 %d", step.name, c.weapon, step.want)
			}
		}
		c.OpenRewind()
		before := c.weapon
		g.leftClick(left, row+2)
		c.battleKey("A")
		c.Wheel(r.Min.X+10, r.Min.Y+10, 1, false)
		if c.weapon != before {
			t.Fatal("巻き戻し中に武器を送った")
		}
		c.battleKey("Esc")
		g.leftClick(r.Max.X-32, row+2)
		if c.State() != Selected || c.weapon != -1 {
			t.Fatalf("戻る: %v %d", c.State(), c.weapon)
		}
	})
	t.Run("TC-FR-BTL-08-27 武器を決める前の相手候補と攻撃範囲", func(t *testing.T) {
		setup := quickSetup()
		staff := menuStaff
		staff.MinRange, staff.MaxRange = 3, 3
		setup.Units[0].Weapons = append(setup.Units[0].Weapons, staff)
		setup.Units[0].AllowedWeaponTypes = append(setup.Units[0].AllowedWeaponTypes, tactics.Staff)
		c := newIdleWith(t, func() tactics.Setup { return setup })
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.ChooseAction(tactics.Attack)
		wantTargets := []int{1, 2}
		wantRange := map[tactics.Pos]bool{}
		for _, w := range []int{0, 1} {
			for _, p := range c.Battle().WeaponRange(0, c.dest, w) {
				wantRange[p] = true
			}
		}
		check := func() {
			t.Helper()
			_, gotRange := c.Overlay()
			if c.weapon != -1 || !reflect.DeepEqual(c.Targets(), wantTargets) || !reflect.DeepEqual(gotRange, wantRange) || !gotRange[tactics.Pos{X: 1, Y: 2}] || !gotRange[tactics.Pos{X: 3, Y: 3}] {
				t.Fatalf("候補 %v、範囲 %v、武器 %d", c.Targets(), gotRange, c.weapon)
			}
		}
		check()
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.battleKey("Esc")
		check()
	})
	t.Run("TC-IF-01-43 直接クリックの予測で別の敵を選び直す", func(t *testing.T) {
		c := newIdleWith(t, quickSetup)
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ShiftForecastWeapon(1)
		c.Click(tactics.Pos{X: 3, Y: 3})
		if c.target != 2 || c.weapon != 0 || c.dest != (tactics.Pos{X: 3, Y: 2}) || c.quickTarget != 2 || c.quickSpots[1] != (tactics.Pos{X: 3, Y: 1}) {
			t.Fatalf("別の敵: 相手 %d、武器 %d、位置 %v、候補 %v", c.target, c.weapon, c.dest, c.quickSpots)
		}
	})
	t.Run("TC-IF-01-43 直接クリックと通常の攻撃は同じ結果で予測中に乱数を引かない", func(t *testing.T) {
		direct := newIdleWith(t, quickSetup)
		normal := newIdleWith(t, quickSetup)
		before := direct.Battle().Units()
		history := direct.Battle().History()
		direct.Click(tactics.Pos{X: 1, Y: 1})
		direct.Click(tactics.Pos{X: 3, Y: 3})
		if len(direct.watched) != 0 || direct.Units()[0].Pos != direct.dest || direct.dest != (tactics.Pos{X: 3, Y: 2}) {
			t.Fatal("直接クリックで範囲か仮位置が違う")
		}
		direct.ShiftForecastWeapon(1)
		if direct.Units()[0].Pos != (tactics.Pos{X: 3, Y: 1}) || !reflect.DeepEqual(direct.Battle().Units(), before) || !reflect.DeepEqual(direct.Battle().History(), history) {
			t.Fatal("予測中に盤面・履歴・仮位置が変わった")
		}
		normal.Click(tactics.Pos{X: 1, Y: 1})
		normal.Click(tactics.Pos{X: 3, Y: 1})
		normal.ChooseAction(tactics.Attack)
		normal.Click(tactics.Pos{X: 3, Y: 3})
		normal.ShiftForecastWeapon(1)
		if direct.weapon != 1 || normal.weapon != 1 {
			t.Fatal("弓が選ばれていない")
		}
		direct.ConfirmForecast()
		normal.ConfirmForecast()
		if !reflect.DeepEqual(direct.events, normal.events) || !reflect.DeepEqual(direct.Battle().Units(), normal.Battle().Units()) || !reflect.DeepEqual(direct.Battle().History(), normal.Battle().History()) {
			t.Fatal("同じ入力の結果か乱数が違う")
		}
		idle := newIdleWith(t, quickSetup)
		idle.Click(tactics.Pos{X: 3, Y: 3})
		if !idle.watched[2] {
			t.Fatal("Idle の敵クリックで範囲を足していない")
		}
	})
	t.Run("TC-IF-01-46 予測中の巻き戻し取消しと成功", func(t *testing.T) {
		c := newIdleWith(t, quickSetup)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.battleKey("D")
		state, target, weapon, dest := c.state, c.target, c.weapon, c.dest
		c.OpenRewind()
		c.battleKey("Esc")
		if c.state != state || c.target != target || c.weapon != weapon || c.dest != dest {
			t.Fatal("巻き戻しの取消しで予測が変わった")
		}
		c.OpenRewind()
		selectExplicitRewind(t, c)
		c.RewindEnter()
		c.RewindEnter()
		if c.rewind.open || c.state != Idle || c.weapon != -1 || c.quickSpots != nil {
			t.Fatalf("巻き戻し成功: %v %d %v", c.state, c.weapon, c.quickSpots)
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
		r := window(c)
		g.leftClick(r.Max.X-80, r.Max.Y-20)
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

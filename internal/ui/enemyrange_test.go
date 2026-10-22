package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// cellCenter はマス p の中央の画面の座標を返す。
func cellCenter(p tactics.Pos) (x, y int) {
	return p.X*CellSize + CellSize/2, p.Y*CellSize + CellSize/2
}

// rowSet は y = 0 の行のマス (x, 0) の集合を作る。
func rowSet(xs ...int) map[tactics.Pos]bool {
	m := map[tactics.Pos]bool{}
	for _, x := range xs {
		m[tactics.Pos{X: x, Y: 0}] = true
	}
	return m
}

// basicDanger は、基本の盤面の危険範囲。ファイター (1, 0) は移動 0・射程 1 なので隣の 3 マス、
// ボス (5, 5) は移動 0・射程 1 なので隣の 2 マス（盤面の外を除く）。どちらも待ち構える。
var basicDanger = posSet(
	tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 2, Y: 0}, tactics.Pos{X: 1, Y: 1},
	tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4},
)

// corridorSetup は 10 × 1 の通路の戦闘を作る（危険範囲の計算を確かめる）。(6, 0) は壁。
//
//	x:  0    1  2      3  4          5  6   7  8  9
//	    ボス     レオン     ソルジャー     壁
//
//	0: レオン (2, 0)。主人公。移動 3、威力 0 の剣（与えるダメージは 0）
//	1: ボス (0, 0)。移動 0、射程 1、待ち構える
//	2: ソルジャー (4, 0)。移動 3、射程 1、突っ込む。与えるダメージは 0
//	増援: 2 ターン目に (9, 0) に現れる。移動 0、射程 1、待ち構える
//
// 敵軍の番には、ソルジャーが (3, 0) に動いてレオンを攻撃する（0 ダメージ）。誰も倒れない。
func corridorSetup() tactics.Setup {
	wall := tactics.Terrain{Name: "壁", Cost: 0}
	terrain := field(10, 1)
	terrain[0][6] = wall
	return tactics.Setup{
		Terrain: terrain,
		Units: []tactics.UnitSetup{
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 2, Y: 0}, Weapon: testSword, Lord: true},
			{Name: "ボス", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testLance, Boss: true, AI: tactics.Hold},
			{Name: "ソルジャー", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 4, Y: 0}, Weapon: testSword, AI: tactics.Charge},
		},
		Reinforcements: []tactics.ReinforcementSetup{
			{Turn: 2, Unit: tactics.UnitSetup{Name: "増援", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 9, Y: 0}, Weapon: testSword, AI: tactics.Hold}},
		},
	}
}

// 敵の範囲と危険範囲（DES-UI-08）を、描画から分けた Controller で確かめる。
func TestEnemyRange(t *testing.T) {
	fighter := tactics.Pos{X: 1, Y: 0}
	boss := tactics.Pos{X: 5, Y: 5}
	empty := tactics.Pos{X: 3, Y: 3}
	// ファイターは移動 0・射程 1、ボスは移動 0・射程 1。
	fighterBlue, fighterRed := posSet(fighter), posSet(tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 2, Y: 0}, tactics.Pos{X: 1, Y: 1})
	bossBlue, bossRed := posSet(boss), posSet(tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4})

	overlayIs := func(t *testing.T, c *Controller, what string, wantBlue, wantRed map[tactics.Pos]bool) {
		t.Helper()
		blue, red := c.Overlay()
		if len(blue) == 0 && len(wantBlue) == 0 && len(red) == 0 && len(wantRed) == 0 {
			return
		}
		if !reflect.DeepEqual(blue, wantBlue) || !reflect.DeepEqual(red, wantRed) {
			t.Errorf("%s: 青 %v、赤 %v\nwant 青 %v、赤 %v", what, blue, red, wantBlue, wantRed)
		}
	}

	t.Run("TC-IF-01-18 Idle・Selected・Moved でカーソルを敵に合わせると、その敵の範囲を出し、離すと消える。行動の一覧の上やほかの状態では出さない", func(t *testing.T) {
		c := newIdle(t)
		overlayIs(t, c, "初め（カーソルはまだ盤面の外）", nil, nil)
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Idle でファイターに合わせる", fighterBlue, fighterRed)
		c.SetCursor(cellCenter(boss))
		overlayIs(t, c, "Idle でボスに合わせる", bossBlue, bossRed)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "Idle で空きマスに離す", nil, nil)
		c.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 2}))
		overlayIs(t, c, "Idle で自軍に合わせる", nil, nil)
		c.SetCursor(600, 16)
		overlayIs(t, c, "Idle で盤面の外（情報欄）", nil, nil)
		if c.State() != Idle {
			t.Fatalf("カーソルで状態が変わった: %v", c.State())
		}

		// Selected: 主人公を選ぶ。カーソルを離すと主人公の範囲、敵に合わせると敵の範囲だけ。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.SetCursor(cellCenter(empty))
		blue, _ := c.Overlay()
		if !reflect.DeepEqual(blue, keys(c.Battle().Reachable(0))) {
			t.Errorf("Selected で離す: 青 %v（主人公の範囲のはず）", blue)
		}
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Selected でファイターに合わせる", fighterBlue, fighterRed)

		// Moved（移動先 (0, 1)）: 行動の一覧は「待機」だけで、(32, 32)〜(200, 54) にあり、ファイターのマスに重ならない。
		c.Click(tactics.Pos{X: 0, Y: 1})
		if c.State() != Moved || len(c.MenuLabels()) != 1 {
			t.Fatalf("状態 %v、一覧 %q", c.State(), c.MenuLabels())
		}
		overlayIs(t, c, "Moved でファイターに合わせる", fighterBlue, fighterRed)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "Moved で離す（移動先から届くマスを赤）", nil, posSet(tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 0, Y: 2}))

		// Moved（移動先 (0, 0)）: 行動の一覧は「攻撃」「待機」で、(32, 0)〜(200, 44) にあり、ファイターのマスを覆う。
		c.Cancel()
		c.Click(tactics.Pos{X: 0, Y: 0})
		if c.State() != Moved || len(c.MenuLabels()) != 2 {
			t.Fatalf("状態 %v、一覧 %q", c.State(), c.MenuLabels())
		}
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Moved で行動の一覧の上（下にファイター）", nil, posSet(fighter, tactics.Pos{X: 0, Y: 1}))
		c.SetCursor(cellCenter(boss))
		overlayIs(t, c, "Moved でボスに合わせる", bossBlue, bossRed)

		// Targeting: 敵に合わせても出さない（攻撃の相手を選ぶ）。
		c.ChooseAction(tactics.Attack)
		c.SetCursor(cellCenter(boss))
		overlayIs(t, c, "Targeting でボスに合わせる", nil, posSet(fighter, tactics.Pos{X: 0, Y: 1}))
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Targeting でファイターに合わせる", nil, posSet(fighter, tactics.Pos{X: 0, Y: 1}))

		// Playing: 出さない。
		c.Click(fighter)
		c.Click(fighter)
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		c.SetCursor(cellCenter(boss))
		overlayIs(t, c, "Playing でボスに合わせる", nil, nil)

		// ItemMenu: 出さない。
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if d.State() != ItemMenu {
			t.Fatalf("状態 = %v", d.State())
		}
		d.SetCursor(cellCenter(boss))
		overlayIs(t, d, "ItemMenu でボスに合わせる", nil, nil)

		// Over: 出さない。
		lost := func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Stats.HP = 1
			return s
		}
		o := newIdleWith(t, lost)
		o.EndPhase()
		o.EndPhase()
		playOut(o)
		if o.State() != Over {
			t.Fatalf("状態 = %v", o.State())
		}
		o.SetCursor(cellCenter(fighter))
		overlayIs(t, o, "Over でファイターに合わせる", nil, nil)
	})

	t.Run("TC-IF-01-19 敵をクリックして範囲を出し続け、もう一度で消す。カーソルを合わせた敵を優先し、自軍の範囲の代わりに敵の範囲を出す", func(t *testing.T) {
		c := newIdle(t)
		// ボスに合わせてクリックし、カーソルを離しても出し続ける。
		c.SetCursor(cellCenter(boss))
		c.Click(boss)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "ボスをクリックして離す", bossBlue, bossRed)
		// 出し続けているボスより、カーソルを合わせたファイターを優先する。
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "ファイターに合わせる", fighterBlue, fighterRed)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "ファイターから離す", bossBlue, bossRed)
		// もう一度ボスをクリックすると、出し続けるのをやめる。カーソルがボスの上にある間は、カーソルの分だけ出る。
		c.SetCursor(cellCenter(boss))
		c.Click(boss)
		overlayIs(t, c, "もう一度クリック（カーソルはボスの上）", bossBlue, bossRed)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "もう一度クリックして離す", nil, nil)

		// ボスを出し続けたまま主人公を選ぶと、出し続けるのをやめ、主人公の範囲を出す。
		c.Click(boss)
		c.Click(tactics.Pos{X: 0, Y: 0})
		if c.State() != Selected {
			t.Fatalf("状態 = %v", c.State())
		}
		ownBlue, ownRed := c.ranges(0)
		overlayIs(t, c, "Selected（カーソルは空きマス）", ownBlue, ownRed)
		// 敵に合わせると、主人公の範囲の代わりに敵の範囲だけを出す（重ねない）。
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Selected でファイターに合わせる", fighterBlue, fighterRed)
		// Selected でボスをクリックしても、出し続けない（クリックは Idle の操作）。
		c.Click(boss)
		c.SetCursor(cellCenter(empty))
		overlayIs(t, c, "Selected でボスをクリックして離す", ownBlue, ownRed)
		c.Cancel()
		overlayIs(t, c, "取り消して Idle", nil, nil)
	})

	t.Run("TC-IF-01-20 Q: Playing・Over 以外で危険範囲を出す・消す。番をまたいで残し、再生中と勝ち負けの画面では出さない", func(t *testing.T) {
		c := newIdle(t)
		if got := c.Danger(); got != nil {
			t.Fatalf("初めから出ている: %v", got)
		}
		dangerIs := func(c *Controller, what string, want map[tactics.Pos]bool) {
			t.Helper()
			if got := c.Danger(); !reflect.DeepEqual(got, want) {
				t.Errorf("%s: 危険範囲 %v\nwant %v", what, got, want)
			}
		}
		c.ToggleDanger()
		dangerIs(c, "Idle で Q", basicDanger)
		c.ToggleDanger()
		dangerIs(c, "Idle でもう一度 Q", nil)
		c.ToggleDanger()
		c.Click(tactics.Pos{X: 0, Y: 0})
		dangerIs(c, "Selected（出したまま）", basicDanger)
		c.ToggleDanger()
		dangerIs(c, "Selected で Q", nil)
		c.ToggleDanger()
		dangerIs(c, "Selected でもう一度 Q", basicDanger)
		c.Click(tactics.Pos{X: 0, Y: 0})
		dangerIs(c, "Moved（出したまま）", basicDanger)
		c.ToggleDanger()
		dangerIs(c, "Moved で Q", nil)
		c.ToggleDanger()
		c.ChooseAction(tactics.Attack)
		dangerIs(c, "Targeting（出したまま）", basicDanger)
		c.ToggleDanger()
		dangerIs(c, "Targeting で Q", nil)
		c.ToggleDanger()
		dangerIs(c, "Targeting でもう一度 Q", basicDanger)

		// 攻撃を決めて再生する。再生中は出さず、Q も受け付けない。再生のあとは出したまま。
		c.Click(fighter)
		c.Click(fighter)
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		dangerIs(c, "Playing", nil)
		c.ToggleDanger()
		playOut(c)
		dangerIs(c, "再生のあと（Playing の Q は受け付けない）", basicDanger)

		// 番をまたいでも残す。
		c.EndPhase()
		c.EndPhase()
		dangerIs(c, "敵軍の番の再生中", nil)
		playOut(c)
		if turn, side := c.Phase(); c.State() != Idle || turn != 2 || side != tactics.Player {
			t.Fatalf("状態 %v、ターン %d、番 %v", c.State(), turn, side)
		}
		dangerIs(c, "ターン 2 の自軍の番", basicDanger)

		// ItemMenu でも受け付ける。
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if d.State() != ItemMenu {
			t.Fatalf("状態 = %v", d.State())
		}
		d.ToggleDanger()
		dangerIs(d, "ItemMenu で Q", basicDanger)

		// Over: 出さず、Q・R を受け付けない。
		lost := func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Stats.HP = 1
			return s
		}
		o := newIdleWith(t, lost)
		o.ToggleDanger()
		o.EndPhase()
		o.EndPhase()
		playOut(o)
		if o.State() != Over {
			t.Fatalf("状態 = %v", o.State())
		}
		dangerIs(o, "Over", nil)
		o.ToggleDanger()
		playOut(o)
		if o.State() != Over {
			t.Fatalf("Over の Q で状態が変わった: %v", o.State())
		}
		dangerIs(o, "Over の Q は受け付けない", nil)
	})

	t.Run("TC-IF-01-31 危険範囲は今の盤面の生きている敵の Reachable の各マスから AttackRange で求める。自軍のマスは通れないが攻撃でき、待ち構える敵を含め、まだ現れていない増援と Moved の仮の位置は数えない", func(t *testing.T) {
		c := newIdleWith(t, corridorSetup)
		c.ToggleDanger()
		// ボス (0, 0) は移動 0 で (1, 0) を攻撃できる（待ち構える敵も含める）。
		// ソルジャー (4, 0) は、左はレオン (2, 0) を通れず (3, 0) まで、右は壁 (6, 0) の手前 (5, 0) まで動ける。
		// その 3 マスの隣の (2, 0)〜(6, 0) を攻撃できる。レオンのいる (2, 0) を含み、レオンを通った先の (0, 0) は含まない。
		// 2 ターン目の増援 (9, 0) はまだいないので (8, 0) を含まない。
		turn1 := rowSet(1, 2, 3, 4, 5, 6)
		if got := c.Danger(); !reflect.DeepEqual(got, turn1) {
			t.Errorf("ターン 1: %v\nwant %v", got, turn1)
		}

		// Moved でレオンを画面の上だけ (3, 0) に動かしても、今の盤面（レオンは (2, 0)）で求める。
		// 仮の位置で求めると、ソルジャーは (3, 0) に入れず、(2, 0) を攻撃できなくなる。
		c.Click(tactics.Pos{X: 2, Y: 0})
		c.Click(tactics.Pos{X: 3, Y: 0})
		if c.State() != Moved || c.Units()[0].Pos != (tactics.Pos{X: 3, Y: 0}) {
			t.Fatalf("状態 %v、描くレオン %v", c.State(), c.Units()[0].Pos)
		}
		if got := c.Danger(); !reflect.DeepEqual(got, turn1) {
			t.Errorf("Moved: %v\nwant %v", got, turn1)
		}
		c.Cancel()
		c.Cancel()

		// ターン 1 の敵軍の番: ソルジャーが (3, 0) からレオンを攻撃する。ターン 2 も増援はまだいない。
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		units := c.Battle().Units()
		if turn, _ := c.Phase(); turn != 2 || units[2].Pos != (tactics.Pos{X: 3, Y: 0}) || len(units) != 3 {
			t.Fatalf("ターン %d、ソルジャー %v、ユニットの数 %d", turn, units[2].Pos, len(units))
		}
		if got := c.Danger(); !reflect.DeepEqual(got, turn1) {
			t.Errorf("ターン 2: %v\nwant %v", got, turn1)
		}

		// ターン 2 の敵軍の番に増援が (9, 0) に現れる。ターン 3 では、増援の攻撃できる (8, 0) を含む。
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		units = c.Battle().Units()
		if turn, _ := c.Phase(); turn != 3 || len(units) != 4 || units[3].Pos != (tactics.Pos{X: 9, Y: 0}) || !units[0].Alive {
			t.Fatalf("ターン %d、ユニット %+v", turn, units)
		}
		if want := rowSet(1, 2, 3, 4, 5, 6, 8); !reflect.DeepEqual(c.Danger(), want) {
			t.Errorf("ターン 3: %v\nwant %v", c.Danger(), want)
		}
	})

	t.Run("TC-IF-01-32 倒れた敵は危険範囲に数えず、カーソルを合わせても範囲を出さない", func(t *testing.T) {
		weak := func() tactics.Setup {
			s := basicSetup()
			s.Units[3].Stats.HP = 1
			return s
		}
		c := newIdleWith(t, weak)
		c.ToggleDanger()
		// 主人公がファイター（HP 1）を倒す。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(fighter)
		c.Click(fighter)
		playOut(c)
		if c.State() != Idle || c.Battle().Units()[3].Alive {
			t.Fatalf("状態 %v、ファイターが生きている %v", c.State(), c.Battle().Units()[3].Alive)
		}
		if got, want := c.Danger(), posSet(tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4}); !reflect.DeepEqual(got, want) {
			t.Errorf("危険範囲 %v\nwant %v（ボスだけ）", got, want)
		}
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "倒れたファイターのマスに合わせる", nil, nil)
	})
}

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
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testLance, Boss: true, AI: tactics.Hold},
			{Name: "ソルジャー", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 4, Y: 0}, Weapon: testSword, AI: tactics.Charge},
		},
		Reinforcements: []tactics.ReinforcementSetup{
			{Turn: 2, Unit: tactics.UnitSetup{Name: "増援", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 9, Y: 0}, Weapon: testSword, AI: tactics.Hold}},
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

	t.Run("TC-IF-01-18 Idle・Selected・Moved でカーソルを敵に合わせると、その敵の範囲を出し、離すと消える。行動の一覧の上やほかの状態では出さない", func(t *testing.T) {
		// カーソルの敵の範囲は CursorFrames（青白の実線・破線）で返し、自軍の範囲の塗り（Overlay）には入れない（DES-UI-29）。
		c := newIdle(t)
		cursorIs(t, c, "初め（カーソルはまだ盤面の外）", nil, nil)
		c.SetCursor(cellCenter(fighter))
		cursorIs(t, c, "Idle でファイターに合わせる", fighterBlue, fighterRed)
		overlayIs(t, c, "Idle でファイターに合わせた塗り", nil, nil)
		c.SetCursor(cellCenter(boss))
		cursorIs(t, c, "Idle でボスに合わせる", bossBlue, bossRed)
		c.SetCursor(cellCenter(empty))
		cursorIs(t, c, "Idle で空きマスに離す", nil, nil)
		c.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 2}))
		cursorIs(t, c, "Idle で自軍に合わせる", nil, nil)
		c.SetCursor(600, 16)
		cursorIs(t, c, "Idle で盤面の外（情報欄）", nil, nil)
		if c.State() != Idle || len(c.watched) != 0 {
			t.Fatalf("カーソルで状態・固定が変わった: %v、%v", c.State(), c.watched)
		}

		// Selected: 主人公を選ぶ。塗りはいつも主人公の範囲で、敵に合わせるとその敵の範囲を線で重ねる。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.SetCursor(cellCenter(empty))
		ownBlue, ownRed := c.ranges(0)
		if !reflect.DeepEqual(ownBlue, keys(c.Battle().Reachable(0))) {
			t.Fatalf("前提: 主人公の範囲 %v", ownBlue)
		}
		overlayIs(t, c, "Selected で離す（主人公の範囲）", ownBlue, ownRed)
		cursorIs(t, c, "Selected で離す", nil, nil)
		c.SetCursor(cellCenter(fighter))
		cursorIs(t, c, "Selected でファイターに合わせる", fighterBlue, fighterRed)
		overlayIs(t, c, "Selected でファイターに合わせた塗り（主人公の範囲のまま）", ownBlue, ownRed)

		// Moved（移動先 (0, 1)）: 行動の一覧は「待機」だけで、(32, 32)〜(200, 54) にあり、ファイターのマスに重ならない。
		c.Click(tactics.Pos{X: 0, Y: 1})
		if c.State() != Moved || len(c.MenuLabels()) != 1 {
			t.Fatalf("状態 %v、一覧 %q", c.State(), c.MenuLabels())
		}
		movedRed := posSet(tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 0, Y: 2})
		cursorIs(t, c, "Moved でファイターに合わせる", fighterBlue, fighterRed)
		overlayIs(t, c, "Moved でファイターに合わせた塗り（移動先から届くマス）", nil, movedRed)
		c.SetCursor(cellCenter(empty))
		cursorIs(t, c, "Moved で離す", nil, nil)
		overlayIs(t, c, "Moved で離す（移動先から届くマスを赤）", nil, movedRed)

		// Moved（移動先 (0, 0)）: 行動の一覧は「攻撃」「待機」で、(32, 0)〜(200, 44) にあり、ファイターのマスを覆う。
		c.Cancel()
		c.Click(tactics.Pos{X: 0, Y: 0})
		if c.State() != Moved || len(c.MenuLabels()) != 2 {
			t.Fatalf("状態 %v、一覧 %q", c.State(), c.MenuLabels())
		}
		c.SetCursor(cellCenter(fighter))
		cursorIs(t, c, "Moved で行動の一覧の上（下にファイター）", nil, nil)
		overlayIs(t, c, "Moved で行動の一覧の上の塗り", nil, posSet(fighter, tactics.Pos{X: 0, Y: 1}))
		c.SetCursor(cellCenter(boss))
		cursorIs(t, c, "Moved でボスに合わせる", bossBlue, bossRed)

		// Targeting: 敵に合わせても出さない（攻撃の相手を選ぶ）。
		c.ChooseAction(tactics.Attack)
		c.SetCursor(cellCenter(boss))
		cursorIs(t, c, "Targeting でボスに合わせる", nil, nil)
		overlayIs(t, c, "Targeting でボスに合わせた塗り", nil, posSet(fighter, tactics.Pos{X: 0, Y: 1}))
		c.SetCursor(cellCenter(fighter))
		cursorIs(t, c, "Targeting でファイターに合わせる", nil, nil)

		// Playing: 出さない。
		c.Click(fighter)
		c.Click(fighter)
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		c.SetCursor(cellCenter(boss))
		cursorIs(t, c, "Playing でボスに合わせる", nil, nil)

		// ItemMenu: 出さない。
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if d.State() != ItemMenu {
			t.Fatalf("状態 = %v", d.State())
		}
		d.SetCursor(cellCenter(boss))
		cursorIs(t, d, "ItemMenu でボスに合わせる", nil, nil)

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
		cursorIs(t, o, "Over でファイターに合わせる", nil, nil)
	})

	t.Run("TC-IF-01-19 敵をクリックして固定し、離しても出し続け、もう一度で外す。カーソルの敵とは別の層で、自軍を選んでいる間も自軍の塗りと見分けられる線で残す", func(t *testing.T) {
		// 固定した敵の範囲は WatchedFrames（橙の線）、カーソルの敵の範囲は CursorFrames（青白の線）で、別の層にする（DES-UI-29）。
		c := newIdle(t)
		// ボスに合わせてクリックし、カーソルを離しても出し続ける。
		c.SetCursor(cellCenter(boss))
		c.Click(boss)
		c.SetCursor(cellCenter(empty))
		watchedIs(t, c, "ボスをクリックして離す", bossBlue, bossRed)
		cursorIs(t, c, "ボスをクリックして離す", nil, nil)
		overlayIs(t, c, "ボスをクリックして離した塗り", nil, nil)
		// カーソルを合わせたファイターの範囲は、固定したボスの範囲とは別の層（カーソル）で出す。固定は変えない。
		c.SetCursor(cellCenter(fighter))
		watchedIs(t, c, "ファイターに合わせる", bossBlue, bossRed)
		cursorIs(t, c, "ファイターに合わせる", fighterBlue, fighterRed)
		c.SetCursor(cellCenter(empty))
		watchedIs(t, c, "ファイターから離す", bossBlue, bossRed)
		cursorIs(t, c, "ファイターから離す", nil, nil)
		// もう一度ボスをクリックすると、固定をやめる。カーソルがボスの上にある間は、カーソルの分だけ出る。
		c.SetCursor(cellCenter(boss))
		c.Click(boss)
		watchedIs(t, c, "もう一度クリック（カーソルはボスの上）", nil, nil)
		cursorIs(t, c, "もう一度クリック（カーソルはボスの上）", bossBlue, bossRed)
		c.SetCursor(cellCenter(empty))
		cursorIs(t, c, "もう一度クリックして離す", nil, nil)

		// ボスを固定したまま主人公を選ぶと、主人公の範囲を塗り、ボスの範囲は固定の線で残す（E-60、DES-UI-29）。
		c.Click(boss)
		c.Click(tactics.Pos{X: 0, Y: 0})
		if c.State() != Selected {
			t.Fatalf("状態 = %v", c.State())
		}
		ownBlue, ownRed := c.ranges(0)
		overlayIs(t, c, "Selected（カーソルは空きマス）", ownBlue, ownRed)
		watchedIs(t, c, "Selected", bossBlue, bossRed)
		if c.RangeStyle() != StyleBad {
			t.Error("Selected の主人公の射程の色が赤でない")
		}
		// 敵に合わせても、塗りは主人公の範囲のまま、その敵の範囲をカーソルの線で重ねる。ボスの固定は残す。
		c.SetCursor(cellCenter(fighter))
		overlayIs(t, c, "Selected でファイターに合わせる", ownBlue, ownRed)
		cursorIs(t, c, "Selected でファイターに合わせる", fighterBlue, fighterRed)
		watchedIs(t, c, "Selected でファイターに合わせる", bossBlue, bossRed)
		// Selected でボスをクリックすると攻撃の操作になる。ボスを攻撃できる移動先はないので、理由を出して Selected のまま。
		// 固定は変えない。
		c.Click(boss)
		c.SetCursor(cellCenter(empty))
		if c.State() != Selected || c.Message() == "" {
			t.Errorf("Selected でボスをクリック: 状態 %v、知らせ %q", c.State(), c.Message())
		}
		watchedIs(t, c, "Selected でボスをクリックして離す", bossBlue, bossRed)
		overlayIs(t, c, "Selected でボスをクリックして離す", ownBlue, ownRed)
		// 移動先 (0, 1) を選んだ Moved でも固定の線で残す。相手を選ぶ状態では出さない。
		c.Click(tactics.Pos{X: 0, Y: 1})
		watchedIs(t, c, "Moved", bossBlue, bossRed)
		c.Cancel()
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		if c.State() != Targeting {
			t.Fatalf("状態 = %v", c.State())
		}
		watchedIs(t, c, "Targeting", nil, nil)
		c.Cancel()
		c.Cancel()
		// 取り消して Idle に戻っても、ボスの範囲は固定の線で出す（塗りにはしない）。
		c.Cancel()
		watchedIs(t, c, "取り消して Idle", bossBlue, bossRed)
		overlayIs(t, c, "取り消して Idle の塗り", nil, nil)
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
		cursorIs(t, c, "倒れたファイターのマスに合わせる", nil, nil)
	})
}

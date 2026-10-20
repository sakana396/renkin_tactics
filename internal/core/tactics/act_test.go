package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// errSetup は検証の誤りを起こしやすい盤面。
//
//	. . # . .
//	. . . . .
//	. . . . .
//	. . . . .
//	. . . . .
//
//	0: 主人公 (0, 0) 移動 2
//	1: 味方   (1, 0) 移動 2
//	2: ボス   (4, 4)
//	3: 敵     (0, 1)
//	4: 味方   (4, 0) 移動 2（倒れた状態にする）
func errSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: grid("..#..", ".....", ".....", ".....", "....."),
		Units: []tactics.UnitSetup{
			lord(0, 0, 2),
			ally("味方", 1, 0, 2),
			boss(4, 4),
			foe("敵", 0, 1),
			ally("倒れた味方", 4, 0, 2),
		},
	}
}

// errBattle は errSetup の戦闘を作り、prepare で状態を整えてから、添字 4 の味方を倒れた状態にする。
func errBattle(t *testing.T, prepare func(b *tactics.Battle)) *tactics.Battle {
	t.Helper()
	b := mustNew(t, errSetup())
	if prepare != nil {
		prepare(b)
	}
	b.KillUnit(4)
	return b
}

// actedUnit4 は、のちに倒れる味方（添字 4）を行動済みにする。
func actedUnit4(b *tactics.Battle) {
	if _, err := b.Act(tactics.Command{Unit: 4, To: pos(4, 0), Action: tactics.Wait}); err != nil {
		panic(err)
	}
}

// actedUnit1 は味方（添字 1）を行動済みにする。
func actedUnit1(b *tactics.Battle) {
	if _, err := b.Act(tactics.Command{Unit: 1, To: pos(1, 0), Action: tactics.Wait}); err != nil {
		panic(err)
	}
}

func over(b *tactics.Battle)       { b.Result = tactics.Won }
func enemyPhase(b *tactics.Battle) { b.Phase = tactics.Enemy }

// actErrorCase は失敗する Act の 1 件。
type actErrorCase struct {
	name    string
	prepare func(b *tactics.Battle)
	cmd     tactics.Command
	want    error
}

// orderCases は、複数の検証に当てはまるコマンドで、最初の誤りが返ることを確かめる（DES-BTL-04）。
var orderCases = []actErrorCase{
	{"0 が 1〜5 より先", func(b *tactics.Battle) { over(b); enemyPhase(b) },
		tactics.Command{Unit: 99, To: pos(9, 9), Action: tactics.Action(3)}, tactics.ErrInvalidAction},
	{"1 が 2〜5 より先", func(b *tactics.Battle) { over(b); enemyPhase(b) },
		tactics.Command{Unit: 99, To: pos(9, 9), Action: tactics.Wait}, tactics.ErrBattleOver},
	{"2 が 3〜5 より先", enemyPhase,
		tactics.Command{Unit: 3, To: pos(9, 9), Action: tactics.Wait}, tactics.ErrNotPlayerPhase},
	{"3 が 4・5 より先（行動済みで倒れた味方）", actedUnit4,
		tactics.Command{Unit: 4, To: pos(9, 9), Action: tactics.Wait}, tactics.ErrInvalidUnit},
	{"4 が 5 より先", actedUnit1,
		tactics.Command{Unit: 1, To: pos(9, 9), Action: tactics.Wait}, tactics.ErrAlreadyActed},
	{"Attack でも 0〜5 の順", func(b *tactics.Battle) { over(b) },
		tactics.Command{Unit: 99, To: pos(9, 9), Action: tactics.Attack, Target: 99}, tactics.ErrBattleOver},
	{"UseItem でも 5 は 6 より先", nil,
		tactics.Command{Unit: 0, To: pos(2, 0), Action: tactics.UseItem, Item: 99}, tactics.ErrUnreachable},
}

// eachCases は、検証 0〜5 の誤りを 1 つずつ起こす（DES-BTL-04、DES-BTL-08）。
var eachCases = []actErrorCase{
	{"ErrInvalidAction（3）", nil, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.Action(3)}, tactics.ErrInvalidAction},
	{"ErrInvalidAction（-1）", nil, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.Action(-1)}, tactics.ErrInvalidAction},
	{"ErrBattleOver（勝ち）", func(b *tactics.Battle) { b.Result = tactics.Won },
		tactics.Command{Unit: 0, To: pos(0, 0)}, tactics.ErrBattleOver},
	{"ErrBattleOver（負け）", func(b *tactics.Battle) { b.Result = tactics.Lost },
		tactics.Command{Unit: 0, To: pos(0, 0)}, tactics.ErrBattleOver},
	{"ErrNotPlayerPhase", enemyPhase, tactics.Command{Unit: 0, To: pos(0, 0)}, tactics.ErrNotPlayerPhase},
	{"ErrInvalidUnit（-1）", nil, tactics.Command{Unit: -1, To: pos(0, 0)}, tactics.ErrInvalidUnit},
	{"ErrInvalidUnit（数を超える）", nil, tactics.Command{Unit: 5, To: pos(0, 0)}, tactics.ErrInvalidUnit},
	{"ErrInvalidUnit（敵）", nil, tactics.Command{Unit: 3, To: pos(0, 1)}, tactics.ErrInvalidUnit},
	{"ErrInvalidUnit（倒れた味方）", nil, tactics.Command{Unit: 4, To: pos(4, 0)}, tactics.ErrInvalidUnit},
	{"ErrAlreadyActed", actedUnit1, tactics.Command{Unit: 1, To: pos(1, 0)}, tactics.ErrAlreadyActed},
	{"ErrUnreachable（壁）", nil, tactics.Command{Unit: 1, To: pos(2, 0)}, tactics.ErrUnreachable},
	{"ErrUnreachable（マップの外）", nil, tactics.Command{Unit: 0, To: pos(-1, 0)}, tactics.ErrUnreachable},
	{"ErrUnreachable（移動を超える）", nil, tactics.Command{Unit: 1, To: pos(1, 3)}, tactics.ErrUnreachable},
	{"ErrUnreachable（味方のいるマス）", nil, tactics.Command{Unit: 0, To: pos(1, 0)}, tactics.ErrUnreachable},
	{"ErrUnreachable（敵のいるマス）", nil, tactics.Command{Unit: 0, To: pos(0, 1)}, tactics.ErrUnreachable},
}

// TestActErrors は Act の検証と、失敗したら状態を変えないことを確かめる（DES-BTL-04、DES-ERR-01）。
func TestActErrors(t *testing.T) {
	groups := []struct {
		id    string
		cases []actErrorCase
	}{
		{"TC-NFR-07-01 検証の順番", orderCases},
		{"TC-NFR-07-02 検証の誤り", eachCases},
	}
	for _, g := range groups {
		for _, tt := range g.cases {
			t.Run(g.id+" / TC-NFR-07-03 "+tt.name, func(t *testing.T) {
				b := errBattle(t, tt.prepare)
				before := snapshot(b)
				events, err := b.Act(tt.cmd)
				if !errors.Is(err, tt.want) {
					t.Errorf("err = %v, want %v", err, tt.want)
				}
				if events != nil {
					t.Errorf("出来事 = %v, want nil", events)
				}
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
	}
}

// TestActNotImplemented は、UseItem が検証 0〜5 を通ったあとの扱いを確かめる。
// アイテム（#9）の Issue で、アイテムのテストに置き換える。
func TestActNotImplemented(t *testing.T) {
	t.Run("TC-NFR-07-04 まだ作っていない行動（UseItem）は状態を変えずに誤りを返す", func(t *testing.T) {
		b := errBattle(t, nil)
		before := snapshot(b)
		events, err := b.Act(tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem})
		if !errors.Is(err, tactics.ErrNotImplemented) {
			t.Errorf("err = %v, want ErrNotImplemented", err)
		}
		if events != nil {
			t.Errorf("出来事 = %v, want nil", events)
		}
		assertEqual(t, "状態", snapshot(b), before)
	})
}

// TestErrorValues は誤りの値を確かめる（DES-BTL-08）。
func TestErrorValues(t *testing.T) {
	t.Run("TC-NFR-07-05 誤りの値は errors.Is で互いに見分けられる", func(t *testing.T) {
		all := []error{
			tactics.ErrBattleOver, tactics.ErrNotPlayerPhase, tactics.ErrInvalidUnit,
			tactics.ErrAlreadyActed, tactics.ErrUnreachable, tactics.ErrNoTarget,
			tactics.ErrOutOfRange, tactics.ErrNoItem, tactics.ErrFullHP,
			tactics.ErrInvalidAction, tactics.ErrInvalidSetup,
		}
		for i, a := range all {
			if a == nil || a.Error() == "" {
				t.Errorf("all[%d] が空", i)
			}
			for j, c := range all {
				if got := errors.Is(a, c); got != (i == j) {
					t.Errorf("errors.Is(%v, %v) = %v", a, c, got)
				}
			}
		}
	})
}

// TestDeterminism は乱数とコマンドの再現性を確かめる（DES-RNG-01、NFR-06）。
func TestDeterminism(t *testing.T) {
	t.Run("TC-NFR-06-01 同じ種なら同じ乱数の列になり、値は 0〜99", func(t *testing.T) {
		a := mustNewSeed(t, 42)
		b := mustNewSeed(t, 42)
		for i := 0; i < 1000; i++ {
			x, y := a.Roll(), b.Roll()
			if x != y {
				t.Fatalf("%d 番目: %d と %d", i, x, y)
			}
			if x < 0 || x > 99 {
				t.Fatalf("%d 番目: %d は 0〜99 でない", i, x)
			}
		}
	})

	t.Run("TC-NFR-06-02 同じ種と同じコマンドの列なら、出来事と状態が同じ", func(t *testing.T) {
		run := func() ([]tactics.Event, state) {
			b, events, err := tactics.New(baseSetup(), 7)
			if err != nil {
				t.Fatal(err)
			}
			cmds := []tactics.Command{
				{Unit: 1, To: pos(1, 3), Action: tactics.Wait},
				{Unit: 0, To: pos(1, 1), Action: tactics.Wait},
				{Unit: 0, To: pos(2, 1), Action: tactics.Wait},
			}
			for _, c := range cmds {
				ev, err := b.Act(c)
				if err != nil {
					t.Fatal(err)
				}
				events = append(events, ev...)
			}
			ev, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			return append(events, ev...), snapshot(b)
		}
		e1, s1 := run()
		e2, s2 := run()
		assertEqual(t, "出来事", e1, e2)
		assertEqual(t, "状態", s1, s2)
	})

	t.Run("TC-NFR-06-03 乱数を引く関数を決まった数の列に差し替えられる", func(t *testing.T) {
		b := mustNewSeed(t, 1)
		seq := []int{40, 90, 70, 50}
		b.SetRoll(func() int {
			x := seq[0]
			seq = seq[1:]
			return x
		})
		var got []int
		for i := 0; i < 4; i++ {
			got = append(got, b.Roll())
		}
		assertEqual(t, "乱数", got, []int{40, 90, 70, 50})
	})
}

func mustNewSeed(t *testing.T, seed uint64) *tactics.Battle {
	t.Helper()
	b, _, err := tactics.New(baseSetup(), seed)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

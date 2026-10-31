package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 勝ち負け（DES-MAP-04）のテスト。敵の動き方（#10）はまだないので、
// 敵軍の番の攻撃は export_test.go の SetEnemyTarget で組む。

// judgeSetup は 5 × 5 の平地に、自軍 2 体と敵 3 体を置く。
//
//	0: 主人公 (lx, ly) 移動 2
//	1: 味方   (0, 0)   移動 2
//	2: ボス   (4, 4)
//	3: 敵     (4, 0)
//	4: 敵 2   (2, 2)
func judgeSetup(lx, ly int) tactics.Setup {
	return tactics.Setup{
		Terrain: grid(".....", ".....", ".....", ".....", "....."),
		Units: []tactics.UnitSetup{
			lord(lx, ly, 2),
			ally("味方", 0, 0, 2),
			boss(4, 4),
			foe("敵", 4, 0),
			foe("敵 2", 2, 2),
		},
	}
}

// targetOf は、敵 attacker だけが target を攻撃し、ほかの敵は何もしないようにする。動いた敵の添字を acted に足す。
func targetOf(b *tactics.Battle, attacker, target int, acted *[]int) {
	b.SetEnemyTarget(func(unit int) int {
		*acted = append(*acted, unit)
		if unit == attacker {
			return target
		}
		return -1
	})
}

// assertOver は、戦闘が終わったあとの Act と EndPhase が ErrBattleOver を返し、状態を変えないことを確かめる。
func assertOver(t *testing.T, b *tactics.Battle) {
	t.Helper()
	noRolls(t, b)
	before := snapshot(b)
	cmds := []tactics.Command{
		{Unit: 1, To: pos(0, 0), Action: tactics.Wait},
		{Unit: 1, To: pos(0, 0), Action: tactics.UseItem, Item: 0},
		{Unit: 1, To: pos(0, 0), Action: tactics.Attack, Target: 3},
	}
	for _, c := range cmds {
		events, err := b.Act(c)
		if !errors.Is(err, tactics.ErrBattleOver) {
			t.Errorf("Act(%+v): err = %v, want ErrBattleOver", c, err)
		}
		if events != nil {
			t.Errorf("Act(%+v): 出来事 = %v, want nil", c, events)
		}
	}
	events, err := b.EndPhase()
	if !errors.Is(err, tactics.ErrBattleOver) {
		t.Errorf("EndPhase: err = %v, want ErrBattleOver", err)
	}
	if events != nil {
		t.Errorf("EndPhase: 出来事 = %v, want nil", events)
	}
	assertEqual(t, "状態", snapshot(b), before)
}

// TestWin は勝ちを確かめる（DES-MAP-04、FR-MAP-05）。
func TestWin(t *testing.T) {
	// winByAttack は、味方を待機させてから、主人公がボスを攻撃して倒す。
	winByAttack := func(t *testing.T) (*tactics.Battle, []tactics.Event) {
		t.Helper()
		b := mustNew(t, judgeSetup(2, 4))
		b.SetHP(2, 1)
		mustAct(t, b, tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.Wait})
		done := rolls(t, b, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(3, 4), Action: tactics.Attack, Target: 2})
		done()
		return b, events
	}

	t.Run("TC-FR-MAP-05-01 自軍の攻撃でボスが倒れると勝ちになり、敵軍の番に進まない", func(t *testing.T) {
		b, events := winByAttack(t)
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventMove, tactics.EventStrike, tactics.EventFall, tactics.EventWon,
		})
		assertEqual(t, "倒れた", events[2].Unit, 2)
		if events[3].Units[2].Alive {
			t.Error("Won の写しでボスが生きている")
		}
		assertEqual(t, "結果", b.Result, tactics.Won)
		assertEqual(t, "番", b.Phase, tactics.Player)
		assertEqual(t, "ターン", b.Turn, 1)
	})

	t.Run("TC-FR-MAP-05-02 敵軍の番に、反撃でボスが倒れると勝ちになり、そこで止まる", func(t *testing.T) {
		b := mustNew(t, judgeSetup(3, 4))
		b.SetHP(2, 1)
		var acted []int
		targetOf(b, 2, 0, &acted)
		// ボスの 1 撃目は外れ、主人公の反撃が当たる。
		done := rolls(t, b, 99, 99, 0, 0, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart, tactics.EventStrike, tactics.EventStrike, tactics.EventFall, tactics.EventWon,
		})
		// ボスのあとの敵（3、4）は動かず、次の自軍の番も始まらない。
		assertEqual(t, "動いた敵", acted, []int{2})
		assertEqual(t, "結果", b.Result, tactics.Won)
		assertEqual(t, "番", b.Phase, tactics.Enemy)
		assertEqual(t, "ターン", b.Turn, 1)
	})

	t.Run("TC-FR-MAP-05-03 勝ちが決まったあとの Act と EndPhase は ErrBattleOver", func(t *testing.T) {
		b, _ := winByAttack(t)
		assertOver(t, b)
	})

	t.Run("TC-FR-MAP-05-04 ボスでない敵が倒れても戦闘は続く", func(t *testing.T) {
		b := mustNew(t, judgeSetup(2, 0))
		b.SetHP(3, 1)
		done := rolls(t, b, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3})
		done()
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventMove, tactics.EventStrike, tactics.EventFall,
		})
		assertEqual(t, "結果", b.Result, tactics.Ongoing)
		assertEqual(t, "番", b.Phase, tactics.Player)
	})
}

// TestLose は負けを確かめる（DES-MAP-04、FR-CMP-09）。
func TestLose(t *testing.T) {
	// loseInEnemyPhase は、敵軍の番に敵（3）が主人公を攻撃して倒す。
	loseInEnemyPhase := func(t *testing.T) (*tactics.Battle, []tactics.Event, []int) {
		t.Helper()
		b := mustNew(t, judgeSetup(4, 1))
		b.SetHP(0, 1)
		var acted []int
		targetOf(b, 3, 0, &acted)
		done := rolls(t, b, 0, 0, 99)
		events := mustEndPhase(t, b)
		done()
		return b, events, acted
	}

	t.Run("TC-FR-CMP-09-01 自軍の番に、反撃で主人公が倒れると負けになる", func(t *testing.T) {
		b := mustNew(t, judgeSetup(2, 0))
		b.SetHP(0, 1)
		// 主人公の 1 撃目は外れ、敵の反撃が当たる。
		done := rolls(t, b, 99, 99, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3})
		done()
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventMove, tactics.EventStrike, tactics.EventStrike, tactics.EventFall, tactics.EventLost,
		})
		assertEqual(t, "倒れた", events[3].Unit, 0)
		assertEqual(t, "結果", b.Result, tactics.Lost)
		assertEqual(t, "番", b.Phase, tactics.Player)
	})

	t.Run("TC-FR-CMP-09-02 敵軍の番に、敵の攻撃で主人公が倒れると負けになり、そこで止まる", func(t *testing.T) {
		b, events, acted := loseInEnemyPhase(t)
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart, tactics.EventStrike, tactics.EventFall, tactics.EventLost,
		})
		// 主人公を倒した敵（3）のあとの敵（4）は動かず、次の自軍の番も始まらない。
		assertEqual(t, "動いた敵", acted, []int{2, 3})
		assertEqual(t, "結果", b.Result, tactics.Lost)
		assertEqual(t, "番", b.Phase, tactics.Enemy)
		assertEqual(t, "ターン", b.Turn, 1)
	})

	t.Run("TC-FR-CMP-09-05 添字が最後の敵の攻撃で主人公が倒れても、ターンを進めずに止まる", func(t *testing.T) {
		// 主人公を、添字が最後の敵 2（4、(2, 2)）の隣に置く。
		b := mustNew(t, judgeSetup(2, 3))
		b.SetHP(0, 1)
		var acted []int
		targetOf(b, 4, 0, &acted)
		done := rolls(t, b, 0, 0, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart, tactics.EventStrike, tactics.EventFall, tactics.EventLost,
		})
		assertEqual(t, "動いた敵", acted, []int{2, 3, 4})
		assertEqual(t, "結果", b.Result, tactics.Lost)
		assertEqual(t, "番", b.Phase, tactics.Enemy)
		assertEqual(t, "ターン", b.Turn, 1)
		assertOver(t, b)
	})

	t.Run("TC-FR-CMP-09-03 負けが決まったあとの Act と EndPhase は ErrBattleOver", func(t *testing.T) {
		b, _, _ := loseInEnemyPhase(t)
		assertOver(t, b)
	})

	t.Run("TC-FR-CMP-09-04 主人公でない自軍が倒れても戦闘は続く", func(t *testing.T) {
		s := judgeSetup(2, 4)
		// 敵 2（4）を味方 (0, 0) の隣に置く。
		s.Units[4].Pos = pos(1, 0)
		b := mustNew(t, s)
		b.SetHP(1, 1)
		var acted []int
		targetOf(b, 4, 1, &acted)
		done := rolls(t, b, 0, 0, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart, tactics.EventStrike, tactics.EventFall, tactics.EventPhaseStart,
		})
		assertEqual(t, "動いた敵", acted, []int{2, 3, 4})
		assertEqual(t, "結果", b.Result, tactics.Ongoing)
		assertEqual(t, "ターン", b.Turn, 2)
	})
}

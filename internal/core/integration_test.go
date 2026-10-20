package core_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 本体のデータの ch01 で、敵の本物の動き方に攻めさせ、主人公が倒れたら負けで止まることを確かめる（REV-013 I-01）。
// 乱数の具体的な結果には頼らず、「負けで止まる」性質だけを確かめる。
func TestCh01LostByEnemies(t *testing.T) {
	t.Run("TC-FR-CMP-09-06 ch01 で自軍が待機と番を終えるだけを続けると、主人公が倒れて Lost で止まり、以後は ErrBattleOver で敵も動かない", func(t *testing.T) {
		cat := loadBase(t)
		for _, seed := range []uint64{1, 2, 3, 4, 5} {
			b, _, err := core.NewBattle(cat, "ch01", seed)
			if err != nil {
				t.Fatalf("種 %d: NewBattle: %v", seed, err)
			}
			last := playUntilOver(t, b, seed)
			if b.Result != tactics.Lost {
				t.Fatalf("種 %d: 結果 = %v、Lost のはず", seed, b.Result)
			}

			// 最後の出来事の列は、主人公の Fall → Lost で終わり、そのあとに何も起きていない。
			n := len(last)
			if n < 2 || last[n-1].Kind != tactics.EventLost || last[n-2].Kind != tactics.EventFall || !last[n-2].Units[last[n-2].Unit].Lord {
				t.Fatalf("種 %d: 最後の出来事が 主人公の Fall → Lost でない: %+v", seed, kinds(last))
			}
			for _, e := range last[:n-1] {
				if e.Kind == tactics.EventLost || e.Kind == tactics.EventWon {
					t.Errorf("種 %d: 勝ち負けの出来事が途中にある", seed)
				}
			}

			// 以後の操作はすべて ErrBattleOver で、状態は変わらない（敵も動かない）。
			before := b.Units()
			turn, phase := b.Turn, b.Phase
			for i := 0; i < 3; i++ {
				if ev, err := b.EndPhase(); !errors.Is(err, tactics.ErrBattleOver) || ev != nil {
					t.Errorf("種 %d: EndPhase = %v, %v", seed, ev, err)
				}
				for u, unit := range before {
					if unit.Side != tactics.Player {
						continue
					}
					if ev, err := b.Act(tactics.Command{Unit: u, To: unit.Pos, Action: tactics.Wait}); !errors.Is(err, tactics.ErrBattleOver) || ev != nil {
						t.Errorf("種 %d: Act(%d) = %v, %v", seed, u, ev, err)
					}
				}
			}
			if !reflect.DeepEqual(b.Units(), before) || b.Turn != turn || b.Phase != phase || b.Result != tactics.Lost {
				t.Errorf("種 %d: 負けのあとに状態が変わった", seed)
			}
		}
	})
}

// playUntilOver は、戦闘が終わるまで自軍の番を進め、最後の Act か EndPhase が返した出来事を返す。
//
// 自軍の番では、主人公をボスに近づくマスへ待機（Wait）で動かし、ほかの自軍は動かさずに番を終える。
// 主人公を敵に近づけて、待ち構える敵にも攻めさせるため。
func playUntilOver(t *testing.T, b *tactics.Battle, seed uint64) []tactics.Event {
	t.Helper()
	lord, boss := -1, -1
	for i, u := range b.Units() {
		if u.Lord {
			lord = i
		}
		if u.Boss {
			boss = i
		}
	}
	for turn := 0; turn < 100; turn++ {
		units := b.Units()
		goal := units[boss].Pos
		best, bestD := units[lord].Pos, distance(units[lord].Pos, goal)
		for p := range b.Reachable(lord) {
			if d := distance(p, goal); d < bestD || d == bestD && (p.Y < best.Y || p.Y == best.Y && p.X < best.X) {
				best, bestD = p, d
			}
		}
		events, err := b.Act(tactics.Command{Unit: lord, To: best, Action: tactics.Wait})
		if err != nil {
			t.Fatalf("種 %d: Act: %v", seed, err)
		}
		if b.Result != tactics.Ongoing {
			return events
		}
		if b.Phase == tactics.Player {
			events, err = b.EndPhase()
			if err != nil {
				t.Fatalf("種 %d: EndPhase: %v", seed, err)
			}
			if b.Result != tactics.Ongoing {
				return events
			}
		}
	}
	t.Fatalf("種 %d: 100 ターンで戦闘が終わらない", seed)
	return nil
}

func distance(a, b tactics.Pos) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func kinds(events []tactics.Event) []tactics.EventKind {
	out := make([]tactics.EventKind, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}

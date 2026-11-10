package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func it07Knight(side tactics.Side, x, y int, alive bool) tactics.Unit {
	return tactics.Unit{Side: side, Alive: alive, Pos: tactics.Pos{X: x, Y: y}, ClassSkills: []tactics.Skill{{Name: "通せんぼ"}}}
}

func it07Plain(forest ...tactics.Pos) func(tactics.Pos) int {
	return func(p tactics.Pos) int {
		for _, f := range forest {
			if f == p {
				return 2
			}
		}
		return 1
	}
}

// 移動経路の ZOC 計測の境界を確かめる（#188、DES-BTL-11、DES-MAP-09）。
func TestIT07PathZOCMeasure(t *testing.T) {
	type tc struct {
		name     string
		w, h     int
		cost     func(tactics.Pos) int
		others   []tactics.Unit
		from, to tactics.Pos
		min, max int
		extra    int
	}
	// 移動者の陣営を s として、重騎士の陣営を相手軍（foe）か同じ陣営（ally）にする。
	build := func(s tactics.Side) []tc {
		foe, ally := 1-s, s
		return []tc{
			{"TC-FR-MAP-04-33 中間の ZOC を通る直線（着地点は ZOC でない）", 5, 3, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 2, true)}, tactics.Pos{X: 0, Y: 1}, tactics.Pos{X: 4, Y: 1}, 1, 1, 1},
			{"TC-FR-MAP-04-33 重騎士のマスは通れず、迂回しても ZOC に入る", 5, 3, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 1, true)}, tactics.Pos{X: 0, Y: 1}, tactics.Pos{X: 4, Y: 1}, 1, 1, 1},
			{"TC-FR-MAP-04-33 ZOC を避ける経路があれば入らず追加費用 0", 5, 5, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 4, true)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 4, Y: 0}, 0, 0, 0},
			{"TC-FR-MAP-04-33 同じ最小費用で入る経路と避ける経路があれば幅で示す", 3, 2, it07Plain(tactics.Pos{X: 0, Y: 1}),
				[]tactics.Unit{it07Knight(foe, 2, 0, true)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 1}, 0, 1, 1},
			{"TC-FR-MAP-04-33 同じ最小費用の別経路（入る側を後から見つける向き）", 3, 3, it07Plain(tactics.Pos{X: 1, Y: 0}),
				[]tactics.Unit{it07Knight(foe, 0, 2, true)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 1}, 0, 1, 1},
			{"TC-FR-MAP-04-33 1 マスに 2 体の重騎士が隣接しても加算は 1", 3, 3, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 1, true), it07Knight(foe, 1, 2, true)}, tactics.Pos{X: 0, Y: 1}, tactics.Pos{X: 1, Y: 1}, 1, 1, 1},
			{"TC-FR-MAP-04-33 味方のマスを通り抜けられ、そのマスが ZOC 内なら加算 1", 3, 2, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 1, 1, true), {Side: ally, Alive: true, Pos: tactics.Pos{X: 1, Y: 0}}}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 2, Y: 0}, 1, 1, 1},
			{"TC-FR-MAP-04-33 着地点が ZOC のとき 1 回と追加費用 1", 3, 1, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 0, true)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0}, 1, 1, 1},
			{"TC-FR-MAP-04-33 戦死した重騎士の ZOC は数えない", 3, 1, it07Plain(),
				[]tactics.Unit{it07Knight(foe, 2, 0, false)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0}, 0, 0, 0},
			{"TC-FR-MAP-04-33 同じ陣営の重騎士の ZOC は数えない", 3, 1, it07Plain(),
				[]tactics.Unit{it07Knight(ally, 2, 0, true)}, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0}, 0, 0, 0},
		}
	}
	for _, side := range []tactics.Side{tactics.Player, tactics.Enemy} {
		for _, c := range build(side) {
			t.Run(fmt.Sprintf("%s 陣営%d", c.name, side), func(t *testing.T) {
				mover := tactics.Unit{Side: side, Alive: true, Pos: c.to}
				units := append([]tactics.Unit{mover}, c.others...)
				tr := newIT07PathTracker(c.cost, c.w, c.h)
				tr.Event(tactics.Event{Kind: tactics.EventMove, Unit: 0, From: c.from, To: c.to}, units, side)
				got := tr.log[0].Move
				if got.Min != c.min || got.Max != c.max || got.ExtraCost != c.extra {
					t.Fatalf("下限%d 上限%d 追加費用%d、期待 %d %d %d", got.Min, got.Max, got.ExtraCost, c.min, c.max, c.extra)
				}
			})
		}
	}
	t.Run("TC-FR-MAP-04-33 2 体の重騎士の ZOC に続けて入る", func(t *testing.T) {
		// 高さ 2 の盤面で、上の段の (2,0)・(5,0) の重騎士の隣は下の段の (2,1)・(5,1)。
		units := []tactics.Unit{
			{Side: tactics.Player, Alive: true, Pos: tactics.Pos{X: 6, Y: 1}},
			it07Knight(tactics.Enemy, 2, 0, true), it07Knight(tactics.Enemy, 5, 0, true),
		}
		tr := newIT07PathTracker(it07Plain(), 7, 2)
		tr.Event(tactics.Event{Kind: tactics.EventMove, Unit: 0, From: tactics.Pos{X: 0, Y: 1}, To: tactics.Pos{X: 6, Y: 1}}, units, tactics.Player)
		got := tr.log[0].Move
		if got.Min != 2 || got.Max != 2 || got.ExtraCost != 2 {
			t.Fatalf("下限%d 上限%d 追加費用%d、期待 2 2 2", got.Min, got.Max, got.ExtraCost)
		}
	})
}

// 再移動の区別、巻き戻しでの最終経路と全試行の区別を確かめる（#188）。
func TestIT07PathTrackerRewindAndRemove(t *testing.T) {
	cost := it07Plain()
	units := func(mover tactics.Pos) []tactics.Unit {
		return []tactics.Unit{
			{Side: tactics.Enemy, Alive: true, Pos: mover},
			it07Knight(tactics.Player, 3, 0, true),
		}
	}
	move := func(tr *it07PathTracker, from, to tactics.Pos) {
		tr.Event(tactics.Event{Kind: tactics.EventMove, Unit: 0, From: from, To: to}, units(to), tactics.Enemy)
	}
	t.Run("TC-FR-MAP-04-34 攻撃のあとの移動だけを再移動にする", func(t *testing.T) {
		tr := newIT07PathTracker(cost, 8, 1)
		tr.PhaseStart()
		move(tr, tactics.Pos{X: 0}, tactics.Pos{X: 2}) // 着地が ZOC。通常の移動
		tr.Event(tactics.Event{Kind: tactics.EventStrike, Unit: 0, Target: 1}, units(tactics.Pos{X: 2}), tactics.Enemy)
		move(tr, tactics.Pos{X: 2}, tactics.Pos{X: 1}) // 攻撃のあと。再移動
		all := tr.AllTrials()
		if all.EnemyMove.Moves != 1 || all.EnemyRemove.Moves != 1 || all.PlayerMove.Moves+all.PlayerRemove.Moves != 0 {
			t.Fatalf("集計 %+v", all)
		}
		if all.EnemyMove.ZOCMoves != 1 || all.EnemyMove.ExtraCost != 1 || all.EnemyRemove.ZOCMoves != 0 {
			t.Fatalf("ZOC の集計 %+v", all)
		}
		// 回復を受けただけのユニットは、行動した側ではないので再移動にしない。
		tr.PhaseStart()
		tr.Event(tactics.Event{Kind: tactics.EventHeal, Unit: 0, Amount: 5}, units(tactics.Pos{X: 1}), tactics.Enemy)
		move(tr, tactics.Pos{X: 1}, tactics.Pos{X: 0})
		if got := tr.AllTrials(); got.EnemyMove.Moves != 2 || got.EnemyRemove.Moves != 1 {
			t.Fatalf("回復を受けたあとの集計 %+v", got)
		}
		// 待機のあとはゲームで再移動できないので、印にしない。
		tr.PhaseStart()
		tr.Event(tactics.Event{Kind: tactics.EventWait, Unit: 0}, units(tactics.Pos{X: 1}), tactics.Enemy)
		move(tr, tactics.Pos{X: 1}, tactics.Pos{X: 0})
		if got := tr.AllTrials(); got.EnemyMove.Moves != 3 || got.EnemyRemove.Moves != 1 {
			t.Fatalf("待機のあとの集計 %+v", got)
		}
		// 次の番では印が消える。番の陣営と違う側の反撃は行動に数えない。
		tr.PhaseStart()
		tr.Event(tactics.Event{Kind: tactics.EventStrike, Unit: 0, Target: 1}, units(tactics.Pos{X: 1}), tactics.Player)
		move(tr, tactics.Pos{X: 1}, tactics.Pos{X: 0})
		if got := tr.AllTrials(); got.EnemyMove.Moves != 4 || got.EnemyRemove.Moves != 1 {
			t.Fatalf("次の番の集計 %+v", got)
		}
	})
	t.Run("TC-FR-MAP-04-34 巻き戻した移動と増援は最終経路から消え、全試行に残る", func(t *testing.T) {
		tr := newIT07PathTracker(cost, 8, 1)
		spawn := func(id string) {
			u := units(tactics.Pos{})
			u[0].ID = id
			tr.Event(tactics.Event{Kind: tactics.EventReinforce, Unit: 0}, u, tactics.Enemy)
		}
		tr.TurnStart(1)
		spawn("X")
		move(tr, tactics.Pos{X: 0}, tactics.Pos{X: 2})
		tr.TurnStart(2)
		move(tr, tactics.Pos{X: 0}, tactics.Pos{X: 2})
		spawn("Y")
		tr.Rewind(2)
		final, spawned := tr.Final()
		if !reflect.DeepEqual(spawned, []string{"X"}) || final.EnemyMove.Moves != 1 {
			t.Fatalf("2 ターン目へ戻した最終経路 %v %+v", spawned, final)
		}
		// 戻した先で別の増援が出る。全試行には X・Y・Z、最終経路には X・Z。
		tr.TurnStart(2)
		spawn("Z")
		if _, s := tr.Final(); !reflect.DeepEqual(s, []string{"X", "Z"}) {
			t.Fatalf("やり直した最終経路 %v", s)
		}
		tr.Rewind(1)
		final, spawned = tr.Final()
		if len(spawned) != 0 || final.EnemyMove.Moves != 0 {
			t.Fatalf("1 ターン目へ戻した最終経路 %v %+v", spawned, final)
		}
		if got := tr.AllTrials(); got.EnemyMove.Moves != 2 || got.EnemyMove.ZOCMoves != 2 {
			t.Fatalf("全試行 %+v", got)
		}
		if !reflect.DeepEqual(tr.allSpawned, []string{"X", "Y", "Z"}) {
			t.Fatalf("全試行の増援 %v", tr.allSpawned)
		}
	})
	t.Run("TC-FR-MAP-04-34 巻き戻す前だけ出た増援と、最終経路で出なかった予定", func(t *testing.T) {
		if got := it07Difference([]string{"X", "Y", "Z"}, []string{"X", "Z"}); !reflect.DeepEqual(got, []string{"Y"}) {
			t.Fatalf("巻き戻す前だけの増援 %v", got)
		}
		if got := it07Difference([]string{"X", "Y"}, []string{"X", "Y"}); len(got) != 0 {
			t.Fatalf("最終経路でも出た増援を数えた %v", got)
		}
		got := it07NotIn([]it07Scheduled{{ID: "X"}, {ID: "Y"}, {ID: "Z"}}, []string{"Y"})
		if !reflect.DeepEqual(got, []string{"X", "Z"}) {
			t.Fatalf("最終経路で未出現 %v", got)
		}
	})
}

// 計測器の最小費用が tactics.Reachable と一致することを、本体の全 9 章の初期盤面で確かめる（#188 REV-106 F-05）。
func TestIT07PathMatchesReachable(t *testing.T) {
	cat := baseCatalog(t)
	zocTiles := 0
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		t.Run("TC-FR-MAP-04-33 "+id+" の tactics.Reachable と一致", func(t *testing.T) {
			setup, err := core.NewSetup(cat, id)
			if err != nil {
				t.Fatal(err)
			}
			// 初期の盤面には自軍が重騎士の近くにいないので、重騎士・ボス以外の敵を自軍に替えて、ZOC の費用が効く移動者を作る。
			setup.Units = append([]tactics.UnitSetup(nil), setup.Units...)
			setup.Events, setup.Reinforcements = nil, nil
			for i, u := range setup.Units {
				zoc := false
				for _, sk := range u.ClassSkills {
					zoc = zoc || sk.Effect == tactics.ZOC
				}
				if u.Side == tactics.Enemy && !zoc && !u.Boss && i%2 == 0 {
					setup.Units[i].Side = tactics.Player
					setup.Units[i].Group, setup.Units[i].Approach, setup.Units[i].Commander = "", 0, false
				}
			}
			bv, err := tactics.NewBoardView(setup)
			if err != nil {
				t.Fatal(err)
			}
			units := bv.Units()
			cost := func(p tactics.Pos) int { return bv.TerrainAt(p).Cost }
			for i, u := range units {
				if !u.Alive {
					continue
				}
				got := it07ShortestZOC(units, i, u.Pos, cost, bv.Width(), bv.Height(), true)
				plain := it07ShortestZOC(units, i, u.Pos, cost, bv.Width(), bv.Height(), false)
				reach := bv.Reachable(i)
				for pos, c := range reach {
					if got.dist[pos] != c {
						t.Fatalf("%s %v→%v: 計測器 %d、Reachable %d", u.Name, u.Pos, pos, got.dist[pos], c)
					}
					if got.dist[pos] != plain.dist[pos] {
						zocTiles++
					}
				}
				for pos, c := range got.dist {
					if _, ok := reach[pos]; ok || c > u.Stats.Mov {
						continue
					}
					occupied := false
					for j, o := range units {
						if j != i && o.Alive && o.Pos == pos {
							occupied = true
						}
					}
					// 味方のマスには止まれないので Reachable にない。それ以外は一致するはず。
					if !occupied {
						t.Fatalf("%s %v→%v: 計測器の費用 %d で行けるが Reachable にない", u.Name, u.Pos, pos, c)
					}
				}
			}
		})
	}
	if zocTiles == 0 {
		t.Fatal("ZOC で費用が変わるマスが 1 つもなく、比較が空になっている")
	}
}

package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// DES-IT07 4.2 節の計算例。横道は壁で閉じ、進入先だけの費用を確かめる。
func TestZOCReachable(t *testing.T) {
	armor := func(side tactics.Side, x, y int) tactics.UnitSetup {
		u := ally("重騎士", x, y, 0)
		if side == tactics.Enemy {
			u = foe("重騎士", x, y)
		}
		u.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		u.Stats.Mov = 0
		return u
	}
	cases := []struct {
		name      string
		start     tactics.Pos
		armors    []tactics.UnitSetup
		target    tactics.Pos
		cost      int
		reachable bool
	}{
		{"相手重騎士なし", pos(0, 1), nil, pos(3, 1), 3, true},
		{"B だけが ZOC 内", pos(0, 1), []tactics.UnitSetup{armor(tactics.Enemy, 1, 0)}, pos(2, 1), 3, true},
		{"B と C が ZOC 内", pos(0, 1), []tactics.UnitSetup{armor(tactics.Enemy, 1, 0), armor(tactics.Enemy, 2, 0)}, pos(2, 1), 4, false},
		{"B に二体が隣接", pos(0, 1), []tactics.UnitSetup{armor(tactics.Enemy, 1, 0), armor(tactics.Enemy, 1, 2)}, pos(2, 1), 3, true},
		{"ZOC 内の B から出発", pos(1, 1), []tactics.UnitSetup{armor(tactics.Enemy, 1, 0)}, pos(3, 1), 2, true},
		{"自軍重騎士だけ", pos(0, 1), []tactics.UnitSetup{armor(tactics.Player, 1, 0)}, pos(3, 1), 3, true},
	}
	for _, tc := range cases {
		t.Run("TC-FR-MAP-01-06 "+tc.name, func(t *testing.T) {
			rows := []string{"######", "......", "######"}
			if len(tc.armors) > 0 {
				top := []byte(rows[0])
				bottom := []byte(rows[2])
				for _, a := range tc.armors {
					if a.Pos.Y == 0 {
						top[a.Pos.X] = '.'
					} else {
						bottom[a.Pos.X] = '.'
					}
				}
				rows[0], rows[2] = string(top), string(bottom)
			}
			u := lord(tc.start.X, tc.start.Y, 3)
			s := tactics.Setup{Terrain: grid(rows...), Units: append([]tactics.UnitSetup{u, boss(5, 1)}, tc.armors...)}
			b := mustNew(t, s)
			for _, reader := range boardReaders(t, s, b) {
				got, ok := reader.Reachable(0)[tc.target]
				if ok != tc.reachable || (ok && got != tc.cost) {
					t.Errorf("%s: 費用 %d, 到達 %v; want %d, %v", tc.name, got, ok, tc.cost, tc.reachable)
				}
			}
		})
	}
}

func TestZOCCanto(t *testing.T) {
	for _, kill := range []bool{false, true} {
		name := "ZOC を持つ敵が残る"
		if kill {
			name = "ZOC を持つ敵を倒す"
		}
		t.Run("TC-FR-BTL-03-21 "+name, func(t *testing.T) {
			cav := ally("騎兵", 0, 1, 7)
			cav.ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
			knight := foe("重騎士", 2, 0)
			knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
			knight.AI = tactics.Hold
			knight.Stats.Mov = 0
			s := tactics.Setup{Terrain: grid("##.#####", "........", ".######."), Units: []tactics.UnitSetup{lord(0, 2, 0), cav, knight, boss(7, 2)}}
			b := mustNew(t, s)
			b.SetRoll(func() int {
				if kill {
					return 0
				}
				return 99
			})
			if kill {
				b.SetHP(2, 1)
			}
			if got := b.Reachable(1)[pos(2, 1)]; got != 3 {
				t.Fatalf("行動前の費用 = %d, want 3", got)
			}
			_, err := b.Act(tactics.Command{Unit: 1, To: pos(2, 1), Action: tactics.Attack, Target: 2})
			if err != nil {
				t.Fatal(err)
			}
			pending, ok := b.PendingCanto()
			if !ok || pending.Remaining != 4 {
				t.Fatalf("残り = %+v, %v; want 4", pending, ok)
			}
			if got, ok := b.CantoReachable()[pos(6, 1)]; !ok || got != 4 {
				t.Errorf("撃破後の平地 4 歩 = %d, %v", got, ok)
			}
			if _, ok := b.CantoReachable()[pos(7, 1)]; ok {
				t.Error("5 歩先に到達した")
			}
			sides := b.BoardView().ZOCSides(pos(2, 1))
			if sides[tactics.Enemy] == kill {
				t.Errorf("撃破後の ZOC = %v, kill=%v", sides, kill)
			}
		})
	}
}

func TestZOCCostBoundaries(t *testing.T) {
	knight := foe("重騎士", 2, 0)
	knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
	knight.Stats.Mov = 0
	tests := []struct {
		name      string
		terrain   []string
		ally      *tactics.UnitSetup
		target    tactics.Pos
		cost      int
		reachable bool
	}{
		{"壁は通れない", []string{"##.###", "..#...", "######"}, nil, pos(2, 1), 0, false},
		{"相手重騎士のマスは占有", []string{"##.###", "......", "######"}, nil, pos(2, 0), 0, false},
		{"味方のマスは通れるが止まれない", []string{"##.###", "......", "######"}, func() *tactics.UnitSetup { u := ally("味方", 1, 1, 0); return &u }(), pos(2, 1), 3, true},
	}
	for _, tc := range tests {
		t.Run("TC-FR-MAP-01-07 "+tc.name, func(t *testing.T) {
			units := []tactics.UnitSetup{lord(0, 1, 4), boss(5, 1), knight}
			if tc.ally != nil {
				units = append(units, *tc.ally)
			}
			b := mustNew(t, tactics.Setup{Terrain: grid(tc.terrain...), Units: units})
			got, ok := b.Reachable(0)[tc.target]
			if ok != tc.reachable || (ok && got != tc.cost) {
				t.Errorf("費用 %d, 到達 %v; want %d, %v", got, ok, tc.cost, tc.reachable)
			}
			if tc.ally != nil {
				if _, ok := b.Reachable(0)[pos(1, 1)]; ok {
					t.Error("味方のマスで止まれる")
				}
			}
		})
	}
}

func TestZOCRewind(t *testing.T) {
	t.Run("TC-FR-BTL-10-34 通せんぼの撃破と再移動を一手で巻き戻す", func(t *testing.T) {
		cav := ally("騎兵", 0, 1, 7)
		cav.ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
		knight := foe("重騎士", 2, 0)
		knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		knight.AI = tactics.Hold
		knight.Stats.Mov = 0
		s := tactics.Setup{Terrain: grid("##.#####", "........", ".######."), Units: []tactics.UnitSetup{lord(0, 2, 0), cav, knight, boss(7, 2)}}
		b := mustNew(t, s)
		b.SetRoll(func() int { return 0 })
		b.SetHP(2, 1)
		before := b.Units()
		if !b.BoardView().ZOCSides(pos(2, 1))[tactics.Enemy] {
			t.Fatal("行動前の ZOC がない")
		}
		attack := tactics.Command{Unit: 1, To: pos(2, 1), Action: tactics.Attack, Target: 2}
		if _, err := b.Act(attack); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Act(tactics.Command{Unit: 1, To: pos(6, 1), Action: tactics.CantoMove}); err != nil {
			t.Fatal(err)
		}
		h := b.History()
		if len(h) != 1 {
			t.Fatalf("履歴 = %d, want 1", len(h))
		}
		if err := b.Rewind(h[0].ID); err != nil {
			t.Fatal(err)
		}
		after := b.Units()
		if after[1].Pos != before[1].Pos || after[2].Alive != before[2].Alive || !b.BoardView().ZOCSides(pos(2, 1))[tactics.Enemy] {
			t.Errorf("巻き戻し後の騎兵・重騎士・ZOC = %+v", after)
		}
		if _, ok := b.PendingCanto(); ok {
			t.Error("再移動待ちが残る")
		}
		if _, err := b.Act(attack); err != nil {
			t.Fatal(err)
		}
		pc, ok := b.PendingCanto()
		if !ok || pc.Remaining != 4 {
			t.Errorf("やり直した残り = %+v, %v", pc, ok)
		}
	})
}

func TestZOCEnemyReachable(t *testing.T) {
	t.Run("TC-FR-MAP-01-06 敵軍にも自軍の重騎士の ZOC が効く", func(t *testing.T) {
		armor := ally("重騎士", 1, 0, 0)
		armor.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		enemy := foe("敵", 0, 1)
		enemy.Stats.Mov = 3
		s := tactics.Setup{Terrain: grid("#.####", "......", "######"), Units: []tactics.UnitSetup{lord(5, 1, 0), armor, enemy, boss(4, 1)}}
		b := mustNew(t, s)
		if got := b.Reachable(2)[pos(2, 1)]; got != 3 {
			t.Errorf("敵の二歩の費用 = %d, want 3", got)
		}
		if got := b.BoardView().ZOCSides(pos(1, 1)); !got[tactics.Player] {
			t.Error("自軍の角印がない")
		}
	})
}

func TestZOCDetour(t *testing.T) {
	t.Run("TC-FR-MAP-01-07 ZOC と山を避ける迂回の最小費用", func(t *testing.T) {
		knight := foe("重騎士", 2, 0)
		knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		terrain := grid("##.###", ".F....", "......")
		terrain[1][1] = tactics.Terrain{Name: "山", Cost: 3}
		s := tactics.Setup{Terrain: terrain, Units: []tactics.UnitSetup{lord(0, 1, 6), knight, boss(5, 1)}}
		b := mustNew(t, s)
		// 直進は山 3 + ZOC 2 + 平地 2 = 7。下の平地の迂回は 6。
		if got := b.Reachable(0)[pos(4, 1)]; got != 6 {
			t.Errorf("最小費用 = %d, want 6", got)
		}
	})
}

func TestZOCInvalidMove(t *testing.T) {
	t.Run("TC-NFR-07-09 ZOC 込みで届かない移動は盤面を変えない", func(t *testing.T) {
		knight := foe("重騎士", 1, 0)
		knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		s := tactics.Setup{Terrain: grid("#.####", "......", "######"), Units: []tactics.UnitSetup{lord(0, 1, 1), knight, boss(5, 1)}}
		b := mustNew(t, s)
		before := b.Units()
		history := len(b.History())
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Wait}); err != tactics.ErrUnreachable {
			t.Errorf("誤り = %v, want ErrUnreachable", err)
		}
		after := b.Units()
		for i := range before {
			if before[i].Pos != after[i].Pos || before[i].HP != after[i].HP || before[i].Acted != after[i].Acted {
				t.Errorf("失敗後の盤面が変わった: %d", i)
			}
		}
		if len(b.History()) != history {
			t.Error("失敗で履歴が増えた")
		}
	})
}

// 撃破した重騎士の隣を通らないと届かない再移動先（DES-IT07 4.3 節。REV-100 F-04）。
// 平地 4×3 の (1,1) に相手の重騎士。騎兵 Mov=5 は (0,2) から ZOC 内の (0,1) に入って攻撃する（費用 2、残り 3）。
// (2,0) へは (0,0)→(1,0)→(2,0) か (1,1)→(2,1)→(2,0)。重騎士が残れば (1,0) か (2,1) が ZOC 内で 4、倒せば 3。
func TestZOCCantoAfterKill(t *testing.T) {
	for _, kill := range []bool{false, true} {
		name := "重騎士が残れば ZOC 内を通る再移動先に届かない"
		if kill {
			name = "重騎士を倒せば ZOC が消え、その隣を通って届く"
		}
		t.Run("TC-FR-BTL-03-21 "+name, func(t *testing.T) {
			cav := ally("騎兵", 0, 2, 5)
			cav.ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
			knight := foe("重騎士", 1, 1)
			knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
			knight.AI = tactics.Hold
			knight.Stats.Mov = 0
			s := tactics.Setup{Terrain: grid("....", "....", "...."), Units: []tactics.UnitSetup{lord(3, 2, 0), cav, knight, boss(3, 0)}}
			b := mustNew(t, s)
			b.SetRoll(func() int {
				if kill {
					return 0
				}
				return 99
			})
			if kill {
				b.SetHP(2, 1)
			}
			if _, err := b.Act(tactics.Command{Unit: 1, To: pos(0, 1), Action: tactics.Attack, Target: 2}); err != nil {
				t.Fatal(err)
			}
			if b.Units()[2].Alive == kill {
				t.Fatalf("重騎士の生死 = %v, kill=%v", b.Units()[2].Alive, kill)
			}
			pending, ok := b.PendingCanto()
			if !ok || pending.Remaining != 3 {
				t.Fatalf("残り = %+v, %v; want 3", pending, ok)
			}
			got, ok := b.CantoReachable()[pos(2, 0)]
			if ok != kill || (ok && got != 3) {
				t.Errorf("(2,0) への再移動 = %d, %v; want 3, %v", got, ok, kill)
			}
		})
	}
}

// 戦死した重騎士は ZOC を持たない（TC-FR-MAP-01-07、REV-100 F-04）。
func TestZOCDeadKnight(t *testing.T) {
	t.Run("TC-FR-MAP-01-07 戦死した重騎士の隣は費用が増えない", func(t *testing.T) {
		knight := foe("重騎士", 2, 0)
		knight.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		knight.Stats.Mov = 0
		s := tactics.Setup{Terrain: grid("##.###", "......", "######"), Units: []tactics.UnitSetup{lord(0, 1, 3), boss(5, 1), knight}}
		b := mustNew(t, s)
		if got, ok := b.Reachable(0)[pos(3, 1)]; ok {
			t.Fatalf("生きた重騎士の隣を通って 3 歩先に届いた: %d", got)
		}
		b.KillUnit(2)
		if got, ok := b.Reachable(0)[pos(3, 1)]; !ok || got != 3 {
			t.Errorf("戦死後の 3 歩先 = %d, %v; want 3, true", got, ok)
		}
		if got := b.BoardView().ZOCSides(pos(2, 1)); got[tactics.Enemy] {
			t.Error("戦死した重騎士の角印が残る")
		}
	})
}

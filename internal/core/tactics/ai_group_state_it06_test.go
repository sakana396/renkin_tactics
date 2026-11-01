package tactics

import (
	"math/rand/v2"
	"testing"
)

// 一団の集合中の状態を深く復元する（TC-FR-BTL-10-33、DES-RWD-03）。
func TestGroupStateRestoreIt06(t *testing.T) {
	b := &Battle{pcg: rand.NewPCG(17, 0), groupEntered: map[string]bool{"g": false}, groupPlan: map[int]Pos{1: {4, 1}, 2: {5, 1}}, groupDanger: map[int]map[Pos]bool{1: {{3, 1}: true}}}
	snap := b.snapshot()
	b.groupEntered["g"] = true
	delete(b.groupPlan, 1)
	b.groupDanger[1][Pos{3, 1}] = false
	if err := b.restore(snap); err != nil {
		t.Fatal(err)
	}
	if b.groupEntered["g"] || b.groupPlan[1] != (Pos{4, 1}) || b.groupPlan[2] != (Pos{5, 1}) || !b.groupDanger[1][Pos{3, 1}] {
		t.Fatalf("集合状態を復元できない: entered=%v plan=%v danger=%v", b.groupEntered, b.groupPlan, b.groupDanger)
	}
}

// 集合先の同点と複数の一団の割り当てを確かめる（TC-FR-BTL-07-21、DES-AI-02）。
func TestGroupPlanOrderIt06(t *testing.T) {
	plain := Terrain{Name: "平地", Cost: 1}
	terrain := make([][]Terrain, 7)
	for y := range terrain {
		terrain[y] = make([]Terrain, 7)
		for x := range terrain[y] {
			terrain[y][x] = plain
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 2}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Def: 3, Mov: 1}, Pos: Pos{1, 3}, Weapon: w}
	foe := func(name, group string, x, y int) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: group, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 0}, Pos: Pos{x, y}, Weapon: w}
	}
	boss := foe("ボス", "", 6, 6)
	boss.Approach = Direct
	boss.Boss = true
	boss.AI = Hold
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe("先", "a", 3, 3), foe("後", "b", 3, 4), boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if b.groupPlan[1] != (Pos{3, 1}) || b.groupPlan[2] != (Pos{4, 4}) {
		t.Fatalf("一団の順・割り当て = %v", b.groupPlan)
	}
	w.MaxRange = 1
	player.Weapon = w
	first := foe("先", "a", 3, 3)
	first.Weapon = w
	makeBattle := func(u UnitSetup) *Battle {
		t.Helper()
		v, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, u, boss}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	single := makeBattle(first)
	if _, err := single.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if single.groupPlan[1] != (Pos{3, 2}) {
		t.Fatalf("Y・X の同点順 = %v", single.groupPlan)
	}
	waitingUnit := first
	waitingUnit.AI = Hold
	waiting := makeBattle(waitingUnit)
	if _, err := waiting.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if len(waiting.groupPlan) != 0 {
		t.Fatalf("待つ一団の計画 = %v", waiting.groupPlan)
	}
	waiting.mapState.charged = map[string]bool{"a": true}
	if _, err := waiting.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if waiting.units[1].AI != Charge || len(waiting.groupPlan) == 0 {
		t.Fatalf("動き出した一団 = %v / %v", waiting.units[1].AI, waiting.groupPlan)
	}
}

// 割に合うかの敵側は装備中の武器で反撃を予測する（TC-FR-BTL-07-21、DES-AI-02・DES-ITM-02）。
func TestGroupDangerEquippedIt06(t *testing.T) {
	tile := Terrain{Name: "平地", Cost: 1}
	terrain := [][]Terrain{{tile, tile, tile}, {tile, tile, tile}, {tile, tile, tile}}
	weak := Weapon{Name: "弱い剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 1}
	strong := Weapon{Name: "強い剣", Type: Sword, Might: 15, Hit: 100, MinRange: 1, MaxRange: 1}
	stats := Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 0}
	b := &Battle{terrain: terrain, units: []Unit{{Alive: true, Side: Player, HP: 20, Stats: stats, Pos: Pos{1, 1}, Weapons: []Weapon{weak}}, {Alive: true, Side: Enemy, HP: 20, Stats: stats, Pos: Pos{2, 1}, Weapons: []Weapon{weak, strong}}}}
	reach := map[int]map[Pos]int{0: b.Reachable(0)}
	threats := b.playerThreats(reach)
	if len(b.dangerFor(1, reach, threats)) == 0 {
		t.Fatal("弱い装備への攻撃が割に合わない扱い")
	}
	b.units[1].Equipped = 1
	if len(b.dangerFor(1, reach, threats)) != 0 {
		t.Fatal("強い装備の反撃を使っていない")
	}
}

// 危険範囲ができるまでの接近と、後から加わる仲間の割り当て（TC-FR-BTL-07-21、DES-AI-02）。
func TestGroupGatherAfterApproachIt06(t *testing.T) {
	road := func(width int) [][]Terrain {
		terrain := make([][]Terrain, 3)
		for y := range terrain {
			terrain[y] = make([]Terrain, width)
			for x := range terrain[y] {
				terrain[y][x] = Terrain{Name: "壁", Cost: 0}
				if y == 1 {
					terrain[y][x] = Terrain{Name: "平地", Cost: 1}
				}
			}
		}
		return terrain
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 1}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Def: 3, Mov: 2}, Pos: Pos{1, 1}, Weapon: w}
	player.Weapon.MaxRange = 2
	foe := func(name string, x, mov int) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: "g", Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: mov}, Pos: Pos{x, 1}, Weapon: w}
	}
	makeBattle := func(width, frontX, rearX int, reinforcement bool) *Battle {
		t.Helper()
		front := foe("前衛", frontX, 4)
		rear := foe("後衛", rearX, 1)
		if width == 13 {
			front.Stats.Mov, rear.Stats.Mov = 2, 2
			player.Stats.Mov = 1
		}
		boss := foe("ボス", width-1, 1)
		boss.Group, boss.Approach, boss.AI, boss.Boss = "", Direct, Hold, true
		s := Setup{Terrain: road(width), Units: []UnitSetup{player, front, boss}}
		if reinforcement {
			s.Reinforcements = []ReinforcementSetup{{Turn: 2, Unit: rear}}
		} else {
			s.Units = append(s.Units, rear)
		}
		b, _, err := New(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, tc := range []struct {
		name               string
		width, front, rear int
	}{
		{"R-01 遠い一団", 13, 9, 10},
		{"R-02 危険域のない後衛", 12, 4, 9},
	} {
		t.Run("TC-FR-BTL-07-21 "+tc.name, func(t *testing.T) {
			b := makeBattle(tc.width, tc.front, tc.rear, false)
			key := b.groupKey("g", Pos{tc.front, 1})
			for turn := 1; turn <= 2; turn++ {
				events, err := b.EndPhase()
				if err != nil {
					t.Fatal(err)
				}
				if turn == 1 && tc.front == 9 && (b.groupEntered[key] || len(b.groupPlan) != 0) {
					t.Fatalf("危険域のない番に集合を始めた: entered=%v plan=%v", b.groupEntered, b.groupPlan)
				}
				if turn == 1 && tc.front == 9 && b.units[1].Pos.X >= tc.front {
					t.Fatalf("危険域のない前衛が自軍へ寄らない: %v", b.units[1].Pos)
				}
				for _, e := range events {
					if e.Kind == EventStrike && e.Unit == 1 {
						t.Fatalf("%d 番目に前衛だけで攻撃した: %v", turn, events)
					}
				}
			}
			if tc.front == 4 && len(b.groupPlan) != 2 {
				t.Fatalf("後衛の集合先がない: %v / %v", b.groupPlan, b.units[3].Pos)
			}
			if tc.front == 4 && distance(b.groupPlan[1], b.groupPlan[3]) != 1 {
				t.Fatalf("後衛の先が前衛に隣接しない: %v", b.groupPlan)
			}
			if tc.front == 4 && b.groupPlan[3].X <= b.groupPlan[1].X {
				t.Fatalf("後衛の先が前衛より自軍側にある: %v", b.groupPlan)
			}
			if tc.front == 4 && b.groupEntered[key] {
				t.Fatalf("後衛の到着前に突入した: %v", b.groupEntered)
			}
			previous := [2]Pos{b.units[1].Pos, b.units[3].Pos}
			entered := false
			for turn := 3; turn <= 12; turn++ {
				if _, err := b.EndPhase(); err != nil {
					t.Fatal(err)
				}
				if b.groupEntered[key] {
					entered = true
					break
				}
				current := [2]Pos{b.units[1].Pos, b.units[3].Pos}
				if current == previous {
					t.Fatalf("%d 番目に一団が止まった: %v、先=%v", turn, current, b.groupPlan)
				}
				previous = current
			}
			if !entered {
				t.Fatalf("12 番目までに一団がそろわない: %v、先=%v", previous, b.groupPlan)
			}
		})
	}
	t.Run("TC-FR-BTL-07-21 2ターン目の増援", func(t *testing.T) {
		b := makeBattle(12, 4, 9, true)
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		frontGoal := b.groupPlan[1]
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if b.groupPlan[1] != frontGoal || len(b.groupPlan) != 2 {
			t.Fatalf("増援に割り当てられない: %v", b.groupPlan)
		}
		if distance(frontGoal, b.groupPlan[3]) != 1 {
			t.Fatalf("増援の先が不適切: %v / %v", b.groupPlan, b.units[3].Pos)
		}
	})
	t.Run("TC-FR-BTL-07-21 前後両側の候補は後ろ側", func(t *testing.T) {
		b := makeBattle(12, 4, 7, false)
		b.groupPlan = map[int]Pos{1: {4, 1}}
		b.planEnemyGroups(len(b.units))
		if b.groupPlan[3] != (Pos{5, 1}) {
			t.Fatalf("後衛の先は前衛の後ろ側 (5,1): %v", b.groupPlan)
		}
	})
}

// 危険がない間の攻撃と、集合中の無計画の一員の待機・退避を確かめる（TC-FR-BTL-07-21、DES-AI-02）。
func TestGroupDangerBranchIt06(t *testing.T) {
	plain := Terrain{Name: "平地", Cost: 1}
	terrain := make([][]Terrain, 3)
	for y := range terrain {
		terrain[y] = make([]Terrain, 8)
		for x := range terrain[y] {
			terrain[y][x] = plain
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 1}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 20, Str: 0, Skl: 5, Spd: 5, Def: 3, Mov: 0}, Pos: Pos{1, 1}, Weapon: w}
	foe := func(name string, x int) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: "g", Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 2}, Pos: Pos{x, 1}, Weapon: w}
	}
	boss := foe("ボス", 7)
	boss.Group, boss.Approach, boss.AI, boss.Boss = "", Direct, Hold, true
	makeBattle := func() *Battle {
		t.Helper()
		b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe("前衛", 4), foe("後衛", 5), boss}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	t.Run("TC-FR-BTL-07-21 弱い自軍には通常どおり攻撃", func(t *testing.T) {
		b := makeBattle()
		attacked := false
		for turn := 1; turn <= 3; turn++ {
			events, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range events {
				if e.Kind == EventStrike && e.Unit == 1 {
					attacked = true
				}
			}
			if b.groupEntered[b.groupKey("g", Pos{4, 1})] {
				t.Fatal("危険がないのに入った状態になった")
			}
			if attacked {
				break
			}
		}
		if !attacked {
			t.Fatal("通常の攻撃候補に進まない")
		}
	})
	t.Run("TC-FR-BTL-07-21 集合中の安全な後衛は寄らない", func(t *testing.T) {
		b := makeBattle()
		b.groupDanger = map[int]map[Pos]bool{1: {{2, 1}: true}, 2: {}}
		from := b.units[2].Pos
		if events := b.actEnemy(nil, 2); len(events) != 0 || b.units[2].Pos != from {
			t.Fatalf("集合中の後衛が通常の経路を進んだ: %v / %v", events, b.units[2].Pos)
		}
	})
	t.Run("TC-FR-BTL-07-21 危険域から安全なマスへ退避", func(t *testing.T) {
		b := makeBattle()
		from := b.units[1].Pos
		b.groupDanger = map[int]map[Pos]bool{1: {from: true}, 2: {{5, 1}: true}}
		events := b.actEnemy(nil, 1)
		if len(events) == 0 || b.units[1].Pos == from || b.groupDanger[1][b.units[1].Pos] {
			t.Fatalf("安全な逃げ場があるのに待った: %v / %v", events, b.units[1].Pos)
		}
	})
}

// 既存の集合先を全一団より先に予約する（TC-FR-BTL-07-21、DES-AI-02）。
func TestGroupPlanReservationIt06(t *testing.T) {
	terrain := make([][]Terrain, 7)
	for y := range terrain {
		for x := 0; x < 7; x++ {
			terrain[y] = append(terrain[y], Terrain{Name: "平地", Cost: 1})
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 2}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Def: 3, Mov: 1}, Pos: Pos{1, 3}, Weapon: w}
	foe := func(name, group string, x, y int) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: group, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 0}, Pos: Pos{x, y}, Weapon: w}
	}
	boss := foe("ボス", "", 6, 6)
	boss.Approach, boss.AI, boss.Boss = Direct, Hold, true
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe("先", "a", 3, 3), foe("中", "b", 3, 4), foe("既存", "c", 6, 5), boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	b.groupPlan = map[int]Pos{3: {4, 4}}
	b.planEnemyGroups(len(b.units))
	if b.groupPlan[3] != (Pos{4, 4}) || b.groupPlan[1] == b.groupPlan[2] || b.groupPlan[1] == b.groupPlan[3] || b.groupPlan[2] == b.groupPlan[3] {
		t.Fatalf("集合先が重なった: %v", b.groupPlan)
	}
	if b.groupPlan[1] != (Pos{3, 1}) {
		t.Fatalf("最初の一団の先 = %v", b.groupPlan)
	}
}

// 危険範囲を事前に作る場合と敵ごとに作る場合を比べる（TC-NFR-06-07、DES-AI-02）。
func TestGroupThreatCacheSameIt06(t *testing.T) {
	tile := Terrain{Name: "平地", Cost: 1}
	terrain := make([][]Terrain, 5)
	for y := range terrain {
		for x := 0; x < 5; x++ {
			terrain[y] = append(terrain[y], tile)
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 2}
	stats := Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Def: 3, Mov: 2}
	b := &Battle{terrain: terrain, units: []Unit{{Alive: true, Side: Player, HP: 20, Stats: stats, Pos: Pos{1, 1}, Weapons: []Weapon{w}}, {Alive: true, Side: Enemy, HP: 20, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3}, Pos: Pos{3, 1}, Weapons: []Weapon{w}}, {Alive: true, Side: Enemy, HP: 20, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3}, Pos: Pos{3, 3}, Weapons: []Weapon{w}}}}
	reach := b.playerReach()
	cached := b.playerThreats(reach)
	for i := 1; i <= 2; i++ {
		// 個別計算は playerThreats を通さず、到達先と射程を直接展開する。
		fresh := make(map[Pos]bool)
		for from := range reach[0] {
			for y := range terrain {
				for x := range terrain[y] {
					p := Pos{x, y}
					if inRange(w, distance(from, p)) {
						fresh[p] = true
					}
				}
			}
		}
		want := b.dangerFor(i, reach, map[int]map[Pos]bool{0: fresh})
		got := b.dangerFor(i, reach, cached)
		if len(got) != len(want) {
			t.Fatalf("敵%d: 再利用=%v 個別=%v", i, got, want)
		}
		for p := range want {
			if !got[p] {
				t.Fatalf("敵%d: %v の判定が違う", i, p)
			}
		}
	}
}

// 近さの異なる境目では、自軍へ近い方を選ぶ（TC-FR-BTL-07-21、DES-AI-02）。
func TestGroupBoundaryClosenessIt06(t *testing.T) {
	terrain := make([][]Terrain, 7)
	for y := range terrain {
		for x := 0; x < 7; x++ {
			terrain[y] = append(terrain[y], Terrain{Name: "平地", Cost: 1})
		}
	}
	terrain[1][2] = Terrain{Name: "森", Cost: 4}
	terrain[2][3] = Terrain{Name: "森", Cost: 4}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 2}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Def: 3, Mov: 1}, Pos: Pos{1, 3}, Weapon: w}
	foe := UnitSetup{Name: "敵", Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: "g", Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 0}, Pos: Pos{3, 3}, Weapon: w}
	boss := foe
	boss.Name, boss.Group, boss.Approach, boss.AI, boss.Boss, boss.Pos = "ボス", "", Direct, Hold, true, Pos{6, 6}
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe, boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	b.planEnemyGroups(len(b.units))
	near := b.closeness()
	danger := b.groupDanger[1]
	minNear, maxNear := unreachable, -1
	for y := range terrain {
		for x := range terrain[y] {
			p := Pos{x, y}
			if danger[p] || p == player.Pos || p == foe.Pos || p == boss.Pos {
				continue
			}
			boundary := false
			for _, d := range directions {
				if danger[Pos{x + d.X, y + d.Y}] {
					boundary = true
				}
			}
			if !boundary {
				continue
			}
			n := near(p)
			if n < minNear {
				minNear = n
			}
			if n > maxNear {
				maxNear = n
			}
		}
	}
	if minNear >= maxNear {
		t.Fatalf("近さの異なる境目がない: %d / %d", minNear, maxNear)
	}
	if got := near(b.groupPlan[1]); got != minNear {
		t.Fatalf("境目の近さ = %d、最小 = %d、先 = %v", got, minNear, b.groupPlan[1])
	}
}

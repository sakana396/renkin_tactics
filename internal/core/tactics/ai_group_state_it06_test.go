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

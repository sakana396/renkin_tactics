package tactics

import "testing"

// 仲間の近くに安全な境目がないときも、遠い境目を割り当てない（TC-FR-MAP-04-28、DES-AI-02）。
func TestGroupGoalNearTeammateIt06(t *testing.T) {
	terrain := make([][]Terrain, 16)
	for y := range terrain {
		terrain[y] = make([]Terrain, 16)
		for x := range terrain[y] {
			terrain[y][x] = Terrain{Name: "平地", Cost: 1}
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 1}
	player := UnitSetup{Name: "自軍", Side: Player, Lord: true, Pos: Pos{1, 1}, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Mov: 1}, Weapon: w}
	foe := func(name string, p Pos) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: "fort_front", Pos: p, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Mov: 5}, Weapon: w}
	}
	boss := foe("ボス", Pos{15, 0})
	boss.Boss, boss.Approach, boss.Group, boss.AI = true, Direct, "", Hold
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe("剣士", Pos{9, 8}), foe("弓兵", Pos{12, 7}), boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if distance(b.groupPlan[1], Pos{12, 7}) > groupNearLimit || distance(b.groupPlan[2], Pos{9, 8}) > groupNearLimit {
		t.Fatalf("仲間から離れた境目を割り当てた: plan=%v", b.groupPlan)
	}
}

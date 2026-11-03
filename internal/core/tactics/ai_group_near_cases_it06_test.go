package tactics

import "testing"

// 集合先の再割り当てと移動先の近さ・待機・退避を調べる（TC-FR-MAP-04-29、DES-AI-02）。
func TestGroupNearCasesIt06(t *testing.T) {
	terrain := make([][]Terrain, 3)
	for y := range terrain {
		terrain[y] = make([]Terrain, 16)
		for x := range terrain[y] {
			terrain[y][x] = Terrain{Name: "平地", Cost: 1}
		}
	}
	w := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 100, MinRange: 1, MaxRange: 1}
	player := UnitSetup{Name: "自軍", Side: Player, Lord: true, Pos: Pos{0, 1}, Stats: Stats{HP: 20, Str: 20, Skl: 5, Spd: 5, Mov: 0}, Weapon: w}
	foe := func(name string, p Pos) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Charge, Approach: GroupApproach, Group: "g", Pos: p, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Mov: 1}, Weapon: w}
	}
	boss := foe("ボス", Pos{15, 0})
	boss.Boss, boss.Approach, boss.Group, boss.AI = true, Direct, "", Hold
	newBattle := func(t *testing.T) *Battle {
		t.Helper()
		b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, foe("剣士", Pos{8, 1}), foe("弓兵", Pos{9, 1}), boss}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	t.Run("TC-FR-MAP-04-29 範囲外の保存先を割り当て直す", func(t *testing.T) {
		b := newBattle(t)
		b.groupPlan = map[int]Pos{1: {0, 0}, 2: {0, 2}}
		b.planEnemyGroups(len(b.units))
		for _, i := range []int{1, 2} {
			goal, ok := b.groupPlan[i]
			if !ok || (goal == (Pos{0, 0}) || goal == (Pos{0, 2})) || !b.groupNear(goal, i, []int{1, 2}) {
				t.Fatalf("一員 %d の割り当て直し = %v / %v", i, goal, b.groupPlan)
			}
		}
	})
	t.Run("TC-FR-MAP-04-29 範囲外の保存先を破棄して安全な現在位置で待つ", func(t *testing.T) {
		b := newBattle(t)
		b.units[0].Weapons[0].Type = Staff // この小ケースでは危険範囲を空にする。
		key := b.groupKey("g", b.units[1].Pos)
		b.groupGathering[key] = true
		b.groupPlan = map[int]Pos{1: {0, 0}, 2: {0, 2}}
		b.planEnemyGroups(len(b.units))
		if len(b.groupPlan) != 0 || !b.groupGathering[key] {
			t.Fatalf("先と集合状態 = %v / %v", b.groupPlan, b.groupGathering)
		}
		from := b.units[1].Pos
		if events := b.actEnemy(nil, 1); len(events) != 0 || b.units[1].Pos != from {
			t.Fatalf("安全な現在位置から動いた: %v / %v", events, b.units[1].Pos)
		}
	})
	for _, tc := range []struct {
		name   string
		reach  map[Pos]int
		danger map[Pos]bool
		want   Pos
		found  bool
	}{
		{"移動先は仲間から 7 以内", map[Pos]int{{1, 1}: 1, {3, 1}: 1}, nil, Pos{3, 1}, true},
		{"安全な先がなく現在位置が安全なら待つ", map[Pos]int{{1, 1}: 1}, nil, Pos{}, false},
		{"現在位置が危険なら 7 を超えて退避", map[Pos]int{{1, 1}: 1}, map[Pos]bool{{8, 1}: true}, Pos{1, 1}, true},
		{"境目の 7 は許し 8 は除く", map[Pos]int{{1, 1}: 1, {2, 1}: 1}, nil, Pos{2, 1}, true},
	} {
		t.Run("TC-FR-MAP-04-29 "+tc.name, func(t *testing.T) {
			b := newBattle(t)
			b.groupDanger = map[int]map[Pos]bool{1: tc.danger}
			got, found := b.groupMove(1, tc.reach, Pos{0, 1})
			if found != tc.found || (found && got != tc.want) {
				t.Fatalf("移動先 = %v / %v、期待 = %v / %v", got, found, tc.want, tc.found)
			}
		})
	}
}

package tactics

import "testing"

func TestZOCAICloseness(t *testing.T) {
	t.Run("TC-FR-BTL-07-23 敵の近さと一団の経路は進入先の ZOC を数える", func(t *testing.T) {
		plain := Terrain{Name: "平地", Cost: 1}
		wall := Terrain{Name: "壁", Cost: 0}
		terrain := [][]Terrain{{wall, wall, plain, wall, wall}, {plain, plain, plain, plain, plain}, {wall, wall, wall, wall, wall}}
		weapon := Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}
		s := Setup{Terrain: terrain, Units: []UnitSetup{
			{Name: "主人公", Side: Player, Stats: Stats{HP: 20, Mov: 0}, Pos: Pos{4, 1}, Weapon: weapon, Lord: true},
			{Name: "重騎士", Side: Player, Stats: Stats{HP: 20, Mov: 0}, Pos: Pos{2, 0}, Weapon: weapon, ClassSkills: []Skill{{Name: "通せんぼ", Effect: ZOC}}},
			{Name: "敵", Side: Enemy, Level: 1, Stats: Stats{HP: 20, Mov: 3}, Pos: Pos{0, 1}, Weapon: weapon, Boss: true},
		}}
		b, _, err := New(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := b.closeness()(Pos{0, 1}); got != 4 {
			t.Errorf("敵から最も近い自軍への近さ = %d, want 4", got)
		}
		if got := b.terrainDistances(Pos{0, 1}, false, Enemy)[Pos{4, 1}]; got != 5 {
			t.Errorf("前向きの集合経路 = %d, want 5", got)
		}
		if got := b.terrainDistances(Pos{4, 1}, true, Enemy)[Pos{0, 1}]; got != 5 {
			t.Errorf("逆向きの集合経路 = %d, want 5", got)
		}
		if _, ok := b.Reachable(2)[Pos{3, 1}]; ok {
			t.Error("敵が ZOC を無視して 3 歩先へ届いた")
		}
	})
}

// 出発マスか集合先が ZOC の中にある非対称の盤面（REV-100 F-05）。逆向きの探索は進入先（cur）の
// Cost+Z を数え、前向きの費用と一致する。出発マスの Z は数えず、集合先の Z は数える。
func TestZOCTerrainDistancesAsymmetric(t *testing.T) {
	plain := Terrain{Name: "平地", Cost: 1}
	weapon := Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}
	row := func() []Terrain { return []Terrain{plain, plain, plain, plain, plain} }
	for _, tc := range []struct {
		name  string
		armor Pos
		want  int
	}{
		{"集合先が ZOC の中", Pos{4, 0}, 5},
		{"出発マスが ZOC の中", Pos{0, 0}, 4},
	} {
		t.Run("TC-FR-BTL-07-23 "+tc.name, func(t *testing.T) {
			s := Setup{Terrain: [][]Terrain{row(), row(), row()}, Units: []UnitSetup{
				{Name: "主人公", Side: Player, Stats: Stats{HP: 20}, Pos: Pos{2, 2}, Weapon: weapon, Lord: true},
				{Name: "重騎士", Side: Player, Stats: Stats{HP: 20}, Pos: tc.armor, Weapon: weapon, ClassSkills: []Skill{{Name: "通せんぼ", Effect: ZOC}}},
				{Name: "敵", Side: Enemy, Level: 1, Stats: Stats{HP: 20, Mov: 3}, Pos: Pos{0, 1}, Weapon: weapon, Boss: true},
			}}
			b, _, err := New(s, 1)
			if err != nil {
				t.Fatal(err)
			}
			if got := b.terrainDistances(Pos{0, 1}, false, Enemy)[Pos{4, 1}]; got != tc.want {
				t.Errorf("前向き (0,1)→(4,1) = %d, want %d", got, tc.want)
			}
			if got := b.terrainDistances(Pos{4, 1}, true, Enemy)[Pos{0, 1}]; got != tc.want {
				t.Errorf("逆向き (0,1)→(4,1) = %d, want %d", got, tc.want)
			}
		})
	}
}

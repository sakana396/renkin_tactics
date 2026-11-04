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

package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestBaitUnit は釣り役の除外・優先順位・一巡を確かめる（TC-DES-NG-23-04、DES-NG-23）。
func TestBaitUnit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		defs    []int
		failed  map[int]bool
		want    int
		cleared bool
	}{
		{"倒れる確率が最小", []int{0, 0, 0, 0, 3}, nil, 4, false},
		{"同点は守備とHP", []int{0, 0, 0, 2, 3}, nil, 4, false},
		{"さらに同点は添字順", []int{0, 0, 0, 3, 3}, nil, 3, false},
		{"巻き戻した候補を除く", []int{0, 0, 0, 3, 3}, map[int]bool{3: true}, 4, false},
		{"候補が尽きたら一巡", []int{0, 0, 0, 3, 3}, map[int]bool{3: true, 4: true}, 3, true},
	} {
		t.Run("TC-DES-NG-23-04 "+tc.name, func(t *testing.T) {
			setup, err := core.NewSetup(baseCatalog(t), "ch01")
			if err != nil {
				t.Fatal(err)
			}
			var player, enemy tactics.UnitSetup
			for _, u := range setup.Units {
				if u.Side == tactics.Player && !u.Lord && len(u.Weapons) > 0 && u.Weapons[0].Type != tactics.Staff {
					player = u
				}
				if u.Side == tactics.Enemy && len(u.Weapons) > 0 {
					enemy = u
				}
			}
			setup.Terrain = make([][]tactics.Terrain, 3)
			for y := range setup.Terrain {
				setup.Terrain[y] = make([]tactics.Terrain, 5)
				for x := range setup.Terrain[y] {
					setup.Terrain[y][x] = tactics.Terrain{Name: "平地", Cost: 1}
				}
			}
			setup.Units = nil
			for i := 0; i < 5; i++ {
				u := player
				u.Name = string(rune('A' + i))
				u.Pos = tactics.Pos{X: i, Y: 0}
				u.Lord = i == 0
				u.Stats.HP = 5
				u.Stats.Def = tc.defs[i]
				u.Stats.Mov = 1
				u.Skill = tactics.Skill{}
				u.ClassSkills = nil
				u.Weapons = append([]tactics.Weapon(nil), player.Weapons...)
				if i == 1 {
					u.AllowedWeaponTypes = append(u.AllowedWeaponTypes, tactics.Staff)
					u.Weapons[0].Type = tactics.Staff
					u.Weapons[0].Heal = 5
					u.Weapons[0].Uses = 10
					u.Weapons[0].MaxUses = 10
				}
				setup.Units = append(setup.Units, u)
			}
			enemy.Boss = true
			enemy.Commander = false
			enemy.Pos = tactics.Pos{X: 2, Y: 2}
			enemy.Stats.Mov = 3
			enemy.Stats.Str = 1
			enemy.Weapons = append([]tactics.Weapon(nil), enemy.Weapons...)
			enemy.Weapons[0].Might = 2
			enemy.Weapons[0].Crit = 5
			enemy.Skill = tactics.Skill{}
			enemy.ClassSkills = nil
			setup.Units = append(setup.Units, enemy)
			setup.Reinforcements = nil
			setup.Events = nil
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			p := &smartPlayer{autoPlayer: &autoPlayer{b: b}, f: smartFeatures{Threat: true}, baitFailed: map[int]bool{2: true}}
			p.lastKill = -4
			for i, v := range tc.failed {
				p.baitFailed[i] = v
			}
			got := p.baitUnit()
			if got != tc.want {
				t.Errorf("釣り役 = %d、期待 %d", got, tc.want)
			}
			if tc.cleared && len(p.baitFailed) != 0 {
				t.Errorf("一巡後の除外 = %v", p.baitFailed)
			}
		})
	}
}

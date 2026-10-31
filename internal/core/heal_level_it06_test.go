package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 兵種の杖の回復量の上限（DES-IT06 7.1 節）と敵のレベル（7.3 節）を、定義から戦闘へ渡す変換の試験。

// healCapCatalog は、(0, 0) の主人公と (1, 0) の錬金術師（魔力 3、兵種の上限 healCap、癒やしの杖）、
// (3, 0) の Lv 5 のボスと、2 ターン目の Lv 7 の増援を持つ目録を返す。
func healCapCatalog(healCap int) *defs.Catalog {
	return defs.NewCatalog(
		defs.GameDef{Party: []string{"hero", "noel"}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{
			{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 24, Str: 5, Skl: 7, Spd: 8, Def: 4, Mov: 5}},
			{ID: "alchemist", Name: "錬金術師", WeaponType: "staff", StaffHealCap: healCap, Base: defs.Stats{HP: 18, Mov: 5, Mag: 3}},
			{ID: "soldier", Name: "ソルジャー", WeaponType: "lance", Base: defs.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5}},
		},
		[]defs.WeaponDef{
			{ID: "iron_sword", Name: "鉄の剣", Type: "sword", Weight: 4, Strikes: 1, DamageType: "physical", Might: 5, Hit: 90, MinRange: 1, MaxRange: 1},
			{ID: "iron_lance", Name: "鉄の槍", Type: "lance", Weight: 5, Strikes: 1, DamageType: "physical", Might: 7, Hit: 80, MinRange: 1, MaxRange: 1},
			{ID: "heal_staff", Name: "癒やしの杖", Type: "staff", Strikes: 1, DamageType: "physical", Heal: 10, Uses: 15, MinRange: 1, MaxRange: 1},
		},
		nil, nil,
		[]defs.CharacterDef{
			{ID: "hero", Name: "主人公", Class: "lord", Weapon: "iron_sword", Level: 1, Lord: true},
			{ID: "noel", Name: "錬金術師", Class: "alchemist", Weapon: "heal_staff", Level: 1},
		},
		[]defs.MapDef{{
			ID: "m", Name: "小さなマップ", Rows: []string{"...."},
			Slots:          []defs.Slot{{X: 0, Y: 0}, {X: 1, Y: 0}},
			Enemies:        []defs.EnemyPlacement{{Level: 5, Class: "soldier", Weapon: "iron_lance", AI: "hold", Boss: true, X: 3, Y: 0}},
			Reinforcements: []defs.Reinforcement{{Turn: 2, EnemyPlacement: defs.EnemyPlacement{Level: 7, Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 2, Y: 0}}},
		}},
		[]defs.ChapterDef{{ID: "c1", Name: "章", Map: "m"}},
		nil,
	)
}

func TestIT06HealCapAndLevelConversion(t *testing.T) {
	t.Run("TC-FR-BTL-03-19 兵種の staff_heal_cap を自軍の戦闘の入力に写す", func(t *testing.T) {
		for _, healCap := range []int{0, 8} {
			setup, err := core.NewSetup(healCapCatalog(healCap), "m")
			if err != nil {
				t.Fatal(err)
			}
			if got := [2]int{setup.Units[0].StaffHealCap, setup.Units[1].StaffHealCap}; got != [2]int{0, healCap} {
				t.Errorf("上限 %d: 主人公・錬金術師の StaffHealCap = %v", healCap, got)
			}
		}
	})

	t.Run("TC-FR-UNT-04-18 初めの敵とターンの増援に定義の level を渡す", func(t *testing.T) {
		setup, err := core.NewSetup(healCapCatalog(0), "m")
		if err != nil {
			t.Fatal(err)
		}
		if setup.Units[2].Level != 5 || setup.Reinforcements[0].Unit.Level != 7 {
			t.Fatalf("敵のレベル %d、増援のレベル %d", setup.Units[2].Level, setup.Reinforcements[0].Unit.Level)
		}
	})
}

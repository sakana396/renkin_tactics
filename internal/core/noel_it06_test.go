package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 錬金術師の加入と出撃枠を、DES-IT06 11・13 章の値から確かめる（DES-CMP-11）。
func TestNoelAndSlotsIT06(t *testing.T) {
	cat := baseCatalog(t)
	t.Run("TC-FR-CMP-08-06 加入順・9人・武器と能力", func(t *testing.T) {
		chapter := cat.Chapters()[1]
		if !reflect.DeepEqual(chapter.Joins, []string{"noel", "elsa"}) {
			t.Fatal(chapter.Joins)
		}
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Roster) != 9 || c.Roster[5].Character != "noel" || c.Roster[6].Character != "elsa" {
			t.Fatal(c.Roster)
		}
		m := c.Roster[5]
		if m.Level != 2 || m.Stats != (tactics.Stats{HP: 17, Str: 2, Skl: 5, Spd: 6, Def: 2, Mov: 5, Lck: 4, Mag: 3, Res: 3}) || !reflect.DeepEqual(m.Weapons, []string{"fire", "heal_staff"}) || len(m.Items) != 0 {
			t.Fatal(m)
		}
		cls, _ := cat.Class("alchemist")
		if cls.StaffHealCap != 8 || !reflect.DeepEqual(cls.AllowedWeaponTypes(), []string{"magic", "staff"}) {
			t.Fatal(cls)
		}
		mage, _ := cat.Class("mage")
		if mage.Base != (defs.Stats{HP: 16, Str: 1, Skl: 5, Spd: 5, Def: 1, Mov: 5, Lck: 2, Mag: 4, Res: 3}) || !reflect.DeepEqual(mage.AllowedWeaponTypes(), []string{"magic"}) {
			t.Fatal(mage)
		}
		c2, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		setup, _, err := c2.StartBattle([]int{0, 1, 2, 3, 4, 5, 6}, 1)
		if err != nil {
			t.Fatal(err)
		}
		u := setup.Battle.Units()[5]
		if u.StaffHealCap != 8 || !reflect.DeepEqual(u.AllowedWeaponTypes, []tactics.WeaponType{tactics.Magic, tactics.Staff}) || len(u.Weapons) != 2 {
			t.Fatal(u)
		}
	})
	t.Run("TC-FR-CMP-08-07 古いセーブの不足だけを追いつかせる", func(t *testing.T) {
		for _, chapter := range []string{"ch01", "ch02", "ch09"} {
			c, err := core.NewCampaignAt(cat, core.Permadeath, chapter)
			if err != nil {
				t.Fatal(err)
			}
			if chapter != "ch01" {
				c.Roster = append(c.Roster[:5], c.Roster[6:]...)
			}
			var b bytes.Buffer
			if err := c.Save(&b); err != nil {
				t.Fatal(err)
			}
			got, err := core.LoadCampaign(cat, &b)
			if err != nil {
				t.Fatal(err)
			}
			found := 0
			for _, m := range got.Roster {
				if m.Character == "noel" {
					found++
				}
			}
			want := 1
			if chapter == "ch01" {
				want = 0
			}
			if found != want {
				t.Fatal(chapter, got.Roster)
			}
			b.Reset()
			if err := got.Save(&b); err != nil {
				t.Fatal(err)
			}
			again, err := core.LoadCampaign(cat, &b)
			if err != nil || !sameRecruitCampaign(got, again) {
				t.Fatal(err, again)
			}
		}
		c, _ := core.NewCampaignAt(cat, core.Permadeath, "ch02")
		c.Roster[5].Alive = false
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		got, err := core.LoadCampaign(cat, &b)
		if err != nil || got.Roster[5].Alive {
			t.Fatal(err, got)
		}
	})
	t.Run("TC-FR-CMP-03-15 全9章の枠と新しい枠の安全", func(t *testing.T) {
		want := []int{6, 7, 6, 7, 6, 7, 6, 6, 7}
		for i, id := range []string{"ch01", "ch02", "ch03", "ch04", "ch05", "ch06", "ch07", "ch08", "ch09"} {
			m, _ := cat.Map(id)
			if len(m.Slots) != want[i] {
				t.Errorf("%s: %d", id, len(m.Slots))
				continue
			}
			added := m.Slots[len(m.Slots)-1]
			terrainCost := 0
			for _, terrain := range cat.Terrains() {
				if terrain.Symbol == string([]rune(m.Rows[added.Y])[added.X]) {
					terrainCost = terrain.Cost
				}
			}
			if terrainCost < 1 {
				t.Errorf("%s: 通れない枠 %v", id, added)
			}
			for _, e := range m.Enemies {
				if e.X == added.X && e.Y == added.Y {
					t.Errorf("%s: 敵と重なる", id)
				}
			}
			c, _ := core.NewCampaignAt(cat, core.Withdraw, id)
			battle, _, err := c.StartBattle([]int{0}, 1)
			if err != nil {
				t.Fatal(id, err)
			}
			units := battle.Battle.Units()
			for j, u := range units {
				if u.Side != tactics.Enemy {
					continue
				}
				for pos := range battle.Battle.Reachable(j) {
					for _, target := range battle.Battle.AttackRange(j, pos) {
						if target == (tactics.Pos{X: added.X, Y: added.Y}) {
							t.Errorf("%s: 1ターン目の敵 %s が届く枠 %v", id, u.Name, added)
							break
						}
					}
				}
			}
		}
		c, _ := core.NewCampaignAt(cat, core.Withdraw, "ch01")
		if _, _, err := c.StartBattle([]int{0, 1, 2, 3, 4}, 1); err != nil {
			t.Fatal(err)
		}
	})
}

// 持てる種類を準備・保存・戦闘の入口で一貫して使う（DES-CMP-11）。
func TestNoelWeaponTypesIT06(t *testing.T) {
	t.Run("TC-FR-UNT-02-03 魔法と杖を持てる", func(t *testing.T) {
		cat := baseCatalog(t)
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		c.StockWeapons = []core.WeaponCount{{Weapon: "fire", Count: 1}, {Weapon: "heal_staff", Count: 1}, {Weapon: "iron_sword", Count: 1}}
		if err := c.TakeWeapon(5, "heal_staff"); err != nil {
			t.Fatal(err)
		}
		if err := c.TakeWeapon(5, "iron_sword"); err != core.ErrWeaponType {
			t.Fatal(err)
		}
		if err := c.TakeWeapon(6, "fire"); err != core.ErrWeaponType {
			t.Fatal(err)
		}
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		got, err := core.LoadCampaign(cat, &b)
		if err != nil || !reflect.DeepEqual(got.Roster[5].Weapons, []string{"fire", "heal_staff", "heal_staff"}) {
			t.Fatal(err, got)
		}
	})
}

package core

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// TestNG08Enemies は実データと戦闘計算を設計例に照合する（DES-NG-24）。
func TestNG08Enemies(t *testing.T) {
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("TC-DES-NG-24-08 新兵種の絵が未配置", func(t *testing.T) {
		for _, id := range []string{"rabble", "mage"} {
			t.Run(id, func(t *testing.T) {
				_, err := fs.Stat(data.AlchemyMedia(), "art/units/enemy_"+id+".png")
				if !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("絵の欠落: %v", err)
				}
			})
		}
	})
	t.Run("TC-DES-NG-24-01 兵種の定義", func(t *testing.T) {
		for _, tc := range []struct {
			id     string
			stats  defs.Stats
			weapon string
		}{
			{"rabble", defs.Stats{HP: 10, Spd: 3, Mov: 5}, "sword"},
			{"mage", defs.Stats{HP: 16, Str: 1, Skl: 5, Spd: 5, Def: 1, Mov: 5, Lck: 2, Mag: 4, Res: 3}, "magic"},
			{"spearman", defs.Stats{HP: 16, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 5, Lck: 3, Res: 1}, "lance"},
		} {
			t.Run(tc.id, func(t *testing.T) {
				c, ok := cat.Class(tc.id)
				if !ok || c.Base != tc.stats || !reflect.DeepEqual(c.WeaponTypes, []string{tc.weapon}) || len(c.Skills) != 0 {
					t.Errorf("兵種 = %+v", c)
				}
			})
		}
	})
	t.Run("TC-DES-NG-24-02 武器の定義", func(t *testing.T) {
		for _, tc := range []struct {
			id, kind, damage        string
			might, hit, weight, max int
			tags                    []string
		}{
			{"crude_sword", "sword", "physical", 3, 75, 1, 1, nil}, {"fire", "magic", "magic", 5, 85, 2, 2, nil}, {"armor_breaker", "lance", "physical", 5, 70, 5, 1, []string{"armor"}},
		} {
			t.Run(tc.id, func(t *testing.T) {
				w, ok := cat.Weapon(tc.id)
				if !ok || w.Type != tc.kind || w.DamageType != tc.damage || w.Might != tc.might || w.Hit != tc.hit || w.Weight != tc.weight || w.Crit != 0 || w.Strikes != 1 || w.MinRange != 1 || w.MaxRange != tc.max || w.Uses != 0 || !reflect.DeepEqual(append([]string(nil), w.EffectiveTags...), tc.tags) {
					t.Errorf("武器 = %+v", w)
				}
			})
		}
	})
	for _, tc := range []struct {
		name, class, weapon, target   string
		equip, damage, hit, count, hp int
	}{
		{"TC-DES-NG-24-03 弱い敵", "rabble", "crude_sword", "haru", 1, 2, 63, 8, 16},
		{"TC-DES-NG-24-04 魔道士", "mage", "fire", "dain", 0, 8, 89, 3, 24},
		{"TC-DES-NG-24-05 重装特効", "spearman", "armor_breaker", "dain", 0, 11, 74, 3, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enemy, err := enemyUnit(cat, defs.EnemyPlacement{Class: tc.class, Weapons: []string{tc.weapon}, Level: 1, AI: "hold", X: 0, Y: 0}, "敵")
			if err != nil {
				t.Fatal(err)
			}
			ally, err := allyUnit(cat, tc.target, defs.Slot{X: 1, Y: 0})
			if err != nil {
				t.Fatal(err)
			}
			enemy.Side = tactics.Player
			enemy.Lord = true
			ally.Side = tactics.Enemy
			ally.Lord = false
			ally.Skill = tactics.Skill{}
			ally.ClassSkills = nil
			ally.Equipped = 0
			ally.Weapons = ally.Weapons[:1]
			if tc.target == "haru" {
				ally.Weapons[0].Weight = 0
			}
			ally.Growth = tactics.Growth{}
			ally.Items = nil
			ally.Boss = true
			b, _, err := tactics.New(tactics.Setup{Terrain: [][]tactics.Terrain{{{Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}}}, Units: []tactics.UnitSetup{enemy, ally}}, 1)
			if err != nil {
				t.Fatal(err)
			}
			f, err := b.Forecast(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.Attack, Target: 1})
			if err != nil {
				t.Fatal(err)
			}
			if f.Attacker.Damage != tc.damage || f.Attacker.HitRate != tc.hit || f.Attacker.Count != 1 || f.Attacker.CritRate != 0 || ally.Stats.HP != tc.hp || (tc.hp-1)/f.Attacker.Damage+1 != tc.count {
				t.Errorf("予測 = %+v, HP=%d", f.Attacker, ally.Stats.HP)
			}
			// 各回の残り HP で戦闘を作り直し、命中した攻撃の実結果を追う。
			remaining := tc.hp
			for hit := 1; hit <= tc.count; hit++ {
				ally.Stats.HP = remaining
				fight, _, err := tactics.New(tactics.Setup{Terrain: [][]tactics.Terrain{{{Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}}}, Units: []tactics.UnitSetup{enemy, ally}}, 1)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := fight.Act(tactics.Command{Unit: 0, To: tactics.Pos{}, Action: tactics.Attack, Target: 1}); err != nil {
					t.Fatal(err)
				}
				target := fight.Units()[1]
				remaining -= tc.damage
				if remaining < 0 {
					remaining = 0
				}
				if target.HP != remaining || target.Alive != (hit < tc.count) {
					t.Fatalf("%d 回目: HP=%d 生存=%v", hit, target.HP, target.Alive)
				}
			}
		})
	}
	for _, tc := range []struct {
		name, id      string
		damage, after int
		falls         bool
	}{{"TC-DES-NG-24-06 小爆弾", "bomb_small", 8, 2, false}, {"TC-DES-NG-24-07 大爆弾", "bomb_large", 12, 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			enemy, err := enemyUnit(cat, defs.EnemyPlacement{Class: "rabble", Weapons: []string{"crude_sword"}, Level: 1, AI: "hold", X: 2, Y: 0}, "敵")
			if err != nil {
				t.Fatal(err)
			}
			enemy.Boss = true
			ally, err := allyUnit(cat, "aoi", defs.Slot{X: 0, Y: 0})
			if err != nil {
				t.Fatal(err)
			}
			item, ok := cat.Item(tc.id)
			if !ok {
				t.Fatal(tc.id)
			}
			ally.Items = []tactics.ItemStack{{Item: ItemOf(item), Count: 1}}
			reserve := ally
			reserve.Name = "控え"
			reserve.Lord = false
			reserve.Pos = tactics.Pos{X: 3, Y: 0}
			b, _, err := tactics.New(tactics.Setup{Terrain: [][]tactics.Terrain{{{Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}}}, Units: []tactics.UnitSetup{ally, enemy, reserve}}, 1)
			if err != nil {
				t.Fatal(err)
			}
			center := tactics.Pos{X: 2, Y: 0}
			f, err := b.BlastForecast(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.UseItem, Item: 0, Blast: &center})
			if err != nil {
				t.Fatal(err)
			}
			if f.Damage != tc.damage {
				t.Errorf("威力 %d", f.Damage)
			}
			found := false
			for _, target := range f.Targets {
				if target.Unit == 1 {
					found = true
					if target.HP != 10 || target.After != tc.after || target.Falls != tc.falls {
						t.Errorf("対象 %+v", target)
					}
				}
			}
			if !found {
				t.Fatal("敵が範囲外")
			}
			if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{}, Action: tactics.UseItem, Item: 0, Blast: &center}); err != nil {
				t.Fatal(err)
			}
			target := b.Units()[1]
			if target.HP != tc.after || target.Alive == tc.falls {
				t.Errorf("実結果: HP=%d 生存=%v", target.HP, target.Alive)
			}
		})
	}
}

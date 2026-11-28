package core_test

import (
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"testing"
)

// TestBombOption は爆弾の選択と在庫を確かめる（DES-NG-23）。
func TestBombOption(t *testing.T) {
	for _, tc := range []struct {
		id           string
		hp, count    int
		ally, strong bool
		want         bool
	}{
		{"TC-DES-NG-23-05 敵が固まる", 8, 1, false, false, true},
		{"TC-DES-NG-23-06 味方を巻き込む", 8, 1, true, false, false},
		{"TC-DES-NG-23-07 撃破なし", 20, 1, false, false, false},
		{"TC-DES-NG-23-08 在庫なし", 8, 0, false, false, false},
		{"TC-DES-NG-23-10 通常攻撃が優位", 8, 1, false, true, true},
	} {
		t.Run(tc.id, func(t *testing.T) {
			setup, err := core.NewSetup(baseCatalog(t), "ch01")
			if err != nil {
				t.Fatal(err)
			}
			var player, enemy tactics.UnitSetup
			for _, u := range setup.Units {
				if u.Side == tactics.Player && len(u.Weapons) > 0 {
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
			player.Pos = tactics.Pos{X: 0, Y: 1}
			player.Lord = true
			player.Stats.Mov = 0
			player.Weapons = append([]tactics.Weapon(nil), player.Weapons...)
			player.Weapons[0].Type = tactics.Sword
			player.Weapons[0].Heal, player.Weapons[0].Uses, player.Weapons[0].MaxUses = 0, 0, 0
			player.AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword}
			player.Weapons[0].MinRange = 1
			player.Weapons[0].MaxRange = 3
			player.Weapons[0].Might = 0
			player.Stats.Str = 0
			player.Stats.Mag = 0
			if tc.strong {
				player.Weapons[0].MaxRange = 4
				player.Weapons[0].Might = 50
				player.Weapons[0].Hit = 100
				player.Stats.Skl = 50
			}
			player.Items = []tactics.ItemStack{{Item: tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: 8, BlastRadius: 1, ThrowRange: 3}, Count: tc.count}}
			enemy.Pos = tactics.Pos{X: 3, Y: 1}
			enemy.Stats.HP = tc.hp
			enemy.Stats.Mov = 0
			enemy.Boss = false
			enemy.Commander = false
			enemy.ID = "enemy-a"
			second := enemy
			second.ID = "enemy-b"
			second.Pos = tactics.Pos{X: 4, Y: 1}
			second.Boss = true
			if tc.count == 0 {
				player.Items = nil
			}
			if tc.ally {
				player.Items[0].Item.ThrowRange = 2
			}
			setup.Units = []tactics.UnitSetup{player, enemy, second}
			if tc.ally {
				ally := player
				ally.ID = "ally"
				ally.Lord = false
				ally.Pos = tactics.Pos{X: 2, Y: 1}
				setup.Units = append(setup.Units, ally)
			}
			setup.Reinforcements = nil
			setup.Events = nil
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			p := &smartPlayer{autoPlayer: &autoPlayer{b: b, ids: make([]string, len(setup.Units)), step: map[int]int{}}, f: smartFeatures{Bomb: true, Threat: true, Focus: true}}
			p.refresh()
			o := p.bombOption(0)
			if (o != nil) != tc.want {
				t.Fatalf("爆弾候補 = %v、期待 %v", o, tc.want)
			}
			if o != nil {
				f, err := b.BlastForecast(o.cmd)
				if err != nil {
					t.Fatal(err)
				}
				kills := 0
				for _, v := range f.Targets {
					if v.Side == tactics.Enemy && v.Falls {
						kills++
					}
					if v.Side == tactics.Player {
						t.Fatal("味方を巻き込む")
					}
				}
				if kills != 2 {
					t.Fatalf("撃破 = %d、期待 2", kills)
				}
				if tc.strong {
					attack := p.attackOption(0)
					if attack == nil || attack.score <= o.score {
						t.Fatal("通常攻撃が爆弾を上回らない")
					}
					p.phase()
					if p.bombUses != 0 || b.Result != tactics.Won {
						t.Fatal("通常攻撃でボスを撃破しなかった")
					}
					return
				}
				if attack := p.attackOption(0); attack == nil {
					t.Fatal("通常攻撃候補がない")
				} else if attack.score >= o.score {
					t.Fatalf("爆弾が通常攻撃を上回らない: %v >= %v", attack.score, o.score)
				}
				p.phase()
				if p.bombUses != 1 || p.bombKills != 2 {
					t.Fatalf("使用・撃破 = %d・%d、期待 1・2", p.bombUses, p.bombKills)
				}
				for _, stack := range b.Units()[0].Items {
					if stack.Item.Bomb && stack.Count > 0 {
						t.Fatal("使用後に爆弾が残った")
					}
				}
				t.Run("TC-DES-NG-23-11 巻き戻し", func(t *testing.T) {
					if !p.hasFirst {
						t.Fatal("行動履歴がない")
					}
					if err := b.Rewind(p.firstID); err != nil {
						t.Fatal(err)
					}
					p.restoreTurn(&smartTurn{step: map[int]int{}})
					if p.bombUses != 0 || p.bombKills != 0 {
						t.Fatal("使用・撃破数が戻らない")
					}
					found := false
					for _, stack := range b.Units()[0].Items {
						if stack.Item.Bomb && stack.Count == 1 {
							found = true
						}
					}
					if !found {
						t.Fatal("在庫が戻らない")
					}
					p.refresh()
					if p.bombOption(0) == nil {
						t.Fatal("巻き戻し後の爆弾候補がない")
					}
				})
			}
		})
	}
}

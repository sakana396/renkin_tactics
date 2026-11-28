package core_test

import (
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"testing"
)

// TestBombOption は爆弾の選択と在庫を確かめる（DES-NG-23）。
func TestBombOption(t *testing.T) {
	// 盤面: 平地 8×3、投げる主人公 (0,1) は動かず、敵 a (3,1)・敵 b (4,1) も動かない。爆弾は威力 8・半径 1・射程 3。
	// b がボスでないときは、届かない (7,1) に HP99 の動かないボスを置く（ボスはちょうど 1 体。勝ちはボスの撃破）。
	// 爆心 (3,1) か (4,1) なら a・b の両方に 8 ダメージ。HP8 の敵は倒れ、HP20 の敵は倒れない（DES-NG01）。
	// 期待値は pickBomb の規則から: 2 体以上かボスを倒す爆弾だけを使い、攻撃でボスを 9 割以上で倒せるなら使わない。
	for _, tc := range []struct {
		id                  string
		hp, hpB, count      int
		ally, strong, bossB bool
		want                bool // bombOption が候補を返すか
		wantKills           int  // 候補の撃破数
		wantUses            int  // phase で使った数
		wantWon             bool // phase のあと勝ったか
	}{
		{"TC-DES-NG-23-05 敵2体を倒す", 8, 8, 1, false, false, false, true, 2, 1, false},
		{"TC-DES-NG-23-06 味方を巻き込む", 8, 8, 1, true, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-07 撃破なし", 20, 20, 1, false, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-08 在庫なし", 8, 8, 0, false, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-10 攻撃でボスを倒せる", 8, 8, 1, false, true, true, true, 2, 0, true},
		{"TC-DES-NG-23-12 1体だけ倒す", 8, 20, 1, false, false, false, true, 1, 0, false},
		{"TC-DES-NG-23-13 ボスを倒す", 20, 8, 1, false, false, true, true, 1, 1, true},
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
				setup.Terrain[y] = make([]tactics.Terrain, 8)
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
			second.Boss = tc.bossB
			second.Stats.HP = tc.hpB
			if tc.count == 0 {
				player.Items = nil
			}
			if tc.ally {
				player.Items[0].Item.ThrowRange = 2
			}
			setup.Units = []tactics.UnitSetup{player, enemy, second}
			if !tc.bossB {
				far := enemy
				far.ID = "enemy-boss"
				far.Pos = tactics.Pos{X: 7, Y: 1}
				far.Stats.HP = 99
				far.Boss = true
				setup.Units = append(setup.Units, far)
			}
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
				if kills != tc.wantKills {
					t.Fatalf("撃破 = %d、期待 %d", kills, tc.wantKills)
				}
				if o.decisive != (kills >= 2 || tc.bossB) {
					t.Fatalf("decisive = %v、撃破 %d・ボス %v", o.decisive, kills, tc.bossB)
				}
			}
			p.phase()
			if p.bombUses != tc.wantUses {
				t.Fatalf("使用 = %d、期待 %d", p.bombUses, tc.wantUses)
			}
			if p.bombUses > 0 && p.bombKills != tc.wantKills {
				t.Fatalf("撃破数 = %d、期待 %d", p.bombKills, tc.wantKills)
			}
			if (b.Result == tactics.Won) != tc.wantWon {
				t.Fatalf("結果 = %v、勝ちの期待 %v", b.Result, tc.wantWon)
			}
			left := 0
			for _, stack := range b.Units()[0].Items {
				if stack.Item.Bomb {
					left += stack.Count
				}
			}
			if left != tc.count-tc.wantUses {
				t.Fatalf("在庫 = %d、期待 %d", left, tc.count-tc.wantUses)
			}
			if tc.id != "TC-DES-NG-23-05 敵2体を倒す" {
				return
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
				if o := p.bombOption(0); o == nil || !o.decisive {
					t.Fatal("巻き戻し後の爆弾候補がない")
				}
			})
		})
	}
}

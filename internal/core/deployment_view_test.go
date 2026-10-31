package core_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// deploymentCatalog は変換と範囲の期待値を手計算できる目録（DES-DEP-02）。
func deploymentCatalog(rows []string, enemies []defs.EnemyPlacement) *defs.Catalog {
	return defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend"}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}, {ID: "forest", Name: "森", Symbol: "F", Cost: 2, Avoid: 20, Def: 1}, {ID: "wall", Name: "壁", Symbol: "#"}},
		[]defs.ClassDef{{ID: "soldier", Name: "兵", WeaponType: "sword", Base: defs.Stats{HP: 20, Str: 3, Skl: 4, Spd: 5, Def: 6, Mov: 3}}, {ID: "bowman", Name: "兵", WeaponType: "bow", Base: defs.Stats{HP: 20, Str: 3, Skl: 4, Spd: 5, Def: 6, Mov: 3}}},
		[]defs.WeaponDef{{ID: "sword", Name: "剣", Type: "sword", Might: 4, Hit: 1000, MinRange: 1, MaxRange: 1}, {ID: "bow", Name: "弓", Type: "bow", Might: 5, Hit: 80, MinRange: 2, MaxRange: 2}},
		[]defs.ItemDef{{ID: "potion", Name: "薬", Heal: 10}}, nil,
		[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "soldier", Weapons: []string{"sword"}, Level: 1, Lord: true}, {ID: "friend", Name: "仲間", Class: "soldier", Weapons: []string{"sword"}, Level: 2}},
		[]defs.MapDef{{ID: "m", Rows: rows, Slots: []defs.Slot{{X: 0, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 0}}, Enemies: enemies,
			Reinforcements: []defs.Reinforcement{{Turn: 3, EnemyPlacement: defs.EnemyPlacement{Class: "bowman", Weapons: []string{"bow"}, AI: "charge", X: 0, Y: 0}}}}}, []defs.ChapterDef{{ID: "c", Map: "m"}}, nil)
}

func TestDeploymentView(t *testing.T) {
	t.Run("TC-FR-CMP-11-08", func(t *testing.T) {
		enemies := []defs.EnemyPlacement{
			{Class: "bowman", Weapons: []string{"bow"}, AI: "hold", X: 4, Y: 1, Bonus: defs.Stats{HP: 2, Str: 1, Skl: 2, Spd: 3, Def: 4, Mov: 1}, Items: []defs.ItemCount{{Item: "potion", Count: 2}}},
			{Class: "soldier", Name: "隊長", Weapons: []string{"sword"}, AI: "charge", Boss: true, X: 4, Y: 0},
		}
		cat := deploymentCatalog([]string{"..F..", "..#..", "....."}, enemies)
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Stats.Str = 30
		c.Roster[0].Level, c.Roster[0].Exp = 4, 42
		c.Roster[0].Items = []defs.ItemCount{{Item: "potion", Count: 3}}
		before := campaignSnapshot(c)
		members := []int{0, 1}
		for _, slots := range [][]int{{0, 1}, {1, 0}, {2, 0}} {
			t.Run("枠", func(t *testing.T) {
				slotsBefore := append([]int(nil), slots...)
				v, err := c.DeploymentView(members, slots)
				if err != nil {
					t.Fatal(err)
				}
				again, err := c.DeploymentView(members, slots)
				if err != nil || !reflect.DeepEqual(v, again) {
					t.Fatalf("再取得: %v", err)
				}
				s, _, err := c.StartBattleAt(members, slots, 17)
				if err != nil {
					t.Fatal(err)
				}
				u := v.Units()
				if len(u) != 4 || !reflect.DeepEqual(u, s.Battle.Units()) {
					t.Fatalf("初期ユニット: %+v", u)
				}
				wantEnemy := tactics.Unit{Name: "兵 1", ClassName: "兵", Side: tactics.Enemy, Stats: tactics.Stats{HP: 22, Str: 4, Skl: 6, Spd: 8, Def: 10, Mov: 4}, HP: 22, Pos: tactics.Pos{X: 4, Y: 1}, Weapons: []tactics.Weapon{{ID: "bow", Name: "弓", Type: tactics.Bow, Might: 5, Hit: 80, MinRange: 2, MaxRange: 2}}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Bow}, Items: []tactics.ItemStack{{Item: tactics.Item{ID: "potion", Name: "薬", Heal: 10}, Count: 2}}, AI: tactics.Hold, Alive: true}
				wantBoss := tactics.Unit{Name: "隊長", ClassName: "兵", Side: tactics.Enemy, Stats: tactics.Stats{HP: 20, Str: 3, Skl: 4, Spd: 5, Def: 6, Mov: 3}, HP: 20, Pos: tactics.Pos{X: 4}, Weapons: []tactics.Weapon{{ID: "sword", Name: "剣", Type: tactics.Sword, Might: 4, Hit: 1000, MinRange: 1, MaxRange: 1}}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Sword}, Boss: true, AI: tactics.Charge, Alive: true}
				if !reflect.DeepEqual(u[2], wantEnemy) || !reflect.DeepEqual(u[3], wantBoss) {
					t.Fatalf("敵の完全な変換: %+v", u[2:])
				}
				positions := []tactics.Pos{{}, {Y: 1}, {X: 1}}
				if u[0].Stats != before.Roster[0].Stats || u[0].Level != 4 || u[0].Exp != 42 || u[0].Items[0].Count != 3 || u[0].Pos != positions[slots[0]] || u[1].Pos != positions[slots[1]] || !u[0].Lord {
					t.Fatal("現在名簿・枠が違う")
				}
				if v.Width() != 5 || v.Height() != 3 || v.Turn != 1 || v.Phase != tactics.Player || v.Result != tactics.Ongoing || v.NeedsResume() {
					t.Fatal("初期盤面の値が違う")
				}
				for y, row := range []string{"..F..", "..#..", "....."} {
					for x, ch := range row {
						want := tactics.Terrain{Name: "平地", Cost: 1}
						if ch == 'F' {
							want = tactics.Terrain{Name: "森", Cost: 2, Avoid: 20, Def: 1}
						}
						if ch == '#' {
							want = tactics.Terrain{Name: "壁"}
						}
						p := tactics.Pos{X: x, Y: y}
						if v.TerrainAt(p) != want || s.Battle.TerrainAt(p) != want {
							t.Fatalf("地形 %v", p)
						}
					}
				}
				danger, battleDanger := map[tactics.Pos]bool{}, map[tactics.Pos]bool{}
				for i := 2; i < 4; i++ {
					if !reflect.DeepEqual(v.Reachable(i), s.Battle.Reachable(i)) {
						t.Fatal("個別移動範囲が違う")
					}
					for p := range v.Reachable(i) {
						if !reflect.DeepEqual(v.AttackRange(i, p), s.Battle.AttackRange(i, p)) {
							t.Fatal("個別攻撃範囲が違う")
						}
						for _, q := range v.AttackRange(i, p) {
							danger[q] = true
						}
					}
					for p := range s.Battle.Reachable(i) {
						for _, q := range s.Battle.AttackRange(i, p) {
							battleDanger[q] = true
						}
					}
				}
				if !reflect.DeepEqual(danger, battleDanger) {
					t.Fatal("全敵範囲が違う")
				}
				u[0].Items[0].Count = 99
				u[2].Items[0].Count = 99
				u[0].Pos = tactics.Pos{X: 99}
				if !reflect.DeepEqual(v.Units(), s.Battle.Units()) || !reflect.DeepEqual(*c, before) || !reflect.DeepEqual(slots, slotsBefore) || !reflect.DeepEqual(members, []int{0, 1}) {
					t.Fatal("取得・返却値変更で状態が変わった")
				}
			})
		}
	})
	t.Run("TC-FR-CMP-11-09", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			rows []string
			slot int
			want map[tactics.Pos]int
		}{
			{"敵が通路をふさぐ", []string{"....", "####"}, 2, map[tactics.Pos]int{{X: 3}: 0}},
			{"森の費用", []string{".F..", "####"}, 0, map[tactics.Pos]int{{X: 3}: 0, {X: 1}: 3}},
			{"空き枠へ移して味方を通過", []string{"....", "####"}, 0, map[tactics.Pos]int{{X: 3}: 0, {X: 1}: 2}},
			{"壁で通路なし", []string{".#..", "####"}, 0, map[tactics.Pos]int{{X: 3}: 0}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cat := deploymentCatalog(tc.rows, []defs.EnemyPlacement{{Class: "bowman", Weapons: []string{"bow"}, AI: "hold", Boss: true, X: 3, Y: 0}, {Class: "soldier", Weapons: []string{"sword"}, AI: "hold", X: 2, Y: 0}})
				c, err := core.NewCampaign(cat, core.Withdraw)
				if err != nil {
					t.Fatal(err)
				}
				v, err := c.DeploymentView([]int{0}, []int{tc.slot})
				if err != nil {
					t.Fatal(err)
				}
				s, _, err := c.StartBattleAt([]int{0}, []int{tc.slot}, 1)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(v.Reachable(1), tc.want) || !reflect.DeepEqual(s.Battle.Reachable(1), tc.want) {
					t.Fatalf("移動範囲=%v 期待=%v", v.Reachable(1), tc.want)
				}
				wantAttack := []tactics.Pos{{X: 1}, {X: 2, Y: 1}}
				if !reflect.DeepEqual(v.AttackRange(1, tactics.Pos{X: 3}), wantAttack) || !reflect.DeepEqual(s.Battle.AttackRange(1, tactics.Pos{X: 3}), wantAttack) {
					t.Fatalf("弓の射程=%v", v.AttackRange(1, tactics.Pos{X: 3}))
				}
				for _, i := range []int{-1, 3} {
					if v.Reachable(i) != nil || v.AttackRange(i, tactics.Pos{}) != nil || s.Battle.Reachable(i) != nil || s.Battle.AttackRange(i, tactics.Pos{}) != nil {
						t.Fatal("不正ユニットの範囲")
					}
				}
			})
		}
		// 倒れた敵は停止・通過を妨げず、本人の範囲は空。実際の撃破後の盤面で確認する。
		cat := deploymentCatalog([]string{".......", "......."}, []defs.EnemyPlacement{{Class: "bowman", Weapons: []string{"bow"}, AI: "hold", Boss: true, X: 6}, {Class: "soldier", Weapons: []string{"sword"}, AI: "hold", X: 5}, {Class: "soldier", Weapons: []string{"sword"}, AI: "hold", X: 4}})
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Stats.Str = 100
		s, _, err := c.StartBattleAt([]int{0, 1}, []int{0, 2}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 3}, Action: tactics.Attack, Target: 4}); err != nil {
			t.Fatal(err)
		}
		b := s.Battle
		v := b.BoardView()
		if b.Units()[4].Alive {
			t.Fatal("敵が倒れていない")
		}
		if cost, ok := v.Reachable(2)[tactics.Pos{X: 4}]; !ok || cost != 2 {
			t.Fatal("倒れた敵が停止を妨げた")
		}
		wantDeadRange := map[tactics.Pos]int{{X: 6}: 0, {X: 6, Y: 1}: 1, {X: 4}: 2, {X: 5, Y: 1}: 2, {X: 4, Y: 1}: 3}
		if !reflect.DeepEqual(v.Reachable(2), wantDeadRange) || !reflect.DeepEqual(b.Reachable(2), wantDeadRange) {
			t.Fatalf("撃破後の移動範囲: %v", v.Reachable(2))
		}
		if !reflect.DeepEqual(v.Reachable(2), b.Reachable(2)) || v.Reachable(4) != nil || v.AttackRange(4, tactics.Pos{X: 4}) != nil || b.Reachable(4) != nil || b.AttackRange(4, tactics.Pos{X: 4}) != nil {
			t.Fatal("倒れたユニットの範囲")
		}
		// 2×1 盤面では射程 2 の攻撃先は存在しない。
		setup := tactics.Setup{Terrain: [][]tactics.Terrain{{{Cost: 1}, {Cost: 1}}}, Units: []tactics.UnitSetup{{Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 1}, Weapon: tactics.Weapon{MinRange: 2, MaxRange: 2}}, {Side: tactics.Enemy, Boss: true, Pos: tactics.Pos{X: 1}, Stats: tactics.Stats{HP: 1}, Weapon: tactics.Weapon{MinRange: 2, MaxRange: 2}}}}
		empty, err := tactics.NewBoardView(setup)
		if err != nil {
			t.Fatal(err)
		}
		battle, _, err := tactics.New(setup, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(empty.AttackRange(0, tactics.Pos{})) != 0 || len(battle.AttackRange(0, tactics.Pos{})) != 0 {
			t.Fatal("盤外の攻撃先を含む")
		}
	})
}

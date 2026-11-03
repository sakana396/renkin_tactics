package core_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 後半 2 章は出来事で倒された分を補う（DES-MAP-08 13 章）。
//
// #154（E-100）で初めの敵・ターンの増援・一団を変えたので、初めの敵とターンの増援の数・敵の騎兵の数・
// 出撃枠の安全は TC-FR-MAP-04-23 に移した。出来事の増援（数・一団・Lv・bonus）と閉路の検査は続ける。
func TestIT06LateChapters(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                          string
		slots, events, eventEnemies int
		bonus                       defs.Stats
		groups                      map[string]int // 出来事の増援の一団ごとの人数
		minLevel, maxLevel          int
	}{
		{"ch08", 6, 2, 7, defs.Stats{HP: 5, Str: 3, Skl: 3, Spd: 2, Def: 3}, map[string]int{"forest_road_reserve": 3, "forest_north_reserve": 4}, 8, 11},
		{"ch09", 7, 3, 12, defs.Stats{HP: 6, Str: 3, Skl: 3, Spd: 2, Def: 3}, map[string]int{"gate_west_reserve": 3, "gate_east_reserve": 3, "gate_inner_reserve": 6}, 9, 12},
	} {
		t.Run("TC-FR-MAP-04-20 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("地図がない")
			}
			var all []defs.EnemyPlacement
			total := 0
			reinforce := 0
			graph := map[string][]string{}
			for _, e := range m.Events {
				if e.Trigger == defs.TriggerEnemyAttacked && e.Effect == defs.EventChargeGroup {
					// #154 の enemy_attacked → charge_group は TC-FR-MAP-04-23 で確かめる。
					continue
				}
				reinforce++
				if e.Trigger != defs.TriggerEnemyDefeated || e.Effect != defs.EventReinforce {
					t.Fatalf("出来事 %s は撃破後の増援でない", e.ID)
				}
				total += len(e.Enemies)
				all = append(all, e.Enemies...)
				for _, enemy := range e.Enemies {
					if enemy.Bonus != tc.bonus || enemy.Level < tc.minLevel || enemy.Level > tc.maxLevel {
						t.Fatalf("増援 %s の能力・レベル: %+v", enemy.ID, enemy)
					}
					graph[e.Unit] = append(graph[e.Unit], enemy.ID)
				}
			}
			if len(m.Slots) != tc.slots || reinforce != tc.events || total != tc.eventEnemies {
				t.Fatalf("枠・増援の出来事・出来事の増援: %d/%d/%d", len(m.Slots), reinforce, total)
			}
			groups := map[string]int{}
			for _, e := range all {
				if e.Group != "" {
					if e.Approach != defs.ApproachGroup {
						t.Fatalf("%s の寄り方", e.ID)
					}
					groups[e.Group]++
				}
			}
			if !reflect.DeepEqual(groups, tc.groups) {
				t.Fatalf("出来事の増援の一団 %v", groups)
			}
			state := map[string]int{}
			var visit func(string) bool
			visit = func(id string) bool {
				if state[id] != 0 {
					return state[id] == 1
				}
				state[id] = 1
				for _, next := range graph[id] {
					if visit(next) {
						return true
					}
				}
				state[id] = 2
				return false
			}
			for id := range graph {
				if visit(id) {
					t.Fatalf("増援の参照に閉路: %s", id)
				}
			}
		})
	}
}

// 後半の想定名簿に育成・追加能力・前章までの武器を反映する（DES-MAP-08 13 章）。
func TestIT06LateRoster(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                           string
		low                          bool
		leon, noel, elsa, tio, hanna int
		leonLck, noelMag, noelRes    int
	}{
		{"ch08", false, 8, 8, 8, 8, 7, 9, 6, 5}, {"ch08", true, 7, 7, 7, 7, 7, 7, 4, 4},
		{"ch09", false, 9, 9, 9, 9, 8, 9, 6, 5}, {"ch09", true, 8, 8, 8, 8, 7, 8, 5, 4},
	} {
		t.Run("TC-FR-MAP-04-22 "+tc.id+"/低め="+map[bool]string{false: "いいえ", true: "はい"}[tc.low], func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			got := map[string]core.Member{}
			for _, m := range c.Roster {
				got[m.Character] = m
			}
			if got["leon"].Level != tc.leon || got["noel"].Level != tc.noel || got["elsa"].Level != tc.elsa || got["tio"].Level != tc.tio || got["hanna"].Level != tc.hanna {
				t.Fatalf("名簿のレベル: %+v", got)
			}
			if got["noel"].Stats.Mag != tc.noelMag || got["noel"].Stats.Res != tc.noelRes || got["leon"].Stats.Lck != tc.leonLck {
				t.Fatalf("追加能力の成長: ノエル %+v レオン %+v", got["noel"], got["leon"])
			}
			if !reflect.DeepEqual(got["leon"].Weapons, []string{"iron_sword", "armor_killer", "killing_sword", "thunder_sword"}) || !reflect.DeepEqual(got["mira"].Weapons, []string{"iron_bow", "long_bow"}) || !reflect.DeepEqual(got["bran"].Weapons, []string{"iron_axe", "hammer", "brave_axe"}) || !reflect.DeepEqual(got["elsa"].Weapons, []string{"iron_lance", "horse_killer"}) {
				t.Fatal("前章までの武器がない")
			}
			if !reflect.DeepEqual(got["leon"].Items, []defs.ItemCount{{Item: "potion", Count: 1}}) {
				t.Fatal("レオンの手持ちが 5 枠の配分になっていない")
			}
			if tc.id == "ch09" && !reflect.DeepEqual(got["gald"].Weapons, []string{"iron_lance", "horse_killer"}) {
				t.Fatal("2 本目の対騎兵の槍がない")
			}
		})
	}
}

// 公開の戦闘 API で後半の章を試し、勝ち筋と同時に生きる敵を確かめる（DES-MAP-08 13 章）。
func TestIT06LateChapterAutoplay(t *testing.T) {
	t.Skip("TC-FR-MAP-04-21 の固定 seed・進路の勝ち筋は #154（E-100）の配置で変わったため、3 種の名簿の TC-FR-MAP-04-24 に置き換えた")
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id, name, route string
		low, lost       bool
		seed            uint64
		events          []string
		appear          int
		members         []string
	}{
		{"ch08", "平均・中央", "center", false, false, 4, []string{"forest_road_alarm"}, 9, nil},
		{"ch08", "平均・東", "east", false, false, 2, []string{"forest_road_alarm", "forest_north_alarm"}, 13, nil},
		{"ch08", "低め・中央", "center", true, false, 13, []string{"forest_road_alarm"}, 9, []string{"leon", "gald", "bran", "mira", "elsa", "rina"}},
		{"ch08", "代役・低め・中央", "center", true, true, 7, []string{"forest_road_alarm"}, 9, nil},
		{"ch09", "平均・東", "east", false, false, 2, []string{"gate_east_alarm", "gate_inner_alarm"}, 17, nil},
		{"ch09", "平均・西", "west", false, false, 1, []string{"gate_west_alarm", "gate_east_alarm", "gate_inner_alarm"}, 20, nil},
		{"ch09", "平均・二手", "split", false, false, 5, []string{"gate_west_alarm", "gate_east_alarm", "gate_inner_alarm"}, 20, nil},
		{"ch09", "低め・東", "east", true, false, 11, []string{"gate_east_alarm", "gate_inner_alarm"}, 17, nil},
		{"ch09", "代役・低め・東", "east", true, true, 1, []string{"gate_east_alarm", "gate_inner_alarm"}, 17, nil},
	} {
		t.Run("TC-FR-MAP-04-21 "+tc.id+" "+tc.name, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			if tc.id == "ch09" && tc.low {
				// 低めの東進では出撃準備で雷の剣を共有在庫へ戻し、回復薬を 2 個にする。
				// 想定名簿には雷の剣を含めたうえで、5 枠内の持ち替えを試す（DES-MAP-08 13 章）。
				for i, m := range c.Roster {
					if m.Character == "leon" {
						if err := c.StoreWeapon(i, 3); err != nil {
							t.Fatal(err)
						}
						if err := c.TakeItem(i, "potion"); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			members := []string{"leon", "gald", "bran", "mira", "noel", "elsa"}
			if tc.id == "ch09" {
				members = append(members, "rina")
			}
			if tc.members != nil {
				members = tc.members
			}
			if tc.lost {
				for i, id := range members {
					if id == "mira" {
						members[i] = "tio"
					}
				}
			}
			s, ids, err := autoBattleWith(c, tc.seed, members)
			if err != nil {
				t.Fatal(err)
			}
			var plan autoPlan
			if tc.id == "ch08" {
				plan = ch07To08Plans("ch08")[0].plan
				if tc.route == "east" {
					plan = ch07To08Plans("ch08")[2].plan
				}
			} else {
				plan = ch09Plans()[tc.route]
			}
			m, _ := cat.Map(tc.id)
			maxLive := len(m.Enemies)
			triggers := map[string]bool{}
			reinforced := 0
			autoObserve = func(events []tactics.Event) {
				for _, e := range events {
					if e.Kind == tactics.EventTrigger {
						triggers[e.MapEvent] = true
					}
					if e.Kind == tactics.EventReinforce {
						reinforced++
					}
					live := 0
					for _, u := range e.Units {
						if u.Side == tactics.Enemy && u.Alive {
							live++
						}
					}
					maxLive = max(maxLive, live)
				}
			}
			r := autoPlay(s.Battle, ids, plan, 50)
			autoObserve = nil
			t.Logf("%v、同時の敵の最大=%d、出来事=%v、出現した増援=%d", r, maxLive, triggers, reinforced)
			if maxLive > 39 {
				t.Fatal("同時の敵が39体を超えた")
			}
			for _, id := range tc.events {
				if !triggers[id] {
					t.Fatalf("出来事 %s が起きない", id)
				}
			}
			if reinforced != tc.appear {
				t.Fatalf("増援 %d 体、期待 %d 体（上限で出なかった増援は 0）", reinforced, tc.appear)
			}
			if r.Result != tactics.Won {
				t.Fatal("勝つ手順がない")
			}
		})
	}
}

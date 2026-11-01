package core_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// ch01・ch03・ch04 の見直し（#131、DES-MAP-08、DES-IT06 13 章）の制作値を確かめる。
// 初めの敵とターンの増援は 13 章の表の「今の敵の数」のまま、足した敵はすべて出来事の増援（E-91・E-97）。
func TestIT06EarlyChapters(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                    string
		slots                 int
		initial, turn, events int
		// 出来事の増援の数。
		eventEnemies int
		// 出来事の魔道士の ID と、DES-IT06 5.3 節の式で求めた攻撃の速さ（各制作記録 2.4 節）。空なら魔道士はいない。
		mage      string
		mageSpeed int
		// 初めの敵・ターンの増援・出来事の増援を通した一団ごとの人数と、ボスを除く敵の騎兵の数。
		groups    map[string]int
		cavaliers int
		// 魔道士を除く出来事の増援の bonus（既存の一般の敵と同じ。E-97 で敵の能力は上げない）と、
		// レベルの範囲（DES-IT06 7.3 節の目安）。
		bonus      defs.Stats
		minL, maxL int
	}{
		{"ch01", 6, 8, 3, 1, 3, "", 0, map[string]int{}, 0, defs.Stats{}, 1, 3},
		{"ch03", 6, 10, 3, 3, 6, "keep_mage", 6, map[string]int{"wall_west": 3}, 5, defs.Stats{HP: 3, Str: 2, Skl: 1, Spd: 1, Def: 1}, 5, 7},
		{"ch04", 7, 12, 3, 3, 7, "village_mage", 6, map[string]int{"village_west": 3}, 5, defs.Stats{HP: 2, Str: 1, Skl: 1, Spd: 1, Def: 1}, 4, 6},
	} {
		t.Run("TC-FR-MAP-04-14 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("章の地図がない")
			}
			if len(m.Slots) != tc.slots || len(m.Enemies) != tc.initial || len(m.Reinforcements) != tc.turn || len(m.Events) != tc.events {
				t.Fatalf("枠・敵・増援・出来事の数: %d/%d/%d/%d", len(m.Slots), len(m.Enemies), len(m.Reinforcements), len(m.Events))
			}
			all := append([]defs.EnemyPlacement(nil), m.Enemies...)
			for _, r := range m.Reinforcements {
				all = append(all, r.EnemyPlacement)
			}
			eventEnemies := 0
			mage := 0
			cls, _ := cat.Class("mage")
			for _, event := range m.Events {
				if event.Effect != defs.EventReinforce {
					t.Fatalf("出来事 %s の効果が増援でない", event.ID)
				}
				eventEnemies += len(event.Enemies)
				all = append(all, event.Enemies...)
				for _, e := range event.Enemies {
					if e.Level < tc.minL || e.Level > tc.maxL {
						t.Fatalf("出来事の増援 %s の Lv%d が目安 %d〜%d の外", e.ID, e.Level, tc.minL, tc.maxL)
					}
					if e.Class != "mage" {
						if e.Bonus != tc.bonus {
							t.Fatalf("出来事の増援 %s の bonus %+v（期待 %+v）", e.ID, e.Bonus, tc.bonus)
						}
						continue
					}
					mage++
					if e.ID != tc.mage || len(e.Weapons) != 1 {
						t.Fatalf("魔道士の ID・武器: %s %v", e.ID, e.Weapons)
					}
					w, ok := cat.Weapon(e.Weapons[0])
					if !ok || w.Type != defs.WeaponMagic || w.DamageType != "magic" {
						t.Fatalf("魔道士の武器が魔法の攻撃でない: %+v", w)
					}
					str, spd := cls.Base.Str+e.Bonus.Str, cls.Base.Spd+e.Bonus.Spd
					if got := spd - max(0, w.Weight-str); got != tc.mageSpeed {
						t.Fatalf("魔道士の攻撃の速さ = %d、期待 %d", got, tc.mageSpeed)
					}
					// 魔力は 7.3 節の参考値（ch03 の Lv6 は 6.25、ch04 の Lv5 は 5.8）を四捨五入した 6。
					if mag := cls.Base.Mag + e.Bonus.Mag; mag != 6 {
						t.Fatalf("魔道士の魔力 = %d、期待 6", mag)
					}
				}
			}
			if eventEnemies != tc.eventEnemies || (tc.mage == "") != (mage == 0) || mage > 1 {
				t.Fatalf("出来事の増援 %d 体・魔道士 %d 体", eventEnemies, mage)
			}
			groups := map[string]int{}
			cavaliers := 0
			for _, e := range all {
				if e.Group != "" {
					if e.Approach != defs.ApproachGroup {
						t.Fatalf("一団 %s の %s が一団の寄り方でない", e.Group, e.ID)
					}
					groups[e.Group]++
				}
				if e.Class == "cavalier" && !e.Boss {
					cavaliers++
				}
			}
			if !reflect.DeepEqual(groups, tc.groups) || cavaliers != tc.cavaliers {
				t.Fatalf("一団 %v・敵の騎兵 %d（期待 %v・%d）", groups, cavaliers, tc.groups, tc.cavaliers)
			}
			// 敵を倒して初めて現れる増援の連鎖に閉路がない（TC-FR-MAP-04-11 と同じ見方）。
			graph := map[string][]string{}
			for _, event := range m.Events {
				if event.Trigger == defs.TriggerEnemyDefeated {
					for _, e := range event.Enemies {
						graph[event.Unit] = append(graph[event.Unit], e.ID)
					}
				}
			}
			state := map[string]int{}
			var cyclic func(string) bool
			cyclic = func(id string) bool {
				if state[id] != 0 {
					return state[id] == 1
				}
				state[id] = 1
				for _, next := range graph[id] {
					if cyclic(next) {
						return true
					}
				}
				state[id] = 2
				return false
			}
			for id := range graph {
				if cyclic(id) {
					t.Fatalf("倒さないと出ない敵の参照が循環: %s", id)
				}
			}
			// 出撃枠の安全: 足した敵は出来事の増援で 1 ターン目にはいないので、1 ターン目の初めの敵の攻撃はどの枠にも届かない。
			c, err := core.NewCampaignAt(cat, core.Withdraw, tc.id)
			if err != nil {
				t.Fatal(err)
			}
			s, _, err := c.StartBattle([]int{0}, 1)
			if err != nil {
				t.Fatal(err)
			}
			slots := map[tactics.Pos]bool{}
			for _, slot := range m.Slots {
				slots[tactics.Pos{X: slot.X, Y: slot.Y}] = true
			}
			for j, u := range s.Battle.Units() {
				if u.Side != tactics.Enemy {
					continue
				}
				for pos := range s.Battle.Reachable(j) {
					for _, target := range s.Battle.AttackRange(j, pos) {
						if slots[target] {
							t.Fatalf("1 ターン目の敵 %s %v の攻撃が枠 %v に届く", u.Name, u.Pos, target)
						}
					}
				}
			}
		})
	}
}

// ch01・ch03・ch04 の自動の試し遊び（#131）。育った想定名簿で、固定 seed の勝つ手順が残ることを確かめる（DES-MAP-08）。
// 勝ちは詰まない手順の下限で、難しさの判定には使わない。進路の出来事が起き、出来事の魔道士が魔法の武器と
// DES-IT06 5.3 節の攻撃の速さで出ることもあわせて見る。
func TestIT06EarlyChapterAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	routes := map[string][]tactics.Pos{
		// ch01 の中央 (5,10) → (9,6) → ボスの砦の手前 (12,2)。
		"ch01 中央": {{X: 5, Y: 10}, {X: 9, Y: 6}, {X: 12, Y: 2}},
		// ch03 の西の切れ目 (4,7) から前庭、内の門 (6,3) へ。
		"ch03 西": {{X: 4, Y: 10}, {X: 4, Y: 7}, {X: 5, Y: 5}, {X: 6, Y: 3}},
		// ch03 の東の切れ目 (11,7) から前庭、内の門 (11,3) へ。
		"ch03 東": {{X: 11, Y: 10}, {X: 11, Y: 7}, {X: 11, Y: 5}, {X: 11, Y: 3}},
		// ch04 の西の切れ目 (4,7) から村へ入り、ボスの前 (9,4) へ。
		"ch04 西": {{X: 4, Y: 12}, {X: 4, Y: 7}, {X: 5, Y: 4}, {X: 9, Y: 4}},
		// ch04 の東の野 (13,12) から東の切れ目 (13,7)、ボスの前 (9,4) へ。
		"ch04 東": {{X: 13, Y: 12}, {X: 13, Y: 7}, {X: 12, Y: 4}, {X: 9, Y: 4}},
	}
	// 順 A は ch02 版 0.2 の順、順 B は受け役を先に動かす順（ch02・ch05 の制作記録 5.1 節と同じ）。
	orderA := []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina", "tio"}
	orderB := []string{"gald", "bran", "mira", "tio", "noel", "elsa", "leon", "rina"}
	for _, tc := range []struct {
		id, name string
		low      bool
		bench    string // 出撃させない仲間（ch03 は 7 人から 6 人、ch04 は 8 人から 7 人を出す）
		route    string
		order    []string
		seed     uint64
		events   []string // 進路で起きる出来事
		mage     string
	}{
		{"ch01", "初期", false, "", "ch01 中央", orderA, 1, []string{"border_east_entry"}, ""},
		{"ch03", "平均", false, "rina", "ch03 西", orderA, 17, []string{"wall_west_falls", "courtyard_entry"}, "keep_mage"},
		{"ch03", "低め", true, "noel", "ch03 東", orderB, 7, []string{"wall_east_entry", "courtyard_entry"}, "keep_mage"},
		{"ch04", "平均", false, "tio", "ch04 西", orderB, 1, []string{"west_gate_falls", "north_entry"}, "village_mage"},
		{"ch04", "平均・東", false, "rina", "ch04 東", orderB, 16, []string{"east_field_entry", "north_entry"}, "village_mage"},
		{"ch04", "低め・ティオを出す", true, "rina", "ch04 西", orderA, 23, []string{"west_gate_falls", "north_entry"}, "village_mage"},
	} {
		t.Run("TC-FR-MAP-04-15 "+tc.id+" "+tc.name, func(t *testing.T) {
			var c *core.Campaign
			if tc.id == "ch01" {
				// ch01 は最初の章なので、名簿は定義の初期値のまま。
				var err error
				if c, err = core.NewCampaignAt(cat, core.Withdraw, tc.id); err != nil {
					t.Fatal(err)
				}
			} else {
				c = estimatedRosterIT06(t, cat, tc.id, tc.low)
			}
			var members []int
			for i, m := range c.Roster {
				if m.Character != tc.bench {
					members = append(members, i)
				}
			}
			s, _, err := c.StartBattle(members, tc.seed)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(members))
			for i, n := range members {
				ids[i] = c.Roster[n].Character
			}
			m, _ := cat.Map(tc.id)
			maxLive := len(m.Enemies)
			triggers := map[string]bool{}
			var mage *tactics.Unit
			autoObserve = func(events []tactics.Event) {
				for _, e := range events {
					if e.Kind == tactics.EventTrigger {
						triggers[e.MapEvent] = true
					}
					if e.Kind == tactics.EventReinforce && tc.mage != "" && e.Units[e.Unit].ID == tc.mage && mage == nil {
						u := e.Units[e.Unit]
						mage = &u
					}
					live := 0
					for _, u := range e.Units {
						if u.Alive && u.Side == tactics.Enemy {
							live++
						}
					}
					maxLive = max(maxLive, live)
				}
			}
			r := autoPlay(s.Battle, ids, autoPlan{Order: tc.order, Routes: [][]tactics.Pos{routes[tc.route]}}, 40)
			autoObserve = nil
			t.Logf("%v 同時の敵の最大=%d、起きた出来事=%v、上限で出なかった増援=0（総定義数が39以下）", r, maxLive, triggers)
			if maxLive > 39 {
				t.Fatal("同時の敵が39体を超えた")
			}
			for _, id := range tc.events {
				if !triggers[id] {
					t.Fatalf("想定の進路で出来事 %s が起きない", id)
				}
			}
			if tc.mage != "" {
				if mage == nil {
					t.Fatalf("出来事の魔道士 %s が出ていない", tc.mage)
				}
				w := mage.EquippedWeapon()
				if w.ID != "fire" || w.Type != tactics.Magic || w.DamageType != tactics.Magical {
					t.Fatalf("出来事の魔道士の武器が魔法の攻撃でない: %+v", w)
				}
				if got := mage.Stats.Spd - max(0, w.Weight-mage.Stats.Str); got != 6 {
					t.Fatalf("出来事の魔道士の攻撃の速さ = %d、期待 6（%+v）", got, mage.Stats)
				}
			}
			if r.Result != tactics.Won {
				t.Fatal("下限の手順で勝てない")
			}
		})
	}
}

// ch03・ch04 の想定名簿が、加入時期・三つの追加能力・前の章の報酬の武器を反映することを確かめる（#131、DES-MAP-08）。
func TestIT06EarlyRoster(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id, name                        string
		low                             bool
		leonLevel, rinaLevel, noelLevel int
		elsaLevel, tioLevel             int
		leonLck, noelMag                int
		elsaWeapons                     []string
	}{
		{"ch03", "平均", false, 4, 3, 3, 3, 0, 7, 3, []string{"iron_lance"}},
		{"ch03", "低め", true, 3, 2, 2, 2, 0, 6, 3, []string{"iron_lance"}},
		{"ch04", "平均", false, 4, 3, 4, 4, 4, 7, 4, []string{"iron_lance", "horse_killer"}},
		{"ch04", "低め", true, 3, 2, 3, 3, 4, 6, 3, []string{"iron_lance", "horse_killer"}},
	} {
		t.Run("TC-FR-MAP-04-16 "+tc.id+" "+tc.name, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			got := map[string]core.Member{}
			for _, m := range c.Roster {
				got[m.Character] = m
			}
			leon, rina, noel, elsa, tio := got["leon"], got["rina"], got["noel"], got["elsa"], got["tio"]
			if leon.Level != tc.leonLevel || rina.Level != tc.rinaLevel || noel.Level != tc.noelLevel || elsa.Level != tc.elsaLevel || tio.Level != tc.tioLevel {
				t.Fatalf("レベル: レオン %d リナ %d ノエル %d エルザ %d ティオ %d", leon.Level, rina.Level, noel.Level, elsa.Level, tio.Level)
			}
			if leon.Stats.Lck != tc.leonLck || noel.Stats.Mag != tc.noelMag {
				t.Fatalf("追加能力: レオン %+v / ノエル %+v", leon.Stats, noel.Stats)
			}
			if !reflect.DeepEqual(leon.Weapons, []string{"iron_sword", "armor_killer"}) || !reflect.DeepEqual(got["bran"].Weapons, []string{"iron_axe", "hammer"}) || !reflect.DeepEqual(elsa.Weapons, tc.elsaWeapons) {
				t.Fatalf("報酬の武器: レオン %v / ブラン %v / エルザ %v", leon.Weapons, got["bran"].Weapons, elsa.Weapons)
			}
		})
	}
}

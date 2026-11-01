package core_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 基準の章の制作値と出来事の依存を確かめる（DES-MAP-08、DES-MAP-07）。
func TestIT06BaseChapters(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id         string
		initial    int
		turn       int
		events     int
		group      string
		groupCount int
		// 出来事の魔道士の ID と、DES-IT06 5.3 節の式で求めた攻撃の速さ（制作記録の点検表）。
		mage      string
		mageSpeed int
		// 初めの敵・ターンの増援・出来事の増援を通した一団ごとの人数と、ボスを除く敵の騎兵の数（E-97 の版 0.3）。
		groups    map[string]int
		cavaliers int
	}{
		{"ch02", 8, 3, 4, "", 0, "road_mage", 6, map[string]int{"road_north": 3, "road_south": 2}, 6},
		{"ch05", 9, 2, 5, "gate_north", 3, "gate_rear_mage", 7, map[string]int{"gate_north": 3, "gate_east": 3, "gate_reserve": 2}, 4},
	} {
		t.Run("TC-FR-MAP-04-10 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("章の地図がない")
			}
			if len(m.Enemies) != tc.initial || len(m.Reinforcements) != tc.turn || len(m.Events) != tc.events {
				t.Fatalf("敵・増援・出来事の数: %d/%d/%d", len(m.Enemies), len(m.Reinforcements), len(m.Events))
			}
			mage := 0
			group := 0
			direct := 0
			for _, e := range m.Enemies {
				if e.Group == tc.group && tc.group != "" && e.Approach == defs.ApproachGroup {
					group++
				} else if e.Approach == "" || e.Approach == defs.ApproachDirect {
					direct++
				}
			}
			cls, _ := cat.Class("mage")
			for _, event := range m.Events {
				for _, e := range event.Enemies {
					if e.Class != "mage" {
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
					// 攻撃の速さ = (base ＋ bonus の速さ) − max(0, 重さ − (base ＋ bonus の力))（DES-IT06 5.3 節）。
					str, spd := cls.Base.Str+e.Bonus.Str, cls.Base.Spd+e.Bonus.Spd
					if got := spd - max(0, w.Weight-str); got != tc.mageSpeed {
						t.Fatalf("魔道士の攻撃の速さ = %d、期待 %d", got, tc.mageSpeed)
					}
				}
			}
			if mage != 1 || group != tc.groupCount || direct == 0 {
				t.Fatalf("魔道士または一団の設定が不足: mage=%d group=%d direct=%d", mage, group, direct)
			}
			// 一団と敵の騎兵（再移動あり）を、初めの敵・ターンの増援・出来事の増援を通して数える（E-97）。
			all := append([]defs.EnemyPlacement(nil), m.Enemies...)
			for _, r := range m.Reinforcements {
				all = append(all, r.EnemyPlacement)
			}
			for _, event := range m.Events {
				all = append(all, event.Enemies...)
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
			// 出撃枠の安全: 1 ターン目の初めの敵の攻撃は、どの枠にも届かない（DES-IT06 13 章、DES-IT05 の制作の検査）。
			// 版 0.3 で足した敵はすべて出来事の増援で、1 ターン目にはいない。ch05 の初めの敵（正面の守り・門のアーマーなど、
			// 待ちで穴の前を守る）は反復 5 から 1 ターン目に正面の枠へ届く配置（ch05 の制作記録 6.1 節）なので、ch02 だけを確かめる。
			if tc.id != "ch02" {
				return
			}
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
		t.Run("TC-FR-MAP-04-11 "+tc.id, func(t *testing.T) {
			m, _ := cat.Map(tc.id)
			// 敵を倒して初めて現れる増援の連鎖を有向グラフにし、閉路を拒む。
			graph := map[string][]string{}
			for _, event := range m.Events {
				if event.Trigger != defs.TriggerEnemyDefeated || event.Effect != defs.EventReinforce {
					continue
				}
				for _, e := range event.Enemies {
					graph[event.Unit] = append(graph[event.Unit], e.ID)
				}
			}
			state := map[string]int{}
			var visit func(string) bool
			visit = func(id string) bool {
				if state[id] == 1 {
					return true
				}
				if state[id] == 2 {
					return false
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
					t.Fatalf("倒さないと出ない敵の参照が循環: %s", id)
				}
			}
		})
	}
}

// 想定名簿は反復 6 の能力と加入時期を含めて作る（DES-MAP-08 13 章）。
// ch02 は ch01 から出た 5 人だけが平均 +2・低め +1 Lv、加入した 2 人は加入値のまま。
// ch05〜ch07 の初めの 5 人は反復 5 の見積もり（estimatedRoster）を使う。2 章に同じ Lv2 で加わる騎兵（エルザ）と
// 錬金術師（ノエル）は、出た章を 1 章 1 Lv としてそろえて育てる。
// 低めは伸びを成長率の 6 割で四捨五入する。幸運・魔力・魔防も同じ式で足す。
func estimatedRosterIT06(t *testing.T, cat *defs.Catalog, id string, low bool) *core.Campaign {
	t.Helper()
	base, err := core.NewCampaignAt(cat, core.Withdraw, id)
	if err != nil {
		t.Fatal(err)
	}
	c := base
	if id == "ch03" || id == "ch04" || id == "ch05" || id == "ch06" || id == "ch07" {
		c = estimatedRoster(t, cat, id, low)
	}
	rate := 1.0
	if low {
		rate = 0.6
	}
	for i := range c.Roster {
		m := &c.Roster[i]
		ch, _ := cat.Character(m.Character)
		joiner := m.Character == "noel" || m.Character == "elsa"
		// 物理の 5 能力を新しく足す数（ch05〜ch07 の初めの 5 人は estimatedRoster で足し済み）。
		grow := 0
		switch {
		case id == "ch02" && !joiner:
			grow = 2
		case id == "ch03" && joiner:
			*m = base.Roster[i]
			grow = 1
		case id == "ch04" && joiner:
			*m = base.Roster[i]
			grow = 2
		case id == "ch05" && joiner:
			*m = base.Roster[i]
			grow = 3
		case id == "ch06" && joiner:
			*m = base.Roster[i]
			grow = 4
		case id == "ch07" && joiner:
			*m = base.Roster[i]
			grow = 5
		}
		if low && grow > 0 {
			grow--
		}
		up := func(g, n int) int { return int(math.Round(float64(g*n) * rate / 100)) }
		m.Level += grow
		m.Stats.HP += up(ch.Growth.HP, grow)
		m.Stats.Str += up(ch.Growth.Str, grow)
		m.Stats.Skl += up(ch.Growth.Skl, grow)
		m.Stats.Spd += up(ch.Growth.Spd, grow)
		m.Stats.Def += up(ch.Growth.Def, grow)
		n := m.Level - ch.Level
		m.Stats.Lck += up(ch.Growth.Lck, n)
		m.Stats.Mag += up(ch.Growth.Mag, n)
		m.Stats.Res += up(ch.Growth.Res, n)
	}
	// 前の章の報酬の武器を持たせる（ch01〜ch06 のアーマーキラーから雷の剣まで）。
	takes := map[string][][2]string{
		"ch03": {{"leon", "armor_killer"}, {"bran", "hammer"}},
		"ch04": {{"leon", "armor_killer"}, {"bran", "hammer"}, {"elsa", "horse_killer"}},
		"ch05": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}},
		"ch06": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}, {"tio", "long_bow"}},
		"ch07": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"leon", "thunder_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}, {"tio", "long_bow"}},
	}
	if list, ok := takes[id]; ok {
		for _, take := range list {
			for i, m := range c.Roster {
				if m.Character == take[0] {
					if id == "ch07" && take[1] == "thunder_sword" {
						// 3 本目の報酬武器を持つため、主人公の回復薬を 1 個在庫へ戻す。
						if err := c.StoreItem(i, "potion"); err != nil {
							t.Fatal(err)
						}
					}
					if err := c.TakeWeapon(i, take[1]); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
	return c
}

// 自動の打ち手は詰まない手順の下限だけを確かめる（DES-MAP-08）。
// あわせて、出来事の増援として出た魔道士の tactics.Unit が、魔法の武器（ファイア）と
// DES-IT06 5.3 節の攻撃の速さを持つことを確かめる（core の enemyUnit の変換を通る）。
func TestIT06BaseChapterAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	routes := map[string][][]tactics.Pos{
		// ch02 の中央 (7,7) → 東 (12,8) → ボス (14,8)。
		"中央": {{{X: 7, Y: 7}, {X: 12, Y: 8}, {X: 14, Y: 8}}},
		// ch02 の手前 (4,7) で整えてから中央・東へ進み、ボス (14,8) へ。
		"整える": {{{X: 4, Y: 7}, {X: 7, Y: 7}, {X: 10, Y: 8}, {X: 12, Y: 8}, {X: 14, Y: 8}}},
		// ch05 の正面 (7,9) → 門 (7,6) → ボス (8,2)。
		"正面": {{{X: 7, Y: 9}, {X: 7, Y: 6}, {X: 8, Y: 2}}},
		// ch05 の森 x2 を北へ回り、ボス (8,2) へ。
		"森": {{{X: 2, Y: 10}, {X: 2, Y: 6}, {X: 4, Y: 3}, {X: 8, Y: 2}}},
	}
	// 行動順 A・C は版 0.2 の正面・森の順、B は受け役を先に動かす順（ch02・ch05 版 0.3 の測り直しで足した）。
	order := []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina", "tio"}
	forestOrder := []string{"mira", "gald", "leon", "bran", "noel", "elsa", "rina", "tio"}
	guardOrder := []string{"gald", "bran", "mira", "tio", "noel", "elsa", "leon", "rina"}
	for _, tc := range []struct {
		id, name  string
		low       bool
		bench     []string // 出撃させない仲間（ch05 は 8 人から 6 人を出す）
		route     string
		order     []string
		seed      uint64
		mage      string
		mageSpeed int
	}{
		{"ch02", "平均", false, nil, "中央", order, 3, "road_mage", 6},
		{"ch02", "低め", true, nil, "整える", guardOrder, 13, "road_mage", 6},
		{"ch05", "平均", false, []string{"rina", "tio"}, "正面", guardOrder, 1, "gate_rear_mage", 7},
		{"ch05", "低め", true, []string{"rina", "tio"}, "森", order, 151, "gate_rear_mage", 7},
		{"ch05", "ミラを失いティオを出す・平均", false, []string{"rina", "mira"}, "森", forestOrder, 5, "gate_rear_mage", 7},
		{"ch05", "ミラを失いティオを出す・低め", true, []string{"rina", "mira"}, "森", forestOrder, 119, "gate_rear_mage", 7},
	} {
		t.Run("TC-FR-MAP-04-12 "+tc.id+" "+tc.name, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			var members []int
			for i, m := range c.Roster {
				benched := false
				for _, b := range tc.bench {
					benched = benched || m.Character == b
				}
				if !benched {
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
			plan := autoPlan{Order: tc.order, Routes: routes[tc.route]}
			m, _ := cat.Map(tc.id)
			maxLive := len(m.Enemies)
			triggers := map[string]bool{}
			charges := 0
			var mage *tactics.Unit
			autoObserve = func(events []tactics.Event) {
				for _, e := range events {
					if e.Kind == tactics.EventTrigger {
						triggers[e.MapEvent] = true
					}
					if e.Kind == tactics.EventGroupCharge && e.Group == "gate_north" {
						charges++
					}
					if e.Kind == tactics.EventReinforce && e.Units[e.Unit].ID == tc.mage && mage == nil {
						u := e.Units[e.Unit]
						mage = &u
					}
					live := 0
					for _, u := range e.Units {
						if u.Alive && u.Side == tactics.Enemy {
							live++
						}
					}
					if live > maxLive {
						maxLive = live
					}
				}
			}
			r := autoPlay(s.Battle, ids, plan, 40)
			autoObserve = nil
			t.Logf("%v 同時の敵の最大=%d、起きた出来事=%v、上限で出なかった増援=0（総定義数が39以下）", r, maxLive, triggers)
			if maxLive > 39 {
				t.Fatal("同時の敵が39体を超えた")
			}
			for _, event := range m.Events {
				if !triggers[event.ID] {
					t.Fatalf("想定の進路で出来事 %s が起きない", event.ID)
				}
			}
			if tc.id == "ch05" && charges != 1 {
				t.Fatalf("一団の突撃の再生は 1 回のはず: %d", charges)
			}
			if mage == nil {
				t.Fatalf("出来事の魔道士 %s が出ていない", tc.mage)
			}
			w := mage.EquippedWeapon()
			if w.ID != "fire" || w.Type != tactics.Magic || w.DamageType != tactics.Magical {
				t.Fatalf("出来事の魔道士の武器が魔法の攻撃でない: %+v", w)
			}
			if got := mage.Stats.Spd - max(0, w.Weight-mage.Stats.Str); got != tc.mageSpeed {
				t.Fatalf("出来事の魔道士の攻撃の速さ = %d、期待 %d（%+v）", got, tc.mageSpeed, mage.Stats)
			}
			if r.Result != tactics.Won {
				t.Fatal("下限の手順で勝てない")
			}
		})
	}
}

// 反復 6 の想定名簿が、加入時期と三つの追加能力・配布済みの武器を反映することを確かめる（DES-MAP-08）。
func TestIT06EstimatedRoster(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id, name                        string
		low                             bool
		leonLevel, noelLevel, elsaLevel int
		leonLck, noelMag                int
	}{
		{"ch02", "平均", false, 3, 2, 2, 7, 3},
		{"ch02", "低め", true, 2, 2, 2, 6, 3},
		// 2 章に同じ Lv2 で加わる騎兵と錬金術師は、ch05 で同じレベルになる（REV-079 F-05）。
		{"ch05", "平均", false, 5, 5, 5, 8, 4},
		{"ch05", "低め", true, 4, 4, 4, 7, 4},
	} {
		t.Run("TC-FR-MAP-04-13 "+tc.id+" "+tc.name, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			var leon, noel, elsa core.Member
			for _, m := range c.Roster {
				switch m.Character {
				case "leon":
					leon = m
				case "noel":
					noel = m
				case "elsa":
					elsa = m
				}
			}
			if leon.Level != tc.leonLevel || leon.Stats.Lck != tc.leonLck || noel.Level != tc.noelLevel || noel.Stats.Mag != tc.noelMag || elsa.Level != tc.elsaLevel {
				t.Fatalf("成長の見積もり: レオン %+v / ノエル %+v / エルザ %+v", leon, noel, elsa)
			}
			if tc.id == "ch05" && (len(leon.Weapons) != 3 || leon.Weapons[1] != "armor_killer" || leon.Weapons[2] != "killing_sword") {
				t.Fatalf("報酬の武器: %v", leon.Weapons)
			}
		})
	}
}

package core_test

import (
	"math"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 基準の章の制作値と出来事の依存を確かめる（DES-MAP-08、DES-MAP-07）。
//
// #154（E-100）で初めの敵・ターンの増援・一団を変えたので、配置に関係する検査（敵・増援・出来事の数、一団、
// 敵の騎兵、出撃枠の安全）は TC-FR-MAP-04-23 に移した。出来事の魔道士の武器と攻撃の速さ、TC-FR-MAP-04-11 は続ける。
func TestIT06BaseChapters(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id string
		// 出来事の魔道士の ID と、DES-IT06 5.3 節の式で求めた攻撃の速さ（制作記録の点検表）。
		mage      string
		mageSpeed int
	}{
		{"ch02", "road_mage", 6},
		{"ch05", "gate_rear_mage", 7},
	} {
		t.Run("TC-FR-MAP-04-10 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("章の地図がない")
			}
			mage := 0
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
			if mage != 1 {
				t.Fatalf("出来事の魔道士が 1 体でない: %d", mage)
			}
			// TC-FR-MAP-04-12 で出現を待って見ていた tactics の形も、変換を通して確かめる。
			checkEventMagesSetup(t, cat, tc.id, tc.mageSpeed)
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
// ch05〜ch09 の初めの 5 人は反復 5 の見積もり（estimatedRoster）を使う。2 章に同じ Lv2 で加わる騎兵（エルザ）と
// 錬金術師（ノエル）は、出た章から 1 章 1 Lv で育て、二人の差をなくす。
// 低めは伸びを成長率の 6 割で四捨五入する。幸運・魔力・魔防も同じ式で足す。
func estimatedRosterIT06(t *testing.T, cat *defs.Catalog, id string, low bool) *core.Campaign {
	t.Helper()
	base, err := core.NewCampaignAt(cat, core.Withdraw, id)
	if err != nil {
		t.Fatal(err)
	}
	c := base
	if id == "ch03" || id == "ch04" || id == "ch05" || id == "ch06" || id == "ch07" || id == "ch08" || id == "ch09" {
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
		case id == "ch08" && joiner:
			*m = base.Roster[i]
			grow = 6
		case id == "ch09" && joiner:
			*m = base.Roster[i]
			grow = 7
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
	// 前の章の報酬の武器を持たせる（DES-MAP-08 13 章）。
	takes := map[string][][2]string{
		"ch03": {{"leon", "armor_killer"}, {"bran", "hammer"}},
		"ch04": {{"leon", "armor_killer"}, {"bran", "hammer"}, {"elsa", "horse_killer"}},
		"ch05": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}},
		"ch06": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}, {"tio", "long_bow"}},
		"ch07": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"leon", "thunder_sword"}, {"bran", "hammer"}, {"elsa", "horse_killer"}, {"tio", "long_bow"}},
		"ch08": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"leon", "thunder_sword"}, {"mira", "long_bow"}, {"bran", "hammer"}, {"bran", "brave_axe"}, {"elsa", "horse_killer"}},
		"ch09": {{"leon", "armor_killer"}, {"leon", "killing_sword"}, {"leon", "thunder_sword"}, {"mira", "long_bow"}, {"bran", "hammer"}, {"bran", "brave_axe"}, {"elsa", "horse_killer"}, {"gald", "horse_killer"}},
	}
	if list, ok := takes[id]; ok {
		// 雷の剣を含める章では、レオンの 5 枠を守るため回復薬 1 個を共有在庫に戻す。
		if id == "ch08" || id == "ch09" {
			for i, m := range c.Roster {
				if m.Character == "leon" {
					if err := c.StoreItem(i, "potion"); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
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
	t.Skip("TC-FR-MAP-04-12 の固定 seed・進路の勝ち筋は #154（E-100）の配置で変わったため、3 種の名簿の TC-FR-MAP-04-24 に置き換えた。出来事の魔道士の武器と攻撃の速さは TC-FR-MAP-04-10 で確かめる")
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

// checkEventMagesSetup は、出来事の増援の魔道士を core の変換（enemyUnit）を通した tactics の形で確かめる。
// 魔法の武器（ファイア）を装備し、DES-IT06 5.3 節の攻撃の速さが want であること（DES-CMP-11、DES-MAP-07）。
// 自動の試し遊びで出現を待たずに確かめるので、配置や勝ち筋に左右されない（#154 の F-04）。
func checkEventMagesSetup(t *testing.T, cat *defs.Catalog, mapID string, want int) {
	t.Helper()
	m, _ := cat.Map(mapID)
	mages := map[string]bool{}
	for _, event := range m.Events {
		for _, e := range event.Enemies {
			if e.Class == "mage" {
				mages[e.ID] = true
			}
		}
	}
	if len(mages) == 0 {
		t.Fatalf("%s に出来事の魔道士がいない", mapID)
	}
	setup, err := core.NewSetup(cat, mapID)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, event := range setup.Events {
		for _, u := range event.Enemies {
			if !mages[u.ID] {
				continue
			}
			seen++
			w := u.Weapons[u.Equipped]
			if w.ID != "fire" || w.Type != tactics.Magic || w.DamageType != tactics.Magical {
				t.Fatalf("出来事の魔道士 %s の武器が魔法の攻撃でない: %+v", u.ID, w)
			}
			if got := u.Stats.Spd - max(0, w.Weight-u.Stats.Str); got != want {
				t.Fatalf("出来事の魔道士 %s の攻撃の速さ = %d、期待 %d（%+v）", u.ID, got, want, u.Stats)
			}
		}
	}
	if seen != len(mages) {
		t.Fatalf("変換された出来事の魔道士は %d 体、定義 %d 体", seen, len(mages))
	}
}

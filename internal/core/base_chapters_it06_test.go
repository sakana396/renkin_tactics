package core_test

import (
	"math"
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
	}{
		{"ch02", 8, 3, 2, "", 0, "road_mage", 6},
		{"ch05", 9, 2, 3, "gate_north", 3, "gate_rear_mage", 7},
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
// ch05 の初めの 5 人は反復 5 の見積もり（estimatedRoster）を使う。2 章に同じ Lv2 で加わる騎兵（エルザ）と
// 錬金術師（ノエル）は、出た章（ch02〜ch04）の 1 章 1 Lv で平均 +3・低め +2 Lv とし、二人の差をなくす。
// 低めは伸びを成長率の 6 割で四捨五入する。幸運・魔力・魔防も同じ式で足す。
func estimatedRosterIT06(t *testing.T, cat *defs.Catalog, id string, low bool) *core.Campaign {
	t.Helper()
	base, err := core.NewCampaignAt(cat, core.Withdraw, id)
	if err != nil {
		t.Fatal(err)
	}
	c := base
	if id == "ch05" {
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
		// 物理の 5 能力を新しく足す数（ch05 の初めの 5 人は estimatedRoster で足し済み）。
		grow := 0
		switch {
		case id == "ch02" && !joiner:
			grow = 2
		case id == "ch05" && joiner:
			*m = base.Roster[i]
			grow = 3
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
	if id == "ch05" {
		for _, take := range []struct {
			character string
			weapon    string
		}{
			{"leon", "armor_killer"}, {"leon", "killing_sword"},
			{"bran", "hammer"}, {"elsa", "horse_killer"},
		} {
			for i, m := range c.Roster {
				if m.Character == take.character {
					if err := c.TakeWeapon(i, take.weapon); err != nil {
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
	order := []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina", "tio"}
	routes := map[string][][]tactics.Pos{
		// ch02 の中央 (7,7) → 東 (12,8) → ボス (14,8)。
		"中央": {{{X: 7, Y: 7}, {X: 12, Y: 8}, {X: 14, Y: 8}}},
		// ch05 の正面 (7,9) → 門 (7,6) → ボス (8,2)。
		"正面": {{{X: 7, Y: 9}, {X: 7, Y: 6}, {X: 8, Y: 2}}},
		// ch05 の森 x2 を北へ回り、ボス (8,2) へ。
		"森": {{{X: 2, Y: 10}, {X: 2, Y: 6}, {X: 4, Y: 3}, {X: 8, Y: 2}}},
	}
	forestOrder := []string{"mira", "gald", "leon", "bran", "noel", "elsa", "rina", "tio"}
	for _, tc := range []struct {
		id, name  string
		low       bool
		bench     []string // 出撃させない仲間（ch05 は 8 人から 6 人を出す）
		route     string
		seed      uint64
		mage      string
		mageSpeed int
	}{
		{"ch02", "平均", false, nil, "中央", 1, "road_mage", 6},
		{"ch02", "低め", true, nil, "中央", 10, "road_mage", 6},
		{"ch05", "平均", false, []string{"rina", "tio"}, "正面", 10, "gate_rear_mage", 7},
		{"ch05", "低め", true, []string{"rina", "tio"}, "森", 19, "gate_rear_mage", 7},
		{"ch05", "ミラを失いティオを出す・平均", false, []string{"rina", "mira"}, "正面", 1, "gate_rear_mage", 7},
		{"ch05", "ミラを失いティオを出す・低め", true, []string{"rina", "mira"}, "森", 34, "gate_rear_mage", 7},
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
			plan := autoPlan{Order: order, Routes: routes[tc.route]}
			if tc.route == "森" {
				plan.Order = forestOrder
			}
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

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
	}{
		{"ch02", 8, 3, 2, "", 0},
		{"ch05", 9, 2, 3, "gate_north", 3},
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
			for _, event := range m.Events {
				for _, e := range event.Enemies {
					if e.Class == "mage" && len(e.Weapons) == 1 && e.Weapons[0] == "fire" && e.Bonus.Spd == -1 {
						mage++
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
func estimatedRosterIT06(t *testing.T, cat *defs.Catalog, id string, low bool) *core.Campaign {
	t.Helper()
	var c *core.Campaign
	if id == "ch05" {
		c = estimatedRoster(t, cat, id, low)
		base, err := core.NewCampaignAt(cat, core.Withdraw, id)
		if err != nil {
			t.Fatal(err)
		}
		for i := range c.Roster {
			if c.Roster[i].Character == "noel" {
				c.Roster[i] = base.Roster[i]
			}
		}
	} else {
		var err error
		c, err = core.NewCampaignAt(cat, core.Withdraw, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range c.Roster {
		m := &c.Roster[i]
		ch, _ := cat.Character(m.Character)
		n := m.Level - ch.Level
		if id == "ch02" && m.Character != "noel" && m.Character != "elsa" {
			n = 2
			if low {
				n = 1
			}
			m.Level += n
			up := func(g int) int {
				rate := 1.0
				if low {
					rate = 0.6
				}
				return int(math.Round(float64(g*n) * rate / 100))
			}
			m.Stats.HP += up(ch.Growth.HP)
			m.Stats.Str += up(ch.Growth.Str)
			m.Stats.Skl += up(ch.Growth.Skl)
			m.Stats.Spd += up(ch.Growth.Spd)
			m.Stats.Def += up(ch.Growth.Def)
		}
		if id == "ch05" && m.Character == "noel" {
			n = 2
			if low {
				n = 1
			}
			m.Level += n
			up := func(g int) int {
				rate := 1.0
				if low {
					rate = 0.6
				}
				return int(math.Round(float64(g*n) * rate / 100))
			}
			m.Stats.HP += up(ch.Growth.HP)
			m.Stats.Str += up(ch.Growth.Str)
			m.Stats.Skl += up(ch.Growth.Skl)
			m.Stats.Spd += up(ch.Growth.Spd)
			m.Stats.Def += up(ch.Growth.Def)
		}
		rate := 1.0
		if low {
			rate = 0.6
		}
		up := func(g int) int { return int(math.Round(float64(g*n) * rate / 100)) }
		m.Stats.Lck += up(ch.Growth.Lck)
		m.Stats.Mag += up(ch.Growth.Mag)
		m.Stats.Res += up(ch.Growth.Res)
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
func TestIT06BaseChapterAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id   string
		low  bool
		seed uint64
	}{
		{"ch02", false, 3}, {"ch02", true, 2},
		{"ch05", false, 1}, {"ch05", true, 4},
	} {
		t.Run("TC-FR-MAP-04-12 "+tc.id, func(t *testing.T) {
			var won bool
			for seed := tc.seed; seed <= tc.seed; seed++ {
				c := estimatedRosterIT06(t, cat, tc.id, tc.low)
				var members []int
				for i, m := range c.Roster {
					if tc.id == "ch05" && (m.Character == "rina" || m.Character == "tio") {
						continue
					}
					members = append(members, i)
				}
				s, _, err := c.StartBattle(members, seed)
				if err != nil {
					t.Fatal(err)
				}
				ids := make([]string, len(s.Battle.Units()))
				for i, n := range members {
					ids[i] = c.Roster[n].Character
				}
				plan := autoPlan{Order: []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina"}}
				if tc.id == "ch02" {
					plan.Routes = [][]tactics.Pos{{{X: 7, Y: 7}, {X: 12, Y: 8}, {X: 14, Y: 8}}}
				} else {
					plan.Routes = [][]tactics.Pos{{{X: 7, Y: 9}, {X: 7, Y: 6}, {X: 8, Y: 2}}}
				}
				m, _ := cat.Map(tc.id)
				maxLive := len(m.Enemies)
				triggers := map[string]bool{}
				charges := 0
				autoObserve = func(events []tactics.Event) {
					for _, e := range events {
						if e.Kind == tactics.EventTrigger {
							triggers[e.MapEvent] = true
						}
						if e.Kind == tactics.EventGroupCharge && e.Group == "gate_north" {
							charges++
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
				t.Logf("同時の敵の最大=%d、起きた出来事=%v、上限で出なかった増援=0（総定義数が39以下）", maxLive, triggers)
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
				if r.Result == tactics.Won {
					t.Logf("%s 低め=%v seed=%d: %v", tc.id, tc.low, seed, r)
					won = true
					break
				}
			}
			if !won {
				t.Fatal("下限の手順で勝てない")
			}
		})
	}
}

// 反復 6 の想定名簿が、加入時期と三つの追加能力・配布済みの武器を反映することを確かめる（DES-MAP-08）。
func TestIT06EstimatedRoster(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                   string
		low                  bool
		leonLevel, noelLevel int
		leonLck, noelMag     int
	}{
		{"ch02", false, 3, 2, 7, 3},
		{"ch02", true, 2, 2, 6, 3},
		{"ch05", false, 5, 4, 8, 4},
		{"ch05", true, 4, 3, 7, 3},
	} {
		t.Run("TC-FR-MAP-04-13 "+tc.id, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			var leon, noel core.Member
			for _, m := range c.Roster {
				if m.Character == "leon" {
					leon = m
				}
				if m.Character == "noel" {
					noel = m
				}
			}
			if leon.Level != tc.leonLevel || leon.Stats.Lck != tc.leonLck || noel.Level != tc.noelLevel || noel.Stats.Mag != tc.noelMag {
				t.Fatalf("成長の見積もり: レオン %+v / ノエル %+v", leon, noel)
			}
			if tc.id == "ch05" && (len(leon.Weapons) != 3 || leon.Weapons[1] != "armor_killer" || leon.Weapons[2] != "killing_sword") {
				t.Fatalf("報酬の武器: %v", leon.Weapons)
			}
		})
	}
}

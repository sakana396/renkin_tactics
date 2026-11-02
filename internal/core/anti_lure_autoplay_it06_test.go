package core_test

import (
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// antiLureRoster は TC-FR-MAP-04-24 で試す名簿の種類（DES-IT06 13 章「平均の成長・低い成長・ミラを失ってティオを出す」）。
type antiLureRoster struct {
	name string
	low  bool
	// lostMira はミラを失った名簿。出撃の並びのミラをティオ（すでに出ていればリナ）に替える。
	lostMira bool
}

var antiLureRosters = []antiLureRoster{
	{"平均", false, false},
	{"低め", true, false},
	{"ミラを失いティオを出す", false, true},
}

// antiLureRoutes・antiLureChosen・antiLureOrder は TC-FR-MAP-04-24 の主な進路・出撃・行動の順（#158 の賢い打ち手でも同じ名簿を使う）。
var antiLureRoutes = map[string][]tactics.Pos{
	"ch01": {{X: 5, Y: 10}, {X: 9, Y: 6}, {X: 12, Y: 2}},
	"ch02": {{X: 7, Y: 7}, {X: 12, Y: 8}, {X: 14, Y: 8}},
	"ch03": {{X: 11, Y: 10}, {X: 11, Y: 7}, {X: 11, Y: 5}, {X: 11, Y: 3}},
	"ch04": {{X: 4, Y: 12}, {X: 4, Y: 7}, {X: 5, Y: 4}, {X: 9, Y: 4}},
	"ch05": {{X: 7, Y: 9}, {X: 7, Y: 6}, {X: 8, Y: 2}},
	"ch06": {{X: 6, Y: 14}, {X: 6, Y: 10}, {X: 7, Y: 6}, {X: 10, Y: 3}},
	"ch07": {{X: 7, Y: 11}, {X: 7, Y: 8}, {X: 7, Y: 4}, {X: 9, Y: 2}},
	"ch08": {{X: 11, Y: 17}, {X: 11, Y: 11}, {X: 11, Y: 6}, {X: 12, Y: 2}},
	"ch09": ch09East,
}
var antiLureChosen = map[string][]string{
	"ch01": {"leon", "gald", "bran", "mira", "rina"},
	"ch02": {"leon", "gald", "bran", "mira", "rina", "noel", "elsa"},
	"ch03": {"leon", "gald", "bran", "mira", "noel", "elsa"},
	"ch04": {"leon", "gald", "bran", "mira", "noel", "elsa", "rina"},
	"ch05": {"leon", "gald", "bran", "mira", "noel", "elsa"},
	"ch06": {"leon", "gald", "bran", "mira", "noel", "elsa", "tio"},
	"ch07": {"leon", "gald", "bran", "mira", "noel", "tio"},
	"ch08": {"leon", "gald", "bran", "mira", "noel", "elsa"},
	"ch09": {"leon", "gald", "bran", "mira", "noel", "elsa", "rina"},
}
var antiLureOrder = []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina", "tio"}

// 想定名簿と既存の主な進路で、固定 20 seed の勝ち数を測る（DES-MAP-08、DES-IT06 13 章）。
// 勝ちは詰まない手順の下限で、難しさの判定には使わない。名簿ごとに並べて走らせる。
func TestIT06AntiLureAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	routes, chosenByMap, order := antiLureRoutes, antiLureChosen, antiLureOrder
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		for _, roster := range antiLureRosters {
			// ch01 は名簿が定義の初期値だけ。ティオは ch04 で加わる（chapters.toml）。
			if ch == 1 && roster.name != "平均" {
				continue
			}
			if ch < 4 && roster.lostMira {
				continue
			}
			name := roster.name
			if ch == 1 {
				name = "初期"
			}
			t.Run("TC-FR-MAP-04-24 "+id+" "+name, func(t *testing.T) {
				t.Parallel()
				m, _ := cat.Map(id)
				scheduled := map[string]int{}
				for _, reinforcement := range m.Reinforcements {
					if reinforcement.ID != "" {
						scheduled[reinforcement.ID] = reinforcement.Turn
					}
				}
				chosen := append([]string(nil), chosenByMap[id]...)
				if roster.lostMira {
					sub := "tio"
					if containsWeapon(chosen, "tio") {
						sub = "rina"
					}
					for i := range chosen {
						if chosen[i] == "mira" {
							chosen[i] = sub
						}
					}
				}
				seenScheduled := map[int]int{}
				seenIDs := map[string]bool{}
				var winSeeds []int
				firstSpawn, firstMax := 0, 0
				firstTurns := map[int]int{}
				for seed := uint64(1); seed <= 20; seed++ {
					var c *core.Campaign
					if ch == 1 {
						var err error
						c, err = core.NewCampaignAt(cat, core.Withdraw, id)
						if err != nil {
							t.Fatal(err)
						}
					} else {
						c = estimatedRosterIT06(t, cat, id, roster.low)
					}
					if roster.lostMira {
						for i := range c.Roster {
							if c.Roster[i].Character == "mira" {
								c.Roster[i].Alive = false
							}
						}
					}
					var members []int
					for _, want := range chosen {
						for i, member := range c.Roster {
							if member.Character == want {
								members = append(members, i)
								break
							}
						}
					}
					if len(members) != len(chosen) {
						t.Fatalf("名簿に %v がそろわない", chosen)
					}
					s, _, err := c.StartBattle(members, seed)
					if err != nil {
						t.Fatal(err)
					}
					ids := make([]string, len(members))
					for i, n := range members {
						ids[i] = c.Roster[n].Character
					}
					turn := 0
					spawned := 0
					spawnTurns := map[int]int{}
					maxLive := 0
					observe := func(events []tactics.Event) {
						for _, event := range events {
							if event.Kind == tactics.EventPhaseStart && event.Side == tactics.Enemy {
								turn = event.Turn
							}
							if event.Kind == tactics.EventReinforce {
								spawned++
								spawnTurns[turn]++
								if scheduledTurn, ok := scheduled[event.Units[event.Unit].ID]; ok {
									if turn != scheduledTurn {
										t.Errorf("%s は %d ターン目に出現（定義 %d）", event.Units[event.Unit].ID, turn, scheduledTurn)
									}
									seenScheduled[turn]++
									seenIDs[event.Units[event.Unit].ID] = true
								}
							}
							live := 0
							for _, unit := range event.Units {
								if unit.Side == tactics.Enemy && unit.Alive {
									live++
								}
							}
							maxLive = max(maxLive, live)
						}
					}
					r := autoPlay(s.Battle, ids, autoPlan{Order: order, Routes: [][]tactics.Pos{routes[id]}, Observe: observe}, 50)
					if maxLive > 39 {
						t.Fatalf("生存敵が %d 体", maxLive)
					}
					if r.Result == tactics.Won {
						if len(winSeeds) == 0 {
							firstSpawn, firstMax, firstTurns = spawned, maxLive, spawnTurns
						}
						winSeeds = append(winSeeds, int(seed))
					}
				}
				t.Logf("%s %s: 出撃 %v、勝つ seed %v、20 seed 中 %d 勝。最初の勝ちで出現した増援=%d、ターン別=%v、同時の敵の最大=%d", id, name, chosen, winSeeds, len(winSeeds), firstSpawn, firstTurns, firstMax)
				// 低めの名簿は参考として数えるだけにする（主な進路・固定の行動順・20 seed では 2〜9 章でほぼ勝てない。
				// 13 章の「明らかな詰みがないか」の下限は、平均とミラを失ってティオを出す名簿で確かめる）。
				if len(winSeeds) == 0 && !roster.low {
					t.Fatal("想定名簿で勝つ固定 seed がない")
				}
				if ch >= 2 && roster.name == "平均" {
					end := 8
					if ch >= 8 {
						end = 12
					}
					for turn := 4; turn <= end; turn++ {
						if seenScheduled[turn] == 0 {
							t.Errorf("%d ターン目の増援が 20 seed の試遊で一度も現れない", turn)
						}
					}
					for reinforcementID := range scheduled {
						if !seenIDs[reinforcementID] {
							t.Errorf("ターン増援 %s が 20 seed の試遊で一度も現れない", reinforcementID)
						}
					}
				}
			})
		}
	}
}

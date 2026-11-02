package core_test

import (
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 想定名簿と既存の主な進路で、固定 20 seed の勝ち数を測る（DES-MAP-08）。
func TestIT06AntiLureAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	routes := map[string][]tactics.Pos{
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
	order := []string{"gald", "mira", "noel", "bran", "elsa", "leon", "rina", "tio"}
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		t.Run("TC-FR-MAP-04-24 "+id, func(t *testing.T) {
			m, _ := cat.Map(id)
			scheduled := map[string]int{}
			for _, reinforcement := range m.Reinforcements {
				if reinforcement.ID != "" {
					scheduled[reinforcement.ID] = reinforcement.Turn
				}
			}
			seenScheduled := map[int]int{}
			seenIDs := map[string]bool{}
			wins := 0
			first := 0
			firstSpawn := 0
			firstMax := 0
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
					c = estimatedRosterIT06(t, cat, id, false)
				}
				chosen := map[string][]string{
					"ch01": {"leon", "gald", "bran", "mira", "rina"},
					"ch02": {"leon", "gald", "bran", "mira", "rina", "noel", "elsa"},
					"ch03": {"leon", "gald", "bran", "mira", "noel", "elsa"},
					"ch04": {"leon", "gald", "bran", "mira", "noel", "elsa", "rina"},
					"ch05": {"leon", "gald", "bran", "mira", "noel", "elsa"},
					"ch06": {"leon", "gald", "bran", "mira", "noel", "elsa", "tio"},
					"ch07": {"leon", "gald", "bran", "mira", "noel", "tio"},
					"ch08": {"leon", "gald", "bran", "mira", "noel", "elsa"},
					"ch09": {"leon", "gald", "bran", "mira", "noel", "elsa", "rina"},
				}[id]
				var members []int
				for _, want := range chosen {
					for i, m := range c.Roster {
						if m.Character == want {
							members = append(members, i)
							break
						}
					}
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
				autoObserve = func(events []tactics.Event) {
					for _, event := range events {
						if event.Kind == tactics.EventPhaseStart && event.Side == tactics.Enemy {
							turn = event.Turn
						}
						if event.Kind == tactics.EventReinforce {
							spawned++
							spawnTurns[turn]++
							if scheduledTurn, ok := scheduled[event.Units[event.Unit].ID]; ok {
								if turn != scheduledTurn {
									t.Fatalf("%s は %d ターン目に出現（定義 %d）", event.Units[event.Unit].ID, turn, scheduledTurn)
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
						if live > maxLive {
							maxLive = live
						}
					}
				}
				r := autoPlay(s.Battle, ids, autoPlan{Order: order, Routes: [][]tactics.Pos{routes[id]}}, 50)
				autoObserve = nil
				if maxLive > 39 {
					t.Fatalf("生存敵が %d 体", maxLive)
				}
				if r.Result == tactics.Won {
					wins++
					if first == 0 {
						first = int(seed)
						firstSpawn = spawned
						firstMax = maxLive
						firstTurns = spawnTurns
					}
				}
			}
			t.Logf("%s: 主進路 seed=%d、20 seed 中 %d 勝。出現した増援=%d、ターン別=%v、同時の敵の最大=%d", id, first, wins, firstSpawn, firstTurns, firstMax)
			if wins == 0 {
				t.Fatal("想定名簿で勝つ固定 seed がない")
			}
			if ch >= 2 {
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

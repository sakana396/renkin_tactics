package core_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestIT09Measure は手直し前後の比較用の試遊（DES-NG-23 案）。
// TC-DES-NG-23-01 は制作記録の進路、02 は 3 章だけ進路と隊分けを変えた試案、03 は制作記録の進路で工夫 Bait を足した打ち手。
func TestIT09Measure(t *testing.T) {
	for _, tc := range []struct {
		id                  string
		revised, bait, bomb bool
	}{{"TC-DES-NG-23-01", false, false, false}, {"TC-DES-NG-23-02", true, false, false}, {"TC-DES-NG-23-03", false, true, false}, {"TC-DES-NG-23-09", false, true, true}} {
		revised := tc.revised
		t.Run(tc.id, func(t *testing.T) {
			if os.Getenv("RENKIN_IT09_MEASURE") == "" {
				t.Skip("RENKIN_IT09_MEASURE を指定して実行")
			}
			cat := alchemyCatalog(t)
			routes := [][]tactics.Pos{{{X: 8, Y: 12}, {X: 8, Y: 8}, {X: 8, Y: 5}}, {{X: 4, Y: 10}, {X: 4, Y: 7}, {X: 8, Y: 3}}, {{X: 8, Y: 10}, {X: 8, Y: 6}, {X: 8, Y: 3}}}
			if revised {
				routes[2] = []tactics.Pos{{X: 12, Y: 2}}
			}
			for i, route := range routes {
				t.Run(fmt.Sprintf("ch%02d", i+1), func(t *testing.T) {
					wins, timeouts, measured := 0, 0, 0
					for seed := uint64(1); seed <= 20; seed++ {
						c, err := core.NewCampaignAt(cat, core.Withdraw, fmt.Sprintf("ch%02d", i+1))
						if err != nil {
							t.Fatal(err)
						}
						s, ids, err := autoCampaignBattle(c, seed, 6)
						if err != nil {
							t.Fatal(err)
						}
						f := smartAll
						f.Hold = false
						f.Bait = tc.bait
						f.Bomb = tc.bomb
						plan := autoPlan{Order: []string{"dain", "tori", "nemu", "aoi", "haru"}, Routes: [][]tactics.Pos{route}}
						if revised && i == 2 {
							plan.Groups = map[string]int{"dain": 0, "tori": 1, "nemu": 2, "aoi": 3, "haru": 4}
							plan.Routes = [][]tactics.Pos{route, route, route, route, route}
						}
						if tc.bomb && os.Getenv("RENKIN_IT09_SCENES") != "" {
							autoLog = func(line string) {
								if strings.Contains(line, "爆弾") {
									t.Logf("seed=%d %s", seed, line)
								}
							}
						}
						r := smartPlay(s.Battle, ids, plan, f, 40)
						if tc.bomb && os.Getenv("RENKIN_IT09_SCENES") != "" {
							autoLog = nil
						}
						measured++
						switch r.Result {
						case tactics.Won:
							wins++
						case tactics.Ongoing:
							timeouts++
						}
						t.Logf("seed=%d result=%v turns=%d fallen=%d %v bombs=%d bombkills=%d", seed, r.Result, r.Turns, len(r.Fallen), r.Fallen, r.BombUses, r.BombKills)
					}
					if measured != 20 {
						t.Fatalf("測定数=%d、期待20", measured)
					}
					t.Logf("wins=%d/20 timeouts=%d", wins, timeouts)
				})
			}
		})
	}
}

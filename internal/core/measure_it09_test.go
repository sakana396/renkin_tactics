package core_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestIT09Measure は手直し前後の比較用の試遊（DES-NG-23 案）。
func TestIT09Measure(t *testing.T) {
	t.Run("TC-DES-NG-23-01", func(t *testing.T) {
		if os.Getenv("RENKIN_IT09_MEASURE") == "" {
			t.Skip("RENKIN_IT09_MEASURE を指定して実行")
		}
		cat := alchemyCatalog(t)
		routes := [][]tactics.Pos{{{X: 8, Y: 12}, {X: 8, Y: 8}, {X: 8, Y: 5}}, {{X: 4, Y: 10}, {X: 4, Y: 7}, {X: 8, Y: 3}}, {{X: 8, Y: 10}, {X: 8, Y: 6}, {X: 8, Y: 3}}}
		for i, route := range routes {
			t.Run(fmt.Sprintf("ch%02d", i+1), func(t *testing.T) {
				wins, measured := 0, 0
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
					r := smartPlay(s.Battle, ids, autoPlan{Order: []string{"dain", "tori", "nemu", "aoi", "haru"}, Routes: [][]tactics.Pos{route}}, f, 40)
					measured++
					if r.Result == tactics.Won {
						wins++
					}
					t.Logf("seed=%d result=%v turns=%d fallen=%d %v", seed, r.Result, r.Turns, len(r.Fallen), r.Fallen)
				}
				if measured != 20 {
					t.Fatalf("測定数=%d、期待20", measured)
				}
				t.Logf("wins=%d/20", wins)
			})
		}
	})
}

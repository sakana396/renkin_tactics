package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 実データで新規開始から章の結果と調合を経て最後まで進む（DES-NG-06、DES-NG-21）。
func TestAlchemyCampaign209(t *testing.T) {
	t.Run("TC-DES-NG-06-18 新作の3章を通して勝つ", func(t *testing.T) {
		cat := alchemyCatalog(t)
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		routes := [][]tactics.Pos{{{X: 8, Y: 12}, {X: 8, Y: 8}, {X: 8, Y: 5}}, {{X: 4, Y: 10}, {X: 4, Y: 7}, {X: 8, Y: 3}}, {{X: 8, Y: 10}, {X: 8, Y: 6}, {X: 8, Y: 3}}}
		order := []string{"dain", "tori", "nemu", "aoi", "haru"}
		chapters := []string{"ch01", "ch02", "ch03"}
		for i, chapter := range chapters {
			if c.Chapter != i || c.Cleared {
				t.Fatalf("%s 開始前: chapter=%d cleared=%v", chapter, c.Chapter, c.Cleared)
			}
			foundHaru := false
			for _, m := range c.Roster {
				if m.Character == "haru" {
					foundHaru = true
				}
			}
			if foundHaru != (i >= 1) {
				t.Fatalf("%s のハル加入=%v", chapter, foundHaru)
			}
			s, ids, err := autoCampaignBattle(c, 2, 6)
			if err != nil {
				t.Fatal(err)
			}
			result := autoPlay(s.Battle, ids, autoPlan{Order: order, Routes: [][]tactics.Pos{routes[i]}}, 40)
			if result.Result != tactics.Won {
				t.Fatalf("%s: %v", chapter, result)
			}
			summary, err := c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			if summary.Chapter != chapter || summary.Cleared != (i == 2) || c.Cleared != (i == 2) {
				t.Fatalf("%s の結果: %+v", chapter, summary)
			}
			if i < 2 {
				ore, catalyst := c.Materials["ore"], c.Materials["catalyst"]
				if err := c.Craft("potion"); err != nil {
					t.Fatalf("%s 後の調合: %v", chapter, err)
				}
				if c.Materials["ore"] != ore-1 || c.Materials["catalyst"] != catalyst-1 {
					t.Fatal("調合で材料が消費されない")
				}
			}
		}
	})
}

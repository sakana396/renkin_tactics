package core_test

import (
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"testing"
)

// 3章の期待値は制作記録の配置から数える（DES-NG-06）。
func TestAlchemyChapterThree(t *testing.T) {
	cat := alchemyCatalog(t)
	t.Run("TC-DES-NG-06-16 3章の配置と爆弾と増援", func(t *testing.T) {
		m, ok := cat.Map("ch03")
		if !ok || len(m.Rows) != 18 || len([]rune(m.Rows[0])) != 18 || len(m.Slots) != 6 {
			t.Fatal("地形または枠")
		}
		if []rune(m.Rows[7])[3] != '#' || []rune(m.Rows[7])[14] != '#' {
			t.Fatal("両翼の壁")
		}
		boss, commander, target := 0, 0, 0
		for _, e := range m.Enemies {
			if e.Boss {
				boss++
				if e.X != 8 || e.Y != 1 || e.Bonus.HP != 8 {
					t.Fatal(e)
				}
			}
			if e.Commander {
				commander++
			}
			if e.X == 3 && e.Y == 6 && e.Class == "bowman" {
				target++
			}
		}
		item, _ := cat.Item("bomb_far")
		if len(m.Enemies) != 10 || boss != 1 || commander != 0 || target != 1 || item.ThrowRange != 5 || item.BlastRadius != 0 || item.BlastDamage != 10 || abs(3-3)+abs(11-6) != 5 {
			t.Fatal("配置または投げ火")
		}
		want := map[string]int{"bandit": 2, "bowman": 15, "spearman": 14}
		if len(m.Reinforcements) != 3 {
			t.Fatal(m.Reinforcements)
		}
		for _, e := range m.Reinforcements {
			x, ok := want[e.Class]
			if !ok || e.X != x || e.Y != 2 || e.Turn != 4 || e.AI != "charge" {
				t.Fatal(e)
			}
			delete(want, e.Class)
		}
		if len(want) != 0 {
			t.Fatal(want)
		}
	})
	t.Run("TC-DES-NG-06-17 3章の自動試遊", func(t *testing.T) {
		wins := 0
		for seed := uint64(1); seed <= 20; seed++ {
			c, err := core.NewCampaignAt(cat, core.Withdraw, "ch03")
			if err != nil {
				t.Fatal(err)
			}
			s, ids, err := autoCampaignBattle(c, seed, 6)
			if err != nil {
				t.Fatal(err)
			}
			r := autoPlay(s.Battle, ids, autoPlan{Order: []string{"dain", "tori", "nemu", "aoi", "haru"}, Routes: [][]tactics.Pos{{{X: 8, Y: 10}, {X: 8, Y: 6}, {X: 8, Y: 3}}}}, 40)
			if r.Result == tactics.Won {
				wins++
			}
			t.Logf("seed %d: %v", seed, r)
		}
		t.Logf("勝利 %d/20", wins)
		if wins < 1 {
			t.Fatalf("勝利 %d/20、基準 1", wins)
		}
	})
}

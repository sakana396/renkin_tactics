package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 期待値は 2 章の制作記録の配置と数値から数える（DES-NG-06）。
func TestAlchemyChapterTwo(t *testing.T) {
	cat := alchemyCatalog(t)
	t.Run("TC-DES-NG-06-14 2章の橋と配置と増援", func(t *testing.T) {
		m, ok := cat.Map("ch02")
		if !ok || len(m.Rows) != 18 || len([]rune(m.Rows[0])) != 18 {
			t.Fatal("マップ")
		}
		item, _ := cat.Item("bomb_small")
		distance := abs(4-4) + abs(9-5)
		if distance != 4 || distance > item.ThrowRange || distance <= item.BlastRadius || item.BlastRadius != 1 || item.BlastDamage != 8 || []rune(m.Rows[9])[4] != '.' {
			t.Fatal("投擲位置と爆弾の条件")
		}
		for x, c := range []rune(m.Rows[8]) {
			if (x == 4 || x == 13) != (c == '.') {
				t.Fatalf("橋 x=%d: %c", x, c)
			}
		}
		boss, charge, commander, inside, killed := 0, 0, 0, 0, 0
		for _, e := range m.Enemies {
			if e.Boss {
				boss++
			}
			if e.AI == "charge" {
				charge++
			}
			if e.Commander {
				commander++
			}
			if abs(e.X-4)+abs(e.Y-5) <= 1 {
				inside++
				c, _ := cat.Class(e.Class)
				if c.Base.HP+e.Bonus.HP <= 8 {
					killed++
				}
			}
		}
		if len(m.Enemies) != 11 || boss != 1 || charge != 8 || commander != 0 || inside != 3 || killed != 0 {
			t.Fatalf("敵 %d ボス %d 突撃 %d 小隊長 %d 範囲 %d 撃破 %d", len(m.Enemies), boss, charge, commander, inside, killed)
		}
		if len(m.Reinforcements) != 3 {
			t.Fatal(m.Reinforcements)
		}
		want := map[string]tactics.Pos{"bandit": {X: 2, Y: 2}, "bowman": {X: 15, Y: 2}, "spearman": {X: 14, Y: 2}}
		for _, e := range m.Reinforcements {
			pos, ok := want[e.Class]
			if !ok || pos.X != e.X || pos.Y != e.Y || e.Turn != 4 || e.AI != "charge" {
				t.Fatal(e)
			}
			delete(want, e.Class)
		}
		if len(want) != 0 {
			t.Fatal(want)
		}
		ch := cat.Chapters()[1]
		if len(ch.Joins) != 1 || ch.Joins[0] != "haru" {
			t.Fatal(ch.Joins)
		}
	})
	t.Run("TC-DES-NG-06-15 2章の自動試遊", func(t *testing.T) {
		wins := 0
		for seed := uint64(1); seed <= 20; seed++ {
			c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
			if err != nil {
				t.Fatal(err)
			}
			s, ids, err := autoCampaignBattle(c, seed, 6)
			if err != nil {
				t.Fatal(err)
			}
			r := autoPlay(s.Battle, ids, autoPlan{Order: []string{"dain", "tori", "nemu", "aoi", "haru"}, Routes: [][]tactics.Pos{{{X: 4, Y: 10}, {X: 4, Y: 7}, {X: 8, Y: 3}}}}, 40)
			if r.Result == tactics.Won {
				wins++
			}
			t.Logf("seed %d: %v", seed, r)
		}
		t.Logf("勝利 %d/20", wins)
		if wins < 8 {
			t.Fatalf("勝利 %d/20、基準 8", wins)
		}
	})
}

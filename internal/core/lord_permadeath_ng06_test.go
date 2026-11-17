package core_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// TestLordWinsWithoutPermadeath は、主人公とボスが同じ爆発で倒れて勝っても、ロストのルールで主人公が戦死にならず、
// 次の章に出撃できることを確かめる（DES-NG-06、デバッグの報告 B-2）。
func TestLordWinsWithoutPermadeath(t *testing.T) {
	t.Run("TC-DES-NG-06-13 ボスと主人公が同じ爆発で倒れて勝っても、ロストで主人公は戦死にならない", func(t *testing.T) {
		// 実データの大きな爆弾を、全員を巻き込む威力にした試験用の目録。
		m := fstest.MapFS{}
		err := fs.WalkDir(data.Alchemy(), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, e := fs.ReadFile(data.Alchemy(), p)
			if e != nil {
				return e
			}
			m[p] = &fstest.MapFile{Data: b}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		src := string(m["items.toml"].Data)
		const old = "blast_damage = 12\nblast_radius = 2"
		if !strings.Contains(src, old) {
			t.Fatal("大きな爆弾の定義が見つからない")
		}
		m["items.toml"].Data = []byte(strings.Replace(src, old, "blast_damage = 99\nblast_radius = 40", 1))
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		for _, rule := range []core.Rule{core.Permadeath, core.Withdraw} {
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			s, _, err := c.StartBattle([]int{0, 1, 2, 3}, 1)
			if err != nil {
				t.Fatal(err)
			}
			lord := s.Battle.Units()[0]
			if !lord.Lord {
				t.Fatal("先頭が主人公でない")
			}
			bomb := -1
			for k, it := range lord.Items {
				if it.Item.BlastDamage == 99 {
					bomb = k
				}
			}
			if bomb < 0 {
				t.Fatal("試験用の爆弾を持っていない")
			}
			at := lord.Pos
			if _, err := s.Battle.Act(tactics.Command{Unit: 0, To: at, Action: tactics.UseItem, Item: bomb, Blast: &at}); err != nil {
				t.Fatal(err)
			}
			if s.Battle.Result != tactics.Won || s.Battle.Units()[0].Alive {
				t.Fatalf("勝ちで主人公が倒れている状態にならない: %v alive=%v", s.Battle.Result, s.Battle.Units()[0].Alive)
			}
			r, err := c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			if !c.Roster[0].Alive || !r.Members[0].Fallen || r.Members[0].Dead {
				t.Fatalf("主人公は倒れたが戦死ではない（rule %v）: alive=%v %+v", rule, c.Roster[0].Alive, r.Members[0])
			}
			// 次の章で、主人公を先頭に出撃できる。
			if _, _, err := c.StartBattle([]int{0}, 1); err != nil {
				t.Fatalf("次の章に出撃できない（rule %v）: %v", rule, err)
			}
		}
	})
}

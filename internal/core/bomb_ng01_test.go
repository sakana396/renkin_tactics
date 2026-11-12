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
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// bombCatalog は、本体の定義データの items.toml に試験用の爆弾を足した目録を返す（本体のデータは変えない）。
func bombCatalog(t *testing.T) *defs.Catalog {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(data.Base(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), p)
		m[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	m["items.toml"].Data = append(m["items.toml"].Data, []byte("\n[[items]]\nid = \"bomb\"\nname = \"爆弾\"\nblast_damage = 6\nblast_radius = 1\nthrow_range = 3\n")...)
	cat, err := defload.Load(m)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// 爆弾の変換と行動の記録（DES-NG-04）。
func TestNG01Bomb(t *testing.T) {
	t.Run("TC-DES-NG-04-01 定義の爆弾の値を tactics のアイテムへそのまま渡す", func(t *testing.T) {
		cat := bombCatalog(t)
		d, _ := cat.Item("bomb")
		want := tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: 6, BlastRadius: 1, ThrowRange: 3}
		if got := core.ItemOf(d); got != want {
			t.Errorf("ItemOf = %+v, 期待 %+v", got, want)
		}
		p, _ := cat.Item("potion")
		if got, want := core.ItemOf(p), (tactics.Item{ID: "potion", Name: "回復薬", Heal: 10}); got != want {
			t.Errorf("回復薬の ItemOf = %+v, 期待 %+v", got, want)
		}
	})

	t.Run("TC-DES-NG-04-02 爆心が記録に残り、流し直しで同じ結果になる。爆弾でない操作の記録は変わらない", func(t *testing.T) {
		cat := bombCatalog(t)
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch01")
		if err != nil {
			t.Fatal(err)
		}
		m, _ := cat.Map(cat.Chapters()[c.Chapter].Map)
		var members, slots []int
		for i := range c.Roster {
			if len(members) < len(m.Slots) {
				members, slots = append(members, i), append(slots, len(slots))
			}
		}
		// 先頭の名簿の手持ちを爆弾 2 個にする（章の最初の戦闘に出る）。
		c.Roster[members[0]].Items = []defs.ItemCount{{Item: "bomb", Count: 2}}
		s, events, err := c.StartBattleAt(members, slots, 3)
		if err != nil {
			t.Fatal(err)
		}
		w := &countingWriter{}
		l := core.NewPlayLog(w, fakeClock())
		l.Start(c, s, slots, 3, events)
		b := s.Battle

		// 爆弾を使うユニットの添字と、位置の近くのマスを爆心にする。
		var user int
		for i, u := range b.Units() {
			if u.Side == tactics.Player && len(u.Items) > 0 && u.Items[0].Item.Bomb {
				user = i
				break
			}
		}
		from := b.Units()[user].Pos
		center := tactics.Pos{X: from.X, Y: from.Y + 2}
		if center.Y >= b.Height() {
			center.Y = from.Y - 2
		}
		if _, err := l.Act(b, tactics.Command{Unit: user, To: from, Action: tactics.UseItem, Item: 0, Blast: &center}); err != nil {
			t.Fatal(err)
		}
		// 別のユニットは待機する（爆弾でない操作の記録に blast が出ないことを見る）。
		other := -1
		for i, u := range b.Units() {
			if u.Side == tactics.Player && i != user && !u.Acted {
				other = i
				break
			}
		}
		if other < 0 {
			t.Fatal("待機できる自軍がいない")
		}
		if _, err := l.Act(b, tactics.Command{Unit: other, To: b.Units()[other].Pos, Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		if err := l.Err(); err != nil {
			t.Fatal(err)
		}

		recs := readLog(t, w.buf.String())
		var actions []map[string]any
		for _, r := range recs {
			if r["type"] == "action" {
				actions = append(actions, r)
			}
		}
		if len(actions) != 2 {
			t.Fatalf("action の行が %d 行", len(actions))
		}
		cmd := actions[0]["cmd"].(map[string]any)
		blast, _ := cmd["blast"].([]any)
		if len(blast) != 2 || num(blast[0]) != center.X || num(blast[1]) != center.Y || cmd["action"] != "item" || actions[0]["item_name"] != "爆弾" {
			t.Errorf("爆弾の操作の記録 = %v, 期待 爆心 %v", actions[0], center)
		}
		found := false
		for _, e := range actions[0]["events"].([]any) {
			if ev := e.(map[string]any); ev["kind"] == "blast" {
				to := ev["to"].([]any)
				found = num(to[0]) == center.X && num(to[1]) == center.Y && ev["item"] == "爆弾"
			}
		}
		if !found {
			t.Errorf("blast の出来事がない: %v", actions[0]["events"])
		}
		if _, has := actions[1]["cmd"].(map[string]any)["blast"]; has {
			t.Errorf("爆弾でない操作の記録に blast がある: %v", actions[1]["cmd"])
		}
		if strings.Contains(w.buf.String(), `"blast":null`) {
			t.Error("blast に null を書いている")
		}

		rep, err := core.ReplayPlayLog(cat, strings.NewReader(w.buf.String()))
		if err != nil {
			t.Fatal(err)
		}
		if rep.FirstDiff != 0 {
			t.Fatalf("流し直しが食い違った: %d 行目\n元: %s\n今: %s", rep.FirstDiff, rep.Want, rep.Got)
		}
	})
}

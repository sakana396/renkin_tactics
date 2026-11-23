package core_test

import (
	"bytes"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
	"strings"
	"testing"
)

// 本体定義と前章報酬を確かめる（DES-NG-13、DES-NG-17）。
func TestNG06DefinitionsAndRewards(t *testing.T) {
	cat := alchemyCatalog(t)
	t.Run("TC-DES-NG-13-01 本体の定義と変換", func(t *testing.T) {
		for _, tc := range []struct {
			id, effect       string
			amount, distance int
		}{{"tonic_power", "power_tonic", 4, 3}, {"snare_bottle", "snare", 0, 4}} {
			d, ok := cat.Item(tc.id)
			if !ok {
				t.Fatal(tc.id)
			}
			it := core.ItemOf(d)
			if it.Effect != tc.effect || it.Amount != tc.amount || it.Range != tc.distance {
				t.Fatal(it)
			}
		}
	})
	t.Run("TC-DES-NG-17-02 直接開始の前章報酬", func(t *testing.T) {
		for _, ch := range []string{"ch02", "ch03"} {
			c, err := core.NewCampaignAt(cat, core.Withdraw, ch)
			if err != nil {
				t.Fatal(err)
			}
			stock := map[string]int{}
			for _, s := range c.Stock {
				stock[s.Item] += s.Count
			}
			want := 1
			if ch == "ch03" {
				want = 2
			}
			if stock["tonic_power"] != want || stock["snare_bottle"] != want {
				t.Fatal(ch, stock)
			}
		}
	})
	t.Run("TC-DES-NG-17-01 報酬の定義", func(t *testing.T) {
		for _, chapter := range []string{"ch01", "ch02"} {
			c, err := core.NewCampaignAt(cat, core.Withdraw, chapter)
			if err != nil {
				t.Fatal(err)
			}
			before := map[string]int{}
			for _, item := range c.Stock {
				before[item.Item] = item.Count
			}
			s, _, err := c.StartBattle([]int{0}, 1)
			if err != nil {
				t.Fatal(err)
			}
			// 勝利後の処理を隔離して、報酬と準備の受け渡しを確かめる。
			s.Battle.Result = tactics.Won
			if _, err = c.FinishChapter(s); err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"tonic_power", "snare_bottle"} {
				count := 0
				for _, item := range c.Stock {
					if item.Item == id {
						count = item.Count
					}
				}
				if count != before[id]+1 {
					t.Fatal(chapter, id, count)
				}
			}
			if err = c.StoreItem(0, c.Roster[0].Items[0].Item); err != nil {
				t.Fatal(err)
			}
			if err = c.TakeItem(0, "tonic_power"); err != nil {
				t.Fatal(err)
			}
		}
	})
}

// 新道具の対象座標と効果の出来事を記録して再生する（DES-NG-15）。
func TestNG06PlayLog(t *testing.T) {
	t.Run("TC-DES-NG-15-02 新道具の記録再生", func(t *testing.T) {
		cat := alchemyCatalog(t)
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		if err = c.StoreItem(0, c.Roster[0].Items[0].Item); err != nil {
			t.Fatal(err)
		}
		if err = c.TakeItem(0, "tonic_power"); err != nil {
			t.Fatal(err)
		}
		s, events, err := c.StartBattle([]int{0}, 11)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		l := core.NewPlayLog(&buf, nil)
		l.Start(c, s, []int{0}, 11, events)
		u := s.Battle.Units()[0]
		slot := -1
		for i, it := range u.Items {
			if it.Item.Effect == "power_tonic" {
				slot = i
			}
		}
		if slot < 0 {
			t.Fatal("薬がない")
		}
		p := u.Pos
		_, err = l.Act(s.Battle, tactics.Command{Unit: 0, To: p, Action: tactics.UseItem, Item: slot, ItemTarget: &p})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = l.EndPhase(s.Battle); err != nil {
			t.Fatal(err)
		}
		text := buf.String()
		for _, want := range []string{`"item_target"`, `"kind":"item_use"`, `"kind":"effect_apply"`, `"kind":"effect_expire"`} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s がない: %s", want, text)
			}
		}
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(text))
		if err != nil || rep.FirstDiff != 0 {
			t.Fatalf("%+v %v", rep, err)
		}
	})
}

// 既存の自動の打ち手が新道具を選ばないことを確認する（DES-NG-15）。
func TestNG06AutoPlayer(t *testing.T) {
	t.Run("TC-DES-NG-15-03 自動の打ち手は新道具を使わない", func(t *testing.T) {
		cat := alchemyCatalog(t)
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Items = []defs.ItemCount{{Item: "tonic_power", Count: 1}, {Item: "snare_bottle", Count: 1}}
		s, ids, err := autoCampaignBattle(c, 1, 5)
		if err != nil {
			t.Fatal(err)
		}
		autoPlay(s.Battle, ids, autoPlan{}, 1)
		items := s.Battle.Units()[0].Items
		if len(items) != 2 || items[0].Count != 1 || items[1].Count != 1 {
			t.Fatal(items)
		}
	})
}

// 粘着瓶の座標と期限切れも記録を流し直せる（DES-NG-15）。
func TestNG06SnareLog(t *testing.T) {
	t.Run("TC-DES-NG-15-02 粘着瓶の記録再生", func(t *testing.T) {
		cat := alchemyCatalog(t)
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch01")
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Items = []defs.ItemCount{{Item: "snare_bottle", Count: 1}}
		s, events, err := c.StartBattle([]int{0}, 11)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		l := core.NewPlayLog(&buf, nil)
		l.Start(c, s, []int{0}, 11, events)
		b := s.Battle
		found := false
		for turn := 0; turn < 6 && b.Result == tactics.Ongoing && !found; turn++ {
			for _, target := range b.Units() {
				if !target.Alive || target.Side != tactics.Enemy {
					continue
				}
				p := target.Pos
				for to := range b.Reachable(0) {
					cmd := tactics.Command{Unit: 0, To: to, Action: tactics.UseItem, Item: 0, ItemTarget: &p}
					if _, err := b.ItemForecast(cmd); err == nil {
						if _, err = l.Act(b, cmd); err != nil {
							t.Fatal(err)
						}
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				best, score := b.Units()[0].Pos, 9999
				for to := range b.Reachable(0) {
					for _, enemy := range b.Units() {
						if enemy.Side != tactics.Enemy || !enemy.Alive {
							continue
						}
						d := abs(to.X-enemy.Pos.X) + abs(to.Y-enemy.Pos.Y)
						if d < score || d == score && (to.Y < best.Y || to.Y == best.Y && to.X < best.X) {
							best, score = to, d
						}
					}
				}
				if _, err = l.Act(b, tactics.Command{Unit: 0, To: best, Action: tactics.Wait}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = l.EndPhase(b); err != nil {
				t.Fatal(err)
			}
		}
		if !found {
			t.Fatal("粘着瓶を使える位置に到達しない")
		}
		if !strings.Contains(buf.String(), `"effect":"snare"`) || !strings.Contains(buf.String(), `"kind":"effect_expire"`) {
			t.Fatal(buf.String())
		}
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(buf.String()))
		if err != nil || rep.FirstDiff != 0 {
			t.Fatalf("%+v %v", rep, err)
		}
	})
}

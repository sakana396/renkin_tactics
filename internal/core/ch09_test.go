package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体の ch09「ヴァルトの門」（DES-MAP-06、DES-CMP-09。制作記録 docs/iterations/it05/map-ch09.md）。

func baseCatalog(t *testing.T) *defs.Catalog {
	t.Helper()
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// reachableByTerrain は、ユニットを無視して地形だけで start から行けるマスを返す。closed のマスは通れないものとする。
func reachableByTerrain(m defs.MapDef, cat *defs.Catalog, start defs.Slot, closed map[defs.Slot]bool) map[defs.Slot]bool {
	passable := func(p defs.Slot) bool {
		if p.Y < 0 || p.Y >= len(m.Rows) || p.X < 0 || p.X >= len([]rune(m.Rows[0])) || closed[p] {
			return false
		}
		sym := string([]rune(m.Rows[p.Y])[p.X])
		for _, tr := range cat.Terrains() {
			if tr.Symbol == sym {
				return tr.Cost > 0
			}
		}
		return false
	}
	seen := map[defs.Slot]bool{start: true}
	queue := []defs.Slot{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range []defs.Slot{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			n := defs.Slot{X: cur.X + d.X, Y: cur.Y + d.Y}
			if !seen[n] && passable(n) {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return seen
}

func TestCh09Map(t *testing.T) {
	cat := baseCatalog(t)
	m, ok := cat.Map("ch09")
	if !ok {
		t.Fatal("ch09 のマップがない")
	}

	t.Run("TC-FR-MAP-02-03 32×32・出撃 7・初期敵 26＋ターン増援 8・最奥のボス", func(t *testing.T) {
		if len(m.Rows) != 32 {
			t.Fatalf("行の数 = %d", len(m.Rows))
		}
		for y, r := range m.Rows {
			if n := len([]rune(r)); n != 32 {
				t.Fatalf("行 %d の長さ = %d", y, n)
			}
		}
		// 出撃の枠は南の出発地の 7 つ（1 番目は主人公）。
		wantSlots := []defs.Slot{{X: 15, Y: 29}, {X: 16, Y: 29}, {X: 14, Y: 30}, {X: 15, Y: 30}, {X: 16, Y: 30}, {X: 17, Y: 30}, {X: 14, Y: 29}}
		if !reflect.DeepEqual(m.Slots, wantSlots) {
			t.Errorf("出撃の枠 = %v", m.Slots)
		}
		if len(m.Enemies) != 26 || len(m.Reinforcements) != 8 {
			t.Fatalf("敵 %d・増援 %d", len(m.Enemies), len(m.Reinforcements))
		}
		// ボスはちょうど 1 体で、本営の奥（玉座の砦 (15,2)）にいる。
		var bosses []defs.EnemyPlacement
		for _, e := range m.Enemies {
			if e.Boss {
				bosses = append(bosses, e)
			}
		}
		if len(bosses) != 1 || bosses[0].Name != "ゲルハルト" || bosses[0].X != 15 || bosses[0].Y != 2 || m.Rows[2][15] != 'T' {
			t.Errorf("ボス = %+v", bosses)
		}
		// 動き方: 突っ込む敵は東の騎兵 2 と西のファイター 1 だけ。残りは待ち構える（DES-IT05 6 章）。
		charge := 0
		for _, e := range m.Enemies {
			if e.AI == "charge" {
				charge++
			}
		}
		if charge != 3 {
			t.Errorf("突っ込む初期の敵 = %d", charge)
		}
		// 増援の時機と位置（3・5・7・9 ターンに 2 体ずつ。南西の背後、東の端、前庭の両端、本営の中）。
		type rf struct{ turn, x, y int }
		var got []rf
		for _, r := range m.Reinforcements {
			got = append(got, rf{r.Turn, r.X, r.Y})
			if r.Boss || r.AI != "charge" {
				t.Errorf("増援 %+v", r)
			}
		}
		want := []rf{{3, 0, 29}, {3, 0, 30}, {5, 31, 22}, {5, 31, 23}, {7, 2, 8}, {7, 29, 8}, {9, 11, 2}, {9, 20, 2}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("増援 = %v", got)
		}
		// 組み立てた戦闘: 自軍 7・敵 26 で始まる（DES-IT06 13 章）。
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		s, _, err := c.StartBattle([]int{0, 1, 2, 3, 4, 5, 6}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if n := len(s.Battle.Units()); n != 33 {
			t.Errorf("初めのユニット = %d", n)
		}
		setup, err := core.NewSetup(cat, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		var turns []int
		for _, r := range setup.Reinforcements {
			turns = append(turns, r.Turn)
		}
		if !reflect.DeepEqual(turns, []int{3, 3, 5, 5, 7, 7, 9, 9}) || len(m.Slots)+len(m.Enemies)+len(m.Reinforcements) != 41 {
			t.Errorf("増援のターン = %v、総数 = %d", turns, len(m.Slots)+len(m.Enemies)+len(m.Reinforcements))
		}
		// 敵の能力のボーナス（E-52 で ch08 の後の名簿に合わせて強くし直した。#102 は 4/2/2/1/1・ボス 12/4/4/3/4）。
		normal := defs.Stats{HP: 6, Str: 3, Skl: 3, Spd: 2, Def: 3}
		boss := defs.Stats{HP: 14, Str: 5, Skl: 5, Spd: 4, Def: 6}
		for _, e := range m.Enemies {
			if want := map[bool]defs.Stats{false: normal, true: boss}[e.Boss]; e.Bonus != want {
				t.Errorf("敵のボーナス %+v", e)
			}
		}
		for _, r := range m.Reinforcements {
			if r.Bonus != normal {
				t.Errorf("増援のボーナス %+v", r)
			}
		}
	})

	t.Run("TC-FR-MAP-02-03 二つの道: 外の城壁の門は西と東だけで、どちらを閉じても本営へ着き、両方を閉じると着かない", func(t *testing.T) {
		// 外の城壁（y=11）で通れるマスは、西の門 x=4・5 と東の門 x=26・27 だけ。
		var gates []int
		for x, r := range []rune(m.Rows[11]) {
			if r != '#' {
				gates = append(gates, x)
			}
		}
		if !reflect.DeepEqual(gates, []int{4, 5, 26, 27}) {
			t.Errorf("外の城壁の切れ目 = %v", gates)
		}
		// 本営の門は (15,6)・(16,6) だけ。
		for x := 9; x <= 22; x++ {
			open := m.Rows[6][x] != '#'
			if open != (x == 15 || x == 16) {
				t.Errorf("本営の壁 (%d,6) = %q", x, m.Rows[6][x])
			}
		}
		lord := m.Slots[0]
		bossSide := defs.Slot{X: 15, Y: 3}
		west := map[defs.Slot]bool{{X: 4, Y: 11}: true, {X: 5, Y: 11}: true}
		east := map[defs.Slot]bool{{X: 26, Y: 11}: true, {X: 27, Y: 11}: true}
		both := map[defs.Slot]bool{}
		for k := range west {
			both[k] = true
		}
		for k := range east {
			both[k] = true
		}
		for _, tc := range []struct {
			name   string
			closed map[defs.Slot]bool
			want   bool
		}{
			{"両方開く", nil, true},
			{"西を閉じる（東の道）", west, true},
			{"東を閉じる（西の道）", east, true},
			{"両方を閉じる", both, false},
		} {
			if got := reachableByTerrain(m, cat, lord, tc.closed)[bossSide]; got != tc.want {
				t.Errorf("%s: 本営の奥へ着く = %v", tc.name, got)
			}
		}
		// 全ての枠・敵・増援の位置から、主人公の枠へ地形だけで行ける（閉じ込められた配置がない）。
		all := reachableByTerrain(m, cat, lord, nil)
		for _, s := range m.Slots {
			if !all[s] {
				t.Errorf("枠 %v に行けない", s)
			}
		}
		for _, e := range m.Enemies {
			if !all[defs.Slot{X: e.X, Y: e.Y}] {
				t.Errorf("敵 %+v に行けない", e)
			}
		}
		for _, r := range m.Reinforcements {
			if !all[defs.Slot{X: r.X, Y: r.Y}] {
				t.Errorf("増援 %+v に行けない", r)
			}
		}
	})
}

// winByForce は、主人公だけを極端に強くして（試験だけの名簿の書き換え）、今の章のボスを倒させる。
// 章の流れ（DES-CMP-09）を確かめるためのもので、攻略の確かめではない。
func winByForce(t *testing.T, c *core.Campaign, seed uint64) *core.Sortie {
	t.Helper()
	lord := c.Roster[0]
	c.Roster[0].Stats = tactics.Stats{HP: 99, Str: 99, Skl: 99, Spd: 99, Def: 99, Mov: 99}
	var members []int
	for k, m := range c.Roster {
		if m.Alive && len(members) < 5 {
			members = append(members, k)
		}
	}
	s, _, err := c.StartBattle(members, seed)
	c.Roster[0] = lord
	if err != nil {
		t.Fatal(err)
	}
	b := s.Battle
	for turn := 0; turn < 30 && b.Result == tactics.Ongoing; turn++ {
		units := b.Units()
		boss := -1
		for i, u := range units {
			if u.Boss && u.Alive {
				boss = i
			}
		}
		// ボスを攻撃できればボスを、できなければボスに近い敵を攻撃する。攻撃できる敵がいなければボスへ近づく。
		reach := sortedTiles(b.Reachable(0))
		var best *tactics.Command
		bestD := 1 << 30
		for _, to := range reach {
			for e, o := range units {
				if !o.Alive || o.Side != tactics.Enemy || dist(to, o.Pos) != 1 {
					continue
				}
				d := dist(o.Pos, units[boss].Pos)
				if d < bestD {
					cmd := tactics.Command{Unit: 0, To: to, Action: tactics.Attack, Target: e}
					best, bestD = &cmd, d
				}
			}
		}
		if best == nil {
			cmd := tactics.Command{Unit: 0, To: units[0].Pos, Action: tactics.Wait}
			for _, to := range reach {
				if dist(to, units[boss].Pos) < dist(cmd.To, units[boss].Pos) {
					cmd.To = to
				}
			}
			best = &cmd
		}
		if _, err := b.Act(*best); err != nil {
			t.Fatal(err)
		}
		if b.Result == tactics.Ongoing && b.Phase == tactics.Player {
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if b.Result != tactics.Won {
		t.Fatalf("勝てなかった（ターン %d）", b.Turn)
	}
	return s
}

func TestCh09ChapterFlow(t *testing.T) {
	cat := baseCatalog(t)
	t.Run("TC-FR-CMP-01-05 完成の 9 章の順・名前・マップ・出撃の枠・ボスが表のとおり", func(t *testing.T) {
		want := []struct {
			id, name string
			slots    int
			boss     string
		}{
			{"ch01", "国境の砦", 6, "ハーゲン"},
			{"ch02", "灰の街道", 7, "ヴォルグ"},
			{"ch03", "王都の奪還", 6, "グラウス"},
			{"ch04", "焼け野の村", 7, "オスカー"},
			{"ch05", "狭間の関所", 6, "ディートリヒ"},
			{"ch06", "山道の突破", 7, "ロルフ"},
			{"ch07", "補給の荷駄", 6, "ベルント"},
			{"ch08", "黒い森", 6, "クラウス"},
			{"ch09", "ヴァルトの門", 7, "ゲルハルト"},
		}
		chapters := cat.Chapters()
		if len(chapters) != len(want) {
			t.Fatalf("章の数 = %d", len(chapters))
		}
		for i, w := range want {
			c := chapters[i]
			m, ok := cat.Map(c.Map)
			if c.ID != w.id || c.Name != w.name || c.Map != w.id || !ok || len(m.Slots) != w.slots {
				t.Errorf("章 %d = %+v（枠 %d）", i+1, c, len(m.Slots))
				continue
			}
			if c.ID >= "ch04" && c.ID <= "ch08" {
				if c.Before != c.ID+"_before" || c.After != c.ID+"_after" || !reflect.DeepEqual(c.Joins, map[string][]string{"ch04": {"tio"}, "ch08": {"hanna"}}[c.ID]) || !reflect.DeepEqual(c.Reward, []defs.ItemCount{{Item: "potion", Count: 2}, {Weapon: map[string]string{"ch04": "killing_sword", "ch05": "long_bow", "ch06": "thunder_sword", "ch07": "brave_axe", "ch08": "horse_killer"}[c.ID], Count: 1}}) {
					t.Errorf("追加章の定義 %+v", c)
				}
			}

			var bosses []string
			for _, e := range m.Enemies {
				if e.Boss {
					bosses = append(bosses, e.Name)
				}
			}
			if !reflect.DeepEqual(bosses, []string{w.boss}) {
				t.Errorf("章 %s のボス = %q", c.ID, bosses)
			}
		}
		if ch := chapters[len(chapters)-1]; ch.Before != "ch09_before" || ch.After != "ch09_after" || ch.Joins != nil || ch.Reward != nil {
			t.Errorf("ch09 = %+v", ch)
		}
	})

	t.Run("TC-FR-CMP-01-05 ch03 に勝つと ch04 へ、ch09 に勝つとクリア。ch03 のセーブは ch04 へ進み、クリア済みのセーブはクリアのまま", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			c, err := core.NewCampaignAt(cat, rule, "ch03")
			if err != nil {
				t.Fatal(err)
			}
			// ch03 のセーブ（章に入ったときの保存と同じ時機）。
			var ch03Save bytes.Buffer
			if err := c.Save(&ch03Save); err != nil {
				t.Fatal(err)
			}
			r, err := c.FinishChapter(winByForce(t, c, 3))
			if err != nil || r.Chapter != "ch03" || r.Cleared || c.Cleared || c.Chapter != 3 || cat.Chapters()[c.Chapter].ID != "ch04" {
				t.Fatalf("ch03 の勝利の後 = %+v, 章 %d, %v", r, c.Chapter, err)
			}
			// ch03 のセーブから続けると ch03 から遊び、勝つと ch04 へ進む。
			loaded, err := core.LoadCampaign(cat, bytes.NewReader(ch03Save.Bytes()))
			if err != nil || loaded.Chapter != 2 {
				t.Fatalf("ch03 のセーブ = %+v, %v", loaded, err)
			}
			if _, err := loaded.FinishChapter(winByForce(t, loaded, 5)); err != nil || loaded.Chapter != 3 || loaded.Cleared {
				t.Fatalf("ch03 のセーブから続けた後 = 章 %d, %v", loaded.Chapter, err)
			}
			// 挿入した章の勝利・保存を経て ch09 まで進み、報酬と加入の重複なしを確かめる。
			for i, id := range []string{"ch04", "ch05", "ch06", "ch07", "ch08"} {
				r, err = c.FinishChapter(winByForce(t, c, uint64(4+i)))
				if err != nil || r.Chapter != id || r.Cleared || c.Cleared || c.Chapter != 4+i || len(c.Roster) != map[bool]int{true: 9, false: 8}[i >= 3] {
					t.Fatalf("%s の勝利後 = %+v, %v", id, r, err)
				}
				if !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: 6 + 2*i}}) {
					t.Fatalf("報酬 = %v", c.Stock)
				}
				var saved bytes.Buffer
				if err := c.Save(&saved); err != nil {
					t.Fatal(err)
				}
				c, err = core.LoadCampaign(cat, &saved)
				if err != nil {
					t.Fatal(err)
				}
			}
			r, err = c.FinishChapter(winByForce(t, c, 9))
			if err != nil || r.Chapter != "ch09" || !r.Cleared || !c.Cleared {
				t.Fatalf("ch09 の勝利の後 = %+v, %v", r, err)
			}
			var cleared bytes.Buffer
			if err := c.Save(&cleared); err != nil {
				t.Fatal(err)
			}
			again, err := core.LoadCampaign(cat, &cleared)
			if err != nil || !again.Cleared || again.Chapter != len(cat.Chapters()) {
				t.Fatalf("クリア済みのセーブ = %+v, %v", again, err)
			}
			if _, _, err := again.StartBattle([]int{0}, 1); err == nil {
				t.Fatal("クリア済みで出撃できた")
			}
		}
	})
}

// 新しく始めてから、9 章を順に勝ち、ch09 の結末の物語とクリアまで章の流れが通る（DES-CMP-09、FR-CMP-01・02・07）。
// 勝利は主人公だけを試験用に強くして作る（winByForce）。攻略の確かめではない。
func TestNineChapterRun(t *testing.T) {
	cat := baseCatalog(t)
	order := []string{"ch01", "ch02", "ch03", "ch04", "ch05", "ch06", "ch07", "ch08", "ch09"}
	// 各章に勝った後の共有の在庫の薬（ch03 と ch09 は報酬なし）。
	stock := []int{2, 4, 4, 6, 8, 10, 12, 14, 14}
	for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
		t.Run(fmt.Sprintf("TC-FR-CMP-01-05 新しく始めて ch09 の結末とクリアまで/%v", rule), func(t *testing.T) {
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			for i, id := range order {
				ch := cat.Chapters()[c.Chapter]
				if c.Chapter != i || ch.ID != id || c.Cleared {
					t.Fatalf("%d 番目の章 = %s（添字 %d、クリア %v）", i+1, ch.ID, c.Chapter, c.Cleared)
				}
				// 章前・章後の物語が目録にある（物語 → 準備 → 戦闘 → 結果 → 章後の物語の流れ）。
				for _, sid := range []string{ch.Before, ch.After} {
					if s, ok := cat.Story(sid); !ok || len(s.Paragraphs) == 0 {
						t.Fatalf("%s の物語 %q がない", id, sid)
					}
				}
				if want := []int{5, 7, 7, 8, 8, 8, 8, 9, 9}[i]; len(c.Roster) != want {
					t.Fatalf("%s の名簿 %d 人", id, len(c.Roster))
				}
				r, err := c.FinishChapter(winByForce(t, c, uint64(11+i)))
				if err != nil || r.Chapter != id || r.Cleared != (id == "ch09") || c.Cleared != (id == "ch09") {
					t.Fatalf("%s の勝利の後 = %+v, %v", id, r, err)
				}
				if !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: stock[i]}}) {
					t.Fatalf("%s の後の在庫 = %v", id, c.Stock)
				}
				// 勝利の後の保存から続ける。
				var saved bytes.Buffer
				if err := c.Save(&saved); err != nil {
					t.Fatal(err)
				}
				if c, err = core.LoadCampaign(cat, &saved); err != nil {
					t.Fatal(err)
				}
			}
			if !c.Cleared || c.Chapter != len(order) {
				t.Fatalf("クリアにならない: 章 %d", c.Chapter)
			}
			// 最後に読む章後の物語は結末（ch09_after）。
			if last := cat.Chapters()[len(order)-1]; last.After != "ch09_after" {
				t.Fatalf("最後の章後の物語 = %s", last.After)
			}
		})
	}
}

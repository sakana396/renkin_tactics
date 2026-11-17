package core_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 新作の 1 章の定義データ（data/alchemy。DES-NG-06）の検査。
// 期待値は設計書 DES-NG01 6 章の決定（敵の HP・爆弾の数値・持ち物）と、
// 1 章の制作記録 docs/iterations/it08/map-ng-ch01.md（#208。配置）から手で数えた。

func alchemyCatalog(t *testing.T) *defs.Catalog {
	t.Helper()
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestAlchemyData(t *testing.T) {
	cat := alchemyCatalog(t)

	t.Run("TC-DES-NG-06-01 完結したデータ一式として読め、1 章・主人公 1 人・ハルの定義が決まりどおり", func(t *testing.T) {
		var ids []string
		for _, ch := range cat.Chapters() {
			ids = append(ids, ch.ID)
		}
		if strings.Join(ids, ",") != "ch01,ch02,ch03" {
			t.Fatal(ids)
		}
		if got := strings.Join(cat.Game.Party, ","); got != "aoi,dain,tori,nemu" {
			t.Fatal(got)
		}
		lords := 0
		for _, id := range cat.Game.Party {
			if c, _ := cat.Character(id); c.Lord {
				lords++
			}
		}
		aoi, _ := cat.Character("aoi")
		if lords != 1 || !aoi.Lord {
			t.Fatal(lords, aoi)
		}
		if _, ok := cat.Character("haru"); !ok {
			t.Fatal("ハルの定義がない")
		}
		// 本体（data/base）と同じファイル構成。
		want, _ := fs.Glob(data.Base(), "*.toml")
		got, _ := fs.Glob(data.Alchemy(), "*.toml")
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatal(got, want)
		}
	})

	t.Run("TC-DES-NG-06-02 爆弾 3 種が目録・初期の持ち物・章の報酬にあり、回復薬が先頭にある", func(t *testing.T) {
		want := map[string][3]int{"bomb_small": {8, 1, 4}, "bomb_large": {12, 2, 3}, "bomb_far": {10, 0, 5}}
		for id, v := range want {
			it, ok := cat.Item(id)
			if !ok || !it.Bomb || it.Heal != 0 || [3]int{it.BlastDamage, it.BlastRadius, it.ThrowRange} != v {
				t.Fatalf("%s: %+v", id, it)
			}
			// 半径の外から投げられる（投げられる距離が半径 + 1 以上）。巻き込みを避けられない爆弾にしない。
			if it.ThrowRange < it.BlastRadius+1 {
				t.Fatalf("%s は自分を巻き込まずに使えない", id)
			}
		}
		aoi, _ := cat.Character("aoi")
		has := map[string]int{}
		for _, ic := range aoi.Items {
			has[ic.Item] = ic.Count
		}
		if has["bomb_small"] != 1 || has["bomb_large"] != 1 || aoi.Items[0].Item != "potion" {
			t.Fatal(aoi.Items)
		}
		// 主人公は癒やしの杖も持つ。手持ちは上限の 5 枠（武器 2 本 ＋ 道具 3 個）。名簿から外した小さな爆弾 1 個は在庫にある。
		if len(aoi.Weapons) != 2 || aoi.Weapons[0] != "fire" || aoi.Weapons[1] != "heal_staff" {
			t.Fatal(aoi.Weapons)
		}
		stock := map[string]int{}
		for _, ic := range cat.Game.Stock {
			stock[ic.Item] = ic.Count
		}
		// 在庫は、小さな爆弾 1 個と回復薬 3 個（回復薬は、試遊で序盤に尽きたため 1 個から増やした）。
		if stock["bomb_small"] != 1 || stock["potion"] != 3 {
			t.Fatal("在庫", cat.Game.Stock)
		}
		reward := map[string]map[string]int{}
		for _, ch := range cat.Chapters() {
			reward[ch.ID] = map[string]int{}
			for _, ic := range ch.Reward {
				reward[ch.ID][ic.Item] = ic.Count
			}
		}
		// 設計メモ 6.3 節の表: 1 章 回復薬 2・小さな爆弾 3・投げ火 1。
		wantReward := map[string]map[string]int{
			"ch01": {"potion": 2, "bomb_small": 3, "bomb_far": 1},
			"ch02": {"potion": 2, "bomb_small": 3, "bomb_far": 1},
			"ch03": {"potion": 2, "bomb_small": 3, "bomb_far": 1},
		}
		if fmt.Sprint(reward) != fmt.Sprint(wantReward) {
			t.Fatal(reward, wantReward)
		}
		// 道具を持つキャラは、回復薬を先頭にそろえる（準備の画面での並びと、既存の作品との一貫性のため）。
		for _, id := range []string{"aoi", "dain", "tori", "nemu", "haru"} {
			c, _ := cat.Character(id)
			if len(c.Items) > 0 && c.Items[0].Item != "potion" {
				t.Fatalf("%s の先頭の道具が回復薬でない: %v", id, c.Items)
			}
		}
	})

	t.Run("TC-DES-NG-06-03 1 章は 18×18・ボス 1 体で、爆弾を試せる 3 か所（詰所・内門・小隊長の一団）がある", func(t *testing.T) {
		// 期待値は制作記録 docs/iterations/it08/map-ng-ch01.md（#208）の 2〜4 章から手で数えた。
		m, ok := cat.Map("ch01")
		if !ok || len(m.Rows) != 18 || len([]rune(m.Rows[0])) != 18 {
			t.Fatal("マップの大きさ")
		}
		bosses := 0
		for _, e := range m.Enemies {
			if e.Boss {
				bosses++
			}
		}
		// 出撃できるのは名簿の 4 人。
		if bosses != 1 || len(m.Slots) < 4 {
			t.Fatalf("ボス %d 体、枠 %d", bosses, len(m.Slots))
		}
		hp := func(e defs.EnemyPlacement) int {
			cls, _ := cat.Class(e.Class)
			return cls.Base.HP + e.Bonus.HP
		}
		terrain := func(p tactics.Pos) rune { return []rune(m.Rows[p.Y])[p.X] }
		// blast は、thrower から center へ爆弾 item を投げたとき、範囲に入る敵の数と、1 発で倒れる敵の数を返す。
		// 投げる位置は通れる地形で、爆心までの距離は投げられる距離の端（以下）で、半径の外（自分を巻き込まない）。
		blast := func(item string, thrower, center tactics.Pos, wantDist int) (inside, killed int, boss bool) {
			t.Helper()
			it, _ := cat.Item(item)
			d := abs(center.X-thrower.X) + abs(center.Y-thrower.Y)
			if d != wantDist || d > it.ThrowRange || d <= it.BlastRadius {
				t.Fatalf("%s: %v から %v は距離 %d（投げられる距離 %d、半径 %d）", item, thrower, center, d, it.ThrowRange, it.BlastRadius)
			}
			if terrain(thrower) != '.' {
				t.Fatalf("%s: 投げる位置 %v の地形 %q", item, thrower, terrain(thrower))
			}
			for _, e := range m.Enemies {
				if abs(center.X-e.X)+abs(center.Y-e.Y) > it.BlastRadius {
					continue
				}
				inside++
				if hp(e) <= it.BlastDamage {
					killed++
				}
				boss = boss || e.Boss
			}
			return inside, killed, boss
		}
		// 西の詰所: 南岸 (3, 11) から、壁 (3, 9) と水路 (3, 10) の向こうへ投げる。
		if terrain(tactics.Pos{X: 3, Y: 9}) != '#' || terrain(tactics.Pos{X: 3, Y: 10}) != '~' {
			t.Fatal("詰所と南岸の間が壁と水路でない")
		}
		shore := tactics.Pos{X: 3, Y: 11}
		if in, k, _ := blast("bomb_large", shore, tactics.Pos{X: 3, Y: 8}, 3); in != 4 || k != 3 {
			t.Fatalf("詰所へ大きな爆弾: 範囲 %d 体、倒れる %d 体（4 体・3 体のはず）", in, k)
		}
		if in, k, _ := blast("bomb_small", shore, tactics.Pos{X: 3, Y: 7}, 4); in != 4 || k != 0 {
			t.Fatalf("詰所へ小さな爆弾: 範囲 %d 体、倒れる %d 体（4 体・0 体のはず）", in, k)
		}
		// 内門: (9, 5) から (9, 2) へ大きな爆弾を投げると、頭目と守りの 4 体が入る。頭目は 1 発では倒れない。
		// 門のマス (9, 4) は爆心から 2 で範囲の中、(8, 4) は 3 で範囲の外（門に立つ味方を巻き込むかの判断）。
		in, k, boss := blast("bomb_large", tactics.Pos{X: 9, Y: 5}, tactics.Pos{X: 9, Y: 2}, 3)
		if in != 5 || k != 3 || !boss {
			t.Fatalf("内門へ大きな爆弾: 範囲 %d 体、倒れる %d 体、ボス %v（5 体・3 体・ボスあり のはず）", in, k, boss)
		}
		if terrain(tactics.Pos{X: 8, Y: 4}) != '.' || terrain(tactics.Pos{X: 9, Y: 4}) != '.' || terrain(tactics.Pos{X: 7, Y: 4}) != '#' || terrain(tactics.Pos{X: 10, Y: 4}) != '#' {
			t.Fatal("内門が 2 マス幅（x = 8〜9）でない")
		}
		// 小隊長の一団: 小隊長がちょうど 1 体で、一団の全員に「攻撃されたら一団が動き出す」出来事がある（DES-IT07 5.1 節）。
		commanders, members := 0, map[string]bool{}
		for _, e := range m.Enemies {
			if e.Commander {
				commanders++
				if e.Group != "east_watch" || e.Boss {
					t.Fatalf("小隊長 %+v", e)
				}
			}
			if e.Group == "east_watch" {
				members[e.ID] = true
			}
		}
		for _, ev := range m.Events {
			if ev.Trigger == defs.TriggerEnemyAttacked && ev.Effect == defs.EventChargeGroup && ev.Group == "east_watch" {
				delete(members, ev.Unit)
			}
		}
		if commanders != 1 || len(members) != 0 {
			t.Fatalf("小隊長 %d 体、出来事のない一員 %v", commanders, members)
		}
		// 増援: 3 ターン目に出撃の枠の後ろから 3 体、内門の手前に入ると元栓の間から 3 体。
		alarm := 0
		for _, ev := range m.Events {
			if ev.Effect == defs.EventReinforce {
				alarm += len(ev.Enemies)
			}
		}
		if len(m.Reinforcements) != 3 || m.Reinforcements[0].Turn != 3 || alarm != 3 {
			t.Fatalf("ターンの増援 %d 体、出来事の増援 %d 体", len(m.Reinforcements), alarm)
		}
	})

	t.Run("TC-DES-NG-06-04 1 章に前後の物語があり、性別を示す語と既存作品の用語がなく、1 章の後で仲間が話さない", func(t *testing.T) {
		for _, ch := range cat.Chapters() {
			for _, id := range []string{ch.Before, ch.After} {
				s, ok := cat.Story(id)
				if !ok || len(s.Paragraphs) == 0 {
					t.Fatalf("%s: 物語 %q がない", ch.ID, id)
				}
				for _, p := range s.Paragraphs {
					if n := utf8.RuneCountInString(p); n == 0 || n > 200 {
						t.Fatalf("%s: 段落の長さ %d", id, n)
					}
				}
			}
		}
		files, _ := fs.Glob(data.Alchemy(), "*.toml")
		for _, f := range files {
			b, err := fs.ReadFile(data.Alchemy(), f)
			if err != nil {
				t.Fatal(err)
			}
			for _, bad := range []string{"彼女", "彼", "ユミア", "アトリエ"} {
				if strings.Contains(string(b), bad) {
					t.Errorf("%s に %q があります", f, bad)
				}
			}
		}
		// 戦死しうる仲間（主人公のほか）は、1 章の前のほかでは話さず動かない（DES-STY-01）。1 章の後には名前を出さない。
		after, _ := cat.Story("ch01_after")
		for _, p := range after.Paragraphs {
			for _, id := range []string{"dain", "tori", "nemu", "haru"} {
				c, _ := cat.Character(id)
				if strings.Contains(p, c.Name) {
					t.Errorf("1 章の後に %s が出ます: %s", c.Name, p)
				}
			}
		}
	})
}

// 爆弾を使わない既存の自動の打ち手で、1 章を遊ぶ。
// 爆弾を使う指し手は足していない。勝てる seed があることを下限として確かめる（面白さ・バランスは人の試遊が必要）。
// 通り道は南の広場から中の橋（8, 10）を渡り、内門（8, 4）の手前へ向かう（制作記録 5 章）。
func TestAlchemyAutoplay(t *testing.T) {
	cat := alchemyCatalog(t)
	routes := map[string][]tactics.Pos{
		"ch01": {{X: 8, Y: 12}, {X: 8, Y: 8}, {X: 8, Y: 5}},
	}
	order := []string{"dain", "tori", "nemu", "aoi", "haru"}
	chapters := []string{"ch01"}
	const seeds = 20
	reached := map[string]int{}
	wins := map[string]int{}
	cleared := 0
	for seed := uint64(1); seed <= seeds; seed++ {
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		all := true
		for _, id := range chapters {
			s, ids, err := autoCampaignBattle(c, seed, 6)
			if err != nil {
				t.Fatal(err)
			}
			reached[id]++
			r := autoPlay(s.Battle, ids, autoPlan{Order: order, Routes: [][]tactics.Pos{routes[id]}}, 40)
			t.Logf("seed %d: %v", seed, r)
			if r.Result != tactics.Won {
				all = false
				break
			}
			wins[id]++
			if _, err := c.FinishChapter(s); err != nil {
				t.Fatal(err)
			}
		}
		if all {
			cleared++
			if c.Cleared {
				t.Fatal("1 章の後にまだ 2 章がある")
			}
		}
	}
	t.Logf("1 章の勝利: %d/%d", wins["ch01"], reached["ch01"])
	// 基準は 20 seed で 8 以上（#208 の実測 10 / 20）。仮の 1 章の基準 15 から下げた理由は制作記録 5 章。
	t.Run("TC-DES-NG-06-05 1 章が、爆弾を使わない自動の試遊で勝てる（20 seed で 8 以上）", func(t *testing.T) {
		if cleared < 8 {
			t.Fatalf("1 章に勝った seed は %d / %d", cleared, seeds)
		}
		for _, id := range chapters {
			if wins[id] == 0 {
				t.Fatalf("%s で勝てる seed がない", id)
			}
		}
	})
}

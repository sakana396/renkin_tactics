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

// 新作の試作 1〜3 章の定義データ（data/alchemy。DES-NG-06）の検査。
// 期待値は設計メモ DES-NG01 6 章の決定（敵の HP・爆弾の数値・配置）から手で数えた。

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

	t.Run("TC-DES-NG-06-01 完結したデータ一式として読め、3 章・主人公 1 人・加入が決まりどおり", func(t *testing.T) {
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
		if got := cat.Chapters()[1].Joins; len(got) != 1 || got[0] != "haru" {
			t.Fatal(got)
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
		// 設計メモ 6.3 節の表: 1 章 回復薬 2・小さな爆弾 3・投げ火 1、2 章 回復薬 2・大きな爆弾 2、3 章 投げ火 1・小さな爆弾 2。
		wantReward := map[string]map[string]int{
			"ch01": {"potion": 2, "bomb_small": 3, "bomb_far": 1},
			"ch02": {"potion": 2, "bomb_large": 2},
			"ch03": {"bomb_far": 1, "bomb_small": 2},
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

	t.Run("TC-DES-NG-06-03 マップは 16×16・ボス 1 体・枠が出撃人数以上で、1 章の通路に敵が密集する", func(t *testing.T) {
		for i, id := range []string{"ch01", "ch02", "ch03"} {
			m, ok := cat.Map(id)
			if !ok || len(m.Rows) != 16 || len([]rune(m.Rows[0])) != 16 {
				t.Fatalf("%s: マップの大きさ", id)
			}
			bosses := 0
			for _, e := range m.Enemies {
				if e.Boss {
					bosses++
				}
			}
			// 出撃できるのは名簿の人数（ch01 は 4 人、ch02・ch03 は 5 人）。
			roster := 4 + min(i, 1)
			if bosses != 1 || len(m.Slots) < roster || len(m.Reinforcements) != 0 || len(m.Events) != 0 {
				t.Fatalf("%s: ボス %d 体、枠 %d、増援 %d、出来事 %d", id, bosses, len(m.Slots), len(m.Reinforcements), len(m.Events))
			}
		}
		// 1 章: 大きな爆弾（半径 2）を (6, 5) へ、(6, 8) から投げる（距離 3 で、半径の外）。
		// 範囲の中の敵は、HP が爆弾の固定ダメージ 12 以下なら 1 発で倒れる。
		m, _ := cat.Map("ch01")
		large, _ := cat.Item("bomb_large")
		center := tactics.Pos{X: 6, Y: 5}
		thrower := tactics.Pos{X: 6, Y: 8}
		if d := abs(center.X-thrower.X) + abs(center.Y-thrower.Y); d != large.ThrowRange || d <= large.BlastRadius {
			t.Fatal("投げる位置が射程の端・半径の外でない", d)
		}
		if row := []rune(m.Rows[thrower.Y]); row[thrower.X] != '.' {
			t.Fatalf("投げる位置の地形 %q", row[thrower.X])
		}
		inside, killed := 0, 0
		for _, e := range m.Enemies {
			if abs(center.X-e.X)+abs(center.Y-e.Y) > large.BlastRadius {
				continue
			}
			inside++
			cls, _ := cat.Class(e.Class)
			if cls.Base.HP+e.Bonus.HP <= large.BlastDamage {
				killed++
			}
		}
		if inside < 5 || killed < 5 {
			t.Fatalf("爆風の中の敵 %d 体、うち 1 発で倒れる敵 %d 体（5 体以上のはず）", inside, killed)
		}
	})

	t.Run("TC-DES-NG-06-04 各章に前後の物語があり、1 章の前は長い。性別を示す語と既存作品の用語がない", func(t *testing.T) {
		stories := map[string]int{}
		for _, ch := range cat.Chapters() {
			for _, id := range []string{ch.Before, ch.After} {
				s, ok := cat.Story(id)
				if !ok || len(s.Paragraphs) == 0 {
					t.Fatalf("%s: 物語 %q がない", ch.ID, id)
				}
				stories[id] = len(s.Paragraphs)
				for _, p := range s.Paragraphs {
					if n := utf8.RuneCountInString(p); n == 0 || n > 200 {
						t.Fatalf("%s: 段落の長さ %d", id, n)
					}
				}
			}
		}
		// 1 章の前（到着の場面）は、ほかのどの物語よりも長い。
		for id, n := range stories {
			if id != "ch01_before" && n >= stories["ch01_before"] {
				t.Fatalf("ch01_before（%d 段落）より %s（%d 段落）が長い", stories["ch01_before"], id, n)
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
	})
}

// 爆弾を使わない既存の自動の打ち手で、1〜3 章を通しで遊ぶ（名簿・成長・報酬は章から章へ引き継ぐ）。
// 爆弾を使う指し手は足していない。勝てる seed があることを下限として確かめる（面白さ・バランスは人の試遊が必要）。
func TestAlchemyAutoplay(t *testing.T) {
	cat := alchemyCatalog(t)
	routes := map[string][]tactics.Pos{
		"ch01": {{X: 6, Y: 11}, {X: 6, Y: 8}},
		"ch02": {{X: 4, Y: 8}, {X: 9, Y: 8}},
		"ch03": {{X: 3, Y: 10}, {X: 4, Y: 7}, {X: 7, Y: 6}},
	}
	order := []string{"dain", "tori", "nemu", "aoi", "haru"}
	chapters := []string{"ch01", "ch02", "ch03"}
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
			if !c.Cleared {
				t.Fatal("3 章を勝ったのにクリアにならない")
			}
		}
	}
	t.Logf("20 seed の通し: 章ごとの勝ち（到達した seed 数中）ch01 %d/%d、ch02 %d/%d、ch03 %d/%d。3 章とも勝った seed %d", wins["ch01"], reached["ch01"], wins["ch02"], reached["ch02"], wins["ch03"], reached["ch03"], cleared)
	t.Run("TC-DES-NG-06-05 3 章すべてが、爆弾を使わない自動の試遊で勝てる（20 seed の通しの 15 以上）", func(t *testing.T) {
		if cleared < 15 {
			t.Fatalf("3 章とも勝った seed は %d / %d", cleared, seeds)
		}
		for _, id := range chapters {
			if wins[id] == 0 {
				t.Fatalf("%s で勝てる seed がない", id)
			}
		}
	})
}

package defload_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体のデータのうち、反復 2 で足した・変えたもの（章・物語・2 章と 3 章のマップ・役割の数値）が、
// 設計書（反復 2）10 章の値になっていることを確かめる。
func TestBaseDataIt02(t *testing.T) {
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatalf("本体のデータに誤りがある:\n%v", err)
	}
	statsOf := func(class string, bonus defs.Stats) defs.Stats {
		c, ok := cat.Class(class)
		if !ok {
			t.Fatalf("クラス %s がない", class)
		}
		return addStats(c.Base, bonus)
	}

	t.Run("TC-FR-SYS-03-06 2 章・3 章のマップの出撃の枠・敵・増援が 10.5・10.6 節のとおり", func(t *testing.T) {
		type enemy struct {
			name, class, weapon, ai string
			boss                    bool
			x, y                    int
		}
		type reinf struct {
			turn         int
			class, weapn string
			x, y         int
		}
		tests := []struct {
			id, name  string
			slots     []defs.Slot
			bonus     defs.Stats // ボスを除く敵と増援のボーナス（10.4 節）
			enemies   []enemy
			bossBonus defs.Stats
			bossStats defs.Stats
			reinf     []reinf
		}{
			{"ch02", "灰の街道",
				[]defs.Slot{{X: 1, Y: 7}, {X: 1, Y: 6}, {X: 1, Y: 8}, {X: 0, Y: 6}, {X: 0, Y: 8}, {X: 2, Y: 7}},
				defs.Stats{HP: 2, Str: 1, Skl: 1, Spd: 1, Def: 1},
				[]enemy{
					{"", "soldier", "iron_lance", "charge", false, 8, 5},
					{"", "fighter", "iron_axe", "charge", false, 7, 9},
					{"", "archer", "iron_bow", "hold", false, 9, 3},
					{"", "myrmidon", "iron_sword", "hold", false, 10, 7},
					{"", "cavalier", "iron_lance", "hold", false, 12, 11},
					{"", "soldier", "iron_lance", "hold", false, 12, 6},
					{"", "archer", "iron_bow", "hold", false, 13, 9},
					{"ヴォルグ", "cavalier", "iron_lance", "hold", true, 14, 8},
				},
				defs.Stats{HP: 8, Str: 2, Skl: 3, Spd: 2, Def: 1},
				defs.Stats{HP: 28, Str: 8, Skl: 8, Spd: 8, Def: 6, Mov: 7},
				[]reinf{{2, "soldier", "iron_lance", 0, 4}, {3, "fighter", "iron_axe", 0, 10}, {4, "cavalier", "iron_lance", 0, 7}},
			},
			{"ch03", "王都の奪還",
				[]defs.Slot{{X: 7, Y: 14}, {X: 6, Y: 15}, {X: 8, Y: 15}, {X: 7, Y: 15}, {X: 8, Y: 14}},
				defs.Stats{HP: 3, Str: 2, Skl: 1, Spd: 1, Def: 1},
				[]enemy{
					{"", "soldier", "iron_lance", "hold", false, 4, 8},
					{"", "archer", "iron_bow", "hold", false, 11, 9},
					{"", "fighter", "iron_axe", "charge", false, 12, 8},
					{"", "myrmidon", "iron_sword", "hold", false, 4, 6},
					{"", "soldier", "iron_lance", "hold", false, 11, 6},
					{"", "archer", "iron_bow", "hold", false, 7, 5},
					{"", "fighter", "iron_axe", "hold", false, 10, 5},
					{"", "armor", "iron_lance", "hold", false, 6, 3},
					{"", "soldier", "iron_lance", "hold", false, 11, 3},
					{"グラウス", "myrmidon", "iron_sword", "hold", true, 8, 1},
				},
				defs.Stats{HP: 10, Str: 4, Skl: 3, Spd: 2, Def: 3},
				defs.Stats{HP: 27, Str: 8, Skl: 11, Spd: 11, Def: 5, Mov: 5},
				[]reinf{{3, "cavalier", "iron_lance", 0, 14}, {3, "cavalier", "iron_lance", 15, 14}, {5, "fighter", "iron_axe", 4, 2}},
			},
		}
		for _, tt := range tests {
			m, ok := cat.Map(tt.id)
			if !ok {
				t.Fatalf("マップ %s がない", tt.id)
			}
			if m.Name != tt.name {
				t.Errorf("%s の名前 = %q", tt.id, m.Name)
			}
			if !reflect.DeepEqual(m.Slots, tt.slots) {
				t.Errorf("%s の出撃の枠 = %+v", tt.id, m.Slots)
			}
			var want []defs.EnemyPlacement
			for _, e := range tt.enemies {
				b := tt.bonus
				if e.boss {
					b = tt.bossBonus
				}
				want = append(want, defs.EnemyPlacement{Name: e.name, Class: e.class, Weapon: e.weapon, Bonus: b, AI: e.ai, Boss: e.boss, X: e.x, Y: e.y})
			}
			if !reflect.DeepEqual(m.Enemies, want) {
				t.Errorf("%s の敵 =\n%+v\nwant\n%+v", tt.id, m.Enemies, want)
			}
			boss := m.Enemies[len(m.Enemies)-1]
			if s := statsOf(boss.Class, boss.Bonus); s != tt.bossStats {
				t.Errorf("%s のボスの能力 = %+v、期待 %+v", tt.id, s, tt.bossStats)
			}
			var wantR []defs.Reinforcement
			for _, r := range tt.reinf {
				wantR = append(wantR, defs.Reinforcement{Turn: r.turn, EnemyPlacement: defs.EnemyPlacement{Class: r.class, Weapon: r.weapn, Bonus: tt.bonus, AI: "charge", X: r.x, Y: r.y}})
			}
			if !reflect.DeepEqual(m.Reinforcements, wantR) {
				t.Errorf("%s の増援 =\n%+v\nwant\n%+v", tt.id, m.Reinforcements, wantR)
			}
		}
		// 3 章の門のアーマーの能力は 25/8/4/3/10/4（10.6 節）。
		m, _ := cat.Map("ch03")
		if a := m.Enemies[7]; statsOf(a.Class, a.Bonus) != (defs.Stats{HP: 25, Str: 8, Skl: 4, Spd: 3, Def: 10, Mov: 4}) {
			t.Errorf("3 章のアーマーの能力 = %+v", statsOf(a.Class, a.Bonus))
		}
	})

	t.Run("TC-IF-03-03 2 章・3 章のマップの地形は 10.5・10.6 節の図と同じで、ボスは砦の上", func(t *testing.T) {
		rows := map[string][]string{
			"ch02": {
				"^^^^^FF^^^^^^^^^",
				"^^^....F...^^^^^",
				"^^..FF......^^^^",
				"^...FF..^^....^^",
				"......^^^^..FF.^",
				"..FF..........F.",
				".....FF...^^....",
				"................",
				"...........FF.T.",
				"..^^..FF........",
				"..^^.....^^^..FF",
				".......FF^^.....",
				"^^...........^^^",
				"^^^^..FF..^^^^^^",
				"^^^^^.....^^^^^^",
				"^^^^^^^^^^^^^^^^",
			},
			"ch03": {
				"^^^############^",
				"^^^#....T.....#^",
				"^^^#..........#^",
				"^^^###.####.###^",
				"^^..F........F^^",
				"^...F..T..T..F.^",
				"................",
				"####.######.####",
				"..F............F",
				"..FF...^^...FF..",
				"......^^^^......",
				"....F......F....",
				"FF............FF",
				"FF..^^....^^..FF",
				"................",
				"................",
			},
		}
		forts := map[string][][2]int{
			"ch02": {{14, 8}},                 // ボス
			"ch03": {{8, 1}, {7, 5}, {10, 5}}, // ボスと、砦の上のアーチャー・ファイター
		}
		for id, want := range rows {
			m, _ := cat.Map(id)
			if !reflect.DeepEqual(m.Rows, want) {
				t.Errorf("%s の行 = %q", id, m.Rows)
			}
			for _, p := range forts[id] {
				if got := m.Rows[p[1]][p[0]]; got != 'T' {
					t.Errorf("%s の (%d, %d) の文字 = %q（砦のはず）", id, p[0], p[1], got)
				}
			}
		}
		// 2 章のアーチャー (9, 3) は山の上。
		m, _ := cat.Map("ch02")
		if got := m.Rows[3][9]; got != '^' {
			t.Errorf("ch02 の (9, 3) の文字 = %q（山のはず）", got)
		}
	})

	t.Run("TC-FR-CMP-01-01 章は 3 つで、章の順・名前・マップ・出撃の枠の数・報酬・ボスが 10.4 節のとおり", func(t *testing.T) {
		potion2 := []defs.ItemCount{{Item: "potion", Count: 2}}
		want := []struct {
			id, name, mapID string
			slots           int
			reward          []defs.ItemCount
			boss            string
		}{
			{"ch01", "国境の砦", "ch01", 5, potion2, "ハーゲン"},
			{"ch02", "灰の街道", "ch02", 6, potion2, "ヴォルグ"},
			{"ch03", "王都の奪還", "ch03", 5, nil, "グラウス"},
		}
		chapters := cat.Chapters()
		if len(chapters) != len(want) {
			t.Fatalf("章の数 = %d", len(chapters))
		}
		for i, w := range want {
			c := chapters[i]
			if c.ID != w.id || c.Name != w.name || c.Map != w.mapID || !reflect.DeepEqual(c.Reward, w.reward) {
				t.Errorf("章 %d = %+v", i+1, c)
			}
			m, ok := cat.Map(c.Map)
			if !ok {
				t.Fatalf("章 %s のマップ %s がない", c.ID, c.Map)
			}
			if len(m.Slots) != w.slots {
				t.Errorf("章 %s の出撃の枠の数 = %d、期待 %d", c.ID, len(m.Slots), w.slots)
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
		// 在庫は空から始まる（10.4 節）。
		if cat.Game.Stock != nil {
			t.Errorf("初めの在庫 = %+v", cat.Game.Stock)
		}
	})

	t.Run("TC-FR-CMP-07-01 各章の前後に物語があり、段落は 10.8 節の下書きのとおりで 200 文字以下", func(t *testing.T) {
		counts := map[string]int{"ch01_before": 5, "ch01_after": 5, "ch02_before": 5, "ch02_after": 4, "ch03_before": 4, "ch03_after": 4}
		for _, c := range cat.Chapters() {
			for _, id := range []string{c.Before, c.After} {
				if id != c.ID+"_before" && id != c.ID+"_after" {
					t.Errorf("章 %s の物語の ID %q が、章の ID に _before・_after を付けたものでない", c.ID, id)
				}
				s, ok := cat.Story(id)
				if !ok {
					t.Errorf("物語 %s がない", id)
					continue
				}
				if len(s.Paragraphs) != counts[id] {
					t.Errorf("物語 %s の段落の数 = %d、期待 %d", id, len(s.Paragraphs), counts[id])
				}
				for j, p := range s.Paragraphs {
					if p == "" || utf8.RuneCountInString(p) > 200 {
						t.Errorf("物語 %s の段落 %d の長さ = %d", id, j, utf8.RuneCountInString(p))
					}
				}
			}
		}
		// 最初と最後の段落、加入の場面のエルザの台詞（10.8 節）。
		spot := []struct {
			id   string
			i    int
			want string
		}{
			{"ch01_before", 0, "小国アルケアは、錬金術の工房で知られる国だった。山と森に守られ、長く戦を知らなかった。"},
			{"ch01_before", 4, "レオン「行こう。守将ハーゲンを討てば、先遣隊は崩れる」"},
			{"ch02_before", 2, "エルザ「村を焼けという命令に、従えなかった。この戦は、錬金術の技を奪うための略奪だ。剣を向ける相手を、選び直したい」"},
			{"ch03_after", 3, "帝国との戦がこれで終わるのかは、まだ誰にも分からない。それでも、工房の煙突から、久しぶりに煙が上がった。"},
		}
		for _, sp := range spot {
			s, _ := cat.Story(sp.id)
			if sp.i >= len(s.Paragraphs) || s.Paragraphs[sp.i] != sp.want {
				t.Errorf("物語 %s の段落 %d = %q", sp.id, sp.i, s.Paragraphs)
			}
		}
	})

	t.Run("TC-FR-CMP-08-01 エルザは名簿になく、2 章の始めに加わる（レベル 2）", func(t *testing.T) {
		var joins [][]string
		for _, c := range cat.Chapters() {
			joins = append(joins, c.Joins)
		}
		if !reflect.DeepEqual(joins, [][]string{nil, {"elsa"}, nil}) {
			t.Errorf("加入 = %q", joins)
		}
		for _, id := range cat.Game.Party {
			if id == "elsa" {
				t.Error("エルザが初めの名簿にいる")
			}
		}
		if ch, ok := cat.Character("elsa"); !ok || ch.Level != 2 || ch.Class != "cavalier" || ch.Lord {
			t.Errorf("エルザ = %+v, %v", ch, ok)
		}
	})

	t.Run("TC-FR-UNT-03-01 前衛の役割が 10.3・10.9 節のとおり尖っている（ロードは技・速さ、ファイターは力、アーマーは守備）", func(t *testing.T) {
		st := map[string]defs.Stats{}
		gr := map[string]defs.Growth{}
		for _, id := range []string{"leon", "gald", "bran"} {
			ch, _ := cat.Character(id)
			st[id] = statsOf(ch.Class, ch.Bonus)
			gr[id] = ch.Growth
		}
		l, g, b := st["leon"], st["gald"], st["bran"]
		// ロード: 技と速さが前衛で最も高く、HP が最も低い（高命中・高速・紙耐久）。
		if !(l.Skl > b.Skl && l.Skl > g.Skl && l.Spd > b.Spd && l.Spd > g.Spd && l.HP < b.HP && l.HP < g.HP) {
			t.Errorf("ロードの能力 %+v（ファイター %+v、アーマー %+v）", l, b, g)
		}
		// ファイター: 力が最も高く、守備が最も低い（高火力・低耐久）。
		if !(b.Str > l.Str && b.Str > g.Str && b.Def < l.Def && b.Def < g.Def) {
			t.Errorf("ファイターの能力 %+v", b)
		}
		// アーマー: 守備が最も高く、技が最も低い（守備が最強・命中が不安）。火力は中のまま（力 6）。
		if !(g.Def > l.Def && g.Def > b.Def && g.Skl < l.Skl && g.Skl < b.Skl && g.Str == 6) {
			t.Errorf("アーマーの能力 %+v", g)
		}
		// 成長率も同じ向きに尖らせる（10.3 節）。
		lg, gg, bg := gr["leon"], gr["gald"], gr["bran"]
		if !(lg.Skl > gg.Skl && lg.Skl > bg.Skl && lg.Spd > gg.Spd && lg.Spd > bg.Spd && bg.Str > lg.Str && bg.Str > gg.Str && gg.Def > lg.Def && gg.Def > bg.Def && gg.HP > lg.HP && gg.HP > bg.HP) {
			t.Errorf("成長率 レオン %+v、ガルド %+v、ブラン %+v", lg, gg, bg)
		}
	})

	t.Run("TC-FR-SYS-03-07 目録の章は定義の順の写しで、ない ID の物語は「ない」と分かる", func(t *testing.T) {
		cs := cat.Chapters()
		cs[1].Joins[0] = "nobody"
		cs[0].ID = "changed"
		again := cat.Chapters()
		if again[0].ID != "ch01" || again[1].Joins[0] != "elsa" {
			t.Errorf("返した章を書き換えると目録が変わった: %+v", again)
		}
		for _, id := range []string{"ch99_before", "", "ch01"} {
			if _, ok := cat.Story(id); ok {
				t.Errorf("ない物語 %q が見つかった", id)
			}
		}
		if s, ok := cat.Story("ch02_after"); !ok || len(s.Paragraphs) != 4 {
			t.Errorf("Story(ch02_after) = %+v, %v", s, ok)
		}
	})
}

// 省略できるキーを省略したときの既定値を確かめる（DES-DAT-01、DES-DAT-03 規則 16、17、19）。
func TestLoadDefaults(t *testing.T) {
	t.Run("TC-FR-SYS-03-08 level・growth（とその中の能力）・stock・敵の name・章の before・joins・reward を省略すると既定値になる", func(t *testing.T) {
		m := baseFS(t)
		for _, e := range []edit{
			// ミラの成長率を hp だけにし、ブランの成長率を消す。
			replace("characters.toml", "growth = { hp = 55, str = 40, skl = 60, spd = 45, def = 20 }", "growth = { hp = 50 }"),
			replace("characters.toml", "growth = { hp = 70, str = 75, skl = 35, spd = 30, def = 15 }\n", ""),
			replace("chapters.toml", `before = "ch03_before"`+"\n", ""),
		} {
			e(t, m)
		}
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatalf("誤りがある:\n%v", err)
		}
		if ch, _ := cat.Character("mira"); ch.Level != 1 || ch.Growth != (defs.Growth{HP: 50}) {
			t.Errorf("ミラ: レベル %d、成長率 %+v", ch.Level, ch.Growth)
		}
		if ch, _ := cat.Character("bran"); ch.Level != 1 || ch.Growth != (defs.Growth{}) {
			t.Errorf("ブラン: レベル %d、成長率 %+v", ch.Level, ch.Growth)
		}
		if len(cat.Game.Stock) != 0 {
			t.Errorf("在庫 = %+v", cat.Game.Stock)
		}
		ch01, _ := cat.Map("ch01")
		if ch01.Enemies[0].Name != "" || ch01.Reinforcements[0].Name != "" {
			t.Errorf("名前を書かない敵の名前 = %q、%q", ch01.Enemies[0].Name, ch01.Reinforcements[0].Name)
		}
		c := cat.Chapters()[2]
		if c.Before != "" || c.After != "ch03_after" || c.Joins != nil || c.Reward != nil {
			t.Errorf("3 章 = %+v", c)
		}
	})
}

package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// ch07「補給の荷駄」・ch08「黒い森」（DES-MAP-06、DES-CMP-09。制作記録 docs/iterations/it05/map-ch07.md・map-ch08.md）。

// 章制作台帳の人数・寸法・地形・配置と、道の作りを確かめる（DES-MAP-06、DES-IT05 6 章）。
func TestCh07To08Maps(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                                   string
		size, slots, enemies, reinforcements int
		boss                                 string
		bossAt                               defs.Slot
	}{
		{"ch07", 16, 5, 10, 4, "ベルント", defs.Slot{X: 11, Y: 1}},
		{"ch08", 24, 5, 16, 6, "クラウス", defs.Slot{X: 12, Y: 1}},
	} {
		t.Run("TC-FR-MAP-04-05 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("マップがない")
			}
			if len(m.Rows) != tc.size || len(m.Slots) != tc.slots || len(m.Enemies) != tc.enemies || len(m.Reinforcements) != tc.reinforcements {
				t.Fatalf("制作台帳と不一致: 行 %d 枠 %d 敵 %d 増援 %d", len(m.Rows), len(m.Slots), len(m.Enemies), len(m.Reinforcements))
			}
			for y, r := range m.Rows {
				if n := len([]rune(r)); n != tc.size {
					t.Fatalf("行 %d の幅 = %d", y, n)
				}
			}
			checkCh07To08Definition(t, m)
			// ボスはちょうど 1 体で、陣の砦の上。
			var bosses []defs.EnemyPlacement
			for _, e := range m.Enemies {
				if e.Boss {
					bosses = append(bosses, e)
				}
			}
			if len(bosses) != 1 || bosses[0].Name != tc.boss || bosses[0].X != tc.bossAt.X || bosses[0].Y != tc.bossAt.Y || m.Rows[tc.bossAt.Y][tc.bossAt.X] != 'T' {
				t.Errorf("ボス = %+v", bosses)
			}
			// 全ての枠・敵・増援の位置へ、主人公の枠から地形だけで行ける。増援はすべて突っ込む（ボスでない）。
			all := reachableByTerrain(m, cat, m.Slots[0], nil)
			for _, p := range m.Slots {
				if !all[p] {
					t.Errorf("枠 %v へ行けない", p)
				}
			}
			for _, e := range m.Enemies {
				if !all[defs.Slot{X: e.X, Y: e.Y}] {
					t.Errorf("敵 %+v へ行けない", e)
				}
			}
			for _, r := range m.Reinforcements {
				if !all[defs.Slot{X: r.X, Y: r.Y}] || r.Boss || r.AI != "charge" {
					t.Errorf("増援 %+v", r)
				}
			}
			if tc.id == "ch07" {
				checkCh07Routes(t, m, cat)
			} else {
				checkCh08Forest(t, m)
			}
		})
	}
}

// checkCh07Routes は、岩山（y=4〜10）を抜ける道が、谷の近道・東の迂回・二つの岩棚だけであることを確かめる（制作記録 2 章）。
func checkCh07Routes(t *testing.T, m defs.MapDef, cat *defs.Catalog) {
	t.Helper()
	for y := 4; y <= 10; y++ {
		var open []int
		for x, r := range m.Rows[y] {
			if r != '#' {
				open = append(open, x)
			}
		}
		if !reflect.DeepEqual(open, []int{5, 7, 8, 10, 13, 14, 15}) {
			t.Errorf("岩山の行 %d の通れるマス = %v", y, open)
		}
	}
	// 谷は平地、岩棚は山。岩棚の弓兵は谷を射程 2 で狙う。
	for y := 4; y <= 10; y++ {
		if m.Rows[y][7] != '.' || m.Rows[y][8] != '.' || m.Rows[y][5] != '^' || m.Rows[y][10] != '^' {
			t.Errorf("谷・岩棚の地形（行 %d）= %q", y, m.Rows[y])
		}
	}
	closeAt := func(xs ...int) map[defs.Slot]bool {
		out := map[defs.Slot]bool{}
		for _, x := range xs {
			out[defs.Slot{X: x, Y: 7}] = true
		}
		return out
	}
	camp := defs.Slot{X: 11, Y: 2}
	for _, c := range []struct {
		name   string
		closed map[defs.Slot]bool
		want   bool
	}{
		{"谷を閉じる（東の迂回）", closeAt(7, 8), true},
		{"東の迂回を閉じる（谷の近道）", closeAt(13, 14, 15), true},
		{"谷と迂回を閉じる（岩棚だけ）", closeAt(7, 8, 13, 14, 15), true},
		{"全部を閉じる", closeAt(5, 7, 8, 10, 13, 14, 15), false},
	} {
		if got := reachableByTerrain(m, cat, m.Slots[0], c.closed)[camp]; got != c.want {
			t.Errorf("%s: 陣へ着く = %v", c.name, got)
		}
	}
}

// checkCh08Forest は、森が盤面の 4 割以上（265 マス）を占め、中央の森の道（x=11・12、y=5〜16）が平地で続くことを確かめる（制作記録 2 章）。
func checkCh08Forest(t *testing.T, m defs.MapDef) {
	t.Helper()
	forest := 0
	for _, r := range m.Rows {
		forest += strings.Count(r, "F")
	}
	if forest*10 < 24*24*4 {
		t.Errorf("森のマス = %d", forest)
	}
	for y := 5; y <= 16; y++ {
		if m.Rows[y][11] != '.' || m.Rows[y][12] != '.' {
			t.Errorf("森の道（行 %d）= %q", y, m.Rows[y])
		}
	}
	var forts []defs.Slot
	for y, r := range m.Rows {
		for x, c := range r {
			if c == 'T' {
				forts = append(forts, defs.Slot{X: x, Y: y})
			}
		}
	}
	if !reflect.DeepEqual(forts, []defs.Slot{{X: 12, Y: 1}, {X: 5, Y: 11}, {X: 18, Y: 11}}) {
		t.Errorf("砦 = %v", forts)
	}
}

// 試し遊びの入力は制作記録 4・5 章。敵・名簿・乱数は戦闘開始後に書き換えない（DES-MAP-06）。
func TestCh07To08Autoplay(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id   string
		plan int
		low  bool
		seed uint64
	}{
		{"ch07", 0, false, 4}, {"ch07", 1, false, 2}, {"ch07", 0, true, 3}, {"ch07", 1, true, 2},
		{"ch08", 0, false, 10}, {"ch08", 1, false, 1}, {"ch08", 2, false, 6}, {"ch08", 0, true, 1}, {"ch08", 1, true, 9},
	} {
		p := ch07To08Plans(tc.id)[tc.plan]
		t.Run(fmt.Sprintf("TC-FR-MAP-04-06 %s/%s/低め%v/seed%d", tc.id, p.name, tc.low, tc.seed), func(t *testing.T) {
			// IT06 で一時停止（TST-IT06 3.2・3.3 節）。ch07 は #132、ch08 は #133 で置き換える。
			if tc.id == "ch08" && ((tc.plan == 0 && !tc.low && tc.seed == 10) || (tc.plan == 1 && tc.low && tc.seed == 9)) {
				pauseIT06(t, "#133")
			}
			if tc.id == "ch07" && tc.plan == 0 && !tc.low && tc.seed == 4 {
				pauseIT06Level(t, "#132")
			}
			c := estimatedRoster(t, cat, tc.id, tc.low)
			s, ids, err := autoBattleWith(c, tc.seed, p.members)
			if err != nil {
				t.Fatal(err)
			}
			if n := len(s.Battle.Units()) - len(mustMap(t, cat, tc.id).Enemies); n != 5 {
				t.Fatalf("出撃 %d 人", n)
			}
			r := autoPlay(s.Battle, ids, p.plan, 40)
			// 攻撃（敵軍を含む）・杖・薬の行動がない番を「移動・待機だけ」の参考値として集計する。
			active := map[int]bool{}
			first := 0
			for _, h := range s.Battle.History() {
				if h.Kind == tactics.HistoryAction && h.Action != tactics.Wait {
					active[h.Turn] = true
				}
				if first == 0 && h.Kind == tactics.HistoryAction && h.Action == tactics.Attack {
					first = h.Turn
				}
			}
			var quiet []int
			for turn := 1; turn <= r.Turns; turn++ {
				if !active[turn] {
					quiet = append(quiet, turn)
				}
			}
			t.Logf("%v 初交戦=%d 移動・待機だけの番=%v", r, first, quiet)
			if r.Result != tactics.Won {
				t.Fatalf("勝てなかった: %v", r)
			}
			if first == 0 || first > 3 {
				t.Fatal("交戦まで歩くだけの区間が長い")
			}
			before := append([]defs.ItemCount(nil), c.Stock...)
			res, err := c.FinishChapter(s)
			next := map[string]string{"ch07": "ch08", "ch08": "ch09"}[tc.id]
			if err != nil || res.Chapter != tc.id || res.Cleared || c.Cleared || cat.Chapters()[c.Chapter].ID != next {
				t.Fatalf("勝利の反映 %+v: %v", res, err)
			}
			if !reflect.DeepEqual(res.Reward, []defs.ItemCount{{Item: "potion", Count: 2}, {Weapon: map[string]string{"ch07": "brave_axe", "ch08": "horse_killer"}[tc.id], Count: 1}}) || len(c.StockWeapons) == 0 || len(before) != 1 || !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: before[0].Count + 2}}) {
				t.Fatalf("報酬 %+v 在庫 %v", res.Reward, c.Stock)
			}
		})
	}
}

func mustMap(t *testing.T, cat *defs.Catalog, id string) defs.MapDef {
	t.Helper()
	m, ok := cat.Map(id)
	if !ok {
		t.Fatalf("マップ %s がない", id)
	}
	return m
}

// checkCh07To08Definition は制作時に決めた地形・配置・能力の期待値（DES-MAP-06、制作記録 2 章）。
// 章の寸法・人数は計画書 5 章、道と砦は設計書 6 章から導く。
func checkCh07To08Definition(t *testing.T, m defs.MapDef) {
	t.Helper()
	type reinforcement struct {
		turn  int
		class string
		x, y  int
	}
	type definition struct {
		rows           []string
		slots          []defs.Slot
		enemies        string
		reinforcements []reinforcement
		bonus          defs.Stats
	}
	want := map[string]definition{
		"ch07": {
			// 制作記録 2 章の全行。
			rows: []string{
				"^^^^..........^^",
				"^^^^..T....T..^^",
				"^^^^..........^^",
				"^^^^.^..........",
				"#####^#..#^##...",
				"#####^#..#^##.F.",
				"#####^#..#^##F..",
				"#####^#..#^##...",
				"#####^#..#^##.T.",
				"#####^#..#^##F..",
				"#####^#..#^##...",
				"...F............",
				"..F.........F...",
				"................",
				".F...........F..",
				"................",
			},
			slots:          []defs.Slot{{X: 7, Y: 13}, {X: 8, Y: 13}, {X: 6, Y: 14}, {X: 7, Y: 14}, {X: 8, Y: 14}},
			enemies:        "cavalier:14,12:charge myrmidon:7,9:hold archer:5,7:hold myrmidon:8,6:hold archer:10,6:hold armor:8,3:hold fighter:14,8:hold soldier:13,4:hold archer:6,1:hold soldier:11,1:hold",
			reinforcements: []reinforcement{{3, "cavalier", 0, 13}, {3, "fighter", 15, 15}, {5, "soldier", 15, 3}, {6, "myrmidon", 4, 0}},
			bonus:          defs.Stats{HP: 5, Str: 3, Skl: 2, Spd: 2, Def: 2},
		},
		"ch08": {
			// 制作記録 2 章の全行。
			rows: []string{
				"FFFFFFF..........FFFFFFF",
				"FFFFFF......T.....FFFFFF",
				"FFFFF..............FFFFF",
				"FFFF..F...FFFF...F..FFFF",
				"FFF....FFFFFFFFFF....FFF",
				"FF..FFFFFF....FFFFFF..FF",
				"F..FFFFFFFF..FFFFFFFF..F",
				"..FFF^FFFFF..FFFFF^FFF..",
				".FFFF^^FFFF..FFFF^^FFFF.",
				"FF.....FFFF..FFFF.....FF",
				"F.......FFF..FFF.......F",
				"F....T..FFF..FFF..T....F",
				"F.......FFF..FFF.......F",
				"FF.....FFFF..FFFF.....FF",
				"FFF.FFFFFFF..FFFFFFF.FFF",
				"FFFFFF^FFF....FFF^FFFFFF",
				"FFFFFFFFF......FFFFFFFFF",
				".FFFFFF............FFFF.",
				"..FFF.....F..F.......FF.",
				"...F.....FF....FF.....F.",
				"........................",
				"....F..............F....",
				".....FF..........FF.....",
				"^^^^................^^^^",
			},
			slots:          []defs.Slot{{X: 11, Y: 20}, {X: 12, Y: 20}, {X: 10, Y: 21}, {X: 11, Y: 21}, {X: 12, Y: 21}},
			enemies:        "myrmidon:10,11:charge fighter:13,10:charge cavalier:12,7:charge myrmidon:3,14:charge armor:5,11:hold archer:3,10:hold soldier:6,9:hold fighter:18,11:hold archer:20,12:hold myrmidon:17,9:hold archer:12,8:hold soldier:11,6:hold cavalier:8,3:hold armor:14,2:hold archer:15,1:hold myrmidon:12,1:hold",
			reinforcements: []reinforcement{{4, "fighter", 0, 7}, {4, "soldier", 0, 8}, {5, "cavalier", 23, 7}, {5, "cavalier", 23, 8}, {7, "myrmidon", 5, 23}, {7, "myrmidon", 18, 23}},
			bonus:          defs.Stats{HP: 5, Str: 3, Skl: 3, Spd: 2, Def: 3},
		},
	}[m.ID]
	if !reflect.DeepEqual(m.Rows, want.rows) {
		t.Errorf("制作記録 2 章と地形が不一致: got %q", m.Rows)
	}
	if !reflect.DeepEqual(m.Slots, want.slots) {
		t.Errorf("出撃枠 %v", m.Slots)
	}
	weapons := map[string]string{"soldier": "iron_lance", "armor": "iron_lance", "cavalier": "iron_lance", "fighter": "iron_axe", "archer": "iron_bow", "myrmidon": "iron_sword"}
	checkEnemy := func(e defs.EnemyPlacement) {
		b := want.bonus
		if e.Boss {
			// ボスは通常の敵に 7/2/2/1/2 を足す（ch04〜ch06 と同じ差）。
			b.HP += 7
			b.Str += 2
			b.Skl += 2
			b.Spd++
			b.Def += 2
		}
		if e.Bonus != b || len(e.Weapons) != 1 || e.Weapons[0] != weapons[e.Class] {
			t.Errorf("敵の能力・武器 %+v", e)
		}
	}
	var placements []string
	for _, e := range m.Enemies {
		placements = append(placements, fmt.Sprintf("%s:%d,%d:%s", e.Class, e.X, e.Y, e.AI))
		checkEnemy(e)
	}
	if strings.Join(placements, " ") != want.enemies {
		t.Errorf("敵配置 %v", placements)
	}
	var rf []reinforcement
	for _, r := range m.Reinforcements {
		rf = append(rf, reinforcement{r.Turn, r.Class, r.X, r.Y})
		checkEnemy(r.EnemyPlacement)
	}
	if !reflect.DeepEqual(rf, want.reinforcements) {
		t.Errorf("増援 %v", rf)
	}
}

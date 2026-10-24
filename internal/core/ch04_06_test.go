package core_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 章制作台帳の人数・寸法・二つの入口を確かめる（DES-MAP-06、DES-CMP-09）。
func TestCh04To06Maps(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                                   string
		size, slots, enemies, reinforcements int
		boss                                 string
		x                                    int
	}{
		{"ch04", 18, 6, 12, 3, "オスカー", 9}, {"ch05", 16, 5, 9, 2, "ディートリヒ", 8}, {"ch06", 20, 6, 14, 4, "ロルフ", 10},
	} {
		t.Run("TC-FR-MAP-04-03 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("マップがない")
			}
			if len(m.Rows) != tc.size || len(m.Slots) != tc.slots || len(m.Enemies) != tc.enemies || len(m.Reinforcements) != tc.reinforcements {
				t.Fatalf("制作台帳と不一致: %+v", m)
			}
			for _, r := range m.Rows {
				if len([]rune(r)) != tc.size {
					t.Fatal("幅が違う")
				}
			}
			checkCh04To06Definition(t, m)
			bosses := 0
			all := reachableByTerrain(m, cat, m.Slots[0], nil)
			for _, p := range m.Slots {
				if !all[p] {
					t.Errorf("枠 %v へ行けない", p)
				}
			}
			for _, e := range m.Enemies {
				if !all[defs.Slot{X: e.X, Y: e.Y}] {
					t.Errorf("敵 %v へ行けない", e)
				}
				if e.Boss {
					bosses++
					if e.Name != tc.boss || e.X != tc.x || e.Y != 2 || m.Rows[e.Y][e.X] != 'T' {
						t.Errorf("ボス %+v", e)
					}
				}
			}
			if bosses != 1 {
				t.Errorf("ボス数 %d", bosses)
			}
			for _, r := range m.Reinforcements {
				if !all[defs.Slot{X: r.X, Y: r.Y}] || r.Boss || r.AI != "charge" {
					t.Errorf("増援 %+v", r)
				}
			}
			if tc.id == "ch06" {
				for _, p := range []defs.Slot{{X: 4, Y: 13}, {X: 5, Y: 7}} {
					if m.Rows[p.Y][p.X] != '^' {
						t.Errorf("弓兵の山 %v がない", p)
					}
				}
				for _, p := range []defs.Slot{{X: 6, Y: 12}, {X: 13, Y: 6}} {
					if m.Rows[p.Y][p.X] != 'T' {
						t.Errorf("道中の砦 %v がない", p)
					}
				}
				return
			}
			gates := []int{4, 5, 12, 13}
			left := []int{4, 5}
			right := []int{12, 13}
			if tc.id == "ch05" {
				gates = []int{2, 7}
				left = []int{2}
				right = []int{7}
			}
			var got []int
			for x, r := range m.Rows[7] {
				if r != '#' {
					got = append(got, x)
				}
			}
			if !reflect.DeepEqual(got, gates) {
				t.Errorf("壁の切れ目 %v", got)
			}
			for i, closedX := range [][]int{left, right, gates} {
				closed := map[defs.Slot]bool{}
				for _, x := range closedX {
					closed[defs.Slot{X: x, Y: 7}] = true
				}
				if got := reachableByTerrain(m, cat, m.Slots[0], closed)[defs.Slot{X: tc.x, Y: 3}]; got != (i < 2) {
					t.Errorf("入口 %v を閉じた到達=%v", closedX, got)
				}
			}
		})
	}
}

// 新しい章でも主人公必須・人数上限・空き枠の規則を保つ（DES-CMP-09）。
func TestCh04To06Deployment(t *testing.T) {
	cat := baseCatalog(t)
	for _, id := range []string{"ch04", "ch05", "ch06"} {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			t.Run(fmt.Sprintf("TC-FR-CMP-03-12 %s/%v", id, rule), func(t *testing.T) {
				c, err := core.NewCampaignAt(cat, rule, id)
				if err != nil {
					t.Fatal(err)
				}
				if len(c.Roster) != 6 {
					t.Fatal("名簿は6人")
				}
				for _, tc := range []struct {
					members []int
					bad     bool
				}{{[]int{0}, false}, {[]int{1, 2}, true}, {nil, true}, {[]int{0, 1, 2, 3, 4, 5}, id == "ch05"}, {[]int{0, 1, 2, 3, 4, 5, 5}, true}, {[]int{0, 1, 2, 3, 5}, false}} {
					before := campaignSnapshot(c)
					s, _, err := c.StartBattle(tc.members, 1)
					if tc.bad {
						if !errors.Is(err, core.ErrInvalidDeployment) || s != nil {
							t.Fatalf("拒否されない %v: %v", tc.members, err)
						}
					} else if err != nil {
						t.Fatal(err)
					} else {
						m, _ := cat.Map(id)
						for i := range tc.members {
							u := s.Battle.Units()[i]
							if u.Pos.X != m.Slots[i].X || u.Pos.Y != m.Slots[i].Y {
								t.Fatal("枠が違う")
							}
						}
						if len(s.Battle.Units()) != len(tc.members)+len(m.Enemies) {
							t.Fatal("空き枠に自軍を追加した")
						}
					}
					if !reflect.DeepEqual(before, campaignSnapshot(c)) {
						t.Fatal("出撃で名簿を変えた")
					}
				}
			})
		}
	}
}

// 試し遊びの入力は制作記録 4・5 章。敵・名簿・乱数は戦闘開始後に書き換えない（DES-MAP-06）。
func TestCh04To06Autoplay(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id   string
		plan int
		low  bool
		seed uint64
	}{
		{"ch04", 0, false, 9}, {"ch04", 1, false, 2}, {"ch04", 0, true, 1}, {"ch04", 1, true, 1},
		{"ch05", 0, false, 1}, {"ch05", 1, false, 8}, {"ch05", 0, true, 4}, {"ch05", 1, true, 8},
		{"ch06", 0, false, 5}, {"ch06", 1, false, 12}, {"ch06", 0, true, 6}, {"ch06", 1, true, 6},
	} {
		t.Run(fmt.Sprintf("TC-FR-MAP-04-04 %s/作戦%d/低め%v/seed%d", tc.id, tc.plan, tc.low, tc.seed), func(t *testing.T) {
			c := ch04To06Roster(t, cat, tc.id, tc.low)
			slots := 6
			if tc.id == "ch05" {
				slots = 5
			}
			s, ids, err := autoCampaignBattle(c, tc.seed, slots)
			if err != nil {
				t.Fatal(err)
			}
			if tc.id == "ch05" && tc.plan == 1 {
				ch05Lure(t, s.Battle)
			}
			r := autoPlay(s.Battle, ids, ch04To06Plans(tc.id)[tc.plan], 40)
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
			next := map[string]string{"ch04": "ch05", "ch05": "ch06", "ch06": "ch09"}[tc.id]
			if err != nil || res.Chapter != tc.id || res.Cleared || c.Cleared || cat.Chapters()[c.Chapter].ID != next {
				t.Fatalf("勝利の反映 %+v: %v", res, err)
			}
			if !reflect.DeepEqual(res.Reward, []defs.ItemCount{{Item: "potion", Count: 2}}) || len(before) != 1 || !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: before[0].Count + 2}}) {
				t.Fatalf("報酬 %+v 在庫 %v", res.Reward, c.Stock)
			}
		})
	}
}

// ch04To06Definition は制作時に決めた配置・能力の期待値（DES-MAP-06、制作記録 2 章）。
// 章の寸法・人数は計画書 5 章、入口と交戦点は設計書 6 章から導く。
func checkCh04To06Definition(t *testing.T, m defs.MapDef) {
	t.Helper()
	type reinforcement struct {
		turn  int
		class string
		x, y  int
	}
	type definition struct {
		slots          []defs.Slot
		enemies        string
		reinforcements []reinforcement
		bonus          defs.Stats
	}
	want := map[string]definition{
		"ch04": {
			slots:          []defs.Slot{{X: 8, Y: 16}, {X: 9, Y: 16}, {X: 7, Y: 17}, {X: 8, Y: 17}, {X: 9, Y: 17}, {X: 10, Y: 17}},
			enemies:        "soldier:4,12:charge fighter:13,12:charge archer:3,10:hold myrmidon:5,9:hold soldier:12,9:hold cavalier:14,10:hold armor:5,6:hold archer:4,4:hold fighter:12,5:hold archer:13,4:hold myrmidon:9,4:hold armor:9,2:hold",
			reinforcements: []reinforcement{{3, "fighter", 0, 15}, {4, "soldier", 17, 14}, {6, "cavalier", 13, 1}},
			bonus:          defs.Stats{HP: 2, Str: 1, Skl: 1, Spd: 1, Def: 1},
		},
		"ch05": {
			slots:          []defs.Slot{{X: 7, Y: 12}, {X: 7, Y: 11}, {X: 6, Y: 12}, {X: 8, Y: 12}, {X: 7, Y: 13}},
			enemies:        "soldier:7,9:hold fighter:10,10:hold armor:7,6:hold archer:8,5:hold myrmidon:6,4:hold soldier:2,5:hold archer:3,3:hold fighter:10,4:hold soldier:8,2:hold",
			reinforcements: []reinforcement{{4, "myrmidon", 14, 12}, {6, "fighter", 1, 2}},
			bonus:          defs.Stats{HP: 3, Str: 2, Skl: 2, Spd: 1, Def: 1},
		},
		"ch06": {
			slots:          []defs.Slot{{X: 9, Y: 18}, {X: 10, Y: 18}, {X: 8, Y: 19}, {X: 9, Y: 19}, {X: 10, Y: 19}, {X: 11, Y: 19}},
			enemies:        "cavalier:11,14:charge soldier:6,15:charge archer:4,13:hold fighter:7,12:hold myrmidon:10,12:hold cavalier:14,10:hold archer:15,10:hold soldier:12,8:hold archer:5,7:hold armor:7,6:hold cavalier:13,5:hold fighter:10,4:hold archer:12,3:hold armor:10,2:hold",
			reinforcements: []reinforcement{{3, "cavalier", 19, 17}, {4, "fighter", 0, 14}, {6, "cavalier", 18, 7}, {7, "myrmidon", 3, 1}},
			bonus:          defs.Stats{HP: 4, Str: 2, Skl: 2, Spd: 2, Def: 2},
		},
	}[m.ID]
	if !reflect.DeepEqual(m.Slots, want.slots) {
		t.Errorf("出撃枠 %v", m.Slots)
	}
	weapons := map[string]string{"soldier": "iron_lance", "armor": "iron_lance", "cavalier": "iron_lance", "fighter": "iron_axe", "archer": "iron_bow", "myrmidon": "iron_sword"}
	checkEnemy := func(e defs.EnemyPlacement) {
		b := want.bonus
		if e.Boss {
			b.HP += 7
			b.Str += 2
			b.Skl += 2
			b.Spd++
			b.Def += 2
		}
		if e.Bonus != b || e.Weapon != weapons[e.Class] {
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

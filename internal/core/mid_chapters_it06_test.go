package core_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// ch06・ch07 の制作値を、出撃枠・敵の総数・後半の一団から確かめる（DES-MAP-08）。
func TestIT06MidChapters(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                          string
		slots, initial, turn, event int
		groups                      map[string]int
		cavaliers                   int
		frontCavalier               defs.Slot
	}{
		{"ch06", 7, 14, 4, 2, map[string]int{"pass_center": 3, "pass_east": 2}, 6, defs.Slot{}},
		{"ch07", 6, 10, 4, 2, map[string]int{"convoy_valley": 2, "convoy_east": 6}, 4, defs.Slot{X: 14, Y: 10}},
	} {
		t.Run("TC-FR-MAP-04-17 "+tc.id, func(t *testing.T) {
			m, ok := cat.Map(tc.id)
			if !ok {
				t.Fatal("地図がない")
			}
			if len(m.Slots) != tc.slots || len(m.Enemies) != tc.initial || len(m.Reinforcements) != tc.turn || len(m.Events) != tc.event {
				t.Fatalf("枠・初期・ターン・出来事: %d/%d/%d/%d", len(m.Slots), len(m.Enemies), len(m.Reinforcements), len(m.Events))
			}
			all := append([]defs.EnemyPlacement(nil), m.Enemies...)
			for _, r := range m.Reinforcements {
				all = append(all, r.EnemyPlacement)
			}
			for _, e := range m.Events {
				if e.Effect != defs.EventReinforce {
					t.Fatalf("%s は増援でない", e.ID)
				}
				if len(e.Enemies) == 0 {
					t.Fatalf("%s に増援がない", e.ID)
				}
				all = append(all, e.Enemies...)
			}
			groups := map[string]int{}
			cavaliers := 0
			for _, e := range all {
				if e.Group != "" {
					if e.Approach != defs.ApproachGroup {
						t.Fatalf("%s が一団の方式でない", e.ID)
					}
					groups[e.Group]++
				}
				if e.Class == "cavalier" && !e.Boss {
					cavaliers++
				}
			}
			if !reflect.DeepEqual(groups, tc.groups) || cavaliers != tc.cavaliers {
				t.Fatalf("一団 %v・騎兵 %d", groups, cavaliers)
			}
			if tc.id == "ch07" && (m.Enemies[0].X != tc.frontCavalier.X || m.Enemies[0].Y != tc.frontCavalier.Y) {
				t.Fatalf("初期の騎兵: (%d,%d)", m.Enemies[0].X, m.Enemies[0].Y)
			}
			if tc.id == "ch07" {
				valley, east := m.Events[0], m.Events[1]
				if valley.ID != "convoy_valley_falls" || valley.Trigger != defs.TriggerPlayerEntered || valley.Area != (defs.Area{MinX: 8, MinY: 4, MaxX: 9, MaxY: 6}) {
					t.Fatalf("谷の出口のきっかけ: %+v", valley)
				}
				if east.ID != "convoy_east_entry" || east.Trigger != defs.TriggerPlayerEntered || east.Area != (defs.Area{MinX: 13, MinY: 8, MaxX: 15, MaxY: 12}) {
					t.Fatalf("東の進入のきっかけ: %+v", east)
				}
				wantPos := map[string]defs.Slot{
					"convoy_east_cavalier":       {X: 15, Y: 12},
					"convoy_east_fighter":        {X: 15, Y: 13},
					"convoy_east_mage":           {X: 15, Y: 11},
					"convoy_east_archer":         {X: 15, Y: 14},
					"convoy_east_front_mage":     {X: 14, Y: 7},
					"convoy_east_front_cavalier": {X: 15, Y: 7},
					"convoy_east_wood_archer":    {X: 14, Y: 9},
					"convoy_east_flank_soldier":  {X: 15, Y: 9},
				}
				for _, enemy := range east.Enemies {
					pos, ok := wantPos[enemy.ID]
					if !ok || enemy.X != pos.X || enemy.Y != pos.Y || m.Rows[enemy.Y][enemy.X] == '^' {
						t.Fatalf("東の増援 %s の地形・位置: (%d,%d)", enemy.ID, enemy.X, enemy.Y)
					}
					delete(wantPos, enemy.ID)
				}
				if len(wantPos) != 0 {
					t.Fatalf("東の増援が足りない: %v", wantPos)
				}
			}
			if tc.id == "ch07" {
				abs := func(n int) int {
					if n < 0 {
						return -n
					}
					return n
				}
				for _, slot := range m.Slots {
					if abs(m.Enemies[0].X-slot.X)+abs(m.Enemies[0].Y-slot.Y) <= 8 {
						t.Fatalf("初期の騎兵が1ターン目に枠 (%d,%d) に届きうる", slot.X, slot.Y)
					}
				}
			}
			// 増援のための参照に閉路があると、倒せない相手を待つ出来事ができる。
			graph := map[string][]string{}
			for _, event := range m.Events {
				if event.Trigger == defs.TriggerEnemyDefeated {
					for _, e := range event.Enemies {
						graph[event.Unit] = append(graph[event.Unit], e.ID)
					}
				}
			}
			state := map[string]int{}
			var visit func(string) bool
			visit = func(id string) bool {
				if state[id] != 0 {
					return state[id] == 1
				}
				state[id] = 1
				for _, next := range graph[id] {
					if visit(next) {
						return true
					}
				}
				state[id] = 2
				return false
			}
			for id := range graph {
				if visit(id) {
					t.Fatalf("出来事の参照が循環: %s", id)
				}
			}
			minLevel, maxLevel := 6, 9
			if tc.id == "ch07" {
				minLevel, maxLevel = 7, 10
			}
			for _, e := range m.Events {
				for _, enemy := range e.Enemies {
					if enemy.Level < minLevel || enemy.Level > maxLevel {
						t.Fatalf("%s のレベル %d", enemy.ID, enemy.Level)
					}
					if enemy.Class == "mage" {
						w, ok := cat.Weapon(enemy.Weapons[0])
						if !ok || w.Type != defs.WeaponMagic || w.DamageType != "magic" {
							t.Fatalf("%s の武器は魔法でない", enemy.ID)
						}
						cls, _ := cat.Class("mage")
						got := cls.Base.Spd + enemy.Bonus.Spd - max(0, w.Weight-cls.Base.Str-enemy.Bonus.Str)
						want := 7
						if tc.id == "ch07" {
							want = 8
						}
						if got != want {
							t.Fatalf("%s の攻撃の速さ %d、期待 %d", enemy.ID, got, want)
						}
					} else {
						want := defs.Stats{HP: 4, Str: 2, Skl: 2, Spd: 2, Def: 2}
						if tc.id == "ch07" {
							want = defs.Stats{HP: 5, Str: 3, Skl: 2, Spd: 2, Def: 2}
						}
						if enemy.Bonus != want {
							t.Fatalf("%s の bonus %+v", enemy.ID, enemy.Bonus)
						}
					}
				}
			}
		})
	}
}

// 育った名簿には追加能力と前章までの武器を含める（DES-MAP-08）。
func TestIT06MidRoster(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id                        string
		low                       bool
		leon, noel, elsa          int
		leonLck, noelMag, noelRes int
		weapon                    string
	}{
		{"ch06", false, 6, 6, 6, 8, 5, 4, "long_bow"}, {"ch06", true, 5, 5, 5, 7, 4, 4, "long_bow"},
		{"ch07", false, 7, 7, 7, 8, 5, 5, "thunder_sword"}, {"ch07", true, 6, 6, 6, 7, 4, 4, "thunder_sword"},
	} {
		t.Run("TC-FR-MAP-04-19 "+tc.id+map[bool]string{true: "低め", false: "平均"}[tc.low], func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			byID := map[string]core.Member{}
			for _, m := range c.Roster {
				byID[m.Character] = m
			}
			if byID["leon"].Level != tc.leon || byID["noel"].Level != tc.noel || byID["elsa"].Level != tc.elsa {
				t.Fatalf("レベル: %d/%d/%d", byID["leon"].Level, byID["noel"].Level, byID["elsa"].Level)
			}
			if byID["leon"].Stats.Lck != tc.leonLck || byID["noel"].Stats.Mag != tc.noelMag || byID["noel"].Stats.Res != tc.noelRes {
				t.Fatalf("追加能力: 主人公の幸運 %d、錬金術師の魔力 %d・魔防 %d", byID["leon"].Stats.Lck, byID["noel"].Stats.Mag, byID["noel"].Stats.Res)
			}
			if tc.id == "ch06" && !containsWeapon(byID["tio"].Weapons, tc.weapon) {
				t.Fatalf("長弓がない: %v", byID["tio"].Weapons)
			}
			if tc.id == "ch07" && !containsWeapon(byID["leon"].Weapons, tc.weapon) {
				t.Fatalf("雷の剣がない: %v", byID["leon"].Weapons)
			}
			if tc.id == "ch07" && (len(byID["leon"].Weapons) != 4 || len(byID["leon"].Items) != 1 || byID["leon"].Items[0].Count != 1) {
				t.Fatalf("主人公の手持ち: 武器 %v、道具 %v", byID["leon"].Weapons, byID["leon"].Items)
			}
		})
	}
}

func containsWeapon(weapons []string, id string) bool {
	for _, w := range weapons {
		if w == id {
			return true
		}
	}
	return false
}

// 育った名簿で勝つ手順の下限を固定 seed で再現する。勝率は難しさの判定に使わない（DES-MAP-08）。
func TestIT06MidAutoplay(t *testing.T) {
	cat := baseCatalog(t)
	for _, tc := range []struct {
		id, name string
		low      bool
		bench    []string
		route    []tactics.Pos
		order    []string
		seed     uint64
	}{
		{"ch06", "平均・西", false, []string{"rina"}, []tactics.Pos{{X: 6, Y: 14}, {X: 6, Y: 10}, {X: 7, Y: 6}, {X: 10, Y: 3}}, []string{"gald", "mira", "noel", "bran", "elsa", "leon", "tio"}, 6},
		{"ch06", "低め・東", true, []string{"rina"}, []tactics.Pos{{X: 13, Y: 14}, {X: 13, Y: 10}, {X: 13, Y: 6}, {X: 10, Y: 3}}, []string{"gald", "mira", "noel", "bran", "elsa", "leon", "tio"}, 26},
		{"ch07", "平均・谷", false, []string{"rina", "elsa"}, []tactics.Pos{{X: 7, Y: 11}, {X: 7, Y: 8}, {X: 7, Y: 4}, {X: 9, Y: 2}}, []string{"gald", "mira", "noel", "bran", "leon", "tio"}, 12},
		{"ch07", "低め・迂回", true, []string{"rina", "elsa"}, []tactics.Pos{{X: 13, Y: 12}, {X: 14, Y: 8}, {X: 13, Y: 4}, {X: 11, Y: 3}}, []string{"gald", "mira", "noel", "bran", "leon", "tio"}, 3},
		{"ch06", "ミラを失いティオを出す・平均", false, []string{"mira"}, []tactics.Pos{{X: 6, Y: 14}, {X: 6, Y: 10}, {X: 7, Y: 6}, {X: 10, Y: 3}}, []string{"gald", "noel", "bran", "elsa", "leon", "tio", "rina"}, 1},
		{"ch06", "ミラを失いティオを出す・低め", true, []string{"mira"}, []tactics.Pos{{X: 13, Y: 14}, {X: 13, Y: 10}, {X: 13, Y: 6}, {X: 10, Y: 3}}, []string{"gald", "noel", "bran", "elsa", "leon", "tio", "rina"}, 36},
		{"ch07", "ミラを失いティオを出す・平均・谷", false, []string{"mira", "elsa"}, []tactics.Pos{{X: 7, Y: 11}, {X: 7, Y: 8}, {X: 7, Y: 4}, {X: 9, Y: 2}}, []string{"gald", "noel", "bran", "leon", "tio", "rina"}, 3},
		{"ch07", "ミラを失いティオを出す・低め・迂回", true, []string{"mira", "elsa"}, []tactics.Pos{{X: 13, Y: 12}, {X: 14, Y: 8}, {X: 13, Y: 4}, {X: 11, Y: 3}}, []string{"gald", "noel", "bran", "leon", "tio", "rina"}, 15},
	} {
		t.Run("TC-FR-MAP-04-18 "+tc.id+tc.name, func(t *testing.T) {
			c := estimatedRosterIT06(t, cat, tc.id, tc.low)
			if strings.Contains(tc.name, "ミラを失い") {
				for i := range c.Roster {
					if c.Roster[i].Character == "mira" {
						c.Roster[i].Alive = false
					}
				}
			}
			var members []int
			for i, m := range c.Roster {
				if !containsWeapon(tc.bench, m.Character) {
					members = append(members, i)
				}
			}
			s, _, err := c.StartBattle(members, tc.seed)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(members))
			for i, n := range members {
				ids[i] = c.Roster[n].Character
			}
			m, _ := cat.Map(tc.id)
			maxLive := len(m.Enemies)
			triggers := map[string]bool{}
			var mage *tactics.Unit
			autoObserve = func(events []tactics.Event) {
				for _, e := range events {
					if e.Kind == tactics.EventTrigger {
						triggers[e.MapEvent] = true
					}
					if e.Kind == tactics.EventReinforce && (e.Units[e.Unit].ID == "pass_group_mage" || e.Units[e.Unit].ID == "convoy_valley_mage" || e.Units[e.Unit].ID == "convoy_east_mage" || e.Units[e.Unit].ID == "convoy_east_front_mage") && mage == nil {
						u := e.Units[e.Unit]
						mage = &u
					}
					live := 0
					for _, u := range e.Units {
						if u.Alive && u.Side == tactics.Enemy {
							live++
						}
					}
					maxLive = max(maxLive, live)
				}
			}
			r := autoPlay(s.Battle, ids, autoPlan{Order: tc.order, Routes: [][]tactics.Pos{tc.route}}, 40)
			autoObserve = nil
			t.Logf("seed=%d %v; 出来事=%v; 同時の敵の最大=%d; 上限で出ない増援=0（総定義数が39以下）", tc.seed, r, triggers, maxLive)
			if maxLive > 39 {
				t.Fatal("同時の敵が39体を超えた")
			}
			for _, event := range m.Events {
				if tc.id == "ch07" && ((event.ID == "convoy_east_entry" && !strings.Contains(tc.name, "迂回")) || (event.ID == "convoy_valley_falls" && strings.Contains(tc.name, "迂回"))) {
					continue
				}
				if !triggers[event.ID] {
					t.Fatalf("出来事 %s が起きない", event.ID)
				}
			}
			if tc.id == "ch07" {
				if strings.Contains(tc.name, "迂回") && triggers["convoy_valley_falls"] {
					t.Fatal("迂回だけの手順で谷の出来事が起きた")
				}
				if strings.Contains(tc.name, "谷") && triggers["convoy_east_entry"] {
					t.Fatal("谷だけの手順で東の出来事が起きた")
				}
			}
			if mage == nil {
				t.Fatal("出来事の魔道士が出ない")
			}
			w := mage.EquippedWeapon()
			if w.ID != "fire" || w.Type != tactics.Magic || w.DamageType != tactics.Magical {
				t.Fatalf("出来事の魔道士の武器 %+v", w)
			}
			wantSpeed := 7
			if tc.id == "ch07" {
				wantSpeed = 8
			}
			if got := mage.Stats.Spd - max(0, w.Weight-mage.Stats.Str); got != wantSpeed {
				t.Fatalf("魔道士の攻撃の速さ %d、期待 %d", got, wantSpeed)
			}
			if r.Result != tactics.Won {
				t.Fatal("下限の手順で勝てない")
			}
		})
	}
}

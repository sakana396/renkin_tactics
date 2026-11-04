package core_test

import (
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体の全 9 章の小隊長の配置を検査する（TC-FR-MAP-04-30。DES-MAP-09、DES-DAT-05、DES-IT07 5.1・5.3・5.5 節）。
// 期待値は DES-IT07 5.5 節の表（初期敵・一団の名前・追加後の初期敵）から書き、本体のデータを写さない。
func TestIT07CommanderPlacement(t *testing.T) {
	cat := baseCatalog(t)
	type chapter struct {
		before, after int      // 5.5 節の「初期敵」「追加後の初期敵」
		initial       []string // 初期の一団
		events        []string // 出来事の一団
	}
	want := map[string]chapter{
		"ch01": {8, 9, []string{"fort_front"}, nil},
		"ch02": {8, 9, []string{"road_east_front"}, []string{"road_north", "road_south"}},
		"ch03": {10, 11, []string{"wall_west_front"}, []string{"wall_west"}},
		"ch04": {12, 14, []string{"west_gate_front", "east_gate_front"}, []string{"village_west"}},
		"ch05": {9, 10, []string{"gate_north"}, []string{"gate_east", "gate_reserve"}},
		"ch06": {14, 16, []string{"pass_center_front", "pass_north_front"}, []string{"pass_center", "pass_east"}},
		"ch07": {10, 11, []string{"convoy_valley_front"}, []string{"convoy_valley", "convoy_east"}},
		"ch08": {16, 18, []string{"forest_west_front", "forest_north_front"}, []string{"forest_road_reserve", "forest_north_reserve"}},
		"ch09": {26, 29, []string{"keep_gate_front", "outer_west_front", "outer_east_front"}, []string{"gate_west_reserve", "gate_east_reserve", "gate_inner_reserve"}},
	}
	totalInitial, totalEvents := 0, 0
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		w := want[id]
		t.Run("TC-FR-MAP-04-30 "+id, func(t *testing.T) {
			m, ok := cat.Map(id)
			if !ok {
				t.Fatal("地図がない")
			}
			// 初期敵の数: 既存の敵はそのままで、一団ごとに小隊長を 1 体足す。39 体以下。
			commanders := 0
			for _, e := range m.Enemies {
				if e.Commander {
					commanders++
				}
			}
			totalInitial += commanders
			for _, event := range m.Events {
				for _, e := range event.Enemies {
					if e.Commander {
						totalEvents++
					}
				}
			}
			if len(m.Enemies) != w.after || len(m.Enemies)-commanders != w.before || len(m.Enemies) > 39 {
				t.Fatalf("初期敵 %d（小隊長を除くと %d）、期待 %d（%d）", len(m.Enemies), len(m.Enemies)-commanders, w.after, w.before)
			}
			// 小隊長の定義を、同じ一団の配下と比べる（DES-IT07 5.1 節）。
			checkCommander := func(where, group string, list []defs.EnemyPlacement) defs.EnemyPlacement {
				t.Helper()
				var leader []defs.EnemyPlacement
				maxLevel := 0
				var bonus *defs.Stats
				for _, e := range list {
					if e.Group != group {
						continue
					}
					if e.Commander {
						leader = append(leader, e)
						continue
					}
					maxLevel = max(maxLevel, e.Level)
					// 魔道士は物理の能力の bonus が違うので、前線の兵種の配下と比べる。
					if e.Class != "mage" {
						if bonus != nil && *bonus != e.Bonus {
							t.Fatalf("%s の一団 %s の配下の bonus がそろわない: %+v / %+v", where, group, *bonus, e.Bonus)
						}
						b := e.Bonus
						bonus = &b
					}
				}
				if len(leader) != 1 || maxLevel == 0 || bonus == nil {
					t.Fatalf("%s の一団 %s の小隊長は %d 体（配下の最大 Lv %d）、期待 1 体", where, group, len(leader), maxLevel)
				}
				c := leader[0]
				if c.Boss || c.AI != defs.AIHold || c.Approach != defs.ApproachGroup || c.ID == "" {
					t.Fatalf("%s の小隊長 %s: ボス %v 思考 %s 寄り方 %s", where, c.ID, c.Boss, c.AI, c.Approach)
				}
				if c.Level != maxLevel+1 {
					t.Fatalf("%s の小隊長 %s の Lv %d、期待は配下の上限 %d の +1", where, c.ID, c.Level, maxLevel)
				}
				if c.Bonus != *bonus {
					t.Fatalf("%s の小隊長 %s の bonus %+v、期待は配下と同じ %+v", where, c.ID, c.Bonus, *bonus)
				}
				// 小隊長を攻撃しても一団の全員が動き出す（REV-092 F-08）。
				linked := false
				for _, event := range m.Events {
					if event.Trigger == defs.TriggerEnemyAttacked && event.Effect == defs.EventChargeGroup && event.Unit == c.ID && event.Group == group {
						linked = true
					}
				}
				if !linked {
					t.Fatalf("%s の小隊長 %s を標的とする enemy_attacked → charge_group がない", where, c.ID)
				}
				return c
			}
			initialGroups := map[string]bool{}
			for _, e := range m.Enemies {
				if e.Group != "" {
					initialGroups[e.Group] = true
				}
			}
			if len(initialGroups) != len(w.initial) {
				t.Fatalf("初期の一団 %v、期待 %v", initialGroups, w.initial)
			}
			for _, group := range w.initial {
				checkCommander("初期", group, m.Enemies)
			}
			// 出来事の一団: 小隊長は同じ出来事の列の先頭で、配下と 7 マス以内（DES-IT07 5.3 節、REV-094 の申し送り）。
			eventGroups := map[string]bool{}
			for _, event := range m.Events {
				for _, e := range event.Enemies {
					if e.Group != "" {
						eventGroups[e.Group] = true
					}
				}
			}
			if len(eventGroups) != len(w.events) {
				t.Fatalf("出来事の一団 %v、期待 %v", eventGroups, w.events)
			}
			for _, group := range w.events {
				found := false
				for _, event := range m.Events {
					if len(event.Enemies) == 0 || !containsGroup(event.Enemies, group) {
						continue
					}
					found = true
					c := checkCommander("出来事 "+event.ID, group, event.Enemies)
					if first := event.Enemies[0]; first.ID != c.ID {
						t.Fatalf("出来事 %s の列の先頭は %s、期待は小隊長 %s", event.ID, first.ID, c.ID)
					}
					for _, e := range event.Enemies {
						if e.Group == group && absInt(e.X-c.X)+absInt(e.Y-c.Y) > groupMaxManhattan {
							t.Fatalf("出来事 %s の小隊長 %s (%d,%d) と %s (%d,%d) が 7 マスより離れる", event.ID, c.ID, c.X, c.Y, e.ID, e.X, e.Y)
						}
					}
				}
				if !found {
					t.Fatalf("出来事の一団 %s がない", group)
				}
			}
			// 小隊長は 1 ターン目に出撃枠を攻撃できない（DES-IT07 5.5 節）。
			campaign, err := core.NewCampaignAt(cat, core.Withdraw, id)
			if err != nil {
				t.Fatal(err)
			}
			battle, _, err := campaign.StartBattle([]int{0}, 1)
			if err != nil {
				t.Fatal(err)
			}
			slots := map[tactics.Pos]bool{}
			for _, slot := range m.Slots {
				slots[tactics.Pos{X: slot.X, Y: slot.Y}] = true
			}
			seen := 0
			for i, u := range battle.Battle.Units() {
				if u.Side != tactics.Enemy || !u.Commander {
					continue
				}
				seen++
				for pos := range battle.Battle.Reachable(i) {
					for _, target := range battle.Battle.AttackRange(i, pos) {
						if slots[target] {
							t.Fatalf("小隊長 %s (%d,%d) の 1 ターン目の攻撃が出撃枠 %v に届く", u.ID, u.Pos.X, u.Pos.Y, target)
						}
					}
				}
			}
			if seen != len(w.initial) {
				t.Fatalf("戦闘の小隊長 %d 体、期待 %d", seen, len(w.initial))
			}
		})
	}
	if totalInitial != 14 || totalEvents != 15 {
		t.Fatalf("小隊長は初期 %d・出来事 %d、期待 14・15（DES-IT07 5.5 節）", totalInitial, totalEvents)
	}
}

// containsGroup は、敵の列に一団 group の一員がいるかを返す。
func containsGroup(list []defs.EnemyPlacement, group string) bool {
	for _, e := range list {
		if e.Group == group {
			return true
		}
	}
	return false
}

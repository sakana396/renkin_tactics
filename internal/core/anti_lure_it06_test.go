package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// E-100 の章ごとの初期突撃、一団と連動、継続増援を検査する（DES-AI-02、DES-MAP-08）。
func TestIT06AntiLure(t *testing.T) {
	cat := baseCatalog(t)
	wantGroups := map[string]map[string]int{
		"ch01": {"fort_front": 2},
		"ch02": {"road_east_front": 3},
		"ch03": {"wall_west_front": 3},
		"ch04": {"west_gate_front": 4},
		"ch05": {"gate_north": 4},
		"ch06": {"pass_east_front": 3, "pass_center_front": 3},
		"ch07": {"convoy_east_front": 3, "convoy_valley_front": 4},
		"ch08": {"forest_east_front": 3, "forest_west_front": 3},
		"ch09": {"gate_east_front": 4, "gate_west_front": 4},
	}
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		t.Run("TC-FR-MAP-04-23 "+id, func(t *testing.T) {
			m, ok := cat.Map(id)
			if !ok {
				t.Fatal("地図がない")
			}
			charge := 0
			groups := map[string][]defs.EnemyPlacement{}
			for _, e := range m.Enemies {
				if e.AI == defs.AICharge {
					charge++
				}
				if e.Group != "" {
					if e.Approach != defs.ApproachGroup {
						t.Fatalf("%s は一団の寄り方ではない", e.ID)
					}
					groups[e.Group] = append(groups[e.Group], e)
				}
			}
			gotGroups := map[string]int{}
			for group, members := range groups {
				gotGroups[group] = len(members)
			}
			if !reflect.DeepEqual(gotGroups, wantGroups[id]) {
				t.Fatalf("初期の一団は %v、期待 %v", gotGroups, wantGroups[id])
			}
			if ch == 1 {
				if len(m.Enemies) != 8 || charge != 2 || len(m.Reinforcements) != 3 || len(groups) != 1 || len(groups["fort_front"]) != 2 {
					t.Fatalf("1 章の例外: 敵 %d 突撃 %d 増援 %d 一団 %v", len(m.Enemies), charge, len(m.Reinforcements), groups)
				}
			} else {
				if charge*5 < len(m.Enemies)*2 {
					t.Fatalf("初期突撃は %d/%d で 4 割未満", charge, len(m.Enemies))
				}
				if len(groups) == 0 {
					t.Fatal("守りの一団がない")
				}
				for group, members := range groups {
					if len(members) < 3 || len(members) > 5 {
						t.Fatalf("%s は %d 体", group, len(members))
					}
				}
				end := 8
				if ch >= 8 {
					end = 12
				}
				counts := map[int]int{}
				for _, r := range m.Reinforcements {
					counts[r.Turn]++
				}
				for turn := 4; turn <= end; turn++ {
					if counts[turn] < 1 || counts[turn] > 2 {
						t.Fatalf("%d ターン目は %d 体", turn, counts[turn])
					}
				}
			}
			for group, members := range groups {
				reachable := reachableByTerrain(m, cat, defs.Slot{X: members[0].X, Y: members[0].Y}, nil)
				for _, member := range members {
					if !reachable[defs.Slot{X: member.X, Y: member.Y}] {
						t.Fatalf("%s の %s は別の到達領域にいる", group, member.ID)
					}
				}
				for _, member := range members {
					linked := false
					for _, event := range m.Events {
						if event.Trigger == defs.TriggerEnemyAttacked && event.Effect == defs.EventChargeGroup && event.Unit == member.ID && event.Group == group {
							linked = true
							break
						}

					}
					if !linked {
						t.Fatalf("%s の %s が攻撃されたときに一団が動かない", group, member.ID)
					}
				}
			}
			if ch >= 2 {
				// 追加した突撃も含め、1 ターン目に出撃枠が攻撃されない。
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
				for i, u := range battle.Battle.Units() {
					if u.Side != tactics.Enemy || m.Enemies[i-1].AI != defs.AICharge {
						continue
					}
					for pos := range battle.Battle.Reachable(i) {
						for _, target := range battle.Battle.AttackRange(i, pos) {
							if slots[target] {
								t.Fatalf("%s (%d,%d) の攻撃が出撃枠に届く", u.ID, u.Pos.X, u.Pos.Y)
							}
						}
					}
				}
				members := make([]int, len(m.Slots))
				for i := range members {
					members[i] = i
				}
				fullBattle, _, err := campaign.StartBattle(members, 1)
				if err != nil {
					t.Fatal(err)
				}
				events, err := fullBattle.Battle.EndPhase()
				if err != nil {
					t.Fatal(err)
				}
				for _, event := range events {
					if event.Kind == tactics.EventStrike && event.Units[event.Target].Side == tactics.Player {
						t.Fatalf("1 ターン目に出撃枠の自軍が攻撃された: %s ← %s (%d,%d)", event.Units[event.Target].Name, event.Units[event.Unit].Name, event.Units[event.Unit].Pos.X, event.Units[event.Unit].Pos.Y)
					}
				}
			}

		})
	}
}

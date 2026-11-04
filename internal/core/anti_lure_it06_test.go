package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// pathSteps は、地形の通れるマスだけを通って start から各マスへ行く歩数を返す（壁を回り込む分を含む）。
func pathSteps(m defs.MapDef, cat *defs.Catalog, start defs.Slot) map[defs.Slot]int {
	cost := map[string]int{}
	for _, tr := range cat.Terrains() {
		cost[tr.Symbol] = tr.Cost
	}
	passable := func(p defs.Slot) bool {
		if p.Y < 0 || p.Y >= len(m.Rows) || p.X < 0 || p.X >= len([]rune(m.Rows[0])) {
			return false
		}
		return cost[string([]rune(m.Rows[p.Y])[p.X])] > 0
	}
	steps := map[defs.Slot]int{start: 0}
	queue := []defs.Slot{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range []defs.Slot{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			n := defs.Slot{X: cur.X + d.X, Y: cur.Y + d.Y}
			if _, ok := steps[n]; !ok && passable(n) {
				steps[n] = steps[cur] + 1
				queue = append(queue, n)
			}
		}
	}
	return steps
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// 守りの一団の一員どうしの近さの目安（E-100 ②、DES-IT06 10.1 節「壁を回り込まずに行き来できる」）。
const (
	groupMaxManhattan = 7 // 一員どうしのマス目の距離の上限
	groupMaxDetour    = 2 // 経路の歩数がマス目の距離を超えてよい分
	// 攻撃のあと、全員が持ち場を離れるまでの敵軍の番の数の上限。集まる間の待ちを含む（DES-IT06 10.2 節）。
	groupLeavePhases = 4
)

// 小隊長を除いた初期敵の数を、初期突撃 4 割の分母にする（DES-IT07 5.5 節）。
func initialEnemiesWithoutCommanders(enemies []defs.EnemyPlacement) int {
	count := 0
	for _, e := range enemies {
		if !e.Commander {
			count++
		}
	}
	return count
}

// E-100 の章ごとの初期突撃、守りの一団と連動、継続増援を検査する（DES-AI-02、DES-MAP-08、DES-IT06 10.1 節・13 章）。
func TestIT06AntiLure(t *testing.T) {
	t.Run("TC-FR-MAP-04-23 小隊長を分母から除く", func(t *testing.T) {
		enemies := []defs.EnemyPlacement{{AI: defs.AICharge}, {AI: defs.AIHold}, {Commander: true}}
		if got := initialEnemiesWithoutCommanders(enemies); got != 2 {
			t.Fatalf("小隊長を除いた初期敵 = %d、期待 2", got)
		}
	})
	cat := baseCatalog(t)
	// 守りの一団は、主な守りの場所ごとに待つ一員だけで作る（E-100 ②）。
	wantGroups := map[string]map[string]int{
		"ch01": {"fort_front": 2},                                                    // 砦の手前の剣士と弓兵（E-100 の 1 章の例外）
		"ch02": {"road_east_front": 3},                                               // ボスの砦の前
		"ch03": {"wall_west_front": 3},                                               // 外の城壁の西の切れ目
		"ch04": {"west_gate_front": 3, "east_gate_front": 3},                         // 城壁の西門・東門
		"ch05": {"gate_north": 3},                                                    // 北の門
		"ch06": {"pass_center_front": 3, "pass_north_front": 3},                      // 峠の中央・本営の前
		"ch07": {"convoy_valley_front": 3},                                           // 谷の中だけ（F-02）
		"ch08": {"forest_west_front": 3, "forest_north_front": 3},                    // 西の砦・北の砦の前
		"ch09": {"keep_gate_front": 4, "outer_west_front": 3, "outer_east_front": 3}, // 本営の門・外壁の西と東の切れ目
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
				if e.Group == "" {
					// 決まり 1 の 4 割は一団の外の敵で数える。
					if e.AI == defs.AICharge {
						charge++
					}
					continue
				}
				if e.Approach != defs.ApproachGroup {
					t.Fatalf("%s は一団の寄り方ではない", e.ID)
				}
				if e.AI != defs.AIHold {
					t.Fatalf("守りの一団 %s の %s が待つ敵ではない（%s）。一員が突撃だと 1 ターン目に一団の全員が動き出す", e.Group, e.ID, e.AI)
				}
				groups[e.Group] = append(groups[e.Group], e)
			}
			gotGroups := map[string]int{}
			for group, members := range groups {
				gotGroups[group] = len(members)
			}
			if !reflect.DeepEqual(gotGroups, wantGroups[id]) {
				t.Fatalf("守りの一団は %v、期待 %v", gotGroups, wantGroups[id])
			}
			if ch == 1 {
				if len(m.Enemies) != 8 || charge != 2 || len(m.Reinforcements) != 3 {
					t.Fatalf("1 章の例外: 敵 %d 突撃 %d ターン増援 %d", len(m.Enemies), charge, len(m.Reinforcements))
				}
				// #151 の出来事の増援 3 体を残す（E-100、F-03）。
				found := false
				for _, e := range m.Events {
					if e.ID == "border_east_entry" {
						found = true
						if e.Trigger != defs.TriggerPlayerEntered || e.Effect != defs.EventReinforce || len(e.Enemies) != 3 {
							t.Fatalf("border_east_entry: きっかけ %s 効果 %s 増援 %d", e.Trigger, e.Effect, len(e.Enemies))
						}
					}
				}
				if !found {
					t.Fatal("出来事の増援 border_east_entry がない")
				}
			} else {
				denominator := initialEnemiesWithoutCommanders(m.Enemies)
				if charge*5 < denominator*2 {
					t.Fatalf("一団の外の初期突撃は %d/%d で 4 割未満", charge, denominator)
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
				for _, a := range members {
					steps := pathSteps(m, cat, defs.Slot{X: a.X, Y: a.Y})
					for _, b := range members {
						manhattan := absInt(a.X-b.X) + absInt(a.Y-b.Y)
						got, ok := steps[defs.Slot{X: b.X, Y: b.Y}]
						if !ok || manhattan > groupMaxManhattan || got > manhattan+groupMaxDetour {
							t.Fatalf("%s の %s (%d,%d) と %s (%d,%d) は近くない: マス目 %d、経路 %d（届く %v）", group, a.ID, a.X, a.Y, b.ID, b.X, b.Y, manhattan, got, ok)
						}
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
				// 1 ターン目に出撃枠が攻撃されない。
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
			checkGroupsHold(t, cat, id, m)
			checkGroupsCharge(t, cat, id, m, groups)
		})
	}
}

// checkGroupsHold は、育った名簿で枠を埋め、自軍が何もしないまま 3 ターン過ごしても、
// 守りの一団が持ち場を離れず、待つ敵のままであることを確かめる（E-100 ②、F-01）。
func checkGroupsHold(t *testing.T, cat *defs.Catalog, id string, m defs.MapDef) {
	t.Helper()
	var c *core.Campaign
	if id == "ch01" {
		var err error
		c, err = core.NewCampaignAt(cat, core.Withdraw, id)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		c = estimatedRosterIT06(t, cat, id, false)
	}
	n := len(m.Slots)
	if n > len(c.Roster) {
		n = len(c.Roster)
	}
	members := make([]int, n)
	for i := range members {
		members[i] = i
	}
	s, _, err := c.StartBattle(members, 1)
	if err != nil {
		t.Fatal(err)
	}
	b := s.Battle
	start := map[int]tactics.Pos{}
	for i, u := range b.Units() {
		if u.Side == tactics.Enemy && u.Group != "" {
			start[i] = u.Pos
		}
	}
	for turn := 1; turn <= 3 && b.Result == tactics.Ongoing; turn++ {
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		for i, p := range start {
			u := b.Units()[i]
			if !u.Alive {
				continue
			}
			if u.Pos != p || u.AI != tactics.Hold {
				t.Fatalf("%d 回目の敵軍の番のあと、守りの一団 %s の %s が持ち場 (%d,%d) を離れた: (%d,%d)、突撃 %v", turn, u.Group, u.ID, p.X, p.Y, u.Pos.X, u.Pos.Y, u.AI == tactics.Charge)
			}
		}
	}
}

// checkGroupsCharge は、守りの一団の一員を 1 体ずつ攻撃し、そのとき一団の全員が突撃に変わり、
// 次の敵軍の番に全員が動き出す（動くか攻撃する）ことを確かめる（E-100 ②、F-01）。
func checkGroupsCharge(t *testing.T, cat *defs.Catalog, id string, m defs.MapDef, groups map[string][]defs.EnemyPlacement) {
	t.Helper()
	for group, placements := range groups {
		for _, target := range placements {
			setup, err := core.NewSetup(cat, id)
			if err != nil {
				t.Fatal(err)
			}
			targetIndex, attacker := -1, -1
			occupied := map[tactics.Pos]bool{}
			for i, u := range setup.Units {
				occupied[u.Pos] = true
				if u.Side == tactics.Enemy && u.ID == target.ID {
					targetIndex = i
				}
			}
			for i, u := range setup.Units {
				if u.Side == tactics.Player && !u.Lord && len(u.Weapons) > 0 && u.Weapons[0].Type != tactics.Staff {
					attacker = i
					break
				}
			}
			if targetIndex < 0 || attacker < 0 {
				t.Fatalf("%s の %s を攻撃する組が作れない", group, target.ID)
			}
			w := setup.Units[attacker].Weapons[0]
			from, found := tactics.Pos{}, false
			for y := range setup.Terrain {
				for x := range setup.Terrain[y] {
					p := tactics.Pos{X: x, Y: y}
					d := absInt(x-target.X) + absInt(y-target.Y)
					if found || occupied[p] || setup.Terrain[y][x].Cost < 1 || d < w.MinRange || d > w.MaxRange {
						continue
					}
					from, found = p, true
				}
			}
			if !found {
				t.Fatalf("%s の %s を攻撃できるマスがない", group, target.ID)
			}
			setup.Units[attacker].Pos = from
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			start := map[int]tactics.Pos{}
			for i, u := range b.Units() {
				if u.Side == tactics.Enemy && u.Group == group {
					start[i] = u.Pos
				}
			}
			events, err := b.Act(tactics.Command{Unit: attacker, To: from, Action: tactics.Attack, Target: targetIndex})
			if err != nil {
				t.Fatalf("%s の %s を攻撃できない: %v", group, target.ID, err)
			}
			charged := false
			for _, e := range events {
				if e.Kind == tactics.EventGroupCharge && e.Group == group {
					charged = true
				}
			}
			if !charged {
				t.Fatalf("%s の %s を攻撃しても一団が突撃に変わらない", group, target.ID)
			}
			for i := range start {
				if u := b.Units()[i]; u.Alive && u.AI != tactics.Charge {
					t.Fatalf("%s の %s を攻撃したのに %s が突撃でない", group, target.ID, u.ID)
				}
			}
			// 一団の方式の敵は、危険な範囲の外で集まってからそろって入る（DES-IT06 10.2 節）。
			// 集まる先にすでにいる一員や、狭い道で前の列を待つ一員はその番に動かないことがある。
			// そこで、次の敵軍の番に一団が動き出すこと、待つ敵に戻らないこと、自軍が動かなくても
			// groupLeavePhases 回の敵軍の番のうちに全員が持ち場を離れる（動くか攻撃する）ことを確かめる。
			acted := map[int]bool{}
			for phase := 1; phase <= groupLeavePhases && b.Result == tactics.Ongoing; phase++ {
				enemyEvents, err := b.EndPhase()
				if err != nil {
					t.Fatal(err)
				}
				moved := false
				for _, e := range enemyEvents {
					if _, member := start[e.Unit]; member && (e.Kind == tactics.EventStrike || e.Kind == tactics.EventMove) {
						acted[e.Unit] = true
						moved = true
					}
				}
				if phase == 1 && !moved {
					t.Fatalf("%s の %s を攻撃したあとの敵軍の番に、一団の誰も動かない", group, target.ID)
				}
				for i := range start {
					if u := b.Units()[i]; u.Alive && u.AI != tactics.Charge {
						t.Fatalf("%s の %s を攻撃したあと、%s が待つ敵に戻った", group, target.ID, u.ID)
					}
				}
			}
			for i, p := range start {
				if u := b.Units()[i]; u.Alive && !acted[i] && b.Result == tactics.Ongoing && id != "ch09" {
					t.Fatalf("%s の %s を攻撃したあとの敵軍の番 %d 回で、%s が (%d,%d) から動かない", group, target.ID, groupLeavePhases, u.ID, p.X, p.Y)
				}
			}
		}
	}
}

// 3 章の 5・6 ターン目のファイターの増援は、内の城壁の西の部屋の外に出ない（E-114・E-115、DES-MAP-03、map-ch03.md 7 章）。
// 元のマスがふさがれると距離 2 までの通れるマスに出るので、その候補のどれもが、壁を回り込まずに
// 元のマスから行ける（通れるマスだけを通る歩数がマス目の距離と同じ）ことを確かめる。
func TestIT06Ch03TurnFiveReinforcementOrigin(t *testing.T) {
	// 5 ターン目は E-114、6 ターン目は E-115 で (5,1) に動かした。
	for _, turn := range []int{5, 6} {
		t.Run(fmt.Sprintf("TC-FR-MAP-04-23 ch03 の %d ターン目の増援の元は (5,1) で、候補が壁の外に出ない", turn), func(t *testing.T) {
			cat := baseCatalog(t)
			m, ok := cat.Map("ch03")
			if !ok {
				t.Fatal("ch03 がない")
			}
			var origin *defs.Slot
			for _, r := range m.Reinforcements {
				if r.Turn == turn && r.Class == "fighter" {
					if origin != nil {
						t.Fatalf("%d ターン目のファイターの増援が 2 体以上ある", turn)
					}
					origin = &defs.Slot{X: r.X, Y: r.Y}
				}
			}
			if origin == nil {
				t.Fatalf("%d ターン目のファイターの増援がない", turn)
			}
			if *origin != (defs.Slot{X: 5, Y: 1}) {
				t.Fatalf("%d ターン目のファイターの元のマス = %+v、期待 (5,1)（E-114・E-115）", turn, *origin)
			}
			steps := pathSteps(m, cat, *origin)
			const reach = 2 // DES-MAP-03: 元のマスがふさがれたときに探す距離
			for y := origin.Y - reach; y <= origin.Y+reach; y++ {
				for x := origin.X - reach; x <= origin.X+reach; x++ {
					d := absInt(x-origin.X) + absInt(y-origin.Y)
					if d > reach || y < 0 || y >= len(m.Rows) || x < 0 || x >= len([]rune(m.Rows[0])) {
						continue
					}
					c := defs.Slot{X: x, Y: y}
					if st, ok := steps[c]; ok && st == d {
						continue
					}
					if passableTerrain(m, cat, c) {
						t.Errorf("候補 (%d,%d) は通れるが、元のマスから壁を回り込まないと行けない（歩数 %v、距離 %d）", x, y, steps[c], d)
					}
				}
			}
		})
	}
}

// passableTerrain は、マス p の地形が通れるかを返す。
func passableTerrain(m defs.MapDef, cat *defs.Catalog, p defs.Slot) bool {
	for _, tr := range cat.Terrains() {
		if tr.Symbol == string([]rune(m.Rows[p.Y])[p.X]) {
			return tr.Cost > 0
		}
	}
	return false
}

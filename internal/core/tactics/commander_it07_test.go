package tactics

import (
	"errors"
	"testing"
)

// Setup も小隊長の無効な組合せを拒み、戦闘状態を作らない（TC-FR-SYS-04-46、DES-DAT-05）。
func TestCommanderSetupValidationIt07(t *testing.T) {
	w := Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}
	base := UnitSetup{Name: "小隊長", Side: Enemy, Level: 1, AI: Hold, Approach: GroupApproach, Group: "g", Commander: true, Stats: Stats{HP: 10}, Pos: Pos{1, 0}, Weapon: w}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Level: 1, Stats: Stats{HP: 10}, Pos: Pos{0, 0}, Weapon: w}
	for _, tc := range []struct {
		name string
		edit func(*UnitSetup)
	}{
		{"ai=charge", func(u *UnitSetup) { u.AI = Charge }},
		{"boss", func(u *UnitSetup) { u.Boss = true }},
		{"approach=direct", func(u *UnitSetup) { u.Approach = Direct }},
		{"一団の名前なし", func(u *UnitSetup) { u.Group = "" }},
	} {
		t.Run("TC-FR-SYS-04-46 "+tc.name, func(t *testing.T) {
			u := base
			tc.edit(&u)
			b, events, err := New(Setup{Terrain: [][]Terrain{{{Cost: 1}, {Cost: 1}}}, Units: []UnitSetup{player, u}}, 1)
			if !errors.Is(err, ErrInvalidSetup) || b != nil || events != nil {
				t.Fatalf("無効な小隊長を受理した: 戦闘=%v 出来事=%v 誤り=%v", b, events, err)
			}
		})
	}
}

func commanderBattleIt07(t *testing.T) *Battle {
	t.Helper()
	w := Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1, Might: 1, Hit: 100}
	plain := Terrain{Cost: 1}
	terrain := make([][]Terrain, 3)
	for y := range terrain {
		terrain[y] = []Terrain{plain, plain, plain, plain, plain, plain, plain, plain}
	}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Level: 1, Stats: Stats{HP: 30, Mov: 0}, Pos: Pos{0, 1}, Weapon: w}
	enemy := func(name string, x, y int) UnitSetup {
		return UnitSetup{Name: name, Side: Enemy, Level: 1, AI: Hold, Approach: GroupApproach, Group: "g", Stats: Stats{HP: 10, Mov: 2}, Pos: Pos{x, y}, Weapon: w}
	}
	leader := enemy("小隊長", 6, 1)
	leader.Commander = true
	guard := enemy("配下", 7, 1)
	boss := enemy("ボス", 7, 2)
	boss.Group, boss.Approach, boss.Boss = "", Direct, true
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, leader, guard, boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// 撃破直後の有効な寄り方と元の定義を確かめる（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderCollapseIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	direct := b.units[2]
	direct.Pos, direct.Group, direct.Approach = Pos{5, 0}, "", Direct
	b.units = append(b.units, direct)
	if _, err := b.EndPhase(); err != nil {
		t.Fatal(err)
	}
	if !b.units[1].Alive || b.units[1].Collapsed || b.units[2].Collapsed || b.units[1].Approach != GroupApproach || b.units[2].Approach != GroupApproach {
		t.Fatal("小隊長の生存中に一団が崩れた")
	}
	key := b.groupKey("g", b.units[1].Pos)
	b.groupEntered[key] = true
	b.groupPlan = map[int]Pos{2: {7, 1}}
	b.groupDanger = map[int]map[Pos]bool{2: {{7, 1}: true}}
	b.groupGathering[key] = true
	b.fall(nil, 1)
	if !b.units[2].Collapsed || b.units[2].Approach != GroupApproach || !b.groupCollapsed[key] || b.units[4].Collapsed {
		t.Fatalf("崩れない: %+v", b.units[2])
	}
	before := b.units[2].Pos
	b.actEnemy(nil, 2)
	if b.units[2].AI != Charge || b.units[2].Pos == before {
		t.Fatalf("待つ配下が直進しない: %+v", b.units[2])
	}
	if !b.groupEntered[key] || b.groupPlan[2] != (Pos{7, 1}) || !b.groupDanger[2][Pos{7, 1}] || !b.groupGathering[key] {
		t.Fatal("番の始めの一団の決定を再計算した")
	}
}

// 未突入の一団でも、撃破後は残った集合先を参照せず自軍へ寄る（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderUnenteredIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	key := b.groupKey("g", b.units[1].Pos)
	b.groupPlan = map[int]Pos{2: {7, 0}}
	b.groupDanger = map[int]map[Pos]bool{2: {{5, 1}: true}}
	b.groupGathering[key] = true
	b.fall(nil, 1)
	if b.groupEntered[key] {
		t.Fatal("一団が突入済みになった")
	}
	before := b.units[2].Pos
	b.actEnemy(nil, 2)
	if b.units[2].Pos.X >= before.X || b.units[2].AI != Charge {
		t.Fatalf("集合先を無視して自軍へ寄らない: %+v", b.units[2])
	}
}

// 敵番の始めに崩れた配下を計画にも集合先の予約にも入れない（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderNextPhaseIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	b.groupPlan = map[int]Pos{2: {2, 0}}
	b.fall(nil, 1)
	b.units[0].Pos = Pos{4, 1}
	b.planEnemyGroups(len(b.units))
	if _, ok := b.groupDanger[2]; ok {
		t.Fatal("崩れた配下を一団の計画に入れた")
	}
	// 残った集合先は古い状態として保持するが、他の一団の予約には使わない。
	other := b.units[2]
	other.Group, other.Collapsed, other.Pos, other.AI = "other", false, Pos{3, 0}, Charge
	b.units = append(b.units, other)
	b.units[0].Pos = Pos{0, 1}
	b.units[0].Weapons[0].MaxRange = 2
	b.planEnemyGroups(len(b.units))
	if b.groupPlan[4] != (Pos{2, 0}) {
		t.Fatalf("生きた一団の集合先を奪った: %+v", b.groupPlan)
	}
}

// 崩れた配下を、生存する別の一団の近さの判定に数えない（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderGroupMoveIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	for y := range b.terrain {
		for len(b.terrain[y]) < 18 {
			b.terrain[y] = append(b.terrain[y], Terrain{Cost: 1})
		}
	}
	b.units[1].Pos = Pos{1, 1}
	b.units[1].Stats.Mov = 12
	b.units[2].Pos = Pos{2, 1}
	other := b.units[2]
	other.Pos, other.Collapsed = Pos{10, 1}, true
	b.units = append(b.units, other)
	to, ok := b.groupMove(1, b.Reachable(1), Pos{12, 1})
	if !ok || to.X > 9 {
		t.Fatalf("崩れた配下を仲間に数えた: %v, %v", to, ok)
	}
}

// 小隊長自身も生存中は一団の集合判断に参加する（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderLeaderPlannedIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	b.mapState.charged = map[string]bool{}
	b.mapState.charged["g"] = true
	b.groupGathering[b.groupKey("g", b.units[1].Pos)] = true
	b.planEnemyGroups(len(b.units))
	if _, ok := b.groupDanger[1]; !ok {
		t.Fatal("小隊長を一団の判断から外した")
	}
}

// 敵軍の番の途中で小隊長を失うと、未行動の配下だけがその番に動く（TC-FR-BTL-07-25、DES-AI-03）。
func checkCommanderMidPhaseIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	// 小隊長の攻撃への自軍の反撃で、小隊長が敵番途中に倒れる。
	b.units[0].Stats.Str = 20
	b.units[1].Pos = Pos{1, 1}
	firstFollower := b.units[2]
	firstFollower.Commander = false
	firstFollower.Pos = Pos{5, 2}
	b.units = []Unit{b.units[0], firstFollower, b.units[1], b.units[2], b.units[3]}
	b.groupEntered[b.groupKey("g", b.units[2].Pos)] = true
	first := b.units[1].Pos
	later := b.units[3].Pos
	events, err := b.EndPhase()
	if err != nil {
		t.Fatal(err)
	}
	fall := false
	for _, e := range events {
		if e.Kind == EventFall && e.Unit == 2 {
			fall = true
		}
	}
	if !fall || b.units[1].Pos != first || b.units[1].AI != Hold || !b.units[1].Collapsed || !b.units[3].Collapsed || b.units[3].AI != Charge || b.units[3].Pos == later || distance(b.units[3].Pos, b.units[0].Pos) >= distance(later, b.units[0].Pos) {
		t.Fatalf("配下の切替: %+v / %+v", b.units[1], b.units[3])
	}
}

// 増援の小隊長が出ないときは配下が初めから崩れる（TC-FR-BTL-07-26、DES-AI-03）。
func checkCommanderReinforcementIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	leader := UnitSetup{Name: "増援小隊長", Side: Enemy, Level: 1, AI: Hold, Approach: GroupApproach, Group: "reserve", Commander: true, Stats: Stats{HP: 10}, Pos: Pos{4, 1}, Weapon: Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}}
	follower := leader
	follower.Commander, follower.Name, follower.Pos = false, "増援配下", Pos{5, 1}
	// 同時 39 体なら両方が出られず、予約は先送りしない。
	for b.liveEnemies() < maxLiveEnemies {
		u := b.units[3]
		u.Pos = Pos{7, 2}
		b.units = append(b.units, u)
	}
	b.mapEvents = []MapEventSetup{{Enemies: []UnitSetup{leader, follower}}}
	b.mapState.reservations = []reservation{{Turn: 2, Event: 0}}
	b.Turn = 2
	if events := b.placeEventReinforcements(nil); len(events) != 0 {
		t.Fatalf("39 体で増援が出た: %+v", events)
	}
	if !b.groupCollapsed[b.groupKey("reserve", leader.Pos)] {
		t.Fatal("出られない小隊長を記録しない")
	}
	b.fall(nil, 4)
	if events := b.placeEventReinforcements(nil); len(events) != 0 {
		t.Fatalf("失敗した増援を先送りした: %+v", events)
	}
}

// 出来事の増援で小隊長が置けない場合も、同じ出来事の配下だけは出す（TC-FR-BTL-07-26、DES-AI-03）。
func checkCommanderEventReinforcementIt07(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		b := commanderBattleIt07(t)
		leader := UnitSetup{Name: "増援小隊長", Side: Enemy, Level: 1, AI: Hold, Approach: GroupApproach, Group: "reserve", Commander: true, Stats: Stats{HP: 10}, Pos: Pos{4, 1}, Weapon: Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}}
		follower := leader
		follower.Commander, follower.Name, follower.Pos = false, "増援配下", Pos{7, 0}
		if blocked {
			for y := 0; y < 3; y++ {
				for x := 2; x <= 6; x++ {
					at := Pos{x, y}
					if distance(at, leader.Pos) > 2 {
						continue
					}
					occupied := false
					for _, u := range b.units {
						if u.Alive && u.Pos == at {
							occupied = true
						}
					}
					if !occupied {
						u := b.units[3]
						u.Pos = at
						b.units = append(b.units, u)
					}
				}
			}
		}
		b.mapEvents = []MapEventSetup{{Enemies: []UnitSetup{leader, follower}}}
		b.mapState.reservations = []reservation{{Turn: 2, Event: 0}}
		b.Turn = 2
		events := b.placeEventReinforcements(nil)
		if blocked {
			if len(events) != 1 || b.units[events[0].Unit].Commander || !b.units[events[0].Unit].Collapsed {
				t.Fatalf("小隊長が出られない出来事: %+v", events)
			}
		} else {
			if len(events) != 2 || !b.units[events[0].Unit].Commander || b.units[events[1].Unit].Collapsed {
				t.Fatalf("小隊長が先の出来事: %+v", events)
			}
		}
		if len(b.placeEventReinforcements(nil)) != 0 {
			t.Fatal("予約を二度使った")
		}
	}
	// 敵 38 体なら先頭の小隊長だけが上限内で出る。
	b := commanderBattleIt07(t)
	leader := UnitSetup{Name: "増援小隊長", Side: Enemy, Level: 1, AI: Hold, Approach: GroupApproach, Group: "reserve", Commander: true, Stats: Stats{HP: 10}, Pos: Pos{4, 1}, Weapon: Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1}}
	follower := leader
	follower.Commander, follower.Name, follower.Pos = false, "増援配下", Pos{7, 0}
	for b.liveEnemies() < maxLiveEnemies-1 {
		u := b.units[3]
		b.units = append(b.units, u)
	}
	b.mapEvents = []MapEventSetup{{Enemies: []UnitSetup{leader, follower}}}
	b.mapState.reservations = []reservation{{Turn: 2, Event: 0}}
	b.Turn = 2
	events := b.placeEventReinforcements(nil)
	if len(events) != 1 || !b.units[events[0].Unit].Commander || b.units[events[0].Unit].Collapsed {
		t.Fatalf("小隊長だけの増援: %+v", events)
	}
}

// 崩れた印と小隊長の生死を巻き戻しで復元する（TC-FR-BTL-10-35、DES-AI-03）。
func checkCommanderRestoreIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	b.units[1].Pos = Pos{1, 1}
	b.units[0].Stats.Str = 20
	if _, err := b.Act(Command{Unit: 0, To: Pos{0, 1}, Action: Attack, Target: 1}); err != nil {
		t.Fatal(err)
	}
	if b.units[1].Alive || !b.units[2].Collapsed {
		t.Fatal("撃破で崩れない")
	}
	if err := b.Rewind(b.History()[0].ID); err != nil {
		t.Fatal(err)
	}
	if !b.units[1].Alive || b.units[2].Collapsed || len(b.groupCollapsed) != 0 {
		t.Fatal("小隊長と配下が戻らない")
	}
	if _, err := b.Act(Command{Unit: 0, To: Pos{0, 1}, Action: Attack, Target: 1}); err != nil {
		t.Fatal(err)
	}
	if b.units[1].Alive || !b.units[2].Collapsed {
		t.Fatal("同じ入力で崩れない")
	}
}

// 同じ名前でも通れない壁の向こうの一団は崩さない（TC-FR-BTL-07-24、DES-AI-03）。
func checkCommanderSeparateRegionIt07(t *testing.T) {
	b := commanderBattleIt07(t)
	for y := range b.terrain {
		b.terrain[y][4].Cost = 0
	}
	b.units[1].Pos, b.units[2].Pos = Pos{3, 1}, Pos{2, 1}
	eastLeader := b.units[1]
	eastLeader.Pos = Pos{6, 1}
	eastFollower := b.units[2]
	eastFollower.Pos = Pos{5, 1}
	b.units = append(b.units, eastLeader, eastFollower)
	b.fall(nil, 1)
	if !b.units[2].Collapsed || b.units[4].Collapsed || b.units[5].Collapsed {
		t.Fatalf("領域を混同した: %+v / %+v / %+v", b.units[2], b.units[4], b.units[5])
	}
}

// 小隊長の規則を試験ケースごとに実行する（DES-AI-03）。
func TestCommanderRulesIt07(t *testing.T) {
	t.Run("TC-FR-BTL-07-24 撃破直後", checkCommanderCollapseIt07)
	t.Run("TC-FR-BTL-07-24 未突入の残った集合先", checkCommanderUnenteredIt07)
	t.Run("TC-FR-BTL-07-24 次の敵番の計画と予約", checkCommanderNextPhaseIt07)
	t.Run("TC-FR-BTL-07-24 崩れた仲間を数えない", checkCommanderGroupMoveIt07)
	t.Run("TC-FR-BTL-07-24 小隊長も集合判断に参加", checkCommanderLeaderPlannedIt07)
	t.Run("TC-FR-BTL-07-24 離れた同名の一団", checkCommanderSeparateRegionIt07)
	t.Run("TC-FR-BTL-07-25 敵番途中", checkCommanderMidPhaseIt07)
	t.Run("TC-FR-BTL-07-26 上限", checkCommanderReinforcementIt07)
	t.Run("TC-FR-BTL-07-26 出来事", checkCommanderEventReinforcementIt07)
	t.Run("TC-FR-BTL-10-35 巻き戻し", checkCommanderRestoreIt07)
}

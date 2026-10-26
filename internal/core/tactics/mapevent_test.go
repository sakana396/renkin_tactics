package tactics_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 出来事の増援・突撃（DES-MAP-07）と、生きている敵 39 体の上限（DES-IT06 9.3 節）の試験。

// eventSetup は 8 × 8 の平地の盤面を作る（出来事はテストごとに足す）。
//
//	0: 主人公 (1, 0) 移動 3、レベル 1（経験を得る）
//	1: 味方   (0, 7) 移動 3
//	2: ボス   (7, 7) ID "boss"、待ち構える、移動 0
//	3: 門番   (4, 0) ID "guard"、一団 "gate"、待ち構える、移動 0
func eventSetup() tactics.Setup {
	l := lord(1, 0, 3)
	l.Level = 1
	b := boss(7, 7)
	b.ID = "boss"
	g := foe("門番", 4, 0)
	g.ID = "guard"
	g.Group = "gate"
	g.AI = tactics.Hold
	g.Stats.Mov = 0
	return tactics.Setup{
		Terrain: grid("........", "........", "........", "........", "........", "........", "........", "........"),
		Units:   []tactics.UnitSetup{l, ally("味方", 0, 7, 3), b, g},
	}
}

// reinforceEvent は、きっかけ trigger（unit の敵）で、(x, y) に増援 1 体を出す出来事を作る。
func reinforceEvent(id string, trigger tactics.TriggerKind, unitID string, x, y int) tactics.MapEventSetup {
	r := foe("増援 "+id, x, y)
	return tactics.MapEventSetup{ID: id, Trigger: trigger, Unit: unitID, Effect: tactics.EffectReinforce, Enemies: []tactics.UnitSetup{r}}
}

// triggers は EventTrigger の出来事の ID の列を返す。
func triggers(events []tactics.Event) []string {
	var out []string
	for _, e := range events {
		if e.Kind == tactics.EventTrigger {
			out = append(out, e.MapEvent)
		}
	}
	return out
}

// newEventBattle は戦闘を作り、乱数を固定の値 roll にする。
func newEventBattle(t *testing.T, s tactics.Setup, roll int) *tactics.Battle {
	t.Helper()
	b := mustNew(t, s)
	b.SetRoll(func() int { return roll })
	return b
}

func TestMapEventSetupValidation(t *testing.T) {
	t.Run("TC-FR-MAP-03-16 初めの敵 39 体は受理し 40 体は拒む", func(t *testing.T) {
		for _, n := range []int{38, 39, 40} {
			t.Run(fmt.Sprintf("敵 %d 体", n), func(t *testing.T) {
				rows := make([]string, 8)
				for i := range rows {
					rows[i] = "........"
				}
				s := tactics.Setup{Terrain: grid(rows...), Units: []tactics.UnitSetup{lord(0, 0, 3), boss(7, 7)}}
				for i := 0; len(s.Units)-1 < n; i++ {
					x, y := i%8, 1+i/8
					s.Units = append(s.Units, foe(fmt.Sprintf("敵 %d", i), x, y))
				}
				b, events, err := tactics.New(s, 1)
				if n <= 39 {
					if err != nil || b == nil || len(events) == 0 {
						t.Fatalf("%d 体を拒んだ: %v", n, err)
					}
					return
				}
				if !errors.Is(err, tactics.ErrInvalidSetup) || b != nil || events != nil {
					t.Fatalf("%d 体を受理した: %v", n, err)
				}
				if _, err := tactics.NewBoardView(s); !errors.Is(err, tactics.ErrInvalidSetup) {
					t.Fatalf("NewBoardView が %d 体を受理した: %v", n, err)
				}
			})
		}
	})

	t.Run("TC-FR-MAP-03-20 出来事の定義の誤りを拒む", func(t *testing.T) {
		ok := reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 7, 0)
		staff := foe("杖", 7, 0)
		staff.Weapon = tactics.Weapon{Name: "杖", Type: tactics.Staff, Heal: 10, MaxUses: 3, MinRange: 1, MaxRange: 1}
		cases := []struct {
			name   string
			edit   func(s *tactics.Setup)
			reject bool
		}{
			{"正しい増援", func(s *tactics.Setup) { s.Events = []tactics.MapEventSetup{ok} }, false},
			{"正しい範囲と突撃", func(s *tactics.Setup) {
				s.Events = []tactics.MapEventSetup{{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(0, 0), Max: pos(7, 7)}, Effect: tactics.EffectChargeGroup, Group: "gate"}}
			}, false},
			{"増援の一団への突撃", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{foe("増援", 7, 0)}
				e.Enemies[0].Group = "later"
				s.Events = []tactics.MapEventSetup{e, {ID: "e2", Trigger: tactics.TriggerEnemyMoved, Unit: "guard", Effect: tactics.EffectChargeGroup, Group: "later"}}
			}, false},
			{"出来事の ID の重なり", func(s *tactics.Setup) { s.Events = []tactics.MapEventSetup{ok, ok} }, true},
			{"出来事の ID が空", func(s *tactics.Setup) { e := ok; e.ID = ""; s.Events = []tactics.MapEventSetup{e} }, true},
			{"敵の ID の重なり", func(s *tactics.Setup) { s.Units[2].ID = "guard" }, true},
			{"増援と初めの敵の ID の重なり", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{foe("増援", 7, 0)}
				e.Enemies[0].ID = "guard"
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"参照先の敵がいない", func(s *tactics.Setup) { e := ok; e.Unit = "nobody"; s.Events = []tactics.MapEventSetup{e} }, true},
			{"敵のきっかけに範囲", func(s *tactics.Setup) {
				e := ok
				e.Area = tactics.Area{Max: pos(1, 1)}
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"範囲のきっかけに敵", func(s *tactics.Setup) {
				e := ok
				e.Trigger = tactics.TriggerPlayerEntered
				e.Area = tactics.Area{Max: pos(1, 1)}
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"範囲の向きが逆", func(s *tactics.Setup) {
				s.Events = []tactics.MapEventSetup{{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(3, 3), Max: pos(2, 3)}, Effect: tactics.EffectChargeGroup, Group: "gate"}}
			}, true},
			{"範囲が盤の外", func(s *tactics.Setup) {
				s.Events = []tactics.MapEventSetup{{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(0, 0), Max: pos(8, 7)}, Effect: tactics.EffectChargeGroup, Group: "gate"}}
			}, true},
			{"ボスの増援", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{boss(7, 0)}
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"杖を持つ増援", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{staff}
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"正しくない能力", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{foe("増援", 7, 0)}
				e.Enemies[0].Stats.HP = 0
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"増援が盤の外", func(s *tactics.Setup) {
				e := ok
				e.Enemies = []tactics.UnitSetup{foe("増援", 8, 0)}
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"増援が 0 体", func(s *tactics.Setup) { e := ok; e.Enemies = nil; s.Events = []tactics.MapEventSetup{e} }, true},
			{"増援に一団", func(s *tactics.Setup) { e := ok; e.Group = "gate"; s.Events = []tactics.MapEventSetup{e} }, true},
			{"突撃の一団がいない", func(s *tactics.Setup) {
				s.Events = []tactics.MapEventSetup{{ID: "e1", Trigger: tactics.TriggerEnemyMoved, Unit: "guard", Effect: tactics.EffectChargeGroup, Group: "none"}}
			}, true},
			{"突撃に増援", func(s *tactics.Setup) {
				e := ok
				e.Effect = tactics.EffectChargeGroup
				e.Group = "gate"
				s.Events = []tactics.MapEventSetup{e}
			}, true},
			{"知らないきっかけ", func(s *tactics.Setup) { e := ok; e.Trigger = 9; s.Events = []tactics.MapEventSetup{e} }, true},
			{"知らない効果", func(s *tactics.Setup) { e := ok; e.Effect = 9; s.Events = []tactics.MapEventSetup{e} }, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := eventSetup()
				tc.edit(&s)
				_, _, err := tactics.New(s, 1)
				if tc.reject != errors.Is(err, tactics.ErrInvalidSetup) || !tc.reject && err != nil {
					t.Fatalf("拒む=%v、誤り=%v", tc.reject, err)
				}
			})
		}
	})
}

func TestMapEventTriggers(t *testing.T) {
	t.Run("TC-FR-MAP-03-17 4 つのきっかけと外れ・0 ダメージ・反撃・経路", func(t *testing.T) {
		attackGuard := tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3}
		cases := []struct {
			name  string
			event tactics.MapEventSetup
			edit  func(s *tactics.Setup)
			roll  int
			run   func(t *testing.T, b *tactics.Battle) []tactics.Event
			fired bool
			phase tactics.Side
		}{
			{name: "攻撃された（外れ）", event: reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 7, 0), roll: 99,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustAct(t, b, attackGuard) }, fired: true},
			{name: "攻撃された（0 ダメージ）", event: reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 7, 0), roll: 0,
				// 乱数 0 では必殺も出るので（DES-BTL-09、#120）、主人公の幸運を上げて門番の反撃の必殺を 0 % にする。
				edit: func(s *tactics.Setup) { s.Units[3].Stats.Def = 50; s.Units[0].Stats.Lck = 50 },
				run:  func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustAct(t, b, attackGuard) }, fired: true},
			{name: "自分から攻撃して反撃された", event: reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 7, 0), roll: 99,
				edit: func(s *tactics.Setup) { s.Units[0].Pos = pos(3, 0) },
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					b.SetEnemyTarget(func(u int) int {
						if u == 3 {
							return 0
						}
						return -1
					})
					return mustEndPhase(t, b)
				}, fired: true, phase: tactics.Enemy},
			{name: "反撃できなければ攻撃されていない", event: reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 7, 0), roll: 99,
				edit: func(s *tactics.Setup) {
					s.Units[0].Pos = pos(3, 0)
					s.Units[0].Weapon = tactics.Weapon{Name: "弓", Type: tactics.Bow, Might: 5, Hit: 90, MinRange: 2, MaxRange: 2}
				},
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					b.SetEnemyTarget(func(u int) int {
						if u == 3 {
							return 0
						}
						return -1
					})
					return mustEndPhase(t, b)
				}, fired: false},
			{name: "動いた", event: reinforceEvent("e1", tactics.TriggerEnemyMoved, "guard", 7, 0), roll: 99,
				edit: func(s *tactics.Setup) { s.Units[3].AI = tactics.Charge; s.Units[3].Stats.Mov = 1 },
				run:  func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustEndPhase(t, b) }, fired: true, phase: tactics.Enemy},
			{name: "指定していない敵だけが動いても起きない", event: reinforceEvent("e1", tactics.TriggerEnemyMoved, "guard", 7, 0), roll: 99,
				edit: func(s *tactics.Setup) {
					other := foe("歩兵", 4, 3)
					other.ID = "other"
					s.Units = append(s.Units, other)
				},
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					events := mustEndPhase(t, b)
					if indexOf(events, tactics.EventMove) < 0 {
						t.Fatalf("指定していない敵が動いていない: %v", kinds(events))
					}
					return events
				}, fired: false},
			{name: "待機は動いていない", event: reinforceEvent("e1", tactics.TriggerEnemyMoved, "guard", 7, 0), roll: 99,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustEndPhase(t, b) }, fired: false},
			{name: "倒れた", event: reinforceEvent("e1", tactics.TriggerEnemyDefeated, "guard", 7, 0), roll: 0,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					b.SetHP(3, 1)
					return mustAct(t, b, attackGuard)
				}, fired: true},
			{name: "倒れていない", event: reinforceEvent("e1", tactics.TriggerEnemyDefeated, "guard", 7, 0), roll: 0,
				// 乱数 0 では必殺も出るので（DES-BTL-09、#120）、門番の幸運を上げて主人公の必殺を 0 % にする。
				edit: func(s *tactics.Setup) { s.Units[3].Stats.Lck = 50 },
				run:  func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustAct(t, b, attackGuard) }, fired: false},
			{name: "範囲に着いた", event: tactics.MapEventSetup{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(2, 0), Max: pos(2, 7)}, Effect: tactics.EffectChargeGroup, Group: "gate"}, roll: 99,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					return mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Wait})
				}, fired: true},
			{name: "経路で通っただけ", event: tactics.MapEventSetup{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(2, 0), Max: pos(2, 7)}, Effect: tactics.EffectChargeGroup, Group: "gate"}, roll: 99,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					return mustAct(t, b, tactics.Command{Unit: 0, To: pos(3, 1), Action: tactics.Wait})
				}, fired: false},
			{name: "初めの配置で範囲の中にいて動かない", event: tactics.MapEventSetup{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(0, 6), Max: pos(1, 7)}, Effect: tactics.EffectChargeGroup, Group: "gate"}, roll: 99,
				run: func(t *testing.T, b *tactics.Battle) []tactics.Event {
					return mustAct(t, b, tactics.Command{Unit: 1, To: pos(0, 7), Action: tactics.Wait})
				}, fired: false},
			{name: "敵が範囲に着いても起きない", event: tactics.MapEventSetup{ID: "e1", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(3, 0), Max: pos(3, 0)}, Effect: tactics.EffectChargeGroup, Group: "gate"}, roll: 99,
				edit: func(s *tactics.Setup) { s.Units[3].AI = tactics.Charge; s.Units[3].Stats.Mov = 1 },
				run:  func(t *testing.T, b *tactics.Battle) []tactics.Event { return mustEndPhase(t, b) }, fired: false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := eventSetup()
				if tc.edit != nil {
					tc.edit(&s)
				}
				s.Events = []tactics.MapEventSetup{tc.event}
				b := newEventBattle(t, s, tc.roll)
				events := tc.run(t, b)
				got := triggers(events)
				if !tc.fired {
					if len(got) != 0 {
						t.Fatalf("起きた: %v", got)
					}
					return
				}
				if len(got) != 1 || got[0] != "e1" {
					t.Fatalf("きっかけ = %v、出来事 = %v", got, kinds(events))
				}
				// きっかけは、原因の出来事・経験などを済ませたあとに出す（DES-IT06 9.2 節）。
				i := indexOf(events, tactics.EventTrigger)
				for j, e := range events {
					cause := (e.Kind == tactics.EventStrike || e.Kind == tactics.EventMove || e.Kind == tactics.EventFall || e.Kind == tactics.EventExp) && e.Turn == 0
					if cause && j > i && (e.Unit == 3 || e.Unit == 0 && tc.phase == tactics.Player) {
						t.Fatalf("きっかけのあとに原因の出来事: %v", kinds(events))
					}
				}
				// 自軍の行動では、きっかけのあとは効果（突撃）だけ。増援は予約するだけで、すぐには出ない。
				for _, e := range events[i+1:] {
					if tc.phase == tactics.Player && e.Kind != tactics.EventGroupCharge {
						t.Fatalf("自軍の行動のきっかけのあとに効果以外: %v", kinds(events))
					}
				}
			})
		}
	})

	t.Run("TC-FR-MAP-03-17 出来事の順・1 回だけ・同じ区切りで複数・勝敗後", func(t *testing.T) {
		s := eventSetup()
		s.Units[3].Stats.HP = 1
		s.Events = []tactics.MapEventSetup{
			reinforceEvent("defeated", tactics.TriggerEnemyDefeated, "guard", 7, 0),
			reinforceEvent("attacked", tactics.TriggerEnemyAttacked, "guard", 6, 0),
			reinforceEvent("other", tactics.TriggerEnemyAttacked, "boss", 5, 0),
		}
		b := newEventBattle(t, s, 0)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3})
		// 観測の順（攻撃 → 倒れた）ではなく、定義の順に起こす。
		assertEqual(t, "きっかけ", triggers(events), []string{"defeated", "attacked"})
		want := []tactics.EventKind{tactics.EventMove, tactics.EventStrike, tactics.EventFall, tactics.EventExp, tactics.EventTrigger, tactics.EventTrigger}
		assertEqual(t, "出来事の順", kinds(events), want)

		// 1 回だけ: 別の味方がもう一度攻撃しても、2 回目は起きない。
		s2 := eventSetup()
		s2.Events = []tactics.MapEventSetup{reinforceEvent("once", tactics.TriggerEnemyAttacked, "guard", 7, 0)}
		s2.Units[1].Pos = pos(4, 2)
		b2 := newEventBattle(t, s2, 99)
		first := mustAct(t, b2, tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3})
		second := mustAct(t, b2, tactics.Command{Unit: 1, To: pos(4, 1), Action: tactics.Attack, Target: 3})
		if len(triggers(first)) != 1 || len(triggers(second)) != 0 {
			t.Fatalf("1 回だけではない: %v / %v", triggers(first), triggers(second))
		}

		// 勝敗が決まったら起こさない。
		s3 := eventSetup()
		s3.Units[2].Pos = pos(2, 1)
		s3.Units[2].Stats.HP = 1
		s3.Events = []tactics.MapEventSetup{reinforceEvent("win", tactics.TriggerEnemyDefeated, "boss", 7, 0)}
		b3 := newEventBattle(t, s3, 0)
		won := mustAct(t, b3, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 2})
		if b3.Result != tactics.Won || len(triggers(won)) != 0 {
			t.Fatalf("勝敗後に起きた: 結果 %v、%v", b3.Result, kinds(won))
		}
	})
}

// indexOf は kind の最初の出来事の添字を返す。なければ -1。
func indexOf(events []tactics.Event, kind tactics.EventKind) int {
	for i, e := range events {
		if e.Kind == kind {
			return i
		}
	}
	return -1
}

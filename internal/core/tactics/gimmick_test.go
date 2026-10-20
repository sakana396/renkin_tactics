package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 砦（DES-MAP-02）と増援（DES-MAP-03）、番の始めの出来事の順（DES-BTL-02）のテスト。

// mustEndPhase は誤りのない EndPhase を行う。
func mustEndPhase(t *testing.T, b *tactics.Battle) []tactics.Event {
	t.Helper()
	events, err := b.EndPhase()
	if err != nil {
		t.Fatalf("EndPhase: %v", err)
	}
	return events
}

// ev は出来事の比べる部分（種類、陣営、ターン、ユニット、マス、量、アイテム）。
type ev struct {
	Kind   tactics.EventKind
	Side   tactics.Side
	Turn   int
	Unit   int
	To     tactics.Pos
	Amount int
	Item   string
}

func evs(events []tactics.Event) []ev {
	out := make([]ev, len(events))
	for i, e := range events {
		out[i] = ev{e.Kind, e.Side, e.Turn, e.Unit, e.To, e.Amount, e.Item}
	}
	return out
}

func phaseStart(side tactics.Side, turn int) ev {
	return ev{Kind: tactics.EventPhaseStart, Side: side, Turn: turn}
}

func healEv(unit, amount int) ev { return ev{Kind: tactics.EventHeal, Unit: unit, Amount: amount} }

func reinforceEv(unit int, to tactics.Pos) ev {
	return ev{Kind: tactics.EventReinforce, Unit: unit, To: to}
}

// fortSetup は 4 隅が砦の 5 × 5 の盤面に、自軍 2 体と敵 2 体を置く。
//
//	T . . . T
//	. . . . .
//	. . . . .
//	. . . . .
//	T . . . T
//
//	0: 主人公 (0, 0) 砦
//	1: 味方   (1, 4) 平地
//	2: ボス   (4, 4) 砦
//	3: 敵     (4, 0) 砦
func fortSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: grid("T...T", ".....", ".....", ".....", "T...T"),
		Units: []tactics.UnitSetup{
			lord(0, 0, 2),
			ally("味方", 1, 4, 2),
			boss(4, 4),
			foe("敵", 4, 0),
		},
	}
}

// TestFort は砦の回復を確かめる（DES-MAP-02）。
func TestFort(t *testing.T) {
	t.Run("TC-FR-MAP-03-01 砦にいるユニットは自分の陣営の番の始めに回復する", func(t *testing.T) {
		b := mustNew(t, fortSetup())
		b.SetHP(0, 10)
		b.SetHP(1, 10)
		b.SetHP(3, 10)
		events := mustEndPhase(t, b)
		// 敵は敵軍の番の始め、主人公は自軍の番の始めに 20 × 20% = 4 回復する。平地の味方は回復しない。
		assertEqual(t, "出来事", evs(events), []ev{
			phaseStart(tactics.Enemy, 1), healEv(3, 4),
			phaseStart(tactics.Player, 2), healEv(0, 4),
		})
		assertEqual(t, "敵の Heal の写しの HP", events[1].Units[3].HP, 14)
		hp := []int{}
		for _, u := range b.Units() {
			hp = append(hp, u.HP)
		}
		assertEqual(t, "HP", hp, []int{14, 10, 20, 14})
	})

	t.Run("TC-FR-MAP-03-02 回復は切り捨てで最大 HP を超えず、0 なら出来事を出さない", func(t *testing.T) {
		cases := []struct {
			name   string
			maxHP  int
			hp     int
			kill   bool
			want   []ev
			wantHP int
		}{
			{"最大 HP 24 は 4.8 を切り捨てて 4", 24, 1, false, []ev{healEv(1, 4)}, 5},
			{"最大 HP 25 はちょうど 5", 25, 1, false, []ev{healEv(1, 5)}, 6},
			{"最大までの 1 だけ回復", 24, 23, false, []ev{healEv(1, 1)}, 24},
			{"HP が最大なら出さない", 24, 24, false, nil, 24},
			{"最大 HP 4 は 0.8 を切り捨てて 0 なので出さない", 4, 1, false, nil, 1},
			{"倒れたユニットは回復しない", 24, 1, true, nil, 0},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				s := fortSetup()
				// 味方を砦 (0, 4) に置く。
				s.Units[1].Pos = pos(0, 4)
				s.Units[1].Stats.HP = tt.maxHP
				b := mustNew(t, s)
				b.SetHP(1, tt.hp)
				if tt.kill {
					b.KillUnit(1)
				}
				events := mustEndPhase(t, b)
				want := append([]ev{phaseStart(tactics.Enemy, 1), phaseStart(tactics.Player, 2)}, tt.want...)
				assertEqual(t, "出来事", evs(events), want)
				assertEqual(t, "HP", b.Units()[1].HP, tt.wantHP)
			})
		}
	})
}

// TestReinforcements は増援を確かめる（DES-MAP-03）。
func TestReinforcements(t *testing.T) {
	t.Run("TC-FR-MAP-03-03 増援はそのターンの敵軍の番の始めに現れる", func(t *testing.T) {
		s := baseSetup()
		s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 2, 2)}}
		b := mustNew(t, s)

		events := mustEndPhase(t, b)
		assertEqual(t, "1 ターン目", evs(events), []ev{phaseStart(tactics.Enemy, 1), phaseStart(tactics.Player, 2)})
		assertEqual(t, "1 ターン目のユニットの数", len(b.Units()), 4)

		events = mustEndPhase(t, b)
		assertEqual(t, "2 ターン目", evs(events), []ev{
			phaseStart(tactics.Enemy, 2), reinforceEv(4, pos(2, 2)), phaseStart(tactics.Player, 3),
		})
		if n := len(events[1].Units); n != 5 {
			t.Errorf("Reinforce の写しのユニットの数 = %d, want 5", n)
		}
		units := b.Units()
		if len(units) != 5 {
			t.Fatalf("ユニットの数 = %d, want 5", len(units))
		}
		r := units[4]
		if r.Name != "増援" || r.Side != tactics.Enemy || r.Pos != pos(2, 2) || r.HP != 20 || !r.Alive {
			t.Errorf("増援 = %+v", r)
		}

		events = mustEndPhase(t, b)
		assertEqual(t, "3 ターン目", evs(events), []ev{phaseStart(tactics.Enemy, 3), phaseStart(tactics.Player, 4)})
		assertEqual(t, "3 ターン目のユニットの数", len(b.Units()), 5)
	})

	t.Run("TC-FR-MAP-03-04 現れた増援はその番には動かない", func(t *testing.T) {
		s := baseSetup()
		s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 2, 2)}}
		b := mustNew(t, s)
		var acted []int
		b.SetEnemyTarget(func(unit int) int {
			acted = append(acted, unit)
			return -1
		})
		for turn := 1; turn <= 3; turn++ {
			acted = nil
			mustEndPhase(t, b)
			want := []int{2, 3}
			if turn == 3 {
				want = []int{2, 3, 4}
			}
			assertEqual(t, "動いた敵", acted, want)
		}
	})

	t.Run("TC-FR-MAP-03-05 現れるマスにユニットがいれば、増援はそのターンにも後のターンにも現れない", func(t *testing.T) {
		s := baseSetup()
		s.Reinforcements = []tactics.ReinforcementSetup{
			// 主人公のいるマス。
			{Turn: 2, Unit: foe("ふさがれた増援", 0, 0)},
			// 同じターンに同じマスに 2 体。2 体目は 1 体目にふさがれる。
			{Turn: 2, Unit: foe("増援 1", 2, 2)},
			{Turn: 2, Unit: foe("増援 2", 2, 2)},
		}
		b := mustNew(t, s)
		mustEndPhase(t, b)
		events := mustEndPhase(t, b)
		assertEqual(t, "2 ターン目", evs(events), []ev{
			phaseStart(tactics.Enemy, 2), reinforceEv(4, pos(2, 2)), phaseStart(tactics.Player, 3),
		})
		assertEqual(t, "増援の名前", b.Units()[4].Name, "増援 1")

		// 主人公がマスを空けても、後のターンには現れない。
		mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 0), Action: tactics.Wait})
		mustEndPhase(t, b)
		mustEndPhase(t, b)
		assertEqual(t, "ユニットの数", len(b.Units()), 5)
	})

	t.Run("TC-FR-MAP-03-06 倒れたユニットのいたマスには増援が現れる", func(t *testing.T) {
		s := baseSetup()
		s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 0, 4)}}
		b := mustNew(t, s)
		b.KillUnit(1)
		mustEndPhase(t, b)
		events := mustEndPhase(t, b)
		assertEqual(t, "2 ターン目", evs(events), []ev{
			phaseStart(tactics.Enemy, 2), reinforceEv(4, pos(0, 4)), phaseStart(tactics.Player, 3),
		})
	})

	t.Run("TC-FR-MAP-03-07 敵軍の番の始めの出来事は PhaseStart → Reinforce（定義の順）→ Heal（添字の順）", func(t *testing.T) {
		s := fortSetup()
		s.Reinforcements = []tactics.ReinforcementSetup{
			{Turn: 3, Unit: foe("増援 C", 2, 3)},
			{Turn: 2, Unit: foe("増援 A", 3, 3)},
			{Turn: 2, Unit: foe("増援 B", 1, 1)},
		}
		b := mustNew(t, s)
		mustEndPhase(t, b)
		b.SetHP(2, 10)
		b.SetHP(3, 10)
		events := mustEndPhase(t, b)
		assertEqual(t, "2 ターン目", evs(events), []ev{
			phaseStart(tactics.Enemy, 2),
			reinforceEv(4, pos(3, 3)), reinforceEv(5, pos(1, 1)),
			healEv(2, 4), healEv(3, 4),
			phaseStart(tactics.Player, 3),
		})
		events = mustEndPhase(t, b)
		assertEqual(t, "3 ターン目", evs(events), []ev{
			phaseStart(tactics.Enemy, 3), reinforceEv(6, pos(2, 3)),
			healEv(2, 4), healEv(3, 4),
			phaseStart(tactics.Player, 4),
		})
	})
}

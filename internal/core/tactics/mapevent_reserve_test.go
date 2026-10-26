package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 出来事の増援の予約・予兆・巻き戻し（DES-IT06 9.3 節）と、一団の突撃（DES-MAP-07）の試験。

// reserveSetup は TC-FR-MAP-03-18 の盤面。ボスは砦にいて、2 ターン目にターンの増援が (6, 0) に、
// 門番が攻撃されると出来事の増援が (0, 4) に出る。
func reserveSetup() tactics.Setup {
	s := eventSetup()
	s.Terrain = grid("........", "........", "........", "........", "........", "........", "........", ".......T")
	s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("ターン増援", 6, 0)}}
	s.Events = []tactics.MapEventSetup{reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 0, 4)}
	return s
}

// hintOf は元のマス p の増援の予兆の長方形を、DES-RNF-01 の式で独立に求める（8 × 8 の盤）。
func hintOf(p tactics.Pos) tactics.ReinforcementHint {
	lo := func(v int) int { return max(0, v-2) / 4 * 4 }
	hi := func(v int) int { return min(7, (min(7, v+2)/4+1)*4-1) }
	return tactics.ReinforcementHint{Min: pos(lo(p.X), lo(p.Y)), Max: pos(hi(p.X), hi(p.Y))}
}

func TestMapEventReservation(t *testing.T) {
	attackGuard := tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3}

	t.Run("TC-FR-MAP-03-18 自軍の番の予約は今のターン・出る順・その番に動かない・予兆", func(t *testing.T) {
		b := newEventBattle(t, reserveSetup(), 99)
		mustEndPhase(t, b)
		if b.Turn != 2 {
			t.Fatalf("ターン %d", b.Turn)
		}
		b.SetHP(2, 10)
		assertEqual(t, "予約の前の予兆", b.BoardView().ReinforcementHints(), []tactics.ReinforcementHint{hintOf(pos(6, 0))})
		events := mustAct(t, b, attackGuard)
		assertEqual(t, "きっかけ", triggers(events), []string{"e1"})
		if n := len(b.Units()); n != 4 {
			t.Fatalf("予約の時点で出た: %d 体", n)
		}
		// 予約したときから、(0, 4) の予兆を足す（左上 Y・X の順なので先に並ぶ）。
		assertEqual(t, "予約のあとの予兆", b.BoardView().ReinforcementHints(), []tactics.ReinforcementHint{hintOf(pos(0, 4)), hintOf(pos(6, 0))})

		events = mustEndPhase(t, b)
		start := indexOf(events, tactics.EventPhaseStart)
		assertEqual(t, "敵軍の番の始め", kinds(events[start:start+4]), []tactics.EventKind{tactics.EventPhaseStart, tactics.EventReinforce, tactics.EventReinforce, tactics.EventHeal})
		assertEqual(t, "ターンの増援", []any{events[start+1].Unit, events[start+1].To}, []any{4, pos(6, 0)})
		assertEqual(t, "出来事の増援", []any{events[start+2].Unit, events[start+2].To}, []any{5, pos(0, 4)})
		assertEqual(t, "砦の回復", events[start+3].Unit, 2)
		for _, e := range events[start+4:] {
			if (e.Kind == tactics.EventMove || e.Kind == tactics.EventStrike || e.Kind == tactics.EventWait) && e.Unit >= 4 {
				t.Fatalf("その番に出た増援が動いた: %+v", e)
			}
		}
		if b.Turn != 3 || len(b.BoardView().ReinforcementHints()) != 0 {
			t.Fatalf("出たあとも予兆が残る: ターン %d、%v", b.Turn, b.BoardView().ReinforcementHints())
		}
		// 次のターンに同じ増援は出ない（1 回だけ）。
		events = mustEndPhase(t, b)
		if indexOf(events, tactics.EventReinforce) >= 0 || len(b.Units()) != 6 {
			t.Fatalf("もう一度出た: %v", kinds(events))
		}
	})

	t.Run("TC-FR-MAP-03-18 敵軍の番の予約は次のターン", func(t *testing.T) {
		s := reserveSetup()
		s.Units[0].Pos = pos(3, 0)
		b := newEventBattle(t, s, 99)
		// 門番が主人公を攻撃し、主人公の反撃で起きる。
		events := mustEndPhase(t, b)
		i := indexOf(events, tactics.EventTrigger)
		if i < 0 || events[i].MapEvent != "e1" {
			t.Fatalf("敵軍の番に起きない: %v", kinds(events))
		}
		for _, e := range events[i:] {
			if e.Kind == tactics.EventReinforce {
				t.Fatalf("同じ番に出た: %v", kinds(events))
			}
		}
		if b.Turn != 2 || len(b.Units()) != 4 {
			t.Fatalf("ターン %d、%d 体", b.Turn, len(b.Units()))
		}
		assertEqual(t, "次の自軍の番の予兆", b.BoardView().ReinforcementHints(), []tactics.ReinforcementHint{hintOf(pos(0, 4)), hintOf(pos(6, 0))})
		events = mustEndPhase(t, b)
		start := indexOf(events, tactics.EventPhaseStart)
		assertEqual(t, "2 ターン目の敵軍の番の始め", kinds(events[start:start+3]), []tactics.EventKind{tactics.EventPhaseStart, tactics.EventReinforce, tactics.EventReinforce})
		assertEqual(t, "出来事の増援", events[start+2].To, pos(0, 4))
	})

	t.Run("TC-FR-MAP-03-18 出る場所は距離 0〜2・Y・X の順", func(t *testing.T) {
		s := reserveSetup()
		s.Units[1].Pos = pos(0, 4)
		b := newEventBattle(t, s, 99)
		mustAct(t, b, attackGuard)
		events := mustEndPhase(t, b)
		i := indexOf(events, tactics.EventReinforce)
		if i < 0 || events[i].To != pos(0, 3) {
			t.Fatalf("出た場所: %v", kinds(events))
		}
	})

	t.Run("TC-FR-MAP-03-18 巻き戻しと履歴の下見で起きたかと予約を戻す", func(t *testing.T) {
		b := newEventBattle(t, reserveSetup(), 99)
		mustEndPhase(t, b)
		mustAct(t, b, attackGuard)
		h := b.History()
		id := h[len(h)-1].ID
		before := []tactics.ReinforcementHint{hintOf(pos(6, 0))}
		reserved := []tactics.ReinforcementHint{hintOf(pos(0, 4)), hintOf(pos(6, 0))}
		v, err := b.PreviewRewind(id)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "下見の予兆", v.ReinforcementHints(), before)
		assertEqual(t, "下見のあとの本物の予兆", b.BoardView().ReinforcementHints(), reserved)
		if err := b.Rewind(id); err != nil {
			t.Fatal(err)
		}
		b.SetRoll(func() int { return 99 })
		assertEqual(t, "巻き戻しのあとの予兆", b.BoardView().ReinforcementHints(), before)
		events := mustAct(t, b, attackGuard)
		assertEqual(t, "もう一度起きる", triggers(events), []string{"e1"})
		events = mustEndPhase(t, b)
		n := 0
		for _, e := range events {
			if e.Kind == tactics.EventReinforce {
				n++
			}
		}
		if n != 2 {
			t.Fatalf("増援の数 %d: %v", n, kinds(events))
		}
	})
}

func TestMapEventChargeGroup(t *testing.T) {
	t.Run("TC-FR-BTL-07-19 出来事の突撃で一団が突撃に変わる", func(t *testing.T) {
		s := eventSetup()
		s.Units[3].Stats.Mov = 2
		second := foe("番兵", 5, 2)
		second.AI, second.Group = tactics.Hold, "gate"
		other := foe("別の一団", 6, 3)
		other.AI, other.Group = tactics.Hold, "other"
		s.Units = append(s.Units, second, other)
		later := foe("あとの番兵", 6, 0)
		later.AI, later.Group = tactics.Hold, "gate"
		s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: later}}
		s.Events = []tactics.MapEventSetup{{ID: "advance", Trigger: tactics.TriggerPlayerEntered, Area: tactics.Area{Min: pos(0, 2), Max: pos(1, 2)}, Effect: tactics.EffectChargeGroup, Group: "gate"}}
		b := newEventBattle(t, s, 99)

		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 2), Action: tactics.Wait})
		assertEqual(t, "出来事の順", kinds(events), []tactics.EventKind{tactics.EventMove, tactics.EventWait, tactics.EventTrigger, tactics.EventGroupCharge})
		g := events[3]
		assertEqual(t, "突撃の出来事", []string{g.MapEvent, g.Group}, []string{"advance", "gate"})
		units := b.Units()
		assertEqual(t, "動き方", []tactics.AI{units[3].AI, units[4].AI, units[5].AI}, []tactics.AI{tactics.Charge, tactics.Charge, tactics.Hold})

		// 一団の敵は自軍に寄り、別の一団は動かない。
		mustAct(t, b, tactics.Command{Unit: 1, To: pos(0, 7), Action: tactics.Wait})
		after := b.Units()
		if after[3].Pos == pos(4, 0) || after[4].Pos == pos(5, 2) || after[5].Pos != pos(6, 3) {
			t.Fatalf("位置: %v %v %v", after[3].Pos, after[4].Pos, after[5].Pos)
		}
		// まだ出ていなかった同じ一団の敵は、出たときから突撃。
		events = mustEndPhase(t, b)
		i := indexOf(events, tactics.EventReinforce)
		if i < 0 || events[i].Units[events[i].Unit].AI != tactics.Charge {
			t.Fatalf("あとの番兵が突撃でない: %v", kinds(events))
		}
	})
}

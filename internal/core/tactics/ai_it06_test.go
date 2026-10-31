package tactics_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 一団の寄り方と敵騎兵の再移動（DES-AI-02、DES-BTL-10）。
func TestEnemyGroupIt06(t *testing.T) {
	t.Run("TC-FR-BTL-07-21 一団とまっすぐ寄る敵の混在", func(t *testing.T) {
		p := lord(1, 3, 1)
		p.Stats.Str = 15 // 反撃の通常ダメージが敵の HP 以上。
		front := enemy("前衛", 3, 3, 1)
		front.Group, front.Approach = "alpha", tactics.GroupApproach
		rear := enemy("後衛", 4, 3, 1)
		rear.Group, rear.Approach = "alpha", tactics.GroupApproach
		direct := enemy("直進", 6, 5, 1)
		s := plain7(p, front, rear, direct, boss(6, 0))
		b := mustNew(t, s)
		events := mustEndPhase(t, b)
		for _, e := range events {
			if e.Kind == tactics.EventStrike && (e.Unit == 1 || e.Unit == 2) {
				t.Fatalf("集合前に攻撃した: %+v", e)
			}
		}
		if b.Units()[3].Pos == direct.Pos {
			t.Fatal("まっすぐ寄る敵が動かなかった")
		}
	})

	t.Run("TC-FR-BTL-07-21 一団の入力の異常", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			edit func(*tactics.Setup)
		}{
			{"名前なし", func(s *tactics.Setup) { s.Units[1].Group = "" }},
			{"方式が不一致", func(s *tactics.Setup) { s.Units[2].Approach = tactics.Direct }},
			{"範囲外", func(s *tactics.Setup) { s.Units[1].Approach = tactics.GroupApproach + 1 }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := plain7(lord(1, 3, 1), enemy("前", 3, 3, 1), enemy("後", 4, 3, 1), boss(6, 0))
				for i := 1; i <= 2; i++ {
					s.Units[i].Group, s.Units[i].Approach = "alpha", tactics.GroupApproach
				}
				tc.edit(&s)
				if _, _, err := tactics.New(s, 1); err == nil {
					t.Fatal("誤った一団を受け入れた")
				}
			})
		}
	})
	t.Run("TC-FR-BTL-07-21 集合した次の番に入る", func(t *testing.T) {
		p := lord(1, 1, 1)
		p.Stats.Str = 20
		front := enemy("前衛", 3, 1, 2)
		rear := enemy("後衛", 5, 1, 2)
		front.Group, rear.Group = "g", "g"
		front.Approach, rear.Approach = tactics.GroupApproach, tactics.GroupApproach
		s := tactics.Setup{Terrain: grid("#######", ".......", "#######"), Units: []tactics.UnitSetup{p, front, rear, boss(6, 1)}}
		b := mustNew(t, s)
		first := mustEndPhase(t, b)
		for _, e := range first {
			if e.Kind == tactics.EventStrike && (e.Unit == 1 || e.Unit == 2) {
				t.Fatalf("そろう前に入った: %v", kinds(first))
			}
		}
		if b.Units()[1].Pos != pos(4, 1) || b.Units()[2].Pos != pos(5, 1) {
			t.Fatalf("集まる先 = %v / %v", b.Units()[1].Pos, b.Units()[2].Pos)
		}
		second := mustEndPhase(t, b)
		entered := false
		for _, e := range second {
			if e.Kind == tactics.EventStrike && e.Unit == 1 {
				entered = true
			}
		}
		if !entered {
			t.Fatalf("そろった次の番に入らなかった: %v", kinds(second))
		}
	})
	t.Run("TC-FR-BTL-07-21 戦死と未出現の仲間を待たない", func(t *testing.T) {
		for _, dead := range []bool{false, true} {
			t.Run(map[bool]string{false: "未出現", true: "戦死"}[dead], func(t *testing.T) {
				p := lord(1, 1, 1)
				p.Stats.Str = 20
				front := enemy("前衛", 3, 1, 2)
				rear := enemy("後衛", 5, 1, 2)
				front.Group, rear.Group = "g", "g"
				front.Approach, rear.Approach = tactics.GroupApproach, tactics.GroupApproach
				s := tactics.Setup{Terrain: grid("#######", ".......", "#######"), Units: []tactics.UnitSetup{p, front, boss(6, 1)}}
				if dead {
					s.Units = append(s.Units, rear)
				} else {
					s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 3, Unit: rear}}
				}
				b := mustNew(t, s)
				if dead {
					b.KillUnit(3)
				}
				mustEndPhase(t, b)
				if b.Units()[1].Pos != pos(4, 1) {
					t.Fatalf("前衛が集まらない: %v", b.Units()[1].Pos)
				}
				events := mustEndPhase(t, b)
				attacked := false
				for _, e := range events {
					if e.Kind == tactics.EventStrike && e.Unit == 1 {
						attacked = true
					}
				}
				if !attacked {
					t.Fatalf("いない仲間を待った: %v", kinds(events))
				}
			})
		}
	})
	t.Run("TC-FR-BTL-07-21 到達できない領域は別に判定する", func(t *testing.T) {
		p := lord(1, 1, 1)
		p.Stats.Str = 20
		upper := enemy("上", 3, 1, 2)
		lower := enemy("下", 3, 3, 2)
		upper.Group, lower.Group = "g", "g"
		upper.Approach, lower.Approach = tactics.GroupApproach, tactics.GroupApproach
		s := tactics.Setup{Terrain: grid("#######", ".......", "#######", ".......", "#######"), Units: []tactics.UnitSetup{p, upper, lower, boss(6, 3)}}
		b := mustNew(t, s)
		mustEndPhase(t, b)
		if b.Units()[1].Pos != pos(4, 1) {
			t.Fatalf("上の集まる先 = %v", b.Units()[1].Pos)
		}
		events := mustEndPhase(t, b)
		attacked := false
		for _, e := range events {
			if e.Kind == tactics.EventStrike && e.Unit == 1 {
				attacked = true
			}
		}
		if !attacked {
			t.Fatal("別の領域の仲間を待った")
		}
	})
	t.Run("TC-FR-BTL-07-21 安全なマスが足りなければ後衛は待つ", func(t *testing.T) {
		p := lord(0, 1, 0)
		p.Stats.Str = 20
		front := enemy("前衛", 2, 1, 2)
		rear := enemy("後衛", 1, 1, 2)
		front.Group, rear.Group = "g", "g"
		front.Approach, rear.Approach = tactics.GroupApproach, tactics.GroupApproach
		s := tactics.Setup{Terrain: grid("#####", "...##", "#####", ".....", "#####"), Units: []tactics.UnitSetup{p, front, rear, boss(4, 3)}}
		b := mustNew(t, s)
		events := mustEndPhase(t, b)
		if b.Units()[1].Pos != pos(2, 1) || b.Units()[2].Pos != pos(1, 1) {
			t.Fatalf("安全な範囲 = %v / %v", b.Units()[1].Pos, b.Units()[2].Pos)
		}
		for _, e := range events {
			if e.Kind == tactics.EventStrike && e.Unit == 2 {
				t.Fatalf("後衛が攻撃した: %v", kinds(events))
			}
		}
	})
}

func TestEnemyCantoIt06(t *testing.T) {
	setup := func() tactics.Setup {
		p := lord(1, 1, 0)
		e := enemy("騎兵", 4, 1, 4)
		e.ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
		e.Stats.Str = 2
		s := tactics.Setup{Terrain: grid("#######", ".......", "#######"), Units: []tactics.UnitSetup{p, e, boss(6, 1)}}
		return s
	}
	t.Run("TC-FR-BTL-07-22 攻撃後に残りの費用で下がる", func(t *testing.T) {
		b := mustNew(t, setup())
		events := mustEndPhase(t, b)
		var got []move
		for _, m := range moves(events) {
			if m.Unit == 1 {
				got = append(got, m)
			}
		}
		want := []move{{1, pos(4, 1), pos(2, 1)}, {1, pos(2, 1), pos(4, 1)}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("移動 = %v, want %v", got, want)
		}
		if _, ok := b.PendingCanto(); ok {
			t.Fatal("敵の再移動で入力待ちになった")
		}
		h := b.History()
		n := 0
		for _, entry := range h {
			if entry.Unit == 1 {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("敵の履歴 = %d 手", n)
		}
	})
	t.Run("TC-FR-BTL-07-22 森の費用を残りから引く", func(t *testing.T) {
		s := setup()
		s.Terrain = grid("#######", "...F...", "#######")
		s.Units[1].Stats.Mov = 5
		b := mustNew(t, s)
		events := mustEndPhase(t, b)
		var got []move
		for _, m := range moves(events) {
			if m.Unit == 1 {
				got = append(got, m)
			}
		}
		want := []move{{1, pos(4, 1), pos(2, 1)}, {1, pos(2, 1), pos(3, 1)}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("地形費用での移動 = %v, want %v", got, want)
		}
	})

	t.Run("TC-NFR-06-07 同じ入力と乱数で同じ再移動", func(t *testing.T) {
		a, c := mustNew(t, setup()), mustNew(t, setup())
		first, second := mustEndPhase(t, a), mustEndPhase(t, c)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(a.Units(), c.Units()) {
			t.Fatal("同じ入力と seed で結果が違う")
		}
	})
	t.Run("TC-FR-BTL-10-32 / TC-FR-BTL-10-33 / TC-NFR-06-08 敵の再移動を含む巻き戻し", func(t *testing.T) {
		b := mustNew(t, setup())
		first := mustEndPhase(t, b)
		after := b.Units()
		h := b.History()
		if len(h) == 0 {
			t.Fatal("履歴がない")
		}
		view, err := b.PreviewRewind(h[0].ID)
		if err != nil || view.Units()[1].Pos != pos(4, 1) {
			t.Fatalf("下見 = %v, %v", view.Units(), err)
		}
		after[1].Pos = pos(0, 0)
		if b.Units()[1].Pos != pos(4, 1) {
			t.Fatal("返却したユニットから本物が変わった")
		}
		if err := b.Rewind(h[0].ID); err != nil {
			t.Fatal(err)
		}
		second := mustEndPhase(t, b)
		if !reflect.DeepEqual(first, second) || b.Units()[1].Pos != pos(4, 1) {
			t.Fatal("敵の再移動を巻き戻して再実行すると結果が変わった")
		}
	})
	t.Run("TC-FR-BTL-07-22 再移動のあとで出来事を評価する", func(t *testing.T) {
		s := setup()
		s.Units[0].Pos = pos(2, 1)
		s.Units[1].Pos = pos(3, 1)
		s.Units[1].ID, s.Units[1].Group = "rider", "g"
		s.Units[2].Group = "g"
		s.Events = []tactics.MapEventSetup{{ID: "retreat", Trigger: tactics.TriggerEnemyMoved, Unit: "rider", Effect: tactics.EffectChargeGroup, Group: "g"}}
		b := mustNew(t, s)
		events := mustEndPhase(t, b)
		moveAt, triggerAt := -1, -1
		for i, e := range events {
			if e.Kind == tactics.EventMove && e.Unit == 1 {
				moveAt = i
			}
			if e.Kind == tactics.EventTrigger && e.MapEvent == "retreat" {
				triggerAt = i
			}
		}
		if moveAt < 0 || triggerAt <= moveAt {
			t.Fatalf("再移動後の出来事の順 = %v", kinds(events))
		}
	})
	t.Run("TC-FR-BTL-07-22 反撃で倒れた敵は下がらない", func(t *testing.T) {
		s := setup()
		s.Units[0].Stats.Str = 25
		b := mustNew(t, s)
		events := mustEndPhase(t, b)
		if b.Units()[1].Alive {
			t.Fatal("敵が反撃で倒れていない")
		}
		n := 0
		for _, m := range moves(events) {
			if m.Unit == 1 {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("戦死後にも移動した: %d 回", n)
		}
	})
}

func TestEnemyApproachIt06(t *testing.T) {
	t.Run("TC-FR-BTL-07-20 省略した方式はまっすぐ寄る", func(t *testing.T) {
		s := plain7(lord(1, 3, 0), enemy("敵", 4, 3, 1), boss(6, 0))
		b := mustNew(t, s)
		mustEndPhase(t, b)
		if got := b.Units()[1]; got.Approach != tactics.Direct || got.Pos != pos(3, 3) {
			t.Fatalf("敵 = %+v", got)
		}
	})
	t.Run("TC-FR-BTL-07-21 一団の名前が必要", func(t *testing.T) {
		s := plain7(lord(1, 3, 0), enemy("敵", 4, 3, 1), boss(6, 0))
		s.Units[1].Approach = tactics.GroupApproach
		_, _, err := tactics.New(s, 1)
		if err == nil || !strings.Contains(err.Error(), "一団") {
			t.Fatalf("err = %v", err)
		}
	})
}

// BenchmarkEnemyPhase46 は 32×32・自軍 7・敵 39 の敵軍思考の予備測定（TC-NFR-02-02）。
// 基準の Windows PC での受入測定は別途行う。
func BenchmarkEnemyPhase46(b *testing.B) {
	rows := make([]string, 32)
	for i := range rows {
		rows[i] = strings.Repeat(".", 32)
	}
	s := tactics.Setup{Terrain: grid(rows...)}
	for i := 0; i < 7; i++ {
		u := ally("自軍", 2, i+2, 3)
		if i == 0 {
			u.Lord = true
		}
		s.Units = append(s.Units, u)
	}
	for i := 0; i < 39; i++ {
		u := enemy("敵", 6+i%6, 2+i/6, 3)
		if i == 38 {
			u.Boss = true
		}
		if i < 20 {
			u.Group, u.Approach = "advance", tactics.GroupApproach
		}
		s.Units = append(s.Units, u)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		battle, _, err := tactics.New(s, 1)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := battle.EndPhase(); err != nil {
			b.Fatal(err)
		}
	}
}

package tactics

import (
	"errors"
	"reflect"
	"testing"
)

// reinforcementBattle は既知の平地と実 PCG を使う（DES-MAP-03、DES-RNF-01）。
func reinforcementBattle(t *testing.T) *Battle {
	t.Helper()
	b := snapshotBattle(t)
	b.terrain = make([][]Terrain, 5)
	for y := range b.terrain {
		b.terrain[y] = make([]Terrain, 5)
		for x := range b.terrain[y] {
			b.terrain[y][x].Cost = 1
		}
	}
	b.units[0].Pos = Pos{4, 4}
	b.units[1].Pos = Pos{0, 4}
	b.units[2].Pos = Pos{4, 0}
	b.reinforcements[0].Unit.Pos = Pos{2, 2}
	b.Turn = 2
	return b
}

func TestAlternativeReinforcements(t *testing.T) {
	t.Run("TC-FR-MAP-03-13 距離・Y・X・地形・占有", func(t *testing.T) {
		cases := []struct {
			name    string
			origin  Pos
			blocked []Pos
			walls   []Pos
			terrain *Terrain
			dead    []Pos
			want    Pos
			exists  bool
		}{
			{name: "元の空き", origin: Pos{2, 2}, want: Pos{2, 2}, exists: true},
			{name: "自軍占有", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, want: Pos{2, 1}, exists: true},
			{name: "森へ出現", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, terrain: &Terrain{Name: "森", Cost: 2}, want: Pos{2, 1}, exists: true},
			{name: "山へ出現", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, terrain: &Terrain{Name: "山", Cost: 3}, want: Pos{2, 1}, exists: true},
			{name: "距離2", origin: Pos{2, 2}, blocked: []Pos{{2, 2}, {2, 1}, {1, 2}, {3, 2}, {2, 3}}, want: Pos{2, 0}, exists: true},
			{name: "X順", origin: Pos{2, 2}, blocked: []Pos{{2, 2}, {2, 1}}, want: Pos{1, 2}, exists: true},
			{name: "端", origin: Pos{0, 0}, blocked: []Pos{{0, 0}}, want: Pos{1, 0}, exists: true},
			{name: "地形", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, walls: []Pos{{2, 1}, {1, 2}}, want: Pos{3, 2}, exists: true},
			{name: "倒れた敵", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, dead: []Pos{{2, 1}}, want: Pos{2, 1}, exists: true},
			{name: "壁越し", origin: Pos{2, 2}, blocked: []Pos{{2, 2}}, walls: []Pos{{2, 1}, {1, 2}, {3, 2}, {2, 3}}, want: Pos{2, 0}, exists: true},
			{name: "距離3は不可", origin: Pos{2, 2}, blocked: []Pos{{2, 2}, {2, 1}, {1, 2}, {3, 2}, {2, 3}, {2, 0}, {1, 1}, {3, 1}, {0, 2}, {4, 2}, {1, 3}, {3, 3}, {2, 4}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := reinforcementBattle(t)
				b.reinforcements[0].Unit.Pos = tc.origin
				for i, p := range tc.blocked {
					u := b.units[0].clone()
					u.Pos = p
					u.Side = Side(i % 2)
					b.units = append(b.units, u)
				}
				for _, p := range tc.dead {
					u := b.units[2].clone()
					u.Pos = p
					u.Alive = false
					b.units = append(b.units, u)
				}
				for _, p := range tc.walls {
					b.terrain[p.Y][p.X].Cost = 0
				}
				if tc.terrain != nil {
					b.terrain[1][2] = *tc.terrain
				}
				before := b.snapshot()
				n := len(b.units)
				p, ok := b.reinforcementPosition(tc.origin)
				if ok != tc.exists || ok && p != tc.want {
					t.Fatalf("選定 %v %v、期待 %v %v", p, ok, tc.want, tc.exists)
				}
				if !reflect.DeepEqual(before, b.snapshot()) {
					t.Fatal("探索が状態を変えた")
				}
				events, err := b.EndPhase()
				if err != nil {
					t.Fatal(err)
				}
				var found []Event
				for _, e := range events {
					if e.Kind == EventReinforce {
						found = append(found, e)
					}
				}
				if tc.exists {
					if len(found) != 1 || found[0].To != tc.want || found[0].Units[n].Pos != tc.want || b.units[n].Pos != tc.want {
						t.Fatalf("実位置: %v", found)
					}
				} else if len(found) != 0 || len(b.units) != n {
					t.Fatal("候補なしで出現した")
				}
			})
		}
	})
	t.Run("TC-FR-MAP-03-14 定義順・出来事・砦・出現番の行動", func(t *testing.T) {
		b := reinforcementBattle(t)
		b.units[0].Pos = Pos{2, 2}
		b.units[2].HP = 40
		b.terrain[0][4].HealPercent = 20
		b.reinforcements = append(b.reinforcements, b.reinforcements[0])
		defs := append([]ReinforcementSetup(nil), b.reinforcements...)
		events, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		if len(events) < 4 || events[0].Kind != EventPhaseStart || events[1].Kind != EventReinforce || events[2].Kind != EventReinforce || events[3].Kind != EventHeal {
			t.Fatal("番開始の順序が不正")
		}
		for i, p := range []Pos{{2, 1}, {1, 2}} {
			e := events[i+1]
			if e.Unit != 3+i || e.To != p || e.Units[3+i].Pos != p || b.units[3+i].Pos != p {
				t.Fatal("定義順または実位置が不正")
			}
		}
		for _, e := range events {
			if (e.Kind == EventMove || e.Kind == EventWait || e.Kind == EventStrike) && e.Unit >= 3 {
				t.Fatal("出現した番に行動した")
			}
		}
		if !reflect.DeepEqual(defs, b.reinforcements) {
			t.Fatal("定義を書き換えた")
		}
	})
	t.Run("TC-FR-MAP-03-15 候補なし・延期なし・再開", func(t *testing.T) {
		for _, occupied := range []bool{false, true} {
			b := reinforcementBattle(t)
			for y := 0; y < 5; y++ {
				for x := 0; x < 5; x++ {
					p := Pos{x, y}
					if abs(x-2)+abs(y-2) > 2 {
						continue
					}
					// 元の平地は生存ユニットでふさぎ、距離 1・2 だけ地形を変える。
					if occupied || p == (Pos{2, 2}) {
						u := b.units[2].clone()
						u.Pos = p
						u.AI = Hold
						u.Weapon.MinRange = 10
						u.Weapon.MaxRange = 10
						b.units = append(b.units, u)
					} else {
						b.terrain[y][x].Cost = 0
					}
				}
			}
			b.units[2].HP = 40
			b.terrain[0][4].HealPercent = 20
			before := b.snapshot()
			n := len(b.units)
			events, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			after := b.snapshot()
			if len(b.units) != n || b.Turn != 3 || b.units[2].HP != 60 || !reflect.DeepEqual(before.rngState, after.rngState) {
				t.Fatal("不出現・他の番処理・乱数が不正")
			}
			for _, e := range events {
				if e.Kind == EventReinforce {
					t.Fatal("空きなしで出現")
				}
			}
			id := b.History()[0].ID
			rewindTo(t, b, id)
			resumed, err := b.EndPhase()
			if err != nil || !reflect.DeepEqual(events, resumed) || !reflect.DeepEqual(after, b.snapshot()) {
				t.Fatal("再開で結果が変化")
			}
			for y := range b.terrain {
				for x := range b.terrain[y] {
					b.terrain[y][x].Cost = 1
				}
			}
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
			if len(b.units) != n {
				t.Fatal("延期した")
			}
		}
	})
}

func TestReinforcementHints(t *testing.T) {
	t.Run("TC-FR-MAP-03-08 時機と抑制なし", func(t *testing.T) {
		for _, tc := range []struct {
			turn   int
			phase  Side
			stage  phaseStage
			result Result
			want   bool
		}{
			{1, Player, playerInput, Ongoing, false}, {1, Enemy, playerStart, Ongoing, false},
			{2, Player, playerInput, Ongoing, true}, {2, Player, enemyStart, Ongoing, true},
			{2, Enemy, enemyAction, Ongoing, false}, {2, Enemy, playerStart, Ongoing, false},
			{3, Player, playerInput, Ongoing, false}, {2, Player, playerInput, Won, false}, {2, Player, playerInput, Lost, false},
		} {
			b := reinforcementBattle(t)
			b.Turn = tc.turn
			b.Phase = tc.phase
			b.progress.stage = tc.stage
			b.Result = tc.result
			for y := range b.terrain {
				for x := range b.terrain[y] {
					b.terrain[y][x].Cost = 0
				}
			}
			if got := len(b.BoardView().ReinforcementHints()) > 0; got != tc.want {
				t.Fatalf("%+v: %v", tc, got)
			}
		}
		// 最早の 2T 定義でも、1T と自軍開始前の before には予兆を出さない。
		live := snapshotBattle(t)
		if len(live.BoardView().ReinforcementHints()) != 0 {
			t.Fatal("1T に予兆")
		}
		if _, err := live.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if len(live.BoardView().ReinforcementHints()) != 1 {
			t.Fatal("2T 自軍に予兆なし")
		}
		for _, h := range live.History() {
			v, err := live.PreviewRewind(h.ID)
			if err != nil || len(v.ReinforcementHints()) != 0 {
				t.Fatal("2T 自軍開始前の予兆")
			}
		}
		b := snapshotBattle(t)
		s := Setup{Terrain: b.terrain, Reinforcements: b.reinforcements}
		for _, u := range b.units {
			s.Units = append(s.Units, UnitSetup{Name: u.Name, Side: u.Side, Level: u.Level, Lord: u.Lord, Boss: u.Boss, Stats: u.Stats, Weapon: u.Weapon, Pos: u.Pos})
		}
		v, err := NewBoardView(s)
		if err != nil || len(v.ReinforcementHints()) != 0 {
			t.Fatal("配置に予兆")
		}
		s.Reinforcements[0].Turn = 1
		if _, _, err := New(s, 42); !errors.Is(err, ErrInvalidSetup) {
			t.Fatal("T=1 を受理")
		}
	})
	t.Run("TC-FR-MAP-03-09 候補全域・丸め・重複・独立", func(t *testing.T) {
		cases := []struct {
			origin Pos
			want   ReinforcementHint
		}{
			{Pos{3, 3}, ReinforcementHint{Min: Pos{0, 0}, Max: Pos{7, 7}}},
			{Pos{0, 3}, ReinforcementHint{Min: Pos{0, 0}, Max: Pos{3, 7}}}, {Pos{0, 4}, ReinforcementHint{Min: Pos{0, 0}, Max: Pos{3, 7}}},
			{Pos{15, 15}, ReinforcementHint{Min: Pos{12, 12}, Max: Pos{15, 15}}}, {Pos{0, 4}, ReinforcementHint{Min: Pos{0, 0}, Max: Pos{3, 7}}},
			{Pos{0, 10}, ReinforcementHint{Min: Pos{0, 8}, Max: Pos{3, 15}}}, {Pos{0, 7}, ReinforcementHint{Min: Pos{0, 4}, Max: Pos{3, 11}}},
			{Pos{0, 14}, ReinforcementHint{Min: Pos{0, 12}, Max: Pos{3, 15}}}, {Pos{15, 14}, ReinforcementHint{Min: Pos{12, 12}, Max: Pos{15, 15}}},
			{Pos{4, 2}, ReinforcementHint{Min: Pos{0, 0}, Max: Pos{7, 7}}},
		}
		for _, tc := range cases {
			b := reinforcementBattle(t)
			b.terrain = make([][]Terrain, 16)
			for y := range b.terrain {
				b.terrain[y] = make([]Terrain, 16)
			}
			b.reinforcements[0].Unit.Pos = tc.origin
			got := b.BoardView().ReinforcementHints()
			for y := max(0, tc.origin.Y-2); y <= min(15, tc.origin.Y+2); y++ {
				for x := max(0, tc.origin.X-2); x <= min(15, tc.origin.X+2); x++ {
					if abs(x-tc.origin.X)+abs(y-tc.origin.Y) <= 2 && (len(got) != 1 || x < got[0].Min.X || x > got[0].Max.X || y < got[0].Min.Y || y > got[0].Max.Y) {
						t.Fatal("代替候補が予兆の外")
					}
				}
			}
			if !reflect.DeepEqual(got, []ReinforcementHint{tc.want}) {
				t.Fatalf("%v: %v", tc.origin, got)
			}
		}
		multi := reinforcementBattle(t)
		multi.terrain = make([][]Terrain, 16)
		for y := range multi.terrain {
			multi.terrain[y] = make([]Terrain, 16)
		}
		r0 := multi.reinforcements[0]
		multi.reinforcements = nil
		for _, p := range []Pos{{15, 14}, {0, 14}, {4, 2}, {0, 3}, {0, 4}} {
			r := r0
			r.Unit.Pos = p
			multi.reinforcements = append(multi.reinforcements, r)
		}
		future := r0
		future.Turn = 3
		future.Unit.Pos = Pos{8, 8}
		multi.reinforcements = append(multi.reinforcements, future)
		ordered := []ReinforcementHint{{Min: Pos{0, 0}, Max: Pos{3, 7}}, {Min: Pos{0, 0}, Max: Pos{7, 7}}, {Min: Pos{0, 12}, Max: Pos{3, 15}}, {Min: Pos{12, 12}, Max: Pos{15, 15}}}
		if !reflect.DeepEqual(multi.BoardView().ReinforcementHints(), ordered) {
			t.Fatal("複数方面・異なるターン・矩形の順序が不正")
		}
		b := reinforcementBattle(t)
		b.reinforcements[0].Unit.Pos = Pos{0, 0}
		r := b.reinforcements[0]
		r.Unit.Pos = Pos{4, 4}
		b.reinforcements = append([]ReinforcementSetup{r}, b.reinforcements...)
		b.reinforcements = append(b.reinforcements, r)
		r.Turn = 3
		b.reinforcements = append(b.reinforcements, r)
		v := b.BoardView()
		want := []ReinforcementHint{{Min: Pos{0, 0}, Max: Pos{3, 3}}, {Min: Pos{0, 0}, Max: Pos{4, 4}}}
		got := v.ReinforcementHints()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("重複・順序・端: %v", got)
		}
		got[0].Min.X = 99
		if !reflect.DeepEqual(v.ReinforcementHints(), want) || !reflect.DeepEqual(b.BoardView().ReinforcementHints(), want) {
			t.Fatal("返却値が内部を変更")
		}
		b.reinforcements[0].Unit.Pos = Pos{0, 0}
		if !reflect.DeepEqual(v.ReinforcementHints(), want) {
			t.Fatal("参照値が定義と共有")
		}
		if len((BoardView{}).ReinforcementHints()) != 0 {
			t.Fatal("ゼロ値の予兆")
		}
	})
	t.Run("TC-FR-MAP-03-10 履歴・巻き戻し・再開・失敗時不変", func(t *testing.T) {
		b := reinforcementBattle(t)
		b.units[0].Pos = Pos{2, 2}
		control := reinforcementBattle(t)
		control.units[0].Pos = Pos{2, 2}
		// 実際に乱数を使う攻撃・大盾・成長も、予兆参照後に対照戦闘と一致する。
		combat, combatControl := snapshotBattle(t), snapshotBattle(t)
		for i := 0; i < 3; i++ {
			_ = combat.BoardView().ReinforcementHints()
		}
		cmd := Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2}
		a, ae := combat.Act(cmd)
		c, ce := combatControl.Act(cmd)
		if ae != nil || ce != nil || !reflect.DeepEqual(a, c) || !reflect.DeepEqual(combat.snapshot(), combatControl.snapshot()) {
			t.Fatal("予兆参照後の戦闘・成長・PCG が不一致")
		}
		before := b.snapshot()
		for i := 0; i < 3; i++ {
			_ = b.BoardView().ReinforcementHints()
		}
		assertReinforcementFailures(t, b)
		events, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		other, err := control.EndPhase()
		if err != nil || !reflect.DeepEqual(events, other) || !reflect.DeepEqual(b.snapshot(), control.snapshot()) {
			t.Fatal("閲覧が進行を変更")
		}
		after := b.snapshot()
		entries := b.History()
		if len(entries) == 0 || entries[0].Kind != HistoryEndPhase {
			t.Fatal("敵軍開始の履歴なし")
		}
		for _, h := range entries[:1] {
			v, err := b.PreviewRewind(h.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := h.Kind == HistoryEndPhase
			if (len(v.ReinforcementHints()) > 0) != want {
				t.Fatalf("before の予兆: %v", h)
			}
			rewindTo(t, b, h.ID)
			if !reflect.DeepEqual(v.ReinforcementHints(), b.BoardView().ReinforcementHints()) {
				t.Fatal("復元と予兆が不一致")
			}
			assertReinforcementFailures(t, b)
			resumed, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			offset := h.EventStart - before.eventCount
			if !reflect.DeepEqual(resumed, events[offset:]) || !reflect.DeepEqual(after, b.snapshot()) {
				t.Fatal("代替位置・再開・二重出現が不正")
			}
			break
		}
		// 再生成された各境界も、before からの再開で二重出現しない。
		for _, kind := range []HistoryKind{HistoryAction, HistoryPlayerStart} {
			for _, h := range b.History() {
				if h.Kind != kind {
					continue
				}
				v, err := b.PreviewRewind(h.ID)
				if err != nil || len(v.ReinforcementHints()) != 0 {
					t.Fatal("処理後に予兆")
				}
				restoreHistoryForResume(t, b, h.ID)
				if _, err := b.Resume(); err != nil || !reflect.DeepEqual(after, b.snapshot()) {
					t.Fatal("処理後の再開が不正")
				}
				break
			}
		}
	})
}

func assertReinforcementFailures(t *testing.T, b *Battle) {
	t.Helper()
	before := b.snapshot()
	var history []historyRecord
	if b.history != nil {
		history = make([]historyRecord, len(b.history))
		copy(history, b.history)
	}
	id, left := b.nextHistoryID, b.RewindsLeft()
	if _, err := b.Act(Command{Action: Action(99)}); err == nil {
		t.Fatal("無効操作成功")
	}
	if _, err := b.PreviewRewind(0); err == nil {
		t.Fatal("不正閲覧成功")
	}
	if err := b.Rewind(0); err == nil {
		t.Fatal("不正巻き戻し成功")
	}
	if !b.NeedsResume() {
		if _, err := b.Resume(); err == nil {
			t.Fatal("不正再開成功")
		}
	} else {
		if _, err := b.EndPhase(); err == nil {
			t.Fatal("再開待ちで番終了")
		}
	}
	if !reflect.DeepEqual(before, b.snapshot()) || !reflect.DeepEqual(history, b.history) || id != b.nextHistoryID || left != b.RewindsLeft() {
		t.Fatal("失敗が状態を変更")
	}
}

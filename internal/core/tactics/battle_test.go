package tactics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestNew は戦闘の作成を確かめる（DES-BTL-01）。
func TestNew(t *testing.T) {
	t.Run("TC-FR-BTL-01-09 正しい Setup から戦闘を作る", func(t *testing.T) {
		s := baseSetup()
		b, _, err := tactics.New(s, 1)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		assertEqual(t, "Width", b.Width(), 5)
		assertEqual(t, "Height", b.Height(), 5)
		units := b.Units()
		if len(units) != len(s.Units) {
			t.Fatalf("len(Units) = %d, want %d", len(units), len(s.Units))
		}
		for i, u := range units {
			us := s.Units[i]
			want := tactics.Unit{
				Name: us.Name, ClassName: us.ClassName, Side: us.Side, Stats: us.Stats,
				HP: us.Stats.HP, Pos: us.Pos, Weapon: us.Weapon, Items: us.Items, Skill: us.Skill,
				Lord: us.Lord, Boss: us.Boss, AI: us.AI, Acted: false, Alive: true,
			}
			assertEqual(t, "Units", u, want)
		}
	})

	t.Run("TC-FR-BTL-01-11 境界の値の Setup は正しい", func(t *testing.T) {
		s := baseSetup()
		s.Terrain = grid(".....", ".....", ".....", ".....", ".....")
		// 最大 HP 1、ほかの能力 0。
		s.Units[1].Stats = tactics.Stats{HP: 1}
		// 射程 1〜1 と 2〜3。
		s.Units[1].Weapon.MinRange, s.Units[1].Weapon.MaxRange = 1, 1
		s.Units[3].Weapon = tactics.Weapon{Type: tactics.Bow, MinRange: 2, MaxRange: 3}
		// アイテムの数と回復量 1。
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "薬", Heal: 1}, Count: 1}}
		// スキルの数値の境界。
		s.Units[0].Skill = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 100}
		s.Units[1].Skill = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 0}
		s.Units[2].Skill = tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Multiplier: 1}
		s.Units[3].Skill = tactics.Skill{Name: "狙撃", Effect: tactics.Snipe, Hit: 0, Multiplier: 0}
		// 増援のターン 2。初めのユニットと同じマスでもよい。
		r := foe("増援", 4, 0)
		s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: r}}
		// 1 × 1 のマップは作れない（主人公とボスが要る）ので、1 行のマップも確かめる。
		if _, _, err := tactics.New(s, 1); err != nil {
			t.Errorf("New: %v", err)
		}
		one := tactics.Setup{Terrain: grid(".."), Units: []tactics.UnitSetup{lord(0, 0, 1), boss(1, 0)}}
		if _, _, err := tactics.New(one, 1); err != nil {
			t.Errorf("New(1 行): %v", err)
		}
	})

	t.Run("TC-FR-BTL-01-12 Setup と Units() の写しを変えても戦闘は変わらない", func(t *testing.T) {
		s := baseSetup()
		b := mustNew(t, s)
		before := snapshot(b)
		s.Terrain[0][0] = wall
		s.Units[0].Items[0].Count = 99
		u := b.Units()
		u[0].Items[0].Count = 50
		u[0].HP = 1
		assertEqual(t, "状態", snapshot(b), before)
		assertEqual(t, "地形", b.TerrainAt(pos(0, 0)), plain)
	})
}

// TestNewInvalidSetup は Setup の規則 1〜6 を確かめる（DES-BTL-01）。
func TestNewInvalidSetup(t *testing.T) {
	tests := []struct {
		name   string
		modify func(s *tactics.Setup)
		reason string // 誤りの文に含まれるべき語
	}{
		{"規則 1 地形の行がない", func(s *tactics.Setup) { s.Terrain = nil }, "地形の行がありません"},
		{"規則 1 行の長さが 0", func(s *tactics.Setup) { s.Terrain = [][]tactics.Terrain{{}} }, "空です"},
		{"規則 1 行の長さがそろっていない", func(s *tactics.Setup) { s.Terrain[2] = s.Terrain[2][:4] }, "地形の行 2"},
		{"規則 2 ユニットがマップの外（x が負）", func(s *tactics.Setup) { s.Units[1].Pos = pos(-1, 0) }, "units[1]"},
		{"規則 2 ユニットがマップの外（y が高さ）", func(s *tactics.Setup) { s.Units[1].Pos = pos(0, 5) }, "units[1]"},
		{"規則 2 ユニットが通れない地形", func(s *tactics.Setup) { s.Terrain[2][2] = wall; s.Units[1].Pos = pos(2, 2) }, "通れない地形"},
		{"規則 2 増援がマップの外", func(s *tactics.Setup) {
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 5, 0)}}
		}, "reinforcements[0]"},
		{"規則 2 増援が通れない地形", func(s *tactics.Setup) {
			s.Terrain[1][1] = wall
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 1, 1)}}
		}, "reinforcements[0]"},
		{"規則 3 初めのユニットが重なる", func(s *tactics.Setup) { s.Units[3].Pos = pos(0, 4) }, "units[1] がいます"},
		{"規則 4 最大 HP が 0", func(s *tactics.Setup) { s.Units[1].Stats.HP = 0 }, "最大 HP"},
		{"規則 4 力が負", func(s *tactics.Setup) { s.Units[1].Stats.Str = -1 }, "負の値"},
		{"規則 4 移動が負", func(s *tactics.Setup) { s.Units[1].Stats.Mov = -1 }, "負の値"},
		{"規則 4 最小射程が 0", func(s *tactics.Setup) { s.Units[1].Weapon.MinRange = 0 }, "射程"},
		{"規則 4 最小射程が最大射程より大きい", func(s *tactics.Setup) {
			s.Units[1].Weapon.MinRange, s.Units[1].Weapon.MaxRange = 2, 1
		}, "射程"},
		{"規則 4 アイテムの数が 0", func(s *tactics.Setup) { s.Units[1].Items[0].Count = 0 }, "数 0"},
		{"規則 4 アイテムの回復量が 0", func(s *tactics.Setup) { s.Units[1].Items[0].Item.Heal = 0 }, "回復量 0"},
		{"規則 4 増援の能力", func(s *tactics.Setup) {
			r := foe("増援", 4, 0)
			r.Stats.HP = 0
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: r}}
		}, "reinforcements[0]: 最大 HP"},
		{"規則 5 主人公がいない", func(s *tactics.Setup) { s.Units[0].Lord = false }, "主人公が 0 体"},
		{"規則 5 主人公が 2 体", func(s *tactics.Setup) { s.Units[1].Lord = true }, "主人公が 2 体"},
		{"規則 5 主人公が敵軍", func(s *tactics.Setup) { s.Units[3].Lord = true }, "主人公が自軍ではありません"},
		{"規則 5 ボスがいない", func(s *tactics.Setup) { s.Units[2].Boss = false }, "ボスが 0 体"},
		{"規則 5 ボスが 2 体", func(s *tactics.Setup) { s.Units[3].Boss = true }, "ボスが 2 体"},
		{"規則 5 ボスが自軍", func(s *tactics.Setup) { s.Units[1].Boss = true }, "ボスが敵軍ではありません"},
		{"規則 5 増援が主人公", func(s *tactics.Setup) {
			r := foe("増援", 4, 0)
			r.Lord = true
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: r}}
		}, "主人公とボスになれません"},
		{"規則 5 増援がボス", func(s *tactics.Setup) {
			r := foe("増援", 4, 0)
			r.Boss = true
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: r}}
		}, "主人公とボスになれません"},
		{"規則 5 増援が自軍", func(s *tactics.Setup) {
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: ally("増援", 4, 0, 1)}}
		}, "敵軍ではありません"},
		{"規則 6 増援のターンが 1", func(s *tactics.Setup) {
			s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 1, Unit: foe("増援", 4, 0)}}
		}, "ターン 1"},
		{"規則 6 武器の種類が範囲外", func(s *tactics.Setup) { s.Units[1].Weapon.Type = tactics.Bow + 1 }, "武器の種類"},
		{"規則 6 武器の種類が負", func(s *tactics.Setup) { s.Units[1].Weapon.Type = -1 }, "武器の種類"},
		{"規則 6 スキルの効果が範囲外", func(s *tactics.Setup) { s.Units[1].Skill.Effect = tactics.Snipe + 1 }, "スキルの効果"},
		{"規則 6 動き方が範囲外", func(s *tactics.Setup) { s.Units[3].AI = tactics.Hold + 1 }, "動き方"},
		{"規則 6 スキルの範囲が負", func(s *tactics.Setup) { s.Units[1].Skill.Range = -1 }, "スキルの数値"},
		{"規則 6 スキルの命中が負", func(s *tactics.Setup) { s.Units[1].Skill.Hit = -1 }, "スキルの数値"},
		{"規則 6 スキルの回避が負", func(s *tactics.Setup) { s.Units[1].Skill.Avoid = -1 }, "スキルの数値"},
		{"規則 6 スキルの確率が 101", func(s *tactics.Setup) { s.Units[1].Skill.Chance = 101 }, "確率 101"},
		{"規則 6 スキルの確率が負", func(s *tactics.Setup) { s.Units[1].Skill.Chance = -1 }, "確率 -1"},
		{"規則 6 スキルの倍率が負", func(s *tactics.Setup) { s.Units[1].Skill.Multiplier = -1 }, "倍率 -1"},
		{"規則 6 怒りの倍率が 0", func(s *tactics.Setup) {
			s.Units[1].Skill = tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Multiplier: 0}
		}, "倍率 0"},
	}
	for _, tt := range tests {
		t.Run("TC-FR-BTL-01-10 "+tt.name, func(t *testing.T) {
			s := baseSetup()
			tt.modify(&s)
			b, events, err := tactics.New(s, 1)
			if !errors.Is(err, tactics.ErrInvalidSetup) {
				t.Fatalf("err = %v, want ErrInvalidSetup", err)
			}
			if !strings.Contains(err.Error(), tt.reason) {
				t.Errorf("err = %q, 理由 %q を含まない", err, tt.reason)
			}
			if b != nil || events != nil {
				t.Errorf("誤りのときは nil を返す: b = %v, events = %v", b, events)
			}
		})
	}
}

// TestPhase は番の進み方を確かめる（DES-BTL-02）。
func TestPhase(t *testing.T) {
	t.Run("TC-FR-BTL-02-01 作った直後は 1 ターン目の自軍の番", func(t *testing.T) {
		b, events, err := tactics.New(baseSetup(), 1)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "Turn", b.Turn, 1)
		assertEqual(t, "Phase", b.Phase, tactics.Player)
		assertEqual(t, "Result", b.Result, tactics.Ongoing)
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventPhaseStart})
		assertEqual(t, "Side", events[0].Side, tactics.Player)
		assertEqual(t, "Turn", events[0].Turn, 1)
	})

	t.Run("TC-FR-BTL-02-02 行動していない自軍が残っていれば自軍の番のまま（好きな順に動かせる）", func(t *testing.T) {
		b := mustNew(t, baseSetup())
		// 添字の大きいほうから動かす。
		events, err := b.Act(tactics.Command{Unit: 1, To: pos(1, 4), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventMove, tactics.EventWait})
		assertEqual(t, "Phase", b.Phase, tactics.Player)
		assertEqual(t, "Turn", b.Turn, 1)
		u := b.Units()
		assertEqual(t, "Acted[0]", u[0].Acted, false)
		assertEqual(t, "Acted[1]", u[1].Acted, true)
	})

	t.Run("TC-FR-BTL-02-03 全員が行動すると敵軍の番を経て次のターンの自軍の番になる", func(t *testing.T) {
		b := mustNew(t, quietSetup())
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 0), Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		events, err := b.Act(tactics.Command{Unit: 1, To: pos(0, 4), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventWait, tactics.EventPhaseStart, tactics.EventPhaseStart,
		})
		assertEqual(t, "敵軍の番", [2]any{events[1].Side, events[1].Turn}, [2]any{tactics.Enemy, 1})
		assertEqual(t, "自軍の番", [2]any{events[2].Side, events[2].Turn}, [2]any{tactics.Player, 2})
		assertEqual(t, "Turn", b.Turn, 2)
		assertEqual(t, "Phase", b.Phase, tactics.Player)
		for i, u := range b.Units() {
			if u.Side == tactics.Player && u.Acted {
				t.Errorf("units[%d] の Acted が戻っていない", i)
			}
		}
		// 次のターンも同じユニットを動かせる。
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(2, 0), Action: tactics.Wait}); err != nil {
			t.Errorf("2 ターン目の Act: %v", err)
		}
	})

	t.Run("TC-FR-BTL-02-04 EndPhase で途中でも番を終えられる", func(t *testing.T) {
		b := mustNew(t, quietSetup())
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 0), Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		events, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventPhaseStart, tactics.EventPhaseStart})
		assertEqual(t, "敵軍の番", [2]any{events[0].Side, events[0].Turn}, [2]any{tactics.Enemy, 1})
		assertEqual(t, "自軍の番", [2]any{events[1].Side, events[1].Turn}, [2]any{tactics.Player, 2})
		assertEqual(t, "Turn", b.Turn, 2)
		assertEqual(t, "Phase", b.Phase, tactics.Player)
		assertEqual(t, "Acted[0]", b.Units()[0].Acted, false)
		// 1 体も動かさずに終えてもよい。
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "Turn", b.Turn, 3)
	})

	t.Run("TC-FR-BTL-02-05 倒れた自軍は数えず、生きている全員が行動したら進む", func(t *testing.T) {
		b := mustNew(t, quietSetup())
		b.KillUnit(1)
		events, err := b.Act(tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventWait, tactics.EventPhaseStart, tactics.EventPhaseStart,
		})
		assertEqual(t, "Turn", b.Turn, 2)
	})

	t.Run("TC-FR-BTL-02-06 敵軍の番で動かない敵は出来事を出さない", func(t *testing.T) {
		b := mustNew(t, quietSetup())
		before := b.Units()
		for turn := 1; turn <= 3; turn++ {
			events, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventPhaseStart, tactics.EventPhaseStart})
		}
		assertEqual(t, "Turn", b.Turn, 4)
		assertEqual(t, "ユニット", b.Units(), before)
	})

	t.Run("TC-FR-BTL-02-07 戦闘が終わっているか敵軍の番なら EndPhase は誤り", func(t *testing.T) {
		tests := []struct {
			result tactics.Result
			phase  tactics.Side
			want   error
		}{
			{tactics.Won, tactics.Player, tactics.ErrBattleOver},
			{tactics.Lost, tactics.Player, tactics.ErrBattleOver},
			{tactics.Won, tactics.Enemy, tactics.ErrBattleOver},
			{tactics.Ongoing, tactics.Enemy, tactics.ErrNotPlayerPhase},
		}
		for _, tt := range tests {
			b := mustNew(t, baseSetup())
			b.Result, b.Phase = tt.result, tt.phase
			before := snapshot(b)
			events, err := b.EndPhase()
			if !errors.Is(err, tt.want) {
				t.Errorf("Result %v Phase %v: err = %v, want %v", tt.result, tt.phase, err, tt.want)
			}
			if events != nil {
				t.Errorf("出来事 = %v, want nil", events)
			}
			assertEqual(t, "状態", snapshot(b), before)
		}
	})
}

// TestWait は待機と出来事を確かめる（DES-BTL-04、DES-EVT-01）。
func TestWait(t *testing.T) {
	t.Run("TC-FR-BTL-01-02 / TC-FR-BTL-03-01 移動して待機すると、Move と Wait を出す", func(t *testing.T) {
		b := mustNew(t, baseSetup())
		events, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventMove, tactics.EventWait})
		mv := events[0]
		assertEqual(t, "Move", [3]any{mv.Unit, mv.From, mv.To}, [3]any{0, pos(0, 0), pos(1, 1)})
		assertEqual(t, "Wait.Unit", events[1].Unit, 0)
		u := b.Units()[0]
		assertEqual(t, "Pos", u.Pos, pos(1, 1))
		assertEqual(t, "Acted", u.Acted, true)
	})

	t.Run("TC-FR-BTL-03-02 移動しないで待機すると、Move を出さない", func(t *testing.T) {
		b := mustNew(t, baseSetup())
		events, err := b.Act(tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventWait})
		u := b.Units()[0]
		assertEqual(t, "Pos", u.Pos, pos(0, 0))
		assertEqual(t, "Acted", u.Acted, true)
	})

	t.Run("TC-FR-BTL-03-03 出来事は、その時点のユニットの写しを持つ", func(t *testing.T) {
		b, start, err := tactics.New(baseSetup(), 1)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "New の写し", start[0].Units, b.Units())

		first, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 0), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		// Move の写しは移動したあとで、まだ行動済みでない。Wait の写しは行動済み。
		assertEqual(t, "Move の写しの位置", first[0].Units[0].Pos, pos(1, 0))
		assertEqual(t, "Move の写しの Acted", first[0].Units[0].Acted, false)
		assertEqual(t, "Wait の写しの Acted", first[1].Units[0].Acted, true)

		second, err := b.Act(tactics.Command{Unit: 1, To: pos(1, 4), Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		// 後の行動で、前の出来事の写しは変わらない。
		assertEqual(t, "前の写し", first[1].Units[1].Pos, pos(0, 4))
		// 次の自軍の番の始めの写しでは、Acted が戻っている。
		last := second[len(second)-1]
		assertEqual(t, "番の始めの写し", last.Units[0].Acted, false)
		assertEqual(t, "番の始めの写しの位置", last.Units[1].Pos, pos(1, 4))

		// 写しを変えても戦闘は変わらない。
		before := snapshot(b)
		last.Units[0].Items[0].Count = 99
		last.Units[0].HP = 1
		assertEqual(t, "状態", snapshot(b), before)
	})
}

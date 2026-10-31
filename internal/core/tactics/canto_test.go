package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 騎兵の再移動と、巻き戻しの 1 手の区切りの試験（DES-BTL-10、DES-IT06 8.1・8.2 節、#124）。

var cantoSkill = tactics.Skill{Name: "再移動", Effect: tactics.Canto}

// cantoSetup は、上の段を通路、下の段を主人公とボスの置き場にした盤面（真ん中の段は壁）。
//
//	y0: . F . . . F . .   騎兵 (0, 0) 移動 7・再移動、味方 (1, 0)（森）、敵 (3, 0)（待ち構える、移動 0）
//	y1: # # # # # # # #
//	y2: . . . . . . . .   主人公 (0, 2) 移動 2、ボス (7, 2)
//
// 騎兵が (2, 0) まで動く最小の費用は 森 2 ＋ 平地 1 = 3（味方のいる森は通り抜けられる）。歩数は 2。
//
//	0: 主人公 1: 騎兵 2: 味方 3: 敵 4: ボス
func cantoSetup() tactics.Setup {
	cav := ally("騎兵", 0, 0, 7)
	cav.ClassName = "騎兵"
	cav.ClassSkills = []tactics.Skill{cantoSkill}
	f := foe("敵", 3, 0)
	f.AI = tactics.Hold
	f.Stats.Mov = 0
	return tactics.Setup{
		Terrain: grid(".F...F..", "########", "........"),
		Units:   []tactics.UnitSetup{lord(0, 2, 2), cav, ally("味方", 1, 0, 2), f, boss(7, 2)},
	}
}

// attackFrom2 は、騎兵が (2, 0) から敵を攻撃するコマンド。
var attackFrom2 = tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.Attack, Target: 3}

// cantoBattle は cantoSetup に edit をかけて戦闘を作り、乱数を roll に固定する（99 は外れ、0 は当たり）。
func cantoBattle(t *testing.T, edit func(s *tactics.Setup), roll int) *tactics.Battle {
	t.Helper()
	s := cantoSetup()
	if edit != nil {
		edit(&s)
	}
	b := mustNew(t, s)
	b.SetRoll(func() int { return roll })
	return b
}

// cantoState は、失敗で変わらないことを比べるための状態（待っている再移動と履歴の長さを含む）。
type cantoState struct {
	State   state
	Pending tactics.PendingCanto
	Waiting bool
	History int
	Rewinds int
}

func cantoSnapshot(b *tactics.Battle) cantoState {
	pc, ok := b.PendingCanto()
	return cantoState{State: snapshot(b), Pending: pc, Waiting: ok, History: len(b.History()), Rewinds: b.RewindsLeft()}
}

func mustCanto(t *testing.T, b *tactics.Battle, cmd tactics.Command) []tactics.Event {
	t.Helper()
	events, err := b.Act(cmd)
	if err != nil {
		t.Fatalf("Act(%+v): %v", cmd, err)
	}
	return events
}

func TestCanto(t *testing.T) {
	t.Run("TC-FR-BTL-03-20 残りは移動力から最小の費用を引いた値で、行動のあとの盤面で行けるマスを求める", func(t *testing.T) {
		cases := []struct {
			name   string
			prep   func(b *tactics.Battle)
			roll   int
			reach  map[tactics.Pos]int
			result tactics.Result
		}{
			// 7 − 3 = 4。歩数（2）で引けば 5 になり、(0, 0) より先にも行けてしまう。
			{name: "外れて敵が残る: 味方のいる森は止まれず、敵のマスにも入れない", roll: 99,
				reach: map[tactics.Pos]int{pos(2, 0): 0, pos(0, 0): 3}},
			// 敵が倒れて空いたマスを通れる。(5, 0) は 1 ＋ 1 ＋ 2 = 4、(6, 0) は 5 で行けない。
			{name: "敵を倒すと空いたマスを使える。費用 5 のマスには行けない", roll: 0, prep: func(b *tactics.Battle) { b.SetHP(3, 1) },
				reach: map[tactics.Pos]int{pos(2, 0): 0, pos(0, 0): 3, pos(3, 0): 1, pos(4, 0): 2, pos(5, 0): 4}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := cantoBattle(t, nil, tc.roll)
				if tc.prep != nil {
					tc.prep(b)
				}
				events := mustCanto(t, b, attackFrom2)
				if indexOf(events, tactics.EventPhaseStart) >= 0 {
					t.Fatalf("再移動を待たずに番が進んだ: %v", kinds(events))
				}
				h := b.History()
				pc, ok := b.PendingCanto()
				assertEqual(t, "待っている再移動", pc, tactics.PendingCanto{Unit: 1, Remaining: 4, HistoryID: h[len(h)-1].ID})
				if !ok {
					t.Fatal("再移動を待っていない")
				}
				assertEqual(t, "行けるマス", b.CantoReachable(), tc.reach)
			})
		}
	})

	t.Run("TC-FR-BTL-03-20 攻撃・回復・道具のあとは待ち、待機・スキルなし・残り 0・戦死・勝敗確定では待たない", func(t *testing.T) {
		staff := tactics.Weapon{Name: "杖", Type: tactics.Staff, Heal: 10, MaxUses: 3, MinRange: 1, MaxRange: 1}
		cases := []struct {
			name    string
			edit    func(s *tactics.Setup)
			prep    func(t *testing.T, b *tactics.Battle)
			cmd     tactics.Command
			roll    int
			waiting bool
			result  tactics.Result
		}{
			{name: "攻撃", cmd: attackFrom2, roll: 99, waiting: true},
			{name: "回復", edit: func(s *tactics.Setup) { s.Units[1].Weapon = staff },
				prep: func(t *testing.T, b *tactics.Battle) { b.SetHP(2, 5) },
				cmd:  tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.Heal, Target: 2}, roll: 99, waiting: true},
			{name: "道具", prep: func(t *testing.T, b *tactics.Battle) { b.SetHP(1, 5) },
				cmd: tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.UseItem, Item: 0}, roll: 99, waiting: true},
			{name: "待機", cmd: tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.Wait}, roll: 99},
			{name: "再移動のスキルがない", edit: func(s *tactics.Setup) { s.Units[1].ClassSkills = nil }, cmd: attackFrom2, roll: 99},
			{name: "残り 0（移動 3 で費用 3）", edit: func(s *tactics.Setup) { s.Units[1].Stats.Mov = 3 }, cmd: attackFrom2, roll: 99},
			{name: "残り 1（移動 4 で費用 3）", edit: func(s *tactics.Setup) { s.Units[1].Stats.Mov = 4 }, cmd: attackFrom2, roll: 99, waiting: true},
			{name: "反撃で戦死", prep: func(t *testing.T, b *tactics.Battle) {
				b.SetHP(1, 1)
				// 騎兵の 1 撃は外れ（99, 99）、敵の反撃は当たり（0, 0）で必殺なし（99）。
				rolls(t, b, 99, 99, 0, 0, 99)
			}, cmd: attackFrom2},
			{name: "ボスを倒して勝ち", edit: func(s *tactics.Setup) { s.Units[3].Boss, s.Units[4].Boss = true, false },
				prep: func(t *testing.T, b *tactics.Battle) { b.SetHP(3, 1) }, cmd: attackFrom2, roll: 0, result: tactics.Won},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := cantoBattle(t, tc.edit, tc.roll)
				if tc.prep != nil {
					tc.prep(t, b)
				}
				mustCanto(t, b, tc.cmd)
				if _, ok := b.PendingCanto(); ok != tc.waiting {
					t.Errorf("再移動を待つ = %v, want %v", ok, tc.waiting)
				}
				if got := b.CantoReachable(); (got != nil) != tc.waiting {
					t.Errorf("行けるマス = %v（待つ %v）", got, tc.waiting)
				}
				if b.Result != tc.result {
					t.Errorf("結果 = %v, want %v", b.Result, tc.result)
				}
			})
		}
	})

	t.Run("TC-FR-BTL-03-20 再移動する・その場で終える。再移動のあとはもう一度行動できない", func(t *testing.T) {
		b := cantoBattle(t, nil, 99)
		mustCanto(t, b, attackFrom2)
		events := mustCanto(t, b, tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoMove})
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventMove})
		if e := events[0]; e.Unit != 1 || e.From != pos(2, 0) || e.To != pos(0, 0) {
			t.Errorf("移動 = %+v", e)
		}
		if u := b.Units()[1]; u.Pos != pos(0, 0) || !u.Acted {
			t.Errorf("騎兵 = %v・行動済み %v", u.Pos, u.Acted)
		}
		if _, ok := b.PendingCanto(); ok {
			t.Error("再移動のあとも待っている")
		}
		for _, cmd := range []tactics.Command{attackFrom2, {Unit: 1, To: pos(0, 0), Action: tactics.Wait}} {
			if _, err := b.Act(cmd); !errors.Is(err, tactics.ErrAlreadyActed) {
				t.Errorf("再移動のあとの %v: err = %v, want ErrAlreadyActed", cmd.Action, err)
			}
		}
		if _, err := b.Act(tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoMove}); !errors.Is(err, tactics.ErrNoCantoPending) {
			t.Errorf("2 回目の再移動: err = %v, want ErrNoCantoPending", err)
		}

		b = cantoBattle(t, nil, 99)
		mustCanto(t, b, attackFrom2)
		events = mustCanto(t, b, tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.CantoEnd})
		if len(events) != 0 {
			t.Errorf("その場で終える出来事 = %v", kinds(events))
		}
		if _, ok := b.PendingCanto(); ok || b.Units()[1].Pos != pos(2, 0) {
			t.Error("その場で終えたあとも待っている、または動いた")
		}
	})

	t.Run("TC-FR-BTL-03-20 最後の自軍も待っている間は敵軍の番へ進まず、ほかの行動と番の終わりを拒む", func(t *testing.T) {
		b := cantoBattle(t, nil, 99)
		mustCanto(t, b, tactics.Command{Unit: 0, To: pos(0, 2), Action: tactics.Wait})
		mustCanto(t, b, tactics.Command{Unit: 2, To: pos(1, 0), Action: tactics.Wait})
		events := mustCanto(t, b, attackFrom2)
		if indexOf(events, tactics.EventPhaseStart) >= 0 || b.Phase != tactics.Player || b.Turn != 1 {
			t.Fatalf("最後の自軍の行動で番が進んだ: %v", kinds(events))
		}
		noRolls(t, b)
		before := cantoSnapshot(b)
		if ev, err := b.EndPhase(); !errors.Is(err, tactics.ErrCantoPending) || ev != nil {
			t.Errorf("EndPhase = %v, %v, want ErrCantoPending", ev, err)
		}
		for _, cmd := range []tactics.Command{
			{Unit: 0, To: pos(0, 2), Action: tactics.Wait},
			{Unit: 1, To: pos(2, 0), Action: tactics.Attack, Target: 3},
			{Unit: 1, To: pos(2, 0), Action: tactics.Wait},
		} {
			if ev, err := b.Act(cmd); !errors.Is(err, tactics.ErrCantoPending) || ev != nil {
				t.Errorf("Act(%+v) = %v, %v, want ErrCantoPending", cmd, ev, err)
			}
		}
		assertEqual(t, "状態", cantoSnapshot(b), before)
		b.SetRoll(func() int { return 99 })
		events = mustCanto(t, b, tactics.Command{Unit: 1, To: pos(2, 0), Action: tactics.CantoEnd})
		if i := indexOf(events, tactics.EventPhaseStart); i < 0 || events[i].Side != tactics.Enemy {
			t.Errorf("再移動を終えても敵軍の番が始まらない: %v", kinds(events))
		}
		if b.Turn != 2 || b.Phase != tactics.Player {
			t.Errorf("ターン %d・番 %v, want 2・自軍", b.Turn, b.Phase)
		}
	})

	t.Run("TC-FR-BTL-03-20 正しくない再移動は状態・乱数・待っている再移動を変えない", func(t *testing.T) {
		cases := []struct {
			name string
			prep func(b *tactics.Battle)
			cmd  tactics.Command
			want error
		}{
			{name: "待っていない", prep: func(b *tactics.Battle) {}, cmd: tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoMove}, want: tactics.ErrNoCantoPending},
			{name: "待っていないのに終える", prep: func(b *tactics.Battle) {}, cmd: tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoEnd}, want: tactics.ErrNoCantoPending},
			{name: "勝敗が決まった", prep: func(b *tactics.Battle) { b.Result = tactics.Won }, cmd: tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoMove}, want: tactics.ErrBattleOver},
			{name: "ほかのユニット", cmd: tactics.Command{Unit: 2, To: pos(1, 0), Action: tactics.CantoMove}, want: tactics.ErrInvalidUnit},
			{name: "費用 5 のマス", cmd: tactics.Command{Unit: 1, To: pos(6, 0), Action: tactics.CantoMove}, want: tactics.ErrUnreachable},
			{name: "味方のいるマス", cmd: tactics.Command{Unit: 1, To: pos(1, 0), Action: tactics.CantoMove}, want: tactics.ErrUnreachable},
			{name: "敵のいるマス", cmd: tactics.Command{Unit: 1, To: pos(3, 0), Action: tactics.CantoMove}, want: tactics.ErrUnreachable},
			{name: "盤の外", cmd: tactics.Command{Unit: 1, To: pos(-1, 0), Action: tactics.CantoMove}, want: tactics.ErrUnreachable},
			{name: "終えるのに今の位置でない", cmd: tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoEnd}, want: tactics.ErrUnreachable},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := cantoBattle(t, nil, 99)
				if tc.prep != nil {
					tc.prep(b)
				} else {
					mustCanto(t, b, attackFrom2)
				}
				noRolls(t, b)
				before := cantoSnapshot(b)
				events, err := b.Act(tc.cmd)
				if !errors.Is(err, tc.want) || events != nil {
					t.Errorf("Act = %v, %v, want %v", events, err, tc.want)
				}
				assertEqual(t, "状態", cantoSnapshot(b), before)
			})
		}
	})
}

func TestCantoHistory(t *testing.T) {
	t.Run("TC-FR-BTL-10-31 移動 ＋ 行動 ＋ 再移動を 1 手とし、再移動だけの履歴を作らない", func(t *testing.T) {
		b := mustNew(t, cantoSetup())
		mustCanto(t, b, tactics.Command{Unit: 0, To: pos(0, 2), Action: tactics.Wait})
		mustCanto(t, b, attackFrom2)
		h := b.History()
		if len(h) != 2 || h[1].Kind != tactics.HistoryAction || h[1].Unit != 1 || h[1].Action != tactics.Attack || h[1].To != pos(2, 0) {
			t.Fatalf("履歴 = %+v", h)
		}
		pc, _ := b.PendingCanto()
		if pc.HistoryID != h[1].ID {
			t.Errorf("待っている再移動の履歴 ID = %d, want %d", pc.HistoryID, h[1].ID)
		}
		// 待っている間も、同じ 1 手の「前の状態」を下見できる。
		v, err := b.PreviewRewind(h[1].ID)
		if err != nil || v.Units()[1].Pos != pos(0, 0) || v.Units()[1].Acted {
			t.Fatalf("下見 = %v, %v", v.Units(), err)
		}
		end := h[1].EventEnd
		mustCanto(t, b, tactics.Command{Unit: 1, To: pos(0, 0), Action: tactics.CantoMove})
		h2 := b.History()
		if len(h2) != 2 || h2[1].ID != h[1].ID || h2[1].EventStart != h[1].EventStart || h2[1].EventEnd != end+1 {
			t.Fatalf("再移動のあとの履歴 = %+v（前 %+v）", h2, h)
		}
		if err := b.Rewind(h[1].ID); err != nil {
			t.Fatal(err)
		}
		if u := b.Units()[1]; u.Pos != pos(0, 0) || u.Acted || u.HP != u.Stats.HP {
			t.Errorf("巻き戻したあとの騎兵 = %+v", u)
		}
		if _, ok := b.PendingCanto(); ok || len(b.History()) != 1 || b.RewindsLeft() != 9 {
			t.Errorf("巻き戻したあと: 待っている %v・履歴 %d・残り %d", ok, len(b.History()), b.RewindsLeft())
		}
	})

	t.Run("TC-FR-BTL-10-31 再移動を待っている間に巻き戻すと、最初の移動の前に戻り、同じ入力で同じ結果になる", func(t *testing.T) {
		b := mustNew(t, cantoSetup())
		first := mustCanto(t, b, attackFrom2)
		id := b.History()[0].ID
		if err := b.Rewind(id); err != nil {
			t.Fatal(err)
		}
		if u := b.Units()[1]; u.Pos != pos(0, 0) || u.Acted {
			t.Errorf("巻き戻したあとの騎兵 = %v・行動済み %v", u.Pos, u.Acted)
		}
		if _, ok := b.PendingCanto(); ok || len(b.History()) != 0 {
			t.Error("巻き戻しても再移動を待っている、または履歴が残る")
		}
		again := mustCanto(t, b, attackFrom2)
		if len(first) != len(again) {
			t.Fatalf("同じ入力の出来事の数 %d ≠ %d", len(first), len(again))
		}
		for i := range first {
			if first[i].Kind != again[i].Kind || first[i].Hit != again[i].Hit || first[i].Amount != again[i].Amount || first[i].Crit != again[i].Crit {
				t.Errorf("出来事[%d] %+v ≠ %+v", i, first[i], again[i])
			}
		}
		if _, ok := b.PendingCanto(); !ok {
			t.Error("同じ入力のあとに再移動を待っていない")
		}
	})
}

func TestClassSkillsTactics(t *testing.T) {
	t.Run("TC-FR-UNT-07-08 キャラのスキルと兵種のスキルを合わせて判定し、同じ効果を二重にかけない", func(t *testing.T) {
		wrath := tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}
		cases := []struct {
			name  string
			skill tactics.Skill
			class []tactics.Skill
			crit  int
		}{
			// 必殺率 = 技 5 ÷ 2 ＋ 武器 0 ＋ 怒り − 相手の幸運 0。
			{name: "兵種の怒りも効く", class: []tactics.Skill{cantoSkill, wrath}, crit: 52},
			{name: "キャラの怒りと兵種の再移動を両方持てる", skill: wrath, class: []tactics.Skill{cantoSkill}, crit: 52},
			{name: "キャラと兵種の両方の怒りでも 1 回だけ", skill: wrath, class: []tactics.Skill{wrath, cantoSkill}, crit: 52},
			{name: "怒りがない", class: []tactics.Skill{cantoSkill}, crit: 2},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := cantoBattle(t, func(s *tactics.Setup) {
					s.Units[1].Skill = tc.skill
					s.Units[1].ClassSkills = tc.class
				}, 99)
				b.SetHP(1, 10)
				f, err := b.Forecast(attackFrom2)
				if err != nil {
					t.Fatal(err)
				}
				if f.Attacker.CritRate != tc.crit {
					t.Errorf("必殺率 = %d, want %d", f.Attacker.CritRate, tc.crit)
				}
				n := 0
				for _, sk := range f.Attacker.Skills {
					if sk.Effect == tactics.Wrath {
						n++
					}
				}
				if want := min(1, tc.crit-2); n != want {
					t.Errorf("予測の怒りの数 = %d, want %d", n, want)
				}
				mustCanto(t, b, attackFrom2)
				if _, ok := b.PendingCanto(); !ok {
					t.Error("兵種の再移動が効かない")
				}
			})
		}
	})

	t.Run("TC-FR-UNT-07-08 兵種のスキルの効果がない・範囲外・重なり・数値の誤りを拒む", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			class []tactics.Skill
		}{
			{"効果なし", []tactics.Skill{{Name: "なし"}}},
			{"範囲外の効果", []tactics.Skill{{Name: "?", Effect: tactics.Canto + 1}}},
			{"同じ効果の重なり", []tactics.Skill{cantoSkill, cantoSkill}},
			{"確率 101", []tactics.Skill{{Name: "大盾", Effect: tactics.Pavise, Chance: 101}}},
			{"怒りの倍率", []tactics.Skill{{Name: "怒り", Effect: tactics.Wrath, Multiplier: 3}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := cantoSetup()
				s.Units[1].ClassSkills = tc.class
				if _, _, err := tactics.New(s, 1); !errors.Is(err, tactics.ErrInvalidSetup) {
					t.Errorf("New: err = %v, want ErrInvalidSetup", err)
				}
			})
		}
		s := cantoSetup()
		s.Units[1].Skill = tactics.Skill{Name: "再移動", Effect: tactics.Canto}
		if _, _, err := tactics.New(s, 1); err != nil {
			t.Errorf("キャラと兵種の両方の再移動: %v", err)
		}
	})

	t.Run("TC-FR-UNT-07-08 兵種のスキルは入力・返した値から独立する", func(t *testing.T) {
		s := cantoSetup()
		b := mustNew(t, s)
		s.Units[1].ClassSkills[0].Effect = tactics.Wrath
		units := b.Units()
		units[1].ClassSkills[0].Effect = tactics.Wrath
		if got := b.Units()[1].ClassSkills; len(got) != 1 || got[0] != cantoSkill {
			t.Errorf("兵種のスキル = %+v", got)
		}
	})
}

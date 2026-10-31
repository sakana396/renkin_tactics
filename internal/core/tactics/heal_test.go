package tactics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 回復の行動（DES-HEL-01）と回復の予測（DES-HEL-02）のテスト。
// 計算例の番号は設計書（反復 2）4.2 節（計算例 7）と 4.4 節（使用回数の計算例 1〜4）のもの。
// リナと癒やしの杖は設計書 10.1・10.3 節の値（v0.3）。

var (
	healStaff   = tactics.Weapon{Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, MinRange: 1, MaxRange: 1, MaxUses: 15}
	growthRina  = tactics.Growth{HP: 55, Str: 0, Skl: 40, Spd: 60, Def: 30}
	potionStack = tactics.ItemStack{Item: tactics.Item{ID: "potion", Name: "回復薬", Heal: 10}, Count: 1}
)

// rina は v0.3 のリナ（僧侶。16/0/2/6/2/5、癒やしの杖）。
func rina(x, y, level, exp int) tactics.UnitSetup {
	u := chara("リナ", tactics.Player, x, y, tactics.Stats{HP: 16, Str: 0, Skl: 2, Spd: 6, Def: 2, Mov: 5}, healStaff, tactics.Skill{})
	return leveled(u, level, exp, growthRina)
}

// healSetup は回復のテストの盤面（7 × 7）。
//
//	0: リナ (2, 1)（Lv 1、経験 0）
//	1: レオン（主人公。Lv 0）(3, 1)
//	2: ボス (6, 0)
//	3: ソルジャー (1, 1)
//	4: ガルド (2, 2)（healBattle で倒れた状態にする）
func healSetup() tactics.Setup {
	return arena(rina(2, 1, 1, 0), leon3(3, 1, 0, 0), farBoss(6, 0), soldier(1, 1), gald(2, 2))
}

// healBattle は healSetup の戦闘を作り、レオンの HP を 12 にし、ガルドを倒れた状態にしてから prepare で整える。
// 敵軍の番には、どの敵も動かない。
func healBattle(t *testing.T, prepare func(b *tactics.Battle)) *tactics.Battle {
	t.Helper()
	b := mustNew(t, healSetup())
	b.SetHP(1, 12)
	b.KillUnit(4)
	b.SetEnemyTarget(func(int) int { return -1 })
	if prepare != nil {
		prepare(b)
	}
	return b
}

// healCmd は、unit が to に移動して target を回復するコマンド。
func healCmd(unit int, to tactics.Pos, target int) tactics.Command {
	return tactics.Command{Unit: unit, To: to, Action: tactics.Heal, Target: target}
}

// uses は、ユニットの杖の残りと最大。
func uses(u tactics.Unit) [2]int { return [2]int{u.EquippedWeapon().Uses, u.EquippedWeapon().MaxUses} }

// TestHeal は回復の行動を確かめる（DES-HEL-01、FR-BTL-03）。
func TestHeal(t *testing.T) {
	t.Run("TC-FR-BTL-03-08 使用回数の計算例 1（回復の流れと出来事）", func(t *testing.T) {
		t.Run("移動しない", func(t *testing.T) {
			b := healBattle(t, nil)
			noRolls(t, b)
			events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
			assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventHeal, tactics.EventExp})
			assertEqual(t, "Heal", heals(events), []heal{{Unit: 1, Amount: 7, Item: "癒やしの杖"}})
			assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 10}})
			snap := events[0].Units
			assertEqual(t, "Heal の写しの杖", uses(snap[0]), [2]int{14, 15})
			assertEqual(t, "Heal の写しのレオン", snap[1].HP, 19)
			assertEqual(t, "Heal の写しのリナは行動済み", snap[0].Acted, true)
			u := b.Units()
			assertEqual(t, "杖", uses(u[0]), [2]int{14, 15})
			assertEqual(t, "レオン", [2]int{u[1].HP, u[1].Stats.HP}, [2]int{19, 19})
			assertEqual(t, "リナは行動済み", u[0].Acted, true)
			assertEqual(t, "リナの経験", u[0].Exp, 10)
			assertEqual(t, "番", [2]any{b.Turn, b.Phase}, [2]any{1, tactics.Player})
		})
		t.Run("移動してから回復する", func(t *testing.T) {
			b := healBattle(t, nil)
			noRolls(t, b)
			events := mustAct(t, b, healCmd(0, pos(3, 2), 1))
			assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventMove, tactics.EventHeal, tactics.EventExp})
			assertEqual(t, "Move の写しの杖", uses(events[0].Units[0]), [2]int{15, 15})
			assertEqual(t, "Move の写しのレオン", events[0].Units[1].HP, 12)
			assertEqual(t, "位置", b.Units()[0].Pos, pos(3, 2))
			assertEqual(t, "杖", uses(b.Units()[0]), [2]int{14, 15})
		})
		t.Run("最後の自軍が回復すると敵軍の番に進む", func(t *testing.T) {
			b := healBattle(t, nil)
			mustAct(t, b, tactics.Command{Unit: 1, To: pos(3, 1), Action: tactics.Wait})
			noRolls(t, b)
			events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
			ks := kinds(events)
			assertEqual(t, "初めの出来事", ks[:3], []tactics.EventKind{tactics.EventHeal, tactics.EventExp, tactics.EventPhaseStart})
			assertEqual(t, "番", [2]any{b.Turn, b.Phase}, [2]any{2, tactics.Player})
		})
	})

	t.Run("TC-FR-BTL-03-09 回復量は最大 HP を超えない", func(t *testing.T) {
		for _, tc := range []struct{ hp, amount, after int }{{12, 7, 19}, {5, 10, 15}, {18, 1, 19}} {
			b := healBattle(t, func(b *tactics.Battle) { b.SetHP(1, tc.hp) })
			events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
			assertEqual(t, "Heal", heals(events), []heal{{Unit: 1, Amount: tc.amount, Item: "癒やしの杖"}})
			assertEqual(t, "HP", b.Units()[1].HP, tc.after)
		}
	})

	errCases := []actErrorCase{
		{"ErrCannotHeal（剣のレオン）", func(b *tactics.Battle) { b.SetHP(0, 10) }, healCmd(1, pos(3, 1), 0), tactics.ErrCannotHeal},
		{"ErrNoUses", func(b *tactics.Battle) { b.SetUses(0, 0) }, healCmd(0, pos(2, 1), 1), tactics.ErrNoUses},
		{"ErrNoTarget（-1）", nil, healCmd(0, pos(2, 1), -1), tactics.ErrNoTarget},
		{"ErrNoTarget（数を超える）", nil, healCmd(0, pos(2, 1), 5), tactics.ErrNoTarget},
		{"ErrNoTarget（敵）", func(b *tactics.Battle) { b.SetHP(3, 10) }, healCmd(0, pos(2, 1), 3), tactics.ErrNoTarget},
		{"ErrNoTarget（倒れた味方）", nil, healCmd(0, pos(2, 1), 4), tactics.ErrNoTarget},
		{"ErrNoTarget（自分）", func(b *tactics.Battle) { b.SetHP(0, 10) }, healCmd(0, pos(2, 1), 0), tactics.ErrNoTarget},
		{"ErrOutOfRange", nil, healCmd(0, pos(1, 2), 1), tactics.ErrOutOfRange},
		{"ErrFullHP", func(b *tactics.Battle) { b.SetHP(1, 19) }, healCmd(0, pos(2, 1), 1), tactics.ErrFullHP},
	}
	orderCases := []actErrorCase{
		{"0 が 1〜6 より先", over, tactics.Command{Unit: 99, To: pos(9, 9), Action: tactics.Action(6), Target: 99}, tactics.ErrInvalidAction},
		{"5 が 6a より先", nil, healCmd(1, pos(6, 6), 0), tactics.ErrUnreachable},
		{"6a が 6b・6c より先", func(b *tactics.Battle) { b.SetUses(1, 0) }, healCmd(1, pos(3, 1), -1), tactics.ErrCannotHeal},
		{"6b が 6c より先", func(b *tactics.Battle) { b.SetUses(0, 0) }, healCmd(0, pos(2, 1), -1), tactics.ErrNoUses},
		{"6c が 6d・6e より先（自分は距離 0 で HP が最大）", nil, healCmd(0, pos(2, 1), 0), tactics.ErrNoTarget},
		{"6c が 6d より先（射程の外の敵）", nil, healCmd(0, pos(4, 1), 3), tactics.ErrNoTarget},
		{"6d が 6e より先", func(b *tactics.Battle) { b.SetHP(1, 19) }, healCmd(0, pos(1, 2), 1), tactics.ErrOutOfRange},
		{"杖の Attack は ErrCannotAttack を ErrNoTarget より先", nil,
			tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Attack, Target: -1}, tactics.ErrCannotAttack},
		{"杖の Attack でも 5 が ErrCannotAttack より先", nil,
			tactics.Command{Unit: 0, To: pos(6, 6), Action: tactics.Attack, Target: 3}, tactics.ErrUnreachable},
	}
	groups := []struct {
		id    string
		cases []actErrorCase
	}{
		{"TC-FR-BTL-03-10", errCases},
		{"TC-NFR-07-06", orderCases},
	}
	for _, g := range groups {
		for _, tc := range g.cases {
			t.Run(g.id+" "+tc.name, func(t *testing.T) {
				b := healBattle(t, tc.prepare)
				noRolls(t, b)
				before := snapshot(b)
				events, err := b.Act(tc.cmd)
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				if events != nil {
					t.Errorf("出来事 = %v, want nil", events)
				}
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
	}

	t.Run("TC-FR-BTL-03-11 杖のユニットは攻撃できない。回数を使い切っても待機と回復薬は使える", func(t *testing.T) {
		prepare := func(b *tactics.Battle) {
			b.SetHP(0, 10)
			b.SetUses(0, 0)
		}
		withPotion := func(t *testing.T) *tactics.Battle {
			s := healSetup()
			s.Units[0].Items = []tactics.ItemStack{potionStack}
			b := mustNew(t, s)
			b.KillUnit(4)
			prepare(b)
			return b
		}

		b := withPotion(t)
		before := snapshot(b)
		_, err := b.Act(tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Attack, Target: 3})
		if !errors.Is(err, tactics.ErrCannotAttack) {
			t.Errorf("Attack: err = %v, want ErrCannotAttack", err)
		}
		assertEqual(t, "Attack のあとの状態", snapshot(b), before)

		b = withPotion(t)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Wait})
		assertEqual(t, "Wait の出来事", kinds(events), []tactics.EventKind{tactics.EventWait})

		b = withPotion(t)
		events = mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.UseItem, Item: 0})
		assertEqual(t, "UseItem の Heal", heals(events), []heal{{Unit: 0, Amount: 6, Item: "回復薬"}})
		assertEqual(t, "HP", b.Units()[0].HP, 16)
	})

	t.Run("TC-FR-BTL-03-12 / TC-FR-UNT-04-13 使用回数の計算例 2・4（15 回使い切ると ErrNoUses。回復の経験は 150 まで）", func(t *testing.T) {
		b := healBattle(t, nil)
		// 10 回目の回復でレベルが上がり、成長の乱数を 5 つだけ引く。
		done := rolls(t, b, 99, 99, 99, 99, 99, 99, 99, 99)
		for i := 1; i <= 15; i++ {
			b.SetHP(1, 12)
			if _, err := b.Act(healCmd(0, pos(2, 1), 1)); err != nil {
				t.Fatalf("%d 回目: %v", i, err)
			}
			if _, err := b.EndPhase(); err != nil {
				t.Fatalf("%d 回目の EndPhase: %v", i, err)
			}
		}
		done()
		u := b.Units()[0]
		assertEqual(t, "杖", uses(u), [2]int{0, 15})
		assertEqual(t, "レベルと経験", [2]int{u.Level, u.Exp}, [2]int{2, 50})

		b.SetHP(1, 12)
		noRolls(t, b)
		before := snapshot(b)
		events, err := b.Act(healCmd(0, pos(2, 1), 1))
		if !errors.Is(err, tactics.ErrNoUses) || events != nil {
			t.Errorf("16 回目: 出来事 %v、err = %v, want ErrNoUses", events, err)
		}
		f, err := b.HealForecast(healCmd(0, pos(2, 1), 1))
		if !errors.Is(err, tactics.ErrNoUses) {
			t.Errorf("HealForecast: err = %v, want ErrNoUses", err)
		}
		assertEqual(t, "予測", f, tactics.HealForecast{})
		assertEqual(t, "状態", snapshot(b), before)
		mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Wait})
	})

	t.Run("TC-FR-BTL-03-13 使用回数の計算例 3（New は杖を満タンにして始める）", func(t *testing.T) {
		for _, n := range []int{0, 3, 99} {
			s := healSetup()
			s.Units[0].Weapon.Uses = n
			b := mustNew(t, s)
			assertEqual(t, "杖", uses(b.Units()[0]), [2]int{15, 15})
			assertEqual(t, "Setup は変えない", s.Units[0].Weapon.Uses, n)
		}
	})
}

// TestHealSetup は Setup の規則 8 を確かめる（DES-HEL-01、DES-BTL-01）。
func TestHealSetup(t *testing.T) {
	t.Run("TC-FR-BTL-03-14 規則 8 に当てはまらなければ ErrInvalidSetup", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			modify func(s *tactics.Setup)
			reason string
		}{
			{"杖の Heal が 0", func(s *tactics.Setup) { s.Units[1].Weapon.Heal = 0 }, "units[1]: 杖の回復量 0"},
			{"杖の MaxUses が 0", func(s *tactics.Setup) { s.Units[1].Weapon.MaxUses = 0 }, "units[1]: 杖の使用回数 0"},
			{"剣の Heal が 1", func(s *tactics.Setup) { s.Units[0].Weapon.Heal = 1 }, "units[0]: 杖でない武器"},
			{"剣の Uses が 1", func(s *tactics.Setup) { s.Units[0].Weapon.Uses = 1 }, "units[0]: 杖でない武器"},
			{"剣の MaxUses が 1", func(s *tactics.Setup) { s.Units[0].Weapon.MaxUses = 1 }, "units[0]: 杖でない武器"},
			{"敵の武器が杖", func(s *tactics.Setup) { s.Units[3].Weapon = healStaff }, "units[3]: 敵の武器"},
			{"増援の武器が杖", func(s *tactics.Setup) { s.Reinforcements[0].Unit.Weapon = healStaff }, "reinforcements[0]: 敵の武器"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := baseSetup()
				s.Units[1].Weapon = healStaff
				s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 2, 2)}}
				if _, _, err := tactics.New(s, 1); err != nil {
					t.Fatalf("変える前: %v", err)
				}
				tc.modify(&s)
				_, _, err := tactics.New(s, 1)
				if !errors.Is(err, tactics.ErrInvalidSetup) {
					t.Fatalf("err = %v, want ErrInvalidSetup", err)
				}
				if !strings.Contains(err.Error(), tc.reason) {
					t.Errorf("err = %q, 理由 %q を含まない", err, tc.reason)
				}
			})
		}
	})

	t.Run("TC-FR-BTL-03-15 杖の Heal 1・MaxUses 1 で作れ、1 回で使い切る", func(t *testing.T) {
		s := healSetup()
		s.Units[0].Weapon.Heal = 1
		s.Units[0].Weapon.MaxUses = 1
		b := mustNew(t, s)
		b.SetHP(1, 12)
		assertEqual(t, "初めの杖", uses(b.Units()[0]), [2]int{1, 1})
		events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
		assertEqual(t, "Heal", heals(events), []heal{{Unit: 1, Amount: 1, Item: "癒やしの杖"}})
		assertEqual(t, "回復したあとの杖", uses(b.Units()[0]), [2]int{0, 1})
		b.SetEnemyTarget(func(int) int { return -1 })
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Act(healCmd(0, pos(2, 1), 1)); !errors.Is(err, tactics.ErrNoUses) {
			t.Errorf("2 回目: err = %v, want ErrNoUses", err)
		}
	})
}

// TestStaffNoCounter は、杖のユニットが反撃しないことを確かめる（DES-HEL-01、DES-BTL-06）。
func TestStaffNoCounter(t *testing.T) {
	t.Run("TC-FR-BTL-06-07 杖のユニットは反撃も追撃もしない", func(t *testing.T) {
		t.Run("リナが 4 速い", func(t *testing.T) {
			r := rina(2, 1, 1, 0)
			r.Stats.Spd = 8 // ファイター（速さ 4）より 4 速い
			b := mustNew(t, arena(r, fighter(3, 1), farBoss(6, 0), leon3(0, 6, 0, 0)))
			done := rolls(t, b, 0, 0, 99)
			events := enemyAttacks(t, b, 1, 0)
			done()
			assertEqual(t, "Strike", strikes(events), []strike{{Unit: 1, Target: 0, Hit: true, Amount: 13}})
			assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 1}})
			assertEqual(t, "リナの HP", b.Units()[0].HP, 3)
		})
		t.Run("ファイターが 4 速い", func(t *testing.T) {
			f := fighter(3, 1)
			f.Stats.Spd = 10
			b := mustNew(t, arena(rina(2, 1, 1, 0), f, farBoss(6, 0), leon3(0, 6, 0, 0)))
			done := rolls(t, b, 0, 0, 99, 99, 99)
			events := enemyAttacks(t, b, 1, 0)
			done()
			assertEqual(t, "Strike", strikes(events), []strike{
				{Unit: 1, Target: 0, Hit: true, Amount: 13},
				{Unit: 1, Target: 0, Hit: false},
			})
		})
	})
}

// TestHealForecast は回復の予測を確かめる（DES-HEL-02、FR-BTL-08）。
func TestHealForecast(t *testing.T) {
	t.Run("TC-FR-BTL-08-18 回復の予測の計算例", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			prepare func(b *tactics.Battle)
			to      tactics.Pos
			want    tactics.HealForecast
		}{
			{"HP 12/19、移動しない", nil, pos(2, 1),
				tactics.HealForecast{Healer: "リナ", Target: "レオン", HP: 12, MaxHP: 19, Amount: 7, After: 19, Uses: 15, MaxUses: 15}},
			{"HP 5/19、移動する", func(b *tactics.Battle) { b.SetHP(1, 5) }, pos(3, 2),
				tactics.HealForecast{Healer: "リナ", Target: "レオン", HP: 5, MaxHP: 19, Amount: 10, After: 15, Uses: 15, MaxUses: 15}},
			{"杖の残り 12", func(b *tactics.Battle) { b.SetUses(0, 12) }, pos(2, 1),
				tactics.HealForecast{Healer: "リナ", Target: "レオン", HP: 12, MaxHP: 19, Amount: 7, After: 19, Uses: 12, MaxUses: 15}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				b := healBattle(t, tc.prepare)
				noRolls(t, b)
				before := snapshot(b)
				f, err := b.HealForecast(healCmd(0, tc.to, 1))
				if err != nil {
					t.Fatalf("HealForecast: %v", err)
				}
				assertEqual(t, "予測", f, tc.want)
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
	})

	t.Run("TC-FR-BTL-08-19 回復の予測の誤り", func(t *testing.T) {
		for _, tc := range []actErrorCase{
			{"Attack は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Attack, Target: 1}, tactics.ErrInvalidAction},
			{"Wait は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Wait, Target: 1}, tactics.ErrInvalidAction},
			{"UseItem は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.UseItem, Target: 1}, tactics.ErrInvalidAction},
			{"範囲外の Action は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Action(6), Target: 1}, tactics.ErrInvalidAction},
			{"Heal でないことが戦闘の終わりより先", over, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Wait, Target: 1}, tactics.ErrInvalidAction},
			{"ErrBattleOver", over, healCmd(0, pos(2, 1), 1), tactics.ErrBattleOver},
			{"ErrNotPlayerPhase", enemyPhase, healCmd(0, pos(2, 1), 1), tactics.ErrNotPlayerPhase},
			{"ErrInvalidUnit（敵）", nil, healCmd(3, pos(1, 1), 0), tactics.ErrInvalidUnit},
			{"ErrUnreachable", nil, healCmd(0, pos(6, 6), 1), tactics.ErrUnreachable},
			{"ErrCannotHeal", func(b *tactics.Battle) { b.SetHP(0, 10) }, healCmd(1, pos(3, 1), 0), tactics.ErrCannotHeal},
			{"ErrNoUses", func(b *tactics.Battle) { b.SetUses(0, 0) }, healCmd(0, pos(2, 1), 1), tactics.ErrNoUses},
			{"ErrNoTarget（敵）", nil, healCmd(0, pos(2, 1), 3), tactics.ErrNoTarget},
			{"ErrOutOfRange", nil, healCmd(0, pos(1, 2), 1), tactics.ErrOutOfRange},
			{"ErrFullHP", func(b *tactics.Battle) { b.SetHP(1, 19) }, healCmd(0, pos(2, 1), 1), tactics.ErrFullHP},
		} {
			t.Run("HealForecast "+tc.name, func(t *testing.T) {
				b := healBattle(t, tc.prepare)
				noRolls(t, b)
				before := snapshot(b)
				f, err := b.HealForecast(tc.cmd)
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				assertEqual(t, "予測", f, tactics.HealForecast{})
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
		for _, tc := range []actErrorCase{
			{"Heal は ErrInvalidAction", nil, healCmd(0, pos(2, 1), 1), tactics.ErrInvalidAction},
			{"杖のユニットの Attack は ErrCannotAttack", nil, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.Attack, Target: 3}, tactics.ErrCannotAttack},
		} {
			t.Run("Forecast "+tc.name, func(t *testing.T) {
				b := healBattle(t, tc.prepare)
				noRolls(t, b)
				before := snapshot(b)
				f, err := b.Forecast(tc.cmd)
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				assertEqual(t, "予測", f, tactics.Forecast{})
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
	})
}

// TestHealExp は回復の経験を確かめる（DES-GRW-01、FR-UNT-04）。
func TestHealExp(t *testing.T) {
	t.Run("TC-FR-UNT-04-11 計算例 7（回復で経験 10。レベルが上がれば成長の乱数だけを引く）", func(t *testing.T) {
		b := mustNew(t, arena(rina(2, 1, 1, 95), leon3(3, 1, 0, 0), farBoss(6, 0), reserve()))
		b.SetHP(1, 12)
		done := rolls(t, b, 10, 0, 50, 59, 40, 99, 99, 99)
		events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventHeal, tactics.EventExp, tactics.EventLevelUp})
		assertEqual(t, "Heal", heals(events), []heal{{Unit: 1, Amount: 7, Item: "癒やしの杖"}})
		assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 10}})
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 2, Gains: tactics.Stats{HP: 1, Spd: 1}}})
		assertEqual(t, "リナ", grownOf(b.Units()[0]), grown{
			Level: 2, Exp: 5, Stats: tactics.Stats{HP: 17, Str: 0, Skl: 2, Spd: 7, Def: 2, Mov: 5}, HP: 17,
		})
		assertEqual(t, "レオン", b.Units()[1].HP, 19)
	})

	t.Run("TC-FR-UNT-04-12 Lv 0・Lv 20 の杖のユニットと、回復される側は経験を得ない", func(t *testing.T) {
		for _, level := range []int{0, 20} {
			b := mustNew(t, arena(rina(2, 1, level, 0), leon3(3, 1, 1, 0), farBoss(6, 0), reserve()))
			b.SetHP(1, 12)
			noRolls(t, b)
			events := mustAct(t, b, healCmd(0, pos(2, 1), 1))
			assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventHeal})
			u := b.Units()
			assertEqual(t, "リナ", [2]int{u[0].Level, u[0].Exp}, [2]int{level, 0})
			assertEqual(t, "レオン", [2]int{u[1].Level, u[1].Exp}, [2]int{1, 0})
		}
	})
}

// TestHealDeterminism は回復を含むコマンドの列の再現性を確かめる（DES-RNG-01、NFR-06）。
func TestHealDeterminism(t *testing.T) {
	t.Run("TC-NFR-06-06 同じ種と同じコマンドの列なら、回復・経験・成長も同じ", func(t *testing.T) {
		run := func() ([]tactics.Event, state) {
			s := arena(rina(2, 1, 1, 95), leon3(3, 1, 0, 0), farBoss(6, 0), soldier(5, 5))
			b, events, err := tactics.New(s, 5)
			if err != nil {
				t.Fatal(err)
			}
			b.SetHP(1, 12)
			for _, c := range []tactics.Command{
				healCmd(0, pos(3, 2), 1),
				{Unit: 1, To: pos(3, 1), Action: tactics.Wait},
			} {
				ev, err := b.Act(c)
				if err != nil {
					t.Fatal(err)
				}
				events = append(events, ev...)
			}
			return events, snapshot(b)
		}
		e1, s1 := run()
		e2, s2 := run()
		if len(levelUps(e1)) == 0 || len(heals(e1)) == 0 {
			t.Fatal("Heal と LevelUp が出ていない（確かめる場面になっていない）")
		}
		assertEqual(t, "出来事", e1, e2)
		assertEqual(t, "状態", s1, s2)
	})
}

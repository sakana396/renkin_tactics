package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 反復 6 の杖の回復 ＋ 魔力（DES-HEL-01・02 の改訂）、撃破の経験のレベル差（DES-GRW-04）、
// 敵のレベル（DES-IT06 7.3 節）の試験。期待値は DES-IT06 7.1・7.2 節の例 H1・G1〜G4 から求める。

// staffSetup は、2 × 3 の平地に、(0, 0) の杖の回復役（Lv 1、魔力 mag、兵種の上限 healCap）と、
// (1, 0) の最大 HP 24 の仲間、(2, 1) のボスを置く。
func staffSetup(mag, healCap int) tactics.Setup {
	healer := lord(0, 0, 0)
	healer.Name = "回復役"
	healer.Level = 1
	healer.Stats.Mag = mag
	healer.StaffHealCap = healCap
	healer.Weapon = healStaff
	friend := ally("仲間", 1, 0, 0)
	friend.Stats.HP = 24
	return tactics.Setup{Terrain: grid("...", "..."), Units: []tactics.UnitSetup{healer, friend, boss(2, 1)}}
}

func TestIT06StaffHeal(t *testing.T) {
	// 癒やしの杖の Heal は 10。基本の回復量 = 10 ＋ 魔力 → 兵種の上限 → 相手の足りない HP。
	for _, tc := range []struct {
		name         string
		mag, healCap int
		hp, want     int
	}{
		{"H1 リナ（魔力 0）は 10 回復して 20/24", 0, 0, 10, 10},
		{"H1 ハンナ（魔力 3）は 13 回復して 23/24", 3, 0, 10, 13},
		{"H1 ノエル（錬金術師、魔力 3、上限 8）は 13 → 8 で 18/24", 3, 8, 10, 8},
		{"H1 育ったリナ（魔力 4）は min(14, 14) で 24/24", 4, 0, 10, 14},
		{"魔力 4 でも足りない 4 だけ回復する", 4, 0, 20, 4},
		{"上限 8 より足りない HP 4 が小さければ 4", 3, 8, 20, 4},
		{"上限が基本の回復量より大きければ効かない", 3, 20, 10, 13},
	} {
		t.Run("TC-FR-BTL-03-19 "+tc.name, func(t *testing.T) {
			b := mustNew(t, staffSetup(tc.mag, tc.healCap))
			b.SetHP(1, tc.hp)
			cmd := healCmd(0, pos(0, 0), 1)
			f, err := b.HealForecast(cmd)
			if err != nil {
				t.Fatalf("HealForecast: %v", err)
			}
			assertEqual(t, "予測の回復量と回復後", [2]int{f.Amount, f.After}, [2]int{tc.want, tc.hp + tc.want})
			assertEqual(t, "予測で HP が変わらない", b.Units()[1].HP, tc.hp)
			noRolls(t, b)
			events := mustAct(t, b, cmd)
			assertEqual(t, "Heal", heals(events), []heal{{Unit: 1, Amount: tc.want, Item: "癒やしの杖"}})
			assertEqual(t, "実際の HP は予測と一致", b.Units()[1].HP, f.After)
			assertEqual(t, "杖の残り", uses(b.Units()[0]), [2]int{14, 15})
			assertEqual(t, "回復役の経験", exps(events), []expEvent{{Unit: 0, Amount: 10}})
		})
	}

	t.Run("TC-FR-BTL-03-19 本体のノエルは18/24まで回復", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		noel, _ := cat.Character("noel")
		class, _ := cat.Class(noel.Class)
		staff, _ := cat.Weapon("heal_staff")
		if class.Base.Mag != 3 || class.StaffHealCap != 8 || staff.Heal != 10 {
			t.Fatalf("本体の魔力・上限・杖 = %d・%d・%d", class.Base.Mag, class.StaffHealCap, staff.Heal)
		}
		setup := staffSetup(class.Base.Mag, class.StaffHealCap)
		setup.Units[0].Weapon.Heal = staff.Heal
		b := mustNew(t, setup)
		b.SetHP(1, 10)
		cmd := healCmd(0, pos(0, 0), 1)
		forecast, err := b.HealForecast(cmd)
		if err != nil || forecast.Amount != 8 || forecast.After != 18 || forecast.MaxHP != 24 {
			t.Fatalf("予測 = %+v, err = %v", forecast, err)
		}
		if _, err := b.Act(cmd); err != nil || b.Units()[1].HP != 18 {
			t.Fatalf("実際の回復 = %d, err = %v", b.Units()[1].HP, err)
		}
	})

	t.Run("TC-FR-BTL-03-19 満タンと回数 0 は予測も行動も拒み、状態を変えない", func(t *testing.T) {
		b := mustNew(t, staffSetup(3, 8))
		cmd := healCmd(0, pos(0, 0), 1)
		if _, err := b.HealForecast(cmd); !errors.Is(err, tactics.ErrFullHP) {
			t.Fatalf("満タンの予測: err = %v, want ErrFullHP", err)
		}
		if _, err := b.Act(cmd); !errors.Is(err, tactics.ErrFullHP) {
			t.Fatalf("満タンの回復: err = %v, want ErrFullHP", err)
		}
		b.SetHP(1, 10)
		b.SetUses(0, 0)
		if _, err := b.HealForecast(cmd); !errors.Is(err, tactics.ErrNoUses) {
			t.Fatalf("回数 0 の予測: err = %v, want ErrNoUses", err)
		}
		if _, err := b.Act(cmd); !errors.Is(err, tactics.ErrNoUses) {
			t.Fatalf("回数 0 の回復: err = %v, want ErrNoUses", err)
		}
		assertEqual(t, "HP", b.Units()[1].HP, 10)
	})

	t.Run("TC-FR-BTL-03-19 回復薬には魔力を足さない", func(t *testing.T) {
		s := staffSetup(5, 0)
		s.Units[0].Weapon = tactics.Weapon{Name: "鉄の剣", Type: tactics.Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
		s.Units[0].Stats.HP = 24
		s.Units[0].Items = []tactics.ItemStack{potionStack}
		b := mustNew(t, s)
		b.SetHP(0, 10)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: 0})
		assertEqual(t, "Heal", heals(events), []heal{{Unit: 0, Amount: 10, Item: "回復薬"}})
	})

	t.Run("TC-FR-BTL-03-19 兵種の上限が負なら ErrInvalidSetup", func(t *testing.T) {
		if _, _, err := tactics.New(staffSetup(3, -1), 1); !errors.Is(err, tactics.ErrInvalidSetup) {
			t.Fatalf("err = %v, want ErrInvalidSetup", err)
		}
	})
}

func TestIT06KillExp(t *testing.T) {
	// 倒したときの経験 = clamp(30 ＋ (敵の Level − 自軍の Level) × 3, 10, 50)。倒す前のレベルで計算する。
	for _, tc := range []struct {
		name               string
		own, exp, other    int
		amount             int // 0 なら Exp を出さない。
		wantLevel, wantExp int
	}{
		{"G1 Lv17 が Lv8 を倒すと 3 → 下限の 10", 17, 0, 8, 10, 17, 10},
		{"G2 Lv6 が Lv10 を倒すと 42", 6, 0, 10, 42, 6, 42},
		{"G3 Lv1 が Lv20 を倒すと 87 → 上限の 50", 1, 0, 20, 50, 1, 50},
		{"G4 Lv6・経験 99 が Lv10 を倒すと 141 → Lv7・経験 41（成長の前の Lv6 で計算）", 6, 99, 10, 42, 7, 41},
		{"差 −7 は 9 → 10", 8, 0, 1, 10, 8, 10},
		{"差 −6 は 12", 7, 0, 1, 12, 7, 12},
		{"差 ＋6 は 48", 1, 0, 7, 48, 1, 48},
		{"差 ＋7 は 51 → 50", 1, 0, 8, 50, 1, 50},
		{"同じレベルなら 30", 5, 0, 5, 30, 5, 30},
		{"Lv20 は経験を得ない", 20, 0, 1, 0, 20, 0},
	} {
		t.Run("TC-FR-UNT-04-19 "+tc.name, func(t *testing.T) {
			a := lord(0, 0, 0)
			a.Level, a.Exp = tc.own, tc.exp
			a.Stats.Str = 100
			a.Weapon.Hit = 1000
			d := boss(1, 0)
			d.Level = tc.other
			d.Stats.Lck = 100 // 必殺率 0 にして、乱数で結果が変わらないようにする。
			b := mustNew(t, tactics.Setup{Terrain: grid(".."), Units: []tactics.UnitSetup{a, d}})
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.Attack, Target: 1})
			var want []expEvent
			if tc.amount > 0 {
				want = []expEvent{{Unit: 0, Amount: tc.amount}}
			}
			assertEqual(t, "Exp", exps(events), want)
			u := b.Units()
			assertEqual(t, "自軍のレベルと経験", [2]int{u[0].Level, u[0].Exp}, [2]int{tc.wantLevel, tc.wantExp})
			assertEqual(t, "敵のレベルと経験は変わらない", [2]int{u[1].Level, u[1].Exp}, [2]int{tc.other, 0})
		})
	}

	t.Run("TC-FR-UNT-04-19 敵が自軍を倒しても敵は経験を得ない", func(t *testing.T) {
		a := lord(0, 0, 0)
		a.Level = 1
		e := foe("敵", 1, 1)
		e.Level = 10
		e.Stats.Str = 100
		e.Weapon.Hit = 1000
		f := ally("仲間", 0, 1, 0)
		f.Level = 1
		f.Stats.Lck = 100
		b := mustNew(t, tactics.Setup{Terrain: grid("...", "..."), Units: []tactics.UnitSetup{a, f, boss(2, 1), e}})
		b.SetHP(1, 1)
		events := enemyAttacks(t, b, 3, 1)
		if b.Units()[1].Alive {
			t.Fatalf("仲間が倒れていない: %+v", events)
		}
		assertEqual(t, "Exp", exps(events), []expEvent(nil))
		assertEqual(t, "敵のレベルと経験", [2]int{b.Units()[3].Level, b.Units()[3].Exp}, [2]int{10, 0})
	})
}

func TestIT06EnemyLevel(t *testing.T) {
	// 初めの敵・ターンの増援・出来事の増援のどれも、敵の Level は 1〜20。Exp と Growth は 0。
	where := []struct {
		name string
		at   func(s *tactics.Setup) *tactics.UnitSetup
	}{
		{"初めの敵", func(s *tactics.Setup) *tactics.UnitSetup { return &s.Units[3] }},
		{"ターンの増援", func(s *tactics.Setup) *tactics.UnitSetup { return &s.Reinforcements[0].Unit }},
		{"出来事の増援", func(s *tactics.Setup) *tactics.UnitSetup { return &s.Events[0].Enemies[0] }},
	}
	for _, w := range where {
		for _, tc := range []struct {
			name   string
			modify func(u *tactics.UnitSetup)
			ok     bool
		}{
			{"Level 0 は拒む", func(u *tactics.UnitSetup) { u.Level = 0 }, false},
			{"Level 1 は受け付ける", func(u *tactics.UnitSetup) { u.Level = 1 }, true},
			{"Level 20 は受け付ける", func(u *tactics.UnitSetup) { u.Level = 20 }, true},
			{"Level 21 は拒む", func(u *tactics.UnitSetup) { u.Level = 21 }, false},
			{"Exp 1 は拒む", func(u *tactics.UnitSetup) { u.Level, u.Exp = 5, 1 }, false},
			{"Growth の HP 1 は拒む", func(u *tactics.UnitSetup) { u.Level, u.Growth.HP = 5, 1 }, false},
			{"Growth の魔防 1 は拒む", func(u *tactics.UnitSetup) { u.Level, u.Growth.Res = 5, 1 }, false},
		} {
			t.Run("TC-FR-UNT-04-18 "+w.name+"の "+tc.name, func(t *testing.T) {
				s := eventSetup()
				s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 6, 0)}}
				s.Events = []tactics.MapEventSetup{reinforceEvent("e1", tactics.TriggerEnemyAttacked, "guard", 0, 4)}
				tc.modify(w.at(&s))
				_, _, err := tactics.New(s, 1)
				if tc.ok {
					if err != nil {
						t.Fatalf("New: %v", err)
					}
					return
				}
				if !errors.Is(err, tactics.ErrInvalidSetup) {
					t.Fatalf("err = %v, want ErrInvalidSetup", err)
				}
			})
		}
	}

	t.Run("TC-FR-UNT-04-18 試験の補助関数の敵の既定値は Level 1、自軍は 0", func(t *testing.T) {
		assertEqual(t, "敵・ボス・自軍", [3]int{foe("敵", 0, 0).Level, boss(0, 0).Level, ally("仲間", 0, 0, 0).Level}, [3]int{1, 1, 0})
	})
}

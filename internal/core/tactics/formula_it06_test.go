package tactics

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// DES-IT06 4.5 節 B1〜B7 の値から期待値を定める。
func TestIT06Formula(t *testing.T) {
	a := Unit{Stats: Stats{HP: 22, Str: 7, Skl: 11, Spd: 12}, HP: 22, Weapon: Weapon{Type: Sword, Might: 5, Hit: 90, Weight: 4}}
	d := Unit{Stats: Stats{HP: 100, Str: 5, Spd: 4, Def: 4, Lck: 2}, HP: 100, Weapon: Weapon{Type: Lance, Weight: 5}, Tags: []string{"armor", "mounted"}, Pos: Pos{X: 1}}
	type formulaCase struct {
		name                            string
		a, d                            Unit
		terrain                         Terrain
		damage, hit, crit, speed, count int
		eff                             bool
	}
	cases := []formulaCase{
		{"B1", a, d, Terrain{Def: 1, Avoid: 20}, 6, 69, 3, 12, 2, false},
	}
	b2 := a
	b2.Weapon = Weapon{Type: Sword, Might: 4, Hit: 80, Weight: 7, EffectiveTags: []string{"armor", "mounted"}}
	d2 := d
	d2.Stats.Str = 6
	d2.Stats.Spd = 2
	d2.Stats.Def = 11
	cases = append(cases, formulaCase{"B2", b2, d2, Terrain{}, 7, 83, 3, 12, 2, true})
	b3 := a
	b3.Stats = Stats{Str: 2, Mag: 3, Skl: 5, Spd: 6}
	b3.Weapon = Weapon{Type: Magic, Might: 5, Hit: 85, Weight: 2, DamageType: Magical}
	d3 := d2
	d3.Stats.Res = 1
	cases = append(cases, formulaCase{"B3", b3, d3, Terrain{Def: 2, Avoid: 20}, 5, 71, 0, 6, 2, false})
	b6 := a
	b6.Stats.Mag = 2
	b6.Weapon = Weapon{Type: Sword, Might: 6, Hit: 80, Weight: 8, DamageType: Magical}
	d6 := d
	d6.Weapon.Type = Axe
	d6.Stats.Str = 7
	d6.Weapon.Weight = 7
	cases = append(cases, formulaCase{"B6", b6, d6, Terrain{Def: 1, Avoid: 20}, 8, 89, 3, 11, 2, false})
	b5 := Unit{Stats: Stats{HP: 22, Str: 11, Skl: 5, Spd: 5}, HP: 22, Weapon: Weapon{Type: Axe, Might: 6, Hit: 65, Weight: 14, Strikes: 2}}
	d5 := d
	d5.Stats.Spd = 2
	cases = append(cases, formulaCase{"B5", b5, d5, Terrain{}, 14, 86, 0, 2, 2, false})
	t.Run("TC-FR-BTL-04-07 / TC-FR-BTL-04-09 / TC-FR-BTL-05-05 / TC-FR-ITM-01-03 / TC-FR-BTL-08-25", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := Battle{units: []Unit{tc.a, tc.d}, terrain: [][]Terrain{{{}, tc.terrain}}}
				f := b.forecastSide(0, 1, true, tc.count, 1)
				if f.Damage != tc.damage || f.HitRate != tc.hit || f.CritRate != tc.crit || f.AttackSpeed != tc.speed || f.Effective != tc.eff {
					t.Fatalf("予測=%+v", f)
				}
			})
		}
	})
	t.Run("TC-FR-BTL-04-08", func(t *testing.T) {
		for _, tc := range []struct{ weight, speed int }{{7, 12}, {8, 11}, {99, 0}} {
			a.Weapon.Weight = tc.weight
			if got := attackSpeed(a); got != tc.speed {
				t.Fatalf("重さ %d: %d", tc.weight, got)
			}
		}
	})
	t.Run("TC-FR-UNT-07-09", func(t *testing.T) {
		a.Skill = Skill{Effect: Wrath, Chance: 50}
		a.Stats.Skl = 5
		for _, tc := range []struct{ hp, want int }{{12, 0}, {11, 50}, {10, 50}} {
			a.HP = tc.hp
			if got := critRate(a, d); got != tc.want {
				t.Fatalf("HP %d: %d", tc.hp, got)
			}
		}
		a.Weapon.Crit = 100
		if critRate(a, d) != 100 {
			t.Fatal("上限")
		}
		d.Stats.Lck = 999
		if critRate(a, d) != 0 {
			t.Fatal("下限")
		}
	})
}

func TestIT06OrderAndRolls(t *testing.T) {
	t.Run("TC-FR-BTL-06-08", func(t *testing.T) {
		for _, tc := range []struct {
			speedA, speedD int
			want           []int
		}{{8, 4, []int{0, 0, 1, 0, 0}}, {7, 4, []int{0, 0, 1}}, {4, 8, []int{0, 0, 1, 1}}} {
			b := Battle{units: []Unit{{Stats: Stats{Spd: tc.speedA}, Weapon: Weapon{Strikes: 2}}, {Stats: Stats{Spd: tc.speedD}, Weapon: Weapon{Strikes: 2}}}}
			if got := b.strikeOrder(0, 1); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("順=%v 期待=%v", got, tc.want)
			}
		}
	})
	t.Run("TC-FR-BTL-04-07 乱数と端数", func(t *testing.T) {
		for _, tc := range []struct {
			name              string
			rolls             []int
			hit, crit, pavise bool
			amount            int
		}{{"B4", []int{10, 20, 49, 29}, true, true, true, 13}, {"必殺境界", []int{10, 20, 50, 29}, true, false, true, 4}, {"外れ", []int{99, 99}, false, false, false, 0}} {
			t.Run(tc.name, func(t *testing.T) {
				b, _, err := New(Setup{Terrain: [][]Terrain{{{Cost: 1}, {Cost: 1}}}, Units: []UnitSetup{{Side: Player, Lord: true, Stats: Stats{HP: 22, Str: 11, Skl: 5, Spd: 5}, Weapon: Weapon{Type: Axe, Might: 8, Hit: 70, Weight: 7, MinRange: 1, MaxRange: 1}, Skill: Skill{Effect: Wrath, Chance: 50}}, {Side: Enemy, Boss: true, Pos: Pos{X: 1}, Stats: Stats{HP: 100, Str: 6, Spd: 2, Def: 11, Lck: 2}, Weapon: Weapon{Type: Lance, Weight: 5, MinRange: 1, MaxRange: 1}, Skill: Skill{Effect: Pavise, Chance: 30}}}}, 1)
				if err != nil {
					t.Fatal(err)
				}
				b.units[0].HP = 11
				n := 0
				b.roll = func() int {
					if n >= len(tc.rolls) {
						t.Fatal("余計な乱数")
					}
					v := tc.rolls[n]
					n++
					return v
				}
				ev, _ := b.strike(nil, 0, 1, true)
				e := ev[0]
				if e.Hit != tc.hit || e.Crit != tc.crit || e.Pavise != tc.pavise || e.Amount != tc.amount || n != len(tc.rolls) {
					t.Fatalf("判定=%+v 引いた数=%d", e, n)
				}
			})
		}
	})
}

// 戦闘の入力も定義の読み込みと独立して能力・武器を検証する（DES-UNT-01、DES-BTL-09）。
func TestIT06ValidationAndGrowth(t *testing.T) {
	t.Run("TC-FR-SYS-04-42", func(t *testing.T) {
		base := UnitSetup{Stats: Stats{HP: 1}, Weapon: Weapon{MinRange: 1, MaxRange: 1}}
		for _, change := range []func(*UnitSetup){func(u *UnitSetup) { u.Stats.Lck = -1 }, func(u *UnitSetup) { u.Stats.Mag = -1 }, func(u *UnitSetup) { u.Stats.Res = -1 }, func(u *UnitSetup) { u.Growth.Lck = 101 }, func(u *UnitSetup) { u.Growth.Mag = -1 }, func(u *UnitSetup) { u.Growth.Res = 101 }, func(u *UnitSetup) { u.Weapon.Crit = 101 }, func(u *UnitSetup) { u.Weapon.Weight = -1 }, func(u *UnitSetup) { u.Weapon.Strikes = 3 }, func(u *UnitSetup) { u.Skill = Skill{Effect: Wrath, Multiplier: 3} },
			// REV-069 F-04・F-05: 種類 Magic は魔法の攻撃だけ、分類は armor・mounted だけで重ならない、武器の列も同じ規則。
			func(u *UnitSetup) { u.Weapon.Type = Magic },
			func(u *UnitSetup) { u.Weapon.EffectiveTags = []string{"armor", "armor"} },
			func(u *UnitSetup) { u.Weapon.EffectiveTags = []string{"flying"} },
			func(u *UnitSetup) { u.Tags = []string{"mounted", "mounted"} },
			func(u *UnitSetup) { u.Tags = []string{"flying"} },
			func(u *UnitSetup) { u.Weapons = []Weapon{{Crit: 101, MinRange: 1, MaxRange: 1}} },
			func(u *UnitSetup) { u.Weapons = []Weapon{{Type: Magic, MinRange: 1, MaxRange: 2}} },
			func(u *UnitSetup) { u.Weapons = []Weapon{{Type: Magic + 1, MinRange: 1, MaxRange: 1}} }} {
			u := base
			change(&u)
			if validateUnit(u) == nil {
				t.Fatalf("不正入力を受理: %+v", u)
			}
		}
	})
	t.Run("TC-FR-SYS-04-42 正しい分類・魔法の武器は受理し、New は武器の列の誤りを拒む", func(t *testing.T) {
		ok := UnitSetup{Stats: Stats{HP: 1}, Tags: []string{"armor", "mounted"},
			Weapon:  Weapon{Type: Magic, DamageType: Magical, MinRange: 1, MaxRange: 2, EffectiveTags: []string{"armor", "mounted"}},
			Weapons: []Weapon{{Type: Sword, DamageType: Magical, MinRange: 1, MaxRange: 1}}}
		if err := validateUnit(ok); err != nil {
			t.Fatalf("正しい入力を拒んだ: %v", err)
		}
		terrain := [][]Terrain{{{Cost: 1}, {Cost: 1}}}
		lord := UnitSetup{Side: Player, Lord: true, Stats: Stats{HP: 10}, Weapon: Weapon{MinRange: 1, MaxRange: 1}}
		boss := UnitSetup{Side: Enemy, Boss: true, Pos: Pos{X: 1}, Stats: Stats{HP: 10}, Weapon: Weapon{MinRange: 1, MaxRange: 1}}
		boss.Weapons = []Weapon{{Weight: -1, MinRange: 1, MaxRange: 1}}
		if b, events, err := New(Setup{Terrain: terrain, Units: []UnitSetup{lord, boss}}, 1); !errors.Is(err, ErrInvalidSetup) || b != nil || events != nil || !strings.Contains(err.Error(), "weapons[0]") {
			t.Fatalf("武器の列の誤りを受理: %v", err)
		}
	})
	t.Run("TC-FR-UNT-05-05", func(t *testing.T) {
		vals := []int{0, 10, 20, 30, 40, 50, 60, 70}
		i := 0
		b := Battle{roll: func() int { v := vals[i]; i++; return v }}
		g := b.rollGrowth(Growth{HP: 0, Str: 100, Skl: 20, Spd: 31, Def: 40, Lck: 51, Mag: 50, Res: 71})
		want := Stats{Str: 1, Spd: 1, Lck: 1, Res: 1}
		if g != want || i != 8 {
			t.Fatalf("成長=%+v 数=%d", g, i)
		}
	})
	t.Run("TC-FR-BTL-04-07 命中の境界と必殺0・100・0ダメージ", func(t *testing.T) {
		for _, tc := range []struct {
			rate, crit, r1, r2, r3 int
			hit                    bool
		}{{69, 0, 70, 68, 0, false}, {69, 0, 70, 67, 0, true}, {0, 100, 0, 0, 0, false}, {100, 100, 99, 99, 99, true}} {
			b := Battle{units: []Unit{{Alive: true, Stats: Stats{HP: 1}, HP: 1, Weapon: Weapon{Hit: tc.rate, Crit: tc.crit}}, {Alive: true, Stats: Stats{HP: 1}, HP: 1}}, terrain: [][]Terrain{{{}}}}
			vals := []int{tc.r1, tc.r2, tc.r3}
			n := 0
			b.roll = func() int { v := vals[n]; n++; return v }
			es, _ := b.strike(nil, 0, 1, true)
			wantN := 2
			if tc.hit {
				wantN = 3
			}
			if es[0].Hit != tc.hit || es[0].Crit != (tc.hit && tc.crit == 100) || es[0].Amount != 0 || n != wantN {
				t.Fatalf("判定=%+v 数=%d", es[0], n)
			}
		}
	})
}

func TestIT06Rewind(t *testing.T) {
	t.Run("TC-NFR-06-08 / TC-FR-BTL-10-32", func(t *testing.T) {
		setup := Setup{Terrain: [][]Terrain{{{Cost: 1}, {Cost: 1}, {Cost: 1}}}, Units: []UnitSetup{{Side: Player, Lord: true, Stats: Stats{HP: 100, Str: 1, Skl: 20, Spd: 4}, Level: 1, Exp: 99, Growth: Growth{Lck: 50, Mag: 50, Res: 50}, Weapon: Weapon{Hit: 100, Crit: 40, MinRange: 1, MaxRange: 1, Strikes: 2, EffectiveTags: []string{"armor"}}}, {Side: Enemy, Boss: true, Pos: Pos{X: 1}, Stats: Stats{HP: 100, Def: 1}, Tags: []string{"armor"}, Weapon: Weapon{Hit: 100, MinRange: 1, MaxRange: 1}}, {Side: Player, Pos: Pos{X: 2}, Stats: Stats{HP: 100}, Weapon: Weapon{MinRange: 1, MaxRange: 1}}}}
		b, _, err := New(setup, 17)
		if err != nil {
			t.Fatal(err)
		}
		cmd := Command{Unit: 0, To: Pos{}, Action: Attack, Target: 1}
		before, _ := b.pcg.MarshalBinary()
		if _, err = b.Forecast(cmd); err != nil {
			t.Fatal(err)
		}
		after, _ := b.pcg.MarshalBinary()
		if !reflect.DeepEqual(before, after) {
			t.Fatal("予測で乱数が変化")
		}
		first, err := b.Act(cmd)
		if err != nil {
			t.Fatal(err)
		}
		us := b.Units()
		us[0].Weapon.EffectiveTags[0] = "変更"
		us[1].Tags[0] = "変更"
		first[0].Units[0].Weapon.EffectiveTags[0] = "変更"
		if b.units[0].Weapon.EffectiveTags[0] != "armor" || b.units[1].Tags[0] != "armor" {
			t.Fatal("返却値から本物が変化")
		}
		if err = b.Rewind(b.History()[0].ID); err != nil {
			t.Fatal(err)
		}
		second, err := b.Act(cmd)
		if err != nil {
			t.Fatal(err)
		}
		// 独立性を確かめるために書き換えた写しだけ元に戻して比べる。
		first[0].Units[0].Weapon.EffectiveTags[0] = "armor"
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("巻き戻しで結果が異なる\n%+v\n%+v", first, second)
		}
	})
}

func TestIT06GrowthApplied(t *testing.T) {
	t.Run("TC-FR-UNT-05-05 加算と特効対象外", func(t *testing.T) {
		n := 0
		b := Battle{units: []Unit{{Side: Player, Level: 1, Exp: 99, Stats: Stats{HP: 1}, Growth: Growth{Lck: 100, Mag: 100, Res: 100}}}, roll: func() int { n++; return 99 }}
		b.gainExp(nil, 0, 1)
		s := b.units[0].Stats
		if s.Lck != 1 || s.Mag != 1 || s.Res != 1 || n != 8 {
			t.Fatalf("成長=%+v 乱数=%d", s, n)
		}
		if effective(Weapon{EffectiveTags: []string{"armor"}}, []string{"mounted"}) {
			t.Fatal("特効の対象外")
		}
	})
}

// 勇者の斧（2 回攻撃）と追撃が重なる盤面で、予測と実際の攻撃が一致する（DES-BTL-09、DES-IT06 4.3・4.4 節、REV-069 F-06）。
func TestIT06BraveAxeForecastMatchesAct(t *testing.T) {
	newBrave := func(t *testing.T) *Battle {
		t.Helper()
		// A: 攻撃の速さ 12 − (9 − 8) = 11。D: 5。差 6 で A が追撃し、斧は槍に有利（+1）。
		a := UnitSetup{Name: "A", Side: Player, Lord: true, Stats: Stats{HP: 60, Str: 8, Skl: 10, Spd: 12, Def: 5, Lck: 10, Mov: 1},
			Weapon: Weapon{Name: "勇者の斧", Type: Axe, Might: 9, Hit: 100, Crit: 5, Weight: 9, Strikes: 2, MinRange: 1, MaxRange: 1}}
		d := UnitSetup{Name: "D", Side: Enemy, Boss: true, AI: Hold, Pos: Pos{X: 1}, Stats: Stats{HP: 80, Str: 8, Skl: 4, Spd: 5, Def: 6, Lck: 2},
			Weapon: Weapon{Name: "槍", Type: Lance, Might: 6, Hit: 100, Weight: 5, MinRange: 1, MaxRange: 1}}
		b, _, err := New(Setup{Terrain: [][]Terrain{{{Name: "平地", Cost: 1}, {Name: "平地", Cost: 1}}}, Units: []UnitSetup{a, d}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		// 乱数 50: 命中率 51 以上なら当たり、必殺率 50 以下なら必殺は出ない。
		b.roll = func() int { return 50 }
		return b
	}
	cmd := Command{Unit: 0, To: Pos{}, Action: Attack, Target: 1}
	t.Run("TC-FR-BTL-06-08 / TC-FR-BTL-08-25 勇者の斧と追撃の予測（4 / 1 回）と実際の撃の順 A,A,D,A,A", func(t *testing.T) {
		b := newBrave(t)
		f, err := b.Forecast(cmd)
		if err != nil {
			t.Fatal(err)
		}
		// ダメージ: A は 8 + 9 + 1 − 6 = 12、D は 8 + 6 − 1 − 5 = 8。必殺率: A は 10/2 + 5 − 2 = 8、D は 4/2 − 10 → 0。
		if f.Attacker.Count != 4 || f.Defender.Count != 1 || f.Attacker.Damage != 12 || f.Defender.Damage != 8 ||
			f.Attacker.CritRate != 8 || f.Defender.CritRate != 0 || f.Attacker.Strikes != 2 {
			t.Fatalf("予測 %+v / %+v", f.Attacker, f.Defender)
		}
		// 重さで落ちた速さ（REV-069 F-03）。
		if f.Attacker.AttackSpeed != 11 || f.Attacker.WeightLoss != 1 || f.Defender.AttackSpeed != 5 || f.Defender.WeightLoss != 0 {
			t.Fatalf("速さ %+v / %+v", f.Attacker, f.Defender)
		}
		events, err := b.Act(cmd)
		if err != nil {
			t.Fatal(err)
		}
		var order []int
		var amounts []int
		// 自軍が 1 体なので行動のあと敵軍の番に進む。比べるのは、この攻撃（最初の番の始まりより前）の撃だけ。
		for _, e := range events {
			if e.Kind == EventPhaseStart {
				break
			}
			if e.Kind == EventStrike {
				order = append(order, e.Unit)
				amounts = append(amounts, e.Amount)
				if !e.Hit || e.Crit {
					t.Fatalf("撃 %+v", e)
				}
			}
		}
		if !reflect.DeepEqual(order, []int{0, 0, 1, 0, 0}) || !reflect.DeepEqual(amounts, []int{12, 12, 8, 12, 12}) {
			t.Fatalf("撃の順 %v、ダメージ %v", order, amounts)
		}
	})
	t.Run("TC-FR-BTL-08-25 予測に特効の効く威力を出す", func(t *testing.T) {
		b := newBrave(t)
		b.units[0].Weapon.EffectiveTags = []string{"armor"}
		b.units[1].Tags = []string{"armor"}
		f, err := b.Forecast(cmd)
		if err != nil {
			t.Fatal(err)
		}
		// 威力 9 の 3 倍 = 27。ダメージは 8 + 27 + 1 − 6 = 30。守る側は特効なし。
		if !f.Attacker.Effective || f.Attacker.EffectiveMight != 27 || f.Attacker.Damage != 30 || f.Defender.Effective || f.Defender.EffectiveMight != 0 {
			t.Fatalf("特効 %+v / %+v", f.Attacker, f.Defender)
		}
	})
}

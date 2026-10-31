package tactics

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// 複数の武器・持ち替え・反撃の武器（DES-ITM-02、DES-IT06 4.3・6.1 節）の試験。期待値は設計の決まりから導く。

// 試験用の武器。命中 100 と技 1（必殺率 0）で、乱数 0 なら必ず当たり、必殺にならない。
var (
	testStaff = Weapon{ID: "heal_staff", Name: "癒やしの杖", Type: Staff, Strikes: 1, Heal: 10, MaxUses: 3, MinRange: 1, MaxRange: 1}
	testFire  = Weapon{ID: "fire", Name: "ファイア", Type: Magic, DamageType: Magical, Strikes: 1, Might: 5, Hit: 100, MinRange: 1, MaxRange: 2}
	testBlade = Weapon{ID: "iron_sword", Name: "鉄の剣", Type: Sword, Strikes: 1, Might: 3, Hit: 100, MinRange: 1, MaxRange: 1}
)

func plainTerrain(w, h int) [][]Terrain {
	rows := make([][]Terrain, h)
	for y := range rows {
		rows[y] = make([]Terrain, w)
		for x := range rows[y] {
			rows[y][x] = Terrain{Name: "平地", Cost: 1}
		}
	}
	return rows
}

// alchemistBattle は、杖を先頭に持つ錬金術師（杖・魔法・剣）、HP の減った味方、距離 2 のボスの盤面を作る。
// 錬金術師は動かない（移動 0）。ボスも動かず（移動 0、待ち構える）、槍は射程 1 なので距離 2 には反撃できない。
// 両者の攻撃の速さは 5 で、追撃はない。
func alchemistBattle(t *testing.T) *Battle {
	t.Helper()
	alch := UnitSetup{Name: "錬金術師", Side: Player, Lord: true, Pos: Pos{0, 0},
		Stats:   Stats{HP: 20, Str: 2, Mag: 5, Skl: 1, Spd: 5, Def: 0, Res: 0, Mov: 0},
		Weapons: []Weapon{testStaff, testFire, testBlade}, AllowedWeaponTypes: []WeaponType{Staff, Magic, Sword},
		Items: []ItemStack{{Item: Item{ID: "potion", Name: "回復薬", Heal: 10}, Count: 1}}}
	ally := UnitSetup{Name: "味方", Side: Player, Pos: Pos{0, 1}, Stats: Stats{HP: 20, Skl: 1, Mov: 0}, Weapon: testBlade}
	boss := UnitSetup{Name: "ボス", Side: Enemy, Level: 1, Boss: true, AI: Hold, Pos: Pos{2, 0}, Stats: Stats{HP: 100, Str: 3, Skl: 1, Spd: 5, Mov: 0},
		Weapon: Weapon{Name: "鉄の槍", Type: Lance, Strikes: 1, Might: 4, Hit: 100, MinRange: 1, MaxRange: 1}}
	b, _, err := New(Setup{Terrain: plainTerrain(4, 3), Units: []UnitSetup{alch, ally, boss}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	b.roll = func() int { return 0 }
	b.units[1].HP = 10
	// 敵軍の番は、ボスが動かずに錬金術師を攻撃する（反撃の武器を確かめるため。距離は射程に関係なく 2）。
	b.actEnemyOverride = func(events []Event, unit int) []Event { return b.attack(events, unit, 0, true) }
	return b
}

func ptr(i int) *int { return &i }

// strikesBy は、出来事のうち s が行った攻撃の数を返す。
func strikesBy(events []Event, s int) (n int, amounts []int) {
	for _, e := range events {
		if e.Kind == EventStrike && e.Unit == s {
			n++
			amounts = append(amounts, e.Amount)
		}
	}
	return n, amounts
}

func TestIT06WeaponCommand(t *testing.T) {
	t.Run("TC-FR-BTL-03-18 攻撃の武器と杖を持つユニットは、攻撃と回復の両方を武器を選んで行える", func(t *testing.T) {
		b := alchemistBattle(t)
		// 攻撃: ファイア（添字 1）は距離 2 に届く。回復: 杖（添字 0、今の装備）で隣の味方。
		f, err := b.Forecast(Command{Unit: 0, To: Pos{0, 0}, Action: Attack, Target: 2, Weapon: ptr(1)})
		// 魔力 5 ＋ 威力 5 − 魔防 0 = 10。命中は 技 1 × 2 ＋ 100 − ボスの攻撃の速さ 5 × 2 = 92。槍は距離 2 に届かず反撃なし。
		if err != nil || f.Attacker.Damage != 10 || f.Attacker.HitRate != 92 || f.Attacker.Count != 1 || f.Defender.Count != 0 {
			t.Fatalf("ファイアの予測 %+v、%v", f, err)
		}
		// 杖の回復量は min(杖の回復 10 ＋ 魔力 5, 兵種の上限（0 はなし）, 不足 HP) で求める（DES-IT06 7.1 節）。
		// 味方は HP 10/20 で不足 10 なので 10、HP 2/20 なら不足 18 なので 15。
		if h, err := b.HealForecast(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1}); err != nil || h.Amount != 10 || h.Uses != 3 {
			t.Fatalf("杖の予測 %+v、%v", h, err)
		}
		b.units[1].HP = 2
		if h, err := b.HealForecast(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1, Weapon: ptr(0)}); err != nil || h.Amount != 15 || h.After != 17 {
			t.Fatalf("杖を選んだ予測 %+v、%v", h, err)
		}
		// 兵種の上限 8 があれば 8。
		b.units[0].StaffHealCap = 8
		if h, err := b.HealForecast(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1, Weapon: ptr(0)}); err != nil || h.Amount != 8 {
			t.Fatalf("上限 8 の予測 %+v、%v", h, err)
		}
	})
	t.Run("TC-FR-BTL-03-18 nil・範囲の外・持てない種類・攻撃に使えない武器を順に拒み、状態と乱数を変えない", func(t *testing.T) {
		b := alchemistBattle(t)
		pcgBefore, _ := b.pcg.MarshalBinary()
		rolls := 0
		b.roll = func() int { rolls++; return 0 }
		at := func(w *int) Command { return Command{Unit: 0, To: Pos{0, 0}, Action: Attack, Target: 2, Weapon: w} }
		heal := func(w *int) Command { return Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1, Weapon: w} }
		for _, tc := range []struct {
			name string
			cmd  Command
			want error
		}{
			{"nil は今の装備（杖）で、攻撃に使えない", at(nil), ErrCannotAttack},
			{"杖を選んでも攻撃に使えない", at(ptr(0)), ErrCannotAttack},
			{"添字 3 は列の外", at(ptr(3)), ErrNoWeapon},
			{"添字 −1 は列の外", at(ptr(-1)), ErrNoWeapon},
			{"剣（射程 1）は距離 2 に届かない", at(ptr(2)), ErrOutOfRange},
			{"回復に魔法は使えない", heal(ptr(1)), ErrCannotHeal},
			{"回復の添字 5 は列の外", heal(ptr(5)), ErrNoWeapon},
			{"待機で武器を選べない", Command{Unit: 0, To: Pos{0, 0}, Action: Wait, Weapon: ptr(1)}, ErrInvalidAction},
			{"回復薬で武器を選べない", Command{Unit: 0, To: Pos{0, 0}, Action: UseItem, Item: 0, Weapon: ptr(0)}, ErrInvalidAction},
		} {
			units, history := b.Units(), len(b.History())
			if _, err := b.Act(tc.cmd); !errors.Is(err, tc.want) {
				t.Errorf("%s: Act の誤り %v、期待 %v", tc.name, err, tc.want)
			}
			if tc.cmd.Action == Attack {
				if _, err := b.Forecast(tc.cmd); !errors.Is(err, tc.want) {
					t.Errorf("%s: 予測の誤り %v、期待 %v", tc.name, err, tc.want)
				}
			}
			if tc.cmd.Action == Heal {
				if _, err := b.HealForecast(tc.cmd); !errors.Is(err, tc.want) {
					t.Errorf("%s: 回復の予測の誤り %v、期待 %v", tc.name, err, tc.want)
				}
			}
			if !reflect.DeepEqual(b.Units(), units) || len(b.History()) != history {
				t.Errorf("%s: 失敗で状態が変わった", tc.name)
			}
		}
		// 持てない種類（剣を持てる種類から外す）は、射程より先に ErrWeaponType で拒む。
		b.units[0].AllowedWeaponTypes = []WeaponType{Staff, Magic}
		if _, err := b.Act(at(ptr(2))); !errors.Is(err, ErrWeaponType) {
			t.Errorf("持てない種類: %v", err)
		}
		if _, err := b.Forecast(at(ptr(2))); !errors.Is(err, ErrWeaponType) {
			t.Errorf("持てない種類の予測: %v", err)
		}
		pcgAfter, _ := b.pcg.MarshalBinary()
		if rolls != 0 || !reflect.DeepEqual(pcgBefore, pcgAfter) || b.units[0].Equipped != 0 {
			t.Fatalf("失敗で乱数 %d 個・装備 %d", rolls, b.units[0].Equipped)
		}
	})
}

func TestIT06Equip(t *testing.T) {
	t.Run("TC-FR-ITM-01-04 杖を先頭にした錬金術師は、攻撃するまで反撃せず、攻撃した魔法で反撃し、回復・道具・待機では装備を変えない", func(t *testing.T) {
		b := alchemistBattle(t)
		// 1 ターン目: 杖で回復（装備は変えない、杖の回数だけ減る）→ 味方の待機で敵軍の番。
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1}); err != nil {
			t.Fatal(err)
		}
		if b.units[0].Equipped != 0 || b.units[0].Weapons[0].Uses != 2 {
			t.Fatalf("回復で装備 %d・杖 %d", b.units[0].Equipped, b.units[0].Weapons[0].Uses)
		}
		events, err := b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if n, _ := strikesBy(events, 0); n != 0 {
			t.Fatalf("杖のまま反撃した: %d 回", n)
		}
		// 2 ターン目: ファイアで攻撃 → 装備がファイアに替わり、敵の攻撃にファイア（射程 1〜2、魔力 5 ＋ 5 = 10）で反撃する。
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Attack, Target: 2, Weapon: ptr(1)}); err != nil {
			t.Fatal(err)
		}
		if b.units[0].Equipped != 1 {
			t.Fatalf("攻撃で装備 %d", b.units[0].Equipped)
		}
		events, err = b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if n, amounts := strikesBy(events, 0); n != 1 || amounts[0] != 10 {
			t.Fatalf("ファイアの反撃 %d 回 %v", n, amounts)
		}
		// 3 ターン目: 杖で回復し、回復薬を使い、待機しても、ファイアのまま反撃する。
		b.units[1].HP = 5
		b.units[0].HP = 15 // ボスの攻撃（力 3 ＋ 4 = 7）で倒れず、回復薬も使える。
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1, Weapon: ptr(0)}); err != nil {
			t.Fatal(err)
		}
		if b.units[0].Equipped != 1 || b.units[0].Weapons[0].Uses != 1 {
			t.Fatalf("回復で装備 %d・杖 %d", b.units[0].Equipped, b.units[0].Weapons[0].Uses)
		}
		if events, err = b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait}); err != nil {
			t.Fatal(err)
		}
		if n, _ := strikesBy(events, 0); n != 1 || b.units[0].Equipped != 1 {
			t.Fatalf("回復のあとの反撃 %d 回・装備 %d", n, b.units[0].Equipped)
		}
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: UseItem, Item: 0}); err != nil {
			t.Fatal(err)
		}
		if b.units[0].Equipped != 1 {
			t.Fatalf("回復薬で装備 %d", b.units[0].Equipped)
		}
	})
	t.Run("TC-FR-ITM-01-04 外れた攻撃も「攻撃に使った」に入り、その武器で反撃する", func(t *testing.T) {
		b := alchemistBattle(t)
		b.units[0].Pos = Pos{1, 0}
		b.roll = func() int { return 99 }
		events, err := b.Act(Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2, Weapon: ptr(2)})
		if err != nil {
			t.Fatal(err)
		}
		if n, _ := strikesBy(events, 0); n != 1 || events[0].Hit || b.units[0].Equipped != 2 {
			t.Fatalf("外れ: %d 回、装備 %d", n, b.units[0].Equipped)
		}
		// ボスの攻撃（距離 1）に、剣（力 2 ＋ 威力 3 = 5、相性 剣 → 槍 は不利で −1）で反撃する。
		b.roll = func() int { return 0 }
		events, err = b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if n, amounts := strikesBy(events, 0); n != 1 || amounts[0] != 4 {
			t.Fatalf("剣の反撃 %d 回 %v", n, amounts)
		}
	})
	t.Run("TC-FR-ITM-01-04 敵の思考は手持ちの攻撃の武器から選び、使った武器を装備する。同じ値なら武器の添字の小さいほう", func(t *testing.T) {
		lord := UnitSetup{Name: "主人公", Side: Player, Lord: true, Pos: Pos{0, 0}, Stats: Stats{HP: 50, Mov: 0}, Weapon: testBlade}
		bow := Weapon{Name: "弓", Type: Bow, Strikes: 1, Might: 2, Hit: 100, MinRange: 2, MaxRange: 2}
		other := testBlade
		other.Name = "もう 1 本の剣"
		for _, tc := range []struct {
			name     string
			pos      Pos
			weapons  []Weapon
			allowed  []WeaponType
			equipped int
			want     int
		}{
			{"距離 2 なら、装備の剣ではなく弓で攻撃する", Pos{2, 0}, []Weapon{testBlade, bow}, []WeaponType{Sword, Bow}, 0, 1},
			{"同じ値の 2 本なら、添字 0 を選ぶ", Pos{1, 0}, []Weapon{testBlade, other}, []WeaponType{Sword}, 1, 0},
		} {
			t.Run(tc.name, func(t *testing.T) {
				boss := UnitSetup{Name: "ボス", Side: Enemy, Level: 1, Boss: true, AI: Charge, Pos: tc.pos, Stats: Stats{HP: 50, Str: 1, Skl: 1, Mov: 0},
					Weapons: tc.weapons, AllowedWeaponTypes: tc.allowed, Equipped: tc.equipped}
				b, _, err := New(Setup{Terrain: plainTerrain(3, 1), Units: []UnitSetup{lord, boss}}, 1)
				if err != nil {
					t.Fatal(err)
				}
				b.roll = func() int { return 0 }
				events, err := b.EndPhase()
				if err != nil {
					t.Fatal(err)
				}
				if n, _ := strikesBy(events, 1); n != 1 || b.units[1].Equipped != tc.want {
					t.Fatalf("敵の攻撃 %d 回、装備 %d、期待 %d", n, b.units[1].Equipped, tc.want)
				}
			})
		}
	})
}

func TestIT06HandSlots(t *testing.T) {
	potion := Item{ID: "potion", Name: "回復薬", Heal: 10}
	base := func() Setup {
		lord := UnitSetup{Name: "主人公", Side: Player, Lord: true, Pos: Pos{0, 0}, Stats: Stats{HP: 20, Mov: 1}, Weapon: testBlade}
		boss := UnitSetup{Name: "ボス", Side: Enemy, Level: 1, Boss: true, Pos: Pos{2, 0}, Stats: Stats{HP: 20}, Weapon: testBlade}
		return Setup{Terrain: plainTerrain(3, 1), Units: []UnitSetup{lord, boss}}
	}
	blades := func(n int) []Weapon {
		out := make([]Weapon, n)
		for i := range out {
			out[i] = testBlade
		}
		return out
	}
	t.Run("TC-FR-ITM-01-05 武器の本数と道具の個数の合計で 5 枠まで", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			change  func(u *UnitSetup)
			ok      bool
			message string
		}{
			{"武器 1 本 ＋ 道具 4 個は通る", func(u *UnitSetup) { u.Items = []ItemStack{{Item: potion, Count: 4}} }, true, ""},
			{"武器 1 本 ＋ 道具 5 個は拒む", func(u *UnitSetup) { u.Items = []ItemStack{{Item: potion, Count: 5}} }, false, "手持ちの枠"},
			{"武器 2 本 ＋ 道具 2 個 ＋ 1 個は通る", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes = Weapon{}, blades(2), []WeaponType{Sword}
				u.Items = []ItemStack{{Item: potion, Count: 2}, {Item: potion, Count: 1}}
			}, true, ""},
			{"武器 2 本 ＋ 道具 2 個 ＋ 2 個は拒む", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes = Weapon{}, blades(2), []WeaponType{Sword}
				u.Items = []ItemStack{{Item: potion, Count: 2}, {Item: potion, Count: 2}}
			}, false, "手持ちの枠"},
			{"武器 5 本は通る", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes = Weapon{}, blades(5), []WeaponType{Sword}
			}, true, ""},
			{"武器 6 本は拒む", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes = Weapon{}, blades(6), []WeaponType{Sword}
			}, false, "武器 6 本"},
			{"Weapon と Weapons の両方は拒む", func(u *UnitSetup) { u.Weapons, u.AllowedWeaponTypes = blades(1), []WeaponType{Sword} }, false, "両方"},
			{"武器が 1 本もなければ拒む", func(u *UnitSetup) { u.Weapon = Weapon{} }, false, "武器が 1 本も"},
			{"Weapons で持てる種類を省いたら拒む", func(u *UnitSetup) { u.Weapon, u.Weapons = Weapon{}, blades(1) }, false, "持てる武器の種類がありません"},
			{"持てる種類にない武器は拒む", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes = Weapon{}, blades(1), []WeaponType{Lance}
			}, false, "weapons[0]"},
			{"装備の添字が列の外なら拒む", func(u *UnitSetup) {
				u.Weapon, u.Weapons, u.AllowedWeaponTypes, u.Equipped = Weapon{}, blades(2), []WeaponType{Sword}, 2
			}, false, "装備の添字"},
			{"装備の添字が負なら拒む", func(u *UnitSetup) { u.Equipped = -1 }, false, "装備の添字"},
			{"持てる種類の重なりは拒む", func(u *UnitSetup) { u.AllowedWeaponTypes = []WeaponType{Sword, Sword} }, false, "重なって"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := base()
				tc.change(&s.Units[0])
				b, events, err := New(s, 1)
				if tc.ok {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidSetup) || b != nil || events != nil || !strings.Contains(err.Error(), tc.message) {
					t.Fatalf("誤り %v（%q を含むはず）", err, tc.message)
				}
			})
		}
	})
	t.Run("TC-FR-ITM-01-05 同じ ID の杖は 1 本ずつ回数を持ち、武器には使用回数がない", func(t *testing.T) {
		s := base()
		staff := testStaff
		staff.Uses = 0 // Setup の Uses は見ず、New が満タンにする。
		weak := staff
		weak.Heal = 1 // 2 本目だけ回復 1。回復量は選んだ杖の値で求める。
		s.Units[0].Weapon, s.Units[0].Weapons = Weapon{}, []Weapon{testBlade, staff, weak}
		s.Units[0].AllowedWeaponTypes = []WeaponType{Sword, Staff}
		s.Units = append(s.Units, UnitSetup{Name: "味方", Side: Player, Pos: Pos{1, 0}, Stats: Stats{HP: 20}, Weapon: testBlade})
		b, _, err := New(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		b.units[2].HP = 1
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 2, Weapon: ptr(2)}); err != nil {
			t.Fatal(err)
		}
		w := b.Units()[0].Weapons
		// 選んだ 2 本目（回復 1 ＋ 魔力 0）で 1 回復する。
		if w[1].Uses != 3 || w[2].Uses != 2 || w[0].Uses != 0 || b.units[0].Equipped != 0 || b.units[2].HP != 2 {
			t.Fatalf("杖の回数 %d / %d、剣 %d、装備 %d", w[1].Uses, w[2].Uses, w[0].Uses, b.units[0].Equipped)
		}
	})
}

func TestIT06WeaponCopies(t *testing.T) {
	t.Run("TC-FR-BTL-10-33 装備・武器の列・持てる種類・杖の回数を深く写し、巻き戻しで戻す", func(t *testing.T) {
		b := alchemistBattle(t)
		if _, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1}); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait}); err != nil {
			t.Fatal(err)
		}
		before := b.Units()
		events, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Attack, Target: 2, Weapon: ptr(1)})
		if err != nil {
			t.Fatal(err)
		}
		// 返した写し（Units・出来事・BoardView）を書き換えても本物は変わらない。
		us := b.Units()
		us[0].Weapons[0].Uses = 99
		us[0].AllowedWeaponTypes[0] = Bow
		events[0].Units[0].Weapons[1].EffectiveTags = append(events[0].Units[0].Weapons[1].EffectiveTags, "armor")
		events[0].Units[0].AllowedWeaponTypes[1] = Bow
		v := b.BoardView()
		v.units[0].Weapons[0].Uses = 98
		if b.units[0].Weapons[0].Uses != 2 || b.units[0].AllowedWeaponTypes[0] != Staff || b.units[0].AllowedWeaponTypes[1] != Magic || len(b.units[0].Weapons[1].EffectiveTags) != 0 {
			t.Fatalf("写しから本物が変わった: %+v", b.units[0])
		}
		if b.units[0].Equipped != 1 {
			t.Fatalf("装備 %d", b.units[0].Equipped)
		}
		// 攻撃の前へ巻き戻すと、装備は杖（0）に、杖の回数は 2 に戻る。
		h := b.History()
		if err := b.Rewind(h[len(h)-1].ID); err != nil {
			t.Fatal(err)
		}
		if got := b.Units(); !reflect.DeepEqual(got, before) || got[0].Equipped != 0 || got[0].Weapons[0].Uses != 2 {
			t.Fatalf("巻き戻し %+v", got[0])
		}
	})
}

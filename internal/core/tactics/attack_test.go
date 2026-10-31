package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 設計書 10 章の仮置きの値で作るユニット（計算例 1〜5 に使う）。

var (
	ironSword = tactics.Weapon{Name: "鉄の剣", Type: tactics.Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
	ironLance = tactics.Weapon{Name: "鉄の槍", Type: tactics.Lance, Might: 7, Hit: 80, MinRange: 1, MaxRange: 1}
	ironAxe   = tactics.Weapon{Name: "鉄の斧", Type: tactics.Axe, Might: 8, Hit: 70, MinRange: 1, MaxRange: 1}
	ironBow   = tactics.Weapon{Name: "鉄の弓", Type: tactics.Bow, Might: 6, Hit: 85, MinRange: 2, MaxRange: 2}

	skillInspire = tactics.Skill{Name: "鼓舞", Effect: tactics.Inspire, Range: 2, Hit: 10, Avoid: 10}
	skillPavise  = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 30}
	skillWrath   = tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}
	skillSnipe   = tactics.Skill{Name: "狙撃", Effect: tactics.Snipe, Hit: 20}
)

func chara(name string, side tactics.Side, x, y int, st tactics.Stats, w tactics.Weapon, sk tactics.Skill) tactics.UnitSetup {
	return tactics.UnitSetup{
		Name: name, ClassName: "テスト", Side: side, Stats: st, Pos: pos(x, y), Weapon: w, Skill: sk,
	}
}

// 自軍のキャラ（10.3 節）。レオンが主人公。
func leon(x, y int) tactics.UnitSetup {
	u := chara("レオン", tactics.Player, x, y, tactics.Stats{HP: 20, Str: 6, Skl: 8, Spd: 9, Def: 4, Mov: 5}, ironSword, skillInspire)
	u.Lord = true
	return u
}

func gald(x, y int) tactics.UnitSetup {
	return chara("ガルド", tactics.Player, x, y, tactics.Stats{HP: 24, Str: 6, Skl: 3, Spd: 2, Def: 10, Mov: 4}, ironLance, skillPavise)
}

func bran(x, y int) tactics.UnitSetup {
	return chara("ブラン", tactics.Player, x, y, tactics.Stats{HP: 26, Str: 8, Skl: 4, Spd: 5, Def: 3, Mov: 5}, ironAxe, skillWrath)
}

func mira(x, y int) tactics.UnitSetup {
	return chara("ミラ", tactics.Player, x, y, tactics.Stats{HP: 17, Str: 5, Skl: 9, Spd: 7, Def: 3, Mov: 5}, ironBow, skillSnipe)
}

// 敵のクラス（10.1 節。ボーナスなし）。
func soldier(x, y int) tactics.UnitSetup {
	return chara("ソルジャー", tactics.Enemy, x, y, tactics.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5}, ironLance, tactics.Skill{})
}

func myrmidon(x, y int) tactics.UnitSetup {
	return chara("剣士", tactics.Enemy, x, y, tactics.Stats{HP: 17, Str: 4, Skl: 8, Spd: 9, Def: 2, Mov: 5}, ironSword, tactics.Skill{})
}

func fighter(x, y int) tactics.UnitSetup {
	return chara("ファイター", tactics.Enemy, x, y, tactics.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5}, ironAxe, tactics.Skill{})
}

func archer(x, y int) tactics.UnitSetup {
	return chara("アーチャー", tactics.Enemy, x, y, tactics.Stats{HP: 17, Str: 5, Skl: 7, Spd: 6, Def: 3, Mov: 5}, ironBow, tactics.Skill{})
}

// farBoss は、戦いに関わらないマップの隅のボス。
func farBoss(x, y int) tactics.UnitSetup {
	u := boss(x, y)
	u.Stats.Mov = 0
	return u
}

// reserve は、盤面の隅 (6, 6) で動かない控えの味方。
// まだ行動していない自軍がいれば、自軍の攻撃のあとに敵軍の番へ進まない（敵が動いて乱数を引かない）。
func reserve() tactics.UnitSetup {
	return ally("控え", 6, 6, 0)
}

// arena は 7 × 7 の盤面で、(3, 3) だけ森、(5, 5) だけ砦にする。
//
//	. . . . . . .
//	. . . . . . .
//	. . . . . . .
//	. . . F . . .
//	. . . . . . .
//	. . . . . T .
//	. . . . . . .
func arena(units ...tactics.UnitSetup) tactics.Setup {
	return tactics.Setup{
		Terrain: grid(".......", ".......", ".......", "...F...", ".......", ".....T.", "......."),
		Units:   units,
	}
}

// rolls は、乱数を引く関数を vals を順に返すものに差し替える。
// 引いた数が vals より多ければテストを止める。返す関数で、引き残しがないことを確かめる。
func rolls(t *testing.T, b *tactics.Battle, vals ...int) func() {
	t.Helper()
	rest := append([]int(nil), vals...)
	b.SetRoll(func() int {
		if len(rest) == 0 {
			t.Fatalf("乱数を %d 個より多く引いた", len(vals))
		}
		x := rest[0]
		rest = rest[1:]
		return x
	})
	return func() {
		t.Helper()
		if len(rest) != 0 {
			t.Errorf("乱数の引き残し %v（%d 個のうち）", rest, len(vals))
		}
	}
}

// noRolls は、乱数を引いたらテストを止めるように差し替える。
func noRolls(t *testing.T, b *tactics.Battle) {
	t.Helper()
	b.SetRoll(func() int {
		t.Fatalf("乱数を引いた")
		return 0
	})
}

// strike は Strike の出来事の比べる部分。
type strike struct {
	Unit, Target int
	Hit          bool
	Amount       int
	Crit, Pavise bool
}

// strikes は出来事の列から Strike だけを取り出す。
func strikes(events []tactics.Event) []strike {
	var out []strike
	for _, e := range events {
		if e.Kind == tactics.EventStrike {
			out = append(out, strike{e.Unit, e.Target, e.Hit, e.Amount, e.Crit, e.Pavise})
		}
	}
	return out
}

func mustForecast(t *testing.T, b *tactics.Battle, cmd tactics.Command) tactics.Forecast {
	t.Helper()
	cmd.Action = tactics.Attack
	f, err := b.Forecast(cmd)
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	return f
}

func mustAct(t *testing.T, b *tactics.Battle, cmd tactics.Command) []tactics.Event {
	t.Helper()
	events, err := b.Act(cmd)
	if err != nil {
		t.Fatalf("Act: %v", err)
	}
	return events
}

// sideValues は予測の片方の側の数値（ダメージ、命中率、回数）。
type sideValues struct{ Damage, HitRate, Count int }

func values(s tactics.ForecastSide) sideValues { return sideValues{s.Damage, s.HitRate, s.Count} }

// skillNames は、効くスキルの名前の列を返す。
func skillNames(s tactics.ForecastSide) []string {
	var out []string
	for _, sk := range s.Skills {
		out = append(out, sk.Name)
	}
	return out
}

// example1 は計算例 1: 平地のレオン (2, 3) が、森 (3, 3) のソルジャーを攻撃する。
//
//	0: レオン (2, 3)、1: ソルジャー (3, 3)、2: ボス (6, 0)、3: 控え (6, 6)
func example1() tactics.Setup {
	return arena(leon(2, 3), soldier(3, 3), farBoss(6, 0), reserve())
}

// TestAttackCalculation は攻撃の計算と命中の判定を確かめる（DES-BTL-05、FR-BTL-04）。
func TestAttackCalculation(t *testing.T) {
	t.Run("TC-FR-BTL-04-01 計算例 1・1'（森のソルジャーをレオンが攻撃）", func(t *testing.T) {
		b := mustNew(t, example1())
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Target: 1})
		assertEqual(t, "攻撃する側", values(f.Attacker), sideValues{Damage: 5, HitRate: 63, Count: 2})
		assertEqual(t, "守る側（反撃）", values(f.Defender), sideValues{Damage: 9, HitRate: 85, Count: 1})
	})

	t.Run("TC-FR-BTL-04-02 計算例 5（命中率 63 で (40, 90) は外れ、(70, 50) は当たり）", func(t *testing.T) {
		b := mustNew(t, example1())
		// 1 撃目 (40, 90) → 反撃 (0, 0) → 追撃 (70, 50)。
		done := rolls(t, b, 40, 90, 0, 0, 99, 70, 50, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: false},
			{Unit: 1, Target: 0, Hit: true, Amount: 9},
			{Unit: 0, Target: 1, Hit: true, Amount: 5},
		})
		units := b.Units()
		assertEqual(t, "HP", [2]int{units[0].HP, units[1].HP}, [2]int{20 - 9, 18 - 5})
	})

	t.Run("TC-FR-BTL-04-03 乱数 2 つの和が 命中率 × 2 より小さければ当たる", func(t *testing.T) {
		for _, tc := range []struct {
			r1, r2 int
			hit    bool
		}{{63, 62, true}, {63, 63, false}, {0, 125, true}, {99, 27, false}} {
			b := mustNew(t, example1())
			// 1 撃目だけを見る。反撃と追撃は外す。
			vals := []int{tc.r1, tc.r2}
			if tc.hit {
				vals = append(vals, 99)
			}
			vals = append(vals, 99, 99, 99, 99)
			done := rolls(t, b, vals...)
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1})
			done()
			if got := strikes(events)[0].Hit; got != tc.hit {
				t.Errorf("(%d, %d): Hit = %v, want %v", tc.r1, tc.r2, got, tc.hit)
			}
		}
	})

	t.Run("TC-FR-BTL-04-04 命中率は 0〜100 に収める", func(t *testing.T) {
		// 計算例 2 の反撃（剣士の命中 121 − ブランの回避 10）は 100 に収まる。
		b := mustNew(t, arena(bran(2, 2), myrmidon(3, 2), farBoss(6, 0), leon(0, 6)))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Target: 1})
		assertEqual(t, "反撃の命中率", f.Defender.HitRate, 100)
		// 命中率 100 なら (99, 99) でも当たる。
		done := rolls(t, b, 0, 0, 99, 99, 99, 99, 99, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "反撃は当たる", []bool{strikes(events)[1].Hit, strikes(events)[2].Hit}, []bool{true, true})

		// 技 0・命中 0 の武器で、速さ 9 の相手を狙うと 0 に収まる。(0, 0) でも外れる。
		weak := leon(2, 3)
		weak.Stats.Skl = 0
		weak.Weapon.Hit = 0
		b = mustNew(t, arena(weak, myrmidon(3, 3), farBoss(6, 0), reserve()))
		f = mustForecast(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Target: 1})
		assertEqual(t, "命中率", f.Attacker.HitRate, 0)
		done = rolls(t, b, 0, 0, 99, 99)
		events = mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "1 撃目は外れる", strikes(events)[0].Hit, false)
	})

	t.Run("TC-FR-BTL-04-05 守る側のいるマスの地形の守備と回避が効く", func(t *testing.T) {
		// 同じソルジャーを、平地 (1, 1)・森 (3, 3)・砦 (5, 5) に置いて、レオンが左から攻撃する。
		for _, tc := range []struct {
			name string
			x, y int
			want sideValues
		}{
			{"平地", 1, 1, sideValues{Damage: 6, HitRate: 83, Count: 2}},
			{"森（守備 1、回避 20）", 3, 3, sideValues{Damage: 5, HitRate: 63, Count: 2}},
			{"砦（守備 2、回避 20）", 5, 5, sideValues{Damage: 4, HitRate: 63, Count: 2}},
		} {
			b := mustNew(t, arena(leon(tc.x-1, tc.y), soldier(tc.x, tc.y), farBoss(6, 0)))
			f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(tc.x-1, tc.y), Target: 1})
			assertEqual(t, tc.name, values(f.Attacker), tc.want)
		}
	})

	t.Run("TC-FR-BTL-04-06 乱数は 1 撃目・反撃・追撃の順に 2 つずつ引く", func(t *testing.T) {
		b := mustNew(t, example1())
		// 1 撃目は外れ、反撃は外れ、追撃は当たり。引いた数がちょうど 6 つ。
		done := rolls(t, b, 99, 99, 99, 99, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1},
			{Unit: 1, Target: 0},
			{Unit: 0, Target: 1, Hit: true, Amount: 5},
		})
	})
}

// TestWeaponTriangle は武器の相性を確かめる（DES-BTL-05、FR-BTL-05）。
func TestWeaponTriangle(t *testing.T) {
	// 能力のそろったユニットどうしで、攻撃する側の武器と守る側の武器だけを変える。
	// 威力 5・命中 90 にそろえる。相性なしで攻撃力 5 ＋ 5 = 10、ダメージ 10 − 3 = 7、命中 10 ＋ 90 = 100、回避 10 → 命中率 90。
	weapon := func(wt tactics.WeaponType) tactics.Weapon {
		return tactics.Weapon{Name: "武器", Type: wt, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
	}
	cases := []struct {
		id   string
		a, d tactics.WeaponType
		want sideValues
	}{
		{"TC-FR-BTL-05-01 剣は斧に有利", tactics.Sword, tactics.Axe, sideValues{8, 100, 1}},
		{"TC-FR-BTL-05-01 斧は槍に有利", tactics.Axe, tactics.Lance, sideValues{8, 100, 1}},
		{"TC-FR-BTL-05-01 槍は剣に有利", tactics.Lance, tactics.Sword, sideValues{8, 100, 1}},
		{"TC-FR-BTL-05-01 剣は槍に不利", tactics.Sword, tactics.Lance, sideValues{6, 75, 1}},
		{"TC-FR-BTL-05-01 斧は剣に不利", tactics.Axe, tactics.Sword, sideValues{6, 75, 1}},
		{"TC-FR-BTL-05-01 槍は斧に不利", tactics.Lance, tactics.Axe, sideValues{6, 75, 1}},
		{"TC-FR-BTL-05-02 同じ種類は相性なし", tactics.Sword, tactics.Sword, sideValues{7, 90, 1}},
		{"TC-FR-BTL-05-02 弓は剣に相性なし", tactics.Bow, tactics.Sword, sideValues{7, 90, 1}},
		{"TC-FR-BTL-05-02 剣は弓に相性なし", tactics.Sword, tactics.Bow, sideValues{7, 90, 1}},
		{"TC-FR-BTL-05-02 槍は弓に相性なし", tactics.Lance, tactics.Bow, sideValues{7, 90, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			a := lord(1, 1, 3)
			a.Weapon = weapon(tc.a)
			if tc.a == tactics.Bow {
				a.Weapon.MinRange, a.Weapon.MaxRange = 1, 2
			}
			d := foe("相手", 2, 1)
			d.Weapon = weapon(tc.d)
			b := mustNew(t, arena(a, d, farBoss(6, 0)))
			f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
			assertEqual(t, "攻撃する側", values(f.Attacker), tc.want)
		})
	}

	t.Run("TC-FR-BTL-05-03 計算例 2（ブランが平地の剣士を攻撃。反撃で剣士が追撃）", func(t *testing.T) {
		b := mustNew(t, arena(bran(2, 2), myrmidon(3, 2), farBoss(6, 0), leon(0, 6)))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Target: 1})
		assertEqual(t, "攻撃する側", values(f.Attacker), sideValues{Damage: 13, HitRate: 45, Count: 1})
		// 剣士は斧に有利: 攻撃力 4 ＋ 5 ＋ 1 = 10、ダメージ 10 − 3 = 7。速さの差がちょうど 4 で追撃する。
		assertEqual(t, "守る側", values(f.Defender), sideValues{Damage: 7, HitRate: 100, Count: 2})
		done := rolls(t, b, 0, 0, 99, 0, 0, 99, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: true, Amount: 13},
			{Unit: 1, Target: 0, Hit: true, Amount: 7},
			{Unit: 1, Target: 0, Hit: true, Amount: 7},
		})
	})

	t.Run("TC-FR-BTL-05-04 ダメージは 0 より小さくならず、当たれば Amount 0 の Strike を出す", func(t *testing.T) {
		weak := lord(1, 1, 3)
		weak.Stats.Str = 0
		weak.Weapon = tactics.Weapon{Name: "木の棒", Type: tactics.Sword, Might: 1, Hit: 100, MinRange: 1, MaxRange: 1}
		// 守備 10 の相手（ガルドの能力の敵。スキルなし）。
		g := gald(2, 1)
		g.Side = tactics.Enemy
		g.Skill = tactics.Skill{}
		b := mustNew(t, arena(weak, g, farBoss(6, 0), reserve()))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "ダメージ", f.Attacker.Damage, 0)
		done := rolls(t, b, 0, 0, 99, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "1 撃目", strikes(events)[0], strike{Unit: 0, Target: 1, Hit: true, Amount: 0})
		assertEqual(t, "相手の HP", b.Units()[1].HP, 24)
	})
}

// TestAttackFlow は攻撃の流れを確かめる（DES-BTL-06、FR-BTL-06）。
func TestAttackFlow(t *testing.T) {
	t.Run("TC-FR-BTL-06-01 守る側が生きていて、射程に入っていれば反撃する", func(t *testing.T) {
		// 速さをそろえて追撃をなくす。
		b := mustNew(t, arena(lord(1, 1, 3), foe("敵", 2, 1), farBoss(6, 0), reserve()))
		done := rolls(t, b, 0, 0, 99, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: true, Amount: 7},
			{Unit: 1, Target: 0, Hit: true, Amount: 7},
		})
	})

	t.Run("TC-FR-BTL-06-02 弓は隣のマスに反撃できず、剣は 2 マス先に反撃できない", func(t *testing.T) {
		// レオンが隣のアーチャーを攻撃する。速さの差 3 で追撃もない。
		b := mustNew(t, arena(leon(1, 1), archer(2, 1), farBoss(6, 0), reserve()))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "反撃の回数（弓が隣）", f.Defender.Count, 0)
		done := rolls(t, b, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "Strike の数", len(strikes(events)), 1)

		// ミラ（速さ 7）が 2 マス先の剣士を攻撃する。剣士の速さを 11 にして追撃の条件（差 4）を満たすが、
		// 反撃できないので追撃もしない（乱数はちょうど 2 つ）。
		my := myrmidon(3, 1)
		my.Stats.Spd = 11
		b = mustNew(t, arena(lord(6, 6, 0), mira(1, 1), my, farBoss(6, 0)))
		f = mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Target: 2})
		assertEqual(t, "反撃の回数（剣が 2 マス先）", f.Defender.Count, 0)
		done = rolls(t, b, 99, 99, 99)
		events = mustAct(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "Strike の数", len(strikes(events)), 1)
	})

	t.Run("TC-FR-BTL-06-03 速さの差が 4 以上なら攻撃する側が追撃し、3 なら追撃しない", func(t *testing.T) {
		for _, tc := range []struct {
			spd   int
			order []int
			count int
		}{{9, []int{0, 1, 0}, 2}, {8, []int{0, 1}, 1}} {
			a := lord(1, 1, 3)
			a.Stats.Spd = tc.spd
			b := mustNew(t, arena(a, foe("敵", 2, 1), farBoss(6, 0), reserve()))
			f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
			assertEqual(t, "攻撃の回数", [2]int{f.Attacker.Count, f.Defender.Count}, [2]int{tc.count, 1})
			b.SetRoll(func() int { return 99 })
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
			var order []int
			for _, s := range strikes(events) {
				order = append(order, s.Unit)
			}
			assertEqual(t, "1 撃を行う順", order, tc.order)
		}
	})

	t.Run("TC-FR-BTL-06-04 守る側が速さで 4 以上上回り、反撃できれば、守る側が追撃する", func(t *testing.T) {
		d := foe("敵", 2, 1)
		d.Stats.Spd = 9
		b := mustNew(t, arena(lord(1, 1, 3), d, farBoss(6, 0), reserve()))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "攻撃の回数", [2]int{f.Attacker.Count, f.Defender.Count}, [2]int{1, 2})
		b.SetRoll(func() int { return 99 })
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		var order []int
		for _, s := range strikes(events) {
			order = append(order, s.Unit)
		}
		assertEqual(t, "1 撃を行う順", order, []int{0, 1, 1})
	})

	t.Run("TC-FR-BTL-06-05 1 撃目で倒れたら、反撃と追撃をせず、倒れたユニットをマップから外す", func(t *testing.T) {
		d := foe("敵", 2, 1)
		d.Stats.HP = 7
		b := mustNew(t, arena(lord(1, 1, 3), ally("味方", 3, 1, 3), d, farBoss(6, 0)))
		done := rolls(t, b, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventStrike, tactics.EventFall})
		assertEqual(t, "Fall", events[1].Unit, 2)
		u := b.Units()[2]
		assertEqual(t, "倒れた敵", [2]any{u.HP, u.Alive}, [2]any{0, false})
		assertEqual(t, "Fall の写し", events[1].Units[2].Alive, false)
		// 倒れた敵のいたマスは、味方が通れて止まれる。
		if _, ok := b.Reachable(1)[pos(2, 1)]; !ok {
			t.Errorf("倒れた敵のいたマス (2, 1) に止まれない")
		}
		// 倒れた敵はもう狙えない。
		if _, err := b.Forecast(tactics.Command{Unit: 1, To: pos(3, 1), Action: tactics.Attack, Target: 2}); !errors.Is(err, tactics.ErrNoTarget) {
			t.Errorf("倒れた敵を狙う: err = %v, want ErrNoTarget", err)
		}
	})

	t.Run("TC-FR-BTL-06-06 反撃で攻撃する側が倒れたら、追撃をしない", func(t *testing.T) {
		a := ally("味方", 1, 1, 3)
		a.Stats.HP = 7
		a.Stats.Spd = 9
		b := mustNew(t, arena(lord(5, 1, 3), a, foe("敵", 2, 1), farBoss(6, 0)))
		done := rolls(t, b, 99, 99, 0, 0, 99)
		events := mustAct(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventStrike, tactics.EventStrike, tactics.EventFall})
		assertEqual(t, "Fall", events[2].Unit, 1)
		u := b.Units()[1]
		assertEqual(t, "倒れた味方", [3]any{u.HP, u.Alive, u.Acted}, [3]any{0, false, true})
		// 主人公はまだ行動していないので、自軍の番が続く。
		assertEqual(t, "Phase", b.Phase, tactics.Player)
	})
}

// TestAttackAct は、移動してから攻撃する Act を確かめる（DES-BTL-04、DES-EVT-01、FR-BTL-03）。
func TestAttackAct(t *testing.T) {
	t.Run("TC-FR-BTL-03-04 移動してから攻撃し、行動済みになる", func(t *testing.T) {
		// 主人公が (0, 1) から (1, 1) に移動して、(2, 1) の敵を攻撃する。反撃の距離は移動したあとで測る。
		b := mustNew(t, arena(lord(0, 1, 3), foe("敵", 2, 1), farBoss(6, 0)))
		// 主人公の攻撃はすべて当たり、敵軍の番の敵の攻撃はすべて外れる。
		done := rolls(t, b, 0, 0, 99, 0, 0, 99, 99, 99, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		done()
		// 自軍は 1 体なので、攻撃のあと敵軍の番と次の自軍の番の始めに進む。
		// 敵軍の番では、敵が主人公の隣で y の最も小さい (1, 0) に移動して攻撃する（DES-AI-01 の比べ方 5）。
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventMove, tactics.EventStrike, tactics.EventStrike, tactics.EventPhaseStart,
			tactics.EventMove, tactics.EventStrike, tactics.EventStrike, tactics.EventPhaseStart,
		})
		assertEqual(t, "敵の Move", [3]any{events[4].Unit, events[4].From, events[4].To}, [3]any{1, pos(2, 1), pos(1, 0)})
		assertEqual(t, "Move", [3]any{events[0].Unit, events[0].From, events[0].To}, [3]any{0, pos(0, 1), pos(1, 1)})
		assertEqual(t, "Move の写しは行動済みでない", events[0].Units[0].Acted, false)
		assertEqual(t, "Strike の写しは行動済み", events[1].Units[0].Acted, true)
		assertEqual(t, "Strike の写しの HP", events[1].Units[1].HP, 13)
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: true, Amount: 7},
			{Unit: 1, Target: 0, Hit: true, Amount: 7},
			{Unit: 1, Target: 0},
			{Unit: 0, Target: 1},
		})
		u := b.Units()[0]
		assertEqual(t, "位置と HP", [2]any{u.Pos, u.HP}, [2]any{pos(1, 1), 13})
		assertEqual(t, "Turn", b.Turn, 2)
	})

	t.Run("TC-FR-BTL-03-05 移動しないで攻撃すると Move を出さない。まだ行動していない自軍がいれば番は続く", func(t *testing.T) {
		b := mustNew(t, arena(lord(1, 1, 3), ally("味方", 5, 5, 3), foe("敵", 2, 1), farBoss(6, 0)))
		b.SetRoll(func() int { return 99 })
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 2})
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventStrike, tactics.EventStrike})
		assertEqual(t, "Acted", b.Units()[0].Acted, true)
		assertEqual(t, "Phase", b.Phase, tactics.Player)
		// 行動済みのユニットはもう攻撃できない。
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 2}); !errors.Is(err, tactics.ErrAlreadyActed) {
			t.Errorf("err = %v, want ErrAlreadyActed", err)
		}
	})

	// attackErrSetup は検証 6 の誤りを起こす盤面。
	//
	//	0: 主人公 (1, 1) 移動 2（剣）
	//	1: 味方   (2, 1)
	//	2: 敵     (1, 2)
	//	3: 敵     (5, 1)（倒れた状態にする）
	//	4: ボス   (6, 6)
	//	5: 弓の味方 (4, 4) 移動 2
	attackErrSetup := func() tactics.Setup {
		return arena(lord(1, 1, 2), ally("味方", 2, 1, 2), foe("敵", 1, 2), foe("倒れた敵", 5, 1), farBoss(6, 6), mira(4, 4))
	}
	cases := []struct {
		name string
		cmd  tactics.Command
		want error
	}{
		{"ErrNoTarget（-1）", tactics.Command{Unit: 0, To: pos(1, 1), Target: -1}, tactics.ErrNoTarget},
		{"ErrNoTarget（数を超える）", tactics.Command{Unit: 0, To: pos(1, 1), Target: 6}, tactics.ErrNoTarget},
		{"ErrNoTarget（味方）", tactics.Command{Unit: 0, To: pos(1, 1), Target: 1}, tactics.ErrNoTarget},
		{"ErrNoTarget（自分）", tactics.Command{Unit: 0, To: pos(1, 1), Target: 0}, tactics.ErrNoTarget},
		{"ErrNoTarget（倒れた敵。射程の外でも ErrNoTarget が先）", tactics.Command{Unit: 0, To: pos(1, 1), Target: 3}, tactics.ErrNoTarget},
		{"ErrOutOfRange（移動したあとのマスから遠い）", tactics.Command{Unit: 0, To: pos(1, 0), Target: 2}, tactics.ErrOutOfRange},
		{"ErrOutOfRange（弓で隣のマス）", tactics.Command{Unit: 5, To: pos(2, 2), Target: 2}, tactics.ErrOutOfRange},
		{"ErrOutOfRange（弓で 3 マス先）", tactics.Command{Unit: 5, To: pos(4, 2), Target: 2}, tactics.ErrOutOfRange},
		{"ErrUnreachable が検証 6 より先", tactics.Command{Unit: 0, To: pos(4, 1), Target: -1}, tactics.ErrUnreachable},
	}
	for _, tc := range cases {
		t.Run("TC-FR-BTL-03-06 "+tc.name+"（状態は変わらない）", func(t *testing.T) {
			b := mustNew(t, attackErrSetup())
			b.KillUnit(3)
			noRolls(t, b)
			before := snapshot(b)
			cmd := tc.cmd
			cmd.Action = tactics.Attack
			events, err := b.Act(cmd)
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

// TestForecast は予測を確かめる（DES-BTL-07、FR-BTL-08）。
func TestForecast(t *testing.T) {
	t.Run("TC-FR-BTL-08-01 予測は名前・今の HP・ダメージ・命中率・回数・必殺・効くスキルを返し、状態を変えない", func(t *testing.T) {
		// 計算例 4 の盤面: ガルド (1, 1) が平地のファイター (2, 1) を攻撃する。レオン (1, 3) から距離 2。
		s := arena(leon(1, 3), gald(1, 1), fighter(2, 1), farBoss(6, 0))
		b := mustNew(t, s)
		noRolls(t, b)
		before := snapshot(b)
		f := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Target: 2})
		assertEqual(t, "予測", f, tactics.Forecast{
			Attacker: tactics.ForecastSide{
				Name: "ガルド", HP: 24, Damage: 9, HitRate: 73, Count: 1, CritRate: 1, AttackSpeed: 2, Strikes: 1,
				Skills: []tactics.Skill{skillInspire, skillPavise},
			},
			Defender: tactics.ForecastSide{Name: "ファイター", HP: 24, Damage: 6, HitRate: 79, Count: 1, CritRate: 2, AttackSpeed: 4, Strikes: 1},
		})
		assertEqual(t, "状態", snapshot(b), before)
	})

	t.Run("TC-FR-BTL-08-02 予測は移動したあとのマス（To）で、射程・地形・鼓舞を測る", func(t *testing.T) {
		// 主人公 (0, 0) が森 (3, 3) に移動して、(4, 3) のソルジャーを攻撃する。
		// 森の守備 1・回避 20 が、反撃する側から見た主人公に効く。
		s := arena(lord(0, 3, 4), soldier(4, 3), farBoss(6, 0))
		b := mustNew(t, s)
		before := snapshot(b)
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(3, 3), Target: 1})
		// 反撃: 槍は剣に有利。攻撃力 5 ＋ 7 ＋ 1 = 13、ダメージ 13 − (3 ＋ 1) = 9。
		// 命中 8 ＋ 80 ＋ 15 = 103、回避 10 ＋ 20 = 30 → 73。
		assertEqual(t, "反撃", values(f.Defender), sideValues{Damage: 9, HitRate: 73, Count: 1})
		assertEqual(t, "状態", snapshot(b), before)
	})

	errCases := []struct {
		name    string
		prepare func(b *tactics.Battle)
		cmd     tactics.Command
		want    error
	}{
		{"Wait は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Wait, Target: 1}, tactics.ErrInvalidAction},
		{"UseItem は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.UseItem}, tactics.ErrInvalidAction},
		{"範囲外の Action は ErrInvalidAction", nil, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Action(6), Target: 1}, tactics.ErrInvalidAction},
		{"Attack でないことが戦闘の終わりより先", func(b *tactics.Battle) { b.Result = tactics.Won },
			tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Wait, Target: 1}, tactics.ErrInvalidAction},
		{"ErrBattleOver", func(b *tactics.Battle) { b.Result = tactics.Lost },
			tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1}, tactics.ErrBattleOver},
		{"ErrNotPlayerPhase", func(b *tactics.Battle) { b.Phase = tactics.Enemy },
			tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 1}, tactics.ErrNotPlayerPhase},
		{"ErrInvalidUnit（敵）", nil, tactics.Command{Unit: 1, To: pos(3, 3), Action: tactics.Attack, Target: 0}, tactics.ErrInvalidUnit},
		{"ErrUnreachable", nil, tactics.Command{Unit: 0, To: pos(6, 6), Action: tactics.Attack, Target: 1}, tactics.ErrUnreachable},
		{"ErrNoTarget", nil, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 9}, tactics.ErrNoTarget},
		{"ErrOutOfRange", nil, tactics.Command{Unit: 0, To: pos(1, 3), Action: tactics.Attack, Target: 1}, tactics.ErrOutOfRange},
	}
	for _, tc := range errCases {
		t.Run("TC-FR-BTL-08-03 "+tc.name, func(t *testing.T) {
			b := mustNew(t, example1())
			if tc.prepare != nil {
				tc.prepare(b)
			}
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

	t.Run("TC-FR-BTL-08-03 行動済みのユニットは ErrAlreadyActed", func(t *testing.T) {
		b := mustNew(t, arena(lord(2, 3, 3), ally("味方", 0, 0, 1), soldier(3, 3), farBoss(6, 0)))
		mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Wait})
		if _, err := b.Forecast(tactics.Command{Unit: 0, To: pos(2, 3), Action: tactics.Attack, Target: 2}); !errors.Is(err, tactics.ErrAlreadyActed) {
			t.Errorf("err = %v, want ErrAlreadyActed", err)
		}
	})
}

// TestSkills はスキルを確かめる（DES-SKL-01、FR-UNT-07）。
func TestSkills(t *testing.T) {
	t.Run("TC-FR-UNT-07-01 計算例 4（鼓舞: レオンから 2 マス以内のガルドの命中と回避が +10）", func(t *testing.T) {
		b := mustNew(t, arena(leon(1, 3), gald(1, 1), fighter(2, 1), farBoss(6, 0)))
		f := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Target: 2})
		assertEqual(t, "ガルド", values(f.Attacker), sideValues{Damage: 9, HitRate: 73, Count: 1})
		// ファイターの反撃: 命中 8 ＋ 70 ＋ 15 = 93、ガルドの回避 4 ＋ 10 = 14 → 79。
		assertEqual(t, "ファイター", values(f.Defender), sideValues{Damage: 6, HitRate: 79, Count: 1})
		// 鼓舞がなければ、命中率は 63 と 89。
		b = mustNew(t, arena(leon(6, 3), gald(1, 1), fighter(2, 1), farBoss(6, 0)))
		f = mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Target: 2})
		assertEqual(t, "鼓舞なし", [2]int{f.Attacker.HitRate, f.Defender.HitRate}, [2]int{63, 89})
	})

	t.Run("TC-FR-UNT-07-02 鼓舞の距離は 2 以内で、攻撃する側は移動したあとのマスで測る", func(t *testing.T) {
		// レオン (0, 3)。ガルド (2, 0) が (2, 1)（距離 3）や (1, 2)（距離 2）に移動して、隣の敵を攻撃する。
		b := mustNew(t, arena(leon(0, 3), gald(2, 0), fighter(2, 2), farBoss(6, 0)))
		far := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(2, 1), Target: 2})
		near := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 2), Target: 2})
		assertEqual(t, "距離 3（鼓舞なし）", [2]int{far.Attacker.HitRate, far.Defender.HitRate}, [2]int{63, 89})
		assertEqual(t, "距離 2（鼓舞あり）", [2]int{near.Attacker.HitRate, near.Defender.HitRate}, [2]int{73, 79})
		assertEqual(t, "距離 3 の効くスキル", skillNames(far.Attacker), []string{"大盾"})
		assertEqual(t, "距離 2 の効くスキル", skillNames(near.Attacker), []string{"鼓舞", "大盾"})

		// 持ち主には効かない。相手の陣営にも効かない。
		b = mustNew(t, arena(leon(1, 1), fighter(2, 1), farBoss(6, 0)))
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		// レオンの命中 16 ＋ 90 ＋ 15 = 121、回避 8 → 100。ファイターの命中 8 ＋ 70 − 15 = 63、回避 18 → 45。
		assertEqual(t, "持ち主", [2]int{f.Attacker.HitRate, f.Defender.HitRate}, [2]int{100, 45})
		assertEqual(t, "持ち主の効くスキル", [2]any{f.Attacker.Skills, f.Defender.Skills}, [2]any{[]tactics.Skill(nil), []tactics.Skill(nil)})

		// 守る側の味方の鼓舞は、守る側に効く（敵どうし）。
		insp := soldier(4, 1)
		insp.Skill = skillInspire
		b = mustNew(t, arena(leon(1, 1), fighter(2, 1), insp, farBoss(6, 6)))
		f = mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "守る側に鼓舞", [2]int{f.Attacker.HitRate, f.Defender.HitRate}, [2]int{100, 55})
		assertEqual(t, "守る側の効くスキル", skillNames(f.Defender), []string{"鼓舞"})

		// 倒れた持ち主の鼓舞は効かない。
		b.KillUnit(2)
		f = mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "倒れた持ち主", f.Defender.HitRate, 45)
	})

	t.Run("TC-FR-UNT-07-03 大盾は当たったときに乱数を 1 つ引き、30 より小さければダメージを半分（切り捨て）にする", func(t *testing.T) {
		// ガルドを、平地のファイターが攻撃する形にするため、ガルドを敵にする。
		g := gald(2, 1)
		g.Side = tactics.Enemy
		// 主人公はブランの能力の斧使い（怒りなし）: 斧は槍に有利。攻撃力 8 ＋ 8 ＋ 1 = 17、ダメージ 17 − 10 = 7。
		a := bran(1, 1)
		a.Lord = true
		a.Skill = tactics.Skill{}
		for _, tc := range []struct {
			name  string
			rolls []int
			first strike
		}{
			{"29 で出る（7 → 3）", []int{0, 0, 99, 29, 99, 99}, strike{Unit: 0, Target: 1, Hit: true, Amount: 3, Pavise: true}},
			{"30 で出ない", []int{0, 0, 99, 30, 99, 99}, strike{Unit: 0, Target: 1, Hit: true, Amount: 7}},
			{"外れたら引かない", []int{99, 99, 99, 99}, strike{Unit: 0, Target: 1}},
		} {
			b := mustNew(t, arena(a, g, farBoss(6, 0), reserve()))
			f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
			assertEqual(t, tc.name+": 予測のダメージは半分にしない", f.Attacker.Damage, 7)
			assertEqual(t, tc.name+": 効くスキル", skillNames(f.Defender), []string{"大盾"})
			done := rolls(t, b, tc.rolls...)
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
			done()
			assertEqual(t, tc.name, strikes(events)[0], tc.first)
		}

		// ダメージ 0 で当たっても大盾の乱数を引く（1 撃目で 3 つ）。
		// 力 0・威力 1 の剣の主人公: 剣は槍に不利で、攻撃力 0 ＋ 1 − 1 = 0。ガルドの反撃は外す。
		weak := lord(1, 1, 3)
		weak.Stats.Str = 0
		weak.Weapon = tactics.Weapon{Name: "木の棒", Type: tactics.Sword, Might: 1, Hit: 100, MinRange: 1, MaxRange: 1}
		b := mustNew(t, arena(weak, g, farBoss(6, 0), reserve()))
		done := rolls(t, b, 0, 0, 99, 0, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "ダメージ 0 で当たる", strikes(events)[0], strike{Unit: 0, Target: 1, Hit: true, Amount: 0, Pavise: true})

		// 相手が反撃できなければ、攻撃する側の大盾は効くスキルに入らない（ガルドが隣のアーチャーを攻撃）。
		b = mustNew(t, arena(leon(6, 6), gald(1, 1), archer(2, 1), farBoss(6, 0)))
		f := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 1), Target: 2})
		assertEqual(t, "反撃の回数", f.Defender.Count, 0)
		assertEqual(t, "攻撃する側の効くスキル", skillNames(f.Attacker), []string(nil))
	})

	t.Run("TC-FR-UNT-07-04 計算例 3（怒り: HP 半分以下で必殺率 +50、必殺なら3倍）", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			hp    int
			rolls []int
			want  strike
			crit  bool
		}{
			{"HP 13（半分ちょうど）で当たり: 39 ダメージ", 13, []int{0, 0, 0}, strike{Unit: 0, Target: 1, Hit: true, Amount: 39, Crit: true}, true},
			{"HP 14 では怒りの補正がなく、乱数99では通常", 14, []int{0, 0, 99, 99, 99, 99, 99, 99, 99}, strike{Unit: 0, Target: 1, Hit: true, Amount: 13}, false},
			{"HP 13 でも外れたら必殺にならない", 13, []int{99, 99, 99, 99, 99, 99, 99, 99}, strike{Unit: 0, Target: 1}, true},
		} {
			b := mustNew(t, arena(bran(2, 2), myrmidon(3, 2), leon(0, 6), farBoss(6, 0)))
			b.SetHP(0, tc.hp)
			f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Target: 1})
			assertEqual(t, tc.name+": 予測の必殺", f.Attacker.CritRate, map[bool]int{true: 52, false: 2}[tc.crit])
			assertEqual(t, tc.name+": 予測のダメージ", f.Attacker.Damage, 13)
			wantSkills := []string{"怒り"}
			assertEqual(t, tc.name+": 効くスキル", skillNames(f.Attacker), wantSkills)
			done := rolls(t, b, tc.rolls...)
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Action: tactics.Attack, Target: 1})
			done()
			assertEqual(t, tc.name, strikes(events)[0], tc.want)
		}
	})

	t.Run("TC-FR-UNT-07-05 怒りは 1 撃ごとにそのときの HP で決まり、反撃でも効く", func(t *testing.T) {
		// 剣士がブラン（HP 20）を攻撃する形にするため、ブランを敵、剣士を自軍にする。
		// 剣士の 1 撃目でブランの HP が 13 になり、ブランの反撃が必殺になる。
		// 剣士の命中率は 100 なので、追撃の乱数 (99, 99) でも当たる。
		br := bran(2, 2)
		br.Side = tactics.Enemy
		my := myrmidon(1, 2)
		my.Side = tactics.Player
		my.Lord = true
		my.Stats.HP = 40
		b := mustNew(t, arena(my, br, farBoss(6, 0), reserve()))
		b.SetHP(1, 20)
		f := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 2), Target: 1})
		assertEqual(t, "攻撃の始めは必殺の条件を満たさない", f.Defender.CritRate, 2)
		done := rolls(t, b, 0, 0, 99, 0, 0, 0, 99, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 2), Action: tactics.Attack, Target: 1})
		done()
		// 剣士の攻撃力 4 ＋ 5 ＋ 1 = 10、ダメージ 10 − 3 = 7。ブランの反撃 13 × 3 = 39。
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: true, Amount: 7},
			{Unit: 1, Target: 0, Hit: true, Amount: 39, Crit: true},
			{Unit: 0, Target: 1, Hit: true, Amount: 7},
		})
	})

	t.Run("TC-FR-UNT-07-06 必殺と大盾が両方出たら、3 倍してから半分にする", func(t *testing.T) {
		// 怒りのブラン（HP 13）が、大盾を持つ敵（平地、守備 0、槍）を攻撃する。
		// 攻撃力 8 ＋ 8 ＋ 1 = 17、ダメージ 17。17 × 3 = 51、半分で 25（切り捨て）。
		g := gald(2, 2)
		g.Side = tactics.Enemy
		g.Stats.Def = 0
		g.Stats.HP = 60
		b := mustNew(t, arena(bran(1, 2), g, leon(6, 6), farBoss(6, 0)))
		b.SetHP(0, 13)
		done := rolls(t, b, 0, 0, 0, 0, 99, 99)
		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 2), Action: tactics.Attack, Target: 1})
		done()
		assertEqual(t, "1 撃目", strikes(events)[0], strike{Unit: 0, Target: 1, Hit: true, Amount: 25, Crit: true, Pavise: true})
		assertEqual(t, "HP", b.Units()[1].HP, 35)
	})

	t.Run("TC-FR-UNT-07-07 狙撃は移動せずに自分から攻撃したときだけ命中 +20（追撃を含み、反撃には効かない）", func(t *testing.T) {
		// ミラ (1, 3) が、森 (3, 3) の剣士を 2 マス先から攻撃する。
		// 命中 18 ＋ 85 ＝ 103、回避 18 ＋ 20 = 38 → 65。狙撃で 85。
		b := mustNew(t, arena(leon(0, 0), mira(1, 3), myrmidon(3, 3), farBoss(6, 0)))
		still := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 3), Target: 2})
		moved := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(3, 1), Target: 2})
		assertEqual(t, "移動しない", still.Attacker.HitRate, 85)
		assertEqual(t, "移動しないときの効くスキル", skillNames(still.Attacker), []string{"狙撃"})
		// (3, 1) は平地で、剣士は森のまま。
		assertEqual(t, "移動する", moved.Attacker.HitRate, 65)
		assertEqual(t, "移動したときの効くスキル", skillNames(moved.Attacker), []string(nil))

		// 追撃にも効く: ミラの速さを 13 にして追撃させ、命中率 85 の境目 (85, 84) で当たり、(85, 85) で外れる。
		m := mira(1, 3)
		m.Stats.Spd = 13
		b = mustNew(t, arena(leon(0, 0), m, myrmidon(3, 3), farBoss(6, 0)))
		done := rolls(t, b, 85, 84, 99, 85, 84, 99)
		events := mustAct(t, b, tactics.Command{Unit: 1, To: pos(1, 3), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "1 撃目と追撃", []bool{strikes(events)[0].Hit, strikes(events)[1].Hit}, []bool{true, true})

		// 反撃には効かない: 敵の弓兵（狙撃）を 2 マス先からミラが攻撃すると、反撃の命中率に狙撃は入らない。
		sn := archer(3, 3)
		sn.Skill = skillSnipe
		b = mustNew(t, arena(leon(0, 0), mira(1, 3), sn, farBoss(6, 0)))
		f := mustForecast(t, b, tactics.Command{Unit: 1, To: pos(1, 3), Target: 2})
		// 弓兵の命中 14 ＋ 85 = 99、ミラの回避 14 → 85。
		assertEqual(t, "反撃の命中率", f.Defender.HitRate, 85)
		assertEqual(t, "反撃の効くスキル", skillNames(f.Defender), []string(nil))
		// Act でも反撃に効かない: ミラの 1 撃目を当て（森の弓兵に 7 ダメージ）、反撃の乱数を (85, 85) にする。
		// 狙撃が効かない命中率 85 なら和 170 で外れ、効けば命中率 100 で当たる。
		done = rolls(t, b, 0, 0, 99, 85, 85)
		events = mustAct(t, b, tactics.Command{Unit: 1, To: pos(1, 3), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "Act の反撃", strikes(events), []strike{
			{Unit: 1, Target: 2, Hit: true, Amount: 7},
			{Unit: 2, Target: 1},
		})

		// Act でも移動すると効かない: ミラが (3, 1) に移動して森の剣士を攻撃し、乱数を (65, 65) にする。
		// 狙撃が効かない命中率 65 なら和 130 で外れ、効けば命中率 85 で当たる。剣士は 2 マス先に反撃できない。
		b = mustNew(t, arena(leon(0, 0), mira(1, 3), myrmidon(3, 3), farBoss(6, 0)))
		done = rolls(t, b, 65, 65)
		events = mustAct(t, b, tactics.Command{Unit: 1, To: pos(3, 1), Action: tactics.Attack, Target: 2})
		done()
		assertEqual(t, "Act で移動して攻撃", strikes(events), []strike{{Unit: 1, Target: 2}})
	})
}

// TestWeapon は武器の装備と種類を確かめる（DES-BTL-05、FR-ITM-01）。
func TestWeapon(t *testing.T) {
	t.Run("TC-FR-ITM-01-01 装備した武器の威力・命中が攻撃に効く", func(t *testing.T) {
		a := lord(1, 1, 3)
		b := mustNew(t, arena(a, foe("敵", 2, 1), farBoss(6, 0)))
		base := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		a.Weapon.Might += 3
		a.Weapon.Hit -= 20
		b = mustNew(t, arena(a, foe("敵", 2, 1), farBoss(6, 0)))
		strong := mustForecast(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Target: 1})
		assertEqual(t, "威力 +3、命中 −20", [2]int{strong.Attacker.Damage - base.Attacker.Damage, strong.Attacker.HitRate - base.Attacker.HitRate}, [2]int{3, -20})
	})

	t.Run("TC-FR-ITM-01-02 武器の射程の最小・最大の距離で攻撃でき、その外では攻撃できない", func(t *testing.T) {
		a := lord(0, 3, 6)
		a.Weapon = tactics.Weapon{Name: "長弓", Type: tactics.Bow, Might: 5, Hit: 90, MinRange: 2, MaxRange: 3}
		b := mustNew(t, arena(a, soldier(5, 3), farBoss(6, 0)))
		for _, tc := range []struct {
			x    int
			want error
		}{{1, tactics.ErrOutOfRange}, {2, nil}, {3, nil}, {4, tactics.ErrOutOfRange}} {
			_, err := b.Forecast(tactics.Command{Unit: 0, To: pos(tc.x, 3), Action: tactics.Attack, Target: 1})
			if !errors.Is(err, tc.want) {
				t.Errorf("距離 %d: err = %v, want %v", 5-tc.x, err, tc.want)
			}
		}
	})
}

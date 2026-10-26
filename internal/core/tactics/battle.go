package tactics

import (
	"fmt"
	"math/rand/v2"
)

// Battle は戦闘の状態（DES-BTL-01）。
//
// Turn、Phase、Result は読むためのもの。状態は Act・EndPhase・Rewind・Resume で変える。
type Battle struct {
	Turn   int
	Phase  Side
	Result Result

	terrain        [][]Terrain
	units          []Unit
	reinforcements []ReinforcementSetup

	// rng は戦闘が持つシード付きの乱数（DES-RNG-01）。
	rng *rand.Rand
	// pcg は保存・復元する実際の乱数源（DES-RWD-03）。
	pcg *rand.PCG
	// 履歴の写し・続き・ID と通し出来事の境界（DES-RWD-01）。
	progress      phaseProgress
	history       []historyRecord
	nextHistoryID uint64
	eventCount    int
	rewindsLeft   int
	// roll は 0〜99 の乱数を 1 つ引く。テストでは export_test.go から差し替える（DES-RNG-01）。
	roll func() int
	// actEnemyOverride は、敵 1 体を動かす処理の差し替え。nil なら actEnemy を使う。
	// テストだけが export_test.go から設定する。
	actEnemyOverride func(events []Event, unit int) []Event
}

// New は Setup を検証し、1 ターン目の自軍の番を始めた戦闘を作る（DES-BTL-01、DES-BTL-02、DES-RNG-01）。
// 返す出来事は、自軍の番の始めの出来事。Setup が正しくなければ ErrInvalidSetup を包んだ誤りを返す。
func New(setup Setup, seed uint64) (*Battle, []Event, error) {
	view, err := NewBoardView(setup)
	if err != nil {
		return nil, nil, err
	}
	b := &Battle{
		Turn:          1,
		Phase:         Player,
		Result:        Ongoing,
		pcg:           rand.NewPCG(seed, 0),
		nextHistoryID: 1,
		rewindsLeft:   10,
	}
	b.rng = rand.New(b.pcg)
	b.roll = func() int { return b.rng.IntN(100) }

	b.terrain, b.units = view.terrain, view.units
	for _, r := range setup.Reinforcements {
		r.Unit.Items = append([]ItemStack(nil), r.Unit.Items...)
		r.Unit.Tags = append([]string(nil), r.Unit.Tags...)
		r.Unit.Weapon = r.Unit.Weapon.clone()
		r.Unit.Weapons = cloneWeapons(r.Unit.Weapons)
		b.reinforcements = append(b.reinforcements, r)
	}
	return b, b.startPlayerPhase(nil), nil
}

// newUnit は UnitSetup から、HP が最大の生きているユニットを作る。
// 杖の使用回数は満タン（Uses = MaxUses）にする。Setup の Uses は見ない（DES-HEL-01、E-33）。
func newUnit(us UnitSetup) Unit {
	u := Unit{
		Name:      us.Name,
		ClassName: us.ClassName,
		Tags:      us.Tags, Weapons: us.Weapons,
		Side:   us.Side,
		Stats:  us.Stats,
		Level:  us.Level,
		Exp:    us.Exp,
		Growth: us.Growth,
		HP:     us.Stats.HP,
		Pos:    us.Pos,
		Weapon: us.Weapon,
		Items:  us.Items,
		Skill:  us.Skill,
		Lord:   us.Lord,
		Boss:   us.Boss,
		AI:     us.AI,
		Alive:  true,
	}
	if u.Weapon.Type == Staff {
		u.Weapon.Uses = u.Weapon.MaxUses
	}
	return u.clone()
}

// validateSetup は Setup の規則 1〜8 を確かめる（DES-BTL-01）。最初に見つかった誤りを返す。
func validateSetup(s Setup) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidSetup, fmt.Sprintf(format, args...))
	}

	// 規則 1: 地形の行が 1 行以上あり、すべて同じ長さ（1 以上）。
	if len(s.Terrain) == 0 {
		return bad("地形の行がありません")
	}
	width := len(s.Terrain[0])
	if width == 0 {
		return bad("地形の行 0 が空です")
	}
	for y, row := range s.Terrain {
		if len(row) != width {
			return bad("地形の行 %d の長さが %d です（行 0 は %d）", y, len(row), width)
		}
	}
	height := len(s.Terrain)
	passable := func(p Pos) bool {
		return p.X >= 0 && p.X < width && p.Y >= 0 && p.Y < height && s.Terrain[p.Y][p.X].Cost >= 1
	}

	lords, bosses := 0, 0
	occupied := map[Pos]int{}
	for i, u := range s.Units {
		where := fmt.Sprintf("units[%d]", i)
		// 規則 2: 座標がマップの中にあり、Cost が 1 以上の地形にある。
		if !passable(u.Pos) {
			return bad("%s: 座標 (%d, %d) がマップの外か、通れない地形です", where, u.Pos.X, u.Pos.Y)
		}
		// 規則 3: 初めのユニットどうしが重ならない。
		if j, ok := occupied[u.Pos]; ok {
			return bad("%s: 座標 (%d, %d) に units[%d] がいます", where, u.Pos.X, u.Pos.Y, j)
		}
		occupied[u.Pos] = i
		if err := validateUnit(u); err != nil {
			return bad("%s: %v", where, err)
		}
		// 規則 5: 主人公は自軍、ボスは敵軍。
		if u.Lord {
			lords++
			if u.Side != Player {
				return bad("%s: 主人公が自軍ではありません", where)
			}
		}
		if u.Boss {
			bosses++
			if u.Side != Enemy {
				return bad("%s: ボスが敵軍ではありません", where)
			}
		}
	}
	if lords != 1 {
		return bad("主人公が %d 体います（ちょうど 1 体）", lords)
	}
	if bosses != 1 {
		return bad("ボスが %d 体います（ちょうど 1 体）", bosses)
	}

	for i, r := range s.Reinforcements {
		where := fmt.Sprintf("reinforcements[%d]", i)
		u := r.Unit
		// 規則 2
		if !passable(u.Pos) {
			return bad("%s: 座標 (%d, %d) がマップの外か、通れない地形です", where, u.Pos.X, u.Pos.Y)
		}
		// 規則 6: 増援のターンは 2 以上。
		if r.Turn < 2 {
			return bad("%s: ターン %d は 2 以上でなければなりません", where, r.Turn)
		}
		// 規則 5: 増援はボスと主人公になれず、敵軍である。
		if u.Lord || u.Boss {
			return bad("%s: 増援は主人公とボスになれません", where)
		}
		if u.Side != Enemy {
			return bad("%s: 増援が敵軍ではありません", where)
		}
		if err := validateUnit(u); err != nil {
			return bad("%s: %v", where, err)
		}
	}
	return nil
}

// validateUnit はユニット 1 体の規則 4・6・7・8 を確かめる（DES-BTL-01、DES-GRW-03、DES-HEL-01）。
func validateUnit(u UnitSetup) error {
	// 規則 4: 能力。
	st := u.Stats
	if st.HP < 1 {
		return fmt.Errorf("最大 HP %d は 1 以上でなければなりません", st.HP)
	}
	if st.Str < 0 || st.Skl < 0 || st.Spd < 0 || st.Def < 0 || st.Mov < 0 || st.Lck < 0 || st.Mag < 0 || st.Res < 0 {
		return fmt.Errorf("能力 %+v に負の値があります", st)
	}
	// 規則 4: 武器の射程は 1 ≦ 最小 ≦ 最大。
	if w := u.Weapon; w.MinRange < 1 || w.MinRange > w.MaxRange {
		return fmt.Errorf("武器の射程 %d〜%d は 1 ≦ 最小 ≦ 最大でなければなりません", w.MinRange, w.MaxRange)
	}
	// 規則 4: アイテムの数と回復量は 1 以上。
	for k, it := range u.Items {
		if it.Count < 1 {
			return fmt.Errorf("items[%d]: 数 %d は 1 以上でなければなりません", k, it.Count)
		}
		if it.Item.Heal < 1 {
			return fmt.Errorf("items[%d]: 回復量 %d は 1 以上でなければなりません", k, it.Item.Heal)
		}
	}
	// 規則 6・8: 定数の型の値。武器の種類は Sword〜Staff のどれか。
	if u.Weapon.Type < Sword || u.Weapon.Type > Magic {
		return fmt.Errorf("武器の種類 %d が正しくありません", u.Weapon.Type)
	}
	if err := validateWeapon(u.Weapon); err != nil {
		return err
	}
	if err := validateStaff(u); err != nil {
		return err
	}
	if u.Skill.Effect < NoSkill || u.Skill.Effect > Snipe {
		return fmt.Errorf("スキルの効果 %d が正しくありません", u.Skill.Effect)
	}
	if u.AI < Charge || u.AI > Hold {
		return fmt.Errorf("動き方 %d が正しくありません", u.AI)
	}
	// 規則 6: スキルの数値（DES-DAT-03 の規則 6 と同じ範囲）。
	sk := u.Skill
	if sk.Range < 0 || sk.Hit < 0 || sk.Avoid < 0 {
		return fmt.Errorf("スキルの数値（範囲 %d、命中 %d、回避 %d）は 0 以上でなければなりません", sk.Range, sk.Hit, sk.Avoid)
	}
	if sk.Chance < 0 || sk.Chance > 100 {
		return fmt.Errorf("スキルの確率 %d は 0〜100 でなければなりません", sk.Chance)
	}
	if sk.Multiplier < 0 || (sk.Effect == Wrath && sk.Multiplier != 0) {
		return fmt.Errorf("スキルの倍率 %d が正しくありません（0 以上、怒りでは 0）", sk.Multiplier)
	}
	return validateGrowth(u)
}

// validateStaff はユニット 1 体の規則 8（杖）を確かめる（DES-HEL-01）。
// 杖は Heal と MaxUses が 1 以上（Uses は New で MaxUses にするので見ない）。杖でない武器は Heal・Uses・MaxUses が 0。
// 敵（増援を含む）の武器は杖でない。
func validateStaff(u UnitSetup) error {
	w := u.Weapon
	if w.Type != Staff {
		if w.Heal != 0 || w.Uses != 0 || w.MaxUses != 0 {
			return fmt.Errorf("杖でない武器の回復量 %d・使用回数 %d・最大 %d は 0 でなければなりません", w.Heal, w.Uses, w.MaxUses)
		}
		return nil
	}
	if u.Side == Enemy {
		return fmt.Errorf("敵の武器は杖にできません")
	}
	if w.Heal < 1 {
		return fmt.Errorf("杖の回復量 %d は 1 以上でなければなりません", w.Heal)
	}
	if w.MaxUses < 1 {
		return fmt.Errorf("杖の使用回数 %d は 1 以上でなければなりません", w.MaxUses)
	}
	return nil
}

// validateGrowth はユニット 1 体の規則 7（レベル・経験・成長率）を確かめる（DES-GRW-03）。
func validateGrowth(u UnitSetup) error {
	if u.Level < 0 || u.Level > maxLevel {
		return fmt.Errorf("レベル %d は 0〜%d でなければなりません", u.Level, maxLevel)
	}
	if u.Side == Enemy && u.Level != 0 {
		return fmt.Errorf("敵のレベル %d は 0 でなければなりません", u.Level)
	}
	if u.Exp < 0 || u.Exp >= expPerLevel {
		return fmt.Errorf("経験 %d は 0〜%d でなければなりません", u.Exp, expPerLevel-1)
	}
	if (u.Level == 0 || u.Level == maxLevel) && u.Exp != 0 {
		return fmt.Errorf("レベル %d の経験 %d は 0 でなければなりません", u.Level, u.Exp)
	}
	g := u.Growth
	for _, r := range []int{g.HP, g.Str, g.Skl, g.Spd, g.Def, g.Lck, g.Mag, g.Res} {
		if r < 0 || r > 100 {
			return fmt.Errorf("成長率 %+v は 0〜100 でなければなりません", g)
		}
	}
	return nil
}

// Width はマップの幅（DES-BTL-01）。
func (b *Battle) Width() int { return len(b.terrain[0]) }

// Height はマップの高さ（DES-BTL-01）。
func (b *Battle) Height() int { return len(b.terrain) }

// inMap は p がマップの中にあるかを返す。
func (b *Battle) inMap(p Pos) bool {
	return p.X >= 0 && p.X < b.Width() && p.Y >= 0 && p.Y < b.Height()
}

// TerrainAt は p の地形を返す（DES-BTL-01、DES-MAP-01）。マップの外ならゼロ値（通れない）を返す。
func (b *Battle) TerrainAt(p Pos) Terrain {
	if !b.inMap(p) {
		return Terrain{}
	}
	return b.terrain[p.Y][p.X]
}

// Units はユニットの写しを返す（DES-BTL-01）。添字はユニットを指す番号で、倒れても変わらない。
func (b *Battle) Units() []Unit {
	return cloneUnits(b.units)
}

// unitAt は p にいる生きているユニットの添字を返す。いなければ -1。
func (b *Battle) unitAt(p Pos) int {
	for i, u := range b.units {
		if u.Alive && u.Pos == p {
			return i
		}
	}
	return -1
}

// validUnit は i が存在するユニットの添字かを返す。
func (b *Battle) validUnit(i int) bool {
	return i >= 0 && i < len(b.units)
}

// validateWeapon は攻撃の追加欄を検証する。省略の回数 0 は 1 に補う（DES-BTL-09）。
func validateWeapon(w Weapon) error {
	if w.Crit < 0 || w.Crit > 100 || w.Weight < 0 || w.Might < 0 || w.Hit < 0 || w.Strikes < 0 || w.Strikes > 2 || w.DamageType < Physical || w.DamageType > Magical {
		return fmt.Errorf("武器の必殺・重さ・回数・区分が正しくありません")
	}
	if w.Type == Staff && (w.Crit != 0 || w.Strikes > 1 || len(w.EffectiveTags) > 0) {
		return fmt.Errorf("杖には必殺・特効・2 回攻撃を付けられません")
	}
	return nil
}

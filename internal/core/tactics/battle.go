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
	// mapEvents は出来事の定義（New が入力から独立させ、戦闘中に変えない）。mapState はその状態（DES-MAP-07）。
	mapEvents    []MapEventSetup
	mapState     eventState
	groupEntered map[string]bool
	groupPlan    map[int]Pos
	groupDanger  map[int]map[Pos]bool
	// canto は待っている再移動（DES-BTL-10）。
	canto cantoState

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
		r.Unit = r.Unit.clone()
		b.reinforcements = append(b.reinforcements, r)
	}
	b.mapEvents = cloneMapEvents(setup.Events)
	b.mapState = eventState{fired: make([]bool, len(b.mapEvents))}
	b.groupEntered = make(map[string]bool)
	return b, b.startPlayerPhase(nil), nil
}

// newUnit は UnitSetup から、HP が最大の生きているユニットを作る。
// 武器 1 本の古い入力は武器の列に直す（DES-ITM-02）。呼ぶ前に validateUnit で確かめておく。
// 杖の使用回数は、持っている杖をすべて満タン（Uses = MaxUses）にする。Setup の Uses は見ない（DES-HEL-01、E-33、DES-IT06 6.2 節）。
func newUnit(us UnitSetup) Unit {
	us, _ = normalizeUnit(us)
	u := Unit{
		Name:               us.Name,
		ClassName:          us.ClassName,
		StaffHealCap:       us.StaffHealCap,
		Tags:               us.Tags,
		Weapons:            us.Weapons,
		Equipped:           us.Equipped,
		AllowedWeaponTypes: us.AllowedWeaponTypes,
		Side:               us.Side,
		Stats:              us.Stats,
		Level:              us.Level,
		Exp:                us.Exp,
		Growth:             us.Growth,
		HP:                 us.Stats.HP,
		Pos:                us.Pos,
		Items:              us.Items,
		Skill:              us.Skill,
		ClassSkills:        us.ClassSkills,
		Lord:               us.Lord,
		Boss:               us.Boss,
		AI:                 us.AI,
		Approach:           us.Approach,
		Alive:              true,
		ID:                 us.ID,
		Group:              us.Group,
	}
	u = u.clone()
	for k := range u.Weapons {
		if u.Weapons[k].Type == Staff {
			u.Weapons[k].Uses = u.Weapons[k].MaxUses
		}
	}
	return u
}

// maxHand は手持ちの枠の数。武器の本数と道具の個数の合計がこれ以下（DES-ITM-02）。
const maxHand = 5

// weaponGiven は、武器 1 本の古い入力の欄 Weapon が指定されているか（ゼロ値でないか）を返す（DES-ITM-02）。
func weaponGiven(w Weapon) bool {
	return w.ID != "" || w.Name != "" || w.Type != Sword || w.Crit != 0 || w.Weight != 0 || w.Strikes != 0 || w.DamageType != Physical ||
		len(w.EffectiveTags) > 0 || w.Might != 0 || w.Hit != 0 || w.MinRange != 0 || w.MaxRange != 0 || w.Heal != 0 || w.Uses != 0 || w.MaxUses != 0
}

// normalizeUnit は武器 1 本の古い入力を武器の列に直した写しを返す（DES-ITM-02、DES-IT06 6.1 節）。
// Weapon と Weapons の両方を指定した入力と、武器が 1 本もない入力は誤りにする。
// Weapon で渡して持てる種類を省いたときだけ、その 1 本の種類を持てる種類にする。
func normalizeUnit(us UnitSetup) (UnitSetup, error) {
	us = us.clone()
	given := weaponGiven(us.Weapon)
	switch {
	case given && len(us.Weapons) > 0:
		return us, fmt.Errorf("Weapon と Weapons の両方は指定できません")
	case given:
		us.Weapons = []Weapon{us.Weapon}
		if len(us.AllowedWeaponTypes) == 0 {
			us.AllowedWeaponTypes = []WeaponType{us.Weapon.Type}
		}
	case len(us.Weapons) == 0:
		return us, fmt.Errorf("武器が 1 本もありません")
	}
	us.Weapon = Weapon{}
	return us, nil
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
	// 規則 9: 初めの敵は生きている敵の上限まで（DES-IT06 9.3 節）。
	if err := validateEnemyCap(s); err != nil {
		return err
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
	// 規則 10: 出来事の定義（DES-MAP-07）。
	return validateMapEvents(s, passable, width, height)
}

// validateUnit はユニット 1 体の規則 4・6・7・8 を確かめる（DES-BTL-01、DES-GRW-03、DES-HEL-01）。
func validateUnit(u UnitSetup) error {
	// 武器 1 本の古い入力の誤りは、今までどおり位置を付けずに返す。
	legacy := len(u.Weapons) == 0
	u, err := normalizeUnit(u)
	if err != nil {
		return err
	}
	// 規則 4: 能力。
	st := u.Stats
	if st.HP < 1 {
		return fmt.Errorf("最大 HP %d は 1 以上でなければなりません", st.HP)
	}
	if st.Str < 0 || st.Skl < 0 || st.Spd < 0 || st.Def < 0 || st.Mov < 0 || st.Lck < 0 || st.Mag < 0 || st.Res < 0 {
		return fmt.Errorf("能力 %+v に負の値があります", st)
	}
	if len(u.Weapons) > maxHand {
		return fmt.Errorf("武器 %d 本は手持ちの枠 %d を超えます", len(u.Weapons), maxHand)
	}
	// 規則 4: アイテムの数と回復量は 1 以上。武器の本数と道具の個数の合計は 5 枠まで（DES-ITM-02）。
	slots := len(u.Weapons)
	for k, it := range u.Items {
		if it.Count < 1 {
			return fmt.Errorf("items[%d]: 数 %d は 1 以上でなければなりません", k, it.Count)
		}
		if it.Item.Heal < 1 {
			return fmt.Errorf("items[%d]: 回復量 %d は 1 以上でなければなりません", k, it.Item.Heal)
		}
		// 減らして比べ、大きな個数でも足し算があふれないようにする。
		if it.Count > maxHand-slots {
			return fmt.Errorf("手持ちの枠（武器 %d 本 ＋ 道具の個数）が %d を超えます", len(u.Weapons), maxHand)
		}
		slots += it.Count
	}
	// 兵種の杖の回復量の上限は 0 以上（DES-IT06 7.1 節）。
	if u.StaffHealCap < 0 {
		return fmt.Errorf("杖の回復量の上限 %d は 0 以上でなければなりません", u.StaffHealCap)
	}
	// 持てる武器の種類（DES-ITM-02）。種類は Sword〜Magic のどれかで、重ならない。
	if len(u.AllowedWeaponTypes) == 0 {
		return fmt.Errorf("持てる武器の種類がありません")
	}
	allowed := map[WeaponType]bool{}
	for _, t := range u.AllowedWeaponTypes {
		if t < Sword || t > Magic || allowed[t] {
			return fmt.Errorf("持てる武器の種類 %v が正しくないか重なっています", u.AllowedWeaponTypes)
		}
		allowed[t] = true
	}
	// 装備の添字は武器の列の中（DES-ITM-02）。
	if u.Equipped < 0 || u.Equipped >= len(u.Weapons) {
		return fmt.Errorf("装備の添字 %d が武器の列（%d 本）の外です", u.Equipped, len(u.Weapons))
	}
	// 武器の列の各要素を確かめる（規則 4・6・8、DES-ITM-02、DES-BTL-09）。
	for k, w := range u.Weapons {
		if err := validateWeaponOf(u, w); err != nil {
			if legacy {
				return err
			}
			return fmt.Errorf("weapons[%d]: %w", k, err)
		}
		if !allowed[w.Type] {
			return fmt.Errorf("weapons[%d]: 武器の種類 %d は持てる種類にありません", k, w.Type)
		}
	}
	if err := validateTags("兵種の分類", u.Tags); err != nil {
		return err
	}
	if u.Skill.Effect < NoSkill || u.Skill.Effect > Canto {
		return fmt.Errorf("スキルの効果 %d が正しくありません", u.Skill.Effect)
	}
	if u.AI < Charge || u.AI > Hold {
		return fmt.Errorf("動き方 %d が正しくありません", u.AI)
	}
	if u.Approach < Direct || u.Approach > GroupApproach || (u.Approach == GroupApproach && (u.Side != Enemy || u.Group == "")) {
		return fmt.Errorf("寄り方 %d と一団 %q が正しくありません", u.Approach, u.Group)
	}
	// 規則 6: スキルの数値（DES-DAT-03 の規則 6 と同じ範囲）。
	if err := validateSkill(u.Skill); err != nil {
		return err
	}
	// 兵種のスキル: 効果があり、重ならない（DES-IT06 8.1 節）。
	seen := map[SkillEffect]bool{}
	for k, sk := range u.ClassSkills {
		if sk.Effect <= NoSkill || sk.Effect > Canto {
			return fmt.Errorf("classSkills[%d]: スキルの効果 %d が正しくありません", k, sk.Effect)
		}
		if seen[sk.Effect] {
			return fmt.Errorf("classSkills[%d]: 兵種のスキルの効果 %d が重なっています", k, sk.Effect)
		}
		seen[sk.Effect] = true
		if err := validateSkill(sk); err != nil {
			return fmt.Errorf("classSkills[%d]: %w", k, err)
		}
	}
	return validateGrowth(u)
}

// validateSkill はスキル 1 つの数値を確かめる（DES-DAT-03 の規則 6 と同じ範囲）。
func validateSkill(sk Skill) error {
	if sk.Range < 0 || sk.Hit < 0 || sk.Avoid < 0 {
		return fmt.Errorf("スキルの数値（範囲 %d、命中 %d、回避 %d）は 0 以上でなければなりません", sk.Range, sk.Hit, sk.Avoid)
	}
	if sk.Chance < 0 || sk.Chance > 100 {
		return fmt.Errorf("スキルの確率 %d は 0〜100 でなければなりません", sk.Chance)
	}
	if sk.Multiplier < 0 || (sk.Effect == Wrath && sk.Multiplier != 0) {
		return fmt.Errorf("スキルの倍率 %d が正しくありません（0 以上、怒りでは 0）", sk.Multiplier)
	}
	return nil
}

// validateWeaponOf は武器 1 本の規則 4・6・8 を確かめる（DES-BTL-01、DES-HEL-01、DES-BTL-09）。
// 射程は 1 ≦ 最小 ≦ 最大、種類は Sword〜Magic のどれか。
// 杖は Heal と MaxUses が 1 以上（Uses は New で MaxUses にするので見ない）。杖でない武器は Heal・Uses・MaxUses が 0。
// 敵（増援を含む）の武器は杖でない。
func validateWeaponOf(u UnitSetup, w Weapon) error {
	if w.MinRange < 1 || w.MinRange > w.MaxRange {
		return fmt.Errorf("武器の射程 %d〜%d は 1 ≦ 最小 ≦ 最大でなければなりません", w.MinRange, w.MaxRange)
	}
	if w.Type < Sword || w.Type > Magic {
		return fmt.Errorf("武器の種類 %d が正しくありません", w.Type)
	}
	if err := validateWeapon(w); err != nil {
		return err
	}
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
	// 敵のレベルは 1〜20（DES-IT06 7.3 節）。0 を許すと撃破の経験の式で黙って弱く扱われる。
	if u.Side == Enemy && u.Level == 0 {
		return fmt.Errorf("敵のレベル %d は 1〜%d でなければなりません", u.Level, maxLevel)
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
	// 敵は育たないので、経験と成長率はすべて 0（DES-IT06 7.3 節）。
	if u.Side == Enemy && (u.Exp != 0 || g != (Growth{})) {
		return fmt.Errorf("敵の経験 %d と成長率 %+v は 0 でなければなりません", u.Exp, g)
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
	// 種類 Magic の武器は魔法の攻撃だけ（DES-IT06 5.1 節）。
	if w.Type == Magic && w.DamageType != Magical {
		return fmt.Errorf("種類 Magic の武器は魔法の攻撃だけです")
	}
	return validateTags("特効の分類", w.EffectiveTags)
}

// validateTags は分類が armor・mounted のどれかで、重ならないことを確かめる（DES-IT06 5.2 節）。
func validateTags(what string, tags []string) error {
	seen := map[string]bool{}
	for _, tag := range tags {
		if tag != "armor" && tag != "mounted" {
			return fmt.Errorf("%s %q は使えません", what, tag)
		}
		if seen[tag] {
			return fmt.Errorf("%s %q が重なっています", what, tag)
		}
		seen[tag] = true
	}
	return nil
}

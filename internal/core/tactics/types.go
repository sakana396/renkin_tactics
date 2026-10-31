// Package tactics は、戦闘の規則と状態を持つ（DES-PKG-05）。
//
// ゲームの状態（World）と定義データの型（defs）を知らず、Go の標準ライブラリだけに
// 依存する（ADR-0002）。入力は Setup で受け取り、ユニットは Units() の添字で指す。
package tactics

// Pos はマスの座標。左上を (0, 0) とし、x は右、y は下に増える（DES-BTL-01）。
type Pos struct {
	X, Y int
}

// Side は陣営（DES-BTL-01）。
type Side int

const (
	// Player は自軍。
	Player Side = iota
	// Enemy は敵軍。
	Enemy
)

// Result は戦闘の結果（DES-BTL-01）。
type Result int

const (
	// Ongoing は戦闘が続いていることを表す。
	Ongoing Result = iota
	// Won は自軍の勝ち。
	Won
	// Lost は自軍の負け。
	Lost
)

// WeaponType は武器の種類（DES-BTL-01）。
type WeaponType int

const (
	// Sword は剣。
	Sword WeaponType = iota
	// Lance は槍。
	Lance
	// Axe は斧。
	Axe
	// Bow は弓。
	Bow
	// Staff は杖。攻撃も反撃もせず、行動 Heal で味方を回復する（DES-HEL-01）。
	Staff
	// Magic は魔法の種類（DES-BTL-09）。
	Magic
)

// DamageType は物理と魔法の区分（DES-BTL-09）。
type DamageType int

const (
	Physical DamageType = iota
	Magical
)

// SkillEffect はスキルの効果の種類（DES-BTL-01、DES-SKL-01）。
type SkillEffect int

const (
	// NoSkill はスキルを持たないことを表す。
	NoSkill SkillEffect = iota
	// Inspire は鼓舞。
	Inspire
	// Pavise は大盾。
	Pavise
	// Wrath は怒り。
	Wrath
	// Snipe は狙撃。
	Snipe
	// Canto は再移動。兵種のスキルとして持ち、攻撃・回復・道具のあと残りの移動力で動ける（DES-BTL-10）。
	Canto
)

// AI は敵の動き方（DES-BTL-01、DES-AI-01）。
type AI int

const (
	// Charge は突っ込む。
	Charge AI = iota
	// Hold は待ち構える。
	Hold
)

// Action は移動のあとの行動（DES-BTL-01、DES-BTL-04）。
type Action int

const (
	// Wait は待機（移動するだけ）。
	Wait Action = iota
	// Attack は攻撃。
	Attack
	// UseItem はアイテムの使用。
	UseItem
	// Heal は杖による味方の回復（DES-HEL-01）。
	Heal
	// CantoMove は、待っている再移動で動く（DES-BTL-10）。
	CantoMove
	// CantoEnd は、再移動せずに今の位置で終える（DES-BTL-10）。To は今の位置に限る。
	CantoEnd
)

// Stats はユニットの能力（DES-BTL-01）。HP は最大 HP。
type Stats struct {
	HP  int
	Str int
	Skl int
	Spd int
	Def int
	// 幸運・魔力・魔防（DES-UNT-01）。
	Lck int
	Mag int
	Res int
	Mov int
}

// Weapon は武器（DES-BTL-01、DES-BTL-09）。射程はマンハッタン距離の最小と最大。
// Heal・Uses・MaxUses は杖の回復量・残りの使用回数・使用回数の最大で、杖でなければどれも 0（DES-HEL-01）。
type Weapon struct {
	Name          string
	Type          WeaponType
	Crit          int
	Weight        int
	Strikes       int
	DamageType    DamageType
	EffectiveTags []string
	Might         int
	Hit           int
	MinRange      int
	MaxRange      int
	Heal          int
	Uses          int
	MaxUses       int
}

// Terrain は地形（DES-BTL-01、DES-MAP-01）。Cost が 0 の地形には入れない。
type Terrain struct {
	Name        string
	Cost        int
	Avoid       int
	Def         int
	HealPercent int
}

// Item は消耗品（DES-BTL-01、DES-ITM-01）。
// ID は tactics の外（core）がアイテムを見分けるための値で、tactics は使わない（DES-GRW-03）。
type Item struct {
	ID   string
	Name string
	Heal int
}

// ItemStack は手持ちのアイテムと、その数（DES-BTL-01）。
type ItemStack struct {
	Item  Item
	Count int
}

// Skill はスキル（DES-BTL-01、DES-SKL-01）。使わない数値は 0 にする。
type Skill struct {
	Name       string
	Effect     SkillEffect
	Range      int
	Hit        int
	Avoid      int
	Chance     int
	Multiplier int
}

// Growth は成長率（%）。レベルが上がったとき、能力ごとにこの値より小さい乱数を引けば 1 上がる（DES-GRW-02、DES-GRW-03）。
type Growth struct {
	HP  int
	Str int
	Skl int
	Spd int
	Def int
	// 幸運・魔力・魔防（DES-UNT-01）。
	Lck int
	Mag int
	Res int
}

// UnitSetup は、戦闘を始めるときのユニット 1 体の入力（DES-BTL-01）。HP は Stats.HP から始まる。
// Level・Exp・Growth はレベル・経験・成長率（DES-GRW-03）。Level が 0 のユニットは経験を得ない。
// 敵の Level は 1〜20 で、Exp と Growth は 0（DES-IT06 7.3 節）。
type UnitSetup struct {
	Name      string
	Tags      []string // 兵種の特効分類（DES-DAT-04）。
	Weapons   []Weapon // 後続の持ち替え用の列（DES-ITM-02）。
	ClassName string
	// StaffHealCap は兵種の杖の回復量の上限。0 なら上限なし（DES-HEL-01、DES-IT06 7.1 節）。
	StaffHealCap int
	Side         Side
	Stats        Stats
	Level        int
	Exp          int
	Growth       Growth
	Pos          Pos
	Weapon       Weapon
	Items        []ItemStack
	Skill        Skill
	// ClassSkills は兵種のスキル（DES-BTL-10、DES-DAT-04）。Skill（キャラのスキル）と合わせて効果を判定する。
	ClassSkills []Skill
	Lord        bool
	Boss        bool
	AI          AI
	// ID・Group は敵の識別子と一団の名前（DES-MAP-07）。戦闘の中で比べるだけで、定義を引く手がかりにしない。
	ID    string
	Group string
}

// ReinforcementSetup は増援 1 体の入力（DES-BTL-01、DES-MAP-03）。現れるマスは Unit.Pos。
type ReinforcementSetup struct {
	Turn int
	Unit UnitSetup
}

// Setup は戦闘の入力（DES-BTL-01）。Terrain は [y][x] の順に並べる。
type Setup struct {
	Terrain        [][]Terrain
	Units          []UnitSetup
	Reinforcements []ReinforcementSetup
	// Events は出来事の列（DES-MAP-07）。定義の順に評価する。
	Events []MapEventSetup
}

// Unit は戦闘の中のユニット（DES-BTL-01）。倒れたユニットは Alive が偽になり、マップから外れる。
// Level・Exp・Growth はレベル・経験・成長率（DES-GRW-03）。
type Unit struct {
	Name      string
	Tags      []string // 兵種の特効分類（DES-DAT-04）。
	Weapons   []Weapon // 後続の持ち替え用の列（DES-ITM-02）。
	ClassName string
	// StaffHealCap は UnitSetup から写す兵種の杖の回復量の上限（DES-HEL-01）。
	StaffHealCap int
	Side         Side
	Stats        Stats
	Level        int
	Exp          int
	Growth       Growth
	HP           int
	Pos          Pos
	Weapon       Weapon
	Items        []ItemStack
	Skill        Skill
	// ClassSkills は UnitSetup から写す兵種のスキル（DES-BTL-10）。
	ClassSkills []Skill
	Lord        bool
	Boss        bool
	AI          AI
	Acted       bool
	Alive       bool
	// ID・Group は UnitSetup から写す敵の識別子と一団の名前（DES-MAP-07）。
	ID    string
	Group string
}

// Command は、自軍のユニット 1 体の移動と行動（DES-BTL-01、DES-BTL-04）。
// Target は Attack のとき攻撃する相手、Heal のとき回復する味方の添字（DES-HEL-01）。Item は UseItem のときだけ使う。
type Command struct {
	Unit   int
	To     Pos
	Action Action
	Target int
	Item   int
}

// clone はユニットの写しを作る（手持ちの列も写す）。
func (u Unit) clone() Unit {
	u.Tags = append([]string(nil), u.Tags...)
	u.Weapon = u.Weapon.clone()
	u.Weapons = cloneWeapons(u.Weapons)
	u.ClassSkills = cloneSkills(u.ClassSkills)
	if u.Items != nil {
		u.Items = append([]ItemStack(nil), u.Items...)
	}
	return u
}

// clone は UnitSetup のスライスの欄（特効の分類・武器と武器の列の特効・手持ち）を深く写し、入力から独立させる（DES-RWD-01）。
// ターンの増援と出来事の増援の定義を写すときに使う。UnitSetup にスライスの欄を足したら、ここでも写す。
func (us UnitSetup) clone() UnitSetup {
	us.Items = append([]ItemStack(nil), us.Items...)
	us.Tags = append([]string(nil), us.Tags...)
	us.Weapon = us.Weapon.clone()
	us.Weapons = cloneWeapons(us.Weapons)
	us.ClassSkills = cloneSkills(us.ClassSkills)
	return us
}

// cloneSkills は兵種のスキルの列を写す（DES-RWD-01）。nil は nil のまま。
func cloneSkills(s []Skill) []Skill {
	if s == nil {
		return nil
	}
	return append([]Skill(nil), s...)
}

// skill は、キャラのスキルと兵種のスキルを合わせて、効果 e を持つスキルを返す（DES-BTL-10 8.1 節）。
// 同じ効果を二重にかけないよう、最初に見つかった 1 つだけを返す（キャラのスキルが先）。
func (u Unit) skill(e SkillEffect) (Skill, bool) {
	if e == NoSkill {
		return Skill{}, false
	}
	if u.Skill.Effect == e {
		return u.Skill, true
	}
	for _, s := range u.ClassSkills {
		if s.Effect == e {
			return s, true
		}
	}
	return Skill{}, false
}

// clone は特効の分類も独立させる（DES-RWD-01）。
func (w Weapon) clone() Weapon { w.EffectiveTags = append([]string(nil), w.EffectiveTags...); return w }

// cloneWeapons は武器の列を深く写す（DES-RWD-01）。
func cloneWeapons(ws []Weapon) []Weapon {
	if ws == nil {
		return nil
	}
	out := make([]Weapon, len(ws))
	for i, w := range ws {
		out[i] = w.clone()
	}
	return out
}

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
)

// Stats はユニットの能力（DES-BTL-01）。HP は最大 HP。
type Stats struct {
	HP  int
	Str int
	Skl int
	Spd int
	Def int
	Mov int
}

// Weapon は武器（DES-BTL-01）。射程はマンハッタン距離の最小と最大。
type Weapon struct {
	Name     string
	Type     WeaponType
	Might    int
	Hit      int
	MinRange int
	MaxRange int
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
type Item struct {
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

// UnitSetup は、戦闘を始めるときのユニット 1 体の入力（DES-BTL-01）。HP は Stats.HP から始まる。
type UnitSetup struct {
	Name      string
	ClassName string
	Side      Side
	Stats     Stats
	Pos       Pos
	Weapon    Weapon
	Items     []ItemStack
	Skill     Skill
	Lord      bool
	Boss      bool
	AI        AI
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
}

// Unit は戦闘の中のユニット（DES-BTL-01）。倒れたユニットは Alive が偽になり、マップから外れる。
type Unit struct {
	Name      string
	ClassName string
	Side      Side
	Stats     Stats
	HP        int
	Pos       Pos
	Weapon    Weapon
	Items     []ItemStack
	Skill     Skill
	Lord      bool
	Boss      bool
	AI        AI
	Acted     bool
	Alive     bool
}

// Command は、自軍のユニット 1 体の移動と行動（DES-BTL-01、DES-BTL-04）。
// Target は Attack のとき、Item は UseItem のときだけ使う。
type Command struct {
	Unit   int
	To     Pos
	Action Action
	Target int
	Item   int
}

// clone はユニットの写しを作る（手持ちの列も写す）。
func (u Unit) clone() Unit {
	if u.Items != nil {
		u.Items = append([]ItemStack(nil), u.Items...)
	}
	return u
}

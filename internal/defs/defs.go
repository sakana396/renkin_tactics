// Package defs は定義データの型と、ID で引ける目録を持つ（DES-PKG-02、DES-DAT-01）。
package defs

// 武器の種類（DES-DAT-01、E-17 D-2）。
const (
	WeaponSword = "sword"
	WeaponLance = "lance"
	WeaponAxe   = "axe"
	WeaponBow   = "bow"
)

// スキルの効果の種類（DES-DAT-01、DES-SKL-01）。
const (
	EffectInspire = "inspire"
	EffectPavise  = "pavise"
	EffectWrath   = "wrath"
	EffectSnipe   = "snipe"
)

// 敵の動き方（DES-DAT-01、E-13 P-03）。
const (
	AICharge = "charge"
	AIHold   = "hold"
)

// GameDef はゲーム全体の定義（DES-DAT-01）。
type GameDef struct {
	StartMap string
}

// TerrainDef は地形の定義（DES-DAT-01）。Cost が 0 の地形は通れない。
type TerrainDef struct {
	ID          string
	Name        string
	Symbol      string
	Cost        int
	Avoid       int
	Def         int
	HealPercent int
}

// Stats は能力の値（DES-DAT-01）。クラスの基本値と、キャラ・敵のボーナスに使う。
type Stats struct {
	HP  int
	Str int
	Skl int
	Spd int
	Def int
	Mov int
}

// ClassDef はクラスの定義（DES-DAT-01）。
type ClassDef struct {
	ID         string
	Name       string
	WeaponType string
	Base       Stats
}

// WeaponDef は武器の定義（DES-DAT-01）。
type WeaponDef struct {
	ID       string
	Name     string
	Type     string
	Might    int
	Hit      int
	MinRange int
	MaxRange int
}

// ItemDef はアイテムの定義（DES-DAT-01）。
type ItemDef struct {
	ID   string
	Name string
	Heal int
}

// SkillDef はスキルの定義（DES-DAT-01、DES-SKL-01）。使わない数値は 0 にする。
type SkillDef struct {
	ID         string
	Name       string
	Effect     string
	Range      int
	Hit        int
	Avoid      int
	Chance     int
	Multiplier int
}

// ItemCount は手持ちのアイテムとその数（DES-DAT-01）。
type ItemCount struct {
	Item  string
	Count int
}

// CharacterDef は自軍のキャラの定義（DES-DAT-01）。Skill は空でもよい。
type CharacterDef struct {
	ID     string
	Name   string
	Class  string
	Bonus  Stats
	Weapon string
	Items  []ItemCount
	Skill  string
	Lord   bool
}

// AllyPlacement は自軍のキャラの初めの配置（DES-DAT-01）。
type AllyPlacement struct {
	Character string
	X         int
	Y         int
}

// EnemyPlacement は敵の配置（DES-DAT-01）。
type EnemyPlacement struct {
	Class  string
	Weapon string
	Bonus  Stats
	Items  []ItemCount
	AI     string
	Boss   bool
	X      int
	Y      int
}

// Reinforcement は増援（DES-DAT-01）。Turn のターンの始めに現れる。
type Reinforcement struct {
	Turn int
	EnemyPlacement
}

// MapDef はマップの定義（DES-DAT-01）。Rows の 1 文字が 1 マスで、地形の Symbol に対応する。
type MapDef struct {
	ID             string
	Name           string
	Rows           []string
	Allies         []AllyPlacement
	Enemies        []EnemyPlacement
	Reinforcements []Reinforcement
}

// Catalog は定義データの目録で、ID で定義を引ける（DES-DAT-01、DES-PKG-02）。
type Catalog struct {
	Game GameDef

	terrains   []TerrainDef
	terrainIDs map[string]int
	classes    map[string]ClassDef
	weapons    map[string]WeaponDef
	items      map[string]ItemDef
	skills     map[string]SkillDef
	characters map[string]CharacterDef
	maps       map[string]MapDef
}

// NewCatalog は定義の一覧から目録を作る（DES-DAT-01）。
// ID の重なりなどの検証は defload が行う。ID が重なったときは後のものが残る。
func NewCatalog(game GameDef, terrains []TerrainDef, classes []ClassDef, weapons []WeaponDef, items []ItemDef, skills []SkillDef, characters []CharacterDef, maps []MapDef) *Catalog {
	c := &Catalog{
		Game:       game,
		terrainIDs: map[string]int{},
		classes:    map[string]ClassDef{},
		weapons:    map[string]WeaponDef{},
		items:      map[string]ItemDef{},
		skills:     map[string]SkillDef{},
		characters: map[string]CharacterDef{},
		maps:       map[string]MapDef{},
	}
	for _, t := range terrains {
		if i, ok := c.terrainIDs[t.ID]; ok {
			c.terrains[i] = t
			continue
		}
		c.terrainIDs[t.ID] = len(c.terrains)
		c.terrains = append(c.terrains, t)
	}
	for _, v := range classes {
		c.classes[v.ID] = v
	}
	for _, v := range weapons {
		c.weapons[v.ID] = v
	}
	for _, v := range items {
		c.items[v.ID] = v
	}
	for _, v := range skills {
		c.skills[v.ID] = v
	}
	for _, v := range characters {
		c.characters[v.ID] = v
	}
	for _, v := range maps {
		c.maps[v.ID] = v
	}
	return c
}

// Terrain は ID で地形を引く（DES-DAT-01）。
func (c *Catalog) Terrain(id string) (TerrainDef, bool) {
	i, ok := c.terrainIDs[id]
	if !ok {
		return TerrainDef{}, false
	}
	return c.terrains[i], true
}

// Terrains は地形を定義の順にすべて返す（DES-DAT-01）。
// マップの行の文字から地形を引くために使う。
func (c *Catalog) Terrains() []TerrainDef {
	return append([]TerrainDef(nil), c.terrains...)
}

// Class は ID でクラスを引く（DES-DAT-01）。
func (c *Catalog) Class(id string) (ClassDef, bool) {
	v, ok := c.classes[id]
	return v, ok
}

// Weapon は ID で武器を引く（DES-DAT-01）。
func (c *Catalog) Weapon(id string) (WeaponDef, bool) {
	v, ok := c.weapons[id]
	return v, ok
}

// Item は ID でアイテムを引く（DES-DAT-01）。
func (c *Catalog) Item(id string) (ItemDef, bool) {
	v, ok := c.items[id]
	return v, ok
}

// Skill は ID でスキルを引く（DES-DAT-01）。
func (c *Catalog) Skill(id string) (SkillDef, bool) {
	v, ok := c.skills[id]
	return v, ok
}

// Character は ID で自軍のキャラを引く（DES-DAT-01）。
func (c *Catalog) Character(id string) (CharacterDef, bool) {
	v, ok := c.characters[id]
	return v, ok
}

// Map は ID でマップを引く（DES-DAT-01）。
func (c *Catalog) Map(id string) (MapDef, bool) {
	v, ok := c.maps[id]
	return v, ok
}

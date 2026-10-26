// Package defs は定義データの型と、ID で引ける目録を持つ（DES-PKG-02、DES-DAT-01）。
package defs

// 武器の種類（DES-DAT-01、E-17 D-2）。
const (
	WeaponSword = "sword"
	WeaponLance = "lance"
	WeaponAxe   = "axe"
	WeaponBow   = "bow"
	// WeaponStaff は杖。攻撃せず、回復に使う（DES-DAT-01 の反復 2 の変更、v0.3）。
	WeaponStaff = "staff"
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

// 出来事のきっかけと効果（DES-MAP-07）。
const (
	TriggerEnemyMoved    = "enemy_moved"
	TriggerEnemyDefeated = "enemy_defeated"
	TriggerEnemyAttacked = "enemy_attacked"
	TriggerPlayerEntered = "player_entered"
	EventReinforce       = "reinforce"
	EventChargeGroup     = "charge_group"
)

// GameDef はゲーム全体の定義（DES-DAT-01。反復 2 で変更）。
// Party は初めの名簿（キャラの ID の並び）、Stock は初めの共有の在庫（空でもよい）。
type GameDef struct {
	Party []string
	Stock []ItemCount
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

// WeaponDef は武器の定義（DES-DAT-01。反復 2 で変更）。
// Heal（回復量）と Uses（使用回数）は杖（Type が WeaponStaff）だけが持ち、杖でない武器では 0。
// 杖は Might・Hit を使わない（0）。
type WeaponDef struct {
	ID       string
	Name     string
	Type     string
	Might    int
	Hit      int
	MinRange int
	MaxRange int
	Heal     int
	Uses     int
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

// Growth は成長率（%）（DES-DAT-01 の反復 2 の変更）。移動は成長しない。
type Growth struct {
	HP  int
	Str int
	Skl int
	Spd int
	Def int
}

// CharacterDef は自軍のキャラの定義（DES-DAT-01。反復 2 で変更）。Skill は空でもよい。
// Level は加わるときのレベル（定義データで省略すると 1）、Growth は成長率。
type CharacterDef struct {
	ID     string
	Name   string
	Class  string
	Bonus  Stats
	Level  int
	Growth Growth
	Weapon string
	Items  []ItemCount
	Skill  string
	Lord   bool
}

// Slot は出撃の枠 1 つの座標（DES-DAT-01 の反復 2 の変更）。
type Slot struct {
	X int
	Y int
}

// EnemyPlacement は敵の配置（DES-DAT-01。反復 2 で変更）。Name は空なら、クラスの名前から付ける。
type EnemyPlacement struct {
	Name   string
	Class  string
	Weapon string
	Bonus  Stats
	Items  []ItemCount
	AI     string
	Boss   bool
	X      int
	Y      int
	// ID はマップの中で、初めの敵・すべての増援を通して重ならない敵の ID（DES-MAP-07）。
	// 省いた敵には、読み込みのときに「配置の種類 ＋ 定義の添字」で作る。Group は一団の名前（空なら一団に入らない）。
	ID    string
	Group string
}

// Area は出来事の範囲。両端を含む盤の中の長方形（DES-MAP-07）。
type Area struct {
	MinX, MinY, MaxX, MaxY int
}

// MapEvent は出来事 1 件の定義（DES-MAP-07）。きっかけ 1 つ・効果 1 つで、1 回だけ起こる。
// Unit は敵のきっかけの敵の ID、Area は player_entered の範囲。Enemies は reinforce の増援、Group は charge_group の一団。
type MapEvent struct {
	ID      string
	Trigger string
	Unit    string
	Area    Area
	Effect  string
	Enemies []EnemyPlacement
	Group   string
}

// Reinforcement は増援（DES-DAT-01）。Turn のターンの始めに現れる。
type Reinforcement struct {
	Turn int
	EnemyPlacement
}

// MapDef はマップの定義（DES-DAT-01。反復 2 で変更）。Rows の 1 文字が 1 マスで、地形の Symbol に対応する。
// Slots は出撃の枠の並びで、1 番目は主人公の枠。
type MapDef struct {
	ID             string
	Name           string
	Rows           []string
	Slots          []Slot
	Enemies        []EnemyPlacement
	Reinforcements []Reinforcement
	Events         []MapEvent
}

// ChapterDef は章 1 つの定義（DES-DAT-01 の反復 2 の変更）。
// Before・After は物語の ID で、空なら物語を出さない。Joins はその章の始めに名簿に加わるキャラの ID の並び。
// Reward はその章に勝ったときに共有の在庫に入るアイテム。
type ChapterDef struct {
	ID     string
	Name   string
	Map    string
	Before string
	After  string
	Joins  []string
	Reward []ItemCount
}

// StoryDef は物語 1 本の定義（DES-DAT-01 の反復 2 の変更）。Paragraphs は段落の並び。
type StoryDef struct {
	ID         string
	Paragraphs []string
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
	chapters   []ChapterDef
	stories    map[string]StoryDef
}

// NewCatalog は定義の一覧から目録を作る（DES-DAT-01。反復 2 で章と物語を足した）。
// ID の重なりなどの検証は defload が行う。ID が重なったときは後のものが残る（章は並びのまま持つ）。
func NewCatalog(game GameDef, terrains []TerrainDef, classes []ClassDef, weapons []WeaponDef, items []ItemDef, skills []SkillDef, characters []CharacterDef, maps []MapDef, chapters []ChapterDef, stories []StoryDef) *Catalog {
	c := &Catalog{
		Game:       game,
		terrainIDs: map[string]int{},
		classes:    map[string]ClassDef{},
		weapons:    map[string]WeaponDef{},
		items:      map[string]ItemDef{},
		skills:     map[string]SkillDef{},
		characters: map[string]CharacterDef{},
		maps:       map[string]MapDef{},
		chapters:   append([]ChapterDef(nil), chapters...),
		stories:    map[string]StoryDef{},
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
	for _, v := range stories {
		c.stories[v.ID] = v
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

// Chapters は章を、定義の順（章の順）にすべて返す（DES-DAT-01 の反復 2 の変更）。
// 返す並びは写しなので、書き換えても目録は変わらない。
func (c *Catalog) Chapters() []ChapterDef {
	out := make([]ChapterDef, len(c.chapters))
	for i, ch := range c.chapters {
		ch.Joins = append([]string(nil), ch.Joins...)
		ch.Reward = append([]ItemCount(nil), ch.Reward...)
		out[i] = ch
	}
	return out
}

// Story は ID で物語を引く（DES-DAT-01 の反復 2 の変更）。
func (c *Catalog) Story(id string) (StoryDef, bool) {
	v, ok := c.stories[id]
	return v, ok
}

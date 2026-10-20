// Package core はゲームの状態を持つ（DES-PKG-04）。
//
// 反復 1 では、定義データ（defs）から戦闘の入力（tactics.Setup）を組み立てるだけ。
// defs と core/tactics だけに依存する（Ebitengine と time を使わない）。
package core

import (
	"fmt"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// NewBattle は、目録のマップ mapID の定義から戦闘を作る（DES-CORE-01、ADR-0002）。
//
// マップの定義から tactics.Setup を組み立て、tactics.New に種 seed とともに渡す。
// 返す出来事は tactics.New が返す、1 ターン目の自軍の番の始めの出来事。
// 定義に誤り（ID がないなど）があれば、その誤りを返す。tactics.New の誤りはそのまま返す。
func NewBattle(cat *defs.Catalog, mapID string, seed uint64) (*tactics.Battle, []tactics.Event, error) {
	setup, err := newSetup(cat, mapID)
	if err != nil {
		return nil, nil, err
	}
	return tactics.New(setup, seed)
}

// newSetup は、目録のマップ mapID の定義から tactics.Setup を組み立てる（DES-CORE-01）。
//
// 自軍は allies の順、敵は enemies の順に並べ、増援は reinforcements の順に並べる。
// 能力はクラスの Base にボーナスを足したもの（FR-UNT-01、FR-UNT-02）。
func newSetup(cat *defs.Catalog, mapID string) (tactics.Setup, error) {
	m, ok := cat.Map(mapID)
	if !ok {
		return tactics.Setup{}, fmt.Errorf("core: マップ %q がありません", mapID)
	}
	var setup tactics.Setup

	terrain, err := convertTerrain(cat, m)
	if err != nil {
		return tactics.Setup{}, err
	}
	setup.Terrain = terrain

	for i, a := range m.Allies {
		u, err := allyUnit(cat, a)
		if err != nil {
			return tactics.Setup{}, fmt.Errorf("core: マップ %q の allies[%d]: %w", m.ID, i, err)
		}
		setup.Units = append(setup.Units, u)
	}

	names, err := enemyNames(cat, m)
	if err != nil {
		return tactics.Setup{}, err
	}
	for i, e := range m.Enemies {
		u, err := enemyUnit(cat, e, names[i])
		if err != nil {
			return tactics.Setup{}, fmt.Errorf("core: マップ %q の enemies[%d]: %w", m.ID, i, err)
		}
		setup.Units = append(setup.Units, u)
	}
	for i, r := range m.Reinforcements {
		u, err := enemyUnit(cat, r.EnemyPlacement, names[len(m.Enemies)+i])
		if err != nil {
			return tactics.Setup{}, fmt.Errorf("core: マップ %q の reinforcements[%d]: %w", m.ID, i, err)
		}
		setup.Reinforcements = append(setup.Reinforcements, tactics.ReinforcementSetup{Turn: r.Turn, Unit: u})
	}
	return setup, nil
}

// convertTerrain は、マップの行の文字を地形の symbol で地形に変える（DES-CORE-01、DES-DAT-02）。
// 文字は rune で数える。結果は [y][x] の順。
func convertTerrain(cat *defs.Catalog, m defs.MapDef) ([][]tactics.Terrain, error) {
	bySymbol := map[string]tactics.Terrain{}
	for _, t := range cat.Terrains() {
		bySymbol[t.Symbol] = tactics.Terrain{
			Name:        t.Name,
			Cost:        t.Cost,
			Avoid:       t.Avoid,
			Def:         t.Def,
			HealPercent: t.HealPercent,
		}
	}
	rows := make([][]tactics.Terrain, 0, len(m.Rows))
	for y, row := range m.Rows {
		var line []tactics.Terrain
		x := 0
		for _, r := range row {
			t, ok := bySymbol[string(r)]
			if !ok {
				return nil, fmt.Errorf("core: マップ %q の rows[%d]: %d 文字目の地形の文字 %q がありません", m.ID, y, x, string(r))
			}
			line = append(line, t)
			x++
		}
		rows = append(rows, line)
	}
	return rows, nil
}

// allyUnit は自軍のキャラの配置から、ユニット 1 体の入力を作る（DES-CORE-01）。
func allyUnit(cat *defs.Catalog, a defs.AllyPlacement) (tactics.UnitSetup, error) {
	ch, ok := cat.Character(a.Character)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("キャラ %q がありません", a.Character)
	}
	cls, ok := cat.Class(ch.Class)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("キャラ %q のクラス %q がありません", ch.ID, ch.Class)
	}
	weapon, err := convertWeapon(cat, ch.Weapon)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	items, err := convertItems(cat, ch.Items)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	skill, err := convertSkill(cat, ch.Skill)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	return tactics.UnitSetup{
		Name:      ch.Name,
		ClassName: cls.Name,
		Side:      tactics.Player,
		Stats:     addStats(cls.Base, ch.Bonus),
		Pos:       tactics.Pos{X: a.X, Y: a.Y},
		Weapon:    weapon,
		Items:     items,
		Skill:     skill,
		Lord:      ch.Lord,
	}, nil
}

// enemyUnit は敵の配置から、名前 name のユニット 1 体の入力を作る（DES-CORE-01）。
// 敵はスキルを持たない。
func enemyUnit(cat *defs.Catalog, e defs.EnemyPlacement, name string) (tactics.UnitSetup, error) {
	cls, ok := cat.Class(e.Class)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("クラス %q がありません", e.Class)
	}
	weapon, err := convertWeapon(cat, e.Weapon)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	items, err := convertItems(cat, e.Items)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	ai, err := convertAI(e.AI)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	return tactics.UnitSetup{
		Name:      name,
		ClassName: cls.Name,
		Side:      tactics.Enemy,
		Stats:     addStats(cls.Base, e.Bonus),
		Pos:       tactics.Pos{X: e.X, Y: e.Y},
		Weapon:    weapon,
		Items:     items,
		Boss:      e.Boss,
		AI:        ai,
	}, nil
}

// enemyNames は、敵（enemies の順）と増援（reinforcements の順）を通した名前の列を返す（DES-CORE-01）。
//
// 名前はクラスの名前。同じクラスが 2 体以上いれば「ソルジャー 1」のように、出てくる順に番号を付ける。
func enemyNames(cat *defs.Catalog, m defs.MapDef) ([]string, error) {
	placements := append([]defs.EnemyPlacement(nil), m.Enemies...)
	for _, r := range m.Reinforcements {
		placements = append(placements, r.EnemyPlacement)
	}
	total := map[string]int{}
	for _, p := range placements {
		total[p.Class]++
	}
	seen := map[string]int{}
	names := make([]string, len(placements))
	for i, p := range placements {
		cls, ok := cat.Class(p.Class)
		if !ok {
			where := fmt.Sprintf("enemies[%d]", i)
			if i >= len(m.Enemies) {
				where = fmt.Sprintf("reinforcements[%d]", i-len(m.Enemies))
			}
			return nil, fmt.Errorf("core: マップ %q の %s: クラス %q がありません", m.ID, where, p.Class)
		}
		names[i] = cls.Name
		if total[p.Class] >= 2 {
			seen[p.Class]++
			names[i] = fmt.Sprintf("%s %d", cls.Name, seen[p.Class])
		}
	}
	return names, nil
}

// addStats はクラスの基本値にボーナスを足す（DES-CORE-01、FR-UNT-01）。
func addStats(base, bonus defs.Stats) tactics.Stats {
	return tactics.Stats{
		HP:  base.HP + bonus.HP,
		Str: base.Str + bonus.Str,
		Skl: base.Skl + bonus.Skl,
		Spd: base.Spd + bonus.Spd,
		Def: base.Def + bonus.Def,
		Mov: base.Mov + bonus.Mov,
	}
}

// convertWeapon は武器の ID から tactics の武器を作る（DES-CORE-01）。
func convertWeapon(cat *defs.Catalog, id string) (tactics.Weapon, error) {
	w, ok := cat.Weapon(id)
	if !ok {
		return tactics.Weapon{}, fmt.Errorf("武器 %q がありません", id)
	}
	var typ tactics.WeaponType
	switch w.Type {
	case defs.WeaponSword:
		typ = tactics.Sword
	case defs.WeaponLance:
		typ = tactics.Lance
	case defs.WeaponAxe:
		typ = tactics.Axe
	case defs.WeaponBow:
		typ = tactics.Bow
	default:
		return tactics.Weapon{}, fmt.Errorf("武器 %q の種類 %q は使えません", id, w.Type)
	}
	return tactics.Weapon{
		Name:     w.Name,
		Type:     typ,
		Might:    w.Might,
		Hit:      w.Hit,
		MinRange: w.MinRange,
		MaxRange: w.MaxRange,
	}, nil
}

// convertItems は手持ちのアイテムの列を tactics の手持ちに変える（DES-CORE-01、FR-ITM-03）。
// 列が空なら nil を返す。
func convertItems(cat *defs.Catalog, items []defs.ItemCount) ([]tactics.ItemStack, error) {
	var out []tactics.ItemStack
	for _, ic := range items {
		it, ok := cat.Item(ic.Item)
		if !ok {
			return nil, fmt.Errorf("アイテム %q がありません", ic.Item)
		}
		out = append(out, tactics.ItemStack{Item: tactics.Item{Name: it.Name, Heal: it.Heal}, Count: ic.Count})
	}
	return out, nil
}

// convertSkill はスキルの ID から tactics のスキルを作る（DES-CORE-01、DES-SKL-01）。
// 数値は定義データ（skills.toml）の値をそのまま渡す（FR-SYS-03）。ID が空ならスキルなし（ゼロ値）。
func convertSkill(cat *defs.Catalog, id string) (tactics.Skill, error) {
	if id == "" {
		return tactics.Skill{}, nil
	}
	s, ok := cat.Skill(id)
	if !ok {
		return tactics.Skill{}, fmt.Errorf("スキル %q がありません", id)
	}
	var effect tactics.SkillEffect
	switch s.Effect {
	case defs.EffectInspire:
		effect = tactics.Inspire
	case defs.EffectPavise:
		effect = tactics.Pavise
	case defs.EffectWrath:
		effect = tactics.Wrath
	case defs.EffectSnipe:
		effect = tactics.Snipe
	default:
		return tactics.Skill{}, fmt.Errorf("スキル %q の効果の種類 %q は使えません", id, s.Effect)
	}
	return tactics.Skill{
		Name:       s.Name,
		Effect:     effect,
		Range:      s.Range,
		Hit:        s.Hit,
		Avoid:      s.Avoid,
		Chance:     s.Chance,
		Multiplier: s.Multiplier,
	}, nil
}

// convertAI は敵の動き方の文字列を tactics の動き方に変える（DES-CORE-01、DES-AI-01）。
func convertAI(ai string) (tactics.AI, error) {
	switch ai {
	case defs.AICharge:
		return tactics.Charge, nil
	case defs.AIHold:
		return tactics.Hold, nil
	default:
		return 0, fmt.Errorf("動き方 %q は使えません", ai)
	}
}

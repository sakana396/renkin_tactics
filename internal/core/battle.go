// Package core はゲームの状態を持つ（DES-PKG-04）。
//
// 定義データからの戦闘の組み立てと、章をまたぐキャンペーンの状態を扱う。
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
// 自軍は partyOrder の順に出撃の枠（slots）の数まで並べ、枠の同じ順の座標に置く（反復 2 で変更）。
// 敵は enemies の順に並べ、増援は reinforcements の順に並べる。
// 能力はクラスの Base にボーナスを足したもの（FR-UNT-01、FR-UNT-02）。
func newSetup(cat *defs.Catalog, mapID string) (tactics.Setup, error) {
	m, ok := cat.Map(mapID)
	if !ok {
		return tactics.Setup{}, fmt.Errorf("core: マップ %q がありません", mapID)
	}
	var allies []tactics.UnitSetup
	for i, id := range partyOrder(cat, len(m.Slots)) {
		u, err := allyUnit(cat, id, m.Slots[i])
		if err != nil {
			return tactics.Setup{}, fmt.Errorf("core: マップ %q の slots[%d]: %w", m.ID, i, err)
		}
		allies = append(allies, u)
	}
	return setupWithAllies(cat, m, allies)
}

// setupWithAllies は自軍の配置に敵・増援・地形を足す共通の組み立て（DES-CORE-01、DES-CMP-05）。
func setupWithAllies(cat *defs.Catalog, m defs.MapDef, allies []tactics.UnitSetup) (tactics.Setup, error) {
	terrain, err := convertTerrain(cat, m)
	if err != nil {
		return tactics.Setup{}, err
	}
	setup := tactics.Setup{Terrain: terrain, Units: allies}
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
	k := len(m.Enemies) + len(m.Reinforcements)
	for i, e := range m.Events {
		me, err := convertMapEvent(cat, e, names[k:k+len(e.Enemies)])
		if err != nil {
			return tactics.Setup{}, fmt.Errorf("core: マップ %q の events[%d]: %w", m.ID, i, err)
		}
		k += len(e.Enemies)
		setup.Events = append(setup.Events, me)
	}
	return setup, nil
}

// convertMapEvent は出来事 1 件の定義を tactics の形に変える（DES-MAP-07）。names は増援の敵の名前の列。
func convertMapEvent(cat *defs.Catalog, e defs.MapEvent, names []string) (tactics.MapEventSetup, error) {
	triggers := map[string]tactics.TriggerKind{
		defs.TriggerEnemyMoved:    tactics.TriggerEnemyMoved,
		defs.TriggerEnemyDefeated: tactics.TriggerEnemyDefeated,
		defs.TriggerEnemyAttacked: tactics.TriggerEnemyAttacked,
		defs.TriggerPlayerEntered: tactics.TriggerPlayerEntered,
	}
	effects := map[string]tactics.EffectKind{
		defs.EventReinforce:   tactics.EffectReinforce,
		defs.EventChargeGroup: tactics.EffectChargeGroup,
	}
	trigger, ok := triggers[e.Trigger]
	if !ok {
		return tactics.MapEventSetup{}, fmt.Errorf("きっかけ %q がありません", e.Trigger)
	}
	effect, ok := effects[e.Effect]
	if !ok {
		return tactics.MapEventSetup{}, fmt.Errorf("効果 %q がありません", e.Effect)
	}
	me := tactics.MapEventSetup{
		ID:      e.ID,
		Trigger: trigger,
		Unit:    e.Unit,
		Area:    tactics.Area{Min: tactics.Pos{X: e.Area.MinX, Y: e.Area.MinY}, Max: tactics.Pos{X: e.Area.MaxX, Y: e.Area.MaxY}},
		Effect:  effect,
		Group:   e.Group,
	}
	for j, p := range e.Enemies {
		u, err := enemyUnit(cat, p, names[j])
		if err != nil {
			return tactics.MapEventSetup{}, fmt.Errorf("enemies[%d]: %w", j, err)
		}
		me.Enemies = append(me.Enemies, u)
	}
	return me, nil
}

// convertTerrain は、マップの行の文字を地形の symbol で地形に変える（DES-CORE-01、DES-DAT-02）。
// 文字は rune で数え、誤りの「N 文字目」は 1 から数える（defload と同じ）。結果は [y][x] の順。
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
				return nil, fmt.Errorf("core: マップ %q の rows[%d]: %d 文字目の地形の文字 %q がありません", m.ID, y, x+1, string(r))
			}
			line = append(line, t)
			x++
		}
		rows = append(rows, line)
	}
	return rows, nil
}

// partyOrder は、出撃させる名簿のキャラの ID を、枠に置く順に最大 n 人返す（DES-CORE-01 の反復 2 の変更）。
//
// game.toml の party の順に並べるが、主人公（lord = true のキャラ）は party のどこにいても先頭（主人公の枠）にする。
// 目録にないキャラは、主人公でないものとしてそのまま並べる（allyUnit が誤りにする）。
func partyOrder(cat *defs.Catalog, n int) []string {
	var lords, others []string
	for _, id := range cat.Game.Party {
		if ch, ok := cat.Character(id); ok && ch.Lord {
			lords = append(lords, id)
		} else {
			others = append(others, id)
		}
	}
	order := append(lords, others...)
	if len(order) > n {
		order = order[:n]
	}
	return order
}

// allyUnit は名簿のキャラ id から、出撃の枠 slot に置くユニット 1 体の入力を作る（DES-CORE-01、DES-CMP-05）。
func allyUnit(cat *defs.Catalog, id string, slot defs.Slot) (tactics.UnitSetup, error) {
	ch, ok := cat.Character(id)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("party のキャラ %q がありません", id)
	}
	cls, ok := cat.Class(ch.Class)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("キャラ %q のクラス %q がありません", ch.ID, ch.Class)
	}
	weapons, err := convertWeapons(cat, ch.Weapons)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	allowed, err := convertWeaponType(cls.WeaponType)
	if err != nil {
		return tactics.UnitSetup{}, fmt.Errorf("クラス %q: %w", cls.ID, err)
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
		Name:         ch.Name,
		ClassName:    cls.Name,
		StaffHealCap: cls.StaffHealCap,
		Tags:         append([]string(nil), cls.Tags...),
		Side:         tactics.Player,
		Stats:        addStats(cls.Base, ch.Bonus),
		Level:        ch.Level,
		Growth:       tactics.Growth{HP: ch.Growth.HP, Str: ch.Growth.Str, Skl: ch.Growth.Skl, Spd: ch.Growth.Spd, Def: ch.Growth.Def, Lck: ch.Growth.Lck, Mag: ch.Growth.Mag, Res: ch.Growth.Res},
		Pos:          tactics.Pos{X: slot.X, Y: slot.Y},
		// 初めの装備は武器の列の先頭（DES-ITM-02）。
		Weapons:            weapons,
		AllowedWeaponTypes: []tactics.WeaponType{allowed},

		Items: items,
		Skill: skill,
		Lord:  ch.Lord,
	}, nil
}

// enemyUnit は敵の配置から、名前 name のユニット 1 体の入力を作る（DES-CORE-01）。
// 敵はスキルを持たない。
func enemyUnit(cat *defs.Catalog, e defs.EnemyPlacement, name string) (tactics.UnitSetup, error) {
	cls, ok := cat.Class(e.Class)
	if !ok {
		return tactics.UnitSetup{}, fmt.Errorf("クラス %q がありません", e.Class)
	}
	weapons, err := convertWeapons(cat, e.Weapons)
	if err != nil {
		return tactics.UnitSetup{}, err
	}
	allowed, err := convertWeaponType(cls.WeaponType)
	if err != nil {
		return tactics.UnitSetup{}, fmt.Errorf("クラス %q: %w", cls.ID, err)
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
		Name:         name,
		ClassName:    cls.Name,
		StaffHealCap: cls.StaffHealCap,
		Tags:         append([]string(nil), cls.Tags...),
		Side:         tactics.Enemy,
		Level:        e.Level,
		Stats:        addStats(cls.Base, e.Bonus),
		Pos:          tactics.Pos{X: e.X, Y: e.Y},
		// 敵の武器は敵の定義から渡す（DES-ITM-02）。初めの装備は先頭。
		Weapons:            weapons,
		AllowedWeaponTypes: []tactics.WeaponType{allowed},
		Items:              items,
		Boss:               e.Boss,
		AI:                 ai,
		ID:                 e.ID,
		Group:              e.Group,
	}, nil
}

// enemyNames は、敵（enemies の順）と増援（reinforcements の順）と出来事の増援（events の順・個体の順）を
// 通した名前の列を返す（DES-CORE-01、DES-MAP-07）。
//
// 名前が指定された敵はその名前を使い、数えない。名前のない同じクラスが 2 体以上いれば「ソルジャー 1」のように、出てくる順に番号を付ける。
func enemyNames(cat *defs.Catalog, m defs.MapDef) ([]string, error) {
	placements := append([]defs.EnemyPlacement(nil), m.Enemies...)
	for _, r := range m.Reinforcements {
		placements = append(placements, r.EnemyPlacement)
	}
	for _, e := range m.Events {
		placements = append(placements, e.Enemies...)
	}
	total := map[string]int{}
	for _, p := range placements {
		if p.Name == "" {
			total[p.Class]++
		}
	}
	seen := map[string]int{}
	names := make([]string, len(placements))
	for i, p := range placements {
		cls, ok := cat.Class(p.Class)
		if !ok {
			where := fmt.Sprintf("enemies[%d]", i)
			if i >= len(m.Enemies)+len(m.Reinforcements) {
				where = "events の増援"
			} else if i >= len(m.Enemies) {
				where = fmt.Sprintf("reinforcements[%d]", i-len(m.Enemies))
			}
			return nil, fmt.Errorf("core: マップ %q の %s: クラス %q がありません", m.ID, where, p.Class)
		}
		if p.Name != "" {
			names[i] = p.Name
			continue
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
		Lck: base.Lck + bonus.Lck, Mag: base.Mag + bonus.Mag, Res: base.Res + bonus.Res,
	}
}

// convertWeaponType は武器の種類の文字列を tactics の種類に変える（DES-CORE-01、DES-ITM-02）。
func convertWeaponType(t string) (tactics.WeaponType, error) {
	switch t {
	case defs.WeaponSword:
		return tactics.Sword, nil
	case defs.WeaponLance:
		return tactics.Lance, nil
	case defs.WeaponAxe:
		return tactics.Axe, nil
	case defs.WeaponBow:
		return tactics.Bow, nil
	case defs.WeaponMagic:
		return tactics.Magic, nil
	case defs.WeaponStaff:
		return tactics.Staff, nil
	}
	return 0, fmt.Errorf("武器の種類 %q は使えません", t)
}

// convertWeapons は武器の ID の列を、順を保って tactics の武器の列に変える（DES-CORE-01、DES-ITM-02）。
func convertWeapons(cat *defs.Catalog, ids []string) ([]tactics.Weapon, error) {
	out := make([]tactics.Weapon, 0, len(ids))
	for _, id := range ids {
		w, err := convertWeapon(cat, id)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

// convertWeapon は武器の ID から tactics の武器を作る（DES-CORE-01、DES-HEL-01）。
// 杖の定義の Uses は MaxUses に渡し、戦闘開始時に tactics.New が満タンにする。ID は名簿へ戻すために写す（DES-ITM-02）。
func convertWeapon(cat *defs.Catalog, id string) (tactics.Weapon, error) {
	w, ok := cat.Weapon(id)
	if !ok {
		return tactics.Weapon{}, fmt.Errorf("武器 %q がありません", id)
	}
	typ, err := convertWeaponType(w.Type)
	if err != nil {
		return tactics.Weapon{}, fmt.Errorf("武器 %q の種類 %q は使えません", id, w.Type)
	}
	damageType := tactics.Physical
	if w.DamageType == "magic" {
		damageType = tactics.Magical
	}
	return tactics.Weapon{
		ID:   w.ID,
		Name: w.Name,
		Crit: w.Crit, Weight: w.Weight, Strikes: w.Strikes, DamageType: damageType, EffectiveTags: append([]string(nil), w.EffectiveTags...),
		Type:     typ,
		Might:    w.Might,
		Hit:      w.Hit,
		MinRange: w.MinRange,
		MaxRange: w.MaxRange,
		Heal:     w.Heal,
		MaxUses:  w.Uses,
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
		out = append(out, tactics.ItemStack{Item: tactics.Item{ID: it.ID, Name: it.Name, Heal: it.Heal}, Count: ic.Count})
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

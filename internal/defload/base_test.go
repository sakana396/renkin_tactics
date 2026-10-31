package defload_test

import (
	"io/fs"
	"reflect"
	"sort"
	"testing"
	"unicode/utf8"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体のデータを読み込み、設計書 10 章・3.7 節の値（反復 2 で変えたものは設計書（反復 2）10 章の値）になっていることを確かめる。
func TestBaseData(t *testing.T) {
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatalf("本体のデータに誤りがある:\n%v", err)
	}

	t.Run("TC-FR-SYS-03-01 本体のデータが誤りなく読み込め、10 章の値になっている", func(t *testing.T) {
		if want := []string{"leon", "gald", "bran", "mira", "rina"}; !reflect.DeepEqual(cat.Game.Party, want) {
			t.Errorf("名簿 = %q", cat.Game.Party)
		}
		terrains := []defs.TerrainDef{
			{ID: "plain", Name: "平地", Symbol: ".", Cost: 1},
			{ID: "forest", Name: "森", Symbol: "F", Cost: 2, Avoid: 20, Def: 1},
			{ID: "mountain", Name: "山", Symbol: "^", Cost: 3, Avoid: 30, Def: 2},
			{ID: "fort", Name: "砦", Symbol: "T", Cost: 1, Avoid: 20, Def: 2, HealPercent: 20},
			{ID: "wall", Name: "壁", Symbol: "#", Cost: 0},
		}
		if got := cat.Terrains(); !reflect.DeepEqual(got, terrains) {
			t.Errorf("地形 = %+v", got)
		}
		classes := []defs.ClassDef{
			{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 18, Str: 5, Skl: 7, Spd: 8, Def: 4, Mov: 5, Lck: 6, Mag: 2, Res: 2}},
			{ID: "armor", Name: "アーマー", WeaponType: "lance", Base: defs.Stats{HP: 22, Str: 6, Skl: 3, Spd: 2, Def: 9, Mov: 4, Lck: 2, Mag: 0, Res: 0}},
			{ID: "fighter", Name: "ファイター", WeaponType: "axe", Base: defs.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5, Lck: 2, Mag: 0, Res: 0}},
			{ID: "archer", Name: "アーチャー", WeaponType: "bow", Base: defs.Stats{HP: 17, Str: 5, Skl: 7, Spd: 6, Def: 3, Mov: 5, Lck: 4, Mag: 0, Res: 1}},
			{ID: "soldier", Name: "ソルジャー", WeaponType: "lance", Base: defs.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5, Lck: 2, Mag: 0, Res: 1}},
			{ID: "myrmidon", Name: "剣士", WeaponType: "sword", Base: defs.Stats{HP: 17, Str: 4, Skl: 8, Spd: 9, Def: 2, Mov: 5, Lck: 4, Mag: 1, Res: 1}},
			{ID: "cavalier", Name: "騎兵", WeaponType: "lance", Base: defs.Stats{HP: 20, Str: 6, Skl: 5, Spd: 6, Def: 5, Mov: 7, Lck: 3, Mag: 0, Res: 1}},
			{ID: "cleric", Name: "僧侶", WeaponType: "staff", Base: defs.Stats{HP: 16, Skl: 2, Spd: 6, Def: 2, Mov: 5, Lck: 6, Mag: 0, Res: 5}},
		}
		for _, want := range classes {
			if got, ok := cat.Class(want.ID); !ok || !reflect.DeepEqual(got, want) {
				t.Errorf("クラス %s = %+v, %v", want.ID, got, ok)
			}
		}
		weapons := []defs.WeaponDef{
			{ID: "iron_sword", Name: "鉄の剣", Type: "sword", Weight: 4, Strikes: 1, DamageType: "physical", Might: 5, Hit: 90, MinRange: 1, MaxRange: 1},
			{ID: "iron_lance", Name: "鉄の槍", Type: "lance", Weight: 5, Strikes: 1, DamageType: "physical", Might: 7, Hit: 80, MinRange: 1, MaxRange: 1},
			{ID: "iron_axe", Name: "鉄の斧", Type: "axe", Weight: 7, Strikes: 1, DamageType: "physical", Might: 8, Hit: 70, MinRange: 1, MaxRange: 1},
			{ID: "iron_bow", Name: "鉄の弓", Type: "bow", Weight: 5, Strikes: 1, DamageType: "physical", Might: 6, Hit: 85, MinRange: 2, MaxRange: 2},
			{ID: "heal_staff", Name: "癒やしの杖", Type: "staff", Weight: 0, Strikes: 1, DamageType: "physical", Heal: 10, Uses: 15, MinRange: 1, MaxRange: 1},
		}
		for _, want := range weapons {
			if got, ok := cat.Weapon(want.ID); !ok || !reflect.DeepEqual(got, want) {
				t.Errorf("武器 %s = %+v, %v", want.ID, got, ok)
			}
		}
		if got, ok := cat.Item("potion"); !ok || got != (defs.ItemDef{ID: "potion", Name: "回復薬", Heal: 10}) {
			t.Errorf("アイテム potion = %+v, %v", got, ok)
		}
		skills := []defs.SkillDef{
			{ID: "inspire", Name: "鼓舞", Effect: "inspire", Range: 2, Hit: 10, Avoid: 10},
			{ID: "pavise", Name: "大盾", Effect: "pavise", Chance: 30},
			{ID: "wrath", Name: "怒り", Effect: "wrath", Chance: 50},
			{ID: "snipe", Name: "狙撃", Effect: "snipe", Hit: 20},
		}
		for _, want := range skills {
			if got, ok := cat.Skill(want.ID); !ok || !reflect.DeepEqual(got, want) {
				t.Errorf("スキル %s = %+v, %v", want.ID, got, ok)
			}
		}
		// 能力は設計書（反復 2）10.3 節の「能力」の列（クラス ＋ ボーナス）と一致すること。
		chars := []struct {
			def   defs.CharacterDef
			stats defs.Stats
		}{
			{defs.CharacterDef{ID: "leon", Name: "レオン", Class: "lord", Bonus: defs.Stats{HP: 1, Str: 2, Skl: 4, Spd: 4}, Level: 1, Growth: defs.Growth{HP: 55, Str: 45, Skl: 65, Spd: 70, Def: 20, Lck: 40, Mag: 20, Res: 20}, Weapon: "iron_sword", Items: []defs.ItemCount{{Item: "potion", Count: 2}}, Skill: "inspire", Lord: true}, defs.Stats{HP: 19, Str: 7, Skl: 11, Spd: 12, Def: 4, Mov: 5, Lck: 6, Mag: 2, Res: 2}},
			{defs.CharacterDef{ID: "gald", Name: "ガルド", Class: "armor", Bonus: defs.Stats{HP: 2, Skl: -2, Def: 2}, Level: 1, Growth: defs.Growth{HP: 85, Str: 40, Skl: 20, Spd: 10, Def: 70, Lck: 20, Mag: 0, Res: 10}, Weapon: "iron_lance", Items: []defs.ItemCount{{Item: "potion", Count: 1}}, Skill: "pavise"}, defs.Stats{HP: 24, Str: 6, Skl: 1, Spd: 2, Def: 11, Mov: 4, Lck: 2, Mag: 0, Res: 0}},
			{defs.CharacterDef{ID: "bran", Name: "ブラン", Class: "fighter", Bonus: defs.Stats{HP: -2, Str: 4, Skl: 1, Spd: 1, Def: -1}, Level: 1, Growth: defs.Growth{HP: 70, Str: 75, Skl: 35, Spd: 30, Def: 15, Lck: 25, Mag: 0, Res: 10}, Weapon: "iron_axe", Items: []defs.ItemCount{{Item: "potion", Count: 1}}, Skill: "wrath"}, defs.Stats{HP: 22, Str: 11, Skl: 5, Spd: 5, Def: 2, Mov: 5, Lck: 2, Mag: 0, Res: 0}},
			{defs.CharacterDef{ID: "mira", Name: "ミラ", Class: "archer", Bonus: defs.Stats{Skl: 2, Spd: 1}, Level: 1, Growth: defs.Growth{HP: 55, Str: 40, Skl: 60, Spd: 45, Def: 20, Lck: 35, Mag: 5, Res: 15}, Weapon: "iron_bow", Items: []defs.ItemCount{{Item: "potion", Count: 1}}, Skill: "snipe"}, defs.Stats{HP: 17, Str: 5, Skl: 9, Spd: 7, Def: 3, Mov: 5, Lck: 4, Mag: 0, Res: 1}},
			{defs.CharacterDef{ID: "elsa", Name: "エルザ", Class: "cavalier", Bonus: defs.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1}, Level: 2, Growth: defs.Growth{HP: 65, Str: 45, Skl: 40, Spd: 40, Def: 35, Lck: 30, Mag: 5, Res: 20}, Weapon: "iron_lance", Items: []defs.ItemCount{{Item: "potion", Count: 1}}}, defs.Stats{HP: 21, Str: 7, Skl: 6, Spd: 7, Def: 5, Mov: 7, Lck: 3, Mag: 0, Res: 1}},
			{defs.CharacterDef{ID: "rina", Name: "リナ", Class: "cleric", Level: 1, Growth: defs.Growth{HP: 55, Str: 0, Skl: 40, Spd: 60, Def: 30, Lck: 45, Mag: 50, Res: 45}, Weapon: "heal_staff", Items: []defs.ItemCount{{Item: "potion", Count: 1}}}, defs.Stats{HP: 16, Skl: 2, Spd: 6, Def: 2, Mov: 5, Lck: 6, Mag: 0, Res: 5}},
		}
		for _, want := range chars {
			got, ok := cat.Character(want.def.ID)
			if !ok || !reflect.DeepEqual(got, want.def) {
				t.Errorf("キャラ %s = %+v, %v", want.def.ID, got, ok)
				continue
			}
			c, _ := cat.Class(got.Class)
			if s := addStats(c.Base, got.Bonus); s != want.stats {
				t.Errorf("キャラ %s の能力 = %+v、期待 %+v", got.ID, s, want.stats)
			}
		}

		m, ok := cat.Map("ch01")
		if !ok {
			t.Fatal("マップ ch01 がない")
		}
		if m.Name != "国境の砦" {
			t.Errorf("マップの名前 = %q", m.Name)
		}
		// 出撃の枠は、反復 1 の自軍の配置と同じ座標を同じ順に並べ、5 番目に (2, 15) を足したもの（設計書（反復 2）10.4 節）。
		slots := []defs.Slot{{X: 1, Y: 14}, {X: 2, Y: 13}, {X: 3, Y: 14}, {X: 1, Y: 15}, {X: 2, Y: 15}}
		if !reflect.DeepEqual(m.Slots, slots) {
			t.Errorf("出撃の枠 = %+v", m.Slots)
		}
		enemies := []defs.EnemyPlacement{
			{Level: 1, Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 6, Y: 10},
			{Level: 2, Class: "soldier", Weapon: "iron_lance", AI: "hold", X: 7, Y: 4},
			{Level: 3, Class: "myrmidon", Weapon: "iron_sword", AI: "hold", X: 9, Y: 8},
			{Level: 1, Class: "myrmidon", Weapon: "iron_sword", AI: "hold", X: 11, Y: 3},
			{Level: 2, Class: "fighter", Weapon: "iron_axe", AI: "charge", X: 10, Y: 11},
			{Level: 3, Class: "fighter", Weapon: "iron_axe", AI: "hold", X: 13, Y: 8},
			{Level: 1, Class: "archer", Weapon: "iron_bow", AI: "hold", X: 12, Y: 7},
			{Level: 7, Name: "ハーゲン", Class: "soldier", Weapon: "iron_lance", Bonus: defs.Stats{HP: 8, Str: 3, Skl: 3, Spd: 2, Def: 3}, AI: "hold", Boss: true, X: 13, Y: 1},
		}
		if !reflect.DeepEqual(m.Enemies, enemies) {
			t.Errorf("敵の配置 = %+v", m.Enemies)
		}
		// ボスの能力は 26/8/7/6/7/5（10.4 節）。
		boss := m.Enemies[7]
		soldier, _ := cat.Class(boss.Class)
		if s := addStats(soldier.Base, boss.Bonus); s != (defs.Stats{HP: 26, Str: 8, Skl: 7, Spd: 6, Def: 7, Mov: 5, Lck: 2, Res: 1}) {
			t.Errorf("ボスの能力 = %+v", s)
		}
		reinf := []defs.Reinforcement{
			{Turn: 3, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "fighter", Weapon: "iron_axe", AI: "charge", X: 0, Y: 3}},
			{Turn: 3, EnemyPlacement: defs.EnemyPlacement{Level: 2, Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 0, Y: 4}},
			{Turn: 5, EnemyPlacement: defs.EnemyPlacement{Level: 3, Class: "myrmidon", Weapon: "iron_sword", AI: "charge", X: 15, Y: 15}},
		}
		if !reflect.DeepEqual(m.Reinforcements, reinf) {
			t.Errorf("増援 = %+v", m.Reinforcements)
		}
	})

	t.Run("TC-FR-SYS-03-02 目録にない ID を引くと、ないことが分かる", func(t *testing.T) {
		checks := map[string]bool{}
		_, checks["Terrain"] = cat.Terrain("lava")
		_, checks["Class"] = cat.Class("knight")
		_, checks["Weapon"] = cat.Weapon("silver_sword")
		_, checks["Item"] = cat.Item("elixir")
		_, checks["Skill"] = cat.Skill("luna")
		_, checks["Character"] = cat.Character("nobody")
		_, checks["Map"] = cat.Map("ch99")
		_, checks["空の ID"] = cat.Class("")
		for name, ok := range checks {
			if ok {
				t.Errorf("%s: ない ID で見つかった", name)
			}
		}
		if got, ok := cat.Terrain("fort"); !ok || got.Symbol != "T" {
			t.Errorf("Terrain(fort) = %+v, %v", got, ok)
		}
	})

	t.Run("TC-IF-03-01 本体のデータは 10 の TOML のテキストのファイル", func(t *testing.T) {
		names, err := fs.Glob(data.Base(), "*")
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(names)
		want := []string{"chapters.toml", "characters.toml", "classes.toml", "game.toml", "items.toml", "maps.toml", "skills.toml", "story.toml", "terrains.toml", "weapons.toml"}
		if !reflect.DeepEqual(names, want) {
			t.Errorf("ファイル = %v", names)
		}
		for _, n := range names {
			b, err := fs.ReadFile(data.Base(), n)
			if err != nil {
				t.Fatal(err)
			}
			if !utf8.Valid(b) {
				t.Errorf("%s が UTF-8 のテキストでない", n)
			}
		}
	})

	t.Run("TC-IF-03-02 マップの地形は 1 行を 1 つの文字列で書き、1 文字が 1 マス", func(t *testing.T) {
		m, _ := cat.Map("ch01")
		// 設計書 10.4 節の図と同じ。
		rows := []string{
			"^^^^^^..FFF..^^^",
			"^^^^...FF....T^^",
			"^^.....F....###.",
			".....FF.....#...",
			"..T..F......#...",
			"....FF..^^......",
			".......^^^..FF..",
			"..FF....^...FF..",
			"..FF.........T..",
			".....^^.........",
			"....^^^..FF.....",
			".........FF..^^.",
			"..FF............",
			"..FF...^^.......",
			"........^.......",
			"................",
		}
		if !reflect.DeepEqual(m.Rows, rows) {
			t.Errorf("行 = %q", m.Rows)
		}
		// ボス (13, 1) と敵 (13, 8) は砦の上（10.4 節）。
		bySymbol := map[byte]string{}
		for _, tr := range cat.Terrains() {
			bySymbol[tr.Symbol[0]] = tr.ID
		}
		for _, p := range [][2]int{{13, 1}, {13, 8}} {
			if got := bySymbol[m.Rows[p[1]][p[0]]]; got != "fort" {
				t.Errorf("(%d, %d) の地形 = %s", p[0], p[1], got)
			}
		}
	})
}

func addStats(a, b defs.Stats) defs.Stats {
	return defs.Stats{HP: a.HP + b.HP, Str: a.Str + b.Str, Skl: a.Skl + b.Skl, Spd: a.Spd + b.Spd, Def: a.Def + b.Def, Mov: a.Mov + b.Mov, Lck: a.Lck + b.Lck, Mag: a.Mag + b.Mag, Res: a.Res + b.Res}
}

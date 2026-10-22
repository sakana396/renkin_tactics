package defload_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// baseFS は本体の定義データを写した、書き換えられる fs を返す。
func baseFS(t *testing.T) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(data.Base(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// edit はファイルの中の old を new に置き換える。old がなければテストを止める。
type edit func(t *testing.T, m fstest.MapFS)

func replace(file, old, new string) edit {
	return func(t *testing.T, m fstest.MapFS) {
		t.Helper()
		s := string(m[file].Data)
		if !strings.Contains(s, old) {
			t.Fatalf("%s に %q がありません", file, old)
		}
		m[file].Data = []byte(strings.Replace(s, old, new, 1))
	}
}

func appendTo(file, text string) edit {
	return func(t *testing.T, m fstest.MapFS) {
		m[file].Data = append(m[file].Data, []byte("\n"+text+"\n")...)
	}
}

func remove(file string) edit {
	return func(t *testing.T, m fstest.MapFS) { delete(m, file) }
}

// smallMap は検証を通る小さなマップ（本体の 3 つのマップの後ろの maps[3]）を足す。rows と、追加の TOML を差し替えられる。
func smallMap(rows, extra string) edit {
	return appendTo("maps.toml", `[[maps]]
id = "small"
name = "小さなマップ"
rows = [`+rows+`]
slots = [{ x = 0, y = 0 }]
`+extra)
}

const smallBoss = `
[[maps.enemies]]
class = "soldier"
weapon = "iron_lance"
ai = "hold"
boss = true
x = 1
y = 0
`

func TestLoad(t *testing.T) {
	tests := []struct {
		name  string
		edits []edit
		want  []string // 誤りに含まれるべき文字列。空なら誤りがないこと
	}{
		// 規則 1
		{"TC-FR-SYS-04-01 文法の誤りはファイル名と行番号を示す",
			[]edit{replace("items.toml", "heal = 10", "heal = = 10")},
			[]string{"items.toml: 6 行目: TOML の文法の誤り"}},
		{"TC-FR-SYS-04-01 値の型の誤りもファイル名と行番号を示す",
			[]edit{replace("items.toml", "heal = 10", `heal = "ten"`)},
			[]string{"items.toml: 6 行目: "}},
		// 規則 2
		{"TC-FR-SYS-04-02 ID が重なる",
			[]edit{replace("weapons.toml", `id = "iron_lance"`, `id = "iron_sword"`)},
			[]string{`weapons.toml: weapons[1].id: ID "iron_sword" が重なっています（weapons[0] と同じ）`}},
		{"TC-FR-SYS-04-02 ID が空",
			[]edit{replace("skills.toml", `id = "snipe"`, `id = ""`)},
			[]string{"skills.toml: skills[3].id: ID が空です"}},
		{"TC-FR-SYS-04-02 ID がない（空とみなす）",
			[]edit{appendTo("items.toml", "[[items]]\nname = \"名なし\"\nheal = 5")},
			[]string{"items.toml: items[1].id: ID が空です"}},
		// 規則 3
		{"TC-FR-SYS-04-03 クラスの参照がない",
			[]edit{replace("characters.toml", `class = "fighter"`, `class = "knight"`)},
			[]string{`characters.toml: characters[2].class: クラス "knight" がありません`}},
		{"TC-FR-SYS-04-03 武器の参照がない",
			[]edit{replace("maps.toml", `weapon = "iron_bow"`, `weapon = "steel_bow"`)},
			[]string{`maps.toml: maps[0].enemies[6].weapon: 武器 "steel_bow" がありません`}},
		{"TC-FR-SYS-04-03 アイテムの参照がない",
			[]edit{replace("characters.toml", `items = [{ item = "potion", count = 2 }]`, `items = [{ item = "elixir", count = 2 }]`)},
			[]string{`characters.toml: characters[0].items[0].item: アイテム "elixir" がありません`}},
		{"TC-FR-SYS-04-03 名簿のキャラの参照がない",
			[]edit{replace("game.toml", `"mira"]`, `"nobody"]`)},
			[]string{`game.toml: party[3]: キャラ "nobody" がありません`}},
		{"TC-FR-SYS-04-03 スキルの参照がない",
			[]edit{replace("characters.toml", `skill = "wrath"`, `skill = "rage"`)},
			[]string{`characters.toml: characters[2].skill: スキル "rage" がありません`}},
		{"TC-FR-SYS-04-03 地形の文字がない",
			[]edit{replace("maps.toml", `"................", # y 15`, `"..............X.", # y 15`)},
			[]string{`maps.toml: maps[0].rows[15]: 15 文字目の地形の文字 "X" がありません`}},
		{"TC-FR-SYS-04-03 章のマップがない",
			[]edit{replace("chapters.toml", `map = "ch01"`, `map = "ch99"`)},
			[]string{`chapters.toml: chapters[0].map: マップ "ch99" がありません`}},
		{"TC-FR-SYS-04-03 スキルは空でもよい",
			[]edit{replace("characters.toml", `skill = "wrath"`, `skill = ""`)},
			nil},
		// 規則 4
		{"TC-FR-SYS-04-04 武器の種類が決まった値でない",
			[]edit{replace("weapons.toml", `type = "axe"`, `type = "spear"`)},
			[]string{`weapons.toml: weapons[2].type: 武器の種類 "spear" は使えません（sword・lance・axe・bow・staff のどれか）`}},
		{"TC-FR-SYS-04-04 クラスの武器の種類が決まった値でない",
			[]edit{replace("classes.toml", `weapon_type = "bow"`, `weapon_type = "gun"`)},
			[]string{`classes.toml: classes[3].weapon_type: 武器の種類 "gun" は使えません`}},
		{"TC-FR-SYS-04-04 スキルの効果の種類が決まった値でない",
			[]edit{replace("skills.toml", `effect = "pavise"`, `effect = "shield"`)},
			[]string{`skills.toml: skills[1].effect: スキルの効果の種類 "shield" は使えません（inspire・pavise・wrath・snipe のどれか）`}},
		{"TC-FR-SYS-04-04 敵の動き方が決まった値でない",
			[]edit{replace("maps.toml", `ai = "charge"`, `ai = "flee"`)},
			[]string{`maps.toml: maps[0].enemies[0].ai: 敵の動き方 "flee" は使えません（charge・hold のどれか）`}},
		// 規則 5
		{"TC-FR-SYS-04-05 キャラの武器の種類がクラスと違う",
			[]edit{replace("characters.toml", `weapon = "iron_sword"`, `weapon = "iron_axe"`)},
			[]string{`characters.toml: characters[0].weapon: 武器 "iron_axe" の種類 axe が、クラス "lord" の武器の種類 sword と違います`}},
		{"TC-FR-SYS-04-05 敵の武器の種類がクラスと違う",
			[]edit{replace("maps.toml", "turn = 5\nclass = \"myrmidon\"\nweapon = \"iron_sword\"", "turn = 5\nclass = \"myrmidon\"\nweapon = \"iron_bow\"")},
			[]string{`maps.toml: maps[0].reinforcements[2].weapon: 武器 "iron_bow" の種類 bow が、クラス "myrmidon" の武器の種類 sword と違います`}},
		// 規則 6
		{"TC-FR-SYS-04-06 ボーナスを足した HP が 0",
			[]edit{replace("characters.toml", "bonus = { hp = 0, str = 0, skl = 2", "bonus = { hp = -17, str = 0, skl = 2")},
			[]string{"characters.toml: characters[3].bonus.hp: 能力（クラス 17 ＋ ボーナス -17 ＝ 0）は 1 以上にしてください"}},
		{"TC-FR-SYS-04-06 ボーナスを足した能力が負",
			[]edit{replace("maps.toml", "bonus = { hp = 8, str = 3, skl = 3, spd = 2, def = 3, mov = 0 }", "bonus = { hp = 8, str = 3, skl = 3, spd = 2, def = 3, mov = -6 }")},
			[]string{"maps.toml: maps[0].enemies[7].bonus.mov: 能力（クラス 5 ＋ ボーナス -6 ＝ -1）は 0 以上にしてください"}},
		{"TC-FR-SYS-04-06 クラスの HP を 0 にすると、ボーナスのない敵の HP が 0",
			[]edit{replace("classes.toml", "hp = 17, str = 4", "hp = 0, str = 4")},
			[]string{"maps.toml: maps[0].enemies[2].bonus.hp: 能力（クラス 0 ＋ ボーナス 0 ＝ 0）は 1 以上にしてください"}},
		{"TC-FR-SYS-04-06 武器の威力・命中が負",
			[]edit{replace("weapons.toml", "might = 5\nhit = 90", "might = -1\nhit = -1")},
			[]string{"weapons.toml: weapons[0].might: 0 以上にしてください（-1）", "weapons.toml: weapons[0].hit: 0 以上にしてください（-1）"}},
		{"TC-FR-SYS-04-06 武器の最小の射程が 0",
			[]edit{replace("weapons.toml", "min_range = 1\nmax_range = 1\n\n[[weapons]]\nid = \"iron_lance\"", "min_range = 0\nmax_range = 1\n\n[[weapons]]\nid = \"iron_lance\"")},
			[]string{"weapons.toml: weapons[0].min_range: 1 以上にしてください（0）"}},
		{"TC-FR-SYS-04-06 武器の最小の射程が最大より大きい",
			[]edit{replace("weapons.toml", "min_range = 2\nmax_range = 2", "min_range = 3\nmax_range = 2")},
			[]string{"weapons.toml: weapons[3].max_range: 最大の射程 2 は最小の射程 3 以上にしてください"}},
		{"TC-FR-SYS-04-06 アイテムの回復が 0",
			[]edit{replace("items.toml", "heal = 10", "heal = 0")},
			[]string{"items.toml: items[0].heal: 1 以上にしてください（0）"}},
		{"TC-FR-SYS-04-06 アイテムの数が 0",
			[]edit{replace("characters.toml", `items = [{ item = "potion", count = 2 }]`, `items = [{ item = "potion", count = 0 }]`)},
			[]string{"characters.toml: characters[0].items[0].count: 1 以上にしてください（0）"}},
		{"TC-FR-SYS-04-06 スキルの範囲・命中・回避が負",
			[]edit{replace("skills.toml", "range = 2\nhit = 10\navoid = 10", "range = -1\nhit = -1\navoid = -1")},
			[]string{"skills.toml: skills[0].range: ", "skills.toml: skills[0].hit: ", "skills.toml: skills[0].avoid: "}},
		{"TC-FR-SYS-04-06 スキルの確率が 100 を超える",
			[]edit{replace("skills.toml", "chance = 30", "chance = 101")},
			[]string{"skills.toml: skills[1].chance: 0 以上 100 以下にしてください（101）"}},
		{"TC-FR-SYS-04-06 スキルの確率が負",
			[]edit{replace("skills.toml", "chance = 30", "chance = -1")},
			[]string{"skills.toml: skills[1].chance: 0 以上 100 以下にしてください（-1）"}},
		{"TC-FR-SYS-04-06 怒りの倍率が 0",
			[]edit{replace("skills.toml", "multiplier = 3", "multiplier = 0")},
			[]string{"skills.toml: skills[2].multiplier: 1 以上にしてください（0）"}},
		{"TC-FR-SYS-04-06 怒りでない倍率が負",
			[]edit{replace("skills.toml", "chance = 30", "chance = 30\nmultiplier = -1")},
			[]string{"skills.toml: skills[1].multiplier: 0 以上にしてください（-1）"}},
		{"TC-FR-SYS-04-06 地形の費用・回避・守備が負",
			[]edit{replace("terrains.toml", "cost = 2\navoid = 20\ndef = 1", "cost = -1\navoid = -1\ndef = -1")},
			[]string{"terrains.toml: terrains[1].cost: ", "terrains.toml: terrains[1].avoid: ", "terrains.toml: terrains[1].def: "}},
		{"TC-FR-SYS-04-06 地形の回復の割合が負",
			[]edit{replace("terrains.toml", "heal_percent = 20", "heal_percent = -1")},
			[]string{"terrains.toml: terrains[3].heal_percent: 0 以上 100 以下にしてください（-1）"}},
		{"TC-FR-SYS-04-06 地形の回復の割合が 100 を超える",
			[]edit{replace("terrains.toml", "heal_percent = 20", "heal_percent = 101")},
			[]string{"terrains.toml: terrains[3].heal_percent: 0 以上 100 以下にしてください（101）"}},
		{"TC-FR-SYS-04-07 範囲の端の値は通る",
			[]edit{
				replace("characters.toml", "bonus = { hp = 0, str = 0, skl = 2, spd = 1, def = 0, mov = 0 }", "bonus = { hp = -16, str = -5, skl = -7, spd = -6, def = -3, mov = -5 }"),
				replace("weapons.toml", "might = 5\nhit = 90", "might = 0\nhit = 0"),
				replace("items.toml", "heal = 10", "heal = 1"),
				replace("skills.toml", "chance = 30", "chance = 100"),
				replace("skills.toml", "range = 2\nhit = 10\navoid = 10", "range = 0\nhit = 0\navoid = 0"),
				replace("skills.toml", "multiplier = 3", "multiplier = 1"),
				replace("terrains.toml", "heal_percent = 20", "heal_percent = 100"),
			},
			nil},
		// 規則 7
		{"TC-FR-SYS-04-08 地形の文字が 2 文字",
			[]edit{replace("terrains.toml", `symbol = "F"`, `symbol = "FF"`)},
			[]string{`terrains.toml: terrains[1].symbol: 地形の文字 "FF" は 1 文字にしてください（2 文字）`}},
		{"TC-FR-SYS-04-08 地形の文字が空",
			[]edit{replace("terrains.toml", `symbol = "#"`, `symbol = ""`)},
			[]string{`terrains.toml: terrains[4].symbol: 地形の文字 "" は 1 文字にしてください（0 文字）`}},
		{"TC-FR-SYS-04-08 地形の文字が重なる",
			[]edit{replace("terrains.toml", `symbol = "^"`, `symbol = "F"`)},
			[]string{`terrains.toml: terrains[2].symbol: 地形の文字 "F" が重なっています（terrains[1] と同じ）`}},
		{"TC-FR-SYS-04-09 地形の文字は rune で 1 文字なら通る",
			[]edit{
				appendTo("terrains.toml", "[[terrains]]\nid = \"deep_forest\"\nname = \"深い森\"\nsymbol = \"森\"\ncost = 2"),
				replace("maps.toml", `"................", # y 15`, `"森森森森森森森森森森森森森森森.", # y 15`),
			},
			nil},
		// 規則 8
		{"TC-FR-SYS-04-10 行の長さがそろっていない",
			[]edit{replace("maps.toml", `"................", # y 15`, `"...............", # y 15`)},
			[]string{"maps.toml: maps[0].rows[15]: 行の長さ 15 が、rows[0] の長さ 16 と違います"}},
		{"TC-FR-SYS-04-11 行の数が 17",
			[]edit{replace("maps.toml", `"................", # y 15`, `"................", "................",`)},
			[]string{"maps.toml: maps[0].rows: 行の数 17 は 1 以上 16 以下にしてください"}},
		{"TC-FR-SYS-04-11 行の数が 0",
			[]edit{smallMap(``, smallBoss)},
			[]string{"maps.toml: maps[3].rows: 行の数 0 は 1 以上 16 以下にしてください"}},
		{"TC-FR-SYS-04-11 行の長さが 17",
			[]edit{smallMap(`"................."`, smallBoss)},
			[]string{"maps.toml: maps[3].rows[0]: 行の長さ 17 は 1 以上 16 以下にしてください"}},
		{"TC-FR-SYS-04-11 行の長さが 0",
			[]edit{smallMap(`""`, smallBoss)},
			[]string{"maps.toml: maps[3].rows[0]: 行の長さ 0 は 1 以上 16 以下にしてください"}},
		{"TC-FR-SYS-04-12 1 行 2 マスのマップは通る",
			[]edit{smallMap(`".."`, smallBoss)},
			nil},
		{"TC-FR-SYS-04-12 rune で 16 文字の行は通る",
			[]edit{smallMap(`"..森森森森森森森森森森森森森森"`, smallBoss), appendTo("terrains.toml", "[[terrains]]\nid = \"deep_forest\"\nname = \"深い森\"\nsymbol = \"森\"\ncost = 2")},
			nil},
		// 規則 9
		{"TC-FR-SYS-04-13 配置の座標がマップの外",
			[]edit{replace("maps.toml", "x = 15\ny = 15", "x = 16\ny = 15"), replace("maps.toml", `{ x = 1, y = 15 },`, `{ x = 1, y = -1 },`)},
			[]string{"maps.toml: maps[0].reinforcements[2]: 座標 (16, 15) がマップの外です", "maps.toml: maps[0].slots[3]: 座標 (1, -1) がマップの外です"}},
		{"TC-FR-SYS-04-13 配置の座標が通れない地形",
			[]edit{replace("maps.toml", "x = 11\ny = 3", "x = 12\ny = 3")},
			[]string{`maps.toml: maps[0].enemies[3]: 座標 (12, 3) の地形 "wall" は通れません`}},
		{"TC-FR-SYS-04-13 増援の座標が通れない地形",
			[]edit{replace("maps.toml", "x = 15\ny = 15", "x = 12\ny = 3")},
			[]string{`maps.toml: maps[0].reinforcements[2]: 座標 (12, 3) の地形 "wall" は通れません`}},
		{"TC-FR-SYS-04-13 初めの敵の配置が出撃の枠と重なる",
			[]edit{replace("maps.toml", "x = 12\ny = 7", "x = 1\ny = 14")},
			[]string{"maps.toml: maps[0].enemies[6]: 座標 (1, 14) が maps[0].slots[0] と重なっています"}},
		{"TC-FR-SYS-04-14 増援は初めの配置と重なってもよい",
			[]edit{replace("maps.toml", "x = 15\ny = 15", "x = 1\ny = 15")},
			nil},
		// 規則 10
		{"TC-FR-SYS-04-15 名簿に主人公がいない（名簿の外に lord = true のキャラがいる）",
			[]edit{replace("game.toml", `party = ["leon", `, `party = [`)},
			[]string{"game.toml: party: lord = true のキャラを、ちょうど 1 人入れてください（0 人）", `characters.toml: characters[0].lord: lord = true のキャラ "leon" が party にいません`}},
		{"TC-FR-SYS-04-15 名簿に主人公が 2 人",
			[]edit{replace("characters.toml", `skill = "pavise"`, "skill = \"pavise\"\nlord = true")},
			[]string{"game.toml: party: lord = true のキャラを、ちょうど 1 人入れてください（2 人）"}},
		{"TC-FR-SYS-04-15 名簿に同じキャラを 2 回入れる",
			[]edit{replace("game.toml", `"mira"]`, `"gald"]`)},
			[]string{`game.toml: party[3]: キャラ "gald" を 2 回入れています（party[1] と同じ）`}},
		// 規則 11
		{"TC-FR-SYS-04-16 ボスがいない",
			[]edit{replace("maps.toml", "boss = true\n", "")},
			[]string{"maps.toml: maps[0].enemies: boss = true の敵を、ちょうど 1 体置いてください（0 体）"}},
		{"TC-FR-SYS-04-16 ボスが 2 体",
			[]edit{replace("maps.toml", "ai = \"hold\"\nx = 12", "ai = \"hold\"\nboss = true\nx = 12")},
			[]string{"maps.toml: maps[0].enemies: boss = true の敵を、ちょうど 1 体置いてください（2 体）"}},
		{"TC-FR-SYS-04-16 増援がボス",
			[]edit{replace("maps.toml", "ai = \"charge\"\nx = 15", "ai = \"charge\"\nboss = true\nx = 15")},
			[]string{"maps.toml: maps[0].reinforcements[2].boss: 増援はボスになれません"}},
		// 規則 12
		{"TC-FR-SYS-04-17 増援のターンが 1",
			[]edit{replace("maps.toml", "turn = 5", "turn = 1")},
			[]string{"maps.toml: maps[0].reinforcements[2].turn: 2 以上にしてください（1）"}},
		{"TC-FR-SYS-04-17 増援のターンが 2 なら通る",
			[]edit{replace("maps.toml", "turn = 5", "turn = 2")},
			nil},
		// 規則 13
		{"TC-FR-SYS-04-20 配列の表の中の知らないキー",
			[]edit{replace("terrains.toml", "cost = 2\navoid = 20\ndef = 1\nheal_percent = 0", "cost = 2\navoid = 20\ndef = 1\nheal_precent = 0")},
			[]string{"terrains.toml: terrains[1].heal_precent: 知らないキーです"}},
		{"TC-FR-SYS-04-20 インラインの表の中の知らないキー",
			[]edit{replace("characters.toml", "bonus = { hp = 1, str = 2, skl = 4", "bonus = { hpp = 1, str = 2, skl = 4"), replace("maps.toml", `{ x = 1, y = 15 },`, `{ x = 1, y = 15, z = 0 },`)},
			[]string{"characters.toml: characters[0].bonus.hpp: 知らないキーです", "maps.toml: maps[0].slots[3].z: 知らないキーです"}},
		{"TC-FR-SYS-04-20 入れ子の配列の表と、いちばん上の知らないキー",
			[]edit{replace("maps.toml", "ai = \"charge\"\nx = 6", "aii = \"charge\"\nx = 6"), appendTo("game.toml", `start_mapp = "ch01"`)},
			[]string{"maps.toml: maps[0].enemies[0].aii: 知らないキーです", "game.toml: start_mapp: 知らないキーです"}},
		// 誤りをすべて集める
		{"TC-FR-SYS-04-18 複数のファイルの誤りをすべて返す",
			[]edit{
				replace("items.toml", "heal = 10", "heal = = 10"),
				replace("skills.toml", "chance = 30", "chance = = 30"),
			},
			[]string{"items.toml: 6 行目: ", "skills.toml: 17 行目: "}},
		{"TC-FR-SYS-04-18 複数の意味の誤りをすべて返す",
			[]edit{
				replace("characters.toml", `class = "fighter"`, `class = "knight"`),
				replace("maps.toml", "turn = 5", "turn = 1"),
				replace("chapters.toml", `map = "ch01"`, `map = "ch99"`),
			},
			[]string{`characters.toml: characters[2].class: クラス "knight" がありません`, "maps.toml: maps[0].reinforcements[2].turn: ", "chapters.toml: chapters[0].map: "}},
		{"TC-FR-SYS-04-19 ファイルがない",
			[]edit{remove("skills.toml")},
			[]string{"skills.toml: ファイルを読めません"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseFS(t)
			for _, e := range tt.edits {
				e(t, m)
			}
			cat, err := defload.Load(m)
			if len(tt.want) == 0 {
				if err != nil {
					t.Fatalf("誤りがないはずが、誤りがある:\n%v", err)
				}
				if cat == nil {
					t.Fatal("目録が nil")
				}
				return
			}
			if err == nil {
				t.Fatalf("誤りがあるはずが、誤りがない（期待: %q）", tt.want)
			}
			if cat != nil {
				t.Error("誤りがあるのに目録を返した")
			}
			var errs defload.Errors
			if !errors.As(err, &errs) {
				t.Fatalf("誤りの型が defload.Errors でない: %T", err)
			}
			for _, w := range tt.want {
				found := false
				for _, d := range errs {
					if strings.Contains(d.Error(), w) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("誤り %q が見つからない。誤りの一覧:\n%v", w, err)
				}
			}
		})
	}
}

// TestLoadKeyCase は、大文字・小文字だけが違うキーの扱いが、ほかの知らないキーがあっても変わらないことを確かめる（DES-DAT-03 規則 13、REV-006 N-01）。
// TOML のライブラリは、大文字・小文字だけが違うキーを定義のキーに対応させて読むので、誤りにしない。
func TestLoadKeyCase(t *testing.T) {
	tests := []struct {
		name  string
		edits []edit
		want  []string // 誤りの一覧（過不足なく一致すること）
	}{
		{"TC-FR-SYS-04-21 大文字・小文字だけが違うキーだけなら誤りがない",
			[]edit{replace("terrains.toml", "def = 0\nheal_percent = 0", "def = 0\nHeal_Percent = 0")},
			nil},
		{"TC-FR-SYS-04-21 同じファイルに知らないキーがあっても、大文字・小文字だけが違うキーは誤りにしない",
			[]edit{
				replace("terrains.toml", "def = 0\nheal_percent = 0", "def = 0\nHeal_Percent = 0"),
				replace("terrains.toml", "def = 1\nheal_percent = 0", "def = 1\nheal_precent = 0"),
			},
			[]string{"terrains.toml: terrains[1].heal_precent: 知らないキーです"}},
		{"TC-FR-SYS-04-21 大文字・小文字だけが違う表の中の知らないキー",
			[]edit{replace("characters.toml", "bonus = { hp = 1, str = 2, skl = 4", "Bonus = { hpp = 1, str = 2, skl = 4")},
			[]string{"characters.toml: characters[0].Bonus.hpp: 知らないキーです"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseFS(t)
			for _, e := range tt.edits {
				e(t, m)
			}
			_, err := defload.Load(m)
			var got []string
			if err != nil {
				var errs defload.Errors
				if !errors.As(err, &errs) {
					t.Fatalf("誤りの型が defload.Errors でない: %T", err)
				}
				for _, d := range errs {
					got = append(got, d.Error())
				}
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("誤りの一覧が違う\n得たもの:\n%s\n期待:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

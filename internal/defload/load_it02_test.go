package defload_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体の杖の定義（設計書（反復 2）3.2 節の例）。検証用にこのブロックを置き換える。
const (
	staffWeapon = `[[weapons]]
id = "heal_staff"
name = "癒やしの杖"
type = "staff"
heal = 10
uses = 15
min_range = 1
max_range = 1`
)

// 反復 2 で変えた・足した意味の検証の規則（DES-DAT-03 規則 3、4、6、9、10、13〜22）を確かめる。
func TestLoadIt02(t *testing.T) {
	long := strings.Repeat("あ", 201)
	tests := []struct {
		name  string
		edits []edit
		want  []string // 誤りに含まれるべき文字列。空なら誤りがないこと
	}{
		// 規則 3
		{"TC-FR-SYS-04-24 章の物語・加入のキャラ・報酬と在庫のアイテムの参照がない",
			[]edit{
				replace("chapters.toml", `before = "ch01_before"`, `before = "ch01_intro"`),
				replace("chapters.toml", `after = "ch02_after"`, `after = "ch02_end"`),
				replace("chapters.toml", `joins = ["elsa"]`, `joins = ["nobody"]`),
				replace("chapters.toml", `reward = [{ item = "potion", count = 2 }]`, `reward = [{ item = "elixir", count = 2 }]`),
				appendTo("game.toml", `stock = [{ item = "elixir", count = 1 }]`),
			},
			[]string{
				`chapters.toml: chapters[0].before: 物語 "ch01_intro" がありません`,
				`chapters.toml: chapters[1].after: 物語 "ch02_end" がありません`,
				`chapters.toml: chapters[1].joins[0]: キャラ "nobody" がありません`,
				`chapters.toml: chapters[0].reward[0].item: アイテム "elixir" がありません`,
				`game.toml: stock[0].item: アイテム "elixir" がありません`,
			}},
		{"TC-FR-SYS-04-25 物語のない章、加入・報酬のない章、在庫は通る",
			[]edit{
				replace("chapters.toml", `before = "ch01_before"`+"\n", ``),
				replace("chapters.toml", `after = "ch01_after"`, `after = ""`),
				replace("chapters.toml", `reward = [{ item = "potion", count = 2 }]`, ``),
				appendTo("game.toml", `stock = [{ item = "potion", count = 1 }]`),
			},
			nil},
		// 規則 4・6・21
		{"TC-FR-SYS-04-26 杖の回復量・使用回数が 0",
			[]edit{replace("weapons.toml", staffWeapon, strings.Replace(strings.Replace(staffWeapon, "heal = 10", "heal = 0", 1), "uses = 15", "uses = 0", 1))},
			[]string{"weapons.toml: weapons[4].heal: 杖は heal を 1 以上にしてください（0）", "weapons.toml: weapons[4].uses: 杖は uses を 1 以上にしてください（0）"}},
		{"TC-FR-SYS-04-26 杖の回復量が負で、使用回数を書いていない",
			[]edit{replace("weapons.toml", staffWeapon, strings.Replace(strings.Replace(staffWeapon, "heal = 10", "heal = -1", 1), "uses = 15\n", "", 1))},
			[]string{"weapons.toml: weapons[4].heal: 杖は heal を 1 以上にしてください（-1）", "weapons.toml: weapons[4].uses: 杖は uses を 1 以上にしてください（書かれていません）"}},
		{"TC-FR-SYS-04-26 杖が威力・命中を持つ",
			[]edit{replace("weapons.toml", staffWeapon, staffWeapon+"\nmight = 3\nhit = 80")},
			[]string{"weapons.toml: weapons[4].might: 杖は might を 0 にしてください（3）", "weapons.toml: weapons[4].hit: 杖は hit を 0 にしてください（80）"}},
		{"TC-FR-SYS-04-26 杖でない武器が回復量・使用回数を書く（値が 0 でも誤り）",
			[]edit{
				replace("weapons.toml", "might = 5\nhit = 90", "might = 5\nhit = 90\nuses = 0"),
				replace("weapons.toml", "might = 8\nhit = 70", "might = 8\nhit = 70\nheal = 5"),
			},
			[]string{"weapons.toml: weapons[0].uses: 杖でない武器は uses を持てません", "weapons.toml: weapons[2].heal: 杖でない武器は heal を持てません"}},
		{"TC-FR-SYS-04-27 回復量・使用回数が 1 の杖、杖のクラスとキャラは通る",
			[]edit{
				replace("weapons.toml", staffWeapon, strings.Replace(strings.Replace(staffWeapon, "heal = 10", "heal = 1", 1), "uses = 15", "uses = 1", 1)),
				replace("characters.toml", "level = 1\ngrowth = { hp = 55, str = 0, skl = 40, spd = 60, def = 30 }\n", ""),
			},
			nil},
		// 規則 22
		{"TC-FR-SYS-04-28 敵と増援が杖を持つ",
			[]edit{
				replace("maps.toml", "class = \"soldier\"\nweapon = \"iron_lance\"\nai = \"charge\"\nx = 6", "class = \"cleric\"\nweapon = \"heal_staff\"\nai = \"charge\"\nx = 6"),
				replace("maps.toml", "turn = 5\nclass = \"myrmidon\"\nweapon = \"iron_sword\"", "turn = 5\nclass = \"myrmidon\"\nweapon = \"heal_staff\""),
			},
			[]string{
				`maps.toml: maps[0].enemies[0].weapon: 敵は杖 "heal_staff" を持てません`,
				`maps.toml: maps[0].reinforcements[2].weapon: 敵は杖 "heal_staff" を持てません`,
			}},
		// 規則 9
		{"TC-FR-SYS-04-29 出撃の枠が通れない地形・ほかの枠と重なる",
			[]edit{
				replace("maps.toml", "{ x = 2, y = 13 },", "{ x = 12, y = 3 },"),
				replace("maps.toml", "{ x = 2, y = 15 },", "{ x = 1, y = 14 },"),
			},
			[]string{
				`maps.toml: maps[0].slots[1]: 座標 (12, 3) の地形 "wall" は通れません`,
				"maps.toml: maps[0].slots[4]: 座標 (1, 14) が maps[0].slots[0] と重なっています",
			}},
		// 規則 14
		{"TC-FR-SYS-04-30 章が 1 つもない",
			[]edit{func(t *testing.T, m fstest.MapFS) { m["chapters.toml"].Data = []byte("# 章なし\n") }},
			[]string{"chapters.toml: chapters: 章を 1 つ以上書いてください"}},
		{"TC-FR-SYS-04-30 章のマップに出撃の枠がない",
			[]edit{replace("maps.toml", "slots = [\n  { x = 1, y = 7 },\n  { x = 1, y = 6 },\n  { x = 1, y = 8 },\n  { x = 0, y = 6 },\n  { x = 0, y = 8 },\n  { x = 2, y = 7 },\n]", "slots = []")},
			[]string{`chapters.toml: chapters[1].map: マップ "ch02" に出撃の枠（slots）がありません`}},
		// 規則 15
		{"TC-FR-SYS-04-31 加入のキャラが名簿にもいる",
			[]edit{replace("game.toml", `"rina"]`, `"rina", "elsa"]`)},
			[]string{`chapters.toml: chapters[1].joins[0]: キャラ "elsa" は party にもいます`}},
		{"TC-FR-SYS-04-31 同じキャラがほかの章でも加わる",
			[]edit{replace("chapters.toml", "after = \"ch03_after\"", "after = \"ch03_after\"\njoins = [\"elsa\"]")},
			[]string{`chapters.toml: chapters[2].joins[0]: キャラ "elsa" は chapters[1].joins[0] でも加わります`}},
		// 規則 16
		{"TC-FR-SYS-04-32 レベルが 0・21、成長率が -1・101",
			[]edit{
				replace("characters.toml", "level = 2", "level = 0"),
				replace("characters.toml", `skill = "inspire"`, "skill = \"inspire\"\nlevel = 21"),
				replace("characters.toml", "growth = { hp = 55, str = 40", "growth = { hp = -1, str = 40"),
				replace("characters.toml", "spd = 30, def = 15 }", "spd = 30, def = 101 }"),
			},
			[]string{
				"characters.toml: characters[4].level: 1 以上 20 以下にしてください（0）",
				"characters.toml: characters[0].level: 1 以上 20 以下にしてください（21）",
				"characters.toml: characters[3].growth.hp: 0 以上 100 以下にしてください（-1）",
				"characters.toml: characters[2].growth.def: 0 以上 100 以下にしてください（101）",
			}},
		{"TC-FR-SYS-04-33 レベル 1・20、成長率 0・100 は通る",
			[]edit{
				replace("characters.toml", "level = 2", "level = 20"),
				replace("characters.toml", `skill = "inspire"`, "skill = \"inspire\"\nlevel = 1"),
				replace("characters.toml", "growth = { hp = 55, str = 40", "growth = { hp = 0, str = 40"),
				replace("characters.toml", "spd = 30, def = 15 }", "spd = 30, def = 100 }"),
			},
			nil},
		// 規則 17
		{"TC-FR-SYS-04-34 手持ちの数の合計が 6、在庫の数が 0、報酬に同じアイテムを 2 回書く",
			[]edit{
				replace("characters.toml", `items = [{ item = "potion", count = 2 }]`, `items = [{ item = "potion", count = 3 }, { item = "potion", count = 3 }]`),
				appendTo("game.toml", `stock = [{ item = "potion", count = 0 }]`),
				replace("chapters.toml", `reward = [{ item = "potion", count = 2 }]`, `reward = [{ item = "potion", count = 2 }, { item = "potion", count = 1 }]`),
			},
			[]string{
				"characters.toml: characters[0].items: 手持ちの数の合計 6 は 5 以下にしてください",
				"game.toml: stock[0].count: 1 以上にしてください（0）",
				`chapters.toml: chapters[0].reward[1].item: アイテム "potion" を 2 回書いています（chapters[0].reward[0] と同じ）`,
			}},
		{"TC-FR-SYS-04-35 手持ちの数の合計が 5、在庫の数が 1 なら通る",
			[]edit{
				replace("characters.toml", `items = [{ item = "potion", count = 2 }]`, `items = [{ item = "potion", count = 2 }, { item = "potion", count = 3 }]`),
				appendTo("game.toml", `stock = [{ item = "potion", count = 1 }]`),
			},
			nil},
		// 規則 18
		{"TC-FR-SYS-04-36 段落がない・空の段落・201 文字の段落",
			[]edit{
				replace("story.toml", "id = \"ch01_after\"\nparagraphs = [", "id = \"ch01_after\"\nparagraphs = [\"\", \""+long+"\", "),
				appendTo("story.toml", "[[stories]]\nid = \"empty\"\nparagraphs = []"),
			},
			[]string{
				"story.toml: stories[1].paragraphs[0]: 段落が空です",
				"story.toml: stories[1].paragraphs[1]: 段落の長さ 201 文字は 200 文字以下にしてください",
				"story.toml: stories[6].paragraphs: 段落を 1 つ以上書いてください",
			}},
		{"TC-FR-SYS-04-37 rune で 200 文字の段落は通る",
			[]edit{replace("story.toml", "id = \"ch01_after\"\nparagraphs = [", "id = \"ch01_after\"\nparagraphs = [\""+long[:len(long)-len("あ")]+"\", ")},
			nil},
		// 規則 19
		{"TC-FR-SYS-04-38 敵・増援の名前が空",
			[]edit{
				replace("maps.toml", `name = "ハーゲン"`, `name = ""`),
				replace("maps.toml", "turn = 5\nclass", "turn = 5\nname = \"\"\nclass"),
			},
			[]string{"maps.toml: maps[0].enemies[7].name: 名前が空です", "maps.toml: maps[0].reinforcements[2].name: 名前が空です"}},
		// 規則 20
		{"TC-FR-SYS-04-39 章・物語の ID が重なる・空",
			[]edit{
				replace("chapters.toml", `id = "ch02"`, `id = "ch01"`),
				replace("chapters.toml", `id = "ch03"`, `id = ""`),
				replace("story.toml", `id = "ch02_before"`, `id = "ch01_before"`),
				replace("story.toml", `id = "ch03_after"`, `id = ""`),
			},
			[]string{
				`chapters.toml: chapters[1].id: ID "ch01" が重なっています（chapters[0] と同じ）`,
				"chapters.toml: chapters[2].id: ID が空です",
				`story.toml: stories[2].id: ID "ch01_before" が重なっています（stories[0] と同じ）`,
				"story.toml: stories[5].id: ID が空です",
			}},
		// 規則 13
		{"TC-FR-SYS-04-40 反復 1 の start_map・allies と、足したファイル・キーの知らないキー",
			[]edit{
				appendTo("game.toml", `start_map = "ch01"`),
				replace("maps.toml", "slots = [\n  { x = 1, y = 14 },", "allies = [{ character = \"leon\", x = 1, y = 14 }]\nslots = [\n  { x = 1, y = 14 },"),
				replace("chapters.toml", `reward = [{ item = "potion", count = 2 }]`, `rewards = [{ item = "potion", count = 2 }]`),
				replace("story.toml", "id = \"ch01_before\"\nparagraphs", "id = \"ch01_before\"\nparagraph = \"\"\nparagraphs"),
				replace("characters.toml", "spd = 70, def = 20 }", "spd = 70, def = 20, mov = 10 }"),
			},
			[]string{
				"game.toml: start_map: 知らないキーです",
				"maps.toml: maps[0].allies: 知らないキーです",
				"chapters.toml: chapters[0].rewards: 知らないキーです",
				"story.toml: stories[0].paragraph: 知らないキーです",
				"characters.toml: characters[0].growth.mov: 知らないキーです",
			}},
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

// 杖・僧侶の定義が、目録の値として読めることを確かめる（DES-DAT-01 の反復 2 の変更、v0.3）。
func TestLoadStaff(t *testing.T) {
	t.Run("TC-FR-SYS-04-27 杖は種類 staff・回復量・使用回数を持ち、杖でない武器の回復量・使用回数は 0", func(t *testing.T) {
		m := baseFS(t)
		replace("characters.toml", "level = 1\ngrowth = { hp = 55, str = 0, skl = 40, spd = 60, def = 30 }\n", "")(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatalf("誤りがある:\n%v", err)
		}
		want := defs.WeaponDef{ID: "heal_staff", Name: "癒やしの杖", Type: defs.WeaponStaff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 15}
		if got, ok := cat.Weapon("heal_staff"); !ok || got != want {
			t.Errorf("杖 = %+v, %v", got, ok)
		}
		if got, _ := cat.Weapon("iron_sword"); got.Heal != 0 || got.Uses != 0 {
			t.Errorf("鉄の剣の回復量・使用回数 = %d, %d", got.Heal, got.Uses)
		}
		if got, ok := cat.Class("cleric"); !ok || got.WeaponType != defs.WeaponStaff {
			t.Errorf("僧侶 = %+v, %v", got, ok)
		}
		// 僧侶はボーナスなし・レベルと成長率を省略したキャラ（レベル 1、成長率 0）。
		if got, ok := cat.Character("rina"); !ok || got.Level != 1 || got.Growth != (defs.Growth{}) || got.Bonus != (defs.Stats{}) {
			t.Errorf("リナ = %+v, %v", got, ok)
		}
	})
}

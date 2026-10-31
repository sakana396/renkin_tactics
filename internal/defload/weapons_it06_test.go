package defload_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 武器の列と 5 枠の定義の検証（DES-ITM-02、DES-IT06 6.1・6.2 節）。期待値は設計の 5 枠の数え方から定める。
func TestIT06WeaponList(t *testing.T) {
	const leonWeapon = `weapon = "iron_sword"`
	const enemyWeapon = `weapon = "iron_lance"`
	for _, tc := range []struct {
		name      string
		file, old string
		new       string
		wantErr   []string
		check     func(t *testing.T, chars, enemies []string)
	}{
		{"TC-FR-SYS-04-41 キャラの weapons を順に読み、武器 3 本 ＋ 回復薬 2 個（5 枠ちょうど）は通る", "characters.toml", leonWeapon,
			`weapons = ["iron_sword", "iron_sword", "iron_sword"]`, nil,
			func(t *testing.T, chars, _ []string) {
				if !reflect.DeepEqual(chars, []string{"iron_sword", "iron_sword", "iron_sword"}) {
					t.Fatalf("武器の列 %v", chars)
				}
			}},
		{"TC-FR-SYS-04-41 古い weapon は 1 本の列に直す", "characters.toml", leonWeapon, leonWeapon, nil,
			func(t *testing.T, chars, enemies []string) {
				if !reflect.DeepEqual(chars, []string{"iron_sword"}) || !reflect.DeepEqual(enemies, []string{"iron_lance"}) {
					t.Fatalf("武器の列 %v / %v", chars, enemies)
				}
			}},
		{"TC-FR-SYS-04-41 キャラの武器 4 本 ＋ 回復薬 2 個（6 枠）は拒む", "characters.toml", leonWeapon,
			`weapons = ["iron_sword", "iron_sword", "iron_sword", "iron_sword"]`,
			[]string{"characters.toml: characters[0].items: 手持ちの枠の合計 6（武器 4 本 ＋ 道具 2 個）は 5 以下にしてください"}, nil},
		{"TC-FR-SYS-04-41 weapon と weapons の両方は拒む", "characters.toml", leonWeapon,
			leonWeapon + "\nweapons = [\"iron_sword\"]",
			[]string{"characters.toml: characters[0].weapons: weapon と weapons の両方は書けません"}, nil},
		{"TC-FR-SYS-04-41 列の中の持てない種類と、ない武器は、その位置で拒む", "characters.toml", leonWeapon,
			`weapons = ["iron_sword", "iron_lance", "steel_sword"]`,
			[]string{
				`characters.toml: characters[0].weapons[1]: 武器 "iron_lance" の種類 lance が、クラス "lord" の武器の種類 sword と違います`,
				`characters.toml: characters[0].weapons[2]: 武器 "steel_sword" がありません`,
			}, nil},
		{"TC-FR-SYS-04-41 敵の weapons も順に読む", "maps.toml", enemyWeapon,
			`weapons = ["iron_lance", "iron_lance"]`, nil,
			func(t *testing.T, _, enemies []string) {
				if !reflect.DeepEqual(enemies, []string{"iron_lance", "iron_lance"}) {
					t.Fatalf("敵の武器の列 %v", enemies)
				}
			}},
		{"TC-FR-SYS-04-41 敵の列の中の杖は拒む", "maps.toml", enemyWeapon,
			`weapons = ["iron_lance", "heal_staff"]`,
			[]string{`maps.toml: maps[0].enemies[0].weapons[1]: 敵は杖 "heal_staff" を持てません`}, nil},
		{"TC-FR-SYS-04-41 敵も武器 2 本 ＋ 道具 4 個（6 枠）は拒む", "maps.toml", enemyWeapon,
			"weapons = [\"iron_lance\", \"iron_lance\"]\nitems = [{ item = \"potion\", count = 4 }]",
			[]string{"maps.toml: maps[0].enemies[0].items: 手持ちの枠の合計 6（武器 2 本 ＋ 道具 4 個）は 5 以下にしてください"}, nil},
		{"TC-FR-SYS-04-41 敵の武器 2 本 ＋ 道具 3 個（5 枠ちょうど）は通る", "maps.toml", enemyWeapon,
			"weapons = [\"iron_lance\", \"iron_lance\"]\nitems = [{ item = \"potion\", count = 3 }]", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := baseFS(t)
			replace(tc.file, tc.old, tc.new)(t, m)
			cat, err := defload.Load(m)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if tc.check != nil {
					ch, _ := cat.Character("leon")
					mp, _ := cat.Map("ch01")
					tc.check(t, ch.Weapons, mp.Enemies[0].Weapons)
				}
				return
			}
			if err == nil || cat != nil {
				t.Fatalf("誤りを受理した: %v", err)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("誤り %q に %q がない", err, want)
				}
			}
		})
	}
}

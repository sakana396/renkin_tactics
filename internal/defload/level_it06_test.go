package defload_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 敵のレベル（DES-IT06 7.3 節）と、兵種の杖の回復量の上限（DES-IT06 7.1 節）の読み込みの試験。

func TestIT06BaseEnemyLevels(t *testing.T) {
	t.Run("TC-FR-SYS-03-12 本体の maps.toml の初めの敵・ターンの増援・出来事の増援の全員に level を書いている", func(t *testing.T) {
		b, err := fs.ReadFile(data.Base(), "maps.toml")
		if err != nil {
			t.Fatal(err)
		}
		// 省略を 1 と読む互換の読み込みと区別するため、目録ではなく TOML のキーを見る。
		var raw struct {
			Maps []struct {
				ID             string           `toml:"id"`
				Enemies        []map[string]any `toml:"enemies"`
				Reinforcements []map[string]any `toml:"reinforcements"`
				Events         []struct {
					Enemies []map[string]any `toml:"enemies"`
				} `toml:"events"`
			} `toml:"maps"`
		}
		if _, err := toml.Decode(string(b), &raw); err != nil {
			t.Fatal(err)
		}
		count := 0
		check := func(where string, list []map[string]any) {
			for i, e := range list {
				count++
				if _, ok := e["level"]; !ok {
					t.Errorf("%s[%d] に level がない", where, i)
				}
			}
		}
		for _, m := range raw.Maps {
			check(m.ID+".enemies", m.Enemies)
			check(m.ID+".reinforcements", m.Reinforcements)
			for k, ev := range m.Events {
				check(fmt.Sprintf("%s.events[%d].enemies", m.ID, k), ev.Enemies)
			}
		}
		if len(raw.Maps) != 9 || count == 0 {
			t.Fatalf("マップ %d 個、敵 %d 体", len(raw.Maps), count)
		}
		// 本体を読めること（1〜20 の検査も通ること）。
		if _, err := defload.Load(data.Base()); err != nil {
			t.Fatal(err)
		}
	})
}

// levelMap は、初めの敵（ボス）・ターンの増援・出来事の増援を 1 体ずつ持つ小さなマップ。
// 各敵の level は引数の文字列（空なら省略）。
func levelMap(enemy, reinf, event string) edit {
	line := func(s string) string {
		if s == "" {
			return ""
		}
		return "level = " + s + "\n"
	}
	return smallMap(eventRows, `
[[maps.enemies]]
class = "soldier"
weapon = "iron_lance"
ai = "hold"
boss = true
id = "guard"
x = 1
y = 0
`+line(enemy)+`
[[maps.reinforcements]]
turn = 2
class = "soldier"
weapon = "iron_lance"
ai = "charge"
x = 7
y = 7
`+line(reinf)+`
[[maps.events]]
id = "alarm"
trigger = "enemy_attacked"
unit = "guard"
effect = "reinforce"

[[maps.events.enemies]]
class = "soldier"
weapon = "iron_lance"
ai = "hold"
x = 0
y = 4
`+line(event))
}

func TestIT06LoadEnemyLevel(t *testing.T) {
	t.Run("TC-FR-UNT-04-18 level の省略は 1、1 と 20 は受理する", func(t *testing.T) {
		for _, tc := range []struct {
			enemy, reinf, event string
			want                [3]int
		}{
			{"", "", "", [3]int{1, 1, 1}},
			{"1", "20", "7", [3]int{1, 20, 7}},
			{"20", "1", "20", [3]int{20, 1, 20}},
		} {
			m := baseFS(t)
			levelMap(tc.enemy, tc.reinf, tc.event)(t, m)
			cat, err := defload.Load(m)
			if err != nil {
				t.Fatalf("%+v: %v", tc, err)
			}
			got, _ := cat.Map("small")
			if lv := [3]int{got.Enemies[0].Level, got.Reinforcements[0].Level, got.Events[0].Enemies[0].Level}; lv != tc.want {
				t.Errorf("%+v: レベル %v", tc, lv)
			}
		}
	})

	t.Run("TC-FR-UNT-04-18 level 0 と 21 は位置と理由を示して拒む", func(t *testing.T) {
		for _, bad := range []string{"0", "21"} {
			for _, tc := range []struct {
				name                string
				enemy, reinf, event string
				path                string
			}{
				{"初めの敵", bad, "", "", "maps[N].enemies[0].level"},
				{"ターンの増援", "", bad, "", "maps[N].reinforcements[0].level"},
				{"出来事の増援", "", "", bad, "maps[N].events[0].enemies[0].level"},
			} {
				t.Run(tc.name+" level "+bad, func(t *testing.T) {
					m := baseFS(t)
					levelMap(tc.enemy, tc.reinf, tc.event)(t, m)
					cat, err := defload.Load(m)
					want := appended(t, "maps.toml: "+tc.path+": 1 以上 20 以下にしてください（"+bad+"）")
					if err == nil || cat != nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("誤り = %v, want %q", err, want)
					}
				})
			}
		}
	})
}

func TestIT06LoadStaffHealCap(t *testing.T) {
	const lord = "id = \"lord\"\nname = \"ロード\"\nweapon_type = \"sword\"\n"
	t.Run("TC-FR-BTL-03-19 兵種の staff_heal_cap を読み、省略は 0（上限なし）", func(t *testing.T) {
		m := baseFS(t)
		replace("classes.toml", lord, lord+"staff_heal_cap = 8\n")(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		lordClass, _ := cat.Class("lord")
		cleric, _ := cat.Class("cleric")
		if lordClass.StaffHealCap != 8 || cleric.StaffHealCap != 0 {
			t.Fatalf("上限: ロード %d、僧侶 %d", lordClass.StaffHealCap, cleric.StaffHealCap)
		}
	})
	t.Run("TC-FR-BTL-03-19 負の staff_heal_cap は拒む", func(t *testing.T) {
		m := baseFS(t)
		replace("classes.toml", lord, lord+"staff_heal_cap = -1\n")(t, m)
		cat, err := defload.Load(m)
		want := "classes.toml: classes[0].staff_heal_cap: 0 以上にしてください（-1）"
		if err == nil || cat != nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("誤り = %v, want %q", err, want)
		}
	})
}

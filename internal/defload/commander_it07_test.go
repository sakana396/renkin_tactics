package defload_test

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 小隊長の定義と同じ到達領域の重複を確かめる（TC-FR-SYS-04-46、DES-DAT-05）。
func TestCommanderDefinitionIt07(t *testing.T) {
	base := smallBoss + enemyAt("maps.enemies", 3, 3, "id = \"leader\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true")
	for _, tc := range []struct{ name, data, want string }{
		{"正常", base, ""},
		{"ID なし", smallBoss + enemyAt("maps.enemies", 3, 3, "group = \"g\"\napproach = \"group\"\ncommander = true"), "小隊長には一意の id"},
		{"直進", smallBoss + enemyAt("maps.enemies", 3, 3, "id = \"leader\"\ngroup = \"g\"\napproach = \"direct\"\ncommander = true"), "小隊長の寄り方"},
		{"名前なし", smallBoss + enemyAt("maps.enemies", 3, 3, "id = \"leader\"\napproach = \"group\"\ncommander = true"), "小隊長には一団の名前"},
		{"ボス", smallBoss + enemyAt("maps.enemies", 3, 3, "id = \"leader\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true\nboss = true"), "小隊長はボスにできません"},
		{"突撃", smallBoss + strings.Replace(enemyAt("maps.enemies", 3, 3, "id = \"leader\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true"), `ai = "hold"`, `ai = "charge"`, 1), "小隊長は hold"},
		{"ID 重複", base + enemyAt("maps.enemies", 4, 4, "id = \"leader\"\ngroup = \"other\"\napproach = \"group\"\ncommander = true"), "ID \"leader\" が重なっています"},
		{"同じ領域", base + enemyAt("maps.enemies", 4, 4, "id = \"second\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true"), "小隊長が 2 体"},
	} {
		t.Run("TC-FR-SYS-04-46 "+tc.name, func(t *testing.T) {
			m := baseFS(t)
			smallMap(eventRows, tc.data)(t, m)
			cat, err := defload.Load(m)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, _ := cat.Map("small")
			if !got.Enemies[1].Commander {
				t.Fatal("小隊長の印が消えた")
			}
		})
	}
	t.Run("TC-FR-SYS-04-46 出来事の小隊長と初期配置の重複", func(t *testing.T) {
		data := base + `
[[maps.events]]
id = "arrival"
trigger = "player_entered"
area = { min_x = 0, min_y = 1, max_x = 0, max_y = 1 }
effect = "reinforce"
` + enemyAt("maps.events.enemies", 4, 4, "id = \"arrival_leader\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true")
		m := baseFS(t)
		smallMap(eventRows, data)(t, m)
		_, err := defload.Load(m)
		if err == nil || !strings.Contains(err.Error(), "小隊長が 2 体") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("TC-FR-SYS-04-46 離れた同名の一団", func(t *testing.T) {
		rows := strings.Repeat(`"....#...",`, 8)
		data := smallBoss + enemyAt("maps.enemies", 3, 3, "id = \"west\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true") +
			enemyAt("maps.enemies", 5, 3, "id = \"east\"\ngroup = \"g\"\napproach = \"group\"\ncommander = true")
		m := baseFS(t)
		smallMap(rows, data)(t, m)
		if _, err := defload.Load(m); err != nil {
			t.Fatal(err)
		}
	})
}

package defload_test

import (
	"github.com/sakana396/renkin_tactics/internal/defload"
	"strings"
	"testing"
)

// 追加欄の許容範囲は DES-IT06 3.1・5.1 節から定める。
func TestIT06Definition(t *testing.T) {
	t.Run("TC-FR-SYS-04-42", func(t *testing.T) {
		for _, tc := range []struct {
			file, old, value, path string
			valid                  bool
		}{
			{"classes.toml", "lck = 6", "lck = -1", "lck", false},
			{"classes.toml", "mag = 2", "mag = -1", "mag", false},
			{"classes.toml", "res = 2", "res = -1", "res", false},
			{"characters.toml", "lck = 40", "lck = 0", "", true},
			{"characters.toml", "mag = 20", "mag = 100", "", true},
			{"characters.toml", "res = 20", "res = 101", "growth.res", false},
			{"weapons.toml", "crit = 0", "crit = 101", "crit", false},
			{"weapons.toml", "weight = 4", "weight = -1", "weight", false},
			{"weapons.toml", "strikes = 1", "strikes = 0", "strikes", false},
			{"weapons.toml", "damage_type = \"physical\"", "damage_type = \"unknown\"", "damage_type", false},
			{"weapons.toml", "effective_tags = []", "effective_tags = [\"unknown\"]", "effective_tags", false},
			{"skills.toml", "chance = 50", "chance = 101", "chance", false},
			{"skills.toml", "chance = 50", "chance = 0", "", true},
			{"skills.toml", "chance = 50", "chance = 100", "", true},
			{"skills.toml", "chance = 50", "chance = 50\nmultiplier = 3", "multiplier", false},
			{"classes.toml", "lck = 6", "luck = 6", "luck", false},
			// 兵種の分類・特効の分類は armor・mounted だけで重ならない。種類 magic は魔法の攻撃だけ（DES-IT06 5.1・5.2 節、REV-069 F-04）。
			{"classes.toml", `weapon_type = "sword"`, "weapon_type = \"sword\"\ntags = [\"flying\"]", "classes[0].tags[0]", false},
			{"classes.toml", `weapon_type = "sword"`, "weapon_type = \"sword\"\ntags = [\"armor\", \"armor\"]", "classes[0].tags[1]", false},
			{"classes.toml", `weapon_type = "sword"`, "weapon_type = \"sword\"\ntags = [\"armor\", \"mounted\"]", "", true},
			{"weapons.toml", "effective_tags = []", "effective_tags = [\"armor\", \"armor\"]", "effective_tags[1]", false},
			{"weapons.toml", "", `[[weapons]]
id = "test_fire"
name = "試しの炎"
type = "magic"
damage_type = "physical"
might = 3
hit = 90
min_range = 1
max_range = 2`, "damage_type", false},
			{"weapons.toml", "", `[[weapons]]
id = "test_fire"
name = "試しの炎"
type = "magic"
damage_type = "magic"
might = 3
hit = 90
min_range = 1
max_range = 2`, "", true},
			{"weapons.toml", `damage_type = "physical"`, `damage_type = "magic"`, "", true},
		} {
			t.Run(tc.old+tc.value, func(t *testing.T) {
				m := baseFS(t)
				if tc.old == "" {
					appendTo(tc.file, tc.value)(t, m)
				} else {
					replace(tc.file, tc.old, tc.value)(t, m)
				}
				_, err := defload.Load(m)
				if tc.valid {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), map[bool]string{true: "characters.toml", false: tc.file}[tc.file == "classes.toml" && (tc.path == "lck" || tc.path == "mag" || tc.path == "res")]) || !strings.Contains(err.Error(), tc.path) {
					t.Fatalf("検証=%v", err)
				}
			})
		}
	})
}

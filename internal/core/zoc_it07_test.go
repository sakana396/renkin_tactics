package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestZOCConversion(t *testing.T) {
	t.Run("TC-FR-UNT-02-04 本体の重騎士は自軍も敵軍も通せんぼを兵種から得る", func(t *testing.T) {
		cat := baseCatalog(t)
		setup, err := core.NewSetup(cat, "ch04")
		if err != nil {
			t.Fatal(err)
		}
		player, enemy := 0, 0
		for _, u := range setup.Units {
			has := false
			for _, sk := range u.ClassSkills {
				if sk.Effect == tactics.ZOC {
					has = true
				}
			}
			if u.ClassName == "アーマー" {
				if !has {
					t.Errorf("重騎士 %s に ZOC がない", u.Name)
				}
				if u.Side == tactics.Player {
					player++
					if u.Name == "ガルド" && u.Skill.Effect != tactics.Pavise {
						t.Error("ガルドの大盾が変わった")
					}
				} else {
					enemy++
				}
			} else if has {
				t.Errorf("%s に ZOC がある", u.Name)
			}
		}
		if player == 0 || enemy == 0 {
			t.Errorf("自軍 %d・敵軍 %d の重騎士", player, enemy)
		}
	})
}

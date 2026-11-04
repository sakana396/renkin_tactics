package core

import (
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 定義の小隊長の印を tactics に渡す（TC-FR-SYS-04-46、DES-DAT-05）。
func TestCommanderConversionIt07(t *testing.T) {
	t.Run("TC-FR-SYS-04-46 core への変換", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		e := defs.EnemyPlacement{Level: 1, Class: "soldier", Weapons: []string{"iron_lance"}, AI: defs.AIHold, Approach: defs.ApproachGroup, ID: "leader", Group: "g", Commander: true}
		u, err := enemyUnit(cat, e, "小隊長")
		if err != nil {
			t.Fatal(err)
		}
		if !u.Commander || u.Group != "g" || u.Approach != tactics.GroupApproach || u.ID != "leader" {
			t.Fatalf("変換 = %+v", u)
		}
	})
}

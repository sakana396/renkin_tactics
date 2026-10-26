package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 出来事の定義を tactics の形に変える（DES-MAP-07、DES-CORE-01）。
func TestMapEventConversion(t *testing.T) {
	eventMap := func(m *defs.MapDef, _ *[]defs.CharacterDef) {
		m.Rows = []string{".....", ".....", "....."}
		m.Enemies = []defs.EnemyPlacement{
			{Class: "soldier", Weapon: "iron_lance", AI: "hold", Boss: true, X: 4, Y: 0},
			{Class: "soldier", Weapon: "iron_lance", AI: "hold", X: 4, Y: 2, ID: "guard", Group: "g"},
		}
		area := defs.Area{MinX: 1, MinY: 0, MaxX: 1, MaxY: 0}
		m.Events = []defs.MapEvent{
			{ID: "advance", Trigger: defs.TriggerPlayerEntered, Area: area, Effect: defs.EventChargeGroup, Group: "g"},
			{ID: "alarm", Trigger: defs.TriggerPlayerEntered, Area: area, Effect: defs.EventReinforce, Enemies: []defs.EnemyPlacement{
				{Class: "fighter", Weapon: "iron_axe", AI: "hold", X: 0, Y: 2},
				{Class: "fighter", Weapon: "iron_axe", AI: "hold", X: 1, Y: 2},
			}},
		}
	}
	t.Run("TC-FR-MAP-03-20 きっかけ・効果・範囲・一団・増援の名前を変える", func(t *testing.T) {
		b, _, err := core.NewBattle(smallCatalog(eventMap), "m", 1)
		if err != nil {
			t.Fatal(err)
		}
		if u := b.Units()[2]; u.ID != "guard" || u.Group != "g" || u.AI != tactics.Hold {
			t.Fatalf("門番 %+v", u)
		}
		// 主人公だけなので、行動のあと敵軍の番に進み、自軍の番に予約した増援がその番の始めに出る。
		events, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.Wait})
		if err != nil {
			t.Fatal(err)
		}
		var trig, names []string
		charged := ""
		for _, e := range events {
			switch e.Kind {
			case tactics.EventTrigger:
				trig = append(trig, e.MapEvent)
			case tactics.EventGroupCharge:
				charged = e.Group
				if e.Units[2].AI != tactics.Charge {
					t.Errorf("一団の敵が突撃でない")
				}
			case tactics.EventReinforce:
				names = append(names, e.Units[e.Unit].Name)
			}
		}
		if len(trig) != 2 || trig[0] != "advance" || trig[1] != "alarm" || charged != "g" {
			t.Fatalf("きっかけ %v、突撃 %q", trig, charged)
		}
		if len(names) != 2 || names[0] != "ファイター 1" || names[1] != "ファイター 2" {
			t.Fatalf("増援の名前 %v", names)
		}
	})
	t.Run("TC-FR-MAP-03-20 知らないきっかけ・効果を拒む", func(t *testing.T) {
		for _, edit := range []func(e *defs.MapEvent){
			func(e *defs.MapEvent) { e.Trigger = "enemy_waited" },
			func(e *defs.MapEvent) { e.Effect = "retreat" },
		} {
			cat := smallCatalog(func(m *defs.MapDef, c *[]defs.CharacterDef) {
				eventMap(m, c)
				edit(&m.Events[0])
			})
			if b, _, err := core.NewBattle(cat, "m", 1); err == nil || b != nil {
				t.Fatalf("受理した: %v", err)
			}
		}
	})
}

package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestZOCMarks(t *testing.T) {
	setup := func() tactics.Setup {
		s := basicSetup()
		blue := tactics.UnitSetup{Name: "味方の重騎士", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 0}, Pos: tactics.Pos{X: 3, Y: 1}, Weapon: testLance, ClassSkills: []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}}
		red := tactics.UnitSetup{Name: "敵の重騎士", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20, Mov: 0}, Pos: tactics.Pos{X: 3, Y: 3}, Weapon: testLance, ClassSkills: []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}, AI: tactics.Hold}
		s.Units = append(s.Units, blue, red)
		return s
	}
	c := newIdleWith(t, setup)
	p := tactics.Pos{X: 3, Y: 2}
	t.Run("TC-FR-UNT-09-29 両軍の角印と移動費の説明", func(t *testing.T) {
		sides := c.Battle().BoardView().ZOCSides(p)
		if !sides[tactics.Player] || !sides[tactics.Enemy] {
			t.Errorf("角印の陣営 = %v", sides)
		}
		v := c.Panel(p, true).View()
		found := false
		for _, line := range v.Texts {
			if line.Text == "相手の重騎士の隣: +1" {
				found = true
			}
		}
		if !found {
			t.Error("移動費の説明がない")
		}
		g := &Game{drawCamera: c.Camera()}
		screen := ebiten.NewImage(MapSize, MapSize)
		g.drawZOC(screen, c.Battle().BoardView())
		pt := g.drawCamera.Screen(p)
		if _, _, _, a := screen.At(pt.X+5, pt.Y+5).RGBA(); a == 0 {
			t.Error("自軍の青い角印がない")
		}
		if _, _, _, a := screen.At(pt.X+CellSize-5, pt.Y+CellSize-5).RGBA(); a == 0 {
			t.Error("敵軍の赤い角印がない")
		}
	})
}

func TestZOCDanger(t *testing.T) {
	t.Run("TC-FR-UNT-09-28 敵の範囲は重騎士の ZOC を含む Reachable から求める", func(t *testing.T) {
		setup := func() tactics.Setup {
			armor := tactics.UnitSetup{Name: "重騎士", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 2, Y: 0}, Weapon: testLance, ClassSkills: []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}}
			enemy := tactics.UnitSetup{Name: "敵", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20, Mov: 2}, Pos: tactics.Pos{X: 0, Y: 1}, Weapon: testSword, AI: tactics.Hold}
			boss := tactics.UnitSetup{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 5, Y: 0}, Weapon: testSword, Boss: true, AI: tactics.Hold}
			return tactics.Setup{Terrain: field(6, 3), Units: []tactics.UnitSetup{{Name: "主人公", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 5, Y: 2}, Weapon: testSword, Lord: true}, armor, enemy, boss}}
		}
		c := newIdleWith(t, setup)
		c.ToggleDanger()
		if _, ok := c.Battle().Reachable(2)[tactics.Pos{X: 2, Y: 1}]; ok {
			t.Error("ZOC 内のマスへ到達した")
		}
		if c.Danger()[tactics.Pos{X: 3, Y: 1}] {
			t.Error("ZOC がなければ届く攻撃マスが危険範囲に入った")
		}
	})
}

package core_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 1 章の砦前の剣士は、一団が動き出しても弓兵から離れすぎない（TC-FR-MAP-04-28、DES-AI-02）。
func TestCh01FortFrontStaysTogether(t *testing.T) {
	setup, err := core.NewSetup(baseCatalog(t), "ch01")
	if err != nil {
		t.Fatal(err)
	}
	for i := range setup.Units {
		u := &setup.Units[i]
		if u.Group == "fort_front" {
			u.AI = tactics.Charge // 砦前の一団が動き出した直後。
		} else if u.Side == tactics.Enemy {
			// 別の敵の攻撃で再現が終わらないよう、位置だけ残す。
			u.Stats.Mov = 0
			for j := range u.Weapons {
				u.Weapons[j].MinRange, u.Weapons[j].MaxRange = 20, 20
			}
		}
		if u.ID == "gald" {
			u.Pos = tactics.Pos{X: 6, Y: 11}
		}
		if u.ID == "mira" {
			u.Pos = tactics.Pos{X: 5, Y: 11}
		}
	}
	setup.Reinforcements = nil
	sword, bow := -1, -1
	for i, u := range setup.Units {
		if u.ID == "anti_lure_ch01_2" {
			sword = i
		}
		if u.ID == "anti_lure_ch01_6" {
			bow = i
		}
	}
	if sword < 0 || bow < 0 {
		t.Fatalf("砦前の一団がない: 剣士 %d、弓兵 %d", sword, bow)
	}
	b, _, err := tactics.New(setup, 1)
	if err != nil {
		t.Fatal(err)
	}
	for turn := 0; turn < 3; turn++ {
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		units := b.Units()
		if !units[sword].Alive || !units[bow].Alive {
			t.Fatalf("%d 回目の敵軍の番で砦前の一団が欠けた", turn+1)
		}
		d := absInt(units[sword].Pos.X-units[bow].Pos.X) + absInt(units[sword].Pos.Y-units[bow].Pos.Y)
		if d > 7 {
			t.Fatalf("%d 回目の敵軍の番: 剣士 %v と弓兵 %v の距離 %d は 7 を超える", turn+1, units[sword].Pos, units[bow].Pos, d)
		}
		if turn < 2 {
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

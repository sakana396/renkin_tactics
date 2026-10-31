package core_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// core.NewBattle の自軍の置き方（DES-CORE-01 の反復 2 の変更）を確かめる。
// 自軍は game.toml の party の順に、マップの slots の数まで、slots の同じ順の座標に置く。主人公は枠の 1 番目。
func TestNewBattlePartySlots(t *testing.T) {
	t.Run("TC-FR-SYS-03-09 本体の 2 章・3 章では、名簿の 5 人が枠の先頭から名簿の順に置かれ、その後ろに敵が続く", func(t *testing.T) {
		cat := loadBase(t)
		tests := []struct {
			mapID   string
			pos     []tactics.Pos
			enemies int
		}{
			{"ch02", []tactics.Pos{{X: 1, Y: 7}, {X: 1, Y: 6}, {X: 1, Y: 8}, {X: 0, Y: 6}, {X: 0, Y: 8}}, 8},
			{"ch03", []tactics.Pos{{X: 7, Y: 14}, {X: 6, Y: 15}, {X: 8, Y: 15}, {X: 7, Y: 15}, {X: 8, Y: 14}}, 10},
		}
		for _, tt := range tests {
			b, _, err := core.NewBattle(cat, tt.mapID, 1)
			if err != nil {
				t.Fatalf("%s: NewBattle: %v", tt.mapID, err)
			}
			units := b.Units()
			if len(units) != len(tt.pos)+tt.enemies {
				t.Fatalf("%s: ユニットの数 = %d", tt.mapID, len(units))
			}
			var names []string
			for i, u := range units[:len(tt.pos)] {
				names = append(names, u.Name)
				if u.Side != tactics.Player || u.Pos != tt.pos[i] || u.HP != u.Stats.HP {
					t.Errorf("%s: units[%d] = %+v", tt.mapID, i, u)
				}
			}
			if want := []string{"レオン", "ガルド", "ブラン", "ミラ", "リナ"}; !reflect.DeepEqual(names, want) {
				t.Errorf("%s: 自軍 = %q", tt.mapID, names)
			}
			if !units[0].Lord {
				t.Errorf("%s: 枠の 1 番目が主人公でない", tt.mapID)
			}
			for _, u := range units[len(tt.pos):] {
				if u.Side != tactics.Enemy {
					t.Errorf("%s: 自軍の後ろに敵でないユニット %+v", tt.mapID, u)
				}
			}
		}
	})

	t.Run("TC-FR-SYS-03-10 主人公は名簿のどこにいても枠の 1 番目に置き、枠より多い名簿は枠の数で切る", func(t *testing.T) {
		chars := func(m *defs.MapDef, c *[]defs.CharacterDef) {
			*c = append(*c,
				defs.CharacterDef{ID: "a", Name: "エー", Class: "fighter", Weapons: []string{"iron_axe"}},
				defs.CharacterDef{ID: "b", Name: "ビー", Class: "fighter", Weapons: []string{"iron_axe"}},
			)
			m.Rows = []string{"...", ".森.", "..."}
		}
		tests := []struct {
			party []string
			slots []defs.Slot
			want  []string
			pos   []tactics.Pos
		}{
			// 主人公が名簿の 2 番目。枠は 2 つなので、b は出撃しない。
			{[]string{"a", "leon", "b"}, []defs.Slot{{X: 0, Y: 2}, {X: 1, Y: 2}}, []string{"レオン", "エー"}, []tactics.Pos{{X: 0, Y: 2}, {X: 1, Y: 2}}},
			// 主人公が名簿の最後。枠は 3 つで、名簿の 2 人がちょうど入り、3 番目の枠は空く。
			{[]string{"b", "leon"}, []defs.Slot{{X: 2, Y: 2}, {X: 0, Y: 2}, {X: 1, Y: 2}}, []string{"レオン", "ビー"}, []tactics.Pos{{X: 2, Y: 2}, {X: 0, Y: 2}}},
			// 枠が 1 つなら主人公だけ。
			{[]string{"a", "b", "leon"}, []defs.Slot{{X: 0, Y: 0}}, []string{"レオン"}, []tactics.Pos{{X: 0, Y: 0}}},
		}
		for _, tt := range tests {
			cat := smallCatalogParty(tt.party, func(m *defs.MapDef, c *[]defs.CharacterDef) {
				chars(m, c)
				m.Slots = tt.slots
			})
			b, _, err := core.NewBattle(cat, "m", 1)
			if err != nil {
				t.Fatalf("名簿 %q: NewBattle: %v", tt.party, err)
			}
			var names []string
			var pos []tactics.Pos
			for _, u := range b.Units() {
				if u.Side == tactics.Player {
					names = append(names, u.Name)
					pos = append(pos, u.Pos)
				}
			}
			if !reflect.DeepEqual(names, tt.want) || !reflect.DeepEqual(pos, tt.pos) {
				t.Errorf("名簿 %q: 自軍 %q、座標 %v、期待 %q、%v", tt.party, names, pos, tt.want, tt.pos)
			}
		}
	})
}

// 騎兵のクラスが、ほかのクラスと違う移動力を持つことを確かめる（FR-UNT-02、設計書（反復 2）10.1 節）。
func TestCavalier(t *testing.T) {
	t.Run("TC-FR-UNT-02-02 2 章の騎兵は槍を持ち、移動が 7（ほかのクラスより長い）", func(t *testing.T) {
		b, _, err := core.NewBattle(loadBase(t), "ch02", 1)
		if err != nil {
			t.Fatal(err)
		}
		cavaliers := 0
		for _, u := range b.Units() {
			if u.ClassName != "騎兵" {
				if u.Stats.Mov >= 7 {
					t.Errorf("%s（%s）の移動 = %d", u.Name, u.ClassName, u.Stats.Mov)
				}
				continue
			}
			cavaliers++
			if u.EquippedWeapon().Type != tactics.Lance || u.Stats.Mov != 7 {
				t.Errorf("%s の武器の種類 %v、移動 %d", u.Name, u.EquippedWeapon().Type, u.Stats.Mov)
			}
		}
		// 初めの敵の騎兵は、ボスのヴォルグを含めて 2 体（10.5 節）。
		if cavaliers != 2 {
			t.Errorf("騎兵の数 = %d", cavaliers)
		}
	})
}

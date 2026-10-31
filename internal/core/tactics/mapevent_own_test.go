package tactics

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// capBattle は 10 × 10 の平地に、生きている敵 live 体（ボスを含む）を置いた戦闘を作る（DES-IT06 9.3 節）。
// 2 ターン目のターンの増援 1 体（元 (9, 0)）と、出来事 e1 の増援 2 体（元 (5, 5)・(5, 6)）を持つ。
func capBattle(t *testing.T, live int) *Battle {
	t.Helper()
	weapon := Weapon{Name: "剣", Type: Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
	mk := func(name string, side Side, x, y int) UnitSetup {
		return UnitSetup{Name: name, Side: side, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 3}, Pos: Pos{x, y}, Weapon: weapon, AI: Hold}
	}
	terrain := make([][]Terrain, 10)
	for y := range terrain {
		terrain[y] = make([]Terrain, 10)
		for x := range terrain[y] {
			terrain[y][x] = Terrain{Name: "平地", Cost: 1}
		}
	}
	lord := mk("主人公", Player, 0, 0)
	lord.Lord = true
	boss := mk("ボス", Enemy, 9, 9)
	boss.Boss = true
	boss.ID = "boss"
	units := []UnitSetup{lord, boss}
	for i := 0; i < live-1; i++ {
		units = append(units, mk(fmt.Sprintf("敵 %d", i), Enemy, i%10, 1+i/10))
	}
	s := Setup{
		Terrain:        terrain,
		Units:          units,
		Reinforcements: []ReinforcementSetup{{Turn: 2, Unit: mk("ターン増援", Enemy, 9, 0)}},
		Events: []MapEventSetup{{ID: "e1", Trigger: TriggerEnemyAttacked, Unit: "boss", Effect: EffectReinforce,
			Enemies: []UnitSetup{mk("出来事増援 1", Enemy, 5, 5), mk("出来事増援 2", Enemy, 5, 6)}}},
	}
	b, _, err := New(s, 7)
	if err != nil {
		t.Fatal(err)
	}
	b.Turn = 2
	b.mapState.fired[0] = true
	b.mapState.reservations = []reservation{{Turn: 2, Event: 0}}
	return b
}

func TestEnemyCap(t *testing.T) {
	t.Run("TC-FR-MAP-03-19 生きている敵 39 体の上限で出ない個体は先送りせず乱数も引かない", func(t *testing.T) {
		cases := []struct {
			name   string
			live   int
			kill   int
			walls  bool
			wantTo []Pos
		}{
			{name: "37 体: ターンと出来事の 1 体目まで", live: 37, wantTo: []Pos{{9, 0}, {5, 5}}},
			{name: "38 → 39 の境界: ターンの増援だけ", live: 38, wantTo: []Pos{{9, 0}}},
			{name: "39 体: 出ない", live: 39},
			{name: "撃破で空いた 1 体分を補う", live: 39, kill: 1, wantTo: []Pos{{9, 0}}},
			{name: "空きがない出来事の増援は出ない", live: 10, walls: true, wantTo: []Pos{{9, 0}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				b := capBattle(t, tc.live)
				for i := 0; i < tc.kill; i++ {
					b.fall(nil, 2+i)
				}
				if tc.walls {
					for y := 3; y <= 8; y++ {
						for x := 3; x <= 7; x++ {
							b.terrain[y][x].Cost = 0
						}
					}
				}
				b.roll = func() int { t.Fatal("乱数を引いた"); return 0 }
				rng, _ := b.pcg.MarshalBinary()
				n := len(b.units)
				var events []Event
				events = b.placeReinforcements(events)
				events = b.placeEventReinforcements(events)
				var got []Pos
				for i, e := range events {
					if e.Kind != EventReinforce || e.Unit != n+i {
						t.Fatalf("出来事 %+v", e)
					}
					got = append(got, e.To)
				}
				if !reflect.DeepEqual(got, tc.wantTo) || len(b.units) != n+len(tc.wantTo) {
					t.Fatalf("出た場所 %v、期待 %v（%d 体）", got, tc.wantTo, len(b.units)-n)
				}
				if b.liveEnemies() > maxLiveEnemies {
					t.Fatalf("生きている敵 %d 体", b.liveEnemies())
				}
				after, _ := b.pcg.MarshalBinary()
				if !bytes.Equal(rng, after) {
					t.Fatal("乱数の状態が変わった")
				}
				// 出来事は起きたまま、予約は使ったものになり、出なかった個体を先送りしない。
				if !b.mapState.fired[0] || !b.mapState.reservations[0].Used {
					t.Fatalf("出来事の状態 %+v", b.mapState)
				}
				b.Turn = 3
				m := len(b.units)
				if events := b.placeEventReinforcements(nil); len(events) != 0 || len(b.units) != m {
					t.Fatalf("先送りした: %v", events)
				}
			})
		}
	})

	t.Run("TC-FR-MAP-03-18 出来事の状態の写しは本物から独立する", func(t *testing.T) {
		b := capBattle(t, 3)
		b.mapState.charged = map[string]bool{"gate": true}
		s := b.snapshot()
		b.mapState.fired[0] = false
		b.mapState.reservations[0].Used = true
		b.mapState.charged["gate"] = false
		if !s.mapState.fired[0] || s.mapState.reservations[0].Used || !s.mapState.charged["gate"] {
			t.Fatalf("写しが本物に引きずられた: %+v", s.mapState)
		}
		if err := b.restore(s); err != nil {
			t.Fatal(err)
		}
		s.mapState.fired[0] = false
		s.mapState.charged["gate"] = false
		if !b.mapState.fired[0] || !b.mapState.charged["gate"] || b.mapState.reservations[0].Used {
			t.Fatalf("復元した本物が写しに引きずられた: %+v", b.mapState)
		}
	})
	t.Run("TC-FR-MAP-03-18 増援の定義は入力の特効の分類・武器の列・手持ちから独立する", func(t *testing.T) {
		// REV-068 の申し送り: #120 が UnitSetup に足したスライスの欄も、出来事の増援とターンの増援で深く写す。
		weapon := Weapon{Name: "槍", Type: Lance, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"mounted"}}
		mk := func(name string, side Side, x, y int) UnitSetup {
			return UnitSetup{Name: name, Side: side, Stats: Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: 3}, Pos: Pos{x, y}, Weapon: weapon.clone(), AI: Hold,
				Tags: []string{"armor"}, Weapons: []Weapon{weapon.clone()}, Items: []ItemStack{{Item: Item{Name: "薬", Heal: 10}, Count: 1}}}
		}
		lord := mk("主人公", Player, 0, 0)
		lord.Lord = true
		boss := mk("ボス", Enemy, 7, 7)
		boss.Boss = true
		boss.ID = "boss"
		terrain := make([][]Terrain, 8)
		for y := range terrain {
			terrain[y] = make([]Terrain, 8)
			for x := range terrain[y] {
				terrain[y][x] = Terrain{Name: "平地", Cost: 1}
			}
		}
		s := Setup{
			Terrain:        terrain,
			Units:          []UnitSetup{lord, boss},
			Reinforcements: []ReinforcementSetup{{Turn: 2, Unit: mk("ターン増援", Enemy, 7, 0)}},
			Events: []MapEventSetup{{ID: "e1", Trigger: TriggerEnemyAttacked, Unit: "boss", Effect: EffectReinforce,
				Enemies: []UnitSetup{mk("出来事増援", Enemy, 5, 5)}}},
		}
		b, _, err := New(s, 7)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range []UnitSetup{s.Events[0].Enemies[0], s.Reinforcements[0].Unit} {
			u.Tags[0] = "changed"
			u.Weapon.EffectiveTags[0] = "changed"
			u.Weapons[0].EffectiveTags[0] = "changed"
			u.Items[0].Count = 9
		}
		for name, u := range map[string]UnitSetup{"出来事の増援": b.mapEvents[0].Enemies[0], "ターンの増援": b.reinforcements[0].Unit} {
			if u.Tags[0] != "armor" || u.Weapon.EffectiveTags[0] != "mounted" || u.Weapons[0].EffectiveTags[0] != "mounted" || u.Items[0].Count != 1 {
				t.Errorf("%s が入力に引きずられた: %+v", name, u)
			}
		}
	})
}

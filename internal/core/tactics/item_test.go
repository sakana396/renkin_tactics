package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// アイテムの使用（DES-BTL-04、DES-ITM-01）のテスト。

var (
	potion = tactics.Item{Name: "回復薬", Heal: 10}
	elixir = tactics.Item{Name: "霊薬", Heal: 20}
)

// heal は Heal の出来事の比べる部分。
type heal struct {
	Unit, Amount int
	Item         string
}

// heals は出来事の列から Heal だけを取り出す。
func heals(events []tactics.Event) []heal {
	var out []heal
	for _, e := range events {
		if e.Kind == tactics.EventHeal {
			out = append(out, heal{e.Unit, e.Amount, e.Item})
		}
	}
	return out
}

// TestUseItem は、移動してからアイテムを使う流れを確かめる（DES-BTL-04、DES-ITM-01）。
func TestUseItem(t *testing.T) {
	t.Run("TC-FR-BTL-03-07 移動してからアイテムを使い、全員が行動したら敵軍の番に進む", func(t *testing.T) {
		b := mustNewStill(t, baseSetup())
		b.SetHP(0, 5)
		b.SetHP(1, 19)
		noRolls(t, b)

		events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.UseItem, Item: 0})
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{tactics.EventMove, tactics.EventHeal})
		assertEqual(t, "Heal", heals(events), []heal{{0, 10, "回復薬"}})
		mv, hl := events[0].Units[0], events[1].Units[0]
		if mv.Pos != pos(1, 1) || mv.Acted || mv.HP != 5 {
			t.Errorf("Move の写し: Pos %v、Acted %v、HP %d", mv.Pos, mv.Acted, mv.HP)
		}
		if !hl.Acted || hl.HP != 15 || hl.Items[0].Count != 1 {
			t.Errorf("Heal の写し: Acted %v、HP %d、手持ち %+v", hl.Acted, hl.HP, hl.Items)
		}
		assertEqual(t, "番", b.Phase, tactics.Player)

		// 移動しないで使う。最後の自軍なので、敵軍の番を経て次の自軍の番まで進む。
		events = mustAct(t, b, tactics.Command{Unit: 1, To: pos(0, 4), Action: tactics.UseItem, Item: 0})
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{
			tactics.EventHeal, tactics.EventPhaseStart, tactics.EventPhaseStart,
		})
		assertEqual(t, "Heal", heals(events), []heal{{1, 1, "回復薬"}})
		assertEqual(t, "ターン", b.Turn, 2)
		assertEqual(t, "番", b.Phase, tactics.Player)
	})

	t.Run("TC-FR-ITM-02-01 回復は最大 HP を超えず、手持ちを 1 減らす", func(t *testing.T) {
		cases := []struct {
			name       string
			hp         int
			wantAmount int
			wantHP     int
		}{
			{"HP 5 なら 10 回復", 5, 10, 15},
			{"HP 10 ならちょうど最大まで 10 回復", 10, 10, 20},
			{"HP 15 なら最大までの 5 だけ回復", 15, 5, 20},
			{"HP 19 なら 1 だけ回復", 19, 1, 20},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				b := mustNew(t, baseSetup())
				b.SetHP(0, tt.hp)
				events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: 0})
				assertEqual(t, "Heal", heals(events), []heal{{0, tt.wantAmount, "回復薬"}})
				u := b.Units()[0]
				assertEqual(t, "HP", u.HP, tt.wantHP)
				assertEqual(t, "手持ち", u.Items, []tactics.ItemStack{{Item: potion, Count: 1}})
				assertEqual(t, "行動済み", u.Acted, true)
			})
		}
	})

	t.Run("TC-FR-ITM-02-02 数が 0 になったアイテムは手持ちから消える", func(t *testing.T) {
		cases := []struct {
			name  string
			items []tactics.ItemStack
			item  int
			want  []tactics.ItemStack
		}{
			{"1 つだけのアイテムを使い切ると手持ちが空", []tactics.ItemStack{{Item: potion, Count: 1}}, 0, nil},
			{"先頭を使い切ると後ろが詰まる",
				[]tactics.ItemStack{{Item: potion, Count: 1}, {Item: elixir, Count: 2}}, 0,
				[]tactics.ItemStack{{Item: elixir, Count: 2}}},
			{"末尾を使い切る",
				[]tactics.ItemStack{{Item: potion, Count: 2}, {Item: elixir, Count: 1}}, 1,
				[]tactics.ItemStack{{Item: potion, Count: 2}}},
			{"2 つ目を 1 減らす",
				[]tactics.ItemStack{{Item: potion, Count: 1}, {Item: elixir, Count: 2}}, 1,
				[]tactics.ItemStack{{Item: potion, Count: 1}, {Item: elixir, Count: 1}}},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				s := baseSetup()
				s.Units[0].Items = tt.items
				given := append([]tactics.ItemStack(nil), tt.items...)
				b := mustNew(t, s)
				b.SetHP(0, 1)
				before := b.Units()
				events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: tt.item})
				assertEqual(t, "手持ち", b.Units()[0].Items, tt.want)
				assertEqual(t, "Heal の写しの手持ち", events[0].Units[0].Items, tt.want)
				// 使う前の写しと、渡した Setup は変わらない。
				assertEqual(t, "使う前の写し", before[0].Items, given)
				assertEqual(t, "Setup", s.Units[0].Items, given)
			})
		}
	})

	t.Run("TC-FR-ITM-02-03 / TC-NFR-07-04 アイテムがないか HP が最大なら、状態を変えずに誤りを返す", func(t *testing.T) {
		cases := []struct {
			name    string
			prepare func(s *tactics.Setup)
			hp      int
			item    int
			want    error
		}{
			{"Item が -1", nil, 5, -1, tactics.ErrNoItem},
			{"Item が数を超える", nil, 5, 1, tactics.ErrNoItem},
			{"手持ちが空", func(s *tactics.Setup) { s.Units[0].Items = nil }, 5, 0, tactics.ErrNoItem},
			{"HP が最大", nil, 20, 0, tactics.ErrFullHP},
			{"アイテムがなく HP も最大なら ErrNoItem が先", nil, 20, 1, tactics.ErrNoItem},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				s := baseSetup()
				if tt.prepare != nil {
					tt.prepare(&s)
				}
				b := mustNew(t, s)
				b.SetHP(0, tt.hp)
				noRolls(t, b)
				before := snapshot(b)
				// 移動できるマスを To にしても、誤りなら移動しない。
				events, err := b.Act(tactics.Command{Unit: 0, To: pos(1, 1), Action: tactics.UseItem, Item: tt.item})
				if !errors.Is(err, tt.want) {
					t.Errorf("err = %v, want %v", err, tt.want)
				}
				if events != nil {
					t.Errorf("出来事 = %v, want nil", events)
				}
				assertEqual(t, "状態", snapshot(b), before)
			})
		}
	})

	t.Run("TC-FR-ITM-03-02 初めから持つ回復薬を使い切ると、手持ちから消えて使えなくなる", func(t *testing.T) {
		b := mustNewStill(t, baseSetup())
		for turn := 1; turn <= 2; turn++ {
			b.SetHP(0, 5)
			mustAct(t, b, tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: 0})
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
		}
		assertEqual(t, "手持ち", b.Units()[0].Items, []tactics.ItemStack(nil))
		b.SetHP(0, 5)
		if _, err := b.Act(tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: 0}); !errors.Is(err, tactics.ErrNoItem) {
			t.Errorf("err = %v, want ErrNoItem", err)
		}
	})
}

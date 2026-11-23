package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// dangerCacheSetup は、粘着瓶を持つ自軍 2 体、動く敵と動かない敵、2 ターン目の増援を置いた盤面。
func dangerCacheSetup() tactics.Setup {
	snare := tactics.Item{Name: "粘着瓶", Effect: "snare", Range: 4}
	return tactics.Setup{Terrain: field(14, 6),
		Units: []tactics.UnitSetup{
			{Name: "主人公", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 4}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testLance, Items: []tactics.ItemStack{{Item: snare, Count: 2}}},
			{Name: "仲間", Side: tactics.Player, Stats: tactics.Stats{HP: 30, Mov: 4}, Pos: tactics.Pos{X: 0, Y: 2}, Weapon: testLance},
			{Name: "動く敵", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 4, Y: 0}, Weapon: testLance, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Boss: true, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 13, Y: 5}, Weapon: testLance, AI: tactics.Hold},
		},
		Reinforcements: []tactics.ReinforcementSetup{{Turn: 2, Unit: tactics.UnitSetup{Name: "増援", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Mov: 2}, Pos: tactics.Pos{X: 9, Y: 0}, Weapon: testLance, AI: tactics.Hold}}},
	}
}

// 覚えた危険範囲が、盤面の変化のあとに古くならない（#227）。
func TestDangerCache227(t *testing.T) {
	t.Run("TC-NFR-01-05 変化のたびに計算し直した結果と同じ・変化がなければ覚えた結果を返す", func(t *testing.T) {
		c := newIdleWith(t, dangerCacheSetup)
		c.ToggleDanger()
		b := c.Battle()
		last := c.Danger()
		same := func(a, b map[tactics.Pos]bool) bool {
			return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
		}
		// 変化のない間は、同じ地図を返す（計算し直さない）。
		if !same(last, c.Danger()) {
			t.Fatal("変化がないのに計算し直した")
		}
		// check は、覚えた結果が計算し直した結果と同じで、changed なら前と違う（古くない）ことを確かめる。
		check := func(what string, changed bool) {
			t.Helper()
			got := c.Danger()
			if want := c.computeDanger(); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: 覚えた結果が古い got=%d マス want=%d マス", what, len(got), len(want))
			}
			if changed && reflect.DeepEqual(got, last) {
				t.Fatalf("%s: 危険範囲が変わるはずなのに変わらない", what)
			}
			last = got
		}
		// 粘着瓶で動く敵を止める。移動 0 になり、危険範囲が縮む。
		p := tactics.Pos{X: 4, Y: 0}
		if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
			t.Fatal(err)
		}
		if !b.Units()[2].Snared {
			t.Fatalf("粘着が付かない %+v", b.Units()[2])
		}
		check("粘着瓶", true)
		// 番を進める。2 ターン目の頭で粘着が切れ、増援が出る。
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if b.Units()[2].Snared || b.Turn != 2 {
			t.Fatalf("粘着の期限切れが起きない: snared=%v ターン=%d", b.Units()[2].Snared, b.Turn)
		}
		check("番の切り替え（期限切れ）", true)
		// 増援が出るまで番を進める。出たところで危険範囲が変わる。
		for i := 0; i < 3 && len(b.Units()) < 5; i++ {
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
			check("番の切り替え", len(b.Units()) == 5)
		}
		if len(b.Units()) != 5 {
			t.Fatalf("増援が出ない: %d 体", len(b.Units()))
		}
		// 自軍の移動（位置が変わるので印が変わる。危険範囲は自軍の位置に左右されない）。
		if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 1}, Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		check("自軍の移動", false)
		// 巻き戻し。
		h := b.History()
		if len(h) == 0 {
			t.Fatal("履歴がない")
		}
		if err := b.Rewind(h[0].ID); err != nil {
			t.Fatal(err)
		}
		check("巻き戻し", true)
		if len(b.Units()) != 4 {
			t.Fatalf("巻き戻しで増援が消えていない: %d 体", len(b.Units()))
		}
		// 敵が倒れる・切り替え。Q を切ると nil、また出すと正しい結果。
		c.ToggleDanger()
		if c.Danger() != nil {
			t.Fatal("Q を切っても出ている")
		}
		c.ToggleDanger()
		check("Q を出し直す", false)
	})
}

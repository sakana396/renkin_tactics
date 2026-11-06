package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 同じフレームの危険範囲を共有しても、撃破・巻き戻し・Qの表示を変えない（DES-UI-29、DES-UI-28）。
func TestIT07SharedFrameRanges(t *testing.T) {
	t.Run("TC-NFR-01-04", func(t *testing.T) {
		c := newIdleWith(t, killSetup)
		c.ToggleDanger()
		c.watched = map[int]bool{2: true}
		c.SetCursor(cellCenter(tactics.Pos{X: 1}))
		boss := map[tactics.Pos]bool{{X: 4, Y: 5}: true, {X: 5, Y: 4}: true}
		both := map[tactics.Pos]bool{{X: 0, Y: 0}: true, {X: 2, Y: 0}: true, {X: 1, Y: 1}: true, {X: 4, Y: 5}: true, {X: 5, Y: 4}: true}
		check := func(want map[tactics.Pos]bool, watched bool) {
			t.Helper()
			danger := c.Danger()
			if len(danger) != len(want) {
				t.Fatalf("全敵範囲=%v,期待%v", danger, want)
			}
			for p := range want {
				if !danger[p] {
					t.Fatalf("全敵範囲に%vがない", p)
				}
			}
			layers := c.enemyRangeLayersWithDanger(danger)
			if !reflect.DeepEqual(layers, c.EnemyRangeLayers()) {
				t.Fatal("範囲の層・順番が変わった")
			}
			if watched {
				if len(layers) != 5 || layers[0].Owner != RangeAll || layers[1].Owner != RangeWatched || layers[3].Owner != RangeCursor {
					t.Fatalf("層の順序=%v", layers)
				}
				if !reflect.DeepEqual(layers[1].Cells, map[tactics.Pos]bool{{X: 1, Y: 0}: true}) {
					t.Fatal("固定の移動範囲が変わった")
				}
			}
			v := c.minimapWithDanger(danger)
			if !reflect.DeepEqual(v, c.Minimap()) {
				t.Fatal("ミニマップの危険範囲・ピン等が変わった")
			}
			if len(v.Danger) != len(want) {
				t.Fatalf("ミニマップの危険マス=%d,期待%d", len(v.Danger), len(want))
			}
			for _, rect := range v.Danger {
				found := false
				for p := range want {
					if rect == v.Layout.CellRect(p) {
						found = true
					}
				}
				if !found {
					t.Fatalf("古い危険範囲が残った: %v", rect)
				}
			}
		}
		check(both, true)
		c.Click(tactics.Pos{})
		c.Click(tactics.Pos{})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1})
		c.Click(tactics.Pos{X: 1})
		playOut(c)
		if c.Battle().Units()[2].Alive {
			t.Fatal("弱い敵が倒れていない")
		}
		check(boss, false)
		c.OpenRewind()
		c.RewindMove(-100)
		c.RewindEnter()
		c.RewindEnter()
		if !c.Battle().Units()[2].Alive {
			t.Fatal("巻き戻しで敵が戻っていない")
		}
		c.SetCursor(cellCenter(tactics.Pos{X: 1}))
		check(both, false)
		c.ToggleDanger()
		check(nil, false)
	})
}

// 測定の準備で負荷や表示条件を落としていないことを確かめる（DES-IT07 9.1節）。
func TestIT07PerformanceInputs(t *testing.T) {
	for _, tc := range []struct {
		chapter          string
		players, enemies int
	}{{"", 7, 39}, {"ch01", 5, 9}, {"ch09", 7, 29}} {
		t.Run("TC-NFR-01-03 "+tc.chapter, func(t *testing.T) {
			t.Setenv("RENKIN_IT07_FPS_CH", tc.chapter)
			c, _, err := newIT07PerformanceController()
			if err != nil {
				t.Fatal(err)
			}
			p, e := 0, 0
			for _, u := range c.Units() {
				if u.Side == tactics.Player {
					p++
				} else {
					e++
				}
				if tc.chapter == "" {
					zoc := false
					for _, sk := range u.ClassSkills {
						zoc = zoc || sk.Effect == tactics.ZOC
					}
					if !zoc || u.Stats.Mov != 7 {
						t.Fatal("合成盤面のZOC/移動力が変わった")
					}
				}
			}
			if p != tc.players || e != tc.enemies || c.WatchedCount() != 2 || c.shownEnemy() != tc.players || !c.danger {
				t.Fatalf("測定条件が違う: 自軍%d/敵%d 固定%d カーソル%d 全敵%v", p, e, c.WatchedCount(), c.shownEnemy(), c.danger)
			}
		})
	}
}

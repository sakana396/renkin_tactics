package ui

import (
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// newAlchemyDangerController は新作の章の初期盤面で、Q（全敵の危険範囲）を出した Controller を作る（#227）。
func newAlchemyDangerController(tb testing.TB, chapter string) *Controller {
	tb.Helper()
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		tb.Fatal(err)
	}
	campaign, err := core.NewCampaignAt(cat, core.Withdraw, chapter)
	if err != nil {
		tb.Fatal(err)
	}
	mapDef, ok := cat.Map(chapter)
	if !ok {
		tb.Fatalf("章 %s のマップがない", chapter)
	}
	members := []int{}
	for i, m := range campaign.Roster {
		if i == 0 || m.Alive {
			members = append(members, i)
			if len(members) == len(mapDef.Slots) {
				break
			}
		}
	}
	c, err := NewController(func() (*tactics.Battle, []tactics.Event, error) {
		sortie, events, err := campaign.StartBattle(members, 1)
		if err != nil {
			return nil, nil, err
		}
		return sortie.Battle, events, nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	playOut(c)
	c.ToggleDanger()
	if len(c.Danger()) == 0 {
		tb.Fatal("危険範囲が空")
	}
	return c
}

// 1 フレームの範囲の計算（Game.Draw と同じ組み立て）を、章ごとに測る（#227、DES-UI-29）。
// 盤面が変わらない連続フレームを測る。Uncached は、覚えずに毎回計算し直す直す前の動き。
func BenchmarkDanger227Frame(b *testing.B) {
	for _, ch := range []string{"ch01", "ch03"} {
		b.Run(fmt.Sprintf("%s/Cached", ch), func(b *testing.B) {
			c := newAlchemyDangerController(b, ch)
			b.Logf("units=%d danger_cells=%d", len(c.Units()), len(c.Danger()))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				danger := c.Danger()
				_ = c.enemyRangeLayersWithDanger(danger)
				_ = c.minimapWithDanger(danger)
			}
		})
		b.Run(fmt.Sprintf("%s/Uncached", ch), func(b *testing.B) {
			c := newAlchemyDangerController(b, ch)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				danger := c.computeDanger()
				_ = c.enemyRangeLayersWithDanger(danger)
				_ = c.minimapWithDanger(danger)
			}
		})
	}
}

// 新作の章で、覚えた危険範囲が計算し直した結果と同じこと（#227）。
func TestDangerCache227Alchemy(t *testing.T) {
	for _, ch := range []string{"ch01", "ch02", "ch03"} {
		t.Run("TC-NFR-01-06 新作の章の初期盤面 "+ch, func(t *testing.T) {
			c := newAlchemyDangerController(t, ch)
			if got, want := c.Danger(), c.computeDanger(); len(got) != len(want) {
				t.Fatalf("%d マス、期待 %d マス", len(got), len(want))
			}
		})
	}
}

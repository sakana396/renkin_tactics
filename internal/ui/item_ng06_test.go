package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func itemNG06Setup() tactics.Setup {
	s := bombSetup()
	s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{ID: "tonic", Name: "剛力薬", Effect: "power_tonic", Amount: 4, Range: 3}, Count: 2}, {Item: tactics.Item{ID: "snare", Name: "粘着瓶", Effect: "snare", Range: 4}, Count: 1}}
	s.Units[1].Stats.Str, s.Units[1].Stats.Mag = 7, 2
	return s
}

func TestNG06ItemUI(t *testing.T) {
	t.Run("TC-DES-NG-16-01 対象と予想", func(t *testing.T) {
		c := newIdleWith(t, itemNG06Setup)
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.ChooseAction(tactics.UseItem)
		labels := c.MenuLabels()
		if len(labels) != 2 || !strings.Contains(labels[0], "力・魔力 +4") || !strings.Contains(labels[1], "距離 1〜4") {
			t.Fatalf("品目 %v", labels)
		}
		c.ChooseItem(0)
		if c.State() != ItemTargeting || !reflect.DeepEqual(c.itemTargets(0), []int{0, 1}) {
			t.Fatalf("候補 %v", c.itemTargets(0))
		}
		c.Click(tactics.Pos{X: 4, Y: 2})
		guide := strings.Join(c.itemGuide(), " ")
		if !strings.Contains(guide, "力 7→11") || !strings.Contains(guide, "魔力 2→6") || !strings.Contains(guide, "次の自軍開始") {
			t.Fatalf("予想 %s", guide)
		}
	})
	t.Run("TC-DES-NG-16-02 対象変更と取消し", func(t *testing.T) {
		c := newIdleWith(t, itemNG06Setup)
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.ChooseAction(tactics.UseItem)
		c.ChooseItem(0)
		before := c.Battle().Units()
		c.Click(tactics.Pos{X: 4, Y: 2})
		c.Click(tactics.Pos{X: 2, Y: 2})
		c.Cancel()
		c.Cancel()
		if c.State() != ItemMenu || !reflect.DeepEqual(before, c.Battle().Units()) {
			t.Fatal("取消しで状態が変化")
		}
		c.ChooseItem(0)
		c.Click(tactics.Pos{X: 4, Y: 2})
		c.Click(tactics.Pos{X: 4, Y: 2})
		units := c.Battle().Units()
		if units[0].Items[0].Count != 1 || units[1].PowerTonic != 4 {
			t.Fatalf("使用結果 %+v %+v", units[0].Items, units[1])
		}
	})
}

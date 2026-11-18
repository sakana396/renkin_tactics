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
	t.Run("TC-DES-NG-16-01 効果中と射程外は候補外", func(t *testing.T) {
		c := newIdleWith(t, func() tactics.Setup { s := itemNG06Setup(); s.Units[1].Items = s.Units[0].Items; return s })
		p := tactics.Pos{X: 4, Y: 2}
		if _, err := c.battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 2}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
			t.Fatal(err)
		}
		c.unit, c.dest = 1, p
		// 本人に効果が残るため候補にならず、射程外の敵も候補にならない。
		if got := c.itemTargets(0); !reflect.DeepEqual(got, []int{0}) {
			t.Fatalf("候補 %v", got)
		}
	})

	t.Run("TC-DES-NG-16-02 攻撃予想と危険範囲", func(t *testing.T) {
		setup := func() tactics.Setup { s := itemNG06Setup(); s.Units[3].Stats.Mov = 3; return s }
		c := newIdleWith(t, setup)
		c.danger = true
		beforeDanger := c.Danger()
		cmd := tactics.Command{Unit: 1, To: tactics.Pos{X: 4, Y: 2}, Action: tactics.Attack, Target: 3}
		before, err := c.battle.Forecast(cmd)
		if err != nil {
			t.Fatal(err)
		}
		p := tactics.Pos{X: 4, Y: 2}
		if _, err = c.battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 2}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
			t.Fatal(err)
		}
		after, err := c.battle.Forecast(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if after.Attacker.Damage != before.Attacker.Damage+4 {
			t.Fatalf("攻撃予想 %d→%d", before.Attacker.Damage, after.Attacker.Damage)
		}
		c2 := newIdleWith(t, setup)
		c2.danger = true
		enemy := tactics.Pos{X: 5, Y: 2}
		if _, err := c2.battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 2}, Action: tactics.UseItem, Item: 1, ItemTarget: &enemy}); err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(beforeDanger, c2.Danger()) {
			t.Fatal("危険範囲が変わらない")
		}
	})

}

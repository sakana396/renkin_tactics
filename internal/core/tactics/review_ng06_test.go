package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// レビュー用: 粘着した突撃の敵は敵軍の番に動かず、隣の自軍は攻撃する。
func TestNG06SnareEnemyPhase(t *testing.T) {
	for _, snared := range []bool{false, true} {
		s := baseSetup()
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "瓶", Effect: "snare", Range: 4}, Count: 1}}
		s.Units[3].Stats.Mov = 5
		b := mustNew(t, s)
		if snared {
			p := tactics.Pos{X: 4, Y: 0}
			if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
				t.Fatal(err)
			}
		}
		ev, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		moved := false
		for _, e := range ev {
			if e.Kind == tactics.EventMove && e.Unit == 3 {
				moved = true
			}
		}
		t.Logf("snared=%v moved=%v pos=%v", snared, moved, b.Units()[3].Pos)
		if snared && moved {
			t.Fatal("粘着した敵が動いた")
		}
		if !snared && !moved {
			t.Fatal("対照: 敵が動かない")
		}
	}
}

// レビュー用: 粘着した敵は隣の自軍を攻撃できる。
func TestNG06SnareAttackInPlace(t *testing.T) {
	s := baseSetup()
	s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "瓶", Effect: "snare", Range: 4}, Count: 1}}
	s.Units[1].Pos = tactics.Pos{X: 4, Y: 1}
	s.Units[3].ClassSkills = []tactics.Skill{{Effect: tactics.Canto}}
	b := mustNew(t, s)
	p := tactics.Pos{X: 4, Y: 0}
	if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
		t.Fatal(err)
	}
	ev, err := b.EndPhase()
	if err != nil {
		t.Fatal(err)
	}
	struck := false
	for _, e := range ev {
		if e.Kind == tactics.EventStrike && e.Unit == 3 {
			struck = true
		}
		if e.Kind == tactics.EventMove && e.Unit == 3 {
			t.Fatal("動いた")
		}
	}
	if !struck {
		t.Fatal("その場の攻撃をしない")
	}
}

// レビュー用: 剛力薬は敵軍の番の反撃に効く。
func TestNG06TonicCounter(t *testing.T) {
	dmg := map[bool]int{}
	for _, tonic := range []bool{false, true} {
		s := baseSetup()
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "薬", Effect: "power_tonic", Amount: 4, Range: 3}, Count: 1}}
		s.Units[1].Pos = tactics.Pos{X: 4, Y: 1}
		s.Units[1].Stats.HP = 60
		s.Units[1].Items = s.Units[0].Items
		s.Units[3].Stats.Mov = 0
		b := mustNew(t, s)
		b.SetRoll(func() int { return 50 })
		if tonic {
			p := tactics.Pos{X: 4, Y: 1}
			if _, err := b.Act(tactics.Command{Unit: 1, To: p, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err != nil {
				t.Fatal(err)
			}
		}
		ev, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range ev {
			if e.Kind == tactics.EventStrike && e.Unit == 1 && e.Hit {
				dmg[tonic] = e.Amount
				break
			}
		}
	}
	t.Logf("%v", dmg)
	if dmg[true] != dmg[false]+4 {
		t.Fatalf("反撃 %v", dmg)
	}
}

// レビュー用: 粘着瓶は味方に使えない。
func TestNG06SnareAlly(t *testing.T) {
	s := baseSetup()
	s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "瓶", Effect: "snare", Range: 4}, Count: 1}}
	b := mustNew(t, s)
	p := tactics.Pos{X: 0, Y: 4}
	if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 0}, Action: tactics.UseItem, Item: 0, ItemTarget: &p}); err == nil {
		t.Fatal("味方に粘着瓶を使えた")
	}
}

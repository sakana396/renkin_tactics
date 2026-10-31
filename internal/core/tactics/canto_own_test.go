package tactics

import "testing"

// 待っている再移動の写しの内部の試験（DES-BTL-10、DES-RWD-01、#124）。

func TestSnapshotCanto(t *testing.T) {
	t.Run("TC-FR-BTL-10-31 待っている再移動と兵種のスキルを写しに含め、復元で戻す", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[0].ClassSkills = []Skill{{Name: "再移動", Effect: Canto}}
		want := PendingCanto{Unit: 0, Remaining: 3, HistoryID: 7}
		b.canto = cantoState{active: true, PendingCanto: want}
		s := b.snapshot()
		b.canto = cantoState{}
		b.units[0].ClassSkills[0].Effect = Wrath
		if s.units[0].ClassSkills[0].Effect != Canto {
			t.Fatal("写しの兵種のスキルが本物と共有されている")
		}
		if err := b.restore(s); err != nil {
			t.Fatal(err)
		}
		if got, ok := b.PendingCanto(); !ok || got != want {
			t.Errorf("復元した再移動 = %+v, %v, want %+v", got, ok, want)
		}
		if b.units[0].ClassSkills[0].Effect != Canto {
			t.Error("兵種のスキルが復元されない")
		}
	})
}

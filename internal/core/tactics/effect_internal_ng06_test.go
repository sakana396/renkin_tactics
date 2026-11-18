package tactics

import (
	"reflect"
	"testing"
)

func TestEffectLifecycleNG06(t *testing.T) {
	t.Run("TC-DES-NG-14-02 反撃と自軍開始", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[1].Pos = Pos{1, 0}
		b.units[1].PowerTonic = 4
		base, _ := b.strikeValues(1, 2, false)
		b.units[1].PowerTonic = 0
		without, _ := b.strikeValues(1, 2, false)
		if base != without+4 {
			t.Fatalf("反撃 %d %d", base, without)
		}
		b.units[1].PowerTonic = 4
		ev := b.startPlayerPhase(nil)
		if b.units[1].PowerTonic != 0 || len(ev) < 2 || ev[0].Kind != EventEffectExpire || ev[1].Kind != EventPhaseStart {
			t.Fatalf("開始 %+v", ev)
		}
	})
	t.Run("TC-DES-NG-14-06 撃破と戦闘終了", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[1].PowerTonic = 4
		b.fall(nil, 1)
		if b.units[1].PowerTonic != 0 {
			t.Fatal("倒れた対象に効果が残る")
		}
		b.units[0].PowerTonic = 4
		b.units[2].Snared = true
		b.fall(nil, 2)
		b.judge(nil)
		if b.Result != Won || b.units[0].PowerTonic != 0 || b.units[2].Snared {
			t.Fatal("戦闘終了時に残った")
		}
	})
	t.Run("TC-DES-NG-14-04 重複・死亡と乱数", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[0].Items = []ItemStack{{Item: Item{Name: "薬", Effect: "power_tonic", Amount: 4, Range: 3}, Count: 2}}
		p := b.units[1].Pos
		cmd := Command{Unit: 0, To: b.units[0].Pos, Action: UseItem, ItemTarget: &p}
		b.units[1].PowerTonic = 4
		before := b.snapshot()
		history := b.History()
		if _, err := b.Act(cmd); err != ErrEffectActive {
			t.Fatalf("重複: %v", err)
		}
		after := b.snapshot()
		if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(history, b.History()) {
			t.Fatal("失敗時に状態が変わった")
		}
		b.fall(nil, 1)
		before = b.snapshot()
		if _, err := b.Act(cmd); err == nil {
			t.Fatal("死亡対象に使えた")
		}
		if !reflect.DeepEqual(before, b.snapshot()) {
			t.Fatal("死亡対象で状態が変わった")
		}
	})
	t.Run("TC-DES-NG-15-01 期限後の巻き戻し", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[0].Items = []ItemStack{{Item: Item{Name: "薬", Effect: "power_tonic", Amount: 4, Range: 3}, Count: 2}}
		p := b.units[1].Pos
		cmd := Command{Unit: 0, To: b.units[0].Pos, Action: UseItem, ItemTarget: &p}
		if _, err := b.Act(cmd); err != nil {
			t.Fatal(err)
		}
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if b.units[1].PowerTonic != 0 {
			t.Fatal("期限切れなし")
		}
		h := b.History()
		if err := b.Rewind(h[1].ID); err != nil {
			t.Fatal(err)
		}
		if b.units[1].PowerTonic != 4 || b.units[0].Items[0].Count != 1 {
			t.Fatal("期限前を復元できない")
		}
	})
}

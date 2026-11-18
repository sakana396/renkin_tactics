package tactics_test

import (
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"reflect"
	"testing"
)

func effectSetup(effect string) tactics.Setup {
	s := baseSetup()
	it := tactics.Item{Name: "道具", Effect: effect, Range: 4}
	if effect == "power_tonic" {
		it.Amount = 4
		it.Range = 3
	}
	s.Units[0].Items = []tactics.ItemStack{{Item: it, Count: 2}}
	s.Units[1].Pos = pos(0, 3)
	s.Units[1].Stats.Str = 7
	s.Units[1].Stats.Mag = 2
	return s
}
func effectCmd(p tactics.Pos) tactics.Command {
	return tactics.Command{Unit: 0, To: pos(0, 0), Action: tactics.UseItem, Item: 0, ItemTarget: &p}
}
func TestEffectItems(t *testing.T) {
	t.Run("TC-DES-NG-14-01 効果と消費", func(t *testing.T) {
		b := mustNewStill(t, effectSetup("power_tonic"))
		noRolls(t, b)
		cmd := effectCmd(pos(0, 3))
		f, err := b.ItemForecast(cmd)
		if err != nil || f.After.Str != 11 || f.After.Mag != 6 {
			t.Fatalf("予想 %+v %v", f, err)
		}
		ev := mustAct(t, b, cmd)
		u := b.Units()
		if u[1].Stats.Str != 7 || u[1].EffectiveStats().Str != 11 || u[1].EffectiveStats().Mag != 6 || u[0].Items[0].Count != 1 || !u[0].Acted || u[1].HP != 20 {
			t.Fatalf("状態 %+v", u)
		}
		if len(ev) != 2 || ev[0].Kind != tactics.EventItemUse || ev[1].Kind != tactics.EventEffectApply {
			t.Fatalf("出来事 %+v", ev)
		}
	})
	t.Run("TC-DES-NG-14-03 粘着と期限", func(t *testing.T) {
		s := effectSetup("snare")
		s.Units[3].Pos = pos(4, 0)
		s.Units[3].Stats.Mov = 5
		s.Units[3].AI = tactics.Hold
		b := mustNewStill(t, s)
		mustAct(t, b, effectCmd(pos(4, 0)))
		if len(b.Reachable(3)) != 1 || !b.Units()[3].Snared {
			t.Fatal("移動が止まらない")
		}
		ev, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		if b.Units()[3].Snared || b.Units()[3].EffectiveStats().Mov != 5 {
			t.Fatal("期限切れ後の移動")
		}
		found := false
		for _, e := range ev {
			if e.Kind == tactics.EventEffectExpire && e.Unit == 3 {
				found = true
			}
		}
		if !found {
			t.Fatal("期限切れの出来事なし")
		}
	})
	t.Run("TC-DES-NG-14-04 失敗時不変", func(t *testing.T) {
		b := mustNewStill(t, effectSetup("power_tonic"))
		for _, p := range []tactics.Pos{pos(4, 0), pos(3, 3), pos(-1, 0), pos(0, 4)} {
			before := b.Units()
			h := b.History()
			if _, err := b.Act(effectCmd(p)); err == nil {
				t.Fatalf("%+v が成功", p)
			}
			if !reflect.DeepEqual(before, b.Units()) || !reflect.DeepEqual(h, b.History()) {
				t.Fatal("失敗で状態が変化")
			}
		}
		mustAct(t, b, effectCmd(pos(0, 3)))
	})
	t.Run("TC-DES-NG-14-05 本人と距離外", func(t *testing.T) {
		s := effectSetup("power_tonic")
		s.Units[1].Pos = pos(0, 4)
		b := mustNewStill(t, s)
		if _, err := b.ItemForecast(effectCmd(pos(0, 4))); err == nil {
			t.Fatal("距離外")
		}
		mustAct(t, b, effectCmd(pos(0, 0)))
		if b.Units()[0].PowerTonic != 4 {
			t.Fatal("本人に付与されない")
		}
	})
	t.Run("TC-DES-NG-15-01 巻き戻し", func(t *testing.T) {
		b := mustNewStill(t, effectSetup("power_tonic"))
		before := b.Units()
		cmd := effectCmd(pos(0, 3))
		first := mustAct(t, b, cmd)
		h := b.History()
		if err := b.Rewind(h[0].ID); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, b.Units()) {
			t.Fatal("復元失敗")
		}
		second := mustAct(t, b, cmd)
		if !reflect.DeepEqual(first, second) {
			t.Fatal("再実行の出来事が異なる")
		}
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if b.Units()[1].PowerTonic != 0 {
			t.Fatal("期限切れなし")
		}
		if err := b.Rewind(b.History()[0].ID); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, b.Units()) {
			t.Fatal("期限切れ後の復元失敗")
		}
	})
	t.Run("TC-DES-NG-15-03 敵 AI は使わない", func(t *testing.T) {
		s := effectSetup("power_tonic")
		s.Units[3].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "粘着瓶", Effect: "snare", Range: 4}, Count: 1}}
		b := mustNewStill(t, s)
		ev, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range ev {
			if e.Kind == tactics.EventItemUse || e.Kind == tactics.EventEffectApply {
				t.Fatal("敵が道具を使った")
			}
		}
		if b.Units()[3].Items[0].Count != 1 {
			t.Fatal("敵の道具が消費された")
		}
	})

	t.Run("TC-DES-NG-13-02 tactics に直接渡す定義の検証", func(t *testing.T) {
		tests := []tactics.Item{{Effect: "unknown", Range: 3}, {Effect: "power_tonic", Range: 3}, {Effect: "power_tonic", Amount: 4}, {Effect: "snare", Range: 4, Amount: 1}, {Effect: "snare", Range: 4, Heal: 2}, {Effect: "power_tonic", Amount: 4, Range: 3, Bomb: true, BlastDamage: 2, ThrowRange: 2}, {Amount: 4, Range: 3}}
		for _, it := range tests {
			s := effectSetup("power_tonic")
			s.Units[0].Items[0].Item = it
			if _, _, err := tactics.New(s, 1); err == nil {
				t.Errorf("不正な定義を受理: %+v", it)
			}
		}
	})

}

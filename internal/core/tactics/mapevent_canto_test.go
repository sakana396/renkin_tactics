package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 再移動のあとの出来事の評価（DES-IT06 9.2・9.3 節、#124。REV-068 の申し送り）。

func TestMapEventAfterCanto(t *testing.T) {
	t.Run("TC-FR-MAP-03-17 再移動は、その移動のあとで同じ評価をする", func(t *testing.T) {
		// 主人公は移動 6・再移動。(1, 0) → (3, 0) の費用 2 で門番を攻撃し、残りは 4。
		attackGuard := tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.Attack, Target: 3}
		cases := []struct {
			name     string
			area     tactics.Area
			canto    tactics.Command
			triggers []string
		}{
			{name: "再移動で範囲に着く", area: tactics.Area{Min: pos(1, 2), Max: pos(1, 2)},
				canto: tactics.Command{Unit: 0, To: pos(1, 2), Action: tactics.CantoMove}, triggers: []string{"entered"}},
			{name: "再移動で範囲を通り過ぎるだけ", area: tactics.Area{Min: pos(2, 0), Max: pos(2, 0)},
				canto: tactics.Command{Unit: 0, To: pos(1, 0), Action: tactics.CantoMove}},
			{name: "攻撃の位置が範囲の中でも、その場で終えるだけでは着いていない", area: tactics.Area{Min: pos(1, 2), Max: pos(1, 2)},
				canto: tactics.Command{Unit: 0, To: pos(3, 0), Action: tactics.CantoEnd}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := eventSetup()
				s.Units[0].Stats.Mov = 6
				s.Units[0].ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
				entered := reinforceEvent("entered", tactics.TriggerPlayerEntered, "", 6, 0)
				entered.Area = tc.area
				s.Events = []tactics.MapEventSetup{reinforceEvent("attacked", tactics.TriggerEnemyAttacked, "guard", 7, 0), entered}
				b := newEventBattle(t, s, 99)

				// 行動の区切りでは、攻撃されたの 1 件だけが起きる。
				events := mustAct(t, b, attackGuard)
				assertEqual(t, "行動のあと", triggers(events), []string{"attacked"})
				if _, ok := b.PendingCanto(); !ok {
					t.Fatal("再移動を待っていない")
				}
				// 再移動の区切り: 待つ前の観測（攻撃された）を繰り返さず、再移動の移動だけで評価する。
				events = mustAct(t, b, tc.canto)
				assertEqual(t, "再移動のあと", triggers(events), tc.triggers)
				if len(tc.triggers) > 0 {
					move, trig := indexOf(events, tactics.EventMove), indexOf(events, tactics.EventTrigger)
					if move < 0 || trig < move {
						t.Errorf("出来事の順 = %v（移動のあとに EventTrigger）", kinds(events))
					}
				}
				// 起きた出来事は「起きた」のまま。番の終わり（敵軍の番の反撃を含む）まで 2 回目は起きない。
				events = mustAct(t, b, tactics.Command{Unit: 1, To: pos(0, 7), Action: tactics.Wait})
				if got := triggers(events); len(got) != 0 {
					t.Errorf("番の終わりまでに起きた出来事 = %v", got)
				}
			})
		}
	})
}

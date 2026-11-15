package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestBlastExperience は爆弾の撃破経験と出来事の順を確かめる（DES-NG-02）。
func TestBlastExperience(t *testing.T) {
	t.Run("TC-DES-NG-02-29 2体の撃破経験は50で制限し成長する", func(t *testing.T) {
		s := blastSetup(bomb(99, 1, 3))
		s.Units[0].Level, s.Units[0].Exp = 3, 70
		s.Units[0].Growth = tactics.Growth{HP: 100}
		s.Units[3].Level, s.Units[4].Level = 2, 4
		// 味方を爆発の外へ移す。
		s.Units[1].Pos = pos(0, 0)
		b := mustNewStill(t, s)
		rolls := 0
		b.SetRoll(func() int { rolls++; return 50 })
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventBlast, tactics.EventBlastHit, tactics.EventFall, tactics.EventBlastHit, tactics.EventFall, tactics.EventExp, tactics.EventLevelUp})
		u := b.Units()[0]
		if u.Level != 4 || u.Exp != 20 || u.Stats.HP != s.Units[0].Stats.HP+1 || rolls != 8 || events[5].Amount != 50 {
			t.Fatalf("成長結果 %+v、乱数 %d、経験 %d", u, rolls, events[5].Amount)
		}
	})
	t.Run("TC-DES-NG-02-30 レベル差と経験の下限", func(t *testing.T) {
		for _, tc := range []struct{ user, enemy, want int }{{5, 1, 18}, {17, 8, 10}} {
			s := blastSetup(bomb(99, 0, 3))
			s.Units[0].Level, s.Units[0].Exp, s.Units[3].Level = tc.user, 10, tc.enemy
			b := mustNewStill(t, s)
			ev := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
			if b.Units()[0].Exp != 10+tc.want || ev[len(ev)-1].Amount != tc.want {
				t.Fatalf("経験: %+v", ev)
			}
		}
	})
	t.Run("TC-DES-NG-02-31 敵を倒さない場合と味方だけ倒した場合は経験なし", func(t *testing.T) {
		for _, tc := range []struct {
			item   tactics.Item
			center tactics.Pos
		}{{bomb(8, 1, 3), pos(3, 3)}, {bomb(99, 0, 4), pos(2, 3)}} {
			b := mustNewStill(t, blastSetup(tc.item))
			ev := mustAct(t, b, throwCmd(0, pos(3, 6), tc.center))
			for _, e := range ev {
				if e.Kind == tactics.EventExp {
					t.Fatal("経験が発生", ev)
				}
			}
			if b.Units()[0].Exp != 0 {
				t.Fatal(b.Units()[0].Exp)
			}
		}
	})
	t.Run("TC-DES-NG-02-33 経験と成長の後に勝利", func(t *testing.T) {
		s := blastSetup(bomb(99, 1, 3))
		s.Units[0].Level, s.Units[0].Exp = 1, 90
		s.Units[1].Pos = pos(0, 0)
		s.Units[3].Boss = true
		s.Units[7].Boss = false
		s.Units[3].Level, s.Units[4].Level = 1, 1
		b := mustNewStill(t, s)
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{tactics.EventBlast, tactics.EventBlastHit, tactics.EventFall, tactics.EventBlastHit, tactics.EventFall, tactics.EventExp, tactics.EventLevelUp, tactics.EventWon})
		if u := b.Units()[0]; u.Level != 2 || u.Exp != 40 {
			t.Fatalf("経験 %+v", u)
		}
	})

	t.Run("TC-DES-NG-02-32 使用者の生死とレベルの境界", func(t *testing.T) {
		for _, level := range []int{0, 20} {
			s := blastSetup(bomb(99, 0, 3))
			s.Units[0].Level = level
			b := mustNewStill(t, s)
			ev := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
			for _, e := range ev {
				if e.Kind == tactics.EventExp || e.Kind == tactics.EventLevelUp {
					t.Fatalf("Lv%d: %+v", level, ev)
				}
			}
		}
		s := blastSetup(bomb(99, 1, 3))
		s.Units[0].Level, s.Units[0].Exp = 19, 60
		s.Units[3].Level, s.Units[4].Level = 19, 19
		s.Units[1].Pos = pos(0, 0)
		b := mustNewStill(t, s)
		rolls := 0
		b.SetRoll(func() int { rolls++; return 50 })
		ev := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		if u := b.Units()[0]; u.Level != 20 || u.Exp != 0 || rolls != 8 {
			t.Fatalf("成長 %+v 乱数 %d", u, rolls)
		}
		if ev[len(ev)-2].Kind != tactics.EventExp || ev[len(ev)-2].Amount != 50 || ev[len(ev)-1].Kind != tactics.EventLevelUp {
			t.Fatal(ev)
		}
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		ev = mustAct(t, b, throwCmd(0, pos(4, 5), pos(5, 3)))
		for _, e := range ev {
			if e.Kind == tactics.EventExp || e.Kind == tactics.EventLevelUp {
				t.Fatal("Lv 20 の再撃破で経験が発生", ev)
			}
		}
		if b.Units()[5].Alive {
			t.Fatal("再撃破の対象が生きている")
		}
		s = blastSetup(bomb(99, 1, 3))
		s.Units[0].Lord = false
		s.Units[1].Lord = true
		s.Units[1].Pos = pos(0, 0)
		b = mustNewStill(t, s)
		ev = mustAct(t, b, throwCmd(0, pos(3, 4), pos(3, 3)))
		for _, e := range ev {
			if e.Kind == tactics.EventExp || e.Kind == tactics.EventLevelUp {
				t.Fatal(ev)
			}
		}
	})
	t.Run("TC-DES-NG-02-34 巻き戻しで経験と成長を再現", func(t *testing.T) {
		s := blastSetup(bomb(99, 1, 3))
		s.Units[0].Level, s.Units[0].Exp = 3, 70
		s.Units[0].Growth = tactics.Growth{HP: 100}
		s.Units[3].Level, s.Units[4].Level = 2, 4
		s.Units[1].Pos = pos(0, 0)
		b := mustNewStill(t, s)
		before := snapshot(b)
		b.SetRoll(func() int { return 50 })
		cmd := throwCmd(0, pos(3, 6), pos(3, 3))
		ev := mustAct(t, b, cmd)
		after := snapshot(b)
		h := b.History()
		if err := b.Rewind(h[len(h)-1].ID); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "巻き戻し", snapshot(b), before)
		ev2 := mustAct(t, b, cmd)
		assertEqual(t, "やり直し", snapshot(b), after)
		assertEqual(t, "出来事", ev2, ev)
	})

}

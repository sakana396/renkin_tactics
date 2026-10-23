package tactics

import (
	"reflect"
	"testing"
)

func TestRewindOwn(t *testing.T) {
	t.Run("TC-FR-BTL-10-27 明示の番終了は直前の自軍入力待ちへ戻る", func(t *testing.T) {
		for _, actAgain := range []bool{false, true} {
			b := snapshotBattle(t)
			rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
			before := b.snapshot()
			events, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			after := b.snapshot()
			h := b.History()[1]
			v, err := b.PreviewRewind(h.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertBoard(t, v, before)
			rewindTo(t, b, h.ID)
			if !reflect.DeepEqual(before, b.snapshot()) || b.NeedsResume() || b.Phase != Player || b.Result != Ongoing || b.progress.stage != playerInput || !b.units[0].Acted || b.units[1].Acted || b.RewindsLeft() != 9 {
				t.Fatal("番終了直前の入力待ちと行動済みを保持しない")
			}
			if actAgain {
				rewindAct(t, b, Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
			} else {
				got, err := b.EndPhase()
				if err != nil || !reflect.DeepEqual(got, events) || !reflect.DeepEqual(after, b.snapshot()) {
					t.Fatal("同じ番終了の結果が異なる", err)
				}
			}
		}
	})
	t.Run("TC-FR-BTL-10-27 自軍の移動・攻撃・回復・薬・待機は復元できる", func(t *testing.T) {
		for _, action := range []Action{Attack, Heal, UseItem, Wait} {
			b := snapshotBattle(t)
			b.units[0].HP = 40
			b.units[1].HP = 40
			cmd := Command{Unit: 0, To: Pos{1, 0}, Action: action, Target: 2}
			if action == Heal {
				b.units[0].Weapon = Weapon{Type: Staff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 3, MaxUses: 3}
				cmd.To = Pos{0, 0}
				cmd.Target = 1
			}
			before := b.snapshot()
			rewindAct(t, b, cmd)
			rewindTo(t, b, b.History()[0].ID)
			if !reflect.DeepEqual(before, b.snapshot()) || b.NeedsResume() || b.RewindsLeft() != 9 {
				t.Fatal("自軍行動の復元失敗", action)
			}
		}
	})
	t.Run("TC-FR-BTL-10-27 / TC-FR-BTL-10-28 対象外は不変で拒否し全種類の閲覧は0回でも可能", func(t *testing.T) {
		for _, automatic := range []bool{false, true} {
			for _, attack := range []bool{false, true} {
				b := snapshotBattle(t)
				if attack {
					b.units[2].Pos = Pos{1, 0}
				}
				if automatic {
					rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
					rewindAct(t, b, Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
				} else {
					if _, err := b.EndPhase(); err != nil {
						t.Fatal(err)
					}
				}
				for _, h := range b.history {
					before := b.snapshot()
					v, err := b.PreviewRewind(h.entry.ID)
					if err != nil {
						t.Fatal(err)
					}
					assertBoard(t, v, h.before)
					if !reflect.DeepEqual(before, b.snapshot()) {
						t.Fatal("閲覧で変更")
					}
					allowed := h.entry.Side == Player && (h.entry.Kind == HistoryAction || h.entry.Kind == HistoryEndPhase && !h.entry.Automatic)
					if !allowed {
						assertRewindFailure(t, b, ErrInvalidRewind, func() error { return b.Rewind(h.entry.ID) })
					}
					b.rewindsLeft = 0
					v, err = b.PreviewRewind(h.entry.ID)
					if err != nil {
						t.Fatal(err)
					}
					assertBoard(t, v, h.before)
					assertRewindFailure(t, b, ErrNoRewinds, func() error { return b.Rewind(h.entry.ID) })
					b.rewindsLeft = 10
				}
			}
		}
	})
}

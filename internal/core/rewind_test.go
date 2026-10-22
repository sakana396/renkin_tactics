package core_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestRewindCampaignBoundary(t *testing.T) {
	t.Run("TC-FR-BTL-10-10 勝利の反映は一度だけで新しい戦闘は十回", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		var saved bytes.Buffer
		if err := c.Save(&saved); err != nil {
			t.Fatal(err)
		}
		s := sortie(t, c, 0)
		beforeRoster := append([]core.Member(nil), c.Roster...)
		cmd := tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.Attack, Target: 1}
		win := func() {
			t.Helper()
			if _, err := s.Battle.Act(cmd); err != nil {
				t.Fatal(err)
			}
			if s.Battle.Result != tactics.Won {
				t.Fatal("勝利にならない")
			}
		}
		win()
		if err := s.Battle.Rewind(s.Battle.History()[0].ID); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Roster, beforeRoster) || c.Chapter != 0 || c.Stock[0].Count != 1 {
			t.Fatal("巻き戻しで Campaign を変更した")
		}
		if _, err := c.FinishChapter(s); !errors.Is(err, core.ErrNotWon) {
			t.Fatal(err)
		}
		if s.Battle.RewindsLeft() != 9 {
			t.Fatal("巻き戻し回数が不正")
		}
		// 同じ章の再出撃と章の始めのロードは独立した新しい Battle。
		restarted := sortie(t, c, 0)
		loaded, err := core.LoadCampaign(campaignCatalog(), bytes.NewReader(saved.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		for _, fresh := range []*core.Sortie{restarted, sortie(t, loaded, 0)} {
			if fresh.Battle.RewindsLeft() != 10 || len(fresh.Battle.History()) != 0 {
				t.Fatal("再出撃・ロードで履歴か回数を持ち越した")
			}
		}
		win()
		if _, err := c.FinishChapter(s); err != nil {
			t.Fatal(err)
		}
		if c.Chapter != 1 || c.Roster[0].Exp != 30 || c.Stock[0].Count != 3 {
			t.Fatal("巻き戻した未来の経験か報酬を重ねた")
		}
		if _, err := c.FinishChapter(s); !errors.Is(err, core.ErrInvalidSortie) {
			t.Fatal(err)
		}
		next := sortie(t, c, 0)
		if next.Battle.RewindsLeft() != 10 || len(next.Battle.History()) != 0 {
			t.Fatal("次章で履歴か回数を持ち越した")
		}
	})
}

package core_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 名簿の武器の列から出撃し、章の終わりに列を戻す（DES-ITM-02、DES-IT06 6.2 節）。
func TestIT06MemberWeapons(t *testing.T) {
	t.Run("TC-FR-ITM-01-04 名簿の武器の列の先頭を初めの装備にし、最後に使った武器は次の章へ持ち越さない", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		c.Roster[0].Weapons = []string{"s", "s"}
		c.Roster[0].Items = nil
		s, _, err := c.StartBattle([]int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		u := s.Battle.Units()[0]
		if len(u.Weapons) != 2 || u.Weapons[0].ID != "s" || u.Weapons[1].ID != "s" || u.Equipped != 0 || !reflect.DeepEqual(u.AllowedWeaponTypes, []tactics.WeaponType{tactics.Sword}) {
			t.Fatalf("出撃の武器 %+v、装備 %d、持てる種類 %v", u.Weapons, u.Equipped, u.AllowedWeaponTypes)
		}
		// 2 本目で攻撃して勝つ（装備は 2 本目になる）。
		boss := s.Battle.Units()[1].Pos
		second := 1
		if _, err := s.Battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: boss.X, Y: boss.Y + 1}, Action: tactics.Attack, Target: 1, Weapon: &second}); err != nil || s.Battle.Result != tactics.Won {
			t.Fatalf("勝てない: %v", err)
		}
		if s.Battle.Units()[0].Equipped != 1 {
			t.Fatal("2 本目を装備していない")
		}
		if _, err := c.FinishChapter(s); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Roster[0].Weapons, []string{"s", "s"}) {
			t.Fatalf("章の終わりの武器 %v", c.Roster[0].Weapons)
		}
		next, _, err := c.StartBattle([]int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if next.Battle.Units()[0].Equipped != 0 {
			t.Fatal("次の章が先頭から始まらない")
		}
	})
	t.Run("TC-FR-ITM-01-04 武器のない仲間は出撃できない", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		c.Roster[0].Weapons = nil
		if _, _, err := c.StartBattle([]int{0}, 1); !errors.Is(err, core.ErrInvalidDeployment) {
			t.Fatalf("武器のない出撃を受理: %v", err)
		}
	})
}

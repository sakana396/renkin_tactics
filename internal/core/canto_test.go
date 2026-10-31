package core_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 兵種のスキルを tactics の形に変える試験（DES-IT06 8.1 節、DES-CORE-01、#124）。

func TestClassSkillsConversion(t *testing.T) {
	t.Run("TC-FR-UNT-07-08 騎兵は自軍も敵も兵種のスキルの再移動を持ち、エルザのキャラのスキルは空", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		members := make([]int, len(c.Roster))
		for i := range members {
			members[i] = i
		}
		s, _, err := c.StartBattle(members, 1)
		if err != nil {
			t.Fatal(err)
		}
		canto := []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
		elsa, cavalierFoes := 0, 0
		for i, u := range s.Battle.Units() {
			want := []tactics.Skill(nil)
			if u.ClassName == "騎兵" {
				want = canto
				if u.Side == tactics.Player {
					elsa++
					if u.Name != "エルザ" || u.Skill != (tactics.Skill{}) {
						t.Errorf("units[%d] 自軍の騎兵 = %s・キャラのスキル %+v（want エルザ・なし）", i, u.Name, u.Skill)
					}
				} else {
					cavalierFoes++
				}
			}
			if !reflect.DeepEqual(u.ClassSkills, want) {
				t.Errorf("units[%d] %s（%s）の兵種のスキル = %+v（want %+v）", i, u.Name, u.ClassName, u.ClassSkills, want)
			}
		}
		if elsa != 1 || cavalierFoes == 0 {
			t.Errorf("自軍の騎兵 %d 体・敵の騎兵 %d 体（want 1 体・1 体以上）", elsa, cavalierFoes)
		}
	})
}

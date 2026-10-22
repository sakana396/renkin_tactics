package core_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

func TestCampaignDeploymentAt(t *testing.T) {
	t.Run("TC-FR-CMP-11-01", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		before := campaignSnapshot(c)
		old := sortie(t, c, 0, 1)
		oldUnits := old.Battle.Units()
		members, slots := []int{0, 2, 1}, []int{1, 0, 2}
		s, _, err := c.StartBattleAt(members, slots, 1)
		if err != nil {
			t.Fatal(err)
		}
		units := s.Battle.Units()
		for i, want := range []struct {
			name string
			pos  tactics.Pos
		}{{"主人公", tactics.Pos{X: 1}}, {"控え", tactics.Pos{}}, {"仲間", tactics.Pos{Y: 1}}} {
			if units[i].Name != want.name || units[i].Pos != want.pos {
				t.Fatalf("ユニット %d = %+v", i, units[i])
			}
		}
		if !reflect.DeepEqual(s.Members, members) || !reflect.DeepEqual(*c, before) || !reflect.DeepEqual(old.Battle.Units(), oldUnits) || !reflect.DeepEqual(members, []int{0, 2, 1}) || !reflect.DeepEqual(slots, []int{1, 0, 2}) {
			t.Fatal("対応または入力・既存状態が変わった")
		}
		members[1], slots[0] = 0, 0
		if !reflect.DeepEqual(s.Members, []int{0, 2, 1}) || !reflect.DeepEqual(s.Battle.Units(), units) {
			t.Fatal("入力と共有している")
		}
		s.Members[1] = 0
		if !reflect.DeepEqual(*c, before) || !reflect.DeepEqual(old.Battle.Units(), oldUnits) {
			t.Fatal("返す値と既存状態を共有している")
		}
	})
	t.Run("TC-FR-CMP-11-02", func(t *testing.T) {
		for _, tc := range []struct {
			members, slots []int
			positions      []tactics.Pos
		}{
			{[]int{0}, []int{0}, []tactics.Pos{{}}},
			{[]int{0}, []int{2}, []tactics.Pos{{Y: 1}}},
			{[]int{0, 2}, []int{2, 0}, []tactics.Pos{{Y: 1}, {}}},
			{[]int{0, 2, 1}, []int{2, 0, 1}, []tactics.Pos{{Y: 1}, {}, {X: 1}}},
		} {
			c := newCampaign(t, core.Withdraw)
			s, _, err := c.StartBattleAt(tc.members, tc.slots, 1)
			if err != nil {
				t.Fatal(err)
			}
			for i, pos := range tc.positions {
				if s.Battle.Units()[i].Pos != pos {
					t.Fatalf("枠 %v: %+v", tc.slots, s.Battle.Units())
				}
			}
		}
	})
	t.Run("TC-FR-CMP-11-03", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			members, slots []int
			change         func(*core.Campaign)
			want           error
			reason         string
		}{
			{"空", nil, nil, nil, core.ErrInvalidDeployment, "人数"},
			{"人数超過（全員生存）", []int{0, 1, 2, 3}, []int{0, 1, 2, 0}, nil, core.ErrInvalidDeployment, "人数"},
			{"名簿負", []int{-1}, []int{0}, nil, core.ErrInvalidDeployment, "名簿"},
			{"名簿上限", []int{4}, []int{0}, nil, core.ErrInvalidDeployment, "名簿"},
			{"戦死", []int{0, 1}, []int{0, 1}, func(c *core.Campaign) { c.Roster[1].Alive = false }, core.ErrInvalidDeployment, "名簿"},
			{"名簿重複", []int{0, 0}, []int{0, 1}, nil, core.ErrInvalidDeployment, "名簿"},
			{"主人公不在", []int{1}, []int{0}, nil, core.ErrInvalidDeployment, "主人公"},
			{"主人公が先頭以外", []int{1, 0}, []int{0, 1}, nil, core.ErrInvalidDeployment, "主人公"},
			{"枠不足", []int{0, 1}, []int{0}, nil, core.ErrInvalidDeployment, "長さ"},
			{"枠余分", []int{0}, []int{0, 1}, nil, core.ErrInvalidDeployment, "長さ"},
			{"枠負", []int{0}, []int{-1}, nil, core.ErrInvalidDeployment, "範囲"},
			{"枠上限", []int{0}, []int{3}, nil, core.ErrInvalidDeployment, "範囲"},
			{"枠重複", []int{0, 1}, []int{0, 0}, nil, core.ErrInvalidDeployment, "重複"},
			{"クリア優先", nil, []int{-1}, func(c *core.Campaign) { c.Cleared = true }, core.ErrCleared, ""},
			{"人数は枠検証より先", nil, []int{-1}, nil, core.ErrInvalidDeployment, "人数"},
			{"名簿は主人公と枠検証より先", []int{1, 4}, nil, nil, core.ErrInvalidDeployment, "名簿"},
			{"主人公は枠検証より先", []int{1}, nil, nil, core.ErrInvalidDeployment, "主人公"},
			{"長さは範囲より先", []int{0}, []int{-1, 0}, nil, core.ErrInvalidDeployment, "長さ"},
			{"全範囲は重複より先", []int{0, 1, 2}, []int{0, 0, 3}, nil, core.ErrInvalidDeployment, "範囲"},
			{"組み立て失敗", []int{0, 1}, []int{0, 1}, func(c *core.Campaign) { c.Roster[1].Character = "missing" }, nil, "missing"},
			{"tactics.New失敗", []int{0}, []int{0}, func(c *core.Campaign) { c.Roster[0].Stats.HP = 0 }, tactics.ErrInvalidSetup, ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := newCampaign(t, core.Withdraw)
				old := sortie(t, c, 0, 1)
				oldUnits, oldMembers := old.Battle.Units(), slices.Clone(old.Members)
				if tc.change != nil {
					tc.change(c)
				}
				before := campaignSnapshot(c)
				members, slots := slices.Clone(tc.members), slices.Clone(tc.slots)
				s, ev, err := c.StartBattleAt(tc.members, tc.slots, 1)
				if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) || !strings.Contains(err.Error(), tc.reason) || s != nil || ev != nil {
					t.Fatalf("出撃=%v 出来事=%v 誤り=%v", s, ev, err)
				}
				if !reflect.DeepEqual(*c, before) || !reflect.DeepEqual(tc.members, members) || !reflect.DeepEqual(tc.slots, slots) || !reflect.DeepEqual(old.Battle.Units(), oldUnits) || !reflect.DeepEqual(old.Members, oldMembers) {
					t.Fatal("失敗で状態が変わった")
				}
			})
		}
	})
	t.Run("TC-FR-CMP-11-04", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			c := newCampaign(t, rule)
			c.Roster[0].Exp = 80
			c.Roster[2].Exp = 42
			c.Roster[2].Items = []defs.ItemCount{{Item: "z", Count: 1}}
			before := campaignSnapshot(c)
			var savedBefore, savedAfter bytes.Buffer
			if err := c.Save(&savedBefore); err != nil {
				t.Fatal(err)
			}
			s, _, err := c.StartBattleAt([]int{0, 2, 1}, []int{2, 0, 1}, 1)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Save(&savedAfter); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(savedBefore.Bytes(), savedAfter.Bytes()) {
				t.Fatal("配置でセーブが変わった")
			}
			if _, err := s.Battle.EndPhase(); err != nil {
				t.Fatal(err)
			}
			if s.Battle.Units()[2].Alive {
				t.Fatal("仲間が倒れていない")
			}
			win(t, s)
			// 公開の並びを変えても、出撃時に保持した名簿との対応で反映する。
			s.Members[1], s.Members[2] = 1, 2
			r, err := c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			gains := tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1}
			want := []core.MemberResult{
				{Member: 0, LevelBefore: 1, LevelAfter: 2, ExpBefore: 80, ExpAfter: 10, Gains: gains},
				{Member: 2, LevelBefore: 3, LevelAfter: 3, ExpBefore: 42, ExpAfter: 42},
				{Member: 1, LevelBefore: 2, LevelAfter: 2, Fallen: true, Dead: rule == core.Permadeath},
			}
			hero := before.Roster[0]
			hero.Level = 2
			hero.Exp = 10
			hero.Stats = tactics.Stats{HP: 21, Str: 21, Skl: 101, Spd: 6, Def: 11, Mov: 5}
			friend := before.Roster[1]
			if rule == core.Permadeath {
				friend.Alive = false
				friend.Items = nil
			}
			if !reflect.DeepEqual(r.Members, want) || !reflect.DeepEqual(c.Roster[0], hero) || !reflect.DeepEqual(c.Roster[1], friend) || !reflect.DeepEqual(c.Roster[2], before.Roster[2]) || !reflect.DeepEqual(c.Roster[3], before.Roster[3]) {
				t.Fatalf("結果=%+v 名簿=%+v", r, c.Roster)
			}
		}
	})
}

package core_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 加入時の期待値は DES-IT05 3.4 節の表から定める（DES-CMP-10）。
func TestRecruitDefinitions(t *testing.T) {
	cat := baseCatalog(t)
	t.Run("TC-FR-CMP-08-04 加入の時機と能力", func(t *testing.T) {
		joins := map[string][]string{"ch02": {"elsa"}, "ch04": {"tio"}, "ch08": {"hanna"}}
		for _, ch := range cat.Chapters() {
			if !reflect.DeepEqual(ch.Joins, joins[ch.ID]) {
				t.Errorf("%s joins=%v", ch.ID, ch.Joins)
			}
		}
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch08")
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Roster) != 8 {
			t.Fatalf("名簿 %d", len(c.Roster))
		}
		for _, tc := range []struct {
			id, name, class, weapon string
			level                   int
			stats                   tactics.Stats
			growth                  defs.Growth
		}{
			{"tio", "ティオ", "archer", "iron_bow", 4, tactics.Stats{HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}, defs.Growth{HP: 60, Str: 55, Skl: 45, Spd: 40, Def: 20, Lck: 35, Mag: 5, Res: 15}},
			{"hanna", "ハンナ", "cleric", "heal_staff", 7, tactics.Stats{HP: 20, Str: 0, Skl: 4, Spd: 7, Def: 6, Mov: 5, Lck: 6, Mag: 3, Res: 7}, defs.Growth{HP: 60, Str: 0, Skl: 40, Spd: 35, Def: 50, Lck: 45, Mag: 40, Res: 50}},
		} {
			ch, ok := cat.Character(tc.id)
			if !ok || ch.Name != tc.name || ch.Class != tc.class || len(ch.Weapons) != 1 || ch.Weapons[0] != tc.weapon || ch.Growth != tc.growth || ch.Skill != "" || ch.Lord || len(ch.Items) != 0 {
				t.Errorf("定義 %s=%+v", tc.id, ch)
			}
			found := false
			for _, m := range c.Roster {
				if m.Character == tc.id {
					found = true
					if m.Level != tc.level || m.Exp != 0 || m.Stats != tc.stats || !m.Alive || len(m.Items) != 0 {
						t.Errorf("加入時 %s=%+v", tc.id, m)
					}
				}
			}
			if !found {
				t.Errorf("%s がいない", tc.id)
			}
		}
	})
	t.Run("TC-FR-CMP-08-05 両ルールで章を進めても戦死者を足し直さない", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 8; i++ {
				if i == 4 && rule == core.Permadeath {
					c.Roster[6].Alive = false
				}
				want := []int{5, 6, 6, 7, 7, 7, 7, 8}[i]
				if len(c.Roster) != want {
					t.Fatalf("章%d 名簿%d", i+1, len(c.Roster))
				}
				if _, err = c.FinishChapter(winByForce(t, c, uint64(i+1))); err != nil {
					t.Fatal(err)
				}
				var b bytes.Buffer
				if err = c.Save(&b); err != nil {
					t.Fatal(err)
				}
				c, err = core.LoadCampaign(cat, &b)
				if err != nil {
					t.Fatal(err)
				}
			}
			if len(c.Roster) != 8 || c.Roster[6].Character != "tio" || c.Roster[7].Character != "hanna" || c.Roster[6].Alive != (rule == core.Withdraw) {
				t.Fatal(c.Roster)
			}
		}
	})
}

func TestRecruitLoad(t *testing.T) {
	cat := baseCatalog(t)
	t.Run("TC-FR-SYS-01-08 古い名簿の加入を追いつかせ再読込で重複しない", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			for _, id := range []string{"ch03", "ch04", "ch05", "ch06", "ch07", "ch08", "ch09", "clear"} {
				t.Run(fmt.Sprintf("%v/%s", rule, id), func(t *testing.T) {
					c, err := core.NewCampaignAt(cat, rule, "ch03")
					if err != nil {
						t.Fatal(err)
					}
					if id == "clear" {
						c.Cleared = true
						c.Chapter = 9
					} else {
						for i, ch := range cat.Chapters() {
							if ch.ID == id {
								c.Chapter = i
							}
						}
					}
					c.Roster[1].Exp = 23
					c.Roster[1].Stats.HP = 31
					c.Roster[2].Alive = false
					before := campaignSnapshot(c)
					var b bytes.Buffer
					if err = c.Save(&b); err != nil {
						t.Fatal(err)
					}
					got, err := core.LoadCampaign(cat, bytes.NewReader(b.Bytes()))
					if err != nil {
						t.Fatal(err)
					}
					want := 6
					if id != "clear" && id >= "ch04" {
						want = 7
					}
					if id != "clear" && id >= "ch08" {
						want = 8
					}
					if len(got.Roster) != want || !reflect.DeepEqual(got.Roster[:6], c.Roster) || got.Rule != rule || got.Chapter != c.Chapter || got.Cleared != c.Cleared || !reflect.DeepEqual(got.Stock, c.Stock) {
						t.Fatalf("読み込み %+v", got)
					}
					for i, st := range []tactics.Stats{{HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}, {HP: 20, Skl: 4, Spd: 7, Def: 6, Mov: 5, Lck: 6, Mag: 3, Res: 7}} {
						if 6+i < want {
							m := got.Roster[6+i]
							if m.Stats != st || m.Level != []int{4, 7}[i] || m.Exp != 0 || !m.Alive || len(m.Items) != 0 {
								t.Fatal(m)
							}
						}
					}
					if want > 6 {
						got.Roster[6].Alive = false
					}
					b.Reset()
					if err = got.Save(&b); err != nil {
						t.Fatal(err)
					}
					again, err := core.LoadCampaign(cat, &b)
					if err != nil || !sameRecruitCampaign(got, again) {
						t.Fatal("再読込", err)
					}
					if !reflect.DeepEqual(before, campaignSnapshot(c)) {
						t.Fatal("元の状態が変化")
					}
				})
			}
		}
	})
	t.Run("TC-FR-SYS-01-08 加入の参照不正は途中の状態を返さない", func(t *testing.T) {
		c, _ := core.NewCampaign(cat, core.Withdraw)
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		var chars []defs.CharacterDef
		var classes []defs.ClassDef
		for _, id := range append(append([]string(nil), cat.Game.Party...), "elsa") {
			ch, _ := cat.Character(id)
			chars = append(chars, ch)
			cls, _ := cat.Class(ch.Class)
			classes = append(classes, cls)
		}
		potion, _ := cat.Item("potion")
		bad := defs.NewCatalog(cat.Game, nil, classes, nil, []defs.ItemDef{potion}, nil, chars, nil, []defs.ChapterDef{{ID: "ch01", Joins: []string{"elsa", "missing"}}}, nil)
		got, err := core.LoadCampaign(bad, &b)
		if got != nil || !errors.Is(err, core.ErrInvalidSave) {
			t.Fatal(got, err)
		}
	})
}

func sameRecruitCampaign(a, b *core.Campaign) bool {
	if a == nil || b == nil {
		return false
	}
	x, y := campaignSnapshot(a), campaignSnapshot(b)
	for i := range x.Roster {
		if len(x.Roster[i].Items) == 0 {
			x.Roster[i].Items = nil
		}
	}
	for i := range y.Roster {
		if len(y.Roster[i].Items) == 0 {
			y.Roster[i].Items = nil
		}
	}
	return slices.Equal(x.Stock, y.Stock) && reflect.DeepEqual(x.Roster, y.Roster) && x.Rule == y.Rule && x.Chapter == y.Chapter && x.Cleared == y.Cleared
}

// 試遊で使う名簿の成長年数を加入章から数え、控えを初期値に保つ（DES-CMP-10、DES-MAP-06）。
func TestRecruitEstimatedRoster(t *testing.T) {
	t.Run("TC-FR-MAP-04-07 育った名簿と控えの名簿の既知値", func(t *testing.T) {
		cat := baseCatalog(t)
		for _, tc := range []struct {
			id    string
			level [2]int
			stats [2]tactics.Stats
		}{
			{"ch04", [2]int{4, 4}, [2]tactics.Stats{{HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}, {HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}}},
			{"ch05", [2]int{5, 4}, [2]tactics.Stats{{HP: 21, Str: 9, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Res: 1}, {HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}}},
			{"ch06", [2]int{6, 5}, [2]tactics.Stats{{HP: 21, Str: 9, Skl: 10, Spd: 8, Def: 4, Mov: 5, Lck: 4, Res: 1}, {HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}}},
			{"ch07", [2]int{7, 6}, [2]tactics.Stats{{HP: 22, Str: 10, Skl: 10, Spd: 8, Def: 5, Mov: 5, Lck: 4, Res: 1}, {HP: 21, Str: 9, Skl: 10, Spd: 7, Def: 4, Mov: 5, Lck: 4, Res: 1}}},
			{"ch08", [2]int{8, 7}, [2]tactics.Stats{{HP: 22, Str: 10, Skl: 11, Spd: 9, Def: 5, Mov: 5, Lck: 4, Res: 1}, {HP: 21, Str: 9, Skl: 10, Spd: 8, Def: 4, Mov: 5, Lck: 4, Res: 1}}},
			{"ch09", [2]int{9, 8}, [2]tactics.Stats{{HP: 23, Str: 11, Skl: 11, Spd: 9, Def: 5, Mov: 5, Lck: 4, Res: 1}, {HP: 21, Str: 9, Skl: 10, Spd: 8, Def: 4, Mov: 5, Lck: 4, Res: 1}}},
		} {
			for k, low := range []bool{false, true} {
				grew := estimatedRosterWithRecruits(t, cat, tc.id, low, false)
				bench := estimatedRosterWithRecruits(t, cat, tc.id, low, true)
				if m := grew.Roster[6]; m.Level != tc.level[k] || m.Stats != tc.stats[k] || m.Exp != 0 {
					t.Fatal(tc.id, low, m)
				}
				if m := bench.Roster[6]; m.Level != 4 || m.Stats != (tactics.Stats{HP: 20, Str: 8, Skl: 9, Spd: 7, Def: 4, Mov: 5, Lck: 4, Mag: 0, Res: 1}) || m.Exp != 0 {
					t.Fatal(m)
				}
				if tc.id >= "ch08" {
					level := 7
					st := tactics.Stats{HP: 20, Skl: 4, Spd: 7, Def: 6, Mov: 5, Lck: 6, Mag: 3, Res: 7}
					if tc.id == "ch09" && !low {
						level = 8
						st.HP = 21
						st.Def = 7
					}
					if m := grew.Roster[7]; m.Level != level || m.Stats != st || m.Exp != 0 {
						t.Fatal(m)
					}
					if m := bench.Roster[7]; m.Level != 7 || m.Stats != (tactics.Stats{HP: 20, Skl: 4, Spd: 7, Def: 6, Mov: 5, Lck: 6, Mag: 3, Res: 7}) || m.Exp != 0 {
						t.Fatal(m)
					}
				}
				for i := 0; i < 6; i++ {
					if !reflect.DeepEqual(grew.Roster[i], bench.Roster[i]) {
						t.Fatal("控えの指定が既存者を変更")
					}
				}
			}
		}
	})
}

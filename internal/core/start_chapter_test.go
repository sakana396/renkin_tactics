package core_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// startChapterCatalog は過去・現在・未来の加入と報酬を区別する 4 章の目録（DES-CMD-03）。
func startChapterCatalog() *defs.Catalog {
	return defs.NewCatalog(defs.GameDef{Party: []string{"hero"}, Stock: []defs.ItemCount{{Item: "z", Count: 2}}}, nil,
		[]defs.ClassDef{{ID: "class", Base: defs.Stats{HP: 20, Str: 3, Skl: 4, Spd: 5, Def: 6, Mov: 7}}}, nil, nil, nil,
		[]defs.CharacterDef{{ID: "hero", Class: "class", Level: 1, Bonus: defs.Stats{Str: 2}, Items: []defs.ItemCount{{Item: "z", Count: 1}}}, {ID: "past", Class: "class", Level: 3}, {ID: "current", Class: "class", Level: 5}, {ID: "future", Class: "class", Level: 7}}, nil,
		[]defs.ChapterDef{{ID: "c1", Joins: []string{"past"}, Reward: []defs.ItemCount{{Item: "a", Count: 3}}}, {ID: "c2", Joins: []string{"past", "hero"}, Reward: []defs.ItemCount{{Item: "z", Count: 4}}}, {ID: "c3", Joins: []string{"current"}, Reward: []defs.ItemCount{{Item: "a", Count: 8}}}, {ID: "c4", Joins: []string{"future"}, Reward: []defs.ItemCount{{Item: "z", Count: 16}}}}, nil)
}

func TestNewCampaignAt(t *testing.T) {
	t.Run("TC-FR-CMP-01-06 指定章までの加入と前章までの報酬", func(t *testing.T) {
		base, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name   string
			cat    *defs.Catalog
			id     string
			index  int
			roster []string
			stock  []defs.ItemCount
		}{
			{"本体の最初", base, "ch01", 0, []string{"leon", "gald", "bran", "mira", "rina"}, nil},
			{"本体の加入章", base, "ch02", 1, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa"}, []defs.ItemCount{{Item: "potion", Count: 2}}},
			{"本体の ch03", base, "ch03", 2, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa"}, []defs.ItemCount{{Item: "potion", Count: 4}}},
			{"本体の ch04", base, "ch04", 3, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio"}, []defs.ItemCount{{Item: "potion", Count: 4}}},
			{"本体の ch05", base, "ch05", 4, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio"}, []defs.ItemCount{{Item: "potion", Count: 6}}},
			{"本体の ch06", base, "ch06", 5, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio"}, []defs.ItemCount{{Item: "potion", Count: 8}}},
			{"本体の ch07", base, "ch07", 6, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio"}, []defs.ItemCount{{Item: "potion", Count: 10}}},
			{"本体の ch08", base, "ch08", 7, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio", "hanna"}, []defs.ItemCount{{Item: "potion", Count: 12}}},
			// 本体の最後の章（#102、#104 で添字 8）。ch03 に報酬はなく、ch01・ch02・ch04〜ch08 の薬は合計 14。加入は ch02 のエルザ・ch04 のティオ・ch08 のハンナ。
			{"本体の ch09", base, "ch09", 8, []string{"leon", "gald", "bran", "mira", "rina", "noel", "elsa", "tio", "hanna"}, []defs.ItemCount{{Item: "potion", Count: 14}}},
			{"試験の最初", startChapterCatalog(), "c1", 0, []string{"hero", "past"}, []defs.ItemCount{{Item: "z", Count: 2}}},
			{"試験の途中", startChapterCatalog(), "c3", 2, []string{"hero", "past", "current"}, []defs.ItemCount{{Item: "a", Count: 3}, {Item: "z", Count: 6}}},
			{"試験の最後", startChapterCatalog(), "c4", 3, []string{"hero", "past", "current", "future"}, []defs.ItemCount{{Item: "a", Count: 11}, {Item: "z", Count: 6}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
					c, err := core.NewCampaignAt(tc.cat, rule, tc.id)
					if err != nil {
						t.Fatal(err)
					}
					if c.Chapter != tc.index || c.Rule != rule || c.Cleared || !reflect.DeepEqual(c.Stock, tc.stock) || len(c.Roster) != len(tc.roster) {
						t.Fatalf("章・在庫・名簿 = %+v", c)
					}
					for i, id := range tc.roster {
						m := c.Roster[i]
						ch, _ := tc.cat.Character(id)
						cls, _ := tc.cat.Class(ch.Class)
						want := tactics.Stats{HP: cls.Base.HP + ch.Bonus.HP, Str: cls.Base.Str + ch.Bonus.Str, Skl: cls.Base.Skl + ch.Bonus.Skl, Spd: cls.Base.Spd + ch.Bonus.Spd, Def: cls.Base.Def + ch.Bonus.Def, Mov: cls.Base.Mov + ch.Bonus.Mov, Lck: cls.Base.Lck + ch.Bonus.Lck, Mag: cls.Base.Mag + ch.Bonus.Mag, Res: cls.Base.Res + ch.Bonus.Res}
						if m.Character != id || m.Level != ch.Level || m.Exp != 0 || !m.Alive || m.Stats != want || !reflect.DeepEqual(m.Items, ch.Items) {
							t.Fatalf("名簿[%d] = %+v", i, m)
						}
					}
					if tc.index == 0 {
						normal, err := core.NewCampaign(tc.cat, rule)
						if err != nil || !reflect.DeepEqual(c, normal) {
							t.Fatal("通常開始と不一致", err)
						}
					}
					// 名簿・手持ち・在庫は目録や別の開始結果と共有しない。
					before := campaignSnapshot(c)
					if len(c.Roster[0].Items) > 0 {
						c.Roster[0].Items[0].Count = 999
					}
					if len(c.Stock) > 0 {
						c.Stock[0].Count = 999
					}
					c.Roster[0].Stats.HP = 999
					again, err := core.NewCampaignAt(tc.cat, rule, tc.id)
					if err != nil || !reflect.DeepEqual(before, campaignSnapshot(again)) {
						t.Fatal("目録を変更した", err)
					}
				}
			})
		}
	})
	t.Run("TC-FR-CMP-01-06 不正な指定は状態を返さない", func(t *testing.T) {
		for _, id := range []string{"", "unknown", "c5", "4"} {
			c, err := core.NewCampaignAt(startChapterCatalog(), core.Withdraw, id)
			if c != nil || err == nil {
				t.Fatal(id, c, err)
			}
		}
		for _, rule := range []core.Rule{-1, 99} {
			c, err := core.NewCampaignAt(startChapterCatalog(), rule, "c3")
			if c != nil || !errors.Is(err, core.ErrInvalidRule) {
				t.Fatal(c, err)
			}
		}
		empty := defs.NewCatalog(defs.GameDef{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		c, err := core.NewCampaignAt(empty, core.Withdraw, "c1")
		if c != nil || err == nil {
			t.Fatal(c, err)
		}
	})
}

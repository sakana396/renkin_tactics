package core_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// campaignCatalog は計算を固定できる 3 章の目録。成長率 100 なら 5 能力が必ず各 1 伸びる。
// edit を渡すと、目録を作る前にキャラの定義を書き換える。
func campaignCatalog(edit ...func(chars []defs.CharacterDef)) *defs.Catalog {
	chars := []defs.CharacterDef{
		{ID: "hero", Name: "主人公", Class: "hero", Weapon: "s", Level: 1, Lord: true, Growth: defs.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}, Items: []defs.ItemCount{{Item: "z", Count: 2}}},
		{ID: "friend", Name: "仲間", Class: "weak", Weapon: "s", Level: 2, Items: []defs.ItemCount{{Item: "a", Count: 1}}},
		{ID: "bench", Name: "控え", Class: "weak", Weapon: "s", Level: 3},
		{ID: "join1", Name: "加入1", Class: "weak", Weapon: "s", Level: 4, Items: []defs.ItemCount{{Item: "z", Count: 1}}},
		{ID: "join2", Name: "加入2", Class: "weak", Weapon: "s", Level: 5},
	}
	for _, e := range edit {
		e(chars)
	}
	return defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend", "bench"}, Stock: []defs.ItemCount{{Item: "z", Count: 2}, {Item: "a", Count: 1}}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{{ID: "hero", Name: "主人公", WeaponType: "sword", Base: defs.Stats{HP: 20, Str: 20, Skl: 100, Spd: 5, Def: 10, Mov: 5}}, {ID: "weak", Name: "兵", WeaponType: "sword", Base: defs.Stats{HP: 10, Str: 20, Skl: 100, Spd: 5, Mov: 5}}},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", Hit: 100, MinRange: 1, MaxRange: 1}},
		[]defs.ItemDef{{ID: "a", Name: "同じ薬", Heal: 10}, {ID: "z", Name: "同じ薬", Heal: 10}}, nil,
		chars, []defs.MapDef{{ID: "m", Rows: []string{"...", "..."}, Slots: []defs.Slot{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}, Enemies: []defs.EnemyPlacement{{Level: 1, Class: "weak", Weapon: "s", AI: "hold", Boss: true, X: 2, Y: 0}}}},
		[]defs.ChapterDef{{ID: "c1", Map: "m", Joins: []string{"join1"}, Reward: []defs.ItemCount{{Item: "a", Count: 2}}}, {ID: "c2", Map: "m", Joins: []string{"join2", "friend"}, Reward: []defs.ItemCount{{Item: "z", Count: 2}}}, {ID: "c3", Map: "m"}}, nil)
}

func newCampaign(t *testing.T, rule core.Rule) *core.Campaign {
	t.Helper()
	c, err := core.NewCampaign(campaignCatalog(), rule)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func sortie(t *testing.T, c *core.Campaign, members ...int) *core.Sortie {
	t.Helper()
	s, _, err := c.StartBattle(members, 1)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func campaignSnapshot(c *core.Campaign) core.Campaign {
	copy := *c
	copy.Roster = slices.Clone(c.Roster)
	for i := range copy.Roster {
		copy.Roster[i].Items = slices.Clone(c.Roster[i].Items)
	}
	copy.Stock = slices.Clone(c.Stock)
	return copy
}
func TestCampaignNew(t *testing.T) {
	t.Run("TC-FR-CMP-04-01", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			cat := campaignCatalog()
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			if c.Rule != rule || c.Chapter != 0 || c.Cleared || len(c.Roster) != 4 {
				t.Fatalf("新規 = %+v", c)
			}
			for i, id := range []string{"hero", "friend", "bench", "join1"} {
				m := c.Roster[i]
				if m.Character != id || m.Level != i+1 || m.Exp != 0 || !m.Alive {
					t.Errorf("名簿 = %+v", m)
				}
			}
			if c.Roster[0].Stats != (tactics.Stats{HP: 20, Str: 20, Skl: 100, Spd: 5, Def: 10, Mov: 5}) || !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "a", Count: 1}, {Item: "z", Count: 2}}) {
				t.Errorf("能力・在庫 = %+v", c)
			}
			c.Roster[0].Items[0].Count = 99
			c.Stock[0].Count = 99
			ch, _ := cat.Character("hero")
			if ch.Items[0].Count != 2 || cat.Game.Stock[0].Count != 2 {
				t.Fatal("定義と共有している")
			}
		}
	})
	t.Run("TC-FR-CMP-04-02", func(t *testing.T) {
		for _, r := range []core.Rule{core.Rule(-1), core.Rule(99)} {
			c, err := core.NewCampaign(campaignCatalog(), r)
			if c != nil || !errors.Is(err, core.ErrInvalidRule) {
				t.Fatalf("%v %v", c, err)
			}
		}
	})
}
func TestCampaignDeployment(t *testing.T) {
	t.Run("TC-FR-CMP-03-01 / TC-FR-CMP-03-02 / TC-FR-UNT-04-14", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		c.Roster[2].Exp = 42
		c.Roster[2].Stats.HP = 25
		before := campaignSnapshot(c)
		members := []int{0, 2, 1}
		s := sortie(t, c, members...)
		members[1] = 0
		u := s.Battle.Units()
		if !reflect.DeepEqual(s.Members, []int{0, 2, 1}) || u[1].Name != "控え" || u[1].Level != 3 || u[1].Exp != 42 || u[1].HP != 25 || u[1].Pos != (tactics.Pos{X: 1, Y: 0}) || u[0].Growth != (tactics.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}) || u[0].Items[0].Item.ID != "z" || u[3].Level != 1 {
			t.Fatalf("出撃 = %+v", u)
		}
		for i, pos := range []tactics.Pos{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			if u[i].Pos != pos {
				t.Fatalf("旧 API の配置 %d = %v", i, u[i].Pos)
			}
		}
		single := sortie(t, c, 0)
		if single.Battle.Units()[0].Pos != (tactics.Pos{}) {
			t.Fatal("主人公のみの旧 API の配置が違う")
		}
		explicit, events, err := c.StartBattleAt([]int{0, 2, 1}, []int{0, 1, 2}, 1)
		if err != nil || !reflect.DeepEqual(explicit.Battle.Units(), u) || len(events) == 0 {
			t.Fatalf("連番枠との互換性: %v", err)
		}
		if !reflect.DeepEqual(*c, before) {
			t.Fatal("出撃で状態が変わった")
		}
		cat := loadBase(t)
		camp, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		legacy, _, err := core.NewBattle(cat, "ch01", 1)
		if err != nil {
			t.Fatal(err)
		}
		selected := make([]int, len(cat.Game.Party))
		for i := range selected {
			selected[i] = i
		}
		current := sortie(t, camp, selected...)
		if !reflect.DeepEqual(legacy.Units(), current.Battle.Units()) {
			t.Fatal("NewBattle と StartBattle が違う")
		}
		setup, err := core.NewSetup(cat, "ch01")
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range setup.Reinforcements {
			if r.Unit.Level != []int{1, 2, 3}[i] {
				t.Fatal("増援のレベルが定義と違う")
			}
		}
	})
	t.Run("TC-FR-CMP-03-03", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			ids  []int
			dead bool
		}{
			{"空", nil, false},
			{"枠超過（全員生存）", []int{0, 1, 2, 3}, false},
			{"範囲外（下限）", []int{-1}, false},
			{"範囲外（上限）", []int{99}, false},
			{"重複", []int{0, 0}, false},
			{"主人公が先頭でない", []int{1, 0}, false},
			{"戦死", []int{0, 3}, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := newCampaign(t, core.Withdraw)
				c.Roster[3].Alive = !tc.dead
				before := campaignSnapshot(c)
				idsBefore := slices.Clone(tc.ids)
				s, ev, err := c.StartBattle(tc.ids, 1)
				if !errors.Is(err, core.ErrInvalidDeployment) || s != nil || ev != nil || !reflect.DeepEqual(*c, before) || !reflect.DeepEqual(tc.ids, idsBefore) {
					t.Fatalf("%v: %v", tc.ids, err)
				}
			})
		}
		c := newCampaign(t, core.Withdraw)
		c.Cleared = true
		_, _, err := c.StartBattle(nil, 1)
		if !errors.Is(err, core.ErrCleared) {
			t.Fatal(err)
		}
	})
}
func TestCampaignItems(t *testing.T) {
	t.Run("TC-FR-CMP-03-04 / TC-FR-CMP-03-05", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		for _, id := range []string{"a", "z", "z"} {
			if err := c.TakeItem(0, id); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(c.Roster[0].Items, []defs.ItemCount{{Item: "z", Count: 4}, {Item: "a", Count: 1}}) || len(c.Stock) != 0 {
			t.Fatalf("取り出し = %+v", c)
		}
		if err := c.StoreItem(1, "a"); err != nil {
			t.Fatal(err)
		}
		before := campaignSnapshot(c)
		if err := c.TakeItem(0, "a"); !errors.Is(err, core.ErrHandFull) || !reflect.DeepEqual(*c, before) {
			t.Fatal("上限の検証", err)
		}
		if err := c.StoreItem(0, "z"); err != nil {
			t.Fatal(err)
		}
		if err := c.StoreItem(0, "a"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "a", Count: 2}, {Item: "z", Count: 1}}) || !reflect.DeepEqual(c.Roster[0].Items, []defs.ItemCount{{Item: "z", Count: 3}}) || len(c.Roster[1].Items) != 0 {
			t.Fatalf("預ける = %+v", c)
		}
	})
	t.Run("TC-FR-CMP-03-06", func(t *testing.T) {
		for _, tc := range []struct {
			name                string
			cleared, dead, full bool
			member              int
			item                string
			wantTake, wantStore error
		}{
			{"クリア最優先", true, true, true, -1, "none", core.ErrCleared, core.ErrCleared},
			{"添字下限", false, false, false, -1, "none", core.ErrInvalidMember, core.ErrInvalidMember},
			{"添字上限", false, false, false, 4, "none", core.ErrInvalidMember, core.ErrInvalidMember},
			{"戦死", false, true, false, 0, "none", core.ErrInvalidMember, core.ErrInvalidMember},
			{"在庫優先", false, false, true, 0, "none", core.ErrNoStock, core.ErrNoHandItem},
			{"手持ち上限", false, false, true, 0, "a", core.ErrHandFull, core.ErrNoHandItem},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := newCampaign(t, core.Withdraw)
				c.Cleared = tc.cleared
				c.Roster[0].Alive = !tc.dead
				if tc.full {
					c.Roster[0].Items[0].Count = 5
				}
				before := campaignSnapshot(c)
				if err := c.TakeItem(tc.member, tc.item); !errors.Is(err, tc.wantTake) {
					t.Fatal(err)
				}
				if err := c.StoreItem(tc.member, tc.item); !errors.Is(err, tc.wantStore) {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(*c, before) {
					t.Fatal("失敗で変化")
				}
			})
		}
	})
}

// win は敵を実際に倒す。成長の期待値は DES-GRW-01・02 の 30 経験、100% の 5 能力各＋1。
func win(t *testing.T, s *core.Sortie) {
	t.Helper()
	boss := s.Battle.Units()[len(s.Members)].Pos
	_, err := s.Battle.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: boss.X, Y: boss.Y + 1}, Action: tactics.Attack, Target: len(s.Members)})
	if err != nil {
		t.Fatal(err)
	}
	if s.Battle.Result != tactics.Won {
		t.Fatal("勝利していない")
	}
}
func TestCampaignFinish(t *testing.T) {
	t.Run("TC-FR-UNT-05-05 章の結果の伸びに幸運・魔力・魔防を含める", func(t *testing.T) {
		// REV-069 F-02: FinishChapter の Gains も 8 つの成長を数える。
		cat := campaignCatalog(func(chars []defs.CharacterDef) {
			chars[0].Growth = defs.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100, Lck: 100, Mag: 100, Res: 100}
		})
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Exp = 80
		before := c.Roster[0].Stats
		s := sortie(t, c, 0)
		win(t, s)
		r, err := c.FinishChapter(s)
		if err != nil {
			t.Fatal(err)
		}
		want := tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1, Lck: 1, Mag: 1, Res: 1}
		if len(r.Members) != 1 || r.Members[0].LevelAfter != 2 || r.Members[0].Gains != want {
			t.Fatalf("結果 %+v", r.Members)
		}
		if after := c.Roster[0].Stats; after.Lck != before.Lck+1 || after.Mag != before.Mag+1 || after.Res != before.Res+1 {
			t.Fatalf("名簿の能力 %+v → %+v", before, after)
		}
	})
	t.Run("TC-FR-CMP-01-02 / TC-FR-CMP-02-01 / TC-FR-CMP-08-02 / TC-FR-CMP-10-01 / TC-FR-ITM-03-03", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		c.Roster[0].Exp = 80
		c.Roster[0].Items[0].Count = 1
		s := sortie(t, c, 0)
		win(t, s)
		r, err := c.FinishChapter(s)
		if err != nil {
			t.Fatal(err)
		}
		if r.Members[0].Member != 0 || c.Roster[r.Members[0].Member].Character != "hero" {
			t.Fatal("旧 API の結果が主人公の名簿に対応しない")
		}
		want := core.MemberResult{Member: 0, LevelBefore: 1, LevelAfter: 2, ExpBefore: 80, ExpAfter: 10, Gains: tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1}}
		if r.Chapter != "c1" || r.Cleared || !reflect.DeepEqual(r.Members, []core.MemberResult{want}) || !reflect.DeepEqual(r.Reward, []defs.ItemCount{{Item: "a", Count: 2}}) || c.Chapter != 1 || c.Cleared || c.Roster[0].Level != 2 || c.Roster[0].Exp != 10 || c.Roster[0].Stats.HP != 21 || c.Roster[0].Items[0].Item != "z" || c.Roster[0].Items[0].Count != 1 || c.Stock[0].Count != 3 || len(c.Roster) != 5 || c.Roster[4].Character != "join2" || c.Roster[2].Level != 3 {
			t.Fatalf("結果=%+v、名簿=%+v", r, c)
		}
		r.Reward[0].Count = 999
		if c.Stock[0].Count != 3 {
			t.Fatal("結果と在庫が共有")
		}
		// 最後の章の添字は章の数から求める（DES-IT05 8.3 節。3 に固定しない）。
		last := len(campaignCatalog().Chapters()) - 1
		for chapter := 1; chapter <= last; chapter++ {
			s := sortie(t, c, 0)
			win(t, s)
			r, err = c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			if c.Chapter != chapter+1 || c.Cleared != (chapter == last) || r.Cleared != c.Cleared {
				t.Fatalf("進行 = %+v", c)
			}
		}
		if c.Stock[1].Count != 4 {
			t.Fatal("報酬の加算", c.Stock)
		}
	})
	t.Run("TC-FR-ITM-03-03 手持ちを使ってから反映", func(t *testing.T) {
		c := newCampaign(t, core.Withdraw)
		c.Roster[1].Stats.HP = 30
		c.Roster[1].Stats.Str = 0
		c.Roster[1].Stats.Def = 0
		s := sortie(t, c, 0, 1)
		if _, err := s.Battle.EndPhase(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Battle.Act(tactics.Command{Unit: 1, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.UseItem, Item: 0}); err != nil {
			t.Fatal(err)
		}
		win(t, s)
		if _, err := c.FinishChapter(s); err != nil {
			t.Fatal(err)
		}
		if len(c.Roster[1].Items) != 0 || c.Roster[0].Items[0].Item != "z" || c.Roster[0].Items[0].Count != 2 || c.Stock[0].Count != 3 {
			t.Fatalf("手持ち・報酬 = %+v", c)
		}
	})

	for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
		t.Run(map[core.Rule]string{core.Withdraw: "TC-FR-CMP-05-01", core.Permadeath: "TC-FR-CMP-06-01"}[rule], func(t *testing.T) {
			c := newCampaign(t, rule)
			s := sortie(t, c, 0, 1)
			if _, err := s.Battle.EndPhase(); err != nil {
				t.Fatal(err)
			}
			if s.Battle.Units()[1].Alive {
				t.Fatal("仲間が倒れていない")
			}
			win(t, s)
			r, err := c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			dead := rule == core.Permadeath
			if len(c.Roster) != 5 || c.Roster[1].Alive == dead || c.Roster[1].Level != 2 || !r.Members[1].Fallen || r.Members[1].Dead != dead || c.Stock[0].Count != 3 {
				t.Fatalf("離脱・ロスト = %+v %+v", c, r)
			}
			if dead {
				if len(c.Roster[1].Items) != 0 {
					t.Fatal("戦死者の手持ち")
				}
				_, _, err := c.StartBattle([]int{0, 1}, 1)
				if !errors.Is(err, core.ErrInvalidDeployment) {
					t.Fatal(err)
				}
				if err := c.StoreItem(1, "a"); !errors.Is(err, core.ErrInvalidMember) {
					t.Fatal(err)
				}
			} else {
				next := sortie(t, c, 0, 1)
				u := next.Battle.Units()[1]
				if u.HP != 10 || !u.Alive || u.Items[0].Item.ID != "a" {
					t.Fatalf("復帰 = %+v", u)
				}
			}
		})
	}
	t.Run("TC-FR-CMP-01-03 / TC-FR-CMP-01-04", func(t *testing.T) {
		c := newCampaign(t, core.Permadeath)
		s := sortie(t, c, 0)
		other := newCampaign(t, core.Withdraw)
		for _, bad := range []*core.Sortie{nil, {}, sortie(t, other, 0)} {
			before := campaignSnapshot(c)
			if _, err := c.FinishChapter(bad); !errors.Is(err, core.ErrInvalidSortie) || !reflect.DeepEqual(*c, before) {
				t.Fatal(err)
			}
		}
		for _, result := range []tactics.Result{tactics.Ongoing, tactics.Lost} {
			s.Battle.Result = result
			before := campaignSnapshot(c)
			if _, err := c.FinishChapter(s); !errors.Is(err, core.ErrNotWon) || !reflect.DeepEqual(*c, before) {
				t.Fatal(err)
			}
		}
		s.Battle.Result = tactics.Ongoing
		stale := sortie(t, c, 0)
		win(t, s)
		if _, err := c.FinishChapter(s); err != nil {
			t.Fatal(err)
		}
		for _, bad := range []*core.Sortie{s, stale} {
			before := campaignSnapshot(c)
			if _, err := c.FinishChapter(bad); !errors.Is(err, core.ErrInvalidSortie) || !reflect.DeepEqual(*c, before) {
				t.Fatal(err)
			}
		}
	})
}

// 倒れる前に得た経験と成長を、離脱でもロストでも書き戻す（DES-CMP-06）。
func TestCampaignFallenGrowth(t *testing.T) {
	for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
		t.Run(map[core.Rule]string{core.Withdraw: "TC-FR-CMP-05-01", core.Permadeath: "TC-FR-CMP-06-01"}[rule]+" 倒れる前の成長", func(t *testing.T) {
			base := campaignCatalog()
			var classes []defs.ClassDef
			for _, id := range []string{"hero", "weak"} {
				v, _ := base.Class(id)
				classes = append(classes, v)
			}
			w, _ := base.Weapon("s")
			a, _ := base.Item("a")
			z, _ := base.Item("z")
			var chars []defs.CharacterDef
			for _, id := range []string{"hero", "friend", "bench", "join1", "join2"} {
				v, _ := base.Character(id)
				if id == "friend" {
					v.Growth = defs.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}
				}
				chars = append(chars, v)
			}
			m, _ := base.Map("m")
			m.Enemies = append(m.Enemies, defs.EnemyPlacement{Level: 1, Class: "weak", Weapon: "s", AI: "hold", X: 2, Y: 1})
			cat := defs.NewCatalog(base.Game, base.Terrains(), classes, []defs.WeaponDef{w}, []defs.ItemDef{a, z}, nil, chars, []defs.MapDef{m}, base.Chapters(), nil)
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			c.Roster[1].Exp = 80
			s := sortie(t, c, 0, 1)
			if _, err := s.Battle.Act(tactics.Command{Unit: 1, To: tactics.Pos{X: 1, Y: 1}, Action: tactics.Attack, Target: 3}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Battle.EndPhase(); err != nil {
				t.Fatal(err)
			}
			if s.Battle.Units()[1].Alive {
				t.Fatal("成長した仲間が倒れていない")
			}
			win(t, s)
			result, err := c.FinishChapter(s)
			if err != nil {
				t.Fatal(err)
			}
			want := core.MemberResult{Member: 1, LevelBefore: 2, LevelAfter: 3, ExpBefore: 80, ExpAfter: 7, Gains: tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1}, Fallen: true, Dead: rule == core.Permadeath}
			if result.Members[1] != want || c.Roster[1].Level != 3 || c.Roster[1].Exp != 7 || c.Roster[1].Stats != (tactics.Stats{HP: 11, Str: 21, Skl: 101, Spd: 6, Def: 1, Mov: 5}) {
				t.Fatalf("倒れる前の成長 = %+v / %+v", result, c.Roster[1])
			}
		})
	}
}

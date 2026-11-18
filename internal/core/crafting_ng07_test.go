package core_test

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestCraftingNG07(t *testing.T) {
	cat := alchemyCatalog(t)
	newCampaign := func() *core.Campaign {
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	recipe := func(item string) string {
		for _, r := range cat.Recipes() {
			if r.Item == item {
				return r.ID
			}
		}
		t.Fatalf("レシピがない: %s", item)
		return ""
	}
	count := func(c *core.Campaign, id string) int {
		for _, v := range c.Stock {
			if v.Item == id {
				return v.Count
			}
		}
		return 0
	}
	t.Run("TC-DES-NG-19-02 材料名と不足数", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 1, "catalyst": 0}
		if err := c.Craft(recipe("bomb_small")); err == nil || err.Error() != "触媒が 1 足りません" {
			t.Fatal(err)
		}
	})
	t.Run("TC-DES-NG-19-01 基準維持の調合", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 5, "catalyst": 4}
		small, far := count(c, "bomb_small"), count(c, "bomb_far")
		for _, id := range []string{"bomb_small", "bomb_small", "bomb_small", "bomb_far"} {
			if err := c.Craft(recipe(id)); err != nil {
				t.Fatal(err)
			}
		}
		if c.Materials["ore"] != 0 || c.Materials["catalyst"] != 0 || count(c, "bomb_small") != small+3 || count(c, "bomb_far") != far+1 {
			t.Fatal(c.Materials, c.Stock)
		}
	})
	t.Run("TC-DES-NG-19-02 / TC-DES-NG-19-03 失敗時は不変", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 1, "catalyst": 1}
		if err := c.Craft(recipe("bomb_small")); err != nil {
			t.Fatal(err)
		}
		var before bytes.Buffer
		c.Save(&before)
		for _, id := range []string{recipe("bomb_small"), "unknown"} {
			if c.Craft(id) == nil {
				t.Fatal(id)
			}
			var after bytes.Buffer
			c.Save(&after)
			if before.String() != after.String() {
				t.Fatal("失敗で状態が変化")
			}
		}
		c.Materials = map[string]int{"ore": 1, "catalyst": 1}
		for i := range c.Stock {
			if c.Stock[i].Item == "bomb_small" {
				c.Stock[i].Count = math.MaxInt
			}
		}
		before.Reset()
		c.Save(&before)
		if c.Craft(recipe("bomb_small")) == nil {
			t.Fatal("上限を超えた")
		}
		var after bytes.Buffer
		c.Save(&after)
		if before.String() != after.String() {
			t.Fatal("上限で状態が変化")
		}
	})
	t.Run("TC-DES-NG-19-04 / TC-DES-NG-19-05 / TC-DES-NG-19-06 保存と旧形式と不正値", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 5, "catalyst": 4}
		if err := c.Craft(recipe("bomb_small")); err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		loaded, err := core.LoadCampaign(cat, bytes.NewReader(b.Bytes()))
		if err != nil || !reflect.DeepEqual(c.Materials, loaded.Materials) || !reflect.DeepEqual(c.Stock, loaded.Stock) {
			t.Fatal(loaded, err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(b.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		delete(doc, "materials")
		old, _ := json.Marshal(doc)
		loaded, err = core.LoadCampaign(cat, bytes.NewReader(old))
		if err != nil || len(loaded.Materials) != 0 || !reflect.DeepEqual(c.Stock, loaded.Stock) {
			t.Fatal(loaded, err)
		}
		for _, raw := range []string{`[{"material":"unknown","count":1}]`, `[{"material":"ore","count":-1}]`, `[{"material":"ore","count":1},{"material":"ore","count":2}]`, `[{"material":"ore","count":99999999999999999999999999999}]`} {
			doc["materials"] = json.RawMessage(raw)
			bad, _ := json.Marshal(doc)
			if _, err := core.LoadCampaign(cat, bytes.NewReader(bad)); err == nil || !strings.Contains(err.Error(), "core:") {
				t.Fatal(raw, err)
			}
		}
	})
	t.Run("TC-DES-NG-21-02 直接開始の前章報酬", func(t *testing.T) {
		for _, tc := range []struct {
			id            string
			ore, catalyst int
		}{{"ch02", 5, 4}, {"ch03", 10, 8}} {
			c, err := core.NewCampaignAt(cat, core.Withdraw, tc.id)
			if err != nil || c.Materials["ore"] != tc.ore || c.Materials["catalyst"] != tc.catalyst {
				t.Fatal(tc, c, err)
			}
		}
	})
	t.Run("TC-DES-NG-21-01 各章の勝利報酬", func(t *testing.T) {
		for chapter := 0; chapter < 3; chapter++ {
			c, err := core.NewCampaignAt(cat, core.Withdraw, cat.Chapters()[chapter].ID)
			if err != nil {
				t.Fatal(err)
			}
			beforeOre, beforeCat := c.Materials["ore"], c.Materials["catalyst"]
			potion, power, snare := count(c, "potion"), count(c, "tonic_power"), count(c, "snare_bottle")
			sortie, _, err := c.StartBattle([]int{0}, 1)
			if err != nil {
				t.Fatal(err)
			}
			sortie.Battle.Result = tactics.Won
			if _, err = c.FinishChapter(sortie); err != nil {
				t.Fatal(err)
			}
			extra := 0
			if chapter < 2 {
				extra = 1
			}
			if c.Materials["ore"] != beforeOre+5 || c.Materials["catalyst"] != beforeCat+4 || count(c, "potion") != potion+2 || count(c, "tonic_power") != power+extra || count(c, "snare_bottle") != snare+extra {
				t.Fatal(chapter, c.Materials, c.Stock)
			}
		}
	})
	t.Run("報酬の上限超過は状態を変えない", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 2, "catalyst": math.MaxInt}
		sortie, _, err := c.StartBattle([]int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		sortie.Battle.Result = tactics.Won
		var before bytes.Buffer
		if err := c.Save(&before); err != nil {
			t.Fatal(err)
		}
		if _, err := c.FinishChapter(sortie); err == nil {
			t.Fatal("上限超過を許した")
		}
		var after bytes.Buffer
		c.Save(&after)
		if before.String() != after.String() {
			t.Fatal("状態が変化した")
		}
	})
	t.Run("TC-DES-NG-22-01 調合後の戦闘記録", func(t *testing.T) {
		c := newCampaign()
		c.Materials = map[string]int{"ore": 1, "catalyst": 1}
		if err := c.Craft(recipe("bomb_small")); err != nil {
			t.Fatal(err)
		}
		sortie, events, err := c.StartBattleAt([]int{0}, []int{0}, 3)
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		log := core.NewPlayLog(&b, nil)
		log.Start(c, sortie, []int{0}, 3, events)
		pos := sortie.Battle.Units()[0].Pos
		if _, err := log.Act(sortie.Battle, tactics.Command{Unit: 0, To: pos, Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(b.String()))
		if err != nil || rep.FirstDiff != 0 {
			t.Fatal(rep, err)
		}
	})

}

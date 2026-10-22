package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// セーブの期待値は DES-SAV-01 のキー・順序・字下げから直接定める。
const saveExample = `{
  "version": 1,
  "rule": "withdraw",
  "chapter": "c2",
  "cleared": false,
  "roster": [
    {
      "character": "hero",
      "level": 2,
      "exp": 40,
      "stats": {
        "hp": 21,
        "str": 6,
        "skl": 9,
        "spd": 10,
        "def": 4,
        "mov": 5
      },
      "items": [
        {
          "item": "a",
          "count": 2
        }
      ],
      "alive": true
    }
  ],
  "stock": []
}
`

func saveCampaign(t *testing.T) *core.Campaign {
	t.Helper()
	c := newCampaign(t, core.Withdraw)
	c.Chapter = 1
	c.Roster = []core.Member{{Character: "hero", Level: 2, Exp: 40, Stats: tactics.Stats{HP: 21, Str: 6, Skl: 9, Spd: 10, Def: 4, Mov: 5}, Items: []defs.ItemCount{{Item: "a", Count: 2}}, Alive: true}}
	c.Stock = nil
	return c
}

func TestCampaignSaveFormat(t *testing.T) {
	t.Run("TC-FR-SYS-01-01 形式と同じ状態の再現", func(t *testing.T) {
		c := saveCampaign(t)
		before := campaignSnapshot(c)
		for i := 0; i < 2; i++ {
			var b bytes.Buffer
			if err := c.Save(&b); err != nil {
				t.Fatal(err)
			}
			if b.String() != saveExample {
				t.Fatalf("JSON が仕様と違う:\n%s", &b)
			}
			loaded, err := core.LoadCampaign(campaignCatalog(), &b)
			if err != nil {
				t.Fatal(err)
			}
			var again bytes.Buffer
			if err = loaded.Save(&again); err != nil || again.String() != saveExample {
				t.Fatalf("往復: %v\n%s", err, &again)
			}
			// 読み込み時に加入をやり直さず、目録を保持して出撃できる。
			if len(loaded.Roster) != 1 {
				t.Fatal("名簿が変わった")
			}
			if _, _, err := loaded.StartBattle([]int{0}, 1); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(*c, before) {
			t.Fatal("保存で状態が変わった")
		}
	})
	t.Run("TC-FR-SYS-01-02 クリアと戦死と空配列", func(t *testing.T) {
		c := saveCampaign(t)
		c.Rule = core.Permadeath
		c.Cleared = true
		c.Chapter = 3
		c.Roster[0].Items = nil
		c.Roster = append(c.Roster, core.Member{Character: "friend", Level: 20, Stats: tactics.Stats{HP: 1}, Alive: false})
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), `"chapter": ""`) || strings.Contains(b.String(), "null") || !strings.Contains(b.String(), `"rule": "permadeath"`) {
			t.Fatal(b.String())
		}
		loaded, err := core.LoadCampaign(campaignCatalog(), &b)
		if err != nil {
			t.Fatal(err)
		}
		if !loaded.Cleared || loaded.Chapter != 3 || loaded.Rule != core.Permadeath || loaded.Roster[1].Alive || loaded.Roster[1].Level != 20 {
			t.Fatalf("%+v", loaded)
		}
	})
	t.Run("TC-FR-SYS-01-03 在庫を ID 順に保存", func(t *testing.T) {
		c := saveCampaign(t)
		c.Stock = []defs.ItemCount{{Item: "z", Count: 2}, {Item: "a", Count: 1}}
		before := campaignSnapshot(c)
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		var written struct {
			Stock []struct {
				Item  string
				Count int
			}
		}
		if err := json.Unmarshal(b.Bytes(), &written); err != nil {
			t.Fatal(err)
		}
		if len(written.Stock) != 2 || written.Stock[0].Item != "a" || written.Stock[0].Count != 1 || written.Stock[1].Item != "z" || written.Stock[1].Count != 2 {
			t.Fatalf("保存した在庫の順が違う: %s", &b)
		}

		loaded, err := core.LoadCampaign(campaignCatalog(), &b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loaded.Stock, []defs.ItemCount{{Item: "a", Count: 1}, {Item: "z", Count: 2}}) || !reflect.DeepEqual(*c, before) {
			t.Fatal("在庫の順または元の状態が違う")
		}
	})
}

// 成長・持ち物・戦死を含む名簿全体を、ゲームの状態として照合する。
func TestCampaignSaveRoster(t *testing.T) {
	t.Run("TC-FR-SYS-01-06 名簿全体の往復と読み込み後の独立性", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			cat := campaignCatalog()
			c, err := core.NewCampaign(cat, rule)
			if err != nil {
				t.Fatal(err)
			}
			c.Chapter = 1
			c.Roster[0].Level = 7
			c.Roster[0].Exp = 99
			c.Roster[0].Stats = tactics.Stats{HP: 29, Str: 24, Skl: 103, Spd: 8, Def: 12, Mov: 6}
			c.Roster[1].Alive = false
			c.Roster[1].Items = nil
			c.Roster[2].Level = 20
			c.Roster[2].Exp = 0
			before := campaignSnapshot(c)
			var b bytes.Buffer
			if err := c.Save(&b); err != nil {
				t.Fatal(err)
			}
			loaded, err := core.LoadCampaign(cat, &b)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Rule != rule || loaded.Chapter != 1 || loaded.Cleared || len(loaded.Roster) != 4 || !slices.Equal(loaded.Stock, c.Stock) {
				t.Fatalf("往復後=%+v", loaded)
			}
			for i, m := range c.Roster {
				got := loaded.Roster[i]
				if got.Character != m.Character || got.Level != m.Level || got.Exp != m.Exp || got.Stats != m.Stats || got.Alive != m.Alive || !slices.Equal(got.Items, m.Items) {
					t.Fatalf("名簿[%d]=%+v、期待=%+v", i, got, m)
				}
			}
			loaded.Roster[0].Items[0].Count = 5
			loaded.Stock[0].Count = 9
			if !reflect.DeepEqual(*c, before) {
				t.Fatal("読み戻した状態が元と共有されている")
			}
			ch, _ := cat.Character("hero")
			if ch.Items[0].Count != 2 || cat.Game.Stock[0].Count != 2 {
				t.Fatal("目録が変わった")
			}
		}
		// 外部の JSON の在庫の順にも依存しない。
		input := strings.Replace(saveExample, `"stock": []`, `"stock": [{"item":"z","count":2},{"item":"a","count":1}]`, 1)
		loaded, err := core.LoadCampaign(campaignCatalog(), strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(loaded.Stock, []defs.ItemCount{{Item: "a", Count: 1}, {Item: "z", Count: 2}}) {
			t.Fatal(loaded.Stock)
		}
	})
}

func TestCampaignLoadValidation(t *testing.T) {
	cases := []struct {
		name, old, new, path string
		version              bool
	}{
		{"文法", `"version": 1`, `"version":`, "json", false},
		{"版なし", `"version": 1,`, "", "version", false},
		{"版文字列", `"version": 1`, `"version": "1"`, "version", false},
		{"版null", `"version": 1`, `"version": null`, "version", false},
		{"未来版と未知キー", `"version": 1`, `"version": 2, "future": 1`, "version", true},
		{"過去版", `"version": 1`, `"version": 0`, "version", true},
		{"未知キー", `"version": 1`, `"version": 1, "future": 1`, "future", false},
		{"能力の未知キー", `"hp": 21`, `"hp": 21, "future": 1`, "future", false},
		{"アイテムの未知キー", `"count": 2`, `"count": 2, "future": 1`, "future", false},
		{"ルール", `"withdraw"`, `"other"`, "rule", false},
		{"章", `"c2"`, `"gone"`, "chapter", false},
		{"クリア章あり", `"cleared": false`, `"cleared": true`, "chapter", false},
		{"キャラ", `"hero"`, `"missing"`, "roster[0].character", false},
		{"主人公なし", `"hero"`, `"friend"`, "roster", false},
		{"主人公戦死", `"alive": true`, `"alive": false`, "roster", false},
		{"レベル下限", `"level": 2`, `"level": 0`, "roster[0].level", false},
		{"レベル上限", `"level": 2`, `"level": 21`, "roster[0].level", false},
		{"経験下限", `"exp": 40`, `"exp": -1`, "roster[0].exp", false},
		{"経験上限", `"exp": 40`, `"exp": 100`, "roster[0].exp", false},
		{"HP下限", `"hp": 21`, `"hp": 0`, "roster[0].stats.hp", false},
		{"力下限", `"str": 6`, `"str": -1`, "roster[0].stats.str", false},
		{"技下限", `"skl": 9`, `"skl": -1`, "roster[0].stats.skl", false},
		{"速さ下限", `"spd": 10`, `"spd": -1`, "roster[0].stats.spd", false},
		{"守備下限", `"def": 4`, `"def": -1`, "roster[0].stats.def", false},
		{"移動下限", `"mov": 5`, `"mov": -1`, "roster[0].stats.mov", false},
		{"アイテムなし", `"item": "a"`, `"item": "missing"`, "roster[0].items[0].item", false},
		{"数下限", `"count": 2`, `"count": 0`, "roster[0].items[0].count", false},
		{"手持ち上限", `"count": 2`, `"count": 6`, "roster[0].items", false},
	}
	t.Run("TC-FR-SYS-01-04 版と読み込み検証", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := strings.Replace(saveExample, tc.old, tc.new, 1)
				c, err := core.LoadCampaign(campaignCatalog(), strings.NewReader(input))
				want := core.ErrInvalidSave
				if tc.version {
					want = core.ErrSaveVersion
				}
				if c != nil || !errors.Is(err, want) || !strings.Contains(strings.ToLower(err.Error()), tc.path) {
					t.Fatalf("結果=%v 誤り=%v 期待=%v 位置=%s", c, err, want, tc.path)
				}
			})
		}
	})
	t.Run("TC-FR-SYS-01-05 境界と集合の検証", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			edit func(map[string]any)
			path string
		}{
			{"空名簿", func(d map[string]any) { d["roster"] = []any{} }, "roster"},
			{"重複キャラ", func(d map[string]any) { r := d["roster"].([]any); d["roster"] = append(r, r[0]) }, "roster[1].character"},
			{"最大レベルの経験", func(d map[string]any) { d["roster"].([]any)[0].(map[string]any)["level"] = 20 }, "roster[0].exp"},
			{"在庫参照", func(d map[string]any) { d["stock"] = []any{map[string]any{"item": "missing", "count": 1}} }, "stock[0].item"},
			{"在庫数", func(d map[string]any) { d["stock"] = []any{map[string]any{"item": "a", "count": -1}} }, "stock[0].count"},
			{"手持ち合計超過", func(d map[string]any) {
				d["roster"].([]any)[0].(map[string]any)["items"] = []any{map[string]any{"item": "a", "count": 3}, map[string]any{"item": "z", "count": 3}}
			}, "roster[0].items"},
			{"在庫重複", func(d map[string]any) {
				d["stock"] = []any{map[string]any{"item": "a", "count": 1}, map[string]any{"item": "a", "count": 2}}
			}, "stock[1].item"},
			{"下限と手持ち5", func(d map[string]any) {
				m := d["roster"].([]any)[0].(map[string]any)
				m["level"] = 1
				m["exp"] = 99
				m["stats"] = map[string]any{"hp": 1, "str": 0, "skl": 0, "spd": 0, "def": 0, "mov": 0}
				m["items"] = []any{map[string]any{"item": "a", "count": 2}, map[string]any{"item": "z", "count": 3}}
			}, ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var d map[string]any
				if err := json.Unmarshal([]byte(saveExample), &d); err != nil {
					t.Fatal(err)
				}
				tc.edit(d)
				data, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				c, err := core.LoadCampaign(campaignCatalog(), bytes.NewReader(data))
				if tc.path == "" {
					if err != nil || c == nil {
						t.Fatal(err)
					}
					return
				}
				if c != nil || !errors.Is(err, core.ErrInvalidSave) || !strings.Contains(err.Error(), tc.path) {
					t.Fatalf("%v %v", c, err)
				}
			})
		}
	})
	t.Run("TC-NFR-07-07 検証順と失敗時の状態不変", func(t *testing.T) {
		current := saveCampaign(t)
		before := campaignSnapshot(current)
		inputs := []struct{ data, path string }{
			{strings.ReplaceAll(strings.Replace(saveExample, `"withdraw"`, `"other"`, 1), `"c2"`, `"gone"`), "rule"},
			{strings.ReplaceAll(strings.Replace(saveExample, `"c2"`, `"gone"`, 1), `"hero"`, `"missing"`), "chapter"},
			{strings.ReplaceAll(strings.Replace(saveExample, `"hero"`, `"missing"`, 1), `"level": 2`, `"level": 0`), "roster[0].character"},
			{strings.ReplaceAll(strings.Replace(saveExample, `"level": 2`, `"level": 0`, 1), `"count": 2`, `"count": 0`), "roster[0].level"},
			{saveExample + " {}", "json"},
		}
		for _, tc := range inputs {
			got, err := core.LoadCampaign(campaignCatalog(), strings.NewReader(tc.data))
			if got != nil || !errors.Is(err, core.ErrInvalidSave) || !strings.Contains(err.Error(), tc.path) {
				t.Fatalf("%v %v", got, err)
			}
		}
		if !reflect.DeepEqual(*current, before) {
			t.Fatal("既存の状態が変わった")
		}
	})
}

type saveBrokenIO struct {
	err   error
	short bool
}

func (b saveBrokenIO) Read([]byte) (int, error) { return 0, b.err }
func (b saveBrokenIO) Write(p []byte) (int, error) {
	if b.short {
		return len(p) - 1, nil
	}
	return 0, b.err
}

func TestCampaignSaveIO(t *testing.T) {
	t.Run("TC-NFR-07-08 入出力の失敗と誤りの識別", func(t *testing.T) {
		c := saveCampaign(t)
		before := campaignSnapshot(c)
		sentinel := errors.New("入出力失敗")
		if err := c.Save(saveBrokenIO{err: sentinel}); !errors.Is(err, sentinel) {
			t.Fatal(err)
		}
		if err := c.Save(saveBrokenIO{short: true}); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal(err)
		}
		got, err := core.LoadCampaign(campaignCatalog(), saveBrokenIO{err: sentinel})
		if got != nil || !errors.Is(err, sentinel) || !errors.Is(err, core.ErrInvalidSave) {
			t.Fatalf("%v %v", got, err)
		}
		if errors.Is(core.ErrInvalidSave, core.ErrSaveVersion) || errors.Is(core.ErrSaveVersion, core.ErrInvalidSave) {
			t.Fatal("誤りを識別できない")
		}
		if !reflect.DeepEqual(*c, before) {
			t.Fatal("失敗で状態が変わった")
		}
	})
}

package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

func TestIT06Save(t *testing.T) {
	t.Run("TC-FR-SYS-01-09 / TC-FR-SYS-01-10", func(t *testing.T) {
		c, err := core.LoadCampaign(baseCatalog(t), bytes.NewBufferString(`{"version":1,"rule":"permadeath","chapter":"ch03","roster":[{"character":"leon","level":1,"exp":0,"stats":{"hp":19,"str":7,"skl":11,"spd":12,"def":4,"mov":5},"alive":true},{"character":"rina","level":9,"exp":12,"stats":{"hp":20,"str":0,"skl":5,"spd":10,"def":3,"mov":5},"items":[{"item":"potion","count":5}],"alive":false}],"stock":[]}`))
		if err != nil {
			t.Fatal(err)
		}
		// 版 1 は初めの武器 1 本（杖）を先頭に渡し、5 枠からあふれた回復薬 1 個を在庫へ戻す（DES-IT06 6.3 節の例）。戦死した仲間も移し替え、生死は変えない。
		r := c.Roster[1]
		if r.Stats.Mag != 4 || r.Alive || !reflect.DeepEqual(r.Weapons, []string{"heal_staff"}) || !reflect.DeepEqual(r.Items, []defs.ItemCount{{Item: "potion", Count: 4}}) || !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: 1}}) {
			t.Fatalf("補い=%+v 在庫=%+v", r, c.Stock)
		}
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		var d map[string]any
		if err := json.Unmarshal(b.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		if d["version"] != float64(2) {
			t.Fatal("版 2 でない")
		}
		again, err := core.LoadCampaign(baseCatalog(t), &b)
		if err != nil {
			t.Fatal(err)
		}
		// 版 1 → 版 2 → 版 2 で、補うことも、あふれを移すことも繰り返さない。
		if !reflect.DeepEqual(again.Roster[1], r) || !reflect.DeepEqual(again.Stock, c.Stock) {
			t.Fatalf("再読込で変化: %+v / %+v", again.Roster[1], again.Stock)
		}
	})
	t.Run("TC-FR-SYS-01-10 #120 の版 2 にある武器 1 本 ＋ 回復薬 5 個は、あふれを在庫へ戻す", func(t *testing.T) {
		// REV-069: #120 の TakeItem は武器を数えなかったので、武器 1 本 ＋ 道具 5 個の版 2 のセーブがありうる。
		save := `{"version":2,"rule":"withdraw","chapter":"ch02","cleared":false,` +
			`"roster":[{"character":"leon","weapons":["iron_sword"],"level":1,"exp":0,"stats":{"hp":19,"str":7,"skl":11,"spd":12,"def":4,"lck":6,"mag":2,"res":2,"mov":5},` +
			`"items":[{"item":"potion","count":2},{"item":"potion","count":3}],"alive":true}],"stock":[{"item":"potion","count":2}],"stock_weapons":[]}`
		c, err := core.LoadCampaign(baseCatalog(t), bytes.NewBufferString(save))
		if err != nil {
			t.Fatal(err)
		}
		// 先頭から 5 枠（武器 1 本 ＋ 回復薬 2 個 ＋ 2 個）を残し、2 行目の残り 1 個を同じ ID の在庫に足す。
		if m := c.Roster[0]; !reflect.DeepEqual(m.Items, []defs.ItemCount{{Item: "potion", Count: 2}, {Item: "potion", Count: 2}}) || !reflect.DeepEqual(c.Stock, []defs.ItemCount{{Item: "potion", Count: 3}}) {
			t.Fatalf("あふれ: 名簿 %+v、在庫 %+v", m, c.Stock)
		}
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		again, err := core.LoadCampaign(baseCatalog(t), &b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(again.Roster[0], c.Roster[0]) || !reflect.DeepEqual(again.Stock, c.Stock) {
			t.Fatal("読み直しで変化")
		}
		// 武器 6 本と、道具 6 個（それだけで 5 枠を超える）は壊れたデータとして拒む。
		for _, bad := range []string{
			`"weapons":["iron_sword","iron_sword","iron_sword","iron_sword","iron_sword","iron_sword"],"level"`,
			`"weapons":["iron_sword"],"level"`,
		} {
			text := strings.Replace(save, `"weapons":["iron_sword"],"level"`, bad, 1)
			if bad == `"weapons":["iron_sword"],"level"` {
				text = strings.Replace(text, `{"item":"potion","count":3}`, `{"item":"potion","count":4}`, 1)
			}
			if _, err := core.LoadCampaign(baseCatalog(t), bytes.NewBufferString(text)); !errors.Is(err, core.ErrInvalidSave) {
				t.Fatalf("壊れたデータを受理: %v", err)
			}
		}
	})
	t.Run("TC-FR-SYS-01-09 / TC-FR-ITM-01-05 武器 1 本と回復薬 4 個を TakeItem で持ち、保存して読み直せる", func(t *testing.T) {
		// TakeItem も武器の本数を数え、武器 1 本 ＋ 道具 4 個で 5 枠になる（DES-IT06 6.1 節。REV-069 の申し送り）。
		c, err := core.NewCampaign(baseCatalog(t), core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		c.Stock = []defs.ItemCount{{Item: "potion", Count: 5}}
		for c.Roster[0].Items[0].Count < 4 {
			if err := c.TakeItem(0, "potion"); err != nil {
				t.Fatal(err)
			}
		}
		before := c.Stock[0].Count
		if err := c.TakeItem(0, "potion"); !errors.Is(err, core.ErrHandFull) || c.Roster[0].Items[0].Count != 4 || c.Stock[0].Count != before {
			t.Fatalf("6 枠目を受け取った: %v", err)
		}
		var buf bytes.Buffer
		if err := c.Save(&buf); err != nil {
			t.Fatal(err)
		}
		got, err := core.LoadCampaign(baseCatalog(t), &buf)
		if err != nil {
			t.Fatal(err)
		}
		m := got.Roster[0]
		if len(m.Weapons) != 1 || len(m.Items) != 1 || m.Items[0].Item != "potion" || m.Items[0].Count != 4 || len(got.Stock) != 1 || got.Stock[0].Count != c.Stock[0].Count {
			t.Fatalf("読み直し: 名簿 %+v、在庫 %+v", m, got.Stock)
		}
	})
}

func TestIT06Stats(t *testing.T) {
	t.Run("TC-FR-UNT-01-07", func(t *testing.T) {
		cat := baseCatalog(t)
		b, _, err := core.NewBattle(cat, "ch01", 1)
		if err != nil {
			t.Fatal(err)
		}
		want := [][3]int{{6, 2, 2}, {2, 0, 0}, {2, 0, 0}, {4, 0, 1}, {6, 0, 5}}
		for i, w := range want {
			s := b.Units()[i].Stats
			if [3]int{s.Lck, s.Mag, s.Res} != w {
				t.Fatalf("能力[%d]=%+v", i, s)
			}
		}
	})
}

func TestIT06SaveValidation(t *testing.T) {
	t.Run("TC-FR-SYS-01-09 版2の必須能力・負数・正しい0", func(t *testing.T) {
		c, err := core.NewCampaign(baseCatalog(t), core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		c.Roster[0].Stats.Lck = 0
		c.Roster[0].Stats.Mag = 0
		c.Roster[0].Stats.Res = 0
		var buf bytes.Buffer
		if err = c.Save(&buf); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"hp", "str", "skl", "spd", "def", "mov", "lck", "mag", "res"} {
			for _, kind := range []string{"欠け", "null", "負"} {
				var d map[string]any
				json.Unmarshal(buf.Bytes(), &d)
				stats := d["roster"].([]any)[0].(map[string]any)["stats"].(map[string]any)
				switch kind {
				case "欠け":
					delete(stats, key)
				case "null":
					stats[key] = nil
				case "負":
					stats[key] = -1
				}
				raw, _ := json.Marshal(d)
				if got, err := core.LoadCampaign(baseCatalog(t), bytes.NewReader(raw)); err == nil || got != nil {
					t.Fatalf("%s %s を受理", key, kind)
				}
			}
		}
		got, err := core.LoadCampaign(baseCatalog(t), bytes.NewReader(buf.Bytes()))
		if err != nil || got.Roster[0].Stats.Mag != 0 || got.Roster[0].Stats.Lck != 0 || got.Roster[0].Stats.Res != 0 {
			t.Fatal("正しい0の往復", err)
		}
	})
}

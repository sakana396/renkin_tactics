package core_test

import (
	"bytes"
	"encoding/json"
	"github.com/sakana396/renkin_tactics/internal/core"
	"testing"
)

func TestIT06Save(t *testing.T) {
	t.Run("TC-FR-SYS-01-09", func(t *testing.T) {
		c, err := core.LoadCampaign(baseCatalog(t), bytes.NewBufferString(`{"version":1,"rule":"permadeath","chapter":"ch03","roster":[{"character":"leon","level":1,"exp":0,"stats":{"hp":19,"str":7,"skl":11,"spd":12,"def":4,"mov":5},"alive":true},{"character":"rina","level":9,"exp":12,"stats":{"hp":20,"str":0,"skl":5,"spd":10,"def":3,"mov":5},"items":[{"item":"potion","count":5}],"alive":false}],"stock":[]}`))
		if err != nil {
			t.Fatal(err)
		}
		if c.Roster[1].Stats.Mag != 4 || c.Roster[1].Alive || len(c.Roster[1].Weapons) != 1 || len(c.Roster[1].Items) != 1 || c.Roster[1].Items[0].Count != 4 || len(c.Stock) != 1 || c.Stock[0].Count != 1 {
			t.Fatalf("補い=%+v", c.Roster[1])
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
		if again.Roster[1].Stats.Mag != 4 || again.Roster[1].Alive || len(again.Roster[1].Weapons) != 1 || again.Roster[1].Items[0].Count != 4 || len(again.Stock) != 1 || again.Stock[0].Count != 1 {
			t.Fatal("再読込で変化")
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

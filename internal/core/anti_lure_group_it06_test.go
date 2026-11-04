package core_test

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 1 章の #173 で採り直した行動の記録を再生し、砦前の一団の距離を調べる（TC-FR-MAP-04-28、DES-AI-02）。
func TestCh01FortFrontStaysTogether(t *testing.T) {
	t.Run("TC-FR-MAP-04-28 ch01 の行動記録", func(t *testing.T) {
		f, err := os.Open("testdata/play173-ch01.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		var b *tactics.Battle
		const sword, bow = 7, 11
		check := func(turn int) {
			t.Helper()
			u := b.Units()
			if !u[sword].Alive || !u[bow].Alive {
				t.Fatalf("%d ターン目に一員が倒れた", turn)
			}
			d := absInt(u[sword].Pos.X-u[bow].Pos.X) + absInt(u[sword].Pos.Y-u[bow].Pos.Y)
			if d > 7 {
				t.Fatalf("%d ターン目: 剣士 %v と弓兵 %v の距離 %d", turn, u[sword].Pos, u[bow].Pos, d)
			}
		}
		for sc.Scan() {
			var rec struct {
				Type string `json:"type"`
				Seed uint64 `json:"seed"`
				Turn int    `json:"turn"`
				Cmd  *struct {
					Unit   int    `json:"unit"`
					To     [2]int `json:"to"`
					Action string `json:"action"`
					Target int    `json:"target"`
					Item   int    `json:"item"`
					Weapon *int   `json:"weapon"`
				} `json:"cmd"`
			}
			if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
				t.Fatal(err)
			}
			if rec.Turn > 4 {
				break
			}
			switch rec.Type {
			case "start":
				setup, err := core.NewSetup(baseCatalog(t), "ch01")
				if err != nil {
					t.Fatal(err)
				}
				b, _, err = tactics.New(setup, rec.Seed)
				if err != nil {
					t.Fatal(err)
				}
				if b.Units()[sword].ID != "anti_lure_ch01_2" || b.Units()[bow].ID != "anti_lure_ch01_6" {
					t.Fatal("一員の番号が変わった")
				}
			case "action":
				acts := map[string]tactics.Action{"wait": tactics.Wait, "attack": tactics.Attack, "item": tactics.UseItem, "heal": tactics.Heal, "canto": tactics.CantoMove}
				a, ok := acts[rec.Cmd.Action]
				if !ok {
					t.Fatalf("未知の行動 %s", rec.Cmd.Action)
				}
				if _, err := b.Act(tactics.Command{Unit: rec.Cmd.Unit, To: tactics.Pos{X: rec.Cmd.To[0], Y: rec.Cmd.To[1]}, Action: a, Target: rec.Cmd.Target, Item: rec.Cmd.Item, Weapon: rec.Cmd.Weapon}); err != nil {
					t.Fatalf("%d ターン目 %s unit=%d to=%v target=%d: %v", rec.Turn, rec.Cmd.Action, rec.Cmd.Unit, rec.Cmd.To, rec.Cmd.Target, err)
				}
			case "end_phase":
				if _, err := b.EndPhase(); err != nil {
					t.Fatalf("%d ターン目: %v", rec.Turn, err)
				}
			}
			if rec.Turn == 4 && b != nil && b.Turn == 5 {
				break
			}
		}
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		if b == nil || b.Turn != 5 {
			t.Fatalf("4 ターン目まで再生できない: %+v", b)
		}
		check(4)
		for turn := 5; turn <= 6; turn++ {
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
			check(turn)
		}
	})
}

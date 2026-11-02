package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// writeLog は ch01 で主人公を 1 回待機させた記録を書く。
func writeLog(t *testing.T, path string) {
	t.Helper()
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	c, err := core.NewCampaign(cat, core.Withdraw)
	if err != nil {
		t.Fatal(err)
	}
	s, events, err := c.StartBattle([]int{0}, 3)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	l := core.NewPlayLog(&buf)
	l.Start(c, s, []int{0}, 3, events)
	u := s.Battle.Units()[0]
	if _, err := l.Act(s.Battle, tactics.Command{Unit: 0, To: u.Pos, Action: tactics.Wait}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	t.Run("TC-NFR-06-10 道具の終了コードと表示", func(t *testing.T) {
		dir := t.TempDir()
		good := filepath.Join(dir, "good.jsonl")
		writeLog(t, good)
		b, _ := os.ReadFile(good)
		bad := filepath.Join(dir, "bad.jsonl")
		os.WriteFile(bad, []byte(strings.Replace(string(b), `"kind":"wait"`, `"kind":"move"`, 1)), 0o644)
		broken := filepath.Join(dir, "broken.jsonl")
		os.WriteFile(broken, []byte("{\n"), 0o644)
		cases := []struct {
			args []string
			code int
			out  string
		}{
			{[]string{good}, 0, "すべて同じ（決まっていない）"},
			{[]string{bad}, 1, "行目で食い違い"},
			{[]string{broken}, 2, ""},
			{[]string{filepath.Join(dir, "none.jsonl")}, 2, ""},
			{nil, 2, ""},
		}
		for _, tc := range cases {
			var out, errOut bytes.Buffer
			if code := run(tc.args, &out, &errOut); code != tc.code || !strings.Contains(out.String(), tc.out) {
				t.Errorf("%v: 終了コード %d、表示 %q %q; 期待 %d と %q", tc.args, code, out.String(), errOut.String(), tc.code, tc.out)
			}
		}
	})
}

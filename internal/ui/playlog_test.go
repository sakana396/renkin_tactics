package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// failingLog は fail 回目の Write から失敗する書き出し先（0 なら失敗しない）。
type failingLog struct {
	buf    bytes.Buffer
	writes int
	fail   int
}

func (w *failingLog) Write(p []byte) (int, error) {
	w.writes++
	if w.fail > 0 && w.writes >= w.fail {
		return 0, errors.New("書けない")
	}
	return w.buf.Write(p)
}

// fakeNow は偽の時計。呼ぶたびに 1.5 秒ずつ進む（E-106）。
func fakeNow() func() time.Time {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	return func() time.Time { at = at.Add(1500 * time.Millisecond); return at }
}

// logScenes は記録の書き出し先・偽の時計・誤りの出し先を持つ場面を作り、戦闘を始める。
func logScenes(t *testing.T, open func(string) (io.Writer, error), stderr io.Writer) *SceneController {
	t.Helper()
	s := NewSceneController(Config{Catalog: sceneCatalog(false, false), Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }, PlayLog: open,
		Now: fakeNow(), Stderr: stderr})
	startScenes(s)
	clickSceneButton(t, s, "配置へ")
	clickSceneButton(t, s, "戦闘開始")
	playOut(s.BattleController())
	return s
}

// waitFriend は仲間（(0,1)）をその場で待機させる。
func waitFriend(c *Controller) {
	c.Click(tactics.Pos{X: 0, Y: 1})
	c.Click(tactics.Pos{X: 0, Y: 1})
	c.ChooseAction(tactics.Wait)
	playOut(c)
}

func logTypes(t *testing.T, data string) ([]string, []map[string]any) {
	t.Helper()
	var types []string
	var recs []map[string]any
	for _, line := range strings.Split(strings.TrimSuffix(data, "\n"), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("JSON でない行: %v\n%s", err, line)
		}
		types = append(types, m["type"].(string))
		recs = append(recs, m)
	}
	return types, recs
}

func TestPlayLogScenes(t *testing.T) {
	t.Run("TC-NFR-06-12 画面の行動・巻き戻し・勝ちが記録に残る", func(t *testing.T) {
		var buf bytes.Buffer
		var chapters []string
		s := logScenes(t, func(ch string) (io.Writer, error) { chapters = append(chapters, ch); return &buf, nil }, &bytes.Buffer{})
		if !reflect.DeepEqual(chapters, []string{"c1"}) {
			t.Fatalf("記録を作った章 %v; 期待 [c1]", chapters)
		}
		c := s.BattleController()
		waitFriend(c)
		c.OpenRewind()
		c.RewindEnter()
		c.RewindEnter()
		if c.Battle().Units()[1].Acted {
			t.Fatal("巻き戻せていない")
		}
		// 主人公がボスを攻撃して 1 撃で倒す（winScene と同じ操作）。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 0})
		c.Click(tactics.Pos{X: 1, Y: 0})
		playOut(c)
		if c.Battle().Result != tactics.Won {
			t.Fatalf("勝っていない: %v", c.Battle().Result)
		}
		types, recs := logTypes(t, buf.String())
		want := []string{"start", "phase", "action", "rewind", "action", "end"}
		if !reflect.DeepEqual(types, want) {
			t.Fatalf("記録の種類 %v; 期待 %v", types, want)
		}
		if recs[0]["seed"].(float64) != 7 || recs[0]["chapter"] != "c1" {
			t.Fatalf("start = %v", recs[0])
		}
		if recs[2]["name"] != "仲間" || recs[3]["undone"].(float64) != 1 || recs[4]["name"] != "主人公" || recs[4]["target_name"] == "" || recs[5]["result"] != "won" {
			t.Fatalf("記録 %v", recs)
		}
		if c.Message() != "" {
			t.Fatalf("誤りの案内が出た: %q", c.Message())
		}
		// 経過の秒は、出撃のときに読んだ時刻から測る（偽の時計は読むたびに 1.5 秒進む。秒は切り捨て）。
		// start は 1.5 秒で 1、以後の操作の行は 3・4・6 秒（3・4.5・6 の切り捨て）。phase・end にはない。
		wantElapsed := []any{1.0, nil, 3.0, 4.0, 6.0, nil}
		for i, r := range recs {
			if r["elapsed"] != wantElapsed[i] {
				t.Fatalf("%d 行目 %v の elapsed = %v; 期待 %v", i+1, r["type"], r["elapsed"], wantElapsed[i])
			}
		}
	})
	t.Run("TC-NFR-06-12 番を終えると end_phase と敵軍の番が残る", func(t *testing.T) {
		var buf bytes.Buffer
		s := logScenes(t, func(string) (io.Writer, error) { return &buf, nil }, &bytes.Buffer{})
		c := s.BattleController()
		waitFriend(c)
		// 主人公が行動していないので、E を 2 回押して番を終える。
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		types, recs := logTypes(t, buf.String())
		if len(types) < 6 || !reflect.DeepEqual(types[:5], []string{"start", "phase", "action", "end_phase", "phase"}) || recs[3]["unacted"].(float64) != 1 || recs[4]["side"] != "enemy" {
			t.Fatalf("記録の種類 %v", types)
		}
		// この盤面と seed 7 では、1 ターン目の敵軍の番で負ける。最後は end（lost・turn 1）の 6 行。
		last := recs[len(recs)-1]
		if c.Battle().Result != tactics.Lost || len(types) != 6 || last["type"] != "end" || last["result"] != "lost" || last["turn"].(float64) != 1 {
			t.Fatalf("勝敗 %v、記録 %v、最後 %v; 期待 負けで 6 行目が end（lost・turn 1）", c.Battle().Result, types, last)
		}
	})
	t.Run("TC-NFR-06-12 記録を作れなくても遊べ、案内を出す", func(t *testing.T) {
		var stderr bytes.Buffer
		s := logScenes(t, func(string) (io.Writer, error) { return nil, errors.New("作れない /x/logs") }, &stderr)
		c := s.BattleController()
		// 案内は短く、詳しい誤り（パス）は標準エラーへ（DES-LOG-02）。
		if c.Message() != "行動の記録を作れません（遊ぶのは続けられます）" || !strings.Contains(stderr.String(), "作れない /x/logs") {
			t.Fatalf("案内 %q、標準エラー %q", c.Message(), stderr.String())
		}
		waitFriend(c)
		if !c.Battle().Units()[1].Acted {
			t.Fatal("行動できない")
		}
	})
	t.Run("TC-NFR-06-12 途中で書けなくなったら一度だけ知らせ、遊ぶのは続く", func(t *testing.T) {
		w := &failingLog{fail: 3}
		var stderr bytes.Buffer
		s := logScenes(t, func(string) (io.Writer, error) { return w, nil }, &stderr)
		c := s.BattleController()
		if c.Message() != "" {
			t.Fatalf("始めに案内 %q", c.Message())
		}
		waitFriend(c)
		if c.Message() != "行動の記録を書けません（遊ぶのは続けられます）" || !c.Battle().Units()[1].Acted || strings.Count(stderr.String(), "書けない") != 1 {
			t.Fatalf("案内 %q、標準エラー %q", c.Message(), stderr.String())
		}
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 0})
		c.Click(tactics.Pos{X: 1, Y: 0})
		playOut(c)
		if c.Message() != "" || c.Battle().Result != tactics.Won || w.writes != 3 || strings.Count(stderr.String(), "書けない") != 1 {
			t.Fatalf("2 回目の案内 %q、勝敗 %v、Write %d 回", c.Message(), c.Battle().Result, w.writes)
		}
	})
}

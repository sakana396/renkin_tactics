package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// countingWriter は Write の回数を数え、fail 回目から失敗する（0 なら失敗しない）。
type countingWriter struct {
	buf    bytes.Buffer
	writes int
	fail   int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.fail > 0 && w.writes >= w.fail {
		return 0, errors.New("書けない")
	}
	return w.buf.Write(p)
}

// logStep は試験の手順 1 つ。誰が・何をしたかを、記録と比べるために残す。
type logStep struct {
	kind   string // act・end_phase・rewind
	cmd    tactics.Command
	before []tactics.Unit
	turn   int
	hist   []tactics.HistoryEntry // rewind の前の履歴
	to     uint64
	result tactics.Result
}

// playForLog は ch02 を決まった手順で遊ぶ。1 ターン目は 1 人を残して番を終え（end_phase）、
// 2 ターン目の 2 手目のあとで、その番の 1 手目の前へ巻き戻す。そのあとは攻撃できる相手がいれば攻撃し、
// いなければ待機する。再移動を待てば、その場で終える。4 ターン目の終わりか勝敗で止める。
// 操作は l（nil なら記録しない）を通す。
func playForLog(t *testing.T, cat *defs.Catalog, l *core.PlayLog, seed uint64) (*core.Campaign, *core.Sortie, []int, []tactics.Event, []logStep) {
	t.Helper()
	c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := cat.Map(cat.Chapters()[c.Chapter].Map)
	var members, slots []int
	for i := range c.Roster {
		if len(members) < len(m.Slots) {
			members = append(members, i)
			slots = append(slots, len(slots))
		}
	}
	s, events, err := c.StartBattleAt(members, slots, seed)
	if err != nil {
		t.Fatal(err)
	}
	b := s.Battle
	l.Start(c, s, slots, seed, events)
	var steps []logStep
	act := func(cmd tactics.Command) {
		t.Helper()
		st := logStep{kind: "act", cmd: cmd, before: b.Units(), turn: b.Turn}
		if _, err := l.Act(b, cmd); err != nil {
			t.Fatalf("Act(%+v): %v", cmd, err)
		}
		st.result = b.Result
		steps = append(steps, st)
		if pc, ok := b.PendingCanto(); ok {
			end := tactics.Command{Unit: pc.Unit, To: b.Units()[pc.Unit].Pos, Action: tactics.CantoEnd}
			st := logStep{kind: "act", cmd: end, before: b.Units(), turn: b.Turn}
			if _, err := l.Act(b, end); err != nil {
				t.Fatalf("CantoEnd: %v", err)
			}
			st.result = b.Result
			steps = append(steps, st)
		}
	}
	choose := func(i int) tactics.Command {
		reach := b.Reachable(i)
		var spots []tactics.Pos
		for p := range reach {
			spots = append(spots, p)
		}
		sort.Slice(spots, func(a, c int) bool {
			if spots[a].Y != spots[c].Y {
				return spots[a].Y < spots[c].Y
			}
			return spots[a].X < spots[c].X
		})
		units := b.Units()
		for _, p := range spots {
			for _, q := range b.AttackRange(i, p) {
				for j, u := range units {
					if u.Alive && u.Side == tactics.Enemy && u.Pos == q {
						cmd := tactics.Command{Unit: i, To: p, Action: tactics.Attack, Target: j}
						if _, err := b.Forecast(cmd); err == nil {
							return cmd
						}
					}
				}
			}
		}
		return tactics.Command{Unit: i, To: units[i].Pos, Action: tactics.Wait}
	}
	unacted := func() []int {
		var out []int
		for i, u := range b.Units() {
			if u.Alive && u.Side == tactics.Player && !u.Acted {
				out = append(out, i)
			}
		}
		return out
	}
	rewound := false
	for b.Result == tactics.Ongoing && b.Turn <= 4 {
		turn := b.Turn
		acted := 0
		for b.Result == tactics.Ongoing && b.Turn == turn && b.Phase == tactics.Player {
			left := unacted()
			if len(left) == 0 {
				break
			}
			if turn == 1 && len(left) == 1 {
				// 1 人を残して番を終える。
				st := logStep{kind: "end_phase", turn: b.Turn, before: b.Units()}
				if _, err := l.EndPhase(b); err != nil {
					t.Fatal(err)
				}
				st.result = b.Result
				steps = append(steps, st)
				break
			}
			act(choose(left[0]))
			acted++
			if turn == 2 && acted == 2 && !rewound {
				rewound = true
				h := b.History()
				var to uint64
				found := false
				for _, e := range h {
					if e.Turn == 2 && e.Side == tactics.Player && e.Kind == tactics.HistoryAction {
						to, found = e.ID, true
						break
					}
				}
				if !found {
					t.Fatal("2 ターン目の自軍の行動が履歴にない")
				}
				st := logStep{kind: "rewind", hist: h, to: to, turn: b.Turn}
				if err := l.Rewind(b, to); err != nil {
					t.Fatal(err)
				}
				steps = append(steps, st)
				acted = 0
			}
		}
		if b.Turn == turn && b.Phase == tactics.Player && b.Result == tactics.Ongoing {
			if len(unacted()) > 0 {
				continue
			}
			t.Fatal("全員が行動したのに番が進まない")
		}
	}
	if !rewound {
		t.Fatal("巻き戻しの手順まで進まなかった")
	}
	return c, s, slots, events, steps
}

// readLog は記録を 1 行ずつ JSON として読む。どの行も改行で終わることを確かめる。
func readLog(t *testing.T, data string) []map[string]any {
	t.Helper()
	if !strings.HasSuffix(data, "\n") {
		t.Fatal("記録が改行で終わっていない")
	}
	var out []map[string]any
	for n, line := range strings.Split(strings.TrimSuffix(data, "\n"), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("%d 行目が JSON でない: %v\n%s", n+1, err, line)
		}
		out = append(out, m)
	}
	return out
}

func num(v any) int { f, _ := v.(float64); return int(f) }

// fakeClock は偽の時計。呼ぶたびに経過の秒が 2 ずつ増える（E-106。core は実時間を読まない）。
func fakeClock() func() int {
	sec := 0
	return func() int { sec += 2; return sec }
}

func TestPlayLog(t *testing.T) {
	cat := loadBase(t)
	t.Run("TC-NFR-06-09 記録の行と項目", func(t *testing.T) {
		w := &countingWriter{}
		l := core.NewPlayLog(w, fakeClock())
		c, s, slots, _, steps := playForLog(t, cat, l, 7)
		if err := l.Err(); err != nil {
			t.Fatal(err)
		}
		recs := readLog(t, w.buf.String())
		if w.writes != len(recs) {
			t.Fatalf("Write の回数 %d と行の数 %d が違う（1 行ごとに 1 回書く）", w.writes, len(recs))
		}
		// 1 行目は start。章・種・名簿・配置・キャンペーン・初めのユニット。
		st := recs[0]
		if st["type"] != "start" || num(st["v"]) != 1 || st["chapter"] != "ch02" || st["map"] != cat.Chapters()[c.Chapter].Map || num(st["seed"]) != 7 {
			t.Fatalf("start = %v", st)
		}
		var gotMembers, gotSlots []int
		for _, v := range st["members"].([]any) {
			gotMembers = append(gotMembers, num(v))
		}
		for _, v := range st["slots"].([]any) {
			gotSlots = append(gotSlots, num(v))
		}
		if !reflect.DeepEqual(gotMembers, s.Members) || !reflect.DeepEqual(gotSlots, slots) {
			t.Fatalf("members=%v slots=%v; 期待 %v %v", gotMembers, gotSlots, s.Members, slots)
		}
		raw, _ := json.Marshal(st["campaign"])
		back, err := core.LoadCampaign(cat, bytes.NewReader(raw))
		if err != nil || back.Chapter != c.Chapter || len(back.Roster) != len(c.Roster) {
			t.Fatalf("campaign をセーブとして読めない: %v", err)
		}
		units := st["units"].([]any)
		first := steps[0].before
		if len(units) != len(first) {
			t.Fatalf("units %d 体; 期待 %d", len(units), len(first))
		}
		for i, v := range units {
			u := v.(map[string]any)
			pos := u["pos"].([]any)
			if num(u["i"]) != i || u["name"] != first[i].Name || num(u["hp"]) != first[i].HP || num(pos[0]) != first[i].Pos.X || num(pos[1]) != first[i].Pos.Y {
				t.Fatalf("units[%d] = %v; 期待 %+v", i, u, first[i])
			}
		}
		// 2 行目は 1 ターン目の自軍の番の始まり。
		if recs[1]["type"] != "phase" || num(recs[1]["turn"]) != 1 || recs[1]["side"] != "player" {
			t.Fatalf("2 行目 = %v", recs[1])
		}
		// 操作の記録は、手順と同じ順・同じ数。消えた手も残る。
		var ops []map[string]any
		for _, r := range recs {
			switch r["type"] {
			case "action", "end_phase", "rewind":
				ops = append(ops, r)
			case "start", "phase", "end":
			default:
				t.Fatalf("知らない種類 %v", r["type"])
			}
		}
		if len(ops) != len(steps) {
			t.Fatalf("操作の記録 %d 行; 手順 %d", len(ops), len(steps))
		}
		actionNames := map[tactics.Action]string{tactics.Wait: "wait", tactics.Attack: "attack", tactics.CantoEnd: "canto_end"}
		histOf := map[uint64]bool{}
		sawAttack := false
		for i, st := range steps {
			r := ops[i]
			switch st.kind {
			case "act":
				cmd := r["cmd"].(map[string]any)
				to := cmd["to"].([]any)
				u := st.before[st.cmd.Unit]
				from := r["from"].([]any)
				if r["type"] != "action" || num(r["turn"]) != st.turn || num(cmd["unit"]) != st.cmd.Unit || num(to[0]) != st.cmd.To.X || num(to[1]) != st.cmd.To.Y ||
					cmd["action"] != actionNames[st.cmd.Action] || num(cmd["target"]) != st.cmd.Target || cmd["weapon"] != nil ||
					r["name"] != u.Name || num(from[0]) != u.Pos.X || num(from[1]) != u.Pos.Y {
					t.Fatalf("手順 %d: %v; 期待 %+v（%s）", i, r, st.cmd, u.Name)
				}
				if st.cmd.Action == tactics.Attack {
					sawAttack = true
					if r["target_name"] != st.before[st.cmd.Target].Name || r["weapon_name"] != u.EquippedWeapon().Name {
						t.Fatalf("手順 %d: 相手・武器 %v", i, r)
					}
					// 攻撃の結果は events の strike に出る。
					found := false
					for _, e := range r["events"].([]any) {
						if ev := e.(map[string]any); ev["kind"] == "strike" && num(ev["unit"]) == st.cmd.Unit && num(ev["target"]) == st.cmd.Target {
							found = true
							if _, ok := ev["hit"].(bool); !ok {
								t.Fatalf("strike に hit がない: %v", ev)
							}
						}
					}
					if !found {
						t.Fatalf("手順 %d: 攻撃の strike がない: %v", i, r)
					}
				}
				if st.cmd.Action == tactics.CantoEnd {
					if r["history"] != nil {
						t.Fatalf("再移動の記録に history がある: %v", r)
					}
				} else {
					if r["history"] == nil {
						t.Fatalf("手順 %d に history がない", i)
					}
					histOf[uint64(num(r["history"]))] = true
				}
			case "end_phase":
				if r["type"] != "end_phase" || num(r["turn"]) != 1 || num(r["unacted"]) != 1 || r["history"] == nil {
					t.Fatalf("end_phase = %v", r)
				}
			case "rewind":
				undone, toTurn := 0, -1
				for k, e := range st.hist {
					if e.ID == st.to {
						toTurn = e.Turn
						for _, f := range st.hist[k:] {
							if f.Side == tactics.Player && f.Kind == tactics.HistoryAction {
								undone++
							}
						}
					}
				}
				if r["type"] != "rewind" || uint64(num(r["to"])) != st.to || num(r["to_turn"]) != toTurn || num(r["undone"]) != undone || undone != 2 || !histOf[st.to] || num(r["turn"]) != 2 {
					t.Fatalf("rewind = %v; 期待 to=%d to_turn=%d undone=%d", r, st.to, toTurn, undone)
				}
			}
		}
		if !sawAttack {
			t.Fatal("攻撃の手順がなかった（試験の手順を見直す）")
		}
		// 番の始まりは phase として、ターンごとに自軍と敵軍が出る。
		phases := map[string]bool{}
		for _, r := range recs {
			if r["type"] == "phase" {
				phases[r["side"].(string)+string(rune('0'+num(r["turn"])))] = true
			}
		}
		for _, k := range []string{"player1", "enemy1", "player2", "enemy2"} {
			if !phases[k] {
				t.Fatalf("phase %s がない: %v", k, phases)
			}
		}
		// seed 7 のこの手順は 3 ターン目に負けるので、end は最後の 1 行だけ（lost・turn 3）。
		ends := 0
		for _, r := range recs {
			if r["type"] == "end" {
				ends++
			}
		}
		last := recs[len(recs)-1]
		if s.Battle.Result != tactics.Lost || ends != 1 || last["type"] != "end" || last["result"] != "lost" || num(last["turn"]) != 3 {
			t.Fatalf("勝敗 %v、end %d 行、最後 %v; 期待 負け・1 行・lost の turn 3", s.Battle.Result, ends, last)
		}
		// 経過の秒は操作の行（start・action・end_phase・rewind）にだけ入り、偽の時計の順に 2 ずつ増える（E-106）。
		sec := 0
		for n, r := range recs {
			switch r["type"] {
			case "start", "action", "end_phase", "rewind":
				sec += 2
				if v, ok := r["elapsed"].(float64); !ok || int(v) != sec {
					t.Fatalf("%d 行目 %v の elapsed = %v; 期待 %d", n+1, r["type"], r["elapsed"], sec)
				}
			default:
				if _, ok := r["elapsed"]; ok {
					t.Fatalf("%d 行目 %v に elapsed がある", n+1, r["type"])
				}
			}
		}
	})
	t.Run("TC-NFR-06-09 時計がなければ elapsed を書かない", func(t *testing.T) {
		var buf bytes.Buffer
		playForLog(t, cat, core.NewPlayLog(&buf, nil), 7)
		if strings.Contains(buf.String(), `"elapsed"`) {
			t.Fatal("時計がないのに elapsed がある")
		}
	})
	t.Run("TC-NFR-06-09 番の終わりへの巻き戻しは end_phase の history に戻る", func(t *testing.T) {
		// ch02 で 1 人が待機し、もう 1 人を残して番を終える。その番の終わりへ巻き戻し、もう一度番を終える。
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		s, events, err := c.StartBattleAt([]int{0, 1}, []int{0, 1}, 7)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		l := core.NewPlayLog(&buf, nil)
		l.Start(c, s, []int{0, 1}, 7, events)
		b := s.Battle
		if _, err := l.Act(b, tactics.Command{Unit: 0, To: b.Units()[0].Pos, Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		if _, err := l.EndPhase(b); err != nil {
			t.Fatal(err)
		}
		var id uint64
		for _, e := range b.History() {
			if e.Kind == tactics.HistoryEndPhase && !e.Automatic {
				id = e.ID
			}
		}
		if err := l.Rewind(b, id); err != nil {
			t.Fatal(err)
		}
		if b.Turn != 1 || b.Phase != tactics.Player {
			t.Fatalf("番の終わりの前に戻っていない: turn %d", b.Turn)
		}
		if _, err := l.EndPhase(b); err != nil {
			t.Fatal(err)
		}
		var ends []map[string]any
		var rw map[string]any
		for _, r := range readLog(t, buf.String()) {
			switch r["type"] {
			case "end_phase":
				ends = append(ends, r)
			case "rewind":
				rw = r
			}
		}
		if len(ends) != 2 || rw == nil {
			t.Fatalf("end_phase %d 行、rewind %v", len(ends), rw)
		}
		// 消えた番の終わりは、rewind の to と同じ history を持つ end_phase の行。やり直した番の終わりは別の ID。
		if ends[0]["history"] == nil || uint64(num(ends[0]["history"])) != id || uint64(num(rw["to"])) != id ||
			num(rw["turn"]) != 2 || num(rw["to_turn"]) != 1 || num(rw["undone"]) != 0 ||
			ends[1]["history"] == nil || num(ends[1]["history"]) == num(ends[0]["history"]) {
			t.Fatalf("end_phase %v、rewind %v; 期待 history %d と同じ to、turn 2・to_turn 1・undone 0、やり直しは別の ID", ends, rw, id)
		}
	})
	t.Run("TC-NFR-06-09 勝ちの end と、巻き戻しのあとの end", func(t *testing.T) {
		// 主人公 1 人とボス 1 体の盤面で、1 撃で勝ち、巻き戻してもう一度勝つ。end は勝つたびに書く。
		cat := campaignCatalog()
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		s, events, err := c.StartBattle([]int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		w := &countingWriter{}
		l := core.NewPlayLog(w, nil)
		l.Start(c, s, []int{0}, 1, events)
		b := s.Battle
		if _, err := l.Act(b, tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.Attack, Target: 1}); err != nil {
			t.Fatal(err)
		}
		if b.Result != tactics.Won {
			t.Fatalf("勝たなかった: %v", b.Result)
		}
		var id uint64
		for _, e := range b.History() {
			if e.Side == tactics.Player && e.Kind == tactics.HistoryAction {
				id = e.ID
			}
		}
		if err := l.Rewind(b, id); err != nil {
			t.Fatal(err)
		}
		if _, err := l.Act(b, tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.Attack, Target: 1}); err != nil {
			t.Fatal(err)
		}
		var types []string
		for _, r := range readLog(t, w.buf.String()) {
			types = append(types, r["type"].(string))
			if r["type"] == "end" && (r["result"] != "won" || num(r["turn"]) != 1) {
				t.Fatalf("end = %v", r)
			}
		}
		want := []string{"start", "phase", "action", "end", "rewind", "action", "end"}
		if !reflect.DeepEqual(types, want) {
			t.Fatalf("種類 %v; 期待 %v", types, want)
		}
	})
	t.Run("TC-NFR-06-09 番をまたいだ巻き戻しの undone は自軍の行動だけを数える", func(t *testing.T) {
		// ch02 で 1 人が待機し、残りを動かさずに番を終える（敵軍の番と 2T の始まりまで進む）。
		// 1 手目の前へ戻すと、履歴には番の終わり・敵の行動・番の始まりもあるが、消えた自軍の行動は 1 つ。
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch02")
		if err != nil {
			t.Fatal(err)
		}
		s, events, err := c.StartBattleAt([]int{0, 1}, []int{0, 1}, 7)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		l := core.NewPlayLog(&buf, nil)
		l.Start(c, s, []int{0, 1}, 7, events)
		b := s.Battle
		if _, err := l.Act(b, tactics.Command{Unit: 0, To: b.Units()[0].Pos, Action: tactics.Wait}); err != nil {
			t.Fatal(err)
		}
		if _, err := l.EndPhase(b); err != nil {
			t.Fatal(err)
		}
		h := b.History()
		if b.Turn != 2 || len(h) < 3 {
			t.Fatalf("2T に進んでいない: turn %d、履歴 %d", b.Turn, len(h))
		}
		if err := l.Rewind(b, h[0].ID); err != nil {
			t.Fatal(err)
		}
		recs := readLog(t, buf.String())
		last := recs[len(recs)-1]
		if last["type"] != "rewind" || num(last["undone"]) != 1 || num(last["turn"]) != 2 || num(last["to_turn"]) != 1 {
			t.Fatalf("rewind = %v; 期待 undone 1・turn 2・to_turn 1", last)
		}
	})
	t.Run("TC-NFR-06-10 流し直しで同じ記録と勝敗になる", func(t *testing.T) {
		w := &countingWriter{}
		_, s, _, _, _ := playForLog(t, cat, core.NewPlayLog(w, nil), 11)
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(w.buf.String()))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Count(w.buf.String(), "\n")
		if rep.FirstDiff != 0 || rep.Lines != lines || rep.Result != s.Battle.Result || rep.Result != tactics.Lost {
			t.Fatalf("報告 %+v; 期待 食い違いなし・%d 行・%v", rep, lines, s.Battle.Result)
		}
	})
	t.Run("TC-NFR-06-10 経過の秒は流し直しで比べない", func(t *testing.T) {
		w := &countingWriter{}
		playForLog(t, cat, core.NewPlayLog(w, fakeClock()), 11)
		// 経過の秒を書き換えても、食い違いにはならない（E-106）。
		changed := strings.ReplaceAll(w.buf.String(), `"elapsed":4,`, `"elapsed":4000,`)
		if changed == w.buf.String() {
			t.Fatal("elapsed 4 の行がない")
		}
		for _, log := range []string{w.buf.String(), changed} {
			rep, err := core.ReplayPlayLog(cat, strings.NewReader(log))
			if err != nil || rep.FirstDiff != 0 || rep.Result != tactics.Lost {
				t.Fatalf("報告 %+v, %v; 期待 食い違いなし・負け", rep, err)
			}
		}
	})
	t.Run("TC-NFR-06-10 ch04 の再移動・回復・道具・レベルアップ・番の終わりへの巻き戻しを流し直す", func(t *testing.T) {
		w := &countingWriter{}
		b := playCh04ForLog(t, cat, core.NewPlayLog(w, fakeClock()), 2)
		log := w.buf.String()
		// seed 2 のこの手順は、6 ターン目まで勝敗が決まらない。どの種類も少なくとも 1 回は起きる。
		if b.Result != tactics.Ongoing || b.Turn != 7 {
			t.Fatalf("勝敗 %v、ターン %d; 期待 決まらず 7 ターン目", b.Result, b.Turn)
		}
		for _, k := range []string{`"action":"canto_move"`, `"action":"heal"`, `"action":"item"`, `"kind":"heal"`, `"kind":"level_up"`, `"type":"rewind"`, `"type":"end_phase"`} {
			if !strings.Contains(log, k) {
				t.Fatalf("記録に %s がない", k)
			}
		}
		// 巻き戻しの to は、消えた番の終わり（end_phase）の history。
		var endIDs []int
		to := -1
		for _, r := range readLog(t, log) {
			switch r["type"] {
			case "end_phase":
				endIDs = append(endIDs, num(r["history"]))
			case "rewind":
				to = num(r["to"])
			}
		}
		if len(endIDs) == 0 || endIDs[0] != to {
			t.Fatalf("rewind の to %d; end_phase の history %v", to, endIDs)
		}
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(log))
		if err != nil || rep.FirstDiff != 0 || rep.Lines != strings.Count(log, "\n") || rep.Result != tactics.Ongoing {
			t.Fatalf("報告 %+v, %v; 期待 食い違いなし・%d 行・決まらない", rep, err, strings.Count(log, "\n"))
		}
	})
	t.Run("TC-NFR-06-10 書き換えた結果の行を食い違いとして知らせる", func(t *testing.T) {
		w := &countingWriter{}
		playForLog(t, cat, core.NewPlayLog(w, nil), 11)
		lines := strings.Split(strings.TrimSuffix(w.buf.String(), "\n"), "\n")
		target := -1
		for i, line := range lines {
			if strings.Contains(line, `"kind":"strike"`) {
				target = i
				break
			}
		}
		if target < 0 {
			t.Fatal("strike の行がない")
		}
		var flipped string
		if strings.Contains(lines[target], `"hit":true`) {
			flipped = strings.Replace(lines[target], `"hit":true`, `"hit":false`, 1)
		} else {
			flipped = strings.Replace(lines[target], `"hit":false`, `"hit":true`, 1)
		}
		orig := lines[target]
		lines[target] = flipped
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(strings.Join(lines, "\n")+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if rep.FirstDiff != target+1 || rep.Want != flipped || rep.Got != orig {
			t.Fatalf("報告 FirstDiff=%d; 期待 %d", rep.FirstDiff, target+1)
		}
	})
	t.Run("TC-NFR-06-10 選んだ武器の添字も流し直す", func(t *testing.T) {
		// 主人公が 2 本目の剣を選んで攻撃する。流し直しでも同じ武器を選ぶので、同じ記録になる。
		cat := campaignCatalog(func(chars []defs.CharacterDef) { chars[0].Weapons = []string{"s", "t"} })
		c, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		s, events, err := c.StartBattle([]int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		l := core.NewPlayLog(&buf, nil)
		l.Start(c, s, []int{0}, 1, events)
		second := 1
		if _, err := l.Act(s.Battle, tactics.Command{Unit: 0, To: tactics.Pos{X: 1, Y: 0}, Action: tactics.Attack, Target: 1, Weapon: &second}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), `"weapon":1},`) || !strings.Contains(buf.String(), `"weapon_name":"もう 1 本の剣"`) {
			t.Fatalf("武器の添字と名前がない:\n%s", buf.String())
		}
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(buf.String()))
		if err != nil || rep.FirstDiff != 0 || rep.Result != tactics.Won {
			t.Fatalf("報告 %+v, %v", rep, err)
		}
	})
	t.Run("TC-NFR-06-10 読めない記録と受け付けられない操作", func(t *testing.T) {
		w := &countingWriter{}
		playForLog(t, cat, core.NewPlayLog(w, nil), 11)
		lines := strings.Split(strings.TrimSuffix(w.buf.String(), "\n"), "\n")
		act := -1
		for i, line := range lines {
			if strings.HasPrefix(line, `{"type":"action"`) {
				act = i
				break
			}
		}
		badCmd := append([]string{}, lines...)
		var rec map[string]any
		_ = json.Unmarshal([]byte(badCmd[act]), &rec)
		rec["cmd"].(map[string]any)["unit"] = 99
		data, _ := json.Marshal(rec)
		badCmd[act] = string(data)
		cases := []struct {
			name, log, line string
		}{
			{"空", "", ""},
			{"JSON でない", lines[0] + "\n{\n", "2 行目"},
			{"start で始まらない", lines[1] + "\n", "1 行目"},
			{"start が 2 つ", lines[0] + "\n" + lines[0] + "\n", "2 行目"},
			{"知らない種類", lines[0] + "\n" + `{"type":"jump"}` + "\n", "2 行目"},
			{"版が違う", strings.Replace(lines[0], `"v":1`, `"v":2`, 1) + "\n", "1 行目"},
		}
		for _, tc := range cases {
			_, err := core.ReplayPlayLog(cat, strings.NewReader(tc.log))
			if !errors.Is(err, core.ErrInvalidPlayLog) || !strings.Contains(err.Error(), tc.line) {
				t.Errorf("%s: 誤り %v; 期待 ErrInvalidPlayLog と %q", tc.name, err, tc.line)
			}
		}
		_, err := core.ReplayPlayLog(cat, strings.NewReader(strings.Join(badCmd[:act+1], "\n")+"\n"))
		if want := strconv.Itoa(act+1) + " 行目"; err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ユニットがいない操作: %v; 期待 %q", err, want)
		}
	})
	t.Run("TC-NFR-06-11 書けなくなっても戦闘は同じに進み、以後は書かない", func(t *testing.T) {
		// 記録しない遊びと、3 回目の Write から失敗する遊びと、記録の口がない（nil）遊びを比べる。
		plain := &countingWriter{}
		_, s1, _, _, _ := playForLog(t, cat, core.NewPlayLog(plain, nil), 5)
		broken := &countingWriter{fail: 3}
		lb := core.NewPlayLog(broken, nil)
		_, s2, _, _, _ := playForLog(t, cat, lb, 5)
		_, s3, _, _, _ := playForLog(t, cat, nil, 5)
		if !reflect.DeepEqual(s1.Battle.Units(), s2.Battle.Units()) || !reflect.DeepEqual(s1.Battle.Units(), s3.Battle.Units()) ||
			s1.Battle.Turn != s2.Battle.Turn || s1.Battle.Turn != s3.Battle.Turn ||
			s1.Battle.Result != s2.Battle.Result || s1.Battle.Result != s3.Battle.Result {
			t.Fatal("記録の書き出しで戦闘の結果が変わった")
		}
		if lb.Err() == nil || !strings.Contains(lb.Err().Error(), "書けない") {
			t.Fatalf("Err = %v", lb.Err())
		}
		if broken.writes != 3 || strings.Count(broken.buf.String(), "\n") != 2 {
			t.Fatalf("失敗のあとも書いた: Write %d 回、%d 行", broken.writes, strings.Count(broken.buf.String(), "\n"))
		}
		var nilLog *core.PlayLog
		if nilLog.Err() != nil {
			t.Fatal("nil の Err")
		}
		// 受け付けられない操作・番を終える・巻き戻しは、誤りを返し、記録に書かない。
		ok := &countingWriter{}
		lo := core.NewPlayLog(ok, nil)
		_, s4, _, _, _ := playForLog(t, cat, lo, 5)
		writes := ok.writes
		b := s4.Battle
		if _, err := lo.Act(b, tactics.Command{Unit: 99, Action: tactics.Wait}); err == nil {
			t.Fatal("ないユニットの操作を受け付けた")
		}
		if b.Result != tactics.Ongoing {
			if _, err := lo.EndPhase(b); err == nil {
				t.Fatal("終わった戦闘で番を終えられた")
			}
		}
		if err := lo.Rewind(b, 1<<40); err == nil {
			t.Fatal("ない履歴へ巻き戻せた")
		}
		if ok.writes != writes || lo.Err() != nil {
			t.Fatalf("失敗した操作を書いた: Write %d → %d", writes, ok.writes)
		}
	})
}

// playCh04ForLog は ch04 を、再移動・回復・道具・レベルアップ・番の終わりへの巻き戻しが起きる決まった手順で遊ぶ（TC-NFR-06-10）。
// 名簿の経験を 99 にしておき、経験を得たらレベルが上がるようにする（記録の campaign に入るので流し直しでも同じ）。
// 自軍は添字の順に、(1) 傷ついた味方を杖で回復できれば回復、(2) 自分が傷ついていて道具を持てば 1 つ目の道具、
// (3) 攻撃できる相手がいれば攻撃、(4) なければ待機。再移動を待てば、移動できる最初のマス（y、x の順）へ動く。
// 2 ターン目は 1 手のあとに番を終え、その番の終わりへ巻き戻してから続ける。6 ターン目か勝敗で止める。
func playCh04ForLog(t *testing.T, cat *defs.Catalog, l *core.PlayLog, seed uint64) *tactics.Battle {
	t.Helper()
	c, err := core.NewCampaignAt(cat, core.Withdraw, "ch04")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := cat.Map(cat.Chapters()[c.Chapter].Map)
	var members, slots []int
	for i := range c.Roster {
		c.Roster[i].Exp = 99
		if len(members) < len(m.Slots) {
			members = append(members, i)
			slots = append(slots, len(slots))
		}
	}
	s, events, err := c.StartBattleAt(members, slots, seed)
	if err != nil {
		t.Fatal(err)
	}
	b := s.Battle
	l.Start(c, s, slots, seed, events)
	sorted := func(reach map[tactics.Pos]int, moved bool) []tactics.Pos {
		var spots []tactics.Pos
		for p, cost := range reach {
			if !moved || cost > 0 {
				spots = append(spots, p)
			}
		}
		sort.Slice(spots, func(a, c int) bool {
			if spots[a].Y != spots[c].Y {
				return spots[a].Y < spots[c].Y
			}
			return spots[a].X < spots[c].X
		})
		return spots
	}
	try := func(cmd tactics.Command) bool {
		_, err := l.Act(b, cmd)
		return err == nil
	}
	actOne := func(i int) {
		t.Helper()
		units := b.Units()
		u := units[i]
		spots := sorted(b.Reachable(i), false)
		done := false
		for j, v := range units {
			if done || !v.Alive || v.Side != tactics.Player || j == i || v.HP >= v.Stats.HP {
				continue
			}
			for _, p := range spots {
				cmd := tactics.Command{Unit: i, To: p, Action: tactics.Heal, Target: j}
				if _, err := b.HealForecast(cmd); err == nil && try(cmd) {
					done = true
					break
				}
			}
		}
		if !done && u.HP < u.Stats.HP && len(u.Items) > 0 {
			done = try(tactics.Command{Unit: i, To: u.Pos, Action: tactics.UseItem, Item: 0})
		}
		for _, p := range spots {
			if done {
				break
			}
			for _, q := range b.AttackRange(i, p) {
				for j, v := range units {
					if !done && v.Alive && v.Side == tactics.Enemy && v.Pos == q {
						cmd := tactics.Command{Unit: i, To: p, Action: tactics.Attack, Target: j}
						if _, err := b.Forecast(cmd); err == nil {
							done = try(cmd)
						}
					}
				}
			}
		}
		if !done && !try(tactics.Command{Unit: i, To: u.Pos, Action: tactics.Wait}) {
			t.Fatalf("ユニット %d が行動できない", i)
		}
		if pc, ok := b.PendingCanto(); ok {
			if to := sorted(b.CantoReachable(), true); len(to) > 0 {
				if !try(tactics.Command{Unit: pc.Unit, To: to[0], Action: tactics.CantoMove}) {
					t.Fatal("再移動できない")
				}
			} else if !try(tactics.Command{Unit: pc.Unit, To: b.Units()[pc.Unit].Pos, Action: tactics.CantoEnd}) {
				t.Fatal("再移動を終えられない")
			}
		}
	}
	endRewound := false
	for b.Result == tactics.Ongoing && b.Turn <= 6 {
		turn := b.Turn
		acted := 0
		for b.Result == tactics.Ongoing && b.Turn == turn && b.Phase == tactics.Player {
			next := -1
			for i, u := range b.Units() {
				if u.Alive && u.Side == tactics.Player && !u.Acted {
					next = i
					break
				}
			}
			if next < 0 {
				t.Fatal("全員が行動したのに番が進まない")
			}
			if turn == 2 && acted == 1 && !endRewound {
				endRewound = true
				if _, err := l.EndPhase(b); err != nil {
					t.Fatal(err)
				}
				var id uint64
				for _, e := range b.History() {
					if e.Kind == tactics.HistoryEndPhase && !e.Automatic && e.Turn == 2 {
						id = e.ID
					}
				}
				if err := l.Rewind(b, id); err != nil {
					t.Fatalf("番の終わりへ巻き戻せない: %v", err)
				}
				continue
			}
			actOne(next)
			acted++
		}
	}
	if !endRewound {
		t.Fatal("番の終わりへの巻き戻しまで進まなかった")
	}
	return b
}

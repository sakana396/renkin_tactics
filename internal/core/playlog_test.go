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

func TestPlayLog(t *testing.T) {
	cat := loadBase(t)
	t.Run("TC-NFR-06-09 記録の行と項目", func(t *testing.T) {
		w := &countingWriter{}
		l := core.NewPlayLog(w)
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
				if r["type"] != "end_phase" || num(r["turn"]) != 1 || num(r["unacted"]) != 1 {
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
		// 勝敗が決まったら end が 1 行。決まらなければない。
		ends := 0
		for _, r := range recs {
			if r["type"] == "end" {
				ends++
			}
		}
		if want := map[bool]int{true: 1, false: 0}[s.Battle.Result != tactics.Ongoing]; ends != want {
			t.Fatalf("end %d 行; 期待 %d", ends, want)
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
		l := core.NewPlayLog(w)
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
		l := core.NewPlayLog(&buf)
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
		_, s, _, _, _ := playForLog(t, cat, core.NewPlayLog(w), 11)
		rep, err := core.ReplayPlayLog(cat, strings.NewReader(w.buf.String()))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Count(w.buf.String(), "\n")
		if rep.FirstDiff != 0 || rep.Lines != lines || rep.Result != s.Battle.Result {
			t.Fatalf("報告 %+v; 期待 食い違いなし・%d 行・%v", rep, lines, s.Battle.Result)
		}
	})
	t.Run("TC-NFR-06-10 書き換えた結果の行を食い違いとして知らせる", func(t *testing.T) {
		w := &countingWriter{}
		playForLog(t, cat, core.NewPlayLog(w), 11)
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
		l := core.NewPlayLog(&buf)
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
		playForLog(t, cat, core.NewPlayLog(w), 11)
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
		_, s1, _, _, _ := playForLog(t, cat, core.NewPlayLog(plain), 5)
		broken := &countingWriter{fail: 3}
		lb := core.NewPlayLog(broken)
		_, s2, _, _, _ := playForLog(t, cat, lb, 5)
		_, s3, _, _, _ := playForLog(t, cat, nil, 5)
		if !reflect.DeepEqual(s1.Battle.Units(), s2.Battle.Units()) || !reflect.DeepEqual(s1.Battle.Units(), s3.Battle.Units()) ||
			s1.Battle.Turn != s2.Battle.Turn || s1.Battle.Turn != s3.Battle.Turn || s1.Battle.Result != s2.Battle.Result {
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
		lo := core.NewPlayLog(ok)
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

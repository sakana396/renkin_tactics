package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// PlayLogVersion は行動の記録の形の版（DES-LOG-01）。
const PlayLogVersion = 1

// ErrInvalidPlayLog は、読めない行動の記録（DES-LOG-03）。
var ErrInvalidPlayLog = errors.New("core: 行動の記録が正しくありません")

// PlayLog は試遊の行動を 1 行 1 記録の JSON（JSON Lines）で w へ書く（DES-LOG-01）。
//
// ファイルは扱わない。書き出し先（ファイルへの追記）は呼び出し元が w として渡す（DES-LOG-02）。
// 記録 1 つごとに w.Write を 1 回呼ぶので、途中でやめても、それまでの行は残る。
// 書けなかったら以後は書かず、最初の誤りを Err で返す。戦闘の状態は書けたかどうかで変わらない。
//
// Act・EndPhase・Rewind は、nil の PlayLog でも戦闘の操作だけを行う。
//
// core は実時間を扱わない（NFR-05）。経過の秒は、呼び出し元（画面）が時計を読む関数 elapsed として渡す。
type PlayLog struct {
	w       io.Writer
	elapsed func() int
	err     error
	ended   bool
}

// NewPlayLog は w へ書く PlayLog を作る（DES-LOG-01）。
// elapsed は出撃してからの経過の秒を返す関数で、操作の行（start・action・end_phase・rewind）の elapsed に入れる。
// nil なら elapsed を書かない。
func NewPlayLog(w io.Writer, elapsed func() int) *PlayLog { return &PlayLog{w: w, elapsed: elapsed} }

// now は操作の行に入れる経過の秒（DES-LOG-01）。時計がなければ nil。
func (l *PlayLog) now() *int {
	if l.elapsed == nil {
		return nil
	}
	return intPtr(l.elapsed())
}

// Err は最初に書けなかった誤りを返す（DES-LOG-02）。
func (l *PlayLog) Err() error {
	if l == nil {
		return nil
	}
	return l.err
}

// logPos はマスの [x, y]。
type logPos [2]int

func toLogPos(p tactics.Pos) *logPos { return &logPos{p.X, p.Y} }

// logStats は能力。0 の能力は省く（レベルアップで上がった能力）。
type logStats struct {
	HP  int `json:"hp,omitempty"`
	Str int `json:"str,omitempty"`
	Skl int `json:"skl,omitempty"`
	Spd int `json:"spd,omitempty"`
	Def int `json:"def,omitempty"`
	Lck int `json:"lck,omitempty"`
	Mag int `json:"mag,omitempty"`
	Res int `json:"res,omitempty"`
	Mov int `json:"mov,omitempty"`
}

// logUnit は戦闘の始めのユニット 1 体（DES-LOG-01）。I は Units の添字。
type logUnit struct {
	I        int      `json:"i"`
	Name     string   `json:"name"`
	Side     string   `json:"side"`
	Class    string   `json:"class"`
	ID       string   `json:"id,omitempty"`
	Group    string   `json:"group,omitempty"`
	Level    int      `json:"level"`
	HP       int      `json:"hp"`
	MaxHP    int      `json:"max_hp"`
	Pos      logPos   `json:"pos"`
	Weapons  []string `json:"weapons"`
	Equipped int      `json:"equipped"`
	Items    []string `json:"items,omitempty"`
	Lord     bool     `json:"lord,omitempty"`
	Boss     bool     `json:"boss,omitempty"`
}

// logEvent は出来事 1 つ（DES-LOG-01）。Kind ごとに決まった項目だけを書く。
type logEvent struct {
	Kind   string    `json:"kind"`
	Unit   *int      `json:"unit,omitempty"`
	Name   string    `json:"name,omitempty"`
	Target *int      `json:"target,omitempty"`
	From   *logPos   `json:"from,omitempty"`
	To     *logPos   `json:"to,omitempty"`
	Hit    *bool     `json:"hit,omitempty"`
	Crit   bool      `json:"crit,omitempty"`
	Pavise bool      `json:"pavise,omitempty"`
	Amount *int      `json:"amount,omitempty"`
	HP     *int      `json:"hp,omitempty"`
	Item   string    `json:"item,omitempty"`
	Level  int       `json:"level,omitempty"`
	Gains  *logStats `json:"gains,omitempty"`
	Event  string    `json:"event,omitempty"`
	Group  string    `json:"group,omitempty"`
}

// logCommand は Act に渡した操作そのもの（DES-LOG-01）。流し直しはこの値だけを使う（DES-LOG-03）。
type logCommand struct {
	Unit   int    `json:"unit"`
	To     logPos `json:"to"`
	Action string `json:"action"`
	Target int    `json:"target"`
	Item   int    `json:"item"`
	Weapon *int   `json:"weapon"`
}

// logRecord は記録 1 行（DES-LOG-01）。Type ごとに決まった項目だけを書く。
type logRecord struct {
	Type string `json:"type"`
	V    int    `json:"v,omitempty"`
	// 操作の行（start・action・end_phase・rewind）の、出撃してからの経過の秒（E-106）。
	Elapsed *int `json:"elapsed,omitempty"`
	// start の項目。
	Chapter  string          `json:"chapter,omitempty"`
	Map      string          `json:"map,omitempty"`
	Seed     *uint64         `json:"seed,omitempty"`
	Members  []int           `json:"members,omitempty"`
	Slots    []int           `json:"slots,omitempty"`
	Campaign json.RawMessage `json:"campaign,omitempty"`
	Units    []logUnit       `json:"units,omitempty"`
	// 番とターン。
	Turn *int   `json:"turn,omitempty"`
	Side string `json:"side,omitempty"`
	// action・end_phase の項目。巻き戻しの戻り先になる履歴の ID。
	History *uint64 `json:"history,omitempty"`
	// action の項目。
	Cmd        *logCommand `json:"cmd,omitempty"`
	Name       string      `json:"name,omitempty"`
	From       *logPos     `json:"from,omitempty"`
	TargetName string      `json:"target_name,omitempty"`
	WeaponName string      `json:"weapon_name,omitempty"`
	ItemName   string      `json:"item_name,omitempty"`
	// end_phase の項目。
	Unacted *int `json:"unacted,omitempty"`
	// rewind の項目。
	To     *uint64 `json:"to,omitempty"`
	ToTurn *int    `json:"to_turn,omitempty"`
	Undone *int    `json:"undone,omitempty"`
	// rewind・end の項目。
	RewindsLeft *int `json:"rewinds_left,omitempty"`
	// end の項目。
	Result string `json:"result,omitempty"`
	// 出来事。start・action・end_phase は最初の番の始まりの前まで、phase はその番の出来事。
	Events []logEvent `json:"events,omitempty"`
}

func intPtr(v int) *int { return &v }

var sideNames = map[tactics.Side]string{tactics.Player: "player", tactics.Enemy: "enemy"}

var actionNames = map[tactics.Action]string{
	tactics.Wait: "wait", tactics.Attack: "attack", tactics.UseItem: "item", tactics.Heal: "heal",
	tactics.CantoMove: "canto_move", tactics.CantoEnd: "canto_end",
}

var resultNames = map[tactics.Result]string{tactics.Won: "won", tactics.Lost: "lost"}

var eventNames = map[tactics.EventKind]string{
	tactics.EventPhaseStart: "phase_start", tactics.EventReinforce: "reinforce", tactics.EventHeal: "heal",
	tactics.EventMove: "move", tactics.EventStrike: "strike", tactics.EventFall: "fall", tactics.EventWait: "wait",
	tactics.EventWon: "won", tactics.EventLost: "lost", tactics.EventExp: "exp", tactics.EventLevelUp: "level_up",
	tactics.EventTrigger: "trigger", tactics.EventGroupCharge: "group_charge",
}

// toLogEvent は出来事を記録の形にする（DES-LOG-01）。
func toLogEvent(e tactics.Event) logEvent {
	out := logEvent{Kind: eventNames[e.Kind]}
	hpOf := func(i int) *int {
		if i < 0 || i >= len(e.Units) {
			return nil
		}
		return intPtr(e.Units[i].HP)
	}
	switch e.Kind {
	case tactics.EventReinforce:
		out.Unit, out.To = intPtr(e.Unit), toLogPos(e.To)
		if e.Unit >= 0 && e.Unit < len(e.Units) {
			out.Name = e.Units[e.Unit].Name
		}
	case tactics.EventHeal:
		out.Unit, out.Amount, out.Item, out.HP = intPtr(e.Unit), intPtr(e.Amount), e.Item, hpOf(e.Unit)
	case tactics.EventMove:
		out.Unit, out.From, out.To = intPtr(e.Unit), toLogPos(e.From), toLogPos(e.To)
	case tactics.EventStrike:
		hit := e.Hit
		out.Unit, out.Target, out.Hit, out.Amount, out.HP = intPtr(e.Unit), intPtr(e.Target), &hit, intPtr(e.Amount), hpOf(e.Target)
		out.Crit, out.Pavise = e.Crit, e.Pavise
	case tactics.EventFall, tactics.EventWait:
		out.Unit = intPtr(e.Unit)
	case tactics.EventExp:
		out.Unit, out.Amount = intPtr(e.Unit), intPtr(e.Amount)
	case tactics.EventLevelUp:
		g := logStats(e.Gains)
		out.Unit, out.Level, out.Gains = intPtr(e.Unit), e.Level, &g
	case tactics.EventTrigger:
		out.Event = e.MapEvent
	case tactics.EventGroupCharge:
		out.Event, out.Group = e.MapEvent, e.Group
	}
	return out
}

// write は記録 1 行を書く（DES-LOG-01・02）。書けなかったら以後は書かない。
func (l *PlayLog) write(r logRecord) {
	if l == nil || l.err != nil {
		return
	}
	data, err := json.Marshal(r)
	if err != nil {
		l.err = fmt.Errorf("core: 行動の記録を JSON に変換できません: %w", err)
		return
	}
	data = append(data, '\n')
	n, err := l.w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		l.err = fmt.Errorf("core: 行動の記録を書けません: %w", err)
	}
}

// writeEvents は、最初の番の始まりの前の出来事を head に入れて書き、そのあとは番ごとに phase の記録を書く（DES-LOG-01）。
// 勝敗が決まったら end の記録を 1 回だけ書く。
func (l *PlayLog) writeEvents(b *tactics.Battle, head logRecord, events []tactics.Event) {
	if l == nil {
		return
	}
	i := 0
	for ; i < len(events) && events[i].Kind != tactics.EventPhaseStart; i++ {
		head.Events = append(head.Events, toLogEvent(events[i]))
	}
	l.write(head)
	for i < len(events) {
		start := events[i]
		r := logRecord{Type: "phase", Turn: intPtr(start.Turn), Side: sideNames[start.Side]}
		for i++; i < len(events) && events[i].Kind != tactics.EventPhaseStart; i++ {
			r.Events = append(r.Events, toLogEvent(events[i]))
		}
		l.write(r)
	}
	if b.Result != tactics.Ongoing && !l.ended {
		l.ended = true
		l.write(logRecord{Type: "end", Result: resultNames[b.Result], Turn: intPtr(b.Turn), RewindsLeft: intPtr(b.RewindsLeft())})
	}
}

// Start は出撃の直後に、章・種・出撃の名簿と配置・名簿とキャンペーンの状態・初めのユニットを書く（DES-LOG-01）。
// c は StartBattleAt を呼んだキャンペーン、members・slots・seed はそのときの引数、events は返った出来事。
// キャンペーンの状態はセーブと同じ形で入れ、流し直しで同じ戦闘を作るのに使う（DES-LOG-03）。
func (l *PlayLog) Start(c *Campaign, s *Sortie, slots []int, seed uint64, events []tactics.Event) {
	if l == nil {
		return
	}
	var save bytes.Buffer
	if err := c.Save(&save); err != nil {
		l.err = err
		return
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, save.Bytes()); err != nil {
		l.err = fmt.Errorf("core: 行動の記録にセーブを入れられません: %w", err)
		return
	}
	ch := c.cat.Chapters()[s.chapter]
	r := logRecord{Type: "start", V: PlayLogVersion, Elapsed: l.now(), Chapter: ch.ID, Map: ch.Map, Seed: &seed,
		Members: append([]int{}, s.members...), Slots: append([]int{}, slots...), Campaign: compact.Bytes()}
	for i, u := range s.Battle.Units() {
		lu := logUnit{I: i, Name: u.Name, Side: sideNames[u.Side], Class: u.ClassName, ID: u.ID, Group: u.Group, Level: u.Level,
			HP: u.HP, MaxHP: u.Stats.HP, Pos: logPos{u.Pos.X, u.Pos.Y}, Weapons: []string{}, Equipped: u.Equipped, Lord: u.Lord, Boss: u.Boss}
		for _, w := range u.Weapons {
			lu.Weapons = append(lu.Weapons, w.Name)
		}
		for _, it := range u.Items {
			lu.Items = append(lu.Items, fmt.Sprintf("%s×%d", it.Item.Name, it.Count))
		}
		r.Units = append(r.Units, lu)
	}
	l.writeEvents(s.Battle, r, events)
}

// Act は b.Act(cmd) を行い、成功したら action の記録を書く（DES-LOG-01）。
// 返り値は b.Act のもの。失敗した操作は書かない（状態が変わらないため）。
func (l *PlayLog) Act(b *tactics.Battle, cmd tactics.Command) ([]tactics.Event, error) {
	if l == nil {
		return b.Act(cmd)
	}
	before := b.Units()
	turn := b.Turn
	start := len(b.History())
	events, err := b.Act(cmd)
	if err != nil {
		return events, err
	}
	lc := &logCommand{Unit: cmd.Unit, To: logPos{cmd.To.X, cmd.To.Y}, Action: actionNames[cmd.Action], Target: cmd.Target, Item: cmd.Item}
	if cmd.Weapon != nil {
		lc.Weapon = intPtr(*cmd.Weapon)
	}
	r := logRecord{Type: "action", Elapsed: l.now(), Turn: intPtr(turn), Cmd: lc}
	// 巻き戻しの戻り先になる履歴の ID。再移動は前の行動と同じ 1 手なので持たない（DES-BTL-10）。
	for _, h := range b.History()[min(start, len(b.History())):] {
		if h.Side == tactics.Player && h.Kind == tactics.HistoryAction {
			id := h.ID
			r.History = &id
			break
		}
	}
	if cmd.Unit >= 0 && cmd.Unit < len(before) {
		u := before[cmd.Unit]
		r.Name, r.From = u.Name, toLogPos(u.Pos)
		switch {
		case cmd.Weapon != nil && *cmd.Weapon >= 0 && *cmd.Weapon < len(u.Weapons):
			r.WeaponName = u.Weapons[*cmd.Weapon].Name
		case cmd.Action == tactics.Attack:
			r.WeaponName = u.EquippedWeapon().Name
		}
		if cmd.Action == tactics.UseItem && cmd.Item >= 0 && cmd.Item < len(u.Items) {
			r.ItemName = u.Items[cmd.Item].Item.Name
		}
	}
	if (cmd.Action == tactics.Attack || cmd.Action == tactics.Heal) && cmd.Target >= 0 && cmd.Target < len(before) {
		r.TargetName = before[cmd.Target].Name
	}
	l.writeEvents(b, r, events)
	return events, nil
}

// EndPhase は b.EndPhase() を行い、成功したら end_phase の記録を書く（DES-LOG-01）。
// Unacted は番を終えたときにまだ行動していなかった自軍の数。History はこの番の終わりの履歴 ID（巻き戻しの戻り先）。
func (l *PlayLog) EndPhase(b *tactics.Battle) ([]tactics.Event, error) {
	if l == nil {
		return b.EndPhase()
	}
	turn, unacted := b.Turn, 0
	for _, u := range b.Units() {
		if u.Alive && u.Side == tactics.Player && !u.Acted {
			unacted++
		}
	}
	start := len(b.History())
	events, err := b.EndPhase()
	if err != nil {
		return events, err
	}
	r := logRecord{Type: "end_phase", Elapsed: l.now(), Turn: intPtr(turn), Unacted: intPtr(unacted)}
	for _, h := range b.History()[min(start, len(b.History())):] {
		if h.Kind == tactics.HistoryEndPhase && !h.Automatic {
			id := h.ID
			r.History = &id
			break
		}
	}
	l.writeEvents(b, r, events)
	return events, nil
}

// Rewind は b.Rewind(id) を行い、成功したら rewind の記録を書く（DES-LOG-01）。
// 消えた手の記録は消さない。Undone は消えた自軍の行動の数、ToTurn は戻った先のターン。
func (l *PlayLog) Rewind(b *tactics.Battle, id uint64) error {
	if l == nil {
		return b.Rewind(id)
	}
	turn := b.Turn
	h := b.History()
	if err := b.Rewind(id); err != nil {
		return err
	}
	r := logRecord{Type: "rewind", Elapsed: l.now(), Turn: intPtr(turn), To: &id, RewindsLeft: intPtr(b.RewindsLeft())}
	for i, e := range h {
		if e.ID != id {
			continue
		}
		undone := 0
		for _, f := range h[i:] {
			if f.Side == tactics.Player && f.Kind == tactics.HistoryAction {
				undone++
			}
		}
		r.ToTurn, r.Undone = intPtr(e.Turn), intPtr(undone)
		break
	}
	// 戻った先では勝敗が決まっていないので、次に決まったときにまた end を書く。
	l.ended = b.Result != tactics.Ongoing
	l.write(r)
	return nil
}

// ReplayReport は流し直しの結果（DES-LOG-03）。
type ReplayReport struct {
	// Lines は読んだ記録の行の数。
	Lines int
	// FirstDiff は、流し直して書いた記録が元の記録と最初に食い違った行（1 から数える）。0 なら食い違いはない。
	FirstDiff int
	// Want・Got は FirstDiff の行の元の記録と流し直しの記録（どちらかがなければ空）。
	Want, Got string
	// Result は流し直したあとの勝敗。
	Result tactics.Result
}

// ReplayPlayLog は記録の start から同じ戦闘を作り、action・end_phase・rewind の操作を順に入れ直す（DES-LOG-03）。
//
// 流し直しの間も同じ PlayLog で記録を書き、元の記録と行ごとに比べる。同じ種と操作なら同じ結果になる（NFR-06）ので、
// 経過の秒（elapsed）は比べない。流し直しの行には、元の行の elapsed をそのまま入れる（E-106）。
// 定義データを変えていなければ食い違いはない。章を直したあとなら、最初に結果が変わった行が分かる。
// 操作が受け付けられなかったら、そこで止めて行の番号つきの誤りを返す（それまでの報告も返す）。
func ReplayPlayLog(cat *defs.Catalog, r io.Reader) (ReplayReport, error) {
	var rep ReplayReport
	var want []string
	var got bytes.Buffer
	// elapsed は今読んでいる元の行の経過の秒。
	var elapsed *int
	log := NewPlayLog(&got, nil)
	var b *tactics.Battle
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024*1024)
	fail := func(line int, format string, args ...any) error {
		return fmt.Errorf("%w: %d 行目: %s", ErrInvalidPlayLog, line, fmt.Sprintf(format, args...))
	}
	for sc.Scan() {
		line := sc.Text()
		want = append(want, line)
		n := len(want)
		var rec logRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return rep.finish(want, got.String(), b, false), fail(n, "JSON として読めません: %v", err)
		}
		if n == 1 && rec.Type != "start" {
			return rep.finish(want, got.String(), b, false), fail(n, "最初の行が start ではありません")
		}
		log.elapsed = nil
		if elapsed = rec.Elapsed; elapsed != nil {
			log.elapsed = func() int { return *elapsed }
		}
		var err error
		switch rec.Type {
		case "start":
			if n != 1 {
				return rep.finish(want, got.String(), b, false), fail(n, "start は最初の行だけです")
			}
			if rec.V != PlayLogVersion {
				return rep.finish(want, got.String(), b, false), fail(n, "版 %d には対応していません", rec.V)
			}
			if rec.Seed == nil {
				return rep.finish(want, got.String(), b, false), fail(n, "seed がありません")
			}
			var c *Campaign
			if c, err = LoadCampaign(cat, bytes.NewReader(rec.Campaign)); err != nil {
				break
			}
			var s *Sortie
			var events []tactics.Event
			if s, events, err = c.StartBattleAt(rec.Members, rec.Slots, *rec.Seed); err != nil {
				break
			}
			b = s.Battle
			log.Start(c, s, rec.Slots, *rec.Seed, events)
		case "action":
			var cmd tactics.Command
			if cmd, err = rec.command(); err == nil {
				_, err = log.Act(b, cmd)
			}
		case "end_phase":
			_, err = log.EndPhase(b)
		case "rewind":
			if rec.To == nil {
				err = errors.New("to がありません")
			} else {
				err = log.Rewind(b, *rec.To)
			}
		case "phase", "end":
			// 操作のあとに書かれる記録。流し直しの記録と比べるだけ。
		default:
			err = fmt.Errorf("知らない種類 %q です", rec.Type)
		}
		if err != nil {
			return rep.finish(want, got.String(), b, false), fail(n, "%v", err)
		}
	}
	if err := sc.Err(); err != nil {
		return rep.finish(want, got.String(), b, false), fmt.Errorf("%w: %v", ErrInvalidPlayLog, err)
	}
	if len(want) == 0 {
		return rep, fmt.Errorf("%w: 記録が空です", ErrInvalidPlayLog)
	}
	if err := log.Err(); err != nil {
		return rep.finish(want, got.String(), b, false), err
	}
	return rep.finish(want, got.String(), b, true), nil
}

// finish は元の記録と流し直しの記録を行ごとに比べ、最初の食い違いを報告に入れる（DES-LOG-03）。
// complete が偽（途中で止めた）なら、両方にある行だけを比べる。
func (rep ReplayReport) finish(want []string, got string, b *tactics.Battle, complete bool) ReplayReport {
	rep.Lines = len(want)
	var gotLines []string
	if got != "" {
		gotLines = strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	}
	n := max(len(want), len(gotLines))
	if !complete {
		n = min(len(want), len(gotLines))
	}
	for i := range n {
		var w, g string
		if i < len(want) {
			w = want[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			rep.FirstDiff, rep.Want, rep.Got = i+1, w, g
			break
		}
	}
	if b != nil {
		rep.Result = b.Result
	}
	return rep
}

// command は action の記録から Act に渡す操作を作る（DES-LOG-03）。
func (r logRecord) command() (tactics.Command, error) {
	if r.Cmd == nil {
		return tactics.Command{}, errors.New("cmd がありません")
	}
	cmd := tactics.Command{Unit: r.Cmd.Unit, To: tactics.Pos{X: r.Cmd.To[0], Y: r.Cmd.To[1]}, Target: r.Cmd.Target, Item: r.Cmd.Item}
	found := false
	for a, name := range actionNames {
		if name == r.Cmd.Action {
			cmd.Action, found = a, true
		}
	}
	if !found {
		return tactics.Command{}, fmt.Errorf("知らない行動 %q です", r.Cmd.Action)
	}
	if r.Cmd.Weapon != nil {
		cmd.Weapon = intPtr(*r.Cmd.Weapon)
	}
	return cmd, nil
}

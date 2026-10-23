package ui

import (
	"fmt"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// State は入力の状態（DES-UI-02）。
type State int

const (
	// Idle は何も選んでいない状態。
	Idle State = iota
	// Selected は、行動していない自軍を選び、移動先を待っている状態。
	Selected
	// Moved は、移動先を選び、行動の一覧を出している状態。まだ Act を呼んでいない。
	Moved
	// Targeting は、攻撃の相手を選んでいる状態。
	Targeting
	// ItemMenu は、使うアイテムを選んでいる状態。
	ItemMenu
	// Playing は、出来事を再生している状態。
	Playing
	// Over は、勝ち負けが決まった状態。
	Over
)

// HealTargeting は回復する味方を選ぶ状態（DES-UI-18）。
const HealTargeting State = Over + 1

// FramesPerEvent は、出来事 1 つを表示するフレームの数（DES-UI-04）。
const FramesPerEvent = 20

// FastFramesPerEvent は、早送り（Space か Enter を押している間）に出来事 1 つを表示するフレームの数（DES-UI-04）。
const FastFramesPerEvent = 4

// LevelUpFrames はレベルアップの帯を表示するフレーム数（DES-UI-16）。
const LevelUpFrames = 60

// Factory は Controller の初期化に渡す戦闘を作る（DES-UI-09）。
type Factory func() (*tactics.Battle, []tactics.Event, error)

// Controller は入力の状態の遷移を持つ（DES-UI-02、DES-UI-04）。
//
// 描画と Ebitengine から分けてあり、マウスとキーの入力は Game が Controller のメソッドに読み替える。
// 時間はフレーム数（Tick）で数え、core には渡さない。
type Controller struct {
	// 巻き戻しの表示と通し出来事の境界を保持する（DES-UI-19）。
	rewind rewindMenu
	record []tactics.Event
	battle *tactics.Battle
	state  State

	// unit は選んだ自軍、dest は移動先、target は攻撃・回復の相手（いなければ -1）。
	unit   int
	dest   tactics.Pos
	target int
	// watched は Idle でクリックして範囲を出し続けている敵（いなければ -1）。
	watched int
	// cursorX と cursorY は、カーソルの画面の座標（DES-UI-08）。まだ知らなければ盤面の外の (-1, -1)。
	cursorX, cursorY int
	// danger は、Q キーで敵の危険範囲を出していること（DES-UI-08）。もう一度押すまで、番をまたいでも残す。
	danger bool
	// confirmEnd は、行動していない自軍がいるときに E を 1 回押したこと。
	confirmEnd bool

	// events は再生中の出来事、index は表示している出来事、frame はその出来事を表示したフレーム数。
	events []tactics.Event
	index  int
	frame  int
	// fast は早送り（Space か Enter を押している間）であること。
	fast bool

	// shownTurn と shownSide は、画面に出しているターンと番。再生中は、表示した PhaseStart に合わせて進める。
	shownTurn int
	shownSide tactics.Side

	message string

	// hints は通常の盤面に出す増援の予兆領域。再生の完了後と巻き戻しの成功後だけ取り直す（DES-UI-24）。
	hints []tactics.ReinforcementHint
}

// NewController は、newBattle で戦闘を作り、初めの出来事の再生から始める（DES-UI-02、DES-UI-04）。
func NewController(newBattle Factory) (*Controller, error) {
	c := &Controller{target: -1, watched: -1, cursorX: -1, cursorY: -1}
	b, events, err := newBattle()
	if err != nil {
		return nil, err
	}
	c.battle = b
	c.play(events)
	return c, nil
}

// State は今の入力の状態を返す。
func (c *Controller) State() State { return c.state }

// Phase は、画面に出しているターンと番を返す（DES-UI-01、DES-UI-03）。再生中は、表示した PhaseStart の値になる。
func (c *Controller) Phase() (turn int, side tactics.Side) { return c.shownTurn, c.shownSide }

// Battle は今の戦闘を返す。
func (c *Controller) Battle() *tactics.Battle { return c.battle }

// Selection は、選んだ自軍、移動先、攻撃の相手（いなければ -1）を返す。
// Selected・Moved・Targeting・HealTargeting・ItemMenu のときだけ意味を持つ。
func (c *Controller) Selection() (unit int, dest tactics.Pos, target int) {
	return c.unit, c.dest, c.target
}

// Message は、直前の操作についての知らせ（確認や誤り）を返す。
func (c *Controller) Message() string { return c.message }

// Click は盤面のマス p を左クリックしたときの処理（DES-UI-02）。
func (c *Controller) Click(p tactics.Pos) {
	if c.rewind.open || c.battle.NeedsResume() {
		return
	}
	switch c.state {
	case Idle:
		c.confirmEnd = false
		c.message = ""
		i, ok := c.unitAt(p)
		switch {
		case ok && c.battle.Units()[i].Side == tactics.Player && !c.battle.Units()[i].Acted:
			c.state, c.unit, c.watched = Selected, i, -1
		case ok && c.battle.Units()[i].Side == tactics.Enemy:
			// 敵の移動と攻撃の範囲を出す・消す。
			if c.watched == i {
				c.watched = -1
			} else {
				c.watched = i
			}
		default:
			c.watched = -1
		}
	case Selected:
		if _, ok := c.battle.Reachable(c.unit)[p]; ok {
			c.state, c.dest = Moved, p
		}
	case Targeting, HealTargeting:
		targets, action := c.Targets(), tactics.Attack
		if c.state == HealTargeting {
			targets, action = c.HealTargets(), tactics.Heal
		}
		for _, t := range targets {
			if c.battle.Units()[t].Pos != p {
				continue
			}
			if c.target == t {
				// 同じ相手をもう一度クリックすると決定する。
				c.act(tactics.Command{Unit: c.unit, To: c.dest, Action: action, Target: t})
			} else {
				c.target = t
			}
			return
		}
	}
}

// ChooseAction は、行動の一覧から a を選んだときの処理（DES-UI-02）。Moved で、一覧にある行動だけを受け付ける。
func (c *Controller) ChooseAction(a tactics.Action) {
	if c.rewind.open || c.battle.NeedsResume() {
		return
	}
	if c.state != Moved || !containsAction(c.Actions(), a) {
		return
	}
	switch a {
	case tactics.Wait:
		c.act(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Wait})
	case tactics.Attack:
		c.state, c.target = Targeting, -1
	case tactics.Heal:
		c.state, c.target = HealTargeting, -1
	case tactics.UseItem:
		c.state = ItemMenu
	}
}

// ChooseItem は、アイテムの一覧から手持ちの k 番目を選んだときの処理（DES-UI-02）。
func (c *Controller) ChooseItem(k int) {
	if c.rewind.open || c.battle.NeedsResume() {
		return
	}
	if c.state != ItemMenu || k < 0 || k >= len(c.battle.Units()[c.unit].Items) {
		return
	}
	c.act(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.UseItem, Item: k})
}

// Cancel は、右クリックか Esc で取り消すときの処理（DES-UI-02）。
func (c *Controller) Cancel() {
	if c.rewind.open {
		if c.rewind.confirm {
			c.rewind.confirm = false
		} else {
			c.rewind = rewindMenu{}
		}
		return
	}
	switch c.state {
	case Idle:
		c.watched, c.confirmEnd, c.message = -1, false, ""
	case Selected:
		c.state = Idle
	case Moved:
		c.state = Selected
	case Targeting, HealTargeting:
		c.state, c.target = Moved, -1
	case ItemMenu:
		c.state = Moved
	}
}

// Escape は戦闘中の選択を取り消す。Over では何もしない（DES-UI-17）。
func (c *Controller) Escape() { c.Cancel() }

// EndPhase は E キーの処理（DES-UI-02、UC-05）。Idle のときだけ受け付ける。
// 行動していない自軍がいれば、1 回目は確認を求め、もう一度押すと番を終える。
func (c *Controller) EndPhase() {
	if c.rewind.open || c.battle.NeedsResume() {
		return
	}
	if c.state != Idle {
		return
	}
	if c.hasUnacted() && !c.confirmEnd {
		c.confirmEnd = true
		c.message = "まだ行動していない自軍がいます。もう一度 E を押すと番を終えます"
		return
	}
	events, err := c.battle.EndPhase()
	if err != nil {
		c.confirmEnd = false
		c.message = fmt.Sprintf("番を終えられません: %v", err)
		return
	}
	c.play(events)
}

// SetFastForward は、Space か Enter を押しているかを毎フレーム伝える（DES-UI-04）。
// 押している間は、出来事 1 つを FastFramesPerEvent フレームで進める。出来事は飛ばさず、すべて順に表示する。
func (c *Controller) SetFastForward(on bool) {
	c.fast = on
}

// Tick は 1 フレームを進める（DES-UI-04、DES-UI-16）。通常は20フレーム、レベルアップは60フレーム表示する。
// 早送りの間は FastFramesPerEvent フレームごとに進める。
func (c *Controller) Tick() {
	if c.state != Playing {
		return
	}
	c.frame++
	limit := FramesPerEvent
	if c.events[c.index].Kind == tactics.EventLevelUp {
		limit = LevelUpFrames
	}
	if c.fast {
		limit = FastFramesPerEvent
	}
	if c.frame < limit {
		return
	}
	c.index++
	c.frame = 0
	if c.index >= len(c.events) {
		c.finish()
		return
	}
	c.showPhase()
}

// showPhase は、表示する出来事が PhaseStart なら、画面に出すターンと番を合わせる（DES-UI-03）。
func (c *Controller) showPhase() {
	if e := c.events[c.index]; e.Kind == tactics.EventPhaseStart {
		c.shownTurn, c.shownSide = e.Turn, e.Side
	}
}

// Event は再生中の出来事と、それを表示したフレーム数を返す（DES-UI-04）。再生中でなければ ok が偽。
func (c *Controller) Event() (e tactics.Event, frame int, ok bool) {
	if c.state != Playing || c.index >= len(c.events) {
		return tactics.Event{}, 0, false
	}
	return c.events[c.index], c.frame, true
}

// Units は盤面に描くユニットを返す（DES-UI-02、DES-UI-04）。
// 再生中は出来事の写し、移動先を選んだあと（Act の前）は、選んだ自軍を移動先に置いた写しを返す。
func (c *Controller) Units() []tactics.Unit {
	if e, _, ok := c.Event(); ok {
		return e.Units
	}
	units := c.battle.Units()
	switch c.state {
	case Moved, Targeting, HealTargeting, ItemMenu:
		units[c.unit].Pos = c.dest
	}
	return units
}

// Actions は、Moved でできる行動を攻撃（杖なら回復）・アイテム・待機の順に返す（DES-UI-02、DES-UI-18）。
// 攻撃は射程に相手がいるとき、アイテムは手持ちがあって HP が最大でないときだけ出す。
func (c *Controller) Actions() []tactics.Action {
	if c.state != Moved {
		return nil
	}
	var out []tactics.Action
	if len(c.Targets()) > 0 {
		out = append(out, tactics.Attack)
	}
	u := c.battle.Units()[c.unit]
	if len(c.HealTargets()) > 0 {
		out = append(out, tactics.Heal)
	}
	if len(u.Items) > 0 && u.HP < u.Stats.HP {
		out = append(out, tactics.UseItem)
	}
	return append(out, tactics.Wait)
}

// Targets は、選んだ自軍が移動先から攻撃できる、生きている敵の添字を返す（DES-UI-02、DES-BTL-04）。
// Moved・Targeting のときだけ返す。
func (c *Controller) Targets() []int {
	if c.state != Moved && c.state != Targeting {
		return nil
	}
	units := c.battle.Units()
	w := units[c.unit].Weapon
	if w.Type == tactics.Staff {
		return nil
	}
	var out []int
	for i, u := range units {
		if !u.Alive || u.Side == units[c.unit].Side {
			continue
		}
		if d := distance(c.dest, u.Pos); d >= w.MinRange && d <= w.MaxRange {
			out = append(out, i)
		}
	}
	return out
}

// Overlay は移動範囲と武器の射程を返す（DES-UI-03、DES-UI-08、DES-UI-18）。
// 第二戻り値の色は RangeStyle に従い、杖なら緑、攻撃なら赤になる。
// 敵の範囲を出しているとき（shownEnemy）は、選んでいる自軍の範囲の代わりに、その敵の範囲を返す。
// そうでなければ、Selected では選んだ自軍の範囲を返す。
// 赤は、移動できるマスから武器が届くマスのうち、移動できないマス。
// Moved・Targeting・HealTargeting では、移動先から武器が届くマスを第二戻り値で返す。
func (c *Controller) Overlay() (blue, red map[tactics.Pos]bool) {
	if e := c.shownEnemy(); e >= 0 {
		return c.ranges(e)
	}
	switch c.state {
	case Selected:
		return c.ranges(c.unit)
	case Moved, Targeting, HealTargeting:
		red = map[tactics.Pos]bool{}
		for _, p := range c.battle.AttackRange(c.unit, c.dest) {
			red[p] = true
		}
		return nil, red
	}
	return nil, nil
}

// SetCursor は、カーソルの画面の座標を毎フレーム伝える（DES-UI-08）。敵に合わせると、その敵の範囲を出す。
func (c *Controller) SetCursor(x, y int) {
	c.cursorX, c.cursorY = x, y
}

// shownEnemy は、範囲を出す敵の添字を返す（DES-UI-08）。出さなければ -1。
// Idle・Selected・Moved のときだけ出す。カーソルを合わせた敵を、クリックして出し続けている敵より優先する。
func (c *Controller) shownEnemy() int {
	switch c.state {
	case Idle, Selected, Moved:
	default:
		return -1
	}
	if i := c.hoveredEnemy(); i >= 0 {
		return i
	}
	// watched は Idle でだけ決まり、Selected に移るときに消える。
	return c.watched
}

// hoveredEnemy は、カーソルのあるマスにいる生きている敵の添字を返す（DES-UI-08）。
// いないとき、カーソルが盤面の外か行動の一覧の上にあるときは -1。
func (c *Controller) hoveredEnemy() int {
	p, ok := CellAt(c.cursorX, c.cursorY)
	if !ok {
		return -1
	}
	if n := len(c.MenuLabels()); n > 0 && HitMenu(MenuRects(c.dest, n), c.cursorX, c.cursorY) >= 0 {
		return -1
	}
	i, ok := c.unitAt(p)
	if !ok || c.battle.Units()[i].Side != tactics.Enemy {
		return -1
	}
	return i
}

// ToggleDanger は Q キーの処理（DES-UI-08）。Playing・Over 以外のとき、敵の危険範囲を出す・消す。
func (c *Controller) ToggleDanger() {
	if c.rewind.open || c.state == Playing || c.state == Over {
		return
	}
	c.danger = !c.danger
}

// Danger は、橙の枠で描く敵の危険範囲を返す（DES-UI-08）。出していないとき、再生中と勝ち負けの画面では nil。
//
// 危険範囲は、生きている敵それぞれについて、Reachable の各マスから AttackRange で攻撃できるマスを合わせたもの。
// 今の盤面（Battle の位置）で求めるので、Moved で画面の上だけ動かした自軍の仮の位置は数えない。
// 自軍のいるマスは、Reachable では通れないが、AttackRange では攻撃できる。待ち構える敵も含める。
// まだ現れていない増援は Battle のユニットにいないので、含まれない。
func (c *Controller) Danger() map[tactics.Pos]bool {
	if !c.danger || c.state == Playing || c.state == Over {
		return nil
	}
	out := map[tactics.Pos]bool{}
	for i, u := range c.battle.Units() {
		if !u.Alive || u.Side != tactics.Enemy {
			continue
		}
		for p := range c.battle.Reachable(i) {
			for _, q := range c.battle.AttackRange(i, p) {
				out[q] = true
			}
		}
	}
	return out
}

// ranges はユニット i の移動できるマスと攻撃できるマスを返す（DES-UI-03）。
func (c *Controller) ranges(i int) (blue, red map[tactics.Pos]bool) {
	blue, red = map[tactics.Pos]bool{}, map[tactics.Pos]bool{}
	reach := c.battle.Reachable(i)
	for p := range reach {
		blue[p] = true
	}
	for p := range reach {
		for _, q := range c.battle.AttackRange(i, p) {
			if !blue[q] {
				red[q] = true
			}
		}
	}
	return blue, red
}

// act は決定したコマンドで Act を 1 回だけ呼び、出来事の再生を始める（DES-UI-02、DES-BTL-04）。
func (c *Controller) act(cmd tactics.Command) {
	events, err := c.battle.Act(cmd)
	if err != nil {
		// 画面ではできる行動だけを出すので、ふつうは来ない。状態は変わっていない（DES-ERR-01）。
		c.state = Idle
		c.message = fmt.Sprintf("行動できません: %v", err)
		return
	}
	c.play(events)
}

// play は出来事の再生を始める（DES-UI-04）。
func (c *Controller) play(events []tactics.Event) {
	c.record = append(c.record, events...)
	c.state, c.events, c.index, c.frame = Playing, events, 0, 0
	c.target, c.watched, c.confirmEnd, c.message = -1, -1, false, ""
	if len(events) == 0 {
		c.finish()
		return
	}
	c.showPhase()
}

// finish は再生を終え、勝ち負けが決まっていれば Over、そうでなければ Idle にする（DES-UI-02）。
func (c *Controller) finish() {
	// 画面に出すターンと番は、表示した PhaseStart で合わせてある（showPhase）。
	c.events, c.index, c.frame = nil, 0, 0
	c.refreshHints()
	if c.battle.Result != tactics.Ongoing {
		c.state = Over
		return
	}
	c.state = Idle
}

// unitAt は、マス p にいる生きているユニットの添字を返す。
func (c *Controller) unitAt(p tactics.Pos) (int, bool) {
	for i, u := range c.battle.Units() {
		if u.Alive && u.Pos == p {
			return i, true
		}
	}
	return -1, false
}

// hasUnacted は、まだ行動していない生きている自軍がいるかを返す。
func (c *Controller) hasUnacted() bool {
	for _, u := range c.battle.Units() {
		if u.Alive && u.Side == tactics.Player && !u.Acted {
			return true
		}
	}
	return false
}

func containsAction(list []tactics.Action, a tactics.Action) bool {
	for _, x := range list {
		if x == a {
			return true
		}
	}
	return false
}

// distance はマンハッタン距離を返す。
func distance(a, b tactics.Pos) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// HealTargets は移動先から回復できる味方を core の検証で選ぶ（DES-UI-18）。
func (c *Controller) HealTargets() []int {
	if c.state != Moved && c.state != HealTargeting {
		return nil
	}
	var out []int
	for i := range c.battle.Units() {
		if _, err := c.battle.HealForecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Heal, Target: i}); err == nil {
			out = append(out, i)
		}
	}
	return out
}

// RangeStyle は表示中の射程の色を返す。敵の射程は赤のままにする（DES-UI-18、DES-UI-08）。
func (c *Controller) RangeStyle() InfoStyle {
	if c.shownEnemy() < 0 && (c.state == Selected || c.state == Moved || c.state == HealTargeting) && c.battle.Units()[c.unit].Weapon.Type == tactics.Staff {
		return StyleGood
	}
	return StyleBad
}

// TargetStyle は回復候補の枠を緑、攻撃候補の枠を赤にする（DES-UI-18）。
func (c *Controller) TargetStyle() InfoStyle {
	if c.state == HealTargeting {
		return StyleGood
	}
	return StyleBad
}

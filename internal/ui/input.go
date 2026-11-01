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

// Canto は、行動の結果を再生したあと、待っている再移動の行き先を選んでいる状態（DES-UI-26、DES-IT06 12.3 節）。
// ほかのユニットは操作できず、番も終えられない。
const Canto State = HealTargeting + 1

// WeaponMenu は攻撃に使う武器を選ぶ状態、StaffMenu は回復に使う杖を選ぶ状態（DES-UI-26、DES-IT06 12.1 節）。
// 決定するまで core の装備は変えない。
const (
	WeaponMenu State = Canto + 1
	StaffMenu  State = Canto + 2
)

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
	camera Camera
	rewind rewindMenu
	record []tactics.Event
	battle *tactics.Battle
	state  State

	// unit は選んだ自軍、dest は移動先、target は攻撃・回復の相手（いなければ -1）。
	unit   int
	dest   tactics.Pos
	target int
	// weapon は攻撃・回復に選んだ武器の列の添字（選んでいなければ -1）。
	// weaponMenu は武器・杖の一覧を出したこと（取り消しで一覧へ戻るか、行動の一覧へ戻るかを分ける。DES-IT06 12.1 節）。
	weapon     int
	weaponMenu bool
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

	// infoPage は情報欄のユニットのページ（DES-UI-28）。表示だけに使う。
	infoPage InfoPage
	// lastCell は、直前にカーソルを置いた盤面のマス（DES-UI-28）。カーソルが情報欄の上にあるときの表示に使う。
	lastCell   tactics.Pos
	lastCellOK bool
}

// NewController は、newBattle で戦闘を作り、初めの出来事の再生から始める（DES-UI-02、DES-UI-04）。
func NewController(newBattle Factory) (*Controller, error) {
	c := &Controller{target: -1, weapon: -1, watched: -1, cursorX: -1, cursorY: -1}
	b, events, err := newBattle()
	if err != nil {
		return nil, err
	}
	c.battle = b
	c.camera = Camera{Width: b.Width(), Height: b.Height()}
	c.camera.focusHero(b.Units())
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
			c.camera.Focus(p)
		}
	case Canto:
		// 今の位置は「その場で終える」、ほかの行けるマスは再移動にする（DES-IT06 12.3 節）。
		if p == c.dest {
			c.EndCanto()
		} else if _, ok := c.battle.CantoReachable()[p]; ok {
			c.act(tactics.Command{Unit: c.unit, To: p, Action: tactics.CantoMove})
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
				// 同じ相手をもう一度クリックすると決定する。選んだ武器・杖をコマンドで渡す（DES-ITM-02）。
				w := c.weapon
				c.act(tactics.Command{Unit: c.unit, To: c.dest, Action: action, Target: t, Weapon: &w})
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
		c.chooseWeapons(c.attackWeapons(), WeaponMenu, Targeting)
	case tactics.Heal:
		c.chooseWeapons(c.staffWeapons(), StaffMenu, HealTargeting)
	case tactics.UseItem:
		c.state = ItemMenu
	}
	if c.state == Targeting || c.state == HealTargeting {
		targets := c.Targets()
		if c.state == HealTargeting {
			targets = c.HealTargets()
		}
		if len(targets) > 0 {
			c.camera.Focus(c.battle.Units()[targets[0]].Pos)
		}
	}
}

// chooseWeapons は、攻撃の武器か杖の候補 ws から、相手を選ぶ状態 next か一覧の状態 menu に進む（DES-UI-26、DES-IT06 12.1 節）。
// 候補が 1 本だけなら一覧を飛ばし、その 1 本で相手を選ぶ（取り消すと行動の一覧へ戻る）。
func (c *Controller) chooseWeapons(ws []int, menu, next State) {
	c.target = -1
	if len(ws) == 1 {
		c.state, c.weapon, c.weaponMenu = next, ws[0], false
		return
	}
	c.state, c.weapon, c.weaponMenu = menu, -1, true
}

// ChooseWeapon は、武器・杖の一覧の k 行目を選んだときの処理（DES-UI-26）。WeaponMenu・StaffMenu のときだけ受け付ける。
// 選んだ武器で届く相手がいなければ、理由を知らせて一覧に残る。core の装備は変えない。
func (c *Controller) ChooseWeapon(k int) {
	if c.rewind.open || c.battle.NeedsResume() {
		return
	}
	ws, next := c.attackWeapons(), Targeting
	if c.state == StaffMenu {
		ws, next = c.staffWeapons(), HealTargeting
	} else if c.state != WeaponMenu {
		return
	}
	if k < 0 || k >= len(ws) {
		return
	}
	if reason := c.weaponReason(ws[k]); reason != "" {
		c.message = reason
		return
	}
	c.state, c.weapon, c.target, c.message = next, ws[k], -1, ""
	targets := c.Targets()
	if next == HealTargeting {
		targets = c.HealTargets()
	}
	if len(targets) > 0 {
		c.camera.Focus(c.battle.Units()[targets[0]].Pos)
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
		// 予測 → 相手 → 武器 → 行動の一覧 の順に戻る（DES-IT06 12.1 節、UC-02 5c）。
		// 相手を選んで予測を出しているときは、相手の選択を外すだけにする。一覧を飛ばしたときは行動の一覧へ戻る。
		if c.target >= 0 {
			c.target = -1
			return
		}
		switch {
		case c.weaponMenu && c.state == Targeting:
			c.state = WeaponMenu
		case c.weaponMenu:
			c.state = StaffMenu
		default:
			c.state = Moved
		}
		c.target, c.weapon = -1, -1
		c.camera.Focus(c.dest)
	case WeaponMenu, StaffMenu:
		c.state, c.weapon, c.weaponMenu, c.message = Moved, -1, false, ""
	case ItemMenu:
		c.state = Moved
	case Canto:
		// 結果が出たあとの取り消しは、攻撃を取り消さず、今の位置で再移動を終える（DES-IT06 8.2・12.3 節）。
		c.EndCanto()
	}
}

// EndCanto は、待っている再移動を今の位置で終える（CantoEnd。DES-IT06 12.3 節）。Canto のときだけ受け付ける。
func (c *Controller) EndCanto() {
	if c.rewind.open || c.battle.NeedsResume() || c.state != Canto {
		return
	}
	c.act(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.CantoEnd})
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
	c.focusEvent(c.events[c.index])
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
	case Moved, Targeting, HealTargeting, ItemMenu, WeaponMenu, StaffMenu:
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

// Targets は、選んだ自軍が移動先から攻撃できる、生きている敵の添字を返す（DES-UI-02、DES-BTL-04、DES-ITM-02）。
// Targeting では選んだ武器で届く敵、Moved では手持ちのどれかの攻撃の武器で届く敵を返す。ほかの状態では nil。
// 届くかは core の予測の検証で決め、画面で射程を計算し直さない。
func (c *Controller) Targets() []int {
	switch c.state {
	case Targeting:
		return c.targetsWith(c.weapon)
	case Moved:
		return unionTargets(c.attackWeapons(), c.targetsWith)
	}
	return nil
}

// targetsWith は、武器の列の w 番目で移動先から攻撃できる敵の添字を、添字の順に返す（DES-ITM-02）。
func (c *Controller) targetsWith(w int) []int {
	var out []int
	for i := range c.battle.Units() {
		if _, err := c.battle.Forecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Attack, Target: i, Weapon: &w}); err == nil {
			out = append(out, i)
		}
	}
	return out
}

// unionTargets は、武器の候補 ws のどれかで届く相手を、添字の順に重ねずに返す。
func unionTargets(ws []int, with func(int) []int) []int {
	seen := map[int]bool{}
	for _, w := range ws {
		for _, t := range with(w) {
			seen[t] = true
		}
	}
	var out []int
	for i := 0; len(seen) > 0; i++ {
		if seen[i] {
			out = append(out, i)
			delete(seen, i)
		}
	}
	return out
}

// attackWeapons は選んだ自軍の攻撃の武器（杖を除く）の添字を、武器の列の順に返す（DES-ITM-02）。
func (c *Controller) attackWeapons() []int {
	var out []int
	for k, w := range c.battle.Units()[c.unit].Weapons {
		if w.Type != tactics.Staff {
			out = append(out, k)
		}
	}
	return out
}

// staffWeapons は選んだ自軍の杖の添字を、武器の列の順に返す（DES-ITM-02）。
func (c *Controller) staffWeapons() []int {
	var out []int
	for k, w := range c.battle.Units()[c.unit].Weapons {
		if w.Type == tactics.Staff {
			out = append(out, k)
		}
	}
	return out
}

// weaponReason は、武器の列の w 番目を選べない理由を返す（DES-IT06 12.1 節）。選べるなら空。
func (c *Controller) weaponReason(w int) string {
	weapon := c.battle.Units()[c.unit].Weapons[w]
	if weapon.Type == tactics.Staff {
		if weapon.Uses < 1 {
			return fmt.Sprintf("%s: %s", weapon.Name, tactics.ErrNoUses.Error())
		}
		if len(c.healTargetsWith(w)) == 0 {
			return fmt.Sprintf("%s で回復できる味方がいません", weapon.Name)
		}
		return ""
	}
	if len(c.targetsWith(w)) == 0 {
		return fmt.Sprintf("%s の射程（%s）に届く敵がいません", weapon.Name, rangeText(weapon))
	}
	return ""
}

// menuWeapons は、今の一覧に並べる武器の添字を返す（WeaponMenu は攻撃の武器、StaffMenu は杖）。
func (c *Controller) menuWeapons() []int {
	switch c.state {
	case WeaponMenu:
		return c.attackWeapons()
	case StaffMenu:
		return c.staffWeapons()
	}
	return nil
}

// hoveredWeapon は、一覧の上でカーソルが指している武器の添字を返す。なければ -1。
func (c *Controller) hoveredWeapon() int {
	ws := c.menuWeapons()
	if len(ws) == 0 {
		return -1
	}
	if i := HitMenu(c.Camera().MenuRects(c.dest, len(ws)), c.cursorX, c.cursorY); i >= 0 {
		return ws[i]
	}
	return -1
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
	case Canto:
		// 再移動で行けるマスだけを青で出す（DES-IT06 12.3 節）。
		blue = map[tactics.Pos]bool{}
		for p := range c.battle.CantoReachable() {
			blue[p] = true
		}
		return blue, nil
	case Moved, Targeting, HealTargeting, WeaponMenu, StaffMenu:
		// 相手を選んでいる間と一覧の上の行は、選んだ武器 1 本の射程、ほかは攻撃の武器を合わせた射程（DES-ITM-02）。
		cells := c.battle.AttackRange(c.unit, c.dest)
		w := c.weapon
		if c.state == WeaponMenu || c.state == StaffMenu {
			w = c.hoveredWeapon()
			if staffs := c.staffWeapons(); w < 0 && c.state == StaffMenu && len(staffs) > 0 {
				w = staffs[0]
			}
		}
		if w >= 0 {
			cells = c.battle.WeaponRange(c.unit, c.dest, w)
		}
		red = map[tactics.Pos]bool{}
		for _, p := range cells {
			red[p] = true
		}
		return nil, red
	}
	return nil, nil
}

// SetCursor は、カーソルの画面の座標を毎フレーム伝える（DES-UI-08）。敵に合わせると、その敵の範囲を出す。
// 盤面の上なら、そのマスを情報欄のために覚えておく（DES-UI-28）。
func (c *Controller) SetCursor(x, y int) {
	c.cursorX, c.cursorY = x, y
	if p, ok := c.Camera().CellAt(x, y); ok {
		c.lastCell, c.lastCellOK = p, true
	}
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
	p, ok := c.Camera().CellAt(c.cursorX, c.cursorY)
	if !ok {
		return -1
	}
	if n := len(c.MenuLabels()); n > 0 && HitMenu(c.Camera().MenuRects(c.dest, n), c.cursorX, c.cursorY) >= 0 {
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
	// 行動の再生のあとに待っている再移動があれば、再移動の状態に入る（DES-IT06 12.3 節）。
	if pc, ok := c.battle.PendingCanto(); ok {
		c.state, c.unit, c.dest = Canto, pc.Unit, c.battle.Units()[pc.Unit].Pos
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

// HealTargets は移動先から回復できる味方を core の検証で選ぶ（DES-UI-18、DES-ITM-02）。
// HealTargeting では選んだ杖、Moved では手持ちのどれかの杖で回復できる味方を返す。
func (c *Controller) HealTargets() []int {
	switch c.state {
	case HealTargeting:
		return c.healTargetsWith(c.weapon)
	case Moved:
		return unionTargets(c.staffWeapons(), c.healTargetsWith)
	}
	return nil
}

// healTargetsWith は、武器の列の w 番目の杖で移動先から回復できる味方の添字を返す（DES-ITM-02）。
func (c *Controller) healTargetsWith(w int) []int {
	var out []int
	for i := range c.battle.Units() {
		if _, err := c.battle.HealForecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Heal, Target: i, Weapon: &w}); err == nil {
			out = append(out, i)
		}
	}
	return out
}

// RangeStyle は表示中の射程の色を返す。敵の射程は赤のままにする（DES-UI-18、DES-UI-08）。
// 攻撃の武器の赤と、選んだ杖の緑を分ける（DES-IT06 6.1 節）。移動先を選ぶ前と行動の一覧では、攻撃の武器がなければ緑。
func (c *Controller) RangeStyle() InfoStyle {
	if c.shownEnemy() >= 0 {
		return StyleBad
	}
	switch c.state {
	case HealTargeting, StaffMenu:
		return StyleGood
	case Selected, Moved:
		if len(c.attackWeapons()) == 0 {
			return StyleGood
		}
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

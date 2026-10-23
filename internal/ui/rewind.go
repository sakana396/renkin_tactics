package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// rewindMenu は入力を保持して重ねる一覧と確認（DES-UI-19）。
// 選択中の 1 件の行動前の盤面だけを写しとして持つ（DES-UI-21）。
type rewindMenu struct {
	camera        Camera
	reasonOffset  int
	detailOffset  int
	open, confirm bool
	selected      int
	message       string

	// previewID は preview を取得した履歴 ID。loaded が偽なら未取得。
	previewID    uint64
	preview      tactics.BoardView
	previewUnits []tactics.Unit
	// previewHints は preview の予兆領域。取得したときに 1 度だけ写す（DES-UI-24）。
	previewHints  []tactics.ReinforcementHint
	previewErr    error
	previewLoaded bool
	// previewLoads は PreviewRewind を呼んだ回数。描画ごとに取得し直さないことの確認に使う（DES-UI-21）。
	previewLoads int
}

// OpenRewind は再生完了後だけ一覧を開き、最新行の行動前を表示する。閲覧では戦闘と選択を変えない（DES-UI-19、DES-UI-21）。
func (c *Controller) OpenRewind() {
	if c.state == Playing || c.rewind.open {
		return
	}
	c.rewind = rewindMenu{camera: c.camera, open: true, selected: len(c.battle.History()) - 1}
	c.loadPreview()
}

// loadPreview は選択した行が変わったときだけ、その 1 件の行動前を取得し直す（DES-UI-21、DES-RWD-05）。
// 取得に失敗したら、前の行の盤面を残さない。
func (c *Controller) loadPreview() {
	h := c.battle.History()
	i := c.rewind.selected
	if i < 0 || i >= len(h) {
		return
	}
	if c.rewind.previewLoaded && c.rewind.previewID == h[i].ID {
		return
	}
	c.fetchPreview(h[i].ID)
}

// fetchPreview は ID の行動前を取得して写しを入れ替える（DES-UI-21）。
func (c *Controller) fetchPreview(id uint64) {
	c.rewind.previewLoads++
	v, err := c.battle.PreviewRewind(id)
	c.rewind.previewID, c.rewind.previewLoaded, c.rewind.previewErr = id, true, err
	c.rewind.preview, c.rewind.previewUnits, c.rewind.previewHints = tactics.BoardView{}, nil, nil
	c.rewind.camera = Camera{}
	if err == nil {
		c.rewind.preview, c.rewind.previewUnits, c.rewind.previewHints = v, v.Units(), v.ReinforcementHints()
		c.rewind.camera = Camera{Width: v.Width(), Height: v.Height()}
		c.focusHistory(&c.rewind.camera, v.Units())
	}
}

// previewOK は選択中の行の行動前を表示できているかを返す（DES-UI-21）。
func (c *Controller) previewOK() bool {
	h := c.battle.History()
	i := c.rewind.selected
	return i >= 0 && i < len(h) && c.rewind.previewLoaded && c.rewind.previewErr == nil && c.rewind.previewID == h[i].ID
}

// rewindSelect は一覧の i 行目を選び、プレビューを切り替える（DES-UI-19、DES-UI-21）。
func (c *Controller) rewindSelect(i int) {
	c.rewind.selected = i
	c.rewind.detailOffset = 0
	c.rewind.reasonOffset = 0
	c.loadPreview()
}

// RewindMove は上下とページ移動を先頭・末尾で止め、選んだ行の盤面に切り替える（DES-UI-19、DES-UI-21）。
func (c *Controller) RewindMove(delta int) {
	if !c.rewind.open || c.rewind.confirm {
		return
	}
	n := len(c.battle.History())
	if n == 0 {
		return
	}
	c.rewindSelect(max(0, min(n-1, c.rewind.selected+delta)))
}

// RewindEnter は確認を経て一度だけ巻き戻す。消える項目の境界を先に保持する（DES-UI-19）。
// 残り 0 回とプレビューを取得できない選択では確定しない。失敗したら一覧に戻り、消費しない（DES-UI-21）。
func (c *Controller) RewindEnter() {
	if !c.rewind.open || c.battle.RewindsLeft() == 0 || !c.previewOK() || !c.rewindAllowed() {
		return
	}
	h := c.battle.History()
	i := c.rewind.selected
	if !c.rewind.confirm {
		c.rewind.confirm = true
		c.rewind.detailOffset = 0
		return
	}
	entry := h[i]
	if err := c.battle.Rewind(entry.ID); err != nil {
		c.rewind.confirm = false
		c.rewind.detailOffset = 0
		c.rewind.message = err.Error()
		c.fetchPreview(entry.ID)
		return
	}
	clear(c.record[min(len(c.record), entry.EventStart):])
	c.record = c.record[:min(len(c.record), entry.EventStart)]
	c.camera = Camera{Width: c.battle.Width(), Height: c.battle.Height()}
	focusHistoryEntry(&c.camera, entry, c.battle.Units())
	c.rewind = rewindMenu{}
	c.events, c.index, c.frame = nil, 0, 0
	c.unit, c.target, c.watched = -1, -1, -1
	c.dest = tactics.Pos{}
	c.confirmEnd, c.fast, c.message = false, false, ""
	c.shownTurn, c.shownSide = c.battle.Turn, c.battle.Phase
	c.state = Idle
	c.refreshHints()
}

// rewindLabel は描画と独立した行動の全文。移動だけの敵も出来事の境界から区別する（DES-UI-19）。
func (c *Controller) rewindLabel(e tactics.HistoryEntry) string {
	action := actionName(e.Action)
	switch e.Kind {
	case tactics.HistoryEndPhase:
		action = "番を終える"
		if e.Automatic {
			action = "番の移行（自動）"
		}
	case tactics.HistoryPlayerStart:
		action = "自軍の番を始める"
	default:
		for j := e.EventStart; j < e.EventEnd && j < len(c.record); j++ {
			if c.record[j].Kind == tactics.EventMove && c.record[j].Unit == e.Unit && c.record[j].To == e.To {
				action = "移動・" + action
				break
			}
		}
	}
	target := ""
	if e.TargetName != "" {
		target = " → " + e.TargetName
	}
	return fmt.Sprintf("%dT %s %s %s%s", e.Turn, sideName(e.Side), e.UnitName, action, target)
}

// fitRewind は行を欄幅で省略し、詳細は別に全文を折り返す（DES-UI-19）。
func fitRewind(s string, width int) string {
	if textWidth(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r)+"…") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// 右欄の一覧・確認の配置（DES-UI-21）。盤面の右 x=512〜832 だけを使い、左右に 8 px の余白を置く。
const (
	rewindLeft   = MapSize + 8
	rewindWidth  = InfoWidth - 16
	rewindRight  = rewindLeft + rewindWidth
	rewindRowTop = 60
	rewindRowH   = 28
	// rewindDetailLines は一覧の詳細に出す行の数、rewindConfirmLines は確認に出す行の数。
	rewindDetailLines  = 5
	rewindConfirmLines = 18
)

// wrapWidth は文を width px で文字の単位で折り返す（DES-UI-21）。
func wrapWidth(s string, width int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		row := ""
		for _, r := range line {
			next := row + string(r)
			if textWidth(next) > width && row != "" {
				out = append(out, row)
				row = string(r)
			} else {
				row = next
			}
		}
		out = append(out, row)
	}
	return out
}

// rewindDetail は、詳細・確認で送って読む全文の行。行動名の全文に、誤りの全文を続ける（DES-UI-21）。
func (c *Controller) rewindDetail() []SceneText {
	var out []SceneText
	h := c.battle.History()
	if i := c.rewind.selected; i >= 0 && i < len(h) {
		text := c.rewindLabel(h[i])
		if c.rewind.confirm {
			text = "この行動から先を取り消して戻りますか。\n" + text
		}
		for _, s := range wrapWidth(text, rewindWidth) {
			out = append(out, SceneText{Text: s})
		}
	}
	if c.rewind.message != "" {
		for _, s := range wrapWidth(c.rewind.message, rewindWidth) {
			out = append(out, SceneText{Text: s, Style: StyleBad})
		}
	}
	return out
}

// RewindView は右欄の 8 行の一覧と確認、回数、閉じているときの入口の配置（DES-UI-19、DES-UI-21）。
func (c *Controller) RewindView() SceneView {
	var v SceneView
	add := func(s string, x, y int, style InfoStyle) {
		v.Texts = append(v.Texts, SceneText{Text: s, X: x, Y: y, Style: style})
	}
	button := func(s string, r image.Rectangle, on bool) { v.Buttons = append(v.Buttons, SceneButton{s, r, on}) }
	if !c.rewind.open {
		if c.state == Playing {
			return v
		}
		button(fmt.Sprintf("巻き戻し 残り %d/10", c.battle.RewindsLeft()), image.Rect(304, 512, 504, 530), true)
		add("T: 巻き戻し", 216, 514, StylePlain)

		return v
	}
	h := c.battle.History()
	n := c.battle.RewindsLeft()
	ok := c.previewOK()
	add(fmt.Sprintf("巻き戻し 残り %d/10", n), rewindLeft, 8, StyleTitle)
	left := image.Rect(rewindLeft, 520, rewindLeft+144, 560)
	right := image.Rect(rewindRight-144, 520, rewindRight, 560)
	detail := c.rewindDetail()
	// 選択中の写しに予兆があれば、右欄の理由の下の空きに同じ案内を出す（DES-UI-24）。
	if hv := c.RewindBoard().Hint; hv.Visible {
		v.Texts = append(v.Texts, hv.Text)
	}
	if c.rewind.confirm {
		add("確認（左の盤面に戻ります）", rewindLeft, 34, StylePlain)
		for i, t := range detail[min(len(detail), c.rewind.detailOffset):] {
			if i >= rewindConfirmLines {
				break
			}
			add(t.Text, rewindLeft, rewindRowTop+i*20, t.Style)
		}
		add(fmt.Sprintf("残り %d → %d", n, n-1), rewindLeft, 436, StylePlain)
		add("ホイール: 全文を読む", rewindLeft, 456, StyleLabel)
		button("やめる", left, true)
		button("戻す", right, n > 0 && ok && c.rewindAllowed())
		return v
	}
	add("選んだ行動の「前」に戻る", rewindLeft, 34, StylePlain)
	start := max(0, c.rewind.selected/8*8)
	for i := start; i < min(len(h), start+8); i++ {
		r := image.Rect(rewindLeft, rewindRowTop+(i-start)*rewindRowH, rewindRight, rewindRowTop+(i-start+1)*rewindRowH)
		v.Rows = append(v.Rows, PrepareRow{Rect: r, Selected: i == c.rewind.selected})
		add(fitRewind(c.rewindLabel(h[i]), rewindWidth-16), rewindLeft+8, r.Min.Y+6, rewindStyle(h[i]))
	}
	if len(h) == 0 {
		add("戻せる行動がありません", rewindLeft+8, rewindRowTop+6, StylePlain)
	}
	button("前", image.Rect(rewindLeft, 292, rewindLeft+80, 320), start > 0)
	button("次", image.Rect(rewindLeft+88, 292, rewindLeft+168, 320), start+8 < len(h))
	for i, t := range detail[min(len(detail), c.rewind.detailOffset):] {
		if i >= rewindDetailLines {
			break
		}
		add(t.Text, rewindLeft, 328+i*20, t.Style)
	}
	add("右欄ホイール: 詳細／盤面: 移動", rewindLeft, 436, StyleLabel)
	for i, line := range c.rewindReasons()[min(c.rewind.reasonOffset, len(c.rewindReasons())):] {
		if i >= 2 {
			break
		}
		add(line, rewindLeft, 456+i*20, StyleBad)
	}
	button("戻る", left, true)
	button("決定", right, n > 0 && ok && c.rewindAllowed())
	return v
}

// RewindBoardView は一覧・確認中に左の盤面と下の案内へ描く値（DES-UI-21）。
type RewindBoardView struct {
	// Visible は一覧・確認を開いていること。偽なら通常の盤面を描く。
	Visible bool
	// Available は盤面を描けること。取得に失敗したら偽で、古い盤面を描かない。
	Available bool
	Board     tactics.BoardView
	Units     []tactics.Unit
	// Frame は選んだ行動者のマス。HasFrame が偽なら描かない。
	Frame    tactics.Pos
	HasFrame bool
	// Notice は盤面に重ねる知らせ、Lines は下の案内の行。
	Notice string
	Lines  []string
	// Hint は表示中の盤面（選択中の写し）の増援の気配（DES-UI-24）。
	Hint HintView
}

// RewindBoard は選択中の行動前の盤面と案内を返す。取得し直さず、保持した 1 件の写しを使う（DES-UI-21）。
// 履歴がなければ元の盤面を静止して見せる。
func (c *Controller) RewindBoard() RewindBoardView {
	if !c.rewind.open {
		return RewindBoardView{}
	}
	back := "Esc・右クリック: 元の盤面へ"
	if c.rewind.confirm {
		back = "Esc・右クリック: 一覧へ"
	}
	h := c.battle.History()
	if len(h) == 0 {
		b := c.battle.BoardView()
		return RewindBoardView{Visible: true, Available: true, Board: b, Units: c.Units(),
			Lines: []string{"戻せる行動がありません（今の盤面）", back}, Hint: c.rewindHintView(b.ReinforcementHints())}
	}
	if !c.previewOK() {
		return RewindBoardView{Visible: true, Notice: "プレビューを表示できません",
			Lines: []string{"プレビュー中・未確定　盤面を表示できません", back}}
	}
	p := c.rewind.preview
	v := RewindBoardView{Visible: true, Available: true, Board: p, Units: c.rewind.previewUnits, Hint: c.rewindHintView(c.rewind.previewHints)}
	v.Lines = []string{"プレビュー中・未確定", fmt.Sprintf("盤面: %dT %s　%s", p.Turn, sideName(p.Phase), back), "盤面ホイール: 上下／Shift 併用: 左右"}
	if e := h[c.rewind.selected]; e.Kind == tactics.HistoryAction && e.Unit >= 0 && e.Unit < len(v.Units) && v.Units[e.Unit].Alive {
		v.Frame, v.HasFrame = v.Units[e.Unit].Pos, true
	}
	return v
}

// RewindClick は表示と同じ矩形を使い、一覧を開いている間は盤面への入力を通さない（DES-UI-19、DES-UI-21）。
// 行のクリックは選択とプレビューの切替だけで、確定しない。
func (c *Controller) RewindClick(x, y int) bool {
	v := c.RewindView()
	p := image.Pt(x, y)
	for _, b := range v.Buttons {
		if p.In(b.Rect) {
			if b.Enabled {
				switch {
				case strings.HasPrefix(b.Label, "巻き戻し"):
					c.OpenRewind()
				case b.Label == "前":
					c.RewindMove(-8)
				case b.Label == "次":
					c.RewindMove(8)
				case b.Label == "戻る" || b.Label == "やめる":
					c.Cancel()
				case b.Label == "戻す" || b.Label == "決定":
					c.RewindEnter()
				}
			}
			return true
		}
	}
	if c.rewind.open {
		for i, r := range v.Rows {
			if p.In(r.Rect) {
				c.rewindSelect(c.rewind.selected/8*8 + i)
			}
		}
		return true
	}
	return false
}

// syncRewind は敗北の選択から成功したときだけ戦闘に復帰する（DES-UI-19、6 章）。
func (s *SceneController) syncRewind() {
	if s.scene == GameOver && s.battle.Battle().Result == tactics.Ongoing {
		s.scene = Battle
		s.message = ""
	}
}

// RewindDetail は長い行動名と誤りの全文を、一覧では 5 行、確認では 18 行ずつ読めるようにする（DES-UI-19、DES-UI-21）。
func (c *Controller) RewindDetail(delta int) {
	if !c.rewind.open {
		return
	}
	limit := rewindDetailLines
	if c.rewind.confirm {
		limit = rewindConfirmLines
	}
	if !c.rewind.confirm {
		c.rewind.reasonOffset = max(0, min(max(0, len(c.rewindReasons())-2), c.rewind.reasonOffset+delta))
	}
	n := len(c.rewindDetail())
	c.rewind.detailOffset = max(0, min(max(0, n-limit), c.rewind.detailOffset+delta))
}

// rewindKey はキーの押し始めだけを受け、一覧と確認の入力を読み替える（DES-UI-19）。
func (c *Controller) rewindKey(key string) {
	switch key {
	case "T":
		c.OpenRewind()
	case "Enter":
		if c.rewind.open {
			c.RewindEnter()
		}
	case "Up":
		c.RewindMove(-1)
	case "Down":
		c.RewindMove(1)
	case "PageUp":
		c.RewindMove(-8)
	case "PageDown":
		c.RewindMove(8)
	}
}

// rewindAllowed は表示文や写しの番に依存せず、履歴の属性で決定可否を調べる（DES-UI-21）。
func (c *Controller) rewindAllowed() bool {
	h := c.battle.History()
	i := c.rewind.selected
	if i < 0 || i >= len(h) {
		return false
	}
	e := h[i]
	return e.Side == tactics.Player && (e.Kind == tactics.HistoryAction || e.Kind == tactics.HistoryEndPhase && !e.Automatic)
}

// rewindStyle は選択の有無にかかわらず、種類ごとの色と濃さを保つ（DES-UI-21）。
func rewindStyle(e tactics.HistoryEntry) InfoStyle {
	if e.Kind == tactics.HistoryPlayerStart {
		return StyleRewindBoundary
	}
	if e.Side == tactics.Enemy || e.Kind == tactics.HistoryEndPhase && e.Automatic {
		return StyleRewindEnemy
	}
	return StylePlain
}

// rewindReasons は併発した理由をすべて折り返し、専用の2行内で送って読めるようにする（DES-UI-21）。
func (c *Controller) rewindReasons() []string {
	var reasons []string
	if len(c.battle.History()) == 0 {
		reasons = append(reasons, "戻せる行動がありません")
	} else {
		if !c.rewindAllowed() {
			reasons = append(reasons, "自軍の行動か『番を終える』の前だけ戻せます")
		}
		if !c.previewOK() {
			reasons = append(reasons, "プレビューを表示できないため戻せません")
		}
	}
	if c.battle.RewindsLeft() == 0 {
		reasons = append(reasons, "残り回数が 0 のため巻き戻せません")
	}
	var lines []string
	for _, r := range reasons {
		lines = append(lines, wrapWidth(r, rewindWidth)...)
	}
	return lines
}

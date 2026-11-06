package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 敵の範囲の出所と線の種類（DES-UI-29、DES-IT07 3.2 節）。
// 描画（enemyrange_draw.go）から分け、Ebitengine なしで確かめられるようにする。

// EnemyRangeOwner は敵の範囲の出所（DES-UI-29）。全敵・固定・カーソルを色と印で分ける。
type EnemyRangeOwner int

const (
	// RangeAll は Q で出す全敵の危険範囲。赤紫の薄い塗り。
	RangeAll EnemyRangeOwner = iota
	// RangeWatched はクリックして固定した敵の範囲。橙の線。
	RangeWatched
	// RangeCursor はカーソルを合わせた敵の範囲。青白の線。
	RangeCursor
)

// RangeStroke は範囲の描き方（DES-UI-29）。線の種類で、色が見分けにくくても移動と攻撃を分ける。
type RangeStroke int

const (
	// StrokeFill はマスの塗り（全敵の危険範囲）。
	StrokeFill RangeStroke = iota
	// StrokeSolid は移動できるマスの実線。
	StrokeSolid
	// StrokeDashed は攻撃できるマス（移動できないマス）の破線。
	StrokeDashed
)

// EnemyRangeLayer は盤面に重ねる敵の範囲の 1 層（DES-UI-29）。
type EnemyRangeLayer struct {
	Owner  EnemyRangeOwner
	Stroke RangeStroke
	Cells  map[tactics.Pos]bool
}

// 敵の範囲の色（DES-UI-29、Q-DES07-02）。全敵は赤紫の塗り、固定は橙、カーソルは青白、固定の印は金。
var (
	// dangerFill は全敵の危険範囲の塗り。下の地形とユニットが読めるよう薄くする。
	dangerFill = color.NRGBA{200, 40, 170, 110}
	// watchedLine は固定した敵の範囲の線（橙。旧来の危険範囲の枠と同じ色）。
	watchedLine = dangerColor
	// cursorLine はカーソルの敵の範囲の線と、情報欄の名前の横の印。
	cursorLine = color.NRGBA{200, 235, 255, 255}
	// minimapDangerFill はミニマップの危険範囲のマスの色。1 マス 4 px でも見えるよう、盤面より濃くする。
	minimapDangerFill = color.NRGBA{200, 40, 170, 140}
	// rangeLineEdge は線の下に敷く暗い縁取り。明るい地形の上でも線を読めるようにする。
	rangeLineEdge = watchEdge
)

// 線をマスの内側に置く量（px）。固定の線の内側にカーソルの線を置き、同じマスで重ならないようにする（DES-UI-29）。
const (
	watchedInset = 3
	cursorInset  = 8
	// rangeDash と rangeGap は破線の 1 つの線と空きの長さ（px）。
	rangeDash = 4
	rangeGap  = 3
)

// enemyRangeLayers は、全敵の塗り → 固定の縁（実線・破線）→ カーソルの縁（実線・破線）の順に層を並べる（DES-UI-29）。
// 空の層は入れない。
func enemyRangeLayers(danger, watchedMove, watchedAttack, cursorMove, cursorAttack map[tactics.Pos]bool) []EnemyRangeLayer {
	var out []EnemyRangeLayer
	for _, l := range []EnemyRangeLayer{
		{RangeAll, StrokeFill, danger},
		{RangeWatched, StrokeSolid, watchedMove},
		{RangeWatched, StrokeDashed, watchedAttack},
		{RangeCursor, StrokeSolid, cursorMove},
		{RangeCursor, StrokeDashed, cursorAttack},
	} {
		if len(l.Cells) > 0 {
			out = append(out, l)
		}
	}
	return out
}

// EnemyRangeLayers は、戦闘の盤面に重ねる敵の範囲の層を描く順に返す（DES-UI-29）。
// どれも今の盤面（Battle）の Reachable と攻撃の射程で求め、UI で独自の移動費は計算しない。
func (c *Controller) EnemyRangeLayers() []EnemyRangeLayer {
	return c.enemyRangeLayersWithDanger(c.Danger())
}

// enemyRangeLayersWithDanger は、同じフレーム内で得た全敵の危険範囲を再利用する（DES-UI-29）。
func (c *Controller) enemyRangeLayersWithDanger(danger map[tactics.Pos]bool) []EnemyRangeLayer {
	wm, wa := c.WatchedFrames()
	cm, ca := c.CursorFrames()
	return enemyRangeLayers(danger, wm, wa, cm, ca)
}

// CursorFrames は、カーソルを合わせた敵の移動できるマス（青白の実線）と攻撃できるマス（青白の破線）を返す（DES-UI-29）。
// Idle・Selected・Moved で、カーソルが生きている敵の上にある間だけ出す（固定は入力を待つ状態でずっと出す。E-124）。固定の集まりは変えない。
func (c *Controller) CursorFrames() (move, attack map[tactics.Pos]bool) {
	e := c.shownEnemy()
	if e < 0 {
		return nil, nil
	}
	return c.ranges(e)
}

// CursorEnemy は、カーソルを合わせて範囲を出している敵の添字を返す（DES-UI-29）。出していなければ -1。
func (c *Controller) CursorEnemy() int { return c.shownEnemy() }

// WatchedPins は、固定した敵のうち範囲を出している敵の位置を、添字の順に返す（DES-UI-29）。金のピンを描く位置。
func (c *Controller) WatchedPins() []tactics.Pos {
	var out []tactics.Pos
	units := c.battle.Units()
	for _, i := range c.shownWatched() {
		out = append(out, units[i].Pos)
	}
	return out
}

// WatchedCount は、固定した敵のうち今の盤面で生きている敵の数を返す（DES-UI-29）。右欄の「固定 n」に使う。
func (c *Controller) WatchedCount() int { return len(c.aliveWatched()) }

// rangeAccepts は、範囲の操作（Q・固定の解除）を受け付ける状態かを返す（DES-UI-29、DES-IT07 3.1 節）。
// 再生中（Playing）・勝敗が決まった Over・巻き戻しの一覧の間は受け付けない。
func (c *Controller) rangeAccepts() bool {
	return !c.rewind.open && c.state != Playing && c.state != Over
}

// ClearWatched は、固定した敵をまとめて外す（DES-UI-29）。範囲の状態の「消す」ボタンの処理。
// 選択・全敵の危険範囲・戦闘の状態は変えない。
func (c *Controller) ClearWatched() {
	if c.rangeAccepts() {
		c.watched = nil
	}
}

// 右欄の操作行の範囲の状態と「消す」ボタンの位置（DES-IT07 3.2・3.4 節）。
// 範囲の状態は (520,304)〜(664,336) に「全敵 Q」と「固定 n」の 2 行、「消す」は操作行の右端 (752,304)〜(824,336)。
var (
	rangeStatusRect = image.Rect(520, 304, 664, 336)
	rangeClearRect  = image.Rect(752, 304, 824, 336)
)

// RangeStatusView は右欄の操作行に出す敵の範囲の状態（DES-UI-29）。
type RangeStatusView struct {
	// Danger は「全敵 Q」と ON・OFF、Watched は「固定 n」。
	Danger, Watched InfoText
	// Clear は固定を一括で外すボタン。Enabled は固定した敵がいて、受け付ける状態であること。
	Clear SceneButton
	// HideTexts・HideClear は、情報欄の中身と重なるので「全敵 Q」「固定 n」・「消す」を描かないこと（REV-096 F-03）。
	// 今の情報欄は y = 336 まで使うので、武器 5 本の「能力」のページなどで地形の行が操作行に来る。情報欄を
	// y < 304 に収める #166 までの手当て。描かない「消す」はクリックも受けない。
	HideTexts, HideClear bool
}

// RangeStatusIn は、情報欄 v を描いた上に重ならない形の範囲の状態を返す（DES-UI-29、REV-096 F-03）。
func (c *Controller) RangeStatusIn(v InfoView) RangeStatusView {
	s := c.RangeStatus()
	for _, t := range []InfoText{s.Danger, s.Watched} {
		if infoOverlaps(v, image.Rect(t.X, t.Y, t.X+textWidth(t.Text), t.Y+lineHeight)) {
			s.HideTexts = true
		}
	}
	s.HideClear = infoOverlaps(v, s.Clear.Rect)
	return s
}

// infoOverlaps は、情報欄 v の文・絵・棒・帯のどれかが r と重なるかを返す。
func infoOverlaps(v InfoView, r image.Rectangle) bool {
	var rects []image.Rectangle
	if v.Banner != nil {
		rects = append(rects, v.Banner.Rect)
	}
	for _, t := range v.Texts {
		scale := 1
		if t.Big {
			scale = 2
		}
		rects = append(rects, image.Rect(t.X, t.Y, t.X+scale*textWidth(t.Text), t.Y+scale*lineHeight))
	}
	for _, ic := range v.Icons {
		rects = append(rects, image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size))
	}
	for _, b := range v.Bars {
		rects = append(rects, image.Rect(b.X, b.Y, b.X+max(b.Fill, b.Frame), b.Y+barHeight))
	}
	for _, q := range rects {
		if q.Overlaps(r) {
			return true
		}
	}
	return false
}

// RangeStatus は右欄の操作行の範囲の状態を返す（DES-UI-29）。
func (c *Controller) RangeStatus() RangeStatusView {
	state := "OFF"
	if c.danger {
		state = "ON"
	}
	n := c.WatchedCount()
	return RangeStatusView{
		Danger:  InfoText{X: rangeStatusRect.Min.X, Y: rangeStatusRect.Min.Y, Text: "全敵 Q " + state, Style: StyleLabel},
		Watched: InfoText{X: rangeStatusRect.Min.X, Y: rangeStatusRect.Min.Y + lineHeight, Text: fmt.Sprintf("固定 %d", n), Style: StyleLabel},
		Clear:   SceneButton{Label: "消す", Rect: rangeClearRect, Enabled: n > 0 && c.rangeAccepts()},
	}
}

// RangeClick は「消す」ボタンのクリックを受ける（DES-UI-29）。ボタンの中なら真を返し、盤面・情報欄には渡さない。
// 巻き戻しの一覧の間は巻き戻しの操作を優先するので受けない（偽を返す）。
func (c *Controller) RangeClick(x, y int) bool {
	if c.rewind.open || !image.Pt(x, y).In(rangeClearRect) || c.RangeStatusIn(c.BattleInfoView(x, y)).HideClear {
		return false
	}
	c.ClearWatched()
	return true
}

// nameMarkRect は、情報欄 v で名前 name の文の右に置く印（10×10）の範囲を返す（DES-UI-29）。
// カーソルの敵の範囲と同じ青白の印で、どの敵の範囲を出しているかを結ぶ。名前の文がなければ ok が偽。
func nameMarkRect(v InfoView, name string) (image.Rectangle, bool) {
	for _, t := range v.Texts {
		if t.Text != name || t.Big {
			continue
		}
		x := t.X + textWidth(t.Text) + 4
		y := t.Y + (lineHeight-10)/2
		return image.Rect(x, y, x+10, y+10), true
	}
	return image.Rectangle{}, false
}

// cursorNameMark は、カーソルの敵の範囲を出しているとき、情報欄 v に出ているその敵の名前の横の印の範囲を返す（DES-UI-29）。
func (c *Controller) cursorNameMark(v InfoView) (image.Rectangle, bool) {
	e := c.shownEnemy()
	if e < 0 {
		return image.Rectangle{}, false
	}
	return nameMarkRect(v, unitName(c.battle.Units()[e]))
}

// rangeFrameRect は、マスの画面の左上 pt に、内側へ inset だけ寄せた線の四角を返す（DES-UI-29）。
func rangeFrameRect(pt image.Point, inset int) image.Rectangle {
	return image.Rect(pt.X+inset, pt.Y+inset, pt.X+CellSize-inset, pt.Y+CellSize-inset)
}

// dashedSegments は、四角 r の 4 辺を、長さ rangeDash・空き rangeGap の破線の線分に分けて返す（DES-UI-29）。
// 各辺の始めから並べ、辺の終わりで切る。
func dashedSegments(r image.Rectangle) [][2]image.Point {
	var out [][2]image.Point
	edges := [][2]image.Point{
		{r.Min, image.Pt(r.Max.X, r.Min.Y)},
		{image.Pt(r.Max.X, r.Min.Y), r.Max},
		{r.Max, image.Pt(r.Min.X, r.Max.Y)},
		{image.Pt(r.Min.X, r.Max.Y), r.Min},
	}
	for _, e := range edges {
		d := e[1].Sub(e[0])
		n := max(abs(d.X), abs(d.Y))
		if n == 0 {
			continue
		}
		step := image.Pt(dashSign(d.X), dashSign(d.Y))
		for s := 0; s < n; s += rangeDash + rangeGap {
			end := min(s+rangeDash, n)
			out = append(out, [2]image.Point{e[0].Add(step.Mul(s)), e[0].Add(step.Mul(end))})
		}
	}
	return out
}

// dashSign は v の符号（-1・0・1）を返す。
func dashSign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// RangeLayers は、配置の盤面に重ねる敵の範囲の層を描く順に返す（DES-UI-29、Q-DES07-09）。
// 配置では固定を使わず、全敵（Danger）とカーソルの敵（Blue・Red）だけを出す。
func (d DeployView) RangeLayers() []EnemyRangeLayer {
	return enemyRangeLayers(d.Danger, nil, nil, d.Blue, d.Red)
}

// minimapDanger は、危険範囲のマスを、ミニマップのマスの範囲にして返す（DES-UI-29）。
// 1 マス 4 px では線を描けないので、マスの色だけで示す。並びは描くときの順に影響しないよう、上から左から並べる。
func minimapDanger(l MinimapLayout, danger map[tactics.Pos]bool) []image.Rectangle {
	var out []image.Rectangle
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			if p := (tactics.Pos{X: x, Y: y}); danger[p] {
				out = append(out, l.CellRect(p))
			}
		}
	}
	return out
}

// minimapPins は、固定した敵の位置のマスの範囲を返す（DES-UI-29）。金の印で描く。
func minimapPins(l MinimapLayout, pins []tactics.Pos) []image.Rectangle {
	var out []image.Rectangle
	for _, p := range pins {
		if p.X < 0 || p.Y < 0 || p.X >= l.Width || p.Y >= l.Height {
			continue
		}
		out = append(out, l.CellRect(p))
	}
	return out
}

// pinShape は、固定した敵のマスの画面の左上 pt に描く金のピンの頭と軸の範囲を返す（DES-UI-29、DES-UI-33）。
// 印の置き場所の分け方（指揮官は左上、固定は右上、装備は右下）に従い、右上に置く。
// ボスは右上に ★ があるので、★ を隠さないよう、その左に置く。
func pinShape(pt image.Point, boss bool) (head, stem image.Rectangle) {
	x := pt.X + CellSize - 9
	if boss {
		x -= 12
	}
	head = image.Rect(x, pt.Y+1, x+6, pt.Y+7)
	stem = image.Rect(x+2, pt.Y+7, x+4, pt.Y+11)
	return head, stem
}

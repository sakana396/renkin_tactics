package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// drawEnemyRanges は敵の範囲の層を、並びの順（全敵の塗り → 固定の縁 → カーソルの縁）に重ねる（DES-UI-29）。
// 塗りは下のユニットが読める薄さにし、線は暗い縁取りの上に描く。
func (g *Game) drawEnemyRanges(dst *ebiten.Image, layers []EnemyRangeLayer) {
	for _, l := range layers {
		for p := range l.Cells {
			if !g.drawCamera.Visible(p) {
				continue
			}
			g.drawRangeCell(dst, l, g.drawCamera.Screen(p))
		}
	}
}

// drawRangeCell は層 l の 1 マスを、画面の左上 pt に描く（DES-UI-29）。
func (g *Game) drawRangeCell(dst *ebiten.Image, l EnemyRangeLayer, pt image.Point) {
	if l.Stroke == StrokeFill {
		vector.DrawFilledRect(dst, float32(pt.X), float32(pt.Y), float32(CellSize), float32(CellSize), dangerFill, false)
		return
	}
	clr, inset := color.Color(watchedLine), watchedInset
	if l.Owner == RangeCursor {
		clr, inset = cursorLine, cursorInset
	}
	r := rangeFrameRect(pt, inset)
	if l.Stroke == StrokeSolid {
		strokeImageRect(dst, r, 4, rangeLineEdge)
		strokeImageRect(dst, r, 2, clr)
		return
	}
	segs := dashedSegments(r)
	for _, s := range segs {
		vector.StrokeLine(dst, float32(s[0].X), float32(s[0].Y), float32(s[1].X), float32(s[1].Y), 4, rangeLineEdge, false)
	}
	for _, s := range segs {
		vector.StrokeLine(dst, float32(s[0].X), float32(s[0].Y), float32(s[1].X), float32(s[1].Y), 2, clr, false)
	}
}

// strokeImageRect は四角 r の枠を太さ w で描く。
func strokeImageRect(dst *ebiten.Image, r image.Rectangle, w float32, clr color.Color) {
	vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), w, clr, false)
}

// drawWatchedPins は、固定した敵の頭上（右上）に金の小さなピンを描く（DES-UI-29）。
func (g *Game) drawWatchedPins(dst *ebiten.Image, pins []tactics.Pos, units []tactics.Unit) {
	for _, p := range pins {
		if !g.drawCamera.Visible(p) {
			continue
		}
		boss := false
		for _, u := range units {
			if u.Alive && u.Pos == p {
				boss = u.Boss
				break
			}
		}
		head, stem := pinShape(g.drawCamera.Screen(p), boss)
		fillImageRect(dst, head.Inset(-scaled(1)), rangeLineEdge)
		fillImageRect(dst, stem, gold)
		fillImageRect(dst, head, gold)
	}
}

// fillImageRect は四角 r を塗る。
func fillImageRect(dst *ebiten.Image, r image.Rectangle, clr color.Color) {
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), clr, false)
}

// drawRangeStatus は右欄の操作行に「全敵 Q」「固定 n」と「消す」ボタンを描く（DES-UI-29）。
func (g *Game) drawRangeStatus(dst *ebiten.Image, v RangeStatusView) {
	if !v.HideTexts {
		for _, t := range []InfoText{v.Danger, v.Watched} {
			g.label(dst, t.Text, t.X, t.Y, infoStyleColor(t.Style))
		}
	}
	if v.HideClear {
		return
	}
	b := v.Clear
	fill, border, text := menuColor, menuBorder, white
	if !b.Enabled {
		fill, border, text = bgColor, muted, muted
	}
	fillImageRect(dst, b.Rect, fill)
	vector.StrokeRect(dst, float32(b.Rect.Min.X), float32(b.Rect.Min.Y), float32(b.Rect.Dx()), float32(b.Rect.Dy()), 1, border, false)
	g.label(dst, b.Label, b.Rect.Min.X+(b.Rect.Dx()-textWidth(b.Label))/2, b.Rect.Min.Y+(b.Rect.Dy()-lineHeight)/2, text)
}

// drawNameMark は情報欄の名前の横に、カーソルの敵の範囲と同じ青白の印を描く（DES-UI-29）。
func (g *Game) drawNameMark(dst *ebiten.Image, r image.Rectangle) {
	strokeImageRect(dst, r, 2, cursorLine)
}

// drawMinimapDanger は、ミニマップに危険範囲のマスの薄い色（地形の上）を描く（DES-UI-29）。
func drawMinimapDanger(dst *ebiten.Image, cells []image.Rectangle) {
	for _, r := range cells {
		fillImageRect(dst, r, minimapDangerFill)
	}
}

// drawMinimapPins は、ミニマップの固定した敵のマスを金の縁で囲む（DES-UI-29）。ユニットの上に描く。
func drawMinimapPins(dst *ebiten.Image, cells []image.Rectangle) {
	for _, r := range cells {
		strokeImageRect(dst, r.Inset(-1), 1, gold)
	}
}

package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// drawPrepareHiRes は高解像度の準備の画面を、窓の全体に描く（DES-NG-11）。枠・背景・棒はコードで描く（Q-P4）。
func (g *Game) drawPrepareHiRes(dst *ebiten.Image, v PrepareHiResView) {
	dst.Fill(hrBg)
	for _, b := range v.Boxes {
		hrDrawBox(dst, b)
	}
	for _, p := range v.Portraits {
		g.drawHRPortrait(dst, p)
	}
	for _, b := range v.Overlays {
		hrDrawBox(dst, b)
	}
	for _, t := range v.Tiles {
		g.drawTileSize(dst, t.Tile, t.Rect.Min.X, t.Rect.Min.Y, t.Rect.Dx())
	}
	for _, d := range v.Dots {
		vector.DrawFilledCircle(dst, float32(d.X), float32(d.Y), float32(d.R), d.Color, true)
		vector.StrokeCircle(dst, float32(d.X), float32(d.Y), float32(d.R), 1, hrBorder, true)
	}
	for _, gl := range v.Glyphs {
		hrDrawGlyph(dst, gl)
	}
	for _, b := range v.Bars {
		r := b.Frame
		hrFill(dst, r, color.NRGBA{16, 26, 31, 255})
		hrFill(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+b.Fill, r.Max.Y), b.Color)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, hrBorder, false)
	}
	for _, m := range v.Marks {
		r := m.Rect
		fill := hrBg
		if m.On {
			fill = hrAmber
		}
		hrFill(dst, r, fill)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, hrBorder, false)
		if m.On {
			x, y := float32(r.Min.X), float32(r.Min.Y)
			vector.StrokeLine(dst, x+5, y+11, x+9, y+16, 3, hrDark, true)
			vector.StrokeLine(dst, x+9, y+16, x+17, y+5, 3, hrDark, true)
		}
	}
	for _, t := range v.Texts {
		hrDrawText(dst, t)
	}
}

// hrDrawGlyph は ▲▼ の三角と → の矢印を、フォントの字形を使わず vector で描く（F-05）。
func hrDrawGlyph(dst *ebiten.Image, g hrGlyph) {
	r := g.Rect
	x0, y0, x1, y1 := float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)
	switch g.Kind {
	case hrGlyphUp:
		hrFillTriangle(dst, x0, y1, x1, y1, (x0+x1)/2, y0, g.Color)
	case hrGlyphDown:
		hrFillTriangle(dst, x0, y0, x1, y0, (x0+x1)/2, y1, g.Color)
	case hrGlyphArrow:
		cy := (y0 + y1) / 2
		head := (y1 - y0) / 2
		vector.StrokeLine(dst, x0, cy, x1-head, cy, 2, g.Color, true)
		hrFillTriangle(dst, x1-head-2, y0, x1-head-2, y1, x1, cy, g.Color)
	}
}

var hrWhite = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// hrFillTriangle は 3 点の三角を塗る（vector.Path を三角形の描画へ渡す）。
func hrFillTriangle(dst *ebiten.Image, ax, ay, bx, by, cx, cy float32, c color.NRGBA) {
	var p vector.Path
	p.MoveTo(ax, ay)
	p.LineTo(bx, by)
	p.LineTo(cx, cy)
	p.Close()
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, hrWhite, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// hrSlotFill は空きの枠・箱の塗り（暗い）、hrPipFill は名簿のカードの使っている枠の塗り（琥珀）。
var (
	hrSlotFill = color.NRGBA{22, 34, 40, 255}
	hrPipFill  = hrAmber
)

func hrFill(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

func hrDrawBox(dst *ebiten.Image, b hrBox) {
	r := b.Rect
	stroke := func(c color.Color, w float32) {
		vector.StrokeRect(dst, float32(r.Min.X)+w/2, float32(r.Min.Y)+w/2, float32(r.Dx())-w, float32(r.Dy())-w, w, c, false)
	}
	switch b.Kind {
	case hrBoxPanel:
		hrFill(dst, r, hrPanel)
		stroke(hrBorder, 2)
	case hrBoxCard:
		hrFill(dst, r, hrCardFil)
		stroke(hrBorder, 1)
	case hrBoxCardSel:
		hrFill(dst, r, hrCardSel)
		stroke(hrAqua, 3)
	case hrBoxCardDim:
		hrFill(dst, r, hrCardDim)
		stroke(color.NRGBA{90, 70, 40, 255}, 1)
	case hrBoxSlot:
		hrFill(dst, r, hrSlotFill)
		stroke(color.NRGBA{80, 90, 88, 255}, 1)
	case hrBoxSlotSel:
		hrFill(dst, r, color.NRGBA{22, 34, 40, 255})
		stroke(hrAmber, 2)
	case hrBoxRow:
		hrFill(dst, r, hrCardFil)
		stroke(hrBorder, 1)
	case hrBoxRowSel:
		hrFill(dst, r, hrCardSel)
		stroke(hrAqua, 2)
	case hrBoxChip:
		hrFill(dst, r, color.NRGBA{22, 34, 40, 255})
		stroke(hrBorder, 1)
	case hrBoxPill:
		hrFill(dst, r, hrPanel)
		stroke(hrAmber, 2)
	case hrBoxBand:
		hrFill(dst, r, color.NRGBA{12, 18, 22, 235})
	case hrBoxLine:
		hrFill(dst, r, hrBorder)
	case hrBoxButton:
		hrFill(dst, r, hrAmber)
		stroke(color.NRGBA{240, 200, 120, 255}, 2)
	case hrBoxButtonOff:
		hrFill(dst, r, color.NRGBA{50, 60, 60, 255})
		stroke(hrDim, 1)
	case hrBoxFrame:
		stroke(hrAmber, 2)
	case hrBoxPip:
		hrFill(dst, r, hrPipFill)
		stroke(hrBorder, 1)
	}
}

// hrDrawText は文を描く。Scale 2 は、等倍の文字を 2 倍に拡大する（補間なしの最近傍で、ぼやけない）。
func hrDrawText(dst *ebiten.Image, t hrTextItem) {
	sc := float64(max(t.Scale, 1))
	op := &text.DrawOptions{}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(float64(t.X), float64(t.Y))
	op.ColorScale.ScaleWithColor(t.Color)
	text.Draw(dst, t.Text, face, op)
}

// drawHRPortrait は肖像（なければユニットの絵、なければタイル）を、箱の大きさに CPU で縮めて描く（DES-NG-08 の縮小）。
func (g *Game) drawHRPortrait(dst *ebiten.Image, p hrPortrait) {
	w, h := p.Rect.Dx(), p.Rect.Dy()
	var eb *ebiten.Image
	if a := g.artStore(); a != nil && !p.Key.Empty() {
		if path, ok := a.PortraitPath(p.Key.Name); ok && p.Key.Player {
			eb = a.Shrunk(path, false, w, h)
		}
		if eb == nil {
			if path, ok := a.UnitArtPath(p.Key); ok {
				eb = a.Shrunk(path, true, w, h)
			}
		}
	}
	if eb == nil {
		hrFill(dst, p.Rect, color.NRGBA{22, 34, 40, 255})
		size := min(w, h, 4*assets.TileSize)
		g.drawTileSize(dst, p.Tile, p.Rect.Min.X+(w-size)/2, p.Rect.Min.Y+(h-size)/2, size)
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.Rect.Min.X), float64(p.Rect.Min.Y))
	if p.Dim {
		op.ColorScale.Scale(0.5, 0.5, 0.5, 1)
	}
	dst.DrawImage(eb, op)
}

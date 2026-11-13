package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// backdropDim は背景の上に重ねる暗い膜の色。文字が読めるよう、半分より濃くする（DES-NG-08）。
var backdropDim = color.NRGBA{A: 160}

// artStore は画面が使う画像の読み込みを返す（DES-NG-08）。場面の進行を持たない描画だけの Game では nil になりうる。
func (g *Game) artStore() *ArtStore {
	if g == nil || g.scenes == nil {
		return nil
	}
	return g.scenes.art
}

// drawArtIn は画像 eb を範囲 r に拡大縮小して描く。scale が 1 未満なら暗く描く（DES-NG-08）。
func drawArtIn(dst, eb *ebiten.Image, r image.Rectangle, scale float32) {
	b := eb.Bounds()
	if r.Empty() || b.Empty() {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(r.Dx())/float64(b.Dx()), float64(r.Dy())/float64(b.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	if scale < 1 {
		op.ColorScale.Scale(scale, scale, scale, 1)
	}
	dst.DrawImage(eb, op)
}

// drawUnitArt は、ユニットの絵があれば、マス（左上 (px, py)、一辺 CellSize）の上に描いて真を返す（DES-NG-08）。
// なければ何も描かず偽を返すので、呼び出し側は今のタイルを描く。下地・HP の棒・印はそのまま呼び出し側が描く。
func (g *Game) drawUnitArt(dst *ebiten.Image, k ArtKey, px, py int, scale float32) bool {
	a := g.artStore()
	p, ok := a.UnitArtPath(k)
	if !ok {
		return false
	}
	eb := a.Ebiten(p, true)
	if eb == nil {
		return false
	}
	b := eb.Bounds()
	drawArtIn(dst, eb, unitArtRect(px, py, CellSize, b.Dx(), b.Dy()), scale)
	return true
}

// drawInfoIcon は情報欄・場面の絵を描く。肖像（自軍）かユニットの絵があればそれ、なければタイル（DES-NG-08）。
func (g *Game) drawInfoIcon(dst *ebiten.Image, ic InfoIcon) {
	if !ic.Art.Empty() {
		a := g.artStore()
		if p, ok := a.InfoArtPath(ic.Art); ok {
			if eb := a.Ebiten(p, true); eb != nil {
				b := eb.Bounds()
				drawArtIn(dst, eb, unitArtRect(ic.X, ic.Y, ic.Size, b.Dx(), b.Dy()), 1)
				return
			}
		}
	}
	g.drawTileSize(dst, ic.Tile, ic.X, ic.Y, ic.Size)
}

// drawSceneBackdrop は場面の背景を、画面を覆うように描き、暗い膜を重ねる（DES-NG-08）。
// 背景がなければ何もしない。ほかの描画より先に呼ぶ。
func (g *Game) drawSceneBackdrop(dst *ebiten.Image, v SceneView) {
	if v.Backdrop == "" {
		return
	}
	eb := g.artStore().Ebiten(v.Backdrop, false)
	if eb == nil {
		return
	}
	screen := image.Rect(0, 0, ScreenWidth, ScreenHeight)
	b := eb.Bounds()
	drawArtIn(dst.SubImage(screen).(*ebiten.Image), eb, coverRect(screen, b.Dx(), b.Dy()), 1)
	vector.DrawFilledRect(dst, 0, 0, float32(ScreenWidth), float32(ScreenHeight), backdropDim, false)
}

// drawScenePortrait は物語の場面の肖像を描く（DES-NG-08）。
func (g *Game) drawScenePortrait(dst *ebiten.Image, v SceneView) {
	if v.Portrait == nil {
		return
	}
	eb := g.artStore().Ebiten(v.Portrait.Path, true)
	if eb == nil {
		return
	}
	b := eb.Bounds()
	drawArtIn(dst, eb, fitRect(v.Portrait.Rect, b.Dx(), b.Dy()), 1)
}

package ui

import "github.com/hajimehoshi/ebiten/v2"

// drawUnitArt は画像があれば盤面に描く（DES-NG-08）。
func (g *Game) drawUnitArt(dst *ebiten.Image, key ArtKey, x, y int, scale float32) bool {
	if g.scenes == nil || g.scenes.art == nil {
		return false
	}
	a := g.scenes.art
	p, ok := a.UnitArtPath(key)
	if !ok {
		return false
	}
	img := a.Ebiten(p, true)
	if img == nil {
		return false
	}
	b := img.Bounds()
	r := unitArtRect(x, y, CellSize, b.Dx(), b.Dy())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(r.Dx())/float64(b.Dx()), float64(r.Dy())/float64(b.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	op.ColorScale.Scale(scale, scale, scale, 1)
	dst.DrawImage(img, op)
	return true
}

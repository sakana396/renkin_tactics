package ui

import (
	"image"
	"image/color"
)

// alphaCutoff は絵の「中身」とみなす不透明度の下限（0〜255）。縁のうすい影を外すための値（DES-NG-08）。
const alphaCutoff = 16

// artBounds は src の中で、不透明度が alphaCutoff を超える画素を囲む最小の範囲を返す（DES-NG-08）。
// 透明な画素しかなければ src の全体を返す。
func artBounds(src image.Image) image.Rectangle {
	b := src.Bounds()
	out := image.Rectangle{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a>>8 > alphaCutoff {
				p := image.Rect(x, y, x+1, y+1)
				if out.Empty() {
					out = p
				} else {
					out = out.Union(p)
				}
			}
		}
	}
	if out.Empty() {
		return b
	}
	return out
}

// faceCrop は、ユニットの絵から頭と肩を含む正方形の範囲を返す（DES-NG-08）。
// 絵の中身（artBounds）の上端から、一辺が「中身の高さの 55 %」（ただし中身の幅まで）の正方形を、中身の横の中央に置く。
func faceCrop(src image.Image) image.Rectangle {
	b := artBounds(src)
	side := min(b.Dy()*55/100, b.Dx())
	if side < 1 {
		return b
	}
	x := b.Min.X + (b.Dx()-side)/2
	return image.Rect(x, b.Min.Y, x+side, b.Min.Y+side)
}

// shrinkImage は src の範囲 crop を、w × h の画像に面積平均で縮める（DES-NG-08）。
// 色は不透明度で重み付けして平均する（透明な画素の色が縁ににじまない）ので、透過は保たれる。
// 同じ入力からはいつも同じ結果になる。crop は src の範囲に切り詰め、拡大のときも同じ式で補う。
// w・h が 0 以下、または crop が空なら nil。
func shrinkImage(src image.Image, crop image.Rectangle, w, h int) *image.NRGBA {
	crop = crop.Intersect(src.Bounds())
	if w <= 0 || h <= 0 || crop.Empty() {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	sw, sh := float64(crop.Dx()), float64(crop.Dy())
	for dy := 0; dy < h; dy++ {
		y0, y1 := sh*float64(dy)/float64(h), sh*float64(dy+1)/float64(h)
		for dx := 0; dx < w; dx++ {
			x0, x1 := sw*float64(dx)/float64(w), sw*float64(dx+1)/float64(w)
			var r, g, b, a, wsum float64
			for sy := int(y0); sy < crop.Dy() && float64(sy) < y1; sy++ {
				wy := min(y1, float64(sy+1)) - max(y0, float64(sy))
				for sx := int(x0); sx < crop.Dx() && float64(sx) < x1; sx++ {
					wt := wy * (min(x1, float64(sx+1)) - max(x0, float64(sx)))
					pr, pg, pb, pa := src.At(crop.Min.X+sx, crop.Min.Y+sy).RGBA()
					// RGBA() は不透明度を掛けた値（0〜65535）。そのまま重みを掛けて足す。
					r += wt * float64(pr)
					g += wt * float64(pg)
					b += wt * float64(pb)
					a += wt * float64(pa)
					wsum += wt
				}
			}
			if wsum == 0 || a == 0 {
				continue
			}
			// 不透明度を掛けた平均から、掛ける前の色を求める。
			out.SetNRGBA(dx, dy, color.NRGBA{
				R: uint8(min(255, r/a*255+0.5)),
				G: uint8(min(255, g/a*255+0.5)),
				B: uint8(min(255, b/a*255+0.5)),
				A: uint8(min(255, a/wsum/257+0.5)),
			})
		}
	}
	return out
}

package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// 画像の種類ごとの置き場と大きさ（DES-NG-08 3 章の推奨の大きさ）。
type kindSpec struct {
	Dir    string // art の下のディレクトリ。title は空
	W, H   int
	Chroma bool // 既定で緑の背景を透過にするか
}

var kinds = map[string]kindSpec{
	"units":     {Dir: "units", W: 128, H: 128, Chroma: true},
	"portraits": {Dir: "portraits", W: 512, H: 512},
	"story":     {Dir: "story", W: 832, H: 576},
	"title":     {W: 832, H: 576},
}

// クロマキーの決まり。境目は画像の縁の緑らしさから決める（REV-118 R-01）。緑かぶりの帯と余白は試作の台帳 art/units/aoi.png の行の手順に合わせた。
const (
	spillBand     = 6    // 透明の画素から何 px までを縁の帯とし、緑かぶりを除くか
	trimAlpha     = 10   // 切り出しの範囲に数える α の下限（約 0.04）
	trimMargin    = 0.06 // 切り出した中身に足す余白（中身の長い辺に対する割合）
	residualAlpha = 64   // 縮小のあと、これ以下の α で緑が残る画素は透明にする
	residualKey   = 8    // その「緑が残る」とみなす k の下限
)

// greenness は緑らしさ k = G − max(R,B)。
func greenness(r, g, b uint8) int {
	return int(g) - int(max(r, b))
}

// keyAlpha は緑らしさ k から α を決める。
func keyAlpha(k, keyOpaque, keyClear int) uint8 {
	switch {
	case k <= keyOpaque:
		return 255
	case k >= keyClear:
		return 0
	default:
		return uint8((keyClear - k) * 255 / (keyClear - keyOpaque))
	}
}

// toNRGBA は画像を非乗算の NRGBA に写す（原点は 0,0）。
func toNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// chromaKey は緑の背景を透過にし、緑かぶりを除く。
// ① k から α を決める。② α が 255 未満の画素と、透明の画素から spillBand px 以内の画素は、G を max(R,B) まで下げる。
func chromaKey(src image.Image) (*image.NRGBA, error) {
	img := toNRGBA(src)
	w, h := img.Rect.Dx(), img.Rect.Dy()
	var hist [511]int
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= 5 && y >= 5 && x < w-5 && y < h-5 {
				continue
			}
			p := img.NRGBAAt(x, y)
			hist[greenness(p.R, p.G, p.B)+255]++
			n++
		}
	}
	lower, count := -255, 0
	for i, v := range hist {
		count += v
		if count*100 >= n {
			lower = i - 255
			break
		}
	}
	keyClear := lower
	keyOpaque := keyClear / 2
	if keyClear < 40 {
		return nil, errors.New("縁の背景が十分に緑ではありません")
	}
	foreground := false
	for y := 5; y < h-5 && !foreground; y++ {
		for x := 5; x < w-5; x++ {
			p := img.NRGBAAt(x, y)
			if p.A > 0 && greenness(p.R, p.G, p.B) <= keyOpaque {
				foreground = true
				break
			}
		}
	}
	if !foreground {
		return nil, errors.New("背景と中身の色を分けられません")
	}
	clear := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			p := img.Pix[i : i+4]
			a := keyAlpha(greenness(p[0], p[1], p[2]), keyOpaque, keyClear)
			p[3] = uint8(int(p[3]) * int(a) / 255)
			clear[y*w+x] = p[3] == 0
		}
	}
	band := dilate(clear, w, h, spillBand)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			p := img.Pix[i : i+4]
			if p[3] < 255 || band[y*w+x] {
				p[1] = min(p[1], max(p[0], p[2]))
			}
		}
	}
	return img, nil
}

// dilate は真の画素を n 回、8 近傍へ広げる。
func dilate(mask []bool, w, h, n int) []bool {
	cur := append([]bool(nil), mask...)
	for ; n > 0; n-- {
		next := append([]bool(nil), cur...)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if !cur[y*w+x] {
					continue
				}
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && ny >= 0 && nx < w && ny < h {
							next[ny*w+nx] = true
						}
					}
				}
			}
		}
		cur = next
	}
	return cur
}

// contentBounds は α が trimAlpha を超える画素を囲む最小の四角。なければ空。
func contentBounds(img *image.NRGBA) image.Rectangle {
	r := image.Rectangle{}
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if img.NRGBAAt(x, y).A > trimAlpha {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

// trimSquare は中身の範囲で切り出し、余白を足して足元をそろえた正方形にする。
// 一辺は中身の長い辺の (1 + 2×trimMargin) 倍。中身は横の中央、下端から長い辺×trimMargin の位置に置く。
func trimSquare(img *image.NRGBA) (*image.NRGBA, error) {
	c := contentBounds(img)
	if c.Empty() {
		return nil, errors.New("透過にしたら中身が残りませんでした（背景が緑でない、または全体が緑です）")
	}
	long := max(c.Dx(), c.Dy())
	pad := int(math.Round(float64(long) * trimMargin))
	side := long + 2*pad
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	at := image.Pt((side-c.Dx())/2, side-pad-c.Dy())
	draw.Draw(dst, image.Rectangle{Min: at, Max: at.Add(c.Size())}, img, c.Min, draw.Src)
	return dst, nil
}

// scaleTo は乗算済みアルファで w×h に縮める（Catmull-Rom）。
func scaleTo(src image.Image, sr image.Rectangle, w, h int) *image.NRGBA {
	pre := image.NewRGBA(sr.Sub(sr.Min))
	draw.Draw(pre, pre.Bounds(), src, sr.Min, draw.Src)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), pre, pre.Bounds(), xdraw.Src, nil)
	return toNRGBA(out)
}

// coverRect は w×h の比になるよう、画像の中央を残して切る四角（はみ出しは中央で切る。DES-NG-08 4.4 節の coverRect と同じ考え方）。
func coverRect(b image.Rectangle, w, h int) image.Rectangle {
	sw, sh := b.Dx(), b.Dy()
	// sw/sh と w/h を比べ、長すぎる向きを切る。
	if sw*h > sh*w {
		cw := int(math.Round(float64(sh) * float64(w) / float64(h)))
		x := b.Min.X + (sw-cw)/2
		return image.Rect(x, b.Min.Y, x+cw, b.Max.Y)
	}
	ch := int(math.Round(float64(sw) * float64(h) / float64(w)))
	y := b.Min.Y + (sh-ch)/2
	return image.Rect(b.Min.X, y, b.Max.X, y+ch)
}

// cleanResidual は縮小のあとの半透明の縁を整える。α が residualAlpha 以下で緑らしさが residualKey 以上の画素は透明にし、
// 残る半透明の画素は G を max(R,B) まで下げる。
func cleanResidual(img *image.NRGBA) {
	for i := 0; i+3 < len(img.Pix); i += 4 {
		p := img.Pix[i : i+4]
		if p[3] == 0 || p[3] == 255 {
			continue
		}
		if p[3] <= residualAlpha && greenness(p[0], p[1], p[2]) >= residualKey {
			p[0], p[1], p[2], p[3] = 0, 0, 0, 0
			continue
		}
		p[1] = min(p[1], max(p[0], p[2]))
	}
}

// processImage は種類に合わせて画像を加工する。chroma が真なら緑の背景を透過にして中身で切り出す。
// 透過にしないときは、中央を残して種類の比に切り、縮める。
func processImage(src image.Image, spec kindSpec, chroma bool) (*image.NRGBA, error) {
	if src.Bounds().Empty() {
		return nil, errors.New("画像の大きさが 0 です")
	}
	if !chroma {
		return scaleTo(src, coverRect(src.Bounds(), spec.W, spec.H), spec.W, spec.H), nil
	}
	if spec.W != spec.H {
		return nil, fmt.Errorf("透過の切り出しは正方形の種類（units・portraits）だけです（%d×%d）", spec.W, spec.H)
	}
	keyed, err := chromaKey(src)
	if err != nil {
		return nil, err
	}
	sq, err := trimSquare(keyed)
	if err != nil {
		return nil, err
	}
	out := scaleTo(sq, sq.Bounds(), spec.W, spec.H)
	cleanResidual(out)
	return out, nil
}

// residualGreen は元画像の縁の色、または緑かぶり除去後の色に近い残存画素を数える。
func residualGreen(img *image.NRGBA, src image.Image) int {
	b := src.Bounds()
	var rs, gs, bs, n int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if x >= b.Min.X+5 && y >= b.Min.Y+5 && x < b.Max.X-5 && y < b.Max.Y-5 {
				continue
			}
			p := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			rs += int(p.R)
			gs += int(p.G)
			bs += int(p.B)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	r, g, bl := rs/n, gs/n, bs/n
	despill := min(g, max(r, bl))
	near := func(a, b int) bool { return a-b <= 20 && b-a <= 20 }
	count := 0
	for i := 0; i+3 < len(img.Pix); i += 4 {
		p := img.Pix[i : i+4]
		if p[3] > 0 && near(int(p[0]), r) && near(int(p[2]), bl) && (near(int(p[1]), g) || near(int(p[1]), despill)) {
			count++
		}
	}
	return count
}

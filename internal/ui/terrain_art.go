package ui

import (
	"hash/fnv"
	"image"
	"image/color"
	"math"
)

// 高解像度の表示（Config.HiRes）の、仮の地形の絵（DES-NG-10 フェーズ 2）。
//
// Go の標準ライブラリの image だけで、48×48 の不透明な画像を地形ごとに作る。
// なめらかにするため 96×96 で描いて 2×2 の平均で縮める。乱数は地形の ID と変種の番号から作る種の自前のもので、
// 同じ引数なら必ず同じ画像になる。AI で作る絵ができたら、media の art/terrain の PNG が優先される（ArtStore.TerrainImage）。
const (
	// terrainSide は仮の地形の画像の一辺（px）。
	terrainSide = 48
	// terrainHi は縮める前に描く一辺（px）。
	terrainHi = 2 * terrainSide
	// terrainVariants は 1 つの地形の変種の数。
	terrainVariants = 4
)

// terrainPlaceholderIDs は、仮の絵がある地形の ID（新作の data/alchemy/terrains.toml）。
var terrainPlaceholderIDs = []string{"plain", "rubble", "bush", "wall", "canal", "facility"}

// hasTerrainPlaceholder は、地形の ID に仮の絵があること。
func hasTerrainPlaceholder(id string) bool {
	for _, p := range terrainPlaceholderIDs {
		if p == id {
			return true
		}
	}
	return false
}

// terrainVariantIndex は、マス (x, y) の変種の番号（0 以上 n 未満）を返す。座標だけで決まる（同じマスはいつも同じ変種）。
// n が 1 以下なら 0。
func terrainVariantIndex(x, y, n int) int {
	if n <= 1 {
		return 0
	}
	h := uint64(uint32(x))*0x9E3779B97F4A7C15 ^ uint64(uint32(y))*0xC2B2AE3D27D4EB4F
	h ^= h >> 29
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 32
	return int(h % uint64(n))
}

// terrainRand は決まった種の擬似乱数（splitmix64）。グローバルな乱数は使わない。
type terrainRand struct{ s uint64 }

func newTerrainRand(id string, variant int) *terrainRand {
	h := fnv.New64a()
	h.Write([]byte(id))
	h.Write([]byte{0, byte(variant)})
	return &terrainRand{s: h.Sum64()}
}

func (r *terrainRand) next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ z>>30) * 0xBF58476D1CE4E5B9
	z = (z ^ z>>27) * 0x94D049BB133111EB
	return z ^ z>>31
}

// f は [0, 1) の値。
func (r *terrainRand) f() float64 { return float64(r.next()>>11) / (1 << 53) }

// rng は [lo, hi) の値。
func (r *terrainRand) rng(lo, hi float64) float64 { return lo + (hi-lo)*r.f() }

// n は [0, n) の整数。
func (r *terrainRand) n(n int) int { return int(r.next() % uint64(n)) }

// 色は低彩度にそろえる。鮮やかなのは水の青緑と琥珀だけ（new-game-visual-draft.md 2.1〜2.2 節）。
var (
	sandBase   = rgb{176, 162, 136}
	sandLight  = rgb{198, 185, 158}
	sandDark   = rgb{132, 120, 100}
	groutColor = rgb{104, 95, 82}
	rubbleGray = rgb{128, 116, 104}
	bushGreen  = rgb{92, 112, 82}
	waterDeep  = rgb{30, 112, 120}
	waterMid   = rgb{48, 150, 152}
	waterLight = rgb{150, 218, 208}
	glowTeal   = rgb{120, 232, 214}
	amberColor = rgb{222, 158, 62}
)

// rgb は 0〜255 の色（計算用）。
type rgb struct{ r, g, b float64 }

func (c rgb) scale(f float64) rgb { return rgb{c.r * f, c.g * f, c.b * f} }

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// canvas は 96×96 の描き場。画素は浮動小数で持ち、最後に 2×2 の平均で 48×48 にする。
type canvas struct {
	pix [terrainHi * terrainHi]rgb
}

func (c *canvas) at(x, y int) *rgb { return &c.pix[y*terrainHi+x] }

// blend は (x, y) に色 col を不透明度 a で重ねる。範囲の外は何もしない。
func (c *canvas) blend(x, y int, col rgb, a float64) {
	if x < 0 || y < 0 || x >= terrainHi || y >= terrainHi || a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	p := c.at(x, y)
	*p = mix(*p, col, a)
}

func (c *canvas) fill(col rgb) {
	for i := range c.pix {
		c.pix[i] = col
	}
}

func (c *canvas) rect(x0, y0, x1, y1 int, col rgb, a float64) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.blend(x, y, col, a)
		}
	}
}

// disc は中心 (cx, cy)・半径 r の楕円（rx, ry）を塗る。
func (c *canvas) ellipse(cx, cy, rx, ry float64, col rgb, a float64) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				c.blend(x, y, col, a)
			}
		}
	}
}

func (c *canvas) disc(cx, cy, r float64, col rgb, a float64) { c.ellipse(cx, cy, r, r, col, a) }

// ring は中心 (cx, cy)・半径 r・太さ w の輪。
func (c *canvas) ring(cx, cy, r, w float64, col rgb, a float64) {
	for y := int(cy - r - w); y <= int(cy+r+w); y++ {
		for x := int(cx - r - w); x <= int(cx+r+w); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if math.Abs(d-r) <= w/2 {
				c.blend(x, y, col, a)
			}
		}
	}
}

// line は太さ w の線分。
func (c *canvas) line(x0, y0, x1, y1, w float64, col rgb, a float64) {
	minX, maxX := int(math.Min(x0, x1)-w), int(math.Max(x0, x1)+w)
	minY, maxY := int(math.Min(y0, y1)-w), int(math.Max(y0, y1)+w)
	dx, dy := x1-x0, y1-y0
	l2 := dx*dx + dy*dy
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/l2))
			}
			if math.Hypot(px-(x0+t*dx), py-(y0+t*dy)) <= w/2 {
				c.blend(x, y, col, a)
			}
		}
	}
}

// grain は全体に細かい明暗のざらつきを足す。
func (c *canvas) grain(r *terrainRand, amount float64) {
	for i := range c.pix {
		c.pix[i] = c.pix[i].scale(1 + r.rng(-amount, amount))
	}
}

// image は 2×2 の平均で 48×48 に縮め、不透明な画像にする。
func (c *canvas) image() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, terrainSide, terrainSide))
	for y := 0; y < terrainSide; y++ {
		for x := 0; x < terrainSide; x++ {
			var sr, sg, sb float64
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					p := c.pix[(2*y+dy)*terrainHi+2*x+dx]
					sr, sg, sb = sr+p.r, sg+p.g, sb+p.b
				}
			}
			out.SetRGBA(x, y, color.RGBA{clamp8(sr / 4), clamp8(sg / 4), clamp8(sb / 4), 255})
		}
	}
	return out
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// terrainPlaceholder は、地形 id の変種 variant（0〜3）の仮の絵（48×48・不透明）を返す。
// 仮の絵がない id は ok が偽。同じ引数なら必ず同じ画像（乱数は id と variant から作る種のもの）。
func terrainPlaceholder(id string, variant int) (img *image.RGBA, ok bool) {
	if !hasTerrainPlaceholder(id) {
		return nil, false
	}
	variant = ((variant % terrainVariants) + terrainVariants) % terrainVariants
	r := newTerrainRand(id, variant)
	c := &canvas{}
	switch id {
	case "plain":
		paintFlagstone(c, r)
	case "rubble":
		paintFlagstone(c, r)
		paintRubble(c, r)
	case "bush":
		paintFlagstone(c, r)
		paintBush(c, r)
	case "wall":
		paintFlagstone(c, r)
		paintWall(c, r)
	case "canal":
		paintCanal(c, r)
	case "facility":
		paintFacility(c, r)
	}
	return c.image(), true
}

// paintFlagstone は砂色の敷石を描く。2 段の石の列で、タイルの縁に目地の半分ずつを置くので、並べても継ぎ目が目立たない。
func paintFlagstone(c *canvas, r *terrainRand) {
	c.fill(groutColor)
	const rows = 2
	rowH := terrainHi / rows
	for row := 0; row < rows; row++ {
		// 縦の目地は、左端（隣のマスの右端と半分ずつ）と、中にもう 1〜2 本。
		cuts := []int{0}
		for _, f := range []float64{r.rng(0.28, 0.42), r.rng(0.62, 0.78)} {
			if r.n(3) > 0 || len(cuts) == 1 {
				cuts = append(cuts, int(f*terrainHi))
			}
		}
		cuts = append(cuts, terrainHi)
		for i := 0; i+1 < len(cuts); i++ {
			x0, x1 := cuts[i], cuts[i+1]
			y0, y1 := row*rowH, (row+1)*rowH
			// 目地は 3 px。縁では 1.5 px ずつになるよう、内側へ 2・1 px 寄せる。
			ix0, ix1, iy0, iy1 := x0+2, x1-1, y0+2, y1-1
			if x0 == 0 {
				ix0 = 2
			}
			if x1 == terrainHi {
				ix1 = terrainHi - 1
			}
			shade := r.rng(0.93, 1.07)
			base := mix(sandBase, sandLight, r.f()*0.5).scale(shade)
			c.rect(ix0, iy0, ix1, iy1, base, 1)
			// 上と左を少し明るく、下と右を少し暗くして、石の厚みを出す。
			c.rect(ix0, iy0, ix1, iy0+2, sandLight, 0.45)
			c.rect(ix0, iy0, ix0+2, iy1, sandLight, 0.3)
			c.rect(ix0, iy1-2, ix1, iy1, sandDark, 0.4)
			c.rect(ix1-2, iy0, ix1, iy1, sandDark, 0.3)
			// 小さな欠けとしみ。
			for k := r.n(3); k > 0; k-- {
				cx := r.rng(float64(ix0)+4, float64(ix1)-4)
				cy := r.rng(float64(iy0)+4, float64(iy1)-4)
				c.ellipse(cx, cy, r.rng(2, 5), r.rng(1.5, 3.5), sandDark, r.rng(0.15, 0.35))
			}
			if r.n(4) == 0 {
				// 欠け: 角を目地の色で削る。
				cs := r.rng(3, 6)
				c.line(float64(ix1)-cs-2, float64(iy1), float64(ix1), float64(iy1)-cs-2, 3, groutColor, 0.9)
			}
		}
	}
	c.grain(r, 0.035)
}

// shadowEllipse は、地面に落ちる影（やや右下へずらした暗い楕円）を描く。
func shadowEllipse(c *canvas, cx, cy, rx, ry, a float64) {
	c.ellipse(cx+3, cy+4, rx, ry*0.8, rgb{40, 36, 30}, a)
}

// paintRubble は瓦礫（灰褐色の割れた石の塊）を描く。
func paintRubble(c *canvas, r *terrainRand) {
	n := 4 + r.n(3)
	for i := 0; i < n; i++ {
		cx := r.rng(14, terrainHi-14)
		cy := r.rng(20, terrainHi-14)
		size := r.rng(7, 15)
		shadowEllipse(c, cx, cy+size*0.45, size*1.05, size*0.55, 0.45)
		// 角ばった塊: 多角形の代わりに、傾けた楕円を重ねる。
		shade := r.rng(0.85, 1.12)
		base := rubbleGray.scale(shade)
		c.ellipse(cx, cy, size, size*0.78, base, 1)
		c.ellipse(cx-size*0.15, cy-size*0.2, size*0.7, size*0.5, mix(base, sandLight, 0.35), 0.8)
		c.ellipse(cx+size*0.25, cy+size*0.25, size*0.55, size*0.4, base.scale(0.7), 0.6)
		// 割れ目。
		x0, y0 := cx-size*0.6, cy-size*0.2+r.rng(-3, 3)
		c.line(x0, y0, cx+size*0.5, cy+size*0.35, 1.6, groutColor, 0.75)
	}
	// 細かい砂利。
	for i := 0; i < 14; i++ {
		x, y := r.rng(4, terrainHi-4), r.rng(4, terrainHi-4)
		c.disc(x, y, r.rng(1.2, 2.6), rubbleGray.scale(r.rng(0.8, 1.15)), 0.9)
	}
}

// paintBush は植え込み（低彩度の緑の丸い茂み）を描く。
func paintBush(c *canvas, r *terrainRand) {
	cx := terrainHi/2 + r.rng(-5, 5)
	cy := terrainHi/2 + r.rng(0, 6)
	shadowEllipse(c, cx, cy+22, 36, 14, 0.5)
	n := 8 + r.n(3)
	type blob struct{ x, y, rad float64 }
	var blobs []blob
	for i := 0; i < n; i++ {
		a := r.rng(0, 2*math.Pi)
		d := r.rng(0, 17)
		blobs = append(blobs, blob{cx + math.Cos(a)*d*1.35, cy + math.Sin(a)*d*0.9, r.rng(12, 19)})
	}
	// 下の円ほど暗く、上の円ほど明るい。
	for pass := 0; pass < 3; pass++ {
		for _, b := range blobs {
			shade := 0.62 + 0.2*float64(pass)
			col := mix(bushGreen, sandBase, 0.05*float64(pass)).scale(shade * r.rng(0.94, 1.06))
			off := float64(pass) * -2.5
			rad := b.rad * (1 - 0.14*float64(pass))
			c.disc(b.x, b.y+off, rad, col, 1)
		}
	}
	// 葉の細かい明るい点と、琥珀ではない淡い実。
	for i := 0; i < 26; i++ {
		a := r.rng(0, 2*math.Pi)
		d := r.rng(0, 26)
		c.disc(cx+math.Cos(a)*d*1.25, cy-4+math.Sin(a)*d*0.8, r.rng(1.2, 2.4), mix(bushGreen, sandLight, 0.5), 0.55)
	}
}

// paintWall は崩れた壁（太い石ブロック、ぎざぎざに欠けた上辺、暗い側面）を描く。
func paintWall(c *canvas, r *terrainRand) {
	const bottom = 86
	// 上辺の高さ（列ごと）。段差を付けて、ぎざぎざに欠けさせる。
	top := make([]int, terrainHi)
	h := 16 + r.n(8)
	for x := 0; x < terrainHi; {
		w := 6 + r.n(10)
		step := r.n(9) - 4
		if r.n(5) == 0 {
			step -= 6
		}
		h = max(10, min(34, h+step))
		for i := 0; i < w && x < terrainHi; i++ {
			top[x] = h
			x++
		}
	}
	// 足元の影。
	c.rect(0, bottom, terrainHi, min(terrainHi, bottom+8), rgb{30, 26, 22}, 0.5)
	c.rect(0, bottom+8, terrainHi, terrainHi, rgb{30, 26, 22}, 0.18)
	for x := 0; x < terrainHi; x++ {
		for y := top[x]; y < bottom; y++ {
			c.blend(x, y, mix(sandDark, sandBase, 0.5), 1)
		}
	}
	// 石ブロックの列。上面（明るい帯）、前面、目地。
	rowH := (bottom - 10) / 3
	for row := 0; row < 3; row++ {
		y0 := 10 + row*rowH
		x := -r.n(20)
		for x < terrainHi {
			w := 26 + r.n(24)
			shade := r.rng(0.9, 1.08)
			for xx := max(x, 0); xx < min(x+w, terrainHi); xx++ {
				for y := max(y0, top[xx]); y < min(y0+rowH, bottom); y++ {
					c.blend(xx, y, mix(sandDark, sandBase, 0.55).scale(shade), 1)
				}
			}
			// 目地（縦）。
			for y := max(y0, 0); y < min(y0+rowH, bottom); y++ {
				c.blend(x, y, groutColor, 0.85)
				c.blend(x+1, y, groutColor, 0.5)
			}
			x += w
		}
		// 目地（横）。
		for xx := 0; xx < terrainHi; xx++ {
			if y0 >= top[xx] {
				c.blend(xx, y0, groutColor, 0.85)
				c.blend(xx, y0+1, groutColor, 0.45)
			}
		}
	}
	// 上面の明るい帯（厚みの上の面）と、縁の暗い線。
	for x := 0; x < terrainHi; x++ {
		c.rect(x, top[x], x+1, top[x]+9, sandLight, 0.85)
		c.blend(x, top[x], sandDark, 0.7)
		if x > 0 && top[x] != top[x-1] {
			// 段差の側面。
			lo, hi := min(top[x], top[x-1]), max(top[x], top[x-1])
			c.rect(x, lo, x+1, hi, sandDark, 0.6)
		}
	}
	// 下ほど暗く、左右の端を暗くして、前に出た壁に見せる。
	for y := 0; y < bottom; y++ {
		for x := 0; x < terrainHi; x++ {
			if y >= top[x] {
				f := float64(y-top[x]) / float64(bottom-top[x]+1)
				c.blend(x, y, rgb{40, 36, 30}, 0.08+0.38*f)
			}
		}
	}
	for i := 0; i < 4; i++ {
		x := r.rng(6, terrainHi-6)
		y := r.rng(float64(top[int(x)])+12, bottom-6)
		c.line(x, y, x+r.rng(-6, 6), y+r.rng(6, 14), 1.4, groutColor, 0.6)
	}
	c.line(0, bottom, terrainHi, bottom, 2.5, rgb{30, 26, 22}, 0.7)
	c.grain(r, 0.025)
}

// paintCanal は水路（青緑の水、明るい波の線、石の縁取り）を描く。通れない見た目にするため、縁を高く見せる。
func paintCanal(c *canvas, r *terrainRand) {
	c.fill(sandBase)
	const rim = 8
	// 水: 上が深く、下が少し明るい。
	for y := rim; y < terrainHi-rim; y++ {
		t := float64(y-rim) / float64(terrainHi-2*rim)
		col := mix(waterDeep, waterMid, t*0.8+0.1)
		c.rect(rim, y, terrainHi-rim, y+1, col, 1)
	}
	// 波の線。
	for i := 0; i < 5; i++ {
		y := r.rng(rim+6, terrainHi-rim-6)
		x := r.rng(rim+2, terrainHi-rim-30)
		length := r.rng(14, 34)
		amp := r.rng(1.2, 2.4)
		phase := r.rng(0, 2*math.Pi)
		px, py := x, y
		for s := 1.0; s <= length; s++ {
			nx := x + s
			ny := y + amp*math.Sin(phase+s*0.45)
			c.line(px, py, nx, ny, 2.2, waterLight, 0.65)
			px, py = nx, ny
		}
	}
	for i := 0; i < 10; i++ {
		c.disc(r.rng(rim+3, terrainHi-rim-3), r.rng(rim+3, terrainHi-rim-3), r.rng(1, 2), waterLight, 0.5)
	}
	// 縁の内側の影（石の縁が水の上に落とす）。
	c.rect(rim, rim, terrainHi-rim, rim+5, rgb{10, 40, 48}, 0.4)
	c.rect(rim, rim, rim+4, terrainHi-rim, rgb{10, 40, 48}, 0.25)
	// 石の縁取り（四辺）。外側の目地、石の面、内側の暗い線。
	c.rect(0, 0, terrainHi, rim, sandBase, 1)
	c.rect(0, terrainHi-rim, terrainHi, terrainHi, sandBase, 1)
	c.rect(0, 0, rim, terrainHi, sandBase, 1)
	c.rect(terrainHi-rim, 0, terrainHi, terrainHi, sandBase, 1)
	for _, v := range [][4]int{{0, 0, terrainHi, rim}, {0, terrainHi - rim, terrainHi, terrainHi}, {0, 0, rim, terrainHi}, {terrainHi - rim, 0, terrainHi, terrainHi}} {
		c.rect(v[0], v[1], v[2], v[3], sandLight, 0.2+0.2*r.f())
	}
	c.rect(0, 0, terrainHi, 2, groutColor, 0.8)
	c.rect(0, terrainHi-2, terrainHi, terrainHi, groutColor, 0.8)
	c.rect(0, 0, 2, terrainHi, groutColor, 0.8)
	c.rect(terrainHi-2, 0, terrainHi, terrainHi, groutColor, 0.8)
	// 縁の石の継ぎ目。
	for i := 0; i < 3; i++ {
		x := int(r.rng(10, terrainHi-10))
		c.rect(x, 0, x+2, rim, groutColor, 0.7)
		x = int(r.rng(10, terrainHi-10))
		c.rect(x, terrainHi-rim, x+2, terrainHi, groutColor, 0.7)
	}
	c.rect(rim-1, rim-1, terrainHi-rim+1, rim, groutColor, 0.8)
	c.rect(rim-1, terrainHi-rim, terrainHi-rim+1, terrainHi-rim+1, groutColor, 0.8)
	c.rect(rim-1, rim-1, rim, terrainHi-rim+1, groutColor, 0.8)
	c.rect(terrainHi-rim, rim-1, terrainHi-rim+1, terrainHi-rim+1, groutColor, 0.8)
	c.grain(r, 0.02)
}

// paintFacility は浄水施設（石の台の上に、青緑に光る円環と、線と円の記号、中心の淡い光）を描く。
func paintFacility(c *canvas, r *terrainRand) {
	paintFlagstone(c, r)
	// 台: 面取りした石の台。
	const m = 8
	c.rect(m, m, terrainHi-m, terrainHi-m, mix(sandBase, sandLight, 0.35), 1)
	c.rect(m, m, terrainHi-m, m+3, sandLight, 0.7)
	c.rect(m, m, m+3, terrainHi-m, sandLight, 0.5)
	c.rect(m, terrainHi-m-3, terrainHi-m, terrainHi-m, sandDark, 0.6)
	c.rect(terrainHi-m-3, m, terrainHi-m, terrainHi-m, sandDark, 0.5)
	c.rect(m-2, terrainHi-m, terrainHi-m+3, terrainHi-m+4, rgb{30, 26, 22}, 0.4)
	cx, cy := float64(terrainHi)/2, float64(terrainHi)/2
	// 中心の淡い光（外へ向けて薄くなる）。
	for rad := 34.0; rad > 0; rad -= 1 {
		t := 1 - rad/34
		c.disc(cx, cy, rad, glowTeal, 0.035+0.04*t*t)
	}
	// 光る円環（外側に淡い光の輪）。
	c.ring(cx, cy, 29, 9, glowTeal, 0.13)
	c.ring(cx, cy, 29, 4.5, mix(waterMid, glowTeal, 0.5), 0.95)
	c.ring(cx, cy, 29, 1.8, rgb{225, 255, 245}, 0.9)
	c.ring(cx, cy, 17, 2.6, mix(waterMid, glowTeal, 0.4), 0.85)
	// 線と円の記号（文字に見えない幾何学）。三角形と、頂点の小さな円。変種ごとに向きを変える。
	rot := float64(r.n(4)) * math.Pi / 2
	var pts [3][2]float64
	for i := range pts {
		a := rot - math.Pi/2 + float64(i)*2*math.Pi/3
		pts[i] = [2]float64{cx + math.Cos(a)*17, cy + math.Sin(a)*17}
	}
	for i := range pts {
		j := (i + 1) % 3
		c.line(pts[i][0], pts[i][1], pts[j][0], pts[j][1], 2, glowTeal, 0.85)
	}
	c.line(cx, cy, pts[0][0], pts[0][1], 1.6, glowTeal, 0.7)
	for i, p := range pts {
		col := glowTeal
		if i == 0 {
			col = amberColor
		}
		c.disc(p[0], p[1], 3.2, col, 0.95)
	}
	c.disc(cx, cy, 4.2, rgb{232, 255, 250}, 0.9)
	// 円環の外に刻んだ短い線（目盛り）。
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		c.line(cx+math.Cos(a)*34, cy+math.Sin(a)*34, cx+math.Cos(a)*38, cy+math.Sin(a)*38, 1.6, sandDark, 0.7)
	}
	c.grain(r, 0.015)
}

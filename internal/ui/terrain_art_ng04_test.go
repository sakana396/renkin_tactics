package ui

import (
	"bytes"
	"image"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/internal/defs"
)

// avgColor は画像の平均色（0〜255）を返す。
func avgColor(img *image.RGBA) (r, g, b float64) {
	n := float64(img.Bounds().Dx() * img.Bounds().Dy())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.RGBAAt(x, y)
			r, g, b = r+float64(c.R), g+float64(c.G), b+float64(c.B)
		}
	}
	return r / n, g / n, b / n
}

func colorDist(a, b [3]float64) float64 {
	d := 0.0
	for i := range a {
		if x := a[i] - b[i]; x < 0 {
			d -= x
		} else {
			d += x
		}
	}
	return d
}

func TestTerrainPlaceholderNG04(t *testing.T) {
	t.Run("TC-DES-NG-10-18 仮の地形は 48×48・不透明で、同じ引数なら同じバイト列", func(t *testing.T) {
		for _, id := range terrainPlaceholderIDs {
			for v := 0; v < terrainVariants; v++ {
				a, ok := terrainPlaceholder(id, v)
				if !ok {
					t.Fatalf("%s の仮の絵がない", id)
				}
				if a.Bounds() != image.Rect(0, 0, 48, 48) {
					t.Fatalf("%s/%d の大きさ %v", id, v, a.Bounds())
				}
				for i := 3; i < len(a.Pix); i += 4 {
					if a.Pix[i] != 255 {
						t.Fatalf("%s/%d の画素 %d が不透明でない", id, v, i/4)
					}
				}
				b, _ := terrainPlaceholder(id, v)
				if !bytes.Equal(a.Pix, b.Pix) {
					t.Fatalf("%s/%d が毎回同じでない", id, v)
				}
			}
		}
	})
	t.Run("TC-DES-NG-10-19 変種は 4 枚とも違う（範囲外の番号は 4 で丸める）", func(t *testing.T) {
		for _, id := range terrainPlaceholderIDs {
			var imgs [terrainVariants]*image.RGBA
			for v := range imgs {
				imgs[v], _ = terrainPlaceholder(id, v)
			}
			for i := 0; i < terrainVariants; i++ {
				for j := i + 1; j < terrainVariants; j++ {
					if bytes.Equal(imgs[i].Pix, imgs[j].Pix) {
						t.Errorf("%s の変種 %d と %d が同じ", id, i, j)
					}
				}
			}
			if w, _ := terrainPlaceholder(id, terrainVariants+1); !bytes.Equal(w.Pix, imgs[1].Pix) {
				t.Errorf("%s の変種 5 は変種 1 と同じのはず", id)
			}
		}
	})
	t.Run("TC-DES-NG-10-20 6 つの地形は互いに違う（平均色が離れている）", func(t *testing.T) {
		avg := map[string][3]float64{}
		for _, id := range terrainPlaceholderIDs {
			img, _ := terrainPlaceholder(id, 0)
			r, g, b := avgColor(img)
			avg[id] = [3]float64{r, g, b}
		}
		for i, a := range terrainPlaceholderIDs {
			for _, b := range terrainPlaceholderIDs[i+1:] {
				if d := colorDist(avg[a], avg[b]); d < 6 {
					t.Errorf("%s と %s の平均色が近すぎる: %v %v（差 %.1f）", a, b, avg[a], avg[b], d)
				}
			}
		}
	})
	t.Run("TC-DES-NG-10-21 石畳は砂色（R > G > B で中程度の明るさ）、水路は青緑寄り（B と G が R より大きい）、植え込みは緑寄り", func(t *testing.T) {
		for v := 0; v < terrainVariants; v++ {
			plain, _ := terrainPlaceholder("plain", v)
			r, g, b := avgColor(plain)
			if !(r > g && g > b) || r < 120 || r > 200 || r-b < 15 || r-b > 70 {
				t.Errorf("石畳 %d の平均色が砂色でない: %.0f %.0f %.0f", v, r, g, b)
			}
			canal, _ := terrainPlaceholder("canal", v)
			r, g, b = avgColor(canal)
			if !(b > r && g > r) {
				t.Errorf("水路 %d が青緑寄りでない: %.0f %.0f %.0f", v, r, g, b)
			}
			facility, _ := terrainPlaceholder("facility", v)
			if _, g, _ := avgColor(facility); g < 100 {
				t.Errorf("浄水施設 %d が暗すぎる", v)
			}
		}
		bush, _ := terrainPlaceholder("bush", 0)
		plain, _ := terrainPlaceholder("plain", 0)
		_, bg, _ := avgColor(bush)
		_, pg, _ := avgColor(plain)
		br, _, _ := avgColor(bush)
		pr, _, _ := avgColor(plain)
		if !(pr-pg > br-bg) {
			t.Error("植え込みは石畳より緑寄りのはず")
		}
	})
	t.Run("TC-DES-NG-10-22 変種はマスの座標だけで決まり（同じマスはいつも同じ）、4 種とも出る", func(t *testing.T) {
		seen := map[int]int{}
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				v := terrainVariantIndex(x, y, terrainVariants)
				if v < 0 || v >= terrainVariants || v != terrainVariantIndex(x, y, terrainVariants) {
					t.Fatalf("(%d,%d) の変種 %d", x, y, v)
				}
				seen[v]++
			}
		}
		for v := 0; v < terrainVariants; v++ {
			if seen[v] < 16 {
				t.Errorf("変種 %d が少なすぎる: %v", v, seen)
			}
		}
		// 隣り合う横のマスが同じ変種ばかりにならない。
		same := 0
		for x := 0; x < 15; x++ {
			if terrainVariantIndex(x, 3, 4) == terrainVariantIndex(x+1, 3, 4) {
				same++
			}
		}
		if same > 8 {
			t.Errorf("隣が同じ変種の組が多すぎる: %d", same)
		}
		if terrainVariantIndex(5, 5, 1) != 0 || terrainVariantIndex(5, 5, 0) != 0 {
			t.Error("n が 1 以下なら 0")
		}
	})
}

// terrainCatalog は新作の目録（地形 6 つ）を返す。
func terrainCatalog(t *testing.T) *defs.Catalog { return alchemyCatalog(t) }

func TestTerrainStoreNG04(t *testing.T) {
	// 地形の名前と ID（data/alchemy/terrains.toml）。
	names := map[string]string{"石畳": "plain", "瓦礫": "rubble", "植え込み": "bush", "崩れた壁": "wall", "水路": "canal", "浄水施設": "facility"}
	t.Run("TC-DES-NG-10-23 名前は目録から ID を引く。仮の絵がない名前（平地・森・知らない名前）と画像なしの store は nil（今の経路）", func(t *testing.T) {
		a := NewArtStore(nil, nil, terrainCatalog(t))
		for name, id := range names {
			if got, ok := a.TerrainID(name); !ok || got != id {
				t.Errorf("%s の ID: %q %v", name, got, ok)
			}
			if a.TerrainImage(name, 3, 4) == nil {
				t.Errorf("%s の仮の絵がない", name)
			}
		}
		for _, name := range []string{"平地", "森", "山", "砦", "壁", "謎の地形", ""} {
			if a.TerrainImage(name, 0, 0) != nil {
				t.Errorf("%q は今の経路のはず", name)
			}
		}
		var none *ArtStore
		if none.TerrainImage("水路", 0, 0) != nil {
			t.Error("nil の store は nil を返す")
		}
		if NewArtStore(nil, nil, nil).TerrainImage("水路", 0, 0) != nil {
			t.Error("目録がなければ名前を引けない")
		}
		if _, ok := terrainPlaceholder("ice", 0); ok {
			t.Error("知らない ID に仮の絵はない")
		}
		// HiRes でない Game は、仮の絵を使わない。
		if (&Game{scenes: &SceneController{cfg: Config{HiRes: false}}}).hiRes() || (&Game{}).hiRes() {
			t.Error("HiRes でないときは偽")
		}
		if !(&Game{scenes: &SceneController{cfg: Config{HiRes: true}}}).hiRes() {
			t.Error("HiRes のときは真")
		}
	})
	t.Run("TC-DES-NG-10-24 仮の絵の変種は座標で選ばれ、同じマスはいつも同じ絵", func(t *testing.T) {
		a := NewArtStore(nil, nil, terrainCatalog(t))
		distinct := map[any]bool{}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				img := a.TerrainImage("石畳", x, y)
				if img != a.TerrainImage("石畳", x, y) {
					t.Fatal("同じマスで絵が変わった")
				}
				distinct[img] = true
			}
		}
		if len(distinct) != terrainVariants {
			t.Errorf("使われた絵 %d 枚（4 枚のはず）", len(distinct))
		}
	})
	good := func(w, h int) []byte { return pngBytes(t, w, h, 255) }
	t.Run("TC-DES-NG-10-25 art/terrain の PNG があれば仮の絵より優先され、_2〜_4 は変種としてあるものだけを座標で選ぶ", func(t *testing.T) {
		media := fstest.MapFS{
			"art/terrain/canal.png":   {Data: good(192, 192)},
			"art/terrain/canal_3.png": {Data: good(192, 192)},
		}
		a := NewArtStore(media, nil, terrainCatalog(t))
		p1, p3 := a.Ebiten("art/terrain/canal.png", true), a.Ebiten("art/terrain/canal_3.png", true)
		if p1 == nil || p3 == nil {
			t.Fatal("PNG を読めない")
		}
		got := map[any]bool{}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				img := a.TerrainImage("水路", x, y)
				if img != p1 && img != p3 {
					t.Fatalf("(%d,%d) が PNG でない絵", x, y)
				}
				got[img] = true
			}
		}
		if len(got) != 2 {
			t.Errorf("2 枚とも使われるはず: %d", len(got))
		}
		// PNG のない地形は仮の絵。
		if a.TerrainImage("瓦礫", 0, 0) == nil {
			t.Error("PNG のない地形は仮の絵")
		}
		// 変種だけ（_2 だけ）でも使える。
		b := NewArtStore(fstest.MapFS{"art/terrain/bush_2.png": {Data: good(64, 64)}}, nil, terrainCatalog(t))
		if b.TerrainImage("植え込み", 1, 1) != b.Ebiten("art/terrain/bush_2.png", true) {
			t.Error("_2 だけの PNG が使われない")
		}
		if got := terrainArtPaths("canal"); len(got) != 4 || got[0] != "art/terrain/canal.png" || got[1] != "art/terrain/canal_2.png" || got[3] != "art/terrain/canal_4.png" {
			t.Errorf("道筋 %v", got)
		}
	})
	t.Run("TC-DES-NG-10-26 壊れた PNG・大きすぎる PNG は「なし」で、誤りを 1 回だけ stderr に出し、仮の絵に戻る。目録にない名前の PNG は使わない", func(t *testing.T) {
		var stderr bytes.Buffer
		media := fstest.MapFS{
			"art/terrain/canal.png":    {Data: []byte("not a png")},
			"art/terrain/rubble.png":   {Data: good(artMaxSide+1, 1)},
			"art/terrain/ice.png":      {Data: good(64, 64)},
			"art/terrain/facility.png": {Data: good(200, 100)},
		}
		a := NewArtStore(media, &stderr, terrainCatalog(t))
		for i := 0; i < 3; i++ {
			if a.TerrainImage("水路", i, 0) == nil || a.TerrainImage("瓦礫", i, 0) == nil {
				t.Fatal("壊れた PNG のとき、仮の絵に戻るはず")
			}
		}
		if a.Ebiten("art/terrain/canal.png", true) != nil || a.Ebiten("art/terrain/rubble.png", true) != nil {
			t.Error("壊れた・大きすぎる PNG は「なし」のはず")
		}
		out := stderr.String()
		if bytes.Count(stderr.Bytes(), []byte("art/terrain/canal.png")) != 1 || bytes.Count(stderr.Bytes(), []byte("画像が大きすぎます（art/terrain/rubble.png")) != 1 {
			t.Errorf("誤りは 1 回ずつのはず: %q", out)
		}
		// 目録にない地形の PNG は引けない。
		if a.TerrainImage("氷", 0, 0) != nil {
			t.Error("目録にない名前は nil")
		}
		// 正方形でない PNG も使える（縦横比を保たず、マスに合わせて描く）。
		if a.TerrainImage("浄水施設", 0, 0) != a.Ebiten("art/terrain/facility.png", true) {
			t.Error("非正方形の PNG も使われるはず")
		}
	})
}

func TestMarkScaleNG04(t *testing.T) {
	t.Run("TC-DES-NG-10-28 印・棒は CellSize/32 に比例し、32 では今までと同じ。どの大きさでもマスの中で重ならない", func(t *testing.T) {
		for _, tc := range []struct {
			cell                                 int
			equip, barThick, barRight, zocY, cmd int
		}{
			{32, 10, 4, 21, 24, 12},
			{48, 15, 6, 32, 36, 18},
			{64, 20, 8, 42, 48, 24},
		} {
			useTestMetrics(t, tc.cell)
			pt := image.Pt(96, 160)
			cell := image.Rect(pt.X, pt.Y, pt.X+CellSize, pt.Y+CellSize)
			equip, bar, cmd := equipMarkRect(pt), unitHPBarRect(pt), CommandMarkRect(pt)
			if equip.Dx() != tc.equip || bar.Dy() != tc.barThick || bar.Max.X != pt.X+tc.barRight || zocEnemyCornerY != tc.zocY || cmd.Dx() != tc.cmd {
				t.Errorf("%d: 印 %d 棒 %v 角印 y=%d 指揮 %d", tc.cell, equip.Dx(), bar, zocEnemyCornerY, cmd.Dx())
			}
			if tc.cell == BaseCellSize && (bar.Min.X != pt.X+2 || bar.Max.Y != pt.Y+31) {
				t.Errorf("32 の棒が今までと違う: %v", bar)
			}
			if !equip.In(cell) || !bar.In(cell) || !cmd.In(cell) || equip.Overlaps(bar) {
				t.Errorf("%d: はみ出し・重なり 印 %v 棒 %v 指揮 %v", tc.cell, equip, bar, cmd)
			}
			for _, boss := range []bool{false, true} {
				head, stem := pinShape(pt, boss)
				if !head.In(cell) || !stem.In(cell) || head.Overlaps(equip) || stem.Overlaps(equip) {
					t.Errorf("%d: ピン boss=%v %v %v", tc.cell, boss, head, stem)
				}
				if !boss && head.Overlaps(cmd) {
					t.Errorf("%d: ピンと指揮の印が重なる", tc.cell)
				}
			}
			// 敵軍の角印（HP の棒の上）が棒と重ならない。
			var arm = scaled(zocMarkArm)
			if pt.Y+zocEnemyCornerY >= bar.Min.Y || zocEnemyCornerY-arm < 0 {
				t.Errorf("%d: 敵軍の角印が棒に重なる", tc.cell)
			}
		}
	})
	t.Run("TC-DES-NG-10-28 既定（32）では scaled は恒等で、ZOC の角印の寸法は変わらない", func(t *testing.T) {
		if scaled(zocMarkInset) != 5 || scaled(zocMarkArm) != 5 || scaled(2) != 2 {
			t.Fatal("32 では scaled は恒等")
		}
	})
}

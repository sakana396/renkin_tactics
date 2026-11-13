package ui

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
)

// 縮小の関数（DES-NG-08）。
func TestShrinkImageNG02(t *testing.T) {
	// 左半分が不透明の赤、右半分が透明（色は緑のまま）の 8×8。
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x < 4 {
				src.SetNRGBA(x, y, color.NRGBA{R: 200, A: 255})
			} else {
				src.SetNRGBA(x, y, color.NRGBA{G: 255, A: 0})
			}
		}
	}
	t.Run("TC-DES-NG-08-95 指定の大きさになり、透過が保たれ、透明の色がにじまない", func(t *testing.T) {
		out := shrinkImage(src, src.Bounds(), 4, 2)
		if out.Bounds() != image.Rect(0, 0, 4, 2) {
			t.Fatalf("大きさ %v", out.Bounds())
		}
		if c := out.NRGBAAt(0, 0); c != (color.NRGBA{R: 200, A: 255}) {
			t.Errorf("不透明の側 %v", c)
		}
		if c := out.NRGBAAt(3, 1); c.A != 0 {
			t.Errorf("透明の側 %v", c)
		}
		// 境目をまたぐ画素（4×4 → 3×3 の中央列）は半透明の赤。緑が混ざらない。
		mid := shrinkImage(src, src.Bounds(), 3, 3).NRGBAAt(1, 1)
		if mid.A < 100 || mid.A > 155 || mid.G != 0 || mid.R != 200 {
			t.Errorf("境目 %v（半透明の赤のはず）", mid)
		}
		if shrinkImage(src, src.Bounds(), 0, 4) != nil || shrinkImage(src, image.Rectangle{}, 4, 4) != nil {
			t.Error("大きさ 0・空の範囲は nil のはず")
		}
	})
	t.Run("TC-DES-NG-08-96 同じ入力は同じ結果。範囲の切り出しと、顔の切り出し", func(t *testing.T) {
		a, b := shrinkImage(src, src.Bounds(), 3, 5), shrinkImage(src, src.Bounds(), 3, 5)
		if !bytes.Equal(a.Pix, b.Pix) {
			t.Error("結果が毎回違う")
		}
		// 切り出し: 右半分だけなら全部透明。
		if c := shrinkImage(src, image.Rect(4, 0, 8, 8), 2, 2); c.NRGBAAt(0, 0).A != 0 {
			t.Errorf("右半分 %v", c.NRGBAAt(0, 0))
		}
		// 人物の絵（20..60 × 10..110 が不透明）の顔の切り出し: 正方形で、中身の上端から始まり、中身の中に収まる。
		art := image.NewNRGBA(image.Rect(0, 0, 128, 128))
		for y := 10; y < 110; y++ {
			for x := 20; x < 60; x++ {
				art.SetNRGBA(x, y, color.NRGBA{B: 255, A: 255})
			}
		}
		if bb := artBounds(art); bb != image.Rect(20, 10, 60, 110) {
			t.Errorf("中身 %v", bb)
		}
		r := faceCrop(art)
		if r.Dx() != r.Dy() || r.Min.Y != 10 || !r.In(image.Rect(20, 10, 60, 110)) || r.Dx() != 40 {
			t.Errorf("顔 %v（上端 10 から幅 40 の正方形のはず）", r)
		}
		// 透明だけの絵は全体を返す（0 除算・空の範囲にならない）。
		if r := faceCrop(image.NewNRGBA(image.Rect(0, 0, 8, 8))); r.Empty() {
			t.Error("透明の絵で空の範囲")
		}
	})
}

// ArtStore の縮小キャッシュ（DES-NG-08）。
func TestArtStoreShrunkNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-97 同じ引数では 1 回だけ作り、壊れた画像・ない画像・大きさ 0 は「なし」", func(t *testing.T) {
		var stderr bytes.Buffer
		media := fstest.MapFS{
			"art/units/aoi.png": {Data: pngBytes(t, 128, 128, 255)},
			"art/units/bad.png": {Data: []byte("not a png")},
		}
		a := NewArtStore(media, &stderr, nil)
		e1 := a.Shrunk("art/units/aoi.png", true, 24, 24)
		e2 := a.Shrunk("art/units/aoi.png", true, 24, 24)
		if e1 == nil || e1 != e2 {
			t.Fatalf("同じ引数で同じ画像のはず: %p %p", e1, e2)
		}
		if b := e1.Bounds(); b.Dx() != 24 || b.Dy() != 24 {
			t.Errorf("大きさ %v", b)
		}
		// 引数が違えば別に作る。
		if a.Shrunk("art/units/aoi.png", false, 24, 24) == e1 || a.Shrunk("art/units/aoi.png", true, 16, 16) == e1 {
			t.Error("顔か大きさが違うのに同じ画像")
		}
		if len(a.shrunk) != 3 {
			t.Errorf("覚えた数 %d、3 のはず", len(a.shrunk))
		}
		if a.Shrunk("art/units/bad.png", true, 24, 24) != nil || a.Shrunk("art/units/bad.png", true, 24, 24) != nil {
			t.Error("壊れた画像は nil のはず")
		}
		if n := strings.Count(stderr.String(), "画像を読めません（art/units/bad.png）"); n != 1 {
			t.Errorf("誤りの知らせが %d 回、1 回のはず: %q", n, stderr.String())
		}
		if a.Shrunk("art/units/none.png", true, 24, 24) != nil || a.Shrunk("art/units/aoi.png", true, 0, 24) != nil {
			t.Error("ない画像・大きさ 0 は nil のはず")
		}
		var nilStore *ArtStore
		if nilStore.Shrunk("x", true, 8, 8) != nil {
			t.Error("nil の ArtStore は nil のはず")
		}
	})
	t.Run("TC-DES-NG-08-98 上限を超える画像は縮小も「なし」", func(t *testing.T) {
		var stderr bytes.Buffer
		a := NewArtStore(fstest.MapFS{"art/units/big.png": {Data: pngBytes(t, artMaxSide+1, 1, 255)}}, &stderr, nil)
		if a.Shrunk("art/units/big.png", false, 8, 8) != nil || !strings.Contains(stderr.String(), "大きすぎます") {
			t.Errorf("上限超えは nil と知らせのはず: %q", stderr.String())
		}
	})
}

// 準備の名簿の行と加入の場面の絵の値（DES-NG-08）。
func TestSceneIconArtNG02(t *testing.T) {
	cat := alchemyCatalog(t)
	newScenes := func(media fstest.MapFS, hires bool, chapter string) *SceneController {
		cfg := Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }, Stderr: &bytes.Buffer{}, HiRes: hires, StartChapter: chapter}
		if media != nil {
			cfg.Media = media
		}
		return NewSceneController(cfg)
	}
	toPrepare := func(t *testing.T, s *SceneController) {
		t.Helper()
		s.Choose(0)
		s.Choose(0)
		for i := 0; s.Scene() == Story || s.Scene() == Joined; i++ {
			if i > 200 {
				t.Fatal("物語が終わらない")
			}
			s.Advance()
		}
		if s.Scene() != Prepare {
			t.Fatalf("場面 %v", s.Scene())
		}
	}
	t.Run("TC-DES-NG-08-99 名簿の行の絵に、そのキャラの絵の値が入る。media がなければ空（今のタイル）", func(t *testing.T) {
		s := newScenes(fstest.MapFS{}, false, "ch02")
		toPrepare(t, s)
		v := s.View()
		if len(v.Icons) != len(v.Rows) || len(v.Icons) < 5 {
			t.Fatalf("絵 %d、行 %d", len(v.Icons), len(v.Rows))
		}
		names := map[string]bool{}
		for i, ic := range v.Icons {
			row := v.Rows[i].Rect
			if ic.Art.Empty() || !ic.Art.Player || ic.Art.Name == "" || ic.Art.ClassName == "" || !ic.Face || ic.Size != prepareRosterArtSize {
				t.Errorf("行 %d の絵 %+v", i, ic)
			}
			names[ic.Art.Name] = true
			// 絵の箱は行の高さに収まり、行の左端と文字（x = 96）の間にある。
			if ic.Y < row.Min.Y || ic.Y+ic.Size > row.Max.Y || ic.X < row.Min.X || ic.X+ic.Size > 96 {
				t.Errorf("行 %d の絵の箱が行 %v からはみ出す: %+v", i, row, ic)
			}
			// 同じ行（y + 4）の文に、絵のキャラの名前がある。
			found := false
			for _, tx := range v.Texts {
				if tx.Y == row.Min.Y+4 && tx.X == 96 && strings.HasPrefix(tx.Text, ic.Art.Name+"　"+ic.Art.ClassName) {
					found = true
				}
			}
			if !found {
				t.Errorf("行 %d の文に %s・%s がない", i, ic.Art.Name, ic.Art.ClassName)
			}
		}
		if !names["アオイ"] || !names["ハル"] {
			t.Errorf("名前 %v", names)
		}
		// 行の位置・大きさは変えていない（24 px ごと）。
		for i := 1; i < len(v.Rows); i++ {
			if v.Rows[i].Rect.Min.Y-v.Rows[i-1].Rect.Min.Y != 24 || v.Icons[i].Y-v.Icons[i-1].Y != 24 {
				t.Errorf("行 %d の間隔", i)
			}
		}

		// media が nil: 絵の値は空で、タイルだけ。
		n := newScenes(nil, false, "ch02")
		toPrepare(t, n)
		for i, ic := range n.View().Icons {
			if !ic.Art.Empty() {
				t.Errorf("media なしの行 %d に絵の値 %+v", i, ic)
			}
		}
	})
	t.Run("TC-DES-NG-08-100 加入の場面の絵に、加入するキャラの絵の値が入る。media がなければ空", func(t *testing.T) {
		toJoined := func(s *SceneController) SceneView {
			t.Helper()
			s.Choose(0)
			s.Choose(0)
			for i := 0; s.Scene() != Joined; i++ {
				if i > 200 || s.Scene() == Prepare {
					t.Fatal("加入の場面にならない", s.Scene())
				}
				s.Advance()
			}
			return s.View()
		}
		v := toJoined(newScenes(fstest.MapFS{}, false, "ch02"))
		if len(v.Icons) != 1 {
			t.Fatalf("絵 %+v", v.Icons)
		}
		ic := v.Icons[0]
		if ic.Art != (ArtKey{Name: "ハル", ClassName: "調合の見習い", Player: true}) || ic.Size != joinedArtSize || ic.Face {
			t.Errorf("加入の絵 %+v", ic)
		}
		// 絵の箱は下の文（y = 256）に触れず、窓に収まる。
		if ic.X != 48 || ic.Y != 208 || ic.Y+ic.Size > 256 || ic.X+ic.Size > ScreenWidth {
			t.Errorf("加入の絵の位置 %+v", ic)
		}
		n := toJoined(newScenes(nil, false, "ch02"))
		if len(n.Icons) != 1 || !n.Icons[0].Art.Empty() {
			t.Errorf("media なしの加入の絵 %+v", n.Icons)
		}
	})
	t.Run("TC-DES-NG-08-101 HiRes の準備の絵は、832×576 の層の中に収まり、窓の中央に置く", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := newScenes(fstest.MapFS{}, true, "ch02")
		toPrepare(t, s)
		v := s.View()
		layer := image.Rect(0, 0, sceneCanvasWidth, sceneCanvasHeight)
		for i, ic := range v.Icons {
			if !(image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size)).In(layer) || ic.Art.Empty() {
				t.Errorf("行 %d の絵 %+v が層 %v に収まらない", i, ic, layer)
			}
		}
		off := sceneOffset()
		if off != image.Pt(128, 128) || !layer.Add(off).In(windowRect()) {
			t.Errorf("層の位置 %v", off)
		}
	})
}

// 描画: 絵があれば縮めた画像を作って覚え、描いて、タイルは描かない（DES-NG-08）。
func TestDrawSceneIconNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-102 絵があれば縮めた画像を 1 つだけ作って描く（顔は 24 px、全身は縦横比を保つ）", func(t *testing.T) {
		cat := alchemyCatalog(t)
		media := fstest.MapFS{"art/units/aoi.png": {Data: pngBytes(t, 128, 128, 255)}}
		store := NewArtStore(media, &bytes.Buffer{}, cat)
		g := &Game{scenes: &SceneController{art: store}}
		dst := ebiten.NewImage(64, 64)
		key := ArtKey{Name: "アオイ", ClassName: "錬金術師", Player: true}
		g.drawSceneIcon(dst, SceneIcon{X: 4, Y: 4, Art: key, Size: 24, Face: true})
		g.drawSceneIcon(dst, SceneIcon{X: 4, Y: 4, Art: key, Size: 24, Face: true})
		if len(store.shrunk) != 1 {
			t.Fatalf("縮めた画像 %d 個、1 個のはず", len(store.shrunk))
		}
		g.drawSceneIcon(dst, SceneIcon{X: 4, Y: 4, Art: key, Size: 48})
		if len(store.shrunk) != 2 {
			t.Fatalf("全身を足して %d 個、2 個のはず", len(store.shrunk))
		}
		for k := range store.shrunk {
			if k.w != k.h || (k.face && k.w != 24) || (!k.face && k.w != 48) {
				t.Errorf("鍵 %+v", k)
			}
		}
	})
}

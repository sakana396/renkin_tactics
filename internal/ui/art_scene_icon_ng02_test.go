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
	t.Run("TC-DES-NG-08-95 同じ入力は同じ結果。範囲の切り出しと、顔の切り出し", func(t *testing.T) {
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
	t.Run("TC-DES-NG-08-96 同じ引数では 1 回だけ作り、壊れた画像・ない画像・大きさ 0 は「なし」", func(t *testing.T) {
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
	t.Run("TC-DES-NG-08-96 上限を超える画像は縮小も「なし」", func(t *testing.T) {
		var stderr bytes.Buffer
		a := NewArtStore(fstest.MapFS{"art/units/big.png": {Data: pngBytes(t, artMaxSide+1, 1, 255)}}, &stderr, nil)
		if a.Shrunk("art/units/big.png", false, 8, 8) != nil || !strings.Contains(stderr.String(), "大きすぎます") {
			t.Errorf("上限超えは nil と知らせのはず: %q", stderr.String())
		}
	})
}

// 名簿と加入場面で画像の指定を組み立てる（DES-NG-08）。
func TestSceneIconViewsNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-97 仮の一章の名簿", func(t *testing.T) {
		for _, present := range []bool{false, true} {
			cfg := Config{Catalog: alchemyCatalog(t), Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }}
			if present {
				cfg.Media = fstest.MapFS{}
			}
			s := NewSceneController(cfg)
			startScenes(s)
			s.Cancel()
			v := s.View()
			if s.Scene() != Prepare || len(v.Rows) != 4 || len(v.Icons) != 4 {
				t.Fatalf("名簿: %v %d %d", s.Scene(), len(v.Rows), len(v.Icons))
			}
			for i, ic := range v.Icons {
				row := v.Rows[i].Rect
				if ic.Art.Empty() == present || ic.Size != 24 || !ic.Face || ic.X != 60 || ic.Y != row.Min.Y || ic.Y+ic.Size > row.Max.Y {
					t.Errorf("行 %d: %+v", i, ic)
				}
				if present && (!ic.Art.Player || ic.Art.Name == "" || ic.Art.ClassName == "") {
					t.Errorf("画像の手がかり: %+v", ic.Art)
				}
				if i > 0 && row.Min.Y-v.Rows[i-1].Rect.Min.Y != 24 {
					t.Error("行間が違う")
				}
			}
		}
	})
	t.Run("TC-DES-NG-08-98 加入のある試験用の章", func(t *testing.T) {
		for _, present := range []bool{false, true} {
			cfg := Config{Catalog: sceneCatalog(true, true), Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }}
			if present {
				cfg.Media = fstest.MapFS{}
			}
			s := NewSceneController(cfg)
			startScenes(s)
			s.Cancel()
			winScene(t, s)
			playOut(s.battle)
			s.Advance()
			s.Advance()
			s.Advance()
			s.Cancel()
			if s.Scene() != Joined {
				t.Fatalf("加入場面: %v", s.Scene())
			}
			v := s.View()
			if len(v.Icons) != 1 {
				t.Fatalf("絵: %+v", v.Icons)
			}
			ic := v.Icons[0]
			if ic.X != 48 || ic.Y != 208 || ic.Size != 48 || ic.Face || ic.Art.Empty() == present {
				t.Errorf("絵: %+v", ic)
			}
			if present && ic.Art != (ArtKey{Name: "加入者", ClassName: "ロード", Player: true}) {
				t.Errorf("手がかり: %+v", ic.Art)
			}
		}
	})
}

// 描画: 絵があれば縮めた画像を作って覚え、描いて、タイルは描かない（DES-NG-08）。
func TestDrawSceneIconNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-99 絵があれば縮めた画像を 1 つだけ作って描く（顔は 24 px、全身は縦横比を保つ）", func(t *testing.T) {
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

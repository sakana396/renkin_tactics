package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
	"testing/fstest"
)

// REV-109 の指摘への対応で足したテスト（DES-NG-10）。

func TestHiResGameOver(t *testing.T) {
	t.Run("TC-DES-NG-10-29 敗北の場面は配置（832×576）が窓の中央に収まり、巻き戻しの入口も層の中", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := hiresScenes(t, Battle)
		s.scene = GameOver
		s.battle.state = Over
		s.battle.gameOver = true
		if !onCanvas(GameOver) || !canvasInUse(GameOver, false) || canvasInUse(GameOver, true) {
			t.Fatal("敗北は一覧を閉じている間だけ中央寄せの層を使う")
		}
		off := sceneOffset()
		inner := image.Rect(0, 0, sceneCanvasWidth, sceneCanvasHeight)
		v := s.View()
		if bad := sceneViewBounds(v, inner); len(bad) > 0 {
			t.Errorf("敗北の表示が配置の枠からはみ出す: %v", bad)
		}
		entry := s.battle.RewindView()
		if len(entry.Buttons) != 1 || !strings.HasPrefix(entry.Buttons[0].Label, "巻き戻し") {
			t.Fatal("巻き戻しの入口がない:", entry.Buttons)
		}
		if bad := sceneViewBounds(entry, inner); len(bad) > 0 {
			t.Errorf("巻き戻しの入口が配置の枠からはみ出す: %v", bad)
		}
		if bad := sceneViewBounds(entry.translated(off), windowRect()); len(bad) > 0 {
			t.Errorf("巻き戻しの入口が窓からはみ出す: %v", bad)
		}
		// 入口は、敗北のボタン（巻き戻す など）と重ならない。
		for _, b := range v.Buttons {
			if b.Rect.Overlaps(entry.Buttons[0].Rect) {
				t.Errorf("入口がボタン「%s」と重なる", b.Label)
			}
		}
		// 層の座標の入口をクリックすると、一覧が開く（窓の座標では off を引いて渡る）。
		r := entry.Buttons[0].Rect
		wx, wy := r.Min.X+4+off.X, r.Min.Y+4+off.Y
		x, y := sceneCursor(wx, wy, GameOver, false)
		s.Click(x, y)
		if !s.battle.rewind.open {
			t.Fatal("入口のクリックで一覧が開かない")
		}
	})
	t.Run("TC-DES-NG-10-29 既定（32）では敗北の入口は今までの位置（y = MapSize）", func(t *testing.T) {
		useTestMetrics(t, BaseCellSize)
		s := hiresScenes(t, Battle)
		s.scene = GameOver
		s.battle.state = Over
		s.battle.gameOver = true
		b := s.battle.RewindView().Buttons
		if len(b) != 1 || b[0].Rect != image.Rect(304, 512, 504, 530) {
			t.Fatal(b)
		}
	})
}

func TestSceneCursor(t *testing.T) {
	tests := []struct {
		name       string
		cell       int
		scene      Scene
		rewindOpen bool
		x, y       int
		wantX      int
		wantY      int
	}{
		{"TC-DES-NG-10-30 HiRes のタイトルは (128,128) を引く", HiResCellSize, Title, false, 300, 400, 172, 272},
		{"TC-DES-NG-10-30 HiRes の準備は引く", HiResCellSize, Prepare, false, 128, 128, 0, 0},
		{"TC-DES-NG-10-30 HiRes の加入・勝利の結果・クリアも引く", HiResCellSize, Clear, false, 500, 500, 372, 372},
		{"TC-DES-NG-10-30 HiRes の敗北も引く", HiResCellSize, GameOver, false, 400, 300, 272, 172},
		{"TC-DES-NG-10-30 HiRes の敗北でも、巻き戻しの一覧を開いている間は引かない", HiResCellSize, GameOver, true, 400, 300, 400, 300},
		{"TC-DES-NG-10-30 HiRes の戦闘は引かない", HiResCellSize, Battle, false, 400, 300, 400, 300},
		{"TC-DES-NG-10-30 HiRes の配置は引かない", HiResCellSize, Deploy, false, 400, 300, 400, 300},
		{"TC-DES-NG-10-30 既定（32）はどの場面でも引かない（タイトル）", BaseCellSize, Title, false, 400, 300, 400, 300},
		{"TC-DES-NG-10-30 既定（32）の敗北も引かない", BaseCellSize, GameOver, false, 400, 300, 400, 300},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useTestMetrics(t, tc.cell)
			x, y := sceneCursor(tc.x, tc.y, tc.scene, tc.rewindOpen)
			if x != tc.wantX || y != tc.wantY {
				t.Fatalf("(%d,%d) = (%d,%d)、期待 (%d,%d)", tc.x, tc.y, x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestFitWindowSize(t *testing.T) {
	tests := []struct {
		name         string
		monW, monH   int
		wantW, wantH int
	}{
		{"TC-DES-NG-10-31 1920×1080 は収まる（そのまま）", 1920, 1080, 1088, 832},
		{"TC-DES-NG-10-31 2560×1440 は収まる（拡大しない）", 2560, 1440, 1088, 832},
		{"TC-DES-NG-10-31 1366×768 は高さに合わせて縮める（縦横比を保つ）", 1366, 768, 903, 691},
		{"TC-DES-NG-10-31 1280×720（1080p の 150%）は縮める", 1280, 720, 847, 648},
		{"TC-DES-NG-10-31 幅が足りない縦長の画面は幅に合わせて縮める", 800, 1200, 720, 550},
		{"TC-DES-NG-10-31 取れない（0×0）ときは論理の大きさのまま", 0, 0, 1088, 832},
		{"TC-DES-NG-10-31 取れない（負）ときも論理の大きさのまま", -1, 1080, 1088, 832},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, h := fitWindowSize(tc.monW, tc.monH, 1088, 832)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("(%d,%d) = %d×%d、期待 %d×%d", tc.monW, tc.monH, w, h, tc.wantW, tc.wantH)
			}
			// 縮めたときは、画面の 9 割に収まり、縦横比は論理の画面とほぼ同じ。
			if tc.monW > 0 && tc.monH > 0 && (w > tc.monW*9/10 || h > tc.monH*9/10) && (w != 1088 || h != 832) {
				t.Errorf("画面に収まらない: %d×%d", w, h)
			}
		})
	}
	t.Run("TC-DES-NG-10-31 既定の表示（832×576）の論理の大きさも同じ規則", func(t *testing.T) {
		if w, h := fitWindowSize(1920, 1080, 832, 576); w != 832 || h != 576 {
			t.Fatal(w, h)
		}
	})
}

func TestCrackLineWidth(t *testing.T) {
	t.Run("TC-DES-NG-10-32 崩れの割れ目の太さは描く箱の大きさに比例する（右欄の印は 48 でも変わらない）", func(t *testing.T) {
		pt := image.Pt(0, 0)
		useTestMetrics(t, BaseCellSize)
		if w := crackLineWidth(CommandMarkRect(pt)); w != 1.5 {
			t.Errorf("32 の盤面の印 = %v、期待 1.5", w)
		}
		right := image.Rect(10, 10, 10+commandMarkSize, 10+commandMarkSize)
		if w := crackLineWidth(right); w != 1.5 {
			t.Errorf("32 の右欄の印 = %v、期待 1.5", w)
		}
		useTestMetrics(t, HiResCellSize)
		// 盤面の印の箱は 18 px になり、太さは 2.25（今までの 1.5 × 48 / 32 と同じ）。
		if r := CommandMarkRect(pt); r.Dx() != 18 {
			t.Fatal(r)
		}
		if w := crackLineWidth(CommandMarkRect(pt)); w != 2.25 {
			t.Errorf("48 の盤面の印 = %v、期待 2.25", w)
		}
		if w := crackLineWidth(right); w != 1.5 {
			t.Errorf("48 の右欄の印 = %v、期待 1.5（変えない）", w)
		}
	})
}

func TestTerrainStretchNote(t *testing.T) {
	t.Run("TC-DES-NG-10-33 正方形でない地形の PNG の知らせは、縦横比を保たず引き伸ばすと言う。ユニットの絵は縦横比を保つと言う", func(t *testing.T) {
		var stderr bytes.Buffer
		media := fstest.MapFS{
			"art/terrain/facility.png": {Data: pngBytes(t, 200, 100, 255)},
			"art/units/aoi.png":        {Data: pngBytes(t, 200, 100, 255)},
		}
		a := NewArtStore(media, &stderr, terrainCatalog(t))
		if a.TerrainImage("浄水施設", 0, 0) == nil {
			t.Fatal("非正方形の PNG も使われるはず")
		}
		a.Ebiten("art/units/aoi.png", true)
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("知らせは 2 行のはず: %q", stderr.String())
		}
		if !strings.Contains(lines[0], "art/terrain/facility.png") || !strings.Contains(lines[0], "縦横比を保たず、マスいっぱいに引き伸ばして描きます") || strings.Contains(lines[0], "縦横比を保って") {
			t.Errorf("地形の知らせ: %q", lines[0])
		}
		if !strings.Contains(lines[1], "art/units/aoi.png") || !strings.Contains(lines[1], "縦横比を保って描きます") {
			t.Errorf("ユニットの知らせ: %q", lines[1])
		}
	})
}

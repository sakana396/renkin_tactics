package ui

import (
	"image"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// useTestMetrics は、テストの間だけ 1 マスの大きさを cell に切り替え、終わったら必ず既定に戻す（DES-NG-10）。
// マスの大きさはパッケージの変数なので、これを使うテストは t.Parallel を使わない。
func useTestMetrics(t *testing.T, cell int) {
	t.Helper()
	useMetrics(cell)
	t.Cleanup(func() { useMetrics(BaseCellSize) })
}

// windowRect は今の窓の全体。
func windowRect() image.Rectangle { return image.Rect(0, 0, ScreenWidth, ScreenHeight) }

// hiresScenes は、新作の定義データで、指定した場面まで進めた場面の進行を返す（shot.go の prepareScene と同じ手順）。
func hiresScenes(t *testing.T, stop Scene) *SceneController {
	t.Helper()
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	s := NewSceneController(Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }, HiRes: true})
	if stop == Title {
		return s
	}
	s.Choose(0)
	if stop == RuleSelect {
		return s
	}
	s.Choose(0)
	if stop == Story {
		return s
	}
	for i := 0; s.Scene() == Story || s.Scene() == Joined; i++ {
		if i >= 200 {
			t.Fatal("物語が終わらない")
		}
		s.Advance()
	}
	if stop == Prepare {
		return s
	}
	s.Advance()
	if stop == Deploy {
		return s
	}
	s.Choose(0)
	return s
}

// sceneViewBounds は、場面の表示のうち、文・ボタン・行・絵・タブが bound に収まっているかを調べ、はみ出したものを返す。
func sceneViewBounds(v SceneView, bound image.Rectangle) []string {
	var out []string
	in := func(what string, r image.Rectangle) {
		if !r.In(bound) {
			out = append(out, what+" "+r.String())
		}
	}
	for _, tx := range v.Texts {
		w := textWidth(tx.Text)
		h := lineHeight
		if tx.Big {
			w, h = 2*w, 2*h
		}
		in("文「"+tx.Text+"」", image.Rect(tx.X, tx.Y, tx.X+w, tx.Y+h))
	}
	for _, b := range v.Buttons {
		in("ボタン「"+b.Label+"」", b.Rect)
	}
	for _, r := range v.Rows {
		in("行", r.Rect)
	}
	for _, ic := range v.Icons {
		in("絵", image.Rect(ic.X, ic.Y, ic.X+2*16, ic.Y+2*16))
	}
	for _, tab := range v.Tabs {
		in("タブ", tab.Rect)
	}
	if v.Portrait != nil {
		in("肖像", v.Portrait.Rect)
	}
	return out
}

// infoViewBounds は、情報欄の表示のうち bound からはみ出したものを返す。
func infoViewBounds(v InfoView, bound image.Rectangle) []string {
	var out []string
	for _, tx := range v.Texts {
		w, h := textWidth(tx.Text), lineHeight
		if tx.Big {
			w, h = 2*w, 2*h
		}
		if r := image.Rect(tx.X, tx.Y, tx.X+w, tx.Y+h); !r.In(bound) {
			out = append(out, "文「"+tx.Text+"」 "+r.String())
		}
	}
	for _, ic := range v.Icons {
		if r := image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size); !r.In(bound) {
			out = append(out, "絵 "+r.String())
		}
	}
	for _, b := range v.Bars {
		if r := image.Rect(b.X, b.Y, b.X+max(b.Fill, b.Frame), b.Y+6); !r.In(bound) {
			out = append(out, "棒 "+r.String())
		}
	}
	return out
}

// 1 マス 48 px の高解像度の表示（DES-NG-10）。
func TestHiResMetrics(t *testing.T) {
	t.Run("TC-DES-NG-10-01 既定は 32 px・盤面 512・窓 832×576、配置の値も今までのまま", func(t *testing.T) {
		useTestMetrics(t, BaseCellSize)
		if CellSize != 32 || MapSize != 512 || ScreenWidth != 832 || ScreenHeight != 576 || sceneOffset() != (image.Point{}) {
			t.Fatal(CellSize, MapSize, ScreenWidth, ScreenHeight, sceneOffset())
		}
		// 期待値は、切り替えを入れる前の定数（右欄 x = 512〜831、案内欄 y = 512〜575 の配置）から手で置いた。
		for name, c := range map[string][2]any{
			"予測窓の候補": {forecastCandidates, [4]image.Rectangle{image.Rect(0, 8, 416, 248), image.Rect(96, 8, 512, 248), image.Rect(0, 264, 416, 504), image.Rect(96, 264, 512, 504)}},
			"右欄の窓":   {forecastFallbackRect, image.Rect(512, 0, 832, 360)},
			"詳細ボタン":  {detailButtonRect, image.Rect(672, 304, 744, 336)},
			"配置の詳細":  {deployDetailButtonRect, image.Rect(752, 336, 824, 356)},
			"全体図":    {overviewButtonRect, image.Rect(744, 472, 824, 504)},
			"次の味方":   {nextAllyButtonRect, image.Rect(704, 520, 824, 544)},
			"範囲の状態":  {rangeStatusRect, image.Rect(520, 304, 664, 336)},
			"消す":     {rangeClearRect, image.Rect(752, 304, 824, 336)},
			"ミニマップ":  {MinimapRect(), image.Rect(512, 360, 832, 512)},
		} {
			if c[0] != c[1] {
				t.Errorf("%s = %v、want %v", name, c[0], c[1])
			}
		}
		for name, c := range map[string][2]int{
			"summaryX": {summaryX, 520}, "summaryPortraitX": {summaryPortraitX, 832 - infoMargin - PortraitSize},
			"detailBottom": {detailBottom, 512 - infoMargin}, "rewindLeft": {rewindLeft, 520}, "rewindRight": {rewindRight, 824},
			"cameraGuideY": {cameraGuideY, 560}, "hintRewindY": {hintRewindY, 560}, "minimapBoxX": {minimapBoxX, 608},
			"equipMarkOffset": {equipMarkOffset, 22}, "unitHPBarRight": {unitHPBarRight, 21}, "zocEnemyCornerY": {zocEnemyCornerY, 24},
		} {
			if c[0] != c[1] {
				t.Errorf("%s = %d、want %d", name, c[0], c[1])
			}
		}
	})
	t.Run("TC-DES-NG-10-02 / TC-DES-NG-10-03 設定で 48 px に切り替わり、盤面 768・窓 1088×832、設定を戻すと既定に戻る", func(t *testing.T) {
		for _, tc := range []struct {
			name                       string
			hires                      bool
			cell, board, width, height int
		}{
			{"HiRes", true, 48, 768, 1088, 832},
			{"既定", false, 32, 512, 832, 576},
		} {
			useTestMetrics(t, BaseCellSize)
			useConfigMetrics(Config{HiRes: tc.hires})
			if CellSize != tc.cell || MapSize != tc.board || ScreenWidth != tc.width || ScreenHeight != tc.height {
				t.Fatal(tc.name, CellSize, MapSize, ScreenWidth, ScreenHeight)
			}
			if ScreenWidth != MapSize+InfoWidth || ScreenHeight != MapSize+GuideHeight {
				t.Fatal(tc.name, "窓は盤面＋右欄・案内欄")
			}
		}
		// 48 のあとに false を渡すと既定へ戻す（Run を続けて呼んでも状態が残らない）。
		useTestMetrics(t, HiResCellSize)
		useConfigMetrics(Config{})
		if CellSize != 32 {
			t.Fatal(CellSize)
		}
	})
}

func TestHiResBoard(t *testing.T) {
	t.Run("TC-DES-NG-10-04 盤面の座標変換: 右端・下端のマスと盤面の外", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		cam := Camera{Width: 16, Height: 16}
		for _, tc := range []struct {
			x, y int
			want tactics.Pos
			ok   bool
		}{
			{0, 0, tactics.Pos{}, true},
			{47, 47, tactics.Pos{}, true},
			{48, 0, tactics.Pos{X: 1}, true},
			{767, 767, tactics.Pos{X: 15, Y: 15}, true},
			{720, 720, tactics.Pos{X: 15, Y: 15}, true},
			{768, 0, tactics.Pos{}, false},
			{0, 768, tactics.Pos{}, false},
			{-1, 0, tactics.Pos{}, false},
		} {
			p, ok := cam.CellAt(tc.x, tc.y)
			if ok != tc.ok || (ok && p != tc.want) {
				t.Errorf("CellAt(%d, %d) = %v, %v", tc.x, tc.y, p, ok)
			}
			if p2, ok2 := CellAt(tc.x, tc.y); p2 != p || ok2 != ok {
				t.Errorf("CellAt の入口が Camera と食い違う (%d, %d)", tc.x, tc.y)
			}
		}
		// 大きな盤面でも、カメラの表示範囲は 16×16 マスで、Screen は盤面の中に収まる。
		big := Camera{X: 5, Y: 7, Width: 32, Height: 32}
		if got := big.Screen(tactics.Pos{X: 20, Y: 22}); got != image.Pt(720, 720) {
			t.Fatal(got)
		}
		if !big.Visible(tactics.Pos{X: 20, Y: 22}) || big.Visible(tactics.Pos{X: 21, Y: 22}) {
			t.Fatal("表示範囲は 16 マス")
		}
		board := image.Rect(0, 0, MapSize, MapSize)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				s := cam.Screen(tactics.Pos{X: x, Y: y})
				if !image.Rect(s.X, s.Y, s.X+CellSize, s.Y+CellSize).In(board) {
					t.Fatal("マスが盤面の外", x, y, s)
				}
			}
		}
	})
	t.Run("TC-DES-NG-10-05 行動の一覧は盤面の中（右端・下端は内側へ寄せる）、ポップアップも盤面の中", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		board := image.Rect(0, 0, MapSize, MapSize)
		for _, dest := range []tactics.Pos{{}, {X: 15}, {Y: 15}, {X: 15, Y: 15}, {X: 7, Y: 7}, {X: 14, Y: 13}} {
			for n := 1; n <= 5; n++ {
				rects := MenuRects(dest, n)
				if len(rects) != n {
					t.Fatal(dest, n, len(rects))
				}
				for i, r := range rects {
					if !r.In(board) {
						t.Errorf("一覧 %v n=%d [%d] = %v が盤面の外", dest, n, i, r)
					}
					if i > 0 && r.Min.Y != rects[i-1].Max.Y {
						t.Errorf("一覧が縦に続かない %v", rects)
					}
				}
				// 移動先のマスと重ならない。
				cell := image.Rect(dest.X*CellSize, dest.Y*CellSize, (dest.X+1)*CellSize, (dest.Y+1)*CellSize)
				for _, r := range rects {
					if r.Overlaps(cell) {
						t.Errorf("一覧 %v が移動先 %v に重なる", r, cell)
					}
				}
			}
		}
		// 期待値: 右端に出られないので左へ。(15,15) の左上は (720,720)。2 個で 44 px なので下端 764 に収まり、上へはずらさない。
		if got := MenuRects(tactics.Pos{X: 15, Y: 15}, 2); got[0] != image.Rect(552, 720, 720, 742) {
			t.Fatal(got)
		}
		// 5 個（110 px）は、下端から 768 − 110 = 658 へずらす。
		if got := MenuRects(tactics.Pos{X: 3, Y: 15}, 5); got[0] != image.Rect(192, 658, 360, 680) {
			t.Fatal(got)
		}
		cam := Camera{Width: 16, Height: 16}
		for _, p := range []tactics.Pos{{}, {X: 15}, {Y: 15}, {X: 15, Y: 15}} {
			for _, w := range []int{40, 200, 400} {
				pt, ok := cam.Popup(p, w)
				if !ok {
					t.Fatal(p)
				}
				if pt.X < 0 || pt.X+w > MapSize || pt.Y < 0 || pt.Y+lineHeight > MapSize {
					t.Errorf("ポップアップ %v w=%d → %v が盤面の外", p, w, pt)
				}
			}
		}
	})
	t.Run("TC-DES-NG-10-06 予測窓の候補は盤面の四隅に収まり、盤面の外の窓は右欄", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		board := image.Rect(0, 0, MapSize, MapSize)
		want := [4]image.Rectangle{image.Rect(0, 8, 416, 248), image.Rect(352, 8, 768, 248), image.Rect(0, 520, 416, 760), image.Rect(352, 520, 768, 760)}
		if forecastCandidates != want {
			t.Fatal(forecastCandidates)
		}
		for _, c := range forecastCandidates {
			if !c.In(board) {
				t.Fatal(c)
			}
		}
		if forecastFallbackRect != image.Rect(768, 0, 1088, 360) {
			t.Fatal(forecastFallbackRect)
		}
		cam := Camera{Width: 16, Height: 16}
		// 相手が上半分なら下の候補、下半分なら上の候補。
		if r, fb := ForecastRect(cam, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 2, Y: 1}); fb || r != forecastCandidates[2] {
			t.Fatal(r, fb)
		}
		if r, fb := ForecastRect(cam, tactics.Pos{X: 1, Y: 14}, tactics.Pos{X: 2, Y: 14}); fb || r != forecastCandidates[0] {
			t.Fatal(r, fb)
		}
	})
	t.Run("TC-DES-NG-10-07 右欄・案内欄のボタンは窓に収まり、盤面と重ならない（案内欄は盤面の下）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		board := image.Rect(0, 0, MapSize, MapSize)
		right := image.Rect(MapSize, 0, ScreenWidth, ScreenHeight)
		for name, r := range map[string]image.Rectangle{
			"詳細": detailButtonRect, "配置の詳細": deployDetailButtonRect, "全体図": overviewButtonRect,
			"範囲の状態": rangeStatusRect, "消す": rangeClearRect, "次の味方": nextAllyButtonRect,
		} {
			if !r.In(windowRect()) || r.Overlaps(board) {
				t.Errorf("%s %v が窓の外か盤面に重なる", name, r)
			}
			if name != "次の味方" && !r.In(right) {
				t.Errorf("%s %v が右欄の外", name, r)
			}
		}
		// 次の味方は盤面の下の案内欄（y = 768〜831）の右側。
		if !nextAllyButtonRect.In(image.Rect(MapSize, MapSize, ScreenWidth, ScreenHeight)) {
			t.Fatal(nextAllyButtonRect)
		}
		// 右欄の上段（情報）の操作行とミニマップの欄は縦に重ならない。
		info := image.Rect(MapSize, 0, ScreenWidth, InfoHeight)
		mm := MinimapRect()
		if mm != image.Rect(768, 360, 1088, 768) || mm.Overlaps(info) || !mm.In(right) {
			t.Fatal(mm)
		}
		for name, r := range map[string]image.Rectangle{"詳細": detailButtonRect, "範囲の状態": rangeStatusRect, "消す": rangeClearRect, "配置の詳細": deployDetailButtonRect} {
			if !r.In(info) {
				t.Errorf("%s %v が情報欄の外", name, r)
			}
		}
		if !overviewButtonRect.In(mm) {
			t.Fatal("全体図のボタンがミニマップの欄の外", overviewButtonRect)
		}
	})
	t.Run("TC-DES-NG-10-08 ミニマップの本体と文は欄の中にあり、情報欄と重ならない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		mm := MinimapRect()
		info := image.Rect(MapSize, 0, ScreenWidth, InfoHeight)
		// 本体は欄の左上から今と同じ相対位置（x は欄の中央、y は見出しの下）。
		box := image.Rect(minimapBoxX, minimapBoxY, minimapBoxX+minimapBox, minimapBoxY+minimapBox)
		if box != image.Rect(864, 376, 992, 504) || !box.In(mm) || box.Overlaps(info) {
			t.Fatal(box)
		}
		c := newIdleWith(t, wideSetup)
		v := c.Minimap()
		if !v.Available || !v.Layout.Area.In(box) {
			t.Fatal(v.Layout)
		}
		for _, tx := range v.Texts {
			if r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight); !r.In(mm) {
				t.Errorf("ミニマップの文「%s」%v が欄の外", tx.Text, r)
			}
		}
		// ミニマップのクリックは、欄の中だけが取る。盤面（x < 768）・情報欄（y < 360）は取らない。
		if c.MinimapClick(10, 10) || c.MinimapClick(700, 600) || c.MinimapClick(900, 100) {
			t.Fatal("欄の外のクリックを取った")
		}
	})
	t.Run("TC-DES-NG-10-09 全体図は盤面いっぱいに置き、大きなマップは 24 px・16 マスは 48 px", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		for _, tc := range []struct{ w, h, cell int }{{32, 32, 24}, {16, 16, 48}, {32, 16, 24}, {20, 20, 38}} {
			l := overviewLayout(tc.w, tc.h)
			if l.Cell != tc.cell {
				t.Errorf("%dx%d: マス %d、want %d", tc.w, tc.h, l.Cell, tc.cell)
			}
			if !l.Area.In(image.Rect(0, 0, MapSize, MapSize)) || l.Area.Dx() != tc.w*tc.cell || l.Area.Dy() != tc.h*tc.cell {
				t.Errorf("%dx%d: 本体 %v", tc.w, tc.h, l.Area)
			}
		}
	})
	t.Run("TC-DES-NG-10-10 ユニットの絵は 48 のマスに縦横比を保って収まる", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		for _, tc := range []struct{ w, h int }{{64, 64}, {128, 256}, {256, 128}, {1024, 1024}} {
			cell := image.Rect(96, 144, 96+CellSize, 144+CellSize)
			r := unitArtRect(96, 144, CellSize, tc.w, tc.h)
			if !r.In(cell) || (r.Dx() != CellSize && r.Dy() != CellSize) {
				t.Errorf("%dx%d → %v が %v に収まらない／余っている", tc.w, tc.h, r, cell)
			}
		}
	})
	t.Run("TC-DES-NG-10-11 HP の棒・装備の印・ZOC の角印・ボスの印は 48 のマスの中で重ならない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		pt := image.Pt(96, 144)
		cell := image.Rect(pt.X, pt.Y, pt.X+CellSize, pt.Y+CellSize)
		bar, equip := unitHPBarRect(pt), equipMarkRect(pt)
		if !bar.In(cell) || !equip.In(cell) || bar.Overlaps(equip) {
			t.Fatal(bar, equip)
		}
		if equip != image.Rect(pt.X+38, pt.Y+38, pt.X+48, pt.Y+48) {
			t.Fatal(equip)
		}
		if bar.Max.X != pt.X+37 || bar.Dy() != 4 {
			t.Fatal(bar)
		}
		if zocEnemyCornerY != 40 {
			t.Fatal(zocEnemyCornerY)
		}
		head, stem := pinShape(pt, true)
		if !head.In(cell) || !stem.In(cell) || head.Overlaps(equip) {
			t.Fatal(head, stem)
		}
		cmd := CommandMarkRect(pt)
		if !cmd.In(cell) || cmd.Overlaps(head) {
			t.Fatal(cmd)
		}
		if r := rangeFrameRect(pt, 2); r != image.Rect(pt.X+2, pt.Y+2, pt.X+46, pt.Y+46) {
			t.Fatal(r)
		}
	})
}

func TestHiResScenes(t *testing.T) {
	t.Run("TC-DES-NG-10-12 盤面のない場面は既定の配置（832×576）が窓の中央に収まる", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		if off := sceneOffset(); off != image.Pt(128, 128) {
			t.Fatal(off)
		}
		for _, sc := range []Scene{Title, RuleSelect, Story, Prepare} {
			s := hiresScenes(t, sc)
			if s.Scene() != sc || !onCanvas(sc) {
				t.Fatal(sc, s.Scene())
			}
			off := sceneOffset()
			canvas := image.Rect(off.X, off.Y, off.X+sceneCanvasWidth, off.Y+sceneCanvasHeight)
			if !canvas.In(windowRect()) {
				t.Fatal(canvas)
			}
			// 配置は既定の窓の大きさの中に収まる。窓の中央へ寄せたあとの位置も窓に収まる。
			v := s.View()
			if bad := sceneViewBounds(v, image.Rect(0, 0, sceneCanvasWidth, sceneCanvasHeight)); len(bad) > 0 {
				t.Errorf("場面 %v: 配置の枠からはみ出す: %v", sc, bad)
			}
			if bad := sceneViewBounds(v.translated(off), windowRect()); len(bad) > 0 {
				t.Errorf("場面 %v: 窓からはみ出す: %v", sc, bad)
			}
			if v.Info != nil {
				if bad := infoViewBounds(*v.Info, image.Rect(0, 0, sceneCanvasWidth, sceneCanvasHeight)); len(bad) > 0 {
					t.Errorf("場面 %v: 情報欄が配置の枠からはみ出す: %v", sc, bad)
				}
			}
		}
	})
	t.Run("TC-DES-NG-10-13 物語の文の折り返しは、窓の幅ではなく配置の幅（832）に従う", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		long := make([]rune, 400)
		for i := range long {
			long[i] = 'あ'
		}
		for _, line := range wrapSceneText(string(long)) {
			if textWidth(line) > sceneCanvasWidth-96 {
				t.Fatal(textWidth(line))
			}
		}
		if _, x, w := storyArtLayout(false); x != 48 || w != 736 {
			t.Fatal(x, w)
		}
		p, x, w := storyArtLayout(true)
		if !p.In(image.Rect(0, 0, sceneCanvasWidth, sceneCanvasHeight)) || x+w > sceneCanvasWidth-storyMargin {
			t.Fatal(p, x, w)
		}
	})
	t.Run("TC-DES-NG-10-14 配置の場面は盤面（768）と右欄・案内欄に収まり、ボタンは窓の中", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := hiresScenes(t, Deploy)
		if s.Scene() != Deploy || onCanvas(Deploy) {
			t.Fatal(s.Scene())
		}
		v := s.View()
		if bad := sceneViewBounds(v, windowRect()); len(bad) > 0 {
			t.Fatal(bad)
		}
		right, guide := image.Rect(MapSize, 0, ScreenWidth, MapSize), image.Rect(0, MapSize, ScreenWidth, ScreenHeight)
		for _, b := range v.Buttons {
			if !b.Rect.In(right) {
				t.Errorf("ボタン「%s」%v が右欄の外", b.Label, b.Rect)
			}
		}
		// 文は、右欄か案内欄のどちらかに収まる（盤面の上には出さない）。
		for _, tx := range v.Texts {
			r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
			if !r.In(right) && !r.In(guide) {
				t.Errorf("文「%s」%v が右欄・案内欄の外", tx.Text, r)
			}
		}
		if v.Minimap == nil || !MinimapRect().In(windowRect()) {
			t.Fatal("ミニマップ")
		}
		// 戦闘開始と準備に戻るは重ならない。
		if v.Buttons[0].Rect.Overlaps(v.Buttons[1].Rect) || v.Buttons[0].Rect.Overlaps(v.Buttons[2].Rect) || v.Buttons[1].Rect.Overlaps(v.Buttons[2].Rect) {
			t.Fatal("ボタンが重なる", v.Buttons)
		}
	})
	t.Run("TC-DES-NG-10-15 戦闘の右欄・巻き戻しの一覧・案内は窓に収まり、盤面と重ならない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		c := newIdleWith(t, wideSetup)
		right := image.Rect(MapSize, 0, ScreenWidth, ScreenHeight)
		// 右欄の上下 2 段。
		cursorOnUnit := c.Camera().Screen(c.Units()[0].Pos)
		iv := c.BattleInfoView(cursorOnUnit.X+4, cursorOnUnit.Y+4)
		if bad := infoViewBounds(iv, image.Rect(MapSize, 0, ScreenWidth, InfoHeight)); len(bad) > 0 {
			t.Errorf("戦闘の右欄が情報欄からはみ出す: %v", bad)
		}
		// 巻き戻し: 一覧は右欄、「戻る」「決定」は右欄の下、入口は盤面の下の案内欄。
		rv := c.RewindView()
		if bad := sceneViewBounds(rv, windowRect()); len(bad) > 0 {
			t.Errorf("巻き戻しの入口が窓の外: %v", bad)
		}
		c.OpenRewind()
		rv = c.RewindView()
		if bad := sceneViewBounds(rv, right); len(bad) > 0 {
			t.Errorf("巻き戻しの一覧が右欄の外: %v", bad)
		}
		for _, b := range rv.Buttons {
			if b.Rect.Overlaps(MinimapRect()) && b.Label != "" {
				t.Errorf("巻き戻しのボタン「%s」%v がミニマップに重なる", b.Label, b.Rect)
			}
		}
		if hintRewindY+lineHeight != ScreenHeight || cameraGuideY+lineHeight != ScreenHeight {
			t.Fatal(hintRewindY, cameraGuideY)
		}
	})
	t.Run("TC-DES-NG-10-16 詳しい画面は窓の幅の中で左右に中央寄せされる", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		c := newIdleWith(t, wideSetup)
		u := c.Units()[0]
		dx := sceneOffset().X
		if dx != 128 {
			t.Fatal(dx)
		}
		v := DetailView(u, SectionStats).translate(dx, 0)
		if bad := infoViewBounds(v, image.Rect(0, 0, ScreenWidth, MapSize)); len(bad) > 0 {
			t.Fatal(bad)
		}
		if detailBottom != MapSize-infoMargin {
			t.Fatal(detailBottom)
		}
	})
}

// translated は場面の表示の位置をすべて off だけずらした写しを返す（高解像度の表示で窓の中央へ置いた位置を確かめる）。
func (v SceneView) translated(off image.Point) SceneView {
	out := v
	out.Texts = nil
	for _, tx := range v.Texts {
		tx.X, tx.Y = tx.X+off.X, tx.Y+off.Y
		out.Texts = append(out.Texts, tx)
	}
	out.Buttons = nil
	for _, b := range v.Buttons {
		b.Rect = b.Rect.Add(off)
		out.Buttons = append(out.Buttons, b)
	}
	out.Rows = nil
	for _, r := range v.Rows {
		r.Rect, r.Mark = r.Rect.Add(off), r.Mark.Add(off)
		out.Rows = append(out.Rows, r)
	}
	out.Icons = nil
	for _, ic := range v.Icons {
		ic.X, ic.Y = ic.X+off.X, ic.Y+off.Y
		out.Icons = append(out.Icons, ic)
	}
	out.Tabs = nil
	for _, tab := range v.Tabs {
		tab.Rect = tab.Rect.Add(off)
		out.Tabs = append(out.Tabs, tab)
	}
	return out
}

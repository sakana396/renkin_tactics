package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// minimapPt は、w × h のマップのミニマップで、マス p の左上から (dx, dy) の画面の座標を返す。
// 期待値は DES-IT06 12.5 節の配置（本体は x = 608〜735、y = 376〜503 の 128×128、1 マスは 128 ÷ 長い辺の整数）から手で求める。
func minimapPt(cell, ox, oy int, p tactics.Pos, dx, dy int) (int, int) {
	return ox + p.X*cell + dx, oy + p.Y*cell + dy
}

// minimapSetup は 32 × 32 の盤面で、四隅と中ほどに地形を置き、wideSetup と同じユニットを並べる（DES-UI-28）。
//
//	地形: (0, 0) 森、(31, 0) 山、(0, 31) 砦、(31, 31) 壁、(5, 5) 名前の分からない通れない地形（壁として描く）
//	0: レオン (20, 20)。主人公
//	1: ガルド (1, 1)
//	2: ボス (31, 31)。待ち構える
//	3: ファイター (21, 20)。待ち構える
func minimapSetup() tactics.Setup {
	s := wideSetup()
	s.Terrain[0][0] = forest
	s.Terrain[0][31] = tactics.Terrain{Name: "山", Cost: 3}
	s.Terrain[31][0] = tactics.Terrain{Name: "砦", Cost: 1, HealPercent: 20}
	s.Terrain[31][31] = tactics.Terrain{Name: "壁"}
	s.Terrain[5][5] = tactics.Terrain{Name: "謎", Cost: 0}
	s.Units[2].Pos = tactics.Pos{X: 30, Y: 30}
	return s
}

// unitRects は、ミニマップのユニットを陣営ごとの範囲にする。
func unitRects(v MinimapView) map[image.Rectangle]tactics.Side {
	out := map[image.Rectangle]tactics.Side{}
	for _, u := range v.Units {
		out[u.Rect] = u.Side
	}
	return out
}

// ミニマップの配置・変換・描くものの元・クリック（DES-UI-28、DES-IT06 12.5 節）を確かめる。
func TestMinimap(t *testing.T) {
	t.Run("TC-FR-MAP-06-05 本体の大きさと位置は長い辺で決め、長方形は中央に寄せる", func(t *testing.T) {
		box := image.Rect(608, 376, 736, 504)
		for _, tc := range []struct {
			w, h, cell int
			area       image.Rectangle
		}{
			// 32×32 は 1 マス 4 px、16×16 は 8 px で、本体がちょうど 128×128。
			{32, 32, 4, image.Rect(608, 376, 736, 504)},
			{16, 16, 8, image.Rect(608, 376, 736, 504)},
			// 20×16: 128 ÷ 20 = 6 px。幅 120・高さ 96 で、左右に (128 − 120) ÷ 2 = 4、上下に 16 の余白。
			{20, 16, 6, image.Rect(612, 392, 732, 488)},
			// 32×18: 4 px。高さ 72 で、上下に 28 の余白。
			{32, 18, 4, image.Rect(608, 404, 736, 476)},
			// 縦長の 18×32 も同じ。
			{18, 32, 4, image.Rect(636, 376, 708, 504)},
			// 5×3: 25 px。幅 125・高さ 75 で、左に 1、上に 26 の余白。
			{5, 3, 25, image.Rect(609, 402, 734, 477)},
		} {
			l := minimapLayout(tc.w, tc.h)
			if l.Cell != tc.cell || l.Area != tc.area {
				t.Errorf("%d×%d: 1 マス %d、本体 %v。%d と %v のはず", tc.w, tc.h, l.Cell, l.Area, tc.cell, tc.area)
			}
			if !l.Area.In(box) {
				t.Errorf("%d×%d: 本体 %v が 128×128 の外", tc.w, tc.h, l.Area)
			}
		}
		// 欄は x = 512〜831、y = 360〜511。本体の左右に 96 px、上に見出しの 16 px、下に 8 px の余白がある。
		if MinimapRect() != image.Rect(512, 360, 832, 512) || box.Min.X-512 != 96 || 832-box.Max.X != 96 || box.Min.Y-360 != 16 || 512-box.Max.Y != 8 {
			t.Errorf("欄 %v と本体 %v の余白", MinimapRect(), box)
		}
		// 盤面がないときは本体を作らない。
		if l := minimapLayout(0, 0); l.Cell != 0 || !l.Area.Empty() {
			t.Errorf("空の盤面 = %+v", l)
		}
	})

	t.Run("TC-FR-MAP-06-05 四隅・余白・長方形の画面の座標とマスの変換", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			w, h   int
			x, y   int
			want   tactics.Pos
			wantOK bool
		}{
			{"32×32 左上", 32, 32, 608, 376, tactics.Pos{X: 0, Y: 0}, true},
			{"32×32 右上", 32, 32, 735, 376, tactics.Pos{X: 31, Y: 0}, true},
			{"32×32 左下", 32, 32, 608, 503, tactics.Pos{X: 0, Y: 31}, true},
			{"32×32 右下", 32, 32, 735, 503, tactics.Pos{X: 31, Y: 31}, true},
			{"32×32 マスの境目", 32, 32, 611, 380, tactics.Pos{X: 0, Y: 1}, true},
			{"32×32 左の余白", 32, 32, 607, 400, tactics.Pos{}, false},
			{"32×32 右の余白", 32, 32, 736, 400, tactics.Pos{}, false},
			{"32×32 見出し", 32, 32, 650, 375, tactics.Pos{}, false},
			{"32×32 下の余白", 32, 32, 650, 504, tactics.Pos{}, false},
			{"16×16 右下", 16, 16, 735, 503, tactics.Pos{X: 15, Y: 15}, true},
			{"16×16 マスの境目", 16, 16, 616, 383, tactics.Pos{X: 1, Y: 0}, true},
			{"32×18 上の余白", 32, 18, 700, 403, tactics.Pos{}, false},
			{"32×18 上端", 32, 18, 700, 404, tactics.Pos{X: 23, Y: 0}, true},
			{"32×18 下端", 32, 18, 700, 475, tactics.Pos{X: 23, Y: 17}, true},
			{"32×18 下の余白", 32, 18, 700, 476, tactics.Pos{}, false},
			{"20×16 左の余白", 20, 16, 611, 400, tactics.Pos{}, false},
			{"20×16 右端", 20, 16, 731, 487, tactics.Pos{X: 19, Y: 15}, true},
		} {
			p, ok := minimapLayout(tc.w, tc.h).CellAt(tc.x, tc.y)
			if ok != tc.wantOK || ok && p != tc.want {
				t.Errorf("%s: (%d, %d) → %v %v。%v %v のはず", tc.name, tc.x, tc.y, p, ok, tc.want, tc.wantOK)
			}
		}
	})

	t.Run("TC-FR-MAP-06-05 クリックでカメラの中心を移し、今の上限で止める", func(t *testing.T) {
		for _, tc := range []struct {
			w, h int
			p    tactics.Pos
			want image.Point
		}{
			{32, 32, tactics.Pos{X: 20, Y: 10}, image.Pt(12, 2)},
			{32, 32, tactics.Pos{X: 0, Y: 0}, image.Pt(0, 0)},
			{32, 32, tactics.Pos{X: 31, Y: 31}, image.Pt(16, 16)},
			{32, 32, tactics.Pos{X: 8, Y: 24}, image.Pt(0, 16)},
			{32, 18, tactics.Pos{X: 31, Y: 17}, image.Pt(16, 2)},
			{16, 16, tactics.Pos{X: 15, Y: 15}, image.Pt(0, 0)},
		} {
			cam := Camera{X: 3, Y: 3, Width: tc.w, Height: tc.h}
			cam.CenterOn(tc.p)
			if cam.X != tc.want.X || cam.Y != tc.want.Y || cam.Width != tc.w || cam.Height != tc.h {
				t.Errorf("%d×%d の %v: カメラ %+v、%v のはず", tc.w, tc.h, tc.p, cam, tc.want)
			}
		}
	})

	t.Run("TC-FR-MAP-06-05 戦闘: 地形・生きているユニット・表示範囲・分かっている予兆を盤面と同じ元で描く", func(t *testing.T) {
		c := newIdleWith(t, minimapSetup)
		v := c.Minimap()
		if !v.Available || v.Layout.Cell != 4 || len(v.Cells) != 32*32 {
			t.Fatalf("前提: %+v", v.Layout)
		}
		kinds := map[image.Rectangle]MinimapTerrain{}
		for _, cell := range v.Cells {
			kinds[cell.Rect] = cell.Terrain
		}
		for _, tc := range []struct {
			p    tactics.Pos
			want MinimapTerrain
		}{
			{tactics.Pos{X: 0, Y: 0}, MinimapForest}, {tactics.Pos{X: 31, Y: 0}, MinimapMountain}, {tactics.Pos{X: 0, Y: 31}, MinimapFort},
			{tactics.Pos{X: 31, Y: 31}, MinimapWall}, {tactics.Pos{X: 5, Y: 5}, MinimapWall}, {tactics.Pos{X: 6, Y: 5}, MinimapPlain},
		} {
			x, y := minimapPt(4, 608, 376, tc.p, 0, 0)
			if got := kinds[image.Rect(x, y, x+4, y+4)]; got != tc.want {
				t.Errorf("%v の地形 = %v、%v のはず", tc.p, got, tc.want)
			}
		}
		want := map[image.Rectangle]tactics.Side{
			image.Rect(688, 456, 692, 460): tactics.Player, // レオン (20, 20)
			image.Rect(612, 380, 616, 384): tactics.Player, // ガルド (1, 1)
			image.Rect(728, 496, 732, 500): tactics.Enemy,  // ボス (30, 30)
			image.Rect(692, 456, 696, 460): tactics.Enemy,  // ファイター (21, 20)
		}
		if got := unitRects(v); !reflect.DeepEqual(got, want) {
			t.Errorf("ユニット = %v\nのはず %v", got, want)
		}
		// カメラは主人公 (20, 20) に寄った (5, 5)。表示範囲はマス 5〜20 の枠。
		if c.camera.X != 5 || c.camera.Y != 5 || v.View != image.Rect(628, 396, 692, 460) {
			t.Errorf("表示範囲 = %v（カメラ %+v）", v.View, c.camera)
		}
		// 予兆がなければ描かない。
		if len(v.Hints) != 0 {
			t.Errorf("予兆 = %v", v.Hints)
		}
		// 移動先を選んだあとは、盤面と同じく移動先に置いた写しで描く。
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 19, Y: 20})
		if got := unitRects(c.Minimap()); got[image.Rect(684, 456, 688, 460)] != tactics.Player || len(got) != 4 {
			t.Errorf("移動先に置いた写し = %v", got)
		}
		c.Cancel()
		c.Cancel()
	})

	t.Run("TC-FR-MAP-06-05 戦闘: 長方形の盤面の本体と表示範囲", func(t *testing.T) {
		c := newIdleWith(t, func() tactics.Setup {
			s := wideSetup()
			s.Terrain = field(32, 18)
			s.Units[0].Pos = tactics.Pos{X: 20, Y: 10}
			s.Units[2].Pos = tactics.Pos{X: 31, Y: 17}
			s.Units[3].Pos = tactics.Pos{X: 21, Y: 10}
			return s
		})
		v := c.Minimap()
		// 32×18 は 4 px で、本体は y = 404〜475。カメラは (5, 0)、表示はマス x 5〜20・y 0〜15。
		if v.Layout.Area != image.Rect(608, 404, 736, 476) || c.camera.X != 5 || c.camera.Y != 0 || v.View != image.Rect(628, 404, 692, 468) {
			t.Errorf("本体 %v、カメラ %+v、表示範囲 %v", v.Layout.Area, c.camera, v.View)
		}
		if got := unitRects(v); got[image.Rect(732, 472, 736, 476)] != tactics.Enemy {
			t.Errorf("右下のボス = %v", got)
		}
		// 16×16 より小さい盤面でも、表示範囲は盤面の中に切る。
		k := newIdle(t)
		if v := k.Minimap(); v.Layout.Cell != 21 || v.View != v.Layout.Area {
			t.Errorf("6×6: %+v、表示範囲 %v", v.Layout, v.View)
		}
	})

	t.Run("TC-FR-MAP-06-05 戦闘: 予兆は分かっているものだけ、倒れた敵とまだ出ていない増援は描かない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		if v := c.Minimap(); len(v.Hints) != 0 {
			t.Errorf("1T の予兆 = %v", v.Hints)
		}
		endTurn(c)
		v := c.Minimap()
		// 16×16 は 8 px。2T の予兆 x 4〜11・y 0〜7 と x 0〜7・y 8〜15（hint_test の手計算）。
		wantHints := []image.Rectangle{image.Rect(640, 376, 704, 440), image.Rect(608, 440, 672, 504)}
		if !reflect.DeepEqual(v.Hints, wantHints) {
			t.Errorf("予兆 = %v、%v のはず", v.Hints, wantHints)
		}
		// まだ出ていない 3 体の増援は描かない（レオン・ガルド・ボスの 3 体だけ）。
		want := map[image.Rectangle]tactics.Side{
			image.Rect(608, 376, 616, 384): tactics.Player,
			image.Rect(608, 392, 616, 400): tactics.Player,
			image.Rect(728, 496, 736, 504): tactics.Enemy,
		}
		if got := unitRects(v); !reflect.DeepEqual(got, want) {
			t.Errorf("ユニット = %v\nのはず %v", got, want)
		}
		// 再生中は盤面と同じく予兆を隠し、出来事の写しのユニットで描く。
		c.EndPhase()
		c.EndPhase()
		if c.State() != Playing {
			t.Fatal("前提: 再生中でない")
		}
		e, _, _ := c.Event()
		v = c.Minimap()
		if len(v.Hints) != 0 {
			t.Errorf("再生中の予兆 = %v", v.Hints)
		}
		alive := 0
		for _, u := range e.Units {
			if u.Alive {
				alive++
			}
		}
		if len(v.Units) != alive {
			t.Errorf("再生中のユニット %d、出来事の写しの生きているユニット %d", len(v.Units), alive)
		}

		// ファイターの HP を 1 にした盤面で、主人公がファイターを倒す。倒れた敵は描かない。
		k := newIdleWith(t, func() tactics.Setup {
			s := basicSetup()
			s.Units[3].Stats.HP = 1
			return s
		})
		target(t, k, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0})
		k.Click(tactics.Pos{X: 1, Y: 0})
		playOut(k)
		if k.Battle().Units()[3].Alive {
			t.Fatal("前提: ファイターが倒れていない")
		}
		// 6×6 は 21 px で、本体は x = 609〜734、y = 377〜502。ファイターのいた (1, 0) は x = 630〜651。
		for r, side := range unitRects(k.Minimap()) {
			if r == image.Rect(630, 377, 651, 398) {
				t.Errorf("倒れた敵 %v %v を描いた", r, side)
			}
		}
		if n := len(k.Minimap().Units); n != 3 {
			t.Errorf("ユニットの数 = %d、3 のはず", n)
		}
	})

	t.Run("TC-FR-MAP-06-05 戦闘のクリック: 地図だけカメラを動かし、余白と再生中は無視し、選択・乱数・履歴・敵の範囲を変えない", func(t *testing.T) {
		c := newIdleWith(t, minimapSetup)
		control := newIdleWith(t, minimapSetup)
		g := &Game{ctrl: c}
		// ボスをクリックして範囲を出し、Q も出しておく。
		for _, k := range []*Controller{c, control} {
			k.Click(tactics.Pos{X: 30, Y: 30})
			k.ToggleDanger()
		}
		if !reflect.DeepEqual(c.watched, map[int]bool{2: true}) || !c.danger {
			t.Fatal("前提: 敵の範囲")
		}
		for _, tc := range []struct {
			name string
			x, y int
			want image.Point
		}{
			{"右下の隅", 735, 503, image.Pt(16, 16)},
			{"左上の隅", 608, 376, image.Pt(0, 0)},
			{"中ほど (20, 10)", 689, 417, image.Pt(12, 2)},
			{"左の余白は無視", 600, 400, image.Pt(12, 2)},
			{"見出しは無視", 650, 362, image.Pt(12, 2)},
			{"下の余白は無視", 650, 508, image.Pt(12, 2)},
		} {
			g.leftClick(tc.x, tc.y)
			if c.camera.X != tc.want.X || c.camera.Y != tc.want.Y {
				t.Errorf("%s: カメラ %+v、%v のはず", tc.name, c.camera, tc.want)
			}
			if c.State() != Idle || !reflect.DeepEqual(c.watched, map[int]bool{2: true}) || !c.danger || c.message != "" {
				t.Errorf("%s: 状態 %v、範囲 %v %v、知らせ %q", tc.name, c.State(), c.watched, c.danger, c.message)
			}
		}
		// ミニマップの欄の外は受けない（盤面と情報欄に渡す）。
		if c.MinimapClick(511, 400) || c.MinimapClick(650, 359) || c.MinimapClick(650, 512) || !c.MinimapClick(512, 360) || !c.MinimapClick(831, 511) {
			t.Error("欄の境界")
		}
		// 自軍を選んでいる間も動かせ、選択は変わらない。
		c.camera.CenterOn(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 20, Y: 20})
		g.leftClick(608, 376)
		if c.State() != Selected || c.unit != 0 || c.camera.X != 0 || c.camera.Y != 0 {
			t.Errorf("選択中: 状態 %v、自軍 %d、カメラ %+v", c.State(), c.unit, c.camera)
		}
		// 行動の一覧を出している間は、矢印・ホイールと同じく動かさない（一覧が移動先から離れないように）。
		c.camera.CenterOn(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 20, Y: 20})
		before := c.camera
		g.leftClick(608, 376)
		if c.State() != Moved || c.camera != before {
			t.Errorf("行動の一覧: 状態 %v、カメラ %+v", c.State(), c.camera)
		}
		c.Cancel()
		c.Cancel()
		// 再生中（自動の追いかけ）はクリックで動かさず、再生のあとはふつうに戻る。
		for _, k := range []*Controller{c, control} {
			k.EndPhase()
			k.EndPhase()
		}
		if c.State() != Playing {
			t.Fatal("前提: 再生中でない")
		}
		before = c.camera
		g.leftClick(735, 503)
		if c.camera != before {
			t.Errorf("再生中に動いた: %+v", c.camera)
		}
		playOut(c)
		playOut(control)
		g.leftClick(608, 376)
		if c.camera.X != 0 || c.camera.Y != 0 {
			t.Errorf("再生のあと: %+v", c.camera)
		}
		// 同じ入力をミニマップなしで行った戦闘と、出来事・ユニット・履歴・巻き戻しの回数が同じ（乱数を引いていない）。
		for _, k := range []*Controller{c, control} {
			k.EndPhase()
			k.EndPhase()
		}
		if !reflect.DeepEqual(c.events, control.events) || !reflect.DeepEqual(c.battle.Units(), control.battle.Units()) ||
			!reflect.DeepEqual(c.battle.History(), control.battle.History()) || c.battle.RewindsLeft() != control.battle.RewindsLeft() {
			t.Error("ミニマップのクリックで戦闘が変わった")
		}
	})

	t.Run("TC-FR-MAP-06-05 巻き戻しの下見: 選んだ行動の前の盤面で描き、下見のカメラだけを動かす。確認中は動かさない", func(t *testing.T) {
		c := newIdleWith(t, minimapSetup)
		// レオンを (20, 20) → (18, 20) に動かして待機する。
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 18, Y: 20})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.camera.CenterOn(tactics.Pos{X: 31, Y: 31})
		normal := c.camera
		c.OpenRewind()
		v := c.Minimap()
		// 下見は行動の前なので、レオンは (20, 20) に描く。
		if got := unitRects(v); got[image.Rect(688, 456, 692, 460)] != tactics.Player || len(got) != 4 {
			t.Errorf("下見のユニット = %v", got)
		}
		if v.View != cameraFrame(c.rewind.camera) {
			t.Errorf("下見の表示範囲 %v（下見のカメラ %+v）", v.View, c.rewind.camera)
		}
		x, y := minimapPt(4, 608, 376, tactics.Pos{X: 31, Y: 0}, 1, 1)
		if !c.MinimapClick(x, y) || c.rewind.camera.X != 16 || c.rewind.camera.Y != 0 || c.camera != normal {
			t.Errorf("下見のカメラ %+v、ふつうのカメラ %+v", c.rewind.camera, c.camera)
		}
		// 右欄の一覧を通さない（行の選択を変えない）。
		selected := c.rewind.selected
		c.RewindEnter()
		if !c.rewind.confirm {
			t.Fatal("前提: 確認でない")
		}
		x, y = minimapPt(4, 608, 376, tactics.Pos{X: 0, Y: 31}, 1, 1)
		before := c.rewind.camera
		if !c.MinimapClick(x, y) || c.rewind.camera != before || !c.rewind.confirm || c.rewind.selected != selected {
			t.Errorf("確認中: カメラ %+v、確認 %v、選択 %d", c.rewind.camera, c.rewind.confirm, c.rewind.selected)
		}
		c.Cancel()
		c.Cancel()
		if c.rewind.open || c.camera != normal || c.battle.RewindsLeft() != 10 {
			t.Errorf("閉じたあと: %+v、残り %d", c.camera, c.battle.RewindsLeft())
		}
	})

	t.Run("TC-FR-MAP-06-05 配置: 配置の盤面で描き、クリックで配置のカメラだけを動かす。仮配置と選択は変えない", func(t *testing.T) {
		s := minimapDeployScenes(t)
		v := s.View()
		if v.Minimap == nil || !v.Minimap.Available || v.Minimap.Layout.Cell != 4 {
			t.Fatalf("前提: %+v", v.Minimap)
		}
		// 主人公はスロット 0 の (31, 31)、仲間はスロット 1 の (0, 0)、敵のボスは (30, 30)。予兆は描かない。
		want := map[image.Rectangle]tactics.Side{
			image.Rect(732, 500, 736, 504): tactics.Player,
			image.Rect(608, 376, 612, 380): tactics.Player,
			image.Rect(728, 496, 732, 500): tactics.Enemy,
		}
		if got := unitRects(*v.Minimap); !reflect.DeepEqual(got, want) || len(v.Minimap.Hints) != 0 {
			t.Errorf("配置のユニット = %v、予兆 %v", got, v.Minimap.Hints)
		}
		if v.Minimap.View != image.Rect(672, 440, 736, 504) {
			t.Errorf("配置の表示範囲 = %v（カメラ %+v）", v.Minimap.View, s.deployCamera)
		}
		clickDeployCell(s, tactics.Pos{X: 15, Y: 15})
		if s.deploySelected != 0 {
			t.Fatal("前提: 主人公を選べない")
		}
		slots := append([]int(nil), s.deploySlots...)
		s.Click(608, 376)
		if s.deployCamera.X != 0 || s.deployCamera.Y != 0 || s.deploySelected != 0 || !reflect.DeepEqual(s.deploySlots, slots) || s.message != "" {
			t.Errorf("配置のクリック: カメラ %+v、選択 %d、枠 %v、知らせ %q", s.deployCamera, s.deploySelected, s.deploySlots, s.message)
		}
		s.Click(600, 400)
		if s.deployCamera.X != 0 || s.deployCamera.Y != 0 || s.deploySelected != 0 {
			t.Errorf("余白のクリック: %+v", s.deployCamera)
		}
		// 同じフレームの盤面の移動があれば、クリックは受けない（DES-UI-25 の優先順位を保つ）。
		s.DeployInput(1, 0, 735, 503, 0, false, true)
		if s.deployCamera.X != 1 || s.deployCamera.Y != 0 {
			t.Errorf("移動と同じフレームのクリック: %+v", s.deployCamera)
		}
		if s.Scene() != Deploy || s.battle != nil {
			t.Error("配置を抜けた")
		}
	})
}

// cameraFrame は、カメラの表示範囲（16 マス、盤面の中で切る）を 32×32 のミニマップの枠として手で求める。
func cameraFrame(c Camera) image.Rectangle {
	x1, y1 := min(c.Width, c.X+16), min(c.Height, c.Y+16)
	return image.Rect(608+4*c.X, 376+4*c.Y, 608+4*x1, 376+4*y1)
}

// minimapDeployScenes は 32×32 の配置の場面を作る。出撃のマスは (31, 31)・(0, 0)・(31, 0)・(0, 31)、敵のボスは (30, 30)。
func minimapDeployScenes(t *testing.T) *SceneController {
	t.Helper()
	rows := make([]string, 32)
	for i := range rows {
		rows[i] = strings.Repeat(".", 32)
	}
	cat := defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend"}},
		[]defs.TerrainDef{{ID: "p", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{{ID: "lord", Name: "ロード", WeaponTypes: []string{"sword"}, Base: defs.Stats{HP: 20, Mov: 3}}},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", MinRange: 1, MaxRange: 1}}, nil, nil,
		[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "lord", Weapons: []string{"s"}, Lord: true, Level: 1}, {ID: "friend", Name: "仲間", Class: "lord", Weapons: []string{"s"}, Level: 1}},
		[]defs.MapDef{{ID: "m", Rows: rows, Slots: []defs.Slot{{X: 31, Y: 31}, {X: 0, Y: 0}, {X: 31, Y: 0}, {X: 0, Y: 31}},
			Enemies: []defs.EnemyPlacement{{Level: 1, Class: "lord", Weapons: []string{"s"}, AI: "hold", Boss: true, X: 30, Y: 30}}}},
		[]defs.ChapterDef{{ID: "c", Map: "m"}}, nil)
	campaign, err := core.NewCampaign(cat, core.Withdraw)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSceneController(Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { t.Fatal("配置で種を引いた"); return 0 }})
	s.campaign = campaign
	s.enterPrepare()
	s.enterDeploy()
	if s.Scene() != Deploy || s.deployCamera != (Camera{X: 16, Y: 16, Width: 32, Height: 32}) {
		t.Fatalf("前提: 場面 %v、カメラ %+v", s.Scene(), s.deployCamera)
	}
	return s
}

// rectOfText は、描く文の範囲を返す（2 倍の文は 2 倍の大きさ）。
func rectOfText(x, y int, s string, big bool) image.Rectangle {
	scale := 1
	if big {
		scale = 2
	}
	return image.Rect(x, y, x+scale*textWidth(s), y+scale*lineHeight)
}

// checkInfoArea は、情報欄に描くものがすべて右上の 320×360 に収まり、ページの印が中身と重ならないことを確かめる（DES-UI-28）。
func checkInfoArea(t *testing.T, step string, v InfoView) {
	t.Helper()
	area := image.Rect(MapSize, 0, ScreenWidth, InfoHeight)
	for _, tx := range v.Texts {
		if r := rectOfText(tx.X, tx.Y, tx.Text, tx.Big); !r.In(area) {
			t.Errorf("%s: 文 %q %v が情報欄の外", step, tx.Text, r)
		}
	}
	for _, ic := range v.Icons {
		if r := image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size); !r.In(area) {
			t.Errorf("%s: 絵 %v が情報欄の外", step, r)
		}
	}
	for _, b := range v.Bars {
		if r := image.Rect(b.X, b.Y, b.X+max(b.Fill, b.Frame), b.Y+barHeight); !r.In(area) {
			t.Errorf("%s: 棒 %v が情報欄の外", step, r)
		}
	}
	if v.Bottom > InfoHeight {
		t.Errorf("%s: 一番下 %d", step, v.Bottom)
	}
	// 同じ行の文どうしは 4 px 以上あけ、くっつけない（「残り 15/15重さ 0」のようにしない）。
	for i, a := range v.Texts {
		ra := rectOfText(a.X, a.Y, a.Text, a.Big)
		for _, b := range v.Texts[i+1:] {
			rb := rectOfText(b.X, b.Y, b.Text, b.Big)
			if a.Y == b.Y && image.Rect(ra.Min.X-4, ra.Min.Y, ra.Max.X+4, ra.Max.Y).Overlaps(rb) {
				t.Errorf("%s: 同じ行の文 %q %v と %q %v がくっつく", step, a.Text, ra, b.Text, rb)
			}
		}
	}
	for _, tab := range v.Tabs {
		if !tab.Rect.In(area) || tab.Rect.Min.Y < v.Bottom || textWidth(tab.Label)+8 > tab.Rect.Dx() {
			t.Errorf("%s: ページの印 %+v（中身の一番下 %d）", step, tab, v.Bottom)
		}
	}
	if len(v.Tabs) > 0 {
		g := InfoTabGuide()
		if r := rectOfText(g.X, g.Y, g.Text, false); !r.In(area) || r.Overlaps(v.Tabs[1].Rect) {
			t.Errorf("%s: Tab の案内 %v", step, r)
		}
	}
}

// 情報欄を 320×360 に置き直す（DES-IT06 12.2・12.5 節、DES-UI-28）。戦闘と配置の右欄は上下 2 段と操作行にし、
// 2 ページの切り替えは詳しい画面に置き換えた（DES-UI-31、E-120。#166）。準備の情報欄は 2 ページのまま。
func TestInfoPagesLayout(t *testing.T) {
	t.Run("TC-FR-MAP-06-07 ミニマップの見出し・凡例は欄の余白にあり、地図の本体と重ならない", func(t *testing.T) {
		c := newIdleWith(t, minimapSetup)
		v := c.Minimap()
		box := image.Rect(608, 376, 736, 504)
		if len(v.Legend) != 4 {
			t.Fatalf("凡例 = %+v", v.Legend)
		}
		for _, l := range v.Legend {
			r := rectOfText(l.Text.X, l.Text.Y, l.Text.Text, false)
			if !r.In(MinimapRect()) || r.Overlaps(box) || !l.Swatch.In(MinimapRect()) || l.Swatch.Overlaps(box) || l.Swatch.Overlaps(r) {
				t.Errorf("凡例 %q %v、印 %v", l.Text.Text, r, l.Swatch)
			}
		}
		for _, tx := range v.Texts {
			if r := rectOfText(tx.X, tx.Y, tx.Text, false); !r.In(MinimapRect()) || r.Overlaps(box) {
				t.Errorf("文 %q %v", tx.Text, r)
			}
		}
		// 本体の外側 1 px の枠も欄の中に収まる。
		if !v.Layout.Area.Inset(-1).In(MinimapRect()) {
			t.Errorf("本体の枠 %v", v.Layout.Area.Inset(-1))
		}
		// 盤面を表示できない下見では、見出しと知らせだけで、古い地図を残さない。
		u := unavailableMinimap()
		if u.Available || len(u.Cells) != 0 || len(u.Units) != 0 || len(u.Texts) != 2 || u.Texts[0].Text != "ミニマップ" {
			t.Errorf("表示できないとき = %+v", u)
		}
	})

	t.Run("TC-FR-MAP-06-07 戦闘の右欄の上下 2 段は、カーソルだけ・選択と相手・回復の予測で y < 296 に収まり、ページの印を出さない（DES-UI-31）", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		top := image.Rect(MapSize, 0, ScreenWidth, infoControlTop)
		fits := func(step string, v InfoView) {
			t.Helper()
			checkInfoArea(t, step, v)
			if len(v.Tabs) != 0 || v.Bottom > 296 {
				t.Errorf("%s: 印 %d 個、一番下 %d", step, len(v.Tabs), v.Bottom)
			}
			for _, tx := range v.Texts {
				if r := rectOfText(tx.X, tx.Y, tx.Text, tx.Big); !r.In(top) {
					t.Errorf("%s: 文 %q %v が操作行（y = 304〜）にかかる", step, tx.Text, r)
				}
			}
		}
		fits("カーソルだけ", c.BattleInfoView(cellCenter(tactics.Pos{X: 2, Y: 3})))
		// キャラと兵種の両方のスキル（#124）と手持ちを持つユニットも、同じ人物の下段（地形・スキル）が収まる。
		two := tactics.Unit{Name: "エルザ", ClassName: "騎兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, HP: 20, Weapons: []tactics.Weapon{testLance}, Alive: true,
			Skill: tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}, ClassSkills: []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}},
			Items: []tactics.ItemStack{{Item: potion, Count: 2}}}
		fits("2 つのスキル", twoTierInfo([]tactics.Unit{two}, 0, 0, &forest))
		c.Click(tactics.Pos{X: 2, Y: 3})
		fits("選択と相手", c.BattleInfoView(cellCenter(tactics.Pos{X: 3, Y: 3})))
		// 盤面の外（ミニマップの上）では選んでいるユニットを出す。
		if got := rows(c.BattleInfoView(650, 400)); len(got) == 0 || got[0] != "レオン" {
			t.Errorf("盤面の外: %q", got)
		}

		// 回復の予測の間も、右欄は 2 段のまま収まる（予測は共通の窓に出す）。
		h := healController(t, 15)
		moveHealer(h, tactics.Pos{X: 3, Y: 2})
		h.ChooseAction(tactics.Heal)
		h.Click(tactics.Pos{X: 3, Y: 1})
		if h.State() != HealTargeting || h.target != 1 {
			t.Fatalf("前提: 状態 %v、相手 %d", h.State(), h.target)
		}
		v := h.BattleInfoView(cellCenter(tactics.Pos{X: 3, Y: 1}))
		fits("回復の予測", v)
		if strings.Contains(strings.Join(rows(v), "\n"), "回復 +") {
			t.Errorf("回復の予測を右欄に出した: %q", rows(v))
		}
	})

	t.Run("TC-FR-MAP-06-07 能力の 7 つの棒、武器と杖の行の桁、5 本の武器と 2 つのスキル（開発者の確認（2026-11-01）で直した）", func(t *testing.T) {
		x0 := MapSize + infoMargin
		u := tactics.Unit{Name: "ノエル", ClassName: "錬金術師", Side: tactics.Player, Alive: true, HP: 20,
			Stats: tactics.Stats{HP: 20, Str: 4, Mag: 7, Skl: 5, Spd: 6, Lck: 9, Def: 3, Res: 8, Mov: 5},
			Weapons: []tactics.Weapon{
				{Name: "勇者の斧", Type: tactics.Axe, Might: 6, Hit: 65, Crit: 0, Weight: 14, Strikes: 2, MinRange: 1, MaxRange: 1},
				{Name: "アーマーキラー", Type: tactics.Sword, Might: 4, Hit: 80, Weight: 7, MinRange: 1, MaxRange: 1},
				{Name: "ホースキラー", Type: tactics.Lance, Might: 6, Hit: 75, Weight: 10, MinRange: 1, MaxRange: 1},
				{Name: "キルソード", Type: tactics.Sword, Might: 6, Hit: 85, Crit: 30, Weight: 7, MinRange: 1, MaxRange: 1},
				{Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, Uses: 15, MaxUses: 15, MinRange: 1, MaxRange: 1},
			},
			Skill: tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}, ClassSkills: []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}},
			Items: []tactics.ItemStack{{Item: potion, Count: 2}}}
		// 「能力」のページ: HP の枠のほかに、力・魔力・技・速さ・幸運・守備・魔防の 7 本を同じ左端・同じ式で 1 列に並べる。
		v := Panel{Unit: &u, Terrain: &forest, Tabs: true}.View()
		checkInfoArea(t, "能力", v)
		var prevY int
		for i, tc := range []struct {
			label string
			value int
		}{{"力 4", 4}, {"魔力 7", 7}, {"技 5", 5}, {"速さ 6", 6}, {"幸運 9", 9}, {"守備 3", 3}, {"魔防 8", 8}} {
			b := barOf(t, v, tc.label)
			tx := textOf(t, v, tc.label)
			if b.Fill != StatBarLen(tc.value) || b.Frame != 0 || b.X != x0+barX || tx.X != x0 {
				t.Errorf("%s: 棒 %+v、文 %+v", tc.label, b, tx)
			}
			if i > 0 && tx.Y != prevY+lineHeight {
				t.Errorf("%s の行 y = %d、前の行 %d の次のはず", tc.label, tx.Y, prevY)
			}
			prevY = tx.Y
		}
		if len(v.Bars) != 8 {
			t.Errorf("棒の数 = %d、8 のはず", len(v.Bars))
		}
		// 「装備・スキル」のページ: 2 回攻撃の武器は「必殺 0 ×2」、2 行目の 3 つの値は決まった桁。
		u.Equipped = 0
		v = Panel{Unit: &u, Terrain: &forest, Page: PageEquip, Tabs: true}.View()
		checkInfoArea(t, "5 本と 2 つのスキル", v)
		if tx := textOf(t, v, "必殺 0 ×2"); tx.X != x0+weaponSideX {
			t.Errorf("必殺の x = %d", tx.X)
		}
		weapon := textOf(t, v, "武器: 勇者の斧（斧）")
		for i, s := range []string{"威力 6", "命中 65", "射程 1"} {
			if tx := textOf(t, v, s); tx.X != x0+weaponCols[i] || tx.Y != weapon.Y+lineHeight {
				t.Errorf("%q の位置 (%d, %d)", s, tx.X, tx.Y)
			}
		}
		if tx := textOf(t, v, "重さ 14"); tx.X != x0+weaponSideX || tx.Y != weapon.Y+lineHeight {
			t.Errorf("重さの位置 (%d, %d)", tx.X, tx.Y)
		}
		// ほかの 4 本は欄の幅で折り返し、2 つのスキルの説明まで印（y = 336）の上に収まる。
		all := strings.Join(rows(v), "")
		for _, name := range []string{"アーマーキラー", "ホースキラー", "キルソード", "癒やしの杖", "怒り、再移動"} {
			if !strings.Contains(all, name) {
				t.Errorf("%q がない: %q", name, rows(v))
			}
		}
		// 攻撃の回数が 1 の武器は「×1」を省き、杖は必殺と回数を出さずに回復・射程・残りを同じ桁に置く。
		u.Equipped = 3
		v = Panel{Unit: &u, Page: PageEquip}.View()
		textOf(t, v, "必殺 30")
		u.Equipped = 4
		v = Panel{Unit: &u, Page: PageEquip}.View()
		staff := textOf(t, v, "武器: 癒やしの杖（杖）")
		for i, s := range []string{"回復 10", "射程 1", "残り 15/15"} {
			if tx := textOf(t, v, s); tx.X != x0+weaponCols[i] || tx.Y != staff.Y+lineHeight {
				t.Errorf("杖の %q の位置 (%d, %d)", s, tx.X, tx.Y)
			}
		}
		checkInfoArea(t, "杖", v)
		for _, tx := range v.Texts {
			if strings.HasPrefix(tx.Text, "必殺") || strings.HasPrefix(tx.Text, "重さ") || strings.Contains(tx.Text, "×") && !strings.HasPrefix(tx.Text, "手持ち") {
				t.Errorf("杖に必殺・回数・重さ %q", tx.Text)
			}
		}
		// 一番長い値でも、3 つ目の桁と右の桁は 8 px 以上あく。
		for _, s := range []string{"残り 15/15", "射程 1〜3", "命中 100"} {
			if weaponCols[2]+textWidth(s)+8 > weaponSideX {
				t.Errorf("%q が右の桁とくっつく（%d + %d + 8 > %d）", s, weaponCols[2], textWidth(s), weaponSideX)
			}
		}
		// 値の桁どうし、右の桁は重ならない。
		for i := 0; i < 2; i++ {
			if weaponCols[i]+textWidth("残り 15/15") > weaponCols[i+1] && weaponCols[i]+textWidth("命中 100") > weaponCols[i+1] {
				t.Errorf("桁 %d が次の桁と重なる", i)
			}
		}
		if weaponCols[2]+textWidth("残り 15/15") > weaponSideX || weaponCols[2]+textWidth("射程 1〜3") > weaponSideX {
			t.Error("3 つ目の桁が右の桁と重なる")
		}
	})

	t.Run("TC-FR-MAP-06-07 カーソルが情報欄の上にあるときは、選んでいるユニットか直前に盤面で見ていたマスを出し続け、「詳細 I」で同じ人物を開く", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		g := &Game{ctrl: c}
		r := detailButtonRect
		tx, ty := r.Min.X+2, r.Min.Y+2
		// 何も見ていなければ空で、「詳細 I」は押せず、押しても開かない。
		if v := c.BattleInfoView(tx, ty); len(v.Texts) != 0 || c.DetailButton().Enabled {
			t.Errorf("まだ盤面を見ていない: %q、押せる %v", rows(v), c.DetailButton().Enabled)
		}
		// レオン (2, 3) を見てからボタンへ動かすと、レオンを出し続け、クリックでレオンの詳しい画面を開く。状態は変えない。
		x, y := cellCenter(tactics.Pos{X: 2, Y: 3})
		c.SetCursor(x, y)
		c.SetCursor(tx, ty)
		if got := rows(c.BattleInfoView(tx, ty)); len(got) == 0 || got[0] != "レオン" || !c.DetailButton().Enabled {
			t.Errorf("ボタンの上: %q", got)
		}
		g.leftClick(tx, ty)
		if !c.DetailOpen() || c.detailUnit != 0 || c.State() != Idle {
			t.Errorf("詳細のクリック: 開く %v、ユニット %d、状態 %v", c.DetailOpen(), c.detailUnit, c.State())
		}
		c.DetailInput(false, true)
		// ボタンの外（左と下）のクリックは受けない。
		if c.InfoClick(r.Min.X-1, r.Min.Y+2) || c.InfoClick(r.Min.X+2, r.Max.Y) || c.DetailOpen() {
			t.Error("ボタンの外のクリックを受けた")
		}
		// 選んでいるユニットがいれば、直前に見ていた空きマスより優先する。
		c.Click(tactics.Pos{X: 2, Y: 3})
		x, y = cellCenter(tactics.Pos{X: 0, Y: 0})
		c.SetCursor(x, y)
		c.SetCursor(tx, ty)
		if got := rows(c.BattleInfoView(tx, ty)); c.State() != Selected || len(got) == 0 || got[0] != "レオン" {
			t.Errorf("選んでいるとき: 状態 %v、%q", c.State(), got)
		}
		// 盤面の上では、カーソルの下のユニットを上段、選んでいるユニットを下段に出す。
		x, y = cellCenter(tactics.Pos{X: 3, Y: 3})
		if got := rows(c.BattleInfoView(x, y)); len(got) < 7 || got[0] != "ソルジャー 3　★ボス" || got[6] != "レオン" {
			t.Errorf("盤面の上: %q", got)
		}
		// 選んでいなければ直前のマスに戻る。空きマスなら地形だけ。ミニマップの上では出し続けない。
		c.Cancel()
		if v := c.BattleInfoView(650, 400); len(v.Texts) != 0 {
			t.Errorf("ミニマップの上: %q", rows(v))
		}
		x, y = cellCenter(tactics.Pos{X: 0, Y: 0})
		c.SetCursor(x, y)
		if got := rows(c.BattleInfoView(tx, ty)); len(got) != 1 || got[0] != "地形: 平地　回避 0　守備 0　回復 0%" {
			t.Errorf("空きマスのあと: %q", got)
		}
	})

	t.Run("TC-FR-MAP-06-07 Tab は詳しい画面の節だけを送り、選択・カメラ・敵の範囲は変えない（E-120）", func(t *testing.T) {
		c := newIdleWith(t, minimapSetup)
		g := &Game{ctrl: c}
		c.Click(tactics.Pos{X: 20, Y: 20})
		if c.State() != Selected {
			t.Fatal("前提: 選択")
		}
		cam := c.camera
		// 詳しい画面を出していない Tab は何もしない。
		if c.DetailInput(true, false) || c.DetailOpen() {
			t.Error("詳しい画面がないのに Tab を受けた")
		}
		g.leftClick(detailButtonRect.Min.X+2, detailButtonRect.Min.Y+2)
		if !c.DetailOpen() || c.detailSection != SectionStats {
			t.Fatalf("詳細のクリック: 開く %v、節 %d", c.DetailOpen(), c.detailSection)
		}
		for _, want := range []DetailSection{SectionWeapons, SectionSkills, SectionStats} {
			c.DetailInput(true, false)
			if c.detailSection != want || c.State() != Selected || c.unit != 0 || c.camera != cam {
				t.Errorf("Tab: 節 %d（%d のはず）、状態 %v、カメラ %+v", c.detailSection, want, c.State(), c.camera)
			}
		}
		// 詳しい画面の間の左クリックは、盤面にもボタンにも渡さない。
		g.leftClick(cellCenter(tactics.Pos{X: 21, Y: 20}))
		if c.State() != Selected || !c.DetailOpen() {
			t.Errorf("詳しい画面の間のクリック: 状態 %v", c.State())
		}
		c.DetailInput(false, true)
		// 敵をクリックして出した範囲も、詳しい画面の開け閉めで変わらない。
		c.Cancel()
		c.Click(tactics.Pos{X: 30, Y: 30})
		c.camera.Focus(tactics.Pos{X: 30, Y: 30})
		ep := c.Camera().Screen(tactics.Pos{X: 30, Y: 30})
		c.SetCursor(ep.X+16, ep.Y+16)
		if !c.OpenDetail() || c.detailUnit != 2 {
			t.Fatalf("敵の詳しい画面: %d", c.detailUnit)
		}
		c.DetailInput(true, false)
		c.DetailInput(false, true)
		if !reflect.DeepEqual(c.watched, map[int]bool{2: true}) || c.State() != Idle {
			t.Errorf("敵の範囲 %v、状態 %v", c.watched, c.State())
		}
		// 巻き戻しの一覧を開いている間は、詳しい画面を開かず、「詳細 I」のクリックも受けない。
		c.OpenRewind()
		if c.OpenDetail() || c.InfoClick(detailButtonRect.Min.X+2, detailButtonRect.Min.Y+2) || c.DetailOpen() {
			t.Error("一覧の間に詳しい画面を開いた")
		}
	})

	t.Run("TC-FR-MAP-06-07 配置と準備の情報欄・ボタン・印はミニマップと重ならず、印のクリックは表示だけを変える", func(t *testing.T) {
		s := minimapDeployScenes(t)
		clickDeployCell(s, tactics.Pos{X: 15, Y: 15})
		// 配置の右欄は上下 2 段で、ページの印を出さない（DES-UI-31）。ボタンは中身の下に置く。
		v := s.View()
		if v.Info == nil || len(v.Info.Tabs) != 0 {
			t.Fatalf("配置の情報欄: %+v", v.Info)
		}
		checkInfoArea(t, "配置", *v.Info)
		for i, b := range v.Buttons {
			if !b.Rect.In(image.Rect(MapSize, 0, ScreenWidth, InfoHeight)) || b.Rect.Min.Y < v.Info.Bottom {
				t.Errorf("ボタン %q %v（情報の下 %d）", b.Label, b.Rect, v.Info.Bottom)
			}
			for _, other := range v.Buttons[i+1:] {
				if b.Rect.Overlaps(other.Rect) {
					t.Errorf("ボタンが重なる %v %v", b.Rect, other.Rect)
				}
			}
		}
		for _, tx := range v.Texts {
			if rectOfText(tx.X, tx.Y, tx.Text, tx.Big).Overlaps(MinimapRect()) {
				t.Errorf("配置の文 %q がミニマップに重なる", tx.Text)
			}
		}
		// 元の印の位置のクリックと Tab は、配置の選択と枠を変えない。
		slots := append([]int(nil), s.deploySlots...)
		r := InfoTabRects()[1]
		s.Click(r.Min.X+2, r.Min.Y+2)
		s.ToggleInfoPage()
		if s.infoPage != PageStats || s.deploySelected != 0 || !reflect.DeepEqual(s.deploySlots, slots) || s.Scene() != Deploy {
			t.Errorf("配置の印の位置のクリックと Tab: ページ %d、選択 %d、枠 %v", s.infoPage, s.deploySelected, s.deploySlots)
		}
		// 準備（ミニマップなし）でも同じ印を出し、クリックで切り替える。選んでいる仲間と出撃の印は変えない。
		p := prepareScenes(t)
		member, selected := p.prepareMember, p.SelectedMembers()
		v = p.View()
		if v.Info == nil || len(v.Info.Tabs) != 2 || v.Minimap != nil {
			t.Fatalf("準備の情報欄 %+v、ミニマップ %v", v.Info, v.Minimap)
		}
		// 準備の情報欄は右列 (416,56)〜(816,472) にずらす（DES-UI-32）。印も同じだけずれる。
		pr := v.Info.Tabs[1].Rect
		if pr != r.Add(image.Pt(416-MapSize, 56)) {
			t.Errorf("準備の印 %v", pr)
		}
		p.Click(r.Min.X+2, r.Min.Y+2)
		if p.infoPage != PageStats {
			t.Errorf("準備で配置の印の位置を受けた: %d", p.infoPage)
		}
		p.Click(pr.Min.X+2, pr.Min.Y+2)
		if p.infoPage != PageEquip || p.prepareMember != member || !reflect.DeepEqual(p.SelectedMembers(), selected) || p.Scene() != Prepare {
			t.Errorf("準備の印のクリック: ページ %d、仲間 %d", p.infoPage, p.prepareMember)
		}
	})

	t.Run("TC-FR-MAP-06-07 Targeting の詳しい画面の Tab は節だけを送り、閉じると予測に戻る（REV-071 F-01、E-120）", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		if _, ok := c.attackForecast(); !ok || c.State() != Targeting {
			t.Fatal("前提: 攻撃の予測")
		}
		target, weapon := c.target, c.weapon
		c.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		if c.DetailInput(true, false) || !c.OpenDetail() {
			t.Fatal("予測の間に詳しい画面を開けない")
		}
		c.DetailInput(true, false)
		c.DetailInput(true, false)
		c.DetailInput(false, true)
		if _, ok := c.ForecastWindow(); !ok || c.State() != Targeting || c.target != target || c.weapon != weapon {
			t.Errorf("閉じたあと: 状態 %v、相手 %d、武器 %d", c.State(), c.target, c.weapon)
		}
	})

	t.Run("TC-FR-MAP-06-07 勝敗が決まった Over では詳しい画面を開かず、「詳細 I」のクリックで場面と戦闘は変えない（REV-071 F-02、DES-IT07 3.1 節）", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		winScene(t, s)
		playOut(s.battle)
		c := s.battle
		if s.Scene() != Battle || c.State() != Over {
			t.Fatalf("前提: 場面 %v、状態 %v", s.Scene(), c.State())
		}
		if c.DetailButton().Enabled {
			t.Error("Over で「詳細 I」を押せる")
		}
		old, before := c.Battle(), c.Battle().Units()
		r := detailButtonRect
		s.Click(r.Min.X+2, r.Min.Y+2)
		if c.DetailOpen() || c.OpenDetail() {
			t.Error("Over で詳しい画面を開いた")
		}
		if s.Scene() != Battle || c.State() != Over || c.Battle() != old || !reflect.DeepEqual(before, old.Units()) {
			t.Error("Over の「詳細 I」のクリックで場面か戦闘が変わった")
		}
		// ボタンの外の Over のクリックは、今までどおり勝敗の帯で次へ進む。
		e := BattleEndRect()
		s.Click(e.Min.X+2, e.Min.Y+2)
		if s.Scene() == Battle {
			t.Error("勝敗の帯のクリックで次へ進まない")
		}
	})

	t.Run("TC-FR-MAP-06-07 巻き戻しの一覧・確認・理由・気配の案内は右上と下の行に収め、ミニマップと重ならない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		endTurn(c)
		// 2T に待機して、予兆のある行動の前を選ぶ。
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.OpenRewind()
		c.rewind.message = strings.Repeat("とても長い誤りの文を折り返して読む。", 30)
		check := func(step string) {
			t.Helper()
			v := c.RewindView()
			hint := false
			for _, tx := range v.Texts {
				r := rectOfText(tx.X, tx.Y, tx.Text, false)
				if r.Overlaps(MinimapRect()) || r.Max.X > ScreenWidth {
					t.Errorf("%s: 文 %q %v がミニマップに重なる", step, tx.Text, r)
				}
				hint = hint || strings.HasPrefix(tx.Text, hintText)
			}
			// 詰めた右欄でも、文どうしは重ならない。
			for i, tx := range v.Texts {
				for _, other := range v.Texts[i+1:] {
					if rectOfText(tx.X, tx.Y, tx.Text, false).Overlaps(rectOfText(other.X, other.Y, other.Text, false)) {
						t.Errorf("%s: 文 %q と %q が重なる", step, tx.Text, other.Text)
					}
				}
			}
			if !hint {
				t.Errorf("%s: 気配の案内がない", step)
			}
			for _, row := range v.Rows {
				if row.Rect.Overlaps(MinimapRect()) {
					t.Errorf("%s: 行 %v", step, row.Rect)
				}
			}
			for _, b := range v.Buttons {
				if b.Rect.Overlaps(MinimapRect()) || !b.Rect.In(image.Rect(0, 0, ScreenWidth, ScreenHeight)) {
					t.Errorf("%s: ボタン %q %v", step, b.Label, b.Rect)
				}
			}
		}
		check("一覧")
		// 一覧の詳細は最大 5 行（DES-UI-21、DES-IT04 7.1 節。REV-071 F-03 案 B）で、y = 228 から 16 px ごと。
		var detail []int
		for _, tx := range c.RewindView().Texts {
			if tx.X == rewindLeft && tx.Y >= 228 && tx.Y < 310 {
				detail = append(detail, tx.Y)
			}
		}
		if !reflect.DeepEqual(detail, []int{228, 244, 260, 276, 292}) {
			t.Errorf("詳細の行の y = %v", detail)
		}
		// 気配の案内は下の行の「戻る」「決定」の下（y = 560）。
		for _, tx := range c.RewindView().Texts {
			if strings.HasPrefix(tx.Text, hintText) && (tx.Y != 560 || tx.X != rewindLeft) {
				t.Errorf("気配の案内の位置 (%d, %d)", tx.X, tx.Y)
			}
		}
		c.RewindEnter()
		if !c.rewind.confirm {
			t.Fatal("前提: 確認でない")
		}
		// 確認では全文が 16 行を超えても、見せる行は情報欄の中で止める（ホイールで送る）。
		if n := len(c.rewindDetail()); n <= rewindConfirmLines {
			t.Fatalf("前提: 確認の全文 %d 行", n)
		}
		check("確認")
		c.Cancel()
		c.Cancel()
		// 盤面の外の予兆の案内（下の行の右）もミニマップと重ならない。
		hv := Camera{Width: 32, Height: 32}.hintView(HintView{Visible: true, Areas: []tactics.ReinforcementHint{area(20, 20, 23, 23)}, Text: SceneText{Text: hintText}})
		if r := rectOfText(hv.Text.X, hv.Text.Y, hv.Text.Text, false); r.Overlaps(MinimapRect()) || r.Max.X > ScreenWidth || r.Max.Y > ScreenHeight {
			t.Errorf("画面外の気配の案内 %v", r)
		}
	})
}

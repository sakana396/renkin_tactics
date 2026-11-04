package ui

import (
	"image"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 期待値は DES-IT07 3.1・3.3 節（DES-UI-30、DES-UI-28 の改訂）の座標と決まりから手で求める。
// 全体図は盤面の (0,0)〜(512,512)、32×32 は 1 マス 16 px、16×16 は 32 px で中央寄せ。
// 「全体図」ボタンは (744,472)〜(824,504)、「次の味方 Z」は (704,520)〜(824,544)。
// カメラの中心は CenterOn（16 マスの表示の左上から 8 のマスを中心にし、上限で止める。DES-UI-28）。

// navGame は、戦闘の Controller を持つ場面と Game を作る（実際の入力の経路 Game.navInput → leftClick を通すため）。
func navGame(c *Controller) *Game {
	return &Game{ctrl: c, scenes: &SceneController{scene: Battle, battle: c}}
}

// allySetup は wideSetup（32×32）に、名簿の最後の自軍を足す。
//
//	0: レオン (20, 20)。主人公
//	1: ガルド (1, 1)
//	2: ボス (31, 31)、3: ファイター (21, 20)（敵）
//	4: ミア (28, 4)
func allySetup() tactics.Setup {
	s := wideSetup()
	mia := s.Units[1]
	mia.Name, mia.Pos = "ミア", tactics.Pos{X: 28, Y: 4}
	s.Units = append(s.Units, mia)
	return s
}

// overviewPt は 32×32 の全体図（1 マス 16 px、左上 (0,0)）で、マス p の中ほどの画面の座標を返す。
func overviewPt(p tactics.Pos) (int, int) { return p.X*16 + 8, p.Y*16 + 8 }

// battleSnapshot は、全体図・次の味方・ドラッグで変えてはならない戦闘の状態（位置・HP・行動済み・履歴の数）。
func battleSnapshot(c *Controller) ([]tactics.Unit, int) {
	return c.battle.Units(), len(c.battle.History())
}

// waitUnit は自軍 i をその場で待機させ、再生を終える。
func waitUnit(t *testing.T, c *Controller, i int) {
	t.Helper()
	p := c.battle.Units()[i].Pos
	c.Click(p)
	c.Click(p)
	c.ChooseAction(tactics.Wait)
	playOut(c)
	if c.State() != Idle || !c.battle.Units()[i].Acted {
		t.Fatalf("前提: 待機のあと 状態 %v、行動済み %v", c.State(), c.battle.Units()[i].Acted)
	}
}

func TestBoardNavIT07(t *testing.T) {
	t.Run("TC-FR-MAP-06-08 全体図の配置・開く状態・地図のクリック・閉じ方・ボタン", func(t *testing.T) {
		// 配置: 32×32 は 16 px、16×16 は 32 px。長方形と小さいマップは長い辺で決めて中央に寄せる。
		for _, tc := range []struct {
			w, h, cell int
			area       image.Rectangle
		}{
			{32, 32, 16, image.Rect(0, 0, 512, 512)},
			{16, 16, 32, image.Rect(0, 0, 512, 512)},
			// 32×18: 16 px で高さ 288。上下に (512 − 288) ÷ 2 = 112。
			{32, 18, 16, image.Rect(0, 112, 512, 400)},
			// 6×6: 512 ÷ 6 = 85 px で 510。左上に 1 の余白。
			{6, 6, 85, image.Rect(1, 1, 511, 511)},
		} {
			l := overviewLayout(tc.w, tc.h)
			if l.Cell != tc.cell || l.Area != tc.area {
				t.Errorf("%d×%d: 1 マス %d、範囲 %v。%d と %v のはず", tc.w, tc.h, l.Cell, l.Area, tc.cell, tc.area)
			}
		}
		if l := overviewLayout(0, 0); l.Cell != 0 || !l.Area.Empty() {
			t.Errorf("盤面がないとき: %+v", l)
		}

		c := newIdleWith(t, wideSetup)
		g := navGame(c)
		units0, hist0 := battleSnapshot(c)
		// Idle でのボタン: 全体図 (744,472)〜(824,504) と 次の味方 Z (704,520)〜(824,544)。どちらも押せる。
		want := []SceneButton{{"全体図", image.Rect(744, 472, 824, 504), true}, {"次の味方 Z", image.Rect(704, 520, 824, 544), true}}
		if got := c.NavButtons(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Idle のボタン %+v", got)
		}
		// M で開く。入力の状態は Idle のまま保つ。
		if !g.navInput(navKeys{Overview: true}, 0, 0) || !c.OverviewOpen() || c.State() != Idle {
			t.Fatalf("M で開かない: 全体図 %v、状態 %v", c.OverviewOpen(), c.State())
		}
		// 全体図の中身: 32×32 の 1024 マス、ユニットは 16 px のマスに陣営と一緒に、表示範囲は今のカメラの 16 マス。
		v := c.MapOverview()
		if v.Layout.Cell != 16 || len(v.Cells) != 32*32 || v.Cells[0].Rect != image.Rect(0, 0, 16, 16) || v.Cells[len(v.Cells)-1].Rect != image.Rect(496, 496, 512, 512) {
			t.Errorf("地形のマス: 1 マス %d、%d 個、最初 %v", v.Layout.Cell, len(v.Cells), v.Cells[0].Rect)
		}
		gotUnits := map[image.Rectangle]tactics.Side{}
		for _, u := range v.Units {
			gotUnits[u.Rect] = u.Unit.Side
		}
		wantUnits := map[image.Rectangle]tactics.Side{
			image.Rect(320, 320, 336, 336): tactics.Player, // レオン (20,20)
			image.Rect(16, 16, 32, 32):     tactics.Player, // ガルド (1,1)
			image.Rect(496, 496, 512, 512): tactics.Enemy,  // ボス (31,31)
			image.Rect(336, 320, 352, 336): tactics.Enemy,  // ファイター (21,20)
		}
		if !reflect.DeepEqual(gotUnits, wantUnits) {
			t.Errorf("全体図のユニット %v", gotUnits)
		}
		cam := c.camera
		if wantView := image.Rect(cam.X*16, cam.Y*16, (cam.X+16)*16, (cam.Y+16)*16); v.View != wantView {
			t.Errorf("表示範囲 %v、%v のはず（カメラ %+v）", v.View, wantView, cam)
		}
		// 凡例は右上の欄 (512,0)〜(832,360) に収め、右下のミニマップの欄と重ねない。
		if v.Panel != image.Rect(512, 0, 832, 360) {
			t.Errorf("凡例の欄 %v", v.Panel)
		}
		for _, tx := range v.Texts {
			if r := rectOfText(tx.X, tx.Y, tx.Text, false); !r.In(v.Panel) {
				t.Errorf("凡例の文 %q %v が右上の欄の外", tx.Text, r)
			}
		}
		for _, lg := range v.Legend {
			if r := rectOfText(lg.Text.X, lg.Text.Y, lg.Text.Text, false); !r.In(v.Panel) || !lg.Swatch.In(v.Panel) {
				t.Errorf("凡例 %q %v が右上の欄の外", lg.Text.Text, r)
			}
		}
		// 全体図中の矢印・ホイール・ほかのキーは渡さない（navInput が真を返し、Update はそのフレームを終える）。
		if !g.navInput(navKeys{}, 100, 100) {
			t.Error("全体図中にほかの入力を通した")
		}
		// 地図のクリック: (20, 25) の中心へ。X = 20 − 8 = 12、Y = 25 − 8 = 17 は上限 16 で止まる。閉じて Idle へ戻る。
		x, y := overviewPt(tactics.Pos{X: 20, Y: 25})
		if !g.navInput(navKeys{Left: true}, x, y) || c.OverviewOpen() || c.camera.X != 12 || c.camera.Y != 16 || c.State() != Idle {
			t.Fatalf("地図のクリック: 全体図 %v、カメラ %+v、状態 %v", c.OverviewOpen(), c.camera, c.State())
		}
		// Selected で開き、Esc・右クリック（Cancel）と M では移動せずに閉じ、選択を保つ。
		c.Click(tactics.Pos{X: 20, Y: 20})
		if c.State() != Selected {
			t.Fatalf("前提: 状態 %v", c.State())
		}
		for _, k := range []navKeys{{Cancel: true}, {Overview: true}} {
			before := c.camera
			if !g.navInput(navKeys{Overview: true}, 0, 0) || !c.OverviewOpen() {
				t.Fatal("Selected で開かない")
			}
			if !g.navInput(k, 0, 0) || c.OverviewOpen() || c.camera != before || c.State() != Selected || c.unit != 0 {
				t.Errorf("%+v で閉じる: 全体図 %v、カメラ %+v、状態 %v、選択 %d", k, c.OverviewOpen(), c.camera, c.State(), c.unit)
			}
		}
		// 「全体図」ボタンでも開き、もう一度押すと閉じる。
		if !g.navInput(navKeys{Left: true}, 750, 480) || !c.OverviewOpen() {
			t.Fatal("ボタンで開かない")
		}
		if !g.navInput(navKeys{Left: true}, 823, 503) || c.OverviewOpen() {
			t.Fatal("ボタンで閉じない")
		}
		c.Cancel()
		// 戦闘の状態（位置・HP・行動済み・履歴）は変わらない。
		if units, hist := battleSnapshot(c); !reflect.DeepEqual(units, units0) || hist != hist0 {
			t.Error("全体図の操作で戦闘の状態が変わった")
		}

		// 開かない状態: 行動のための Moved・Targeting・Canto、再生中、Over、巻き戻しの一覧、番を終える確認。
		for _, st := range []State{Moved, Targeting, HealTargeting, ItemMenu, WeaponMenu, StaffMenu, Canto, Playing, Over} {
			c.state = st
			if g.navInput(navKeys{Overview: true}, 0, 0) || c.OverviewOpen() {
				t.Errorf("%v で開いた", st)
			}
		}
		c.state = Idle
		c.confirmEnd = true
		if c.OpenOverview() || c.OverviewOpen() {
			t.Error("番を終える確認の間に開いた")
		}
		c.confirmEnd = false
		// ボタン: 再生中・Over・巻き戻しの一覧の間は描かない。行動の一覧などでは押せない形で描き、押しても受けない。
		for _, st := range []State{Playing, Over} {
			c.state = st
			if b := c.NavButtons(); b != nil {
				t.Errorf("%v でボタンを描いた %+v", st, b)
			}
		}
		c.state = Moved
		if b := c.NavButtons(); len(b) != 2 || b[0].Enabled || b[1].Enabled {
			t.Errorf("Moved のボタン %+v", b)
		}
		if !g.navInput(navKeys{Left: true}, 750, 480) || c.OverviewOpen() {
			t.Error("押せないボタンで開いた")
		}
		c.state = Idle
		c.OpenRewind()
		if b := c.NavButtons(); b != nil || g.navInput(navKeys{Overview: true, Left: true}, 750, 480) || c.OverviewOpen() {
			t.Errorf("巻き戻しの一覧でボタン %+v、全体図 %v", b, c.OverviewOpen())
		}
		c.Cancel()

		// ミニマップ (608,376)〜(736,504) のクリックは従来どおり中心だけ移す。全体図を開いたときは全体図が取り、カメラを動かさない。
		c.camera = Camera{X: 0, Y: 0, Width: 32, Height: 32}
		g.ctrl = c
		if g.navInput(navKeys{Left: true}, 608+4*24+1, 376+4*24+1) {
			t.Fatal("ミニマップのクリックを全体図の入力が取った")
		}
		g.leftClick(608+4*24+1, 376+4*24+1)
		if c.camera.X != 16 || c.camera.Y != 16 || c.State() != Idle {
			t.Fatalf("ミニマップ: カメラ %+v、状態 %v", c.camera, c.State())
		}
		c.OpenOverview()
		if !g.navInput(navKeys{Left: true}, 608+1, 376+1) || c.camera.X != 16 || c.camera.Y != 16 || !c.OverviewOpen() {
			t.Errorf("全体図中のミニマップ: カメラ %+v、全体図 %v", c.camera, c.OverviewOpen())
		}
		c.CloseOverview()

		// 16×16 のマップは 1 マス 32 px。マス (3, 5) の中心のクリックは、上限 0 で止まって (0, 0)。
		small := newIdleWith(t, func() tactics.Setup {
			s := wideSetup()
			s.Terrain = field(16, 16)
			s.Units[0].Pos = tactics.Pos{X: 10, Y: 10}
			s.Units[2].Pos = tactics.Pos{X: 15, Y: 15}
			s.Units[3].Pos = tactics.Pos{X: 11, Y: 10}
			return s
		})
		sg := navGame(small)
		sg.navInput(navKeys{Overview: true}, 0, 0)
		if v := small.MapOverview(); v.Layout.Cell != 32 || v.Units[0].Rect != image.Rect(320, 320, 352, 352) {
			t.Errorf("16×16: 1 マス %d、レオン %v", v.Layout.Cell, v.Units[0].Rect)
		}
		if !sg.navInput(navKeys{Left: true}, 3*32+16, 5*32+16) || small.OverviewOpen() || small.camera.X != 0 || small.camera.Y != 0 {
			t.Errorf("16×16 のクリック: 全体図 %v、カメラ %+v", small.OverviewOpen(), small.camera)
		}

		// 配置: M で開き、地図のクリックで配置のカメラを移して閉じる。仮配置・選んだ味方・出撃の印は変えない。Esc は配置を離れず全体図だけ閉じる。
		s := minimapDeployScenes(t)
		dg := &Game{scenes: s}
		s.deploySelected = 1
		slots := append([]int(nil), s.deploySlots...)
		marks := len(s.deployment)
		if b := s.DeployNavButtons(); !reflect.DeepEqual(b, want) {
			t.Errorf("配置のボタン %+v", b)
		}
		if !dg.navInput(navKeys{Overview: true}, 0, 0) || !s.DeployOverviewOpen() {
			t.Fatal("配置で開かない")
		}
		if v, ok := s.DeployMapOverview(); !ok || v.Layout.Cell != 16 || len(v.Units) != 3 || len(v.Hints) != 0 || len(v.Pins) != 0 {
			t.Errorf("配置の全体図: %v、1 マス %d、ユニット %d", ok, v.Layout.Cell, len(v.Units))
		}
		x, y = overviewPt(tactics.Pos{X: 4, Y: 30})
		if !dg.navInput(navKeys{Left: true}, x, y) || s.DeployOverviewOpen() || s.deployCamera.X != 0 || s.deployCamera.Y != 16 {
			t.Fatalf("配置の地図のクリック: 全体図 %v、カメラ %+v", s.DeployOverviewOpen(), s.deployCamera)
		}
		if !reflect.DeepEqual(s.deploySlots, slots) || s.deploySelected != 1 || len(s.deployment) != marks || s.Scene() != Deploy {
			t.Errorf("配置が変わった: 枠 %v、選択 %d、場面 %v", s.deploySlots, s.deploySelected, s.Scene())
		}
		dg.navInput(navKeys{Overview: true}, 0, 0)
		if !dg.navInput(navKeys{Cancel: true}, 0, 0) || s.DeployOverviewOpen() || s.Scene() != Deploy || s.deploySelected != 1 {
			t.Errorf("配置の Esc: 全体図 %v、場面 %v、選択 %d", s.DeployOverviewOpen(), s.Scene(), s.deploySelected)
		}
	})

	t.Run("TC-FR-MAP-06-08 全体図の陣営の色と危険範囲・固定の印は盤面と同じ元を使う", func(t *testing.T) {
		c := newIdleWith(t, allySetup)
		waitUnit(t, c, 1)
		c.ToggleDanger()
		c.Click(tactics.Pos{X: 31, Y: 31}) // ボスを固定する
		c.OpenOverview()
		v := c.MapOverview()
		// 陣営: 自軍は青、敵は赤、行動済みの自軍（ガルド）は盤面と同じく半分の明るさ。
		colors := map[string]any{}
		for _, u := range v.Units {
			colors[u.Unit.Name] = overviewUnitColor(u.Unit)
		}
		dark := playerBase
		dark.R, dark.G, dark.B = dark.R/2, dark.G/2, dark.B/2
		if colors["レオン"] != playerBase || colors["ガルド"] != dark || colors["ボス"] != enemyBase || colors["ファイター"] != enemyBase {
			t.Errorf("陣営の色 %v", colors)
		}
		// 危険範囲は c.Danger() のマスを 16 px で、固定はボス (31,31) のマス。
		danger := c.Danger()
		if len(danger) == 0 || len(v.Danger) != len(danger) {
			t.Fatalf("危険範囲 %d マス、全体図 %d", len(danger), len(v.Danger))
		}
		for _, r := range v.Danger {
			if !danger[tactics.Pos{X: r.Min.X / 16, Y: r.Min.Y / 16}] || r.Dx() != 16 || r.Dy() != 16 {
				t.Errorf("危険範囲の外のマス %v", r)
			}
		}
		if !reflect.DeepEqual(v.Pins, []image.Rectangle{image.Rect(496, 496, 512, 512)}) {
			t.Errorf("固定の印 %v", v.Pins)
		}
	})

	t.Run("TC-IF-01-46 全体図のモーダルはクリックとキーを先に取り、巻き戻しの一覧を最優先にする（全体図の部分）", func(t *testing.T) {
		c := newIdleWith(t, wideSetup)
		waitUnit(t, c, 1)
		g := navGame(c)
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.OpenOverview()
		cam := c.camera
		// 全体図の外（情報欄・ミニマップ・次の味方のボタン）のクリックも全体図が取り、何も変えない。
		for _, p := range []image.Point{{600, 100}, {700, 450}, {750, 530}, {300, 540}} {
			if !g.navInput(navKeys{Left: true}, p.X, p.Y) || !c.OverviewOpen() || c.camera != cam || c.State() != Selected || c.unit != 0 {
				t.Errorf("%v のクリック: 全体図 %v、カメラ %+v、状態 %v", p, c.OverviewOpen(), c.camera, c.State())
			}
		}
		// 全体図を出している間は Z を受けない（配置も同じ）。
		if c.NextAlly() || c.unit != 0 || c.camera != cam {
			t.Errorf("全体図中の Z: 選択 %d、カメラ %+v", c.unit, c.camera)
		}
		ds := minimapDeployScenes(t)
		ds.DeployOpenOverview()
		if ds.DeployOpenOverview() || !ds.DeployOverviewOpen() || ds.DeployNextAlly() || ds.deploySelected != -1 {
			t.Errorf("配置の全体図中の Z: 選択 %d", ds.deploySelected)
		}
		// 同じフレームの M と左クリックは、閉じることだけを 1 つ処理する（地図のクリックで移さない）。
		x, y := overviewPt(tactics.Pos{X: 0, Y: 0})
		g.navInput(navKeys{Overview: true, Left: true}, x, y)
		if c.OverviewOpen() || c.camera != cam {
			t.Errorf("M と左クリック: 全体図 %v、カメラ %+v", c.OverviewOpen(), c.camera)
		}
		// 同じフレームの M と Z は M だけを処理する（Z で選択とカメラを移さない）。
		g.navInput(navKeys{Overview: true, Next: true}, 0, 0)
		if !c.OverviewOpen() || c.unit != 0 || c.camera != cam {
			t.Errorf("M と Z: 全体図 %v、選択 %d、カメラ %+v", c.OverviewOpen(), c.unit, c.camera)
		}
		// T は全体図を出していても巻き戻しの一覧を開き、一覧を取り消すと全体図と Selected に戻る。
		c.OpenRewind()
		if !c.rewind.open || c.OverviewOpen() {
			t.Fatalf("T: 一覧 %v、全体図 %v", c.rewind.open, c.OverviewOpen())
		}
		c.Cancel()
		if c.rewind.open || !c.OverviewOpen() || c.State() != Selected || c.unit != 0 {
			t.Fatalf("一覧の取消し: 全体図 %v、状態 %v", c.OverviewOpen(), c.State())
		}
		// 巻き戻しに成功すると全体図を消し、Idle の復元盤面を出す。
		c.OpenRewind()
		c.RewindEnter()
		c.RewindEnter()
		if c.rewind.open || c.OverviewOpen() || c.State() != Idle || c.battle.Units()[1].Acted {
			t.Errorf("巻き戻しの成功: 一覧 %v、全体図 %v、状態 %v、ガルド行動済み %v", c.rewind.open, c.OverviewOpen(), c.State(), c.battle.Units()[1].Acted)
		}
	})

	t.Run("TC-FR-MAP-06-09 次の未行動の味方へ名簿の順に移り、Selected では選択も移す", func(t *testing.T) {
		c := newIdleWith(t, allySetup)
		g := navGame(c)
		units0, hist0 := battleSnapshot(c)
		// Idle の Z: 名簿の順にレオン (20,20) → ガルド (1,1) → ミア (28,4) → レオン。カメラの中心を移す（敵は飛ばす）。
		for _, w := range []struct {
			unit int
			cam  image.Point
		}{
			{0, image.Pt(12, 12)}, // 20 − 8
			{1, image.Pt(0, 0)},   // 1 − 8 は 0 で止まる
			{4, image.Pt(16, 0)},  // 28 − 8 = 20 は上限 16、4 − 8 は 0
			{0, image.Pt(12, 12)},
		} {
			if !g.navInput(navKeys{Next: true}, 0, 0) || c.nav.ally != w.unit || c.camera.X != w.cam.X || c.camera.Y != w.cam.Y || c.State() != Idle {
				t.Fatalf("Z: 味方 %d、カメラ %+v、状態 %v。%d と %v のはず", c.nav.ally, c.camera, c.State(), w.unit, w.cam)
			}
		}
		if units, hist := battleSnapshot(c); !reflect.DeepEqual(units, units0) || hist != hist0 {
			t.Error("Z で戦闘の状態が変わった")
		}
		// ガルドが行動済みなら飛ばす。下欄の「次の味方 Z」ボタンでも同じ。
		waitUnit(t, c, 1)
		c.camera = Camera{Width: 32, Height: 32}
		if !g.navInput(navKeys{Left: true}, 704, 520) || c.nav.ally != 4 || c.camera.X != 16 || c.camera.Y != 0 {
			t.Fatalf("ボタン: 味方 %d、カメラ %+v", c.nav.ally, c.camera)
		}
		// Selected の Z: 選んでいるレオンの次の未行動（ミア）へ選択を移し、もう一度でレオンに戻る。
		c.Click(tactics.Pos{X: 20, Y: 20})
		for _, w := range []int{4, 0} {
			if !g.navInput(navKeys{Next: true}, 0, 0) || c.State() != Selected || c.unit != w {
				t.Fatalf("Selected の Z: 状態 %v、選択 %d。%d のはず", c.State(), c.unit, w)
			}
		}
		// 全員行動済みなら移さない。
		acted := c.battle.Units()
		for i := range acted {
			acted[i].Acted = acted[i].Side == tactics.Player
		}
		if i, ok := nextUnactedAlly(acted, 0); ok {
			t.Errorf("全員行動済みで %d へ移った", i)
		}
		if i, ok := nextUnactedAlly(acted[2:4], -1); ok {
			t.Errorf("敵だけで %d へ移った", i)
		}
		// 受け付けない状態: 行動の一覧など。巻き戻しの一覧ではボタンを描かず受けない。
		c.Cancel()
		before := c.camera
		for _, st := range []State{Moved, Targeting, Canto, Playing} {
			c.state = st
			if g.navInput(navKeys{Next: true}, 0, 0) || c.camera != before {
				t.Errorf("%v で Z を受けた", st)
			}
		}
		c.state = Idle
		c.OpenRewind()
		if c.NavButtons() != nil || g.navInput(navKeys{Next: true, Left: true}, 704, 520) || c.camera != before {
			t.Error("巻き戻しの一覧で Z を受けた")
		}
		c.Cancel()

		// 配置: 名簿の順に主人公（枠 (31,31)）→ 仲間（枠 (0,0)）→ 主人公。選択とカメラの中心を移し、枠と出撃の印は変えない。
		s := minimapDeployScenes(t)
		dg := &Game{scenes: s}
		slots := append([]int(nil), s.deploySlots...)
		marks := len(s.deployment)
		for _, w := range []struct {
			sel int
			cam image.Point
		}{{0, image.Pt(16, 16)}, {1, image.Pt(0, 0)}, {0, image.Pt(16, 16)}} {
			if !dg.navInput(navKeys{Next: true}, 0, 0) || s.deploySelected != w.sel || s.deployCamera.X != w.cam.X || s.deployCamera.Y != w.cam.Y {
				t.Fatalf("配置の Z: 選択 %d、カメラ %+v。%d と %v のはず", s.deploySelected, s.deployCamera, w.sel, w.cam)
			}
		}
		if !reflect.DeepEqual(s.deploySlots, slots) || len(s.deployment) != marks {
			t.Errorf("配置の Z で枠が変わった: %v", s.deploySlots)
		}
		// 未配置の味方を優先する（名簿で先の主人公より、未配置の仲間）。
		s.deploySlots[1] = -1
		s.deploySelected, s.deployNav = -1, deployNav{}
		if !s.DeployNextAlly() || s.deploySelected != 1 {
			t.Errorf("未配置の優先: 選択 %d", s.deploySelected)
		}
		s.deploySlots[1] = slots[1]
	})

	t.Run("TC-FR-MAP-06-10 中ボタンのドラッグを 32 px のマス移動に蓄積し、上限で止め、選択には使わない", func(t *testing.T) {
		// 蓄積: 盤面の外で押し始めてもつかまない。右へ 31 px では動かず、40 px でカメラを左へ 1、端数 8 を持ち越す。
		var d boardDrag
		if dx, dy := d.update(true, true, 600, 10); dx != 0 || dy != 0 || d.active {
			t.Fatal("盤面の外でつかんだ")
		}
		d = boardDrag{}
		d.update(true, true, 100, 100)
		if dx, _ := d.update(true, false, 131, 100); dx != 0 {
			t.Fatalf("31 px で %d", dx)
		}
		if dx, _ := d.update(true, false, 140, 100); dx != -1 || d.accX != -8 {
			t.Fatalf("40 px で %d、端数 %d", dx, d.accX)
		}
		// 下から上へ 64 px で下へ 2。離すと終える。
		if _, dy := d.update(true, false, 140, 36); dy != 2 {
			t.Fatalf("上へ 64 px で %d", dy)
		}
		d.update(false, false, 140, 36)
		if d.active {
			t.Fatal("離しても終えない")
		}

		c := newIdleWith(t, wideSetup)
		g := navGame(c)
		units0, hist0 := battleSnapshot(c)
		c.camera = Camera{X: 8, Y: 8, Width: 32, Height: 32}
		// ファイター (21,20) の上で押し始めて、右へ 64 px・下へ 96 px: カメラは X 8 − 2 = 6、Y 8 − 3 = 5。敵の固定も選択もしない。
		c.SetCursor((21-8)*32+16, (20-8)*32+16)
		g.navInput(navKeys{Middle: true, MiddleStart: true}, (21-8)*32+16, (20-8)*32+16)
		g.navInput(navKeys{Middle: true}, (21-8)*32+16+64, (20-8)*32+16+96)
		if c.camera.X != 6 || c.camera.Y != 5 || c.State() != Idle || len(c.watched) != 0 {
			t.Fatalf("ドラッグ: カメラ %+v、状態 %v、固定 %v", c.camera, c.State(), c.watched)
		}
		g.navInput(navKeys{}, 0, 0)
		// 上限: 左端 (X=0) で右へ 100 px 引いても動かず、溜めない。続けて左へ 32 px で、すぐ右へ 1 動く。
		c.camera = Camera{X: 0, Y: 0, Width: 32, Height: 32}
		g.navInput(navKeys{Middle: true, MiddleStart: true}, 100, 100)
		g.navInput(navKeys{Middle: true}, 200, 100)
		if c.camera.X != 0 {
			t.Fatalf("上限を越えた: %+v", c.camera)
		}
		g.navInput(navKeys{Middle: true}, 168, 100)
		if c.camera.X != 1 {
			t.Fatalf("上限で溜めた: %+v", c.camera)
		}
		g.navInput(navKeys{}, 0, 0)
		// 右下の端 (16,16) で右下へ引いても止まる。
		c.camera = Camera{X: 16, Y: 16, Width: 32, Height: 32}
		g.navInput(navKeys{Middle: true, MiddleStart: true}, 300, 300)
		g.navInput(navKeys{Middle: true}, 100, 100)
		if c.camera.X != 16 || c.camera.Y != 16 {
			t.Fatalf("右下の上限を越えた: %+v", c.camera)
		}
		g.navInput(navKeys{}, 0, 0)
		if units, hist := battleSnapshot(c); !reflect.DeepEqual(units, units0) || hist != hist0 {
			t.Error("ドラッグで戦闘の状態が変わった")
		}
		// 行動の一覧（Moved）では動かさない（矢印・ホイールと同じ）。
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 20, Y: 20})
		if c.State() != Moved {
			t.Fatalf("前提: %v", c.State())
		}
		before := c.camera
		g.navInput(navKeys{Middle: true, MiddleStart: true}, 100, 100)
		g.navInput(navKeys{Middle: true}, 300, 300)
		g.navInput(navKeys{}, 0, 0)
		if c.camera != before || c.State() != Moved {
			t.Fatalf("Moved で動いた: %+v、状態 %v", c.camera, c.State())
		}
		// 右クリックは従来どおり取消し（Moved → Selected）で、カメラを動かさない。全体図の入力はこれを取らない。
		if g.navInput(navKeys{Cancel: true}, 100, 100) {
			t.Fatal("右クリックを盤面の移動が取った")
		}
		c.Cancel()
		if c.State() != Selected || c.camera != before {
			t.Errorf("右クリック: 状態 %v、カメラ %+v", c.State(), c.camera)
		}
		// 配置でも同じく配置のカメラだけを動かす。
		s := minimapDeployScenes(t)
		dg := &Game{scenes: s}
		dg.navInput(navKeys{Middle: true, MiddleStart: true}, 100, 100)
		dg.navInput(navKeys{Middle: true}, 164, 132)
		if s.deployCamera.X != 14 || s.deployCamera.Y != 15 || s.deploySelected != -1 {
			t.Errorf("配置のドラッグ: カメラ %+v、選択 %d", s.deployCamera, s.deploySelected)
		}
	})
}

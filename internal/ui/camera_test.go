package ui

import (
	"bytes"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestCameraCoordinates(t *testing.T) {
	t.Run("TC-FR-MAP-06-01 座標と境界と最小の寄り", func(t *testing.T) {
		for _, tt := range []struct{ ox, oy, x, y, wx, wy int }{
			{0, 0, 0, 0, 0, 0}, {7, 9, 32, 64, 8, 11}, {16, 16, 0, 0, 16, 16}, {16, 16, 511, 511, 31, 31},
		} {
			c := Camera{X: tt.ox, Y: tt.oy, Width: 32, Height: 32}
			p, ok := c.CellAt(tt.x, tt.y)
			if !ok || p != (tactics.Pos{X: tt.wx, Y: tt.wy}) {
				t.Fatalf("座標: %v %v", p, ok)
			}
		}
		c := Camera{Width: 32, Height: 32}
		for _, tt := range []struct{ p, w tactics.Pos }{
			{tactics.Pos{X: 20, Y: 5}, tactics.Pos{X: 5}}, {tactics.Pos{X: 3, Y: 3}, tactics.Pos{X: 3}}, {tactics.Pos{X: 31, Y: 31}, tactics.Pos{X: 16, Y: 16}},
		} {
			c.Focus(tt.p)
			if c.X != tt.w.X || c.Y != tt.w.Y {
				t.Fatalf("寄り: %+v", c)
			}
		}
		for _, n := range []int{1, 16, 18, 20, 24, 32} {
			c = Camera{Width: n, Height: n}
			c.Move(100, 100)
			w := max(0, n-16)
			if c.X != w || c.Y != w {
				t.Fatal(c)
			}
		}
		c = Camera{X: 16, Y: 16, Width: 32, Height: 32}
		if c.Screen(tactics.Pos{X: 31, Y: 31}) != image.Pt(480, 480) || c.Visible(tactics.Pos{}) || !c.Visible(tactics.Pos{X: 31, Y: 31}) {
			t.Fatal("既知描画位置と表示範囲")
		}
		c = Camera{Width: 32, Height: 18}
		c.Move(100, 100)
		if c.X != 16 || c.Y != 2 {
			t.Fatal("長方形", c)
		}
		c = Camera{Width: 3, Height: 2}
		for _, p := range []image.Point{image.Pt(-1, 0), image.Pt(0, -1), image.Pt(512, 0), image.Pt(0, 512), image.Pt(96, 0), image.Pt(0, 64), image.Pt(600, 20)} {
			if _, ok := c.CellAt(p.X, p.Y); ok {
				t.Fatal(p)
			}
		}
	})
}

// wideSetup は位置の既知な32四方。主人公と隣の敵、離れた仲間とボスを置く。
func wideSetup() tactics.Setup {
	s := basicSetup()
	s.Terrain = field(32, 32)
	s.Units[0].Pos = tactics.Pos{X: 20, Y: 20}
	s.Units[1].Pos = tactics.Pos{X: 1, Y: 1}
	s.Units[2].Pos = tactics.Pos{X: 31, Y: 31}
	s.Units[3].Pos = tactics.Pos{X: 21, Y: 20}
	return s
}

func TestCameraInput(t *testing.T) {
	t.Run("TC-FR-MAP-06-02 / TC-IF-01-03 / TC-IF-01-18 / TC-IF-01-19 / TC-IF-01-20 / TC-IF-01-31 / TC-IF-01-32 変換とホバーと入力の優先順", func(t *testing.T) {
		for d := 0; d < 30; d++ {
			want := d == 1 || d == 13 || d == 17 || d == 21 || d == 25 || d == 29
			if keyRepeat(d) != want {
				t.Fatal("キー保持", d)
			}
		}
		c := newIdleWith(t, wideSetup)
		c.camera = Camera{X: 7, Y: 9, Width: 32, Height: 32}
		p := tactics.Pos{X: 8, Y: 11}
		if c.camera.Screen(p) != image.Pt(32, 64) {
			t.Fatal("描画変換")
		}
		r := c.camera.MenuRects(tactics.Pos{X: 22, Y: 24}, 3)
		if r[0] != image.Rect(312, 446, 480, 468) || HitMenu(r, 313, 447) != 0 {
			t.Fatal("メニュー", r)
		}
		c.camera = Camera{X: 16, Y: 16, Width: 32, Height: 32}
		c.SetCursor(5*32+16, 4*32+16)
		if c.hoveredEnemy() != 3 {
			t.Fatal("画面座標のホバー")
		}
		panel := c.Panel(c.Camera().CellAt(176, 144))
		if panel.Unit == nil || panel.Unit.Name != "ファイター" {
			t.Fatal(panel)
		}
		c.Scroll(-100, -100)
		c.Wheel(10, 10, -1, false)
		c.Wheel(10, 10, -1, true)
		if c.camera.X != 1 || c.camera.Y != 1 {
			t.Fatal(c.camera)
		}
		c.Wheel(10, 10, 1, false)
		c.Wheel(10, 10, 1, true)
		if c.camera.X != 0 || c.camera.Y != 0 {
			t.Fatal("上方向ホイール", c.camera)
		}
		c.Wheel(10, 10, -1, false)
		c.Wheel(10, 10, -1, true)
		before := c.camera
		c.Wheel(600, 10, -1, false)
		c.Wheel(1, 512, -1, false)
		if c.camera != before {
			t.Fatal("盤外ホイール")
		}
		for _, st := range []State{Moved, ItemMenu, Playing, Over} {
			c.state = st
			c.Scroll(1, 1)
			if c.camera != before {
				t.Fatal("固定", st)
			}
		}
		c.state = Idle
		c.confirmEnd = true
		c.Scroll(1, 1)
		if c.camera != before {
			t.Fatal("番終了確認")
		}
		c.confirmEnd = false
		for _, st := range []State{Idle, Selected, Targeting, HealTargeting} {
			c.state = st
			c.Scroll(1, 1)
		}
		if c.camera.X != 5 || c.camera.Y != 5 {
			t.Fatal("許可状態", c.camera)
		}
	})
	t.Run("TC-FR-MAP-06-02 / TC-FR-MAP-03-11 予兆とポップアップの表示境界", func(t *testing.T) {
		cam := Camera{X: 16, Y: 16, Width: 32, Height: 32}
		if pt, ok := cam.Popup(tactics.Pos{X: 16, Y: 16}, 40); !ok || pt != image.Pt(0, 32) {
			t.Fatal(pt, ok)
		}
		if pt, ok := cam.Popup(tactics.Pos{X: 31, Y: 31}, 80); !ok || pt != image.Pt(432, 464) {
			t.Fatal(pt, ok)
		}
		if _, ok := cam.Popup(tactics.Pos{}, 40); ok {
			t.Fatal("画面外ポップアップ")
		}
		areas := []tactics.ReinforcementHint{{Min: tactics.Pos{X: 12, Y: 12}, Max: tactics.Pos{X: 19, Y: 19}}}
		ds, ms := cam.hintShapes(areas)
		if len(ds) == 0 {
			t.Fatal("交差部分なし")
		}
		bounds := image.Rect(0, 0, 512, 512)
		for _, r := range ds {
			if !r.In(bounds) {
				t.Fatal(r)
			}
		}
		for _, p := range ms {
			if !p.In(bounds) {
				t.Fatal(p)
			}
		}
		v := cam.hintView(HintView{Visible: true, Areas: areas, Text: SceneText{Text: hintText}})
		if v.Text.Text != "付近に気配… 画面外にも気配" || len(v.Areas) != 1 {
			t.Fatal(v)
		}
		if v.Text.X+textWidth(v.Text.Text) > ScreenWidth {
			t.Fatal("案内幅")
		}
		inside := []tactics.ReinforcementHint{{Min: tactics.Pos{X: 16, Y: 16}, Max: tactics.Pos{X: 19, Y: 19}}}
		if v := cam.hintView(HintView{Visible: true, Areas: inside, Text: SceneText{Text: hintText}}); v.Text.Text != hintText {
			t.Fatal(v)
		}
	})
}

func TestCameraFocusLifecycle(t *testing.T) {
	t.Run("TC-FR-MAP-06-03 対象選択と取消しと再生の焦点", func(t *testing.T) {
		c := newIdleWith(t, wideSetup)
		if c.camera.X != 5 || c.camera.Y != 5 {
			t.Fatal("主人公への初期化", c.camera)
		}
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.ChooseAction(tactics.Attack)
		if c.camera.X != 6 || c.camera.Y != 5 {
			t.Fatal("最初の対象", c.camera)
		}
		c.Scroll(-100, -100)
		c.Tick()
		if c.camera.X != 0 || c.camera.Y != 0 {
			t.Fatal("手動閲覧を保持")
		}
		c.Cancel()
		if c.state != Moved || c.camera.X != 5 || c.camera.Y != 5 {
			t.Fatal("仮移動先に戻る", c.camera)
		}
		units := c.battle.Units()
		units[0].Pos = tactics.Pos{X: 0, Y: 0}
		units[3].Pos = tactics.Pos{X: 31, Y: 31}
		events := []tactics.Event{{Kind: tactics.EventMove, Unit: 0, To: tactics.Pos{X: 20, Y: 5}, Units: units}, {Kind: tactics.EventPhaseStart, Turn: 2, Units: units}, {Kind: tactics.EventStrike, Unit: 0, Target: 3, Units: units}, {Kind: tactics.EventHeal, Unit: 0, Units: units}, {Kind: tactics.EventReinforce, Unit: 3, To: tactics.Pos{X: 31, Y: 2}, Units: units}, {Kind: tactics.EventFall, Unit: 3, Units: units}, {Kind: tactics.EventExp, Unit: 0, Units: units}, {Kind: tactics.EventLevelUp, Unit: 3, Units: units}}
		c.camera = Camera{Width: 32, Height: 32}
		c.play(events)
		c.SetFastForward(true)
		wants := []image.Point{image.Pt(5, 0), image.Pt(5, 0), image.Pt(16, 16), image.Pt(0, 0), image.Pt(16, 0), image.Pt(16, 16), image.Pt(0, 0), image.Pt(16, 16)}
		for i, w := range wants {
			if c.camera.X != w.X || c.camera.Y != w.Y {
				t.Fatalf("出来事%d: %+v、期待%v", i, c.camera, w)
			}
			before := c.camera
			c.Scroll(-10, -10)
			if c.camera != before {
				t.Fatal("再生中移動")
			}
			for n := 0; n < FastFramesPerEvent; n++ {
				c.Tick()
			}
		}
	})
	t.Run("TC-FR-MAP-06-03 / TC-FR-BTL-10-12 / TC-FR-BTL-10-24 / TC-FR-BTL-10-25 履歴専用カメラと復帰", func(t *testing.T) {
		c := newIdleWith(t, wideSetup)
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.Click(tactics.Pos{X: 20, Y: 20})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.camera = Camera{X: 16, Y: 16, Width: 32, Height: 32}
		original := c.camera
		units, h, left := c.battle.Units(), c.battle.History(), c.battle.RewindsLeft()
		c.OpenRewind()
		if c.Camera().X != 5 || c.Camera().Y != 5 {
			t.Fatal("before の行動者", c.Camera())
		}
		c.Wheel(10, 10, -1, true)
		if c.Camera().X != 6 || c.rewind.detailOffset != 0 {
			t.Fatal("盤面への振り分け")
		}
		c.rewind.message = strings.Repeat("長い理由", 150)
		cam := c.Camera()
		c.Wheel(600, 10, -1, false)
		if c.Camera() != cam || c.rewind.detailOffset != 1 {
			t.Fatal("右欄への振り分け")
		}
		c.RewindEnter()
		cam = c.Camera()
		c.Wheel(10, 10, -1, false)
		c.Scroll(1, 1)
		if c.Camera() != cam {
			t.Fatal("確認で固定")
		}
		c.Cancel()
		if c.Camera() != cam || c.rewind.confirm {
			t.Fatal("確認取消し")
		}
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.EndPhase()
		if !reflect.DeepEqual(units, c.battle.Units()) || !reflect.DeepEqual(h, c.battle.History()) || c.battle.RewindsLeft() != left {
			t.Fatal("閲覧中の戦闘変更")
		}
		c.Cancel()
		if c.Camera() != original {
			t.Fatal("元のカメラへ復帰")
		}
		c.OpenRewind()
		c.RewindEnter()
		c.RewindEnter()
		if c.state != Idle || c.Camera().X != 5 || c.Camera().Y != 5 || c.battle.RewindsLeft() != left-1 {
			t.Fatal("成功", c.Camera())
		}
		c.OpenRewind()
		c.fetchPreview(999999)
		if c.previewOK() || c.Camera() != (Camera{}) {
			t.Fatal("取得失敗で流用")
		}
	})
}

func TestCameraDeploy(t *testing.T) {
	t.Run("TC-FR-MAP-06-03 / TC-FR-CMP-11-05 / TC-FR-CMP-11-06 / TC-FR-CMP-11-10 配置の四隅と主人公と初期化", func(t *testing.T) {
		cat := defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend"}},
			[]defs.TerrainDef{{ID: "p", Name: "平地", Symbol: ".", Cost: 1}},
			[]defs.ClassDef{{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 20, Mov: 3}}},
			[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", MinRange: 1, MaxRange: 1}}, nil, nil,
			[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "lord", Weapon: "s", Lord: true, Level: 1}, {ID: "friend", Name: "仲間", Class: "lord", Weapon: "s", Level: 1}},
			[]defs.MapDef{{ID: "m", Rows: func() []string {
				r := make([]string, 32)
				for i := range r {
					r[i] = strings.Repeat(".", 32)
				}
				return r
			}(), Slots: []defs.Slot{{X: 31, Y: 31}, {X: 0, Y: 0}, {X: 31, Y: 0}, {X: 0, Y: 31}}, Enemies: []defs.EnemyPlacement{{Class: "lord", Weapon: "s", AI: "hold", Boss: true, X: 30, Y: 30}}}},
			[]defs.ChapterDef{{ID: "c", Map: "m"}}, nil)
		campaign, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		seeds := 0
		s := NewSceneController(Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { seeds++; return 71 }})
		s.campaign = campaign
		s.enterPrepare()
		s.enterDeploy()
		baseline := campaignBytes(t, s)
		if s.deployCamera.X != 16 || s.deployCamera.Y != 16 {
			t.Fatal("配置の主人公", s.deployCamera)
		}
		s.SetDeployCursor(14*32+16, 14*32+16)
		if s.View().Deploy.Blue[tactics.Pos{X: 30, Y: 30}] != true {
			t.Fatal("配置の敵ホバー")
		}
		s.Click(15*32+16, 15*32+16)
		s.deployCamera.Move(-100, 0)
		s.Click(16, 15*32+16)
		if !reflect.DeepEqual(s.deploySlots, []int{3, 1}) {
			t.Fatal("画面外の空き枠", s.deploySlots)
		}
		if seeds != 0 || !bytes.Equal(baseline, campaignBytes(t, s)) {
			t.Fatal("配置閲覧で戦闘を開始した")
		}
		s.startDeployedBattle()
		if seeds != 1 || s.battle.camera.X != 0 || s.battle.camera.Y != 16 || s.deployCamera != (Camera{}) {
			t.Fatal("確定主人公で戦闘開始", s.battle.camera)
		}
		s.enterPrepare()
		s.enterDeploy()
		s.deployCamera.Move(-100, -100)
		s.leaveDeploy()
		if s.deployCamera != (Camera{}) {
			t.Fatal("準備へ戻ると破棄")
		}
		s.enterDeploy()
		if s.deployCamera.X != 16 || s.deployCamera.Y != 16 {
			t.Fatal("新規配置で初期化")
		}
	})
}

func TestCameraHistorySwitchAndRandom(t *testing.T) {
	t.Run("TC-FR-MAP-06-03 / TC-FR-BTL-10-28 行変更と再現性と明示終了", func(t *testing.T) {
		c := newIdleWith(t, wideSetup)
		control := newIdleWith(t, wideSetup)
		for _, v := range []*Controller{c, control} {
			v.Click(tactics.Pos{X: 20, Y: 20})
			v.Click(tactics.Pos{X: 20, Y: 20})
			v.ChooseAction(tactics.Wait)
			playOut(v)
			v.EndPhase()
			v.EndPhase()
			playOut(v)
		}
		c.OpenRewind()
		c.rewindSelect(0)
		if c.Camera().X != 5 || c.Camera().Y != 5 {
			t.Fatal("最初の行")
		}
		c.Wheel(10, 10, -1, false)
		c.rewindKey("Down")
		if c.rewind.selected != 1 || c.Camera().X != 0 || c.Camera().Y != 0 {
			t.Fatal("上下は行の選択", c.Camera())
		}
		c.RewindEnter()
		c.Scroll(10, 10)
		c.Cancel()
		if c.rewind.selected != 1 {
			t.Fatal("確認取消しで行変更")
		}
		c.Cancel()
		c.EndPhase()
		c.EndPhase()
		control.EndPhase()
		control.EndPhase()
		if !reflect.DeepEqual(c.events, control.events) || !reflect.DeepEqual(c.battle.Units(), control.battle.Units()) || !reflect.DeepEqual(c.battle.History(), control.battle.History()) {
			t.Fatal("閲覧で乱数か戦闘が変化")
		}
	})
}

func TestCameraGuideLayout(t *testing.T) {
	t.Run("TC-FR-MAP-06-02 案内が最大の情報欄と重ならない", func(t *testing.T) {
		c := newIdleWith(t, example4Setup)
		target(t, c, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2})
		p := c.Panel(tactics.Pos{X: 2, Y: 2}, true)
		p.Unit.Level, p.Unit.Exp = 2, 40
		cam := Camera{Width: 32, Height: 32}
		hv := cam.hintView(HintView{Visible: true, Areas: []tactics.ReinforcementHint{{Min: tactics.Pos{X: 28, Y: 28}, Max: tactics.Pos{X: 31, Y: 31}}}, Text: SceneText{Text: hintText, Y: 546}})
		guides := []image.Rectangle{image.Rect(rewindLeft, cameraGuideY, rewindLeft+textWidth(cameraGuide), cameraGuideY+lineHeight), image.Rect(hv.Text.X, hv.Text.Y, hv.Text.X+textWidth(hv.Text.Text), hv.Text.Y+lineHeight)}
		for _, e := range []ForecastEmphasis{ForecastNormal, ForecastDanger, ForecastDefeat} {
			p.Emphasis = e
			v := p.View()
			for _, r := range guides {
				if !r.In(image.Rect(512, 0, 832, 576)) {
					t.Fatal("案内が欄外", r)
				}
				for _, tx := range v.Texts {
					if r.Overlaps(image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)) {
						t.Fatal("案内と情報の重なり", tx, r)
					}
				}
			}
		}
	})
}

func TestCameraHealTarget(t *testing.T) {
	t.Run("TC-FR-MAP-06-03 回復対象と取消しの焦点", func(t *testing.T) {
		c := newIdleWith(t, func() tactics.Setup {
			s := wideSetup()
			s.Units[1].Pos = tactics.Pos{X: 19, Y: 20}
			s.Units[1].Weapon = tactics.Weapon{Name: "杖", Type: tactics.Staff, Heal: 10, MaxUses: 15, MinRange: 1, MaxRange: 1}
			return s
		})
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		c.Click(tactics.Pos{X: 19, Y: 20})
		c.Click(tactics.Pos{X: 19, Y: 20})
		c.camera.Move(-100, -100)
		c.ChooseAction(tactics.Heal)
		if c.state != HealTargeting || c.camera.X != 5 || c.camera.Y != 5 {
			t.Fatal("先頭の回復対象", c.camera)
		}
		c.Scroll(-100, -100)
		c.Cancel()
		if c.state != Moved || c.camera.X != 4 || c.camera.Y != 5 {
			t.Fatal("回復取消し", c.camera)
		}
	})
}

// Package ui は、Ebitengine による描画と入力を持つ（DES-PKG-06）。
//
// 入力の状態の遷移は Controller（描画から分けた型）が持ち、Game はマウスとキーを Controller に読み替えて、
// Controller の返す値で盤面を描く（DES-UI-02）。
package ui

import (
	"errors"
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// face は文字を描く書体。bitmapfont の日本語のグリフを使う（IF-02、DES-UI-01）。
var face = text.NewGoXFace(bitmapfont.Face)

// textWidth は描画と同じ書体で最長の行を測る（DES-UI-01、12）。
func textWidth(s string) int {
	width := 0
	for _, line := range strings.Split(s, "\n") {
		w, _ := text.Measure(line, face, 0)
		width = max(width, int(w))
	}
	return width
}

// Game は ebiten.Game を実装する（DES-PKG-06）。
type Game struct {
	drawCamera Camera
	ctrl       *Controller
	scenes     *SceneController
	sheets     map[assets.Sheet]*ebiten.Image
	// hintLayer は増援の気配を不透明に描いてから薄く重ねる層（DES-UI-24）。
	hintLayer *ebiten.Image
	// shotCursor は開発用撮影で情報欄のカーソル位置を固定する。
	shotCursor *image.Point
}

// Run はタイトルからキャンペーンを始める（DES-UI-09、DES-CMD-02）。
func Run(cfg Config) error {
	scenes := NewSceneController(cfg)
	imgs, err := assets.Load()
	if err != nil {
		return err
	}
	g := &Game{scenes: scenes, sheets: map[assets.Sheet]*ebiten.Image{}}
	for s, img := range imgs {
		g.sheets[s] = ebiten.NewImageFromImage(img)
	}
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	return nil
}

// Update は場面ごとの入力を読み替える。戦闘中のキー操作と早送りは Controller へ渡す（DES-UI-09、17）。
func (g *Game) Update() error {
	s := g.scenes
	left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	cancel := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		s.rewindKey("T")
	}
	if s.Scene() == Battle || s.Scene() == GameOver {
		c := s.BattleController()
		if c.rewind.open {
			_, wheel := ebiten.Wheel()
			x, y := ebiten.CursorPosition()
			c.Wheel(x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift))
			if cancel {
				c.Cancel()
			} else if left && wheel == 0 {
				c.RewindClick(ebiten.CursorPosition())
			} else if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
				c.rewindKey("Enter")
			} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				c.rewindKey("Up")
			} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				c.rewindKey("Down")
			} else if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
				c.rewindKey("PageUp")
			} else if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
				c.rewindKey("PageDown")
			}
			s.syncRewind()
			return nil
		}
		if left && c.RewindClick(ebiten.CursorPosition()) {
			return nil
		}
	}
	if s.Scene() != Battle {
		deploy := s.Scene() == Deploy
		if deploy {
			dx, dy := cameraKeys()
			x, y := ebiten.CursorPosition()
			_, wheel := ebiten.Wheel()
			s.DeployInput(dx, dy, x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift), left && !cancel)
			if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
				s.ToggleDeployDanger()
			}
		}
		if cancel {
			s.Cancel()
		} else if left && !deploy {
			s.Click(ebiten.CursorPosition())
		} else if s.Scene() == Story && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) {
			s.Advance()
		}
	} else {
		c := s.BattleController()
		g.ctrl = c
		dx, dy := cameraKeys()
		c.Scroll(dx, dy)
		x, y := ebiten.CursorPosition()
		_, wheel := ebiten.Wheel()
		scrolled := c.Wheel(x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift)) || dx != 0 || dy != 0
		c.SetCursor(x, y)
		if c.State() == Over {
			if left {
				s.Click(ebiten.CursorPosition())
			}
		} else {
			if left && !scrolled {
				g.leftClick(ebiten.CursorPosition())
			}
			if cancel {
				c.Cancel()
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				c.EndPhase()
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
				c.ToggleDanger()
			}
			c.SetFastForward(ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyEnter))
			c.Tick()
		}
	}
	if s.Quit() {
		return ebiten.Termination
	}
	return nil
}

// leftClick は左クリックを、ボタンか盤面のマスの操作に読み替える（DES-UI-02）。
func (g *Game) leftClick(x, y int) {
	c := g.ctrl
	if labels := c.MenuLabels(); len(labels) > 0 {
		_, dest, _ := c.Selection()
		if i := HitMenu(c.Camera().MenuRects(dest, len(labels)), x, y); i >= 0 {
			c.ClickMenu(i)
		}
		return
	}
	if p, ok := c.Camera().CellAt(x, y); ok {
		c.Click(p)
	}
}

// Layout は固定の大きさの画面を使う（DES-UI-01）。
func (g *Game) Layout(_, _ int) (int, int) { return ScreenWidth, ScreenHeight }

// Draw は盤面、範囲、ユニット、危険範囲、選んだ相手の上の予測、出来事、行動の一覧、番の帯、情報欄、案内欄を描く
// （DES-UI-01、DES-UI-03、DES-UI-04、DES-UI-07、DES-UI-08）。
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(bgColor)
	if g.scenes != nil {
		// 巻き戻しの一覧・確認中は、戦闘でも敗北の選択でも、左に行動前の盤面・右に一覧を描く（DES-UI-21）。
		if sc := g.scenes.Scene(); (sc == Battle || sc == GameOver) && g.scenes.BattleController() != nil && g.scenes.BattleController().rewind.open {
			g.ctrl = g.scenes.BattleController()
			g.drawRewindPreview(screen)
			return
		}
		if g.scenes.Scene() != Battle {
			g.drawScene(screen)
			if g.scenes.Scene() == GameOver {
				g.ctrl = g.scenes.BattleController()
				g.drawRewind(screen)
			}
			return
		}
		g.ctrl = g.scenes.BattleController()
	} else if g.ctrl.rewind.open {
		g.drawRewindPreview(screen)
		return
	}
	g.drawCamera = g.ctrl.Camera()
	board := screen.SubImage(image.Rect(0, 0, MapSize, MapSize)).(*ebiten.Image)
	g.drawTerrain(board)
	g.drawOverlay(board)
	hint := g.ctrl.ReinforcementHintView()
	g.drawHint(board, hint)
	g.drawUnits(board)
	g.drawDanger(board)
	g.drawSelection(board)
	g.drawTargetLabel(board)
	g.drawEvent(board)
	g.drawMenu(board)
	g.drawOver(board)
	cx, cy := ebiten.CursorPosition()
	if g.shotCursor != nil {
		cx, cy = g.shotCursor.X, g.shotCursor.Y
	}
	g.drawInfo(screen, cx, cy)
	g.drawGuide(screen)
	g.drawHintText(screen, hint)
	g.drawRewind(screen)
}

// drawHint は増援の予兆領域の外周を黄土色の破線で、中央付近に「…」を描く（DES-UI-24）。
// 地形・範囲の塗りの上、ユニットの下に描く。不透明に描いた層を薄く重ね、重なる領域を濃くしない。
func (g *Game) drawHint(dst *ebiten.Image, v HintView) {
	if !v.Visible {
		return
	}
	if g.hintLayer == nil {
		g.hintLayer = ebiten.NewImage(MapSize, MapSize)
	}
	g.hintLayer.Clear()
	dashes, marks := g.drawCamera.hintShapes(v.Areas)
	for _, r := range dashes {
		vector.DrawFilledRect(g.hintLayer, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), hintColor, false)
	}
	for _, p := range marks {
		g.label(g.hintLayer, hintMark, p.X, p.Y, hintColor)
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(hintAlpha)
	dst.DrawImage(g.hintLayer, op)
}

// drawHintText は案内欄に「付近に気配…」を描く（DES-UI-24）。
func (g *Game) drawHintText(dst *ebiten.Image, v HintView) {
	if v.Visible {
		g.label(dst, v.Text.Text, v.Text.X, v.Text.Y, infoStyleColor(v.Text.Style))
	}
}

// drawTile はタイル t を、画面の (x, y) に 2 倍で描く（DES-UI-01、DES-UI-05）。scale が 1 未満なら暗く描く。
func (g *Game) drawTile(dst *ebiten.Image, t assets.Tile, x, y int, scale float32) {
	sheet := g.sheets[t.Sheet]
	if sheet == nil {
		return
	}
	img := sheet.SubImage(t.Rect()).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(CellSize/assets.TileSize, CellSize/assets.TileSize)
	op.GeoM.Translate(float64(x), float64(y))
	if scale < 1 {
		op.ColorScale.Scale(scale, scale, scale, 1)
	}
	dst.DrawImage(img, op)
}

// drawTerrain は地形を描く。どのマスにも草の下地を描き、平地でなければタイルを重ねる（DES-UI-01、DES-UI-05）。
func (g *Game) drawTerrain(dst *ebiten.Image) {
	g.drawBoardTerrain(dst, g.ctrl.Battle())
}

// terrainSource は戦闘と盤面参照値に共通する地形の参照口（DES-UI-21、DES-RWD-05）。
type terrainSource interface {
	Width() int
	Height() int
	TerrainAt(tactics.Pos) tactics.Terrain
}

// drawBoardTerrain は地形を、戦闘と巻き戻しのプレビューで同じ絵で描く（DES-UI-05、DES-UI-21）。
func (g *Game) drawBoardTerrain(dst *ebiten.Image, b terrainSource) {
	for y := g.drawCamera.Y; y < min(b.Height(), g.drawCamera.Y+16); y++ {
		for x := g.drawCamera.X; x < min(b.Width(), g.drawCamera.X+16); x++ {
			pt := g.drawCamera.Screen(tactics.Pos{X: x, Y: y})
			px, py := pt.X, pt.Y
			g.drawTile(dst, assets.Grass, px, py, 1)
			if t, ok := terrainTile(b.TerrainAt(tactics.Pos{X: x, Y: y})); ok {
				g.drawTile(dst, t, px, py, 1)
			}
		}
	}
}

// drawOverlay は移動を青、攻撃の射程を赤、杖の射程を緑で重ねる（DES-UI-03、DES-UI-18）。敵の範囲もこれで描く（DES-UI-08）。
func (g *Game) drawOverlay(dst *ebiten.Image) {
	blue, red := g.ctrl.Overlay()
	for p := range blue {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), CellSize, CellSize, blueOverlay, false)
	}
	clr := redOverlay
	if g.ctrl.RangeStyle() == StyleGood {
		clr = greenOverlay
	}
	for p := range red {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), CellSize, CellSize, clr, false)
	}
}

// drawDanger は、敵の危険範囲を橙の枠でマスごとに描く（DES-UI-08）。移動と攻撃を分けない。
// 自軍のいるマスも見えるように、ユニットの上に描く。選択や攻撃の相手の枠（drawSelection）は、この上に描く。
func (g *Game) drawDanger(dst *ebiten.Image) {
	for p := range g.ctrl.Danger() {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.StrokeRect(dst, float32(g.drawCamera.Screen(p).X)+3, float32(g.drawCamera.Screen(p).Y)+3, CellSize-6, CellSize-6, 2, dangerColor, false)
	}
}

// drawUnits はユニットを描く（DES-UI-03）。陣営で色の違う下地、HP の棒、ボスの印を描き、行動済みの自軍は暗くする。
func (g *Game) drawUnits(dst *ebiten.Image) {
	g.drawUnitList(dst, g.ctrl.Units())
}

// drawUnitList は戦闘と配置で同じユニットの絵を描く（DES-UI-03、DES-UI-20）。
func (g *Game) drawUnitList(dst *ebiten.Image, units []tactics.Unit) {
	for _, u := range units {
		if !u.Alive || !g.drawCamera.Visible(u.Pos) {
			continue
		}
		pt := g.drawCamera.Screen(u.Pos)
		px, py := float32(pt.X), float32(pt.Y)
		base := playerBase
		if u.Side == tactics.Enemy {
			base = enemyBase
		}
		scale := float32(1)
		if u.Side == tactics.Player && u.Acted {
			scale = actedScale
			base.R, base.G, base.B = base.R/2, base.G/2, base.B/2
		}
		vector.DrawFilledRect(dst, px+1, py+1, CellSize-2, CellSize-2, base, false)
		if u.Boss {
			vector.StrokeRect(dst, px+1, py+1, CellSize-2, CellSize-2, 2, gold, false)
		}
		g.drawTile(dst, unitTile(u), int(px), int(py), scale)
		// HP の棒。
		w := float32(CellSize - 4)
		vector.DrawFilledRect(dst, px+2, py+CellSize-5, w, 4, hpBack, false)
		if u.Stats.HP > 0 {
			vector.DrawFilledRect(dst, px+2, py+CellSize-5, w*float32(u.HP)/float32(u.Stats.HP), 4, hpFront, false)
		}
		if u.Boss {
			g.label(dst, "★", int(px)+CellSize-12, int(py)-1, gold)
		}
	}
}

// drawSelection は選んだ自軍と攻撃・回復の相手を枠で示す（DES-UI-02、DES-UI-18）。
func (g *Game) drawSelection(dst *ebiten.Image) {
	c := g.ctrl
	unit, dest, _ := c.Selection()
	units := c.Units()
	switch c.State() {
	case Selected:
		g.strokeCell(dst, units[unit].Pos, cursorColor)
	case Moved, ItemMenu, Canto, WeaponMenu, StaffMenu:
		g.strokeCell(dst, dest, cursorColor)
	case Targeting, HealTargeting:
		clr := cursorColor
		if c.TargetView().Danger {
			clr = badColor
		}
		g.strokeCell(dst, dest, clr)
		targets := c.Targets()
		if c.State() == HealTargeting {
			targets = c.HealTargets()
		}
		for _, t := range targets {
			g.strokeCell(dst, units[t].Pos, c.targetFrameColor(t))
		}
	}
}

func (g *Game) strokeCell(dst *ebiten.Image, p tactics.Pos, clr color.Color) {
	if !g.drawCamera.Visible(p) {
		return
	}
	vector.StrokeRect(dst, float32(g.drawCamera.Screen(p).X)+1, float32(g.drawCamera.Screen(p).Y)+1, CellSize-2, CellSize-2, 2, clr, false)
}

// drawTargetLabel は、選んだ相手のマスの上に「ダメージ×回数 命中率%」を、出来事の文字と同じ形（popup）で出す（DES-UI-07）。
// 相手を選ぶ前は出さない。
func (g *Game) drawTargetLabel(dst *ebiten.Image) {
	if v := g.ctrl.TargetView(); v.Visible {
		g.popup(dst, v.Text, v.Pos, infoStyleColor(v.Style))
	}
}

// drawEvent は再生中の出来事を描く（DES-UI-03、DES-UI-04、DES-UI-16）。
// PhaseStart は番の帯、Strike は当たり・外れ・ダメージ・必殺・大盾、Heal は回復量を文字で出す。
func (g *Game) drawEvent(dst *ebiten.Image) {
	e, _, ok := g.ctrl.Event()
	if !ok {
		return
	}
	if v := GrowthEventView(e); v.Visible {
		if v.Band {
			vector.DrawFilledRect(dst, 0, MapSize/2-32, MapSize, 64, bandColor, false)
			g.centerLabel(dst, v.Text, MapSize/2-lineHeight/2, gold)
		} else {
			g.popup(dst, v.Text, v.Pos, gold)
		}
		return
	}
	switch e.Kind {
	case tactics.EventPhaseStart:
		vector.DrawFilledRect(dst, 0, MapSize/2-32, MapSize, 64, bandColor, false)
		g.bigLabel(dst, BandText(e), MapSize/2-32+20, white)
	case tactics.EventStrike:
		g.strokeCell(dst, e.Units[e.Unit].Pos, cursorColor)
		clr := strikeHit
		if !e.Hit {
			clr = strikeMiss
		}
		g.popup(dst, StrikeText(e), e.Units[e.Target].Pos, clr)
	case tactics.EventHeal:
		g.popup(dst, "+"+strconv.Itoa(e.Amount), e.Units[e.Unit].Pos, healTextClr)
	case tactics.EventMove, tactics.EventWait, tactics.EventReinforce, tactics.EventFall:
		g.strokeCell(dst, e.Units[e.Unit].Pos, cursorColor)
	}
}

// popup は、マス p の上に文字を出す（DES-UI-04）。盤面の外に出ないように寄せる。
func (g *Game) popup(dst *ebiten.Image, s string, p tactics.Pos, clr color.Color) {
	w := textWidth(s)
	pt, ok := g.drawCamera.Popup(p, w)
	if !ok {
		return
	}
	x, y := pt.X, pt.Y
	vector.DrawFilledRect(dst, float32(x-2), float32(y-1), float32(w+4), 15, bandColor, false)
	g.label(dst, s, x, y, clr)
}

// drawMenu は、行動の一覧かアイテムの一覧を、移動先の横に描く（DES-UI-02）。
func (g *Game) drawMenu(dst *ebiten.Image) {
	labels := g.ctrl.MenuLabels()
	if len(labels) == 0 {
		return
	}
	_, dest, _ := g.ctrl.Selection()
	for i, r := range g.ctrl.Camera().MenuRects(dest, len(labels)) {
		vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), menuColor, false)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, menuBorder, false)
		g.label(dst, labels[i], r.Min.X+8, r.Min.Y+4, white)
	}
}

// drawOver は戦闘終了の帯と次への操作を描く（DES-UI-17）。
func (g *Game) drawOver(dst *ebiten.Image) {
	if g.ctrl.State() != Over {
		return
	}
	vector.DrawFilledRect(dst, 0, 0, MapSize, MapSize, overShade, false)
	r := BattleEndRect()
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bandColor, false)
	var clr color.Color = gold
	if g.ctrl.Battle().Result == tactics.Lost {
		clr = muted
	}
	g.bigLabel(dst, ResultText(g.ctrl.Battle().Result), MapSize/2-40, clr)
	g.centerLabel(dst, "左クリック: 次へ", MapSize/2+12, white)
}

// drawInfo は右の情報欄を描く（DES-UI-01、DES-UI-06、DES-UI-07、DES-BTL-07）。攻撃の相手を選んでいれば予測（武器の絵・相性の印・
// 「撃破」「危険」・表）を上に、その下にカーソルのあるユニットの情報（絵・能力の棒・武器の絵・スキル）と地形を出す。
// (cx, cy) はカーソルの画面の座標で、マスは CellAt で引く。
func (g *Game) drawInfo(dst *ebiten.Image, cx, cy int) {
	vector.DrawFilledRect(dst, MapSize, 0, InfoWidth, ScreenHeight, panelColor, false)
	p := g.ctrl.Panel(g.ctrl.Camera().CellAt(cx, cy))
	// 予測・ユニット・地形は、位置を決めた InfoView のとおりに描く（DES-UI-06、DES-UI-07）。
	v := p.View()
	g.drawInfoView(dst, v)
	g.label(dst, cameraGuide, rewindLeft, cameraGuideY, muted)
}

// drawInfoView は戦闘と準備で情報欄の描画を共有する（DES-UI-06、13）。
func (g *Game) drawInfoView(dst *ebiten.Image, v InfoView) {
	if b := v.Banner; b != nil {
		vector.DrawFilledRect(dst, float32(b.Rect.Min.X), float32(b.Rect.Min.Y), float32(b.Rect.Dx()), float32(b.Rect.Dy()), infoStyleColor(b.Style), false)
	}
	for _, ic := range v.Icons {
		g.drawTileSize(dst, ic.Tile, ic.X, ic.Y, ic.Size)
	}
	for _, b := range v.Bars {
		if b.Frame > 0 {
			vector.DrawFilledRect(dst, float32(b.X), float32(b.Y), float32(b.Frame), barHeight, hpBack, false)
		}
		clr := statBarColor
		if b.Frame > 0 {
			clr = hpFront
		}
		vector.DrawFilledRect(dst, float32(b.X), float32(b.Y), float32(b.Fill), barHeight, clr, false)
	}
	for _, t := range v.Texts {
		if t.Big {
			g.scaledLabel(dst, t.Text, t.X, t.Y, 2, infoStyleColor(t.Style))
			continue
		}
		g.label(dst, t.Text, t.X, t.Y, infoStyleColor(t.Style))
	}
}

// drawTileSize はタイル t を、画面の (x, y) に size × size の大きさで描く（DES-UI-06）。
func (g *Game) drawTileSize(dst *ebiten.Image, t assets.Tile, x, y, size int) {
	sheet := g.sheets[t.Sheet]
	if sheet == nil {
		return
	}
	img := sheet.SubImage(t.Rect()).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(size)/assets.TileSize, float64(size)/assets.TileSize)
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}

// drawGuide は案内欄を描く（DES-UI-01）。
func (g *Game) drawGuide(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, 0, MapSize, MapSize, GuideHeight, panelColor, false)
	lines := g.ctrl.Guide()
	if g.scenes != nil {
		lines = g.scenes.BattleGuide()
	}
	for i, line := range lines {
		if i >= GuideHeight/lineHeight {
			break
		}
		clr := white
		if i > 0 {
			clr = muted
		}
		g.label(dst, line, 8, MapSize+2+i*lineHeight, clr)
	}
}

// label は文字を (x, y) を左上にして、影をつけて描く（IF-02。bitmapfont の日本語のグリフ）。
func (g *Game) label(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)+1, float64(y)+1)
	op.ColorScale.ScaleWithColor(textShadow)
	text.Draw(dst, s, face, op)
	op = &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// scaledLabel は文字を (x, y) を左上にして、scale 倍の大きさで影をつけて描く（相性の印。DES-UI-07）。
func (g *Game) scaledLabel(dst *ebiten.Image, s string, x, y int, scale float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x)+scale, float64(y)+scale)
	op.ColorScale.ScaleWithColor(textShadow)
	text.Draw(dst, s, face, op)
	op = &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// bigLabel は文字を 2 倍の大きさで、盤面の横の中央に描く。
func (g *Game) bigLabel(dst *ebiten.Image, s string, y int, clr color.Color) {
	w := textWidth(s) * 2
	op := &text.DrawOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(MapSize/2-w/2), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// centerLabel は文字を盤面の横の中央に描く。
func (g *Game) centerLabel(dst *ebiten.Image, s string, y int, clr color.Color) {
	g.label(dst, s, MapSize/2-textWidth(s)/2, y, clr)
}

// drawScene は表示用の値を描く（DES-UI-09〜12、15）。
func (g *Game) drawScene(dst *ebiten.Image) {
	g.drawSceneView(dst, g.scenes.View())
}

// drawSceneView は配置・文・ボタンを描く（DES-UI-20、DES-UI-19、DES-UI-09）。
func (g *Game) drawSceneView(dst *ebiten.Image, v SceneView) {
	if v.Deploy != nil {
		g.drawDeploy(dst, *v.Deploy)
	}
	if v.Info != nil {
		vector.DrawFilledRect(dst, MapSize, 0, InfoWidth, ScreenHeight, panelColor, false)
		g.drawInfoView(dst, *v.Info)
	}
	for _, r := range v.Rows {
		if r.Selected {
			vector.DrawFilledRect(dst, float32(r.Rect.Min.X), float32(r.Rect.Min.Y), float32(r.Rect.Dx()), float32(r.Rect.Dy()), playerBase, false)
		}
	}
	for _, t := range v.Texts {
		if t.Big {
			g.scaledLabel(dst, t.Text, t.X, t.Y, 2, infoStyleColor(t.Style))
		} else {
			g.label(dst, t.Text, t.X, t.Y, infoStyleColor(t.Style))
		}
	}
	for _, ic := range v.Icons {
		g.drawTile(dst, ic.Tile, ic.X, ic.Y, 1)
	}
	for _, b := range v.Buttons {
		r := b.Rect
		fill, border := menuColor, menuBorder
		if !b.Enabled {
			fill, border = bgColor, muted
		}
		vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), fill, false)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, border, false)
		clr := white
		if !b.Enabled {
			clr = muted
		}
		g.label(dst, b.Label, r.Min.X+12, r.Min.Y+(r.Dy()-lineHeight)/2, clr)
	}
}

// drawDeploy は地形→出撃下地→個別範囲→ユニット→出撃枠→危険枠→選択枠を重ねる（DES-UI-23）。
func (g *Game) drawDeploy(dst *ebiten.Image, d DeployView) {
	g.drawCamera = g.scenes.deployCamera
	dst = dst.SubImage(image.Rect(0, 0, MapSize, MapSize)).(*ebiten.Image)
	for y, row := range d.Terrain {
		for x, t := range row {
			p := tactics.Pos{X: x, Y: y}
			if !g.drawCamera.Visible(p) {
				continue
			}
			pt := g.drawCamera.Screen(p)
			g.drawTile(dst, assets.Grass, pt.X, pt.Y, 1)
			if tile, ok := terrainTile(t); ok {
				g.drawTile(dst, tile, pt.X, pt.Y, 1)
			}
		}
	}
	for _, p := range d.Slots {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), CellSize, CellSize, blueOverlay, false)
	}
	for p := range d.Blue {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), CellSize, CellSize, blueOverlay, false)
	}
	for p := range d.Red {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), CellSize, CellSize, redOverlay, false)
	}
	g.drawUnitList(dst, d.Units)
	for _, p := range d.Slots {
		if !g.drawCamera.Visible(p) {
			continue
		}
		g.strokeCell(dst, p, menuBorder)
	}
	for p := range d.Danger {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.StrokeRect(dst, float32(g.drawCamera.Screen(p).X)+3, float32(g.drawCamera.Screen(p).Y)+3, CellSize-6, CellSize-6, 2, dangerColor, false)
	}
	if d.Selected >= 0 {
		g.strokeCell(dst, d.Units[d.Selected].Pos, cursorColor)
	}
}

// drawRewind は一覧を閉じているときの巻き戻しの入口を描く（DES-UI-19）。
func (g *Game) drawRewind(dst *ebiten.Image) {
	g.drawSceneView(dst, g.ctrl.RewindView())
}

// drawRewindPreview は左に選んだ行動の前の盤面、下にプレビューの案内、右欄に一覧・確認を描く（DES-UI-21）。
// 範囲・仮移動・予測・行動の一覧・再生と勝敗の帯は描かない。ユニットは戦闘と同じ drawUnitList で描く。
func (g *Game) drawRewindPreview(dst *ebiten.Image) {
	v := g.ctrl.RewindBoard()
	g.drawCamera = g.ctrl.Camera()
	board := dst.SubImage(image.Rect(0, 0, MapSize, MapSize)).(*ebiten.Image)
	if v.Available {
		g.drawBoardTerrain(board, v.Board)
		g.drawHint(board, v.Hint)
		g.drawUnitList(board, v.Units)
		if v.HasFrame {
			g.strokeCell(board, v.Frame, cursorColor)
		}
	}
	if v.Notice != "" {
		vector.DrawFilledRect(dst, 0, MapSize/2-32, MapSize, 64, bandColor, false)
		g.centerLabel(dst, v.Notice, MapSize/2-lineHeight/2, white)
	}
	vector.DrawFilledRect(dst, 0, MapSize, MapSize, GuideHeight, panelColor, false)
	for i, line := range v.Lines {
		clr := white
		if i > 0 {
			clr = muted
		}
		g.label(dst, line, 8, MapSize+2+i*lineHeight, clr)
	}
	vector.DrawFilledRect(dst, MapSize, 0, InfoWidth, ScreenHeight, panelColor, false)
	g.drawSceneView(dst, g.ctrl.RewindView())
}

// cameraKeys はフレーム数だけで矢印のリピートを決める（DES-UI-25）。
func cameraKeys() (dx, dy int) {
	if keyRepeat(inpututil.KeyPressDuration(ebiten.KeyArrowLeft)) {
		dx--
	}
	if keyRepeat(inpututil.KeyPressDuration(ebiten.KeyArrowRight)) {
		dx++
	}
	if keyRepeat(inpututil.KeyPressDuration(ebiten.KeyArrowUp)) {
		dy--
	}
	if keyRepeat(inpututil.KeyPressDuration(ebiten.KeyArrowDown)) {
		dy++
	}
	return
}

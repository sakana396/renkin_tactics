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
	// drag は中ボタンで盤面をつかむ操作の途中の値（DES-UI-30）。
	drag boardDrag
	// canvas は盤面を持たない場面を、窓の中央へ置くために一度描く層（DES-NG-10）。既定の表示では使わない。
	canvas *ebiten.Image
	// sound は効果音と BGM の再生（DES-NG-09）。zero 値なら無音。
	sound soundGlue
}

// Run はタイトルからキャンペーンを始める（DES-UI-09、DES-CMD-02）。
func Run(cfg Config) error {
	useConfigMetrics(cfg)
	scenes := NewSceneController(cfg)
	imgs, err := assets.Load()
	if err != nil {
		return err
	}
	g := &Game{scenes: scenes, sheets: map[assets.Sheet]*ebiten.Image{}, sound: newSoundGlue(cfg)}
	for s, img := range imgs {
		g.sheets[s] = ebiten.NewImageFromImage(img)
	}
	setWindow(cfg)
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	return nil
}

// Update は 1 フレームの処理のあとに、音を同期する（DES-NG-09）。
func (g *Game) Update() error {
	err := g.update()
	if g.sound.sound != nil {
		g.syncSound(readSoundInput(g.scenes.Scene()))
	}
	return err
}

// update は場面ごとの入力を読み替える。戦闘中のキー操作と早送りは Controller へ渡す（DES-UI-09、17）。
func (g *Game) update() error {
	s := g.scenes
	left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	cancel := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	x, y := g.cursor()
	if g.priorityInput(priorityKeys{
		Rewind: inpututil.IsKeyJustPressed(ebiten.KeyT), Tab: inpututil.IsKeyJustPressed(ebiten.KeyTab),
		Close: cancel, Detail: inpututil.IsKeyJustPressed(ebiten.KeyI), Click: left,
		Danger: inpututil.IsKeyJustPressed(ebiten.KeyQ),
		X:      x, Y: y,
	}) {
		return nil
	}
	if s.Scene() == Battle || s.Scene() == GameOver {
		c := s.BattleController()
		if c.rewind.open {
			_, wheel := ebiten.Wheel()
			x, y := g.cursor()
			c.Wheel(x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift))
			if cancel {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
					c.battleKey("RightClick")
				} else {
					c.battleKey("Esc")
				}
			} else if left && wheel == 0 {
				// 入力の判定はミニマップ → 右欄 → 盤面の順（DES-UI-28）。
				if x, y := g.cursor(); !c.MinimapClick(x, y) {
					c.RewindClick(x, y)
				}
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
		if left && c.RewindClick(g.cursor()) {
			return nil
		}
	}
	// 全体図・次の味方・中ボタンのドラッグ（DES-UI-30）。全体図を出している間はほかの入力を渡さない。
	if x, y := g.cursor(); g.navInput(readNavKeys(left, cancel), x, y) {
		return nil
	}
	if s.Scene() != Battle {
		deploy := s.Scene() == Deploy
		if deploy {
			dx, dy := cameraKeys()
			x, y := g.cursor()
			_, wheel := ebiten.Wheel()
			s.DeployInput(dx, dy, x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift), left && !cancel)
			if inpututil.IsKeyJustPressed(ebiten.KeyQ) && !left {
				s.ToggleDeployDanger()
			}
		}
		x, y := g.cursor()
		g.sceneInput(sceneKeys{
			Tab: inpututil.IsKeyJustPressed(ebiten.KeyTab), Left: left, Cancel: cancel, Deploy: deploy,
			Advance: inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace),
			Right:   inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
			Up:      inpututil.IsKeyJustPressed(ebiten.KeyArrowUp), Down: inpututil.IsKeyJustPressed(ebiten.KeyArrowDown),
			Enter: inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		}, x, y)
	} else {
		c := s.BattleController()
		g.ctrl = c
		dx, dy := cameraKeys()
		c.Scroll(dx, dy)
		x, y := g.cursor()
		_, wheel := ebiten.Wheel()
		scrolled := c.Wheel(x, y, wheel, ebiten.IsKeyPressed(ebiten.KeyShift)) || dx != 0 || dy != 0
		c.SetCursor(x, y)
		if c.State() == Over {
			if left {
				s.Click(g.cursor())
			}
		} else {
			if left && !scrolled {
				g.leftClick(g.cursor())
			}
			if cancel {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
					c.battleKey("RightClick")
				} else {
					c.battleKey("Esc")
				}
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyA) {
				c.battleKey("A")
			} else if inpututil.IsKeyJustPressed(ebiten.KeyD) {
				c.battleKey("D")
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

// priorityKeys は Game.Update が先に処理するキーとカーソルの写し（DES-IT07 3.1 節）。
type priorityKeys struct {
	Rewind, Tab, Close, Detail, Click, Danger bool
	X, Y                                      int
}

// priorityInput は最優先の入力が成立したフレームを消費する（DES-IT07 3.1 節）。
func (g *Game) priorityInput(k priorityKeys) bool {
	s := g.scenes
	if k.Rewind && (s.Scene() == Battle || s.Scene() == GameOver) && s.BattleController() != nil {
		s.rewindKey("T")
		if s.BattleController().rewind.open {
			return true
		}
	}
	if s.Scene() == Deploy {
		if s.detailOpen {
			s.DeployDetailInput(k.Tab, k.Close || k.Detail)
			return true
		}
		if s.deployNav.overview {
			return false
		}
		if k.Detail {
			s.SetDeployCursor(k.X, k.Y)
			if s.OpenDeployDetail() {
				return true
			}
		}
	}
	if s.Scene() == Battle && s.BattleController() != nil {
		c := s.BattleController()
		if c.detailOpen {
			c.DetailInput(k.Tab, k.Close || k.Detail)
			return true
		}
		if c.OverviewOpen() {
			return false
		}
		if k.Detail {
			c.SetCursor(k.X, k.Y)
			if c.OpenDetail() {
				return true
			}
		}
	}
	return false
}

// sceneKeys は戦闘以外の場面へ渡す、そのフレームの入力（DES-UI-09）。Deploy は配置の入力を先に渡したこと。
type sceneKeys struct {
	Tab, Left, Cancel, Deploy, Advance bool
	// Right は右クリック、Up・Down・Enter は準備（高解像度）のキー（DES-NG-11）。
	Right, Up, Down, Enter bool
}

// sceneInput は戦闘以外の場面の Tab・取消し・左クリック・送りを場面へ渡す（DES-UI-09、DES-UI-32）。
// 準備の実入力は Game.Update → sceneInput → SceneController.Click・Cancel の経路を通る。
func (g *Game) sceneInput(k sceneKeys, x, y int) {
	s := g.scenes
	if k.Tab {
		s.ToggleInfoPage()
	}
	if s.Scene() == Prepare && s.cfg.HiRes {
		switch {
		case k.Cancel:
			if !(k.Right && s.rightClickPrepare(x, y)) {
				s.Cancel()
			}
		case k.Left:
			s.Click(x, y)
		case k.Up:
			s.movePrepareMember(-1)
		case k.Down:
			s.movePrepareMember(1)
		case k.Enter:
			s.Advance()
		}
		return
	}
	if k.Cancel {
		s.Cancel()
	} else if k.Left && !k.Deploy {
		s.Click(x, y)
	} else if s.Scene() == Story && k.Advance {
		s.Advance()
	}
}

// leftClick は左クリックを、ボタンか盤面のマスの操作に読み替える（DES-UI-02）。
// 判定は詳しい画面 → 予測の窓 → ミニマップ → 範囲の「消す」 → 情報欄 → 盤面の順（DES-UI-28、DES-UI-29、DES-IT07 3.1 節）。
// 詳しい画面を出している間は、どこをクリックしても何もしない（閉じるのは I・Esc・右クリック）。
func (g *Game) leftClick(x, y int) {
	c := g.ctrl
	if c.detailOpen || c.ForecastClick(x, y) || c.MinimapClick(x, y) || c.RangeClick(x, y) || c.InfoClick(x, y) {
		return
	}
	if labels := c.MenuLabels(); len(labels) > 0 {
		_, dest, _ := c.Selection()
		if i := HitMenu(c.Camera().MenuRects(dest, len(labels)), x, y); i >= 0 {
			c.ClickMenu(i)
			return
		}
		// 再移動の状態では、「終える」のボタンの外のクリックを盤面のマスの操作にする（青いマスで再移動。DES-IT06 12.3 節、E-98）。
		// ほかの一覧の状態では、ボタンの外のクリックを受け付けない。
		if c.State() != Canto {
			return
		}
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
			if g.scenes.Scene() == Deploy {
				g.drawDeployNav(screen)
			}
			if g.scenes.Scene() == Deploy && g.scenes.detailOpen {
				v := g.scenes.deployView()
				if g.scenes.detailUnit >= 0 && g.scenes.detailUnit < len(v.Deploy.Units) {
					g.drawUnitDetail(screen, v.Deploy.Units[g.scenes.detailUnit], g.scenes.detailSection)
				}
			}
			return
		}
		g.ctrl = g.scenes.BattleController()
	} else if g.ctrl.rewind.open {
		g.drawRewindPreview(screen)
		return
	}
	// 盤面の敵範囲とミニマップで共有する。フレームをまたいで保持しない。
	danger := g.ctrl.Danger()
	g.drawCamera = g.ctrl.Camera()
	board := screen.SubImage(image.Rect(0, 0, MapSize, MapSize)).(*ebiten.Image)
	g.drawTerrain(board)
	g.drawOverlay(board)
	hint := g.ctrl.ReinforcementHintView()
	g.drawHint(board, hint)
	for _, l := range battleLayers {
		switch l {
		case layerUnits:
			g.drawUnits(board)
		case layerEnemyRanges:
			g.drawEnemyRanges(board, g.ctrl.enemyRangeLayersWithDanger(danger))
		case layerZOC:
			g.drawZOC(board, g.ctrl.battle.BoardView())
		case layerEquipMarks:
			g.drawEquipMarks(board, g.ctrl.Units())
		case layerWatchedPins:
			g.drawWatchedPins(board, g.ctrl.WatchedPins(), g.ctrl.Units())
		}
	}
	g.drawSelection(board)
	g.drawTargetLabel(board)
	g.drawBombLabels(board)
	g.drawEvent(board)
	g.drawMenu(board)
	g.drawOver(board)
	cx, cy := ebiten.CursorPosition()
	if g.shotCursor != nil {
		cx, cy = g.shotCursor.X, g.shotCursor.Y
	}
	g.drawInfo(screen, cx, cy)
	if w, ok := g.ctrl.ForecastWindow(); ok {
		g.drawForecastWindow(screen, w)
	}
	g.drawMinimap(screen, g.ctrl.minimapWithDanger(danger))
	g.drawGuide(screen)
	g.drawHintText(screen, hint)
	g.drawRewind(screen)
	g.drawBattleNav(screen)
	if g.ctrl.detailOpen {
		units := g.ctrl.Units()
		if g.ctrl.detailUnit >= 0 && g.ctrl.detailUnit < len(units) {
			g.drawUnitDetail(screen, units[g.ctrl.detailUnit], g.ctrl.detailSection)
		}
	}
}

// drawUnitDetail は盤面と右欄 (0,0)〜(832,512) に詳しい画面を、下 (0,512)〜(832,576) に操作の案内を描く（DES-UI-31）。
func (g *Game) drawUnitDetail(dst *ebiten.Image, u tactics.Unit, section DetailSection) {
	fillImageRect(dst, image.Rect(0, 0, ScreenWidth, MapSize), panelColor)
	// 高解像度の表示では、既定の窓の幅の配置を窓の中央へ寄せる（DES-NG-10）。
	g.drawInfoView(dst, DetailView(u, section).translate(sceneOffset().X, 0))
	fillImageRect(dst, image.Rect(0, MapSize, ScreenWidth, ScreenHeight), bgColor)
	g.label(dst, detailGuide, 16+sceneOffset().X, ScreenHeight-40, muted)
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
	op.GeoM.Scale(float64(CellSize)/assets.TileSize, float64(CellSize)/assets.TileSize)
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
			g.drawTerrainCell(dst, b.TerrainAt(tactics.Pos{X: x, Y: y}), x, y, px, py)
		}
	}
}

// hiRes は、高解像度の表示（新作）かを返す（DES-NG-10）。
func (g *Game) hiRes() bool { return g != nil && g.scenes != nil && g.scenes.cfg.HiRes }

// drawTerrainCell は、マス (x, y)（画面の左上 (px, py)）の地形 t を描く（DES-UI-05、DES-NG-10 フェーズ 2）。
// 高解像度の表示では、差し替えの PNG か仮の絵があればそれだけで 1 マスを埋める。
// なければ（既存の作品・知らない地形）、草の下地に地形のタイルを重ねる今の描き方。
func (g *Game) drawTerrainCell(dst *ebiten.Image, t tactics.Terrain, x, y, px, py int) {
	if g.hiRes() {
		if eb := g.artStore().TerrainImage(t.Name, x, y); eb != nil {
			drawArtIn(dst, eb, image.Rect(px, py, px+CellSize, py+CellSize), 1)
			return
		}
	}
	g.drawTile(dst, assets.Grass, px, py, 1)
	if tile, ok := terrainTile(t); ok {
		g.drawTile(dst, tile, px, py, 1)
	}
}

// drawOverlay は移動を青、攻撃の射程を赤、杖の射程を緑で重ねる（DES-UI-03、DES-UI-18）。敵の範囲もこれで描く（DES-UI-08）。
func (g *Game) drawOverlay(dst *ebiten.Image) {
	blue, red := g.ctrl.Overlay()
	for p := range blue {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), float32(CellSize), float32(CellSize), blueOverlay, false)
	}
	clr := redOverlay
	if g.ctrl.RangeStyle() == StyleGood {
		clr = greenOverlay
	}
	for p := range red {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), float32(CellSize), float32(CellSize), clr, false)
	}
	// 爆弾の爆発の範囲は、投げられる範囲の上に濃い橙で重ねる（DES-NG-05）。
	for p := range g.ctrl.BlastOverlay() {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), float32(CellSize), float32(CellSize), blastOverlay, false)
	}
}

// drawZOC は、盤面 board の重騎士の ZOC の角印を陣営色の細い線で描く（DES-UI-29、DES-IT07 3.8 節）。
// 形と位置は zocMarkLines で決める。描く順は battleLayers・deployLayers・rewindLayers に従う。
func (g *Game) drawZOC(dst *ebiten.Image, board tactics.BoardView) {
	for _, l := range zocMarkLines(board, g.drawCamera) {
		clr := color.Color(playerBase)
		if l.Side == tactics.Enemy {
			clr = enemyBase
		}
		vector.StrokeLine(dst, float32(l.From.X), float32(l.From.Y), float32(l.To.X), float32(l.To.Y), float32(scaled(2)), clr, false)
	}
}

// drawUnits はユニットを描く（DES-UI-03）。陣営で色の違う下地、HP の棒、ボスの印を描き、行動済みの自軍は暗くする。
func (g *Game) drawUnits(dst *ebiten.Image) {
	g.drawUnitList(dst, g.ctrl.Units())
}

// drawUnitList は戦闘と配置で同じユニットの絵を描く（DES-UI-03、DES-UI-20）。装備の印は drawEquipMarks で範囲の線の上に描く（DES-UI-33）。
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
		vector.DrawFilledRect(dst, px+1, py+1, float32(CellSize-2), float32(CellSize-2), base, false)
		if u.Boss {
			vector.StrokeRect(dst, px+1, py+1, float32(CellSize-2), float32(CellSize-2), float32(scaled(2)), gold, false)
		}
		// 画像があれば、下地の上に重ねる。なければ今のタイル（DES-NG-08）。
		if !g.drawUnitArt(dst, artKeyOf(u), int(px), int(py), scale) {
			g.drawTile(dst, unitTile(u), int(px), int(py), scale)
		}
		// HP の棒。右下の装備の印と重ねない（DES-UI-33）。
		bar := unitHPBarRect(pt)
		fillImageRect(dst, bar, hpBack)
		if u.Stats.HP > 0 {
			w := float32(bar.Dx()) * float32(u.HP) / float32(u.Stats.HP)
			vector.DrawFilledRect(dst, float32(bar.Min.X), float32(bar.Min.Y), w, float32(bar.Dy()), hpFront, false)
		}
		if u.Boss {
			g.label(dst, "★", int(px)+CellSize-12, int(py)-1, gold)
		}
		// 小隊長・配下・崩れの印は絵の左上に置く（DES-UI-35、DES-IT07 3.6 節）。
		g.drawCommandMark(dst, CommandMarkOf(u), CommandMarkRect(pt))
	}
}

// drawCommandMark は、箱 r に指揮の印を描く（DES-UI-35）。縁を 1 px 大きく先に描き、その上に金または灰で形を描く。
// 菱形は横の線を重ねて描き、ドットの絵のように角をはっきりさせる。崩れは小点に縁の色の割れ目を入れる。
func (g *Game) drawCommandMark(dst *ebiten.Image, m CommandMark, r image.Rectangle) {
	shape := m.shape()
	if shape == shapeNone {
		return
	}
	clr := m.color()
	cx, cy := float32(r.Min.X)+float32(r.Dx())/2, float32(r.Min.Y)+float32(r.Dy())/2
	switch shape {
	case shapeDiamond:
		diamond := func(radius int, c color.Color) {
			for dy := -radius; dy <= radius; dy++ {
				half := radius - max(dy, -dy)
				vector.DrawFilledRect(dst, cx-float32(half)-0.5, cy+float32(dy)-0.5, float32(2*half+1), 1, c, false)
			}
		}
		radius := r.Dx() / 2
		diamond(radius, commandMarkEdge)
		diamond(radius-1, clr)
	case shapeDot, shapeCrackedDot:
		radius := float32(r.Dx()) / 4
		vector.DrawFilledCircle(dst, cx, cy, radius+1, commandMarkEdge, true)
		vector.DrawFilledCircle(dst, cx, cy, radius, clr, true)
		if shape == shapeCrackedDot {
			// 割れ目: 右上から中央を通って左下へ折れる線。太さは描く箱の大きさに比例させる（右欄の印は 48 px でも変えない）。
			lw := crackLineWidth(r)
			vector.StrokeLine(dst, cx+radius, cy-radius, cx, cy, lw, commandMarkEdge, true)
			vector.StrokeLine(dst, cx, cy, cx-radius/2, cy+radius/4, lw, commandMarkEdge, true)
			vector.StrokeLine(dst, cx-radius/2, cy+radius/4, cx-radius, cy+radius, lw, commandMarkEdge, true)
		}
	}
}

// crackLineWidth は、箱 r に描く崩れの印の割れ目の線の太さ（px）を返す（DES-UI-35、DES-NG-10）。
// 12 px の箱で 1.5 px とし、箱の大きさに比例させる。盤面の印の箱は scaled で大きくなり、右欄の印の箱は 12 px のまま。
func crackLineWidth(r image.Rectangle) float32 {
	return 1.5 * float32(r.Dx()) / commandMarkSize
}

// drawEquipMarks は、ユニットの右下に装備の印を、暗い下地の上へ種類の絵で描く（DES-UI-33）。
// 敵の範囲の線の上に描き、固定のピン（右上）と同じく線に隠れないようにする。
func (g *Game) drawEquipMarks(dst *ebiten.Image, units []tactics.Unit) {
	for _, m := range equipMarks(units, g.drawCamera) {
		fillImageRect(dst, m.Rect, hpBack)
		g.drawTileSize(dst, m.Tile, m.Rect.Min.X, m.Rect.Min.Y, m.Rect.Dx())
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
	case BombTargeting:
		g.strokeCell(dst, dest, cursorColor)
		// 爆心は金の枠、巻き込まれる自軍（自分を含む）は赤い枠にする（DES-NG-05）。
		if f, ok := c.BombForecast(); ok {
			g.strokeCell(dst, f.Center, gold)
			for _, l := range c.BombLabels() {
				if l.Ally {
					g.strokeCell(dst, l.Pos, badColor)
				}
			}
		}
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
	vector.StrokeRect(dst, float32(g.drawCamera.Screen(p).X)+1, float32(g.drawCamera.Screen(p).Y)+1, float32(CellSize-2), float32(CellSize-2), 2, clr, false)
}

// drawBombLabels は、爆発に巻き込まれるユニットの上に予想のダメージを、出来事の文字と同じ形（popup）で出す（DES-NG-05）。
func (g *Game) drawBombLabels(dst *ebiten.Image) {
	for _, l := range g.ctrl.BombLabels() {
		g.popup(dst, l.Text, l.Pos, infoStyleColor(l.Style))
	}
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
			vector.DrawFilledRect(dst, 0, float32(MapSize/2-32), float32(MapSize), 64, bandColor, false)
			g.centerLabel(dst, v.Text, MapSize/2-lineHeight/2, gold)
		} else {
			g.popup(dst, v.Text, v.Pos, gold)
		}
		return
	}
	switch e.Kind {
	case tactics.EventPhaseStart:
		vector.DrawFilledRect(dst, 0, float32(MapSize/2-32), float32(MapSize), 64, bandColor, false)
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
	case tactics.EventBlast:
		g.strokeCell(dst, e.To, gold)
	case tactics.EventBlastHit:
		g.popup(dst, strconv.Itoa(e.Amount)+" ダメージ", e.Units[e.Unit].Pos, strikeHit)
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
	vector.DrawFilledRect(dst, 0, 0, float32(MapSize), float32(MapSize), overShade, false)
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
	vector.DrawFilledRect(dst, float32(MapSize), 0, InfoWidth, float32(ScreenHeight), panelColor, false)
	w, forecast := g.ctrl.ForecastWindow()
	if !forecast || !w.Fallback {
		v := g.ctrl.BattleInfoView(cx, cy)
		g.drawInfoView(dst, v)
		if r, ok := g.ctrl.cursorNameMark(v); ok {
			g.drawNameMark(dst, r)
		}
		g.drawRangeStatus(dst, g.ctrl.RangeStatusIn(v))
		g.drawDetailButton(dst, g.ctrl.DetailButton())
	}
	g.label(dst, cameraGuide, rewindLeft, cameraGuideY, muted)
}

// drawDetailButton は操作行の「詳細 I」を「消す」と同じ形で描く（DES-UI-31）。
func (g *Game) drawDetailButton(dst *ebiten.Image, b SceneButton) {
	fill, border, text := menuColor, menuBorder, white
	if !b.Enabled {
		fill, border, text = bgColor, muted, muted
	}
	fillImageRect(dst, b.Rect, fill)
	vector.StrokeRect(dst, float32(b.Rect.Min.X), float32(b.Rect.Min.Y), float32(b.Rect.Dx()), float32(b.Rect.Dy()), 1, border, false)
	g.label(dst, b.Label, b.Rect.Min.X+(b.Rect.Dx()-textWidth(b.Label))/2, b.Rect.Min.Y+(b.Rect.Dy()-lineHeight)/2, text)
}

// drawForecastWindow は盤面または右欄に共通の予測を描く（DES-UI-31）。
func (g *Game) drawForecastWindow(dst *ebiten.Image, w ForecastWindow) {
	fillImageRect(dst, w.Rect, panelColor)
	vector.StrokeRect(dst, float32(w.Rect.Min.X), float32(w.Rect.Min.Y), float32(w.Rect.Dx()), float32(w.Rect.Dy()), 2, menuBorder, false)
	g.drawInfoView(dst, w.View)
}

// drawInfoView は戦闘と準備で情報欄の描画を共有する（DES-UI-06、13）。
func (g *Game) drawInfoView(dst *ebiten.Image, v InfoView) {
	if b := v.Banner; b != nil {
		vector.DrawFilledRect(dst, float32(b.Rect.Min.X), float32(b.Rect.Min.Y), float32(b.Rect.Dx()), float32(b.Rect.Dy()), infoStyleColor(b.Style), false)
	}
	for _, ic := range v.Icons {
		g.drawInfoIcon(dst, ic)
	}
	for _, m := range v.Marks {
		g.drawCommandMark(dst, m.Mark, m.Rect)
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
	// ページの印は選んでいるものを塗り、右に Tab キーの案内を出す（DES-UI-28）。
	for _, tab := range v.Tabs {
		r := tab.Rect
		fill := menuColor
		if tab.Selected {
			fill = playerBase
		}
		vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), fill, false)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, menuBorder, false)
		g.label(dst, tab.Label, r.Min.X+(r.Dx()-textWidth(tab.Label))/2, r.Min.Y+(r.Dy()-lineHeight)/2, white)
	}
	if len(v.Tabs) > 0 {
		// 案内は印と一緒にずらす（準備では右列へずらしてある。DES-UI-32）。
		t := InfoTabGuide()
		d := v.Tabs[0].Rect.Min.Sub(InfoTabRects()[0].Min)
		g.label(dst, t.Text, t.X+d.X, t.Y+d.Y, infoStyleColor(t.Style))
	}
}

// minimapTerrainColor はミニマップの地形の塗り（DES-UI-28）。
func minimapTerrainColor(t MinimapTerrain) color.Color {
	switch t {
	case MinimapForest:
		return color.NRGBA{40, 110, 50, 255}
	case MinimapMountain:
		return color.NRGBA{140, 110, 80, 255}
	case MinimapFort:
		return color.NRGBA{170, 170, 160, 255}
	case MinimapWall:
		return color.NRGBA{60, 60, 70, 255}
	}
	return color.NRGBA{100, 160, 80, 255}
}

// drawMinimap は右下の欄にミニマップを描く（DES-UI-28）。地形 → 予兆 → ユニット → 表示範囲の白い枠の順に重ね、
// 余白に本体の枠・見出し・凡例を置く。
func (g *Game) drawMinimap(dst *ebiten.Image, v MinimapView) {
	r := MinimapRect()
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bgColor, false)
	fill := func(r image.Rectangle, clr color.Color) {
		vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), clr, false)
	}
	stroke := func(r image.Rectangle, clr color.Color) {
		vector.StrokeRect(dst, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx())-1, float32(r.Dy())-1, 1, clr, false)
	}
	if v.Available {
		stroke(v.Layout.Area.Inset(-1), menuBorder)
		for _, c := range v.Cells {
			fill(c.Rect, minimapTerrainColor(c.Terrain))
		}
		drawMinimapDanger(dst, v.Danger)
		for _, h := range v.Hints {
			stroke(h, hintColor)
		}
		for _, u := range v.Units {
			clr := playerBase
			if u.Side == tactics.Enemy {
				clr = enemyBase
			}
			fill(u.Rect, clr)
		}
		drawMinimapPins(dst, v.Pins)
		stroke(v.View, white)
	}
	for _, l := range v.Legend {
		switch l.Mark {
		case MarkPlayer:
			fill(l.Swatch, playerBase)
		case MarkEnemy:
			fill(l.Swatch, enemyBase)
		case MarkView:
			stroke(l.Swatch, white)
		case MarkHint:
			stroke(l.Swatch, hintColor)
		}
		g.label(dst, l.Text.Text, l.Text.X, l.Text.Y, infoStyleColor(l.Text.Style))
	}
	for _, t := range v.Texts {
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
	vector.DrawFilledRect(dst, 0, float32(MapSize), float32(MapSize), GuideHeight, panelColor, false)
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
	if g.scenes.fullWindowPrepare() {
		g.drawPrepareHiRes(dst, g.scenes.prepareHiResView())
		return
	}
	v := g.scenes.View()
	off := sceneOffset()
	// 敗北では、巻き戻しの入口も配置の層の中に描く（一覧を開いているときは Draw が先に別の画面を描く）。
	var entry *SceneView
	if g.scenes.Scene() == GameOver && g.scenes.BattleController() != nil {
		g.ctrl = g.scenes.BattleController()
		ev := g.ctrl.RewindView()
		entry = &ev
	}
	if off == (image.Point{}) || !onCanvas(g.scenes.Scene()) {
		g.drawSceneView(dst, v)
		if entry != nil {
			g.drawSceneView(dst, *entry)
		}
		return
	}
	// 高解像度の表示では、盤面を持たない場面を既定の窓の大きさの層に描き、窓の中央へ置く。背景だけは窓の全体を覆う（DES-NG-10）。
	g.drawSceneBackdrop(dst, v)
	if g.canvas == nil {
		g.canvas = ebiten.NewImage(sceneCanvasWidth, sceneCanvasHeight)
	}
	g.canvas.Clear()
	g.drawSceneBody(g.canvas, v)
	if entry != nil {
		g.drawSceneBody(g.canvas, *entry)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(off.X), float64(off.Y))
	dst.DrawImage(g.canvas, op)
}

// cursor は、場面の配置の座標でのカーソルの位置を返す。盤面を持たない場面は窓の中央に置くので、その分を引く（DES-NG-10）。
func (g *Game) cursor() (x, y int) {
	x, y = ebiten.CursorPosition()
	if g.scenes == nil {
		return x, y
	}
	c := g.scenes.BattleController()
	if g.scenes.fullWindowPrepare() {
		return x, y
	}
	return sceneCursor(x, y, g.scenes.Scene(), c != nil && c.rewind.open)
}

// drawSceneView は配置・文・ボタンを描く（DES-UI-20、DES-UI-19、DES-UI-09）。
func (g *Game) drawSceneView(dst *ebiten.Image, v SceneView) {
	// 背景は最初に描く（DES-NG-08）。
	g.drawSceneBackdrop(dst, v)
	g.drawSceneBody(dst, v)
}

// drawSceneBody は drawSceneView のうち、背景より上に描くもの（DES-NG-10）。
func (g *Game) drawSceneBody(dst *ebiten.Image, v SceneView) {
	if v.Deploy != nil {
		g.drawDeploy(dst, *v.Deploy)
	}
	if v.Info != nil {
		// 下地は既定では右の情報欄の全体。準備の名簿では右列だけを塗る（DES-UI-32）。
		area := v.InfoArea
		if area.Empty() {
			area = image.Rect(MapSize, 0, ScreenWidth, ScreenHeight)
		}
		vector.DrawFilledRect(dst, float32(area.Min.X), float32(area.Min.Y), float32(area.Dx()), float32(area.Dy()), panelColor, false)
		g.drawInfoView(dst, *v.Info)
		if v.Deploy != nil && !v.Deploy.NameMark.Empty() {
			g.drawNameMark(dst, v.Deploy.NameMark)
		}
	}
	// 準備の「名簿」「持ち物」のタブは、選んでいるものを塗る（DES-UI-32）。
	for _, tab := range v.Tabs {
		r := tab.Rect
		fill := menuColor
		if tab.Selected {
			fill = playerBase
		}
		vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), fill, false)
		vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, menuBorder, false)
		g.label(dst, tab.Label, r.Min.X+(r.Dx()-textWidth(tab.Label))/2, r.Min.Y+(r.Dy()-lineHeight)/2, white)
	}
	if v.Minimap != nil {
		g.drawMinimap(dst, *v.Minimap)
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
	g.drawScenePortrait(dst, v)
	for _, ic := range v.Icons {
		g.drawSceneIcon(dst, ic)
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
			g.drawTerrainCell(dst, t, x, y, pt.X, pt.Y)
		}
	}
	for _, p := range d.Slots {
		if !g.drawCamera.Visible(p) {
			continue
		}
		vector.DrawFilledRect(dst, float32(g.drawCamera.Screen(p).X), float32(g.drawCamera.Screen(p).Y), float32(CellSize), float32(CellSize), blueOverlay, false)
	}
	g.drawUnitList(dst, d.Units)
	for _, p := range d.Slots {
		if !g.drawCamera.Visible(p) {
			continue
		}
		g.strokeCell(dst, p, menuBorder)
	}
	// 全敵の塗りとカーソルの敵の線は、戦闘と同じ層の順に重ね、その上に ZOC の角印と装備の印を描く（DES-UI-29、DES-UI-33）。
	for _, l := range deployLayers {
		switch l {
		case layerEnemyRanges:
			g.drawEnemyRanges(dst, d.RangeLayers())
		case layerZOC:
			g.drawZOC(dst, d.Board)
		case layerEquipMarks:
			g.drawEquipMarks(dst, d.Units)
		case layerFrame:
			if d.Selected >= 0 {
				g.strokeCell(dst, d.Units[d.Selected].Pos, cursorColor)
			}
		}
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
		for _, l := range rewindLayers {
			switch l {
			case layerUnits:
				g.drawUnitList(board, v.Units)
			case layerZOC:
				g.drawZOC(board, v.Board)
			case layerEquipMarks:
				g.drawEquipMarks(board, v.Units)
			case layerFrame:
				if v.HasFrame {
					g.strokeCell(board, v.Frame, cursorColor)
				}
			}
		}
	}
	if v.Notice != "" {
		vector.DrawFilledRect(dst, 0, float32(MapSize/2-32), float32(MapSize), 64, bandColor, false)
		g.centerLabel(dst, v.Notice, MapSize/2-lineHeight/2, white)
	}
	vector.DrawFilledRect(dst, 0, float32(MapSize), float32(MapSize), GuideHeight, panelColor, false)
	for i, line := range v.Lines {
		clr := white
		if i > 0 {
			clr = muted
		}
		g.label(dst, line, 8, MapSize+2+i*lineHeight, clr)
	}
	vector.DrawFilledRect(dst, float32(MapSize), 0, InfoWidth, float32(ScreenHeight), panelColor, false)
	g.drawSceneView(dst, g.ctrl.RewindView())
	g.drawMinimap(dst, g.ctrl.Minimap())
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

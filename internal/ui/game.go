// Package ui は、Ebitengine による描画と入力を持つ（DES-PKG-06）。
//
// 入力の状態の遷移は Controller（描画から分けた型）が持ち、Game はマウスとキーを Controller に読み替えて、
// Controller の返す値で盤面を描く（DES-UI-02）。
package ui

import (
	"errors"
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
	ctrl   *Controller
	scenes *SceneController
	sheets map[assets.Sheet]*ebiten.Image
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
	if s.Scene() != Battle {
		if cancel {
			s.Cancel()
		} else if left {
			s.Click(ebiten.CursorPosition())
		} else if s.Scene() == Story && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) {
			s.Advance()
		}
	} else {
		c := s.BattleController()
		g.ctrl = c
		c.SetCursor(ebiten.CursorPosition())
		if c.State() == Over {
			if left {
				s.Click(ebiten.CursorPosition())
			}
		} else {
			if left {
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
		if i := HitMenu(MenuRects(dest, len(labels)), x, y); i >= 0 {
			c.ClickMenu(i)
		}
		return
	}
	if p, ok := CellAt(x, y); ok {
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
		if g.scenes.Scene() != Battle {
			g.drawScene(screen)
			return
		}
		g.ctrl = g.scenes.BattleController()
	}
	g.drawTerrain(screen)
	g.drawOverlay(screen)
	g.drawUnits(screen)
	g.drawDanger(screen)
	g.drawSelection(screen)
	g.drawTargetLabel(screen)
	g.drawEvent(screen)
	g.drawMenu(screen)
	g.drawOver(screen)
	cx, cy := ebiten.CursorPosition()
	g.drawInfo(screen, cx, cy)
	g.drawGuide(screen)
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
	b := g.ctrl.Battle()
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			px, py := x*CellSize, y*CellSize
			g.drawTile(dst, assets.Grass, px, py, 1)
			if t, ok := terrainTile(b.TerrainAt(tactics.Pos{X: x, Y: y})); ok {
				g.drawTile(dst, t, px, py, 1)
			}
		}
	}
}

// drawOverlay は、移動できるマスを青、攻撃できるマスを赤で重ねる（DES-UI-03）。敵の範囲もこれで描く（DES-UI-08）。
func (g *Game) drawOverlay(dst *ebiten.Image) {
	blue, red := g.ctrl.Overlay()
	for p := range blue {
		vector.DrawFilledRect(dst, float32(p.X*CellSize), float32(p.Y*CellSize), CellSize, CellSize, blueOverlay, false)
	}
	for p := range red {
		vector.DrawFilledRect(dst, float32(p.X*CellSize), float32(p.Y*CellSize), CellSize, CellSize, redOverlay, false)
	}
}

// drawDanger は、敵の危険範囲を橙の枠でマスごとに描く（DES-UI-08）。移動と攻撃を分けない。
// 自軍のいるマスも見えるように、ユニットの上に描く。選択や攻撃の相手の枠（drawSelection）は、この上に描く。
func (g *Game) drawDanger(dst *ebiten.Image) {
	for p := range g.ctrl.Danger() {
		vector.StrokeRect(dst, float32(p.X*CellSize)+3, float32(p.Y*CellSize)+3, CellSize-6, CellSize-6, 2, dangerColor, false)
	}
}

// drawUnits はユニットを描く（DES-UI-03）。陣営で色の違う下地、HP の棒、ボスの印を描き、行動済みの自軍は暗くする。
func (g *Game) drawUnits(dst *ebiten.Image) {
	for _, u := range g.ctrl.Units() {
		if !u.Alive {
			continue
		}
		px, py := float32(u.Pos.X*CellSize), float32(u.Pos.Y*CellSize)
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

// drawSelection は、選んだ自軍と攻撃の相手を枠で示す（DES-UI-02）。
func (g *Game) drawSelection(dst *ebiten.Image) {
	c := g.ctrl
	unit, dest, target := c.Selection()
	units := c.Units()
	switch c.State() {
	case Selected:
		strokeCell(dst, units[unit].Pos, cursorColor)
	case Moved, ItemMenu:
		strokeCell(dst, dest, cursorColor)
	case Targeting:
		clr := cursorColor
		if c.TargetView().Danger {
			clr = badColor
		}
		strokeCell(dst, dest, clr)
		for _, t := range c.Targets() {
			clr := targetColor
			if t == target {
				clr = cursorColor
			}
			strokeCell(dst, units[t].Pos, clr)
		}
	}
}

func strokeCell(dst *ebiten.Image, p tactics.Pos, clr color.Color) {
	vector.StrokeRect(dst, float32(p.X*CellSize)+1, float32(p.Y*CellSize)+1, CellSize-2, CellSize-2, 2, clr, false)
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
		strokeCell(dst, e.Units[e.Unit].Pos, cursorColor)
		clr := strikeHit
		if !e.Hit {
			clr = strikeMiss
		}
		g.popup(dst, StrikeText(e), e.Units[e.Target].Pos, clr)
	case tactics.EventHeal:
		g.popup(dst, "+"+strconv.Itoa(e.Amount), e.Units[e.Unit].Pos, healTextClr)
	case tactics.EventMove, tactics.EventWait, tactics.EventReinforce, tactics.EventFall:
		strokeCell(dst, e.Units[e.Unit].Pos, cursorColor)
	}
}

// popup は、マス p の上に文字を出す（DES-UI-04）。盤面の外に出ないように寄せる。
func (g *Game) popup(dst *ebiten.Image, s string, p tactics.Pos, clr color.Color) {
	w := textWidth(s)
	x := p.X*CellSize + CellSize/2 - w/2
	x = max(0, min(x, MapSize-w))
	y := p.Y*CellSize - 16
	if y < 0 {
		y = p.Y*CellSize + CellSize
	}
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
	for i, r := range MenuRects(dest, len(labels)) {
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
	p := g.ctrl.Panel(CellAt(cx, cy))
	// 予測・ユニット・地形は、位置を決めた InfoView のとおりに描く（DES-UI-06、DES-UI-07）。
	v := p.View()
	g.drawInfoView(dst, v)
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
	v := g.scenes.View()
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
		g.label(dst, b.Label, r.Min.X+12, r.Min.Y+10, clr)
	}
}

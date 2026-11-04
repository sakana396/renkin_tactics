package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// navKeys は、盤面の移動（DES-UI-30）に使う、そのフレームの入力。
// Overview は M、Next は Z、Left は左クリック、Cancel は右クリックか Esc の押し始め。
// Middle は中ボタンを押していること、MiddleStart はその押し始め。
type navKeys struct {
	Overview, Next, Left, Cancel bool
	Middle, MiddleStart          bool
}

// readNavKeys は Ebitengine の入力から navKeys を作る（DES-UI-30）。
func readNavKeys(left, cancel bool) navKeys {
	return navKeys{
		Overview:    inpututil.IsKeyJustPressed(ebiten.KeyM),
		Next:        inpututil.IsKeyJustPressed(ebiten.KeyZ),
		Left:        left,
		Cancel:      cancel,
		Middle:      ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle),
		MiddleStart: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle),
	}
}

// navInput は、戦闘（勝敗の確定後と巻き戻しの一覧・確認を除く）と配置の、全体図・次の味方・中ボタンのドラッグを受ける（DES-UI-30）。
// 真を返したら、そのフレームのほかの入力（矢印・ホイール・クリック・キー）を渡さない。
// 判定は「全体図のモーダル → 全体図・次の味方のボタン → M → Z」の順で、同じフレームでは 1 つだけを処理する（DES-IT07 3.1 節）。
// 中ボタンのドラッグはカメラだけを動かし、ほかの入力は止めない。
func (g *Game) navInput(k navKeys, x, y int) bool {
	s := g.scenes
	switch s.Scene() {
	case Battle:
		c := s.BattleController()
		if c == nil || c.rewind.open || c.State() == Over {
			g.drag = boardDrag{}
			return false
		}
		if c.OverviewOpen() {
			// 全体図が入力を取る。中の矢印・ホイール・ほかのキーは無効（DES-IT07 3.1 節）。
			g.drag = boardDrag{}
			switch {
			case k.Cancel || k.Overview:
				c.CloseOverview()
			case k.Left:
				c.OverviewClick(x, y)
			}
			return true
		}
		if k.Left {
			buttons := c.NavButtons()
			// 押せない形で描いたボタンは、各操作が受け付けない。
			if i := navButtonAt(buttons, x, y); i == 0 {
				c.OpenOverview()
				return true
			} else if i == 1 {
				c.NextAlly()
				return true
			}
		}
		if k.Overview && c.OpenOverview() {
			g.drag = boardDrag{}
			return true
		}
		if k.Next && c.NextAlly() {
			return true
		}
		g.dragBoard(k, x, y, func() Camera { return c.camera }, c.Scroll)
	case Deploy:
		if s.DeployOverviewOpen() {
			g.drag = boardDrag{}
			switch {
			case k.Cancel || k.Overview:
				s.DeployCloseOverview()
			case k.Left:
				s.DeployOverviewClick(x, y)
			}
			return true
		}
		if k.Left {
			buttons := s.DeployNavButtons()
			if i := navButtonAt(buttons, x, y); i == 0 {
				s.DeployOpenOverview()
				return true
			} else if i == 1 {
				s.DeployNextAlly()
				return true
			}
		}
		if k.Overview && s.DeployOpenOverview() {
			g.drag = boardDrag{}
			return true
		}
		if k.Next && s.DeployNextAlly() {
			return true
		}
		g.dragBoard(k, x, y, func() Camera { return s.deployCamera }, func(dx, dy int) { s.DeployScroll(dx, dy) })
	default:
		g.drag = boardDrag{}
	}
	return false
}

// dragBoard は中ボタンのドラッグを、矢印と同じ盤面の移動（scroll）に渡す（DES-UI-30）。
// 動かせない状態（行動の一覧など）や上限で動かなかった分の蓄積は捨てる。
func (g *Game) dragBoard(k navKeys, x, y int, camera func() Camera, scroll func(dx, dy int)) {
	dx, dy := g.drag.update(k.Middle, k.MiddleStart, x, y)
	if dx == 0 && dy == 0 {
		return
	}
	before := camera()
	scroll(dx, dy)
	after := camera()
	g.drag.settle(dx, dy, after.X-before.X, after.Y-before.Y)
}

// drawBattleNav は戦闘の全体図（開いていれば）と、盤面の移動のボタンを描く（DES-UI-30）。
func (g *Game) drawBattleNav(dst *ebiten.Image) {
	c := g.ctrl
	if c.OverviewOpen() {
		g.drawOverview(dst, c.MapOverview())
	}
	g.drawSceneView(dst, SceneView{Buttons: c.NavButtons()})
}

// drawDeployNav は配置の全体図（開いていれば）と、盤面の移動のボタンを描く（DES-UI-30）。
func (g *Game) drawDeployNav(dst *ebiten.Image) {
	s := g.scenes
	if s.DeployOverviewOpen() {
		if v, ok := s.DeployMapOverview(); ok {
			g.drawOverview(dst, v)
		}
	}
	g.drawSceneView(dst, SceneView{Buttons: s.DeployNavButtons()})
}

// drawOverview は盤面の (0,0)〜(512,512) に全体図を、右上の欄に凡例を描く（DES-UI-30）。右下のミニマップはそのまま残す。
// 地形 → 予兆 → 全敵の危険範囲 → ユニット（陣営の色の下地と絵）→ 固定した敵の金の縁 → 表示範囲の白い枠の順に重ねる。
func (g *Game) drawOverview(dst *ebiten.Image, v MapOverviewView) {
	fillImageRect(dst, image.Rect(0, 0, MapSize, MapSize), bgColor)
	l := v.Layout
	for _, c := range v.Cells {
		g.drawTileSize(dst, assets.Grass, c.Rect.Min.X, c.Rect.Min.Y, l.Cell)
		if t, ok := terrainTile(c.Terrain); ok {
			g.drawTileSize(dst, t, c.Rect.Min.X, c.Rect.Min.Y, l.Cell)
		}
	}
	for _, h := range v.Hints {
		strokeImageRect(dst, h, 1, hintColor)
	}
	for _, r := range v.Danger {
		fillImageRect(dst, r, dangerFill)
	}
	for _, u := range v.Units {
		fillImageRect(dst, u.Rect, overviewUnitColor(u.Unit))
		g.drawTileSize(dst, unitTile(u.Unit), u.Rect.Min.X, u.Rect.Min.Y, l.Cell)
		if u.Unit.Boss {
			strokeImageRect(dst, u.Rect, 1, gold)
		}
		if !u.Equip.Empty() {
			// 装備の種類の色は、明るい地形と下地の上でも読めるよう暗い縁を敷く（マスの中に収める）。
			fillImageRect(dst, u.Equip.Inset(-1).Intersect(u.Rect), textShadow)
			fillImageRect(dst, u.Equip, u.EquipColor)
		}
	}
	for _, r := range v.Pins {
		strokeImageRect(dst, r.Inset(-1), 2, gold)
	}
	if !v.View.Empty() {
		strokeImageRect(dst, v.View, 2, white)
	}
	fillImageRect(dst, v.Panel, panelColor)
	for _, lg := range v.Legend {
		switch lg.Mark {
		case MarkPlayer:
			fillImageRect(dst, lg.Swatch, playerBase)
		case MarkActed:
			fillImageRect(dst, lg.Swatch, overviewUnitColor(tactics.Unit{Side: tactics.Player, Acted: true}))
		case MarkEnemy:
			fillImageRect(dst, lg.Swatch, enemyBase)
		case MarkView:
			strokeImageRect(dst, lg.Swatch, 1, white)
		case MarkHint:
			strokeImageRect(dst, lg.Swatch, 1, hintColor)
		case MarkDanger:
			fillImageRect(dst, lg.Swatch, minimapDangerFill)
		case MarkPin:
			strokeImageRect(dst, lg.Swatch, 2, gold)
		}
		g.label(dst, lg.Text.Text, lg.Text.X, lg.Text.Y, infoStyleColor(lg.Text.Style))
	}
	for _, w := range v.Weapons {
		fillImageRect(dst, w.Swatch.Inset(-1), textShadow)
		fillImageRect(dst, w.Swatch, w.Color)
		g.label(dst, w.Text.Text, w.Text.X, w.Text.Y, infoStyleColor(w.Text.Style))
	}
	for _, t := range v.Texts {
		g.label(dst, t.Text, t.X, t.Y, infoStyleColor(t.Style))
	}
}

// overviewUnitColor は全体図のユニットの下地の色（DES-UI-30）。盤面と同じ陣営の色で、行動済みの自軍は盤面と同じく暗くする。
func overviewUnitColor(u tactics.Unit) color.NRGBA {
	base := playerBase
	if u.Side == tactics.Enemy {
		base = enemyBase
	}
	if u.Side == tactics.Player && u.Acted {
		base.R, base.G, base.B = base.R/2, base.G/2, base.B/2
	}
	return base
}

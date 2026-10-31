package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// ミニマップの欄の配置（DES-UI-28、DES-IT06 12.5 節）。
// 右下の x = 512〜831、y = 360〜511 の 320×152 をミニマップだけに使う。
// 地図の本体は x = 608〜735、y = 376〜503 の 128×128 の中に、縦横の比を保って縮める。
const (
	// MinimapTop はミニマップの欄の上端。情報欄の下端と同じ。
	MinimapTop = InfoHeight
	// minimapBox は地図の本体を置く正方形の一辺（px）。
	minimapBox = 128
	// minimapBoxX は地図の本体の左端。欄の中央に置き、左右の余白は 96 px ずつ。
	minimapBoxX = MapSize + (InfoWidth-minimapBox)/2
	// minimapBoxY は地図の本体の上端。上に見出しの 16 px を取る（下の余白は 8 px）。
	minimapBoxY = MinimapTop + lineHeight
)

// MinimapRect はミニマップの欄の範囲（DES-UI-28）。この中のクリックは盤面・情報欄に渡さない。
func MinimapRect() image.Rectangle {
	return image.Rect(MapSize, MinimapTop, ScreenWidth, MapSize)
}

// MinimapLayout は、幅 Width・高さ Height のマップを 128×128 に縮めたときの配置（DES-UI-28）。
// Area は地図の本体（マスを描く範囲）、Cell は 1 マスの大きさ（px）。
type MinimapLayout struct {
	Width, Height int
	Cell          int
	Area          image.Rectangle
}

// minimapLayout は、縦横の長い方が 128 px に収まる整数のマスの大きさを選び、本体を 128×128 の中央に寄せる（DES-UI-28）。
// 32×32 なら 4 px、16×16 なら 8 px。長方形のマップは短い方の向きに余白が付く。
func minimapLayout(w, h int) MinimapLayout {
	l := MinimapLayout{Width: w, Height: h}
	if w <= 0 || h <= 0 {
		return l
	}
	l.Cell = max(1, minimapBox/max(w, h))
	x := minimapBoxX + (minimapBox-w*l.Cell)/2
	y := minimapBoxY + (minimapBox-h*l.Cell)/2
	l.Area = image.Rect(x, y, x+w*l.Cell, y+h*l.Cell)
	return l
}

// CellRect はマス p を描く範囲を返す（DES-UI-28）。
func (l MinimapLayout) CellRect(p tactics.Pos) image.Rectangle {
	x, y := l.Area.Min.X+p.X*l.Cell, l.Area.Min.Y+p.Y*l.Cell
	return image.Rect(x, y, x+l.Cell, y+l.Cell)
}

// CellAt は画面の座標をマスに変える（DES-UI-28）。地図の本体の外（余白）なら ok が偽。
func (l MinimapLayout) CellAt(x, y int) (tactics.Pos, bool) {
	if l.Cell <= 0 || !image.Pt(x, y).In(l.Area) {
		return tactics.Pos{}, false
	}
	return tactics.Pos{X: (x - l.Area.Min.X) / l.Cell, Y: (y - l.Area.Min.Y) / l.Cell}, true
}

// spanRect は、マス a から b まで（両端を含む）を囲む範囲を、地図の本体の中に切って返す（DES-UI-28）。
func (l MinimapLayout) spanRect(a, b tactics.Pos) image.Rectangle {
	return image.Rectangle{Min: l.CellRect(a).Min, Max: l.CellRect(b).Max}.Intersect(l.Area)
}

// MinimapTerrain はミニマップで塗り分ける地形の種類（DES-UI-28）。盤面の地形の絵と同じ分け方にする。
type MinimapTerrain int

const (
	MinimapPlain MinimapTerrain = iota
	MinimapForest
	MinimapMountain
	MinimapFort
	MinimapWall
)

// minimapTerrain は、盤面に重ねる地形の絵（terrainTile）から塗りの種類を決める（DES-UI-05、DES-UI-28）。
func minimapTerrain(t tactics.Terrain) MinimapTerrain {
	tile, ok := terrainTile(t)
	if !ok {
		return MinimapPlain
	}
	switch tile {
	case assets.Forest:
		return MinimapForest
	case assets.Mountain:
		return MinimapMountain
	case assets.Fort:
		return MinimapFort
	case assets.Wall:
		return MinimapWall
	}
	return MinimapPlain
}

// MinimapCell は地形の 1 マス（DES-UI-28）。
type MinimapCell struct {
	Rect    image.Rectangle
	Terrain MinimapTerrain
}

// MinimapUnit は生きているユニットの 1 マス。自軍は青、敵は赤で塗る（DES-UI-28）。
type MinimapUnit struct {
	Rect image.Rectangle
	Side tactics.Side
}

// MinimapMark は凡例の印の種類（DES-UI-28）。
type MinimapMark int

const (
	MarkPlayer MinimapMark = iota
	MarkEnemy
	MarkView
	MarkHint
)

// MinimapLegend は余白に置く凡例の 1 行。Swatch に印を描き、右に文を出す（DES-UI-28）。
type MinimapLegend struct {
	Swatch image.Rectangle
	Mark   MinimapMark
	Text   SceneText
}

// MinimapView はミニマップに描くもの（DES-UI-28）。描画から分け、Ebitengine なしで確かめられるようにする。
// Available が偽なら、地図は描かず、見出しと Texts だけを描く（盤面を表示できない巻き戻しの下見）。
type MinimapView struct {
	Available bool
	Layout    MinimapLayout
	Cells     []MinimapCell
	Units     []MinimapUnit
	// Hints は分かっている予兆の範囲。View は今の盤面の表示範囲（白い枠）。
	Hints  []image.Rectangle
	View   image.Rectangle
	Legend []MinimapLegend
	Texts  []SceneText
}

// minimapTexts は見出しと、左右の余白の凡例・案内（DES-UI-28）。
func minimapTexts() ([]MinimapLegend, []SceneText) {
	texts := []SceneText{
		{Text: "ミニマップ", X: MapSize + infoMargin, Y: MinimapTop, Style: StyleTitle},
		{Text: "クリックで", X: minimapBoxX + minimapBox + infoMargin, Y: minimapBoxY + 4, Style: StyleLabel},
		{Text: "その場所へ", X: minimapBoxX + minimapBox + infoMargin, Y: minimapBoxY + 4 + lineHeight, Style: StyleLabel},
	}
	var legend []MinimapLegend
	for i, it := range []struct {
		mark MinimapMark
		text string
	}{{MarkPlayer, "自軍"}, {MarkEnemy, "敵"}, {MarkView, "表示範囲"}, {MarkHint, "気配"}} {
		y := minimapBoxY + 4 + i*(lineHeight+2)
		x := MapSize + infoMargin
		legend = append(legend, MinimapLegend{
			Swatch: image.Rect(x, y+4, x+8, y+12),
			Mark:   it.mark,
			Text:   SceneText{Text: it.text, X: x + 12, Y: y, Style: StyleLabel},
		})
	}
	return legend, texts
}

// buildMinimap は、盤面の見え方（地形・描くユニット・分かっている予兆・カメラ）からミニマップを作る（DES-UI-28）。
// 倒れたユニットと盤の外のユニットは描かない。まだ出ていない増援は units に入っていないので描かれない。
func buildMinimap(b terrainSource, units []tactics.Unit, hints []tactics.ReinforcementHint, cam Camera) MinimapView {
	v := MinimapView{Available: true, Layout: minimapLayout(b.Width(), b.Height())}
	v.Legend, v.Texts = minimapTexts()
	l := v.Layout
	if l.Cell == 0 {
		v.Available = false
		return v
	}
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			p := tactics.Pos{X: x, Y: y}
			v.Cells = append(v.Cells, MinimapCell{Rect: l.CellRect(p), Terrain: minimapTerrain(b.TerrainAt(p))})
		}
	}
	for _, h := range hints {
		if r := l.spanRect(h.Min, h.Max); !r.Empty() {
			v.Hints = append(v.Hints, r)
		}
	}
	for _, u := range units {
		if !u.Alive || u.Pos.X < 0 || u.Pos.Y < 0 || u.Pos.X >= l.Width || u.Pos.Y >= l.Height {
			continue
		}
		v.Units = append(v.Units, MinimapUnit{Rect: l.CellRect(u.Pos), Side: u.Side})
	}
	last := tactics.Pos{X: min(l.Width, cam.X+16) - 1, Y: min(l.Height, cam.Y+16) - 1}
	v.View = l.spanRect(tactics.Pos{X: cam.X, Y: cam.Y}, last)
	return v
}

// unavailableMinimap は盤面を表示できないときのミニマップ。古い盤面を残さない（DES-UI-21、DES-UI-28）。
func unavailableMinimap() MinimapView {
	var v MinimapView
	_, v.Texts = minimapTexts()
	v.Texts = append(v.Texts[:1], SceneText{Text: "表示できません", X: minimapBoxX, Y: minimapBoxY + 4, Style: StyleLabel})
	return v
}

// gridTerrain は配置の地形の表を terrainSource として使う（DES-UI-23、DES-UI-28）。
type gridTerrain [][]tactics.Terrain

func (g gridTerrain) Height() int { return len(g) }
func (g gridTerrain) Width() int {
	if len(g) == 0 {
		return 0
	}
	return len(g[0])
}
func (g gridTerrain) TerrainAt(p tactics.Pos) tactics.Terrain { return g[p.Y][p.X] }

// CenterOn はカメラの中心をマス p に移し、今の上限で止める（DES-UI-28、Q-DES06-10）。
// 16 マスの表示の中心は、左上から 0 から数えて 8 のマスとする。
func (c *Camera) CenterOn(p tactics.Pos) {
	c.X, c.Y = p.X-8, p.Y-8
	c.Move(0, 0)
}

// Minimap は、今の盤面（ふつうの戦闘か巻き戻しの下見）のミニマップを返す（DES-UI-28）。
// 盤面に描いているものと同じ元を使う。戦闘では再生中の写しや移動先に置いた写し（Units）と、表示中の予兆だけを描く。
func (c *Controller) Minimap() MinimapView {
	if c.rewind.open {
		v := c.RewindBoard()
		if !v.Available {
			return unavailableMinimap()
		}
		var hints []tactics.ReinforcementHint
		if v.Hint.Visible {
			hints = v.Hint.Areas
		}
		return buildMinimap(v.Board, v.Units, hints, c.rewind.camera)
	}
	var hints []tactics.ReinforcementHint
	if h := c.ReinforcementHintView(); h.Visible {
		hints = h.Areas
	}
	return buildMinimap(c.battle, c.Units(), hints, c.camera)
}

// MinimapClick はミニマップの欄のクリックを受ける（DES-UI-28）。欄の中なら真を返し、盤面・情報欄には渡さない。
// 地図の本体のクリックでカメラの中心を移す。余白のクリックは無視する。
// 移動・攻撃の操作として扱わず、選択・乱数・履歴・敵の範囲の設定を変えない。
// カメラを動かせるのは、矢印・ホイールで盤面を動かせるときと同じ（再生中・行動の一覧・確認の間は動かさない）。
func (c *Controller) MinimapClick(x, y int) bool {
	if !image.Pt(x, y).In(MinimapRect()) {
		return false
	}
	if c.rewind.open {
		if c.rewind.confirm || len(c.battle.History()) > 0 && !c.previewOK() {
			return true
		}
		v := c.RewindBoard()
		if p, ok := minimapLayout(v.Board.Width(), v.Board.Height()).CellAt(x, y); ok {
			c.rewind.camera.CenterOn(p)
		}
		return true
	}
	if c.confirmEnd {
		return true
	}
	switch c.state {
	case Idle, Selected, Targeting, HealTargeting:
		if p, ok := minimapLayout(c.battle.Width(), c.battle.Height()).CellAt(x, y); ok {
			c.camera.CenterOn(p)
		}
	}
	return true
}

// deployMinimap は配置の盤面（DeployView）のミニマップを返す（DES-UI-23、DES-UI-28）。
// 配置ではまだ出ていない増援を見せないので、予兆は描かない。
func (s *SceneController) deployMinimap(d *DeployView) MinimapView {
	if len(d.Terrain) == 0 {
		return unavailableMinimap()
	}
	return buildMinimap(gridTerrain(d.Terrain), d.Units, nil, s.deployCamera)
}

// deployMinimapClick は配置のミニマップのクリックで配置のカメラを動かす（DES-UI-28）。仮配置と選択は変えない。
func (s *SceneController) deployMinimapClick(x, y int) bool {
	if !image.Pt(x, y).In(MinimapRect()) {
		return false
	}
	if p, ok := minimapLayout(s.deployCamera.Width, s.deployCamera.Height).CellAt(x, y); ok {
		s.deployCamera.CenterOn(p)
	}
	return true
}

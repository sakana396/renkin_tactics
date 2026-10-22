package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面の構成（DES-UI-01）。タイルは 16 px を 2 倍で描き、16 × 16 マスの盤面は 512 × 512 px。
const (
	// CellSize は 1 マスの大きさ（px）。
	CellSize = 32
	// MapSize は盤面の幅と高さ（px）。
	MapSize = 16 * CellSize
	// InfoWidth は右の情報欄の幅（px）。
	InfoWidth = 320
	// GuideHeight は下の案内欄の高さ（px）。
	GuideHeight = 64
	// ScreenWidth と ScreenHeight は窓の大きさ（px）。
	ScreenWidth  = MapSize + InfoWidth
	ScreenHeight = MapSize + GuideHeight
)

// 行動の一覧とアイテムの一覧のボタンの大きさ（px）。
const (
	menuWidth  = 168
	menuHeight = 22
)

// CellAt は、画面の座標 (x, y) にあるマスを返す（DES-UI-01）。盤面の外なら ok が偽。
func CellAt(x, y int) (p tactics.Pos, ok bool) {
	if x < 0 || y < 0 || x >= MapSize || y >= MapSize {
		return tactics.Pos{}, false
	}
	return tactics.Pos{X: x / CellSize, Y: y / CellSize}, true
}

// MenuRects は、移動先 dest の横に n 個のボタンを縦に並べたときの、各ボタンの範囲を返す（DES-UI-02）。
// 盤面の右に出られなければ左に出し、下に出られなければ上にずらす。描画とクリックの判定の両方で使う。
func MenuRects(dest tactics.Pos, n int) []image.Rectangle {
	x := (dest.X + 1) * CellSize
	if x+menuWidth > MapSize {
		x = dest.X*CellSize - menuWidth
	}
	y := dest.Y * CellSize
	if y+n*menuHeight > MapSize {
		y = MapSize - n*menuHeight
	}
	rects := make([]image.Rectangle, n)
	for i := range rects {
		rects[i] = image.Rect(x, y+i*menuHeight, x+menuWidth, y+(i+1)*menuHeight)
	}
	return rects
}

// HitMenu は、画面の座標 (x, y) が rects のどれに入るかを返す。どれにも入らなければ -1。
func HitMenu(rects []image.Rectangle, x, y int) int {
	pt := image.Pt(x, y)
	for i, r := range rects {
		if pt.In(r) {
			return i
		}
	}
	return -1
}

// ClickMenu は、ボタンの i 番目を選んだときの処理（DES-UI-02）。Moved では行動、ItemMenu ではアイテムを選ぶ。
func (c *Controller) ClickMenu(i int) {
	switch c.state {
	case Moved:
		if actions := c.Actions(); i >= 0 && i < len(actions) {
			c.ChooseAction(actions[i])
		}
	case ItemMenu:
		c.ChooseItem(i)
	}
}

// MenuLabels は、ボタンに出す文を返す（DES-UI-02）。Moved では行動、ItemMenu では手持ちのアイテム、ほかは空。
func (c *Controller) MenuLabels() []string {
	var out []string
	switch c.state {
	case Moved:
		for _, a := range c.Actions() {
			out = append(out, actionName(a))
		}
	case ItemMenu:
		for _, s := range c.battle.Units()[c.unit].Items {
			out = append(out, itemLabel(s))
		}
	}
	return out
}

// BattleEndRect は勝利・敗北の帯の描画とクリックに使う範囲（DES-UI-17）。
func BattleEndRect() image.Rectangle {
	return image.Rect(0, MapSize/2-56, MapSize, MapSize/2+40)
}

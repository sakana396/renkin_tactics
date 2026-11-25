package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面の構成（DES-UI-01、DES-NG-10）。既定は、タイルを 16 px の 2 倍（1 マス 32 px）で描き、16 × 16 マスの盤面は 512 × 512 px。
// 新作の高解像度の表示（Config.HiRes）では 1 マス 48 px、盤面は 768 × 768 px になる。
const (
	// BaseCellSize は既定の 1 マスの大きさ（px）。HiResCellSize は高解像度の表示の 1 マスの大きさ（px）。
	BaseCellSize  = 32
	HiResCellSize = 48
	// boardCells は盤面の表示範囲（マス）。縦横とも 16 マスで固定する。
	boardCells = 16
	// InfoWidth は右の情報欄の幅（px）。
	InfoWidth = 320
	// InfoHeight は右の情報欄の高さ（px）。その下はミニマップだけに使う（DES-UI-28）。
	InfoHeight = 360
	// GuideHeight は下の案内欄の高さ（px）。
	GuideHeight = 64
	// 盤面を持たない場面（タイトル・物語・準備など）の配置の大きさ（px）。既定の窓の大きさと同じ。
	// 高解像度の表示では、窓の中央に置く（sceneOffset）。
	sceneCanvasWidth  = boardCells*BaseCellSize + InfoWidth
	sceneCanvasHeight = boardCells*BaseCellSize + GuideHeight
)

// Metrics は表示の大きさと配置をまとめた値（DES-NG-10）。
type Metrics struct {
	Cell, Map, ScreenW, ScreenH int
	forecastCandidates          [4]image.Rectangle
	forecastFallbackRect        image.Rectangle
	summaryX                    int
	summaryPortraitX            int
	summaryNarrow               int
	detailButtonRect            image.Rectangle
	deployDetailButtonRect      image.Rectangle
	detailBottom                int
	rewindLeft                  int
	rewindRight                 int
	cameraGuideY                int
	hintRewindY                 int
	minimapBoxX                 int
	equipMarkSize               int
	equipMarkOffset             int
	unitHPBarRight              int
	zocEnemyCornerY             int
	overviewButtonRect          image.Rectangle
	nextAllyButtonRect          image.Rectangle
	rangeStatusRect             image.Rectangle
	rangeClearRect              image.Rectangle
}

// NewMetrics はマスの大きさから配置を計算する（DES-NG-10）。
func NewMetrics(cell int) Metrics {
	m := Metrics{Cell: cell, Map: boardCells * cell}
	m.ScreenW = m.Map + InfoWidth
	m.ScreenH = m.Map + GuideHeight
	sh := m.boardShift()
	// 予測窓の候補: 盤面の四隅の 416×240（上は y = 8、下は盤面の下端から 8 px 空ける。右は盤面の右端に付ける）。
	m.forecastCandidates = [4]image.Rectangle{
		image.Rect(0, 8, 416, 248), image.Rect(m.Map-416, 8, m.Map, 248),
		image.Rect(0, m.Map-248, 416, m.Map-8), image.Rect(m.Map-416, m.Map-248, m.Map, m.Map-8),
	}
	m.forecastFallbackRect = image.Rect(m.Map, 0, m.ScreenW, InfoHeight)
	m.summaryX = m.Map + infoMargin
	m.summaryPortraitX = m.ScreenW - infoMargin - PortraitSize
	m.summaryNarrow = m.summaryPortraitX - infoMargin - m.summaryX
	m.detailButtonRect = image.Rect(672+sh, infoControlTop, 744+sh, infoControlTop+32)
	m.deployDetailButtonRect = image.Rect(752+sh, 336, 824+sh, 356)
	m.detailBottom = m.Map - infoMargin
	m.rewindLeft = m.Map + 8
	m.rewindRight = m.rewindLeft + rewindWidth
	m.cameraGuideY = m.ScreenH - lineHeight
	m.hintRewindY = m.ScreenH - lineHeight
	m.minimapBoxX = m.Map + (InfoWidth-minimapBox)/2
	m.equipMarkSize = m.scaled(EquipMarkSize)
	m.equipMarkOffset = m.Cell - m.equipMarkSize
	m.unitHPBarRight = m.equipMarkOffset - m.scaled(1)
	m.zocEnemyCornerY = m.Cell - m.scaled(8)
	m.overviewButtonRect = image.Rect(744+sh, 472, 824+sh, 504)
	m.nextAllyButtonRect = image.Rect(704+sh, 520+sh, 824+sh, 544+sh)
	m.rangeStatusRect = image.Rect(520+sh, 304, 664+sh, 336)
	m.rangeClearRect = image.Rect(752+sh, 304, 824+sh, 336)
	return m
}

func (m Metrics) scaled(n int) int { return n * m.Cell / BaseCellSize }
func (m Metrics) boardShift() int  { return m.Map - boardCells*BaseCellSize }
func (m Metrics) sceneOffset() image.Point {
	return image.Pt((m.ScreenW-sceneCanvasWidth)/2, (m.ScreenH-sceneCanvasHeight)/2)
}

// metricsForConfig は設定から表示の大きさを決める（DES-NG-10）。
func metricsForConfig(cfg Config) Metrics {
	if cfg.HiRes {
		return NewMetrics(HiResCellSize)
	}
	return NewMetrics(BaseCellSize)
}

// onCanvas は、場面 sc が盤面を持たない配置（sceneOffset で中央に置く）かを返す（DES-NG-10）。
// 戦闘・配置は盤面と右の情報欄で窓を使う。敗北は盤面を描かないので、タイトルなどと同じく中央に置く。
// ただし敗北でも、巻き戻しの一覧・確認を開いている間は盤面と右欄で窓を使う（canvasInUse）。
func onCanvas(sc Scene) bool {
	switch sc {
	case Battle, Deploy:
		return false
	}
	return true
}

// canvasInUse は、場面 sc を今、中央寄せの層に描くか（入力の座標も補正するか）を返す（DES-NG-10）。
// 敗北で巻き戻しの一覧を開いている間（rewindOpen）は、左に盤面・右に一覧を窓いっぱいに描くので、層を使わない。
func canvasInUse(sc Scene, rewindOpen bool) bool {
	if sc == GameOver && rewindOpen {
		return false
	}
	return onCanvas(sc)
}

// sceneCursor は、窓の座標 (x, y) を場面の配置の座標へ直す。中央寄せの層に描く場面では、sceneOffset の分を引く（DES-NG-10）。
// 盤面を持つ場面と、高解像度でない表示（sceneOffset が (0, 0)）では、そのまま返す。
func (m Metrics) sceneCursor(x, y int, sc Scene, rewindOpen bool) (int, int) {
	if !canvasInUse(sc, rewindOpen) {
		return x, y
	}
	off := m.sceneOffset()
	return x - off.X, y - off.Y
}

// 行動の一覧とアイテムの一覧のボタンの大きさ（px）。
const (
	menuWidth  = 168
	menuHeight = 22
)

// CellAt は原点・16×16 の座標変換を返す互換の入口（DES-UI-01、DES-UI-25）。実画面では Camera.CellAt を使う。
func CellAt(m Metrics, x, y int) (p tactics.Pos, ok bool) {
	return (Camera{Width: boardCells, Height: boardCells}).CellAt(m, x, y)
}

// MenuRects は、移動先 dest の横に n 個のボタンを縦に並べたときの、各ボタンの範囲を返す（DES-UI-02）。
// 盤面の右に出られなければ左に出し、下に出られなければ上にずらす。描画とクリックの判定の両方で使う。
func MenuRects(m Metrics, dest tactics.Pos, n int) []image.Rectangle {
	return menuRectsWidth(m, dest, n, menuWidth)
}

// menuRectsWidth は MenuRects のボタンの幅を w にしたもの。長い行（爆弾の効果の文）を一覧の枠と盤面に収めるのに使う（DES-NG-05）。
func menuRectsWidth(m Metrics, dest tactics.Pos, n, menuWidth int) []image.Rectangle {
	x := (dest.X + 1) * m.Cell
	if x+menuWidth > m.Map {
		x = dest.X*m.Cell - menuWidth
	}
	x = max(0, min(x, m.Map-menuWidth))
	y := dest.Y * m.Cell
	if y+n*menuHeight > m.Map {
		y = m.Map - n*menuHeight
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

// ClickMenu は、ボタンの i 番目を選んだときの処理（DES-UI-02）。Moved では行動、ItemMenu ではアイテム、
// WeaponMenu・StaffMenu では武器・杖を選ぶ（DES-UI-26）。
func (c *Controller) ClickMenu(i int) {
	switch c.state {
	case Moved:
		if actions := c.Actions(); i >= 0 && i < len(actions) {
			c.ChooseAction(actions[i])
		}
	case ItemMenu:
		if ks := c.availableItems(); i >= 0 && i < len(ks) {
			c.ChooseItem(ks[i])
		}
	case WeaponMenu, StaffMenu:
		c.ChooseWeapon(i)
	case Canto:
		if i == 0 {
			c.EndCanto()
		}
	}
}

// MenuLabels は、ボタンに出す文を返す（DES-UI-02）。Moved では行動、ItemMenu では手持ちのアイテム、Canto では「終える」（DES-IT06 12.3 節）、
// WeaponMenu・StaffMenu では武器・杖（DES-UI-26）、ほかは空。
func (c *Controller) MenuLabels() []string {
	var out []string
	switch c.state {
	case Moved:
		for _, a := range c.Actions() {
			out = append(out, actionName(a))
		}
	case ItemMenu:
		u := c.battle.Units()[c.unit]
		for _, k := range c.availableItems() {
			out = append(out, itemLabel(u.Items[k]))
		}
	case WeaponMenu, StaffMenu:
		// 武器の一覧は最大 5 行。選べない武器には印を付け、理由は案内欄に出す（DES-IT06 12.1 節）。
		units := c.battle.Units()
		for _, w := range c.menuWeapons() {
			label := units[c.unit].Weapons[w].Name
			if c.weaponReason(w) != "" {
				label += "（使えない）"
			}
			out = append(out, label)
		}
	case Canto:
		out = append(out, cantoEndLabel)
	}
	return out
}

// BattleEndRect は勝利・敗北の帯の描画とクリックに使う範囲（DES-UI-17）。
func BattleEndRect(m Metrics) image.Rectangle {
	return image.Rect(0, m.Map/2-56, m.Map, m.Map/2+40)
}

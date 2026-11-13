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

// マスの大きさに従う値（DES-NG-10）。useMetrics が一か所で切り替える。
// 既定は、1 マス 32 px・盤面 512・窓 832 × 576。
var (
	// CellSize は 1 マスの大きさ（px）。
	CellSize = BaseCellSize
	// MapSize は盤面の幅と高さ（px）。
	MapSize = boardCells * BaseCellSize
	// ScreenWidth と ScreenHeight は窓の大きさ（px）。
	ScreenWidth  = boardCells*BaseCellSize + InfoWidth
	ScreenHeight = boardCells*BaseCellSize + GuideHeight
)

// useMetrics は、1 マスの大きさを cell（px）に切り替え、それに従う値を計算し直す（DES-NG-10）。
// 窓を作る前に 1 回だけ呼ぶ。グローバルな状態を書き換えるので、テストでは useTestMetrics を使う。
func useMetrics(cell int) {
	CellSize = cell
	MapSize = boardCells * cell
	ScreenWidth = MapSize + InfoWidth
	ScreenHeight = MapSize + GuideHeight
	recomputeMetrics()
}

// scaled は、32 px のマスを前提に決めた長さ n（px）を、今のマスの大きさに比例させた長さにする（DES-NG-10 フェーズ 2）。
// 既定（32）では n のまま。1 マス 48 px では 1.5 倍（端数は切り捨て）。HP の棒・印など、マスの中の小さな部品に使う。
func scaled(n int) int { return n * CellSize / BaseCellSize }

// boardShift は、既定の盤面（512 px）に対する盤面の大きさの差（px）。既定では 0、1 マス 48 px では 256。
// 右欄の x と、盤面の下の案内欄の y は、この分だけ後ろへずれる（DES-NG-10）。
func boardShift() int { return MapSize - boardCells*BaseCellSize }

// recomputeMetrics は、盤面の大きさ（MapSize）に従う配置の値を計算し直す（DES-NG-10）。
// 既定の値は、盤面 512・右欄 x = 512〜831・案内欄 y = 512〜575 の配置と同じになる。
// package の init と useMetrics から呼ぶ。盤面の大きさに従う値を足すときは、ここに足す。
func recomputeMetrics() {
	sh := boardShift()
	// 予測窓の候補: 盤面の四隅の 416×240（上は y = 8、下は盤面の下端から 8 px 空ける。右は盤面の右端に付ける）。
	forecastCandidates = [4]image.Rectangle{
		image.Rect(0, 8, 416, 248), image.Rect(MapSize-416, 8, MapSize, 248),
		image.Rect(0, MapSize-248, 416, MapSize-8), image.Rect(MapSize-416, MapSize-248, MapSize, MapSize-8),
	}
	forecastFallbackRect = image.Rect(MapSize, 0, ScreenWidth, InfoHeight)
	summaryX = MapSize + infoMargin
	summaryPortraitX = ScreenWidth - infoMargin - PortraitSize
	summaryNarrow = summaryPortraitX - infoMargin - summaryX
	detailButtonRect = image.Rect(672+sh, infoControlTop, 744+sh, infoControlTop+32)
	deployDetailButtonRect = image.Rect(752+sh, 336, 824+sh, 356)
	detailBottom = MapSize - infoMargin
	rewindLeft = MapSize + 8
	rewindRight = rewindLeft + rewindWidth
	cameraGuideY = ScreenHeight - lineHeight
	hintRewindY = ScreenHeight - lineHeight
	minimapBoxX = MapSize + (InfoWidth-minimapBox)/2
	equipMarkSize = scaled(EquipMarkSize)
	equipMarkOffset = CellSize - equipMarkSize
	unitHPBarRight = equipMarkOffset - scaled(1)
	zocEnemyCornerY = CellSize - scaled(8)
	overviewButtonRect = image.Rect(744+sh, 472, 824+sh, 504)
	nextAllyButtonRect = image.Rect(704+sh, 520+sh, 824+sh, 544+sh)
	rangeStatusRect = image.Rect(520+sh, 304, 664+sh, 336)
	rangeClearRect = image.Rect(752+sh, 304, 824+sh, 336)
}

func init() { recomputeMetrics() }

// useConfigMetrics は、設定に従って 1 マスの大きさを切り替える。窓を作る前に呼ぶ（DES-NG-10）。
func useConfigMetrics(cfg Config) {
	if cfg.HiRes {
		useMetrics(HiResCellSize)
		return
	}
	useMetrics(BaseCellSize)
}

// sceneOffset は、盤面を持たない場面の配置（sceneCanvasWidth × sceneCanvasHeight）を窓の中央に置く位置を返す（DES-NG-10）。
// 既定の表示では窓と同じ大きさなので (0, 0)。
func sceneOffset() image.Point {
	return image.Pt((ScreenWidth-sceneCanvasWidth)/2, (ScreenHeight-sceneCanvasHeight)/2)
}

// onCanvas は、場面 sc が盤面を持たない配置（sceneOffset で中央に置く）かを返す（DES-NG-10）。
// 戦闘・敗北・配置は盤面と右の情報欄で窓を使う。
func onCanvas(sc Scene) bool {
	switch sc {
	case Battle, GameOver, Deploy:
		return false
	}
	return true
}

// 行動の一覧とアイテムの一覧のボタンの大きさ（px）。
const (
	menuWidth  = 168
	menuHeight = 22
)

// CellAt は原点・16×16 の座標変換を返す互換の入口（DES-UI-01、DES-UI-25）。実画面では Camera.CellAt を使う。
func CellAt(x, y int) (p tactics.Pos, ok bool) {
	return (Camera{Width: boardCells, Height: boardCells}).CellAt(x, y)
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

// ClickMenu は、ボタンの i 番目を選んだときの処理（DES-UI-02）。Moved では行動、ItemMenu ではアイテム、
// WeaponMenu・StaffMenu では武器・杖を選ぶ（DES-UI-26）。
func (c *Controller) ClickMenu(i int) {
	switch c.state {
	case Moved:
		if actions := c.Actions(); i >= 0 && i < len(actions) {
			c.ChooseAction(actions[i])
		}
	case ItemMenu:
		// 一覧に出ているのは使えるアイテムだけなので、ボタンの番号を手持ちの添字に直す（DES-NG-05）。
		if ks := usableItems(c.battle.Units()[c.unit]); i >= 0 && i < len(ks) {
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

// MenuLabels は、ボタンに出す文を返す（DES-UI-02）。Moved では行動、ItemMenu では使えるアイテムだけ（DES-NG-05。usableItems）、Canto では「終える」（DES-IT06 12.3 節）、
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
		for _, k := range usableItems(u) {
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
func BattleEndRect() image.Rectangle {
	return image.Rect(0, MapSize/2-56, MapSize, MapSize/2+40)
}

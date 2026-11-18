package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// zocNote は、カーソルの下のマスが重騎士の隣のときに出す移動費の説明（DES-UI-29、DES-IT07 3.8 節）。
// 費用を払うのは相手の陣営だけなので、「相手の」重騎士の隣と書く。
const zocNote = "相手の重騎士の隣: +1"

// zocMarkInset はマスの角から角印の頂点までの距離、zocMarkArm は角印の 1 本の長さ（DES-UI-29）。
const (
	zocMarkInset = 5
	zocMarkArm   = 5
)

// zocEnemyCornerY は敵軍の角印の頂点の y（マスの上端から）。HP の棒（下端の 5 px）より上に置く（DES-NG-10。マスの大きさに従う）。
var zocEnemyCornerY int

// zocMarkLine は重騎士の ZOC の角印の線分 1 本（DES-UI-29、DES-IT07 3.8 節）。Side は重騎士の陣営で、線の色を決める。
type zocMarkLine struct {
	From, To image.Point
	Side     tactics.Side
}

// zocMarkLines は、盤面 board の生きている重騎士の隣の通れるマスに付ける角印を、画面の線分で返す（DES-UI-29、DES-IT07 3.8 節）。
// 自軍の重騎士は青でマスの左上、敵軍の重騎士は赤でマスの左下（HP の棒の上）に、細い L 字を描く。
// 右下は装備の印（DES-UI-33）、右上は固定のピンとボスの印に使うので避ける。両軍の ZOC が重なるマスには両方を描く。
// カメラの外のマスは返さない。並びは上の行から、同じ行は左から、同じマスは自軍・敵軍の順。
// 角印のマスは core の ZOCSides で決め、UI が独自に隣接や生死を判定しない。
func zocMarkLines(board tactics.BoardView, cam Camera) []zocMarkLine {
	var lines []zocMarkLine
	for y := 0; y < board.Height(); y++ {
		for x := 0; x < board.Width(); x++ {
			p := tactics.Pos{X: x, Y: y}
			if !cam.Visible(p) {
				continue
			}
			sides := board.ZOCSides(p)
			pt := cam.Screen(p)
			if sides[tactics.Player] {
				c := image.Pt(pt.X+scaled(zocMarkInset), pt.Y+scaled(zocMarkInset))
				lines = append(lines,
					zocMarkLine{From: c, To: image.Pt(c.X+scaled(zocMarkArm), c.Y), Side: tactics.Player},
					zocMarkLine{From: c, To: image.Pt(c.X, c.Y+scaled(zocMarkArm)), Side: tactics.Player})
			}
			if sides[tactics.Enemy] {
				c := image.Pt(pt.X+scaled(zocMarkInset), pt.Y+zocEnemyCornerY)
				lines = append(lines,
					zocMarkLine{From: c, To: image.Pt(c.X+scaled(zocMarkArm), c.Y), Side: tactics.Enemy},
					zocMarkLine{From: image.Pt(c.X, c.Y-scaled(zocMarkArm)), To: c, Side: tactics.Enemy})
			}
		}
	}
	return lines
}

// hasZOCAt は、盤面 board のマス p がどちらかの陣営の重騎士の隣（通れるマス）かを返す（DES-UI-29）。
func hasZOCAt(board tactics.BoardView, p tactics.Pos) bool {
	sides := board.ZOCSides(p)
	return sides[tactics.Player] || sides[tactics.Enemy]
}

// boardLayer は、盤面に重ねて描く層（DES-UI-29、DES-IT07 3.2・3.8 節）。
type boardLayer int

const (
	// layerUnits はユニットの絵。
	layerUnits boardLayer = iota
	// layerEnemyRanges は敵の範囲（全敵の塗り → 固定の縁 → カーソルの縁）。
	layerEnemyRanges
	// layerZOC は重騎士の ZOC の角印。
	layerZOC
	// layerEquipMarks はユニットの装備の印（DES-UI-33）。
	layerEquipMarks
	// layerWatchedPins は固定した敵の頭上のピン。
	layerWatchedPins
	// layerFrame は選択・行動したユニットの枠。
	layerFrame
)

// 層を描く順（先に描くものが下）。危険範囲を下、ZOC の角印を上に描く（DES-IT07 3.8 節）。
var (
	// battleLayers は戦闘の盤面で、ユニットの上に重ねる順。選択の枠などはこのあとに描く。
	battleLayers = []boardLayer{layerUnits, layerEnemyRanges, layerZOC, layerEquipMarks, layerWatchedPins}
	// deployLayers は配置の盤面で、出撃枠の縁の上に重ねる順。選んだ味方の枠を最後に描く。
	deployLayers = []boardLayer{layerEnemyRanges, layerZOC, layerEquipMarks, layerFrame}
	// rewindLayers は巻き戻しの下見の盤面の順。下見側の盤面の生存者で角印を描く。
	rewindLayers = []boardLayer{layerUnits, layerZOC, layerEquipMarks, layerFrame}
)

package ui

import (
	"image"
	"image/color"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 小隊長・配下・崩れの印（DES-UI-35、DES-IT07 5.4 節）。印の種類・形・色・位置をここで決め、
// 描画（game.go の drawCommandMark）から分けて Ebitengine なしで確かめられるようにする。
// 号令の文・配下を結ぶ線は出さない（E-118）。

// CommandMark は、ユニットに付ける指揮の印の種類（DES-UI-35）。
type CommandMark int

const (
	// NoCommandMark は印なし（自軍、ボス、初めからまっすぐ寄る敵＝ごろつき、一団のない敵）。
	NoCommandMark CommandMark = iota
	// CommanderMark は小隊長の金の菱形。
	CommanderMark
	// SubordinateMark は配下の、小隊長と同じ色の小点。
	SubordinateMark
	// CollapsedMark は崩れた配下の灰の割れた小点。章の終わりまで残す（Q-DES07-10、E-120）。
	CollapsedMark
)

// CommandMarkOf は、ユニットの印を返す（DES-UI-35）。tactics の状態（Commander・Collapsed・一団の寄り方）だけから決めるので、
// 再生中は出来事の写し、巻き戻しの下見は下見の盤面の写しのとおりに切り替わる。小隊長の撃破の出来事（EventFall）の写しでは、
// 配下はすでに崩れている（DES-AI-03）ので、印は撃破の出来事の時点で変わる。
func CommandMarkOf(u tactics.Unit) CommandMark {
	if u.Side != tactics.Enemy || u.Boss {
		return NoCommandMark
	}
	switch {
	case u.Commander:
		return CommanderMark
	case u.Collapsed:
		return CollapsedMark
	case u.Approach == tactics.GroupApproach && u.Group != "":
		return SubordinateMark
	}
	return NoCommandMark
}

// Word は、右欄の名前の横に印と並べる短い語を返す（DES-UI-35）。印なしは空。
func (m CommandMark) Word() string {
	switch m {
	case CommanderMark:
		return "小隊長"
	case SubordinateMark:
		return "配下"
	case CollapsedMark:
		return "崩れ"
	}
	return ""
}

// 印の形（DES-UI-35）。陣営の色の下地の上でも、色ではなく形で見分けられるようにする。
type commandMarkShape int

const (
	shapeNone commandMarkShape = iota
	// shapeDiamond は菱形（小隊長）。
	shapeDiamond
	// shapeDot は丸い小点（配下）。
	shapeDot
	// shapeCrackedDot は割れ目の線の入った小点（崩れ）。
	shapeCrackedDot
)

// commandMarkGray は崩れの印の灰。
var commandMarkGray = color.NRGBA{150, 150, 150, 255}

// commandMarkEdge は印の縁。金・灰が陣営の下地や絵に埋もれないように、少し大きく先に描く。
var commandMarkEdge = color.NRGBA{20, 20, 20, 255}

// shape は印の形を返す。
func (m CommandMark) shape() commandMarkShape {
	switch m {
	case CommanderMark:
		return shapeDiamond
	case SubordinateMark:
		return shapeDot
	case CollapsedMark:
		return shapeCrackedDot
	}
	return shapeNone
}

// color は印の色を返す。小隊長と配下は同じ金、崩れは灰（DES-UI-35）。
func (m CommandMark) color() color.NRGBA {
	if m == CollapsedMark {
		return commandMarkGray
	}
	return gold
}

const (
	// commandMarkSize は印の箱の一辺（px）。菱形は箱いっぱい、小点は箱の中央の半分ほどに描く。
	commandMarkSize = 12
	// commandMarkInset は 32×32 の絵の左上から印の箱までの間（px）。
	commandMarkInset = 1
	// commandMarkGap は右欄の名前と印、印と語の間（px）。
	commandMarkGap = 6
)

// CommandMarkRect は、マスの左上の画面の座標 cell に対する、盤面の印の箱を返す（DES-UI-35、DES-IT07 3.6 節）。
// 印は絵の左上に置き、固定範囲（右上）・装備（右下）の印と分ける。
func CommandMarkRect(cell image.Point) image.Rectangle {
	min := cell.Add(image.Pt(scaled(commandMarkInset), scaled(commandMarkInset)))
	size := scaled(commandMarkSize)
	return image.Rectangle{Min: min, Max: min.Add(image.Pt(size, size))}
}

// InfoMark は、右欄に描く指揮の印（DES-UI-35）。Rect は印の箱（画面の座標）。
type InfoMark struct {
	Rect image.Rectangle
	Mark CommandMark
}

// addCommandMark は、右欄の名前（左端 nameX、上端 y、文字列 name）の横に、印と短い語を置く（DES-UI-35）。
// 印なしなら何も足さない。
func (v *InfoView) addCommandMark(u tactics.Unit, nameX, y int, name string) {
	m := CommandMarkOf(u)
	if m == NoCommandMark {
		return
	}
	x := nameX + textWidth(name) + commandMarkGap
	top := y + (lineHeight-commandMarkSize)/2
	v.Marks = append(v.Marks, InfoMark{Rect: image.Rect(x, top, x+commandMarkSize, top+commandMarkSize), Mark: m})
	st := StyleTitle
	if m == CollapsedMark {
		st = StyleLabel
	}
	v.text(x+commandMarkSize+commandMarkGap/2, y, m.Word(), st)
}

package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// hintText は増援の気配の案内。領域の数・ターン・人数・位置を文に出さない（DES-UI-24）。
const hintText = "付近に気配…"

// hintMark は予兆領域の中央付近に添える小さい印（DES-UI-24）。
const hintMark = "…"

// 予兆の破線の形（DES-UI-24）。マスの端から hintInset px 内側に、太さ hintThick px、長さ hintDash px の線を
// hintPeriod px おきに描く。線の位置を画面の絶対座標にそろえ、重なる領域の同じ辺では同じ線になるようにする。
const (
	hintInset  = 2
	hintThick  = 2
	hintDash   = 4
	hintPeriod = 8
)

// hintRewindY は巻き戻しの右欄で案内を出す行の上端。下の行の「戻る」「決定」（y = 520〜560）の下の空きに置き、
// 右下のミニマップ（y = 360〜511）と重ねない（DES-UI-24、DES-UI-28、REV-071 F-03）。
const hintRewindY = ScreenHeight - lineHeight

// HintView は盤面に描く増援の気配の表示値（DES-UI-24）。
// Areas は tactics の予兆領域（4 マス単位に丸めた矩形）そのままで、正確な出現マスを持たない。
type HintView struct {
	Visible bool
	Areas   []tactics.ReinforcementHint
	// Text は案内の全文と位置。
	Text SceneText
}

// refreshHints は今の戦闘から予兆領域を取り直す。再生の完了後と巻き戻しの成功後だけ呼ぶ（DES-UI-24、DES-IT04 6.3 節）。
// 再生中は、先に進んだ本物の Turn の予兆を、表示中の出来事に混ぜない。
func (c *Controller) refreshHints() {
	c.hints = c.battle.BoardView().ReinforcementHints()
}

// ReinforcementHintView は通常の盤面の増援の気配を返す（DES-UI-24）。
// 出来事の再生中・勝敗の確定後（Over）・巻き戻しの一覧を開いている間は隠す。
// 案内は自軍入力待ちの案内欄の 3 行目の右端に出す（DES-IT04 7.4 節）。
func (c *Controller) ReinforcementHintView() HintView {
	if c.state == Playing || c.state == Over || c.rewind.open || len(c.hints) == 0 {
		return HintView{}
	}
	return c.Camera().hintView(HintView{
		Visible: true,
		Areas:   append([]tactics.ReinforcementHint(nil), c.hints...),
		Text:    SceneText{Text: hintText, X: MapSize - 8 - textWidth(hintText), Y: MapSize + 2 + 2*lineHeight, Style: StyleHint},
	})
}

// rewindHintView は巻き戻しの一覧・確認で、選択中の写しの予兆を返す（DES-UI-24、DES-UI-21）。
// 案内は右欄の理由の行の下の空き（y=hintRewindY）に出す。盤面を描けないときは隠す。
func (c *Controller) rewindHintView(areas []tactics.ReinforcementHint) HintView {
	if len(areas) == 0 {
		return HintView{}
	}
	return c.Camera().hintView(HintView{
		Visible: true,
		Areas:   append([]tactics.ReinforcementHint(nil), areas...),
		Text:    SceneText{Text: hintText, X: rewindLeft, Y: hintRewindY, Style: StyleHint},
	})
}

// hintShapes は予兆領域の外周の破線と、中央付近の印の左上の座標を返す（DES-UI-24）。
// 同じ線・同じ印は 1 つにまとめ、重なる領域を描いても濃くならないようにする。
func hintShapes(areas []tactics.ReinforcementHint) (dashes []image.Rectangle, marks []image.Point) {
	seenDash := map[image.Rectangle]bool{}
	seenMark := map[image.Point]bool{}
	add := func(r image.Rectangle) {
		if !seenDash[r] {
			seenDash[r] = true
			dashes = append(dashes, r)
		}
	}
	for _, a := range areas {
		x0, y0 := a.Min.X*CellSize, a.Min.Y*CellSize
		x1, y1 := (a.Max.X+1)*CellSize, (a.Max.Y+1)*CellSize
		// 横の辺: 角から始め、x ≡ 2 (mod 8) の位置にそろえる。
		for x := x0 + hintInset; x+hintDash <= x1-hintInset; x += hintPeriod {
			add(image.Rect(x, y0+hintInset, x+hintDash, y0+hintInset+hintThick))
			add(image.Rect(x, y1-hintInset-hintThick, x+hintDash, y1-hintInset))
		}
		// 縦の辺: 横の辺の線と角で重ならないよう、半周期ずらす。
		for y := y0 + hintInset + hintDash/2 + hintThick; y+hintDash <= y1-hintInset-hintThick; y += hintPeriod {
			add(image.Rect(x0+hintInset, y, x0+hintInset+hintThick, y+hintDash))
			add(image.Rect(x1-hintInset-hintThick, y, x1-hintInset, y+hintDash))
		}
		p := image.Pt((x0+x1)/2-textWidth(hintMark)/2, (y0+y1)/2-lineHeight/2)
		if !seenMark[p] {
			seenMark[p] = true
			marks = append(marks, p)
		}
	}
	return dashes, marks
}

// hintView は世界の予兆を削らず、画面外にもあることだけを案内する（DES-UI-25）。
func (c Camera) hintView(v HintView) HintView {
	for _, a := range v.Areas {
		if !c.Visible(a.Min) || !c.Visible(a.Max) {
			v.Text.Text = hintText + " 画面外にも気配"
			v.Text.X = rewindLeft
			if v.Text.Y != hintRewindY {
				v.Text.Y = ScreenHeight - 2*lineHeight
			}
			break
		}
	}
	return v
}

// hintShapes は共通カメラで変換して表示領域と交差する部分だけ返す（DES-UI-25）。
func (c Camera) hintShapes(areas []tactics.ReinforcementHint) ([]image.Rectangle, []image.Point) {
	d, m := hintShapes(areas)
	offset := image.Pt(c.X*CellSize, c.Y*CellSize)
	bounds := image.Rect(0, 0, MapSize, MapSize)
	var ds []image.Rectangle
	var ms []image.Point
	for _, r := range d {
		r = r.Sub(offset).Intersect(bounds)
		if !r.Empty() {
			ds = append(ds, r)
		}
	}
	for _, p := range m {
		p = p.Sub(offset)
		if p.In(bounds) {
			ms = append(ms, p)
		}
	}
	return ds, ms
}

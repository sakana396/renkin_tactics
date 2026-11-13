package ui

import "image"

// 場面の肖像の配置（DES-NG-08）。文字の左に、正方形の箱へ収めて出す。
const (
	storyPortraitX    = 48
	storyPortraitY    = 120
	storyPortraitSize = 224
	// storyTextGap は肖像と文字の間。
	storyTextGap = 24
	// storyMargin は文字の右の余白（wrapSceneText の左右 48 px と同じ）。
	storyMargin = 48
)

// ScenePortrait は物語の場面に出す肖像（DES-NG-08）。Path は media の下の道筋、Rect は箱（絵はこの中に縦横比を保って収める）。
type ScenePortrait struct {
	Path string
	Rect image.Rectangle
}

// fitRect は、w × h の絵を縦横比を保って box の中に最大に収め、中央に置いた範囲を返す（DES-NG-08）。
// w・h が 0 以下、または box が空なら空の範囲。
func fitRect(box image.Rectangle, w, h int) image.Rectangle {
	if w <= 0 || h <= 0 || box.Empty() {
		return image.Rectangle{}
	}
	dw, dh := box.Dx(), box.Dy()
	// 箱の縦横比より絵が横長なら幅に合わせ、そうでなければ高さに合わせる。
	if w*dh >= h*dw {
		dh = h * dw / w
	} else {
		dw = w * dh / h
	}
	x := box.Min.X + (box.Dx()-dw)/2
	y := box.Min.Y + (box.Dy()-dh)/2
	return image.Rect(x, y, x+dw, y+dh)
}

// coverRect は、w × h の絵を screen を隙間なく覆う最小の大きさにし、中央に置いた範囲を返す（DES-NG-08）。
// 画面の縦横比と違うときは、はみ出した分を切る（描くときに screen で切り取る）。
func coverRect(screen image.Rectangle, w, h int) image.Rectangle {
	if w <= 0 || h <= 0 || screen.Empty() {
		return image.Rectangle{}
	}
	dw, dh := screen.Dx(), screen.Dy()
	if w*dh >= h*dw {
		// 絵が画面より横長: 高さに合わせ、幅がはみ出す。
		dw = (w*dh + h - 1) / h
	} else {
		dh = (h*dw + w - 1) / w
	}
	x := screen.Min.X + (screen.Dx()-dw)/2
	y := screen.Min.Y + (screen.Dy()-dh)/2
	return image.Rect(x, y, x+dw, y+dh)
}

// storyArtLayout は物語の肖像の箱と、文の左端・折り返しの幅を返す（DES-NG-08）。
// 肖像がなければ、今までの配置（左端 48、幅は画面から左右 48 を引いた値）を返し、箱は空。
func storyArtLayout(hasPortrait bool) (portrait image.Rectangle, textX, textWidth int) {
	if !hasPortrait {
		return image.Rectangle{}, 48, sceneCanvasWidth - 96
	}
	portrait = image.Rect(storyPortraitX, storyPortraitY, storyPortraitX+storyPortraitSize, storyPortraitY+storyPortraitSize)
	textX = portrait.Max.X + storyTextGap
	return portrait, textX, sceneCanvasWidth - storyMargin - textX
}

// unitArtRect はユニットの絵を、マスの範囲 cell（左上 (x, y)、一辺 size）に収める範囲を返す（DES-NG-08）。
func unitArtRect(x, y, size, w, h int) image.Rectangle {
	return fitRect(image.Rect(x, y, x+size, y+size), w, h)
}

package ui

import (
	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"strings"
)

// textWidth は描画と同じ bitmapfont の送り幅で、最長の行を測る（DES-UI-01、12）。
// text.GoXFace の Measure と同じ font.MeasureString を使い、描画から独立させる。
func textWidth(s string) int {
	width := 0
	for _, line := range strings.Split(s, "\n") {
		width = max(width, int(font.MeasureString(bitmapfont.Face, line))>>6)
	}
	return width
}

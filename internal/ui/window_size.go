package ui

import "github.com/hajimehoshi/ebiten/v2"

// 窓の初期の大きさを決めるときに、画面（モニター）の大きさのうち窓に使える割合（分子 / 分母）。
// 窓の枠・タイトルバー・タスクバーの分を残す（DES-NG-10）。
const (
	windowFitNum = 9
	windowFitDen = 10
)

// fitWindowSize は、論理の画面 logicalW × logicalH の窓の初期の大きさを返す（DES-NG-10）。
// モニターの大きさ monW × monH のうち 9 割に収まるなら、論理の大きさのまま返す。
// 収まらなければ、縦横比を保ったまま、収まる倍率に縮める。拡大はしない。
// モニターの大きさが取れないとき（0 以下）は論理の大きさのまま返す。画面を持たないテストで確かめられるよう、Ebitengine を呼ばない。
func fitWindowSize(monW, monH, logicalW, logicalH int) (w, h int) {
	if monW <= 0 || monH <= 0 || logicalW <= 0 || logicalH <= 0 {
		return logicalW, logicalH
	}
	availW, availH := monW*windowFitNum/windowFitDen, monH*windowFitNum/windowFitDen
	if logicalW <= availW && logicalH <= availH {
		return logicalW, logicalH
	}
	// 倍率 = min(availW / logicalW, availH / logicalH) を整数で比べて、縮める側の軸を決める。
	if availW*logicalH < availH*logicalW {
		return availW, max(1, logicalH*availW/logicalW)
	}
	return max(1, logicalW*availH/logicalH), availH
}

// setWindow は窓の初期の大きさと、大きさの変更の可否を決める（DES-NG-10）。
// 高解像度の表示（1088×832）のときだけ、画面に収まる大きさに縮め、利用者による大きさの変更を許す。
// Layout が固定の論理の大きさを返すので、絵は窓に合わせて拡大縮小される。既定の表示は今までと同じ（固定の大きさ）。
func setWindow(cfg Config) {
	if !cfg.HiRes {
		ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
		return
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	monW, monH := 0, 0
	if m := ebiten.Monitor(); m != nil {
		monW, monH = m.Size()
	}
	ebiten.SetWindowSize(fitWindowSize(monW, monH, ScreenWidth, ScreenHeight))
}

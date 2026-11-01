package ui

import "image/color"

// 色（DES-UI-03）。半透明の色があるので、アルファを掛けていない NRGBA で書く。
var (
	bgColor      = color.NRGBA{22, 28, 37, 255}
	panelColor   = color.NRGBA{37, 46, 58, 255}
	white        = color.NRGBA{232, 237, 238, 255}
	muted        = color.NRGBA{177, 193, 201, 255}
	gold         = color.NRGBA{255, 214, 92, 255}
	blueOverlay  = color.NRGBA{30, 90, 255, 130}
	greenOverlay = color.NRGBA{30, 220, 80, 130}
	redOverlay   = color.NRGBA{255, 30, 30, 140}
	playerBase   = color.NRGBA{50, 110, 230, 210}
	enemyBase    = color.NRGBA{220, 50, 50, 210}
	hpBack       = color.NRGBA{20, 20, 20, 220}
	hpFront      = color.NRGBA{90, 220, 110, 255}
	cursorColor  = color.NRGBA{255, 230, 80, 255}
	targetColor  = color.NRGBA{240, 80, 80, 255}
	dangerColor  = color.NRGBA{255, 150, 30, 255}
	// watchMoveColor と watchAttackColor は、自軍を選んでいる間に枠で描く、クリックして出している敵の移動と攻撃の範囲（DES-UI-27）。
	watchMoveColor   = color.NRGBA{200, 120, 255, 255}
	watchAttackColor = color.NRGBA{255, 70, 90, 255}
	watchEdge        = color.NRGBA{20, 10, 40, 230}
	bandColor        = color.NRGBA{0, 0, 0, 170}
	menuColor        = color.NRGBA{30, 36, 48, 235}
	menuBorder       = color.NRGBA{200, 210, 220, 255}
	strikeHit        = color.NRGBA{255, 240, 120, 255}
	strikeMiss       = color.NRGBA{200, 200, 200, 255}
	healTextClr      = color.NRGBA{120, 255, 150, 255}
	overShade        = color.NRGBA{0, 0, 0, 150}
	textShadow       = color.NRGBA{0, 0, 0, 255}
	// hintColor は増援の気配の黄土色。破線は hintAlpha を掛けて薄く描く（DES-UI-24）。
	hintColor = color.NRGBA{200, 160, 80, 255}
)

// hintAlpha は予兆の破線と印の濃さ。地形と範囲の塗りを隠さないよう低くする（DES-UI-24）。
const hintAlpha = 0.6

const (
	windowTitle = "レンキン・タクティクス"
	// lineHeight は案内欄の 1 行の高さ（px）。
	lineHeight = 16
	// actedScale は行動済みの自軍を暗く描くときの明るさ。
	actedScale = 0.45
)

// infoMargin は情報欄の余白（px）。
const infoMargin = 8

// 情報欄の能力の棒とスキルの名前の色（DES-UI-06）と、相性の印の色（DES-UI-07）。
var (
	statBarColor = color.NRGBA{110, 170, 255, 255}
	skillColor   = color.NRGBA{255, 240, 60, 255}
	goodColor    = color.NRGBA{90, 230, 110, 255}
	badColor     = color.NRGBA{255, 80, 80, 255}
)

// infoStyleColor は情報欄の文の色の種類を色に読み替える（DES-UI-06、DES-UI-07）。
func infoStyleColor(s InfoStyle) color.Color {
	switch s {
	case StyleRewindEnemy:
		return color.NRGBA{R: 255, G: 100, B: 100, A: 128}
	case StyleRewindBoundary:
		return color.NRGBA{R: 180, G: 180, B: 180, A: 128}
	case StyleGood:
		return goodColor
	case StyleBad:
		return badColor
	case StyleBlack:
		return color.Black
	case StyleTitle:
		return gold
	case StyleLabel:
		return muted
	case StyleSkill:
		return skillColor
	case StyleHint:
		return hintColor
	}
	return white
}

// targetFrameColor は攻撃・回復の候補枠の色を返し、選んだ相手をカーソル色で示す（DES-UI-02、DES-UI-18）。
func (c *Controller) targetFrameColor(target int) color.Color {
	if target == c.target {
		return cursorColor
	}
	if c.TargetStyle() == StyleGood {
		return goodColor
	}
	return targetColor
}

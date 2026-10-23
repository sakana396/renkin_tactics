package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// Camera は表示左上と盤面寸法をマス単位で保持する（DES-UI-25）。保存や戦闘の状態には含めない。
type Camera struct{ X, Y, Width, Height int }

// Move は盤面の端で止まる整数移動（DES-UI-25）。
func (c *Camera) Move(dx, dy int) {
	c.X = max(0, min(max(0, c.Width-16), c.X+dx))
	c.Y = max(0, min(max(0, c.Height-16), c.Y+dy))
}

// Focus は対象を見せる最小の移動だけ行う（DES-UI-25）。
func (c *Camera) Focus(p tactics.Pos) {
	if p.X < c.X {
		c.X = p.X
	} else if p.X >= c.X+16 {
		c.X = p.X - 15
	}
	if p.Y < c.Y {
		c.Y = p.Y
	} else if p.Y >= c.Y+16 {
		c.Y = p.Y - 15
	}
	c.Move(0, 0)
}

// CellAt は表示領域と盤内を検証して世界座標へ変換する（DES-UI-25）。
func (c Camera) CellAt(x, y int) (tactics.Pos, bool) {
	if x < 0 || y < 0 || x >= MapSize || y >= MapSize {
		return tactics.Pos{}, false
	}
	p := tactics.Pos{X: x/CellSize + c.X, Y: y/CellSize + c.Y}
	return p, p.X >= 0 && p.Y >= 0 && p.X < c.Width && p.Y < c.Height
}

// Screen は世界座標の描画位置を返す（DES-UI-25）。
func (c Camera) Screen(p tactics.Pos) image.Point {
	return image.Pt((p.X-c.X)*CellSize, (p.Y-c.Y)*CellSize)
}

// Visible は盤面の表示対象を判定する（DES-UI-25）。
func (c Camera) Visible(p tactics.Pos) bool {
	return p.X >= c.X && p.Y >= c.Y && p.X < min(c.Width, c.X+16) && p.Y < min(c.Height, c.Y+16)
}

// MenuRects は描画と入力で共通のメニュー矩形を返す（DES-UI-25）。
func (c Camera) MenuRects(p tactics.Pos, n int) []image.Rectangle {
	return MenuRects(tactics.Pos{X: p.X - c.X, Y: p.Y - c.Y}, n)
}

const cameraGuide = "矢印／ホイールで盤面移動"
const cameraGuideY = ScreenHeight - lineHeight

// Camera は履歴閲覧中なら専用カメラを返す（DES-UI-25）。
func (c *Controller) Camera() Camera {
	if c.rewind.open {
		return c.rewind.camera
	}
	return c.camera
}

// Scroll は入力待ちだけ盤面移動を受け付ける（DES-UI-25）。
func (c *Controller) Scroll(dx, dy int) {
	if c.rewind.open {
		if !c.rewind.confirm && (len(c.battle.History()) == 0 || c.previewOK()) {
			c.rewind.camera.Move(dx, dy)
		}
		return
	}
	if c.confirmEnd {
		return
	}
	switch c.state {
	case Idle, Selected, Targeting, HealTargeting:
		c.camera.Move(dx, dy)
	}
}

// Wheel はカーソル位置で送り先を一つに決める（DES-UI-25）。
func (c *Controller) Wheel(x, y int, delta float64, shift bool) bool {
	if delta == 0 {
		return false
	}
	step := 1
	if delta > 0 {
		step = -1
	}
	if x >= 0 && x < MapSize && y >= 0 && y < MapSize {
		if shift {
			c.Scroll(step, 0)
		} else {
			c.Scroll(0, step)
		}
		return true
	}
	if c.rewind.open && x >= MapSize && x < ScreenWidth && y >= 0 && y < ScreenHeight {
		c.RewindDetail(step)
		return true
	}
	return false
}

// focusHero は新しい盤面の主人公へ寄る（DES-UI-25）。
func (c *Camera) focusHero(units []tactics.Unit) {
	for _, u := range units {
		if u.Side == tactics.Player && u.Lord && u.Alive {
			c.Focus(u.Pos)
			return
		}
	}
}

// focusEvent は先行した Battle を参照せず出来事の写しだけから寄る（DES-UI-25）。
func (c *Controller) focusEvent(e tactics.Event) {
	switch e.Kind {
	case tactics.EventMove, tactics.EventReinforce:
		c.camera.Focus(e.To)
	case tactics.EventStrike:
		if e.Unit >= 0 && e.Unit < len(e.Units) {
			c.camera.Focus(e.Units[e.Unit].Pos)
		}
		if e.Target >= 0 && e.Target < len(e.Units) {
			c.camera.Focus(e.Units[e.Target].Pos)
		}
	case tactics.EventHeal, tactics.EventFall, tactics.EventWait, tactics.EventExp, tactics.EventLevelUp:
		if e.Unit >= 0 && e.Unit < len(e.Units) {
			c.camera.Focus(e.Units[e.Unit].Pos)
		}
	}
}

// keyRepeat は押し始めと12フレーム後から4フレームごとを判定する（DES-UI-25）。
func keyRepeat(duration int) bool { return duration == 1 || duration >= 13 && (duration-13)%4 == 0 }

// focusHistory は選択した before の対象を優先する（DES-UI-25）。
func (c *Controller) focusHistory(cam *Camera, units []tactics.Unit) {
	h := c.battle.History()
	i := c.rewind.selected
	if i < 0 || i >= len(h) {
		return
	}
	focusHistoryEntry(cam, h[i], units)
}

// focusHistoryEntry は復元で履歴が削られた後も、保持した行から寄る（DES-UI-25）。
func focusHistoryEntry(cam *Camera, e tactics.HistoryEntry, units []tactics.Unit) {
	if e.Kind != tactics.HistoryAction {
		return
	}
	if e.Unit >= 0 && e.Unit < len(units) {
		cam.Focus(units[e.Unit].Pos)
	}
	if (e.Action == tactics.Attack || e.Action == tactics.Heal) && e.Target >= 0 && e.Target < len(units) {
		cam.Focus(units[e.Target].Pos)
	}
}

// Popup はマス上の文字を盤面内へ寄せる共通配置（DES-UI-25）。
func (c Camera) Popup(p tactics.Pos, width int) (image.Point, bool) {
	if !c.Visible(p) {
		return image.Point{}, false
	}
	pt := c.Screen(p)
	x := max(0, min(pt.X+CellSize/2-width/2, MapSize-width))
	y := pt.Y - 16
	if y < 0 {
		y = pt.Y + CellSize
	}
	return image.Pt(x, y), true
}

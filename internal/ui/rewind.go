package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// rewindMenu は入力を保持して重ねる一覧と確認（DES-UI-19）。
type rewindMenu struct {
	detailOffset  int
	open, confirm bool
	selected      int
	message       string
}

// OpenRewind は再生完了後だけ一覧を開く。閲覧では戦闘と選択を変えない（DES-UI-19）。
func (c *Controller) OpenRewind() {
	if c.state == Playing || c.rewind.open {
		return
	}
	c.rewind = rewindMenu{open: true, selected: len(c.battle.History()) - 1}
}

// RewindMove は上下とページ移動を先頭・末尾で止める（DES-UI-19）。
func (c *Controller) RewindMove(delta int) {
	if !c.rewind.open || c.rewind.confirm {
		return
	}
	n := len(c.battle.History())
	if n == 0 {
		return
	}
	c.rewind.selected = max(0, min(n-1, c.rewind.selected+delta))
	c.rewind.detailOffset = 0
}

// RewindEnter は確認を経て一度だけ巻き戻す。消える項目の境界を先に保持する（DES-UI-19）。
func (c *Controller) RewindEnter() {
	if !c.rewind.open || c.battle.RewindsLeft() == 0 {
		return
	}
	h := c.battle.History()
	i := c.rewind.selected
	if i < 0 || i >= len(h) {
		return
	}
	if !c.rewind.confirm {
		c.rewind.confirm = true
		return
	}
	entry := h[i]
	if err := c.battle.Rewind(entry.ID); err != nil {
		c.rewind.confirm = false
		c.rewind.message = err.Error()
		return
	}
	clear(c.record[min(len(c.record), entry.EventStart):])
	c.record = c.record[:min(len(c.record), entry.EventStart)]
	c.resumeEnemy = entry.Kind == tactics.HistoryAction && entry.Side == tactics.Enemy
	c.rewind = rewindMenu{}
	c.events, c.index, c.frame = nil, 0, 0
	c.unit, c.target, c.watched = -1, -1, -1
	c.dest = tactics.Pos{}
	c.confirmEnd, c.fast, c.message = false, false, ""
	c.shownTurn, c.shownSide = c.battle.Turn, c.battle.Phase
	c.state = Idle
}

// Resume は続きだけを実行し、通常と同じ出来事再生へ渡す（DES-UI-19）。
func (c *Controller) Resume() {
	if c.rewind.open || c.state == Playing || !c.battle.NeedsResume() {
		return
	}
	events, err := c.battle.Resume()
	if err != nil {
		c.message = err.Error()
		return
	}
	c.play(events)
}

// rewindLabel は描画と独立した行動の全文。移動だけの敵も出来事の境界から区別する（DES-UI-19）。
func (c *Controller) rewindLabel(e tactics.HistoryEntry) string {
	action := actionName(e.Action)
	switch e.Kind {
	case tactics.HistoryEndPhase:
		action = "番を終える"
		if e.Automatic {
			action = "番の移行（自動）"
		}
	case tactics.HistoryPlayerStart:
		action = "自軍の番を始める"
	default:
		for j := e.EventStart; j < e.EventEnd && j < len(c.record); j++ {
			if c.record[j].Kind == tactics.EventMove && c.record[j].Unit == e.Unit && c.record[j].To == e.To {
				action = "移動・" + action
				break
			}
		}
	}
	target := ""
	if e.TargetName != "" {
		target = " → " + e.TargetName
	}
	return fmt.Sprintf("%dT %s %s %s%s", e.Turn, sideName(e.Side), e.UnitName, action, target)
}

// fitRewind は行を欄幅で省略し、詳細は別に全文を折り返す（DES-UI-19）。
func fitRewind(s string, width int) string {
	if textWidth(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r)+"…") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// RewindView は 8 行の一覧、確認、回数、入口と再開の配置（DES-UI-19）。
func (c *Controller) RewindView() SceneView {
	var v SceneView
	add := func(s string, x, y int) { v.Texts = append(v.Texts, SceneText{Text: s, X: x, Y: y}) }
	button := func(s string, r image.Rectangle, on bool) { v.Buttons = append(v.Buttons, SceneButton{s, r, on}) }
	if !c.rewind.open {
		if c.state == Playing {
			return v
		}
		button(fmt.Sprintf("巻き戻し 残り %d/10", c.battle.RewindsLeft()), image.Rect(304, 512, 504, 530), true)
		add("T: 巻き戻し", 216, 514)
		if c.battle.NeedsResume() {
			label := "番の処理を再開"
			if c.resumeEnemy {
				label = "敵軍の行動を再開"
			}
			button(label, image.Rect(144, 550, 464, 574), true)
		}
		return v
	}
	add(fmt.Sprintf("巻き戻し 残り %d/10", c.battle.RewindsLeft()), 48, 32)
	h := c.battle.History()
	n := c.battle.RewindsLeft()
	if c.rewind.confirm {
		add(fmt.Sprintf("この行動から先を取り消して戻りますか。残り %d → %d", n, n-1), 48, 112)
		button("やめる", image.Rect(48, 480, 208, 520), true)
		button("戻す", image.Rect(592, 480, 784, 520), n > 0)
	} else {
		add("選んだ行動の「前」に戻る", 48, 64)
		start := max(0, c.rewind.selected/8*8)
		for i := start; i < min(len(h), start+8); i++ {
			r := image.Rect(48, 96+(i-start)*30, 784, 124+(i-start)*30)
			v.Rows = append(v.Rows, PrepareRow{Rect: r, Selected: i == c.rewind.selected})
			add(fitRewind(c.rewindLabel(h[i]), 720), 56, r.Min.Y+6)
		}
		if len(h) == 0 {
			add("戻せる行動がありません", 48, 112)
		}
		button("前", image.Rect(48, 344, 144, 380), start > 0)
		button("次", image.Rect(160, 344, 256, 380), start+8 < len(h))
		button("戻る", image.Rect(48, 520, 208, 560), true)
		button("この前へ", image.Rect(592, 520, 784, 560), n > 0 && len(h) > 0)
		if c.rewind.selected >= 0 && c.rewind.selected < len(h) {
			for i, s := range wrapSceneText(c.rewindLabel(h[c.rewind.selected]))[c.rewind.detailOffset:] {
				if i >= 5 {
					break
				}
				add(s, 48, 388+i*20)
			}
		}
		add("詳細: マウスホイールで全文を読む", 280, 372)
		if n == 0 {
			add("残り回数が 0 のため巻き戻せません", 280, 352)
		}
	}
	if c.rewind.message != "" {
		add(c.rewind.message, 48, 496)
	}
	return v
}

// RewindClick は表示と同じ矩形を使い、一覧の外への入力を通さない（DES-UI-19）。
func (c *Controller) RewindClick(x, y int) bool {
	v := c.RewindView()
	p := image.Pt(x, y)
	for _, b := range v.Buttons {
		if p.In(b.Rect) {
			if b.Enabled {
				switch {
				case strings.HasPrefix(b.Label, "巻き戻し"):
					c.OpenRewind()
				case b.Label == "前":
					c.RewindMove(-8)
				case b.Label == "次":
					c.RewindMove(8)
				case b.Label == "戻る" || b.Label == "やめる":
					c.Cancel()
				case b.Label == "戻す" || b.Label == "この前へ":
					c.RewindEnter()
				default:
					c.Resume()
				}
			}
			return true
		}
	}
	if c.rewind.open {
		for i, r := range v.Rows {
			if p.In(r.Rect) {
				c.rewind.selected = c.rewind.selected/8*8 + i
				c.rewind.detailOffset = 0
			}
		}
		return true
	}
	return false
}

// syncRewind は敗北の選択から成功したときだけ戦闘に復帰する（DES-UI-19、6 章）。
func (s *SceneController) syncRewind() {
	if s.scene == GameOver && s.battle.Battle().Result == tactics.Ongoing {
		s.scene = Battle
		s.message = ""
	}
}

// RewindDetail は長い行動名の全文を 5 行ずつ読めるようにする（DES-UI-19）。
func (c *Controller) RewindDetail(delta int) {
	h := c.battle.History()
	if !c.rewind.open || c.rewind.confirm || c.rewind.selected < 0 || c.rewind.selected >= len(h) {
		return
	}
	n := len(wrapSceneText(c.rewindLabel(h[c.rewind.selected])))
	c.rewind.detailOffset = max(0, min(max(0, n-5), c.rewind.detailOffset+delta))
}

// rewindKey はキーの押し始めだけを受け、一覧と確認の入力を読み替える（DES-UI-19）。
func (c *Controller) rewindKey(key string) {
	switch key {
	case "T":
		c.OpenRewind()
	case "Enter":
		if c.rewind.open {
			c.RewindEnter()
		} else {
			c.Resume()
		}
	case "Up":
		c.RewindMove(-1)
	case "Down":
		c.RewindMove(1)
	case "PageUp":
		c.RewindMove(-8)
	case "PageDown":
		c.RewindMove(8)
	}
}

package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestRewindUI(t *testing.T) {
	t.Run("TC-FR-BTL-10-11 一覧の境界と確認・取消し・回数", func(t *testing.T) {
		c := newIdle(t)
		c.rewindKey("T")
		if !strings.Contains(viewStrings(c.RewindView()), "戻せる行動がありません") {
			t.Fatal(c.RewindView())
		}
		c.RewindEnter()
		c.Cancel()
		for i := 0; i < 3; i++ {
			c.EndPhase()
			c.EndPhase()
			playOut(c)
		}
		h := c.battle.History()
		before := c.battle.Units()
		c.OpenRewind()
		if c.rewind.selected != len(h)-1 || len(c.RewindView().Rows) > 8 {
			t.Fatal("最新ページでない")
		}
		c.rewindKey("PageUp")
		c.rewindKey("PageUp")
		c.rewindKey("PageUp")
		if c.rewind.selected != 0 || len(c.RewindView().Rows) != 8 {
			t.Fatal("先頭ページ")
		}
		c.rewindKey("PageDown")
		if c.rewind.selected != 8 {
			t.Fatal("次ページ")
		}
		c.RewindMove(100)
		if c.rewind.selected != len(h)-1 {
			t.Fatal("末尾")
		}
		c.rewindSelect(0)
		c.RewindEnter()
		if !c.rewind.confirm {
			t.Fatal("確認なし")
		}
		c.Cancel()
		if !c.rewind.open || c.rewind.confirm {
			t.Fatal("確認取消し")
		}
		c.Cancel()
		if !reflect.DeepEqual(h, c.battle.History()) || !reflect.DeepEqual(before, c.battle.Units()) || c.battle.RewindsLeft() != 10 {
			t.Fatal("閲覧が戦闘を変更")
		}
		for i := 0; i < 10; i++ {
			c.OpenRewind()
			consumeExplicitRewind(t, c, i)
		}
		c.OpenRewind()
		selectExplicitRewind(t, c)
		c.RewindEnter()
		if c.rewind.confirm || !strings.Contains(viewStrings(c.RewindView()), "残り回数が 0") {
			t.Fatal("0 回の表示")
		}
		if buttonOf(t, c.RewindView(), "決定").Enabled {
			t.Fatal("0 回で確定可能")
		}
	})
	t.Run("TC-FR-BTL-10-12 / TC-IF-01-12 / TC-FR-UNT-09-12 再生中拒否と選択保持・成功時の表示破棄", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		c.rewindKey("T")
		c.RewindClick(310, 520)
		if c.rewind.open {
			t.Fatal("Playing で開いた")
		}
		c.SetFastForward(true)
		playOut(c)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ToggleDanger()
		state, unit, dest := c.state, c.unit, c.dest
		c.RewindClick(310, 520)
		c.Cancel()
		if c.state != state || c.unit != unit || c.dest != dest {
			t.Fatal("選択消失")
		}
		c.OpenRewind()
		entry := c.battle.History()[0]
		c.RewindMove(-100)
		c.RewindEnter()
		c.RewindEnter()
		c.RewindEnter()
		if c.battle.RewindsLeft() != 9 || c.state != Idle || c.target != -1 || c.watched != -1 || c.confirmEnd || c.events != nil || len(c.record) != entry.EventStart || !c.danger || c.rewind.open {
			t.Fatal("成功時の破棄または二重消費", c)
		}
		for _, b := range c.RewindView().Buttons {
			if !b.Rect.In(image.Rect(0, 0, ScreenWidth, ScreenHeight)) {
				t.Fatal("画面外", b)
			}
		}
	})
	t.Run("TC-FR-BTL-10-13 敵軍行動を閲覧できるが決定できない", func(t *testing.T) {
		c := newIdle(t)
		endTurn(c)
		c.OpenRewind()
		c.rewindSelect(historyIndex(t, c.battle.History(), tactics.HistoryAction, tactics.Enemy))
		checkReadOnlyRewind(t, c)
		c.Cancel()
		checkInputLayout(t, c)
	})
	t.Run("TC-FR-BTL-10-11 行の移動と長い名前・全文詳細", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		e := c.battle.History()[0]
		if !strings.Contains(c.rewindLabel(e), "移動・待機") {
			t.Fatal(c.rewindLabel(e))
		}
		e.UnitName = strings.Repeat("長い名前", 100)
		if textWidth(fitRewind(c.rewindLabel(e), 720)) > 720 {
			t.Fatal("行の省略なし")
		}
	})
}

func TestRewindScene(t *testing.T) {
	t.Run("TC-FR-BTL-10-13 / TC-IF-01-13 / TC-FR-CMP-02-02 勝利の帯から取消しと復帰・反映後は入口なし", func(t *testing.T) {
		m := &memoryStore{}
		s := fixtureScenes(m, false, false)
		startScenes(s)
		winScene(t, s)
		playOut(s.battle)
		s.battle.OpenRewind()
		s.Advance()
		s.battle.Cancel()
		if s.Scene() != Battle || s.battle.state != Over || m.writes != 1 {
			t.Fatal("取消しで章が進行")
		}
		s.battle.OpenRewind()
		s.battle.RewindEnter()
		s.battle.RewindEnter()
		if s.battle.state == Over || s.battle.battle.Result != tactics.Ongoing || s.campaign.Chapter != 0 || m.writes != 1 {
			t.Fatal("反映前復帰失敗")
		}
		// 同じ行動をやり直し、最終的な勝利だけを反映する。
		s.battle.Click(tactics.Pos{})
		s.battle.Click(tactics.Pos{})
		s.battle.ChooseAction(tactics.Attack)
		s.battle.Click(tactics.Pos{X: 1})
		s.battle.Click(tactics.Pos{X: 1})
		playOut(s.battle)
		s.Advance()
		if s.Scene() != Result {
			t.Fatal("勝利再反映", s.Scene())
		}
		for _, b := range s.View().Buttons {
			if strings.Contains(b.Label, "巻き戻") {
				t.Fatal("反映後の入口")
			}
		}
	})
	t.Run("TC-FR-BTL-10-13 / TC-FR-CMP-09-07 敗北の選択を保ってから復帰", func(t *testing.T) {
		m := &memoryStore{}
		s := fixtureScenes(m, false, false)
		startScenes(s)
		s.campaign.Roster[0].Stats.HP = 1
		s.campaign.Roster[0].Stats.Def = 0
		clickSceneButton(t, s, "配置へ")
		clickSceneButton(t, s, "戦闘開始")
		playOut(s.battle)
		s.battle.EndPhase()
		s.battle.EndPhase()
		playOut(s.battle)
		s.Advance()
		if s.Scene() != GameOver {
			t.Fatal("敗北なし")
		}
		s.Choose(2)
		s.battle.Cancel()
		s.syncRewind()
		if s.Scene() != GameOver || s.battle.state != Over {
			t.Fatal("取消しが場面変更")
		}
		s.Choose(2)
		s.battle.RewindMove(-100)
		s.battle.RewindEnter()
		s.battle.RewindEnter()
		s.syncRewind()
		if s.Scene() != Battle || s.battle.state == Over || !s.battle.battle.Units()[0].Alive || s.battle.battle.Units()[0].Acted || s.battle.battle.NeedsResume() || s.battle.battle.RewindsLeft() != 9 || m.writes != 1 {
			t.Fatal("敗北復帰失敗")
		}
	})
}

func TestRewindBoundaries(t *testing.T) {
	t.Run("TC-FR-BTL-10-13 自動移行と自軍開始前を閲覧できるが決定できない", func(t *testing.T) {
		for _, kind := range []tactics.HistoryKind{tactics.HistoryEndPhase, tactics.HistoryPlayerStart} {
			c := newIdle(t)
			for _, p := range []tactics.Pos{{X: 0, Y: 0}, {X: 0, Y: 2}} {
				c.Click(p)
				c.Click(p)
				c.ChooseAction(tactics.Wait)
				playOut(c)
			}
			c.OpenRewind()
			c.rewindSelect(historyIndex(t, c.battle.History(), kind, tactics.Player))
			checkReadOnlyRewind(t, c)
			c.Cancel()
			checkInputLayout(t, c)
		}
	})
	t.Run("TC-FR-BTL-10-11 敵の移動だけと待機を区別し長い詳細を読める", func(t *testing.T) {
		c := newIdleWith(t, func() tactics.Setup {
			s := basicSetup()
			s.Units[3].Pos = tactics.Pos{X: 5, Y: 0}
			s.Units[3].Stats.Mov = 1
			s.Units[3].AI = tactics.Charge
			return s
		})
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		moved, waited := false, false
		for _, e := range c.battle.History() {
			if e.Side != tactics.Enemy || e.Kind != tactics.HistoryAction {
				continue
			}
			label := c.rewindLabel(e)
			if e.Unit == 3 {
				moved = strings.Contains(label, "移動・待機")
			}
			if e.Unit == 2 {
				waited = strings.Contains(label, "待機") && !strings.Contains(label, "移動")
			}
		}
		if !moved || !waited {
			t.Fatal("移動だけと待機の表示", c.battle.History())
		}
		// 長い名前は実際の履歴から表示し、詳細の末尾まで送れる。
		long := strings.Repeat("長い名前", 100) + "末尾"
		c = newIdleWith(t, func() tactics.Setup { s := basicSetup(); s.Units[1].Name = long; return s })
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.OpenRewind()
		for i := 0; i < 1000; i++ {
			c.Wheel(600, 400, -1, false)
		}
		if !strings.Contains(viewStrings(c.RewindView()), "末尾") {
			t.Fatal("詳細末尾なし")
		}
		for i := 0; i < 1000; i++ {
			c.Wheel(600, 400, 1, false)
		}
		if c.rewind.detailOffset != 0 {
			t.Fatal("詳細先頭")
		}
		for _, tx := range c.RewindView().Texts {
			if tx.X+textWidth(tx.Text) > ScreenWidth || tx.Y+lineHeight > ScreenHeight {
				t.Fatal("表示が窓外", tx)
			}
		}
	})
}

func TestRewindCancellation(t *testing.T) {
	t.Run("TC-FR-BTL-10-12 入力の各状態を保ち一覧中の通常入力を拒む", func(t *testing.T) {
		for _, state := range []State{Idle, Selected, Moved, Targeting, HealTargeting, ItemMenu, Over} {
			c := newIdle(t)
			c.state = state
			c.unit = 0
			c.dest = tactics.Pos{X: 0, Y: 1}
			c.target = 3
			c.watched = 2
			c.confirmEnd = true
			c.message = "保持する知らせ"
			c.danger = true
			before := *c
			c.OpenRewind()
			c.Click(tactics.Pos{X: 0, Y: 2})
			c.ChooseAction(tactics.Wait)
			c.ChooseItem(0)
			c.EndPhase()
			c.ToggleDanger()
			c.Cancel()
			if !reflect.DeepEqual(before, *c) {
				t.Fatalf("状態 %v の取消しで元の入力が変わった", state)
			}
		}
	})
}

func TestRewindMouse(t *testing.T) {
	t.Run("TC-FR-BTL-10-11 マウスで行選択と確認・やめる・戻す", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		c.RewindClick(310, 520)
		// 右欄の 1 行目（x=520〜824、y=40〜62）。DES-UI-21 の配置を、ミニマップと重ならないよう詰めた（DES-UI-28）。
		c.RewindClick(600, 50)
		if c.rewind.selected != 0 {
			t.Fatal("行を選べない")
		}
		// 「決定」は右欄の下端の右（x=680〜824、y=520〜560）。
		c.RewindClick(750, 540)
		if !c.rewind.confirm || c.battle.RewindsLeft() != 10 {
			t.Fatal("確認なしで消費")
		}
		// 「やめる」は右欄の下端の左（x=520〜664）。
		c.RewindClick(560, 540)
		if c.rewind.confirm || !c.rewind.open {
			t.Fatal("やめるが一覧に戻らない")
		}
		c.RewindClick(750, 540)
		c.RewindClick(750, 540)
		c.RewindClick(750, 540)
		if c.rewind.open || c.battle.RewindsLeft() != 9 {
			t.Fatal("マウスで復帰または消費が不正")
		}
	})
}

// 自軍入力待ちの全ボタンが盤面・案内文・互いを隠さないことを確かめる（DES-UI-19、UC-14）。
func checkInputLayout(t *testing.T, c *Controller) {
	t.Helper()
	v := c.RewindView()
	for i, b := range v.Buttons {
		if !b.Rect.In(image.Rect(0, MapSize, MapSize, ScreenHeight)) {
			t.Fatal("自軍入力待ちのボタンが盤面に重なるか案内欄の外にある", b)
		}
		for j, line := range c.Guide() {
			y := MapSize + 2 + j*lineHeight
			if b.Rect.Overlaps(image.Rect(8, y, 8+textWidth(line), y+lineHeight)) {
				t.Fatal("自軍入力待ちのボタンが案内文を隠す", b, line)
			}
		}
		for _, tx := range v.Texts {
			if b.Rect.Overlaps(image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)) {
				t.Fatal("自軍入力待ちのボタンが操作案内を隠す", b, tx)
			}
		}
		for _, other := range v.Buttons[i+1:] {
			if b.Rect.Overlaps(other.Rect) {
				t.Fatal("自軍入力待ちのボタン同士が重なる", b, other)
			}
		}
	}
}

func TestRewindSceneEntry(t *testing.T) {
	t.Run("TC-FR-BTL-10-12 巻き戻しのキーとボタンは戦闘と敗北の選択だけ", func(t *testing.T) {
		for _, scene := range []Scene{Title, RuleSelect, Story, Joined, Prepare, Deploy, Battle, Result, GameOver, Clear} {
			for _, input := range []string{"T", "ボタン"} {
				s := fixtureScenes(&memoryStore{}, true, true)
				startScenes(s)
				s.Cancel()
				clickSceneButton(t, s, "配置へ")
				s.storyChapter = 1
				// 以前の戦闘が残っていても、配置などの場面から開かない。
				c := newIdle(t)
				s.battle = c
				s.scene = scene
				if input == "T" {
					s.rewindKey("T")
				} else {
					r := c.RewindView().Buttons[0].Rect
					s.Click(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
				}
				want := scene == Battle || scene == GameOver
				if c.rewind.open != want {
					t.Fatalf("場面 %v、入力 %s: 開いた=%v、期待=%v", scene, input, c.rewind.open, want)
				}
			}
		}
	})
}

// consumeExplicitRewind は毎回作り直された明示の番終了を探し、成功と新しいIDを確認する。
func consumeExplicitRewind(t *testing.T, c *Controller, n int) {
	t.Helper()
	h := c.battle.History()
	i := -1
	for j, e := range h {
		if e.Kind == tactics.HistoryEndPhase && !e.Automatic {
			i = j
			break
		}
	}
	if i < 0 {
		t.Fatal("明示の番終了なし")
	}
	oldID := h[i].ID
	c.rewindSelect(i)
	c.RewindEnter()
	c.RewindEnter()
	if c.battle.NeedsResume() || c.state != Idle || c.rewind.open || c.battle.RewindsLeft() != 9-n {
		t.Fatal("1消費・自軍入力待ちでない")
	}
	// 同じ番終了を再実行し、古いIDを再利用しないことを確かめる。
	endTurn(c)
	h = c.battle.History()
	if len(h) <= i || h[i].ID <= oldID || h[i].Automatic {
		t.Fatal("番終了の再作成でIDを再利用")
	}
}

// checkReadOnlyRewind は選択・プレビューを保ち、クリックとEnterの双方を拒否することを確かめる。
func checkReadOnlyRewind(t *testing.T, c *Controller) {
	t.Helper()
	before, history, left := c.battle.Units(), c.battle.History(), c.battle.RewindsLeft()
	if !c.RewindBoard().Available || buttonOf(t, c.RewindView(), "決定").Enabled {
		t.Fatal("閲覧専用の盤面・決定")
	}
	if !strings.Contains(strings.Join(c.rewindReasons(), ""), "自軍の行動か『番を終える』の前だけ戻せます") {
		t.Fatal("対象外の理由なし")
	}
	for _, input := range []func(){func() { clickButton(t, c, "決定") }, func() { c.rewindKey("Enter") }, c.RewindEnter} {
		input()
		if c.rewind.confirm || !c.rewind.open || c.battle.RewindsLeft() != left || !reflect.DeepEqual(before, c.battle.Units()) || !reflect.DeepEqual(history, c.battle.History()) {
			t.Fatal("閲覧専用の行で確認・確定を受理")
		}
	}
	if strings.Contains(strings.Join(c.RewindBoard().Lines, ""), "再開") {
		t.Fatal("プレビューに再開案内")
	}
}

// selectExplicitRewind は回数・取得失敗による拒否を対象外の拒否と分けて確かめるため、明示の番終了を選ぶ。
func selectExplicitRewind(t *testing.T, c *Controller) {
	t.Helper()
	for i, h := range c.battle.History() {
		if h.Side == tactics.Player && h.Kind == tactics.HistoryEndPhase && !h.Automatic {
			c.rewindSelect(i)
			if !c.rewindAllowed() || !c.previewOK() {
				t.Fatal("前提: 決定対象のプレビューでない")
			}
			return
		}
	}
	t.Fatal("前提: 明示の番終了なし")
}

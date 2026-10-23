package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// previewUnits は tactics の PreviewRewind が返す行動前のユニット（DES-RWD-05）。UI の写しと比べる基準。
func previewUnits(t *testing.T, c *Controller, id uint64) []tactics.Unit {
	t.Helper()
	v, err := c.battle.PreviewRewind(id)
	if err != nil {
		t.Fatalf("PreviewRewind(%d): %v", id, err)
	}
	return v.Units()
}

// historyIndex は種類 kind（と陣営 side）の最初の行を返す。なければテストを止める。
func historyIndex(t *testing.T, h []tactics.HistoryEntry, kind tactics.HistoryKind, side tactics.Side) int {
	t.Helper()
	for i, e := range h {
		if e.Kind == kind && e.Side == side {
			return i
		}
	}
	t.Fatalf("履歴に種類 %v・陣営 %v の行がない: %v", kind, side, h)
	return -1
}

// longHistory は 3 回の番を回し、8 行を超える履歴を持つ Idle の Controller を返す。
func longHistory(t *testing.T) *Controller {
	t.Helper()
	c := newIdle(t)
	for i := 0; i < 3; i++ {
		c.EndPhase()
		c.EndPhase()
		playOut(c)
	}
	if n := len(c.battle.History()); n <= 8 {
		t.Fatalf("履歴 %d 行。8 行を超えるはず", n)
	}
	return c
}

// buttonOf は右欄のボタンを名前で探す。
func buttonOf(t *testing.T, v SceneView, label string) SceneButton {
	t.Helper()
	for _, b := range v.Buttons {
		if b.Label == label {
			return b
		}
	}
	t.Fatalf("ボタン %s がない: %v", label, v.Buttons)
	return SceneButton{}
}

func clickButton(t *testing.T, c *Controller, label string) {
	t.Helper()
	b := buttonOf(t, c.RewindView(), label)
	c.RewindClick(b.Rect.Min.X+1, b.Rect.Min.Y+1)
}

func TestRewindPreviewSelect(t *testing.T) {
	t.Run("TC-FR-BTL-10-24 / TC-FR-BTL-10-28 開いた直後・上下・行クリック・8 行境界・ページ移動で選んだ行の前の盤面に切り替える", func(t *testing.T) {
		c := longHistory(t)
		h := c.battle.History()
		last := len(h) - 1
		c.OpenRewind()
		check := func(step string, want int) {
			t.Helper()
			if c.rewind.selected != want {
				t.Fatalf("%s: 選択 %d、期待 %d", step, c.rewind.selected, want)
			}
			v := c.RewindBoard()
			if !v.Visible || !v.Available || !reflect.DeepEqual(v.Units, previewUnits(t, c, h[want].ID)) {
				t.Fatalf("%s: 行 %d の行動前の盤面でない", step, want)
			}
			if p, _ := c.battle.PreviewRewind(h[want].ID); v.Board.Turn != p.Turn || v.Board.Phase != p.Phase || v.Board.Width() != 6 {
				t.Fatalf("%s: 盤面のターン・陣営・地形が違う", step)
			}
		}
		check("開いた直後は最新行", last)
		c.rewindKey("Up")
		check("上", last-1)
		c.rewindKey("Down")
		check("下", last)
		c.RewindMove(100)
		check("末尾で止まる", last)
		c.rewindKey("PageUp")
		c.rewindKey("PageUp")
		c.rewindKey("PageUp")
		check("先頭ページ", 0)
		c.RewindMove(-1)
		check("先頭で止まる", 0)
		// 1 ページ目の 8 行目（y=256〜284）をクリックし、下で 9 行目（2 ページ目の先頭）へ移る。
		c.RewindClick(600, rewindRowTop+7*rewindRowH+4)
		check("行クリック", 7)
		if c.rewind.confirm || c.battle.RewindsLeft() != 10 {
			t.Fatal("行クリックで確定した")
		}
		c.rewindKey("Down")
		check("8 行の境界", 8)
		if rows := c.RewindView().Rows; len(rows) == 0 || !rows[0].Selected {
			t.Fatal("2 ページ目の先頭が選ばれていない")
		}
		clickButton(t, c, "前")
		check("前のページ", 0)
		clickButton(t, c, "次")
		check("次のページ", 8)
		c.rewindKey("PageDown")
		check("PageDown", min(16, last))
	})
	t.Run("TC-FR-BTL-10-24 移動前の位置と行動者の枠・同じ行では取得し直さない", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		if c.battle.Units()[1].Pos != (tactics.Pos{X: 1, Y: 2}) {
			t.Fatal("前提: ガルドが (1, 2) にいない")
		}
		c.OpenRewind()
		v := c.RewindBoard()
		// 行動の前はガルドが (0, 2) にいて、行動済みでない（basicSetup）。
		if v.Units[1].Pos != (tactics.Pos{X: 0, Y: 2}) || v.Units[1].Acted || !v.HasFrame || v.Frame != (tactics.Pos{X: 0, Y: 2}) {
			t.Fatal("移動前の盤面と行動者の枠でない", v.Units[1], v.Frame)
		}
		if v.Board.Turn != 1 || v.Board.Phase != tactics.Player || !strings.Contains(strings.Join(v.Lines, "\n"), "プレビュー中・未確定") || !strings.Contains(strings.Join(v.Lines, "\n"), "盤面: 1T 自軍") {
			t.Fatal("案内", v.Lines)
		}
		loads := c.rewind.previewLoads
		for i := 0; i < 5; i++ {
			c.RewindBoard()
			c.RewindView()
			c.RewindMove(1)
		}
		if c.rewind.previewLoads != loads || loads != 1 {
			t.Fatalf("同じ行の描画・移動で取得し直した: %d → %d", loads, c.rewind.previewLoads)
		}
	})
	t.Run("TC-FR-BTL-10-24 番開始前は実際のターン・陣営を示し、行のラベルは次の自軍ターンを保つ", func(t *testing.T) {
		c := newIdle(t)
		// 1 行目は自軍の行動（入力待ちの前）、次に番を終える。
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		h := c.battle.History()
		i := historyIndex(t, h, tactics.HistoryPlayerStart, tactics.Player)
		c.OpenRewind()
		c.rewindSelect(i)
		if label := c.rewindLabel(h[i]); !strings.HasPrefix(label, "2T 自軍") || !strings.Contains(label, "自軍の番を始める") {
			t.Fatal("行のラベル", label)
		}
		lines := strings.Join(c.RewindBoard().Lines, "\n")
		if !strings.Contains(lines, "盤面: 1T 敵軍") || strings.Contains(lines, "再開") || c.RewindBoard().HasFrame {
			t.Fatal("番開始前の盤面の案内", lines)
		}
		e := historyIndex(t, h, tactics.HistoryAction, tactics.Enemy)
		c.rewindSelect(e)
		if lines := strings.Join(c.RewindBoard().Lines, "\n"); !strings.Contains(lines, "盤面: 1T 敵軍") || strings.Contains(lines, "再開") {
			t.Fatal("敵軍の行動前の案内", lines)
		}
		if h[0].Kind != tactics.HistoryAction || h[0].Side != tactics.Player {
			t.Fatal("前提: 1 行目が自軍の行動でない")
		}
		c.rewindSelect(0)
		if lines := strings.Join(c.RewindBoard().Lines, "\n"); !strings.Contains(lines, "盤面: 1T 自軍") || strings.Contains(lines, "再開") {
			t.Fatal("自軍の入力待ちの前は再開の案内を出さない", lines)
		}
	})
	t.Run("TC-FR-BTL-10-24 履歴なしは今の盤面を静止して示し、決定できない", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		want := c.Units()
		c.OpenRewind()
		v := c.RewindBoard()
		if !v.Available || !reflect.DeepEqual(v.Units, want) || v.Units[1].Pos != (tactics.Pos{X: 1, Y: 2}) || v.HasFrame {
			t.Fatal("履歴なしで元の盤面を出さない", v.Units)
		}
		if !strings.Contains(strings.Join(v.Lines, "\n"), "戻せる行動がありません") || !strings.Contains(viewStrings(c.RewindView()), "戻せる行動がありません") {
			t.Fatal("空の案内", v.Lines)
		}
		if buttonOf(t, c.RewindView(), "決定").Enabled {
			t.Fatal("空で決定できる")
		}
		c.RewindEnter()
		if c.rewind.confirm {
			t.Fatal("空で確認に進んだ")
		}
	})
	t.Run("TC-FR-BTL-10-24 取得に失敗したら古い盤面を見せず決定を無効にし、別の行で戻る", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		c.OpenRewind()
		if !c.RewindBoard().Available {
			t.Fatal("前提: 最新行の盤面がない")
		}
		// 実行済みでない ID の取得は ErrInvalidRewind で失敗する（DES-RWD-05）。
		c.fetchPreview(1 << 40)
		v := c.RewindBoard()
		if v.Available || v.Units != nil || v.Notice != "プレビューを表示できません" || !strings.Contains(strings.Join(v.Lines, "\n"), "表示できません") {
			t.Fatal("失敗時に古い盤面か案内がない", v)
		}
		if buttonOf(t, c.RewindView(), "決定").Enabled || !strings.Contains(viewStrings(c.RewindView()), "プレビューを表示できない") {
			t.Fatal("失敗時に決定できる")
		}
		c.RewindEnter()
		c.RewindEnter()
		if c.rewind.confirm || c.battle.RewindsLeft() != 10 {
			t.Fatal("失敗時に確定した")
		}
		c.rewindKey("Up")
		if v := c.RewindBoard(); !v.Available || !reflect.DeepEqual(v.Units, previewUnits(t, c, c.battle.History()[c.rewind.selected].ID)) {
			t.Fatal("別の行で戻らない")
		}
	})
	t.Run("TC-FR-BTL-10-24 残り 0 回でも選択とプレビューができ、決定は無効", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		for i := 0; i < 10; i++ {
			c.OpenRewind()
			consumeExplicitRewind(t, c, i)
		}
		if c.battle.RewindsLeft() != 0 {
			t.Fatal("前提: 残り 0 回でない", c.battle.RewindsLeft())
		}
		h := c.battle.History()
		c.OpenRewind()
		c.rewindKey("Up")
		if v := c.RewindBoard(); !v.Available || !reflect.DeepEqual(v.Units, previewUnits(t, c, h[len(h)-2].ID)) {
			t.Fatal("0 回で閲覧できない")
		}
		c.RewindEnter()
		if c.rewind.confirm || buttonOf(t, c.RewindView(), "決定").Enabled || !strings.Contains(viewStrings(c.RewindView()), "残り回数が 0") {
			t.Fatal("0 回の決定が無効でない")
		}
		if !reflect.DeepEqual(h, c.battle.History()) {
			t.Fatal("0 回の閲覧で履歴が変わった")
		}
	})
}

func TestRewindPreviewConfirm(t *testing.T) {
	t.Run("TC-FR-BTL-10-25 仮移動・番終了の確認から閲覧して取り消すと元の入力と表示に戻る", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		// 仮移動（Moved）: ガルドを (1, 2) に仮に置いたまま一覧を開く。
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		before, shown := *c, c.Units()
		c.OpenRewind()
		if v := c.RewindBoard(); v.Units[1].Pos != (tactics.Pos{X: 0, Y: 2}) || v.Units[0].Acted {
			t.Fatal("一覧に仮移動か今の盤面が混ざる", v.Units)
		}
		c.RewindEnter()
		c.Cancel()
		if !c.rewind.open || c.rewind.confirm {
			t.Fatal("確認の取消しで一覧に戻らない")
		}
		c.Cancel()
		if c.rewind.open || c.state != Moved || !reflect.DeepEqual(shown, c.Units()) || c.RewindBoard().Visible {
			t.Fatal("一覧の取消しで仮移動の表示に戻らない")
		}
		after := *c
		before.rewind, after.rewind = rewindMenu{}, rewindMenu{}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("元の入力が変わった")
		}
		// 番終了の確認: 1 回目の E のあとに開いて取り消す。
		c.Cancel()
		c.Cancel()
		c.EndPhase()
		if !c.confirmEnd {
			t.Fatal("前提: 番終了の確認でない")
		}
		msg := c.message
		c.OpenRewind()
		c.Cancel()
		if !c.confirmEnd || c.message != msg || c.state != Idle {
			t.Fatal("番終了の確認が消えた")
		}
	})
	t.Run("TC-FR-BTL-10-25 確認の取消しは選択とプレビューを保ち、盤面の入力を通さない", func(t *testing.T) {
		c := longHistory(t)
		h := c.battle.History()
		c.OpenRewind()
		c.rewindSelect(0)
		c.RewindEnter()
		if !c.rewind.confirm || !c.RewindBoard().Available || !strings.Contains(strings.Join(c.RewindBoard().Lines, "\n"), "一覧へ") {
			t.Fatal("確認中にプレビューがない")
		}
		loads := c.rewind.previewLoads
		c.Cancel()
		if c.rewind.selected != 0 || c.rewind.previewID != h[0].ID || c.rewind.previewLoads != loads {
			t.Fatal("確認の取消しで選択かプレビューが変わった")
		}
		// 盤面のクリック・行動・番終了・Q・再開は受け付けない。盤面のクリックは一覧にも透過しない。
		units := c.battle.Units()
		if !c.RewindClick(16, 16) || c.rewind.selected != 0 {
			t.Fatal("盤面のクリックが一覧へ透過した")
		}
		c.Click(tactics.Pos{})
		c.ChooseAction(tactics.Wait)
		c.EndPhase()
		danger := c.danger
		c.ToggleDanger()
		c.rewindKey("Enter")
		if c.danger != danger || c.state == Playing || !reflect.DeepEqual(units, c.battle.Units()) || !reflect.DeepEqual(h, c.battle.History()) {
			t.Fatal("閲覧中に盤面の入力を受けた")
		}
	})
	t.Run("TC-FR-BTL-10-25 確定の成功だけ 1 消費し、未来を捨てて見ていた盤面に戻る", func(t *testing.T) {
		c := longHistory(t)
		i := 0
		c.OpenRewind()
		c.rewindSelect(i)
		seen := c.RewindBoard().Units
		clickButton(t, c, "決定")
		if c.battle.RewindsLeft() != 10 || !c.rewind.confirm {
			t.Fatal("確認の前に消費した")
		}
		if !strings.Contains(viewStrings(c.RewindView()), "残り 10 → 9") {
			t.Fatal("確認に消費の表示がない", viewStrings(c.RewindView()))
		}
		clickButton(t, c, "戻す")
		if c.battle.RewindsLeft() != 9 || len(c.battle.History()) != i || c.rewind.open {
			t.Fatal("成功で 1 消費・未来の破棄にならない")
		}
		if !reflect.DeepEqual(seen, c.battle.Units()) {
			t.Fatal("見ていた盤面と復元後の盤面が違う")
		}
		if c.RewindBoard().Visible || c.rewind.previewLoaded {
			t.Fatal("成功後にプレビューが残った")
		}
		c.rewindKey("Enter")
		c.rewindKey("Enter")
		if c.battle.RewindsLeft() != 9 {
			t.Fatal("閉じた一覧への Enter で二重に消費した")
		}
	})
	t.Run("TC-FR-BTL-10-25 敵軍・番の境界のプレビューから確認と確定はできない", func(t *testing.T) {
		for _, kind := range []tactics.HistoryKind{tactics.HistoryAction, tactics.HistoryPlayerStart} {
			c := newIdle(t)
			endTurn(c)
			side := tactics.Enemy
			if kind == tactics.HistoryPlayerStart {
				side = tactics.Player
			}
			c.OpenRewind()
			c.rewindSelect(historyIndex(t, c.battle.History(), kind, side))
			checkReadOnlyRewind(t, c)
		}
	})
	t.Run("TC-FR-BTL-10-25 敗北の選択から閲覧して取り消すと敗北のまま、確定で戦闘へ戻る", func(t *testing.T) {
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
			t.Fatal("前提: 敗北でない")
		}
		clickSceneButton(t, s, "巻き戻す")
		s.battle.RewindMove(-100)
		v := s.battle.RewindBoard()
		if !v.Visible || !v.Available || v.Board.Result != tactics.Ongoing || !v.Units[0].Alive {
			t.Fatal("敗北から行動前を閲覧できない")
		}
		s.syncRewind()
		if s.Scene() != GameOver {
			t.Fatal("プレビューの Ongoing で戦闘へ移った")
		}
		s.battle.Cancel()
		s.syncRewind()
		if s.Scene() != GameOver || s.battle.state != Over || s.battle.RewindBoard().Visible {
			t.Fatal("取消しで敗北の選択に戻らない")
		}
		clickSceneButton(t, s, "巻き戻す")
		s.battle.RewindMove(-100)
		s.battle.RewindEnter()
		s.battle.RewindEnter()
		s.syncRewind()
		if s.Scene() != Battle || s.battle.battle.RewindsLeft() != 9 || m.writes != 1 {
			t.Fatal("確定で戦闘へ戻らない")
		}
	})
	t.Run("TC-FR-BTL-10-25 勝利の帯から閲覧すると帯を描かず、取消しで帯に戻る", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		winScene(t, s)
		playOut(s.battle)
		c := s.battle
		c.OpenRewind()
		if !c.RewindBoard().Visible || c.RewindBoard().Board.Result != tactics.Ongoing || c.state != Over {
			t.Fatal("勝利の帯から行動前を閲覧できない")
		}
		c.Cancel()
		if c.state != Over || s.Scene() != Battle || c.RewindBoard().Visible {
			t.Fatal("取消しで勝利の帯に戻らない")
		}
	})
}

func TestRewindPreviewLayout(t *testing.T) {
	t.Run("TC-FR-BTL-10-24 / TC-FR-UNT-09-12 右欄と下の案内が 832 × 576 に収まり、盤面を覆わない", func(t *testing.T) {
		long := strings.Repeat("長い名前", 100) + "末尾"
		c := newIdleWith(t, func() tactics.Setup { s := basicSetup(); s.Units[1].Name = long; return s })
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		for i := 0; i < 3; i++ {
			c.EndPhase()
			c.EndPhase()
			playOut(c)
		}
		right := image.Rect(MapSize, 0, ScreenWidth, ScreenHeight)
		checkView := func(step string) {
			t.Helper()
			v := c.RewindView()
			for _, tx := range v.Texts {
				r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
				if !r.In(right) || tx.X < rewindLeft || r.Max.X > rewindRight {
					t.Fatalf("%s: 文が右欄の外 %v %q", step, r, tx.Text)
				}
			}
			for i, b := range v.Buttons {
				if !b.Rect.In(right) {
					t.Fatalf("%s: ボタンが右欄の外 %v", step, b)
				}
				for _, o := range v.Buttons[i+1:] {
					if b.Rect.Overlaps(o.Rect) {
						t.Fatalf("%s: ボタンが重なる %v %v", step, b, o)
					}
				}
				for _, tx := range v.Texts {
					if b.Rect.Overlaps(image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)) {
						t.Fatalf("%s: ボタンが文を隠す %v %q", step, b, tx.Text)
					}
				}
			}
			for _, r := range v.Rows {
				if !r.Rect.In(right) {
					t.Fatalf("%s: 行が右欄の外 %v", step, r)
				}
			}
			lines := c.RewindBoard().Lines
			if len(lines) > GuideHeight/lineHeight {
				t.Fatalf("%s: 下の案内が %d 行", step, len(lines))
			}
			for _, line := range lines {
				if 8+textWidth(line) > MapSize {
					t.Fatalf("%s: 下の案内が盤面の幅を超える %q", step, line)
				}
			}
		}
		c.OpenRewind()
		checkView("最新行")
		c.RewindMove(-100)
		checkView("長い行名")
		c.RewindDetail(1000)
		if !strings.Contains(viewStrings(c.RewindView()), "末尾") {
			t.Fatal("詳細の全文を読めない")
		}
		checkView("詳細の末尾")
		c.RewindEnter()
		checkView("確認")
		c.RewindDetail(1000)
		if !strings.Contains(viewStrings(c.RewindView()), "末尾") || !strings.Contains(viewStrings(c.RewindView()), "残り 10 → 9") {
			t.Fatal("確認で全文と消費を読めない")
		}
		checkView("確認の末尾")
		c.Cancel()
		c.rewind.message = strings.Repeat("巻き戻せません。", 30) + "誤りの末尾"
		c.RewindDetail(1000)
		if !strings.Contains(viewStrings(c.RewindView()), "誤りの末尾") {
			t.Fatal("誤りの全文を読めない")
		}
		checkView("誤り")
		c.fetchPreview(1 << 40)
		checkView("取得失敗")
		c.Cancel()
		c = newIdle(t)
		c.OpenRewind()
		checkView("履歴なし")
	})
}

// プレビューの盤面も戦闘と同じ絵の選択（unitTile）を通す（DES-UI-21、DES-UI-22、REV-045 F-03）。
func TestRewindPreviewCavalier(t *testing.T) {
	t.Run("TC-FR-UNT-09-18 プレビューの盤面のエルザと敵の騎兵も騎乗の絵になる", func(t *testing.T) {
		s := NewSceneController(Config{Catalog: cavalierCatalog(), Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }})
		startScenes(s)
		clickSceneButton(t, s, "配置へ")
		clickSceneButton(t, s, "戦闘開始")
		c := s.BattleController()
		playOut(c)
		// エルザ（名簿の 2 番目。枠 (0, 1)）が待機してから一覧を開く。
		c.Click(tactics.Pos{X: 0, Y: 1})
		c.Click(tactics.Pos{X: 0, Y: 1})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		if len(c.battle.History()) != 1 {
			t.Fatal("前提: エルザの待機の履歴がない")
		}
		c.OpenRewind()
		v := c.RewindBoard()
		if !v.Available {
			t.Fatal("プレビューがない")
		}
		want := map[string]assets.Tile{"主人公": assets.PlayerLord, "エルザ": assets.Cavalier, "ガルド": assets.PlayerLance}
		cavaliers := 0
		for _, u := range v.Units {
			got := unitTile(u)
			if w, ok := want[u.Name]; ok && got != w {
				t.Errorf("プレビューの %s の絵 = %v、%v のはず", u.Name, got, w)
			}
			if u.Side == tactics.Enemy && u.ClassName == "アーマー" && got != assets.EnemyLance {
				t.Errorf("プレビューの敵のアーマーの絵 = %v", got)
			}
			if got == assets.Cavalier {
				cavaliers++
			}
		}
		if cavaliers != 3 {
			t.Errorf("プレビューの騎乗の絵が %d 体、3 体（エルザ・隊長・騎兵）のはず", cavaliers)
		}
	})
}

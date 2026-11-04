package ui

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// framesIs は、範囲の 2 つの集まり（移動・攻撃）が wantBlue・wantRed であることを確かめる。空と nil は同じに扱う。
func framesIs(t *testing.T, what string, blue, red, wantBlue, wantRed map[tactics.Pos]bool) {
	t.Helper()
	if len(blue) == 0 && len(wantBlue) == 0 {
		blue, wantBlue = nil, nil
	}
	if len(red) == 0 && len(wantRed) == 0 {
		red, wantRed = nil, nil
	}
	if !reflect.DeepEqual(blue, wantBlue) || !reflect.DeepEqual(red, wantRed) {
		t.Errorf("%s: 移動 %v、攻撃 %v\nwant 移動 %v、攻撃 %v", what, blue, red, wantBlue, wantRed)
	}
}

// cursorIs は、カーソルの敵の範囲（CursorFrames。青白の実線・破線）を確かめる（DES-UI-29）。
func cursorIs(t *testing.T, c *Controller, what string, wantBlue, wantRed map[tactics.Pos]bool) {
	t.Helper()
	blue, red := c.CursorFrames()
	framesIs(t, what+"（カーソル）", blue, red, wantBlue, wantRed)
}

// watchedIs は、固定した敵の範囲（WatchedFrames。橙の実線・破線）を確かめる（DES-UI-29）。
func watchedIs(t *testing.T, c *Controller, what string, wantBlue, wantRed map[tactics.Pos]bool) {
	t.Helper()
	blue, red := c.WatchedFrames()
	framesIs(t, what+"（固定）", blue, red, wantBlue, wantRed)
}

// layerSummary は層の並び（出所と描き方）だけを返す。
func layerSummary(ls []EnemyRangeLayer) []EnemyRangeLayer {
	out := make([]EnemyRangeLayer, len(ls))
	for i, l := range ls {
		out[i] = EnemyRangeLayer{Owner: l.Owner, Stroke: l.Stroke}
	}
	return out
}

// coreRanges は、core の Reachable と AttackRange だけから、敵 ws の移動できるマスと、攻撃できて移動できないマスを求める（DES-UI-29）。
// UI の計算と比べるための期待値で、UI の関数は使わない。
func coreRanges(b *tactics.Battle, ws ...int) (move, attack map[tactics.Pos]bool) {
	move, attack = map[tactics.Pos]bool{}, map[tactics.Pos]bool{}
	for _, i := range ws {
		for p := range b.Reachable(i) {
			move[p] = true
		}
	}
	for _, i := range ws {
		for p := range b.Reachable(i) {
			for _, q := range b.AttackRange(i, p) {
				if !move[q] {
					attack[q] = true
				}
			}
		}
	}
	return move, attack
}

// coreDanger は、core の Reachable と AttackRange だけから、敵 ws の攻撃できるマスをすべて求める（全敵の危険範囲の期待値）。
func coreDanger(b *tactics.Battle, ws ...int) map[tactics.Pos]bool {
	out := map[tactics.Pos]bool{}
	for _, i := range ws {
		for p := range b.Reachable(i) {
			for _, q := range b.AttackRange(i, p) {
				out[q] = true
			}
		}
	}
	return out
}

// 敵の範囲の出所の区別（DES-UI-29、DES-IT07 3.2 節）を確かめる。
func TestEnemyRangeIT07(t *testing.T) {
	fighter := tactics.Pos{X: 1, Y: 0}
	boss := tactics.Pos{X: 5, Y: 5}
	empty := tactics.Pos{X: 3, Y: 3}
	bossBlue, bossRed := posSet(boss), posSet(tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4})
	// clearXY は「消す」ボタン (752,304)〜(824,336) の中央。
	clearXY := [2]int{788, 320}

	t.Run("TC-FR-UNT-09-27 全敵は塗り、固定は橙の実線・破線と金のピン、カーソルは青白の実線・破線。塗り → 固定 → カーソルの順。右欄に「全敵 Q」「固定 n」、「消す」で一括解除。Selected の敵クリックは固定を変えない", func(t *testing.T) {
		c := newIdle(t)
		g := &Game{ctrl: c}
		status := func(what, danger, watched string, enabled bool) {
			t.Helper()
			v := c.RangeStatus()
			if v.Danger.Text != danger || v.Watched.Text != watched || v.Clear.Enabled != enabled {
				t.Errorf("%s: 状態 %q・%q・消す %v、want %q・%q・%v", what, v.Danger.Text, v.Watched.Text, v.Clear.Enabled, danger, watched, enabled)
			}
		}
		// 初め: どの層もなく、右欄は「全敵 Q OFF」「固定 0」、「消す」は押せない。
		if ls := c.EnemyRangeLayers(); len(ls) != 0 {
			t.Fatalf("初めから層がある: %v", ls)
		}
		status("初め", "全敵 Q OFF", "固定 0", false)

		// 右欄の配置（DES-IT07 3.2・3.4 節）: 範囲の状態は (520,304) から 2 行、「消す」は (752,304)〜(824,336)。
		v := c.RangeStatus()
		if v.Danger.X != 520 || v.Danger.Y != 304 || v.Watched.X != 520 || v.Watched.Y != 304+lineHeight {
			t.Errorf("範囲の状態の位置 %+v・%+v", v.Danger, v.Watched)
		}
		if v.Clear.Rect != image.Rect(752, 304, 824, 336) || v.Clear.Label != "消す" {
			t.Errorf("消すボタン %+v", v.Clear)
		}
		guide := InfoTabGuide()
		guideRect := rectOfText(guide.X, guide.Y, guide.Text, false)
		for _, tx := range []InfoText{v.Danger, v.Watched} {
			for _, label := range []string{tx.Text, "全敵 Q OFF", "固定 99"} {
				r := rectOfText(tx.X, tx.Y, label, false)
				if !r.In(image.Rect(520, 304, 664, 336)) {
					t.Errorf("%q %v が範囲の状態の欄 (520,304)〜(664,336) からはみ出す", label, r)
				}
				if r.Overlaps(v.Clear.Rect) || r.Overlaps(MinimapRect()) {
					t.Errorf("%q %v がボタン・ミニマップと重なる", label, r)
				}
				for _, tab := range InfoTabRects() {
					if r.Overlaps(tab) {
						t.Errorf("%q %v がページの印 %v と重なる", label, r, tab)
					}
				}
			}
		}
		for _, tab := range append(InfoTabRects(), guideRect, MinimapRect()) {
			if v.Clear.Rect.Overlaps(tab) {
				t.Errorf("消すボタンが %v と重なる", tab)
			}
		}

		// Q: 全敵の危険範囲を赤紫の塗りの層で出す。
		c.ToggleDanger()
		want := []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}}
		if ls := c.EnemyRangeLayers(); !reflect.DeepEqual(layerSummary(ls), want) || !reflect.DeepEqual(ls[0].Cells, basicDanger) {
			t.Fatalf("Q の層 %v", ls)
		}
		status("Q", "全敵 Q ON", "固定 0", false)

		// Idle で敵を左クリック（Game.leftClick の経路）して固定に足す。カーソルは盤面の外。
		c.SetCursor(-1, -1)
		g.leftClick(cellCenter(fighter))
		g.leftClick(cellCenter(boss))
		if c.State() != Idle || !reflect.DeepEqual(c.watched, map[int]bool{2: true, 3: true}) {
			t.Fatalf("固定: 状態 %v、%v", c.State(), c.watched)
		}
		status("固定 2", "全敵 Q ON", "固定 2", true)
		bothBlue := posSet(fighter, boss)
		bothRed := posSet(tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 2, Y: 0}, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4})
		want = []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}, {Owner: RangeWatched, Stroke: StrokeSolid}, {Owner: RangeWatched, Stroke: StrokeDashed}}
		ls := c.EnemyRangeLayers()
		if !reflect.DeepEqual(layerSummary(ls), want) {
			t.Fatalf("固定 2 の層 %v", layerSummary(ls))
		}
		framesIs(t, "固定 2 の層", ls[1].Cells, ls[2].Cells, bothBlue, bothRed)
		// 金のピンは固定した敵の位置に、添字の順で置く。
		if pins := c.WatchedPins(); !reflect.DeepEqual(pins, []tactics.Pos{boss, fighter}) {
			t.Errorf("ピン %v", pins)
		}

		// カーソルを固定したボスに合わせる: カーソルの層を最後に重ね、固定は変えない。
		c.SetCursor(cellCenter(boss))
		want = append(want, EnemyRangeLayer{Owner: RangeCursor, Stroke: StrokeSolid}, EnemyRangeLayer{Owner: RangeCursor, Stroke: StrokeDashed})
		ls = c.EnemyRangeLayers()
		if !reflect.DeepEqual(layerSummary(ls), want) {
			t.Fatalf("カーソルの層 %v", layerSummary(ls))
		}
		framesIs(t, "カーソルの層", ls[3].Cells, ls[4].Cells, bossBlue, bossRed)
		framesIs(t, "カーソルのときの固定の層", ls[1].Cells, ls[2].Cells, bothBlue, bothRed)
		if !reflect.DeepEqual(c.watched, map[int]bool{2: true, 3: true}) || c.CursorEnemy() != 2 {
			t.Errorf("カーソルで固定が変わった: %v、カーソル %d", c.watched, c.CursorEnemy())
		}
		// 情報欄の敵の名前の横に、カーソルの線と同じ印を置く（名前の右 4 px、10×10、情報欄の中）。
		iv := c.BattleInfoView(cellCenter(boss))
		name := textOf(t, iv, "ボス　★ボス")
		mark, ok := c.cursorNameMark(iv)
		if wantMark := image.Rect(name.X+textWidth(name.Text)+4, name.Y+3, name.X+textWidth(name.Text)+14, name.Y+13); !ok || mark != wantMark {
			t.Errorf("名前の横の印 %v %v、want %v", mark, ok, wantMark)
		}
		if !mark.In(image.Rect(MapSize, 0, ScreenWidth, InfoHeight)) {
			t.Errorf("名前の横の印 %v が情報欄の外", mark)
		}
		c.SetCursor(cellCenter(empty))
		if _, ok := c.cursorNameMark(c.BattleInfoView(cellCenter(empty))); ok {
			t.Error("カーソルが敵の上にないのに印を出す")
		}

		// もう一度クリックで外す。
		g.leftClick(cellCenter(fighter))
		status("1 体外す", "全敵 Q ON", "固定 1", true)
		watchedIs(t, c, "1 体外す", bossBlue, bossRed)

		// 「消す」で固定を一括で外す。Q・状態は変えず、盤面のクリックとしては扱わない。
		g.leftClick(cellCenter(fighter))
		status("足し直す", "全敵 Q ON", "固定 2", true)
		g.leftClick(clearXY[0], clearXY[1])
		if len(c.watched) != 0 || c.State() != Idle || !c.danger {
			t.Fatalf("消す: 固定 %v、状態 %v、Q %v", c.watched, c.State(), c.danger)
		}
		status("消す", "全敵 Q ON", "固定 0", false)
		if ls := c.EnemyRangeLayers(); !reflect.DeepEqual(layerSummary(ls), []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}}) || len(c.WatchedPins()) != 0 {
			t.Errorf("消したあとの層 %v、ピン %v", layerSummary(ls), c.WatchedPins())
		}

		// Selected: 敵のクリックは直接攻撃の操作で、固定を変えない。消すボタンは Selected でも受ける。
		g.leftClick(cellCenter(boss))
		g.leftClick(cellCenter(tactics.Pos{X: 0, Y: 0}))
		if c.State() != Selected {
			t.Fatalf("状態 = %v", c.State())
		}
		g.leftClick(cellCenter(fighter))
		if c.State() == Idle || c.State() == Selected || !reflect.DeepEqual(c.watched, map[int]bool{2: true}) {
			t.Errorf("Selected の敵クリック: 状態 %v、固定 %v", c.State(), c.watched)
		}
		for n := 0; c.State() != Selected && n < 10; n++ {
			c.Cancel()
		}
		if c.State() != Selected || !reflect.DeepEqual(c.watched, map[int]bool{2: true}) {
			t.Fatalf("取り消し: 状態 %v、固定 %v", c.State(), c.watched)
		}
		g.leftClick(clearXY[0], clearXY[1])
		if len(c.watched) != 0 || c.State() != Selected {
			t.Errorf("Selected の消す: 固定 %v、状態 %v", c.watched, c.State())
		}

		// 再生中は「消す」を受けない（ボタンの中のクリックは盤面へ渡さない）。
		c.Cancel()
		g.leftClick(cellCenter(boss))
		c.EndPhase()
		c.EndPhase()
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		status("Playing", "全敵 Q ON", "固定 1", false)
		if !c.RangeClick(clearXY[0], clearXY[1]) || len(c.watched) != 1 {
			t.Errorf("Playing の消す: 固定 %v", c.watched)
		}
		// 再生中は固定の線もピンも出さない（E-124）。
		watchedIs(t, c, "Playing", nil, nil)
		if len(c.WatchedPins()) != 0 {
			t.Errorf("Playing のピン %v", c.WatchedPins())
		}
		playOut(c)
		if c.RangeClick(751, 320) || c.RangeClick(788, 336) {
			t.Error("ボタンの外を受けた")
		}
		// 再生のあとの入力待ちでは出す。巻き戻しの一覧の間は出さず、閉じると戻る（E-124）。
		watchedIs(t, c, "再生のあと", bossBlue, bossRed)
		c.OpenRewind()
		if !c.rewind.open {
			t.Fatal("巻き戻しの一覧が開かない")
		}
		watchedIs(t, c, "巻き戻しの一覧", nil, nil)
		if len(c.WatchedPins()) != 0 {
			t.Errorf("巻き戻しの一覧のピン %v", c.WatchedPins())
		}
		c.Cancel()
		watchedIs(t, c, "巻き戻しの一覧を閉じる", bossBlue, bossRed)

		// 入力を待つ状態はすべて固定を出す（E-124）: ItemMenu・Targeting（相手を選んだ予測を含む）。
		d := damaged(t)
		dg := &Game{ctrl: d}
		dg.leftClick(cellCenter(boss))
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if d.State() != ItemMenu {
			t.Fatalf("状態 = %v", d.State())
		}
		watchedIs(t, d, "ItemMenu", bossBlue, bossRed)
		d.Cancel()
		d.Cancel()
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.ChooseAction(tactics.Attack)
		d.Click(fighter)
		if d.State() != Targeting {
			t.Fatalf("状態 = %v", d.State())
		}
		watchedIs(t, d, "Targeting の予測", bossBlue, bossRed)
		if !reflect.DeepEqual(d.WatchedPins(), []tactics.Pos{boss}) {
			t.Errorf("Targeting の予測のピン %v", d.WatchedPins())
		}

		// 入力を待つ状態を一つずつ当てても、固定の線とピンは消えない。武器・杖の一覧（Canto の次の 2 つ）も含める（E-124）。
		w := newIdle(t)
		w.Click(boss)
		for _, st := range []State{Idle, Selected, Moved, Targeting, ItemMenu, HealTargeting, Canto, Canto + 1, Canto + 2} {
			w.state = st
			watchedIs(t, w, fmt.Sprintf("状態 %d", st), bossBlue, bossRed)
			if !reflect.DeepEqual(w.WatchedPins(), []tactics.Pos{boss}) {
				t.Errorf("状態 %d のピン %v", st, w.WatchedPins())
			}
		}
		for _, st := range []State{Playing, Over} {
			w.state = st
			watchedIs(t, w, fmt.Sprintf("状態 %d", st), nil, nil)
			if len(w.WatchedPins()) != 0 {
				t.Errorf("状態 %d のピン %v", st, w.WatchedPins())
			}
		}

		// Over では出さない。
		lost := func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Stats.HP = 1
			return s
		}
		o := newIdleWith(t, lost)
		o.Click(boss)
		o.EndPhase()
		o.EndPhase()
		playOut(o)
		if o.State() != Over {
			t.Fatalf("状態 = %v", o.State())
		}
		watchedIs(t, o, "Over", nil, nil)
		if len(o.WatchedPins()) != 0 {
			t.Errorf("Over のピン %v", o.WatchedPins())
		}
	})

	t.Run("TC-FR-UNT-09-27 範囲の状態と「消す」は情報欄の中身と重ねない。武器 5 本でも右欄の上下 2 段は y < 304 に収まり、すべて描く（REV-096 F-03、#166）", func(t *testing.T) {
		killer := tactics.Weapon{Name: "アーマーキラー", Type: tactics.Lance, Might: 5, Hit: 80, MinRange: 1, MaxRange: 1}
		fire := tactics.Weapon{Name: "ファイア", Type: tactics.Magic, DamageType: tactics.Magical, Might: 5, Hit: 90, MinRange: 1, MaxRange: 2}
		five := func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Weapon, s.Units[0].Items = tactics.Weapon{}, nil
			s.Units[0].Weapons = []tactics.Weapon{ironSword, killer, ironAxe, ironBow, fire}
			s.Units[0].AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe, tactics.Bow, tactics.Magic}
			return s
		}
		visible := func(v InfoView, s RangeStatusView) []image.Rectangle {
			var out []image.Rectangle
			if !s.HideTexts {
				for _, tx := range []InfoText{s.Danger, s.Watched} {
					out = append(out, rectOfText(tx.X, tx.Y, tx.Text, false))
				}
			}
			if !s.HideClear {
				out = append(out, s.Clear.Rect)
			}
			return out
		}
		noOverlap := func(what string, v InfoView, s RangeStatusView) {
			t.Helper()
			for _, r := range visible(v, s) {
				for _, tx := range v.Texts {
					if q := rectOfText(tx.X, tx.Y, tx.Text, tx.Big); q.Overlaps(r) {
						t.Errorf("%s: 範囲の状態 %v が情報欄の %q %v と重なる", what, r, tx.Text, q)
					}
				}
				for _, ic := range v.Icons {
					if q := image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size); q.Overlaps(r) {
						t.Errorf("%s: 範囲の状態 %v が情報欄の絵 %v と重なる", what, r, q)
					}
				}
			}
		}
		// 武器 5 本のレオン（平地）にカーソルを合わせる。右欄は上下 2 段で y < 296 に収まる（#166、DES-UI-31）ので、
		// 範囲の状態と「消す」をすべて描いて重ならない。
		c := newIdleWith(t, five)
		g := &Game{ctrl: c}
		hx, hy := cellCenter(tactics.Pos{X: 0, Y: 0})
		c.SetCursor(hx, hy)
		v := c.BattleInfoView(hx, hy)
		if terrain := textOf(t, v, "地形: 平地　回避 0　守備 0　回復 0%"); terrain.Y+lineHeight > infoControlTop || v.Bottom > infoControlTop {
			t.Fatalf("前提: 地形の行 y = %d、一番下 %d（操作行 y = 304〜 より上のはず）", terrain.Y, v.Bottom)
		}
		s := c.RangeStatusIn(v)
		if s.HideTexts || s.HideClear {
			t.Errorf("武器 5 本: 文を隠す %v、消すを隠す %v", s.HideTexts, s.HideClear)
		}
		noOverlap("武器 5 本", v, s)
		// 描いている「消す」はクリックを受ける。
		g.leftClick(cellCenter(tactics.Pos{X: 5, Y: 5}))
		c.SetCursor(hx, hy)
		if c.WatchedCount() != 1 {
			t.Fatalf("前提: 固定 %d", c.WatchedCount())
		}
		c.SetCursor(clearXY[0], clearXY[1])
		g.leftClick(clearXY[0], clearXY[1])
		if c.WatchedCount() != 0 {
			t.Errorf("武器 5 本の情報欄のときの消す: 固定 %d", c.WatchedCount())
		}
		// レオンを選んで敵にカーソルを合わせ、上下 2 段を出しても重ならない。武器 1 本のユニットでも同じ。
		c.Click(tactics.Pos{X: 0, Y: 0})
		v = c.BattleInfoView(cellCenter(tactics.Pos{X: 1, Y: 0}))
		if s := c.RangeStatusIn(v); s.HideTexts || s.HideClear {
			t.Errorf("上下 2 段で隠した: %+v", s)
		} else {
			noOverlap("上下 2 段", v, s)
		}
		one := newIdle(t)
		v = one.BattleInfoView(hx, hy)
		if s := one.RangeStatusIn(v); s.HideTexts || s.HideClear {
			t.Errorf("武器 1 本で隠した: %+v", s)
		} else {
			noOverlap("武器 1 本", v, s)
		}
		// 情報欄の「消す」の位置に中身が重なる場合は、描かずクリックも受けない（盤面へ渡す）。
		blocked := InfoView{Texts: []InfoText{{X: 760, Y: 310, Text: "長い行"}}}
		if s := one.RangeStatusIn(blocked); !s.HideClear {
			t.Error("重なる消すを隠さない")
		}
	})

	t.Run("TC-FR-UNT-09-27 線とピンの形: 固定は内側 3 px、カーソルは内側 8 px。破線は 4 px の線と 3 px の空き。ピンは右上で、ボスの ★ を隠さない", func(t *testing.T) {
		pt := image.Pt(64, 32)
		if r := rangeFrameRect(pt, watchedInset); r != image.Rect(67, 35, 93, 61) {
			t.Errorf("固定の線 %v", r)
		}
		if r := rangeFrameRect(pt, cursorInset); r != image.Rect(72, 40, 88, 56) {
			t.Errorf("カーソルの線 %v", r)
		}
		// 26 px の辺は 0〜4、7〜11、14〜18、21〜25 の 4 本。4 辺で 16 本。
		segs := dashedSegments(image.Rect(67, 35, 93, 61))
		if len(segs) != 16 {
			t.Fatalf("破線の数 %d", len(segs))
		}
		top := [][2]image.Point{{{67, 35}, {71, 35}}, {{74, 35}, {78, 35}}, {{81, 35}, {85, 35}}, {{88, 35}, {92, 35}}}
		if !reflect.DeepEqual(segs[:4], top) {
			t.Errorf("上の辺 %v", segs[:4])
		}
		if segs[4] != [2]image.Point{{93, 35}, {93, 39}} {
			t.Errorf("右の辺の始め %v", segs[4])
		}
		// 16 px の辺は 0〜4、7〜11、14〜16 の 3 本で、辺の終わりで切る。
		short := dashedSegments(image.Rect(72, 40, 88, 56))
		if len(short) != 12 || short[2] != [2]image.Point{{86, 40}, {88, 40}} {
			t.Errorf("16 px の破線 %v", short)
		}
		head, stem := pinShape(pt, false)
		if head != image.Rect(87, 33, 93, 39) || stem != image.Rect(89, 39, 91, 43) {
			t.Errorf("ピン %v %v", head, stem)
		}
		// ボスの ★ はマスの x = 20〜 に描く。ピンはその左、マスの右半分寄りに置く。
		bossHead, _ := pinShape(pt, true)
		if bossHead.Max.X > pt.X+CellSize-12 || bossHead.Min.X < pt.X+8 {
			t.Errorf("ボスのピン %v", bossHead)
		}
	})

	t.Run("TC-FR-UNT-09-28 戦死した敵は固定から外れ、巻き戻しでは戻った盤面の生きている敵だけで求め直す。範囲は core の Reachable と攻撃射程に一致し、ミニマップは危険マスの色と固定のピンで示す", func(t *testing.T) {
		// checkMinimap は、ミニマップの危険マスが Danger の各マスの CellRect（上から左から）、ピンが pins の CellRect であることを確かめる。
		checkMinimap := func(what string, c *Controller, pins []tactics.Pos) {
			t.Helper()
			mv := c.Minimap()
			l := mv.Layout
			var wantDanger []image.Rectangle
			for y := 0; y < l.Height; y++ {
				for x := 0; x < l.Width; x++ {
					if c.Danger()[tactics.Pos{X: x, Y: y}] {
						wantDanger = append(wantDanger, l.CellRect(tactics.Pos{X: x, Y: y}))
					}
				}
			}
			var wantPins []image.Rectangle
			for _, p := range pins {
				wantPins = append(wantPins, l.CellRect(p))
			}
			if len(wantDanger) == 0 || !reflect.DeepEqual(mv.Danger, wantDanger) || !reflect.DeepEqual(mv.Pins, wantPins) {
				t.Errorf("%s: ミニマップの危険 %v、ピン %v\nwant %v、%v", what, mv.Danger, mv.Pins, wantDanger, wantPins)
			}
			for _, r := range append(append([]image.Rectangle{}, mv.Danger...), mv.Pins...) {
				if !r.In(l.Area) {
					t.Errorf("%s: %v が地図の外", what, r)
				}
			}
		}

		k := newIdleWith(t, killSetup)
		g := &Game{ctrl: k}
		k.ToggleDanger()
		k.SetCursor(-1, -1)
		g.leftClick(cellCenter(tactics.Pos{X: 1}))
		g.leftClick(cellCenter(tactics.Pos{X: 5, Y: 5}))
		m, a := coreRanges(k.Battle(), 1, 2)
		watchedIs(t, k, "固定 2", m, a)
		if l := k.Minimap().Layout; l.Cell != 21 {
			t.Fatalf("前提: ミニマップの 1 マス %d（6×6 は 128÷6 の 21 px）", l.Cell)
		}
		checkMinimap("固定 2", k, []tactics.Pos{{X: 5, Y: 5}, {X: 1, Y: 0}})
		// 主人公が弱い敵を倒す。
		k.Click(tactics.Pos{X: 0})
		k.Click(tactics.Pos{X: 0})
		k.ChooseAction(tactics.Attack)
		k.Click(tactics.Pos{X: 1})
		k.Click(tactics.Pos{X: 1})
		// 決定の時点で戦死は盤面に入っている。再生中も「固定 n」に戦死した敵を数えない。
		if k.State() != Playing || k.WatchedCount() != 1 || k.RangeStatus().Watched.Text != "固定 1" {
			t.Errorf("再生中: 状態 %v、固定 %d", k.State(), k.WatchedCount())
		}
		playOut(k)
		if k.Battle().Units()[2].Alive || k.State() != Idle {
			t.Fatalf("前提: 弱い敵が倒れていない、状態 %v", k.State())
		}
		if k.WatchedCount() != 1 || !reflect.DeepEqual(k.WatchedPins(), []tactics.Pos{{X: 5, Y: 5}}) {
			t.Errorf("戦死のあと: 固定 %d、ピン %v", k.WatchedCount(), k.WatchedPins())
		}
		m, a = coreRanges(k.Battle(), 1)
		watchedIs(t, k, "戦死のあと", m, a)
		checkMinimap("戦死のあと", k, []tactics.Pos{{X: 5, Y: 5}})
		if got, want := k.Danger(), posSet(tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4}); !reflect.DeepEqual(got, want) {
			t.Errorf("戦死のあとの危険範囲 %v", got)
		}
		// 巻き戻しの一覧の間は「消す」を受けない。成功すると、戻った盤面に生きている固定の敵だけで求め直す。
		// 戦死で外した弱い敵は、戻った盤面で生きていても固定に戻さない。
		k.OpenRewind()
		if !k.rewind.open || k.RangeClick(clearXY[0], clearXY[1]) || k.WatchedCount() != 1 {
			t.Fatal("巻き戻しの一覧で消すボタンを受けた")
		}
		k.RewindMove(-100)
		for k.rewind.open {
			before := k.battle.RewindsLeft()
			k.RewindEnter()
			if k.rewind.open && !k.rewind.confirm && k.battle.RewindsLeft() == before {
				t.Fatal("巻き戻せない")
			}
		}
		if !k.Battle().Units()[2].Alive || k.State() != Idle {
			t.Fatalf("巻き戻しのあと: 弱い敵 %v、状態 %v", k.Battle().Units()[2].Alive, k.State())
		}
		m, a = coreRanges(k.Battle(), 1)
		watchedIs(t, k, "巻き戻しのあと", m, a)
		if !reflect.DeepEqual(k.WatchedPins(), []tactics.Pos{{X: 5, Y: 5}}) {
			t.Errorf("巻き戻しのあとのピン %v", k.WatchedPins())
		}
		if got, want := k.Danger(), coreDanger(k.Battle(), 1, 2); !reflect.DeepEqual(got, want) {
			t.Errorf("巻き戻しのあとの全敵 %v、core %v", got, want)
		}
		checkMinimap("巻き戻しのあと", k, []tactics.Pos{{X: 5, Y: 5}})

		// 位置が変わる盤面（通路）: 固定・カーソル・全敵のどれも、今の盤面の core の到達と射程に一致する。
		c := newIdleWith(t, corridorSetup)
		g = &Game{ctrl: c}
		c.ToggleDanger()
		c.SetCursor(-1, -1)
		g.leftClick(cellCenter(tactics.Pos{X: 4}))
		// レオンがその場で待機すると敵軍の番になり、ソルジャーは (3, 0) へ動く。
		g.leftClick(cellCenter(tactics.Pos{X: 2}))
		g.leftClick(cellCenter(tactics.Pos{X: 2}))
		c.ChooseAction(tactics.Wait)
		playOut(c)
		if c.Battle().Units()[2].Pos != (tactics.Pos{X: 3}) || c.State() != Idle {
			t.Fatalf("前提: ソルジャー %v、状態 %v", c.Battle().Units()[2].Pos, c.State())
		}
		m, a = coreRanges(c.Battle(), 2)
		watchedIs(t, c, "敵軍の番のあと", m, a)
		c.SetCursor(cellCenter(tactics.Pos{X: 0}))
		m, a = coreRanges(c.Battle(), 1)
		cursorIs(t, c, "ボスに合わせる", m, a)
		if got, want := c.Danger(), coreDanger(c.Battle(), 1, 2); !reflect.DeepEqual(got, want) {
			t.Errorf("全敵 %v、core %v", got, want)
		}
		checkMinimap("通路", c, []tactics.Pos{{X: 3}})
		// 2 ターン目の敵軍の番に増援 (9, 0) が現れる。増援も固定する。
		c.SetCursor(-1, -1)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		if len(c.Battle().Units()) != 4 || c.State() != Idle {
			t.Fatalf("前提: 増援 %d 体、状態 %v", len(c.Battle().Units()), c.State())
		}
		g.leftClick(cellCenter(tactics.Pos{X: 9}))
		if !reflect.DeepEqual(c.WatchedPins(), []tactics.Pos{{X: 3}, {X: 9}}) || c.WatchedCount() != 2 {
			t.Fatalf("増援の固定: ピン %v", c.WatchedPins())
		}
		// レオンの待機の前へ巻き戻すと、増援はまだいないので固定から外し、ソルジャーは戻った盤面 (4, 0) で求め直す。
		c.OpenRewind()
		c.RewindMove(-100)
		for c.rewind.open {
			before := c.battle.RewindsLeft()
			c.RewindEnter()
			if c.rewind.open && !c.rewind.confirm && c.battle.RewindsLeft() == before {
				t.Fatal("巻き戻せない")
			}
		}
		if len(c.Battle().Units()) != 3 || c.Battle().Units()[2].Pos != (tactics.Pos{X: 4}) {
			t.Fatalf("巻き戻しのあと: %d 体、ソルジャー %v", len(c.Battle().Units()), c.Battle().Units()[2].Pos)
		}
		if c.WatchedCount() != 1 || !reflect.DeepEqual(c.WatchedPins(), []tactics.Pos{{X: 4}}) || !reflect.DeepEqual(c.watched, map[int]bool{2: true}) {
			t.Errorf("巻き戻しのあとの固定 %v、ピン %v", c.watched, c.WatchedPins())
		}
		m, a = coreRanges(c.Battle(), 2)
		watchedIs(t, c, "通路の巻き戻しのあと", m, a)
		checkMinimap("通路の巻き戻しのあと", c, []tactics.Pos{{X: 4}})

		// 相手軍の重騎士がいる盤面（#173、DES-BTL-11）: 範囲は ZOC を含む core の Reachable から求める。
		// 敵 (0,1) は移動 2。(1,0)・(2,1) は自軍の重騎士 (2,0) の隣なので入る費用が 2 で届かず、
		// 重騎士 (2,0) と (3,1) は攻撃できない（ZOC がなければどちらも攻撃できる）。
		z := newIdleWith(t, func() tactics.Setup {
			armor := tactics.UnitSetup{Name: "重騎士", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 2, Y: 0}, Weapon: testLance, ClassSkills: []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}}
			enemy := tactics.UnitSetup{Name: "敵", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20, Mov: 2}, Pos: tactics.Pos{X: 0, Y: 1}, Weapon: testSword, AI: tactics.Hold}
			boss := tactics.UnitSetup{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 5, Y: 0}, Weapon: testSword, Boss: true, AI: tactics.Hold}
			return tactics.Setup{Terrain: field(6, 3), Units: []tactics.UnitSetup{{Name: "主人公", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 5, Y: 2}, Weapon: testSword, Lord: true}, armor, enemy, boss}}
		})
		gz := &Game{ctrl: z}
		z.ToggleDanger()
		z.SetCursor(-1, -1)
		gz.leftClick(cellCenter(tactics.Pos{X: 0, Y: 1}))
		zocDanger := posSet(
			tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0},
			tactics.Pos{X: 0, Y: 1}, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 2, Y: 1},
			tactics.Pos{X: 0, Y: 2}, tactics.Pos{X: 1, Y: 2}, tactics.Pos{X: 2, Y: 2},
			tactics.Pos{X: 4, Y: 0}, tactics.Pos{X: 5, Y: 1},
		)
		if got := z.Danger(); !reflect.DeepEqual(got, zocDanger) || !reflect.DeepEqual(got, coreDanger(z.Battle(), 2, 3)) {
			t.Errorf("ZOC の盤面の全敵 %v\nwant %v", got, zocDanger)
		}
		m, a = coreRanges(z.Battle(), 2)
		if m[tactics.Pos{X: 1, Y: 0}] || m[tactics.Pos{X: 2, Y: 1}] || a[tactics.Pos{X: 3, Y: 1}] {
			t.Fatalf("前提: core の到達が ZOC を数えていない %v %v", m, a)
		}
		watchedIs(t, z, "ZOC の盤面の固定", m, a)
		z.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 1}))
		cursorIs(t, z, "ZOC の盤面のカーソル", m, a)
	})

	t.Run("TC-FR-CMP-11-13 配置の Q は戦闘と同じ層で塗り、入れ替えごとに求め直し、まだ出ない増援を含めない。カーソルの敵は青白の線と名前の印。敵のクリックで配置は変わらない", func(t *testing.T) {
		s := deployRangeScenes(t)
		layers := func() []EnemyRangeLayer { return s.View().Deploy.RangeLayers() }
		if ls := layers(); len(ls) != 0 {
			t.Fatalf("初め: %v", ls)
		}
		// 配置の全敵 Q の状態は、右欄の操作行ではなく案内欄 (16,552) に出す（E-124、DES-IT07 v0.5 3.2 節の配置の例外）。
		guideIs := func(what, want string) {
			t.Helper()
			found := false
			for _, tx := range s.View().Texts {
				if tx.Text == want && tx.X == 16 && tx.Y == 552 {
					found = true
				}
				if tx.Text == "全敵 Q ON" || tx.Text == "全敵 Q OFF" {
					t.Errorf("%s: 配置に右欄の範囲の状態 %+v がある", what, tx)
				}
			}
			if !found {
				t.Errorf("%s: 案内欄 (16,552) に %q がない", what, want)
			}
		}
		guideIs("初め", "Q: 敵の危険範囲 OFF")
		s.ToggleDeployDanger()
		guideIs("Q", "Q: 敵の危険範囲 ON")
		// TC-FR-CMP-11-10 の盤面: 弓 (4,0) は Mov 2・射程 2 で主人公 (2,0)・仲間 (0,0) を通れない。ボス (6,0) は Mov 0。
		// 全敵は x=1,2,3,5,6。3 ターン目の増援 (5,0) は含めない。
		ls := layers()
		if !reflect.DeepEqual(layerSummary(ls), []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}}) || !reflect.DeepEqual(ls[0].Cells, rowSet(1, 2, 3, 5, 6)) {
			t.Fatalf("Q の層 %v", ls)
		}
		if d := s.View().Deploy; len(d.Units) != 4 {
			t.Fatalf("前提: 増援を含めない 4 体のはず、%d 体", len(d.Units))
		}
		mv := s.View().Minimap
		var want []image.Rectangle
		for x := 0; x < 7; x++ {
			if ls[0].Cells[tactics.Pos{X: x}] {
				want = append(want, mv.Layout.CellRect(tactics.Pos{X: x}))
			}
		}
		if !reflect.DeepEqual(mv.Danger, want) || len(mv.Pins) != 0 {
			t.Errorf("配置のミニマップ: 危険 %v、ピン %v、want %v", mv.Danger, mv.Pins, want)
		}
		// 敵にカーソルを合わせると、全敵の塗りの上に青白の実線（移動）・破線（攻撃）を重ね、名前の横に同じ印を出す。
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		ls = layers()
		wantLayers := []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}, {Owner: RangeCursor, Stroke: StrokeSolid}, {Owner: RangeCursor, Stroke: StrokeDashed}}
		if !reflect.DeepEqual(layerSummary(ls), wantLayers) {
			t.Fatalf("カーソルの層 %v", layerSummary(ls))
		}
		framesIs(t, "配置のカーソル", ls[1].Cells, ls[2].Cells, rowSet(3, 4, 5), rowSet(1, 2, 6))
		v := s.View()
		name := textOf(t, *v.Info, "弓兵")
		if wantMark := image.Rect(name.X+textWidth(name.Text)+4, name.Y+3, name.X+textWidth(name.Text)+14, name.Y+13); v.Deploy.NameMark != wantMark {
			t.Errorf("名前の横の印 %v、want %v", v.Deploy.NameMark, wantMark)
		}
		// 敵のクリックは配置を変えない（選んでいる味方も保つ）。
		clickDeployCell(s, tactics.Pos{X: 2})
		selected := s.deploySelected
		clickDeployCell(s, tactics.Pos{X: 4})
		if s.deploySelected != selected || !reflect.DeepEqual(s.deploySlots, []int{0, 1}) {
			t.Fatalf("敵クリックが配置を変えた: %d、%v", s.deploySelected, s.deploySlots)
		}
		// 味方 (2,0) を空き枠 (1,0) へ移すと、求め直して全敵は x=0〜6。カーソルを離すと印も消える。
		s.SetDeployCursor(-1, -1)
		clickDeployCell(s, tactics.Pos{X: 1})
		ls = layers()
		if len(ls) != 1 || !reflect.DeepEqual(ls[0].Cells, rowSet(0, 1, 2, 3, 4, 5, 6)) || !s.View().Deploy.NameMark.Empty() {
			t.Errorf("入れ替えのあと: %v", ls)
		}
	})
}

package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// hintSetup は 16 × 16 の平地で、増援が 2T に 2 方面、3T に 1 方面から来る戦闘（DES-RNF-01、DES-UI-24）。
//
//	0: レオン (0, 0)。主人公
//	1: ガルド (0, 2)
//	2: ボス (15, 15)。移動 0、待ち構える
//	増援: 2T に (9, 5) と (2, 13)、3T に (14, 1)。いずれも待ち構える
//
// 予兆の領域は DES-IT04 6.2 節の丸め（距離 2 の範囲を 4 マス単位に広げ、盤内に切る）から次になる。
//
//	2T: (9, 5) → x 4〜11・y 0〜7、(2, 13) → x 0〜7・y 8〜15（左上 Y の昇順）
//	3T: (14, 1) → x 12〜15・y 0〜3
func hintSetup() tactics.Setup {
	reinforce := func(turn int, name string, p tactics.Pos) tactics.ReinforcementSetup {
		return tactics.ReinforcementSetup{Turn: turn, Unit: tactics.UnitSetup{Name: name, ClassName: "ソルジャー", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: p, Weapon: testLance, AI: tactics.Hold}}
	}
	return tactics.Setup{
		Terrain: field(16, 16),
		Units: []tactics.UnitSetup{
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testSword, Lord: true},
			{Name: "ガルド", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 2}, Weapon: testLance},
			{Name: "ボス", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 15, Y: 15}, Weapon: testLance, Boss: true, AI: tactics.Hold},
		},
		Reinforcements: []tactics.ReinforcementSetup{
			reinforce(2, "増援東", tactics.Pos{X: 9, Y: 5}),
			reinforce(2, "増援南", tactics.Pos{X: 2, Y: 13}),
			reinforce(3, "増援北", tactics.Pos{X: 14, Y: 1}),
		},
	}
}

func area(x0, y0, x1, y1 int) tactics.ReinforcementHint {
	return tactics.ReinforcementHint{Min: tactics.Pos{X: x0, Y: y0}, Max: tactics.Pos{X: x1, Y: y1}}
}

var (
	hintAreasT2 = []tactics.ReinforcementHint{area(4, 0, 11, 7), area(0, 8, 7, 15)}
	hintAreasT3 = []tactics.ReinforcementHint{area(12, 0, 15, 3)}
)

// endTurn は番を終え（未行動の確認を含む）、敵軍の番と次の自軍の番の開始まで再生する。
func endTurn(c *Controller) {
	c.EndPhase()
	c.EndPhase()
	playOut(c)
}

// textRect は文の画面上の範囲。
func textRect(t SceneText) image.Rectangle {
	return image.Rect(t.X, t.Y, t.X+textWidth(t.Text), t.Y+lineHeight)
}

// checkHintText は案内が正確な情報を含まない固定の文であることを確かめる（DES-UI-24）。
func checkHintText(t *testing.T, step string, s string) {
	t.Helper()
	if s != hintText {
		t.Fatalf("%s: 案内 %q、期待 %q", step, s, hintText)
	}
	for _, r := range s {
		if unicode.IsDigit(r) {
			t.Fatalf("%s: 案内に数字 %q", step, s)
		}
	}
	for _, w := range []string{"増援", "ソルジャー", "ターン", "T", "体", "人"} {
		if strings.Contains(s, w) {
			t.Fatalf("%s: 案内に %q", step, w)
		}
	}
}

// checkNormalHint は通常の盤面の予兆の表示値を確かめる。want が空なら隠れていること。
func checkNormalHint(t *testing.T, c *Controller, step string, want []tactics.ReinforcementHint) {
	t.Helper()
	v := c.ReinforcementHintView()
	if len(want) == 0 {
		if v.Visible || len(v.Areas) != 0 || v.Text.Text != "" {
			t.Fatalf("%s: 予兆を出している %+v", step, v)
		}
		return
	}
	if !v.Visible || !reflect.DeepEqual(v.Areas, want) {
		t.Fatalf("%s: 予兆 %+v、期待 %v", step, v, want)
	}
	checkHintText(t, step, v.Text.Text)
}

// checkGuideLayout は、案内欄の文・ボタンと気配の案内が 832 × 576 に収まり、重ならないことを確かめる（TC-FR-UNT-09-12）。
func checkGuideLayout(t *testing.T, c *Controller, step string) {
	t.Helper()
	v := c.ReinforcementHintView()
	if !v.Visible {
		t.Fatalf("%s: 予兆が出ていない", step)
	}
	r := textRect(v.Text)
	if !r.In(image.Rect(0, MapSize, MapSize, ScreenHeight)) {
		t.Fatalf("%s: 気配の案内が案内欄の外 %v", step, r)
	}
	for j, line := range c.Guide() {
		if j >= GuideHeight/lineHeight {
			break
		}
		y := MapSize + 2 + j*lineHeight
		if r.Overlaps(image.Rect(8, y, 8+textWidth(line), y+lineHeight)) {
			t.Fatalf("%s: 気配の案内が案内文 %q と重なる %v", step, line, r)
		}
	}
	rv := c.RewindView()
	for _, b := range rv.Buttons {
		if r.Overlaps(b.Rect) {
			t.Fatalf("%s: 気配の案内がボタン %q と重なる", step, b.Label)
		}
	}
	for _, tx := range rv.Texts {
		if r.Overlaps(textRect(tx)) {
			t.Fatalf("%s: 気配の案内が %q と重なる", step, tx.Text)
		}
	}
}

func TestReinforcementHintNormal(t *testing.T) {
	t.Run("TC-FR-MAP-03-11 通常の盤面は今の戦闘の予兆を出し、再生中は先に進んだ本物の値を混ぜない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		checkNormalHint(t, c, "1T 自軍", nil)

		c.EndPhase()
		c.EndPhase()
		if c.State() != Playing {
			t.Fatalf("状態 %v、再生中のはず", c.State())
		}
		// 本物の戦闘は 2T の自軍の入力待ちまで進み、予兆がある。
		if got := c.battle.BoardView().ReinforcementHints(); !reflect.DeepEqual(got, hintAreasT2) {
			t.Fatalf("本物の予兆 %v、期待 %v", got, hintAreasT2)
		}
		for c.State() == Playing {
			checkNormalHint(t, c, "1T→2T の再生中", nil)
			if c.hints != nil {
				t.Fatalf("再生中に予兆の値を取り直した %v", c.hints)
			}
			c.Tick()
		}
		if turn, _ := c.Phase(); turn != 2 {
			t.Fatalf("表示中のターン %d", turn)
		}
		checkNormalHint(t, c, "2T 自軍", hintAreasT2)

		// 選択・移動・行動の一覧の間も同じ予兆を出す。
		c.Click(tactics.Pos{X: 0, Y: 2})
		checkNormalHint(t, c, "2T Selected", hintAreasT2)
		c.Click(tactics.Pos{X: 1, Y: 2})
		checkNormalHint(t, c, "2T Moved", hintAreasT2)
		c.ChooseAction(tactics.Wait)
		checkNormalHint(t, c, "2T 待機の再生中", nil)
		playOut(c)
		checkNormalHint(t, c, "2T 待機の後", hintAreasT2)

		endTurn(c)
		checkNormalHint(t, c, "3T 自軍", hintAreasT3)
		// 出現した 2T の増援は、2T の予兆の領域の中にいる（正確なマスは示さないが、領域は外さない）。
		for _, u := range c.battle.Units() {
			if !strings.HasPrefix(u.Name, "増援") {
				continue
			}
			in := false
			for _, a := range hintAreasT2 {
				in = in || (u.Pos.X >= a.Min.X && u.Pos.X <= a.Max.X && u.Pos.Y >= a.Min.Y && u.Pos.Y <= a.Max.Y)
			}
			if !in {
				t.Fatalf("%s %v が 2T の予兆の外", u.Name, u.Pos)
			}
		}
		endTurn(c)
		checkNormalHint(t, c, "4T 自軍（予定なし）", nil)
	})

	t.Run("TC-FR-MAP-03-11 勝敗の確定後（Over）と巻き戻しの一覧を開いている間は通常の予兆を出さない", func(t *testing.T) {
		lost := func() tactics.Setup {
			s := hintSetup()
			s.Units[0].Stats.HP = 1
			s.Units = append(s.Units, tactics.UnitSetup{Name: "ファイター", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30, Str: 5}, Pos: tactics.Pos{X: 1, Y: 0}, Weapon: testAxe, AI: tactics.Hold})
			return s
		}
		o := newIdleWith(t, lost)
		endTurn(o)
		if o.State() != Over {
			t.Fatalf("状態 %v、Over のはず", o.State())
		}
		checkNormalHint(t, o, "Over", nil)

		c := newIdleWith(t, hintSetup)
		endTurn(c)
		c.OpenRewind()
		checkNormalHint(t, c, "一覧を開いている間", nil)
		c.Cancel()
		checkNormalHint(t, c, "一覧を閉じた後", hintAreasT2)
	})

	t.Run("TC-FR-MAP-03-11 配置の画面には予兆を出さない", func(t *testing.T) {
		s := prepareScenes(t)
		clickSceneButton(t, s, "配置へ")
		if s.Scene() != Deploy || s.battle != nil {
			t.Fatalf("配置の画面に入らない（場面 %v）", s.Scene())
		}
		if strings.Contains(viewStrings(s.View()), "気配") {
			t.Fatal("配置に気配の案内がある")
		}
	})

	t.Run("TC-FR-MAP-03-11 / TC-FR-UNT-09-12 気配の案内は通常位置で案内文・巻き戻しの入口と重ならず再開操作はない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		endTurn(c)
		checkGuideLayout(t, c, "2T Idle")
		c.message = strings.Repeat("長い知らせ", 20)
		checkGuideLayout(t, c, "2T Idle の知らせ")
		c.message = ""
		c.Click(tactics.Pos{X: 0, Y: 0})
		checkGuideLayout(t, c, "2T Selected")
		c.Click(tactics.Pos{X: 1, Y: 1})
		checkGuideLayout(t, c, "2T Moved")
		c.ChooseItem(0)
		c.Cancel()
		c.Cancel()
		c.Cancel()
		c.EndPhase()
		checkGuideLayout(t, c, "2T 番終了の確認")
		c.EndPhase()
		playOut(c)

		// 2T の明示の番終了前は自軍入力待ちとなり、通常位置に同じ予兆を出す。
		h := c.battle.History()
		i := -1
		for j, e := range h {
			if e.Kind == tactics.HistoryEndPhase && e.Side == tactics.Player {
				i = j
			}
		}
		if i < 0 {
			t.Fatalf("2T の番終了がない %v", h)
		}
		c.OpenRewind()
		c.rewindSelect(i)
		c.RewindEnter()
		c.RewindEnter()
		if c.battle.NeedsResume() || c.battle.Turn != 2 {
			t.Fatalf("2T の番終了前の自軍入力待ちに戻っていない（Turn %d、再開 %v）", c.battle.Turn, c.battle.NeedsResume())
		}
		checkNormalHint(t, c, "2T 明示の番終了前（自軍入力待ち）", hintAreasT2)
		checkGuideLayout(t, c, "2T 自軍入力待ち")
		checkInputLayout(t, c)
		c.EndPhase()
		c.EndPhase()
		checkNormalHint(t, c, "番終了の再生中", nil)
		playOut(c)
		checkNormalHint(t, c, "番終了後の 3T", hintAreasT3)
	})
}

func TestReinforcementHintRewind(t *testing.T) {
	t.Run("TC-FR-MAP-03-11 巻き戻しの一覧は選択中の写しの予兆を出し、描画のたびに取得し直さない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		endTurn(c) // 2T 自軍
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		endTurn(c) // 3T 自軍
		h := c.battle.History()
		find := func(match func(tactics.HistoryEntry) bool) int {
			t.Helper()
			for j, e := range h {
				if match(e) {
					return j
				}
			}
			t.Fatalf("履歴に該当の行がない %v", h)
			return -1
		}
		wait2 := find(func(e tactics.HistoryEntry) bool { return e.Kind == tactics.HistoryAction && e.Side == tactics.Player })
		end1 := find(func(e tactics.HistoryEntry) bool { return e.Kind == tactics.HistoryEndPhase })
		start2 := find(func(e tactics.HistoryEntry) bool { return e.Kind == tactics.HistoryPlayerStart })
		cases := []struct {
			name string
			row  int
			want []tactics.ReinforcementHint
		}{
			{"1T の番終了の前", end1, nil},
			{"2T 自軍開始の前（番開始処理の前）", start2, nil},
			{"2T のガルドの待機の前", wait2, hintAreasT2},
			{"最新行", len(h) - 1, nil},
		}
		c.OpenRewind()
		for _, tc := range cases {
			c.rewindSelect(tc.row)
			preview, err := c.battle.PreviewRewind(h[tc.row].ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(preview.ReinforcementHints(), tc.want) {
				t.Fatalf("%s: tactics の写しの予兆 %v、期待 %v", tc.name, preview.ReinforcementHints(), tc.want)
			}
			loads := c.rewind.previewLoads
			for k := 0; k < 3; k++ {
				b := c.RewindBoard()
				texts := viewStrings(c.RewindView())
				if len(tc.want) == 0 {
					if b.Hint.Visible || strings.Contains(texts, hintText) {
						t.Fatalf("%s: 予兆を出している %+v", tc.name, b.Hint)
					}
					continue
				}
				if !b.Hint.Visible || !reflect.DeepEqual(b.Hint.Areas, tc.want) {
					t.Fatalf("%s: 盤面の予兆 %+v、期待 %v", tc.name, b.Hint, tc.want)
				}
				checkHintText(t, tc.name, b.Hint.Text.Text)
				if !strings.Contains(texts, hintText) {
					t.Fatalf("%s: 右欄に案内がない", tc.name)
				}
				// 現在（3T）の予兆を一覧に混ぜない。
				if strings.Contains(strings.Join(b.Lines, ""), hintText) {
					t.Fatalf("%s: 下の案内に気配の文がある %v", tc.name, b.Lines)
				}
			}
			if c.rewind.previewLoads != loads {
				t.Fatalf("%s: 描画で取得し直した（%d → %d）", tc.name, loads, c.rewind.previewLoads)
			}
		}

		// 確認でも同じ写しの予兆を出す。取り消せば一覧に戻り、本物の予兆（3T）は変わらない。
		c.rewindSelect(wait2)
		c.RewindEnter()
		if !c.rewind.confirm || !c.RewindBoard().Hint.Visible || !strings.Contains(viewStrings(c.RewindView()), hintText) {
			t.Fatal("確認で予兆を出していない")
		}
		c.Cancel()
		c.Cancel()
		checkNormalHint(t, c, "一覧を閉じた後", hintAreasT3)

		// 確定すれば、通常の予兆も戻した盤面（2T）の値になる。
		c.OpenRewind()
		c.rewindSelect(wait2)
		c.RewindEnter()
		c.RewindEnter()
		if c.rewind.open || c.battle.Turn != 2 {
			t.Fatalf("戻せていない（一覧 %v、Turn %d）", c.rewind.open, c.battle.Turn)
		}
		checkNormalHint(t, c, "確定後の 2T", hintAreasT2)
	})

	t.Run("TC-FR-MAP-03-11 履歴がなければ今の盤面の予兆に従い、取得に失敗したら出さない", func(t *testing.T) {
		c := newIdleWith(t, hintSetup)
		c.OpenRewind()
		if b := c.RewindBoard(); b.Hint.Visible {
			t.Fatalf("1T の履歴なしで予兆 %+v", b.Hint)
		}
		c.Cancel()
		endTurn(c)
		c.OpenRewind()
		c.rewindSelect(len(c.battle.History()) - 1)
		c.fetchPreview(1 << 40)
		if b := c.RewindBoard(); b.Hint.Visible || strings.Contains(viewStrings(c.RewindView()), hintText) {
			t.Fatalf("取得失敗で予兆 %+v", b.Hint)
		}
	})

	t.Run("TC-FR-MAP-03-11 / TC-FR-UNT-09-12 右欄の気配の案内は、一覧・確認の文・理由・ボタンと重ならない", func(t *testing.T) {
		long := strings.Repeat("長い名前", 100)
		c := newIdleWith(t, func() tactics.Setup { s := hintSetup(); s.Units[1].Name = long; return s })
		endTurn(c)
		right := image.Rect(rewindLeft, 0, rewindRight, ScreenHeight)
		check := func(step string) {
			t.Helper()
			hv := c.RewindBoard().Hint
			if !hv.Visible {
				t.Fatalf("%s: 予兆が出ていない", step)
			}
			r := textRect(hv.Text)
			if !r.In(right) {
				t.Fatalf("%s: 案内が右欄の外 %v", step, r)
			}
			v := c.RewindView()
			n := 0
			for _, tx := range v.Texts {
				if tx == hv.Text {
					n++
					continue
				}
				if r.Overlaps(textRect(tx)) {
					t.Fatalf("%s: 案内が %q と重なる", step, tx.Text)
				}
			}
			if n != 1 {
				t.Fatalf("%s: 案内が %d 個", step, n)
			}
			for _, b := range v.Buttons {
				if r.Overlaps(b.Rect) {
					t.Fatalf("%s: 案内がボタン %q と重なる", step, b.Label)
				}
			}
		}
		wait := func() {
			c.Click(tactics.Pos{X: 0, Y: 2})
			c.Click(tactics.Pos{X: 1, Y: 2})
			c.ChooseAction(tactics.Wait)
			playOut(c)
		}
		wait()
		c.OpenRewind()
		check("一覧")
		c.RewindDetail(1000)
		check("詳細の末尾")
		c.rewind.message = strings.Repeat("巻き戻せません。", 30)
		check("誤り")
		c.rewind.message = ""
		c.RewindEnter()
		if !c.rewind.confirm {
			t.Fatal("確認に進まない")
		}
		check("確認")
		c.RewindDetail(1000)
		check("確認の末尾")
		c.RewindEnter()
		// 2T の待機と巻き戻しをくり返して残り 0 回にし、理由の行（y=456〜）も出す。
		for c.battle.RewindsLeft() > 0 {
			wait()
			c.OpenRewind()
			c.RewindEnter()
			c.RewindEnter()
			if c.rewind.open {
				t.Fatal("戻せない")
			}
		}
		wait()
		c.OpenRewind()
		if !strings.Contains(viewStrings(c.RewindView()), "残り回数が 0") {
			t.Fatal("理由の行がない")
		}
		check("残り 0 回の理由")
	})
}

// 予兆の形は外周の破線と中央の印だけで、領域の内側を塗らず、盤面の外に出ない（DES-UI-24）。
func TestReinforcementHintShapes(t *testing.T) {
	t.Run("TC-FR-MAP-03-11 破線は領域の外周の帯にだけあり、重なる領域でも同じ線を重ねない", func(t *testing.T) {
		areas := []tactics.ReinforcementHint{area(0, 0, 7, 7), area(4, 0, 7, 3), area(0, 0, 7, 7)}
		dashes, marks := hintShapes(areas)
		if len(dashes) == 0 {
			t.Fatal("破線がない")
		}
		seen := map[image.Rectangle]bool{}
		board := image.Rect(0, 0, MapSize, MapSize)
		for _, d := range dashes {
			if seen[d] {
				t.Fatalf("同じ線が 2 本 %v", d)
			}
			seen[d] = true
			if !d.In(board) {
				t.Fatalf("盤面の外 %v", d)
			}
			onEdge := false
			for _, a := range areas {
				outer := image.Rect(a.Min.X*CellSize, a.Min.Y*CellSize, (a.Max.X+1)*CellSize, (a.Max.Y+1)*CellSize)
				inner := outer.Inset(hintInset + hintThick)
				if d.In(outer) && !d.Overlaps(inner) {
					onEdge = true
				}
			}
			if !onEdge {
				t.Fatalf("外周の帯にない線 %v", d)
			}
		}
		// 同じ辺（右辺 x=256）の線は重なる 2 つの領域で共有し、角でも縦横の線を重ねない。
		for i, d := range dashes {
			for _, e := range dashes[i+1:] {
				if d.Overlaps(e) {
					t.Fatalf("線が重なる %v %v", d, e)
				}
			}
		}
		if len(marks) != 2 {
			t.Fatalf("印 %d 個、期待 2（同じ矩形は 1 つ）", len(marks))
		}
		want := image.Pt(128-textWidth(hintMark)/2, 128-lineHeight/2)
		if marks[0] != want {
			t.Fatalf("印 %v、期待 %v（8×8 マスの中央）", marks[0], want)
		}
	})
}

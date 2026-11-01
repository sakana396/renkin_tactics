package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 画面の操作 4 点（DES-UI-27、DES-IT06 12.4 節。#128）。

// quickBow は射程 2 だけの弓、quickSword は命中 50 の剣（乱数の結果が判定に効くようにする）。
var (
	quickSword = tactics.Weapon{Name: "剣", Type: tactics.Sword, Might: 3, Hit: 50, MinRange: 1, MaxRange: 1}
	quickBow   = tactics.Weapon{Name: "弓", Type: tactics.Bow, Might: 3, Hit: 50, MinRange: 2, MaxRange: 2}
)

// quickSetup は、敵をクリックして攻撃する操作の盤面。7 × 7 の平地で、(4, 1) は壁。
//
//	0: 兵長 (1, 1)。主人公。移動 3。剣（射程 1）と弓（射程 2）
//	1: 近くの敵 (1, 2)。兵長の隣
//	2: 敵 (3, 3)。兵長から距離 4
//	3: 壁の向こうの敵 (5, 1)。隣の (4, 1) は壁
//	4: 遠くの敵 (6, 6)。ボス
//
// 敵はどれも移動 0 で待ち構える。
func quickSetup() tactics.Setup {
	terrain := field(7, 7)
	terrain[1][4] = tactics.Terrain{Name: "壁", Cost: 0}
	enemy := func(name string, x, y int) tactics.UnitSetup {
		return tactics.UnitSetup{Name: name, Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: x, Y: y}, Weapon: testLance, AI: tactics.Hold}
	}
	far := enemy("遠くの敵", 6, 6)
	far.Boss = true
	return tactics.Setup{Terrain: terrain, Units: []tactics.UnitSetup{
		{Name: "兵長", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 3, Skl: 5}, Pos: tactics.Pos{X: 1, Y: 1},
			Weapons: []tactics.Weapon{quickSword, quickBow}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Sword, tactics.Bow}},
		enemy("近くの敵", 1, 2),
		enemy("敵", 3, 3),
		enemy("壁の向こうの敵", 5, 1),
		far,
	}}
}

// quickOneSetup は、攻撃の武器を 1 本だけ持つ自軍の盤面。7 × 4 の平地で、(3, 1) は壁。
// 剣士 (3, 0)（移動 3、剣だけ）と、壁の下の敵 (3, 2)。敵の隣で行けるのは (2, 2) と (4, 2)（どちらも費用 3）。
func quickOneSetup() tactics.Setup {
	terrain := field(7, 4)
	terrain[1][3] = tactics.Terrain{Name: "壁", Cost: 0}
	return tactics.Setup{Terrain: terrain, Units: []tactics.UnitSetup{
		{Name: "剣士", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 3, Y: 0}, Weapon: quickSword},
		{Name: "壁の下の敵", Side: tactics.Enemy, Level: 1, Boss: true, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 3, Y: 2}, Weapon: testLance, AI: tactics.Hold},
	}}
}

// overlapSetup は、敵 2 体の範囲が重なる 7 × 1 の通路。
//
//	0: 主人公 (6, 0)。移動 0
//	1: 敵 A (2, 0)。移動 1、射程 1
//	2: 敵 B (4, 0)。移動 1、射程 1
//
// A は (1, 0)〜(3, 0) に動けて、B のいる (4, 0) には止まれないので、A の赤は (0, 0) と (4, 0)。
// B は (3, 0)〜(5, 0) に動けて、赤は (2, 0) と主人公のいる (6, 0)。
func overlapSetup() tactics.Setup {
	return tactics.Setup{Terrain: field(7, 1), Units: []tactics.UnitSetup{
		{Name: "主人公", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 6, Y: 0}, Weapon: testSword},
		{Name: "敵 A", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Mov: 1}, Pos: tactics.Pos{X: 2, Y: 0}, Weapon: testSword, AI: tactics.Hold},
		{Name: "敵 B", Side: tactics.Enemy, Level: 1, Boss: true, Stats: tactics.Stats{HP: 30, Mov: 1}, Pos: tactics.Pos{X: 4, Y: 0}, Weapon: testSword, AI: tactics.Hold},
	}}
}

// killSetup は、主人公 (0, 0)（必ず当たって倒す剣）の隣に HP 1 の敵 (1, 0)、離れたボス (5, 5) を置く 6 × 6 の平地。
func killSetup() tactics.Setup {
	strong := tactics.Weapon{Name: "剣", Type: tactics.Sword, Might: 50, Hit: 300, MinRange: 1, MaxRange: 1}
	return tactics.Setup{Terrain: field(6, 6), Units: []tactics.UnitSetup{
		{Name: "主人公", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: strong},
		{Name: "ボス", Side: tactics.Enemy, Level: 1, Boss: true, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 5, Y: 5}, Weapon: testLance, AI: tactics.Hold},
		{Name: "弱い敵", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 1}, Pos: tactics.Pos{X: 1, Y: 0}, Weapon: testLance, AI: tactics.Hold},
	}}
}

// overlayIs は、Controller の範囲（Overlay）が青 wantBlue・赤 wantRed であることを確かめる。空と nil は同じに扱う。
func overlayIs(t *testing.T, c *Controller, what string, wantBlue, wantRed map[tactics.Pos]bool) {
	t.Helper()
	blue, red := c.Overlay()
	if len(blue) == 0 && len(wantBlue) == 0 && len(red) == 0 && len(wantRed) == 0 {
		return
	}
	if !reflect.DeepEqual(blue, wantBlue) || !reflect.DeepEqual(red, wantRed) {
		t.Errorf("%s: 青 %v、赤 %v\nwant 青 %v、赤 %v", what, blue, red, wantBlue, wantRed)
	}
}

func TestUIControls(t *testing.T) {
	t.Run("TC-FR-CMP-11-12 配置で初めの敵にカーソルを当てると能力を 2 ページで出し、クリックしても配置を変えない。まだ出ていない増援は見せない", func(t *testing.T) {
		s := deployRangeScenes(t)
		baseline := campaignBytes(t, s)
		infoRows := func() []string {
			t.Helper()
			v := s.View()
			if v.Info == nil {
				return nil
			}
			if len(v.Info.Tabs) != 2 {
				t.Errorf("ページの印 %d 個", len(v.Info.Tabs))
			}
			return rows(*v.Info)
		}
		unchanged := func(what string, selected int, slots []int) {
			t.Helper()
			if s.deploySelected != selected || !reflect.DeepEqual(s.deploySlots, slots) || !bytes.Equal(baseline, campaignBytes(t, s)) || s.battle != nil {
				t.Errorf("%s: 選択 %d、枠 %v（%d・%v のはず）", what, s.deploySelected, s.deploySlots, selected, slots)
			}
		}
		// 選んでいない。弓兵 (4, 0) にカーソルを当てると、弓兵の能力（「能力」のページ）を出し、案内の文は出さない。
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		got := infoRows()
		if len(got) < 2 || got[0] != "弓兵" || !strings.Contains(strings.Join(got, "\n"), "力 4") || strings.Contains(viewStrings(s.View()), "配置する味方を選んでください") {
			t.Errorf("弓兵: %q", got)
		}
		// Tab で「装備・スキル」に切り替えると、弓兵の武器を出す。表示だけで、選択と配置は変わらない。
		s.ToggleInfoPage()
		if got := strings.Join(infoRows(), "\n"); !strings.Contains(got, "武器: 弓（弓）") {
			t.Errorf("弓兵の装備: %q", got)
		}
		unchanged("Tab", -1, []int{0, 1})
		s.ToggleInfoPage()
		// 3 ターン目の増援の位置 (5, 0) は空きマス。増援は見せず、情報も出さない。
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 5}))
		if got := infoRows(); got != nil || !strings.Contains(viewStrings(s.View()), "配置する味方を選んでください") {
			t.Errorf("増援の位置: %q", got)
		}
		for _, u := range s.View().Deploy.Units {
			if u.Pos == (tactics.Pos{X: 5}) {
				t.Error("まだ出ていない増援を配置の盤面に出した")
			}
		}
		// 主人公 (2, 0) を選ぶ。空きマス (3, 0) と盤面の外では選んでいる主人公、ボス (6, 0) ではボスを出す。
		clickDeployCell(s, tactics.Pos{X: 2})
		for _, xy := range [][2]int{{3*CellSize + 16, 16}, {100, 540}, {650, 400}} {
			s.SetDeployCursor(xy[0], xy[1])
			if got := infoRows(); len(got) == 0 || got[0] != "主人公" {
				t.Errorf("選んでいるとき %v: %q", xy, got)
			}
		}
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 6}))
		if got := infoRows(); len(got) == 0 || got[0] != "騎兵 1　★ボス" {
			t.Errorf("ボス: %q", got)
		}
		unchanged("ホバー", 0, []int{0, 1})
		// 敵をクリックしても、味方の配置を入れ替えず、選択も保つ。
		clickDeployCell(s, tactics.Pos{X: 4})
		clickDeployCell(s, tactics.Pos{X: 6})
		unchanged("敵のクリック", 0, []int{0, 1})
		// 選択を外し、ボスを見てから情報欄の印へ動かすと、ボスを出し続け、印のクリックでページを変える。
		s.Cancel()
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 6}))
		tab := InfoTabRects()[1]
		s.SetDeployCursor(tab.Min.X+2, tab.Min.Y+2)
		s.Click(tab.Min.X+2, tab.Min.Y+2)
		if got := strings.Join(infoRows(), "\n"); s.infoPage != PageEquip || !strings.HasPrefix(got, "騎兵 1　★ボス") || !strings.Contains(got, "武器: 剣（剣）") {
			t.Errorf("印のクリック: ページ %d、%q", s.infoPage, got)
		}
		// ミニマップの上では出し続けない。
		s.SetDeployCursor(650, 400)
		if got := infoRows(); got != nil {
			t.Errorf("ミニマップの上: %q", got)
		}
		unchanged("印のクリック", -1, []int{0, 1})
	})

	t.Run("TC-FR-UNT-09-21 情報欄は 予測 → カーソルの下のユニット → 選んでいるユニット → 地形 の順に出し、盤の外や空いたマスでは選んでいるユニットに戻る", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		leon, boss := tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3}
		first := func(x, y int) string {
			t.Helper()
			got := rows(c.InfoView(x, y))
			if len(got) == 0 {
				return ""
			}
			return got[0]
		}
		last := func(x, y int) string {
			t.Helper()
			got := rows(c.InfoView(x, y))
			for _, r := range got {
				if strings.HasPrefix(r, "地形: ") {
					return r
				}
			}
			return ""
		}
		// Idle（選んでいない）: ユニット → 地形。盤の外は空。
		for _, tc := range []struct {
			name      string
			x, y      int
			want, ter string
		}{
			{"レオン", 2*CellSize + 16, 3*CellSize + 16, "レオン", "地形: 平地"},
			{"ボス", 3*CellSize + 16, 3*CellSize + 16, "ソルジャー 3　★ボス", "地形: 森"},
			{"空きマス", 16, 16, "地形: 平地", "地形: 平地"},
			{"案内欄", 100, 540, "", ""},
			{"ミニマップ", 650, 400, "", ""},
		} {
			if got, ter := first(tc.x, tc.y), last(tc.x, tc.y); got != tc.want || ter != tc.ter {
				t.Errorf("Idle の %s: %q・%q（%q・%q のはず）", tc.name, got, ter, tc.want, tc.ter)
			}
		}
		// レオンを選ぶ（Selected）。カーソルの下のボスを優先し、空きマス・盤の外ではレオン（レオンのマスの地形）に戻る。
		c.Click(leon)
		bx, by := cellCenter(boss)
		if got := first(bx, by); got != "ソルジャー 3　★ボス" {
			t.Errorf("Selected のボス: %q", got)
		}
		for _, xy := range [][2]int{{16, 16}, {6*CellSize + 16, 6*CellSize + 16}, {100, 540}, {650, 400}, {700, 100}} {
			if got, ter := first(xy[0], xy[1]), last(xy[0], xy[1]); got != "レオン" || ter != "地形: 平地" {
				t.Errorf("Selected の %v: %q・%q", xy, got, ter)
			}
		}
		// 移動先 (2, 2) を選んだ Moved では、移動先に置いたレオンに戻る。表示の切り替えで選択と状態は変わらない。
		c.Click(tactics.Pos{X: 2, Y: 2})
		before := c.Battle().Units()
		if got := first(16, 16); got != "レオン" || c.State() != Moved || c.dest != (tactics.Pos{X: 2, Y: 2}) {
			t.Errorf("Moved: %q、状態 %v、移動先 %v", got, c.State(), c.dest)
		}
		if got := first(cellCenter(tactics.Pos{X: 2, Y: 2})); got != "レオン" {
			t.Errorf("Moved の移動先のマス: %q", got)
		}
		// 今いるマスにとどめて攻撃し、相手を選ぶと、予測の表示を優先する（ユニットの行とページの印を出さない）。
		c.Cancel()
		c.Click(leon)
		c.ChooseAction(tactics.Attack)
		c.Click(boss)
		for _, xy := range [][2]int{{16, 16}, {bx, by}, {100, 540}} {
			v := c.InfoView(xy[0], xy[1])
			if got := strings.Join(rows(v), "\n"); len(v.Tabs) != 0 || strings.Contains(got, "クラス: ") || !strings.Contains(got, "命中率") {
				t.Errorf("予測 %v: 印 %d、%q", xy, len(v.Tabs), got)
			}
		}
		if c.State() != Targeting || c.target != 1 || !reflect.DeepEqual(c.Battle().Units(), before) {
			t.Errorf("表示で選択・戦闘が変わった: 状態 %v、相手 %d", c.State(), c.target)
		}
	})

	t.Run("TC-IF-01-42 敵の範囲は添字の集まりで持ち、クリックで足し・もう一度で消し、選択・スクロール・行動で保つ。重なりは合わせ、戦死と巻き戻しでは生きている敵だけ", func(t *testing.T) {
		// 重なり: A と B を足すと、青は (1, 0)〜(5, 0)、赤は (0, 0) と (6, 0)。A の赤 (4, 0) は B の青に、B の赤 (2, 0) は A の青に合わせる。
		c := newIdleWith(t, overlapSetup)
		c.Click(tactics.Pos{X: 2})
		c.Click(tactics.Pos{X: 4})
		c.SetCursor(-1, -1)
		overlayIs(t, c, "A と B", rowSet(1, 2, 3, 4, 5), rowSet(0, 6))
		// もう一度 A をクリックすると A だけ消える。
		c.Click(tactics.Pos{X: 2})
		overlayIs(t, c, "A を消す", rowSet(3, 4, 5), rowSet(2, 6))
		// カーソルを合わせた A の範囲は、出している B の範囲と合わせて出す（B を隠さない）。
		c.SetCursor(cellCenter(tactics.Pos{X: 2}))
		overlayIs(t, c, "A に合わせる", rowSet(1, 2, 3, 4, 5), rowSet(0, 6))
		c.SetCursor(-1, -1)
		c.Click(tactics.Pos{X: 2})
		// 空いたマスのクリック、スクロール、Q の切り替えでは集まりを変えない。Q の危険範囲は別に出す。
		c.Click(tactics.Pos{X: 0})
		c.Scroll(1, 0)
		c.ToggleDanger()
		overlayIs(t, c, "空きマス・スクロール・Q", rowSet(1, 2, 3, 4, 5), rowSet(0, 6))
		if got := c.Danger(); !reflect.DeepEqual(got, rowSet(0, 1, 2, 3, 4, 5, 6)) {
			t.Errorf("Q の危険範囲 %v", got)
		}
		c.ToggleDanger()
		overlayIs(t, c, "Q を消す", rowSet(1, 2, 3, 4, 5), rowSet(0, 6))
		if c.RangeStyle() != StyleBad {
			t.Error("敵の範囲の色が赤でない")
		}

		// 位置が変わったら計算し直す: 通路でソルジャー (4, 0) とボス (0, 0) を出す。
		// ソルジャーはレオン (2, 0) を通れず、青 (3, 0)〜(5, 0)、赤 (2, 0)・(6, 0)。ボスは青 (0, 0)、赤 (1, 0)。
		c = newIdleWith(t, corridorSetup)
		c.Click(tactics.Pos{X: 4})
		c.Click(tactics.Pos{X: 0})
		c.SetCursor(-1, -1)
		overlayIs(t, c, "ソルジャーとボス", rowSet(0, 3, 4, 5), rowSet(1, 2, 6))
		// レオンを選ぶと自軍の範囲を出し、集まりは保つ。レオンを (1, 0) へ動かして待機すると、自軍は全員行動済みなので
		// 敵軍の番になり、ソルジャーが (2, 0) に動いてレオンを攻撃する。ソルジャーは (2, 0) から右へ (5, 0) まで動けるので、
		// 青 (2, 0)〜(5, 0)、赤 (1, 0)・(6, 0)。ボスの赤 (1, 0) と合わせる。
		c.Click(tactics.Pos{X: 2})
		own, ownRed := c.ranges(0)
		overlayIs(t, c, "レオンを選ぶ", own, ownRed)
		// 自軍を選んで動かしている間も、出している敵の範囲を枠（WatchedFrames）で残す（E-60）。
		if blue, red := c.WatchedFrames(); !reflect.DeepEqual(blue, rowSet(0, 3, 4, 5)) || !reflect.DeepEqual(red, rowSet(1, 2, 6)) {
			t.Errorf("Selected の敵の範囲の枠: 青 %v、赤 %v", blue, red)
		}
		c.Click(tactics.Pos{X: 1})
		if blue, red := c.WatchedFrames(); !reflect.DeepEqual(blue, rowSet(0, 3, 4, 5)) || !reflect.DeepEqual(red, rowSet(1, 2, 6)) {
			t.Errorf("Moved の敵の範囲の枠: 青 %v、赤 %v", blue, red)
		}
		c.Cancel()
		c.Click(tactics.Pos{X: 1})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		if c.State() != Idle {
			t.Fatalf("待機のあと: 状態 %v", c.State())
		}
		if c.Battle().Units()[2].Pos != (tactics.Pos{X: 2}) {
			t.Fatalf("前提: ソルジャーの位置 %v", c.Battle().Units()[2].Pos)
		}
		overlayIs(t, c, "レオンが動いたあと", rowSet(0, 2, 3, 4, 5), rowSet(1, 6))
		// 2 ターン目の番を終えると、敵軍の番に増援 (9, 0) が現れる。範囲は保つ。
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		if c.State() != Idle || len(c.Battle().Units()) != 4 {
			t.Fatalf("3 ターン目: 状態 %v、ユニット %d", c.State(), len(c.Battle().Units()))
		}
		overlayIs(t, c, "敵軍の番のあと", rowSet(0, 2, 3, 4, 5), rowSet(1, 6))
		// 増援 (9, 0) も足す（青 (9, 0)、赤 (8, 0)）。
		c.Click(tactics.Pos{X: 9})
		overlayIs(t, c, "増援を足す", rowSet(0, 2, 3, 4, 5, 9), rowSet(1, 6, 8))
		// 巻き戻しに成功したら、同じ集まりを戻った盤面に照らす。最初の行動（レオンの待機）の前へ戻ると、増援はまだいないので除き、
		// ソルジャーとボスは戻った盤面（レオン (2, 0)、ソルジャー (4, 0)）で出す。
		c.OpenRewind()
		c.RewindMove(-100)
		for c.rewind.open {
			before := c.battle.RewindsLeft()
			c.RewindEnter()
			if c.rewind.open && !c.rewind.confirm && c.battle.RewindsLeft() == before {
				t.Fatal("巻き戻せない")
			}
		}
		if c.State() != Idle || len(c.Battle().Units()) != 3 {
			t.Fatalf("巻き戻しのあと: 状態 %v、ユニット %d", c.State(), len(c.Battle().Units()))
		}
		if !reflect.DeepEqual(c.watched, map[int]bool{1: true, 2: true}) {
			t.Errorf("巻き戻しのあとの集まり %v", c.watched)
		}
		overlayIs(t, c, "巻き戻しのあと", rowSet(0, 3, 4, 5), rowSet(1, 2, 6))

		// 戦死した敵は除く。弱い敵とボスを出し、主人公が弱い敵を倒すと、ボスの範囲だけが残る。
		k := newIdleWith(t, killSetup)
		k.Click(tactics.Pos{X: 1})
		k.Click(tactics.Pos{X: 5, Y: 5})
		k.SetCursor(-1, -1)
		k.Click(tactics.Pos{X: 0})
		k.Click(tactics.Pos{X: 0})
		k.ChooseAction(tactics.Attack)
		k.Click(tactics.Pos{X: 1})
		k.Click(tactics.Pos{X: 1})
		playOut(k)
		if k.Battle().Units()[2].Alive {
			t.Fatal("前提: 弱い敵が倒れていない")
		}
		if !reflect.DeepEqual(k.watched, map[int]bool{1: true}) {
			t.Errorf("戦死のあとの集まり %v", k.watched)
		}
		overlayIs(t, k, "戦死のあと", posSet(tactics.Pos{X: 5, Y: 5}), posSet(tactics.Pos{X: 4, Y: 5}, tactics.Pos{X: 5, Y: 4}))
		// Esc は集まりを空にする。別の Battle（新しい Controller）では空。
		k.Cancel()
		overlayIs(t, k, "Esc", nil, nil)
		if fresh := newIdleWith(t, killSetup); len(fresh.watched) != 0 {
			t.Errorf("新しい戦闘の集まり %v", fresh.watched)
		}
	})

	t.Run("TC-IF-01-43 移動先を選んでいるときに敵をクリックすると、武器を選び、仮の移動先から予測を出す。候補がなければ理由を出し、取り消しは 予測 → 相手 → 武器 → 行動の一覧 → 移動先", func(t *testing.T) {
		c := newIdleWith(t, quickSetup)
		before := c.Battle().Units()
		history := len(c.Battle().History())
		still := func(what string) {
			t.Helper()
			if !reflect.DeepEqual(c.Battle().Units(), before) || len(c.Battle().History()) != history {
				t.Errorf("%s: 決定の前に戦闘が変わった", what)
			}
		}
		c.Click(tactics.Pos{X: 1, Y: 1})
		// 遠くの敵 (6, 6) には、どの武器でも届く移動先がない。理由を出して移動先の選択を保つ（範囲も出さない）。
		c.Click(tactics.Pos{X: 6, Y: 6})
		if c.State() != Selected || c.Message() != "遠くの敵 を攻撃できる移動先がありません" || len(c.watched) != 0 {
			t.Errorf("候補なし: 状態 %v、知らせ %q、範囲 %v", c.State(), c.Message(), c.watched)
		}
		if !strings.Contains(strings.Join(c.Guide(), "\n"), "遠くの敵 を攻撃できる移動先がありません") {
			t.Error("案内欄に理由がない")
		}
		// 武器ごとの仮の移動先（今の位置 → 費用が最も小さい → Y・X の小さい）。
		for _, tc := range []struct {
			name   string
			enemy  tactics.Pos
			target int
			labels []string
			dest   [2]tactics.Pos // 剣・弓の仮の移動先
		}{
			// 隣の敵: 剣は今の位置。弓は最小の射程 2 のため、費用 1 の (1, 0)・(0, 1)・(2, 1) のうち Y の小さい (1, 0)。
			{"近くの敵", tactics.Pos{X: 1, Y: 2}, 1, []string{"剣", "弓"}, [2]tactics.Pos{{X: 1, Y: 1}, {X: 1, Y: 0}}},
			// 距離 4 の敵: 剣は費用 3 の (3, 2)・(2, 3) のうち Y の小さい (3, 2)。弓は費用 2 の (3, 1)・(2, 2) のうち (3, 1)。
			{"敵", tactics.Pos{X: 3, Y: 3}, 2, []string{"剣", "弓"}, [2]tactics.Pos{{X: 3, Y: 2}, {X: 3, Y: 1}}},
			// 壁の向こうの敵: 隣へは行けないので剣は使えない。弓は (3, 1)。
			{"壁の向こうの敵", tactics.Pos{X: 5, Y: 1}, 3, []string{"剣（使えない）", "弓"}, [2]tactics.Pos{{}, {X: 3, Y: 1}}},
		} {
			c.Click(tc.enemy)
			if c.State() != WeaponMenu || !reflect.DeepEqual(c.MenuLabels(), tc.labels) || c.dest != (tactics.Pos{X: 1, Y: 1}) || len(c.watched) != 0 {
				t.Fatalf("%s: 状態 %v、一覧 %v、移動先 %v", tc.name, c.State(), c.MenuLabels(), c.dest)
			}
			still(tc.name + " の一覧")
			for w := range 2 {
				if tc.labels[w] == "剣（使えない）" {
					c.ChooseWeapon(w)
					if c.State() != WeaponMenu || c.Message() != "剣 では 壁の向こうの敵 を攻撃できる移動先がありません" {
						t.Errorf("%s の剣: 状態 %v、知らせ %q", tc.name, c.State(), c.Message())
					}
					continue
				}
				c.ChooseWeapon(w)
				if c.State() != Targeting || c.dest != tc.dest[w] || c.weapon != w || c.target != tc.target {
					t.Errorf("%s の武器 %d: 状態 %v、移動先 %v、武器 %d、相手 %d", tc.name, w, c.State(), c.dest, c.weapon, c.target)
				}
				// 予測の上で移動先を示す（描くユニットは仮の移動先）。予測は選んだ武器で出す。
				p := c.Panel(tactics.Pos{}, false)
				if c.Units()[0].Pos != tc.dest[w] || len(p.Forecast) == 0 || p.ForecastWeapons[0] != []tactics.WeaponType{tactics.Sword, tactics.Bow}[w] {
					t.Errorf("%s の武器 %d の予測: 位置 %v、武器 %v", tc.name, w, c.Units()[0].Pos, p.ForecastWeapons)
				}
				still(tc.name + " の予測")
				// 予測 → 相手 → 武器の一覧（仮の移動先で） → 行動の一覧 → 移動先の選択。
				for i, want := range []struct {
					state  State
					target int
				}{{Targeting, -1}, {WeaponMenu, -1}, {Moved, -1}, {Selected, -1}} {
					c.Cancel()
					if c.State() != want.state || c.target != want.target {
						t.Errorf("%s の武器 %d の取り消し %d: 状態 %v、相手 %d", tc.name, w, i+1, c.State(), c.target)
					}
					if want.state != Selected && c.dest != tc.dest[w] {
						t.Errorf("%s の武器 %d の取り消し %d: 移動先 %v", tc.name, w, i+1, c.dest)
					}
				}
				still(tc.name + " の取り消し")
				c.Click(tc.enemy)
			}
			// 敵をクリックして出した一覧の取り消しは、移動先の選択へ戻る。
			c.Cancel()
			if c.State() != Selected {
				t.Errorf("%s の一覧の取り消し: 状態 %v", tc.name, c.State())
			}
		}

		// 決定はふつうの確認（同じ相手をもう一度クリック）で行う。同じ入力をふつうの流れで行った戦闘と結果が一致する（乱数を引かない）。
		control := newIdleWith(t, quickSetup)
		control.Click(tactics.Pos{X: 1, Y: 1})
		control.Click(tactics.Pos{X: 3, Y: 1})
		control.ChooseAction(tactics.Attack)
		control.ChooseWeapon(1)
		control.Click(tactics.Pos{X: 3, Y: 3})
		control.Click(tactics.Pos{X: 3, Y: 3})
		c.Click(tactics.Pos{X: 3, Y: 3})
		c.ChooseWeapon(1)
		c.Click(tactics.Pos{X: 3, Y: 3})
		if c.State() != Playing || control.State() != Playing {
			t.Fatalf("決定: 状態 %v・%v", c.State(), control.State())
		}
		if !reflect.DeepEqual(c.events, control.events) || !reflect.DeepEqual(c.Battle().Units(), control.Battle().Units()) {
			t.Error("敵をクリックした攻撃とふつうの流れで結果が違う")
		}

		// 攻撃の武器が 1 本だけなら一覧を省く。仮の移動先は費用 3 の (2, 2)・(4, 2) のうち X の小さい (2, 2)。
		// 取り消しは 予測 → 相手 → 行動の一覧 → 移動先。
		one := newIdleWith(t, quickOneSetup)
		one.Click(tactics.Pos{X: 3, Y: 0})
		one.Click(tactics.Pos{X: 3, Y: 2})
		if one.State() != Targeting || one.dest != (tactics.Pos{X: 2, Y: 2}) || one.target != 1 || one.weapon != 0 {
			t.Fatalf("武器 1 本: 状態 %v、移動先 %v、相手 %d、武器 %d", one.State(), one.dest, one.target, one.weapon)
		}
		for i, want := range []State{Targeting, Moved, Selected} {
			one.Cancel()
			if one.State() != want || one.target != -1 {
				t.Errorf("武器 1 本の取り消し %d: 状態 %v、相手 %d", i+1, one.State(), one.target)
			}
		}

		// 攻撃の武器がなければ理由を出す。Idle の敵のクリックは、今までどおり範囲を足す操作。
		healer := newIdleWith(t, func() tactics.Setup {
			s := quickOneSetup()
			s.Units[0].Weapon = tactics.Weapon{}
			s.Units[0].Weapons = []tactics.Weapon{menuStaff}
			s.Units[0].AllowedWeaponTypes = []tactics.WeaponType{tactics.Staff}
			return s
		})
		healer.Click(tactics.Pos{X: 3, Y: 0})
		healer.Click(tactics.Pos{X: 3, Y: 2})
		if healer.State() != Selected || healer.Message() != "攻撃の武器がないので 壁の下の敵 を攻撃できません" {
			t.Errorf("武器なし: 状態 %v、知らせ %q", healer.State(), healer.Message())
		}
		healer.Cancel()
		healer.Click(tactics.Pos{X: 3, Y: 2})
		if !reflect.DeepEqual(healer.watched, map[int]bool{1: true}) || healer.State() != Idle {
			t.Errorf("Idle の敵のクリック: 状態 %v、範囲 %v", healer.State(), healer.watched)
		}
	})
}

package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 爆弾の操作と表示（DES-NG-05）のテスト。実際の入力の経路（Game.leftClick → Controller）で確かめる。

var testBomb = tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: 8, BlastRadius: 1, ThrowRange: 3}

// bombSetup は 8 × 8 の平地。
//
//	0: 主人公 (2, 2) 移動 3、HP 20、爆弾（ダメージ 8・半径 1・投げる 3）× 1
//	1: 味方 (4, 2) HP 20
//	2: ボス (7, 7) 待ち構える
//	3: 敵 (5, 2) HP 8（爆発でちょうど倒れる）。待ち構える
//	4: 敵 (6, 2) HP 30。待ち構える
func bombSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: field(8, 8),
		Units: []tactics.UnitSetup{
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 2, Y: 2}, Weapon: testSword, Items: []tactics.ItemStack{{Item: testBomb, Count: 1}}, Lord: true},
			{Name: "ガルド", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 4, Y: 2}, Weapon: testLance},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 7, Y: 7}, Weapon: testLance, Boss: true, AI: tactics.Hold},
			{Name: "ファイター", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 8}, Pos: tactics.Pos{X: 5, Y: 2}, Weapon: testAxe, AI: tactics.Hold},
			{Name: "ソルジャー", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 6, Y: 2}, Weapon: testLance, AI: tactics.Hold},
		},
	}
}

// bombGame は、主人公を選び、移動先 dest を選び、行動の一覧から「アイテム」→ 爆弾を選んで BombTargeting にした Game を返す。
// 選ぶ操作はすべて Game.leftClick（盤面のマスとボタンのクリック）で行う。
func bombGame(t *testing.T, dest tactics.Pos) (*Game, *Controller) {
	t.Helper()
	c := newIdleWith(t, bombSetup)
	g := &Game{ctrl: c}
	g.leftClick(cellCenter(tactics.Pos{X: 2, Y: 2}))
	g.leftClick(cellCenter(dest))
	if c.State() != Moved {
		t.Fatalf("移動先を選んだあとの状態 = %v", c.State())
	}
	clickMenuLabel(t, g, "アイテム")
	if c.State() != ItemMenu {
		t.Fatalf("アイテムを選んだあとの状態 = %v", c.State())
	}
	clickMenuLabel(t, g, c.MenuLabels()[0])
	return g, c
}

// clickMenuLabel は、いまのボタンの一覧から文が label のボタンをクリックする。
func clickMenuLabel(t *testing.T, g *Game, label string) {
	t.Helper()
	c := g.ctrl
	labels := c.MenuLabels()
	for i, l := range labels {
		if l == label {
			_, dest, _ := c.Selection()
			r := c.Camera().MenuRects(dest, len(labels))[i]
			g.leftClick(r.Min.X+2, r.Min.Y+2)
			return
		}
	}
	t.Fatalf("ボタン %q がない: %v", label, labels)
}

func TestNG01BombUI(t *testing.T) {
	t.Run("TC-DES-NG-05-01 HP が最大でも「アイテム」が出て、一覧に爆弾の効果が出る。爆弾を選ぶとすぐには使わず爆心を選ぶ状態になる", func(t *testing.T) {
		c := newIdleWith(t, bombSetup)
		g := &Game{ctrl: c}
		g.leftClick(cellCenter(tactics.Pos{X: 2, Y: 2}))
		g.leftClick(cellCenter(tactics.Pos{X: 2, Y: 2}))
		clickMenuLabel(t, g, "アイテム")
		if got, want := c.MenuLabels(), []string{"爆弾 ×1（爆弾 ダメージ 8　範囲 1　投げる 3）"}; !reflect.DeepEqual(got, want) {
			t.Errorf("一覧 = %v, 期待 %v", got, want)
		}
		clickMenuLabel(t, g, c.MenuLabels()[0])
		if c.State() != BombTargeting {
			t.Fatalf("状態 = %v, 期待 BombTargeting", c.State())
		}
		if n := c.Battle().Units()[0].Items[0].Count; n != 1 {
			t.Errorf("爆心を選ぶ前に在庫が減った: %d", n)
		}
	})

	t.Run("TC-DES-NG-05-02 投げられる範囲は移動先から 3 マス以内（マップの端を切る）で、赤く示す", func(t *testing.T) {
		_, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		blue, red := c.Overlay()
		if len(blue) != 0 {
			t.Errorf("青 = %v, 期待 空", blue)
		}
		// (2, 2) から距離 3 以内: y = 0 は 3 マス、y = 1 は 5、y = 2 は 6、y = 3 は 5、y = 4 は 3、y = 5 は 1（x は 0 以上）。
		if len(red) != 23 {
			t.Errorf("投げられるマス = %d, 期待 23", len(red))
		}
		for _, p := range []tactics.Pos{{X: 5, Y: 2}, {X: 2, Y: 5}, {X: 0, Y: 1}} {
			if !red[p] {
				t.Errorf("%v は距離 3 以内なので範囲に入る", p)
			}
		}
		for _, p := range []tactics.Pos{{X: 6, Y: 2}, {X: 5, Y: 3}, {X: 2, Y: 6}} {
			if red[p] {
				t.Errorf("%v は距離 4 以上なので範囲に入らない", p)
			}
		}
		if c.RangeStyle() != StyleBad {
			t.Error("投げられる範囲は赤で描く")
		}
		if c.BlastOverlay() != nil {
			t.Error("爆心を選ぶ前は爆発の範囲を出さない")
		}
	})

	t.Run("TC-DES-NG-05-03 範囲の外のクリックは理由を知らせ、爆心を選ばない", func(t *testing.T) {
		g, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		g.leftClick(cellCenter(tactics.Pos{X: 6, Y: 2}))
		if _, ok := c.BombForecast(); ok || c.State() != BombTargeting || !strings.Contains(c.Message(), "投げられる範囲") {
			t.Errorf("状態 %v、メッセージ %q", c.State(), c.Message())
		}
	})

	t.Run("TC-DES-NG-05-04 爆心を選ぶと爆発の範囲を強調し、敵・味方ごとの予想ダメージと撃破を出す", func(t *testing.T) {
		g, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		g.leftClick(cellCenter(tactics.Pos{X: 4, Y: 2}))
		want := posSet(tactics.Pos{X: 4, Y: 2}, tactics.Pos{X: 3, Y: 2}, tactics.Pos{X: 5, Y: 2}, tactics.Pos{X: 4, Y: 1}, tactics.Pos{X: 4, Y: 3})
		if got := c.BlastOverlay(); !reflect.DeepEqual(got, want) {
			t.Errorf("爆発の範囲 = %v, 期待 %v", got, want)
		}
		labels := c.BombLabels()
		type row struct {
			Unit  int
			Pos   tactics.Pos
			Text  string
			Style InfoStyle
			Ally  bool
		}
		var got []row
		for _, l := range labels {
			got = append(got, row{l.Unit, l.Pos, l.Text, l.Style, l.Ally})
		}
		wantRows := []row{
			{1, tactics.Pos{X: 4, Y: 2}, "味方 -8 HP 20→12", StyleBad, true},
			{3, tactics.Pos{X: 5, Y: 2}, "敵 -8 撃破", StyleGood, false},
		}
		if !reflect.DeepEqual(got, wantRows) {
			t.Errorf("予想 = %+v\n期待 %+v", got, wantRows)
		}
		guide := strings.Join(c.Guide(), "\n")
		if !strings.Contains(guide, "敵 1 体（撃破 1）") || !strings.Contains(guide, "味方 1 体を巻き込む（倒れる 0）") || !strings.Contains(guide, "もう一度クリックで投げる") {
			t.Errorf("案内 = %q", guide)
		}
		if n := c.Battle().Units()[0].Items[0].Count; n != 1 {
			t.Errorf("選んだだけで在庫が減った: %d", n)
		}
	})

	t.Run("TC-DES-NG-05-05 使う人自身も範囲に入れば「自分」として予想に出る（移動先の位置で数える）", func(t *testing.T) {
		g, c := bombGame(t, tactics.Pos{X: 3, Y: 2})
		g.leftClick(cellCenter(tactics.Pos{X: 3, Y: 2}))
		var texts []string
		for _, l := range c.BombLabels() {
			texts = append(texts, l.Text)
		}
		// 移動先 (3, 2) が爆心。範囲 (2, 2)〜(4, 2) などに、使う人（(3, 2)）と味方（(4, 2)）が入る。
		if want := []string{"自分 -8 HP 20→12", "味方 -8 HP 20→12"}; !reflect.DeepEqual(texts, want) {
			t.Errorf("予想 = %v, 期待 %v", texts, want)
		}
		if l := c.BombLabels()[0]; l.Pos != (tactics.Pos{X: 3, Y: 2}) || !l.Ally {
			t.Errorf("自分の予想の位置 %v、Ally %v", l.Pos, l.Ally)
		}
		if guide := strings.Join(c.Guide(), "\n"); !strings.Contains(guide, "自分を含む味方 2 体を巻き込む") {
			t.Errorf("案内 = %q", guide)
		}
	})

	t.Run("TC-DES-NG-05-06 決定の前に取り消せる: 爆心 → 選び直し → 一覧 → 行動の一覧。何も使わない", func(t *testing.T) {
		g, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		g.leftClick(cellCenter(tactics.Pos{X: 4, Y: 2}))
		c.Cancel()
		if _, ok := c.BombForecast(); ok || c.State() != BombTargeting {
			t.Fatalf("1 回目の取消し: 状態 %v", c.State())
		}
		c.Cancel()
		if c.State() != ItemMenu {
			t.Fatalf("2 回目の取消し: 状態 %v", c.State())
		}
		c.Cancel()
		if c.State() != Moved {
			t.Fatalf("3 回目の取消し: 状態 %v", c.State())
		}
		u := c.Battle().Units()
		if u[0].Items[0].Count != 1 || u[0].Acted || u[3].HP != 8 {
			t.Errorf("取り消したのに状態が変わった: %+v", u[0])
		}
		if len(c.Battle().History()) != 0 {
			t.Errorf("履歴 %d 件", len(c.Battle().History()))
		}
	})

	t.Run("TC-DES-NG-05-07 同じマスをもう一度クリックすると決定し、爆発を再生して結果が盤面に出る", func(t *testing.T) {
		g, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		g.leftClick(cellCenter(tactics.Pos{X: 4, Y: 2}))
		// 別のマスをクリックすると、決定せずに爆心を選び直す。
		g.leftClick(cellCenter(tactics.Pos{X: 3, Y: 3}))
		if c.State() != BombTargeting || c.Battle().Units()[0].Acted {
			t.Fatalf("別のマスで決定してしまった: 状態 %v", c.State())
		}
		g.leftClick(cellCenter(tactics.Pos{X: 4, Y: 2}))
		g.leftClick(cellCenter(tactics.Pos{X: 4, Y: 2}))
		if c.State() != Playing {
			t.Fatalf("決定後の状態 = %v, 期待 Playing", c.State())
		}
		var texts []string
		for c.State() == Playing {
			if e, frame, ok := c.Event(); ok && frame == 0 && (e.Kind == tactics.EventBlast || e.Kind == tactics.EventBlastHit || e.Kind == tactics.EventFall) {
				texts = append(texts, EventText(e))
			}
			c.Tick()
		}
		want := []string{"レオン が 爆弾 を投げた", "ガルド は爆発で 8 ダメージ", "ファイター は爆発で 8 ダメージ", "ファイター は倒れた"}
		if !reflect.DeepEqual(texts, want) {
			t.Errorf("出来事の文 = %v, 期待 %v", texts, want)
		}
		u := c.Battle().Units()
		if u[1].HP != 12 || u[3].Alive || u[0].Items != nil || !u[0].Acted {
			t.Errorf("結果: 味方 HP %d、敵 Alive %v、手持ち %v、行動済み %v", u[1].HP, u[3].Alive, u[0].Items, u[0].Acted)
		}
	})

	t.Run("TC-DES-NG-05-08 爆心を選んでいる間も、移動先に置いた主人公を描き、範囲の外の取消しは盤面を変えない", func(t *testing.T) {
		_, c := bombGame(t, tactics.Pos{X: 3, Y: 3})
		if got := c.Units()[0].Pos; got != (tactics.Pos{X: 3, Y: 3}) {
			t.Errorf("盤面の主人公の位置 = %v, 期待 (3, 3)", got)
		}
		if got := c.Battle().Units()[0].Pos; got != (tactics.Pos{X: 2, Y: 2}) {
			t.Errorf("core の主人公の位置が変わった: %v", got)
		}
		if _, red := c.Overlay(); !red[tactics.Pos{X: 6, Y: 3}] || red[tactics.Pos{X: 7, Y: 3}] {
			t.Error("投げられる範囲が移動先 (3, 3) を中心にしていない")
		}
	})

	t.Run("TC-DES-NG-05-09 回復薬だけで HP が最大なら、今までどおり「アイテム」は出ない", func(t *testing.T) {
		c := newIdle(t)
		g := &Game{ctrl: c}
		g.leftClick(cellCenter(tactics.Pos{X: 0, Y: 0}))
		g.leftClick(cellCenter(tactics.Pos{X: 0, Y: 0}))
		for _, a := range c.Actions() {
			if a == tactics.UseItem {
				t.Error("HP が最大の回復薬だけで「アイテム」が出ている")
			}
		}
	})

	t.Run("TC-DES-NG-05-10 爆弾の予想は core の BlastForecast と一致する（範囲と数は core が決める）", func(t *testing.T) {
		_, c := bombGame(t, tactics.Pos{X: 2, Y: 2})
		center := tactics.Pos{X: 4, Y: 2}
		g := &Game{ctrl: c}
		g.leftClick(cellCenter(center))
		f, ok := c.BombForecast()
		if !ok {
			t.Fatal("予想がない")
		}
		want, err := c.Battle().BlastForecast(tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 2}, Action: tactics.UseItem, Item: 0, Blast: &center})
		if err != nil || !reflect.DeepEqual(f, want) {
			t.Errorf("予想 %+v\n期待 %+v（%v）", f, want, err)
		}
	})
}

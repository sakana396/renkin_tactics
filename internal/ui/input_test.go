package ui

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

var (
	plain     = tactics.Terrain{Name: "平地", Cost: 1}
	testSword = tactics.Weapon{Name: "剣", Type: tactics.Sword, Might: 0, Hit: 300, MinRange: 1, MaxRange: 1}
	testAxe   = tactics.Weapon{Name: "斧", Type: tactics.Axe, Might: 0, Hit: 300, MinRange: 1, MaxRange: 1}
	testLance = tactics.Weapon{Name: "槍", Type: tactics.Lance, Might: 0, Hit: 300, MinRange: 1, MaxRange: 1}
	potion    = tactics.Item{Name: "回復薬", Heal: 10}
)

// field は w × h の平地を作る。
func field(w, h int) [][]tactics.Terrain {
	rows := make([][]tactics.Terrain, h)
	for y := range rows {
		rows[y] = make([]tactics.Terrain, w)
		for x := range rows[y] {
			rows[y][x] = plain
		}
	}
	return rows
}

// basicSetup は 6 × 6 の平地の戦闘を作る。
//
//	0: 主人公 (0, 0)。移動 3、HP 20、力 0、守備 0。剣（威力 0）なので、相性でファイターには 1、ボスには 0 ダメージ。回復薬 × 1
//	1: 味方 (0, 2)。移動 3、守備 5。手持ちなし
//	2: ボス (5, 5)。移動 0、待ち構える
//	3: 敵 (1, 0)。移動 0、待ち構える。HP 30。命中は必ず当たり、主人公に 2 ダメージ
//
// 敵軍の番には、敵 3 が主人公を攻撃して HP を 18 にする（主人公の反撃は 1 ダメージで、敵 3 は倒れない）。乱数の結果によらない。
func basicSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: field(6, 6),
		Units: []tactics.UnitSetup{
			{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testSword, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Lord: true},
			{Name: "ガルド", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Def: 5, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 2}, Weapon: testLance},
			{Name: "ボス", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 5, Y: 5}, Weapon: testLance, Boss: true, AI: tactics.Hold},
			// 斧は剣に不利なので、攻撃力は 3 − 1 = 2。
			{Name: "ファイター", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30, Str: 3}, Pos: tactics.Pos{X: 1, Y: 0}, Weapon: testAxe, AI: tactics.Hold},
		},
	}
}

// factoryOf は、setup から戦闘を作る Factory と、呼ばれた回数を返す。
func factoryOf(t *testing.T, setup func() tactics.Setup) (Factory, *int) {
	t.Helper()
	calls := 0
	return func() (*tactics.Battle, []tactics.Event, error) {
		calls++
		return tactics.New(setup(), 1)
	}, &calls
}

// newIdle は basicSetup の戦闘で Controller を作り、初めの出来事の再生を終えた Idle にする。
func newIdle(t *testing.T) *Controller {
	t.Helper()
	return newIdleWith(t, basicSetup)
}

// newIdleWith は setup の戦闘で Controller を作り、初めの出来事の再生を終えた Idle にする。
func newIdleWith(t *testing.T, setup func() tactics.Setup) *Controller {
	t.Helper()
	f, _ := factoryOf(t, setup)
	c, err := NewController(f)
	if err != nil {
		t.Fatalf("NewController: %v", err)
	}
	playOut(c)
	if c.State() != Idle {
		t.Fatalf("状態 = %v、Idle になるはず", c.State())
	}
	return c
}

// playOut は再生を最後まで Tick で進め、進めたフレーム数を返す。
func playOut(c *Controller) int {
	n := 0
	for c.State() == Playing && n < 10000 {
		c.Tick()
		n++
	}
	return n
}

// damaged は、敵軍の番を 1 回進めて主人公の HP を 18 にした Idle の Controller を返す。
func damaged(t *testing.T) *Controller {
	t.Helper()
	c := newIdle(t)
	c.EndPhase()
	c.EndPhase()
	playOut(c)
	if c.State() != Idle || c.Battle().Units()[0].HP != 18 {
		t.Fatalf("状態 %v、主人公の HP %d", c.State(), c.Battle().Units()[0].HP)
	}
	return c
}

func posSet(ps ...tactics.Pos) map[tactics.Pos]bool {
	m := map[tactics.Pos]bool{}
	for _, p := range ps {
		m[p] = true
	}
	return m
}

func keys(m map[tactics.Pos]int) map[tactics.Pos]bool {
	out := map[tactics.Pos]bool{}
	for p := range m {
		out[p] = true
	}
	return out
}

// 入力の状態の遷移（DES-UI-02）を、描画から分けた Controller で確かめる。
func TestControllerTransitions(t *testing.T) {
	t.Run("TC-IF-01-01 起動すると初めの出来事を再生し、20 フレームで Idle になる", func(t *testing.T) {
		f, calls := factoryOf(t, basicSetup)
		c, err := NewController(f)
		if err != nil {
			t.Fatal(err)
		}
		if *calls != 1 || c.State() != Playing {
			t.Fatalf("呼んだ回数 %d、状態 %v", *calls, c.State())
		}
		if e, _, ok := c.Event(); !ok || e.Kind != tactics.EventPhaseStart || e.Side != tactics.Player {
			t.Fatalf("再生中の出来事 = %+v, %v", e, ok)
		}
		if n := playOut(c); n != FramesPerEvent {
			t.Errorf("再生のフレーム数 = %d、%d のはず", n, FramesPerEvent)
		}
		if c.State() != Idle {
			t.Errorf("状態 = %v", c.State())
		}
		if turn, side := c.Phase(); turn != 1 || side != tactics.Player {
			t.Errorf("表示のターンと番 = %d, %v", turn, side)
		}
	})

	t.Run("TC-IF-01-02 Idle で行動していない自軍をクリックすると Selected になり、移動できるマスを青、攻撃できるマスを赤で返す", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		if c.State() != Selected {
			t.Fatalf("状態 = %v", c.State())
		}
		if u, _, _ := c.Selection(); u != 1 {
			t.Errorf("選んだユニット = %d", u)
		}
		blue, red := c.Overlay()
		if want := keys(c.Battle().Reachable(1)); !reflect.DeepEqual(blue, want) {
			t.Errorf("青 = %v\nwant %v", blue, want)
		}
		// 移動 3 で届くマスの隣のうち、移動できないマス。敵 3 のいる (1, 0) と、主人公のいる (0, 0) は赤になる。
		if !red[tactics.Pos{X: 1, Y: 0}] || !red[tactics.Pos{X: 0, Y: 0}] {
			t.Errorf("赤に (1, 0)・(0, 0) がない: %v", red)
		}
		for p := range red {
			if blue[p] {
				t.Errorf("%v が青と赤の両方にある", p)
			}
		}
		if !red[tactics.Pos{X: 4, Y: 2}] || red[tactics.Pos{X: 5, Y: 2}] {
			t.Errorf("赤は移動できるマスの隣だけ: (4, 2) %v、(5, 2) %v", red[tactics.Pos{X: 4, Y: 2}], red[tactics.Pos{X: 5, Y: 2}])
		}
	})

	t.Run("TC-IF-01-03 Idle で空きマス・行動済みの自軍をクリックしても Idle のまま。敵をクリックすると、カーソルを離してもその敵の範囲を出し続け、もう一度で消す", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 3, Y: 3})
		if c.State() != Idle {
			t.Fatalf("空きマス: 状態 = %v", c.State())
		}
		if blue, red := c.Overlay(); blue != nil || red != nil {
			t.Errorf("空きマス: 範囲 = %v, %v", blue, red)
		}
		c.Click(tactics.Pos{X: 1, Y: 0})
		// クリックのあと、カーソルを空きマスに離しても出し続ける（DES-UI-08）。
		c.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		blue, red := c.Overlay()
		if c.State() != Idle || !reflect.DeepEqual(blue, posSet(tactics.Pos{X: 1, Y: 0})) {
			t.Errorf("敵: 状態 %v、青 %v（移動 0 なので今いるマスだけ）", c.State(), blue)
		}
		if !reflect.DeepEqual(red, posSet(tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 2, Y: 0}, tactics.Pos{X: 1, Y: 1})) {
			t.Errorf("敵: 赤 = %v", red)
		}
		c.Click(tactics.Pos{X: 1, Y: 0})
		if blue, red := c.Overlay(); blue != nil || red != nil {
			t.Errorf("もう一度クリックしても範囲が消えない: %v, %v", blue, red)
		}
		// 自軍 1 を待機させて行動済みにし、もう一度クリックする。
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.ChooseAction(tactics.Wait)
		playOut(c)
		c.Click(tactics.Pos{X: 0, Y: 2})
		if c.State() != Idle {
			t.Errorf("行動済みの自軍: 状態 = %v", c.State())
		}
		if blue, red := c.Overlay(); blue != nil || red != nil {
			t.Errorf("行動済みの自軍: 範囲 = %v, %v", blue, red)
		}
	})

	t.Run("TC-IF-01-04 Selected で移動できるマスをクリックすると Moved になる。Act はまだ呼ばず、画面の上だけで移動先に描く", func(t *testing.T) {
		c := newIdle(t)
		before := c.Battle().Units()
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 2, Y: 3})
		if c.State() != Moved {
			t.Fatalf("状態 = %v", c.State())
		}
		if _, dest, _ := c.Selection(); dest != (tactics.Pos{X: 2, Y: 3}) {
			t.Errorf("移動先 = %v", dest)
		}
		if after := c.Battle().Units(); !reflect.DeepEqual(after, before) {
			t.Errorf("戦闘の状態が変わった（Act を呼んだ）")
		}
		if got := c.Units()[1].Pos; got != (tactics.Pos{X: 2, Y: 3}) {
			t.Errorf("描くユニットの位置 = %v", got)
		}
		_, red := c.Overlay()
		if !reflect.DeepEqual(red, posSet(tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 1, Y: 3}, tactics.Pos{X: 3, Y: 3}, tactics.Pos{X: 2, Y: 4})) {
			t.Errorf("赤（移動先から届くマス） = %v", red)
		}
	})

	t.Run("TC-IF-01-05 Selected で移動できないマス・敵のマスをクリックしても Selected のまま", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		for _, p := range []tactics.Pos{{X: 5, Y: 5}, {X: 1, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 0}} {
			c.Click(p)
			if c.State() != Selected {
				t.Errorf("%v: 状態 = %v", p, c.State())
			}
		}
	})

	t.Run("TC-IF-01-06 行動の一覧には、できる行動だけを出す", func(t *testing.T) {
		c := newIdle(t)
		// 主人公: 敵が隣にいて HP が最大 → 攻撃・待機。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"攻撃", "待機"}) {
			t.Errorf("主人公（HP 最大） = %v", got)
		}
		c.Cancel()
		c.Cancel()
		// 味方: 射程に敵がいない、手持ちもない → 待機だけ。
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 0, Y: 3})
		if got := c.Actions(); !reflect.DeepEqual(got, []tactics.Action{tactics.Wait}) {
			t.Errorf("味方 = %v", got)
		}
		// 一覧にない行動は選べない。
		c.ChooseAction(tactics.Attack)
		c.ChooseAction(tactics.UseItem)
		if c.State() != Moved {
			t.Errorf("一覧にない行動で状態が変わった: %v", c.State())
		}
		// 主人公: HP が最大でない → 攻撃・アイテム・待機。
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 0})
		if got := d.Actions(); !reflect.DeepEqual(got, []tactics.Action{tactics.Attack, tactics.UseItem, tactics.Wait}) {
			t.Errorf("主人公（HP 18） = %v", got)
		}
		// 移動して敵から離れると攻撃は出ない。
		d.Cancel()
		d.Click(tactics.Pos{X: 0, Y: 1})
		if got := d.MenuLabels(); !reflect.DeepEqual(got, []string{"アイテム", "待機"}) {
			t.Errorf("主人公（離れたマス） = %v", got)
		}

		// 射程の中に倒れた敵しかいなければ「攻撃」を出さない。敵 3 の HP を 1 にして、主人公が倒す。
		weak := func() tactics.Setup {
			s := basicSetup()
			s.Units[3].Stats.HP = 1
			return s
		}
		w := newIdleWith(t, weak)
		w.Click(tactics.Pos{X: 0, Y: 0})
		w.Click(tactics.Pos{X: 0, Y: 0})
		w.ChooseAction(tactics.Attack)
		w.Click(tactics.Pos{X: 1, Y: 0})
		w.Click(tactics.Pos{X: 1, Y: 0})
		playOut(w)
		if w.State() != Idle || w.Battle().Units()[3].Alive {
			t.Fatalf("敵 3 を倒せていない: 状態 %v", w.State())
		}
		w.Click(tactics.Pos{X: 0, Y: 2})
		w.Click(tactics.Pos{X: 1, Y: 1}) // 倒れた敵のいたマス (1, 0) の隣。
		if got := w.Actions(); !reflect.DeepEqual(got, []tactics.Action{tactics.Wait}) || len(w.Targets()) != 0 {
			t.Errorf("倒れた敵の隣 = %v、相手 %v", got, w.Targets())
		}

		// HP が減っていて手持ちがない味方は「待機」だけ。味方 (3, 3) が敵 (4, 3) に 4 ダメージを受ける。
		hurt := func() tactics.Setup {
			s := basicSetup()
			s.Units[1].Pos, s.Units[1].Stats.Def = tactics.Pos{X: 3, Y: 3}, 0
			s.Units[3].Pos = tactics.Pos{X: 4, Y: 3}
			return s
		}
		h := newIdleWith(t, hurt)
		h.EndPhase()
		h.EndPhase()
		playOut(h)
		if u := h.Battle().Units()[1]; h.State() != Idle || u.HP >= u.Stats.HP || len(u.Items) != 0 {
			t.Fatalf("味方の HP が減っていない: 状態 %v、%+v", h.State(), u)
		}
		h.Click(tactics.Pos{X: 3, Y: 3})
		h.Click(tactics.Pos{X: 3, Y: 1})
		if got := h.MenuLabels(); !reflect.DeepEqual(got, []string{"待機"}) {
			t.Errorf("HP が減って手持ちのない味方 = %v", got)
		}
	})

	t.Run("TC-IF-01-07 待機を選ぶと Act を 1 回だけ呼び、出来事を再生してから Idle に戻る", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.ClickMenu(0)
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		u := c.Battle().Units()[1]
		if !u.Acted || u.Pos != (tactics.Pos{X: 2, Y: 3}) {
			t.Errorf("ユニット = %+v", u)
		}
		// 再生中に選び直しても Act は呼ばれない。
		c.ChooseAction(tactics.Wait)
		if n := playOut(c); n != 2*FramesPerEvent {
			t.Errorf("再生のフレーム数 = %d（Move と Wait の 2 つ）", n)
		}
		if c.State() != Idle {
			t.Errorf("状態 = %v", c.State())
		}
	})

	t.Run("TC-IF-01-08 取り消すと Targeting → Moved → Selected → Idle の順に戻り、戦闘の状態は変わらない", func(t *testing.T) {
		c := newIdle(t)
		before := c.Battle().Units()
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 0})
		want := []State{Moved, Selected, Idle, Idle}
		for i, w := range want {
			c.Cancel()
			if c.State() != w {
				t.Errorf("%d 回目の取り消し: 状態 = %v、%v のはず", i+1, c.State(), w)
			}
		}
		if _, _, target := c.Selection(); target != -1 {
			t.Errorf("取り消したあとも相手が残っている: %d", target)
		}
		if !reflect.DeepEqual(c.Battle().Units(), before) {
			t.Errorf("戦闘の状態が変わった")
		}
		// ItemMenu からも Moved に戻る。Esc も同じ。
		d := damaged(t)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		d.ChooseAction(tactics.UseItem)
		if d.State() != ItemMenu {
			t.Fatalf("状態 = %v", d.State())
		}
		d.Escape()
		if d.State() != Moved {
			t.Errorf("Esc: 状態 = %v", d.State())
		}
	})

	t.Run("TC-IF-01-09 攻撃: 相手を 1 回クリックすると選び、同じ相手をもう一度クリックすると決定する", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		if c.State() != Targeting || !reflect.DeepEqual(c.Targets(), []int{3}) {
			t.Fatalf("状態 %v、相手 %v", c.State(), c.Targets())
		}
		// 相手でないマスは無視する。
		c.Click(tactics.Pos{X: 5, Y: 5})
		c.Click(tactics.Pos{X: 2, Y: 2})
		if _, _, target := c.Selection(); c.State() != Targeting || target != -1 {
			t.Errorf("相手でないマス: 状態 %v、相手 %d", c.State(), target)
		}
		c.Click(tactics.Pos{X: 1, Y: 0})
		if _, _, target := c.Selection(); c.State() != Targeting || target != 3 || c.Battle().Units()[0].Acted {
			t.Errorf("1 回目: 状態 %v、相手 %d", c.State(), target)
		}
		c.Click(tactics.Pos{X: 1, Y: 0})
		if c.State() != Playing || !c.Battle().Units()[0].Acted {
			t.Fatalf("2 回目: 状態 %v", c.State())
		}
		if e, _, _ := c.Event(); e.Kind != tactics.EventStrike || e.Unit != 0 || e.Target != 3 {
			t.Errorf("初めの出来事 = %+v（移動しないので Strike から）", e)
		}

		// 相手が 2 体いるとき、別の相手をクリックすると選び直すだけで Act を呼ばない。
		two := func() tactics.Setup {
			s := basicSetup()
			s.Units = append(s.Units, tactics.UnitSetup{Name: "ソルジャー", Side: tactics.Enemy, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 0, Y: 1}, Weapon: testLance, AI: tactics.Hold})
			return s
		}
		d := newIdleWith(t, two)
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 0})
		d.ChooseAction(tactics.Attack)
		if !reflect.DeepEqual(d.Targets(), []int{3, 4}) {
			t.Fatalf("相手 = %v", d.Targets())
		}
		d.Click(tactics.Pos{X: 1, Y: 0})
		d.Click(tactics.Pos{X: 0, Y: 1})
		if _, _, target := d.Selection(); d.State() != Targeting || target != 4 || d.Battle().Units()[0].Acted {
			t.Errorf("選び直し: 状態 %v、相手 %d、行動済み %v", d.State(), target, d.Battle().Units()[0].Acted)
		}
		d.Click(tactics.Pos{X: 0, Y: 1})
		if e, _, _ := d.Event(); d.State() != Playing || e.Kind != tactics.EventStrike || e.Target != 4 {
			t.Errorf("選び直した相手で決定: 状態 %v、出来事 %+v", d.State(), e)
		}
	})

	t.Run("TC-IF-01-10 アイテム: 一覧から選ぶと UseItem で Act を呼び、回復を再生する", func(t *testing.T) {
		c := damaged(t)
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.UseItem)
		if got := c.MenuLabels(); !reflect.DeepEqual(got, []string{"回復薬 ×1（回復 10）"}) {
			t.Errorf("アイテムの一覧 = %v", got)
		}
		c.ChooseItem(1) // ない番目は無視する。
		if c.State() != ItemMenu {
			t.Fatalf("ない番目で状態が変わった: %v", c.State())
		}
		c.ClickMenu(0)
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		e, _, _ := c.Event()
		if e.Kind != tactics.EventHeal || e.Amount != 2 || e.Item != "回復薬" {
			t.Errorf("出来事 = %+v", e)
		}
		if u := c.Battle().Units()[0]; u.HP != 20 || len(u.Items) != 0 || !u.Acted {
			t.Errorf("主人公 = %+v", u)
		}
	})

	t.Run("TC-IF-01-11 E: 行動していない自軍がいれば 1 回目は確認を求め、2 回目で番を終える。Idle 以外では受け付けない", func(t *testing.T) {
		c := newIdle(t)
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.EndPhase()
		if c.State() != Selected || c.Battle().Phase != tactics.Player {
			t.Fatalf("Selected で E を受け付けた: %v", c.State())
		}
		c.Cancel()
		c.EndPhase()
		if c.State() != Idle || !strings.Contains(c.Message(), "もう一度 E") || c.Battle().Phase != tactics.Player {
			t.Fatalf("1 回目: 状態 %v、知らせ %q", c.State(), c.Message())
		}
		// ほかの操作をはさむと、確認はやり直しになる。
		c.Click(tactics.Pos{X: 3, Y: 3})
		c.EndPhase()
		if c.State() != Idle {
			t.Fatalf("操作をはさんだ 1 回目: 状態 %v", c.State())
		}
		c.EndPhase()
		if c.State() != Playing {
			t.Fatalf("2 回目: 状態 %v", c.State())
		}
		if e, _, _ := c.Event(); e.Kind != tactics.EventPhaseStart || e.Side != tactics.Enemy {
			t.Errorf("初めの出来事 = %+v", e)
		}
		if turn, side := c.Phase(); turn != 1 || side != tactics.Enemy {
			t.Errorf("表示のターンと番 = %d, %v", turn, side)
		}
		playOut(c)
		if turn, side := c.Phase(); c.State() != Idle || turn != 2 || side != tactics.Player {
			t.Errorf("再生のあと: 状態 %v、ターン %d、番 %v", c.State(), turn, side)
		}
	})

	t.Run("TC-IF-01-12 再生中は入力を受け付けず、出来事を 20 フレームずつ進める。早送りは押している間だけ 4 フレームずつ進め、出来事を飛ばさない", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		n := len(c.events)
		if n < 3 {
			t.Fatalf("出来事の数 = %d", n)
		}
		before := c.Battle().Units()
		c.Click(tactics.Pos{X: 0, Y: 2})
		c.Cancel()
		c.Escape()
		c.EndPhase()
		c.ChooseAction(tactics.Wait)
		c.Restart()
		if c.State() != Playing || !reflect.DeepEqual(c.Battle().Units(), before) {
			t.Fatalf("再生中に入力を受け付けた: 状態 %v", c.State())
		}
		for i := 0; i < FramesPerEvent-1; i++ {
			c.Tick()
		}
		if _, f, _ := c.Event(); c.index != 0 || f != FramesPerEvent-1 {
			t.Errorf("19 フレーム目: 出来事 %d、フレーム %d", c.index, f)
		}
		c.Tick()
		if c.index != 1 {
			t.Errorf("20 フレーム目: 出来事 %d", c.index)
		}
		// 押している間は 4 フレームで次の出来事に進む。
		c.SetFastForward(true)
		for i := 0; i < FastFramesPerEvent-1; i++ {
			c.Tick()
		}
		if c.index != 1 {
			t.Errorf("早送りの 3 フレーム目: 出来事 %d", c.index)
		}
		c.Tick()
		if c.index != 2 {
			t.Errorf("早送りの 4 フレーム目: 出来事 %d", c.index)
		}
		// 押したまま最後まで進める。出来事を 1 つずつ、すべて表示する。
		seen := []int{c.index}
		ticks := 0
		for c.State() == Playing && ticks < 1000 {
			c.Tick()
			ticks++
			if c.State() == Playing && c.index != seen[len(seen)-1] {
				seen = append(seen, c.index)
				// 早送りの間も、表示した PhaseStart に番の表示を合わせる。
				if e, _, _ := c.Event(); e.Kind == tactics.EventPhaseStart {
					if turn, side := c.Phase(); turn != e.Turn || side != e.Side {
						t.Errorf("早送り中の PhaseStart（%d, %v）で表示が %d, %v", e.Turn, e.Side, turn, side)
					}
				}
			}
		}
		for i, idx := range seen {
			if idx != 2+i {
				t.Errorf("早送りで飛ばした出来事がある: %v（出来事は %d 個）", seen, n)
				break
			}
		}
		if len(seen) != n-2 || ticks != FastFramesPerEvent*(n-2) {
			t.Errorf("早送りで表示した出来事 %v、フレーム数 %d（出来事は %d 個）", seen, ticks, n)
		}
		if turn, side := c.Phase(); c.State() != Idle || turn != 2 || side != tactics.Player {
			t.Errorf("早送りのあと: 状態 %v、ターン %d、番 %v", c.State(), turn, side)
		}
		// 再生中でなければ、押していても何もしない。
		c.Tick()
		if c.State() != Idle {
			t.Errorf("Idle で早送り: 状態 %v", c.State())
		}

		// 離すと 20 フレームに戻る。
		d := newIdle(t)
		d.EndPhase()
		d.EndPhase()
		d.SetFastForward(true)
		for i := 0; i < FastFramesPerEvent; i++ {
			d.Tick()
		}
		d.SetFastForward(false)
		for i := 0; i < FramesPerEvent-1; i++ {
			d.Tick()
		}
		if d.index != 1 {
			t.Errorf("離したあとの 19 フレーム目: 出来事 %d", d.index)
		}
		d.Tick()
		if d.index != 2 {
			t.Errorf("離したあとの 20 フレーム目: 出来事 %d", d.index)
		}
	})

	t.Run("TC-IF-01-13 勝ち負けが決まると Over になる。R で新しい戦闘を作ってやり直し、Esc で終了する", func(t *testing.T) {
		// 勝ち: 主人公がボス（HP 1）の隣にいる。
		won := func() tactics.Setup {
			return tactics.Setup{
				Terrain: field(3, 1),
				Units: []tactics.UnitSetup{
					{Name: "レオン", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Str: 5, Mov: 1}, Pos: tactics.Pos{X: 0, Y: 0}, Weapon: testSword, Lord: true},
					{Name: "ボス", Side: tactics.Enemy, Stats: tactics.Stats{HP: 1}, Pos: tactics.Pos{X: 1, Y: 0}, Weapon: testLance, Boss: true, AI: tactics.Hold},
				},
			}
		}
		f, calls := factoryOf(t, won)
		c, err := NewController(f)
		if err != nil {
			t.Fatal(err)
		}
		playOut(c)
		c.Restart() // Over でなければ何もしない。
		if *calls != 1 {
			t.Fatalf("Idle で R を受け付けた")
		}
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 0})
		c.Click(tactics.Pos{X: 1, Y: 0})
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		playOut(c)
		if c.State() != Over || c.Battle().Result != tactics.Won {
			t.Fatalf("状態 %v、結果 %v", c.State(), c.Battle().Result)
		}
		c.Cancel()
		c.EndPhase()
		c.Click(tactics.Pos{X: 0, Y: 0})
		if c.State() != Over || c.Quit() {
			t.Fatalf("Over でほかの入力を受け付けた: %v", c.State())
		}
		old := c.Battle()
		c.Restart()
		if *calls != 2 || c.Battle() == old || c.State() != Playing || c.Battle().Result != tactics.Ongoing {
			t.Fatalf("やり直し: 呼んだ回数 %d、状態 %v", *calls, c.State())
		}
		playOut(c)
		if c.State() != Idle {
			t.Errorf("やり直しのあと: 状態 %v", c.State())
		}

		// 負け: 主人公（HP 1）が敵の隣にいて、番を終える。敵の番の途中で決まる。
		lost := func() tactics.Setup {
			s := basicSetup()
			s.Units[0].Stats.HP = 1
			return s
		}
		f2, _ := factoryOf(t, lost)
		d, err := NewController(f2)
		if err != nil {
			t.Fatal(err)
		}
		playOut(d)
		d.EndPhase()
		d.EndPhase()
		playOut(d)
		if d.State() != Over || d.Battle().Result != tactics.Lost {
			t.Fatalf("負け: 状態 %v、結果 %v", d.State(), d.Battle().Result)
		}
		if got := d.Guide(); len(got) != 2 || got[1] != "負け……　R: やり直す　Esc: 終了" {
			t.Errorf("負けの案内 = %q", got)
		}
		d.Escape()
		if !d.Quit() {
			t.Errorf("Over で Esc を押しても終了しない")
		}
	})

	t.Run("TC-IF-01-14 やり直しで戦闘を作れなければ、知らせを出して Over のまま", func(t *testing.T) {
		n := 0
		f := func() (*tactics.Battle, []tactics.Event, error) {
			n++
			if n > 1 {
				return nil, nil, errors.New("作れない")
			}
			s := basicSetup()
			s.Units[0].Stats.HP = 1
			return tactics.New(s, 1)
		}
		c, err := NewController(f)
		if err != nil {
			t.Fatal(err)
		}
		playOut(c)
		c.EndPhase()
		c.EndPhase()
		playOut(c)
		c.Restart()
		if c.State() != Over || !strings.Contains(c.Message(), "作れない") {
			t.Errorf("状態 %v、知らせ %q", c.State(), c.Message())
		}
		if _, err := NewController(func() (*tactics.Battle, []tactics.Event, error) { return nil, nil, errors.New("x") }); err == nil {
			t.Errorf("初めの戦闘を作れないのに誤りを返さない")
		}
	})
}

// 画面の配置とボタンの判定（DES-UI-01、DES-UI-02）。
func TestLayout(t *testing.T) {
	t.Run("TC-IF-01-15 窓は 832 × 576 で、盤面のマスと行動の一覧のボタンを座標から引ける", func(t *testing.T) {
		if ScreenWidth != 832 || ScreenHeight != 576 || MapSize != 512 {
			t.Errorf("窓 %d × %d、盤面 %d", ScreenWidth, ScreenHeight, MapSize)
		}
		cells := []struct {
			x, y int
			want tactics.Pos
			ok   bool
		}{
			{0, 0, tactics.Pos{X: 0, Y: 0}, true},
			{31, 32, tactics.Pos{X: 0, Y: 1}, true},
			{511, 511, tactics.Pos{X: 15, Y: 15}, true},
			{512, 0, tactics.Pos{}, false},
			{0, 512, tactics.Pos{}, false},
			{-1, 3, tactics.Pos{}, false},
		}
		for _, tc := range cells {
			if got, ok := CellAt(tc.x, tc.y); got != tc.want || ok != tc.ok {
				t.Errorf("CellAt(%d, %d) = %v, %v", tc.x, tc.y, got, ok)
			}
		}
		// 左上のマスなら右に出す。右下のマスなら左に寄せ、下にはみ出さないよう上にずらす。
		r := MenuRects(tactics.Pos{X: 0, Y: 0}, 3)
		if r[0] != image.Rect(32, 0, 32+menuWidth, menuHeight) || r[2].Min.Y != 2*menuHeight {
			t.Errorf("左上 = %v", r)
		}
		r = MenuRects(tactics.Pos{X: 15, Y: 15}, 3)
		if r[0].Max.X != 15*CellSize || r[2].Max.Y != MapSize {
			t.Errorf("右下 = %v", r)
		}
		if got := HitMenu(r, r[1].Min.X+1, r[1].Min.Y+1); got != 1 {
			t.Errorf("HitMenu = %d", got)
		}
		if got := HitMenu(r, 0, 0); got != -1 {
			t.Errorf("外の HitMenu = %d", got)
		}
	})
}

// 素材の読み込みとタイルの選び方（DES-UI-05）。
func TestAssets(t *testing.T) {
	t.Run("TC-IF-01-16 画像を読み込め、使うタイルが画像の中にある。地形とユニットのタイルを選べる", func(t *testing.T) {
		sheets, err := assets.Load()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []assets.Sheet{assets.Town, assets.Dungeon} {
			if b := sheets[s].Bounds(); b.Dx() != 12*assets.TileSize || b.Dy() != 11*assets.TileSize {
				t.Errorf("画像 %v の大きさ = %v", s, b)
			}
		}
		used := []assets.Tile{assets.Grass, assets.Forest, assets.Mountain, assets.Fort, assets.Wall,
			assets.PlayerLord, assets.PlayerSword, assets.PlayerLance, assets.PlayerAxe, assets.PlayerBow,
			assets.EnemyBoss, assets.EnemySword, assets.EnemyLance, assets.EnemyAxe, assets.EnemyBow}
		for _, tile := range used {
			if !tile.Rect().In(sheets[tile.Sheet].Bounds()) {
				t.Errorf("タイル %v が画像の外", tile)
			}
		}
		terrains := []struct {
			t    tactics.Terrain
			want assets.Tile
			ok   bool
		}{
			{tactics.Terrain{Name: "平地", Cost: 1}, assets.Tile{}, false},
			{tactics.Terrain{Name: "森", Cost: 2}, assets.Forest, true},
			{tactics.Terrain{Name: "山", Cost: 3}, assets.Mountain, true},
			{tactics.Terrain{Name: "砦", Cost: 1, HealPercent: 20}, assets.Fort, true},
			{tactics.Terrain{Name: "壁"}, assets.Wall, true},
			{tactics.Terrain{Name: "沼", Cost: 0}, assets.Wall, true},
			{tactics.Terrain{Name: "村", Cost: 1, HealPercent: 10}, assets.Fort, true},
			{tactics.Terrain{Name: "橋", Cost: 1}, assets.Tile{}, false},
		}
		for _, tc := range terrains {
			if got, ok := terrainTile(tc.t); got != tc.want || ok != tc.ok {
				t.Errorf("terrainTile(%s) = %v, %v", tc.t.Name, got, ok)
			}
		}
		units := []struct {
			u    tactics.Unit
			want assets.Tile
		}{
			{tactics.Unit{Side: tactics.Player, Lord: true, Weapon: testSword}, assets.PlayerLord},
			{tactics.Unit{Side: tactics.Player, Weapon: testSword}, assets.PlayerSword},
			{tactics.Unit{Side: tactics.Player, Weapon: testLance}, assets.PlayerLance},
			{tactics.Unit{Side: tactics.Player, Weapon: testAxe}, assets.PlayerAxe},
			{tactics.Unit{Side: tactics.Player, Weapon: tactics.Weapon{Type: tactics.Bow}}, assets.PlayerBow},
			{tactics.Unit{Side: tactics.Enemy, Boss: true, Weapon: testLance}, assets.EnemyBoss},
			{tactics.Unit{Side: tactics.Enemy, Weapon: testSword}, assets.EnemySword},
			{tactics.Unit{Side: tactics.Enemy, Weapon: testLance}, assets.EnemyLance},
			{tactics.Unit{Side: tactics.Enemy, Weapon: testAxe}, assets.EnemyAxe},
			{tactics.Unit{Side: tactics.Enemy, Weapon: tactics.Weapon{Type: tactics.Bow}}, assets.EnemyBow},
		}
		for i, tc := range units {
			if got := unitTile(tc.u); got != tc.want {
				t.Errorf("%d: unitTile = %v、%v のはず", i, got, tc.want)
			}
		}
	})
}

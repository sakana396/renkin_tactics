package tactics_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 錬金の爆弾（DES-NG-02、DES-NG-03）のテスト。期待値は設計メモ（design-ng01.md）の規則から手で数えた。

// bomb は、固定ダメージ dmg・半径 radius・投げられる距離 throw の爆弾。
func bomb(dmg, radius, throw int) tactics.Item {
	return tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: dmg, BlastRadius: radius, ThrowRange: throw}
}

// withBomb は、ユニットの手持ちを爆弾 n 個だけにする。
func withBomb(u tactics.UnitSetup, it tactics.Item, n int) tactics.UnitSetup {
	u.Items = []tactics.ItemStack{{Item: it, Count: n}}
	return u
}

// blastSetup は 9 × 9 の平地。0 は主人公（(3, 6)、移動 2、爆弾を持つ）で、爆心 (3, 3) の周りに次のユニットを置く。
// 距離は爆心 (3, 3) からのマンハッタン距離。
//
//	1: 味方 (2, 3) 距離 1   2: 味方 (2, 2) 距離 2
//	3: 敵 A (3, 3) 距離 0   4: 敵 B (4, 3) 距離 1   5: 敵 C (5, 3) 距離 2   6: 敵 D (5, 4) 距離 3
//	7: ボス (8, 8)（遠い）
func blastSetup(it tactics.Item) tactics.Setup {
	rows := make([]string, 9)
	for i := range rows {
		rows[i] = "........."
	}
	l := withBomb(lord(3, 6, 2), it, 2)
	l.Level = 1
	return tactics.Setup{
		Terrain: grid(rows...),
		Units: []tactics.UnitSetup{
			l,
			ally("味方 1", 2, 3, 2),
			ally("味方 2", 2, 2, 2),
			foe("敵 A", 3, 3),
			foe("敵 B", 4, 3),
			foe("敵 C", 5, 3),
			foe("敵 D", 5, 4),
			boss(8, 8),
		},
	}
}

func throwCmd(unit int, to, center tactics.Pos) tactics.Command {
	return tactics.Command{Unit: unit, To: to, Action: tactics.UseItem, Item: 0, Blast: &center}
}

// hitUnits は EventBlastHit の Unit を出来事の順に取り出す。
func hitUnits(events []tactics.Event) []int {
	var out []int
	for _, e := range events {
		if e.Kind == tactics.EventBlastHit {
			out = append(out, e.Unit)
		}
	}
	return out
}

// TestBlastRange は、爆発の範囲（半径 0・1・2）と、敵・味方・本人を区別しない巻き込みを確かめる（DES-NG-02）。
func TestBlastRange(t *testing.T) {
	cases := []struct {
		name   string
		radius int
		want   []int
	}{
		{"TC-DES-NG-02-01 半径 0 は爆心の 1 マスだけ", 0, []int{3}},
		{"TC-DES-NG-02-02 半径 1 は距離 1 までの敵と味方を巻き込む", 1, []int{1, 3, 4}},
		{"TC-DES-NG-02-03 半径 2 は距離 2 までの敵と味方を巻き込み、距離 3 は入らない", 2, []int{1, 2, 3, 4, 5}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			b := mustNewStill(t, blastSetup(bomb(8, tt.radius, 3)))
			noRolls(t, b)
			events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
			assertEqual(t, "巻き込んだユニット（添字の順）", hitUnits(events), tt.want)
			hit := map[int]bool{}
			for _, i := range tt.want {
				hit[i] = true
			}
			for i, u := range b.Units() {
				wantHP := 20
				if hit[i] {
					wantHP = 12
				}
				if u.HP != wantHP {
					t.Errorf("ユニット %d（%s）の HP = %d, 期待 %d", i, u.Name, u.HP, wantHP)
				}
			}
		})
	}

	t.Run("TC-DES-NG-02-04 使った本人も範囲に入れば巻き込まれ、移動後の位置で数える", func(t *testing.T) {
		// 主人公は (3, 6) から (3, 4) へ移動して、(3, 3) に使う。移動前の位置 (3, 6) は距離 3 で範囲外、移動後は距離 1。
		b := mustNewStill(t, blastSetup(bomb(8, 1, 3)))
		events := mustAct(t, b, throwCmd(0, pos(3, 4), pos(3, 3)))
		assertEqual(t, "巻き込んだユニット", hitUnits(events), []int{0, 1, 3, 4})
		assertEqual(t, "種類の先頭", kinds(events)[:2], []tactics.EventKind{tactics.EventMove, tactics.EventBlast})
		if b.Units()[0].HP != 12 {
			t.Errorf("主人公の HP = %d, 期待 12", b.Units()[0].HP)
		}
		// 移動前の位置で数えると範囲外になるはずの使い方（移動しない）では、本人は巻き込まれない。
		b = mustNewStill(t, blastSetup(bomb(8, 1, 3)))
		events = mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		assertEqual(t, "移動しないとき", hitUnits(events), []int{1, 3, 4})
	})
}

// TestBlastEffects は、固定ダメージ・HP の下限・撃破・経験なしを確かめる（DES-NG-02）。
func TestBlastEffects(t *testing.T) {
	t.Run("TC-DES-NG-02-05 HP の境界: ちょうど 0 は倒れ、1 残れば倒れず、0 未満にならない", func(t *testing.T) {
		cases := []struct {
			name      string
			hp        int
			wantHP    int
			wantAlive bool
		}{
			{"HP 9 は 1 残る", 9, 1, true},
			{"HP 8 はちょうど 0 で倒れる", 8, 0, false},
			{"HP 3 は 0 未満にならず 0 で止まる", 3, 0, false},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				b := mustNewStill(t, blastSetup(bomb(8, 0, 3)))
				b.SetHP(3, tt.hp)
				events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
				u := b.Units()[3]
				if u.HP != tt.wantHP || u.Alive != tt.wantAlive {
					t.Errorf("HP %d、Alive %v, 期待 HP %d、Alive %v", u.HP, u.Alive, tt.wantHP, tt.wantAlive)
				}
				want := []tactics.EventKind{tactics.EventBlast, tactics.EventBlastHit}
				if !tt.wantAlive {
					want = append(want, tactics.EventFall)
				}
				assertEqual(t, "種類", kinds(events), want)
				// Amount は HP で切らない固定ダメージ。
				assertEqual(t, "BlastHit の量", events[1].Amount, 8)
				assertEqual(t, "BlastHit の相手", events[1].Target, 0)
			})
		}
	})

	t.Run("TC-DES-NG-02-06 防御・地形・武器の相性・命中判定は無視し、乱数を引かない", func(t *testing.T) {
		s := blastSetup(bomb(8, 0, 3))
		s.Units[3].Stats.Def = 99
		s.Units[3].Skill = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 100}
		s.Terrain[3][3] = forest
		b := mustNewStill(t, s)
		noRolls(t, b)
		mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		if hp := b.Units()[3].HP; hp != 12 {
			t.Errorf("HP = %d, 期待 12（防御 99・森・大盾でも固定ダメージ 8）", hp)
		}
	})

	t.Run("TC-DES-NG-02-07 爆弾の撃破では経験を与えない", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(99, 1, 3)))
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		for _, e := range events {
			if e.Kind == tactics.EventExp || e.Kind == tactics.EventLevelUp {
				t.Errorf("経験の出来事 %v が出た", e.Kind)
			}
		}
		if u := b.Units()[0]; u.Level != 1 || u.Exp != 0 {
			t.Errorf("使った人のレベル %d・経験 %d, 期待 1・0", u.Level, u.Exp)
		}
		assertEqual(t, "倒れた敵", []bool{b.Units()[3].Alive, b.Units()[4].Alive}, []bool{false, false})
	})

	t.Run("TC-DES-NG-02-08 範囲に誰もいなくても使え、在庫が減って行動済みになる", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(8, 0, 3)))
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 8)))
		assertEqual(t, "種類", kinds(events), []tactics.EventKind{tactics.EventBlast})
		assertEqual(t, "爆心", events[0].To, pos(3, 8))
		assertEqual(t, "名前", events[0].Item, "爆弾")
		if !b.Units()[0].Acted {
			t.Error("行動済みになっていない")
		}
	})
}

// TestBlastStock は、在庫の減り方と、1 回の行動で 1 つだけ使えることを確かめる（DES-NG-02）。
func TestBlastStock(t *testing.T) {
	t.Run("TC-DES-NG-02-09 使うと在庫が 1 減り、0 になったら手持ちから消える。HP が最大でも使える", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(8, 0, 3)))
		mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 8)))
		assertEqual(t, "手持ち", b.Units()[0].Items, []tactics.ItemStack{{Item: bomb(8, 0, 3), Count: 1}})

		s := blastSetup(bomb(8, 0, 3))
		s.Units[0] = withBomb(s.Units[0], bomb(8, 0, 3), 1)
		b = mustNewStill(t, s)
		mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 8)))
		if items := b.Units()[0].Items; len(items) != 0 {
			t.Errorf("手持ち %+v, 期待 空", items)
		}
	})

	t.Run("TC-DES-NG-02-10 1 回の行動で使えるのは 1 つ（使った人は行動済みになり、同じ番にもう一度は使えない）", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(8, 0, 3)))
		mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 8)))
		before := snapshot(b)
		if _, err := b.Act(throwCmd(0, pos(3, 6), pos(3, 8))); !errors.Is(err, tactics.ErrAlreadyActed) {
			t.Fatalf("誤り = %v, 期待 ErrAlreadyActed", err)
		}
		assertEqual(t, "失敗しても状態は変わらない", snapshot(b), before)
	})
}

// TestBlastValidation は、検証の誤りと、失敗したら状態を変えないことを確かめる（DES-NG-02、DES-ERR-01）。
func TestBlastValidation(t *testing.T) {
	cases := []struct {
		name string
		cmd  func() tactics.Command
		want error
	}{
		{"TC-DES-NG-02-11 投げられる距離ちょうど（3）は通る", func() tactics.Command { return throwCmd(0, pos(3, 6), pos(3, 3)) }, nil},
		{"TC-DES-NG-02-12 投げられる距離を 1 超える（4）と ErrOutOfThrowRange", func() tactics.Command { return throwCmd(0, pos(3, 6), pos(3, 2)) }, tactics.ErrOutOfThrowRange},
		{"TC-DES-NG-02-12 距離は移動後の位置から数える（移動前は 2 だが、移動後は 4）", func() tactics.Command { return throwCmd(0, pos(3, 7), pos(3, 3)) }, tactics.ErrOutOfThrowRange},
		{"TC-DES-NG-02-13 マップの外は ErrBlastOutOfMap", func() tactics.Command { return throwCmd(0, pos(3, 6), pos(3, 9)) }, tactics.ErrBlastOutOfMap},
		{"TC-DES-NG-02-13 負の座標もマップの外", func() tactics.Command { return throwCmd(0, pos(3, 6), pos(-1, 6)) }, tactics.ErrBlastOutOfMap},
		{"TC-DES-NG-02-14 爆心がなければ ErrNoBlastCenter", func() tactics.Command {
			return tactics.Command{Unit: 0, To: pos(3, 6), Action: tactics.UseItem, Item: 0}
		}, tactics.ErrNoBlastCenter},
		{"TC-DES-NG-02-15 在庫にない添字は ErrNoItem", func() tactics.Command {
			c := throwCmd(0, pos(3, 6), pos(3, 3))
			c.Item = 1
			return c
		}, tactics.ErrNoItem},
		{"TC-DES-NG-02-16 UseItem 以外に爆心を付けると ErrInvalidAction", func() tactics.Command {
			c := throwCmd(0, pos(3, 6), pos(3, 3))
			c.Action = tactics.Wait
			return c
		}, tactics.ErrInvalidAction},
		{"TC-DES-NG-02-17 移動できないマスからは使えない（ErrUnreachable）", func() tactics.Command { return throwCmd(0, pos(0, 0), pos(0, 1)) }, tactics.ErrUnreachable},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			b := mustNewStill(t, blastSetup(bomb(8, 1, 3)))
			before, histories := snapshot(b), len(b.History())
			_, err := b.Act(tt.cmd())
			if tt.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("誤り = %v, 期待 %v", err, tt.want)
			}
			assertEqual(t, "状態", snapshot(b), before)
			assertEqual(t, "履歴の数", len(b.History()), histories)
			if _, ferr := b.BlastForecast(tt.cmd()); !errors.Is(ferr, tt.want) {
				t.Errorf("BlastForecast の誤り = %v, 期待 %v", ferr, tt.want)
			}
		})
	}

	t.Run("TC-DES-NG-02-18 回復薬に爆心を付けると ErrNotBomb で、状態は変わらない", func(t *testing.T) {
		s := blastSetup(bomb(8, 1, 3))
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "回復薬", Heal: 10}, Count: 1}}
		b := mustNewStill(t, s)
		b.SetHP(0, 5)
		before := snapshot(b)
		if _, err := b.Act(throwCmd(0, pos(3, 6), pos(3, 3))); !errors.Is(err, tactics.ErrNotBomb) {
			t.Fatalf("誤り = %v, 期待 ErrNotBomb", err)
		}
		assertEqual(t, "状態", snapshot(b), before)
	})

	t.Run("TC-DES-NG-02-19 通れない地形（壁）にも爆心を置け、隣のユニットを巻き込む", func(t *testing.T) {
		s := blastSetup(bomb(8, 1, 3))
		s.Terrain[3] = grid(".#.......")[0] // (1, 3) を壁に。爆心は (1, 3)。
		b := mustNewStill(t, s)
		events := mustAct(t, b, throwCmd(0, pos(3, 4), pos(1, 3)))
		// 距離 1 以内: (2, 3) の味方 1 だけ（壁の上には誰もいない）。
		assertEqual(t, "巻き込んだユニット", hitUnits(events), []int{1})
	})

	t.Run("TC-DES-NG-02-20 回復薬は今までどおり、HP が最大だと ErrFullHP", func(t *testing.T) {
		s := blastSetup(bomb(8, 1, 3))
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "回復薬", Heal: 10}, Count: 1}}
		b := mustNewStill(t, s)
		_, err := b.Act(tactics.Command{Unit: 0, To: pos(3, 6), Action: tactics.UseItem, Item: 0})
		if !errors.Is(err, tactics.ErrFullHP) {
			t.Fatalf("誤り = %v, 期待 ErrFullHP", err)
		}
	})
}

// TestBlastJudge は、爆発による勝ち負けを確かめる（DES-NG-02）。
func TestBlastJudge(t *testing.T) {
	t.Run("TC-DES-NG-02-21 ボスを爆発で倒せば勝ち", func(t *testing.T) {
		setup := blastSetup(bomb(99, 0, 3))
		setup.Units[3] = foe("敵 A", 0, 0)
		setup.Units[7] = boss(3, 3)
		b := mustNewStill(t, setup)
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 3)))
		if b.Result != tactics.Won || events[len(events)-1].Kind != tactics.EventWon {
			t.Errorf("Result = %v, 最後の出来事 %v", b.Result, events[len(events)-1].Kind)
		}
	})

	t.Run("TC-DES-NG-02-22 主人公が自分の爆発で倒れれば負け", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(99, 0, 3)))
		events := mustAct(t, b, throwCmd(0, pos(3, 6), pos(3, 6)))
		if b.Result != tactics.Lost || events[len(events)-1].Kind != tactics.EventLost {
			t.Errorf("Result = %v, 最後の出来事 %v", b.Result, events[len(events)-1].Kind)
		}
		if b.Units()[0].Alive {
			t.Error("主人公が倒れていない")
		}
	})
}

// TestBlastForecast は、爆発の予想が実際の結果と一致し、状態を変えないことを確かめる（DES-NG-03）。
func TestBlastForecast(t *testing.T) {
	t.Run("TC-DES-NG-03-01 予想は巻き込む全員（味方・本人を含む）を添字の順に並べ、HP・倒れるかを数える", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(8, 2, 3)))
		b.SetHP(2, 8) // 味方 2 は HP 8 でちょうど倒れる
		b.SetHP(4, 9) // 敵 B は HP 9 で 1 残る
		b.SetHP(5, 2) // 敵 C は 0 未満にならず倒れる
		before := snapshot(b)
		f, err := b.BlastForecast(throwCmd(0, pos(3, 5), pos(3, 3)))
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "状態は変わらない", snapshot(b), before)
		assertEqual(t, "中心・半径・ダメージ", [3]int{f.Center.X*10 + f.Center.Y, f.Radius, f.Damage}, [3]int{33, 2, 8})
		want := []tactics.BlastTarget{
			{Unit: 0, Name: "主人公", Side: tactics.Player, HP: 20, After: 12},
			{Unit: 1, Name: "味方 1", Side: tactics.Player, HP: 20, After: 12},
			{Unit: 2, Name: "味方 2", Side: tactics.Player, HP: 8, After: 0, Falls: true},
			{Unit: 3, Name: "敵 A", Side: tactics.Enemy, HP: 20, After: 12},
			{Unit: 4, Name: "敵 B", Side: tactics.Enemy, HP: 9, After: 1},
			{Unit: 5, Name: "敵 C", Side: tactics.Enemy, HP: 2, After: 0, Falls: true},
		}
		assertEqual(t, "対象", f.Targets, want)

		// 同じ入力で Act した結果と一致する。
		events := mustAct(t, b, throwCmd(0, pos(3, 5), pos(3, 3)))
		for _, tg := range f.Targets {
			if u := b.Units()[tg.Unit]; u.HP != tg.After || u.Alive == tg.Falls {
				t.Errorf("ユニット %d: HP %d、Alive %v, 予想 After %d、Falls %v", tg.Unit, u.HP, u.Alive, tg.After, tg.Falls)
			}
		}
		assertEqual(t, "巻き込んだ数", len(hitUnits(events)), len(f.Targets))
	})

	t.Run("TC-DES-NG-03-02 爆弾でないアイテム・UseItem 以外は拒む", func(t *testing.T) {
		s := blastSetup(bomb(8, 1, 3))
		s.Units[0].Items = []tactics.ItemStack{{Item: tactics.Item{Name: "回復薬", Heal: 10}, Count: 1}}
		b := mustNewStill(t, s)
		if _, err := b.BlastForecast(tactics.Command{Unit: 0, To: pos(3, 6), Action: tactics.UseItem, Item: 0}); !errors.Is(err, tactics.ErrNotBomb) {
			t.Errorf("回復薬の誤り = %v, 期待 ErrNotBomb", err)
		}
		c := throwCmd(0, pos(3, 6), pos(3, 3))
		c.Action = tactics.Attack
		if _, err := b.BlastForecast(c); !errors.Is(err, tactics.ErrInvalidAction) {
			t.Errorf("Attack の誤り = %v, 期待 ErrInvalidAction", err)
		}
	})
}

// TestBlastCells は、範囲のマスの計算（マップの端を切る）を確かめる（DES-NG-02）。
func TestBlastCells(t *testing.T) {
	b := mustNewStill(t, blastSetup(bomb(8, 1, 3)))
	cases := []struct {
		name   string
		center tactics.Pos
		radius int
		want   int
	}{
		{"TC-DES-NG-02-23 半径 0 は 1 マス", pos(4, 4), 0, 1},
		{"TC-DES-NG-02-23 半径 1 は 5 マス", pos(4, 4), 1, 5},
		{"TC-DES-NG-02-23 半径 2 は 13 マス", pos(4, 4), 2, 13},
		{"TC-DES-NG-02-23 隅は半径 2 でマップの外を切って 6 マス", pos(0, 0), 2, 6},
		{"TC-DES-NG-02-23 辺は半径 1 で 4 マス", pos(0, 4), 1, 4},
		{"TC-DES-NG-02-23 負の半径は空", pos(4, 4), -1, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cells := b.CellsWithin(tt.center, tt.radius)
			if len(cells) != tt.want {
				t.Fatalf("%d マス %v, 期待 %d", len(cells), cells, tt.want)
			}
			for i, p := range cells {
				if p.X < 0 || p.Y < 0 || p.X >= b.Width() || p.Y >= b.Height() || abs(p.X-tt.center.X)+abs(p.Y-tt.center.Y) > tt.radius {
					t.Errorf("範囲外のマス %v", p)
				}
				if i > 0 && (cells[i-1].Y > p.Y || cells[i-1].Y == p.Y && cells[i-1].X >= p.X) {
					t.Errorf("Y・X の順でない: %v の次が %v", cells[i-1], p)
				}
			}
		})
	}
}

// TestBlastRewind は、巻き戻しで爆弾の使用・在庫・被害が完全に元へ戻ることを確かめる（DES-NG-02、DES-RWD-01）。
func TestBlastRewind(t *testing.T) {
	t.Run("TC-DES-NG-02-24 巻き戻すと、位置・在庫・HP・生死・行動済みが元に戻り、やり直せる", func(t *testing.T) {
		b := mustNewStill(t, blastSetup(bomb(99, 2, 3)))
		before := snapshot(b)
		events := mustAct(t, b, throwCmd(0, pos(3, 4), pos(3, 3)))
		if b.Units()[0].Alive || len(hitUnits(events)) < 4 {
			t.Fatalf("前提: 本人と複数が倒れる爆発のはず（本人 Alive %v、巻き込み %d）", b.Units()[0].Alive, len(hitUnits(events)))
		}
		if b.Result != tactics.Lost {
			t.Fatalf("前提: 本人が倒れて負け（Result %v）", b.Result)
		}
		h := b.History()
		if err := b.Rewind(h[len(h)-1].ID); err != nil {
			t.Fatal(err)
		}
		after := snapshot(b)
		assertEqual(t, "戻した後の状態", after, before)
		assertEqual(t, "在庫", b.Units()[0].Items, []tactics.ItemStack{{Item: bomb(99, 2, 3), Count: 2}})

		// 戻したあと、別の使い方でやり直せる。
		events = mustAct(t, b, throwCmd(0, pos(3, 6), pos(6, 6)))
		assertEqual(t, "やり直しの種類", kinds(events), []tactics.EventKind{tactics.EventBlast})
		assertEqual(t, "在庫", b.Units()[0].Items, []tactics.ItemStack{{Item: bomb(99, 2, 3), Count: 1}})
	})
}

// TestBlastSetup は、爆弾のアイテムの入力の検証を確かめる（DES-NG-02）。
func TestBlastSetup(t *testing.T) {
	cases := []struct {
		name string
		it   tactics.Item
		ok   bool
	}{
		{"TC-DES-NG-02-25 半径 0 の爆弾は正しい", bomb(1, 0, 1), true},
		{"TC-DES-NG-02-25 回復薬は今までどおり正しい", tactics.Item{Name: "回復薬", Heal: 10}, true},
		{"TC-DES-NG-02-26 爆弾の回復量は 0 でなければならない", tactics.Item{Bomb: true, Heal: 5, BlastDamage: 5, BlastRadius: 1, ThrowRange: 2}, false},
		{"TC-DES-NG-02-26 固定ダメージは 1 以上", bomb(0, 1, 2), false},
		{"TC-DES-NG-02-26 投げられる距離は 1 以上", bomb(5, 1, 0), false},
		{"TC-DES-NG-02-26 半径は 0 以上", bomb(5, -1, 2), false},
		{"TC-DES-NG-02-26 爆弾でないアイテムに爆弾の値は書けない", tactics.Item{Heal: 10, BlastDamage: 3}, false},
		{"TC-DES-NG-02-26 回復量 0 の回復薬は今までどおり誤り", tactics.Item{Name: "空"}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := blastSetup(bomb(8, 1, 3))
			s.Units[0].Items = []tactics.ItemStack{{Item: tt.it, Count: 1}}
			_, _, err := tactics.New(s, 1)
			if tt.ok != (err == nil) {
				t.Fatalf("New の誤り = %v, 期待 ok=%v", err, tt.ok)
			}
			if err != nil && !errors.Is(err, tactics.ErrInvalidSetup) {
				t.Errorf("誤り %v は ErrInvalidSetup を包んでいない", err)
			}
		})
	}
}

// TestBlastEnemyAI は、敵が爆弾を持っていても使わないことを確かめる（DES-NG-02）。
func TestBlastEnemyAI(t *testing.T) {
	t.Run("TC-DES-NG-02-27 敵は爆弾を持っていても使わない", func(t *testing.T) {
		s := blastSetup(bomb(8, 1, 3))
		s.Units[3] = withBomb(foe("敵 A", 3, 5), bomb(8, 1, 3), 1)
		b, _, err := tactics.New(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		events, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range events {
			if e.Kind == tactics.EventBlast || e.Kind == tactics.EventBlastHit {
				t.Fatalf("敵が爆弾を使った: %v", fmt.Sprint(e.Kind))
			}
		}
		assertEqual(t, "敵の爆弾は減らない", b.Units()[3].Items, []tactics.ItemStack{{Item: bomb(8, 1, 3), Count: 1}})
	})
}

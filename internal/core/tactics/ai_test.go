package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 敵の動き方（DES-AI-01、FR-BTL-07）のテスト。
//
// ユニットは helper_test.go の unit（HP 20、力 5、技 5、速さ 5、守備 3、鉄の剣）を基本にする。
// その 2 体が平地で戦うと、ダメージ 5 ＋ 5 − 3 = 7、命中率 (10 ＋ 90) − 10 = 90、回数 1 になる（DES-BTL-05）。
// 敵軍の番は EndPhase で進める。

// move は Move の出来事の比べる部分。
type move struct {
	Unit     int
	From, To tactics.Pos
}

// moves は出来事の列から Move だけを取り出す。
func moves(events []tactics.Event) []move {
	var out []move
	for _, e := range events {
		if e.Kind == tactics.EventMove {
			out = append(out, move{e.Unit, e.From, e.To})
		}
	}
	return out
}

func enemy(name string, x, y, mov int) tactics.UnitSetup {
	return unit(name, tactics.Enemy, x, y, mov)
}

func mustEndPhase(t *testing.T, b *tactics.Battle) []tactics.Event {
	t.Helper()
	events, err := b.EndPhase()
	if err != nil {
		t.Fatalf("EndPhase: %v", err)
	}
	return events
}

// plain7 は 7 × 7 の平地。
func plain7(units ...tactics.UnitSetup) tactics.Setup {
	return tactics.Setup{
		Terrain: grid(".......", ".......", ".......", ".......", ".......", ".......", "......."),
		Units:   units,
	}
}

// phaseOnly は、敵が何もしなかった敵軍の番の出来事（PhaseStart 2 つ）。
var phaseOnly = []tactics.EventKind{tactics.EventPhaseStart, tactics.EventPhaseStart}

// TestEnemyChoice は、攻撃の候補の比べ方 1〜5 を確かめる（DES-AI-01）。
func TestEnemyChoice(t *testing.T) {
	// 横の盤面: 0: 主人公 (1, 3)、1: 味方 (5, 3)、2: 敵 (3, 3) 移動 1、3: ボス (6, 0)。
	// 敵が攻撃できるのは、(2, 3) から主人公、(4, 3) から味方の 2 つ。
	row := func(p0, p1 tactics.UnitSetup) tactics.Setup {
		return plain7(p0, p1, enemy("敵", 3, 3, 1), boss(6, 0))
	}

	t.Run("TC-FR-BTL-07-01 比べ方 1: 倒せる相手を先にする（ダメージ × 回数 ≧ 今の HP）", func(t *testing.T) {
		// 味方は守備 5 で、敵のダメージは 5（主人公への 7 より小さい）。
		p1 := ally("味方", 5, 3, 2)
		p1.Stats.Def = 5
		for _, tc := range []struct {
			name    string
			hp      int
			rolls   []int
			move    move
			strikes []strike
			fall    bool
		}{
			// HP 5: 5 × 1 ≧ 5 で倒せる。比べ方 2（450 < 630）と 4（添字 1 > 0）では主人公を選ぶはずの組。
			{"HP 5 なら倒せる味方を狙う", 5, []int{0, 0}, move{2, pos(3, 3), pos(4, 3)},
				[]strike{{Unit: 2, Target: 1, Hit: true, Amount: 5}}, true},
			// HP 6: 倒せないので、比べ方 2 で主人公（7 × 90 = 630 > 5 × 90 = 450）。
			{"HP 6 なら倒せないので主人公を狙う", 6, []int{99, 99, 99, 99}, move{2, pos(3, 3), pos(2, 3)},
				[]strike{{Unit: 2, Target: 0}, {Unit: 0, Target: 2}}, false},
		} {
			b := mustNew(t, row(lord(1, 3, 2), p1))
			b.SetHP(1, tc.hp)
			done := rolls(t, b, tc.rolls...)
			events := mustEndPhase(t, b)
			done()
			assertEqual(t, tc.name+": Move", moves(events), []move{tc.move})
			assertEqual(t, tc.name+": Strike", strikes(events), tc.strikes)
			assertEqual(t, tc.name+": 味方が倒れた", !b.Units()[1].Alive, tc.fall)
		}
	})

	t.Run("TC-FR-BTL-07-02 比べ方 2: ダメージ × 命中率 × 回数 が大きい相手を狙う", func(t *testing.T) {
		// 主人公は守備 5 で 5 × 90 × 1 = 450、味方は 7 × 90 × 1 = 630。
		p0 := lord(1, 3, 2)
		p0.Stats.Def = 5
		b := mustNew(t, row(p0, ally("味方", 5, 3, 2)))
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "Move", moves(events), []move{{2, pos(3, 3), pos(4, 3)}})
		assertEqual(t, "Strike", strikes(events), []strike{{Unit: 2, Target: 1}, {Unit: 1, Target: 2}})

		// 回数も掛ける: 味方は守備 5・速さ 1 で、ダメージ 5、命中率 100 − 2 = 98、追撃で 2 回。
		// 5 × 98 × 2 = 980 が、主人公の 7 × 90 × 1 = 630 より大きい（倒せない: 10 < 20）。
		p1 := ally("味方", 5, 3, 2)
		p1.Stats.Def, p1.Stats.Spd = 5, 1
		b = mustNew(t, row(lord(1, 3, 2), p1))
		done = rolls(t, b, 99, 99, 99, 99, 99, 99)
		events = mustEndPhase(t, b)
		done()
		assertEqual(t, "回数: Move", moves(events), []move{{2, pos(3, 3), pos(4, 3)}})
		assertEqual(t, "回数: Strike", strikes(events), []strike{{Unit: 2, Target: 1}, {Unit: 1, Target: 2}, {Unit: 2, Target: 1}})
	})

	t.Run("TC-FR-BTL-07-03 比べ方 3: 反撃のダメージ × 命中率 × 回数 が小さい相手を狙う", func(t *testing.T) {
		// 与える値はどちらも 630。主人公は力 7 で、反撃は 9 × 90 × 1 = 810。味方の反撃は 7 × 90 × 1 = 630。
		p0 := lord(1, 3, 2)
		p0.Stats.Str = 7
		b := mustNew(t, row(p0, ally("味方", 5, 3, 2)))
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "Move", moves(events), []move{{2, pos(3, 3), pos(4, 3)}})
		assertEqual(t, "Strike", strikes(events), []strike{{Unit: 2, Target: 1}, {Unit: 1, Target: 2}})
	})

	t.Run("TC-FR-BTL-07-04 比べ方 4: 値がすべて同じなら、添字の小さい自軍を狙う", func(t *testing.T) {
		// 縦の盤面: 0: 主人公 (3, 5)、1: 味方 (3, 1)、2: 敵 (3, 3) 移動 1。
		// 主人公は (3, 4) から、味方は y の小さい (3, 2) から狙える。比べ方 5 なら味方を選ぶはずの組。
		b := mustNew(t, plain7(lord(3, 5, 2), ally("味方", 3, 1, 2), enemy("敵", 3, 3, 1), boss(6, 0)))
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "Move", moves(events), []move{{2, pos(3, 3), pos(3, 4)}})
		assertEqual(t, "Strike", strikes(events), []strike{{Unit: 2, Target: 0}, {Unit: 0, Target: 2}})
	})

	t.Run("TC-FR-BTL-07-05 比べ方 5: 同じ相手なら、移動先の y が小さく、同じなら x が小さいマス", func(t *testing.T) {
		// 主人公 (3, 3)、敵 (3, 5) 移動 3。主人公の隣で行けるのは (3, 4)・(2, 3)・(4, 3)
		// （(3, 2) は主人公を回り込むので 5 かかる）。y の小さい (2, 3)・(4, 3) のうち x の小さい (2, 3)。
		b := mustNew(t, plain7(lord(3, 3, 2), enemy("敵", 3, 5, 3), boss(6, 0)))
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "Move", moves(events), []move{{1, pos(3, 5), pos(2, 3)}})
		assertEqual(t, "Strike", strikes(events), []strike{{Unit: 1, Target: 0}, {Unit: 0, Target: 1}})
	})

	t.Run("TC-FR-BTL-07-06 比べる値は予測と同じ（鼓舞を含む）。移動せずに攻撃すれば Move を出さず、狙撃が効く", func(t *testing.T) {
		// TC-FR-BTL-07-05 の盤面で、鼓舞を持つボスを (6, 3) に置く。(4, 3) だけがボスから距離 2 で、
		// 敵の命中が +10: 命中率 110 − 10 = 100、7 × 100 = 700 が、ほかのマスの 630 より大きい。
		bs := boss(6, 3)
		bs.Skill = tactics.Skill{Name: "鼓舞", Effect: tactics.Inspire, Range: 2, Hit: 10, Avoid: 10}
		b := mustNew(t, plain7(lord(3, 3, 2), enemy("敵", 3, 5, 3), bs))
		// 敵の 1 撃は命中率 100 なので (99, 99) でも当たる。主人公の反撃は命中率 100 − 20 = 80 で外れる。
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "鼓舞: Move", moves(events), []move{{1, pos(3, 5), pos(4, 3)}})
		assertEqual(t, "鼓舞: Strike", strikes(events), []strike{{Unit: 1, Target: 0, Hit: true, Amount: 7}, {Unit: 0, Target: 1}})

		// 狙撃を持つ移動 0 のアーチャー (3, 1) が、2 マス先の主人公 (3, 3) を攻撃する。
		// 命中は 7 × 2 ＋ 85 ＋ 20 = 119、回避 10 で命中率 100。(99, 99) の和 198 < 200 で当たる（狙撃がなければ 99 で外れる）。
		// ダメージは 5 ＋ 6 − 3 = 8。剣の主人公は 2 マス先に反撃できない。
		ar := archer(3, 1)
		ar.Stats.Mov = 0
		ar.Skill = skillSnipe
		b = mustNew(t, plain7(lord(3, 3, 2), ar, boss(6, 0)))
		done = rolls(t, b, 99, 99)
		events = mustEndPhase(t, b)
		done()
		assertEqual(t, "狙撃: 出来事", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart, tactics.EventStrike, tactics.EventPhaseStart,
		})
		assertEqual(t, "狙撃: Strike", strikes(events), []strike{{Unit: 1, Target: 0, Hit: true, Amount: 8}})
	})
}

// TestEnemyApproach は、攻撃の候補のない突っ込む敵の近づき方と「近さ」を確かめる（DES-AI-01）。
func TestEnemyApproach(t *testing.T) {
	tests := []struct {
		id, name string
		setup    tactics.Setup
		// want は敵（添字 1 か 2）の Move。nil なら動かない。
		want []move
	}{
		{
			// 主人公 (0, 2)、森 (3, 2)、敵 (4, 2) 移動 2。近さは (3, 2) が 3、(3, 1)・(3, 3) が 4、今いるマスが 5。
			// 出発するマスの費用を数えると、(3, 2)・(3, 1)・(3, 3) がどれも 5 になり、(3, 1) を選んでしまう。
			id: "TC-FR-BTL-07-07", name: "出発するマスの費用は数えない",
			setup: tactics.Setup{
				Terrain: grid("......", "......", "...F..", "......", "......"),
				Units:   []tactics.UnitSetup{lord(0, 2, 2), enemy("敵", 4, 2, 2), boss(5, 4)},
			},
			want: []move{{1, pos(4, 2), pos(3, 2)}},
		},
		{
			// 森 (0, 1) の主人公と、平地 (8, 1) の味方の真ん中 (4, 1) に敵（移動 1）。
			// 近さは (5, 1) が 3（味方まで）、(3, 1) が 1 ＋ 1 ＋ 2 = 4（森の主人公まで）。
			// 自軍のいるマスの費用を数えないと、どちらも 2 で、x の小さい (3, 1) を選んでしまう。
			id: "TC-FR-BTL-07-08", name: "自軍のいるマスに入る費用は数える",
			setup: tactics.Setup{
				Terrain: grid(".........", "F........", ".........", "........."),
				Units:   []tactics.UnitSetup{lord(0, 1, 2), enemy("敵", 4, 1, 1), boss(0, 3), ally("味方", 8, 1, 2)},
			},
			want: []move{{1, pos(4, 1), pos(5, 1)}},
		},
		{
			// 1 本道 (y = 1) の主人公 (0, 1)、ボス (3, 1)、敵 (5, 1) 移動 1。下に回り道がある。
			// ユニットを無視すると、近さは (4, 1) が 4、今いるマスが 5、(6, 1) が 10 なので (4, 1) へ。
			// ボスを通れないものとすると、(4, 1) は 12、(6, 1) は 10 になり、(6, 1) を選んでしまう。
			id: "TC-FR-BTL-07-09", name: "近さはユニットを無視する",
			setup: tactics.Setup{
				Terrain: grid("#######", ".......", ".#####.", "......."),
				Units:   []tactics.UnitSetup{lord(0, 1, 2), boss(3, 1), enemy("敵", 5, 1, 1)},
			},
			want: []move{{2, pos(5, 1), pos(4, 1)}},
		},
		{
			// 主人公 (0, 1)、森 (1, 1)・(2, 1)、敵 (3, 1) 移動 1（森には入れない）。
			// 近さは今いるマスが 5（森を通っても上を通っても）、(3, 0) が 4 なので (3, 0) へ。
			// 森を 1 と数えると、今いるマスが 3 になり、動かなくなってしまう。
			id: "TC-FR-BTL-07-10", name: "近さは森の費用 2 で数える",
			setup: tactics.Setup{
				Terrain: grid(".....", ".FF.."),
				Units:   []tactics.UnitSetup{lord(0, 1, 2), enemy("敵", 3, 1, 1), boss(4, 1)},
			},
			want: []move{{1, pos(3, 1), pos(3, 0)}},
		},
		{
			// 主人公 (0, 0)、敵 (4, 4) 移動 1。近さは (3, 4)・(4, 3) が 7 で同じ。y の小さい (4, 3) へ。
			id: "TC-FR-BTL-07-11", name: "近さが同じなら y の小さいマス",
			setup: tactics.Setup{
				Terrain: grid("......", "......", "......", "......", "......", "......"),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), enemy("敵", 4, 4, 1), boss(5, 0)},
			},
			want: []move{{1, pos(4, 4), pos(4, 3)}},
		},
		{
			// 主人公 (0, 0)、味方 (4, 0)、ボス (2, 0)、壁 (2, 1)、敵 (2, 2) 移動 1。
			// 近さは (1, 2)・(3, 2) が 3 で同じ（今いるマスは 4）。x の小さい (1, 2) へ。
			id: "TC-FR-BTL-07-12", name: "近さも y も同じなら x の小さいマス",
			setup: tactics.Setup{
				Terrain: grid(".....", "..#..", "....."),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), boss(2, 0), enemy("敵", 2, 2, 1), ally("味方", 4, 0, 2)},
			},
			want: []move{{2, pos(2, 2), pos(1, 2)}},
		},
		{
			// 1 本道の主人公 (0, 0)、ボス (3, 0)、敵 (4, 0) 移動 1。ボスのマスには止まれず、(2, 0) へは 2 かかる。
			// 行ける (5, 0) の近さは 5 で、今いるマスの 4 より小さくないので動かない。
			id: "TC-FR-BTL-07-13", name: "今いるマスより近くなるマスがなければ動かない",
			setup: tactics.Setup{
				Terrain: grid("......", "######"),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), boss(3, 0), enemy("敵", 4, 0, 1)},
			},
		},
		{
			// 壁の列 x = 2 で分けられ、どの自軍にもたどり着けない（近さはどのマスも無限大）。
			id: "TC-FR-BTL-07-14", name: "どの自軍にもたどり着けなければ動かない",
			setup: tactics.Setup{
				Terrain: grid("..#..", "..#.."),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), enemy("敵", 3, 0, 3), boss(4, 1)},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			b := mustNew(t, tc.setup)
			noRolls(t, b)
			before := b.Units()
			events := mustEndPhase(t, b)
			want := phaseOnly
			if len(tc.want) > 0 {
				want = []tactics.EventKind{tactics.EventPhaseStart, tactics.EventMove, tactics.EventPhaseStart}
			}
			assertEqual(t, "出来事", kinds(events), want)
			assertEqual(t, "Move", moves(events), tc.want)
			if len(tc.want) == 0 {
				assertEqual(t, "ユニット", b.Units(), before)
			}
		})
	}
}

// TestEnemyHold は、待ち構える敵と、敵の動く順を確かめる（DES-AI-01、DES-BTL-02）。
func TestEnemyHold(t *testing.T) {
	t.Run("TC-FR-BTL-07-15 待ち構える敵は、攻撃の候補があれば攻撃し、以後は突っ込むに変わる", func(t *testing.T) {
		// 主人公 (1, 0) 移動 2、待ち構えるボス (3, 0) 移動 1。
		bs := boss(3, 0)
		bs.Stats.Mov = 1
		s := tactics.Setup{Terrain: grid("......", "......", "......", "......"), Units: []tactics.UnitSetup{lord(1, 0, 2), bs}}
		b := mustNew(t, s)
		done := rolls(t, b, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "1 ターン目: Move", moves(events), []move{{1, pos(3, 0), pos(2, 0)}})
		assertEqual(t, "1 ターン目: Strike", strikes(events), []strike{{Unit: 1, Target: 0}, {Unit: 0, Target: 1}})
		assertEqual(t, "動き方", b.Units()[1].AI, tactics.Charge)

		// 2 ターン目: 主人公が (1, 2) に下がると攻撃の候補はない。突っ込むに変わったので近づく。
		// 近さは (1, 0)・(2, 1) が 2 で同じ（今いるマスは 3）。y の小さい (1, 0) へ。
		noRolls(t, b)
		events = mustAct(t, b, tactics.Command{Unit: 0, To: pos(1, 2), Action: tactics.Wait})
		assertEqual(t, "2 ターン目: 出来事", kinds(events), []tactics.EventKind{
			tactics.EventMove, tactics.EventWait, tactics.EventPhaseStart, tactics.EventMove, tactics.EventPhaseStart,
		})
		assertEqual(t, "2 ターン目: Move", moves(events), []move{{0, pos(1, 0), pos(1, 2)}, {1, pos(2, 0), pos(1, 0)}})
	})

	t.Run("TC-FR-BTL-07-16 待ち構える敵は、攻撃の候補がなければ動かず、出来事も出さない", func(t *testing.T) {
		// 主人公 (0, 0)、待ち構えるボス (5, 3) 移動 3、待ち構える敵 (3, 3) 移動 3。
		// どちらも主人公の隣に行けない。突っ込むなら近づくはずの位置。
		e := enemy("敵", 3, 3, 3)
		e.AI = tactics.Hold
		bs := boss(5, 3)
		bs.Stats.Mov = 3
		s := tactics.Setup{Terrain: grid("......", "......", "......", "......"), Units: []tactics.UnitSetup{lord(0, 0, 2), bs, e}}
		b := mustNew(t, s)
		noRolls(t, b)
		before := b.Units()
		for turn := 1; turn <= 2; turn++ {
			events := mustEndPhase(t, b)
			assertEqual(t, "出来事", kinds(events), phaseOnly)
		}
		assertEqual(t, "ユニット（動き方は待ち構えるのまま）", b.Units(), before)
	})

	t.Run("TC-FR-BTL-07-17 敵は添字の順に 1 体ずつ動き、先に動いた敵の位置を踏まえる", func(t *testing.T) {
		// 主人公 (0, 0)、ボス (4, 2)、敵 A (2, 0) 移動 1（添字 2）、敵 B (1, 1) 移動 1（添字 3）。
		// A は (1, 0) から攻撃する。B は (1, 0) が空いていれば y の小さい (1, 0) を選ぶが、A がいるので (0, 1) から攻撃する。
		s := tactics.Setup{
			Terrain: grid(".....", ".....", "....."),
			Units:   []tactics.UnitSetup{lord(0, 0, 2), boss(4, 2), enemy("敵 A", 2, 0, 1), enemy("敵 B", 1, 1, 1)},
		}
		b := mustNew(t, s)
		done := rolls(t, b, 99, 99, 99, 99, 99, 99, 99, 99)
		events := mustEndPhase(t, b)
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventPhaseStart,
			tactics.EventMove, tactics.EventStrike, tactics.EventStrike,
			tactics.EventMove, tactics.EventStrike, tactics.EventStrike,
			tactics.EventPhaseStart,
		})
		assertEqual(t, "Move", moves(events), []move{{2, pos(2, 0), pos(1, 0)}, {3, pos(1, 1), pos(0, 1)}})
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 2, Target: 0}, {Unit: 0, Target: 2}, {Unit: 3, Target: 0}, {Unit: 0, Target: 3},
		})
	})
}

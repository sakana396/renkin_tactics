package tactics_test

import (
	"errors"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 経験と成長（DES-GRW-01〜03）のテスト。計算例の番号は設計書（反復 2）4.2 節のもの。
// キャラの能力と成長率は、設計書（反復 2）10.1・10.3 節の値（v0.3）。

// leveled は、ユニットの入力にレベル・経験・成長率を付ける。
func leveled(u tactics.UnitSetup, level, exp int, g tactics.Growth) tactics.UnitSetup {
	u.Level = level
	u.Exp = exp
	u.Growth = g
	return u
}

var (
	growthLeon = tactics.Growth{HP: 55, Str: 45, Skl: 65, Spd: 70, Def: 20}
	growthBran = tactics.Growth{HP: 70, Str: 75, Skl: 35, Spd: 30, Def: 15}
	growthMira = tactics.Growth{HP: 55, Str: 40, Skl: 60, Spd: 45, Def: 20}
)

// leon3 は v0.3 のレオン（主人公。19/7/11/12/4/5）。
func leon3(x, y, level, exp int) tactics.UnitSetup {
	u := chara("レオン", tactics.Player, x, y, tactics.Stats{HP: 19, Str: 7, Skl: 11, Spd: 12, Def: 4, Mov: 5}, ironSword, skillInspire)
	u.Lord = true
	return leveled(u, level, exp, growthLeon)
}

// bran3 は v0.3 のブラン（22/11/5/5/2/5）。
func bran3(x, y, level, exp int) tactics.UnitSetup {
	u := chara("ブラン", tactics.Player, x, y, tactics.Stats{HP: 22, Str: 11, Skl: 5, Spd: 5, Def: 2, Mov: 5}, ironAxe, skillWrath)
	return leveled(u, level, exp, growthBran)
}

// mira3 は v0.3 のミラ（17/5/9/7/3/5。反復 1 と同じ）。
func mira3(x, y, level, exp int) tactics.UnitSetup {
	return leveled(mira(x, y), level, exp, growthMira)
}

// expEvent は Exp の出来事の比べる部分。
type expEvent struct{ Unit, Amount int }

func exps(events []tactics.Event) []expEvent {
	var out []expEvent
	for _, e := range events {
		if e.Kind == tactics.EventExp {
			out = append(out, expEvent{e.Unit, e.Amount})
		}
	}
	return out
}

// levelUp は LevelUp の出来事の比べる部分。
type levelUp struct {
	Unit, Level int
	Gains       tactics.Stats
}

func levelUps(events []tactics.Event) []levelUp {
	var out []levelUp
	for _, e := range events {
		if e.Kind == tactics.EventLevelUp {
			out = append(out, levelUp{e.Unit, e.Level, e.Gains})
		}
	}
	return out
}

// grown は、ユニットのレベル・経験・能力・今の HP。
type grown struct {
	Level, Exp int
	Stats      tactics.Stats
	HP         int
}

func grownOf(u tactics.Unit) grown { return grown{u.Level, u.Exp, u.Stats, u.HP} }

// attackCmd は、移動せずに target を攻撃するコマンド。
func attackCmd(b *tactics.Battle, unit, target int) tactics.Command {
	return tactics.Command{Unit: unit, To: b.Units()[unit].Pos, Action: tactics.Attack, Target: target}
}

// enemyAttacks は、敵軍の番に enemy だけが target を攻撃するようにして、自軍の番を終える。
func enemyAttacks(t *testing.T, b *tactics.Battle, enemy, target int) []tactics.Event {
	t.Helper()
	b.SetEnemyTarget(func(u int) int {
		if u == enemy {
			return target
		}
		return -1
	})
	events, err := b.EndPhase()
	if err != nil {
		t.Fatalf("EndPhase: %v", err)
	}
	return events
}

// example2 は計算例 2: ミラ（Lv 1、経験 90）が、2 マス先の HP 1 のソルジャーを倒す。成長の乱数は (12, 75, 40, 44, 90, 99, 99, 99)。
//
//	0: ミラ (1, 1)、1: ソルジャー (3, 1)、2: ボス (6, 0)、3: レオン（主人公。Lv 0）(0, 6)
func example2(t *testing.T) (*tactics.Battle, []tactics.Event) {
	t.Helper()
	b := mustNew(t, arena(mira3(1, 1, 1, 90), soldier(3, 1), farBoss(6, 0), leon(0, 6)))
	b.SetHP(1, 1)
	done := rolls(t, b, 0, 0, 99, 12, 75, 40, 44, 90, 99, 99, 99)
	events := mustAct(t, b, attackCmd(b, 0, 1))
	done()
	return b, events
}

// example4 は計算例 4: レオン（Lv 3、経験 99）が、隣のソルジャーを攻撃し、1 撃も当てずに生き残る。
// 成長の乱数は (99, 0, 65, 69, 20, 99, 99, 99)。
//
//	0: レオン (2, 1)、1: ソルジャー (3, 1)、2: ボス (6, 0)、3: 控え (6, 6)
func example4(t *testing.T) (*tactics.Battle, []tactics.Event) {
	t.Helper()
	b := mustNew(t, arena(leon3(2, 1, 3, 99), soldier(3, 1), farBoss(6, 0), reserve()))
	// 1 撃目・反撃・追撃はすべて外れ。
	done := rolls(t, b, 99, 99, 99, 99, 99, 99, 99, 0, 65, 69, 20, 99, 99, 99)
	events := mustAct(t, b, attackCmd(b, 0, 1))
	done()
	return b, events
}

// TestExp は経験を得る規則を確かめる（DES-GRW-01、FR-UNT-04）。
func TestExp(t *testing.T) {
	t.Run("TC-FR-UNT-04-01 計算例 1（当てて倒さなければ経験 10。成長の乱数は引かない）", func(t *testing.T) {
		b := mustNew(t, arena(leon3(2, 1, 1, 0), soldier(3, 1), farBoss(6, 0), reserve()))
		// 1 撃目は当たり、反撃と追撃は外れ。成長の乱数を引けば、引き過ぎで止まる。
		done := rolls(t, b, 0, 0, 99, 99, 99, 99, 99)
		events := mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventStrike, tactics.EventStrike, tactics.EventStrike, tactics.EventExp,
		})
		assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 10}})
		u := b.Units()[0]
		assertEqual(t, "レベルと経験", [2]int{u.Level, u.Exp}, [2]int{1, 10})
	})

	t.Run("TC-FR-UNT-04-02 倒せば 30、当てれば 10（ダメージ 0 でも）、それ以外は 1", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			foeHP   int
			foeDef  int
			rolls   []int
			want    int
			wantDie bool
		}{
			{"1 撃目で倒す", 1, 4, []int{0, 0, 99}, 30, true},
			{"当てて倒さない", 0, 4, []int{0, 0, 99, 99, 99, 99, 99}, 10, false},
			{"ダメージ 0 で当たる", 0, 99, []int{0, 0, 99, 99, 99, 99, 99}, 10, false},
			{"すべて外れる", 0, 4, []int{99, 99, 99, 99, 99, 99}, 1, false},
			{"自分はすべて外れ、相手の反撃だけ当たる", 0, 4, []int{99, 99, 0, 0, 99, 99, 99}, 1, false},
			{"1 撃目が外れ、追撃で倒す", 1, 4, []int{99, 99, 99, 99, 0, 0, 99}, 30, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				foe := soldier(3, 1)
				foe.Stats.Def = tc.foeDef
				b := mustNew(t, arena(leon3(2, 1, 1, 0), foe, farBoss(6, 0), reserve()))
				if tc.foeHP > 0 {
					b.SetHP(1, tc.foeHP)
				}
				done := rolls(t, b, tc.rolls...)
				events := mustAct(t, b, attackCmd(b, 0, 1))
				done()
				assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: tc.want}})
				assertEqual(t, "経験", b.Units()[0].Exp, tc.want)
				assertEqual(t, "相手が倒れた", !b.Units()[1].Alive, tc.wantDie)
			})
		}
	})

	t.Run("TC-FR-UNT-04-03 計算例 6（敵軍の番に攻撃され、反撃できずに生き残れば経験 1）", func(t *testing.T) {
		// 0: ミラ (2, 1)、1: ファイター (3, 1)、2: ボス (6, 0)、3: レオン（Lv 0）(0, 6)
		b := mustNew(t, arena(mira3(2, 1, 1, 0), fighter(3, 1), farBoss(6, 0), leon(0, 6)))
		// ファイターの 1 撃だけ（ミラは隣に反撃できず、追撃もない）。
		done := rolls(t, b, 99, 99)
		events := enemyAttacks(t, b, 1, 0)
		done()
		assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 1}})
		assertEqual(t, "経験", b.Units()[0].Exp, 1)
	})

	t.Run("TC-FR-UNT-04-04 倒れた自軍・Lv 0 の自軍・待機・アイテム・敵は経験を得ない", func(t *testing.T) {
		check := func(t *testing.T, b *tactics.Battle, events []tactics.Event, i, level, exp int) {
			t.Helper()
			assertEqual(t, "Exp", exps(events), []expEvent(nil))
			assertEqual(t, "LevelUp", levelUps(events), []levelUp(nil))
			u := b.Units()[i]
			assertEqual(t, "レベルと経験", [2]int{u.Level, u.Exp}, [2]int{level, exp})
			for _, u := range b.Units() {
				if u.Side == tactics.Enemy && (u.Level != 0 || u.Exp != 0) {
					t.Errorf("敵 %s のレベルと経験が %d・%d", u.Name, u.Level, u.Exp)
				}
			}
		}

		t.Run("自軍が攻撃されて倒れる", func(t *testing.T) {
			b := mustNew(t, arena(mira3(2, 1, 1, 50), fighter(3, 1), farBoss(6, 0), leon(0, 6)))
			b.SetHP(0, 1)
			done := rolls(t, b, 0, 0, 99)
			events := enemyAttacks(t, b, 1, 0)
			done()
			check(t, b, events, 0, 1, 50)
		})
		t.Run("Lv 0 の自軍が倒す", func(t *testing.T) {
			b := mustNew(t, arena(leon(2, 1), soldier(3, 1), farBoss(6, 0), reserve()))
			b.SetHP(1, 1)
			done := rolls(t, b, 0, 0, 99)
			events := mustAct(t, b, attackCmd(b, 0, 1))
			done()
			check(t, b, events, 0, 0, 0)
		})
		t.Run("待機", func(t *testing.T) {
			b := mustNew(t, arena(leon3(2, 1, 1, 50), soldier(5, 1), farBoss(6, 0), reserve()))
			noRolls(t, b)
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 2), Action: tactics.Wait})
			check(t, b, events, 0, 1, 50)
		})
		t.Run("アイテムの使用", func(t *testing.T) {
			u := leon3(2, 1, 1, 50)
			u.Items = []tactics.ItemStack{{Item: tactics.Item{ID: "potion", Name: "回復薬", Heal: 10}, Count: 1}}
			b := mustNew(t, arena(u, soldier(5, 1), farBoss(6, 0), reserve()))
			b.SetHP(0, 5)
			noRolls(t, b)
			events := mustAct(t, b, tactics.Command{Unit: 0, To: pos(2, 1), Action: tactics.UseItem, Item: 0})
			check(t, b, events, 0, 1, 50)
		})
		t.Run("主人公が攻撃されて倒れる", func(t *testing.T) {
			b := mustNew(t, arena(leon3(2, 1, 1, 50), fighter(3, 1), farBoss(6, 0)))
			b.SetHP(0, 1)
			done := rolls(t, b, 0, 0, 99)
			events := enemyAttacks(t, b, 1, 0)
			done()
			check(t, b, events, 0, 1, 50)
			ks := kinds(events)
			assertEqual(t, "終わりの出来事", ks[len(ks)-2:], []tactics.EventKind{tactics.EventFall, tactics.EventLost})
		})
	})

	t.Run("TC-FR-UNT-04-05 計算例 2・4（100 以上でレベルを上げ、余りを持ち越す）", func(t *testing.T) {
		b, events := example2(t)
		assertEqual(t, "計算例 2 の Exp", exps(events), []expEvent{{Unit: 0, Amount: 30}})
		u := b.Units()[0]
		assertEqual(t, "計算例 2 のレベルと経験", [2]int{u.Level, u.Exp}, [2]int{2, 20})

		b, events = example4(t)
		assertEqual(t, "計算例 4 の Exp", exps(events), []expEvent{{Unit: 0, Amount: 1}})
		u = b.Units()[0]
		assertEqual(t, "計算例 4 のレベルと経験", [2]int{u.Level, u.Exp}, [2]int{4, 0})
	})

	t.Run("TC-FR-UNT-04-06 計算例 5（Lv 20 になったら経験を 0 にし、以後は経験を得ない）", func(t *testing.T) {
		b := mustNew(t, arena(leon3(2, 1, 19, 95), soldier(3, 1), farBoss(6, 0), reserve()))
		// 1 撃目は当たり、反撃と追撃は外れ。そのあと成長の乱数 5 つ。
		done := rolls(t, b, 0, 0, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99)
		events := mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 10}})
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 20}})
		u := b.Units()[0]
		assertEqual(t, "レベルと経験", [2]int{u.Level, u.Exp}, [2]int{20, 0})

		// Lv 20 のユニットが倒しても、経験を得ず、成長の乱数も引かない。
		b = mustNew(t, arena(leon3(2, 1, 20, 0), soldier(3, 1), farBoss(6, 0), reserve()))
		b.SetHP(1, 1)
		done = rolls(t, b, 0, 0, 99)
		events = mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "Lv 20 の出来事", kinds(events), []tactics.EventKind{tactics.EventStrike, tactics.EventFall})
		u = b.Units()[0]
		assertEqual(t, "Lv 20 のレベルと経験", [2]int{u.Level, u.Exp}, [2]int{20, 0})
	})

	t.Run("TC-FR-UNT-04-07 計算例 3（出来事の順と、Exp・LevelUp の写しは更新後の値）", func(t *testing.T) {
		// 0: ブラン (2, 2)、1: 剣士 (3, 2)、2: ボス (6, 0)、3: レオン（Lv 0）(0, 6)
		b := mustNew(t, arena(bran3(2, 2, 1, 80), myrmidon(3, 2), farBoss(6, 0), leon(0, 6)))
		b.SetHP(1, 10)
		done := rolls(t, b, 20, 30, 99, 50, 75, 10, 30, 5, 99, 99, 99)
		events := mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventStrike, tactics.EventFall, tactics.EventExp, tactics.EventLevelUp,
		})
		assertEqual(t, "Strike", strikes(events), []strike{{Unit: 0, Target: 1, Hit: true, Amount: 16}})
		assertEqual(t, "Exp", exps(events), []expEvent{{Unit: 0, Amount: 30}})
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 2, Gains: tactics.Stats{HP: 1, Skl: 1, Def: 1}}})
		want := grown{Level: 2, Exp: 10, Stats: tactics.Stats{HP: 23, Str: 11, Skl: 6, Spd: 5, Def: 3, Mov: 5}, HP: 23}
		assertEqual(t, "Exp の写し", grownOf(events[2].Units[0]), want)
		assertEqual(t, "LevelUp の写し", grownOf(events[3].Units[0]), want)
		assertEqual(t, "状態", grownOf(b.Units()[0]), want)
	})

	t.Run("TC-FR-UNT-04-08 ボスを倒した 1 撃でも、経験とレベルアップのあとに勝ちを出す", func(t *testing.T) {
		b := mustNew(t, arena(leon3(5, 0, 1, 90), farBoss(6, 0), reserve()))
		b.SetHP(1, 1)
		done := rolls(t, b, 0, 0, 99, 99, 99, 99, 99, 99, 99, 99, 99)
		events := mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "出来事", kinds(events), []tactics.EventKind{
			tactics.EventStrike, tactics.EventFall, tactics.EventExp, tactics.EventLevelUp, tactics.EventWon,
		})
		assertEqual(t, "結果", b.Result, tactics.Won)
	})
}

// TestGrowthSetup は Setup の規則 7 を確かめる（DES-GRW-03、DES-BTL-01）。
func TestGrowthSetup(t *testing.T) {
	t.Run("TC-FR-UNT-04-09 規則 7 に当てはまらなければ ErrInvalidSetup", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			modify func(s *tactics.Setup)
		}{
			{"Level −1", func(s *tactics.Setup) { s.Units[0].Level = -1 }},
			{"Level 21", func(s *tactics.Setup) { s.Units[0].Level = 21 }},
			{"敵の Level 1", func(s *tactics.Setup) { s.Units[3].Level = 1 }},
			{"増援の Level 1", func(s *tactics.Setup) { s.Reinforcements[0].Unit.Level = 1 }},
			{"Exp −1", func(s *tactics.Setup) { s.Units[0].Level, s.Units[0].Exp = 1, -1 }},
			{"Exp 100", func(s *tactics.Setup) { s.Units[0].Level, s.Units[0].Exp = 1, 100 }},
			{"Level 0 で Exp 1", func(s *tactics.Setup) { s.Units[0].Exp = 1 }},
			{"Level 20 で Exp 1", func(s *tactics.Setup) { s.Units[0].Level, s.Units[0].Exp = 20, 1 }},
			{"Growth の HP −1", func(s *tactics.Setup) { s.Units[1].Level, s.Units[1].Growth.HP = 1, -1 }},
			{"Growth の守備 101", func(s *tactics.Setup) { s.Units[1].Level, s.Units[1].Growth.Def = 1, 101 }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := baseSetup()
				s.Reinforcements = []tactics.ReinforcementSetup{{Turn: 2, Unit: foe("増援", 2, 2)}}
				// 変える前の入力は正しい。
				if _, _, err := tactics.New(s, 1); err != nil {
					t.Fatalf("変える前: %v", err)
				}
				tc.modify(&s)
				_, _, err := tactics.New(s, 1)
				if !errors.Is(err, tactics.ErrInvalidSetup) {
					t.Fatalf("err = %v, want ErrInvalidSetup", err)
				}
			})
		}
	})

	t.Run("TC-FR-UNT-04-10 規則 7 の境界の値で作れ、Units() に写る", func(t *testing.T) {
		s := baseSetup()
		zero := tactics.Growth{}
		full := tactics.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}
		s.Units[0] = leveled(s.Units[0], 20, 0, zero)
		s.Units[1] = leveled(s.Units[1], 1, 99, full)
		s.Units[1].Items = []tactics.ItemStack{{Item: tactics.Item{ID: "potion", Name: "回復薬", Heal: 10}, Count: 1}}
		b := mustNew(t, s)
		u := b.Units()
		assertEqual(t, "主人公", [2]int{u[0].Level, u[0].Exp}, [2]int{20, 0})
		assertEqual(t, "主人公の成長率", u[0].Growth, zero)
		assertEqual(t, "味方", [2]int{u[1].Level, u[1].Exp}, [2]int{1, 99})
		assertEqual(t, "味方の成長率", u[1].Growth, full)
		assertEqual(t, "アイテムの ID", u[1].Items[0].Item.ID, "potion")
	})
}

// TestGrowth はレベルアップのときの成長を確かめる（DES-GRW-02、FR-UNT-05）。
func TestGrowth(t *testing.T) {
	t.Run("TC-FR-UNT-05-01 計算例 2（成長率より小さい乱数の能力が 1 上がる）", func(t *testing.T) {
		b, events := example2(t)
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 2, Gains: tactics.Stats{HP: 1, Skl: 1, Spd: 1}}})
		assertEqual(t, "状態", grownOf(b.Units()[0]), grown{
			Level: 2, Exp: 20, Stats: tactics.Stats{HP: 18, Str: 5, Skl: 10, Spd: 8, Def: 3, Mov: 5}, HP: 18,
		})
	})

	t.Run("TC-FR-UNT-05-02 計算例 4（乱数が成長率と等しければ上がらない）", func(t *testing.T) {
		b, events := example4(t)
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 4, Gains: tactics.Stats{Str: 1, Spd: 1}}})
		assertEqual(t, "状態", grownOf(b.Units()[0]), grown{
			Level: 4, Exp: 0, Stats: tactics.Stats{HP: 19, Str: 8, Skl: 11, Spd: 13, Def: 4, Mov: 5}, HP: 19,
		})
	})

	t.Run("TC-FR-UNT-05-03 成長率 0 は上がらず引き直さない。100 は必ず上がる。移動は上がらない", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			growth tactics.Growth
			roll   int
			gains  tactics.Stats
		}{
			{"成長率 0、乱数 0", tactics.Growth{}, 0, tactics.Stats{}},
			{"成長率 100、乱数 99", tactics.Growth{HP: 100, Str: 100, Skl: 100, Spd: 100, Def: 100}, 99, tactics.Stats{HP: 1, Str: 1, Skl: 1, Spd: 1, Def: 1}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				u := leveled(leon3(2, 1, 1, 99), 1, 99, tc.growth)
				b := mustNew(t, arena(u, soldier(3, 1), farBoss(6, 0), reserve()))
				// 1 撃目・反撃・追撃はすべて外れ。そのあと成長の乱数がちょうど 5 つ。
				r := tc.roll
				done := rolls(t, b, 99, 99, 99, 99, 99, 99, r, r, r, r, r, 99, 99, 99)
				events := mustAct(t, b, attackCmd(b, 0, 1))
				done()
				assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 2, Gains: tc.gains}})
				st := b.Units()[0].Stats
				assertEqual(t, "移動", st.Mov, 5)
				assertEqual(t, "最大 HP", st.HP, 19+tc.gains.HP)
			})
		}
	})

	t.Run("TC-FR-UNT-05-04 最大 HP が上がれば、今の HP も 1 上がる", func(t *testing.T) {
		u := leveled(leon3(2, 1, 1, 99), 1, 99, tactics.Growth{HP: 100})
		b := mustNew(t, arena(u, soldier(3, 1), farBoss(6, 0), reserve()))
		b.SetHP(0, 10)
		done := rolls(t, b, 99, 99, 99, 99, 99, 99, 0, 0, 0, 0, 0, 99, 99, 99)
		mustAct(t, b, attackCmd(b, 0, 1))
		done()
		got := b.Units()[0]
		assertEqual(t, "最大 HP と今の HP", [2]int{got.Stats.HP, got.HP}, [2]int{20, 11})
	})
}

// TestGrowthDeterminism は成長の乱数の順と再現性を確かめる（DES-RNG-01、NFR-06）。
func TestGrowthDeterminism(t *testing.T) {
	t.Run("TC-NFR-06-04 1 撃ごとの乱数（大盾を含む）をすべて引いたあとに、成長の乱数を HP・力・技・速さ・守備の順に引く", func(t *testing.T) {
		shield := soldier(3, 1)
		shield.Skill = skillPavise
		b := mustNew(t, arena(leon3(2, 1, 1, 99), shield, farBoss(6, 0), reserve()))
		done := rolls(t, b,
			0, 0, 99, 50, // 1 撃目: 当たり。大盾は 50 < 30 でないので出ない
			99, 99, // 反撃: 外れ
			0, 0, 99, 10, // 追撃: 当たり。大盾は 10 < 30 で出る
			0, 99, 0, 99, 0, 99, 99, 99, // 成長: HP ○、力 ×、技 ○、速さ ×、守備 ○
		)
		events := mustAct(t, b, attackCmd(b, 0, 1))
		done()
		assertEqual(t, "Strike", strikes(events), []strike{
			{Unit: 0, Target: 1, Hit: true, Amount: 7},
			{Unit: 1, Target: 0, Hit: false},
			{Unit: 0, Target: 1, Hit: true, Amount: 3, Pavise: true},
		})
		assertEqual(t, "LevelUp", levelUps(events), []levelUp{{Unit: 0, Level: 2, Gains: tactics.Stats{HP: 1, Skl: 1, Def: 1}}})
		assertEqual(t, "経験", b.Units()[0].Exp, 9)
	})

	t.Run("TC-NFR-06-05 同じ種と同じコマンドの列なら、経験・レベル・成長も同じ", func(t *testing.T) {
		run := func() ([]tactics.Event, state) {
			s := arena(leon3(2, 1, 1, 99), soldier(3, 1), farBoss(6, 0), mira3(1, 3, 1, 99))
			b, events, err := tactics.New(s, 11)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range []tactics.Command{
				{Unit: 0, To: pos(2, 1), Action: tactics.Attack, Target: 1},
				{Unit: 3, To: pos(1, 3), Action: tactics.Wait},
			} {
				ev, err := b.Act(c)
				if err != nil {
					t.Fatal(err)
				}
				events = append(events, ev...)
			}
			return events, snapshot(b)
		}
		e1, s1 := run()
		e2, s2 := run()
		if len(levelUps(e1)) == 0 {
			t.Fatal("LevelUp が出ていない（確かめる場面になっていない）")
		}
		assertEqual(t, "出来事", e1, e2)
		assertEqual(t, "状態", s1, s2)
	})
}

package tactics

// テストだけで使う口（DES-RNG-01）。

// SetRoll は、0〜99 の乱数を引く関数を差し替える。
func (b *Battle) SetRoll(f func() int) { b.roll = f }

// Roll は、乱数を 1 つ引く。
func (b *Battle) Roll() int { return b.roll() }

// KillUnit は、攻撃で倒れたのと同じ処理（fall）でユニットを倒れた状態にする。
// 自軍が攻撃で倒れるのは敵の攻撃（#10）からなので、それまでの代わりに使う。
func (b *Battle) KillUnit(i int) { b.fall(nil, i) }

// SetHP は、ユニットの今の HP を変える（怒りの条件や、回復・勝ち負けの前の HP を作るため）。
func (b *Battle) SetHP(i, hp int) { b.units[i].HP = hp }

// SetEnemyTarget は、敵軍の番で敵 1 体を動かす処理を「動かずに f(unit) の添字のユニットを攻撃する。
// -1 なら何もしない」に差し替える。敵の動き方（#10）がなくても、敵の攻撃で勝ち負けが決まる流れを確かめるため。
func (b *Battle) SetEnemyTarget(f func(unit int) int) {
	b.actEnemyOverride = func(events []Event, unit int) []Event {
		if t := f(unit); t >= 0 {
			return b.attack(events, unit, t, false)
		}
		return events
	}
}

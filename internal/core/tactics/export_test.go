package tactics

// テストだけで使う口（DES-RNG-01）。

// SetRoll は、0〜99 の乱数を引く関数を差し替える。
func (b *Battle) SetRoll(f func() int) { b.roll = f }

// Roll は、乱数を 1 つ引く。
func (b *Battle) Roll() int { return b.roll() }

// KillUnit は、ユニットを倒れた状態にする（攻撃の Issue までの代わり）。
func (b *Battle) KillUnit(i int) { b.units[i].Alive = false }

// ErrNotImplemented は、まだ作っていない行動の誤り。
var ErrNotImplemented = errNotImplemented

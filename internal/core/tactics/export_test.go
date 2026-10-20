package tactics

// テストだけで使う口（DES-RNG-01）。

// SetRoll は、0〜99 の乱数を引く関数を差し替える。
func (b *Battle) SetRoll(f func() int) { b.roll = f }

// Roll は、乱数を 1 つ引く。
func (b *Battle) Roll() int { return b.roll() }

// KillUnit は、攻撃で倒れたのと同じ処理（fall）でユニットを倒れた状態にする。
// 自軍が攻撃で倒れるのは敵の攻撃（#10）からなので、それまでの代わりに使う。
func (b *Battle) KillUnit(i int) { b.fall(nil, i) }

// ErrNotImplemented は、まだ作っていない行動の誤り。
var ErrNotImplemented = errNotImplemented

// SetHP は、ユニットの今の HP を変える（怒りの条件を作るため）。
func (b *Battle) SetHP(i, hp int) { b.units[i].HP = hp }

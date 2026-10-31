package tactics

import (
	"container/heap"
	"math"
)

// 敵の動き方（DES-AI-01）。

// unreachable は、どの自軍にもたどり着けないマスの近さ（無限大）。
const unreachable = math.MaxInt

// candidate は攻撃の候補 1 つ（DES-AI-01）。敵が to に移動し、武器の列の weapon 番目で target を攻撃する（DES-ITM-02）。
type candidate struct {
	to     Pos
	target int
	weapon int
	// kill は倒せるか（ダメージ × 攻撃の回数 ≧ 相手の今の HP。必殺と大盾は数えない）。
	kill bool
	// gain は ダメージ × 命中率 × 攻撃の回数。
	gain int
	// risk は 反撃のダメージ × 反撃の命中率 × 反撃の回数。
	risk int
}

// better は、候補 c が候補 o よりよいかを返す（DES-AI-01 の比べ方 1〜5、DES-IT06 10.1 節の 6）。乱数は使わない。
func (c candidate) better(o candidate) bool {
	// 1. 倒せるものを先にする。
	if c.kill != o.kill {
		return c.kill
	}
	// 2. ダメージ × 命中率 × 攻撃の回数 が大きいもの。
	if c.gain != o.gain {
		return c.gain > o.gain
	}
	// 3. 反撃のダメージ × 反撃の命中率 × 反撃の回数 が小さいもの。
	if c.risk != o.risk {
		return c.risk < o.risk
	}
	// 4. 狙う自軍の添字が小さいもの。
	if c.target != o.target {
		return c.target < o.target
	}
	// 5. 移動先のマスの y が小さいもの、同じなら x が小さいもの。
	if c.to != o.to {
		return posLess(c.to, o.to)
	}
	// 6. 武器の列の添字が小さいもの。
	return c.weapon < o.weapon
}

// posLess は、a が b より y が小さいか、y が同じで x が小さいかを返す。
func posLess(a, b Pos) bool {
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.X < b.X
}

// actEnemy は敵 1 体を動き方に従って動かす（DES-AI-01、DES-BTL-02）。
//
// 攻撃の候補があれば、最もよい候補のマスに移動して攻撃する。待ち構える敵は、攻撃したら突っ込むに変わる。
// 候補がなければ、突っ込む敵は近さが小さくなるマスに移動し、待ち構える敵は動かない。
// 動かない敵は出来事を出さない。
func (b *Battle) actEnemy(events []Event, unit int) []Event {
	reach := b.Reachable(unit)
	if best, ok := b.bestCandidate(unit, reach); ok {
		b.units[unit].AI = Charge
		from := b.units[unit].Pos
		events = b.moveEnemy(events, unit, best.to)
		// 攻撃に使った武器を装備する（DES-ITM-02）。
		b.units[unit].Equipped = best.weapon
		return b.attack(events, unit, best.target, best.to == from)
	}
	if b.units[unit].AI != Charge {
		return events
	}
	if to, ok := b.approach(unit, reach); ok {
		events = b.moveEnemy(events, unit, to)
	}
	return events
}

// moveEnemy は敵を to に移動させ、移動したら Move を出す（DES-EVT-01）。今いるマスなら何もしない。
func (b *Battle) moveEnemy(events []Event, unit int, to Pos) []Event {
	from := b.units[unit].Pos
	if to == from {
		return events
	}
	b.units[unit].Pos = to
	return b.emit(events, Event{Kind: EventMove, Unit: unit, From: from, To: to})
}

// bestCandidate は、敵 unit の攻撃の候補のうち最もよいものを返す（DES-AI-01）。候補がなければ偽を返す。
// 候補は、移動できる各マス（今いるマスを含む）・手持ちの攻撃の武器（杖を除く）・その射程に入る生きている自軍のユニットの組
// （DES-ITM-02。持ち替えて攻撃できるので、今の装備だけに限らない）。値は予測（DES-BTL-07）と同じ（鼓舞・狙撃を含む）。
func (b *Battle) bestCandidate(unit int, reach map[Pos]int) (candidate, bool) {
	var best candidate
	found := false
	for k, w := range b.units[unit].Weapons {
		if w.Type == Staff {
			continue
		}
		for to := range reach {
			for t, p := range b.units {
				if !p.Alive || p.Side != Player || !inRange(w, distance(to, p.Pos)) {
					continue
				}
				f := b.forecast(unit, t, to, k)
				c := candidate{
					to:     to,
					target: t,
					weapon: k,
					kill:   f.Attacker.Damage*f.Attacker.Count >= p.HP,
					gain:   f.Attacker.Damage * f.Attacker.HitRate * f.Attacker.Count,
					risk:   f.Defender.Damage * f.Defender.HitRate * f.Defender.Count,
				}
				if !found || c.better(best) {
					best, found = c, true
				}
			}
		}
	}
	return best, found
}

// approach は、突っ込む敵 unit が攻撃の候補のないときに移動するマスを返す（DES-AI-01）。
// 近さが今いるマスより小さくなる移動先のうち、近さが最も小さいマス（同じなら y、x の小さいマス）。
// そのようなマスがなければ偽を返す（動かない）。
func (b *Battle) approach(unit int, reach map[Pos]int) (Pos, bool) {
	near := b.closeness()
	cur := b.units[unit].Pos
	bestPos, bestNear := cur, near(cur)
	found := false
	for p := range reach {
		n := near(p)
		if n < bestNear || (found && n == bestNear && posLess(p, bestPos)) {
			bestPos, bestNear, found = p, n, true
		}
	}
	return bestPos, found
}

// closeness は、マスの「近さ」を返す関数を作る（DES-AI-01）。
//
// 近さは、あるマスから生きている自軍のいるマスのどれかまでの最小の費用。
// ユニットはいないものとし、通るマスに入る費用（地形の Cost）の和で数える。
// 出発するマスの費用は数えず、自軍のいるマスに入る費用は数える。Cost が 1 未満のマスは通れない。
// どの自軍にもたどり着けないマスは unreachable（無限大）を返す。
func (b *Battle) closeness() func(Pos) int {
	// 自軍のいるマスから逆向きにダイクストラ法でたどる。
	// q から隣の p へ進むとき、前向きには p から q に入るので、q の費用を足す。
	dist := map[Pos]int{}
	q := &posQueue{}
	for _, u := range b.units {
		if u.Alive && u.Side == Player {
			if _, ok := dist[u.Pos]; !ok {
				dist[u.Pos] = 0
				heap.Push(q, posCost{pos: u.Pos, cost: 0})
			}
		}
	}
	for q.Len() > 0 {
		cur := heap.Pop(q).(posCost)
		if cur.cost > dist[cur.pos] {
			continue
		}
		c := b.TerrainAt(cur.pos).Cost
		if c < 1 {
			continue
		}
		for _, d := range directions {
			next := Pos{cur.pos.X + d.X, cur.pos.Y + d.Y}
			if !b.inMap(next) {
				continue
			}
			total := cur.cost + c
			if prev, ok := dist[next]; ok && prev <= total {
				continue
			}
			dist[next] = total
			heap.Push(q, posCost{pos: next, cost: total})
		}
	}
	return func(p Pos) int {
		if n, ok := dist[p]; ok {
			return n
		}
		return unreachable
	}
}

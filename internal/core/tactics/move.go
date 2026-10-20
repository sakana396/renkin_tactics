package tactics

import "container/heap"

// directions は隣のマスへの向き（上、左、右、下）。
var directions = [...]Pos{{0, -1}, {-1, 0}, {1, 0}, {0, 1}}

// Reachable は、ユニット unit が移動できるマスと、そこまでの最小の費用を返す（DES-BTL-03、DES-MAP-01）。
//
// 費用は入るマスの地形の Cost で、合計が Mov 以下のマス（今いるマスを含む）に行ける。
// Cost が 0 の地形には入れない。味方のいるマスは通り抜けられるが止まれず、相手の陣営のいるマスには入れない。
// unit が存在しないか倒れていれば nil を返す。
func (b *Battle) Reachable(unit int) map[Pos]int {
	if !b.validUnit(unit) || !b.units[unit].Alive {
		return nil
	}
	u := b.units[unit]
	occupant := map[Pos]Side{}
	for i, o := range b.units {
		if i != unit && o.Alive {
			occupant[o.Pos] = o.Side
		}
	}

	// ダイクストラ法で、通れるマスまでの最小の費用を求める。
	best := map[Pos]int{u.Pos: 0}
	q := &posQueue{{pos: u.Pos, cost: 0}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(posCost)
		if cur.cost > best[cur.pos] {
			continue
		}
		for _, d := range directions {
			next := Pos{cur.pos.X + d.X, cur.pos.Y + d.Y}
			if !b.inMap(next) {
				continue
			}
			c := b.terrain[next.Y][next.X].Cost
			if c < 1 {
				continue
			}
			if side, ok := occupant[next]; ok && side != u.Side {
				continue
			}
			total := cur.cost + c
			if total > u.Stats.Mov {
				continue
			}
			if prev, ok := best[next]; ok && prev <= total {
				continue
			}
			best[next] = total
			heap.Push(q, posCost{pos: next, cost: total})
		}
	}

	// 味方のいるマスには止まれない。
	for p := range best {
		if _, ok := occupant[p]; ok {
			delete(best, p)
		}
	}
	return best
}

// AttackRange は、ユニット unit が from から武器で攻撃できるマスを返す（DES-BTL-01）。
// from からのマンハッタン距離が武器の射程（最小〜最大）に入るマップの中のマスを、y、x の小さい順に返す。
// unit が存在しないか倒れていれば nil を返す（Reachable と同じ）。
func (b *Battle) AttackRange(unit int, from Pos) []Pos {
	if !b.validUnit(unit) || !b.units[unit].Alive {
		return nil
	}
	w := b.units[unit].Weapon
	var out []Pos
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			p := Pos{x, y}
			if d := distance(from, p); d >= w.MinRange && d <= w.MaxRange {
				out = append(out, p)
			}
		}
	}
	return out
}

// distance はマンハッタン距離を返す。
func distance(a, b Pos) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// posCost はダイクストラ法の待ち行列の要素。
type posCost struct {
	pos  Pos
	cost int
}

// posQueue は費用の小さい順に取り出す待ち行列（container/heap）。
type posQueue []posCost

func (q posQueue) Len() int { return len(q) }
func (q posQueue) Less(i, j int) bool {
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	if q[i].pos.Y != q[j].pos.Y {
		return q[i].pos.Y < q[j].pos.Y
	}
	return q[i].pos.X < q[j].pos.X
}
func (q posQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *posQueue) Push(x any)   { *q = append(*q, x.(posCost)) }
func (q *posQueue) Pop() any {
	old := *q
	x := old[len(old)-1]
	*q = old[:len(old)-1]
	return x
}

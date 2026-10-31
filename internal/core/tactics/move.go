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
	return (BoardView{terrain: b.terrain, units: b.units}).Reachable(unit)
}

// Reachable は Battle と共通の移動費・占有規則で移動先を返す（DES-RWD-05、DES-BTL-03、DES-DEP-02）。
func (b BoardView) Reachable(unit int) map[Pos]int {
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

// AttackRange は、ユニット unit が from から武器で攻撃できるマスを返す（DES-BTL-01、DES-ITM-02）。
// 手持ちの攻撃の武器（杖を除く）すべての射程を合わせ、from からのマンハッタン距離がどれかの射程に入るマップの中のマスを、
// y、x の小さい順に返す。攻撃の武器を持たない（杖だけの）ユニットは、杖の射程を合わせて返す。
// unit が存在しないか倒れていれば nil を返す（Reachable と同じ）。
func (b *Battle) AttackRange(unit int, from Pos) []Pos {
	return (BoardView{terrain: b.terrain, units: b.units}).AttackRange(unit, from)
}

// WeaponRange は、ユニット unit が from から武器の列の weapon 番目で届くマスを、y、x の小さい順に返す（DES-ITM-02）。
// unit が存在しないか倒れているか、weapon が列の外なら nil を返す。
func (b *Battle) WeaponRange(unit int, from Pos, weapon int) []Pos {
	return (BoardView{terrain: b.terrain, units: b.units}).WeaponRange(unit, from, weapon)
}

// AttackRange は Battle と共通の、攻撃の武器の射程を合わせた y・x 順の攻撃マスを返す（DES-RWD-05、DES-BTL-03、DES-DEP-02、DES-ITM-02）。
func (b BoardView) AttackRange(unit int, from Pos) []Pos {
	if !b.validUnit(unit) || !b.units[unit].Alive {
		return nil
	}
	ws := b.units[unit].Weapons
	var attack []Weapon
	for _, w := range ws {
		if w.Type != Staff {
			attack = append(attack, w)
		}
	}
	if len(attack) == 0 {
		attack = ws
	}
	return rangeCells(b.Width(), b.Height(), from, attack)
}

// WeaponRange は Battle と共通の、選んだ武器 1 本の y・x 順の届くマスを返す（DES-ITM-02）。
func (b BoardView) WeaponRange(unit int, from Pos, weapon int) []Pos {
	if !b.validUnit(unit) || !b.units[unit].Alive || weapon < 0 || weapon >= len(b.units[unit].Weapons) {
		return nil
	}
	return rangeCells(b.Width(), b.Height(), from, b.units[unit].Weapons[weapon:weapon+1])
}

// rangeCells は、from からの距離が ws のどれかの射程に入るマスを y・x の小さい順に返す。
func rangeCells(width, height int, from Pos, ws []Weapon) []Pos {
	var out []Pos
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := Pos{x, y}
			d := distance(from, p)
			for _, w := range ws {
				if inRange(w, d) {
					out = append(out, p)
					break
				}
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

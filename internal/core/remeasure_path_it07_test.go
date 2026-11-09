package core_test

import (
	"container/heap"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 移動経路の ZOC 計測と、巻き戻しを含む増援の最終経路・全試行の区別（#188、TC-FR-MAP-04-33・34。DES-MAP-09、DES-BTL-11）。
//
// core の公開イベント（EventMove）は出発と到着だけを持ち、実際に通ったマスの列を持たない。
// そこで計測側が、移動の直前の盤面（イベントの写しで移動者を出発点に戻したもの）から、
// ゲームと同じ移動費・占有の決まりで出発点から到着点までの最小費用の経路を求め、次を数える。
//   - 追加費用 = ZOC を数えた最小費用 − ZOC を数えない最小費用（ゲームの規則上の正確な値）
//   - ZOC に入る回数 = 最小費用の経路のうち、入った回数が最も少ないもの（Min）と多いもの（Max）。
//     実際にどの経路を通ったかは分からないので、幅で示す。到着点を含み、出発点は含まない。
// 最小費用でない遠回りの経路は、追加費用の定義に含めない（ゲームは到着点の最小費用だけで移動を許す）。

// it07PathMove は 1 回の移動の計測値。
type it07PathMove struct {
	Enemy, Remove       bool
	Min, Max, ExtraCost int
}

// it07PathStat は陣営・移動の種類ごとの集計。
type it07PathStat struct {
	// Moves は移動の回数。ZOCMoves は最小費用の経路のすべてが ZOC に入る移動、PossibleMoves は ZOC に入りうる移動。
	Moves, ZOCMoves, PossibleMoves int
	// EntriesMin・EntriesMax は ZOC に入る回数の下限と上限の合計。ExtraCost は追加費用の合計。
	EntriesMin, EntriesMax, ExtraCost int
}

// it07PathStats は自軍・敵軍の、通常の移動と再移動（攻撃などのあとの移動）の集計。
type it07PathStats struct {
	PlayerMove, PlayerRemove, EnemyMove, EnemyRemove it07PathStat
}

func (s *it07PathStats) slot(m it07PathMove) *it07PathStat {
	switch {
	case m.Enemy && m.Remove:
		return &s.EnemyRemove
	case m.Enemy:
		return &s.EnemyMove
	case m.Remove:
		return &s.PlayerRemove
	}
	return &s.PlayerMove
}

func (s *it07PathStats) add(m it07PathMove) {
	t := s.slot(m)
	t.Moves++
	if m.Min > 0 {
		t.ZOCMoves++
	}
	if m.Max > 0 {
		t.PossibleMoves++
	}
	t.EntriesMin += m.Min
	t.EntriesMax += m.Max
	t.ExtraCost += m.ExtraCost
}

type it07Obs struct {
	Reinforce string
	Move      *it07PathMove
}

// it07PathTracker は観測した出来事から、全試行と最終経路の ZOC・増援を数える。
// 最終経路は、巻き戻しのたびに、戻したターンの最初の行動より後の記録を捨てて残るもの。
type it07PathTracker struct {
	cost  func(tactics.Pos) int
	w, h  int
	acted map[int]bool
	marks map[int]int
	all   it07PathStats
	log   []it07Obs
	// allSpawned は全試行で出た増援の ID（初めて出た順）。
	allSpawned []string
	seen       map[string]bool
}

func newIT07PathTracker(cost func(tactics.Pos) int, w, h int) *it07PathTracker {
	return &it07PathTracker{cost: cost, w: w, h: h, acted: map[int]bool{}, marks: map[int]int{}, seen: map[string]bool{}}
}

// PhaseStart は番の始まり。その番に行動した印を消す。
func (t *it07PathTracker) PhaseStart() { t.acted = map[int]bool{} }

// TurnStart は賢い打ち手が自軍の番の最初の行動をする前。巻き戻しの戻り先の位置を覚える。
func (t *it07PathTracker) TurnStart(turn int) { t.marks[turn] = len(t.log) }

// Rewind は turn の最初の行動の前へ戻したとき、それより後の最終経路の記録を捨てる。全試行の集計は変えない。
func (t *it07PathTracker) Rewind(turn int) {
	if n, ok := t.marks[turn]; ok && n <= len(t.log) {
		t.log = t.log[:n]
	}
	t.acted = map[int]bool{}
}

// Event は 1 つの出来事を数える。side は出来事が属する番の陣営、units はその時点の全ユニットの写し。
func (t *it07PathTracker) Event(e tactics.Event, units []tactics.Unit, side tactics.Side) {
	if e.Unit < 0 || e.Unit >= len(units) {
		return
	}
	u := units[e.Unit]
	switch e.Kind {
	case tactics.EventReinforce:
		id := it07ID(u)
		t.log = append(t.log, it07Obs{Reinforce: id})
		if !t.seen[id] {
			t.seen[id] = true
			t.allSpawned = append(t.allSpawned, id)
		}
	case tactics.EventStrike, tactics.EventWait:
		// EventHeal の Unit は回復を受けたユニットで、行動した側ではないので、行動の印には使わない。
		if u.Side == side {
			t.acted[e.Unit] = true
		}
	case tactics.EventMove:
		m := t.measure(units, e)
		m.Enemy = u.Side == tactics.Enemy
		m.Remove = t.acted[e.Unit]
		t.all.add(m)
		t.log = append(t.log, it07Obs{Move: &m})
	}
}

// Final は最終経路の ZOC 集計と、最終経路で出た増援の ID を返す。
func (t *it07PathTracker) Final() (it07PathStats, []string) {
	var stats it07PathStats
	var spawned []string
	for _, o := range t.log {
		if o.Move != nil {
			stats.add(*o.Move)
		} else {
			spawned = append(spawned, o.Reinforce)
		}
	}
	return stats, spawned
}

// AllTrials は全試行の ZOC 集計を返す。
func (t *it07PathTracker) AllTrials() it07PathStats { return t.all }

func (t *it07PathTracker) measure(units []tactics.Unit, e tactics.Event) it07PathMove {
	before := append([]tactics.Unit(nil), units...)
	before[e.Unit].Pos = e.From
	withZ := it07ShortestZOC(before, e.Unit, e.From, t.cost, t.w, t.h, true)
	without := it07ShortestZOC(before, e.Unit, e.From, t.cost, t.w, t.h, false)
	d, ok := withZ.dist[e.To]
	if !ok {
		return it07PathMove{}
	}
	return it07PathMove{Min: withZ.minZ[e.To], Max: withZ.maxZ[e.To], ExtraCost: d - without.dist[e.To]}
}

type it07Shortest struct {
	dist       map[tactics.Pos]int
	minZ, maxZ map[tactics.Pos]int
}

type it07Node struct {
	pos  tactics.Pos
	cost int
}
type it07Queue []it07Node

func (q it07Queue) Len() int           { return len(q) }
func (q it07Queue) Less(i, j int) bool { return q[i].cost < q[j].cost }
func (q it07Queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *it07Queue) Push(x any)        { *q = append(*q, x.(it07Node)) }
func (q *it07Queue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

// it07ShortestZOC は mover が from から行ける各マスの最小費用と、最小費用の経路で ZOC に入る回数の下限・上限を返す。
// 規則は tactics.Reachable と同じ（入るマスの地形の費用と、zoc が真のとき相手軍の生存重騎士の隣への +1。
// 相手の陣営のいるマスには入れず、味方のマスは通り抜けられる）。移動力の上限は掛けない。
func it07ShortestZOC(units []tactics.Unit, mover int, from tactics.Pos, cost func(tactics.Pos) int, w, h int, zoc bool) it07Shortest {
	side := units[mover].Side
	occupant := map[tactics.Pos]tactics.Side{}
	var knights []tactics.Pos
	for i, u := range units {
		if !u.Alive {
			continue
		}
		if i != mover {
			occupant[u.Pos] = u.Side
		}
		if u.Side != side && it07HasZOC(u) {
			knights = append(knights, u.Pos)
		}
	}
	inZOC := func(p tactics.Pos) bool {
		if !zoc {
			return false
		}
		for _, k := range knights {
			if abs(k.X-p.X)+abs(k.Y-p.Y) == 1 {
				return true
			}
		}
		return false
	}
	r := it07Shortest{dist: map[tactics.Pos]int{from: 0}, minZ: map[tactics.Pos]int{from: 0}, maxZ: map[tactics.Pos]int{from: 0}}
	q := &it07Queue{{pos: from}}
	done := map[tactics.Pos]bool{}
	for q.Len() > 0 {
		cur := heap.Pop(q).(it07Node)
		if done[cur.pos] || cur.cost > r.dist[cur.pos] {
			continue
		}
		done[cur.pos] = true
		for _, d := range [...]tactics.Pos{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			next := tactics.Pos{X: cur.pos.X + d.X, Y: cur.pos.Y + d.Y}
			if next.X < 0 || next.Y < 0 || next.X >= w || next.Y >= h {
				continue
			}
			c := cost(next)
			if c < 1 {
				continue
			}
			if s, ok := occupant[next]; ok && s != side {
				continue
			}
			z := 0
			if inZOC(next) {
				z = 1
			}
			total := cur.cost + c + z
			prev, seen := r.dist[next]
			switch {
			case !seen || total < prev:
				r.dist[next] = total
				r.minZ[next], r.maxZ[next] = r.minZ[cur.pos]+z, r.maxZ[cur.pos]+z
				heap.Push(q, it07Node{pos: next, cost: total})
			case total == prev && !done[next]:
				r.minZ[next] = min(r.minZ[next], r.minZ[cur.pos]+z)
				r.maxZ[next] = max(r.maxZ[next], r.maxZ[cur.pos]+z)
			}
		}
	}
	return r
}

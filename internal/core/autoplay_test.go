package core_test

import (
	"container/heap"
	"fmt"
	"math"
	"sort"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 自動の試し遊び（DES-MAP-06 の「実現可能な攻略を自動の固定 seed で再現」）。
//
// テストだけで使う、決まった手順で遊ぶ自軍の打ち手。公開の口（Reachable・Forecast・Act）だけを使い、
// 乱数は戦闘の seed だけで決まる（同じ seed・名簿・作戦なら同じ結果）。人の遊び方の代わりではなく、
// 「この作戦で勝てる手順がある」ことを確かめるための下限の打ち手である。

// autoPlan は自動の試し遊びの作戦。
type autoPlan struct {
	// Groups は名簿のキャラ ID ごとの隊の番号。書いていないキャラは隊 0。
	Groups map[string]int
	// Routes は隊ごとの通り道（順に目指すマス）。最後のマスに着いたら、そこに留まらずボスを目指す。
	Routes [][]tactics.Pos
	// Order は行動の順（キャラ ID）。書いていないキャラは後ろ。
	Order []string
}

// autoResult は自動の試し遊びの結果。
type autoResult struct {
	Result tactics.Result
	Turns  int
	// Fallen は倒れた自軍の名前。
	Fallen []string
	// Kills は倒した敵の数。
	Kills int
	// Deaths は自軍が倒れた記録（ターン・倒れた自軍・倒した敵とその位置）。
	Deaths []string
}

// autoLog は試し遊びの記録の出し先（調べるときだけ設定する）。
var autoLog func(string)

func (r autoResult) String() string {
	return fmt.Sprintf("結果=%v ターン=%d 倒れた自軍=%v 倒した敵=%d %v", r.Result, r.Turns, r.Fallen, r.Kills, r.Deaths)
}

// autoPlay は戦闘を maxTurn ターンまで自動で遊ぶ。ids は Units の添字ごとの自軍のキャラ ID（敵は空）。
func autoPlay(b *tactics.Battle, ids []string, plan autoPlan, maxTurn int) autoResult {
	p := &autoPlayer{b: b, ids: ids, plan: plan, step: map[int]int{}, log: autoLog}
	for b.Result == tactics.Ongoing && b.Turn <= maxTurn {
		p.playerPhase()
	}
	res := autoResult{Result: b.Result, Turns: b.Turn, Deaths: p.deaths}
	for _, u := range b.Units() {
		if !u.Alive {
			if u.Side == tactics.Player {
				res.Fallen = append(res.Fallen, u.Name)
			} else {
				res.Kills++
			}
		}
	}
	return res
}

type autoPlayer struct {
	b    *tactics.Battle
	ids  []string
	plan autoPlan
	// step は隊ごとの通り道の進み具合（Routes の添字）。
	step map[int]int
	// log は打ち手の記録を受け取る（nil なら記録しない）。
	log func(string)
	// deaths は自軍が倒れた記録。
	deaths []string
	// lastKill は最後に敵を倒したターン、kills はそのときまでに倒した敵の数。
	lastKill, kills int
}

// note は、出来事から自軍が倒れたことを記録する。
func (p *autoPlayer) note(events []tactics.Event) {
	units := p.b.Units()
	last := map[int]int{}
	turn := p.b.Turn
	for _, e := range events {
		switch e.Kind {
		case tactics.EventPhaseStart:
			turn = e.Turn
		case tactics.EventStrike:
			last[e.Target] = e.Unit
		case tactics.EventFall:
			if u := units[e.Unit]; u.Side == tactics.Player {
				k := units[last[e.Unit]]
				p.deaths = append(p.deaths, fmt.Sprintf("T%d %s←%s%v", turn, u.Name, k.Name, k.Pos))
			}
		}
	}
}

// do は行動し、記録を残す。
func (p *autoPlayer) do(cmd tactics.Command, why string) error {
	u := p.b.Units()[cmd.Unit]
	events, err := p.b.Act(cmd)
	p.note(events)
	if p.log != nil {
		p.log(fmt.Sprintf("T%d %s HP%d %v→%v %v 対象%d %s err=%v", p.b.Turn, u.Name, u.HP, u.Pos, cmd.To, cmd.Action, cmd.Target, why, err))
	}
	return err
}

func (p *autoPlayer) group(i int) int { return p.plan.Groups[p.ids[i]] }

// order は自軍の行動の順を返す。
func (p *autoPlayer) order() []int {
	rank := func(i int) int {
		for k, id := range p.plan.Order {
			if id == p.ids[i] {
				return k
			}
		}
		return len(p.plan.Order) + i
	}
	var out []int
	for i, u := range p.b.Units() {
		if u.Side == tactics.Player {
			out = append(out, i)
		}
	}
	sort.SliceStable(out, func(a, c int) bool { return rank(out[a]) < rank(out[c]) })
	return out
}

func (p *autoPlayer) playerPhase() {
	turn := p.b.Turn
	kills := 0
	for _, u := range p.b.Units() {
		if u.Side == tactics.Enemy && !u.Alive {
			kills++
		}
	}
	if kills > p.kills || p.lastKill == 0 {
		p.kills, p.lastKill = kills, turn
	}
	p.advanceRoutes()
	for _, i := range p.order() {
		if p.b.Result != tactics.Ongoing || p.b.Turn != turn || p.b.Phase != tactics.Player {
			return
		}
		u := p.b.Units()[i]
		if !u.Alive || u.Acted {
			continue
		}
		p.act(i)
	}
	if p.b.Result == tactics.Ongoing && p.b.Turn == turn && p.b.Phase == tactics.Player {
		events, _ := p.b.EndPhase()
		p.note(events)
	}
}

// advanceRoutes は、隊の生きている全員が今の目標から 4 以内に来たら、次の目標へ進める。
func (p *autoPlayer) advanceRoutes() {
	for g, route := range p.plan.Routes {
		for p.step[g] < len(route) {
			goal := route[p.step[g]]
			all, any := true, false
			for i, u := range p.b.Units() {
				if u.Side != tactics.Player || !u.Alive || p.group(i) != g {
					continue
				}
				any = true
				if dist(u.Pos, goal) > 4 {
					all = false
				}
			}
			if !any || !all {
				break
			}
			p.step[g]++
		}
	}
}

// goal は自軍 i の今の目標を返す。通り道を終えたらボスの位置。
func (p *autoPlayer) goal(i int) tactics.Pos {
	g := p.group(i)
	if g < len(p.plan.Routes) && p.step[g] < len(p.plan.Routes[g]) {
		return p.plan.Routes[g][p.step[g]]
	}
	for _, u := range p.b.Units() {
		if u.Boss && u.Alive {
			return u.Pos
		}
	}
	return p.b.Units()[i].Pos
}

func dist(a, b tactics.Pos) int {
	return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
}

// strikeOn は、敵 o が自軍 u（at に立つ）を攻撃するときの 1 撃のダメージ・命中率・回数（鼓舞は数えない）。
func (p *autoPlayer) strikeOn(o, u tactics.Unit, at tactics.Pos) (dmg, hit, count int) {
	tr := p.b.TerrainAt(at)
	t := tri(o.Weapon.Type, u.Weapon.Type)
	dmg = max(0, o.Stats.Str+o.Weapon.Might+t-(u.Stats.Def+tr.Def))
	hit = min(100, max(0, o.Stats.Skl*2+o.Weapon.Hit+15*t-(u.Stats.Spd*2+tr.Avoid)))
	count = 1
	if o.Stats.Spd-u.Stats.Spd >= 4 {
		count = 2
	}
	return dmg, hit, count
}

// danger は、自軍 i が at に立ったとき、次の敵軍の番に受けうるダメージの合計（最悪の場合）を返す。
// 敵の動き方（DES-AI-01）にならい、各敵は攻撃できる相手のうち、倒せる相手、次にダメージ × 命中率 × 回数の
// 大きい相手を選ぶとする。相手の候補は、行動済みの自軍と、at に立つ i（まだ動く仲間は数えない）。
// skip の敵は数えない（この行動で倒す見込みの敵）。命中率 0 の攻撃は数えない。
func (p *autoPlayer) danger(i int, at tactics.Pos, skip int, reach map[int]map[tactics.Pos]int) int {
	return p.dangerOf(i, at, skip, reach, true)
}

// dangerOf は danger の本体。pessimistic が偽なら、主人公も敵の動き方のとおりに狙われるとみなす。
func (p *autoPlayer) dangerOf(i int, at tactics.Pos, skip int, reach map[int]map[tactics.Pos]int, pessimistic bool) int {
	units := p.b.Units()
	type target struct {
		k  int
		at tactics.Pos
	}
	targets := []target{{i, at}}
	for k, o := range units {
		if k != i && o.Alive && o.Side == tactics.Player && o.Acted {
			targets = append(targets, target{k, o.Pos})
		}
	}
	can := func(e int, at tactics.Pos) bool {
		o := units[e]
		for t := range reach[e] {
			if d := dist(t, at); d >= o.Weapon.MinRange && d <= o.Weapon.MaxRange {
				return true
			}
		}
		return false
	}
	// 敵を添字の順に動かし、当たったものとして相手の HP を減らしながら狙いを選ぶ（弱った相手に集まる）。
	hp := map[int]int{}
	for _, tg := range targets {
		hp[tg.k] = units[tg.k].HP
	}
	total := 0
	for e, o := range units {
		if e == skip || !o.Alive || o.Side != tactics.Enemy {
			continue
		}
		best, bestKill, bestGain := -1, false, -1
		var bestDmg int
		for _, tg := range targets {
			if hp[tg.k] <= 0 || !can(e, tg.at) {
				continue
			}
			u := units[tg.k]
			dmg, hit, count := p.strikeOn(o, u, tg.at)
			kill := dmg*count >= hp[tg.k] && hit > 0
			gain := dmg * hit * count
			if best < 0 || kill && !bestKill || kill == bestKill && gain > bestGain {
				best, bestKill, bestGain, bestDmg = tg.k, kill, gain, dmg*count
				if hit == 0 {
					bestDmg = 0
				}
			}
		}
		if best >= 0 {
			hp[best] -= bestDmg
		}
		if best == i {
			total += bestDmg
		} else if pessimistic && units[i].Lord && can(e, at) {
			// 主人公は倒れたら負けなので、攻撃できる敵はすべて主人公を狙うとみなす（ほかの相手が先に倒れると狙いが移るため）。
			dmg, hit, count := p.strikeOn(o, units[i], at)
			if hit > 0 {
				total += dmg * count
			}
		}
	}
	return total
}

func tri(a, d tactics.WeaponType) int {
	beats := func(x, y tactics.WeaponType) bool {
		return (x == tactics.Sword && y == tactics.Axe) || (x == tactics.Axe && y == tactics.Lance) || (x == tactics.Lance && y == tactics.Sword)
	}
	switch {
	case beats(a, d):
		return 1
	case beats(d, a):
		return -1
	}
	return 0
}

// enemyReach は敵ごとの移動できるマス。行動済みの自軍（この番はもう動かない）だけを壁とみなし、
// まだ動く自軍は通り道をふさがないものとする（ふさいでいた仲間が動くと、道が開くため）。
func (p *autoPlayer) enemyReach() map[int]map[tactics.Pos]int {
	units := p.b.Units()
	blocked := map[tactics.Pos]bool{}
	for _, o := range units {
		if o.Alive && o.Side == tactics.Player && o.Acted {
			blocked[o.Pos] = true
		}
	}
	w, h := p.b.Width(), p.b.Height()
	out := map[int]map[tactics.Pos]int{}
	for e, o := range units {
		if !o.Alive || o.Side != tactics.Enemy {
			continue
		}
		best := map[tactics.Pos]int{o.Pos: 0}
		q := &costQueue{{o.Pos, 0}}
		for q.Len() > 0 {
			cur := heap.Pop(q).(costItem)
			if cur.c > best[cur.p] {
				continue
			}
			for _, d := range []tactics.Pos{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
				n := tactics.Pos{X: cur.p.X + d.X, Y: cur.p.Y + d.Y}
				if n.X < 0 || n.Y < 0 || n.X >= w || n.Y >= h || blocked[n] {
					continue
				}
				c := p.b.TerrainAt(n).Cost
				if c < 1 || cur.c+c > o.Stats.Mov {
					continue
				}
				if prev, ok := best[n]; ok && prev <= cur.c+c {
					continue
				}
				best[n] = cur.c + c
				heap.Push(q, costItem{n, cur.c + c})
			}
		}
		out[e] = best
	}
	return out
}

// pathCost は goal までの地形の費用（ユニットは無視）を各マスについて返す。
func (p *autoPlayer) pathCost(goal tactics.Pos) map[tactics.Pos]int {
	w, h := p.b.Width(), p.b.Height()
	best := map[tactics.Pos]int{goal: 0}
	q := &costQueue{{goal, 0}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(costItem)
		if cur.c > best[cur.p] {
			continue
		}
		c := p.b.TerrainAt(cur.p).Cost
		if c < 1 && cur.p != goal {
			continue
		}
		for _, d := range []tactics.Pos{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			n := tactics.Pos{X: cur.p.X + d.X, Y: cur.p.Y + d.Y}
			if n.X < 0 || n.Y < 0 || n.X >= w || n.Y >= h || p.b.TerrainAt(n).Cost < 1 {
				continue
			}
			total := cur.c + max(1, c)
			if prev, ok := best[n]; ok && prev <= total {
				continue
			}
			best[n] = total
			heap.Push(q, costItem{n, total})
		}
	}
	return best
}

type costItem struct {
	p tactics.Pos
	c int
}
type costQueue []costItem

func (q costQueue) Len() int { return len(q) }
func (q costQueue) Less(i, j int) bool {
	if q[i].c != q[j].c {
		return q[i].c < q[j].c
	}
	if q[i].p.Y != q[j].p.Y {
		return q[i].p.Y < q[j].p.Y
	}
	return q[i].p.X < q[j].p.X
}
func (q costQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *costQueue) Push(x any)   { *q = append(*q, x.(costItem)) }
func (q *costQueue) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// sortedTiles は移動できるマスを y・x の順に並べる（打ち手の選び方を決定的にする）。
func sortedTiles(reach map[tactics.Pos]int) []tactics.Pos {
	out := make([]tactics.Pos, 0, len(reach))
	for t := range reach {
		out = append(out, t)
	}
	sort.Slice(out, func(a, c int) bool {
		if out[a].Y != out[c].Y {
			return out[a].Y < out[c].Y
		}
		return out[a].X < out[c].X
	})
	return out
}

// limit は、自軍 i が HP hp の状態で受けてよい最悪のダメージ。主人公は半分まで、ほかは 85% まで。
// 敵を倒せない番が 3 回続くと、攻め手が止まっているとみなし、主人公は 4 分の 3、ほかは HP − 1 まで受ける。
func (p *autoPlayer) limit(u tactics.Unit, hp int) int {
	stalled := p.b.Turn-p.lastKill >= 3
	switch {
	case u.Lord && stalled:
		return (hp - 1) * 3 / 4
	case u.Lord:
		return (hp - 1) / 2
	case stalled:
		return hp - 1
	}
	return (hp - 1) * 85 / 100
}

// anchor は、隊 g の盾役（主人公と回復役を除いて守備がいちばん高い仲間）の添字。いなければ主人公、それもいなければ -1。
func (p *autoPlayer) anchor(g int) int {
	best, lord := -1, -1
	units := p.b.Units()
	for j, o := range units {
		if !o.Alive || o.Side != tactics.Player || p.group(j) != g {
			continue
		}
		if o.Lord {
			lord = j
			continue
		}
		if o.Weapon.Type == tactics.Staff {
			continue
		}
		if best < 0 || o.Stats.Def > units[best].Stats.Def {
			best = j
		}
	}
	if best < 0 {
		return lord
	}
	return best
}

// act は自軍 i を 1 回行動させる。回復 → 攻撃 → 薬 → 移動の順に、よい手を探す。
func (p *autoPlayer) act(i int) {
	units := p.b.Units()
	u := units[i]
	reach := p.b.Reachable(i)
	tiles := sortedTiles(reach)
	er := p.enemyReach()
	cost := p.pathCost(p.goal(i))

	if u.Weapon.Type == tactics.Staff {
		if p.tryHeal(i, tiles, er) {
			return
		}
	}

	// 攻撃の候補。
	type option struct {
		cmd   tactics.Command
		score float64
	}
	var best, risky *option
	if u.Weapon.Type != tactics.Staff {
		for _, t := range tiles {
			for e, o := range units {
				if !o.Alive || o.Side != tactics.Enemy {
					continue
				}
				d := dist(t, o.Pos)
				if d < u.Weapon.MinRange || d > u.Weapon.MaxRange {
					continue
				}
				cmd := tactics.Command{Unit: i, To: t, Action: tactics.Attack, Target: e}
				f, err := p.b.Forecast(cmd)
				if err != nil {
					continue
				}
				a, df := f.Attacker, f.Defender
				h := float64(a.HitRate) / 100
				var kill float64
				switch {
				case a.Damage >= o.HP:
					kill = 1 - math.Pow(1-h, float64(a.Count))
				case a.Damage*a.Count >= o.HP:
					kill = math.Pow(h, float64(a.Count))
				}
				counter := df.Damage * df.Count
				if df.HitRate == 0 {
					counter = 0
				}
				skip := -1
				if kill >= 0.7 {
					skip = e
				}
				dg := p.danger(i, t, skip, er)
				if counter >= u.HP {
					continue
				}
				score := kill*100 + float64(a.Damage*a.Count)*h*2 - float64(counter)*float64(df.HitRate)/100 - float64(dg)*0.5
				if o.Boss {
					score += 50 + kill*1000
				}
				if dg > p.limit(u, u.HP-counter) && !(o.Boss && kill >= 0.6) {
					if risky == nil || score > risky.score {
						risky = &option{cmd, score}
					}
					continue
				}
				if best == nil || score > best.score {
					best = &option{cmd, score}
				}
			}
		}
	}
	if best != nil {
		if err := p.do(best.cmd, fmt.Sprintf("攻撃 %.0f", best.score)); err == nil {
			return
		}
	}

	// 薬: HP が 6 割を切っていれば、受けるダメージの少ないマスで使う。
	if u.HP*10 < u.Stats.HP*6 {
		for k, it := range u.Items {
			if it.Count <= 0 {
				continue
			}
			to, _ := p.safest(i, tiles, er, cost, false)
			if err := p.do(tactics.Command{Unit: i, To: to, Action: tactics.UseItem, Item: k}, "薬"); err == nil {
				return
			}
		}
	}

	// 移動。HP が 4 割を切っていれば下がる。安全なマスがなく、攻撃できるなら戦う（主人公は除く）。
	// 隊に回復役がいなければ、下がっても治らないので進む。
	progress := u.HP*10 >= u.Stats.HP*4 || !p.hasHealer(p.group(i), i)
	to, safe := p.safest(i, tiles, er, cost, progress)
	if !safe && risky != nil && !u.Lord {
		if err := p.do(risky.cmd, fmt.Sprintf("やむなく攻撃 %.0f", risky.score)); err == nil {
			return
		}
	}
	p.do(tactics.Command{Unit: i, To: to, Action: tactics.Wait}, fmt.Sprintf("移動 危険%d", p.danger(i, to, -1, er)))
}

// safest は、受けてよいダメージ（limit）に収まるマスのうち、progress なら目標へ近いマス、
// そうでなければ受けるダメージが少なく回復できるマスを返す。収まるマスがなければ、受けるダメージが最も少ないマス。
// 隊の先頭より 4 を超えて先へは出ない。主人公は隊の先頭より前に出ない。回復役は仲間から離れすぎない。
func (p *autoPlayer) safest(i int, tiles []tactics.Pos, er map[int]map[tactics.Pos]int, cost map[tactics.Pos]int, progress bool) (tactics.Pos, bool) {
	u := p.b.Units()[i]
	// 盾役は先頭を行き、ほかの仲間は盾役より 1 を超えて先へ出ない。主人公は盾役より前に出ない。
	front, lead := math.MinInt/4, 0
	if a := p.anchor(p.group(i)); a >= 0 && a != i {
		if c, ok := cost[p.b.Units()[a].Pos]; ok {
			front, lead = c, 1
			if u.Lord {
				lead = -1
			}
		}
	}
	type cand struct {
		t     tactics.Pos
		dg, c int
	}
	var best, fallback *cand
	for _, t := range tiles {
		c, ok := cost[t]
		if !ok {
			c = math.MaxInt / 4
		}
		dg := p.danger(i, t, -1, er)
		// 安全なマスがないときは、敵の動き方のとおりの見込みでいちばん少ないマスへ。
		real := p.dangerOf(i, t, -1, er, false)
		if fallback == nil || real < fallback.dg || real == fallback.dg && c < fallback.c {
			fallback = &cand{t, real, c}
		}
		if dg > p.limit(u, u.HP) {
			continue
		}
		score := c
		if c < front-lead {
			score = front - lead + 3*(front-lead-c)
		}
		if !progress {
			score = dg*10 - p.b.TerrainAt(t).HealPercent + c/8
		}
		if u.Weapon.Type == tactics.Staff && progress {
			score += 2 * p.behind(i, t)
		}
		if best == nil || score < best.c || score == best.c && dg < best.dg {
			best = &cand{t, dg, score}
		}
	}
	if best != nil {
		return best.t, true
	}
	return fallback.t, false
}

// hasHealer は、隊 g に i 以外の回復役（使用回数の残る杖）がいるかを返す。
func (p *autoPlayer) hasHealer(g, i int) bool {
	for j, o := range p.b.Units() {
		if j != i && o.Alive && o.Side == tactics.Player && p.group(j) == g && o.Weapon.Type == tactics.Staff && o.Weapon.Uses > 0 {
			return true
		}
	}
	return false
}

// behind は、回復役が隊の仲間からどれだけ離れているか（最も近い仲間までの距離のうち 2 を超えた分）。
func (p *autoPlayer) behind(i int, t tactics.Pos) int {
	near := math.MaxInt
	for j, o := range p.b.Units() {
		if j == i || !o.Alive || o.Side != tactics.Player || p.group(j) != p.group(i) {
			continue
		}
		near = min(near, dist(o.Pos, t))
	}
	if near == math.MaxInt {
		return 0
	}
	return max(0, near-2)
}

// tryHeal は、回復役 i が、減った HP がいちばん多い味方を回復できれば回復する。
func (p *autoPlayer) tryHeal(i int, tiles []tactics.Pos, er map[int]map[tactics.Pos]int) bool {
	units := p.b.Units()
	u := units[i]
	type cand struct {
		cmd     tactics.Command
		lost, d int
	}
	var best *cand
	for j, o := range units {
		if j == i || !o.Alive || o.Side != tactics.Player || o.HP >= o.Stats.HP {
			continue
		}
		lost := o.Stats.HP - o.HP
		if lost < 5 && o.HP*10 > o.Stats.HP*7 {
			continue
		}
		for _, t := range tiles {
			if dist(t, o.Pos) < u.Weapon.MinRange || dist(t, o.Pos) > u.Weapon.MaxRange {
				continue
			}
			dg := p.danger(i, t, -1, er)
			if dg > p.limit(u, u.HP) {
				continue
			}
			if best == nil || lost > best.lost || lost == best.lost && dg < best.d {
				best = &cand{tactics.Command{Unit: i, To: t, Action: tactics.Heal, Target: j}, lost, dg}
			}
		}
	}
	if best == nil {
		return false
	}
	return p.do(best.cmd, "回復") == nil
}

// autoCampaignBattle は、キャンペーンの今の章で、名簿の生きている仲間を枠の数まで（名簿の順に）出撃させる。
func autoCampaignBattle(c *core.Campaign, seed uint64, slots int) (*core.Sortie, []string, error) {
	var members []int
	for k, m := range c.Roster {
		if m.Alive && len(members) < slots {
			members = append(members, k)
		}
	}
	s, _, err := c.StartBattle(members, seed)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, len(s.Battle.Units()))
	for k, m := range members {
		ids[k] = c.Roster[m].Character
	}
	return s, ids, nil
}

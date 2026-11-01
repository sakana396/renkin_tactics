package tactics

import (
	"container/heap"
	"sort"
	"strconv"
)

// planEnemyGroups は敵軍の番の始めに一団の集まる先と突入を決める（DES-AI-02）。
// 増援・回復後の盤面を基準にし、行動中は計画を変えない。
func (b *Battle) planEnemyGroups(count int) {
	if b.groupPlan == nil {
		b.groupPlan = make(map[int]Pos)
	}
	b.groupDanger = make(map[int]map[Pos]bool)
	groups := make(map[string][]int)
	for i := 0; i < count; i++ {
		u := b.units[i]
		key := b.groupKey(u.Group, u.Pos)
		if u.Alive && u.Side == Enemy && u.Approach == GroupApproach && !b.groupEntered[key] {
			groups[key] = append(groups[key], i)
		}
	}
	playerReach := make(map[int]map[Pos]int)
	occupied := make(map[Pos]int)
	for i, u := range b.units {
		if u.Alive {
			occupied[u.Pos] = i
		}
		if u.Alive && u.Side == Player {
			playerReach[i] = b.Reachable(i)
		}
	}
	near := b.closeness()
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return groups[keys[i]][0] < groups[keys[j]][0] })
	threats := b.playerThreats(playerReach)
	assigned := make(map[Pos]bool)
	// 先に全一団の生存者の集合先を予約する。処理順が後の一団も例外にしない。
	for i, goal := range b.groupPlan {
		if i < len(b.units) && b.units[i].Alive {
			assigned[goal] = true
		}
	}
	for _, key := range keys {
		members := groups[key]
		// 先が決まった仲間の現在位置は、空く予定のマスとして候補に含める。
		member := make(map[int]bool, len(members))
		for _, i := range members {
			member[i] = true
		}
		active := b.mapState.charged[b.units[members[0]].Group]
		for _, i := range members {
			if b.units[i].AI == Charge {
				active = true
			}
			if _, ok := b.bestCandidate(i, b.Reachable(i)); ok {
				active = true
			}
		}
		if !active {
			continue
		}
		for _, i := range members {
			b.units[i].AI = Charge
		}
		// 集合を始めた一団の割り当ては、危険範囲が消えても保つ。
		planned := false
		for _, i := range members {
			if _, ok := b.groupPlan[i]; ok {
				planned = true
			}
		}
		for _, i := range members {
			b.groupDanger[i] = b.dangerFor(i, playerReach, threats)
		}
		anyDanger := false
		for _, i := range members {
			if len(b.groupDanger[i]) != 0 {
				anyDanger = true
			}
		}
		if !planned && !anyDanger {
			continue
		}

		groupGoals := make(map[Pos]bool)
		for _, i := range members {
			if goal, ok := b.groupPlan[i]; ok {
				groupGoals[goal] = true
			}
		}
		allReady := true
		// 危険範囲の境目を先に決め、危険範囲のない仲間の隣接先を作る。
		for pass := 0; pass < 2; pass++ {
			for _, i := range members {
				if _, ok := b.groupPlan[i]; ok {
					continue
				}
				u := b.units[i]
				danger := b.groupDanger[i]
				if (len(danger) == 0) != (pass == 1) {
					continue
				}
				path := b.terrainDistances(u.Pos, false)
				var goal Pos
				found := false
				bestNear, bestCost := unreachable, unreachable
				for y, row := range b.terrain {
					for x, tile := range row {
						p := Pos{x, y}
						owner, taken := occupied[p]
						_, ownerPlanned := b.groupPlan[owner]
						if tile.Cost < 1 || assigned[p] || danger[p] || (taken && owner != i && (!member[owner] || !ownerPlanned)) {
							continue
						}
						cost, ok := path[p]
						if !ok {
							continue
						}
						boundary := false
						for _, d := range directions {
							if danger[Pos{p.X + d.X, p.Y + d.Y}] {
								boundary = true
								break
							}
						}
						if !boundary || len(danger) == 0 {
							continue
						}
						n := near(p)
						if !found || n < bestNear || (n == bestNear && (cost < bestCost || (cost == bestCost && posLess(p, goal)))) {
							goal, bestNear, bestCost, found = p, n, cost, true
						}
					}
				}
				if !found {
					// 狭い道では、前の列に隣り合う安全なマスを後ろの列へ割り当てる。
					for y, row := range b.terrain {
						for x, tile := range row {
							p := Pos{x, y}
							owner, taken := occupied[p]
							_, ownerPlanned := b.groupPlan[owner]
							if tile.Cost < 1 || assigned[p] || danger[p] || (taken && owner != i && (!member[owner] || !ownerPlanned)) {
								continue
							}
							cost, ok := path[p]
							if !ok {
								continue
							}
							adjacent := false
							for _, d := range directions {
								neighbor := Pos{p.X + d.X, p.Y + d.Y}
								if groupGoals[neighbor] || (len(danger) != 0 && b.groupMemberAt(neighbor, members)) {
									adjacent = true
									break
								}
							}
							if !adjacent {
								continue
							}
							n := near(p)
							if !found || (len(danger) == 0 && n > bestNear) || (len(danger) != 0 && n < bestNear) || (n == bestNear && (cost < bestCost || (cost == bestCost && posLess(p, goal)))) {
								goal, bestNear, bestCost, found = p, n, cost, true
							}
						}
					}
				}
				if !found {
					// 境目を仲間がふさいでいても、逃げられる安全なマスを割り当てる。
					if len(danger) != 0 && danger[u.Pos] {
						for p, cost := range b.Reachable(i) {
							if danger[p] || assigned[p] {
								continue
							}
							n := near(p)
							if !found || n < bestNear || (n == bestNear && (cost < bestCost || (cost == bestCost && posLess(p, goal)))) {
								goal, bestNear, bestCost, found = p, n, cost, true
							}
						}
					}
				}
				if !found {
					allReady = false
					continue
				}
				b.groupPlan[i] = goal
				assigned[goal] = true
				groupGoals[goal] = true
			}
		}
		for _, i := range members {
			goal, ok := b.groupPlan[i]
			if !ok || b.units[i].Pos != goal {
				allReady = false
			}
		}
		if allReady {
			b.groupEntered[key] = true
			for _, i := range members {
				delete(b.groupPlan, i)
			}
		}
	}
}

// groupMemberAt は同じ一団の一員が現在いるマスを調べる（DES-AI-02）。
func (b *Battle) groupMemberAt(p Pos, members []int) bool {
	for _, i := range members {
		if b.units[i].Pos == p {
			return true
		}
	}
	return false
}

// groupKey は一団の到達できる領域ごとの状態の鍵（DES-AI-02）。
func (b *Battle) groupKey(group string, at Pos) string {
	if group == "" {
		return ""
	}
	// 同じ地形の連結領域の左上の座標を代表とする。地形は戦闘中に変わらない。
	if first, ok := b.groupRegions[at]; ok {
		return group + ":" + strconv.Itoa(first.X) + ":" + strconv.Itoa(first.Y)
	}
	seen := map[Pos]bool{at: true}
	queue := []Pos{at}
	first := at
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if posLess(p, first) {
			first = p
		}
		for _, d := range directions {
			next := Pos{p.X + d.X, p.Y + d.Y}
			if seen[next] || !b.inMap(next) || b.TerrainAt(next).Cost < 1 {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
		}
	}
	if b.groupRegions == nil {
		b.groupRegions = make(map[Pos]Pos, len(seen))
	}
	for p := range seen {
		b.groupRegions[p] = first
	}
	return group + ":" + strconv.Itoa(first.X) + ":" + strconv.Itoa(first.Y)
}

// playerThreats は盤面ごとに自軍の移動と武器射程の和を一度だけ作る（DES-AI-02、DES-IT06 10.4 節）。
func (b *Battle) playerThreats(reach map[int]map[Pos]int) map[int]map[Pos]bool {
	out := make(map[int]map[Pos]bool, len(reach))
	for i, places := range reach {
		threat := make(map[Pos]bool)
		for p := range places {
			for _, w := range attackWeapons(b.units[i]) {
				for dy := -w.MaxRange; dy <= w.MaxRange; dy++ {
					for dx := -w.MaxRange; dx <= w.MaxRange; dx++ {
						at := Pos{p.X + dx, p.Y + dy}
						if b.inMap(at) && inRange(w, abs(dx)+abs(dy)) {
							threat[at] = true
						}
					}
				}
			}
		}
		out[i] = threat
	}
	return out
}

// dangerFor は敵ごとの割に合わない自軍の攻撃範囲を返す（DES-AI-02）。
func (b *Battle) dangerFor(enemy int, reach map[int]map[Pos]int, threats map[int]map[Pos]bool) map[Pos]bool {
	out := make(map[Pos]bool)
	target := b.units[enemy]
	// 地形を外した仮の戦闘は敵ごとに一度だけ作る。
	sim := *b
	sim.terrain = append([][]Terrain(nil), b.terrain...)
	sim.terrain[target.Pos.Y] = append([]Terrain(nil), b.terrain[target.Pos.Y]...)
	sim.terrain[target.Pos.Y][target.Pos.X].Def = 0
	sim.terrain[target.Pos.Y][target.Pos.X].Avoid = 0
	sim.units = append([]Unit(nil), b.units...)
	for i := range reach {
		player := b.units[i]
		found := false
		var best candidate
		for wi, w := range player.Weapons {
			if w.Type == Staff {
				continue
			}
			for rangeDistance := w.MinRange; rangeDistance <= w.MaxRange; rangeDistance++ {
				// 現在の到達可否とは独立に、武器の射程から攻撃を仮定する。
				// 守る側の地形補正を外し、仮の位置は戦闘本体に書き込まない。
				to := Pos{target.Pos.X + rangeDistance, target.Pos.Y}
				if sim.inMap(to) {
					sim.terrain[to.Y][to.X].Def = 0
					sim.terrain[to.Y][to.X].Avoid = 0
				}
				sim.units[i].Pos = to
				sim.units[i].Equipped = wi
				attackCount, counterCount := 0, 0
				for _, striker := range sim.strikeOrder(i, enemy) {
					if striker == i {
						attackCount++
					} else {
						counterCount++
					}
				}
				attackDamage, attackHit := sim.strikeValues(i, enemy, false)
				dealt := attackDamage * attackCount
				value := dealt * attackHit
				reply := 0
				if counterCount != 0 {
					counterDamage, counterHit := sim.strikeValues(enemy, i, false)
					reply = counterDamage * counterHit * counterCount
				}
				c := candidate{to: to, target: enemy, weapon: wi, kill: dealt >= target.HP, gain: value, risk: reply}
				if !found || c.better(best) {
					best, found = c, true
				}
			}
		}
		sim.units[i] = player
		if !found || (!best.kill && best.gain < best.risk) {
			continue
		}
		for at := range threats[i] {
			out[at] = true
		}
	}
	return out
}

// terrainDistances は占有を除いた地形の最小費用を返す（DES-AI-02）。
// reverse は各マスから from へ進む費用を逆向きに求める。
func (b *Battle) terrainDistances(from Pos, reverse bool) map[Pos]int {
	dist := map[Pos]int{from: 0}
	q := &posQueue{{pos: from, cost: 0}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(posCost)
		if cur.cost != dist[cur.pos] {
			continue
		}
		for _, d := range directions {
			next := Pos{cur.pos.X + d.X, cur.pos.Y + d.Y}
			if !b.inMap(next) {
				continue
			}
			c := b.TerrainAt(next).Cost
			if c < 1 {
				continue
			}
			if reverse {
				c = b.TerrainAt(cur.pos).Cost
			}
			total := cur.cost + c
			if old, ok := dist[next]; ok && old <= total {
				continue
			}
			dist[next] = total
			heap.Push(q, posCost{pos: next, cost: total})
		}
	}
	return dist
}

// groupMove は集まる先へ、この番に届く安全なマスまで進む（DES-AI-02）。
func (b *Battle) groupMove(unit int, reach map[Pos]int, goal Pos) (Pos, bool) {
	danger := b.groupDanger[unit]
	toGoal := b.terrainDistances(goal, true)
	var best Pos
	bestRemain, bestCost := unreachable, unreachable
	found := false
	for p, cost := range reach {
		if danger[p] {
			continue
		}
		remain, ok := toGoal[p]
		if !ok {
			continue
		}
		if !found || remain < bestRemain || (remain == bestRemain && (cost < bestCost || (cost == bestCost && posLess(p, best)))) {
			best, bestRemain, bestCost, found = p, remain, cost, true
		}
	}
	return best, found
}

func (b *Battle) playerReach() map[int]map[Pos]int {
	out := make(map[int]map[Pos]int)
	for i, u := range b.units {
		if u.Alive && u.Side == Player {
			out[i] = b.Reachable(i)
		}
	}
	return out
}

// enemyRetreat は敵の騎兵が攻撃後に下がる先を四つの優先順で選ぶ（DES-AI-02、DES-BTL-10）。
func (b *Battle) enemyRetreat(unit, remaining int) (Pos, bool) {
	reach := (BoardView{terrain: b.terrain, units: b.units}).reachableWithin(unit, remaining)
	players := b.playerReach()
	var best Pos
	bestThreat, bestDistance, bestCost := unreachable, -1, unreachable
	found := false
	for p, cost := range reach {
		threat := 0
		nearest := unreachable
		for i, places := range players {
			d := distance(p, b.units[i].Pos)
			if d < nearest {
				nearest = d
			}
			attacks := false
			weapons := attackWeapons(b.units[i])
			for from := range places {
				for _, weapon := range weapons {
					if inRange(weapon, distance(from, p)) {
						attacks = true
						break
					}
				}
				if attacks {
					break
				}
			}
			if attacks {
				threat++
			}
		}
		if !found || threat < bestThreat || (threat == bestThreat && (nearest > bestDistance || (nearest == bestDistance && (cost < bestCost || (cost == bestCost && posLess(p, best)))))) {
			best, bestThreat, bestDistance, bestCost, found = p, threat, nearest, cost, true
		}
	}
	return best, found
}

// attackWeapons は手持ちの攻撃用武器を返す（DES-AI-02）。
func attackWeapons(u Unit) []Weapon {
	out := make([]Weapon, 0, len(u.Weapons))
	for _, w := range u.Weapons {
		if w.Type != Staff {
			out = append(out, w)
		}
	}
	return out
}

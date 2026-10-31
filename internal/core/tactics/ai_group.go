package tactics

import (
	"container/heap"
	"strconv"
)

// planEnemyGroups は敵軍の番の始めに一団の集まる先と突入を決める（DES-AI-02）。
// 増援・回復後の盤面を基準にし、行動中は計画を変えない。
func (b *Battle) planEnemyGroups(count int) {
	b.groupPlan = make(map[int]Pos)
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
	for key, members := range groups {
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
		groupDanger := make(map[Pos]bool)
		for _, i := range members {
			for p := range b.dangerFor(i, playerReach) {
				groupDanger[p] = true
			}
		}
		if len(groupDanger) == 0 {
			b.groupEntered[key] = true
			continue
		}

		assigned := make(map[Pos]bool)
		allReady := true
		for _, i := range members {
			u := b.units[i]
			danger := groupDanger
			path := b.terrainDistances(u.Pos, false)
			var goal Pos
			found := false
			bestNear, bestCost := unreachable, unreachable
			for y, row := range b.terrain {
				for x, tile := range row {
					p := Pos{x, y}
					owner, taken := occupied[p]
					if tile.Cost < 1 || assigned[p] || danger[p] || (taken && owner != i) {
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
					if !boundary {
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
						if tile.Cost < 1 || assigned[p] || danger[p] || (taken && owner != i) {
							continue
						}
						cost, ok := path[p]
						if !ok {
							continue
						}
						adjacent := false
						for _, d := range directions {
							if assigned[Pos{p.X + d.X, p.Y + d.Y}] {
								adjacent = true
								break
							}
						}
						if !adjacent {
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
			if u.Pos != goal {
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

// groupKey は一団の到達できる領域ごとの状態の鍵（DES-AI-02）。
func (b *Battle) groupKey(group string, at Pos) string {
	if group == "" {
		return ""
	}
	// 同じ地形の連結領域の左上の座標を代表とする。
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
	return group + ":" + strconv.Itoa(first.X) + ":" + strconv.Itoa(first.Y)
}

// dangerFor は、その敵に割に合わない自軍の次の移動と攻撃の範囲を返す（DES-AI-02）。
func (b *Battle) dangerFor(enemy int, reach map[int]map[Pos]int) map[Pos]bool {
	out := make(map[Pos]bool)
	target := b.units[enemy]
	for i, places := range reach {
		player := b.units[i]
		weapons := attackWeapons(player)
		found, unfavorable := false, false
		var best candidate
		bestWeapon := 0
		for p := range places {
			for weaponIndex, weapon := range weapons {
				if !inRange(weapon, distance(p, target.Pos)) {
					continue
				}
				sim := *b
				sim.units = append([]Unit(nil), b.units...)
				sim.units[i].Weapon = weapon
				f := sim.forecast(i, enemy, p)
				dealt := f.Attacker.Damage * f.Attacker.Count
				value := dealt * f.Attacker.HitRate
				reply := 0
				for _, counter := range attackWeapons(target) {
					if !inRange(counter, distance(target.Pos, p)) {
						continue
					}
					sim.units[enemy].Weapon = counter
					answer := sim.forecast(i, enemy, p)
					counterValue := answer.Defender.Damage * answer.Defender.HitRate * answer.Defender.Count
					if counterValue > reply {
						reply = counterValue
					}
				}
				c := candidate{to: p, target: enemy, kill: dealt >= target.HP, gain: value, risk: reply}
				if !found || c.better(best) || (!best.better(c) && weaponIndex < bestWeapon) {
					best, bestWeapon, found = c, weaponIndex, true
					unfavorable = dealt >= target.HP || value >= reply
				}
			}
		}
		if !unfavorable {
			continue
		}
		for p := range places {
			for y := range b.terrain {
				for x := range b.terrain[y] {
					at := Pos{x, y}
					for _, weapon := range weapons {
						if inRange(weapon, distance(p, at)) {
							out[at] = true
							break
						}
					}
				}
			}
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
	danger := b.dangerFor(unit, b.playerReach())
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

// attackWeapons は手持ちの攻撃用武器を返す（DES-AI-02）。空の列だけは従来の武器を使う。
func attackWeapons(u Unit) []Weapon {
	if len(u.Weapons) == 0 {
		if u.Weapon.Type == Staff {
			return nil
		}
		return []Weapon{u.Weapon}
	}
	out := make([]Weapon, 0, len(u.Weapons))
	for _, w := range u.Weapons {
		if w.Type != Staff {
			out = append(out, w)
		}
	}
	return out
}

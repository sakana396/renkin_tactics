package core_test

import (
	"fmt"
	"math"
	"sort"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 賢い打ち手（#158、E-102・E-107）。
//
// autoPlay（今の打ち手）と同じく、テストだけで使う自軍の打ち手。公開の口（Reachable・Forecast・HealForecast・
// Act・EndPhase・PendingCanto・CantoReachable・History・Rewind）だけを使い、乱数は戦闘の seed だけで決まる（同じ seed・名簿・作戦なら同じ結果）。
// 先の乱数を覗かない（命中・必殺は確率として見積もる。巻き戻しはゲームの決まりの回数だけ使い、戻した状態の続きを引く）。
// ゲームの規則（core・core/tactics の本体）は変えない。
//
// 人のやり方に近づけるため、次の工夫を足す（smartFeatures の欄ごとに外せる。ミュータント検査に使う）。

// smartFeatures は賢い打ち手の工夫。偽にした工夫は使わない（今の打ち手に近い動きに戻る）。
type smartFeatures struct {
	// Weapons は、手持ちの武器から相性・特効のよい武器に持ち替えて攻撃する。
	Weapons bool
	// Threat は、敵の攻撃の届く範囲（危険範囲）と命中・必殺の確率から倒れる確率を見積もり、
	// 倒れない見込みの仲間だけを範囲に入れる（受けられる仲間で 1 体ずつ釣り出す）。偽なら今の打ち手の「最悪のダメージが HP の 85% まで」。
	Threat bool
	// Focus は、全員の攻撃の候補を比べ、倒しきれる相手を先に、倒す見込みの高い順に攻撃する。偽なら固定の行動順で 1 体ずつ選ぶ。
	Focus bool
	// Heal は、回復役を危険範囲の外に置き、減った HP の多い仲間を回復する。
	Heal bool
	// Hold は、突っ込む敵が 2 ターンで届く近さにいるあいだ、進まずに隊の真ん中に固まって待ち受ける。
	Hold bool
	// Rewind は、仲間が倒れたターンを、ゲームの巻き戻し（1 戦闘 10 回まで。DES-RWD-02・04）でそのターンの初めに戻し、
	// 倒れた仲間を倒れたマスに置かず、より用心させてやり直す。同じターンで 2 回やり直しても倒れるなら 1 つ前のターンに戻る。
	// 1 ターンに 3 回まで。残りの 3 回は主人公が倒れて負けたときのために取っておく。乱数は戻した状態の続きで、先を覗かない。
	Rewind bool
}

// smartAll はすべての工夫を使う賢い打ち手。
var smartAll = smartFeatures{Weapons: true, Threat: true, Focus: true, Heal: true, Hold: true, Rewind: true}

// smartPlay は、賢い打ち手で戦闘を maxTurn ターンまで遊ぶ。plan の Routes・Order・Observe は autoPlay と同じ意味。
func smartPlay(b *tactics.Battle, ids []string, plan autoPlan, f smartFeatures, maxTurn int) autoResult {
	p := &smartPlayer{f: f}
	p.autoPlayer = &autoPlayer{b: b, ids: ids, plan: plan, step: map[int]int{}, log: autoLog}
	p.ban = map[int]map[int]map[tactics.Pos]bool{}
	p.caution = map[int]map[int]bool{}
	tries := map[int]int{}
	records := map[int]*smartTurn{}
	for b.Result == tactics.Ongoing && b.Turn <= maxTurn {
		turn := b.Turn
		rec := &smartTurn{step: map[int]int{}, lastKill: p.lastKill, kills: p.kills, deaths: len(p.deaths)}
		for g, n := range p.step {
			rec.step[g] = n
		}
		for i, u := range b.Units() {
			if u.Side == tactics.Player && u.Alive {
				rec.alive = append(rec.alive, i)
			}
		}
		records[turn] = rec
		p.firstID, p.hasFirst, p.endPos = 0, false, map[int]tactics.Pos{}
		p.phase()
		rec.firstID, rec.hasFirst, rec.endPos = p.firstID, p.hasFirst, p.endPos
		if !f.Rewind || len(p.deaths) == rec.deaths || b.RewindsLeft() < 1 {
			continue
		}
		// 主人公が倒れて負けたときのために、残りの 3 回は主人公の分に取っておく。
		if b.Result != tactics.Lost && b.RewindsLeft() <= smartRewindReserve {
			continue
		}
		// 倒れた仲間を、倒れたターンの終わりに立っていたマスに置かないようにして、そのターンをやり直す。
		// 同じターンを 2 回やり直しても倒れるなら、1 つ前のターンに戻り、そのターンの終わりのマスも避ける。
		back := turn
		if tries[turn] >= 2 && records[turn-1] != nil && records[turn-1].hasFirst && tries[turn-1] < 2 {
			back = turn - 1
		}
		target := records[back]
		if !target.hasFirst || tries[back] >= 3 {
			continue
		}
		units := b.Units()
		for _, i := range rec.alive {
			if units[i].Alive {
				continue
			}
			p.avoid(back, i, target.endPos)
		}
		if err := b.Rewind(target.firstID); err != nil {
			continue
		}
		tries[back]++
		p.step, p.lastKill, p.kills, p.deaths = target.step, target.lastKill, target.kills, p.deaths[:target.deaths]
		for t := range records {
			if t > back {
				delete(records, t)
				if back < turn {
					delete(tries, t)
				}
			}
		}
		p.rewinds++
		if p.log != nil {
			p.log(fmt.Sprintf("T%d で倒れたので T%d に巻き戻す（残り %d 回）", turn, back, b.RewindsLeft()))
		}
	}
	res := autoResult{Result: b.Result, Turns: b.Turn, Deaths: p.deaths, Rewinds: p.rewinds}
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

// smartRewindReserve は、主人公が倒れたときのために取っておく巻き戻しの回数。
var smartRewindReserve = 3

// smartTurn は、巻き戻しのために覚えるターンの初めの打ち手の状態と、そのターンの行動（工夫 Rewind）。
type smartTurn struct {
	step            map[int]int
	lastKill, kills int
	deaths          int
	alive           []int
	firstID         uint64
	hasFirst        bool
	endPos          map[int]tactics.Pos
}

// avoid は、ターン turn をやり直すとき、自軍 i を endPos の i のマスに立たせず、用心させる。
func (p *smartPlayer) avoid(turn, i int, endPos map[int]tactics.Pos) {
	if p.ban[turn] == nil {
		p.ban[turn] = map[int]map[tactics.Pos]bool{}
		p.caution[turn] = map[int]bool{}
	}
	if p.ban[turn][i] == nil {
		p.ban[turn][i] = map[tactics.Pos]bool{}
	}
	if at, ok := endPos[i]; ok {
		p.ban[turn][i][at] = true
	}
	p.caution[turn][i] = true
}

type smartPlayer struct {
	*autoPlayer
	f smartFeatures
	// zone は敵ごとに、武器の列の添字ごとの攻撃の届くマス（行動のたびに作り直す）。
	zone map[int]map[int]map[tactics.Pos]bool
	// firstID はこのターンの最初の自軍の行動の履歴 ID（巻き戻しの戻り先）。endPos は自軍がこのターンに最後に立ったマス。
	firstID  uint64
	hasFirst bool
	endPos   map[int]tactics.Pos
	// ban・caution は、巻き戻したターンごとに、仲間が立たないマスと、用心させる仲間（工夫 Rewind）。
	ban     map[int]map[int]map[tactics.Pos]bool
	caution map[int]map[int]bool
	// rewinds は巻き戻した回数。
	rewinds int
	// engaged は、突っ込む敵が 2 ターンで届く近さにいるか（自軍の番の初めに決める。工夫 Threat）。
	engaged bool
	// center は自軍の番の初めの、生きている自軍の位置の真ん中（待ち受けで固まる先）。
	center tactics.Pos
}

// hit2 は、表示の命中率 r（0〜100）の攻撃が当たる本当の確率（0〜99 の乱数 2 つの和が 2r より小さい。DES-BTL-06）。
var hit2 = func() [101]float64 {
	var out [101]float64
	for r := 0; r <= 100; r++ {
		n := 0
		for a := 0; a < 100; a++ {
			// b は 0〜99 で a+b < 2r となる数。
			n += min(100, max(0, 2*r-a))
		}
		out[r] = float64(n) / 10000
	}
	return out
}()

// blow は 1 撃（ダメージ・当たる確率・必殺の確率）。
type blow struct {
	dmg        int
	hit, crit  float64
	pavise     float64
	multiplier int
}

// lossDist は、HP hp のユニットが blows を順に受けたときの、受けるダメージ（hp で止める）の分布を返す。
func lossDist(hp int, blows []blow) []float64 {
	d := make([]float64, hp+1)
	d[0] = 1
	for _, bl := range blows {
		if bl.hit <= 0 || bl.dmg <= 0 {
			continue
		}
		next := make([]float64, hp+1)
		for taken, pr := range d {
			if pr == 0 {
				continue
			}
			if taken == hp {
				next[hp] += pr
				continue
			}
			next[taken] += pr * (1 - bl.hit)
			for _, c := range []struct {
				p float64
				m int
			}{{1 - bl.crit, 1}, {bl.crit, bl.multiplier}} {
				if c.p <= 0 {
					continue
				}
				full := bl.dmg * c.m
				next[min(hp, taken+full)] += pr * bl.hit * c.p * (1 - bl.pavise)
				if bl.pavise > 0 {
					next[min(hp, taken+full/2)] += pr * bl.hit * c.p * bl.pavise
				}
			}
		}
		d = next
	}
	return d
}

// risk は blows を受けたときの倒れる確率と、受けるダメージの見込み（期待値）。
func risk(hp int, blows []blow) (death, expected float64) {
	d := lossDist(hp, blows)
	for taken, pr := range d {
		expected += float64(taken) * pr
	}
	return d[hp], expected
}

func hasSkill(u tactics.Unit, e tactics.SkillEffect) (tactics.Skill, bool) {
	if u.Skill.Effect == e {
		return u.Skill, true
	}
	for _, s := range u.ClassSkills {
		if s.Effect == e {
			return s, true
		}
	}
	return tactics.Skill{}, false
}

func speedWith(u tactics.Unit, w tactics.Weapon) int {
	return max(0, u.Stats.Spd-max(0, w.Weight-u.Stats.Str))
}

func effectiveOn(w tactics.Weapon, tags []string) bool {
	for _, a := range w.EffectiveTags {
		for _, d := range tags {
			if a == d {
				return true
			}
		}
	}
	return false
}

// enemyBlows は、敵 o が武器 w で、uw を装備して at に立つ自軍 u を攻撃するときの 1 撃と回数（DES-BTL-05 の式を写した見積もり）。
// 敵の鼓舞は敵の今の位置で数え、自軍の鼓舞は数えない（受ける側を重く見る）。
func (p *smartPlayer) enemyBlows(e int, w tactics.Weapon, u tactics.Unit, at tactics.Pos, uw tactics.Weapon) (blow, int, bool) {
	units := p.b.Units()
	o := units[e]
	tr := p.b.TerrainAt(at)
	t := tri(w.Type, uw.Type)
	might := w.Might
	if effectiveOn(w, u.Tags) {
		might *= 3
	}
	power, def := o.Stats.Str, u.Stats.Def
	if w.DamageType == tactics.Magical {
		power, def = o.Stats.Mag, u.Stats.Res
	}
	dmg := max(0, power+might+t-def-tr.Def)
	insp := 0
	for j, a := range units {
		if j == e || !a.Alive || a.Side != tactics.Enemy {
			continue
		}
		if sk, ok := hasSkill(a, tactics.Inspire); ok && dist(a.Pos, o.Pos) <= sk.Range {
			insp += sk.Hit
		}
	}
	rate := min(100, max(0, o.Stats.Skl*2+w.Hit+15*t+insp-(speedWith(u, uw)*2+tr.Avoid)))
	crit := o.Stats.Skl/2 + w.Crit - u.Stats.Lck
	if sk, ok := hasSkill(o, tactics.Wrath); ok && o.HP <= o.Stats.HP/2 {
		crit += sk.Chance
	}
	crit = min(100, max(0, crit))
	count := max(1, w.Strikes)
	if speedWith(o, w)-speedWith(u, uw) >= 4 {
		count *= 2
	}
	bl := blow{dmg: dmg, hit: hit2[rate], crit: float64(crit) / 100, multiplier: 3}
	if sk, ok := hasSkill(u, tactics.Pavise); ok {
		bl.pavise = float64(sk.Chance) / 100
	}
	return bl, count, rate > 0
}

// refresh は敵の届く範囲を作り直す。行動済みの自軍だけを壁とみなす（autoPlayer.enemyReach と同じ）。
func (p *smartPlayer) refresh() {
	er := p.enemyReach()
	units := p.b.Units()
	w, h := p.b.Width(), p.b.Height()
	p.zone = map[int]map[int]map[tactics.Pos]bool{}
	for e, tiles := range er {
		byWeapon := map[int]map[tactics.Pos]bool{}
		for k, wp := range units[e].Weapons {
			if wp.Type == tactics.Staff {
				continue
			}
			z := map[tactics.Pos]bool{}
			for t := range tiles {
				for dy := -wp.MaxRange; dy <= wp.MaxRange; dy++ {
					for dx := -wp.MaxRange; dx <= wp.MaxRange; dx++ {
						d := abs(dx) + abs(dy)
						q := tactics.Pos{X: t.X + dx, Y: t.Y + dy}
						if d < wp.MinRange || d > wp.MaxRange || q.X < 0 || q.Y < 0 || q.X >= w || q.Y >= h {
							continue
						}
						z[q] = true
					}
				}
			}
			byWeapon[k] = z
		}
		p.zone[e] = byWeapon
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// incoming は、自軍 i が uw を装備して at に立ったとき、次の敵軍の番に i が受ける 1 撃の列を返す。
// 敵の動き方（DES-AI-01）にならい、各敵は攻撃できる相手（行動済みの自軍と、at に立つ i）のうち、倒せる相手、
// 次にダメージ × 命中率 × 回数の大きい相手を、武器を持ち替えて選ぶとする。skip の敵は数えない（この行動で倒す見込み）。
// 主人公は、攻撃の届く敵がすべて主人公を狙うとみなす（倒れたら負けのため）。attackers は i を狙う敵の数。
func (p *smartPlayer) incoming(i int, at tactics.Pos, uw tactics.Weapon, skip int) (blows []blow, attackers int) {
	units := p.b.Units()
	type target struct {
		k  int
		at tactics.Pos
		w  tactics.Weapon
	}
	targets := []target{{i, at, uw}}
	for k, o := range units {
		if k != i && o.Alive && o.Side == tactics.Player && o.Acted {
			targets = append(targets, target{k, o.Pos, o.EquippedWeapon()})
		}
	}
	hp := map[int]int{}
	for _, tg := range targets {
		hp[tg.k] = units[tg.k].HP
	}
	keys := make([]int, 0, len(p.zone))
	for e := range p.zone {
		keys = append(keys, e)
	}
	sort.Ints(keys)
	for _, e := range keys {
		if e == skip {
			continue
		}
		bestK, bestKill, bestGain := -1, false, -1.0
		var bestBlow, mineBlow blow
		var bestCount, mineCount int
		mineGain := -1.0
		for _, tg := range targets {
			if hp[tg.k] <= 0 {
				continue
			}
			for k, z := range p.zone[e] {
				if !z[tg.at] {
					continue
				}
				bl, count, ok := p.enemyBlows(e, units[e].Weapons[k], units[tg.k], tg.at, tg.w)
				if !ok {
					continue
				}
				kill := bl.dmg*count >= hp[tg.k]
				gain := float64(bl.dmg*count) * bl.hit
				if bestK < 0 || kill && !bestKill || kill == bestKill && gain > bestGain {
					bestK, bestKill, bestGain, bestBlow, bestCount = tg.k, kill, gain, bl, count
				}
				if tg.k == i && gain > mineGain {
					mineGain, mineBlow, mineCount = gain, bl, count
				}
			}
		}
		if bestK < 0 {
			continue
		}
		hp[bestK] -= bestBlow.dmg * bestCount
		switch {
		case bestK == i:
			attackers++
			for range bestCount {
				blows = append(blows, bestBlow)
			}
		case units[i].Lord && mineCount > 0:
			attackers++
			for range mineCount {
				blows = append(blows, mineBlow)
			}
		}
	}
	return blows, attackers
}

// riskLimit は、自軍 u が受けてよい倒れる確率。主人公は 1%、ほかは 4%。
// 敵を倒せない番が 4 回続くと攻め手が止まっているとみなし、主人公は 3%、ほかは 15% まで受ける。
func (p *smartPlayer) riskLimit(u tactics.Unit) float64 {
	stalled := p.stalled()
	switch {
	case u.Lord && stalled:
		return 0.03
	case u.Lord:
		return 0.01
	case stalled:
		return 0.15
	}
	return 0.04
}

// stalled は、敵を倒せない番が 4 回続いたか。
func (p *smartPlayer) stalled() bool { return p.b.Turn-p.lastKill >= 4 }

// sustain は、u が見込みで受けてよいダメージ（期待値）。HP が最大の 5 割（主人公は 6 割）を下回らない分まで。
func (p *smartPlayer) sustain(u tactics.Unit, hp int) float64 {
	if u.Lord {
		return float64(hp) - 0.6*float64(u.Stats.HP)
	}
	return float64(hp) - 0.5*float64(u.Stats.HP)
}

// safeFor は、i が at に立つことを工夫 Threat に従って許すかと、その評価（倒れる確率・受けるダメージの見込み・狙う敵の数）。
func (p *smartPlayer) safeFor(i int, at tactics.Pos, uw tactics.Weapon, skip int, pre []blow, hp int) (ok bool, death, expected float64, attackers int) {
	u := p.b.Units()[i]
	blows, attackers := p.incoming(i, at, uw, skip)
	all := append(append([]blow(nil), pre...), blows...)
	death, expected = risk(hp, all)
	if p.banned(i, at) {
		return false, death, expected, attackers
	}
	if !p.f.Threat {
		// 今の打ち手と同じ: 最悪のダメージ（すべて当たる）が limit に収まる。
		worst := 0
		for _, bl := range all {
			if bl.hit > 0 {
				worst += bl.dmg
			}
		}
		return worst <= p.limit(u, hp), death, expected, attackers
	}
	if p.cautious(i) {
		// 巻き戻したターンで倒れた仲間は、倒れる見込みをほぼなくし、HP の 7 割を残す。
		return death <= p.riskLimit(u)/4 && (attackers == 0 || expected <= float64(hp)-0.7*float64(u.Stats.HP)), death, expected, attackers
	}
	// 攻め手が止まっているとき（stalled）は、受けるダメージの見込みを問わず、倒れる確率だけで決める。
	ok = death <= p.riskLimit(u) && (attackers == 0 || p.stalled() || expected <= p.sustain(u, hp))
	return ok, death, expected, attackers
}

type smartOption struct {
	cmd   tactics.Command
	score float64
	why   string
}

// attackOption は自軍 i の攻撃の候補のうち最もよいものを返す。
func (p *smartPlayer) attackOption(i int) *smartOption {
	units := p.b.Units()
	u := units[i]
	var weapons []int
	if p.f.Weapons {
		for k, w := range u.Weapons {
			if w.Type != tactics.Staff && allowed(u, w.Type) {
				weapons = append(weapons, k)
			}
		}
	} else if u.EquippedWeapon().Type != tactics.Staff {
		weapons = []int{u.Equipped}
	}
	if len(weapons) == 0 {
		return nil
	}
	tiles := sortedTiles(p.b.Reachable(i))
	var best *smartOption
	for _, t := range tiles {
		for _, k := range weapons {
			w := u.Weapons[k]
			for e, o := range units {
				if !o.Alive || o.Side != tactics.Enemy {
					continue
				}
				d := dist(t, o.Pos)
				if d < w.MinRange || d > w.MaxRange {
					continue
				}
				kk := k
				cmd := tactics.Command{Unit: i, To: t, Action: tactics.Attack, Target: e, Weapon: &kk}
				if !p.f.Weapons {
					cmd.Weapon = nil
				}
				f, err := p.b.Forecast(cmd)
				if err != nil {
					continue
				}
				a, df := f.Attacker, f.Defender
				mine := make([]blow, a.Count)
				for s := range mine {
					mine[s] = blow{dmg: a.Damage, hit: hit2[a.HitRate], crit: float64(a.CritRate) / 100, multiplier: 3}
				}
				killP, dealt := risk(o.HP, mine)
				var counter []blow
				if df.Count > 0 && df.HitRate > 0 {
					for range df.Count {
						bl := blow{dmg: df.Damage, hit: hit2[df.HitRate], crit: float64(df.CritRate) / 100, multiplier: 3}
						if sk, ok := hasSkill(u, tactics.Pavise); ok {
							bl.pavise = float64(sk.Chance) / 100
						}
						counter = append(counter, bl)
					}
				}
				// 反撃は、こちらの初めの 1 撃で倒せなかったときに来る。
				first := 0.0
				if a.Count > 0 && a.Damage >= o.HP {
					first = hit2[a.HitRate]
				}
				skip := -1
				if killP >= 0.7 {
					skip = e
				}
				// 反撃は、こちらの初めの 1 撃で倒せなかったときだけ来るので、当たる確率に掛ける。
				for c := range counter {
					counter[c].hit *= 1 - first
				}
				counterDeath, afterCounter := risk(u.HP, counter)
				if counterDeath > p.riskLimit(u) && !(o.Boss && killP >= 0.9) {
					continue
				}
				ok, death, expected, attackers := p.safeFor(i, t, w, skip, counter, u.HP)
				score := 100*killP + 40*dealt/float64(o.Stats.HP) - 400*death - 30*expected/float64(u.Stats.HP) - 2*float64(attackers)
				if !p.f.Threat {
					score = killP*100 + dealt*2 - afterCounter - expected*0.5
				}
				if o.Boss {
					score += 50 + 1000*killP
					if killP >= 0.6 {
						ok = ok || death <= p.riskLimit(u)*3
					}
				}
				if !ok {
					continue
				}
				if best == nil || score > best.score {
					best = &smartOption{cmd: cmd, score: score, why: fmt.Sprintf("攻撃 倒す%.0f%% 倒れる%.1f%% 受ける%.1f", killP*100, death*100, expected)}
				}
			}
		}
	}
	return best
}

func allowed(u tactics.Unit, t tactics.WeaponType) bool {
	for _, a := range u.AllowedWeaponTypes {
		if a == t {
			return true
		}
	}
	return len(u.AllowedWeaponTypes) == 0
}

// healOption は回復役 i の回復の候補（減った HP の多い仲間を、危険の少ないマスから）を返す。
func (p *smartPlayer) healOption(i int) *smartOption {
	units := p.b.Units()
	u := units[i]
	staff := -1
	for k, w := range u.Weapons {
		if w.Type == tactics.Staff && w.Uses > 0 && allowed(u, w.Type) {
			staff = k
			break
		}
	}
	if staff < 0 {
		return nil
	}
	tiles := sortedTiles(p.b.Reachable(i))
	var best *smartOption
	for j, o := range units {
		if j == i || !o.Alive || o.Side != tactics.Player || o.HP >= o.Stats.HP {
			continue
		}
		lost := o.Stats.HP - o.HP
		if lost < 5 && o.HP*10 > o.Stats.HP*7 {
			continue
		}
		for _, t := range tiles {
			d := dist(t, o.Pos)
			w := u.Weapons[staff]
			if d < w.MinRange || d > w.MaxRange {
				continue
			}
			k := staff
			cmd := tactics.Command{Unit: i, To: t, Action: tactics.Heal, Target: j, Weapon: &k}
			hf, err := p.b.HealForecast(cmd)
			if err != nil || hf.Amount <= 0 {
				continue
			}
			ok, death, expected, _ := p.safeFor(i, t, u.EquippedWeapon(), -1, nil, u.HP)
			if !ok {
				continue
			}
			score := 20 + 3*float64(hf.Amount) - 400*death - 30*expected/float64(u.Stats.HP)
			if o.Lord {
				score += 10
			}
			if best == nil || score > best.score {
				best = &smartOption{cmd: cmd, score: score, why: fmt.Sprintf("回復 %d", hf.Amount)}
			}
		}
	}
	return best
}

// act は選んだ行動をする。
func (p *smartPlayer) act(o *smartOption) error {
	u := p.b.Units()[o.cmd.Unit]
	turn := p.b.Turn
	events, err := p.b.Act(o.cmd)
	if err == nil {
		p.mark(turn, o.cmd.Unit, o.cmd.To)
	}
	p.note(events)
	if p.log != nil {
		p.log(fmt.Sprintf("T%d %s HP%d %v→%v %v 対象%d %s err=%v", p.b.Turn, u.Name, u.HP, u.Pos, o.cmd.To, o.cmd.Action, o.cmd.Target, o.why, err))
	}
	// 再移動は使わず、その場で終える（今の打ち手と同じ。再移動で下がる工夫は勝ち数が増えなかったので入れていない）。
	if pc, ok := p.b.PendingCanto(); ok && err == nil {
		more, cerr := p.b.Act(tactics.Command{Unit: pc.Unit, To: p.b.Units()[pc.Unit].Pos, Action: tactics.CantoEnd})
		p.note(more)
		err = cerr
	}
	return err
}

// bestTile は自軍 i が移動して待つマスを選ぶ。
// 許されるマスのうち目標へ近いマスへ進む（待ち受けのときは隊の真ん中に固まる）。
// 回復役は仲間から離れすぎない。隊の先頭（目標にいちばん近い仲間）より 2 を超えて先へ出ない。
func (p *smartPlayer) bestTile(i int, tiles []tactics.Pos, hp int) tactics.Pos {
	units := p.b.Units()
	u := units[i]
	cost := p.pathCost(p.goal(i))
	front := math.MaxInt / 4
	for j, o := range units {
		if j != i && o.Alive && o.Side == tactics.Player {
			if c, ok := cost[o.Pos]; ok {
				front = min(front, c)
			}
		}
	}
	type cand struct {
		t     tactics.Pos
		score float64
	}
	var best, fallback *cand
	for _, t := range tiles {
		c, ok := cost[t]
		if !ok {
			c = math.MaxInt / 8
		}
		safe, death, expected, attackers := p.safeFor(i, t, u.EquippedWeapon(), -1, nil, hp)
		fb := 1000*death + expected + float64(c)/16
		if p.banned(i, t) {
			fb += 10000
		}
		if fallback == nil || fb < fallback.score {
			fallback = &cand{t, fb}
		}
		if !safe {
			continue
		}
		score := float64(c)
		if p.engaged {
			// 向かってくる敵が近いあいだは、進まずに隊の真ん中に固まって待ち受ける。
			score = float64(c)/2 + 2*float64(dist(t, p.center))
		}
		if c < front-2 {
			score += float64(3 * (front - 2 - c))
		}
		if p.f.Threat {
			// 仲間から離れない（いちばん近い仲間まで 2 を超えた分）。
			near := math.MaxInt / 4
			for j, o := range units {
				if j != i && o.Alive && o.Side == tactics.Player {
					near = min(near, dist(o.Pos, t))
				}
			}
			if near < math.MaxInt/4 {
				score += float64(4 * max(0, near-2))
			}
		}
		if p.f.Threat && u.Lord && c < front {
			// 主人公は隊の先頭より前に出ない。
			score += float64(3 * (front - c))
		}
		switch {
		case p.f.Threat && p.stalled():
			// 攻め手が止まっているときは、受けられる範囲で前へ出る。
			score += 0.3*expected + 100*death
		case p.f.Threat:
			score += 2*expected + 2*float64(attackers) + 500*death
		}
		if p.f.Heal && u.EquippedWeapon().Type == tactics.Staff {
			// 回復役は危険範囲の外の、仲間の 2 マス後ろにつく。
			score += 2*float64(p.behind(i, t)) + 10*expected
		}
		if best == nil || score < best.score {
			best = &cand{t, score}
		}
	}
	if best != nil {
		return best.t
	}
	return fallback.t
}

// potion は、弱った自軍 i が回復役の届かないとき、危険の少ないマスで薬を使う候補を返す。
func (p *smartPlayer) potion(i int) *smartOption {
	u := p.b.Units()[i]
	if u.HP*10 >= u.Stats.HP*6 {
		return nil
	}
	for k, it := range u.Items {
		if it.Count <= 0 {
			continue
		}
		to := p.bestTile(i, sortedTiles(p.b.Reachable(i)), min(u.Stats.HP, u.HP+it.Item.Heal))
		return &smartOption{cmd: tactics.Command{Unit: i, To: to, Action: tactics.UseItem, Item: k}, why: "薬"}
	}
	return nil
}

// moveOrder は、攻撃しない仲間の動く順。受けに強い仲間（守備 ＋ HP の高い順）を先に置き、ほかの仲間の危険を見積もりやすくする。
func (p *smartPlayer) moveOrder(rest []int) []int {
	units := p.b.Units()
	out := append([]int(nil), rest...)
	sort.SliceStable(out, func(a, c int) bool {
		x, y := units[out[a]], units[out[c]]
		if p.f.Threat && x.Lord != y.Lord {
			// 主人公は最後に、動き終えた仲間の後ろ（仲間が壁になるマス）を選ぶ。
			return y.Lord
		}
		return x.Stats.Def*2+x.HP > y.Stats.Def*2+y.HP
	})
	return out
}

func (p *smartPlayer) phase() {
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

	p.engaged = false
	n := 0
	var sx, sy int
	for _, o := range p.b.Units() {
		if o.Alive && o.Side == tactics.Player {
			sx, sy, n = sx+o.Pos.X, sy+o.Pos.Y, n+1
		}
	}
	if n > 0 {
		p.center = tactics.Pos{X: (sx + n/2) / n, Y: (sy + n/2) / n}
	}
	if p.f.Hold && !p.stalled() {
		units := p.b.Units()
		for _, e := range units {
			if !e.Alive || e.Side != tactics.Enemy || e.AI != tactics.Charge || e.Boss {
				continue
			}
			for _, o := range units {
				if o.Alive && o.Side == tactics.Player && dist(e.Pos, o.Pos) <= 2*e.Stats.Mov+e.EquippedWeapon().MaxRange+1 {
					p.engaged = true
				}
			}
		}
	}
	p.advanceRoutes()
	live := func() bool {
		return p.b.Result == tactics.Ongoing && p.b.Turn == turn && p.b.Phase == tactics.Player
	}
	rest := func() []int {
		var out []int
		for _, i := range p.order() {
			if u := p.b.Units()[i]; u.Alive && !u.Acted {
				out = append(out, i)
			}
		}
		return out
	}
	tried := map[int]bool{}
	for live() {
		r := rest()
		if len(r) == 0 {
			break
		}
		p.refresh()
		var chosen *smartOption
		if p.f.Focus {
			// 全員の攻撃・回復の候補から、いちばんよいものを選ぶ。
			for _, i := range r {
				if tried[i] {
					continue
				}
				for _, o := range []*smartOption{p.healIf(i), p.attackOption(i)} {
					if o != nil && (chosen == nil || o.score > chosen.score) {
						chosen = o
					}
				}
			}
		} else {
			i := r[0]
			chosen = p.healIf(i)
			if chosen == nil {
				chosen = p.attackOption(i)
			}
			if chosen == nil {
				chosen = p.moveOption(i)
			}
		}
		if chosen == nil {
			// 攻撃も回復もしない仲間は、受けに強い順に動く。
			i := p.moveOrder(r)[0]
			chosen = p.moveOption(i)
		}
		if err := p.act(chosen); err != nil {
			tried[chosen.cmd.Unit] = true
			if u := p.b.Units()[chosen.cmd.Unit]; !u.Acted && live() {
				// 失敗したら、その場で待つ。
				p.do(tactics.Command{Unit: chosen.cmd.Unit, To: u.Pos, Action: tactics.Wait}, "待機")
			}
		}
	}
	if live() {
		events, _ := p.b.EndPhase()
		p.note(events)
	}
}

// healIf は、工夫 Heal を使うとき、回復役 i の回復の候補を返す。
func (p *smartPlayer) healIf(i int) *smartOption {
	if !p.f.Heal {
		return nil
	}
	return p.healOption(i)
}

// moveOption は、攻撃も回復もしないときの行動（薬か移動）。
func (p *smartPlayer) moveOption(i int) *smartOption {
	if o := p.potion(i); o != nil {
		return o
	}
	u := p.b.Units()[i]
	to := p.bestTile(i, sortedTiles(p.b.Reachable(i)), u.HP)
	why := "移動"
	if p.log != nil {
		_, death, expected, attackers := p.safeFor(i, to, u.EquippedWeapon(), -1, nil, u.HP)
		why = fmt.Sprintf("移動 倒れる%.1f%% 受ける%.1f 敵%d", death*100, expected, attackers)
	}
	return &smartOption{cmd: tactics.Command{Unit: i, To: to, Action: tactics.Wait}, why: why}
}

// mark は、ターン turn の自軍の行動のあと、最初の行動の履歴 ID と、立ったマスを覚える（工夫 Rewind）。
func (p *smartPlayer) mark(turn, unit int, to tactics.Pos) {
	if p.endPos == nil {
		p.endPos = map[int]tactics.Pos{}
	}
	p.endPos[unit] = to
	if p.hasFirst {
		return
	}
	h := p.b.History()
	for k := len(h) - 1; k >= 0; k-- {
		if h[k].Turn == turn && h[k].Side == tactics.Player && h[k].Kind == tactics.HistoryAction {
			p.firstID, p.hasFirst = h[k].ID, true
			return
		}
	}
}

// banned は、巻き戻したターンで、自軍 i が at に立たないと決めたかを返す。
func (p *smartPlayer) banned(i int, at tactics.Pos) bool {
	return p.ban[p.b.Turn][i][at]
}

// cautious は、巻き戻したターンで、自軍 i を用心させるかを返す。
func (p *smartPlayer) cautious(i int) bool {
	return p.caution[p.b.Turn][i]
}

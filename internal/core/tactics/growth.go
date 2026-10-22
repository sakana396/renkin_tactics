package tactics

// 経験（DES-GRW-01）、レベルアップと成長（DES-GRW-02）、その出来事（DES-GRW-03）。

const (
	// expKill は、相手（敵）が攻撃の流れの中で倒れたときの経験（DES-GRW-01）。
	expKill = 30
	// expHit は、自分の 1 撃が 1 回以上当たったときの経験（DES-GRW-01）。
	expHit = 10
	// expOther は、それ以外（すべて外れた、1 撃も行わなかった）のときの経験（DES-GRW-01）。
	expOther = 1
	// expPerLevel は、レベルを 1 上げるのに要る経験（DES-GRW-02）。
	expPerLevel = 100
	// maxLevel はレベルの上限（DES-GRW-02）。
	maxLevel = 20
)

// attackExp は、a と d の攻撃の流れが終わったとき、加わった自軍のユニットに経験を足す（DES-GRW-01）。
// playerHit は、自軍のユニットの 1 撃が 1 回以上当たったか（ダメージが 0 でも当たりに数える）。
// 自軍のユニットが倒れていれば経験を足さない。
func (b *Battle) attackExp(events []Event, a, d int, playerHit bool) []Event {
	p, e := a, d
	if b.units[p].Side != Player {
		p, e = d, a
	}
	if b.units[p].Side != Player || !b.units[p].Alive {
		return events
	}
	amount := expOther
	switch {
	case !b.units[e].Alive:
		amount = expKill
	case playerHit:
		amount = expHit
	}
	return b.gainExp(events, p, amount)
}

// gainExp は、ユニット i に経験 amount を足し、100 以上ならレベルを上げて成長させる（DES-GRW-01、DES-GRW-02）。
//
// 経験を得るのは、Level が 1〜19 の自軍のユニットだけ。それ以外は何もしない。
// 経験・レベル・能力・今の HP を更新してから Exp を出し、レベルが上がったら LevelUp を出す（DES-GRW-03）。
// 1 回に足す経験は 30 以下で、足す前の経験は 99 以下なので、1 回に上がるのは 1 レベルまで。
func (b *Battle) gainExp(events []Event, i, amount int) []Event {
	u := &b.units[i]
	if u.Side != Player || u.Level < 1 || u.Level >= maxLevel {
		return events
	}
	u.Exp += amount
	up := u.Exp >= expPerLevel
	var gains Stats
	if up {
		u.Level++
		u.Exp -= expPerLevel // 余りは持ち越す。
		if u.Level == maxLevel {
			u.Exp = 0 // 上限に届いたら余りは捨てる。
		}
		gains = b.rollGrowth(u.Growth)
		u.Stats.HP += gains.HP
		u.Stats.Str += gains.Str
		u.Stats.Skl += gains.Skl
		u.Stats.Spd += gains.Spd
		u.Stats.Def += gains.Def
		u.HP += gains.HP // 最大 HP が上がったら、今の HP も 1 上げる。
	}
	events = b.emit(events, Event{Kind: EventExp, Unit: i, Amount: amount})
	if up {
		events = b.emit(events, Event{Kind: EventLevelUp, Unit: i, Level: b.units[i].Level, Gains: gains})
	}
	return events
}

// rollGrowth は、HP・力・技・速さ・守備の順に 0〜99 の乱数を 1 つずつ（計 5 つ）引き、
// 成長率より小さい能力を 1 上げる分を返す（DES-GRW-02、DES-RNG-01）。移動は上がらない。引き直さない。
func (b *Battle) rollGrowth(g Growth) Stats {
	up := func(rate int) int {
		if b.roll() < rate {
			return 1
		}
		return 0
	}
	var s Stats
	s.HP = up(g.HP)
	s.Str = up(g.Str)
	s.Skl = up(g.Skl)
	s.Spd = up(g.Spd)
	s.Def = up(g.Def)
	return s
}

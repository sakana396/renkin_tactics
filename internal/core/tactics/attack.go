package tactics

// 攻撃の計算（DES-BTL-05）、攻撃の流れ（DES-BTL-06）、スキル（DES-SKL-01）、予測（DES-BTL-07）。

// doubleSpeed は追撃に要る速さの差（DES-BTL-05）。
const doubleSpeed = 4

// criticalMultiplier は通常必殺の倍率（DES-BTL-09）。
const criticalMultiplier = 3

// triangle は、攻撃する側の武器 a と守る側の武器 d の相性 t を返す（DES-BTL-05）。
// 剣は斧に、斧は槍に、槍は剣に有利。有利なら +1、不利なら −1、それ以外（弓を含む）は 0。
func triangle(a, d WeaponType) int {
	beats := func(x, y WeaponType) bool {
		return (x == Sword && y == Axe) || (x == Axe && y == Lance) || (x == Lance && y == Sword)
	}
	switch {
	case beats(a, d):
		return 1
	case beats(d, a):
		return -1
	}
	return 0
}

// inRange は、距離 d が武器 w の射程（最小〜最大）に入っているかを返す。
func inRange(w Weapon, d int) bool {
	return d >= w.MinRange && d <= w.MaxRange
}

// inspired は、ユニット i が鼓舞で受ける命中・回避の補正と、効いている鼓舞のスキルを返す（DES-SKL-01）。
// 持ち主から Range 以内（マンハッタン距離）にいる同じ陣営のユニット（持ち主は除く）に効く。
// 距離は今の位置で測る。攻撃する側は、呼ぶ前に移動したあとのマス（To）に置いておく。
func (b *Battle) inspired(i int) (hit, avoid int, skills []Skill) {
	u := b.units[i]
	for j, o := range b.units {
		if j == i || !o.Alive || o.Side != u.Side || o.Skill.Effect != Inspire {
			continue
		}
		if distance(o.Pos, u.Pos) <= o.Skill.Range {
			hit += o.Skill.Hit
			avoid += o.Skill.Avoid
			skills = append(skills, o.Skill)
		}
	}
	return hit, avoid, skills
}

// sniping は、ユニット s の攻撃に狙撃が効くかを返す（DES-SKL-01）。
// still は、s が自分から攻撃していて、移動せずに攻撃したか。反撃では偽を渡す。
func (b *Battle) sniping(s int, still bool) bool {
	return still && b.units[s].Skill.Effect == Snipe
}

// wrathful は、ユニットが怒りの条件（HP が最大 HP の半分（切り捨て）以下）を満たすかを返す（DES-SKL-01）。
func wrathful(u Unit) bool {
	return u.Skill.Effect == Wrath && u.HP <= u.Stats.HP/2
}

// strikeValues は、s が r を 1 撃するときのダメージ（必殺と大盾を入れない）と命中率を返す（DES-BTL-05）。
// 相性・地形・鼓舞を入れ、still が真なら狙撃も入れる（sniping を参照）。
func (b *Battle) strikeValues(s, r int, still bool) (damage, rate int) {
	su, ru := b.units[s], b.units[r]
	t := triangle(su.Weapon.Type, ru.Weapon.Type)
	tr := b.TerrainAt(ru.Pos)

	might := su.Weapon.Might
	if effective(su.Weapon, ru.Tags) {
		might *= 3
	}
	power, defense := su.Stats.Str, ru.Stats.Def
	if su.Weapon.DamageType == Magical {
		power, defense = su.Stats.Mag, ru.Stats.Res
	}
	damage = max(0, power+might+t-defense-tr.Def)

	sHit, _, _ := b.inspired(s)
	_, rAvoid, _ := b.inspired(r)
	hit := su.Stats.Skl*2 + su.Weapon.Hit + 15*t + sHit
	if b.sniping(s, still) {
		hit += su.Skill.Hit
	}
	avoid := attackSpeed(ru)*2 + tr.Avoid + rAvoid
	rate = min(100, max(0, hit-avoid))
	return damage, rate
}

// canCounter は、守る側 d が攻撃する側 a に反撃できるか（d の武器の射程に a がいるか）を返す（DES-BTL-06）。
// 杖のユニットは反撃しない（射程に a がいても偽。DES-HEL-01）。
func (b *Battle) canCounter(a, d int) bool {
	w := b.units[d].Weapon
	return w.Type != Staff && inRange(w, distance(b.units[a].Pos, b.units[d].Pos))
}

// strikeOrder は、a が d を攻撃するときに 1 撃を行うユニットの順を返す（DES-BTL-06）。
// 1 撃目 → 反撃 → 追撃。途中で倒れることは考えない（倒れたら呼ぶ側で止める）。
func (b *Battle) strikeOrder(a, d int) []int {
	order := []int{a}
	counter := b.canCounter(a, d)
	if counter {
		order = append(order, d)
	}
	diff := attackSpeed(b.units[a]) - attackSpeed(b.units[d])
	switch {
	case diff >= doubleSpeed:
		order = append(order, a)
	case -diff >= doubleSpeed && counter:
		order = append(order, d)
	}
	expanded := make([]int, 0, 5)
	for _, s := range order {
		n := 1
		if s == a {
			n = max(1, b.units[a].Weapon.Strikes)
		}
		for range n {
			expanded = append(expanded, s)
		}
	}
	return expanded
}

// attack は、a が d を攻撃する流れを進め、出来事を足す（DES-BTL-06）。
// a は呼ぶ前に移動を終えておく。still は a が移動せずに攻撃したか（狙撃。DES-SKL-01）。
// どちらかが倒れたら、そこで攻撃を終える。
// 流れの終わりに、加わった自軍のユニットに経験を足し（DES-GRW-01、DES-GRW-02）、そのあとで勝ち負けを確かめる。
func (b *Battle) attack(events []Event, a, d int, still bool) []Event {
	playerHit := false
	for _, s := range b.strikeOrder(a, d) {
		r := d
		if s == d {
			r = a
		}
		var hit bool
		events, hit = b.strike(events, s, r, s == a && still)
		if hit && b.units[s].Side == Player {
			playerHit = true
		}
		if !b.units[r].Alive {
			break
		}
	}
	events = b.attackExp(events, a, d, playerHit)
	return b.judge(events)
}

// strike は、s が r に 1 撃を行う（DES-BTL-06）。
//
// 乱数を引く順番（DES-RNG-01）: 乱数を 2 つ引き、和が 命中率 × 2 より小さければ当たる。
// 当たれば必殺の乱数を 1 つ引き、そのあと r が大盾を持っていれば大盾の乱数を引く。
// 必殺は 3 倍、大盾はそのあと半分にして切り捨てる（DES-BTL-09）。
// HP は 0 で止め、0 になったら倒れる。当たったかを合わせて返す。
func (b *Battle) strike(events []Event, s, r int, still bool) ([]Event, bool) {
	damage, rate := b.strikeValues(s, r, still)
	r1 := b.roll()
	r2 := b.roll()
	e := Event{Kind: EventStrike, Unit: s, Target: r, Hit: r1+r2 < rate*2}
	if e.Hit {
		ru := &b.units[r]
		e.Crit = b.roll() < critRate(b.units[s], *ru)
		if e.Crit {
			damage *= criticalMultiplier
		}
		if ru.Skill.Effect == Pavise && b.roll() < ru.Skill.Chance {
			e.Pavise = true
			damage /= 2
		}
		e.Amount = damage
		ru.HP = max(0, ru.HP-damage)
	}
	events = b.emit(events, e)
	if b.units[r].HP == 0 {
		events = b.fall(events, r)
	}
	return events, e.Hit
}

// fall はユニット i を倒れた状態にし、マップから外す（DES-BTL-06、E-15）。
// 倒れたユニットは Alive が偽になり、移動・攻撃の対象・鼓舞から外れる（添字は変わらない）。
func (b *Battle) fall(events []Event, i int) []Event {
	b.units[i].HP = 0
	b.units[i].Alive = false
	return b.emit(events, Event{Kind: EventFall, Unit: i})
}

// judge は、攻撃が終わるたびに勝ち負けを確かめる（DES-MAP-04、FR-MAP-05、FR-CMP-09）。
// ボスが倒れていれば勝ち（Won を出す）、主人公が倒れていれば負け（Lost を出す）。
// 1 回の攻撃で倒れるのは 1 体だけなので、両方が同時に起きることはない。
func (b *Battle) judge(events []Event) []Event {
	if b.Result != Ongoing {
		return events
	}
	for _, u := range b.units {
		switch {
		case u.Boss && !u.Alive:
			b.Result = Won
			return b.emit(events, Event{Kind: EventWon})
		case u.Lord && !u.Alive:
			b.Result = Lost
			return b.emit(events, Event{Kind: EventLost})
		}
	}
	return events
}

// validateAttack は Attack の検証 6 を確かめる（DES-BTL-04）。状態は変えない。
// 武器が杖でなく（ErrCannotAttack。DES-HEL-01）、Target が生きている相手の陣営のユニットで（ErrNoTarget）、
// To から武器の射程に入っている（ErrOutOfRange）。
func (b *Battle) validateAttack(cmd Command) error {
	if b.units[cmd.Unit].Weapon.Type == Staff {
		return ErrCannotAttack
	}
	if !b.validUnit(cmd.Target) {
		return ErrNoTarget
	}
	t := b.units[cmd.Target]
	if !t.Alive || t.Side == b.units[cmd.Unit].Side {
		return ErrNoTarget
	}
	if !inRange(b.units[cmd.Unit].Weapon, distance(cmd.To, t.Pos)) {
		return ErrOutOfRange
	}
	return nil
}

// Forecast は攻撃の予測（DES-BTL-07）。
type Forecast struct {
	Attacker ForecastSide
	Defender ForecastSide
}

// ForecastSide は、予測の片方の側（DES-BTL-07）。
type ForecastSide struct {
	// Name と HP は、名前と今の HP。
	Name string
	HP   int
	// Damage は 1 撃あたりのダメージ（必殺と大盾を入れない）。
	Damage int
	// HitRate は命中率（鼓舞・狙撃を入れる）。
	HitRate int
	// Count は攻撃の回数（攻撃側は最大 4、反撃側は最大 2）。
	Count int
	// 必殺率・攻撃の速さ・特効（DES-BTL-09）。
	CritRate    int
	AttackSpeed int
	Effective   bool
	// WeightLoss は武器の重さで落ちた速さ（速さ − 攻撃の速さ）。EffectiveMight は特効が効くときの威力（3 倍）で、
	// 効かなければ 0（DES-IT06 12.2 節、UC-03）。
	WeightLoss     int
	EffectiveMight int
	// Strikes は攻撃する側の 1 機会の回数（DES-BTL-09）。
	Strikes int
	// Skills は効くスキル（鼓舞は持ち主のスキル、ほかは自分のスキル）。
	// 鼓舞（味方の添字の順）、狙撃、怒り（HP の条件付き）、大盾（相手の攻撃の回数が 1 以上のとき）の順。
	Skills []Skill
}

// Forecast は、Action が Attack のコマンドについて、状態を変えずに攻撃の予測を返す（DES-BTL-07、FR-BTL-08）。
// Action が Attack でなければ ErrInvalidAction を返す。ほかの検証は Act と同じ（DES-BTL-04）。
func (b *Battle) Forecast(cmd Command) (Forecast, error) {
	if cmd.Action != Attack {
		return Forecast{}, ErrInvalidAction
	}
	if err := b.validateCommand(cmd); err != nil {
		return Forecast{}, err
	}
	if err := b.validateAttack(cmd); err != nil {
		return Forecast{}, err
	}
	return b.forecast(cmd.Unit, cmd.Target, cmd.To), nil
}

// forecast は、a が to に移動して d を攻撃するときの予測を返す（DES-BTL-07）。検証はせず、状態も変えない。
// 敵の動き方（DES-AI-01）でも同じ値を使う。
func (b *Battle) forecast(a, d int, to Pos) Forecast {
	// ユニットの列を写した戦闘の上で、a を to に置いて計算する（乱数は引かない）。
	sim := *b
	sim.units = append([]Unit(nil), b.units...)
	still := sim.units[a].Pos == to
	sim.units[a].Pos = to

	var counts [2]int
	for _, s := range sim.strikeOrder(a, d) {
		if s == a {
			counts[0]++
		} else {
			counts[1]++
		}
	}
	return Forecast{
		Attacker: sim.forecastSide(a, d, still, counts[0], counts[1]),
		Defender: sim.forecastSide(d, a, false, counts[1], counts[0]),
	}
}

// forecastSide は、i が相手 o を攻撃する側の予測を作る（DES-BTL-07）。
// count と oppCount は、i と o の攻撃の回数。
func (b *Battle) forecastSide(i, o int, still bool, count, oppCount int) ForecastSide {
	u := b.units[i]
	damage, rate := b.strikeValues(i, o, still)
	_, _, skills := b.inspired(i)
	if b.sniping(i, still) {
		skills = append(skills, u.Skill)
	}
	if u.Skill.Effect == Wrath {
		skills = append(skills, u.Skill)
	}
	if u.Skill.Effect == Pavise && oppCount > 0 {
		skills = append(skills, u.Skill)
	}
	eff := effective(u.Weapon, b.units[o].Tags)
	effMight := 0
	if eff {
		effMight = u.Weapon.Might * 3
	}
	return ForecastSide{
		WeightLoss:     u.Stats.Spd - attackSpeed(u),
		EffectiveMight: effMight,
		Name:           u.Name,
		HP:             u.HP,
		Damage:         damage,
		HitRate:        rate,
		Count:          count,
		CritRate:       critRate(u, b.units[o]),
		AttackSpeed:    attackSpeed(u),
		Effective:      eff,
		Strikes:        max(1, u.Weapon.Strikes),
		Skills:         skills,
	}
}

// attackSpeed は武器の重さを力で支えた攻撃の速さ（DES-BTL-09）。
func attackSpeed(u Unit) int { return max(0, u.Stats.Spd-max(0, u.Weapon.Weight-u.Stats.Str)) }

// effective は一致する分類が複数あっても 1 回だけ特効を適用する（DES-BTL-09）。
func effective(w Weapon, tags []string) bool {
	for _, a := range w.EffectiveTags {
		for _, d := range tags {
			if a == d {
				return true
			}
		}
	}
	return false
}

// critRate は怒りを足してから上限と下限を適用する（DES-BTL-09）。
func critRate(a, d Unit) int {
	bonus := 0
	if wrathful(a) {
		bonus = a.Skill.Chance
	}
	return min(100, max(0, a.Stats.Skl/2+a.Weapon.Crit+bonus-d.Stats.Lck))
}

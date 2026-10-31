package tactics

// 回復の行動（DES-HEL-01）と回復の予測（DES-HEL-02）。

// validateHeal は Heal の検証 6a〜6e を順に確かめる（DES-HEL-01）。状態は変えない。
// 6a. 武器が杖である（ErrCannotHeal）。6b. 杖の Uses が 1 以上（ErrNoUses）。
// 6c. Target が生きている同じ陣営のユニットで、自分でない（ErrNoTarget）。
// 6d. To から Target までの距離が杖の射程に入っている（ErrOutOfRange）。6e. Target の HP が最大でない（ErrFullHP）。
func (b *Battle) validateHeal(cmd Command) error {
	u := b.units[cmd.Unit]
	if u.Weapon.Type != Staff {
		return ErrCannotHeal
	}
	if u.Weapon.Uses < 1 {
		return ErrNoUses
	}
	if !b.validUnit(cmd.Target) || cmd.Target == cmd.Unit {
		return ErrNoTarget
	}
	t := b.units[cmd.Target]
	if !t.Alive || t.Side != u.Side {
		return ErrNoTarget
	}
	if !inRange(u.Weapon, distance(cmd.To, t.Pos)) {
		return ErrOutOfRange
	}
	if t.HP >= t.Stats.HP {
		return ErrFullHP
	}
	return nil
}

// healWithStaff は、杖のユニット i が target を回復する（DES-HEL-01）。i は呼ぶ前に移動を終えておく。
// 杖の Uses を 1 減らしてから、杖の Heal ＋ 魔力を兵種の上限と不足 HP で抑えて回復し、Heal を出す。
// そのあと、回復した杖のユニットに経験を足す（DES-GRW-01）。命中の乱数は引かない。勝ち負けは決まらない。
func (b *Battle) healWithStaff(events []Event, i, target int) []Event {
	w := &b.units[i].Weapon
	w.Uses--
	events = b.heal(events, target, staffHealAmount(b.units[i]), w.Name)
	return b.gainExp(events, i, expHit)
}

// staffHealAmount は杖の回復量に魔力と兵種の上限を適用する（DES-HEL-01）。
func staffHealAmount(u Unit) int {
	amount := u.Weapon.Heal + u.Stats.Mag
	if u.StaffHealCap > 0 {
		amount = min(amount, u.StaffHealCap)
	}
	return amount
}

// HealForecast は回復の予測（DES-HEL-02）。
type HealForecast struct {
	// Healer と Target は、回復する側と回復される側の名前。
	Healer string
	Target string
	// HP と MaxHP は、回復される側の今の HP と最大 HP。
	HP    int
	MaxHP int
	// Amount は実際に回復する量（min(杖の Heal ＋ 魔力、兵種の上限（あれば）、MaxHP − HP)）。After は回復したあとの HP。
	Amount int
	After  int
	// Uses と MaxUses は、杖の今の残りの使用回数と最大（回復したあとは Uses − 1 になる）。
	Uses    int
	MaxUses int
}

// HealForecast は、Action が Heal のコマンドについて、状態を変えずに回復の予測を返す（DES-HEL-02、FR-BTL-08）。
// Action が Heal でなければ ErrInvalidAction を返す。ほかの検証は Act と同じ（DES-BTL-04、DES-HEL-01）。
func (b *Battle) HealForecast(cmd Command) (HealForecast, error) {
	if cmd.Action != Heal {
		return HealForecast{}, ErrInvalidAction
	}
	if err := b.validateCommand(cmd); err != nil {
		return HealForecast{}, err
	}
	if err := b.validateHeal(cmd); err != nil {
		return HealForecast{}, err
	}
	h, t := b.units[cmd.Unit], b.units[cmd.Target]
	amount := min(staffHealAmount(h), t.Stats.HP-t.HP)
	return HealForecast{
		Healer:  h.Name,
		Target:  t.Name,
		HP:      t.HP,
		MaxHP:   t.Stats.HP,
		Amount:  amount,
		After:   t.HP + amount,
		Uses:    h.Weapon.Uses,
		MaxUses: h.Weapon.MaxUses,
	}, nil
}

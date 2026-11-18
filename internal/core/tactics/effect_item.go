package tactics

import "errors"

// ErrEffectActive は同じ効果が既にあることを表す（DES-NG-14）。
var ErrEffectActive = errors.New("tactics: 同じ効果が既にあります")

// moveBudget は一時効果を含む移動予算を返す（DES-NG-14）。
func (u Unit) moveBudget() int {
	if u.Snared {
		return 0
	}
	return u.Stats.Mov
}

// EffectiveStats は一時効果を含む能力を返す（DES-NG-14）。
func (u Unit) EffectiveStats() Stats {
	s := u.Stats
	s.Str += u.PowerTonic
	s.Mag += u.PowerTonic
	s.Mov = u.moveBudget()
	return s
}

// effectTarget は移動後の対象を探す（DES-NG-14）。
func (b *Battle) effectTarget(cmd Command) int {
	if cmd.ItemTarget == nil {
		return -1
	}
	for i, u := range b.units {
		pos := u.Pos
		if i == cmd.Unit {
			pos = cmd.To
		}
		if u.Alive && pos == *cmd.ItemTarget {
			return i
		}
	}
	return -1
}
func (b *Battle) validateEffectItem(cmd Command, it Item) error {
	if cmd.Blast != nil || cmd.ItemTarget == nil {
		return ErrInvalidAction
	}
	if !b.inMap(*cmd.ItemTarget) {
		return ErrNoTarget
	}
	target := b.effectTarget(cmd)
	if target < 0 {
		return ErrNoTarget
	}
	u := b.units[target]
	if it.Effect == "power_tonic" && u.Side != Player || it.Effect == "snare" && u.Side != Enemy {
		return ErrNoTarget
	}
	d := distance(cmd.To, *cmd.ItemTarget)
	if d > it.Range || it.Effect == "snare" && d == 0 {
		return ErrOutOfRange
	}
	if it.Effect == "power_tonic" && u.PowerTonic != 0 || it.Effect == "snare" && u.Snared {
		return ErrEffectActive
	}
	return nil
}

// ItemEffectForecast は新道具の対象と能力の変化（DES-NG-16）。
type ItemEffectForecast struct {
	Unit          int
	Name, Effect  string
	Before, After Stats
}

// ItemForecast は状態を変えずに使用の検証と予想を行う（DES-NG-14・16）。
func (b *Battle) ItemForecast(cmd Command) (ItemEffectForecast, error) {
	if cmd.Action != UseItem {
		return ItemEffectForecast{}, ErrInvalidAction
	}
	if err := b.validateCommand(cmd); err != nil {
		return ItemEffectForecast{}, err
	}
	u := b.units[cmd.Unit]
	if cmd.Item < 0 || cmd.Item >= len(u.Items) {
		return ItemEffectForecast{}, ErrNoItem
	}
	it := u.Items[cmd.Item].Item
	if it.Effect == "" {
		return ItemEffectForecast{}, ErrInvalidAction
	}
	if err := b.validateEffectItem(cmd, it); err != nil {
		return ItemEffectForecast{}, err
	}
	i := b.effectTarget(cmd)
	target := b.units[i]
	before := target.EffectiveStats()
	after := before
	if it.Effect == "power_tonic" {
		after.Str += it.Amount
		after.Mag += it.Amount
	} else {
		after.Mov = 0
	}
	return ItemEffectForecast{Unit: i, Name: target.Name, Effect: it.Effect, Before: before, After: after}, nil
}
func (b *Battle) useEffectItem(events []Event, user, slot int, pos Pos) []Event {
	cmd := Command{Unit: user, To: b.units[user].Pos, ItemTarget: &pos}
	target := b.effectTarget(cmd)
	it := b.spendItem(user, slot)
	events = b.emit(events, Event{Kind: EventItemUse, Unit: user, Target: target, To: pos, Item: it.Name, Effect: it.Effect, Amount: it.Amount})
	if it.Effect == "power_tonic" {
		b.units[target].PowerTonic = it.Amount
	} else {
		b.units[target].Snared = true
	}
	return b.emit(events, Event{Kind: EventEffectApply, Unit: target, Target: user, To: pos, Item: it.Name, Effect: it.Effect, Amount: it.Amount})
}
func (b *Battle) expireEffects(events []Event) []Event {
	for i := range b.units {
		u := &b.units[i]
		if u.PowerTonic != 0 {
			n := u.PowerTonic
			u.PowerTonic = 0
			events = b.emit(events, Event{Kind: EventEffectExpire, Unit: i, To: u.Pos, Effect: "power_tonic", Amount: n})
		}
		if u.Snared {
			u.Snared = false
			events = b.emit(events, Event{Kind: EventEffectExpire, Unit: i, To: u.Pos, Effect: "snare"})
		}
	}
	return events
}
func (b *Battle) clearEffects() {
	for i := range b.units {
		b.units[i].PowerTonic = 0
		b.units[i].Snared = false
	}
}

// ItemTargets は移動の検証を一度だけ行い、有効な対象を列挙する（DES-NG-16）。
func (b *Battle) ItemTargets(cmd Command) []int {
	if cmd.Action != UseItem || b.validateCommand(cmd) != nil {
		return nil
	}
	u := b.units[cmd.Unit]
	if cmd.Item < 0 || cmd.Item >= len(u.Items) {
		return nil
	}
	it := u.Items[cmd.Item].Item
	if it.Effect == "" {
		return nil
	}
	var out []int
	for i, target := range b.units {
		p := target.Pos
		if i == cmd.Unit {
			p = cmd.To
		}
		cmd.ItemTarget = &p
		if b.validateEffectItem(cmd, it) == nil {
			out = append(out, i)
		}
	}
	return out
}

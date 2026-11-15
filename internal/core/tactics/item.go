package tactics

// アイテムの使用と回復（DES-ITM-01、DES-MAP-02）。

// heal は、ユニット i の HP を amount だけ回復する（最大 HP を超えない）（DES-MAP-02、DES-ITM-01）。
// 実際に回復した量が 1 以上なら Heal を出す。item はアイテムの名前（砦なら空）。
func (b *Battle) heal(events []Event, i, amount int, item string) []Event {
	u := &b.units[i]
	amount = min(amount, u.Stats.HP-u.HP)
	if amount <= 0 {
		return events
	}
	u.HP += amount
	return b.emit(events, Event{Kind: EventHeal, Unit: i, Amount: amount, Item: item})
}

// validateUseItem は UseItem の検証 6 を確かめる（DES-BTL-04、DES-ITM-01）。状態は変えない。
// Item 番目のアイテムがあり（ErrNoItem）、HP が最大でない（ErrFullHP）。
//
// 爆弾のときは、爆心の指定・マップの中・投げられる距離を確かめる（DES-NG-02）。爆弾は HP が最大でも使える。
func (b *Battle) validateUseItem(cmd Command) error {
	u := b.units[cmd.Unit]
	if cmd.Item < 0 || cmd.Item >= len(u.Items) {
		return ErrNoItem
	}
	if it := u.Items[cmd.Item].Item; it.Bomb {
		return b.validateBomb(cmd, it)
	}
	if cmd.Blast != nil {
		return ErrNotBomb
	}
	if u.HP >= u.Stats.HP {
		return ErrFullHP
	}
	return nil
}

// useItem は、ユニット i が手持ちの k 番目のアイテムを使う（DES-ITM-01、FR-ITM-02）。
// HP を Heal だけ回復し（最大 HP を超えない）、数を 1 減らす。0 になったら手持ちから消す。
func (b *Battle) useItem(events []Event, i, k int) []Event {
	it := b.spendItem(i, k)
	return b.heal(events, i, it.Heal, it.Name)
}

// spendItem は、ユニット i の手持ちの k 番目のアイテムを 1 つ減らし（0 になったら手持ちから消す）、そのアイテムを返す（DES-ITM-01、DES-NG-02）。
func (b *Battle) spendItem(i, k int) Item {
	u := &b.units[i]
	it := u.Items[k].Item
	// 手持ちの列は写しを作ってから変える（出来事の写しと列を共有しないため）。
	items := append([]ItemStack(nil), u.Items...)
	items[k].Count--
	if items[k].Count == 0 {
		items = append(items[:k], items[k+1:]...)
	}
	if len(items) == 0 {
		items = nil
	}
	u.Items = items
	return it
}

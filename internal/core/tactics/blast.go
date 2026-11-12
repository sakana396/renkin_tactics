package tactics

// 爆弾（DES-NG-02、NG-FR-02）。

// CellsWithin は、マップの中で center からのマンハッタン距離が radius 以下のマスを、Y・X の順で返す（DES-NG-02、DES-NG-03）。
// 爆発の範囲と、投げられる範囲の両方に使う。radius が負なら空。
func (b *Battle) CellsWithin(center Pos, radius int) []Pos {
	var out []Pos
	for y := center.Y - radius; y <= center.Y+radius; y++ {
		for x := center.X - radius; x <= center.X+radius; x++ {
			p := Pos{X: x, Y: y}
			if b.inMap(p) && distance(center, p) <= radius {
				out = append(out, p)
			}
		}
	}
	return out
}

// validateBomb は爆弾の UseItem の検証を確かめる（DES-NG-02）。状態は変えない。
// 爆心がある（ErrNoBlastCenter）、マップの中（ErrBlastOutOfMap）、移動後の位置 To から ThrowRange 以内（ErrOutOfThrowRange）。
// 爆心の地形は問わない（通れないマスでもよい）。
func (b *Battle) validateBomb(cmd Command, it Item) error {
	if cmd.Blast == nil {
		return ErrNoBlastCenter
	}
	if !b.inMap(*cmd.Blast) {
		return ErrBlastOutOfMap
	}
	if distance(cmd.To, *cmd.Blast) > it.ThrowRange {
		return ErrOutOfThrowRange
	}
	return nil
}

// blastVictims は、user が to にいるものとして、center から radius 以内にいる生きているユニットの添字を、添字の順に返す（DES-NG-02）。
// 敵・味方・使った本人を区別しない。
func (b *Battle) blastVictims(user int, to, center Pos, radius int) []int {
	var out []int
	for i, u := range b.units {
		if !u.Alive {
			continue
		}
		pos := u.Pos
		if i == user {
			pos = to
		}
		if distance(center, pos) <= radius {
			out = append(out, i)
		}
	}
	return out
}

// useBomb は、ユニット i が手持ちの k 番目の爆弾を center に使う（DES-NG-02）。呼ぶ前に i は移動を終えている。
// 在庫を 1 減らし、EventBlast を出す。爆心から BlastRadius 以内の生きているユニットすべてに、添字の順に BlastDamage を与える
// （命中判定・防御・地形・武器の相性はなし。HP は 0 で止める）。1 体ごとに EventBlastHit を出し、HP が 0 になったら EventFall が続く。
// 経験は与えない。最後に勝ち負けを確かめる。
func (b *Battle) useBomb(events []Event, i, k int, center Pos) []Event {
	it := b.spendItem(i, k)
	events = b.emit(events, Event{Kind: EventBlast, Unit: i, To: center, Item: it.Name})
	for _, v := range b.blastVictims(i, b.units[i].Pos, center, it.BlastRadius) {
		u := &b.units[v]
		u.HP = max(0, u.HP-it.BlastDamage)
		events = b.emit(events, Event{Kind: EventBlastHit, Unit: v, Target: i, Amount: it.BlastDamage, Item: it.Name})
		if u.HP == 0 {
			events = b.fall(events, v)
		}
	}
	return b.judge(events)
}

// BlastTarget は、爆発の予想の 1 体分（DES-NG-03）。
type BlastTarget struct {
	// Unit・Name・Side は巻き込まれるユニット。
	Unit int
	Name string
	Side Side
	// HP は今の HP、After は爆発のあとの HP（0 で止める）。Falls は倒れること。
	HP    int
	After int
	Falls bool
}

// BlastForecast は爆弾の使用の予想（DES-NG-03）。
type BlastForecast struct {
	// Center・Radius・Damage は爆心・爆発の半径・固定ダメージ。
	Center Pos
	Radius int
	Damage int
	// Targets は巻き込まれる生きているユニット（使う人自身を含む）を、添字の順に並べたもの。
	Targets []BlastTarget
}

// BlastForecast は、Action が UseItem で、Item が爆弾のコマンドについて、状態を変えずに爆発の予想を返す（DES-NG-03）。
// Action が UseItem でなければ ErrInvalidAction、爆弾でなければ ErrNotBomb。ほかの検証は Act と同じ（DES-BTL-04、DES-NG-02）。
// 使う人は To に移動したものとして数える。
func (b *Battle) BlastForecast(cmd Command) (BlastForecast, error) {
	if cmd.Action != UseItem {
		return BlastForecast{}, ErrInvalidAction
	}
	if err := b.validateCommand(cmd); err != nil {
		return BlastForecast{}, err
	}
	u := b.units[cmd.Unit]
	if cmd.Item < 0 || cmd.Item >= len(u.Items) {
		return BlastForecast{}, ErrNoItem
	}
	it := u.Items[cmd.Item].Item
	if !it.Bomb {
		return BlastForecast{}, ErrNotBomb
	}
	if err := b.validateBomb(cmd, it); err != nil {
		return BlastForecast{}, err
	}
	f := BlastForecast{Center: *cmd.Blast, Radius: it.BlastRadius, Damage: it.BlastDamage}
	for _, v := range b.blastVictims(cmd.Unit, cmd.To, *cmd.Blast, it.BlastRadius) {
		t := b.units[v]
		after := max(0, t.HP-it.BlastDamage)
		f.Targets = append(f.Targets, BlastTarget{Unit: v, Name: t.Name, Side: t.Side, HP: t.HP, After: after, Falls: after == 0})
	}
	return f, nil
}

package tactics

// Act は自軍のユニット 1 体を移動させ、行動させる（DES-BTL-04）。
//
// 共通の検証 0〜5 と、行動ごとの検証 6 を順に確かめ、最初に当てはまった誤りを返す。
// 失敗したら状態を変えない（DES-ERR-01）。
// Attack は移動してから Target を攻撃する（DES-BTL-06）。
// 行動のあと、戦闘が続いていて、生きている自軍が全員行動済みなら、敵軍の番と次の自軍の番の始めまでを進める（DES-BTL-02）。
// 返す出来事は、その間に起きたことすべて（DES-EVT-01）。
func (b *Battle) Act(cmd Command) ([]Event, error) {
	if err := b.validateCommand(cmd); err != nil {
		return nil, err
	}
	switch cmd.Action {
	case Attack:
		if err := b.validateAttack(cmd); err != nil {
			return nil, err
		}
	case UseItem:
		// アイテム（#9）の Issue で作る。検証 6 もそこで足す。
		return nil, errNotImplemented
	}

	// ここから状態を変える。まず移動する。
	var events []Event
	u := &b.units[cmd.Unit]
	from := u.Pos
	if cmd.To != from {
		u.Pos = cmd.To
		events = b.emit(events, Event{Kind: EventMove, Unit: cmd.Unit, From: from, To: cmd.To})
	}
	u.Acted = true
	switch cmd.Action {
	case Wait:
		events = b.emit(events, Event{Kind: EventWait, Unit: cmd.Unit})
	case Attack:
		events = b.attack(events, cmd.Unit, cmd.Target, cmd.To == from)
	}

	if b.Result != Ongoing {
		return events, nil
	}
	if b.allPlayersActed() {
		events = b.runEnemyPhase(events)
	}
	return events, nil
}

// validateCommand は Act の共通の検証 0〜5 を順に確かめる（DES-BTL-04）。状態は変えない。
func (b *Battle) validateCommand(cmd Command) error {
	// 0. Action が 3 つのどれかである。
	if cmd.Action < Wait || cmd.Action > UseItem {
		return ErrInvalidAction
	}
	// 1. 戦闘が終わっていない。
	if b.Result != Ongoing {
		return ErrBattleOver
	}
	// 2. 自軍の番である。
	if b.Phase != Player {
		return ErrNotPlayerPhase
	}
	// 3. Unit が存在し、生きている自軍のユニットである。
	if !b.validUnit(cmd.Unit) || !b.units[cmd.Unit].Alive || b.units[cmd.Unit].Side != Player {
		return ErrInvalidUnit
	}
	// 4. Unit がまだ行動していない。
	if b.units[cmd.Unit].Acted {
		return ErrAlreadyActed
	}
	// 5. To が移動できるマスである。
	if _, ok := b.Reachable(cmd.Unit)[cmd.To]; !ok {
		return ErrUnreachable
	}
	return nil
}

// EndPhase は自軍の番を途中で終え、敵軍の番と次の自軍の番の始めまでを進める（DES-BTL-02、FR-BTL-02）。
// 失敗したら状態を変えない（DES-ERR-01）。
func (b *Battle) EndPhase() ([]Event, error) {
	if b.Result != Ongoing {
		return nil, ErrBattleOver
	}
	if b.Phase != Player {
		return nil, ErrNotPlayerPhase
	}
	return b.runEnemyPhase(nil), nil
}

// allPlayersActed は、生きている自軍が全員行動済みかを返す。
func (b *Battle) allPlayersActed() bool {
	for _, u := range b.units {
		if u.Alive && u.Side == Player && !u.Acted {
			return false
		}
	}
	return true
}

// startPlayerPhase は自軍の番を始める（DES-BTL-02）。
// 自軍の全員の Acted を偽にし、PhaseStart を出し、砦にいる自軍を回復する。
func (b *Battle) startPlayerPhase(events []Event) []Event {
	b.Phase = Player
	for i := range b.units {
		if b.units[i].Side == Player {
			b.units[i].Acted = false
		}
	}
	events = b.emit(events, Event{Kind: EventPhaseStart, Side: Player, Turn: b.Turn})
	return b.healOnForts(events, Player)
}

// runEnemyPhase は敵軍の番を始めから終わりまで進め、次のターンの自軍の番を始める（DES-BTL-02）。
func (b *Battle) runEnemyPhase(events []Event) []Event {
	// 敵軍の番の始め: PhaseStart → 増援 → 砦の回復。
	b.Phase = Enemy
	events = b.emit(events, Event{Kind: EventPhaseStart, Side: Enemy, Turn: b.Turn})
	// この番に現れた増援は動かないので、増援を置く前の数だけ回す。
	n := len(b.units)
	events = b.placeReinforcements(events)
	events = b.healOnForts(events, Enemy)

	// 敵軍の番: 生きている敵を添字の順に 1 体ずつ動かす。
	for i := 0; i < n; i++ {
		if b.Result != Ongoing {
			return events
		}
		if u := b.units[i]; u.Alive && u.Side == Enemy {
			events = b.actEnemy(events, i)
		}
	}
	if b.Result != Ongoing {
		return events
	}

	// 敵軍の番の終わり: ターンを進め、自軍の番を始める。
	b.Turn++
	return b.startPlayerPhase(events)
}

// placeReinforcements は、このターンの増援を置く（DES-MAP-03）。
// 増援の Issue（#9）で作る。今は何もしない。
func (b *Battle) placeReinforcements(events []Event) []Event {
	return events
}

// healOnForts は、side の陣営で砦にいるユニットを回復する（DES-MAP-02）。
// 砦の Issue（#9）で作る。今は何もしない。
func (b *Battle) healOnForts(events []Event, side Side) []Event {
	return events
}

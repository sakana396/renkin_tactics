package tactics

// 騎兵の再移動と、巻き戻しの 1 手の区切り（DES-BTL-10、DES-IT06 8.2 節）。

// PendingCanto は待っている再移動（DES-BTL-10）。
// Unit は再移動するユニットの添字、Remaining は残りの移動力、HistoryID は行動の前の履歴（移動 ＋ 行動 ＋ 再移動の 1 手）の ID。
type PendingCanto struct {
	Unit      int
	Remaining int
	HistoryID uint64
}

// cantoState は戦闘の中の待っている再移動（DES-BTL-10）。値だけを持つので、snapshot・restore でそのまま写せる。
type cantoState struct {
	active bool
	PendingCanto
}

// PendingCanto は待っている再移動を返す（DES-BTL-10）。待っていなければ ok が偽。
func (b *Battle) PendingCanto() (PendingCanto, bool) {
	return b.canto.PendingCanto, b.canto.active
}

// CantoReachable は、待っている再移動で行けるマスと、そこまでの最小の費用を返す（DES-BTL-10）。
// 行動のあとの盤面で、残りの移動力まで、Reachable と同じ地形の費用と占有の決まりで求める。今の位置（費用 0）を含む。
// 待っていなければ nil を返す。
func (b *Battle) CantoReachable() map[Pos]int {
	if !b.canto.active {
		return nil
	}
	return (BoardView{terrain: b.terrain, units: b.units}).reachableWithin(b.canto.Unit, b.canto.Remaining)
}

// canCanto は、行動 action を済ませたユニット unit が再移動を待てるかを返す（DES-IT06 8.2 節）。
// 攻撃・回復・道具のあとで、生きていて、再移動のスキルを持ち、残りが 0 より大きいとき。
// 勝敗が決まっていないことは、呼ぶ側（Act）が先に確かめる。
func (b *Battle) canCanto(unit int, action Action, remaining int) bool {
	if action != Attack && action != Heal && action != UseItem {
		return false
	}
	u := b.units[unit]
	_, ok := u.skill(Canto)
	return ok && u.Alive && remaining > 0
}

// actCanto は待っている再移動を CantoMove（動く）か CantoEnd（その場で終える）で閉じる（DES-BTL-10）。
//
// 検証に失敗したら、状態・乱数・待っている再移動を変えない（DES-ERR-01）。
// 再移動のあとで出来事を評価し（DES-IT06 9.2 節）、最初の Act が公開した履歴の終わりを延ばす（新しい 1 手は作らない）。
// そのあと、戦闘が続いていて生きている自軍が全員行動済みなら、敵軍の番を始める（DES-BTL-02）。
func (b *Battle) actCanto(cmd Command) ([]Event, error) {
	if b.Result != Ongoing {
		return nil, ErrBattleOver
	}
	if b.Phase != Player {
		return nil, ErrNotPlayerPhase
	}
	if b.NeedsResume() {
		return nil, ErrResumeRequired
	}
	if !b.canto.active {
		return nil, ErrNoCantoPending
	}
	if cmd.Unit != b.canto.Unit {
		return nil, ErrInvalidUnit
	}
	u := &b.units[cmd.Unit]
	from := u.Pos
	switch cmd.Action {
	case CantoMove:
		if _, ok := b.CantoReachable()[cmd.To]; !ok {
			return nil, ErrUnreachable
		}
	case CantoEnd:
		if cmd.To != from {
			return nil, ErrUnreachable
		}
	}

	// ここから状態を変える。
	id := b.canto.HistoryID
	b.canto = cantoState{}
	var events []Event
	if cmd.To != from {
		u.Pos = cmd.To
		events = b.emit(events, Event{Kind: EventMove, Unit: cmd.Unit, From: from, To: cmd.To})
	}
	// 再移動は、その移動のあとで出来事を評価する（DES-IT06 9.2 節）。
	events = b.evaluateMapEvents(events, events)
	for i := range b.history {
		if b.history[i].entry.ID == id {
			b.finishHistory(i)
			break
		}
	}
	if b.Result != Ongoing {
		return events, nil
	}
	if b.allPlayersActed() {
		events = b.runEnemyPhase(events, true)
	}
	return events, nil
}

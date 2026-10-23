package tactics

import "fmt"

// History は実行済みの戻り先を古い順で独立した配列にして返す（DES-RWD-02）。
func (b *Battle) History() []HistoryEntry { return b.historyEntries() }

// RewindsLeft は写しに含めない戦闘ごとの残り回数を返す（DES-RWD-04）。
func (b *Battle) RewindsLeft() int { return b.rewindsLeft }

// NeedsResume は勝敗が未決定で、自動進行の続きがあるかを返す（DES-RWD-02）。
func (b *Battle) NeedsResume() bool {
	return b.Result == Ongoing && b.progress.stage != playerInput
}

// Rewind は検証と復元の成功後に未来を捨て、1 回消費する（DES-RWD-02・04）。
func (b *Battle) Rewind(id uint64) error {
	if b.rewindsLeft < 1 {
		return fmt.Errorf("%w: 残り 0 回です", ErrNoRewinds)
	}
	for i, h := range b.history {
		if h.entry.ID != id {
			continue
		}
		if !h.completed {
			break
		}
		if h.entry.Side != Player || (h.entry.Kind != HistoryAction && (h.entry.Kind != HistoryEndPhase || h.entry.Automatic)) {
			return fmt.Errorf("%w: 自軍の行動か明示の番終了の前だけ戻せます", ErrInvalidRewind)
		}
		// restore は別の PCG と深い写しを用意してから置き換え、失敗時は現在値を保つ。
		if err := b.restore(h.before); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidRewind, err)
		}
		clear(b.history[i:])
		b.history = b.history[:i]
		b.rewindsLeft--
		return nil
	}
	return fmt.Errorf("%w: 履歴 ID %d は実行済みの戻り先にありません", ErrInvalidRewind, id)
}

// Resume は保留した番処理を自軍の入力待ちか勝敗まで進める（DES-RWD-02）。
func (b *Battle) Resume() ([]Event, error) {
	if !b.NeedsResume() {
		return nil, fmt.Errorf("%w: 自動進行の続きがありません", ErrNoPendingResume)
	}
	return b.continuePhase(nil), nil
}

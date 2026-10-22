package tactics

// HistoryKind は戻り先となる処理の単位（DES-RWD-01・02）。
type HistoryKind int

const (
	// HistoryAction は自軍・敵軍の 1 体の移動と行動。
	HistoryAction HistoryKind = iota
	// HistoryEndPhase は敵軍の番を始める前。Automatic で自動移行を区別する。
	HistoryEndPhase
	// HistoryPlayerStart は次の自軍の番を始める前。
	HistoryPlayerStart
)

// 内部名は既存の記録処理でも使う。
const (
	historyAction      = HistoryAction
	historyEndPhase    = HistoryEndPhase
	historyPlayerStart = HistoryPlayerStart
)

// HistoryEntry は一覧に使う値。写しやスライスを外へ渡さない（DES-RWD-01）。
// EventStart・EventEnd は New の出来事も含めた通し添字の半開区間。
// Unit・Target は行動者・相手がない場合 -1。Action・To は HistoryAction だけで使う。
type HistoryEntry struct {
	ID         uint64
	Turn       int
	Side       Side
	Kind       HistoryKind
	Action     Action
	Unit       int
	Target     int
	UnitName   string
	TargetName string
	To         Pos
	Automatic  bool
	EventStart int
	EventEnd   int
}

type historyEntry = HistoryEntry

// historyRecord は一覧用の値と行動前の完全な写しを結ぶ（DES-RWD-01）。
type historyRecord struct {
	entry     historyEntry
	before    battleSnapshot
	completed bool
}

// historyEntries は一覧用の値の独立した配列を返す（DES-RWD-01）。
func (b *Battle) historyEntries() []historyEntry {
	out := make([]historyEntry, len(b.history))
	for i, h := range b.history {
		out[i] = h.entry
	}
	return out
}

// beginHistory は検証後、変更の直前に戻り先を記録する（DES-RWD-01）。
func (b *Battle) beginHistory(e historyEntry) int {
	e.ID = b.nextHistoryID
	b.nextHistoryID++
	e.Turn = b.Turn
	e.Side = b.Phase
	if e.Kind == historyPlayerStart {
		e.Side = Player
		e.Turn = b.Turn + 1
	}
	e.EventStart = b.eventCount
	e.EventEnd = b.eventCount
	if b.validUnit(e.Unit) {
		e.UnitName = b.units[e.Unit].Name
	}
	if b.validUnit(e.Target) {
		e.TargetName = b.units[e.Target].Name
	}
	b.history = append(b.history, historyRecord{entry: e, before: b.snapshot()})
	return len(b.history) - 1
}

// finishHistory は処理の終了時点の出来事の境界を確定する（DES-RWD-01）。
func (b *Battle) finishHistory(i int) {
	b.history[i].entry.EventEnd = b.eventCount
	b.history[i].completed = true
}

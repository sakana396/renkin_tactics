package tactics

// EventKind は出来事の種類（DES-EVT-01）。
//
// Action の Wait、Result の Won・Lost と名前が重なるので、Event を頭に付ける。
type EventKind int

const (
	// EventPhaseStart は番が始まったこと。Side、Turn を持つ。
	EventPhaseStart EventKind = iota
	// EventReinforce は増援が現れたこと。Unit、To を持つ。
	EventReinforce
	// EventHeal は砦かアイテムで回復したこと。Unit、Amount、Item を持つ。
	EventHeal
	// EventMove は移動したこと。Unit、From、To を持つ。
	EventMove
	// EventStrike は 1 撃を行ったこと。Unit、Target、Hit、Amount、Crit、Pavise を持つ。
	// Amount は必殺・大盾を入れた与えたダメージ（HP の残りで切らない）。外れたら 0。
	EventStrike
	// EventFall は倒れたこと。Unit を持つ。
	EventFall
	// EventWait は待機したこと。Unit を持つ。
	EventWait
	// EventWon は勝ちが決まったこと。
	EventWon
	// EventLost は負けが決まったこと。
	EventLost
)

// Event は起きたこと 1 つ（DES-EVT-01）。Kind ごとに決まった項目だけが意味を持つ。
// Units は、その出来事が起きた直後のユニットの写し（Units() と同じ添字）。
type Event struct {
	Kind   EventKind
	Side   Side
	Turn   int
	Unit   int
	Target int
	From   Pos
	To     Pos
	Hit    bool
	Amount int
	Crit   bool
	Pavise bool
	Item   string
	Units  []Unit
}

// emit は、今の状態の写しを付けて出来事を events に足す（DES-EVT-01）。
func (b *Battle) emit(events []Event, e Event) []Event {
	e.Units = b.Units()
	return append(events, e)
}

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
	// EventHeal は砦・回復薬・杖で回復したこと。Unit、Amount、Item を持つ（DES-HEL-01）。
	// Unit は回復されたユニット、Amount は実際に回復した量、Item は回復薬か杖の名前（砦なら空）。
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
	// EventExp は経験を得たこと。Unit、Amount（得た経験。1・10・30 のどれか）を持つ（DES-GRW-03）。
	EventExp
	// EventLevelUp はレベルが上がったこと。Unit、Level（上がったあとのレベル）、
	// Gains（上がった能力が 1、ほかは 0。移動は 0）を持つ（DES-GRW-03）。
	EventLevelUp
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
	Level  int
	Gains  Stats
	Units  []Unit
}

// emit は、今の状態の写しを付けて出来事を events に足す（DES-EVT-01）。
func (b *Battle) emit(events []Event, e Event) []Event {
	b.eventCount++
	e.Units = b.Units()
	return append(events, e)
}

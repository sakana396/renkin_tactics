package tactics

import (
	"fmt"
	"sort"
)

// 出来事の増援・突撃（DES-MAP-07）と、生きている敵の上限（DES-MAP-03・DES-RNF-01 の改訂）。

// maxLiveEnemies は一度に生きている敵の上限（DES-IT06 9.3 節、E-91）。
// 初めの敵がこれを超える Setup は拒み、出現で超える個体は出さない。
const maxLiveEnemies = 39

// TriggerKind は出来事のきっかけの種類（DES-MAP-07）。
type TriggerKind int

const (
	// TriggerEnemyMoved は、指定の敵が実際に移動したこと（同じ位置での待機は含まない）。
	TriggerEnemyMoved TriggerKind = iota
	// TriggerEnemyDefeated は、指定の敵が倒れたこと。
	TriggerEnemyDefeated
	// TriggerEnemyAttacked は、指定の敵を受ける側とする 1 撃が試みられたこと（外れ・0 ダメージ・反撃を含む）。
	TriggerEnemyAttacked
	// TriggerPlayerEntered は、自軍が実際に移動して、着いたマスが範囲の中にあること。
	TriggerPlayerEntered
)

// EffectKind は出来事の効果の種類（DES-MAP-07）。
type EffectKind int

const (
	// EffectReinforce は、次に来る敵軍の番の始めに増援を出す。
	EffectReinforce EffectKind = iota
	// EffectChargeGroup は、指定の一団を突撃（Charge）にする。
	EffectChargeGroup
)

// Area は両端を含む盤の中の長方形（DES-MAP-07）。
type Area struct{ Min, Max Pos }

// contains は p が長方形の中にあるかを返す。
func (a Area) contains(p Pos) bool {
	return p.X >= a.Min.X && p.X <= a.Max.X && p.Y >= a.Min.Y && p.Y <= a.Max.Y
}

// MapEventSetup は出来事 1 件の入力（DES-MAP-07）。きっかけ 1 つ・効果 1 つで、1 回だけ起こる。
// Unit はきっかけの敵の ID（敵の 3 つのきっかけだけ）、Area は TriggerPlayerEntered だけで使う。
// Enemies は EffectReinforce の増援（現れるマスは Unit.Pos）、Group は EffectChargeGroup の一団の名前。
type MapEventSetup struct {
	ID      string
	Trigger TriggerKind
	Unit    string
	Area    Area
	Effect  EffectKind
	Enemies []UnitSetup
	Group   string
}

// reservation は出来事で予約した増援（DES-IT06 9.3 節）。Turn は出る予定のターン、Event は定義の添字。
type reservation struct {
	Turn  int
	Event int
	Used  bool
}

// eventState は戦闘の中の出来事の状態（DES-IT06 9.3 節）。snapshot・restore で深く写す。
type eventState struct {
	fired        []bool
	reservations []reservation
	charged      map[string]bool
}

// clone は独立した写しを作る。
func (s eventState) clone() eventState {
	out := eventState{
		fired:        append([]bool(nil), s.fired...),
		reservations: append([]reservation(nil), s.reservations...),
	}
	if s.charged != nil {
		out.charged = make(map[string]bool, len(s.charged))
		for g, v := range s.charged {
			out.charged[g] = v
		}
	}
	return out
}

// cloneMapEvents は出来事の定義を、増援の手持ち・特効の分類・武器の列まで入力から独立させる（UnitSetup.clone）。
func cloneMapEvents(events []MapEventSetup) []MapEventSetup {
	out := make([]MapEventSetup, len(events))
	for i, e := range events {
		enemies := make([]UnitSetup, len(e.Enemies))
		for j, u := range e.Enemies {
			enemies[j] = u.clone()
		}
		e.Enemies = enemies
		out[i] = e
	}
	return out
}

// validateEnemyCap は初めの敵が上限を超えないことを確かめる（DES-IT06 9.3 節）。
func validateEnemyCap(s Setup) error {
	n := 0
	for _, u := range s.Units {
		if u.Side == Enemy {
			n++
		}
	}
	if n > maxLiveEnemies {
		return fmt.Errorf("%w: 初めの敵が %d 体います（%d 体まで）", ErrInvalidSetup, n, maxLiveEnemies)
	}
	return nil
}

// validateMapEvents は出来事の定義を確かめる（DES-MAP-07 の拒むもの）。最初に見つかった誤りを返す。
// 敵の ID は、初めの敵・ターンの増援・出来事の増援を通して重ならない（空の ID は比べない）。
func validateMapEvents(s Setup, passable func(Pos) bool, width, height int) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidSetup, fmt.Sprintf(format, args...))
	}
	ids := map[string]string{}
	groups := map[string]bool{}
	approaches := map[string]Approach{}
	addEnemy := func(where string, u UnitSetup) error {
		if u.Group != "" {
			groups[u.Group] = true
			if first, ok := approaches[u.Group]; ok && first != u.Approach {
				return bad("%s: 一団 %q の寄り方がそろっていません", where, u.Group)
			}
			approaches[u.Group] = u.Approach
		}
		if u.ID == "" {
			return nil
		}
		if first, ok := ids[u.ID]; ok {
			return bad("%s: 敵の ID %q が %s と重なっています", where, u.ID, first)
		}
		ids[u.ID] = where
		return nil
	}
	for i, u := range s.Units {
		if u.Side != Enemy {
			continue
		}
		if err := addEnemy(fmt.Sprintf("units[%d]", i), u); err != nil {
			return err
		}
	}
	for i, r := range s.Reinforcements {
		if err := addEnemy(fmt.Sprintf("reinforcements[%d]", i), r.Unit); err != nil {
			return err
		}
	}
	for i, e := range s.Events {
		for j, u := range e.Enemies {
			where := fmt.Sprintf("events[%d].enemies[%d]", i, j)
			if !passable(u.Pos) {
				return bad("%s: 座標 (%d, %d) がマップの外か、通れない地形です", where, u.Pos.X, u.Pos.Y)
			}
			if u.Lord || u.Boss {
				return bad("%s: 増援は主人公とボスになれません", where)
			}
			if u.Side != Enemy {
				return bad("%s: 増援が敵軍ではありません", where)
			}
			if err := validateUnit(u); err != nil {
				return bad("%s: %v", where, err)
			}
			if err := addEnemy(where, u); err != nil {
				return err
			}
		}
	}

	eventIDs := map[string]int{}
	for i, e := range s.Events {
		where := fmt.Sprintf("events[%d]", i)
		if e.ID == "" {
			return bad("%s: 出来事の ID が空です", where)
		}
		if j, ok := eventIDs[e.ID]; ok {
			return bad("%s: 出来事の ID %q が events[%d] と重なっています", where, e.ID, j)
		}
		eventIDs[e.ID] = i
		switch e.Trigger {
		case TriggerEnemyMoved, TriggerEnemyDefeated, TriggerEnemyAttacked:
			if e.Area != (Area{}) {
				return bad("%s: 敵のきっかけに範囲は書けません", where)
			}
			if _, ok := ids[e.Unit]; !ok || e.Unit == "" {
				return bad("%s: きっかけの敵 %q がいません", where, e.Unit)
			}
		case TriggerPlayerEntered:
			if e.Unit != "" {
				return bad("%s: 範囲のきっかけに敵は書けません", where)
			}
			a := e.Area
			if a.Min.X > a.Max.X || a.Min.Y > a.Max.Y {
				return bad("%s: 範囲 (%d, %d)〜(%d, %d) の向きが逆です", where, a.Min.X, a.Min.Y, a.Max.X, a.Max.Y)
			}
			if a.Min.X < 0 || a.Min.Y < 0 || a.Max.X >= width || a.Max.Y >= height {
				return bad("%s: 範囲 (%d, %d)〜(%d, %d) が盤の外です", where, a.Min.X, a.Min.Y, a.Max.X, a.Max.Y)
			}
		default:
			return bad("%s: きっかけ %d が正しくありません", where, e.Trigger)
		}
		switch e.Effect {
		case EffectReinforce:
			if len(e.Enemies) == 0 {
				return bad("%s: 増援の敵がいません", where)
			}
			if e.Group != "" {
				return bad("%s: 増援の効果に一団は書けません", where)
			}
		case EffectChargeGroup:
			if len(e.Enemies) != 0 {
				return bad("%s: 突撃の効果に増援は書けません", where)
			}
			if !groups[e.Group] || e.Group == "" {
				return bad("%s: 一団 %q がいません", where, e.Group)
			}
		default:
			return bad("%s: 効果 %d が正しくありません", where, e.Effect)
		}
	}
	return nil
}

// liveEnemies は生きている敵の数を返す。
func (b *Battle) liveEnemies() int {
	n := 0
	for _, u := range b.units {
		if u.Alive && u.Side == Enemy {
			n++
		}
	}
	return n
}

// spawnEnemy は増援 1 体を出す（DES-MAP-03、DES-IT06 9.3 節）。
// 出すと生きている敵が上限を超えるとき、または空きがないときは出さず、出来事も出さない。乱数は引かない。
// 突撃に変わった一団の敵は、出たときから突撃にする。
func (b *Battle) spawnEnemy(events []Event, us UnitSetup) []Event {
	if b.liveEnemies() >= maxLiveEnemies {
		return events
	}
	pos, ok := b.reinforcementPosition(us.Pos)
	if !ok {
		return events
	}
	u := newUnit(us)
	u.Pos = pos
	if u.Group != "" && b.mapState.charged[u.Group] {
		u.AI = Charge
	}
	b.units = append(b.units, u)
	return b.emit(events, Event{Kind: EventReinforce, Unit: len(b.units) - 1, To: pos})
}

// placeEventReinforcements は、このターンに予約した出来事の増援を、出来事の定義の順・個体の順に出す（DES-IT06 9.3 節）。
// 出られなかった個体は先送りせず、予約は使ったものにする。
func (b *Battle) placeEventReinforcements(events []Event) []Event {
	order := make([]int, 0, len(b.mapState.reservations))
	for i, r := range b.mapState.reservations {
		if !r.Used && r.Turn == b.Turn {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		return b.mapState.reservations[order[i]].Event < b.mapState.reservations[order[j]].Event
	})
	for _, i := range order {
		b.mapState.reservations[i].Used = true
		for _, us := range b.mapEvents[b.mapState.reservations[i].Event].Enemies {
			events = b.spawnEnemy(events, us)
		}
	}
	return events
}

// observed は、1 つの区切りの出来事の列から、出来事 e のきっかけが成り立ったかを返す（DES-IT06 9.2 節）。
func (b *Battle) observed(e MapEventSetup, events []Event) bool {
	isTarget := func(i int) bool {
		return b.validUnit(i) && b.units[i].Side == Enemy && b.units[i].ID == e.Unit
	}
	for _, ev := range events {
		switch e.Trigger {
		case TriggerEnemyMoved:
			if ev.Kind == EventMove && isTarget(ev.Unit) {
				return true
			}
		case TriggerEnemyDefeated:
			if ev.Kind == EventFall && isTarget(ev.Unit) {
				return true
			}
		case TriggerEnemyAttacked:
			if ev.Kind == EventStrike && isTarget(ev.Target) {
				return true
			}
		case TriggerPlayerEntered:
			if ev.Kind == EventMove && b.validUnit(ev.Unit) && b.units[ev.Unit].Side == Player && e.Area.contains(ev.To) {
				return true
			}
		}
	}
	return false
}

// evaluateMapEvents は、行動を済ませたあとで、その区切りの出来事 observed から、
// まだ起きていない出来事を定義の順に評価して効果をかける（DES-IT06 9.2 節）。
// 勝敗が決まっていれば何もしない。効果はきっかけを新しく起こさないので、繰り返し評価しない。
func (b *Battle) evaluateMapEvents(events []Event, observed []Event) []Event {
	if b.Result != Ongoing || len(b.mapEvents) == 0 {
		return events
	}
	// 評価の前に観測を写し、効果の出来事を観測に混ぜない。
	window := append([]Event(nil), observed...)
	for i, e := range b.mapEvents {
		if b.mapState.fired[i] || !b.observed(e, window) {
			continue
		}
		b.mapState.fired[i] = true
		events = b.emit(events, Event{Kind: EventTrigger, MapEvent: e.ID})
		switch e.Effect {
		case EffectReinforce:
			turn := b.Turn
			if b.Phase == Enemy {
				turn++
			}
			b.mapState.reservations = append(b.mapState.reservations, reservation{Turn: turn, Event: i})
		case EffectChargeGroup:
			events = b.chargeGroup(events, e)
		}
	}
	return events
}

// chargeGroup は、一団 e.Group を突撃にし、まだ出ていない同じ一団にも効くよう記録する（DES-IT06 9.3 節）。
func (b *Battle) chargeGroup(events []Event, e MapEventSetup) []Event {
	if b.mapState.charged == nil {
		b.mapState.charged = map[string]bool{}
	}
	b.mapState.charged[e.Group] = true
	for i := range b.units {
		u := &b.units[i]
		if u.Alive && u.Side == Enemy && u.Group == e.Group {
			u.AI = Charge
		}
	}
	return b.emit(events, Event{Kind: EventGroupCharge, MapEvent: e.ID, Group: e.Group})
}

// reservedOrigins は、まだ使っていない予約の増援の元のマスを返す（DES-RNF-01 の改訂）。
func reservedOrigins(mapEvents []MapEventSetup, s eventState) []Pos {
	var out []Pos
	for _, r := range s.reservations {
		if r.Used {
			continue
		}
		for _, u := range mapEvents[r.Event].Enemies {
			out = append(out, u.Pos)
		}
	}
	return out
}

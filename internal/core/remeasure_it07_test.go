package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// TestIT07Remeasure は DES-IT07 6 章の旧新版比較用計測器（TC-FR-MAP-04-31）。
// 長時間の全計測は RENKIN_IT07_REPORT を指定したときだけ行う。
func TestIT07Remeasure(t *testing.T) {
	t.Run("TC-FR-MAP-04-31", remeasureIT07)
}

func remeasureIT07(t *testing.T) {
	path := os.Getenv("RENKIN_IT07_REPORT")
	if path == "" {
		t.Skip("RENKIN_IT07_REPORT を指定したときだけ全章を測る")
	}
	cat := baseCatalog(t)
	chapters := map[string]bool{}
	for _, id := range strings.Split(os.Getenv("RENKIN_IT07_CH"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			chapters[id] = true
		}
	}
	report := it07MeasureReport{
		Commit:     os.Getenv("RENKIN_IT07_COMMIT"),
		Conditions: "既存名簿・出撃・進路・行動順、seed 1〜20、最大50ターン。今の打ち手と賢い打ち手（巻き戻し主人公用reserve3）。打ち手は再移動を使わず終了する。",
		Observations: []string{
			"Observe は打ち手が通知した Event と、その時点の Event.Units を記録する。smartPlay が後で巻き戻した出来事も含み、巻き戻し後の現状態とは異なることがある。",
			"Battle に経路全体の公開取得APIはない。移動イベント、攻撃時の位置、再移動先を記録する。ZOCEndpointEntries は各移動イベントの着地点だけを数えたもので、経路上の進入回数ではない。",
			"#188: Path* は、各 EventMove の直前の盤面でゲームと同じ移動費・占有の規則の最小費用の経路を求めて数える。ExtraCost は最小費用の差（ZOC あり − なし）で正確。EntriesMin/Max は最小費用の経路のうち ZOC に入る回数が最少/最多のもの（到着点を含み出発点を含まない）で、実際の経路は分からないので幅で示す。最小費用でない遠回りは数えない。",
			"#188: 再移動は、同じ番に同じユニットが攻撃（行動した側の EventStrike）をしたあとの移動。待機のあとはゲームで再移動できないので印を付けても影響しない。杖のあとは区別できない。打ち手の自軍は再移動を使わないので、実際に現れるのは敵の再移動。PathAllTrials は巻き戻した試行を含む全試行、PathFinal は巻き戻しで捨てた記録を除いた最終経路。Unreachable・Violations は到達不能と不変条件（下限≦上限≦追加費用）の違反の回数で、0 でなければ計測器の誤り",
			"#188: ReinforcementsFinal は最終経路で出た増援、ReinforcementsAllTrials は全試行で出た増援（初出順）、ReinforcementsRewoundOnly は巻き戻す前だけ出た増援（全試行の出現のうち最終経路にない分。同じ ID が最終経路にもあれば差し引く）。UnspawnedFinalPathIDs は最終経路で出なかった予定の増援。旧新版の UnspawnedReinforcementIDs は全試行基準のまま。",
			"増援未出現の理由は予定ターン未到達、出来事未発火、発火後の次の敵フェイズ未到達、または試行後の未出現に分ける。試行後の未出現が上限か配置位置かは判定できない。",
			"崩れは小隊長の EventFall 時点の全 Unit 写しから記録する。巻き戻して同じ撃破を再試行した場合は、その試行ごとに重複して記録する。ZOC は ClassSkills のスキル名『通せんぼ』で認識する。",
			"崩れ欄は小隊長撃破イベント時点で崩れ状態が有効な配下の一覧で、遷移回数ではない。",
			"Commander・Collapsed・ZOC はフィールド名／スキル名を reflection で確認し、旧版にないフィールドや定数への直接参照を避ける。Unit の報告は ID・名前・位置・HP・生存・指揮官・崩れだけに絞る。",
			"初期状態は StartBattle 直後の Battle.Units を記録する。最大生存敵数には初期状態と全イベント写しを含める。RENKIN_IT07_CH は動作確認時の章限定。本計測では未指定にする。",
		},
	}
	players := []struct {
		name string
		play func(*tactics.Battle, []string, autoPlan) autoResult
	}{
		{"今", func(b *tactics.Battle, ids []string, p autoPlan) autoResult { return autoPlay(b, ids, p, 50) }},
		{"賢", func(b *tactics.Battle, ids []string, p autoPlan) autoResult {
			return smartPlay(b, ids, p, smartAll, 50)
		}},
	}
	var mu sync.Mutex
	t.Cleanup(func() {
		sort.Slice(report.Games, func(i, j int) bool {
			a, b := report.Games[i], report.Games[j]
			if a.Chapter != b.Chapter {
				return a.Chapter < b.Chapter
			}
			if a.Roster != b.Roster {
				return a.Roster < b.Roster
			}
			if a.Player != b.Player {
				return a.Player < b.Player
			}
			return a.Seed < b.Seed
		})
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Errorf("計測結果JSON: %v", err)
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Errorf("出力先の作成: %v", err)
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Errorf("計測結果の出力: %v", err)
			return
		}
		t.Logf("計測結果を %s に出力しました（%d 試遊）", path, len(report.Games))
	})
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		if len(chapters) > 0 && !chapters[id] {
			continue
		}
		for _, roster := range antiLureRosters {
			if ch == 1 && roster.name != "平均" || ch < 4 && roster.lostMira {
				continue
			}
			for _, player := range players {
				id, roster, player := id, roster, player
				t.Run(id+" "+roster.name+" "+player.name, func(t *testing.T) {
					t.Parallel()
					t.Logf("%s %s %s: seed 1〜20 を計測", id, roster.name, player.name)
					pathPoints, order := antiLureRoute(id, roster)
					for seed := uint64(1); seed <= 20; seed++ {
						b, ids, chosen := antiLureStart(t, cat, id, roster, seed)
						units := b.Units()
						game := it07MeasureGame{
							Chapter: id, Roster: roster.name, Deployment: chosen, Route: pathPoints,
							Order: order, Seed: seed, Player: player.name,
							InitialUnits: it07UnitSnapshots(units), InitialLiveEnemies: it07LiveEnemies(units),
							ScheduledReinforcements: it07Schedules(t, cat, id), SpawnedReinforcementTurn: map[string]int{}, TriggerTurn: map[string]int{}, TriggerSide: map[string]string{},
						}
						game.MaxLiveEnemies = game.InitialLiveEnemies
						game.InitialCommanderAlive = it07Commanders(units, true)
						turns := map[int]*it07MeasureTurn{}
						tracker := newIT07PathTracker(func(p tactics.Pos) int { return b.TerrainAt(p).Cost }, b.Width(), b.Height())
						currentTurn, currentSide := 1, tactics.Player
						observe := func(events []tactics.Event) {
							batchTurn, batchSide := b.Turn, b.Phase
							firstPhase := -1
							for i, event := range events {
								if event.Kind == tactics.EventPhaseStart {
									firstPhase = i
									break
								}
							}
							if firstPhase >= 0 {
								phase := events[firstPhase]
								if phase.Side == tactics.Enemy {
									batchTurn, batchSide = phase.Turn, tactics.Player
								} else {
									batchTurn, batchSide = phase.Turn-1, tactics.Enemy
								}
								if batchTurn < 1 {
									batchTurn = 1
								}
							}
							currentTurn, currentSide = batchTurn, batchSide
							for _, e := range events {
								if e.Kind == tactics.EventPhaseStart {
									currentSide = e.Side
									currentTurn = e.Turn
									tracker.PhaseStart()
								}
								turn := turns[currentTurn]
								if turn == nil {
									turn = &it07MeasureTurn{Turn: currentTurn}
									turns[currentTurn] = turn
								}
								eventsUnits := e.Units
								tracker.Event(e, eventsUnits, currentSide)
								game.MaxLiveEnemies = max(game.MaxLiveEnemies, it07LiveEnemies(eventsUnits))
								turn.MaxLiveEnemies = max(turn.MaxLiveEnemies, it07LiveEnemies(eventsUnits))
								if e.Kind == tactics.EventPhaseStart && e.Side == tactics.Enemy {
									turn.EnemyPhaseStarted = true
									game.EnemyPhaseTurns = appendUniqueInt(game.EnemyPhaseTurns, currentTurn)
								}
								if e.Kind == tactics.EventTrigger {
									turn.Triggers = append(turn.Triggers, e.MapEvent)
									game.TriggerTurn[e.MapEvent] = currentTurn
									if currentSide == tactics.Enemy {
										game.TriggerSide[e.MapEvent] = "enemy"
									} else {
										game.TriggerSide[e.MapEvent] = "player"
									}
								}
								if e.Unit >= 0 && e.Unit < len(eventsUnits) {
									unit := eventsUnits[e.Unit]
									uid := it07ID(unit)
									switch e.Kind {
									case tactics.EventReinforce:
										turn.Reinforcements = append(turn.Reinforcements, uid)
										game.SpawnedReinforcementTurn[uid] = currentTurn
										if it07Flag(unit, "Commander") {
											turn.CommandersSpawned = append(turn.CommandersSpawned, uid)
										}
									case tactics.EventMove:
										turn.Movements = append(turn.Movements, it07Movement{Unit: uid, Side: it07SideName(unit.Side), From: e.From, To: e.To})
										if it07Flag(unit, "Commander") && unit.Alive {
											turn.CommandersAlive = append(turn.CommandersAlive, uid)
										}
									case tactics.EventStrike:
										if e.Target >= 0 && e.Target < len(eventsUnits) {
											turn.Attacks = append(turn.Attacks, it07Attack{Unit: uid, Position: unit.Pos, Target: it07ID(eventsUnits[e.Target])})
										}
									case tactics.EventFall:
										if it07Flag(unit, "Commander") {
											turn.CommandersDefeated = append(turn.CommandersDefeated, uid)
										}
									}
									if e.Kind == tactics.EventMove && it07EnteredZOCEndpoint(unit, eventsUnits) {
										turn.ZOCEndpointEntries++
										turn.ZOCExtraCost++
										if unit.Side == tactics.Player {
											turn.PlayerZOCEndpointEntries++
										} else {
											turn.EnemyZOCEndpointEntries++
										}
									}
									if e.Kind == tactics.EventFall && it07Flag(unit, "Commander") {
										for _, fallenOrActive := range eventsUnits {
											if it07Flag(fallenOrActive, "Collapsed") {
												turn.CollapsedAtCommanderFall = append(turn.CollapsedAtCommanderFall, it07ID(fallenOrActive))
											}
										}
									}
								}
							}
						}
						plan := autoPlan{Order: order, Routes: [][]tactics.Pos{pathPoints}, Observe: observe, OnTurnStart: tracker.TurnStart, OnRewind: tracker.Rewind}
						result := player.play(b, ids, plan)
						game.Result, game.Turns = it07ResultName(result.Result), result.Turns
						game.HitTurnLimit = result.Result == tactics.Ongoing && result.Turns > 50
						game.Fallen, game.Deaths, game.Kills = result.Fallen, result.Deaths, result.Kills
						game.Rewinds = result.Rewinds
						game.FinalUnits = it07UnitSnapshots(b.Units())
						game.FinalCommanderAlive = it07Commanders(b.Units(), true)
						game.FinalCommanderDefeated = it07Commanders(b.Units(), false)
						game.PathAllTrials = tracker.AllTrials()
						game.PathFinal, game.ReinforcementsFinal = tracker.Final()
						game.ReinforcementsAllTrials = tracker.allSpawned
						game.ReinforcementsRewoundOnly = it07Difference(game.ReinforcementsAllTrials, game.ReinforcementsFinal)
						game.UnspawnedFinalPathIDs = it07NotIn(game.ScheduledReinforcements, game.ReinforcementsFinal)
						game.UnspawnedReinforcementIDs = it07Unspawned(game.ScheduledReinforcements, turns)
						game.UnspawnedReinforcementDetails = it07UnspawnedDetails(game.ScheduledReinforcements, game.SpawnedReinforcementTurn, game.TriggerTurn, game.TriggerSide, game.EnemyPhaseTurns, result.Turns)
						for _, turn := range turns {
							game.TurnDetails = append(game.TurnDetails, *turn)
						}
						sort.Slice(game.TurnDetails, func(i, j int) bool { return game.TurnDetails[i].Turn < game.TurnDetails[j].Turn })
						if game.MaxLiveEnemies > 39 {
							t.Errorf("seed %d: 最大生存敵数 %d が上限39を超過", seed, game.MaxLiveEnemies)
						}
						if game.Rewinds > 10 {
							t.Errorf("seed %d: 巻き戻し %d 回が上限10を超過", seed, game.Rewinds)
						}
						mu.Lock()
						report.Games = append(report.Games, game)
						mu.Unlock()
					}
				})
			}
		}
	}
}

type it07MeasureReport struct {
	Commit, Conditions string
	Observations       []string
	Games              []it07MeasureGame
}
type it07MeasureGame struct {
	Chapter, Roster                                                    string
	Deployment                                                         []string
	Route                                                              []tactics.Pos
	Order                                                              []string
	Player                                                             string
	Seed                                                               uint64
	Result                                                             string
	Turns                                                              int
	HitTurnLimit                                                       bool
	Fallen, Deaths                                                     []string
	Kills, MaxLiveEnemies, InitialLiveEnemies, Rewinds                 int
	InitialCommanderAlive, FinalCommanderAlive, FinalCommanderDefeated []string
	ScheduledReinforcements                                            []it07Scheduled
	SpawnedReinforcementTurn                                           map[string]int
	TriggerTurn                                                        map[string]int
	TriggerSide                                                        map[string]string
	EnemyPhaseTurns                                                    []int
	UnspawnedReinforcementIDs                                          []string
	UnspawnedReinforcementDetails                                      []it07UnspawnedDetail
	// #188: 移動経路の ZOC（全試行と最終経路）と、巻き戻しを含む増援の最終経路・全試行の区別。
	PathAllTrials, PathFinal                         it07PathStats
	ReinforcementsAllTrials, ReinforcementsFinal     []string
	ReinforcementsRewoundOnly, UnspawnedFinalPathIDs []string
	InitialUnits, FinalUnits                         []it07UnitSnapshot
	TurnDetails                                      []it07MeasureTurn
}
type it07UnitSnapshot struct {
	ID, Name                    string
	Side                        tactics.Side
	Pos                         tactics.Pos
	HP                          int
	Alive, Commander, Collapsed bool
}
type it07Scheduled struct {
	ID      string
	Turn    int
	Trigger string
}
type it07MeasureTurn struct {
	Turn                                                                                int
	EnemyPhaseStarted                                                                   bool
	MaxLiveEnemies                                                                      int
	Triggers                                                                            []string
	Reinforcements, CommandersSpawned, CommandersAlive, CommandersDefeated              []string
	CollapsedAtCommanderFall                                                            []string
	ZOCEndpointEntries, PlayerZOCEndpointEntries, EnemyZOCEndpointEntries, ZOCExtraCost int
	Attacks                                                                             []it07Attack
	Movements                                                                           []it07Movement
}
type it07Attack struct {
	Unit     string
	Position tactics.Pos
	Target   string
}
type it07Movement struct {
	Unit, Side string
	From, To   tactics.Pos
}
type it07UnspawnedDetail struct {
	ID, Reason                              string
	TriggerSide                             string
	ScheduledTurn, TriggerTurn, AttemptTurn int
	CauseUnknown                            bool
}

func it07UnitSnapshots(units []tactics.Unit) []it07UnitSnapshot {
	out := make([]it07UnitSnapshot, len(units))
	for i, unit := range units {
		out[i] = it07UnitSnapshot{ID: it07ID(unit), Name: unit.Name, Side: unit.Side, Pos: unit.Pos, HP: unit.HP, Alive: unit.Alive, Commander: it07Flag(unit, "Commander"), Collapsed: it07Flag(unit, "Collapsed")}
	}
	return out
}
func it07ID(unit tactics.Unit) string {
	if unit.ID != "" {
		return unit.ID
	}
	return unit.Name
}
func it07Flag(unit tactics.Unit, name string) bool {
	v := reflect.ValueOf(unit).FieldByName(name)
	return v.IsValid() && v.Kind() == reflect.Bool && v.Bool()
}
func it07HasZOC(unit tactics.Unit) bool {
	v := reflect.ValueOf(unit).FieldByName("ClassSkills")
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return false
	}
	for i := 0; i < v.Len(); i++ {
		skill := v.Index(i)
		if skill.Kind() == reflect.Struct {
			name := skill.FieldByName("Name")
			if name.IsValid() && name.Kind() == reflect.String && name.String() == "通せんぼ" {
				return true
			}
			effect := skill.FieldByName("Effect")
			if effect.IsValid() && effect.CanInterface() && strings.Contains(strings.ToLower(fmt.Sprint(effect.Interface())), "zoc") {
				return true
			}
		}
	}
	return false
}
func it07EnteredZOCEndpoint(mover tactics.Unit, units []tactics.Unit) bool {
	for _, enemy := range units {
		if !enemy.Alive || enemy.Side == mover.Side || !it07HasZOC(enemy) {
			continue
		}
		if abs(enemy.Pos.X-mover.Pos.X)+abs(enemy.Pos.Y-mover.Pos.Y) == 1 {
			return true
		}
	}
	return false
}
func it07LiveEnemies(units []tactics.Unit) int {
	n := 0
	for _, u := range units {
		if u.Side == tactics.Enemy && u.Alive {
			n++
		}
	}
	return n
}
func it07ResultName(result tactics.Result) string {
	switch result {
	case tactics.Won:
		return "Won"
	case tactics.Lost:
		return "Lost"
	default:
		return "Ongoing"
	}
}
func it07SideName(side tactics.Side) string {
	if side == tactics.Enemy {
		return "enemy"
	}
	return "player"
}
func it07Commanders(units []tactics.Unit, alive bool) []string {
	var out []string
	for _, unit := range units {
		if it07Flag(unit, "Commander") && unit.Alive == alive {
			out = append(out, it07ID(unit))
		}
	}
	return out
}
func it07Schedules(t *testing.T, cat *defs.Catalog, id string) []it07Scheduled {
	t.Helper()
	m, _ := cat.Map(id)
	setup, err := core.NewSetup(cat, id)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]it07Scheduled, 0, len(m.Reinforcements))
	for i, reinforcement := range m.Reinforcements {
		id := reinforcement.ID
		if id == "" {
			id = setup.Reinforcements[i].Unit.Name
		}
		out = append(out, it07Scheduled{ID: id, Turn: reinforcement.Turn})
	}
	for _, event := range m.Events {
		for _, enemy := range event.Enemies {
			out = append(out, it07Scheduled{ID: enemy.ID, Trigger: event.ID})
		}
	}
	return out
}
func it07Unspawned(scheduled []it07Scheduled, turns map[int]*it07MeasureTurn) []string {
	seen := map[string]bool{}
	for _, turn := range turns {
		for _, id := range turn.Reinforcements {
			seen[id] = true
		}
	}
	var out []string
	for _, item := range scheduled {
		if !seen[item.ID] {
			out = append(out, item.ID)
		}
	}
	return out
}
func it07UnspawnedDetails(scheduled []it07Scheduled, spawned, triggered map[string]int, triggerSides map[string]string, enemyTurns []int, battleTurn int) []it07UnspawnedDetail {
	seen := map[string]bool{}
	for id := range spawned {
		seen[id] = true
	}
	var out []it07UnspawnedDetail
	maxEnemyTurn := 0
	for _, turn := range enemyTurns {
		maxEnemyTurn = max(maxEnemyTurn, turn)
	}
	for _, item := range scheduled {
		if seen[item.ID] {
			continue
		}
		detail := it07UnspawnedDetail{ID: item.ID, ScheduledTurn: item.Turn}
		if item.Trigger == "" {
			switch {
			case item.Turn > maxEnemyTurn || battleTurn < item.Turn:
				detail.Reason = "予定ターン未到達"
			default:
				detail.Reason = "出現試行後に未出現（上限または配置位置の可能性。原因不明）"
				detail.AttemptTurn, detail.CauseUnknown = item.Turn, true
			}
		} else if triggerTurn, ok := triggered[item.Trigger]; !ok {
			detail.Reason = "出来事未発火"
		} else {
			detail.TriggerTurn = triggerTurn
			detail.TriggerSide = triggerSides[item.Trigger]
			attemptTurn := triggerTurn
			if triggerSides[item.Trigger] == "enemy" {
				attemptTurn++
			}
			foundAttempt := false
			for _, turn := range enemyTurns {
				if turn == attemptTurn {
					foundAttempt = true
					break
				}
			}
			if !foundAttempt {
				detail.Reason = "出来事発火後、次の敵フェイズ前に戦闘終了"
			} else {
				detail.Reason = "出来事後の敵フェイズで試行後に未出現（上限または配置位置の可能性。原因不明）"
				detail.AttemptTurn, detail.CauseUnknown = attemptTurn, true
			}
		}
		out = append(out, detail)
	}
	return out
}

// it07Difference は all のうち final にない ID を返す（巻き戻す前だけ出た増援）。
func it07Difference(all, final []string) []string {
	in := map[string]int{}
	for _, id := range final {
		in[id]++
	}
	var out []string
	for _, id := range all {
		if in[id] > 0 {
			in[id]--
			continue
		}
		out = append(out, id)
	}
	return out
}

// it07NotIn は予定のうち spawned に出てこない ID を返す。
func it07NotIn(scheduled []it07Scheduled, spawned []string) []string {
	in := map[string]bool{}
	for _, id := range spawned {
		in[id] = true
	}
	var out []string
	for _, item := range scheduled {
		if !in[item.ID] {
			out = append(out, item.ID)
		}
	}
	return out
}

func appendUniqueInt(values []int, value int) []int {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

// 計測器で見つかった増援の誤分類と両軍 ZOC の取りこぼしを回帰試験する（DES-MAP-09）。
func TestIT07ObservationRules(t *testing.T) {
	t.Run("TC-FR-MAP-04-31 自軍番と敵軍番で発火した増援の試行時機", func(t *testing.T) {
		for _, tc := range []struct {
			side      string
			enemyTurn int
			attempt   bool
		}{
			{"player", 3, true}, {"enemy", 3, false}, {"enemy", 4, true},
		} {
			t.Run(fmt.Sprintf("%s 敵番%d", tc.side, tc.enemyTurn), func(t *testing.T) {
				details := it07UnspawnedDetails([]it07Scheduled{{ID: "wave", Trigger: "alarm"}}, nil, map[string]int{"alarm": 3}, map[string]string{"alarm": tc.side}, []int{tc.enemyTurn}, tc.enemyTurn)
				if len(details) != 1 || details[0].CauseUnknown != tc.attempt {
					t.Fatalf("試行したが未出現か = %v、期待 %v", details, tc.attempt)
				}
			})
		}
	})
	t.Run("TC-FR-MAP-04-31 IDなしの1章増援を生成名で照合できる", func(t *testing.T) {
		scheduled := it07Schedules(t, baseCatalog(t), "ch01")
		seen := map[string]int{}
		for _, r := range scheduled {
			if r.ID == "" {
				t.Fatal("空IDで増援を照合している")
			}
			seen[r.ID] = 3
		}
		if got := it07UnspawnedDetails(scheduled, seen, nil, nil, []int{3, 5}, 6); len(got) != 0 {
			t.Fatalf("出た増援を未出現として数えた: %v", got)
		}
	})
	t.Run("TC-FR-MAP-04-31 両軍のZOC着地と戦死した重騎士", func(t *testing.T) {
		for _, side := range []tactics.Side{tactics.Player, tactics.Enemy} {
			mover := tactics.Unit{Side: side, Alive: true, Pos: tactics.Pos{X: 1}}
			knight := tactics.Unit{Side: 1 - side, Alive: true, Pos: tactics.Pos{X: 2}, ClassSkills: []tactics.Skill{{Name: "通せんぼ"}}}
			if !it07EnteredZOCEndpoint(mover, []tactics.Unit{mover, knight}) {
				t.Fatalf("陣営%dのZOC着地を落とした", side)
			}
			knight.Alive = false
			if it07EnteredZOCEndpoint(mover, []tactics.Unit{mover, knight}) {
				t.Fatal("戦死した重騎士を数えた")
			}
		}
	})
}

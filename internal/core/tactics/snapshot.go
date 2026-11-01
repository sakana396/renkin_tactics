package tactics

import (
	"fmt"
	"math/rand/v2"
)

// phaseStage は自動進行のどこから続けるかを表す（DES-RWD-01）。
type phaseStage int

const (
	playerInput phaseStage = iota
	enemyStart
	enemyAction
	playerStart
)

// phaseProgress は番開始の処理済み範囲と、次の敵・増援前の数を持つ（DES-RWD-01）。
type phaseProgress struct {
	stage      phaseStage
	nextEnemy  int
	enemyCount int
	automatic  bool
}

// battleSnapshot は復元対象だけの深い写し。履歴と次の ID・残り回数は含めない（DES-RWD-01・03）。
// 明示の番終了は playerInput、自動移行は enemyStart を保存する（DES-IT04 7.1 節）。
// 待っている再移動（DES-BTL-10）も値として含める。
// 地形と増援・出来事の定義は New が入力から独立させ、戦闘中に変更しないので共有する。出来事の状態は深く写す（DES-IT06 9.3 節）。
type battleSnapshot struct {
	Turn           int
	Phase          Side
	Result         Result
	units          []Unit
	terrain        [][]Terrain
	reinforcements []ReinforcementSetup
	progress       phaseProgress
	mapEvents      []MapEventSetup
	mapState       eventState
	canto          cantoState
	groupEntered   map[string]bool
	groupPlan      map[int]Pos
	groupDanger    map[int]map[Pos]bool
	rngState       []byte
	eventCount     int
}

// snapshot は PCG の内部状態も値として保存する（DES-RWD-01・03）。
func (b *Battle) snapshot() battleSnapshot {
	// PCG.MarshalBinary は常に nil の誤りを返す。
	data, _ := b.pcg.MarshalBinary()
	return battleSnapshot{
		Turn: b.Turn, Phase: b.Phase, Result: b.Result,
		units: b.Units(), terrain: b.terrain, reinforcements: b.reinforcements,
		progress: b.progress, mapEvents: b.mapEvents, mapState: b.mapState.clone(), canto: b.canto, groupEntered: cloneGroupEntered(b.groupEntered), groupPlan: cloneGroupPlan(b.groupPlan), groupDanger: cloneGroupDanger(b.groupDanger), rngState: append([]byte(nil), data...), eventCount: b.eventCount,
	}
}

// restore は別の PCG の復元成功後に戦闘を置き換える。失敗時は何も変えない（DES-RWD-03、DES-ERR-01）。
// テスト専用の SetRoll の関数差し替えは保存せず、復元後は実 PCG を使う。
// 巻き戻しのテストは固定列を使わず実 PCG で行う。
// 履歴の破棄・回数の消費・ID の検証は、この内部処理を使う公開操作の責務とする。
func (b *Battle) restore(s battleSnapshot) error {
	pcg := new(rand.PCG)
	if err := pcg.UnmarshalBinary(s.rngState); err != nil {
		return fmt.Errorf("乱数の状態を復元できません: %w", err)
	}
	units := make([]Unit, len(s.units))
	for i, u := range s.units {
		units[i] = u.clone()
	}
	b.Turn, b.Phase, b.Result = s.Turn, s.Phase, s.Result
	b.units, b.terrain, b.reinforcements = units, s.terrain, s.reinforcements
	b.progress, b.eventCount = s.progress, s.eventCount
	b.mapEvents, b.mapState = s.mapEvents, s.mapState.clone()
	b.canto = s.canto
	b.groupEntered, b.groupPlan = cloneGroupEntered(s.groupEntered), cloneGroupPlan(s.groupPlan)
	b.groupDanger = cloneGroupDanger(s.groupDanger)
	b.pcg = pcg
	b.rng = rand.New(pcg)
	b.roll = func() int { return b.rng.IntN(100) }
	return nil
}

func cloneGroupEntered(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneGroupPlan(in map[int]Pos) map[int]Pos {
	out := make(map[int]Pos, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneGroupDanger(in map[int]map[Pos]bool) map[int]map[Pos]bool {
	out := make(map[int]map[Pos]bool, len(in))
	for i, danger := range in {
		out[i] = make(map[Pos]bool, len(danger))
		for p, v := range danger {
			out[i][p] = v
		}
	}
	return out
}

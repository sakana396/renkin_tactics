package tactics

import "fmt"

// BoardView は進行操作・乱数・履歴・回数を持たない独立した盤面参照値（DES-RWD-05、DES-DEP-02）。
// Turn・Phase・Result は取得時点の実際の値。変更しても元の戦闘には影響しない。
type BoardView struct {
	Turn               int
	Phase              Side
	Result             Result
	terrain            [][]Terrain
	units              []Unit
	pendingResume      bool
	reinforcementHints []ReinforcementHint
}

// NewBoardView は初期入力の検証と変換だけを行い、番処理や乱数の作成を行わない（DES-RWD-05、DES-DEP-02）。
// 増援は検証するが配置せず、初期ユニットだけを最大 HP・Alive にする。
func NewBoardView(setup Setup) (BoardView, error) {
	if err := validateSetup(setup); err != nil {
		return BoardView{}, err
	}
	units := make([]Unit, len(setup.Units))
	for i, u := range setup.Units {
		units[i] = newUnit(u)
	}
	return BoardView{Turn: 1, Phase: Player, Result: Ongoing, terrain: cloneTerrain(setup.Terrain), units: units}, nil
}

// BoardView は現在の戦闘から独立した参照値を返す（DES-RWD-05）。
func (b *Battle) BoardView() BoardView {
	v := newBoardView(b.terrain, b.units, b.Turn, b.Phase, b.Result, b.NeedsResume())
	v.reinforcementHints = reinforcementHints(v.Width(), v.Height(), b.Turn, b.Result, b.progress, b.reinforcements)
	return v
}

// PreviewRewind は実行済み ID の行動前を、復元・回数消費・乱数への接触なしに返す（DES-RWD-05）。
// 残り 0 回でも閲覧できる。不正・未実行・破棄済みの ID は ErrInvalidRewind を返す。
func (b *Battle) PreviewRewind(id uint64) (BoardView, error) {
	for _, h := range b.history {
		if h.entry.ID != id {
			continue
		}
		if !h.completed {
			break
		}
		s := h.before
		v := newBoardView(s.terrain, s.units, s.Turn, s.Phase, s.Result, s.Result == Ongoing && s.progress.stage != playerInput)
		v.reinforcementHints = reinforcementHints(v.Width(), v.Height(), s.Turn, s.Result, s.progress, s.reinforcements)
		return v, nil
	}
	return BoardView{}, fmt.Errorf("%w: 履歴 ID %d は実行済みの戻り先にありません", ErrInvalidRewind, id)
}

// newBoardView は盤面の可変配列をすべて独立させる（DES-RWD-05）。
func newBoardView(terrain [][]Terrain, units []Unit, turn int, phase Side, result Result, pending bool) BoardView {
	return BoardView{Turn: turn, Phase: phase, Result: result, terrain: cloneTerrain(terrain), units: cloneUnits(units), pendingResume: pending}
}

// NeedsResume は取得時点で保留した自動進行があるかを返す（DES-RWD-05）。
func (v BoardView) NeedsResume() bool { return v.pendingResume }

// Width は盤面の幅。ゼロ値なら 0（DES-RWD-05、DES-DEP-02）。
func (v BoardView) Width() int {
	if len(v.terrain) == 0 {
		return 0
	}
	return len(v.terrain[0])
}

// Height は盤面の高さ（DES-RWD-05、DES-DEP-02）。
func (v BoardView) Height() int { return len(v.terrain) }

func (v BoardView) inMap(p Pos) bool {
	return p.X >= 0 && p.X < v.Width() && p.Y >= 0 && p.Y < v.Height()
}
func (v BoardView) validUnit(i int) bool { return i >= 0 && i < len(v.units) }

// TerrainAt は地形の値を返す。盤外ならゼロ値（DES-RWD-05、DES-MAP-01）。
func (v BoardView) TerrainAt(p Pos) Terrain {
	if !v.inMap(p) {
		return Terrain{}
	}
	return v.terrain[p.Y][p.X]
}

// Units は Items も含めて内部から独立した写しを返す。倒れても添字を保つ（DES-RWD-05）。
func (v BoardView) Units() []Unit { return cloneUnits(v.units) }

// cloneUnits は手持ちを含む深い写しを作る（DES-RWD-05）。
func cloneUnits(units []Unit) []Unit {
	out := make([]Unit, len(units))
	for i, u := range units {
		out[i] = u.clone()
	}
	return out
}

// cloneTerrain は地形の各行を独立させる（DES-RWD-05、DES-DEP-02）。
func cloneTerrain(terrain [][]Terrain) [][]Terrain {
	out := make([][]Terrain, len(terrain))
	for y, row := range terrain {
		out[y] = append([]Terrain(nil), row...)
	}
	return out
}

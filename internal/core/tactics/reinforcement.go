package tactics

import "sort"

// reinforcementRadius は代替出現を探すマンハッタン距離の上限（DES-MAP-03、DES-RNF-01）。
const reinforcementRadius = 2

// reinforcementPosition は距離・Y・X の昇順で立てる空きマスを参照する（DES-MAP-03）。
// 経路や移動力を使わず、倒れたユニットは占有に含めない。状態・乱数は変えない。
func (b *Battle) reinforcementPosition(origin Pos) (Pos, bool) {
	for d := 0; d <= reinforcementRadius; d++ {
		for y := origin.Y - d; y <= origin.Y+d; y++ {
			for x := origin.X - d; x <= origin.X+d; x++ {
				p := Pos{x, y}
				if abs(x-origin.X)+abs(y-origin.Y) != d || !b.inMap(p) || b.TerrainAt(p).Cost < 1 || b.unitAt(p) >= 0 {
					continue
				}
				return p, true
			}
		}
	}
	return Pos{}, false
}

// ReinforcementHint は候補全域を覆う予兆の閉矩形。Min・Max の両端を含む（DES-RNF-01）。
// 正確な出現位置・ターン・人数・ユニット情報は持たない。
type ReinforcementHint struct{ Min, Max Pos }

// ReinforcementHints は内部と独立した予兆領域の列を返す（DES-RNF-01）。
func (v BoardView) ReinforcementHints() []ReinforcementHint {
	return append([]ReinforcementHint(nil), v.reinforcementHints...)
}

// reinforcementHints は自軍開始後から敵軍開始前までの定義だけを領域にする（DES-RNF-01）。
// 地形・占有に依存せず、同一矩形の重複を除き、左上 Y・X、右下 Y・X の順に並べる。
func reinforcementHints(width, height, turn int, result Result, progress phaseProgress, definitions []ReinforcementSetup) []ReinforcementHint {
	if width < 1 || height < 1 || result != Ongoing || (progress.stage != playerInput && progress.stage != enemyStart) {
		return nil
	}
	var hints []ReinforcementHint
	seen := make(map[ReinforcementHint]bool)
	for _, r := range definitions {
		if r.Turn != turn {
			continue
		}
		p := r.Unit.Pos
		h := ReinforcementHint{
			Min: Pos{max(0, p.X-reinforcementRadius) / 4 * 4, max(0, p.Y-reinforcementRadius) / 4 * 4},
			Max: Pos{min(width-1, (min(width-1, p.X+reinforcementRadius)/4+1)*4-1), min(height-1, (min(height-1, p.Y+reinforcementRadius)/4+1)*4-1)},
		}
		if !seen[h] {
			hints = append(hints, h)
			seen[h] = true
		}
	}
	sort.Slice(hints, func(i, j int) bool {
		a, b := hints[i], hints[j]
		if a.Min.Y != b.Min.Y {
			return a.Min.Y < b.Min.Y
		}
		if a.Min.X != b.Min.X {
			return a.Min.X < b.Min.X
		}
		if a.Max.Y != b.Max.Y {
			return a.Max.Y < b.Max.Y
		}
		return a.Max.X < b.Max.X
	})
	return hints
}

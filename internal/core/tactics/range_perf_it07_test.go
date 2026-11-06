package tactics

import (
	"fmt"
	"reflect"
	"testing"
)

// 描画用の射程計算を速くしても、マンハッタン距離による集合と順番を保つ（DES-BTL-01、DES-UI-29）。
func TestRangeCellsEquivalenceIT07(t *testing.T) {
	t.Run("TC-FR-BTL-01-13", func(t *testing.T) {
		for _, size := range [][2]int{{1, 1}, {1, 7}, {7, 1}, {5, 7}, {32, 32}} {
			w, h := size[0], size[1]
			for _, from := range []Pos{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}, {w / 2, h / 2}, {-2, 1}, {w + 1, h / 2}} {
				for _, weapons := range [][]Weapon{nil, {}, {{MinRange: 1, MaxRange: 1}}, {{MinRange: 2, MaxRange: 2}}, {{MinRange: 1, MaxRange: 4}}, {{MinRange: 1, MaxRange: 1}, {MinRange: 4, MaxRange: 5}}, {{MinRange: 1, MaxRange: int(^uint(0) >> 1)}}} {
					var want []Pos
					for y := 0; y < h; y++ {
						for x := 0; x < w; x++ {
							dx, dy := x-from.X, y-from.Y
							if dx < 0 {
								dx = -dx
							}
							if dy < 0 {
								dy = -dy
							}
							for _, weapon := range weapons {
								if weapon.MinRange <= dx+dy && dx+dy <= weapon.MaxRange {
									want = append(want, Pos{x, y})
									break
								}
							}
						}
					}
					if got := rangeCells(w, h, from, weapons); !reflect.DeepEqual(got, want) {
						t.Fatalf("%dx%d from%v weapons%v: got%v want%v", w, h, from, weapons, got, want)
					}
				}
			}
		}
	})
}

// 短い射程に対する盤面の大きさの影響を測る（DES-IT07 9.1節、TC-NFR-01-03）。
func BenchmarkRangeCellsIT07(b *testing.B) {
	for _, size := range []int{32, 256} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			weapons := []Weapon{{MinRange: 1, MaxRange: 1}}
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				if got := rangeCells(size, size, Pos{size / 2, size / 2}, weapons); len(got) != 4 {
					b.Fatal(got)
				}
			}
		})
	}
}

package tactics_test

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// diamond は、平地だけのとき c から距離 mov 以内のマスと費用（距離）を返す。
func diamond(c tactics.Pos, mov, w, h int) map[tactics.Pos]int {
	out := map[tactics.Pos]int{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := abs(x-c.X) + abs(y-c.Y)
			if d <= mov {
				out[pos(x, y)] = d
			}
		}
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// TestReachable は移動できるマスを確かめる（DES-BTL-03、DES-MAP-01）。
func TestReachable(t *testing.T) {
	tests := []struct {
		name  string
		setup tactics.Setup
		unit  int
		want  map[tactics.Pos]int
	}{
		{
			name: "TC-FR-BTL-01-01 平地では距離が移動以下のマスに行ける",
			setup: tactics.Setup{
				Terrain: grid(".......", ".......", ".......", ".......", ".......", ".......", "......."),
				Units:   []tactics.UnitSetup{lord(3, 3, 2), boss(6, 6)},
			},
			unit: 0,
			want: diamond(pos(3, 3), 2, 7, 7),
		},
		{
			name: "TC-FR-BTL-01-03 移動 0 なら今いるマスだけ",
			setup: tactics.Setup{
				Terrain: grid("...", "...", "..."),
				Units:   []tactics.UnitSetup{lord(1, 1, 0), boss(2, 2)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(1, 1): 0},
		},
		{
			name: "TC-FR-BTL-01-03 費用の合計がちょうど移動のマスまで行け、1 つ先には行けない",
			setup: tactics.Setup{
				Terrain: grid("......", "######"),
				Units:   []tactics.UnitSetup{lord(0, 0, 3), boss(5, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(1, 0): 1, pos(2, 0): 2, pos(3, 0): 3},
		},
		{
			name: "TC-FR-BTL-01-04 味方のいるマスは通り抜けられるが止まれない",
			setup: tactics.Setup{
				Terrain: grid("....", "####"),
				Units:   []tactics.UnitSetup{lord(0, 0, 3), ally("味方", 1, 0, 1), boss(3, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(2, 0): 2},
		},
		{
			name: "TC-FR-BTL-01-05 相手の陣営のユニットのいるマスには入れない（細い道）",
			setup: tactics.Setup{
				Terrain: grid(".....", "#####"),
				Units:   []tactics.UnitSetup{lord(0, 0, 4), foe("敵", 2, 0), boss(4, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(1, 0): 1},
		},
		{
			name: "TC-FR-BTL-01-05 相手の陣営のユニットは回り道する",
			setup: tactics.Setup{
				Terrain: grid("...", "...", "..."),
				Units:   []tactics.UnitSetup{lord(0, 1, 2), foe("敵", 1, 1), boss(2, 2)},
			},
			unit: 0,
			// (2, 1) へは (1, 1) を通れないので費用 4 かかり、行けない。
			want: map[tactics.Pos]int{pos(0, 1): 0, pos(0, 0): 1, pos(0, 2): 1, pos(1, 0): 2, pos(1, 2): 2},
		},
		{
			name: "TC-FR-BTL-01-05 敵から見ると自軍が相手で、敵どうしは通り抜けられる",
			setup: tactics.Setup{
				Terrain: grid(".....", "#####"),
				Units: []tactics.UnitSetup{
					func() tactics.UnitSetup { u := boss(0, 0); u.Stats.Mov = 4; return u }(),
					foe("敵", 1, 0), lord(3, 0, 1),
				},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(2, 0): 2},
		},
		{
			name: "TC-FR-MAP-01-01 森は費用 2 で数える",
			setup: tactics.Setup{
				Terrain: grid(".F..", "####"),
				Units:   []tactics.UnitSetup{lord(0, 0, 4), boss(3, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(1, 0): 2, pos(2, 0): 3},
		},
		{
			name: "TC-FR-MAP-01-01 森を通るより回り道が安ければ、回り道の費用になる",
			setup: tactics.Setup{
				Terrain: grid(".FFF.", ".....", "#####", "....."),
				Units:   []tactics.UnitSetup{lord(0, 0, 6), boss(0, 3)},
			},
			unit: 0,
			// (4, 0) は森を通ると 7、下の平地を回ると 6。
			want: map[tactics.Pos]int{
				pos(0, 0): 0, pos(1, 0): 2, pos(2, 0): 4, pos(3, 0): 6, pos(4, 0): 6,
				pos(0, 1): 1, pos(1, 1): 2, pos(2, 1): 3, pos(3, 1): 4, pos(4, 1): 5,
			},
		},
		{
			name: "TC-FR-MAP-01-02 Cost が 0 の地形（壁）には入れず、回り道する",
			setup: tactics.Setup{
				Terrain: grid(".#.", ".#.", "...", "..."),
				Units:   []tactics.UnitSetup{lord(0, 0, 6), boss(0, 3)},
			},
			unit: 0,
			want: map[tactics.Pos]int{
				pos(0, 0): 0, pos(0, 1): 1, pos(0, 2): 2, pos(1, 2): 3, pos(2, 2): 4,
				pos(1, 3): 4, pos(2, 3): 5, pos(2, 1): 5, pos(2, 0): 6,
			},
		},
		{
			name: "TC-FR-MAP-01-03 移動 1 では隣の森（費用 2）に入れない",
			setup: tactics.Setup{
				Terrain: grid(".F", "##"),
				Units:   []tactics.UnitSetup{lord(0, 0, 1), boss(1, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0},
		},
		{
			name: "TC-FR-MAP-01-03 移動 2 なら隣の森（費用 2）に入れる",
			setup: tactics.Setup{
				Terrain: grid(".F.", "###"),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), boss(2, 0)},
			},
			unit: 0,
			want: map[tactics.Pos]int{pos(0, 0): 0, pos(1, 0): 2},
		},
		{
			name: "TC-FR-MAP-01-05 マップの端より外には出ない",
			setup: tactics.Setup{
				Terrain: grid("...", "...", "..."),
				Units:   []tactics.UnitSetup{lord(0, 0, 2), boss(2, 2)},
			},
			unit: 0,
			want: diamond(pos(0, 0), 2, 3, 3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := mustNew(t, tt.setup)
			assertEqual(t, "Reachable", b.Reachable(tt.unit), tt.want)
		})
	}
}

// TestReachableFallenUnit は倒れたユニットの扱いを確かめる（DES-BTL-03）。
func TestReachableFallenUnit(t *testing.T) {
	s := tactics.Setup{
		Terrain: grid("....", "####"),
		Units:   []tactics.UnitSetup{lord(0, 0, 3), foe("敵", 1, 0), ally("味方", 2, 0, 1), boss(3, 0)},
	}
	t.Run("TC-FR-BTL-01-06 倒れたユニットのいたマスは通れて、止まれる", func(t *testing.T) {
		b := mustNew(t, s)
		b.KillUnit(1)
		b.KillUnit(2)
		assertEqual(t, "Reachable", b.Reachable(0), map[tactics.Pos]int{pos(0, 0): 0, pos(1, 0): 1, pos(2, 0): 2})
	})
	t.Run("TC-FR-BTL-01-07 倒れたユニット・存在しないユニットの移動範囲と攻撃範囲は nil", func(t *testing.T) {
		b := mustNew(t, s)
		b.KillUnit(1)
		if got := b.Reachable(1); got != nil {
			t.Errorf("Reachable(倒れた) = %v, want nil", got)
		}
		for _, i := range []int{-1, 4} {
			if got := b.Reachable(i); got != nil {
				t.Errorf("Reachable(%d) = %v, want nil", i, got)
			}
			if got := b.AttackRange(i, pos(0, 0)); got != nil {
				t.Errorf("AttackRange(%d) = %v, want nil", i, got)
			}
		}
	})
}

// TestAttackRange は武器で攻撃できるマスを確かめる（DES-BTL-01）。
func TestAttackRange(t *testing.T) {
	bow := tactics.Weapon{Name: "鉄の弓", Type: tactics.Bow, Might: 6, Hit: 85, MinRange: 2, MaxRange: 2}
	tests := []struct {
		name   string
		weapon *tactics.Weapon
		from   tactics.Pos
		want   []tactics.Pos
	}{
		{
			name: "TC-FR-BTL-01-08 剣（射程 1）は隣の 4 マス",
			from: pos(2, 2),
			want: []tactics.Pos{pos(2, 1), pos(1, 2), pos(3, 2), pos(2, 3)},
		},
		{
			name:   "TC-FR-BTL-01-08 弓（射程 2）は隣を含まず、マップの端で切れる",
			weapon: &bow,
			from:   pos(0, 0),
			want:   []tactics.Pos{pos(2, 0), pos(1, 1), pos(0, 2)},
		},
		{
			name:   "TC-FR-BTL-01-08 今いるマスでない from から測る",
			weapon: &bow,
			from:   pos(4, 4),
			want:   []tactics.Pos{pos(4, 2), pos(3, 3), pos(2, 4)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseSetup()
			if tt.weapon != nil {
				s.Units[0].Weapon = *tt.weapon
			}
			b := mustNew(t, s)
			assertEqual(t, "AttackRange", b.AttackRange(0, tt.from), tt.want)
		})
	}
}

// TestTerrainAt は盤面の大きさと地形を確かめる（DES-BTL-01、DES-MAP-01）。
func TestTerrainAt(t *testing.T) {
	t.Run("TC-FR-MAP-01-04 盤面の大きさと地形、マップの外はゼロ値", func(t *testing.T) {
		b := mustNew(t, tactics.Setup{
			Terrain: grid(".F#", "T.."),
			Units:   []tactics.UnitSetup{lord(0, 0, 1), boss(2, 1)},
		})
		assertEqual(t, "Width", b.Width(), 3)
		assertEqual(t, "Height", b.Height(), 2)
		assertEqual(t, "(1,0)", b.TerrainAt(pos(1, 0)), forest)
		assertEqual(t, "(2,0)", b.TerrainAt(pos(2, 0)), wall)
		assertEqual(t, "(0,1)", b.TerrainAt(pos(0, 1)), fort)
		for _, p := range []tactics.Pos{pos(-1, 0), pos(3, 0), pos(0, 2), pos(0, -1)} {
			assertEqual(t, "外", b.TerrainAt(p), tactics.Terrain{})
		}
	})
}

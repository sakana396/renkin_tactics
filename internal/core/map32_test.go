package core_test

import (
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

func TestMap32Boundary(t *testing.T) {
	t.Run("TC-FR-MAP-02-02", func(t *testing.T) {
		files := fstest.MapFS{}
		names, err := fs.ReadDir(data.Base(), ".")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			b, err := fs.ReadFile(data.Base(), name.Name())
			if err != nil {
				t.Fatal(err)
			}
			files[name.Name()] = &fstest.MapFile{Data: b}
		}
		fixture, err := os.ReadFile("../defload/testdata/map32.toml")
		if err != nil {
			t.Fatal(err)
		}
		files["maps.toml"].Data = append(files["maps.toml"].Data, fixture...)
		files["chapters.toml"].Data = []byte(strings.Replace(string(files["chapters.toml"].Data), `map = "ch01"`, `map = "map32"`, 1))
		cat, err := defload.Load(files)
		if err != nil {
			t.Fatal(err)
		}
		for _, slot := range []int{0, 1, 2} {
			c, err := core.NewCampaign(cat, core.Withdraw)
			if err != nil {
				t.Fatal(err)
			}
			before := campaignSnapshot(c)
			v, err := c.DeploymentView([]int{0}, []int{slot})
			if err != nil {
				t.Fatal(err)
			}
			s, _, err := c.StartBattleAt([]int{0}, []int{slot}, 17)
			if err != nil {
				t.Fatal(err)
			}
			control, _, err := c.StartBattleAt([]int{0}, []int{slot}, 17)
			if err != nil {
				t.Fatal(err)
			}
			b := s.Battle
			positions := []tactics.Pos{{X: 31, Y: 31}, {X: 15, Y: 16}, {X: 31, Y: 15}}
			origin := positions[slot]
			if b.Width() != 32 || b.Height() != 32 || v.Width() != 32 || v.Height() != 32 || b.Units()[0].Pos != origin || !reflect.DeepEqual(v.Units(), b.Units()) {
				t.Fatal("寸法または出撃位置が違う")
			}
			views := []tactics.BoardView{v, b.BoardView()}
			// 全地形を fixture の既知の配置から検査する。画面の 16 マスを境界にしない。
			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					p := tactics.Pos{X: x, Y: y}
					want := tactics.Terrain{Name: "平地", Cost: 1}
					if y == 0 && (x == 0 || x == 31) {
						want = tactics.Terrain{Name: "森", Cost: 2, Avoid: 20, Def: 1}
					}
					if x == 0 && y == 31 {
						want = tactics.Terrain{Name: "壁"}
					}
					if b.TerrainAt(p) != want {
						t.Fatalf("Battle 地形 %v", p)
					}
					for _, view := range views {
						if view.TerrainAt(p) != want {
							t.Fatalf("表示地形 %v", p)
						}
					}
				}
			}
			for _, p := range []tactics.Pos{{X: -1}, {Y: -1}, {X: 32, Y: 31}, {X: 31, Y: 32}} {
				if b.TerrainAt(p) != (tactics.Terrain{}) {
					t.Fatal("盤外地形")
				}
				for _, view := range views {
					if view.TerrainAt(p) != (tactics.Terrain{}) {
						t.Fatal("表示の盤外地形")
					}
				}
			}
			// レオンの移動力 5。周囲は平地で占有がないのでマンハッタン距離がそのまま費用。
			wantReach := map[tactics.Pos]int{}
			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					dx, dy := x-origin.X, y-origin.Y
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					if dx+dy <= 5 {
						wantReach[tactics.Pos{X: x, Y: y}] = dx + dy
					}
				}
			}
			if !reflect.DeepEqual(b.Reachable(0), wantReach) {
				t.Fatalf("移動費: %v", b.Reachable(0))
			}
			wantRange := []tactics.Pos{}
			// y・x 順の隣接マス（剣の射程 1）を独立に列挙する。
			for _, p := range []tactics.Pos{{X: origin.X, Y: origin.Y - 1}, {X: origin.X - 1, Y: origin.Y}, {X: origin.X + 1, Y: origin.Y}, {X: origin.X, Y: origin.Y + 1}} {
				if p.X >= 0 && p.X < 32 && p.Y >= 0 && p.Y < 32 {
					wantRange = append(wantRange, p)
				}
			}
			if !reflect.DeepEqual(b.AttackRange(0, origin), wantRange) {
				t.Fatalf("射程: %v", b.AttackRange(0, origin))
			}
			for _, view := range views {
				if !reflect.DeepEqual(view.Reachable(0), wantReach) || !reflect.DeepEqual(view.AttackRange(0, origin), wantRange) {
					t.Fatal("参照値の範囲が違う")
				}
			}
			returned := v.Reachable(0)
			delete(returned, origin)
			units := v.Units()
			units[0].Pos = tactics.Pos{}
			if !reflect.DeepEqual(v.Reachable(0), wantReach) || v.Units()[0].Pos != origin || !reflect.DeepEqual(*c, before) {
				t.Fatal("取得値が元状態と共有されている")
			}
			to := tactics.Pos{X: origin.X - 1, Y: origin.Y}
			if slot == 1 {
				to.X = 16
			}
			cmd := tactics.Command{Unit: 0, To: to, Action: tactics.Wait}
			ev, err := b.Act(cmd)
			if err != nil {
				t.Fatal(err)
			}
			other, err := control.Battle.Act(cmd)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ev, other) || !reflect.DeepEqual(b.Units(), control.Battle.Units()) {
				t.Fatal("同じ seed・入力の結果が違う")
			}
			if b.Units()[0].Pos != to || b.Turn != 2 || len(b.Units()) != 2 {
				t.Fatal("移動後は2T自軍の番で、増援の出現前であること")
			}
			// DES-MAP-03: 増援は指定ターンの敵軍の番の始めに現れる。
			ev, err = b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			other, err = control.Battle.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ev, other) || !reflect.DeepEqual(b.Units(), control.Battle.Units()) {
				t.Fatal("番終了の再現性")
			}
			if b.Units()[0].Pos != to || b.Turn != 3 || len(b.Units()) != 3 || b.Units()[2].Pos != (tactics.Pos{X: 31, Y: 31}) {
				t.Fatalf("移動と盤端増援: %+v", b.Units())
			}
			if v.Units()[0].Pos != origin || len(v.Units()) != 2 || !reflect.DeepEqual(*c, before) {
				t.Fatal("実行で配置表示・キャンペーンが変わった")
			}
		}
	})
}

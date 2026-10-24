package defload_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
)

func TestMapDimensions(t *testing.T) {
	t.Run("TC-FR-MAP-02-01", func(t *testing.T) {
		// 1×1 は出撃とボスを別々に置けないため、寸法は受理し配置の重複だけを拒む。
		for _, size := range [][2]int{{1, 1}, {1, 32}, {32, 1}, {16, 16}, {17, 17}, {32, 32}, {17, 32}, {32, 17}} {
			t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
				m := baseFS(t)
				row := `"` + strings.Repeat(".", size[0]) + `",`
				boss := strings.ReplaceAll(smallBoss, "x = 1", fmt.Sprintf("x = %d", size[0]-1))
				boss = strings.ReplaceAll(boss, "y = 0", fmt.Sprintf("y = %d", size[1]-1))
				smallMap(strings.Repeat(row, size[1]), boss)(t, m)
				cat, err := defload.Load(m)
				if size == [2]int{1, 1} {
					want := appended(t, "maps.toml: maps[N].enemies[0]: 座標 (0, 0) が maps[N].slots[0] と重なっています")
					if err == nil || err.Error() != want || cat != nil {
						t.Fatalf("1×1 の寸法以外の検証: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				got, _ := cat.Map("small")
				if len(got.Rows) != size[1] || len([]rune(got.Rows[0])) != size[0] {
					t.Fatalf("寸法: %+v", got)
				}
			})
		}
		fixture, err := os.ReadFile("testdata/map32.toml")
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ name, old, new, want string }{
			{"正常", "", "", ""},
			{"幅33", strings.Repeat(".", 32), strings.Repeat(".", 33), "maps[N].rows[1]: 行の長さ 33 は 1 以上 32 以下"},
			{"高さ33", "rows = [", "rows = [\n  \"................................\",", "maps[N].rows: 行の数 33 は 1 以上 32 以下"},
			{"空行", strings.Repeat(".", 32), "", "maps[N].rows[1]: 行の長さ 0 は 1 以上 32 以下"},
			{"不揃い", strings.Repeat(".", 32), strings.Repeat(".", 31), "maps[N].rows[1]: 行の長さ 31 が、rows[0] の長さ 32 と違います"},
			{"未知地形", "F..............................F", "F..............................X", `maps[N].rows[0]: 32 文字目の地形の文字 "X" がありません`},
			{"枠盤外", "x = 31, y = 31", "x = 32, y = 31", "maps[N].slots[0]: 座標 (32, 31) がマップの外です"},
			{"敵盤外", "x = 0\ny = 0", "x = -1\ny = 0", "maps[N].enemies[0]: 座標 (-1, 0) がマップの外です"},
			{"増援盤外", "x = 31\ny = 31", "x = 31\ny = 32", "maps[N].reinforcements[0]: 座標 (31, 32) がマップの外です"},
			{"枠壁", "x = 31, y = 31", "x = 0, y = 31", `maps[N].slots[0]: 座標 (0, 31) の地形 "wall" は通れません`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				m := baseFS(t)
				text := string(fixture)
				if tc.old != "" {
					if !strings.Contains(text, tc.old) {
						t.Fatal("置換元なし")
					}
					text = strings.Replace(text, tc.old, tc.new, 1)
				}
				appendTo("maps.toml", text)(t, m)
				cat, err := defload.Load(m)
				if tc.want != "" {
					if err == nil || !strings.Contains(err.Error(), appended(t, tc.want)) || cat != nil {
						t.Fatalf("期待=%s 誤り=%v", tc.want, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				// 追加の試験用マップによって本体のマップの定義が変わらない。
				original, err := defload.Load(baseFS(t))
				if err != nil {
					t.Fatal(err)
				}
				for _, id := range []string{"ch01", "ch02", "ch03", "ch09"} {
					got, _ := cat.Map(id)
					want, _ := original.Map(id)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("既存 %s が変わった", id)
					}
				}
			})
		}
		t.Run("rune32", func(t *testing.T) {
			m := baseFS(t)
			smallMap(`"`+strings.Repeat("森", 32)+`"`, smallBoss)(t, m)
			appendTo("terrains.toml", "[[terrains]]\nid = \"deep_forest\"\nname = \"深い森\"\nsymbol = \"森\"\ncost = 2")(t, m)
			if _, err := defload.Load(m); err != nil {
				t.Fatal(err)
			}
		})
	})
}

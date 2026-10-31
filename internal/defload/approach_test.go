package defload_test

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 敵の寄り方の定義と一団の検証（TC-FR-BTL-07-21、DES-AI-02）。
func TestLoadApproach(t *testing.T) {
	base := smallBoss + enemyAt("maps.enemies", 3, 3, "group = \"g\"\napproach = \"group\"") +
		enemyAt("maps.reinforcements", 4, 4, "turn = 2\ngroup = \"g\"\napproach = \"group\"")
	t.Run("TC-FR-BTL-07-21 初めの敵と増援の方式", func(t *testing.T) {
		m := baseFS(t)
		smallMap(eventRows, base)(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := cat.Map("small")
		if got.Enemies[1].Approach != defs.ApproachGroup || got.Reinforcements[0].Approach != defs.ApproachGroup {
			t.Fatalf("寄り方が消えた: %+v", got)
		}
	})
	for _, tc := range []struct{ name, extra, want string }{
		{"一団の名前なし", smallBoss + enemyAt("maps.enemies", 3, 3, "approach = \"group\""), "一団の名前"},
		{"方式が不一致", smallBoss + enemyAt("maps.enemies", 3, 3, "group = \"g\"\napproach = \"group\"") + enemyAt("maps.enemies", 4, 4, "group = \"g\""), "寄り方がそろっていません"},
		{"範囲外", smallBoss + enemyAt("maps.enemies", 3, 3, "approach = \"unknown\""), "敵の寄り方"},
	} {
		t.Run("TC-FR-BTL-07-21 "+tc.name, func(t *testing.T) {
			m := baseFS(t)
			smallMap(eventRows, tc.extra)(t, m)
			_, err := defload.Load(m)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

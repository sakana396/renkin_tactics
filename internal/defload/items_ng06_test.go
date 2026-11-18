package defload_test

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 錬金道具の定義の排他と値域を確認する（DES-NG-13）。
func TestNG06Items(t *testing.T) {
	const base = "\n[[items]]\nid = \"test\"\nname = \"試験\"\n"
	for _, tc := range []struct{ name, body, key string }{
		{"effect欠落", "range = 3\namount = 4\n", "effect"},
		{"未知", "effect = \"unknown\"\nrange = 3\n", "effect"},
		{"range欠落", "effect = \"snare\"\n", "range"},
		{"rangeゼロ", "effect = \"snare\"\nrange = 0\n", "range"},
		{"amount欠落", "effect = \"power_tonic\"\nrange = 3\n", "amount"},
		{"amountゼロ", "effect = \"power_tonic\"\nrange = 3\namount = 0\n", "amount"},
		{"粘着にamount", "effect = \"snare\"\nrange = 4\namount = 1\n", "amount"},
		{"回復混在", "effect = \"snare\"\nrange = 4\nheal = 1\n", "heal"},
		{"爆弾混在", "effect = \"snare\"\nrange = 4\nblast_damage = 1\n", "blast_damage"},
	} {
		t.Run("TC-DES-NG-13-02 "+tc.name, func(t *testing.T) {
			m := baseFS(t)
			appendTo("items.toml", base+tc.body)(t, m)
			_, err := defload.Load(m)
			if err == nil || !strings.Contains(err.Error(), "items[1]."+tc.key) {
				t.Fatalf("%s: %v", tc.key, err)
			}
		})
	}
	t.Run("TC-DES-NG-13-03 既存の回復と爆弾", func(t *testing.T) {
		m := baseFS(t)
		appendTo("items.toml", base+"blast_damage = 6\nblast_radius = 1\nthrow_range = 3\n")(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		a, _ := cat.Item("potion")
		b, _ := cat.Item("test")
		if a.Heal != 10 || !b.Bomb || b.BlastDamage != 6 || b.BlastRadius != 1 || b.ThrowRange != 3 {
			t.Fatal(a, b)
		}
	})
}

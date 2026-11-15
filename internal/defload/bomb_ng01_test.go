package defload_test

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 爆弾のアイテム定義の読み込みと検証（DES-NG-01）。期待値は設計メモ（design-ng01.md）の規則から定める。
func TestNG01BombItems(t *testing.T) {
	const bombBase = "[[items]]\nid = \"bomb\"\nname = \"爆弾\"\n"
	for _, tc := range []struct {
		name    string
		extra   string
		wantErr []string
		want    [4]int // Bomb（0/1）、BlastDamage、BlastRadius、ThrowRange
	}{
		{"TC-DES-NG-01-01 3 つがそろえば爆弾として読む", bombBase + "blast_damage = 12\nblast_radius = 1\nthrow_range = 3\n", nil, [4]int{1, 12, 1, 3}},
		{"TC-DES-NG-01-02 半径 0（爆心の 1 マスだけ）も、キーを書けば爆弾", bombBase + "blast_damage = 5\nblast_radius = 0\nthrow_range = 1\n", nil, [4]int{1, 5, 0, 1}},
		{"TC-DES-NG-01-03 heal と同時には書けない", bombBase + "heal = 10\nblast_damage = 12\nblast_radius = 1\nthrow_range = 3\n",
			[]string{"items.toml: items[1].heal: 爆弾（blast_damage・blast_radius・throw_range）は heal を持てません"}, [4]int{}},
		{"TC-DES-NG-01-04 3 つの一部だけは誤り（throw_range がない）", bombBase + "blast_damage = 12\nblast_radius = 1\n",
			[]string{"items.toml: items[1].throw_range: 爆弾は blast_damage・blast_radius・throw_range を 3 つともそろえてください（throw_range が書かれていません）"}, [4]int{}},
		{"TC-DES-NG-01-04 3 つの一部だけは誤り（blast_radius だけ）", bombBase + "blast_radius = 1\n",
			[]string{"items[1].blast_damage: 爆弾は", "items[1].throw_range: 爆弾は"}, [4]int{}},
		{"TC-DES-NG-01-05 blast_damage は 1 以上", bombBase + "blast_damage = 0\nblast_radius = 1\nthrow_range = 3\n",
			[]string{"items.toml: items[1].blast_damage: 1 以上にしてください（0）"}, [4]int{}},
		{"TC-DES-NG-01-05 throw_range は 1 以上", bombBase + "blast_damage = 3\nblast_radius = 1\nthrow_range = 0\n",
			[]string{"items.toml: items[1].throw_range: 1 以上にしてください（0）"}, [4]int{}},
		{"TC-DES-NG-01-05 blast_radius は 0 以上", bombBase + "blast_damage = 3\nblast_radius = -1\nthrow_range = 2\n",
			[]string{"items.toml: items[1].blast_radius: 0 以上にしてください（-1）"}, [4]int{}},
		{"TC-DES-NG-01-06 どれも書かないアイテムは今までどおり heal が 1 以上でなければならない", bombBase,
			[]string{"items.toml: items[1].heal: 1 以上にしてください（0）"}, [4]int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := baseFS(t)
			appendTo("items.toml", tc.extra)(t, m)
			cat, err := defload.Load(m)
			if tc.wantErr != nil {
				if err == nil || cat != nil {
					t.Fatalf("誤りを受理した: %v", err)
				}
				for _, want := range tc.wantErr {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("誤り %q に %q がない", err, want)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			it, ok := cat.Item("bomb")
			if !ok {
				t.Fatal("bomb がない")
			}
			bomb := 0
			if it.Bomb {
				bomb = 1
			}
			if got := [4]int{bomb, it.BlastDamage, it.BlastRadius, it.ThrowRange}; got != tc.want || it.Heal != 0 {
				t.Errorf("読んだ値 %v（heal %d）, 期待 %v（heal 0）", got, it.Heal, tc.want)
			}
		})
	}

	t.Run("TC-DES-NG-01-07 既存の回復薬の書き方は変わらず、爆弾ではない", func(t *testing.T) {
		cat, err := defload.Load(baseFS(t))
		if err != nil {
			t.Fatal(err)
		}
		it, _ := cat.Item("potion")
		if it.Heal != 10 || it.Bomb || it.BlastDamage != 0 || it.BlastRadius != 0 || it.ThrowRange != 0 {
			t.Errorf("回復薬 %+v", it)
		}
	})
}

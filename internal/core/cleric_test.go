package core_test

import (
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 定義の回復量と使用回数を、そのまま戦闘へ渡す（DES-CORE-01、DES-HEL-01）。
func TestStaffConversion(t *testing.T) {
	t.Run("TC-FR-SYS-03-11 杖の種類・回復量・最大使用回数を定義から変換する", func(t *testing.T) {
		for _, tc := range []struct{ heal, uses int }{{10, 15}, {1, 1}, {23, 7}} {
			t.Run(fmt.Sprintf("回復%d・回数%d", tc.heal, tc.uses), func(t *testing.T) {
				files := fstest.MapFS{}
				err := fs.WalkDir(data.Base(), ".", func(p string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					}
					b, err := fs.ReadFile(data.Base(), p)
					if p == "weapons.toml" {
						s := strings.Replace(string(b), "heal = 10", fmt.Sprintf("heal = %d", tc.heal), 1)
						b = []byte(strings.Replace(s, "uses = 15", fmt.Sprintf("uses = %d", tc.uses), 1))
					}
					files[p] = &fstest.MapFile{Data: b}
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				cat, err := defload.Load(files)
				if err != nil {
					t.Fatal(err)
				}
				setup, err := core.NewSetup(cat, "ch01")
				if err != nil {
					t.Fatal(err)
				}
				want := tactics.Weapon{ID: "heal_staff", Name: "癒やしの杖", Type: tactics.Staff, Strikes: 1, Heal: tc.heal, MaxUses: tc.uses, MinRange: 1, MaxRange: 1}
				if got := setup.Units[4].Weapons; len(got) != 1 || !reflect.DeepEqual(got[0], want) {
					t.Errorf("入力の杖 = %+v、期待 %+v", got, want)
				}
				b, _, err := core.NewBattle(cat, "ch01", 1)
				if err != nil {
					t.Fatal(err)
				}
				want.Uses = tc.uses
				if got := b.Units()[4].EquippedWeapon(); !reflect.DeepEqual(got, want) {
					t.Errorf("開始時の杖 = %+v、期待 %+v", got, want)
				}
			})
		}
	})
}

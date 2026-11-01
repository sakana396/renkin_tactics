package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 加入表示と兵種の絵を画面で共通に使う（DES-CMP-11）。
func TestNoelScenesIT06(t *testing.T) {
	t.Run("TC-FR-CMP-08-06 錬金術師から騎兵への加入表示", func(t *testing.T) {
		s := recruitScenes(t, "ch02", core.Withdraw)
		s.beforeStory()
		if s.Scene() != Story {
			t.Fatal(s.Scene())
		}
		s.Cancel()
		if s.Scene() != Joined || !strings.Contains(viewStrings(s.View()), "ノエルが仲間に加わった") {
			t.Fatal(s.Scene(), s.View())
		}
		s.Advance()
		if s.Scene() != Joined || !strings.Contains(viewStrings(s.View()), "エルザが仲間に加わった") {
			t.Fatal(s.Scene(), s.View())
		}
		s.Advance()
		if s.Scene() != Prepare || len(s.SelectedMembers()) != 7 {
			t.Fatal(s.Scene(), s.SelectedMembers())
		}
	})
	t.Run("TC-FR-UNT-02-03 兵種の絵は装備で変わらず魔法の印を出す", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			tile assets.Tile
		}{
			{"錬金術師", assets.Alchemist}, {"魔道士", assets.Mage},
		} {
			u := tactics.Unit{ClassName: tc.name, Side: tactics.Player, Weapons: []tactics.Weapon{{Type: tactics.Magic}, {Type: tactics.Staff}}}
			if got := unitTile(u); got != tc.tile {
				t.Fatal(tc.name, got)
			}
			u.Equipped = 1
			if got := unitTile(u); got != tc.tile {
				t.Fatal(tc.name, got)
			}
			u.Side = tactics.Enemy
			u.Boss = true
			if got := unitTile(u); got != tc.tile {
				t.Fatal(tc.name, got)
			}
		}
		if got, ok := weaponTile(tactics.Magic); !ok || got != assets.WeaponMagic {
			t.Fatal(got, ok)
		}
		imgs, err := assets.Load()
		if err != nil || !assets.Alchemist.Rect().In(imgs[assets.Mages].Bounds()) || !assets.Mage.Rect().In(imgs[assets.Mages].Bounds()) || !assets.WeaponMagic.Rect().In(imgs[assets.Mages].Bounds()) {
			t.Fatal(err)
		}
	})
	t.Run("TC-IF-01-44 9人目まで名簿を送る", func(t *testing.T) {
		s := recruitScenes(t, "ch09", core.Withdraw)
		v := s.View()
		if len(v.Rows) != 8 || v.Rows[0].Member != 0 || v.Rows[7].Member != 7 {
			t.Fatal(v.Rows)
		}
		for _, b := range v.Buttons {
			if b.Label == "▼ 名簿" {
				p := image.Pt(b.Rect.Min.X+1, b.Rect.Min.Y+1)
				s.Click(p.X, p.Y)
				break
			}
		}
		v = s.View()
		if len(v.Rows) != 8 || v.Rows[0].Member != 1 || v.Rows[7].Member != 8 {
			t.Fatal(v.Rows)
		}
		s.Click(v.Rows[7].Rect.Min.X, v.Rows[7].Rect.Min.Y)
		if s.prepareMember != 8 {
			t.Fatal(s.prepareMember)
		}
	})
}

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
		// 名簿の行・印と、名簿や品のボタン・見出しは重ならない（DES-IT06 12.1 節）。
		for _, r := range v.Rows {
			for _, b := range v.Buttons {
				if r.Rect.Overlaps(b.Rect) || r.Mark.Overlaps(b.Rect) {
					t.Fatal("名簿とボタンが重なる", r, b)
				}
			}
		}
		title := v.Texts[0]
		for _, b := range v.Buttons {
			if image.Rect(title.X, title.Y, title.X+textWidth(title.Text), title.Y+lineHeight).Overlaps(b.Rect) {
				t.Fatal("見出しとボタンが重なる", b)
			}
		}
		// 戦死者は送りに数えない。生存者が 8 人なら送りのボタンを出さず、8 人とも表示する。
		s.campaign.Roster[3].Alive = false
		s.enterPrepare()
		v = s.View()
		if len(v.Rows) != 8 || v.Rows[3].Member != 4 || v.Rows[7].Member != 8 {
			t.Fatal(v.Rows)
		}
		for _, b := range v.Buttons {
			if strings.HasSuffix(b.Label, "名簿") {
				t.Fatal("生存者 8 人で送りのボタンがある", b)
			}
		}
	})
}

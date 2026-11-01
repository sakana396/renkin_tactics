package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 準備のボタンで武器と道具を扱い、特効を表示名で案内する（DES-ITM-02、DES-UI-13）。
func TestIT06PrepareWeaponView(t *testing.T) {
	t.Run("TC-FR-CMP-03-14 武器の受け渡し・先頭と初めの装備", func(t *testing.T) {
		s := newScenes(t, &memoryStore{})
		startScenes(s)
		s.Cancel()
		s.campaign.StockWeapons = []core.WeaponCount{{Weapon: "armor_killer", Count: 1}}
		if !strings.Contains(viewStrings(s.View()), "アーマーキラー") {
			t.Fatal("在庫の表示がない")
		}
		clickSceneButton(t, s, "手持ちへ: アーマーキラー")
		if !reflect.DeepEqual(s.campaign.Roster[0].Weapons, []string{"iron_sword", "armor_killer"}) {
			t.Fatal(s.campaign.Roster[0].Weapons)
		}
		clickSceneButton(t, s, "先頭へ: アーマーキラー")
		if got := s.prepareUnit(0).Weapons[0].ID; got != "armor_killer" {
			t.Fatal(got)
		}
		clickSceneButton(t, s, "在庫へ: 鉄の剣")
		if !reflect.DeepEqual(s.campaign.Roster[0].Weapons, []string{"armor_killer"}) {
			t.Fatal(s.campaign.Roster[0].Weapons)
		}
		clickSceneButton(t, s, "在庫へ: アーマーキラー")
		if !strings.Contains(viewStrings(s.View()), core.ErrLastWeapon.Error()) {
			t.Fatal("最後の武器の拒否が見えない")
		}
	})
	t.Run("TC-FR-ITM-01-06 特効の分類を日本語で表示", func(t *testing.T) {
		for _, tc := range []struct{ tag, want string }{{"armor", "特効 鎧"}, {"mounted", "特効 騎馬"}} {
			got := weaponDetail(tactics.Weapon{Name: "特効", Type: tactics.Sword, EffectiveTags: []string{tc.tag}})
			if !strings.Contains(got, tc.want) {
				t.Errorf("%q", got)
			}
		}
	})
}

package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
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
		buttons := s.View().Buttons
		var mark, iron image.Rectangle
		for _, b := range buttons {
			if b.Label == "初めの装備" {
				if b.Enabled {
					t.Fatal("初めの装備を押せる")
				}
				mark = b.Rect
			}
			if b.Label == "在庫へ: 鉄の剣" {
				iron = b.Rect
			}
			if b.Label == "先頭へ: 鉄の剣" {
				t.Fatal("先頭の武器に先頭へがある")
			}
		}
		if mark.Empty() || iron.Empty() || mark.Min.Y != iron.Min.Y {
			t.Fatal("先頭の武器の行に印がない", mark, iron)
		}
		s.Click(mark.Min.X, mark.Min.Y)
		if !reflect.DeepEqual(s.campaign.Roster[0].Weapons, []string{"iron_sword", "armor_killer"}) {
			t.Fatal("初めの装備を押して武器が変わった")
		}
		clickSceneButton(t, s, "先頭へ: アーマーキラー")
		if got := s.prepareUnit(0).Weapons[0].ID; got != "armor_killer" {
			t.Fatal(got)
		}
		mark, iron = image.Rectangle{}, image.Rectangle{}
		for _, b := range s.View().Buttons {
			if b.Label == "初めの装備" {
				mark = b.Rect
			}
			if b.Label == "在庫へ: アーマーキラー" {
				iron = b.Rect
			}
			if b.Label == "先頭へ: アーマーキラー" {
				t.Fatal("新しい先頭の武器に先頭へが残った")
			}
		}
		if mark.Empty() || iron.Empty() || mark.Min.Y != iron.Min.Y {
			t.Fatal("先頭の印が移らない", mark, iron)
		}
		clickSceneButton(t, s, "在庫へ: 鉄の剣")
		if !reflect.DeepEqual(s.campaign.Roster[0].Weapons, []string{"armor_killer"}) {
			t.Fatal(s.campaign.Roster[0].Weapons)
		}
		clickSceneButton(t, s, "在庫へ: アーマーキラー")
		if !strings.Contains(viewStrings(s.View()), core.ErrLastWeapon.Error()) {
			t.Fatal("最後の武器の拒否が見えない")
		}
		for _, b := range s.View().Buttons {
			if strings.HasPrefix(b.Label, "先頭へ:") {
				t.Fatal("武器が 1 本でも先頭へがある")
			}
		}
	})
	t.Run("TC-FR-CMP-03-14 在庫が 5 行を超えると送りのボタンで残りを出す", func(t *testing.T) {
		s := recruitScenes(t, "ch08", core.Withdraw)
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 1}}
		s.campaign.StockWeapons = nil
		for _, id := range []string{"armor_killer", "brave_axe", "hammer", "horse_killer", "killing_sword", "long_bow", "thunder_sword"} {
			count := 1
			if id == "horse_killer" {
				count = 2
			}
			s.campaign.StockWeapons = append(s.campaign.StockWeapons, core.WeaponCount{Weapon: id, Count: count})
		}
		s.message = "案内\n" + strings.Repeat("案内", 20)
		stockLabels := func() ([]string, map[string]bool) {
			var got []string
			enabled := map[string]bool{}
			for _, b := range s.View().Buttons {
				if b.Rect.Min.X == 16 && strings.HasPrefix(b.Label, "手持ちへ: ") {
					if textWidth(b.Label)+12 > b.Rect.Dx() {
						t.Fatalf("在庫のボタンの文字が枠を越す: %q", b.Label)
					}
					got = append(got, strings.TrimPrefix(b.Label, "手持ちへ: "))
				}
				enabled[b.Label] = b.Enabled
			}
			return got, enabled
		}
		got, en := stockLabels()
		if !reflect.DeepEqual(got, []string{"回復薬", "アーマーキラー", "勇者の斧", "ハンマー", "ホースキラー ×2"}) || en["▲ 前へ"] || !en["▼ 次へ"] {
			t.Fatal(got, en)
		}
		if !strings.Contains(viewStrings(s.View()), "1〜5/8") {
			t.Fatal("在庫の位置の表示がない")
		}
		for range 3 {
			clickSceneButton(t, s, "▼ 次へ")
		}
		got, en = stockLabels()
		if !reflect.DeepEqual(got, []string{"ハンマー", "ホースキラー ×2", "キルソード", "長弓", "雷の剣"}) || !en["▲ 前へ"] || en["▼ 次へ"] {
			t.Fatal(got, en)
		}
		// 送ったあとのボタンも、表示した武器を持たせる。
		clickSceneButton(t, s, "手持ちへ: 雷の剣")
		if w := s.campaign.Roster[0].Weapons; w[len(w)-1] != "thunder_sword" {
			t.Fatal(w)
		}
		// 送りのボタン・品のボタン・案内・配置へが重ならず、窓と盤面の幅に収まる。
		v := s.View()
		var rects []image.Rectangle
		for _, b := range v.Buttons {
			rects = append(rects, b.Rect)
		}
		for _, tx := range v.Texts {
			if tx.Style == StyleBad {
				rects = append(rects, image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+16))
			}
		}
		for i, r := range rects {
			if !r.In(image.Rect(0, 0, MapSize, ScreenHeight)) {
				t.Fatal("窓外", r)
			}
			for _, o := range rects[i+1:] {
				if r.Overlaps(o) {
					t.Fatal("重複", r, o)
				}
			}
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

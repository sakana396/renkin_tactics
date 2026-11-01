package core_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
)

// 準備では武器を 1 本ずつ受け渡し、先頭を次の装備に選ぶ（DES-ITM-02、DES-IT06 6.2 節）。
func TestIT06PrepareWeapons(t *testing.T) {
	cat := baseCatalog(t)
	newCampaign := func(t *testing.T) *core.Campaign {
		t.Helper()
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	t.Run("TC-FR-ITM-03-04 過去の報酬は開発用の開始で在庫へ入り同じIDは2本", func(t *testing.T) {
		c := newCampaign(t)
		if len(c.StockWeapons) != 7 {
			t.Fatalf("種類数 %v", c.StockWeapons)
		}
		for _, w := range c.StockWeapons {
			want := 1
			if w.Weapon == "horse_killer" {
				want = 2
			}
			if w.Count != want {
				t.Errorf("%+v", w)
			}
		}
	})
	t.Run("TC-FR-CMP-03-14 持たせる・先頭へ・外す・保存", func(t *testing.T) {
		c := newCampaign(t)
		if err := c.TakeWeapon(0, "killing_sword"); err != nil {
			t.Fatal(err)
		}
		if err := c.TakeWeapon(0, "armor_killer"); err != nil {
			t.Fatal(err)
		}
		if err := c.MoveWeaponFirst(0, 2); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Roster[0].Weapons, []string{"armor_killer", "iron_sword", "killing_sword"}) {
			t.Fatal(c.Roster[0].Weapons)
		}
		s, _, err := c.StartBattleAt([]int{0}, []int{0}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Battle.Units()[0].Weapons[0].ID; got != "armor_killer" {
			t.Fatal(got)
		}
		if err := c.StoreWeapon(0, 2); err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		if err := c.Save(&b); err != nil {
			t.Fatal(err)
		}
		again, err := core.LoadCampaign(cat, &b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Roster[0].Weapons, again.Roster[0].Weapons) || !reflect.DeepEqual(c.StockWeapons, again.StockWeapons) {
			t.Fatal("保存と読込で武器が変わった")
		}
	})
	t.Run("TC-FR-CMP-03-14 同じ ID の 2 本も別の枠として扱う", func(t *testing.T) {
		c := newCampaign(t)
		if err := c.TakeWeapon(1, "horse_killer"); err != nil {
			t.Fatal(err)
		}
		if err := c.TakeWeapon(1, "horse_killer"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.Roster[1].Weapons, []string{"iron_lance", "horse_killer", "horse_killer"}) {
			t.Fatal(c.Roster[1].Weapons)
		}
		for _, w := range c.StockWeapons {
			if w.Weapon == "horse_killer" {
				t.Fatal("2 本を取り出しても在庫に残った")
			}
		}
		if err := c.StoreWeapon(1, 1); err != nil {
			t.Fatal(err)
		}
		for _, w := range c.StockWeapons {
			if w.Weapon == "horse_killer" && w.Count == 1 {
				return
			}
		}
		t.Fatal("1 本を戻しても在庫にない")
	})
	t.Run("TC-FR-CMP-03-14 誤りでは状態不変", func(t *testing.T) {
		c := newCampaign(t)
		tests := []struct {
			name string
			want error
			act  func() error
		}{
			{"在庫なし", core.ErrNoWeaponStock, func() error { return c.TakeWeapon(0, "fire") }},
			{"持てない種類", core.ErrWeaponType, func() error { return c.TakeWeapon(0, "hammer") }},
			{"手持ち外", core.ErrNoHandWeapon, func() error { return c.StoreWeapon(0, 99) }},
			{"先頭の指定外", core.ErrNoHandWeapon, func() error { return c.MoveWeaponFirst(0, -1) }},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				before := append([]string(nil), c.Roster[0].Weapons...)
				stock := append([]core.WeaponCount(nil), c.StockWeapons...)
				if err := tc.act(); !errors.Is(err, tc.want) {
					t.Fatalf("%v", err)
				}
				if !reflect.DeepEqual(before, c.Roster[0].Weapons) || !reflect.DeepEqual(stock, c.StockWeapons) {
					t.Fatal("失敗で変更")
				}
			})
		}
		c.Roster[0].Items[0].Count = 4 // 武器 1 本 ＋ 道具 4 個で満杯。
		before := append([]core.WeaponCount(nil), c.StockWeapons...)
		if err := c.TakeWeapon(0, "killing_sword"); !errors.Is(err, core.ErrHandFull) {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, c.StockWeapons) {
			t.Fatal("在庫が減った")
		}
		c.Roster[0].Items[0].Count = 2
		one, err := core.NewCampaign(cat, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		if err := one.StoreWeapon(0, 0); !errors.Is(err, core.ErrLastWeapon) {
			t.Fatal(err)
		}
	})
}

// 版 1 の移し替えでは攻略済みの章の武器を配り直さない（DES-SAV-05、DES-IT06 6.3 節）。
func TestIT06OldSaveNoRetroReward(t *testing.T) {
	const old = `{"version":1,"rule":"withdraw","chapter":"ch03","roster":[{"character":"leon","level":1,"exp":0,"stats":{"hp":19,"str":7,"skl":11,"spd":12,"def":4,"mov":5},"alive":true}],"stock":[]}`
	t.Run("TC-FR-ITM-03-04 旧セーブの過去の章の武器は配らない", func(t *testing.T) {
		c, err := core.LoadCampaign(baseCatalog(t), bytes.NewBufferString(old))
		if err != nil {
			t.Fatal(err)
		}
		if len(c.StockWeapons) != 0 {
			t.Fatal(c.StockWeapons)
		}
		if !reflect.DeepEqual(c.Roster[0].Weapons, []string{"iron_sword"}) {
			t.Fatal(c.Roster[0].Weapons)
		}
	})
}

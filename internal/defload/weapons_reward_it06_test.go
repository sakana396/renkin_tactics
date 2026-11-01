package defload_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体の新しい武器と章の報酬を承認済みの表と照合する（DES-ITM-02、DES-DAT-04）。
func TestIT06RewardDefinitions(t *testing.T) {
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	weapons := []defs.WeaponDef{
		{ID: "armor_killer", Name: "アーマーキラー", Type: "sword", DamageType: "physical", Might: 4, Hit: 80, Weight: 7, Strikes: 1, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"armor"}},
		{ID: "hammer", Name: "ハンマー", Type: "axe", DamageType: "physical", Might: 7, Hit: 60, Weight: 14, Strikes: 1, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"armor"}},
		{ID: "horse_killer", Name: "ホースキラー", Type: "lance", DamageType: "physical", Might: 6, Hit: 75, Weight: 10, Strikes: 1, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"mounted"}},
		{ID: "killing_sword", Name: "キルソード", Type: "sword", DamageType: "physical", Might: 6, Hit: 85, Crit: 30, Weight: 7, Strikes: 1, MinRange: 1, MaxRange: 1},
		{ID: "brave_axe", Name: "勇者の斧", Type: "axe", DamageType: "physical", Might: 6, Hit: 65, Weight: 14, Strikes: 2, MinRange: 1, MaxRange: 1},
		{ID: "long_bow", Name: "長弓", Type: "bow", DamageType: "physical", Might: 5, Hit: 65, Weight: 9, Strikes: 1, MinRange: 2, MaxRange: 3},
		{ID: "thunder_sword", Name: "雷の剣", Type: "sword", DamageType: "magic", Might: 6, Hit: 80, Weight: 8, Strikes: 1, MinRange: 1, MaxRange: 2},
		{ID: "fire", Name: "ファイア", Type: "magic", DamageType: "magic", Might: 5, Hit: 85, Weight: 2, Strikes: 1, MinRange: 1, MaxRange: 2},
	}
	t.Run("TC-FR-ITM-01-06 武器8種の数値と分類", func(t *testing.T) {
		for _, want := range weapons {
			got, ok := cat.Weapon(want.ID)
			if !ok || !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %+v", want.ID, got)
			}
		}
		for id, tag := range map[string]string{"armor": "armor", "cavalier": "mounted"} {
			cls, _ := cat.Class(id)
			if !reflect.DeepEqual(cls.Tags, []string{tag}) {
				t.Errorf("%s: %v", id, cls.Tags)
			}
		}
	})
	t.Run("TC-FR-ITM-03-04 章の報酬と回復薬の維持", func(t *testing.T) {
		ids := []string{"armor_killer", "hammer", "horse_killer", "killing_sword", "long_bow", "thunder_sword", "brave_axe", "horse_killer"}
		for i, ch := range cat.Chapters() {
			if i >= 8 {
				if ch.Reward != nil {
					t.Errorf("%s: %+v", ch.ID, ch.Reward)
				}
				continue
			}
			want := []defs.ItemCount{{Weapon: ids[i], Count: 1}}
			if i != 2 {
				want = append([]defs.ItemCount{{Item: "potion", Count: 2}}, want...)
			}
			if !reflect.DeepEqual(ch.Reward, want) {
				t.Errorf("%s: %+v", ch.ID, ch.Reward)
			}
		}
	})
}

// 報酬の行と道具専用欄の誤りを識別する（DES-ITM-02、DES-IT06 6.2 節）。
func TestIT06RewardValidation(t *testing.T) {
	cases := []struct{ name, old, new, want string }{
		{"両方", `{ weapon = "armor_killer", count = 1 }`, `{ item = "potion", weapon = "armor_killer", count = 1 }`, "item と weapon の両方"},
		{"空", `{ weapon = "armor_killer", count = 1 }`, `{ count = 1 }`, "item または weapon"},
		{"武器なし", `{ weapon = "armor_killer", count = 1 }`, `{ weapon = "none", count = 1 }`, `武器 "none" がありません`},
		{"個数0", `{ weapon = "armor_killer", count = 1 }`, `{ weapon = "armor_killer", count = 0 }`, "count: 1 以上"},
		{"道具に武器", `items = [{ item = "potion", count = 2 }]`, `items = [{ weapon = "armor_killer", count = 2 }]`, "道具の手持ちに武器は書けません"},
	}
	for _, tc := range cases {
		t.Run("TC-FR-SYS-04-43 "+tc.name, func(t *testing.T) {
			fs := baseFS(t)
			file := "chapters.toml"
			if tc.name == "道具に武器" {
				file = "characters.toml"
			}
			replace(file, tc.old, tc.new)(t, fs)
			_, err := defload.Load(fs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("誤り %v、期待 %q", err, tc.want)
			}
		})
	}
}

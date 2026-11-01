package defload_test

import (
	"reflect"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/defload"
	"testing"
	"testing/fstest"
)

// 複数の武器の種類の読込と拒否を確かめる（DES-CMP-11）。
func TestClassWeaponTypesIT06(t *testing.T) {
	t.Run("TC-FR-UNT-02-03 複数種と従来の単種", func(t *testing.T) {
		cat, err := defload.Load(baseFS(t))
		if err != nil {
			t.Fatal(err)
		}
		cls, _ := cat.Class("alchemist")
		if !cls.CanUseWeapon("magic") || !cls.CanUseWeapon("staff") || cls.CanUseWeapon("sword") {
			t.Fatal(cls)
		}
		lord, _ := cat.Class("lord")
		if !lord.CanUseWeapon("sword") || lord.CanUseWeapon("staff") {
			t.Fatal(lord)
		}
		// 古い書式の weapon_type は 1 要素の WeaponTypes に直す（DES-IT06 5.2 節）。
		if !reflect.DeepEqual(cls.WeaponTypes, []string{"magic", "staff"}) || !reflect.DeepEqual(lord.WeaponTypes, []string{"sword"}) || !reflect.DeepEqual(lord.AllowedWeaponTypes(), []string{"sword"}) {
			t.Fatal(cls.WeaponTypes, lord.WeaponTypes)
		}
		mage, _ := cat.Class("mage")
		fs := baseFS(t)
		const listForm = "id = \"mage\"\nname = \"魔道士\"\nweapon_types = [\"magic\"]"
		before := string(fs["classes.toml"].Data)
		if !strings.Contains(before, listForm) {
			t.Fatal("魔道士が列の書式でない")
		}
		fs["classes.toml"] = &fstest.MapFile{Data: []byte(strings.Replace(before, listForm, "id = \"mage\"\nname = \"魔道士\"\nweapon_type = \"magic\"", 1))}
		old, err := defload.Load(fs)
		if err != nil {
			t.Fatal(err)
		}
		oldMage, _ := old.Class("mage")
		if !reflect.DeepEqual(mage.WeaponTypes, []string{"magic"}) || !reflect.DeepEqual(oldMage, mage) {
			t.Fatal(mage, oldMage)
		}
	})
	for _, tc := range []struct{ name, old, replacement, want string }{
		{"両方の書式", "weapon_types = [\"magic\", \"staff\"]", "weapon_type = \"magic\"\nweapon_types = [\"magic\", \"staff\"]", "weapon_type と weapon_types"},
		{"重複", "weapon_types = [\"magic\", \"staff\"]", "weapon_types = [\"magic\", \"magic\"]", "重なっています"},
		{"未知の種類", "weapon_types = [\"magic\", \"staff\"]", "weapon_types = [\"magic\", \"gun\"]", "weapon_types[1]"},
		{"空の種類", "weapon_types = [\"magic\", \"staff\"]", "weapon_types = [\"magic\", \"\"]", "weapon_types[1]"},
		{"持てない武器", "weapon_types = [\"magic\", \"staff\"]", "weapon_types = [\"magic\", \"sword\"]", "種類 staff"},
	} {
		t.Run("TC-FR-SYS-04-44 "+tc.name, func(t *testing.T) {
			fs := baseFS(t)
			before := string(fs["classes.toml"].Data)
			if !strings.Contains(before, tc.old) {
				t.Fatal("置換の前提がない")
			}
			fs["classes.toml"] = &fstest.MapFile{Data: []byte(strings.Replace(before, tc.old, tc.replacement, 1))}
			_, err := defload.Load(fs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
		})
	}
}

// 加入の段落は DES-IT06 11 章の承認済みの文をそのまま使う（DES-STY-01）。
func TestNoelStoryIT06(t *testing.T) {
	t.Run("TC-FR-CMP-07-08 加入の全文と話す場面", func(t *testing.T) {
		cat, err := defload.Load(baseFS(t))
		if err != nil {
			t.Fatal(err)
		}
		story, ok := cat.Story("ch02_before")
		want := "列のしんがりでは、若い女が追っ手の騎兵に小さな瓶を投げていた。瓶が割れて炎が上がるたび、馬がおびえて足を止める。王都の工房の錬金術士、ノエルと名乗った。ノエル「谷までの道は、私が焼いて開ける。薬の心得もあるから、けが人は任せて」"
		if !ok || len(story.Paragraphs) != 6 || story.Paragraphs[1] != want {
			t.Fatal(story)
		}
		for _, chapter := range cat.Chapters() {
			for _, id := range []string{chapter.Before, chapter.After} {
				if id == "ch02_before" {
					continue
				}
				other, _ := cat.Story(id)
				if strings.Contains(strings.Join(other.Paragraphs, ""), "ノエル「") {
					t.Fatal(id)
				}
			}
		}
	})
}

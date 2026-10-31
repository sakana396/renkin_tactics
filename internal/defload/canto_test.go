package defload_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 兵種のスキル（再移動）の定義の試験（DES-IT06 8.1 節、DES-DAT-04、#124）。

func TestClassSkills(t *testing.T) {
	t.Run("TC-FR-UNT-07-08 本体のデータ: 騎兵だけが再移動を持ち、エルザのキャラのスキルは空", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		sk, ok := cat.Skill("canto")
		if want := (defs.SkillDef{ID: "canto", Name: "再移動", Effect: defs.EffectCanto}); !ok || sk != want {
			t.Errorf("スキル canto = %+v, %v（want %+v）", sk, ok, want)
		}
		for _, id := range []string{"lord", "armor", "fighter", "archer", "soldier", "myrmidon", "cleric"} {
			if c, _ := cat.Class(id); len(c.Skills) != 0 {
				t.Errorf("クラス %s の兵種のスキル = %v（want 空）", id, c.Skills)
			}
		}
		c, _ := cat.Class("cavalier")
		if !reflect.DeepEqual(c.Skills, []string{"canto"}) {
			t.Errorf("騎兵の兵種のスキル = %v", c.Skills)
		}
		// 返した値を書き換えても目録は変わらない。
		c.Skills[0] = "wrath"
		if c2, _ := cat.Class("cavalier"); c2.Skills[0] != "canto" {
			t.Errorf("目録の兵種のスキルが書き換わった: %v", c2.Skills)
		}
		if ch, _ := cat.Character("elsa"); ch.Class != "cavalier" || ch.Skill != "" {
			t.Errorf("エルザ = クラス %q・スキル %q（want cavalier・空）", ch.Class, ch.Skill)
		}
	})

	tests := []struct {
		name  string
		edits []edit
		want  []string
	}{
		{"TC-FR-UNT-07-08 兵種のスキルを 2 つ持てる（キャラのスキルとも重ねられる）",
			[]edit{replace("classes.toml", `skills = ["canto"]`, `skills = ["canto", "wrath"]`)},
			nil},
		{"TC-FR-UNT-07-08 兵種のスキルの参照がない",
			[]edit{replace("classes.toml", `skills = ["canto"]`, `skills = ["gallop"]`)},
			[]string{`classes.toml: classes[6].skills[0]: スキル "gallop" がありません`}},
		{"TC-FR-UNT-07-08 兵種のスキルが重なる",
			[]edit{replace("classes.toml", `skills = ["canto"]`, `skills = ["canto", "canto"]`)},
			[]string{`classes.toml: classes[6].skills[1]: 兵種のスキル "canto" が重なっています`}},
		{"TC-FR-UNT-07-08 再移動の効果の種類が決まった値でない",
			[]edit{replace("skills.toml", `effect = "canto"`, `effect = "gallop"`)},
			[]string{`skills.toml: skills[4].effect: スキルの効果の種類 "gallop" は使えません（inspire・pavise・wrath・snipe・canto のどれか）`}},
		{"TC-FR-UNT-07-08 兵種のスキルの型が文字列の列でない",
			[]edit{replace("classes.toml", `skills = ["canto"]`, `skills = "canto"`)},
			[]string{`classes.toml`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseFS(t)
			for _, e := range tt.edits {
				e(t, m)
			}
			cat, err := defload.Load(m)
			if len(tt.want) == 0 {
				if err != nil {
					t.Fatalf("誤りがないはずが、誤りがある:\n%v", err)
				}
				if c, _ := cat.Class("cavalier"); !reflect.DeepEqual(c.Skills, []string{"canto", "wrath"}) {
					t.Errorf("騎兵の兵種のスキル = %v", c.Skills)
				}
				return
			}
			if err == nil || cat != nil {
				t.Fatalf("誤りがあるはずが、誤りがない（期待: %q）", tt.want)
			}
			var errs defload.Errors
			if !errors.As(err, &errs) {
				// 型の誤りは TOML の読み込みの誤りとして返る。ファイル名が分かればよい。
				if !strings.Contains(err.Error(), tt.want[0]) {
					t.Fatalf("誤り %v にファイル名 %q がない", err, tt.want[0])
				}
				return
			}
			for _, w := range tt.want {
				found := false
				for _, d := range errs {
					if strings.Contains(d.Error(), w) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("誤り %q が見つからない。誤りの一覧:\n%v", w, err)
				}
			}
		})
	}
}

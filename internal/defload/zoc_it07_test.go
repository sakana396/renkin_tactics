package defload_test

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

func TestZOCDefinition(t *testing.T) {
	t.Run("TC-FR-SYS-04-45 本体の通せんぼを受理する", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		sk, ok := cat.Skill("zoc")
		if !ok || sk != (defs.SkillDef{ID: "zoc", Name: "通せんぼ", Effect: defs.EffectZOC}) {
			t.Errorf("通せんぼ = %+v, %v", sk, ok)
		}
		armor, _ := cat.Class("armor")
		if len(armor.Skills) != 1 || armor.Skills[0] != "zoc" {
			t.Errorf("アーマーの兵種のスキル = %v", armor.Skills)
		}
	})
	cases := []struct {
		name  string
		edits []edit
		want  string
	}{
		{"数値を付ける", []edit{replace("skills.toml", "effect = \"zoc\"", "effect = \"zoc\"\nhit = 1")}, "通せんぼには数値を付けられません"},
		{"効果名を誤る", []edit{replace("skills.toml", "effect = \"zoc\"", "effect = \"unknown\"")}, "スキルの効果の種類"},
		{"同じスキルを二度書く", []edit{replace("classes.toml", "skills = [\"zoc\"]", "skills = [\"zoc\", \"zoc\"]")}, "重なっています"},
		{"同じ効果の別のスキルを重ねる", []edit{
			replace("skills.toml", "effect = \"zoc\"", "effect = \"zoc\"\n\n[[skills]]\nid = \"zoc2\"\nname = \"通せんぼ二\"\neffect = \"zoc\""),
			replace("classes.toml", "skills = [\"zoc\"]", "skills = [\"zoc\", \"zoc2\"]"),
		}, "兵種のスキルの効果 \"zoc\" が重なっています"},
		{"知らない参照", []edit{replace("classes.toml", "skills = [\"zoc\"]", "skills = [\"unknown\"]")}, "スキル \"unknown\" がありません"},
	}
	for _, tc := range cases {
		t.Run("TC-FR-SYS-04-45 "+tc.name, func(t *testing.T) {
			m := baseFS(t)
			for _, e := range tc.edits {
				e(t, m)
			}
			_, err := defload.Load(m)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("誤り = %v, want %q", err, tc.want)
			}
		})
	}
}

package defload_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

func TestCraftingDefinitionsNG07(t *testing.T) {
	t.Run("TC-DES-NG-18-01 新作の材料とレシピ", func(t *testing.T) {
		c, err := defload.Load(data.Alchemy())
		if err != nil {
			t.Fatal(err)
		}
		ms := c.Materials()
		if len(ms) != 2 || ms[0].ID != "ore" || ms[1].ID != "catalyst" {
			t.Fatal(ms)
		}
		want := map[string][2]int{"bomb_small": {1, 1}, "bomb_far": {2, 1}, "bomb_large": {2, 2}, "potion": {1, 1}, "tonic_power": {1, 2}, "snare_bottle": {2, 1}}
		if len(c.Recipes()) != len(want) {
			t.Fatal(c.Recipes())
		}
		for id, counts := range want {
			r, ok := c.Recipe(id)
			if !ok || r.Item != id || r.Count != 1 || len(r.Ingredients) != 2 || r.Ingredients[0].Material != "ore" || r.Ingredients[0].Count != counts[0] || r.Ingredients[1].Material != "catalyst" || r.Ingredients[1].Count != counts[1] {
				t.Fatalf("%s: %+v", id, r)
			}
		}
	})
	t.Run("TC-DES-NG-18-03 旧作", func(t *testing.T) {
		c, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Materials()) != 0 || len(c.Recipes()) != 0 {
			t.Fatal(c.Materials(), c.Recipes())
		}
	})
	const material = "[[materials]]\nid='ore'\nname='鉱粉'\n"
	const recipe = "[[recipes]]\nid='r'\nname='レシピ'\nitem='potion'\ncount=1\ningredients=[{material='ore',count=1}]\n"
	for _, tc := range []struct{ name, mats, recipes, reward, location string }{
		{"材料の空 ID", strings.Replace(material, "id='ore'", "id=''", 1), recipe, "", "materials[0].id"},
		{"材料の重複", material + material, recipe, "", "materials[1].id"},
		{"材料の空名", strings.Replace(material, "name='鉱粉'", "name=''", 1), recipe, "", "materials[0].name"},
		{"レシピの空 ID", material, strings.Replace(recipe, "id='r'", "id=''", 1), "", "recipes[0].id"},
		{"レシピの重複", material, recipe + recipe, "", "recipes[1].id"},
		{"レシピの空名", material, strings.Replace(recipe, "name='レシピ'", "name=''", 1), "", "recipes[0].name"},
		{"未知の出力", material, strings.Replace(recipe, "item='potion'", "item='unknown'", 1), "", "recipes[0].item"},
		{"生成数", material, strings.Replace(recipe, "count=1\ningredients", "count=2\ningredients", 1), "", "recipes[0].count"},
		{"材料なし", material, strings.Replace(recipe, "ingredients=[{material='ore',count=1}]", "ingredients=[]", 1), "", "recipes[0].ingredients"},
		{"未知の材料", material, strings.Replace(recipe, "material='ore'", "material='bad'", 1), "", "recipes[0].ingredients[0].material"},
		{"必要数ゼロ", material, strings.Replace(recipe, "material='ore',count=1", "material='ore',count=0", 1), "", "recipes[0].ingredients[0].count"},
		{"必要数負", material, strings.Replace(recipe, "material='ore',count=1", "material='ore',count=-1", 1), "", "recipes[0].ingredients[0].count"},
		{"材料重複", material, strings.Replace(recipe, "{material='ore',count=1}]", "{material='ore',count=1},{material='ore',count=1}]", 1), "", "recipes[0].ingredients[1].material"},
		{"報酬未知", material, recipe, "{material='bad',count=1}", "material"},
		{"報酬混在", material, recipe, "{material='ore',item='potion',count=1}", "reward"},
		{"報酬ゼロ", material, recipe, "{material='ore',count=0}", "count"},
		{"報酬負", material, recipe, "{material='ore',count=-1}", "count"},
		{"報酬重複", material, recipe, "{material='ore',count=1},{material='ore',count=1}", "material"},
	} {
		t.Run("TC-DES-NG-18-02 "+tc.name, func(t *testing.T) {
			m := baseFS(t)
			m["materials.toml"] = &fstest.MapFile{Data: []byte(tc.mats)}
			m["recipes.toml"] = &fstest.MapFile{Data: []byte(tc.recipes)}
			if tc.reward != "" {
				appendTo("chapters.toml", "\n[[chapters]]\nid='extra'\nname='追加'\nmap='map1'\nreward=["+tc.reward+"]\n")(t, m)
			}
			_, err := defload.Load(m)
			if err == nil || !strings.Contains(err.Error(), tc.location) || !strings.Contains(err.Error(), ".toml:") {
				t.Fatalf("%s: %v", tc.location, err)
			}
		})
	}
}

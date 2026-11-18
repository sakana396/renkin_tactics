package ui

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defs"
)

func TestCraftWindowNG07(t *testing.T) {
	t.Run("TC-DES-NG-20-01 不足と失敗を表示し閉じる", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.campaign.Materials = map[string]int{"ore": 0, "catalyst": 0}
		b := s.prepareHiResView().Craft
		if !b.Enabled {
			t.Fatal("調合ボタンがない")
		}
		s.clickPrepareHiRes(b.Rect.Min.X+1, b.Rect.Min.Y+1)
		w := s.craftWindow()
		if len(w.Rows) == 0 {
			t.Fatal("レシピがない")
		}
		var lines []string
		for _, v := range w.Texts {
			lines = append(lines, v.Text)
		}
		if !strings.Contains(strings.Join(lines, " "), "不足") {
			t.Fatal(lines)
		}
		stock := append(s.campaign.Stock[:0:0], s.campaign.Stock...)
		materials := map[string]int{"ore": 0, "catalyst": 0}
		r := w.Rows[0].Rect
		s.clickPrepareHiRes(r.Min.X+1, r.Min.Y+1)
		s.makeCraft()
		if s.craftMessage == "" || !reflect.DeepEqual(stock, s.campaign.Stock) || !reflect.DeepEqual(materials, s.campaign.Materials) {
			t.Fatal("失敗時の状態", s.craftMessage)
		}
		s.cancelPrepareHiRes()
		if s.craftOpen || s.Scene() != Prepare {
			t.Fatal("窓が閉じない")
		}
	})
	t.Run("TC-DES-NG-20-02 背後の操作を遮り押下ごとに一個", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.campaign.Materials = map[string]int{"ore": 10, "catalyst": 10}
		b := s.prepareHiResView().Craft
		s.clickPrepareHiRes(b.Rect.Min.X+1, b.Rect.Min.Y+1)
		w := s.craftWindow()
		r := w.Rows[0].Rect
		s.clickPrepareHiRes(r.Min.X+1, r.Min.Y+1)
		recipe := s.cfg.Catalog.Recipes()[0]
		before := s.campaign.Materials[recipe.Ingredients[0].Material]
		s.clickPrepareHiRes(w.Make.Rect.Min.X+1, w.Make.Rect.Min.Y+1)
		if got := s.campaign.Materials[recipe.Ingredients[0].Material]; got != before-recipe.Ingredients[0].Count {
			t.Fatal(got)
		}
		s.clickPrepareHiRes(hrDeployRect.Min.X+1, hrDeployRect.Min.Y+1)
		if s.Scene() != Prepare || !s.craftOpen {
			t.Fatal("背後の配置へ伝播")
		}
		selected := s.prepareItem
		if !s.rightClickPrepare(0, 0) || s.craftOpen || s.prepareItem != selected {
			t.Fatal("右クリックが伝播")
		}
		s.clickPrepareHiRes(b.Rect.Min.X+1, b.Rect.Min.Y+1)
		s.cancelPrepareHiRes()
		if s.craftOpen || s.prepareItem != selected {
			t.Fatal("取消しが伝播")
		}
	})
}

func TestCraftReturnFromDeployNG07(t *testing.T) {
	t.Run("TC-DES-NG-19-04 調合後に配置から戻っても在庫を保持", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.campaign.Materials = map[string]int{"ore": 5, "catalyst": 4}
		button := s.prepareHiResView().Craft
		s.clickPrepareHiRes(button.Rect.Min.X+1, button.Rect.Min.Y+1)
		w := s.craftWindow()
		recipes := s.cfg.Catalog.Recipes()
		if len(w.Rows) == 0 || len(recipes) == 0 {
			t.Fatal("レシピがない")
		}
		recipe := recipes[w.Rows[0].Index]
		count := func() int {
			for _, item := range s.campaign.Stock {
				if item.Item == recipe.Item {
					return item.Count
				}
			}
			return 0
		}
		before := count()
		row := w.Rows[0].Rect
		s.clickPrepareHiRes(row.Min.X+1, row.Min.Y+1)
		s.clickPrepareHiRes(w.Make.Rect.Min.X+1, w.Make.Rect.Min.Y+1)
		if count() != before+1 {
			t.Fatal("調合で道具が増えない", s.craftMessage)
		}
		expected := map[string]int{"ore": 5, "catalyst": 4}
		for _, ingredient := range recipe.Ingredients {
			expected[ingredient.Material] -= ingredient.Count
		}
		if !reflect.DeepEqual(expected, s.campaign.Materials) {
			t.Fatal("材料の消費", s.campaign.Materials)
		}
		s.Cancel()
		s.clickPrepareHiRes(hrDeployRect.Min.X+1, hrDeployRect.Min.Y+1)
		if s.Scene() != Deploy {
			t.Fatal("配置に進めない", s.Scene())
		}
		s.Cancel()
		if s.Scene() != Prepare || count() != before+1 || !reflect.DeepEqual(expected, s.campaign.Materials) {
			t.Fatal("配置から戻った状態", s.Scene(), s.campaign.Materials, s.campaign.Stock)
		}
	})
}

func TestCraftLongMaterialsNG07(t *testing.T) {
	t.Run("TC-DES-NG-20-03 長い名前と多数の材料を送り表示", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		recipes := s.cfg.Catalog.Recipes()
		recipe := recipes[0]
		recipe.Ingredients = nil
		var materials []defs.MaterialDef
		for i := 0; i < 30; i++ {
			id := fmt.Sprintf("material_%d", i)
			name := strings.Repeat("長い材料名", 15) + fmt.Sprintf("末尾%d終", i)
			materials = append(materials, defs.MaterialDef{ID: id, Name: name})
			recipe.Ingredients = append(recipe.Ingredients, defs.MaterialCount{Material: id, Count: i + 1})
		}
		s.cfg.Catalog.SetAlchemy(materials, []defs.RecipeDef{recipe})
		lines := s.craftLines()
		var joined strings.Builder
		for _, line := range lines {
			joined.WriteString(line.text)
			if textWidth(line.text) > 744 {
				t.Fatal("文字が欄をはみ出す", line)
			}
		}
		for i, material := range materials {
			if strings.Count(joined.String(), material.Name) != 2 || !strings.Contains(joined.String(), fmt.Sprintf("不足 %d", i+1)) {
				t.Fatal("材料または不足数が欠けた", i)
			}
		}
		win := image.Rect(120, 92, 968, 740)
		seen := map[string]bool{}
		for {
			v := s.craftWindow()
			for _, row := range v.Rows {
				if !row.Rect.In(win) {
					t.Fatal("行が窓外", row)
				}
			}
			for _, text := range v.Texts {
				seen[text.Text] = true
			}
			if !v.Down.Enabled {
				break
			}
			s.clickCraft(image.Pt(v.Down.Rect.Min.X+1, v.Down.Rect.Min.Y+1), v)
		}
		for _, line := range lines {
			if !seen[line.text] {
				t.Fatal("送りで表示できない", line)
			}
		}
	})
}

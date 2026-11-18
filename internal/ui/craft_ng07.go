package ui

import (
	"fmt"
	"image"
	"image/color"
)

// craftView は調合窓の描画と入力に共通する値（DES-NG-20）。
type craftView struct {
	Boxes                 []hrBox
	Texts                 []hrTextItem
	Glyphs                []hrGlyph
	Rows                  []craftRow
	Up, Down, Make, Close SceneButton
}
type craftRow struct {
	Rect  image.Rectangle
	Index int
}

const craftRows = 24

type craftLine struct {
	text    string
	recipe  int
	heading bool
}

// craftLines は長い名前や多数の材料も省略せず、送り可能な行に分ける（DES-NG-20）。
func (s *SceneController) craftLines() []craftLine {
	var lines []craftLine
	add := func(text string, recipe int, heading bool) {
		for _, line := range wrapSceneTextWidth(text, 744) {
			lines = append(lines, craftLine{text, recipe, heading})
			lines[len(lines)-1].text = line
		}
	}
	add("材料", -1, true)
	for _, m := range s.cfg.Catalog.Materials() {
		add(fmt.Sprintf("%s　所持 %d", m.Name, s.campaign.Materials[m.ID]), -1, false)
	}
	for i, r := range s.cfg.Catalog.Recipes() {
		item, _ := s.cfg.Catalog.Item(r.Item)
		add(item.Name, i, true)
		add(itemDetailText(item), i, false)
		for _, need := range r.Ingredients {
			material, _ := s.cfg.Catalog.Material(need.Material)
			missing := max(need.Count-s.campaign.Materials[need.Material], 0)
			add(fmt.Sprintf("必要 %s %d（不足 %d）", material.Name, need.Count, missing), i, false)
		}
	}
	return lines
}

// craftWindow は材料・レシピ・不足数を準備画面の上に表示する（DES-NG-20）。
func (s *SceneController) craftWindow() craftView {
	var v craftView
	box := func(r image.Rectangle, k hrBoxKind) { v.Boxes = append(v.Boxes, hrBox{r, k}) }
	txt := func(t string, x, y int, c color.NRGBA) {
		v.Texts = append(v.Texts, hrTextItem{Text: t, X: x, Y: y, Scale: 1, Color: c})
	}
	panel := image.Rect(120, 92, 968, 740)
	box(panel, hrBoxPanel)
	txt("調合", 148, 110, hrAmber)
	lines := s.craftLines()
	recipes := s.cfg.Catalog.Recipes()
	s.craftTop = min(max(s.craftTop, 0), max(len(lines)-craftRows, 0))
	v.Up = SceneButton{Label: "前へ", Rect: image.Rect(760, 108, 844, 136), Enabled: s.craftTop > 0}
	v.Down = SceneButton{Label: "次へ", Rect: image.Rect(850, 108, 940, 136), Enabled: s.craftTop+craftRows < len(lines)}
	for _, b := range []SceneButton{v.Up, v.Down} {
		k, c := hrBoxButton, hrDark
		if !b.Enabled {
			k, c = hrBoxButtonOff, hrDim
		}
		box(b.Rect, k)
		txt(b.Label, b.Rect.Min.X+25, b.Rect.Min.Y+6, c)
		kind := hrGlyphUp
		if b.Rect == v.Down.Rect {
			kind = hrGlyphDown
		}
		v.Glyphs = append(v.Glyphs, hrGlyph{Kind: kind, Rect: image.Rect(b.Rect.Min.X+7, b.Rect.Min.Y+9, b.Rect.Min.X+19, b.Rect.Min.Y+19), Color: c})
	}
	for i := s.craftTop; i < min(len(lines), s.craftTop+craftRows); i++ {
		line := lines[i]
		y := 150 + (i-s.craftTop)*20
		rect := image.Rect(148, y, 940, y+20)
		if line.recipe >= 0 {
			kind := hrBoxRow
			if line.recipe == s.craftSelected {
				kind = hrBoxRowSel
			}
			box(rect, kind)
			v.Rows = append(v.Rows, craftRow{rect, line.recipe})
		}
		c := hrText
		if line.heading {
			c = hrAmber
		}
		txt(line.text, 164, y+2, c)
	}
	v.Make = SceneButton{Label: "決定", Rect: image.Rect(642, 666, 774, 714), Enabled: s.craftSelected >= 0 && s.craftSelected < len(recipes)}
	v.Close = SceneButton{Label: "閉じる", Rect: image.Rect(796, 666, 940, 714), Enabled: true}
	for _, b := range []SceneButton{v.Make, v.Close} {
		k := hrBoxButton
		c := hrDark
		if !b.Enabled {
			k = hrBoxButtonOff
			c = hrDim
		}
		box(b.Rect, k)
		txt(b.Label, b.Rect.Min.X+25, b.Rect.Min.Y+15, c)
	}
	messageColor := hrBad
	if s.craftMessage == "調合しました" {
		messageColor = hrText
	}
	txt(fitText(s.craftMessage, 460), 148, 681, messageColor)
	return v
}
func (s *SceneController) closeCraft() { s.craftOpen = false; s.craftMessage = "" }
func (s *SceneController) clickCraft(pt image.Point, v craftView) {
	switch {
	case pt.In(v.Close.Rect):
		s.closeCraft()
	case pt.In(v.Make.Rect):
		s.makeCraft()
	case pt.In(v.Up.Rect):
		if v.Up.Enabled {
			s.craftTop--
		}
	case pt.In(v.Down.Rect):
		if v.Down.Enabled {
			s.craftTop++
		}
	default:
		for _, r := range v.Rows {
			if pt.In(r.Rect) {
				s.craftSelected = r.Index
				s.craftMessage = ""
				return
			}
		}
	}
}
func (s *SceneController) makeCraft() {
	recipes := s.cfg.Catalog.Recipes()
	if s.craftSelected < 0 || s.craftSelected >= len(recipes) {
		s.craftMessage = "レシピを選んでください"
		return
	}
	if err := s.campaign.Craft(recipes[s.craftSelected].ID); err != nil {
		s.craftMessage = err.Error()
		return
	}
	s.craftMessage = "調合しました"
}
func (s *SceneController) moveCraft(delta int) {
	n := len(s.cfg.Catalog.Recipes())
	if n == 0 {
		return
	}
	s.craftSelected = min(max(s.craftSelected+delta, 0), n-1)
	for i, line := range s.craftLines() {
		if line.recipe != s.craftSelected {
			continue
		}
		if i < s.craftTop {
			s.craftTop = i
		}
		if i >= s.craftTop+craftRows {
			s.craftTop = i - craftRows + 1
		}
		break
	}
}

package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 準備の画面の作り直し（高解像度。DES-NG-11、設計メモ DES-NG05）。
// Config.HiRes のときだけ、この経路を通る。HiRes でない準備（prepare.go）は変えない。
// 窓の全体（1088×832）に描き、入力の座標も窓の座標のまま使う。

// 領域（DES-NG05 2 章。右・下の端を含まない）。
var (
	hrHeader       = image.Rect(0, 0, 1088, 84)
	hrRosterPanel  = image.Rect(20, 100, 348, 740)
	hrCenterPanel  = image.Rect(364, 100, 724, 740)
	hrHandPanel    = image.Rect(740, 100, 1068, 290)
	hrStockPanel   = image.Rect(740, 300, 1068, 430)
	hrChapterPanel = image.Rect(740, 440, 1068, 740)
	hrFooter       = image.Rect(0, 752, 1088, 832)
	hrDeployRect   = image.Rect(850, 764, 1060, 818)
	hrCountRect    = image.Rect(860, 22, 1060, 64)
	// hrPortraitRect は中央のカードの肖像（336×336）。
	hrPortraitRect = image.Rect(376, 112, 712, 448)
)

const (
	// hrCardRows は左の欄に一度に出すカードの枚数。超える分は ▲▼ で送る。
	hrCardRows   = 5
	hrCardTop    = 145
	hrCardHeight = 104
	hrCardStep   = hrCardHeight + 12
	// hrStockRows は在庫の欄に一度に出す行の数。超える分は ▲▼ で送る。
	hrStockRows = 3
	hrStockTop  = 342
	hrStockStep = 30
	// hrEnemyRows は章の情報に出す敵の行の最大。
	hrEnemyRows = 6
)

// 色（DES-NG05 2 章。方針書の暖色と水の色）。
var (
	hrBg      = color.NRGBA{20, 33, 40, 255}
	hrPanel   = color.NRGBA{33, 50, 56, 255}
	hrCardFil = color.NRGBA{40, 60, 66, 255}
	hrCardSel = color.NRGBA{36, 74, 78, 255}
	hrCardDim = color.NRGBA{28, 42, 47, 255}
	hrBorder  = color.NRGBA{150, 108, 48, 255}
	hrAmber   = color.NRGBA{214, 160, 70, 255}
	hrAqua    = color.NRGBA{88, 196, 184, 255}
	hrText    = color.NRGBA{240, 230, 208, 255}
	hrDim     = color.NRGBA{140, 140, 128, 255}
	hrBad     = color.NRGBA{255, 110, 100, 255}
	hrHPBar   = color.NRGBA{100, 200, 110, 255}
	hrStatBar = color.NRGBA{90, 160, 230, 255}
	hrDark    = color.NRGBA{30, 24, 16, 255}
)

// hrBoxKind は枠・面の種類。
type hrBoxKind int

const (
	hrBoxPanel hrBoxKind = iota
	hrBoxCard
	hrBoxCardSel
	hrBoxCardDim
	hrBoxSlot
	hrBoxSlotSel
	hrBoxRow
	hrBoxRowSel
	hrBoxChip
	hrBoxPill
	hrBoxBand
	hrBoxLine
	hrBoxButton
	hrBoxButtonOff
	hrBoxFrame
	// hrBoxPip は名簿のカードの、手持ちの使用数の箱の使っている枠（琥珀で塗る）。空きは hrBoxSlot（暗いまま）。
	hrBoxPip
)

type hrBox struct {
	Rect image.Rectangle
	Kind hrBoxKind
}

// hrTextItem は 1 行の文。Scale は 1（本文）か 2（見出し・名前）。
type hrTextItem struct {
	Text  string
	X, Y  int
	Scale int
	Color color.NRGBA
}

type hrBar struct {
	Frame image.Rectangle
	Fill  int
	Color color.NRGBA
}

type hrMark struct {
	Rect image.Rectangle
	On   bool
}

type hrDot struct {
	X, Y, R int
	Color   color.NRGBA
}

// hrGlyphKind は、フォントの字形に頼らず vector で描く小さな図形の種類（F-05）。
type hrGlyphKind int

const (
	hrGlyphUp hrGlyphKind = iota
	hrGlyphDown
	hrGlyphArrow
)

// hrGlyph は vector で描く三角（▲▼）と矢印（→）。Rect の中に収めて描く。
type hrGlyph struct {
	Kind  hrGlyphKind
	Rect  image.Rectangle
	Color color.NRGBA
}

type hrTile struct {
	Tile assets.Tile
	Rect image.Rectangle
}

type hrPortrait struct {
	Rect image.Rectangle
	Key  ArtKey
	Tile assets.Tile
	Dim  bool
}

// PrepareCard は左の欄のカード 1 枚。Rect はカード全体、Mark は出撃の切り替え（チェック）の入力範囲。
type PrepareCard struct {
	Member           int
	Rect, Mark       image.Rectangle
	Selected, Deploy bool
}

// hrHandSlot は手持ちの 1 枠。Weapon は武器の列の添字（道具・空きは -1）。
type hrHandSlot struct {
	Rect     image.Rectangle
	Name     string
	Weapon   int
	Item     string
	Equipped bool
}

// hrStockRow は在庫の 1 行。
type hrStockRow struct {
	Rect     image.Rectangle
	Item     prepareItem
	Selected bool
}

// PrepareHiResView は高解像度の準備の画面の値。描画と入力の判定が同じ値を使う（DES-NG05 5 章 1）。
type PrepareHiResView struct {
	Boxes []hrBox
	// Overlays は肖像の上に重ねる面（名前の帯）。
	Overlays  []hrBox
	Texts     []hrTextItem
	Bars      []hrBar
	Marks     []hrMark
	Dots      []hrDot
	Glyphs    []hrGlyph
	Tiles     []hrTile
	Portraits []hrPortrait

	Cards []PrepareCard
	Hand  [prepareHandSlots]hrHandSlot
	Stock []hrStockRow

	RosterUp, RosterDown SceneButton
	StockUp, StockDown   SceneButton
	Deploy               SceneButton
}

// EnemyGroup は章の情報の敵の 1 行（同じ兵種・同じ武器の種類の数）。
type EnemyGroup struct {
	Name   string
	Weapon tactics.WeaponType
	Count  int
	Boss   bool
}

// ChapterInfo は準備の「章の情報」（DES-NG05 4 章）。
type ChapterInfo struct {
	Goal      string
	Total     int
	Groups    []EnemyGroup // 兵種・武器の種類ごと。ボスは含めない。
	Bosses    []EnemyGroup
	Reinforce bool
}

// chapterInfo は章のマップの敵を数える（DES-NG05 4 章）。敵の武器は、持ち物の列の先頭を使う。
func chapterInfo(cat *defs.Catalog, chapter int) ChapterInfo {
	ch := cat.Chapters()[chapter]
	info := ChapterInfo{Goal: ch.Goal}
	m, ok := cat.Map(ch.Map)
	if !ok {
		return info
	}
	info.Total = len(m.Enemies)
	info.Reinforce = len(m.Reinforcements) > 0
	for _, e := range m.Enemies {
		cls, _ := cat.Class(e.Class)
		wt := tactics.WeaponType(-1)
		if len(e.Weapons) > 0 {
			if w, ok := cat.Weapon(e.Weapons[0]); ok {
				wt = prepareWeaponTypes[w.Type]
			}
		}
		if e.Boss {
			name := e.Name
			if name == "" {
				name = cls.Name
			}
			info.Bosses = append(info.Bosses, EnemyGroup{Name: name, Weapon: wt, Count: 1, Boss: true})
			continue
		}
		found := false
		for i := range info.Groups {
			if info.Groups[i].Name == cls.Name && info.Groups[i].Weapon == wt {
				info.Groups[i].Count++
				found = true
				break
			}
		}
		if !found {
			info.Groups = append(info.Groups, EnemyGroup{Name: cls.Name, Weapon: wt, Count: 1})
		}
	}
	return info
}

// weaponCycle は武器の相性の早見（有利な向き）を、tactics の規則（WeaponBeats）から作る（DES-BTL-05）。
// 槍から始め、有利な相手をたどって一周する。例: 槍 → 剣 → 斧 → 槍。
func weaponCycle() []tactics.WeaponType {
	kinds := []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe}
	cur := tactics.Lance
	out := []tactics.WeaponType{cur}
	for range kinds {
		for _, k := range kinds {
			if tactics.WeaponBeats(cur, k) {
				cur = k
				break
			}
		}
		out = append(out, cur)
	}
	return out
}

// weaponMatchupText は装備の武器の種類の相性の一言を返す。例:「槍は剣に強く、斧に弱い」。規則は tactics.WeaponBeats から引く。
func weaponMatchupText(t tactics.WeaponType) string {
	var strong, weak []string
	for _, k := range []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe} {
		if tactics.WeaponBeats(t, k) {
			strong = append(strong, weaponTypeName(k))
		}
		if tactics.WeaponBeats(k, t) {
			weak = append(weak, weaponTypeName(k))
		}
	}
	if len(strong) == 0 && len(weak) == 0 {
		return weaponTypeName(t) + "に武器の相性はない"
	}
	return weaponTypeName(t) + "は" + strings.Join(strong, "・") + "に強く、" + strings.Join(weak, "・") + "に弱い"
}

// hrBarLen は能力の棒の塗る長さ（px）。value を max で止めた割合で、幅 width の枠に塗る。
func hrBarLen(value, max, width int) int {
	return min(max0(value), max) * width / max
}

func max0(v int) int { return max(v, 0) }

// hrRight は文 s を右端 right に揃えたときの x。
func hrRight(s string, scale, right int) int { return right - textWidth(s)*scale }

// prepareHiResView は準備の画面の値を作る（DES-NG-11）。
func (s *SceneController) prepareHiResView() PrepareHiResView {
	var v PrepareHiResView
	cat := s.cfg.Catalog
	ch := cat.Chapters()[s.campaign.Chapter]
	info := chapterInfo(cat, s.campaign.Chapter)
	box := func(r image.Rectangle, k hrBoxKind) { v.Boxes = append(v.Boxes, hrBox{r, k}) }
	txt := func(str string, x, y, scale int, c color.NRGBA) {
		v.Texts = append(v.Texts, hrTextItem{Text: str, X: x, Y: y, Scale: scale, Color: c})
	}
	// ヘッダー: 章の名前（大）、目的、出撃の人数。
	box(image.Rect(0, hrHeader.Max.Y-1, hrHeader.Dx(), hrHeader.Max.Y), hrBoxLine)
	txt(fitText(fmt.Sprintf("第 %d 章　%s", s.campaign.Chapter+1, ch.Name), 760/2), 32, 14, 2, hrAmber)
	if info.Goal != "" {
		txt(fitText("目的　"+info.Goal, 780), 34, 52, 1, hrText)
	}
	box(hrCountRect, hrBoxPill)
	count := fmt.Sprintf("出撃 %d / %d", len(s.SelectedMembers()), s.prepareSlots())
	txt(count, hrCountRect.Min.X+(hrCountRect.Dx()-textWidth(count)*2)/2, hrCountRect.Min.Y+(hrCountRect.Dy()-2*lineHeight)/2, 2, hrAmber)

	s.hrRoster(&v, box, txt)
	s.hrCenter(&v, box, txt)
	// 手持ちの欄の性能の表示が、在庫の行の位置を使うので、在庫を先に作る。
	s.hrStock(&v, box, txt)
	s.hrHand(&v, box, txt)
	s.hrChapter(&v, info, box, txt)

	// フッター: 操作の案内、理由、「配置へ」。
	box(image.Rect(0, hrFooter.Min.Y, hrFooter.Dx(), hrFooter.Min.Y+1), hrBoxLine)
	txt("↑↓ キャラを選ぶ　クリック 持ち物を渡す・戻す　右クリック 武器を装備にする", 32, 766, 1, hrText)
	if s.message != "" {
		for i, line := range wrapSceneTextWidth(s.message, hrDeployRect.Min.X-32-24) {
			if i >= 2 {
				break
			}
			txt(line, 32, 788+i*18, 1, hrBad)
		}
	} else {
		txt("Esc 選んだ在庫の品を外す　Enter 配置へ", 32, 790, 1, hrDim)
	}
	v.Deploy = SceneButton{Label: "配置へ ▶", Rect: hrDeployRect, Enabled: true}
	box(hrDeployRect, hrBoxButton)
	label := "配置へ ▶"
	txt(label, hrDeployRect.Min.X+(hrDeployRect.Dx()-textWidth(label)*2)/2, hrDeployRect.Min.Y+(hrDeployRect.Dy()-2*lineHeight)/2, 2, hrDark)
	return v
}

type (
	hrBoxFn func(image.Rectangle, hrBoxKind)
	hrTxtFn func(string, int, int, int, color.NRGBA)
)

// hrPanelTitle はパネルの枠・見出し・見出しの下の線を作る。
func hrPanelTitle(r image.Rectangle, title string, box hrBoxFn, txt hrTxtFn) {
	box(r, hrBoxPanel)
	txt(fitText(title, r.Dx()-32), r.Min.X+16, r.Min.Y+8, 1, hrAmber)
	box(image.Rect(r.Min.X+12, r.Min.Y+34, r.Max.X-12, r.Min.Y+35), hrBoxLine)
}

// hrRoster は左の欄（出撃メンバーのカード）。
func (s *SceneController) hrRoster(v *PrepareHiResView, box hrBoxFn, txt hrTxtFn) {
	hrPanelTitle(hrRosterPanel, "出撃メンバー", box, txt)
	living := s.livingMembers()
	top := min(max(s.prepareRosterTop, 0), max(len(living)-hrCardRows, 0))
	s.prepareRosterTop = top
	if len(living) > hrCardRows {
		v.RosterUp = SceneButton{"▲", image.Rect(268, 104, 304, 130), top > 0}
		v.RosterDown = SceneButton{"▼", image.Rect(308, 104, 344, 130), top < len(living)-hrCardRows}
		hrArrow(v, box, v.RosterUp)
		hrArrow(v, box, v.RosterDown)
	}
	txt("↑↓ / クリック: キャラを選ぶ", 36, 722, 1, hrDim)
	end := min(len(living), top+hrCardRows)
	for k, i := range living[top:end] {
		m := s.campaign.Roster[i]
		u := s.prepareUnit(i)
		y := hrCardTop + k*hrCardStep
		deployed := s.deployment[i]
		sel := i == s.prepareMember
		r := image.Rect(32, y, 336, y+hrCardHeight)
		kind := hrBoxCard
		switch {
		case sel:
			r.Min.X = 42
			kind = hrBoxCardSel
		case !deployed:
			kind = hrBoxCardDim
		}
		box(r, kind)
		mark := image.Rect(r.Min.X, y, r.Min.X+36, y+hrCardHeight)
		v.Cards = append(v.Cards, PrepareCard{Member: i, Rect: r, Mark: mark, Selected: sel, Deploy: deployed})
		v.Marks = append(v.Marks, hrMark{Rect: image.Rect(r.Min.X+8, y+40, r.Min.X+30, y+62), On: deployed})
		v.Portraits = append(v.Portraits, hrPortrait{Rect: image.Rect(r.Min.X+38, y+10, r.Min.X+122, y+94), Key: s.sceneArtKey(artKeyOf(u)), Tile: unitTile(u), Dim: !deployed})
		tx, right := r.Min.X+132, r.Max.X-10
		fg := hrText
		if !deployed {
			fg = hrDim
		}
		txt(fitText(u.Name, (right-tx)/2), tx, y+8, 2, hrAmber)
		txt(fitText(fmt.Sprintf("%s　Lv %d", u.ClassName, m.Level), right-tx), tx, y+38, 1, fg)
		v.Bars = append(v.Bars, hrBar{Frame: image.Rect(tx, y+60, tx+84, y+68), Fill: hrBarLen(m.Stats.HP, 30, 84), Color: hrHPBar})
		txt(fmt.Sprintf("HP %d", m.Stats.HP), tx+90, y+56, 1, fg)
		used := len(m.Weapons) + itemCount(m.Items)
		for n := range prepareHandSlots {
			kind := hrBoxSlot
			if n < used {
				kind = hrBoxPip
			}
			box(image.Rect(tx+n*15, y+82, tx+n*15+12, y+94), kind)
		}
		txt(fmt.Sprintf("手持ち %d/%d", used, prepareHandSlots), tx+80, y+78, 1, fg)
	}
}

// hrArrow は送りの小さなボタンを描く（箱と、vector の三角）。ボタンの文（▲▼）は入力の判定の名前で、描かない。
func hrArrow(v *PrepareHiResView, box hrBoxFn, b SceneButton) {
	kind, c := hrBoxButton, hrDark
	if !b.Enabled {
		kind, c = hrBoxButtonOff, hrDim
	}
	box(b.Rect, kind)
	gk := hrGlyphUp
	if b.Label == "▼" {
		gk = hrGlyphDown
	}
	const w, h = 14, 10
	cx, cy := b.Rect.Min.X+b.Rect.Dx()/2, b.Rect.Min.Y+b.Rect.Dy()/2
	v.Glyphs = append(v.Glyphs, hrGlyph{Kind: gk, Rect: image.Rect(cx-w/2, cy-h/2, cx+w/2, cy+h/2), Color: c})
}

// hrCenter は中央の大きなカード（肖像・名前・能力・スキル・装備）。
func (s *SceneController) hrCenter(v *PrepareHiResView, box hrBoxFn, txt hrTxtFn) {
	box(hrCenterPanel, hrBoxPanel)
	u := s.prepareUnit(s.prepareMember)
	m := s.campaign.Roster[s.prepareMember]
	v.Portraits = append(v.Portraits, hrPortrait{Rect: hrPortraitRect, Key: s.sceneArtKey(artKeyOf(u)), Tile: unitTile(u)})
	box(hrPortraitRect.Inset(-2), hrBoxFrame)
	band := image.Rect(hrPortraitRect.Min.X, hrPortraitRect.Max.Y-48, hrPortraitRect.Max.X, hrPortraitRect.Max.Y)
	v.Overlays = append(v.Overlays, hrBox{band, hrBoxBand})
	name := fitText(u.Name, 150/2)
	txt(name, band.Min.X+16, band.Min.Y+12, 2, hrAmber)
	exp := fmt.Sprint(m.Exp)
	if m.Level >= 20 {
		exp = "―"
	}
	lv := fmt.Sprintf("%s　Lv %d　経験 %s", u.ClassName, m.Level, exp)
	avail := band.Dx() - 16 - 16 - textWidth(name)*2 - 12
	lv = fitText(lv, avail)
	txt(lv, band.Max.X-12-textWidth(lv), band.Min.Y+16, 1, hrText)

	st := u.Stats
	grid := []struct {
		label, value string
		fill         int
		color        color.NRGBA
	}{
		{"HP", fmt.Sprintf("%d/%d", u.HP, st.HP), hrBarLen(st.HP, 30, 100), hrHPBar},
		{"力", fmt.Sprint(st.Str), hrBarLen(st.Str, 20, 100), hrStatBar},
		{"魔力", fmt.Sprint(st.Mag), hrBarLen(st.Mag, 20, 100), hrStatBar},
		{"技", fmt.Sprint(st.Skl), hrBarLen(st.Skl, 20, 100), hrStatBar},
		{"速さ", fmt.Sprint(st.Spd), hrBarLen(st.Spd, 20, 100), hrStatBar},
		{"幸運", fmt.Sprint(st.Lck), hrBarLen(st.Lck, 20, 100), hrStatBar},
		{"守備", fmt.Sprint(st.Def), hrBarLen(st.Def, 20, 100), hrStatBar},
		{"魔防", fmt.Sprint(st.Res), hrBarLen(st.Res, 20, 100), hrStatBar},
		{"移動", fmt.Sprint(st.Mov), hrBarLen(st.Mov, 10, 100), hrStatBar},
	}
	for i, g := range grid {
		x, y := 380+(i%3)*112, 462+(i/3)*56
		txt(g.label, x, y, 1, hrText)
		txt(g.value, hrRight(g.value, 1, x+100), y, 1, hrText)
		v.Bars = append(v.Bars, hrBar{Frame: image.Rect(x, y+22, x+100, y+30), Fill: g.fill, Color: g.color})
	}
	// スキル: 名前の行と、説明（1 つ 1 行、3 行まで）。
	skills := unitSkills(u)
	txt("スキル", 380, 618, 1, hrDim)
	if len(skills) == 0 {
		txt("（なし）", 440, 618, 1, hrText)
	} else {
		names := make([]string, len(skills))
		for i, sk := range skills {
			names[i] = sk.Name
		}
		txt(fitText(strings.Join(names, "、"), 704-440), 440, 618, 1, skillColorNRGBA)
		for i, sk := range skills {
			if i >= 3 {
				break
			}
			txt(fitText(skillDescription(sk), 324), 380, 636+i*16, 1, hrText)
		}
	}
	if len(u.Weapons) > 0 {
		w := u.Weapons[0]
		txt(fitText(fmt.Sprintf("武器　%s（%s）", w.Name, weaponTypeName(w.Type)), 324), 380, 690, 1, hrText)
		txt(fitText(weaponMatchupText(w.Type), 324), 380, 710, 1, hrAqua)
	} else {
		txt("武器　なし", 380, 690, 1, hrText)
	}
}

// skillColorNRGBA はスキルの名前の色（情報欄と同じ黄）。
var skillColorNRGBA = color.NRGBA{255, 240, 60, 255}

// hrHand は右上の手持ち（5 枠）。
func (s *SceneController) hrHand(v *PrepareHiResView, box hrBoxFn, txt hrTxtFn) {
	m := s.campaign.Roster[s.prepareMember]
	u := s.prepareUnit(s.prepareMember)
	used := len(m.Weapons) + itemCount(m.Items)
	hrPanelTitle(hrHandPanel, fmt.Sprintf("%sの手持ち　%d / %d", u.Name, used, prepareHandSlots), box, txt)
	emptyFrom := -1
	for k, sl := range s.prepareHand() {
		r := image.Rect(752+k*62, 150, 752+k*62+54, 204)
		hs := hrHandSlot{Rect: r, Name: sl.name, Weapon: sl.weapon, Item: sl.item, Equipped: sl.equipped}
		v.Hand[k] = hs
		if sl.name == "" {
			box(r, hrBoxSlot)
			if emptyFrom < 0 {
				emptyFrom = k
			}
			continue
		}
		box(r, hrBoxSlotSel)
		icon := image.Rect(r.Min.X+11, r.Min.Y+11, r.Min.X+43, r.Min.Y+43)
		if sl.weapon >= 0 {
			id := m.Weapons[sl.weapon]
			w := s.prepareWeapon(id)
			if t, ok := weaponTile(w.Type); ok {
				v.Tiles = append(v.Tiles, hrTile{Tile: t, Rect: icon})
			}
		} else {
			v.Dots = append(v.Dots, s.hrItemDot(sl.item, icon.Min.X+16, icon.Min.Y+16, 13))
		}
		txt(fitText(sl.name, 58), r.Min.X, 210, 1, hrText)
		if sl.equipped {
			txt("E", r.Min.X+4, r.Min.Y+2, 1, hrAmber)
		}
	}
	if emptyFrom >= 0 {
		txt(fmt.Sprintf("（空き）×%d", prepareHandSlots-emptyFrom), 752+emptyFrom*62, 210, 1, hrDim)
	}
	// 操作の説明の場所に、選んだ品（なければ指している品）の性能を出す（F-01）。何もなければ操作の説明。
	if it, selected, ok := s.hrDetailItem(v); ok {
		s.hrItemDetail(it, selected, txt)
		return
	}
	txt("在庫の行を選び、空き枠をクリックで渡す", 756, 236, 1, hrAqua)
	txt("枠をクリックで、在庫に戻す", 756, 254, 1, hrText)
	txt("E: 装備（右クリックで武器を装備にする）", 756, 270, 1, hrDim)
}

// hrDetailItem は手持ちの欄の性能の表示に使う品を返す。在庫の行を選んでいればその品（selected が真）、
// そうでなければマウスが指している手持ちの枠・在庫の行の品。何も指していなければ ok が偽。
func (s *SceneController) hrDetailItem(v *PrepareHiResView) (it prepareItem, selected, ok bool) {
	if s.prepareItem.ok {
		return s.prepareItem, true, true
	}
	pt := s.prepareCursor
	m := s.campaign.Roster[s.prepareMember]
	for _, h := range v.Hand {
		if h.Name == "" || !pt.In(h.Rect) {
			continue
		}
		if h.Weapon >= 0 {
			return prepareItem{ok: true, weapon: true, id: m.Weapons[h.Weapon]}, false, true
		}
		return prepareItem{ok: true, id: h.Item}, false, true
	}
	for _, r := range v.Stock {
		if pt.In(r.Rect) {
			return r.Item, false, true
		}
	}
	return prepareItem{}, false, false
}

// 性能の表示の行の位置。手持ちの欄（y < 290）に最大 4 行を、14 px 間隔で収める。
const (
	hrDetailX     = 756
	hrDetailY     = 229
	hrDetailStep  = 14
	hrDetailWidth = 304
)

// hrItemDetail は品の名前（全文）と性能を、手持ちの欄の下部に出す（DES-UI-32 の prepareDetail と同じ文面。F-01）。
// 名前の行、性能（折り返して 2 行まで）、選んだ在庫の武器を装備できないときはその理由（赤）の順。
func (s *SceneController) hrItemDetail(it prepareItem, selected bool, txt hrTxtFn) {
	var name, stats string
	if it.weapon {
		w := s.prepareWeapon(it.id)
		name = w.Name
		// 名前は 1 行目に全文を出すので、性能の文からは省く。
		stats = strings.TrimPrefix(weaponDetail(w), w.Name+"　")
	} else {
		d, _ := s.cfg.Catalog.Item(it.id)
		name, stats = d.Name, itemDetailText(d)
	}
	head, c := name, hrText
	if selected {
		head, c = "選んだ品: "+name, hrAqua
	}
	y := hrDetailY
	txt(fitText(head, hrDetailWidth), hrDetailX, y, 1, c)
	for i, line := range wrapSceneTextWidth(stats, hrDetailWidth) {
		if i >= 2 {
			break
		}
		y += hrDetailStep
		txt(line, hrDetailX, y, 1, hrText)
	}
	if selected && it.weapon {
		if reason := s.cannotEquipReason(it.id); reason != "" {
			txt(fitText(reason, hrDetailWidth), hrDetailX, y+hrDetailStep, 1, hrBad)
		}
	}
}

func (s *SceneController) hrItemName(it prepareItem) string {
	if it.weapon {
		return s.prepareWeapon(it.id).Name
	}
	d, _ := s.cfg.Catalog.Item(it.id)
	return d.Name
}

// hrItemDot は道具の絵の代わりの丸（DES-NG05 Q-P4: 素材は作らない）。r は丸の最大の半径。
// 回復薬は赤。爆弾は定義の爆発の範囲（BlastRadius）で、色と大きさを変える: 範囲 0（1 マス）は橙で小さく、
// 範囲 1 は灰、範囲 2 以上は暗い赤褐色で大きい。爆弾の ID は使わず、定義の値だけから決める。
func (s *SceneController) hrItemDot(id string, x, y, r int) hrDot {
	d, _ := s.cfg.Catalog.Item(id)
	c := color.NRGBA{200, 70, 60, 255}
	if d.Bomb {
		radius := min(max(d.BlastRadius, 0), 2)
		c = [...]color.NRGBA{{230, 140, 50, 255}, {150, 150, 160, 255}, {140, 64, 40, 255}}[radius]
		r = r * (5 + radius) / 7
	}
	return hrDot{X: x, Y: y, R: r, Color: c}
}

// hrStock は右中の在庫（武器 → 杖 → 道具の順の行）。
func (s *SceneController) hrStock(v *PrepareHiResView, box hrBoxFn, txt hrTxtFn) {
	hrPanelTitle(hrStockPanel, "在庫", box, txt)
	var lines []prepareLine
	for _, l := range s.prepareStock() {
		if l.heading == "" {
			lines = append(lines, l)
		}
	}
	top := min(max(s.prepareStockTop, 0), max(len(lines)-hrStockRows, 0))
	s.prepareStockTop = top
	if len(lines) > hrStockRows {
		v.StockUp = SceneButton{"▲", image.Rect(1000, 304, 1030, 330), top > 0}
		v.StockDown = SceneButton{"▼", image.Rect(1034, 304, 1064, 330), top < len(lines)-hrStockRows}
		hrArrow(v, box, v.StockUp)
		hrArrow(v, box, v.StockDown)
	}
	if len(lines) == 0 {
		txt("（なし）", 756, hrStockTop+4, 1, hrDim)
	}
	for k, l := range lines[top:min(len(lines), top+hrStockRows)] {
		y := hrStockTop + k*hrStockStep
		r := image.Rect(752, y, 1056, y+26)
		sel := s.prepareItem.ok && s.prepareItem == l.item
		kind := hrBoxRow
		if sel {
			kind = hrBoxRowSel
		}
		box(r, kind)
		v.Stock = append(v.Stock, hrStockRow{Rect: r, Item: l.item, Selected: sel})
		if l.item.weapon {
			if t, ok := weaponTile(s.prepareWeapon(l.item.id).Type); ok {
				v.Tiles = append(v.Tiles, hrTile{Tile: t, Rect: image.Rect(r.Min.X+6, y+5, r.Min.X+22, y+21)})
			}
		} else {
			v.Dots = append(v.Dots, s.hrItemDot(l.item.id, r.Min.X+14, y+13, 7))
		}
		n := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l.suffix), "×"))
		if n == "" {
			n = "1"
		}
		cnt := "× " + n
		txt(fitText(l.name, 200), r.Min.X+32, y+5, 1, hrText)
		txt(cnt, hrRight(cnt, 1, r.Max.X-10), y+5, 1, hrAmber)
	}
}

// hrChapter は右下の章の情報（敵の数、兵種ごとの数と武器、ボス、武器の相性の早見）。
func (s *SceneController) hrChapter(v *PrepareHiResView, info ChapterInfo, box hrBoxFn, txt hrTxtFn) {
	hrPanelTitle(hrChapterPanel, "章の情報", box, txt)
	txt(fmt.Sprintf("敵　%d 体", info.Total), 756, 484, 1, hrText)
	if info.Reinforce {
		txt("増援あり", hrRight("増援あり", 1, 1052), 484, 1, hrBad)
	}
	rows := append(append([]EnemyGroup{}, info.Groups...), info.Bosses...)
	if len(rows) > hrEnemyRows {
		more := len(rows) - (hrEnemyRows - 1)
		rows = append(rows[:hrEnemyRows-1], EnemyGroup{Name: fmt.Sprintf("ほか %d 種", more)})
	}
	for k, g := range rows {
		y := 506 + k*26
		chip := image.Rect(756, y, 788, y+22)
		if g.Count > 0 {
			box(chip, hrBoxChip)
			c := "―"
			if g.Weapon >= 0 {
				c = weaponTypeName(g.Weapon)
			}
			txt(c, chip.Min.X+(chip.Dx()-textWidth(c))/2, y+3, 1, hrAmber)
		}
		name := g.Name
		cnt := fmt.Sprintf("× %d", g.Count)
		if g.Boss {
			name += "（ボス）"
		}
		if g.Count == 0 {
			cnt = ""
		}
		txt(fitText(name, 190), 800, y+3, 1, hrText)
		if cnt != "" {
			txt(cnt, hrRight(cnt, 1, 1052), y+3, 1, hrText)
		}
	}
	txt("武器の相性（矢印の向きが有利）", 756, 668, 1, hrDim)
	cycle := weaponCycle()
	for i, t := range cycle {
		r := image.Rect(756+i*72, 692, 756+i*72+44, 718)
		box(r, hrBoxChip)
		n := weaponTypeName(t)
		txt(n, r.Min.X+(r.Dx()-textWidth(n))/2, r.Min.Y+5, 1, hrAmber)
		if i < len(cycle)-1 {
			v.Glyphs = append(v.Glyphs, hrGlyph{Kind: hrGlyphArrow, Rect: image.Rect(r.Max.X+6, r.Min.Y+4, r.Max.X+22, r.Min.Y+22), Color: hrAqua})
		}
	}
}

// --- 入力 ---

// clickPrepareHiRes は高解像度の準備のクリックを、送りのボタン → 配置へ → カード（チェック・本体）→ 手持ち → 在庫の順に受ける（DES-NG-11）。
func (s *SceneController) clickPrepareHiRes(x, y int) {
	pt := image.Pt(x, y)
	v := s.prepareHiResView()
	if v.RosterUp.Label != "" {
		if pt.In(v.RosterUp.Rect) {
			if v.RosterUp.Enabled {
				s.prepareRosterTop--
			}
			return
		}
		if pt.In(v.RosterDown.Rect) {
			if v.RosterDown.Enabled {
				s.prepareRosterTop++
			}
			return
		}
	}
	if v.StockUp.Label != "" {
		if pt.In(v.StockUp.Rect) {
			if v.StockUp.Enabled {
				s.prepareStockTop--
			}
			return
		}
		if pt.In(v.StockDown.Rect) {
			if v.StockDown.Enabled {
				s.prepareStockTop++
			}
			return
		}
	}
	if pt.In(v.Deploy.Rect) {
		s.enterDeploy()
		return
	}
	for _, c := range v.Cards {
		if pt.In(c.Mark) {
			s.toggleDeployment(c.Member)
			return
		}
		if pt.In(c.Rect) {
			s.prepareMember = c.Member
			s.message = ""
			return
		}
	}
	for n, h := range v.Hand {
		if pt.In(h.Rect) {
			s.clickHandSlot(n, h)
			return
		}
	}
	for _, r := range v.Stock {
		if pt.In(r.Rect) {
			s.selectStock(r.Item)
			return
		}
	}
}

// selectStock は在庫の行を選ぶ（もう一度クリックで外す）。手持ちが満杯・装備できない武器のときは、理由を案内に出す。
func (s *SceneController) selectStock(it prepareItem) {
	s.message = ""
	if s.prepareItem == it {
		s.prepareItem = prepareItem{}
		return
	}
	s.prepareItem = it
	m := s.campaign.Roster[s.prepareMember]
	if len(m.Weapons)+itemCount(m.Items) >= prepareHandSlots {
		s.message = "手持ちは 5 枠までです。枠をクリックして在庫に戻すと、空きができます"
		return
	}
	if it.weapon {
		s.message = s.cannotEquipReason(it.id)
	}
}

// cannotEquipReason は選んだ仲間が武器を装備できないときの理由（できるなら空）。
func (s *SceneController) cannotEquipReason(id string) string {
	m := s.campaign.Roster[s.prepareMember]
	chr, _ := s.cfg.Catalog.Character(m.Character)
	cls, _ := s.cfg.Catalog.Class(chr.Class)
	if d, _ := s.cfg.Catalog.Weapon(id); !cls.CanUseWeapon(d.Type) {
		return fmt.Sprintf("%s（%s）は%sを装備できない", chr.Name, cls.Name, weaponTypeName(prepareWeaponTypes[d.Type]))
	}
	return ""
}

// clickHandSlot は手持ちの枠のクリック。空き枠なら選んだ在庫の品を渡し、品のある枠なら在庫へ戻す。
func (s *SceneController) clickHandSlot(n int, h hrHandSlot) {
	c, m := s.campaign, s.prepareMember
	s.message = ""
	var err error
	switch {
	case h.Name == "":
		if !s.prepareItem.ok {
			s.message = "先に在庫の行を選んでください"
			return
		}
		for _, l := range s.prepareStock() {
			if l.heading == "" && l.item == s.prepareItem {
				err = l.act()
				break
			}
		}
		if err == nil {
			// 在庫がなくなったら選択を外す。
			still := false
			for _, l := range s.prepareStock() {
				if l.heading == "" && l.item == s.prepareItem {
					still = true
				}
			}
			if !still {
				s.prepareItem = prepareItem{}
			}
		}
	case h.Weapon >= 0:
		err = c.StoreWeapon(m, h.Weapon)
	default:
		err = c.StoreItem(m, h.Item)
	}
	if errors.Is(err, core.ErrHandFull) {
		s.message = "手持ちは 5 枠までです"
	} else if err != nil {
		s.message = err.Error()
	}
}

// rightClickPrepare は右クリックが手持ちの武器の枠の上なら、その武器を装備（先頭）にして真を返す。それ以外は偽で、ふつうの取消しにする。
func (s *SceneController) rightClickPrepare(x, y int) bool {
	if s.scene != Prepare || !s.cfg.HiRes {
		return false
	}
	pt := image.Pt(x, y)
	for _, h := range s.prepareHiResView().Hand {
		if pt.In(h.Rect) && h.Weapon >= 0 {
			s.message = ""
			if h.Weapon == 0 {
				s.message = "すでに装備している武器です"
			} else if err := s.campaign.MoveWeaponFirst(s.prepareMember, h.Weapon); err != nil {
				s.message = err.Error()
			}
			return true
		}
	}
	return false
}

// movePrepareMember は ↑↓ で選ぶキャラを生存者の中で動かし、カードが見える位置まで名簿を送る。
func (s *SceneController) movePrepareMember(delta int) {
	living := s.livingMembers()
	at := 0
	for k, i := range living {
		if i == s.prepareMember {
			at = k
		}
	}
	at = min(max(at+delta, 0), len(living)-1)
	s.prepareMember = living[at]
	s.message = ""
	if at < s.prepareRosterTop {
		s.prepareRosterTop = at
	} else if at >= s.prepareRosterTop+hrCardRows {
		s.prepareRosterTop = at - hrCardRows + 1
	}
}

// cancelPrepareHiRes は Esc・右クリックで、選んだ在庫の品を外す。
func (s *SceneController) cancelPrepareHiRes() {
	s.prepareItem = prepareItem{}
	s.message = ""
}

// fullWindowPrepare は、今の場面が窓の全体に描く高解像度の準備か（中央寄せの層を使わない。DES-NG-11）。
func (s *SceneController) fullWindowPrepare() bool { return s.scene == Prepare && s.cfg.HiRes }

// hrSceneView は View() が高解像度の準備で返す値。入力の判定に使うカードと「配置へ」だけを持つ。
func (s *SceneController) hrSceneView() SceneView {
	hv := s.prepareHiResView()
	v := SceneView{Buttons: []SceneButton{hv.Deploy}}
	for _, c := range hv.Cards {
		v.Rows = append(v.Rows, PrepareRow{Member: c.Member, Rect: c.Rect, Mark: c.Mark, Selected: c.Selected})
	}
	return v
}

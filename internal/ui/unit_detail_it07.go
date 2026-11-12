package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 詳しい画面 UnitDetail（DES-UI-31、E-120）。盤面と右欄 (0,0)〜(832,512) に重ねる UI だけの表示で、入力の状態・仮選択・
// 仮配置は変えない。下 (0,512)〜(832,576) は操作の案内。

// DetailSection は詳しい画面の節（Tab で送る。DES-UI-31）。
type DetailSection int

const (
	// SectionStats は能力 9 種と武器の短い行（E-97 の目的を保つ）。
	SectionStats DetailSection = iota
	// SectionWeapons は手持ち 5 枠と全武器の詳しい値。
	SectionWeapons
	// SectionSkills はキャラと兵種のスキル。
	SectionSkills
	detailSections
)

// detailGuide は詳しい画面の下の操作の案内。
const detailGuide = "Tab: 節を送る　I・Esc・右クリック: 戻る"

// deployDetailButtonRect は配置の既存ボタンの下に置く詳細の入口（DES-UI-31）。
var deployDetailButtonRect = image.Rect(752, 336, 824, 356)

// 詳しい画面の配置（DES-IT07 3.4 節）。左の列 x = 16〜304 に能力と武器の短い行・スキル、右の列 x = 320〜816 に手持ち 5 枠と全武器。
const (
	detailLeftX      = 16
	detailLeftWidth  = 288
	detailRightX     = 320
	detailRightWidth = 496
	detailBottom     = MapSize - infoMargin
	detailStatGap    = 18
	detailWeaponGap  = 18
	detailHandSlots  = 5
)

// OpenDetail は戦闘で I を押したときに詳しい画面を開く（DES-UI-31、DES-IT07 3.1 節の状態の表）。カーソルの下の生きているユニット、
// いなければ選択中のユニットを出す。両方なければ、または再生中・Over・巻き戻しの一覧の間は開かない。開けたら真。
func (c *Controller) OpenDetail() bool {
	if c.rewind.open || c.state == Playing || c.state == Over {
		return false
	}
	i := c.detailTarget()
	if i < 0 {
		return false
	}
	c.detailOpen, c.detailUnit, c.detailSection = true, i, SectionStats
	return true
}

// detailTarget は詳しい画面に出すユニットの添字を返す（右欄の上段と同じ人物。DES-UI-31）。カーソルの下の生きているユニット、
// カーソルが右欄の上なら直前に盤面で見ていたマスのユニット、どちらもいなければ選択中のユニット。いなければ −1。
func (c *Controller) detailTarget() int {
	units := c.Units()
	cell, ok := c.Camera().CellAt(c.cursorX, c.cursorY)
	if !ok && image.Pt(c.cursorX, c.cursorY).In(image.Rect(MapSize, 0, ScreenWidth, InfoHeight)) {
		cell, ok = c.lastCell, c.lastCellOK
	}
	if ok {
		if i := liveUnitIndex(units, cell); i >= 0 {
			return i
		}
	}
	if _, ok := c.selectedCell(); ok {
		return c.unit
	}
	return -1
}

// DetailButton は操作行の「詳細 I」を返す（DES-UI-31）。開けるユニットがいて、再生中・Over でなければ押せる。
func (c *Controller) DetailButton() SceneButton {
	enabled := c.state != Playing && c.state != Over && c.detailTarget() >= 0
	return SceneButton{Label: "詳細 I", Rect: detailButtonRect, Enabled: enabled}
}

// DetailOpen は詳しい画面を出しているかを返す（DES-UI-31）。
func (c *Controller) DetailOpen() bool { return c.detailOpen }

// DetailInput は詳しい画面の Tab（節を送る）と、I・Esc・右クリック（閉じる）を受ける（DES-UI-31）。閉じると元の入力状態に戻る。
// 詳しい画面を出していなければ何もせず偽を返す。
func (c *Controller) DetailInput(tab, close bool) bool {
	if !c.detailOpen {
		return false
	}
	if close {
		c.detailOpen = false
	} else if tab {
		c.detailSection = (c.detailSection + 1) % detailSections
	}
	return true
}

// OpenDeployDetail は配置で I を押したときに詳しい画面を開く（DES-UI-31）。カーソルの下の生きているユニット、
// いなければ配置で選択中のユニット。仮配置は変えない。開けたら真。
func (s *SceneController) OpenDeployDetail() bool {
	if s.scene != Deploy {
		return false
	}
	d := s.deployView().Deploy
	i := s.deployInfoUnit(d.Units, d.Selected)
	if i < 0 {
		return false
	}
	s.detailOpen, s.detailUnit, s.detailSection = true, i, SectionStats
	return true
}

// DeployDetailInput は配置の詳しい画面の Tab と閉じる操作を受ける（DES-UI-31）。閉じると仮配置へ戻る。
func (s *SceneController) DeployDetailInput(tab, close bool) bool {
	if !s.detailOpen {
		return false
	}
	if close {
		s.detailOpen = false
	} else if tab {
		s.detailSection = (s.detailSection + 1) % detailSections
	}
	return true
}

// DetailView は詳しい画面に描くものを返す（DES-UI-31）。能力 9 種、手持ち 5 枠、装備と全武器の詳しい値、キャラ・兵種のスキルを
// 一画面に並べ、能力の欄に武器の種類の絵・名前・装備の E 印を並べる（E-97 の目的を保つ）。section の見出しに ▶ を付ける。
// 一番下が (0,512) を越えるときは、越える行を出さずに「…」で止める。
func DetailView(u tactics.Unit, section DetailSection) InfoView {
	var v InfoView
	head := func(x, y int, label string, s DetailSection) {
		st, mark := StyleLabel, "　"
		if s == section {
			st, mark = StyleTitle, "▶"
		}
		v.text(x, y, mark+label, st)
	}
	title := strings.TrimSpace(unitName(u) + "　" + u.ClassName + "　" + levelText(u))
	v.text(detailLeftX, 12, fitRewind(title, ScreenWidth-2*detailLeftX), StyleTitle)

	// 左の列: 能力 9 種と武器の短い行。
	head(detailLeftX, 40, "能力", SectionStats)
	v.Icons = append(v.Icons, InfoIcon{X: detailLeftX + 160, Y: 60, Size: PortraitSize, Tile: unitTile(u)})
	s := u.Stats
	y := 60
	for _, row := range [][2]string{
		{"HP", fmt.Sprintf("%d/%d", u.HP, s.HP)}, {"力", fmt.Sprint(s.Str)}, {"魔力", fmt.Sprint(s.Mag)},
		{"技", fmt.Sprint(s.Skl)}, {"速さ", fmt.Sprint(s.Spd)}, {"幸運", fmt.Sprint(s.Lck)},
		{"守備", fmt.Sprint(s.Def)}, {"魔防", fmt.Sprint(s.Res)}, {"移動", fmt.Sprint(s.Mov)},
	} {
		v.text(detailLeftX+16, y, row[0], StyleLabel)
		v.text(detailLeftX+80, y, row[1], StylePlain)
		y += detailStatGap
	}
	y += 4
	for i, w := range u.Weapons {
		if i == u.Equipped {
			v.text(detailLeftX+16, y, weaponRowMark, StyleTitle)
		}
		if tile, ok := weaponTile(w.Type); ok {
			v.Icons = append(v.Icons, InfoIcon{X: detailLeftX + 28, Y: y, Size: assets.TileSize, Tile: tile})
		}
		v.text(detailLeftX+48, y, fitRewind(w.Name+"（"+weaponTypeName(w.Type)+"）", detailLeftWidth-48), StylePlain)
		y += detailWeaponGap
	}

	// 左の列の下: スキル。名前（黄）と効果の文を折り返して並べる。
	y = max(y+8, 344)
	head(detailLeftX, y, "スキル", SectionSkills)
	y += 20
	skills := unitSkills(u)
	if len(skills) == 0 {
		v.text(detailLeftX+16, y, "なし", StylePlain)
	}
	lines := []InfoText{}
	for _, sk := range skills {
		lines = append(lines, InfoText{X: detailLeftX + 16, Text: sk.Name, Style: StyleSkill})
		for _, l := range wrapWidth(skillDescription(sk), detailLeftWidth-32) {
			lines = append(lines, InfoText{X: detailLeftX + 32, Text: l, Style: StylePlain})
		}
	}
	v.addClipped(lines, y)

	// 右の列: 手持ち 5 枠（武器の列の順に武器、続けて道具を 1 個ずつ、残りは「―」）と全武器の詳しい値。
	head(detailRightX, 40, "手持ち 5 枠・全武器", SectionWeapons)
	y = 60
	for i, label := range handSlotLabels(u) {
		v.text(detailRightX+16, y, fmt.Sprintf("%d　%s", i+1, fitRewind(label, detailRightWidth-48)), StylePlain)
		y += lineHeight
	}
	y += 8
	lines = lines[:0]
	for i, w := range u.Weapons {
		mark := "　"
		if i == u.Equipped {
			mark = weaponRowMark
		}
		lines = append(lines, InfoText{X: detailRightX + 16, Text: mark + " " + w.Name + "（" + weaponTypeName(w.Type) + "）", Style: StyleTitle})
		for _, l := range wrapWidth(strings.TrimPrefix(weaponDetail(w), w.Name+"　"), detailRightWidth-32) {
			lines = append(lines, InfoText{X: detailRightX + 32, Text: l, Style: StylePlain})
		}
	}
	v.addClipped(lines, y)
	v.Bottom = detailBottom
	return v
}

// addClipped は lines を y から 1 行ずつ並べ、detailBottom を越える行は出さず、最後の行を「…」にする（DES-UI-31）。
func (v *InfoView) addClipped(lines []InfoText, y int) {
	for i, l := range lines {
		if y+lineHeight > detailBottom {
			return
		}
		if i < len(lines)-1 && y+2*lineHeight > detailBottom {
			v.text(l.X, y, "…", StyleLabel)
			return
		}
		v.text(l.X, y, l.Text, l.Style)
		y += lineHeight
	}
}

// handSlotLabels は手持ちの 5 枠の名前を返す（DES-ITM-02）。武器（装備は E 印）、道具を 1 個ずつ、残りは「―」。
func handSlotLabels(u tactics.Unit) []string {
	var out []string
	for i, w := range u.Weapons {
		mark := "　"
		if i == u.Equipped {
			mark = weaponRowMark
		}
		out = append(out, mark+" "+w.Name+"（"+weaponTypeName(w.Type)+"）")
	}
	for _, st := range u.Items {
		for range st.Count {
			out = append(out, "　"+st.Item.Name+"（"+itemEffect(st.Item)+"）")
		}
	}
	for len(out) < detailHandSlots {
		out = append(out, "―")
	}
	return out
}

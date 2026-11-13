package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 共通の予測窓と、右欄の上下 2 段・操作行（DES-UI-31・34、DES-UI-07 の改訂。DES-IT07 3.4 節）。

// ForecastWindow は攻撃と回復で共有する予測窓（DES-UI-31）。Rect は画面の座標、View は窓の中に描くもの。
// Fallback は盤面に置けず右欄 320×360 に収め直したこと。その間は右欄の通常情報を描かない。
type ForecastWindow struct {
	Rect     image.Rectangle
	View     InfoView
	Fallback bool
}

// 予測窓の大きさと置き場所の候補（DES-IT07 3.4 節）。盤面 512×512 の中の 416×240 の 4 か所と、右欄の 320×360。
var (
	forecastCandidates = [...]image.Rectangle{
		image.Rect(0, 8, 416, 248), image.Rect(96, 8, 512, 248),
		image.Rect(0, 264, 416, 504), image.Rect(96, 264, 512, 504),
	}
	forecastFallbackRect = image.Rect(MapSize, 0, ScreenWidth, InfoHeight)
)

// forecastLayout は窓の中の相対の配置（DES-IT07 3.4 節）。盤面の窓と右欄の窓で同じ文字の大きさのまま位置だけを変える。
type forecastLayout struct {
	// bannerBottom は撃破・危険の帯の下端。帯は (8,8) から窓の幅 −8 まで。
	bannerBottom int
	// defX は守る側の列（名前・武器・表の値）の x。
	defX int
	// nameY・weaponY・tableY・skillY・controlY は各欄の上端。weapon の欄は 32 px（武器の絵の大きさ）。
	nameY, weaponY, tableY, skillY, controlY int
	// rowGap は 6 行の表の行の間隔。skillLines は効くスキルと特効の欄に使える行数。
	rowGap, skillLines int
}

var (
	// 盤面の 416×240: 帯 (8,8)〜(408,36)、名前 (16,40)・(280,40)、武器 (16,60)〜(400,92)、
	// 表 (16,92)〜(400,188)、スキル (16,188)〜(400,220)、武器送り (16,220)〜(400,236)（E-127）。
	forecastBoardLayout = forecastLayout{bannerBottom: 36, defX: 280, nameY: 40, weaponY: 60, tableY: 92, skillY: 188, controlY: 220, rowGap: lineHeight, skillLines: 2}
	// 右欄の 320×360: 帯 (8,8)〜(312,40)、名前 (16,44)・(224,44)、武器 (16,68)〜(304,100)、表 (16,104)〜(304,240)、
	// スキル (16,248)〜(304,296)、武器送り (16,304)〜(304,352)。
	forecastFallbackLayout = forecastLayout{bannerBottom: 40, defX: 224, nameY: 44, weaponY: 68, tableY: 104, skillY: 248, controlY: 304, rowGap: 22, skillLines: 3}
)

// forecastValueX は表の攻撃する側の値の列（窓の左端から）。
const forecastValueX = 96

// forecastButtonWidth は窓の右端の「決定」「戻る」の 1 つ分の幅（「決定」は右端から 96、「戻る」は 48 px）。
const forecastButtonWidth = 48

// ForecastRect は予測窓の置き場所を返す（DES-UI-31、E-122）。相手 target と仮の移動先 dest の両マスに重ならない盤面の
// 候補を、相手と反対側の上下から選ぶ（相手が盤面の上半分なら下の候補から）。どれも重なるときは右欄に収め直し、fallback が真。
func ForecastRect(camera Camera, target, dest tactics.Pos) (r image.Rectangle, fallback bool) {
	cell := func(p tactics.Pos) image.Rectangle {
		s := camera.Screen(p)
		return image.Rect(s.X, s.Y, s.X+CellSize, s.Y+CellSize)
	}
	tr, dr := cell(target), cell(dest)
	order := [4]int{2, 3, 0, 1}
	if tr.Min.Y >= MapSize/2 {
		order = [4]int{0, 1, 2, 3}
	}
	for _, i := range order {
		if c := forecastCandidates[i]; !c.Overlaps(tr) && !c.Overlaps(dr) {
			return c, false
		}
	}
	return forecastFallbackRect, true
}

// ForecastWindow は攻撃・回復の予測を共通の窓にする（DES-UI-31、DES-UI-07 の改訂）。値は Battle.Forecast・HealForecast
// から作る（attackForecast・healForecast）。予測を出していないとき・巻き戻しの一覧の間は ok が偽。
func (c *Controller) ForecastWindow() (ForecastWindow, bool) {
	if c.rewind.open {
		return ForecastWindow{}, false
	}
	f, attack := c.attackForecast()
	h, heal := c.healForecast()
	if !attack && !heal {
		return ForecastWindow{}, false
	}
	units := c.Units()
	r, fallback := ForecastRect(c.Camera(), units[c.target].Pos, c.dest)
	l := forecastBoardLayout
	if fallback {
		l = forecastFallbackLayout
	}
	w := ForecastWindow{Rect: r, Fallback: fallback}
	v := &w.View
	add := func(x, y int, s string, st InfoStyle) { v.text(r.Min.X+x, r.Min.Y+y, s, st) }
	width := r.Dx()
	if heal {
		// 回復は同じ窓の枠に HealForecast の値を出す。杖の選択順は変えない（DES-UI-18）。
		add(16, l.nameY, "回復の予測", StyleTitle)
		for i, line := range wrapWidth(h.Healer+" → "+h.Target, width-32) {
			add(16, l.nameY+lineHeight+i*lineHeight, line, StylePlain)
		}
		add(16, l.tableY, "HP "+hpText(h.HP, h.After), StylePlain)
		add(16, l.tableY+l.rowGap, fmt.Sprintf("回復 +%d", h.Amount), StyleGood)
		add(16, l.tableY+2*l.rowGap, fmt.Sprintf("杖 残り %d → %d/%d", h.Uses, h.Uses-1, h.MaxUses), StylePlain)
		add(16, l.controlY, "相手をクリック: 決定　Esc: 戻る", StyleLabel)
		v.Bottom = r.Max.Y
		return w, true
	}

	// 帯: 撃破は金の地に黒、危険は赤の地に白（DES-UI-07）。
	if e := forecastEmphasis(f); e != ForecastNormal {
		label, st, fg := "撃破　この攻撃で倒せる", StyleTitle, StyleBlack
		if e == ForecastDanger {
			label, st, fg = "危険　反撃で倒される", StyleBad, StylePlain
		}
		v.Banner = &InfoBanner{Rect: image.Rect(r.Min.X+8, r.Min.Y+8, r.Max.X-8, r.Min.Y+l.bannerBottom), Style: st}
		add(16, 8+(l.bannerBottom-8-lineHeight)/2, label, fg)
	}
	attackerW, defenderW := width-l.defX, l.defX-16 // 守る側・攻撃する側の欄の幅の上限
	aw, dw := units[c.unit].Weapons[c.weapon], units[c.target].EquippedWeapon()
	// 帯がないときは上余白も使える。長い名前や右欄の狭い武器欄には、上部を折り返し用の行にする。
	compactHeader := v.Banner == nil && (len(wrapWidth(forecastName(f.Attacker, units[c.unit].Boss), defenderW-8)) > 1 ||
		len(wrapWidth(forecastName(f.Defender, units[c.target].Boss), attackerW-16)) > 1 ||
		len(wrapWidth("▲ "+aw.Name+"（"+weaponTypeName(aw.Type)+"）", defenderW-weaponTextX-8)) > 2 ||
		len(wrapWidth("▲ "+dw.Name+"（"+weaponTypeName(dw.Type)+"）", attackerW-weaponTextX-16)) > 2)
	for _, side := range []struct {
		x, room int
		name    string
	}{{16, defenderW - 8, forecastName(f.Attacker, units[c.unit].Boss)}, {l.defX, attackerW - 16, forecastName(f.Defender, units[c.target].Boss)}} {
		lines := wrapWidth(side.name, side.room)
		y := l.nameY
		if compactHeader {
			y = 8
		} else if len(lines) > 1 {
			// 長い名前は帯のない盤面窓の上余白、または右欄の名前の高さを使う（DES-UI-31）。
			y -= 4
		}
		for i, line := range lines {
			add(side.x, y+i*lineHeight, line, StyleTitle)
		}
	}

	// 武器の欄: 両側に種類の絵（WeaponIconSize）と名前・種類、その下に相性の印。攻撃する側は仮選択の武器、守る側は装備（DES-ITM-02）。
	tri := Triangle(aw.Type, dw.Type)
	for i, side := range []struct {
		x, room int
		w       tactics.Weapon
		mark    int
	}{{16, defenderW, aw, tri}, {l.defX, attackerW - 8, dw, -tri}} {
		if tile, ok := weaponTile(side.w.Type); ok {
			v.Icons = append(v.Icons, InfoIcon{X: r.Min.X + side.x, Y: r.Min.Y + l.weaponY, Size: WeaponIconSize, Tile: tile})
		}
		tx := side.x + weaponTextX
		room := side.room - weaponTextX - 8
		textY := l.weaponY
		if compactHeader {
			textY = 40
		}
		weaponName := side.w.Name + "（" + weaponTypeName(side.w.Type) + "）"
		weaponLines := wrapWidth(weaponName, room)
		if len(weaponLines) == 1 {
			add(tx, textY, weaponName, StylePlain)
			switch side.mark {
			case 1:
				add(tx, textY+lineHeight, "▲", StyleGood)
			case -1:
				add(tx, textY+lineHeight, "▼", StyleBad)
			}
		} else {
			// 絵の右の 2 行を武器名に使い、相性は絵の上端に置く（DES-UI-31）。
			mark := ""
			if side.mark > 0 {
				mark = "▲"
			} else if side.mark < 0 {
				mark = "▼"
			}
			if mark != "" {
				weaponLines = wrapWidth(mark+" "+weaponName, room)
			}
			for j, line := range weaponLines {
				add(tx, textY+j*lineHeight, line, StylePlain)
			}
		}
		// 直接クリックで武器ごとに仮の位置が違うときは、窓に仮の移動先を示す（DES-UI-34）。
		if i == 0 && c.quickSpots != nil && len(weaponLines) == 1 {
			add(tx+20, textY+lineHeight, fitRewind(fmt.Sprintf("位置 %d,%d", c.dest.X, c.dest.Y), room-20), StyleLabel)
		}
	}

	// 6 行の表は Forecast から作る。必殺と大盾は予測後 HP に織り込まない（ForecastAfter）。
	afterA, afterD := ForecastAfter(f)
	counter := "なし"
	if f.Defender.Count > 0 {
		counter = "あり"
	}
	for i, row := range [][3]string{
		{"HP", hpText(f.Attacker.HP, afterA), hpText(f.Defender.HP, afterD)},
		{"ダメージ", damageCountText(f.Attacker), damageCountText(f.Defender)},
		{"命中", hitRateText(f.Attacker), hitRateText(f.Defender)},
		{"必殺", critText(f.Attacker), critText(f.Defender)},
		{"速さ", speedText(f.Attacker), speedText(f.Defender)},
		{"反撃", "―", counter},
	} {
		y := l.tableY + i*l.rowGap
		add(16, y, row[0], StyleLabel)
		add(forecastValueX, y, fitRewind(row[1], l.defX-forecastValueX-8), StylePlain)
		add(l.defX, y, fitRewind(row[2], attackerW-8), StylePlain)
	}

	// 効くスキルと特効は各側の欄で折り返し、欄の行数で止める。
	for _, side := range []struct {
		x, room int
		s       tactics.ForecastSide
	}{{16, defenderW - 8, f.Attacker}, {l.defX, attackerW - 8, f.Defender}} {
		skills := strings.Join(forecastSkills(side.s), "・")
		if skills == "" {
			skills = "―"
		}
		for i, line := range wrapWidth("スキル "+skills+"　特効 "+effectiveText(side.s), side.room) {
			if i >= l.skillLines {
				break
			}
			add(side.x, l.skillY+i*lineHeight, line, StyleSkill)
		}
	}

	// 武器送り・決定・戻る（DES-UI-34）。押せる範囲は ForecastClick と同じ。
	room := width - 2*forecastButtonWidth - 8 - 16 - textWidth("◀  ▶　仮選択")
	controlName := aw.Name
	if textWidth(controlName) > room {
		controlName = "武器"
	}
	add(16, l.controlY, "◀ "+controlName+" ▶　仮選択", StyleTitle)
	add(width-2*forecastButtonWidth, l.controlY, "決定", StyleGood)
	add(width-forecastButtonWidth, l.controlY, "戻る", StylePlain)
	v.Bottom = r.Max.Y
	return w, true
}

// damageCountText は通常ダメージと攻撃の回数を「5×2」の形にする（DES-UI-31）。回数 0 は「反撃なし」。
func damageCountText(s tactics.ForecastSide) string {
	if s.Count == 0 {
		return "反撃なし"
	}
	return fmt.Sprintf("%d×%d", s.Damage, s.Count)
}

// forecastContains は (x, y) が予測窓の中かを返す（DES-UI-31）。クリックとホイールで同じ範囲を使う。
func (c *Controller) forecastContains(x, y int) bool {
	w, ok := c.ForecastWindow()
	return ok && image.Pt(x, y).In(w.Rect)
}

// ForecastClick は予測窓の中のクリックを受ける（DES-UI-31・34、DES-IT07 3.1 節の優先）。窓の中なら真を返し、盤面・右欄に渡さない。
// 武器送りの行の左端 40 px は ◀、右端の 48 px は「戻る」、その左の 48 px は「決定」、ほかは ▶。行の外は何もしない。
func (c *Controller) ForecastClick(x, y int) bool {
	w, ok := c.ForecastWindow()
	if !ok || !image.Pt(x, y).In(w.Rect) {
		return false
	}
	if c.state != Targeting {
		return true
	}
	l := forecastBoardLayout
	if w.Fallback {
		l = forecastFallbackLayout
	}
	if y < w.Rect.Min.Y+l.controlY || y >= w.Rect.Min.Y+l.controlY+lineHeight+8 {
		return true
	}
	rel := x - w.Rect.Min.X
	switch {
	case rel < 40:
		c.ShiftForecastWeapon(-1)
	case rel >= w.Rect.Dx()-forecastButtonWidth:
		c.Cancel()
	case rel >= w.Rect.Dx()-2*forecastButtonWidth:
		c.ConfirmForecast()
	default:
		c.ShiftForecastWeapon(1)
	}
	return true
}

// 右欄の上下 2 段と操作行の配置（DES-IT07 3.4 節）。
const (
	// infoLowerTop は下段の上端、infoControlTop は操作行の上端。上段は y = 0〜176。
	infoLowerTop   = 176
	infoControlTop = 304
	// summaryX は段の文の左端。summaryPortraitX は段の右上に置くユニットの絵の左端。
	summaryX         = MapSize + infoMargin
	summaryPortraitX = ScreenWidth - infoMargin - PortraitSize
	// summaryWide・summaryNarrow は段の文の幅の上限。絵の横の 4 行は絵に重ねない。
	summaryWide   = InfoWidth - 3*infoMargin
	summaryNarrow = summaryPortraitX - infoMargin - summaryX
	// summaryTerrainY は下段の最下行（地形）の y。
	summaryTerrainY = infoControlTop - 3*lineHeight/2
)

// detailButtonRect は操作行の「詳細 I」（DES-UI-31）。描画とクリックで同じ範囲を使う。
var detailButtonRect = image.Rect(672, infoControlTop, 744, infoControlTop+32)

// BattleInfoView は戦闘の右欄の上下 2 段を返す（DES-UI-31）。中身は y < 296 に収め、操作行（y = 304〜）の範囲の状態・詳細・
// 消すは別に描く。
// 上段はカーソルの下の生きているユニット（いなければ選択中）、下段は選択中のユニット。同じ人物なら下段は地形・スキルの要約。
// 地形は下段の最下行に、カーソルのマス（なければ選択中のユニットのマス）のものを出す。
// カーソルが右欄の上にあるときは、直前に盤面で見ていたマスを使う（DES-UI-28）。
func (c *Controller) BattleInfoView(x, y int) InfoView {
	units := c.Units()
	cell, onMap := c.Camera().CellAt(x, y)
	if !onMap && image.Pt(x, y).In(image.Rect(MapSize, 0, ScreenWidth, InfoHeight)) {
		cell, onMap = c.lastCell, c.lastCellOK
	}
	upper := -1
	if onMap {
		upper = liveUnitIndex(units, cell)
	}
	selected := -1
	if p, ok := c.selectedCell(); ok {
		selected = c.unit
		if !onMap {
			cell, onMap = p, true
		}
	}
	var terrain *tactics.Terrain
	if onMap && cell.X >= 0 && cell.Y >= 0 && cell.X < c.battle.Width() && cell.Y < c.battle.Height() {
		t := c.battle.TerrainAt(cell)
		terrain = &t
	}
	v := twoTierInfo(units, upper, selected, terrain)
	if terrain != nil && hasZOCAt(c.battle.BoardView(), cell) {
		// 上段の要約の下に置き、下段の地形と操作行を避ける（DES-UI-29、DES-UI-31）。
		v.text(summaryX, infoLowerTop-lineHeight-8, zocNote, StyleLabel)
	}
	return v
}

// liveUnitIndex は units のうちマス p にいる生きているユニットの添字を返す。いなければ −1。
func liveUnitIndex(units []tactics.Unit, p tactics.Pos) int {
	for i, u := range units {
		if u.Alive && u.Pos == p {
			return i
		}
	}
	return -1
}

// twoTierInfo は右欄の上下 2 段を作る（DES-UI-31）。戦闘と配置で共有する。upper・selected は units の添字（なければ −1）。
// 上段がいなければ選択中を上段に出す。terrain が nil でなければ下段の最下行に出す。
func twoTierInfo(units []tactics.Unit, upper, selected int, terrain *tactics.Terrain) InfoView {
	var v InfoView
	if upper < 0 {
		upper = selected
	}
	if upper >= 0 {
		v.addUnitSummary(units[upper], 0)
	}
	switch {
	case selected >= 0 && selected != upper:
		v.addUnitSummary(units[selected], infoLowerTop)
	case selected >= 0:
		// 同じ人物: 下段は地形・スキルの要約にする。
		y := infoLowerTop + infoMargin
		v.text(summaryX, y, "地形・スキル", StyleLabel)
		skills := unitSkills(units[selected])
		if len(skills) == 0 {
			v.text(summaryX, y+lineHeight, "スキル なし", StylePlain)
		}
		for i, sk := range skills {
			if row := y + (i+1)*lineHeight; row+lineHeight <= summaryTerrainY {
				v.text(summaryX, row, fitRewind(sk.Name+"　"+skillDescription(sk), summaryWide), StyleSkill)
			}
		}
	}
	if terrain != nil {
		lines := TerrainLines(*terrain)
		v.text(summaryX, summaryTerrainY, fitRewind(lines[0]+"　"+lines[1], summaryWide), StylePlain)
	}
	// 一番下は最下行（地形）の下端。配置のボタン（y = 296〜）とも重ならない。
	v.Bottom = summaryTerrainY + lineHeight
	return v
}

// addUnitSummary は段の上端 top から、名前・兵種と Lv・HP・装備の名前と種類（E 印）・力/魔力と守備/魔防・速さを
// 1 行ずつ並べ、段の右上にユニットの絵を置く（DES-UI-31、DES-UI-33）。6 行は 96 px で、下段（128 px）の最下行の地形と重ならない。
func (v *InfoView) addUnitSummary(u tactics.Unit, top int) {
	y := top + infoMargin
	v.Icons = append(v.Icons, InfoIcon{X: summaryPortraitX, Y: y, Size: PortraitSize, Tile: unitTile(u), Art: artKeyOf(u)})
	// 上下段とも名前の横に指揮の印と短い語を置く（DES-UI-35）。
	v.addCommandMark(u, summaryX, y, unitName(u))
	equip := "装備 なし"
	if w := u.EquippedWeapon(); w.Name != "" {
		equip = fmt.Sprintf("装備 %s %s（%s）", weaponRowMark, w.Name, weaponTypeName(w.Type))
	}
	s := u.Stats
	for i, row := range []struct {
		text string
		st   InfoStyle
	}{
		{unitName(u), StyleTitle},
		{strings.TrimSpace(u.ClassName + "　" + levelText(u)), StylePlain},
		{fmt.Sprintf("HP %d/%d", u.HP, s.HP), StylePlain},
		{equip, StylePlain},
		{fmt.Sprintf("力/魔力 %d/%d　守備/魔防 %d/%d", s.Str, s.Mag, s.Def, s.Res), StylePlain},
		{fmt.Sprintf("速さ %d", s.Spd), StylePlain},
	} {
		width := summaryWide
		if i < 4 {
			width = summaryNarrow
		}
		text := row.text
		if i > 0 {
			// 名前は敵の範囲の印（nameMarkRect）が文の一致で探すので省かない。
			text = fitRewind(text, width)
		}
		v.text(summaryX, y+i*lineHeight, text, row.st)
	}
}

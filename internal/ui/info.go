package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 情報欄に出す文と行を作り、描くものの位置を決める（DES-UI-01、DES-UI-06、DES-BTL-07）。
// 描画（game.go の drawInfo）から分け、Ebitengine なしで確かめられるようにする。

// Panel は情報欄に出す中身（DES-UI-01）。
type Panel struct {
	// Heal は回復対象を選んだときだけ入る予測（DES-UI-18）。
	Heal *tactics.HealForecast
	// Forecast は攻撃の予測の表（FR-BTL-08、UC-03）。攻撃の相手を選んでいるときだけ入る。
	Forecast []ForecastRow
	// Emphasis は攻撃の予測の強調（DES-UI-07）。
	Emphasis ForecastEmphasis
	// ForecastWeapons は、予測の両側（攻撃する側、守る側）の武器の種類（DES-UI-07）。武器の絵と相性の印に使う。
	// Forecast があるときだけ意味がある。
	ForecastWeapons [2]tactics.WeaponType
	// ForecastAfter は、すべての 1 撃が当たったときの後の HP（攻撃する側、守る側。DES-UI-07）。
	// 「撃破」「危険」に使う。Forecast があるときだけ意味がある。
	ForecastAfter [2]int
	// Unit は、カーソルのあるマスにいる生きているユニット（FR-UNT-09、UC-04）。いなければ nil。
	Unit *tactics.Unit
	// Terrain は、カーソルのあるマスの地形（FR-UNT-09、FR-MAP-01）。カーソルが盤面の外なら nil。
	Terrain *tactics.Terrain
}

// ForecastRow は予測の表の 1 行（DES-BTL-07）。項目の名前、攻撃する側の値、守る側の値を並べる。
type ForecastRow struct {
	Label, Attacker, Defender string
}

// Panel は、カーソルがマス cursor にあるときの情報欄の中身を返す（DES-UI-01）。onMap が偽ならカーソルは盤面の外。
//
// 攻撃の相手を選んでいる（Targeting で相手が決まっている）ときは予測を入れる。
// カーソルのあるマスに生きているユニットがいればそのユニット、続けて地形を入れる。ユニットは盤面に描くもの（Units）から引くので、
// 再生中は出来事の写し、移動先を選んだあとは移動先に置いた写しになる。
func (c *Controller) Panel(cursor tactics.Pos, onMap bool) Panel {
	var p Panel
	units := c.Units()
	if f, ok := c.healForecast(); ok {
		p.Heal = &f
	}
	if f, ok := c.attackForecast(); ok {
		// 相手は Targets から選んでいるので、ふつうは誤りにならない。誤りなら予測を出さない。
		p.Forecast = ForecastRows(f, units[c.unit].Boss, units[c.target].Boss)
		p.ForecastWeapons = [2]tactics.WeaponType{units[c.unit].Weapon.Type, units[c.target].Weapon.Type}
		a, d := ForecastAfter(f)
		p.ForecastAfter = [2]int{a, d}
		p.Emphasis = forecastEmphasis(f)
	}
	if !onMap || cursor.X < 0 || cursor.Y < 0 || cursor.X >= c.battle.Width() || cursor.Y >= c.battle.Height() {
		return p
	}
	for _, u := range units {
		if u.Alive && u.Pos == cursor {
			p.Unit = &u
			break
		}
	}
	t := c.battle.TerrainAt(cursor)
	p.Terrain = &t
	return p
}

// 情報欄の見せ方（DES-UI-06）の大きさと位置（px）。x は情報欄の左端（余白 infoMargin の内側）から数える。
const (
	// PortraitSize はユニットの絵の大きさ。タイルの 4 倍。
	PortraitSize = 4 * assets.TileSize
	// portraitTextX は、絵の右に並べる名前とクラスの位置。
	portraitTextX = PortraitSize + 8
	// WeaponIconSize は武器の種類の絵の大きさ。盤面と同じ 2 倍。
	WeaponIconSize = 2 * assets.TileSize
	// weaponTextX は、武器の絵の右に並べる文の位置。
	weaponTextX = WeaponIconSize + 4
	// skillNameX はスキルの名前の位置（「スキル:」の右）。
	skillNameX = 48
	// barX は能力の棒の左端。
	barX = 64
	// BarMax は棒の長さの上限と、HP の棒の枠の長さ。
	BarMax = 120
	// statBarUnit は能力の値 1 あたりの棒の長さ。
	statBarUnit = 6
	// barHeight は棒の高さ。行の高さの真ん中に置く。
	barHeight = 8
	// blockGap は、絵・能力・武器のまとまりの間の空き。
	blockGap = 4
	// forecastAttackX と forecastDefendX は、予測の表の攻撃する側と守る側の列の位置（DES-BTL-07、DES-UI-07）。
	forecastAttackX = 80
	forecastDefendX = 200
)

// InfoStyle は情報欄の文の色の種類（DES-UI-06）。描画の側で色に読み替える。
type InfoStyle int

const (
	// StylePlain はふつうの文（白）。
	StylePlain InfoStyle = iota
	// StyleTitle はユニットの名前（金）。
	StyleTitle
	// StyleLabel は項目の名前（薄い色）。
	StyleLabel
	// StyleSkill はスキルの名前（黄）。
	StyleSkill
	// StyleGood は有利の印 ▲（緑。DES-UI-07）。
	StyleGood
	// StyleBad は不利の印 ▼（赤。DES-UI-07）。
	StyleBad
	// StyleBlack は撃破の帯の文（黒。DES-UI-07）。
	StyleBlack
	// StyleHint は増援の気配の案内（黄土色。DES-UI-24）。
	StyleHint
	// 閲覧専用の敵軍行と自軍番開始行の薄い文字（DES-UI-21）。
	StyleRewindEnemy
	StyleRewindBoundary
)

// InfoText は情報欄に描く文（DES-UI-06）。(X, Y) は文の左上の画面の座標。
// Big が真なら 2 倍の大きさで描く（相性の印・帯の見出し。DES-UI-07）。
type InfoText struct {
	X, Y  int
	Text  string
	Style InfoStyle
	Big   bool
}

// InfoBar は能力の棒（DES-UI-06）。(X, Y) は左上の画面の座標で、Fill の長さを塗る。
// Frame が 0 でなければ、先に長さ Frame の枠（下地）を描く（HP の棒）。
type InfoBar struct {
	X, Y, Fill, Frame int
}

// InfoIcon は情報欄に描く絵（DES-UI-06）。(X, Y) は左上の画面の座標、Size は描く大きさ。
type InfoIcon struct {
	X, Y, Size int
	Tile       assets.Tile
}

// InfoView は、情報欄に描くもの（DES-UI-06、DES-UI-07）。予測の表、ユニット、地形の順に上から並べる。
type InfoView struct {
	// Banner は撃破・危険の帯。通常は nil（DES-UI-07）。
	Banner *InfoBanner
	Texts  []InfoText
	Bars   []InfoBar
	Icons  []InfoIcon
	// Bottom は描くものの一番下の y（画面の座標）。
	Bottom int
}

// View は、情報欄に描くものと位置を返す（DES-UI-01、DES-UI-06、DES-UI-07）。
// 予測の表があれば上に置き、ユニットと地形はその下（見出し・武器の絵・表・空き 1 行のあと）から並べる。
func (p Panel) View() InfoView {
	x := MapSize + infoMargin
	y := infoMargin
	var v InfoView
	if p.Heal != nil {
		f := p.Heal
		for i, s := range []string{"回復", f.Healer + " → " + f.Target, "HP " + hpText(f.HP, f.After), fmt.Sprintf("回復 +%d", f.Amount), fmt.Sprintf("杖 残り %d → %d/%d", f.Uses, f.Uses-1, f.MaxUses)} {
			st := StylePlain
			if i == 0 {
				st = StyleGood
			}
			v.text(x, y, s, st)
			y += lineHeight
		}
		y += lineHeight
	}
	if len(p.Forecast) > 0 {
		// 予測の表は速さと特効を分けて 10 行になったので、表のあとの空き行を省いて情報欄の高さを保つ（REV-069 F-03）。
		y = v.addForecast(p, x, y)
	}
	if p.Unit != nil {
		y = v.addUnit(*p.Unit, x, y)
		if len(p.Forecast) == 0 {
			y += lineHeight
		}
	}
	if p.Terrain != nil {
		for _, s := range TerrainLines(*p.Terrain) {
			v.text(x, y, s, StylePlain)
			y += lineHeight
		}
	}
	v.Bottom = y
	return v
}

// addForecast は帯・見出し・武器の絵・相性の印・表を並べる（DES-BTL-07、DES-UI-07）。
func (v *InfoView) addForecast(p Panel, x, y int) int {
	if p.Emphasis != ForecastNormal {
		st, fg := StyleTitle, StyleBlack
		title, desc := "撃破", "この攻撃で倒せる"
		if p.Emphasis == ForecastDanger {
			st, fg = StyleBad, StylePlain
			title, desc = "危険", "反撃で倒される"
		}
		v.Banner = &InfoBanner{Rect: image.Rect(x, y, x+InfoWidth-2*infoMargin, y+40), Style: st}
		v.Texts = append(v.Texts, InfoText{X: x + 8, Y: y + 4, Text: title, Style: fg, Big: true})
		v.text(x+80, y+12, desc, fg)
		y += 44
	}
	v.text(x, y, "攻撃の予測", StyleTitle)
	y += lineHeight

	// 両側の列の頭に武器の種類の絵を置き、絵の右に相性の印（2 倍。絵と同じ高さ）を置く。
	t := Triangle(p.ForecastWeapons[0], p.ForecastWeapons[1])
	marks := [2]int{t, -t}
	for i, cx := range [2]int{x + forecastAttackX, x + forecastDefendX} {
		if tl, ok := weaponTile(p.ForecastWeapons[i]); ok {
			v.Icons = append(v.Icons, InfoIcon{X: cx, Y: y, Size: WeaponIconSize, Tile: tl})
		}
		switch marks[i] {
		case 1:
			v.Texts = append(v.Texts, InfoText{X: cx + weaponTextX, Y: y, Text: "▲", Style: StyleGood, Big: true})
		case -1:
			v.Texts = append(v.Texts, InfoText{X: cx + weaponTextX, Y: y, Text: "▼", Style: StyleBad, Big: true})
		}
	}
	y += WeaponIconSize

	// 表は、項目の名前を左に、攻撃する側と守る側の値を列に並べる。名前の行は金にする。
	for i, r := range p.Forecast {
		st := StylePlain
		if i == 0 {
			st = StyleTitle
		}
		if r.Label != "" {
			v.text(x, y, r.Label, StyleLabel)
		}
		if r.Attacker != "" {
			v.text(x+forecastAttackX, y, r.Attacker, st)
		}
		if r.Defender != "" {
			v.text(x+forecastDefendX, y, r.Defender, st)
		}
		y += lineHeight
	}
	return y
}

func (v *InfoView) text(x, y int, s string, st InfoStyle) {
	v.Texts = append(v.Texts, InfoText{X: x, Y: y, Text: s, Style: st})
}

// addUnit は、ユニットの絵・名前・クラス・HP と能力の棒・移動・武器・手持ち・スキルを (x, y) から並べ、
// 使った一番下の y を返す。自軍のレベルと経験は名前・クラスの下に置く（FR-UNT-09、DES-UI-06、DES-UI-16）。
func (v *InfoView) addUnit(u tactics.Unit, x, y int) int {
	// 名前の横に、盤面と同じタイルを 4 倍で描く。レベルがある自軍は絵の横に3行で並べる。
	v.Icons = append(v.Icons, InfoIcon{X: x, Y: y, Size: PortraitSize, Tile: unitTile(u)})
	lines := 2
	showLevel := u.Side == tactics.Player && u.Level > 0
	if showLevel {
		lines = 3
	}
	ty := y + (PortraitSize-lines*lineHeight)/2
	v.text(x+portraitTextX, ty, unitName(u), StyleTitle)
	v.text(x+portraitTextX, ty+lineHeight, "クラス: "+u.ClassName, StylePlain)
	if showLevel {
		exp := fmt.Sprint(u.Exp)
		if u.Level == 20 {
			exp = "―"
		}
		v.text(x+portraitTextX, ty+2*lineHeight, fmt.Sprintf("Lv %d　経験 %s", u.Level, exp), StylePlain)
	}
	y += PortraitSize + blockGap

	// HP は枠に割合で塗り、力・技・速さ・守備は値に比例した棒にする。移動は数字だけ。
	s := u.Stats
	v.text(x, y, fmt.Sprintf("HP %d/%d", u.HP, s.HP), StylePlain)
	v.Bars = append(v.Bars, InfoBar{X: x + barX, Y: y + (lineHeight-barHeight)/2, Fill: HPBarLen(u.HP, s.HP), Frame: BarMax})
	y += lineHeight
	for i, st := range []struct {
		name  string
		value int
	}{{"力", s.Str}, {"技", s.Skl}, {"速さ", s.Spd}, {"守備", s.Def}} {
		v.text(x, y, fmt.Sprintf("%s %d", st.name, st.value), StylePlain)
		v.Bars = append(v.Bars, InfoBar{X: x + barX, Y: y + (lineHeight-barHeight)/2, Fill: StatBarLen(st.value)})
		if i < 3 {
			labels := []string{"魔力", "幸運", "魔防"}
			values := []int{s.Mag, s.Lck, s.Res}
			v.text(x+200, y, fmt.Sprintf("%s %d", labels[i], values[i]), StylePlain)
		}
		y += lineHeight
	}
	v.text(x, y, fmt.Sprintf("移動 %d", s.Mov), StylePlain)
	y += lineHeight + blockGap

	// 武器の行は、種類の絵を左に置き、右に名前と威力・命中・射程の 2 行を並べる。
	w := u.Weapon
	if t, ok := weaponTile(w.Type); ok {
		v.Icons = append(v.Icons, InfoIcon{X: x, Y: y, Size: WeaponIconSize, Tile: t})
	}
	v.text(x+weaponTextX, y, fmt.Sprintf("武器: %s（%s）", w.Name, weaponTypeName(w.Type)), StylePlain)
	if w.Type == tactics.Staff {
		v.text(x+weaponTextX, y+lineHeight, fmt.Sprintf("回復 %d　射程 %s　残り %d/%d", w.Heal, rangeText(w), w.Uses, w.MaxUses), StylePlain)
	} else {
		v.text(x+weaponTextX, y+lineHeight, fmt.Sprintf("威力 %d　命中 %d　射程 %s", w.Might, w.Hit, rangeText(w)), StylePlain)
	}
	v.text(x+200, y, fmt.Sprintf("必殺 %d ×%d", w.Crit, max(1, w.Strikes)), StylePlain)
	v.text(x+200, y+lineHeight, fmt.Sprintf("重さ %d", w.Weight), StylePlain)
	y += WeaponIconSize

	v.text(x, y, "手持ち: "+itemsText(u.Items), StylePlain)
	y += lineHeight

	// スキルは名前を黄で 1 行に出し、次の行に効果の文を出す。キャラのスキルのあとに兵種のスキルを並べる（DES-IT06 12.2 節）。
	v.text(x, y, "スキル:", StyleLabel)
	skills := unitSkills(u)
	if len(skills) == 0 {
		v.text(x+skillNameX, y, "なし", StylePlain)
		return y + lineHeight
	}
	names := make([]string, len(skills))
	descs := make([]string, len(skills))
	for i, sk := range skills {
		names[i], descs[i] = sk.Name, skillDescription(sk)
	}
	v.text(x+skillNameX, y, strings.Join(names, "、"), StyleSkill)
	y += lineHeight
	v.text(x, y, "　"+strings.Join(descs, "／"), StylePlain)
	return y + lineHeight
}

// unitSkills は、キャラのスキル（あれば）と兵種のスキルを、この順に返す（DES-BTL-10、DES-IT06 8.1・12.2 節）。
func unitSkills(u tactics.Unit) []tactics.Skill {
	var out []tactics.Skill
	if u.Skill.Effect != tactics.NoSkill {
		out = append(out, u.Skill)
	}
	return append(out, u.ClassSkills...)
}

// unitName はユニットの名前を返す。ボスは名前ではボスと分からない（例「ソルジャー 3」。REV-010 F-03）ので、盤面と同じ ★ の印を添える。
func unitName(u tactics.Unit) string {
	if u.Boss {
		return u.Name + "　★ボス"
	}
	return u.Name
}

// StatBarLen は、力・技・速さ・守備の棒の長さ（px）を返す（DES-UI-06）。値 × 6 で、BarMax（値 20）で止める。
func StatBarLen(value int) int {
	return min(max(value, 0)*statBarUnit, BarMax)
}

// HPBarLen は、HP の棒の塗る長さ（px）を返す（DES-UI-06）。長さ BarMax の枠に、今の HP ÷ 最大 HP の割合で塗る。
// 最大 HP が 0 以下なら塗らない。
func HPBarLen(hp, maxHP int) int {
	if maxHP <= 0 {
		return 0
	}
	return min(max(hp, 0), maxHP) * BarMax / maxHP
}

// TerrainLines は、地形の名前・回避・守備・回復の行を返す（FR-UNT-09、FR-MAP-01、DES-UI-01）。
func TerrainLines(t tactics.Terrain) []string {
	return []string{
		"地形: " + t.Name,
		fmt.Sprintf("回避 %d　守備 %d　回復 %d%%", t.Avoid, t.Def, t.HealPercent),
	}
}

// ForecastRows は、予測を表の行にする（DES-BTL-07、DES-UI-07、FR-BTL-08、UC-03）。
// 名前（ボスなら ★）、HP（今 → すべて当たったときの後。ForecastAfter）、ダメージ（1 撃あたり）、命中率、攻撃の回数、必殺、
// 必殺率・速さ（重さで落ちた分）・特効（効く威力）・効くスキルを並べる。回数は攻撃側最大4回、反撃側最大2回（DES-UI-26）。
//
// 回数0の側では、ダメージ・命中率・必殺率を「―」にして怒りを出さない（DES-UI-26）。
func ForecastRows(f tactics.Forecast, attackerBoss, defenderBoss bool) []ForecastRow {
	a, d := f.Attacker, f.Defender
	aAfter, dAfter := ForecastAfter(f)
	rows := []ForecastRow{
		{"", forecastName(a, attackerBoss), forecastName(d, defenderBoss)},
		{"HP", hpText(a.HP, aAfter), hpText(d.HP, dAfter)},
		{"ダメージ", damageText(a), damageText(d)},
		{"命中率", hitRateText(a), hitRateText(d)},
		{"回数", countText(a.Count), countText(d.Count)},
		{"必殺", critText(a), critText(d)},
		{"速さ", speedText(a), speedText(d)},
		{"特効", effectiveText(a), effectiveText(d)},
	}
	as, ds := forecastSkills(a), forecastSkills(d)
	n := max(len(as), len(ds), 1)
	for i := 0; i < n; i++ {
		row := ForecastRow{Attacker: "―", Defender: "―"}
		if i == 0 {
			row.Label = "スキル"
		} else {
			row.Attacker, row.Defender = "", ""
		}
		if i < len(as) {
			row.Attacker = as[i]
		}
		if i < len(ds) {
			row.Defender = ds[i]
		}
		rows = append(rows, row)
	}
	return rows
}

// ForecastAfter は、すべての 1 撃が当たったものとしたときの、攻撃する側と守る側の後の HP を返す（DES-UI-07）。
// 攻撃側の最初の機会（武器によって1〜2撃）→反撃→速い側の追撃の順で引く（DES-BTL-09）。
// HP は 0 で止め、どちらかが 0 になったらそこで止める。必殺と大盾は数えない。
func ForecastAfter(f tactics.Forecast) (attacker, defender int) {
	attacker, defender = f.Attacker.HP, f.Defender.HP
	// strike は hp から dmg を引き（0 で止める）、倒れたら真を返す。
	strike := func(hp *int, dmg int) bool {
		*hp = max(*hp-max(dmg, 0), 0)
		return *hp == 0
	}

	first := min(f.Attacker.Count, max(1, f.Attacker.Strikes))
	for range first {
		if strike(&defender, f.Attacker.Damage) {
			return
		}
	}
	if f.Defender.Count > 0 && strike(&attacker, f.Defender.Damage) {
		return
	}
	for range f.Attacker.Count - first {
		if strike(&defender, f.Attacker.Damage) {
			return
		}
	}
	if f.Defender.Count == 2 {
		strike(&attacker, f.Defender.Damage)
	}

	return
}

// Triangle は、武器の種類 a で種類 d を攻撃するときの、攻撃する側の相性 t を返す（DES-BTL-05 の表、DES-UI-07）。
// 剣は斧に、斧は槍に、槍は剣に有利で +1、その逆は −1、それ以外（弓を含む）は 0。
// tactics の相性の関数は公開されていないので、画面の側で同じ表から求める（REV-016 F-06）。
func Triangle(a, d tactics.WeaponType) int {
	beats := func(x, y tactics.WeaponType) bool {
		return (x == tactics.Sword && y == tactics.Axe) || (x == tactics.Axe && y == tactics.Lance) || (x == tactics.Lance && y == tactics.Sword)
	}
	switch {
	case beats(a, d):
		return 1
	case beats(d, a):
		return -1
	}
	return 0
}

// TargetLabel は、撃破・危険を頭に付けた予測の文と、相手のマスを返す（DES-UI-07）。
// 回数が 1 なら「×1」を省く。攻撃の相手を選んでいなければ ok が偽。
func (c *Controller) TargetLabel() (s string, p tactics.Pos, ok bool) {
	v := c.TargetView()
	return v.Text, v.Pos, v.Visible
}

// ForecastEmphasis は攻撃の予測の強調状態（DES-UI-07）。
type ForecastEmphasis int

const (
	// ForecastNormal は通常の予測。
	ForecastNormal ForecastEmphasis = iota
	// ForecastDefeat は相手を倒せる予測。
	ForecastDefeat
	// ForecastDanger は反撃で倒される予測。
	ForecastDanger
)

// InfoBanner は情報欄の帯の範囲と下地の色（DES-UI-07）。
type InfoBanner struct {
	Rect  image.Rectangle
	Style InfoStyle
}

// TargetView は盤面の予測の文字と危険枠（DES-UI-07）。
type TargetView struct {
	Text      string
	Pos       tactics.Pos
	Style     InfoStyle
	Visible   bool
	Danger    bool
	DangerPos tactics.Pos
}

// attackForecast は攻撃の相手を選んでいるときだけ予測を作る（DES-UI-07）。
// 回復など別の状態には攻撃の予測を返さない。
func (c *Controller) attackForecast() (tactics.Forecast, bool) {
	if c.state != Targeting || c.target < 0 {
		return tactics.Forecast{}, false
	}
	f, err := c.battle.Forecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Attack, Target: c.target})
	return f, err == nil
}

// forecastEmphasis は通常の命中を順に引いた HP から強調を決める（DES-UI-07）。
func forecastEmphasis(f tactics.Forecast) ForecastEmphasis {
	a, d := ForecastAfter(f)
	if a == 0 {
		return ForecastDanger
	}
	if d == 0 {
		return ForecastDefeat
	}
	return ForecastNormal
}

// TargetView は攻撃の相手の文字・色と、自分の移動先の危険枠を返す（DES-UI-07）。
func (c *Controller) TargetView() TargetView {
	if f, ok := c.healForecast(); ok {
		return TargetView{Text: fmt.Sprintf("+%d", f.Amount), Pos: c.Units()[c.target].Pos, Style: StyleGood, Visible: true}
	}
	f, ok := c.attackForecast()
	if !ok {
		return TargetView{}
	}
	a := f.Attacker
	s := fmt.Sprint(a.Damage)
	if a.Count != 1 {
		s += fmt.Sprintf("×%d", a.Count)
	}
	s += fmt.Sprintf(" %d%%", a.HitRate)
	v := TargetView{Text: s, Pos: c.Units()[c.target].Pos, Visible: true, DangerPos: c.dest}
	switch forecastEmphasis(f) {
	case ForecastDefeat:
		v.Text = "撃破 " + s
		v.Style = StyleTitle
	case ForecastDanger:
		v.Text = "危険 " + s
		v.Style = StyleBad
		v.Danger = true
	}
	return v
}

func forecastName(s tactics.ForecastSide, boss bool) string {
	if boss {
		return s.Name + "★"
	}
	return s.Name
}

// hpText は HP の「今 → 後」の文を返す（DES-UI-07）。
func hpText(now, after int) string {
	return fmt.Sprintf("%d → %d", now, after)
}

// damageText は 1 撃あたりのダメージの文を返す。攻撃の回数が 0 なら「―」（DES-UI-07）。
func damageText(s tactics.ForecastSide) string {
	if s.Count == 0 {
		return "―"
	}
	return fmt.Sprint(s.Damage)
}

// hitRateText は命中率の文を返す。攻撃の回数が 0 なら「―」（DES-UI-07）。
func hitRateText(s tactics.ForecastSide) string {
	if s.Count == 0 {
		return "―"
	}
	return fmt.Sprintf("%d%%", s.HitRate)
}

// countText は攻撃の回数の文を返す（DES-BTL-07）。
func countText(n int) string {
	if n == 0 {
		return "反撃なし"
	}
	return fmt.Sprintf("×%d", n)
}

// critText は必殺の文を返す。攻撃の回数が 0 なら出さない（REV-011 F-04）。
func critText(s tactics.ForecastSide) string {
	if s.Count > 0 {
		return fmt.Sprintf("%d%%", s.CritRate)
	}
	return "―"
}

// forecastSkills は、効くスキルの名前を並べる（DES-SKL-01 の「予測への表示」）。大盾は「大盾 30%」のように確率を添える。
// 攻撃の回数が 0 なら怒りを出さない（REV-011 F-04）。
func forecastSkills(s tactics.ForecastSide) []string {
	var out []string
	for _, sk := range s.Skills {
		switch {
		case sk.Effect == tactics.Wrath && s.Count == 0:
			continue
		case sk.Effect == tactics.Wrath:
			out = append(out, fmt.Sprintf("HP半分以下必殺+%d", sk.Chance))
		case sk.Effect == tactics.Pavise:
			out = append(out, fmt.Sprintf("%s %d%%", sk.Name, sk.Chance))
		default:
			out = append(out, sk.Name)
		}
	}
	return out
}

// weaponTypeName は武器の種類の名前を返す。
func weaponTypeName(t tactics.WeaponType) string {
	switch t {
	case tactics.Sword:
		return "剣"
	case tactics.Lance:
		return "槍"
	case tactics.Axe:
		return "斧"
	case tactics.Staff:
		return "杖"
	case tactics.Magic:
		return "魔法"
	case tactics.Bow:
		return "弓"
	}
	return "？"
}

// rangeText は射程の文を返す。最小と最大が同じなら 1 つだけ出す。
func rangeText(w tactics.Weapon) string {
	if w.MinRange == w.MaxRange {
		return fmt.Sprint(w.MinRange)
	}
	return fmt.Sprintf("%d〜%d", w.MinRange, w.MaxRange)
}

// itemsText は手持ちの名前と数を並べる。なければ「なし」。
func itemsText(items []tactics.ItemStack) string {
	if len(items) == 0 {
		return "なし"
	}
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = fmt.Sprintf("%s ×%d", s.Item.Name, s.Count)
	}
	return strings.Join(parts, "、")
}

// skillDescription はスキルの効果の短い説明を返す（DES-SKL-01）。数値はスキルの値から作る。
func skillDescription(s tactics.Skill) string {
	switch s.Effect {
	case tactics.Inspire:
		return fmt.Sprintf("距離 %d 以内の味方の命中 +%d・回避 +%d", s.Range, s.Hit, s.Avoid)
	case tactics.Pavise:
		return fmt.Sprintf("攻撃を受けると %d%% でダメージ半分", s.Chance)
	case tactics.Wrath:
		return fmt.Sprintf("HP 半分以下で必殺 ＋%d", s.Chance)
	case tactics.Snipe:
		return fmt.Sprintf("移動せずに攻撃すると命中 +%d", s.Hit)
	case tactics.Canto:
		return "攻撃・回復・道具のあと、残りの移動で動ける"
	}
	return ""
}

// healForecast は回復対象を選んでいるときだけ core の予測を返す（DES-UI-18）。
func (c *Controller) healForecast() (tactics.HealForecast, bool) {
	if c.state != HealTargeting || c.target < 0 {
		return tactics.HealForecast{}, false
	}
	f, err := c.battle.HealForecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Heal, Target: c.target})
	return f, err == nil
}

// speedText は予測での攻撃の速さと、武器の重さで落ちた分（DES-UI-26、DES-IT06 12.2 節）。例:「11（重さ −1）」。
func speedText(s tactics.ForecastSide) string {
	if s.WeightLoss > 0 {
		return fmt.Sprintf("%d（重さ −%d）", s.AttackSpeed, s.WeightLoss)
	}
	return fmt.Sprintf("%d", s.AttackSpeed)
}

// effectiveText は予測での特効と、効く威力（DES-UI-26、DES-IT06 12.2 節）。例:「威力 12」。効かなければ「―」。
func effectiveText(s tactics.ForecastSide) string {
	if s.Effective {
		return fmt.Sprintf("威力 %d", s.EffectiveMight)
	}
	return "―"
}

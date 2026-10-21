package ui

import (
	"fmt"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 情報欄に出す文と行を作り、描くものの位置を決める（DES-UI-01、DES-UI-06、DES-BTL-07）。
// 描画（info_draw.go）から分け、Ebitengine なしで確かめられるようにする。

// Panel は情報欄に出す中身（DES-UI-01）。
type Panel struct {
	// Forecast は攻撃の予測の表（FR-BTL-08、UC-03）。攻撃の相手を選んでいるときだけ入る。
	Forecast []ForecastRow
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
	if c.state == Targeting && c.target >= 0 {
		f, err := c.battle.Forecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Attack, Target: c.target})
		// 相手は Targets から選んでいるので、ふつうは誤りにならない。誤りなら予測を出さない。
		if err == nil {
			p.Forecast = ForecastRows(f, units[c.unit].Boss, units[c.target].Boss)
		}
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
)

// InfoText は情報欄に描く文（DES-UI-06）。(X, Y) は文の左上の画面の座標。
type InfoText struct {
	X, Y  int
	Text  string
	Style InfoStyle
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

// InfoView は、情報欄のユニットと地形の部分に描くもの（DES-UI-06）。予測の表は今の描き方のまま別に描く。
type InfoView struct {
	Texts []InfoText
	Bars  []InfoBar
	Icons []InfoIcon
	// Bottom は描くものの一番下の y（画面の座標）。
	Bottom int
}

// View は、情報欄のユニットと地形の部分に描くものと位置を返す（DES-UI-01、DES-UI-06）。
// 予測の表があれば、その下（見出し・表・空き 1 行のあと）から並べる。
func (p Panel) View() InfoView {
	x := MapSize + infoMargin
	y := infoMargin
	if len(p.Forecast) > 0 {
		y += (len(p.Forecast) + 2) * lineHeight
	}
	var v InfoView
	if p.Unit != nil {
		y = v.addUnit(*p.Unit, x, y) + lineHeight
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

func (v *InfoView) text(x, y int, s string, st InfoStyle) {
	v.Texts = append(v.Texts, InfoText{X: x, Y: y, Text: s, Style: st})
}

// addUnit は、ユニットの絵・名前・クラス・HP と能力の棒・移動・武器・手持ち・スキルを (x, y) から並べ、
// 使った一番下の y を返す（FR-UNT-09、DES-UI-06）。
func (v *InfoView) addUnit(u tactics.Unit, x, y int) int {
	// 名前の横に、盤面と同じタイルを 4 倍で描く。名前とクラスは絵の高さの真ん中に 2 行で並べる。
	v.Icons = append(v.Icons, InfoIcon{X: x, Y: y, Size: PortraitSize, Tile: unitTile(u)})
	ty := y + (PortraitSize-2*lineHeight)/2
	v.text(x+portraitTextX, ty, unitName(u), StyleTitle)
	v.text(x+portraitTextX, ty+lineHeight, "クラス: "+u.ClassName, StylePlain)
	y += PortraitSize + blockGap

	// HP は枠に割合で塗り、力・技・速さ・守備は値に比例した棒にする。移動は数字だけ。
	s := u.Stats
	v.text(x, y, fmt.Sprintf("HP %d/%d", u.HP, s.HP), StylePlain)
	v.Bars = append(v.Bars, InfoBar{X: x + barX, Y: y + (lineHeight-barHeight)/2, Fill: HPBarLen(u.HP, s.HP), Frame: BarMax})
	y += lineHeight
	for _, st := range []struct {
		name  string
		value int
	}{{"力", s.Str}, {"技", s.Skl}, {"速さ", s.Spd}, {"守備", s.Def}} {
		v.text(x, y, fmt.Sprintf("%s %d", st.name, st.value), StylePlain)
		v.Bars = append(v.Bars, InfoBar{X: x + barX, Y: y + (lineHeight-barHeight)/2, Fill: StatBarLen(st.value)})
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
	v.text(x+weaponTextX, y+lineHeight, fmt.Sprintf("威力 %d　命中 %d　射程 %s", w.Might, w.Hit, rangeText(w)), StylePlain)
	y += WeaponIconSize

	v.text(x, y, "手持ち: "+itemsText(u.Items), StylePlain)
	y += lineHeight

	// スキルは名前を黄で 1 行に出し、次の行に効果の文を出す。
	v.text(x, y, "スキル:", StyleLabel)
	if u.Skill.Effect == tactics.NoSkill {
		v.text(x+skillNameX, y, "なし", StylePlain)
		return y + lineHeight
	}
	v.text(x+skillNameX, y, u.Skill.Name, StyleSkill)
	y += lineHeight
	v.text(x, y, "　"+skillDescription(u.Skill), StylePlain)
	return y + lineHeight
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

// ForecastRows は、予測を表の行にする（DES-BTL-07、FR-BTL-08、UC-03）。
// 名前（ボスなら ★）、今の HP、ダメージ（1 撃あたり）、命中率、攻撃の回数、必殺、効くスキルの順に並べる。
// 回数は、反撃できなければ「反撃なし」、追撃すれば「×2」。
//
// 攻撃の回数が 0 の側では、必殺と怒りを出さない。Forecast の Crit と怒りは回数によらず条件だけで真になるが、
// 攻撃しない側の必殺は起きないため（REV-011 F-04。表示だけで決め、tactics は変えない）。
func ForecastRows(f tactics.Forecast, attackerBoss, defenderBoss bool) []ForecastRow {
	a, d := f.Attacker, f.Defender
	rows := []ForecastRow{
		{"", forecastName(a, attackerBoss), forecastName(d, defenderBoss)},
		{"HP", fmt.Sprint(a.HP), fmt.Sprint(d.HP)},
		{"ダメージ", fmt.Sprint(a.Damage), fmt.Sprint(d.Damage)},
		{"命中率", fmt.Sprintf("%d%%", a.HitRate), fmt.Sprintf("%d%%", d.HitRate)},
		{"回数", countText(a.Count), countText(d.Count)},
		{"必殺", critText(a), critText(d)},
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

func forecastName(s tactics.ForecastSide, boss bool) string {
	if boss {
		return s.Name + "★"
	}
	return s.Name
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
	if s.Crit && s.Count > 0 {
		return "あり"
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
		return fmt.Sprintf("HP が半分以下なら必殺（%d 倍）", s.Multiplier)
	case tactics.Snipe:
		return fmt.Sprintf("移動せずに攻撃すると命中 +%d", s.Hit)
	}
	return ""
}

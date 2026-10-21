package ui

import (
	"fmt"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 情報欄に出す文と行を作る（DES-UI-01、DES-BTL-07）。描画（game.go）から分け、Ebitengine なしで確かめられるようにする。

// Panel は情報欄に出す中身（DES-UI-01）。
type Panel struct {
	// Forecast は攻撃の予測の表（FR-BTL-08、UC-03）。攻撃の相手を選んでいるときだけ入る。
	Forecast []ForecastRow
	// Lines は、カーソルのあるユニットの情報と、カーソルのあるマスの地形（FR-UNT-09、UC-04）。
	Lines []string
}

// ForecastRow は予測の表の 1 行（DES-BTL-07）。項目の名前、攻撃する側の値、守る側の値を並べる。
type ForecastRow struct {
	Label, Attacker, Defender string
}

// Panel は、カーソルがマス cursor にあるときの情報欄の中身を返す（DES-UI-01）。onMap が偽ならカーソルは盤面の外。
//
// 攻撃の相手を選んでいる（Targeting で相手が決まっている）ときは予測を入れる。
// カーソルのあるマスに生きているユニットがいればその情報、続けて地形を入れる。ユニットは盤面に描くもの（Units）から引くので、
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
			p.Lines = append(UnitLines(u), "")
			break
		}
	}
	p.Lines = append(p.Lines, TerrainLines(c.battle.TerrainAt(cursor))...)
	return p
}

// UnitLines は、ユニットの名前（ボスなら印）・クラス・HP・能力・武器・手持ち・スキルの行を返す（FR-UNT-09、DES-UI-01）。
func UnitLines(u tactics.Unit) []string {
	name := u.Name
	if u.Boss {
		// ボスは名前ではボスと分からない（例「ソルジャー 3」。REV-010 F-03）ので、盤面と同じ ★ の印を出す。
		name += "　★ボス"
	}
	s := u.Stats
	w := u.Weapon
	lines := []string{
		name,
		"クラス: " + u.ClassName,
		fmt.Sprintf("HP %d/%d", u.HP, s.HP),
		fmt.Sprintf("力 %d　技 %d　速さ %d　守備 %d　移動 %d", s.Str, s.Skl, s.Spd, s.Def, s.Mov),
		fmt.Sprintf("武器: %s（%s）", w.Name, weaponTypeName(w.Type)),
		fmt.Sprintf("威力 %d　命中 %d　射程 %s", w.Might, w.Hit, rangeText(w)),
		"手持ち: " + itemsText(u.Items),
	}
	if u.Skill.Effect == tactics.NoSkill {
		return append(lines, "スキル: なし")
	}
	return append(lines, "スキル: "+u.Skill.Name, "　"+skillDescription(u.Skill))
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

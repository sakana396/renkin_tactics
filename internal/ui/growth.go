package ui

import (
	"fmt"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// GrowthView は経験のポップアップまたはレベルアップの帯の描画内容（DES-UI-16）。
// Band が真なら盤面中央の帯、偽なら Pos のマスの上に表示する。
type GrowthView struct {
	Text    string
	Pos     tactics.Pos
	Band    bool
	Visible bool
}

// GrowthEventView は出来事から経験・レベルアップの表示を作る（DES-UI-16）。
func GrowthEventView(e tactics.Event) GrowthView {
	if e.Kind != tactics.EventExp && e.Kind != tactics.EventLevelUp {
		return GrowthView{}
	}
	if e.Unit < 0 || e.Unit >= len(e.Units) {
		return GrowthView{}
	}
	if e.Kind == tactics.EventExp {
		return GrowthView{Text: fmt.Sprintf("+%d 経験", e.Amount), Pos: e.Units[e.Unit].Pos, Visible: true}
	}
	return GrowthView{Text: EventText(e), Band: true, Visible: true}
}

// growthText は伸びた能力だけをHP・力・技・速さ・守備の順に並べる（DES-UI-16）。
func growthText(g tactics.Stats) string {
	var parts []string
	for _, s := range []struct {
		name  string
		value int
	}{
		{"HP", g.HP}, {"力", g.Str}, {"技", g.Skl}, {"速さ", g.Spd}, {"守備", g.Def},
	} {
		if s.value > 0 {
			parts = append(parts, fmt.Sprintf("%s+%d", s.name, s.value))
		}
	}
	if len(parts) == 0 {
		return "上がった能力なし"
	}
	return strings.Join(parts, " ")
}

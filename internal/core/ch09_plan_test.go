package core_test

import (
	"math"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// ch09 の想定する攻略（制作記録 docs/iterations/it05/map-ch09.md、DES-MAP-06）。
var (
	ch09West = []tactics.Pos{{X: 8, Y: 25}, {X: 6, Y: 21}, {X: 3, Y: 17}, {X: 5, Y: 13}, {X: 5, Y: 9}, {X: 12, Y: 8}, {X: 15, Y: 7}}
	ch09East = []tactics.Pos{{X: 23, Y: 25}, {X: 24, Y: 19}, {X: 24, Y: 15}, {X: 26, Y: 12}, {X: 26, Y: 9}, {X: 19, Y: 8}, {X: 16, Y: 7}}
)

// ch09Plans は ch09 の作戦。split は二手に分かれて前庭で合流する。west・east は全員で片方の道を行く。
func ch09Plans() map[string]autoPlan {
	order := []string{"gald", "mira", "bran", "elsa", "leon", "rina"}
	return map[string]autoPlan{
		"split": {
			Groups: map[string]int{"leon": 0, "mira": 0, "elsa": 0, "gald": 1, "bran": 1, "rina": 1},
			Routes: [][]tactics.Pos{ch09West, ch09East},
			Order:  order,
		},
		"west": {Routes: [][]tactics.Pos{ch09West}, Order: order},
		"east": {Routes: [][]tactics.Pos{ch09East}, Order: order},
	}
}

// ch09Roster は、1 章から通して遊んだときの想定の名簿で ch09 から始めるキャンペーンを作る（制作記録 2 章）。
// 平均は、ch01〜ch03 の経験の合計（約 1,500）を 6 人で分けた見込みで、レオン・ガルド・ブラン・ミラ・エルザは 3、
// リナは 2 レベル上がり、各能力は成長率 × 上がった数（四捨五入）だけ伸びる。低めは、上がるレベルを 1 少なくし、
// 伸びを成長率の 6 割にする。
func ch09Roster(t *testing.T, cat *defs.Catalog, low bool) *core.Campaign {
	t.Helper()
	c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
	if err != nil {
		t.Fatal(err)
	}
	gain := map[string]int{"leon": 3, "gald": 3, "bran": 3, "mira": 3, "elsa": 3, "rina": 2}
	for k := range c.Roster {
		m := &c.Roster[k]
		ch, _ := cat.Character(m.Character)
		n := gain[m.Character]
		rate := 1.0
		if low {
			n--
			rate = 0.6
		}
		up := func(g int) int { return int(math.Round(float64(g*n) * rate / 100)) }
		m.Level += n
		m.Stats.HP += up(ch.Growth.HP)
		m.Stats.Str += up(ch.Growth.Str)
		m.Stats.Skl += up(ch.Growth.Skl)
		m.Stats.Spd += up(ch.Growth.Spd)
		m.Stats.Def += up(ch.Growth.Def)
	}
	return c
}

package core_test

import "github.com/sakana396/renkin_tactics/internal/core/tactics"

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

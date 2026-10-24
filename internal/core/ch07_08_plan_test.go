package core_test

import (
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// ch07・ch08 の想定する攻略（制作記録 docs/iterations/it05/map-ch07.md・map-ch08.md 4 章、DES-MAP-06）。

// ch07To08Plan は自動の試し遊びの作戦と、出撃させる仲間（名簿のキャラ ID の順）。
type ch07To08Plan struct {
	name    string
	members []string
	plan    autoPlan
}

// ch07To08Plans は方面・出撃者・行動順を変えた作戦（DES-MAP-06）。5 枠なので、どちらの章も 1 人を外す。
func ch07To08Plans(id string) []ch07To08Plan {
	order := []string{"gald", "mira", "bran", "elsa", "leon", "rina"}
	withoutElsa := []string{"leon", "gald", "bran", "mira", "rina"}
	switch id {
	case "ch07":
		return []ch07To08Plan{
			// 谷の近道: 岩棚の弓兵の射程を受けながら、谷をふさぐ剣士を倒して陣へ出る。
			{"谷の近道", withoutElsa, autoPlan{Routes: [][]tactics.Pos{{{X: 7, Y: 11}, {X: 7, Y: 8}, {X: 7, Y: 4}, {X: 9, Y: 2}}}, Order: order}},
			// 東の迂回: 遠回りの街道と砦を通り、陣の東から入る。
			{"東の迂回", withoutElsa, autoPlan{Routes: [][]tactics.Pos{{{X: 13, Y: 12}, {X: 14, Y: 8}, {X: 13, Y: 4}, {X: 11, Y: 3}}}, Order: []string{"gald", "bran", "mira", "leon", "rina"}}},
		}
	default:
		withElsa := []string{"leon", "gald", "bran", "mira", "elsa"}
		center := []tactics.Pos{{X: 11, Y: 17}, {X: 11, Y: 11}, {X: 11, Y: 6}, {X: 12, Y: 2}}
		return []ch07To08Plan{
			// 森の道・回復役: リナを連れ、中央の道で突っ込んでくる敵を受けてから北の陣へ上る（エルザを外す）。
			{"森の道・回復役", withoutElsa, autoPlan{Routes: [][]tactics.Pos{center}, Order: order}},
			// 森の道・騎兵: 回復役を外して騎兵を出し、倒す手を増やして同じ道を押す。
			{"森の道・騎兵", withElsa, autoPlan{Routes: [][]tactics.Pos{center}, Order: order}},
			// 東の砦: 騎兵を出し、東の空き地の砦を奪って回復の足場にしてから北へ回る。
			{"東の砦", withElsa, autoPlan{Routes: [][]tactics.Pos{{{X: 16, Y: 17}, {X: 18, Y: 12}, {X: 18, Y: 8}, {X: 14, Y: 3}}}, Order: order}},
		}
	}
}

// autoBattleWith は、名簿から members（キャラ ID）の仲間を、生きている者だけ並びの順に出撃させる。
func autoBattleWith(c *core.Campaign, seed uint64, members []string) (*core.Sortie, []string, error) {
	var idx []int
	for _, id := range members {
		for k, m := range c.Roster {
			if m.Character == id && m.Alive {
				idx = append(idx, k)
			}
		}
	}
	s, _, err := c.StartBattle(idx, seed)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, len(s.Battle.Units()))
	for k, m := range idx {
		ids[k] = c.Roster[m].Character
	}
	return s, ids, nil
}

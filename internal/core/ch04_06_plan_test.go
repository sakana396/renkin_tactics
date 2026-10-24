package core_test

import (
	"math"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// estimatedRoster は通しの経験を見積もった平均・低めの名簿で、章 id から始めるキャンペーンを作る（DES-MAP-06、各制作記録 3 章）。
// ch03 の後（ch04 の開始）は、ch01〜ch03 の経験（約 1,500）を 6 人で分け、リナ以外は 3、リナは 2 レベル上がる見込み。
// ch04 から先は、章ごとに平均 1 レベル分を加える（ch09 は ch04〜ch08 の 5 章分。E-52）。
// 低めは 1 レベル少なく、伸びを成長率の 6 割にする。移動力と手持ちは変えない。
func estimatedRoster(t *testing.T, cat *defs.Catalog, id string, low bool) *core.Campaign {
	t.Helper()
	return estimatedRosterWithRecruits(t, cat, id, low, false)
}

// estimatedRosterWithRecruits は、新加入者だけが控えで成長していない場合も作る（DES-CMP-10）。
func estimatedRosterWithRecruits(t *testing.T, cat *defs.Catalog, id string, low, bench bool) *core.Campaign {
	t.Helper()
	c, err := core.NewCampaignAt(cat, core.Withdraw, id)
	if err != nil {
		t.Fatal(err)
	}
	extra, ok := map[string]int{"ch04": 0, "ch05": 1, "ch06": 2, "ch07": 3, "ch08": 4, "ch09": 5}[id]
	if !ok {
		t.Fatalf("見積もりのない章 %s", id)
	}
	for k := range c.Roster {
		m := &c.Roster[k]
		ch, _ := cat.Character(m.Character)
		n := 3 + extra
		if m.Character == "rina" {
			n--
		}
		// 新しい仲間は加入前の章の成長を数えない（DES-CMP-10）。
		switch m.Character {
		case "tio":
			n = extra
		case "hanna":
			n = max(0, extra-4)
		}
		rate := 1.0
		if low {
			n = max(0, n-1)
			rate = 0.6
		}
		if bench && (m.Character == "tio" || m.Character == "hanna") {
			n = 0
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

// ch04To06Plans は方面・地形・行動順を変えた作戦（DES-MAP-06）。
func ch04To06Plans(id string) []autoPlan {
	order := []string{"gald", "mira", "bran", "elsa", "leon", "rina"}
	switch id {
	case "ch04":
		west := []tactics.Pos{{X: 4, Y: 12}, {X: 4, Y: 7}, {X: 5, Y: 4}}
		east := []tactics.Pos{{X: 13, Y: 12}, {X: 13, Y: 7}, {X: 12, Y: 4}}
		return []autoPlan{
			{Groups: map[string]int{"leon": 0, "mira": 0, "elsa": 0, "gald": 1, "bran": 1, "rina": 1}, Routes: [][]tactics.Pos{west, east}, Order: order},
			{Routes: [][]tactics.Pos{east}, Order: order},
		}
	case "ch05":
		return []autoPlan{
			{Routes: [][]tactics.Pos{{{X: 7, Y: 9}, {X: 7, Y: 6}, {X: 8, Y: 3}}}, Order: order},
			{Routes: [][]tactics.Pos{{{X: 2, Y: 10}, {X: 2, Y: 6}, {X: 4, Y: 3}}}, Order: []string{"mira", "gald", "leon", "bran", "rina", "elsa"}},
		}
	default:
		return []autoPlan{
			{Routes: [][]tactics.Pos{{{X: 6, Y: 14}, {X: 6, Y: 10}, {X: 7, Y: 6}, {X: 10, Y: 3}}}, Order: order},
			{Routes: [][]tactics.Pos{{{X: 13, Y: 14}, {X: 13, Y: 10}, {X: 13, Y: 6}, {X: 10, Y: 3}}}, Order: []string{"elsa", "gald", "mira", "bran", "leon", "rina"}},
		}
	}
}

// ch05Lure は前衛を見せ、次の番に砦へ下がって関所の敵を南へ釣る別解の入口（DES-MAP-06）。
// その後は森の通路へ寄せる作戦で続ける。全て公開の行動を使い、敵・乱数は変えない。
func ch05Lure(t *testing.T, b *tactics.Battle) {
	t.Helper()
	for _, cmd := range []tactics.Command{
		{Unit: 0, To: tactics.Pos{X: 6, Y: 14}, Action: tactics.Wait},
		{Unit: 2, To: tactics.Pos{X: 5, Y: 13}, Action: tactics.Wait},
		{Unit: 3, To: tactics.Pos{X: 8, Y: 14}, Action: tactics.Wait},
		{Unit: 4, To: tactics.Pos{X: 7, Y: 15}, Action: tactics.Wait},
		{Unit: 1, To: tactics.Pos{X: 7, Y: 10}, Action: tactics.Wait},
		{Unit: 1, To: tactics.Pos{X: 6, Y: 10}, Action: tactics.Wait},
	} {
		if _, err := b.Act(cmd); err != nil {
			t.Fatalf("釣って下がる T%d %v: %v", b.Turn, cmd, err)
		}
	}
	if b.Turn != 2 || b.Units()[1].Pos != (tactics.Pos{X: 6, Y: 10}) {
		t.Fatal("砦へ下がれていない")
	}
	// 初期 (7,6) のアーマーが関所の南へ出たことを確認する。
	if u := b.Units()[7]; u.Pos.Y < 8 {
		t.Fatalf("アーマーを釣れていない: %v", u.Pos)
	}
}

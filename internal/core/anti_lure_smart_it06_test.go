package core_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// antiLureStart は、TC-FR-MAP-04-24 と同じ名簿・出撃で章 id の戦闘を seed で始める（#158）。
// 返す ids は Units の添字ごとの自軍のキャラ ID、chosen は出撃の並び。
func antiLureStart(t *testing.T, cat *defs.Catalog, id string, roster antiLureRoster, seed uint64) (*tactics.Battle, []string, []string) {
	t.Helper()
	chosen := append([]string(nil), antiLureChosen[id]...)
	if roster.lostMira {
		sub := "tio"
		if containsWeapon(chosen, "tio") {
			sub = "rina"
		}
		for i := range chosen {
			if chosen[i] == "mira" {
				chosen[i] = sub
			}
		}
	}
	var c *core.Campaign
	if id == "ch01" {
		var err error
		c, err = core.NewCampaignAt(cat, core.Withdraw, id)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		c = estimatedRosterIT06(t, cat, id, roster.low)
	}
	if roster.lostMira {
		for i := range c.Roster {
			if c.Roster[i].Character == "mira" {
				c.Roster[i].Alive = false
			}
		}
	}
	var members []int
	for _, want := range chosen {
		for i, member := range c.Roster {
			if member.Character == want {
				members = append(members, i)
				break
			}
		}
	}
	if len(members) != len(chosen) {
		t.Fatalf("名簿に %v がそろわない", chosen)
	}
	s, _, err := c.StartBattle(members, seed)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(members))
	for i, n := range members {
		ids[i] = c.Roster[n].Character
	}
	return s.Battle, ids, chosen
}

// antiLurePlan は TC-FR-MAP-04-24 と同じ作戦（章・名簿ごとの主な進路・固定の行動順）。
func antiLurePlan(id string, roster antiLureRoster) autoPlan {
	path, order := antiLureRoute(id, roster)
	return autoPlan{Order: order, Routes: [][]tactics.Pos{path}}
}

// smartCase は、賢い打ち手で決まった seed を遊ぶケース。
type smartCase struct {
	id     string
	roster antiLureRoster
	seed   uint64
}

// 賢い打ち手は、平均の名簿で 1〜9 章のどれにも勝つ固定 seed がある。代役（ミラを失いティオを出す）の名簿も、
// 20 seed で勝てた章（ch04・ch05・ch06・ch08・ch09）は勝つ（TC-FR-MAP-04-25。#158、DES-MAP-08、DES-IT06 13 章）。
// ch03 の平均は E-111 で参考の扱いだが、勝つ seed が残ったので回帰として遊ぶ。
// seed は TestMeasureSmartAutoplay の測定で勝った最初の seed。重さのため 1 章 1 seed だけを遊ぶ。
func TestIT06SmartAutoplayWins(t *testing.T) {
	cat := baseCatalog(t)
	avg, lost := antiLureRosters[0], antiLureRosters[2]
	cases := []smartCase{
		{"ch01", avg, 2}, {"ch02", avg, 5}, {"ch03", avg, 11}, {"ch04", avg, 1}, {"ch05", avg, 2},
		{"ch06", avg, 1}, {"ch07", avg, 2}, {"ch08", avg, 1}, {"ch09", avg, 1},
		{"ch04", lost, 1}, {"ch05", lost, 1}, {"ch06", lost, 1}, {"ch08", lost, 2}, {"ch09", lost, 10},
	}
	for _, tc := range cases {
		name := tc.roster.name
		if tc.id == "ch01" {
			name = "初期"
		}
		t.Run(fmt.Sprintf("TC-FR-MAP-04-25 %s %s seed %d", tc.id, name, tc.seed), func(t *testing.T) {
			t.Parallel()
			b, ids, _ := antiLureStart(t, cat, tc.id, tc.roster, tc.seed)
			r := smartPlay(b, ids, antiLurePlan(tc.id, tc.roster), smartAll, 50)
			t.Logf("%v 巻き戻し %d 回", r, r.Rewinds)
			if r.Result != tactics.Won {
				t.Fatalf("賢い打ち手が勝てない: %v", r)
			}
			if r.Rewinds > 10 {
				t.Fatalf("巻き戻しが 10 回を超えた: %d", r.Rewinds)
			}
		})
	}
}

// 低めの名簿でも、賢い打ち手は 20 seed で勝てた章（ch04・ch06）の固定 seed に勝つ（TC-FR-MAP-04-26。#158）。
// 勝てなかった章（ch02・ch03・ch05・ch07・ch08・ch09）の負け方は制作記録の「#158 賢い打ち手」の節に書く。
func TestIT06SmartAutoplayLowWins(t *testing.T) {
	cat := baseCatalog(t)
	low := antiLureRosters[1]
	for _, tc := range []smartCase{{"ch04", low, 2}, {"ch06", low, 18}} {
		t.Run(fmt.Sprintf("TC-FR-MAP-04-26 %s 低め seed %d", tc.id, tc.seed), func(t *testing.T) {
			t.Parallel()
			b, ids, _ := antiLureStart(t, cat, tc.id, tc.roster, tc.seed)
			r := smartPlay(b, ids, antiLurePlan(tc.id, tc.roster), smartAll, 50)
			t.Logf("%v 巻き戻し %d 回", r, r.Rewinds)
			if r.Result != tactics.Won {
				t.Fatalf("賢い打ち手が低めの名簿で勝てない: %v", r)
			}
		})
	}
}

// 賢い打ち手の工夫は、それぞれ勝ち負けを分ける（TC-FR-MAP-04-27。#158）。
// 工夫ごとに、すべての工夫を使うと勝ち、その工夫だけを外すと負け、今の打ち手も負ける固定 seed を 1 つずつ遊ぶ。
// 同じ seed をもう一度遊ぶと同じ結果になる（乱数は戦闘の seed だけ）ことも確かめる。
func TestIT06SmartAutoplayFeatures(t *testing.T) {
	cat := baseCatalog(t)
	avg, low, lost := antiLureRosters[0], antiLureRosters[1], antiLureRosters[2]
	for _, tc := range []struct {
		feature string
		smartCase
	}{
		{"Weapons", smartCase{"ch08", lost, 5}},
		{"Threat", smartCase{"ch06", avg, 1}},
		{"Focus", smartCase{"ch04", avg, 1}},
		{"Heal", smartCase{"ch09", avg, 1}},
		{"Hold", smartCase{"ch05", lost, 7}},
		{"Rewind", smartCase{"ch04", low, 2}},
	} {
		t.Run(fmt.Sprintf("TC-FR-MAP-04-27 %s %s %s seed %d", tc.feature, tc.id, tc.roster.name, tc.seed), func(t *testing.T) {
			t.Parallel()
			play := func(f smartFeatures) autoResult {
				b, ids, _ := antiLureStart(t, cat, tc.id, tc.roster, tc.seed)
				return smartPlay(b, ids, antiLurePlan(tc.id, tc.roster), f, 50)
			}
			all := play(smartAll)
			if all.Result != tactics.Won {
				t.Fatalf("すべての工夫で勝てない: %v", all)
			}
			if again := play(smartAll); again.String() != all.String() || again.Rewinds != all.Rewinds {
				t.Fatalf("同じ seed で結果が違う: %v / %v", all, again)
			}
			off := smartAll
			if !off.off(tc.feature) {
				t.Fatalf("工夫 %s がない", tc.feature)
			}
			if r := play(off); r.Result == tactics.Won {
				t.Fatalf("%s を外しても勝つ: %v", tc.feature, r)
			}
			b, ids, _ := antiLureStart(t, cat, tc.id, tc.roster, tc.seed)
			if r := autoPlay(b, ids, antiLurePlan(tc.id, tc.roster), 50); r.Result == tactics.Won {
				t.Fatalf("今の打ち手でも勝つ: %v", r)
			}
		})
	}
}

// 今の打ち手と賢い打ち手の勝ち数を、平均・低め・代役の名簿 × 1〜9 章 × seed 1〜20 で測る（#158 の制作記録の表）。
// 時間がかかるので RENKIN_MEASURE を設定したときだけ動く。RENKIN_SMART_OFF に工夫の名前（Weapons など）を
// カンマで並べると、その工夫を外した賢い打ち手も測る（ミュータント検査）。
//
//	RENKIN_MEASURE=1 go test ./internal/core -run TestMeasureSmartAutoplay -v -timeout 60m
func TestMeasureSmartAutoplay(t *testing.T) {
	if os.Getenv("RENKIN_MEASURE") == "" {
		t.Skip("RENKIN_MEASURE を設定したときだけ測る")
	}
	cat := baseCatalog(t)
	players := []struct {
		name string
		play func(b *tactics.Battle, ids []string, plan autoPlan) autoResult
	}{
		{"今", func(b *tactics.Battle, ids []string, plan autoPlan) autoResult { return autoPlay(b, ids, plan, 50) }},
		{"賢", func(b *tactics.Battle, ids []string, plan autoPlan) autoResult {
			return smartPlay(b, ids, plan, smartAll, 50)
		}},
	}
	if os.Getenv("RENKIN_MEASURE_SMART_ONLY") != "" {
		players = players[1:]
	}
	for _, name := range strings.Split(os.Getenv("RENKIN_SMART_OFF"), ",") {
		if name == "" {
			continue
		}
		f := smartAll
		if !f.off(name) {
			t.Fatalf("工夫 %s がない", name)
		}
		players = append(players, struct {
			name string
			play func(b *tactics.Battle, ids []string, plan autoPlan) autoResult
		}{"賢-" + name, func(b *tactics.Battle, ids []string, plan autoPlan) autoResult { return smartPlay(b, ids, plan, f, 50) }})
	}
	chapters := os.Getenv("RENKIN_MEASURE_CH")
	var mu sync.Mutex
	table := map[string]string{}
	for ch := 1; ch <= 9; ch++ {
		id := fmt.Sprintf("ch%02d", ch)
		if chapters != "" && !strings.Contains(chapters, id) {
			continue
		}
		for _, roster := range antiLureRosters {
			if ch == 1 && roster.name != "平均" || ch < 4 && roster.lostMira {
				continue
			}
			for _, pl := range players {
				t.Run(id+" "+roster.name+" "+pl.name, func(t *testing.T) {
					t.Parallel()
					var wins []int
					var losses []string
					for seed := uint64(1); seed <= 20; seed++ {
						b, ids, _ := antiLureStart(t, cat, id, roster, seed)
						r := pl.play(b, ids, antiLurePlan(id, roster))
						if r.Result == tactics.Won {
							wins = append(wins, int(seed))
						} else if len(losses) < 20 {
							losses = append(losses, fmt.Sprintf("seed%d %v T%d 倒れた%v %v", seed, r.Result, r.Turns, r.Fallen, r.Deaths))
						}
					}
					mu.Lock()
					table[id+" "+roster.name+" "+pl.name] = fmt.Sprintf("%d %v", len(wins), wins)
					mu.Unlock()
					t.Logf("%s %s %s: 20 seed 中 %d 勝 %v", id, roster.name, pl.name, len(wins), wins)
					if roster.low || os.Getenv("RENKIN_MEASURE_LOSSES") != "" {
						for _, l := range losses {
							t.Logf("  負け: %s", l)
						}
					}
				})
			}
		}
	}
	t.Cleanup(func() {
		for ch := 1; ch <= 9; ch++ {
			id := fmt.Sprintf("ch%02d", ch)
			for _, roster := range antiLureRosters {
				line := id + " " + roster.name
				any := false
				for _, pl := range players {
					if v, ok := table[id+" "+roster.name+" "+pl.name]; ok {
						line += " | " + pl.name + " " + v
						any = true
					}
				}
				if any {
					t.Log(line)
				}
			}
		}
	})
}

// off は名前の工夫を外す。名前がなければ偽を返す。
func (f *smartFeatures) off(name string) bool {
	switch name {
	case "Weapons":
		f.Weapons = false
	case "Threat":
		f.Threat = false
	case "Focus":
		f.Focus = false
	case "Heal":
		f.Heal = false
	case "Hold":
		f.Hold = false
	case "Rewind":
		f.Rewind = false
	default:
		return false
	}
	return true
}

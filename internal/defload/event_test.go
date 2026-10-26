package defload_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 出来事の定義（DES-MAP-07）と、初めの敵 39 体の上限（DES-IT06 9.3 節）の読み込みの試験。

// eventRows は 8 × 8 の平地の行。
var eventRows = strings.Repeat(`"........",`, 8)

// enemyAt は (x, y) に置く敵の TOML を作る。extra は足すキー。
func enemyAt(table string, x, y int, extra string) string {
	return fmt.Sprintf(`
[[%s]]
class = "soldier"
weapon = "iron_lance"
ai = "hold"
x = %d
y = %d
%s`, table, x, y, extra)
}

func TestLoadEnemyCap(t *testing.T) {
	t.Run("TC-FR-MAP-03-16 初めの敵 39 体は受理し 40 体は拒む", func(t *testing.T) {
		for _, n := range []int{39, 40} {
			t.Run(fmt.Sprintf("敵 %d 体", n), func(t *testing.T) {
				extra := smallBoss
				for i := 0; i < n-1; i++ {
					extra += enemyAt("maps.enemies", i%8, 1+i/8, "")
				}
				m := baseFS(t)
				smallMap(eventRows, extra)(t, m)
				cat, err := defload.Load(m)
				if n == 39 {
					if err != nil {
						t.Fatal(err)
					}
					got, _ := cat.Map("small")
					if len(got.Enemies) != 39 {
						t.Fatalf("敵 %d 体", len(got.Enemies))
					}
					return
				}
				want := appended(t, "maps.toml: maps[N].enemies: 初めの敵が 40 体います（39 体まで）")
				if err == nil || !strings.Contains(err.Error(), want) || cat != nil {
					t.Fatalf("40 体を拒まない: %v", err)
				}
			})
		}
	})
}

func TestLoadMapEvents(t *testing.T) {
	// guard は id と一団を書いた敵。出来事 gate_alarm は攻撃されると増援を、advance は範囲に着くと突撃を起こす。
	valid := smallBoss + enemyAt("maps.enemies", 3, 3, `id = "guard"
group = "gate_group"`) + `
[[maps.reinforcements]]
turn = 2
class = "soldier"
weapon = "iron_lance"
ai = "charge"
x = 7
y = 7

[[maps.events]]
id = "gate_alarm"
trigger = "enemy_attacked"
unit = "guard"
effect = "reinforce"
` + enemyAt("maps.events.enemies", 0, 4, `id = "reserve_1"
group = "late"`) + enemyAt("maps.events.enemies", 0, 5, "") + `
[[maps.events]]
id = "advance"
trigger = "player_entered"
area = { min_x = 3, min_y = 4, max_x = 7, max_y = 7 }
effect = "charge_group"
group = "gate_group"
`
	t.Run("TC-FR-MAP-03-20 出来事を読み込む", func(t *testing.T) {
		m := baseFS(t)
		smallMap(eventRows, valid)(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := cat.Map("small")
		if len(got.Events) != 2 {
			t.Fatalf("出来事 %+v", got.Events)
		}
		e0, e1 := got.Events[0], got.Events[1]
		assertEvent := func(name string, ok bool) {
			t.Helper()
			if !ok {
				t.Errorf("%s: %+v / %+v", name, e0, e1)
			}
		}
		assertEvent("増援", e0.ID == "gate_alarm" && e0.Trigger == defs.TriggerEnemyAttacked && e0.Unit == "guard" && e0.Effect == defs.EventReinforce && len(e0.Enemies) == 2)
		assertEvent("増援の敵", e0.Enemies[0].ID == "reserve_1" && e0.Enemies[0].Group == "late" && e0.Enemies[0].X == 0 && e0.Enemies[0].Y == 4 && e0.Enemies[1].ID == "")
		assertEvent("突撃", e1.ID == "advance" && e1.Trigger == defs.TriggerPlayerEntered && e1.Area == (defs.Area{MinX: 3, MinY: 4, MaxX: 7, MaxY: 7}) && e1.Effect == defs.EventChargeGroup && e1.Group == "gate_group")
		if got.Enemies[1].ID != "guard" || got.Enemies[1].Group != "gate_group" || got.Enemies[0].ID != "" {
			t.Errorf("敵の ID と一団: %+v", got.Enemies)
		}
	})

	t.Run("TC-FR-MAP-03-20 出来事の定義の誤りを拒む", func(t *testing.T) {
		cases := []struct{ name, old, new, want string }{
			{"出来事の ID の重なり", `id = "advance"`, `id = "gate_alarm"`, `maps[N].events[1].id: ID "gate_alarm" が重なっています`},
			{"出来事の ID が空", `id = "advance"`, `id = ""`, "maps[N].events[1].id: ID が空です"},
			{"敵の ID の重なり", `id = "reserve_1"`, `id = "guard"`, `maps[N].events[0].enemies[0].id: 敵の ID "guard" が重なっています`},
			{"作った ID との重なり", `id = "reserve_1"`, `id = "enemies[0]"`, `maps[N].events[0].enemies[0].id: 敵の ID "enemies[0]" が重なっています`},
			{"敵の ID が空", `id = "reserve_1"`, `id = ""`, "maps[N].events[0].enemies[0].id: ID が空です"},
			{"参照先がない", `unit = "guard"`, `unit = "nobody"`, `maps[N].events[0].unit: 敵 "nobody" がいません`},
			{"ID を書いていない敵の参照", `unit = "guard"`, `unit = "enemies[0]"`, `maps[N].events[0].unit: 敵 "enemies[0]" がいません`},
			{"敵のきっかけに unit がない", `unit = "guard"`, ``, "maps[N].events[0].unit: enemy_attacked には unit が要ります"},
			{"敵のきっかけに area", `unit = "guard"`, `unit = "guard"
area = { min_x = 0, min_y = 0, max_x = 1, max_y = 1 }`, "maps[N].events[0].area: enemy_attacked に area は書けません"},
			{"範囲のきっかけに unit", `trigger = "player_entered"`, `trigger = "player_entered"
unit = "guard"`, "maps[N].events[1].unit: player_entered に unit は書けません"},
			{"範囲のきっかけに area がない", `area = { min_x = 3, min_y = 4, max_x = 7, max_y = 7 }`, ``, "maps[N].events[1].area: player_entered には area が要ります"},
			{"範囲の向きが逆", `min_x = 3, min_y = 4, max_x = 7`, `min_x = 3, min_y = 4, max_x = 2`, "maps[N].events[1].area: 範囲 (3, 4)〜(2, 7) の向きが逆です"},
			{"範囲が盤の外", `max_x = 7, max_y = 7`, `max_x = 8, max_y = 7`, "maps[N].events[1].area: 範囲 (3, 4)〜(8, 7) が盤の外です"},
			{"知らないきっかけ", `trigger = "enemy_attacked"`, `trigger = "enemy_waited"`, `maps[N].events[0].trigger: きっかけ "enemy_waited" は使えません`},
			{"知らない効果", `effect = "reinforce"`, `effect = "retreat"`, `maps[N].events[0].effect: 効果 "retreat" は使えません`},
			{"増援が 0 体", `effect = "charge_group"
group = "gate_group"`, `effect = "reinforce"`, "maps[N].events[1].enemies: reinforce には増援の敵が 1 体以上要ります"},
			{"増援に group", `effect = "reinforce"`, `effect = "reinforce"
group = "gate_group"`, "maps[N].events[0].group: reinforce に group は書けません"},
			{"突撃に増援", `effect = "reinforce"`, `effect = "charge_group"
group = "gate_group"`, "maps[N].events[0].enemies: charge_group に増援の敵は書けません"},
			{"突撃の一団がいない", `effect = "charge_group"
group = "gate_group"`, `effect = "charge_group"
group = "nobody"`, `maps[N].events[1].group: 一団 "nobody" の敵がいません`},
			{"ボスの増援", `id = "reserve_1"`, `id = "reserve_1"
boss = true`, "maps[N].events[0].enemies[0].boss: 増援はボスになれません"},
			{"杖を持つ増援", `weapon = "iron_lance"
ai = "hold"
x = 0
y = 4`, `weapon = "heal_staff"
ai = "hold"
x = 0
y = 4`, `maps[N].events[0].enemies[0].weapon: 敵は杖 "heal_staff" を持てません`},
			{"正しくない能力", `id = "reserve_1"`, `id = "reserve_1"
bonus = { hp = -100 }`, "maps[N].events[0].enemies[0]"},
			{"増援が盤の外", `x = 0
y = 4`, `x = 8
y = 4`, "maps[N].events[0].enemies[0]: 座標 (8, 4) がマップの外です"},
			{"知らないキー", `effect = "reinforce"`, `effect = "reinforce"
delay = 2`, "maps[N].events[0].delay: 知らないキーです"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if !strings.Contains(valid, tc.old) {
					t.Fatalf("置換元 %q がない", tc.old)
				}
				m := baseFS(t)
				smallMap(eventRows, strings.Replace(valid, tc.old, tc.new, 1))(t, m)
				cat, err := defload.Load(m)
				if err == nil || cat != nil || !strings.Contains(err.Error(), appended(t, tc.want)) {
					t.Fatalf("期待=%s 誤り=%v", tc.want, err)
				}
			})
		}
	})
}

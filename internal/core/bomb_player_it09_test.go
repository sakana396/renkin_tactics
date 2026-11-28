package core_test

import (
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"testing"
)

// TestBombOption は爆弾の選択と在庫を確かめる（DES-NG-23）。
func TestBombOption(t *testing.T) {
	// 盤面: 平地 8×3、投げる主人公 (0,1) は動かず、敵 a (3,1)・敵 b (4,1) も動かない。爆弾は威力 8・半径 1・射程 3。
	// b がボスでないときは、届かない (7,1) に HP99 の動かないボスを置く（ボスはちょうど 1 体。勝ちはボスの撃破）。
	// 爆心 (3,1) か (4,1) なら a・b の両方に 8 ダメージ。HP8 の敵は倒れ、HP20 の敵は倒れない（DES-NG01）。
	// 期待値は pickBomb の規則から: 2 体以上かボスを倒す爆弾だけを使い、攻撃でボスを 9 割以上で倒せるなら使わない。
	for _, tc := range []struct {
		id                  string
		hp, hpB, count      int
		ally, strong, bossB bool
		want                bool // bombOption が候補を返すか
		wantKills           int  // 候補の撃破数
		wantUses            int  // phase で使った数
		wantWon             bool // phase のあと勝ったか
	}{
		{"TC-DES-NG-23-05 敵2体を倒す", 8, 8, 1, false, false, false, true, 2, 1, false},
		{"TC-DES-NG-23-06 味方を巻き込む", 8, 8, 1, true, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-07 撃破なし", 20, 20, 1, false, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-08 在庫なし", 8, 8, 0, false, false, false, false, 0, 0, false},
		{"TC-DES-NG-23-10 攻撃でボスを倒せる", 8, 8, 1, false, true, true, true, 2, 0, true},
		{"TC-DES-NG-23-12 1体だけ倒す", 8, 20, 1, false, false, false, true, 1, 0, false},
		{"TC-DES-NG-23-13 ボスを倒す", 20, 8, 1, false, false, true, true, 1, 1, true},
	} {
		t.Run(tc.id, func(t *testing.T) {
			setup, err := core.NewSetup(baseCatalog(t), "ch01")
			if err != nil {
				t.Fatal(err)
			}
			var player, enemy tactics.UnitSetup
			for _, u := range setup.Units {
				if u.Side == tactics.Player && len(u.Weapons) > 0 {
					player = u
				}
				if u.Side == tactics.Enemy && len(u.Weapons) > 0 {
					enemy = u
				}
			}
			setup.Terrain = make([][]tactics.Terrain, 3)
			for y := range setup.Terrain {
				setup.Terrain[y] = make([]tactics.Terrain, 8)
				for x := range setup.Terrain[y] {
					setup.Terrain[y][x] = tactics.Terrain{Name: "平地", Cost: 1}
				}
			}
			player.Pos = tactics.Pos{X: 0, Y: 1}
			player.Lord = true
			player.Stats.Mov = 0
			player.Weapons = append([]tactics.Weapon(nil), player.Weapons...)
			player.Weapons[0].Type = tactics.Sword
			player.Weapons[0].Heal, player.Weapons[0].Uses, player.Weapons[0].MaxUses = 0, 0, 0
			player.AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword}
			player.Weapons[0].MinRange = 1
			player.Weapons[0].MaxRange = 3
			player.Weapons[0].Might = 0
			player.Stats.Str = 0
			player.Stats.Mag = 0
			if tc.strong {
				player.Weapons[0].MaxRange = 4
				player.Weapons[0].Might = 50
				player.Weapons[0].Hit = 100
				player.Stats.Skl = 50
			}
			player.Items = []tactics.ItemStack{{Item: tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: 8, BlastRadius: 1, ThrowRange: 3}, Count: tc.count}}
			enemy.Pos = tactics.Pos{X: 3, Y: 1}
			enemy.Stats.HP = tc.hp
			enemy.Stats.Mov = 0
			enemy.Boss = false
			enemy.Commander = false
			enemy.ID = "enemy-a"
			second := enemy
			second.ID = "enemy-b"
			second.Pos = tactics.Pos{X: 4, Y: 1}
			second.Boss = tc.bossB
			second.Stats.HP = tc.hpB
			if tc.count == 0 {
				player.Items = nil
			}
			if tc.ally {
				player.Items[0].Item.ThrowRange = 2
			}
			setup.Units = []tactics.UnitSetup{player, enemy, second}
			if !tc.bossB {
				far := enemy
				far.ID = "enemy-boss"
				far.Pos = tactics.Pos{X: 7, Y: 1}
				far.Stats.HP = 99
				far.Boss = true
				setup.Units = append(setup.Units, far)
			}
			if tc.ally {
				ally := player
				ally.ID = "ally"
				ally.Lord = false
				ally.Pos = tactics.Pos{X: 2, Y: 1}
				setup.Units = append(setup.Units, ally)
			}
			setup.Reinforcements = nil
			setup.Events = nil
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			p := &smartPlayer{autoPlayer: &autoPlayer{b: b, ids: make([]string, len(setup.Units)), step: map[int]int{}}, f: smartFeatures{Bomb: true, Threat: true, Focus: true}}
			p.refresh()
			o := p.bombOption(0)
			if (o != nil) != tc.want {
				t.Fatalf("爆弾候補 = %v、期待 %v", o, tc.want)
			}
			if o != nil {
				f, err := b.BlastForecast(o.cmd)
				if err != nil {
					t.Fatal(err)
				}
				kills := 0
				for _, v := range f.Targets {
					if v.Side == tactics.Enemy && v.Falls {
						kills++
					}
					if v.Side == tactics.Player {
						t.Fatal("味方を巻き込む")
					}
				}
				if kills != tc.wantKills {
					t.Fatalf("撃破 = %d、期待 %d", kills, tc.wantKills)
				}
				if o.decisive != (kills >= 2 || tc.bossB) {
					t.Fatalf("decisive = %v、撃破 %d・ボス %v", o.decisive, kills, tc.bossB)
				}
			}
			p.phase()
			if p.bombUses != tc.wantUses {
				t.Fatalf("使用 = %d、期待 %d", p.bombUses, tc.wantUses)
			}
			if p.bombUses > 0 && p.bombKills != tc.wantKills {
				t.Fatalf("撃破数 = %d、期待 %d", p.bombKills, tc.wantKills)
			}
			if (b.Result == tactics.Won) != tc.wantWon {
				t.Fatalf("結果 = %v、勝ちの期待 %v", b.Result, tc.wantWon)
			}
			left := 0
			for _, stack := range b.Units()[0].Items {
				if stack.Item.Bomb {
					left += stack.Count
				}
			}
			if left != tc.count-tc.wantUses {
				t.Fatalf("在庫 = %d、期待 %d", left, tc.count-tc.wantUses)
			}
			if tc.id != "TC-DES-NG-23-05 敵2体を倒す" {
				return
			}
			t.Run("TC-DES-NG-23-11 巻き戻し", func(t *testing.T) {
				if !p.hasFirst {
					t.Fatal("行動履歴がない")
				}
				if err := b.Rewind(p.firstID); err != nil {
					t.Fatal(err)
				}
				p.restoreTurn(&smartTurn{step: map[int]int{}})
				if p.bombUses != 0 || p.bombKills != 0 {
					t.Fatal("使用・撃破数が戻らない")
				}
				found := false
				for _, stack := range b.Units()[0].Items {
					if stack.Item.Bomb && stack.Count == 1 {
						found = true
					}
				}
				if !found {
					t.Fatal("在庫が戻らない")
				}
				p.refresh()
				if o := p.bombOption(0); o == nil || !o.decisive {
					t.Fatal("巻き戻し後の爆弾候補がない")
				}
			})
		})
	}
}

// smallBoard は、ch01 の自軍・敵の 1 体ずつを型にした、平地 8×3 の小さな盤面を作る（爆弾の打ち手の試験用）。
// 自軍は主人公で動かず、剣（射程 1〜3、威力 0、力 0）を持つ。敵は動かない。
func smallBoard(t *testing.T) (setup tactics.Setup, player, enemy tactics.UnitSetup) {
	t.Helper()
	s, err := core.NewSetup(baseCatalog(t), "ch01")
	if err != nil {
		t.Fatal(err)
	}
	setup = s
	for _, u := range setup.Units {
		if u.Side == tactics.Player && len(u.Weapons) > 0 {
			player = u
		}
		if u.Side == tactics.Enemy && len(u.Weapons) > 0 {
			enemy = u
		}
	}
	setup.Terrain = make([][]tactics.Terrain, 3)
	for y := range setup.Terrain {
		setup.Terrain[y] = make([]tactics.Terrain, 8)
		for x := range setup.Terrain[y] {
			setup.Terrain[y][x] = tactics.Terrain{Name: "平地", Cost: 1}
		}
	}
	player.Pos = tactics.Pos{X: 0, Y: 1}
	player.Lord = true
	player.Stats.Mov = 0
	player.Weapons = append([]tactics.Weapon(nil), player.Weapons...)
	player.Weapons[0].Type = tactics.Sword
	player.Weapons[0].Heal, player.Weapons[0].Uses, player.Weapons[0].MaxUses = 0, 0, 0
	player.AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword}
	player.Weapons[0].MinRange, player.Weapons[0].MaxRange, player.Weapons[0].Might = 1, 3, 0
	player.Stats.Str, player.Stats.Mag = 0, 0
	enemy.Stats.Mov = 0
	enemy.Boss, enemy.Commander = false, false
	setup.Reinforcements = nil
	setup.Events = nil
	return setup, player, enemy
}

var testBomb = tactics.Item{ID: "bomb", Name: "爆弾", Bomb: true, BlastDamage: 8, BlastRadius: 1, ThrowRange: 3}

// TestBombBossIgnoresSafety は、ボスを倒す爆弾が投げる人の安全を問わないことを確かめる（DES-NG-23）。
// 主人公 (0,1) の隣 (1,1) に、主人公を確実に倒せる強い敵（HP99）を置く。主人公は動けないので、どの行動でも
// 次の敵の番に倒れる見込みが主人公の上限 1% を超え、safeFor は許さない。HP8 のボス (4,1) は威力 8 の爆弾で倒れ、
// 爆発のあとボスの撃破で勝つ（DES-NG01）。従って爆弾の候補があり、phase で使って勝つのが期待値。
// ボスでなければ（HP8 の普通の敵と離れたボス）、同じ盤面で候補はない（安全を問う）。
func TestBombBossIgnoresSafety(t *testing.T) {
	for _, tc := range []struct {
		id      string
		boss    bool
		want    bool
		wantWon bool
	}{
		{"TC-DES-NG-23-15 ボスを倒す爆弾は危険でも使う", true, true, true},
		{"TC-DES-NG-23-15 ボスでなければ危険なら使わない", false, false, false},
	} {
		t.Run(tc.id, func(t *testing.T) {
			setup, player, enemy := smallBoard(t)
			player.Items = []tactics.ItemStack{{Item: testBomb, Count: 1}}
			strong := enemy
			strong.ID = "strong"
			strong.Pos = tactics.Pos{X: 1, Y: 1}
			strong.Stats.HP, strong.Stats.Str, strong.Stats.Skl, strong.Stats.Spd = 99, 60, 60, 30
			strong.AI = tactics.Hold
			strong.Weapons = append([]tactics.Weapon(nil), strong.Weapons...)
			strong.Weapons[0].MinRange, strong.Weapons[0].MaxRange, strong.Weapons[0].Hit = 1, 1, 100
			target := enemy
			target.ID = "target"
			target.Pos = tactics.Pos{X: 4, Y: 1}
			target.Stats.HP = 8
			target.Boss = tc.boss
			setup.Units = []tactics.UnitSetup{player, strong, target}
			if !tc.boss {
				far := enemy
				far.ID = "far-boss"
				far.Pos = tactics.Pos{X: 7, Y: 1}
				far.Stats.HP = 99
				far.Boss = true
				setup.Units = append(setup.Units, far)
			}
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			p := &smartPlayer{autoPlayer: &autoPlayer{b: b, ids: make([]string, len(setup.Units)), step: map[int]int{}}, f: smartFeatures{Bomb: true, Threat: true, Focus: true}}
			p.refresh()
			u := b.Units()[0]
			if ok, death, _, _ := p.safeFor(0, u.Pos, u.EquippedWeapon(), -1, nil, u.HP); ok || death <= p.riskLimit(u) {
				t.Fatalf("前提: 主人公の今のマスは危険のはず（許す %v、倒れる %.3f）", ok, death)
			}
			o := p.bombOption(0)
			if (o != nil) != tc.want {
				t.Fatalf("爆弾候補 = %v、期待 %v", o, tc.want)
			}
			if !tc.want {
				return
			}
			p.phase()
			if p.bombUses != 1 || p.bombKills != 1 || b.Result != tactics.Won {
				t.Fatalf("使用・撃破・結果 = %d・%d・%v、期待 1・1・勝ち", p.bombUses, p.bombKills, b.Result)
			}
		})
	}
}

// TestPotionSkipsBomb は、道具ありの打ち手が回復しない道具を薬として選ばないことを確かめる（DES-NG-23）。
// 2 人目の自軍が主人公の隣で威力 8 の爆弾を投げ、主人公（最大 HP の 6 割未満になる HP）を巻き込んで減らす（DES-NG01 の固定ダメージ）。
// 主人公の持ち物が [爆弾, 薬] なら、道具ありは薬（添字 1）を、道具なし（Bomb が偽）は今の動きのまま先頭の爆弾（添字 0）を選ぶ。
// 持ち物が爆弾だけなら、道具ありは薬の候補なし、道具なしは爆弾（添字 0）。
func TestPotionSkipsBomb(t *testing.T) {
	potion := tactics.Item{ID: "potion", Name: "薬", Heal: 10}
	for _, tc := range []struct {
		id       string
		bomb     bool
		potion   bool
		wantItem int // -1 は候補なし
	}{
		{"TC-DES-NG-23-16 道具ありは爆弾を飛ばして薬", true, true, 1},
		{"TC-DES-NG-23-16 道具ありで爆弾だけなら候補なし", true, false, -1},
		{"TC-DES-NG-23-16 道具なしは先頭の爆弾（既知の不具合を保つ）", false, true, 0},
		{"TC-DES-NG-23-16 道具なしで爆弾だけでも爆弾", false, false, 0},
	} {
		t.Run(tc.id, func(t *testing.T) {
			setup, player, enemy := smallBoard(t)
			player.Stats.HP = 15
			player.Items = []tactics.ItemStack{{Item: testBomb, Count: 1}}
			if tc.potion {
				player.Items = append(player.Items, tactics.ItemStack{Item: potion, Count: 1})
			}
			ally := player
			ally.ID, ally.Name, ally.Lord = "ally", "仲間", false
			ally.Pos = tactics.Pos{X: 1, Y: 1}
			ally.Stats.Mov = 0
			ally.Items = []tactics.ItemStack{{Item: testBomb, Count: 1}}
			enemy.Pos = tactics.Pos{X: 7, Y: 1}
			enemy.Boss = true
			enemy.Stats.HP = 99
			setup.Units = []tactics.UnitSetup{player, ally, enemy}
			b, _, err := tactics.New(setup, 1)
			if err != nil {
				t.Fatal(err)
			}
			c := tactics.Pos{X: 0, Y: 1}
			if _, err := b.Act(tactics.Command{Unit: 1, To: ally.Pos, Action: tactics.UseItem, Item: 0, Blast: &c}); err != nil {
				t.Fatal(err)
			}
			u := b.Units()[0]
			if u.HP != 15-8 || u.HP*10 >= u.Stats.HP*6 {
				t.Fatalf("前提: 主人公の HP = %d/%d、期待 7/15", u.HP, u.Stats.HP)
			}
			p := &smartPlayer{autoPlayer: &autoPlayer{b: b, ids: make([]string, len(setup.Units)), step: map[int]int{}}, f: smartFeatures{Bomb: tc.bomb, Threat: true}}
			p.refresh()
			o := p.potion(0)
			got := -1
			if o != nil {
				got = o.cmd.Item
			}
			if got != tc.wantItem {
				t.Fatalf("薬の候補の持ち物 = %d、期待 %d", got, tc.wantItem)
			}
		})
	}
}

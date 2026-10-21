package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 設計書 10 章の仮置きの値（武器、地形、スキル）。
var (
	forest     = tactics.Terrain{Name: "森", Cost: 2, Avoid: 20, Def: 1}
	ironSword  = tactics.Weapon{Name: "鉄の剣", Type: tactics.Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
	ironLance  = tactics.Weapon{Name: "鉄の槍", Type: tactics.Lance, Might: 7, Hit: 80, MinRange: 1, MaxRange: 1}
	ironAxe    = tactics.Weapon{Name: "鉄の斧", Type: tactics.Axe, Might: 8, Hit: 70, MinRange: 1, MaxRange: 1}
	ironBow    = tactics.Weapon{Name: "鉄の弓", Type: tactics.Bow, Might: 6, Hit: 85, MinRange: 2, MaxRange: 2}
	inspire    = tactics.Skill{Name: "鼓舞", Effect: tactics.Inspire, Range: 2, Hit: 10, Avoid: 10}
	pavise     = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 30}
	wrath      = tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Multiplier: 3}
	snipe      = tactics.Skill{Name: "狙撃", Effect: tactics.Snipe, Hit: 20}
	leonStats  = tactics.Stats{HP: 20, Str: 6, Skl: 8, Spd: 9, Def: 4, Mov: 5}
	soldierSts = tactics.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5}
)

// example1Setup は設計書 3.5 節の計算例 1 の盤面。7 × 7 の平地で (3, 3) が森。
//
//	0: レオン (2, 3)。ロード、鉄の剣、回復薬 × 2、鼓舞
//	1: ソルジャー 3 (3, 3)（森）。ボス、鉄の槍、待ち構える
//
// レオンが今いるマスから攻撃すると、レオンは 5 ダメージ・命中率 63・2 回、ソルジャーは 9 ダメージ・命中率 85・1 回。
func example1Setup() tactics.Setup {
	terrain := field(7, 7)
	terrain[3][3] = forest
	return tactics.Setup{
		Terrain: terrain,
		Units: []tactics.UnitSetup{
			{Name: "レオン", ClassName: "ロード", Side: tactics.Player, Stats: leonStats, Pos: tactics.Pos{X: 2, Y: 3}, Weapon: ironSword, Items: []tactics.ItemStack{{Item: potion, Count: 2}}, Skill: inspire, Lord: true},
			{Name: "ソルジャー 3", ClassName: "ソルジャー", Side: tactics.Enemy, Stats: soldierSts, Pos: tactics.Pos{X: 3, Y: 3}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
		},
	}
}

// example4Setup は設計書 3.5 節の計算例 4 の盤面。7 × 7 の平地。
//
//	0: レオン (1, 1)。ガルドから距離 2（鼓舞が効く）
//	1: ガルド (2, 2)。アーマー、鉄の槍、大盾
//	2: ファイター (3, 2)。鉄の斧
//	3: ボス (6, 6)
//
// ガルドが今いるマスから攻撃すると、ガルドは 9 ダメージ・命中率 73・1 回（鼓舞、大盾 30%）、ファイターは 6・79・1 回。
func example4Setup() tactics.Setup {
	return tactics.Setup{
		Terrain: field(7, 7),
		Units: []tactics.UnitSetup{
			{Name: "レオン", ClassName: "ロード", Side: tactics.Player, Stats: leonStats, Pos: tactics.Pos{X: 1, Y: 1}, Weapon: ironSword, Skill: inspire, Lord: true},
			{Name: "ガルド", ClassName: "アーマー", Side: tactics.Player, Stats: tactics.Stats{HP: 24, Str: 6, Skl: 3, Spd: 2, Def: 10, Mov: 4}, Pos: tactics.Pos{X: 2, Y: 2}, Weapon: ironLance, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Skill: pavise},
			{Name: "ファイター", ClassName: "ファイター", Side: tactics.Enemy, Stats: tactics.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5}, Pos: tactics.Pos{X: 3, Y: 2}, Weapon: ironAxe, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Stats: soldierSts, Pos: tactics.Pos{X: 6, Y: 6}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
		},
	}
}

// wrathSetup は、怒りの敵を弱らせてから予測を見る盤面。6 × 6 の平地。命中はすべて 300 で必ず当たる。
//
//	0: 主人公 (0, 5)。移動 0
//	1: 剣兵 (2, 2)。力 9 の剣（威力 0）。斧の怒りの敵に 10 ダメージ。守備 30 で反撃は 0
//	2: 弓兵 (3, 4)。移動 0。怒りの敵から距離 2 で、隣にしか届かない斧の敵は反撃できない。狙撃
//	3: 剣士 (4, 2)。移動 0。怒りの敵の隣。守備 30
//	4: 怒りの敵 (3, 2)。HP 20、守備 0、斧、怒り、待ち構える。移動 0
//	5: ボス (5, 0)
func wrathSetup() tactics.Setup {
	sword := tactics.Weapon{Name: "剣", Type: tactics.Sword, Hit: 300, MinRange: 1, MaxRange: 1}
	bow := tactics.Weapon{Name: "弓", Type: tactics.Bow, Hit: 300, MinRange: 2, MaxRange: 2}
	axe := tactics.Weapon{Name: "斧", Type: tactics.Axe, Hit: 300, MinRange: 1, MaxRange: 1}
	return tactics.Setup{
		Terrain: field(6, 6),
		Units: []tactics.UnitSetup{
			{Name: "主人公", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 0, Y: 5}, Weapon: sword, Lord: true},
			{Name: "剣兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Str: 9, Def: 30, Mov: 1}, Pos: tactics.Pos{X: 2, Y: 2}, Weapon: sword},
			{Name: "弓兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Str: 1, Def: 30}, Pos: tactics.Pos{X: 3, Y: 4}, Weapon: bow, Skill: snipe},
			{Name: "剣士", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Str: 1, Def: 30}, Pos: tactics.Pos{X: 4, Y: 2}, Weapon: sword},
			{Name: "ブラン", Side: tactics.Enemy, Stats: tactics.Stats{HP: 20, Str: 5}, Pos: tactics.Pos{X: 3, Y: 2}, Weapon: axe, Skill: wrath, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 5, Y: 0}, Weapon: axe, Boss: true, AI: tactics.Hold},
		},
	}
}

// target は、自軍 unit を今いるマスにとどめて攻撃を選び、p の敵を相手に選んだ Targeting にする。
func target(t *testing.T, c *Controller, unit tactics.Pos, p tactics.Pos) {
	t.Helper()
	c.Click(unit)
	c.Click(unit)
	c.ChooseAction(tactics.Attack)
	c.Click(p)
	if c.State() != Targeting {
		t.Fatalf("状態 = %v、Targeting のはず", c.State())
	}
	if _, _, tg := c.Selection(); tg < 0 {
		t.Fatalf("相手を選べていない")
	}
}

// rows は、情報欄に描く文を、同じ高さのものを空白でつないで 1 行にし、上から並べる（DES-UI-06）。
func rows(v InfoView) []string {
	var out []string
	for i, tx := range v.Texts {
		if i > 0 && tx.Y == v.Texts[i-1].Y {
			out[len(out)-1] += " " + tx.Text
			continue
		}
		out = append(out, tx.Text)
	}
	return out
}

// panelRows は、情報欄の中身の文の行を返す。
func panelRows(p Panel) []string { return rows(p.View()) }

// 情報欄（DES-UI-01、DES-UI-06、FR-UNT-09、UC-04）を確かめる。
func TestInfoPanel(t *testing.T) {
	leonLines := []string{
		"レオン",
		"クラス: ロード",
		"HP 20/20",
		"力 6",
		"技 8",
		"速さ 9",
		"守備 4",
		"移動 5",
		"武器: 鉄の剣（剣）",
		"威力 5　命中 90　射程 1",
		"手持ち: 回復薬 ×2",
		"スキル: 鼓舞",
		"　距離 2 以内の味方の命中 +10・回避 +10",
		"地形: 平地",
		"回避 0　守備 0　回復 0%",
	}

	t.Run("TC-FR-UNT-09-01 カーソルのある自軍の名前・クラス・HP・能力・武器・手持ち・スキルと地形", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		p := c.Panel(tactics.Pos{X: 2, Y: 3}, true)
		if got := panelRows(p); !reflect.DeepEqual(got, leonLines) {
			t.Errorf("行 = %q\nのはず %q", got, leonLines)
		}
		// ユニットと地形の間は 1 行あける（地形の行は、スキルの効果の行の 2 行下から）。
		v := p.View()
		desc, terrain := textOf(t, v, leonLines[12]), textOf(t, v, "地形: 平地")
		if terrain.Y < desc.Y+2*lineHeight {
			t.Errorf("地形の行の y = %d、スキルの効果の行 %d + %d 以上のはず", terrain.Y, desc.Y, 2*lineHeight)
		}
		if p.Forecast != nil {
			t.Errorf("Idle で予測が出ている: %v", p.Forecast)
		}
		// 状態は変わらない（UC-04 の事後条件）。
		if c.State() != Idle || c.Battle().Units()[0].Pos != (tactics.Pos{X: 2, Y: 3}) {
			t.Errorf("状態 %v", c.State())
		}
	})

	t.Run("TC-FR-UNT-09-02 カーソルのある敵のボスの印と、いるマスの地形（森）。スキルなし", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		want := []string{
			"ソルジャー 3　★ボス",
			"クラス: ソルジャー",
			"HP 18/18",
			"力 5",
			"技 4",
			"速さ 4",
			"守備 4",
			"移動 5",
			"武器: 鉄の槍（槍）",
			"威力 7　命中 80　射程 1",
			"手持ち: なし",
			"スキル: なし",
			"地形: 森",
			"回避 20　守備 1　回復 0%",
		}
		if got := panelRows(c.Panel(tactics.Pos{X: 3, Y: 3}, true)); !reflect.DeepEqual(got, want) {
			t.Errorf("行 = %q\nのはず %q", got, want)
		}
	})

	t.Run("TC-FR-UNT-09-03 武器の種類・射程の幅、大盾・怒り・狙撃の説明、複数の手持ち、砦の回復", func(t *testing.T) {
		u := tactics.Unit{
			Name: "ミラ", ClassName: "アーチャー", Stats: tactics.Stats{HP: 17, Str: 5, Skl: 9, Spd: 7, Def: 3, Mov: 5}, HP: 9,
			Weapon: tactics.Weapon{Name: "長弓", Type: tactics.Bow, Might: 6, Hit: 85, MinRange: 2, MaxRange: 3},
			Items:  []tactics.ItemStack{{Item: potion, Count: 1}, {Item: tactics.Item{Name: "特効薬", Heal: 20}, Count: 2}},
			Skill:  snipe,
		}
		want := []string{
			"ミラ",
			"クラス: アーチャー",
			"HP 9/17",
			"力 5",
			"技 9",
			"速さ 7",
			"守備 3",
			"移動 5",
			"武器: 長弓（弓）",
			"威力 6　命中 85　射程 2〜3",
			"手持ち: 回復薬 ×1、特効薬 ×2",
			"スキル: 狙撃",
			"　移動せずに攻撃すると命中 +20",
		}
		unitRows := func(u tactics.Unit) []string { return panelRows(Panel{Unit: &u}) }
		if got := unitRows(u); !reflect.DeepEqual(got, want) {
			t.Errorf("行 = %q\nのはず %q", got, want)
		}
		descs := map[string]tactics.Skill{
			"　攻撃を受けると 30% でダメージ半分": pavise,
			"　HP が半分以下なら必殺（3 倍）":   wrath,
		}
		for d, sk := range descs {
			u.Skill = sk
			if got := unitRows(u); got[len(got)-1] != d || got[len(got)-2] != "スキル: "+sk.Name {
				t.Errorf("%s の説明 = %q", sk.Name, got[len(got)-2:])
			}
		}
		for typ, name := range map[tactics.WeaponType]string{tactics.Sword: "剣", tactics.Lance: "槍", tactics.Axe: "斧", tactics.Bow: "弓"} {
			u.Weapon.Type = typ
			if got := unitRows(u)[8]; got != "武器: 長弓（"+name+"）" {
				t.Errorf("武器の種類 %v = %q", typ, got)
			}
		}
		fort := tactics.Terrain{Name: "砦", Cost: 1, Avoid: 20, Def: 2, HealPercent: 20}
		if got := TerrainLines(fort); !reflect.DeepEqual(got, []string{"地形: 砦", "回避 20　守備 2　回復 20%"}) {
			t.Errorf("砦 = %q", got)
		}
	})

	t.Run("TC-FR-UNT-09-04 カーソルがユニットにないときは地形だけ、盤面の外では何も出さない", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		if got := panelRows(c.Panel(tactics.Pos{X: 0, Y: 0}, true)); !reflect.DeepEqual(got, []string{"地形: 平地", "回避 0　守備 0　回復 0%"}) {
			t.Errorf("空きマス = %q", got)
		}
		for _, tc := range []struct {
			p  tactics.Pos
			ok bool
		}{{tactics.Pos{X: 2, Y: 3}, false}, {tactics.Pos{X: 7, Y: 0}, true}, {tactics.Pos{X: 0, Y: 7}, true}, {tactics.Pos{X: 0, Y: -1}, true}} {
			if got := c.Panel(tc.p, tc.ok); got.Unit != nil || got.Terrain != nil || got.Forecast != nil || len(got.View().Texts) != 0 {
				t.Errorf("盤面の外 %v %v = %+v", tc.p, tc.ok, got)
			}
		}
	})

	t.Run("TC-FR-UNT-09-05 移動先を選んだあとは移動先のユニット、再生中は出来事の写しのユニットで出し、倒れたユニットは出さない", func(t *testing.T) {
		c := newIdle(t)
		// 主人公を (0, 1) に動かした Moved。もとのマスには誰もいない。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 1})
		if got := panelRows(c.Panel(tactics.Pos{X: 0, Y: 1}, true)); len(got) == 0 || got[0] != "レオン" {
			t.Errorf("移動先 = %q", got)
		}
		if got := panelRows(c.Panel(tactics.Pos{X: 0, Y: 0}, true)); got[0] != "地形: 平地" {
			t.Errorf("もとのマス = %q", got)
		}
		c.Cancel()
		c.Cancel()
		// 主人公がファイターを攻撃する。1 撃目の写しでは、反撃の前なので主人公の HP は 20、ファイターは 29。
		target(t, c, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0})
		c.Click(tactics.Pos{X: 1, Y: 0})
		if e, _, ok := c.Event(); !ok || e.Kind != tactics.EventStrike || e.Unit != 0 {
			t.Fatalf("再生中の出来事 = %+v", e)
		}
		if c.Battle().Units()[0].HP != 18 {
			t.Fatalf("戦闘の主人公の HP = %d", c.Battle().Units()[0].HP)
		}
		if got := panelRows(c.Panel(tactics.Pos{X: 0, Y: 0}, true)); got[2] != "HP 20/20" {
			t.Errorf("再生中の主人公 = %q", got)
		}
		if got := panelRows(c.Panel(tactics.Pos{X: 1, Y: 0}, true)); got[2] != "HP 29/30" {
			t.Errorf("再生中のファイター = %q", got)
		}
		playOut(c)
		if got := panelRows(c.Panel(tactics.Pos{X: 0, Y: 0}, true)); got[2] != "HP 18/20" {
			t.Errorf("再生のあとの主人公 = %q", got)
		}

		// ファイターの HP を 1 にした盤面で、主人公がファイターを倒す。倒れた敵のマスは地形だけになり、
		// あとから味方がそのマスに入れば（Moved）、味方の情報が出る。
		k := newIdleWith(t, func() tactics.Setup {
			s := basicSetup()
			s.Units[3].Stats.HP = 1
			return s
		})
		target(t, k, tactics.Pos{X: 0, Y: 0}, tactics.Pos{X: 1, Y: 0})
		k.Click(tactics.Pos{X: 1, Y: 0})
		playOut(k)
		if u := k.Battle().Units()[3]; u.Alive || u.Pos != (tactics.Pos{X: 1, Y: 0}) || k.State() != Idle {
			t.Fatalf("前提: ファイター %+v、状態 %v", u, k.State())
		}
		if got := panelRows(k.Panel(tactics.Pos{X: 1, Y: 0}, true)); !reflect.DeepEqual(got, []string{"地形: 平地", "回避 0　守備 0　回復 0%"}) {
			t.Errorf("倒れた敵のマス = %q", got)
		}
		k.Click(tactics.Pos{X: 0, Y: 2})
		k.Click(tactics.Pos{X: 1, Y: 0})
		if got := panelRows(k.Panel(tactics.Pos{X: 1, Y: 0}, true)); k.State() != Moved || len(got) == 0 || got[0] != "ガルド" {
			t.Errorf("倒れた敵のマスに入った味方 = %q（状態 %v）", got, k.State())
		}
	})
}

// 攻撃の予測（DES-BTL-07、FR-BTL-08、UC-03）の表示を確かめる。
func TestForecastPanel(t *testing.T) {
	t.Run("TC-FR-BTL-08-04 計算例 1: 攻撃する側と守る側の名前・HP・ダメージ・命中率・回数（追撃 ×2）、ボスの印", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		want := []ForecastRow{
			{"", "レオン", "ソルジャー 3★"},
			{"HP", "20 → 11", "18 → 8"},
			{"ダメージ", "5", "9"},
			{"命中率", "63%", "85%"},
			{"回数", "×2", "×1"},
			{"必殺", "―", "―"},
			{"スキル", "―", "―"},
		}
		p := c.Panel(tactics.Pos{X: 3, Y: 3}, true)
		if !reflect.DeepEqual(p.Forecast, want) {
			t.Errorf("予測 = %q\nのはず %q", p.Forecast, want)
		}
		// 予測の下には、カーソルのあるユニット（相手）の情報も出す。
		v := p.View()
		if name := textOf(t, v, "ソルジャー 3　★ボス"); name.Style != StyleTitle || name.Y < forecastBottom(len(want)) {
			t.Errorf("相手の名前 = %+v、予測の表の下 %d より下のはず", name, forecastBottom(len(want)))
		}
	})

	t.Run("TC-FR-BTL-08-05 計算例 4: 鼓舞と大盾 30% を効くスキルに出す", func(t *testing.T) {
		c := newIdleWith(t, example4Setup)
		target(t, c, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2})
		want := []ForecastRow{
			{"", "ガルド", "ファイター"},
			{"HP", "24 → 18", "24 → 15"},
			{"ダメージ", "9", "6"},
			{"命中率", "73%", "79%"},
			{"回数", "×1", "×1"},
			{"必殺", "―", "―"},
			{"スキル", "鼓舞", "―"},
			{"", "大盾 30%", ""},
		}
		if got := c.Panel(tactics.Pos{X: 3, Y: 2}, false).Forecast; !reflect.DeepEqual(got, want) {
			t.Errorf("予測 = %q\nのはず %q", got, want)
		}
	})

	t.Run("TC-FR-BTL-08-06 相手を選ぶ前と、取り消して Moved に戻ったあとは予測を出さない", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.ChooseAction(tactics.Attack)
		if got := c.Panel(tactics.Pos{X: 3, Y: 3}, true).Forecast; got != nil {
			t.Errorf("相手を選ぶ前 = %q", got)
		}
		c.Click(tactics.Pos{X: 3, Y: 3})
		if c.Panel(tactics.Pos{X: 3, Y: 3}, true).Forecast == nil {
			t.Fatalf("相手を選んだのに予測がない")
		}
		c.Cancel()
		if got := c.Panel(tactics.Pos{X: 3, Y: 3}, true).Forecast; got != nil {
			t.Errorf("取り消したあと（Moved） = %q", got)
		}
	})

	t.Run("TC-FR-BTL-08-07 反撃できない相手は「反撃なし」で、回数 0 の側には必殺と怒りを出さない。回数のある側には出す", func(t *testing.T) {
		c := newIdleWith(t, wrathSetup)
		// 剣兵が怒りの敵を攻撃し、HP を 20 から 10（最大の半分）にする。剣兵は守備 30 で反撃は 0。
		target(t, c, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2})
		c.Click(tactics.Pos{X: 3, Y: 2})
		playOut(c)
		if hp := c.Battle().Units()[4].HP; hp != 10 || c.State() != Idle {
			t.Fatalf("怒りの敵の HP = %d、状態 %v", hp, c.State())
		}
		// 弓兵が距離 2 から攻撃する。怒りの敵は反撃できない（回数 0）が、tactics の予測では必殺と怒りが真になる（REV-011 F-04）。
		target(t, c, tactics.Pos{X: 3, Y: 4}, tactics.Pos{X: 3, Y: 2})
		_, dest, tg := c.Selection()
		f, err := c.Battle().Forecast(tactics.Command{Unit: 2, To: dest, Action: tactics.Attack, Target: tg})
		if err != nil || f.Defender.Count != 0 || !f.Defender.Crit {
			t.Fatalf("前提: 予測 %+v, %v", f.Defender, err)
		}
		want := []ForecastRow{
			{"", "弓兵", "ブラン"},
			{"HP", "20 → 20", "10 → 9"},
			{"ダメージ", "1", "―"},
			{"命中率", "100%", "―"},
			{"回数", "×1", "反撃なし"},
			{"必殺", "―", "―"},
			{"スキル", "狙撃", "―"},
		}
		if got := c.Panel(tactics.Pos{X: 3, Y: 2}, true).Forecast; !reflect.DeepEqual(got, want) {
			t.Errorf("反撃なし = %q\nのはず %q", got, want)
		}
		// 隣の剣士が攻撃すると、怒りの敵は反撃できるので、必殺と怒りを出す。
		c.Cancel()
		c.Cancel()
		c.Cancel()
		target(t, c, tactics.Pos{X: 4, Y: 2}, tactics.Pos{X: 3, Y: 2})
		want = []ForecastRow{
			{"", "剣士", "ブラン"},
			{"HP", "20 → 20", "10 → 8"},
			{"ダメージ", "2", "0"},
			{"命中率", "100%", "100%"},
			{"回数", "×1", "×1"},
			{"必殺", "―", "あり"},
			{"スキル", "―", "怒り"},
		}
		if got := c.Panel(tactics.Pos{X: 3, Y: 2}, true).Forecast; !reflect.DeepEqual(got, want) {
			t.Errorf("反撃あり = %q\nのはず %q", got, want)
		}
	})
}

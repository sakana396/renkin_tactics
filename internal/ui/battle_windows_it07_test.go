package ui

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// fallbackSetup は 16 × 16 の平地。ガルド (5, 8)（アーマー、鉄の槍、大盾、回復薬）の上にファイター (5, 7)（鉄の斧）。
// 相手のマス (160,224)〜(192,256) は盤面の上の候補 2 つに、ガルドのマス (160,256)〜(192,288) は下の候補 2 つに重なるので、
// 予測の窓は右欄に収め直す（DES-IT07 3.4 節）。
func fallbackSetup() tactics.Setup {
	return tactics.Setup{Terrain: field(16, 16), Units: []tactics.UnitSetup{
		{Name: "ガルド", ClassName: "アーマー", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 24, Str: 6, Skl: 3, Spd: 2, Def: 10, Mov: 4}, Pos: tactics.Pos{X: 5, Y: 8}, Weapon: ironLance, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Skill: pavise},
		{Name: "ファイター", ClassName: "ファイター", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 22, Str: 5, Skl: 3, Spd: 4, Def: 2, Mov: 5}, Pos: tactics.Pos{X: 5, Y: 7}, Weapon: ironAxe, AI: tactics.Hold},
		{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 15, Y: 15}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
	}}
}

// checkForecastWindow は、予測の窓の文・絵・帯が窓の中にあり、文どうしが重ならないことを確かめる（DES-UI-31）。
// 盤面の窓はミニマップ・右欄を覆わず、右欄の窓はミニマップに重ならない。
func checkForecastWindow(t *testing.T, step string, w ForecastWindow) {
	t.Helper()
	if w.Fallback {
		if w.Rect != image.Rect(512, 0, 832, 360) {
			t.Errorf("%s: 右欄の窓 %v", step, w.Rect)
		}
	} else if w.Rect.Dx() != 416 || w.Rect.Dy() != 240 || !w.Rect.In(image.Rect(0, 0, MapSize, MapSize)) {
		t.Errorf("%s: 盤面の窓 %v", step, w.Rect)
	}
	if w.Rect.Overlaps(MinimapRect()) {
		t.Errorf("%s: 窓 %v がミニマップに重なる", step, w.Rect)
	}
	if b := w.View.Banner; b != nil && !b.Rect.In(w.Rect) {
		t.Errorf("%s: 帯 %v が窓の外", step, b.Rect)
	}
	for _, ic := range w.View.Icons {
		if r := image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size); !r.In(w.Rect) {
			t.Errorf("%s: 絵 %v が窓の外", step, r)
		}
	}
	for i, a := range w.View.Texts {
		ra := rectOfText(a.X, a.Y, a.Text, a.Big)
		if !ra.In(w.Rect) {
			t.Errorf("%s: 文 %q %v が窓の外", step, a.Text, ra)
		}
		if a.Big {
			t.Errorf("%s: 文 %q を大きく描く", step, a.Text)
		}
		for _, b := range w.View.Texts[i+1:] {
			if rb := rectOfText(b.X, b.Y, b.Text, b.Big); ra.Overlaps(rb) {
				t.Errorf("%s: 文 %q %v と %q %v が重なる", step, a.Text, ra, b.Text, rb)
			}
		}
		for _, ic := range w.View.Icons {
			if r := image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size); ra.Overlaps(r) {
				t.Errorf("%s: 文 %q が絵 %v に重なる", step, a.Text, r)
			}
		}
	}
}

// textAt は v の文のうち、text と同じものを返す。なければ失敗にする。
func textAt(t *testing.T, v InfoView, text string) InfoText {
	t.Helper()
	for _, tx := range v.Texts {
		if tx.Text == text {
			return tx
		}
	}
	t.Fatalf("%q がない: %q", text, rows(v))
	return InfoText{}
}

// 共通の予測窓・右欄の上下 2 段・詳しい画面を、設計の座標と core の値で確かめる（DES-UI-31・34、DES-UI-07 の改訂、#166）。
func TestIT07BattleWindows(t *testing.T) {
	t.Run("TC-FR-BTL-08-30 窓は相手と仮の移動先の両マスを避けて 4 候補から置き、相手と反対側の上下を優先する。どれも重なれば右欄に収め直す", func(t *testing.T) {
		camera := Camera{Width: 16, Height: 16}
		top, topR := image.Rect(0, 8, 416, 248), image.Rect(96, 8, 512, 248)
		bottom, bottomR := image.Rect(0, 264, 416, 504), image.Rect(96, 264, 512, 504)
		right := image.Rect(512, 0, 832, 360)
		for _, tc := range []struct {
			name         string
			target, dest tactics.Pos
			want         image.Rectangle
		}{
			// 相手 (32,32)〜(64,64) は上半分なので下の候補から。下の左は両マスに重ならない。
			{"相手が上", tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 2, Y: 1}, bottom},
			// 相手 (32,448)〜(64,480) は下半分なので上の候補から。
			{"相手が下", tactics.Pos{X: 1, Y: 14}, tactics.Pos{X: 1, Y: 13}, top},
			// 移動先 (64,288)〜(96,320) が下の左に重なる。下の右（x = 96〜）には重ならない。
			{"移動先が下の左を塞ぐ", tactics.Pos{X: 2, Y: 7}, tactics.Pos{X: 2, Y: 9}, bottomR},
			// 移動先 (416,32)〜(448,64) が上の右に重なり、相手 (448,448)〜(480,480) は下の 2 つに重なる。上の左（x < 416）は空く。
			{"右端の上下", tactics.Pos{X: 14, Y: 14}, tactics.Pos{X: 13, Y: 1}, top},
			// 長弓の射程 3 で離れる: 相手 (448,288)〜(480,320) は下半分、移動先 (448,192)〜(480,224) は上の右に重なる。上の左は空く。
			{"射程 3 で離れて置ける", tactics.Pos{X: 14, Y: 9}, tactics.Pos{X: 14, Y: 6}, top},
			// 長弓の射程 3 で離れる: 移動先 (96,192)〜(128,224) が上の 2 つに、相手 (96,288)〜(128,320) が下の 2 つに重なる。
			{"射程 3 で離れて置けない", tactics.Pos{X: 3, Y: 9}, tactics.Pos{X: 3, Y: 6}, right},
			// 隣り合う: 相手 (160,224)〜(192,256) が上の 2 つに、移動先 (160,256)〜(192,288) が下の 2 つに重なる。
			{"上下とも塞ぐ", tactics.Pos{X: 5, Y: 7}, tactics.Pos{X: 5, Y: 8}, right},
		} {
			r, fallback := ForecastRect(camera, tc.target, tc.dest)
			if r != tc.want || fallback != (tc.want == right) {
				t.Errorf("%s: 窓 %v、右欄 %v（%v のはず）", tc.name, r, fallback, tc.want)
			}
		}
		// カメラを動かしたときは画面の座標で避ける。カメラ y = 8 なら (1, 9) は (32,32) に描くので、下の候補を選ぶ。
		scrolled := Camera{Width: 32, Height: 32, Y: 8}
		if r, _ := ForecastRect(scrolled, tactics.Pos{X: 1, Y: 9}, tactics.Pos{X: 1, Y: 10}); r != bottom {
			t.Errorf("カメラを動かしたとき: %v", r)
		}
		_ = topR

		// 実際の予測: 計算例 1（相手 (3,3) は上半分）は盤面の下の左。右欄の窓では右欄の通常情報の代わりに窓が「詳細 I」「消す」の位置を受ける。
		c := newIdleWith(t, example1Setup)
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		if w, ok := c.ForecastWindow(); !ok || w.Rect != bottom || w.Fallback {
			t.Errorf("計算例 1 の窓 %v %v", w.Rect, w.Fallback)
		}
		f := newIdleWith(t, fallbackSetup)
		target(t, f, tactics.Pos{X: 5, Y: 8}, tactics.Pos{X: 5, Y: 7})
		w, ok := f.ForecastWindow()
		if !ok || w.Rect != right || !w.Fallback {
			t.Fatalf("右欄の窓 %v %v", w.Rect, w.Fallback)
		}
		checkForecastWindow(t, "右欄", w)
		// 予測を出していないとき・巻き戻しの一覧の間は窓を出さない。
		f.OpenRewind()
		if _, ok := f.ForecastWindow(); ok {
			t.Error("巻き戻しの一覧の間に窓を出した")
		}
		f.battleKey("Esc")
		if _, ok := f.ForecastWindow(); !ok {
			t.Error("一覧を取り消したあと窓が戻らない")
		}
		f.Cancel()
		if _, ok := f.ForecastWindow(); ok {
			t.Error("相手を選んでいないのに窓を出した")
		}
	})

	t.Run("TC-FR-BTL-08-31 窓に帯・両者の名前・武器と相性・6 行の表・効くスキルと特効・武器送りを Battle.Forecast の値で出し、盤面の「5×2 63%」と危険の赤枠を保つ。回復は同じ窓に HealForecast の値", func(t *testing.T) {
		forecastOf := func(c *Controller) tactics.Forecast {
			t.Helper()
			w := c.weapon
			f, err := c.Battle().Forecast(tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.Attack, Target: c.target, Weapon: &w})
			if err != nil {
				t.Fatal(err)
			}
			return f
		}
		for _, tc := range []struct {
			name           string
			setup          func() tactics.Setup
			banner         string
			after          [2]string
			bannerStyle    InfoStyle
			label          string
			dangerFrameOn  bool
			bannerExpected bool
		}{
			// 計算例 1: レオン 20 → 11（ソルジャー 9×1）、ソルジャー 18 → 8（レオン 5×2）。
			{"通常", example1Setup, "", [2]string{"20 → 11", "18 → 8"}, 0, "5×2 63%", false, false},
			// レオンの HP 9: 反撃 9 で倒れる。
			{"危険", example1HP(9), "危険　反撃で倒される", [2]string{"9 → 0", "18 → 13"}, StyleBad, "危険 5×2 63%", true, true},
			// ソルジャーの HP 10: 5×2 で倒せる。
			{"撃破", func() tactics.Setup { s := example1Setup(); s.Units[1].Stats.HP = 10; return s }, "撃破　この攻撃で倒せる", [2]string{"20 → 11", "10 → 0"}, StyleTitle, "撃破 5×2 63%", false, true},
		} {
			c := newIdleWith(t, tc.setup)
			target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
			w, ok := c.ForecastWindow()
			if !ok {
				t.Fatalf("%s: 窓がない", tc.name)
			}
			checkForecastWindow(t, tc.name, w)
			v, o := w.View, w.Rect.Min
			at := func(text string, x, y int) {
				t.Helper()
				if tx := textAt(t, v, text); tx.X != o.X+x || tx.Y != o.Y+y {
					t.Errorf("%s: %q の位置 (%d,%d)、窓の中の (%d,%d) のはず", tc.name, text, tx.X-o.X, tx.Y-o.Y, x, y)
				}
			}
			// 帯 (8,8)〜(408,36)。
			if tc.bannerExpected {
				if v.Banner == nil || v.Banner.Rect != image.Rect(o.X+8, o.Y+8, o.X+408, o.Y+36) || v.Banner.Style != tc.bannerStyle {
					t.Errorf("%s: 帯 %+v", tc.name, v.Banner)
				}
				textAt(t, v, tc.banner)
			} else if v.Banner != nil {
				t.Errorf("%s: 通常なのに帯 %+v", tc.name, v.Banner)
			}
			// 名前 (16,40)・(280,40)、武器 (16,60)〜（絵 32 px、名前は絵の右）。剣は槍に不利で、攻撃する側に ▼、守る側に ▲。
			at("レオン", 16, 40)
			at("ソルジャー 3★", 280, 40)
			at("鉄の剣（剣）", 16+WeaponIconSize+4, 60)
			at("鉄の槍（槍）", 280+WeaponIconSize+4, 60)
			at("▼", 16+WeaponIconSize+4, 76)
			at("▲", 280+WeaponIconSize+4, 76)
			if len(v.Icons) != 2 || v.Icons[0] != (InfoIcon{X: o.X + 16, Y: o.Y + 60, Size: WeaponIconSize, Tile: mustTile(t, tactics.Sword)}) || v.Icons[1].X != o.X+280 || v.Icons[1].Size != WeaponIconSize {
				t.Errorf("%s: 武器の絵 %+v", tc.name, v.Icons)
			}
			// 6 行の表 (16,92)〜(400,188)（DES-IT07 v0.7、E-127）。値は Battle.Forecast と一致し、必殺・大盾は予測後 HP に入れない。
			f := forecastOf(c)
			labels := []string{"HP", "ダメージ", "命中", "必殺", "速さ", "反撃"}
			attacker := []string{tc.after[0], fmt.Sprintf("%d×%d", f.Attacker.Damage, f.Attacker.Count), fmt.Sprintf("%d%%", f.Attacker.HitRate), fmt.Sprintf("%d%%", f.Attacker.CritRate), fmt.Sprint(f.Attacker.AttackSpeed), "―"}
			defender := []string{tc.after[1], fmt.Sprintf("%d×%d", f.Defender.Damage, f.Defender.Count), fmt.Sprintf("%d%%", f.Defender.HitRate), fmt.Sprintf("%d%%", f.Defender.CritRate), fmt.Sprint(f.Defender.AttackSpeed), "あり"}
			if attacker[1] != "5×2" || attacker[2] != "63%" || defender[1] != "9×1" {
				t.Errorf("%s: 計算例 1 の値と違う: %v %v", tc.name, attacker, defender)
			}
			for i := range labels {
				y := 92 + i*16
				if y+lineHeight > 188 {
					t.Fatalf("表の行 %d が欄の外", i)
				}
				for _, cell := range []struct {
					text string
					x    int
				}{{labels[i], 16}, {attacker[i], 96}, {defender[i], 280}} {
					found := false
					for _, tx := range v.Texts {
						if tx.Text == cell.text && tx.X == o.X+cell.x && tx.Y == o.Y+y {
							found = true
						}
					}
					if !found {
						t.Errorf("%s: 表 (%d,%d) に %q がない: %q", tc.name, cell.x, y, cell.text, rows(v))
					}
				}
			}
			// 効くスキルと特効 (16,188)〜、武器送り・決定・戻る (16,220)〜。
			at("スキル ―　特効 ―", 16, 188)
			at("◀ 鉄の剣 ▶　仮選択", 16, 220)
			at("決定", 416-96, 220)
			at("戻る", 416-48, 220)
			// 相手のマスの上の文字と、危険時の仮の移動先の赤枠（DES-UI-07）。
			if m := c.TargetView(); m.Text != tc.label || m.Pos != (tactics.Pos{X: 3, Y: 3}) || m.Danger != tc.dangerFrameOn || (tc.dangerFrameOn && m.DangerPos != c.dest) {
				t.Errorf("%s: 盤面の表示 %+v", tc.name, m)
			}
		}

		// 反撃なし・特効・効くスキル: 距離 1 の弓兵は弓で反撃できない。アーマーキラーは鎧に特効、ガルドの大盾は「大盾 30%」。
		killer := tactics.Weapon{Name: "アーマーキラー", Type: tactics.Sword, Might: 5, Hit: 80, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"armor"}}
		c := newIdleWith(t, func() tactics.Setup {
			return tactics.Setup{Terrain: field(7, 7), Units: []tactics.UnitSetup{
				{Name: "剣士", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Mov: 3}, Pos: tactics.Pos{X: 1, Y: 1}, Weapon: killer},
				{Name: "弓兵", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30, Def: 2}, Pos: tactics.Pos{X: 2, Y: 1}, Weapon: ironBow, AI: tactics.Hold, Tags: []string{"armor"}, Skill: pavise, Boss: true},
			}}
		})
		target(t, c, tactics.Pos{X: 1, Y: 1}, tactics.Pos{X: 2, Y: 1})
		w, _ := c.ForecastWindow()
		f := forecastOf(c)
		all := strings.Join(rows(w.View), " ")
		if f.Defender.Count != 0 || !f.Attacker.Effective {
			t.Fatalf("前提: 反撃 %d、特効 %v", f.Defender.Count, f.Attacker.Effective)
		}
		for _, s := range []string{"反撃なし", "なし", fmt.Sprintf("スキル ―　特効 威力 %d", f.Attacker.EffectiveMight), "大盾 30%"} {
			if !strings.Contains(all, s) {
				t.Errorf("%q がない: %s", s, all)
			}
		}

		// 武器を送ると、送った武器で Battle.Forecast を呼び直した値を出す。窓の「仮選択」の武器名も替わる。
		// 直接クリックで武器ごとに仮の位置が違うときは、窓に仮の移動先を出す（DES-UI-34）。
		q := newIdleWith(t, quickSetup)
		q.Click(tactics.Pos{X: 1, Y: 1})
		q.Click(tactics.Pos{X: 3, Y: 3})
		if q.State() != Targeting || q.quickSpots == nil {
			t.Fatalf("前提: 直接クリック %v", q.State())
		}
		names := map[string]bool{}
		for range 2 {
			w, ok := q.ForecastWindow()
			f := forecastOf(q)
			name := q.Units()[0].Weapons[q.weapon].Name
			all := strings.Join(rows(w.View), " ")
			names[name] = true
			if s := fmt.Sprintf("位置 %d,%d", q.dest.X, q.dest.Y); !strings.Contains(all, s) {
				t.Errorf("仮の位置 %q がない: %s", s, all)
			}
			if !ok || !strings.Contains(all, "◀ "+name+" ▶　仮選択") || !strings.Contains(all, fmt.Sprintf("%d×%d", f.Attacker.Damage, f.Attacker.Count)) || !strings.Contains(all, fmt.Sprintf("%d%%", f.Attacker.HitRate)) {
				t.Errorf("武器 %s の窓: %s", name, all)
			}
			q.ShiftForecastWeapon(1)
		}

		if len(names) != 2 {
			t.Errorf("武器が送られない: %v", names)
		}
		if strings.Contains(all, "位置 ") {
			t.Error("移動してからの攻撃に位置を出した")
		}

		// 回復: 同じ窓の枠に HealForecast の値を出す。杖の選択順は変えない。
		h := healController(t, 15)
		moveHealer(h, tactics.Pos{X: 3, Y: 2})
		h.ChooseAction(tactics.Heal)
		h.Click(tactics.Pos{X: 3, Y: 1})
		hw, ok := h.ForecastWindow()
		wpn := h.weapon
		hf, err := h.Battle().HealForecast(tactics.Command{Unit: h.unit, To: h.dest, Action: tactics.Heal, Target: h.target, Weapon: &wpn})
		if !ok || err != nil {
			t.Fatalf("回復の窓 %v %v", ok, err)
		}
		checkForecastWindow(t, "回復", hw)
		hall := strings.Join(rows(hw.View), " ")
		for _, s := range []string{"回復の予測", "リナ → レオン", fmt.Sprintf("HP %d → %d", hf.HP, hf.After), fmt.Sprintf("回復 +%d", hf.Amount), "杖 残り 15 → 14/15"} {
			if !strings.Contains(hall, s) {
				t.Errorf("回復の窓に %q がない: %s", s, hall)
			}
		}
	})

	t.Run("TC-FR-UNT-09-24 右欄の上段にカーソルの下、下段に選択中のユニット。同じ人物なら下段は地形・スキル。両段に装備の名前・種類と E 印。操作行に範囲の状態・詳細 I・消す", func(t *testing.T) {
		two := func() tactics.Setup {
			s := example1Setup()
			s.Units[0].Weapon = tactics.Weapon{}
			s.Units[0].Weapons = []tactics.Weapon{ironSword, ironBow}
			s.Units[0].AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword, tactics.Bow}
			return s
		}
		c := newIdleWith(t, two)
		c.Click(tactics.Pos{X: 2, Y: 3})
		bx, by := cellCenter(tactics.Pos{X: 3, Y: 3})
		v := c.BattleInfoView(bx, by)
		// 上段 (512,0)〜(832,176) はソルジャー、下段 (512,176)〜(832,304) はレオン。文は x = 520、段の上端 + 8 から 16 px ずつ。
		for i, s := range []string{"ソルジャー 3　★ボス", "ソルジャー　Lv 1", "HP 18/18", "装備 E 鉄の槍（槍）", "力/魔力 5/0　守備/魔防 4/0", "速さ 4"} {
			if tx := textAt(t, v, s); tx.X != 520 || tx.Y != 8+i*16 {
				t.Errorf("上段の %q (%d,%d)", s, tx.X, tx.Y)
			}
		}
		for i, s := range []string{"レオン", "ロード", "HP 20/20", "装備 E 鉄の剣（剣）", "力/魔力 6/0　守備/魔防 4/0", "速さ 9"} {
			if tx := textAt(t, v, s); tx.X != 520 || tx.Y != 184+i*16 {
				t.Errorf("下段の %q (%d,%d)", s, tx.X, tx.Y)
			}
		}
		// 地形は下段の最下行 (520,280)。カーソルのマス（森）。
		if tx := textAt(t, v, "地形: 森　回避 20　守備 1　回復 0%"); tx.X != 520 || tx.Y != 280 {
			t.Errorf("地形の行 (%d,%d)", tx.X, tx.Y)
		}
		// 絵は各段の右上 (760, 段の上端 + 8)。
		if len(v.Icons) != 2 || v.Icons[0].X != 760 || v.Icons[0].Y != 8 || v.Icons[1].X != 760 || v.Icons[1].Y != 184 || v.Icons[0].Size != PortraitSize {
			t.Errorf("絵 %+v", v.Icons)
		}
		// 中身は y < 296、ページの印はない。
		if len(v.Tabs) != 0 || v.Bottom != 296 {
			t.Errorf("印 %d、一番下 %d", len(v.Tabs), v.Bottom)
		}
		checkInfoArea(t, "2 段", v)
		for _, tx := range v.Texts {
			if r := rectOfText(tx.X, tx.Y, tx.Text, tx.Big); r.Max.Y > 296 || (tx.Y < 176 && r.Max.Y > 176) {
				t.Errorf("%q %v が段からはみ出す", tx.Text, r)
			}
			if tx.Y < 64+8 || (tx.Y >= 184 && tx.Y < 184+64) {
				if r := rectOfText(tx.X, tx.Y, tx.Text, tx.Big); r.Max.X > 760 {
					t.Errorf("%q %v が絵に重なる", tx.Text, r)
				}
			}
		}
		// 操作行 (512,304)〜(832,360): 範囲の状態 (520,304)、詳細 I (672,304)〜(744,336)、消す (752,304)〜(824,336)。
		rs := c.RangeStatusIn(v)
		if rs.Danger.X != 520 || rs.Danger.Y != 304 || rs.HideTexts || rs.HideClear || rs.Clear.Rect != image.Rect(752, 304, 824, 336) {
			t.Errorf("範囲の状態 %+v", rs)
		}
		if b := c.DetailButton(); b.Rect != image.Rect(672, 304, 744, 336) || b.Label != "詳細 I" || !b.Enabled || b.Rect.Overlaps(rs.Clear.Rect) || b.Rect.Overlaps(image.Rect(520, 304, 664, 336)) {
			t.Errorf("詳細 I %+v", b)
		}
		// 同じ人物（カーソルがレオンか空きマス）なら、上段にレオン、下段は地形・スキルの要約。
		for _, p := range []tactics.Pos{{X: 2, Y: 3}, {X: 0, Y: 0}} {
			v := c.BattleInfoView(cellCenter(p))
			got := rows(v)
			if len(got) != 9 || got[0] != "レオン" || got[6] != "地形・スキル" || got[7] != "鼓舞　距離 2 以内の味方の命中 +10・回避 +10" || !strings.HasPrefix(got[8], "地形: 平地") {
				t.Errorf("同じ人物 %v: %q", p, got)
			}
		}
		// 予測で武器を送っても、両段の E 印は装備（鉄の剣）のまま。
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 3, Y: 3})
		for range 2 {
			c.ShiftForecastWeapon(1)
			if c.weapon == 1 {
				break
			}
		}
		got := strings.Join(rows(c.BattleInfoView(bx, by)), "\n")
		if !strings.Contains(got, "装備 E 鉄の剣（剣）") || strings.Contains(got, "装備 E 鉄の弓") {
			t.Errorf("予測の仮選択で E 印が変わった（武器 %d）: %q", c.weapon, got)
		}
		// 配置でも、上段にカーソルの下（弓兵）、下段に選んでいる味方（主人公）を出す。
		ds := deployRangeScenes(t)
		clickDeployCell(ds, tactics.Pos{X: 2})
		ds.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		if dv := ds.View(); dv.Info == nil {
			t.Error("配置の右欄がない")
		} else if got := rows(*dv.Info); len(got) < 7 || got[0] != "弓兵" || got[6] != "主人公" {
			t.Errorf("配置の 2 段: %q", got)
		}
		// 武器を持たないユニットは「装備 なし」。
		bare := tactics.Unit{Name: "兵", Alive: true, HP: 10, Stats: tactics.Stats{HP: 10}, Equipped: -1}
		if got := rows(twoTierInfo([]tactics.Unit{bare}, 0, -1, nil)); len(got) < 4 || got[3] != "装備 なし" {
			t.Errorf("武器なし: %q", got)
		}
	})

	t.Run("TC-FR-UNT-09-25 I の詳しい画面はカーソルの下、なければ選択中のユニットを出し、能力 9 種・手持ち 5 枠・全武器・スキルを一画面に並べる。Tab は節を送り、閉じると元の状態と仮選択・仮配置に戻る", func(t *testing.T) {
		// 状態ごとに開くユニット: カーソルの下のユニットを優先し、空きマスなら選択中。どちらもなければ開かない。
		c := newIdleWith(t, example1Setup)
		c.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 0}))
		if c.OpenDetail() || c.DetailOpen() {
			t.Error("Idle の空きマスで開いた")
		}
		c.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		if !c.OpenDetail() || c.detailUnit != 1 {
			t.Errorf("Idle のソルジャー: %d", c.detailUnit)
		}
		c.DetailInput(false, true)
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 0}))
		if !c.OpenDetail() || c.detailUnit != 0 || c.State() != Selected {
			t.Errorf("Selected の空きマス: %d %v", c.detailUnit, c.State())
		}
		c.DetailInput(false, true)
		c.Click(tactics.Pos{X: 2, Y: 2})
		if !c.OpenDetail() || c.detailUnit != 0 || c.State() != Moved || c.dest != (tactics.Pos{X: 2, Y: 2}) {
			t.Errorf("Moved: %d %v", c.detailUnit, c.State())
		}
		c.DetailInput(false, true)
		// ItemMenu・StaffMenu・Canto も選択中のユニットを出す（selectedCell の状態）。状態だけを当てて確かめる。
		for _, st := range []State{ItemMenu, StaffMenu, Canto} {
			saved := c.state
			c.state = st
			if !c.OpenDetail() || c.detailUnit != 0 {
				t.Errorf("%v: %d", st, c.detailUnit)
			}
			c.DetailInput(false, true)
			c.state = saved
		}
		// Targeting: 開いて Tab で節を送り、閉じると相手・武器・仮の移動先が残る。
		c.Cancel()
		c.Cancel()
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		c.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		state, weapon, tg, dest := c.State(), c.weapon, c.target, c.dest
		if !c.OpenDetail() || c.detailUnit != 1 {
			t.Fatal("相手の詳しい画面を開けない")
		}
		for _, want := range []DetailSection{SectionWeapons, SectionSkills, SectionStats, SectionWeapons} {
			c.DetailInput(true, false)
			if c.detailSection != want {
				t.Errorf("Tab: 節 %d（%d のはず）", c.detailSection, want)
			}
		}
		c.DetailInput(false, true)
		if c.DetailOpen() || c.State() != state || c.weapon != weapon || c.target != tg || c.dest != dest {
			t.Error("閉じると予測の選択が変わった")
		}
		// HealTargeting も選択中（杖役）かカーソルの下を出す。
		h := healController(t, 15)
		moveHealer(h, tactics.Pos{X: 3, Y: 2})
		h.ChooseAction(tactics.Heal)
		h.Click(tactics.Pos{X: 3, Y: 1})
		h.SetCursor(cellCenter(tactics.Pos{X: 0, Y: 6}))
		if h.State() != HealTargeting || !h.OpenDetail() || h.detailUnit != 0 {
			t.Errorf("HealTargeting: %v %d", h.State(), h.detailUnit)
		}
		// 再生中・Over は開かない。
		p := newIdleWith(t, example1Setup)
		p.SetCursor(cellCenter(tactics.Pos{X: 2, Y: 3}))
		for _, st := range []State{Playing, Over} {
			p.state = st
			if p.OpenDetail() || p.DetailButton().Enabled {
				t.Errorf("%v で開いた", st)
			}
		}

		// 中身: 5 本の武器（装備は 2 本目）、2 つのスキル、手持ちの回復薬 2 個のノエル。
		u := tactics.Unit{Name: "ノエル", ClassName: "錬金術師", Side: tactics.Player, Alive: true, HP: 18, Level: 3, Exp: 20,
			Stats: tactics.Stats{HP: 20, Str: 4, Mag: 7, Skl: 5, Spd: 6, Lck: 9, Def: 3, Res: 8, Mov: 5},
			Weapons: []tactics.Weapon{
				{Name: "勇者の斧", Type: tactics.Axe, Might: 6, Hit: 65, Weight: 14, Strikes: 2, MinRange: 1, MaxRange: 1},
				{Name: "アーマーキラー", Type: tactics.Sword, Might: 4, Hit: 80, Weight: 7, MinRange: 1, MaxRange: 1, EffectiveTags: []string{"armor"}},
				{Name: "ホースキラー", Type: tactics.Lance, Might: 6, Hit: 75, Weight: 10, MinRange: 1, MaxRange: 1},
				{Name: "ファイア", Type: tactics.Magic, DamageType: tactics.Magical, Might: 5, Hit: 90, MinRange: 1, MaxRange: 2},
			},
			Equipped: 1,
			Skill:    tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}, ClassSkills: []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}},
			Items: []tactics.ItemStack{{Item: potion, Count: 1}}}
		v := DetailView(u, SectionWeapons)
		all := strings.Join(rows(v), "\n")
		// 能力 9 種（ラベルと値）。
		for _, s := range []string{"HP 18/20", "力 4", "魔力 7", "技 5", "速さ 6", "幸運 9", "守備 3", "魔防 8", "移動 5"} {
			if !strings.Contains(all, s) {
				t.Errorf("能力 %q がない", s)
			}
		}
		// 手持ち 5 枠: 武器 4 本（装備に E）、回復薬 1 個、空き 0。
		for _, s := range []string{"1　　 勇者の斧（斧）", "2　E アーマーキラー（剣）", "5　　回復薬（回復 10）"} {
			if !strings.Contains(all, s) {
				t.Errorf("手持ち %q がない: %s", s, all)
			}
		}
		// 全武器の詳しい値（名前を除いた weaponDetail の値）と、能力の欄の武器の短い行（種類の絵・名前・E 印。E-97）。
		for _, w := range u.Weapons {
			if !strings.Contains(all, strings.TrimPrefix(weaponDetail(w), w.Name+"　")) {
				t.Errorf("%s の詳しい値がない", w.Name)
			}
		}
		e := textAt(t, v, "E")
		if killer := textAt(t, v, "アーマーキラー（剣）"); e.Y != killer.Y || e.X >= killer.X {
			t.Errorf("能力の欄の E 印 %+v と名前 %+v", e, killer)
		}
		small := 0
		for _, ic := range v.Icons {
			if ic.Size == 16 {
				small++
			}
		}
		if small != 4 {
			t.Errorf("武器の種類の絵 %d 個（4 のはず）", small)
		}
		// スキル: 名前と説明。節の見出しは今の節にだけ ▶。
		for _, s := range []string{"怒り", "再移動", skillDescription(u.Skill)} {
			if !strings.Contains(all, s) {
				t.Errorf("スキル %q がない", s)
			}
		}
		for sec, heads := range map[DetailSection][3]string{
			SectionStats:   {"▶能力", "　手持ち 5 枠・全武器", "　スキル"},
			SectionWeapons: {"　能力", "▶手持ち 5 枠・全武器", "　スキル"},
			SectionSkills:  {"　能力", "　手持ち 5 枠・全武器", "▶スキル"},
		} {
			got := strings.Join(rows(DetailView(u, sec)), "\n")
			for _, h := range heads {
				if !strings.Contains(got, h) {
					t.Errorf("節 %d に %q がない", sec, h)
				}
			}
		}
		// 一画面 (0,0)〜(832,512) に収まり、文と絵が重ならない。
		area := image.Rect(0, 0, ScreenWidth, MapSize)
		for i, a := range v.Texts {
			ra := rectOfText(a.X, a.Y, a.Text, a.Big)
			if !ra.In(area) {
				t.Errorf("詳しい画面の %q %v が外", a.Text, ra)
			}
			for _, b := range v.Texts[i+1:] {
				if rb := rectOfText(b.X, b.Y, b.Text, b.Big); ra.Overlaps(rb) {
					t.Errorf("%q と %q が重なる", a.Text, b.Text)
				}
			}
			for _, ic := range v.Icons {
				if ra.Overlaps(image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size)) {
					t.Errorf("%q が絵に重なる", a.Text)
				}
			}
		}
		// 武器・スキルが多くて下にあふれる行は出さず「…」で止める。
		long := u
		for range 6 {
			long.ClassSkills = append(long.ClassSkills, tactics.Skill{Name: "再移動", Effect: tactics.Canto})
		}
		lv := DetailView(long, SectionSkills)
		if !strings.Contains(strings.Join(rows(lv), "\n"), "…") {
			t.Error("あふれたスキルを「…」で止めない")
		}
		for _, tx := range lv.Texts {
			if rectOfText(tx.X, tx.Y, tx.Text, tx.Big).Max.Y > MapSize {
				t.Errorf("%q が下にあふれる", tx.Text)
			}
		}

		// 配置: カーソルの下、なければ選択中。開け閉めと Tab で仮配置を変えない。
		s := deployRangeScenes(t)
		baseline := campaignBytes(t, s)
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 5}))
		if s.OpenDeployDetail() {
			t.Error("配置の空きマスで開いた")
		}
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		if !s.OpenDeployDetail() || s.View().Deploy.Units[s.detailUnit].Name != "弓兵" {
			t.Errorf("配置の弓兵: %d", s.detailUnit)
		}
		s.DeployDetailInput(true, false)
		if s.detailSection != SectionWeapons {
			t.Errorf("配置の Tab: %d", s.detailSection)
		}
		s.DeployDetailInput(false, true)
		clickDeployCell(s, tactics.Pos{X: 2})
		slots := append([]int(nil), s.deploySlots...)
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 3}))
		if !s.OpenDeployDetail() || s.View().Deploy.Units[s.detailUnit].Name != "主人公" {
			t.Errorf("配置の選択中: %d", s.detailUnit)
		}
		s.DeployDetailInput(false, true)
		if s.detailOpen || s.deploySelected != 0 || !reflect.DeepEqual(s.deploySlots, slots) || !bytes.Equal(baseline, campaignBytes(t, s)) || s.Scene() != Deploy {
			t.Errorf("配置の詳しい画面で仮配置が変わった: 選択 %d、枠 %v", s.deploySelected, s.deploySlots)
		}
	})

	t.Run("TC-IF-01-46 クリックは 詳しい画面 → 予測の窓 → ミニマップ → 情報欄 → 盤面 の順に 1 か所だけが受け、再生中・Over は新しい操作を受けない。T は巻き戻しの一覧を先に開き、取消しで重ね表示と仮選択に戻り、成功で消す", func(t *testing.T) {
		// 右欄に収め直した窓: 窓が「詳細 I」「消す」の位置を受け、詳しい画面も固定の解除もしない。
		c := newIdleWith(t, fallbackSetup)
		g := &Game{ctrl: c}
		g.leftClick(cellCenter(tactics.Pos{X: 15, Y: 15})) // ボスを固定する
		if c.WatchedCount() != 1 {
			t.Fatalf("前提: 固定 %d", c.WatchedCount())
		}
		target(t, c, tactics.Pos{X: 5, Y: 8}, tactics.Pos{X: 5, Y: 7})
		// 「詳細 I」(672,304) は窓の武器送りの行（右欄の窓では (16,304)〜(304,328)）の ▶ に当たり、武器 1 本なので何もしない。
		// 「消す」の下の端 (754,330) は窓の行の外で、窓が受けて何もしない。
		g.leftClick(detailButtonRect.Min.X+2, detailButtonRect.Min.Y+2)
		g.leftClick(rangeClearRect.Min.X+2, rangeClearRect.Max.Y-4)
		if c.DetailOpen() || c.WatchedCount() != 1 || c.State() != Targeting || c.target != 1 {
			t.Errorf("右欄の窓の下に渡した: 詳しい画面 %v、固定 %d、状態 %v", c.DetailOpen(), c.WatchedCount(), c.State())
		}
		// ミニマップは窓の外なので、窓を出していてもミニマップが受ける。
		mm := c.Minimap().Layout.Area
		if c.ForecastClick(mm.Min.X+2, mm.Min.Y+2) || !c.MinimapClick(mm.Min.X+2, mm.Min.Y+2) || c.State() != Targeting {
			t.Errorf("ミニマップのクリック: 状態 %v", c.State())
		}
		// 窓の行の右端の「戻る」は窓が受ける。
		g.leftClick(512+320-10, 304+4)
		if c.State() != Targeting || c.target != -1 {
			t.Errorf("右欄の窓の戻る: %v %d", c.State(), c.target)
		}

		// 盤面の窓: 窓の中の盤面のマスは窓が受け、窓の外の盤面は盤面が受ける。
		b := newIdleWith(t, example1Setup)
		bg := &Game{ctrl: b}
		target(t, b, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		w, _ := b.ForecastWindow()
		inside := image.Pt(w.Rect.Min.X+200, w.Rect.Min.Y+100)
		bg.leftClick(inside.X, inside.Y)
		if b.State() != Targeting || b.target != 1 {
			t.Errorf("窓の中のクリックを盤面に渡した: %v %d", b.State(), b.target)
		}
		// 窓の外の相手のマスをもう一度クリックすると盤面が受けて決定する。
		bg.leftClick(cellCenter(tactics.Pos{X: 3, Y: 3}))
		if b.State() != Playing {
			t.Errorf("窓の外の相手のクリック: %v", b.State())
		}
		// 再生中は窓も詳しい画面も出さない。
		if _, ok := b.ForecastWindow(); ok || b.OpenDetail() {
			t.Error("再生中に窓か詳しい画面を出した")
		}

		// 詳しい画面の間は、窓・盤面・右欄のどこをクリックしても受けない。
		d := newIdleWith(t, example1Setup)
		dg := &Game{ctrl: d}
		target(t, d, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		d.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		d.OpenDetail()
		dw, _ := d.ForecastWindow()
		for _, pt := range []image.Point{{dw.Rect.Min.X + 16, dw.Rect.Min.Y + 220}, cellPt(tactics.Pos{X: 3, Y: 3}), {700, 320}, {650, 400}} {
			dg.leftClick(pt.X, pt.Y)
		}
		if !d.DetailOpen() || d.State() != Targeting || d.target != 1 {
			t.Errorf("詳しい画面の間のクリック: 開く %v、状態 %v", d.DetailOpen(), d.State())
		}
		// T: 巻き戻しの一覧を先に開く。取り消すと詳しい画面と予測（相手・武器・仮の移動先）に戻る。
		weapon, dest := d.weapon, d.dest
		d.OpenRewind()
		if !d.rewind.open {
			t.Fatal("巻き戻しの一覧が開かない")
		}
		if _, ok := d.ForecastWindow(); ok {
			t.Error("一覧の間に窓を出した")
		}
		d.battleKey("Esc")
		if !d.DetailOpen() || d.State() != Targeting || d.target != 1 || d.weapon != weapon || d.dest != dest {
			t.Errorf("一覧の取消し: 開く %v、状態 %v、相手 %d、武器 %d", d.DetailOpen(), d.State(), d.target, d.weapon)
		}
		// 成功すると重ね表示と仮の武器を消し、Idle の盤面に戻る。
		r := newIdleWith(t, example1Setup)
		r.EndPhase()
		r.EndPhase()
		playOut(r)
		target(t, r, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		r.SetCursor(cellCenter(tactics.Pos{X: 3, Y: 3}))
		r.OpenDetail()
		r.OpenRewind()
		h := r.Battle().History()
		for j, e := range h {
			if e.Kind == tactics.HistoryEndPhase && e.Side == tactics.Player {
				r.rewindSelect(j)
			}
		}
		r.RewindEnter()
		r.RewindEnter()
		if r.DetailOpen() || r.State() != Idle || r.weapon != -1 || r.target != -1 {
			t.Errorf("巻き戻しの成功: 開く %v、状態 %v、武器 %d", r.DetailOpen(), r.State(), r.weapon)
		}
		if _, ok := r.ForecastWindow(); ok {
			t.Error("巻き戻しのあと窓が残る")
		}
	})
}

// mustTile は武器の種類の絵を返す。
func mustTile(t *testing.T, w tactics.WeaponType) assets.Tile {
	t.Helper()
	tile, ok := weaponTile(w)
	if !ok {
		t.Fatalf("%v の絵がない", w)
	}
	return tile
}

// Game.Update が最初に通す入口の優先順位と、配置の詳細ボタンを確かめる（DES-IT07 3.1・3.4 節）。
func TestIT07PriorityAndDeployDetail(t *testing.T) {
	t.Run("TC-IF-01-46 詳細中の T と Esc は巻き戻しだけを受ける", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		c.SetCursor(cellCenter(tactics.Pos{X: 2, Y: 3}))
		if !c.OpenDetail() {
			t.Fatal("詳細を開けない")
		}
		g := &Game{scenes: &SceneController{scene: Battle, battle: c}}
		if !g.priorityInput(priorityKeys{Rewind: true, Close: true}) || !c.rewind.open || !c.DetailOpen() {
			t.Fatalf("T と Esc: 一覧 %v、詳細 %v", c.rewind.open, c.DetailOpen())
		}
	})
	t.Run("TC-IF-01-46 配置の I とクリック・Q は詳細だけを受ける", func(t *testing.T) {
		for _, other := range []string{"クリック", "Q"} {
			s := deployRangeScenes(t)
			g := &Game{scenes: s}
			x, y := cellCenter(tactics.Pos{X: 4})
			before := append([]int(nil), s.deploySlots...)
			k := priorityKeys{Detail: true, X: x, Y: y}
			if other == "クリック" {
				k.Click = true
			} else {
				k.Danger = true
			}
			if !g.priorityInput(k) || !s.detailOpen {
				t.Fatalf("%s: I で開かない", other)
			}
			// Update はここで戻るので、このフレームのクリック・Q は配置へ届かない。
			if s.deployDanger || !reflect.DeepEqual(before, s.deploySlots) || s.deploySelected != -1 {
				t.Errorf("%s: 同時入力で配置が変わった", other)
			}
		}
	})
	t.Run("TC-IF-01-46 全体図の間は I で詳細を開かない", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		c.SetCursor(cellCenter(tactics.Pos{X: 2, Y: 3}))
		if !c.OpenOverview() {
			t.Fatal("全体図を開けない")
		}
		g := &Game{scenes: &SceneController{scene: Battle, battle: c}}
		if g.priorityInput(priorityKeys{Detail: true, X: c.cursorX, Y: c.cursorY}) || c.DetailOpen() {
			t.Error("全体図の間に詳細を開いた")
		}
	})
	t.Run("TC-FR-UNT-09-25 配置の詳細 I は描画とクリックで同じ座標", func(t *testing.T) {
		s := deployRangeScenes(t)
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		v := s.View()
		button := v.Buttons[2]
		if button.Label != "詳細 I" || button.Rect != deployDetailButtonRect || !button.Enabled {
			t.Fatalf("詳細のボタン %+v", button)
		}
		for _, b := range v.Buttons[:2] {
			if b.Rect.Overlaps(button.Rect) {
				t.Errorf("既存ボタンと重なる: %+v", b)
			}
		}
		s.Click(button.Rect.Min.X+2, button.Rect.Min.Y+2)
		if !s.detailOpen || s.detailUnit < 0 || s.View().Deploy.Units[s.detailUnit].Name != "弓兵" {
			t.Errorf("詳細ボタンのクリック: 開く %v、ユニット %d", s.detailOpen, s.detailUnit)
		}
	})
}

// 長い名前・武器名を盤面と右欄の両方で全文読めることを確かめる（DES-UI-31、REV-102 F-03）。
func TestIT07LongForecastNames(t *testing.T) {
	for _, tc := range []struct {
		name     string
		setup    func() tactics.Setup
		from, to tactics.Pos
		fallback bool
	}{
		{"盤面", example1Setup, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3}, false},
		{"右欄", fallbackSetup, tactics.Pos{X: 5, Y: 8}, tactics.Pos{X: 5, Y: 7}, true},
	} {
		t.Run("TC-FR-BTL-08-31 / TC-FR-BTL-08-32 "+tc.name+"の長い名前と武器名", func(t *testing.T) {
			attackerName := strings.Repeat("長", 24)
			defenderName := strings.Repeat("敵", 9)
			weaponName := strings.Repeat("剣", 16)
			setup := func() tactics.Setup {
				s := tc.setup()
				s.Units[0].Name, s.Units[1].Name = attackerName, defenderName
				s.Units[0].Weapon.Name = weaponName
				return s
			}
			c := newIdleWith(t, setup)
			target(t, c, tc.from, tc.to)
			w, ok := c.ForecastWindow()
			if !ok || w.Fallback != tc.fallback {
				t.Fatalf("窓 %+v、あり %v", w, ok)
			}
			checkForecastWindow(t, tc.name, w)
			all := strings.Join(rows(w.View), "")
			for _, want := range []string{attackerName, defenderName, weaponName} {
				if !strings.Contains(all, want) {
					t.Errorf("%q の全文がない: %q", want, all)
				}
			}
			if strings.Contains(all, "…") {
				t.Errorf("省略記号で切った: %q", all)
			}
		})
	}
}

// cellPt はマス p の中央の画面の座標。
func cellPt(p tactics.Pos) image.Point {
	x, y := cellCenter(p)
	return image.Pt(x, y)
}

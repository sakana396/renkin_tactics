package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// barOf は、文が label で始まる行の棒を返す（DES-UI-06）。棒は行の高さの中にある。
func barOf(t *testing.T, v InfoView, label string) InfoBar {
	t.Helper()
	for _, tx := range v.Texts {
		if strings.HasPrefix(tx.Text, label) {
			for _, b := range v.Bars {
				if b.Y >= tx.Y && b.Y < tx.Y+lineHeight {
					return b
				}
			}
			t.Fatalf("%q の行に棒がない", tx.Text)
		}
	}
	t.Fatalf("%q の行がない: %q", label, rows(v))
	return InfoBar{}
}

// portraitOf は、ユニットの絵（大きさ PortraitSize の絵）を返す。
func portraitOf(t *testing.T, v InfoView) InfoIcon {
	t.Helper()
	for _, ic := range v.Icons {
		if ic.Size == PortraitSize {
			return ic
		}
	}
	t.Fatalf("ユニットの絵がない: %+v", v.Icons)
	return InfoIcon{}
}

// forecastBottom は、表が n 行の予測（見出し 1 行・武器の絵・表）の一番下の y を返す（DES-UI-07）。
func forecastBottom(n int) int {
	return infoMargin + lineHeight + WeaponIconSize + n*lineHeight
}

// textOf は、文が s の InfoText を返す。
func textOf(t *testing.T, v InfoView, s string) InfoText {
	t.Helper()
	for _, tx := range v.Texts {
		if tx.Text == s {
			return tx
		}
	}
	t.Fatalf("文 %q がない: %q", s, rows(v))
	return InfoText{}
}

// 情報欄の見せ方（DES-UI-06、FR-UNT-09、UC-04）を確かめる。期待値は設計書 5.6 節の大きさ（絵 64 px、棒は値 × 6 px で 120 px まで、
// HP は 120 px の枠に割合で塗る）から求める。
func TestInfoView(t *testing.T) {
	// 計算例 1 のレオン（力 6・技 8・速さ 9・守備 4、HP 20、鉄の剣、鼓舞）。
	leon := func() tactics.Unit {
		return tactics.Unit{
			Name: "レオン", ClassName: "ロード", Side: tactics.Player, Lord: true, Alive: true,
			Stats: leonStats, HP: 20, Weapon: ironSword, Skill: inspire,
			Items: []tactics.ItemStack{{Item: potion, Count: 2}},
		}
	}
	view := func(u tactics.Unit) InfoView { return Panel{Unit: &u}.View() }

	t.Run("TC-FR-UNT-09-07 力・技・速さ・守備の棒は値 × 6 px で、120 px で止める。移動は棒にしない", func(t *testing.T) {
		for _, tc := range []struct{ value, want int }{{0, 0}, {1, 6}, {6, 36}, {19, 114}, {20, 120}, {25, 120}} {
			if got := StatBarLen(tc.value); got != tc.want {
				t.Errorf("StatBarLen(%d) = %d、%d のはず", tc.value, got, tc.want)
			}
		}
		u := leon()
		u.Stats.Def = 25
		v := view(u)
		for _, tc := range []struct {
			label string
			want  int
		}{{"力 6", 36}, {"技 8", 48}, {"速さ 9", 54}, {"守備 25", 120}} {
			if b := barOf(t, v, tc.label); b.Fill != tc.want || b.Frame != 0 {
				t.Errorf("%s の棒 = %+v、長さ %d で枠なしのはず", tc.label, b, tc.want)
			}
		}
		// 棒は HP と 4 つの能力の 5 本だけで、左端がそろう。
		if len(v.Bars) != 5 {
			t.Fatalf("棒の数 = %d、5 のはず（移動は数字だけ）", len(v.Bars))
		}
		mov := textOf(t, v, "移動 5")
		for _, b := range v.Bars {
			if b.X != v.Bars[0].X {
				t.Errorf("棒の左端がそろっていない: %+v", v.Bars)
			}
			if b.Y >= mov.Y && b.Y < mov.Y+lineHeight {
				t.Errorf("移動の行に棒がある: %+v", b)
			}
		}
	})

	t.Run("TC-FR-UNT-09-08 HP の棒は 120 px の枠に今の HP ÷ 最大 HP の割合で塗り、数字（今/最大）も出す", func(t *testing.T) {
		for _, tc := range []struct{ hp, max, want int }{{13, 26, 60}, {26, 26, 120}, {0, 26, 0}, {1, 3, 40}, {20, 20, 120}, {30, 20, 120}, {5, 0, 0}} {
			if got := HPBarLen(tc.hp, tc.max); got != tc.want {
				t.Errorf("HPBarLen(%d, %d) = %d、%d のはず", tc.hp, tc.max, got, tc.want)
			}
		}
		u := leon()
		u.Stats.HP, u.HP = 26, 13
		v := view(u)
		if b := barOf(t, v, "HP 13/26"); b.Fill != 60 || b.Frame != 120 {
			t.Errorf("HP 13/26 の棒 = %+v、枠 120 に 60 を塗るはず", b)
		}
	})

	t.Run("TC-FR-UNT-09-09 名前の横にユニットの絵（盤面と同じタイル）を 64 px で描く。ユニットがいなければ絵を描かない", func(t *testing.T) {
		v := view(leon())
		if len(v.Icons) == 0 {
			t.Fatal("絵がない")
		}
		pic := v.Icons[0]
		if pic.Size != 64 || pic.Tile != assets.PlayerLord || pic.X != MapSize+infoMargin || pic.Y != infoMargin {
			t.Errorf("ユニットの絵 = %+v、情報欄の左上に PlayerLord を 64 px のはず", pic)
		}
		// 名前とクラスは絵の右に、絵の高さの中に並べる。
		for _, s := range []string{"レオン", "クラス: ロード"} {
			tx := textOf(t, v, s)
			if tx.X < pic.X+pic.Size || tx.Y < pic.Y || tx.Y+lineHeight > pic.Y+pic.Size {
				t.Errorf("%q の位置 (%d, %d) が絵 %+v の右にない", s, tx.X, tx.Y, pic)
			}
		}
		// 能力は絵の下から並べる。
		if hp := textOf(t, v, "HP 20/20"); hp.Y < pic.Y+pic.Size {
			t.Errorf("HP の行 y = %d が絵と重なる", hp.Y)
		}
		// 敵は盤面と同じ敵のタイル、ボスはボスのタイル。
		enemy := tactics.Unit{Name: "ソルジャー", Side: tactics.Enemy, Stats: soldierSts, HP: 18, Weapon: ironLance}
		if got := view(enemy).Icons[0].Tile; got != unitTile(enemy) || got != assets.EnemyLance {
			t.Errorf("敵の絵 = %+v", got)
		}
		enemy.Boss = true
		if got := view(enemy).Icons[0].Tile; got != assets.EnemyBoss {
			t.Errorf("ボスの絵 = %+v", got)
		}
		// ユニットのいないマスと盤面の外では絵を描かない。
		c := newIdleWith(t, example1Setup)
		if v := c.Panel(tactics.Pos{X: 0, Y: 0}, true).View(); len(v.Icons) != 0 || len(v.Bars) != 0 {
			t.Errorf("空きマスの絵 = %+v、棒 = %+v", v.Icons, v.Bars)
		}
		if v := c.Panel(tactics.Pos{X: 0, Y: 0}, false).View(); len(v.Icons) != 0 || len(v.Texts) != 0 {
			t.Errorf("盤面の外 = %+v", v)
		}
	})

	t.Run("TC-FR-UNT-09-10 スキルの名前を黄で 1 行に出し、次の行に効果の文を出す。スキルがなければ「なし」だけ", func(t *testing.T) {
		v := view(leon())
		name := textOf(t, v, "鼓舞")
		label := textOf(t, v, "スキル:")
		desc := textOf(t, v, "　距離 2 以内の味方の命中 +10・回避 +10")
		if name.Style != StyleSkill || label.Style != StyleLabel || desc.Style != StylePlain {
			t.Errorf("色の種類: 名前 %v、見出し %v、効果 %v", name.Style, label.Style, desc.Style)
		}
		if name.Y != label.Y || name.X <= label.X || desc.Y != name.Y+lineHeight {
			t.Errorf("位置: 見出し %+v、名前 %+v、効果 %+v", label, name, desc)
		}
		u := leon()
		u.Skill = tactics.Skill{}
		v = view(u)
		none := textOf(t, v, "なし")
		if none.Style != StylePlain || none.Y != textOf(t, v, "スキル:").Y {
			t.Errorf("スキルなし = %+v", none)
		}
		for _, tx := range v.Texts {
			if tx.Style == StyleSkill || tx.Y > none.Y {
				t.Errorf("スキルなしで出した文 %+v", tx)
			}
		}
	})

	t.Run("TC-FR-UNT-09-11 武器の行に武器の種類の絵を添える。剣・槍・斧・杖は Tiny Dungeon、弓は Tiny Town", func(t *testing.T) {
		sheets, err := assets.Load()
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			typ  tactics.WeaponType
			want assets.Tile
			kind assets.Sheet
		}{
			{tactics.Sword, assets.WeaponSword, assets.Dungeon},
			{tactics.Lance, assets.WeaponLance, assets.Dungeon},
			{tactics.Axe, assets.WeaponAxe, assets.Dungeon},
			{tactics.Bow, assets.WeaponBow, assets.Town},
			{tactics.Staff, assets.WeaponStaff, assets.Dungeon},
		} {
			got, ok := weaponTile(tc.typ)
			if !ok || got != tc.want || got.Sheet != tc.kind {
				t.Errorf("weaponTile(%v) = %+v, %v", tc.typ, got, ok)
			}
			if !got.Rect().In(sheets[got.Sheet].Bounds()) {
				t.Errorf("%v の絵 %+v が画像の外", tc.typ, got)
			}
			// 武器の絵は、武器の名前の行の左に 32 px で描く。
			u := leon()
			u.Weapon.Type = tc.typ
			v := view(u)
			w := textOf(t, v, "武器: 鉄の剣（"+weaponTypeName(tc.typ)+"）")
			if len(v.Icons) != 2 || v.Icons[1].Tile != tc.want || v.Icons[1].Size != 32 || v.Icons[1].Y != w.Y || v.Icons[1].X+v.Icons[1].Size > w.X {
				t.Errorf("%v の武器の絵 = %+v、文 %+v の左に 32 px のはず", tc.typ, v.Icons, w)
			}
		}
		// 5 つの絵はすべて違う。
		seen := map[assets.Tile]bool{}
		for _, tl := range []assets.Tile{assets.WeaponSword, assets.WeaponLance, assets.WeaponAxe, assets.WeaponBow, assets.WeaponStaff} {
			seen[tl] = true
		}
		if len(seen) != 5 {
			t.Errorf("武器の絵が重なっている")
		}
		// 知らない種類なら武器の絵を描かない（文は出す）。
		u := leon()
		u.Weapon.Type = tactics.WeaponType(99)
		if _, ok := weaponTile(u.Weapon.Type); ok {
			t.Errorf("知らない種類に絵がある")
		}
		if v := view(u); len(v.Icons) != 1 {
			t.Errorf("知らない種類の絵 = %+v、ユニットの絵だけのはず", v.Icons)
		}
	})

	t.Run("TC-FR-UNT-09-12 予測の表（10 行）と並べても、情報欄（幅 320 px、高さ 576 px）に収まる", func(t *testing.T) {
		c := newIdleWith(t, example4Setup)
		target(t, c, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2})
		// スキルと手持ちのあるガルドにカーソルを合わせる。
		p := c.Panel(tactics.Pos{X: 2, Y: 2}, true)
		if len(p.Forecast) != 10 || p.Unit == nil || p.Unit.Name != "ガルド" {
			t.Fatalf("前提: 予測 %d 行、ユニット %+v", len(p.Forecast), p.Unit)
		}
		// レベル・経験を含め、同じ10行の表で帯なし・撃破・危険を調べる（DES-UI-16）。
		p.Unit.Level, p.Unit.Exp = 2, 40
		for _, emphasis := range []ForecastEmphasis{ForecastNormal, ForecastDefeat, ForecastDanger} {
			p.Emphasis = emphasis
			shift := 0
			if emphasis != ForecastNormal {
				shift = 44
			}
			v := p.View()
			for _, b := range c.RewindView().Buttons {
				if !b.Rect.In(image.Rect(0, MapSize, MapSize, ScreenHeight)) {
					t.Fatal("巻き戻しボタンが案内欄の外にある", b)
				}
				for i, line := range c.Guide() {
					y := MapSize + 2 + i*lineHeight
					if b.Rect.Overlaps(image.Rect(8, y, 8+textWidth(line), y+lineHeight)) {
						t.Fatal("巻き戻しボタンが案内文を隠す", line)
					}
				}
				for _, tx := range c.RewindView().Texts {
					if b.Rect.Overlaps(image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)) {
						t.Fatal("巻き戻しボタンが操作案内を隠す", tx)
					}
				}
				for _, tx := range v.Texts {
					if b.Rect.Overlaps(image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)) {
						t.Fatal("巻き戻しボタンが情報を隠す", tx)
					}
				}
			}
			textOf(t, v, "Lv 2　経験 40")
			if v.Bottom > ScreenHeight {
				t.Errorf("一番下 = %d、%d を超える", v.Bottom, ScreenHeight)
			}
			// 予測（見出し・武器の絵 32 px・表 10 行。空き行なし）の下 y = 8 + 16 + 32 + 160 = 216 から並べる（帯ありはさらに 44 px 下）。
			pic := portraitOf(t, v)
			if pic.Y != 216+shift || forecastBottom(10)+shift != pic.Y {
				t.Errorf("ユニットの絵の y = %d、予測の表の下 %d のはず", pic.Y, 216+shift)
			}
			for _, tx := range v.Texts {
				scale := 1
				if tx.Big {
					scale = 2
				}
				if tx.X < MapSize || tx.X+scale*textWidth(tx.Text) > ScreenWidth || tx.Y+scale*lineHeight > ScreenHeight {
					t.Errorf("文 %q（%d, %d）が情報欄からはみ出す", tx.Text, tx.X, tx.Y)
				}
			}
			for _, b := range v.Bars {
				if b.X < MapSize || b.X+max(b.Fill, b.Frame) > ScreenWidth || b.Y+barHeight > ScreenHeight {
					t.Errorf("棒 %+v が情報欄からはみ出す", b)
				}
			}
			for _, ic := range v.Icons {
				if ic.X < MapSize || ic.X+ic.Size > ScreenWidth || ic.Y+ic.Size > ScreenHeight {
					t.Errorf("絵 %+v が情報欄からはみ出す", ic)
				}
			}
		}
	})
}

package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// weaponRowTexts は、「能力」のページの武器の行（移動の行より下で、地形の行より上）の文を返す（DES-IT06 12.2 節）。
func weaponRowTexts(v InfoView) []InfoText {
	moveY := -1
	var out []InfoText
	for _, tx := range v.Texts {
		switch {
		case strings.HasPrefix(tx.Text, "移動 "):
			moveY = tx.Y
		case strings.HasPrefix(tx.Text, "地形: "):
			return out
		case moveY >= 0 && tx.Y > moveY:
			out = append(out, tx)
		}
	}
	return out
}

// weaponRowIcons は、武器の行の絵（タイルと同じ 16 px の絵）を返す（DES-IT06 12.2 節）。
func weaponRowIcons(v InfoView) []InfoIcon {
	var out []InfoIcon
	for _, ic := range v.Icons {
		if ic.Size == assets.TileSize {
			out = append(out, ic)
		}
	}
	return out
}

// 「能力」のページの武器の行（DES-IT06 12.2 節、E-96・E-97、FR-UNT-09）を確かめる。期待する位置は、欄の左端 x = 520 から
// 2 列（1 列 152 px）に分け、列の左端に装備の印「E」、10 px 右に 16 px の種類の絵、30 px 右に名前を置く配置から手で求めた。
func TestInfoWeaponRows(t *testing.T) {
	staff := tactics.Weapon{Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, MinRange: 1, MaxRange: 1, Uses: 15, MaxUses: 15}
	fire := tactics.Weapon{Name: "ファイア", Type: tactics.Magic, Might: 5, Hit: 90, MinRange: 1, MaxRange: 2}
	killer := tactics.Weapon{Name: "アーマーキラー", Type: tactics.Lance, Might: 5, Hit: 80, MinRange: 1, MaxRange: 1}
	unit := func(weapons []tactics.Weapon, equipped int) tactics.Unit {
		return tactics.Unit{
			Name: "レオン", ClassName: "ロード", Side: tactics.Player, Lord: true, Alive: true, Level: 3, Exp: 10,
			Stats: leonStats, HP: 20, Weapons: weapons, Equipped: equipped, Skill: inspire,
		}
	}
	plain := tactics.Terrain{Name: "平地"}
	x := MapSize + infoMargin

	t.Run("TC-FR-UNT-09-22 能力のページの下に、持っている武器を種類の絵と名前の短い行で並べ、装備中の武器に印を付ける。5 本でも欄・配置のボタン・印に重ならない", func(t *testing.T) {
		five := []tactics.Weapon{ironSword, killer, ironAxe, ironBow, fire}
		for _, tc := range []struct {
			name     string
			weapons  []tactics.Weapon
			equipped int
			// want は武器の行の文を上から（同じ行は左から）並べたもの。
			want  []string
			tiles []assets.Tile
		}{
			{"武器 1 本", []tactics.Weapon{ironSword}, 0, []string{"E", "鉄の剣"}, []assets.Tile{assets.WeaponSword}},
			{"杖 1 本（回復役も同じように絵で並ぶ）", []tactics.Weapon{staff}, 0, []string{"E", "癒やしの杖"}, []assets.Tile{assets.WeaponStaff}},
			{"武器と杖の 2 本で、2 本目を装備", []tactics.Weapon{fire, staff}, 1, []string{"ファイア", "E", "癒やしの杖"}, []assets.Tile{assets.WeaponMagic, assets.WeaponStaff}},
			{"武器 5 本で、3 本目を装備", five, 2,
				[]string{"鉄の剣", "アーマーキラー", "E", "鉄の斧", "鉄の弓", "ファイア"},
				[]assets.Tile{assets.WeaponSword, assets.WeaponLance, assets.WeaponAxe, assets.WeaponBow, assets.WeaponMagic}},
		} {
			u := unit(tc.weapons, tc.equipped)
			// 戦闘の情報欄（地形とページの印あり）。
			v := Panel{Unit: &u, Terrain: &plain, Tabs: true}.View()
			texts := weaponRowTexts(v)
			got := make([]string, len(texts))
			for i, tx := range texts {
				got[i] = tx.Text
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Errorf("%s: 武器の行 = %q、%q のはず", tc.name, got, tc.want)
			}
			icons := weaponRowIcons(v)
			if len(icons) != len(tc.weapons) {
				t.Fatalf("%s: 武器の絵 %d 枚、%d 枚のはず", tc.name, len(icons), len(tc.weapons))
			}
			// 移動の行（y = 204）の下 4 px（y = 220 + 4）から、1 行 16 px で 2 本ずつ並べる。
			top := textOf(t, v, "移動 5").Y + lineHeight + blockGap
			for k, ic := range icons {
				cx, cy := x+(k%2)*152, top+(k/2)*lineHeight
				if ic.X != cx+10 || ic.Y != cy || ic.Tile != tc.tiles[k] {
					t.Errorf("%s: %d 本目の絵 %+v、(%d, %d) の %v のはず", tc.name, k+1, ic, cx+10, cy, tc.tiles[k])
				}
				name := textOf(t, v, tc.weapons[k].Name)
				if name.X != cx+30 || name.Y != cy {
					t.Errorf("%s: %d 本目の名前 %+v、(%d, %d) のはず", tc.name, k+1, name, cx+30, cy)
				}
				// 名前は列の右端の 4 px 手前までに収め、右の列とくっつけない。
				if r := rectOfText(name.X, name.Y, name.Text, false); r.Max.X > cx+152-4 {
					t.Errorf("%s: %d 本目の名前 %v が列（右端 %d）からはみ出す", tc.name, k+1, r, cx+152)
				}
			}
			mark := textOf(t, v, "E")
			if eq := tc.equipped; mark.X != x+(eq%2)*152 || mark.Y != top+(eq/2)*lineHeight || mark.Style != StyleTitle {
				t.Errorf("%s: 装備の印 %+v が %d 本目の行にない", tc.name, mark, eq+1)
			}
			// 絵と文は互いに重ならず、移動の行とも重ならない。
			var rects []image.Rectangle
			for _, ic := range icons {
				rects = append(rects, image.Rect(ic.X, ic.Y, ic.X+ic.Size, ic.Y+ic.Size))
			}
			for _, tx := range texts {
				rects = append(rects, rectOfText(tx.X, tx.Y, tx.Text, false))
			}
			move := textOf(t, v, "移動 5")
			rects = append(rects, rectOfText(move.X, move.Y, move.Text, false))
			for i, a := range rects {
				for _, b := range rects[i+1:] {
					if a.Overlaps(b) {
						t.Errorf("%s: %v と %v が重なる", tc.name, a, b)
					}
				}
			}
			// 地形の行は武器の行の下にあり、すべて情報欄（y < 360）の中、ページの印（y = 336〜）の上に収まる。
			terrain := textOf(t, v, "地形: 平地")
			if last := top + (len(tc.weapons)+1)/2*lineHeight; terrain.Y < last {
				t.Errorf("%s: 地形の行 y = %d が武器の行（下端 %d）に重なる", tc.name, terrain.Y, last)
			}
			if v.Bottom > infoTabsBottom {
				t.Errorf("%s: 一番下 %d が印の上端 %d を超える", tc.name, v.Bottom, infoTabsBottom)
			}
			checkInfoArea(t, tc.name, v)
			// 配置の画面（地形なし）では、ボタン（y = 296〜）より上に収まる。
			if d := (Panel{Unit: &u, Tabs: true}).View(); d.Bottom > deployButtonTop {
				t.Errorf("%s: 配置の情報の一番下 %d がボタンの上端 %d を超える", tc.name, d.Bottom, deployButtonTop)
			}
			// 詳しい値とスキルは「装備・スキル」のページのままで、そちらには武器の行を足さない。
			e := Panel{Unit: &u, Terrain: &plain, Page: PageEquip, Tabs: true}.View()
			if len(weaponRowIcons(e)) != 0 {
				t.Errorf("%s: 装備・スキルのページに武器の行がある", tc.name)
			}
			for _, r := range rows(e) {
				if r == "E" || strings.HasPrefix(r, "E ") {
					t.Errorf("%s: 装備・スキルのページに装備の印 %q", tc.name, r)
				}
			}
		}

		// 武器を持たないユニットには、武器の行を足さない（移動のすぐあとに地形の空き行）。
		none := unit(nil, 0)
		if v := (Panel{Unit: &none, Terrain: &plain}).View(); len(weaponRowTexts(v)) != 0 || len(weaponRowIcons(v)) != 0 {
			t.Errorf("武器なしで武器の行 %q", rows(v))
		}

		// 列の幅を超える長い名前は「…」で省き、全文は装備・スキルのページで読める。
		long := tactics.Weapon{Name: "とてもとても長い名前の剣の武器", Type: tactics.Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
		u := unit([]tactics.Weapon{ironSword, long}, 0)
		v := Panel{Unit: &u, Terrain: &plain}.View()
		texts := weaponRowTexts(v)
		got := texts[len(texts)-1]
		if !strings.HasSuffix(got.Text, "…") || rectOfText(got.X, got.Y, got.Text, false).Max.X > x+2*152-4 {
			t.Errorf("長い名前 %+v", got)
		}
		u.Equipped = 1
		if !strings.Contains(strings.Join(equipRows(Panel{Unit: &u}), "|"), long.Name) {
			t.Errorf("装備・スキルのページに全文がない: %q", equipRows(Panel{Unit: &u}))
		}

		// 回復の予測を開いている間は、予測を優先して武器の行を出さず、予測とユニットのページを欄に収める。
		u = unit(five, 0)
		h := Panel{Heal: &tactics.HealForecast{Healer: "リナ", Target: "レオン", HP: 10, MaxHP: 20, Amount: 10, After: 20, Uses: 15, MaxUses: 15},
			Unit: &u, Terrain: &plain, Tabs: true}.View()
		if len(weaponRowIcons(h)) != 0 || len(weaponRowTexts(h)) != 0 {
			t.Errorf("回復の予測の間に武器の行 %q", rows(h))
		}
		checkInfoArea(t, "回復の予測", h)
		// 攻撃の予測を開いている間は、今までどおり予測だけを出す。
		a := Panel{Forecast: []ForecastRow{{"", "レオン", "ボス"}}, Unit: &u, Terrain: &plain, Tabs: true}.View()
		if len(weaponRowIcons(a)) != 0 || strings.Contains(strings.Join(rows(a), "|"), "鉄の斧") {
			t.Errorf("攻撃の予測の間に武器の行 %q", rows(a))
		}
	})

	t.Run("TC-FR-UNT-09-23 敵のレベルを能力のページの名前・クラスの下に出す。敵には経験を出さない", func(t *testing.T) {
		enemy := func(level int) tactics.Unit {
			return tactics.Unit{Name: "ファイター", ClassName: "ファイター", Side: tactics.Enemy, Alive: true, Level: level,
				Stats: tactics.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5}, HP: 24, Weapons: []tactics.Weapon{ironAxe}}
		}
		for _, tc := range []struct {
			name string
			u    tactics.Unit
			// want は名前・クラスの下の行（なければ空）。
			want string
		}{
			{"敵 Lv 5", enemy(5), "Lv 5"},
			{"敵 Lv 20（経験の「―」も出さない）", enemy(20), "Lv 20"},
			{"自軍 Lv 3・経験 10 は今までどおり", unit([]tactics.Weapon{ironSword}, 0), "Lv 3　経験 10"},
			{"レベル 0（試験の自軍）は出さない", func() tactics.Unit { u := unit([]tactics.Weapon{ironSword}, 0); u.Level = 0; return u }(), ""},
		} {
			v := Panel{Unit: &tc.u, Terrain: &plain, Tabs: true}.View()
			r := rows(v)
			got := ""
			if !strings.HasPrefix(r[2], "HP ") {
				got = r[2]
			}
			if got != tc.want || !strings.HasPrefix(r[1], "クラス: ") {
				t.Errorf("%s: 名前・クラスの下 = %q、%q のはず（%q）", tc.name, got, tc.want, r)
			}
			if tc.want == "" {
				continue
			}
			// 3 行（名前・クラス・レベル）を 64 px の絵の横の中央に並べる。
			p, lv := portraitOf(t, v), textOf(t, v, tc.want)
			if lv.X != x+portraitTextX || lv.Y != p.Y+(PortraitSize-3*lineHeight)/2+2*lineHeight {
				t.Errorf("%s: レベルの行 %+v（絵 %+v）", tc.name, lv, p)
			}
			checkInfoArea(t, tc.name, v)
		}
	})
}

package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 予測の見せ方（DES-UI-07、FR-BTL-08、UC-03）を確かめる。期待値は設計書 5.6 節の計算例（「後」の HP）と 3.5 節の相性の表から求める。

// example2Setup は設計書 3.5 節の計算例 2 の盤面。7 × 7 の平地。defenderHP は剣士の最大 HP（10 章では 17）。
//
//	0: レオン (0, 6)。ブランから遠く、鼓舞は効かない
//	1: ブラン (2, 2)。ファイター、鉄の斧、怒り（HP 26 は最大の半分を超えるので必殺にならない）
//	2: 剣士 (3, 2)。鉄の剣、待ち構える
//	3: ボス (6, 0)
//
// ブランが今いるマスから攻撃すると、ブランは 13 ダメージ・命中率 45・1 回、剣士は 7 ダメージ・2 回（剣士の追撃）。
func example2Setup(defenderHP int) func() tactics.Setup {
	return func() tactics.Setup {
		return tactics.Setup{
			Terrain: field(7, 7),
			Units: []tactics.UnitSetup{
				{Name: "レオン", ClassName: "ロード", Side: tactics.Player, Stats: leonStats, Pos: tactics.Pos{X: 0, Y: 6}, Weapon: ironSword, Skill: inspire, Lord: true},
				{Name: "ブラン", ClassName: "ファイター", Side: tactics.Player, Stats: tactics.Stats{HP: 26, Str: 8, Skl: 4, Spd: 5, Def: 3, Mov: 5}, Pos: tactics.Pos{X: 2, Y: 2}, Weapon: ironAxe, Skill: wrath},
				{Name: "剣士", ClassName: "剣士", Side: tactics.Enemy, Stats: tactics.Stats{HP: defenderHP, Str: 4, Skl: 8, Spd: 9, Def: 2, Mov: 5}, Pos: tactics.Pos{X: 3, Y: 2}, Weapon: ironSword, AI: tactics.Hold},
				{Name: "ボス", Side: tactics.Enemy, Stats: soldierSts, Pos: tactics.Pos{X: 6, Y: 0}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
			},
		}
	}
}

// example1HP は、計算例 1 の盤面で、レオンの最大 HP（＝今の HP）を hp にしたもの。
func example1HP(hp int) func() tactics.Setup {
	return func() tactics.Setup {
		s := example1Setup()
		s.Units[0].Stats.HP = hp
		return s
	}
}

// archerSetup は、レオンが隣のアーチャー（弓は隣に反撃できない）を攻撃する盤面。7 × 7 の平地。
//
//	0: レオン (2, 3)。鉄の剣
//	1: アーチャー (3, 3)。10 章のアーチャー（HP 17、力 5、技 7、速さ 6、守備 3）、鉄の弓、待ち構える
//	2: ボス (6, 6)
//
// 相性なし。レオンは攻撃力 6 ＋ 5 = 11、ダメージ 11 − 3 = 8。命中 16 ＋ 90 = 106、回避 12 → 命中率 94。速さの差 3 で追撃なし。
// アーチャーは回数 0 だが、Forecast はダメージ 5 ＋ 6 − 4 = 7、命中率 14 ＋ 85 − 18 = 81 を返す。
func archerSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: field(7, 7),
		Units: []tactics.UnitSetup{
			{Name: "レオン", ClassName: "ロード", Side: tactics.Player, Stats: leonStats, Pos: tactics.Pos{X: 2, Y: 3}, Weapon: ironSword, Skill: inspire, Lord: true},
			{Name: "アーチャー", ClassName: "アーチャー", Side: tactics.Enemy, Stats: tactics.Stats{HP: 17, Str: 5, Skl: 7, Spd: 6, Def: 3, Mov: 5}, Pos: tactics.Pos{X: 3, Y: 3}, Weapon: ironBow, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Stats: soldierSts, Pos: tactics.Pos{X: 6, Y: 6}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
		},
	}
}

// 予測の列の左端（画面の座標）と、武器の絵の行の y（DES-UI-07）。
const (
	attackColX = MapSize + infoMargin + forecastAttackX
	defendColX = MapSize + infoMargin + forecastDefendX
	iconRowY   = infoMargin + lineHeight
)

// textsAt は、(x, y) に置いた文を返す。
func textsAt(v InfoView, x, y int) []InfoText {
	var out []InfoText
	for _, tx := range v.Texts {
		if tx.X == x && tx.Y == y {
			out = append(out, tx)
		}
	}
	return out
}

// sideMarks は、列 colX の武器の絵の右に置いた相性の印（2 倍）と、その右の「撃破」「危険」（絵の高さの真ん中）の文を返す。
// なければ空の InfoText。
func sideMarks(t *testing.T, v InfoView, colX int) (mark, badge InfoText) {
	t.Helper()
	for i, at := range [][2]int{{colX + weaponTextX, iconRowY}, {colX + forecastBadgeX, iconRowY + (WeaponIconSize-lineHeight)/2}} {
		x, y := at[0], at[1]
		got := textsAt(v, x, y)
		if len(got) > 1 {
			t.Fatalf("(%d, %d) に文が %d 個: %+v", x, y, len(got), got)
		}
		if len(got) == 1 {
			if i == 0 {
				mark = got[0]
			} else {
				badge = got[0]
			}
		}
	}
	return mark, badge
}

func TestForecastVisual(t *testing.T) {
	t.Run("TC-FR-BTL-08-09 「後」の HP は設計書 5.6 節の計算例 5 件のとおり。後が 0 なら守る側に「撃破」、攻撃する側に「危険」", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			setup          func() tactics.Setup
			unit, target   tactics.Pos
			wantHP         ForecastRow
			wantAfter      [2]int
			attackerBadge  string
			defenderBadge  string
			attackerWeapon tactics.WeaponType
		}{
			{"例 1（レオン → 森のソルジャー）", example1Setup, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3},
				ForecastRow{"HP", "20 → 11", "18 → 8"}, [2]int{11, 8}, "", "", tactics.Sword},
			{"例 1 でレオンの HP が 9（レオンが倒れるので追撃はない）", example1HP(9), tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3},
				ForecastRow{"HP", "9 → 0", "18 → 13"}, [2]int{0, 13}, "危険", "", tactics.Sword},
			{"例 2（ブラン → 剣士。剣士の反撃と追撃）", example2Setup(17), tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2},
				ForecastRow{"HP", "26 → 12", "17 → 4"}, [2]int{12, 4}, "", "", tactics.Axe},
			{"例 2 で剣士の HP が 10", example2Setup(10), tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2},
				ForecastRow{"HP", "26 → 26", "10 → 0"}, [2]int{26, 0}, "", "撃破", tactics.Axe},
			{"例 4（ガルド → ファイター。大盾は数えない）", example4Setup, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2},
				ForecastRow{"HP", "24 → 18", "24 → 15"}, [2]int{18, 15}, "", "", tactics.Lance},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := newIdleWith(t, tc.setup)
				target(t, c, tc.unit, tc.target)
				p := c.Panel(tc.target, true)
				if len(p.Forecast) < 2 || p.Forecast[1] != tc.wantHP {
					t.Fatalf("HP の行 = %q、%q のはず", p.Forecast, tc.wantHP)
				}
				if p.ForecastAfter != tc.wantAfter || p.ForecastWeapons[0] != tc.attackerWeapon {
					t.Errorf("後の HP = %v、武器 %v", p.ForecastAfter, p.ForecastWeapons)
				}
				v := p.View()
				for _, side := range []struct {
					col  int
					want string
				}{{attackColX, tc.attackerBadge}, {defendColX, tc.defenderBadge}} {
					_, badge := sideMarks(t, v, side.col)
					if badge.Text != side.want || (side.want != "" && badge.Style != StyleAlert) {
						t.Errorf("列 %d の「撃破」「危険」 = %+v、%q のはず", side.col, badge, side.want)
					}
				}
			})
		}
	})

	t.Run("TC-FR-BTL-08-10 「後」は 1 撃目・反撃・追撃の順に引き、0 で止め、どちらかが 0 になったら止める", func(t *testing.T) {
		side := func(hp, dmg, count int) tactics.ForecastSide {
			return tactics.ForecastSide{HP: hp, Damage: dmg, HitRate: 50, Count: count}
		}
		for _, tc := range []struct {
			name     string
			a, d     tactics.ForecastSide
			wantA    int
			wantD    int
			wantRows [2]string
		}{
			{"反撃なしで攻撃する側が追撃", side(20, 5, 2), side(18, 9, 0), 20, 8, [2]string{"20 → 20", "18 → 8"}},
			{"1 撃目で倒せば反撃しない（ダメージが HP を超えても 0）", side(5, 30, 1), side(10, 9, 1), 5, 0, [2]string{"5 → 5", "10 → 0"}},
			{"守る側の追撃で攻撃する側が倒れる", side(14, 1, 1), side(20, 7, 2), 0, 19, [2]string{"14 → 0", "20 → 19"}},
			{"反撃で倒れれば追撃しない（追撃は反撃のあと）", side(9, 5, 2), side(18, 9, 1), 0, 13, [2]string{"9 → 0", "18 → 13"}},
			{"ダメージ 0 なら減らない", side(20, 0, 1), side(20, 0, 1), 20, 20, [2]string{"20 → 20", "20 → 20"}},
		} {
			f := tactics.Forecast{Attacker: tc.a, Defender: tc.d}
			a, d := ForecastAfter(f)
			if a != tc.wantA || d != tc.wantD {
				t.Errorf("%s: 後 = %d, %d、%d, %d のはず", tc.name, a, d, tc.wantA, tc.wantD)
			}
			if r := ForecastRows(f, false, false)[1]; r.Attacker != tc.wantRows[0] || r.Defender != tc.wantRows[1] {
				t.Errorf("%s: HP の行 = %q", tc.name, r)
			}
		}
	})

	t.Run("TC-FR-BTL-08-11 両側に武器の種類の絵と相性の印。有利な側に緑の ▲、不利な側に赤の ▼、弓（相性なし）は出さない", func(t *testing.T) {
		types := []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe, tactics.Bow}
		want := map[[2]tactics.WeaponType]int{
			{tactics.Sword, tactics.Axe}: 1, {tactics.Axe, tactics.Lance}: 1, {tactics.Lance, tactics.Sword}: 1,
			{tactics.Axe, tactics.Sword}: -1, {tactics.Lance, tactics.Axe}: -1, {tactics.Sword, tactics.Lance}: -1,
		}
		for _, a := range types {
			for _, d := range types {
				if got := Triangle(a, d); got != want[[2]tactics.WeaponType{a, d}] {
					t.Errorf("Triangle(%v, %v) = %d", a, d, got)
				}
			}
		}
		for _, tc := range []struct {
			name         string
			setup        func() tactics.Setup
			unit, target tactics.Pos
			weapons      [2]assets.Tile
			marks        [2]string
			styles       [2]InfoStyle
		}{
			{"有利（剣士の剣 → ブランの斧）", wrathSetup, tactics.Pos{X: 4, Y: 2}, tactics.Pos{X: 3, Y: 2},
				[2]assets.Tile{assets.WeaponSword, assets.WeaponAxe}, [2]string{"▲", "▼"}, [2]InfoStyle{StyleGood, StyleBad}},
			{"不利（計算例 1。レオンの剣 → ソルジャーの槍）", example1Setup, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3},
				[2]assets.Tile{assets.WeaponSword, assets.WeaponLance}, [2]string{"▼", "▲"}, [2]InfoStyle{StyleBad, StyleGood}},
			{"不利（計算例 4。ガルドの槍 → ファイターの斧）", example4Setup, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2},
				[2]assets.Tile{assets.WeaponLance, assets.WeaponAxe}, [2]string{"▼", "▲"}, [2]InfoStyle{StyleBad, StyleGood}},
			{"相性なし（弓兵の弓 → ブランの斧）", wrathSetup, tactics.Pos{X: 3, Y: 4}, tactics.Pos{X: 3, Y: 2},
				[2]assets.Tile{assets.WeaponBow, assets.WeaponAxe}, [2]string{"", ""}, [2]InfoStyle{}},
		} {
			c := newIdleWith(t, tc.setup)
			target(t, c, tc.unit, tc.target)
			v := c.Panel(tc.target, true).View()
			// 武器の絵は、両側の列の頭（見出しの下）に 32 px で描く。
			icons := []InfoIcon{
				{X: attackColX, Y: iconRowY, Size: WeaponIconSize, Tile: tc.weapons[0]},
				{X: defendColX, Y: iconRowY, Size: WeaponIconSize, Tile: tc.weapons[1]},
			}
			if len(v.Icons) < 2 || !reflect.DeepEqual(v.Icons[:2], icons) {
				t.Errorf("%s: 武器の絵 = %+v\nのはず %+v", tc.name, v.Icons, icons)
			}
			for i, col := range []int{attackColX, defendColX} {
				mark, _ := sideMarks(t, v, col)
				if mark.Text != tc.marks[i] || (tc.marks[i] != "" && (mark.Style != tc.styles[i] || !mark.Big)) {
					t.Errorf("%s: 列 %d の印 = %+v、%q（%v）のはず", tc.name, i, mark, tc.marks[i], tc.styles[i])
				}
			}
		}
	})

	t.Run("TC-FR-BTL-08-12 反撃できない側（回数 0）は、ダメージ・命中率・必殺を「―」、回数は「反撃なし」。HP は今 → 後のまま出す", func(t *testing.T) {
		c := newIdleWith(t, archerSetup)
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		_, dest, tg := c.Selection()
		f, err := c.Battle().Forecast(tactics.Command{Unit: 0, To: dest, Action: tactics.Attack, Target: tg})
		// Forecast は回数 0 の側にもダメージと命中率を返す（それを「―」にするのは画面の側）。
		if err != nil || f.Defender.Count != 0 || f.Defender.Damage != 7 || f.Defender.HitRate != 81 {
			t.Fatalf("前提: 守る側の予測 %+v, %v", f.Defender, err)
		}
		want := []ForecastRow{
			{"", "レオン", "アーチャー"},
			{"HP", "20 → 20", "17 → 9"},
			{"ダメージ", "8", "―"},
			{"命中率", "94%", "―"},
			{"回数", "×1", "反撃なし"},
			{"必殺", "―", "―"},
			{"スキル", "―", "―"},
		}
		if got := c.Panel(tactics.Pos{X: 3, Y: 3}, true).Forecast; !reflect.DeepEqual(got, want) {
			t.Errorf("予測 = %q\nのはず %q", got, want)
		}
		// 攻撃する側の回数が 0 になることはないが、表は側ごとに同じ規則で作る。
		rows := ForecastRows(tactics.Forecast{
			Attacker: tactics.ForecastSide{Name: "A", HP: 10, Damage: 3, HitRate: 70, Count: 0, Crit: true},
			Defender: tactics.ForecastSide{Name: "D", HP: 10, Damage: 4, HitRate: 60, Count: 1},
		}, false, false)
		for i, w := range []ForecastRow{{"ダメージ", "―", "4"}, {"命中率", "―", "60%"}, {"回数", "反撃なし", "×1"}, {"必殺", "―", "―"}} {
			if rows[i+2] != w {
				t.Errorf("回数 0 の攻撃する側: %q、%q のはず", rows[i+2], w)
			}
		}
	})

	t.Run("TC-FR-BTL-08-13 選んだ相手のマスの上に「ダメージ×回数 命中率%」。回数 1 なら「×1」を省く", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			setup        func() tactics.Setup
			unit, target tactics.Pos
			want         string
		}{
			{"計算例 1（追撃 ×2）", example1Setup, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3}, "5×2 63%"},
			{"計算例 4（1 回）", example4Setup, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2}, "9 73%"},
			{"反撃なしの相手（1 回）", archerSetup, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3}, "8 94%"},
		} {
			c := newIdleWith(t, tc.setup)
			target(t, c, tc.unit, tc.target)
			s, p, ok := c.TargetLabel()
			if !ok || s != tc.want || p != tc.target {
				t.Errorf("%s: TargetLabel = %q, %v, %v、%q, %v のはず", tc.name, s, p, ok, tc.want, tc.target)
			}
		}
	})

	t.Run("TC-FR-BTL-08-14 盤面の上の文字は、相手を選ぶ前・取り消して Moved に戻ったあと・Idle では出さない", func(t *testing.T) {
		c := newIdleWith(t, example1Setup)
		if _, _, ok := c.TargetLabel(); ok {
			t.Errorf("Idle で出した")
		}
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.Click(tactics.Pos{X: 2, Y: 3})
		c.ChooseAction(tactics.Attack)
		if _, _, tg := c.Selection(); c.State() != Targeting || tg >= 0 {
			t.Fatalf("前提: 状態 %v、相手 %d", c.State(), tg)
		}
		if _, _, ok := c.TargetLabel(); ok {
			t.Errorf("相手を選ぶ前に出した")
		}
		c.Click(tactics.Pos{X: 3, Y: 3})
		if _, _, ok := c.TargetLabel(); !ok {
			t.Fatalf("相手を選んだのに出さない")
		}
		c.Cancel()
		if _, _, ok := c.TargetLabel(); ok || c.State() != Moved {
			t.Errorf("取り消したあと（%v）に出した", c.State())
		}
	})

	t.Run("TC-FR-BTL-08-15 予測の文は列からはみ出さず、絵・印・「撃破」「危険」と重ならない", func(t *testing.T) {
		for _, tc := range []struct {
			setup        func() tactics.Setup
			unit, target tactics.Pos
		}{
			{example1HP(9), tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3}},
			{example2Setup(10), tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2}},
			{example4Setup, tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2}},
			{wrathSetup, tactics.Pos{X: 4, Y: 2}, tactics.Pos{X: 3, Y: 2}},
		} {
			c := newIdleWith(t, tc.setup)
			target(t, c, tc.unit, tc.target)
			p := c.Panel(tc.target, true)
			v := p.View()
			bottom := forecastBottom(len(p.Forecast))
			for _, tx := range v.Texts {
				if tx.Y >= bottom {
					continue
				}
				right := tx.X + textWidth(tx.Text)
				if tx.Big {
					right = tx.X + 2*textWidth(tx.Text)
				}
				switch {
				case tx.X < attackColX && right > attackColX:
					t.Errorf("項目の名前 %q が攻撃する側の列にはみ出す", tx.Text)
				case tx.X >= attackColX && tx.X < defendColX && right > defendColX:
					t.Errorf("攻撃する側の %q（右端 %d）が守る側の列にはみ出す", tx.Text, right)
				case right > ScreenWidth-infoMargin:
					t.Errorf("守る側の %q（右端 %d）が情報欄からはみ出す", tx.Text, right)
				}
				// 絵の右に置く印（2 倍）と「撃破」「危険」は、絵とも互いとも重ならない。表の文は絵より下に置く。
				if tx.Big {
					for _, o := range v.Texts {
						if o.Style == StyleAlert && o.Y < tx.Y+2*lineHeight && o.X < right && o.X+textWidth(o.Text) > tx.X {
							t.Errorf("印 %+v と %+v が重なる", tx, o)
						}
					}
				}
				for _, ic := range v.Icons[:2] {
					if tx.X < ic.X+ic.Size && right > ic.X && tx.Y < ic.Y+ic.Size && tx.Y+lineHeight > ic.Y {
						t.Errorf("文 %q（%d, %d）が絵 %+v と重なる", tx.Text, tx.X, tx.Y, ic)
					}
				}
			}
		}
	})
}

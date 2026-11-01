package ui

import (
	"image"
	"os"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// cavalierCatalog は、騎兵の絵を場面ごとに確かめる目録（DES-UI-22）。
// 騎兵（エルザと 2 章の加入者）とアーマーは同じ槍・同じ移動力にし、能力値・名前・武器では見分けられないようにする。
// 1 章は主人公がボスの騎兵を 1 撃で倒せる。
func cavalierCatalog() *defs.Catalog {
	chapters := []defs.ChapterDef{{ID: "c1", Name: "街道", Map: "m"}, {ID: "c2", Name: "谷", Map: "m", Joins: []string{"luca"}}}
	return defs.NewCatalog(defs.GameDef{Party: []string{"hero", "elsa", "gard"}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{
			{ID: "lord", Name: "ロード", WeaponTypes: []string{"sword"}, Base: defs.Stats{HP: 20, Str: 20, Skl: 100, Def: 10, Mov: 2}},
			{ID: "cavalier", Name: "騎兵", WeaponTypes: []string{"lance"}, Base: defs.Stats{HP: 1, Str: 1, Skl: 1, Mov: 2}},
			{ID: "armor", Name: "アーマー", WeaponTypes: []string{"lance"}, Base: defs.Stats{HP: 1, Str: 1, Skl: 1, Mov: 2}},
		},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", Hit: 100, MinRange: 1, MaxRange: 1}, {ID: "l", Name: "槍", Type: "lance", Hit: 100, MinRange: 1, MaxRange: 1}},
		nil, nil,
		[]defs.CharacterDef{
			{ID: "hero", Name: "主人公", Class: "lord", Weapons: []string{"s"}, Lord: true, Level: 1},
			{ID: "elsa", Name: "エルザ", Class: "cavalier", Weapons: []string{"l"}, Level: 2},
			{ID: "gard", Name: "ガルド", Class: "armor", Weapons: []string{"l"}, Level: 2},
			{ID: "luca", Name: "ルカ", Class: "cavalier", Weapons: []string{"l"}, Level: 5, Bonus: defs.Stats{HP: 9, Str: 4, Def: 3}},
		},
		[]defs.MapDef{{ID: "m", Rows: []string{"....", "....", "...."},
			Slots: []defs.Slot{{X: 0, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: 2}},
			Enemies: []defs.EnemyPlacement{
				{Level: 1, Name: "隊長", Class: "cavalier", Weapons: []string{"l"}, Boss: true, AI: "hold", X: 1, Y: 0},
				{Level: 1, Class: "cavalier", Weapons: []string{"l"}, Bonus: defs.Stats{HP: 5, Spd: 3}, AI: "hold", X: 3, Y: 2},
				{Level: 1, Class: "armor", Weapons: []string{"l"}, AI: "hold", X: 3, Y: 0},
			}}}, chapters, nil)
}

// 両軍の騎兵の絵の選び方と素材の記録（DES-UI-22）。
func TestCavalierTile(t *testing.T) {
	t.Run("TC-FR-UNT-09-17 両軍の騎兵を ClassName で騎乗の絵にし、主人公・騎兵・ボス・武器の順で選ぶ", func(t *testing.T) {
		lance := tactics.Weapon{Type: tactics.Lance}
		for _, tc := range []struct {
			name string
			u    tactics.Unit
			want assets.Tile
		}{
			{"自軍の騎兵", tactics.Unit{Name: "エルザ", ClassName: "騎兵", Side: tactics.Player, Weapons: []tactics.Weapon{lance}}, assets.Cavalier},
			{"敵の騎兵", tactics.Unit{Name: "騎兵 1", ClassName: "騎兵", Side: tactics.Enemy, Weapons: []tactics.Weapon{lance}}, assets.Cavalier},
			{"騎兵の武器が槍でなくても騎乗", tactics.Unit{ClassName: "騎兵", Side: tactics.Player, Weapons: []tactics.Weapon{tactics.Weapon{Type: tactics.Sword}}}, assets.Cavalier},
			{"騎兵のボスも騎乗（ボスより騎兵が先）", tactics.Unit{Name: "ヴォルグ", ClassName: "騎兵", Side: tactics.Enemy, Boss: true, Weapons: []tactics.Weapon{lance}}, assets.Cavalier},
			{"主人公は騎兵より先", tactics.Unit{ClassName: "騎兵", Side: tactics.Player, Lord: true, Weapons: []tactics.Weapon{lance}}, assets.PlayerLord},
			{"騎兵でないボスは従来のボス", tactics.Unit{ClassName: "ソルジャー", Side: tactics.Enemy, Boss: true, Weapons: []tactics.Weapon{lance}}, assets.EnemyBoss},
			{"自軍のアーマーは従来の槍", tactics.Unit{Name: "ガルド", ClassName: "アーマー", Side: tactics.Player, Weapons: []tactics.Weapon{lance}}, assets.PlayerLance},
			{"敵のソルジャーは従来の槍", tactics.Unit{ClassName: "ソルジャー", Side: tactics.Enemy, Weapons: []tactics.Weapon{lance}}, assets.EnemyLance},
			{"名前がエルザでもアーマーなら従来の槍", tactics.Unit{Name: "エルザ", ClassName: "アーマー", Side: tactics.Player, Weapons: []tactics.Weapon{lance}, Stats: tactics.Stats{Mov: 7}}, assets.PlayerLance},
			{"未知のクラスは従来の武器別", tactics.Unit{ClassName: "竜騎士", Side: tactics.Enemy, Weapons: []tactics.Weapon{lance}}, assets.EnemyLance},
			{"クラス名の空は従来の武器別", tactics.Unit{Side: tactics.Player, Weapons: []tactics.Weapon{tactics.Weapon{Type: tactics.Bow}}}, assets.PlayerBow},
			{"一部だけ一致するクラス名は騎兵でない", tactics.Unit{ClassName: "重騎兵", Side: tactics.Enemy, Weapons: []tactics.Weapon{lance}}, assets.EnemyLance},
		} {
			if got := unitTile(tc.u); got != tc.want {
				t.Errorf("%s: unitTile = %v、%v のはず", tc.name, got, tc.want)
			}
		}
		// 騎兵のボスは、騎乗の絵のまま名前に ★ の印を添える（盤面の印は drawUnitList が Boss で描く）。
		boss := tactics.Unit{Name: "ヴォルグ", ClassName: "騎兵", Side: tactics.Enemy, Boss: true, Alive: true, HP: 10, Stats: tactics.Stats{HP: 10}, Weapons: []tactics.Weapon{lance}}
		v := Panel{Unit: &boss}.View()
		pic := portraitOf(t, v)
		if pic.Tile != assets.Cavalier {
			t.Errorf("騎兵のボスの肖像 = %v", pic.Tile)
		}
		// 印つきの名前は肖像と重ならず、画面に収まる。
		if tx := textOf(t, v, "ヴォルグ　★ボス"); tx.X < pic.X+pic.Size || tx.X+textWidth(tx.Text) > ScreenWidth {
			t.Errorf("印つきの名前の位置 x = %d、肖像 %+v", tx.X, pic)
		}
	})

	t.Run("TC-FR-UNT-09-17 騎乗の絵は自作の画像の番号 0 で、Tiny Dungeon 97 の代用ではなく、README と一致する", func(t *testing.T) {
		sheets, err := assets.Load()
		if err != nil {
			t.Fatal(err)
		}
		img := sheets[assets.Original]
		if img == nil || img.Bounds().Dx() != assets.TileSize || img.Bounds().Dy() != assets.TileSize {
			t.Fatalf("自作の画像の大きさ = %v、16 × 16 のはず", img)
		}
		if assets.Cavalier != (assets.Tile{Sheet: assets.Original, N: 0}) || !assets.Cavalier.Rect().In(img.Bounds()) {
			t.Fatalf("騎兵のタイル = %v", assets.Cavalier)
		}
		// 透明でない画素があり、アーマー（96）・ボス（97）・ソルジャー（98）と画素が違う。
		opaque := 0
		at := func(tile assets.Tile, x, y int) (uint32, uint32, uint32, uint32) {
			r := tile.Rect()
			return sheets[tile.Sheet].At(r.Min.X+x, r.Min.Y+y).RGBA()
		}
		for y := 0; y < assets.TileSize; y++ {
			for x := 0; x < assets.TileSize; x++ {
				if _, _, _, a := at(assets.Cavalier, x, y); a > 0 {
					opaque++
				}
			}
		}
		if opaque < assets.TileSize*assets.TileSize/3 {
			t.Errorf("騎兵の絵の画素が %d しかない", opaque)
		}
		for _, other := range []assets.Tile{assets.PlayerLance, assets.EnemyBoss, assets.EnemyLance} {
			same := true
			for y := 0; y < assets.TileSize && same; y++ {
				for x := 0; x < assets.TileSize; x++ {
					r1, g1, b1, a1 := at(assets.Cavalier, x, y)
					r2, g2, b2, a2 := at(other, x, y)
					if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
						same = false
						break
					}
				}
			}
			if same || other == assets.Cavalier {
				t.Errorf("騎兵の絵が %v と同じ", other)
			}
		}
		readme, err := os.ReadFile("assets/README.md")
		if err != nil {
			t.Fatal(err)
		}
		var row string
		for _, line := range strings.Split(string(readme), "\n") {
			if strings.HasPrefix(line, "| 両軍の騎兵") {
				row = line
			}
		}
		if row != "| 両軍の騎兵（自軍のエルザ・敵の騎兵。DES-UI-22、#80） | 自作（original.png） | 0 | 馬に乗った鋼の兜の人物 |" {
			t.Errorf("README の騎兵の行 = %q", row)
		}
		for _, want := range []string{"`original.png`", "CC0 1.0", "| 自軍の杖（僧侶。DES-UI-18、#45） | Tiny Dungeon | 108 | 緑のゴースト |"} {
			if !strings.Contains(string(readme), want) {
				t.Errorf("README に %q がない", want)
			}
		}
	})
}

// 騎兵の絵を、場面ごとの表示値で確かめる（DES-UI-22）。
func TestCavalierScenes(t *testing.T) {
	t.Run("TC-FR-UNT-09-18 自軍のエルザと敵の騎兵が、準備・配置・戦闘・情報欄・加入で同じ騎乗の絵になる", func(t *testing.T) {
		s := NewSceneController(Config{Catalog: cavalierCatalog(), Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }})
		startScenes(s)
		if s.Scene() != Prepare {
			t.Fatal(s.Scene())
		}
		// 名簿の順: 主人公、エルザ、ガルド。
		// 敵は名前ではなく陣営とクラスで期待値を引く（配置の敵は名前を持たない）。
		want := map[string]assets.Tile{"主人公": assets.PlayerLord, "エルザ": assets.Cavalier, "ガルド": assets.PlayerLance, "敵の騎兵": assets.Cavalier, "敵のアーマー": assets.EnemyLance}
		check := func(where string, u tactics.Unit) {
			t.Helper()
			key := u.Name
			if u.Side == tactics.Enemy {
				key = "敵の" + u.ClassName
			}
			w, ok := want[key]
			if !ok {
				t.Fatalf("%s: 知らないユニット %q（クラス %q）", where, u.Name, u.ClassName)
			}
			if got := unitTile(u); got != w {
				t.Errorf("%s: %s（%s）の絵 = %v、%v のはず", where, u.Name, u.ClassName, got, w)
			}
			// 情報欄の肖像（64 px）も盤面と同じ絵。
			if pic := portraitOf(t, Panel{Unit: &u}.View()); pic.Tile != w {
				t.Errorf("%s: %s の肖像 = %v、%v のはず", where, u.Name, pic.Tile, w)
			}
		}

		// 準備: 名簿の行の絵と、選んだ味方の情報欄。
		v := s.View()
		if len(v.Icons) < 3 || v.Icons[0].Tile != assets.PlayerLord || v.Icons[1].Tile != assets.Cavalier || v.Icons[2].Tile != assets.PlayerLance {
			t.Fatalf("準備の名簿の絵 = %+v", v.Icons)
		}
		s.prepareMember = 1
		if v := s.View(); v.Info == nil || portraitOf(t, *v.Info).Tile != assets.Cavalier {
			t.Fatal("準備の情報欄のエルザが騎乗の絵でない")
		}
		for i := range s.campaign.Roster {
			check("準備", s.prepareUnit(i))
		}

		// 配置: 味方と、定義から作る敵の両方に ClassName が渡る。
		clickSceneButton(t, s, "配置へ")
		d := s.View().Deploy
		if d == nil || len(d.Units) != 6 {
			t.Fatalf("配置のユニット = %+v", d)
		}
		for _, u := range d.Units {
			// 配置の敵は武器をまだ持たない（正しい敵の表示は DES-DEP-02 の担当）ので、騎兵だけを確かめる。
			if u.Side == tactics.Enemy && u.ClassName != "騎兵" {
				continue
			}
			check("配置", u)
		}
		if d.Units[3].ClassName != "騎兵" || !d.Units[3].Boss || d.Units[4].ClassName != "騎兵" || d.Units[5].ClassName != "アーマー" {
			t.Errorf("配置の敵のクラス = %q %q %q", d.Units[3].ClassName, d.Units[4].ClassName, d.Units[5].ClassName)
		}
		clickDeployCell(s, d.Slots[1])
		if v := s.View(); v.Info == nil || portraitOf(t, *v.Info).Tile != assets.Cavalier {
			t.Fatal("配置の情報欄のエルザが騎乗の絵でない")
		}
		clickDeployCell(s, d.Slots[1])

		// 戦闘: core が cavalier を「騎兵」に変換し、盤面のユニットと情報欄が騎乗の絵になる。
		clickSceneButton(t, s, "戦闘開始")
		c := s.BattleController()
		playOut(c)
		cavaliers := 0
		for _, u := range c.Units() {
			check("戦闘", u)
			if unitTile(u) == assets.Cavalier {
				cavaliers++
				if pic := portraitOf(t, c.Panel(u.Pos, true).View()); pic.Tile != assets.Cavalier {
					t.Errorf("戦闘の情報欄の %s = %v", u.Name, pic.Tile)
				}
			}
		}
		if cavaliers != 3 {
			t.Errorf("戦闘の騎乗の絵が %d 体、3 体（エルザ・隊長・騎兵）のはず", cavaliers)
		}

		// 加入: 2 章で加わる騎兵（能力値・名前がエルザと違う）も、クラスを渡して騎乗の絵にする。
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.Click(tactics.Pos{X: 0, Y: 0})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 0})
		c.Click(tactics.Pos{X: 1, Y: 0})
		if c.Battle().Result != tactics.Won {
			t.Fatal("1 章に勝てない")
		}
		playOut(c)
		for i := 0; i < 10 && s.Scene() != Joined; i++ {
			s.Advance()
		}
		if s.Scene() != Joined {
			t.Fatal("加入の場面にならない", s.Scene())
		}
		v = s.View()
		if len(v.Icons) != 1 || v.Icons[0].Tile != assets.Cavalier || !strings.Contains(viewStrings(v), "騎兵　Lv 5") {
			t.Fatalf("加入の絵 = %+v、文 = %s", v.Icons, viewStrings(v))
		}
		if v.Icons[0].X != 48 || v.Icons[0].Y != 208 || !(image.Pt(v.Icons[0].X, v.Icons[0].Y).In(image.Rect(0, 0, ScreenWidth, ScreenHeight))) {
			t.Errorf("加入の絵の位置 = %+v", v.Icons[0])
		}
	})
}

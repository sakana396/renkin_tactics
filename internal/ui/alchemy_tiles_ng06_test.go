package ui

import (
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 新作の地形・ユニットの絵（DES-NG-06。表示だけで、規則は変えない）。
func TestAlchemyTilesNG06(t *testing.T) {
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("TC-DES-NG-06-11 新作の地形は名前で描き分け、水路と崩れた壁・浄水施設と砦が同じ絵にならない", func(t *testing.T) {
		// 期待値は設計メモ 6.6 節の対応表から。瓦礫は山、植え込みは森、崩れた壁は壁、水路・浄水施設は専用の絵、石畳は平地（下地だけ）。
		type want struct {
			tile assets.Tile
			ok   bool
		}
		wants := map[string]want{
			"石畳":   {assets.Tile{}, false},
			"瓦礫":   {assets.Mountain, true},
			"植え込み": {assets.Forest, true},
			"崩れた壁": {assets.Wall, true},
			"水路":   {assets.Canal, true},
			"浄水施設": {assets.Purifier, true},
		}
		imgs, err := assets.Load()
		if err != nil {
			t.Fatal(err)
		}
		seen := 0
		for _, d := range cat.Terrains() {
			w, ok := wants[d.Name]
			if !ok {
				t.Fatalf("対応表にない地形 %q", d.Name)
			}
			seen++
			tr := tactics.Terrain{Name: d.Name, Cost: d.Cost, Avoid: d.Avoid, Def: d.Def, HealPercent: d.HealPercent}
			got, gotOK := terrainTile(tr)
			if got != w.tile || gotOK != w.ok {
				t.Errorf("terrainTile(%s) = %v, %v、%v, %v のはず", d.Name, got, gotOK, w.tile, w.ok)
			}
			if gotOK && !got.Rect().In(imgs[got.Sheet].Bounds()) {
				t.Errorf("%s のタイル %v が画像の外", d.Name, got)
			}
		}
		if seen != len(wants) {
			t.Fatalf("データの地形 %d 種、対応表 %d 種", seen, len(wants))
		}
		canal, _ := terrainTile(tactics.Terrain{Name: "水路"})
		wall, _ := terrainTile(tactics.Terrain{Name: "崩れた壁"})
		purifier, _ := terrainTile(tactics.Terrain{Name: "浄水施設", Cost: 1, HealPercent: 20})
		fort, _ := terrainTile(tactics.Terrain{Name: "砦", Cost: 1, HealPercent: 20})
		if canal == wall || purifier == fort {
			t.Errorf("同じ絵になっている: 水路 %v 崩れた壁 %v、浄水施設 %v 砦 %v", canal, wall, purifier, fort)
		}
		// 既存の作品の地形は変わらない（DES-UI-05）。
		for _, tc := range []struct {
			name string
			want assets.Tile
		}{{"森", assets.Forest}, {"山", assets.Mountain}, {"壁", assets.Wall}} {
			if got, ok := terrainTile(tactics.Terrain{Name: tc.name, Cost: 1}); !ok || got != tc.want {
				t.Errorf("%s: %v, %v", tc.name, got, ok)
			}
		}
	})
	t.Run("TC-DES-NG-06-12 ハル（最初の武器が魔法の自軍）は魔道士の絵。敵や既存の兵種は変わらない", func(t *testing.T) {
		haru, ok := cat.Character("haru")
		if !ok || len(haru.Weapons) == 0 {
			t.Fatal(haru, ok)
		}
		first, _ := cat.Weapon(haru.Weapons[0])
		if first.Type != "magic" {
			t.Fatalf("ハルの最初の武器の種類 %q（魔法のはず）", first.Type)
		}
		cls, _ := cat.Class(haru.Class)
		if cls.Name == "錬金術師" || cls.Name == "魔道士" {
			t.Fatalf("ハルのクラス名 %q は、クラス名では魔道士にならない", cls.Name)
		}
		u := tactics.Unit{Name: haru.Name, ClassName: cls.Name, Side: tactics.Player, Weapons: []tactics.Weapon{{Type: tactics.Magic}, {Type: tactics.Staff}}}
		if got := unitTile(u); got != assets.Mage {
			t.Errorf("ハルの絵 %v（魔道士 %v のはず）", got, assets.Mage)
		}
		u.Equipped = 1
		if got := unitTile(u); got != assets.Mage {
			t.Errorf("持ち替えたハルの絵 %v", got)
		}
		// 自軍の剣・槍は変わらない。ロードは主人公の絵のまま。
		if got := unitTile(tactics.Unit{Side: tactics.Player, Weapons: []tactics.Weapon{testSword}}); got != assets.PlayerSword {
			t.Errorf("自軍の剣 %v", got)
		}
		if got := unitTile(tactics.Unit{Side: tactics.Player, Lord: true, Weapons: []tactics.Weapon{{Type: tactics.Magic}}}); got != assets.PlayerLord {
			t.Errorf("主人公 %v", got)
		}
		// 敵の魔法は、これまでどおり剣の絵（この修正は自軍だけ）。
		if got := unitTile(tactics.Unit{Side: tactics.Enemy, Weapons: []tactics.Weapon{{Type: tactics.Magic}}}); got != assets.EnemySword {
			t.Errorf("敵の魔法 %v", got)
		}
	})
}

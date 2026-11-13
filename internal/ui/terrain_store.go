package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// artTerrainDir は地形の絵の置き場（media の下。DES-NG-10 フェーズ 2）。
const artTerrainDir = "art/terrain"

// terrainArtPaths は、地形 id の絵の道筋を、変種の順（<id>.png、<id>_2.png、_3、_4）に返す。
func terrainArtPaths(id string) []string {
	paths := []string{artTerrainDir + "/" + id + ".png"}
	for i := 2; i <= terrainVariants; i++ {
		paths = append(paths, artTerrainDir+"/"+id+"_"+strconv.Itoa(i)+".png")
	}
	return paths
}

// terrainSet は地形 1 つの絵の組。選び方は、差し替えの PNG（あるものだけ）、なければ仮の絵（4 枚）。
type terrainSet struct {
	imgs []*ebiten.Image
	// fromFile は imgs が media の PNG であること（仮の絵ではない）。
	fromFile bool
}

// TerrainID は地形の表示名から地形の ID を返す。目録にない名前は ok が偽。
func (a *ArtStore) TerrainID(name string) (string, bool) {
	if a == nil {
		return "", false
	}
	id, ok := a.idx.terrainByName[name]
	return id, ok
}

// terrainSetOf は地形 id の絵の組を、1 回だけ作って覚える。
// media の art/terrain の PNG が 1 枚でもあればそれだけを使い、なければ仮の絵があれば仮の絵、どちらもなければ空。
func (a *ArtStore) terrainSetOf(id string) *terrainSet {
	if s, ok := a.terrains[id]; ok {
		return s
	}
	s := &terrainSet{}
	a.terrains[id] = s
	for _, p := range terrainArtPaths(id) {
		if eb := a.Ebiten(p, true); eb != nil {
			s.imgs = append(s.imgs, eb)
		}
	}
	if len(s.imgs) > 0 {
		s.fromFile = true
		return s
	}
	for v := 0; v < terrainVariants; v++ {
		if img, ok := terrainPlaceholder(id, v); ok {
			s.imgs = append(s.imgs, ebiten.NewImageFromImage(img))
		}
	}
	return s
}

// TerrainImage は、地形の名前 name のマス (x, y) に描く絵を返す（DES-NG-10 フェーズ 2）。
// 差し替えの PNG（art/terrain/<地形の ID>.png、_2〜_4 は変種）があればそれ、なければ仮の絵。
// どちらもなければ nil（今のドット絵のタイルで描く）。変種はマスの座標で決まる。
func (a *ArtStore) TerrainImage(name string, x, y int) *ebiten.Image {
	id, ok := a.TerrainID(name)
	if !ok {
		return nil
	}
	s := a.terrainSetOf(id)
	if len(s.imgs) == 0 {
		return nil
	}
	return s.imgs[terrainVariantIndex(x, y, len(s.imgs))]
}

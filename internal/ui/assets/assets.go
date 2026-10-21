// Package assets は、画面で使う Kenney の画像を埋め込み、地形とユニットのタイルを切り出す（DES-UI-05）。
//
// 画像の読み込みはふしぎ錬金（sakana396/fushigi_renkin の internal/ui/assets/assets.go）と同じ形にした。
// 使ったタイルの番号と意味は README.md に記録する。
package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
)

//go:embed tiny_town.png
var tinyTownPNG []byte

//go:embed tiny_dungeon.png
var tinyDungeonPNG []byte

// TileSize はタイル 1 枚の大きさ（ピクセル）。画面では 2 倍で描く（DES-UI-01）。
const TileSize = 16

// Columns は、どちらの画像でも 1 行に並ぶタイルの数。タイルの番号は左上から右へ 0, 1, 2, … と数える。
const Columns = 12

// Sheet はタイルの画像の種類。
type Sheet int

const (
	// Town は Tiny Town（tiny_town.png）。
	Town Sheet = iota
	// Dungeon は Tiny Dungeon（tiny_dungeon.png）。
	Dungeon
)

// Tile は画像の種類とタイルの番号の組（DES-UI-05）。
type Tile struct {
	Sheet Sheet
	N     int
}

// 地形のタイル（DES-UI-05）。どの地形も、下地に Grass を描いてから重ねる。
var (
	// Grass は平地（下地にも使う）。
	Grass = Tile{Town, 0}
	// Forest は森の木。
	Forest = Tile{Town, 16}
	// Mountain は山（岩の多い地面）。
	Mountain = Tile{Town, 43}
	// Fort は砦（石の建物）。
	Fort = Tile{Town, 125}
	// Wall は壁（石の壁）。
	Wall = Tile{Dungeon, 40}
)

// ユニットのタイル（DES-UI-05）。どれも Tiny Dungeon の人物。
var (
	// PlayerLord は自軍の主人公（ロード）。
	PlayerLord = Tile{Dungeon, 88}
	// PlayerSword は主人公でない自軍の剣の使い手。
	PlayerSword = Tile{Dungeon, 85}
	// PlayerLance は自軍の槍の使い手（アーマー）。
	PlayerLance = Tile{Dungeon, 96}
	// PlayerAxe は自軍の斧の使い手（ファイター）。
	PlayerAxe = Tile{Dungeon, 87}
	// PlayerBow は自軍の弓の使い手（アーチャー）。
	PlayerBow = Tile{Dungeon, 112}
	// EnemyBoss は敵のボス。
	EnemyBoss = Tile{Dungeon, 97}
	// EnemySword は敵の剣の使い手（剣士）。
	EnemySword = Tile{Dungeon, 111}
	// EnemyLance は敵の槍の使い手（ソルジャー）。
	EnemyLance = Tile{Dungeon, 98}
	// EnemyAxe は敵の斧の使い手（ファイター）。
	EnemyAxe = Tile{Dungeon, 86}
	// EnemyBow は敵の弓の使い手（アーチャー）。
	EnemyBow = Tile{Dungeon, 100}
)

// Rect は、画像の中でのタイルの範囲を返す（DES-UI-05）。
func (t Tile) Rect() image.Rectangle {
	x, y := (t.N%Columns)*TileSize, (t.N/Columns)*TileSize
	return image.Rect(x, y, x+TileSize, y+TileSize)
}

// Load は、埋め込んだ画像を読み、画像の種類ごとに返す（DES-UI-05）。
func Load() (map[Sheet]image.Image, error) {
	sheets := map[Sheet]image.Image{}
	for _, s := range []struct {
		sheet Sheet
		name  string
		data  []byte
	}{
		{Town, "Tiny Town", tinyTownPNG},
		{Dungeon, "Tiny Dungeon", tinyDungeonPNG},
	} {
		img, err := png.Decode(bytes.NewReader(s.data))
		if err != nil {
			return nil, fmt.Errorf("%s の画像: %w", s.name, err)
		}
		sheets[s.sheet] = img
	}
	return sheets, nil
}

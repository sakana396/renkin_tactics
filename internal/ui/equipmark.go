package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// 盤面の装備の印（DES-UI-33、DES-IT07 3.6 節）。ユニットの 32×32 の絵の右下に、装備している武器の種類の絵を小さく置く。
// 印の置き場所は、指揮官（小隊長）を左上、固定の範囲のピンを右上（pinShape）、装備を右下に分ける。
const (
	// EquipMarkSize は装備の印の大きさ（10×10）。
	EquipMarkSize = 10
	// equipMarkOffset は、マスの左上から印の左上までの位置（相対 x=22〜31, y=22〜31）。
	equipMarkOffset = CellSize - EquipMarkSize
	// unitHPBarRight は、HP の棒の右端（マスの左端から）。装備の印の左に 1 px 空けて止める（#167 の PR の設計への提案）。
	unitHPBarRight = equipMarkOffset - 1
)

// EquipMark は盤面に描く装備の印 1 つ（DES-UI-33）。Rect は画面の座標。
type EquipMark struct {
	Pos  tactics.Pos
	Rect image.Rectangle
	Tile assets.Tile
}

// equipMarkRect は、画面の左上が pt のマスに置く装備の印の範囲を返す（DES-UI-33）。
func equipMarkRect(pt image.Point) image.Rectangle {
	return image.Rect(pt.X+equipMarkOffset, pt.Y+equipMarkOffset, pt.X+CellSize, pt.Y+CellSize)
}

// unitHPBarRect は、画面の左上が pt のマスに描く HP の棒の枠（下地）の範囲を返す（DES-UI-03）。
// 下端の 4 px に置き、右端は装備の印の手前で止める（DES-UI-33）。
func unitHPBarRect(pt image.Point) image.Rectangle {
	return image.Rect(pt.X+2, pt.Y+CellSize-5, pt.X+unitHPBarRight, pt.Y+CellSize-1)
}

// equipMarks は、ユニットの列 units のうち、カメラ cam に見えている生きているユニットの装備の印を、列の順に返す（DES-UI-33）。
// 印は装備している武器（Equipped）の種類の絵 1 つだけで、武器が複数でも 1 つ、杖だけなら杖になる。武器のないユニットには付けない。
// 装備は core が攻撃の決定に成功したときだけ変えるので、予測で武器を送っただけでは印は変わらない（DES-IT06 6 章）。
func equipMarks(units []tactics.Unit, cam Camera) []EquipMark {
	var out []EquipMark
	for _, u := range units {
		if !u.Alive || !cam.Visible(u.Pos) || u.Equipped < 0 || u.Equipped >= len(u.Weapons) {
			continue
		}
		t, ok := weaponTile(u.Weapons[u.Equipped].Type)
		if !ok {
			continue
		}
		out = append(out, EquipMark{Pos: u.Pos, Rect: equipMarkRect(cam.Screen(u.Pos)), Tile: t})
	}
	return out
}

// EquipMarks は、戦闘の盤面に描く装備の印を返す（DES-UI-33）。ユニットは盤面に描くもの（Units）から引く。
func (c *Controller) EquipMarks() []EquipMark {
	return equipMarks(c.Units(), c.Camera())
}

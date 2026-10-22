package ui

import (
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// terrainTile は地形に重ねるタイルを返す（DES-UI-05）。平地なら ok が偽（下地だけを描く）。
//
// tactics の地形は記号を持たないので、名前で選ぶ。名前の分からない地形は、
// 通れなければ壁、回復があれば砦、それ以外は平地として描く。
func terrainTile(t tactics.Terrain) (assets.Tile, bool) {
	switch t.Name {
	case "平地":
		return assets.Tile{}, false
	case "森":
		return assets.Forest, true
	case "山":
		return assets.Mountain, true
	case "砦":
		return assets.Fort, true
	case "壁":
		return assets.Wall, true
	}
	switch {
	case t.Cost == 0:
		return assets.Wall, true
	case t.HealPercent > 0:
		return assets.Fort, true
	}
	return assets.Tile{}, false
}

// unitTile はユニットのタイルを返す（DES-UI-05）。陣営と武器の種類で選び、主人公とボスは専用のタイルにする。
func unitTile(u tactics.Unit) assets.Tile {
	if u.Side == tactics.Player {
		if u.Lord {
			return assets.PlayerLord
		}
		switch u.Weapon.Type {
		case tactics.Lance:
			return assets.PlayerLance
		case tactics.Axe:
			return assets.PlayerAxe
		case tactics.Staff:
			return assets.PlayerStaff
		case tactics.Bow:
			return assets.PlayerBow
		}
		return assets.PlayerSword
	}
	if u.Boss {
		return assets.EnemyBoss
	}
	switch u.Weapon.Type {
	case tactics.Lance:
		return assets.EnemyLance
	case tactics.Axe:
		return assets.EnemyAxe
	case tactics.Bow:
		return assets.EnemyBow
	}
	return assets.EnemySword
}

// weaponTile は武器の種類の絵を返す（DES-UI-06）。知らない種類なら ok が偽（絵を描かない）。
func weaponTile(t tactics.WeaponType) (assets.Tile, bool) {
	switch t {
	case tactics.Sword:
		return assets.WeaponSword, true
	case tactics.Lance:
		return assets.WeaponLance, true
	case tactics.Axe:
		return assets.WeaponAxe, true
	case tactics.Staff:
		return assets.WeaponStaff, true
	case tactics.Bow:
		return assets.WeaponBow, true
	}
	return assets.Tile{}, false
}

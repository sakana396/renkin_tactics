package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// equipSetup は装備の印（DES-UI-33）を確かめる 7 × 7 の平地。
//
//	0: 剣兵 (2, 2)。鉄の剣（装備）・鉄の槍・鉄の弓の 3 本。弓は射程 2 なので隣の敵には届かない
//	1: 杖役 (0, 6)。癒やしの杖だけ
//	2: ファイター (3, 2)。鉄の斧。HP 60 で倒れない。待ち構える
//	3: ボス (6, 6)。鉄の槍。待ち構える
func equipSetup() tactics.Setup {
	staff := tactics.Weapon{Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, MinRange: 1, MaxRange: 1, Uses: 15, MaxUses: 15}
	return tactics.Setup{
		Terrain: field(7, 7),
		Units: []tactics.UnitSetup{
			{Name: "剣兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Def: 30, Mov: 3}, Pos: tactics.Pos{X: 2, Y: 2}, Weapons: []tactics.Weapon{ironSword, ironLance, ironBow}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Bow}, Lord: true},
			{Name: "杖役", Side: tactics.Player, Stats: tactics.Stats{HP: 20, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 6}, Weapons: []tactics.Weapon{staff}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Staff}},
			{Name: "ファイター", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 60}, Pos: tactics.Pos{X: 3, Y: 2}, Weapon: ironAxe, AI: tactics.Hold},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 6, Y: 6}, Weapon: ironLance, Boss: true, AI: tactics.Hold},
		},
	}
}

// markAt は、盤面のマス p の装備の印を返す。なければ ok が偽。
func markAt(ms []EquipMark, p tactics.Pos) (EquipMark, bool) {
	for _, m := range ms {
		if m.Pos == p {
			return m, true
		}
	}
	return EquipMark{}, false
}

// equippedRowName は、情報欄の「能力」のページで装備の印「E」と同じ行・同じ列に並ぶ武器の名前を返す（DES-IT06 12.2 節）。
// 印は列の左端、名前は列の左端から 30 px 右に置く。
func equippedRowName(t *testing.T, u tactics.Unit) string {
	t.Helper()
	v := Panel{Unit: &u}.View()
	var mark *InfoText
	for i, tx := range v.Texts {
		if tx.Text == weaponRowMark {
			if mark != nil {
				t.Fatalf("装備の印が 2 つある: %q", rows(v))
			}
			mark = &v.Texts[i]
		}
	}
	if mark == nil {
		t.Fatalf("装備の印がない: %q", rows(v))
	}
	for _, tx := range v.Texts {
		if tx.Y == mark.Y && tx.X == mark.X+30 {
			return tx.Text
		}
	}
	t.Fatalf("装備の印の行に名前がない: %q", rows(v))
	return ""
}

// toTargeting は、剣兵をその場にとどめて攻撃を選び、ファイターを相手に選んだ予測（Targeting）にする。
// 武器の一覧があれば 1 行目（今の装備の剣）を選ぶ。予測で武器を送る操作の形によらず、そのあと仮選択の武器（c.weapon）を替えて確かめる。
func toTargeting(t *testing.T, c *Controller) {
	t.Helper()
	unit, enemy := tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 3, Y: 2}
	c.Click(unit)
	c.Click(unit)
	c.ChooseAction(tactics.Attack)
	if c.State() == WeaponMenu {
		c.ChooseWeapon(0)
	}
	c.Click(enemy)
	if _, _, tg := c.Selection(); c.State() != Targeting || tg != 2 {
		t.Fatalf("状態 = %v、相手 %d。ファイターを選んだ Targeting のはず", c.State(), tg)
	}
}

// 盤面と情報欄の装備の印（DES-UI-33、DES-IT07 3.6 節）を確かめる。期待する位置は、マス (x, y) の左上 (32x, 32y) から
// 相対 x=22〜31, y=22〜31 の 10×10 という設計の値から手で求めた。
func TestEquipMarkIT07(t *testing.T) {
	sword, staffPos, fighter, boss := tactics.Pos{X: 2, Y: 2}, tactics.Pos{X: 0, Y: 6}, tactics.Pos{X: 3, Y: 2}, tactics.Pos{X: 6, Y: 6}

	t.Run("TC-FR-UNT-09-26 盤面の各ユニットの右下に装備中の武器の種類の 10×10 の絵を 1 つだけ出す。複数の武器でも装備の 1 つ、杖だけなら杖。武器なし・戦死・カメラの外には出さない", func(t *testing.T) {
		c := newIdleWith(t, equipSetup)
		ms := c.EquipMarks()
		want := []EquipMark{
			{Pos: sword, Rect: image.Rect(86, 86, 96, 96), Tile: assets.WeaponSword},
			{Pos: staffPos, Rect: image.Rect(22, 214, 32, 224), Tile: assets.WeaponStaff},
			{Pos: fighter, Rect: image.Rect(118, 86, 128, 96), Tile: assets.WeaponAxe},
			{Pos: boss, Rect: image.Rect(214, 214, 224, 224), Tile: assets.WeaponLance},
		}
		if !reflect.DeepEqual(ms, want) {
			t.Errorf("装備の印 = %+v\nのはず %+v", ms, want)
		}
		for _, m := range ms {
			if m.Rect.Dx() != 10 || m.Rect.Dy() != 10 {
				t.Errorf("%v の印の大きさ %v、10×10 のはず", m.Pos, m.Rect)
			}
		}
		// 移動先を選んだ（Moved）あとは、盤面に描く移動先の写しに印が付く。
		moved := tactics.Pos{X: 2, Y: 3}
		c.Click(sword)
		c.Click(moved)
		if c.State() != Moved {
			t.Fatalf("状態 %v、Moved のはず", c.State())
		}
		if m, ok := markAt(c.EquipMarks(), moved); !ok || m.Rect != image.Rect(86, 118, 96, 128) || m.Tile != assets.WeaponSword {
			t.Errorf("移動先の印 %+v", m)
		}
		if _, ok := markAt(c.EquipMarks(), sword); ok {
			t.Errorf("移動の前のマス %v に印が残る", sword)
		}

		// 2 本目を装備しているユニットは 2 本目の種類。武器なし・戦死は出さない。カメラを右へ 1 マス動かすと画面の位置もずれ、外のマスは出さない。
		units := []tactics.Unit{
			{Pos: tactics.Pos{X: 1, Y: 1}, Alive: true, Weapons: []tactics.Weapon{ironSword, ironBow}, Equipped: 1},
			{Pos: tactics.Pos{X: 2, Y: 1}, Alive: true},
			{Pos: tactics.Pos{X: 3, Y: 1}, Alive: false, Weapons: []tactics.Weapon{ironAxe}},
			{Pos: tactics.Pos{X: 0, Y: 1}, Alive: true, Weapons: []tactics.Weapon{ironAxe}},
		}
		got := equipMarks(units, Camera{X: 1, Y: 0, Width: 7, Height: 7})
		if want := []EquipMark{{Pos: tactics.Pos{X: 1, Y: 1}, Rect: image.Rect(22, 54, 32, 64), Tile: assets.WeaponBow}}; !reflect.DeepEqual(got, want) {
			t.Errorf("2 本目・武器なし・戦死・カメラの外 = %+v\nのはず %+v", got, want)
		}
	})

	t.Run("TC-FR-UNT-09-26 印の置き場所: 装備は右下、固定のピンは右上、ボスの ★ は右上、指揮官のための左上は空ける。HP の棒は装備の印の手前で止め、重ならない", func(t *testing.T) {
		c := newIdleWith(t, equipSetup)
		// ボスを固定する（Idle で敵をクリック）。
		c.Click(boss)
		if pins := c.WatchedPins(); !reflect.DeepEqual(pins, []tactics.Pos{boss}) {
			t.Fatalf("固定のピン %v", pins)
		}
		for _, m := range c.EquipMarks() {
			pt := c.Camera().Screen(m.Pos)
			cell := image.Rect(pt.X, pt.Y, pt.X+CellSize, pt.Y+CellSize)
			if !m.Rect.In(cell) {
				t.Errorf("%v の印 %v がマス %v の外", m.Pos, m.Rect, cell)
			}
			// 右下の 4 分の 1 に収め、左上（指揮官の印。#172）と右上（固定・★）に出ない。
			if m.Rect.Min.X < pt.X+CellSize/2 || m.Rect.Min.Y < pt.Y+CellSize/2 {
				t.Errorf("%v の印 %v が右下にない", m.Pos, m.Rect)
			}
			isBoss := m.Pos == boss
			head, stem := pinShape(pt, isBoss)
			if m.Rect.Overlaps(head) || m.Rect.Overlaps(stem) {
				t.Errorf("%v の印 %v が固定のピン %v %v と重なる", m.Pos, m.Rect, head, stem)
			}
			if star := rectOfText(pt.X+CellSize-12, pt.Y-1, "★", false); isBoss && m.Rect.Overlaps(star) {
				t.Errorf("ボスの印 %v が ★ %v と重なる", m.Rect, star)
			}
			bar := unitHPBarRect(pt)
			if bar != image.Rect(pt.X+2, pt.Y+27, pt.X+21, pt.Y+31) {
				t.Errorf("%v の HP の棒 %v", m.Pos, bar)
			}
			if bar.Overlaps(m.Rect) {
				t.Errorf("%v の HP の棒 %v が印 %v と重なる", m.Pos, bar, m.Rect)
			}
		}
	})

	t.Run("TC-FR-ITM-01-07 / TC-FR-UNT-09-26 予測で武器を仮選択しただけでは core の装備・盤面の印・情報欄の E 印は変わらず、攻撃の決定に成功したときだけその武器になる", func(t *testing.T) {
		c := newIdleWith(t, equipSetup)
		toTargeting(t, c)
		c.weapon = 1 // 予測で鉄の槍を仮選択した。
		if _, ok := c.attackForecast(); !ok {
			t.Fatalf("槍の予測が出ない")
		}
		checkEquip := func(what string, tile assets.Tile, idx int, name string) {
			t.Helper()
			if m, ok := markAt(c.EquipMarks(), sword); !ok || m.Tile != tile {
				t.Errorf("%s: 盤面の印 %+v、%v のはず", what, m, tile)
			}
			if u := c.battle.Units()[0]; u.Equipped != idx {
				t.Errorf("%s: core の装備 %d、%d のはず", what, u.Equipped, idx)
			}
			if got := equippedRowName(t, c.Units()[0]); got != name {
				t.Errorf("%s: 情報欄の E 印の行 %q、%q のはず", what, got, name)
			}
		}
		checkEquip("槍を仮選択", assets.WeaponSword, 0, "鉄の剣")
		// 相手をもう一度クリックして決定する。
		c.Click(fighter)
		if c.State() != Playing {
			t.Fatalf("決定のあと %v、Playing のはず（%s）", c.State(), c.message)
		}
		playOut(c)
		checkEquip("槍で攻撃を決定", assets.WeaponLance, 1, "鉄の槍")
	})

	t.Run("TC-FR-ITM-01-07 / TC-FR-UNT-09-26 槍を仮選択したあと決定が失敗したとき（core が攻撃のコマンドを拒む）は、盤面の印・情報欄の E 印・core の装備・位置・行動の履歴を変えない", func(t *testing.T) {
		c := newIdleWith(t, equipSetup)
		toTargeting(t, c)
		history := len(c.battle.History())
		c.weapon = 1 // 予測で鉄の槍を仮選択した。
		// 画面では届く相手しか選べないので、決定の経路（act）に、隣の相手に届かない弓（3 本目）のコマンドを渡して失敗させる。
		bow := 2
		c.act(tactics.Command{Unit: 0, To: sword, Action: tactics.Attack, Target: 2, Weapon: &bow})
		if c.State() == Playing || !strings.HasPrefix(c.message, "行動できません") {
			t.Fatalf("決定が失敗していない: 状態 %v、%q", c.State(), c.message)
		}
		if m, ok := markAt(c.EquipMarks(), sword); !ok || m.Tile != assets.WeaponSword {
			t.Errorf("盤面の印 %+v、剣のはず", m)
		}
		if u := c.battle.Units()[0]; u.Equipped != 0 || u.Pos != sword {
			t.Errorf("core の剣兵 %+v", u)
		}
		if got := equippedRowName(t, c.Units()[0]); got != "鉄の剣" {
			t.Errorf("情報欄の E 印の行 %q", got)
		}
		if len(c.battle.History()) != history {
			t.Errorf("履歴 %d → %d", history, len(c.battle.History()))
		}
	})
}

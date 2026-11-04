package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 武器・杖の選択の画面（DES-UI-26、DES-IT06 12.1 節）。
var (
	menuStaff = tactics.Weapon{ID: "heal_staff", Name: "癒やしの杖", Type: tactics.Staff, Heal: 10, MaxUses: 3, MinRange: 1, MaxRange: 1}
	menuFire  = tactics.Weapon{ID: "fire", Name: "ファイア", Type: tactics.Magic, DamageType: tactics.Magical, Might: 5, Hit: 85, Weight: 2, MinRange: 1, MaxRange: 2}
	menuBlade = tactics.Weapon{ID: "iron_sword", Name: "鉄の剣", Type: tactics.Sword, Might: 5, Hit: 90, Weight: 4, MinRange: 1, MaxRange: 1}
)

// weaponMenuController は、杖 2 本と攻撃の武器 2 本（ファイア・剣）を持つ錬金術師（動かない）と、
// 敵に 1 回攻撃されて HP の減った味方、距離 2 のボスの盤面で、自軍の番の入力待ちにする。
func weaponMenuController(t *testing.T) *Controller {
	t.Helper()
	c := newIdleWith(t, func() tactics.Setup {
		return tactics.Setup{Terrain: field(7, 7), Units: []tactics.UnitSetup{
			{Name: "ノエル", ClassName: "錬金術師", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mag: 5}, Pos: tactics.Pos{X: 1, Y: 1},
				Weapons: []tactics.Weapon{menuStaff, menuFire, menuBlade, menuStaff}, AllowedWeaponTypes: []tactics.WeaponType{tactics.Staff, tactics.Magic, tactics.Sword}},
			{Name: "兵", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 1, Y: 2}, Weapon: testSword},
			{Name: "ボス", Side: tactics.Enemy, Level: 1, Boss: true, AI: tactics.Hold, Stats: tactics.Stats{HP: 30, Str: 5}, Pos: tactics.Pos{X: 1, Y: 3}, Weapon: testSword},
		}}
	})
	// 番を 1 回終え、ボスが味方を攻撃する（味方は HP 15 になる）。
	c.EndPhase()
	c.EndPhase()
	playOut(c)
	if c.Battle().Units()[1].HP >= 20 {
		t.Fatal("前提: 味方の HP が減っていない")
	}
	return c
}

func TestWeaponMenu(t *testing.T) {
	t.Run("TC-IF-01-41 攻撃は相手から予測へ進み、決定まで装備を変えない", func(t *testing.T) {
		c := weaponMenuController(t)
		history := len(c.Battle().History())
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 1})
		if c.State() != Moved || !reflect.DeepEqual(c.Actions(), []tactics.Action{tactics.Attack, tactics.Heal, tactics.Wait}) {
			t.Fatalf("行動の一覧 %v（状態 %v）", c.Actions(), c.State())
		}
		c.ChooseAction(tactics.Attack)
		if c.State() != Targeting || c.target != -1 || !reflect.DeepEqual(c.Targets(), []int{2}) {
			t.Fatalf("相手選択: 状態 %v、相手 %d、候補 %v", c.State(), c.target, c.Targets())
		}
		c.Click(tactics.Pos{X: 1, Y: 3})
		p := c.Panel(tactics.Pos{}, false)
		if len(p.Forecast) == 0 || p.ForecastWeapons[0] != tactics.Magic || c.weapon != 1 || c.Battle().Units()[0].Equipped != 0 {
			t.Fatalf("予測 %v、武器 %d、装備 %d", p.Forecast, c.weapon, c.Battle().Units()[0].Equipped)
		}
		for i, want := range []State{Targeting, Moved, Selected} {
			c.Cancel()
			if c.State() != want || c.target != -1 {
				t.Fatalf("取り消し %d: 状態 %v、相手 %d", i+1, c.State(), c.target)
			}
		}
		if c.Battle().Units()[0].Equipped != 0 || len(c.Battle().History()) != history {
			t.Fatal("取り消しで装備か履歴が変わった")
		}
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.ChooseAction(tactics.Attack)
		c.Click(tactics.Pos{X: 1, Y: 3})
		c.Click(tactics.Pos{X: 1, Y: 3})
		playOut(c)
		if u := c.Battle().Units()[0]; u.Equipped != 1 || !u.Acted {
			t.Fatalf("決定のあと 装備 %d、行動 %v", u.Equipped, u.Acted)
		}
		c.infoPage = PageEquip
		pos := c.Camera().Screen(c.Battle().Units()[0].Pos)
		if got := strings.Join(rows(c.InfoView(pos.X+2, pos.Y+2)), "\n"); !strings.Contains(got, "ほかの武器: ") || !strings.Contains(got, "鉄の剣") {
			t.Fatalf("決定後の装備・スキル: %q", got)
		}
	})
	t.Run("TC-IF-01-41 回復は 行動 → 杖 → 相手 で、選んだ杖の回数だけ減り、装備は変えない", func(t *testing.T) {
		c := weaponMenuController(t)
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.ChooseAction(tactics.Heal)
		if c.State() != StaffMenu || !reflect.DeepEqual(c.MenuLabels(), []string{"癒やしの杖", "癒やしの杖"}) || c.RangeStyle() != StyleGood {
			t.Fatalf("杖の一覧 %v（状態 %v）", c.MenuLabels(), c.State())
		}
		c.ClickMenu(1)
		if c.State() != HealTargeting || !reflect.DeepEqual(c.HealTargets(), []int{1}) || c.RangeStyle() != StyleGood {
			t.Fatalf("回復の相手 %v（状態 %v）", c.HealTargets(), c.State())
		}
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		playOut(c)
		u := c.Battle().Units()[0]
		if u.Weapons[3].Uses != 2 || u.Weapons[0].Uses != 3 || u.Equipped != 0 {
			t.Fatalf("杖の回数 %d / %d、装備 %d", u.Weapons[0].Uses, u.Weapons[3].Uses, u.Equipped)
		}
	})
	t.Run("TC-IF-01-41 攻撃の武器が 1 本だけなら一覧を飛ばし、取り消すと行動の一覧へ戻る", func(t *testing.T) {
		c := weaponMenuController(t)
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.Click(tactics.Pos{X: 1, Y: 2})
		c.ChooseAction(tactics.Attack)
		if c.State() != Targeting || !reflect.DeepEqual(c.Targets(), []int{2}) {
			t.Fatalf("状態 %v、相手 %v", c.State(), c.Targets())
		}
		// 予測 → 相手 → 行動の一覧（一覧を飛ばしたので武器の段はない）。
		c.Click(tactics.Pos{X: 1, Y: 3})
		c.Cancel()
		if _, _, target := c.Selection(); c.State() != Targeting || target != -1 {
			t.Fatalf("1 回目の取り消しのあと %v・相手 %d", c.State(), target)
		}
		c.Cancel()
		if c.State() != Moved {
			t.Fatalf("取り消しのあと %v", c.State())
		}
	})
	t.Run("TC-IF-01-41 一覧の上の行の詳しい欄", func(t *testing.T) {
		if got := weaponDetail(menuFire); got != "ファイア　魔法　威力 5　命中 85　必殺 0　重さ 2　射程 1〜2　特効 なし" {
			t.Errorf("ファイア %q", got)
		}
		killer := menuBlade
		killer.Name, killer.EffectiveTags = "アーマーキラー", []string{"armor"}
		if got := weaponDetail(killer); got != "アーマーキラー　物理　威力 5　命中 90　必殺 0　重さ 4　射程 1　特効 鎧" {
			t.Errorf("特効 %q", got)
		}
		if got := weaponDetail(menuStaff); got != "癒やしの杖　回復 10　射程 1　残り 0/3" {
			t.Errorf("杖 %q", got)
		}
	})
}

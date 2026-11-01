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
	t.Run("TC-IF-01-41 攻撃は 行動 → 武器 → 相手 → 予測 → 決定 の順で、届かない武器は理由を出し、決定まで装備を変えない", func(t *testing.T) {
		c := weaponMenuController(t)
		history := len(c.Battle().History())
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.Click(tactics.Pos{X: 1, Y: 1})
		if c.State() != Moved || !reflect.DeepEqual(c.Actions(), []tactics.Action{tactics.Attack, tactics.Heal, tactics.Wait}) {
			t.Fatalf("行動の一覧 %v（状態 %v）", c.Actions(), c.State())
		}
		c.ChooseAction(tactics.Attack)
		if c.State() != WeaponMenu || !reflect.DeepEqual(c.MenuLabels(), []string{"ファイア", "鉄の剣（使えない）"}) {
			t.Fatalf("武器の一覧 %v（状態 %v）", c.MenuLabels(), c.State())
		}
		// 剣（射程 1）は距離 2 のボスに届かない。理由を出して一覧に残る。
		c.ClickMenu(1)
		if c.State() != WeaponMenu || !strings.Contains(c.Message(), "鉄の剣 の射程（1）に届く敵がいません") {
			t.Fatalf("届かない武器: 状態 %v、知らせ %q", c.State(), c.Message())
		}
		c.ClickMenu(0)
		if c.State() != Targeting || !reflect.DeepEqual(c.Targets(), []int{2}) {
			t.Fatalf("ファイアの相手 %v（状態 %v）", c.Targets(), c.State())
		}
		_, red := c.Overlay()
		want := map[tactics.Pos]bool{}
		for _, p := range c.Battle().WeaponRange(0, tactics.Pos{X: 1, Y: 1}, 1) {
			want[p] = true
		}
		if !reflect.DeepEqual(red, want) {
			t.Fatalf("ファイアの射程 %v", red)
		}
		c.Click(tactics.Pos{X: 1, Y: 3})
		p := c.Panel(tactics.Pos{}, false)
		if len(p.Forecast) == 0 || p.ForecastWeapons[0] != tactics.Magic || c.Battle().Units()[0].Equipped != 0 {
			t.Fatalf("予測 %v、武器 %v、装備 %d", p.Forecast, p.ForecastWeapons, c.Battle().Units()[0].Equipped)
		}
		// 取り消し: 予測 → 相手 → 武器の一覧 → 行動の一覧 → 移動先の選択（REV-072 F-01）。装備は変わらない。
		for i, want := range []struct {
			state  State
			target int
		}{{Targeting, -1}, {WeaponMenu, -1}, {Moved, -1}, {Selected, -1}} {
			c.Cancel()
			if _, _, target := c.Selection(); c.State() != want.state || target != want.target {
				t.Fatalf("取り消し %d 回目のあと %v・相手 %d、期待 %v・%d", i+1, c.State(), target, want.state, want.target)
			}
			if i == 0 && (c.weapon != 1 || len(c.Panel(tactics.Pos{}, false).Forecast) != 0) {
				t.Fatalf("1 回目の取り消しで武器 %d、予測が残る", c.weapon)
			}
		}
		if c.Battle().Units()[0].Equipped != 0 || len(c.Battle().History()) != history {
			t.Fatal("取り消しで装備か履歴が変わった")
		}
		// 決定すると、選んだファイア（添字 1）で攻撃し、装備がファイアになる。
		c.Click(tactics.Pos{X: 1, Y: 1})
		c.ChooseAction(tactics.Attack)
		c.ClickMenu(0)
		c.Click(tactics.Pos{X: 1, Y: 3})
		c.Click(tactics.Pos{X: 1, Y: 3})
		playOut(c)
		if u := c.Battle().Units()[0]; u.Equipped != 1 || !u.Acted {
			t.Fatalf("決定のあと 装備 %d、行動 %v", u.Equipped, u.Acted)
		}
		// 情報欄は装備（ファイア）と、ほかに持っている武器を列の順に出す。
		u := c.Battle().Units()[0]
		v := (Panel{Unit: &u}).View()
		textOf(t, v, "武器: ファイア（魔法）")
		textOf(t, v, "ほかの武器: 癒やしの杖、鉄の剣、癒やしの杖")
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

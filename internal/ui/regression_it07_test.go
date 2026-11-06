package ui

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// regressionSetupIT07 は小隊長・配下と両軍の ZOC を同じ盤面で試す（DES-AI-03、DES-BTL-11）。
func regressionSetupIT07() tactics.Setup {
	s := commandSetup()
	s.Units[1].ClassSkills = []tactics.Skill{zocSkill}
	s.Units = append(s.Units, tactics.UnitSetup{
		Name: "味方の重騎士", Side: tactics.Player, Stats: tactics.Stats{HP: 30, Mov: 2},
		Pos: tactics.Pos{X: 0, Y: 2}, Weapon: testLance, ClassSkills: []tactics.Skill{zocSkill},
	})
	return s
}

// equalBattleState は乱数源・非公開の履歴も比較する。乱数を引くクロージャ自体は各戦闘で別なので、
// 関数だけは nil かどうかを比較し、その参照先の rng・pcg の値は通常どおり比較する。
func equalBattleState(a, b reflect.Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return equalBattleState(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !equalBattleState(a.Field(i), b.Field(i)) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() || a.Kind() == reflect.Slice && a.IsNil() != b.IsNil() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !equalBattleState(a.Index(i), b.Index(i)) {
				return false
			}
		}
	case reflect.Map:
		if a.Len() != b.Len() || a.IsNil() != b.IsNil() {
			return false
		}
		for _, key := range a.MapKeys() {
			v := b.MapIndex(key)
			if !v.IsValid() || !equalBattleState(a.MapIndex(key), v) {
				return false
			}
		}
	case reflect.Func:
		return a.IsNil() == b.IsNil()
	default:
		return a.Equal(b)
	}
	return true
}

// 戦闘全体の比較には非公開の乱数・履歴・一団の状態も含める（DES-IT07 2・6 章）。
func TestIT07IntegratedRegression(t *testing.T) {
	t.Run("TC-NFR-06-16 下見を挟んでも小隊長撃破・崩れ・巻き戻しと再入力が一致する", func(t *testing.T) {
		viewed := newIdleWith(t, regressionSetupIT07)
		control := newIdleWith(t, regressionSetupIT07)
		check := func(where string) {
			t.Helper()
			if !equalBattleState(reflect.ValueOf(viewed.Battle()), reflect.ValueOf(control.Battle())) {
				t.Fatalf("%s: 乱数・履歴を含む戦闘状態が違う", where)
			}
		}
		for pass := 0; pass < 2; pass++ {
			viewed.ToggleDanger()
			viewed.Danger()
			viewed.ToggleDanger()
			if !viewed.OpenOverview() {
				t.Fatal("全体図が開かない")
			}
			viewed.MapOverview()
			viewed.CloseOverview()
			viewed.cursorX, viewed.cursorY = cellCenter(tactics.Pos{X: 1, Y: 1})
			if !viewed.OpenDetail() {
				t.Fatal("詳細が開かない")
			}
			viewed.DetailInput(true, false)
			viewed.DetailInput(false, true)
			check("範囲・全体図・詳細の後")
			for _, c := range []*Controller{viewed, control} {
				c.Click(tactics.Pos{X: 0, Y: 1})
				c.Click(tactics.Pos{X: 0, Y: 1})
				c.ChooseAction(tactics.Attack)
				c.Click(tactics.Pos{X: 1, Y: 1})
			}
			if _, ok := viewed.ForecastWindow(); !ok {
				t.Fatal("予測窓が開かない")
			}
			viewed.ForecastWindow()
			check("予測の後")
			for _, c := range []*Controller{viewed, control} {
				c.Click(tactics.Pos{X: 1, Y: 1})
				playOut(c)
			}
			if viewed.Battle().Units()[1].Alive || !viewed.Battle().Units()[2].Collapsed {
				t.Fatal("小隊長の撃破と配下の崩れが起きていない")
			}
			for _, c := range []*Controller{viewed, control} {
				c.EndPhase()
				c.EndPhase()
				playOut(c)
			}
			if viewed.Battle().Turn != 2 {
				t.Fatal("小隊長撃破後の敵軍の番が進んでいない")
			}
			check("小隊長撃破と敵軍の番の後")
			if pass == 0 {
				for _, c := range []*Controller{viewed, control} {
					c.OpenRewind()
					c.RewindMove(-100)
				}
				viewed.RewindBoard()
				check("巻き戻しの下見の後")
				for _, c := range []*Controller{viewed, control} {
					c.RewindEnter()
					c.RewindEnter()
					if c.rewind.open || c.State() != Idle || !c.Battle().Units()[1].Alive || c.Battle().Units()[2].Collapsed {
						t.Fatal("巻き戻しで小隊長と配下が戻らない")
					}
				}
				check("巻き戻しの後")
			}
		}
	})

	t.Run("TC-NFR-07-09 届かない移動と武器は乱数・履歴・装備・一団を変えない", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			cmd  tactics.Command
			want error
		}{
			{"ZOC 込みの移動費が移動力を超える", tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 2}, Action: tactics.Wait}, tactics.ErrUnreachable},
			{"武器の射程外", tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 1}, Action: tactics.Attack, Target: 2}, tactics.ErrOutOfRange},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := newIdleWith(t, regressionSetupIT07)
				control := newIdleWith(t, regressionSetupIT07)
				if _, err := c.Battle().Act(tc.cmd); !errors.Is(err, tc.want) {
					t.Fatalf("誤り = %v、期待 %v", err, tc.want)
				}
				if !equalBattleState(reflect.ValueOf(c.Battle()), reflect.ValueOf(control.Battle())) {
					t.Fatal("失敗した操作が戦闘状態を変えた")
				}
			})
		}
	})

	t.Run("TC-NFR-07-09 ZOC 込みで届かない再移動は待機状態と乱数を変えない", func(t *testing.T) {
		setup := func() tactics.Setup {
			s := regressionSetupIT07()
			s.Units[0].Pos = tactics.Pos{X: 0, Y: 2}
			s.Units[0].Stats.Mov = 5
			s.Units[0].ClassSkills = []tactics.Skill{{Name: "再移動", Effect: tactics.Canto}}
			s.Units[0].Weapon = testSword
			s.Units[1].Stats.HP = 30
			s.Units[5].Pos = tactics.Pos{X: 0, Y: 0}
			return s
		}
		c, control := newIdleWith(t, setup), newIdleWith(t, setup)
		for _, b := range []*tactics.Battle{c.Battle(), control.Battle()} {
			if _, err := b.Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 1}, Action: tactics.Attack, Target: 1}); err != nil {
				t.Fatal(err)
			}
			if pending, ok := b.PendingCanto(); !ok || pending.Remaining != 3 {
				t.Fatalf("前提: 再移動の残り = %v, %v、期待 3", pending, ok)
			}
		}
		if _, err := c.Battle().Act(tactics.Command{Unit: 0, To: tactics.Pos{X: 2, Y: 0}, Action: tactics.CantoMove}); !errors.Is(err, tactics.ErrUnreachable) {
			t.Fatalf("誤り = %v、期待 ErrUnreachable", err)
		}
		if !equalBattleState(reflect.ValueOf(c.Battle()), reflect.ValueOf(control.Battle())) {
			t.Fatal("失敗した再移動が戦闘状態を変えた")
		}
	})

	t.Run("TC-NFR-07-09 無効な小隊長を含む定義から戦闘を作らず入力も変えない", func(t *testing.T) {
		s, control := regressionSetupIT07(), regressionSetupIT07()
		s.Units[1].Approach, control.Units[1].Approach = tactics.Direct, tactics.Direct
		b, events, err := tactics.New(s, 1)
		if !errors.Is(err, tactics.ErrInvalidSetup) || b != nil || events != nil {
			t.Fatalf("無効な小隊長の結果 = %v, %v, %v", b, events, err)
		}
		if !reflect.DeepEqual(s, control) {
			t.Fatal("定義の拒否で入力が変わった")
		}
	})
}

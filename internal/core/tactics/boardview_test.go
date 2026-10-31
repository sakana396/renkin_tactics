package tactics

import (
	"errors"
	"reflect"
	"testing"
)

// assertBoard は既知の行動前の値と参照値を比較する（DES-RWD-05）。
func assertBoard(t *testing.T, v BoardView, s battleSnapshot) {
	t.Helper()
	if v.Turn != s.Turn || v.Phase != s.Phase || v.Result != s.Result || v.NeedsResume() != (s.Result == Ongoing && s.progress.stage != playerInput) || !reflect.DeepEqual(v.Units(), s.units) {
		t.Fatal("盤面・実際の番・勝敗・再開待ちが行動前と違う")
	}
	if v.Height() != len(s.terrain) || v.Width() != len(s.terrain[0]) {
		t.Fatal("盤面の大きさが違う")
	}
	for y, row := range s.terrain {
		for x, terrain := range row {
			if v.TerrainAt(Pos{x, y}) != terrain {
				t.Fatal("地形が違う")
			}
		}
	}
}

func TestPreviewRewindBoard(t *testing.T) {
	t.Run("TC-FR-BTL-10-21 行動前と復元後の全表示値", func(t *testing.T) {
		for _, name := range []string{"移動攻撃", "薬", "杖", "成長", "撃破"} {
			t.Run(name, func(t *testing.T) {
				b := snapshotBattle(t)
				b.units[0].ClassName = "騎兵"
				b.units[0].Weapons[0].Hit = 1000
				cmd := Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2}
				switch name {
				case "薬":
					b.units[0].HP = 40
					cmd = Command{Unit: 0, To: Pos{0, 0}, Action: UseItem, Item: 0}
				case "杖":
					b.units[1].HP = 40
					b.units[0].setWeapon(Weapon{Type: Staff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 2, MaxUses: 3})
					cmd = Command{Unit: 0, To: Pos{0, 0}, Action: Heal, Target: 1}
				case "成長":
					b.units[0].Growth = Growth{100, 100, 100, 100, 100, 0, 0, 0}
				case "撃破":
					b.units[2].HP = 1
					b.units[2].Skill = Skill{}
				}
				before := b.snapshot()
				rewindAct(t, b, cmd)
				switch name {
				case "移動攻撃":
					if b.units[0].Pos != (Pos{1, 0}) || !b.units[0].Acted {
						t.Fatal("移動攻撃を通っていない")
					}
				case "薬":
					if b.units[0].HP != 50 || b.units[0].Items[0].Count != 1 {
						t.Fatal("薬を通っていない")
					}
				case "杖":
					if b.units[1].HP != 50 || b.units[0].Weapons[0].Uses != 1 {
						t.Fatal("杖を通っていない")
					}
				case "成長":
					if b.units[0].Level != 2 || b.units[0].Stats.Str != before.units[0].Stats.Str+1 {
						t.Fatal("成長を通っていない")
					}
				case "撃破":
					if b.units[2].Alive || b.Result != Won {
						t.Fatal("撃破を通っていない")
					}
				}
				id := b.History()[0].ID
				v, err := b.PreviewRewind(id)
				if err != nil {
					t.Fatal(err)
				}
				assertBoard(t, v, before)
				rewindTo(t, b, id)
				assertBoard(t, v, b.snapshot())
				if !reflect.DeepEqual(v, b.BoardView()) {
					t.Fatal("プレビューと復元後の盤面が違う")
				}
			})
		}
	})
	t.Run("TC-FR-BTL-10-21 番の境界は履歴ラベルでなく実際の値", func(t *testing.T) {
		b := snapshotBattle(t)
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		for _, h := range b.history {
			v, err := b.PreviewRewind(h.entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertBoard(t, v, h.before)
			if h.entry.Kind == HistoryPlayerStart && (v.Turn != 1 || v.Phase != Enemy || !v.NeedsResume() || h.entry.Turn != 2) {
				t.Fatal("自軍開始前の実値と次ターンのラベルを区別できない")
			}
		}
	})
}

func TestPreviewFallenAndActed(t *testing.T) {
	t.Run("TC-FR-BTL-10-21 倒れたユニットと行動済みの添字を保持", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[1].Alive = false
		b.units[1].HP = 0
		b.units[1].Acted = true
		b.units[0].Acted = true
		before := b.snapshot()
		// 明示の番終了は行動済みも保った自軍入力待ち（DES-IT04 7.1 節）。
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		v, err := b.PreviewRewind(b.History()[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		assertBoard(t, v, before)
		rewindTo(t, b, b.History()[0].ID)
		assertBoard(t, v, b.snapshot())
		current := b.BoardView()
		if current.NeedsResume() || current.NeedsResume() != b.NeedsResume() {
			t.Fatal("番終了前への巻き戻し後の盤面が戦闘の再開待ちと一致しない")
		}
		for _, result := range []Result{Won, Lost} {
			b.Result = result
			assertBoard(t, b.BoardView(), b.snapshot())
		}
	})
}

func TestPreviewIndependence(t *testing.T) {
	t.Run("TC-FR-BTL-10-22 繰り返し閲覧と返却値変更は実PCGと履歴を変えない", func(t *testing.T) {
		b, control := snapshotBattle(t), snapshotBattle(t)
		before := b.snapshot()
		cmd := Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2}
		rewindAct(t, b, cmd)
		rewindAct(t, control, cmd)
		// 履歴自体も独立した対照戦闘と比較し、浅い写し同士の比較を避ける。
		for n := 0; n < 3; n++ {
			v, err := b.PreviewRewind(b.History()[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			units := v.Units()
			units[0].Items[0].Count = 99
			units[1] = Unit{}
			terrain := v.TerrainAt(Pos{0, 0})
			terrain.Cost = 99
			assertBoard(t, v, before)
			v.Turn = 99
			v.Phase = Enemy
			v.Result = Lost
			// パッケージ内からも、参照値の内部と元の戦闘・履歴の独立を確かめる。
			v.units[0].Items[0].Count = 88
			v.terrain[0][0].Cost = 88
			current := b.BoardView()
			current.units[0].HP = 1
			current.units[0].Items[0].Count = 88
			current.terrain[0][0].Cost = 88
			reach := b.BoardView().Reachable(0)
			clear(reach)
			attack := b.BoardView().AttackRange(0, Pos{0, 0})
			if len(attack) > 0 {
				attack[0] = Pos{99, 99}
			}
		}
		if !reflect.DeepEqual(b.snapshot(), control.snapshot()) || !reflect.DeepEqual(b.history, control.history) || b.nextHistoryID != control.nextHistoryID || b.rewindsLeft != control.rewindsLeft {
			t.Fatal("閲覧が戦闘・PCG・履歴・ID・回数・出来事境界・進行位置を変えた")
		}
		// その後の実PCGによる敵軍・成長・増援も対照戦闘と一致する。
		got, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		want, err := control.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(b.snapshot(), control.snapshot()) || !reflect.DeepEqual(b.history, control.history) {
			t.Fatal("閲覧後の同じ入力の結果が違う")
		}
		// 取得時点の盤面は、その後の元の変更からも独立する。
		v := b.BoardView()
		units := v.Units()
		b.units[0].Items[0].Count = 77
		b.units[0].HP = 1
		if !reflect.DeepEqual(v.Units(), units) {
			t.Fatal("元の変更が取得済み盤面へ伝わった")
		}
	})
}

func TestPreviewIDs(t *testing.T) {
	t.Run("TC-FR-BTL-10-23 ID検証と0回の閲覧", func(t *testing.T) {
		for _, name := range []string{"有効0回", "不正", "未実行", "破棄した未来"} {
			t.Run(name, func(t *testing.T) {
				b := snapshotBattle(t)
				rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
				id := b.History()[0].ID
				switch name {
				case "不正":
					id = 999
				case "未実行":
					b.history[0].completed = false
				case "破棄した未来":
					rewindTo(t, b, id)
				}
				b.rewindsLeft = 0
				if name == "有効0回" {
					v, err := b.PreviewRewind(id)
					if err != nil {
						t.Fatal(err)
					}
					assertBoard(t, v, b.history[0].before)
				} else {
					assertRewindFailure(t, b, ErrInvalidRewind, func() error {
						v, err := b.PreviewRewind(id)
						if !reflect.DeepEqual(v, BoardView{}) {
							t.Fatal("失敗時に古い盤面が返った")
						}
						return err
					})
				}
				assertRewindFailure(t, b, ErrNoRewinds, func() error { return b.Rewind(id) })
			})
		}
		b := snapshotBattle(t)
		if _, err := b.PreviewRewind(0); !errors.Is(err, ErrInvalidRewind) {
			t.Fatalf("履歴なし: %v", err)
		}
	})
}

func TestNewBoardView(t *testing.T) {
	t.Run("TC-FR-BTL-10-22 初期盤面は入力から独立し番処理を持たない", func(t *testing.T) {
		source := snapshotBattle(t)
		setup := Setup{Terrain: source.terrain, Reinforcements: source.reinforcements}
		for _, u := range source.Units() {
			setup.Units = append(setup.Units, UnitSetup{Name: u.Name, ClassName: u.ClassName, Side: u.Side, Stats: u.Stats, Level: u.Level, Exp: u.Exp, Growth: u.Growth, Pos: u.Pos, Weapons: u.Weapons, AllowedWeaponTypes: u.AllowedWeaponTypes, Items: u.Items, Skill: u.Skill, Lord: u.Lord, Boss: u.Boss, AI: u.AI})
		}
		setup.Units[1].Weapons = []Weapon{{Type: Staff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 1, MaxUses: 3}}
		setup.Units[1].AllowedWeaponTypes = []WeaponType{Staff}
		v, err := NewBoardView(setup)
		if err != nil {
			t.Fatal(err)
		}
		if v.Turn != 1 || v.Phase != Player || v.Result != Ongoing || v.NeedsResume() || len(v.Units()) != 3 {
			t.Fatal("初期値または未出現増援の除外が違う")
		}
		for _, u := range v.Units() {
			if u.HP != 100 || !u.Alive || u.Acted {
				t.Fatal("初期ユニットの変換が違う")
			}
		}
		if v.Units()[1].Weapons[0].Uses != 3 {
			t.Fatal("杖が満タンでない")
		}
		for _, method := range []string{"Act", "EndPhase", "Resume", "Rewind"} {
			if _, ok := reflect.TypeOf(v).MethodByName(method); ok {
				t.Fatalf("操作 %s が存在する", method)
			}
			if _, ok := reflect.TypeOf(&v).MethodByName(method); ok {
				t.Fatalf("ポインタに操作 %s が存在する", method)
			}
		}
		setup.Terrain[0][0].Cost = 99
		setup.Units[0].Items[0].Count = 99
		setup.Units[0].Stats.HP = 1
		if v.TerrainAt(Pos{0, 0}).Cost != 1 || v.Units()[0].Items[0].Count != 2 || v.Units()[0].Stats.HP != 100 {
			t.Fatal("入力変更が参照値へ伝わった")
		}
		var zero BoardView
		if zero.Width() != 0 || zero.Height() != 0 || zero.TerrainAt(Pos{}) != (Terrain{}) || zero.Reachable(0) != nil || zero.AttackRange(0, Pos{}) != nil {
			t.Fatal("ゼロ値が安全に参照できない")
		}
	})
	t.Run("TC-FR-BTL-10-23 初期盤面の不正入力はゼロ値と理由を返す", func(t *testing.T) {
		for _, s := range []Setup{{}, {Terrain: [][]Terrain{{{Cost: 1}}}}} {
			v, err := NewBoardView(s)
			if !errors.Is(err, ErrInvalidSetup) || !reflect.DeepEqual(v, BoardView{}) {
				t.Fatalf("不正入力: %v, %v", v, err)
			}
		}
	})
}

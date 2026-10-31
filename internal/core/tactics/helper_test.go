package tactics_test

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// テストで使う地形。
var (
	plain  = tactics.Terrain{Name: "平地", Cost: 1}
	forest = tactics.Terrain{Name: "森", Cost: 2, Avoid: 20, Def: 1}
	wall   = tactics.Terrain{Name: "壁", Cost: 0}
	fort   = tactics.Terrain{Name: "砦", Cost: 1, Avoid: 20, Def: 2, HealPercent: 20}
)

// grid は文字の行から地形を作る（. 平地、F 森、# 壁、T 砦）。
func grid(rows ...string) [][]tactics.Terrain {
	out := make([][]tactics.Terrain, len(rows))
	for y, row := range rows {
		for _, r := range row {
			switch r {
			case '.':
				out[y] = append(out[y], plain)
			case 'F':
				out[y] = append(out[y], forest)
			case '#':
				out[y] = append(out[y], wall)
			case 'T':
				out[y] = append(out[y], fort)
			default:
				panic("grid: 知らない文字 " + string(r))
			}
		}
	}
	return out
}

// unit は、能力がそろったユニットの入力を作る。
func unit(name string, side tactics.Side, x, y, mov int) tactics.UnitSetup {
	level := 0
	if side == tactics.Enemy {
		level = 1
	}
	return tactics.UnitSetup{
		Level:     level,
		Name:      name,
		ClassName: "テスト",
		Side:      side,
		Stats:     tactics.Stats{HP: 20, Str: 5, Skl: 5, Spd: 5, Def: 3, Mov: mov},
		Pos:       tactics.Pos{X: x, Y: y},
		Weapon:    tactics.Weapon{Name: "鉄の剣", Type: tactics.Sword, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1},
		Items:     []tactics.ItemStack{{Item: tactics.Item{Name: "回復薬", Heal: 10}, Count: 2}},
		AI:        tactics.Charge,
	}
}

func lord(x, y, mov int) tactics.UnitSetup {
	u := unit("主人公", tactics.Player, x, y, mov)
	u.Lord = true
	return u
}

func ally(name string, x, y, mov int) tactics.UnitSetup {
	return unit(name, tactics.Player, x, y, mov)
}

func boss(x, y int) tactics.UnitSetup {
	u := unit("ボス", tactics.Enemy, x, y, 0)
	u.Boss = true
	u.AI = tactics.Hold
	return u
}

func foe(name string, x, y int) tactics.UnitSetup {
	return unit(name, tactics.Enemy, x, y, 3)
}

// baseSetup は 5 × 5 の平地に、自軍 2 体と敵 2 体を置く。
//
//	0: 主人公 (0, 0) 移動 2
//	1: 味方   (0, 4) 移動 2
//	2: ボス   (4, 4)
//	3: 敵     (4, 0)
func baseSetup() tactics.Setup {
	return tactics.Setup{
		Terrain: grid(".....", ".....", ".....", ".....", "....."),
		Units: []tactics.UnitSetup{
			lord(0, 0, 2),
			ally("味方", 0, 4, 2),
			boss(4, 4),
			foe("敵", 4, 0),
		},
	}
}

// quietSetup は baseSetup の敵 (4, 0) を、移動 1 の待ち構える敵にする。
// 敵が移動できる (3, 0)・(4, 1) の隣に自軍が来なければ、攻撃の候補がなく、敵軍の番に敵は動かない（DES-AI-01）。
func quietSetup() tactics.Setup {
	s := baseSetup()
	s.Units[3].AI = tactics.Hold
	s.Units[3].Stats.Mov = 1
	return s
}

// mustNew は誤りのない戦闘を作る。
func mustNew(t *testing.T, s tactics.Setup) *tactics.Battle {
	t.Helper()
	b, _, err := tactics.New(s, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

// state は比べるための状態の写し。
type state struct {
	Turn   int
	Phase  tactics.Side
	Result tactics.Result
	Units  []tactics.Unit
}

func snapshot(b *tactics.Battle) state {
	return state{Turn: b.Turn, Phase: b.Phase, Result: b.Result, Units: b.Units()}
}

// kinds は出来事の種類の列を返す。
func kinds(events []tactics.Event) []tactics.EventKind {
	out := make([]tactics.EventKind, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}

func pos(x, y int) tactics.Pos { return tactics.Pos{X: x, Y: y} }

func assertEqual[T any](t *testing.T, name string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got  %+v\n want %+v", name, got, want)
	}
}

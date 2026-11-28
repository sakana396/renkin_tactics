package ui

import (
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestDanger234(t *testing.T) {
	t.Run("TC-FR-UNT-09-31 危険の塗りと移動範囲と縁の順", func(t *testing.T) {
		for _, state := range []State{Selected, Canto} {
			if !reflect.DeepEqual(battleUnderlayLayers(state), []boardLayer{layerDangerFill, layerOverlay}) {
				t.Fatalf("地形の上の塗りの順: %v", battleUnderlayLayers(state))
			}
		}
		if !reflect.DeepEqual(battleUnderlayLayers(Idle), []boardLayer{layerOverlay}) {
			t.Fatal("非選択時の順")
		}
		if !reflect.DeepEqual(battleLayers, []boardLayer{layerUnits, layerEnemyRanges, layerZOC, layerEquipMarks, layerWatchedPins}) {
			t.Fatalf("ユニットと縁の順: %v", battleLayers)
		}
		p := map[tactics.Pos]bool{{X: 1}: true}
		layers := enemyRangeLayers(p, p, p, p, p)
		want := []EnemyRangeLayer{{Owner: RangeAll, Stroke: StrokeFill}, {Owner: RangeWatched, Stroke: StrokeSolid}, {Owner: RangeWatched, Stroke: StrokeDashed}, {Owner: RangeCursor, Stroke: StrokeSolid}, {Owner: RangeCursor, Stroke: StrokeDashed}}
		if !reflect.DeepEqual(layerSummary(layers), want) {
			t.Fatalf("敵の層: %v", layerSummary(layers))
		}
	})
	t.Run("TC-FR-CMP-11-14 配置の全敵範囲の再利用と配置変更", func(t *testing.T) {
		s := deployRangeScenes(t)
		s.ToggleDeployDanger()
		first := s.View().Deploy.Danger
		if !reflect.DeepEqual(first, rowSet(1, 2, 3, 5, 6)) {
			t.Fatalf("初期: %v", first)
		}
		// 同じ map の再利用を、表示に使わないマスを一時的に加えて確かめる。
		marker := tactics.Pos{X: -999, Y: -999}
		first[marker] = true
		if !s.View().Deploy.Danger[marker] {
			t.Fatal("同じ配置で再利用されない")
		}
		delete(first, marker)
		clickDeployCell(s, tactics.Pos{X: 2})
		clickDeployCell(s, tactics.Pos{X: 1})
		got := s.View().Deploy.Danger
		if !reflect.DeepEqual(got, rowSet(0, 1, 2, 3, 4, 5, 6)) {
			t.Fatalf("配置変更後: %v", got)
		}
		if !reflect.DeepEqual(first, rowSet(1, 2, 3, 5, 6)) {
			t.Fatal("古い結果が書き換わった")
		}

		// 占有枠同士の交換は結果が同じでもキャッシュを作り直す。
		got[marker] = true
		clickDeployCell(s, tactics.Pos{X: 1})
		clickDeployCell(s, tactics.Pos{X: 0})
		if actual := s.View().Deploy.Danger; !reflect.DeepEqual(actual, rowSet(0, 1, 2, 3, 4, 5, 6)) {
			t.Fatalf("交換後: %v", actual)
		}
		clickSceneButton(t, s, "準備に戻る")
		if s.Scene() != Prepare || s.deployDangerCache != nil {
			t.Fatal("準備に戻った後のキャッシュ")
		}
		for _, row := range s.View().Rows {
			if row.Member == 1 {
				s.Click(row.Mark.Min.X, row.Mark.Min.Y)
				break
			}
		}
		if len(s.SelectedMembers()) != 1 {
			t.Fatal("出撃者を変更できない")
		}
		clickSceneButton(t, s, "配置へ")
		s.ToggleDeployDanger()
		if actual := s.View().Deploy.Danger; !reflect.DeepEqual(actual, rowSet(1, 2, 3, 5, 6)) {
			t.Fatalf("再配置後: %v", actual)
		}

	})
}

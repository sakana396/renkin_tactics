package ui

import (
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func TestRewindOwnView(t *testing.T) {
	t.Run("TC-FR-BTL-10-29 種類ごとの色と濃さを選択中も保つ", func(t *testing.T) {
		for _, automatic := range []bool{false, true} {
			c := newIdle(t)
			if automatic {
				for _, p := range []tactics.Pos{{X: 0, Y: 0}, {X: 0, Y: 2}} {
					c.Click(p)
					c.Click(p)
					c.ChooseAction(tactics.Wait)
					playOut(c)
				}
			} else {
				c.Click(tactics.Pos{X: 0, Y: 2})
				c.Click(tactics.Pos{X: 0, Y: 2})
				c.ChooseAction(tactics.Wait)
				playOut(c)
				endTurn(c)
			}
			c.OpenRewind()
			for i, e := range c.battle.History() {
				want := StylePlain
				allowed := true
				if e.Kind == tactics.HistoryPlayerStart {
					want = StyleRewindBoundary
					allowed = false
				} else if e.Side == tactics.Enemy || e.Automatic {
					want = StyleRewindEnemy
					allowed = false
				}
				// 選択前後の実際の一覧の表示値を調べる。
				for _, selected := range []int{0, i} {
					c.rewindSelect(selected)
					v := c.RewindView()
					found := false
					for _, tx := range v.Texts {
						if tx.Y == rewindRowTop+i*rewindRowH+6 {
							found = true
							if tx.Style != want {
								t.Fatalf("行%dの色 %v、期待 %v", i, tx.Style, want)
							}
						}
					}
					if !found {
						t.Fatal("一覧の文なし")
					}
				}
				if !c.RewindView().Rows[i].Selected || buttonOf(t, c.RewindView(), "決定").Enabled != allowed {
					t.Fatal("選択枠・決定可否")
				}
				if !allowed {
					checkReadOnlyRewind(t, c)
				} else {
					c.RewindEnter()
					if !c.rewind.confirm {
						t.Fatal("対象行の確認なし")
					}
					c.Cancel()
				}
			}
		}
		for _, tc := range []struct {
			style InfoStyle
			want  color.NRGBA
		}{{StyleRewindEnemy, color.NRGBA{255, 100, 100, 128}}, {StyleRewindBoundary, color.NRGBA{180, 180, 180, 128}}, {StylePlain, white}} {
			if got := infoStyleColor(tc.style); got != tc.want {
				t.Fatal("描画色・濃さ", got, tc.want)
			}
		}
	})
	t.Run("TC-FR-BTL-10-28 / TC-FR-BTL-10-29 最新の閲覧専用行と理由の併記・通常の案内", func(t *testing.T) {
		c := longHistory(t)
		before := c.battle.Units()
		h := c.battle.History()
		c.OpenRewind()
		checkReadOnlyRewind(t, c)
		if c.rewind.selected != len(h)-1 {
			t.Fatal("最新行でない")
		}
		c.fetchPreview(1 << 40)
		reasons := strings.Join(c.rewindReasons(), "")
		if !strings.Contains(reasons, "前だけ戻せます") || !strings.Contains(reasons, "プレビューを表示できない") {
			t.Fatal("理由を併記しない", reasons)
		}
		c.RewindDetail(1000)
		if !strings.Contains(viewStrings(c.RewindView()), "戻せません") {
			t.Fatal("理由の末尾を読めない")
		}
		c.Cancel()
		c.rewindKey("Enter")
		if c.state == Playing || !reflect.DeepEqual(before, c.battle.Units()) || !reflect.DeepEqual(h, c.battle.History()) {
			t.Fatal("通常のEnterで再開")
		}
		checkInputLayout(t, c)
		if strings.Contains(viewStrings(c.RewindView())+strings.Join(c.Guide(), ""), "再開") {
			t.Fatal("再開の案内・操作あり")
		}
	})
	t.Run("TC-FR-BTL-10-27 / TC-FR-BTL-10-29 明示の番終了から自軍行動と番終了が可能", func(t *testing.T) {
		for _, actAgain := range []bool{false, true} {
			c := newIdle(t)
			endTurn(c)
			c.OpenRewind()
			c.rewindSelect(0)
			c.RewindEnter()
			c.RewindEnter()
			if c.battle.NeedsResume() || c.confirmEnd || c.state != Idle || c.battle.RewindsLeft() != 9 {
				t.Fatal("自軍入力待ちでない")
			}
			checkInputLayout(t, c)
			if actAgain {
				c.Click(tactics.Pos{X: 0, Y: 2})
				c.Click(tactics.Pos{X: 0, Y: 2})
				c.ChooseAction(tactics.Wait)
				if c.state != Playing {
					t.Fatal("自軍行動できない")
				}
			} else {
				endTurn(c)
				if c.battle.Turn != 2 {
					t.Fatal("番終了できない")
				}
			}
		}
	})
}

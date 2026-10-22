package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 反復 2 の帯・盤面の文字・危険枠を描画から分けて確かめる（DES-UI-07）。
func TestForecastEmphasis(t *testing.T) {
	t.Run("TC-FR-BTL-08-20 撃破・危険・通常の帯と盤面の表示", func(t *testing.T) {
		for _, tc := range []struct {
			name                      string
			setup                     func() tactics.Setup
			emphasis                  ForecastEmphasis
			label, title, description string
			style                     InfoStyle
		}{
			{"通常", example1Setup, ForecastNormal, "5×2 63%", "", "", StylePlain},
			{"危険", example1HP(9), ForecastDanger, "危険 5×2 63%", "危険", "反撃で倒される", StyleBad},
			{"撃破", example1HP(20), ForecastDefeat, "撃破 5×2 63%", "撃破", "この攻撃で倒せる", StyleTitle},
		} {
			t.Run(tc.name, func(t *testing.T) {
				setup := tc.setup
				if tc.emphasis == ForecastDefeat {
					setup = func() tactics.Setup { s := tc.setup(); s.Units[1].Stats.HP = 10; return s }
				}
				c := newIdleWith(t, setup)
				target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
				p := c.Panel(tactics.Pos{X: 2, Y: 3}, true)
				if p.Emphasis != tc.emphasis {
					t.Fatalf("強調 = %v、期待 %v", p.Emphasis, tc.emphasis)
				}
				label, pos, ok := c.TargetLabel()
				if !ok || label != tc.label || pos != (tactics.Pos{X: 3, Y: 3}) {
					t.Errorf("盤面 = %q %v %v", label, pos, ok)
				}
				m := c.TargetView()
				if m.Style != tc.style || m.Text != label || m.Pos != pos || !m.Visible {
					t.Errorf("盤面の表示 = %+v", m)
				}
				if m.Danger != (tc.emphasis == ForecastDanger) || m.DangerPos != c.dest {
					t.Errorf("危険枠 = %+v", m)
				}
				wantColor := color.NRGBA{232, 237, 238, 255}
				switch tc.emphasis {
				case ForecastDanger:
					wantColor = color.NRGBA{255, 80, 80, 255}
				case ForecastDefeat:
					wantColor = color.NRGBA{255, 214, 92, 255}
				}
				if infoStyleColor(m.Style) != wantColor {
					t.Error("文字色が違う")
				}
				v := p.View()
				shift := 0
				if tc.emphasis == ForecastNormal {
					if v.Banner != nil {
						t.Errorf("通常に帯 %+v", v.Banner)
					}
				} else {
					shift = 44
					if v.Banner == nil || v.Banner.Rect != image.Rect(520, 8, 824, 48) {
						t.Fatalf("帯 = %+v", v.Banner)
					}
					title := textOf(t, v, tc.title)
					desc := textOf(t, v, tc.description)
					wantStyle := StylePlain
					if tc.emphasis == ForecastDefeat {
						wantStyle = StyleBlack
					}
					if !title.Big || title.Style != wantStyle || desc.Big || desc.Style != wantStyle {
						t.Errorf("帯の文字 = %+v %+v", title, desc)
					}
					if infoStyleColor(StyleBlack) != color.Black {
						t.Error("撃破の文が黒でない")
					}
					if v.Banner.Style != tc.style {
						t.Errorf("帯の色 = %v", v.Banner.Style)
					}
					for _, tx := range []InfoText{title, desc} {
						scale := 1
						if tx.Big {
							scale = 2
						}
						if !image.Rect(tx.X, tx.Y, tx.X+scale*textWidth(tx.Text), tx.Y+scale*lineHeight).In(v.Banner.Rect) {
							t.Errorf("帯の文がはみ出す: %+v", tx)
						}
					}
				}
				if textOf(t, v, "攻撃の予測").Y != 8+shift {
					t.Errorf("予測の位置が違う")
				}
				for _, tx := range v.Texts {
					if (tx.Text == "撃破" || tx.Text == "危険") && (!tx.Big || tx.Y >= 48) {
						t.Errorf("古い印が残る: %+v", tx)
					}
				}
			})
		}
	})
	t.Run("TC-FR-BTL-08-21 攻撃の相手を選んでいない状態と不正な予測は強調しない", func(t *testing.T) {
		c := newIdleWith(t, example1HP(9))
		target(t, c, tactics.Pos{X: 2, Y: 3}, tactics.Pos{X: 3, Y: 3})
		// 将来の回復状態も含め、Targeting 以外の状態では攻撃の予測を作らない。
		for _, state := range []State{Idle, Selected, Moved, ItemMenu, Playing, Over, State(100), Targeting} {
			c.state = state
			if state == Targeting {
				c.dest = tactics.Pos{X: 6, Y: 6}
			}
			p := c.Panel(tactics.Pos{}, false)
			if len(p.Forecast) != 0 || p.Emphasis != ForecastNormal || p.View().Banner != nil {
				t.Errorf("状態 %v: %+v", state, p)
			}
			if m := c.TargetView(); m.Visible || m.Danger {
				t.Errorf("状態 %v: %+v", state, m)
			}
		}
	})
}

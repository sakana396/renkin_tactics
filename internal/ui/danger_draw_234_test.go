package ui

import (
	"fmt"
	"image"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// drawCheck234 は実際の Game.Draw と、期待する順に独立して重ねた画素を比べる。
type drawCheck234 struct {
	game        *Game
	controllers []*Controller
	index       int
	results     []error
}

func (d *drawCheck234) Layout(_, _ int) (int, int) {
	return baseMetrics().ScreenW, baseMetrics().ScreenH
}
func (d *drawCheck234) Update() error {
	if d.index >= len(d.controllers) {
		return ebiten.Termination
	}
	return nil
}
func (d *drawCheck234) Draw(screen *ebiten.Image) {
	c := d.controllers[d.index]
	d.game.ctrl = c
	blue, _ := c.Overlay()
	danger := c.Danger()
	occupied := map[tactics.Pos]bool{}
	for _, u := range c.Units() {
		if u.Alive {
			occupied[u.Pos] = true
		}
	}
	p := tactics.Pos{X: 2, Y: 1}
	found := danger[p] && !occupied[p] && (c.state == Idle || blue[p]) && c.Camera().Visible(p)
	if !found {
		d.results = append(d.results, fmt.Errorf("状態 %v: 比較できるマスがない", c.state))
		d.index++
		return
	}
	d.game.Draw(screen)
	pt := c.Camera().Screen(d.game.metrics, p)
	x, y := pt.X+16, pt.Y+16
	actual := screen.At(x, y)
	ref := ebiten.NewImage(d.game.metrics.ScreenW, d.game.metrics.ScreenH)
	ref.Fill(bgColor)
	d.game.drawCamera = c.Camera()
	d.game.drawTerrain(ref)
	vector.DrawFilledRect(ref, float32(pt.X), float32(pt.Y), float32(d.game.metrics.Cell), float32(d.game.metrics.Cell), dangerFill, false)
	if c.state != Idle {
		vector.DrawFilledRect(ref, float32(pt.X), float32(pt.Y), float32(d.game.metrics.Cell), float32(d.game.metrics.Cell), blueOverlay, false)
	}
	ar, ag, ab, aa := actual.RGBA()
	wr, wg, wb, wa := ref.At(x, y).RGBA()
	if ar != wr || ag != wg || ab != wb || aa != wa {
		d.results = append(d.results, fmt.Errorf("状態 %v マス %v: 画素 %v、期待 %v", c.state, p, actual, ref.At(x, y)))
	} else {
		d.results = append(d.results, nil)
	}
	d.index++
}

var drawResults234 []error
var drawError234 error

func runDrawCheck234() {
	makeController := func() (*Controller, error) {
		c, e := NewController(func() (*tactics.Battle, []tactics.Event, error) { return tactics.New(cantoSetup(), 1) })
		if e == nil {
			playOut(c)
			c.ToggleDanger()
		}
		return c, e
	}
	idle, e := makeController()
	if e != nil {
		drawError234 = e
		return
	}
	selected, e := makeController()
	if e != nil {
		drawError234 = e
		return
	}
	selected.Click(tactics.Pos{X: 0, Y: 0})
	canto, e := makeController()
	if e != nil {
		drawError234 = e
		return
	}
	canto.Click(tactics.Pos{X: 0, Y: 0})
	canto.Click(tactics.Pos{X: 1, Y: 0})
	canto.ChooseAction(tactics.Attack)
	canto.Click(tactics.Pos{X: 2, Y: 0})
	canto.Click(tactics.Pos{X: 2, Y: 0})
	playOut(canto)
	if selected.State() != Selected || canto.State() != Canto {
		drawError234 = fmt.Errorf("状態の準備に失敗: %v %v", selected.State(), canto.State())
		return
	}
	cursor := image.Pt(-1, -1)
	d := &drawCheck234{game: &Game{metrics: baseMetrics(), shotCursor: &cursor}, controllers: []*Controller{idle, selected, canto}}
	ebiten.SetWindowSize(1, 1)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetWindowDecorated(false)
	drawError234 = ebiten.RunGameWithOptions(d, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true})
	drawResults234 = d.results
}
func TestDangerDraw234(t *testing.T) {
	t.Run("TC-FR-UNT-09-31 Game.Draw の画素", func(t *testing.T) {
		if os.Getenv("RENKIN_234_DRAW") == "" {
			t.Skip("RENKIN_234_DRAW=1 で描画検証を実行")
		}
		if drawError234 != nil {
			t.Fatal(drawError234)
		}
		if len(drawResults234) != 3 {
			t.Fatalf("結果数 %d", len(drawResults234))
		}
		for _, e := range drawResults234 {
			if e != nil {
				t.Error(e)
			}
		}
	})
}

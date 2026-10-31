package ui

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// ShotOptions は開発用 PNG の場面と操作列を指定する。
type ShotOptions struct {
	Scene string
	Steps []string
	Path  string
}

// RunShot はゲームと同じ描画器を使い、画面外の窓で論理画面を PNG に書き出す。
func RunShot(cfg Config, opt ShotOptions) error {
	if opt.Path == "" {
		return errors.New("出力先を指定してください")
	}
	scenes := NewSceneController(cfg)
	imgs, err := assets.Load()
	if err != nil {
		return err
	}
	g := &Game{scenes: scenes, sheets: map[assets.Sheet]*ebiten.Image{}}
	for s, img := range imgs {
		g.sheets[s] = ebiten.NewImageFromImage(img)
	}
	shot := &shotGame{Game: g, options: opt}
	ebiten.SetWindowSize(1, 1)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetWindowDecorated(false)
	if err := ebiten.RunGameWithOptions(shot, &ebiten.RunGameOptions{InitUnfocused: true, ScreenTransparent: true, SkipTaskbar: true}); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	if !shot.done {
		return errors.New("画面を描画できませんでした")
	}
	return shot.err
}

type shotGame struct {
	*Game
	options ShotOptions
	started bool
	done    bool
	err     error
}

func (g *shotGame) Update() error {
	if g.done {
		return ebiten.Termination
	}
	if g.started {
		return nil
	}
	g.started = true
	if err := g.prepareScene(); err != nil {
		return err
	}
	for n, step := range g.options.Steps {
		if err := g.step(step); err != nil {
			return fmt.Errorf("操作 %d (%s): %w", n+1, step, err)
		}
	}
	return nil
}

func (g *shotGame) Draw(_ *ebiten.Image) {
	if g.done || !g.started {
		return
	}
	img := ebiten.NewImage(ScreenWidth, ScreenHeight)
	defer img.Dispose()
	g.Game.Draw(img)
	rgba := image.NewRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	img.ReadPixels(rgba.Pix)
	f, err := os.Create(g.options.Path)
	if err == nil {
		err = png.Encode(f, rgba)
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}
	g.err = err
	g.done = true
}

func (g *shotGame) prepareScene() error {
	s := g.scenes
	switch g.options.Scene {
	case "", "title":
		return nil
	case "rule", "story", "prepare", "deploy", "battle":
	default:
		return fmt.Errorf("場面 %q は指定できません", g.options.Scene)
	}
	s.Choose(0)
	if g.options.Scene == "rule" {
		return nil
	}
	s.Choose(0)
	if g.options.Scene == "story" {
		return nil
	}
	for i := 0; s.Scene() == Story || s.Scene() == Joined; i++ {
		if i >= 100 {
			return errors.New("物語が終わりません")
		}
		s.Advance()
	}
	if s.Scene() != Prepare {
		return fmt.Errorf("準備へ進めません: %v", s.Scene())
	}
	if g.options.Scene == "prepare" {
		return nil
	}
	s.Advance()
	if s.Scene() != Deploy {
		return errors.New("配置へ進めません")
	}
	if g.options.Scene == "deploy" {
		return nil
	}
	s.Choose(0)
	if s.Scene() != Battle {
		return errors.New("戦闘へ進めません")
	}
	for i := 0; s.BattleController().State() == Playing; i++ {
		if i >= 10000 {
			return errors.New("戦闘開始の再生が終わりません")
		}
		s.BattleController().Tick()
	}
	return nil
}

func (g *shotGame) step(raw string) error {
	parts := strings.Split(raw, ":")
	s := g.scenes
	getInt := func(i int) (int, error) {
		if i >= len(parts) {
			return 0, errors.New("引数が足りません")
		}
		return strconv.Atoi(parts[i])
	}
	getPoint := func() (int, int, error) {
		x, err := getInt(1)
		if err != nil {
			return 0, 0, err
		}
		y, err := getInt(2)
		return x, y, err
	}
	battle := func() (*Controller, error) {
		if s.Scene() != Battle || s.BattleController() == nil {
			return nil, errors.New("戦闘中だけの操作です")
		}
		return s.BattleController(), nil
	}
	switch parts[0] {
	case "choose", "menu", "weapon", "tick":
		n, err := getInt(1)
		if err != nil || n < 0 {
			return errors.New("0 以上の整数を指定してください")
		}
		switch parts[0] {
		case "choose":
			s.Choose(n)
		case "menu", "weapon":
			c, err := battle()
			if err != nil {
				return err
			}
			if parts[0] == "menu" {
				c.ClickMenu(n)
			} else {
				c.ChooseWeapon(n)
			}
		case "tick":
			c, err := battle()
			if err != nil {
				return err
			}
			for range n {
				c.Tick()
			}
		}
	case "advance":
		s.Advance()
	case "cancel":
		if s.Scene() == Battle {
			s.BattleController().Cancel()
		} else {
			s.Cancel()
		}
	case "click", "cursor", "clickpx", "scroll", "target":
		x, y, err := getPoint()
		if err != nil {
			return err
		}
		switch parts[0] {
		case "clickpx":
			g.shotCursor = &image.Point{X: x, Y: y}
			if s.Scene() == Battle {
				g.ctrl = s.BattleController()
				if g.ctrl.RewindClick(x, y) {
					s.syncRewind()
				} else {
					g.leftClick(x, y)
				}
			} else {
				s.Click(x, y)
			}
		case "cursor":
			g.shotCursor = &image.Point{X: x, Y: y}
			if s.Scene() == Battle {
				s.BattleController().SetCursor(x, y)
			} else if s.Scene() == Deploy {
				s.SetDeployCursor(x, y)
			}
		case "scroll":
			if s.Scene() == Battle {
				s.BattleController().Scroll(x, y)
			} else if s.Scene() == Deploy {
				s.DeployScroll(x, y)
			} else {
				return errors.New("盤面のある場面だけの操作です")
			}
		case "click", "target":
			c, err := battle()
			if err != nil {
				return err
			}
			p := tactics.Pos{X: x, Y: y}
			if x < 0 || y < 0 || x >= c.Battle().Width() || y >= c.Battle().Height() {
				return errors.New("盤面の座標が範囲外です")
			}
			c.camera.Focus(p)
			pt := c.Camera().Screen(p)
			g.shotCursor = &image.Point{X: pt.X + CellSize/2, Y: pt.Y + CellSize/2}
			c.SetCursor(g.shotCursor.X, g.shotCursor.Y)
			c.Click(p)
		}
	case "action":
		c, err := battle()
		if err != nil {
			return err
		}
		if len(parts) != 2 {
			return errors.New("action:attack|heal|item|wait を指定してください")
		}
		var a tactics.Action
		switch parts[1] {
		case "attack":
			a = tactics.Attack
		case "heal":
			a = tactics.Heal
		case "item":
			a = tactics.UseItem
		case "wait":
			a = tactics.Wait
		default:
			return errors.New("知らない行動です")
		}
		c.ChooseAction(a)
	case "rewind":
		c, err := battle()
		if err != nil {
			return err
		}
		c.OpenRewind()
	case "rewindkey":
		if len(parts) != 2 {
			return errors.New("巻き戻しのキーを指定してください")
		}
		s.rewindKey(parts[1])
	case "danger":
		if s.Scene() == Battle {
			s.BattleController().ToggleDanger()
		} else if s.Scene() == Deploy {
			s.ToggleDeployDanger()
		} else {
			return errors.New("盤面のある場面だけの操作です")
		}
	case "endphase":
		c, err := battle()
		if err != nil {
			return err
		}
		c.EndPhase()
	default:
		return errors.New("知らない操作です")
	}
	return nil
}

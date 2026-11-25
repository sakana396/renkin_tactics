package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// discardStore は何も保存しない SaveStore（測定用）。
type discardStore227 struct{}

func (discardStore227) Exists() (bool, error) { return false, nil }
func (discardStore227) Read(func(io.Reader) error) error {
	return fmt.Errorf("セーブデータはありません")
}
func (discardStore227) Write(f func(io.Writer) error) error { return f(io.Discard) }

// drawBench227 は新作（48 px）の章の戦闘で、Q を出した状態の 1 フレーム（Update と Draw）の時間を測る（#227）。
// 環境変数 RENKIN_227_DRAW に章 ID（ch01 など）を入れて、Windows 向けの試験の実行ファイルから手で動かす。
// 結果は標準出力に出す。通常の試験ではウィンドウを開かない。
type drawBench227 struct {
	*Game
	chapter    string
	frame      int
	update, dr []time.Duration
	err        error
	lastUpdate time.Duration
}

const (
	drawBenchWarmup  = 30
	drawBenchSamples = 300
)

// drawBenchPhases は測る場面。それぞれ、操作を行ってから標本を取る。
var drawBenchPhases227 = []struct {
	name string
	set  func(g *drawBench227) error
}{
	{"Q なし", func(g *drawBench227) error { return nil }},
	{"Q あり", func(g *drawBench227) error { g.scenes.BattleController().ToggleDanger(); return nil }},
	{"Q あり・味方を選択", func(g *drawBench227) error {
		c := g.scenes.BattleController()
		for _, u := range c.Units() {
			if u.Alive && u.Side == tactics.Player && !u.Acted {
				c.Click(u.Pos)
				return nil
			}
		}
		return errors.New("味方がいない")
	}},
	{"Q あり・味方を選択・敵にカーソル", func(g *drawBench227) error {
		c := g.scenes.BattleController()
		for _, u := range c.Units() {
			if u.Alive && u.Side == tactics.Enemy && c.camera.Visible(u.Pos) {
				pt := c.camera.Screen(baseMetrics(), u.Pos)
				c.SetCursor(pt.X+baseMetrics().Cell/2, pt.Y+baseMetrics().Cell/2)
				return nil
			}
		}
		return errors.New("見える敵がいない")
	}},
}

func (g *drawBench227) Update() error {
	t0 := time.Now()
	defer func() { g.lastUpdate = time.Since(t0) }()
	per := drawBenchWarmup + drawBenchSamples
	if g.frame == 0 {
		sg := &shotGame{Game: g.Game, options: ShotOptions{Scene: "battle"}}
		if err := sg.prepareScene(); err != nil {
			return err
		}
	}
	if g.frame%per == 0 && g.frame/per < len(drawBenchPhases227) {
		if err := drawBenchPhases227[g.frame/per].set(g); err != nil {
			return err
		}
	}
	if g.frame >= len(drawBenchPhases227)*per {
		return ebiten.Termination
	}
	return nil
}

func (g *drawBench227) Draw(dst *ebiten.Image) {
	t0 := time.Now()
	g.Game.Draw(dst)
	// 描画の完了を待つため、1 画素を読む。
	_ = dst.At(0, 0)
	d := time.Since(t0)
	if g.frame%(drawBenchWarmup+drawBenchSamples) >= drawBenchWarmup {
		g.dr = append(g.dr, d)
		g.update = append(g.update, g.lastUpdate)
	}
	g.frame++
}

func median227(ds []time.Duration) time.Duration {
	c := append([]time.Duration(nil), ds...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}

func runDrawBench227(chapter string) {
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		fmt.Println("読み込み失敗:", err)
		return
	}
	cfg := Config{Catalog: cat, Media: data.AlchemyMedia(), HiRes: true, Store: discardStore227{}, NextSeed: func() uint64 { return 1 }, StartChapter: chapter}
	imgs, err := assets.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	g := &Game{metrics: metricsForConfig(cfg), scenes: NewSceneController(cfg), sheets: map[assets.Sheet]*ebiten.Image{}}
	for s, img := range imgs {
		g.sheets[s] = ebiten.NewImageFromImage(img)
	}
	b := &drawBench227{Game: g, chapter: chapter}
	ebiten.SetWindowSize(1, 1)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetWindowDecorated(false)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGameWithOptions(b, &ebiten.RunGameOptions{InitUnfocused: true, ScreenTransparent: true, SkipTaskbar: true}); err != nil && !errors.Is(err, ebiten.Termination) {
		fmt.Println("実行失敗:", err)
		return
	}
	n := drawBenchSamples
	if len(b.dr) < len(drawBenchPhases227)*n {
		fmt.Println("標本が足りない", len(b.dr))
		return
	}
	for i, ph := range drawBenchPhases227 {
		dr, up := b.dr[i*n:(i+1)*n], b.update[i*n:(i+1)*n]
		fmt.Printf("RENKIN227 %s %s: Draw 中央値 %v p95 %v 最大 %v / Update 中央値 %v 最大 %v\n", chapter, ph.name, median227(dr), pct227(dr, 95), maxDur227(dr), median227(up), maxDur227(up))
	}
	fmt.Fprint(os.Stdout, "")
}

func pct227(ds []time.Duration, p int) time.Duration {
	c := append([]time.Duration(nil), ds...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)*p/100]
}

func maxDur227(ds []time.Duration) time.Duration {
	m := ds[0]
	for _, d := range ds {
		m = max(m, d)
	}
	return m
}

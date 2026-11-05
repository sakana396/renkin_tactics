package ui

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// frameMeasureIT07 は本体の Draw を使い、描画回数と実時間を測る（DES-IT07 9.1 節）。
// 入力は固定して戦闘を進めず、下見の重い描画を連続して実行する。
type frameMeasureIT07 struct {
	deviceScale float64
	*Game
	started  time.Time
	sampled  time.Time
	lastDraw time.Time
	frames   int
	samples  []float64
	worstGap time.Duration
}

func (g *frameMeasureIT07) Update() error {
	if !g.started.IsZero() && time.Since(g.started) >= 8*time.Second {
		return ebiten.Termination
	}
	return nil
}

func (g *frameMeasureIT07) Draw(dst *ebiten.Image) {
	g.Game.Draw(dst)
	now := time.Now()
	if g.started.IsZero() {
		g.started = now
	}
	// 初期の読み込みと描画準備を含む最初の 2 秒は慣らしとする。
	if now.Sub(g.started) < 2*time.Second {
		return
	}
	if g.sampled.IsZero() {
		g.sampled, g.lastDraw = now, now
		return
	}
	g.worstGap = max(g.worstGap, now.Sub(g.lastDraw))
	g.lastDraw = now
	g.frames++
	if elapsed := now.Sub(g.sampled); elapsed >= time.Second {
		g.samples = append(g.samples, float64(g.frames)/elapsed.Seconds())
		g.sampled, g.frames = now, 0
	}
}

// TestIT07FrameTiming は基準 PC で手動起動する描画測定（TC-NFR-01-03、DES-UI-29・30・33・35）。
// 時刻は測定のためテスト側だけで使う。通常の単体テストではウィンドウを開かない。
var frameResultIT07 *frameMeasureIT07
var frameErrorIT07 error

// RunGame はメインスレッドから呼ぶ必要があるため、測定ループは TestMain で実行する。
func TestMain(m *testing.M) {
	if os.Getenv("RENKIN_IT07_FPS") != "" {
		frameResultIT07, frameErrorIT07 = runFrameTimingIT07()
	}
	os.Exit(m.Run())
}

func runFrameTimingIT07() (*frameMeasureIT07, error) {
	setup := tactics.Setup{Terrain: field(32, 32)}
	for i := 0; i < 7; i++ {
		setup.Units = append(setup.Units, tactics.UnitSetup{Name: fmt.Sprintf("自軍%d", i), Side: tactics.Player, Level: 1, Lord: i == 0, ClassName: "アーマー", Stats: tactics.Stats{HP: 30, Mov: 7}, Pos: tactics.Pos{X: 2, Y: 2 + i}, Weapon: testLance, ClassSkills: []tactics.Skill{zocSkill}})
	}
	for i := 0; i < 39; i++ {
		u := tactics.UnitSetup{Name: fmt.Sprintf("敵%d", i), ID: fmt.Sprintf("enemy%d", i), Side: tactics.Enemy, Level: 1, Boss: i == 38, ClassName: "アーマー", Stats: tactics.Stats{HP: 30, Mov: 7}, Pos: tactics.Pos{X: 6 + i%6, Y: 2 + i/6}, Weapon: testLance, AI: tactics.Hold, ClassSkills: []tactics.Skill{zocSkill}}
		if i < 38 {
			u.Group = "g"
			u.Approach = tactics.GroupApproach
			u.Commander = i == 0
		}
		setup.Units = append(setup.Units, u)
	}
	c, err := NewController(func() (*tactics.Battle, []tactics.Event, error) { return tactics.New(setup, 1) })
	if err != nil {
		return nil, err
	}
	playOut(c)
	c.ToggleDanger()
	c.watched = map[int]bool{7: true, 8: true}
	cursor := image.Pt(6*CellSize+CellSize/2, 2*CellSize+CellSize/2)
	c.SetCursor(cursor.X, cursor.Y)
	if len(c.Units()) != 46 || len(c.Danger()) == 0 || c.shownEnemy() != 7 {
		return nil, fmt.Errorf("測定条件が揃わない")
	}
	imgs, err := assets.Load()
	if err != nil {
		return nil, err
	}
	game := &Game{ctrl: c, sheets: map[assets.Sheet]*ebiten.Image{}, shotCursor: &cursor}
	for sheet, img := range imgs {
		game.sheets[sheet] = ebiten.NewImageFromImage(img)
	}
	measured := &frameMeasureIT07{Game: game, deviceScale: ebiten.DeviceScaleFactor()}
	ebiten.SetWindowTitle("レンキン・タクティクス IT07 描画測定（8秒）")
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetTPS(60)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetRunnableOnUnfocused(true)
	err = ebiten.RunGame(measured)
	return measured, err
}

func TestIT07FrameTiming(t *testing.T) {
	t.Run("TC-NFR-01-03", func(t *testing.T) {
		if os.Getenv("RENKIN_IT07_FPS") == "" {
			t.Skip("RENKIN_IT07_FPS を指定したときだけ測る")
		}
		if frameErrorIT07 != nil {
			t.Fatal(frameErrorIT07)
		}
		measured := frameResultIT07
		if measured == nil || len(measured.samples) < 4 {
			t.Fatalf("描画サンプル不足: %v", measured)
		}
		lowest := measured.samples[0]
		for _, fps := range measured.samples {
			lowest = min(lowest, fps)
		}
		// 合否は基準 PC と表示設定を記録した上で判断する。測定値を閾値に合わせて丸めない。
		t.Logf("OS=%s arch=%s Go=%s seed=1 map=32x32 player=7 enemy=39 window=%dx%d vsync=true device_scale=%.2f warmup=2s samples_fps=%v lowest_fps=%.3f worst_frame_gap=%v", runtime.GOOS, runtime.GOARCH, runtime.Version(), ScreenWidth, ScreenHeight, measured.deviceScale, measured.samples, lowest, measured.worstGap)
	})
}

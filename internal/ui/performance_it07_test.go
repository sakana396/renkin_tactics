package ui

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// frameMeasureIT07 は本体の Draw を使い、描画回数と実時間を測る（DES-IT07 9.1 節）。
// 入力は固定して戦闘を進めず、下見の重い描画を連続して実行する。
type frameMeasureIT07 struct {
	deviceScale float64
	*Game
	started   time.Time
	sampled   time.Time
	lastDraw  time.Time
	frames    int
	samples   []float64
	worstGap  time.Duration
	condition string
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
	c, condition, err := newIT07PerformanceController()
	if err != nil {
		return nil, err
	}
	cursor := image.Pt(c.cursorX, c.cursorY)
	imgs, err := assets.Load()
	if err != nil {
		return nil, err
	}
	game := &Game{ctrl: c, sheets: map[assets.Sheet]*ebiten.Image{}, shotCursor: &cursor}
	for sheet, img := range imgs {
		game.sheets[sheet] = ebiten.NewImageFromImage(img)
	}
	measured := &frameMeasureIT07{Game: game, deviceScale: ebiten.DeviceScaleFactor(), condition: condition}
	ebiten.SetWindowTitle("レンキン・タクティクス IT07 描画測定（8秒）")
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetTPS(60)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetRunnableOnUnfocused(true)
	err = ebiten.RunGame(measured)
	return measured, err
}

// newIT07PerformanceController はフレーム測定・ベンチマークで同じ盤面と表示状態を作る。
func newIT07PerformanceController() (*Controller, string, error) {
	newBattle, condition, err := it07PerformanceBattleFactory()
	if err != nil {
		return nil, "", err
	}
	c, err := NewController(newBattle)
	if err != nil {
		return nil, "", err
	}
	playOut(c)
	c.ToggleDanger()
	c.watched = map[int]bool{7: true, 8: true}
	cursor := image.Pt(6*CellSize+CellSize/2, 2*CellSize+CellSize/2)
	c.SetCursor(cursor.X, cursor.Y)
	if os.Getenv("RENKIN_IT07_FPS_CH") != "" {
		c.watched = map[int]bool{}
		first := -1
		for i, u := range c.Units() {
			if u.Alive && u.Side == tactics.Enemy {
				if first < 0 {
					first = i
					c.camera.CenterOn(u.Pos)
					pt := c.camera.Screen(u.Pos)
					c.SetCursor(pt.X+CellSize/2, pt.Y+CellSize/2)
				}
				if len(c.watched) < 2 {
					c.watched[i] = true
				}
			}
		}
		if first < 0 || c.shownEnemy() != first || len(c.watched) != 2 {
			return nil, "", fmt.Errorf("実章の固定・カーソル範囲が揃わない")
		}
	}
	if os.Getenv("RENKIN_IT07_FPS_CH") == "" && (len(c.Units()) != 46 || len(c.Danger()) == 0 || c.shownEnemy() != 7) {
		return nil, "", fmt.Errorf("合成測定条件が揃わない")
	}
	if len(c.Units()) == 0 {
		return nil, "", fmt.Errorf("%s の盤面にユニットがいない", condition)
	}
	players, enemies := 0, 0
	for _, unit := range c.Units() {
		if unit.Side == tactics.Player {
			players++
		} else {
			enemies++
		}
	}
	condition = fmt.Sprintf("%s controller_units=%d player=%d enemy=%d danger_cells=%d watched=%v cursor_enemy=%d camera=(%d,%d)", condition, len(c.Units()), players, enemies, len(c.Danger()), c.aliveWatched(), c.shownEnemy(), c.camera.X, c.camera.Y)
	return c, condition, nil
}

func it07PerformanceBattleFactory() (func() (*tactics.Battle, []tactics.Event, error), string, error) {
	if chapter := os.Getenv("RENKIN_IT07_FPS_CH"); chapter != "" {
		cat, err := defload.Load(data.Base())
		if err != nil {
			return nil, "", err
		}
		campaign, err := core.NewCampaignAt(cat, core.Withdraw, chapter)
		if err != nil {
			return nil, "", err
		}
		mapDef, ok := cat.Map(chapter)
		if !ok {
			return nil, "", fmt.Errorf("章 %s のマップがない", chapter)
		}
		members := make([]int, 0, len(mapDef.Slots))
		for i, member := range campaign.Roster {
			if i == 0 || member.Alive {
				members = append(members, i)
				if len(members) == len(mapDef.Slots) {
					break
				}
			}
		}
		if len(members) == 0 || members[0] != 0 {
			return nil, "", fmt.Errorf("%s の出撃名簿に主人公がいない", chapter)
		}
		return func() (*tactics.Battle, []tactics.Event, error) {
			sortie, events, err := campaign.StartBattle(members, 1)
			if err != nil {
				return nil, nil, err
			}
			return sortie.Battle, events, nil
		}, fmt.Sprintf("chapter=%s max_deploy=%d seed=1", chapter, len(members)), nil
	}
	setup := it07PerformanceSyntheticSetup()
	return func() (*tactics.Battle, []tactics.Event, error) { return tactics.New(setup, 1) }, "synthetic map=32x32 player=7 enemy=39 seed=1", nil
}

func it07PerformanceSyntheticSetup() tactics.Setup {
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
	return setup
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
		t.Logf("OS=%s arch=%s Go=%s input={%s} window=%dx%d vsync=true device_scale=%.2f warmup=2s samples_fps=%v lowest_fps=%.3f worst_frame_gap=%v", runtime.GOOS, runtime.GOARCH, runtime.Version(), measured.condition, ScreenWidth, ScreenHeight, measured.deviceScale, measured.samples, lowest, measured.worstGap)
	})
}

// BenchmarkIT07EnemyRangeLayers はDrawで同時に要求される敵範囲とミニマップの照会を測る。
func BenchmarkIT07EnemyRangeLayers(b *testing.B) {
	c, condition, err := newIT07PerformanceController()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.Logf("input={%s} controller_units=%d danger=true", condition, len(c.Units()))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.EnemyRangeLayers()
		_ = c.Minimap()
	}
}

// Game.Drawと同じく、盤面とミニマップの範囲を1フレーム内で共有する計算を測る（DES-UI-29）。
func BenchmarkIT07SharedFrameRanges(b *testing.B) {
	c, condition, err := newIT07PerformanceController()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.Logf("input={%s}", condition)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		danger := c.Danger()
		_ = c.enemyRangeLayersWithDanger(danger)
		_ = c.minimapWithDanger(danger)
	}
}

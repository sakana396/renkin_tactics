package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/audio"
)

// 音の結び付け（DES-NG-09）。戦闘の出来事・画面の操作・場面から、internal/ui/audio の音を鳴らす。
// 既存の入力・描画・状態は変えず、Game.Update の終わりに syncSound を 1 回呼ぶだけでつなぐ。

// MuteKey はミュートを切り替えるキー。M は全体図（DES-UI-30）が使っているので、N にした。
const MuteKey = ebiten.KeyN

// soundGlue は音の管理と、「どの出来事まで鳴らしたか」の記録（DES-NG-09）。
type soundGlue struct {
	sound *audio.Sound
	// seenCtrl・seenFirst・seenIndex は、鳴らし終えた出来事（再生中の Controller・出来事の並びの先頭・添字）。
	seenCtrl  *Controller
	seenFirst *tactics.Event
	seenIndex int
	// scene・sceneSet は、BGM を選んだ場面。
	scene    Scene
	sceneSet bool
	// over は、前のフレームで戦闘の勝敗が決まっていたか。勝敗で止めた BGM を、巻き戻しで戻ったときに選び直すために覚える。
	over bool
}

// newSoundGlue は設定の media から音を作る。media が nil なら無音（既存の作品。DES-NG-09）。
func newSoundGlue(cfg Config) soundGlue {
	if cfg.Media == nil {
		return soundGlue{}
	}
	return soundGlue{sound: audio.New(cfg.Media, audio.OpenOto, cfg.Stderr)}
}

// soundInput はそのフレームの音に関わる入力。
type soundInput struct {
	Confirm, Cancel, Select, Mute bool
}

// readSoundInput は Ebitengine の入力から soundInput を作る。
// 決定は左クリックと、物語の送り（Enter・Space）、取消しは右クリックと Esc、選択音は Tab・I・Q（表示の切り替え）。
func readSoundInput(scene Scene) soundInput {
	return soundInput{
		Confirm: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			scene == Story && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)),
		Cancel: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Select: inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyI) ||
			inpututil.IsKeyJustPressed(ebiten.KeyQ),
		Mute: inpututil.IsKeyJustPressed(MuteKey),
	}
}

// soundForEvent は戦闘の出来事に対応する効果音を返す（DES-NG-09 の対応表）。鳴らさない出来事は nil。
func soundForEvent(e tactics.Event) []audio.SE {
	switch e.Kind {
	case tactics.EventStrike:
		switch {
		case !e.Hit:
			return []audio.SE{audio.SEMiss}
		case e.Crit:
			return []audio.SE{audio.SECritical}
		}
		return []audio.SE{audio.SEHit}
	case tactics.EventHeal:
		return []audio.SE{audio.SEHeal}
	case tactics.EventBlast:
		return []audio.SE{audio.SEBlast}
	case tactics.EventLevelUp:
		return []audio.SE{audio.SELevelUp}
	case tactics.EventWon:
		return []audio.SE{audio.SEWin}
	case tactics.EventLost:
		return []audio.SE{audio.SELose}
	case tactics.EventMove:
		return []audio.SE{audio.SEStep}
	}
	return nil
}

// eventEndsBGM は、勝敗のジングルを聞かせるため BGM を止める出来事かを返す（DES-NG-09）。
func eventEndsBGM(e tactics.Event) bool {
	return e.Kind == tactics.EventWon || e.Kind == tactics.EventLost
}

// bgmForScene は場面ごとの BGM を返す（DES-NG-09 の対応表）。結果・ゲームオーバーは、勝敗のジングルのあとなので止める。
func bgmForScene(s Scene) audio.BGM {
	switch s {
	case Title, RuleSelect:
		return audio.BGMTitle
	case Story, Joined, Clear:
		return audio.BGMStory
	case Prepare, Deploy:
		return audio.BGMPrepare
	case Battle:
		return audio.BGMBattle
	}
	return audio.BGMNone
}

// syncSound は 1 フレームの終わりに呼ぶ。ミュート・操作の効果音・場面の BGM・再生中の出来事の効果音を処理する（DES-NG-09）。
// sound が nil（無音）のときは何もしない。
func (g *Game) syncSound(in soundInput) {
	sg := &g.sound
	if sg.sound == nil {
		return
	}
	sg.sound.Tick()
	if in.Mute {
		sg.sound.ToggleMute()
	}
	switch {
	case in.Cancel:
		sg.sound.Play(audio.SECancel)
	case in.Confirm:
		sg.sound.Play(audio.SEConfirm)
	case in.Select:
		sg.sound.Play(audio.SESelect)
	}
	scene := g.scenes.Scene()
	if !sg.sceneSet || sg.scene != scene {
		sg.scene, sg.sceneSet = scene, true
		sg.sound.PlayBGM(bgmForScene(scene))
	}
	// 勝敗が決まった状態から外れた（巻き戻した）とき、止めた Battle の BGM を選び直す。
	// 場面は変わらないので、場面の切り替えでは戻らない。
	c := g.scenes.BattleController()
	over := scene == Battle && battleOver(c)
	if sg.over && !over && scene == Battle {
		sg.sound.PlayBGM(bgmForScene(scene))
	}
	sg.over = over
	sg.observeEvent(c)
}

// battleOver は戦闘の勝敗が決まっていること（状態が Over、または結果が Ongoing でない。再生中の最後の出来事も含む）。
func battleOver(c *Controller) bool {
	return c != nil && (c.State() == Over || (c.Battle() != nil && c.Battle().Result != tactics.Ongoing))
}

// observeEvent は、戦闘の出来事の再生が次の出来事へ進んだとき、その出来事の効果音を鳴らす（DES-NG-09）。
func (sg *soundGlue) observeEvent(c *Controller) {
	if c == nil {
		sg.seenCtrl, sg.seenFirst = nil, nil
		return
	}
	e, _, ok := c.Event()
	if !ok {
		sg.seenCtrl, sg.seenFirst = nil, nil
		return
	}
	first := &c.events[0]
	if c == sg.seenCtrl && first == sg.seenFirst && c.index == sg.seenIndex {
		return
	}
	sg.seenCtrl, sg.seenFirst, sg.seenIndex = c, first, c.index
	for _, name := range soundForEvent(e) {
		sg.sound.Play(name)
	}
	if eventEndsBGM(e) {
		sg.sound.PlayBGM(audio.BGMNone)
	}
}

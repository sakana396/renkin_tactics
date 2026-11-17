package ui

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/ui/audio"
)

// recBackend は鳴った音を、PCM の長さで区別して記録する偽物の Backend（実際のデバイスは開かない）。
type recBackend struct {
	played []int
	bgms   []int
	stops  int
	vols   []float64
}

func (r *recBackend) PlaySE(pcm []byte, _ float64)  { r.played = append(r.played, len(pcm)) }
func (r *recBackend) PlayBGM(pcm []byte, _ float64) { r.bgms = append(r.bgms, len(pcm)) }
func (r *recBackend) StopBGM()                      { r.stops++ }
func (r *recBackend) SetBGMVolume(v float64)        { r.vols = append(r.vols, v) }
func (r *recBackend) Err() error                    { return nil }

func tinyWAV(n int) []byte {
	var b bytes.Buffer
	le := binary.LittleEndian
	b.WriteString("RIFF")
	_ = binary.Write(&b, le, uint32(36+n*2))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(22050), uint32(44100), uint16(2), uint16(16)} {
		_ = binary.Write(&b, le, v)
	}
	b.WriteString("data")
	_ = binary.Write(&b, le, uint32(n*2))
	b.Write(make([]byte, n*2))
	return b.Bytes()
}

// soundRig は、音ごとに長さの違う WAV を持つ media と偽物の Backend で作った Game の音（長さ → 名前の対応つき）。
type soundRig struct {
	g      *Game
	rec    *recBackend
	seName map[int]audio.SE
	bgmLen map[int]audio.BGM
}

func newSoundRig(t *testing.T, scenes *SceneController) *soundRig {
	t.Helper()
	media := fstest.MapFS{}
	rig := &soundRig{rec: &recBackend{}, seName: map[int]audio.SE{}, bgmLen: map[int]audio.BGM{}}
	for i, n := range audio.AllSE() {
		samples := 100 + i*10
		media[n.Path()] = &fstest.MapFile{Data: tinyWAV(samples)}
		rig.seName[samples*4] = n
	}
	for i, n := range audio.AllBGM() {
		samples := 1000 + i*100
		media[n.Path()] = &fstest.MapFile{Data: tinyWAV(samples)}
		rig.bgmLen[samples*4] = n
	}
	glue := soundGlue{sound: audio.New(media, func() (audio.Backend, error) { return rig.rec, nil }, &bytes.Buffer{})}
	rig.g = &Game{scenes: scenes, sound: glue}
	return rig
}

func (r *soundRig) names() []audio.SE {
	var out []audio.SE
	for _, n := range r.rec.played {
		out = append(out, r.seName[n])
	}
	return out
}

func (r *soundRig) bgmNames() []audio.BGM {
	var out []audio.BGM
	for _, n := range r.rec.bgms {
		out = append(out, r.bgmLen[n])
	}
	return out
}

func TestSoundForEvent(t *testing.T) {
	tests := []struct {
		name string
		e    tactics.Event
		want []audio.SE
		ends bool
	}{
		{"TC-DES-NG-09-19 命中", tactics.Event{Kind: tactics.EventStrike, Hit: true, Amount: 5}, []audio.SE{audio.SEHit}, false},
		{"TC-DES-NG-09-19 外れ", tactics.Event{Kind: tactics.EventStrike, Hit: false}, []audio.SE{audio.SEMiss}, false},
		{"TC-DES-NG-09-19 必殺（会心）", tactics.Event{Kind: tactics.EventStrike, Hit: true, Crit: true, Amount: 9}, []audio.SE{audio.SECritical}, false},
		{"TC-DES-NG-09-19 大盾で防いだ命中は命中の音", tactics.Event{Kind: tactics.EventStrike, Hit: true, Pavise: true, Amount: 1}, []audio.SE{audio.SEHit}, false},
		{"TC-DES-NG-09-19 外れは Crit が付いていても外れの音", tactics.Event{Kind: tactics.EventStrike, Hit: false, Crit: true}, []audio.SE{audio.SEMiss}, false},
		{"TC-DES-NG-09-19 回復", tactics.Event{Kind: tactics.EventHeal, Amount: 5}, []audio.SE{audio.SEHeal}, false},
		{"TC-DES-NG-09-19 爆弾の使用", tactics.Event{Kind: tactics.EventBlast}, []audio.SE{audio.SEBlast}, false},
		{"TC-DES-NG-09-19 爆発が 1 体に当たっても、爆発の音とは別に鳴らさない", tactics.Event{Kind: tactics.EventBlastHit, Amount: 8}, nil, false},
		{"TC-DES-NG-09-19 レベルアップ", tactics.Event{Kind: tactics.EventLevelUp, Level: 2}, []audio.SE{audio.SELevelUp}, false},
		{"TC-DES-NG-09-19 勝ち（BGM を止める）", tactics.Event{Kind: tactics.EventWon}, []audio.SE{audio.SEWin}, true},
		{"TC-DES-NG-09-19 負け（BGM を止める）", tactics.Event{Kind: tactics.EventLost}, []audio.SE{audio.SELose}, true},
		{"TC-DES-NG-09-19 移動", tactics.Event{Kind: tactics.EventMove}, []audio.SE{audio.SEStep}, false},
		{"TC-DES-NG-09-19 倒れる・待機・経験・番の始まり・増援は鳴らさない", tactics.Event{Kind: tactics.EventFall}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := soundForEvent(tt.e); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("音 = %v, 期待 %v", got, tt.want)
			}
			if got := eventEndsBGM(tt.e); got != tt.ends {
				t.Errorf("BGM を止める = %v, 期待 %v", got, tt.ends)
			}
		})
	}
	t.Run("TC-DES-NG-09-19 音のない出来事の種類", func(t *testing.T) {
		for _, k := range []tactics.EventKind{tactics.EventPhaseStart, tactics.EventReinforce, tactics.EventWait, tactics.EventExp, tactics.EventTrigger, tactics.EventGroupCharge, tactics.EventFall} {
			if got := soundForEvent(tactics.Event{Kind: k}); got != nil {
				t.Errorf("種類 %v に音 %v が付いた", k, got)
			}
		}
	})
}

func TestBGMForScene(t *testing.T) {
	tests := []struct {
		scene Scene
		want  audio.BGM
	}{
		{Title, audio.BGMTitle}, {RuleSelect, audio.BGMTitle},
		{Story, audio.BGMStory}, {Joined, audio.BGMStory}, {Clear, audio.BGMStory},
		{Prepare, audio.BGMPrepare}, {Deploy, audio.BGMPrepare},
		{Battle, audio.BGMBattle},
		{Result, audio.BGMNone}, {GameOver, audio.BGMNone},
	}
	for _, tt := range tests {
		t.Run("TC-DES-NG-09-20 場面の BGM", func(t *testing.T) {
			if got := bgmForScene(tt.scene); got != tt.want {
				t.Errorf("場面 %d の BGM = %q, 期待 %q", tt.scene, got, tt.want)
			}
		})
	}
}

func TestObserveEvent(t *testing.T) {
	t.Run("TC-DES-NG-09-21 出来事の再生が進むたびに、その出来事の音を 1 回ずつ鳴らす", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		if c.State() != Playing {
			t.Fatalf("状態 = %v", c.State())
		}
		var want []audio.SE
		for _, e := range c.events {
			want = append(want, soundForEvent(e)...)
		}
		if len(want) < 2 {
			t.Fatalf("期待の音が少なすぎる: %v（テストの前提が崩れた）", want)
		}
		rig := newSoundRig(t, nil)
		sg := &rig.g.sound
		for n := 0; c.State() == Playing && n < 10000; n++ {
			rig.g.sound.sound.Tick()
			sg.observeEvent(c)
			c.Tick()
		}
		if got := rig.names(); !reflect.DeepEqual(got, want) {
			t.Errorf("鳴った音 = %v, 期待 %v（再生した出来事から数えた）", got, want)
		}
	})
	t.Run("TC-DES-NG-09-21 同じ出来事を表示している間は、何度呼んでも鳴らし直さない", func(t *testing.T) {
		c := newIdle(t)
		c.EndPhase()
		c.EndPhase()
		rig := newSoundRig(t, nil)
		for i := 0; i < 30; i++ {
			rig.g.sound.sound.Tick()
			rig.g.sound.observeEvent(c) // c.Tick を呼ばないので、出来事は進まない
		}
		first := soundForEvent(c.events[0])
		if len(rig.names()) != len(first) {
			t.Errorf("鳴った数 = %d, 期待 %d", len(rig.names()), len(first))
		}
	})
	t.Run("TC-DES-NG-09-21 Controller がない・再生中でないときは鳴らさず、記録を消す", func(t *testing.T) {
		rig := newSoundRig(t, nil)
		rig.g.sound.observeEvent(nil)
		c := newIdle(t)
		rig.g.sound.observeEvent(c)
		if len(rig.names()) != 0 || rig.g.sound.seenCtrl != nil {
			t.Errorf("鳴った音 = %v", rig.names())
		}
	})
	t.Run("TC-DES-NG-09-21 勝敗の出来事で BGM を止める", func(t *testing.T) {
		rig := newSoundRig(t, nil)
		rig.g.sound.sound.PlayBGM(audio.BGMBattle)
		c := &Controller{events: []tactics.Event{{Kind: tactics.EventWon}}, state: Playing}
		rig.g.sound.observeEvent(c)
		if !reflect.DeepEqual(rig.names(), []audio.SE{audio.SEWin}) || rig.rec.stops != 1 {
			t.Errorf("音 = %v, 止めた回数 = %d", rig.names(), rig.rec.stops)
		}
	})
}

func TestSyncSound(t *testing.T) {
	t.Run("TC-DES-NG-09-22 場面が変わったときだけ BGM を切り替え、同じ場面では再開しない", func(t *testing.T) {
		s := newScenes(t, &memoryStore{})
		rig := newSoundRig(t, s)
		for i := 0; i < 5; i++ {
			rig.g.syncSound(soundInput{})
		}
		if got := rig.bgmNames(); !reflect.DeepEqual(got, []audio.BGM{audio.BGMTitle}) {
			t.Fatalf("タイトルの BGM = %v", got)
		}
		s.Choose(0) // タイトル → 難度（ルール）選択。どちらもタイトルの曲
		rig.g.syncSound(soundInput{})
		if got := rig.bgmNames(); len(got) != 1 {
			t.Errorf("同じ曲なのに再開した: %v", got)
		}
		s.Choose(0)
		rig.g.syncSound(soundInput{})
		if s.Scene() == RuleSelect {
			t.Fatal("ルール選択から進まなかった")
		}
		if got := rig.bgmNames(); len(got) != 2 || got[1] != bgmForScene(s.Scene()) {
			t.Errorf("場面 %v の BGM = %v", s.Scene(), got)
		}
	})
	t.Run("TC-DES-NG-09-27 勝敗で止めた BGM は、同じ Battle の場面のまま巻き戻すと battle が再び鳴る", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		rig := newSoundRig(t, s)
		winScene(t, s)
		c := s.BattleController()
		snd := rig.g.sound.sound
		// 勝ちの出来事を再生して、勝敗で BGM を止める。
		for n := 0; c.State() == Playing && n < 10000; n++ {
			rig.g.syncSound(soundInput{})
			c.Tick()
		}
		rig.g.syncSound(soundInput{})
		if s.Scene() != Battle || c.State() != Over {
			t.Fatalf("場面 %v、状態 %v（Battle の Over のはず）", s.Scene(), c.State())
		}
		if snd.CurrentBGM() != audio.BGMNone {
			t.Fatalf("勝敗のあとの BGM = %q（止まっているはず）", snd.CurrentBGM())
		}
		for i := 0; i < 5; i++ { // Over のあいだは戻らない
			rig.g.syncSound(soundInput{})
		}
		if snd.CurrentBGM() != audio.BGMNone {
			t.Fatalf("Over のあいだに BGM が戻った: %q", snd.CurrentBGM())
		}
		before := len(rig.rec.bgms)
		// 実際の巻き戻しの経路（一覧を開き、確認して確定する）。
		c.OpenRewind()
		c.RewindEnter()
		c.RewindEnter()
		if s.Scene() != Battle || c.State() == Over || c.Battle().Result != tactics.Ongoing {
			t.Fatalf("巻き戻せていない（場面 %v、状態 %v）", s.Scene(), c.State())
		}
		rig.g.syncSound(soundInput{})
		if snd.CurrentBGM() != audio.BGMBattle {
			t.Fatalf("巻き戻したあとの BGM = %q, 期待 %q", snd.CurrentBGM(), audio.BGMBattle)
		}
		got := rig.bgmNames()
		if len(got) != before+1 || got[len(got)-1] != audio.BGMBattle {
			t.Errorf("鳴った BGM = %v（巻き戻しで battle を 1 回だけ選び直すはず）", got)
		}
		for i := 0; i < 5; i++ { // 戻ったあとは再び選び直さない
			rig.g.syncSound(soundInput{})
		}
		if len(rig.rec.bgms) != before+1 {
			t.Errorf("BGM を余分に選び直した: %v", rig.bgmNames())
		}
	})
	t.Run("TC-DES-NG-09-23 決定・取消し・選択の入力に効果音を鳴らす（取消しが先）", func(t *testing.T) {
		tests := []struct {
			in   soundInput
			want audio.SE
		}{
			{soundInput{Confirm: true}, audio.SEConfirm},
			{soundInput{Cancel: true}, audio.SECancel},
			{soundInput{Select: true}, audio.SESelect},
			{soundInput{Confirm: true, Cancel: true}, audio.SECancel},
		}
		for _, tt := range tests {
			rig := newSoundRig(t, newScenes(t, &memoryStore{}))
			rig.g.syncSound(tt.in)
			if got := rig.names(); !reflect.DeepEqual(got, []audio.SE{tt.want}) {
				t.Errorf("入力 %+v の音 = %v, 期待 %v", tt.in, got, tt.want)
			}
		}
	})
	t.Run("TC-DES-NG-09-24 ミュートのキーは M（全体図）ではなく N。押すとミュートが切り替わる", func(t *testing.T) {
		if MuteKey != ebiten.KeyN {
			t.Fatalf("ミュートのキー = %v, 期待 N（M は全体図 DES-UI-30 のキー）", MuteKey)
		}
		rig := newSoundRig(t, newScenes(t, &memoryStore{}))
		rig.g.syncSound(soundInput{Mute: true})
		if !rig.g.sound.sound.Muted() {
			t.Fatal("ミュートにならない")
		}
		rig.g.syncSound(soundInput{Confirm: true})
		if len(rig.names()) != 0 {
			t.Errorf("ミュート中に鳴った: %v", rig.names())
		}
		rig.g.syncSound(soundInput{Mute: true})
		if rig.g.sound.sound.Muted() {
			t.Error("解除できない")
		}
	})
	t.Run("TC-DES-NG-09-25 media が nil（既存の作品）なら音の組み立てをせず、Update も音を扱わない", func(t *testing.T) {
		if g := newSoundGlue(Config{}); g.sound != nil {
			t.Error("media が nil なのに音を作った")
		}
		g := &Game{scenes: newScenes(t, &memoryStore{})}
		g.syncSound(soundInput{Confirm: true, Mute: true}) // 落ちない
	})
}

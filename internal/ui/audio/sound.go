package audio

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// 音量（0〜1）。BGM は効果音より小さくする（DES-NG-09）。
const (
	SEVolume  = 0.7
	BGMVolume = 0.3
)

// 鳴らす数の上限（DES-NG-09）。爆発で多数が巻き込まれても、うるさくならないようにする。
const (
	// MaxSEPerFrame は 1 フレームに鳴らす効果音の数の上限。
	MaxSEPerFrame = 3
	// MinSEInterval は同じ効果音をもう一度鳴らすまでの最小のフレーム数。
	MinSEInterval = 6
	// MaxVoices は同時に鳴っている効果音の数の上限（Backend が守る）。
	MaxVoices = 8
)

// SampleRate は Backend へ渡す PCM（16 bit・ステレオ）の標本化周波数。合成した WAV と同じにする。
const SampleRate = 22050

// Backend は音を出す口（DES-NG-09）。実装は oto を使うもの（OpenOto）とテスト用の偽物。
// pcm は SampleRate Hz・16 bit・ステレオ・リトルエンディアン。
type Backend interface {
	// PlaySE は効果音を 1 回鳴らす。重ねて鳴らせる。同時の数の上限（MaxVoices）を超えるときは飛ばす。
	PlaySE(pcm []byte, volume float64)
	// PlayBGM は今の BGM を止めて、pcm をループで鳴らす。
	PlayBGM(pcm []byte, volume float64)
	// StopBGM は BGM を止める。
	StopBGM()
	// SetBGMVolume は鳴っている BGM の音量を変える。
	SetBGMVolume(volume float64)
	// Err は、デバイスが使えなくなったなど、以後鳴らせない誤りを返す。なければ nil。
	Err() error
}

// Sound は音の再生の管理（DES-NG-09）。nil の *Sound は何もしない（無音）。
// 1 つの goroutine（ゲームの Update）から使う。
type Sound struct {
	media   fs.FS
	backend Backend
	stderr  io.Writer

	muted   bool
	current BGM
	frame   int
	played  int
	last    map[SE]int
	warned  map[string]bool
	se      map[SE][]byte
	bgm     map[BGM][]byte
	missing map[string]bool
}

// New は音の管理を作る。media が nil なら、デバイスを開かずに無音にする。
// open の誤り（デバイスがない・初期化に失敗）は、stderr（nil なら os.Stderr）に 1 回だけ出して無音にする。
func New(media fs.FS, open func() (Backend, error), stderr io.Writer) *Sound {
	if stderr == nil {
		stderr = os.Stderr
	}
	s := &Sound{
		media: media, stderr: stderr,
		last: map[SE]int{}, warned: map[string]bool{},
		se: map[SE][]byte{}, bgm: map[BGM][]byte{}, missing: map[string]bool{},
	}
	if media == nil || open == nil {
		return s
	}
	b, err := safeOpen(open)
	if err != nil {
		s.warn("open", "音を出せないので、無音で続けます: %v", err)
		return s
	}
	s.backend = b
	return s
}

// safeOpen は open が panic しても、誤りとして返す。
func safeOpen(open func() (Backend, error)) (b Backend, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("音のデバイスの初期化が異常終了しました: %v", r)
		}
	}()
	return open()
}

// warn は key ごとに 1 回だけ、標準エラーに知らせる。
func (s *Sound) warn(key, format string, args ...any) {
	if s.warned[key] {
		return
	}
	s.warned[key] = true
	fmt.Fprintf(s.stderr, "音: "+format+"\n", args...)
}

// Silent は、音を出せない（media がない・デバイスがない・使えなくなった）かを返す。
func (s *Sound) Silent() bool { return s == nil || s.backend == nil }

// load は WAV を読み、PCM（ステレオ 16 bit）にする。ファイルがなければ nil（誤りにしない）。壊れていれば 1 回だけ知らせて nil。
func (s *Sound) load(path string) []byte {
	if s.missing[path] {
		return nil
	}
	data, err := fs.ReadFile(s.media, path)
	if err != nil {
		s.missing[path] = true
		if !errors.Is(err, fs.ErrNotExist) {
			s.warn(path, "%s を読めないので飛ばします: %v", path, err)
		}
		return nil
	}
	stream, err := wav.DecodeWithSampleRate(SampleRate, bytesReader(data))
	var pcm []byte
	if err == nil {
		pcm, err = io.ReadAll(stream)
	}
	if err == nil && len(pcm) < 4 {
		err = errors.New("音のデータがありません")
	}
	if err != nil {
		s.missing[path] = true
		s.warn(path, "%s が壊れているので飛ばします: %v", path, err)
		return nil
	}
	return pcm[:len(pcm)/4*4]
}

// Tick は 1 フレームごとに呼ぶ。数の上限の数え直しと、デバイスの異常の確認をする。
func (s *Sound) Tick() {
	if s == nil {
		return
	}
	s.frame++
	s.played = 0
	if s.backend != nil {
		if err := s.backend.Err(); err != nil {
			s.warn("backend", "音を出せなくなったので、無音で続けます: %v", err)
			s.backend = nil
		}
	}
}

// Play は効果音を鳴らす。ミュート中・無音・ファイルなしのときは何もしない。
// 1 フレームに MaxSEPerFrame まで、同じ音は MinSEInterval フレームあけて鳴らす（飛ばした音は鳴らさない）。
func (s *Sound) Play(name SE) {
	if s.Silent() || s.muted {
		return
	}
	if s.played >= MaxSEPerFrame {
		return
	}
	if at, ok := s.last[name]; ok && s.frame-at < MinSEInterval {
		return
	}
	pcm, ok := s.se[name]
	if !ok {
		pcm = s.load(name.Path())
		s.se[name] = pcm
	}
	if pcm == nil {
		return
	}
	s.last[name] = s.frame
	s.played++
	s.backend.PlaySE(pcm, SEVolume)
}

// PlayBGM は BGM を切り替える。今と同じ曲なら何もしない（途切れさせない）。BGMNone なら止める。
// ファイルがない曲・壊れた曲は、無音にする（今の曲を止める）。ミュート中は音量 0 で鳴らし続け、解除すると続きから聞こえる。
func (s *Sound) PlayBGM(name BGM) {
	if s.Silent() || name == s.current {
		return
	}
	s.current = name
	if name == BGMNone {
		s.backend.StopBGM()
		return
	}
	pcm, ok := s.bgm[name]
	if !ok {
		pcm = s.load(name.Path())
		s.bgm[name] = pcm
	}
	if pcm == nil {
		s.backend.StopBGM()
		return
	}
	s.backend.PlayBGM(pcm, s.bgmVolume())
}

// CurrentBGM は、今選んでいる BGM（BGMNone なら止まっている）。
func (s *Sound) CurrentBGM() BGM {
	if s == nil {
		return BGMNone
	}
	return s.current
}

func (s *Sound) bgmVolume() float64 {
	if s.muted {
		return 0
	}
	return BGMVolume
}

// ToggleMute はミュートを切り替え、切り替えたあとの値（真ならミュート）を返す。
func (s *Sound) ToggleMute() bool {
	if s == nil {
		return false
	}
	s.muted = !s.muted
	if s.backend != nil {
		s.backend.SetBGMVolume(s.bgmVolume())
	}
	return s.muted
}

// Muted はミュート中かを返す。
func (s *Sound) Muted() bool { return s != nil && s.muted }

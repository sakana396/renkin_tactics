package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
)

// bytesReader は WAV のバイト列を読む口にする。
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

// otoBackend は oto（Ebitengine が使う音の出力）で鳴らす Backend（DES-NG-09）。
//
// Ebitengine の audio.Context は、デバイスの初期化の失敗を RunGame の誤りにしてゲームを止めるので使わず、
// oto を直接使って、失敗を呼び出し側が無音に変えられるようにする。
type otoBackend struct {
	ctx *oto.Context
	bgm *oto.Player
	ses []*oto.Player
}

// OpenOto は音のデバイスを開く。デバイスがない・初期化に失敗したときは誤りを返す（panic にしない）。
// 1 つのプロセスで開けるのは 1 回だけ。
func OpenOto() (b Backend, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("音のデバイスの初期化が異常終了しました: %v", r)
		}
	}()
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		return nil, errors.New("音のデバイスの準備が終わりません")
	}
	return &otoBackend{ctx: ctx}, nil
}

func (o *otoBackend) PlaySE(pcm []byte, volume float64) {
	live := o.ses[:0]
	for _, p := range o.ses {
		if p.IsPlaying() {
			live = append(live, p)
		} else {
			_ = p.Close()
		}
	}
	o.ses = live
	if len(o.ses) >= MaxVoices {
		return
	}
	p := o.ctx.NewPlayer(bytes.NewReader(pcm))
	p.SetVolume(volume)
	p.Play()
	o.ses = append(o.ses, p)
}

func (o *otoBackend) PlayBGM(pcm []byte, volume float64) {
	o.StopBGM()
	if len(pcm) == 0 {
		return
	}
	p := o.ctx.NewPlayer(&loopReader{data: pcm})
	p.SetVolume(volume)
	p.Play()
	o.bgm = p
}

func (o *otoBackend) StopBGM() {
	if o.bgm != nil {
		o.bgm.Pause()
		_ = o.bgm.Close()
		o.bgm = nil
	}
}

func (o *otoBackend) SetBGMVolume(volume float64) {
	if o.bgm != nil {
		o.bgm.SetVolume(volume)
	}
}

func (o *otoBackend) Err() error { return o.ctx.Err() }

// loopReader は data を終わりなく繰り返し読む（BGM のループ）。
type loopReader struct {
	data []byte
	pos  int
}

func (l *loopReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		c := copy(p[n:], l.data[l.pos:])
		n += c
		l.pos = (l.pos + c) % len(l.data)
	}
	return n, nil
}

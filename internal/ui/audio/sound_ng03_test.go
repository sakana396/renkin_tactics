package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// fakeBackend は鳴った音を記録するだけの偽物（実際のデバイスは開かない）。
type fakeBackend struct {
	ses       []int // 鳴らした効果音の PCM の長さ
	seVols    []float64
	bgms      []int
	bgmVols   []float64
	stops     int
	volumes   []float64
	err       error
	playCount int
}

func (f *fakeBackend) PlaySE(pcm []byte, v float64) {
	f.ses = append(f.ses, len(pcm))
	f.seVols = append(f.seVols, v)
}
func (f *fakeBackend) PlayBGM(pcm []byte, v float64) {
	f.bgms = append(f.bgms, len(pcm))
	f.bgmVols = append(f.bgmVols, v)
}
func (f *fakeBackend) StopBGM()               { f.stops++ }
func (f *fakeBackend) SetBGMVolume(v float64) { f.volumes = append(f.volumes, v) }
func (f *fakeBackend) Err() error             { return f.err }
func openFake(f *fakeBackend) func() (Backend, error) {
	return func() (Backend, error) { return f, nil }
}

// testWAV は 16 bit・モノ・22050 Hz の WAV（n 標本）を作る。
func testWAV(n int) []byte {
	var b bytes.Buffer
	le := binary.LittleEndian
	b.WriteString("RIFF")
	_ = binary.Write(&b, le, uint32(36+n*2))
	b.WriteString("WAVEfmt ")
	_ = binary.Write(&b, le, uint32(16))
	_ = binary.Write(&b, le, uint16(1))
	_ = binary.Write(&b, le, uint16(1))
	_ = binary.Write(&b, le, uint32(22050))
	_ = binary.Write(&b, le, uint32(44100))
	_ = binary.Write(&b, le, uint16(2))
	_ = binary.Write(&b, le, uint16(16))
	b.WriteString("data")
	_ = binary.Write(&b, le, uint32(n*2))
	for i := 0; i < n; i++ {
		_ = binary.Write(&b, le, int16(1000))
	}
	return b.Bytes()
}

// fullMedia は、すべての音が入った media。
func fullMedia() fstest.MapFS {
	m := fstest.MapFS{}
	for _, s := range AllSE() {
		m[s.Path()] = &fstest.MapFile{Data: testWAV(100)}
	}
	for _, b := range AllBGM() {
		m[b.Path()] = &fstest.MapFile{Data: testWAV(400)}
	}
	return m
}

func TestSoundSilent(t *testing.T) {
	t.Run("TC-DES-NG-09-06 media が nil なら、デバイスを開かず無音（stderr にも出さない）", func(t *testing.T) {
		var errOut bytes.Buffer
		opened := false
		s := New(nil, func() (Backend, error) { opened = true; return &fakeBackend{}, nil }, &errOut)
		s.Play(SEHit)
		s.PlayBGM(BGMBattle)
		s.Tick()
		s.ToggleMute()
		if opened || !s.Silent() || errOut.Len() != 0 {
			t.Errorf("opened=%v silent=%v stderr=%q", opened, s.Silent(), errOut.String())
		}
	})
	t.Run("TC-DES-NG-09-06 nil の *Sound を呼んでも落ちない", func(t *testing.T) {
		var s *Sound
		s.Play(SEHit)
		s.PlayBGM(BGMTitle)
		s.Tick()
		if s.ToggleMute() || s.Muted() || !s.Silent() || s.CurrentBGM() != BGMNone {
			t.Error("nil の Sound は無音で、ミュートにならない")
		}
	})
}

func TestSoundMissingAndBroken(t *testing.T) {
	t.Run("TC-DES-NG-09-07 ファイルがない音は、誤りにせず飛ばす", func(t *testing.T) {
		var errOut bytes.Buffer
		f := &fakeBackend{}
		s := New(fstest.MapFS{}, openFake(f), &errOut)
		s.Play(SEHit)
		s.PlayBGM(BGMBattle)
		if len(f.ses) != 0 || len(f.bgms) != 0 || errOut.Len() != 0 {
			t.Errorf("ses=%v bgms=%v stderr=%q", f.ses, f.bgms, errOut.String())
		}
	})
	t.Run("TC-DES-NG-09-08 壊れた WAV は stderr に 1 回だけ出して飛ばす", func(t *testing.T) {
		var errOut bytes.Buffer
		f := &fakeBackend{}
		m := fullMedia()
		m[SEHit.Path()] = &fstest.MapFile{Data: []byte("これは WAV ではない")}
		s := New(m, openFake(f), &errOut)
		for i := 0; i < 3; i++ {
			s.Play(SEHit)
			for j := 0; j < MinSEInterval; j++ {
				s.Tick()
			}
		}
		s.Play(SEMiss) // 壊れていない音は鳴る
		if len(f.ses) != 1 {
			t.Errorf("鳴った数 = %d, 期待 1（壊れた音は飛ばし、ほかは鳴る）", len(f.ses))
		}
		if n := strings.Count(errOut.String(), "\n"); n != 1 || !strings.Contains(errOut.String(), "hit.wav") {
			t.Errorf("stderr = %q, 期待 hit.wav について 1 行", errOut.String())
		}
	})
	t.Run("TC-DES-NG-09-08 データのない WAV と読めないファイルも、飛ばして続ける", func(t *testing.T) {
		var errOut bytes.Buffer
		f := &fakeBackend{}
		m := fullMedia()
		m[SECancel.Path()] = &fstest.MapFile{Data: testWAV(0)}
		s := New(m, openFake(f), &errOut)
		s.Play(SECancel)
		s.PlayBGM(BGMTitle)
		if len(f.ses) != 0 || len(f.bgms) != 1 {
			t.Errorf("ses=%v bgms=%v", f.ses, f.bgms)
		}
	})
}

func TestSoundDevice(t *testing.T) {
	t.Run("TC-DES-NG-09-09 デバイスを開けなくても落ちず、stderr に 1 回だけ出して無音で続ける", func(t *testing.T) {
		var errOut bytes.Buffer
		s := New(fullMedia(), func() (Backend, error) { return nil, errors.New("デバイスがありません") }, &errOut)
		for i := 0; i < 5; i++ {
			s.Play(SEHit)
			s.PlayBGM(BGMBattle)
			s.Tick()
			s.ToggleMute()
		}
		if !s.Silent() {
			t.Error("無音になっていない")
		}
		if n := strings.Count(errOut.String(), "\n"); n != 1 {
			t.Errorf("stderr の行数 = %d, 期待 1: %q", n, errOut.String())
		}
	})
	t.Run("TC-DES-NG-09-09 初期化が panic しても落ちない", func(t *testing.T) {
		var errOut bytes.Buffer
		s := New(fullMedia(), func() (Backend, error) { panic("oto") }, &errOut)
		s.Play(SEHit)
		if !s.Silent() || errOut.Len() == 0 {
			t.Errorf("silent=%v stderr=%q", s.Silent(), errOut.String())
		}
	})
	t.Run("TC-DES-NG-09-09 鳴らしている最中にデバイスが使えなくなったら、1 回だけ出して無音にする", func(t *testing.T) {
		var errOut bytes.Buffer
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &errOut)
		s.Play(SEHit)
		f.err = errors.New("消えた")
		s.Tick()
		s.Tick()
		s.Play(SEMiss)
		if len(f.ses) != 1 || !s.Silent() || strings.Count(errOut.String(), "\n") != 1 {
			t.Errorf("ses=%v silent=%v stderr=%q", f.ses, s.Silent(), errOut.String())
		}
	})
}

func TestSoundBGM(t *testing.T) {
	t.Run("TC-DES-NG-09-10 同じ曲を指定しても再開しない。別の曲へは切り替え、BGMNone で止める", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		s.PlayBGM(BGMTitle)
		s.PlayBGM(BGMTitle)
		s.PlayBGM(BGMTitle)
		if len(f.bgms) != 1 {
			t.Fatalf("同じ曲の再生回数 = %d, 期待 1", len(f.bgms))
		}
		s.PlayBGM(BGMBattle)
		if len(f.bgms) != 2 || s.CurrentBGM() != BGMBattle {
			t.Errorf("切り替え後の再生回数 = %d, 今の曲 = %q", len(f.bgms), s.CurrentBGM())
		}
		s.PlayBGM(BGMNone)
		s.PlayBGM(BGMNone)
		if f.stops != 1 || s.CurrentBGM() != BGMNone {
			t.Errorf("止めた回数 = %d, 今の曲 = %q", f.stops, s.CurrentBGM())
		}
		s.PlayBGM(BGMBattle) // 止めたあとに同じ曲を選べば、また鳴る
		if len(f.bgms) != 3 {
			t.Errorf("止めたあとの再生回数 = %d, 期待 3", len(f.bgms))
		}
	})
	t.Run("TC-DES-NG-09-11 効果音の音量より BGM の音量が小さい", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		s.PlayBGM(BGMTitle)
		s.Play(SEHit)
		if f.bgmVols[0] != BGMVolume || f.seVols[0] != SEVolume || !(BGMVolume < SEVolume) || BGMVolume <= 0 || SEVolume > 1 {
			t.Errorf("BGM %v, 効果音 %v", f.bgmVols, f.seVols)
		}
	})
	t.Run("TC-DES-NG-09-12 ファイルのない曲は、今の曲を止めて無音にする", func(t *testing.T) {
		f := &fakeBackend{}
		m := fullMedia()
		delete(m, BGMStory.Path())
		s := New(m, openFake(f), &bytes.Buffer{})
		s.PlayBGM(BGMTitle)
		s.PlayBGM(BGMStory)
		if len(f.bgms) != 1 || f.stops != 1 {
			t.Errorf("bgms=%v stops=%d", f.bgms, f.stops)
		}
	})
}

func TestSoundMute(t *testing.T) {
	t.Run("TC-DES-NG-09-13 ミュート中は効果音を鳴らさず、BGM は音量 0 になり、解除で戻る", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		s.PlayBGM(BGMTitle)
		if !s.ToggleMute() || !s.Muted() {
			t.Fatal("ミュートにならない")
		}
		s.Play(SEHit)
		if len(f.ses) != 0 {
			t.Errorf("ミュート中に鳴った: %v", f.ses)
		}
		s.PlayBGM(BGMBattle) // ミュート中の切り替えは、音量 0 で始める
		if f.bgmVols[len(f.bgmVols)-1] != 0 {
			t.Errorf("ミュート中の BGM の音量 = %v", f.bgmVols)
		}
		if s.ToggleMute() || s.Muted() {
			t.Fatal("解除できない")
		}
		s.Play(SEHit)
		if len(f.ses) != 1 {
			t.Errorf("解除後に鳴らない: %v", f.ses)
		}
		if len(f.volumes) != 2 || f.volumes[0] != 0 || f.volumes[1] != BGMVolume {
			t.Errorf("BGM の音量の変更 = %v, 期待 [0 %v]", f.volumes, BGMVolume)
		}
	})
}

func TestSoundLimits(t *testing.T) {
	t.Run("TC-DES-NG-09-14 1 フレームに鳴らす効果音は MaxSEPerFrame まで", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		for _, n := range AllSE() {
			s.Play(n)
		}
		if len(f.ses) != MaxSEPerFrame {
			t.Fatalf("鳴った数 = %d, 期待 %d", len(f.ses), MaxSEPerFrame)
		}
		s.Tick()
		s.Play(AllSE()[MaxSEPerFrame]) // 次のフレームでは、また鳴らせる
		if len(f.ses) != MaxSEPerFrame+1 {
			t.Errorf("次のフレームの鳴った数 = %d", len(f.ses))
		}
	})
	t.Run("TC-DES-NG-09-14 同じ効果音は MinSEInterval フレームあけるまで鳴らさない（境界）", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		s.Play(SEBlast)
		for i := 0; i < MinSEInterval-1; i++ {
			s.Tick()
			s.Play(SEBlast)
		}
		if len(f.ses) != 1 {
			t.Fatalf("間隔の途中で鳴った: %d", len(f.ses))
		}
		s.Tick()
		s.Play(SEBlast)
		if len(f.ses) != 2 {
			t.Errorf("間隔を過ぎても鳴らない: %d", len(f.ses))
		}
	})
	t.Run("TC-DES-NG-09-15 効果音は重ねて鳴らせる（別の音は同じフレームに続けて鳴る）", func(t *testing.T) {
		f := &fakeBackend{}
		s := New(fullMedia(), openFake(f), &bytes.Buffer{})
		s.Play(SEHit)
		s.Play(SEHeal)
		if len(f.ses) != 2 {
			t.Errorf("鳴った数 = %d, 期待 2", len(f.ses))
		}
	})
}

// TestSoundFilesMatchNames は、定数のすべてに WAV があり、ステレオ 16 bit に読め、BGM が 20〜40 秒であること（DES-NG-09）。
func TestSoundFilesMatchNames(t *testing.T) {
	media := mediaFS(t)
	f := &fakeBackend{}
	var errOut bytes.Buffer
	s := New(media, openFake(f), &errOut)
	for _, n := range AllSE() {
		n := n
		t.Run("TC-DES-NG-09-16 効果音 "+string(n)+" のファイルがあり、読める", func(t *testing.T) {
			pcm := s.load(n.Path())
			if pcm == nil {
				t.Fatalf("%s を読めない（stderr: %s）", n.Path(), errOut.String())
			}
			if sec := float64(len(pcm)) / 4 / SampleRate; sec < 0.05 || sec > 5 {
				t.Errorf("長さ %.2f 秒", sec)
			}
		})
	}
	for _, n := range AllBGM() {
		n := n
		t.Run("TC-DES-NG-09-16 BGM "+string(n)+" のファイルがあり、20〜40 秒で読める", func(t *testing.T) {
			pcm := s.load(n.Path())
			if pcm == nil {
				t.Fatalf("%s を読めない（stderr: %s）", n.Path(), errOut.String())
			}
			if sec := float64(len(pcm)) / 4 / SampleRate; sec < 20 || sec > 40 {
				t.Errorf("長さ %.2f 秒は 20〜40 秒でない", sec)
			}
		})
	}
	t.Run("TC-DES-NG-09-16 audio の下の WAV は、定数にある音だけ", func(t *testing.T) {
		known := map[string]bool{}
		for _, n := range AllSE() {
			known[n.Path()] = true
		}
		for _, n := range AllBGM() {
			known[n.Path()] = true
		}
		_ = fs.WalkDir(media, "audio", func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".wav") && !known[p] {
				t.Errorf("%s に対応する定数がない", p)
			}
			return nil
		})
	})
}

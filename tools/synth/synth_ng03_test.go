package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// samplesOf は WAV のバイト列から 16 bit の標本を取り出す。
func samplesOf(t *testing.T, b []byte) []int16 {
	t.Helper()
	if len(b) < 44 {
		t.Fatalf("WAV が短すぎる: %d バイト", len(b))
	}
	s := make([]int16, (len(b)-44)/2)
	if err := binary.Read(bytes.NewReader(b[44:]), binary.LittleEndian, s); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestSynthDeterministic は合成が決定的であること（DES-NG-09）。
func TestSynthDeterministic(t *testing.T) {
	t.Run("TC-DES-NG-09-01 同じ種で 2 回合成すると、すべての音が同じバイト列", func(t *testing.T) {
		a, b := Render(), Render()
		if len(a) != len(allAssets()) || len(a) == 0 {
			t.Fatalf("音の数 = %d, 期待 %d", len(a), len(allAssets()))
		}
		for p, x := range a {
			if !bytes.Equal(x, b[p]) {
				t.Errorf("%s: 2 回の結果が違う", p)
			}
		}
	})
	t.Run("TC-DES-NG-09-01 種が違えば、ノイズを使う音は違うバイト列になる", func(t *testing.T) {
		var hit asset
		for _, a := range allAssets() {
			if a.Path == "audio/se/hit.wav" {
				hit = a
			}
		}
		x := encodeWAV(hit.Make(newRNG(hit.Seed)).pcm())
		y := encodeWAV(hit.Make(newRNG(hit.Seed + 1)).pcm())
		if bytes.Equal(x, y) {
			t.Error("種を変えても同じ")
		}
	})
}

// TestSynthWAVFormat は WAV のヘッダ・長さ・振幅（DES-NG-09）。
func TestSynthWAVFormat(t *testing.T) {
	for p, b := range Render() {
		p, b := p, b
		t.Run("TC-DES-NG-09-02 "+p+" のヘッダと長さ", func(t *testing.T) {
			if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[12:16]) != "fmt " || string(b[36:40]) != "data" {
				t.Fatal("ヘッダの識別子が正しくない")
			}
			le := binary.LittleEndian
			if le.Uint16(b[20:]) != 1 || le.Uint16(b[22:]) != 1 || le.Uint32(b[24:]) != 22050 || le.Uint16(b[34:]) != 16 {
				t.Errorf("形式が PCM・モノ・22050 Hz・16 bit でない")
			}
			if int(le.Uint32(b[4:])) != len(b)-8 || int(le.Uint32(b[40:])) != len(b)-44 {
				t.Errorf("ヘッダの長さがファイルの大きさと合わない")
			}
		})
		t.Run("TC-DES-NG-09-03 "+p+" は無音でなく、クリップしない", func(t *testing.T) {
			s := samplesOf(t, b)
			peak := 0
			for _, v := range s {
				peak = max(peak, int(math.Abs(float64(v))))
			}
			if peak < 3000 {
				t.Errorf("最大振幅 %d は小さすぎる（無音に近い）", peak)
			}
			// 32767 に張り付く（クリップ）ことを避けるため、97% 以下にする。
			if peak > 31800 {
				t.Errorf("最大振幅 %d がクリップに近い", peak)
			}
		})
	}
}

// TestSynthBGMLoop は BGM が 20〜40 秒で、末尾から先頭へ継ぎ目なくつながること（DES-NG-09）。
func TestSynthBGMLoop(t *testing.T) {
	for p, b := range Render() {
		if !strings.HasPrefix(p, "audio/bgm/") {
			continue
		}
		p, b := p, b
		t.Run("TC-DES-NG-09-04 "+p+" の長さとループ点", func(t *testing.T) {
			s := samplesOf(t, b)
			sec := float64(len(s)) / sampleRate
			if sec < 20 || sec > 40 {
				t.Errorf("長さ %.2f 秒は 20〜40 秒でない", sec)
			}
			// 末尾の次に先頭が来る。その 1 歩の差が、曲の中の隣り合う標本の差の最大を超えないこと。
			maxStep := 0.0
			for i := 1; i < len(s); i++ {
				maxStep = math.Max(maxStep, math.Abs(float64(s[i])-float64(s[i-1])))
			}
			seam := math.Abs(float64(s[0]) - float64(s[len(s)-1]))
			if seam > maxStep {
				t.Errorf("継ぎ目の差 %.0f が、曲の中の最大の差 %.0f を超える", seam, maxStep)
			}
			// 先頭と末尾の 20 ms はほぼ無音に近い必要はないが、急な立ち上がり（プチ音）が出ないよう、継ぎ目の差は満点の 3% 未満にする。
			if seam > 0.03*32767 {
				t.Errorf("継ぎ目の差 %.0f が満点の 3%% 以上", seam)
			}
		})
	}
}

// TestSynthTotalSize は WAV の合計が 5 MB 以下であること（DES-NG-09）。
func TestSynthTotalSize(t *testing.T) {
	t.Run("TC-DES-NG-09-05 WAV の合計が 5 MB 以下", func(t *testing.T) {
		total := 0
		for _, b := range Render() {
			total += len(b)
		}
		if total > 5*1024*1024 {
			t.Errorf("合計 %d バイトが 5 MB を超える", total)
		}
	})
}

// TestSynthSETail は、効果音の末尾が急に切れないこと（DES-NG-09）。
func TestSynthSETail(t *testing.T) {
	t.Run("TC-DES-NG-09-28 すべての効果音で、最後の 10 ms の RMS が最大振幅の 2% 以下", func(t *testing.T) {
		n := sampleRate / 100
		for p, b := range Render() {
			if !strings.HasPrefix(p, "audio/se/") {
				continue
			}
			s := samplesOf(t, b)
			peak, sum := 0.0, 0.0
			for _, v := range s {
				peak = math.Max(peak, math.Abs(float64(v)))
			}
			for _, v := range s[len(s)-n:] {
				sum += float64(v) * float64(v)
			}
			if rms := math.Sqrt(sum / float64(n)); rms > 0.02*peak {
				t.Errorf("%s: 末尾 10 ms の RMS %.1f が最大 %.0f の 2%% を超える", p, rms, peak)
			}
		}
	})
	t.Run("TC-DES-NG-09-28 fadeOut は長さを変えず、最後の標本が 0 で、最後の区間だけ直線に下がる", func(t *testing.T) {
		tr := newTrack(1, false)
		for i := range tr.s {
			tr.s[i] = 1
		}
		tr.fadeOut(0.2)
		n := int(math.Round(0.2 * sampleRate))
		if len(tr.s) != sampleRate {
			t.Fatalf("長さ %d, 期待 %d", len(tr.s), sampleRate)
		}
		if tr.s[len(tr.s)-1] != 0 || tr.s[len(tr.s)-n-1] != 1 || tr.s[len(tr.s)-n] != 1 {
			t.Errorf("端の値 = %v, %v, %v", tr.s[len(tr.s)-n-1], tr.s[len(tr.s)-n], tr.s[len(tr.s)-1])
		}
		step := tr.s[len(tr.s)-n] - tr.s[len(tr.s)-n+1]
		for i := len(tr.s) - n; i+1 < len(tr.s); i++ {
			if d := tr.s[i] - tr.s[i+1]; math.Abs(d-step) > 1e-9 {
				t.Fatalf("標本 %d で傾きが一定でない: %g, %g", i, d, step)
			}
		}
	})
}

// TestSynthCommittedWAV は、コミットされた WAV が合成の出力とバイト単位で一致すること（DES-NG-09）。
// 本物の音に差し替えた行（作り方が「自作（合成」で始まらない行）は比べない。
func TestSynthCommittedWAV(t *testing.T) {
	t.Run("TC-DES-NG-09-29 台帳で自作（合成）の WAV は、tools/synth の出力と 1 バイトも違わない", func(t *testing.T) {
		media := filepath.Join("..", "..", "data", "alchemy", "media")
		raw, err := os.ReadFile(filepath.Join(media, "ASSETS.md"))
		if err != nil {
			t.Fatal(err)
		}
		synthesized := map[string]bool{}
		listed := map[string]bool{}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "|") {
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) < 2 {
				continue
			}
			file := strings.Trim(strings.TrimSpace(cells[0]), "` ")
			listed[file] = true
			if strings.HasPrefix(strings.TrimSpace(cells[1]), "自作（合成") {
				synthesized[file] = true
			}
		}
		rendered := Render()
		for p, want := range rendered {
			if !listed[p] {
				t.Errorf("%s が台帳にない", p)
				continue
			}
			if !synthesized[p] {
				continue // 本物の音に差し替えた
			}
			got, err := os.ReadFile(filepath.Join(media, filepath.FromSlash(p)))
			if err != nil {
				t.Errorf("%s を読めない: %v", p, err)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s: コミットされた WAV（%d バイト）が合成の出力（%d バイト）と違う。go run ./tools/synth で作り直す", p, len(got), len(want))
			}
		}
		for p := range synthesized {
			if _, ok := rendered[p]; !ok {
				t.Errorf("台帳の %s は自作（合成）だが、tools/synth にない", p)
			}
		}
		if len(synthesized) == 0 {
			t.Error("台帳に自作（合成）の行がない（読み取りの誤り）")
		}
	})
}

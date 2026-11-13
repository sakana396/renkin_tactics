package main

import "math"

// sampleRate は出力する WAV の標本化周波数（Hz）。16 bit・モノで書く（DES-NG-09）。
const sampleRate = 22050

// rng は、種から毎回同じ列を出す乱数（xorshift64*）。標準の math/rand に頼らず、Go の版が変わっても列が変わらないようにする。
type rng struct{ s uint64 }

func newRNG(seed uint64) *rng {
	// 種が 0 でも、状態が 0 にならないようにする。
	return &rng{s: seed*0x9E3779B97F4A7C15 + 0x1234567887654321}
}

func (r *rng) next() uint64 {
	r.s ^= r.s >> 12
	r.s ^= r.s << 25
	r.s ^= r.s >> 27
	return r.s * 0x2545F4914F6CDD1D
}

// noise は -1 以上 1 未満の一様な乱数を返す。
func (r *rng) noise() float64 { return float64(r.next()>>11)/float64(1<<52) - 1 }

// unit は 0 以上 1 未満の一様な乱数を返す。
func (r *rng) unit() float64 { return float64(r.next()>>11) / float64(1<<53) }

// waveKind は楽器の波形。
type waveKind int

const (
	waveSine waveKind = iota
	waveTri
	waveSquare
	wavePulse // デューティ比 25%
	waveSaw
	waveBell // 倍音が整数倍でない鐘の音
)

func osc(w waveKind, ph float64) float64 {
	frac := ph - math.Floor(ph)
	switch w {
	case waveTri:
		return 4*math.Abs(frac-0.5) - 1
	case waveSquare:
		if frac < 0.5 {
			return 0.7
		}
		return -0.7
	case wavePulse:
		if frac < 0.25 {
			return 0.7
		}
		return -0.7
	case waveSaw:
		return 2*frac - 1
	case waveBell:
		return (math.Sin(2*math.Pi*ph) + 0.35*math.Sin(2*math.Pi*ph*2.76) + 0.18*math.Sin(2*math.Pi*ph*5.4)) / 1.4
	}
	return math.Sin(2 * math.Pi * ph)
}

// track は 1 本の音の波形。loop が真なら、端をはみ出した音は反対の端に回り込む（BGM のループ点を継ぎ目なくするため）。
type track struct {
	s    []float64
	loop bool
}

func newTrack(sec float64, loop bool) *track {
	return &track{s: make([]float64, int(math.Round(sec*sampleRate))), loop: loop}
}

func (t *track) add(i int, v float64) {
	n := len(t.s)
	if t.loop {
		t.s[((i%n)+n)%n] += v
	} else if i >= 0 && i < n {
		t.s[i] += v
	}
}

// voice は 1 つの音色の設定。decay は指数で減る時定数（秒。0 なら減らない）、release は鳴らし終えたあとの余韻（秒）。
// detune は 2 つ目の発振器をずらす量（セント。0 なら 1 つだけ）。
type voice struct {
	wave                        waveKind
	amp, attack, decay, release float64
	detune                      float64
}

// mtof は MIDI のノート番号を周波数（Hz）にする。
func mtof(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// note は start 秒から dur 秒の間、freq の音を鳴らす。立ち上がりと余韻は最小でも数 ms 取り、プチ音を出さない。
func (t *track) note(start, dur, freq float64, v voice) {
	attack := math.Max(v.attack, 0.004)
	release := math.Max(v.release, 0.012)
	n := int((dur + release) * sampleRate)
	first := int(math.Round(start * sampleRate))
	f2 := freq * math.Pow(2, v.detune/1200)
	var ph, ph2 float64
	for k := 0; k < n; k++ {
		tt := float64(k) / sampleRate
		env := 1.0
		if tt < attack {
			env = tt / attack
		}
		if v.decay > 0 {
			env *= math.Exp(-tt / v.decay)
		}
		if tt > dur {
			env *= math.Max(0, 1-(tt-dur)/release)
		}
		s := osc(v.wave, ph)
		ph += freq / sampleRate
		if v.detune != 0 {
			s = (s + osc(v.wave, ph2)) / 2
			ph2 += f2 / sampleRate
		}
		t.add(first+k, s*env*v.amp)
	}
}

// sweep は周波数が f0 から f1 へ指数的に動く音を鳴らす。decay が正なら指数で減り、0 なら終わりへ向けて直線で減る。
func (t *track) sweep(start, dur, f0, f1 float64, w waveKind, amp, decay float64) {
	n := int(dur * sampleRate)
	first := int(math.Round(start * sampleRate))
	var ph float64
	for k := 0; k < n; k++ {
		x := float64(k) / float64(n)
		f := f0 * math.Pow(f1/f0, x)
		ph += f / sampleRate
		env := math.Min(1, float64(k)/(0.003*sampleRate))
		if decay > 0 {
			env *= math.Exp(-float64(k) / sampleRate / decay)
		} else {
			env *= 1 - x
		}
		env *= math.Min(1, (1-x)*dur/0.01)
		t.add(first+k, osc(w, ph)*env*amp)
	}
}

// noiseBurst は、2 段のローパスをかけたノイズを鳴らす。カットオフは cut0 から cut1 へ指数的に動く。
// decay が正なら指数で減り、0 なら山形（立ち上がって落ちる）の包絡を使う。
func (t *track) noiseBurst(r *rng, start, dur, amp, cut0, cut1, decay float64) {
	n := int(dur * sampleRate)
	first := int(math.Round(start * sampleRate))
	var y1, y2 float64
	for k := 0; k < n; k++ {
		x := float64(k) / float64(n)
		fc := cut0 * math.Pow(cut1/cut0, x)
		a := 1 - math.Exp(-2*math.Pi*fc/sampleRate)
		y1 += a * (r.noise() - y1)
		y2 += a * (y1 - y2)
		var env float64
		if decay > 0 {
			env = math.Exp(-float64(k) / sampleRate / decay)
		} else {
			env = math.Pow(math.Sin(math.Pi*x), 1.5)
		}
		env *= math.Min(1, float64(k)/(0.002*sampleRate)) * math.Min(1, (1-x)*dur/0.01)
		// ローパスで弱まる分を、カットオフが低いほど補う。
		gain := 1.6 / math.Sqrt(math.Max(a, 0.02))
		t.add(first+k, y2*env*amp*math.Min(gain, 6)/2)
	}
}

// hat は高い帯域のノイズ（ハイハット）を鳴らす。ノイズからローパスを引いて作る。
func (t *track) hat(r *rng, start, dur, amp float64) {
	n := int(dur * sampleRate)
	first := int(math.Round(start * sampleRate))
	a := 1 - math.Exp(-2*math.Pi*3500/sampleRate)
	var y float64
	for k := 0; k < n; k++ {
		x := r.noise()
		y += a * (x - y)
		env := math.Exp(-float64(k) / sampleRate / (dur / 4))
		env *= math.Min(1, float64(k)/(0.001*sampleRate)) * math.Min(1, float64(n-k)/(0.005*sampleRate))
		t.add(first+k, (x-y)*env*amp)
	}
}

// echo は帰還のある遅れ（こだま）を足す。loop の波形は、遅れを端で回り込ませる。
func (t *track) echo(delaySec, feedback, mix float64) {
	d := int(delaySec * sampleRate)
	n := len(t.s)
	x := append([]float64(nil), t.s...)
	y := append([]float64(nil), x...)
	passes := 1
	if t.loop {
		passes = 8
	}
	for p := 0; p < passes; p++ {
		for i := 0; i < n; i++ {
			j := i - d
			if j < 0 {
				if !t.loop {
					y[i] = x[i]
					continue
				}
				j += n
			}
			y[i] = x[i] + feedback*y[j]
		}
	}
	for i := range t.s {
		t.s[i] = x[i] + mix*(y[i]-x[i])
	}
}

// normalize は、いちばん大きい振幅を peak（0〜1）に合わせる。
func (t *track) normalize(peak float64) {
	m := 0.0
	for _, v := range t.s {
		m = math.Max(m, math.Abs(v))
	}
	if m == 0 {
		return
	}
	g := peak / m
	for i := range t.s {
		t.s[i] *= g
	}
}

// pcm は 16 bit の整数にする。normalize 済みなら、はみ出さない。
func (t *track) pcm() []int16 {
	out := make([]int16, len(t.s))
	for i, v := range t.s {
		out[i] = int16(math.Round(math.Max(-1, math.Min(1, v)) * 32767))
	}
	return out
}

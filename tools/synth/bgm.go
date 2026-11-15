package main

import "math"

// BGM の合成（DES-NG-09）。どれも本物の曲ではなく、コードで作った仮の曲である。
// 1 曲は 20 秒（小節数 × 4 拍の長さが 20 秒になる）。端をはみ出した音は反対の端へ回り込ませ、
// こだまも端で回り込ませるので、末尾から先頭へ継ぎ目なくつながる。
// 旋律・コード進行は、このためにその場で作ったもので、既存の曲の旋律は使っていない。

const bgmSeconds = 20.0

// mel は旋律の 1 音。beat は小節の頭からの拍、len は拍の長さ、midi はノート番号。
type mel struct{ beat, len, midi float64 }

func bgmAssets() []asset {
	return []asset{
		{"audio/bgm/title.wav", 2101, "タイトル。72 BPM・6 小節・D ドリアン。Dm・B♭・Gm・Dm・B♭・A のコードを、三角波のパッドと正弦波の低音で支え、8 分音符のアルペジオと鐘の旋律をこだまで包む", makeTitle},
		{"audio/bgm/story.wav", 2102, "物語。60 BPM・5 小節・E フリジアン。E の持続低音に、Em・F・Em・Dm・C の静かなパッドと、まばらな竪琴のような音、ゆっくり揺れる風のノイズ（古代都市の静けさ）", makeStory},
		{"audio/bgm/prepare.wav", 2103, "準備。96 BPM・8 小節・A マイナーから C メジャー。Am・F・C・G・Am・F・G・Em のコード。三角波の低音、矩形パルスのアルペジオ、はじく旋律、軽いハイハット", makePrepare},
		{"audio/bgm/battle.wav", 2104, "戦闘。120 BPM・10 小節・D ハーモニックマイナー。Dm・Dm・B♭・A・Dm・Dm・Gm・A・B♭・A。低い持続音、8 分音符の低音、16 分音符のアルペジオ、控えめなキックとスネア、短い旋律の断片", makeBattle},
	}
}

// chord は、小節ごとのコードのノート番号（低い順）。
type chord []float64

var (
	cDm  = chord{50, 53, 57, 62}
	cBb  = chord{46, 53, 58, 62}
	cGm  = chord{43, 50, 55, 58}
	cA   = chord{45, 52, 57, 61}
	cEm  = chord{40, 47, 52, 55}
	cF   = chord{41, 48, 53, 57}
	cC   = chord{48, 52, 55, 60}
	cG   = chord{43, 50, 55, 59}
	cAm  = chord{45, 52, 57, 60}
	cDm2 = chord{38, 45, 50, 53}
)

func makeTitle(r *rng) *track {
	const bpm, bars = 72.0, 6
	beat := 60 / bpm
	bar := beat * 4
	t := newTrack(bgmSeconds, true)
	prog := []chord{cDm, cBb, cGm, cDm, cBb, cA}
	for b, c := range prog {
		at := float64(b) * bar
		for _, m := range c[1:] {
			t.note(at, bar, mtof(m+12), voice{wave: waveTri, amp: 0.10, attack: 0.6, release: 1.0, detune: 6})
		}
		t.note(at, bar, mtof(c[0]-12), voice{wave: waveSine, amp: 0.34, attack: 0.05, decay: 2.0, release: 0.3})
		// 8 分音符のアルペジオ（コードの音を上へ・下へ）。
		pat := []int{1, 2, 3, 2, 1, 2, 3, 2}
		for i, p := range pat {
			t.note(at+float64(i)*beat/2, beat/2, mtof(c[p]+24), voice{wave: waveTri, amp: 0.13, decay: 0.3, release: 0.05})
		}
	}
	melody := [][]mel{
		{{0, 2, 74}, {2, 1, 72}, {3, 1, 69}},
		{{0, 3, 70}, {3, 1, 72}},
		{{0, 2, 67}, {2, 1, 70}, {3, 1, 74}},
		{{0, 4, 69}},
		{{0, 1.5, 70}, {1.5, 1.5, 74}, {3, 1, 77}},
		{{0, 2, 76}, {2, 2, 73}},
	}
	for b, ms := range melody {
		for _, m := range ms {
			t.note(float64(b)*bar+m.beat*beat, m.len*beat, mtof(m.midi), voice{wave: waveBell, amp: 0.2, decay: 0.9, release: 0.3})
		}
	}
	t.echo(beat*0.75, 0.4, 0.3)
	t.normalize(0.7)
	return t
}

func makeStory(r *rng) *track {
	const bpm = 60.0
	beat := 60 / bpm
	bar := beat * 4
	t := newTrack(bgmSeconds, true)
	// 持続低音。20 秒の間にゆっくり 2 回膨らむ（周期が 20 秒の約数なので、ループの継ぎ目で値が合う）。
	for k := 0; k < 4; k++ {
		at := float64(k) * 5
		t.note(at, 5, mtof(28), voice{wave: waveSine, amp: 0.35, attack: 1.5, release: 1.5})
		t.note(at, 5, mtof(35), voice{wave: waveTri, amp: 0.08, attack: 2, release: 1.5, detune: 4})
	}
	prog := []chord{cEm, cF, cEm, cDm, cC}
	for b, c := range prog {
		at := float64(b) * bar
		for _, m := range c[1:] {
			t.note(at, bar, mtof(m+12), voice{wave: waveSine, amp: 0.10, attack: 1.2, release: 1.5, detune: 5})
			t.note(at, bar, mtof(m+12), voice{wave: waveTri, amp: 0.04, attack: 1.2, release: 1.5})
		}
	}
	// まばらな竪琴のような音（E フリジアンの音から選んだ旋律）。
	harp := [][]mel{
		{{0.0, 1.5, 76}, {1.5, 1.5, 79}, {3, 1, 83}},
		{{0.5, 1.5, 81}, {2, 1.5, 77}, {3.5, 0.5, 76}},
		{{0, 2, 74}, {2.5, 1.5, 79}},
		{{1, 1.5, 77}, {2.5, 1.5, 74}},
		{{0, 1, 72}, {1, 1, 76}, {2, 2, 79}},
	}
	for b, ms := range harp {
		for _, m := range ms {
			t.note(float64(b)*bar+m.beat*beat, m.len*beat, mtof(m.midi), voice{wave: waveTri, amp: 0.20, decay: 0.8, release: 0.3})
		}
	}
	// 風のようなノイズ。振幅を 20 秒に 2 周期だけ揺らす。
	wind := newTrack(bgmSeconds, true)
	wind.noiseBurst(r, 0, bgmSeconds, 1, 500, 500, 100)
	for i := range wind.s {
		x := float64(i) / sampleRate / bgmSeconds
		t.s[i] += wind.s[i] * 0.05 * (0.55 + 0.45*math.Sin(2*math.Pi*2*x))
	}
	t.echo(0.9, 0.5, 0.4)
	t.normalize(0.7)
	return t
}

func makePrepare(r *rng) *track {
	const bpm = 96.0
	beat := 60 / bpm
	bar := beat * 4
	t := newTrack(bgmSeconds, true)
	prog := []chord{cAm, cF, cC, cG, cAm, cF, cG, cEm}
	for b, c := range prog {
		at := float64(b) * bar
		// 低音は根音と 5 度を 8 分で行き来する。
		for i := 0; i < 8; i++ {
			m := c[0]
			if i%4 == 2 {
				m = c[1]
			}
			t.note(at+float64(i)*beat/2, beat/2, mtof(m), voice{wave: waveTri, amp: 0.30, decay: 0.2, release: 0.04})
		}
		pat := []int{1, 2, 3, 2, 1, 3, 2, 3}
		for i, p := range pat {
			t.note(at+float64(i)*beat/2, beat/2, mtof(c[p]+12), voice{wave: wavePulse, amp: 0.07, decay: 0.12, release: 0.03})
		}
		for i := 0; i < 8; i++ {
			if i%2 == 1 {
				t.hat(r, at+float64(i)*beat/2, 0.05, 0.05)
			}
		}
	}
	melody := [][]mel{
		{{0, 1, 76}, {1, 1, 72}, {2, 2, 69}},
		{{0, 1, 72}, {1, 1, 69}, {2, 1, 65}, {3, 1, 69}},
		{{0, 2, 72}, {2, 1, 76}, {3, 1, 79}},
		{{0, 2, 74}, {2, 2, 71}},
		{{0, 1.5, 76}, {1.5, 0.5, 72}, {2, 2, 69}},
		{{0, 1, 69}, {1, 1, 72}, {2, 2, 77}},
		{{0, 1, 76}, {1, 1, 74}, {2, 2, 71}},
		{{0, 2, 67}, {2, 1, 71}, {3, 1, 64}},
	}
	for b, ms := range melody {
		for _, m := range ms {
			t.note(float64(b)*bar+m.beat*beat, m.len*beat, mtof(m.midi), voice{wave: waveTri, amp: 0.2, decay: 0.45, release: 0.1})
		}
	}
	t.echo(beat*0.75, 0.3, 0.2)
	t.normalize(0.7)
	return t
}

func makeBattle(r *rng) *track {
	const bpm = 120.0
	beat := 60 / bpm
	bar := beat * 4
	t := newTrack(bgmSeconds, true)
	prog := []chord{cDm2, cDm2, cBb, cA, cDm2, cDm2, cGm, cA, cBb, cA}
	// 低い持続音は、D1 を 20 秒のあいだ鳴らし続ける。
	t.note(0, bgmSeconds, mtof(26), voice{wave: waveSine, amp: 0.3, attack: 2, release: 2})
	t.note(0, bgmSeconds, mtof(33), voice{wave: waveTri, amp: 0.06, attack: 2, release: 2, detune: 7})
	for b, c := range prog {
		at := float64(b) * bar
		for i := 0; i < 8; i++ {
			m := c[0]
			if i == 3 || i == 7 {
				m = c[0] + 12
			}
			t.note(at+float64(i)*beat/2, beat/2, mtof(m), voice{wave: waveSaw, amp: 0.10, decay: 0.15, release: 0.03})
			t.note(at+float64(i)*beat/2, beat/2, mtof(m), voice{wave: waveSine, amp: 0.25, decay: 0.2, release: 0.03})
		}
		pat := []int{1, 3, 2, 3, 1, 2, 3, 2, 1, 3, 2, 3, 2, 1, 3, 2}
		for i, p := range pat {
			t.note(at+float64(i)*beat/4, beat/4, mtof(c[p]+24), voice{wave: wavePulse, amp: 0.05, decay: 0.08, release: 0.02})
		}
		// 拍の 1・3 に控えめなキック、2・4 にスネア。
		for _, k := range []float64{0, 2} {
			t.sweep(at+k*beat, 0.2, 120, 45, waveSine, 0.5, 0.08)
		}
		for _, k := range []float64{1, 3} {
			t.noiseBurst(r, at+k*beat, 0.12, 0.45, 2200, 900, 0.05)
		}
	}
	// 短い旋律の断片（D ハーモニックマイナーの音）。
	frag := map[int][]mel{
		3: {{0, 0.5, 74}, {0.5, 0.5, 73}, {1, 1, 69}, {2, 2, 73}},
		7: {{0, 0.5, 77}, {0.5, 0.5, 76}, {1, 1, 73}, {2, 2, 76}},
		9: {{0, 1, 74}, {1, 1, 73}, {2, 2, 69}},
	}
	for b, ms := range frag {
		for _, m := range ms {
			t.note(float64(b)*bar+m.beat*beat, m.len*beat, mtof(m.midi), voice{wave: waveSquare, amp: 0.10, decay: 0.5, release: 0.1, detune: 9})
		}
	}
	t.echo(beat*0.75, 0.25, 0.15)
	t.normalize(0.7)
	return t
}

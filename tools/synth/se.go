package main

// 効果音の合成（DES-NG-09）。どれも本物の音ではなく、コードで作った仮の音である。
// seed は台帳（ASSETS-audio.md）に書く乱数の種。ノイズを使わない音も、列を決めておくために持つ。

type asset struct {
	Path string // media の下の道筋（例: audio/se/hit.wav）
	Seed uint64
	Desc string // 台帳の「指示または出典」に書く、作り方の要点
	Make func(r *rng) *track
}

func seAssets() []asset {
	return []asset{
		{"audio/se/select.wav", 1101, "短い電子音 1 つ（正弦波 1320 Hz と倍音）", func(r *rng) *track {
			t := newTrack(0.12, false)
			t.note(0, 0.05, 1320, voice{wave: waveSine, amp: 0.8, decay: 0.03, release: 0.02})
			t.note(0, 0.05, 2640, voice{wave: waveSine, amp: 0.2, decay: 0.02, release: 0.02})
			t.normalize(0.5)
			return t
		}},
		{"audio/se/confirm.wav", 1102, "上がる 2 音の電子音（880 Hz → 1320 Hz、三角波）", func(r *rng) *track {
			t := newTrack(0.3, false)
			v := voice{wave: waveTri, amp: 0.8, decay: 0.12, release: 0.03}
			t.note(0, 0.07, 880, v)
			t.note(0.07, 0.12, 1320, v)
			t.normalize(0.6)
			return t
		}},
		{"audio/se/cancel.wav", 1103, "下がる 2 音の電子音（660 Hz → 440 Hz、三角波）", func(r *rng) *track {
			t := newTrack(0.3, false)
			v := voice{wave: waveTri, amp: 0.8, decay: 0.12, release: 0.03}
			t.note(0, 0.07, 660, v)
			t.note(0.07, 0.12, 440, v)
			t.normalize(0.6)
			return t
		}},
		{"audio/se/hit.wav", 1104, "ノイズの破裂（カットオフ 4000 → 600 Hz）と、200 → 60 Hz に落ちる正弦波", func(r *rng) *track {
			t := newTrack(0.3, false)
			t.noiseBurst(r, 0, 0.16, 1.0, 4000, 600, 0.045)
			t.sweep(0, 0.16, 200, 60, waveSine, 0.9, 0.06)
			t.normalize(0.85)
			return t
		}},
		{"audio/se/miss.wav", 1105, "山形の包絡のノイズ（カットオフ 800 → 3500 Hz）。空を切る音", func(r *rng) *track {
			t := newTrack(0.3, false)
			t.noiseBurst(r, 0, 0.24, 1.0, 800, 3500, 0)
			t.normalize(0.55)
			return t
		}},
		{"audio/se/critical.wav", 1106, "命中の音に、300 → 50 Hz の低音と、鐘の音 1568 Hz・2093 Hz の余韻を重ねる", func(r *rng) *track {
			t := newTrack(0.7, false)
			t.noiseBurst(r, 0, 0.2, 1.0, 6000, 500, 0.06)
			t.sweep(0, 0.22, 300, 50, waveSine, 1.0, 0.08)
			t.note(0.02, 0.4, 1568, voice{wave: waveBell, amp: 0.35, decay: 0.2, release: 0.1})
			t.note(0.06, 0.4, 2093, voice{wave: waveBell, amp: 0.25, decay: 0.2, release: 0.1})
			t.normalize(0.95)
			return t
		}},
		{"audio/se/heal.wav", 1107, "上がる 4 音（C5・E5・G5・C6、鐘の音）と高い音のきらめき", func(r *rng) *track {
			t := newTrack(0.9, false)
			for i, m := range []float64{72, 76, 79, 84} {
				t.note(float64(i)*0.08, 0.2, mtof(m), voice{wave: waveBell, amp: 0.6, decay: 0.16, release: 0.1})
			}
			t.sweep(0.1, 0.4, 1800, 3200, waveSine, 0.12, 0.15)
			t.echo(0.12, 0.3, 0.3)
			t.normalize(0.6)
			return t
		}},
		{"audio/se/blast.wav", 1108, "破裂のノイズ、140 → 30 Hz の落ちる低音、600 → 80 Hz のこもった轟き（約 1.8 秒）、細かい破片の音（乱数）", func(r *rng) *track {
			t := newTrack(2.0, false)
			t.noiseBurst(r, 0, 0.18, 1.0, 9000, 1500, 0.04)
			t.sweep(0, 1.0, 140, 30, waveSine, 1.0, 0.4)
			t.sweep(0, 0.9, 90, 28, waveTri, 0.6, 0.3)
			t.noiseBurst(r, 0.02, 1.8, 0.9, 600, 80, 0.6)
			for i := 0; i < 14; i++ {
				at := 0.25 + r.unit()*1.0
				t.noiseBurst(r, at, 0.03+r.unit()*0.04, 0.12+r.unit()*0.12, 3000+r.unit()*2000, 500, 0.015)
			}
			t.normalize(0.95)
			return t
		}},
		{"audio/se/levelup.wav", 1109, "上がる 5 音の短いファンファーレ（C5・E5・G5・C6・E6、矩形波と三角波）", func(r *rng) *track {
			t := newTrack(1.4, false)
			ms := []float64{72, 76, 79, 84, 88}
			for i, m := range ms {
				d := 0.09
				if i == len(ms)-1 {
					d = 0.6
				}
				at := float64(i) * 0.09
				t.note(at, d, mtof(m), voice{wave: waveSquare, amp: 0.3, decay: 0.5, release: 0.15})
				t.note(at, d, mtof(m), voice{wave: waveTri, amp: 0.5, decay: 0.5, release: 0.15})
			}
			t.echo(0.1, 0.3, 0.25)
			t.normalize(0.7)
			return t
		}},
		{"audio/se/win.wav", 1110, "勝利の短いジングル（上がる分散和音のあと、C メジャーの和音を伸ばす。約 2 秒）", func(r *rng) *track {
			t := newTrack(2.4, false)
			for i, m := range []float64{67, 72, 76, 79} {
				at := float64(i) * 0.15
				t.note(at, 0.15, mtof(m), voice{wave: waveTri, amp: 0.5, release: 0.05})
				t.note(at, 0.15, mtof(m), voice{wave: waveSquare, amp: 0.2, release: 0.05})
			}
			for _, m := range []float64{60, 72, 76, 79, 84} {
				t.note(0.65, 1.2, mtof(m), voice{wave: waveTri, amp: 0.4, decay: 1.2, release: 0.5})
				t.note(0.65, 1.2, mtof(m), voice{wave: waveSquare, amp: 0.1, decay: 1.0, release: 0.5, detune: 8})
			}
			t.echo(0.15, 0.3, 0.25)
			t.normalize(0.75)
			return t
		}},
		{"audio/se/lose.wav", 1111, "敗北の短いジングル（下がる 4 音のあと、低い短三和音を伸ばす。約 2.4 秒）", func(r *rng) *track {
			t := newTrack(2.8, false)
			for i, m := range []float64{64, 62, 60, 57} {
				at := float64(i) * 0.3
				t.note(at, 0.3, mtof(m), voice{wave: waveTri, amp: 0.55, decay: 0.5, release: 0.1})
			}
			for _, m := range []float64{45, 52, 57, 60} {
				t.note(1.2, 1.2, mtof(m), voice{wave: waveTri, amp: 0.4, decay: 1.0, release: 0.6, detune: 10})
			}
			t.echo(0.2, 0.35, 0.3)
			t.normalize(0.7)
			return t
		}},
		{"audio/se/step.wav", 1112, "ノイズの小さな足音（カットオフ 1200 → 300 Hz）と短い低音", func(r *rng) *track {
			t := newTrack(0.1, false)
			t.noiseBurst(r, 0, 0.05, 1.0, 1200, 300, 0.015)
			t.sweep(0, 0.05, 140, 90, waveSine, 0.6, 0.02)
			t.normalize(0.4)
			return t
		}},
	}
}

package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// pngBytes は w × h の PNG を作る。alpha が 0 以外なら (0, 0) をその透明度の赤にする。
func pngBytes(t *testing.T, w, h int, alpha uint8) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: alpha})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func alchemyCatalog(t *testing.T) *defs.Catalog {
	t.Helper()
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// 画像の読み込み（DES-NG-08）。ない・壊れている画像は「なし」で、ゲームを止めず、誤りは 1 回だけ出す。
func TestArtStoreLoadNG02(t *testing.T) {
	good := pngBytes(t, 128, 128, 255)
	tests := []struct {
		name      string
		media     fstest.MapFS
		path      string
		square    bool
		wantOK    bool
		wantW     int
		wantStder string // 空なら stderr は空
	}{
		{"TC-DES-NG-08-20 正方形の PNG を読める", fstest.MapFS{"art/units/aoi.png": {Data: good}}, "art/units/aoi.png", true, true, 128, ""},
		{"TC-DES-NG-08-21 ファイルがなければ「なし」で、stderr には何も出さない", fstest.MapFS{}, "art/units/aoi.png", true, false, 0, ""},
		{"TC-DES-NG-08-22 壊れた PNG は「なし」で、誤りを stderr に出す", fstest.MapFS{"art/units/aoi.png": {Data: []byte("not a png")}}, "art/units/aoi.png", true, false, 0, "画像を読めません（art/units/aoi.png）"},
		{"TC-DES-NG-08-23 非正方形は使えるが、正方形でないと知らせる（縦横比は保って描く）", fstest.MapFS{"art/units/aoi.png": {Data: pngBytes(t, 200, 100, 255)}}, "art/units/aoi.png", true, true, 200, "正方形ではありません（art/units/aoi.png: 200×100）"},
		{"TC-DES-NG-08-24 背景は正方形でなくても知らせない", fstest.MapFS{"art/title.png": {Data: pngBytes(t, 800, 600, 255)}}, "art/title.png", false, true, 800, ""},
		{"TC-DES-NG-08-25 透過の PNG は透明度を保って読む", fstest.MapFS{"art/units/aoi.png": {Data: pngBytes(t, 4, 4, 0)}}, "art/units/aoi.png", true, true, 4, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			a := NewArtStore(tt.media, &stderr, nil)
			// 2 回引いても誤りは 1 回だけ。
			img := a.Image(tt.path, tt.square)
			_ = a.Image(tt.path, tt.square)
			if (img != nil) != tt.wantOK {
				t.Fatalf("Image が使える = %v、%v のはず", img != nil, tt.wantOK)
			}
			if img != nil {
				if img.Bounds().Dx() != tt.wantW {
					t.Errorf("幅 %d、%d のはず", img.Bounds().Dx(), tt.wantW)
				}
				if !a.Has(tt.path, tt.square) {
					t.Error("Has が偽")
				}
			}
			if tt.wantStder == "" {
				if stderr.Len() != 0 {
					t.Errorf("stderr に出さないはずが %q", stderr.String())
				}
			} else if n := strings.Count(stderr.String(), tt.wantStder); n != 1 {
				t.Errorf("stderr に %q が %d 回（1 回のはず）: %q", tt.wantStder, n, stderr.String())
			}
		})
	}
	t.Run("TC-DES-NG-08-25 透過の画素の透明度が保たれる", func(t *testing.T) {
		a := NewArtStore(fstest.MapFS{"p.png": {Data: pngBytes(t, 4, 4, 0)}}, &bytes.Buffer{}, nil)
		_, _, _, al := a.Image("p.png", true).At(0, 0).RGBA()
		if al != 0 {
			t.Errorf("透明の画素の alpha = %d", al)
		}
	})
	t.Run("TC-DES-NG-08-26 media が nil・nil の store・不正な道筋は何も返さない", func(t *testing.T) {
		var nilStore *ArtStore
		for _, a := range []*ArtStore{NewArtStore(nil, &bytes.Buffer{}, nil), nilStore} {
			if a.Has("art/title.png", false) || a.Ebiten("art/title.png", false) != nil {
				t.Error("なしのはずが画像がある")
			}
			if _, ok := a.UnitArtPath(ArtKey{Name: "アオイ", Player: true}); ok {
				t.Error("UnitArtPath が ok")
			}
		}
		a := NewArtStore(fstest.MapFS{"../x.png": {Data: pngBytes(t, 2, 2, 255)}}, &bytes.Buffer{}, nil)
		if a.Has("../x.png", false) {
			t.Error("media の外を指す道筋を読んだ")
		}
	})
	t.Run("TC-DES-NG-08-27 描画用の画像は元の大きさで、同じ画像を返す", func(t *testing.T) {
		a := NewArtStore(fstest.MapFS{"p.png": {Data: good}}, &bytes.Buffer{}, nil)
		eb := a.Ebiten("p.png", true)
		if eb == nil || eb.Bounds().Dx() != 128 {
			t.Fatalf("Ebiten = %v", eb)
		}
		if a.Ebiten("p.png", true) != eb {
			t.Error("キャッシュされていない")
		}
	})
}

// 名前の規則による解決（DES-NG-08）。
func TestArtResolveNG02(t *testing.T) {
	cat := alchemyCatalog(t)
	media := fstest.MapFS{
		"art/units/aoi.png":             {Data: pngBytes(t, 8, 8, 255)},
		"art/units/enemy_bandit.png":    {Data: pngBytes(t, 8, 8, 255)},
		"art/portraits/aoi.png":         {Data: pngBytes(t, 8, 8, 255)},
		"art/portraits/dain.png":        {Data: pngBytes(t, 8, 8, 255)},
		"art/units/enemy_swordsman.png": {Data: []byte("broken")},
		"art/story/ch01_before.png":     {Data: pngBytes(t, 8, 8, 255)},
		"art/title.png":                 {Data: pngBytes(t, 8, 8, 255)},
	}
	a := NewArtStore(media, &bytes.Buffer{}, cat)
	chars := map[string]string{}
	for _, c := range cat.Characters() {
		chars[c.Name] = c.ID
	}
	classes := map[string]string{}
	for _, c := range cat.Classes() {
		classes[c.Name] = c.ID
	}
	t.Run("TC-DES-NG-08-30 名前の表はキャラの名前から ID、クラスの表示名から ID を引く", func(t *testing.T) {
		if chars["アオイ"] != "aoi" || chars["ダイン"] != "dain" || classes["野盗"] != "bandit" {
			t.Fatalf("キャラ %v クラス %v", chars, classes)
		}
		idx := newArtIndex(cat)
		if idx.charByName["アオイ"] != "aoi" || idx.classByName["野盗"] != "bandit" {
			t.Errorf("表 %+v", idx)
		}
	})
	t.Run("TC-DES-NG-08-41 Characters・Classes は ID の昇順で全件を返す", func(t *testing.T) {
		var ids []string
		for _, c := range cat.Characters() {
			ids = append(ids, c.ID)
		}
		if got := strings.Join(ids, ","); got != "aoi,dain,haru,nemu,tori" {
			t.Errorf("キャラ %s", got)
		}
		ids = nil
		for _, c := range cat.Classes() {
			ids = append(ids, c.ID)
		}
		if !sort.StringsAreSorted(ids) || len(ids) < 8 {
			t.Errorf("クラス %v", ids)
		}
	})
	tests := []struct {
		name string
		key  ArtKey
		want string
	}{
		{"TC-DES-NG-08-31 自軍はキャラの ID の絵", ArtKey{Name: "アオイ", ClassName: "錬金術師", Player: true}, "art/units/aoi.png"},
		{"TC-DES-NG-08-32 敵はクラスの ID の絵（enemy_ を付ける）", ArtKey{Name: "野盗", ClassName: "野盗", Player: false}, "art/units/enemy_bandit.png"},
		{"TC-DES-NG-08-33 絵のファイルがなければなし（今のタイル）", ArtKey{Name: "ダイン", ClassName: "重装兵", Player: true}, ""},
		{"TC-DES-NG-08-34 壊れた画像はなし", ArtKey{Name: "剣士", ClassName: "剣士", Player: false}, ""},
		{"TC-DES-NG-08-35 名前が目録にないユニット・手がかりが空はなし", ArtKey{Name: "だれか", ClassName: "不明", Player: true}, ""},
		{"TC-DES-NG-08-36 自軍の絵は、敵と同じクラスでもクラスの絵を使わない", ArtKey{Name: "だれか", ClassName: "野盗", Player: true}, ""},
		{"TC-DES-NG-08-37 敵の絵はキャラの名前では引かない", ArtKey{Name: "アオイ", ClassName: "不明", Player: false}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := a.UnitArtPath(tt.key)
			if got != tt.want || ok != (tt.want != "") {
				t.Errorf("UnitArtPath(%+v) = %q, %v、%q のはず", tt.key, got, ok, tt.want)
			}
		})
	}
	t.Run("TC-DES-NG-08-38 情報欄は自軍なら肖像、なければユニットの絵、敵はユニットの絵", func(t *testing.T) {
		want := map[string]string{
			"アオイ": "art/portraits/aoi.png", // 肖像あり
			"ダイン": "art/portraits/dain.png",
		}
		for name, w := range want {
			if got, ok := a.InfoArtPath(ArtKey{Name: name, Player: true}); !ok || got != w {
				t.Errorf("%s: %q, %v", name, got, ok)
			}
		}
		if got, ok := a.InfoArtPath(ArtKey{Name: "トーリ", Player: true}); ok {
			t.Errorf("トーリは絵なし: %q", got)
		}
		if got, ok := a.InfoArtPath(ArtKey{ClassName: "野盗"}); !ok || got != "art/units/enemy_bandit.png" {
			t.Errorf("敵: %q, %v", got, ok)
		}
	})
	t.Run("TC-DES-NG-08-39 artKeyOf はユニットの名前・クラス・陣営を写す", func(t *testing.T) {
		got := artKeyOf(tactics.Unit{Name: "アオイ", ClassName: "錬金術師", Side: tactics.Player})
		if got != (ArtKey{Name: "アオイ", ClassName: "錬金術師", Player: true}) {
			t.Errorf("%+v", got)
		}
		if artKeyOf(tactics.Unit{Name: "野盗", Side: tactics.Enemy}).Player {
			t.Error("敵が Player")
		}
	})
	t.Run("TC-DES-NG-08-40 物語の背景の道筋は ID から決まり、画像がなければなし", func(t *testing.T) {
		p, ok := storyBackdropPath("ch01_before")
		if p != "art/story/ch01_before.png" || !ok {
			t.Errorf("%q %v", p, ok)
		}
		if _, ok := storyBackdropPath(""); ok {
			t.Error("空の ID が ok")
		}
		if _, ok := storyBackdropPath("../x"); ok {
			t.Error("道筋を含む ID が ok")
		}
		if _, ok := a.BackdropPath(storyBackdropPath("ch01_before")); !ok {
			t.Error("ある背景が使えない")
		}
		if _, ok := a.BackdropPath(storyBackdropPath("ch02_before")); ok {
			t.Error("ない背景が使える")
		}
		if _, ok := a.BackdropPath(artTitlePath, true); !ok {
			t.Error("タイトルの背景が使えない")
		}
	})
}

// 物語の段落から話者の名前を取り出す規則（DES-NG-08）。
func TestSpeakerNameNG02(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"TC-DES-NG-08-50 「名前「…」」の形", "アオイ「この水路を直さないと」", "アオイ", true},
		{"TC-DES-NG-08-51 台詞のあとに地の文が続いても先頭の話者", "アオイ「前に立つのは苦手です」荷の中には包みが並んでいる。", "アオイ", true},
		{"TC-DES-NG-08-52 台詞が 2 人分なら最初の話者", "ダイン「それは、薬ですか」　アオイ「薬、のつもりです」", "ダイン", true},
		{"TC-DES-NG-08-53 名前の前後の空白（半角・全角）を除く", " 　ダイン　 「はい」", "ダイン", true},
		{"TC-DES-NG-08-54 名前の後ろの括弧書きを除く（全角・半角）", "アオイ（小声）「はい」", "アオイ", true},
		{"TC-DES-NG-08-55 半角の括弧書きも除く", "アオイ(小声)「はい」", "アオイ", true},
		{"TC-DES-NG-08-56 地の文のあとの台詞（句読点が先にある）は話者なし", "春の終わり、役人が訪ねてきた。役人「これを動かしてほしい」", "", false},
		{"TC-DES-NG-08-57 「がない段落は話者なし", "アオイは、薬を煎じた。", "", false},
		{"TC-DES-NG-08-58 閉じの」がなければ話者なし", "アオイ「はい", "", false},
		{"TC-DES-NG-08-59 名前が空（「から始まる）は話者なし", "「はい」と言った", "", false},
		{"TC-DES-NG-08-60 名前が長すぎる先頭（地の文）は話者なし", "とても長い地の文の書き出しが続いている「はい」", "", false},
		{"TC-DES-NG-08-61 括弧書きだけの名前は話者なし", "（小声）「はい」", "", false},
		{"TC-DES-NG-08-62 空の段落は話者なし", "", "", false},
		{"TC-DES-NG-08-63 改行を含む名前は話者なし", "アオイ\nダイン「はい」", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := speakerName(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("speakerName(%q) = %q, %v、%q, %v のはず", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
	t.Run("TC-DES-NG-08-64 名前が一致するキャラがいない・肖像がなければ肖像なし", func(t *testing.T) {
		cat := alchemyCatalog(t)
		a := NewArtStore(fstest.MapFS{"art/portraits/aoi.png": {Data: pngBytes(t, 8, 8, 255)}}, &bytes.Buffer{}, cat)
		for para, want := range map[string]string{
			"アオイ「はい」": "art/portraits/aoi.png",
			"役人「はい」":  "", // キャラにいない名前
			"ダイン「はい」": "", // 肖像のファイルがない
			"地の文だけ。":  "",
		} {
			got, ok := a.StoryPortraitPath(para)
			if got != want || ok != (want != "") {
				t.Errorf("%q: %q, %v", para, got, ok)
			}
		}
	})
}

// 描く位置と大きさを決める純粋な関数（DES-NG-08）。
func TestArtLayoutNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-70 fitRect は縦横比を保って箱に収め、中央に置く", func(t *testing.T) {
		box := image.Rect(10, 20, 10+100, 20+100)
		tests := []struct {
			name string
			w, h int
			want image.Rectangle
		}{
			{"正方形は箱いっぱい", 128, 128, box},
			{"小さい正方形も拡大して箱いっぱい", 16, 16, box},
			{"横長 2:1 は幅いっぱいで上下に余白", 200, 100, image.Rect(10, 45, 110, 95)},
			{"縦長 1:2 は高さいっぱいで左右に余白", 100, 200, image.Rect(35, 20, 85, 120)},
			{"大きさ 0 は空", 0, 10, image.Rectangle{}},
		}
		for _, tt := range tests {
			if got := fitRect(box, tt.w, tt.h); got != tt.want {
				t.Errorf("%s: fitRect = %v、%v のはず", tt.name, got, tt.want)
			}
		}
		if got := fitRect(image.Rectangle{}, 10, 10); !got.Empty() {
			t.Errorf("空の箱: %v", got)
		}
	})
	t.Run("TC-DES-NG-08-71 unitArtRect は 1 マスに収め、128 px の絵を CellSize に縮める", func(t *testing.T) {
		got := unitArtRect(64, 96, CellSize, 128, 128)
		if want := image.Rect(64, 96, 64+CellSize, 96+CellSize); got != want {
			t.Errorf("%v、%v のはず", got, want)
		}
		got = unitArtRect(0, 0, CellSize, 64, 32)
		if want := image.Rect(0, 8, CellSize, 24); got != want {
			t.Errorf("横長: %v、%v のはず", got, want)
		}
	})
	t.Run("TC-DES-NG-08-72 coverRect は画面を隙間なく覆い、はみ出しは中央で切る", func(t *testing.T) {
		screen := image.Rect(0, 0, 800, 600)
		tests := []struct {
			name string
			w, h int
			want image.Rectangle
		}{
			{"同じ比率はそのまま", 400, 300, screen},
			{"横長の絵は高さに合わせ、左右がはみ出す", 1600, 600, image.Rect(-400, 0, 1200, 600)},
			{"縦長の絵は幅に合わせ、上下がはみ出す", 800, 1200, image.Rect(0, -300, 800, 900)},
		}
		for _, tt := range tests {
			got := coverRect(screen, tt.w, tt.h)
			if got != tt.want {
				t.Errorf("%s: %v、%v のはず", tt.name, got, tt.want)
			}
			if got.Min.X > screen.Min.X || got.Min.Y > screen.Min.Y || got.Max.X < screen.Max.X || got.Max.Y < screen.Max.Y {
				t.Errorf("%s: 画面を覆っていない %v", tt.name, got)
			}
		}
		if !coverRect(screen, 0, 0).Empty() {
			t.Error("大きさ 0 は空")
		}
	})
	t.Run("TC-DES-NG-08-73 storyArtLayout は肖像なしなら今の配置、ありなら肖像の右に文を寄せ画面に収める", func(t *testing.T) {
		box, x, w := storyArtLayout(false)
		if !box.Empty() || x != 48 || w != ScreenWidth-96 {
			t.Errorf("肖像なし: %v %d %d", box, x, w)
		}
		box, x, w = storyArtLayout(true)
		if box.Dx() != box.Dy() || box.Min.X != 48 || x < box.Max.X || x+w != ScreenWidth-48 || w <= 0 {
			t.Errorf("肖像あり: %v %d %d", box, x, w)
		}
		if box.Max.Y > ScreenHeight || box.Min.Y < 0 {
			t.Errorf("肖像が画面の外: %v", box)
		}
	})
}

// 場面の表示（DES-NG-08）。画像がなければ今のまま、あれば背景と肖像が出る。base では何も変わらない。
func TestSceneArtViewNG02(t *testing.T) {
	cat := alchemyCatalog(t)
	media := fstest.MapFS{
		"art/portraits/aoi.png":     {Data: pngBytes(t, 8, 8, 255)},
		"art/story/ch01_before.png": {Data: pngBytes(t, 8, 8, 255)},
		"art/title.png":             {Data: pngBytes(t, 8, 8, 255)},
	}
	newScenes := func(media fstest.MapFS) *SceneController {
		cfg := Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }, Stderr: &bytes.Buffer{}}
		if media != nil {
			cfg.Media = media
		}
		return NewSceneController(cfg)
	}
	toStory := func(t *testing.T, s *SceneController, paragraph int) {
		t.Helper()
		s.Choose(0)
		s.Choose(0)
		if s.Scene() != Story {
			t.Fatalf("場面 %v", s.Scene())
		}
		for i := 0; i < paragraph; i++ {
			s.Advance()
		}
	}
	// アオイの台詞の段落（story.toml の ch01_before の 9 番目）と、ダインの台詞（肖像なし）、地の文。
	const aoiPara, dainPara, narrationPara = 8, 9, 0
	t.Run("TC-DES-NG-08-80 話者の肖像があれば出し、文を右へ寄せる", func(t *testing.T) {
		s := newScenes(media)
		toStory(t, s, aoiPara)
		v := s.View()
		if v.Portrait == nil || v.Portrait.Path != "art/portraits/aoi.png" {
			t.Fatalf("Portrait = %+v", v.Portrait)
		}
		box, x, _ := storyArtLayout(true)
		if v.Portrait.Rect != box {
			t.Errorf("箱 %v、%v のはず", v.Portrait.Rect, box)
		}
		body := 0
		for _, tx := range v.Texts {
			// 本文は y = 160 から 20 px ごと。案内（y = 488 以降）と見出し（y = 32）は除く。
			if tx.Y >= 160 && tx.Y < 400 {
				body++
				if tx.X != x {
					t.Errorf("本文の左端 %d、%d のはず: %q", tx.X, x, tx.Text)
				}
				if textWidth(tx.Text) > ScreenWidth-48-x {
					t.Errorf("本文が右の余白に食い込む: %q", tx.Text)
				}
			}
		}
		if body == 0 {
			t.Error("本文がない")
		}
		if v.Backdrop != "art/story/ch01_before.png" {
			t.Errorf("背景 %q", v.Backdrop)
		}
	})
	t.Run("TC-DES-NG-08-81 肖像のない話者・地の文では肖像を出さず、文の配置は今のまま", func(t *testing.T) {
		for _, para := range []int{dainPara, narrationPara} {
			s := newScenes(media)
			toStory(t, s, para)
			v := s.View()
			if v.Portrait != nil {
				t.Errorf("段落 %d: 肖像が出た %+v", para, v.Portrait)
			}
			base := newScenes(nil)
			toStory(t, base, para)
			if !sameTexts(v.Texts, base.View().Texts) {
				t.Errorf("段落 %d: 画像がないときと文の配置が違う", para)
			}
		}
	})
	t.Run("TC-DES-NG-08-82 背景のない章では背景なし。タイトルは title.png があれば背景", func(t *testing.T) {
		s := newScenes(media)
		if got := s.View().Backdrop; got != "art/title.png" {
			t.Errorf("タイトルの背景 %q", got)
		}
		s2 := newScenes(fstest.MapFS{})
		if got := s2.View().Backdrop; got != "" {
			t.Errorf("画像がなければ空のはず: %q", got)
		}
		toStory(t, s2, 0)
		if v := s2.View(); v.Backdrop != "" || v.Portrait != nil {
			t.Errorf("画像なしの物語: %+v %+v", v.Backdrop, v.Portrait)
		}
	})
	t.Run("TC-DES-NG-08-83 Media が nil（既存の作品）では、全場面の View に画像の指定がなく、文と部品が画像なしと同じ", func(t *testing.T) {
		withNil := newScenes(nil)
		withEmpty := newScenes(fstest.MapFS{})
		if withNil.art == nil || withNil.art.fsys != nil {
			t.Fatal("Media nil の store")
		}
		for _, s := range []*SceneController{withNil, withEmpty} {
			v := s.View()
			if v.Backdrop != "" || v.Portrait != nil {
				t.Fatalf("タイトル: %+v", v)
			}
			toStory(t, s, aoiPara)
			v = s.View()
			if v.Backdrop != "" || v.Portrait != nil {
				t.Fatalf("物語: %+v", v)
			}
			if len(v.Texts) == 0 || v.Texts[len(v.Texts)-1].Style != StyleLabel {
				t.Errorf("案内の文がない")
			}
		}
		if !sameTexts(withNil.View().Texts, withEmpty.View().Texts) {
			t.Error("Media が nil と空とで文が違う")
		}
	})
	t.Run("TC-DES-NG-08-84 情報欄の絵は ArtKey を持ち、画像があるときだけ描き替えられる", func(t *testing.T) {
		u := tactics.Unit{Name: "アオイ", ClassName: "錬金術師", Side: tactics.Player, Lord: true, Alive: true, HP: 10, Stats: tactics.Stats{HP: 10}}
		var v InfoView
		v.addStats(u, 0, 0)
		found := false
		for _, ic := range v.Icons {
			if ic.Size == PortraitSize {
				found = true
				if ic.Art != (ArtKey{Name: "アオイ", ClassName: "錬金術師", Player: true}) {
					t.Errorf("Art = %+v", ic.Art)
				}
				if ic.Tile != unitTile(u) {
					t.Error("タイルが今のまま残っていない")
				}
			}
		}
		if !found {
			t.Fatal("ユニットの絵のアイコンがない")
		}
	})
}

func sameTexts(a, b []SceneText) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

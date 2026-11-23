package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
)

// fill は w×h の画像を c で塗る。
func fill(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// rect は img の r の範囲を c で塗る。
func rect(img *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

var (
	green = color.NRGBA{0, 255, 0, 255}
	red   = color.NRGBA{200, 40, 30, 255}
)

func TestCodexCall(t *testing.T) {
	t.Run("TC-FR-SYS-08-01 Codex CLI への依頼文と引数", func(t *testing.T) {
		p := "line one\nline two | x"
		w := wrapPrompt("  " + p + "\n\n")
		if !strings.Contains(w, "<<<PROMPT\n"+p+"\nPROMPT>>>") {
			t.Fatalf("指示文がそのまま印の間に入っていない: %q", w)
		}
		if !strings.Contains(w, "image_gen tool exactly once") || !strings.Contains(w, "do not create or edit files") {
			t.Fatalf("1 回だけ・ファイルを作らない、の依頼がない: %q", w)
		}
		got := strings.Join(execArgs("/w", ""), " ")
		want := "exec --skip-git-repo-check --ephemeral -c approval_policy=\"never\" -s read-only -C /w --json -"
		if got != want {
			t.Fatalf("引数 = %q、want %q", got, want)
		}
		got = strings.Join(execArgs("/w", "m1"), " ")
		if got != "exec --skip-git-repo-check --ephemeral -c approval_policy=\"never\" -s read-only -C /w --json -m m1 -" {
			t.Fatalf("モデル指定の引数 = %q", got)
		}
	})

	t.Run("TC-FR-SYS-08-02 codex exec --json の出来事を読む", func(t *testing.T) {
		ok := `{"type":"thread.started","thread_id":"abc-1"}
{"type":"turn.started"}
not json
{"type":"item.completed","item":{"id":"i0","type":"reasoning","text":"thinking"}}
{"type":"item.completed","item":{"id":"i1","type":"agent_message","text":" /h/.codex/generated_images/abc-1/x.png \n"}}
{"type":"turn.completed","usage":{}}
`
		res, err := parseExecEvents(strings.NewReader(ok))
		if err != nil || res.ThreadID != "abc-1" || res.Message != "/h/.codex/generated_images/abc-1/x.png" {
			t.Fatalf("got %+v, %v", res, err)
		}
		if _, err := parseExecEvents(strings.NewReader(`{"type":"thread.started","thread_id":"a"}` + "\n" + `{"type":"error","message":"usage limit"}`)); err == nil || !strings.Contains(err.Error(), "usage limit") {
			t.Fatalf("error の出来事が誤りにならない: %v", err)
		}
		if _, err := parseExecEvents(strings.NewReader(`{"type":"thread.started","thread_id":"a"}` + "\n" + `{"type":"turn.failed","error":{"message":"boom"}}`)); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("turn.failed が誤りにならない: %v", err)
		}
		if _, err := parseExecEvents(strings.NewReader(`{"type":"turn.started"}`)); err == nil {
			t.Fatal("thread.started がないのに誤りにならない")
		}
	})

	t.Run("TC-FR-SYS-08-03 生成した画像を探す", func(t *testing.T) {
		home := t.TempDir()
		dir := filepath.Join(home, "generated_images", "t1")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := findGenerated(home, "t1"); err == nil {
			t.Fatal("PNG が 0 個なのに誤りにならない")
		}
		os.WriteFile(filepath.Join(dir, "a.png"), []byte("x"), 0o644)
		os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644)
		if p, err := findGenerated(home, "t1"); err != nil || p != filepath.Join(dir, "a.png") {
			t.Fatalf("got %q, %v", p, err)
		}
		os.WriteFile(filepath.Join(dir, "b.PNG"), []byte("x"), 0o644)
		if _, err := findGenerated(home, "t1"); err == nil {
			t.Fatal("PNG が 2 個なのに誤りにならない")
		}
		for _, bad := range []string{"", "..", "a/b", `a\b`} {
			if _, err := findGenerated(home, bad); err == nil {
				t.Fatalf("スレッドの ID %q が誤りにならない", bad)
			}
		}
		if _, err := findGenerated(home, "missing"); err == nil {
			t.Fatal("置き場がないのに誤りにならない")
		}
	})

	t.Run("TC-FR-SYS-08-04 設定ファイルのモデル", func(t *testing.T) {
		cases := []struct{ cfg, want string }{
			{"model = \"gpt-x\"\nmodel_reasoning_effort = \"low\"\n", "gpt-x"},
			{"# c\n  model=\"m2\"\n", "m2"},
			{"[profiles.a]\nmodel = \"inner\"\n", ""},
			{"approval = 1\n", ""},
		}
		for _, c := range cases {
			if got := configuredModel(c.cfg); got != c.want {
				t.Errorf("configuredModel(%q) = %q、want %q", c.cfg, got, c.want)
			}
		}
	})
}

func TestProcess(t *testing.T) {
	t.Run("TC-FR-SYS-08-05 クロマキーの α と緑かぶりの除去", func(t *testing.T) {
		cases := []struct {
			k    int
			want uint8
		}{{-50, 255}, {70, 255}, {71, 252}, {120, 127}, {169, 2}, {170, 0}, {255, 0}}
		for _, c := range cases {
			if got := keyAlpha(c.k, 70, 170); got != c.want {
				t.Errorf("keyAlpha(%d) = %d、want %d", c.k, got, c.want)
			}
		}
		// 20×20 の緑の背景に、赤の 4×4（x 2〜5）と、緑がかった灰色の 2×2（x 12〜13）。
		img := fill(40, 40, green)
		rect(img, image.Rect(7, 7, 11, 11), red)
		rect(img, image.Rect(30, 30, 32, 32), color.NRGBA{100, 150, 100, 255}) // k = 50 → 不透明、縁の帯の中
		out, err := chromaKey(img)
		if err != nil {
			t.Fatal(err)
		}
		if a := out.NRGBAAt(0, 0).A; a != 0 {
			t.Errorf("緑の背景の α = %d、want 0", a)
		}
		if c := out.NRGBAAt(8, 8); c != red {
			t.Errorf("赤の中身 = %v、want 変わらない %v", c, red)
		}
		if c := out.NRGBAAt(30, 30); c.A != 255 || c.G != 100 {
			t.Errorf("縁の帯の緑がかった画素 = %v、want α 255・G 100（max(R,B)）", c)
		}
	})

	t.Run("TC-FR-SYS-08-06 切り出しと足元をそろえた正方形", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
		rect(img, image.Rect(30, 10, 40, 60), red)              // 中身は 10×50
		img.SetNRGBA(80, 80, color.NRGBA{255, 0, 0, trimAlpha}) // α が下限ちょうどの画素は中身に数えない
		sq, err := trimSquare(img)
		if err != nil {
			t.Fatal(err)
		}
		// 長い辺 50、余白 round(50×0.06)=3、一辺 56。中身は x (56−10)/2=23、y 56−3−50=3。
		if sq.Rect.Dx() != 56 || sq.Rect.Dy() != 56 {
			t.Fatalf("一辺 = %v、want 56", sq.Rect)
		}
		if got := contentBounds(sq); got != image.Rect(23, 3, 33, 53) {
			t.Fatalf("中身の位置 = %v、want (23,3)-(33,53)", got)
		}
		if _, err := trimSquare(image.NewNRGBA(image.Rect(0, 0, 8, 8))); err == nil {
			t.Fatal("中身がないのに誤りにならない")
		}
	})

	t.Run("TC-FR-SYS-08-07 中央を残して比に切る", func(t *testing.T) {
		cases := []struct {
			b    image.Rectangle
			w, h int
			want image.Rectangle
		}{
			// 1536×1024 → 832:576。幅 round(1024×832/576)=1479、左右に (1536−1479)/2=28。
			{image.Rect(0, 0, 1536, 1024), 832, 576, image.Rect(28, 0, 1507, 1024)},
			// 縦長 1024×1536 → 832:576。高さ round(1024×576/832)=709、上下に (1536−709)/2=413。
			{image.Rect(0, 0, 1024, 1536), 832, 576, image.Rect(0, 413, 1024, 1122)},
			// 1254×1254 → 正方形はそのまま。
			{image.Rect(0, 0, 1254, 1254), 512, 512, image.Rect(0, 0, 1254, 1254)},
			// 832×576 ちょうど → そのまま。
			{image.Rect(0, 0, 832, 576), 832, 576, image.Rect(0, 0, 832, 576)},
		}
		for _, c := range cases {
			if got := coverRect(c.b, c.w, c.h); got != c.want {
				t.Errorf("coverRect(%v, %d, %d) = %v、want %v", c.b, c.w, c.h, got, c.want)
			}
		}
	})

	t.Run("TC-FR-SYS-08-08 種類ごとの加工の大きさと透過", func(t *testing.T) {
		bg := fill(1536, 1024, color.NRGBA{180, 150, 100, 255})
		out, err := processImage(bg, kinds["story"], false)
		if err != nil || out.Rect.Dx() != 832 || out.Rect.Dy() != 576 {
			t.Fatalf("story = %v, %v、want 832×576", out.Rect, err)
		}
		if c := out.NRGBAAt(0, 0); c.A != 255 {
			t.Errorf("背景の画像が透過になった: %v", c)
		}
		dark := fill(100, 100, color.NRGBA{0, 177, 64, 255})
		rect(dark, image.Rect(30, 20, 70, 80), red)
		d, err := processImage(dark, kinds["units"], true)
		if err != nil || d.NRGBAAt(0, 0).A != 0 || d.NRGBAAt(64, 64).A != 255 {
			t.Fatalf("暗い緑の背景: %v", err)
		}
		rect(dark, image.Rect(0, 0, 20, 20), red)
		if _, err := processImage(dark, kinds["units"], true); err == nil {
			t.Fatal("縁が抜けない画像を受け入れた")
		}
		unit := fill(400, 400, green)
		rect(unit, image.Rect(150, 100, 250, 300), red)
		u, err := processImage(unit, kinds["units"], true)
		if err != nil || u.Rect.Dx() != 128 || u.Rect.Dy() != 128 {
			t.Fatalf("units = %v, %v、want 128×128", u.Rect, err)
		}
		if a := u.NRGBAAt(0, 0).A; a != 0 {
			t.Errorf("units の隅の α = %d、want 0", a)
		}
		if c := u.NRGBAAt(64, 64); c.A != 255 || c.R < 190 || c.G > 50 {
			t.Errorf("units の中央 = %v、want 赤の不透明", c)
		}
		if n := residualGreen(u, unit); n != 0 {
			t.Errorf("緑の残る半透明の画素 = %d、want 0", n)
		}
		p, err := processImage(fill(1254, 1254, red), kinds["portraits"], false)
		if err != nil || p.Rect.Dx() != 512 || p.Rect.Dy() != 512 {
			t.Fatalf("portraits = %v, %v、want 512×512", p.Rect, err)
		}
		if _, err := processImage(unit, kinds["story"], true); err == nil {
			t.Fatal("正方形でない種類の透過が誤りにならない")
		}
		if _, err := processImage(fill(16, 16, green), kinds["units"], true); err == nil {
			t.Fatal("全体が緑なのに誤りにならない")
		}
	})
}

func sampleMeta() genMeta {
	return genMeta{Tool: "codex-cli 0.169.0", Model: "gpt-x", ThreadID: "t1", Date: "2026-11-18", Prompt: "Line one.\n  Line two.", Source: "/x.png"}
}

func TestLedger(t *testing.T) {
	t.Run("TC-FR-SYS-08-09 台帳の行を組み立てる", func(t *testing.T) {
		in := rowInput{File: "art/story/ch01_before.png", Kind: "story", SrcW: 1536, SrcH: 1024, Meta: sampleMeta(),
			Terms: "規約 A 第 3 条", CheckedOn: "2026-11-18", CheckedBy: "Claude（エージェント）"}
		row, err := makeRow(in)
		if err != nil {
			t.Fatal(err)
		}
		cells := strings.Split(strings.Trim(row, "| "), " | ")
		if len(cells) != 7 {
			t.Fatalf("列の数 = %d、want 7: %q", len(cells), row)
		}
		want := []string{
			"`art/story/ch01_before.png`",
			"AI 生成（Codex CLI の画像生成）。元の 1536×1024 の画像を、tools/imagegen で、中央を残して 832:576 の比に切り（はみ出しを左右か上下で等しく切る）、832×576 の PNG に縮小した（Catmull-Rom）。ほかの加工はしていない",
			"codex-cli 0.169.0（codex exec の組み込みの image_gen ツール。Codex の会話のモデルは gpt-x。画像のモデル名は表示されない）、2026-11-18。加工は tools/imagegen",
			"「Line one. Line two.」（全文。改行は空白に置き換えた）。Codex の会話のモデルに、wrapPrompt の依頼文で指示を変えずに image_gen へ渡すよう頼んだ。image_gen が実際に受け取った文は確認していない",
			"規約 A 第 3 条", "2026-11-18", "Claude（エージェント）",
		}
		for i := range want {
			if cells[i] != want[i] {
				t.Errorf("欄 %d = %q、want %q", i, cells[i], want[i])
			}
		}
		u := in
		u.File, u.Kind, u.Chroma = "art/units/aoi.png", "units", true
		row, err = makeRow(u)
		if err != nil || !strings.HasPrefix(strings.Split(row, " | ")[1], "AI 生成（Codex CLI の画像生成）。元の 1536×1024 の画像（背景は緑）を") || !strings.Contains(row, "128×128 の PNG") {
			t.Fatalf("units の作り方 = %q, %v", row, err)
		}
		bad := []func(*rowInput){
			func(r *rowInput) { r.Meta.Prompt = "a | b" },
			func(r *rowInput) { r.Terms = "" },
			func(r *rowInput) { r.CheckedBy = "  " },
			func(r *rowInput) { r.CheckedOn = "2026/11/18" },
			func(r *rowInput) { r.Meta.Date = "" },
			func(r *rowInput) { r.Kind = "tiles" },
		}
		for i, f := range bad {
			r := in
			f(&r)
			if _, err := makeRow(r); err == nil {
				t.Errorf("誤りの入力 %d が誤りにならない", i)
			}
		}
	})

	t.Run("TC-FR-SYS-08-10 台帳に行を足す・置き換える", func(t *testing.T) {
		ledger := "# 台帳\n\n```\n| ファイル | x |\n| `art/a.png` | sample |\n```\n\n| ファイル | 作り方 |\n|---|---|\n| `art/units/aoi.png` | A |\n| `audio/se/x.wav` | B |\n\n## 後ろ\n"
		got, err := insertRow(ledger, "art/story/ch01_before.png", "| `art/story/ch01_before.png` | C |", false)
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(ledger, "| `audio/se/x.wav` | B |\n", "| `audio/se/x.wav` | B |\n| `art/story/ch01_before.png` | C |\n", 1)
		if got != want {
			t.Fatalf("got\n%s\nwant\n%s", got, want)
		}
		if _, err := insertRow(ledger, "art/units/aoi.png", "| `art/units/aoi.png` | D |", false); err == nil {
			t.Fatal("同じファイルの行があるのに誤りにならない")
		}
		got, err = insertRow(ledger, "art/units/aoi.png", "| `art/units/aoi.png` | D |", true)
		if err != nil || !strings.Contains(got, "| `art/units/aoi.png` | D |\n| `audio/se/x.wav` | B |") || strings.Contains(got, "| A |") {
			t.Fatalf("置き換え = %q, %v", got, err)
		}
		// 囲みの中の見本の行（art/a.png）は、表の行とみなさない。
		if _, err := insertRow(ledger, "art/a.png", "| `art/a.png` | E |", false); err != nil {
			t.Fatalf("囲みの中の見本の行が重複とみなされた: %v", err)
		}
		if _, err := insertRow("# 見出しだけ\n", "art/x.png", "| x |", false); err == nil {
			t.Fatal("表がないのに誤りにならない")
		}
	})

	t.Run("TC-FR-SYS-08-11 置き場の名前と保存先の制限", func(t *testing.T) {
		cases := []struct{ kind, id, want string }{
			{"units", "aoi", "art/units/aoi.png"},
			{"units", "enemy_bandit", "art/units/enemy_bandit.png"},
			{"portraits", "nemu", "art/portraits/nemu.png"},
			{"story", "ch01_before", "art/story/ch01_before.png"},
			{"title", "", "art/title.png"},
		}
		for _, c := range cases {
			if got, err := artPath(c.kind, c.id); err != nil || got != c.want {
				t.Errorf("artPath(%q, %q) = %q, %v、want %q", c.kind, c.id, got, err, c.want)
			}
		}
		for _, c := range [][2]string{{"units", "Aoi"}, {"story", "ch01-before"}, {"units", ""}, {"tiles", "x"}, {"story", "../x"}} {
			if _, err := artPath(c[0], c[1]); err == nil {
				t.Errorf("artPath(%q, %q) が誤りにならない", c[0], c[1])
			}
		}
		root, err := repositoryRoot()
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{filepath.Join(root, "other", "raw.png"), filepath.Join(root, "data", "alchemy", "media", "raw.png")} {
			var errb bytes.Buffer
			if code := run([]string{"generate", "-prompt", "unused", "-out", path}, &bytes.Buffer{}, &errb); code != 2 {
				t.Fatalf("リポジトリ内 %s: %d %s", path, code, errb.String())
			}
		}
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(root, link); err == nil {
			var errb bytes.Buffer
			if code := run([]string{"generate", "-prompt", "unused", "-out", filepath.Join(link, "new", "raw.png")}, &bytes.Buffer{}, &errb); code != 2 {
				t.Fatalf("リンク経由: %d %s", code, errb.String())
			}
		} else if runtime.GOOS != "windows" {
			t.Fatal(err)
		}
		media := t.TempDir()
		var stderr bytes.Buffer
		code := run([]string{"generate", "-prompt", "p.txt", "-out", filepath.Join(media, "art", "x.png"), "-media", media}, &bytes.Buffer{}, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "の外に保存") {
			t.Fatalf("media の中への保存 = %d %q、want 2", code, stderr.String())
		}
	})
}

// writePNG は img を p に PNG で書く。
func writePNG(t *testing.T, p string, img image.Image) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEndToEnd(t *testing.T) {
	t.Run("TC-FR-SYS-08-12 place で置いた画像と行が台帳の検査を通る", func(t *testing.T) {
		media := t.TempDir()
		ledger := "# 台帳\n\n| ファイル | 作り方 | 使ったツールと版・生成した日 | 指示（プロンプト）または出典の URL | 利用条件 | 確かめた日 | 確かめた人 |\n|---|---|---|---|---|---|---|\n"
		os.WriteFile(filepath.Join(media, "ASSETS.md"), []byte(ledger), 0o644)
		raw := filepath.Join(t.TempDir(), "raw.png")
		writePNG(t, raw, fill(1536, 1024, color.NRGBA{120, 110, 90, 255}))
		mj, _ := json.Marshal(sampleMeta())
		os.WriteFile(raw+".json", mj, 0o644)
		uraw := filepath.Join(t.TempDir(), "unit.png")
		u := fill(300, 300, green)
		rect(u, image.Rect(100, 50, 200, 250), red)
		writePNG(t, uraw, u)
		os.WriteFile(uraw+".json", mj, 0o644)
		args := func(in, kind, id string) []string {
			return []string{"place", "-in", in, "-kind", kind, "-id", id, "-media", media,
				"-terms", "規約 A 第 3 条", "-checked-on", "2026-11-18", "-checked-by", "Claude（エージェント）"}
		}
		var out, errb bytes.Buffer
		if code := run(args(raw, "story", "ch01_before"), &out, &errb); code != 0 {
			t.Fatalf("place story = %d: %s", code, errb.String())
		}
		if code := run(args(uraw, "units", "aoi"), &out, &errb); code != 0 {
			t.Fatalf("place units = %d: %s", code, errb.String())
		}
		if problems := data.CheckMediaLedger(os.DirFS(media)); len(problems) != 0 {
			t.Fatalf("台帳の検査の問題: %v", problems)
		}
		f, err := os.Open(filepath.Join(media, "art", "story", "ch01_before.png"))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil || cfg.Width != 832 || cfg.Height != 576 {
			t.Fatalf("置いた背景 = %+v, %v、want 832×576", cfg, err)
		}
		// もう一度置くと、-replace なしでは誤り。-replace なら行は 1 つのまま。
		errb.Reset()
		if code := run(args(raw, "story", "ch01_before"), &out, &errb); code != 1 {
			t.Fatalf("2 回目の place = %d、want 1", code)
		}
		if code := run(append(args(raw, "story", "ch01_before"), "-replace"), &out, &errb); code != 0 {
			t.Fatalf("-replace = %d: %s", code, errb.String())
		}
		b, _ := os.ReadFile(filepath.Join(media, "ASSETS.md"))
		if n := strings.Count(string(b), "`art/story/ch01_before.png`"); n != 1 {
			t.Fatalf("置き換えたあとの行の数 = %d、want 1", n)
		}
		if problems := data.CheckMediaLedger(os.DirFS(media)); len(problems) != 0 {
			t.Fatalf("置き換えたあとの台帳の検査の問題: %v", problems)
		}
	})

	t.Run("TC-FR-SYS-08-13 generate が Codex CLI の出力から生の画像と記録を保存する", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("偽の Codex CLI はシェルのスクリプトなので、Windows では動かさない")
		}
		dir := t.TempDir()
		home := filepath.Join(dir, "codexhome")
		os.MkdirAll(filepath.Join(home, "generated_images", "th-9"), 0o755)
		os.WriteFile(filepath.Join(home, "config.toml"), []byte("model = \"gpt-test\"\n"), 0o644)
		writePNG(t, filepath.Join(home, "generated_images", "th-9", "exec-1.png"), fill(4, 4, red))
		fake := filepath.Join(dir, "codex")
		script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codex-cli 9.9.9'; exit 0; fi\ncat > \"" + filepath.Join(dir, "stdin.txt") + "\"\n" +
			"echo '{\"type\":\"thread.started\",\"thread_id\":\"th-9\"}'\n" +
			"echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"done\"}}'\n"
		if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CODEX_HOME", home)
		prompt := filepath.Join(dir, "p.txt")
		os.WriteFile(prompt, []byte("A quiet stone city.\n"), 0o644)
		outPath := filepath.Join(dir, "raw", "ch01.png")
		var out, errb bytes.Buffer
		if code := run([]string{"generate", "-prompt", prompt, "-out", outPath, "-codex", fake, "-media", filepath.Join(dir, "media")}, &out, &errb); code != 0 {
			t.Fatalf("generate = %d: %s", code, errb.String())
		}
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("生の画像がない: %v", err)
		}
		var m genMeta
		b, _ := os.ReadFile(outPath + ".json")
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		if m.Tool != "codex-cli 9.9.9" || m.Model != "gpt-test" || m.ThreadID != "th-9" || m.Prompt != "A quiet stone city." || !datePattern.MatchString(m.Date) {
			t.Fatalf("生成の記録 = %+v", m)
		}
		for _, remove := range []string{"", ".json"} {
			if remove != "" {
				os.Remove(outPath)
				os.WriteFile(outPath+".json", []byte("元の記録"), 0644)
			}
			before, _ := os.ReadFile(outPath + remove)
			if code := run([]string{"generate", "-prompt", prompt, "-out", outPath, "-codex", fake}, &out, &errb); code != 1 {
				t.Fatalf("既存 %s: %d", remove, code)
			}
			after, _ := os.ReadFile(outPath + remove)
			if !bytes.Equal(before, after) {
				t.Fatal("既存ファイルが変わった")
			}
		}
		if code := run([]string{"generate", "-prompt", prompt, "-out", outPath, "-codex", fake, "-replace"}, &out, &errb); code != 0 {
			t.Fatalf("置き換え: %d %s", code, errb.String())
		}
		sent, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
		if !strings.Contains(string(sent), "<<<PROMPT\nA quiet stone city.\nPROMPT>>>") {
			t.Fatalf("Codex へ渡した依頼 = %q", sent)
		}
	})
}

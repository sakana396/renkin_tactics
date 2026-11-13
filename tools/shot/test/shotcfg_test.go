package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/tools/shot/shotcfg"
)

// 引数なし（-out だけ）の既定は今までと同じ: base・HiRes なし・ch01・title。
func TestParseDefaults(t *testing.T) {
	t.Run("TC-DES-NG-10-34 引数の既定は今までと同じ", testParseDefaults)
}

func testParseDefaults(t *testing.T) {
	var stderr bytes.Buffer
	o, err := shotcfg.Parse([]string{"-out", "x.png"}, &stderr)
	if err != nil {
		t.Fatal(err, stderr.String())
	}
	want := shotcfg.Options{Game: "base", HiRes: false, Chapter: "ch01", Scene: "title", Out: "x.png"}
	if o != want {
		t.Fatalf("既定 = %+v、期待 = %+v", o, want)
	}
	cfg, opt, err := shotcfg.Build(o)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HiRes || cfg.Media != nil || cfg.StartChapter != "ch01" || opt.Scene != "title" || opt.Path != "x.png" || len(opt.Steps) != 0 {
		t.Fatalf("base の既定の設定が違う: %+v %+v", cfg, opt)
	}
}

// -game alchemy は新作の定義データと media を使い、-hires で高解像度にする。-chapter は開始章になる。
func TestParseAlchemy(t *testing.T) {
	t.Run("TC-DES-NG-10-34 -game alchemy・-hires・-chapter・-scene gameover", testParseAlchemy)
}

func testParseAlchemy(t *testing.T) {
	var stderr bytes.Buffer
	o, err := shotcfg.Parse([]string{"-game", "alchemy", "-hires", "-chapter", "ch02", "-scene", "gameover", "-steps", "choose:0,tick:3", "-out", "y.png"}, &stderr)
	if err != nil {
		t.Fatal(err, stderr.String())
	}
	cfg, opt, err := shotcfg.Build(o)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.HiRes || cfg.Media == nil || cfg.StartChapter != "ch02" || opt.Scene != "gameover" || len(opt.Steps) != 2 {
		t.Fatalf("alchemy の設定が違う: %+v %+v", cfg, opt)
	}
	// 新作の定義データには、base にない章がある（章の一覧が alchemy のもの）。
	found := false
	for _, ch := range cfg.Catalog.Chapters() {
		if ch.ID == "ch02" && ch.Name == "水路の向こう" {
			found = true
		}
	}
	if !found {
		t.Fatal("alchemy の定義データではない")
	}
	// -hires なしの alchemy は HiRes が偽（media は使う）。
	o, _ = shotcfg.Parse([]string{"-game", "alchemy", "-out", "z.png"}, &stderr)
	if cfg, _, err = shotcfg.Build(o); err != nil || cfg.HiRes || cfg.Media == nil {
		t.Fatal(err, cfg.HiRes, cfg.Media)
	}
}

func TestParseErrors(t *testing.T) {
	t.Run("TC-DES-NG-10-34 引数の誤りは誤りにする", testParseErrors)
}

func testParseErrors(t *testing.T) {
	for name, args := range map[string][]string{
		"-out なし": {"-game", "alchemy"},
		"知らない作品":  {"-game", "other", "-out", "a.png"},
		"知らない引数":  {"-nothing", "-out", "a.png"},
	} {
		var stderr bytes.Buffer
		if _, err := shotcfg.Parse(args, &stderr); err == nil || stderr.Len() == 0 {
			t.Errorf("%s: 誤りにならない: %v %q", name, err, stderr.String())
		}
	}
	o, _ := shotcfg.Parse([]string{"-chapter", "ch99", "-out", "a.png"}, &bytes.Buffer{})
	if _, _, err := shotcfg.Build(o); err == nil || !strings.Contains(err.Error(), "ch99") {
		t.Errorf("存在しない章が誤りにならない: %v", err)
	}
}

// WSL の実行口は、足した引数（-game・-hires）もそのまま渡す。
func TestShotWrapperNewFlags(t *testing.T) {
	t.Run("TC-DES-NG-10-34 WSL の実行口は足した引数をそのまま渡す", testShotWrapperNewFlags)
}

func testShotWrapperNewFlags(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"go": `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then
    shift
    exe="$1"
    break
  fi
  shift
done
cat > "$exe" <<'SCRIPT'
#!/bin/sh
printf '%s\n' "$@" > "$SHOT_TEST_LOG"
SCRIPT
chmod +x "$exe"
`,
		"wslpath": "#!/bin/sh\nprintf 'WIN:%s\\n' \"$2\"\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "sample.png")
	log := filepath.Join(dir, "args.txt")
	cmd := exec.Command("bash", "../shot.sh", "-game", "alchemy", "-hires", "-chapter", "ch02", "-scene", "gameover", "-out", out)
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "SHOT_TEST_LOG="+log)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("実行に失敗: %v\n%s", err, b)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	want := "-game\nalchemy\n-hires\n-chapter\nch02\n-scene\ngameover\n-out\nWIN:" + out + "\n"
	if string(b) != want {
		t.Fatalf("渡した引数 = %q、期待 = %q", strings.TrimSpace(string(b)), strings.TrimSpace(want))
	}
}

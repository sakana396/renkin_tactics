package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// WSL の実行口は出力先だけ Windows のパスに変え、操作列をそのまま渡す。
func TestShotWrapper(t *testing.T) {
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
	out := filepath.Join(dir, "images", "sample.png")
	log := filepath.Join(dir, "args.txt")
	cmd := exec.Command("bash", "../shot.sh", "-chapter", "ch09", "-scene", "battle", "-steps", "rewind,cursor:1:2", "-out", out)
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "SHOT_TEST_LOG="+log)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("実行に失敗: %v\n%s", err, b)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	want := "-chapter\nch09\n-scene\nbattle\n-steps\nrewind,cursor:1:2\n-out\nWIN:" + out + "\n"
	if string(b) != want {
		t.Fatalf("渡した引数 = %q、期待 = %q", strings.TrimSpace(string(b)), strings.TrimSpace(want))
	}
}

package savefile

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlayLogs(t *testing.T) {
	at := time.Date(2026, 11, 2, 15, 30, 0, 0, time.Local)
	clock := func() time.Time { return at }
	t.Run("TC-IF-04-08 セーブのディレクトリの下の logs に、時刻と章の名前で作り、追記する", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "save")
		w, err := PlayLogs{Dir: dir, Now: clock}.Create("ch02")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "logs", "20261102-153000-ch02.jsonl")
		if got, err := os.ReadFile(path); err != nil || len(got) != 0 {
			t.Fatalf("作った直後のファイル %q, %v", got, err)
		}
		for _, line := range []string{"{\"a\":1}\n", "{\"b\":2}\n"} {
			if n, err := io.WriteString(w, line); err != nil || n != len(line) {
				t.Fatalf("Write = %d, %v", n, err)
			}
			// 1 行ごとにファイルを閉じるので、書いた行はすぐに読める（途中でやめても残る）。
			got, _ := os.ReadFile(path)
			if want := map[string]string{"{\"a\":1}\n": "{\"a\":1}\n", "{\"b\":2}\n": "{\"a\":1}\n{\"b\":2}\n"}[line]; string(got) != want {
				t.Fatalf("中身 %q; 期待 %q", got, want)
			}
		}
	})
	t.Run("TC-IF-04-08 同じ名前があれば番号を足し、前の記録を上書きしない", func(t *testing.T) {
		dir := t.TempDir()
		logs := PlayLogs{Dir: dir, Now: clock}
		first, err := logs.Create("ch01")
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(first, "old\n")
		second, err := logs.Create("ch01")
		if err != nil {
			t.Fatal(err)
		}
		third, err := logs.Create("ch01")
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(second, "new\n")
		for name, want := range map[string]string{"20261102-153000-ch01.jsonl": "old\n", "20261102-153000-ch01-2.jsonl": "new\n", "20261102-153000-ch01-3.jsonl": ""} {
			got, err := os.ReadFile(filepath.Join(dir, "logs", name))
			if err != nil || string(got) != want {
				t.Fatalf("%s = %q, %v; 期待 %q", name, got, err, want)
			}
		}
		if third.(AppendFile).Path != filepath.Join(dir, "logs", "20261102-153000-ch01-3.jsonl") {
			t.Fatal(third)
		}
	})
	t.Run("TC-IF-04-08 章 ID のファイル名に使えない文字は _ にする", func(t *testing.T) {
		dir := t.TempDir()
		w, err := PlayLogs{Dir: dir, Now: clock}.Create("../x y")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, "logs", "20261102-153000-___x_y.jsonl"); w.(AppendFile).Path != want {
			t.Fatalf("パス %q; 期待 %q", w.(AppendFile).Path, want)
		}
	})
	t.Run("TC-IF-04-08 作れない場所と、消えたファイルへの追記は誤り", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "file")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := (PlayLogs{Dir: file, Now: clock}).Create("ch01"); err == nil {
			t.Fatal("ファイルの下に logs を作れた")
		}
		w, err := PlayLogs{Dir: dir, Now: clock}.Create("ch01")
		if err != nil {
			t.Fatal(err)
		}
		path := w.(AppendFile).Path
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, "x\n"); err == nil {
			t.Fatal("消えたファイルに書けた")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("消えたファイルを作り直した")
		}
	})
}

package audio

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
)

func mediaFS(t *testing.T) fs.FS {
	t.Helper()
	return data.AlchemyMedia()
}

// TestAudioLedger は音の台帳 ASSETS-audio.md の検査（DES-NG-09）。
// あとで台帳が ASSETS.md に 1 つにまとまったら、data の台帳の検査に任せ、このテストは外す。
func TestAudioLedger(t *testing.T) {
	media := mediaFS(t)
	files := map[string]bool{}
	total := int64(0)
	_ = fs.WalkDir(media, "audio", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files[p] = true
			if info, e := d.Info(); e == nil {
				total += info.Size()
			}
		}
		return nil
	})
	raw, err := fs.ReadFile(media, "ASSETS-audio.md")
	if err != nil {
		t.Fatalf("ASSETS-audio.md を読めない: %v", err)
	}
	rows := map[string][]string{}
	inTable := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if cells[0] == "ファイル" {
			inTable = true
			continue
		}
		if !inTable || strings.Trim(cells[0], "-: ") == "" {
			continue
		}
		rows[strings.Trim(cells[0], "` ")] = cells
	}

	t.Run("TC-DES-NG-09-17 audio の下のすべてのファイルが、台帳に行を持つ", func(t *testing.T) {
		if len(files) == 0 {
			t.Fatal("audio の下にファイルがない")
		}
		for f := range files {
			if _, ok := rows[f]; !ok {
				t.Errorf("%s が ASSETS-audio.md にない", f)
			}
		}
	})
	t.Run("TC-DES-NG-09-17 台帳の各行のファイルが実在し、7 列がすべて埋まっている", func(t *testing.T) {
		for f, cells := range rows {
			if !files[f] {
				t.Errorf("台帳の %s が実在しない", f)
			}
			if len(cells) != 7 {
				t.Errorf("%s: 列の数 = %d, 期待 7", f, len(cells))
				continue
			}
			for i, c := range cells {
				if c == "" || c == "-" {
					t.Errorf("%s: %d 列目が空", f, i+1)
				}
			}
			if !strings.HasPrefix(cells[1], "自作") || !strings.Contains(cells[4], "CC0 1.0") {
				t.Errorf("%s: 作り方・利用条件が決まりの書き方でない", f)
			}
		}
	})
	t.Run("TC-DES-NG-09-18 audio の WAV の合計が 5 MB 以下", func(t *testing.T) {
		if total > 5*1024*1024 {
			t.Errorf("合計 %d バイトが 5 MB を超える", total)
		}
	})
}

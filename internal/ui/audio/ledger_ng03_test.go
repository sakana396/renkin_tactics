package audio

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
)

func mediaFS(t *testing.T) fs.FS {
	t.Helper()
	return data.AlchemyMedia()
}

// audioLedgerRows は台帳 ASSETS.md の表の行のうち、「ファイル」が audio/ で始まるものだけを返す（ファイル → 行の欄）。
// 画像（art/）の行は読まない。列の形式の検査は data.CheckMediaLedger に任せる（DES-NG-09、DES-NG-08）。
func audioLedgerRows(media fs.FS) (map[string][]string, error) {
	raw, err := fs.ReadFile(media, "ASSETS.md")
	if err != nil {
		return nil, err
	}
	rows := map[string][]string{}
	inFence := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		file := strings.Trim(cells[0], "` ")
		if strings.HasPrefix(file, "audio/") {
			rows[file] = cells
		}
	}
	return rows, nil
}

// checkAudioLedger は、audio の下のファイルと、台帳の audio/ の行が過不足なく対応することを検査し、問題の文を返す。
func checkAudioLedger(media fs.FS) []string {
	var problems []string
	files := map[string]bool{}
	_ = fs.WalkDir(media, "audio", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files[p] = true
		}
		return nil
	})
	rows, err := audioLedgerRows(media)
	if err != nil {
		return []string{"ASSETS.md を読めない: " + err.Error()}
	}
	for f := range files {
		if _, ok := rows[f]; !ok {
			problems = append(problems, f+" が ASSETS.md にない")
		}
	}
	for f := range rows {
		if !files[f] {
			problems = append(problems, "台帳の "+f+" が実在しない")
		}
	}
	return problems
}

// TestAudioLedger は音の台帳の検査（DES-NG-09）。audio/ の行だけを見る。
// 作り方・利用条件の中身（自作・CC0 など）は求めない（本物の音に差し替えても落ちないように）。
func TestAudioLedger(t *testing.T) {
	media := mediaFS(t)
	t.Run("TC-DES-NG-09-17 audio の下のすべてのファイルが台帳の audio/ の行を持ち、行のファイルが実在する", func(t *testing.T) {
		n := 0
		_ = fs.WalkDir(media, "audio", func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				n++
			}
			return nil
		})
		if n == 0 {
			t.Fatal("audio の下にファイルがない")
		}
		if problems := checkAudioLedger(media); len(problems) != 0 {
			t.Errorf("問題: %v", problems)
		}
	})
	t.Run("TC-DES-NG-09-17 画像（art/）の行を足しても、audio の検査は落ちない。audio/ の行の過不足は落ちる", func(t *testing.T) {
		row := func(f string) string { return "| `" + f + "` | AI 生成 | x | x | x | 2026-11-13 | x |\n" }
		head := "| ファイル | 作り方 | a | b | c | d | e |\n|---|---|---|---|---|---|---|\n"
		wav := &fstest.MapFile{Data: []byte("x")}
		ok := fstest.MapFS{
			"ASSETS.md":         {Data: []byte(head + row("audio/se/a.wav") + row("art/units/aoi.png") + row("art/title.png"))},
			"audio/se/a.wav":    wav,
			"art/units/aoi.png": wav,
			"art/title.png":     wav,
		}
		if problems := checkAudioLedger(ok); len(problems) != 0 {
			t.Errorf("art の行で audio の検査が落ちた: %v", problems)
		}
		noRow := fstest.MapFS{"ASSETS.md": {Data: []byte(head + row("art/title.png"))}, "audio/se/a.wav": wav}
		if problems := checkAudioLedger(noRow); len(problems) != 1 || !strings.Contains(problems[0], "audio/se/a.wav が ASSETS.md にない") {
			t.Errorf("行のない音を検出できない: %v", problems)
		}
		noFile := fstest.MapFS{"ASSETS.md": {Data: []byte(head + row("audio/se/b.wav"))}, "audio/se/a.wav": wav}
		if problems := checkAudioLedger(noFile); len(problems) != 2 {
			t.Errorf("実在しない行と行のない音の 2 件のはずが %v", problems)
		}
	})
	t.Run("TC-DES-NG-09-18 audio の WAV の合計が 5 MB 以下", func(t *testing.T) {
		total := int64(0)
		_ = fs.WalkDir(media, "audio", func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				if info, e := d.Info(); e == nil {
					total += info.Size()
				}
			}
			return nil
		})
		if total > 5*1024*1024 {
			t.Errorf("合計 %d バイトが 5 MB を超える", total)
		}
	})
}

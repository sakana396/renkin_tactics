package savefile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PlayLogDirName は、セーブのディレクトリの下で行動の記録を置くディレクトリの名前（DES-LOG-02）。
const PlayLogDirName = "logs"

// PlayLogs は、セーブのディレクトリ Dir の下の logs に行動の記録のファイルを作る（DES-LOG-02）。
// Now は呼び出し元（cmd）が渡す時計で、ファイル名にだけ使う。記録の中身は知らない。
type PlayLogs struct {
	Dir string
	Now func() time.Time
}

// Create は戦闘 1 回分の空の記録のファイルを作り、そのファイルへ追記する口を返す（DES-LOG-02）。
//
// 名前は「年月日-時分秒-章 ID.jsonl」（例: 20261102-153000-ch02.jsonl）。同じ名前があれば
// 「-2」「-3」…を足し、前のファイルを上書きしない。章 ID の英数字・「_」・「-」以外の文字は「_」にする。
func (p PlayLogs) Create(chapter string) (io.Writer, error) {
	dir := filepath.Join(p.Dir, PlayLogDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("行動の記録のディレクトリを作る %q: %w", dir, err)
	}
	base := p.Now().Format("20060102-150405") + "-" + safeName(chapter)
	for n := 1; n <= 1000; n++ {
		name := base
		if n > 1 {
			name = fmt.Sprintf("%s-%d", base, n)
		}
		path := filepath.Join(dir, name+".jsonl")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("行動の記録を作る %q: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("行動の記録を閉じる %q: %w", path, err)
		}
		return AppendFile{Path: path}, nil
	}
	return nil, fmt.Errorf("行動の記録の名前 %q が使い切られています", base)
}

// safeName は章 ID をファイル名に使える文字だけにする（DES-LOG-02）。
func safeName(s string) string {
	if s == "" {
		return "_"
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}

// AppendFile は、Write のたびに Path を追記で開いて書き、閉じる（DES-LOG-02）。
// ファイルを開いたままにしないので、途中で遊ぶ版を閉じても、それまでに書いた行は残る。
type AppendFile struct{ Path string }

// Write は p をファイルの末尾に足す（DES-LOG-02）。ファイルがなければ誤りにする（作り直さない）。
func (a AppendFile) Write(p []byte) (n int, err error) {
	f, err := os.OpenFile(a.Path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return 0, fmt.Errorf("行動の記録を開く %q: %w", a.Path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("行動の記録を閉じる %q: %w", a.Path, closeErr)
		}
	}()
	n, err = f.Write(p)
	if err != nil {
		return n, fmt.Errorf("行動の記録に書く %q: %w", a.Path, err)
	}
	return n, nil
}

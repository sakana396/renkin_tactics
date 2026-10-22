// Package savefile はセーブの置き場所とファイルの置き換えを扱う（DES-PKG-08）。
// セーブの中身は呼び出し元が読み書きする。
package savefile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Store は Dir の下のセーブファイルを扱う（DES-SAV-02）。
// 同じ Dir への Write は呼び出し元で直列に実行する。
type Store struct{ Dir string }

// Path はセーブファイルのパスを返す（DES-SAV-02）。
func (s Store) Path() string { return filepath.Join(s.Dir, "save.json") }

// Exists はセーブファイルの有無を返す（DES-SAV-02）。
func (s Store) Exists() (bool, error) {
	_, err := os.Stat(s.Path())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("セーブの存在を確認 %q: %w", s.Path(), err)
	}
	return true, nil
}

// Read はセーブファイルを開き、fn に読ませてから閉じる（DES-SAV-02）。
func (s Store) Read(fn func(io.Reader) error) (err error) {
	f, err := os.Open(s.Path())
	if err != nil {
		return fmt.Errorf("セーブを開く %q: %w", s.Path(), err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("セーブを閉じる %q: %w", s.Path(), closeErr))
		}
	}()
	if err := fn(f); err != nil {
		return fmt.Errorf("セーブを読む %q: %w", s.Path(), err)
	}
	return nil
}

// Write は一時ファイルを同期して閉じてからセーブを置き換える（DES-SAV-02）。
// 置き換えまでに失敗したら旧セーブを保ち、一時ファイルを片付ける。
func (s Store) Write(fn func(io.Writer) error) error {
	return s.write(fn, func(dir, pattern string) (tempFile, error) { return os.CreateTemp(dir, pattern) }, os.Rename)
}

// tempFile は同期・クローズの失敗も試験できる一時ファイルの口（DES-SAV-02）。
type tempFile interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}

// write は OS 操作を受け取り、置き換えの手順を実行する（DES-SAV-02）。
func (s Store) write(fn func(io.Writer) error, createTemp func(string, string) (tempFile, error), rename func(string, string) error) (err error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("セーブのディレクトリを作る %q: %w", s.Dir, err)
	}
	// 中断で残った一時ファイルだけを、新しいファイルを作る前に消す。
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return fmt.Errorf("一時ファイルを探す %q: %w", s.Dir, err)
	}
	for _, entry := range entries {
		matched, _ := filepath.Match("save-*.tmp", entry.Name())
		if !matched {
			continue
		}
		path := filepath.Join(s.Dir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("残った一時ファイルを消す %q: %w", path, err)
		}
	}
	f, err := createTemp(s.Dir, "save-*.tmp")
	if err != nil {
		return fmt.Errorf("一時ファイルを作る %q: %w", s.Dir, err)
	}
	closed, replaced := false, false
	defer func() {
		if !closed {
			if closeErr := f.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("一時ファイルを閉じる %q: %w", f.Name(), closeErr))
			}
		}
		if !replaced {
			if removeErr := os.Remove(f.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("一時ファイルを消す %q: %w", f.Name(), removeErr))
			}
		}
	}()
	if err := fn(f); err != nil {
		return fmt.Errorf("一時ファイルを書く %q: %w", f.Name(), err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("一時ファイルを同期 %q: %w", f.Name(), err)
	}
	closed = true
	if err := f.Close(); err != nil {
		return fmt.Errorf("一時ファイルを閉じる %q: %w", f.Name(), err)
	}
	if err := rename(f.Name(), s.Path()); err != nil {
		return fmt.Errorf("セーブを置き換える %q: %w", s.Path(), err)
	}
	replaced = true
	return nil
}

package savefile

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeText(s Store, value string) error {
	return s.Write(func(w io.Writer) error { _, err := io.WriteString(w, value); return err })
}
func assertText(t *testing.T, s Store, want string) {
	t.Helper()
	err := s.Read(func(r io.Reader) error {
		b, err := io.ReadAll(r)
		if string(b) != want {
			t.Errorf("中身=%q; 期待=%q", b, want)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
func assertNoTemps(t *testing.T, s Store) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(s.Dir, "save-*.tmp"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("一時ファイル=%v, 誤り=%v", paths, err)
	}
}
func TestStore(t *testing.T) {
	t.Run("TC-IF-04-05 パスと存在確認と読み込み", func(t *testing.T) {
		s := Store{Dir: t.TempDir()}
		if s.Path() != filepath.Join(s.Dir, "save.json") {
			t.Fatal(s.Path())
		}
		if exists, err := s.Exists(); exists || err != nil {
			t.Fatalf("存在=%v, 誤り=%v", exists, err)
		}
		called := false
		if err := s.Read(func(io.Reader) error { called = true; return nil }); !errors.Is(err, fs.ErrNotExist) || called {
			t.Fatalf("呼び出し=%v, 誤り=%v", called, err)
		}
		if err := writeText(s, "保存した中身"); err != nil {
			t.Fatal(err)
		}
		if exists, err := s.Exists(); !exists || err != nil {
			t.Fatalf("存在=%v, 誤り=%v", exists, err)
		}
		assertText(t, s, "保存した中身")
		failure := errors.New("読み込みを拒否")
		if err := s.Read(func(io.Reader) error { return failure }); !errors.Is(err, failure) {
			t.Fatal(err)
		}
	})
	t.Run("TC-IF-04-06 存在確認と読み込みのOSエラー", func(t *testing.T) {
		// NUL を含むパスは、Linux と Windows のどちらでも不正になる。
		s := Store{Dir: filepath.Join(t.TempDir(), "invalid\x00dir")}
		if exists, err := s.Exists(); exists || err == nil {
			t.Fatalf("存在=%v, 誤り=%v", exists, err)
		}
		if err := s.Read(func(io.Reader) error { t.Fatal("呼ばれてはいけない"); return nil }); err == nil {
			t.Fatal("読み込みが成功した")
		}
	})
	t.Run("TC-NFR-08-01 新規保存と上書き", func(t *testing.T) {
		s := Store{Dir: filepath.Join(t.TempDir(), "nested", "save")}
		for _, value := range []string{"前のセーブ", "短", ""} {
			if err := writeText(s, value); err != nil {
				t.Fatal(err)
			}
			assertText(t, s, value)
			assertNoTemps(t, s)
		}
	})
	t.Run("TC-NFR-08-02 書き込み途中の失敗", func(t *testing.T) {
		for _, existing := range []bool{false, true} {
			s := Store{Dir: t.TempDir()}
			if existing {
				if err := writeText(s, "前のセーブ"); err != nil {
					t.Fatal(err)
				}
			}
			failure := errors.New("途中で失敗")
			err := s.Write(func(w io.Writer) error {
				if _, err := io.WriteString(w, "途中"); err != nil {
					t.Fatal(err)
				}
				if existing {
					assertText(t, s, "前のセーブ")
				}
				return failure
			})
			if !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if existing {
				assertText(t, s, "前のセーブ")
			} else if exists, err := s.Exists(); exists || err != nil {
				t.Fatalf("存在=%v, 誤り=%v", exists, err)
			}
			assertNoTemps(t, s)
		}
	})
	t.Run("TC-NFR-08-03 残った一時ファイルの片付け", func(t *testing.T) {
		s := Store{Dir: t.TempDir()}
		if err := writeText(s, "前のセーブ"); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"save-old.tmp", "save-other.tmp", "keep.tmp", "save-keep.json"} {
			if err := os.WriteFile(filepath.Join(s.Dir, name), []byte("未完了"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		assertText(t, s, "前のセーブ")
		if err := writeText(s, "次のセーブ"); err != nil {
			t.Fatal(err)
		}
		assertText(t, s, "次のセーブ")
		assertNoTemps(t, s)
		for _, name := range []string{"keep.tmp", "save-keep.json"} {
			if _, err := os.Stat(filepath.Join(s.Dir, name)); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("TC-NFR-08-04 ディレクトリ作成失敗", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, []byte("元のファイル"), 0o600); err != nil {
			t.Fatal(err)
		}
		s := Store{Dir: filepath.Join(blocker, "child")}
		if err := s.Write(func(io.Writer) error { t.Fatal("呼ばれてはいけない"); return nil }); err == nil {
			t.Fatal("成功した")
		}
		b, err := os.ReadFile(blocker)
		if err != nil || string(b) != "元のファイル" {
			t.Fatalf("中身=%q, 誤り=%v", b, err)
		}
	})
	t.Run("TC-NFR-08-05 一時ファイル作成・同期・クローズ・置き換え失敗", func(t *testing.T) {
		for _, stage := range []string{"create", "sync", "close", "rename"} {
			t.Run(stage, func(t *testing.T) {
				s := Store{Dir: t.TempDir()}
				if err := writeText(s, "前のセーブ"); err != nil {
					t.Fatal(err)
				}
				failure := errors.New("注入した失敗")
				create := func(dir, pattern string) (tempFile, error) {
					if stage == "create" {
						return nil, failure
					}
					if dir != s.Dir || pattern != "save-*.tmp" {
						t.Fatalf("一時ファイルの場所=%q, パターン=%q", dir, pattern)
					}
					f, err := os.CreateTemp(dir, pattern)
					if err != nil {
						return nil, err
					}
					return &failingFile{File: f, stage: stage, failure: failure}, nil
				}
				rename := func(old, new string) error {
					if stage == "rename" {
						return failure
					}
					return os.Rename(old, new)
				}
				err := s.write(func(w io.Writer) error { _, err := io.WriteString(w, "次のセーブ"); return err }, create, rename)
				if !errors.Is(err, failure) || !strings.Contains(err.Error(), strings.Trim(strconv.Quote(s.Dir), "\"")) {
					t.Fatalf("誤り=%v", err)
				}
				assertText(t, s, "前のセーブ")
				assertNoTemps(t, s)
			})
		}
	})
	t.Run("TC-NFR-08-06 実際の置き換え失敗", func(t *testing.T) {
		s := Store{Dir: t.TempDir()}
		if err := os.Mkdir(s.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(s.Path(), "keep")
		if err := os.WriteFile(marker, []byte("元の中身"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeText(s, "次のセーブ"); err == nil {
			t.Fatal("置き換えが成功した")
		}
		if b, err := os.ReadFile(marker); err != nil || string(b) != "元の中身" {
			t.Fatalf("中身=%q, 誤り=%v", b, err)
		}
		assertNoTemps(t, s)
	})
}

type failingFile struct {
	*os.File
	stage   string
	failure error
}

func (f *failingFile) Sync() error {
	if f.stage == "sync" {
		return f.failure
	}
	return f.File.Sync()
}
func (f *failingFile) Close() error {
	err := f.File.Close()
	if f.stage == "close" {
		return f.failure
	}
	return err
}

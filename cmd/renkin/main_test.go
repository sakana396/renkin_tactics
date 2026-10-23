package main

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/savefile"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// copyBase は本体のデータを写した fstest.MapFS を返す。
func copyBase(t *testing.T) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	err := fs.WalkDir(data.Base(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), path)
		if err != nil {
			return err
		}
		out[path] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func fixedNow() time.Time { return time.Unix(0, 12345) }

// 起動の流れを画面を開かずに確かめる（DES-CMD-02）。
func TestRun(t *testing.T) {
	t.Run("TC-FR-SYS-04-22 定義の誤りはすべて表示し画面を動かさない", func(t *testing.T) {
		fsys := copyBase(t)
		fsys["chapters.toml"] = &fstest.MapFile{Data: []byte(strings.Replace(string(fsys["chapters.toml"].Data), `map = "ch01"`, `map = "ch99"`, 1))}
		fsys["items.toml"] = &fstest.MapFile{Data: []byte("[[items]]\nid = \"potion\"\nname = \"回復薬\"\nheal = 0\n")}
		var stderr bytes.Buffer
		started := false
		code := run(nil, &stderr, fsys, fixedNow, func(ui.Config) error { started = true; return nil })
		if code != 1 || started {
			t.Fatal(code, started)
		}
		for _, want := range []string{"定義データに誤りがあります", "chapters.toml: chapters[0].map", "ch99", "items.toml: items[0].heal"} {
			if !strings.Contains(stderr.String(), want) {
				t.Error(stderr.String())
			}
		}
	})
	t.Run("TC-FR-SYS-04-23 正しい目録と保存先と種を画面へ渡す", func(t *testing.T) {
		var stderr bytes.Buffer
		var cfg ui.Config
		dir := t.TempDir()
		code := run([]string{"-seed", "7", "-savedir", dir}, &stderr, data.Base(), fixedNow, func(c ui.Config) error { cfg = c; return nil })
		if code != 0 || cfg.Catalog == nil || cfg.Store == nil || cfg.NextSeed == nil || cfg.StartChapter != "" || stderr.Len() != 0 {
			t.Fatal(code, stderr.String())
		}
		store, ok := cfg.Store.(savefile.Store)
		if !ok || store.Dir != dir {
			t.Fatal(cfg.Store)
		}
		c, err := core.NewCampaign(cfg.Catalog, core.Withdraw)
		if err != nil {
			t.Fatal(err)
		}
		sortie, events, err := c.StartBattle([]int{0, 1, 2, 3, 4}, cfg.NextSeed())
		if err != nil || len(sortie.Battle.Units()) != 13 || len(events) != 1 {
			t.Fatal(sortie, events, err)
		}
		s := ui.NewSceneController(cfg)
		if s.Scene() != ui.Title {
			t.Fatal(s.Scene())
		}
		code = run([]string{"-savedir", dir}, &stderr, data.Base(), fixedNow, func(ui.Config) error { return errors.New("窓を開けない") })
		if code != 1 || !strings.Contains(stderr.String(), "窓を開けない") {
			t.Fatal(code, stderr.String())
		}
	})
}

func TestSeed(t *testing.T) {
	t.Run("TC-IF-01-17 戦闘ごとに種を取得し明示指定は 0 も固定する", func(t *testing.T) {
		for _, args := range [][]string{{"-seed", "7"}, {"-seed", "0"}, nil} {
			calls := 0
			now := func() time.Time { calls++; return time.Unix(0, int64(calls)) }
			var cfg ui.Config
			if code := run(args, &bytes.Buffer{}, data.Base(), now, func(c ui.Config) error { cfg = c; return nil }); code != 0 {
				t.Fatal(code)
			}
			if cfg.StartChapter != "" {
				t.Fatal("開始章未指定の Config が変わった")
			}
			if calls != 0 {
				t.Fatal("起動時に時刻取得")
			}
			for i := 1; i <= 2; i++ {
				want := uint64(i)
				if len(args) > 0 {
					want = 7
					if args[1] == "0" {
						want = 0
					}
				}
				if got := cfg.NextSeed(); got != want {
					t.Fatal(got, want)
				}
			}
			if len(args) > 0 && calls != 0 {
				t.Fatal(calls)
			}
		}
		started := false
		if code := run([]string{"-seed", "abc"}, &bytes.Buffer{}, data.Base(), fixedNow, func(ui.Config) error { started = true; return nil }); code != 2 || started {
			t.Fatal(code, started)
		}
	})
}

func TestSaveDirStartup(t *testing.T) {
	t.Run("TC-IF-04-07 保存先の解決と取得失敗時の終了", func(t *testing.T) {
		for _, osName := range []string{"windows", "darwin", "linux"} {
			for _, fail := range []bool{false, true} {
				getter := func() (string, error) {
					if fail {
						return "", errors.New("保存先取得失敗")
					}
					return "user", nil
				}
				var stderr bytes.Buffer
				started := false
				code := runWithDirEnv(nil, &stderr, data.Base(), fixedNow, func(cfg ui.Config) error {
					started = true
					store := cfg.Store.(savefile.Store)
					want := filepath.Join("user", "renkin_tactics")
					if osName == "linux" {
						want = filepath.Join("user", ".local", "share", "renkin_tactics")
					}
					if store.Dir != want {
						t.Fatal(store.Dir, want)
					}
					return nil
				}, savefile.DirEnv{OS: osName, UserConfigDir: getter, UserHomeDir: getter})
				if fail {
					if code != 1 || started || !strings.Contains(stderr.String(), "保存先取得失敗") {
						t.Fatal(code, started, stderr.String())
					}
				} else if code != 0 || !started || stderr.Len() != 0 {
					t.Fatal(code, started, stderr.String())
				}
			}
		}
	})
}

func TestStartChapterFlag(t *testing.T) {
	t.Run("TC-IF-01-37 開始章と専用保存先の検証", func(t *testing.T) {
		fsys := copyBase(t)
		fsys["chapters.toml"].Data = append(fsys["chapters.toml"].Data, []byte("\n[[chapters]]\nid = \"ch09\"\nname = \"試験用の最終章\"\nmap = \"ch03\"\n")...)
		for _, tc := range []struct {
			name    string
			args    []string
			code    int
			chapter string
			reasons []string
		}{
			{"指定", []string{"-chapter", "ch02", "-savedir", "dev"}, 0, "ch02", nil},
			{"4章目", []string{"-chapter", "ch09", "-savedir", "dev"}, 0, "ch09", nil},
			{"保存先なし", []string{"-chapter", "ch02"}, 2, "", []string{"-chapter", "-savedir"}},
			{"空の保存先", []string{"-chapter", "ch02", "-savedir", ""}, 2, "", []string{"-savedir"}},
			{"未知", []string{"-chapter", "missing", "-savedir", "dev"}, 2, "", []string{"missing", "ch01", "ch02", "ch03", "ch09"}},
			{"空の章", []string{"-chapter", "", "-savedir", "dev"}, 2, "", []string{"章", "ch01", "ch09"}},
			{"値なし", []string{"-chapter"}, 2, "", []string{"chapter"}},
			{"未指定", nil, 0, "", nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var stderr bytes.Buffer
				started := false
				code := run(tc.args, &stderr, fsys, fixedNow, func(cfg ui.Config) error {
					started = true
					if cfg.StartChapter != tc.chapter {
						t.Fatal(cfg.StartChapter, tc.chapter)
					}
					if tc.chapter != "" && cfg.Store.(savefile.Store).Dir != "dev" {
						t.Fatal(cfg.Store)
					}
					return nil
				})
				if code != tc.code || started != (tc.code == 0) {
					t.Fatal(code, started, stderr.String())
				}
				for _, reason := range tc.reasons {
					if !strings.Contains(stderr.String(), reason) {
						t.Fatal(stderr.String(), reason)
					}
				}
				if tc.code == 0 && stderr.Len() != 0 {
					t.Fatal(stderr.String())
				}
			})
		}
	})
}

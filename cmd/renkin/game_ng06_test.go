package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/savefile"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// 起動オプション -game（DES-NG-06）。
func TestGameFlag(t *testing.T) {
	chapterIDs := func(cfg ui.Config) string {
		var ids []string
		for _, ch := range cfg.Catalog.Chapters() {
			ids = append(ids, ch.ID)
		}
		return strings.Join(ids, ",")
	}
	t.Run("TC-DES-NG-06-06 -game alchemy は新作の 3 章を渡し、指定しないか base なら既存の作品のまま", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
			want string
		}{
			{"alchemy", []string{"-game", "alchemy", "-savedir", "dev"}, "ch01,ch02,ch03"},
			{"base を明示", []string{"-game", "base", "-savedir", "dev"}, "ch01,ch02,ch03,ch04,ch05,ch06,ch07,ch08,ch09"},
			{"指定なし", []string{"-savedir", "dev"}, "ch01,ch02,ch03,ch04,ch05,ch06,ch07,ch08,ch09"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var stderr bytes.Buffer
				var cfg ui.Config
				started := false
				code := run(tc.args, &stderr, data.Base(), fixedNow, func(c ui.Config) error { started = true; cfg = c; return nil })
				if code != 0 || !started || stderr.Len() != 0 {
					t.Fatal(code, started, stderr.String())
				}
				if got := chapterIDs(cfg); got != tc.want {
					t.Fatal(got, tc.want)
				}
			})
		}
	})
	t.Run("TC-DES-NG-06-07 不正な -game は誤りで終了コード 2、画面を動かさない", func(t *testing.T) {
		for _, args := range [][]string{{"-game", "foo"}, {"-game", ""}, {"-game", "Alchemy"}, {"-game"}} {
			var stderr bytes.Buffer
			started := false
			code := run(args, &stderr, data.Base(), fixedNow, func(ui.Config) error { started = true; return nil })
			if code != 2 || started {
				t.Fatal(args, code, started)
			}
			// 値が無いときは flag が、不正な値のときは run が理由を出す。
			if len(args) == 2 && (!strings.Contains(stderr.String(), "-game") || !strings.Contains(stderr.String(), "alchemy")) {
				t.Fatal(args, stderr.String())
			}
		}
	})
	t.Run("TC-DES-NG-06-08 新作のセーブは、-savedir がなければ既存の作品と別のディレクトリ。-savedir は指定どおり", func(t *testing.T) {
		env := savefile.DirEnv{OS: "linux", XDGDataHome: "xdg"}
		storeDir := func(args ...string) string {
			var dir string
			code := runWithDirEnv(args, &bytes.Buffer{}, data.Base(), fixedNow, func(c ui.Config) error { dir = c.Store.(savefile.Store).Dir; return nil }, env)
			if code != 0 {
				t.Fatal(args, code)
			}
			return dir
		}
		base := filepath.Join("xdg", "renkin_tactics")
		if got := storeDir(); got != base {
			t.Fatal(got)
		}
		if got := storeDir("-game", "base"); got != base {
			t.Fatal(got)
		}
		alchemy := storeDir("-game", "alchemy")
		if alchemy != filepath.Join("xdg", "renkin_tactics_alchemy") || alchemy == base {
			t.Fatal(alchemy)
		}
		if got := storeDir("-game", "alchemy", "-savedir", "mine"); got != "mine" {
			t.Fatal(got)
		}
	})
	t.Run("TC-DES-NG-06-09 -game alchemy でも -chapter は新作の章 ID を検査する", func(t *testing.T) {
		var stderr bytes.Buffer
		started := false
		if code := run([]string{"-game", "alchemy", "-chapter", "ch03", "-savedir", "dev"}, &stderr, data.Base(), fixedNow, func(c ui.Config) error { started = true; return nil }); code != 0 || !started {
			t.Fatal(code, stderr.String())
		}
		stderr.Reset()
		if code := run([]string{"-game", "alchemy", "-chapter", "ch04", "-savedir", "dev"}, &stderr, data.Base(), fixedNow, func(ui.Config) error { return nil }); code != 2 || !strings.Contains(stderr.String(), "ch04") || !strings.Contains(stderr.String(), "ch03") {
			t.Fatal(code, stderr.String())
		}
	})
}

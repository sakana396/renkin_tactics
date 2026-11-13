package main

import (
	"bytes"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// 高解像度の表示の切り替え（DES-NG-10）。
func TestHiResFlag(t *testing.T) {
	t.Run("TC-DES-NG-10-17 -game alchemy だけ HiRes を渡し、base と指定なしは渡さない", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
			want bool
		}{
			{"alchemy", []string{"-game", "alchemy", "-savedir", "dev"}, true},
			{"base を明示", []string{"-game", "base", "-savedir", "dev"}, false},
			{"指定なし", []string{"-savedir", "dev"}, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var stderr bytes.Buffer
				var cfg ui.Config
				started := false
				code := run(tc.args, &stderr, data.Base(), fixedNow, func(c ui.Config) error { started = true; cfg = c; return nil })
				if code != 0 || !started || stderr.Len() != 0 {
					t.Fatal(code, started, stderr.String())
				}
				if cfg.HiRes != tc.want {
					t.Fatal(cfg.HiRes, tc.want)
				}
			})
		}
	})
}

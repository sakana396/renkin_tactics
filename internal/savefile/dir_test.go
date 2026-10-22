package savefile

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveDir(t *testing.T) {
	failure := errors.New("場所を取得できない")
	for _, tc := range []struct {
		name, override, goos, xdg, config, home, want string
		configErr, homeErr                            error
	}{
		{name: "TC-IF-04-01 明示指定を最優先", override: "relative/save", goos: "windows", configErr: failure, homeErr: failure, want: "relative/save"},
		{name: "TC-IF-04-02 Windows", goos: "windows", xdg: "ignored", config: "AppData", want: filepath.Join("AppData", "renkin_tactics")},
		{name: "TC-IF-04-02 macOS", goos: "darwin", xdg: "ignored", config: "Library/Application Support", want: filepath.Join("Library/Application Support", "renkin_tactics")},
		{name: "TC-IF-04-03 Linux XDG", goos: "linux", xdg: "data", homeErr: failure, want: filepath.Join("data", "renkin_tactics")},
		{name: "TC-IF-04-03 Linux 既定", goos: "linux", home: "home", want: filepath.Join("home", ".local", "share", "renkin_tactics")},
		{name: "TC-IF-04-03 ほかのUnix", goos: "freebsd", home: "home", want: filepath.Join("home", ".local", "share", "renkin_tactics")},
		{name: "TC-IF-04-04 Windows 取得失敗", goos: "windows", configErr: failure},
		{name: "TC-IF-04-04 macOS 取得失敗", goos: "darwin", configErr: failure},
		{name: "TC-IF-04-04 Linux 取得失敗", goos: "linux", homeErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveDir(tc.override, DirEnv{OS: tc.goos, XDGDataHome: tc.xdg, UserConfigDir: func() (string, error) { return tc.config, tc.configErr }, UserHomeDir: func() (string, error) { return tc.home, tc.homeErr }})
			if tc.want == "" {
				if got != "" || !errors.Is(err, failure) {
					t.Fatalf("場所=%q, 誤り=%v", got, err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("場所=%q, 誤り=%v; 期待=%q", got, err, tc.want)
			}
		})
	}
}

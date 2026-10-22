package savefile

import (
	"fmt"
	"path/filepath"
)

// DirEnv は保存先の解決に使う OS と環境を呼び出し元から受け取る（DES-SAV-02）。
// OS には runtime.GOOS、XDGDataHome には os.Getenv("XDG_DATA_HOME")、
// 関数には os.UserConfigDir と os.UserHomeDir を渡す。
// 明示指定や OS の分岐で不要な関数は呼ばない。
type DirEnv struct {
	OS            string
	XDGDataHome   string
	UserConfigDir func() (string, error)
	UserHomeDir   func() (string, error)
}

// ResolveDir は -savedir の指定を優先して保存先を返す（DES-SAV-02、10.7 節）。
// ディレクトリの作成は Store.Write が行う。
func ResolveDir(override string, env DirEnv) (string, error) {
	if override != "" {
		return override, nil
	}
	var base string
	switch env.OS {
	case "windows", "darwin":
		dir, err := env.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("セーブの設定ディレクトリを取得: %w", err)
		}
		base = dir
	default:
		base = env.XDGDataHome
		if base == "" {
			home, err := env.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("セーブのホームディレクトリを取得: %w", err)
			}
			base = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(base, "renkin_tactics"), nil
}

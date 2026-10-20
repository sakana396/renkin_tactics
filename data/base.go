// Package data は本体の定義データを埋め込む（DES-PKG-07）。
package data

import (
	"embed"
	"io/fs"
)

//go:embed base/*.toml
var base embed.FS

// Base は本体の定義データ（data/base/*.toml）を、ファイル名だけで開ける fs.FS として返す（DES-PKG-07、DES-DAT-02）。
func Base() fs.FS {
	sub, err := fs.Sub(base, "base")
	if err != nil {
		// 埋め込みの道筋は固定なので、ここには来ない。
		panic(err)
	}
	return sub
}

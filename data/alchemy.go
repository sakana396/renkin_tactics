package data

import (
	"embed"
	"io/fs"
)

//go:embed alchemy/*.toml
var alchemy embed.FS

// Alchemy は新作「古代都市の復旧」の試作の定義データ（data/alchemy/*.toml）を、ファイル名だけで開ける fs.FS として返す（DES-NG-06）。
// 本体の定義データ（Base）とは別の、完結したデータ一式である。
func Alchemy() fs.FS {
	sub, err := fs.Sub(alchemy, "alchemy")
	if err != nil {
		// 埋め込みの道筋は固定なので、ここには来ない。
		panic(err)
	}
	return sub
}

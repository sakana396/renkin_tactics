package data

import (
	"embed"
	"io/fs"
)

//go:embed alchemy/media
var alchemyMedia embed.FS

// AlchemyMedia は新作の画像と音（data/alchemy/media）を、media の下のパスで開ける fs.FS として返す（DES-NG-07）。
// 定義データ（Alchemy）とは別に持つ。ファイルがなくても動くように、画面・音の側は、ない素材を飛ばす。
func AlchemyMedia() fs.FS {
	sub, err := fs.Sub(alchemyMedia, "alchemy/media")
	if err != nil {
		// 埋め込みの道筋は固定なので、ここには来ない。
		panic(err)
	}
	return sub
}

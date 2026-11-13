// Package main は、新作の仮の音（効果音と BGM）をコードで合成して WAV に書き出す（DES-NG-09）。
//
// 本物の音ではない。決まった乱数の種で毎回同じバイト列を出す（決定的）。標準の Go だけで動く。
// 実行: go run ./tools/synth [-out data/alchemy/media]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Render は、すべての音を合成して、道筋の順に WAV のバイト列で返す。
func Render() map[string][]byte {
	out := map[string][]byte{}
	for _, a := range allAssets() {
		out[a.Path] = encodeWAV(a.Make(newRNG(a.Seed)).pcm())
	}
	return out
}

func allAssets() []asset {
	return append(seAssets(), bgmAssets()...)
}

func main() {
	dir := flag.String("out", "data/alchemy/media", "書き出す media のディレクトリ")
	ledger := flag.Bool("ledger", false, "WAV は書かず、台帳（ASSETS-audio.md）の表の行を標準出力へ出す")
	flag.Parse()
	if *ledger {
		for _, a := range allAssets() {
			fmt.Println(ledgerRow(a, ledgerDate))
		}
		return
	}
	files := Render()
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	total := 0
	for _, p := range paths {
		dst := filepath.Join(*dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "誤り:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(dst, files[p], 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "誤り:", err)
			os.Exit(1)
		}
		total += len(files[p])
		fmt.Printf("%s（%d バイト）\n", dst, len(files[p]))
	}
	fmt.Printf("合計 %d バイト\n", total)
}

// ledgerDate は、台帳の行に書く「生成した日」「確かめた日」。再生成しても行が変わらないよう、固定する。
const ledgerDate = "2026-11-13"

// ledgerRow は台帳の 1 行（列は data/alchemy/media/ASSETS.md と同じ 7 列）を作る（DES-NG-09）。
func ledgerRow(a asset, date string) string {
	return fmt.Sprintf("| `%s` | 自作（合成。tools/synth、乱数の種 %d） | tools/synth（Go の標準ライブラリのみ。コミットの版は履歴を参照）、%s | %s | 自作。CC0 1.0 として扱う（既存の自作の画像と同じ条件。ライセンスの原本はない） | %s | 実装担当エージェント（合成した音を耳では確かめていない） |",
		a.Path, a.Seed, date, a.Desc, date)
}

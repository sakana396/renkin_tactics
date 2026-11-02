// Command replaylog は、行動の記録（JSON Lines）を今の定義データで流し直し、
// 同じ結果になるか、どの行で食い違ったかを表示する（DES-LOG-03）。
//
//	go run ./tools/replaylog <記録のファイル>...
//
// 食い違いも誤りもなければ終了コード 0、食い違いがあれば 1、記録が読めなければ 2 を返す。
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

var resultNames = map[tactics.Result]string{tactics.Ongoing: "決まっていない", tactics.Won: "勝ち", tactics.Lost: "負け"}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "使い方: go run ./tools/replaylog <記録のファイル>...")
		return 2
	}
	cat, err := defload.Load(data.Base())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	code := 0
	for _, path := range args {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			code = 2
			continue
		}
		rep, err := core.ReplayPlayLog(cat, f)
		f.Close()
		switch {
		case rep.FirstDiff > 0:
			fmt.Fprintf(stdout, "%s: %d 行目で食い違い\n  記録:     %s\n  流し直し: %s\n", path, rep.FirstDiff, rep.Want, rep.Got)
			code = max(code, 1)
		case err == nil:
			fmt.Fprintf(stdout, "%s: %d 行すべて同じ（%s）\n", path, rep.Lines, resultNames[rep.Result])
		}
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			code = 2
		}
	}
	return code
}

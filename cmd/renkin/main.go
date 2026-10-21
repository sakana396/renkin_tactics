// Command renkin は、レンキン・タクティクスを起動する（DES-PKG-01、DES-CMD-01）。
//
// 定義データを読み込んで検証し、乱数の種を決め、始めるマップの戦闘を作って画面を動かす。
// 起動と誤りの表示は、ふしぎ錬金（sakana396/fushigi_renkin の cmd/fushigi）と同じ形にした。
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr, data.Base(), time.Now, ui.Run))
}

// run は起動の流れを行い、終了コードを返す（DES-CMD-01）。
//
//  1. fsys の定義データを defload.Load で読む。誤りがあれば、すべてを stderr に書き、
//     Windows ではメッセージボックスでも出して、1 を返す（UC-01 2a、FR-SYS-04）
//  2. 乱数の種を決める（-seed、なければ時刻。DES-RNG-01）
//  3. 始めるマップの戦闘を作る関数を start に渡して、画面を動かす。やり直すたびに、その関数で種を決め直す
func run(args []string, stderr io.Writer, fsys fs.FS, now func() time.Time, start func(ui.Factory) error) int {
	flags := flag.NewFlagSet("renkin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Uint64("seed", 0, "乱数の種。指定すると、やり直しても同じ種を使う（指定しなければ時刻から決める）")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	seedSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})

	cat, err := defload.Load(fsys)
	if err != nil {
		message := fmt.Sprintf("定義データに誤りがあります:\n%v", err)
		fmt.Fprintln(stderr, message)
		reportDataError(message)
		return 1
	}

	nextSeed := seeder(seedSet, *seed, now)
	factory := func() (*tactics.Battle, []tactics.Event, error) {
		return core.NewBattle(cat, cat.Game.StartMap, nextSeed())
	}
	if err := start(factory); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// seeder は、戦闘を作るたびに使う種を返す関数を作る（DES-RNG-01）。
// -seed を指定していれば、いつも同じ種を返す。指定していなければ、呼んだときの時刻から種を作る。
func seeder(set bool, seed uint64, now func() time.Time) func() uint64 {
	if set {
		return func() uint64 { return seed }
	}
	return func() uint64 { return uint64(now().UnixNano()) }
}

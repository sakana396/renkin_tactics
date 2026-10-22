// Command renkin は、レンキン・タクティクスを起動する（DES-PKG-01、DES-CMD-02）。
//
// 定義データと保存先を読み、タイトルから始める。
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/savefile"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr, data.Base(), time.Now, ui.Run))
}

// run は定義データ・種・保存先を画面へ渡し、終了コードを返す（DES-CMD-02）。
func run(args []string, stderr io.Writer, fsys fs.FS, now func() time.Time, start func(ui.Config) error) int {
	return runWithDirEnv(args, stderr, fsys, now, start, savefile.DirEnv{OS: runtime.GOOS, XDGDataHome: os.Getenv("XDG_DATA_HOME"), UserConfigDir: os.UserConfigDir, UserHomeDir: os.UserHomeDir})
}

// runWithDirEnv は保存先の解決失敗も OS に依存せず検査する口（DES-CMD-02）。
func runWithDirEnv(args []string, stderr io.Writer, fsys fs.FS, now func() time.Time, start func(ui.Config) error, env savefile.DirEnv) int {
	flags := flag.NewFlagSet("renkin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	savedir := flags.String("savedir", "", "セーブのディレクトリ（指定しなければ OS の標準の場所）")
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
	dir, err := savefile.ResolveDir(*savedir, env)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := start(ui.Config{Catalog: cat, Store: savefile.Store{Dir: dir}, NextSeed: nextSeed}); err != nil {
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

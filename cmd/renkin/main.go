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
	"strings"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/savefile"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// 遊ぶ作品の名前（-game の値。DES-NG-06）。
const (
	gameBase    = "base"
	gameAlchemy = "alchemy"
	// alchemyDirSuffix は、新作のセーブの標準の置き場の名前の後ろに足す文字列。
	alchemyDirSuffix = "_alchemy"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr, data.Base(), time.Now, ui.Run))
}

// run は定義データ・種・保存先を画面へ渡し、終了コードを返す（DES-CMD-02・03）。
func run(args []string, stderr io.Writer, fsys fs.FS, now func() time.Time, start func(ui.Config) error) int {
	return runWithDirEnv(args, stderr, fsys, now, start, savefile.DirEnv{OS: runtime.GOOS, XDGDataHome: os.Getenv("XDG_DATA_HOME"), UserConfigDir: os.UserConfigDir, UserHomeDir: os.UserHomeDir})
}

// runWithDirEnv は保存先の解決失敗も OS に依存せず検査する口（DES-CMD-02・03）。
func runWithDirEnv(args []string, stderr io.Writer, fsys fs.FS, now func() time.Time, start func(ui.Config) error, env savefile.DirEnv) int {
	flags := flag.NewFlagSet("renkin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	savedir := flags.String("savedir", "", "セーブのディレクトリ（指定しなければ OS の標準の場所）")
	chapter := flags.String("chapter", "", "開発用の開始章 ID（-savedir も必須。成長は初期値）")
	game := flags.String("game", gameBase, "遊ぶ作品。base（既定。既存の作品）か alchemy（新作「古代都市の復旧」の試作 1〜3 章）")
	seed := flags.Uint64("seed", 0, "乱数の種。指定すると、やり直しても同じ種を使う（指定しなければ時刻から決める）")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	// 作品の選択（DES-NG-06）。alchemy は、渡された定義データ（既存の作品）の代わりに新作の定義データを使う。
	switch *game {
	case gameBase:
	case gameAlchemy:
		fsys = data.Alchemy()
	default:
		fmt.Fprintf(stderr, "-game %q は使えません。指定できる作品: %s, %s\n", *game, gameBase, gameAlchemy)
		return 2
	}
	seedSet := false
	chapterSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "chapter" {
			chapterSet = true
		}
		if f.Name == "seed" {
			seedSet = true
		}
	})

	if chapterSet && *savedir == "" {
		fmt.Fprintln(stderr, "-chapter を指定するときは、専用の保存先を -savedir で指定してください")
		return 2
	}

	cat, err := defload.Load(fsys)
	if err != nil {
		message := fmt.Sprintf("定義データに誤りがあります:\n%v", err)
		fmt.Fprintln(stderr, message)
		reportDataError(message)
		return 1
	}

	if chapterSet {
		var ids []string
		found := false
		for _, ch := range cat.Chapters() {
			ids = append(ids, ch.ID)
			if ch.ID == *chapter {
				found = true
			}
		}
		if !found {
			fmt.Fprintf(stderr, "章 %q はありません。指定できる章 ID: %s\n", *chapter, strings.Join(ids, ", "))
			return 2
		}
	}

	nextSeed := seeder(seedSet, *seed, now)
	dir, err := savefile.ResolveDir(*savedir, env)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// 新作のセーブは、-savedir を指定しないとき、既存の作品とは別のディレクトリ（標準の場所の renkin_tactics_alchemy）に置く。
	// セーブと行動の記録の置き場が分かれるので、savefile の仕組みは変えない（DES-NG-06）。
	if *game == gameAlchemy && *savedir == "" {
		dir += alchemyDirSuffix
	}
	// 行動の記録は、セーブのディレクトリの下の logs に、出撃ごとに 1 つ作る（DES-LOG-02）。
	// 同じ時計で、記録の経過の秒も測る（DES-LOG-01、E-106）。詳しい誤りは stderr へ出す。
	logs := savefile.PlayLogs{Dir: dir, Now: now}
	if err := start(ui.Config{Catalog: cat, Store: savefile.Store{Dir: dir}, NextSeed: nextSeed, StartChapter: *chapter,
		PlayLog: logs.Create, Now: now, Stderr: stderr}); err != nil {
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

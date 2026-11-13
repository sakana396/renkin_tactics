// Package shotcfg は、tools/shot の引数を読み、撮影に使う設定を作る。
// OS に依存しないので、Windows 以外でもテストできる（REV-109 F-07）。
package shotcfg

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/savefile"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// 撮る作品の名前（-game の値）。cmd/renkin の -game と同じ。
const (
	GameBase    = "base"
	GameAlchemy = "alchemy"
)

// Options は読んだ引数。
type Options struct {
	Game    string
	HiRes   bool
	Chapter string
	Scene   string
	Steps   string
	Out     string
	LogDir  string
}

type memoryStore struct{}

func (memoryStore) Exists() (bool, error) { return false, nil }
func (memoryStore) Read(func(io.Reader) error) error {
	return fmt.Errorf("セーブデータはありません")
}
func (memoryStore) Write(f func(io.Writer) error) error { return f(io.Discard) }

// Parse は引数を読む。誤りは stderr に書き、誤りを返す。
// 引数なしの既定は今までと同じ（-game base、-hires なし、-chapter ch01、-scene title）。
func Parse(args []string, stderr io.Writer) (Options, error) {
	var o Options
	fl := flag.NewFlagSet("shot", flag.ContinueOnError)
	fl.SetOutput(stderr)
	fl.StringVar(&o.Game, "game", GameBase, "撮る作品: base（既存の作品）か alchemy（新作。定義データと media を使う）")
	fl.BoolVar(&o.HiRes, "hires", false, "高解像度の表示（1 マス 48 px）にする。新作の見た目を撮るときは -game alchemy と一緒に付ける")
	fl.StringVar(&o.Chapter, "chapter", "ch01", "開始章 ID")
	fl.StringVar(&o.Scene, "scene", "title", "場面: title, rule, story, prepare, deploy, battle, gameover")
	fl.StringVar(&o.Steps, "steps", "", "操作列。カンマで区切る。詳しくは README.md")
	fl.StringVar(&o.Out, "out", "", "PNG の出力先")
	fl.StringVar(&o.LogDir, "logdir", "", "行動の記録を書くディレクトリ（その下の logs に作る。指定しなければ記録しない）")
	if err := fl.Parse(args); err != nil {
		return o, err
	}
	if o.Out == "" {
		fmt.Fprintln(stderr, "-out で PNG の出力先を指定してください")
		return o, fmt.Errorf("-out がありません")
	}
	if o.Game != GameBase && o.Game != GameAlchemy {
		fmt.Fprintf(stderr, "-game %q は使えません。指定できる作品: %s, %s\n", o.Game, GameBase, GameAlchemy)
		return o, fmt.Errorf("-game %q は使えません", o.Game)
	}
	return o, nil
}

// Build は読んだ引数から、ゲームの設定と撮影の指定を作る。
// alchemy は新作の定義データと media を、base は既存の定義データだけを使う（media なし）。
func Build(o Options) (ui.Config, ui.ShotOptions, error) {
	var fsys fs.FS = data.Base()
	var media fs.FS
	if o.Game == GameAlchemy {
		fsys, media = data.Alchemy(), data.AlchemyMedia()
	}
	cat, err := defload.Load(fsys)
	if err != nil {
		return ui.Config{}, ui.ShotOptions{}, err
	}
	found := false
	var ids []string
	for _, ch := range cat.Chapters() {
		ids = append(ids, ch.ID)
		if ch.ID == o.Chapter {
			found = true
		}
	}
	if !found {
		return ui.Config{}, ui.ShotOptions{}, fmt.Errorf("章 %q はありません。指定できる章 ID: %s", o.Chapter, strings.Join(ids, ", "))
	}
	var actions []string
	if o.Steps != "" {
		actions = strings.Split(o.Steps, ",")
	}
	cfg := ui.Config{Catalog: cat, Media: media, HiRes: o.HiRes, Store: memoryStore{}, NextSeed: func() uint64 { return 1 }, StartChapter: o.Chapter}
	if o.LogDir != "" {
		// 遊ぶ版と同じ書き出し先を使う（DES-LOG-02）。
		cfg.PlayLog, cfg.Now, cfg.Stderr = savefile.PlayLogs{Dir: o.LogDir, Now: time.Now}.Create, time.Now, os.Stderr
	}
	return cfg, ui.ShotOptions{Scene: o.Scene, Steps: actions, Path: o.Out}, nil
}

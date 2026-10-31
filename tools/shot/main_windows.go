//go:build windows

// Command shot は、開発中の画面を PNG に書き出す。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

type memoryStore struct{}

func (memoryStore) Exists() (bool, error) { return false, nil }
func (memoryStore) Read(func(io.Reader) error) error {
	return fmt.Errorf("セーブデータはありません")
}
func (memoryStore) Write(f func(io.Writer) error) error { return f(io.Discard) }

func main() {
	chapter := flag.String("chapter", "ch01", "開始章 ID")
	scene := flag.String("scene", "title", "場面: title, rule, story, prepare, deploy, battle")
	steps := flag.String("steps", "", "操作列。カンマで区切る。詳しくは README.md")
	out := flag.String("out", "", "PNG の出力先")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "-out で PNG の出力先を指定してください")
		os.Exit(2)
	}
	cat, err := defload.Load(data.Base())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	found := false
	for _, ch := range cat.Chapters() {
		if ch.ID == *chapter {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "章 %q はありません\n", *chapter)
		os.Exit(2)
	}
	var actions []string
	if *steps != "" {
		actions = strings.Split(*steps, ",")
	}
	err = ui.RunShot(ui.Config{Catalog: cat, Store: memoryStore{}, NextSeed: func() uint64 { return 1 }, StartChapter: *chapter}, ui.ShotOptions{Scene: *scene, Steps: actions, Path: *out})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

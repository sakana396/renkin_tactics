package main

import (
	"fmt"
	"io"

	"github.com/sakana396/renkin_tactics/internal/ui"
	"github.com/sakana396/renkin_tactics/tools/shot/shotcfg"
)

// run は引数を読んで撮影し、終了コードを返す。
func run(args []string, stderr io.Writer) int {
	o, err := shotcfg.Parse(args, stderr)
	if err != nil {
		return 2
	}
	cfg, opt, err := shotcfg.Build(o)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := ui.RunShot(cfg, opt); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

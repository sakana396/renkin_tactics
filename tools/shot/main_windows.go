//go:build windows

// Command shot は、開発中の画面を PNG に書き出す。
package main

import "os"

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

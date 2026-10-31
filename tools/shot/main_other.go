//go:build !windows

// Command shot は、WSL では shot.sh から Windows 版を起動する。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "WSL では tools/shot/shot.sh を使ってください")
	os.Exit(2)
}

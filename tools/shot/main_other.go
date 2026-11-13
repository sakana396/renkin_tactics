//go:build !windows

// Command shot は、WSL では shot.sh から Windows 版を起動する。
// Linux で直接撮るときは、環境変数 SHOT_NATIVE=1 を付け、xvfb-run -a の下で実行する。
package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Getenv("SHOT_NATIVE") == "1" {
		os.Exit(run(os.Args[1:], os.Stderr))
	}
	fmt.Fprintln(os.Stderr, "WSL では tools/shot/shot.sh を使ってください")
	os.Exit(2)
}

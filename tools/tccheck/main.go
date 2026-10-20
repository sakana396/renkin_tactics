package main

import (
	"fmt"
	"os"
)

// main はリポジトリのルートで実行し、誤りがあれば終了コード 1 を返す（QA-02）。
func main() {
	r := Check(os.DirFS("."))
	for _, w := range r.Warnings {
		fmt.Println("警告:", w)
	}
	for _, e := range r.Errors {
		fmt.Println("誤り:", e)
	}
	if len(r.Errors) > 0 {
		fmt.Printf("テストケース ID の整合: 誤り %d 件\n", len(r.Errors))
		os.Exit(1)
	}
	fmt.Printf("テストケース ID の整合: OK（警告 %d 件）\n", len(r.Warnings))
}

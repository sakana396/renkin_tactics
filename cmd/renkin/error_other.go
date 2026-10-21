//go:build !windows

package main

// reportDataError は、Windows 以外では何もしない（DES-CMD-01）。誤りは run が標準エラーに書く。
func reportDataError(string) {}

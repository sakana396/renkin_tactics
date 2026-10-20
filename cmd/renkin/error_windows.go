package main

import (
	"golang.org/x/sys/windows"
)

// reportDataError は、定義データの誤りを Windows のメッセージボックスでも出す（DES-CMD-01、UC-01 2a）。
// 標準エラーへは run が書く。ふしぎ錬金（sakana396/fushigi_renkin の cmd/fushigi/error_windows.go）と同じ形。
func reportDataError(message string) {
	caption, err := windows.UTF16PtrFromString("レンキン・タクティクス")
	if err != nil {
		return
	}
	body, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	_, _ = windows.MessageBox(0, body, caption, windows.MB_OK|windows.MB_ICONERROR)
}

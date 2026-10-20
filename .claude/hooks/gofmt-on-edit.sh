#!/usr/bin/env bash
# PostToolUse(Write|Edit): 編集された .go ファイルに gofmt をかける。
# 入力はフックの JSON（stdin）。jq がない環境でも動くよう python3 で読む。
f=$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))')
case "$f" in
  *.go) [ -f "$f" ] && gofmt -w "$f" ;;
esac
exit 0

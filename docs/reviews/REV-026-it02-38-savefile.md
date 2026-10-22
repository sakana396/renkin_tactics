# REV-026: セーブのファイル（internal/savefile）のレビュー

- 対象: PR #51、ブランチ `it02/38-savefile`（61cdaa8 と対応のコミット）。`internal/savefile` の新規作成（`dir.go` の `ResolveDir`・`DirEnv`、`store.go` の `Store`・`Path`・`Exists`・`Read`・`Write`）とテスト、テスト仕様書（反復 2）とトレーサビリティ。Issue #38（設計書 v0.3 の 9 章の Issue 4。DES-PKG-08、DES-SAV-02、10.7 節）
- 日付: 2026-10-22
- 作成: Codex（管理役が `codex exec` で頼んだ）
- レビュー: spec-reviewer（仕様適合レビュー。#48〜#51 をまとめてレビューした）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: F-01 に対応した（管理役が直した）。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd/renkin を除く）と `GOOS=windows go vet ./...` は成功
- `go test ./internal/savefile` ほか（core・tactics・defload・tccheck）は成功。CI は成功
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- 6.2 節を照らした: `Path`・`Exists`・`Read`（ないときは `fs.ErrNotExist` を包む）・`Write` の手順 1〜6（`MkdirAll` 0o755、同じディレクトリの `save-*.tmp`、Sync、Close、Rename、失敗したら一時ファイルを消して `save.json` を残す）、残った `save-*.tmp` を次の `Write` で片付ける、`Read` は一時ファイルを読まない
- 10.7 節の置き場所の表（`-savedir`、Windows・macOS は `os.UserConfigDir()`、Linux とほかの Unix は `XDG_DATA_HOME` か `~/.local/share`、その下の `renkin_tactics`）と、決められないときの誤り
- 8.2 節の DES-SAV-02 のテスト（`fn` が途中で失敗したとき・一時ファイルが残っているときに、前の `save.json` が変わらず読める）は、TC-NFR-08-02・03 で確かめていた
- ADR-0002・DES-PKG-08: 依存は標準ライブラリだけ
- ミュータント検査: 実装担当は 3 個を試し、すべて検出した。レビューでも 2 個（片付けの対象を `*.tmp` に広げる、`fn` の誤りを無視する）を試し、どちらもテストが失敗した

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | トレーサビリティの IF-04 が「実装済み」になっている。起動からの呼び出しと `-savedir`（DES-CMD-02）は #40 の担当でまだない | 設計どおりへの修正 | 状態を「一部実装」にし、残りを #40 と書いた |
| I-01 | 前に残った一時ファイルを消せないと、`Write` 全体が失敗する | 情報 | 対応なし（設計の範囲内） |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | `ResolveDir(override, DirEnv)` で OS・環境変数・`UserConfigDir`・`UserHomeDir` を引数で受ける | 設計の範囲内（Issue の申し送りどおり） |
| 2 | 中の関数 `write` で、一時ファイルの作成と名前の変更を差し替えられるようにする | 設計の範囲内 |
| 3 | 同じ置き場所への `Write` は、呼ぶ側で直列にする | 設計の範囲内 |

## 申し送り

- #40（起動）: `ResolveDir(flag, DirEnv{OS: runtime.GOOS, XDGDataHome: os.Getenv("XDG_DATA_HOME"), UserConfigDir: os.UserConfigDir, UserHomeDir: os.UserHomeDir})` で置き場所を決め、`Store{Dir}` を作る。誤りなら標準エラーに書いて終了コード 1

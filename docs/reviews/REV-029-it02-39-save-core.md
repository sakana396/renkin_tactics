# REV-029: セーブの中身の読み書き（core）のレビュー

- 対象: PR #54、ブランチ `it02/39-save-core`（7a52284）。`internal/core/save.go` の新規作成（`Campaign.Save`・`LoadCampaign`、`ErrSaveVersion`・`ErrInvalidSave`）、`internal/core/save_test.go` の新規作成、`docs/04_test/test-spec-it02.md`、`docs/traceability.md` の担当の行（NFR-07、FR-SYS-01）。Issue #39（設計書 v0.3 の 9 章の Issue 5）
- 日付: 2026-10-22
- 作成: Codex（実装。管理役が `codex exec` で頼んだ）。コミットとプルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（core・defs・defload・savefile・tools）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui`・`cmd/renkin` は CI で確かめた（`gh pr checks 54` は pass）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-SAV-01 を照らした。キーの順、空白 2 つの字下げ、空の配列は `[]`、在庫は ID の順、クリア済みなら `chapter` は空。TC-FR-SYS-01-01 の期待値は 6.1 節の例と同じ値で、設計から導かれている
- DES-SAV-03 を照らした。先に `version` だけを読み、版の誤りを知らないキーより先に返す。そのあと規則 1 → 3 → 4 → 5（名簿全体）→ 6（名簿全体）→ 7 の順。誤りの文に位置（`roster[0].level` など）が入る。JSON の後ろの値も拒む
- REV-028 の申し送りどおり、`Rule`・`Chapter`・`Cleared`・`Roster`・`Stock` を読み書きし、`Sortie` は持たない。`io.Reader`/`io.Writer` で読み書きし、ファイルの置き換えは `internal/savefile` に任せる。読み込みで加入をやり直さない
- core の約束（Ebitengine・実時間・乱数を使わない、失敗したら nil を返して状態を変えない）と、依存の向き（`defs`・`tactics` だけ）を確かめた
- 9 章の「書き換える反復 1 のテスト」に Issue 5 の行はなく、反復 1 のテストは変えていない
- 実装者は、ミュータント 8 種がすべてテストで失敗することを確かめた（在庫の並べ替えは、テストを補強してから失敗するようになった）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `encoding/json` がキーの大文字と小文字を区別しないため、`"RULE"` のような形式にないキーを受け入れる。重なったキーは後の値になる（DES-SAV-03 規則 1） | 設計どおりへの修正 | 今回は見送る（管理役の判断）。読み込める状態は正しく、害は小さい。反復 2 の振り返りで、直すかどうかを決める |
| I-01 | TC-NFR-07-07 は名簿 1 人の例だけで、規則 5 を名簿全体で確かめてから規則 6 に進む順を確かめていない | 情報 | 対応なし |
| I-02 | 手持ちの同じ ID の重なりは読み込める。規則 7 が重なりを禁じているのは在庫だけで、設計どおり | 情報 | 対応なし |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | `Save` は、章の添字が範囲外なら `ErrInvalidSave`、規則が不正なら `ErrInvalidRule` を返す | 設計の範囲内（DES-CMP-08） |
| 2 | 短い書き込みは `io.ErrShortWrite`。読み込みの I/O の失敗は `ErrInvalidSave` で包み、元の誤りも `errors.Is` で見分けられる | 設計の範囲内（NFR-07） |
| 3 | 読み込むときも在庫を ID の順にする | 設計の範囲内（DES-CMP-01、DES-SAV-01） |

## 申し送り

- #40（セーブの画面とのつなぎ）: `Campaign.Save` と `LoadCampaign` を、`savefile.Store` の `Write`・`Read` に渡す。章に入るたびに、`Save` の中身をメモリにも持つ（DES-CMP-07）。`ErrSaveVersion`・`ErrInvalidSave` の文をタイトル画面に出す（DES-UI-10）

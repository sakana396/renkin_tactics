# REV-008: 大文字・小文字だけが違うキーの扱いのレビュー

- 対象: PR #18、ブランチ `it01/17-key-case`（8fa8af1。`internal/defload/load.go` の `unknownKeys`・`walkUnknown` の修正、`internal/defload/load_test.go` への `TestLoadKeyCase` の追加、`docs/04_test/test-spec-it01.md` への 1 件の追記、`docs/traceability.md` の FR-SYS-04 の行の更新）。Issue #17（REV-006 N-01）
- 日付: 2026-10-21
- 作成: Claude Code（実装用サブエージェント。Codex の枠が埋まっているため、開発者の指示で Claude Code が実装した）
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR を変更していない。実装の判断 1 は設計の範囲内と判定した）
- 結果: 直す必要のある指摘はない。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet ./...`、`GOOS=windows go vet ./...` は成功
- `go test ./...` は成功。`go run ./tools/tccheck` は整合 OK（警告 0 件）。CI（`gh pr checks 18`）も成功
- `load.go` だけを修正前に戻すと、TC-FR-SYS-04-21 の 2 つのサブテストが落ちる（`terrains[0].Heal_Percent` の誤検出、`characters[0].Bonus` を表のまま報告する）ことを確かめた
- BurntSushi/toml v1.4.0 の `unifyStruct` と照らした。キーとタグの対応のさせ方（完全に一致するものを先に、なければ最初に `EqualFold` で合うもの）が `walkUnknown` と同じである。読み込み用の構造体の欄にはすべてタグが付いていて、オプションもない
- インラインの表の配列（`Items = [{ Item = …, countt = 2 }]`）、引用符付きのキー（`"a.b"`、`"X y"`）、`START_MAP`、`[[maps.Enemies]]` の中の `AII` で、誤検出も見逃しもないことを一時的なテストで確かめた

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| N-01 | 1 つの表に `count` と `Count` を両方書くと、どちらも toml が読んだ扱いになり、誤りにならない（どちらの値が使われるかは map を回す順で決まる）。toml の動作で、修正前から同じ | 情報 | 対応なし。大文字・小文字が違うだけのキーを誤りにするなら、規則 13 の意味が変わるので、設計の変更として開発者に確かめる |
| N-02 | 誤りに出る道筋は、ファイルに書いたとおりの綴りになる（`Bonus`）。意味の検証の誤りはタグの綴りになる（`items`）。引用符付きのキーは道筋で引用符なしになる。どれも既存の動作 | 情報 | 対応なし |

## 実装の判断の判定

| # | 判断 | 判定 |
|---|---|---|
| 1 | `walkUnknown` でも、キーとタグを toml と同じ順で対応させ、大文字・小文字が違う表の中にある知らないキーを、添字つきの道筋（`characters[0].Bonus.hpp`）で示す | 設計の範囲内（規則 13 の意味を変えず、`Undecoded()` を正とする Issue #17 の指示を満たす） |

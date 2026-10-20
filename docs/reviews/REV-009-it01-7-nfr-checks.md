# REV-009: 非機能の検査（依存、網羅率、クロスビルド）のレビュー

- 対象: PR #19、ブランチ `it01/7-nfr-checks`（950d188 と対応のコミット。`internal/core/deps_test.go` の新規作成、`.github/workflows/ci.yml` への手順 3 つの追加、`docs/04_test/test-spec-it01.md` への 11 件の追記、`docs/traceability.md` の担当の行の更新）。Issue #7
- 日付: 2026-10-21
- 作成: Claude Code（実装用サブエージェント。Codex の枠が埋まっているため、開発者の指示で Claude Code が実装した）
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR を変更していない。実装の判断 1〜6 は設計の範囲内と判定した）
- 結果: F-01・F-02 に対応した。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet ./...`、`GOOS=windows go vet ./...` は成功
- `go test -count=1 ./...` は成功
- `go run ./tools/tccheck` は整合 OK（警告 4 件。CI と手で確かめる TC-NFR-03-01・02、TC-NFR-09-01・02）。CI（`gh pr checks 19`）も成功。CI のログで、網羅率の手順が「合計: 97.0%」を出して成功し、Windows と Linux のクロスビルドの手順も成功したことを確かめた
- 一時的なワークツリーで、`tactics` と `internal/core` に `time` の import を足すと TC-NFR-04-01 が失敗することを確かめた。`internal/core` の下に新しいパッケージを足して `time` を import させたときも、`-count=1` なら失敗する
- CI の網羅率の判定式（awk）に値を渡し、79.9 は失敗、80.0・80・100.0 は成功、空と `abc` は失敗になることを確かめた（TC-NFR-09-02 の結果）。CI の `run` は `bash -e` で動くので、`go test` が失敗すれば手順が止まり、`go tool cover` が失敗すれば値が空になって失敗する
- 足した 3 つの手順に `if: steps.changes.outputs.code == 'true'` があり、文書だけの変更では省かれ、tccheck は残ることを確かめた（PLAN-001 5 章、QA-01）
- #11（`it01/11-core-battle`）と合わせた状態を `git merge-tree` で作り、衝突しないことを確かめた。`go vet`・`go test` は成功、`internal/core/...` の網羅率の合計は 97.5%、tccheck は OK
- 合わせた状態の `go list` で、`internal/core` の import が `fmt`・`core/tactics`・`defs`、`tactics` の import が標準ライブラリ（`container/heap`・`errors`・`fmt`・`math/rand/v2`）だけであることを確かめた（DES-PKG-04・05、ADR-0002）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | TC-NFR-09-02 は「CI で確かめる」と書かれているが、CI の網羅率はいつも 80% 以上なので、失敗する側は CI で通らない。手もとで 1 回確かめた結果が PR の説明にしかない | 設計どおりへの修正 | 対象の列を手動（CI の判定式）に変え、判定式に値を渡す手順を書いた。結果はこの記録の検証の節に残した |
| F-02 | `go test` のキャッシュは、テストから起動した `go list` が読んだソースを追わないので、手もとでは `internal/core` の下に足したパッケージの違反が `(cached)` で誤って通る。CI は毎回まっさらな環境なので影響しない | 情報 | 手もとでは `go test -count=1` で実行すると、テスト仕様書と `deps_test.go` のコメントに書いた |
| F-03 | `go list` は動かしている環境の GOOS で数えるので、`_windows.go` などの import は Linux 上では調べない。今は該当するファイルがない | 情報 | 対応なし。そういうファイルができたら考える |
| F-04 | Ebitengine は `github.com/hajimehoshi/ebiten` で始まる直接の import として見ている（設計書 7 章のとおり）。`tactics` は標準ライブラリだけかを別に調べている | 情報 | 対応なし |
| F-05 | CI と手で確かめるテストケースは、tccheck で「テストコードにありません」の警告になる。TC-QA-02-01 の扱いに合い、仕様書の節に理由が書いてある | 情報 | 対応なし |

## 実装の判断の判定

| # | 判断 | 判定 |
|---|---|---|
| 1 | 依存の検査を `internal/core/deps_test.go`（`package core_test`）に置き、`go list -json ./...` で `internal/core` 以下を調べる | 設計の範囲内（DES-NFR-01。#11 の `package core` とぶつからない） |
| 2 | 判定を `importViolations` に分け、作った import の一覧で境界と異常を確かめる | 設計の範囲内 |
| 3 | 標準ライブラリかどうかを `go list std` の一覧で決める | 設計の範囲内（ADR-0002） |
| 4 | 網羅率の値が読めないときも失敗にする | 設計の範囲内（DES-NFR-03） |
| 5 | CI だけで確かめるテストケースを、対象の列の「（CI）」で表し、tccheck の警告は残す | 設計の範囲内（TC-QA-02-01）。TC-NFR-09-02 は F-01 のとおり直した |
| 6 | 足した CI の手順に `if: steps.changes.outputs.code == 'true'` を付ける | 設計の範囲内（PLAN-001 5 章、QA-01） |

# REV-027: 僧侶リナの本体のデータと、core の杖の変換のレビュー

- 対象: PR #52、ブランチ `it02/44-cleric-data`（ebd2a42。`data/base` の `classes.toml`（僧侶）・`weapons.toml`（癒やしの杖）・`characters.toml`（リナ）・`game.toml`（`party` にリナ）・`story.toml`（ch01_before の 4 段落目）、`internal/core/battle.go` の `convertWeapon`（`staff`・`Heal`・`MaxUses`）、`internal/core/cleric_test.go` の新規作成、反復 1・反復 2 のテストの人数と添字の書き換え、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`、`docs/traceability.md` の担当の行）。Issue #44（設計書 v0.3 の 9 章の Issue 10）
- 日付: 2026-10-22
- 作成: Codex（実装。管理役が `codex exec` で頼んだ）。コミットとプルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー。#52・#53 をまとめてレビューした）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない。Codex が設計書 9 章の表に書き足したテストの ID は、管理役が元に戻し、PR の説明に書いた）
- 結果: 直すべき指摘はない。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（core・defload・defs・tools）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・tccheck で成功。`internal/ui`・`cmd/renkin` は CI で確かめた（CI は成功）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- 僧侶（16/0/2/6/2/5、杖）、癒やしの杖（回復 10、回数 15、射程 1〜1、威力・命中なし）、リナ（Lv 1、成長率 55/0/40/60/30、回復薬 × 1、スキルなし）が 10.1〜10.3 節・3.2 節のとおりで、ch01_before の 4 段落目が 10.8 節の文と同じことを確かめた
- `convertWeapon` が `staff` → `tactics.Staff`、`heal` → `Heal`、`uses` → `MaxUses` に変換し、`Uses` は `tactics.New` が満タンにすること（DES-CORE-01 v0.3、E-33）を確かめた
- 9 章の「書き換える反復 1 のテスト」の Issue 10 の行と、REV-023 I-01 の申し送り（TC-FR-CMP-07-01、TC-FR-SYS-03-09、TC-FR-SYS-03-01）が、テストケース ID を変えずに直されている。TC-FR-CMP-09-06 は書き換えずに通った
- ミュータント検査: レビューで 2 個を試した。`MaxUses` を 15 に固定する変更はテストが失敗した。上限 15 で切る変更は失敗しなかった（I-01）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| I-01 | TC-FR-SYS-03-11 の 3 組はどれも回数 15 以下なので、`MaxUses` を 15 で切る変更は捕まらない。本体の値は 15 で実害はない | 情報 | 対応なし |
| I-02 | #53（Issue #37）と `internal/core/battle_test.go`・`test-spec-it01.md`・`traceability.md` が衝突する | 情報 | 9 章の取り決めどおり、後からマージする #53 が合わせる |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | defload のテストで、杖・僧侶をテストで足す形をやめ、本体の定義を置き換えて検証する | 設計の範囲内（テストケースの意味と ID は変わらない） |

## 申し送り

- #37: マージの前に #52 を取り込み、リナの `Level: 1`・`Growth`、敵の添字 5、TC-FR-UNT-01-01〜03・02-01・ITM-03-01、仕様書とトレーサビリティの FR-SYS-03 の行を合わせる（REV-028）
- #45（画面の回復）: 本体の僧侶と杖が入った。情報欄の「残り n/15」と絵を合わせる

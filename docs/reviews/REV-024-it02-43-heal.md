# REV-024: 回復の行動と予測（tactics）のレビュー

- 対象: PR #49、ブランチ `it02/43-heal`（6099028 と、main の取り込みのコミット。b4baaf6 の WIP を引き継いだ）。`internal/core/tactics` の `heal.go`（`validateHeal`・`healWithStaff`・`HealForecast`）の新規作成、`types.go`（`Staff`・`Weapon.Heal`・`Uses`・`MaxUses`・`Action` の `Heal`）、`battle.go`（`newUnit` で満タンにする、規則 8 の `validateStaff`）、`attack.go`（`canCounter`・`ErrCannotAttack`）、`act.go`、`errors.go`、`event.go`、`heal_test.go` の新規作成、反復 1 のテストの書き換え、テスト仕様書とトレーサビリティ。Issue #43（設計書 v0.3 の 9 章の Issue 9）
- 日付: 2026-10-22
- 作成: Claude Code（実装用サブエージェント。途中まで）、Codex（残り。管理役が `codex exec` で頼んだ）
- レビュー: spec-reviewer（仕様適合レビュー。#48〜#51 をまとめてレビューした）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: F-01 に対応した（管理役が直した）。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd/renkin を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。CI は成功
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-HEL-01 を照らした: 定数を後ろに足して反復 1 の値を変えない、規則 8（敵・増援の杖、杖の `Heal`・`MaxUses` が 1 以上、杖でない武器は 3 つとも 0、`Uses` は見ない）、`New` で満タンにする、検証 6a〜6e の順、攻撃の検証の `ErrCannotAttack` を `ErrNoTarget` より前に、`canCounter` は杖なら偽、相性は杖に 0、出来事の順（Move→Heal→Exp→LevelUp）、`Heal` の写しでは使用回数が減っている、回復される側は経験を得ない
- 期待値の出どころ: 計算例 7（乱数 (10, 0, 50, 59, 40) → Lv 2・経験 5、HP・速さ ＋1）、使用回数の計算例 1〜4（14/15、16 回目に ErrNoUses、15/15 で始まる、150 → Lv 2・経験 50）、4.5 節の予測（7→19、10→15）を計算し直し、テストと一致した
- 9 章の「書き換える反復 1 のテスト」の Issue 9 の行（`Action(4)`・`Staff + 1`、TC-NFR-07-05 に誤り 3 つ）を直していた
- 範囲: 本体のデータ（#44）と画面（#45）は先取りしていない
- ミュータント検査: 実装担当は 6 個を試し、すべて検出した。レビューでも 2 個（`HealForecast` で共通の検証を省く、回復してから使用回数を減らす）を試し、どちらもテストが失敗した
- main（#48・#50・#51）の取り込みで、トレーサビリティの FR-BTL-08 と FR-UNT-02〜04 の行が衝突した。管理役が両方の内容を残して解いた（FR-BTL-08 は #43・#46 を両方書き、TC-FR-BTL-08-17〜21、状態は「一部実装」）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `test-spec-it02.md` の Issue 案 9 の見出しを消したため、回復の 16 件が Issue 案 8（FR-UNT-09）の節に入って見える | 設計どおりへの修正 | 見出しを戻し、説明文を見出しの下に置いた |
| I-01 | テスト仕様書にミュータント検査の結果の表がある（テストケースではない） | 情報 | 対応なし |

## 設計への提案の判定

なし（提案はない）。

## 申し送り

- #44（core の杖の変換）: `staff` → `tactics.Staff`、`uses` → `MaxUses`
- #45（画面の回復）: `HealForecast` と、`ui` の武器の種類の表で `Staff` を扱う

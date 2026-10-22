# REV-028: キャンペーンの状態と流れ（core）のレビュー

- 対象: PR #53、ブランチ `it02/37-campaign`（af6190b と、main（#52）の取り込みのコミット）。`internal/core/campaign.go` の新規作成（`Campaign`・`Member`・`Sortie`・`ChapterResult`・`MemberResult`、`NewCampaign`・`TakeItem`・`StoreItem`・`StartBattle`・`FinishChapter`、誤り 9 つ）、`internal/core/battle.go` の `setupWithAllies`・自軍の `Level`・`Growth`・敵の名前・`Item.ID`、`internal/core/campaign_test.go` の新規作成、`battle_test.go` の書き換え、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`、`docs/traceability.md` の担当の行）。Issue #37（設計書 v0.3 の 9 章の Issue 3）
- 日付: 2026-10-22
- 作成: Codex（実装、衝突の解消、F-01 の直し。管理役が `codex exec` で頼んだ）。コミットとプルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー。#52・#53 をまとめてレビューした）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: F-01 に対応し、#52 を取り込んだ。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（core・defload・defs・tools）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui`・`cmd/renkin` は CI で確かめた
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-CMP-03〜08 を照らした。新規開始（`party` の順、在庫は ID の順、加入）、入れ替えの検証の順と上限 5（数の合計）、出撃の検証の順（クリア済み → 数 → 添字・生死・重なり → 先頭が主人公）と `Campaign` を変えないこと、`FinishChapter` の検証の順（`ErrInvalidSortie` → `ErrNotWon`）、写しへの書き戻し（加入の誤りでも状態を変えない。NFR-07）、倒れる前の成長を残すこと、ロストの戦死と手持ちの消失、離脱の復帰（HP 最大）、報酬、クリア（`Chapter` は章の数）
- 敵の名前（名前のある敵は番号の数から除く）、敵・増援の `Level` 0、`Item.ID`（REV-022 の申し送り）を確かめた。9 章の「書き換える反復 1 のテスト」の Issue 3 の行は ID を変えずに直されている
- TC-FR-CMP-10-01 の期待値（経験 80 ＋ 30 → Lv 2・経験 10、成長率 100 で各 ＋1）は DES-GRW-01・02 から出ている
- ミュータント検査: レビューで 2 個を試した。`FinishChapter` で負けを通す変更はテストが失敗した。出撃の枠の上限の確認を外す変更はテストが失敗しなかった（F-01）。対応のあと、Codex が同じ変更でテストが失敗することを確かめた
- #52 の取り込みで衝突した `battle_test.go`・`test-spec-it01.md`・`traceability.md` は、レビューの指示（リナの `Level`・`Growth`、敵の添字 5、ハーゲンと「ソルジャー 3」、FR-SYS-03 の行を #44・#37 で TC-FR-SYS-03-01〜11）どおりに解いた。管理役が取り込んだ状態で検査が通ることを確かめた

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | TC-FR-CMP-03-03 の枠超過の例 `{0,1,2,3}` は、名簿の 3 番目を戦死にしているため、枠の上限の確認を外しても戦死の検証で `ErrInvalidDeployment` が返り、テストが通る（DES-CMP-05 の検証 2、UC-11 の 2） | 設計どおりへの修正 | 枠超過の例を全員生存にし、戦死の例と分けた。上限の確認を外すとテストが失敗することを確かめた |
| I-01 | `StartBattle` のあとに `TakeItem`・`StoreItem` を呼んでから `FinishChapter` すると、手持ちが戦闘の値に上書きされ、アイテムが消えるか増える。設計はこの順番を決めていない | 情報 | 対応なし。#41 に申し送る |
| I-02 | テスト名が ID だけのもの（TC-FR-CMP-04-01 など）がある | 情報 | 対応なし |
| I-03 | TC-FR-CMP-08-02 は「戦死者を再加入させない」を、生きているキャラで確かめている（名簿にいるかで判定するので動きは同じ） | 情報 | 対応なし |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | `setupWithAllies` を切り出し、`NewBattle` と `StartBattle` で共有する | 設計の範囲内（5.5 節） |
| 2 | `Sortie` が、出撃時点の名簿の写しと `members` の写しを内部に持つ | 設計の範囲内（5.5 節） |
| 3 | `FinishChapter` を写しに書き戻してから置き換える | 設計の範囲内（NFR-07、DES-CMP-08） |

## 申し送り

- #39（セーブの中身）: `Campaign` の `Rule`・`Chapter`・`Cleared`・`Roster`・`Stock` を読み書きし、`Sortie` は持たない
- #41（章の準備の画面）: 出撃したら準備に戻らない（I-01）。`ErrHandFull` などの理由を表示する（UC-11 の 4a）

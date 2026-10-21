# トレーサビリティマトリクス

- 文書 ID: TRC-001
- 対象: 開発の道具と反復 1。反復ごとに行を追加する

要求から設計・テスト・実装までの対応を表す。実装の担当者（エージェントを含む）は、自分が担当した要求の行だけを更新する。

| 要求 | Issue | 設計 | テストケース | 実装 | 状態 |
|---|---|---|---|---|---|
| QA-02 | — | tools/tccheck | TC-QA-02-01〜04 | `tools/tccheck`、CI | 実装済み（ふしぎ錬金から引き継ぎ） |
| FR-BTL-01 | #6 | DES-BTL-01、DES-BTL-03、DES-BTL-04 | TC-FR-BTL-01-01〜12 | `internal/core/tactics` | 実装済み |
| FR-BTL-02 | #6 | DES-BTL-02 | TC-FR-BTL-02-01〜07 | `internal/core/tactics` | 実装済み（敵軍の番は敵が動かない骨組み。動き方は #10） |
| FR-BTL-03 | #6、#8、#9 | DES-BTL-04 | TC-FR-BTL-03-01〜07 | `internal/core/tactics` | 実装済み |
| FR-BTL-04 | #8 | DES-BTL-05、DES-RNG-01 | TC-FR-BTL-04-01〜06 | `internal/core/tactics` | 実装済み |
| FR-BTL-05 | #8 | DES-BTL-05 | TC-FR-BTL-05-01〜04 | `internal/core/tactics` | 実装済み |
| FR-BTL-06 | #8 | DES-BTL-06 | TC-FR-BTL-06-01〜06 | `internal/core/tactics` | 実装済み |
| FR-BTL-07 | #10 | DES-AI-01（反復 1 は突っ込む・待ち構えるの 2 種類。一部） | TC-FR-BTL-07-01〜18 | `internal/core/tactics` | 一部実装（反復 1 の 2 種類を実装済み） |
| FR-BTL-08 | #8、#13 | DES-BTL-07、DES-UI-01 | TC-FR-BTL-08-01〜07、TC-FR-BTL-08-08（手動） | `internal/core/tactics`、`internal/ui` | 実装済み（#8、#13。手動のケースは開発者が確かめる） |
| FR-UNT-01 | #11 | DES-DAT-01、DES-CORE-01 | TC-FR-UNT-01-01〜06 | `internal/core` | 実装済み |
| FR-UNT-02 | #11 | DES-DAT-01、DES-CORE-01 | TC-FR-UNT-02-01 | `internal/core` | 実装済み |
| FR-UNT-03 | — | （H-01 で確かめる。振り返り。検証は D） | — | — | 振り返りで確かめる |
| FR-UNT-07 | #8 | DES-SKL-01（反復 1 はキャラのスキルだけ。一部） | TC-FR-UNT-07-01〜07 | `internal/core/tactics` | 実装済み |
| FR-UNT-09 | #13、#27 | DES-UI-01、DES-UI-06 | TC-FR-UNT-09-01〜05、TC-FR-UNT-09-07〜12、TC-FR-UNT-09-06・TC-FR-UNT-09-13（手動） | `internal/ui` | 実装済み（#13、#27。手動のケースは開発者が確かめる） |
| FR-MAP-01 | #6 | DES-BTL-03、DES-MAP-01 | TC-FR-MAP-01-01〜05 | `internal/core/tactics` | 実装済み |
| FR-MAP-03 | #9 | DES-MAP-02、DES-MAP-03 | TC-FR-MAP-03-01〜07 | `internal/core/tactics` | 実装済み（反復 1 のギミックは砦と増援） |
| FR-MAP-05 | #9 | DES-MAP-04（反復 1 はボスを倒す・主人公が倒れるだけ。一部） | TC-FR-MAP-05-01〜04 | `internal/core/tactics` | 実装済み（反復 1 の範囲） |
| FR-CMP-09 | #9、#12 | DES-MAP-04 | TC-FR-CMP-09-01〜06 | `internal/core/tactics`（06 は `internal/core` の結合のテスト） | 実装済み |
| FR-ITM-01 | #8 | DES-BTL-05、DES-ITM-01 | TC-FR-ITM-01-01〜02 | `internal/core/tactics` | 実装済み |
| FR-ITM-02 | #9 | DES-ITM-01 | TC-FR-ITM-02-01〜03 | `internal/core/tactics` | 実装済み |
| FR-ITM-03 | #9、#11 | DES-ITM-01、DES-DAT-01、DES-CORE-01（反復 1 は初めから持つだけ。一部） | TC-FR-ITM-03-01〜02 | `internal/core`、`internal/core/tactics` | 実装済み（#9、#11） |
| FR-SYS-03 | #5、#11 | DES-DAT-01、DES-DAT-02、DES-CORE-01 | TC-FR-SYS-03-01〜05 | `internal/defs`、`internal/defload`、`data`、`internal/core` | 実装済み |
| FR-SYS-04 | #5、#12 | DES-DAT-03、DES-CMD-01 | TC-FR-SYS-04-01〜23 | `internal/defload`、`cmd/renkin` | 実装済み（#5、#12） |
| IF-01 | #12 | DES-UI-02（DES-UI-01、03〜05、DES-CMD-01 を含む） | TC-IF-01-01〜17、TC-IF-01-21〜30（手動） | `internal/ui`、`internal/ui/assets`、`cmd/renkin` | 実装済み（手動のケースは開発者が確かめる） |
| IF-02 | #12、#13 | DES-UI-01 | TC-IF-02-01〜02、TC-IF-02-03（手動） | `internal/ui` | 実装済み（#12。情報欄は #13） |
| IF-03 | #5 | DES-DAT-02、ADR-0003 | TC-IF-03-01〜02 | `data`、`internal/defload` | 実装済み |
| NFR-03 | #7、#12 | DES-NFR-02 | TC-NFR-03-01〜02、TC-NFR-03-03〜04（手動） | `.github/workflows/ci.yml`、`cmd/renkin` | 一部実装（#7、#12。Windows 11 の手の確認 TC-NFR-03-03 は済み（REV-015）。Linux の手の確認 TC-NFR-03-04 は反復 1 では行わず、CI で代える（E-23）） |
| NFR-04 | #7 | DES-NFR-01 | TC-NFR-04-01〜04 | `internal/core`（`deps_test.go`） | 実装済み |
| NFR-05 | #7 | DES-NFR-01 | TC-NFR-05-01〜03 | `internal/core`（`deps_test.go`） | 実装済み |
| NFR-06 | #6 | DES-RNG-01 | TC-NFR-06-01〜03 | `internal/core/tactics` | 実装済み |
| NFR-07 | #6、#9 | DES-ERR-01、DES-BTL-08 | TC-NFR-07-01〜05 | `internal/core/tactics` | 実装済み（TC-NFR-07-04 は #9 で `UseItem` の検証 6 に書き換えた） |
| NFR-09 | #7 | DES-NFR-03 | TC-NFR-09-01〜02 | `.github/workflows/ci.yml` | 実装済み |

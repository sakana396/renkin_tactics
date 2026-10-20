# トレーサビリティマトリクス

- 文書 ID: TRC-001
- 対象: 開発の道具と反復 1。反復ごとに行を追加する

要求から設計・テスト・実装までの対応を表す。実装の担当者（エージェントを含む）は、自分が担当した要求の行だけを更新する。

| 要求 | Issue | 設計 | テストケース | 実装 | 状態 |
|---|---|---|---|---|---|
| QA-02 | — | tools/tccheck | TC-QA-02-01〜04 | `tools/tccheck`、CI | 実装済み（ふしぎ錬金から引き継ぎ） |
| FR-BTL-01 | #6 | DES-BTL-01、DES-BTL-03、DES-BTL-04 | TC-FR-BTL-01-01〜12 | `internal/core/tactics` | 実装済み |
| FR-BTL-02 | #6 | DES-BTL-02 | TC-FR-BTL-02-01〜07 | `internal/core/tactics` | 実装済み（敵軍の番は敵が動かない骨組み。動き方は #10） |
| FR-BTL-03 | #6、#8、#9 | DES-BTL-04 | TC-FR-BTL-03-01〜06 | `internal/core/tactics` | 一部実装（#6、#8） |
| FR-BTL-04 | #8 | DES-BTL-05、DES-RNG-01 | TC-FR-BTL-04-01〜06 | `internal/core/tactics` | 実装済み |
| FR-BTL-05 | #8 | DES-BTL-05 | TC-FR-BTL-05-01〜04 | `internal/core/tactics` | 実装済み |
| FR-BTL-06 | #8 | DES-BTL-06 | TC-FR-BTL-06-01〜06 | `internal/core/tactics` | 実装済み |
| FR-BTL-07 | #10 | DES-AI-01（反復 1 は突っ込む・待ち構えるの 2 種類。一部） | — | — | 未着手 |
| FR-BTL-08 | #8、#13 | DES-BTL-07、DES-UI-01 | TC-FR-BTL-08-01〜03 | `internal/core/tactics` | 一部実装（#8） |
| FR-UNT-01 | #11 | DES-DAT-01、DES-CORE-01 | — | — | 未着手 |
| FR-UNT-02 | #11 | DES-DAT-01、DES-CORE-01 | — | — | 未着手 |
| FR-UNT-03 | — | （H-01 で確かめる。振り返り。検証は D） | — | — | 振り返りで確かめる |
| FR-UNT-07 | #8 | DES-SKL-01（反復 1 はキャラのスキルだけ。一部） | TC-FR-UNT-07-01〜07 | `internal/core/tactics` | 実装済み |
| FR-UNT-09 | #13 | DES-UI-01 | — | — | 未着手 |
| FR-MAP-01 | #6 | DES-BTL-03、DES-MAP-01 | TC-FR-MAP-01-01〜05 | `internal/core/tactics` | 実装済み |
| FR-MAP-03 | #9 | DES-MAP-02、DES-MAP-03 | — | — | 未着手 |
| FR-MAP-05 | #9 | DES-MAP-04（反復 1 はボスを倒す・主人公が倒れるだけ。一部） | — | — | 未着手 |
| FR-CMP-09 | #9 | DES-MAP-04 | — | — | 未着手 |
| FR-ITM-01 | #8 | DES-BTL-05、DES-ITM-01 | TC-FR-ITM-01-01〜02 | `internal/core/tactics` | 実装済み |
| FR-ITM-02 | #9 | DES-ITM-01 | — | — | 未着手 |
| FR-ITM-03 | #9、#11 | DES-ITM-01、DES-DAT-01、DES-CORE-01（反復 1 は初めから持つだけ。一部） | — | — | 未着手 |
| FR-SYS-03 | #5、#11 | DES-DAT-01、DES-DAT-02、DES-CORE-01 | TC-FR-SYS-03-01〜02 | `internal/defs`、`internal/defload`、`data` | 一部実装（#5） |
| FR-SYS-04 | #5、#12 | DES-DAT-03、DES-CMD-01 | TC-FR-SYS-04-01〜20 | `internal/defload` | 一部実装（#5） |
| IF-01 | #12 | DES-UI-02 | — | — | 未着手 |
| IF-02 | #12 | DES-UI-01 | — | — | 未着手 |
| IF-03 | #5 | DES-DAT-02、ADR-0003 | TC-IF-03-01〜02 | `data`、`internal/defload` | 実装済み |
| NFR-03 | #7 | DES-NFR-02 | — | — | 未着手 |
| NFR-04 | #7 | DES-NFR-01 | — | — | 未着手 |
| NFR-05 | #7 | DES-NFR-01 | — | — | 未着手 |
| NFR-06 | #6 | DES-RNG-01 | TC-NFR-06-01〜03 | `internal/core/tactics` | 実装済み |
| NFR-07 | #6 | DES-ERR-01、DES-BTL-08 | TC-NFR-07-01〜05 | `internal/core/tactics` | 実装済み |
| NFR-09 | #7 | DES-NFR-03 | — | — | 未着手 |

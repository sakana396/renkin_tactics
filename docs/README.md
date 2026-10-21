# 文書一覧

本プロジェクトの成果物とその置き場所。工程が進むたびにこの表を更新する。開発は反復型で進める（[PLAN-001](00_plan/project-plan.md) 3 章）。

| 工程 | 成果物 | 置き場所 | 状態 |
|---|---|---|---|
| 計画 | プロジェクト計画書 | [00_plan/project-plan.md](00_plan/project-plan.md) | v0.1 承認済み |
| 企画 | 要求抽出記録（ヒアリング） | [01_concept/elicitation-notes.md](01_concept/elicitation-notes.md) | 追記中 |
| 企画 | 企画書 | [01_concept/concept.md](01_concept/concept.md) | v0.3 承認済み |
| 企画 | 技術選定 ADR | [adr/0001-tech-stack.md](adr/0001-tech-stack.md) | 承認済み |
| 設計 | ふしぎ錬金に戻す形 ADR | [adr/0002-return-to-fushigi.md](adr/0002-return-to-fushigi.md) | 承認済み |
| 設計 | 定義データの形式 ADR | [adr/0003-definition-data-format.md](adr/0003-definition-data-format.md) | 承認済み |
| 要求 | 要求仕様書 (SRS) | [02_requirements/srs.md](02_requirements/srs.md) | v0.1 承認済み |
| 要求 | ユースケース | [02_requirements/use-cases.md](02_requirements/use-cases.md) | v0.3（案。反復 2 の範囲を書き足し、開発者の確認待ち） |
| 計画 | 反復 1 計画書 | [iterations/it01/plan.md](iterations/it01/plan.md) | v0.2 承認済み |
| 設計 | 設計書（反復 1） | [03_design/design-it01.md](03_design/design-it01.md) | v0.3 承認済み |
| 計画 | 反復 2 計画書 | [iterations/it02/plan.md](iterations/it02/plan.md) | v0.1（案。開発者の確認待ち） |
| 設計 | 設計書（反復 2） | [03_design/design-it02.md](03_design/design-it02.md) | v0.1（案。開発者の確認待ち） |
| 試験 | テスト仕様書（反復 1） | [04_test/test-spec-it01.md](04_test/test-spec-it01.md) | 実装とともに書き足す |
| 試験 | テスト仕様書（開発の道具） | [04_test/test-spec-tools.md](04_test/test-spec-tools.md) | ふしぎ錬金から引き継ぎ |
| 横断 | トレーサビリティマトリクス | [traceability.md](traceability.md) | 開発の道具と反復 1 の行 |
| 横断 | レビュー記録 | [reviews/](reviews/) | 記録中 |

## エージェント向けのルール

コーディングエージェントの共通ルールはリポジトリ直下の [AGENTS.md](../AGENTS.md) にある。

## ID 規約

トレーサビリティのため、要素には次の ID を振る。ふしぎ錬金と同じ規約を使う。

| 接頭辞 | 対象 | 例 |
|---|---|---|
| `FR-<区分>-` | 機能要求（区分の一覧は [SRS](02_requirements/srs.md) 3 章。例: BTL 戦闘、UNT ユニット、MAP マップ） | `FR-BTL-01` |
| `NFR-` | 非機能要求 | `NFR-06` |
| `IF-` | 外部インタフェース要求 | `IF-03` |
| `CST-` | 制約 | `CST-02` |
| `QA-` | 品質保証の要求 | `QA-02` |
| `TBD-` | 要求仕様書の未決事項 | `TBD-03` |
| `UC-` | ユースケース | `UC-02` |
| `DES-` | 設計要素 | `DES-BTL-01` |
| `TC-` | テストケース | `TC-FR-BTL-01-01` |
| `ADR-` | 設計判断記録 | `ADR-0001` |
| `E-` | 要求抽出記録の発言 | `E-04` |
| `Q-` | 企画書の未決事項 | `Q-01` |
| `REV-` | レビュー記録 | `REV-001` |

## 図

UML は [Mermaid](https://mermaid.js.org/) で Markdown に直接書く（GitHub 上でそのまま描画され、差分も追える）。

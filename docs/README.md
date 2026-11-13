# 文書一覧

本プロジェクトの成果物とその置き場所。工程が進むたびにこの表を更新する。開発は反復型で進める（[PLAN-001](00_plan/project-plan.md) 3 章）。

| 工程 | 成果物 | 置き場所 | 状態 |
|---|---|---|---|
| 計画 | プロジェクト計画書 | [00_plan/project-plan.md](00_plan/project-plan.md) | v0.1 承認済み |
| 企画 | 要求抽出記録（ヒアリング） | [01_concept/elicitation-notes.md](01_concept/elicitation-notes.md) | 追記中 |
| 企画 | 企画書 | [01_concept/concept.md](01_concept/concept.md) | v0.6 承認（E-94、#119）。錬金術師・魔道士、仲間 9 人、出撃枠 +1 |
| 企画 | 新しいゲームの企画たたき台（反復外） | [01_concept/new-game-draft.md](01_concept/new-game-draft.md) | v0.1 未承認。#191 の未決事項をエージェントが仮決めした案。既存の企画・要求は変えない |
| 企画 | 新しいゲームの見た目と音の方針（反復外） | [01_concept/new-game-visual-draft.md](01_concept/new-game-visual-draft.md) | v0.3 未承認。AI 生成画像の指示書（絵柄・配色・キャラの外見・指示文の型・画像の一覧・出典の記録）。既存の企画・要求は変えない |
| 企画 | 新作のキャラの画像の指示文（組み立て済み） | [01_concept/new-game-prompts.md](01_concept/new-game-prompts.md) | v0.1 未承認。自軍 4 人（ユニット・肖像）と敵 6 種の ChatGPT 用の指示文 |
| 企画 | 技術選定 ADR | [adr/0001-tech-stack.md](adr/0001-tech-stack.md) | 承認済み |
| 設計 | ふしぎ錬金に戻す形 ADR | [adr/0002-return-to-fushigi.md](adr/0002-return-to-fushigi.md) | 承認済み |
| 設計 | 定義データの形式 ADR | [adr/0003-definition-data-format.md](adr/0003-definition-data-format.md) | 承認済み |
| 要求 | 要求仕様書 (SRS) | [02_requirements/srs.md](02_requirements/srs.md) | v0.8 承認（E-122・E-123。DES-IT07 7 章の文面と補足、#164）。ZOC・小隊長・画面の操作を追記 |
| 要求 | ユースケース | [02_requirements/use-cases.md](02_requirements/use-cases.md) | v0.13（UC-02 の直接クリックの取消し、E-124）。v0.12 承認（E-122・E-123。DES-IT07 7 章の文面と補足、#164）。予測中の武器・予測窓・詳細画面・範囲・持ち物・ZOC・小隊長を追記 |
| 計画 | 反復 1 計画書 | [iterations/it01/plan.md](iterations/it01/plan.md) | v0.2 承認済み |
| 設計 | 設計書（反復 1） | [03_design/design-it01.md](03_design/design-it01.md) | v0.5 承認（E-122・E-123。DES-IT07 7 章の文面と補足、#164）。DES-BTL-03・AI-01・UI-07・08 の追記。v0.4 承認済み（E-40） |
| 計画 | 反復 2 計画書 | [iterations/it02/plan.md](iterations/it02/plan.md) | v0.3 承認済み（E-32、E-33） |
| 設計 | 設計書（反復 2） | [03_design/design-it02.md](03_design/design-it02.md) | v0.4 承認済み（E-32、E-33、E-40） |
| 試験 | テスト仕様書（反復 1） | [04_test/test-spec-it01.md](04_test/test-spec-it01.md) | 実装とともに書き足す |
| 試験 | テスト仕様書（反復 2） | [04_test/test-spec-it02.md](04_test/test-spec-it02.md) | 骨組み。実装とともに書き足す |
| 試験 | テスト仕様書（反復 3） | [04_test/test-spec-it03.md](04_test/test-spec-it03.md) | 骨組み。実装とともに書き足す |
| 試験 | テスト仕様書（反復 4） | [04_test/test-spec-it04.md](04_test/test-spec-it04.md) | v0.1 骨組み。自動は未実装、手動は未実施 |
| 試験 | テスト仕様書（反復 5） | [04_test/test-spec-it05.md](04_test/test-spec-it05.md) | v0.2。自動は実装済み。#98・#105 の手動結果と持ち越しは 3.10 節（E-60〜E-72） |
| 試験 | テスト仕様書（反復 6） | [04_test/test-spec-it06.md](04_test/test-spec-it06.md) | v2.13（#120・#123・#124・#126・#127・#130・#149・#131・#132・#133・#154・#158）。能力・計算・予測・セーブ、回復と敵のレベル、出来事、再移動、一団の敵の思考の自動ケースを順次実装。旧自動試遊の一部は IT06 で一時停止（#130〜#133 で置き換え）。#154 で初期突撃・待つ守りの一団・一団の近さ・連続増援と 3 種の名簿の 20 seed の自動試遊を足した。#172・#173 の固定 seed を反映。後続の自動ケース・手動・基準の PC での測定は未実施 |
| 試験 | テスト仕様書（反復 7） | [04_test/test-spec-it07.md](04_test/test-spec-it07.md) | v1.3（#188）。ZOCの経路計測と増援の最終経路・全試行の区別を追加し、新版920試遊を測り直し。v1.2（#189）は射程とフレーム内範囲共有の回帰を追加し、同じ基準PCの合成条件で最低79.808fps。人の試遊等は後続 |
| 試験 | テスト仕様書（開発の道具） | [04_test/test-spec-tools.md](04_test/test-spec-tools.md) | ふしぎ錬金から引き継ぎ |
| 試験 | テスト仕様書（新作の試作 NG01: 錬金の爆弾） | [04_test/test-spec-ng01.md](04_test/test-spec-ng01.md) | v0.2（未承認）。自動は実装済み、画面の見た目は手動・未実施 |
| 試験 | テスト仕様書（新作の試作データ NG-06。TST-NG01 の 7 章） | [04_test/test-spec-ng01.md](04_test/test-spec-ng01.md) | v0.2（未承認）。データ・自動の試遊・`-game` は自動で実装済み。人の試遊（TC-DES-NG-06-10）は未実施 |
| 試験 | テスト仕様書（新作の試作 NG02: 画像の差し込み） | [04_test/test-spec-ng02.md](04_test/test-spec-ng02.md) | v0.1（未承認）。自動は実装済み、実画面の確認（TC-DES-NG-08-94）は手動・未実施 |
| 試験 | テスト仕様書（新作の試作 NG03: 音） | [04_test/test-spec-ng03.md](04_test/test-spec-ng03.md) | v0.1（未承認）。自動は実装済み、耳での確認（TC-DES-NG-09-26）は手動・未実施 |
| 試験 | テスト仕様書（新作の試作 NG04: 高解像度の表示） | [04_test/test-spec-ng04.md](04_test/test-spec-ng04.md) | v0.3（未承認）。自動は実装済み（フェーズ 2: TC-DES-NG-10-18〜28、REV-109 対応: TC-DES-NG-10-29〜34 を追加）。窓での実操作は未確認 |
| 横断 | トレーサビリティマトリクス | [traceability.md](traceability.md) | 開発の道具と反復 1 の行 |
| 横断 | レビュー記録 | [reviews/](reviews/) | 記録中 |
| 計画 | 反復 1 の振り返り | [iterations/it01/retrospective.md](iterations/it01/retrospective.md) | v0.2 承認（E-37） |
| 計画 | 反復 2 の振り返り | [iterations/it02/retrospective.md](iterations/it02/retrospective.md) | v0.2 承認（E-37） |
| 計画 | 反復 3 計画書 | [iterations/it03/plan.md](iterations/it03/plan.md) | v0.2 承認（E-37） |
| 設計 | 設計書（反復 3） | [03_design/design-it03.md](03_design/design-it03.md) | v0.3 承認（E-37、E-40） |
| 計画 | 反復 3 の振り返り | [iterations/it03/retrospective.md](iterations/it03/retrospective.md) | v0.2 承認（E-42） |
| 計画 | 反復 4 計画書 | [iterations/it04/plan.md](iterations/it04/plan.md) | v0.4（承認（E-46）） |
| 設計 | 設計書（反復 4） | [03_design/design-it04.md](03_design/design-it04.md) | v0.4（承認（E-46）） |
| 計画 | 反復 4 の振り返り | [iterations/it04/retrospective.md](iterations/it04/retrospective.md) | v0.2 承認（E-48） |
| 計画 | 反復 5 計画書 | [iterations/it05/plan.md](iterations/it05/plan.md) | v0.4 承認（E-57）。仲間 2 人の加入（E-55） |
| 設計 | 設計書（反復 5） | [03_design/design-it05.md](03_design/design-it05.md) | v0.4 承認（E-57）。仲間 2 人の加入（E-55） |
| 計画 | 反復 5 の振り返り | [iterations/it05/retrospective.md](iterations/it05/retrospective.md) | v0.2 承認（E-72） |
| 計画 | 反復 6 計画書 | [iterations/it06/plan.md](iterations/it06/plan.md) | v0.4 承認（E-114）。v0.3 まで承認（E-87）。手ごたえの引き上げと、武器・魔法の広がり（E-73〜E-87）。v0.4 で試遊の行動の記録（#155、E-103・E-104・E-106）を追加 |
| 設計 | 設計書（反復 6） | [03_design/design-it06.md](03_design/design-it06.md) | v0.9 承認（E-122・E-123。DES-IT07 7 章の文面と補足、#164）。8・10・12・13 章に反復 7 の追記。v0.8（13.1 節の試遊の行動の記録（DES-LOG-01〜03、#155、0.8）、13 章「自動の試遊」の 2 つの打ち手（#158、0.7）、10・13 章の釣り出し対策と一団の待ち方（E-99・E-100・E-108・E-109、#154、0.6）、12.2 節の改訂（E-97、#149、0.5）は承認（E-114））。0.4 まで承認（E-93）。REV-066 の指摘を反映。未決事項は E-89 で決定、全章の出撃枠 +1（E-90）、一度に 46 体（自軍 7・敵 39。E-91）。検算は [03_design/scripts/verify_des06.py](03_design/scripts/verify_des06.py) |
| 設計 | 式の一覧（DES-FML-01） | [03_design/formulas.md](03_design/formulas.md) | v0.3 承認（E-122・E-123。DES-IT07 7 章の文面と補足、#164）。ZOC 込みの移動の費用と近さを追記。v0.2 承認（E-93）。REV-066 の指摘を反映。勇者の斧の反撃は 1 回（E-89） |
| 計画 | 反復 6 の振り返り（IT06-RETRO） | [iterations/it06/retrospective.md](iterations/it06/retrospective.md) | v0.2（承認（E-116））。通しの試遊（E-113）と行動の記録で H-22〜27 を判定し、知見（E-114）を記録 |
| 試験 | 反復7 全9章の測り直し | [iterations/it07/remeasure.md](iterations/it07/remeasure.md) | #174。旧新版各920回、9章の時機2案、通しの回帰と基準PCの性能。未達・未承認あり |
| 試験 | 反復7 描画性能の改善 | [iterations/it07/render-performance.md](iterations/it07/render-performance.md) | #189。同じ合成条件で12.228→79.808fps。実章初期も別途測定 |
| 計画 | 反復 7 計画書 | [iterations/it07/plan.md](iterations/it07/plan.md) | v0.3（承認（E-119））。画面の作り直し・指揮官・重騎士の ZOC（E-117・E-118）。REV-090 の指摘を反映 |
| 設計 | 設計書（反復 7） | [03_design/design-it07.md](03_design/design-it07.md) | v0.7（3.4・3.8 節に E-127 を反映）。v0.6（3.1・3.3 節に E-126（番を終える確認の間の M・Z）を反映）。v0.5（3.2・3.5 節に E-124 を反映）。v0.4 承認（E-122）。未決事項は E-120・E-121 で決定。REV-091 を反映。画面・指揮官・重騎士の ZOC |
| 設計 | 設計メモ（新作の試作 NG01: 錬金の爆弾） | [03_design/design-ng01.md](03_design/design-ng01.md) | v0.2（未承認）。企画たたき台 2.7 節の最小の試作。既存の設計は変えない |
| 設計 | 設計メモ（新作の試作データ NG-06。DES-NG01 の 6 章） | [03_design/design-ng01.md](03_design/design-ng01.md) | v0.2（未承認）。`data/alchemy`（1〜3 章）と `-game`。名前・文面は仮で、既存作品との重なりは未確認 |
| 設計 | 設計メモ（新作の試作 NG02: 画像の差し込み） | [03_design/design-ng02-art.md](03_design/design-ng02-art.md) | v0.2（未承認）。DES-NG-08。置き場と名前の規則・画像の大きさの上限・台帳と検査・試し用の絵の道具。`game.go` へ結び付け済み |
| 設計 | 設計メモ（新作の試作 NG03: 音） | [03_design/design-ng03-audio.md](03_design/design-ng03-audio.md) | v0.2（未承認）。DES-NG-09。効果音・BGM の再生、仮の音の合成（`tools/synth`）、台帳。ミュートのキーは M が全体図と重なるため N（設計への提案）。耳での確認は未実施 |
| 設計 | 設計メモ（新作の試作 NG04: 高解像度の表示） | [03_design/design-ng04-display.md](03_design/design-ng04-display.md) | v0.3（未承認）。DES-NG-10。`Config.HiRes` で 1 マス 48 px に切り替え（フェーズ 1）、コードで描いた仮の地形・AI 生成の地形の差し替え口・印と棒の比例（フェーズ 2）、REV-109 への対応（9 章: 敗北の中央寄せ・窓の大きさ） |
| 設計 | ch09「ヴァルトの門」の制作記録（反復 5） | [iterations/it05/map-ch09.md](iterations/it05/map-ch09.md) | v0.2（#102、#104 で E-52 の強化。開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |
| 制作 | ch04 の制作記録（反復 5） | [iterations/it05/map-ch04.md](iterations/it05/map-ch04.md) | v0.1（#103。自動試遊は実施、開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |
| 制作 | ch05 の制作記録（反復 5） | [iterations/it05/map-ch05.md](iterations/it05/map-ch05.md) | v0.1（#103。自動試遊は実施、開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |
| 制作 | ch02 の制作記録（反復 6） | [iterations/it06/map-ch02.md](iterations/it06/map-ch02.md) | v0.3（#130、REV-079、E-96・E-97。自動の下限は確認、開発者の短い試遊は 1 回目（版 0.2）を実施し、版 0.3 は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch05 の制作記録（反復 6） | [iterations/it06/map-ch05.md](iterations/it06/map-ch05.md) | v0.3（#130、REV-079、E-96・E-97。自動の下限は確認、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch01 の制作記録（反復 6） | [iterations/it06/map-ch01.md](iterations/it06/map-ch01.md) | v0.1（#131。自動の下限は確認、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch03 の制作記録（反復 6） | [iterations/it06/map-ch03.md](iterations/it06/map-ch03.md) | v0.2（#131。自動の下限は確認、壁の外の増援の候補を記録。#154 で 5T の増援の元を (5,1) に動かした（E-114）。開発者の試遊は E-113）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch04 の制作記録（反復 6） | [iterations/it06/map-ch04.md](iterations/it06/map-ch04.md) | v0.1（#131。自動の下限は確認、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch06 の制作記録（反復 6） | [iterations/it06/map-ch06.md](iterations/it06/map-ch06.md) | v0.2（#132。自動の下限は確認、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch07 の制作記録（反復 6） | [iterations/it06/map-ch07.md](iterations/it06/map-ch07.md) | v0.2（#132。自動の下限は確認、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch08 の制作記録（反復 6） | [iterations/it06/map-ch08.md](iterations/it06/map-ch08.md) | v0.2（#133。報酬武器を含む名簿で自動の下限を再測定、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch09 の制作記録（反復 6） | [iterations/it06/map-ch09.md](iterations/it06/map-ch09.md) | v0.2（#133。報酬武器を含む名簿で自動の下限を再測定、開発者の短い試遊は未実施）。#154 で釣り出し対策を追記（REV-084 で作り直した） |
| 制作 | ch01 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch01.md](iterations/it07/map-ch01.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch02 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch02.md](iterations/it07/map-ch02.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch03 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch03.md](iterations/it07/map-ch03.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch04 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch04.md](iterations/it07/map-ch04.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch05 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch05.md](iterations/it07/map-ch05.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch06 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch06.md](iterations/it07/map-ch06.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch07 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch07.md](iterations/it07/map-ch07.md) | v0.3（#172、REV-101。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch08 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch08.md](iterations/it07/map-ch08.md) | v0.2（#172。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch09 の小隊長の制作記録（反復 7） | [iterations/it07/map-ch09.md](iterations/it07/map-ch09.md) | v0.3（#172、REV-101。小隊長の配置・敵の数・自動の試し遊びの旧新と #173 の ZOC 後の測定。開発者の試遊は未実施） |
| 制作 | ch06 の制作記録（反復 5） | [iterations/it05/map-ch06.md](iterations/it05/map-ch06.md) | v0.1（#103。自動試遊は実施、開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |
| 制作 | ch07 の制作記録（反復 5） | [iterations/it05/map-ch07.md](iterations/it05/map-ch07.md) | v0.1（#104。自動試遊は実施、開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |
| 制作 | ch08 の制作記録（反復 5） | [iterations/it05/map-ch08.md](iterations/it05/map-ch08.md) | v0.1（#104。自動試遊は実施、開発者の試遊は #105 で実施（6 章、TST-IT05 3.10 節）） |

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

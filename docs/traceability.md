# トレーサビリティマトリクス

- 文書 ID: TRC-001
- 対象: 開発の道具と反復 1。反復ごとに行を追加する

要求から設計・テスト・実装までの対応を表す。実装の担当者（エージェントを含む）は、自分が担当した要求の行だけを更新する。

| 要求 | Issue | 設計 | テストケース | 実装 | 状態 |
|---|---|---|---|---|---|
| QA-02 | — | tools/tccheck | TC-QA-02-01〜04 | `tools/tccheck`、CI | 実装済み（ふしぎ錬金から引き継ぎ） |
| FR-BTL-01 | #6、#76 | DES-BTL-01、DES-BTL-03、DES-BTL-04、DES-RWD-05 | TC-FR-BTL-01-01〜12 | `internal/core/tactics` | 実装済み（#76: 移動・地形・射程の期待値を維持し、BoardView でも共通計算の回帰を確認） |
| FR-BTL-02 | #6 | DES-BTL-02 | TC-FR-BTL-02-01〜07 | `internal/core/tactics` | 実装済み（敵軍の番は敵が動かない骨組み。動き方は #10） |
| FR-BTL-03 | #6、#8、#9、#43、#45 | DES-BTL-04、DES-HEL-01、DES-UI-18 | TC-FR-BTL-03-01〜17 | `internal/core/tactics`、`internal/ui` | 実装済み |
| FR-BTL-04 | #8 | DES-BTL-05、DES-RNG-01 | TC-FR-BTL-04-01〜06 | `internal/core/tactics` | 実装済み |
| FR-BTL-05 | #8 | DES-BTL-05 | TC-FR-BTL-05-01〜04 | `internal/core/tactics` | 実装済み |
| FR-BTL-06 | #8、#43 | DES-BTL-06、DES-HEL-01 | TC-FR-BTL-06-01〜07 | `internal/core/tactics` | 実装済み |
| FR-BTL-07 | #10 | DES-AI-01（反復 1 は突っ込む・待ち構えるの 2 種類。一部） | TC-FR-BTL-07-01〜18 | `internal/core/tactics` | 一部実装（反復 1 の 2 種類を実装済み） |
| FR-BTL-08 | #8、#13、#29、#43、#45、#46、#66 | DES-BTL-07、DES-UI-01、DES-UI-07、DES-HEL-02、DES-UI-18 | TC-FR-BTL-08-01〜07、TC-FR-BTL-08-08（手動）、TC-FR-BTL-08-09〜15、TC-FR-BTL-08-16（手動）、TC-FR-BTL-08-17〜23、TC-FR-BTL-08-24（手動） | `internal/core/tactics`、`internal/ui` | 実装済み（#45 の回復の操作・予測、#46 の撃破・危険の強調を含む。#66: 回復表示の手動ケース TC-FR-BTL-08-24 を追加、今回は未確認（E-39）。既存の手動ケース全体の合格とはしない） |
| FR-UNT-01 | #11、#37 | DES-DAT-01、DES-CORE-01 | TC-FR-UNT-01-01〜06 | `internal/core` | 実装済み |
| FR-UNT-02 | #11、#36、#44 | DES-DAT-01、DES-CORE-01、DES-DAT-03 規則 4・21・22（反復 2）、設計書（反復 2）10.1 節 | TC-FR-UNT-02-01〜02（反復 2 の杖の定義は TC-FR-SYS-04-26〜28） | `internal/core`、`internal/defs`、`internal/defload`、`data` | 実装済み（#11、#36。騎兵と杖の定義データの型・検証。#44 で僧侶・杖・リナの本体のデータと杖の変換を実装） |
| FR-UNT-03 | #36 | （H-01、H-08 で確かめる。振り返り。検証は D）、設計書（反復 2）10.3・10.9 節 | TC-FR-UNT-03-01（数値の向きだけを確かめる） | `data` | 役割の数値を入れた（#36）。役に立つかは振り返りで確かめる |
| FR-UNT-04 | #35、#43、#37、#42 | DES-GRW-01、DES-GRW-02、DES-GRW-03、DES-HEL-01、DES-CMP-05、DES-CMP-06、DES-UI-16 | TC-FR-UNT-04-01〜17 | `internal/core/tactics`、`internal/core`、`internal/ui` | 実装済み（#35、#43、#37、#42。経験・成長、キャンペーンの持ち越し、戦闘中の再生） |
| FR-UNT-05 | #35 | DES-GRW-02 | TC-FR-UNT-05-01〜04 | `internal/core/tactics` | 一部実装（#35。成長の判定。成長率の定義データ（DES-DAT-01）は #36） |
| FR-UNT-07 | #8 | DES-SKL-01（反復 1 はキャラのスキルだけ。一部） | TC-FR-UNT-07-01〜07 | `internal/core/tactics` | 実装済み |
| FR-UNT-09 | #13、#27、#28、#46、#42、#45、#64、#65、#66、#75 | DES-UI-01、DES-UI-06、DES-UI-07、DES-UI-08、DES-UI-16、DES-UI-18、DES-UI-19、DES-UI-20、DES-UI-22 | TC-FR-UNT-09-01〜05、TC-FR-UNT-09-07〜12、TC-FR-UNT-09-14〜15、TC-FR-UNT-09-06・TC-FR-UNT-09-13（手動）。敵の範囲（#28）は TC-IF-01-18〜19、TC-IF-01-31〜32、TC-IF-01-33〜34（手動）。TC-FR-UNT-09-16（手動）、TC-FR-UNT-09-17〜18（自動・未実装）、TC-FR-UNT-09-19（手動・未実施） | `internal/ui` | 実装済み（#13、#27、#28、#46、#42、#45。#64 で巻き戻しボタンと情報欄の配置、#65 で配置の表示範囲も確認。手動のケースは開発者が確かめる。#66: 僧侶・杖の見分け TC-FR-UNT-09-16 を追加、今回は未確認（E-39））。IT04: #75 で要求改訂と試験文書の骨組みを作成（要求の文面は承認（E-43））。追加機能と自動ケースは後続 Issue で実装、手動は未実施 |
| FR-MAP-01 | #6、#76 | DES-BTL-03、DES-MAP-01、DES-RWD-05 | TC-FR-MAP-01-01〜05 | `internal/core/tactics` | 実装済み（#76: 移動・地形・射程の期待値を維持し、BoardView でも共通計算の回帰を確認） |
| FR-MAP-03 | #9、#75 | DES-MAP-02、DES-MAP-03、DES-RNF-01、DES-UI-24、DES-IT04 6.1 節（DES-MAP-03 の変更） | TC-FR-MAP-03-01〜07、TC-FR-MAP-03-08〜11（自動・未実装）、TC-FR-MAP-03-12（手動・未実施）、TC-FR-MAP-03-13〜15（自動・未実装） | `internal/core/tactics`、`internal/ui` | 実装済み（反復 1 のギミックは砦と増援）。IT04: #75 で要求改訂と試験文書の骨組みを作成（要求の文面は承認（E-43））。追加機能と自動ケースは後続 Issue で実装、手動は未実施 |
| FR-MAP-05 | #9 | DES-MAP-04（反復 1 はボスを倒す・主人公が倒れるだけ。一部） | TC-FR-MAP-05-01〜04 | `internal/core/tactics` | 実装済み（反復 1 の範囲） |
| FR-CMP-01 | #36、#37 | DES-DAT-01〜03、DES-CMP-01〜03、DES-CMP-06〜08、設計書（反復 2）10.4〜10.6 節 | TC-FR-CMP-01-01〜04 | `internal/defs`、`internal/defload`、`data`、`internal/core` | 実装済み（章の定義と流れ） |
| FR-CMP-07 | #36、#44、#40 | DES-DAT-01〜03（反復 2 の変更）、設計書（反復 2）10.8 節、DES-STY-01、DES-UI-12 | TC-FR-CMP-07-01、TC-FR-CMP-07-02〜04 | `internal/defs`、`internal/defload`、`data`、`internal/ui` | 実装済み（#40 で物語を表示） |
| FR-CMP-08 | #36、#37、#40 | DES-DAT-01〜03、DES-CMP-03、設計書（反復 2）10.3・10.4 節、DES-UI-12 | TC-FR-CMP-08-01〜02、TC-FR-CMP-08-03 | `internal/defs`、`internal/defload`、`data`、`internal/core`、`internal/ui` | 実装済み（#40 で加入を表示） |
| FR-CMP-09 | #9、#12、#40、#64 | DES-MAP-04、DES-CMP-07、DES-UI-15、DES-UI-17、DES-UI-19 | TC-FR-CMP-09-01〜06、TC-FR-CMP-09-07〜08 | `internal/core/tactics`（06 は `internal/core` の結合のテスト）、`internal/ui` | 実装済み（#40 で敗北とメモリからのやり直し）（#64 で反映前の巻き戻しと取消しを確認） |
| FR-ITM-01 | #8 | DES-BTL-05、DES-ITM-01 | TC-FR-ITM-01-01〜02 | `internal/core/tactics` | 実装済み |
| FR-ITM-02 | #9 | DES-ITM-01 | TC-FR-ITM-02-01〜03 | `internal/core/tactics` | 実装済み |
| FR-ITM-03 | #9、#11、#37、#41 | DES-ITM-01、DES-DAT-01、DES-CORE-01、DES-CMP-04、DES-CMP-06、DES-UI-13 | TC-FR-ITM-03-01〜03、TC-FR-CMP-03-09・11 | `internal/core`、`internal/core/tactics`、`internal/ui` | 一部実装（初期手持ち・章の報酬・在庫との入れ替えと画面。反復 2 の範囲。装備の入手・マップで拾う方法は対象外） |
| FR-SYS-03 | #5、#11、#36、#44、#37 | DES-DAT-01、DES-DAT-02、DES-CORE-01（反復 2 の変更を含む） | TC-FR-SYS-03-01〜11 | `internal/defs`、`internal/defload`、`data`、`internal/core` | 実装済み（#36 で反復 2 の定義データと自軍の置き方。#44 で僧侶・杖・リナと武器の変換。#37 でレベル・成長率・敵の名前も戦闘に渡す） |
| FR-SYS-04 | #5、#12、#36、#40 | DES-DAT-03（反復 2 の変更を含む）、DES-CMD-02 | TC-FR-SYS-04-01〜40 | `internal/defload`、`cmd/renkin` | 実装済み（#5、#12、#36、#40。#40 で Config によるタイトル起動へ変更） |
| IF-01 | #12、#28、#40、#45、#64、#65、#66 | DES-UI-02（DES-UI-01、03〜05、DES-CMD-02 を含む）、DES-UI-08、DES-UI-09〜12、DES-UI-15、DES-UI-17、DES-UI-18、DES-UI-19、DES-UI-20、DES-CMD-02 | TC-IF-01-01〜20、TC-IF-01-21〜30（手動）、TC-IF-01-31〜32、TC-IF-01-33〜34（手動）、TC-IF-01-35〜36、TC-FR-BTL-10-11〜13、TC-FR-BTL-10-14（手動）、TC-FR-CMP-11-05〜06、TC-FR-CMP-11-07（手動） | `internal/ui`、`internal/ui/assets`、`cmd/renkin` | 実装済み（#40 の場面遷移、#64 の巻き戻し一覧・再開、#65 の配置を含む。手動のケースは開発者が確認。#66: Windows の使用報告を記録。巻き戻し先の理解に課題、同じ結果や配置の個別操作は未確認（E-39、TST-IT03 3.2 節）） |
| IF-02 | #12、#13、#40 | DES-UI-01、DES-UI-10〜12、DES-UI-15 | TC-IF-02-01〜02、TC-IF-02-03（手動）、TC-FR-CMP-07-02〜04、TC-FR-CMP-08-03、TC-FR-CMP-02-03 | `internal/ui` | 実装済み（#40 の場面も日本語。手動のケースは開発者が確認） |
| IF-03 | #5、#36 | DES-DAT-02（反復 2 の変更を含む）、ADR-0003 | TC-IF-03-01〜03 | `data`、`internal/defload` | 実装済み（#36 で章・物語のファイルと 2 章・3 章のマップ） |
| NFR-03 | #7、#12、#66 | DES-NFR-02 | TC-NFR-03-01〜02、TC-NFR-03-03〜04（手動） | `.github/workflows/ci.yml`、`cmd/renkin` | 一部実装（#7、#12。Windows 11 の手動確認 TC-NFR-03-03 は済み（REV-015）。Linux の手動確認 TC-NFR-03-04 は未実施。反復 3 の範囲外として先の反復へ送る（Q-IT03-09、E-37、#66）。CI の自動検査とは区別する） |
| NFR-04 | #7 | DES-NFR-01 | TC-NFR-04-01〜04 | `internal/core`（`deps_test.go`） | 実装済み |
| NFR-05 | #7 | DES-NFR-01 | TC-NFR-05-01〜03 | `internal/core`（`deps_test.go`） | 実装済み |
| NFR-06 | #6、#35、#43、#61、#62、#76 | DES-RNG-01（反復 2 で追記）、DES-HEL-01、DES-RWD-03、DES-RWD-05 | TC-NFR-06-01〜03、TC-NFR-06-04〜06、TC-FR-BTL-10-06・16、TC-FR-BTL-10-22 | `internal/core/tactics` | 実装済み（#6。成長の乱数は #35、回復を含む再現性は #43。#61 で従来の PCG 列と内部復元の再現性を確認。#62 で公開巻き戻しの大盾・成長と再現性を確認）。#76: プレビューを挟んだ実 PCG と対照戦闘の一致を確認 |
| NFR-07 | #6、#9、#43、#37、#39、#61、#62、#76 | DES-ERR-01、DES-BTL-08、DES-HEL-01、DES-CMP-08、DES-SAV-03、DES-RWD-01〜04、DES-RWD-05 | TC-NFR-07-01〜08、TC-FR-CMP-03-03・06、TC-FR-CMP-01-03〜04、TC-FR-BTL-10-08〜09・18〜19、TC-FR-BTL-10-22〜23 | `internal/core/tactics`、`internal/core` | 実装済み（TC-NFR-07-04 は #9 で `UseItem` の検証 6 に書き換えた。#43 で TC-NFR-07-05 に誤り 3 つを追加し、TC-NFR-07-06 で回復の検証順を確認。#39 でセーブの検証順・I/O 失敗・状態不変を確認。#61 で失敗時の履歴・PCG・進行位置の不変を確認。#62 で巻き戻し・再開の誤りと回数・未来の不変、新しい誤り 4 個の識別を確認）。#76: プレビューの成功・失敗・返却値変更で状態不変を確認 |
| NFR-09 | #7 | DES-NFR-03 | TC-NFR-09-01〜02 | `.github/workflows/ci.yml` | 実装済み |
| IF-04 | #38、#40 | DES-PKG-08、DES-SAV-02、DES-CMD-02 | TC-IF-04-01〜06、TC-IF-04-07 | `internal/savefile`、`internal/ui`、`cmd/renkin` | 実装済み（#40 で保存先と起動を接続） |
| NFR-08 | #38 | DES-PKG-08、DES-SAV-02 | TC-NFR-08-01〜06 | `internal/savefile` | 実装済み |
| FR-CMP-02 | #37、#40、#64 | DES-CMP-06、DES-UI-15、DES-UI-17、DES-UI-19 | TC-FR-CMP-02-01、TC-FR-CMP-02-02〜03 | `internal/core`、`internal/ui` | 実装済み（#40 で戦闘終了・クリア画面）（#64 で反映前の巻き戻しと取消しを確認） |
| FR-CMP-03 | #37、#41、#63、#65 | DES-CMP-04、DES-CMP-05、DES-CMP-08、DES-UI-13、DES-DEP-01、DES-UI-20 | TC-FR-CMP-03-01〜11、TC-FR-CMP-11-01〜04 | `internal/core`、`internal/ui` | 一部実装（出撃選択・在庫の入れ替えと画面。装備選択は反復 2 の対象外。#63 で旧 API の互換と枠の指定。#65 で準備から配置を経る戦闘開始） |
| FR-CMP-04 | #37、#40 | DES-CMP-01、DES-CMP-03、DES-CMP-08、DES-UI-11 | TC-FR-CMP-04-01〜02、TC-FR-CMP-04-03 | `internal/core`、`internal/ui` | 実装済み（#40 でルール選択） |
| FR-CMP-05 | #37 | DES-CMP-05、DES-CMP-06 | TC-FR-CMP-05-01 | `internal/core` | 実装済み（離脱と次章での復帰） |
| FR-CMP-06 | #37 | DES-CMP-05、DES-CMP-06 | TC-FR-CMP-06-01 | `internal/core` | 実装済み（戦死者を名簿に残し、出撃不可・手持ち消失） |
| FR-CMP-10 | #37、#41 | DES-CMP-06、DES-UI-14 | TC-FR-CMP-10-01〜02 | `internal/core`、`internal/ui` | 実装済み（章の結果の生成と表示） |
| FR-SYS-01 | #39、#40 | DES-SAV-01、DES-SAV-03、DES-SAV-02、DES-CMP-07 | TC-FR-SYS-01-01〜06、TC-FR-SYS-01-07 | `internal/core/save.go`、`internal/ui` | 実装済み（#40 で自動保存とメモリ保持） |
| FR-SYS-02 | #40 | DES-UI-09、DES-UI-10 | TC-FR-SYS-02-01〜04 | `internal/ui`、`cmd/renkin` | 実装済み |
| FR-BTL-10 | #60、#61、#62、#64、#66、#75、#76 | DES-RWD-01〜04、DES-UI-19、DES-RWD-05、DES-UI-21 | TC-FR-BTL-10-01〜13、TC-FR-BTL-10-14（手動）、TC-FR-BTL-10-15〜20、TC-FR-BTL-10-21〜23（自動）、TC-FR-BTL-10-24〜25（自動・未実装）、TC-FR-BTL-10-26（手動・未実施） | `internal/core/tactics/snapshot.go`、`history.go`、`rewind.go`、`battle.go`、`act.go`、`event.go`、`boardview.go`、`move.go`、`internal/core`（`rewind_test.go`。結果反映の境界の確認）、`internal/ui`（`rewind.go`、入力・表示・場面遷移） | 実装済み（#61: 内部の深い写し・PCG 復元・履歴。#62: 公開の巻き戻し・再開・回数、未来の破棄、勝敗からの復帰と結果反映の境界。UI は #64。TC-FR-BTL-10-14 は #66 で Windows の使用報告を記録。表示・動作に異常の声なし、戻り先の理解は課題（不合格寄り）、同じ結果と各操作は未確認（E-39、TST-IT03 3.2 節））。IT04: #75 で要求改訂と試験文書の骨組みを作成（要求の文面は承認（E-43））。#76: BoardView・PreviewRewind・初期盤面・共通範囲を実装済み（TC-21〜23）。UI の TC-24〜25 は後続 Issue、手動は未実施 |
| FR-CMP-11 | #60、#63、#65、#66、#75 | DES-DEP-01、DES-UI-20、DES-DEP-02、DES-UI-23 | TC-FR-CMP-11-01〜06、TC-FR-CMP-11-07（手動）、TC-FR-CMP-11-08〜10（自動・未実装）、TC-FR-CMP-11-11（手動・未実施） | `internal/core`、`internal/ui` | 実装済み（#63: core の枠指定・検証・旧 API 互換・結果の名簿対応。#65: 配置・準備への戻り・戦闘開始。TC-FR-CMP-11-07 は #66 で Windows の配置使用を記録。迷った・おかしいという声なし、交換・空き枠・準備への戻り等は個別には未確認。敵の範囲表示を反復 4 へ要望（E-39））。IT04: #75 で要求改訂と試験文書の骨組みを作成（要求の文面は承認（E-43））。追加機能と自動ケースは後続 Issue で実装、手動は未実施 |

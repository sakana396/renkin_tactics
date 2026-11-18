# REV-122: #207 試作の準備の画面の取り込み（B-4）

- 日付: 2026-11-18
- 対象: Issue #207、PR #224、ブランチ `it08/207-prepare`（初回 `99d2a9d`）
- 方針・統合・記録: 管理役（Claude Code）
- 実装: Codex（5 回）。コミット・push・PR の作成、Windows 向けの実行ファイルでのテストとミュータント、TC-DES-NG-11-17 の期待値を main の #207 の前の出力に合わせる直し、`Game.sceneCursorAt` の切り出しとそのテスト、E-148 の書き起こしは管理役
- 独立レビュー: Claude spec-reviewer（Opus、読み取り専用）
- 根拠: IT08-PLAN v0.3（3.2 節 B-4、5 章 RV-4・RV-5、9 章）、DES-NG05 v0.5（承認 E-143。3 章・8.1〜8.5 節）、DES-IT02（DES-DAT-01・03）、TST-NG05、TST-NG04（TC-DES-NG-10-12・30）、TST-NG02（TC-DES-NG-08-99）、SRS v0.9 の FR-CMP-12、E-136、E-148、制作記録 map-ng-ch01〜03.md、試作のレビュー REV-110、REV-116 F-01
- 結論: 修正すれば承認（初回）→ 対応のあと承認（管理役が対応とミュータントを確かめた）

## 独立レビューで確かめたこと（`99d2a9d`）

- `prepare_hires_ng05.go`・`prepare_hires_ng05_draw.go` は試作の最終（REV-110 対応後）とバイト単位で同じ。テストの違いは現行の 1〜3 章に合わせた期待値と、在庫に鉄の斧を足す前提だけ
- 旧作の準備: TC-DES-NG-11-17 のゴールデンを origin/main（#207 の前）のコードに当てて動かし、全行が一致した。試作のゴールデンから直した 2 行（「道具　回復 10」）は #204（REV-116 F-01）の結果で、テストを緩めたことにはならない
- goal: 60 文字まで・空白だけは誤り（全角の空白も）で E-148 のとおり。1〜3 章の goal は制作記録の勝利条件とボスの名前に合う
- TC-DES-NG-11-14 の敵の数を `maps.toml` から数え直し、制作記録とも一致（ch01 18、ch02 11、ch03 10）
- 48 px の準備の画面: ch01〜ch03 と、名簿・在庫の行を選んだ状態を撮り、はみ出し・重なりはない
- 検証（レビュー担当）: gofmt 出力なし。`GOOS=windows go vet ./...` 成功。Linux の go vet・go test は ui・ui/audio・cmd・mkplaceholder・shot を除いて成功。Windows 向けの実行ファイルで `internal/ui`・`internal/ui/audio`・`cmd/renkin`・`tools/mkplaceholder` が PASS。tccheck OK。CI 成功

## 指摘と対応

| ID | 重さ | 指摘 | 対応 |
|---|---|---|---|
| F-01 | 高 | `Game.cursor()` が新作の準備でも (128,128) を引くので、実際のゲームでクリックがずれる（試作の分岐が取り込みで抜けた）。テストと撮影の `clickpx` は変換を通らないので見逃した | 新作の準備では窓の座標をそのまま使うようにした（Codex。`windowSceneCursor`）。Codex のテストは変換の関数だけを確かめていて、`Game.cursor()` からの呼び出しで分岐を消しても落ちなかったので、管理役が `Game.sceneCursorAt` を切り出し、Game から確かめるテスト（TC-DES-NG-11-24、`TestPrepareGameCursor`）を足した。分岐を消すと新作の行が落ちることを確かめた。TST-NG04 の TC-DES-NG-10-30 を「HiRes でない準備」と分かる書き方にした |
| F-02 | 中 | 新作の準備の回復薬の性能が「道具　HP を 10 回復」で、旧作（「道具　回復 10」）と違う。#211 が入ると効果の道具が「HP を 0 回復」になる | `itemDetailText` を `itemEffect` にまとめ、TC-DES-NG-11-20 の期待値と仕様の文を直した（Codex） |
| F-03 | 低 | `ToggleInfoPage`・`infoTabClick` の新作の分岐が抜けた | 戻した（Tab の選択音は残す）（Codex） |
| F-04 | 低 | `soundInputOf`・`hiresPrepare`・`sceneKeys` のコメントと設計 ID がない | 戻した（Codex） |
| F-05 | 低 | DES-IT02 v0.6 のクラス図・変更履歴の順・版の行、README の版 | 直した（Codex。E-148 の範囲） |
| F-06 | 低 | TST-NG02 の版と履歴、84 px の出どころ | TST-NG02 v0.6、84 px は実装のカードの肖像の大きさと書いた（Codex） |
| F-07 | 情報 | DES-NG05 と IT08-PLAN 5 章の文面が E-148 の前のまま | 後の文書の PR でまとめて直す（REV-120 F-09 と同じ） |
| F-08 | 情報 | TST-NG05 4 章の目視が「未実施」 | レビュー担当の撮影を記録した。窓での実際のマウス操作は #214 |
| F-09 | 人の判断 | E-148 の要約の確認 | 2026-11-18 に開発者が「よい」と確かめた |
| F-10 | 情報 | #211（PR #223）との衝突（`game.go`、効果の道具の丸の色） | #207 を先にマージし、#211 の側で合わせる |

## 実装の判断（設計の範囲内）

- TC-DES-NG-11-17 の期待値は、試作のゴールデンを main の #207 の前の出力に合わせた（origin/main で全行一致）
- TC-DES-NG-11-19・22 は、在庫に鉄の斧を足して送りを出す前提で確かめる
- DES-NG05 8.4 節の 9・10 は DES-NG04 と試験の置き場所から導ける

## 開発者の承認

- E-148（新作の準備に ↑↓・Enter と音、goal の検査を定義データの設計に書き足す、DES-NG05 8.4 節の 1〜8）をこの PR で書き起こした。要約は 2026-11-18 に開発者が「よい」と確かめた
- DES-IT02 v0.6 の追記（`docs/03_design` の変更）は、E-148 の「書き足す」による

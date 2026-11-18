# REV-120: #206 試作の 48 px の表示の取り込み（B-3）

- 日付: 2026-11-18
- 対象: Issue #206、PR #221、ブランチ `it08/206-hires`（初回 `7933253`）
- 方針・統合・記録: 管理役（Claude Code）
- 実装: Codex（4 回。3 回目は TC-DES-NG-10-12 の試験の準備の直し、4 回目は指摘の直し）。コミット・push・PR の作成、Windows 向けの実行ファイルでのテストとミュータントの実行、main（#208）の取り込みは管理役
- 独立レビュー: Claude spec-reviewer（Opus、読み取り専用）
- 根拠: IT08-PLAN v0.3（3.2 節 B-3、5 章 RV-4・RV-5、9 章）、DES-NG04 v0.4（承認 E-143。6 章 D-8〜D-12、8.5・9.3 節）、TST-NG04、TST-NG02（TC-DES-NG-08-99）、SRS v0.9 の FR-MAP-07、E-135、E-142、E-146、E-147、試作のレビュー REV-109
- 結論: 修正すれば承認（初回）→ 対応のあと承認（管理役が対応を確かめた）

## 独立レビューで確かめたこと（`7933253`）

- 試作と比べた: `layout.go`・`terrain_art.go`・`terrain_store.go`・`window_size.go`・`equipmark.go`・`zoc.go`・`command_mark.go`・`rewind.go`・`deploy.go`・`battle_windows_it07.go`・`boardnav.go`・`enemyrange*.go`・`minimap.go`・`shot.go`、テスト、`tools/shot/shotcfg`・`run.go` は試作とバイト単位で同じ。違いは main にある爆弾の一覧（`MenuRectsFor`）と、準備の画面の作り直し（#207）を入れていないことだけ
- 範囲: `internal/core`・`core/tactics`・`data/alchemy`・`prepare*.go`・`go.mod` は変えていない
- DES-NG04: `Config.HiRes` だけを足し、`Run`・`RunShot` は窓を作る前に `useConfigMetrics` を呼ぶ。`cmd/renkin` は新作のときだけ `HiRes: true`。盤面のない場面と敗北は 832×576 の層に描いて (128,128) に置く。巻き戻しの一覧を開いている間は層を使わない。地形の絵は PNG があればそれ、なければ仮の絵（48×48、変種 4）。全体図・ミニマップはドット絵の色のまま。印・棒は比例する。HiRes のときだけ窓の大きさを変えられ、`fitWindowSize` で画面の 9 割に収める
- 期待値: TC-DES-NG-10-05・06・08・09・11・28・31・32 は設計書の数値から手で導けた
- 旧作の見た目: 旧作と `-hires` なしの新作の撮影 10 枚が main と PR でバイト単位で同じ
- 48 px の画面: title・story・prepare・deploy・battle を撮り、はみ出し・重なり・欠けはない。敗北は旧作 2 章に `-hires` で撮った（新作 1 章では敗北にならない）
- 検証（レビュー担当）: `gofmt -l .` 出力なし。`GOOS=windows go vet ./...` 成功。Linux の `go vet`・`go test` は ui・cmd・mkplaceholder・shot を除いて成功。Windows 向けの実行ファイルで `internal/ui`・`cmd/renkin`・`tools/mkplaceholder` が PASS（`tools/shot/test` は Windows の実行環境のためにラッパーの検査だけ落ちる。Linux の CI では成功）。tccheck OK。CI 成功

## 指摘と対応

| ID | 重さ | 指摘 | 対応 |
|---|---|---|---|
| F-01 | 中 | 敗北の入口の位置を決める `Controller.gameOver` の配線（`Advance` で真、`syncRewind` で偽）を確かめるテストがない | 本番の経路（Lost のあと `Advance`、巻き戻しのあと戦闘へ）を通るテストを TC-DES-NG-10-29 に足した（Codex）。管理役が 2 か所の代入を消すミュータントを作り、どちらも TC-DES-NG-10-29 で落ちることを確かめた |
| F-02 | 中 | #208 と合わせると TC-DES-NG-10-17 の「指定なし」が落ちる。tools/shot・README・要求抽出記録が衝突 | #208 を先にマージし、管理役が main を取り込んで衝突を解いた。TC-DES-NG-10-17 の指定なしを `HiRes` が真に改め、TST-NG04 の行と 3 章の注記も直した。tools/shot の既定はどちらも base で TC-DES-NG-10-34 と合う |
| F-03 | 中 | TST-NG04 の一括の置き換えで文面が崩れた | 冒頭の段落、0.4 の変更履歴と 3 章の「試作の前提」を元に戻し、TC-34 を ch01・「水の止まった都」に直し、版 0.5 と履歴を足した（Codex） |
| F-04 | 中 | TST-NG04 2 章の main での確認が記録されていない | レビュー担当の撮影結果を管理役が確かめ、TST-NG04 2 章に記録した。2・3 章の 48 px の撮影と人の目での確認、窓の操作・大きさは #214 |
| F-05 | 低 | TC-DES-NG-10-12 の仕様の行が「準備は NG05 へ移した」のまま | 「#207 までは準備もここで確かめる」とした（Codex） |
| F-06 | 人の判断 | E-146 の要約の確認と `docs/01_concept` の変更の承認 | 2026-11-18 に開発者が E-146 の要約を「よい」と確かめた（E-147） |
| F-07 | 低 | `docs/README.md` の TST-NG04 の行が「未取り込み」 | 直した（Codex） |
| F-08 | 低 | PR の説明に型の見出し・元のコミット・REV-109 がない | PR の説明を書き直した |
| F-09 | 情報 | DES-NG04 8.5・9.3 節と IT08-PLAN 5 章 RV-4・RV-5 の文面が「承認を待つ」「未決」のまま | 後の文書の PR でまとめて直す（E-146 で決定済み） |
| F-10 | 情報 | TC-DES-NG-08-99 の高解像度の部分が media なしの準備で調べている | 直さない（顔の絵より厳しい側で、合否は変わらない） |
| F-11 | 情報 | 詳しい画面の見出しの切り詰めの幅 | `sceneCanvasWidth` 基準にした（Codex） |

## 実装の判断（設計の範囲内）

- 敗北の入口の位置は `Controller.gameOver` で切り替える（試作のまま）
- TC-DES-NG-10-12 の加入の場面は、試験用の目録で 1 章に勝ってから作る。準備も #207 までは TC-DES-NG-10-12 で調べる
- `tools/shot` を `shotcfg`・`run.go` に分けた。撮影は `-hires` を付けたときだけ 48 px（遊ぶ版の新作は常に 48 px）

## 開発者の承認

- E-146（DES-NG04 8.5 節の提案 3 つと 9.3 節の提案 1・2、`SHOT_NATIVE=1` の取り込み）をこの PR で書き起こした。要約は 2026-11-18 に開発者が「よい」と確かめた（E-147）
- REV-109 A-2（`Config.HiRes` と変数化）は、DES-NG04 v0.4 の承認（E-143）で決まっている

# REV-132: 表示の大きさを Metrics の値にまとめる（D-12）

- 日付: 2026-11-25〜2026-11-27
- 対象: PR #246、ブランチ `it09/233-metrics`（初回 `f3859d4`、再確認 `1632965`）。`internal/ui`（77 ファイル）、`docs/03_design/design-ng04-display.md`（DES-NG04 v0.6）、`docs/03_design/design-ng05-prepare.md`（DES-NG05 v0.7）、`docs/04_test/test-spec-ng04.md`（v0.7）、`docs/04_test/test-spec-ng05.md`、`docs/README.md`、`docs/traceability.md`、`docs/01_concept/elicitation-notes.md`（E-169 の確認、E-170）
- 方針・統合・記録: 管理役（Claude Code）
- 実装: Codex（段階 1〜5 の一部）→ Claude（Sonnet のサブエージェント）が仕上げ・指摘の直し
- 独立レビュー: Claude spec-reviewer（Opus、読み取り専用）
- 根拠: Issue #233、IT09-PLAN v0.2（承認 E-164。6 章・8 章 J）、DES-NG04 v0.5（承認 E-166）2・3・6 章 D-12、DES-NG05 v0.6 8.4 節、TST-NG04 v0.6、E-135・E-142・E-161、AGENTS.md
- 結論: 修正すれば承認（初回）→ 承認（再確認）。開発者が DES-NG04 v0.6・DES-NG05 v0.7 を承認した（E-170）

## 独立レビューで確かめたこと（`f3859d4`）

- コードの差分を、名前の入れ替え（`MapSize`→`m.Map` など）とみなして機械的に照合した。入れ替えでないのは、`NewController` の既定値（32）と `startDeployedBattle` での上書きだけだった。入力の座標の変換、カメラ、ホイール・ドラッグ、ミニマップ・全体図のクリック、予測窓の候補の順、巻き戻しの一覧と入口、配置の画面のクリック、`setWindow`・`Layout`・`RunShot` の式は変わっていない
- `Game`・`SceneController`・`Controller` の Metrics は、どれも同じ `Config` から作られる。ゼロ値の Metrics を使う本番の経路はない
- テストの期待値の数字は変わっていない（新しく足した TC-DES-NG-10-35・36 を除く）
- 管理役が整理の前（main）と後で 22 場面を tools/shot で撮り、22 場面ともバイト単位で一致することを確かめた
- 新しい import はなく、パッケージの依存の決まりと ADR-0002 に反しない
- 検証: `gofmt -l .` 出力なし、`GOOS=windows go vet ./...` 成功、`go test`（X11・alsa が要るものを除く）成功、`internal/ui` のテスト（GOOS=windows で作り WSL から実行）PASS、`go run ./tools/tccheck` OK（警告 56 件）

## 指摘と対応

| ID | 重さ | 指摘 | 対応 |
|---|---|---|---|
| F-01 | 中 | TC-DES-NG-10-29「既定（32）」のサブテストが、`hiresScenes` の 48 のまま動き、32 を確かめていない | 直した。`s.metrics`・`s.battle.metrics` を `baseMetrics()` に差し替え、32 で確かめる。48 にするミュータントで失敗を確かめた |
| F-02 | 低 | TC-DES-NG-16-05（高解像度）の Controller が 32 のまま。`hoverItemRow` が `baseMetrics()` で固定 | 直した。`c.metrics = mt` を入れ、`hoverItemRow` は `c.metrics` を使う |
| F-03 | 低 | DES-NG04 v0.6 案の 3 章・3.1 節に古い名前が残り、新旧が混ざる | 直した。2 章に名前の読み替えの注、3 章本文と 3.1 節の表を新しい名前に。8・9 章の経緯は当時の名前 |
| F-04 | 低 | DES-NG05 8.4 節 9 番に `useConfigMetrics` が残る | 直した。`metricsForConfig`（`Config.HiRes`）にし、DES-NG05 v0.7 にした（承認 E-170） |
| F-05 | 低 | DES-NG04 6 章の D-12 の持ち越しの理由（E-142）と提案の出どころを消した | 直した。経緯を戻し、「#233 での実装」の段落を足した |
| F-06 | 低 | `NewController` の既定値（32）が書かれていない。上書きが誤りの確認の前 | 直した。コメントを足し、代入を誤りの確認のあとへ移した |
| F-07 | 情報 | トレーサビリティの FR-MAP-07 の Issue の列に #233 がない | 直した |
| F-08 | 情報 | TC-35 と TC-01 の重なり、TC-03 が構造上あたりまえになった | 直さない |
| F-09 | 情報 | test-spec-ng04 の 4 章の前の空行、対象の版 | 直した |

## 再確認（`1632965`）

独立レビューが `git diff f3859d4..1632965` を読み、F-01〜F-09 の対応と新しい食い違いを確かめた。検証: `gofmt -l .` 出力なし、`GOOS=windows go vet ./...` 成功、`internal/ui` のテスト PASS、`go run ./tools/tccheck` OK。

- コードの変更は 3 つ（`scenes.go` の代入位置、`input.go` のコメント、テスト 2 本）で、撮影する画面を描く経路に影響しない
- 撮影（after2）では 22 場面のうち 20 場面が整理の前と一致した。新作のタイトル（87 画素）と物語（172 画素）は、画面全体に散らばった色の値 1 だけの差だった。main（整理の前）でタイトルを撮り直しても同じ差の画像（同じハッシュ）が出た。新しいコードで撮り直した物語は整理の前と一致した。同じコードでも 2 通りに揺れるので、撮影の揺れ（拡大して描く背景の絵の丸め）と判断した
- F-01〜F-07・F-09 は対応済み、F-08 は直さない

| ID | 重さ | 内容 | 対応 |
|---|---|---|---|
| R-01 | 情報 | DES-NG04 の変更履歴 0.6 の行に、3 章本文の直しと読み替えの注が書かれていない | 任意。直さない（次に DES-NG04 を変えるときに足す） |
| R-02 | 情報 | F-01 の期待値は `mt.Map` から作る。48 では一致しないので差し替えの漏れを捕まえられる | 直さない |

結論（再確認）: 承認。

## 開発者の承認

- DES-NG04 v0.6（Metrics の形、名前の読み替えの注、3 章、6 章の経緯と #233 の段落）と DES-NG05 v0.7（関数名の直し）: 2026-11-27、開発者が選択式の質問に「承認する (推奨)」と答えた（E-170）
- E-169 の要約: E-170 で開発者が「よい」と確かめた
- E-170 の要約は、次の承認のときに確かめる

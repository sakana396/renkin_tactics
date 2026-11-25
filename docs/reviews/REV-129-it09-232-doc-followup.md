# REV-129: 文書の後追い（UC-08 の名簿と「承認を待つ」「未決」の文面）

- 日付: 2026-11-25
- 対象: PR #243、ブランチ `it09/232-doc-followup`（初回 `dec2295`）。`docs/02_requirements/use-cases.md`（UC-DOC-001 v0.17 案）、`docs/02_requirements/srs.md`（SRS v0.12 案、E-165 で追加）、`docs/03_design/design-ng02-art.md`（v0.6 案）、`design-ng03-audio.md`（v0.5 案）、`design-ng04-display.md`（v0.5 案）、`design-ng05-prepare.md`（v0.6 案）、`docs/iterations/it08/plan.md`（IT08-PLAN v0.4 案）、`docs/README.md`、`docs/01_concept/elicitation-notes.md`（E-164 の確認、E-165）
- 方針・統合・記録: 管理役（Claude Code）
- 実装: Codex（初回と指摘の直し）
- 独立レビュー: Claude spec-reviewer（Opus、読み取り専用）
- 根拠: Issue #232、IT09-PLAN v0.2（承認 E-164。3.1 節・6 章・8 章 A）、IT08-RETRO v0.2 7.2 節、REV-116「RV-4 の扱い」、REV-117 F-10、REV-118 R-07、REV-120 F-09、REV-122 F-07、REV-127 F-10、REV-128、E-140・E-142・E-144〜E-148・E-152・E-153・E-159〜E-165、SRS-001 v0.11（FR-CMP-03）、test-spec-ng03 v0.6、traceability（FR-SYS-07〜09）、`data/base`・`data/alchemy` の定義、`ASSETS.md`、map-ng-ch01.md 7 章
- 結論: 修正すれば承認（初回）

## 独立レビューで確かめたこと（`dec2295`）

- 差分は文書 6 本だけ。コードと既存の E-xx は変わっていない
- Issue の一覧の対象は、すべて直っている
- 直した文面の根拠は E-xx の決定どおり（RV-4 を除く。F-01）
- UC-08 の名簿は定義と一致する（旧作 5 人で主人公はレオン。新作 4 人で主人公はアオイ。ch02 で調合の見習いが加わる）
- 検証: `gofmt -l .` 出力なし、`GOOS=windows go vet ./...` 成功、`go test`（X11・alsa が要るものを除く）成功、`go run ./tools/tccheck` OK（警告 56 件）、CI 成功

## 指摘と対応

| ID | 重さ | 指摘 | 対応 |
|---|---|---|---|
| F-01 | 中 | IT08-PLAN RV-4 を「決定」にしたが、新作の人物名の検索が残り、E-145 が抜け、REV-110 分の出どころが誤り | 直した。人物名の検索は残り、#239 で行う（E-165）。出どころを書き分けた |
| F-02 | 低 | `docs/README.md` の版の行が古い | 直した |
| F-03 | 低 | UC の変更履歴の順、UC-08 の印、「対象」に IT09-PLAN がない | 直した |
| F-04 | 低 | DES-NG03 の TC-DES-NG-09-26 の書き方と曲の質 | 直した |
| F-05 | 低 | DES-NG05 8.4 節の 9・10 の扱い | 直した。9 は決定に含める（E-165）、10 は実装の記録 |
| F-06 | 低 | SRS FR-CMP-03 と UC-08 の食い違い | 開発者の決定（E-165）でこの PR で直した。SRS v0.12 案 |
| F-07 | 情報 | IT08-PLAN の D-12 の行とリスク表が古い | 直した |
| F-08 | 情報 | DES-NG02 7 章の TC-DES-NG-08-94 の記述 | 直した |
| F-09 | 情報 | DES-NG04 6 章の #233 | 直さない（E-161 から導ける） |
| F-10 | 情報 | UC-08 に仮の人物名 | 直した。兵種で書いた |
| F-11 | 情報 | PR の説明の「テスト」 | PR の説明を直した |
| F-12 | 情報 | E-164 の要約の確認 | E-165 で開発者が「よい」と確かめた |

## 開発者の承認

- E-165: FR-CMP-03 をこの PR で直す、人物名の検索は #239、DES-NG05 8.4 節 9 を決定に含める、E-164 の要約でよい
- 6 文書と SRS の改訂の承認: （待ち）

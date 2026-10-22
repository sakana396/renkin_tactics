# REV-039: 巻き戻しの一覧・確認・敗北からの復帰（画面）のレビュー

- 対象: PR #71、ブランチ `it03/64-rewind-ui`（6ec2985、修正 1d6ee6a）。`internal/ui/rewind.go`（新規）・`input.go`・`game.go`・`scenes.go`・`scene_view.go`・`text.go` の変更、テスト（`rewind_test.go` の TC-FR-BTL-10-11〜13、`input_test.go` の TC-IF-01-12・13、`info_view_test.go` の TC-FR-UNT-09-12）、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`・`test-spec-it03.md`、`docs/traceability.md` の担当の行。Issue #64（設計書 DES-IT03 v0.2 の 9 章の Issue 4）
- 日付: 2026-10-23
- 作成: Codex（実装と修正。管理役が `codex exec` で頼んだ）。コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルでき、CI（`gh pr checks 71`）で確かめた
- `go run ./tools/tccheck` は整合 OK（警告 24 件は既存の手動のケース）
- DES-UI-19 を確かめた。入口は入力待ち・勝敗の帯・ゲームオーバーの選択だけで、Playing では T とボタンで開かない（TC-12、TC-IF-01-12）。一覧は古い順に 1 ページ 8 行で、最新の行を選んで開く。ページ送りは先頭と末尾で止まる。確定は確認を経て一度だけ。0 回と履歴なしでは理由を出す（TC-11）。取消しでは、仮の移動・選択・番の終了の確認を含む入力を保つ（TC-12）。成功時は再生列・未来の記録・選択・帯を消し、危険範囲の設定は保つ
- 6 章と UC-14 6a・6b を確かめた。再開を待つ間は自軍の操作と番の終了を受け付けない。「敵軍の行動を再開」と「番の処理を再開」を出し分け、同じ展開を再生する（TC-13）。勝利の帯・敗北の選択からの取消しでは場面を保つ。成功すると Battle へ戻り、反映前の Campaign と保存の回数は変わらない。最終の勝利だけを反映し、反映後は入口がない
- REV-038 の申し送り（RewindsLeft・NeedsResume・History の使用、帯の消去、移動だけの敵の To による出し分け）を確かめた
- 8.3 節の Issue 4 の行（TC-IF-01-12・13、TC-FR-CMP-02-02、TC-FR-CMP-09-07）は ID を保って補強している。TC-11〜14 は 3 か所でそろい、表の列数・閉じ・空行も正しい。ui は `golang.org/x` を直接 import していない
- TC-FR-BTL-10-14（手動）は未実施。Issue 6（#66）で開発者が Windows で確認する

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | 再開ボタン（`rewind.go:139`、y 440〜480）が盤面の 13〜14 行を隠し、戻した直後の盤面を確かめにくい（UC-14 6） | 設計どおりへの修正 | 直した（1d6ee6a）。案内欄（y 550〜574）へ移し、盤面に重ならないことをテストで確かめる |
| F-02 | 巻き戻しボタン（`rewind.go:132`）が案内の 2 行目に 2 px 重なる。TC-FR-UNT-09-12 は案内の文との重なりを調べていない | 設計どおりへの修正 | 直した（1d6ee6a）。高さを y 512〜530 にし、案内の文との重なりを TC-FR-UNT-09-12 で確かめる |
| F-03 | 自動の番の移行の行が、ボタンの文言の「番の処理を再開」になっている | 設計の範囲内 | 直した（1d6ee6a）。「番の移行（自動）」にした |
| F-04 | プルリクエストの説明の反復が IT02 になっている | 設計どおりへの修正 | 直した（説明を IT03 に。原因は管理役の PR 作成の手順で、手順も直した） |
| F-05 | TC-FR-UNT-09-12 の対象と traceability の FR-UNT-09 の設計の欄に DES-UI-19 がない | 設計どおりへの修正 | 直した（1d6ee6a） |

## 設計への提案の判定

なし（`HistoryEntry.Action` を表示に使うこと、ホイールで詳細を送ることは、設計の範囲内）

## 申し送り

- #65: 取り込みでは、`rewind_test.go` の準備からの `s.Advance()` を「配置へ」「戦闘開始」に置き換える。`drawSceneView` の先頭で配置を描く。test-spec-it01 の TC-FR-UNT-09-12 と、traceability の FR-UNT-09・IF-01 の行は双方の内容を合わせる
- #66: TC-FR-BTL-10-14 を Windows で確かめる。巻き戻しボタン・再開ボタンが案内の文と重ならないことも見る

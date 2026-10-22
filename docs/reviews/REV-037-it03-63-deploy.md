# REV-037: core の出撃枠の割り当てのレビュー

- 対象: PR #69、ブランチ `it03/63-deploy`（05ecaf1、main の取り込みと修正 b98f3b0）。`internal/core/campaign.go` の変更、テスト（`deployment_test.go` の TC-FR-CMP-11-01〜04、`campaign_test.go` の TC-FR-CMP-03-01〜03・TC-FR-CMP-10-01 の補強）、`docs/04_test/test-spec-it02.md`・`test-spec-it03.md`、`docs/traceability.md` の担当の行。Issue #63（設計書 DES-IT03 v0.2 の 9 章の Issue 3）
- 日付: 2026-10-22
- 作成: Codex（実装。管理役が `codex exec` で頼んだ）。コミット・main の取り込み・文書の軽い直し（F-01・F-02）・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめた。CI（`gh pr checks 69`）で確かめた
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-DEP-01 を確かめた。`StartBattleAt(members, slots, seed)` は `members[i]` を `slots[i]` の枠に置き、`members` とユニットの順を変えない。主人公も最後の枠・空き枠へ置ける（E-37、Q-IT03-07）。検証は、クリア済み → 人数 → 名簿 → 主人公 → 枠の長さ → 全要素の範囲 → 重複の順で、理由付きの `ErrInvalidDeployment` を返す。組み立てや tactics.New の失敗も含め、失敗したときは Campaign・入力・既存の Sortie が変わらない
- 旧 `StartBattle` は連番の slots で共通の処理を使い、従来の配置を保つ（TC-FR-CMP-03-01〜03 を ID のまま補強。8.3 節）。勝利の結果は出撃順で、元の名簿に対応する。配置はセーブに入らない（TC-FR-CMP-11-04、TC-FR-CMP-10-01）
- ui は変更していない

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | traceability の FR-CMP-03 の行に TC-FR-CMP-11-01〜04 を足したが、状態の列に #63 がなく、test-spec-it03 の 11-xx の対象（FR-CMP-11 だけ）とも食い違う | 設計どおりへの修正 | 直した（b98f3b0）。FR-CMP-03 の状態に「#63 で旧 API の互換と枠の指定」を書き、TC-FR-CMP-11-01〜04 の対象に FR-CMP-03 を足した |
| F-02 | test-spec-it03 の FR-CMP-11 の表の見出しが「対象」で、2 章で定めた「対象（要求・設計）」と違う | 設計どおりへの修正 | 直した（b98f3b0） |
| I-01 | traceability.md の FR-CMP-11 の行が #68 の FR-BTL-10 の行と隣り合い、衝突する | 情報 | 取り込みで両方の行を残して解いた。2 行とも 6 列で、閉じの区切りがあり、表の中に空行はない |

## 設計への提案の判定

なし（#65 で主人公を `members[0]` に保ち、`slots` だけを変えるという申し送りは、5.1・5.2 節のとおりで設計の範囲内）

## 申し送り

- #65: 配置の交換では `slots` だけを入れ替え、`StartBattleAt` は「戦闘開始」で一度だけ呼ぶ（7.2 節）

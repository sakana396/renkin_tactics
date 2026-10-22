# REV-040: 出撃の配置と準備からの遷移（画面）のレビュー

- 対象: PR #72、ブランチ `it03/65-deploy-ui`（7bb9e3d、main の取り込みと修正のあと）。`internal/ui/deploy.go`（新規）・`game.go`・`prepare.go`・`scenes.go`・`scene_view.go` の変更、テスト（`deploy_test.go` の TC-FR-CMP-11-05・06 と TC-FR-UNT-09-12、`prepare_result_test.go` の TC-FR-CMP-03-07・10、`scenes_test.go` の出撃手順）、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`・`test-spec-it03.md`、`docs/traceability.md` の担当の行。Issue #65（設計書 DES-IT03 v0.2 の 9 章の Issue 5）
- 日付: 2026-10-23
- 作成: Codex（実装と、main の取り込みの衝突の解決。管理役が `codex exec` で頼んだ）。コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。取り込みと修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: #71 のあとに取り込み、衝突を解いたので、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルでき、CI（`gh pr checks 72`）で確かめた
- `go run ./tools/tccheck` は整合 OK（警告 25 件は手動のケース。TC-FR-BTL-10-14 と TC-FR-CMP-11-07 を含む）
- DES-UI-20 と DES-UI-13 の変更を確かめた。準備の「配置へ」で配置に入り、主人公を先頭とする連番の枠から始まる。入れ替えるのは slots だけで、members の順は保つ。主人公も交換でき、空いたマスへも移せる（E-37）。本人の再クリックで選択を解除する。枠外・敵・壁では位置を変えず理由を出す。選択中の取消しは解除、未選択なら準備へ戻る。再び入ると最初の並びから始まる。開始の失敗では配置と Campaign を保ち、開始後は準備へ戻らない（TC-05。出撃 1・4・5 人）
- NextSeed を呼ぶのは「戦闘開始」の 1 回だけで、配置中は Battle と Sortie を作らない。最初の番の出来事は 1 回だけ（TC-06）。地形・出撃の枠・初めの敵を描き、増援は描かない。選んだ味方の情報、案内、ボタンは画面に収まる（TC-FR-UNT-09-12）
- REV-037 の申し送りを確かめた。8.3 節の Issue 5 の行（TC-FR-CMP-03-07〜11）は ID を保ち、「配置へ」に合わせて直している。TC-05〜07 は 3 か所でそろい、表の列数・閉じ・空行も正しい。ui は `golang.org/x` を直接 import していない
- main（#71）の取り込みを確かめた。衝突の印は残っていない。`drawSceneView` の先頭で配置を描く。#71 のテストの準備からの出撃は「配置へ」「戦闘開始」の手順に直した。配置の場面では巻き戻しを開かない。traceability の FR-UNT-09・IF-01 の行は #64・#65 の双方の内容を持ち、6 列で閉じがある
- TC-FR-CMP-11-07（手動）は未実施。#66 で開発者が Windows で確認する

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| G-01 | `deploy.go:124-125` の武器の種類の対応表は描画で使われない。地形と敵の HP の変換は core の写しなので、あとで食い違うおそれがある | 設計の範囲内 | 直した（取り込みのコミット）。対応表を消し、core の `convertTerrain`・`enemyUnit` と同じ式であることをコメントに書いた |
| G-02 | プルリクエストの説明の反復が IT02 になっている | 設計どおりへの修正 | 直した（説明を IT03 に。原因は管理役の PR 作成の手順で、手順も直した） |

## 設計への提案の判定

なし（旧 `SceneController.StartBattle` を `startDeployedBattle` に置き換えたことは、7.2 節のとおりで設計の範囲内）

## 申し送り

- #66: TC-FR-CMP-11-07 と TC-FR-BTL-10-14 を Windows で確かめる。開始後の敵の表示が配置の画面と一致するかも見る。test-spec-it02 の 140・146 行目にある表の中の空行（既存）を直す

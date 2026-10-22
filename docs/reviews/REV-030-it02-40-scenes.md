# REV-030: 場面の切り替え・タイトル・章の流れと起動（UI）のレビュー

- 対象: PR #55、ブランチ `it02/40-scenes`（e758926、修正 a330ccd）。`internal/ui/scenes.go`・`scene_view.go`・`style.go` の新規作成、`internal/ui/game.go`・`input.go`・`layout.go`・`text.go` の変更、`cmd/renkin/main.go` の DES-CMD-02 への置き換え、テスト（`scenes_test.go`、`cmd/renkin/main_test.go`、反復 1 のテストの書き換え）、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`、`docs/traceability.md` の担当の行。Issue #40（設計書 v0.3 の 9 章の Issue 6）
- 日付: 2026-10-22
- 作成: Codex（実装と修正。管理役が `codex exec` で頼んだ）。コミットとプルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui`・`cmd/renkin` は CI で確かめた（`gh pr checks 55`）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- 修正後、`go mod tidy -diff` は差分なし（実装者の報告）
- REV-029 の申し送りを確かめた。`Campaign.Save`・`LoadCampaign` を `SaveStore.Write`・`Read` に渡している。新規開始・勝ち・続きからのたびに、章の始めをメモリに持つ（DES-CMP-07）。やり直しはファイルを読まない（TC-FR-CMP-09-07）。`ErrSaveVersion`・`ErrInvalidSave` の文をタイトルに出す（DES-UI-10）
- DES-CMD-02 を確かめた。`savefile.ResolveDir` で置き場所を決め、`-savedir` を優先する。決められなければ標準エラーに書いて終了コード 1 で終え、画面は動かさない（TC-IF-04-07）
- DES-UI-17 を確かめた。R・Esc をやめ、勝利・敗北の帯のクリックでだけ遷移する。勝ちのときは `FinishChapter` とセーブを 1 回ずつ行う（TC-FR-CMP-02-02）
- 負けてやり直したときは、物語と加入を出さずに準備から始める（DES-STY-01、DES-UI-12）
- 仮の準備と仮の結果は 9 章の注記どおり。`DefaultMembers`・`StartBattle(members)` は #41 が置き換えられる形で、REV-028 I-01 の妨げにならない
- #41・#42・#45 の範囲は先取りしていない
- 9 章の「書き換える反復 1 のテスト」の Issue 6 の行（TC-IF-01-13・14・17・20、TC-FR-SYS-04-22・23）は、ID を変えずに直し、test-spec-it01.md の行もそろえた

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `internal/ui/metrics.go` が `golang.org/x/image/font` を直接 import している。ui の依存の規則（AGENTS.md）に反し、go.mod の `// indirect` とも合わない | 設計どおりへの修正 | 直した（a330ccd）。`textWidth` を Ebitengine の `text.Measure` で測る形に戻し、`metrics.go` を消した |
| F-02 | `Controller.Restart`・`Quit` を何もしない関数として残している。書き換えた TC-IF-01-13・14・20 は、それが何もしないことを確かめているだけで、規則を確かめていない（DES-UI-17） | 設計どおりへの修正 | 直した（a330ccd）。両方を消し、TC-IF-01-13 は Over で Escape・Cancel を押しても状態・戦闘・場面が変わらないことを、TC-IF-01-14 は初期作成の誤りを返すことを確かめる。ID は変えず、test-spec-it01.md も合わせた |
| F-03 | 勝ったあとのセーブの失敗の文が、結果のあとも、物語・加入・準備にずっと出る（DES-UI-12・14、DES-SAV-02） | 設計どおりへの修正 | 直した（a330ccd）。結果の場面から進むときに `saveWarning` を空にし、TC-FR-SYS-01-07 で物語・加入・準備に出ないことを確かめる |
| F-04 | `FinishChapter` が失敗しても、戦闘の場面に理由が出ない。状態は変わらない | 設計どおりへの修正 | 直した（a330ccd）。`SceneController.BattleGuide` で戦闘の案内欄に出す |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | 加入が複数あれば、1 人ずつクリックで送って表示する | 設計の範囲内（DES-UI-12） |
| 2 | 存在確認が失敗したときもセーブがあるものとして扱い、上書きの確認を出す | 設計の範囲内（DES-UI-10「読めないものを含む」） |

## 申し送り

- #41（準備・結果）: `SceneController` の仮の準備と仮の結果（`DefaultMembers`、`View` の `Prepare`・`Result`）を置き換える。準備で配置したあとに戻れないこと（REV-028 I-01）は #41 で扱う
- 手動のテスト（TC-IF-01-22・28〜30）は、開発者が Windows で確かめる

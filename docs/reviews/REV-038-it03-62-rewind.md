# REV-038: 巻き戻し・回数・敵軍の途中からの再開のレビュー

- 対象: PR #70、ブランチ `it03/62-rewind`（bd8d9fb、修正 7e9f49e）。`internal/core/tactics/rewind.go`（新規）・`act.go`・`history.go`・`battle.go`・`errors.go`・`snapshot.go` の変更、テスト（`tactics/rewind_test.go` の TC-FR-BTL-10-01〜10、`core/rewind_test.go` の TC-FR-BTL-10-10、`act_test.go` の TC-NFR-07-05）、`docs/04_test/test-spec-it01.md`・`test-spec-it03.md`、`docs/traceability.md` の担当の行。Issue #62（設計書 DES-IT03 v0.2 の 9 章の Issue 2）
- 日付: 2026-10-22
- 作成: Codex（実装と修正。管理役が `codex exec` で頼んだ）。コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめた。ui・cmd は変更していない
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-RWD-02 を確かめた。Rewind は「残り回数 → 実行済みの ID → restore → 未来の破棄と 1 回の消費」の順で検証する。restore の成功後に置き換えるので、失敗時は状態・PCG・履歴・次の ID・回数・続きが変わらない（TC-08 で比べている）。`continuePhase` を通常の進行と Resume で共用し、済んだ敵・増援・砦の回復を二重に処理しない。その番の増援は動かない（TC-04・05）。敵軍の行動前に戻した状態の自軍の操作は `ErrNotPlayerPhase`、自動移行・番の開始前に戻した状態は `ErrResumeRequired` を返す
- DES-RWD-03 を確かめた。巻き戻しのテストは実 PCG だけで行い、同じ行動の出来事・全戦闘値・PCG が一致する。同じ seed と巻き戻しを含む入力列も一致する（TC-06。E-37 の「同じ行動なら同じ結果」）
- DES-RWD-04 と 6 章を確かめた。Battle は残り 10 回で始まり、成功したときだけ 1 回減る。巻き戻しは Campaign を変えない。勝ち直したあとの勝利だけを一度反映する。再出撃・ロード・次章の戦闘は残り 10 回で履歴を持たない（TC-10）
- REV-036 の申し送りを確かめた。F-01 は、`NeedsResume` で Result を見て、勝敗で抜けるときに progress を戻すことで対応した。F-02 は、restore のコメントに書き、テストで固定列を使わないことで対応した。F-03 は、TC-06 で Pavise と LevelUp を確かめることで対応した
- TC-NFR-07-05 を誤り 18 個に書き換えた（8.3 節）。TC-01〜10 は 3 か所でそろい、表の列数・閉じ・空行も正しい
- tactics の import は標準ライブラリだけ（ADR-0002）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | TC-FR-BTL-10-07 は、毎回新しく作った写しに戻すので、「古い写しに戻しても回数は増えない」を見分けられない | 設計どおりへの修正 | 直した（7e9f49e）。A・B を記録し、B の前で残り 9 回、古い A の前へ戻して残り 8 回になることを確かめる。写しの回数を戻すミュータントで失敗することを確かめた |
| F-02 | traceability の FR-BTL-10 の実装欄に、変更していない `internal/core/campaign.go` を挙げている | 設計どおりへの修正 | 直した（7e9f49e）。`internal/core`（`rewind_test.go`）にした |
| F-03 | `history.go` の `historyEntries` のコメントと関数の間に空行が残っている | 設計どおりへの修正 | 直した（7e9f49e） |

## 設計への提案の判定

なし（`SetRoll` を restore で保存しないこと、`HistoryEntry`・`HistoryKind` の公開、TC の 01〜10 と 11〜14 の分担は、設計の範囲内）

## 申し送り

- #64: 一覧の 0 回・閲覧・取消しと Resume のボタンは、`RewindsLeft`・`NeedsResume`・`History` を使う。Over とゲームオーバーの選択からの入口では、Rewind が成功したら勝敗の帯を消す
- #64: REV-036 I-01 のとおり、移動だけの敵の履歴は Action が Wait なので、To で移動の有無を表示し分ける

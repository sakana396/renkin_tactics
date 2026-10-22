# REV-036: tactics の状態の写し・乱数の復元・行動の履歴の土台のレビュー

- 対象: PR #68、ブランチ `it03/61-snapshot`（d107531）。`internal/core/tactics/snapshot.go`・`history.go`・`battle.go`・`act.go`・`event.go` の変更、テスト（`snapshot_test.go` の TC-FR-BTL-10-15〜20、`act_test.go` の TC-NFR-06-01）、`docs/04_test/test-spec-it01.md`・`test-spec-it03.md`、`docs/traceability.md` の担当の行。Issue #61（設計書 DES-IT03 v0.2 の 9 章の Issue 1）
- 日付: 2026-10-22
- 作成: Codex（実装。管理役が `codex exec` で頼んだ）。コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: マージしてよい（指摘はいずれも低。#62 への申し送りとする）

## 検証

- `gofmt -l .` は出力なし。`go vet`（ui・cmd を除く）と `GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめた。CI（`gh pr checks 68`）は成功
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- DES-RWD-01 を確かめた。写しは Turn・Phase・Result、全ユニット（手持ちは `clone` で深く写す）、進行位置、PCG のバイト列、出来事の通し位置を持つ。回数・次の ID・履歴は持たない。地形と増援の定義は New で入力から切り離して共有し、戦闘中に書き換えない。戻り先は、自軍の移動前、敵軍の番の開始前、各敵の処理前（Hold を変える前）、Turn を進める前で、明示の番終了と自動移行を区別する。動かない敵も待機の履歴を残す。失敗した Act・EndPhase、参照、予測では履歴・ID・乱数が変わらない
- DES-RWD-03 を確かめた。`rand.NewPCG(seed, 0)` を明示的に持つ。復元は別の PCG の `UnmarshalBinary` が成功してから一度に置き換え、壊れたバイト列では何も変えない。従来の `rand.New(rand.NewPCG(42, 0))` の列と一致する（TC-NFR-06-01 を補強。8.3 節）
- tactics の import は標準ライブラリだけ（ADR-0002）。公開の巻き戻し・再開の操作は足していない（#62 の担当）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | 敵軍の番の途中で勝敗が決まると `progress` が敵の処理の途中のまま残る。`NeedsResume` を progress だけで判定すると、終わった戦闘で続きがあると答える | 設計どおりへの修正 | #62 への申し送り（勝敗で抜けるときに progress を戻すか、`NeedsResume` で Result を見る） |
| F-02 | `restore` が roll を本物の PCG に戻すので、`SetRoll` の差し替えが黙って消える（4.3 節） | 設計どおりへの修正 | #62 への申し送り（固定列を使うなら列と位置を一緒に保存する。使わないなら restore のコメントに書く） |
| F-03 | TC-FR-BTL-10-16 は大盾の判定を通ったことを確かめていない | 設計どおりへの修正 | #62 の TC-FR-BTL-10-06 で、大盾の乱数を引く場合を確かめる |
| I-01 | 移動だけの敵の履歴は Action が Wait で、To だけが変わる | 情報 | #64 の一覧で移動の有無を表示し分ける |
| I-02 | traceability.md の FR-BTL-10 と FR-CMP-11 の行が隣り合い、#69 と衝突する | 情報 | #69 の取り込みで両方の行を残し、列数と閉じを確かめる |

## 設計への提案の判定

なし（TC 番号を 15〜20 にして 01〜14 を後続に残すこと、出来事の境界を通し添字の半開区間にすることは、設計の範囲内）

## 申し送り

- #62: F-01・F-02・F-03 への対応。TC-FR-BTL-10-01〜14 を使い、16 の内容（PCG の復元）と重複させない
- #64: 一覧の敵の行で、移動したかどうかを To から判断する

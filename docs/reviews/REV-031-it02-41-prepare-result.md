# REV-031: 章の準備と章の結果（UI）のレビュー

- 対象: PR #56、ブランチ `it02/41-prepare-result`（7c7b9a6、main の取り込みと修正のあと）。`internal/ui/prepare.go` の新規作成、`internal/ui/scenes.go`・`scene_view.go`・`game.go`・`growth.go`・`text.go` の変更、テスト（`prepare_result_test.go`）、`docs/04_test/test-spec-it02.md`、`docs/traceability.md` の担当の行。Issue #41（設計書 v0.3 の 9 章の Issue 7）
- 日付: 2026-10-22
- 作成: Codex（実装と修正。管理役が `codex exec` で頼んだ）。コミット・main の取り込み・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめ、実行は CI で確かめた（`gh pr checks 56`）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- ui は golang.org/x を直接 import していない（REV-030 F-01）。何もしない関数をテストで確かめる箇所はない（REV-030 F-02）
- REV-030 の申し送りを確かめた。仮の準備（`DefaultMembers`）と仮の結果を置き換えた。出撃したあとは、戦闘中の取消し・選択・クリックで準備に戻らない（REV-028 I-01。TC-FR-CMP-03-10）
- DES-UI-13 を確かめた。生きているキャラだけを名簿の順に出す。主人公と枠の数までに印を付け、主人公を選んだ状態で始める。主人公の印は外せない。枠を超えたら「出撃は N 人までです」と出す。在庫と手持ちのボタンで `TakeItem`・`StoreItem` を呼び、`ErrHandFull` は「手持ちは 5 個までです」と出す（UC-11 の 4a）。出撃のときは主人公を先頭にして、あとは名簿の順に並べる。種は 1 回だけ取る
- DES-UI-14 を確かめた。「第 N 章　章の名前　勝利」、上がった能力だけ、離脱の説明、赤い「戦死」、報酬の文を出す。報酬がなければ報酬の文を出さない
- 9 章の「書き換える反復 1 のテスト」に Issue 7 の行はない。反復 1 のテストは書き換えていない

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `test-spec-it02.md` の Issue 案 7 の表で、見出しと行の間に空行があり、TC-FR-CMP-03-07〜11・TC-FR-CMP-10-02 が表として表示されない | 設計どおりへの修正 | 直した（027fffa。main を取り込んで衝突を解くときに空行を消した） |
| F-02 | 新規開始のセーブの失敗の文が準備の場面まで残り、名簿の 1 行目に重なる（DES-UI-12・13） | 設計どおりへの修正 | 直した。`enterPrepare` で `saveWarning` を空にし、TC-FR-CMP-03-07 で通常の進行と物語を飛ばす場合の両方を確かめる |
| I-01 | 準備の情報欄に「Lv・経験」の行を独自に足している。#42 が `addUnit` に同じ行を入れたので、両方をマージすると重なる | 情報 | 直した。`prepareInfo` の追加行を消し、共通の情報欄の表示を使う。TC-FR-CMP-03-11 で「Lv 20　経験 ―」が 1 行だけ出ることを確かめる |
| I-02 | `resultGains` と #42 の `growthText` がほぼ同じ処理 | 情報 | まとめた。`growthText` は伸びがなければ空を返し、成長の帯（`EventText`）の側で「上がった能力なし」を出す |
| I-03 | `prepareUnit` が武器・スキルの変換を ui の中でもう一度書いている | 情報 | 対応なし |

## 設計への提案の判定

なし

## 申し送り

- 実際の描画（行の明るさ、情報欄、ボタンの配置）は、開発者が Windows で確かめる

# REV-075: 開発用の道具「ゲームの画面を窓なしで PNG に書き出す」（#143）のレビュー

- 対象: ブランチ `it06/143-shot-tool`（4431208）。PR #144。Issue #143。`internal/ui/shot.go`（新規）、`internal/ui/game.go`、`tools/shot/`（main_windows.go・main_other.go・shot.sh・README.md・test/shot_test.go）
- 日付: 2026-11-01
- 作成: Codex（実装）、Claude Code（管理役。コミット・PR の作成）
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 要求・設計の文書と ADR は変更していない。開発用の道具を作ることは開発者が了承した（2026-11-01「tools作っていい」）。テストケース ID の対象外（tools/tccheck と同じ扱い）
- 結果: 承認。F-01〜F-03 は道具の中の軽微な点で、完了条件は満たしているため、申し送りとした（次に道具を触る Issue で直す）。設計への提案はない

## 検証

- `gofmt -l .` は出力なし。`go vet`・`go test`（internal/ui・cmd/renkin 以外）は成功し、`tools/shot/test` も成功。internal/ui・cmd/renkin は `GOOS=windows go vet ./...`、`GOOS=windows go build ./...`、`GOOS=windows go test -c` で確かめた（WSL に X11 のヘッダがないため。テストの実行は CI）。`go run ./tools/tccheck` は整合 OK（既存の警告 54 件）。CI（`gh pr checks 144`）は成功
- WSL から `tools/shot/shot.sh` で、タイトル、ch09 の準備・戦闘・巻き戻しの一覧、ch01 のユニットの選択・ターン 2・攻撃の予測を撮った。すべて 832×576 で、画像を目で見て場面が合っていることを確かめた。種が 1 に固定なので、敵の番を挟んでも毎回同じ盤面になる。管理役も ch09 の戦闘を撮り、約 5 秒で窓を出さずにできることを確かめた
- 本体への影響: `Game` に足した `shotCursor` は通常の起動では nil で、Draw の 1 か所の判定にしか出てこない。`Game.Update` は変えていない。`RunShot` は cmd/renkin から呼ばれないので、本体の実行ファイルには入らない
- セーブ: 道具は読み書きしない保存先（Exists は false、Write は捨てる）を渡し、`internal/savefile` を読み込まない

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `clickpx`・`cancel` の扱いが `Game.Update` と一部違う（戦闘が終わった状態のボタン、敗北の場面の巻き戻しの一覧）（低） | 道具の中の修正 | 申し送り。次に shot.go を触る Issue で、Update と同じ分け方にする |
| F-02 | README に、出来事の再生中は `tick:N` が要ることと、行動していない自軍がいると `endphase` が 2 回要ることがない。攻撃の予測の手順の例もない（低） | 道具の中の修正 | 申し送り。例: `click:1:14,click:4:12,action:wait,tick:300,endphase,endphase,tick:6000,click:4:12,click:4:12,action:attack,target:4:11`（ch01 の攻撃の予測） |
| F-03 | shot.sh が `-out=パス` の形を Windows のパスに変換しない（低） | 道具の中の修正 | 申し送り。それまでは `-out パス` の形で書く |

## 実装者の判断の判定

| # | 判断 | 判定 |
|---|---|---|
| P-01 | 撮影の処理を `internal/ui/shot.go` に置き、Game に `shotCursor` を足す | 範囲内（Issue の「小さな入口」） |
| P-02 | 隠した窓で 1 フレームだけ描き、画面外の画像を PNG にして終える | 範囲内（Issue の「実装の方法は自由」） |
| P-03 | 成り立たない操作は誤りにせず無視する（README に書いてある） | 範囲内 |

## 申し送り

- F-01〜F-03（上の表）
- #129（PR #139）や #127 で情報欄のページや新しい操作が入ったら、shot.go に操作（ページの切り替えなど）を足し、README の表を更新する。ボタンで切り替えるものは、今の `clickpx` で撮れる
- 道具の自動テストは shot.sh の引数の受け渡しだけを確かめている。撮影そのもの（`RunShot`）は Windows の描画が要るので、手で撮って確かめる

# REV-032: 戦闘中の経験・レベルアップの再生と情報欄のレベル（UI）のレビュー

- 対象: PR #57、ブランチ `it02/42-levelup-ui`（bdbd564）。`internal/ui/growth.go` の新規作成、`internal/ui/game.go`・`info.go`・`input.go`・`text.go` の変更、テスト（`growth_test.go`、`info_view_test.go` の TC-FR-UNT-09-12）、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`、`docs/traceability.md` の担当の行。Issue #42（設計書 v0.3 の 9 章の Issue 8）
- 日付: 2026-10-22
- 作成: Codex（実装。管理役が `codex exec` で頼んだ）。コミットとプルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: マージしてよい（#56 との順は申し送りのとおり）

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめ、実行は CI で確かめた（`gh pr checks 57`）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- ui は golang.org/x を直接 import していない（REV-030 F-01）
- DES-UI-16 を確かめた。`Exp` はユニットのマスの上に「+N 経験」を 20 フレーム出す。`LevelUp` は盤面の中ほどの帯に、上がった能力だけを 60 フレーム出す（何も上がらなければ「上がった能力なし」）。早送りが効く。`EventText` に `Exp`・`LevelUp` の文を足し、戦闘の記録と案内欄にも出る
- 期待値は設計から導かれている。経験 80+30 で Lv 2・経験 10（DES-GRW-01・02）。写しは `Exp` の時点で更新後の値（DES-GRW-03）
- 情報欄は、自軍なら名前とクラスの下に「Lv N　経験 N」を出し、Lv 20 は「―」、敵（Lv 0）は出さない
- 書き換えた反復 1 のテスト TC-FR-UNT-09-12 は ID を変えず、`test-spec-it01.md` の行も合わせた。予測と情報欄の位置の期待値は変えていない

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| I-01 | #41 の準備の情報欄（`prepareInfo`）にも Lv の行があり、両方をマージすると重なる | 情報 | #56 の側で `prepareInfo` の追加行を消す（REV-031） |
| I-02 | 9 章の書き換えの表に Issue 8 の行はないが、TC-FR-UNT-09-12 に Lv の確認を足した（ID は同じ、期待値の位置は変えていない） | 情報 | 対応なし（設計の範囲内） |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| 1 | Lv 0 の自軍には Lv を出さない | 設計の範囲内（反復 2 の自軍は Lv 1 以上。DES-GRW-03 規則 7） |

## 申し送り

- #41 とのマージ: `test-spec-it02.md` の 161〜170 行あたりが衝突する。`resultGains`（#41）と `growthText` は、マージ後に 1 つにまとめてもよい
- #45（回復の画面）: 回復のあとの `Exp`・`LevelUp` は、この再生処理でそのまま出せる
- 帯と経験の表示は、開発者が Windows で確かめる

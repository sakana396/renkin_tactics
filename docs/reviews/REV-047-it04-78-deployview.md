# REV-047: core の配置用盤面を組み立てる口のレビュー

- 対象: PR #88、ブランチ `it04/78-deployview`（e7fedce、main の取り込み a6b8d8d）。Issue #78（DES-IT04 9 章の Issue 3）。`internal/core/campaign.go`、`deployment_view_test.go`（新規）・`deployment_test.go`、`docs/04_test/test-spec-it04.md`・`test-spec-it03.md`、`docs/traceability.md` の 1 行
- 日付: 2026-10-23
- 作成: Codex（実装・テスト・文書の更新。管理役が `codex exec` で頼んだ）。コミット・プルリクエスト・main の取り込みは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（`docs/00_plan/`〜`docs/03_design/`・`docs/adr/` を変更していない）
- 結果: 直さないとマージできない指摘はない。F-01 は後からマージする PR #87 で直す。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功。Linux の `go vet`・`go test` は、X11 がないため ui・cmd を除いて成功。`go run ./tools/tccheck` は整合 OK（TC-FR-CMP-11-08・09 の警告が消えた）
- `Campaign.DeploymentView` は `deploymentSetup` を StartBattleAt と共用している（出撃者・枠の検証、現在名簿の変換、`setupWithAllies`）。最後に `tactics.NewBoardView` へ渡し、tactics.New・NextSeed・番処理は呼ばない（5.1 節）。失敗時はゼロ値と、StartBattleAt と同じ文面の誤りを返し、Campaign と入力を変えない
- TC-08 の敵の期待値（HP 22・Str 4・Skl 6・Spd 8・Def 10・Mov 4、「兵 1」、射程 2〜2、Hold、Items）は、クラスの基礎値と Bonus から導いている。全フィールドを手で書いた構造体と比べ、増援が入らないこと、地形の全値、初期の Turn・Phase・Result・再開待ちも確かめている。TC-09 の期待マスは手で計算して一致した
- TC-FR-CMP-11-01〜03 は既存の確認を残し、配置表示の一致、誤りの文面と検証順の一致、ゼロ値、状態不変を足した（8.3 節）。弱めた箇所はない
- PR #87 は予兆を `Battle.BoardView`・`PreviewRewind` にだけ足し、`NewBoardView` には足していない。予兆は T が 2 以上の該当ターンにだけ出るので、配置には出ない。矛盾はない
- ミュータント検査を別の写しで 4 件行った。slots の無視、Cleared を見ない、初期 Units から敵を除く、の 3 件は検出した。tactics.New を経由して表示を作る M3 は、1T の開始時点では外から見えない等価ミュータントで、コードを読んで呼んでいないことを確かめた（F-02）
- 変更した文書の表の列数・閉じの区切り・表の中の空行を確かめた（REV-031、REV-033）。traceability で変わったのは担当の 1 行だけ。main（#86）の取り込みは自動で済み、tccheck と core のテストが通った

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | test-spec-it04 の節の番号「3.5」が PR #87 と重なり、追記する場所も同じ（低） | 設計どおりへの修正 | 後からマージする PR #87 で 3.6 に番号を振り直す |
| F-02 | tactics.New を経由する実装（M3）は、core のテストで見分けられない（情報） | 設計の範囲内 | コードを読んで確かめた。seed を取らないことは #79 の TC-FR-CMP-11-10 で確かめる |
| F-03 | TC-01 にある配置表示の独立は、TC-08 で確かめている（情報） | 設計の範囲内 | 対応は不要 |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| P-01 | 失敗時はゼロ値の BoardView を返す | 設計の範囲内 |
| P-02 | 共用する組み立てが結果反映用の名簿の写しも返し、配置表示では捨てる | 設計の範囲内 |
| P-03 | 配置表示の誤りの文面を StartBattleAt と同じにする | 設計の範囲内 |

## 申し送り

- #79（Issue 4）: UI の HP だけの敵と地形の変換の写しを DeploymentView に置き換える（REV-040 G-01）。取得に失敗したら古い範囲を消し、表示で seed を取らないことを TC-FR-CMP-11-10 で確かめる（5.1 節）
- #81（PR #87）: 配置用の参照値に予兆を含めない契約を保つ。節の番号が重なったら振り直す（F-01）

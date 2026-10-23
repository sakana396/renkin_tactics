# REV-044: tactics の盤面参照値と履歴プレビューのレビュー

- 対象: PR #85、ブランチ `it04/76-boardview`（a3dae8f、指摘の修正 7a9d6e3）。Issue #76（DES-IT04 9 章の Issue 1）。`internal/core/tactics/boardview.go`（新規）・`battle.go`・`move.go`、`boardview_test.go`（新規）・`move_test.go`、`docs/04_test/test-spec-it04.md`・`test-spec-it01.md`、`docs/traceability.md` の 5 行
- 日付: 2026-10-23
- 作成: Codex（実装・テスト・文書の更新、指摘の修正。管理役が `codex exec` で頼んだ）。コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（`docs/00_plan/`〜`docs/03_design/`・`docs/adr/` を変更していない）
- 結果: F-01・F-02・F-04 を直した。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功。Linux の `go vet`・`go test` は、X11 がないため ui・cmd を除いて成功。`go run ./tools/tccheck` は整合 OK（警告 45 件。#75 の 48 件から TC-FR-BTL-10-21〜23 の 3 件が減った）
- BoardView・NewBoardView・Battle.BoardView・PreviewRewind と、移動・射程の共通処理を DES-RWD-05（2.2・4.1 節）と照らした。tactics は World・defs を知らない。閲覧は restore・Rewind・PCG に触れない。失敗時はゼロ値と ErrInvalidRewind・ErrInvalidSetup を返し、状態を変えない。0 回でも閲覧でき、Rewind 本体は ErrNoRewinds を先に返す
- New は NewBoardView を通るので、検証と UnitSetup の変換を戦闘の初期化と共用している（2.2 節）
- テストの期待値は、行動前の snapshot、同じ ID で Rewind した後の値、同じ入力を続けた対照戦闘から作っている。HistoryPlayerStart の行は、4.1 節の「行ラベルは 2、実際の値は 1T 敵軍・再開待ち」から導いている
- 既存の TC-FR-BTL-01-01・03〜08、TC-FR-MAP-01-01〜05 は期待値を変えず、Battle・現在の BoardView・NewBoardView で同じ確認をする形にした（8.3 節）
- ミュータント検査を別の写しで 9 件行った。7 件は検出した。M5（未実行の ID で continue する）は等価ミュータント。M7（`BoardView.Units()` が内部をそのまま返す）と M2（現在の参照値の再開待ちを常に偽にする）は残った（F-01・F-02）。修正のあと、実装者が両方のミュータントでテストが失敗することを確かめた
- 変更した文書の表の列数・閉じの区切り・表の中の空行を確かめた（REV-031、REV-033）。traceability で変わったのは担当の 5 行だけ

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | TC-FR-BTL-10-22 は、期待値を同じ `v.Units()` から取っている。`Units()` が内部を返す実装でもテストが通り、返した配列の独立（2.2 節）を確かめられていない（中） | 設計どおりへの修正 | 期待値を行動前の深い写しから作るようにした。M7 で失敗することを確かめた（7a9d6e3） |
| F-02 | `Battle.BoardView()` の NeedsResume が真になる場合をテストしていない（低） | 設計どおりへの修正 | 巻き戻した直後に、真で `b.NeedsResume()` と一致することを確かめるようにした。M2 で失敗することを確かめた（7a9d6e3） |
| F-03 | PreviewRewind の返す内容に、4.1 節の「予兆の領域」がない（情報） | 設計の範囲内 | 9 章で予兆の抽出は Issue 6（#81）の担当。申し送りに残す |
| F-04 | テスト仕様書 3.4 節に、これからの作業（管理役への引き継ぎ）が書かれている（低） | 設計どおりへの修正 | 事実だけにし、ミュータント 2 件の結果を足した（7a9d6e3） |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| P-01 | BoardView の Turn・Phase・Result を公開フィールドにし、再開待ちは NeedsResume() で返す | 設計の範囲内 |
| P-02 | NewBoardView は Turn 1・自軍・Ongoing・再開待ちなしを返す | 設計の範囲内 |
| P-03 | ゼロ値の BoardView も安全に参照できる | 設計の範囲内 |
| P-04 | TC-22・23 に、NewBoardView の独立・初期化と不正 Setup の確かめを足す | 設計の範囲内（配置専用の TC-FR-CMP-11-08・09 は #78） |

## 申し送り

- #77（Issue 2）: PreviewRewind で選んだ ID の before と、実際の Turn・Phase を表示する。取得に失敗したら古い盤面を見せない（4.1・4.2 節）
- #78（Issue 3）: Campaign.DeploymentView は、内部の Setup の組み立てを StartBattleAt と共用し、NewBoardView へ渡す（5 章）
- #81（Issue 6）: 現在と履歴の参照値に予兆の領域を足す（4.1 節、6.3 節）

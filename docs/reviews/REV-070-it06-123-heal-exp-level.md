# REV-070: 杖の回復 ＋ 魔力、撃破の経験のレベル差、敵のレベル（#123）のレビュー

- 対象: ブランチ `it06/123-heal-exp-level`（2763f86。main d1de7b7 から）。PR #138。Issue #123（DES-IT06 v0.2 14 章の仮番号 4）。`internal/core/tactics`（`heal.go`・`growth.go`・`battle.go`・`types.go`）、`internal/core`（`battle.go`）、`internal/defs`、`internal/defload`、`data/base/maps.toml`、`internal/ui` のテスト、`docs/04_test/`、`docs/traceability.md`、`docs/README.md`
- 日付: 2026-10-31
- 作成: Codex（途中までの実装）、Claude Code のサブエージェント（仕上げ）
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 要求・設計・企画の文書と ADR は変更していない。試し遊び 2 件の一時停止は開発者が了承した（F-01）
- 結果: 修正すれば承認。F-01 は開発者が了承した。F-02 は見送り。設計への提案 P-01〜P-05 は設計の範囲内

## 検証

- `gofmt -l .` は出力なし。`go vet`・`go test`（internal/ui・cmd/renkin 以外）は成功（試し遊びのスキップ 9 件。#120 の 7 件 ＋ 今回の 2 件）。internal/ui・cmd/renkin は `GOOS=windows go vet ./...` と `GOOS=windows go test -c` で確かめた（WSL に X11 のヘッダがないため。実行は CI）。`go run ./tools/tccheck` は整合 OK（警告は手動・未実装のケースだけ）。`python3 docs/03_design/scripts/verify_des06.py` は 700 項目成功。CI（2763f86）は成功
- 本体の maps.toml: 9 マップの初めの敵・ターンの増援の全員に `level` があり、ボスは 7.3 節の「今のボスの Lv の案」、一般の敵と増援は「Lv の案」の範囲に入ることを確かめた
- 一時停止した 2 件は、スキップを外すと本当に失敗する（ch04 は 13 ターン目、ch07 は 20 ターン目に主人公が倒れる）ことを確かめた
- 期待値: TC-FR-BTL-03-19 は例 H1、TC-FR-UNT-04-19 は例 G1〜G4 と式から導けることを確かめた。予測と実際の回復は同じ関数を通り、テストで突き合わせている
- ミュータント検査: 実装者の 21 件（TST-IT06 3.3 節）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | 試し遊び 2 件（TC-FR-MAP-04-04 ch04/作戦0/低め/seed1、TC-FR-MAP-04-06 ch07/谷の近道/平均/seed4）の一時停止を、まだ誰も了承していない（#120 では E-95 で開発者が了承した） | 開発者の確認 | 2026-10-31、選択式の質問で開発者が「止めてマージしてよい」と答えた（ch04 の 1 件は #131、ch07 の 1 件は #132 で作り直す。一般の敵の level は #130〜#133 で決め直す前提）。TST-IT06 3.3 節の「承認を待つ」を了承済みに直した |
| F-02 | 敵の Level 21 を拒むときの誤りの文が「0〜20」になる（低） | 設計どおりへの修正（任意） | 見送り。拒むこと自体は正しい。敵の能力を見直す #130〜#133 か、次に validateGrowth を触る Issue で直す |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| P-01 | 一般の敵と増援の level を、7.3 節の範囲の下から定義の順に回して付ける | 設計の範囲内（#130〜#133 で能力とともに見直す） |
| P-02 | 試し遊び 2 件を、サブテストだけ `t.Skip` で止める | 方法は設計の範囲内。了承は F-01 |
| P-03 | `StaffHealCap` を、敵の入力にも写す | 設計の範囲内 |
| P-04 | tactics で、負の `StaffHealCap` を拒む | 設計の範囲内 |
| P-05 | 敵のレベルの表示（12.2 節）を、画面の Issue に回す | 設計の範囲内 |

## 申し送り

- #125: 錬金術師の兵種に `staff_heal_cap = 8` を書き、例 H1 のノエルの 18/24 を本体のデータで確かめる
- 画面の Issue: 情報欄に敵のレベルを出す（DES-IT06 12.2 節、UC v0.10）
- 出来事の増援を本体のデータに足す Issue: `[[maps.events.enemies]]` にも `level` を書く（TC-FR-SYS-03-12 が確かめる）
- #131・#132: 今回止めた 2 件を新しい検査に置き換える。一般の敵の level は機械的に付けた初めの案で、今は能力と結び付いていない。能力とともに見直す
- #121・#124 との取り込み: `UnitSetup`・`Unit`・`newUnit`・`allyUnit`・`enemyUnit` の `StaffHealCap`・`Level` の欄を両方残す

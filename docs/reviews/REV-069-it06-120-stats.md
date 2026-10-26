# REV-069: 能力 3 つと、必殺・重さ・魔法・特効・2 回攻撃の計算と予測（#120）のレビュー

- 対象: ブランチ `it06/120-stats-formula`（855d59e と、main（#126）の取り込み 47c9506。指摘の修正は本記録と同じコミット）。PR #137。Issue #120（DES-IT06 v0.2 14 章の仮番号 1）。`internal/core/tactics`、`internal/core`（`battle.go`・`campaign.go`・`save.go`）、`internal/defs`、`internal/defload`、`internal/ui`、`data/base`、`docs/04_test/`、`docs/traceability.md`
- 日付: 2026-10-26
- 作成: Codex（実装）、Claude Code のサブエージェント（取り込み・仕上げ・指摘の修正）
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書と ADR を変更していない）
- 結果: F-01〜F-06 を直した。F-07 は開発者の確認待ち。設計への提案 P-01〜P-04 は設計の範囲内、P-05 は範囲の先取りで F-01 で外した

## 検証

- `gofmt -l .` は出力なし。`go vet`・`go test -count=1`（internal/ui・cmd/renkin 以外）は成功（試し遊びのスキップ 7 件）。internal/ui・cmd/renkin は `GOOS=windows go vet ./...` と `GOOS=windows go test -c` で確かめ、internal/ui のテストの実行ファイルは WSL から実行して全件成功した（WSL に X11 のヘッダがないため。cmd/renkin の実行は CI）。`go run ./tools/tccheck` は整合 OK。`python3 docs/03_design/scripts/verify_des06.py` は 700 項目成功
- ミュータント検査: Codex の 5 件、取り込みの 3 件、指摘の修正の 7 件をすべて検出し、元に戻した（TST-IT06 3.2 節）

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | 「武器 ＋ 道具 ≦ 5」の検証と版 1 のあふれの移し替え（高）。DES-IT06 6.3 節の移し替えは #122 の範囲で、#120 で先に入れたため、TakeItem で武器 1 本 ＋ 回復薬 5 個にしたセーブが読めない | 範囲の修正（案 A） | 版 1 は初めの武器 1 本を足すだけ、版 2 の読み込みは道具 5 個以下の今の検査に戻した。TC-FR-SYS-01-09 の「4 個 ＋ 在庫 1」は TC-FR-SYS-01-10（#122、自動・未実装）に移し、TakeItem → Save → LoadCampaign の往復のテストを足した |
| F-02 | 章の結果の伸び（中）。`statsDifference` に幸運・魔力・魔防がない | 設計どおりへの修正 | 3 つを足し、FinishChapter の Gains を確かめるサブテストを足した（TC-FR-UNT-05-05） |
| F-03 | 予測の重さと特効（中）。DES-IT06 12.2 節・UC-03 v0.10 の「重さで落ちた速さ」「特効の効く威力」が出ない | 設計どおりへの修正 | `ForecastSide` に `WeightLoss`・`EffectiveMight` を足し、表に「速さ 11（重さ −1）」「特効 威力 12」の行を出した。ui のモデルのテストで確かめた（TC-FR-BTL-08-25、TC-FR-UNT-09-12） |
| F-04 | 定義の検証の抜け（中）。種類 magic で物理の攻撃、特効の分類の重なり、兵種の tags の未知・重なりを受理する | 設計どおりへの修正 | defload と tactics の両方で拒むようにし、TC-FR-SYS-04-42 の表に行を足した |
| F-05 | 武器の列の検証（低）。`tactics.New` が `UnitSetup.Weapons` の要素を確かめない | 設計どおりへの修正 | `validateUnit` で各要素を `validateWeapon` に通す。テストを足した（TC-FR-SYS-04-42） |
| F-06 | 勇者の斧と追撃（低）。予測と実際の一致を確かめるケースがない | テストの追加 | 同じ盤面で Forecast の回数 4 / 1・ダメージ・必殺率と、Act の撃の順 A,A,D,A,A・ダメージを比べた（TC-FR-BTL-06-08、TC-FR-BTL-08-25） |
| F-07 | 今の章を新しい式で遊べるかの確認と、試し遊び 7 件の一時停止の了承（Issue の完了条件） | 開発者の確認 | 開発者の確認待ち（管理役が記入） |

## 設計への提案の判定

| # | 提案 | 判定 |
|---|---|---|
| P-01〜P-03 | spec-reviewer の報告のとおり | 設計の範囲内 |
| P-04 | spec-reviewer の報告のとおり | 設計の範囲内（#122 でそろえる） |
| P-05 | 版 1 の持ち物のあふれの移し替えと 5 枠の検証を #120 で先に入れる | 範囲の先取り。F-01 で外した |

## 申し送り

- #121: 5 枠を TakeItem で武器も数える。Weapon と Weapons の両方の指定を拒む。prepareUnit を Weapons[0] から作る。版 2 に武器 1 本 ＋ 道具 5 個のセーブがありうるので、あふれを在庫へ戻す
- #122: 版 1・版 2 のあふれの処理（TC-FR-SYS-01-10）、兵種の分類の本体データ、在庫の形
- #123: 出来事の増援の level
- #131〜#133: 一時停止した試し遊び 7 件を新しい検査に置き換える。ch09 の「二手に分かれる・低め」は seed 1〜12 で 0 勝だった

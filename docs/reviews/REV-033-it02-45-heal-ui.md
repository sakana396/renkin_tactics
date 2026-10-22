# REV-033: 回復の操作と予測、情報欄の杖（UI）のレビュー

- 対象: PR #58、ブランチ `it02/45-heal-ui`（bf0da55、修正 4b42c25）。`internal/ui/input.go`・`info.go`・`game.go`・`text.go`・`tiles.go`・`style.go`・`assets/assets.go`・`assets/README.md` の変更、テスト（`heal_test.go`、`info_view_test.go` の TC-FR-UNT-09-11）、`docs/04_test/test-spec-it01.md`・`test-spec-it02.md`、`docs/traceability.md` の担当の行。Issue #45（設計書 v0.3 の 9 章の Issue 11）
- 日付: 2026-10-22
- 作成: Codex（実装と修正。管理役が `codex exec` で頼んだ）。main の取り込みと衝突の解決（test-spec-it01.md の TC-FR-UNT-09-11・12、traceability.md の FR-UNT-09）・コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）。修正の確認は管理役（Claude Code）
- 承認: 不要（要求・設計・企画の文書と ADR は変更していない）
- 結果: 修正を確かめ、マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功
- `go test` は core・tactics・defload・savefile・tccheck で成功。`internal/ui` は `GOOS=windows go test -c` でコンパイルできることを確かめ、実行は CI で確かめた（`gh pr checks 58`）
- `go run ./tools/tccheck` は整合 OK（警告 23 件は既存の手動のケース）
- ui は golang.org/x を直接 import していない（REV-030 F-01）。何もしない関数をテストで確かめる箇所はない（REV-030 F-02）
- DES-UI-18 を確かめた。杖のユニットには「攻撃」を出さない。回復の相手は core の `HealForecast` の検証で選ぶ（使用回数・射程・HP が最大・自分・敵）。一度目のクリックで予測、二度目で `Act(Heal)`、右クリック・Esc で `Moved` に戻る。予測は「回復」（緑）・「リナ → レオン」・「HP 12 → 19」・「回復 +7」・「杖 残り 15 → 14/15」で、攻撃の表と帯は出さない。盤面に緑の「+7」を出す。杖の範囲は緑で、敵の範囲は赤のまま。杖を使い切ると「回復」を出さず、案内に `ErrNoUses` の理由を出す（UC-02 の 5c）
- 情報欄の杖（DES-UI-06）を確かめた。「癒やしの杖（杖）」と「回復 10　射程 1　残り n/15」、杖（Tiny Dungeon 125）と僧侶（Tiny Dungeon 108）の絵を出す。#42 の「Lv・経験」の行も残っている
- 9 章の「書き換える反復 1 のテスト」には Issue 11 の行がない。TC-FR-UNT-09-11 は ID を変えずに杖を足した。TC-FR-UNT-09-12 は main（#42・#46）の版のまま
- 回復のあとの Exp・LevelUp は #42 の再生（DES-UI-16）で出る

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | `traceability.md` の FR-UNT-09 の行で、取り込みのときに状態の列の説明と閉じの ` \|` が消えている | 設計どおりへの修正 | 直した（4b42c25）。「実装済み（#13、#27、#28、#46、#42、#45。手動のケースは開発者が確かめる）」に戻した |
| F-02 | `test-spec-it02.md` の Issue 案 11 の表と `assets/README.md` の武器の表で、見出しと行の間に空行があり、表として表示されない（REV-031 F-01 と同じ） | 設計どおりへの修正 | 直した（4b42c25）。2 か所の空行を消した |
| F-03 | `HealTargeting` で、選んだ味方の枠が他の候補と同じ緑で、`Targeting` のように選んだ相手を示さない | 設計どおりへの修正 | 直した（4b42c25）。`targetFrameColor` で、攻撃・回復のどちらでも選んだ相手をカーソルの色にする。TC-IF-01-36 で確かめる |
| I-01 | 緑の範囲・枠・「+7」と、僧侶・杖の絵の見分けやすさを確かめる手動のテストケースがない | 情報 | 申し送りにする |

## 設計への提案の判定

なし（`HealTargeting` を `Over + 1` で足したこと、使い切りの理由を案内欄に出すこと、`HealTargeting` でも危険範囲の操作の行を出すことは、設計の範囲内）

## 申し送り

- 実際の描画（緑の範囲と枠、盤面の「+7」、僧侶と杖の絵の見分けやすさ、回復のあとの経験・レベルアップの再生）は、開発者が Windows で確かめる
- 反復 2 の振り返り: 衝突を解くときに表の行を壊した（F-01）、表の中に空行を入れた（REV-031 F-01、本 F-02）。共有の文書の取り込みの確かめ方を見直す

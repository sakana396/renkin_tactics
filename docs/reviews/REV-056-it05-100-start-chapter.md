# REV-056: 開発用の開始章の指定（-chapter）のレビュー

- 対象: PR #107、ブランチ `it05/100-start-chapter`（d768a0c、main の取り込みと指摘の修正 aa54d2a）。Issue #100（DES-IT05 v0.3 9 章の Issue 3）。`cmd/renkin/main.go`・`main_test.go`、`internal/core/campaign.go`・`start_chapter_test.go`（新規）、`internal/ui/scenes.go`・`start_chapter_test.go`（新規）、`docs/04_test/test-spec-it01.md`（TC-FR-SYS-04-23、TC-IF-01-17）、`docs/04_test/test-spec-it05.md`、`docs/traceability.md`（FR-CMP-01、FR-SYS-04、IF-01）
- 日付: 2026-10-24
- 作成: Codex（管理役が `codex exec` で頼んだ）。main の取り込み・指摘の修正・コミット・プルリクエストは Claude Code
- レビュー: spec-reviewer（仕様適合レビュー）
- 承認: 不要（要求・設計・企画の文書を変更していない）
- 結果: F-01・F-02 を直した。マージしてよい

## 検証

- `gofmt -l .` は出力なし。`GOOS=windows go vet ./...` は成功。Linux の `go test` は ui・cmd を除いて成功し、ui・cmd は `GOOS=windows go test -c` でコンパイルを確かめた。CI は成功（ui・cmd のテストを含む）。`go run ./tools/tccheck` は整合 OK（警告 32 件、変更前と同じ）
- DES-CMD-03 と照らした。`-chapter` だけ、空の `-savedir`、知らない章 ID・空の章 ID（章 ID の一覧つき）、値のないフラグは、どれも画面を動かさず終了コード 2。未指定では `StartChapter` が空のまま。8.3 節の Issue 3 の行のとおり、TC-FR-SYS-04-23 と TC-IF-01-17 の期待結果も直した
- NewCampaignAt は、指定した章までの joins を既存の重複の検査を通して加え（加入は二重にならない）、前の章までの報酬を加える。能力は初期値で、経験は 0。期待値は 5 章の文と定義データから求めている。不正な指定では nil を返す。UI は指定した章の章前の物語から始め、その章の加入だけを表示して 1 回保存する。続きからは保存した章を使う
- ミュータント検査（レビューで実施）: 「指定した章の joins だけ入れる」「知らない ID を最初の章にする」「指定した章の報酬も入れる」の 3 件を、TC-FR-CMP-01-06 がすべて検出した。実装者も別の 3 件を検出している
- main（#99、PR #108）を取り込んだあと、core・defload のテストと tccheck が通ることを確かめた

## 指摘と対応

| # | 指摘 | 区分 | 対応 |
|---|---|---|---|
| F-01 | test-spec-it05.md と traceability の FR-CMP-01・IF-01 の行が #106 と衝突する（中） | 設計どおりへの修正 | 先に #99（PR #108）を取り込み、test-spec-it05.md を 2.1 節（#99）・2.2 節（#100）・3 章の検証記録の小節にまとめた（aa54d2a）。#106 はこの形を基にして取り込む |
| F-02 | FR-SYS-04 の行に #100（TC-23 の改訂）がない。FR-CMP-01 の行のパッケージに `internal/ui` がない（低） | 設計どおりへの修正 | 両方を足した（aa54d2a） |
| F-03 | 「ch01 の指定は NewCampaign と同じ」は、NewCampaign が NewCampaignAt を呼ぶので自分自身との比較になる。ただし具体的な値と既存のテストで担保されている（情報） | 設計の範囲内 | 直さない |
| F-04 | コミットの要約が「feat: feat:」と重なっている（情報） | 設計の範囲内 | 管理役の手順スクリプトを直し、次から重ねない |

## 設計への提案の判定

| # | 提案 | 判定 | 理由 |
|---|---|---|---|
| P-01 | `-savedir` の検査を定義データを読む前に、知らない章 ID の検査を読んだ後にする | 設計の範囲内 | 設計は順番を決めていない。どちらも終了コード 2 |
| P-02 | NewCampaign を NewCampaignAt（最初の章）への委譲にする | 設計の範囲内 | 5 章の「同じ結果になる」のとおり |
| P-03 | TC-IF-01-37 の ch09 は、試験用の 4 章目として足す | 設計の範囲内 | 本体の ch09 は #102 の担当 |
| P-04 | 章が 1 つもない目録には、専用の誤りを返す | 設計の範囲内 | 誤りの文面が変わるだけ |

## 申し送り

- #102 で、TC-FR-CMP-01-06 に本体の ch09 の場合を足す

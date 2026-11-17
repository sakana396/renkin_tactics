# REV-117: #205 試作の画像と音の取り込み（B-2）

- 日付: 2026-11-15（初回レビュー）、2026-11-18（対応）
- 対象: Issue #205、PR #219、ブランチ `it08/205-media`（初回 `28e7af4`）
- 方針・統合・記録: 管理役（Claude Code）
- 実装: Codex（3 回と、指摘の手直し 1 回）。コミット・push・PR の作成、Windows 向けの実行ファイルでの internal/ui・internal/ui/audio のテストの実行、`tools/shot` の撮影は管理役
- 独立レビュー: Claude spec-reviewer（Opus、読み取り専用）
- 根拠: IT08-PLAN v0.3（3.2 節 B-2、5 章 RV-2・RV-3・D-13、9 章）、DES-NG02 v0.4・DES-NG03 v0.3（承認 E-143）、TST-NG02・TST-NG03、SRS v0.9 の FR-SYS-07〜09、DES-IT01 1.1 節と AGENTS.md のパッケージ表、E-135、E-142、E-144、E-145
- 結論: 承認（再確認、N-01〜N-04 の対応のあと）。初回は修正すれば承認

## 独立レビューで確かめたこと（`28e7af4`）

- 試作と比べた: `data/alchemy/media` の 32 ファイルと `ASSETS.md`、`data/media_ledger.go`・`alchemy_media.go`、`internal/ui/audio`、`internal/ui/sound_ng03*.go`、`art_shrink.go`、`media_art_ng02_test.go`、`tools/synth`・`tools/mkplaceholder` は、試作とバイト単位で同じ。画面のファイルは、48 px（`HiRes`・`sceneCanvasWidth`）と準備の画面の作り直しの部分を除いて取り込んだ
- 例外コミット: 4c99918（名簿の行の顔 24 px・加入の場面の全身 48 px・`Shrunk`）と 209dc67（剣を持つ野盗の絵と台帳の行）が入っている
- DES-NG-07: `cmd/renkin` と `tools/shot` は `-game alchemy` のときだけ `data.AlchemyMedia()` を渡す。media が nil なら、画像は読みにいかず、音は組み立てない
- 旧作の描き方: media が nil のときは前と同じタイルを描き、背景・肖像は空で、物語の本文は前と同じ x = 48、幅 736 で折り返す（TC-DES-NG-08-83）
- 台帳（DES-NG-08 5 章）: 32 行と 32 ファイルが一致し、検査の 1〜7 が実装されている
- 音（DES-NG-09）: 定数（効果音 0.7、BGM 0.3、1 フレームに 3 つ、6 フレームの間隔、同時に 8 つ）、出来事と場面への対応、ミュートの N が設計と合う。WAV の合計は 4,041,146 バイト（5 MB 以下）
- 期待値: TC-DES-NG-08-73・95・97・98 は設計書の数値から手で導けた
- 依存の向き: `ui/audio` が使うのは oto と `ebiten/v2/audio/wav` だけ。`golang.org/x/image` を直接使うのは `tools/mkplaceholder` だけ。`core`・`core/tactics` は変わっていない（ADR-0002）
- RV-3（E-144）: CI に `libasound2-dev` を足し、README に書いた
- 検証（レビュー担当）: `gofmt -l .` 出力なし。`GOOS=windows go vet ./...` 成功。Linux の `go vet`・`go test` は、ui・cmd・mkplaceholder・shot を除いて成功（除いた分は X11・ALSA のヘッダがないため）。Windows 向けの実行ファイルで `internal/ui`・`internal/ui/audio`・`tools/mkplaceholder` が PASS。tccheck は OK。CI（`28e7af4`）は成功

## 指摘と対応

| ID | 重さ | 指摘 | 対応 |
|---|---|---|---|
| F-01 | 中 | CI の「文書だけの変更」の判定が、`data/` の下の `.md` をコードとして扱わない（DES-NG02 5 章。試作 `29385e5` が抜けた） | 試作と同じ条件とコメントを足した（Codex） |
| F-02 | 中 | `.gitignore` に `/placeholder-out/` がない（DES-NG02 6 章） | `.gitignore` を作り、`/placeholder-out/` と `*.orig` を入れた（Codex） |
| F-03 | 中 | 手動の TC-DES-NG-08-94 が未実施 | 管理役が 2026-11-18 に行った。`tools/mkplaceholder` の仮の背景（物語・タイトル）だけを `media/art` に写し（本物のユニットの絵・肖像は上書きしない）、`tools/shot -game alchemy` でタイトル（背景）、1 章の物語 9 段落目（アオイの肖像が左、本文が右に寄る）、準備の名簿（顔の絵 24 px）、ユニットの詳しい欄（肖像）、戦闘の盤面と情報欄を撮って確かめた。撮影のあと仮の絵を消し、`git status` に残らないことを確かめた。絵がないときの見た目（物語は背景なし・本文が左から）と旧作の盤面は、初回の撮影で確かめた。人の目での確認は #214 |
| F-04 | 人の判断 | 仮の音を開発者が聞いていない（D-13、TC-DES-NG-09-26） | 開発者が「#214 の試遊で聞く」と決めた（E-145）。TC-DES-NG-09-26 は手動・未実施のまま #214 に持ち越し、台帳の「確かめた人」もそのとき書き換える。#214 の Issue に書き足した |
| F-05 | 人の判断 | E-144 の要約の確認と `docs/01_concept` の変更の承認。野盗の絵は 2026-11-14 | 要約の結論の欄を「2026-11-13（野盗の絵は 11-14）」に直し、開発者が「よい」と確かめた（E-145）。問いの文の「17 枚」は書き起こしなのでそのまま |
| F-06 | 低 | 範囲外のもの（`terrainByName`・`ebitenStretch`・`hiresPrepare`・台帳の `terrain/` と TC-DES-NG-10-27）が混ざっている | 外した（Codex）。#206・#207 で取り込む |
| F-07 | 低 | traceability の FR-SYS-07〜09 と TST-NG02・NG03 の段落・版・変更履歴が古い | 直した（TST-NG02 v0.5、TST-NG03 v0.4。Codex と管理役） |
| F-08 | 低 | PR の説明に試作の元のコミット・REV-108・型の見出しがない | PR の説明を書き直した |
| F-09 | 情報 | 絵の「利用条件」の条項の番号と版が未記入。追記を受け持つ Issue がない | #210 の Issue に「ChatGPT で作った絵の利用条件の条項を台帳に追記する」を書き足した |
| F-10 | 情報 | DES-NG02・NG03・IT08-PLAN の RV-2・RV-3 が「未決」の文面のまま | 後の文書の PR（振り返りなど）でまとめて直す（設計書の文面の変更なので承認が要る） |
| F-11 | 情報 | `cmd/renkin` が新作のときだけ `Media` を渡すことのテストがない | TC-DES-NG-08-29 を足した（Codex）。管理役とレビュー担当が Windows 向けの実行ファイルで確かめた（PASS） |
| F-12 | 情報 | README に、実行にも `libasound2` が要ることが書いていない | 1 文足した（Codex） |

## 実装の判断（設計の範囲内）

- 効果音の入力は、`update` の前の場面で読む（物語の送りの Enter に合う）
- `drawUnitArt` を `art_unit_draw.go` に分けた（描く範囲は同じ）
- TC-DES-NG-08-99 の高解像度の分は #206 に回した
- 情報欄の `InfoIcon.Art` は、media のあるなしで切り替えない（`ArtStore` が「なし」を返す）

## 開発者の承認

- E-144（RV-2・RV-3 の決定）はこの PR で `docs/01_concept/elicitation-notes.md` に書き起こした。E-144 の要約は 2026-11-18 に開発者が「よい」と確かめた
- その確認と「仮の音は #214 で聞く」の答えは E-145 として #208 のブランチに書き起こした（main に入るのは #208 のマージのとき。E-145 の要約の確認は #208 の PR で行う）

## 再確認の結果（2026-11-18、`0069f1e`。`git diff 28e7af4 0069f1e`）

- F-01: `ci.yml` の判定に `^data/.*\.md$` を足した。試作 `29385e5` と同じ
- F-02: `.gitignore` に `/placeholder-out/` と `*.orig` を入れた
- F-03: 管理役の撮影を確かめた。タイトルの背景と暗い膜（`t1.png`）。1 章の物語 9 段落目で、アオイの肖像が箱（x = 48、y = 120、224 px）に入り、本文が x = 296 から右の余白に収まり、背景も出ている（`s9.png`）。準備の名簿 4 行の顔の絵が行の高さ 24 px に収まり、隣の行と重ならない（`p1.png`）。詳しい欄の 64 px の肖像（`d1.png`）。絵がないときと旧作の画面は、初回の `m3.png`・`old.png` で確かめた
- F-04: TC-DES-NG-09-26 は #214 に持ち越した（E-145）。#214 の Issue に、聞いたら台帳の音の 16 行の「確かめた人」を書き換えることまで書かれている
- F-05: E-144 の結論の欄が「2026-11-13（野盗の絵は 11-14）」になり、開発者が要約を確かめた（2026-11-18）。E-144 はまだ main に入っていないので、「すでにある E-xx は変えない」には当たらない
- F-06: `terrainByName`・`ebitenStretch`・`hiresPrepare`（Enter・↑↓）、台帳の `terrain/` の規則、TC-DES-NG-10-27 のテストを外した。`soundInputOf` は DES-NG-09 4.4 節の表どおり
- F-07〜F-12: 対応を確かめた。TC-DES-NG-08-29 は、base では `Config.Media` が nil で、alchemy では nil でないことを確かめる。base でも渡すように壊せば落ちる
- 検証（`0069f1e`、レビュー担当が実行）: `gofmt -l .` 出力なし。`GOOS=windows go vet ./...` 成功。Linux の `go vet`・`go test` は ui・cmd・mkplaceholder・shot を除いて成功。Windows 向けの実行ファイルで、`cmd/renkin` の `TestGameMedia`・`TestGameFlag`、`internal/ui` の画像・音・場面のテストが PASS。tccheck は OK（警告 113 件）

### 再確認で見つけた点と対応

| ID | 重さ | 内容 | 対応 |
|---|---|---|---|
| N-01 | 低 | `data/media_ledger.go` の誤りの文に `terrain/<地形の ID>.png` が残っている（F-06 の残り） | 消した。#206 で規則を戻すときに一緒に戻す |
| N-02 | 低 | TST-NG02 の TC-DES-NG-08-29 の行が空行のあとにあり表から外れている。節がテストの置き場所と合わない。0.5 の履歴の日付 | 2.1 節「作品ごとの素材（`cmd/renkin/game_ng06_test.go`）」を作って表に入れ、0.5 の日付を 2026-11-18 にした |
| N-03 | 低 | `docs/README.md` の TST-NG02 の行が「実画面の確認は手動・未実施」のまま | 「管理役の撮影で実施（人の目では #214）」にした |
| N-04 | 情報 | E-145 は #208 のブランチにある。「開発者の承認」の節の書き方と F-11 の対応の欄 | 「開発者の承認」の節を書き分け、F-11 を「確かめた（PASS）」にした |

- 結論（再確認）: 修正すれば承認。N-01〜N-04 を直したあと、最新のコミットの CI が通っていればマージしてよい（レビュー担当の再確認は不要、管理役が差分を確かめる）。管理役が直した差分を確かめた

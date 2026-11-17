# 画像の生成と台帳への記入（tools/imagegen）

手元の Codex CLI の画像生成（組み込みの `image_gen`。開発者の ChatGPT のサブスクリプションの枠で動く）で画像を作り、新作の画像の置き場 `data/alchemy/media/art/` に正しい名前・大きさで置き、台帳 `ASSETS.md` に 1 行足す道具（#210。IT08-PLAN 9 章 J、E-131・E-133）。非公式の道具は使わない。

名前・推奨の大きさ・台帳の書き方は DES-NG-08（[design-ng02-art.md](../../docs/03_design/design-ng02-art.md) 3 章・5 章）、絵柄と指示文の型は [CON-DRAFT-NG02](../../docs/01_concept/new-game-visual-draft.md) 5 章、キャラの指示文は [new-game-prompts.md](../../docs/01_concept/new-game-prompts.md) にある。

## 要るもの

- Codex CLI（`codex`。0.169.0 で確かめた）に ChatGPT のアカウントでログインしてあること。`codex features list` で `image_generation` が `true` であること
- 画像生成は、ChatGPT のプランの利用枠（Codex と共通）を使う。1 枚に 1 分ほどかかる。**作る枚数は、作るときに利用枠の残りを見て決める**（E-133）
- ツールの利用規約を、使う前に確かめて台帳に書く（CON-DRAFT-NG01 2.9 節）。確かめた結果は台帳の「利用条件」の欄にある

## 手順

### 1. 指示文を書く

英語の指示文を 1 つのファイルに書く。先頭に CON-DRAFT-NG02 5.1 節の共通のスタイルブロックを付ける。特定の作品名・作家名は入れない。台帳には、このファイルの全文がそのまま載る（`|` は表が崩れるので使えない）。

- 背景（`story`・`title`）: 横長（例: "wide landscape 3:2"）、人物なし、大事なものは横の中央に。画面（832×576）に合わせて左右が少し切れ、暗い膜が重なる
- ユニットの絵（`units`）: "plain flat solid bright-green background (for cutting out)"、全身が余白つきで入ること
- 肖像（`portraits`）: 正方形、"plain flat light-gray background"

### 2. 生成する（generate）

```sh
go run ./tools/imagegen generate -prompt <指示文.txt> -out <生の画像.png>
```

- 生の画像と、生成の記録 `<生の画像.png>.json`（Codex CLI の版、会話のモデル、スレッドの ID、日付、指示文の全文、Codex の保存先）を書く。`-out` は `data/alchemy/media` の外にする（中は拒否する。終了コード 2）。生の画像はリポジトリに入れない
- 中では次のように Codex CLI を呼ぶ。空の一時ディレクトリで、読み取り専用のサンドボックス、セッションを残さない設定で動かし、標準入力で「組み込みの image_gen を 1 回だけ使い、指示文を変えず、ファイルを作らず、保存した PNG の道筋だけを返す」よう頼む

  ```sh
  codex exec --skip-git-repo-check --ephemeral -s read-only -C <一時ディレクトリ> --json - < 依頼文
  ```

- Codex CLI は画像を `$CODEX_HOME/generated_images/<スレッドの ID>/`（`CODEX_HOME` がなければ `~/.codex`）に保存する。道具は JSONL の `thread.started` からスレッドの ID を取り、その中の PNG（1 つだけのはず）を `-out` に写す
- `-model` で Codex の会話のモデルを変えられる（省けば Codex の設定のまま）。画像のモデル名は Codex CLI が表示しないので、台帳には「表示されない」と書く
- 失敗したら（利用枠の上限など）、Codex CLI の誤りを出して終了コード 1

生の画像を開いて（Claude なら Read で）見る。指示に合わなければ、指示文を直して生成し直す。

### 3. 置いて台帳に書く（place）

```sh
go run ./tools/imagegen place -in <生の画像.png> -kind story -id ch01_before \
  -terms '<利用条件>' -checked-on 2026-11-18 -checked-by '<確かめた人>'
```

| `-kind` | 置き場 | 大きさ | 加工 |
|---|---|---|---|
| `units` | `art/units/<ID>.png`（敵は `enemy_<クラスの ID>`） | 128×128 | 緑の背景を透過にし、中身で切り出して足元をそろえた正方形にし、縮める |
| `portraits` | `art/portraits/<ID>.png` | 512×512 | 中央を正方形に切り、縮める |
| `story` | `art/story/<物語の ID>.png` | 832×576 | 中央を残して 832:576 に切り、縮める |
| `title` | `art/title.png`（`-id` は使わない） | 832×576 | 同上 |

- 透過（クロマキー）は、試作の台帳の `art/units/aoi.png` の行の手順に合わせた: 緑らしさ k = G − max(R,B) が 70 以下は不透明、170 以上は透明、その間は直線。α が 1 未満の画素と透明の画素から 6 px 以内の画素は G を max(R,B) に揃える。α > 0.04 の範囲で切り出し、6% の余白を足す。縮小は乗算済みアルファの Catmull-Rom（試作の Lanczos とは違う）。縮小のあと、α 64 以下で k が 8 以上の画素は透明にし、残る半透明の画素は G を max(R,B) に揃える
- `-chroma on|off` で透過の有無を変えられる（既定は `units` だけ透過）
- 台帳の行の「作り方」（加工の手順）、「使ったツールと版・生成した日」、「指示」（全文。改行は空白に）は、画像と生成の記録から組み立てる。「利用条件」「確かめた日」「確かめた人」は引数で渡す
- 同じ名前の画像か台帳の行がすでにあれば止まる。作り直すときは `-replace` を付けると、画像と台帳の行を置き換える

### 4. 確かめる

```sh
go test ./data                                   # 台帳の検査（DES-NG-08 5 章）
tools/shot/shot.sh -game alchemy -chapter ch01 -scene story -out <PNG>   # 画面に出るか
```

撮った PNG を開いて、絵が出ていること、文字が読めることを見る。

## 注意

- 最初の 1 枚（`art/story/ch01_before.png`、#210）は、この道具を作る前に、手で `codex exec` を呼んで生成した（引数と依頼文は generate と同じ）。生成の記録は道具で作ったものではなく、手で作った。置くのと台帳の記入は place で行った。道具の generate は、そのあと 2 回目の生成で動作を確かめた（その画像は採っていない）。台帳の行にも同じことを書いてある

- 生成した画像の権利・利用条件は、ツールの規約による。台帳の「利用条件」の欄に、規約の名前・版・条項と、確かめた日・人を書く。**これは法律の助言ではない。公開する前に、権利者である開発者が条件を確かめる**（CON-DRAFT-NG01 2.9 節）
- 既存の作品に似ていないかは、人の目で確かめる（CON-DRAFT-NG01 2.8 節）

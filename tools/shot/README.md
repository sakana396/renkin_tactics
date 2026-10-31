# 画面の PNG 撮影

WSL から `tools/shot/shot.sh` を実行する。Windows 版を一時ディレクトリにビルドして起動し、画面外の小さな窓で描いた論理画面を PNG に保存する。Windows の相互運用が有効な WSL と、Windows 側で動く Ebitengine の描画環境が必要。

```sh
tools/shot/shot.sh -chapter ch09 -scene battle -out /tmp/ch09-battle.png
tools/shot/shot.sh -chapter ch09 -scene battle -steps 'rewind' -out /tmp/ch09-rewind.png
tools/shot/shot.sh -chapter ch09 -scene battle -steps 'cursor:160:160' -out /tmp/ch09-info.png
```

PNG は常に 832×576 px。セーブファイルは読み書きせず、乱数の種は 1 に固定する。`-chapter` は ch01〜ch09、`-scene` は `title`、`rule`、`story`、`prepare`、`deploy`、`battle` を指定する。場面への移動では新規開始、物語の送り、出撃確定を自動で行う。

`-steps` には次の操作をカンマ区切りで左から順に書く。盤面の座標は左上を (0,0) とした章全体のマス座標。画面の座標は 832×576 のピクセル座標。

| 操作 | 内容 |
|---|---|
| `click:X:Y` | 戦闘のマスをクリック。自軍を選び、移動先を選び、攻撃相手を選ぶときに使う |
| `target:X:Y` | 相手のマスをクリック。もう一度同じ操作をすると攻撃を決定する |
| `cursor:X:Y` | カーソルを画面座標へ動かし、情報欄や敵の範囲を表示する |
| `clickpx:X:Y` | 画面座標をクリック。場面のボタンや巻き戻しの一覧に使う |
| `menu:N` | 戦闘中の一覧の N 行目を選ぶ（先頭は 0） |
| `action:attack`、`action:heal`、`action:item`、`action:wait` | 移動後の行動を選ぶ |
| `weapon:N` | 武器・杖の一覧の N 行目を選ぶ |
| `rewind`、`rewindkey:Up`、`rewindkey:Down` | 巻き戻しを開く・一覧を動かす |
| `scroll:X:Y` | 盤面のカメラを X・Y マス動かす |
| `tick:N` | 出来事の再生を N フレーム進める |
| `endphase`、`danger` | 番を終える・敵の危険範囲を切り替える |
| `choose:N`、`advance`、`cancel` | 場面の選択、物語の送り、取り消し |

たとえば、戦闘の画面で自軍を選んで移動し、攻撃予測を撮るときは、実際の章で到達可能なマスと相手を指定する。

たとえば操作列は `click:自軍X:自軍Y,click:移動先X:移動先Y,action:attack,target:敵X:敵Y` の順にする。敵の位置と移動できるマスは章の定義と現在の盤面に合わせる。

選択が成立しないときは、画面の状態に応じて操作が無視される。座標と操作順を見直す。`-out` の親ディレクトリは自動で作る。

# 新作の画像と音（メディア）

- `art/`: 画像（PNG）。置き場と名前の規則は [docs/03_design/design-ng02-art.md](../../../docs/03_design/design-ng02-art.md)（DES-NG-08）、作る画像の一覧と指示の型は [docs/01_concept/new-game-visual-draft.md](../../../docs/01_concept/new-game-visual-draft.md) にある。一辺が 4096 px を超える画像は使われない
- `audio/`: 音（WAV）。効果音と BGM

ここに置いたファイルは `go:embed` で本体に入る。**出典と利用条件を、同じ場所の `ASSETS.md` の表に必ず書く**（書いていないファイルはテストが落とす）。

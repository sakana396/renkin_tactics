# 新作のキャラの画像の指示文（組み立て済み）

- 文書 ID: CON-DRAFT-NG03
- 版: 0.1（**未承認**。2026-11-13、開発者の依頼でエージェントが作成）
- 入力: [見た目の方針書](new-game-visual-draft.md) 3 章・5 章、アオイの生成結果（ChatGPT、GPT-6 チャットモード）

## 使い方

1. ChatGPT に、**アオイの全身の絵（`art/units/aoi.png` の元の絵）を添付**してから、各指示文を貼る。1 キャラにつき 1 回の生成。肖像は、アオイの肖像を添付して貼る
2. ChatGPT は**1 回の生成で 1 枚しか出ない**ので、候補から選ぶのではなく、出た絵を見て判断する。気に入らなければ、同じ指示文（または手直しした指示文）で作り直す。**緑の背景のまま**（ユニットの絵）、または薄い灰色の背景のまま（肖像）で保存する。切り抜きは、こちらで行う。作り直したら、台帳に書く指示文は、最後に採用した 1 枚の分だけ
3. 画像を `data/alchemy/media/art/` に置くときは、`ASSETS.md` に **この指示文の全文**、モデル名、生成した日を書く（指示文に手を加えたら、加えた後の全文を書く）
4. 生成された絵に、指示にない要素が足されることがある（アオイでは眼鏡・イヤリングなど）。それは台帳に書く

## 作る順と、添付する画像（2026-11-13、開発者の指示: 肖像 → ユニット。ダインから作り直す）

**肖像を先に作り、その肖像を見本にして、ユニットの絵を作る。** 肖像を作るときに、同じ人物のユニットの絵を見本にすると、肖像が、ユニットの絵（デフォルメの頭身・ポーズ）に引っ張られた（ダインで確認。開発者の指摘）。そのため、肖像の見本は**アオイの肖像だけ**にする。

| 順 | 作るもの | 添付する画像 | 指示文の追加 |
|---|---|---|---|
| 1 | ダインの肖像 | アオイの肖像のみ（絵柄の見本） | なし |
| 2 | ダインのユニットの絵 | ①アオイのユニットの絵（絵柄・頭身の見本）、②ダインの肖像（人物の見本） | 下の「人物の見本」 |
| 3 | トーリの肖像 | アオイの肖像のみ | なし |
| 4 | トーリのユニットの絵 | ①アオイのユニットの絵、②トーリの肖像 | 下の「人物の見本」 |
| 5 | ネムの肖像 | アオイの肖像のみ | なし |
| 6 | ネムのユニットの絵 | ①アオイのユニットの絵、②ネムの肖像 | 下の「人物の見本」 |
| 7 | ハルの肖像 | アオイの肖像のみ | なし |
| 8 | ハルのユニットの絵 | ①アオイのユニットの絵、②ハルの肖像 | 下の「人物の見本」 |
| 9〜14 | 敵 6 種のユニットの絵（肖像は不要） | 新しい「ダインのユニットの絵」（絵柄・頭身の見本。敵は暖色を使わない） | なし |

**人物の見本**（**ユニットの絵を作るときだけ**、2 枚目の画像を人物の見本にする。参照画像の指示の次の行に貼る）:

```
The second attached image shows the same character in a portrait: keep exactly
the same face, hair, outfit, colors and accessories, and redraw the character
as a full-body chibi following the first image's proportions.
```

- すでに置いてあるダインとトーリのユニットの絵（`units/dain.png`、`units/tori.png`）は、新しい絵ができるまで、そのまま置いておく。新しい絵に置き換えるときに、台帳の行も書き直す
- 参照画像を 2 枚添付できないときは、①だけでよい（人物の見本は省く）

## 注意

- ユニットの絵の背景は**緑**で切り抜く。キャラの服や持ち物に**緑色を使わない**（切り抜きで削れる）。トーリのスカーフを緑から黄土色にしたのは、このため（方針書 3.1 節の「緑の布」を変更）
- 敵は**暖色（琥珀・銅・橙）を使わない**。光る淡い水色の線があるのが公国の兵、ないのが野盗
- 敵の `captain` の絵は、隊長ドルクと将軍ベルナド（ボス）が同じ絵を使う（ボスは盤面で金の枠）。個別の絵が要るときは別に決める（方針書 Q-V4）
- 画像の置き場の名前: 自軍 `units/<id>.png`・`portraits/<id>.png`、敵 `units/enemy_<クラス ID>.png`

## 自軍（4 人）

### ダイン（護衛の重装兵） — ユニットの絵（`units/dain.png`）

```
Use the first attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions.
The second attached image shows the same character in a portrait: keep exactly the same face, hair, outfit, colors and accessories, and redraw the character as a full-body chibi following the first image's proportions.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a broad-shouldered, sturdy man in his late 30s with a calm, dependable expression, short cropped hair and a light beard, wearing heavy iron-gray plate armor with an indigo cloth sash, copper rivets and buckles, holding a large tower shield and a spear.
```

### ダイン（護衛の重装兵） — 肖像（`portraits/dain.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading and color treatment, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Bust-up portrait, 3/4 view, calm expression, soft warm rim light, plain flat light-gray background, square 1:1.
a broad-shouldered, sturdy man in his late 30s with a calm, dependable expression, short cropped hair and a light beard, wearing heavy iron-gray plate armor with an indigo cloth sash, copper rivets and buckles, holding a large tower shield and a spear.
```

### トーリ（槍兵） — ユニットの絵（`units/tori.png`）

```
Use the first attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions.
The second attached image shows the same character in a portrait: keep exactly the same face, hair, outfit, colors and accessories, and redraw the character as a full-body chibi following the first image's proportions.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a lean, tall young man around 19 with short messy hair and an energetic expression, wearing brown leather armor with a muted ochre-yellow scarf, copper buckles, holding a long spear taller than himself.
```

### トーリ（槍兵） — 肖像（`portraits/tori.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading and color treatment, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Bust-up portrait, 3/4 view, calm expression, soft warm rim light, plain flat light-gray background, square 1:1.
a lean, tall young man around 19 with short messy hair and an energetic expression, wearing brown leather armor with a muted ochre-yellow scarf, copper buckles, holding a long spear taller than himself.
```

### ネム（弓兵） — ユニットの絵（`units/nemu.png`）

```
Use the first attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions.
The second attached image shows the same character in a portrait: keep exactly the same face, hair, outfit, colors and accessories, and redraw the character as a full-body chibi following the first image's proportions.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a tall, athletic woman in her mid-20s with a long braid over her shoulder and a calm, focused look, wearing a wide-brimmed hat and a sand-colored cloak with an aqua-teal cord and a copper clasp, carrying a longbow nearly as tall as her and a quiver.
```

### ネム（弓兵） — 肖像（`portraits/nemu.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading and color treatment, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Bust-up portrait, 3/4 view, calm expression, soft warm rim light, plain flat light-gray background, square 1:1.
a tall, athletic woman in her mid-20s with a long braid over her shoulder and a calm, focused look, wearing a wide-brimmed hat and a sand-colored cloak with an aqua-teal cord and a copper clasp, carrying a longbow nearly as tall as her and a quiver.
```

### ハル（調合の見習い・杖で回復） — ユニットの絵（`units/haru.png`）

```
Use the first attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions.
The second attached image shows the same character in a portrait: keep exactly the same face, hair, outfit, colors and accessories, and redraw the character as a full-body chibi following the first image's proportions.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a small teenage boy around 16 with untidy messy hair and a shy, cheerful expression, wearing an oversized near-white cream robe with very long sleeves and pale aqua trim, a copper ring on his belt, holding a short healing staff topped with an aqua-teal crystal and a small glass vial.
```

### ハル（調合の見習い・杖で回復） — 肖像（`portraits/haru.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading and color treatment, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated palette with amber/copper and aqua-teal as the only vivid accents, fantasy ancient stone city setting, no text, no letters, no watermark.
Bust-up portrait, 3/4 view, calm expression, soft warm rim light, plain flat light-gray background, square 1:1.
a small teenage boy around 16 with untidy messy hair and a shy, cheerful expression, wearing an oversized near-white cream robe with very long sleeves and pale aqua trim, a copper ring on his belt, holding a short healing staff topped with an aqua-teal crystal and a small glass vial.
```

## 敵（6 種。ユニットの絵のみ）

敵の肖像は、今は使わない（物語で話すのはドルクとベルナドだけ。必要になったら別に作る）。

### 野盗（`units/enemy_bandit.png`）— **作り直し済み（2026-11-14、武器を斧から剣に変えたため）**

データ上の野盗の武器は、斧から**剣**に変えた（試遊で、斧が槍の前衛に有利で、序盤が難しかったため）。今の絵は斧を持っているので、**剣を持つ姿で作り直す**。人物（顔・装備）は、今の絵と同じにしたいので、**①ダインのユニットの絵（絵柄の見本）、②今の野盗の絵（人物の見本。`art/units/enemy_bandit.png` の元の絵）を添付**し、次の指示文を貼る:

```
Use the first attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions.
The second attached image shows the same bandit holding an axe: keep exactly the same face, hair, outfit, colors and accessories, and redraw him as a full-body chibi following the first image's proportions, but replace the axe with a crude notched short sword held in the same hand.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a rough bandit in mismatched patched gear, ragged cloth and dirty brown and gray leather, wielding a crude notched short sword, unshaven scowling face, absolutely no glowing lines anywhere (his weapon is ordinary, not an ancient relic).
```

（下は、最初の指示文。斧を持つ版。）

### 野盗（`units/enemy_bandit.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a rough bandit in mismatched patched gear, ragged cloth and dirty brown and gray leather, wielding a crude notched sword, unshaven scowling face, absolutely no glowing lines anywhere (his weapon is ordinary, not an ancient relic).
```

### 公国の剣士（`units/enemy_swordsman.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a soldier of a duchy in scavenged ancient stone-plate armor bolted with iron, holding a sword with thin glowing pale-cyan lines running along the blade, iron-gray and black outfit, helmet with a visor, stern posture.
```

### 公国の槍兵（`units/enemy_spearman.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a soldier of a duchy in scavenged ancient stone-plate armor bolted with iron, holding a long spear with thin glowing pale-cyan lines along the shaft and head, iron-gray and black outfit, open-face helmet, alert posture.
```

### 公国の弓兵（`units/enemy_bowman.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
an archer of a duchy in light scavenged ancient stone-plate armor, holding a bow whose limbs and string glow with thin pale-cyan lines, iron-gray and black outfit, hooded, focused posture.
```

### 公国の重装兵（`units/enemy_heavy.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a bulky heavily armored soldier of a duchy in thick scavenged ancient stone-plate armor bolted with iron, carrying a huge slab-like shield with glowing pale-cyan lines and a short heavy weapon, iron-gray and black, full helmet, wide stance.
```

### 公国の隊長（敵のボスも同じ絵）（`units/enemy_captain.png`）

```
Use the attached image as a style reference: match its art style, line weight, shading, color treatment and chibi proportions, but draw a different character as described below.
2D hand-drawn anime-style illustration, soft cel shading with subtle texture, clean thin-to-medium outlines, slightly desaturated cold palette with iron-gray, black and pale-cyan glow as the only accents, no warm colors, fantasy ancient stone city setting, no text, no letters, no watermark.
Full-body chibi character, 2.5 heads tall, front three-quarter view, standing pose, centered, entire body visible with margin on all sides, plain flat solid bright-green background (for cutting out), square 1:1.
a commander of a duchy, a man in his 40s to 50s, large and imposing, with oversized shoulder plates of scavenged ancient stone and iron, a dark gray cloak with a cold blue-gray commander cloth, a long spear with strong glowing pale-cyan lines and the lines spreading across his armor, stern commanding look.
```


# 設計書（反復 2: 章立てと成長）

- 文書 ID: DES-IT02
- 版: 0.3（案。方針は E-32 で開発者が承認。数値などの細部は Claude Code の案で、レビュー（REV-021）を通したものをマージする）
- 作成日: 2026-10-21
- 対象: [反復 2 計画書](../iterations/it02/plan.md) の範囲
- 入力: [SRS-001](../02_requirements/srs.md) v0.1、[ユースケース](../02_requirements/use-cases.md) v0.4（案）、[設計書（反復 1）](design-it01.md) v0.3、[ADR-0002](../adr/0002-return-to-fushigi.md)、[ADR-0003](../adr/0003-definition-data-format.md)、CON-001 3〜5 章、E-07、E-08、E-25、E-26、E-29〜E-32

反復 1 の設計書（DES-IT01）に書き足すのではなく、反復 2 で**変える部分と足す部分**だけをこの文書に書く。ここに書いていないことは、反復 1 の設計書のとおりとする。反復 1 の設計要素を変えるときは、同じ ID を使い「（反復 2 で変更）」と書く。数の値は 10 章の仮置きで、遊んで調整する。

E-26 の「細かい点は任せる」に従い、決める点はすべて Claude Code の推奨の案で書いた。開発者がまとめて確かめる点（C-01〜C-10）は、計画書 9.2 節に一覧がある。本文では、その点に「（C-xx）」と印を付けた。

**v0.3 の改訂（E-29〜E-32）**: 開発者が反復 1 の版を遊び、「前衛が弱い」「後衛がほしい」「倒せる・倒される戦闘を予測で目立たせたい」と分かった。開発者が承認した方針（E-32）に沿って、次の 3 つを足した。どれも Claude Code の案で、本文では「（v0.3）」と印を付けた。数値などの細部で、開発者が遊ぶときに直せる任意の確認点（C-11〜C-16）は、計画書 9.4 節に一覧がある。

1. **クラスの役割を尖らせる**（10.3・10.9 節）: ロードは高命中・高速・紙耐久、ファイターは高火力・低耐久、アーマーは守備が最強で火力は中・命中が不安。敵の能力は変えない（E-32）ので、クラスの `base` は変えず、自軍のキャラのボーナスと成長率で尖らせる
2. **回復役（杖の僧侶）を仲間に加える**（4.4・4.5 節、DES-HEL-01・02）: 行動「回復」を足す
3. **予測で、倒せる・倒される戦闘を目立たせる**（7.11 節、DES-UI-07 の変更）

## 1. 反復 1 の設計からの変更点

| 設計要素 | 反復 2 での扱い | 節 |
|---|---|---|
| DES-PKG-04 `internal/core` | 責務に、キャンペーンの状態・流れと、セーブの中身の読み書きを足す | 2 章 |
| DES-PKG-08 `internal/savefile`（新） | セーブのファイルの置き場所と、壊れない書き方 | 2 章、6.2 節 |
| DES-DAT-01〜03 | 成長率・レベル・章・物語・出撃の枠・敵の名前を足す。`game.toml` の `start_map` と、マップの `allies` をやめる | 3 章 |
| DES-BTL-01 | `UnitSetup`・`Unit` に `Level`・`Exp`・`Growth` を、`Item` に `ID` を足す。`Setup` の規則 7 を足す | 4.3 節 |
| DES-BTL-06 | 攻撃の流れの終わりに、経験とレベルアップを足す（勝ち負けを確かめる前） | 4.1・4.2 節 |
| DES-RNG-01 | 成長の乱数を、攻撃の流れの乱数のあとに引く | 4.2 節 |
| DES-EVT-01 | 出来事 `Exp`・`LevelUp` を足す | 4.3 節 |
| DES-CORE-01 | 自軍を `game.toml` の名簿と、マップの出撃の枠から置く。敵の名前を定義データから取れるようにする | 5.5 節 |
| DES-UI-01、02 | 戦闘の画面は 1 つの場面になる。勝ち負けの画面（`Over`）の R・Esc をやめ、章の結果とゲームオーバーの場面に進む（DES-UI-17） | 7 章 |
| DES-UI-06 | 情報欄に、自軍のレベルと経験を足す | 7.8 節 |
| DES-CMD-01 | DES-CMD-02 に置き換える（セーブの置き場所を決め、タイトルから始める）。その前の Issue 2 の間は、始めるマップを `chapters.toml` の最初の章のマップにする（9 章） | 8.1 節 |
| DES-DAT-01〜03（v0.3） | 武器の種類に杖（`staff`）を、武器に回復量（`heal`）を足す。検証の規則 4・6 を変え、21・22 を足す | 3 章 |
| DES-BTL-01（v0.3） | `WeaponType` に `Staff`、`Weapon` に `Heal`、`Action` に `Heal` を足す。`Setup` の規則 8 を足す | 4.4 節 |
| DES-BTL-04（v0.3） | 行動「回復」（`Heal`）を足す。杖のユニットは攻撃できない | 4.4 節（DES-HEL-01） |
| DES-BTL-06（v0.3） | 杖のユニットは反撃しない（攻撃を受けるだけ） | 4.4 節 |
| DES-BTL-08（v0.3） | 誤り `ErrCannotAttack`・`ErrCannotHeal` を足す | 4.4 節 |
| DES-GRW-01（v0.3） | 回復すると経験 10 | 4.1 節 |
| DES-UI-02、03（v0.3） | 行動の一覧に「回復」、入力の状態に回復の相手を選ぶ状態（`HealTargeting`）を足す。杖のユニットの範囲は緑で描く | 7.10 節（DES-UI-18） |
| DES-UI-07（v0.3） | 「撃破」「危険」を、予測の上の大きな帯と、盤面の相手の上の色つきの文字で目立たせる | 7.11 節 |

変えないもの: 攻撃の計算（DES-BTL-05）、攻撃の流れ（DES-BTL-06。杖が反撃しないことを除く）、攻撃の予測の中身（DES-BTL-07）、番の進み方（DES-BTL-02）、スキル（DES-SKL-01）、アイテムの使い方（DES-ITM-01）、地形とギミック（DES-MAP-01〜03）、勝ち負け（DES-MAP-04）、敵の動き方（DES-AI-01）、見せ方の改善（DES-UI-06・08。DES-UI-06 はレベルの表示を除く）、敵の能力（クラスの `base`、マップの敵のボーナス。E-32）。

## 2. アーキテクチャ

| ID | パッケージ | 責務 | 依存してよい先 |
|---|---|---|---|
| DES-PKG-04（反復 2 で変更） | `internal/core` | ゲームの状態。定義データから戦闘を組み立てる（DES-CORE-01）。キャンペーン（名簿・章・在庫・離脱かロスト）の状態と流れ（5 章）。セーブの中身（JSON）の読み書き（6 章）。ファイルと OS は扱わず、`io.Reader`・`io.Writer` を受け取る | `defs`、`core/tactics` |
| DES-PKG-08 | `internal/savefile` | セーブのファイルを、決まったディレクトリに壊れない形で書き、読む（NFR-08）。中身の形は知らない | Go の標準ライブラリだけ |

```mermaid
flowchart TB
  MAIN[cmd/renkin] --> UI[internal/ui]
  MAIN --> DEFLOAD[internal/defload]
  MAIN --> DATA[data]
  MAIN --> SAVE[internal/savefile]
  UI --> CORE[internal/core]
  UI --> TACTICS[internal/core/tactics]
  UI --> DEFS[internal/defs]
  CORE --> DEFS
  CORE --> TACTICS
  DEFLOAD --> DEFS
  UI --> EBI[(Ebitengine)]
  DEFLOAD --> TOML[(BurntSushi/toml)]
```

- `core` は `os` を import しない。DES-NFR-01 の依存の検査（Ebitengine と `time` を使わない）は `core` にそのまま効く
- `ui` は `savefile` を import しない。`ui` は、セーブを読み書きする口（`ui.SaveStore`、7.1 節）だけを知り、`cmd/renkin` が `savefile.Store` を渡す
- `internal/savefile` を足すので、AGENTS.md のパッケージ構成の表に 1 行足した（この設計のプルリクエストで足した。REV-020 F-12）
- `tactics` は反復 2 でも World と `defs` を知らない（ADR-0002）。経験と成長は `tactics` の中で扱うが、成長率とレベルは `UnitSetup` で受け取り、結果は `Units()` で読み出す
- 回復（v0.3）も、`tactics` が受け取るのは `Weapon`（種類 `Staff` と回復量 `Heal`）だけで、クラスやキャラの定義は知らない。`tactics` に足すのは、`WeaponType` の `Staff`、`Weapon.Heal`、`Action` の `Heal`、`Battle.HealForecast`、誤り 2 つ。出来事は反復 1 の `Heal` を使い回し、種類を足さない（4.4 節）。ふしぎ錬金に戻すときに結ぶ口（H-04）は、杖の種類と回復の行動の分だけ増える

## 3. 定義データ（反復 2 で変更）

### 3.1 クラス図の変更（DES-DAT-01）

```mermaid
classDiagram
  class GameDef {
    +Party []string
    +Stock []ItemCount
  }
  class CharacterDef {
    +ID string
    +Name string
    +Class string
    +Bonus Stats
    +Level int
    +Growth Growth
    +Weapon string
    +Items []ItemCount
    +Skill string
    +Lord bool
  }
  class Growth {
    +HP int
    +Str int
    +Skl int
    +Spd int
    +Def int
  }
  class WeaponDef {
    +Heal int
  }
  class ChapterDef {
    +ID string
    +Name string
    +Map string
    +Before string
    +After string
    +Joins []string
    +Reward []ItemCount
  }
  class StoryDef {
    +ID string
    +Paragraphs []string
  }
  class MapDef {
    +ID string
    +Name string
    +Rows []string
    +Slots []Slot
    +Enemies []EnemyPlacement
    +Reinforcements []Reinforcement
  }
  class EnemyPlacement {
    +Name string
    +Class string
    +Weapon string
    +Bonus Stats
    +Items []ItemCount
    +AI string
    +Boss bool
    +X int
    +Y int
  }
  class Catalog {
    +Game GameDef
    +Chapters() []ChapterDef
    +Story(id) StoryDef, bool
  }
  Catalog --> GameDef
  Catalog --> ChapterDef
  Catalog --> StoryDef
  CharacterDef --> Growth
  MapDef --> EnemyPlacement
```

- 図にない型と、`Catalog` のほかの関数は反復 1 のまま
- `GameDef.StartMap` をやめ、`Party`（初めの名簿。キャラの ID の並び）と `Stock`（初めの共有の在庫。省略できる）にする
- `CharacterDef` に `Level`（加わるときのレベル。省略すると 1）と `Growth`（成長率。%）を足す
- `MapDef.Allies` をやめ、`Slots`（出撃の枠。`Slot` は `X`・`Y` を持つ）にする。枠の並びの 1 番目は主人公の枠
- `EnemyPlacement` に `Name`（省略できる）を足す。ボスに名前を付け、物語とつなげるため（10.4 節）
- `ChapterDef` は章 1 つ。`Catalog.Chapters()` は、`chapters.toml` に書いた順（章の順）に返す。`Before`・`After` は物語の ID で、空なら物語を出さない。`Joins` は、その章の始めに名簿に加わるキャラの ID の並び。`Reward` は、その章に勝ったときに共有の在庫に入るアイテム
- `StoryDef` は物語 1 本。`Paragraphs` は段落の並び
- （v0.3）武器の種類（`WeaponType`、`Type`）に `staff`（杖）を足す。定数 `defs.WeaponStaff`。`WeaponDef` に `Heal`（杖の回復量）を足す。杖は `heal` を持ち、`might`・`hit` を使わない（0）。杖でない武器は `heal` を持たない（0）

### 3.2 ファイルと TOML の例（DES-DAT-02）

`data/base/` に `chapters.toml` と `story.toml` を足す。ほかのファイルは反復 1 のまま（中身を変えるものがある）。

```toml
# data/base/game.toml
party = ["leon", "gald", "bran", "mira", "rina"]
# stock = [{ item = "potion", count = 1 }]  # 初めの在庫（省略すると空）

# data/base/characters.toml（足すキー）
[[characters]]
id = "elsa"
name = "エルザ"
class = "cavalier"
bonus = { hp = 1, str = 1, skl = 1, spd = 1, def = 0, mov = 0 }
level = 2
growth = { hp = 65, str = 45, skl = 40, spd = 40, def = 35 }
weapon = "iron_lance"
items = [{ item = "potion", count = 1 }]

# （v0.3）杖と僧侶
# data/base/weapons.toml
[[weapons]]
id = "heal_staff"
name = "癒やしの杖"
type = "staff"
heal = 10        # might・hit は書かない（0）
min_range = 1
max_range = 1

# data/base/classes.toml
[[classes]]
id = "cleric"
name = "僧侶"
weapon_type = "staff"
base = { hp = 16, str = 0, skl = 2, spd = 6, def = 2, mov = 5 }

# data/base/characters.toml（ボーナスは負の値も書ける。規則 6 は足したあとの値を見る）
[[characters]]
id = "bran"
name = "ブラン"
class = "fighter"
bonus = { hp = -2, str = 4, skl = 1, spd = 1, def = -1, mov = 0 }
growth = { hp = 70, str = 75, skl = 35, spd = 30, def = 15 }
weapon = "iron_axe"
items = [{ item = "potion", count = 1 }]
skill = "wrath"

# data/base/chapters.toml
[[chapters]]
id = "ch02"
name = "灰の街道"
map = "ch02"
before = "ch02_before"
after = "ch02_after"
joins = ["elsa"]
reward = [{ item = "potion", count = 2 }]

# data/base/story.toml
[[stories]]
id = "ch02_before"
paragraphs = [
  "西の谷へ向かう街道は、逃げる人々で埋まっていた。",
  "エルザ「村を焼けという命令に、従えなかった」",
]

# data/base/maps.toml（変えるキー）
[[maps]]
id = "ch02"
name = "灰の街道"
rows = [ "^^^^^FF^^^^^^^^^", ] # …（16 行。10.5 節）
slots = [{ x = 1, y = 7 }, { x = 1, y = 6 }, { x = 1, y = 8 }, { x = 0, y = 6 }, { x = 0, y = 8 }, { x = 2, y = 7 }]

[[maps.enemies]]
name = "ヴォルグ"
class = "cavalier"
weapon = "iron_lance"
bonus = { hp = 8, str = 2, skl = 3, spd = 2, def = 1, mov = 0 }
ai = "hold"
boss = true
x = 14
y = 8
```

### 3.3 意味の検証（DES-DAT-03）

反復 1 の規則 1〜13 のうち、3・4・6・8・9・10 を変え、14〜22 を足す（4・6・21・22 は v0.3）。

| # | 規則 |
|---|---|
| 3（変更） | 参照（クラス、武器、アイテム、キャラ、地形の文字、**章のマップ、章の物語、名簿と加入のキャラ**）が存在する。「始めるマップ」はなくなる |
| 4（v0.3 で変更） | 武器の種類は `sword`・`lance`・`axe`・`bow`・`staff` のどれか。スキルの効果の種類と敵の動き方は反復 1 のまま |
| 6（v0.3 で変更） | 反復 1 の数の範囲に、武器の `heal` は 0 以上、を足す。クラスの能力の範囲は、ボーナス（負の値も書ける）を足したあとの値で確かめる（反復 1 のまま。v0.3 でボーナスに負の値を使うので明記した） |
| 8（変更なし。確認） | マップの行の数と行の長さは、どちらも 1 以上 16 以下（反復 2 もスクロールしない。E-26） |
| 9（変更） | 出撃の枠・敵の配置・増援の座標がマップの中にあり、通れる地形にある。出撃の枠どうし、初めの敵の配置どうし、出撃の枠と初めの敵の配置は、どれも重ならない |
| 10（変更） | `game.toml` の `party` が 1 人以上で、同じキャラが 2 回ない。`lord = true` のキャラが `party` にちょうど 1 人いて、`party` の外に `lord = true` のキャラはいない |
| 14 | 章が 1 つ以上ある。各章の `slots` の数（マップの出撃の枠の数）が 1 以上 |
| 15 | 加入（`joins`）のキャラは、`party` にも、ほかの章の `joins` にも出てこない（1 人は 1 回だけ加わる） |
| 16 | キャラの `level` は 1 以上 20 以下。省略すると 1 で、`level = 0` と書いたら誤り。`growth` の各値は 0 以上 100 以下。`growth` を省略すると、すべて 0（成長しない）。表の中の能力を省略したものも 0 |
| 17 | キャラの `items` の数の合計は 5 以下（手持ちの上限。DES-CMP-04）。`stock`・`reward` の各要素の数は 1 以上で、同じアイテムを 2 回書かない |
| 18 | 物語の `paragraphs` は 1 つ以上で、各段落は空でなく、200 文字（rune で数える）以下 |
| 19 | 敵の `name` は、書くなら空でない |
| 20 | 章・物語の ID は、それぞれの中で重ならず、空でない（規則 2 と同じ） |
| 21（v0.3） | 杖（`type = "staff"`）は `heal` が 1 以上で、`might`・`hit` が 0。杖でない武器は `heal` が 0 |
| 22（v0.3） | 敵の配置と増援の武器は杖でない（敵は回復しない。4.4 節）。杖のクラスの敵も置かない（規則 5 で同じことになる） |

- 誤りの位置は反復 1 と同じく、キーの道筋で示す（例: `chapters.toml: chapters[1].joins[0]: キャラ "elsa" は party にもいます`）
- 規則 13（知らないキー）は、足したファイルとキーにも効く。`start_map`・`allies` を書くと、知らないキーの誤りになる

## 4. 経験と成長（core/tactics）

### 4.1 経験（DES-GRW-01）

経験は `tactics` の中で、攻撃の流れ（DES-BTL-06）が終わるたびに足す（FR-UNT-04）。

- 経験を得るのは、`Level` が 1 以上の**自軍**のユニットだけ。敵と、`Level` が 0 の自軍（反復 1 のテストの `Setup` など）は経験を得ない。`Level` が 20 のユニットも得ない
- 1 回の攻撃の流れ（自軍の番の `Attack` も、敵軍の番の敵の攻撃も）には、自軍と敵が 1 体ずつ加わる。流れが終わったとき、加わった自軍のユニットが生きていれば、次の経験を足す（C-01）

| 条件（上から順に見る） | 経験 |
|---|---|
| 相手（敵）がこの流れの中で倒れた | 30 |
| このユニットの 1 撃が 1 回以上当たった（ダメージが 0 でも、`Hit` が真なら当たり） | 10 |
| それ以外（すべて外れた。反撃できず 1 撃も行わなかった） | 1 |

- 待機とアイテムの使用では、経験を得ない
- 加わった自軍のユニットが倒れたときは、経験を足さない
- （v0.3）**回復**（`Heal`。4.4 節）を行った自軍のユニット（杖を持つユニット）は、経験 **10** を得る（C-13）。回復される側は経験を得ない。回復の行動は、相手の HP が最大でないことを確かめてから行うので、回復の量は必ず 1 以上で、経験を得ない回復はない。経験の上限・レベル 0 と 20 の扱いは、上と同じ
- 杖のユニットが敵に攻撃されて生き残ったときは、反撃できないので「それ以外」の 1 を得る（例 6 のミラと同じ）

### 4.2 レベルアップと成長（DES-GRW-02）

- 経験を足して 100 以上になったら、レベルを 1 上げ、経験から 100 を引く（余りは持ち越す）。1 回に足す経験は 30 以下で、足す前の経験は 99 以下なので、1 回に上がるのは 1 レベルまで
- レベルが 20 になったら、経験を 0 にする（余りは捨てる）
- レベルが上がったら、**HP・力・技・速さ・守備の順に**、0〜99 の乱数（`roll`。DES-RNG-01）を 1 つずつ、計 5 つ引く。引いた値が、そのキャラのその能力の成長率より小さければ、その能力を 1 上げる（FR-UNT-05）。移動は上がらない。成長率が 0 なら上がらず、100 なら必ず上がる。5 つとも上がらなくても、引き直さない
- 最大 HP が上がったら、今の HP も 1 上げる
- 能力の上限は設けない（反復 2 では、上がっても 1 人 3〜4 レベルの見込み。10.2 節）

**乱数を引く順（DES-RNG-01 に足す）**: 1 回の攻撃の流れの中では、1 撃ごとの乱数（命中 2 つ、大盾 1 つ）をすべて引いたあと、レベルが上がったときだけ、成長の乱数を 5 つ引く。回復の行動（v0.3）は命中の乱数を引かず、レベルが上がったときだけ成長の乱数を 5 つ引く。同じ種と同じコマンドの列なら、成長も同じになる（NFR-06）。

**出来事の順**: `Strike`（…）→ 倒れたら `Fall` → `Exp` → レベルが上がったら `LevelUp` → 勝ち負けが決まったら `Won`・`Lost`。ボスを倒した 1 撃でも、経験とレベルアップを済ませてから勝ちを出す。

**計算例**（10 章の仮置きの値。スクリプトで検算した）

| # | 場面 | 計算 | 結果 |
|---|---|---|---|
| 1 | レオン（Lv 1、経験 0）が、ソルジャーに 1 撃を当て、倒さない | 経験 0 ＋ 10 = 10 < 100 | Lv 1、経験 10。成長の乱数は引かない。出来事は `Exp`（10）だけ |
| 2 | ミラ（Lv 1、経験 90。成長率 55/40/60/45/20）が敵を倒す。成長の乱数は (12, 75, 40, 44, 90) | 90 ＋ 30 = 120 → Lv 2、経験 20。HP 12 < 55 ○、力 75 < 40 ×、技 40 < 60 ○、速さ 44 < 45 ○、守備 90 < 20 × | Lv 2、経験 20。HP・技・速さ ＋1 |
| 3 | ブラン（Lv 1、経験 80、HP 22。v0.3 の能力 22/11/5/5/2/5、成長率 70/75/35/30/15）が、平地の剣士（HP 10）を攻撃。乱数の列は (20, 30, 50, 75, 10, 30, 5) | 1 撃目: 命中率 47（10 ＋ 70 − 15 − 18）。20 ＋ 30 = 50 < 47 × 2 = 94 → 当たり、16 ダメージ（11 ＋ 8 − 1 − 2）で剣士が倒れる（反撃はない）。経験 80 ＋ 30 = 110 → Lv 2、経験 10。成長: HP 50 < 70 ○、力 75 < 75 ×（小さくないので上がらない）、技 10 < 35 ○、速さ 30 < 30 ×、守備 5 < 15 ○ | 出来事は `Strike` → `Fall` → `Exp`（30）→ `LevelUp`（Lv 2、HP・技・守備 ＋1）。乱数は 7 つ引いた。`Exp` と `LevelUp` の写しのブランは、どちらも更新後の Lv 2、経験 10、能力 23/11/6/5/3/5、今の HP 23（4.3 節） |
| 4 | レオン（Lv 3、経験 99。v0.3 の成長率 55/45/65/70/20）が、1 撃も当てずに生き残る。成長の乱数は (99, 0, 65, 69, 20) | 99 ＋ 1 = 100 → Lv 4、経験 0。HP 99 ×、力 0 ○、技 65 < 65 ×、速さ 69 < 70 ○、守備 20 < 20 × | Lv 4、経験 0。力・速さ ＋1 |
| 5 | Lv 19、経験 95 のユニットが当てる | 95 ＋ 10 = 105 → Lv 20、経験は 0 にする | Lv 20、経験 0。以後は経験を得ない（`Exp` も出さない） |
| 6 | 敵軍の番に、ファイターが隣のミラ（弓）を攻撃し、外れる | ミラは隣に反撃できない。1 撃も行わず、生き残った | ミラの経験 ＋1 |
| 7（v0.3） | リナ（僧侶、Lv 1、経験 95。成長率 55/0/40/60/30）が、隣の HP 12/19 のレオンを癒やしの杖（回復 10）で回復する。成長の乱数は (10, 0, 50, 59, 40) | 回復の量は min(10, 19 − 12) = 7。経験 95 ＋ 10 = 105 → Lv 2、経験 5。HP 10 < 55 ○、力 0 < 0 ×、技 50 < 40 ×、速さ 59 < 60 ○、守備 40 < 30 × | 出来事は `Heal`（レオン、7、「癒やしの杖」）→ `Exp`（リナ、10）→ `LevelUp`（Lv 2、HP・速さ ＋1）。乱数は成長の 5 つだけ引いた |

### 4.3 型と出来事（DES-GRW-03。DES-BTL-01・DES-EVT-01 の変更）

```mermaid
classDiagram
  class UnitSetup {
    +Level int
    +Exp int
    +Growth Growth
  }
  class Unit {
    +Level int
    +Exp int
    +Growth Growth
  }
  class Growth {
    +HP int
    +Str int
    +Skl int
    +Spd int
    +Def int
  }
  class Item {
    +ID string
    +Name string
    +Heal int
  }
  UnitSetup --> Growth
  Unit --> Growth
```

- 図は足す欄だけを書いた。ほかの欄は反復 1 のまま
- `Item.ID` は、`tactics` の外（`core`）がアイテムを見分けるための値で、`tactics` は使わない（比べもしない）。戦闘のあとに手持ちを名簿に書き戻すために使う（DES-CMP-06）
- `Setup` の検証に規則 7 を足す。当てはまらなければ `ErrInvalidSetup` を返す

| # | `Setup` の規則 |
|---|---|
| 7 | `Level` は 0 以上 20 以下。敵（増援を含む）の `Level` は 0。`Exp` は 0 以上 99 以下で、`Level` が 0 か 20 なら 0。`Growth` の各値は 0 以上 100 以下 |

- 出来事を 2 つ足す。`EventKind` の定数は、反復 1 の定数のあと（`EventLost` のあと）に足し、反復 1 の値を変えない。`Event` に `Level`・`Gains`（`Stats` の型。移動は 0）を足す

| `Kind` | 起きたこと | 主な項目 |
|---|---|---|
| `Exp` | 経験を得た | `Unit`、`Amount`（得た経験。1・10・30 のどれか） |
| `LevelUp` | レベルが上がった | `Unit`、`Level`（上がったあとのレベル）、`Gains`（上がった能力。上がった能力が 1、ほかは 0） |

- **写しの値**: 経験を足し、レベルアップと成長の判定をすべて済ませて、ユニットの経験・レベル・能力・今の HP を更新してから、`Exp` と（レベルが上がったら）`LevelUp` を出す。どちらの出来事の写し（`Units`）も、更新後の値になる。`Exp` の写しで経験が足す前の値に見えることはない。画面は、上がる前の値を `Gains` と `Level` から求める

### 4.4 回復の行動（DES-HEL-01。DES-BTL-01・04・06・08 の変更。v0.3）

杖を持つユニット（僧侶）は、攻撃の代わりに、近くの味方を回復する（E-30）。FR-BTL-03 の「行動の種類は設計で決める」への答えとして、行動「回復」を足す。

**型の追加（DES-BTL-01）**: どれも反復 1 の定数の値を変えないよう、後ろに足す。

| 型 | 足すもの |
|---|---|
| `WeaponType` | `Staff`（杖）。`Bow` のあと |
| `Weapon` | `Heal int`（杖の回復量。杖でなければ 0） |
| `Action` | `Heal`（回復）。`UseItem` のあと。`Command.Target` は、`Heal` のときは回復する味方の添字 |

- `Setup` の検証に規則 8 を足す。当てはまらなければ `ErrInvalidSetup` を返す

| # | `Setup` の規則 |
|---|---|
| 8 | 武器の種類は `Sword`〜`Staff` のどれか（規則 6 の範囲を広げる）。杖は `Heal` が 1 以上、杖でない武器は `Heal` が 0。敵（増援を含む）の武器は杖でない |

**`Act` の `Heal`（DES-BTL-04 に足す）**

| `Action` | 意味 | 追加の検証（共通の検証 0〜5 のあと、順に確かめる） |
|---|---|---|
| `Heal` | 移動してから、`Target` の味方の HP を、杖の `Heal` だけ回復する（最大 HP を超えない） | 6a. ユニットの武器が杖である（`ErrCannotHeal`）。6b. `Target` が生きている同じ陣営のユニットで、自分でない（`ErrNoTarget`）。6c. `To` から `Target` までの距離が杖の射程（最小〜最大）に入っている（`ErrOutOfRange`）。6d. `Target` の HP が最大でない（`ErrFullHP`） |

- 共通の検証 0 は「`Action` が 4 つ（`Wait`・`Attack`・`UseItem`・`Heal`）のどれかである」に変える
- `Attack` の検証の最初に「ユニットの武器が杖でない（`ErrCannotAttack`）」を足す（`ErrNoTarget` より前に確かめる）
- 杖のユニットも、手持ちの回復薬を `UseItem` で自分に使える。待機もできる
- 杖は使用回数を持たない（反復 1 の武器と同じ）
- 回復の流れ: 移動（`Move`）→ `Heal`（`Unit` は回復された味方、`Amount` は実際に回復した量、`Item` は杖の名前）→ `Exp`（回復した杖のユニット、10。4.1 節）→ レベルが上がったら `LevelUp`。そのあと、反復 1 と同じく行動済みにし、自軍が全員行動済みなら敵軍の番に進む。回復で勝ち負けは決まらない
- 出来事は反復 1 の `Heal`（砦・回復薬）を使い回し、種類を足さない。`Item` が杖の名前になるだけで、画面は反復 1 と同じく回復された側の上に「+7」を出す

**反撃しない（DES-BTL-06 の変更）**: 杖のユニットは、攻撃されても反撃しない。杖の射程（1）に相手がいても、反撃できる距離として数えない（`canCounter` が偽）。反撃しないので追撃もしない。攻撃の予測（DES-BTL-07）では、杖の側の回数は 0（「反撃なし」）になる。相性（DES-BTL-05）は、杖を持つ守る側では 0（弓と同じ）。攻撃の計算の式は変えない

**敵の動き方（DES-AI-01）は変えない**: 敵は杖を持たない（規則 8、DES-DAT-03 の規則 22）ので、敵は回復しない。敵は杖のユニットも攻撃の候補にする（反撃のダメージが 0 なので、候補の比べ方の 3 で選ばれやすい）。守る位置に置くのはプレイヤーの役目

**範囲**: `AttackRange(unit, from)` は、杖のユニットでも杖の射程のマスを返す（`tactics` は変えない）。画面はそれを緑で描く（DES-UI-18）。危険範囲（DES-UI-08）は敵だけを数えるので変わらない

**誤り（DES-BTL-08 に足す）**

| 誤り | 意味 |
|---|---|
| `ErrCannotAttack` | 杖のユニットは攻撃できない |
| `ErrCannotHeal` | 杖を持っていないので回復できない |

- `ErrInvalidAction` の意味に「`HealForecast` に `Heal` でないコマンドを渡した」を足す

### 4.5 回復の予測（DES-HEL-02。v0.3）

`HealForecast(cmd Command) (HealForecast, error)` は、`Action` が `Heal` のコマンドについて、状態を変えずに次を返す（UC-02 の 5c）。検証は `Act` と同じで、`Action` が `Heal` でなければ `ErrInvalidAction` を返す。

| 項目 | 内容 |
|---|---|
| `Healer` | 回復する側の名前 |
| `Target` | 回復される側の名前 |
| `HP`、`MaxHP` | 回復される側の今の HP と最大 HP |
| `Amount` | 実際に回復する量。min(杖の `Heal`, `MaxHP` − `HP`) |
| `After` | 回復したあとの HP（`HP` ＋ `Amount`） |

- 攻撃の予測（`Forecast`）は変えない。`Forecast` に `Heal` のコマンドを渡したら、反復 1 のまま `ErrInvalidAction`
- 計算例（スクリプトで検算した）: リナ（癒やしの杖、回復 10）が HP 12/19 のレオンを回復すると、`Amount` 7、`After` 19。HP 5/19 なら `Amount` 10、`After` 15

## 5. キャンペーン（core）

### 5.1 クラス図（DES-CMP-01）

```mermaid
classDiagram
  class Campaign {
    +Rule Rule
    +Chapter int
    +Cleared bool
    +Roster []Member
    +Stock []ItemCount
    +TakeItem(member int, item string) error
    +StoreItem(member int, item string) error
    +StartBattle(members []int, seed uint64) Sortie, []Event, error
    +FinishChapter(s Sortie) ChapterResult, error
    +Save(w io.Writer) error
  }
  class Member {
    +Character string
    +Level int
    +Exp int
    +Stats tactics.Stats
    +Items []ItemCount
    +Alive bool
  }
  class Sortie {
    +Battle tactics.Battle
    +Members []int
  }
  class ChapterResult {
    +Chapter string
    +Members []MemberResult
    +Reward []ItemCount
    +Cleared bool
  }
  class MemberResult {
    +Member int
    +LevelBefore int
    +LevelAfter int
    +ExpBefore int
    +ExpAfter int
    +Gains tactics.Stats
    +Fallen bool
    +Dead bool
  }
  Campaign --> Member
  Campaign ..> Sortie
  Campaign ..> ChapterResult
  ChapterResult --> MemberResult
```

- 図では `*` を省いた。`StartBattle` は `*Sortie` を返し、`FinishChapter` は `*Sortie` を受け取る。`Sortie.Battle` は `*tactics.Battle`
- `NewCampaign(cat *defs.Catalog, rule Rule) (*Campaign, error)` で新しく始め、`LoadCampaign(cat *defs.Catalog, r io.Reader) (*Campaign, error)` でセーブから読む（6 章）。`Campaign` は目録を内部に持つ（セーブには入れない）
- `Rule` は `Withdraw`（離脱）と `Permadeath`（ロスト）（FR-CMP-04）
- `Chapter` は今の章の添字（`cat.Chapters()` の 0 から数えた位置）。`Cleared` が真なら、最後の章に勝ったあと
- `Roster` は名簿。初めの名簿の順に、加わった順で後ろに足す。ロストで倒れた仲間も、`Alive` を偽にして名簿に残す（加入を 2 回しないため。画面には出さない）
- `Member.Stats` は今の能力（クラス＋ボーナス＋成長。移動を含む）。`Items` は手持ち（`defs.ItemCount`）
- `Stock` は共有の在庫。アイテムの ID ごとに 1 つの要素で、ID の順に並べる。数が 0 になったら要素を消す
- 名簿のキャラの名前・クラス・武器・スキル・成長率は、目録のキャラの定義から引く（名簿には持たない）。成長率を調整しても、セーブを作り直さずに効く

### 5.2 流れ（DES-CMP-02）

```mermaid
stateDiagram-v2
  [*] --> Title
  Title --> RuleSelect: 新規開始
  RuleSelect --> StoryBefore: 離脱かロストを選ぶ（NewCampaign、セーブ）
  Title --> StoryBefore: 続きから（LoadCampaign）
  Title --> Clear: 続きから（クリア済みのセーブ）
  StoryBefore --> Joined: 加入がある
  StoryBefore --> Prepare: 加入がない
  Joined --> Prepare
  Prepare --> Battle: 出撃（StartBattle）
  Battle --> Result: 勝ち（FinishChapter、セーブ）
  Battle --> GameOver: 負け
  GameOver --> Prepare: この章をやり直す（LoadCampaign）
  GameOver --> Title: タイトルへ
  Result --> StoryAfter
  StoryAfter --> StoryBefore: 次の章がある
  StoryAfter --> Clear: 最後の章だった
  Clear --> Title
```

- 図の状態は画面の場面（7 章）で、`core` は状態を持たない。`core` の状態は `Chapter`・`Cleared` と名簿・在庫だけ
- 物語がない章（`Before`・`After` が空）では、物語の場面を飛ばす
- 加入（`Joins`）は、`core` では章に入るとき（5.3 節）に名簿に足す。画面は、章の前の物語のあとに「加わった」と出す（DES-UI-12）

### 5.3 新規開始と章に入る（DES-CMP-03）

`NewCampaign(cat, rule)`:

1. `rule` が `Withdraw`・`Permadeath` のどれかでなければ `ErrInvalidRule`
2. 名簿を `game.toml` の `party` の順に作る。各キャラは、レベル＝キャラの `level`、経験 0、能力＝クラスの `base` ＋ キャラの `bonus`、手持ち＝キャラの `items`、`Alive` は真
3. 在庫を `game.toml` の `stock` にする
4. `Chapter` を 0 にし、章に入る（下）

**章に入る**（`NewCampaign` と `FinishChapter` の中で行う）: 今の章の `Joins` のキャラを、`Joins` の順に名簿の後ろに足す。足し方は 2 と同じ。

### 5.4 章の準備（DES-CMP-04）

出撃するキャラの選択は画面が持ち、`StartBattle` に渡す（5.5 節）。アイテムの入れ替えは、`Campaign` の操作で行う（FR-CMP-03）。

| 操作 | 意味 | 検証（順に確かめ、最初に当てはまった誤りを返す） |
|---|---|---|
| `TakeItem(member, item)` | 在庫の `item` を 1 つ、名簿の `member` 番目の手持ちに移す | クリア済みでない（`ErrCleared`）。`member` が名簿にいて生きている（`ErrInvalidMember`）。在庫に `item` が 1 つ以上ある（`ErrNoStock`）。手持ちの数の合計が 5 より小さい（`ErrHandFull`） |
| `StoreItem(member, item)` | 名簿の `member` 番目の手持ちの `item` を 1 つ、在庫に移す | クリア済みでない（`ErrCleared`）。`member` が名簿にいて生きている（`ErrInvalidMember`）。手持ちに `item` がある（`ErrNoHandItem`） |

- 手持ちに同じ ID の要素があれば数を 1 増やし、なければ後ろに足す。数が 0 になった要素は消す
- 手持ちの上限は、数の合計で 5（10.7 節）
- 入れ替えは、セーブするまでファイルには残らない。負けてやり直すと、章の始めのセーブに戻る（DES-CMP-07）

### 5.5 戦闘を作る（DES-CMP-05、DES-CORE-01 の変更）

`StartBattle(members []int, seed uint64) (*Sortie, []tactics.Event, error)` は、今の章のマップで戦闘を作る（FR-CMP-03、TBD-04）。

検証（順に確かめ、最初に当てはまった誤りを返す。状態は変えない）:

1. クリア済みでない（`ErrCleared`）
2. `members` の数が 1 以上、今の章のマップの出撃の枠の数以下（`ErrInvalidDeployment`）
3. 各要素が名簿の添字で、生きていて、重ならない（`ErrInvalidDeployment`）
4. `members[0]` が主人公（`lord = true` のキャラ）である（`ErrInvalidDeployment`）。主人公は必ず出撃し、枠の 1 番目に置く

組み立て:

- 自軍は `members` の順に、マップの `slots` の同じ順の座標に置く。`Units()` の添字 `i`（`i < len(members)`）は `members[i]` の名簿のキャラになる
- 自軍の `UnitSetup` は、名前・クラスの名前・武器・スキル・主人公かどうか・成長率をキャラの定義から、`Level`・`Exp`・`Stats`・手持ちを名簿から作る。手持ちの `Item.ID` にアイテムの ID を入れる。HP は最大から始まる（離脱で倒れた仲間も、次の章では HP が最大で戻る。FR-CMP-05）
- 敵・増援・地形は反復 1 の DES-CORE-01 のまま。ただし、敵の配置に `name` があれば、それを名前にする。番号を付けるときは、`name` のある敵を数えない
- （v0.3）武器は、反復 1 の変換に、種類 `staff` → `tactics.Staff` と、`heal` → `Weapon.Heal` を足して作る
- `Sortie` は、作った戦闘（`Battle`）と、`members` の写しを持つ。どの `Campaign` のどの章で作ったかを内部に持つ（DES-CMP-06 の検証のため）
- `StartBattle` は `Campaign` を変えない。何度呼んでもよい

**`core.NewBattle` の変更（DES-CORE-01）**: `core.NewBattle(cat, mapID, seed)` は、反復 1 のテストのために残す。自軍は、`game.toml` の `party` のキャラを、`party` の順にマップの `slots` の数まで置く（主人公は `party` のどこにいても枠の 1 番目に置く）。レベル・成長率はキャラの定義から、経験は 0 にする。`StartBattle` と同じ組み立ての関数を使う。

### 5.6 章の終わり（DES-CMP-06）

`FinishChapter(s *Sortie) (ChapterResult, error)` は、勝った戦闘の結果を名簿に書き戻し、次の章に進む（FR-CMP-01、02、05、06、10、FR-ITM-03）。

検証（状態は変えない）:

1. `s` がこの `Campaign` の今の章で作ったもので、まだ `FinishChapter` に渡していない（`ErrInvalidSortie`）
2. `s.Battle.Result` が `Won`（`ErrNotWon`）

書き戻しと進め方:

1. 出撃した各キャラ（`members[i]` と `Units()[i]`）について、名簿のレベル・経験・能力（`Stats`。最大 HP を含む）・手持ちを、ユニットの値にする。手持ちは `Item.ID` で ID に戻す。倒れる前に上がったレベルと能力も残す
2. 倒れたキャラ（`Alive` が偽）は、`Rule` が `Permadeath` なら名簿の `Alive` を偽にし、手持ちを空にする（C-08。FR-CMP-06）。`Withdraw` なら名簿に残し、次の章で戻る（FR-CMP-05）
3. 今の章の `Reward` を在庫に足す
4. `Chapter` を 1 進める。最後の章だったら `Cleared` を真にする（`Chapter` は章の数と同じ値になる）。そうでなければ次の章に入る（DES-CMP-03。加入を名簿に足す）
5. `s` を渡し済みにする

`ChapterResult` には、終えた章の ID、出撃した各キャラの前後のレベルと経験、上がった能力（`Gains`。後 − 前）、倒れたか（`Fallen`）、戦死したか（`Dead`。ロストで倒れたとき）、報酬、クリアしたかを入れる。出撃しなかったキャラは載せない（経験も得ない）。

- 主人公が倒れると負け（FR-CMP-09）なので、勝った戦闘で主人公が倒れていることはない。ロストでも主人公は戦死しない

### 5.7 負けとやり直し（DES-CMP-07）

- 負け（`Result` が `Lost`）のとき、`core` は何もしない。`FinishChapter` は `ErrNotWon` を返す
- 画面は、章に入るたびに（`NewCampaign`・`LoadCampaign`・`FinishChapter` が成功したあと）、`Campaign.Save` の中身（章の始めの状態）をメモリにも持つ。ファイルへのセーブ（DES-SAV-02）は「続きから」のためのもので、やり直しには使わない（ファイルのセーブに失敗していても、やり直せる）
- 「この章をやり直す」では、メモリに持った中身から `LoadCampaign` し、章の準備の場面から始める。章の前の物語と加入の表示は飛ばす（C-06）。準備で入れ替えたアイテムも、章の始めに戻る
- ロストで遊んでいても、負けた戦闘の中で倒れた仲間は、やり直すと戻る（その戦闘は無かったことになる）

### 5.8 誤り（DES-CMP-08）

`Campaign` の操作と `NewCampaign`・`LoadCampaign` は、失敗したら状態を変えず、次の誤りを返す（NFR-07）。`errors.Is` で見分けられるようにする。

| 誤り | 意味 |
|---|---|
| `ErrInvalidRule` | 離脱・ロストのどれでもない |
| `ErrCleared` | クリア済みなので、準備も出撃もできない |
| `ErrInvalidMember` | 名簿にいない、または戦死している |
| `ErrNoStock` | 在庫にそのアイテムがない |
| `ErrHandFull` | 手持ちが上限（5）に達している |
| `ErrNoHandItem` | 手持ちにそのアイテムがない |
| `ErrInvalidDeployment` | 出撃の選び方が正しくない（数、重なり、戦死、主人公が先頭でない） |
| `ErrInvalidSortie` | ほかの章・ほかのキャンペーンの戦闘か、渡し済み |
| `ErrNotWon` | 戦闘に勝っていない |
| `ErrSaveVersion` | セーブの版が読めない値（6.3 節） |
| `ErrInvalidSave` | セーブの中身が正しくない（6.3 節。理由を含める） |

## 6. セーブ

### 6.1 形式（DES-SAV-01）

セーブは JSON の 1 つのファイル（枠は 1 つ。FR-SYS-01）。`Campaign.Save(w)` が書き、`LoadCampaign(cat, r)` が読む。

```json
{
  "version": 1,
  "rule": "withdraw",
  "chapter": "ch02",
  "cleared": false,
  "roster": [
    {
      "character": "leon",
      "level": 2,
      "exp": 40,
      "stats": { "hp": 21, "str": 6, "skl": 9, "spd": 10, "def": 4, "mov": 5 },
      "items": [{ "item": "potion", "count": 2 }],
      "alive": true
    }
  ],
  "stock": [{ "item": "potion", "count": 2 }]
}
```

- `version` は 1。形を変えたら上げる
- `rule` は `withdraw`（離脱）か `permadeath`（ロスト）
- `chapter` は今の章の ID。クリア済み（`cleared` が真）なら空の文字列
- `roster` は名簿の順。`items`・`stock` が空なら `[]`
- 字下げは空白 2 つ。同じ状態なら同じ文字列になる（在庫は ID の順。名簿は名簿の順）

### 6.2 ファイルの置き場所と書き方（DES-SAV-02、DES-PKG-08）

`internal/savefile` の `Store{Dir string}` が、ファイルを扱う（IF-04、NFR-08）。

| 関数 | 動き |
|---|---|
| `Path() string` | `Dir` の下の `save.json` |
| `Exists() (bool, error)` | `save.json` があるか |
| `Read(fn func(io.Reader) error) error` | `save.json` を開いて `fn` に渡す。なければ `fs.ErrNotExist` を包んだ誤り |
| `Write(fn func(io.Writer) error) error` | 下の手順で書く |

`Write` の手順（NFR-08）:

1. `Dir` を作る（なければ。`os.MkdirAll`、0o755）
2. 同じディレクトリに一時ファイルを作る（`os.CreateTemp(Dir, "save-*.tmp")`）
3. `fn` に一時ファイルを渡して中身を書く
4. 一時ファイルを `Sync` し、閉じる
5. 一時ファイルの名前を `save.json` に変える（`os.Rename`。同じディレクトリの中なので置き換えは一度に起きる。Windows でも、ある `save.json` を置き換える）
6. 2〜5 のどこかで失敗したら、一時ファイルを消して誤りを返す。`save.json` は書く前のまま残る

- 書いている途中で強制終了しても、`save.json` は前のままで、残るのは一時ファイルだけ（`Read` は一時ファイルを読まない）。残った一時ファイルは、次の `Write` で `save-*.tmp` を消して片付ける
- 置き場所は `cmd/renkin` が決める（DES-CMD-02）。IF-04（OS の慣例に従うユーザーのデータ用ディレクトリ）に合わせた:

| 場合 | ディレクトリ |
|---|---|
| `-savedir` の指定がある | そのディレクトリ |
| Windows | `os.UserConfigDir()`（`%AppData%`）の下の `renkin_tactics` |
| Linux（ほかの Unix を含む） | 環境変数 `XDG_DATA_HOME` が空でなければその下、空なら `os.UserHomeDir()` の下の `.local/share` の、さらに下の `renkin_tactics`（例: `~/.local/share/renkin_tactics/save.json`） |
| macOS | `os.UserConfigDir()`（`~/Library/Application Support`）の下の `renkin_tactics` |

- `-savedir` は、試験と遊び比べ（E-25）のときに、ふだんのセーブを上書きしないために使う

**セーブする時機**（自動。プレイヤーが選ぶ操作はない）:

| 時機 | 中身 |
|---|---|
| 新規開始で離脱かロストを選んだ直後（`NewCampaign` のあと） | 1 章の始め |
| 「勝利」の帯をクリックして `FinishChapter` が成功した直後（章の結果を見る前。C-07） | 次の章の始め（加入を足したあと）。最後の章ならクリア済み |

- セーブに失敗したら、次の場面（新規開始のあとは 1 章の前の物語、章に勝ったあとは章の結果）に「セーブできませんでした: 理由」と出して、遊びは続ける（ファイルには前のセーブが残る。やり直しはメモリの中身で行える。DES-CMP-07）
- セーブする時機には、勝ちが決まった時点ではなく、「勝利」の帯をクリックしたとき（DES-UI-17）にした。帯をクリックする前に窓を閉じると、その章の勝ちは残らない（C-07 にほかの案を添えた）

### 6.3 読むときの検証（DES-SAV-03）

`LoadCampaign(cat, r)` は、次の順で読む。1 つでも当てはまれば、`Campaign` を返さず誤りを返す。

1. まず `version` だけを読み（ほかのキーは見ない）、規則 2 を確かめる。`version` を読めない（JSON の文法の誤り、`version` がない・数でない）ときは規則 1 の誤り（`ErrInvalidSave`）
2. そのあと、知らないキーを許さずに全体を読み、規則 1・3〜7 を表の順に確かめ、最初に当てはまった誤りを返す

| # | 規則 | 誤り |
|---|---|---|
| 1 | JSON の文法が正しく、知らないキーがない | `ErrInvalidSave` |
| 2 | `version` が 1 | `ErrSaveVersion` |
| 3 | `rule` が `withdraw`・`permadeath` のどれか | `ErrInvalidSave` |
| 4 | `cleared` が偽なら `chapter` が目録の章の ID。真なら `chapter` が空 | `ErrInvalidSave` |
| 5 | 名簿が 1 人以上で、各キャラが目録にあり、重ならない。主人公がちょうど 1 人いて、生きている | `ErrInvalidSave` |
| 6 | レベルは 1 以上 20 以下。経験は 0 以上 99 以下で、レベル 20 なら 0。能力は HP 1 以上、ほかは 0 以上 | `ErrInvalidSave` |
| 7 | 手持ちと在庫のアイテムが目録にあり、数は 1 以上。手持ちの数の合計は 5 以下。在庫に同じ ID が 2 回ない | `ErrInvalidSave` |

- 誤りの文には、理由と位置（例: `roster[2].level`）を含める
- 定義データの章が減ったなどで今の章がなくなったセーブは、規則 4 で読めない。反復 2 では、読めないセーブは「新規開始」で上書きする（DES-UI-10）

## 7. 画面（ui）

窓の大きさ（832 × 576 ピクセル）、文字、色、素材は反復 1 のまま（DES-UI-01、05）。ボタンは、反復 1 の行動の一覧と同じ形（`menuColor` の下地と `menuBorder` の枠）で描く。

### 7.1 場面の切り替え（DES-UI-09）

- `ui.Run(cfg Config)` で始める。`Config` は、目録（`*defs.Catalog`）、セーブの口（`SaveStore`）、戦闘の種を決める関数（`NextSeed func() uint64`）を持つ
- `SaveStore` は `Exists() (bool, error)`、`Read(func(io.Reader) error) error`、`Write(func(io.Writer) error) error` を持つインタフェース（`savefile.Store` が満たす）。テストでは記憶の中に持つ偽物を使う
- 画面は場面（`Title`・`RuleSelect`・`Story`・`Joined`・`Prepare`・`Battle`・`Result`・`GameOver`・`Clear`）を 1 つずつ持ち、DES-CMP-02 の図のとおりに切り替える
- 各場面の入力と状態の遷移は、反復 1 の `Controller` と同じく、描画から分けた型にしてテストを書ける形にする。`Battle` の場面は反復 1 の `Controller` を使う
- 窓を閉じればいつでも終わる（マップの途中はセーブしない。TBD-10）

### 7.2 タイトル（DES-UI-10。FR-SYS-02）

- 上に「レンキン・タクティクス」を大きく出し、ボタンを 3 つ縦に並べる: 「新規開始」「続きから」「終了」
- 場面に入るとき、セーブを読んでみる（`SaveStore.Read` と `LoadCampaign`）
  - 読めたら、「続きから」の下に「第 N 章　章の名前（離脱）」のように出す。クリア済みなら「クリア済み」と出す
  - セーブがなければ、「続きから」を暗く描き、押せなくする
  - 読めなければ、「続きから」を暗く描いて押せなくし、下に「セーブデータを読めません: 理由」と出す
- 「新規開始」: セーブがある（読めないものを含む）ときは、「セーブを上書きして、はじめから始めますか」と「はい」「いいえ」を出す。「はい」で離脱かロストの選択に進む
- 「続きから」: 章の前の物語から始める。クリア済みならクリアの場面に進む
- 「終了」: 窓を閉じる

### 7.3 離脱かロストの選択（DES-UI-11。FR-CMP-04）

- 「倒れた仲間をどうしますか」と、ボタンを 2 つ出す。ボタンの下に説明を 1 行ずつ出す
  - 「離脱」: 倒れた仲間は、その章では戦えないが、次の章で戻る
  - 「ロスト」: 倒れた仲間は戦死し、二度と戻らない
- 選ぶと `NewCampaign` を呼び、セーブして（DES-SAV-02）、1 章の前の物語に進む
- 右クリックか Esc でタイトルに戻る

### 7.4 物語と加入（DES-STY-01、DES-UI-12。FR-CMP-07、08）

**物語の定義と出し方（DES-STY-01）**

- 物語は `story.toml` に 1 本ずつ書き（ID と段落の並び。DES-DAT-01・02）、章の `before`・`after` が ID で指す。空なら、その物語を出さない
- 章の前の物語は、章に入ったあと（新規開始・続きから・前の章の物語のあと）、章の準備の前に出す。章のあとの物語は、章の結果のあとに出す。負けてやり直したときは、章の前の物語を出さない（DES-CMP-07）
- 物語の文は、誰が生きているかで変えない。そのため、ロストで戦死しうる仲間は、全員が生きている 1 章の前の物語と、加入の場面（2 章の前の物語のエルザ）のほかでは話さず、動かない（10.8 節）
- 1 段落は 200 文字以下（DES-DAT-03 の規則 18）。物語の量は 1 本で数段落まで（CST-03）

**物語の場面（DES-UI-12）**

- 上に「第 N 章　章の名前」を出し、その下に段落を 1 つずつ出す。次の段落に進むと、前の段落を消して入れ替える。右下に「段落の番号/段落の数」を出す
- 段落は、左右に 48 ピクセルの余白をとった幅で、文字の単位で折り返す
- 左クリック・Enter・Space で次の段落に進む。最後の段落の次で、物語の場面を終える
- 右クリックか Esc で、物語の残りを飛ばす
- 直前のセーブに失敗していたら（新規開始のあとのセーブ。DES-SAV-02）、物語の上に「セーブできませんでした: 理由」を出す
- 章の前の物語のあと、その章に加入（`Joins`）があれば、`Joined` の場面で「エルザが仲間に加わった」と、加わったキャラの絵・クラス・レベル・能力を出す。クリックで章の準備に進む。負けてやり直したときは出さない（DES-CMP-07）

### 7.5 章の準備（DES-UI-13。FR-CMP-03、FR-ITM-03）

```text
┌──────────────────────────────────────────┬──────────────────┐
│ 第 2 章　灰の街道　出撃 5/6                 │ （選んだキャラの   │
│ [✓] レオン  ロード    Lv 2  HP 20  手持ち 2 │   情報。情報欄と   │
│ [✓] ガルド  アーマー  Lv 2  HP 25  手持ち 1 │   同じ形）         │
│ [ ] ブラン  ファイター Lv 1  HP 22  手持ち 1 │                  │
│ …                                        │                  │
│ 在庫: 回復薬 ×2                            │                  │
│ [手持ちへ: 回復薬] [在庫へ: 回復薬]          │                  │
│                                [出撃]     │                  │
└──────────────────────────────────────────┴──────────────────┘
```

- 上に「第 N 章　章の名前　出撃 選んだ数/枠の数」を出す
- 名簿の生きているキャラを、名簿の順に 1 行ずつ出す（戦死したキャラは出さない）。行は、出撃の印、ユニットの絵、名前、クラス、レベル、最大 HP、手持ちの数
- 場面に入ったときは、主人公と、名簿の順に枠の数まで、出撃の印を付けておく。主人公を選んだ状態にする（右の欄とアイテムのボタンは主人公のもの）
- 出撃の印をクリックすると、付け外しする。主人公の印は外せない（「主人公は必ず出撃します」と出す）。枠の数を超えて付けようとしたら付けず、「出撃は N 人までです」と出す
- 行の名前をクリックすると、そのキャラを選ぶ（行を明るく描く）。右の欄に、選んだキャラの情報（DES-UI-06 と同じ形。レベルと経験を含む。成長率は出さない。C-02）を出す
- 在庫の行に、在庫のアイテムと数を出す。その下に、選んだキャラについて、在庫の各アイテムの「手持ちへ: 名前」と、手持ちの各アイテムの「在庫へ: 名前」のボタンを出す。押すと `TakeItem`・`StoreItem` を呼ぶ。誤り（`ErrHandFull` など）は、文にして出す（例: 「手持ちは 5 個までです」）
- 「出撃」を押すと、出撃の印のあるキャラを、主人公を先頭にして、あとは名簿の順に並べて `StartBattle` に渡し（種は `NextSeed`）、戦闘の場面に進む
- 出撃するキャラの、枠の中での並びは選べない（名簿の順。10.7 節）

### 7.6 章の結果（DES-UI-14。FR-CMP-10）

- 上に「第 N 章　章の名前　勝利」を出す
- 出撃した各キャラを 1 行ずつ出す: 名前、「Lv 1 → 2」、「経験 80 → 10」、上がった能力（例: 「HP+1 技+1 守備+1」。上がった能力だけ）。倒れたキャラは、離脱なら「離脱（次の章で戻る）」、ロストなら赤で「戦死」と出す
- 下に報酬（例: 「回復薬 ×2 を在庫に入れた」）を出す。報酬がなければ出さない
- セーブに失敗していたら、その文を出す（DES-SAV-02）
- クリックで、章のあとの物語に進む

### 7.7 ゲームオーバーとクリア（DES-UI-15。FR-CMP-02、FR-CMP-09）

- **ゲームオーバー**: 「敗北」「主人公が倒れた」と、ボタン「この章をやり直す」「タイトルへ」を出す。「この章をやり直す」は DES-CMP-07 のとおり（メモリの中身から読む）。読めなかったら、タイトルに戻り、タイトルの下にその理由を出す
- **クリア**: 「クリア」と、生きている仲間の名前とレベルを出す。ロストで戦死した仲間がいれば、「戦死: 名前、…」と出す。クリックでタイトルに戻る

### 7.8 戦闘の画面の変更（DES-UI-16。FR-UNT-04、FR-UNT-09）

- 出来事の再生（DES-UI-04）に、`Exp` と `LevelUp` を足す
  - `Exp`: ユニットのマスの上に「+30 経験」を、出来事の文字と同じ形で出す（20 フレーム）
  - `LevelUp`: 盤面の中ほどに帯を出し、「ブラン　レベル 2　HP+1 技+1 守備+1」（上がった能力だけ。何も上がらなければ「上がった能力なし」）を 60 フレーム出す。早送りが効く
- 情報欄（DES-UI-06）の名前とクラスの下に、自軍なら「Lv 2　経験 40」を出す（Lv 20 なら経験は「―」）。敵（レベル 0）は出さない

### 7.9 戦闘の終わり（DES-UI-17。FR-CMP-02、FR-CMP-09、FR-CMP-10）

- 勝ちが決まったら、反復 1 の勝ち負けの画面の代わりに、「勝利」の帯を出す。クリックで `FinishChapter` を呼び、セーブして（C-07）、章の結果の場面に進む
- 負けが決まったら、「敗北」の帯を出し、クリックでゲームオーバーの場面に進む
- 反復 1 の勝ち負けの画面の R（やり直す）と Esc（終了）はやめる。`Controller` の `Restart`・`Quit` は、戦闘の場面では使わない

### 7.10 回復の操作と予測（DES-UI-18。DES-UI-02・03 の変更。FR-BTL-03、FR-BTL-08、UC-02 の 5c。v0.3）

```mermaid
stateDiagram-v2
  Moved --> HealTargeting: 「回復」
  HealTargeting --> HealTargeting: 回復できる味方をクリック（予測を出す）
  HealTargeting --> Playing: 選んだ味方をもう一度クリック（決定）
  HealTargeting --> Moved: 取り消し
```

- 図は反復 1 の DES-UI-02 に足す部分だけを書いた
- **行動の一覧**: 杖のユニットには「攻撃」を出さない。移動先から杖の射程に、HP が最大でない生きている味方（自分を除く）がいれば「回復」を出す。並びは、回復・アイテム・待機の順（杖でないユニットは反復 1 のまま）
- **回復の相手を選ぶ（`HealTargeting`）**: 回復できる味方のマスを緑の枠で示す。味方をクリックすると、その味方を選んで回復の予測を出し、同じ味方をもう一度クリックすると決定して `Act`（`Heal`）を呼ぶ。操作は攻撃の相手を選ぶとき（`Targeting`）と同じ。右クリックか Esc で `Moved` に戻る。回復できない味方や敵をクリックしても何もしない
- **範囲の色（DES-UI-03 の変更）**: 杖のユニットを選んでいるとき（`Selected`・`Moved`・`HealTargeting`）は、杖の届くマスを、攻撃の赤の代わりに緑で描く。移動できるマスの青は反復 1 のまま
- **回復の予測（情報欄）**: 攻撃の予測の位置に、見出し「回復」（緑）と、「リナ → レオン」「HP 12 → 19」「回復 +7」の行を出す（`HealForecast`。DES-HEL-02）。相性・命中率・「撃破」「危険」の帯は出さない
- **盤面の文字**: 選んだ味方のマスの上に「+7」を、DES-UI-07 の相手の上の文字と同じ形（下地の帯つき）で、緑で出す
- **再生**: `Heal` の出来事は反復 1 のまま（回復された側の上に「+7」）。続く `Exp`・`LevelUp` は DES-UI-16 のとおり
- 案内欄に、`HealTargeting` のときの操作（「回復する味方を選ぶ」「右クリック: 戻る」）を出す

### 7.11 予測の強調（DES-UI-07 の変更。FR-BTL-08、UC-03。v0.3）

E-30 の「相手が死ぬ戦闘の場合、また自分が戦闘で死ぬ場合は　戦闘予測で目立たせてもいいかも」に合わせ、反復 1 の「撃破」「危険」を目立たせる。「後」の HP の求め方と、「撃破」（守る側の「後」が 0）・「危険」（攻撃する側の「後」が 0）の条件は反復 1 のまま変えない。1 撃ずつ順に引き、どちらかが 0 になったら止めるので、「撃破」と「危険」が同時に出ることはない。

- **予測の上の帯**: 攻撃の予測の表の上に、情報欄の幅いっぱい（余白の内側、304 ピクセル）、高さ 40 ピクセルの帯を出す
  - 撃破: 金の下地に、黒で「撃破」（2 倍の大きさ）と「この攻撃で倒せる」（等倍）
  - 危険: 赤の下地に、白で「危険」（2 倍の大きさ）と「反撃で倒される」（等倍）
  - どちらでもなければ帯を出さず、予測の表の位置は反復 1 のまま
  - 帯を出すときは、予測の表とその下の欄を、帯の分（40 ＋ 余白 4 ピクセル）だけ下にずらす
  - 反復 1 の、列の上の小さな橙の「撃破」「危険」は、帯に置き換えてやめる
- **盤面の文字**: 選んだ相手のマスの上の「ダメージ×回数 命中率%」の前に「撃破 」「危険 」を付け、文字を撃破は金、危険は赤で描く（例: 「撃破 7×2 89%」）。どちらでもなければ反復 1 のまま白
- **危険のときの枠**: 危険のときは、攻撃する側の移動先のマスにも、赤い枠を描く（自分が倒される位置を盤面でも見せる）
- **回復と重ならない**: 帯と色つきの文字は攻撃の相手を選んでいるとき（`Targeting`）だけ出す。回復の相手を選んでいるとき（`HealTargeting`）は攻撃の予測を作らないので出ない（7.10 節）
- 帯を出したときも、情報欄の中身が画面の高さ（576 ピクセル）に収まることを、`InfoView` の位置のテストで確かめる。収まらないときは、実装の担当が設計への提案として報告する
- `Panel` に、撃破・危険・なしのどれかを持つ欄を足し、`InfoView` が帯の位置と文を決める（描画から分けてテストできる形。反復 1 の DES-UI-06・07 と同じ）

## 8. 起動・非機能・対応

### 8.1 起動の流れ（DES-CMD-02。DES-CMD-01 を置き換える）

1. 定義データを読む（DES-CMD-01 の 1 のまま）
2. 乱数の種を決める関数を作る（DES-RNG-01 のまま。`-seed` を指定すれば、戦闘を作るたびに同じ種）
3. セーブの置き場所を決める（DES-SAV-02 の表）。`-savedir` がなく、置き場所を決められない（`os.UserConfigDir()`・`os.UserHomeDir()` が失敗する）ときは、理由を標準エラーに書いて終了コード 1 で終える
4. `ui.Run(ui.Config{...})` でタイトルから始める

### 8.2 非機能

| ID | 内容 | 要求 |
|---|---|---|
| DES-NFR-01（反復 1 のまま） | `internal/core` 以下が Ebitengine と `time` を import しないことを確かめるテストは、キャンペーンとセーブの読み書きにも効く | NFR-04、NFR-05 |
| DES-SAV-02 | 一時ファイルに書いてから名前を変える。テストでは、`fn` が途中で誤りを返したとき・一時ファイルが残っているときに、前の `save.json` が変わらず読めることを確かめる | NFR-08 |
| DES-RNG-01（反復 2 で追記） | 成長の乱数も戦闘の乱数から引く。同じ種と同じコマンドの列なら、経験・レベル・成長も同じ | NFR-06 |

### 8.3 設計と SRS の対応

反復 2 で足す・変える行だけを書く。ほかは反復 1 の設計書 8 章のまま。

| 要求 | 設計 |
|---|---|
| FR-CMP-01 | DES-CMP-01、DES-CMP-02、DES-CMP-06、DES-DAT-01〜03、10.4 節 |
| FR-CMP-02 | DES-CMP-06、DES-UI-15、DES-UI-17 |
| FR-CMP-03 | DES-CMP-04、DES-CMP-05、DES-UI-13 |
| FR-CMP-04 | DES-CMP-01、DES-CMP-03、DES-UI-11 |
| FR-CMP-05、FR-CMP-06 | DES-CMP-05、DES-CMP-06、DES-CMP-07 |
| FR-CMP-07 | DES-STY-01、DES-UI-12、DES-DAT-01〜03 |
| FR-CMP-08 | DES-CMP-03、DES-UI-12 |
| FR-CMP-09 | DES-MAP-04（反復 1 のまま）、DES-CMP-07、DES-UI-15、DES-UI-17 |
| FR-CMP-10 | DES-CMP-06、DES-UI-14、DES-UI-17 |
| FR-UNT-04 | DES-GRW-01、DES-GRW-02、DES-GRW-03、DES-UI-16 |
| FR-UNT-05 | DES-GRW-02、DES-DAT-01 |
| FR-UNT-09 | DES-UI-16（レベルと経験）、DES-UI-13 |
| FR-ITM-03 | DES-CMP-04、DES-CMP-06（報酬） |
| FR-SYS-01 | DES-SAV-01、DES-SAV-02、DES-SAV-03 |
| FR-SYS-02 | DES-UI-09、DES-UI-10 |
| FR-SYS-03、FR-SYS-04 | DES-DAT-01〜03（反復 2 で変更） |
| IF-01 | DES-UI-09〜17 |
| IF-04 | DES-SAV-02、DES-CMD-02 |
| NFR-06 | DES-RNG-01（反復 2 で追記）、DES-GRW-02 |
| NFR-07 | DES-CMP-08、DES-SAV-03 |
| NFR-08 | DES-SAV-02 |
| FR-BTL-03（v0.3） | DES-HEL-01（行動「回復」）、DES-UI-18 |
| FR-BTL-06（v0.3） | DES-HEL-01（杖は反撃しない） |
| FR-BTL-08（v0.3） | DES-HEL-02（回復の予測）、DES-UI-07（反復 2 で変更。撃破・危険の強調）、DES-UI-18 |
| FR-UNT-02（v0.3） | 10.1 節（僧侶）、DES-DAT-03 の規則 4・21・22 |
| FR-UNT-03（v0.3） | 10.3・10.9 節（役割の数値。検証は D で、振り返りに記録する） |
| FR-UNT-04（v0.3） | DES-GRW-01（回復の経験） |
| FR-ITM-01（v0.3） | DES-HEL-01。杖は攻撃に使わない武器で、FR-ITM-01 の「武器を装備して攻撃する」の例外になる（計画書 9.4 節） |

## 9. 実装の分け方（Issue の案）

設計の承認のあとで、次の Issue を切る（ラベル `it02`、マイルストーン `IT02`）。番号は表の中の番号で、GitHub の Issue 番号とは別。

| # | Issue | 要求・設計 | 依存 |
|---|---|---|---|
| 1 | 戦闘の経験と成長（`tactics`） | FR-UNT-04、05、NFR-06 / DES-GRW-01〜03、DES-RNG-01 の追記 | なし |
| 2 | 定義データ: 成長率・レベル・騎兵・章・物語・出撃の枠・敵の名前と、本体のデータ（2 章・3 章のマップ、物語の文）。`core.NewBattle` の自軍の置き方と、`cmd/renkin` の始めるマップ。（v0.3）杖の定義データ（武器の種類 `staff`、`heal`、規則 4・6・21・22）と、自軍の役割の数値（10.3 節のボーナスと成長率。出撃の枠は 1 章 5・2 章 6・3 章 5）。僧侶・杖・リナの本体のデータは 10 で足す | FR-SYS-03、04、IF-03、FR-CMP-01、07、08、FR-UNT-02、03 / DES-DAT-01〜03（反復 2 の変更）、10 章、DES-CORE-01 のうち自軍の置き方（`party` と `slots`） | なし |
| 3 | キャンペーンの状態と流れ（`core`） | FR-CMP-01〜06、08、10、FR-ITM-03、FR-UNT-04 / DES-CMP-01〜08（セーブを除く）、DES-CORE-01 のうちレベル・成長率・敵の名前 | 1、2 |
| 4 | セーブのファイル（`internal/savefile`） | IF-04、NFR-08 / DES-PKG-08、DES-SAV-02 | なし |
| 5 | セーブの中身の読み書き（`core`） | FR-SYS-01、NFR-07 / DES-SAV-01、DES-SAV-03 | 3 |
| 6 | 画面: 場面の切り替え・タイトル・離脱かロスト・物語と加入・ゲームオーバー・クリア・戦闘の終わりと、起動 | FR-SYS-02、FR-CMP-02、04、07、08、09、IF-01、02、IF-04 / DES-STY-01、DES-UI-09〜12、DES-UI-15、DES-UI-17、DES-CMD-02 | 4、5 |
| 7 | 画面: 章の準備と章の結果 | FR-CMP-03、10、FR-ITM-03 / DES-UI-13、DES-UI-14 | 6 |
| 8 | 画面: 戦闘中の経験・レベルアップの再生と、情報欄のレベル | FR-UNT-04、09 / DES-UI-16 | 1、6 |
| 9（v0.3） | 回復の行動と予測（`tactics`）: `Staff`・`Weapon.Heal`・`Action` の `Heal`、`Setup` の規則 8、杖は反撃しない、回復の経験、`HealForecast`、誤り 2 つ | FR-BTL-03、06、08、FR-UNT-04、NFR-06、NFR-07 / DES-HEL-01、DES-HEL-02、DES-GRW-01（回復の経験） | 1 |
| 10（v0.3） | 僧侶リナの本体のデータ（クラス `cleric`、癒やしの杖、`party` にリナ、1 章の前の物語の段落）と、`core` の武器の変換（`staff` → `tactics.Staff`、`heal`） | FR-UNT-02、FR-SYS-03、FR-CMP-07 / 10.1・10.2・10.3・10.8 節、DES-CORE-01（v0.3） | 2、9 |
| 11（v0.3） | 画面: 回復の操作と予測（行動の一覧の「回復」、`HealTargeting`、緑の範囲、回復の予測、盤面の「+7」） | FR-BTL-03、FR-BTL-08、IF-01 / DES-UI-18 | 6、9 |
| 12（v0.3） | 画面: 予測の強調（撃破・危険の帯、盤面の色つきの文字、危険の枠） | FR-BTL-08 / DES-UI-07（反復 2 で変更） | なし |

- 1・2・4 は並べて進められる。3 は 1 と 2 のあと。5 は 3 のあと。6 は 4 と 5 のあと。7 と 8 は 6 のあと並べて進められる
- （v0.3）数値の付け替え（役割を尖らせる）は、データの Issue 2 に入れた。回復は、`tactics` の 9（1 のあと。経験の仕組みを使うので）、本体のデータと `core` の 10（2 と 9 のあと）、画面の 11（6 と 9 のあと）に分けた。予測の強調 12 は、反復 1 の `Forecast` だけを使うので依存がなく、ほかの Issue より先に進めてよい（開発者が早く遊んで確かめられる）
- 3 と 10 はどちらが先でもよい。本体のデータを使う `core` のテスト（自軍の人数）は、後からマージする方が合わせる
- `internal/core/tactics` を触る 1 と 9 は、9 を 1 のあとにして衝突を避ける。`internal/ui` を触る 6・7・8・11・12 は衝突しやすい。12 は小さいので先にマージし、後の Issue が合わせる
- 2 は `start_map` をやめるので、`cmd/renkin` の起動（今は `cat.Game.StartMap` で `core.NewBattle` を呼ぶ）を、「`chapters.toml` の最初の章のマップで `core.NewBattle`」に変える。6 で DES-CMD-02 に置き換えるまでの仮の形
- 6 は `ui.Run` を `ui.Run(cfg Config)` に変え、戦闘の終わり（DES-UI-17）の帯と遷移、`FinishChapter` とセーブの呼び出しを入れ、R・Esc をやめる。6 の間は、章の準備と章の結果を仮の場面で通す（章の準備は、名簿の順に枠の数まで全員が出撃し、クリックで戦闘に進む。章の結果は `ChapterResult` を文で並べるだけ）。7 でこれを置き換える
- 8 は、経験とレベルアップの出来事の再生（DES-UI-16）と、情報欄のレベルと経験だけを行う
- `internal/ui` を触る 6・7・8 は衝突しやすい。7 と 8 がどちらも情報欄（`info.go`）を変えるときは、先にマージした方に合わせる
- テストケースは、実装の担当がテスト仕様書（`docs/04_test/test-spec-it02.md`。骨組みはこの設計のプルリクエストで作った）に書き足す（implement-req の手順 3）

**書き換える反復 1 のテスト**（テストケース ID は変えず、テスト仕様書 `test-spec-it01.md` の該当の行も直す）

| Issue | 書き換えるテスト | 理由 |
|---|---|---|
| 1 | なし（見込み） | `Level` が 0 の自軍は経験を得ず、`Event`・`Unit` に足す欄はゼロ値なので、反復 1 の `tactics` のテストは変わらない |
| 2 | TC-FR-SYS-03-01（`internal/defload/base_test.go` の `StartMap` と `Allies`） | `start_map`・`allies` をやめ、`party`・`slots`・章・物語を足す |
| 2 | TC-FR-SYS-04-03（始めるマップ・自軍の配置のキャラの参照）、TC-FR-SYS-04-13（自軍の座標）、TC-FR-SYS-04-15（規則 10）、TC-FR-SYS-04-20（`allies` の知らないキー）（`internal/defload/load_test.go`） | 規則 3・9・10 の変更と、`allies` をやめること |
| 2 | TC-FR-SYS-04-22・23、TC-IF-01-17（`cmd/renkin` のテスト）、TC-IF-01-21（手動。`start_map` を壊す手順） | `cmd/renkin` の始めるマップを、最初の章のマップにする |
| 2 | TC-FR-UNT-01-01、TC-FR-UNT-01-05（`internal/core/battle_test.go` の小さな目録の `GameDef{StartMap}`・`Allies`・`defs.NewCatalog` の引数） | `GameDef`・`MapDef` の欄と、`defs.NewCatalog` の引数（章・物語）が変わる |
| 3 | TC-FR-UNT-01-01（自軍の `UnitSetup` の `DeepEqual`） | 自軍の `UnitSetup` に `Level`・`Growth` が入る |
| 3 | TC-FR-UNT-01-02、TC-FR-UNT-01-03、TC-FR-SYS-03-05（`internal/core/battle_test.go` の 120・132・264 行あたり） | 1 章のボスに名前（ハーゲン）を付けると、名前のない敵だけに番号を付けるので、増援のソルジャーが「ソルジャー 4」から「ソルジャー 3」になる。ボスの名前も変わる |
| 6 | TC-IF-01-13、TC-IF-01-14（`internal/ui/input_test.go`）、TC-IF-01-20（`internal/ui/enemyrange_test.go` の `Restart`） | 勝ち負けの画面の R・Esc をやめ、戦闘の終わりの遷移にする |
| 6 | TC-FR-SYS-04-22・23、TC-IF-01-17（2 で直したもの） | `ui.Run(cfg Config)` と DES-CMD-02 に変える |
| 2（v0.3） | TC-FR-SYS-03-01（`internal/defload/base_test.go` のキャラのボーナス）、TC-FR-UNT-01-01（`internal/core/battle_test.go` の自軍の能力） | レオン・ガルド・ブランのボーナスを 10.3 節の値にする |
| 2（v0.3） | TC-FR-SYS-04-04（`internal/defload/load_test.go` の武器の種類の誤りの文） | 誤りの文の「sword・lance・axe・bow のどれか」に `staff` が入る |
| 9（v0.3） | TC-NFR-07-01・02・03（`internal/core/tactics/act_test.go` の `Action(3)`）、TC-FR-BTL-08-03（`attack_test.go` の `Action(3)`）、TC-FR-BTL-01-10（`battle_test.go` の `Bow + 1`） | `Action(3)` は `Heal`、`Bow + 1` は `Staff` になり、正しい値になる。範囲外の例を `Action(4)`・`Staff + 1` に変える |
| 10（v0.3） | TC-FR-UNT-01-01・02・03、TC-FR-UNT-02-01、TC-FR-SYS-03-05、TC-FR-ITM-03-01（`internal/core/battle_test.go`）、TC-FR-SYS-03-01（`base_test.go`） | `party` にリナが入り、1 章の出撃の枠が 5 なので、自軍が 5 人になり、敵の添字が 1 つずれる。クラスと武器に僧侶・杖が入る |
| 12（v0.3） | TC-FR-BTL-08-09・15・17（`internal/ui/forecast_test.go` の「撃破」「危険」の位置と色）、TC-FR-BTL-08-13（盤面の文字に「撃破 」「危険 」が付く）、TC-FR-UNT-09-12（`info_view_test.go`。帯を足しても情報欄に収まる） | 列の上の小さな印をやめて帯に置き換え、盤面の文字の頭と色を変える |

## 10. この設計で仮置きした事項

数はすべて Claude Code の案で、遊んで調整する（IT02-PLAN 5 章）。数値は、スクリプト（反復 1 の 3.5 節の式で、能力・ダメージ・命中率・経験・成長を計算し、マップの配置を調べるもの）で検算した。

### 10.1 クラス（足すもの）

| ID | 名前 | 武器 | HP | 力 | 技 | 速さ | 守備 | 移動 | 使う陣営 |
|---|---|---|---|---|---|---|---|---|---|
| cavalier | 騎兵 | 槍 | 20 | 6 | 5 | 6 | 5 | 7 | 自軍・敵 |
| cleric（v0.3） | 僧侶 | 杖 | 16 | 0 | 2 | 6 | 2 | 5 | 自軍 |

- 反復 1 で自軍だけに使ったアーマーを、3 章では敵にも使う（10.6 節）
- （v0.3）反復 1・反復 2 v0.2 のクラス（ロード〜騎兵）の `base` は変えない。ファイター・アーチャー・アーマー・騎兵は敵も使うので、`base` を変えると敵の能力が変わる（E-32 で、敵は変えないと決まった）。自軍の役割は、10.3 節のキャラのボーナスと成長率で尖らせる（C-14）
- 僧侶の力と技は、杖が攻撃しないので使わない（力 0）。HP と守備は自軍でいちばん低く、速さ（回避）だけで身を守る。後ろに置いて守る役
- 武器（足すもの）: 癒やしの杖（`heal_staff`）。種類 杖、回復 10、射程 1〜1、威力と命中は 0（使わない）。回復の量は回復薬（10）と同じにし、能力では増やさない（C-13）。名前は錬金術の工房の品として付けた
- 反復 1 には、クラスごとに地形の費用を変える規則がないので、騎兵も森で 2、山で 3 を使う。移動の長さだけが強み（C-04）
- 絵は、Kenney の素材から実装のときに選び、`internal/ui/assets/README.md` に書く

### 10.2 経験と成長

| 事項 | 仮置き | 理由 |
|---|---|---|
| 経験（C-01） | 倒す 30、当てる 10、それ以外 1。100 で 1 レベル | FE に近い形で、いちばん単純。敵とのレベル差は使わない（敵はレベルを持たない） |
| レベルの上限 | 20 | FE と同じ。反復 2 では届かない |
| 成長の判定 | 能力ごとに乱数 1 つ、成長率より小さければ ＋1 | FE と同じ |
| 経験の目安 | 敵をすべて倒し、倒さない当たりを 1 体 1.5 回とすると、1 章 490、2 章 490、3 章 580、計 1560。（v0.3）戦う出撃の人数（リナを除く）は 1 章 4、2 章 5、3 章 4 なので、1 人 約 360（＋3.5 レベル）。リナは回復 1 回で 10 なので、1 章で 6 回回復すると 60、3 章で 約 180（＋2 レベル弱） | スクリプトで計算した。1 章ごとに 1 レベルくらい上がる。僧侶は遅れて育つ |
| 回復の経験（v0.3、C-13） | 回復 1 回で 10 | 攻撃を当てたときと同じ。回復する場面は限られるので、多すぎない |

### 10.3 自軍のキャラ（成長率と加入）

| ID | 名前 | クラス | 加わる | レベル | 成長率（HP/力/技/速さ/守備 %） | 能力（HP/力/技/速さ/守備/移動） | 武器 | 手持ち | スキル |
|---|---|---|---|---|---|---|---|---|---|
| leon | レオン（主人公） | ロード | 初め | 1 | 55/45/65/70/20 | 19/7/11/12/4/5（ボーナス 1/2/4/4/0） | 鉄の剣 | 回復薬 × 2 | 鼓舞 |
| gald | ガルド | アーマー | 初め | 1 | 85/40/20/10/70 | 24/6/1/2/11/4（ボーナス 2/0/−2/0/2） | 鉄の槍 | 回復薬 × 1 | 大盾 |
| bran | ブラン | ファイター | 初め | 1 | 70/75/35/30/15 | 22/11/5/5/2/5（ボーナス −2/4/1/1/−1） | 鉄の斧 | 回復薬 × 1 | 怒り |
| mira | ミラ | アーチャー | 初め | 1 | 55/40/60/45/20 | 17/5/9/7/3/5（ボーナス 0/0/2/1/0） | 鉄の弓 | 回復薬 × 1 | 狙撃 |
| rina（v0.3） | リナ | 僧侶 | 初め | 1 | 55/0/40/60/30 | 16/0/2/6/2/5（ボーナスなし） | 癒やしの杖 | 回復薬 × 1 | なし |
| elsa | エルザ | 騎兵 | 2 章の始め | 2 | 65/45/40/40/35 | 21/7/6/7/5/7（ボーナス 1/1/1/1/0） | 鉄の槍 | 回復薬 × 1 | なし（C-04） |

- （v0.3）v0.2 では「反復 1 の 4 人の能力は変えない」としたが、E-29・E-30 を受けて、レオン・ガルド・ブランのボーナスと成長率を変えた（v0.2 の値: レオン 20/6/8/9/4・成長率 70/40/50/55/30、ガルド 24/6/3/2/10・80/45/30/20/55、ブラン 26/8/4/5/3・85/55/30/35/20）。ミラとエルザは変えない
- 役割（E-30）: **ロード**は技と速さを上げ（高命中・高速）、HP を 1 下げた（紙耐久。守備は 4 のまま。主人公が倒れると負けなので、守備まで下げると 1 章のソルジャー 2 撃で倒れる）。**ファイター**は力を上げ（高火力）、HP と守備を下げた（低耐久）。**アーマー**は守備を上げ、技を下げた（命中が不安）。火力（力 6）は中のまま
- ボーナスに負の値を使った（ガルドの技、ブランの HP と守備）。規則 6 は足したあとの値を見るので、検証の規則は変えない（C-14）
- 成長率も役割に合わせて尖らせた（レオンは技・速さ、ブランは力、ガルドは HP・守備、リナは速さ。レオンの HP と守備、ブランの守備、ガルドの技と速さは低い）。合計は 220〜255 %（リナは力を使わないので 185 %）
- リナ（v0.3）はアルケアの工房の錬金術士の見習い。1 章の砦に薬を届けに来ていて、隊に加わる。プロット（CON-001 3 章）の 3 章「工房の錬金術士を集めて」につながる。名前は Claude Code が付けた（E-07、C-16）
- エルザはプロットの「帝国の側にも、戦争に疑問を持つ将がいて、途中で味方になる」（CON-001 3 章）の将。名前は Claude Code が付けた（E-07）

### 10.4 章の並び

| 章 | ID | 名前 | マップ | 出撃の枠（C-05） | 加入 | 報酬 | ボス |
|---|---|---|---|---|---|---|---|
| 1 | ch01 | 国境の砦 | `ch01`（反復 1 のもの） | 5（v0.3。v0.2 は 4） | — | 回復薬 × 2 | ハーゲン（ソルジャー。反復 1 のボス） |
| 2 | ch02 | 灰の街道 | `ch02`（10.5 節） | 6（v0.3。v0.2 は 5） | エルザ | 回復薬 × 2 | ヴォルグ（騎兵） |
| 3 | ch03 | 王都の奪還（C-03） | `ch03`（10.6 節） | 5（v0.3。v0.2 は 4） | — | なし | グラウス（剣士） |

- `game.toml`: `party = ["leon", "gald", "bran", "mira", "rina"]`（v0.3）、`stock` なし（在庫は空から始まる）
- 物語の ID は `ch01_before`・`ch01_after` のように、章の ID に `_before`・`_after` を付ける（10.8 節）
- 1 章の出撃の枠は、反復 1 の自軍の配置と同じ座標を同じ順に並べ、5 番目に (2, 15) を足す（v0.3）: (1, 14)、(2, 13)、(3, 14)、(1, 15)、(2, 15)。名簿の順に出撃すれば、反復 1 の盤面にリナを足した形になる（ボスの名前と、増援のソルジャーの番号を除く。9 章）
- （v0.3、C-12）リナが 1 章から名簿にいるので、出撃の枠を 1 つずつ増やした。3 章で、名簿が 6 人なら、出撃する 5 人を選ぶ（v0.2 の「5 人から 4 人を選ぶ」場面を保つ）
- 敵は章が進むごとに、ボーナスで強くする: 2 章の敵（ボスを除く）は 2/1/1/1/1、3 章の敵（ボスを除く）は 3/2/1/1/1（HP/力/技/速さ/守備）。（v0.3）敵の能力は E-32 で変えないと決まったので、v0.2 のまま。自軍の目安（2 章で Lv 2、3 章で Lv 3）での数字は 10.9 節に書いた（例: 3 章で Lv 3 のレオンがソルジャーに 7 ダメージ × 2 回・命中率 89、ソルジャーからレオンに 11 ダメージ・命中率 79。v0.2 の値では 6 × 2 回・83 と 10・85）

### 10.5 マップ「灰の街道」（`ch02`。撤退戦）

退く自軍の前を、帝国の騎兵がふさぐ。後ろ（西）からは追っ手が増援で来るので、前に出て隊長を討つ。

```text
   x 0123456789012345
y 0  ^^^^^FF^^^^^^^^^
  1  ^^^....F...^^^^^
  2  ^^..FF......^^^^
  3  ^...FF..^^....^^
  4  ......^^^^..FF.^
  5  ..FF..........F.
  6  .....FF...^^....
  7  ................
  8  ...........FF.T.
  9  ..^^..FF........
 10  ..^^.....^^^..FF
 11  .......FF^^.....
 12  ^^...........^^^
 13  ^^^^..FF..^^^^^^
 14  ^^^^^.....^^^^^^
 15  ^^^^^^^^^^^^^^^^
```

| 陣営 | ユニット | 座標 | 動き方 |
|---|---|---|---|
| 出撃の枠 | 1〜6 | (1, 7)、(1, 6)、(1, 8)、(0, 6)、(0, 8)、(2, 7)（6 番目は v0.3） | — |
| 敵 | ソルジャー | (8, 5) | 突っ込む |
| 敵 | ファイター | (7, 9) | 突っ込む |
| 敵 | アーチャー | (9, 3)（山の上） | 待ち構える |
| 敵 | 剣士 | (10, 7) | 待ち構える |
| 敵 | 騎兵 | (12, 11) | 待ち構える |
| 敵 | ソルジャー | (12, 6) | 待ち構える |
| 敵 | アーチャー | (13, 9) | 待ち構える |
| 敵 | ボス: ヴォルグ（騎兵。ボーナス 8/2/3/2/1、能力 28/8/8/8/6/7） | (14, 8)（砦の上） | 待ち構える |
| 増援 | ソルジャー | 2 ターン目に (0, 4) | 突っ込む |
| 増援 | ファイター | 3 ターン目に (0, 10) | 突っ込む |
| 増援 | 騎兵 | 4 ターン目に (0, 7) | 突っ込む |

- 敵のボーナスは 10.4 節（ボスを除き 2/1/1/1/1）
- ボスの守備は、砦の上でもダメージが通る値にした。自軍の目安（Lv 2、エルザ Lv 3）で、レオン 3 × 2 回・ガルド 5・ブラン 13・ミラ 3・エルザ 6 のダメージが通り、砦の回復（5）を上回る（v0.3 の値。v0.2 ではレオン 2・ブラン 10）
- スクリプトで確かめたこと: 16 行がすべて 16 文字。出撃の枠・敵・増援の座標はすべて通れる地形にあり、枠と敵は重ならない。増援の座標は枠と敵に重ならない。通れるマス（256）はすべてつながっている。1 ターン目の初めの位置では、どの敵の攻撃も出撃の枠（v0.3 で足した (2, 7) を含む）に届かない。主人公の枠からボスまでの移動の費用は 14

### 10.6 マップ「王都の奪還」（`ch03`）

崩れた城壁の外（南）から入り、二重の城壁の切れ目を抜けて、玉座の間（北の砦）の総大将を討つ。

```text
   x 0123456789012345
y 0  ^^^############^
  1  ^^^#....T.....#^
  2  ^^^#..........#^
  3  ^^^###.####.###^
  4  ^^..F........F^^
  5  ^...F..T..T..F.^
  6  ................
  7  ####.######.####
  8  ..F............F
  9  ..FF...^^...FF..
 10  ......^^^^......
 11  ....F......F....
 12  FF............FF
 13  FF..^^....^^..FF
 14  ................
 15  ................
```

| 陣営 | ユニット | 座標 | 動き方 |
|---|---|---|---|
| 出撃の枠 | 1〜5 | (7, 14)、(6, 15)、(8, 15)、(7, 15)、(8, 14)（5 番目は v0.3） | — |
| 敵 | ソルジャー | (4, 8)（外の城壁の切れ目の前） | 待ち構える |
| 敵 | アーチャー | (11, 9) | 待ち構える |
| 敵 | ファイター | (12, 8) | 突っ込む |
| 敵 | 剣士 | (4, 6)（外の城壁の切れ目の内） | 待ち構える |
| 敵 | ソルジャー | (11, 6)（外の城壁の切れ目の内） | 待ち構える |
| 敵 | アーチャー | (7, 5)（砦の上） | 待ち構える |
| 敵 | ファイター | (10, 5)（砦の上） | 待ち構える |
| 敵 | アーマー | (6, 3)（内の城壁の門） | 待ち構える |
| 敵 | ソルジャー | (11, 3)（内の城壁の門） | 待ち構える |
| 敵 | ボス: グラウス（剣士。ボーナス 10/4/3/2/3、能力 27/8/11/11/5/5） | (8, 1)（砦の上） | 待ち構える |
| 増援 | 騎兵 | 3 ターン目に (0, 14)、(15, 14) | 突っ込む |
| 増援 | ファイター | 5 ターン目に (4, 2)（城の中） | 突っ込む |

- 敵のボーナスは 10.4 節（ボスを除き 3/2/1/1/1）
- 門のアーマー（能力 25/8/4/3/10/4）は、斧のブランなら 11 ダメージ通るが、剣のレオンは 2 × 2 回、弓のミラは 2 × 2 回しか通らない（v0.3 の値、自軍の目安 Lv 3）。剣士のボスには、槍のガルド（8）・エルザ（9）がよく効く。キャラの強み（相性と役割）を使い分ける場面にした
- ボスは砦の上で、自軍の目安（Lv 3、エルザ Lv 4）で、レオン 6・ガルド 8・ブラン 12（命中率 25。ボスの反撃は 12 × 2 回で、ブランは倒される）・ミラ 5・エルザ 9 のダメージが通り、砦の回復（5）を上回る（v0.3 の値）
- スクリプトで確かめたこと: 16 行がすべて 16 文字。座標はすべて通れる地形にあり、枠と敵は重ならない。増援の座標は枠と敵に重ならない。通れるマス（216）はすべてつながっている。1 ターン目の初めの位置では、どの敵の攻撃も出撃の枠（v0.3 で足した (8, 14) を含む）に届かない。主人公の枠からボスまでの移動の費用は 20

### 10.7 そのほか

| 事項 | 仮置き | 理由 |
|---|---|---|
| 手持ちの上限 | 数の合計で 5 | FE と同じくらい。在庫と入れ替える意味を残す |
| 出撃の枠の中の並び | 主人公が 1 番目、あとは名簿の順。プレイヤーは選べない | いちばん単純。枠は近くにまとめて置く |
| 敵の能力の上げ方 | 章ごとのボーナス（10.4 節） | 敵はレベルを持たない（FR-UNT-04 の「敵の扱いは設計で決める」への答え） |
| 成長率を画面に出すか（C-02） | 出さない | 「強みを見つける」（E-03）を、上がった能力から遊んで見つけるため |
| ロストで倒れた仲間の手持ち（C-08） | 一緒に失う | FE と同じ。ロストの重さを残す |
| 負けたときのやり直し（C-06） | 章の準備から | 同じ物語を何度も読まないため |
| セーブの時機（C-07） | 「勝利」の帯をクリックした直後（章の結果を見る前） | 結果や物語を見ている間に終えても、勝ちが消えない。帯をクリックする前に終えると勝ちは残らない（ほかの案: 勝ちが決まった時点で `FinishChapter` とセーブをする） |
| セーブの置き場所 | Windows・macOS は `os.UserConfigDir()` の下、Linux は `XDG_DATA_HOME`（既定 `~/.local/share`）の下（DES-SAV-02） | IF-04 の「OS の慣例に従う」に合わせた。Linux の慣例ではデータは `~/.local/share` に置く（`~/.config` は設定の場所） |
| 物語の 1 段落の長さ | 200 文字以下 | 1 画面に収まり、量を多くしない（CST-03） |
| 物語の出し方 | 1 段落ずつ、前の段落と入れ替える | クリックで進める（管理役の案）。1 画面に 1 段落なので、長さの心配がない |
| 1 章の物語の舞台（C-09） | 「砦の奪還」 | 反復 1 のマップは、砦にボスがいて自軍が攻める形なので、プロットの「砦の防衛」ではなく奪還として書いた。3 章の舞台は、プロットの章のうち撤退戦から王都の奪還に進み、山道の突破と補給線の襲撃は飛ばした（C-03） |
| 敵の能力（v0.3） | v0.2 のまま（クラスの `base` もマップの敵のボーナスも変えない） | E-32「敵はそのままでいい」「難しい方が嬉しい」。全体を易しくしない |
| 役割の付け方（v0.3、C-14） | キャラのボーナス（負の値も使う）と成長率で尖らせる。クラスの `base` は変えない | 敵と共有するクラスの `base` を変えると、敵の能力が変わる |
| リナが加わる時期（v0.3、C-11） | 1 章の初めから名簿にいる | 回復役がいる遊び方を 1 章から確かめる。1 章の物語（全員が生きている）で、地の文だけで出せる |
| 回復の量（v0.3、C-13） | 杖の `heal`（10）だけ。能力では増やさない | いちばん単純。回復薬と同じ量で、比べやすい |
| 予測の強調（v0.3、C-15） | 予測の上の帯と、盤面の相手の上の色つきの文字 | 情報欄と盤面のどちらを見ていても気づける |

### 10.8 物語の文（下書き）

Claude Code が、CON-001 3 章のプロット（E-08）に沿って書いた下書き。**開発者の確認を待っている**（E-07）。実装の担当は、これを `story.toml` にそのまま写す。1 行が 1 段落。

**ch01_before（第 1 章　国境の砦の前）**（全員が生きているので、仲間も話す）

1. 小国アルケアは、錬金術の工房で知られる国だった。山と森に守られ、長く戦を知らなかった。
2. その春、大国ヴァルト帝国が国境を越えた。狙いは、工房の技と、山の奥に眠る鉱石だという。
3. 国境の守備隊を率いる若い隊長レオンは、見回りから戻る途中で、砦に帝国の旗が上がるのを見た。砦に残っていた兵の多くは、もう生きていない。
4. 隊には、砦へ薬を届けに来ていた工房の見習い、リナもいた。癒やしの杖を握る手は震えていたが、引き返すとは言わなかった。（v0.3 で足した段落）
5. ガルド「隊長、砦の守りは固い。だが、取り返さねば、この先の村が焼かれる」
6. レオン「行こう。守将ハーゲンを討てば、先遣隊は崩れる」

**ch01_after（第 1 章のあと）**

1. 砦の上に、再びアルケアの旗が上がった。だが、喜ぶ者はいなかった。中庭には、名も知らぬ兵たちが並べて寝かされていた。
2. 夜、王都から早馬が来た。帝国の本隊が別の道から王都に迫り、王都は三日ともたずに落ちたという。
3. 若い兵が「王様は……」とつぶやいた。使者は首を振った。
4. 生き残った兵と工房の錬金術士たちは、散り散りに西の谷へ逃れている。
5. レオン「砦を捨てる。ここに残れば、帝国の本隊に飲まれるだけだ。西へ退き、兵を集め直す」

**ch02_before（第 2 章　灰の街道の前）**（加入の場面。エルザは話す）

1. 西の谷へ向かう街道は、逃げる人々で埋まっていた。後ろからは帝国の追っ手が迫り、前では帝国の騎兵が道をふさいでいる。
2. その夜、一人の帝国の騎士が、白い布を掲げて陣を訪れた。帝国の騎兵隊長、エルザと名乗った。
3. エルザ「村を焼けという命令に、従えなかった。この戦は、錬金術の技を奪うための略奪だ。剣を向ける相手を、選び直したい」
4. 古参の兵「信じろってのか。昨日まで俺たちを追っていた奴を」
5. レオン「信じるかどうかは、明日の戦いで決める。道をふさぐ隊長ヴォルグを討つ。力を貸してくれ」

**ch02_after（第 2 章のあと）**

1. ヴォルグが倒れると、帝国の騎兵は散っていった。逃げる人々の列は、ようやく谷の奥へと流れ始めた。
2. 倒れた帝国の騎兵たちも、もとはどこかの村の若者だった。レオンは、敵の亡骸にも布をかけさせた。
3. 谷には、各地から逃れた兵と錬金術士が集まっていた。工房の長は、残った薬と道具をすべて差し出すと言った。
4. レオン「王都を取り戻す。侵攻軍の総大将を討てば、帝国はこの国にとどまる理由を失う」

**ch03_before（第 3 章　王都の奪還の前）**

1. 集まった兵は、守備隊を合わせても百に満たなかった。王都の城門の前には、その何倍もの帝国兵が陣を敷いている。
2. 捕らえた帝国兵の話では、総大将グラウスは城の奥の玉座の間にいる。自ら剣を取る男で、近づけば必ず出てくるという。
3. レオン「数では勝てない。狙うのは一人だ」
4. 夜明けとともに、レオンたちは崩れた城壁の隙間から王都に入った。

**ch03_after（第 3 章のあと）**

1. グラウスが倒れると、王都の帝国兵は武器を捨てた。総大将を失った侵攻軍は、国境の向こうへ退いていった。
2. 王都の広場には、生き残った人々が集まり、亡くなった者の名を一人ずつ読み上げた。その声は、日が暮れても終わらなかった。
3. レオン「勝ったんじゃない。終わらせただけだ」
4. 帝国との戦がこれで終わるのかは、まだ誰にも分からない。それでも、工房の煙突から、久しぶりに煙が上がった。

- どの段落も 200 文字以下（最も長い段落は 70 文字ほど）
- 物語の文は、誰が生きているかで変えない（DES-STY-01）。ロストで戦死しうる仲間（ガルド・ブラン・ミラ・リナ、加わったあとのエルザ）は、全員が生きている ch01_before と、加入の場面の ch02_before のエルザのほかでは、話さず動かない。ほかの台詞は、主人公（倒れると負けなので、必ず生きている）と名のない兵にした
- （v0.3）リナは ch01_before の地の文（4 段落目）にだけ出し、台詞は足さない（REV-020 F-07 の方針。戦死しうる仲間の台詞を増やさない）。ch02_after の「錬金術士」の段落はそのまま。段落を写すときは「（v0.3 で足した段落）」の注は写さない

### 10.9 クラスの役割の検算（v0.3）

10.3 節の値が、E-30 の役割どおりの数字になることを、スクリプト（反復 1 の 3.5 節の式。`balance.py`。検算のスクリプトは管理役の作業場所にあり、リポジトリには入れない）で確かめた。相手は 1 章の敵（ソルジャー・ファイター・剣士・アーチャー。ボーナスなし。HP の平均 19）で、両方とも平地にいるものとする。命中率は予測に出す値（実際に当たる確率は、乱数 2 つの平均なので、60 なら 68%、70 なら 82%、90 なら 98%）。

**4 種類の敵の平均**

| キャラ（クラス） | 役割（E-30） | 1 撃 | 1 戦闘の合計（追撃を含む） | 命中率 | 追撃できる相手 | 受けるダメージ（1 撃） | HP / 守備 |
|---|---|---|---|---|---|---|---|
| レオン（ロード） | 高命中・高速・紙耐久 | 9.0（v0.2 8.0） | **15.5**（11.8） | **96**（91） | **3/4**（2/4） | 7.8（7.8） | 19 / 4（20 / 4） |
| ブラン（ファイター） | 高火力・低耐久 | **16.0**（13.0） | **16.0**（13.0） | 68（66） | 0/4（0/4） | **9.8**（8.8） | **22 / 2**（26 / 3） |
| ガルド（アーマー） | 守備が最強・火力は中・命中が不安 | 10.0（10.0） | 10.0（10.0） | **70**（74） | 0/4（0/4） | **1.5**（2.2） | 24 / **11**（24 / 10） |
| ミラ（アーチャー） | 変えない | 8.0 | 8.0 | 92 | 0/4 | 8.8 | 17 / 3 |

- 1 戦闘の合計はブラン（16.0）とレオン（15.5）が並んで最も高い。ブランは 1 撃で敵の HP の 8 割ほど（2 撃でどの敵も倒す）、レオンは追撃で稼ぐ。ガルドとミラは 10 以下
- 受けるダメージはブランが最も大きく、HP と守備はブランとレオンが前衛で最も低い。ガルドは 1 撃 1.5 で、ほぼ傷つかない
- ガルドの命中率は 59〜79（ファイターに 59、ソルジャーに 74）で、外すことがある

**相手ごとの数字（1 章）**: 「与」は自分から攻撃したとき（ダメージ × 回数、命中率）、「受」はその敵から 1 撃を受けたとき（ダメージ、命中率）

| キャラ | ソルジャー（HP 18） | ファイター（HP 24） | 剣士（HP 17） | アーチャー（HP 17） |
|---|---|---|---|---|
| レオン | 与 7 × 2、89 / 受 9、79 | 与 10 × 2、100 / 受 10、39 | 与 10 × 1、94 / 受 5、82 | 与 9 × 2、100（撃破）/ 受 7、75 |
| ブラン | 与 16、87 / 受 9、63 | 与 16、72 / 受 13、68 | 与 16、47 / 受 8 × 2、100 | 与 16、68 / 受 9、89 |
| ガルド | 与 9、74 / 受 1、84 | 与 9、59 / 受 5、89 | 与 12、79 / 受 0 | 与 10、70 / 受 0 |
| リナ | 受 10、76 | 受 13、66 | 受 7、94 | 受 9、87 |

- **ロード・ファイターが「使える」場面**（E-32）: レオンは、ファイター（20 ダメージ、受ける命中率 39）とアーチャー（18 ダメージで撃破）を相手に、ほぼ当て、ほぼ避ける。ブランは、ソルジャー（相性有利。16 ダメージ・命中率 87）を 1 戦闘でほぼ倒す。どちらも、剣士（ブランは命中 47 で、2 撃 16 を受ける）や、ソルジャーの群れ（レオンは 2 撃で 18）には出さない、と使い分ける
- **主人公がすぐ倒れないこと（1 章）**: レオンはソルジャーの 2 撃（9 × 2 = 18）に耐え（HP 19）、3 撃目で倒れる。v0.2（HP 20、同じく 2 撃に耐える）と同じで、回避が 18 から 24 に上がったので、敵の命中率は 6 ずつ下がる。紙耐久が効いてくるのは章が進んでから（3 章の Lv 3 では、ソルジャーの 1 撃が 11 で、2 撃で倒れる。v0.2 は 2 撃に耐えた）
- **1 章のボス**（ソルジャー、砦の上、能力 26/8/7/6/7）: レオン 2 × 2 回・命中率 65、ガルド 4・50、ブラン 11・63、ミラ 2・71（v0.2 はレオン 1、ガルド 4、ブラン 8、ミラ 2）。砦の回復（5）を上回る
- **僧侶**: リナは、剣士の 1 撃（7）のほかは 2 撃で倒れる（HP 16）。敵の攻撃の候補として選ばれやすいので、前衛の後ろに置く
- 2 章（Lv 2）・3 章（Lv 3）でも、役割の順（1 戦闘の合計はブランとレオン、受けるダメージはガルドが最も小さい）は変わらないことを確かめた（スクリプトの出力。2 章・3 章のボスへのダメージは 10.5・10.6 節）

## 11. 変更履歴

| 版 | 日付 | 内容 |
|---|---|---|
| 0.1 | 2026-10-21 | 初版（案）。E-25・E-26 をもとに、Claude Code が推奨の案で書いた |
| 0.2 | 2026-10-21 | REV-020 の指摘を反映した: 9 章（Issue の中身と、書き換える反復 1 のテスト）、`Exp`・`LevelUp` の写し、セーブを読む順、やり直しをメモリの中身から行うこと、セーブの置き場所（IF-04）、DES-STY-01 と DES-UI-17、物語の下書き（戦死しうる仲間の台詞）、規則 9・16 の明記ほか。AGENTS.md の表に `internal/savefile` を足し、テスト仕様書（反復 2）の骨組みを作った。開発者が承認した（E-28。C-01〜C-10 は推奨どおり、物語の下書きは遊ぶときに確かめる） |
| 0.3 | 2026-10-21 | 開発者が反復 1 の版を遊んだ結果（E-29〜E-32）から、次を足した（案）。クラスの役割を尖らせる（10.3・10.9 節。レオン・ガルド・ブランのボーナスと成長率。敵の能力は変えない）。回復役の僧侶リナと杖、行動「回復」（DES-HEL-01・02、4.1・4.4・4.5 節、規則 4・6・21・22、`Setup` の規則 8、DES-UI-18）。予測の撃破・危険の強調（DES-UI-07 の変更、7.11 節）。出撃の枠を 1 章 5・2 章 6・3 章 5 にした。4.2 節の計算例を v0.3 の値で計算し直し、例 7 を足した。9 章に Issue 9〜12 と、書き換える反復 1 のテストを足した。方針は E-32 で開発者が承認した。数値などの細部は Claude Code の案で、レビュー（REV-021）を通したものをマージする |

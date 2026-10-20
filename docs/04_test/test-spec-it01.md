# テスト仕様書（反復 1）

- 文書 ID: TST-IT01
- 版: 0.1（実装とともに書き足す）
- 対象: [反復 1 計画書](../iterations/it01/plan.md) の範囲、[設計書 DES-IT01](../03_design/design-it01.md)

## 1. 方針

| 項目 | 内容 |
|---|---|
| テストケース ID | `TC-<要求ID>-<連番2桁>` |
| コードとの対応 | 表駆動テストのサブテスト名にテストケース ID を入れる |
| 書き足し方 | 各 Issue の実装の担当が、自分の担当する要求の節だけに行を足す（AGENTS.md「並列作業の注意」）。同じ要求を複数の Issue が担当するときは、その要求の節に書き足す。節がなければ、2 章の終わりに足す |
| 種別 | 正常・境界・異常 |
| 乱数 | 当たり・外れは、乱数を差し替えて決まった値で確かめる（DES-RNG-01） |

## 2. テストケース

列は「ID、対象（要求・設計）、前提、入力・操作、期待結果、種別」とする。

### FR-SYS-03・FR-SYS-04・IF-03（定義データ。#5。FR-SYS-03 は #11、FR-SYS-04 は #12 も）

テストコードは `internal/defload` にある。FR-SYS-04 のケースは、本体のデータ（`data/base/`）を写して一部を書き換えたものを `defload.Load` に渡す。期待結果の「誤り」は、`defload.Errors` の 1 つの要素の文面（`ファイル名: 位置: 理由`）に含まれることを確かめる。キーの道筋の添字は 0 から数える。

| ID | 対象 | 前提 | 入力・操作 | 期待結果 | 種別 |
|---|---|---|---|---|---|
| TC-FR-SYS-03-01 | FR-SYS-03 / DES-DAT-01、DES-DAT-02、設計書 10 章・3.7 節 | 本体のデータ | `defload.Load(data.Base())` | 誤りがない。始めるマップ、地形・クラス・武器・アイテム・スキル・キャラ・マップ（配置・増援）の値が 10 章と 3.7 節のとおり。キャラとボスの能力（クラス＋ボーナス）が 10.3・10.4 節の「能力」と一致する | 正常 |
| TC-FR-SYS-03-02 | FR-SYS-03 / DES-DAT-01 | 本体のデータを読み込んだ目録 | 目録にない ID（空の ID を含む）で地形・クラス・武器・アイテム・スキル・キャラ・マップを引く | どれも「ない」（`false`）が返る。ある ID なら定義が返る | 異常 |
| TC-IF-03-01 | IF-03 / DES-DAT-02、ADR-0003 | 本体のデータ | `data.Base()` のファイルの一覧と中身を見る | 2.2 節の 8 ファイル（`*.toml`）があり、どれも UTF-8 のテキスト | 正常 |
| TC-IF-03-02 | IF-03 / DES-DAT-02、ADR-0003 | 本体のデータを読み込んだ目録 | マップ `ch01` の行を見る | 1 行が 1 つの文字列で、10.4 節の図と同じ。(13, 1)・(13, 8) の文字が砦 | 正常 |
| TC-FR-SYS-04-01 | FR-SYS-04 / DES-DAT-03 規則 1 | 本体のデータ | `items.toml` の 6 行目を `heal = = 10`（文法の誤り）、または `heal = "ten"`（値の型の誤り）にする | 誤り `items.toml: 6 行目: …` | 異常 |
| TC-FR-SYS-04-02 | FR-SYS-04 / DES-DAT-03 規則 2 | 本体のデータ | 武器の ID を重ねる。スキルの ID を空にする。ID のないアイテムを足す | 誤り `weapons.toml: weapons[1].id: ID "iron_sword" が重なっています（weapons[0] と同じ）`、`skills[3].id: ID が空です`、`items[1].id: ID が空です` | 異常 |
| TC-FR-SYS-04-03 | FR-SYS-04 / DES-DAT-03 規則 3 | 本体のデータ | キャラのクラス、敵の武器、手持ちのアイテム、自軍の配置のキャラ、キャラのスキル、マップの地形の文字、始めるマップを、ない ID・文字にする。キャラのスキルを空にする | それぞれ誤り（例 `characters.toml: characters[2].class: クラス "knight" がありません`、`maps.toml: maps[0].rows[15]: 15 文字目の地形の文字 "X" がありません`、`game.toml: start_map: マップ "ch99" がありません`）。スキルが空なら誤りがない | 異常 |
| TC-FR-SYS-04-04 | FR-SYS-04 / DES-DAT-03 規則 4 | 本体のデータ | 武器の種類を `spear`、クラスの武器の種類を `gun`、スキルの効果を `shield`、敵の動き方を `flee` にする | それぞれ誤り（例 `weapons.toml: weapons[2].type: 武器の種類 "spear" は使えません（sword・lance・axe・bow のどれか）`） | 異常 |
| TC-FR-SYS-04-05 | FR-SYS-04 / DES-DAT-03 規則 5 | 本体のデータ | レオン（ロード、剣）に鉄の斧を持たせる。増援の剣士に鉄の弓を持たせる | 誤り `characters.toml: characters[0].weapon: 武器 "iron_axe" の種類 axe が、クラス "lord" の武器の種類 sword と違います`、`maps[0].reinforcements[2].weapon: …` | 異常 |
| TC-FR-SYS-04-06 | FR-SYS-04 / DES-DAT-03 規則 6 | 本体のデータ | ボーナスを足した HP を 0、移動を -1 にする。クラスの HP を 0 にする（ボーナスのない敵の、足したあとの HP が 0 になる）。武器の威力・命中を -1、最小の射程を 0、最小 3・最大 2 にする。アイテムの回復・数を 0 にする。スキルの範囲・命中・回避を -1、確率を 101 と -1、怒りの倍率を 0、怒りでない倍率を -1 にする。地形の費用・回避・守備を -1、回復の割合を 101 と -1 にする | それぞれのキーの道筋で誤り（例 `characters.toml: characters[3].bonus.hp: 能力（クラス 17 ＋ ボーナス -17 ＝ 0）は 1 以上にしてください`、`skills.toml: skills[1].chance: 0 以上 100 以下にしてください（101）`、`maps.toml: maps[0].enemies[2].bonus.hp: 能力（クラス 0 ＋ ボーナス 0 ＝ 0）は 1 以上にしてください`）。クラスの `base` だけでは確かめない | 異常 |
| TC-FR-SYS-04-07 | FR-SYS-04 / DES-DAT-03 規則 6 | 本体のデータ | 範囲の端の値にする: ボーナスを足した能力を HP 1・ほか 0、武器の威力・命中 0、回復 1、確率 100、スキルの範囲・命中・回避 0、怒りの倍率 1、回復の割合 100 | 誤りがない | 境界 |
| TC-FR-SYS-04-08 | FR-SYS-04 / DES-DAT-03 規則 7 | 本体のデータ | 地形の文字を `FF`（2 文字）、空、ほかの地形と同じ文字にする | 誤り `terrains.toml: terrains[1].symbol: 地形の文字 "FF" は 1 文字にしてください（2 文字）`、`(0 文字)`、`地形の文字 "F" が重なっています（terrains[1] と同じ）` | 異常 |
| TC-FR-SYS-04-09 | FR-SYS-04 / DES-DAT-03 規則 7 | 本体のデータ | 文字が `森`（複数バイトで 1 rune）の地形を足し、マップの行に使う | 誤りがない | 境界 |
| TC-FR-SYS-04-10 | FR-SYS-04 / DES-DAT-03 規則 8 | 本体のデータ | `ch01` の最後の行を 15 文字にする | 誤り `maps.toml: maps[0].rows[15]: 行の長さ 15 が、rows[0] の長さ 16 と違います` | 異常 |
| TC-FR-SYS-04-11 | FR-SYS-04 / DES-DAT-03 規則 8 | 本体のデータ | `ch01` の行を 17 行にする。行が 0 行、長さ 17 の行、長さ 0 の行のマップを足す | 誤り `maps[0].rows: 行の数 17 は 1 以上 16 以下にしてください`、`行の数 0 …`、`maps[1].rows[0]: 行の長さ 17 …`、`行の長さ 0 …` | 境界 |
| TC-FR-SYS-04-12 | FR-SYS-04 / DES-DAT-03 規則 8 | 本体のデータ（`ch01` は 16 行 × 16 文字） | 1 行 2 マスのマップ、rune で 16 文字（複数バイトの文字を含む）の行のマップを足す | 誤りがない | 境界 |
| TC-FR-SYS-04-13 | FR-SYS-04 / DES-DAT-03 規則 9 | 本体のデータ | 増援を (16, 15)、自軍を (1, -1) に置く。敵と増援を壁 (12, 3) に置く。敵を自軍と同じ (1, 14) に置く | 誤り `maps[0].reinforcements[2]: 座標 (16, 15) がマップの外です`、`maps[0].enemies[3]: 座標 (12, 3) の地形 "wall" は通れません`、`maps[0].reinforcements[2]: 座標 (12, 3) の地形 "wall" は通れません`、`maps[0].enemies[6]: 座標 (1, 14) が maps[0].allies[0] と重なっています` など | 異常 |
| TC-FR-SYS-04-14 | FR-SYS-04 / DES-DAT-03 規則 9 | 本体のデータ | 増援を自軍の初めの配置と同じ (1, 15) に置く | 誤りがない（重なりを調べるのは初めの配置どうしだけ） | 境界 |
| TC-FR-SYS-04-15 | FR-SYS-04 / DES-DAT-03 規則 10 | 本体のデータ | 自軍の配置からレオンを除く。ガルドも `lord = true` にする。ミラの代わりにガルドを 2 回置く | 誤り `maps[0].allies: lord = true のキャラを、ちょうど 1 人置いてください（0 人）`、`（2 人）`、`maps[0].allies[3].character: キャラ "gald" を 2 回置いています（maps[0].allies[1] と同じ）` | 異常 |
| TC-FR-SYS-04-16 | FR-SYS-04 / DES-DAT-03 規則 11 | 本体のデータ | ボスをなくす。アーチャーもボスにする。増援をボスにする | 誤り `maps[0].enemies: boss = true の敵を、ちょうど 1 体置いてください（0 体）`、`（2 体）`、`maps[0].reinforcements[2].boss: 増援はボスになれません` | 異常 |
| TC-FR-SYS-04-17 | FR-SYS-04 / DES-DAT-03 規則 12 | 本体のデータ | 増援の `turn` を 1、2 にする | 1 なら誤り `maps[0].reinforcements[2].turn: 2 以上にしてください（1）`。2 なら誤りがない | 境界 |
| TC-FR-SYS-04-18 | FR-SYS-04 / DES-DAT-03 | 本体のデータ | 2 つのファイルに文法の誤りを入れる。3 つのファイルに意味の誤りを入れる | 誤りは 1 つで止まらず、すべてが `defload.Errors` で返る。目録は返らない | 異常 |
| TC-FR-SYS-04-19 | FR-SYS-04 / DES-DAT-02、DES-DAT-03 | 本体のデータ | `skills.toml` を消す | 誤り `skills.toml: ファイルを読めません（…）` | 異常 |
| TC-FR-SYS-04-20 | FR-SYS-04 / DES-DAT-03 規則 13 | 本体のデータ | 地形に `heal_precent`、キャラのボーナス（インラインの表）に `hpp`、自軍の配置に `z`、敵（入れ子の配列の表）に `aii`、`game.toml` のいちばん上に `start_mapp` を書く | 誤り `terrains.toml: terrains[1].heal_precent: 知らないキーです`、`characters.toml: characters[0].bonus.hpp: …`、`maps.toml: maps[0].allies[3].z: …`、`maps.toml: maps[0].enemies[0].aii: …`、`game.toml: start_mapp: …` | 異常 |

### FR-BTL-01・FR-BTL-02・FR-BTL-03・FR-MAP-01・NFR-06・NFR-07（戦闘の土台。#6）

テストは `internal/core/tactics` の `*_test.go` にあり、小さな `Setup` を Go のコードで組み立てる（定義データに依存しない）。盤面の図の `.` は平地（費用 1）、`F` は森（費用 2）、`#` は壁（費用 0）。

| ID | 対象 | 前提 | 入力・操作 | 期待結果 | 種別 |
|---|---|---|---|---|---|
| TC-FR-BTL-01-01 | FR-BTL-01 / DES-BTL-03 | 7 × 7 の平地の中央に移動 2 の主人公 | `Reachable` | 距離 2 以内の 13 マス（今いるマスを含む）で、費用は距離と同じ | 正常 |
| TC-FR-BTL-01-02 | FR-BTL-01 / DES-BTL-04 | 1 ターン目の自軍の番 | 移動できるマスを `To` にして `Wait` で `Act` | ユニットの `Pos` が `To` になり、`Acted` が真になる | 正常 |
| TC-FR-BTL-01-03 | FR-BTL-01 / DES-BTL-03 | 平地の 1 本道に移動 3 の主人公。別に移動 0 の主人公 | `Reachable` | 費用の合計がちょうど 3 のマスまで含み、4 のマスは含まない。移動 0 なら今いるマスだけ | 境界 |
| TC-FR-BTL-01-04 | FR-BTL-01 / DES-BTL-03 | 1 本道で、主人公（移動 3）の隣に味方 | `Reachable` | 味方のいるマスは含まない（止まれない）。その先のマスは費用 2 で含む（通り抜けられる） | 正常 |
| TC-FR-BTL-01-05 | FR-BTL-01 / DES-BTL-03 | 1 本道や 3 × 3 の盤面で、主人公の隣に敵。別に、敵から見た盤面 | `Reachable` | 相手の陣営のいるマスとその先には入れず、回り道の費用になる。敵から見ると、敵どうしは通り抜けられ、自軍のマスには入れない | 正常 |
| TC-FR-BTL-01-06 | FR-BTL-01 / DES-BTL-03 | 1 本道の敵と味方が倒れている | `Reachable` | 倒れたユニットのいたマスは通れて、止まれる | 境界 |
| TC-FR-BTL-01-07 | FR-BTL-01 / DES-BTL-01 | 倒れたユニットがいる | 倒れたユニット、添字 -1、添字が数を超えるユニットで `Reachable`・`AttackRange` | nil を返す | 異常 |
| TC-FR-BTL-01-08 | FR-BTL-01 / DES-BTL-01 | 5 × 5 の平地 | 剣（射程 1）で (2, 2) から、弓（射程 2）で (0, 0) と (4, 4) から `AttackRange` | 剣は隣の 4 マス。弓は隣を含まず、マップの端で切れる。y、x の小さい順 | 正常 |
| TC-FR-BTL-01-09 | FR-BTL-01 / DES-BTL-01 | 正しい `Setup`（自軍 2 体、敵 2 体） | `New` | 誤りがなく、盤面の大きさが正しく、ユニットは `Setup` の順で、HP は最大、生きていて、行動していない | 正常 |
| TC-FR-BTL-01-10 | FR-BTL-01 / DES-BTL-01 | 正しい `Setup` を 1 か所だけ壊す | `Setup` の規則 1〜6 のそれぞれに反する `Setup`（地形の行がない・空・長さ違い、座標がマップの外・通れない地形、重なり、能力・射程・アイテムの範囲、主人公とボスの数と陣営、増援の主人公・ボス・陣営・ターン、定数の値、スキルの数値）で `New` | `ErrInvalidSetup` を包んだ誤りを返し、誤りの文に理由（位置と規則）を含む。戦闘と出来事は nil | 異常 |
| TC-FR-BTL-01-11 | FR-BTL-01 / DES-BTL-01 | — | 最大 HP 1、ほかの能力 0、射程 1〜1、アイテムの数と回復量 1、大盾の確率 0 と 100、怒りの倍率 1、増援のターン 2（初めのユニットと同じマス）、1 行のマップで `New` | 誤りにならない | 境界 |
| TC-FR-BTL-01-12 | FR-BTL-01 / DES-BTL-01 | 戦闘を作った | 渡した `Setup` の地形と手持ちを変え、`Units()` の写しの HP と手持ちを変える | 戦闘の状態と地形は変わらない | 正常 |
| TC-FR-BTL-02-01 | FR-BTL-02 / DES-BTL-02 | — | `New` | 1 ターン目の自軍の番で、戦闘は続いている。出来事は `PhaseStart`（自軍、1 ターン目）だけ | 正常 |
| TC-FR-BTL-02-02 | FR-BTL-02 / DES-BTL-02 | 自軍 2 体 | 添字の大きいほうの自軍を先に `Wait` | 自軍の番のまま。動かした自軍だけが行動済み | 正常 |
| TC-FR-BTL-02-03 | FR-BTL-02 / DES-BTL-02 | 自軍 2 体 | 2 体とも `Wait` | 2 体目の `Act` が `Wait`、`PhaseStart`（敵軍、1 ターン目）、`PhaseStart`（自軍、2 ターン目）を返す。2 ターン目の自軍の番になり、自軍の `Acted` は偽に戻る。次のターンも動かせる | 正常 |
| TC-FR-BTL-02-04 | FR-BTL-02 / DES-BTL-02 / UC-05 | 自軍 2 体のうち 1 体だけ行動した | `EndPhase`。続けて、誰も動かさずに `EndPhase` | `PhaseStart`（敵軍、1 ターン目）、`PhaseStart`（自軍、2 ターン目）を返し、2 ターン目の自軍の番になる。行動した自軍の `Acted` は偽に戻る。2 回目で 3 ターン目になる | 正常 |
| TC-FR-BTL-02-05 | FR-BTL-02 / DES-BTL-02 | 自軍 2 体のうち 1 体が倒れている | 生きている 1 体を `Wait` | 敵軍の番を経て 2 ターン目の自軍の番になる（倒れた自軍は数えない） | 境界 |
| TC-FR-BTL-02-06 | FR-BTL-02 / DES-BTL-02 | 敵 2 体 | `EndPhase` を 3 回 | 毎回の出来事は `PhaseStart` 2 つだけで、敵の位置・HP などは変わらない（敵の動き方は #10）。4 ターン目になる | 正常 |
| TC-FR-BTL-02-07 | FR-BTL-02 / DES-BTL-02 / DES-BTL-08 | 戦闘が終わっている（勝ち・負け）、または敵軍の番 | `EndPhase` | 戦闘が終わっていれば `ErrBattleOver`（敵軍の番でも）、敵軍の番なら `ErrNotPlayerPhase`。出来事は nil で、状態は変わらない | 異常 |
| TC-FR-BTL-03-01 | FR-BTL-03 / DES-BTL-04 / DES-EVT-01 / UC-02 | 1 ターン目の自軍の番 | (0, 0) の主人公を (1, 1) に `Wait` | 出来事は `Move`（`Unit`、`From` (0, 0)、`To` (1, 1)）、`Wait`（`Unit`）の順 | 正常 |
| TC-FR-BTL-03-02 | FR-BTL-03 / DES-BTL-04 / DES-EVT-01 | 1 ターン目の自軍の番 | 今いるマスを `To` にして `Wait` | 出来事は `Wait` だけ（`Move` を出さない）。行動済みになる | 境界 |
| TC-FR-BTL-03-03 | FR-BTL-03 / DES-EVT-01 | 1 ターン目の自軍の番 | `New`、続けて 2 体を `Wait` で動かす | 出来事の `Units` は、その出来事の直後の写し（`Move` は移動したあとで行動済みでない、`Wait` は行動済み、次の番の始めは `Acted` が偽）。後の行動で前の写しは変わらず、写しを変えても戦闘は変わらない | 正常 |
| TC-FR-MAP-01-01 | FR-MAP-01 / DES-BTL-03 / DES-MAP-01 | 森（費用 2）のある 1 本道。森の列と、その下の平地の列 | `Reachable` | 森に入る費用は 2 で数える。森を通るより回り道が安いマスは、回り道の費用になる | 正常 |
| TC-FR-MAP-01-02 | FR-MAP-01 / DES-BTL-03 / DES-MAP-01 | 壁（費用 0）の列で仕切られた盤面 | `Reachable` | 壁のマスには入れず、壁の向こうへは回り道の費用で行く | 異常 |
| TC-FR-MAP-01-03 | FR-MAP-01 / DES-BTL-03 | 主人公の隣が森（費用 2） | 移動 1 と移動 2 で `Reachable` | 移動 1 では森に入れず、移動 2 なら入れる | 境界 |
| TC-FR-MAP-01-04 | FR-MAP-01 / DES-MAP-01 | 3 × 2 の盤面（平地、森、壁、砦） | `Width`、`Height`、`TerrainAt` | 3、2 と、`Setup` のとおりの地形。マップの外はゼロ値 | 正常 |
| TC-FR-MAP-01-05 | FR-MAP-01 / DES-BTL-03 | 3 × 3 の平地の角に移動 2 の主人公 | `Reachable` | マップの中のマスだけを返す | 境界 |
| TC-NFR-06-01 | NFR-06 / DES-RNG-01 | 種 42 の戦闘を 2 つ | それぞれで乱数を 1000 回引く | 2 つの列が同じで、値はすべて 0〜99 | 正常 |
| TC-NFR-06-02 | NFR-06 / DES-RNG-01 | 種 7 の戦闘を 2 つ | 同じコマンドの列（`Wait` 3 回、`EndPhase`） | `New` からの出来事の列と、最後の状態が同じ | 正常 |
| TC-NFR-06-03 | NFR-06 / DES-RNG-01 | — | `export_test.go` の口で、乱数を引く関数を (40, 90, 70, 50) を返すものに差し替えて 4 回引く | 40、90、70、50 が順に出る | 正常 |
| TC-NFR-07-01 | NFR-07 / DES-BTL-04 | 戦闘の終わり・敵軍の番・行動済みで倒れた味方・行動済みの味方などを組み合わせる | 検証 0〜5 の 2 つ以上に当てはまるコマンド（`Wait`、`Attack`、`UseItem`）で `Act` | 番号の小さい検証の誤りを返す（0 `ErrInvalidAction` → 1 `ErrBattleOver` → 2 `ErrNotPlayerPhase` → 3 `ErrInvalidUnit` → 4 `ErrAlreadyActed` → 5 `ErrUnreachable`）。`UseItem` でも 5 を 6 より先に確かめる | 異常 |
| TC-NFR-07-02 | NFR-07 / DES-BTL-04 / DES-BTL-08 | 1 ターン目の自軍の番 | `Action` が範囲外（3、-1）、勝ち・負けのあと、敵軍の番、`Unit` が -1・数を超える・敵・倒れた味方、行動済み、`To` が壁・マップの外・移動を超える・味方のいるマス・敵のいるマス | それぞれ `ErrInvalidAction`、`ErrBattleOver`、`ErrNotPlayerPhase`、`ErrInvalidUnit`、`ErrAlreadyActed`、`ErrUnreachable` を返す | 異常 |
| TC-NFR-07-03 | NFR-07 / DES-ERR-01 | TC-NFR-07-01・02 のそれぞれ | 失敗する `Act` | 出来事は nil で、ターン・番・結果・ユニット（位置、HP、行動済み、手持ちなど）が変わらない | 異常 |
| TC-NFR-07-04 | NFR-07 / DES-ERR-01 | 1 ターン目の自軍の番 | 検証 0〜5 を通る `Attack`・`UseItem` で `Act`（攻撃は #8、アイテムは #9 で作るまでの扱い） | まだ作っていないことを示す誤りを返し、出来事は nil で、状態は変わらない | 異常 |
| TC-NFR-07-05 | NFR-07 / DES-BTL-08 | — | 公開の誤り 11 個どうしを `errors.Is` で比べる | 自分とだけ一致し、ほかとは一致しない | 正常 |

### NFR-03・NFR-04・NFR-05・NFR-09（非機能の検査。#7）

NFR-04・NFR-05 のテストコードは `internal/core/deps_test.go` にある。実際のパッケージは `go list -json ./...`（`internal/core` 以下）で直接の import を、`go list std` で標準ライブラリの一覧を得て調べる。ほかのケースは、import の一覧を Go のコードで組み立てて、判定の関数に渡す。Ebitengine は `github.com/hajimehoshi/ebiten` で始まる import とする。`go test` のキャッシュは、テストの中から起動した `go list` が読んだソースを追わない（`internal/core` の下にパッケージを足しても、`(cached)` で前の結果が出る）ので、手もとでは `go test -count=1` で実行する（CI は毎回まっさらな環境なので影響しない）。

NFR-03・NFR-09 は、CI（`.github/workflows/ci.yml`）の手順で確かめる（対象の列に「CI」と書いた）。ただし、網羅率が 80% 未満で失敗する側（TC-NFR-09-02）は、ふだんの CI では起きないので、CI の手順の判定式を手で動かして確かめる（対象の列に「手動。CI の判定式」と書いた）。どれもテストコードはなく、tccheck では「テストコードにありません」の警告になる。CI のケースの結果は、プルリクエストと `main` への push ごとの CI の記録に残る。手動のケースの結果は、その Issue のレビュー記録（`docs/reviews/`）に残す。Windows 11 と Linux で起動して遊べること（NFR-03 の D）は、画面ができてから手で確かめる（#7 の範囲外）。

| ID | 対象 | 前提 | 入力・操作 | 期待結果 | 種別 |
|---|---|---|---|---|---|
| TC-NFR-03-01 | NFR-03 / DES-NFR-02（CI） | Go のコードを変えたプルリクエスト、または `main` への push | CI の手順「クロスビルド（Windows）」で `GOOS=windows go build ./...` | ビルドが成功し、手順が成功になる | 正常 |
| TC-NFR-03-02 | NFR-03 / DES-NFR-02（CI） | TC-NFR-03-01 と同じ | CI の手順「クロスビルド（Linux）」で `GOOS=linux go build ./...` | ビルドが成功し、手順が成功になる | 正常 |
| TC-NFR-04-01 | NFR-04、NFR-05 / DES-NFR-01、ADR-0002 | このリポジトリの `internal/core` 以下 | `go list` で各パッケージの直接の import を得て調べる | 結果に `internal/core/tactics` があり、どのパッケージも Ebitengine と `time` を import せず、`tactics` は標準ライブラリだけを import している | 正常 |
| TC-NFR-04-02 | NFR-04 / DES-NFR-01、ADR-0002 | — | core が `github.com/hajimehoshi/bitmapfont/v3`、`example.com/hajimehoshi/ebiten/v2`、`internal/defs` を、`tactics` が `errors`、`fmt`、`math/rand/v2` を import する | 違反はない（似た名前は Ebitengine とみなさない。`tactics` 以外は標準ライブラリでなくてもよい） | 境界 |
| TC-NFR-04-03 | NFR-04 / DES-NFR-01 | — | core が `github.com/hajimehoshi/ebiten/v2` を、`tactics` が `github.com/hajimehoshi/ebiten/v2/inpututil` を import する | 2 つとも Ebitengine の違反になる（`tactics` は標準ライブラリでない違反にもなる） | 異常 |
| TC-NFR-04-04 | NFR-04 / DES-NFR-01、ADR-0002 | — | core が `github.com/BurntSushi/toml` を、`tactics` が `fmt`、`internal/defs`、`github.com/BurntSushi/toml` を import する | `tactics` の `internal/defs` と `github.com/BurntSushi/toml` だけが、標準ライブラリでない違反になる | 異常 |
| TC-NFR-05-01 | NFR-05 / DES-NFR-01 | このリポジトリの `internal/core` 以下 | TC-NFR-04-01 と同じ | どのパッケージも `time` を import していない | 正常 |
| TC-NFR-05-02 | NFR-05 / DES-NFR-01 | — | core が `example.com/time`、`runtime` を、`tactics` が `runtime` を import する | 違反はない（`time` と同じ名前のときだけ違反にする） | 境界 |
| TC-NFR-05-03 | NFR-05 / DES-NFR-01 | — | core と `tactics` が `time` を import する | 2 つとも `time` の違反になる | 異常 |
| TC-NFR-09-01 | NFR-09 / DES-NFR-03（CI） | TC-NFR-03-01 と同じ | CI の手順「core の命令網羅率」で `go test -coverprofile` を `./internal/core/...` に行い、`go tool cover -func` の `total` を読む | 合計が 80% 以上なら手順が成功になる | 正常 |
| TC-NFR-09-02 | NFR-09 / DES-NFR-03（手動。CI の判定式） | — | CI の手順「core の命令網羅率」の判定式（`awk -v t="$total" 'BEGIN { exit !(t != "" && t + 0 >= 80) }'`）に、`total` として 79.9・80.0・空を渡して終了コードを見る | 79.9 と空は失敗（終了コード 1）、80.0 は成功（終了コード 0）。失敗のとき、手順と CI が失敗になる | 境界 |

### FR-BTL-04・FR-BTL-05・FR-BTL-06・FR-BTL-08・FR-UNT-07・FR-ITM-01（攻撃。#8。FR-BTL-08 は #13 も）

### FR-ITM-02・FR-ITM-03・FR-MAP-03・FR-MAP-05・FR-CMP-09（アイテム・砦・増援・勝ち負け。#9。FR-ITM-03 は #11 も）

### FR-BTL-07（敵の動き方。#10）

### FR-UNT-01・FR-UNT-02（定義データから戦闘を作る。#11）

### IF-01・IF-02・FR-UNT-09（画面。#12、#13。手で確かめる）

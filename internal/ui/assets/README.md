# 画像素材の出典

ふしぎ錬金（`sakana396/fushigi_renkin` の `internal/ui/assets/`、ふしぎ錬金の ADR-0003）から、配布物の画像と同梱の利用許諾をそのまま持ち込んだ（2026-10-20、E-02）。利用許諾のファイルは原本から変更していない。読み込むコードは `assets.go`（DES-UI-05、#12）。

| 素材名と版 | 作者 | 配布元 | 利用許諾 | 確かめた日 | 取り込んだファイル（元のファイル名） |
|---|---|---|---|---|---|
| Tiny Town 1.1 | Kenney | <https://kenney.nl/assets/tiny-town> | CC0 1.0（`kenney_tiny_town_License.txt`） | 2026-10-15 | `tiny_town.png`（`Tilemap/tilemap_packed.png`）、`kenney_tiny_town_License.txt`（`License.txt`） |
| Tiny Dungeon 1.0 | Kenney | <https://kenney.nl/assets/tiny-dungeon> | CC0 1.0（`kenney_tiny_dungeon_License.txt`） | 2026-10-15 | `tiny_dungeon.png`（`Tilemap/tilemap_packed.png`）、`kenney_tiny_dungeon_License.txt`（`License.txt`） |

## 使ったタイル（DES-UI-05、#12）

タイルの番号は、画像の左上を 0 として右へ数える（1 行に 12 枚。番号 n は x = n % 12、y = n / 12 の位置）。16 × 16 px を 2 倍で描く。定数は `assets.go` にある。

| 用途 | 画像 | 番号 | 絵 |
|---|---|---|---|
| 平地（どのマスにも下地として描く） | Tiny Town | 0 | 草地 |
| 森 | Tiny Town | 16 | 緑の木 |
| 山 | Tiny Town | 43 | 石の多い草地 |
| 砦 | Tiny Town | 125 | 窓のある石壁 |
| 壁 | Tiny Dungeon | 40 | 石のれんが |
| 自軍の主人公（ロード） | Tiny Dungeon | 88 | 若い男 |
| 自軍の剣（主人公以外。反復 1 の本体のデータにはいない） | Tiny Dungeon | 85 | 男 |
| 自軍の槍（アーマー） | Tiny Dungeon | 96 | 鎧の騎士 |
| 自軍の斧（ファイター） | Tiny Dungeon | 87 | 角の兜の戦士 |
| 自軍の弓（アーチャー） | Tiny Dungeon | 112 | 緑の鉢巻きの狩人 |
| 敵のボス | Tiny Dungeon | 97 | 面頬の騎士 |
| 敵の剣（剣士） | Tiny Dungeon | 111 | 頭巾の男 |
| 敵の槍（ソルジャー） | Tiny Dungeon | 98 | 兜の兵士 |
| 敵の斧（ファイター） | Tiny Dungeon | 86 | 禿頭の荒くれ |
| 敵の弓（アーチャー） | Tiny Dungeon | 100 | 白髪の男 |

- 地形は `tactics.Terrain` の名前（平地・森・山・砦・壁）で選ぶ。名前の分からない地形は、通れなければ壁、回復があれば砦、それ以外は平地として描く
- ユニットは陣営と武器の種類で選ぶ（`tactics.Unit` はクラスの ID を持たないため）。自軍と敵は下地の色（青・赤）でも見分ける

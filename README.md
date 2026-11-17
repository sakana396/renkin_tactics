# レンキン・タクティクス（仮称）

ファイアーエムブレムのようなシミュレーション RPG。Go ＋ [Ebitengine](https://ebitengine.org/) で作る。

姉妹プロジェクト [ふしぎ錬金](https://github.com/sakana396/fushigi_renkin) の戦闘の中心の部分を作り込むためのリポジトリ。要求・設計・試験の文書は [docs/](docs/README.md) にある。

画像素材は Kenney の CC0 素材を使う（[internal/ui/assets/README.md](internal/ui/assets/README.md)）。

起動は `go run ./cmd/renkin`。既定は新作「古代都市の復旧」で、旧作は `-game base` を指定する。セーブ先は作品ごとに分かれる。

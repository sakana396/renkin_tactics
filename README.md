# レンキン・タクティクス（仮称）

ファイアーエムブレムのようなシミュレーション RPG。Go ＋ [Ebitengine](https://ebitengine.org/) で作る。

姉妹プロジェクト [ふしぎ錬金](https://github.com/sakana396/fushigi_renkin) の戦闘の中心の部分を作り込むためのリポジトリ。要求・設計・試験の文書は [docs/](docs/README.md) にある。

画像素材は Kenney の CC0 素材を使う（[internal/ui/assets/README.md](internal/ui/assets/README.md)）。

Linux でビルドするには、Ebitengine に必要な X11・OpenGL の開発用パッケージに加え、ALSA の開発用ヘッダ（Debian / Ubuntu では `libasound2-dev`）をインストールする。

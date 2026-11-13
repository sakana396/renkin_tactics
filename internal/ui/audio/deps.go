// Package audio は新作の効果音と BGM の再生を受け持つ（DES-NG-09）。
package audio

// Ebitengine の audio パッケージを依存に固定する（go.mod の整理で oto が消えないように。実際の再生の実装は後続の作業で足す）。
import _ "github.com/hajimehoshi/ebiten/v2/audio"

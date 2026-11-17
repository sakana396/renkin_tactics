// mkplaceholder は、新作の画像の仕組みを試すための仮の絵を書き出す道具（DES-NG-08）。
//
// 全キャラの units/・portraits/、敵のクラスの units/enemy_*.png、物語の背景 story/*.png、title.png に、
// 単色の人型のシルエットと名前の文字が入った PNG を書く。
// 出力先は既定で ./placeholder-out（media の外）。**ここで作った絵は data/alchemy/media にコミットしない**
// （出典の台帳 ASSETS.md に載せない前提の、試すためだけの画像）。使い方は docs/03_design/design-ng02-art.md の 6 章。
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

const (
	unitSize     = 128
	portraitSize = 512
	backdropW    = 832
	backdropH    = 576
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run は引数を読んで絵を書き出し、終了コードを返す。
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mkplaceholder", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "./placeholder-out", "出力先のディレクトリ（art/ の中身と同じ並びで書く）。data/alchemy/media の下は指定できない")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if inMedia(*out) {
		fmt.Fprintln(stderr, "出力先が data/alchemy/media の下です。仮の絵はコミットしないので、media の外を指定してください")
		return 2
	}
	files, err := Generate()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, f := range files {
		path := filepath.Join(*out, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		w, err := os.Create(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		err = png.Encode(w, f.Image)
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "%d 枚を %s に書き出しました。art/ の下へコピーして試してください（コミットしないこと）\n", len(files), *out)
	return 0
}

// inMedia は出力先が data/alchemy/media の下かどうかを、パスの文字だけで判定する。
func inMedia(out string) bool {
	abs, err := filepath.Abs(out)
	if err != nil {
		return false
	}
	media, err := filepath.Abs(filepath.Join("data", "alchemy", "media"))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(media, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && len(rel) > 2 && rel[:3] != ".."+string(filepath.Separator))
}

// File は書き出す 1 枚。Path は art/ の下の道筋（スラッシュ区切り）。
type File struct {
	Path  string
	Image image.Image
}

// Generate は新作の定義データから、書き出す絵の一覧を作る。
func Generate() ([]File, error) {
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		return nil, fmt.Errorf("定義データを読めません: %w", err)
	}
	var files []File
	for _, c := range cat.Characters() {
		files = append(files,
			File{"units/" + c.ID + ".png", silhouette(unitSize, unitSize, c.ID, c.Name)},
			File{"portraits/" + c.ID + ".png", silhouette(portraitSize, portraitSize, c.ID, c.Name)},
		)
	}
	for _, c := range cat.Classes() {
		files = append(files, File{"units/enemy_" + c.ID + ".png", silhouette(unitSize, unitSize, "enemy_"+c.ID, c.Name)})
	}
	for _, ch := range cat.Chapters() {
		for _, id := range []string{ch.Before, ch.After} {
			if id != "" {
				files = append(files, File{"story/" + id + ".png", backdrop(id, id)})
			}
		}
	}
	files = append(files, File{"title.png", backdrop("title", "タイトル")})
	return files, nil
}

// tint は名前から決まる、落ち着いた色。
func tint(key string) color.NRGBA {
	h := fnv.New32a()
	h.Write([]byte(key))
	v := h.Sum32()
	return color.NRGBA{R: 90 + uint8(v&0x3f), G: 90 + uint8(v>>6&0x3f), B: 90 + uint8(v>>12&0x3f), A: 255}
}

// silhouette は透過の背景に、単色の人型（頭と胴）と名前を描く。
func silhouette(w, h int, key, name string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	c := tint(key)
	// 頭: 中心 (w/2, h*0.32)、半径 h*0.14。胴: 台形（肩から足元へ広がる）。
	cx, cy, r := w/2, h*32/100, h*14/100
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x-cx, y-cy
			inHead := dx*dx+dy*dy <= r*r
			// 胴は y が h*0.50〜h*0.95。上端の幅 h*0.30、下端の幅 h*0.60。
			body := false
			if y >= h*50/100 && y <= h*95/100 {
				t := float64(y-h*50/100) / float64(h*45/100)
				half := (0.15 + 0.15*t) * float64(h)
				body = abs(x-cx) <= int(half)
			}
			if inHead || body {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	// 名前の文字を、下寄りに白で描く（等倍の bitmapfont。大きい絵では拡大して読みやすくする）。
	drawText(img, name, w/2, h*85/100, max(1, h/64))
	return img
}

// backdrop は暗めの単色に、名前の文字だけを入れた背景を作る。
func backdrop(key, label string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, backdropW, backdropH))
	c := tint(key)
	draw.Draw(img, img.Bounds(), &image.Uniform{color.NRGBA{R: c.R / 2, G: c.G / 2, B: c.B / 2, A: 255}}, image.Point{}, draw.Src)
	drawText(img, label, backdropW/2, backdropH/2, 2)
	return img
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawText は文字列を (cx, baseline) を中心に、scale 倍で白く描く。
// 等倍で小さな画像に描いたあと、最近傍で拡大して合成する。
func drawText(dst *image.NRGBA, s string, cx, baseline, scale int) {
	d := &font.Drawer{Face: bitmapfont.Face}
	width := d.MeasureString(s).Ceil()
	if width <= 0 {
		return
	}
	pad := 2
	tmp := image.NewNRGBA(image.Rect(0, 0, width+pad*2, 20))
	td := &font.Drawer{Dst: tmp, Src: image.White, Face: bitmapfont.Face, Dot: fixed.P(pad, 14)}
	td.DrawString(s)
	left := cx - tmp.Bounds().Dx()*scale/2
	top := baseline - 14*scale
	for y := 0; y < tmp.Bounds().Dy()*scale; y++ {
		for x := 0; x < tmp.Bounds().Dx()*scale; x++ {
			px := tmp.NRGBAAt(x/scale, y/scale)
			if px.A == 0 {
				continue
			}
			p := image.Pt(left+x, top+y)
			if p.In(dst.Bounds()) {
				dst.SetNRGBA(p.X, p.Y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
}

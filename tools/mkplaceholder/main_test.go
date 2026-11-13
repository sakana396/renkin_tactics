package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// 試し用の絵を作る道具（DES-NG-08）。
func TestMkplaceholderNG02(t *testing.T) {
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	files, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]File{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	t.Run("TC-DES-NG-08-90 全キャラの units・portraits、敵のクラスの絵、タイトルの背景を作る", func(t *testing.T) {
		for _, c := range cat.Characters() {
			for _, p := range []string{"units/" + c.ID + ".png", "portraits/" + c.ID + ".png"} {
				if _, ok := byPath[p]; !ok {
					t.Errorf("%s がない", p)
				}
			}
		}
		for _, c := range cat.Classes() {
			if _, ok := byPath["units/enemy_"+c.ID+".png"]; !ok {
				t.Errorf("敵 %s の絵がない", c.ID)
			}
		}
		if _, ok := byPath["title.png"]; !ok {
			t.Error("title.png がない")
		}
	})
	t.Run("TC-DES-NG-08-91 大きさは units 128・portraits 512 の正方形で、背景は透過でなく、四隅は透明", func(t *testing.T) {
		u := byPath["units/aoi.png"].Image.Bounds()
		p := byPath["portraits/aoi.png"].Image.Bounds()
		if u.Dx() != 128 || u.Dy() != 128 || p.Dx() != 512 || p.Dy() != 512 {
			t.Errorf("units %v portraits %v", u, p)
		}
		_, _, _, a := byPath["units/aoi.png"].Image.At(0, 0).RGBA()
		if a != 0 {
			t.Error("人型の外は透明のはず")
		}
		_, _, _, a = byPath["title.png"].Image.At(0, 0).RGBA()
		if a == 0 {
			t.Error("背景は不透明のはず")
		}
	})
	t.Run("TC-DES-NG-08-92 書き出した絵を、画面側の名前の規則で全部引ける（media の art/ へコピーした状態を模す）", func(t *testing.T) {
		media := fstest.MapFS{}
		for _, f := range files {
			var b bytes.Buffer
			if err := png.Encode(&b, f.Image); err != nil {
				t.Fatal(err)
			}
			media["art/"+f.Path] = &fstest.MapFile{Data: b.Bytes()}
		}
		store := ui.NewArtStore(media, &bytes.Buffer{}, cat)
		for _, c := range cat.Characters() {
			if _, ok := store.UnitArtPath(ui.ArtKey{Name: c.Name, Player: true}); !ok {
				t.Errorf("%s のユニットの絵を引けない", c.ID)
			}
			if _, ok := store.PortraitPath(c.Name); !ok {
				t.Errorf("%s の肖像を引けない", c.ID)
			}
		}
		for _, c := range cat.Classes() {
			if _, ok := store.UnitArtPath(ui.ArtKey{ClassName: c.Name}); !ok {
				t.Errorf("敵 %s の絵を引けない", c.ID)
			}
		}
	})
	t.Run("TC-DES-NG-08-93 -out に書き出し、media の下は拒否する", func(t *testing.T) {
		dir := t.TempDir()
		var out, errb bytes.Buffer
		if code := run([]string{"-out", dir}, &out, &errb); code != 0 {
			t.Fatalf("終了コード %d: %s", code, errb.String())
		}
		for _, p := range []string{"units/aoi.png", "portraits/aoi.png", "title.png"} {
			if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
				t.Error(err)
			}
		}
		// リポジトリのルートからの相対で、media の下を指す出力先。
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(filepath.Join("..", "..")); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(wd)
		errb.Reset()
		if code := run([]string{"-out", "data/alchemy/media/art"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "media") {
			t.Errorf("media の下を拒否するはず: %d %q", code, errb.String())
		}
		if _, err := os.Stat(filepath.Join("data", "alchemy", "media", "art", "units")); err == nil {
			t.Error("media の下に書いてしまった")
		}
		if !inMedia("data/alchemy/media") || inMedia("placeholder-out") || inMedia("data/alchemy/media-other") {
			t.Error("inMedia の判定")
		}
	})
}

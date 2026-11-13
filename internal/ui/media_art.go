package ui

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 画像の置き場（media の下）の名前の規則（DES-NG-08）。
const (
	artUnitDir     = "art/units"
	artPortraitDir = "art/portraits"
	artStoryDir    = "art/story"
	// artTitlePath はタイトル画面の背景。
	artTitlePath = "art/title.png"
	// artEnemyPrefix は敵の絵のファイル名の前に付ける語（units/enemy_<クラスの ID>.png）。
	artEnemyPrefix = "enemy_"
)

// ArtKey は、ユニットの絵を引くための名前と陣営だけの値（DES-NG-08）。
// 表示の値（InfoIcon）に載せるため、比較できる値だけで持つ。
type ArtKey struct {
	// Name は自軍のキャラの名前、ClassName はクラスの表示名、Player は自軍かどうか。
	Name, ClassName string
	Player          bool
}

// Empty は絵を引く手がかりがないこと。
func (k ArtKey) Empty() bool { return k.Name == "" && k.ClassName == "" }

// artKeyOf はユニットから絵の手がかりを作る（DES-NG-08）。
func artKeyOf(u tactics.Unit) ArtKey {
	return ArtKey{Name: u.Name, ClassName: u.ClassName, Player: u.Side == tactics.Player}
}

// artIndex は、画面に出る名前（キャラの名前・クラスの表示名）から ID を引く表（DES-NG-08）。
// 同じ名前が重なったときは、ID の昇順で先のものを使う。
type artIndex struct {
	charByName  map[string]string
	classByName map[string]string
}

// newArtIndex は目録から表を作る。cat が nil なら空の表を返す。
func newArtIndex(cat *defs.Catalog) artIndex {
	idx := artIndex{charByName: map[string]string{}, classByName: map[string]string{}}
	if cat == nil {
		return idx
	}
	for _, c := range cat.Characters() {
		if _, dup := idx.charByName[c.Name]; !dup {
			idx.charByName[c.Name] = c.ID
		}
	}
	for _, c := range cat.Classes() {
		if _, dup := idx.classByName[c.Name]; !dup {
			idx.classByName[c.Name] = c.ID
		}
	}
	return idx
}

// unitPath はユニットの絵のファイルの道筋を返す（DES-NG-08）。
// 自軍は units/<キャラの ID>.png、敵は units/enemy_<クラスの ID>.png。名前が目録にないときは ok が偽。
func (idx artIndex) unitPath(k ArtKey) (string, bool) {
	if k.Player {
		id, ok := idx.charByName[k.Name]
		if !ok {
			return "", false
		}
		return artUnitDir + "/" + id + ".png", true
	}
	id, ok := idx.classByName[k.ClassName]
	if !ok {
		return "", false
	}
	return artUnitDir + "/" + artEnemyPrefix + id + ".png", true
}

// portraitPath は名前から人物の絵（肖像）のファイルの道筋を返す（DES-NG-08）。キャラの名前でなければ ok が偽。
func (idx artIndex) portraitPath(name string) (string, bool) {
	id, ok := idx.charByName[name]
	if !ok {
		return "", false
	}
	return artPortraitDir + "/" + id + ".png", true
}

// storyBackdropPath は物語の ID から場面の背景のファイルの道筋を返す（DES-NG-08）。ID が空なら ok が偽。
func storyBackdropPath(storyID string) (string, bool) {
	if storyID == "" || strings.ContainsAny(storyID, "/\\") {
		return "", false
	}
	return artStoryDir + "/" + storyID + ".png", true
}

type artEntry struct {
	img image.Image
	eb  *ebiten.Image
}

// ArtStore は media の下の PNG を読み、覚えておく（DES-NG-08）。
// ない・壊れている画像は「なし」として扱い、ゲームを止めない。誤りは同じ画像につき 1 回だけ stderr に出す。
// 描画の goroutine だけから使う（排他は持たない）。
type ArtStore struct {
	fsys    fs.FS
	stderr  io.Writer
	idx     artIndex
	entries map[string]*artEntry
}

// NewArtStore は media（nil なら画像なし）と目録から作る。stderr が nil なら os.Stderr（DES-NG-08）。
func NewArtStore(media fs.FS, stderr io.Writer, cat *defs.Catalog) *ArtStore {
	if stderr == nil {
		stderr = os.Stderr
	}
	return &ArtStore{fsys: media, stderr: stderr, idx: newArtIndex(cat), entries: map[string]*artEntry{}}
}

// entry は道筋の画像を読んで覚える。ない・壊れているときは img が nil の項目を覚える。
// square が真で正方形でなければ、1 回だけ知らせる（縦横比は保って描くので、使えなくはしない）。
func (a *ArtStore) entry(path string, square bool) *artEntry {
	if e, ok := a.entries[path]; ok {
		return e
	}
	e := &artEntry{}
	a.entries[path] = e
	if a.fsys == nil || !fs.ValidPath(path) {
		return e
	}
	f, err := a.fsys.Open(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(a.stderr, "画像を開けません（%s）: %v\n", path, err)
		}
		return e
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		fmt.Fprintf(a.stderr, "画像を読めません（%s）: %v\n", path, err)
		return e
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		fmt.Fprintf(a.stderr, "画像の大きさが 0 です（%s）\n", path)
		return e
	}
	if square && b.Dx() != b.Dy() {
		fmt.Fprintf(a.stderr, "画像が正方形ではありません（%s: %d×%d）。縦横比を保って描きます\n", path, b.Dx(), b.Dy())
	}
	e.img = img
	return e
}

// Image は道筋の画像を返す。ない・壊れているときは nil。square は正方形を期待する画像（ユニット・肖像）かどうか。
func (a *ArtStore) Image(path string, square bool) image.Image {
	if a == nil {
		return nil
	}
	return a.entry(path, square).img
}

// Has は道筋の画像が使えること。
func (a *ArtStore) Has(path string, square bool) bool { return a.Image(path, square) != nil }

// Ebiten は道筋の画像を描画用に返す。使えなければ nil。
func (a *ArtStore) Ebiten(path string, square bool) *ebiten.Image {
	if a == nil {
		return nil
	}
	e := a.entry(path, square)
	if e.img == nil {
		return nil
	}
	if e.eb == nil {
		e.eb = ebiten.NewImageFromImage(e.img)
	}
	return e.eb
}

// UnitArtPath はユニットの絵があれば、その道筋を返す（DES-NG-08）。なければ ok が偽（今のタイルで描く）。
func (a *ArtStore) UnitArtPath(k ArtKey) (string, bool) {
	if a == nil || k.Empty() {
		return "", false
	}
	p, ok := a.idx.unitPath(k)
	if !ok || !a.Has(p, true) {
		return "", false
	}
	return p, true
}

// PortraitPath は名前の人物の絵（肖像）があれば、その道筋を返す（DES-NG-08）。
func (a *ArtStore) PortraitPath(name string) (string, bool) {
	if a == nil || name == "" {
		return "", false
	}
	p, ok := a.idx.portraitPath(name)
	if !ok || !a.Has(p, true) {
		return "", false
	}
	return p, true
}

// InfoArtPath は情報欄の絵の道筋を返す（DES-NG-08）。自軍は肖像、なければユニットの絵。敵はユニットの絵。
func (a *ArtStore) InfoArtPath(k ArtKey) (string, bool) {
	if k.Player {
		if p, ok := a.PortraitPath(k.Name); ok {
			return p, true
		}
	}
	return a.UnitArtPath(k)
}

// BackdropPath は背景の道筋があればそのまま、画像が使えなければ ok が偽を返す（DES-NG-08）。
func (a *ArtStore) BackdropPath(path string, ok bool) (string, bool) {
	if !ok || !a.Has(path, false) {
		return "", false
	}
	return path, true
}

// StoryPortraitPath は物語の段落の話者の肖像があれば、その道筋を返す（DES-NG-08）。
func (a *ArtStore) StoryPortraitPath(paragraph string) (string, bool) {
	name, ok := speakerName(paragraph)
	if !ok {
		return "", false
	}
	return a.PortraitPath(name)
}

// speakerName は、段落が「名前「…」」の形のとき、その名前を返す（DES-NG-08）。
//
// 段落の先頭から最初の「「」までを名前とする。名前は前後の空白（半角・全角）と、後ろに付く括弧書き（「アオイ（小声）」）を除く。
// 名前に句読点・閉じ括弧・改行を含むとき（地の文の途中の台詞）、名前が空のとき、閉じの「」」がないときは話者なし。
func speakerName(paragraph string) (string, bool) {
	const open, closing = "「", "」"
	i := strings.Index(paragraph, open)
	if i < 0 || !strings.Contains(paragraph[i:], closing) {
		return "", false
	}
	name := paragraph[:i]
	if strings.ContainsAny(name, "。、，．！？」\n\r") {
		return "", false
	}
	name = strings.TrimSpace(strings.Trim(name, "　"))
	// 括弧書きを外す。
	for _, pair := range [][2]string{{"（", "）"}, {"(", ")"}} {
		if strings.HasSuffix(name, pair[1]) {
			if j := strings.LastIndex(name, pair[0]); j >= 0 {
				name = strings.TrimSpace(strings.Trim(name[:j], "　"))
			}
		}
	}
	if name == "" || len([]rune(name)) > speakerNameMax {
		return "", false
	}
	return name, true
}

// speakerNameMax は話者の名前の最大の文字数。これより長い先頭は地の文とみなす。
const speakerNameMax = 12

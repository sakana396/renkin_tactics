// imagegen は、手元の Codex CLI の画像生成（組み込みの image_gen。ChatGPT のサブスクリプションの枠）で画像を作り、
// 新作の画像の置き場（data/alchemy/media/art）に正しい名前と大きさで置いて、台帳 ASSETS.md に行を足す道具（#210。IT08-PLAN 9 章 J、E-131・E-133）。
//
// 2 つの段に分ける。
//
//	generate: 指示文から Codex CLI で 1 枚作り、生の画像と生成の記録（<画像>.json）を リポジトリの外に保存する
//	place:    生の画像を種類に合わせて加工し（透過・切り出し・縮小）、art の下に置き、台帳に 1 行足す
//
// 使い方は tools/imagegen/README.md。名前・大きさ・台帳の書き方は DES-NG-08・DES-NG-12（docs/03_design/design-ng02-art.md）。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `使い方:
  go run ./tools/imagegen generate -prompt <指示文.txt> -out <生の画像.png>
  go run ./tools/imagegen place -in <生の画像.png> -kind <units|portraits|story|title> -id <ID> \
      -terms <利用条件> -checked-on <YYYY-MM-DD> -checked-by <確かめた人>
詳しくは tools/imagegen/README.md`

// run は引数を読んで段を実行し、終了コードを返す（0 成功、1 失敗、2 引数の誤り）。
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "place":
		return runPlace(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "段 %q が分かりません\n%s\n", args[0], usage)
		return 2
	}
}

// insideDir は p が dir の中（dir 自身を含む）かどうか。
func insideDir(p, dir string) bool {
	ap, err1 := filepath.Abs(p)
	ad, err2 := filepath.Abs(dir)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(ad, ap)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolvePath は存在する親まで遡ってリンクを解き、未作成の末尾を付け直す。
func resolvePath(path string) (string, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(p)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if _, statErr := os.Lstat(p); statErr == nil {
			return "", err
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", err
		}
		tail = append(tail, filepath.Base(p))
		p = parent
	}
}

// repositoryRoot は作業ディレクトリから go.mod を探す（DES-NG-12）。
func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return resolvePath(dir)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod が見つかりません")
		}
		dir = parent
	}
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	promptPath := fs.String("prompt", "", "指示文のファイル（英語の全文。台帳にそのまま書く）")
	out := fs.String("out", "", "生の画像の保存先（.png。リポジトリの外）")
	replace := fs.Bool("replace", false, "既存の生の画像と記録を置き換える")
	codex := fs.String("codex", "codex", "Codex CLI のコマンド")
	model := fs.String("model", "", "Codex の会話のモデル（省けば Codex の設定のまま）")
	media := fs.String("media", "data/alchemy/media", "画像の置き場（ここの中には保存しない）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *promptPath == "" || *out == "" {
		fmt.Fprintln(stderr, "-prompt と -out を指定してください")
		return 2
	}
	root, err := repositoryRoot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	resolvedOut, err := resolvePath(*out)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	resolvedMeta, err := resolvePath(*out + ".json")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	mediaPath := *media
	if !filepath.IsAbs(mediaPath) {
		mediaPath = filepath.Join(root, mediaPath)
	}
	resolvedMedia, err := resolvePath(mediaPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if insideDir(resolvedOut, root) || insideDir(resolvedMeta, root) || insideDir(resolvedOut, resolvedMedia) || insideDir(resolvedMeta, resolvedMedia) {
		fmt.Fprintln(stderr, "生の画像と記録はリポジトリと media の外に保存してください")
		return 2
	}
	if !*replace {
		for _, path := range []string{*out, *out + ".json"} {
			if _, err := os.Lstat(path); err == nil {
				fmt.Fprintf(stderr, "%s はもうあります（置き換えるときは -replace）\n", path)
				return 1
			} else if !os.IsNotExist(err) {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
	}
	raw, err := os.ReadFile(*promptPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	prompt := strings.TrimSpace(string(raw))
	if prompt == "" {
		fmt.Fprintln(stderr, "指示文が空です")
		return 2
	}
	home, err := codexHome()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ver, err := exec.Command(*codex, "--version").Output()
	if err != nil {
		fmt.Fprintf(stderr, "Codex CLI の版を調べられません: %v\n", err)
		return 1
	}
	usedModel := *model
	if usedModel == "" {
		if cfg, err := os.ReadFile(filepath.Join(home, "config.toml")); err == nil {
			usedModel = configuredModel(string(cfg))
		}
	}
	// 作業ディレクトリを空にする。read-only でもディスクは読め、~/.codex の設定も読まれる（DES-NG-12）。
	work, err := os.MkdirTemp("", "imagegen-")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(work)
	cmd := exec.Command(*codex, execArgs(work, *model)...)
	cmd.Stdin = strings.NewReader(wrapPrompt(prompt))
	var events bytes.Buffer
	cmd.Stdout = &events
	cmd.Stderr = stderr
	fmt.Fprintln(stderr, "Codex CLI で生成しています（数分かかることがあります）…")
	runErr := cmd.Run()
	res, perr := parseExecEvents(&events)
	if runErr != nil || perr != nil {
		fmt.Fprintf(stderr, "生成に失敗しました: %v %v\n", runErr, perr)
		return 1
	}
	src, err := findGenerated(home, res.ThreadID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	meta := genMeta{
		Tool:     strings.TrimSpace(string(ver)),
		Model:    usedModel,
		ThreadID: res.ThreadID,
		Date:     time.Now().Format("2006-01-02"),
		Prompt:   prompt,
		Source:   src,
	}
	mj, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(*out+".json", append(mj, '\n'), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "生の画像: %s\n生成の記録: %s.json\nCodex の保存先: %s\n", *out, *out, src)
	return 0
}

var idPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// artPath は種類と ID から、media からの道筋を作る（DES-NG-08 3 章の名前の規則）。
func artPath(kind, id string) (string, error) {
	spec, ok := kinds[kind]
	if !ok {
		return "", fmt.Errorf("種類 %q が分かりません（units・portraits・story・title）", kind)
	}
	if kind == "title" {
		return "art/title.png", nil
	}
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("ID %q は小文字の英数字と _ にしてください（敵は enemy_<クラスの ID>）", id)
	}
	return "art/" + spec.Dir + "/" + id + ".png", nil
}

func runPlace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("place", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("in", "", "生の画像（generate の -out）")
	metaPath := fs.String("meta", "", "生成の記録（省けば <生の画像>.json）")
	kind := fs.String("kind", "", "種類（units・portraits・story・title）")
	id := fs.String("id", "", "キャラ・クラス（enemy_<ID>）・物語の ID。title では使わない")
	chroma := fs.String("chroma", "auto", "緑の背景を透過にするか（auto は units だけ。on・off）")
	terms := fs.String("terms", "", "利用条件（規約の名前・版・条項と、生成物の利用を許す根拠）")
	checkedOn := fs.String("checked-on", "", "利用条件を確かめた日（YYYY-MM-DD）")
	checkedBy := fs.String("checked-by", "", "利用条件を確かめた人")
	media := fs.String("media", "data/alchemy/media", "画像の置き場（ASSETS.md のあるディレクトリ）")
	replace := fs.Bool("replace", false, "同じ名前の画像と台帳の行があれば置き換える")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *in == "" || *kind == "" {
		fmt.Fprintln(stderr, "-in と -kind を指定してください")
		return 2
	}
	rel, err := artPath(*kind, *id)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	spec := kinds[*kind]
	useChroma := spec.Chroma
	switch *chroma {
	case "auto":
	case "on":
		useChroma = true
	case "off":
		useChroma = false
	default:
		fmt.Fprintln(stderr, "-chroma は auto・on・off のどれかです")
		return 2
	}
	mp := *metaPath
	if mp == "" {
		mp = *in + ".json"
	}
	var meta genMeta
	mb, err := os.ReadFile(mp)
	if err != nil {
		fmt.Fprintf(stderr, "生成の記録を読めません: %v\n", err)
		return 1
	}
	if err := json.Unmarshal(mb, &meta); err != nil {
		fmt.Fprintf(stderr, "生成の記録 %s が壊れています: %v\n", mp, err)
		return 1
	}
	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(stderr, "画像を読めません: %v\n", err)
		return 1
	}
	dst := filepath.Join(*media, filepath.FromSlash(rel))
	if _, err := os.Stat(dst); err == nil && !*replace {
		fmt.Fprintf(stderr, "%s はもうあります（置き換えるときは -replace）\n", dst)
		return 1
	}
	row, err := makeRow(rowInput{
		File: rel, Kind: *kind, Chroma: useChroma,
		SrcW: src.Bounds().Dx(), SrcH: src.Bounds().Dy(),
		Meta: meta, Terms: *terms, CheckedOn: *checkedOn, CheckedBy: *checkedBy,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ledgerPath := filepath.Join(*media, "ASSETS.md")
	ledger, err := os.ReadFile(ledgerPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	newLedger, err := insertRow(string(ledger), rel, row, *replace)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	out, err := processImage(src, spec, useChroma)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(ledgerPath, []byte(newLedger), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "置いた画像: %s（%d×%d）\n台帳に行を足した: %s\n", dst, out.Rect.Dx(), out.Rect.Dy(), ledgerPath)
	if useChroma {
		fmt.Fprintf(stdout, "背景に似た色の残存画素: %d 個\n", residualGreen(out, src))
	}
	fmt.Fprintln(stdout, "次に `go test ./data` で台帳の検査を通し、tools/shot で画面を撮って確かめる")
	return 0
}

package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// ebitenPrefix は Ebitengine のパッケージの import パスの始まり（DES-NFR-01）。
const ebitenPrefix = "github.com/hajimehoshi/ebiten"

// timePackage は実時間を扱う標準ライブラリのパッケージ（DES-NFR-01、NFR-05）。
const timePackage = "time"

// tacticsPath は標準ライブラリだけを import してよいパッケージ（DES-PKG-05、ADR-0002）。
const tacticsPath = "github.com/sakana396/renkin_tactics/internal/core/tactics"

// pkgImports は go list で得た、1 つのパッケージが直接 import するもの。
type pkgImports struct {
	ImportPath string
	Imports    []string
}

// importViolations は core 以下のパッケージの直接の import から、規則に反するものを集める（DES-NFR-01）。
//   - どのパッケージも Ebitengine（ebitenPrefix で始まるもの）を import しない（NFR-04）
//   - どのパッケージも time を import しない（NFR-05）
//   - tactics は標準ライブラリ（std にあるもの）だけを import する（ADR-0002）
func importViolations(pkgs []pkgImports, std map[string]bool) []string {
	var out []string
	for _, p := range pkgs {
		for _, imp := range p.Imports {
			switch {
			case strings.HasPrefix(imp, ebitenPrefix):
				out = append(out, fmt.Sprintf("%s が Ebitengine（%s）を import している", p.ImportPath, imp))
			case imp == timePackage:
				out = append(out, fmt.Sprintf("%s が time を import している", p.ImportPath))
			}
			if p.ImportPath == tacticsPath && !std[imp] {
				out = append(out, fmt.Sprintf("%s が標準ライブラリでない %s を import している", p.ImportPath, imp))
			}
		}
	}
	return out
}

// goList は go list の出力を返す。失敗したらテストを止める。
func goList(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return out
}

// corePackages は internal/core 以下の各パッケージと、その直接の import を go list で得る（DES-NFR-01）。
//
// go test のキャッシュは、テストの中から起動した go list が読んだソースを追わない。
// internal/core の下にパッケージを足しても (cached) で前の結果が出るので、
// 手もとでは go test -count=1 で実行する（CI は毎回まっさらな環境なので影響しない）。
func corePackages(t *testing.T) []pkgImports {
	t.Helper()
	// テストの作業ディレクトリは internal/core なので、./... は internal/core 以下を指す。
	dec := json.NewDecoder(bytes.NewReader(goList(t, "-json", "./...")))
	var pkgs []pkgImports
	for {
		var p pkgImports
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("go list の出力を読めない: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

// stdPackages は標準ライブラリのパッケージの集合を go list std で得る。
func stdPackages(t *testing.T) map[string]bool {
	t.Helper()
	std := map[string]bool{}
	for _, p := range strings.Fields(string(goList(t, "std"))) {
		std[p] = true
	}
	return std
}

// TestCoreImports は core 以下のパッケージの依存を確かめる（DES-NFR-01）。
func TestCoreImports(t *testing.T) {
	pkgs := corePackages(t)
	std := stdPackages(t)

	t.Run("TC-NFR-04-01 / TC-NFR-05-01 core 以下の実際のパッケージは規則を守っている", func(t *testing.T) {
		found := false
		for _, p := range pkgs {
			if p.ImportPath == tacticsPath {
				found = true
			}
		}
		if !found {
			t.Fatalf("go list の結果に %s がない: %v", tacticsPath, pkgs)
		}
		if got := importViolations(pkgs, std); len(got) != 0 {
			t.Errorf("規則に反する import がある:\n%s", strings.Join(got, "\n"))
		}
	})

	const corePath = "github.com/sakana396/renkin_tactics/internal/core"
	cases := []struct {
		name string
		pkgs []pkgImports
		want []string
	}{
		{
			name: "TC-NFR-04-02 似た名前は違反にしない",
			pkgs: []pkgImports{
				{ImportPath: corePath, Imports: []string{
					"github.com/hajimehoshi/bitmapfont/v3",
					"example.com/hajimehoshi/ebiten/v2",
					"github.com/sakana396/renkin_tactics/internal/defs",
				}},
				{ImportPath: tacticsPath, Imports: []string{"errors", "fmt", "math/rand/v2"}},
			},
		},
		{
			name: "TC-NFR-04-03 Ebitengine とそのサブパッケージの import を見つける",
			pkgs: []pkgImports{
				{ImportPath: corePath, Imports: []string{"fmt", "github.com/hajimehoshi/ebiten/v2"}},
				{ImportPath: tacticsPath, Imports: []string{"github.com/hajimehoshi/ebiten/v2/inpututil"}},
			},
			want: []string{
				corePath + " が Ebitengine（github.com/hajimehoshi/ebiten/v2）を import している",
				tacticsPath + " が Ebitengine（github.com/hajimehoshi/ebiten/v2/inpututil）を import している",
				tacticsPath + " が標準ライブラリでない github.com/hajimehoshi/ebiten/v2/inpututil を import している",
			},
		},
		{
			name: "TC-NFR-04-04 tactics が標準ライブラリでないものを import したら見つける",
			pkgs: []pkgImports{
				{ImportPath: corePath, Imports: []string{"github.com/BurntSushi/toml"}},
				{ImportPath: tacticsPath, Imports: []string{
					"fmt",
					"github.com/sakana396/renkin_tactics/internal/defs",
					"github.com/BurntSushi/toml",
				}},
			},
			want: []string{
				tacticsPath + " が標準ライブラリでない github.com/sakana396/renkin_tactics/internal/defs を import している",
				tacticsPath + " が標準ライブラリでない github.com/BurntSushi/toml を import している",
			},
		},
		{
			name: "TC-NFR-05-02 time に似た名前は違反にしない",
			pkgs: []pkgImports{
				{ImportPath: corePath, Imports: []string{"example.com/time", "runtime"}},
				{ImportPath: tacticsPath, Imports: []string{"runtime"}},
			},
		},
		{
			name: "TC-NFR-05-03 time の import を見つける",
			pkgs: []pkgImports{
				{ImportPath: corePath, Imports: []string{"fmt", "time"}},
				{ImportPath: tacticsPath, Imports: []string{"time"}},
			},
			want: []string{
				corePath + " が time を import している",
				tacticsPath + " が time を import している",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := importViolations(tc.pkgs, std)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("違反 = %q、期待値 = %q", got, tc.want)
			}
		})
	}
}

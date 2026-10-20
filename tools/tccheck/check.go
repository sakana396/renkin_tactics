// Package main は、テストケース ID の整合を検査する（QA-02）。
//
// テスト仕様書・テストコード・トレーサビリティマトリクスの 3 か所で、
// テストケース ID が食い違っていないかを調べ、並列作業での衝突を CI で検出する。
package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// idPattern はテストケース ID と、トレーサビリティの範囲表記（〜NN）を表す。
// 例: TC-FR-ITM-04-01、TC-NFR-06-01〜04、TC-IF-02-01
var idPattern = regexp.MustCompile(`TC-((?:[A-Z]+-)+\d{2})-(\d{2})(?:〜(\d{2}))?`)

// codePattern はテストコードの文字列の先頭にあるテストケース ID（サブテスト名）を表す。
// 1 つのサブテストで複数のテストケースを確かめるときは「 / 」で区切って並べる。
// 期待値の文字列などの途中に現れる ID は対象にしない。
var codePattern = regexp.MustCompile(`(?:["\x60]| / )TC-((?:[A-Z]+-)+\d{2})-(\d{2})()`)

// Report は検査の結果を表す。Errors が 1 件でもあれば CI を失敗させる。
type Report struct {
	Errors   []string
	Warnings []string
}

func (r *Report) errorf(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Report) warnf(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// expand は 1 つの一致を ID の一覧に展開する（範囲表記を含む）。
func expand(m []string) []string {
	from, _ := strconv.Atoi(m[2])
	to := from
	if m[3] != "" {
		to, _ = strconv.Atoi(m[3])
	}
	var ids []string
	for n := from; n <= to; n++ {
		ids = append(ids, fmt.Sprintf("TC-%s-%02d", m[1], n))
	}
	return ids
}

// cells は Markdown の表の 1 行をセルに分ける。表の行でなければ nil を返す。
func cells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// specIDs はテスト仕様書の定義行（6 列の表で、1 列目がテストケース ID）を集める。
// 結果の記録など、列数の違う表は定義とみなさない。
func specIDs(fsys fs.FS, dir string, r *Report) map[string]string {
	defined := map[string]string{}
	files, _ := fs.Glob(fsys, path.Join(dir, "*.md"))
	for _, file := range files {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			r.errorf("%s: 読めません: %v", file, err)
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			c := cells(line)
			if len(c) != 6 || !strings.HasPrefix(c[0], "TC-") {
				continue
			}
			id := c[0]
			if m := idPattern.FindStringSubmatch(id); m == nil || m[0] != id || m[3] != "" {
				r.errorf("%s:%d: テストケース ID の形式が正しくありません: %s", file, i+1, id)
				continue
			}
			where := fmt.Sprintf("%s:%d", file, i+1)
			if prev, ok := defined[id]; ok {
				r.errorf("%s: テストケース ID %s が重複しています（最初の定義: %s）", where, id, prev)
				continue
			}
			defined[id] = where
		}
	}
	return defined
}

// codeIDs はテストコード（*_test.go）の文字列の先頭に書かれたテストケース ID を集める。
// 1 つのテストケースを複数のサブテストで確かめることがあるため、重複は許す。
func codeIDs(fsys fs.FS, r *Report) map[string]string {
	used := map[string]string{}
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			r.errorf("%s: 読めません: %v", p, err)
			return nil
		}
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for n := 1; sc.Scan(); n++ {
			for _, m := range codePattern.FindAllStringSubmatch(sc.Text(), -1) {
				for _, id := range expand(m) {
					if _, ok := used[id]; !ok {
						used[id] = fmt.Sprintf("%s:%d", p, n)
					}
				}
			}
		}
		return nil
	})
	return used
}

// traceIDs はトレーサビリティマトリクスに書かれたテストケース ID を集める（範囲表記を展開する）。
func traceIDs(fsys fs.FS, file string, r *Report) map[string]string {
	refs := map[string]string{}
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		r.errorf("%s: 読めません: %v", file, err)
		return refs
	}
	for i, line := range strings.Split(string(data), "\n") {
		for _, m := range idPattern.FindAllStringSubmatch(line, -1) {
			for _, id := range expand(m) {
				refs[id] = fmt.Sprintf("%s:%d", file, i+1)
			}
		}
	}
	return refs
}

// Check は 3 か所のテストケース ID の整合を検査する。
func Check(fsys fs.FS) Report {
	var r Report
	defined := specIDs(fsys, "docs/04_test", &r)
	used := codeIDs(fsys, &r)
	traced := traceIDs(fsys, "docs/traceability.md", &r)

	for _, id := range sortedKeys(used) {
		if _, ok := defined[id]; !ok {
			r.errorf("%s: テストコードの %s がテスト仕様書にありません", used[id], id)
		}
	}
	for _, id := range sortedKeys(traced) {
		if _, ok := defined[id]; !ok {
			r.errorf("%s: トレーサビリティの %s がテスト仕様書にありません", traced[id], id)
		}
	}
	for _, id := range sortedKeys(defined) {
		if _, ok := traced[id]; !ok {
			r.errorf("%s: %s がトレーサビリティマトリクスにありません", defined[id], id)
		}
		if _, ok := used[id]; !ok {
			r.warnf("%s: %s はテストコードにありません（手動のテストなら問題ありません）", defined[id], id)
		}
	}
	return r
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

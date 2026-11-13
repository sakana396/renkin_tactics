package data

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// 台帳（ASSETS.md）の列の数と名前（DES-NG-08）。
var ledgerColumns = []string{"ファイル", "作り方", "使ったツールと版・生成した日", "指示（プロンプト）または出典の URL", "利用条件", "確かめた日", "確かめた人"}

// ledgerMakes は「作り方」の欄が始まるべき語。
var ledgerMakes = []string{"AI 生成", "自作", "既存素材"}

// ledgerEmpty は、空とみなす欄の値。
var ledgerEmpty = map[string]bool{"-": true, "ー": true, "－": true, "なし": true, "未定": true, "不明": true, "tbd": true, "todo": true, "?": true, "？": true}

var ledgerDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// artNameRule は art の下のファイルの名前の規則（DES-NG-08 3 章）。
// 規則に合わないファイルは画面に出ないので、台帳に載せていても誤りとして報告する。
// units/enemy_<クラスの ID>.png も、小文字の英数字とアンダースコアの名前として units/ の規則に含まれる。
// terrain/<地形の ID>.png（変種は _2〜_4 を付ける）も同じ規則に含まれる（DES-NG-10 フェーズ 2）。
// ID が実在するか（キャラ・クラス・物語・地形）は検査しない（data パッケージは定義を知らない）。
var artNameRule = regexp.MustCompile(`^art/(title\.png|(units|portraits|story|terrain)/[a-z0-9_]+\.png)$`)

// ledgerDirs は台帳で管理するディレクトリ（media の下）。
var ledgerDirs = []string{"art", "audio"}

// ledgerExempt は台帳に書かなくてよいファイル名。
var ledgerExempt = map[string]bool{"README.md": true, "ASSETS.md": true}

// CheckMediaLedger は media の台帳 ASSETS.md を検査し、問題の文を並べて返す（DES-NG-08）。問題がなければ nil。
//
// 検査すること: art と audio の下のすべてのファイル（README.md・ASSETS.md を除く）が台帳の「ファイル」欄にあること、
// 台帳の行のファイルが実在すること、同じファイルを 2 行に書かないこと、必須の欄が空でないこと、
// 「作り方」が AI 生成・自作・既存素材のどれかで始まること、「確かめた日」が実在する YYYY-MM-DD であること、
// art の下のファイル名が名前の規則（artNameRule）に合うこと。HTML コメント（<!-- -->）の中の行は読まない。
func CheckMediaLedger(media fs.FS) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	files := map[string]bool{}
	for _, dir := range ledgerDirs {
		err := fs.WalkDir(media, dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && !ledgerExempt[path.Base(p)] {
				files[p] = true
			}
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			add("%s を調べられません: %v", dir, err)
		}
	}

	rows, err := readLedger(media)
	if err != nil {
		add("台帳 ASSETS.md を読めません: %v", err)
		rows = nil
	}
	listed := map[string]bool{}
	for _, r := range rows {
		file := r.cells[0]
		if file == "" {
			add("台帳 %d 行目: 「ファイル」が空です", r.line)
			continue
		}
		if listed[file] {
			add("台帳 %d 行目: %s が重ねて書かれています", r.line, file)
		}
		listed[file] = true
		if !fs.ValidPath(file) {
			add("台帳 %d 行目: ファイルの道筋 %q が正しくありません（media からの相対で、スラッシュ区切り）", r.line, file)
		} else if info, err := fs.Stat(media, file); err != nil || info.IsDir() {
			add("台帳 %d 行目: %s が実在しません", r.line, file)
		} else if !underLedgerDir(file) {
			add("台帳 %d 行目: %s は art か audio の下ではありません", r.line, file)
		}
		for i := 1; i < len(ledgerColumns); i++ {
			if ledgerEmpty[strings.ToLower(r.cells[i])] || r.cells[i] == "" {
				add("台帳 %d 行目（%s）: 「%s」が空です", r.line, file, ledgerColumns[i])
			}
		}
		if m := r.cells[1]; m != "" && !hasAnyPrefix(m, ledgerMakes) {
			add("台帳 %d 行目（%s）: 「作り方」は %s のどれかで始めてください: %q", r.line, file, strings.Join(ledgerMakes, "・"), m)
		}
		if d := r.cells[5]; d != "" && !ledgerEmpty[d] {
			if !ledgerDate.MatchString(d) {
				add("台帳 %d 行目（%s）: 「確かめた日」は YYYY-MM-DD の形にしてください: %q", r.line, file, d)
			} else if _, err := time.Parse("2006-01-02", d); err != nil {
				add("台帳 %d 行目（%s）: 「確かめた日」が実在しない日付です: %q", r.line, file, d)
			}
		}
	}
	// art の下のファイル名の規則。ディスクのファイルと、台帳に書かれた道筋の両方を見る（重ねて報告しない）。
	named := map[string]bool{}
	checkName := func(f string) {
		if named[f] || !strings.HasPrefix(f, "art/") || !fs.ValidPath(f) || artNameRule.MatchString(f) {
			return
		}
		named[f] = true
		add("%s は art の名前の規則に合いません（units/<小文字の英数字と _>.png、units/enemy_….png、portraits/….png、story/….png、terrain/<地形の ID>.png、title.png。画面に出ません）", f)
	}
	for f := range listed {
		checkName(f)
	}
	for f := range files {
		checkName(f)
	}
	var missing []string
	for f := range files {
		if !listed[f] {
			missing = append(missing, f)
		}
	}
	sort.Strings(missing)
	for _, f := range missing {
		add("%s が台帳 ASSETS.md にありません（出典と利用条件を 1 行書いてください）", f)
	}
	return problems
}

func underLedgerDir(p string) bool {
	for _, d := range ledgerDirs {
		if strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

type ledgerRow struct {
	line  int
	cells []string // len(ledgerColumns) 個。足りなければ空で埋める。
}

// stripHTMLComments は 1 行から HTML コメント（<!-- -->）を取り除く。コメントが次の行へ続くときは inComment を真にして返す。
func stripHTMLComments(line string, inComment bool) (string, bool) {
	var b strings.Builder
	for line != "" {
		if inComment {
			i := strings.Index(line, "-->")
			if i < 0 {
				return b.String(), true
			}
			line, inComment = line[i+3:], false
			continue
		}
		i := strings.Index(line, "<!--")
		if i < 0 {
			b.WriteString(line)
			break
		}
		b.WriteString(line[:i])
		line, inComment = line[i+4:], true
	}
	return b.String(), inComment
}

// readLedger は ASSETS.md の「ファイル」で始まる見出しの表の行を読む。囲み（```）と HTML コメントの中は読まない。
func readLedger(media fs.FS) ([]ledgerRow, error) {
	f, err := media.Open("ASSETS.md")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []ledgerRow
	inFence, inTable, inComment := false, false, false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		raw := sc.Text()
		if !inFence {
			raw, inComment = stripHTMLComments(raw, inComment)
		}
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitRow(line)
		if !inTable {
			if len(cells) > 0 && cells[0] == ledgerColumns[0] {
				inTable = true
				if len(cells) != len(ledgerColumns) {
					return nil, fmt.Errorf("%d 行目: 表の列は %d 個（%s）にしてください", n, len(ledgerColumns), strings.Join(ledgerColumns, "・"))
				}
			}
			continue
		}
		if isSeparator(cells) {
			continue
		}
		for len(cells) < len(ledgerColumns) {
			cells = append(cells, "")
		}
		if len(cells) > len(ledgerColumns) {
			return nil, fmt.Errorf("%d 行目: 列が多すぎます（欄の中の | は使えません）", n)
		}
		cells[0] = strings.Trim(cells[0], "` ")
		rows = append(rows, ledgerRow{line: n, cells: cells})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !inTable {
		return nil, errors.New("「ファイル」で始まる見出しの表がありません")
	}
	return rows, nil
}

func splitRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

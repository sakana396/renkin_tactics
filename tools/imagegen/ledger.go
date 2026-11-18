package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// genMeta は generate が生の画像の隣（<画像>.json）に書く記録。place が台帳の行に使う。
type genMeta struct {
	Tool     string `json:"tool"`      // Codex CLI の版（`codex --version` の出力）
	Model    string `json:"model"`     // Codex の会話のモデル（設定または -model）。画像のモデル名は表示されない
	ThreadID string `json:"thread_id"` // 生成したスレッドの ID
	Date     string `json:"date"`      // 生成した日（YYYY-MM-DD）
	Prompt   string `json:"prompt"`    // 指示文の全文（Codex への依頼の包みは含まない）
	Source   string `json:"source"`    // Codex CLI が保存した元の画像の道筋
}

// rowInput は台帳の 1 行を組み立てる材料。
type rowInput struct {
	File      string  // media からの道筋（例: art/story/ch01_before.png）
	Kind      string  // units・portraits・story・title
	Chroma    bool    // 緑の背景を透過にしたか
	SrcW      int     // 元の画像の幅
	SrcH      int     // 元の画像の高さ
	Meta      genMeta // 生成の記録
	Terms     string  // 利用条件
	CheckedOn string  // 確かめた日
	CheckedBy string  // 確かめた人
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// oneLine は欄に入れる文を 1 行にする（改行と連続する空白を 1 つの空白に）。
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// makeDescription は「作り方」の欄の文（加工の手順）を組み立てる。
func makeDescription(in rowInput) string {
	spec := kinds[in.Kind]
	src := fmt.Sprintf("元の %d×%d の画像", in.SrcW, in.SrcH)
	if in.Chroma {
		return fmt.Sprintf("AI 生成（Codex CLI の画像生成）。%s（背景は緑）を、tools/imagegen で、緑の背景を透過にし"+
			"（クロマキー: 緑らしさ k = G − max(R,B) が %d 以下は不透明、%d 以上は透明、その間は直線で α を決める。"+
			"α が 1 未満の画素と、透明の画素から %d px 以内の画素は G を max(R,B) に揃える）、"+
			"α > 0.04 の範囲で切り出して %.0f%% の余白を足し、足元をそろえて正方形にし、乗算済みアルファで %d×%d の PNG に縮小した（Catmull-Rom）。"+
			"縮小のあと、α %d 以下で k が %d 以上の画素は透明にし、残る半透明の画素は G を max(R,B) に揃えた",
			src, keyOpaque, keyClear, spillBand, trimMargin*100, spec.W, spec.H, residualAlpha, residualKey)
	}
	return fmt.Sprintf("AI 生成（Codex CLI の画像生成）。%sを、tools/imagegen で、中央を残して %d:%d の比に切り（はみ出しを左右か上下で等しく切る）、"+
		"%d×%d の PNG に縮小した（Catmull-Rom）。ほかの加工はしていない", src, spec.W, spec.H, spec.W, spec.H)
}

// makeRow は台帳 ASSETS.md の 1 行を組み立てる（DES-NG-08 5 章の 7 列）。
// 欄の中に | は書けない（表が崩れる）ので、含まれていれば誤りにする。必須の欄が空のときも誤り。
func makeRow(in rowInput) (string, error) {
	if _, ok := kinds[in.Kind]; !ok {
		return "", fmt.Errorf("種類 %q が分かりません", in.Kind)
	}
	if !datePattern.MatchString(in.Meta.Date) {
		return "", fmt.Errorf("生成した日 %q は YYYY-MM-DD の形にしてください", in.Meta.Date)
	}
	if !datePattern.MatchString(in.CheckedOn) {
		return "", fmt.Errorf("確かめた日 %q は YYYY-MM-DD の形にしてください", in.CheckedOn)
	}
	tool := oneLine(in.Meta.Tool)
	if in.Meta.Model != "" {
		tool += "（codex exec の組み込みの image_gen ツール。Codex の会話のモデルは " + oneLine(in.Meta.Model) + "。画像のモデル名は表示されない）"
	} else {
		tool += "（codex exec の組み込みの image_gen ツール。画像のモデル名は表示されない）"
	}
	tool += "、" + in.Meta.Date + "。加工は tools/imagegen"
	cells := []string{
		"`" + in.File + "`",
		makeDescription(in),
		tool,
		"「" + oneLine(in.Meta.Prompt) + "」（全文。改行は空白に置き換えた）。Codex の会話のモデルに、wrapPrompt の依頼文で指示を変えずに image_gen へ渡すよう頼んだ。image_gen が実際に受け取った文は確認していない",
		oneLine(in.Terms),
		in.CheckedOn,
		oneLine(in.CheckedBy),
	}
	names := []string{"ファイル", "作り方", "使ったツールと版・生成した日", "指示（プロンプト）", "利用条件", "確かめた日", "確かめた人"}
	for i, c := range cells {
		if strings.Contains(c, "|") {
			return "", fmt.Errorf("「%s」に | が含まれています（台帳の表が崩れるので使えません）", names[i])
		}
	}
	if oneLine(in.Meta.Tool) == "" || oneLine(in.Meta.Prompt) == "" || oneLine(in.Terms) == "" || oneLine(in.CheckedBy) == "" {
		return "", errors.New("ツールの版・指示文・利用条件・確かめた人は空にできません")
	}
	return "| " + strings.Join(cells, " | ") + " |", nil
}

// insertRow は台帳の本文に行を足す。「| ファイル |」で始まる見出しの表（囲み ``` の外）の最後の行の次に入れる。
// 同じファイルの行があれば、replace が真なら置き換え、偽なら誤り。
func insertRow(ledger, file, row string, replace bool) (string, error) {
	lines := strings.Split(ledger, "\n")
	inFence := false
	header, last, existing := -1, -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if header < 0 {
			if strings.HasPrefix(t, "|") && strings.TrimSpace(strings.Split(strings.TrimPrefix(t, "|"), "|")[0]) == "ファイル" {
				header, last = i, i
			}
			continue
		}
		if !strings.HasPrefix(t, "|") {
			break
		}
		last = i
		first := strings.Trim(strings.TrimSpace(strings.Split(strings.TrimPrefix(t, "|"), "|")[0]), "` ")
		if first == file {
			existing = i
		}
	}
	if header < 0 {
		return "", errors.New("台帳に「ファイル」で始まる見出しの表がありません")
	}
	if existing >= 0 {
		if !replace {
			return "", fmt.Errorf("台帳に %s の行がもうあります（置き換えるときは -replace）", file)
		}
		lines[existing] = row
		return strings.Join(lines, "\n"), nil
	}
	out := append([]string{}, lines[:last+1]...)
	out = append(out, row)
	out = append(out, lines[last+1:]...)
	return strings.Join(out, "\n"), nil
}

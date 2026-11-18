package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// wrapPrompt は、Codex CLI に渡す依頼文を組み立てる。指示文はそのまま、印の間に入れる。
// Codex には、組み込みの image_gen を 1 回だけ使い、ファイルを作らず、保存した PNG の道筋だけを返すよう頼む。
func wrapPrompt(prompt string) string {
	var b strings.Builder
	b.WriteString("Use the built-in image_gen tool exactly once to generate one image from the prompt between the markers below. ")
	b.WriteString("Pass the prompt as-is; do not rewrite or add to it. ")
	b.WriteString("Do not run shell commands and do not create or edit files. ")
	b.WriteString("After the image is generated, reply with only the absolute path of the saved PNG file.\n\n")
	b.WriteString("<<<PROMPT\n")
	b.WriteString(strings.TrimSpace(prompt))
	b.WriteString("\nPROMPT>>>\n")
	return b.String()
}

// execArgs は `codex exec` の引数。読み取り専用のサンドボックスで、セッションを残さず、出来事を JSONL で受け取る。
func execArgs(workdir, model string) []string {
	args := []string{"exec", "--skip-git-repo-check", "--ephemeral", "-c", `approval_policy="never"`, "-s", "read-only", "-C", workdir, "--json"}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args, "-")
}

// execResult は `codex exec --json` の出来事から読み取ったもの。
type execResult struct {
	ThreadID string // thread.started の thread_id。生成した画像の置き場の名前になる
	Message  string // 最後の agent_message の文（画像の道筋が入るはず）
}

// parseExecEvents は `codex exec --json` の JSONL を読み、スレッドの ID と最後の返事を取り出す。
// 誤り（type が error・turn.failed）があれば、その内容を誤りとして返す。
func parseExecEvents(r io.Reader) (execResult, error) {
	var res execResult
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var ev struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Message  string `json:"message"`
			Error    *struct {
				Message string `json:"message"`
			} `json:"error"`
			Item *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // 形の違う行は読み飛ばす
		}
		switch ev.Type {
		case "thread.started":
			res.ThreadID = ev.ThreadID
		case "item.completed":
			if ev.Item != nil && ev.Item.Type == "agent_message" {
				res.Message = strings.TrimSpace(ev.Item.Text)
			}
		case "error":
			return res, fmt.Errorf("Codex CLI が誤りを返しました: %s", ev.Message)
		case "turn.failed":
			msg := ""
			if ev.Error != nil {
				msg = ev.Error.Message
			}
			return res, fmt.Errorf("Codex CLI の処理が失敗しました: %s", msg)
		}
	}
	if err := sc.Err(); err != nil {
		return res, err
	}
	if res.ThreadID == "" {
		return res, errors.New("Codex CLI の出力に thread.started がありません")
	}
	return res, nil
}

// findGenerated は、Codex CLI が画像を保存する <codexHome>/generated_images/<スレッドの ID>/ の PNG を探す。
// 1 つもなければ誤り。2 つ以上あれば誤り（1 回だけ生成するよう頼んでいるので、どれを使うか決められない）。
func findGenerated(codexHome, threadID string) (string, error) {
	if threadID == "" || strings.ContainsAny(threadID, `/\`) || threadID == "." || threadID == ".." {
		return "", fmt.Errorf("スレッドの ID %q が正しくありません", threadID)
	}
	dir := filepath.Join(codexHome, "generated_images", threadID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("生成した画像の置き場 %s を読めません（画像が作られなかった可能性があります）: %w", dir, err)
	}
	var pngs []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".png") {
			pngs = append(pngs, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(pngs)
	switch len(pngs) {
	case 0:
		return "", fmt.Errorf("%s に PNG がありません（画像が作られなかった可能性があります）", dir)
	case 1:
		return pngs[0], nil
	default:
		return "", fmt.Errorf("%s に PNG が %d 個あります。候補を確認し、生成の記録を手で用意してから place に渡してください", dir, len(pngs))
	}
}

var configModel = regexp.MustCompile(`^\s*model\s*=\s*"([^"]+)"`)

// configuredModel は Codex の設定ファイル（config.toml）の最上位の model を返す。なければ空。
// 最初の表（[...]）より前の行だけを見る。
func configuredModel(config string) string {
	for _, line := range strings.Split(config, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			break
		}
		if m := configModel.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// codexHome は Codex CLI の設定の置き場（CODEX_HOME、なければ ~/.codex）。
func codexHome() (string, error) {
	if h := os.Getenv("CODEX_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

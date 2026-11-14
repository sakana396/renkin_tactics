package defload_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 章の目的の文 goal（DES-NG-11、DES-NG05 の定義の形式の変更）を確かめる。
func TestChapterGoalNG05(t *testing.T) {
	t.Run("TC-DES-NG-11-01 goal を読み込む", func(t *testing.T) {
		m := baseFS(t)
		replace("chapters.toml", `name = "国境の砦"`, "name = \"国境の砦\"\ngoal = \"砦の敵将を倒す\"")(t, m)
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		if g := cat.Chapters()[0].Goal; g != "砦の敵将を倒す" {
			t.Fatal(g)
		}
	})
	t.Run("TC-DES-NG-11-02 goal を書かない章は空で通る（base は変わらない）", func(t *testing.T) {
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		for _, ch := range cat.Chapters() {
			if ch.Goal != "" {
				t.Errorf("base の章 %s に goal がある: %q", ch.ID, ch.Goal)
			}
		}
	})
	t.Run("TC-DES-NG-11-03 長すぎる・空白だけの goal は誤り、上限ちょうどは通る", func(t *testing.T) {
		for _, tc := range []struct {
			goal string
			want string
		}{
			{strings.Repeat("あ", 61), "目的の長さ 61 文字は 60 文字以下にしてください"},
			{"   ", "目的が空白だけです"},
			{strings.Repeat("あ", 60), ""},
		} {
			m := baseFS(t)
			replace("chapters.toml", `name = "国境の砦"`, "name = \"国境の砦\"\ngoal = \""+tc.goal+"\"")(t, m)
			_, err := defload.Load(m)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				continue
			}
			var errs defload.Errors
			if !errors.As(err, &errs) || !strings.Contains(err.Error(), "chapters[0].goal: "+tc.want) {
				t.Errorf("%q: %v", tc.goal, err)
			}
		}
	})
	t.Run("TC-DES-NG-11-04 alchemy の 3 章に goal がある", func(t *testing.T) {
		cat, err := defload.Load(data.Alchemy())
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"入口の広場の野盗の頭目を倒す", "先遣隊長ドルクを倒す", "将軍ベルナドを倒す"}
		chs := cat.Chapters()
		if len(chs) != 3 {
			t.Fatal(len(chs))
		}
		for i, w := range want {
			if chs[i].Goal != w {
				t.Errorf("章 %d: %q", i+1, chs[i].Goal)
			}
		}
	})
}

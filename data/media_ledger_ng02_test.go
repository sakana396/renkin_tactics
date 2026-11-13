package data

import (
	"strings"
	"testing"
	"testing/fstest"
)

const ledgerHead = "# 台帳\n\n| ファイル | 作り方 | 使ったツールと版・生成した日 | 指示（プロンプト）または出典の URL | 利用条件 | 確かめた日 | 確かめた人 |\n|---|---|---|---|---|---|---|\n"

// ledgerRowOf は欄がすべて埋まった正しい行（file だけ変える）。
func ledgerRowOf(file string) string {
	return "| `" + file + "` | AI 生成 | ツール v1、2026-11-13 | 指示の全文 | 規約の第 4 条で商用利用可 | 2026-11-13 | 開発者 |\n"
}

func mediaFS(ledger string, files ...string) fstest.MapFS {
	m := fstest.MapFS{"README.md": {Data: []byte("r")}}
	if ledger != "" {
		m["ASSETS.md"] = &fstest.MapFile{Data: []byte(ledger)}
	}
	for _, f := range files {
		m[f] = &fstest.MapFile{Data: []byte("x")}
	}
	return m
}

// 台帳の検査（DES-NG-08）。画像と音のファイルが、出典と利用条件の行を持つことを確かめる。
func TestCheckMediaLedgerNG02(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
		// want は問題の文に含まれるべき語（順不同）。空なら問題なし。
		want []string
	}{
		{"TC-DES-NG-08-01 画像と音がすべて台帳にあり、欄が埋まっていれば問題なし",
			mediaFS(ledgerHead+ledgerRowOf("art/units/aoi.png")+ledgerRowOf("audio/se/bomb.wav"), "art/units/aoi.png", "audio/se/bomb.wav"), nil},
		{"TC-DES-NG-08-02 台帳にない画像は落ちる",
			mediaFS(ledgerHead+ledgerRowOf("art/units/aoi.png"), "art/units/aoi.png", "art/portraits/aoi.png"), []string{"art/portraits/aoi.png が台帳"}},
		{"TC-DES-NG-08-03 台帳にない音は落ちる（audio の下も対象）",
			mediaFS(ledgerHead, "audio/bgm/title.wav"), []string{"audio/bgm/title.wav が台帳"}},
		{"TC-DES-NG-08-04 必須の欄が空（「なし」「-」も空）なら落ちる",
			mediaFS(ledgerHead+"| `art/title.png` | AI 生成 | ツール、2026-11-13 | 指示 | なし | 2026-11-13 | - |\n", "art/title.png"),
			[]string{"「利用条件」が空", "「確かめた人」が空"}},
		{"TC-DES-NG-08-05 台帳の行のファイルが実在しなければ落ちる",
			mediaFS(ledgerHead + ledgerRowOf("art/units/ghost.png")), []string{"art/units/ghost.png が実在しません"}},
		{"TC-DES-NG-08-06 同じファイルの 2 行・作り方の不正・日付の不正は落ちる",
			mediaFS(ledgerHead+ledgerRowOf("art/a.png")+ledgerRowOf("art/a.png")+
				"| `art/b.png` | 買った | ツール | 指示 | 条件 | 2026-11-13 | 人 |\n"+
				"| `art/c.png` | 自作 | ツール | 指示 | 条件 | 昨日 | 人 |\n", "art/a.png", "art/b.png", "art/c.png"),
			[]string{"art/a.png が重ねて", "「作り方」は", "「確かめた日」は YYYY-MM-DD"}},
		{"TC-DES-NG-08-07 README.md と ASSETS.md は対象外。ファイルがなく表が空なら問題なし",
			mediaFS(ledgerHead, "art/README.md", "audio/README.md"), nil},
		{"TC-DES-NG-08-08 台帳がない・表の見出しがないときは落ちる",
			mediaFS("", "art/a.png"), []string{"台帳 ASSETS.md を読めません"}},
		{"TC-DES-NG-08-09 囲み（```）の中の見本の行は読まない。art が無くても落ちない",
			mediaFS("# 台帳\n```\n| `art/sample.png` | AI 生成 | a | b | c | 2026-01-01 | d |\n```\n" + ledgerHead), nil},
		{"TC-DES-NG-08-10 media の外を指す道筋・art と audio 以外・列が足りない行は落ちる",
			mediaFS(ledgerHead+ledgerRowOf("../x.png")+ledgerRowOf("other/y.png")+"| `art/z.png` | 自作 |\n", "other/y.png", "art/z.png"),
			[]string{"道筋 \"../x.png\" が正しくありません", "art か audio の下ではありません", "art/z.png）: 「使ったツールと版・生成した日」が空"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMediaLedger(tt.fsys)
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Fatalf("問題なしのはずが %v", got)
				}
				return
			}
			all := strings.Join(got, "\n")
			for _, w := range tt.want {
				if !strings.Contains(all, w) {
					t.Errorf("問題に %q がない: %v", w, got)
				}
			}
		})
	}
}

// 実際に埋め込まれた media に台帳の検査を使う（DES-NG-08）。画像・音を置いて台帳を書き忘れると、ここで落ちる。
func TestAlchemyMediaLedgerNG02(t *testing.T) {
	t.Run("TC-DES-NG-08-11 実際の data/alchemy/media が台帳と一致している", func(t *testing.T) {
		if problems := CheckMediaLedger(AlchemyMedia()); len(problems) != 0 {
			t.Fatalf("台帳 data/alchemy/media/ASSETS.md の問題:\n%s", strings.Join(problems, "\n"))
		}
	})
}

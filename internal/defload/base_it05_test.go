package defload_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 反復 5 の本体の物語（DES-CMP-09、DES-IT05 3.3 節）。途中の版（ch01〜ch03・ch09）の分を #102 で確かめる。
func TestBaseStoryIt05(t *testing.T) {
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	// 反復 4 までの ch03_after の結び（結末）。#102 で ch09_after へ移した。
	ending := []string{
		"王都の広場には、生き残った人々が集まり、亡くなった者の名を一人ずつ読み上げた。その声は、日が暮れても終わらなかった。",
		"レオン「勝ったんじゃない。終わらせただけだ」",
		"帝国との戦がこれで終わるのかは、まだ誰にも分からない。それでも、工房の煙突から、久しぶりに煙が上がった。",
	}

	t.Run("TC-FR-CMP-07-05 新しい章の章前・章後の物語は 4〜6 段落・各 200 文字以下", func(t *testing.T) {
		// 途中の版で足した章は ch09 だけ（ch04〜ch08 は #103・#104 で足し、この表に加える）。
		for _, id := range []string{"ch09_before", "ch09_after"} {
			s, ok := cat.Story(id)
			if !ok {
				t.Fatalf("物語 %s がない", id)
			}
			if n := len(s.Paragraphs); n < 4 || n > 6 {
				t.Errorf("物語 %s の段落の数 = %d", id, n)
			}
			for j, p := range s.Paragraphs {
				if p == "" || utf8.RuneCountInString(p) > 200 {
					t.Errorf("物語 %s の段落 %d の長さ = %d", id, j, utf8.RuneCountInString(p))
				}
			}
		}
	})

	t.Run("TC-FR-CMP-07-05 結末は ch09_after にあり、ch03_after は結末の文を含まず次の章（ヴァルトの門）へつなぐ", func(t *testing.T) {
		after, _ := cat.Story("ch09_after")
		joined := strings.Join(after.Paragraphs, "\n")
		for _, p := range ending {
			if !strings.Contains(joined, p) {
				t.Errorf("ch09_after に結末の文 %q がない", p)
			}
		}
		// 結末の最後の段落は、反復 4 までと同じ結び。
		if after.Paragraphs[len(after.Paragraphs)-1] != ending[len(ending)-1] {
			t.Errorf("ch09_after の最後の段落 = %q", after.Paragraphs[len(after.Paragraphs)-1])
		}
		bridge, _ := cat.Story("ch03_after")
		text := strings.Join(bridge.Paragraphs, "\n")
		for _, p := range ending {
			if strings.Contains(text, p) {
				t.Errorf("ch03_after に結末の文 %q が残っている", p)
			}
		}
		if !strings.Contains(text, "ヴァルトの門") {
			t.Errorf("ch03_after が次の章へつながない: %q", bridge.Paragraphs)
		}
		// 章前の物語のボスの名前が、マップのボスと一致する。
		before, _ := cat.Story("ch09_before")
		if !strings.Contains(strings.Join(before.Paragraphs, "\n"), "ゲルハルト") {
			t.Errorf("ch09_before にボスの名前がない")
		}
	})
}

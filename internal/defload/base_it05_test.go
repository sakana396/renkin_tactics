package defload_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
)

// 反復 5 の本体の物語（DES-CMP-09、DES-IT05 3.3 節）。途中の版（ch01〜ch06・ch09）の分を確かめる。
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
		// ch07・ch08 は #104 で足し、この表に加える。
		for _, id := range []string{"ch04_before", "ch04_after", "ch05_before", "ch05_after", "ch06_before", "ch06_after", "ch09_before", "ch09_after"} {
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

	t.Run("TC-FR-CMP-07-05 戦死しうる仲間の名前を物語に出さない", func(t *testing.T) {
		// 初期名簿と加入者から主人公以外を求める（DES-STY-01、DES-IT02 10.8 節）。
		members := append([]string(nil), cat.Game.Party...)
		for _, chapter := range cat.Chapters() {
			members = append(members, chapter.Joins...)
		}
		var names []string
		for _, id := range members {
			ch, ok := cat.Character(id)
			if !ok || ch.Name == "" {
				t.Fatalf("仲間 %s の名前がない", id)
			}
			if !ch.Lord {
				names = append(names, ch.Name)
			}
		}
		if len(names) == 0 {
			t.Fatal("戦死しうる仲間がいない")
		}
		for _, id := range []string{"ch04_before", "ch04_after", "ch05_before", "ch05_after", "ch06_before", "ch06_after", "ch09_before"} {
			story, ok := cat.Story(id)
			if !ok {
				t.Fatalf("物語 %s がない", id)
			}
			for j, p := range story.Paragraphs {
				for _, name := range names {
					if strings.Contains(p, name) {
						t.Errorf("物語 %s の段落 %d に戦死しうる仲間 %s の名前がある", id, j, name)
					}
				}
			}
		}
	})

	t.Run("TC-FR-CMP-07-05 ch04〜ch06 は退く後衛・関所・山道から補給と森へつなぐ", func(t *testing.T) {
		for _, tc := range []struct{ id, boss, after string }{
			{"ch04", "オスカー", "関所"}, {"ch05", "ディートリヒ", "山道"}, {"ch06", "ロルフ", "荷駄"},
		} {
			before, ok := cat.Story(tc.id + "_before")
			if !ok {
				t.Fatalf("%s_before がない", tc.id)
			}
			after, ok := cat.Story(tc.id + "_after")
			if !ok {
				t.Fatalf("%s_after がない", tc.id)
			}
			if !strings.Contains(strings.Join(before.Paragraphs, ""), tc.boss) || !strings.Contains(strings.Join(after.Paragraphs, ""), tc.after) {
				t.Errorf("%s の目的・次の章へのつながりがない", tc.id)
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

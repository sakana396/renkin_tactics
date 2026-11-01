package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

func prepareScenes(t *testing.T) *SceneController {
	t.Helper()
	s := newScenes(t, &memoryStore{})
	startScenes(s)
	s.Cancel()
	if s.Scene() != Prepare {
		t.Fatal(s.Scene())
	}
	return s
}
func clickSceneButton(t *testing.T, s *SceneController, label string) {
	t.Helper()
	for _, b := range s.View().Buttons {
		if b.Label == label {
			s.Click(b.Rect.Min.X, b.Rect.Min.Y)
			return
		}
	}
	t.Fatalf("ボタン %s がない", label)
}
func TestPrepare(t *testing.T) {
	t.Run("TC-FR-CMP-03-07 初期選択と生存者の表示", func(t *testing.T) {
		for _, skip := range []bool{false, true} {
			s := newScenes(t, &memoryStore{writeErr: errors.New("保存失敗")})
			startScenes(s)
			if s.Scene() != Story || !strings.Contains(viewStrings(s.View()), "セーブできませんでした") {
				t.Fatal("物語に保存失敗が出ない", s.View())
			}
			if skip {
				s.Cancel()
			} else {
				for s.Scene() == Story {
					s.Advance()
				}
			}
			if s.Scene() != Prepare || s.saveWarning != "" || strings.Contains(viewStrings(s.View()), "セーブできませんでした") {
				t.Fatal("準備に保存警告が残る", s.View())
			}
		}
		s := prepareScenes(t)
		// 主人公が名簿の途中でも、主人公を選び先頭に出撃させる。
		s.campaign.Roster[0], s.campaign.Roster[2] = s.campaign.Roster[2], s.campaign.Roster[0]
		s.campaign.Roster[1].Alive = false
		s.enterPrepare()
		v := s.View()
		for _, tx := range v.Texts {
			if tx.X+textWidth(tx.Text) > MapSize {
				t.Fatalf("左の欄から文字が出る: %+v 幅 %d", tx, textWidth(tx.Text))
			}
		}
		if len(v.Rows) != 4 || s.prepareMember != 2 || !reflect.DeepEqual(s.SelectedMembers(), []int{2, 0, 3, 4}) {
			t.Fatal(v.Rows, s.SelectedMembers(), s.prepareMember)
		}
		if !strings.Contains(viewStrings(v), "出撃 4/6") || len(v.Icons) != 4 || v.Info == nil || v.Buttons[len(v.Buttons)-1].Label != "配置へ" {
			t.Fatal(v)
		}
		for _, tx := range v.Info.Texts {
			if strings.Contains(tx.Text, "成長率") {
				t.Fatal(tx)
			}
		}
		row := v.Rows[0]
		s.Click(row.Rect.Min.X, row.Rect.Min.Y)
		if s.prepareMember != 0 || !s.View().Rows[0].Selected {
			t.Fatal("行を選べない")
		}
	})
	t.Run("TC-FR-CMP-03-08 印の取消しと主人公と枠の上限", func(t *testing.T) {
		s := prepareScenes(t)
		// ch01 の枠は 6 つ（DES-IT06 13 章）。生存者を 7 人にして、7 人目で枠を超える。
		s.campaign.Roster = append(s.campaign.Roster,
			core.Member{Character: "elsa", Alive: true, Level: 2},
			core.Member{Character: "noel", Alive: true, Level: 2})
		s.enterPrepare()
		clickMark := func(n int) { r := s.View().Rows[n].Mark; s.Click(r.Min.X, r.Min.Y) }
		clickMark(0)
		if !strings.Contains(viewStrings(s.View()), "主人公は必ず出撃します") || len(s.SelectedMembers()) != 6 {
			t.Fatal(s.View())
		}
		clickMark(6)
		if !strings.Contains(viewStrings(s.View()), "出撃は 6 人までです") || len(s.SelectedMembers()) != 6 {
			t.Fatal(s.View())
		}
		clickMark(1)
		clickMark(6)
		if !reflect.DeepEqual(s.SelectedMembers(), []int{0, 2, 3, 4, 5, 6}) {
			t.Fatal(s.SelectedMembers())
		}
		clickMark(6)
		if len(s.SelectedMembers()) != 5 {
			t.Fatal(s.SelectedMembers())
		}
	})
	t.Run("TC-FR-CMP-03-09 入れ替えと上限の理由", func(t *testing.T) {
		s := prepareScenes(t)
		clickSceneButton(t, s, "在庫へ: 回復薬")
		if s.campaign.Roster[0].Items[0].Count != 1 || s.campaign.Stock[0].Count != 1 {
			t.Fatal(s.campaign)
		}
		clickSceneButton(t, s, "手持ちへ: 回復薬")
		if s.campaign.Roster[0].Items[0].Count != 2 || len(s.campaign.Stock) != 0 {
			t.Fatal(s.campaign)
		}
		s.campaign.Roster[0].Items[0].Count = 5
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 1}}
		clickSceneButton(t, s, "手持ちへ: 回復薬")
		if !strings.Contains(viewStrings(s.View()), "手持ちは 5 枠までです") || s.campaign.Stock[0].Count != 1 || s.campaign.Roster[0].Items[0].Count != 5 {
			t.Fatal(s.View())
		}
	})
	t.Run("TC-FR-CMP-03-10 出撃の順序と準備へ戻らない", func(t *testing.T) {
		s := prepareScenes(t)
		r := s.View().Rows[1].Mark
		s.Click(r.Min.X, r.Min.Y)
		before := append([]defs.ItemCount(nil), s.campaign.Roster[0].Items...)
		s.Click(500, 500)
		if s.Scene() != Prepare {
			t.Fatal("ボタン外で出撃した")
		}
		seeds := 0
		s.cfg.NextSeed = func() uint64 { seeds++; return 71 }
		clickSceneButton(t, s, "配置へ")
		if s.Scene() != Deploy || seeds != 0 {
			t.Fatal("配置へ進まない、または配置中に種を取得", s.Scene(), seeds)
		}
		clickSceneButton(t, s, "戦闘開始")
		if seeds != 1 {
			t.Fatal("種の取得回数", seeds)
		}
		if s.Scene() != Battle || !reflect.DeepEqual(s.sortie.Members, []int{0, 2, 3, 4}) {
			t.Fatal(s.Scene(), s.sortie)
		}
		s.Cancel()
		s.Choose(0)
		s.Click(0, 0)
		if s.Scene() != Battle || !reflect.DeepEqual(s.campaign.Roster[0].Items, before) {
			t.Fatal("戦闘から準備へ戻った")
		}
	})
	t.Run("TC-FR-CMP-03-11 選んだ仲間の情報とボタンの境界", func(t *testing.T) {
		s := prepareScenes(t)
		s.campaign.Roster[1].Level = 20
		s.campaign.Roster[1].Exp = 0
		row := s.View().Rows[1]
		s.Click(row.Rect.Max.X, row.Rect.Min.Y)
		if s.prepareMember != 0 {
			t.Fatal("右端を含めた")
		}
		s.Click(row.Rect.Min.X, row.Rect.Min.Y)
		s.Cancel()
		if s.Scene() != Prepare {
			t.Fatal("準備を取り消した")
		}
		v := s.View()
		var texts []string
		for _, tx := range v.Info.Texts {
			texts = append(texts, tx.Text)
		}
		var levelLines []string
		for _, tx := range texts {
			if strings.HasPrefix(tx, "Lv ") {
				levelLines = append(levelLines, tx)
			}
		}
		if !reflect.DeepEqual(levelLines, []string{"Lv 20　経験 ―"}) {
			t.Fatalf("レベルと経験は共通の情報欄に1行だけ表示する: %q", levelLines)
		}
		info := strings.Join(texts, "\n")
		for _, want := range []string{"ガルド", "アーマー", "Lv 20　経験 ―", "HP 24/24"} {
			if !strings.Contains(info, want) {
				t.Fatal(want, info)
			}
		}
		// 武器とスキルは「装備・スキル」のページ。Tab で切り替えても、選んでいる仲間と出撃の印は変わらない（DES-UI-28）。
		before := fmt.Sprint(s.prepareMember, s.SelectedMembers())
		s.ToggleInfoPage()
		texts = nil
		for _, tx := range s.View().Info.Texts {
			texts = append(texts, tx.Text)
		}
		info = strings.Join(texts, "\n")
		for _, want := range []string{"ガルド", "鉄の槍", "大盾"} {
			if !strings.Contains(info, want) {
				t.Fatal(want, info)
			}
		}
		if fmt.Sprint(s.prepareMember, s.SelectedMembers()) != before {
			t.Fatal("ページの切り替えで選択が変わった")
		}
		s.ToggleInfoPage()
		s.Click(row.Mark.Min.X, row.Mark.Min.Y)
		clickSceneButton(t, s, "在庫へ: 回復薬")
		if len(s.campaign.Roster[1].Items) != 0 || s.campaign.Roster[0].Items[0].Count != 2 {
			t.Fatal("選んでいない仲間を変更した")
		}
		clickSceneButton(t, s, "手持ちへ: 回復薬")
		if s.campaign.Roster[1].Items[0].Count != 1 {
			t.Fatal("控えに渡せない")
		}
		if v.Info.Bottom > ScreenHeight {
			t.Fatal("情報欄の下端", v.Info.Bottom)
		}
	})

}
func TestChapterResultView(t *testing.T) {
	t.Run("TC-FR-CMP-10-02 成長と離脱と戦死と報酬", func(t *testing.T) {
		for _, dead := range []bool{false, true} {
			s := prepareScenes(t)
			s.scene = Result
			s.storyChapter = 0
			s.result = core.ChapterResult{Members: []core.MemberResult{{Member: 0, LevelBefore: 1, LevelAfter: 2, ExpBefore: 80, ExpAfter: 10, Gains: tactics.Stats{HP: 1, Skl: 1, Def: 1}}, {Member: 1, LevelBefore: 2, LevelAfter: 2, Fallen: true, Dead: dead}}, Reward: []defs.ItemCount{{Item: "potion", Count: 2}}}
			v := s.View()
			text := viewStrings(v)
			for _, want := range []string{"第 1 章　国境の砦　勝利", "Lv 1 → 2", "経験 80 → 10", "HP+1 技+1 守備+1", "回復薬 ×2 を在庫に入れた"} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if strings.Contains(text, "力+0") || strings.Contains(text, "速さ+0") || strings.Contains(text, "上がった能力なし") {
				t.Fatal(text)
			}
			for _, tx := range v.Texts {
				if tx.Y == 212 {
					t.Fatal("能力が上がらなかった仲間に成長の行がある", tx)
				}
			}
			fallen := "離脱（次の章で戻る）"
			if dead {
				fallen = "戦死"
			}
			found := false
			for _, tx := range v.Texts {
				if strings.Contains(tx.Text, fallen) {
					found = true
					if dead && tx.Style != StyleBad {
						t.Fatal(tx)
					}
				}
			}
			if !found {
				t.Fatal(text)
			}
			// 最大の 6 人でも、報酬・警告より上に全員の結果を収める。
			for len(s.result.Members) < 6 {
				s.result.Members = append(s.result.Members, s.result.Members[0])
			}
			for _, tx := range s.View().Texts {
				height := 16
				if tx.Big {
					height = 32
				}
				if tx.Y+height > ScreenHeight || tx.X+textWidth(tx.Text)*height/16 > ScreenWidth {
					t.Fatalf("結果が画面の外に出る: %+v", tx)
				}
			}
			s.result.Reward = nil
			if strings.Contains(viewStrings(s.View()), "在庫に入れた") {
				t.Fatal("空の報酬")
			}
		}
	})
}

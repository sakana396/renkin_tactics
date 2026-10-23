package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
)

func TestStartChapterScenes(t *testing.T) {
	t.Run("TC-FR-CMP-01-06 指定章の物語と加入と保存", func(t *testing.T) {
		for _, chapter := range []string{"", "c1", "c2", "c3"} {
			for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
				cat := sceneCatalog(true, true)
				store := &memoryStore{}
				s := NewSceneController(Config{Catalog: cat, Store: store, StartChapter: chapter})
				s.Choose(0)
				if s.Scene() != RuleSelect || store.writes != 0 {
					t.Fatal(s.Scene(), store.writes)
				}
				s.Choose(int(rule))
				index := 0
				if chapter == "c2" {
					index = 1
				}
				if chapter == "c3" {
					index = 2
				}
				if s.Scene() != Story || s.storyChapter != index || s.storyAfter || s.paragraph != 0 || s.campaign.Chapter != index || s.campaign.Rule != rule || store.writes != 1 {
					t.Fatal(s.Scene(), s.campaign, store.writes)
				}
				if !strings.Contains(viewStrings(s.View()), "前の物語の最初。") {
					t.Fatal(s.View())
				}
				loaded, err := core.LoadCampaign(cat, bytes.NewReader(store.data))
				if err != nil || loaded.Chapter != index || loaded.Rule != rule || !bytes.Equal(s.chapterStart, store.data) {
					t.Fatal(loaded, err)
				}
				// 保存では空の手持ちを [] に正規化するため、保存形式で往復を比較する。
				var roundTrip bytes.Buffer
				if err := loaded.Save(&roundTrip); err != nil || !bytes.Equal(roundTrip.Bytes(), store.data) {
					t.Fatal("指定章の保存が往復で変わった", err)
				}
				s.Cancel()
				if chapter == "c2" {
					for _, name := range []string{"加入者が仲間に加わった", "加入者2が仲間に加わった"} {
						if s.Scene() != Joined || !strings.Contains(viewStrings(s.View()), name) {
							t.Fatal(s.Scene(), s.View())
						}
						s.Advance()
					}
				}
				if s.Scene() != Prepare || store.writes != 1 {
					t.Fatal(s.Scene(), store.writes)
				}
				// 続きからでは、起動指定より保存された章を優先する。
				resumed := NewSceneController(Config{Catalog: cat, Store: store, StartChapter: "c3"})
				resumed.Choose(1)
				if resumed.Scene() != Story || resumed.campaign.Chapter != index || store.writes != 1 {
					t.Fatal(resumed.Scene(), resumed.campaign, store.writes)
				}
			}
		}
	})
}

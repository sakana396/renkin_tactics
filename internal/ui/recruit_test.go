package ui

import (
	"fmt"
	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"image"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 本体の名簿を使い加入と準備の入力を確かめる（DES-CMP-10、DES-UI-13）。
func recruitScenes(t *testing.T, id string, rule core.Rule) *SceneController {
	t.Helper()
	s := newScenes(t, &memoryStore{})
	c, err := core.NewCampaignAt(s.cfg.Catalog, rule, id)
	if err != nil {
		t.Fatal(err)
	}
	s.campaign = c
	s.storyChapter = c.Chapter
	s.enterPrepare()
	return s
}
func TestRecruitPrepare(t *testing.T) {
	t.Run("TC-FR-CMP-03-13 初期の印と控え・空いた枠・出撃者", func(t *testing.T) {
		for _, id := range []string{"ch04", "ch08", "ch09"} {
			for _, dead := range []bool{false, true} {
				s := recruitScenes(t, id, core.Permadeath)
				if dead {
					s.campaign.Roster[3].Alive = false
					s.campaign.Roster[4].Alive = false
					s.enterPrepare()
				}
				var want []int
				for i, m := range s.campaign.Roster {
					if m.Alive && len(want) < s.prepareSlots() {
						want = append(want, i)
					}
				}
				if !reflect.DeepEqual(s.SelectedMembers(), want) {
					t.Fatal(id, dead, s.SelectedMembers(), want)
				}
				if !dead && (s.deployment[6] || s.deployment[7]) {
					t.Fatal("加入者が初期出撃")
				}
				if dead && !s.deployment[6] {
					t.Fatal("ティオが空いた枠に入らない")
				}
				if dead && id == "ch09" && !s.deployment[7] {
					t.Fatal("ハンナが空いた枠に入らない")
				}
				s.toggleDeployment(0)
				if !s.deployment[0] || !strings.Contains(s.message, "主人公") {
					t.Fatal(s.message)
				}
				if !dead {
					s.toggleDeployment(6)
					if s.deployment[6] || !strings.Contains(s.message, "まで") {
						t.Fatal(s.message)
					}
				}
				clickSceneButton(t, s, "配置へ")
				if s.Scene() != Deploy || len(s.deployMembers) != len(want) {
					t.Fatal(s.Scene(), s.deployMembers)
				}
				s.Choose(0)
				if s.Scene() != Battle {
					t.Fatal(s.Scene())
				}
				var got []string
				for _, u := range s.sortie.Battle.Units() {
					if u.Side == tactics.Player {
						got = append(got, u.Name)
					}
				}
				var names []string
				for _, i := range want {
					ch, _ := s.cfg.Catalog.Character(s.campaign.Roster[i].Character)
					names = append(names, ch.Name)
				}
				if !reflect.DeepEqual(got, names) {
					t.Fatal(got, names)
				}
			}
		}
	})
	t.Run("TC-IF-01-38 長い案内は盤面の幅で折り返して配置へと重ならない", func(t *testing.T) {
		s := recruitScenes(t, "ch08", core.Withdraw)
		// 8 人・品 2 種類で案内の位置を下げ、従来の全画面幅では盤面をはみ出す長さを使う。
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 1}, {Item: "potion", Count: 2}}
		s.message = "案内\n" + strings.Repeat("案内", 20)
		v := s.View()
		var lines []string
		for _, tx := range v.Texts {
			if tx.Style != StyleBad {
				continue
			}
			lines = append(lines, tx.Text)
			r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+16)
			if !r.In(image.Rect(48, 0, MapSize-8, ScreenHeight)) {
				t.Fatal("案内が盤面の幅を超えた", r)
			}
			for _, b := range v.Buttons {
				if r.Overlaps(b.Rect) {
					t.Fatal("案内がボタンと重複", r, b)
				}
			}
		}
		if len(lines) < 2 || strings.Join(lines, "") != strings.ReplaceAll(s.message, "\n", "") {
			t.Fatal("案内の折り返し・内容", lines)
		}
	})
	t.Run("TC-IF-01-38 6・7・8人と品0・1・2種類の配置とクリック", func(t *testing.T) {
		// 検証済みの目録に試験用の薬を足し、異なる2種類の品を表示する。
		source := fstest.MapFS{}
		if err := fs.WalkDir(data.Base(), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := fs.ReadFile(data.Base(), path)
			source[path] = &fstest.MapFile{Data: b}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		source["items.toml"].Data = append(source["items.toml"].Data, []byte("\n[[items]]\nid = \"test_potion\"\nname = \"試験の薬\"\nheal = 5\n")...)
		cat, err := defload.Load(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"ch03", "ch04", "ch08"} {
			for stock := 0; stock <= 2; stock++ {
				for hand := 0; hand <= 2; hand++ {
					t.Run(fmt.Sprintf("%s/在庫%d/手持ち%d", id, stock, hand), func(t *testing.T) {
						s := recruitScenes(t, id, core.Withdraw)
						s.cfg.Catalog = cat
						s.campaign, err = core.NewCampaignAt(cat, core.Withdraw, id)
						if err != nil {
							t.Fatal(err)
						}
						s.enterPrepare()
						s.campaign.Stock = nil
						s.campaign.StockWeapons = nil
						s.campaign.Roster[0].Items = nil
						for i := 0; i < stock; i++ {
							s.campaign.Stock = append(s.campaign.Stock, defs.ItemCount{Item: []string{"potion", "test_potion"}[i], Count: 1})
						}
						for i := 0; i < hand; i++ {
							s.campaign.Roster[0].Items = append(s.campaign.Roster[0].Items, defs.ItemCount{Item: []string{"potion", "test_potion"}[i], Count: 1})
						}
						s.message = "主人公は必ず出撃します"
						v := s.View()
						n := len(v.Rows)
						var rects []image.Rectangle
						for i, r := range v.Rows {
							if r.Rect != image.Rect(56, 72+i*24, 504, 96+i*24) || r.Mark != image.Rect(16, 72+i*24, 52, 96+i*24) || v.Icons[i].Y != 72+i*24 {
								t.Fatal(r, v.Icons[i])
							}
							rects = append(rects, r.Rect, r.Mark)
							s.Click(r.Rect.Min.X, r.Rect.Min.Y)
							if s.prepareMember != r.Member {
								t.Fatal("行の選択", i)
							}
							was := s.deployment[r.Member]
							s.Click(r.Mark.Min.X, r.Mark.Min.Y)
							if r.Member == 0 {
								if !s.deployment[0] {
									t.Fatal("主人公")
								}
							} else if was && s.deployment[r.Member] {
								t.Fatal("印の取消し", i)
							} else if !was && !s.deployment[r.Member] {
								t.Fatal("印の選択", i)
							}
						}
						// 元の品数の画面で矩形を確認する。
						for _, b := range v.Buttons {
							rects = append(rects, b.Rect)
						}
						// #122 で行を 24 px、品のボタンを 22 px ごとに詰め、手持ちの列に武器（鉄の剣）も並べた。
						inventoryY := 72 + n*24 + 12
						messageY := inventoryY + 24
						if rows := max(stock, hand+len(s.campaign.Roster[0].Weapons)); rows > 0 {
							messageY += rows*22 + 4
						}
						for _, tx := range v.Texts {
							if strings.HasPrefix(tx.Text, "在庫:") {
								if tx.Y != inventoryY {
									t.Fatal(tx)
								}
								rects = append(rects, image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+16))
							}
							if tx.Text == "主人公は必ず出撃します" {
								if tx.Y != messageY {
									t.Fatal(tx, messageY)
								}
								rects = append(rects, image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+16))
							}
						}
						for i, r := range rects {
							if !r.In(image.Rect(0, 0, 512, 576)) {
								t.Fatal("窓外", r)
							}
							for _, other := range rects[i+1:] {
								if r.Overlaps(other) {
									t.Fatal("重複", r, other)
								}
							}
						}
					})
				}
			}
		}
	})
}
func TestRecruitJoinedScenes(t *testing.T) {
	t.Run("TC-FR-CMP-08-05 加入の表示・保存からの続き・敗北からのやり直し", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			for _, tc := range []struct {
				id, name string
				count    int
			}{{"ch04", "ティオ", 7}, {"ch08", "ハンナ", 8}} {
				s := recruitScenes(t, tc.id, rule)
				s.saveChapter()
				s.enterTitle("")
				s.Choose(1)
				if s.Scene() != Story {
					t.Fatal(s.Scene())
				}
				s.Cancel()
				if s.Scene() != Joined || !strings.Contains(viewStrings(s.View()), tc.name+"が仲間に加わった") || len(s.View().Icons) != 1 {
					t.Fatal(s.Scene(), s.View())
				}
				s.Advance()
				if s.Scene() != Prepare || len(s.campaign.Roster) != tc.count {
					t.Fatal(s.Scene())
				}
				// 敗北からの実際のやり直しの入口で、保存された加入者を二重にしない。
				s.scene = GameOver
				s.Choose(0)
				if s.Scene() != Prepare || len(s.campaign.Roster) != tc.count {
					t.Fatal(s.Scene(), s.campaign)
				}
			}
		}
	})
}

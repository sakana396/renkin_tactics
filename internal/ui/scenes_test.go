package ui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// memoryStore はファイルを使わず保存の呼び出しと失敗を確かめる（DES-UI-09）。
type memoryStore struct {
	data                         []byte
	exists                       bool
	readErr, writeErr, existsErr error
	writes                       int
}

func (m *memoryStore) Exists() (bool, error) { return m.exists, m.existsErr }
func (m *memoryStore) Read(fn func(io.Reader) error) error {
	if m.readErr != nil {
		return m.readErr
	}
	return fn(bytes.NewReader(m.data))
}
func (m *memoryStore) Write(fn func(io.Writer) error) error {
	m.writes++
	if m.writeErr != nil {
		return m.writeErr
	}
	var b bytes.Buffer
	if err := fn(&b); err != nil {
		return err
	}
	m.data = append([]byte(nil), b.Bytes()...)
	m.exists = true
	return nil
}
func newScenes(t *testing.T, m *memoryStore) *SceneController {
	t.Helper()
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	return NewSceneController(Config{Catalog: cat, Store: m, NextSeed: func() uint64 { return 7 }})
}

func TestTitleScenes(t *testing.T) {
	t.Run("TC-FR-SYS-02-01 未保存のタイトルと終了", func(t *testing.T) {
		s := newScenes(t, &memoryStore{})
		if s.Scene() != Title || s.View().Buttons[1].Enabled {
			t.Fatal(s.View())
		}
		s.Choose(1)
		if s.Scene() != Title {
			t.Fatal("続きからを受け付けた")
		}
		s.Choose(2)
		if !s.Quit() {
			t.Fatal("終了しない")
		}
	})
	t.Run("TC-FR-CMP-04-03 離脱とロストの選択と自動保存", func(t *testing.T) {
		for _, rule := range []core.Rule{core.Withdraw, core.Permadeath} {
			m := &memoryStore{}
			s := newScenes(t, m)
			s.Choose(0)
			if s.Scene() != RuleSelect {
				t.Fatal(s.Scene())
			}
			s.Cancel()
			if s.Scene() != Title {
				t.Fatal(s.Scene())
			}
			s.Choose(0)
			s.Choose(int(rule))
			if s.Scene() != Story || s.Campaign().Rule != rule || m.writes != 1 || len(s.chapterStart) == 0 {
				t.Fatal("開始・保存が不正")
			}
			c, err := core.LoadCampaign(s.cfg.Catalog, bytes.NewReader(m.data))
			if err != nil || c.Rule != rule || c.Chapter != 0 {
				t.Fatal(c, err)
			}
		}
	})
	t.Run("TC-FR-SYS-02-02 読めないセーブも上書き確認する", func(t *testing.T) {
		for i, data := range []string{`{"version":2}`, `{"version":1,"rule":"bad"}`} {
			m := &memoryStore{exists: true, data: []byte(data)}
			s := newScenes(t, m)
			reason := []error{core.ErrSaveVersion, core.ErrInvalidSave}[i]
			if !strings.Contains(viewStrings(s.View()), "セーブデータを読めません: "+reason.Error()) {
				t.Fatal(s.View())
			}
			if s.View().Buttons[1].Enabled || s.message == "" {
				t.Fatal(s.View())
			}
			s.Choose(0)
			if !s.confirmOverwrite || s.Scene() != Title {
				t.Fatal("確認なし")
			}
			s.Choose(1)
			if s.confirmOverwrite || m.writes != 0 {
				t.Fatal("いいえで上書き")
			}
			s.Choose(0)
			s.Choose(0)
			if s.Scene() != RuleSelect || m.writes != 0 {
				t.Fatal("ルール前に保存")
			}
		}
	})
	t.Run("TC-FR-SYS-02-03 続きからと章表示", func(t *testing.T) {
		m := &memoryStore{}
		s := newScenes(t, m)
		s.Choose(0)
		s.Choose(0)
		s = newScenes(t, m)
		if !s.View().Buttons[1].Enabled || s.saveSummary() != "第 1 章　国境の砦（離脱）" {
			t.Fatal(s.View())
		}
		s.Choose(1)
		if s.Scene() != Story || m.writes != 1 || len(s.chapterStart) == 0 {
			t.Fatal("再開が不正")
		}
	})
	t.Run("TC-FR-SYS-02-04 存在確認と読み込みの失敗", func(t *testing.T) {
		for _, m := range []*memoryStore{{existsErr: errors.New("存在確認失敗")}, {exists: true, readErr: errors.New("読み込み失敗")}} {
			s := newScenes(t, m)
			if s.message == "" || s.View().Buttons[1].Enabled {
				t.Fatal(s.View())
			}
		}
	})
}

// sceneCatalog は 3 章の遷移を短い実戦で確かめる目録。主人公の攻撃は必ず当たり、ボスを 1 撃で倒す。
func sceneCatalog(before, after bool) *defs.Catalog {
	chapters := []defs.ChapterDef{{ID: "c1", Name: "砦", Map: "m"}, {ID: "c2", Name: "街道", Map: "m", Joins: []string{"join", "join2"}}, {ID: "c3", Name: "王都", Map: "m"}}
	for i := range chapters {
		if before {
			chapters[i].Before = "before"
		}
		if after {
			chapters[i].After = "after"
		}
	}
	return defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend"}, Stock: []defs.ItemCount{{Item: "p", Count: 1}}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 20, Str: 20, Skl: 100, Def: 10, Mov: 2}}, {ID: "soldier", Name: "兵", WeaponType: "sword", Base: defs.Stats{HP: 1, Str: 20, Skl: 100, Mov: 1}}},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", Hit: 100, MinRange: 1, MaxRange: 1}},
		[]defs.ItemDef{{ID: "p", Name: "薬", Heal: 10}}, nil,
		[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "lord", Weapon: "s", Lord: true, Level: 1}, {ID: "friend", Name: "仲間", Class: "lord", Weapon: "s", Level: 2}, {ID: "join", Name: "加入者", Class: "lord", Weapon: "s", Level: 4}, {ID: "join2", Name: "加入者2", Class: "lord", Weapon: "s", Level: 5}},
		[]defs.MapDef{{ID: "m", Rows: []string{"...", "..."}, Slots: []defs.Slot{{X: 0, Y: 0}, {X: 0, Y: 1}}, Enemies: []defs.EnemyPlacement{{Class: "soldier", Weapon: "s", Boss: true, AI: "hold", X: 1, Y: 0}}}}, chapters,
		[]defs.StoryDef{{ID: "before", Paragraphs: []string{"前の物語の最初。", "前の物語の最後。"}}, {ID: "after", Paragraphs: []string{"あとの物語。"}}})
}
func fixtureScenes(m *memoryStore, before, after bool) *SceneController {
	return NewSceneController(Config{Catalog: sceneCatalog(before, after), Store: m, NextSeed: func() uint64 { return 7 }})
}
func startScenes(s *SceneController) { s.Choose(0); s.Choose(0) }
func winScene(t *testing.T, s *SceneController) {
	t.Helper()
	if s.Scene() != Prepare {
		t.Fatal(s.Scene())
	}
	clickSceneButton(t, s, "配置へ")
	clickSceneButton(t, s, "戦闘開始")
	c := s.BattleController()
	playOut(c)
	c.Click(tactics.Pos{X: 0, Y: 0})
	c.Click(tactics.Pos{X: 0, Y: 0})
	c.ChooseAction(tactics.Attack)
	c.Click(tactics.Pos{X: 1, Y: 0})
	c.Click(tactics.Pos{X: 1, Y: 0})
	if c.Battle().Result != tactics.Won || c.State() != Playing {
		t.Fatal("勝利を再生していない")
	}
}
func viewStrings(v SceneView) string {
	var lines []string
	for _, tx := range v.Texts {
		lines = append(lines, tx.Text)
	}
	return strings.Join(lines, "\n")
}

func TestStoryScenes(t *testing.T) {
	t.Run("TC-FR-CMP-07-02 物語は 1 段落ずつ置き換え、最後で準備へ", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, true, true)
		startScenes(s)
		if s.Scene() != Story || s.paragraph != 0 {
			t.Fatal(s.Scene())
		}
		v := viewStrings(s.View())
		if !strings.Contains(v, "第 1 章　砦") || !strings.Contains(v, "前の物語の最初。") || !strings.Contains(v, "1/2") || strings.Contains(v, "前の物語の最後。") {
			t.Fatal(v)
		}
		s.Advance()
		v = viewStrings(s.View())
		if !strings.Contains(v, "前の物語の最後。") || !strings.Contains(v, "2/2") || strings.Contains(v, "前の物語の最初。") {
			t.Fatal(v)
		}
		s.Advance()
		if s.Scene() != Prepare {
			t.Fatal(s.Scene())
		}
	})
	t.Run("TC-FR-CMP-07-03 空の物語と残りを飛ばす操作", func(t *testing.T) {
		for _, before := range []bool{true, false} {
			s := fixtureScenes(&memoryStore{}, before, false)
			startScenes(s)
			if before {
				s.Cancel()
			}
			if s.Scene() != Prepare {
				t.Fatal(s.Scene())
			}
			winScene(t, s)
			playOut(s.battle)
			s.Advance()
			s.Advance()
			if s.Scene() == Story {
				s.Cancel()
			}
			if s.Scene() != Joined {
				t.Fatal(s.Scene())
			}
		}
	})
	t.Run("TC-FR-CMP-07-04 200 文字を余白 48 px の内側で折り返す", func(t *testing.T) {
		source := strings.Repeat("物", 200)
		lines := wrapSceneText(source)
		if strings.Join(lines, "") != source || len(lines) < 2 {
			t.Fatal(lines)
		}
		for _, line := range lines {
			if textWidth(line) > ScreenWidth-96 {
				t.Fatal(line)
			}
		}
	})
	t.Run("TC-FR-CMP-08-03 章の前の物語のあと複数の加入者を表示する", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, true, true)
		startScenes(s)
		s.Cancel()
		winScene(t, s)
		playOut(s.battle)
		s.Advance()
		s.Advance()
		s.Advance()
		if s.Scene() != Story || s.storyChapter != 1 {
			t.Fatal(s.Scene())
		}
		s.Cancel()
		for _, want := range []string{"加入者が仲間に加わった", "加入者2が仲間に加わった"} {
			v := s.View()
			text := viewStrings(v)
			if s.Scene() != Joined || !strings.Contains(text, want) || !strings.Contains(text, "ロード　Lv") || !strings.Contains(text, "HP 20　力 20　技 100　速さ 0　守備 10　移動 2") || len(v.Icons) != 1 {
				t.Fatal(text)
			}
			s.Advance()
		}
		if s.Scene() != Prepare || len(s.campaign.Roster) != 4 {
			t.Fatal(s.Scene())
		}
	})
}

func TestBattleScenes(t *testing.T) {
	t.Run("TC-IF-01-13 Battle の Over で Cancel を受けても場面と戦闘が変わらない", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		winScene(t, s)
		playOut(s.battle)
		old := s.battle.Battle()
		before := old.Units()
		s.Cancel()
		if s.Scene() != Battle || s.battle.State() != Over || s.battle.Battle() != old || !reflect.DeepEqual(before, old.Units()) || old.Result != tactics.Won {
			t.Fatal("Cancel が場面と戦闘を変えた")
		}
	})
	t.Run("TC-FR-CMP-02-02 章の結果反映の失敗を戦闘の案内欄に出す", func(t *testing.T) {
		m := &memoryStore{}
		s := fixtureScenes(m, false, false)
		startScenes(s)
		winScene(t, s)
		playOut(s.battle)
		s.campaign.Chapter++ // 出撃と章を不一致にして FinishChapter を失敗させる。
		s.Advance()
		if s.Scene() != Battle || s.message != core.ErrInvalidSortie.Error() || !strings.Contains(strings.Join(s.BattleGuide(), "\n"), core.ErrInvalidSortie.Error()) || len(s.BattleGuide()) > GuideHeight/lineHeight || m.writes != 1 {
			t.Fatal("結果反映の誤りが戦闘の案内欄に出ない", s.BattleGuide())
		}
	})

	t.Run("TC-FR-CMP-02-02 勝利の帯のクリックでだけ章を進めて保存し最後の物語からクリアへ", func(t *testing.T) {
		m := &memoryStore{}
		s := fixtureScenes(m, true, true)
		startScenes(s)
		s.Cancel()
		for chapter := 0; chapter < 3; chapter++ {
			if chapter > 0 {
				s.Cancel()
				if s.Scene() == Joined {
					s.Advance()
					s.Advance()
				}
			}
			winScene(t, s)
			s.Advance()
			if s.Scene() != Battle || m.writes != chapter+1 || s.campaign.Chapter != chapter {
				t.Fatal("再生中に章を進めた")
			}
			playOut(s.battle)
			s.battle.Escape()
			if s.battle.State() != Over {
				t.Fatal("Esc を受け付けた")
			}
			if m.writes != chapter+1 || s.campaign.Chapter != chapter {
				t.Fatal("帯をクリックする前に保存")
			}
			s.Click(0, 0)
			s.Click(MapSize, MapSize/2)
			if s.Scene() != Battle || m.writes != chapter+1 {
				t.Fatal("帯の外で保存")
			}
			s.Click(MapSize/2, MapSize/2)
			if s.Scene() != Result || m.writes != chapter+2 || s.campaign.Chapter != chapter+1 || s.result.Chapter != fmt.Sprintf("c%d", chapter+1) {
				t.Fatal(s.Scene(), m.writes, s.campaign.Chapter)
			}
			stored, err := core.LoadCampaign(s.cfg.Catalog, bytes.NewReader(m.data))
			if err != nil || stored.Chapter != chapter+1 || stored.Cleared != (chapter == 2) {
				t.Fatal(stored, err)
			}
			s.Advance()
			if s.Scene() != Story || !s.storyAfter || s.storyChapter != chapter {
				t.Fatal("終えた章の物語でない")
			}
			s.Advance()
		}
		if s.Scene() != Clear || !strings.Contains(viewStrings(s.View()), "主人公　Lv 1") {
			t.Fatal(s.View())
		}
		s.Advance()
		if s.Scene() != Title || s.saveSummary() != "クリア済み" {
			t.Fatal(s.View())
		}
		s.Choose(1)
		if s.Scene() != Clear {
			t.Fatal("クリア済みの再開")
		}
	})
	t.Run("TC-FR-CMP-09-07 敗北では保存せずメモリから準備へ戻す", func(t *testing.T) {
		for _, rule := range []int{0, 1} {
			m := &memoryStore{writeErr: errors.New("保存失敗")}
			s := fixtureScenes(m, true, true)
			s.Choose(0)
			s.Choose(rule)
			if !strings.Contains(viewStrings(s.View()), "セーブできませんでした: 保存失敗") {
				t.Fatal(s.View())
			}
			s.Cancel()
			baseline := append([]byte(nil), s.chapterStart...)
			if err := s.campaign.TakeItem(0, "p"); err != nil {
				t.Fatal(err)
			}
			s.campaign.Roster[0].Stats.HP = 1
			s.campaign.Roster[0].Stats.Def = 0
			clickSceneButton(t, s, "配置へ")
			clickSceneButton(t, s, "戦闘開始")
			playOut(s.battle)
			s.battle.EndPhase()
			s.battle.EndPhase()
			playOut(s.battle)
			if s.battle.Battle().Result != tactics.Lost {
				t.Fatal("負けていない")
			}
			s.Advance()
			if s.Scene() != GameOver || m.writes != 1 {
				t.Fatal(s.Scene(), m.writes)
			}
			m.readErr = errors.New("ファイルを読んではいけない")
			s.Choose(0)
			var b bytes.Buffer
			if err := s.campaign.Save(&b); err != nil {
				t.Fatal(err)
			}
			if s.Scene() != Prepare || !bytes.Equal(b.Bytes(), baseline) || m.writes != 1 {
				t.Fatal("章の始めへ戻らない")
			}
			clickSceneButton(t, s, "配置へ")
			clickSceneButton(t, s, "戦闘開始")
			if s.Scene() != Battle || !s.sortie.Battle.Units()[0].Alive || s.sortie.Battle.Units()[0].HP != 20 {
				t.Fatal("再出撃できない")
			}
		}
	})
	t.Run("TC-FR-CMP-09-08 メモリが読めなければ理由を表示してタイトルへ", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		s.scene = GameOver
		s.chapterStart = []byte("壊れた JSON")
		s.Choose(0)
		if s.Scene() != Title || !strings.Contains(viewStrings(s.View()), "この章をやり直せません") {
			t.Fatal(s.View())
		}
		s.scene = GameOver
		s.Choose(1)
		if s.Scene() != Title {
			t.Fatal(s.Scene())
		}
	})
	t.Run("TC-FR-CMP-02-03 クリア画面は戦死者を分けて表示する", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		startScenes(s)
		s.campaign.Rule = core.Permadeath
		s.campaign.Roster[1].Alive = false
		s.scene = Clear
		v := viewStrings(s.View())
		if !strings.Contains(v, "主人公　Lv 1") || !strings.Contains(v, "戦死: 仲間") || strings.Contains(v, "仲間　Lv") {
			t.Fatal(v)
		}
	})
	t.Run("TC-FR-SYS-01-07 勝利後の保存失敗も結果へ進み次章のメモリを保持", func(t *testing.T) {
		m := &memoryStore{}
		s := fixtureScenes(m, true, true)
		startScenes(s)
		s.Cancel()
		old := append([]byte(nil), m.data...)
		m.writeErr = errors.New("書込失敗")
		winScene(t, s)
		playOut(s.battle)
		s.Advance()
		if s.Scene() != Result || !strings.Contains(viewStrings(s.View()), "セーブできませんでした: 書込失敗") || !bytes.Equal(old, m.data) {
			t.Fatal(s.View())
		}
		c, err := core.LoadCampaign(s.cfg.Catalog, bytes.NewReader(s.chapterStart))
		if err != nil || c.Chapter != 1 || len(c.Roster) != 4 {
			t.Fatal(c, err)
		}
		for _, want := range []Scene{Story, Story, Story, Joined, Joined, Prepare} {
			s.Advance()
			if s.Scene() != want || s.saveWarning != "" || strings.Contains(viewStrings(s.View()), "セーブできませんでした") {
				t.Fatalf("結果のあとに保存警告が残った: 場面 %v、表示 %s", s.Scene(), viewStrings(s.View()))
			}
		}
	})
	t.Run("TC-IF-01-35 ボタンの内外と無効な選択", func(t *testing.T) {
		s := fixtureScenes(&memoryStore{}, false, false)
		s.Click(0, 0)
		s.Choose(-1)
		s.Choose(99)
		if s.Scene() != Title {
			t.Fatal(s.Scene())
		}
		b := s.View().Buttons[0]
		s.Click(b.Rect.Max.X, b.Rect.Min.Y)
		if s.Scene() != Title {
			t.Fatal("右端を受け付けた")
		}
		s.Click(b.Rect.Min.X, b.Rect.Min.Y)
		if s.Scene() != RuleSelect {
			t.Fatal(s.Scene())
		}
		s.Choose(-1)
		s.Choose(2)
		if s.Scene() != RuleSelect {
			t.Fatal(s.Scene())
		}
	})
}

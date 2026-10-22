package ui

import (
	"bytes"
	"fmt"
	"io"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// SaveStore は保存先の実装を画面から隠す口（DES-UI-09）。
type SaveStore interface {
	Exists() (bool, error)
	Read(func(io.Reader) error) error
	Write(func(io.Writer) error) error
}

// Config は起動時に渡す目録・保存先・戦闘の種（DES-UI-09、DES-CMD-02）。
type Config struct {
	Catalog  *defs.Catalog
	Store    SaveStore
	NextSeed func() uint64
}

// Scene はキャンペーンの場面（DES-UI-09）。
type Scene int

const (
	Title Scene = iota
	RuleSelect
	Story
	Joined
	Prepare
	Battle
	Result
	GameOver
	Clear
)

// SceneController は描画や OS 入力から独立した場面の進行（DES-UI-09）。
type SceneController struct {
	cfg              Config
	scene            Scene
	campaign         *core.Campaign
	loaded           *core.Campaign
	sortie           *core.Sortie
	battle           *Controller
	result           core.ChapterResult
	chapterStart     []byte
	saveExists       bool
	confirmOverwrite bool
	quit             bool
	message          string
	saveWarning      string
	storyChapter     int
	storyAfter       bool
	paragraphs       []string
	paragraph        int
	joined           int
}

// NewSceneController はタイトルでセーブを調べる（DES-UI-10）。
func NewSceneController(cfg Config) *SceneController {
	s := &SceneController{cfg: cfg}
	s.enterTitle("")
	return s
}

// Scene・Campaign・BattleController・ChapterResult は各場面の連携用の参照（DES-UI-09）。
func (s *SceneController) Scene() Scene                      { return s.scene }
func (s *SceneController) Campaign() *core.Campaign          { return s.campaign }
func (s *SceneController) BattleController() *Controller     { return s.battle }
func (s *SceneController) ChapterResult() core.ChapterResult { return s.result }
func (s *SceneController) Quit() bool                        { return s.quit }

// enterTitle は戻るたびにセーブを読み直し、読めない理由も残す（DES-UI-10、15）。
func (s *SceneController) enterTitle(reason string) {
	s.scene = Title
	s.confirmOverwrite = false
	s.loaded = nil
	s.message = reason
	s.saveWarning = ""
	s.campaign = nil
	s.sortie = nil
	s.battle = nil
	exists, err := s.cfg.Store.Exists()
	s.saveExists = exists || err != nil
	if err == nil && exists {
		err = s.cfg.Store.Read(func(r io.Reader) error {
			c, e := core.LoadCampaign(s.cfg.Catalog, r)
			if e == nil {
				s.loaded = c
			}
			return e
		})
	}
	if err != nil {
		s.loaded = nil
		s.message = fmt.Sprintf("セーブデータを読めません: %v", err)
		if reason != "" {
			s.message = reason + "\n" + s.message
		}
	}
}

// rememberChapter はファイルへの保存と独立して章の始めを保持する（DES-CMP-07）。
func (s *SceneController) rememberChapter() error {
	var b bytes.Buffer
	if err := s.campaign.Save(&b); err != nil {
		return err
	}
	s.chapterStart = append([]byte(nil), b.Bytes()...)
	return nil
}

// saveChapter は章の始めを保持してから自動保存し、保存失敗でも続行する（DES-SAV-02）。
func (s *SceneController) saveChapter() {
	s.saveWarning = ""
	if err := s.rememberChapter(); err != nil {
		s.saveWarning = fmt.Sprintf("セーブできませんでした: %v", err)
		return
	}
	if err := s.cfg.Store.Write(s.campaign.Save); err != nil {
		s.saveWarning = fmt.Sprintf("セーブできませんでした: %v", err)
	}
}

// Choose は場面のボタンを選ぶ（DES-UI-10、11、15）。
func (s *SceneController) Choose(i int) {
	switch s.scene {
	case Title:
		if s.confirmOverwrite {
			switch i {
			case 0:
				s.confirmOverwrite = false
				s.message = ""
				s.scene = RuleSelect
			case 1:
				s.confirmOverwrite = false
			}
			return
		}
		switch i {
		case 0:
			if s.saveExists {
				s.confirmOverwrite = true
			} else {
				s.message = ""
				s.scene = RuleSelect
			}
		case 1:
			if s.loaded != nil {
				s.campaign = s.loaded
				s.message = ""
				if err := s.rememberChapter(); err != nil {
					s.enterTitle(err.Error())
					return
				}
				if s.campaign.Cleared {
					s.scene = Clear
				} else {
					s.beforeStory()
				}
			}
		case 2:
			s.quit = true
		}
	case RuleSelect:
		if i < 0 || i > 1 {
			return
		}
		c, err := core.NewCampaign(s.cfg.Catalog, core.Rule(i))
		if err != nil {
			s.message = err.Error()
			return
		}
		s.campaign = c
		s.saveChapter()
		s.beforeStory()
	case GameOver:
		switch i {
		case 0:
			c, err := core.LoadCampaign(s.cfg.Catalog, bytes.NewReader(s.chapterStart))
			if err != nil {
				s.enterTitle(fmt.Sprintf("この章をやり直せません: %v", err))
				return
			}
			s.campaign = c
			s.sortie = nil
			s.battle = nil
			s.message = ""
			s.saveWarning = ""
			s.scene = Prepare
		case 1:
			s.enterTitle("")
		}
	}
}

// beforeStory・showStory は章の前後の物語を選び、空なら次へ進める（DES-STY-01）。
func (s *SceneController) beforeStory() { s.showStory(s.campaign.Chapter, false) }
func (s *SceneController) showStory(chapter int, after bool) {
	s.storyChapter = chapter
	s.storyAfter = after
	s.paragraph = 0
	s.paragraphs = nil
	s.joined = 0
	ch := s.cfg.Catalog.Chapters()[chapter]
	id := ch.Before
	if after {
		id = ch.After
	}
	if story, ok := s.cfg.Catalog.Story(id); id != "" && ok {
		s.paragraphs = story.Paragraphs
	}
	s.scene = Story
	if len(s.paragraphs) == 0 {
		s.endStory()
	}
}
func (s *SceneController) endStory() {
	if s.storyAfter {
		if s.campaign.Cleared {
			s.scene = Clear
		} else {
			s.beforeStory()
		}
		return
	}
	if len(s.cfg.Catalog.Chapters()[s.storyChapter].Joins) > 0 {
		s.scene = Joined
	} else {
		s.scene = Prepare
	}
}

// Advance は左クリックと、物語の Enter・Space の操作（DES-UI-12、15、17）。
func (s *SceneController) Advance() {
	switch s.scene {
	case Story:
		s.paragraph++
		if s.paragraph >= len(s.paragraphs) {
			s.endStory()
		}
	case Joined:
		s.joined++
		if s.joined >= len(s.cfg.Catalog.Chapters()[s.storyChapter].Joins) {
			s.scene = Prepare
		}
	case Prepare:
		s.StartBattle(s.DefaultMembers())
	case Battle:
		if s.battle == nil || s.battle.State() != Over {
			return
		}
		switch s.sortie.Battle.Result {
		case tactics.Won:
			finished := s.campaign.Chapter
			result, err := s.campaign.FinishChapter(s.sortie)
			if err != nil {
				s.message = err.Error()
				return
			}
			s.result = result
			s.storyChapter = finished
			s.saveChapter()
			s.scene = Result
		case tactics.Lost:
			s.scene = GameOver
		}
	case Result:
		s.showStory(s.storyChapter, true)
	case Clear:
		s.enterTitle("")
	}
}

// Cancel はルール選択の取消しと、物語の残りの省略（DES-UI-11、12）。
func (s *SceneController) Cancel() {
	switch s.scene {
	case RuleSelect:
		s.enterTitle("")
	case Story:
		s.endStory()
	case Title:
		if s.confirmOverwrite {
			s.confirmOverwrite = false
		}
	}
}

// DefaultMembers は仮の準備で名簿順に枠まで選ぶ。主人公を先頭にする（9 章 Issue 6）。
func (s *SceneController) DefaultMembers() []int {
	ch := s.cfg.Catalog.Chapters()[s.campaign.Chapter]
	m, _ := s.cfg.Catalog.Map(ch.Map)
	var members []int
	for i, v := range s.campaign.Roster {
		c, _ := s.cfg.Catalog.Character(v.Character)
		if v.Alive && c.Lord {
			members = append(members, i)
		}
	}
	for i, v := range s.campaign.Roster {
		c, _ := s.cfg.Catalog.Character(v.Character)
		if v.Alive && !c.Lord && len(members) < len(m.Slots) {
			members = append(members, i)
		}
	}
	return members
}

// StartBattle は準備で選んだ名簿から戦闘の Controller を作る（DES-CMP-05、DES-UI-09）。
func (s *SceneController) StartBattle(members []int) {
	if s.scene != Prepare {
		return
	}
	sortie, events, err := s.campaign.StartBattle(members, s.cfg.NextSeed())
	if err != nil {
		s.message = err.Error()
		return
	}
	ctrl, err := NewController(func() (*tactics.Battle, []tactics.Event, error) { return sortie.Battle, events, nil })
	if err != nil {
		s.message = err.Error()
		return
	}
	s.sortie = sortie
	s.battle = ctrl
	s.message = ""
	s.saveWarning = ""
	s.scene = Battle
}

// saveSummary はタイトルに章名とルール、またはクリア済みを出す（DES-UI-10）。
func (s *SceneController) saveSummary() string {
	if s.loaded == nil {
		return ""
	}
	if s.loaded.Cleared {
		return "クリア済み"
	}
	rule := "離脱"
	if s.loaded.Rule == core.Permadeath {
		rule = "ロスト"
	}
	return fmt.Sprintf("第 %d 章　%s（%s）", s.loaded.Chapter+1, s.cfg.Catalog.Chapters()[s.loaded.Chapter].Name, rule)
}

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

// Config は起動時に渡す目録・保存先・戦闘の種と開発用の開始章（DES-UI-09、DES-CMD-02・03）。
type Config struct {
	Catalog  *defs.Catalog
	Store    SaveStore
	NextSeed func() uint64
	// StartChapter は新規開始の章 ID。空なら最初の章から始める（DES-CMD-03）。
	StartChapter string
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
	Deploy
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
	prepareMember    int
	deployMembers    []int
	deploySlots      []int
	deploySelected   int
	deployCursor     tactics.Pos
	deployDanger     bool
	deployCamera     Camera
	deployment       map[int]bool
	// infoPage は準備と配置の情報欄のページ（DES-UI-28）。表示だけに使う。
	infoPage InfoPage
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
	if (s.scene == Battle || s.scene == GameOver) && s.battle != nil && s.battle.rewind.open {
		return
	}
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
		var c *core.Campaign
		var err error
		if s.cfg.StartChapter == "" {
			c, err = core.NewCampaign(s.cfg.Catalog, core.Rule(i))
		} else {
			c, err = core.NewCampaignAt(s.cfg.Catalog, core.Rule(i), s.cfg.StartChapter)
		}
		if err != nil {
			s.message = err.Error()
			return
		}
		s.campaign = c
		s.saveChapter()
		s.beforeStory()
	case Prepare:
		s.choosePrepare(i)
	case Deploy:
		s.chooseDeploy(i)
	case GameOver:
		switch i {
		case 2:
			if s.battle != nil {
				s.battle.OpenRewind()
			}
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
			s.enterPrepare()
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
		s.enterPrepare()
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
			s.enterPrepare()
		}
	case Prepare:
		s.enterDeploy()
	case Battle:
		if s.battle == nil || (s.battle.State() != Over || s.battle.rewind.open) {
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
		s.saveWarning = ""
		s.showStory(s.storyChapter, true)
	case Clear:
		s.enterTitle("")
	}
}

// Cancel はルール選択の取消し、物語の省略、配置の解除と準備への戻り（DES-UI-11、12、20）。
func (s *SceneController) Cancel() {
	if (s.scene == Battle || s.scene == GameOver) && s.battle != nil && s.battle.rewind.open {
		s.battle.Cancel()
		return
	}
	switch s.scene {
	case Deploy:
		if s.deploySelected >= 0 {
			s.deploySelected = -1
			s.message = ""
		} else {
			s.leaveDeploy()
		}
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

// startDeployedBattle は配置を確定して戦闘の Controller を作る（DES-UI-20）。
func (s *SceneController) startDeployedBattle() {
	if s.scene != Deploy {
		return
	}
	// 検証失敗では種を取得しない（DES-DEP-02、DES-UI-23）。
	if _, err := s.campaign.DeploymentView(s.deployMembers, s.deploySlots); err != nil {
		s.message = err.Error()
		return
	}
	sortie, events, err := s.campaign.StartBattleAt(s.deployMembers, s.deploySlots, s.cfg.NextSeed())
	if err != nil {
		s.message = err.Error()
		return
	}
	ctrl, err := NewController(func() (*tactics.Battle, []tactics.Event, error) { return sortie.Battle, events, nil })
	if err != nil {
		s.message = err.Error()
		return
	}
	s.resetDeploy()
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

// BattleGuide は戦闘の案内欄に章の結果反映の誤りも出す（DES-UI-17）。
func (s *SceneController) BattleGuide() []string {
	lines := s.battle.Guide()
	if s.message != "" {
		lines = append(lines, s.message)
	}
	return lines
}

// rewindKey は戦闘と敗北の選択だけで巻き戻しのキーを受け付ける（DES-UI-19、DES-UI-20）。
func (s *SceneController) rewindKey(key string) {
	if (s.scene == Battle || s.scene == GameOver) && s.battle != nil {
		s.battle.rewindKey(key)
	}
}

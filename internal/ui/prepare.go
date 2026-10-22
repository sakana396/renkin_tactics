package ui

import (
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// enterPrepare は生存者を名簿順に選び、主人公を情報の初期選択にする（DES-UI-13）。
func (s *SceneController) enterPrepare() {
	s.scene = Prepare
	s.saveWarning = ""
	s.message = ""
	s.deployment = map[int]bool{}
	for i, m := range s.campaign.Roster {
		ch, _ := s.cfg.Catalog.Character(m.Character)
		if m.Alive && ch.Lord {
			s.prepareMember = i
			s.deployment[i] = true
		}
	}
	for i, m := range s.campaign.Roster {
		if m.Alive && len(s.deployment) < s.prepareSlots() {
			s.deployment[i] = true
		}
	}
}
func (s *SceneController) prepareSlots() int {
	ch := s.cfg.Catalog.Chapters()[s.campaign.Chapter]
	m, _ := s.cfg.Catalog.Map(ch.Map)
	return len(m.Slots)
}

// SelectedMembers は印のある仲間を主人公・名簿順に返す（DES-UI-13）。
func (s *SceneController) SelectedMembers() []int {
	var lord, others []int
	for i, m := range s.campaign.Roster {
		if !s.deployment[i] || !m.Alive {
			continue
		}
		ch, _ := s.cfg.Catalog.Character(m.Character)
		if ch.Lord {
			lord = append(lord, i)
		} else {
			others = append(others, i)
		}
	}
	return append(lord, others...)
}

// toggleDeployment は人数と主人公の制約を守って印を切り替える（DES-UI-13）。
func (s *SceneController) toggleDeployment(i int) {
	ch, _ := s.cfg.Catalog.Character(s.campaign.Roster[i].Character)
	s.message = ""
	switch {
	case ch.Lord:
		s.message = "主人公は必ず出撃します"
	case s.deployment[i]:
		delete(s.deployment, i)
	case len(s.deployment) >= s.prepareSlots():
		s.message = fmt.Sprintf("出撃は %d 人までです", s.prepareSlots())
	default:
		s.deployment[i] = true
	}
}

// choosePrepare は描画順の在庫・手持ちボタンを操作する（DES-UI-13）。
func (s *SceneController) choosePrepare(i int) {
	if i < 0 {
		return
	}
	stock := s.campaign.Stock
	hand := s.campaign.Roster[s.prepareMember].Items
	var err error
	switch {
	case i < len(stock):
		err = s.campaign.TakeItem(s.prepareMember, stock[i].Item)
	case i < len(stock)+len(hand):
		err = s.campaign.StoreItem(s.prepareMember, hand[i-len(stock)].Item)
	case i == len(stock)+len(hand):
		s.StartBattle(s.SelectedMembers())
		return
	default:
		return
	}
	s.message = ""
	if errors.Is(err, core.ErrHandFull) {
		s.message = "手持ちは 5 個までです"
	} else if err != nil {
		s.message = err.Error()
	}
}

// PrepareRow は名簿の行と印の入力範囲、明るくする選択状態（DES-UI-13）。
type PrepareRow struct {
	Member     int
	Rect, Mark image.Rectangle
	Selected   bool
}

// prepareView は名簿・在庫・出撃と情報欄の配置を作る（DES-UI-13）。
func (s *SceneController) prepareView() SceneView {
	var v SceneView
	ch := s.cfg.Catalog.Chapters()[s.campaign.Chapter]
	v.Texts = append(v.Texts, SceneText{Text: fmt.Sprintf("第 %d 章　%s　出撃 %d/%d", s.campaign.Chapter+1, ch.Name, len(s.SelectedMembers()), s.prepareSlots()), X: 16, Y: 24, Style: StyleTitle})
	y := 72
	for i, m := range s.campaign.Roster {
		if !m.Alive {
			continue
		}
		u := s.prepareUnit(i)
		mark := "[ ]"
		if s.deployment[i] {
			mark = "[✓]"
		}
		v.Rows = append(v.Rows, PrepareRow{Member: i, Rect: image.Rect(56, y, MapSize-8, y+36), Mark: image.Rect(16, y, 52, y+36), Selected: i == s.prepareMember})
		v.Texts = append(v.Texts, SceneText{Text: mark, X: 20, Y: y + 10}, SceneText{Text: fmt.Sprintf("%s　%s　Lv %d　HP %d　手持ち %d", u.Name, u.ClassName, m.Level, m.Stats.HP, itemCount(m.Items)), X: 96, Y: y + 10})
		v.Icons = append(v.Icons, SceneIcon{unitTile(u), 60, y + 2})
		y += 40
	}
	y += 12
	var stock []string
	for _, ic := range s.campaign.Stock {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		stock = append(stock, fmt.Sprintf("%s ×%d", it.Name, ic.Count))
	}
	label := strings.Join(stock, "　")
	if label == "" {
		label = "なし"
	}
	v.Texts = append(v.Texts, SceneText{Text: "在庫: " + label, X: 16, Y: y})
	y += 28
	for col, items := range [][]defs.ItemCount{s.campaign.Stock, s.campaign.Roster[s.prepareMember].Items} {
		for n, ic := range items {
			it, _ := s.cfg.Catalog.Item(ic.Item)
			prefix := "手持ちへ: "
			if col == 1 {
				prefix = "在庫へ: "
			}
			v.Buttons = append(v.Buttons, SceneButton{prefix + it.Name, image.Rect(16+col*248, y+n*40, 252+col*248, y+n*40+36), true})
		}
	}
	v.Buttons = append(v.Buttons, SceneButton{"出撃", image.Rect(360, 528, 504, 568), true})
	info := s.prepareInfo()
	v.Info = &info
	return v
}

// prepareUnit は名簿と定義から情報表示用の値を作る。戦闘は作らず乱数も引かない（DES-UI-13）。
func (s *SceneController) prepareUnit(i int) tactics.Unit {
	m := s.campaign.Roster[i]
	ch, _ := s.cfg.Catalog.Character(m.Character)
	cls, _ := s.cfg.Catalog.Class(ch.Class)
	w, _ := s.cfg.Catalog.Weapon(ch.Weapon)
	wt := map[string]tactics.WeaponType{"sword": tactics.Sword, "lance": tactics.Lance, "axe": tactics.Axe, "bow": tactics.Bow, "staff": tactics.Staff}[string(w.Type)]
	u := tactics.Unit{Name: ch.Name, ClassName: cls.Name, Side: tactics.Player, Stats: m.Stats, HP: m.Stats.HP, Level: m.Level, Exp: m.Exp, Lord: ch.Lord, Alive: m.Alive,
		Weapon: tactics.Weapon{Name: w.Name, Type: wt, Might: w.Might, Hit: w.Hit, MinRange: w.MinRange, MaxRange: w.MaxRange, Heal: w.Heal, Uses: w.Uses, MaxUses: w.Uses}}
	for _, ic := range m.Items {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		u.Items = append(u.Items, tactics.ItemStack{Item: tactics.Item{ID: it.ID, Name: it.Name, Heal: it.Heal}, Count: ic.Count})
	}
	if sk, ok := s.cfg.Catalog.Skill(ch.Skill); ok {
		effect := map[string]tactics.SkillEffect{"inspire": tactics.Inspire, "pavise": tactics.Pavise, "wrath": tactics.Wrath, "snipe": tactics.Snipe}[string(sk.Effect)]
		u.Skill = tactics.Skill{Name: sk.Name, Effect: effect, Range: sk.Range, Hit: sk.Hit, Avoid: sk.Avoid, Chance: sk.Chance, Multiplier: sk.Multiplier}
	}
	return u
}

// prepareInfo は戦闘の情報欄と同じ部品で能力・装備を表示する（DES-UI-13）。
func (s *SceneController) prepareInfo() InfoView {
	u := s.prepareUnit(s.prepareMember)
	return (Panel{Unit: &u}).View()
}

func itemCount(items []defs.ItemCount) int {
	n := 0
	for _, ic := range items {
		n += ic.Count
	}
	return n
}

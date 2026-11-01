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
	s.prepareStockTop = 0
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

// prepareStockRows は準備の在庫の列に一度に出すボタンの数。超える分は「▲」「▼」で送る（DES-IT06 12.1 節）。
const prepareStockRows = 5

// prepareButton は準備のボタンと、押したときの操作（DES-UI-13）。
type prepareButton struct {
	SceneButton
	act func() error
}

// prepareButtons は在庫・手持ちの道具と武器、配置へ、在庫を送るボタンを描画順に作る（DES-UI-13、DES-IT06 12.1 節）。
// 返す y は品のボタンの列の下端の行数（案内の位置に使う）。
func (s *SceneController) prepareButtons(y int) ([]prepareButton, int) {
	c := s.campaign
	m := s.prepareMember
	stock, stockWeapons := c.Stock, c.StockWeapons
	hand, handWeapons := c.Roster[m].Items, c.Roster[m].Weapons
	total := len(stock) + len(stockWeapons)
	s.prepareStockTop = min(max(s.prepareStockTop, 0), max(total-prepareStockRows, 0))
	top, end := s.prepareStockTop, min(total, s.prepareStockTop+prepareStockRows)
	rect := func(col, row int) image.Rectangle {
		return image.Rect(16+col*160, y+row*22, 172+col*160, y+row*22+20)
	}
	var bs []prepareButton
	for k := top; k < end && k < len(stock); k++ {
		it, _ := s.cfg.Catalog.Item(stock[k].Item)
		id := stock[k].Item
		bs = append(bs, prepareButton{SceneButton{"手持ちへ: " + it.Name, rect(0, k-top), true}, func() error { return c.TakeItem(m, id) }})
	}
	for n, ic := range hand {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		id := ic.Item
		bs = append(bs, prepareButton{SceneButton{"在庫へ: " + it.Name, rect(1, n), true}, func() error { return c.StoreItem(m, id) }})
	}
	for k := max(top, len(stock)); k < end; k++ {
		w, _ := s.cfg.Catalog.Weapon(stockWeapons[k-len(stock)].Weapon)
		id := w.ID
		bs = append(bs, prepareButton{SceneButton{"手持ちへ: " + w.Name, rect(0, k-top), true}, func() error { return c.TakeWeapon(m, id) }})
	}
	for n, id := range handWeapons {
		w, _ := s.cfg.Catalog.Weapon(id)
		n := n
		bs = append(bs, prepareButton{SceneButton{"在庫へ: " + w.Name, rect(1, len(hand)+n), true}, func() error { return c.StoreWeapon(m, n) }})
	}
	for n, id := range handWeapons {
		w, _ := s.cfg.Catalog.Weapon(id)
		n := n
		bs = append(bs, prepareButton{SceneButton{"先頭へ: " + w.Name, rect(2, len(hand)+n), true}, func() error { return c.MoveWeaponFirst(m, n) }})
	}
	rows := max(end-top, len(hand)+len(handWeapons))
	bs = append(bs, prepareButton{SceneButton{"配置へ", image.Rect(360, 528, 504, 568), true}, nil})
	if total > prepareStockRows {
		r := rect(0, prepareStockRows)
		bs = append(bs,
			prepareButton{SceneButton{"▲ 前へ", image.Rect(r.Min.X, r.Min.Y, r.Min.X+76, r.Max.Y), top > 0}, func() error { s.prepareStockTop--; return nil }},
			prepareButton{SceneButton{"▼ 次へ", image.Rect(r.Max.X-76, r.Min.Y, r.Max.X, r.Max.Y), end < total}, func() error { s.prepareStockTop++; return nil }})
		rows = max(rows, prepareStockRows+1)
	}
	return bs, rows
}

// choosePrepare は描画順の在庫・手持ちボタンを操作する（DES-UI-13）。
func (s *SceneController) choosePrepare(i int) {
	bs, _ := s.prepareButtons(0)
	if i < 0 || i >= len(bs) || !bs[i].Enabled {
		return
	}
	if bs[i].act == nil {
		s.enterDeploy()
		return
	}
	err := bs[i].act()
	s.message = ""
	if errors.Is(err, core.ErrHandFull) {
		s.message = "手持ちは 5 枠までです"
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
		v.Rows = append(v.Rows, PrepareRow{Member: i, Rect: image.Rect(56, y, MapSize-8, y+24), Mark: image.Rect(16, y, 52, y+24), Selected: i == s.prepareMember})
		v.Texts = append(v.Texts, SceneText{Text: mark, X: 20, Y: y + 4}, SceneText{Text: fmt.Sprintf("%s　%s　Lv %d　HP %d　手持ち %d/5", u.Name, u.ClassName, m.Level, m.Stats.HP, len(m.Weapons)+itemCount(m.Items)), X: 96, Y: y + 4})
		v.Icons = append(v.Icons, SceneIcon{unitTile(u), 60, y})
		y += 24
	}
	y += 12
	var stock []string
	for _, ic := range s.campaign.Stock {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		stock = append(stock, fmt.Sprintf("%s ×%d", it.Name, ic.Count))
	}
	for _, wc := range s.campaign.StockWeapons {
		w, _ := s.cfg.Catalog.Weapon(wc.Weapon)
		stock = append(stock, fmt.Sprintf("%s ×%d", w.Name, wc.Count))
	}
	label := strings.Join(stock, "　")
	if label == "" {
		label = "なし"
	} else if len([]rune(label)) > 45 {
		label = fmt.Sprintf("道具 %d 種・武器 %d 種（下の一覧から選ぶ）", len(s.campaign.Stock), len(s.campaign.StockWeapons))
	}
	bs, rows := s.prepareButtons(y + 24)
	if total := len(stock); total > prepareStockRows {
		label += fmt.Sprintf("　%d〜%d/%d", s.prepareStockTop+1, s.prepareStockTop+prepareStockRows, total)
	}
	v.Texts = append(v.Texts, SceneText{Text: "在庫: " + label, X: 16, Y: y})
	y += 24
	// 案内は品のボタンの最後から 4 px 下。品がないときは最初のボタンの位置（DES-UI-13）。
	if rows > 0 {
		y += rows*22 + 4
	}
	if s.message != "" {
		for i, line := range wrapSceneTextWidth(s.message, MapSize-48-8) {
			v.Texts = append(v.Texts, SceneText{Text: line, X: 48, Y: y + i*20, Style: StyleBad})
		}
	}
	for _, b := range bs {
		v.Buttons = append(v.Buttons, b.SceneButton)
	}
	info := s.prepareInfo()
	v.Info = &info
	return v
}

// prepareUnit は名簿と定義から情報表示用の値を作る。戦闘は作らず乱数も引かない（DES-UI-13）。
func (s *SceneController) prepareUnit(i int) tactics.Unit {
	m := s.campaign.Roster[i]
	ch, _ := s.cfg.Catalog.Character(m.Character)
	cls, _ := s.cfg.Catalog.Class(ch.Class)
	wt := map[string]tactics.WeaponType{"sword": tactics.Sword, "lance": tactics.Lance, "axe": tactics.Axe, "bow": tactics.Bow, "staff": tactics.Staff, "magic": tactics.Magic}
	u := tactics.Unit{Name: ch.Name, ClassName: cls.Name, Side: tactics.Player, Stats: m.Stats, HP: m.Stats.HP, Level: m.Level, Exp: m.Exp, Lord: ch.Lord, Alive: m.Alive,
		AllowedWeaponTypes: []tactics.WeaponType{wt[cls.WeaponType]}}
	// 名簿の武器の列を順に写し、先頭（初めの装備）を装備にする（DES-ITM-02、REV-069 の申し送り）。
	for _, id := range m.Weapons {
		w, _ := s.cfg.Catalog.Weapon(id)
		tw := tactics.Weapon{ID: w.ID, Name: w.Name, Type: wt[w.Type], Crit: w.Crit, Weight: w.Weight, Strikes: w.Strikes, EffectiveTags: append([]string(nil), w.EffectiveTags...), Might: w.Might, Hit: w.Hit, MinRange: w.MinRange, MaxRange: w.MaxRange, Heal: w.Heal, Uses: w.Uses, MaxUses: w.Uses}
		if w.DamageType == "magic" {
			tw.DamageType = tactics.Magical
		}
		u.Weapons = append(u.Weapons, tw)
	}
	for _, ic := range m.Items {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		u.Items = append(u.Items, tactics.ItemStack{Item: tactics.Item{ID: it.ID, Name: it.Name, Heal: it.Heal}, Count: ic.Count})
	}
	effects := map[string]tactics.SkillEffect{"inspire": tactics.Inspire, "pavise": tactics.Pavise, "wrath": tactics.Wrath, "snipe": tactics.Snipe, "canto": tactics.Canto}
	if sk, ok := s.cfg.Catalog.Skill(ch.Skill); ok {
		u.Skill = tactics.Skill{Name: sk.Name, Effect: effects[string(sk.Effect)], Range: sk.Range, Hit: sk.Hit, Avoid: sk.Avoid, Chance: sk.Chance, Multiplier: sk.Multiplier}
	}
	// 兵種のスキルも戦闘と同じく並べる（DES-BTL-10、DES-IT06 12.2 節）。
	for _, id := range cls.Skills {
		if sk, ok := s.cfg.Catalog.Skill(id); ok {
			u.ClassSkills = append(u.ClassSkills, tactics.Skill{Name: sk.Name, Effect: effects[string(sk.Effect)], Range: sk.Range, Hit: sk.Hit, Avoid: sk.Avoid, Chance: sk.Chance, Multiplier: sk.Multiplier})
		}
	}
	return u
}

// prepareInfo は戦闘の情報欄と同じ部品で能力・装備を表示する（DES-UI-13）。
func (s *SceneController) prepareInfo() InfoView {
	u := s.prepareUnit(s.prepareMember)
	return (Panel{Unit: &u, Page: s.infoPage, Tabs: true}).View()
}

func itemCount(items []defs.ItemCount) int {
	n := 0
	for _, ic := range items {
		n += ic.Count
	}
	return n
}

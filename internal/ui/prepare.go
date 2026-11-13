package ui

import (
	"errors"
	"fmt"
	"image"

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
	s.prepareRosterTop = 0
	s.prepareTab = TabRoster
	s.prepareItem = prepareItem{}
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

// PrepareTabKind は準備の表示タブ（DES-UI-32）。表示だけを切り替え、名簿・在庫・出撃の印は変えない。
type PrepareTabKind int

const (
	// TabRoster は「名簿」（生存者の行・出撃の印と、選んだ仲間の能力・装備）。
	TabRoster PrepareTabKind = iota
	// TabItems は「持ち物」（選んだ仲間の 5 枠と在庫）。
	TabItems
)

// PrepareTab は準備のタブの 1 つ（DES-UI-32）。描画とクリックで同じ範囲を使う。
type PrepareTab struct {
	Rect     image.Rectangle
	Label    string
	Kind     PrepareTabKind
	Selected bool
}

// 準備の画面の領域（DES-IT07 3.5 節。左上を (0,0) とし、右・下の端を含まない）。
var (
	prepareTabArea    = image.Rect(16, 16, 496, 48)
	prepareLeftArea   = image.Rect(16, 56, 400, 472)
	prepareRightArea  = image.Rect(416, 56, 816, 472)
	prepareBottomArea = image.Rect(0, 512, 832, 576)
	// prepareDeployRect は下欄の「配置へ」のボタン。
	prepareDeployRect = image.Rect(672, 528, 816, 568)
)

const (
	// prepareRosterRows は名簿に一度に出す生存者の行の数（DES-IT06 12.1 節）。超える分は「▲ 名簿」「▼ 名簿」で送る。
	prepareRosterRows = 8
	// prepareRosterY は名簿の最初の行の y。行は 24 px ごと（DES-IT06 12.1 節）。
	prepareRosterY = 64
	// prepareRosterArtSize は名簿の行の絵の一辺。行の高さ（24 px）に収め、隣の行と重ねない（DES-NG-08）。
	prepareRosterArtSize = 24
	// prepareSlotY・prepareSlotStep・prepareSlotHeight は持ち物の 5 枠の行の位置（DES-UI-32）。
	prepareSlotY      = 88
	prepareSlotStep   = 32
	prepareSlotHeight = 28
	// prepareStockLines は在庫の列に一度に出す行（見出しを含む）の数。超える分は「▲ 前へ」「▼ 次へ」で送る（DES-UI-32）。
	prepareStockLines = 10
	// prepareStockY・prepareStockStep は在庫の行の位置。品のボタンは高さ 20（DES-IT06 12.1 節）。
	prepareStockY    = 84
	prepareStockStep = 24
	// prepareDetailY は選んだ品の性能と理由を出す右列の下部の y（DES-UI-32）。
	prepareDetailY = 360
	// prepareHandSlots は手持ちの枠の数（DES-ITM-02）。
	prepareHandSlots = 5
)

// prepareItem は持ち物の表示で選んだ品（DES-UI-32）。表示だけに使い、状態は変えない。
type prepareItem struct {
	ok     bool
	weapon bool
	id     string
}

// prepareTabs は「名簿」「持ち物」のタブをタブの領域に並べる（DES-UI-32）。
func (s *SceneController) prepareTabs() []PrepareTab {
	a := prepareTabArea
	mid := (a.Min.X + a.Max.X) / 2
	tabs := []PrepareTab{
		{Rect: image.Rect(a.Min.X, a.Min.Y, mid-4, a.Max.Y), Label: "名簿", Kind: TabRoster},
		{Rect: image.Rect(mid+4, a.Min.Y, a.Max.X, a.Max.Y), Label: "持ち物", Kind: TabItems},
	}
	for i := range tabs {
		tabs[i].Selected = tabs[i].Kind == s.prepareTab
	}
	return tabs
}

// setPrepareTab はタブを切り替える。選んだ仲間・名簿・在庫・出撃の印は変えず、品の選択と案内だけを消す（DES-UI-32）。
func (s *SceneController) setPrepareTab(k PrepareTabKind) {
	s.prepareTab = k
	s.prepareItem = prepareItem{}
	s.message = ""
}

// livingMembers は名簿のうち生きている仲間の添字を名簿の順に返す。名簿の送りは戦死者を数えない（DES-IT06 12.1 節）。
func (s *SceneController) livingMembers() []int {
	var out []int
	for i, m := range s.campaign.Roster {
		if m.Alive {
			out = append(out, i)
		}
	}
	return out
}

// prepareButton は準備のボタンと、押したときの操作（DES-UI-13、DES-UI-32）。
// item.ok なら、押したときにその品を選ぶ（性能と理由を右列の下部に出す）。
type prepareButton struct {
	SceneButton
	act  func() error
	item prepareItem
}

// prepareLine は持ち物の右列の在庫の 1 行。見出し（武器・杖・道具）か品のボタン（DES-UI-32）。
type prepareLine struct {
	heading string
	// name は品名、suffix は個数（2 つ以上のときの「 ×n」）。ボタンの文は「手持ちへ: 」＋品名＋個数。
	name, suffix string
	act          func() error
	item         prepareItem
}

// prepareStock は在庫を武器・杖・道具の見出しで分けた行に並べる（DES-UI-32）。
// 各見出しの中は在庫の順のまま。品がない見出しは出さない。
func (s *SceneController) prepareStock() []prepareLine {
	c := s.campaign
	m := s.prepareMember
	count := func(n int) string {
		if n > 1 {
			return fmt.Sprintf(" ×%d", n)
		}
		return ""
	}
	var weapons, staves, items []prepareLine
	for _, wc := range c.StockWeapons {
		w, _ := s.cfg.Catalog.Weapon(wc.Weapon)
		id := w.ID
		l := prepareLine{name: w.Name, suffix: count(wc.Count), act: func() error { return c.TakeWeapon(m, id) }, item: prepareItem{ok: true, weapon: true, id: id}}
		if w.Type == defs.WeaponStaff {
			staves = append(staves, l)
		} else {
			weapons = append(weapons, l)
		}
	}
	for _, ic := range c.Stock {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		id := ic.Item
		items = append(items, prepareLine{name: it.Name, suffix: count(ic.Count), act: func() error { return c.TakeItem(m, id) }, item: prepareItem{ok: true, id: id}})
	}
	var out []prepareLine
	for _, g := range []struct {
		name  string
		lines []prepareLine
	}{{"武器", weapons}, {"杖", staves}, {"道具", items}} {
		if len(g.lines) > 0 {
			out = append(out, prepareLine{heading: g.name})
			out = append(out, g.lines...)
		}
	}
	return out
}

// prepareSlot は持ち物の 5 枠の 1 行（DES-UI-32）。
type prepareSlot struct {
	name     string
	weapon   int // 武器の列の添字。道具・空きなら -1。
	item     string
	equipped bool
}

// prepareHand は選んだ仲間の手持ちを常に 5 行の枠に並べる（DES-UI-32、DES-ITM-02）。
// 武器の列の順に武器、続けて道具を 1 個ずつ並べ、残りは空き（name が空）。先頭の武器が初めの装備。
func (s *SceneController) prepareHand() []prepareSlot {
	m := s.campaign.Roster[s.prepareMember]
	var out []prepareSlot
	for n, id := range m.Weapons {
		w, _ := s.cfg.Catalog.Weapon(id)
		out = append(out, prepareSlot{name: w.Name, weapon: n, equipped: n == 0})
	}
	for _, ic := range m.Items {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		for range ic.Count {
			out = append(out, prepareSlot{name: it.Name, weapon: -1, item: ic.Item})
		}
	}
	for len(out) < prepareHandSlots {
		out = append(out, prepareSlot{weapon: -1})
	}
	return out
}

// prepareButtons は今のタブのボタンを描画順に作る（DES-UI-13、DES-UI-32、DES-IT06 12.1 節）。
// 名簿: 名簿の送り。持ち物: 5 枠の「在庫へ」と「先頭へ」、在庫の「手持ちへ」と送り。どちらも最後に「配置へ」。
func (s *SceneController) prepareButtons() []prepareButton {
	c := s.campaign
	m := s.prepareMember
	var bs []prepareButton
	if s.prepareTab == TabRoster {
		if living := len(s.livingMembers()); living > prepareRosterRows {
			y := prepareRosterY + prepareRosterRows*24 + 8
			bs = append(bs,
				prepareButton{SceneButton: SceneButton{"▲ 名簿", image.Rect(232, y, 312, y+24), s.prepareRosterTop > 0}, act: func() error { s.prepareRosterTop--; return nil }},
				prepareButton{SceneButton: SceneButton{"▼ 名簿", image.Rect(320, y, 400, y+24), s.prepareRosterTop < living-prepareRosterRows}, act: func() error { s.prepareRosterTop++; return nil }})
		}
	} else {
		for k, sl := range s.prepareHand() {
			y := prepareSlotY + k*prepareSlotStep
			main := image.Rect(40, y, 232, y+prepareSlotHeight)
			side := image.Rect(240, y, prepareLeftArea.Max.X, y+prepareSlotHeight)
			switch {
			case sl.name == "":
				bs = append(bs, prepareButton{SceneButton: SceneButton{"―", main, false}})
			case sl.weapon >= 0:
				n := sl.weapon
				item := prepareItem{ok: true, weapon: true, id: c.Roster[m].Weapons[n]}
				bs = append(bs, prepareButton{SceneButton: SceneButton{fitLabel("在庫へ: ", sl.name, "", main), main, true}, act: func() error { return c.StoreWeapon(m, n) }, item: item})
				if n == 0 {
					bs = append(bs, prepareButton{SceneButton: SceneButton{"初めの装備", side, false}})
				} else {
					bs = append(bs, prepareButton{SceneButton: SceneButton{fitLabel("先頭へ: ", sl.name, "", side), side, true}, act: func() error { return c.MoveWeaponFirst(m, n) }, item: item})
				}
			default:
				id := sl.item
				bs = append(bs, prepareButton{SceneButton: SceneButton{fitLabel("在庫へ: ", sl.name, "", main), main, true}, act: func() error { return c.StoreItem(m, id) }, item: prepareItem{ok: true, id: id}})
			}
		}
		lines := s.prepareStock()
		total := len(lines)
		s.prepareStockTop = min(max(s.prepareStockTop, 0), max(total-prepareStockLines, 0))
		top, end := s.prepareStockTop, min(total, s.prepareStockTop+prepareStockLines)
		for k := top; k < end; k++ {
			l := lines[k]
			if l.heading != "" {
				continue
			}
			y := prepareStockY + (k-top)*prepareStockStep
			r := image.Rect(432, y, 800, y+20)
			bs = append(bs, prepareButton{SceneButton: SceneButton{fitLabel("手持ちへ: ", l.name, l.suffix, r), r, true}, act: l.act, item: l.item})
		}
		if total > prepareStockLines {
			y := prepareStockY + prepareStockLines*prepareStockStep
			bs = append(bs,
				prepareButton{SceneButton: SceneButton{"▲ 前へ", image.Rect(432, y, 512, y+20), top > 0}, act: func() error { s.prepareStockTop--; return nil }},
				prepareButton{SceneButton: SceneButton{"▼ 次へ", image.Rect(720, y, 800, y+20), end < total}, act: func() error { s.prepareStockTop++; return nil }})
		}
	}
	bs = append(bs, prepareButton{SceneButton: SceneButton{"配置へ", prepareDeployRect, true}})
	return bs
}

// choosePrepare は描画順のボタンを操作する（DES-UI-13、DES-UI-32）。出し入れは既存の操作を呼び、失敗では状態を変えず理由を出す。
func (s *SceneController) choosePrepare(i int) {
	bs := s.prepareButtons()
	if i < 0 || i >= len(bs) || !bs[i].Enabled {
		return
	}
	if bs[i].act == nil {
		s.enterDeploy()
		return
	}
	if bs[i].item.ok {
		s.prepareItem = bs[i].item
	}
	err := bs[i].act()
	s.message = ""
	if errors.Is(err, core.ErrHandFull) {
		s.message = "手持ちは 5 枠までです"
	} else if err != nil {
		s.message = err.Error()
	}
}

// cancelPrepare は右クリック・Esc を、品の選択 → タブの選択 → 準備画面の既存の戻り先（なし）の順に戻す（DES-UI-32）。
func (s *SceneController) cancelPrepare() {
	switch {
	case s.prepareTab == TabItems && s.prepareItem.ok:
		s.prepareItem = prepareItem{}
		s.message = ""
	case s.prepareTab == TabItems:
		s.setPrepareTab(TabRoster)
	}
}

// clickPrepare は準備のクリックを、情報欄のページの印 → タブ → 名簿の行・印 → ボタンの順に受ける（DES-UI-13、DES-UI-32）。
func (s *SceneController) clickPrepare(x, y int) {
	if s.infoTabClick(x, y) {
		return
	}
	pt := image.Pt(x, y)
	for _, tab := range s.prepareTabs() {
		if pt.In(tab.Rect) {
			if tab.Kind != s.prepareTab {
				s.setPrepareTab(tab.Kind)
			}
			return
		}
	}
	v := s.View()
	for _, r := range v.Rows {
		if pt.In(r.Mark) {
			s.toggleDeployment(r.Member)
			return
		}
		if pt.In(r.Rect) {
			s.prepareMember = r.Member
			s.message = ""
			return
		}
	}
	for i, b := range v.Buttons {
		if pt.In(b.Rect) && b.Enabled {
			s.Choose(i)
			return
		}
	}
}

// PrepareRow は名簿の行と印の入力範囲、明るくする選択状態（DES-UI-13）。
type PrepareRow struct {
	Member     int
	Rect, Mark image.Rectangle
	Selected   bool
}

// fitText は文が幅に収まらなければ末尾を「…」にして縮める（DES-UI-32。長い名前でも行が隣と重ならないように）。
func fitText(s string, width int) string {
	if textWidth(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r)+"…") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// fitLabel はボタンの文「prefix 品名 suffix」を、描くときの左の余白 12 px を除いたボタンの幅に収める（DES-UI-32）。
// 収まらなければ品名だけを「…」で縮め、操作の言葉と個数は残す。
func fitLabel(prefix, name, suffix string, r image.Rectangle) string {
	return prefix + fitText(name, r.Dx()-12-textWidth(prefix)-textWidth(suffix)) + suffix
}

// prepareView は準備の画面を作る。タブ、左列・右列（名簿／持ち物）、下欄の理由と「配置へ」（DES-UI-13、DES-UI-32）。
func (s *SceneController) prepareView() SceneView {
	var v SceneView
	ch := s.cfg.Catalog.Chapters()[s.campaign.Chapter]
	v.Texts = append(v.Texts, SceneText{Text: fmt.Sprintf("第 %d 章　%s　出撃 %d/%d", s.campaign.Chapter+1, ch.Name, len(s.SelectedMembers()), s.prepareSlots()), X: 512, Y: 24, Style: StyleTitle})
	v.Tabs = s.prepareTabs()
	if s.prepareTab == TabRoster {
		s.rosterView(&v)
	} else {
		s.itemsView(&v)
	}
	// 下欄: 操作の理由は「配置へ」の左に、下欄の中で 3 行まで折り返す（DES-UI-32）。
	if s.message != "" {
		width := prepareDeployRect.Min.X - 16 - 16
		for i, line := range wrapSceneTextWidth(s.message, width) {
			if i >= 3 {
				break
			}
			v.Texts = append(v.Texts, SceneText{Text: line, X: 16, Y: prepareBottomArea.Min.Y + 4 + i*16, Style: StyleBad})
		}
	}
	for _, b := range s.prepareButtons() {
		v.Buttons = append(v.Buttons, b.SceneButton)
	}
	return v
}

// rosterView は名簿のタブ。左列に生存者の行と出撃の印、右列に選んだ仲間の能力・装備（DES-UI-13、DES-UI-32）。
func (s *SceneController) rosterView(v *SceneView) {
	y := prepareRosterY
	living := s.livingMembers()
	top := min(max(s.prepareRosterTop, 0), len(living))
	for _, i := range living[top:min(top+prepareRosterRows, len(living))] {
		m := s.campaign.Roster[i]
		u := s.prepareUnit(i)
		mark := "[ ]"
		if s.deployment[i] {
			mark = "[✓]"
		}
		v.Rows = append(v.Rows, PrepareRow{Member: i, Rect: image.Rect(56, y, prepareLeftArea.Max.X, y+24), Mark: image.Rect(16, y, 52, y+24), Selected: i == s.prepareMember})
		row := fitText(fmt.Sprintf("%s　%s　Lv %d　HP %d　手持ち %d/5", u.Name, u.ClassName, m.Level, m.Stats.HP, len(m.Weapons)+itemCount(m.Items)), prepareLeftArea.Max.X-96-4)
		v.Texts = append(v.Texts, SceneText{Text: mark, X: 20, Y: y + 4}, SceneText{Text: row, X: 96, Y: y + 4})
		v.Icons = append(v.Icons, SceneIcon{Tile: unitTile(u), X: 60, Y: y, Art: s.sceneArtKey(artKeyOf(u)), Size: prepareRosterArtSize, Face: true})
		y += 24
	}
	info := s.prepareInfo()
	v.Info = &info
	v.InfoArea = prepareRightArea
}

// itemsView は持ち物のタブ。左列に選んだ仲間の 5 枠、右列に見出しで分けた在庫と、選んだ品の性能・理由（DES-UI-32）。
func (s *SceneController) itemsView(v *SceneView) {
	m := s.campaign.Roster[s.prepareMember]
	u := s.prepareUnit(s.prepareMember)
	head := fitText(fmt.Sprintf("%s（%s）　手持ち %d/5", u.Name, u.ClassName, len(m.Weapons)+itemCount(m.Items)), prepareLeftArea.Dx())
	v.Texts = append(v.Texts, SceneText{Text: head, X: prepareLeftArea.Min.X, Y: prepareLeftArea.Min.Y + 4, Style: StyleTitle})
	for k, sl := range s.prepareHand() {
		if sl.equipped {
			v.Texts = append(v.Texts, SceneText{Text: "E", X: 20, Y: prepareSlotY + k*prepareSlotStep + (prepareSlotHeight-lineHeight)/2, Style: StyleTitle})
		}
	}
	guideY := prepareSlotY + prepareHandSlots*prepareSlotStep + 4
	v.Texts = append(v.Texts,
		SceneText{Text: "E: 初めの装備（武器の列の先頭）", X: 16, Y: guideY, Style: StyleLabel},
		SceneText{Text: "右クリック・Esc: 品の選択 → 名簿へ", X: 16, Y: guideY + 20, Style: StyleLabel})
	// 右列: 在庫の見出し、送りの位置、武器・杖・道具の見出し。
	lines := s.prepareStock()
	title := "在庫"
	top := min(max(s.prepareStockTop, 0), max(len(lines)-prepareStockLines, 0))
	if len(lines) > prepareStockLines {
		title += fmt.Sprintf("　%d〜%d/%d", top+1, top+prepareStockLines, len(lines))
	}
	v.Texts = append(v.Texts, SceneText{Text: title, X: prepareRightArea.Min.X, Y: prepareRightArea.Min.Y + 4, Style: StyleTitle})
	if len(lines) == 0 {
		v.Texts = append(v.Texts, SceneText{Text: "なし", X: 432, Y: prepareStockY + 2})
	}
	for k := top; k < min(len(lines), top+prepareStockLines); k++ {
		if lines[k].heading != "" {
			v.Texts = append(v.Texts, SceneText{Text: lines[k].heading, X: prepareRightArea.Min.X, Y: prepareStockY + (k-top)*prepareStockStep + 2, Style: StyleLabel})
		}
	}
	for i, d := range s.prepareDetail() {
		d.X, d.Y = prepareRightArea.Min.X, prepareDetailY+i*16
		v.Texts = append(v.Texts, d)
	}
}

// prepareDetail は選んだ品の名前・性能と、選んだ仲間が装備できない理由を、右列の幅で折り返した行で返す（DES-UI-32）。
func (s *SceneController) prepareDetail() []SceneText {
	it := s.prepareItem
	if !it.ok {
		return nil
	}
	var out []SceneText
	add := func(str string, st InfoStyle) {
		for _, line := range wrapSceneTextWidth(str, prepareRightArea.Dx()) {
			out = append(out, SceneText{Text: line, Style: st})
		}
	}
	if !it.weapon {
		d, _ := s.cfg.Catalog.Item(it.id)
		add("選んだ品: "+d.Name, StyleTitle)
		if d.Bomb {
			add("道具　"+itemEffect(core.ItemOf(d)), StylePlain)
		} else {
			add(fmt.Sprintf("道具　HP を %d 回復", d.Heal), StylePlain)
		}
	} else {
		w := s.prepareWeapon(it.id)
		add("選んだ品: "+w.Name, StyleTitle)
		add(weaponDetail(w), StylePlain)
		m := s.campaign.Roster[s.prepareMember]
		chr, _ := s.cfg.Catalog.Character(m.Character)
		cls, _ := s.cfg.Catalog.Class(chr.Class)
		if d, _ := s.cfg.Catalog.Weapon(it.id); !cls.CanUseWeapon(d.Type) {
			add(fmt.Sprintf("%s（%s）は%sを装備できない", chr.Name, cls.Name, weaponTypeName(w.Type)), StyleBad)
		}
	}
	// 右列の下部（y < 472）に収まる行だけを出す。
	return out[:min(len(out), (prepareRightArea.Max.Y-prepareDetailY)/16)]
}

// prepareUnit は名簿と定義から情報表示用の値を作る。戦闘は作らず乱数も引かない（DES-UI-13）。
func (s *SceneController) prepareUnit(i int) tactics.Unit {
	m := s.campaign.Roster[i]
	ch, _ := s.cfg.Catalog.Character(m.Character)
	cls, _ := s.cfg.Catalog.Class(ch.Class)
	wt := prepareWeaponTypes
	u := tactics.Unit{Name: ch.Name, ClassName: cls.Name, Side: tactics.Player, Stats: m.Stats, HP: m.Stats.HP, Level: m.Level, Exp: m.Exp, Lord: ch.Lord, Alive: m.Alive,
		AllowedWeaponTypes: func() []tactics.WeaponType {
			kinds := cls.AllowedWeaponTypes()
			out := make([]tactics.WeaponType, len(kinds))
			for i, kind := range kinds {
				out[i] = wt[kind]
			}
			return out
		}()}
	// 名簿の武器の列を順に写し、先頭（初めの装備）を装備にする（DES-ITM-02、REV-069 の申し送り）。
	for _, id := range m.Weapons {
		u.Weapons = append(u.Weapons, s.prepareWeapon(id))
	}
	for _, ic := range m.Items {
		it, _ := s.cfg.Catalog.Item(ic.Item)
		u.Items = append(u.Items, tactics.ItemStack{Item: core.ItemOf(it), Count: ic.Count})
	}
	effects := map[string]tactics.SkillEffect{"inspire": tactics.Inspire, "pavise": tactics.Pavise, "wrath": tactics.Wrath, "snipe": tactics.Snipe, "canto": tactics.Canto, "zoc": tactics.ZOC}
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

// prepareWeaponTypes は定義の武器の種類を tactics の種類に写す（DES-UI-13）。
var prepareWeaponTypes = map[string]tactics.WeaponType{"sword": tactics.Sword, "lance": tactics.Lance, "axe": tactics.Axe, "bow": tactics.Bow, "staff": tactics.Staff, "magic": tactics.Magic}

// prepareWeapon は定義の武器 1 本を情報表示用の値にする。杖の回数は満タン（DES-ITM-02、DES-UI-32）。
func (s *SceneController) prepareWeapon(id string) tactics.Weapon {
	w, _ := s.cfg.Catalog.Weapon(id)
	tw := tactics.Weapon{ID: w.ID, Name: w.Name, Type: prepareWeaponTypes[w.Type], Crit: w.Crit, Weight: w.Weight, Strikes: w.Strikes, EffectiveTags: append([]string(nil), w.EffectiveTags...), Might: w.Might, Hit: w.Hit, MinRange: w.MinRange, MaxRange: w.MaxRange, Heal: w.Heal, Uses: w.Uses, MaxUses: w.Uses}
	if w.DamageType == "magic" {
		tw.DamageType = tactics.Magical
	}
	return tw
}

// prepareInfo は戦闘の情報欄と同じ部品で能力・装備を表示する（DES-UI-13）。
// 名簿のタブでは右列（DES-UI-32）に置くため、情報欄の右上 (MapSize, 0) の配置を右列の左上へずらす。
func (s *SceneController) prepareInfo() InfoView {
	u := s.prepareUnit(s.prepareMember)
	v := (Panel{Unit: &u, Page: s.infoPage, Tabs: true}).View()
	v = v.translate(prepareRightArea.Min.X-MapSize, prepareRightArea.Min.Y)
	// 長い名前などで右列を越す文は縮める（DES-UI-32）。
	for i, t := range v.Texts {
		if !t.Big {
			v.Texts[i].Text = fitText(t.Text, prepareRightArea.Max.X-t.X)
		}
	}
	return v
}

// translate は情報欄に描くものの位置をすべて (dx, dy) だけずらした写しを返す（DES-UI-32）。
func (v InfoView) translate(dx, dy int) InfoView {
	d := image.Pt(dx, dy)
	out := InfoView{Bottom: v.Bottom + dy}
	if v.Banner != nil {
		b := *v.Banner
		b.Rect = b.Rect.Add(d)
		out.Banner = &b
	}
	for _, t := range v.Texts {
		t.X, t.Y = t.X+dx, t.Y+dy
		out.Texts = append(out.Texts, t)
	}
	for _, b := range v.Bars {
		b.X, b.Y = b.X+dx, b.Y+dy
		out.Bars = append(out.Bars, b)
	}
	for _, ic := range v.Icons {
		ic.X, ic.Y = ic.X+dx, ic.Y+dy
		out.Icons = append(out.Icons, ic)
	}
	for _, tab := range v.Tabs {
		tab.Rect = tab.Rect.Add(d)
		out.Tabs = append(out.Tabs, tab)
	}
	return out
}

func itemCount(items []defs.ItemCount) int {
	n := 0
	for _, ic := range items {
		n += ic.Count
	}
	return n
}

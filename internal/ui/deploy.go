package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// DeployView は戦闘を作らずに表示する地形・初期敵・出撃枠と仮配置（DES-UI-20）。
type DeployView struct {
	Terrain  [][]tactics.Terrain
	Units    []tactics.Unit
	Slots    []tactics.Pos
	Selected int
}

// enterDeploy は出撃者の写しと連番の枠から配置を始める（DES-UI-13、DES-UI-20）。
func (s *SceneController) enterDeploy() {
	s.deployMembers = s.SelectedMembers()
	s.deploySlots = make([]int, len(s.deployMembers))
	for i := range s.deploySlots {
		s.deploySlots[i] = i
	}
	s.deploySelected = -1
	s.message = ""
	s.scene = Deploy
}

// leaveDeploy は手持ちと出撃の印を残し、仮配置だけを捨てる（DES-UI-20）。
func (s *SceneController) leaveDeploy() {
	s.deployMembers = nil
	s.deploySlots = nil
	s.deploySelected = -1
	s.message = ""
	s.scene = Prepare
}

// deployMap は現在の章の定義を返す（DES-UI-20）。
func (s *SceneController) deployMap() defs.MapDef {
	ch := s.cfg.Catalog.Chapters()[s.campaign.Chapter]
	m, _ := s.cfg.Catalog.Map(ch.Map)
	return m
}

// clickDeploy は members の順を保って枠だけを交換・移動する（DES-UI-20）。
func (s *SceneController) clickDeploy(p tactics.Pos) {
	slot := -1
	for i, v := range s.deployMap().Slots {
		if p == (tactics.Pos{X: v.X, Y: v.Y}) {
			slot = i
			break
		}
	}
	if slot < 0 {
		s.message = "出撃のマスを選んでください"
		return
	}
	s.message = ""
	occupant := -1
	for i, v := range s.deploySlots {
		if v == slot {
			occupant = i
			break
		}
	}
	if s.deploySelected < 0 {
		s.deploySelected = occupant
		return
	}
	selected := s.deploySelected
	if occupant != selected {
		if occupant >= 0 {
			s.deploySlots[occupant] = s.deploySlots[selected]
		}
		s.deploySlots[selected] = slot
	}
	s.deploySelected = -1
}

// chooseDeploy は配置の確定か準備への戻りを受け付ける（DES-UI-20）。
func (s *SceneController) chooseDeploy(i int) {
	switch i {
	case 0:
		s.startDeployedBattle()
	case 1:
		s.leaveDeploy()
	}
}

// deployView は目録と名簿から配置の表示を組み立て、乱数も番の処理も進めない（DES-UI-20）。
func (s *SceneController) deployView() SceneView {
	m := s.deployMap()
	d := &DeployView{Selected: s.deploySelected}
	terrain := map[rune]tactics.Terrain{}
	for _, t := range s.cfg.Catalog.Terrains() {
		for _, symbol := range t.Symbol {
			terrain[symbol] = tactics.Terrain{Name: t.Name, Cost: t.Cost, Avoid: t.Avoid, Def: t.Def, HealPercent: t.HealPercent}
		}
	}
	for _, row := range m.Rows {
		var line []tactics.Terrain
		for _, symbol := range row {
			line = append(line, terrain[symbol])
		}
		d.Terrain = append(d.Terrain, line)
	}
	for _, slot := range m.Slots {
		d.Slots = append(d.Slots, tactics.Pos{X: slot.X, Y: slot.Y})
	}
	for i, member := range s.deployMembers {
		u := s.prepareUnit(member)
		// 開始失敗の理由を出す際も表示を組み立てられるよう、不正な枠は描画しない。
		if slot := s.deploySlots[i]; slot >= 0 && slot < len(d.Slots) {
			u.Pos = d.Slots[slot]
		} else {
			u.Alive = false
		}
		d.Units = append(d.Units, u)
	}
	for _, e := range m.Enemies {
		cls, _ := s.cfg.Catalog.Class(e.Class)
		w, _ := s.cfg.Catalog.Weapon(e.Weapon)
		wt := map[string]tactics.WeaponType{"sword": tactics.Sword, "lance": tactics.Lance, "axe": tactics.Axe, "bow": tactics.Bow, "staff": tactics.Staff}[string(w.Type)]
		d.Units = append(d.Units, tactics.Unit{Side: tactics.Enemy, Alive: true, Boss: e.Boss, HP: cls.Base.HP + e.Bonus.HP, Stats: tactics.Stats{HP: cls.Base.HP + e.Bonus.HP}, Pos: tactics.Pos{X: e.X, Y: e.Y}, Weapon: tactics.Weapon{Type: wt}})
	}
	v := SceneView{Deploy: d, Buttons: []SceneButton{
		{"戦闘開始", image.Rect(528, 432, 816, 468), true},
		{"準備に戻る", image.Rect(528, 476, 816, 512), true},
	}, Texts: []SceneText{
		{Text: "左クリック: 味方を選ぶ・交換・空き出撃マスへ移す", X: 16, Y: 516, Style: StyleLabel},
		{Text: "右クリック・Esc: 選択を解除／未選択なら準備に戻る", X: 16, Y: 534, Style: StyleLabel},
	}}
	if s.deploySelected >= 0 {
		info := (Panel{Unit: &d.Units[s.deploySelected]}).View()
		v.Info = &info
	} else {
		v.Texts = append(v.Texts, SceneText{Text: "配置する味方を選んでください", X: 528, Y: 24, Style: StyleLabel})
	}
	return v
}

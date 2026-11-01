package ui

import (
	"image"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// deployButtonTop は配置のボタンの上端（DES-UI-20、DES-UI-28）。ページの印（y = 336〜）の上に置く。
const deployButtonTop = infoTabTop - 40

// DeployView は戦闘を作らずに表示する地形・初期敵・出撃枠と仮配置（DES-UI-20、DES-UI-23）。
type DeployView struct {
	Terrain           [][]tactics.Terrain
	Units             []tactics.Unit
	Slots             []tactics.Pos
	Selected          int
	Blue, Red, Danger map[tactics.Pos]bool
}

// enterDeploy は出撃者の写しと連番の枠から配置を始める（DES-UI-13、DES-UI-20）。
func (s *SceneController) enterDeploy() {
	s.resetDeploy()
	s.deployMembers = s.SelectedMembers()
	s.deploySlots = make([]int, len(s.deployMembers))
	for i := range s.deploySlots {
		s.deploySlots[i] = i
	}
	s.deploySelected = -1
	s.message = ""
	s.scene = Deploy
	b, err := s.campaign.DeploymentView(s.deployMembers, s.deploySlots)
	if err == nil {
		s.deployCamera = Camera{Width: b.Width(), Height: b.Height()}
		s.deployCamera.focusHero(b.Units())
	}
}

// leaveDeploy は手持ちと出撃の印を残し、仮配置だけを捨てる（DES-UI-20）。
func (s *SceneController) leaveDeploy() {
	s.resetDeploy()
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

// resetDeploy は仮配置と範囲の設定を捨てる（DES-UI-23）。
func (s *SceneController) resetDeploy() {
	s.deployCamera = Camera{}
	s.deployMembers = nil
	s.deploySlots = nil
	s.deploySelected = -1
	s.deployCursor = tactics.Pos{X: -1, Y: -1}
	s.deployLastCell, s.deployLastOK, s.deployOnInfo = tactics.Pos{}, false, false
	s.deployDanger = false
}

// SetDeployCursor は配置中のホバー位置だけを変え、味方の選択は保つ（DES-UI-23）。
func (s *SceneController) SetDeployCursor(x, y int) {
	if s.scene != Deploy {
		return
	}
	p, ok := s.deployCamera.CellAt(x, y)
	if !ok {
		p = tactics.Pos{X: -1, Y: -1}
	} else {
		s.deployLastCell, s.deployLastOK = p, true
	}
	s.deployCursor = p
	s.deployOnInfo = image.Pt(x, y).In(image.Rect(MapSize, 0, ScreenWidth, InfoHeight))
}

// deployInfoUnit は、配置の情報欄に出すユニットの添字を units から返す（DES-UI-27、DES-IT06 12.4 節）。なければ -1。
// 優先の順は「カーソルの下の生きているユニット（初めの敵も含む）→ 選んでいる味方」。カーソルが情報欄の上にあって
// 味方を選んでいなければ、ページの印をクリックできるよう、直前に盤面で見ていたマスのユニットを出し続ける。表示だけを変える。
func (s *SceneController) deployInfoUnit(units []tactics.Unit, selected int) int {
	at := func(p tactics.Pos) int {
		for i, u := range units {
			if u.Alive && u.Pos == p {
				return i
			}
		}
		return -1
	}
	if i := at(s.deployCursor); i >= 0 {
		return i
	}
	if selected >= 0 {
		return selected
	}
	if s.deployOnInfo && s.deployLastOK {
		return at(s.deployLastCell)
	}
	return -1
}

// ToggleDeployDanger は配置だけの Q の押し始めを受ける（DES-UI-23）。
func (s *SceneController) ToggleDeployDanger() {
	if s.scene == Deploy {
		s.deployDanger = !s.deployDanger
	}
}

// deployView は共通の配置用盤面から表示し、取得失敗では古い盤面・範囲を残さない（DES-DEP-02、DES-UI-23）。
func (s *SceneController) deployView() SceneView {
	d := &DeployView{Selected: -1}
	for _, slot := range s.deployMap().Slots {
		d.Slots = append(d.Slots, tactics.Pos{X: slot.X, Y: slot.Y})
	}
	board, err := s.campaign.DeploymentView(s.deployMembers, s.deploySlots)
	if err != nil {
		s.message = err.Error()
	} else {
		d.Selected = s.deploySelected
		d.Units = board.Units()
		for y := 0; y < board.Height(); y++ {
			row := make([]tactics.Terrain, board.Width())
			for x := range row {
				row[x] = board.TerrainAt(tactics.Pos{X: x, Y: y})
			}
			d.Terrain = append(d.Terrain, row)
		}
		for i, u := range d.Units {
			if !u.Alive || u.Side != tactics.Enemy {
				continue
			}
			hovered := u.Pos == s.deployCursor
			if !hovered && !s.deployDanger {
				continue
			}
			reach := board.Reachable(i)
			if hovered {
				d.Blue = map[tactics.Pos]bool{}
				d.Red = map[tactics.Pos]bool{}
				for p := range reach {
					d.Blue[p] = true
				}
			}
			if s.deployDanger && d.Danger == nil {
				d.Danger = map[tactics.Pos]bool{}
			}
			for p := range reach {
				for _, q := range board.AttackRange(i, p) {
					if hovered && !d.Blue[q] {
						d.Red[q] = true
					}
					if s.deployDanger {
						d.Danger[q] = true
					}
				}
			}
		}
	}
	// ボタンは情報欄（y < 360）の中、ページの印の上に横に並べ、右下のミニマップと重ねない（DES-UI-28）。
	v := SceneView{Deploy: d, Buttons: []SceneButton{
		{"戦闘開始", image.Rect(528, deployButtonTop, 668, deployButtonTop+32), true},
		{"準備に戻る", image.Rect(676, deployButtonTop, 816, deployButtonTop+32), true},
	}, Texts: []SceneText{
		{Text: "左クリック: 味方を選ぶ・交換・空き出撃マスへ移す", X: 16, Y: 516, Style: StyleLabel},
		{Text: "右クリック・Esc: 選択を解除／未選択なら準備に戻る", X: 16, Y: 534, Style: StyleLabel},
	}}
	mm := s.deployMinimap(d)
	v.Minimap = &mm
	if i := s.deployInfoUnit(d.Units, d.Selected); i >= 0 {
		// 敵の能力も、味方と同じ 2 ページで出す（DES-UI-27、REV-071）。まだ出ていない増援は配置の盤面にいない。
		info := (Panel{Unit: &d.Units[i], Page: s.infoPage, Tabs: true}).View()
		v.Info = &info
	} else {
		v.Texts = append(v.Texts, SceneText{Text: "配置する味方を選んでください", X: 528, Y: 24, Style: StyleLabel})
	}
	state := "OFF"
	if s.deployDanger {
		state = "ON"
	}
	v.Texts = append(v.Texts, SceneText{Text: cameraGuide, X: 528, Y: 520, Style: StyleLabel})
	v.Texts = append(v.Texts, SceneText{Text: "Q: 敵の危険範囲 " + state, X: 16, Y: 552, Style: StyleLabel})
	return v
}

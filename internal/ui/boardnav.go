package ui

import (
	"image"
	"image/color"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 盤面の移動のボタンの位置（DES-UI-30、DES-IT07 3.3 節）。
// 全体図はミニマップ欄の (744,472)〜(824,504)（minimapTexts の y=380・396 の案内より下）、
// 次の味方は下欄の (704,520)〜(824,544)（気配の案内 y=544〜560 と cameraGuideY の行より上）に置く。
// 位置は盤面の大きさに従う（DES-NG-10。recomputeMetrics が作る）。
var (
	overviewButtonRect image.Rectangle
	nextAllyButtonRect image.Rectangle
)

const (
	overviewButtonLabel = "全体図"
	nextAllyButtonLabel = "次の味方 Z"
)

// battleNav は戦闘の全体図と次の味方の、UI だけの状態（DES-UI-30）。戦闘の状態・乱数・保存には含めない。
// overview は全体図（MapOverview の重ね表示）を開いていること。入力の状態（Idle・Selected）は変えずに保ち、閉じるとそのまま戻る。
// ally は Idle の Z で最後にカメラを移した味方の添字（allyOK が偽ならまだない）。
type battleNav struct {
	overview bool
	ally     int
	allyOK   bool
}

// deployNav は配置の全体図と次の味方の、UI だけの状態（DES-UI-30）。仮配置と出撃の印には含めない。
type deployNav struct {
	overview bool
	ally     int
	allyOK   bool
}

// overviewLayout は、幅 w・高さ h のマップを盤面の (0,0)〜(512,512) に置いた全体図の配置（DES-UI-30）。
// 1 マスは長い辺で 512 px に収まる整数の大きさ（32×32 なら 16 px、16×16 なら 32 px）とし、中央に寄せる。
func overviewLayout(w, h int) MinimapLayout {
	l := MinimapLayout{Width: w, Height: h}
	if w <= 0 || h <= 0 {
		return l
	}
	l.Cell = max(1, MapSize/max(w, h))
	x := (MapSize - w*l.Cell) / 2
	y := (MapSize - h*l.Cell) / 2
	l.Area = image.Rect(x, y, x+w*l.Cell, y+h*l.Cell)
	return l
}

// OverviewCell は全体図の地形の 1 マス（DES-UI-30）。盤面と同じ地形の絵を縮めて描く。
type OverviewCell struct {
	Rect    image.Rectangle
	Terrain tactics.Terrain
}

// OverviewUnit は全体図の生きているユニットの 1 マス（DES-UI-30）。陣営の色（行動済みの自軍は暗く）の下地にユニットの絵を描く。
// Equip はマスの右下の 6×6 の装備の種類の色の範囲、EquipColor はその色（DES-UI-33、DES-IT07 3.6 節）。
// 武器のないユニットは Equip が空で、描かない。
type OverviewUnit struct {
	Rect       image.Rectangle
	Unit       tactics.Unit
	Equip      image.Rectangle
	EquipColor color.NRGBA
}

// overviewEquipSize は全体図の装備の種類の色の大きさ（6×6。DES-IT07 3.6 節）。
const overviewEquipSize = 6

// OverviewWeaponLegend は凡例の装備の種類の色の 1 行（DES-UI-33）。
type OverviewWeaponLegend struct {
	Swatch image.Rectangle
	Color  color.NRGBA
	Text   SceneText
}

// overviewWeaponTypes は凡例に並べる武器の種類の順（DES-UI-33）。
var overviewWeaponTypes = []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe, tactics.Bow, tactics.Magic, tactics.Staff}

// weaponTypeColor は全体図の装備の種類の色（DES-UI-33）。盤面の陣営の色（青・赤）と見分けられる色にする。
// 盤面の印と同じく、絵のある種類（weaponTile）だけに色を付ける。
func weaponTypeColor(t tactics.WeaponType) (color.NRGBA, bool) {
	switch t {
	case tactics.Sword:
		return color.NRGBA{245, 225, 80, 255}, true
	case tactics.Lance:
		return color.NRGBA{110, 215, 255, 255}, true
	case tactics.Axe:
		return color.NRGBA{120, 225, 90, 255}, true
	case tactics.Bow:
		return color.NRGBA{255, 150, 40, 255}, true
	case tactics.Magic:
		return color.NRGBA{205, 125, 255, 255}, true
	case tactics.Staff:
		return color.NRGBA{250, 250, 250, 255}, true
	}
	return color.NRGBA{}, false
}

// overviewEquip は、全体図のマス cell に置く装備の種類の色の範囲と色を返す（DES-UI-33）。
// 装備している武器（Equipped）の種類 1 つだけで、盤面の印（equipMarks）と同じく、武器のないユニットには付けない。
func overviewEquip(u tactics.Unit, cell image.Rectangle) (image.Rectangle, color.NRGBA) {
	if u.Equipped < 0 || u.Equipped >= len(u.Weapons) {
		return image.Rectangle{}, color.NRGBA{}
	}
	if _, ok := weaponTile(u.Weapons[u.Equipped].Type); !ok {
		return image.Rectangle{}, color.NRGBA{}
	}
	clr, ok := weaponTypeColor(u.Weapons[u.Equipped].Type)
	if !ok {
		return image.Rectangle{}, color.NRGBA{}
	}
	return image.Rect(cell.Max.X-overviewEquipSize, cell.Max.Y-overviewEquipSize, cell.Max.X, cell.Max.Y), clr
}

// MapOverviewView は全体図に描くもの（DES-UI-30）。描画から分け、Ebitengine なしで確かめられるようにする。
// 盤面と同じ元（地形・ユニット・分かっている予兆・全敵の危険範囲・固定した敵）を使い、ミニマップと同じ順に重ねる。
// Panel は右上の凡例の欄、Legend と Texts はその中に描く。
type MapOverviewView struct {
	Layout MinimapLayout
	Cells  []OverviewCell
	Units  []OverviewUnit
	Hints  []image.Rectangle
	Danger []image.Rectangle
	Pins   []image.Rectangle
	View   image.Rectangle
	Panel  image.Rectangle
	Legend []MinimapLegend
	// Weapons は凡例の装備の種類の色（DES-UI-33）。
	Weapons []OverviewWeaponLegend
	Texts   []SceneText
}

// 全体図の凡例で足す印（DES-UI-30）。ミニマップの印（MinimapMark）の続きの値にする。
const (
	MarkActed MinimapMark = MarkHint + 1 + iota
	MarkDanger
	MarkPin
)

// overviewTexts は右上の欄の見出し・操作の案内・凡例（DES-UI-30）。
func overviewTexts() ([]MinimapLegend, []SceneText) {
	x := MapSize + infoMargin
	texts := []SceneText{
		{Text: "全体図", X: x, Y: 4, Style: StyleTitle},
		{Text: "左クリック: その場所へ移って閉じる", X: x, Y: 4 + 2*lineHeight, Style: StyleLabel},
		{Text: "M・Esc・右クリック: 閉じる", X: x, Y: 4 + 3*lineHeight, Style: StyleLabel},
	}
	var legend []MinimapLegend
	for i, it := range []struct {
		mark MinimapMark
		text string
	}{{MarkPlayer, "自軍"}, {MarkActed, "行動済みの自軍"}, {MarkEnemy, "敵"}, {MarkView, "表示範囲"}, {MarkHint, "気配"}, {MarkDanger, "全敵の危険範囲"}, {MarkPin, "固定した敵"}} {
		y := 4 + 5*lineHeight + i*(lineHeight+2)
		legend = append(legend, MinimapLegend{
			Swatch: image.Rect(x, y+2, x+12, y+14),
			Mark:   it.mark,
			Text:   SceneText{Text: it.text, X: x + 18, Y: y, Style: StyleLabel},
		})
	}
	return legend, texts
}

// overviewWeaponLegend は、凡例の after 行目から装備の種類の色の行を並べる（DES-UI-33）。
func overviewWeaponLegend(after int) []OverviewWeaponLegend {
	x := MapSize + infoMargin
	var out []OverviewWeaponLegend
	for i, t := range overviewWeaponTypes {
		y := 4 + 5*lineHeight + (after+i)*(lineHeight+2)
		clr, _ := weaponTypeColor(t)
		out = append(out, OverviewWeaponLegend{
			Swatch: image.Rect(x+3, y+5, x+3+overviewEquipSize, y+5+overviewEquipSize),
			Color:  clr,
			Text:   SceneText{Text: "装備 " + weaponTypeName(t), X: x + 18, Y: y, Style: StyleLabel},
		})
	}
	return out
}

// buildOverview は盤面の見え方から全体図を作る（DES-UI-30）。倒れたユニットと盤の外のユニットは描かない。
// まだ出ていない増援は units に入っていないので描かれない。
func buildOverview(b terrainSource, units []tactics.Unit, hints []tactics.ReinforcementHint, danger map[tactics.Pos]bool, pins []tactics.Pos, cam Camera) MapOverviewView {
	v := MapOverviewView{Layout: overviewLayout(b.Width(), b.Height()), Panel: image.Rect(MapSize, 0, ScreenWidth, InfoHeight)}
	v.Legend, v.Texts = overviewTexts()
	v.Weapons = overviewWeaponLegend(len(v.Legend))
	l := v.Layout
	if l.Cell == 0 {
		return v
	}
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			p := tactics.Pos{X: x, Y: y}
			v.Cells = append(v.Cells, OverviewCell{Rect: l.CellRect(p), Terrain: b.TerrainAt(p)})
		}
	}
	for _, h := range hints {
		if r := l.spanRect(h.Min, h.Max); !r.Empty() {
			v.Hints = append(v.Hints, r)
		}
	}
	for _, u := range units {
		if !u.Alive || u.Pos.X < 0 || u.Pos.Y < 0 || u.Pos.X >= l.Width || u.Pos.Y >= l.Height {
			continue
		}
		r := l.CellRect(u.Pos)
		eq, clr := overviewEquip(u, r)
		v.Units = append(v.Units, OverviewUnit{Rect: r, Unit: u, Equip: eq, EquipColor: clr})
	}
	v.Danger = minimapDanger(l, danger)
	v.Pins = minimapPins(l, pins)
	last := tactics.Pos{X: min(l.Width, cam.X+16) - 1, Y: min(l.Height, cam.Y+16) - 1}
	v.View = l.spanRect(tactics.Pos{X: cam.X, Y: cam.Y}, last)
	return v
}

// nextIndex は、ref の次から名簿の順に巡って、ok を満たす最初の添字を返す（DES-UI-30）。ref 自身は最後に調べる。
// ref が負なら先頭から調べる。満たすものがなければ ok が偽。
func nextIndex(n, ref int, ok func(int) bool) (int, bool) {
	for k := 1; k <= n; k++ {
		i := ((ref+k)%n + n) % n
		if ok(i) {
			return i, true
		}
	}
	return -1, false
}

// nextUnactedAlly は、ref の次の未行動の生きている自軍の添字を名簿の順に返す（DES-UI-30）。全員行動済みなら ok が偽。
func nextUnactedAlly(units []tactics.Unit, ref int) (int, bool) {
	return nextIndex(len(units), ref, func(i int) bool {
		u := units[i]
		return u.Side == tactics.Player && u.Alive && !u.Acted
	})
}

// navAccepts は、全体図を開く・次の味方へ移る操作を受け付けるかを返す（DES-UI-30、DES-IT07 3.1・3.3 節）。
// `Idle`・`Selected` のときだけ受ける。行動のための `Moved`・`Targeting`・`Canto` など、再生中・勝敗の確定後・
// 巻き戻しの一覧・確認、番を終える確認の間は受けない（矢印・ホイールで盤面を動かせない間と同じ）。
func (c *Controller) navAccepts() bool {
	if c.rewind.open || c.confirmEnd || c.battle.NeedsResume() {
		return false
	}
	return c.state == Idle || c.state == Selected
}

// OverviewOpen は全体図を出しているかを返す（DES-UI-30）。巻き戻しの一覧・確認の間は一覧を優先して出さない
// （一覧を取り消すと、開く前の全体図に戻る。DES-IT07 3.1 節）。
func (c *Controller) OverviewOpen() bool { return c.nav.overview && !c.rewind.open }

// OpenOverview は M キーと「全体図」ボタンで全体図を開く（DES-UI-30）。受け付ける状態でだけ開き、開いたら真を返す。
// 入力の状態・選択・戦闘の状態は変えない。開いている間の M・ボタンは CloseOverview・OverviewClick が受ける。
func (c *Controller) OpenOverview() bool {
	if c.nav.overview || !c.navAccepts() {
		return false
	}
	c.nav.overview = true
	return true
}

// CloseOverview は Esc・右クリックで全体図を移動せずに閉じる（DES-UI-30）。閉じたら真を返す。
func (c *Controller) CloseOverview() bool {
	if !c.OverviewOpen() {
		return false
	}
	c.nav.overview = false
	return true
}

// OverviewClick は全体図を出しているときの左クリックを受ける（DES-UI-30）。全体図がすべてのクリックを取り、真を返す。
// 地図のマスのクリックはカメラの中心をそのマスへ移して閉じる。「全体図」ボタンは閉じる。ほかは何もしない。
// 移動・攻撃・選択・敵の範囲の設定には使わない。
func (c *Controller) OverviewClick(x, y int) bool {
	if !c.OverviewOpen() {
		return false
	}
	pt := image.Pt(x, y)
	if pt.In(overviewButtonRect) {
		c.nav.overview = false
		return true
	}
	if p, ok := overviewLayout(c.battle.Width(), c.battle.Height()).CellAt(x, y); ok {
		c.camera.CenterOn(p)
		c.nav.overview = false
	}
	return true
}

// NextAlly は Z キーと「次の味方 Z」ボタンの処理（DES-UI-30）。名簿の順に次の未行動の味方へカメラの中心を移す。
// `Selected` では選んでいる味方の次へ選択も移す。`Idle` では前に Z で移した味方の次から巡る。
// 全員行動済みなら何も変えない。処理したら真を返す。
func (c *Controller) NextAlly() bool {
	if !c.navAccepts() || c.nav.overview {
		return false
	}
	units := c.battle.Units()
	ref := -1
	if c.state == Selected {
		ref = c.unit
	} else if c.nav.allyOK {
		ref = c.nav.ally
	}
	i, ok := nextUnactedAlly(units, ref)
	if !ok {
		return false
	}
	c.nav.ally, c.nav.allyOK = i, true
	if c.state == Selected {
		c.unit = i
	}
	c.camera.CenterOn(units[i].Pos)
	return true
}

// MapOverview は戦闘の全体図を返す（DES-UI-30）。盤面に描いているものと同じ元を使う。
func (c *Controller) MapOverview() MapOverviewView {
	var hints []tactics.ReinforcementHint
	if h := c.ReinforcementHintView(); h.Visible {
		hints = h.Areas
	}
	return buildOverview(c.battle, c.Units(), hints, c.Danger(), c.WatchedPins(), c.camera)
}

// NavButtons は戦闘で描く「全体図」「次の味方 Z」のボタン（DES-UI-30）。再生中・勝敗の確定後・巻き戻しの一覧と確認の間は
// 描かない（巻き戻しの「決定」ボタンと重なるため。DES-IT07 3.3 節）。受け付けない状態では押せない形で描く。
func (c *Controller) NavButtons() []SceneButton {
	if c.rewind.open || c.state == Playing || c.state == Over {
		return nil
	}
	on := c.navAccepts()
	return []SceneButton{
		{overviewButtonLabel, overviewButtonRect, on || c.OverviewOpen()},
		{nextAllyButtonLabel, nextAllyButtonRect, on && !c.OverviewOpen()},
	}
}

// DeployOverviewOpen は配置で全体図を出しているかを返す（DES-UI-30）。
func (s *SceneController) DeployOverviewOpen() bool { return s.scene == Deploy && s.deployNav.overview }

// DeployOpenOverview は配置の M キーと「全体図」ボタンで全体図を開く（DES-UI-30）。開いたら真を返す。
func (s *SceneController) DeployOpenOverview() bool {
	if s.scene != Deploy || s.deployNav.overview {
		return false
	}
	s.deployNav.overview = true
	return true
}

// DeployCloseOverview は配置の Esc・右クリックで全体図を移動せずに閉じる（DES-UI-30）。閉じたら真を返す。
func (s *SceneController) DeployCloseOverview() bool {
	if !s.DeployOverviewOpen() {
		return false
	}
	s.deployNav.overview = false
	return true
}

// DeployOverviewClick は配置の全体図のクリックを受ける（DES-UI-30）。地図のマスのクリックは配置のカメラの中心を移して閉じる。
// 仮配置・選んでいる味方・出撃の印は変えない（DES-IT07 3.3 節）。全体図を出していれば真を返す。
func (s *SceneController) DeployOverviewClick(x, y int) bool {
	if !s.DeployOverviewOpen() {
		return false
	}
	if image.Pt(x, y).In(overviewButtonRect) {
		s.deployNav.overview = false
		return true
	}
	if p, ok := overviewLayout(s.deployCamera.Width, s.deployCamera.Height).CellAt(x, y); ok {
		s.deployCamera.CenterOn(p)
		s.deployNav.overview = false
	}
	return true
}

// DeployNextAlly は配置の Z キーと「次の味方 Z」ボタンの処理（DES-UI-30、DES-IT07 3.1 節）。
// 名簿の順に次の配置対象の味方へ選択とカメラの中心を移す。未配置の味方を優先し、全員配置済みなら出撃中の味方を巡る。
// 仮配置と出撃の印は変えない。処理したら真を返す。
func (s *SceneController) DeployNextAlly() bool {
	if s.scene != Deploy || s.deployNav.overview || len(s.deploySlots) == 0 {
		return false
	}
	ref := s.deploySelected
	if ref < 0 && s.deployNav.allyOK {
		ref = s.deployNav.ally
	}
	n := len(s.deploySlots)
	i, ok := nextIndex(n, ref, func(i int) bool { return s.deploySlots[i] < 0 })
	if !ok {
		i, ok = nextIndex(n, ref, func(i int) bool { return s.deploySlots[i] >= 0 })
	}
	if !ok {
		return false
	}
	s.deploySelected = i
	s.deployNav.ally, s.deployNav.allyOK = i, true
	if slots := s.deployMap().Slots; s.deploySlots[i] >= 0 && s.deploySlots[i] < len(slots) {
		slot := slots[s.deploySlots[i]]
		s.deployCamera.CenterOn(tactics.Pos{X: slot.X, Y: slot.Y})
	}
	return true
}

// DeployMapOverview は配置の全体図を返す（DES-UI-30）。配置の盤面（DeployView）と同じ地形・ユニット・全敵の危険範囲を使う。
// 配置ではまだ出ていない増援を見せないので予兆は描かず、固定は使わない。盤面を取れなければ ok が偽。
func (s *SceneController) DeployMapOverview() (MapOverviewView, bool) {
	d := s.deployView().Deploy
	if d == nil || len(d.Terrain) == 0 {
		return MapOverviewView{}, false
	}
	return buildOverview(gridTerrain(d.Terrain), d.Units, nil, d.Danger, nil, s.deployCamera), true
}

// DeployNavButtons は配置で描く「全体図」「次の味方 Z」のボタン（DES-UI-30）。全体図を出している間は次の味方を押せない。
func (s *SceneController) DeployNavButtons() []SceneButton {
	if s.scene != Deploy {
		return nil
	}
	return []SceneButton{
		{overviewButtonLabel, overviewButtonRect, true},
		{nextAllyButtonLabel, nextAllyButtonRect, !s.deployNav.overview},
	}
}

// navButtonAt は、描いているボタンのうち (x, y) を含むものの添字を返す（DES-UI-30）。なければ -1。
func navButtonAt(buttons []SceneButton, x, y int) int {
	for i, b := range buttons {
		if image.Pt(x, y).In(b.Rect) {
			return i
		}
	}
	return -1
}

// boardDrag は中ボタンで盤面をつかむ操作の途中の値（DES-UI-30）。動かした px を 32 px のマスの移動へ蓄積する。
type boardDrag struct {
	active       bool
	lastX, lastY int
	accX, accY   int
}

// update は 1 フレームの中ボタンの状態から、カメラを動かすマスの数を返す（DES-UI-30）。
// 盤面の 512×512 の中で押し始めたときだけつかみ、離すと終える。つかんだ盤面をカーソルと同じ向きに動かすので、
// カメラはカーソルの動きと逆へ動く。1 マスに満たない端数は次のフレームへ持ち越す。
func (d *boardDrag) update(pressed, start bool, x, y int) (dx, dy int) {
	if !pressed {
		*d = boardDrag{}
		return 0, 0
	}
	if !d.active {
		if start && x >= 0 && y >= 0 && x < MapSize && y < MapSize {
			*d = boardDrag{active: true, lastX: x, lastY: y}
		}
		return 0, 0
	}
	d.accX += d.lastX - x
	d.accY += d.lastY - y
	d.lastX, d.lastY = x, y
	dx, dy = d.accX/CellSize, d.accY/CellSize
	d.accX -= dx * CellSize
	d.accY -= dy * CellSize
	return dx, dy
}

// settle は、求めた移動 (wantX, wantY) のうちカメラが実際に動いた量 (gotX, gotY) と違う向きの蓄積を捨てる（DES-UI-30）。
// カメラの上限で止まったあとに引き続けた分を溜めず、戻す向きのドラッグがすぐ効くようにする。
func (d *boardDrag) settle(wantX, wantY, gotX, gotY int) {
	if gotX != wantX {
		d.accX = 0
	}
	if gotY != wantY {
		d.accY = 0
	}
}

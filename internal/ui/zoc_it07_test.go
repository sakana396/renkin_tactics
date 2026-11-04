package ui

import (
	"image"
	"reflect"
	"slices"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

var zocSkill = tactics.Skill{Name: "通せんぼ", Effect: tactics.ZOC}

// playerMark は、画面の左上 (sx, sy) のマスに付ける自軍の角印（左上の L 字、頂点はマスの角から 5 px、長さ 5 px）。
// 期待値は DES-IT07 3.8 節（自軍は青、敵軍は赤の細い角）から手で置いた座標で、zocMarkLines は使わない。
func playerMark(sx, sy int) []zocMarkLine {
	return []zocMarkLine{
		{From: image.Pt(sx+5, sy+5), To: image.Pt(sx+10, sy+5), Side: tactics.Player},
		{From: image.Pt(sx+5, sy+5), To: image.Pt(sx+5, sy+10), Side: tactics.Player},
	}
}

// enemyMark は、画面の左上 (sx, sy) のマスに付ける敵軍の角印（左下の L 字。頂点は (sx+5, sy+24) で、
// HP の棒（y = 27〜）の上。右下の装備の印（x = 22〜）と右上のピンを避ける）。
func enemyMark(sx, sy int) []zocMarkLine {
	return []zocMarkLine{
		{From: image.Pt(sx+5, sy+24), To: image.Pt(sx+10, sy+24), Side: tactics.Enemy},
		{From: image.Pt(sx+5, sy+19), To: image.Pt(sx+5, sy+24), Side: tactics.Enemy},
	}
}

// markRect は角印の線分を太さ 2 で囲む範囲を返す。
func markRect(l zocMarkLine) image.Rectangle {
	return image.Rect(min(l.From.X, l.To.X)-1, min(l.From.Y, l.To.Y)-1, max(l.From.X, l.To.X)+1, max(l.From.Y, l.To.Y)+1)
}

// joinMarks は角印の線分を順につなぐ。
func joinMarks(parts ...[]zocMarkLine) []zocMarkLine {
	var out []zocMarkLine
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// hasText は情報欄に s の行があるかを返す。
func hasText(v InfoView, s string) bool {
	for _, tx := range v.Texts {
		if tx.Text == s {
			return true
		}
	}
	return false
}

// zocBattleSetup は基本の盤面に、自軍の重騎士 (3,1) と敵軍の重騎士 (3,3)、壁 (4,1) を足す。
// 自軍の ZOC は (3,0)・(2,1)・(3,2)（(4,1) は壁なので付けない）、敵軍の ZOC は (3,2)・(2,3)・(4,3)・(3,4)。
func zocBattleSetup() tactics.Setup {
	s := basicSetup()
	s.Terrain[1][4] = tactics.Terrain{Name: "壁", Cost: 0}
	blue := tactics.UnitSetup{Name: "味方の重騎士", Side: tactics.Player, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 3, Y: 1}, Weapon: testLance, ClassSkills: []tactics.Skill{zocSkill}}
	red := tactics.UnitSetup{Name: "敵の重騎士", Side: tactics.Enemy, Level: 1, Stats: tactics.Stats{HP: 20}, Pos: tactics.Pos{X: 3, Y: 3}, Weapon: testLance, ClassSkills: []tactics.Skill{zocSkill}, AI: tactics.Hold}
	s.Units = append(s.Units, blue, red)
	return s
}

func TestZOCMarks(t *testing.T) {
	t.Run("TC-FR-UNT-09-29 隣の通れるマスに陣営色の角印を付け、範囲の上に描き、カーソルの下に移動費の説明を出す", func(t *testing.T) {
		c := newIdleWith(t, zocBattleSetup)
		b := c.Battle().BoardView()
		want := joinMarks(
			playerMark(96, 0),                     // (3,0)
			playerMark(64, 32),                    // (2,1)
			playerMark(96, 64), enemyMark(96, 64), // (3,2) は両軍
			enemyMark(64, 96), enemyMark(128, 96), // (2,3)・(4,3)
			enemyMark(96, 128), // (3,4)
		)
		if got := zocMarkLines(b, Camera{Width: 6, Height: 6}); !reflect.DeepEqual(got, want) {
			t.Errorf("角印\n got %v\nwant %v", got, want)
		}
		// カメラの外のマスは返さず、画面の座標はカメラの位置からずらす。
		if got, want := zocMarkLines(b, Camera{Y: 3, Width: 6, Height: 6}), joinMarks(enemyMark(64, 0), enemyMark(128, 0), enemyMark(96, 32)); !reflect.DeepEqual(got, want) {
			t.Errorf("カメラ (0,3) の角印\n got %v\nwant %v", got, want)
		}
		// 危険範囲を下、角印を上に描く（DES-IT07 3.8 節）。角印はユニットの上で、固定のピンより先。
		at := func(ls []boardLayer, l boardLayer) int { return slices.Index(ls, l) }
		if i, j := at(battleLayers, layerEnemyRanges), at(battleLayers, layerZOC); i < 0 || j < 0 || i > j || at(battleLayers, layerUnits) > j || at(battleLayers, layerEquipMarks) < j {
			t.Errorf("戦闘の層の順 %v", battleLayers)
		}
		// 角印はユニットのいるマスでも、装備の印・HP の棒・固定のピン（ボスの印の左を含む）と重ならない。
		pt := image.Pt(0, 0)
		head, _ := pinShape(pt, false)
		bossHead, _ := pinShape(pt, true)
		for _, l := range joinMarks(playerMark(0, 0), enemyMark(0, 0)) {
			r := markRect(l)
			for _, other := range []image.Rectangle{equipMarkRect(pt), unitHPBarRect(pt), head, bossHead} {
				if r.Overlaps(other) {
					t.Errorf("角印 %v が %v と重なる", r, other)
				}
			}
		}
		// カーソルの下の説明は、自軍の重騎士だけの隣・敵軍の重騎士だけの隣・両軍の隣で出し、どちらの隣でもないマスと壁では出さない。
		for _, tc := range []struct {
			p    tactics.Pos
			want bool
		}{
			{tactics.Pos{X: 2, Y: 1}, true},
			{tactics.Pos{X: 2, Y: 3}, true},
			{tactics.Pos{X: 3, Y: 2}, true},
			{tactics.Pos{X: 0, Y: 5}, false},
			{tactics.Pos{X: 4, Y: 1}, false},
		} {
			if got := hasText(c.Panel(tc.p, true).View(), zocNote); got != tc.want {
				t.Errorf("%v の説明 = %v, want %v", tc.p, got, tc.want)
			}
		}
		if got := skillDescription(zocSkill); got != "隣のマスに入る相手の移動費 +1" {
			t.Errorf("スキルの説明 = %q", got)
		}
	})

	t.Run("TC-FR-UNT-09-29 巻き戻しの下見では、下見側の盤面の生存者で角印を描く", func(t *testing.T) {
		// 弱い敵 (1,0) を重騎士にする。主人公が倒すと今の盤面には角印がなく、倒す前の下見にはある。
		k := newIdleWith(t, func() tactics.Setup {
			s := killSetup()
			s.Units[2].ClassSkills = []tactics.Skill{zocSkill}
			return s
		})
		cam := Camera{Width: 6, Height: 6}
		alive := joinMarks(enemyMark(0, 0), enemyMark(64, 0), enemyMark(32, 32)) // (0,0)・(2,0)・(1,1)
		if got := zocMarkLines(k.Battle().BoardView(), cam); !reflect.DeepEqual(got, alive) {
			t.Fatalf("前提: 倒す前の角印 %v", got)
		}
		k.Click(tactics.Pos{X: 0})
		k.Click(tactics.Pos{X: 0})
		k.ChooseAction(tactics.Attack)
		k.Click(tactics.Pos{X: 1})
		k.Click(tactics.Pos{X: 1})
		playOut(k)
		if k.Battle().Units()[2].Alive || k.State() != Idle {
			t.Fatalf("前提: 重騎士が倒れていない、状態 %v", k.State())
		}
		if got := zocMarkLines(k.Battle().BoardView(), cam); len(got) != 0 {
			t.Errorf("戦死した重騎士の角印が残る %v", got)
		}
		k.OpenRewind()
		k.RewindMove(-100)
		v := k.RewindBoard()
		if !v.Available {
			t.Fatal("下見の盤面がない")
		}
		if got := zocMarkLines(v.Board, k.Camera()); !reflect.DeepEqual(got, alive) {
			t.Errorf("下見の角印\n got %v\nwant %v", got, alive)
		}
		// 下見は、ユニット → 角印 → 行動したユニットの枠の順に描く。
		at := func(l boardLayer) int { return slices.Index(rewindLayers, l) }
		if at(layerUnits) < 0 || at(layerUnits) > at(layerZOC) || at(layerZOC) > at(layerFrame) {
			t.Errorf("下見の層の順 %v", rewindLayers)
		}
	})
}

// deployZOCScenes は 7 × 2 の配置の画面を作る。主人公 (0,0)・自軍の重騎士 (0,1)、敵軍の重騎士 (4,0)、ボス (6,1)。
func deployZOCScenes(t *testing.T) *SceneController {
	t.Helper()
	cat := defs.NewCatalog(defs.GameDef{Party: []string{"hero", "armor"}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{{ID: "lord", Name: "ロード", WeaponTypes: []string{"sword"}, Base: defs.Stats{HP: 20, Mov: 2}}, {ID: "armor", Name: "アーマー", WeaponTypes: []string{"sword"}, Base: defs.Stats{HP: 24, Mov: 2}, Skills: []string{"zoc"}}, {ID: "boss", Name: "騎兵", WeaponTypes: []string{"sword"}, Base: defs.Stats{HP: 30}}},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", MinRange: 1, MaxRange: 1}}, nil,
		[]defs.SkillDef{{ID: "zoc", Name: "通せんぼ", Effect: defs.EffectZOC}},
		[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "lord", Weapons: []string{"s"}, Lord: true, Level: 1}, {ID: "armor", Name: "重騎士", Class: "armor", Weapons: []string{"s"}, Level: 1}},
		[]defs.MapDef{{ID: "m", Rows: []string{".......", "......."}, Slots: []defs.Slot{{X: 0}, {X: 0, Y: 1}}, Enemies: []defs.EnemyPlacement{{Level: 1, Class: "armor", Weapons: []string{"s"}, AI: "hold", X: 4}, {Level: 1, Class: "boss", Weapons: []string{"s"}, AI: "hold", Boss: true, X: 6, Y: 1}}}},
		[]defs.ChapterDef{{ID: "c", Map: "m"}}, nil)
	c, err := core.NewCampaign(cat, core.Withdraw)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSceneController(Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { return 71 }})
	s.campaign = c
	s.enterPrepare()
	s.enterDeploy()
	return s
}

func TestZOCDeploy(t *testing.T) {
	t.Run("TC-FR-CMP-11-13 配置でも ZOC の角印を範囲の上に描き、入れ替えごとに求め直し、カーソルの下の移動費の説明を案内欄に出す", func(t *testing.T) {
		s := deployZOCScenes(t)
		marks := func() []zocMarkLine { return zocMarkLines(s.View().Deploy.Board, s.deployCamera) }
		// 自軍の重騎士 (0,1) の隣は (0,0)・(1,1)、敵軍の重騎士 (4,0) の隣は (3,0)・(5,0)・(4,1)。
		want := joinMarks(playerMark(0, 0), enemyMark(96, 0), enemyMark(160, 0), playerMark(32, 32), enemyMark(128, 32))
		if got := marks(); !reflect.DeepEqual(got, want) {
			t.Fatalf("配置の角印\n got %v\nwant %v", got, want)
		}
		note := func() (SceneText, bool) {
			for _, tx := range s.View().Texts {
				if tx.Text == zocNote {
					return tx, true
				}
			}
			return SceneText{}, false
		}
		for _, tc := range []struct {
			p    tactics.Pos
			want bool
		}{
			{tactics.Pos{X: 3}, true},
			{tactics.Pos{X: 1, Y: 1}, true},
			{tactics.Pos{X: 2}, false},
		} {
			s.SetDeployCursor(cellCenter(tc.p))
			if _, ok := note(); ok != tc.want {
				t.Errorf("カーソル %v の説明 = %v, want %v", tc.p, ok, tc.want)
			}
		}
		// 説明は案内欄の Q の状態の右に置き、Q の状態と重ならず、盤面の幅（512）に収まる。
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 3}))
		s.ToggleDeployDanger()
		tx, _ := note()
		r := rectOfText(tx.X, tx.Y, tx.Text, false)
		for _, other := range s.View().Texts {
			if other.Text == "Q: 敵の危険範囲 ON" && r.Overlaps(rectOfText(other.X, other.Y, other.Text, false)) {
				t.Errorf("説明 %v が Q の状態と重なる", r)
			}
		}
		if tx.Y != 552 || r.Min.X < 0 || r.Max.X > MapSize {
			t.Errorf("説明の位置 %v", r)
		}
		// 全敵の塗りを下、角印を上、選んだ味方の枠を最後に描く。
		at := func(l boardLayer) int { return slices.Index(deployLayers, l) }
		if at(layerEnemyRanges) < 0 || at(layerEnemyRanges) > at(layerZOC) || at(layerZOC) > at(layerFrame) {
			t.Errorf("配置の層の順 %v", deployLayers)
		}
		// 主人公と重騎士を入れ替えると、重騎士 (0,0) の隣 (1,0)・(0,1) に付け直す。
		clickDeployCell(s, tactics.Pos{X: 0, Y: 1})
		clickDeployCell(s, tactics.Pos{X: 0})
		want = joinMarks(playerMark(32, 0), enemyMark(96, 0), enemyMark(160, 0), playerMark(0, 32), enemyMark(128, 32))
		if got := marks(); !reflect.DeepEqual(got, want) {
			t.Errorf("入れ替えのあとの角印\n got %v\nwant %v", got, want)
		}
	})
}

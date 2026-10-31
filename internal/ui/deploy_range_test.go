package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// deployRangeScenes は占有で敵の移動先が変わる一本道（DES-UI-23、DES-DEP-02）。
func deployRangeScenes(t *testing.T) *SceneController {
	t.Helper()
	cat := defs.NewCatalog(defs.GameDef{Party: []string{"hero", "friend"}},
		[]defs.TerrainDef{{ID: "plain", Name: "平地", Symbol: ".", Cost: 1}},
		[]defs.ClassDef{{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 20, Mov: 2}}, {ID: "bow", Name: "弓兵", WeaponType: "bow", Base: defs.Stats{HP: 22, Str: 4, Skl: 6, Spd: 8, Def: 10, Mov: 2}}, {ID: "boss", Name: "騎兵", WeaponType: "sword", Base: defs.Stats{HP: 30}}},
		[]defs.WeaponDef{{ID: "s", Name: "剣", Type: "sword", MinRange: 1, MaxRange: 1}, {ID: "b", Name: "弓", Type: "bow", MinRange: 2, MaxRange: 2}}, nil, nil,
		[]defs.CharacterDef{{ID: "hero", Name: "主人公", Class: "lord", Weapon: "s", Lord: true, Level: 1}, {ID: "friend", Name: "仲間", Class: "lord", Weapon: "s", Level: 1}},
		[]defs.MapDef{{ID: "m", Rows: []string{"......."}, Slots: []defs.Slot{{X: 2}, {X: 0}, {X: 1}}, Enemies: []defs.EnemyPlacement{{Level: 1, Class: "bow", Weapon: "b", AI: "hold", X: 4}, {Level: 1, Class: "boss", Weapon: "s", AI: "hold", Boss: true, X: 6}}, Reinforcements: []defs.Reinforcement{{Turn: 3, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "boss", Weapon: "s", AI: "hold", X: 5}}}}},
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

func TestDeployRange(t *testing.T) {
	t.Run("TC-FR-CMP-11-10 ホバー・Q・占有変更・失敗・初期化と種の取得", func(t *testing.T) {
		s := deployRangeScenes(t)
		seeds := 0
		s.cfg.NextSeed = func() uint64 { seeds++; return 71 }
		baseline := campaignBytes(t, s)
		check := func(blue, red, danger map[tactics.Pos]bool) {
			t.Helper()
			v := s.View()
			d := v.Deploy
			for _, tx := range v.Texts {
				if tx.X+textWidth(tx.Text) > ScreenWidth || tx.Y+lineHeight > ScreenHeight {
					t.Fatal("案内・理由が画面外", tx)
				}
			}
			if !reflect.DeepEqual(d.Blue, blue) || !reflect.DeepEqual(d.Red, red) || !reflect.DeepEqual(d.Danger, danger) {
				t.Fatalf("範囲 blue=%v red=%v danger=%v、期待 %v %v %v", d.Blue, d.Red, d.Danger, blue, red, danger)
			}
			if seeds != 0 || s.battle != nil || s.sortie != nil || !bytes.Equal(baseline, campaignBytes(t, s)) {
				t.Fatal("閲覧で Campaign・種・戦闘が変化", seeds)
			}
		}
		check(nil, nil, nil)
		if !strings.Contains(viewStrings(s.View()), "Q: 敵の危険範囲 OFF") {
			t.Fatal("初期案内")
		}
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		blue, red, danger := rowSet(3, 4, 5), rowSet(1, 2, 6), rowSet(1, 2, 3, 5, 6)
		check(blue, red, nil)
		s.ToggleDeployDanger()
		check(blue, red, danger)
		if !strings.Contains(viewStrings(s.View()), "Q: 敵の危険範囲 ON") {
			t.Fatal("ON の案内")
		}
		clickDeployCell(s, tactics.Pos{X: 2})
		selected := s.deploySelected
		clickDeployCell(s, tactics.Pos{X: 4})
		if s.deploySelected != selected || !reflect.DeepEqual(s.deploySlots, []int{0, 1}) || s.message == "" {
			t.Fatal("敵クリックが配置を変更")
		}
		check(blue, red, danger)
		for _, xy := range [][2]int{{600, 40}, {-1, 0}, {32, 32}, {16, 16}} {
			s.SetDeployCursor(xy[0], xy[1])
			check(nil, nil, danger)
		}
		if s.deploySelected != selected {
			t.Fatal("ホバーで選択変更")
		}
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		s.ToggleDeployDanger()
		check(blue, red, nil)
		s.ToggleDeployDanger()
		// 交換は占有集合を保つので範囲も不変。空き枠へ動かすと (2,0) が開く。
		clickDeployCell(s, tactics.Pos{X: 0})
		check(blue, red, danger)
		if !reflect.DeepEqual(s.deploySlots, []int{1, 0}) {
			t.Fatal("交換")
		}
		clickDeployCell(s, tactics.Pos{X: 2})
		clickDeployCell(s, tactics.Pos{X: 1})
		blue, red, danger = rowSet(2, 3, 4, 5), rowSet(0, 1, 6), rowSet(0, 1, 2, 3, 4, 5, 6)
		check(blue, red, danger)
		clickDeployCell(s, tactics.Pos{X: 1})
		s.Cancel()
		check(blue, red, danger)
		// 取得に失敗しても配置と表示設定は残し、古い盤面・範囲は消す。
		clickDeployCell(s, tactics.Pos{X: 1})
		s.deploySlots[0] = 99
		invalid := append([]int(nil), s.deploySlots...)
		check(nil, nil, nil)
		if s.message == "" || !reflect.DeepEqual(s.deploySlots, invalid) || s.deploySelected != 1 || s.View().Info != nil || len(s.View().Deploy.Units) != 0 || len(s.View().Deploy.Terrain) != 0 {
			t.Fatal("取得失敗の理由・配置")
		}
		clickSceneButton(t, s, "戦闘開始")
		check(nil, nil, nil)
		s.deploySlots[0] = 1
		check(blue, red, danger)
		s.Cancel()
		check(blue, red, danger)
		s.Cancel()
		if s.Scene() != Prepare || s.deployDanger || s.deployCursor != (tactics.Pos{X: -1, Y: -1}) {
			t.Fatal("未選択の取消しと設定破棄")
		}
		s.ToggleDeployDanger()
		s.SetDeployCursor(128, 0)
		if s.deployDanger || s.deployCursor != (tactics.Pos{X: -1, Y: -1}) {
			t.Fatal("準備で範囲の入力を受けた")
		}
		s.enterDeploy()
		check(nil, nil, nil)
		if s.deployDanger || s.deployCursor != (tactics.Pos{X: -1, Y: -1}) {
			t.Fatal("再入場で設定を初期化しない")
		}
		s.ToggleDeployDanger()
		clickSceneButton(t, s, "準備に戻る")
		if s.Scene() != Prepare || s.deployDanger {
			t.Fatal("戻るボタンで設定を破棄しない")
		}
		s.enterDeploy()
		check(nil, nil, nil)
		// 敵は HP 以外も core の値を表示し、未出現の増援は含めない。
		d := s.View().Deploy
		if len(d.Units) != 4 || d.Units[2].Stats != (tactics.Stats{HP: 22, Str: 4, Skl: 6, Spd: 8, Def: 10, Mov: 2}) || d.Units[2].Weapon.MinRange != 2 || d.Units[2].AI != tactics.Hold || d.Units[3].ClassName != "騎兵" {
			t.Fatal("完全な初期敵でない", d.Units)
		}
		s.ToggleDeployDanger()
		s.SetDeployCursor(cellCenter(tactics.Pos{X: 4}))
		clickSceneButton(t, s, "戦闘開始")
		if seeds != 1 || s.Scene() != Battle || s.battle.danger || s.deployDanger || s.deployMembers != nil || s.deploySlots != nil {
			t.Fatal("実開始の一度の種取得・設定破棄", seeds)
		}
		s.Choose(0)
		if seeds != 1 {
			t.Fatal("二重開始")
		}
	})
}

package ui

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

func clickDeployCell(s *SceneController, p tactics.Pos) { s.Click(p.X*CellSize+16, p.Y*CellSize+16) }
func campaignBytes(t *testing.T, s *SceneController) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := s.campaign.Save(&b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDeploy(t *testing.T) {
	t.Run("TC-FR-CMP-11-05 配置の交換と空き枠・取消し・遷移", func(t *testing.T) {
		for _, count := range []int{1, 4, 5} {
			t.Run(fmt.Sprintf("%d人", count), func(t *testing.T) {
				s := prepareScenes(t)
				// 準備で変更した手持ちと、主人公が途中の名簿も保つ。
				clickSceneButton(t, s, "在庫へ: 回復薬")
				s.campaign.Roster[0], s.campaign.Roster[2] = s.campaign.Roster[2], s.campaign.Roster[0]
				s.enterPrepare()
				members := []int{2, 0, 1, 3, 4}[:count]
				s.deployment = map[int]bool{}
				for _, m := range members {
					s.deployment[m] = true
				}
				baseline := campaignBytes(t, s)
				clickSceneButton(t, s, "配置へ")
				if s.Scene() != Deploy || s.battle != nil || s.sortie != nil || !reflect.DeepEqual(s.deployMembers, members) {
					t.Fatal("配置へ入れない", s.Scene())
				}
				view := s.View()
				if view.Deploy == nil || len(view.Deploy.Slots) != 6 || len(view.Deploy.Units) != count+len(s.deployMap().Enemies) || len(view.Deploy.Terrain) != 16 {
					t.Fatal("マップ・敵・枠がない", view.Deploy)
				}
				board, err := s.campaign.DeploymentView(s.deployMembers, s.deploySlots)
				if err != nil || !reflect.DeepEqual(view.Deploy.Units, board.Units()) {
					t.Fatal("配置が共通の完全なユニット表示と違う", err)
				}
				for y, row := range view.Deploy.Terrain {
					for x, terrain := range row {
						if terrain != board.TerrainAt(tactics.Pos{X: x, Y: y}) {
							t.Fatal("地形が共通の表示と違う", x, y)
						}
					}
				}
				slots := view.Deploy.Slots
				for i := 0; i < count; i++ {
					if view.Deploy.Units[i].Pos != slots[i] {
						t.Fatal("初期配置", i)
					}
				}
				clickDeployCell(s, slots[0])
				if s.deploySelected != 0 || s.View().Deploy.Selected != 0 || s.View().Info == nil {
					t.Fatal("主人公を選べない")
				}
				clickDeployCell(s, slots[0])
				if s.deploySelected != -1 {
					t.Fatal("本人で解除できない")
				}
				clickDeployCell(s, slots[0])
				clickDeployCell(s, slots[4])
				if s.deploySlots[0] != 4 || !reflect.DeepEqual(s.deployMembers, members) {
					t.Fatal("主人公の移動・交換", s.deploySlots)
				}
				if count == 5 && s.deploySlots[4] != 0 {
					t.Fatal("交換されない")
				}
				clickDeployCell(s, slots[4])
				before := append([]int(nil), s.deploySlots...)
				invalidCells := []tactics.Pos{{X: 15, Y: 0}}
				for _, enemy := range s.deployMap().Enemies {
					invalidCells = append(invalidCells, tactics.Pos{X: enemy.X, Y: enemy.Y})
				}
				for y, row := range view.Deploy.Terrain {
					for x, terrain := range row {
						if terrain.Cost == 0 {
							invalidCells = append(invalidCells, tactics.Pos{X: x, Y: y})
							break
						}
					}
				}
				for _, p := range invalidCells {
					clickDeployCell(s, p)
					if !reflect.DeepEqual(s.deploySlots, before) || !strings.Contains(viewStrings(s.View()), "出撃のマスを選んでください") {
						t.Fatal("枠外を拒否しない")
					}
				}
				s.Cancel()
				if s.Scene() != Deploy || s.deploySelected != -1 {
					t.Fatal("選択解除で戻った")
				}
				if !bytes.Equal(campaignBytes(t, s), baseline) {
					t.Fatal("配置で Campaign が変化")
				}
				// 未選択の取消しでも、ボタンでも準備に戻り、選択と手持ちは保つ。
				s.Cancel()
				if s.Scene() != Prepare || !reflect.DeepEqual(s.SelectedMembers(), members) || !bytes.Equal(campaignBytes(t, s), baseline) {
					t.Fatal("準備の状態を失った")
				}
				clickSceneButton(t, s, "配置へ")
				for i, slot := range s.deploySlots {
					if i != slot {
						t.Fatal("配置が初期化されない")
					}
				}
				clickDeployCell(s, slots[0])
				clickSceneButton(t, s, "準備に戻る")
				if s.Scene() != Prepare {
					t.Fatal("ボタンで戻れない")
				}
				clickSceneButton(t, s, "配置へ")
				// core の開始失敗では、仮配置とキャンペーンを保って理由を表示する。
				// ch01 の枠は 6 つ（添字 0〜5）なので、添字 6 は範囲外（DES-IT06 13 章）。
				s.deploySlots[0] = len(slots)
				invalid := append([]int(nil), s.deploySlots...)
				clickSceneButton(t, s, "戦闘開始")
				if s.Scene() != Deploy || s.message == "" || !reflect.DeepEqual(s.deploySlots, invalid) || !bytes.Equal(campaignBytes(t, s), baseline) {
					t.Fatal("失敗で配置が変わる")
				}
				s.deploySlots[0] = 0
				clickDeployCell(s, slots[0])
				clickDeployCell(s, slots[4])
				clickSceneButton(t, s, "戦闘開始")
				if s.Scene() != Battle || !reflect.DeepEqual(s.sortie.Members, members) || s.sortie.Battle.Units()[0].Pos != slots[4] {
					t.Fatal("開始の配置", s.Scene())
				}
				s.Cancel()
				s.Choose(1)
				if s.Scene() != Battle {
					t.Fatal("開始後に準備へ戻った")
				}
			})
		}
	})
	t.Run("TC-FR-CMP-11-06 戦闘開始だけが種を取得し初回の番を一度再生", func(t *testing.T) {
		s := prepareScenes(t)
		seeds := 0
		s.cfg.NextSeed = func() uint64 { seeds++; return 71 }
		clickSceneButton(t, s, "配置へ")
		slots := s.View().Deploy.Slots
		clickDeployCell(s, slots[0])
		clickDeployCell(s, slots[1])
		s.Cancel()
		clickSceneButton(t, s, "配置へ")
		s.ToggleDeployDanger()
		s.SetDeployCursor(cellCenter(tactics.Pos{X: s.deployMap().Enemies[0].X, Y: s.deployMap().Enemies[0].Y}))
		_ = s.View()
		_ = s.View()
		if seeds != 0 || s.battle != nil || s.sortie != nil {
			t.Fatal("配置中に戦闘を作った", seeds)
		}
		clickSceneButton(t, s, "戦闘開始")
		if seeds != 1 || s.battle.State() != Playing || s.sortie.Battle.Turn != 1 || len(s.sortie.Battle.History()) != 0 {
			t.Fatal("開始回数", seeds)
		}
		s.Choose(0)
		s.Advance()
		if seeds != 1 {
			t.Fatal("二重開始", seeds)
		}
		if len(s.battle.events) != 1 || s.battle.events[0].Kind != tactics.EventPhaseStart || s.battle.events[0].Turn != 1 || s.battle.events[0].Side != tactics.Player {
			t.Fatal("初回の番が一度でない", s.battle.events)
		}
		for _, u := range s.sortie.Battle.Units() {
			if u.HP != u.Stats.HP {
				t.Fatal("初回のHP", u)
			}
		}
		playOut(s.battle)
		if s.battle.State() != Idle || s.sortie.Battle.Turn != 1 {
			t.Fatal("初回の番", s.battle.State())
		}
	})
}

func TestDeployLayout(t *testing.T) {
	t.Run("TC-FR-UNT-09-12 配置の情報・ボタン・案内が画面に収まる", func(t *testing.T) {
		s := prepareScenes(t)
		clickSceneButton(t, s, "配置へ")
		// ch01 は名簿 5 人・枠 6 つで、6 つ目の枠は空く（DES-IT06 13 章）。味方のいる枠を順に選ぶ。
		slots := s.View().Deploy.Slots
		if len(slots) != 6 || len(s.deployMembers) != 5 {
			t.Fatal("ch01 の枠と出撃の人数", len(slots), len(s.deployMembers))
		}
		for _, p := range slots[:len(s.deployMembers)] {
			clickDeployCell(s, p)
			s.ToggleDeployDanger()
			// Q と枠外の理由も、既存の案内・情報・ボタンと重ねない（DES-UI-23）。
			clickDeployCell(s, tactics.Pos{X: 15, Y: 0})
			v := s.View()
			if v.Info == nil || v.Info.Bottom > v.Buttons[0].Rect.Min.Y {
				t.Fatal("情報とボタンが重なる", v.Info)
			}
			for _, tx := range v.Info.Texts {
				if tx.X < MapSize || tx.X+textWidth(tx.Text) > ScreenWidth || tx.Y+lineHeight > 432 {
					t.Fatal("情報が欄外", tx)
				}
			}
			for i, tx := range v.Texts {
				if tx.X < 0 || tx.X+textWidth(tx.Text) > ScreenWidth || tx.Y+lineHeight > ScreenHeight {
					t.Fatal("案内が画面外", tx)
				}
				r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
				for _, other := range v.Texts[i+1:] {
					if r.Overlaps(image.Rect(other.X, other.Y, other.X+textWidth(other.Text), other.Y+lineHeight)) {
						t.Fatal("案内同士が重なる", tx, other)
					}
				}
				for _, b := range v.Buttons {
					if r.Overlaps(b.Rect) {
						t.Fatal("案内とボタンが重なる", tx)
					}
				}
			}
			for _, b := range v.Buttons {
				if !b.Rect.In(image.Rect(MapSize, 0, ScreenWidth, MapSize)) || textWidth(b.Label)+24 > b.Rect.Dx() {
					t.Fatal("ボタンが欄外", b)
				}
			}
			s.Cancel()
		}
	})
}

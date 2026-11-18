package ui

import (
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui/audio"
)

// REV-110 の指摘への対応の試験（DES-NG-11。TC-DES-NG-11-20〜25）。

// 新作の定義データ（data/alchemy）の武器・道具の ID。
var (
	alchemyWeaponIDs = []string{"iron_sword", "iron_lance", "iron_axe", "iron_bow", "fire", "heal_staff", "relic_lance"}
	alchemyItemIDs   = []string{"potion", "bomb_small", "bomb_large", "bomb_far"}
)

// handDetailTexts は手持ちの欄の下部（性能の表示の位置）に出ている文を、上から順に返す。
func handDetailTexts(v PrepareHiResView) []hrTextItem {
	var out []hrTextItem
	for _, tx := range v.Texts {
		if tx.X == hrDetailX && tx.Y >= hrDetailY-1 && tx.Y < hrHandPanel.Max.Y {
			out = append(out, tx)
		}
	}
	return out
}

// clickStockRow は在庫の行 id を左クリックして選ぶ（実際のクリックの経路）。
func clickStockRow(t *testing.T, s *SceneController, id string) {
	t.Helper()
	// 行が見えるまで、在庫を 1 行ずつ送る。
	for top := 0; top < 10; top++ {
		s.prepareStockTop = top
		for _, r := range s.prepareHiResView().Stock {
			if r.Item.id == id {
				s.Click(r.Rect.Min.X+5, r.Rect.Min.Y+5)
				return
			}
		}
	}
	t.Fatalf("在庫の行 %s が見えない", id)
}

func TestPrepareHiResReview(t *testing.T) {
	t.Run("TC-DES-NG-11-20 選んだ品（なければ指している品）の性能を、手持ちの欄の下部に出す", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.campaign.StockWeapons = append(s.campaign.StockWeapons, core.WeaponCount{Weapon: "iron_axe", Count: 1})
		s.prepareMember = 1 // ダイン（重装兵。斧は装備できない）。手持ちは鉄の槍・回復薬
		texts := func() []string {
			var out []string
			for _, tx := range handDetailTexts(s.prepareHiResView()) {
				out = append(out, tx.Text)
			}
			return out
		}
		// 何も選ばず・指さずなら、今の操作の説明のまま。
		if got := texts(); !reflect.DeepEqual(got, []string{"在庫の行を選び、空き枠をクリックで渡す", "枠をクリックで、在庫に戻す", "E: 装備（右クリックで武器を装備にする）"}) {
			t.Fatal(got)
		}
		// 在庫の武器を選ぶ: 名前（全文）、性能（今の準備の weaponDetail の文面。名前を除く）、装備できない理由（赤）。
		clickStockRow(t, s, "iron_axe")
		const stats = "物理　威力 8　命中 70　必殺 0　重さ 7　射程 1　特効 なし"
		if got := strings.Join(wrapSceneTextWidth(stats, hrDetailWidth), "|"); got == "" || len(wrapSceneTextWidth(stats, hrDetailWidth)) != 2 {
			t.Fatal("性能の文は 2 行に折り返される想定:", got)
		}
		lines := wrapSceneTextWidth(stats, hrDetailWidth)
		want := append([]string{"選んだ品: 鉄の斧"}, lines...)
		want = append(want, "ダイン（重装兵）は斧を装備できない")
		if got := texts(); !reflect.DeepEqual(got, want) {
			t.Fatalf("\n got %q\nwant %q", got, want)
		}
		if w := s.prepareWeapon("iron_axe"); strings.Join(lines, "") != strings.TrimPrefix(weaponDetail(w), w.Name+"　") {
			t.Fatal("今の準備の weaponDetail と文面が違う")
		}
		hd := handDetailTexts(s.prepareHiResView())
		if hd[len(hd)-1].Color != hrBad || hd[0].Color != hrAqua {
			t.Fatal("理由は赤、選んだ品の行はアクア")
		}
		// 装備できる武器（アオイの魔法）なら理由の行はない。
		s.prepareMember = 0
		s.prepareItem = prepareItem{}
		s.campaign.StockWeapons = append(s.campaign.StockWeapons, core.WeaponCount{Weapon: "fire", Count: 1})
		clickStockRow(t, s, "fire")
		for _, tx := range handDetailTexts(s.prepareHiResView()) {
			if tx.Color == hrBad {
				t.Fatalf("装備できる武器に理由が出ている: %q", tx.Text)
			}
		}
		// 在庫の道具: 爆弾の数値と回復量（今の準備の itemDetailText と同じ文面）。
		s.prepareItem = prepareItem{}
		clickStockRow(t, s, "bomb_small")
		if got, want := texts(), []string{"選んだ品: 小さな爆弾", "道具　爆弾 ダメージ 8　範囲 1　投げる 4"}; !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
		s.prepareItem = prepareItem{}
		clickStockRow(t, s, "potion")
		if got, want := texts(), []string{"選んだ品: 回復薬", "道具　HP を 10 回復"}; !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
		// 選んでいなければ、指している手持ちの枠・在庫の行の品（名前は「選んだ品:」なし）。選んでいればそちらが先。
		s.prepareItem = prepareItem{}
		s.prepareMember = 1
		v := s.prepareHiResView()
		s.prepareCursor = v.Hand[1].Rect.Min.Add(image.Pt(5, 5)) // ダインの回復薬
		if got, want := texts(), []string{"回復薬", "道具　HP を 10 回復"}; !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
		s.prepareMember = 0
		v = s.prepareHiResView()
		s.prepareCursor = v.Hand[3].Rect.Min.Add(image.Pt(5, 5)) // アオイの小さな爆弾（枠の下の名前は「小さな爆…」に切れる）
		if got := texts(); len(got) < 2 || got[0] != "小さな爆弾" || got[1] != "道具　爆弾 ダメージ 8　範囲 1　投げる 4" {
			t.Fatal(got)
		}
		s.prepareCursor = v.Hand[0].Rect.Min.Add(image.Pt(5, 5)) // アオイのファイア（武器の枠）
		if got := texts(); len(got) < 2 || got[0] != "ファイア" {
			t.Fatal(got)
		}
		s.prepareCursor = v.Stock[0].Rect.Min.Add(image.Pt(5, 5))
		if got := texts(); len(got) < 2 || got[0] != s.hrItemName(v.Stock[0].Item) {
			t.Fatal(got)
		}
		s.prepareItem = prepareItem{ok: true, id: "potion"}
		s.prepareCursor = v.Hand[3].Rect.Min.Add(image.Pt(5, 5))
		if got := texts(); got[0] != "選んだ品: 回復薬" {
			t.Fatal("選択が指しているものより先:", got)
		}
		// 空き枠・何もない所を指すと、操作の説明に戻る。
		s.prepareItem = prepareItem{}
		s.prepareMember = 1
		v = s.prepareHiResView()
		s.prepareCursor = v.Hand[4].Rect.Min.Add(image.Pt(5, 5))
		if got := texts(); got[0] != "在庫の行を選び、空き枠をクリックで渡す" {
			t.Fatal(got)
		}
		s.prepareCursor = image.Pt(-1, -1)
		if got := texts(); got[0] != "在庫の行を選び、空き枠をクリックで渡す" {
			t.Fatal(got)
		}
	})
	t.Run("TC-DES-NG-11-20 性能の文は、全武器・全道具で 2 行以内に収まり、手持ちの欄の中に収まる", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		cat, err := defload.Load(data.Alchemy())
		if err != nil {
			t.Fatal(err)
		}
		s := ng05Scenes(t, "ch01", true)
		var items []prepareItem
		for _, id := range alchemyWeaponIDs {
			if _, ok := cat.Weapon(id); !ok {
				t.Fatal("武器の定義がない:", id)
			}
			items = append(items, prepareItem{ok: true, weapon: true, id: id})
		}
		for _, id := range alchemyItemIDs {
			if _, ok := cat.Item(id); !ok {
				t.Fatal("道具の定義がない:", id)
			}
			items = append(items, prepareItem{ok: true, id: id})
		}
		if len(items) < 10 {
			t.Fatal(len(items))
		}
		for _, it := range items {
			s.prepareItem = it
			for _, m := range s.livingMembers() {
				s.prepareMember = m
				hd := handDetailTexts(s.prepareHiResView())
				if len(hd) < 2 || len(hd) > 4 {
					t.Errorf("%v: 行数 %d", it, len(hd))
				}
				for _, tx := range hd {
					r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
					if !r.In(hrHandPanel) {
						t.Errorf("%v: 文「%s」%v が手持ちの欄の外", it, tx.Text, r)
					}
				}
				if it.weapon {
					w := s.prepareWeapon(it.id)
					if n := len(wrapSceneTextWidth(strings.TrimPrefix(weaponDetail(w), w.Name+"　"), hrDetailWidth)); n > 2 {
						t.Errorf("%s の性能が %d 行に折り返され、後ろが落ちる", w.Name, n)
					}
				}
			}
		}
	})
	t.Run("TC-DES-NG-11-21 道具の丸は、爆弾の範囲（BlastRadius）で色と大きさが変わり、爆弾の ID を使わない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch03", true)
		cat := s.cfg.Catalog
		seen := map[color.NRGBA]string{}
		var bombs []hrDot
		var radii []int
		for _, id := range alchemyItemIDs {
			d, _ := cat.Item(id)
			dot := s.hrItemDot(d.ID, 0, 0, 13)
			if other, dup := seen[dot.Color]; dup {
				t.Errorf("%s と %s の丸の色が同じ", d.ID, other)
			}
			seen[dot.Color] = d.ID
			if d.Bomb {
				bombs = append(bombs, dot)
				radii = append(radii, d.BlastRadius)
			}
		}
		if len(bombs) < 3 {
			t.Fatal("新作の爆弾は 3 種ある:", len(bombs))
		}
		// 範囲が広いほど大きい（同じ範囲なら同じ大きさ）。範囲 2 以上は最大の半径。
		for i := range bombs {
			for j := range bombs {
				if radii[i] < radii[j] && bombs[i].R >= bombs[j].R {
					t.Errorf("範囲 %d の丸 %d が、範囲 %d の丸 %d より小さくない", radii[i], bombs[i].R, radii[j], bombs[j].R)
				}
			}
			if radii[i] >= 2 && bombs[i].R != 13 {
				t.Errorf("範囲 %d の丸の半径 %d", radii[i], bombs[i].R)
			}
		}
		// 色の手書きの期待値: 範囲 0 は橙、1 は灰、2 は暗い赤褐色、回復薬は赤。
		want := map[string]color.NRGBA{
			"bomb_far": {230, 140, 50, 255}, "bomb_small": {150, 150, 160, 255}, "bomb_large": {140, 64, 40, 255}, "potion": {200, 70, 60, 255},
		}
		for id, c := range want {
			if got := s.hrItemDot(id, 0, 0, 13).Color; got != c {
				t.Errorf("%s: %v", id, got)
			}
		}
		// 手持ちの絵でも、在庫の行でも使う。ch03 の準備には、3 種の爆弾の丸が並ぶ。
		v := s.prepareHiResView()
		colors := map[color.NRGBA]bool{}
		for _, d := range v.Dots {
			colors[d.Color] = true
		}
		if len(colors) < 3 {
			t.Fatal("見える丸の色が少ない:", len(colors))
		}
	})
	t.Run("TC-DES-NG-11-22 ▲▼ と → は vector の図形で描く（文字ではない）。クリックの範囲は変えない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch03", true)
		s.campaign.Roster = append(s.campaign.Roster, core.Member{Character: "haru", Level: 1, Weapons: []string{"fire"}, Stats: tactics.Stats{HP: 10}, Alive: true})
		v := s.prepareHiResView()
		for _, tx := range v.Texts {
			if strings.ContainsAny(tx.Text, "▲▼→") {
				t.Errorf("文字の字形が残っている: %q", tx.Text)
			}
		}
		// 範囲（旧と同じ値）。
		if v.StockUp.Rect != image.Rect(1000, 304, 1030, 330) || v.StockDown.Rect != image.Rect(1034, 304, 1064, 330) ||
			v.RosterUp.Rect != image.Rect(268, 104, 304, 130) || v.RosterDown.Rect != image.Rect(308, 104, 344, 130) {
			t.Fatal(v.StockUp, v.StockDown, v.RosterUp, v.RosterDown)
		}
		var up, down, arrow []hrGlyph
		for _, g := range v.Glyphs {
			switch g.Kind {
			case hrGlyphUp:
				up = append(up, g)
			case hrGlyphDown:
				down = append(down, g)
			case hrGlyphArrow:
				arrow = append(arrow, g)
			}
		}
		if len(up) != 2 || len(down) != 2 || len(arrow) != len(weaponCycle())-1 {
			t.Fatal(len(up), len(down), len(arrow))
		}
		// 三角はそれぞれのボタンの中。使えるボタンは暗い色、使えないボタンは薄い色。
		for _, b := range []struct {
			btn  SceneButton
			kind hrGlyphKind
		}{{v.RosterUp, hrGlyphUp}, {v.RosterDown, hrGlyphDown}, {v.StockUp, hrGlyphUp}, {v.StockDown, hrGlyphDown}} {
			found := false
			for _, g := range v.Glyphs {
				if g.Kind == b.kind && g.Rect.In(b.btn.Rect) {
					found = true
					c := hrDark
					if !b.btn.Enabled {
						c = hrDim
					}
					if g.Color != c {
						t.Errorf("%s %v の色 %v", b.btn.Label, b.btn.Rect, g.Color)
					}
				}
			}
			if !found {
				t.Errorf("%s %v に三角がない", b.btn.Label, b.btn.Rect)
			}
		}
		// 矢印は章の情報の早見の、箱と箱の間。
		for i, g := range arrow {
			if !g.Rect.In(hrChapterPanel) || g.Rect.Dx() < 12 || g.Rect.Dy() < 8 {
				t.Errorf("矢印 %d %v", i, g.Rect)
			}
		}
		// 描ける: 三角と矢印を、画像に描いても落ちない（画素は、ゲームが始まる前は読めないので、目で確かめる）。
		img := ebiten.NewImage(40, 40)
		for _, kind := range []hrGlyphKind{hrGlyphUp, hrGlyphDown, hrGlyphArrow} {
			hrDrawGlyph(img, hrGlyph{Kind: kind, Rect: image.Rect(10, 12, 26, 28), Color: hrAmber})
		}
		// ▼ をクリックすると送れる（範囲は変えていない）。
		s.Click(v.RosterDown.Rect.Min.X+3, v.RosterDown.Rect.Min.Y+3)
		if s.prepareRosterTop != 1 {
			t.Fatal(s.prepareRosterTop)
		}
	})
	t.Run("TC-DES-NG-11-23 名簿のカードの手持ちの使用数の箱は、使っている枠を琥珀で塗り、空きは暗いまま", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		v := s.prepareHiResView()
		for _, c := range v.Cards {
			m := s.campaign.Roster[c.Member]
			used := len(m.Weapons) + itemCount(m.Items)
			pips, slots := 0, 0
			for _, b := range v.Boxes {
				if !b.Rect.In(c.Rect) || b.Rect.Dx() != 12 || b.Rect.Dy() != 12 {
					continue
				}
				switch b.Kind {
				case hrBoxPip:
					pips++
				case hrBoxSlot:
					slots++
				}
			}
			if pips != used || slots != prepareHandSlots-used {
				t.Errorf("メンバー %d: 塗った箱 %d（使用 %d）、空きの箱 %d", c.Member, pips, used, slots)
			}
		}
		// 塗りの色: 使っている枠は琥珀、空きは暗い（琥珀でない）。描いても落ちない。
		if hrPipFill != hrAmber || hrSlotFill == hrAmber || hrSlotFill.R > 60 || hrSlotFill.G > 60 || hrSlotFill.B > 60 {
			t.Errorf("塗りの色 %v %v", hrPipFill, hrSlotFill)
		}
		img := ebiten.NewImage(20, 20)
		for _, k := range []hrBoxKind{hrBoxPip, hrBoxSlot} {
			hrDrawBox(img, hrBox{Rect: image.Rect(0, 0, 12, 12), Kind: k})
		}
	})
	t.Run("TC-DES-NG-11-24 準備の入力は Game.sceneInput の経路で動く（左クリック・右クリック・↑↓・Enter・Esc）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		g := &Game{scenes: s}
		click := func(r image.Rectangle) { g.sceneInput(sceneKeys{Left: true}, r.Min.X+5, r.Min.Y+5) }
		right := func(x, y int) { g.sceneInput(sceneKeys{Right: true, Cancel: true}, x, y) }
		// カード: 左クリックで選ぶ（マウスの位置が準備の値に渡る）。
		v := s.prepareHiResView()
		c1 := v.Cards[1]
		click(c1.Rect.Add(image.Pt(40, 0)))
		if s.prepareMember != c1.Member || s.prepareCursor != image.Pt(c1.Rect.Min.X+45, c1.Rect.Min.Y+5) {
			t.Fatal(s.prepareMember, s.prepareCursor)
		}
		// チェック: 出撃を外し、もう一度で戻す。
		v = s.prepareHiResView()
		click(v.Cards[1].Mark)
		if s.deployment[c1.Member] {
			t.Fatal("チェックで外れない")
		}
		click(s.prepareHiResView().Cards[1].Mark)
		if !s.deployment[c1.Member] {
			t.Fatal("チェックで戻らない")
		}
		// ↑↓: 選ぶキャラが動く。
		g.sceneInput(sceneKeys{Down: true}, 0, 0)
		if s.prepareMember != s.livingMembers()[2] {
			t.Fatal("↓", s.prepareMember)
		}
		g.sceneInput(sceneKeys{Up: true}, 0, 0)
		g.sceneInput(sceneKeys{Up: true}, 0, 0)
		if s.prepareMember != s.livingMembers()[0] {
			t.Fatal("↑", s.prepareMember)
		}
		// 在庫の行 → 手持ちの空き枠（ダイン）。
		s.prepareMember = 1
		before := len(s.campaign.Roster[1].Weapons) + itemCount(s.campaign.Roster[1].Items)
		v = s.prepareHiResView()
		row := v.Stock[len(v.Stock)-1]
		click(row.Rect)
		if s.prepareItem != row.Item {
			t.Fatal("在庫の行が選べない")
		}
		click(v.Hand[2].Rect)
		if got := len(s.campaign.Roster[1].Weapons) + itemCount(s.campaign.Roster[1].Items); got != before+1 {
			t.Fatal("空き枠で渡せない", got)
		}
		// 手持ちの枠: 左クリックで在庫に戻す。
		click(s.prepareHiResView().Hand[2].Rect)
		if got := len(s.campaign.Roster[1].Weapons) + itemCount(s.campaign.Roster[1].Items); got != before {
			t.Fatal("枠のクリックで戻せない", got)
		}
		// 右クリック: 手持ちの武器の枠の上なら装備の切り替え（選択は外さない）。それ以外は在庫の選択を外す。
		s.prepareMember = 0
		first, second := s.campaign.Roster[0].Weapons[0], s.campaign.Roster[0].Weapons[1]
		v = s.prepareHiResView()
		right(v.Hand[1].Rect.Min.X+5, v.Hand[1].Rect.Min.Y+5)
		if w := s.campaign.Roster[0].Weapons; w[0] != second || w[1] != first {
			t.Fatal("右クリックで装備が替わらない", w)
		}
		s.prepareItem = prepareItem{ok: true, id: "potion"}
		right(5, 5)
		if s.prepareItem.ok {
			t.Fatal("右クリックで選択が外れない")
		}
		s.prepareItem = prepareItem{ok: true, id: "potion"}
		g.sceneInput(sceneKeys{Cancel: true}, 5, 5)
		if s.prepareItem.ok {
			t.Fatal("Esc で選択が外れない")
		}
		// 配置へ: ボタンの左クリック、Enter。
		click(hrDeployRect)
		if s.Scene() != Deploy {
			t.Fatal(s.Scene())
		}
		s = ng05Scenes(t, "ch01", true)
		g = &Game{scenes: s}
		g.sceneInput(sceneKeys{Enter: true}, 0, 0)
		if s.Scene() != Deploy {
			t.Fatal("Enter で配置へ進まない", s.Scene())
		}
		// HiRes でない準備は、↑↓・Enter を受けない。
		s = ng05Scenes(t, "ch01", false)
		g = &Game{scenes: s}
		g.sceneInput(sceneKeys{Down: true}, 0, 0)
		g.sceneInput(sceneKeys{Enter: true}, 0, 0)
		if s.Scene() != Prepare || s.prepareMember != 0 {
			t.Fatal("今の準備が ↑↓・Enter に反応した")
		}
	})
	t.Run("TC-DES-NG-11-25 高解像度の準備の Enter（配置へ）は決定音、↑↓（キャラの選択）は選択音を鳴らす。今の準備・ほかの場面は変わらない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		play := func(scene Scene, hires bool, k soundKeys) []audio.SE {
			s := ng05Scenes(t, "ch01", hires)
			rig := newSoundRig(t, s)
			rig.g.syncSound(soundInputOf(scene, hires && scene == Prepare, k))
			return rig.names()
		}
		cases := []struct {
			name  string
			scene Scene
			hires bool
			k     soundKeys
			want  []audio.SE
		}{
			{"HiRes の準備の Enter は決定", Prepare, true, soundKeys{Enter: true}, []audio.SE{audio.SEConfirm}},
			{"HiRes の準備の ↑ は選択", Prepare, true, soundKeys{Up: true}, []audio.SE{audio.SESelect}},
			{"HiRes の準備の ↓ は選択", Prepare, true, soundKeys{Down: true}, []audio.SE{audio.SESelect}},
			{"HiRes の準備の左クリックは決定（今まで通り）", Prepare, true, soundKeys{Left: true}, []audio.SE{audio.SEConfirm}},
			{"HiRes の準備の Esc は取消し", Prepare, true, soundKeys{Esc: true}, []audio.SE{audio.SECancel}},
			{"Enter と Esc の同時は取消しだけ", Prepare, true, soundKeys{Enter: true, Esc: true}, []audio.SE{audio.SECancel}},
			{"今の準備の Enter は鳴らさない", Prepare, false, soundKeys{Enter: true}, nil},
			{"今の準備の ↑↓ は鳴らさない", Prepare, false, soundKeys{Up: true, Down: true}, nil},
			{"今の準備の左クリックは決定", Prepare, false, soundKeys{Left: true}, []audio.SE{audio.SEConfirm}},
		}
		for _, tc := range cases {
			if got := play(tc.scene, tc.hires, tc.k); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s: %v", tc.name, got)
			}
		}
		// 準備以外の場面の規則は変えない: 物語の Enter・Space は決定、ほかの場面の Enter・↑↓ は鳴らさない。
		if got := soundInputOf(Story, false, soundKeys{Enter: true}); !got.Confirm {
			t.Error("物語の Enter は決定")
		}
		if got := soundInputOf(Story, false, soundKeys{Space: true}); !got.Confirm {
			t.Error("物語の Space は決定")
		}
		for _, sc := range []Scene{Title, Deploy, Battle, Clear} {
			if got := soundInputOf(sc, false, soundKeys{Enter: true, Up: true, Down: true}); got != (soundInput{}) {
				t.Errorf("場面 %v: %+v", sc, got)
			}
		}
		if got := soundInputOf(Battle, false, soundKeys{Tab: true}); !got.Select {
			t.Error("Tab は選択音")
		}
	})
}

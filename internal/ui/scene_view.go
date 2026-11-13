package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
	"github.com/sakana396/renkin_tactics/internal/ui/assets"
)

// SceneButton は描画とクリック判定で共有するボタン（DES-UI-09、10）。
type SceneButton struct {
	Label   string
	Rect    image.Rectangle
	Enabled bool
}

// SceneText は各場面の日本語の文と位置（DES-UI-10〜12、15）。
type SceneText struct {
	Text  string
	X, Y  int
	Big   bool
	Style InfoStyle
}

// SceneIcon は加入した仲間の絵（DES-UI-12）。
type SceneIcon struct {
	Tile assets.Tile
	X, Y int
}

// SceneView は場面の表示を描画から独立して検証するための値（DES-UI-09）。
type SceneView struct {
	Texts   []SceneText
	Buttons []SceneButton
	Icons   []SceneIcon
	Rows    []PrepareRow
	Info    *InfoView
	// InfoArea は Info の下地を塗る範囲（DES-UI-32）。空なら右の情報欄の全体。
	InfoArea image.Rectangle
	// Tabs は準備の「名簿」「持ち物」のタブ（DES-UI-32）。ほかの場面では空。
	Tabs   []PrepareTab
	Deploy *DeployView
	// Minimap は配置のミニマップ（DES-UI-28）。ほかの場面では nil。
	Minimap *MinimapView
	// Backdrop は背景の画像の道筋（media の下。DES-NG-08）。画像がなければ空で、今までの見た目のまま。
	Backdrop string
	// Portrait は物語の場面の肖像（DES-NG-08）。話者がいない・絵がなければ nil。
	Portrait *ScenePortrait
}

// wrapSceneText は 48 px の左右余白を残し、文字の単位で折り返す（DES-UI-12）。
func wrapSceneText(s string) []string {
	return wrapSceneTextWidth(s, sceneCanvasWidth-96)
}

// wrapSceneTextWidth は場面の表示領域の幅で文字を折り返す（DES-UI-12、DES-UI-13）。
func wrapSceneTextWidth(s string, width int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		row := ""
		for _, r := range line {
			next := row + string(r)
			if textWidth(next) > width && row != "" {
				out = append(out, row)
				row = string(r)
			} else {
				row = next
			}
		}
		out = append(out, row)
	}
	return out
}

// View は描画とボタン判定に使う文と配置を組み立てる（DES-UI-09〜12、15、9 章 Issue 6）。
func (s *SceneController) View() SceneView {
	var v SceneView
	add := func(str string, x, y int, big bool, style InfoStyle) {
		v.Texts = append(v.Texts, SceneText{str, x, y, big, style})
	}
	lines := func(str string, y int, style InfoStyle) {
		for i, line := range wrapSceneText(str) {
			add(line, 48, y+i*20, false, style)
		}
	}
	button := func(label string, y int, enabled bool) {
		v.Buttons = append(v.Buttons, SceneButton{label, image.Rect(240, y, 592, y+36), enabled})
	}
	heading := func(ch int) {
		add(fmt.Sprintf("第 %d 章　%s", ch+1, s.cfg.Catalog.Chapters()[ch].Name), 48, 32, true, StyleTitle)
	}
	switch s.scene {
	case Title:
		v.Backdrop, _ = s.art.BackdropPath(artTitlePath, true)
		add(windowTitle, sceneCanvasWidth/2-textWidth(windowTitle), 48, true, StyleTitle)
		if s.confirmOverwrite {
			lines("セーブを上書きして、はじめから始めますか", 136, StylePlain)
			button("はい", 208, true)
			button("いいえ", 272, true)
		} else {
			button("新規開始", 160, true)
			button("続きから", 240, s.loaded != nil)
			button("終了", 336, true)
			lines(s.saveSummary(), 284, StyleLabel)
		}
	case RuleSelect:
		add("倒れた仲間をどうしますか", 48, 48, true, StyleTitle)
		button("離脱", 160, true)
		lines("倒れた仲間は、その章では戦えないが、次の章で戻る", 208, StylePlain)
		button("ロスト", 288, true)
		lines("倒れた仲間は戦死し、二度と戻らない", 336, StylePlain)
		lines("右クリック・Esc: タイトルへ", 400, StyleLabel)
	case Story:
		heading(s.storyChapter)
		// 物語の背景と、話者の肖像（DES-NG-08）。画像がなければ今までの配置のまま。
		ch := s.cfg.Catalog.Chapters()[s.storyChapter]
		storyID := ch.Before
		if s.storyAfter {
			storyID = ch.After
		}
		v.Backdrop, _ = s.art.BackdropPath(storyBackdropPath(storyID))
		textX, wrapWidth := 48, sceneCanvasWidth-96
		if p, ok := s.art.StoryPortraitPath(s.paragraphs[s.paragraph]); ok {
			var box image.Rectangle
			box, textX, wrapWidth = storyArtLayout(true)
			v.Portrait = &ScenePortrait{Path: p, Rect: box}
		}
		for i, line := range wrapSceneTextWidth(s.paragraphs[s.paragraph], wrapWidth) {
			add(line, textX, 160+i*20, false, StylePlain)
		}
		count := fmt.Sprintf("%d/%d", s.paragraph+1, len(s.paragraphs))
		add(count, sceneCanvasWidth-48-textWidth(count), 488, false, StyleLabel)
		lines("左クリック・Enter・Space: 次へ　右クリック・Esc: 残りを飛ばす", 528, StyleLabel)
	case Joined:
		heading(s.storyChapter)
		id := s.cfg.Catalog.Chapters()[s.storyChapter].Joins[s.joined]
		ch, _ := s.cfg.Catalog.Character(id)
		class, _ := s.cfg.Catalog.Class(ch.Class)
		for _, m := range s.campaign.Roster {
			if m.Character != id {
				continue
			}
			add(ch.Name+"が仲間に加わった", 48, 144, true, StyleTitle)
			var w defs.WeaponDef
			if len(m.Weapons) > 0 {
				w, _ = s.cfg.Catalog.Weapon(m.Weapons[0])
			}
			weaponType := tactics.Sword
			switch string(w.Type) {
			case "lance":
				weaponType = tactics.Lance
			case "axe":
				weaponType = tactics.Axe
			case "bow":
				weaponType = tactics.Bow
			case "staff":
				weaponType = tactics.Staff
			case "magic":
				weaponType = tactics.Magic
			}
			// 騎兵の絵を選べるよう、クラス名も渡す（DES-UI-22）。
			v.Icons = append(v.Icons, SceneIcon{unitTile(tactics.Unit{ClassName: class.Name, Side: tactics.Player, Weapons: []tactics.Weapon{{Type: weaponType}}}), 48, 208})
			lines(fmt.Sprintf("%s　Lv %d", class.Name, m.Level), 256, StylePlain)
			lines(fmt.Sprintf("HP %d　力 %d　技 %d　速さ %d　守備 %d　移動 %d", m.Stats.HP, m.Stats.Str, m.Stats.Skl, m.Stats.Spd, m.Stats.Def, m.Stats.Mov), 296, StylePlain)
		}
		lines("左クリック: 章の準備へ", 528, StyleLabel)
	case Prepare:
		v = s.prepareView()
	case Deploy:
		v = s.deployView()
	case Result:
		add(fmt.Sprintf("第 %d 章　%s　勝利", s.storyChapter+1, s.cfg.Catalog.Chapters()[s.storyChapter].Name), 48, 32, true, StyleTitle)
		for i, m := range s.result.Members {
			ch, _ := s.cfg.Catalog.Character(s.campaign.Roster[m.Member].Character)
			fallen := ""
			if m.Fallen {
				fallen = "　離脱（次の章で戻る）"
			}
			style := StylePlain
			if m.Dead {
				fallen = "　戦死"
				style = StyleBad
			}
			lines(fmt.Sprintf("%s　Lv %d → %d　経験 %d → %d%s", ch.Name, m.LevelBefore, m.LevelAfter, m.ExpBefore, m.ExpAfter, fallen), 144+i*48, style)
			if gains := growthText(m.Gains); gains != "" {
				lines(gains, 164+i*48, StyleLabel)
			}
		}
		var rewards []string
		for _, r := range s.result.Reward {
			if r.Weapon != "" {
				weapon, _ := s.cfg.Catalog.Weapon(r.Weapon)
				rewards = append(rewards, fmt.Sprintf("%s ×%d", weapon.Name, r.Count))
			} else {
				item, _ := s.cfg.Catalog.Item(r.Item)
				rewards = append(rewards, fmt.Sprintf("%s ×%d", item.Name, r.Count))
			}
		}
		if len(rewards) > 0 {
			lines(strings.Join(rewards, "　")+" を在庫に入れた", 444, StylePlain)
		}
		lines("左クリック: 章のあとの物語へ", 548, StyleLabel)
	case GameOver:
		add("敗北", 48, 48, true, StyleTitle)
		lines("主人公が倒れた", 112, StylePlain)
		button("この章をやり直す", 208, true)
		button("タイトルへ", 288, true)
		button("巻き戻す", 368, s.battle != nil)
	case Clear:
		add("クリア", 48, 48, true, StyleTitle)
		var dead []string
		n := 0
		for _, m := range s.campaign.Roster {
			ch, _ := s.cfg.Catalog.Character(m.Character)
			if m.Alive {
				lines(fmt.Sprintf("%s　Lv %d", ch.Name, m.Level), 128+n*40, StylePlain)
				n++
			} else {
				dead = append(dead, ch.Name)
			}
		}
		if len(dead) > 0 {
			lines("戦死: "+strings.Join(dead, "、"), 400, StyleBad)
		}
		lines("左クリック: タイトルへ", 528, StyleLabel)
	}
	if s.message != "" && s.scene != Prepare {
		y := 424
		if s.scene == Deploy {
			// 注意文は盤面の下の案内欄の 3 行目に置く。x は既定の 280 とし、文が窓の右端を越えるときは左へ寄せる（DES-NG-10）。
			x := max(16, min(280, ScreenWidth-16-textWidth(s.message)))
			add(s.message, x, MapSize+40, false, StyleBad)
		} else {
			lines(s.message, y, StyleBad)
		}
	}
	if s.saveWarning != "" {
		y := 88
		if s.scene == Result {
			y = 476
		}
		lines(s.saveWarning, y, StyleBad)
	}
	return v
}

// Click は描画と同じ範囲で場面のボタンを判定する（DES-UI-09）。
func (s *SceneController) Click(x, y int) {
	if (s.scene == Battle || s.scene == GameOver) && s.battle != nil && s.battle.RewindClick(x, y) {
		s.syncRewind()
		return
	}
	// 勝敗が決まった Over では、右欄の「詳細 I」の位置のクリックを先に受けて何もしない（Over は新しい操作を受けない。DES-IT07 3.1 節）。
	if s.scene == Battle && s.battle != nil && s.battle.State() == Over && s.battle.InfoClick(x, y) {
		return
	}
	if s.scene == Battle && !image.Pt(x, y).In(BattleEndRect()) {
		return
	}
	if s.scene == Deploy {
		// 入力の判定はミニマップ → 情報欄 → ボタン → 盤面の順（DES-UI-28）。
		if s.deployMinimapClick(x, y) || s.infoTabClick(x, y) {
			return
		}
		if image.Pt(x, y).In(deployDetailButtonRect) {
			s.OpenDeployDetail()
			return
		}
		for i, b := range s.View().Buttons {
			if image.Pt(x, y).In(b.Rect) && b.Enabled {
				s.Choose(i)
				return
			}
		}
		if p, ok := s.deployCamera.CellAt(x, y); ok {
			s.clickDeploy(p)
		}
		return
	}
	if s.scene == Prepare {
		s.clickPrepare(x, y)
		return
	}
	v := s.View()
	if len(v.Buttons) > 0 {
		for i, b := range v.Buttons {
			if image.Pt(x, y).In(b.Rect) && b.Enabled {
				s.Choose(i)
				return
			}
		}
		return
	}
	s.Advance()
}

// infoTabClick は準備と配置の情報欄のページの印のクリックを受ける（DES-UI-28）。印を出しているときだけ受け、表示だけを変える。
func (s *SceneController) infoTabClick(x, y int) bool {
	if s.scene != Prepare {
		return false
	}
	v := s.View()
	if v.Info == nil {
		return false
	}
	// 印の範囲は表示と同じものを使う。準備では右列へずらしてある（DES-UI-32）。
	for _, tab := range v.Info.Tabs {
		if image.Pt(x, y).In(tab.Rect) {
			s.infoPage = tab.Page
			return true
		}
	}
	return false
}

// ToggleInfoPage は準備と配置での Tab キーの処理（DES-UI-28）。表示だけを変える。
func (s *SceneController) ToggleInfoPage() {
	if s.scene == Prepare {
		s.infoPage = s.infoPage.next()
	}
}

package ui

import (
	"image"
	"reflect"
	"testing"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// ng05Scenes は新作の定義データで、章 chapter の準備まで進めた場面の進行を返す。hires が偽なら HiRes でない準備。
func ng05Scenes(t *testing.T, chapter string, hires bool) *SceneController {
	t.Helper()
	cat, err := defload.Load(data.Alchemy())
	if err != nil {
		t.Fatal(err)
	}
	s := NewSceneController(Config{Catalog: cat, Store: &memoryStore{}, NextSeed: func() uint64 { return 7 }, HiRes: hires, StartChapter: chapter})
	s.Choose(0)
	s.Choose(0)
	for i := 0; s.Scene() == Story || s.Scene() == Joined; i++ {
		if i >= 200 {
			t.Fatal("物語が終わらない")
		}
		s.Advance()
	}
	if s.Scene() != Prepare {
		t.Fatal(s.Scene())
	}
	return s
}

func TestPrepareHiResNG05(t *testing.T) {
	win := image.Rect(0, 0, 1088, 832)
	t.Run("TC-DES-NG-11-05 領域は窓 1088×832 に収まり、互いに重ならない", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		if ScreenWidth != 1088 || ScreenHeight != 832 {
			t.Fatal(ScreenWidth, ScreenHeight)
		}
		regions := []image.Rectangle{hrHeader, hrRosterPanel, hrCenterPanel, hrHandPanel, hrStockPanel, hrChapterPanel, hrFooter}
		for i, r := range regions {
			if !r.In(win) {
				t.Errorf("領域 %d %v が窓の外", i, r)
			}
			for j := i + 1; j < len(regions); j++ {
				if r.Overlaps(regions[j]) {
					t.Errorf("領域 %d %v と %d %v が重なる", i, r, j, regions[j])
				}
			}
		}
		// 設計メモ 2 章の座標（手で置いた期待値）。
		want := map[string]image.Rectangle{
			"roster": image.Rect(20, 100, 348, 740), "center": image.Rect(364, 100, 724, 740),
			"hand": image.Rect(740, 100, 1068, 290), "stock": image.Rect(740, 300, 1068, 430),
			"chapter": image.Rect(740, 440, 1068, 740), "deploy": image.Rect(850, 764, 1060, 818),
		}
		got := map[string]image.Rectangle{"roster": hrRosterPanel, "center": hrCenterPanel, "hand": hrHandPanel, "stock": hrStockPanel, "chapter": hrChapterPanel, "deploy": hrDeployRect}
		if !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
		if hrPortraitRect.Dx() != 336 || hrPortraitRect.Dy() != 336 || !hrPortraitRect.Inset(-2).In(hrCenterPanel) {
			t.Fatal(hrPortraitRect)
		}
	})
	t.Run("TC-DES-NG-11-06 カードの縦の並びは 5 枚まで、左の欄に収まり、6 人以上はスクロールする", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch02", true)
		v := s.prepareHiResView()
		if len(v.Cards) != 5 || v.RosterUp.Label != "" {
			t.Fatal("ch02 の名簿は 5 人でスクロールなし:", len(v.Cards), v.RosterUp)
		}
		// 5 枚のカードは、左の欄の中で縦に並び、重ならない。
		for i, c := range v.Cards {
			if !c.Rect.In(hrRosterPanel) || !c.Mark.In(c.Rect) {
				t.Errorf("カード %d %v が欄の外", i, c.Rect)
			}
			if i > 0 && c.Rect.Min.Y-v.Cards[i-1].Rect.Max.Y != 12 {
				t.Errorf("カード %d の間隔", i)
			}
			if c.Rect.Dy() != 104 {
				t.Errorf("カード %d の高さ %d", i, c.Rect.Dy())
			}
		}
		// 6 人目を足す。送りのボタンが出て、▼ で 6 人目が見える。
		s.campaign.Roster = append(s.campaign.Roster, core.Member{Character: "haru", Level: 1, Weapons: []string{"fire"}, Stats: tactics.Stats{HP: 10}, Alive: true})
		v = s.prepareHiResView()
		if len(v.Cards) != 5 || v.RosterDown.Label == "" || !v.RosterDown.Enabled || v.RosterUp.Enabled {
			t.Fatal(len(v.Cards), v.RosterUp, v.RosterDown)
		}
		s.Click(v.RosterDown.Rect.Min.X+3, v.RosterDown.Rect.Min.Y+3)
		v = s.prepareHiResView()
		if v.Cards[len(v.Cards)-1].Member != 5 || v.Cards[0].Member != 1 || !v.RosterUp.Enabled || v.RosterDown.Enabled {
			t.Fatal(v.Cards[0].Member, v.Cards[len(v.Cards)-1].Member)
		}
		// ▼ をもう一度押しても動かない。
		s.Click(v.RosterDown.Rect.Min.X+3, v.RosterDown.Rect.Min.Y+3)
		if s.prepareRosterTop != 1 {
			t.Fatal(s.prepareRosterTop)
		}
	})
	t.Run("TC-DES-NG-11-07 描くものはすべて窓に収まり、領域をはみ出さない（3 章・全員）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		for _, ch := range []string{"ch01", "ch02", "ch03"} {
			s := ng05Scenes(t, ch, true)
			for _, i := range s.livingMembers() {
				s.prepareMember = i
				v := s.prepareHiResView()
				for _, b := range append(append([]hrBox{}, v.Boxes...), v.Overlays...) {
					if !b.Rect.In(win) {
						t.Errorf("%s: 箱 %v が窓の外", ch, b.Rect)
					}
				}
				for _, tx := range v.Texts {
					r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text)*tx.Scale, tx.Y+lineHeight*tx.Scale)
					if !r.In(win) {
						t.Errorf("%s: 文「%s」%v が窓の外", ch, tx.Text, r)
					}
					in := false
					for _, reg := range []image.Rectangle{hrHeader, hrRosterPanel, hrCenterPanel, hrHandPanel, hrStockPanel, hrChapterPanel, hrFooter} {
						if r.In(reg) {
							in = true
						}
					}
					if !in {
						t.Errorf("%s: 文「%s」%v がどの領域にも収まらない", ch, tx.Text, r)
					}
				}
				for _, b := range v.Bars {
					if !b.Frame.In(win) || b.Fill < 0 || b.Fill > b.Frame.Dx() {
						t.Errorf("%s: 棒 %v %d", ch, b.Frame, b.Fill)
					}
				}
				for _, p := range v.Portraits {
					if !p.Rect.In(win) {
						t.Errorf("%s: 肖像 %v", ch, p.Rect)
					}
				}
			}
		}
	})
	t.Run("TC-DES-NG-11-08 カードのクリックで選び、チェックのクリックで出撃を切り替える（主人公は外せない）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		v := s.prepareHiResView()
		c1 := v.Cards[1]
		s.Click(c1.Rect.Max.X-5, c1.Rect.Min.Y+50)
		if s.prepareMember != c1.Member {
			t.Fatal(s.prepareMember)
		}
		if v = s.prepareHiResView(); !v.Cards[1].Selected || v.Cards[0].Selected || v.Cards[1].Rect.Min.X != 42 || v.Cards[0].Rect.Min.X != 32 {
			t.Fatal("選んだカードは右へ 10 px ずれる", v.Cards[1].Rect, v.Cards[0].Rect)
		}
		if !s.deployment[c1.Member] {
			t.Fatal("初めは出撃")
		}
		v = s.prepareHiResView()
		s.Click(v.Cards[1].Mark.Min.X+4, v.Cards[1].Mark.Min.Y+50)
		if s.deployment[c1.Member] {
			t.Fatal("チェックで外れる")
		}
		v = s.prepareHiResView()
		s.Click(v.Cards[1].Mark.Min.X+4, v.Cards[1].Mark.Min.Y+50)
		if !s.deployment[c1.Member] {
			t.Fatal("もう一度で出撃に戻る")
		}
		// 主人公（カード 0）は外せない。理由が出る。
		v = s.prepareHiResView()
		s.Click(v.Cards[0].Mark.Min.X+4, v.Cards[0].Mark.Min.Y+50)
		if !s.deployment[v.Cards[0].Member] || s.message != "主人公は必ず出撃します" {
			t.Fatal(s.message)
		}
		// 出撃の人数の表示。
		found := false
		for _, tx := range s.prepareHiResView().Texts {
			if tx.Text == "出撃 4 / 6" {
				found = true
			}
		}
		if !found {
			t.Fatal("人数の表示がない")
		}
	})
	t.Run("TC-DES-NG-11-09 在庫の行を選び、空き枠をクリックで 1 個渡す", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.prepareMember = 1 // ダイン（手持ち 2/5）
		before := itemCount(s.campaign.Roster[1].Items) + len(s.campaign.Roster[1].Weapons)
		v := s.prepareHiResView()
		if before != 2 || len(v.Stock) == 0 {
			t.Fatal(before, len(v.Stock))
		}
		row := v.Stock[len(v.Stock)-1] // 回復薬（在庫の最後の道具）
		if row.Item.weapon {
			t.Fatal("道具の行のはず")
		}
		stockBefore := stockCount(s.campaign, row.Item.id)
		// 空き枠を先にクリックしても渡らない。
		empty := v.Hand[2].Rect
		s.Click(empty.Min.X+5, empty.Min.Y+5)
		if s.message != "先に在庫の行を選んでください" || itemCount(s.campaign.Roster[1].Items) != before-len(s.campaign.Roster[1].Weapons) {
			t.Fatal(s.message)
		}
		s.Click(row.Rect.Min.X+5, row.Rect.Min.Y+5)
		if !s.prepareItem.ok || s.prepareItem != row.Item {
			t.Fatal("選べていない")
		}
		if !s.prepareHiResView().Stock[len(v.Stock)-1].Selected {
			t.Fatal("選んだ行が印されない")
		}
		s.Click(empty.Min.X+5, empty.Min.Y+5)
		if got := itemCount(s.campaign.Roster[1].Items) + len(s.campaign.Roster[1].Weapons); got != before+1 {
			t.Fatal(got)
		}
		if stockCount(s.campaign, row.Item.id) != stockBefore-1 {
			t.Fatal("在庫が 1 減っていない")
		}
		if s.message != "" {
			t.Fatal(s.message)
		}
		if hv := s.prepareHiResView(); hv.Hand[2].Name == "" {
			t.Fatal("手持ちの 3 枠目に出ていない")
		}
		// 同じ行をもう一度クリックすると選択が外れる。Esc でも外れる。
		s.prepareItem = row.Item
		s.Cancel()
		if s.prepareItem.ok {
			t.Fatal("Esc で選択が外れない")
		}
	})
	t.Run("TC-DES-NG-11-10 手持ちの枠のクリックで在庫に戻す（最後の武器は戻せない）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.prepareMember = 1
		m := &s.campaign.Roster[1]
		if len(m.Weapons) != 1 || itemCount(m.Items) != 1 {
			t.Fatal("ダインの初めの手持ちは武器 1・道具 1:", m.Weapons, m.Items)
		}
		hv := s.prepareHiResView()
		slot := hv.Hand[1] // 回復薬
		if slot.Item == "" {
			t.Fatal(slot)
		}
		stockBefore := stockCount(s.campaign, slot.Item)
		s.Click(slot.Rect.Min.X+5, slot.Rect.Min.Y+5)
		if itemCount(s.campaign.Roster[1].Items) != 0 || stockCount(s.campaign, slot.Item) != stockBefore+1 {
			t.Fatal("在庫へ戻っていない")
		}
		// 最後の 1 本の武器は外せない。状態は変わらず、理由が出る。
		w := hv.Hand[0].Rect
		s.Click(w.Min.X+5, w.Min.Y+5)
		if len(s.campaign.Roster[1].Weapons) != 1 || s.message == "" {
			t.Fatal(s.message)
		}
	})
	t.Run("TC-DES-NG-11-11 手持ちが 5 枠なら渡せず、理由を案内欄に出す", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.prepareMember = 0 // アオイは 5/5
		hv := s.prepareHiResView()
		for _, h := range hv.Hand {
			if h.Name == "" {
				t.Fatal("5 枠が埋まっていない")
			}
		}
		row := hv.Stock[0]
		s.Click(row.Rect.Min.X+5, row.Rect.Min.Y+5)
		if s.message != "手持ちは 5 枠までです。枠をクリックして在庫に戻すと、空きができます" {
			t.Fatal(s.message)
		}
		// 満杯の枠をクリックすると戻す（渡しではない）。
		before := len(s.campaign.Roster[0].Weapons) + itemCount(s.campaign.Roster[0].Items)
		s.Click(hv.Hand[4].Rect.Min.X+5, hv.Hand[4].Rect.Min.Y+5)
		if got := len(s.campaign.Roster[0].Weapons) + itemCount(s.campaign.Roster[0].Items); got != before-1 {
			t.Fatal(got)
		}
	})
	t.Run("TC-DES-NG-11-12 手持ちの武器を右クリックで装備（先頭）にする", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.prepareMember = 0 // アオイ: ファイア、癒やしの杖、…
		m := s.campaign.Roster[0]
		if len(m.Weapons) < 2 {
			t.Fatal(m.Weapons)
		}
		first, second := m.Weapons[0], m.Weapons[1]
		hv := s.prepareHiResView()
		if !hv.Hand[0].Equipped || hv.Hand[1].Equipped {
			t.Fatal("先頭が装備")
		}
		r := hv.Hand[1].Rect
		if !s.rightClickPrepare(r.Min.X+5, r.Min.Y+5) {
			t.Fatal("武器の枠の右クリックを受けない")
		}
		if w := s.campaign.Roster[0].Weapons; w[0] != second || w[1] != first {
			t.Fatal(w)
		}
		if hv = s.prepareHiResView(); !hv.Hand[0].Equipped {
			t.Fatal("装備の印")
		}
		// 先頭の武器の右クリックは何も変えず、理由を出す。道具の枠・枠の外は受けない（ふつうの取消し）。
		r = hv.Hand[0].Rect
		if !s.rightClickPrepare(r.Min.X+5, r.Min.Y+5) || s.campaign.Roster[0].Weapons[0] != second || s.message == "" {
			t.Fatal(s.message)
		}
		r = hv.Hand[2].Rect
		if s.rightClickPrepare(r.Min.X+5, r.Min.Y+5) || s.rightClickPrepare(5, 5) {
			t.Fatal("道具の枠・枠の外は受けない")
		}
		// 中央のカードの「武器」の行も、先頭の武器に変わる。
		found := false
		for _, tx := range s.prepareHiResView().Texts {
			if tx.Text == "武器　"+s.prepareWeapon(second).Name+"（"+weaponTypeName(s.prepareWeapon(second).Type)+"）" {
				found = true
			}
		}
		if !found {
			t.Fatal("武器の行が先頭の武器にならない")
		}
	})
	t.Run("TC-DES-NG-11-13 「配置へ」のボタン・Enter で配置へ進み、Esc は在庫の選択を外す", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", true)
		s.Click(hrDeployRect.Min.X+5, hrDeployRect.Min.Y+5)
		if s.Scene() != Deploy {
			t.Fatal(s.Scene())
		}
		s = ng05Scenes(t, "ch01", true)
		s.Advance()
		if s.Scene() != Deploy {
			t.Fatal(s.Scene())
		}
		s = ng05Scenes(t, "ch01", true)
		if v := s.View(); len(v.Buttons) != 1 || v.Buttons[0].Rect != hrDeployRect || !v.Buttons[0].Enabled {
			t.Fatal(v.Buttons)
		}
		s.Choose(0)
		if s.Scene() != Deploy {
			t.Fatal(s.Scene())
		}
		// 窓の外の座標は何もしない。
		s = ng05Scenes(t, "ch01", true)
		s.Click(1087, 0)
		s.Click(-5, 900)
		if s.Scene() != Prepare {
			t.Fatal(s.Scene())
		}
	})
	t.Run("TC-DES-NG-11-14 敵の集計: ch01〜ch03 の実データ（兵種・武器・ボス・増援）", func(t *testing.T) {
		cat, err := defload.Load(data.Alchemy())
		if err != nil {
			t.Fatal(err)
		}
		g := func(name string, w tactics.WeaponType, n int) EnemyGroup {
			return EnemyGroup{Name: name, Weapon: w, Count: n}
		}
		boss := func(name string, w tactics.WeaponType) EnemyGroup {
			return EnemyGroup{Name: name, Weapon: w, Count: 1, Boss: true}
		}
		want := []ChapterInfo{
			{Goal: "入口の広場の野盗の頭目を倒す", Total: 9, Groups: []EnemyGroup{g("野盗", tactics.Sword, 6), g("槍兵", tactics.Lance, 1), g("弓兵", tactics.Bow, 1)}, Bosses: []EnemyGroup{boss("野盗の頭目", tactics.Sword)}},
			{Goal: "先遣隊長ドルクを倒す", Total: 9, Groups: []EnemyGroup{g("槍兵", tactics.Lance, 2), g("剣士", tactics.Sword, 2), g("弓兵", tactics.Bow, 2), g("重装兵", tactics.Lance, 2)}, Bosses: []EnemyGroup{boss("先遣隊長ドルク", tactics.Lance)}},
			{Goal: "将軍ベルナドを倒す", Total: 12, Groups: []EnemyGroup{g("重装兵", tactics.Lance, 3), g("槍兵", tactics.Lance, 3), g("剣士", tactics.Sword, 3), g("弓兵", tactics.Bow, 2)}, Bosses: []EnemyGroup{boss("将軍ベルナド", tactics.Lance)}},
		}
		for i, w := range want {
			got := chapterInfo(cat, i)
			if !reflect.DeepEqual(got, w) {
				t.Errorf("章 %d:\n got %+v\nwant %+v", i+1, got, w)
			}
		}
		// ボス以外の敵の数 + ボス = 敵の総数。
		for i := range want {
			got := chapterInfo(cat, i)
			n := len(got.Bosses)
			for _, gr := range got.Groups {
				n += gr.Count
			}
			if n != got.Total {
				t.Errorf("章 %d: 合計 %d と総数 %d", i+1, n, got.Total)
			}
		}
		// 増援: base の定義には増援のある章があるので、その章で真になる。
		base, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		any := false
		for i, ch := range base.Chapters() {
			m, _ := base.Map(ch.Map)
			if got := chapterInfo(base, i).Reinforce; got != (len(m.Reinforcements) > 0) {
				t.Errorf("base 章 %d の増援 %v", i+1, got)
			}
			any = any || len(m.Reinforcements) > 0
		}
		if !any {
			t.Error("base に増援のある章がない（この検査の前提）")
		}
	})
	t.Run("TC-DES-NG-11-15 目的の文は goal があるときだけヘッダーに出す", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		for ch, goal := range map[string]string{"ch01": "目的　入口の広場の野盗の頭目を倒す", "ch02": "目的　先遣隊長ドルクを倒す", "ch03": "目的　将軍ベルナドを倒す"} {
			v := ng05Scenes(t, ch, true).prepareHiResView()
			found := false
			for _, tx := range v.Texts {
				if tx.Text == goal {
					found = true
					if tx.Y+lineHeight > hrHeader.Max.Y {
						t.Errorf("%s: 目的がヘッダーの外", ch)
					}
				}
			}
			if !found {
				t.Errorf("%s: 目的 %q が出ていない", ch, goal)
			}
		}
		s := ng05Scenes(t, "ch01", true)
		s.cfg.Catalog = baseCatalogForTest(t)
		s.campaign.Chapter = 0
		v := s.prepareHiResView()
		for _, tx := range v.Texts {
			if len(tx.Text) >= len("目的") && tx.Text[:len("目的")] == "目的" {
				t.Errorf("goal が空の章に目的の行がある: %q", tx.Text)
			}
		}
	})
	t.Run("TC-DES-NG-11-16 武器の相性の早見と一言は tactics の規則と一致する", func(t *testing.T) {
		cycle := weaponCycle()
		want := []tactics.WeaponType{tactics.Lance, tactics.Sword, tactics.Axe, tactics.Lance}
		if !reflect.DeepEqual(cycle, want) {
			t.Fatal(cycle)
		}
		kinds := []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Axe, tactics.Bow, tactics.Magic, tactics.Staff}
		// tactics の規則（槍 > 剣 > 斧 > 槍。弓・魔法・杖は相性なし）と、全組み合わせで一致する。
		for _, a := range kinds {
			for _, b := range kinds {
				rule := (a == tactics.Lance && b == tactics.Sword) || (a == tactics.Sword && b == tactics.Axe) || (a == tactics.Axe && b == tactics.Lance)
				if tactics.WeaponBeats(a, b) != rule {
					t.Errorf("%v → %v: WeaponBeats %v", a, b, tactics.WeaponBeats(a, b))
				}
			}
		}
		for i := 0; i+1 < len(cycle); i++ {
			if !tactics.WeaponBeats(cycle[i], cycle[i+1]) {
				t.Errorf("早見の %d 番目が有利の向きでない", i)
			}
		}
		for typ, w := range map[tactics.WeaponType]string{
			tactics.Lance: "槍は剣に強く、斧に弱い", tactics.Sword: "剣は斧に強く、槍に弱い", tactics.Axe: "斧は槍に強く、剣に弱い",
			tactics.Bow: "弓に武器の相性はない", tactics.Magic: "魔法に武器の相性はない",
		} {
			if got := weaponMatchupText(typ); got != w {
				t.Errorf("%v: %q", typ, got)
			}
		}
	})
	t.Run("TC-DES-NG-11-17 HiRes でない準備は今と同じ SceneView を返し、窓の中央の層を使う", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch01", false)
		if s.fullWindowPrepare() {
			t.Fatal("HiRes でないのに全面")
		}
		if got, want := s.View(), s.prepareView(); !reflect.DeepEqual(got, want) {
			t.Fatal("View と prepareView が違う")
		}
		v := s.View()
		if len(v.Tabs) != 2 || v.Info == nil || len(v.Rows) == 0 {
			t.Fatal("今の準備（タブ・情報欄・名簿の行）のまま:", len(v.Tabs))
		}
		if v.Buttons[len(v.Buttons)-1].Rect != prepareDeployRect {
			t.Fatal(v.Buttons)
		}
		// HiRes の準備は窓の全体なので、座標を補正しない。
		h := ng05Scenes(t, "ch01", true)
		if !h.fullWindowPrepare() {
			t.Fatal("HiRes なのに全面でない")
		}
		// 今の準備のクリック: 持ち物のタブに切り替わる。
		tab := v.Tabs[1].Rect
		s.Click(tab.Min.X+3, tab.Min.Y+3)
		if s.prepareTab != TabItems {
			t.Fatal("今の準備のタブが動かない")
		}
		s.Cancel()
		if s.prepareTab != TabRoster {
			t.Fatal("今の準備の Esc が動かない")
		}
	})
	t.Run("TC-DES-NG-11-18 ↑↓ でキャラを選び、見える位置まで名簿を送る", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch02", true)
		for range 4 {
			s.campaign.Roster = append(s.campaign.Roster, core.Member{Character: "haru", Level: 1, Weapons: []string{"fire"}, Stats: tactics.Stats{HP: 10}, Alive: true})
		}
		s.prepareMember = 0
		s.movePrepareMember(-1)
		if s.prepareMember != 0 {
			t.Fatal("先頭より上へは行かない")
		}
		for range 7 {
			s.movePrepareMember(1)
		}
		if s.prepareMember != 7 || s.prepareRosterTop != 3 {
			t.Fatal(s.prepareMember, s.prepareRosterTop)
		}
		v := s.prepareHiResView()
		sel := 0
		for _, c := range v.Cards {
			if c.Selected {
				sel++
				if c.Member != 7 {
					t.Fatal(c.Member)
				}
			}
		}
		if sel != 1 {
			t.Fatal("選んだカードが見えない")
		}
		for range 5 {
			s.movePrepareMember(1)
		}
		if s.prepareMember != 8 {
			t.Fatal(s.prepareMember)
		}
		for range 20 {
			s.movePrepareMember(-1)
		}
		if s.prepareMember != 0 || s.prepareRosterTop != 0 {
			t.Fatal(s.prepareMember, s.prepareRosterTop)
		}
	})
	t.Run("TC-DES-NG-11-19 手持ちの 5 枠と在庫の行は、それぞれの欄に収まり、重ならない。在庫は 3 行を超えるとスクロールする", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		s := ng05Scenes(t, "ch03", true)
		v := s.prepareHiResView()
		for i, h := range v.Hand {
			if !h.Rect.In(hrHandPanel) || h.Rect.Dx() != 54 || h.Rect.Dy() != 54 {
				t.Errorf("枠 %d %v", i, h.Rect)
			}
			if i > 0 && h.Rect.Overlaps(v.Hand[i-1].Rect) {
				t.Errorf("枠 %d が重なる", i)
			}
		}
		if len(v.Stock) != hrStockRows || v.StockDown.Label == "" || !v.StockDown.Enabled || v.StockUp.Enabled {
			t.Fatal(len(v.Stock), v.StockUp, v.StockDown)
		}
		for i, r := range v.Stock {
			if !r.Rect.In(hrStockPanel) || (i > 0 && r.Rect.Overlaps(v.Stock[i-1].Rect)) {
				t.Errorf("行 %d %v", i, r.Rect)
			}
		}
		first := v.Stock[0].Item
		s.Click(v.StockDown.Rect.Min.X+3, v.StockDown.Rect.Min.Y+3)
		if v2 := s.prepareHiResView(); v2.Stock[0].Item == first || !v2.StockUp.Enabled {
			t.Fatal("送れていない")
		}
	})
}

// stockCount は在庫の道具・武器の個数を返す。
func stockCount(c *core.Campaign, id string) int {
	for _, ic := range c.Stock {
		if ic.Item == id {
			return ic.Count
		}
	}
	for _, wc := range c.StockWeapons {
		if wc.Weapon == id {
			return wc.Count
		}
	}
	return 0
}

// baseCatalogForTest は base の定義データの目録を返す（goal がない章の確認用）。
func baseCatalogForTest(t *testing.T) *defs.Catalog {
	t.Helper()
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

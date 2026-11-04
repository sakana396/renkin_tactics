package ui

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 準備の名簿と持ち物のタブ（DES-UI-32、DES-IT07 3.5 節）。期待する座標は設計書の表の値を使う。
var (
	it07TabArea    = image.Rect(16, 16, 496, 48)
	it07LeftArea   = image.Rect(16, 56, 400, 472)
	it07RightArea  = image.Rect(416, 56, 816, 472)
	it07BottomArea = image.Rect(0, 512, 832, 576)
)

// memberOf は名簿から仲間の添字を探す。
func memberOf(t *testing.T, s *SceneController, character string) int {
	t.Helper()
	for i, m := range s.campaign.Roster {
		if m.Character == character {
			return i
		}
	}
	t.Fatalf("名簿に %s がいない", character)
	return -1
}

// selectMember は名簿のタブで仲間の行をクリックして選ぶ（送りが要れば送る）。
func selectMember(t *testing.T, s *SceneController, member int) {
	t.Helper()
	clickPrepareTab(t, s, "名簿")
	for range 10 {
		for _, r := range s.View().Rows {
			if r.Member == member {
				s.Click(r.Rect.Min.X, r.Rect.Min.Y)
				if s.prepareMember != member {
					t.Fatal("行で仲間を選べない", member)
				}
				return
			}
		}
		clickSceneButton(t, s, "▼ 名簿")
	}
	t.Fatal("名簿の行がない", member)
}

// textRect は文の範囲（1 行の高さ 16）。
func sceneTextRect(tx SceneText) image.Rectangle {
	return image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
}

// checkPrepareLayout は、準備の文・ボタン・行・タブが窓の中で互いに重ならず、
// 名簿の行と持ち物の 5 枠は左列、在庫と情報は右列、理由と「配置へ」は下欄に収まることを確かめる（DES-UI-32）。
func checkPrepareLayout(t *testing.T, step string, v SceneView) {
	t.Helper()
	type box struct {
		name string
		r    image.Rectangle
	}
	var boxes []box
	for _, tab := range v.Tabs {
		if !tab.Rect.In(it07TabArea) {
			t.Errorf("%s: タブ %q %v が領域の外", step, tab.Label, tab.Rect)
		}
		boxes = append(boxes, box{"タブ " + tab.Label, tab.Rect})
	}
	for _, r := range v.Rows {
		if !r.Rect.In(it07LeftArea) || !r.Mark.In(it07LeftArea) {
			t.Errorf("%s: 名簿の行 %v %v が左列の外", step, r.Rect, r.Mark)
		}
		boxes = append(boxes, box{"行", r.Rect}, box{"印", r.Mark})
	}
	for _, b := range v.Buttons {
		area := it07LeftArea
		switch {
		case b.Label == "配置へ":
			area = it07BottomArea
		case b.Rect.Min.X >= it07RightArea.Min.X:
			area = it07RightArea
		}
		if !b.Rect.In(area) {
			t.Errorf("%s: ボタン %q %v が %v の外", step, b.Label, b.Rect, area)
		}
		if textWidth(b.Label)+12 > b.Rect.Dx() {
			t.Errorf("%s: ボタン %q の文字が枠を越す", step, b.Label)
		}
		boxes = append(boxes, box{"ボタン " + b.Label, b.Rect})
	}
	for _, tx := range v.Texts {
		r := sceneTextRect(tx)
		if !r.In(image.Rect(0, 0, ScreenWidth, ScreenHeight)) {
			t.Errorf("%s: 文 %q %v が窓の外", step, tx.Text, r)
		}
		if tx.Style == StyleBad && tx.Y >= it07BottomArea.Min.Y && !r.In(it07BottomArea) {
			t.Errorf("%s: 理由 %q が下欄の外", step, tx.Text)
		}
		boxes = append(boxes, box{"文 " + tx.Text, r})
	}
	if v.Info != nil {
		for _, tx := range v.Info.Texts {
			r := image.Rect(tx.X, tx.Y, tx.X+textWidth(tx.Text), tx.Y+lineHeight)
			if !r.In(it07RightArea) {
				t.Errorf("%s: 情報 %q %v が右列の外", step, tx.Text, r)
			}
		}
		for _, tab := range v.Info.Tabs {
			if !tab.Rect.In(it07RightArea) {
				t.Errorf("%s: 情報のページの印 %v が右列の外", step, tab.Rect)
			}
		}
	}
	for i, a := range boxes {
		for _, b := range boxes[i+1:] {
			// 名簿の行と印の範囲は、その中に描く印・絵・文を含むので、中に収まるものは除く（DES-UI-13）。
			if (a.name == "行" || a.name == "印") && b.r.In(a.r) || (b.name == "行" || b.name == "印") && a.r.In(b.r) {
				continue
			}
			if a.name == "行" && b.name == "印" || a.name == "印" && b.name == "行" {
				continue
			}
			if a.r.Overlaps(b.r) {
				t.Errorf("%s: %s %v と %s %v が重なる", step, a.name, a.r, b.name, b.r)
			}
		}
	}
}

// itemsColumn は右列（在庫）に並ぶ見出しと品のボタンを上から順に返す。見出しは「#見出し」。
func itemsColumn(v SceneView) []string {
	type entry struct {
		y    int
		text string
	}
	var es []entry
	for _, tx := range v.Texts {
		if tx.X >= it07RightArea.Min.X && tx.Style == StyleLabel && tx.Y < 360 {
			es = append(es, entry{tx.Y, "#" + tx.Text})
		}
	}
	for _, b := range v.Buttons {
		if strings.HasPrefix(b.Label, "手持ちへ: ") {
			es = append(es, entry{b.Rect.Min.Y, strings.TrimPrefix(b.Label, "手持ちへ: ")})
		}
	}
	for i := range es {
		for j := i + 1; j < len(es); j++ {
			if es[j].y < es[i].y {
				es[i], es[j] = es[j], es[i]
			}
		}
	}
	var out []string
	for _, e := range es {
		out = append(out, e.text)
	}
	return out
}

// handColumn は左列の 5 枠のボタンの文を上から順に返す。
func handColumn(v SceneView) ([]string, []image.Rectangle) {
	var labels []string
	var rects []image.Rectangle
	for _, b := range v.Buttons {
		if b.Rect.Min.X == 40 {
			labels = append(labels, b.Label)
			rects = append(rects, b.Rect)
		}
	}
	return labels, rects
}

// longNameCatalog は主人公の名前だけを長くした目録を作る（DES-UI-32 の長い名前の確認）。
func longNameCatalog(t *testing.T, name string) *defs.Catalog {
	t.Helper()
	return renamedCatalog(t, map[string][2]string{"characters.toml": {"レオン", name}})
}

// renamedCatalog は、定義のファイルごとに 1 つの名前（name = "…"）を置き換えた目録を作る。
func renamedCatalog(t *testing.T, renames map[string][2]string) *defs.Catalog {
	t.Helper()
	source := fstest.MapFS{}
	if err := fs.WalkDir(data.Base(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), path)
		source[path] = &fstest.MapFile{Data: b}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for file, r := range renames {
		src := string(source[file].Data)
		old := fmt.Sprintf("name = %q", r[0])
		if !strings.Contains(src, old) {
			t.Fatal("前提: 名前", file, r[0])
		}
		source[file].Data = []byte(strings.Replace(src, old, fmt.Sprintf("name = %q", r[1]), 1))
	}
	cat, err := defload.Load(source)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestIT07PrepareItems(t *testing.T) {
	t.Run("TC-FR-CMP-03-16 名簿と持ち物の切り替え・5 枠・見出しで分けた在庫", func(t *testing.T) {
		s := recruitScenes(t, "ch09", core.Withdraw)
		if len(s.livingMembers()) != 9 {
			t.Fatal("前提: 9 人", len(s.livingMembers()))
		}
		// 在庫に武器・杖・道具。在庫の順は武器（剣・槍）→ 杖 → 道具の見出しの中で保つ。
		s.campaign.StockWeapons = []core.WeaponCount{{Weapon: "heal_staff", Count: 1}, {Weapon: "armor_killer", Count: 1}, {Weapon: "horse_killer", Count: 2}}
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 3}}

		// 初めは名簿のタブ。タブは (16,16)〜(496,48) に名簿・持ち物の順。
		v := s.View()
		if len(v.Tabs) != 2 || v.Tabs[0].Label != "名簿" || v.Tabs[1].Label != "持ち物" || !v.Tabs[0].Selected || v.Tabs[1].Selected {
			t.Fatalf("タブ %+v", v.Tabs)
		}
		// 名簿: 生存者の行（8 人まで、送りで 9 人目）・出撃の印、右列に選んだ仲間の能力・装備。
		if len(v.Rows) != 8 || v.Info == nil || len(v.Info.Tabs) != 2 {
			t.Fatalf("名簿 %d 行、情報 %v", len(v.Rows), v.Info)
		}
		marks := 0
		for _, tx := range v.Texts {
			if tx.Text == "[✓]" || tx.Text == "[ ]" {
				marks++
			}
		}
		if marks != 8 {
			t.Fatal("出撃の印", marks)
		}
		checkPrepareLayout(t, "名簿", v)
		gald := memberOf(t, s, "gald")
		selectMember(t, s, gald)
		info := ""
		for _, tx := range s.View().Info.Texts {
			info += tx.Text + "\n"
		}
		if !strings.Contains(info, "ガルド") || !strings.Contains(info, "アーマー") {
			t.Fatal("右列に選んだ仲間の能力がない", info)
		}
		s.ToggleInfoPage()
		info = ""
		for _, tx := range s.View().Info.Texts {
			info += tx.Text + "\n"
		}
		if !strings.Contains(info, "鉄の槍") {
			t.Fatal("右列に選んだ仲間の装備がない", info)
		}
		s.ToggleInfoPage()

		// 何度切り替えても、選んだ仲間・在庫・名簿・出撃の印・人数は変わらない。
		baseline := campaignBytes(t, s)
		selected := append([]int(nil), s.SelectedMembers()...)
		for i := range 5 {
			clickPrepareTab(t, s, "持ち物")
			v := s.View()
			if !v.Tabs[1].Selected || v.Tabs[0].Selected || v.Info != nil || len(v.Rows) != 0 {
				t.Fatalf("%d 回目の持ち物: %+v", i, v.Tabs)
			}
			checkPrepareLayout(t, "持ち物", v)
			clickPrepareTab(t, s, "名簿")
			if !s.View().Tabs[0].Selected {
				t.Fatal("名簿へ戻らない")
			}
			if s.prepareMember != gald || !reflect.DeepEqual(s.SelectedMembers(), selected) || !bytes.Equal(campaignBytes(t, s), baseline) || s.Scene() != Prepare {
				t.Fatal("切り替えで状態が変わった", i)
			}
		}
		// 選んでいるタブをもう一度押しても変わらない。
		clickPrepareTab(t, s, "名簿")
		if !s.View().Tabs[0].Selected {
			t.Fatal("名簿のタブ")
		}

		// 持ち物: 左列に選んだ仲間（ガルド: 鉄の槍・回復薬 1）の 5 枠を常に 5 行。空きは「―」で押せない。装備は E 印。
		clickPrepareTab(t, s, "持ち物")
		v = s.View()
		labels, rects := handColumn(v)
		if !reflect.DeepEqual(labels, []string{"在庫へ: 鉄の槍", "在庫へ: 回復薬", "―", "―", "―"}) {
			t.Fatal(labels)
		}
		for k, r := range rects {
			if r.Min.Y != 88+k*32 {
				t.Fatal("5 枠の行の位置", k, r)
			}
		}
		for _, b := range v.Buttons {
			if b.Label == "―" && b.Enabled {
				t.Fatal("空きの枠を押せる")
			}
		}
		var es []SceneText
		for _, tx := range v.Texts {
			if tx.Text == "E" {
				es = append(es, tx)
			}
		}
		if len(es) != 1 || !sceneTextRect(es[0]).In(image.Rect(16, rects[0].Min.Y, rects[0].Min.X, rects[0].Max.Y)) {
			t.Fatal("E 印は先頭の武器の行の左に 1 つ", es)
		}
		if !strings.Contains(viewStrings(v), "ガルド（アーマー）　手持ち 2/5") {
			t.Fatal("左列の見出し", viewStrings(v))
		}
		// 右列: 在庫を武器／杖／道具の見出しで分ける。
		if got := itemsColumn(v); !reflect.DeepEqual(got, []string{"#武器", "アーマーキラー", "ホースキラー ×2", "#杖", "癒やしの杖", "#道具", "回復薬 ×3"}) {
			t.Fatal("在庫の見出し", got)
		}
		// 品を選ぶと、性能を右列の下部に出す。装備できない武器は理由も出す（状態は変えない）。
		before := campaignBytes(t, s)
		clickSceneButton(t, s, "手持ちへ: アーマーキラー")
		v = s.View()
		var detail []SceneText
		for _, tx := range v.Texts {
			if tx.X >= it07RightArea.Min.X && tx.Y >= 360 && tx.Y < it07RightArea.Max.Y {
				detail = append(detail, tx)
			}
		}
		joined := ""
		for _, tx := range detail {
			joined += tx.Text
			if !sceneTextRect(tx).In(it07RightArea) {
				t.Fatal("性能が右列の外", tx)
			}
		}
		if !strings.Contains(joined, "選んだ品: アーマーキラー") || !strings.Contains(joined, "威力") || !strings.Contains(joined, "特効 鎧") {
			t.Fatal("性能の表示", joined)
		}
		if detail[len(detail)-1].Style != StyleBad || detail[len(detail)-1].Text != "ガルド（アーマー）は剣を装備できない" {
			t.Fatal("装備できない理由", detail)
		}
		if !bytes.Equal(campaignBytes(t, s), before) {
			t.Fatal("装備できない武器で状態が変わった")
		}
		checkPrepareLayout(t, "性能", v)
		// 杖・道具も性能を出す。
		clickSceneButton(t, s, "在庫へ: 回復薬")
		if !strings.Contains(viewStrings(s.View()), "選んだ品: 回復薬") || !strings.Contains(viewStrings(s.View()), "回復") {
			t.Fatal("道具の性能", viewStrings(s.View()))
		}

		// 在庫を送る: 見出しを含めて 10 行を超えると「▲ 前へ」「▼ 次へ」と位置を出す。
		s.campaign.StockWeapons = append(s.campaign.StockWeapons,
			core.WeaponCount{Weapon: "brave_axe", Count: 1}, core.WeaponCount{Weapon: "hammer", Count: 1},
			core.WeaponCount{Weapon: "killing_sword", Count: 1}, core.WeaponCount{Weapon: "long_bow", Count: 1},
			core.WeaponCount{Weapon: "thunder_sword", Count: 1})
		v = s.View()
		if got := itemsColumn(v); !reflect.DeepEqual(got, []string{"#武器", "アーマーキラー", "ホースキラー ×2", "勇者の斧", "ハンマー", "キルソード", "長弓", "雷の剣", "#杖", "癒やしの杖"}) {
			t.Fatal("送りの前", got)
		}
		if !strings.Contains(viewStrings(v), "1〜10/12") {
			t.Fatal("在庫の位置", viewStrings(v))
		}
		checkPrepareLayout(t, "送りの前", v)
		clickSceneButton(t, s, "▼ 次へ")
		clickSceneButton(t, s, "▼ 次へ")
		v = s.View()
		// 3〜12 行目。回復薬はガルドが 1 個戻して 4 個。
		if got := itemsColumn(v); !reflect.DeepEqual(got, []string{"ホースキラー ×2", "勇者の斧", "ハンマー", "キルソード", "長弓", "雷の剣", "#杖", "癒やしの杖", "#道具", "回復薬 ×4"}) || !strings.Contains(viewStrings(v), "3〜12/12") {
			t.Fatal("送りの後", got)
		}
		checkPrepareLayout(t, "送りの後", v)
		if s.prepareMember != gald || !reflect.DeepEqual(s.SelectedMembers(), selected) {
			t.Fatal("送りで選択が変わった")
		}
		// 準備に入り直す（章の始め・やり直し）と、名簿のタブから始める（UC-11 基本フロー 1）。
		s.enterPrepare()
		if v := s.View(); !v.Tabs[0].Selected || len(v.Rows) == 0 || strings.Contains(viewStrings(v), "選んだ品") {
			t.Fatal("入り直しで名簿のタブにならない")
		}
	})
	t.Run("TC-FR-CMP-03-16 長い名前と 9 人目でも行・ボタン・案内が重ならない", func(t *testing.T) {
		long := "レオンハルト・フォン・アイゼンベルク・ヴァルトシュタイン"
		s := recruitScenes(t, "ch09", core.Withdraw)
		cat := longNameCatalog(t, long)
		s.cfg.Catalog = cat
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		s.campaign = c
		s.enterPrepare()
		s.message = "主人公は必ず出撃します"
		v := s.View()
		checkPrepareLayout(t, "長い名前の名簿", v)
		if !strings.HasPrefix(v.Texts[2].Text, "レオンハルト") || !strings.HasSuffix(v.Texts[2].Text, "…") {
			t.Fatal("長い名前を縮めていない", v.Texts[2])
		}
		clickSceneButton(t, s, "▼ 名簿")
		v = s.View()
		if v.Rows[len(v.Rows)-1].Member != 8 {
			t.Fatal("9 人目", v.Rows)
		}
		checkPrepareLayout(t, "9 人目", v)
		clickPrepareTab(t, s, "持ち物")
		s.message = "主人公は必ず出撃します"
		checkPrepareLayout(t, "長い名前の持ち物", s.View())
	})
	t.Run("TC-FR-CMP-03-17 出し入れは既存の操作を呼び、満杯・装備できない・最後の 1 本は状態を変えず理由を出す", func(t *testing.T) {
		s := recruitScenes(t, "ch09", core.Withdraw)
		store := s.cfg.Store.(*memoryStore)
		writes := store.writes
		s.campaign.StockWeapons = []core.WeaponCount{{Weapon: "armor_killer", Count: 1}, {Weapon: "killing_sword", Count: 1}}
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 3}}
		leon := memberOf(t, s, "leon")
		selectMember(t, s, leon)
		clickPrepareTab(t, s, "持ち物")
		// 道具の出し入れ（StoreItem・TakeItem）。
		hand := s.campaign.Roster[leon].Items[0].Count
		clickSceneButton(t, s, "在庫へ: 回復薬")
		if s.campaign.Roster[leon].Items[0].Count != hand-1 || s.campaign.Stock[0].Count != 4 {
			t.Fatal("在庫へ", s.campaign.Roster[leon].Items, s.campaign.Stock)
		}
		clickSceneButton(t, s, "手持ちへ: 回復薬 ×4")
		if s.campaign.Roster[leon].Items[0].Count != hand || s.campaign.Stock[0].Count != 3 {
			t.Fatal("手持ちへ", s.campaign.Roster[leon].Items, s.campaign.Stock)
		}
		// 武器の出し入れ（TakeWeapon）。武器は列の後ろに入り、先頭（初めの装備・E）は変わらない。
		clickSceneButton(t, s, "手持ちへ: アーマーキラー")
		clickSceneButton(t, s, "手持ちへ: キルソード")
		if !reflect.DeepEqual(s.campaign.Roster[leon].Weapons, []string{"iron_sword", "armor_killer", "killing_sword"}) {
			t.Fatal(s.campaign.Roster[leon].Weapons)
		}
		labels, _ := handColumn(s.View())
		if !reflect.DeepEqual(labels, []string{"在庫へ: 鉄の剣", "在庫へ: アーマーキラー", "在庫へ: キルソード", "在庫へ: 回復薬", "在庫へ: 回復薬"}) {
			t.Fatal("5 枠", labels)
		}
		// 満杯: 在庫から持たせると 6 枠になるので拒む。
		before := campaignBytes(t, s)
		clickSceneButton(t, s, "手持ちへ: 回復薬 ×3")
		if !bytes.Equal(campaignBytes(t, s), before) || !strings.Contains(viewStrings(s.View()), "手持ちは 5 枠までです") {
			t.Fatal("満杯で状態が変わった、または理由がない", viewStrings(s.View()))
		}
		checkPrepareLayout(t, "満杯", s.View())
		// 理由は下欄 (0,512)〜(832,576) の「配置へ」の左に出す（DES-UI-32）。
		for _, tx := range s.View().Texts {
			if tx.Text == "手持ちは 5 枠までです" && (tx.Style != StyleBad || !sceneTextRect(tx).In(image.Rect(0, 512, 672, 576))) {
				t.Fatal("理由の位置", tx)
			}
		}
		// 先頭へ（MoveWeaponFirst）で E 印が移る。武器の順は DES-IT06 6 章のまま。
		clickSceneButton(t, s, "先頭へ: キルソード")
		if !reflect.DeepEqual(s.campaign.Roster[leon].Weapons, []string{"killing_sword", "iron_sword", "armor_killer"}) {
			t.Fatal(s.campaign.Roster[leon].Weapons)
		}
		_, rects := handColumn(s.View())
		for _, tx := range s.View().Texts {
			if tx.Text == "E" && tx.Y < rects[0].Min.Y || tx.Text == "E" && tx.Y >= rects[0].Max.Y {
				t.Fatal("E 印が先頭の行にない", tx)
			}
		}
		// 装備できない武器: ガルドにアーマーキラー（剣）は持たせられない。
		clickSceneButton(t, s, "在庫へ: アーマーキラー")
		gald := memberOf(t, s, "gald")
		selectMember(t, s, gald)
		clickPrepareTab(t, s, "持ち物")
		before = campaignBytes(t, s)
		clickSceneButton(t, s, "手持ちへ: アーマーキラー")
		if !bytes.Equal(campaignBytes(t, s), before) || !strings.Contains(viewStrings(s.View()), core.ErrWeaponType.Error()) || !strings.Contains(viewStrings(s.View()), "ガルド（アーマー）は剣を装備できない") {
			t.Fatal("装備できない武器", viewStrings(s.View()))
		}
		// 最後の武器・杖 1 本: ガルドの鉄の槍、リナの癒やしの杖は外せない。
		for _, who := range []struct{ id, label string }{{"gald", "在庫へ: 鉄の槍"}, {"rina", "在庫へ: 癒やしの杖"}} {
			m := memberOf(t, s, who.id)
			selectMember(t, s, m)
			clickPrepareTab(t, s, "持ち物")
			before = campaignBytes(t, s)
			clickSceneButton(t, s, who.label)
			if !bytes.Equal(campaignBytes(t, s), before) || !strings.Contains(viewStrings(s.View()), core.ErrLastWeapon.Error()) {
				t.Fatal("最後の 1 本", who.id, viewStrings(s.View()))
			}
		}
		// 右クリック・Esc: 品の選択 → タブの選択 → 準備画面の既存の戻り先（準備のまま）。
		if !strings.Contains(viewStrings(s.View()), "選んだ品: 癒やしの杖") {
			t.Fatal("前提: 品を選んでいる")
		}
		before = campaignBytes(t, s)
		member, selected := s.prepareMember, append([]int(nil), s.SelectedMembers()...)
		s.Cancel()
		if v := s.View(); strings.Contains(viewStrings(v), "選んだ品") || strings.Contains(viewStrings(v), core.ErrLastWeapon.Error()) || !v.Tabs[1].Selected {
			t.Fatal("1 回目の取消しは品の選択を外す", viewStrings(v))
		}
		s.Cancel()
		if v := s.View(); !v.Tabs[0].Selected || s.Scene() != Prepare {
			t.Fatal("2 回目の取消しは名簿のタブへ")
		}
		s.Cancel()
		s.Cancel()
		if v := s.View(); !v.Tabs[0].Selected || s.Scene() != Prepare || s.prepareMember != member || !reflect.DeepEqual(s.SelectedMembers(), selected) || !bytes.Equal(campaignBytes(t, s), before) {
			t.Fatal("名簿での取消しで準備を離れた、または状態が変わった")
		}
		// 出し入れとタブでセーブを書かない。持ち物のタブからも「配置へ」で配置へ進み、戻ると準備、戦闘を始められる。
		if store.writes != writes {
			t.Fatal("準備の操作でセーブを書いた", store.writes, writes)
		}
		clickPrepareTab(t, s, "持ち物")
		clickSceneButton(t, s, "配置へ")
		if s.Scene() != Deploy {
			t.Fatal("配置へ進まない", s.Scene())
		}
		s.Cancel()
		if s.Scene() != Prepare || !bytes.Equal(campaignBytes(t, s), before) {
			t.Fatal("配置から準備へ戻らない", s.Scene())
		}
		clickSceneButton(t, s, "配置へ")
		clickSceneButton(t, s, "戦闘開始")
		if s.Scene() != Battle || !reflect.DeepEqual(s.sortie.Members, selected) {
			t.Fatal("戦闘を始められない", s.Scene())
		}
	})
	t.Run("TC-FR-CMP-03-16 長い品名でもボタンの文は枠に収め、操作の言葉と個数は残す", func(t *testing.T) {
		longWeapon := "アーマーキラー・オブ・ザ・とても長い名前の剣・改"
		longItem := "とても長い名前の回復薬・大瓶入り特製・旅の商人のおすすめ品"
		cat := renamedCatalog(t, map[string][2]string{"weapons.toml": {"アーマーキラー", longWeapon}, "items.toml": {"回復薬", longItem}})
		s := recruitScenes(t, "ch09", core.Withdraw)
		s.cfg.Catalog = cat
		c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
		if err != nil {
			t.Fatal(err)
		}
		s.campaign = c
		s.enterPrepare()
		leon := memberOf(t, s, "leon")
		s.campaign.Roster[leon].Weapons = []string{"iron_sword", "armor_killer"}
		s.campaign.StockWeapons = []core.WeaponCount{{Weapon: "armor_killer", Count: 2}}
		s.campaign.Stock = []defs.ItemCount{{Item: "potion", Count: 3}}
		selectMember(t, s, leon)
		clickPrepareTab(t, s, "持ち物")
		v := s.View()
		// ボタンの文は枠（左の余白 12 px を除く）に収まる。重なりと領域も確かめる。
		checkPrepareLayout(t, "長い品名", v)
		want := map[string]bool{}
		for _, b := range v.Buttons {
			switch {
			case strings.HasPrefix(b.Label, "手持ちへ: アーマーキラー") && strings.HasSuffix(b.Label, "… ×2"):
				want["在庫の武器"] = true
			case strings.HasPrefix(b.Label, "手持ちへ: とても長い") && strings.HasSuffix(b.Label, "… ×3"):
				want["在庫の道具"] = true
			case strings.HasPrefix(b.Label, "在庫へ: アーマーキラー") && strings.HasSuffix(b.Label, "…"):
				want["手持ちの武器"] = true
			case strings.HasPrefix(b.Label, "先頭へ: アーマーキラー") && strings.HasSuffix(b.Label, "…"):
				want["先頭へ"] = true
			case strings.HasPrefix(b.Label, "在庫へ: とても長い") && strings.HasSuffix(b.Label, "…"):
				want["手持ちの道具"] = true
			}
		}
		if len(want) != 5 {
			t.Fatal("長い品名を縮めたボタン", want)
		}
		// 縮めたボタンも、その品を出し入れする。選んだ品の欄は全名を右列の幅で折り返す。
		for _, b := range v.Buttons {
			if strings.HasPrefix(b.Label, "手持ちへ: アーマーキラー") {
				s.Click(b.Rect.Min.X, b.Rect.Min.Y)
				break
			}
		}
		if w := s.campaign.Roster[leon].Weapons; !reflect.DeepEqual(w, []string{"iron_sword", "armor_killer", "armor_killer"}) {
			t.Fatal(w)
		}
		joined := ""
		for _, tx := range s.View().Texts {
			if tx.X >= it07RightArea.Min.X && tx.Y >= 360 {
				joined += tx.Text
			}
		}
		if !strings.Contains(joined, "選んだ品: "+longWeapon) {
			t.Fatal("選んだ品の全名", joined)
		}
		checkPrepareLayout(t, "長い品名の選択", s.View())
	})
	t.Run("TC-FR-CMP-03-16 / TC-FR-CMP-03-17 実入力の経路（Game.Update の場面の入力 → SceneController）でタブ・出し入れ・Tab・取消し", func(t *testing.T) {
		s := recruitScenes(t, "ch09", core.Withdraw)
		g := &Game{scenes: s}
		click := func(x, y int) { g.sceneInput(sceneKeys{Left: true}, x, y) }
		cancel := func() { g.sceneInput(sceneKeys{Cancel: true}, 0, 0) }
		gald := memberOf(t, s, "gald")
		baseline := campaignBytes(t, s)
		// 名簿の行（y = 64 から 24 px ごと、x 56〜399）を押して仲間を選ぶ。
		row := -1
		for i, r := range s.View().Rows {
			if r.Member == gald {
				row = i
			}
		}
		click(100, 64+row*24+2)
		if s.prepareMember != gald {
			t.Fatal("行を押して選べない", s.prepareMember)
		}
		// Tab キーは名簿の情報のページを送る。状態は変わらない。
		g.sceneInput(sceneKeys{Tab: true}, 0, 0)
		if s.infoPage != PageEquip || !bytes.Equal(campaignBytes(t, s), baseline) {
			t.Fatal("Tab", s.infoPage)
		}
		// タブ (16,16)〜(496,48) の右半分（持ち物）を押す。
		click(400, 30)
		if !s.View().Tabs[1].Selected || s.prepareMember != gald || !bytes.Equal(campaignBytes(t, s), baseline) {
			t.Fatal("持ち物のタブ")
		}
		// 5 枠の 2 行目（y = 120〜147、x 40〜231）はガルドの回復薬。押すと在庫へ戻す。
		click(100, 130)
		if len(s.campaign.Roster[gald].Items) != 0 || !strings.Contains(viewStrings(s.View()), "選んだ品: 回復薬") {
			t.Fatal("在庫へ", s.campaign.Roster[gald].Items)
		}
		// 1 行目の鉄の槍は最後の 1 本なので、状態は変わらず理由を出す。
		before := campaignBytes(t, s)
		click(100, 100)
		if !bytes.Equal(campaignBytes(t, s), before) || !strings.Contains(viewStrings(s.View()), core.ErrLastWeapon.Error()) {
			t.Fatal("最後の 1 本")
		}
		// 取消し: 品の選択 → 名簿のタブ → 準備のまま。
		cancel()
		if strings.Contains(viewStrings(s.View()), "選んだ品") || !s.View().Tabs[1].Selected {
			t.Fatal("1 回目の取消し")
		}
		cancel()
		cancel()
		if !s.View().Tabs[0].Selected || s.Scene() != Prepare || !bytes.Equal(campaignBytes(t, s), before) {
			t.Fatal("2・3 回目の取消し")
		}
		// 下欄の「配置へ」(672,528)〜(816,568) で配置へ進む。
		click(700, 540)
		if s.Scene() != Deploy {
			t.Fatal("配置へ", s.Scene())
		}
	})
}

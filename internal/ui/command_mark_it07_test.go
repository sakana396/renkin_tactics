package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// commandSetup は 8 × 3 の平地に、主人公 (0, 1)（必ず当たって倒す剣）、隣の小隊長 (1, 1)（HP 1）、
// 離れた配下 (6, 1)、ごろつき（初めからまっすぐ寄る敵）(7, 0)、ボス (7, 2) を置く（DES-UI-35）。
// 名前には「小隊長」「配下」を使わず、印の語と出来事の文を名前と取り違えないようにする。
//
//	0: 主人公 1: 小隊長「兵 A」 2: 配下「兵 B」 3: ごろつき 4: ボス
func commandSetup() tactics.Setup {
	strong := tactics.Weapon{Name: "剣", Type: tactics.Sword, Might: 50, Hit: 300, MinRange: 1, MaxRange: 1}
	group := func(name string, x, y, hp int) tactics.UnitSetup {
		return tactics.UnitSetup{Name: name, Side: tactics.Enemy, Level: 1, AI: tactics.Hold, Approach: tactics.GroupApproach, Group: "g", Stats: tactics.Stats{HP: hp, Mov: 1}, Pos: tactics.Pos{X: x, Y: y}, Weapon: testLance}
	}
	leader := group("兵 A", 1, 1, 1)
	leader.Commander = true
	return tactics.Setup{Terrain: field(8, 3), Units: []tactics.UnitSetup{
		{Name: "主人公", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 1}, Weapon: strong},
		leader,
		group("兵 B", 6, 1, 30),
		{Name: "ごろつき", Side: tactics.Enemy, Level: 1, AI: tactics.Hold, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 7, Y: 0}, Weapon: testLance},
		{Name: "ボス", Side: tactics.Enemy, Level: 1, Boss: true, AI: tactics.Hold, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 7, Y: 2}, Weapon: testLance},
	}}
}

// marksOf は盤面のユニットの印を、添字の順に返す。
func marksOf(units []tactics.Unit) []CommandMark {
	out := make([]CommandMark, len(units))
	for i, u := range units {
		out[i] = CommandMarkOf(u)
	}
	return out
}

// infoMarkAt は、カーソルを p に置いたときの右欄の印と語を返す。印がなければ ok が偽。
func infoMarkAt(t *testing.T, c *Controller, p tactics.Pos) (InfoMark, InfoText, InfoText, bool) {
	t.Helper()
	v := c.InfoView(cellCenter(p))
	if len(v.Marks) == 0 {
		return InfoMark{}, InfoText{}, InfoText{}, false
	}
	if len(v.Marks) != 1 {
		t.Fatalf("%v: 印 %d 個", p, len(v.Marks))
	}
	m := v.Marks[0]
	var name, word InfoText
	for _, tx := range v.Texts {
		if tx.Y == m.Rect.Min.Y-(lineHeight-commandMarkSize)/2 && tx.X < m.Rect.Min.X && name.Text == "" {
			name = tx
		}
		if tx.X > m.Rect.Min.X && tx.Text == m.Mark.Word() {
			word = tx
		}
	}
	return m, name, word, true
}

func TestCommandMarks(t *testing.T) {
	t.Run("TC-FR-UNT-09-30 小隊長は金の菱形、配下は同色の小点、崩れは灰の割れた小点。ごろつき・ボス・自軍は印なし。印は絵の左上", func(t *testing.T) {
		if CommanderMark.shape() != shapeDiamond || SubordinateMark.shape() != shapeDot || CollapsedMark.shape() != shapeCrackedDot || NoCommandMark.shape() != shapeNone {
			t.Fatal("印の形が菱形・小点・割れた小点でない")
		}
		if CommanderMark.color() != gold || SubordinateMark.color() != CommanderMark.color() || CollapsedMark.color() != commandMarkGray || commandMarkGray == gold {
			t.Fatal("印の色: 小隊長と配下は同じ金、崩れは灰のはず")
		}
		for m, w := range map[CommandMark]string{CommanderMark: "小隊長", SubordinateMark: "配下", CollapsedMark: "崩れ", NoCommandMark: ""} {
			if m.Word() != w {
				t.Errorf("%d の語 %q、期待 %q", m, m.Word(), w)
			}
		}
		// 32×32 の絵の左上に置き、ボスの ★（右上 x ≧ 20）、装備の印（右下 x, y = 22〜31。DES-IT07 3.6 節）、HP の棒（下 5 px）と重ねない。
		cell := image.Pt(64, 96)
		r := CommandMarkRect(cell).Sub(cell)
		if !r.In(image.Rect(0, 0, CellSize/2, CellSize/2)) || r.Overlaps(image.Rect(CellSize-12, 0, CellSize, CellSize)) || r.Overlaps(image.Rect(22, 22, 32, 32)) || r.Overlaps(image.Rect(0, CellSize-5, CellSize, CellSize)) || r.Dx() < 8 {
			t.Fatalf("盤面の印の箱 %v（マスの左上からの相対）", r)
		}
		c := newIdleWith(t, commandSetup)
		want := []CommandMark{NoCommandMark, CommanderMark, SubordinateMark, NoCommandMark, NoCommandMark}
		if got := marksOf(c.Units()); !equalMarks(got, want) {
			t.Fatalf("初めの印 %v、期待 %v", got, want)
		}
	})

	t.Run("TC-FR-UNT-09-30 右欄の名前の横に同じ形の印と「小隊長／配下」の語を置き、ごろつき・ボス・自軍には置かない", func(t *testing.T) {
		c := newIdleWith(t, commandSetup)
		for _, tc := range []struct {
			pos  tactics.Pos
			mark CommandMark
			name string
		}{
			{tactics.Pos{X: 1, Y: 1}, CommanderMark, "兵 A"},
			{tactics.Pos{X: 6, Y: 1}, SubordinateMark, "兵 B"},
		} {
			m, name, word, ok := infoMarkAt(t, c, tc.pos)
			if !ok || m.Mark != tc.mark || name.Text != tc.name || word.Text != tc.mark.Word() {
				t.Fatalf("%s: 印 %v 名前 %q 語 %q", tc.name, m.Mark, name.Text, word.Text)
			}
			nameEnd := name.X + textWidth(name.Text)
			if m.Rect.Min.X <= nameEnd || m.Rect.Dx() != commandMarkSize || word.X < m.Rect.Max.X || word.X+textWidth(word.Text) > ScreenWidth-infoMargin {
				t.Fatalf("%s: 名前の右端 %d、印 %v、語 x=%d", tc.name, nameEnd, m.Rect, word.X)
			}
		}
		for _, p := range []tactics.Pos{{X: 0, Y: 1}, {X: 7, Y: 0}, {X: 7, Y: 2}} {
			if m, _, _, ok := infoMarkAt(t, c, p); ok {
				t.Fatalf("%v に印 %v", p, m.Mark)
			}
		}
	})

	t.Run("TC-FR-UNT-09-30 小隊長を倒すと、撃破の出来事の写しから配下が崩れの印に替わり、章の終わりまで残る。号令の文は出さない", func(t *testing.T) {
		c := newIdleWith(t, commandSetup)
		c.act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 1}, Action: tactics.Attack, Target: 1})
		fell := false
		for c.State() == Playing {
			e, _, _ := c.Event()
			if e.Kind == tactics.EventFall && e.Unit == 1 {
				fell = true
			}
			got := CommandMarkOf(c.Units()[2])
			want := SubordinateMark
			if fell {
				want = CollapsedMark
			}
			if got != want {
				t.Fatalf("出来事 %v（撃破のあと=%v）の配下の印 %v、期待 %v", e.Kind, fell, got, want)
			}
			for _, word := range []string{"崩れ", "号令", "小隊長"} {
				if strings.Contains(EventText(e), word) {
					t.Fatalf("出来事の文 %q に %q", EventText(e), word)
				}
			}
			c.Tick()
		}
		if !fell {
			t.Fatal("小隊長が倒れない")
		}
		if m, _, word, ok := infoMarkAt(t, c, c.Units()[2].Pos); !ok || m.Mark != CollapsedMark || word.Text != "崩れ" {
			t.Fatalf("右欄の配下の印 %v 語 %q", m.Mark, word.Text)
		}
		// 3 回の番を回しても崩れの印は残る（Q-DES07-10、E-120）。配下は移動 1 なので主人公に届かず、生きている。
		for i := 0; i < 3; i++ {
			c.EndPhase()
			playOut(c)
			u := c.Units()[2]
			if !u.Alive || CommandMarkOf(u) != CollapsedMark {
				t.Fatalf("%d 回目の番のあと: 生存 %v 印 %v", i+1, u.Alive, CommandMarkOf(u))
			}
		}
	})

	t.Run("TC-FR-UNT-09-30 巻き戻しの下見は、下見の盤面（撃破の前）の印で描く", func(t *testing.T) {
		c := newIdleWith(t, commandSetup)
		c.act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 1}, Action: tactics.Attack, Target: 1})
		playOut(c)
		if CommandMarkOf(c.Units()[2]) != CollapsedMark {
			t.Fatal("撃破のあとの配下が崩れでない")
		}
		// 一覧で小隊長を倒した行動の行を選ぶ（主人公だけなので、そのあとに敵軍の番の行が続く）。
		c.OpenRewind()
		h := c.battle.History()
		c.rewindSelect(historyIndex(t, h, tactics.HistoryAction, tactics.Player))
		v := c.RewindBoard()
		want := []CommandMark{NoCommandMark, CommanderMark, SubordinateMark, NoCommandMark, NoCommandMark}
		if !v.Available || !equalMarks(marksOf(v.Units), want) || !v.Units[1].Alive {
			t.Fatalf("下見の印 %v（小隊長の生存 %v）、期待 %v", marksOf(v.Units), v.Units[1].Alive, want)
		}
		// 最新の行（敵軍の番の前）の下見では、小隊長は倒れ、配下は崩れている。
		c.rewindSelect(len(h) - 1)
		if v := c.RewindBoard(); v.Units[1].Alive || CommandMarkOf(v.Units[2]) != CollapsedMark {
			t.Fatalf("最新の行の下見: 小隊長の生存 %v、配下の印 %v", v.Units[1].Alive, CommandMarkOf(v.Units[2]))
		}
		if CommandMarkOf(c.Units()[2]) != CollapsedMark {
			t.Fatal("下見を開いただけで本物の盤面の印が変わった")
		}
	})

	t.Run("TC-FR-UNT-09-30 出来事で小隊長が出られずに出た配下は、初めから崩れの印（DES-IT07 5.3 節。REV-101 F-04）", func(t *testing.T) {
		c := newIdleWith(t, reserveSetup)
		c.act(tactics.Command{Unit: 0, To: tactics.Pos{X: 0, Y: 1}, Action: tactics.Attack, Target: 1})
		playOut(c)
		c.EndPhase()
		playOut(c)
		var follower *tactics.Unit
		for i, u := range c.Units() {
			if u.Commander && u.Alive {
				t.Fatalf("出られないはずの小隊長が出た: %+v", u)
			}
			if u.ID == "foll" {
				follower = &c.Units()[i]
			}
		}
		if follower == nil || !follower.Alive {
			t.Fatal("配下が出ない")
		}
		if CommandMarkOf(*follower) != CollapsedMark {
			t.Fatalf("小隊長なしで出た配下の印 %v、期待は崩れ", CommandMarkOf(*follower))
		}
		if m, _, word, ok := infoMarkAt(t, c, follower.Pos); !ok || m.Mark != CollapsedMark || word.Text != "崩れ" {
			t.Fatalf("右欄の印 %v 語 %q", m.Mark, word.Text)
		}
	})
}

// reserveSetup は 10 × 3 の平地。主人公 (0, 1) の隣の見張り（HP 1、ID watch）を倒すと、次の敵軍の番の始めに
// 小隊長 (8, 1) と配下 (3, 2) の増援が出る。(8, 1) と距離 2 までのマスは動かない敵がふさぐので、小隊長は出られない。
// 盤面はつながっているので、小隊長と配下は同じ到達領域の一団（DES-IT07 5.1 節）。
//
//	0: 主人公 1: 見張り 2: ボス 3〜: ふさぐ敵
func reserveSetup() tactics.Setup {
	strong := tactics.Weapon{Name: "剣", Type: tactics.Sword, Might: 50, Hit: 300, MinRange: 1, MaxRange: 1}
	reserve := func(id string, x, y int) tactics.UnitSetup {
		return tactics.UnitSetup{Name: "増援 " + id, ID: id, Side: tactics.Enemy, Level: 1, AI: tactics.Hold, Approach: tactics.GroupApproach, Group: "r", Stats: tactics.Stats{HP: 30, Mov: 1}, Pos: tactics.Pos{X: x, Y: y}, Weapon: testLance}
	}
	leader := reserve("lead", 8, 1)
	leader.Commander = true
	units := []tactics.UnitSetup{
		{Name: "主人公", Side: tactics.Player, Lord: true, Stats: tactics.Stats{HP: 30, Mov: 3}, Pos: tactics.Pos{X: 0, Y: 1}, Weapon: strong},
		{Name: "見張り", ID: "watch", Side: tactics.Enemy, Level: 1, AI: tactics.Hold, Stats: tactics.Stats{HP: 1}, Pos: tactics.Pos{X: 1, Y: 1}, Weapon: testLance},
		{Name: "ボス", Side: tactics.Enemy, Level: 1, Boss: true, AI: tactics.Hold, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: 4, Y: 0}, Weapon: testLance},
	}
	for y := 0; y < 3; y++ {
		for x := 6; x < 10; x++ {
			if abs(x-8)+abs(y-1) <= 2 {
				units = append(units, tactics.UnitSetup{Name: "ふさぐ敵", Side: tactics.Enemy, Level: 1, AI: tactics.Hold, Stats: tactics.Stats{HP: 30}, Pos: tactics.Pos{X: x, Y: y}, Weapon: testLance})
			}
		}
	}
	return tactics.Setup{Terrain: field(10, 3), Units: units,
		Events: []tactics.MapEventSetup{{ID: "alarm", Trigger: tactics.TriggerEnemyDefeated, Unit: "watch", Effect: tactics.EffectReinforce, Enemies: []tactics.UnitSetup{leader, reserve("foll", 3, 2)}}}}
}

func equalMarks(a, b []CommandMark) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

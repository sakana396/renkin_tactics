package ui

import (
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// itemHelpController は、剛力薬と粘着瓶を持つ自軍を移動先に置き、アイテムの一覧を開いた Controller を返す。
func itemHelpController(t *testing.T, help bool) *Controller {
	t.Helper()
	c := newIdleWith(t, itemNG06Setup)
	c.itemHelp = help
	c.Click(tactics.Pos{X: 2, Y: 2})
	c.Click(tactics.Pos{X: 2, Y: 2})
	c.ChooseAction(tactics.UseItem)
	if c.State() != ItemMenu {
		t.Fatalf("状態 %v", c.State())
	}
	return c
}

// hoverItemRow は一覧の i 行目の中心にカーソルを置く。
func hoverItemRow(c *Controller, i int) {
	r := c.Camera().MenuRectsFor(c.dest, c.MenuLabels())[i]
	c.SetCursor((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

func TestItemHelp227(t *testing.T) {
	t.Run("TC-DES-NG-16-03 道具の説明の文は効果・距離・持続を、itemEffect と同じ言い方で書く", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			it   tactics.Item
			want string
		}{
			{"剛力薬", tactics.Item{Effect: "power_tonic", Amount: 4, Range: 3}, "味方 1 人の力・魔力 +4　距離 0〜3　次の自軍開始まで"},
			{"粘着瓶", tactics.Item{Effect: "snare", Range: 4}, "敵 1 体の移動を 0 にする　距離 1〜4　次の自軍開始まで"},
			{"爆弾", tactics.Item{Bomb: true, BlastDamage: 12, BlastRadius: 2, ThrowRange: 3}, "爆心から距離 2 までの全員にダメージ 12（敵味方の区別なし）　投げる 3"},
			{"回復薬", tactics.Item{Heal: 10}, "自分の HP を 10 回復する"},
		} {
			if got := itemDescription(tc.it); got != tc.want {
				t.Errorf("%s: %q、期待 %q", tc.name, got, tc.want)
			}
		}
	})
	t.Run("TC-DES-NG-16-04 一覧で指した道具・選んだ道具の説明を案内欄に出す。指していなければ出さない。旧作は出さない", func(t *testing.T) {
		tonic, snare := itemDescription(itemNG06Setup().Units[0].Items[0].Item), itemDescription(itemNG06Setup().Units[0].Items[1].Item)
		c := itemHelpController(t, true)
		last := func() string { g := c.Guide(); return g[len(g)-1] }
		if g := c.Guide(); len(g) != 3 {
			t.Fatalf("指していないのに行が増えた: %v", g)
		}
		hoverItemRow(c, 0)
		if last() != tonic {
			t.Fatalf("1 行目を指す: %q", last())
		}
		hoverItemRow(c, 1)
		if last() != snare {
			t.Fatalf("2 行目を指す: %q", last())
		}
		// 一覧の外へカーソルを動かすと消える。
		c.SetCursor(0, 0)
		if g := c.Guide(); len(g) != 3 {
			t.Fatalf("一覧の外: %v", g)
		}
		// 選んだ道具（対象を選ぶ間）。
		c.ChooseItem(1)
		if c.State() != ItemTargeting {
			t.Fatalf("状態 %v", c.State())
		}
		if last() != snare {
			t.Fatalf("選んだ道具: %q", last())
		}
		// 旧作では出さない。
		old := itemHelpController(t, false)
		hoverItemRow(old, 0)
		if g := old.Guide(); len(g) != 3 || strings.Contains(strings.Join(g, ""), "味方 1 人") {
			t.Fatalf("旧作に説明が出た: %v", g)
		}
	})
	t.Run("TC-DES-NG-16-05 説明を足しても案内欄は 4 行まで・幅に収まり、右の案内と重ならない（高解像度）", func(t *testing.T) {
		useTestMetrics(t, HiResCellSize)
		c := itemHelpController(t, true)
		max := GuideHeight / lineHeight
		for k := 0; k < 2; k++ {
			hoverItemRow(c, k)
			check := func(what string) {
				g := c.Guide()
				if len(g) > max {
					t.Fatalf("%s: %d 行", what, len(g))
				}
				// 左の案内は、右端の「矢印／ホイールで盤面移動」（右欄の x）に届かない。
				for _, l := range g {
					if w := 8 + textWidth(l); w > MapSize-16 {
						t.Fatalf("%s: 幅 %d が盤面の幅 %d を超える: %s", what, w, MapSize, l)
					}
				}
			}
			check("一覧")
			c.ChooseItem(k)
			check("対象を選ぶ")
			c.Click(tactics.Pos{X: 4, Y: 2})
			check("対象を選んだあと")
			c.Cancel()
			c.Cancel()
		}
		// 使えない道具の説明（4 行目まで埋まる message があるとき）は、行を増やさない。
		c.message = "使える対象がいません"
		hoverItemRow(c, 0)
		if g := c.Guide(); len(g) > max {
			t.Fatalf("message あり: %d 行", len(g))
		}
	})
}

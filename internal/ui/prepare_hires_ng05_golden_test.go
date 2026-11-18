package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core"
)

// 以下は TC-DES-NG-11-17 の期待値。HiRes でない準備の SceneView を、改修前（コミット f864969）の
// 実装の出力から取った値（文・ボタン・タブ・行・絵・情報欄の座標と文面）。HiRes の準備を足したあとも、これと一致しなければならない。
// 状態: 名簿のタブ / 持ち物のタブ（ダイン、在庫に鉄の斧を足す）/ 在庫の品を選んだ状態 3 つ / 案内の理由が出ている状態。

// prepareViewDigest は SceneView の中身を、比べやすい 1 つの文にする。
func prepareViewDigest(v SceneView) string {
	var b strings.Builder
	for _, t := range v.Texts {
		fmt.Fprintf(&b, "T %d,%d,%v,%d %s\n", t.X, t.Y, t.Big, t.Style, t.Text)
	}
	for _, x := range v.Buttons {
		fmt.Fprintf(&b, "B %v %v %s\n", x.Rect, x.Enabled, x.Label)
	}
	for _, x := range v.Tabs {
		fmt.Fprintf(&b, "TAB %v %s %d %v\n", x.Rect, x.Label, x.Kind, x.Selected)
	}
	for _, x := range v.Rows {
		fmt.Fprintf(&b, "R %d %v %v %v\n", x.Member, x.Rect, x.Mark, x.Selected)
	}
	for _, x := range v.Icons {
		fmt.Fprintf(&b, "I %d,%d,%d,%v\n", x.X, x.Y, x.Size, x.Face)
	}
	fmt.Fprintf(&b, "info=%v area=%v\n", v.Info != nil, v.InfoArea)
	if v.Info != nil {
		fmt.Fprintf(&b, "infotexts=%d bars=%d icons=%d tabs=%d\n", len(v.Info.Texts), len(v.Info.Bars), len(v.Info.Icons), len(v.Info.Tabs))
		for _, t := range v.Info.Texts {
			fmt.Fprintf(&b, "IT %d,%d %s\n", t.X, t.Y, t.Text)
		}
	}
	return b.String()
}

// prepareViewStates は、HiRes でない準備の代表的な状態ごとの SceneView の文を、上の順に返す。
func prepareViewStates(t *testing.T) string {
	t.Helper()
	s := ng05Scenes(t, "ch01", false)
	var out strings.Builder
	out.WriteString("== roster\n" + prepareViewDigest(s.View()))
	s.campaign.StockWeapons = append(s.campaign.StockWeapons, core.WeaponCount{Weapon: "iron_axe", Count: 1})
	s.prepareMember = 1
	s.prepareTab = TabItems
	out.WriteString("== items\n" + prepareViewDigest(s.View()))
	for _, l := range s.prepareStock() {
		if l.heading == "" {
			s.prepareItem = l.item
			out.WriteString("== select " + l.name + "\n" + prepareViewDigest(s.View()))
		}
	}
	s.message = "手持ちは 5 枠までです"
	out.WriteString("== message\n" + prepareViewDigest(s.View()))
	return out.String()
}

const prepareViewGolden = `== roster
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 20,68,false,0 [✓]
T 96,68,false,0 アオイ　錬金術師　Lv 1　HP 17　手持ち 5/5
T 20,92,false,0 [✓]
T 96,92,false,0 ダイン　重装兵　Lv 1　HP 24　手持ち 2/5
T 20,116,false,0 [✓]
T 96,116,false,0 トーリ　槍兵　Lv 1　HP 18　手持ち 2/5
T 20,140,false,0 [✓]
T 96,140,false,0 ネム　弓兵　Lv 1　HP 15　手持ち 2/5
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 true
TAB (260,16)-(496,48) 持ち物 1 false
R 0 (56,64)-(400,88) (16,64)-(52,88) true
R 1 (56,88)-(400,112) (16,88)-(52,112) false
R 2 (56,112)-(400,136) (16,112)-(52,136) false
R 3 (56,136)-(400,160) (16,136)-(52,160) false
I 60,64,24,true
I 60,88,24,true
I 60,112,24,true
I 60,136,24,true
info=true area=(416,56)-(816,472)
infotexts=15 bars=8 icons=3 tabs=2
IT 496,72 アオイ
IT 496,88 クラス: 錬金術師
IT 496,104 Lv 1　経験 0
IT 424,132 HP 17/17
IT 424,148 力 2
IT 424,164 魔力 4
IT 424,180 技 5
IT 424,196 速さ 6
IT 424,212 幸運 4
IT 424,228 守備 2
IT 424,244 魔防 3
IT 424,260 移動 5
IT 424,280 E
IT 454,280 ファイア
IT 606,280 癒やしの杖
== items
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 16,60,false,1 ダイン（重装兵）　手持ち 2/5
T 20,94,false,1 E
T 16,252,false,2 E: 初めの装備（武器の列の先頭）
T 16,272,false,2 右クリック・Esc: 品の選択 → 名簿へ
T 416,60,false,1 在庫
T 416,86,false,2 武器
T 416,134,false,2 道具
B (40,88)-(232,116) true 在庫へ: 鉄の槍
B (240,88)-(400,116) false 初めの装備
B (40,120)-(232,148) true 在庫へ: 回復薬
B (40,152)-(232,180) false ―
B (40,184)-(232,212) false ―
B (40,216)-(232,244) false ―
B (432,108)-(800,128) true 手持ちへ: 鉄の斧
B (432,156)-(800,176) true 手持ちへ: 小さな爆弾
B (432,180)-(800,200) true 手持ちへ: 回復薬 ×3
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 false
TAB (260,16)-(496,48) 持ち物 1 true
info=false area=(0,0)-(0,0)
== select 鉄の斧
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 16,60,false,1 ダイン（重装兵）　手持ち 2/5
T 20,94,false,1 E
T 16,252,false,2 E: 初めの装備（武器の列の先頭）
T 16,272,false,2 右クリック・Esc: 品の選択 → 名簿へ
T 416,60,false,1 在庫
T 416,86,false,2 武器
T 416,134,false,2 道具
T 416,360,false,1 選んだ品: 鉄の斧
T 416,376,false,0 鉄の斧　物理　威力 8　命中 70　必殺 0　重さ 7　射程 1　特効 なし
T 416,392,false,5 ダイン（重装兵）は斧を装備できない
B (40,88)-(232,116) true 在庫へ: 鉄の槍
B (240,88)-(400,116) false 初めの装備
B (40,120)-(232,148) true 在庫へ: 回復薬
B (40,152)-(232,180) false ―
B (40,184)-(232,212) false ―
B (40,216)-(232,244) false ―
B (432,108)-(800,128) true 手持ちへ: 鉄の斧
B (432,156)-(800,176) true 手持ちへ: 小さな爆弾
B (432,180)-(800,200) true 手持ちへ: 回復薬 ×3
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 false
TAB (260,16)-(496,48) 持ち物 1 true
info=false area=(0,0)-(0,0)
== select 小さな爆弾
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 16,60,false,1 ダイン（重装兵）　手持ち 2/5
T 20,94,false,1 E
T 16,252,false,2 E: 初めの装備（武器の列の先頭）
T 16,272,false,2 右クリック・Esc: 品の選択 → 名簿へ
T 416,60,false,1 在庫
T 416,86,false,2 武器
T 416,134,false,2 道具
T 416,360,false,1 選んだ品: 小さな爆弾
T 416,376,false,0 道具　爆弾 ダメージ 8　範囲 1　投げる 4
B (40,88)-(232,116) true 在庫へ: 鉄の槍
B (240,88)-(400,116) false 初めの装備
B (40,120)-(232,148) true 在庫へ: 回復薬
B (40,152)-(232,180) false ―
B (40,184)-(232,212) false ―
B (40,216)-(232,244) false ―
B (432,108)-(800,128) true 手持ちへ: 鉄の斧
B (432,156)-(800,176) true 手持ちへ: 小さな爆弾
B (432,180)-(800,200) true 手持ちへ: 回復薬 ×3
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 false
TAB (260,16)-(496,48) 持ち物 1 true
info=false area=(0,0)-(0,0)
== select 回復薬
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 16,60,false,1 ダイン（重装兵）　手持ち 2/5
T 20,94,false,1 E
T 16,252,false,2 E: 初めの装備（武器の列の先頭）
T 16,272,false,2 右クリック・Esc: 品の選択 → 名簿へ
T 416,60,false,1 在庫
T 416,86,false,2 武器
T 416,134,false,2 道具
T 416,360,false,1 選んだ品: 回復薬
T 416,376,false,0 道具　HP を 10 回復
B (40,88)-(232,116) true 在庫へ: 鉄の槍
B (240,88)-(400,116) false 初めの装備
B (40,120)-(232,148) true 在庫へ: 回復薬
B (40,152)-(232,180) false ―
B (40,184)-(232,212) false ―
B (40,216)-(232,244) false ―
B (432,108)-(800,128) true 手持ちへ: 鉄の斧
B (432,156)-(800,176) true 手持ちへ: 小さな爆弾
B (432,180)-(800,200) true 手持ちへ: 回復薬 ×3
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 false
TAB (260,16)-(496,48) 持ち物 1 true
info=false area=(0,0)-(0,0)
== message
T 512,24,false,1 第 1 章　水の止まった都　出撃 4/6
T 16,60,false,1 ダイン（重装兵）　手持ち 2/5
T 20,94,false,1 E
T 16,252,false,2 E: 初めの装備（武器の列の先頭）
T 16,272,false,2 右クリック・Esc: 品の選択 → 名簿へ
T 416,60,false,1 在庫
T 416,86,false,2 武器
T 416,134,false,2 道具
T 416,360,false,1 選んだ品: 回復薬
T 416,376,false,0 道具　HP を 10 回復
T 16,516,false,5 手持ちは 5 枠までです
B (40,88)-(232,116) true 在庫へ: 鉄の槍
B (240,88)-(400,116) false 初めの装備
B (40,120)-(232,148) true 在庫へ: 回復薬
B (40,152)-(232,180) false ―
B (40,184)-(232,212) false ―
B (40,216)-(232,244) false ―
B (432,108)-(800,128) true 手持ちへ: 鉄の斧
B (432,156)-(800,176) true 手持ちへ: 小さな爆弾
B (432,180)-(800,200) true 手持ちへ: 回復薬 ×3
B (672,528)-(816,568) true 配置へ
TAB (16,16)-(252,48) 名簿 0 false
TAB (260,16)-(496,48) 持ち物 1 true
info=false area=(0,0)-(0,0)
`

package core_test

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 敵軍の番のルール計算の時間の予備測定（DES-NFR-04、DES-IT05 8.1 節。受入の測定は #105 の TC-NFR-02-01）。
// 時間の閾値で合否を決める単体テストにはしないので、RENKIN_MEASURE を設定したときだけ動き、結果を記録に出す。
// 時計はテストの側に置き、core には実時間を入れない。
//
//	RENKIN_MEASURE=1 go test ./internal/core -run TestMeasureCh09EnemyPhase -v
func TestMeasureCh09EnemyPhase(t *testing.T) {
	if os.Getenv("RENKIN_MEASURE") == "" {
		t.Skip("RENKIN_MEASURE を設定したときだけ測る")
	}
	for _, tc := range []struct {
		name   string
		charge bool
	}{
		{"本体のまま（突っ込む 3 体＋増援、ほかは待ち構える）", false},
		{"全敵が突っ込む（試験用に ai を書き換え）", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cat := measureCatalog(t, tc.charge)
			var worst, total time.Duration
			var worstAt string
			count := 0
			for seed := uint64(1); seed <= 30; seed++ {
				c, err := core.NewCampaignAt(cat, core.Withdraw, "ch09")
				if err != nil {
					t.Fatal(err)
				}
				// 自軍は倒れないように固くし、出発地で待つ（敵が全員集まり、攻撃の候補が増える）。
				for k := range c.Roster {
					c.Roster[k].Stats.HP, c.Roster[k].Stats.Def = 99, 99
				}
				s, _, err := c.StartBattle([]int{0, 1, 2, 3, 4, 5}, seed)
				if err != nil {
					t.Fatal(err)
				}
				b := s.Battle
				for b.Result == tactics.Ongoing && b.Turn <= 15 {
					alive := 0
					for _, u := range b.Units() {
						if u.Alive {
							alive++
						}
					}
					start := time.Now()
					if _, err := b.EndPhase(); err != nil {
						t.Fatal(err)
					}
					d := time.Since(start)
					total += d
					count++
					if d > worst {
						worst, worstAt = d, fmt.Sprintf("seed %d・%d ターン目・生存 %d 体", seed, b.Turn-1, alive)
					}
				}
			}
			t.Logf("敵軍の番 %d 回: 平均 %v、最大 %v（%s）", count, total/time.Duration(count), worst, worstAt)
		})
	}
}

// measureCatalog は本体の目録。charge なら ch09 の敵をすべて突っ込むにする（測定用の書き換え）。
func measureCatalog(t *testing.T, charge bool) *defs.Catalog {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(data.Base(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), p)
		m[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if charge {
		s := string(m["maps.toml"].Data)
		i := strings.Index(s, "id = \"ch09\"")
		m["maps.toml"].Data = []byte(s[:i] + strings.ReplaceAll(s[i:], `ai = "hold"`, `ai = "charge"`))
	}
	cat, err := defload.Load(m)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

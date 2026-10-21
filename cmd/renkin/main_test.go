package main

import (
	"bytes"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/ui"
)

// copyBase は本体のデータを写した fstest.MapFS を返す。
func copyBase(t *testing.T) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	err := fs.WalkDir(data.Base(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(data.Base(), path)
		if err != nil {
			return err
		}
		out[path] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func fixedNow() time.Time { return time.Unix(0, 12345) }

// endPhaseUntilOver は、自軍が番を終えるだけを 30 ターンまで（戦闘が終わるまで）続け、出来事をすべて返す。
// 敵の攻撃で乱数を引くので、種が違えば出来事も違う。
func endPhaseUntilOver(t *testing.T, b *tactics.Battle) []tactics.Event {
	t.Helper()
	var all []tactics.Event
	for i := 0; i < 30 && b.Result == tactics.Ongoing; i++ {
		ev, err := b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, ev...)
	}
	return all
}

// 起動の流れ（DES-CMD-01）を、画面を開かずに確かめる。
func TestRun(t *testing.T) {
	t.Run("TC-FR-SYS-04-22 定義データに誤りがあれば、すべてを標準エラーに書き、終了コード 1 で終える", func(t *testing.T) {
		fsys := copyBase(t)
		fsys["game.toml"] = &fstest.MapFile{Data: []byte("start_map = \"ch99\"\n")}
		fsys["items.toml"] = &fstest.MapFile{Data: []byte("[[items]]\nid = \"potion\"\nname = \"回復薬\"\nheal = 0\n")}
		var stderr bytes.Buffer
		started := false
		code := run(nil, &stderr, fsys, fixedNow, func(ui.Factory) error {
			started = true
			return nil
		})
		if code != 1 || started {
			t.Fatalf("終了コード %d、画面を動かした %v", code, started)
		}
		out := stderr.String()
		for _, want := range []string{"定義データに誤りがあります", "game.toml", "ch99", "items.toml: items[0].heal"} {
			if !strings.Contains(out, want) {
				t.Errorf("標準エラーに %q がない:\n%s", want, out)
			}
		}
	})

	t.Run("TC-FR-SYS-04-23 定義データに誤りがなければ、始めるマップの戦闘を作る関数を渡して画面を動かす", func(t *testing.T) {
		var stderr bytes.Buffer
		var factory ui.Factory
		code := run([]string{"-seed", "7"}, &stderr, data.Base(), fixedNow, func(f ui.Factory) error {
			factory = f
			return nil
		})
		if code != 0 || factory == nil || stderr.Len() != 0 {
			t.Fatalf("終了コード %d、標準エラー %q", code, stderr.String())
		}
		b, events, err := factory()
		if err != nil || b.Turn != 1 || b.Phase != tactics.Player || len(b.Units()) != 12 || len(events) != 1 {
			t.Errorf("作った戦闘: 誤り %v、ターン %d、ユニット %d、出来事 %d", err, b.Turn, len(b.Units()), len(events))
		}
		// 画面が誤りを返せば、標準エラーに書いて終了コード 1。
		stderr.Reset()
		code = run(nil, &stderr, data.Base(), fixedNow, func(ui.Factory) error { return errors.New("窓を開けない") })
		if code != 1 || !strings.Contains(stderr.String(), "窓を開けない") {
			t.Errorf("終了コード %d、標準エラー %q", code, stderr.String())
		}
	})
}

// 乱数の種の決め方（DES-RNG-01、DES-CMD-01）。
func TestSeed(t *testing.T) {
	t.Run("TC-IF-01-17 やり直しの種: -seed を指定すれば毎回同じ種、なければ呼んだときの時刻から作る。-seed が数でなければ起動しない", func(t *testing.T) {
		fixed := seeder(true, 7, fixedNow)
		if fixed() != 7 || fixed() != 7 {
			t.Errorf("-seed 7 の種が 7 でない")
		}
		clock := int64(100)
		now := func() time.Time { clock++; return time.Unix(0, clock) }
		byTime := seeder(false, 0, now)
		if a, b := byTime(), byTime(); a != 101 || b != 102 {
			t.Errorf("時刻からの種 = %d, %d", a, b)
		}
		// run が -seed を使う: 毎回値の変わる now を渡しても、-seed 7 で作った戦闘はどれも種 7 の戦闘と同じ（NFR-06）。
		calls := 0
		ticking := func() time.Time { calls++; return time.Unix(0, int64(1000+calls)) }
		var factory ui.Factory
		run([]string{"-seed", "7"}, &bytes.Buffer{}, data.Base(), ticking, func(f ui.Factory) error { factory = f; return nil })
		cat, err := defload.Load(data.Base())
		if err != nil {
			t.Fatal(err)
		}
		want, _, _ := core.NewBattle(cat, "ch01", 7)
		wantEvents := endPhaseUntilOver(t, want)
		for i := 0; i < 2; i++ {
			b, _, err := factory()
			if err != nil {
				t.Fatal(err)
			}
			if got := endPhaseUntilOver(t, b); !reflect.DeepEqual(got, wantEvents) {
				t.Errorf("-seed 7 の %d 回目の戦闘が、種 7 の戦闘と違う", i+1)
			}
		}
		if calls != 0 {
			t.Errorf("-seed を指定したのに now を %d 回呼んだ", calls)
		}
		// -seed なしなら、作るたびに now を呼ぶ。
		run(nil, &bytes.Buffer{}, data.Base(), ticking, func(f ui.Factory) error { factory = f; return nil })
		for i := 1; i <= 2; i++ {
			if _, _, err := factory(); err != nil || calls != i {
				t.Errorf("-seed なしで %d 回作ったのに now を %d 回呼んだ（誤り %v）", i, calls, err)
			}
		}
		var stderr bytes.Buffer
		started := false
		if code := run([]string{"-seed", "abc"}, &stderr, data.Base(), fixedNow, func(ui.Factory) error { started = true; return nil }); code != 2 || started {
			t.Errorf("-seed abc: 終了コード %d、画面を動かした %v", code, started)
		}
	})
}

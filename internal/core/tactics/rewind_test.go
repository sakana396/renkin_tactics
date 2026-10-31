package tactics

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// 巻き戻しの試験は SetRoll を使わず、本番と同じ PCG を通す（DES-RWD-03）。
func rewindAct(t *testing.T, b *Battle, cmd Command) []Event {
	t.Helper()
	ev, err := b.Act(cmd)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func rewindTo(t *testing.T, b *Battle, id uint64) {
	t.Helper()
	if err := b.Rewind(id); err != nil {
		t.Fatal(err)
	}
}

func TestRewindAction(t *testing.T) {
	t.Run("TC-FR-BTL-10-01 移動・反撃・追撃と生死を行動前へ戻す", func(t *testing.T) {
		for _, lethal := range []bool{false, true} {
			b := snapshotBattle(t)
			b.units[0].Weapons[0].Hit = 1000
			b.units[2].Weapons[0].Hit = 1000
			b.units[2].Skill = Skill{}
			if lethal {
				b.units[2].HP = 1
			}
			before := b.snapshot()
			ev := rewindAct(t, b, Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2})
			strikes := 0
			for _, e := range ev {
				if e.Kind == EventStrike {
					strikes++
				}
			}
			if !lethal && strikes != 3 {
				t.Fatalf("攻撃・反撃・追撃の回数 = %d", strikes)
			}
			if lethal && (b.Result != Won || b.units[2].Alive) {
				t.Fatal("ボス撃破を通っていない")
			}
			h := b.History()
			rewindTo(t, b, h[0].ID)
			if !reflect.DeepEqual(b.snapshot(), before) || len(b.History()) != 0 || b.NeedsResume() {
				t.Fatal("行動前の盤面・添字・出来事境界に戻らない")
			}
		}
	})
	t.Run("TC-FR-BTL-10-02 成長で増えた HP と能力を戻す", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[0].Growth = Growth{100, 100, 100, 100, 100, 0, 0, 0}
		b.units[2].Stats.Str = 0
		b.units[2].Weapons[0].Might = 0
		b.units[0].Weapons[0].Hit = 1000
		before := b.Units()
		rewindAct(t, b, Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2})
		u := b.units[0]
		wantStats := before[0].Stats
		wantStats.HP++
		wantStats.Str++
		wantStats.Skl++
		wantStats.Spd++
		wantStats.Def++
		if u.Level != 2 || u.Stats != wantStats || u.HP <= before[0].HP {
			t.Fatal("全能力と今の HP の成長を通っていない")
		}
		rewindTo(t, b, b.History()[0].ID)
		if !reflect.DeepEqual(b.Units(), before) {
			t.Fatal("成長前へ戻らない")
		}
	})
	t.Run("TC-FR-BTL-10-03 薬と杖・経験・回復 HP を戻す", func(t *testing.T) {
		for _, action := range []Action{UseItem, Heal} {
			b := snapshotBattle(t)
			b.units[0].HP = 40
			b.units[1].HP = 40
			if action == Heal {
				b.units[0].setWeapon(Weapon{Name: "杖", Type: Staff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 3, MaxUses: 3})
			}
			before := b.Units()
			rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: action, Target: 1, Item: 0})
			if action == Heal && (b.units[0].Weapons[0].Uses != 2 || b.units[1].HP != 50 || b.units[0].Level != 2) {
				t.Fatal("杖の消費・回復・経験を通っていない")
			}
			if action == UseItem && (b.units[0].Items[0].Count != 1 || b.units[0].HP != 50) {
				t.Fatal("薬の消費と回復を通っていない")
			}
			rewindTo(t, b, b.History()[0].ID)
			if !reflect.DeepEqual(b.Units(), before) {
				t.Fatal("手持ち・杖・経験・HP が戻らない")
			}
		}
	})
}

func TestRewindResume(t *testing.T) {
	t.Run("TC-FR-BTL-10-04 敵 B から再開し敵 A と Hold を重ねない", func(t *testing.T) {
		for _, waiting := range []bool{false, true} {
			b := snapshotBattle(t)
			b.units[0].Pos = Pos{1, 0}
			enemy := b.units[2].clone()
			enemy.Pos = Pos{2, 1}
			enemy.Name = "敵 B"
			if !waiting {
				enemy.Stats.Mov = 1
			}
			b.units = append(b.units, enemy)
			if waiting {
				b.units[0].Pos = Pos{0, 0}
			}
			cmd := Command{Unit: 1, To: Pos{0, 1}, Action: Wait}
			rewindAct(t, b, cmd)
			boundary := b.eventCount
			ev, err := b.EndPhase()
			if err != nil {
				t.Fatal(err)
			}
			after := b.snapshot()
			var h HistoryEntry
			for _, e := range b.History() {
				if e.Kind == HistoryAction && e.Unit == 3 {
					h = e
				}
			}
			if h.ID == 0 || (waiting && h.Action != Wait) || (!waiting && b.units[3].AI != Charge) {
				t.Fatal("敵 B の待機・Hold 変化を通っていない")
			}
			restoreHistoryForResume(t, b, h.ID)
			if !b.NeedsResume() || b.units[3].AI != Hold {
				t.Fatal("敵 B の行動前に戻らない")
			}
			if _, err = b.Act(Command{Unit: 0, Action: Wait}); !errors.Is(err, ErrNotPlayerPhase) {
				t.Fatal(err)
			}
			if _, err = b.EndPhase(); !errors.Is(err, ErrNotPlayerPhase) {
				t.Fatal(err)
			}
			got, err := b.Resume()
			if err != nil {
				t.Fatal(err)
			}
			// 出来事の通し添字から、番終了前までの境界を引く。
			if !reflect.DeepEqual(got, ev[h.EventStart-boundary:]) || !reflect.DeepEqual(b.snapshot(), after) || b.NeedsResume() {
				t.Fatal("敵 A を重ねたか、敵 B からの展開が変わった")
			}
			rewindTo(t, b, b.History()[0].ID)
			rewindAct(t, b, cmd)
			again, err := b.EndPhase()
			if err != nil || !reflect.DeepEqual(again, ev) || !reflect.DeepEqual(after, b.snapshot()) {
				t.Fatal("自軍行動からの公開巻き戻しの再現が不正")
			}
		}
	})
	t.Run("TC-FR-BTL-10-05 番境界・増援・砦・自動移行の再開", func(t *testing.T) {
		for _, automatic := range []bool{false, true} {
			for _, blocked := range []bool{false, true} {
				for _, kind := range []HistoryKind{HistoryEndPhase, HistoryAction, HistoryPlayerStart} {
					b := snapshotBattle(t)
					b.terrain[1][0].HealPercent = 20
					b.terrain[0][2].HealPercent = 20
					b.units[1].HP = 40
					b.units[2].HP = 40
					b.reinforcements[0].Turn = 1
					if blocked {
						b.reinforcements[0].Unit.Pos = b.units[2].Pos
					}
					var ev []Event
					if automatic {
						rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
						ev = rewindAct(t, b, Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
					} else {
						var err error
						ev, err = b.EndPhase()
						if err != nil {
							t.Fatal(err)
						}
					}
					after := b.snapshot()
					var h HistoryEntry
					for _, e := range b.History() {
						if e.Kind == kind && (kind != HistoryAction || e.Side == Enemy) {
							h = e
							break
						}
					}
					if h.ID == 0 {
						t.Fatal("番境界がない")
					}
					restoreHistoryForResume(t, b, h.ID)
					if b.NeedsResume() != (automatic || kind != HistoryEndPhase) {
						t.Fatal("入力待ち・再開待ちの区別が不正")
					}
					if kind == HistoryEndPhase {
						if b.Phase != Player || b.Turn != 1 || b.units[2].HP != 40 {
							t.Fatal("敵番開始前へ戻らない")
						}
						if automatic {
							if _, err := b.EndPhase(); !errors.Is(err, ErrResumeRequired) {
								t.Fatal(err)
							}
							if _, err := b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait}); !errors.Is(err, ErrResumeRequired) {
								t.Fatal(err)
							}
						}
					}
					if kind == HistoryPlayerStart && (b.Turn != 1 || b.units[1].HP != 40) {
						t.Fatal("自軍番開始前へ戻らない")
					}
					var got []Event
					var err error
					if !automatic && kind == HistoryEndPhase {
						got, err = b.EndPhase()
					} else {
						got, err = b.Resume()
					}
					if err != nil {
						t.Fatal(err)
					}
					// 元の処理で返った出来事から、選んだ境界以降だけが再発生する。
					offset := h.EventStart - (after.eventCount - len(ev))
					if !reflect.DeepEqual(got, ev[offset:]) || !reflect.DeepEqual(b.snapshot(), after) {
						t.Fatal("番処理を重ねたか省略した")
					}
					wantUnits := 4
					wantPos := Pos{2, 1}
					if blocked {
						wantPos = Pos{1, 0}
					}
					if b.units[3].Pos != wantPos {
						t.Fatal("占有時も固定順の代替位置へ出ない")
					}
					if len(b.units) != wantUnits || b.Turn != 2 || b.units[1].HP != 60 || b.units[1].Acted {
						t.Fatal("増援・砦・Turn・Acted が不正")
					}
					for _, e := range b.History() {
						if e.Side == Enemy && e.Unit == 3 {
							t.Fatal("その番の増援を動かした")
						}
					}
				}
			}
		}
	})
}

func TestRewindRandomAndCount(t *testing.T) {
	t.Run("TC-FR-BTL-10-06 公開巻き戻しで大盾・成長の乱数を再現", func(t *testing.T) {
		run := func() ([]Event, battleSnapshot) {
			b := snapshotBattle(t)
			b.units[0].Weapons[0].Hit = 1000
			b.units[2].Skill.Chance = 100
			// 先行する公開の回復行動の成長で PCG を進め、初期 seed への置換を検出する。
			b.units[0].HP = 90
			b.units[1].setWeapon(Weapon{Name: "杖", Type: Staff, MinRange: 1, MaxRange: 1, Heal: 10, Uses: 3, MaxUses: 3})
			rewindAct(t, b, Command{Unit: 1, To: Pos{0, 1}, Action: Heal, Target: 0})
			cmd := Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2}
			first := rewindAct(t, b, cmd)
			pavise, level := false, false
			for _, e := range first {
				pavise = pavise || e.Pavise
				level = level || e.Kind == EventLevelUp
			}
			if !pavise || !level {
				t.Fatal("大盾の乱数判定と成長を通っていない")
			}
			after := b.snapshot()
			for i := 0; i < 3; i++ {
				rewindTo(t, b, b.History()[1].ID)
				got := rewindAct(t, b, cmd)
				if !reflect.DeepEqual(got, first) || !reflect.DeepEqual(b.snapshot(), after) {
					t.Fatal("公開 API で乱数が戻らない")
				}
			}
			return first, b.snapshot()
		}
		a, sa := run()
		z, sz := run()
		if !reflect.DeepEqual(a, z) || !reflect.DeepEqual(sa, sz) {
			t.Fatal("同じ seed と巻き戻し入力列が不一致")
		}
	})
	t.Run("TC-FR-BTL-10-07 一手も複数手も一回消費し十回で使い切る", func(t *testing.T) {
		b := snapshotBattle(t)
		// 残り 10 回の間に A・B を記録し、古い A の写しでも回数が戻らないことを確かめる（DES-RWD-04）。
		rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		aID := b.History()[0].ID
		rewindAct(t, b, Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
		bID := b.History()[1].ID
		rewindTo(t, b, bID)
		if b.RewindsLeft() != 9 {
			t.Fatalf("B の前へ戻した残り = %d, 期待 9", b.RewindsLeft())
		}
		rewindTo(t, b, aID)
		if b.RewindsLeft() != 8 {
			t.Fatalf("古い A の前へ戻した残り = %d, 期待 8", b.RewindsLeft())
		}
		for i := 2; i < 10; i++ {
			rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
			id := b.History()[0].ID
			if i%2 == 0 {
				if _, err := b.EndPhase(); err != nil {
					t.Fatal(err)
				}
			}
			rewindTo(t, b, id)
			if b.RewindsLeft() != 9-i {
				t.Fatalf("残り %d", b.RewindsLeft())
			}
		}
		rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		if !errors.Is(b.Rewind(b.History()[0].ID), ErrNoRewinds) {
			t.Fatal("十一回目が成功した")
		}
	})
}

// assertRewindFailure は公開操作の失敗時に、PCG・履歴・ID・回数・続きまで比較する。
func assertRewindFailure(t *testing.T, b *Battle, want error, f func() error) {
	t.Helper()
	before := b.snapshot()
	history := slices.Clone(b.history)
	id, left := b.nextHistoryID, b.RewindsLeft()
	err := f()
	if !errors.Is(err, want) {
		t.Fatalf("誤り = %v, 期待 %v", err, want)
	}
	if !reflect.DeepEqual(before, b.snapshot()) || !reflect.DeepEqual(history, b.history) || id != b.nextHistoryID || left != b.RewindsLeft() {
		t.Fatal("失敗時に戦闘・PCG・履歴・回数・続きが変わった")
	}
}

func TestRewindErrorsAndBranch(t *testing.T) {
	t.Run("TC-FR-BTL-10-08 不正な巻き戻しと再開は状態不変", func(t *testing.T) {
		for _, bad := range []string{"ないID", "未実行", "復元失敗", "残り0優先"} {
			b := snapshotBattle(t)
			rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
			id := b.History()[0].ID
			want := ErrInvalidRewind
			switch bad {
			case "ないID":
				id = 999
			case "未実行":
				b.history[0].completed = false
			case "復元失敗":
				b.history[0].before.rngState = []byte{1}
			case "残り0優先":
				b.rewindsLeft = 0
				id = 999
				want = ErrNoRewinds
			}
			assertRewindFailure(t, b, want, func() error { return b.Rewind(id) })
		}
		b := snapshotBattle(t)
		assertRewindFailure(t, b, ErrNoPendingResume, func() error { _, err := b.Resume(); return err })
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		restoreHistoryForResume(t, b, b.History()[1].ID)
		assertRewindFailure(t, b, ErrNotPlayerPhase, func() error { _, err := b.EndPhase(); return err })
		assertRewindFailure(t, b, ErrNotPlayerPhase, func() error { _, err := b.Act(Command{Action: Wait}); return err })
		pending := b.snapshot()
		pending.progress = phaseProgress{stage: enemyStart}
		pending.Phase = Player
		if err := b.restore(pending); err != nil {
			t.Fatal(err)
		}
		assertRewindFailure(t, b, ErrResumeRequired, func() error { _, err := b.EndPhase(); return err })
		assertRewindFailure(t, b, ErrResumeRequired, func() error { _, err := b.Act(Command{Action: Wait}); return err })
	})
	t.Run("TC-FR-BTL-10-09 未来を捨て別の行動に未使用 ID を付ける", func(t *testing.T) {
		b := snapshotBattle(t)
		rewindAct(t, b, Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		if _, err := b.EndPhase(); err != nil {
			t.Fatal(err)
		}
		old := b.History()
		next := b.nextHistoryID
		old[0].UnitName = "改変"
		if b.History()[0].UnitName != "主人公" {
			t.Fatal("一覧の改変が伝わった")
		}
		rewindTo(t, b, old[0].ID)
		for _, h := range old {
			assertRewindFailure(t, b, ErrInvalidRewind, func() error { return b.Rewind(h.ID) })
		}
		before := b.snapshot()
		left := b.RewindsLeft()
		b.History()
		b.Forecast(Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2})
		if _, err := b.Act(Command{Unit: 99, Action: Wait}); err == nil {
			t.Fatal("無効な Act が成功")
		}
		if !reflect.DeepEqual(before, b.snapshot()) || len(b.History()) != 0 || b.nextHistoryID != next || b.RewindsLeft() != left {
			t.Fatal("閲覧・予測・無効な行動で変化")
		}
		rewindAct(t, b, Command{Unit: 0, To: Pos{1, 0}, Action: Wait})
		if b.History()[0].ID != next || b.History()[0].To != (Pos{1, 0}) {
			t.Fatal("分岐後の行動または ID が不正")
		}
	})
	t.Run("TC-FR-BTL-10-10 勝敗後から復帰し終了時に再開待ちを残さない", func(t *testing.T) {
		for _, loss := range []bool{false, true} {
			b := snapshotBattle(t)
			b.units[0].Pos = Pos{1, 0}
			b.units[0].Weapons[0].Hit = 1000
			b.units[2].Weapons[0].Hit = 1000
			b.units[2].Skill = Skill{}
			if loss {
				b.units[0].HP = 1
			} else {
				b.units[2].HP = 1
			}
			// 自軍行動を用意し、敵軍中の主人公撃破、または反撃でボス撃破。
			cmd := Command{Unit: 1, To: Pos{0, 1}, Action: Wait}
			rewindAct(t, b, cmd)
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
			want := Won
			if loss {
				want = Lost
			}
			if b.Result != want || b.NeedsResume() {
				t.Fatal("勝敗または終了後の再開待ちが不正")
			}
			assertRewindFailure(t, b, ErrNoPendingResume, func() error { _, err := b.Resume(); return err })
			h := b.History()[0]
			rewindTo(t, b, h.ID)
			if b.Result != Ongoing || b.NeedsResume() || !b.units[0].Alive || !b.units[2].Alive {
				t.Fatal("勝敗決定前へ復帰できない")
			}
			rewindAct(t, b, cmd)
			if _, err := b.EndPhase(); err != nil {
				t.Fatal(err)
			}
			if b.Result != want || b.NeedsResume() {
				t.Fatal("再開後の勝敗が不正")
			}
		}
	})
}

// restoreHistoryForResume は公開操作の拒否とプレビューを検査し、内部の継続位置だけを復元する（DES-IT04 7.1 節）。
func restoreHistoryForResume(t *testing.T, b *Battle, id uint64) {
	t.Helper()
	for i, h := range b.history {
		if h.entry.ID != id {
			continue
		}
		if h.entry.Side == Player && (h.entry.Kind == HistoryAction || h.entry.Kind == HistoryEndPhase && !h.entry.Automatic) {
			rewindTo(t, b, id)
			return
		}
		v, err := b.PreviewRewind(id)
		if err != nil {
			t.Fatal(err)
		}
		assertBoard(t, v, h.before)
		assertRewindFailure(t, b, ErrInvalidRewind, func() error { return b.Rewind(id) })
		if err := b.restore(h.before); err != nil {
			t.Fatal(err)
		}
		b.history = b.history[:i]
		return
	}
	t.Fatal("内部復元のIDなし")
}

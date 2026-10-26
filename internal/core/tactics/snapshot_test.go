package tactics

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

// snapshotBattle は実 PCG と、履歴を確認する最小の戦闘を作る（DES-RWD-01・03）。
func snapshotBattle(t *testing.T) *Battle {
	t.Helper()
	weapon := Weapon{Name: "剣", Type: Sword, MinRange: 1, MaxRange: 1, Hit: 70, Might: 2}
	player := UnitSetup{Name: "主人公", Side: Player, Lord: true, Stats: Stats{HP: 100, Str: 5, Skl: 5, Spd: 10, Def: 5, Mov: 2}, Pos: Pos{0, 0}, Weapon: weapon, Level: 1, Exp: 99, Growth: Growth{50, 50, 50, 50, 50, 0, 0, 0}, Items: []ItemStack{{Item: Item{Name: "薬", Heal: 10}, Count: 2}}}
	ally := player
	ally.Name = "味方"
	ally.Lord = false
	ally.Pos = Pos{0, 1}
	enemy := player
	enemy.Name = "ボス"
	enemy.Side = Enemy
	enemy.Lord = false
	enemy.Boss = true
	enemy.Level = 0
	enemy.Exp = 0
	enemy.Pos = Pos{2, 0}
	enemy.Stats.Spd = 1
	enemy.Stats.Mov = 0
	enemy.AI = Hold
	enemy.Skill = Skill{Effect: Pavise, Chance: 50}
	terrain := [][]Terrain{{{Cost: 1}, {Cost: 1}, {Cost: 1}}, {{Cost: 1}, {Cost: 1}, {Cost: 1}}}
	b, _, err := New(Setup{Terrain: terrain, Units: []UnitSetup{player, ally, enemy}, Reinforcements: []ReinforcementSetup{{Turn: 2, Unit: UnitSetup{Name: "増援", Side: Enemy, Stats: Stats{HP: 10}, Weapon: weapon, Pos: Pos{2, 1}, Items: player.Items}}}}, 42)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSnapshotState(t *testing.T) {
	t.Run("TC-FR-BTL-10-15 全戦闘値の深い写しと復元", func(t *testing.T) {
		b := snapshotBattle(t)
		b.units[1].Alive = false
		b.units[1].HP = 0
		b.units[0].Weapon = Weapon{Type: Staff, Heal: 10, Uses: 2, MaxUses: 5, MinRange: 1, MaxRange: 1}
		b.progress = phaseProgress{stage: enemyAction, nextEnemy: 2, enemyCount: 3}
		before := b.snapshot()
		want := b.Units()
		b.units[0].Items[0].Count = 1
		if before.units[0].Items[0].Count != 2 {
			t.Fatal("保存後の手持ちが写しを変更した")
		}
		b.units[0] = Unit{}
		b.units = append(b.units, Unit{Name: "未来の増援"})
		b.Turn = 5
		b.Phase = Enemy
		b.Result = Lost
		b.progress = phaseProgress{}
		if err := b.restore(before); err != nil {
			t.Fatal(err)
		}
		if b.Turn != 1 || b.Phase != Player || b.Result != Ongoing || !reflect.DeepEqual(b.units, want) || b.progress != before.progress {
			t.Fatal("戦闘値・添字・続きが復元されない")
		}
		b.units[0].Items[0].Count = 0
		if before.units[0].Items[0].Count != 2 {
			t.Fatal("復元後の手持ちが写しを変更した")
		}
		before.units[0].Items[0].Count = 9
		if b.units[0].Items[0].Count != 0 {
			t.Fatal("写しが復元後の手持ちを変更した")
		}
		// 地形と増援定義は入力から独立しており、写し同士では読み取り専用で共有できる。
		source := snapshotBattle(t)
		setup := Setup{Terrain: source.terrain, Reinforcements: source.reinforcements}
		for _, u := range source.units {
			setup.Units = append(setup.Units, UnitSetup{Name: u.Name, Side: u.Side, Lord: u.Lord, Boss: u.Boss, Stats: u.Stats, Weapon: u.Weapon, Pos: u.Pos, Items: u.Items})
		}
		other, _, err := New(setup, 1)
		if err != nil {
			t.Fatal(err)
		}
		saved := other.snapshot()
		setup.Terrain[0][0].Cost = 0
		setup.Reinforcements[0].Unit.Items[0].Count = 99
		setup.Units[0].Items[0].Count = 99
		returned := other.Units()
		returned[0].Items[0].Count = 99
		if saved.terrain[0][0].Cost != 1 || saved.reinforcements[0].Unit.Items[0].Count != 2 || saved.units[0].Items[0].Count != 2 {
			t.Fatal("入力または返却値が写しを変更した")
		}

	})
	t.Run("TC-FR-BTL-10-16 実 PCG の復元と攻撃・大盾・成長の再現", func(t *testing.T) {
		b := snapshotBattle(t)
		cmd := Command{Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 2}
		// 初期 seed に戻すだけの実装では通らないよう、途中の内部状態を保存する。
		for i := 0; i < 7; i++ {
			b.roll()
		}
		before := b.snapshot()
		events, err := b.Act(cmd)
		if err != nil {
			t.Fatal(err)
		}
		after := b.snapshot()
		strike, level := false, false
		for _, e := range events {
			strike = strike || e.Kind == EventStrike
			level = level || e.Kind == EventLevelUp
		}
		if !strike || !level {
			t.Fatal("攻撃と成長を通っていない")
		}
		for i := 0; i < 3; i++ {
			if err := b.restore(before); err != nil {
				t.Fatal(err)
			}
			got, err := b.Act(cmd)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, events) || !reflect.DeepEqual(b.snapshot(), after) {
				t.Fatal("同じ行動の結果が一致しない")
			}
		}
		// 旧実装と同じ PCG の列を、保存による乱数消費も含めて確かめる。
		b = snapshotBattle(t)
		ref := rand.New(rand.NewPCG(42, 0))
		for i := 0; i < 1000; i++ {
			_ = b.snapshot()
			if b.roll() != ref.IntN(100) {
				t.Fatalf("旧 PCG 列と不一致: %d", i)
			}
		}
	})
	t.Run("TC-FR-BTL-10-19 壊れた PCG の復元は状態不変", func(t *testing.T) {
		for _, data := range [][]byte{nil, {1, 2, 3}, make([]byte, 20)} {
			b := snapshotBattle(t)
			before := b.snapshot()
			bad := b.snapshot()
			bad.rngState = data
			bad.units[0].HP = 1
			if err := b.restore(bad); err == nil {
				t.Fatal("壊れた PCG を受理した")
			}
			if !reflect.DeepEqual(b.snapshot(), before) {
				t.Fatal("復元失敗で状態が変わった")
			}
		}
	})
}

func TestHistoryFoundation(t *testing.T) {
	t.Run("TC-FR-BTL-10-17 行動と番の履歴・出来事の境界", func(t *testing.T) {
		b := snapshotBattle(t)
		if len(b.history) != 0 {
			t.Fatal("最初の番の開始を記録した")
		}
		initialEvents := b.eventCount
		ev, err := b.Act(Command{Unit: 0, To: Pos{1, 0}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if len(b.history) != 1 {
			t.Fatal("自軍の行動単位でない")
		}
		first := b.history[0]
		if first.entry.Kind != historyAction || first.entry.Action != Wait || first.entry.Unit != 0 || first.entry.UnitName != "主人公" || first.entry.To != (Pos{1, 0}) || first.entry.EventStart != initialEvents || first.entry.EventEnd != initialEvents+len(ev) || first.before.units[0].Pos != (Pos{0, 0}) {
			t.Fatalf("自軍の履歴: %+v", first.entry)
		}
		ev, err = b.EndPhase()
		if err != nil {
			t.Fatal(err)
		}
		if len(b.history) != 4 {
			t.Fatalf("履歴数 %d", len(b.history))
		}
		end, enemy, start := b.history[1], b.history[2], b.history[3]
		if end.entry.Kind != historyEndPhase || end.entry.Automatic || enemy.entry.Side != Enemy || enemy.entry.Unit != 2 || enemy.before.units[2].AI != Hold || start.entry.Kind != historyPlayerStart || start.entry.Side != Player || start.entry.Turn != 2 || start.before.Turn != 1 || b.Turn != 2 {
			t.Fatal("番境界または敵の行動前が不正")
		}
		if enemy.entry.Action != Attack || enemy.entry.Target != 0 || enemy.entry.TargetName != "主人公" || enemy.entry.UnitName != "ボス" || enemy.before.progress.stage != enemyAction || enemy.before.progress.nextEnemy != 2 || enemy.before.progress.enemyCount != 3 || end.before.progress.stage != playerInput || end.entry.Automatic || start.before.progress.stage != playerStart {
			t.Fatal("敵の要約と再開位置が不正")
		}
		if end.entry.EventStart != first.entry.EventEnd || end.entry.EventEnd != enemy.entry.EventStart || enemy.entry.EventEnd != start.entry.EventStart || start.entry.EventEnd != b.eventCount || b.eventCount != first.entry.EventEnd+len(ev) {
			t.Fatal("出来事の境界が不連続")
		}
		for i, h := range b.history {
			if h.entry.ID != uint64(i+1) {
				t.Fatal("履歴 ID が単調増加しない")
			}
		}
		// 動かない敵も、出来事を増やさず履歴に残す。
		b = snapshotBattle(t)
		b.units[0].Pos = Pos{0, 0}
		b.units[2].Pos = Pos{2, 1}
		_, err = b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		_, err = b.Act(Command{Unit: 1, To: Pos{0, 1}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if !b.history[2].entry.Automatic || b.history[2].before.progress.stage != enemyStart {
			t.Fatal("自動移行を区別していない")
		}
		h := b.history[3]
		if h.entry.Action != Wait || h.entry.EventStart != h.entry.EventEnd {
			t.Fatal("無行動の敵に出来事を追加した")
		}
	})
	t.Run("TC-FR-BTL-10-18 閲覧・予測・失敗は履歴と乱数を変えない", func(t *testing.T) {
		b := snapshotBattle(t)
		before := b.snapshot()
		id, count := b.nextHistoryID, b.eventCount
		b.Units()
		b.Reachable(0)
		b.Forecast(Command{Unit: 0, Target: 2, To: Pos{1, 0}, Action: Attack})
		for _, cmd := range []Command{{Unit: 99, Action: Wait}, {Unit: 0, To: Pos{1, 0}, Action: Attack, Target: 99}, {Unit: 0, To: Pos{1, 0}, Action: UseItem, Item: 99}, {Unit: 0, To: Pos{1, 0}, Action: Heal, Target: 1}} {
			if _, err := b.Act(cmd); err == nil {
				t.Fatal("無効な行動が成功した")
			}
		}
		if !reflect.DeepEqual(b.snapshot(), before) || len(b.history) != 0 || b.nextHistoryID != id || b.eventCount != count {
			t.Fatal("閲覧・予測・失敗で内部値が変わった")
		}
		b.Phase = Enemy
		before = b.snapshot()
		if _, err := b.EndPhase(); err == nil {
			t.Fatal("敵軍の番終了が成功した")
		}
		if !reflect.DeepEqual(b.snapshot(), before) || len(b.history) != 0 {
			t.Fatal("失敗した番終了で変化した")
		}
	})
	t.Run("TC-FR-BTL-10-20 写しの乱数と一覧の独立・ID の非復元", func(t *testing.T) {
		b := snapshotBattle(t)
		s := b.snapshot()
		saved := append([]byte(nil), s.rngState...)
		b.roll()
		if !reflect.DeepEqual(s.rngState, saved) {
			t.Fatal("乱数の保存値を共有した")
		}
		_, err := b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		id := b.nextHistoryID
		count := len(b.history)
		entries := b.historyEntries()
		entries[0].UnitName = "改変"
		if b.history[0].entry.UnitName != "主人公" {
			t.Fatal("一覧の改変が伝わった")
		}
		if err := b.restore(s); err != nil {
			t.Fatal(err)
		}
		if b.nextHistoryID != id || len(b.history) != count {
			t.Fatal("写しから履歴または ID を復元した")
		}
		_, err = b.Act(Command{Unit: 0, To: Pos{0, 0}, Action: Wait})
		if err != nil {
			t.Fatal(err)
		}
		if b.history[len(b.history)-1].entry.ID != id {
			t.Fatal("履歴 ID を再利用した")
		}
	})
}

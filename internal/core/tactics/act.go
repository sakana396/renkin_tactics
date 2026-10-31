package tactics

// Act は自軍のユニット 1 体を移動させ、行動させる（DES-BTL-04）。
//
// 共通の検証 0〜5 と、行動ごとの検証 6 を順に確かめ、最初に当てはまった誤りを返す。
// 失敗したら状態を変えない（DES-ERR-01）。
// Attack は移動してから Target を攻撃する（DES-BTL-06）。UseItem は移動してから Item 番目のアイテムを使う（DES-ITM-01）。
// Heal は移動してから、選んだ杖で Target の味方を回復する（DES-HEL-01）。
// Attack と Heal は Weapon で武器の列の添字を選べる（nil なら今の装備）。装備を変えるのは成功した Attack だけ（DES-ITM-02）。
// 行動のあと、戦闘が続いていて、生きている自軍が全員行動済みなら、敵軍の番と次の自軍の番の始めまでを進める（DES-BTL-02）。
// 返す出来事は、その間に起きたことすべて（DES-EVT-01）。
//
// 再移動のスキルを持つユニットが攻撃・回復・道具のあとに残りの移動力を持てば、再移動を待つ（PendingCanto）。
// 待っている間は敵軍の番へ進まず、CantoMove・CantoEnd と巻き戻しだけを受け付ける（DES-BTL-10）。
func (b *Battle) Act(cmd Command) ([]Event, error) {
	if cmd.Action == CantoMove || cmd.Action == CantoEnd {
		return b.actCanto(cmd)
	}
	if err := b.validateCommand(cmd); err != nil {
		return nil, err
	}
	weapon := -1
	var err error
	switch cmd.Action {
	case Attack:
		if weapon, err = b.validateAttack(cmd); err != nil {
			return nil, err
		}
	case UseItem:
		if err := b.validateUseItem(cmd); err != nil {
			return nil, err
		}
	case Heal:
		if weapon, err = b.validateHeal(cmd); err != nil {
			return nil, err
		}
	}

	// 検証成功後、移動の前に記録する（DES-RWD-01）。
	target := -1
	if cmd.Action == Attack || cmd.Action == Heal {
		target = cmd.Target
	}
	record := b.beginHistory(historyEntry{Kind: historyAction, Action: cmd.Action, Unit: cmd.Unit, Target: target, To: cmd.To})
	// 残りの移動力は、移動力から行動の前の位置から To までの最小の費用を引いたもの（DES-IT06 8.2 節）。
	remaining := b.units[cmd.Unit].Stats.Mov - b.Reachable(cmd.Unit)[cmd.To]
	// ここから状態を変える。まず移動する。
	var events []Event
	u := &b.units[cmd.Unit]
	from := u.Pos
	if cmd.To != from {
		u.Pos = cmd.To
		events = b.emit(events, Event{Kind: EventMove, Unit: cmd.Unit, From: from, To: cmd.To})
	}
	u.Acted = true
	switch cmd.Action {
	case Wait:
		events = b.emit(events, Event{Kind: EventWait, Unit: cmd.Unit})
	case Attack:
		// 攻撃を決めたときだけ、選んだ武器を装備する。外れても「攻撃に使った」に入り、反撃・追撃もこの武器で行う（DES-ITM-02）。
		u.Equipped = weapon
		events = b.attack(events, cmd.Unit, cmd.Target, cmd.To == from)
	case UseItem:
		events = b.useItem(events, cmd.Unit, cmd.Item)
	case Heal:
		events = b.healWithStaff(events, cmd.Unit, cmd.Target, weapon)
	}
	// 移動・攻撃・戦死・経験・成長・勝敗を済ませたあとで、出来事を評価する（DES-MAP-07）。
	events = b.evaluateMapEvents(events, events)

	// 再移動を待つときも、最初の Act の成功で履歴を公開し、同じ「前の状態」へ戻せるようにする（DES-BTL-10）。
	b.finishHistory(record)
	if b.Result != Ongoing {
		return events, nil
	}
	if b.canCanto(cmd.Unit, cmd.Action, remaining) {
		b.canto = cantoState{active: true, PendingCanto: PendingCanto{Unit: cmd.Unit, Remaining: remaining, HistoryID: b.history[record].entry.ID}}
		return events, nil
	}
	if b.allPlayersActed() {
		events = b.runEnemyPhase(events, true)
	}
	return events, nil
}

// validateCommand は Act の共通の検証 0〜5 を順に確かめる（DES-BTL-04）。状態は変えない。
func (b *Battle) validateCommand(cmd Command) error {
	// 0. Action が 4 つ（Wait・Attack・UseItem・Heal）のどれかである（DES-HEL-01）。
	if cmd.Action < Wait || cmd.Action > Heal {
		return ErrInvalidAction
	}
	// 武器を選べるのは攻撃と回復だけ（DES-ITM-02）。
	if cmd.Weapon != nil && cmd.Action != Attack && cmd.Action != Heal {
		return ErrInvalidAction
	}
	// 1. 戦闘が終わっていない。
	if b.Result != Ongoing {
		return ErrBattleOver
	}
	// 2. 自軍の番である。
	if b.Phase != Player {
		return ErrNotPlayerPhase
	}
	if b.NeedsResume() {
		return ErrResumeRequired
	}
	// 再移動を待っている間は、ほかの行動を受け付けない（DES-BTL-10）。
	if b.canto.active {
		return ErrCantoPending
	}
	// 3. Unit が存在し、生きている自軍のユニットである。
	if !b.validUnit(cmd.Unit) || !b.units[cmd.Unit].Alive || b.units[cmd.Unit].Side != Player {
		return ErrInvalidUnit
	}
	// 4. Unit がまだ行動していない。
	if b.units[cmd.Unit].Acted {
		return ErrAlreadyActed
	}
	// 5. To が移動できるマスである。
	if _, ok := b.Reachable(cmd.Unit)[cmd.To]; !ok {
		return ErrUnreachable
	}
	return nil
}

// EndPhase は自軍の番を途中で終え、敵軍の番と次の自軍の番の始めまでを進める（DES-BTL-02、FR-BTL-02）。
// 失敗したら状態を変えない（DES-ERR-01）。
func (b *Battle) EndPhase() ([]Event, error) {
	if b.Result != Ongoing {
		return nil, ErrBattleOver
	}
	if b.Phase != Player {
		return nil, ErrNotPlayerPhase
	}
	if b.NeedsResume() {
		return nil, ErrResumeRequired
	}
	if b.canto.active {
		return nil, ErrCantoPending
	}
	return b.runEnemyPhase(nil, false), nil
}

// allPlayersActed は、生きている自軍が全員行動済みかを返す。
func (b *Battle) allPlayersActed() bool {
	for _, u := range b.units {
		if u.Alive && u.Side == Player && !u.Acted {
			return false
		}
	}
	return true
}

// startPlayerPhase は自軍の番を始める（DES-BTL-02）。
// 自軍の全員の Acted を偽にし、PhaseStart を出し、砦にいる自軍を回復する。
func (b *Battle) startPlayerPhase(events []Event) []Event {
	b.Phase = Player
	for i := range b.units {
		if b.units[i].Side == Player {
			b.units[i].Acted = false
		}
	}
	events = b.emit(events, Event{Kind: EventPhaseStart, Side: Player, Turn: b.Turn})
	return b.healOnForts(events, Player)
}

// runEnemyPhase は敵軍の番と次の自軍の番の始めまで進め、処理ごとの行動前を記録する（DES-BTL-02、DES-RWD-01）。
func (b *Battle) runEnemyPhase(events []Event, automatic bool) []Event {
	// 明示の番終了だけは、自動進行へ移る前の入力待ちを保存する（DES-IT04 7.1 節）。
	var before *battleSnapshot
	if !automatic {
		s := b.snapshot()
		before = &s
	}
	b.progress = phaseProgress{stage: enemyStart, automatic: automatic}
	return b.continuePhaseBefore(events, before)
}

// continuePhase は通常進行と巻き戻し後の再開で同じ番処理を使う（DES-RWD-02、DES-BTL-02）。
func (b *Battle) continuePhase(events []Event) []Event {
	return b.continuePhaseBefore(events, nil)
}

// continuePhaseBefore は明示の番終了の写しを渡し、敵軍開始の履歴を一度だけ作る（DES-RWD-01・02）。
func (b *Battle) continuePhaseBefore(events []Event, before *battleSnapshot) []Event {
	if b.progress.stage == enemyStart {
		automatic := b.progress.automatic
		var snapshot battleSnapshot
		if before != nil {
			snapshot = *before
		} else {
			snapshot = b.snapshot()
		}
		record := b.beginHistoryBefore(historyEntry{Kind: historyEndPhase, Unit: -1, Target: -1, Automatic: automatic}, snapshot)
		// 敵軍の番の始め: PhaseStart → ターンの増援 → 予約した出来事の増援 → 砦の回復（DES-IT06 9.3 節）。
		b.Phase = Enemy
		events = b.emit(events, Event{Kind: EventPhaseStart, Side: Enemy, Turn: b.Turn})
		// この番に現れた増援は動かないので、増援を置く前の数だけ回す。
		n := len(b.units)
		events = b.placeReinforcements(events)
		events = b.placeEventReinforcements(events)
		events = b.healOnForts(events, Enemy)
		b.planEnemyGroups(n)

		b.progress = phaseProgress{stage: enemyAction, enemyCount: n}
		b.finishHistory(record)
	}
	if b.progress.stage == enemyAction {
		// 敵軍の番: 生きている敵を添字の順に 1 体ずつ動かす。
		for i := b.progress.nextEnemy; i < b.progress.enemyCount; i++ {
			if b.Result != Ongoing {
				b.progress = phaseProgress{}
				return events
			}
			if u := b.units[i]; u.Alive && u.Side == Enemy {
				b.progress.nextEnemy = i
				record := b.beginHistory(historyEntry{Kind: historyAction, Unit: i, Target: -1, To: u.Pos})
				eventStart := len(events)
				if b.actEnemyOverride != nil {
					events = b.actEnemyOverride(events, i)
				} else {
					events = b.actEnemy(events, i)
				}
				// 敵 1 体の行動を済ませたあとで、出来事を評価する（DES-MAP-07）。
				events = b.evaluateMapEvents(events, events[eventStart:])
				// 敵の選択は従来どおり一度だけ実行し、出来事から一覧の要約を得る。
				for _, e := range events[eventStart:] {
					if e.Kind == EventMove {
						b.history[record].entry.To = e.To
					}
					if e.Kind == EventStrike && e.Unit == i {
						b.history[record].entry.Action = Attack
						b.history[record].entry.Target = e.Target
						b.history[record].entry.TargetName = b.units[e.Target].Name
						break
					}
				}
				b.progress.nextEnemy = i + 1
				b.finishHistory(record)
			}
		}
		if b.Result != Ongoing {
			b.progress = phaseProgress{}
			return events
		}
		b.progress = phaseProgress{stage: playerStart}
	}

	// 敵軍の番の終わり: ターンを進め、自軍の番を始める。
	record := b.beginHistory(historyEntry{Kind: historyPlayerStart, Unit: -1, Target: -1})
	b.Turn++
	events = b.startPlayerPhase(events)
	b.progress = phaseProgress{}
	b.finishHistory(record)
	return events
}

// placeReinforcements は、このターンの増援を、増援の定義の順に置く（DES-MAP-03）。
// 元のマスから距離 2 以内の立てる空きマスを距離・Y・X の順で選ぶ。候補がなければ出さず、延期しない。
// 出すと生きている敵が上限を超える個体も出さず、延期しない（DES-IT06 9.3 節）。
// 現れた増援はユニットの列の終わりに足し、Reinforce を出す。
func (b *Battle) placeReinforcements(events []Event) []Event {
	for _, r := range b.reinforcements {
		if r.Turn != b.Turn {
			continue
		}
		events = b.spawnEnemy(events, r.Unit)
	}
	return events
}

// healOnForts は、side の陣営で砦にいる生きているユニットを、ユニットの添字の順に回復する（DES-MAP-02）。
// 回復する量は 最大 HP × HealPercent ÷ 100（切り捨て）で、最大 HP を超えない。0 なら Heal を出さない。
func (b *Battle) healOnForts(events []Event, side Side) []Event {
	for i := range b.units {
		u := &b.units[i]
		pct := b.TerrainAt(u.Pos).HealPercent
		if !u.Alive || u.Side != side || pct < 1 {
			continue
		}
		events = b.heal(events, i, u.Stats.HP*pct/100, "")
	}
	return events
}

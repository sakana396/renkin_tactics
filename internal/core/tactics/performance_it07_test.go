package tactics_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// TestIT07EnemyPhaseTiming は32×32盤面・自軍7体・敵軍39体の敵フェイズ時間を測る（TC-NFR-02-03）。
// 基準PCでの受入判定は測定結果を記録するときに行う。
func TestIT07EnemyPhaseTiming(t *testing.T) {
	if os.Getenv("RENKIN_IT07_PERF") == "" {
		t.Skip("RENKIN_IT07_PERF を設定したときだけ予備測定する")
	}
	t.Run("TC-NFR-02-03 敵フェイズ時間", func(t *testing.T) {
		const count = 30
		durations := make([]time.Duration, 0, count)
		minDuration, maxDuration := time.Duration(1<<63-1), time.Duration(0)
		minSeed, maxSeed := 0, 0
		var total time.Duration
		for seed := 1; seed <= count; seed++ {
			battle, _, err := tactics.New(performanceIT07Setup(), uint64(seed))
			if err != nil {
				t.Fatalf("seed %d: 戦闘を作れない: %v", seed, err)
			}
			units := battle.Units()
			players, enemies, playerZOC, enemyZOC, commanderHold := 0, 0, false, false, false
			for _, unit := range units {
				if unit.Side == tactics.Player {
					players++
					playerZOC = playerZOC || performanceHasZOC(unit.ClassSkills)
				}
				if unit.Side == tactics.Enemy {
					enemies++
					enemyZOC = enemyZOC || performanceHasZOC(unit.ClassSkills)
					if unit.Commander {
						commanderHold = unit.AI == tactics.Hold
					}
				}
			}
			if players != 7 || enemies != 39 {
				t.Fatalf("seed %d: 初期体数 自軍=%d 敵軍=%d、期待7/39", seed, players, enemies)
			}
			if !playerZOC || !enemyZOC || !commanderHold {
				t.Fatalf("seed %d: ZOC（自軍=%v、敵軍=%v）・AI=Holdの小隊長=%v", seed, playerZOC, enemyZOC, commanderHold)
			}

			started := time.Now()
			_, err = battle.EndPhase()
			elapsed := time.Since(started)
			if err != nil {
				t.Fatalf("seed %d: 敵フェイズを実行できない: %v", seed, err)
			}
			if battle.Turn != 2 || battle.Phase != tactics.Player || battle.Result != tactics.Ongoing {
				t.Fatalf("seed %d: 敵軍の番が途中で打ち切られた: Turn=%d Phase=%v Result=%v", seed, battle.Turn, battle.Phase, battle.Result)
			}
			durations = append(durations, elapsed)
			total += elapsed
			if elapsed < minDuration {
				minDuration, minSeed = elapsed, seed
			}
			if elapsed > maxDuration {
				maxDuration, maxSeed = elapsed, seed
			}
		}
		mean := total / time.Duration(count)
		threshold := 1 * time.Second
		t.Logf("GoOS=%s GoArch=%s GoVersion=%s NumCPU=%d GOMAXPROCS=%d RENKIN_IT07_PERF=%q count=%d min=%s(seed=%d) mean=%s max=%s(seed=%d) 参考閾値=%s以内", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU(), runtime.GOMAXPROCS(0), os.Getenv("RENKIN_IT07_PERF"), len(durations), minDuration, minSeed, mean, maxDuration, maxSeed, threshold)
	})
}

// performanceIT07Setup は BenchmarkEnemyPhase46（ai_it06_test.go）の探索負荷を保ち、
// ZOCを持つ自軍・敵軍とHoldの小隊長を加える（DES-IT07 4・5章）。
func performanceIT07Setup() tactics.Setup {
	rows := make([]string, 32)
	for i := range rows {
		rows[i] = strings32Dots
	}
	setup := tactics.Setup{Terrain: grid(rows...)}
	for i := 0; i < 7; i++ {
		u := ally("自軍", 2, i+2, 7)
		// 主人公の戦死で敵軍の番が途中で終わらないよう、測定用に自軍を固くする。
		// 反撃で敵も減らさず、39 体の思考を最後まで測る。
		u.Stats.HP, u.Stats.Def, u.Stats.Res = 1000, 99, 99
		u.Stats.Str, u.Stats.Mag = 0, 0
		if i == 0 {
			u.Lord = true
			u.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		}
		setup.Units = append(setup.Units, u)
	}
	for i := 0; i < 39; i++ {
		u := enemy("敵", 6+i%6, 2+i/6, 3)
		if i == 38 {
			u.Boss = true
			u.Group, u.Approach = "", tactics.Direct
		} else {
			u.Group, u.Approach = "advance", tactics.GroupApproach
		}
		if i == 0 {
			u.Commander, u.AI = true, tactics.Hold
			u.ClassSkills = []tactics.Skill{{Name: "通せんぼ", Effect: tactics.ZOC}}
		}
		u.ClassSkills = append(u.ClassSkills, tactics.Skill{Name: "再移動", Effect: tactics.Canto})
		if i == 38 {
			u.AI = tactics.Hold
		}
		u.Weapons = []tactics.Weapon{u.Weapon, {Name: "長槍", Type: tactics.Lance, Might: 4, Hit: 85, MinRange: 1, MaxRange: 2}, {Name: "弓", Type: tactics.Bow, Might: 4, Hit: 80, MinRange: 2, MaxRange: 2}, {Name: "強い剣", Type: tactics.Sword, Might: 8, Hit: 70, MinRange: 1, MaxRange: 1}, {Name: "遠い弓", Type: tactics.Bow, Might: 3, Hit: 70, MinRange: 2, MaxRange: 3}}
		u.Weapon = tactics.Weapon{}
		u.Items = nil
		u.AllowedWeaponTypes = []tactics.WeaponType{tactics.Sword, tactics.Lance, tactics.Bow}
		setup.Units = append(setup.Units, u)
	}
	return setup
}

const strings32Dots = "................................"

func performanceHasZOC(skills []tactics.Skill) bool {
	for _, skill := range skills {
		if skill.Effect == tactics.ZOC {
			return true
		}
	}
	return false
}

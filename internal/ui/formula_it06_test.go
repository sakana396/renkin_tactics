package ui

import (
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"testing"
)

func TestIT06Forecast(t *testing.T) {
	t.Run("TC-FR-BTL-08-25", func(t *testing.T) {
		for _, tc := range []struct {
			name                                       string
			count, strikes, hpD, damageD, wantA, wantD int
		}{{"2回が反撃より先", 2, 2, 20, 99, 30, 0}, {"4回", 4, 2, 100, 1, 29, 60}, {"反撃後に打切り", 4, 2, 100, 99, 0, 80}} {
			f := tactics.Forecast{Attacker: tactics.ForecastSide{HP: 30, Damage: 10, Count: tc.count, Strikes: tc.strikes, CritRate: 50, AttackSpeed: 12, Effective: true, WeightLoss: 1, EffectiveMight: 12}, Defender: tactics.ForecastSide{HP: tc.hpD, Damage: tc.damageD, Count: 1}}
			a, d := ForecastAfter(f)
			if a != tc.wantA || d != tc.wantD {
				t.Fatalf("%s: HP=(%d,%d)", tc.name, a, d)
			}
			rows := ForecastRows(f, false, false)
			// 重さで落ちた速さと、特効の効く威力を出す（DES-IT06 12.2 節、UC-03、REV-069 F-03）。
			if rows[5].Attacker != "50%" || rows[6] != (ForecastRow{"速さ", "12（重さ −1）", "0"}) || rows[7] != (ForecastRow{"特効", "威力 12", "―"}) {
				t.Fatalf("予測=%+v", rows)
			}
		}
	})
}

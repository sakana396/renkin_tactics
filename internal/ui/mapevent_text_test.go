package ui

import (
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 出来事のきっかけと一団の突撃の文（DES-MAP-07、IF-02）。
func TestMapEventTexts(t *testing.T) {
	t.Run("TC-FR-MAP-03-17 / TC-FR-BTL-07-19 出来事が起きた・一団が動き出した", func(t *testing.T) {
		cases := []struct {
			e    tactics.Event
			text string
		}{
			{e: tactics.Event{Kind: tactics.EventTrigger, MapEvent: "gate_alarm"}, text: "出来事が起きた"},
			{e: tactics.Event{Kind: tactics.EventGroupCharge, MapEvent: "advance", Group: "gate_group"}, text: "敵の一団が動き出した"},
		}
		for _, tc := range cases {
			if got := EventText(tc.e); got != tc.text {
				t.Errorf("EventText(%v) = %q、%q のはず", tc.e.Kind, got, tc.text)
			}
		}
	})
}

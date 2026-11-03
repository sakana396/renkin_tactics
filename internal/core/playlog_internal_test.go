package core

import (
	"encoding/json"
	"testing"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// 本体の定義データにはまだきっかけと一団の突撃がないので、出来事の形だけを確かめる（DES-LOG-01）。
func TestPlayLogEventKinds(t *testing.T) {
	cases := []struct {
		name string
		e    tactics.Event
		want string
	}{
		{"trigger は event だけ", tactics.Event{Kind: tactics.EventTrigger, MapEvent: "ev1", Group: "g"}, `{"kind":"trigger","event":"ev1"}`},
		{"group_charge は event と group", tactics.Event{Kind: tactics.EventGroupCharge, MapEvent: "ev2", Group: "g1"}, `{"kind":"group_charge","event":"ev2","group":"g1"}`},
	}
	for _, tc := range cases {
		t.Run("TC-NFR-06-09 "+tc.name, func(t *testing.T) {
			got, err := json.Marshal(toLogEvent(tc.e))
			if err != nil || string(got) != tc.want {
				t.Fatalf("%s, %v; 期待 %s", got, err, tc.want)
			}
		})
	}
}

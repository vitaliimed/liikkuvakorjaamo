package crm

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestPrivateResponses(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Name = "SECRET_CLIENT"
	j.Report.ReceivedCents = 987654
	if _, ok := jobResponse(j, testMechanic)["draft"]; ok {
		t.Fatal("draft leaked")
	}
	if _, ok := jobResponse(j, testOwner)["draft"]; !ok {
		t.Fatal("owner draft missing")
	}
	v, ok := eventView(Event{Type: "report", Text: "SECRET_TEXT", Snapshot: &j}, testMechanic)
	if !ok {
		t.Fatal("event missing")
	}
	b, _ := json.Marshal(v)
	if strings.Contains(string(b), "SECRET") || strings.Contains(string(b), "snapshot") {
		t.Fatal(string(b))
	}
	b, _ = json.Marshal(technicalVisit(j))
	if strings.Contains(string(b), "SECRET") || strings.Contains(string(b), "987654") {
		t.Fatal(string(b))
	}
	for _, d := range []string{"incoming", "outgoing", "internal"} {
		_, e := Apply(j, Action{Kind: "message", Revision: j.Revision, Direction: d, Channel: "email", Text: "spoof"}, testMechanic, testNow)
		if !errors.Is(e, ErrForbidden) {
			t.Fatal(e)
		}
	}
}

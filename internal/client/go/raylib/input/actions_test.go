package input

import (
	"testing"
)

func TestAllActionsHaveNames(t *testing.T) {
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		name := a.String()
		if name == "unknown" {
			t.Errorf("InputAction(%d) has no name in actionNames", int(a))
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		name := a.String()
		parsed, ok := ParseAction(name)
		if !ok {
			t.Errorf("ParseAction(%q) returned false", name)
			continue
		}
		if parsed != a {
			t.Errorf("round-trip failed: InputAction(%d).String()=%q → ParseAction → %d", int(a), name, int(parsed))
		}
	}
}

func TestAllActionsReturnsEachActionOnce(t *testing.T) {
	all := AllActions()
	if len(all) != int(actionCount)-1 {
		t.Errorf("AllActions() len = %d, want %d", len(all), int(actionCount)-1)
	}
	seen := make(map[InputAction]bool)
	for _, a := range all {
		if seen[a] {
			t.Errorf("duplicate action %v in AllActions()", a)
		}
		seen[a] = true
	}
}

func TestActionNoneIsZero(t *testing.T) {
	if ActionNone != 0 {
		t.Errorf("ActionNone = %d, want 0", ActionNone)
	}
}

func TestParseActionUnknown(t *testing.T) {
	_, ok := ParseAction("not.a.real.action")
	if ok {
		t.Error("ParseAction(unknown) returned true")
	}
}

func TestIsReplContext(t *testing.T) {
	replActions := []InputAction{
		ActionReplOpen, ActionReplClose, ActionReplHistoryPrev, ActionReplHistoryNext,
	}
	for _, a := range replActions {
		if !isReplContext(a) {
			t.Errorf("isReplContext(%v) = false, want true", a)
		}
	}
	worldActions := []InputAction{
		ActionThrustForward, ActionSimTrackStop, ActionCameraPitchUp,
	}
	for _, a := range worldActions {
		if isReplContext(a) {
			t.Errorf("isReplContext(%v) = true, want false", a)
		}
	}
}

package ui

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/space/engine"
)

// TestNewInputStateDefaults verifies the zero-value contract of NewInputState.
func TestNewInputStateDefaults(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)

	if inp.SelectionActive {
		t.Error("SelectionActive should be false by default")
	}
	if inp.SelectedIndex != -1 {
		t.Errorf("SelectedIndex = %d, want -1", inp.SelectedIndex)
	}
	if inp.SelectionMode != SelectionModeNone {
		t.Errorf("SelectionMode = %v, want SelectionModeNone", inp.SelectionMode)
	}
	if inp.SelectedCategory != engine.CategoryStar {
		t.Errorf("SelectedCategory = %v, want CategoryStar", inp.SelectedCategory)
	}
	if inp.FilteredIndices == nil {
		t.Error("FilteredIndices should be initialised (not nil)")
	}
	if inp.PerfOptions == nil {
		t.Error("PerfOptions should be initialised (not nil)")
	}
	if inp.PerformanceTab != 0 {
		t.Errorf("PerformanceTab = %d, want 0", inp.PerformanceTab)
	}
	if inp.FilterText != "" {
		t.Errorf("FilterText = %q, want empty string", inp.FilterText)
	}
	if inp.ScrollOffset != 0 {
		t.Errorf("ScrollOffset = %d, want 0", inp.ScrollOffset)
	}
	if inp.DistanceCache == nil {
		t.Error("DistanceCache should be initialised (not nil)")
	}
}

// TestStartSelection verifies that StartSelection sets the expected fields.
func TestStartSelection(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)

	inp.StartSelection(SelectionModeJump)

	if !inp.SelectionActive {
		t.Error("SelectionActive should be true after StartSelection")
	}
	if inp.SelectedIndex != 0 {
		t.Errorf("SelectedIndex = %d, want 0 after StartSelection", inp.SelectedIndex)
	}
	if inp.SelectionMode != SelectionModeJump {
		t.Errorf("SelectionMode = %v, want SelectionModeJump", inp.SelectionMode)
	}
}

// TestCancelSelection verifies that CancelSelection clears selection state.
func TestCancelSelection(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.StartSelection(SelectionModeTrack)

	inp.CancelSelection()

	if inp.SelectionActive {
		t.Error("SelectionActive should be false after CancelSelection")
	}
	if inp.SelectedIndex != -1 {
		t.Errorf("SelectedIndex = %d, want -1 after CancelSelection", inp.SelectedIndex)
	}
	if inp.SelectionMode != SelectionModeNone {
		t.Errorf("SelectionMode = %v, want SelectionModeNone after CancelSelection", inp.SelectionMode)
	}
}

// TestMainWindowInputSuspended verifies modal dialogs suspend main-window controls.
func TestMainWindowInputSuspended(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)

	if inp.MainWindowInputSuspended() {
		t.Fatal("MainWindowInputSuspended should be false without an active dialog")
	}

	inp.StartSelection(SelectionModeTrack)
	if !inp.MainWindowInputSuspended() {
		t.Fatal("MainWindowInputSuspended should be true while a dialog is active")
	}

	inp.CancelSelection()
	if inp.MainWindowInputSuspended() {
		t.Fatal("MainWindowInputSuspended should be false after the dialog closes")
	}

	var nilInput *InputState
	if nilInput.MainWindowInputSuspended() {
		t.Fatal("MainWindowInputSuspended should be false for a nil InputState")
	}
}

// TestConfirmSelectionReturnsValues verifies ConfirmSelection returns the current index and mode.
func TestConfirmSelectionReturnsValues(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.StartSelection(SelectionModeTrackEquatorial)
	inp.SelectedIndex = 7

	idx, mode := inp.ConfirmSelection()

	if idx != 7 {
		t.Errorf("ConfirmSelection index = %d, want 7", idx)
	}
	if mode != SelectionModeTrackEquatorial {
		t.Errorf("ConfirmSelection mode = %v, want SelectionModeTrackEquatorial", mode)
	}
	if inp.SelectionActive {
		t.Error("SelectionActive should be false after ConfirmSelection")
	}
	if inp.SelectedIndex != -1 {
		t.Errorf("SelectedIndex = %d, want -1 after ConfirmSelection", inp.SelectedIndex)
	}
	if inp.SelectionMode != SelectionModeNone {
		t.Errorf("SelectionMode = %v, want SelectionModeNone after ConfirmSelection", inp.SelectionMode)
	}
}

func TestOpenSystemSelectorPrefersActivePath(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	options := []SystemOption{
		{Label: "solar_system.json", Path: "data/systems/solar_system.json"},
		{Label: "alpha_centauri_system.json", Path: "data/systems/alpha_centauri_system.json"},
	}

	inp.OpenSystemSelector(options, "data/systems/alpha_centauri_system.json")

	if !inp.SelectionActive {
		t.Fatal("SelectionActive should be true after opening the system selector")
	}
	if inp.SelectionMode != SelectionModeSystemSelector {
		t.Fatalf("SelectionMode = %v, want SelectionModeSystemSelector", inp.SelectionMode)
	}
	if inp.SelectedIndex != 1 {
		t.Fatalf("SelectedIndex = %d, want 1 for active system", inp.SelectedIndex)
	}
	if len(inp.SystemOptions) != 2 {
		t.Fatalf("len(SystemOptions) = %d, want 2", len(inp.SystemOptions))
	}
}

func TestConfirmSystemSelectionQueuesReload(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.OpenSystemSelector([]SystemOption{
		{Label: "solar_system.json", Path: "data/systems/solar_system.json"},
		{Label: "alpha_centauri_system.json", Path: "data/systems/alpha_centauri_system.json"},
	}, "data/systems/solar_system.json")
	inp.SelectedIndex = 1

	selectedPath, reloadRequired := inp.ConfirmSystemSelection()

	if !reloadRequired {
		t.Fatal("reloadRequired should be true when a different system is chosen")
	}
	if selectedPath != "data/systems/alpha_centauri_system.json" {
		t.Fatalf("selectedPath = %q, want alpha centauri path", selectedPath)
	}
	if got := inp.ConsumePendingSystemPath(); got != selectedPath {
		t.Fatalf("ConsumePendingSystemPath() = %q, want %q", got, selectedPath)
	}
	if !inp.SelectionActive {
		t.Fatal("SelectionActive should remain true until the app completes or rejects reload")
	}
}

func TestConfirmSystemSelectionCancelsForActivePath(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.OpenSystemSelector([]SystemOption{
		{Label: "solar_system.json", Path: "data/systems/solar_system.json"},
	}, "data/systems/solar_system.json")

	selectedPath, reloadRequired := inp.ConfirmSystemSelection()

	if reloadRequired {
		t.Fatal("reloadRequired should be false when the active system is selected")
	}
	if selectedPath != "data/systems/solar_system.json" {
		t.Fatalf("selectedPath = %q, want active system path", selectedPath)
	}
	if inp.SelectionActive {
		t.Fatal("SelectionActive should be false after confirming the active system")
	}
	if got := inp.ConsumePendingSystemPath(); got != "" {
		t.Fatalf("ConsumePendingSystemPath() = %q, want empty string", got)
	}
}

// TestSelectNext verifies SelectNext increments up to maxIndex and no further.
func TestSelectNext(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.StartSelection(SelectionModeJump)

	inp.SelectNext(2)
	if inp.SelectedIndex != 1 {
		t.Errorf("SelectNext(2): got %d, want 1", inp.SelectedIndex)
	}

	inp.SelectNext(2)
	if inp.SelectedIndex != 2 {
		t.Errorf("SelectNext(2): got %d, want 2", inp.SelectedIndex)
	}

	inp.SelectNext(2)
	if inp.SelectedIndex != 2 {
		t.Errorf("SelectNext at maxIndex: got %d, want 2 (no change)", inp.SelectedIndex)
	}
}

// TestSelectPrevious verifies SelectPrevious decrements down to 0 and no further.
func TestSelectPrevious(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.StartSelection(SelectionModeJump)
	inp.SelectedIndex = 2

	inp.SelectPrevious()
	if inp.SelectedIndex != 1 {
		t.Errorf("SelectPrevious: got %d, want 1", inp.SelectedIndex)
	}

	inp.SelectPrevious()
	if inp.SelectedIndex != 0 {
		t.Errorf("SelectPrevious: got %d, want 0", inp.SelectedIndex)
	}

	inp.SelectPrevious()
	if inp.SelectedIndex != 0 {
		t.Errorf("SelectPrevious at 0: got %d, want 0 (no change)", inp.SelectedIndex)
	}
}

// TestSelectionModeValues documents known SelectionMode constants.
func TestSelectionModeValues(t *testing.T) {
	tests := []struct {
		name    string
		got     SelectionMode
		wantInt int
	}{
		{"SelectionModeNone", SelectionModeNone, 0},
		{"SelectionModeJump", SelectionModeJump, 1},
		{"SelectionModeTrack", SelectionModeTrack, 2},
		{"SelectionModeTrackEquatorial", SelectionModeTrackEquatorial, 3},
		{"SelectionModePerformance", SelectionModePerformance, 4},
		{"SelectionModeSystemSelector", SelectionModeSystemSelector, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.got) != tt.wantInt {
				t.Errorf("%s = %d, want %d", tt.name, int(tt.got), tt.wantInt)
			}
		})
	}
}

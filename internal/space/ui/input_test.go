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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.got) != tt.wantInt {
				t.Errorf("%s = %d, want %d", tt.name, int(tt.got), tt.wantInt)
			}
		})
	}
}

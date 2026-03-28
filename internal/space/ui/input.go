package ui

import "github.com/digital-michael/space_sim/internal/space/engine"

// SelectionMode represents what action to take after object selection.
type SelectionMode int

const (
	SelectionModeNone SelectionMode = iota
	SelectionModeJump
	SelectionModeTrack
	SelectionModeTrackEquatorial
	SelectionModePerformance
)

// PerformanceOptions holds runtime rendering/physics optimisation settings.
type PerformanceOptions struct {
	FrustumCulling      bool
	LODEnabled          bool
	InstancedRendering  bool
	SpatialPartition    bool
	PointRendering      bool
	ImportanceThreshold int
	UseInPlaceSwap      bool
}

// NewPerformanceOptions creates the default performance options.
func NewPerformanceOptions() *PerformanceOptions {
	return &PerformanceOptions{
		FrustumCulling:      true,
		LODEnabled:          false,
		InstancedRendering:  true,
		SpatialPartition:    true,
		PointRendering:      false,
		ImportanceThreshold: 0,
		UseInPlaceSwap:      true,
	}
}

// InputState holds current user input state for object selection and navigation.
type InputState struct {
	SelectionActive    bool
	SelectedIndex      int
	SelectionMode      SelectionMode
	SelectedCategory   engine.ObjectCategory
	FilteredIndices    []int
	PerfOptions        *PerformanceOptions
	PerformanceTab     int
	FilterText         string
	ScrollOffset       int
	DistanceCache      map[int]string
	LastDistanceUpdate float64
}

// NewInputState creates an InputState with firstCategory as the active tab.
func NewInputState(firstCategory engine.ObjectCategory) *InputState {
	return &InputState{
		SelectionActive:  false,
		SelectedIndex:    -1,
		SelectionMode:    SelectionModeNone,
		SelectedCategory: firstCategory,
		FilteredIndices:  make([]int, 0),
		PerfOptions:      NewPerformanceOptions(),
		PerformanceTab:   0,
		FilterText:       "",
		ScrollOffset:     0,
		DistanceCache:    make(map[int]string),
	}
}

// StartSelection begins the object selection process.
func (i *InputState) StartSelection(mode SelectionMode) {
	i.SelectionActive = true
	i.SelectedIndex = 0
	i.SelectionMode = mode
}

// MainWindowInputSuspended reports whether a modal dialog currently owns input.
func (i *InputState) MainWindowInputSuspended() bool {
	return i != nil && i.SelectionActive
}

// CancelSelection cancels the current selection.
func (i *InputState) CancelSelection() {
	i.SelectionActive = false
	i.SelectedIndex = -1
	i.SelectionMode = SelectionModeNone
}

// SelectNext moves to the next object in the list.
func (i *InputState) SelectNext(maxIndex int) {
	if i.SelectedIndex < maxIndex {
		i.SelectedIndex++
	}
}

// SelectPrevious moves to the previous object in the list.
func (i *InputState) SelectPrevious() {
	if i.SelectedIndex > 0 {
		i.SelectedIndex--
	}
}

// ConfirmSelection returns the selected index and mode, then resets state.
func (i *InputState) ConfirmSelection() (int, SelectionMode) {
	selected := i.SelectedIndex
	mode := i.SelectionMode
	i.SelectionActive = false
	i.SelectedIndex = -1
	i.SelectionMode = SelectionModeNone
	return selected, mode
}

// CycleCategory advances to the next category in the provided order slice.
func (i *InputState) CycleCategory(order []engine.ObjectCategory) {
	for idx, cat := range order {
		if cat == i.SelectedCategory {
			i.SelectedCategory = order[(idx+1)%len(order)]
			i.SelectedIndex = 0
			return
		}
	}
	if len(order) > 0 {
		i.SelectedCategory = order[0]
	}
	i.SelectedIndex = 0
}

// CycleCategoryBack moves to the previous category in the provided order slice.
func (i *InputState) CycleCategoryBack(order []engine.ObjectCategory) {
	for idx, cat := range order {
		if cat == i.SelectedCategory {
			prevIdx := (idx - 1 + len(order)) % len(order)
			i.SelectedCategory = order[prevIdx]
			i.SelectedIndex = 0
			return
		}
	}
	if len(order) > 0 {
		i.SelectedCategory = order[0]
	}
	i.SelectedIndex = 0
}

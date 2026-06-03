package app

import (
	"strings"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	render "github.com/digital-michael/space_sim/internal/client/go/raylib/ui/render"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInputTrackStop processes the ActionSimTrackStop keybinding (ESC cascade).
// The caller has already confirmed the key is pressed.
func (a *App) handleInputTrackStop(session *runtimeSession) bool {
	if a.runtime.SettingsVisible {
		if a.runtime.Settings.AvailableFiles != nil {
			a.runtime.Settings.AvailableFiles = nil
		} else if a.runtime.Settings.KeybindCapture >= 0 {
			a.runtime.Settings.KeybindCapture = -1
			a.runtime.Settings.KeybindConflict = ""
			a.runtime.Settings.KeybindCapturePendingMod = 0
		} else if a.runtime.Settings.SaveAsEditing {
			a.runtime.Settings.SaveAsPath = a.runtime.Settings.SaveAsPathPrev
			a.runtime.Settings.SaveAsEditing = false
		} else {
			a.runtime.SettingsVisible = false
		}
		return false
	}
	if session.inputState.SelectionActive {
		session.inputState.CancelSelection()
		return false
	}
	if session.cameraState.Mode == ui.CameraModeTracking {
		session.cameraState.StopTracking()
		return false
	}
	if a.runtime.MouseModeEnabled {
		a.runtime.MouseModeEnabled = false
		rl.EnableCursor()
		return false
	}
	return false
}

// handleInputSettings processes keyboard input for the settings dialog.
// Returns true only if the application should quit (never from this path).
func (a *App) handleInputSettings(session *runtimeSession, km *input.KeyMap, shiftHeld bool) bool {
	// Sync AdvancedInfo each frame so mouse clicks in the renderer take effect immediately.
	session.inputState.AdvancedInfo = a.runtime.Settings.AdvancedInfo

	// Keybind capture mode: intercept the next key press for rebinding.
	if a.runtime.Settings.KeybindCapture >= 0 {
		action := input.InputAction(a.runtime.Settings.KeybindCapture)
		pendingMod := a.runtime.Settings.KeybindCapturePendingMod

		if pendingMod != 0 {
			key, pressed := km.AnyPressed()
			if pressed && key == int32(rl.KeyEscape) {
				a.runtime.Settings.KeybindCapture = -1
				a.runtime.Settings.KeybindConflict = ""
				a.runtime.Settings.KeybindCapturePendingMod = 0
			} else if pressed && !isModifierKey(key) {
				mods := liveModSet()
				conflictAction, hasConflict := km.ConflictFor(key, mods, action)
				if hasConflict {
					a.runtime.Settings.KeybindConflict = "Conflict: key bound to " + conflictAction.String()
				} else {
					km.SetBinding(action, key, mods)
					a.runtime.Settings.KeybindConflict = ""
					a.runtime.Settings.KeybindsDirty = true
				}
				a.runtime.Settings.KeybindCapture = -1
				a.runtime.Settings.KeybindCapturePendingMod = 0
			} else if !anyModifierDown() {
				conflictAction, hasConflict := km.ConflictFor(pendingMod, 0, action)
				if hasConflict {
					a.runtime.Settings.KeybindConflict = "Conflict: key bound to " + conflictAction.String()
				} else {
					km.SetBinding(action, pendingMod, 0)
					a.runtime.Settings.KeybindConflict = ""
					a.runtime.Settings.KeybindsDirty = true
				}
				a.runtime.Settings.KeybindCapture = -1
				a.runtime.Settings.KeybindCapturePendingMod = 0
			}
			return false
		}

		key, pressed := km.AnyPressed()
		if pressed {
			if key == int32(rl.KeyEscape) {
				a.runtime.Settings.KeybindCapture = -1
				a.runtime.Settings.KeybindConflict = ""
			} else if isModifierKey(key) {
				a.runtime.Settings.KeybindCapturePendingMod = key
			} else {
				conflictAction, hasConflict := km.ConflictFor(key, 0, action)
				if hasConflict {
					a.runtime.Settings.KeybindConflict = "Conflict: key bound to " + conflictAction.String()
					a.runtime.Settings.KeybindCapture = -1
				} else {
					km.SetBinding(action, key, 0)
					a.runtime.Settings.KeybindConflict = ""
					a.runtime.Settings.KeybindsDirty = true
					a.runtime.Settings.KeybindCapture = -1
				}
			}
		}
		return false
	}

	// Save-As inline path editor.
	if a.runtime.Settings.SaveAsEditing {
		for {
			ch := rl.GetCharPressed()
			if ch == 0 {
				break
			}
			if ch >= 32 && ch < 127 {
				a.runtime.Settings.SaveAsPath += string(rune(ch))
			}
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && len(a.runtime.Settings.SaveAsPath) > 0 {
			runes := []rune(a.runtime.Settings.SaveAsPath)
			a.runtime.Settings.SaveAsPath = string(runes[:len(runes)-1])
		}
		if rl.IsKeyPressed(rl.KeyEnter) {
			if a.runtime.Settings.SaveAsPath != "" {
				if err := input.WriteKeybindingsFile(a.runtime.Settings.SaveAsPath, km, "laptop"); err == nil {
					a.runtime.KeybindingsPath = a.runtime.Settings.SaveAsPath
					a.runtime.Settings.KeybindingsPath = a.runtime.Settings.SaveAsPath
					a.runtime.Settings.KeybindsDirty = false
					a.cfg.AppConfig.KeybindingsPath = a.runtime.Settings.SaveAsPath
				}
			}
			a.runtime.Settings.SaveAsEditing = false
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			a.runtime.Settings.SaveAsPath = a.runtime.Settings.SaveAsPathPrev
			a.runtime.Settings.SaveAsEditing = false
		}
		return false
	}

	// File picker overlay.
	if a.runtime.Settings.AvailableFiles != nil {
		if rl.IsKeyPressed(rl.KeyEscape) {
			a.runtime.Settings.AvailableFiles = nil
			return false
		}
		maxIdx := len(a.runtime.Settings.AvailableFiles) - 1
		if rl.IsKeyPressed(rl.KeyUp) && a.runtime.Settings.FilePickerIdx > 0 {
			a.runtime.Settings.FilePickerIdx--
		}
		if rl.IsKeyPressed(rl.KeyDown) && a.runtime.Settings.FilePickerIdx < maxIdx {
			a.runtime.Settings.FilePickerIdx++
		}
		if rl.IsKeyPressed(rl.KeyLeft) {
			if a.runtime.Settings.FilePickerIdx > 0 {
				a.runtime.Settings.FilePickerIdx--
			} else {
				a.runtime.Settings.FilePickerIdx = maxIdx
			}
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			if a.runtime.Settings.FilePickerIdx < maxIdx {
				a.runtime.Settings.FilePickerIdx++
			} else {
				a.runtime.Settings.FilePickerIdx = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) {
			sel := a.runtime.Settings.AvailableFiles[a.runtime.Settings.FilePickerIdx]
			newKm, err := input.LoadKeyMap(defaultProfilesDir, sel)
			if err == nil {
				a.keyMap.Store(newKm)
				km = newKm
				a.runtime.KeybindingsPath = sel
				a.runtime.Settings.KeybindingsPath = sel
				a.runtime.Settings.SaveAsPath = sel
				a.runtime.Settings.KeybindsDirty = false
				a.cfg.AppConfig.KeybindingsPath = sel
			}
			a.runtime.Settings.AvailableFiles = nil
		}
		return false
	}

	// ESC: close the settings dialog (no sub-state is active at this point).
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.runtime.SettingsVisible = false
		return false
	}

	// Tab / Shift+Tab: switch active tab.
	if rl.IsKeyPressed(rl.KeyTab) {
		if shiftHeld {
			a.runtime.Settings.ActiveTab = (a.runtime.Settings.ActiveTab + 3) % 4
		} else {
			a.runtime.Settings.ActiveTab = (a.runtime.Settings.ActiveTab + 1) % 4
		}
		a.runtime.Settings.SelectedRow = 0
	}

	// Up/Down: navigate rows within current tab.
	tabRowMax := [4]int{0, 4, 6, 49}
	if a.runtime.Settings.ActiveTab > 0 {
		if rl.IsKeyPressed(rl.KeyUp) && a.runtime.Settings.SelectedRow > 0 {
			a.runtime.Settings.SelectedRow--
		}
		if rl.IsKeyPressed(rl.KeyDown) && a.runtime.Settings.SelectedRow < tabRowMax[a.runtime.Settings.ActiveTab] {
			a.runtime.Settings.SelectedRow++
		}
	}

	// Space/Enter: toggle selected option.
	if rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyEnter) {
		switch a.runtime.Settings.ActiveTab {
		case 1:
			switch a.runtime.Settings.SelectedRow {
			case 0:
				a.runtime.Settings.HUD.Debug = !a.runtime.Settings.HUD.Debug
				a.runtime.HUD.Debug = a.runtime.Settings.HUD.Debug
			case 1:
				a.runtime.Settings.HUD.Info = !a.runtime.Settings.HUD.Info
				a.runtime.HUD.Info = a.runtime.Settings.HUD.Info
			case 2:
				a.runtime.Settings.HUD.Help = !a.runtime.Settings.HUD.Help
				a.runtime.HUD.Help = a.runtime.Settings.HUD.Help
			case 3:
				a.runtime.Settings.AdvancedInfo = !a.runtime.Settings.AdvancedInfo
				session.inputState.AdvancedInfo = a.runtime.Settings.AdvancedInfo
			}
		case 2:
			switch a.runtime.Settings.SelectedRow {
			case 0:
				a.runtime.Settings.Perf.FrustumCulling = !a.runtime.Settings.Perf.FrustumCulling
				a.runtime.PerfConfig.FrustumCulling = a.runtime.Settings.Perf.FrustumCulling
				session.inputState.PerfOptions.FrustumCulling = a.runtime.Settings.Perf.FrustumCulling
			case 1:
				a.runtime.Settings.Perf.LODEnabled = !a.runtime.Settings.Perf.LODEnabled
				a.runtime.PerfConfig.LODEnabled = a.runtime.Settings.Perf.LODEnabled
				session.inputState.PerfOptions.LODEnabled = a.runtime.Settings.Perf.LODEnabled
			case 2:
				a.runtime.Settings.Perf.InstancedRendering = !a.runtime.Settings.Perf.InstancedRendering
				a.runtime.PerfConfig.InstancedRendering = a.runtime.Settings.Perf.InstancedRendering
				session.inputState.PerfOptions.InstancedRendering = a.runtime.Settings.Perf.InstancedRendering
			case 3:
				a.runtime.Settings.Perf.SpatialPartition = !a.runtime.Settings.Perf.SpatialPartition
				a.runtime.PerfConfig.SpatialPartition = a.runtime.Settings.Perf.SpatialPartition
				session.inputState.PerfOptions.SpatialPartition = a.runtime.Settings.Perf.SpatialPartition
			case 4:
				a.runtime.Settings.Perf.PointRendering = !a.runtime.Settings.Perf.PointRendering
				a.runtime.PerfConfig.PointRendering = a.runtime.Settings.Perf.PointRendering
				session.inputState.PerfOptions.PointRendering = a.runtime.Settings.Perf.PointRendering
			case 6:
				a.runtime.Settings.Perf.UseInPlaceSwap = !a.runtime.Settings.Perf.UseInPlaceSwap
				a.runtime.PerfConfig.UseInPlaceSwap = a.runtime.Settings.Perf.UseInPlaceSwap
				session.inputState.PerfOptions.UseInPlaceSwap = a.runtime.Settings.Perf.UseInPlaceSwap
				if session.sim != nil {
					if a.runtime.Settings.Perf.UseInPlaceSwap {
						session.sim.GetState().EnableInPlaceSwap()
					} else {
						session.sim.GetState().DisableInPlaceSwap()
					}
				}
			}
		case 3:
			switch a.runtime.Settings.SelectedRow {
			case 0:
				files, _ := input.ScanKeybindingsDir("configs")
				if len(files) > 0 {
					a.runtime.Settings.AvailableFiles = files
					a.runtime.Settings.FilePickerIdx = 0
				}
			case 1:
				a.runtime.Settings.SaveAsPathPrev = a.runtime.Settings.SaveAsPath
				a.runtime.Settings.SaveAsEditing = true
			default:
				actionIdx := a.runtime.Settings.SelectedRow - 2
				allActions := input.OrderedActions()
				if actionIdx >= 0 && actionIdx < len(allActions) {
					a.runtime.Settings.KeybindCapture = int(allActions[actionIdx])
					a.runtime.Settings.KeybindConflict = ""
				}
			}
		}
	}

	// Left/Right: UIScale (Display tab, row 4).
	if a.runtime.Settings.ActiveTab == 1 && a.runtime.Settings.SelectedRow == 4 {
		scales := []float32{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}
		current := a.runtime.Settings.UIScale
		if current <= 0 {
			current = 1.0
		}
		if rl.IsKeyPressed(rl.KeyLeft) {
			for i := len(scales) - 1; i >= 0; i-- {
				if current > scales[i]+0.01 {
					a.runtime.Settings.UIScale = scales[i]
					break
				}
			}
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			for i := 0; i < len(scales); i++ {
				if current < scales[i]-0.01 {
					a.runtime.Settings.UIScale = scales[i]
					break
				}
			}
		}
		if a.runtime.Settings.UIScale != current {
			a.runtime.UIScale = a.runtime.Settings.UIScale
			render.SetUIScale(a.runtime.UIScale)
		}
	}

	// Left/Right: ImportanceThreshold (Performance tab, row 5).
	if a.runtime.Settings.ActiveTab == 2 && a.runtime.Settings.SelectedRow == 5 {
		thresholds := []int{0, 5, 10, 15, 30, 40, 50, 60, 70, 80, 90}
		current := a.runtime.Settings.Perf.ImportanceThreshold
		if rl.IsKeyPressed(rl.KeyLeft) {
			for i := len(thresholds) - 1; i >= 0; i-- {
				if current > thresholds[i] {
					a.runtime.Settings.Perf.ImportanceThreshold = thresholds[i]
					break
				}
			}
			if current <= thresholds[0] {
				a.runtime.Settings.Perf.ImportanceThreshold = thresholds[len(thresholds)-1]
			}
			a.runtime.PerfConfig.ImportanceThreshold = a.runtime.Settings.Perf.ImportanceThreshold
			session.inputState.PerfOptions.ImportanceThreshold = a.runtime.Settings.Perf.ImportanceThreshold
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			for i := 0; i < len(thresholds); i++ {
				if current < thresholds[i] {
					a.runtime.Settings.Perf.ImportanceThreshold = thresholds[i]
					break
				}
			}
			if current >= thresholds[len(thresholds)-1] {
				a.runtime.Settings.Perf.ImportanceThreshold = thresholds[0]
			}
			a.runtime.PerfConfig.ImportanceThreshold = a.runtime.Settings.Perf.ImportanceThreshold
			session.inputState.PerfOptions.ImportanceThreshold = a.runtime.Settings.Perf.ImportanceThreshold
		}
	}

	return false
}

// skipSeparator steps idx in the given direction (±1) past any separator slots.
// Returns the adjusted index, clamped to [0, len(opts)-1].
func skipSeparator(opts []ui.SystemOption, idx, dir int) int {
	max := len(opts) - 1
	for idx >= 0 && idx <= max && opts[idx].IsSeparator {
		idx += dir
	}
	if idx < 0 {
		return 0
	}
	if idx > max {
		return max
	}
	return idx
}

// handleInputSelection processes keyboard input for the object-selection overlay.
// Returns true only if the application should quit (never from this path).
func (a *App) handleInputSelection(session *runtimeSession, state *engine.SimulationState, km *input.KeyMap) bool {
	// Script browser sub-mode.
	if session.inputState.SelectionMode == ui.SelectionModeScripts {
		is := session.inputState
		maxIndex := len(is.ScriptOptions) - 1
		pageSize := 10

		if rl.IsKeyPressed(rl.KeyUp) {
			is.SelectPrevious()
			if is.SelectedIndex < is.ScrollOffset {
				is.ScrollOffset = is.SelectedIndex
			}
		}
		if rl.IsKeyPressed(rl.KeyDown) {
			is.SelectNext(maxIndex)
			if is.SelectedIndex >= is.ScrollOffset+pageSize {
				is.ScrollOffset = is.SelectedIndex - pageSize + 1
			}
		}
		if rl.IsKeyPressed(rl.KeyPageUp) {
			is.SelectedIndex -= pageSize
			if is.SelectedIndex < 0 {
				is.SelectedIndex = 0
			}
			is.ScrollOffset = is.SelectedIndex
		}
		if rl.IsKeyPressed(rl.KeyPageDown) {
			is.SelectedIndex += pageSize
			if is.SelectedIndex > maxIndex {
				is.SelectedIndex = maxIndex
			}
			if is.SelectedIndex >= is.ScrollOffset+pageSize {
				is.ScrollOffset = is.SelectedIndex - pageSize + 1
			}
		}
		if rl.IsKeyPressed(rl.KeyHome) {
			is.SelectedIndex = 0
			is.ScrollOffset = 0
		}
		if rl.IsKeyPressed(rl.KeyEnd) {
			is.SelectedIndex = maxIndex
			is.ScrollOffset = maxIndex - pageSize + 1
			if is.ScrollOffset < 0 {
				is.ScrollOffset = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyEnter) {
			if path, ok := is.ConfirmScriptSelection(); ok {
				// Use snapSrc (always non-nil) so this works in both local
				// and remote-renderer mode (where session.sim is nil).
				snap := session.snapSrc.LatestSnapshot()
				a.startScript(a.runCtx, path, snap)
			}
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			is.CancelSelection()
		}
		return false
	}

	// System-selector sub-mode.
	if session.inputState.SelectionMode == ui.SelectionModeSystemSelector {
		maxIndex := len(session.inputState.SystemOptions) - 1
		pageSize := 10
		opts := session.inputState.SystemOptions

		if rl.IsKeyPressed(rl.KeyUp) {
			session.inputState.SelectPrevious()
			session.inputState.SelectedIndex = skipSeparator(opts, session.inputState.SelectedIndex, -1)
			if session.inputState.SelectedIndex < session.inputState.ScrollOffset {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex
			}
		}
		if rl.IsKeyPressed(rl.KeyDown) {
			session.inputState.SelectNext(maxIndex)
			session.inputState.SelectedIndex = skipSeparator(opts, session.inputState.SelectedIndex, 1)
			if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+pageSize {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex - pageSize + 1
			}
		}
		if rl.IsKeyPressed(rl.KeyPageUp) {
			session.inputState.SelectedIndex -= pageSize
			if session.inputState.SelectedIndex < 0 {
				session.inputState.SelectedIndex = 0
			}
			session.inputState.SelectedIndex = skipSeparator(opts, session.inputState.SelectedIndex, -1)
			session.inputState.ScrollOffset = session.inputState.SelectedIndex
		}
		if rl.IsKeyPressed(rl.KeyPageDown) {
			session.inputState.SelectedIndex += pageSize
			if session.inputState.SelectedIndex > maxIndex {
				session.inputState.SelectedIndex = maxIndex
			}
			session.inputState.SelectedIndex = skipSeparator(opts, session.inputState.SelectedIndex, 1)
			if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+pageSize {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex - pageSize + 1
			}
		}
		if rl.IsKeyPressed(rl.KeyHome) {
			if maxIndex >= 0 {
				session.inputState.SelectedIndex = 0
			}
			session.inputState.ScrollOffset = 0
		}
		if rl.IsKeyPressed(rl.KeyEnd) {
			session.inputState.SelectedIndex = skipSeparator(opts, maxIndex, -1)
			session.inputState.ScrollOffset = session.inputState.SelectedIndex - pageSize + 1
			if session.inputState.ScrollOffset < 0 {
				session.inputState.ScrollOffset = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyEnter) {
			session.inputState.ConfirmSystemSelection()
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			session.inputState.CancelSelection()
		}
		return false
	}

	// TAB / SHIFT+TAB: cycle selection mode (Face → Jump → Track → Face...).
	if rl.IsKeyPressed(rl.KeyTab) {
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
		modes := []ui.SelectionMode{ui.SelectionModeFace, ui.SelectionModeJump, ui.SelectionModeTrack}
		cur := session.inputState.SelectionMode
		idx := 0
		for i, m := range modes {
			if m == cur {
				idx = i
				break
			}
		}
		if shift {
			idx = (idx + len(modes) - 1) % len(modes)
		} else {
			idx = (idx + 1) % len(modes)
		}
		session.inputState.SelectionMode = modes[idx]
	}

	// Text input for filtering.
	char := rl.GetCharPressed()
	for char > 0 {
		if char >= 32 && char <= 126 {
			session.inputState.FilterText += string(rune(char))
			session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
			session.inputState.SelectedIndex = 0
			session.inputState.ScrollOffset = 0
		}
		char = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(session.inputState.FilterText) > 0 {
		session.inputState.FilterText = session.inputState.FilterText[:len(session.inputState.FilterText)-1]
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
		if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
			session.inputState.SelectedIndex = 0
		}
		session.inputState.ScrollOffset = 0
	}

	// Left/Right: category cycling.
	if rl.IsKeyPressed(rl.KeyLeft) {
		session.inputState.CycleCategoryBack(session.navigationOrder)
		session.inputState.FilterText = ""
		session.inputState.ScrollOffset = 0
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
		if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
			session.inputState.SelectedIndex = 0
		}
	}
	if rl.IsKeyPressed(rl.KeyRight) {
		session.inputState.CycleCategory(session.navigationOrder)
		session.inputState.FilterText = ""
		session.inputState.ScrollOffset = 0
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
		if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
			session.inputState.SelectedIndex = 0
		}
	}

	// Up/Down: item selection.
	if rl.IsKeyPressed(rl.KeyUp) {
		session.inputState.SelectPrevious()
		if session.inputState.SelectedIndex < session.inputState.ScrollOffset {
			session.inputState.ScrollOffset = session.inputState.SelectedIndex
		}
	}
	if rl.IsKeyPressed(rl.KeyDown) {
		session.inputState.SelectNext(len(session.inputState.FilteredIndices) - 1)
		visibleItems := selectionDialogVisibleItems()
		if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+visibleItems {
			session.inputState.ScrollOffset = session.inputState.SelectedIndex - visibleItems + 1
		}
	}

	// PageUp/PageDown.
	if rl.IsKeyPressed(rl.KeyPageUp) {
		visibleItems := selectionDialogVisibleItems()
		session.inputState.SelectedIndex -= visibleItems
		if session.inputState.SelectedIndex < 0 {
			session.inputState.SelectedIndex = 0
		}
		session.inputState.ScrollOffset = session.inputState.SelectedIndex
	}
	if rl.IsKeyPressed(rl.KeyPageDown) {
		visibleItems := selectionDialogVisibleItems()
		maxIndex := len(session.inputState.FilteredIndices) - 1
		session.inputState.SelectedIndex += visibleItems
		if session.inputState.SelectedIndex > maxIndex {
			session.inputState.SelectedIndex = maxIndex
		}
		if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+visibleItems {
			session.inputState.ScrollOffset = session.inputState.SelectedIndex - visibleItems + 1
		}
	}

	// Home/End.
	if rl.IsKeyPressed(rl.KeyHome) {
		session.inputState.SelectedIndex = 0
		session.inputState.ScrollOffset = 0
	}
	if rl.IsKeyPressed(rl.KeyEnd) {
		maxIndex := len(session.inputState.FilteredIndices) - 1
		session.inputState.SelectedIndex = maxIndex
		visibleItems := selectionDialogVisibleItems()
		session.inputState.ScrollOffset = maxIndex - visibleItems + 1
		if session.inputState.ScrollOffset < 0 {
			session.inputState.ScrollOffset = 0
		}
	}

	// ESC: cancel the selection dialog.
	if rl.IsKeyPressed(rl.KeyEscape) {
		session.inputState.CancelSelection()
		return false
	}

	// Enter: confirm selection.
	if rl.IsKeyPressed(rl.KeyEnter) {
		selectedIndex, mode := session.inputState.ConfirmSelection()
		if selectedIndex >= 0 && selectedIndex < len(session.inputState.FilteredIndices) {
			actualIndex := session.inputState.FilteredIndices[selectedIndex]
			if actualIndex == -1 {
				var asteroidIndices []int
				for i, obj := range state.Objects {
					if strings.HasPrefix(obj.Meta.Name, "Asteroid-") && obj.Visible {
						asteroidIndices = append(asteroidIndices, i)
					}
				}
				if len(asteroidIndices) > 0 {
					actualIndex = asteroidIndices[rl.GetRandomValue(0, int32(len(asteroidIndices)-1))]
				} else {
					return false
				}
			} else if actualIndex == -2 {
				var kboIndices []int
				for i, obj := range state.Objects {
					if strings.HasPrefix(obj.Meta.Name, "KBO-") && obj.Visible {
						kboIndices = append(kboIndices, i)
					}
				}
				if len(kboIndices) > 0 {
					actualIndex = kboIndices[rl.GetRandomValue(0, int32(len(kboIndices)-1))]
				} else {
					return false
				}
			}
			targetObj := state.Objects[actualIndex]
			switch mode {
			case ui.SelectionModeFace:
				session.cameraState.FaceTarget(targetObj.Anim.Position)
			case ui.SelectionModeJump:
				session.cameraState.StartJumpTo(actualIndex, targetObj.Anim.Position, ui.CalculateAutoZoomDistance(targetObj.Meta.ViewRadius(), 0.40))
			case ui.SelectionModeTrack:
				session.cameraState.StartTracking(actualIndex)
				session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(targetObj.Meta.ViewRadius(), 0.40)
			case ui.SelectionModeTrackEquatorial:
				session.cameraState.StartTrackingEquatorial(actualIndex)
				session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(targetObj.Meta.ViewRadius(), 0.40)
			}
		}
	}

	return false
}

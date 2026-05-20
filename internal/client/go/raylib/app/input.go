package app

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInput processes keyboard input for camera modes and object selection.
func (a *App) handleInput(session *runtimeSession, state *engine.SimulationState) bool {
	km := a.keyMap.Load()
	km.DrainQueue()

	selectionDialogOpen := session.inputState.MainWindowInputSuspended()
	mainWindowInputSuspended := selectionDialogOpen || a.runtime.SettingsVisible
	shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

	// ui.system_selector: Open the runtime system selector.
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUISystemSelector) {
		a.openSystemSelector(session.inputState)
		return false
	}

	// ui.label_cycle: Cycle label mode (off → on → nearest → off)
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUILabelCycle) {
		switch a.runtime.LabelMode {
		case ui.LabelModeOff:
			a.runtime.LabelMode = ui.LabelModeOn
		case ui.LabelModeOn:
			a.runtime.LabelMode = ui.LabelModeNearest
		default:
			a.runtime.LabelMode = ui.LabelModeOff
		}
	}

	// ui.infra_cycle: Cycle infra ambient-light mode
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUIInfraCycle) {
		a.runtime.InfraMode = (a.runtime.InfraMode + 1) % 3
	}

	// ui.mouse_mode_toggle: Toggle mouse mode (camera control vs UI cursor)
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUIMouseModeToggle) {
		a.runtime.MouseModeEnabled = !a.runtime.MouseModeEnabled
		if a.runtime.MouseModeEnabled {
			rl.DisableCursor()
		} else {
			rl.EnableCursor()
		}
	}

	// ui.quit: Quit application
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUIQuit) {
		return true
	}

	// sim.timescale_decrease / sim.timescale_increase: Step through time rates
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimTimescaleDecrease) {
		back := session.sim.GetState().GetBack()
		// Time rates: paused, real-time, 1 hour/sec, 1 day/sec, 1 week/sec, 1 month/sec, 1 year/sec
		timeRates := []float32{0.0, 1.0, 3600.0, 86400.0, 604800.0, 2628000.0, 31557600.0}
		timeLabels := []string{"PAUSED", "real-time", "1 hr/sec", "1 day/sec", "1 week/sec", "1 month/sec", "1 year/sec"}

		currentIdx := -1
		for i, rate := range timeRates {
			if math.Abs(float64(rate-back.SecondsPerSecond)) < 0.01 {
				currentIdx = i
				break
			}
		}

		if currentIdx > 0 {
			back.SecondsPerSecond = timeRates[currentIdx-1]
			fmt.Printf("Time rate: %s\n", timeLabels[currentIdx-1])
		} else if currentIdx == -1 && back.SecondsPerSecond > 0 {
			// Find closest lower rate
			for i := len(timeRates) - 1; i >= 0; i-- {
				if timeRates[i] < back.SecondsPerSecond {
					back.SecondsPerSecond = timeRates[i]
					fmt.Printf("Time rate: %s\n", timeLabels[i])
					break
				}
			}
		}
	}
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimTimescaleIncrease) {
		back := session.sim.GetState().GetBack()
		timeRates := []float32{0.0, 1.0, 3600.0, 86400.0, 604800.0, 2628000.0, 31557600.0}
		timeLabels := []string{"PAUSED", "real-time", "1 hr/sec", "1 day/sec", "1 week/sec", "1 month/sec", "1 year/sec"}

		currentIdx := -1
		for i, rate := range timeRates {
			if math.Abs(float64(rate-back.SecondsPerSecond)) < 0.01 {
				currentIdx = i
				break
			}
		}

		if currentIdx >= 0 && currentIdx < len(timeRates)-1 {
			back.SecondsPerSecond = timeRates[currentIdx+1]
			fmt.Printf("Time rate: %s\n", timeLabels[currentIdx+1])
		} else if currentIdx == -1 {
			// Find closest higher rate
			for i := 0; i < len(timeRates); i++ {
				if timeRates[i] > back.SecondsPerSecond {
					back.SecondsPerSecond = timeRates[i]
					fmt.Printf("Time rate: %s\n", timeLabels[i])
					break
				}
			}
		}
	}

	// sim.dataset_increase / sim.dataset_decrease: Increase/decrease asteroid dataset
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimDatasetIncrease) {
		if a.runtime.AsteroidDataset < 3 {
			a.runtime.AsteroidDataset++
			session.sim.SetAsteroidDataset(a.runtime.AsteroidDataset)
		}
	}
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimDatasetDecrease) {
		if a.runtime.AsteroidDataset > 0 {
			a.runtime.AsteroidDataset--
			session.sim.SetAsteroidDataset(a.runtime.AsteroidDataset)
		}
	}

	// sim.tick_speed_decrease / sim.tick_speed_increase: Physics tick rate
	// Controls how many sim ticks fire per real second (0%–100%)
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimTickSpeedDecrease) {
		// Decrease anim speed
		currentSpeed := session.sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := len(speedSteps) - 1; i >= 0; i-- {
			if currentSpeed > speedSteps[i] {
				session.sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimTickSpeedIncrease) {
		currentSpeed := session.sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := 0; i < len(speedSteps); i++ {
			if currentSpeed < speedSteps[i] {
				session.sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}

	// ui.record_toggle / ui.record_pause
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUIRecordToggle) {
		if a.runtime.RecordingActive {
			a.stopRecording()
		} else {
			a.startRecording("")
		}
	}
	if !mainWindowInputSuspended && km.IsPressed(input.ActionUIRecordPause) {
		if a.runtime.RecordingActive {
			a.runtime.RecordingPaused = !a.runtime.RecordingPaused
			if a.runtime.RecordingPaused {
				fmt.Println("[REC] Paused")
			} else {
				fmt.Println("[REC] Resumed")
			}
		}
	}

	// camera.center: Center view (free-fly: face origin; tracking: reset zoom)
	if !mainWindowInputSuspended && km.IsPressed(input.ActionCameraCenter) {
		if session.cameraState.Mode == ui.CameraModeFree {
			// Free-fly mode: center camera view on origin (sun)
			toOrigin := engine.Vector3{X: 0, Y: 0, Z: 0}.Sub(session.cameraState.Position)
			if toOrigin.Length() > 0.1 {
				session.cameraState.Forward = toOrigin.Normalize()
				// Update yaw and pitch from forward vector
				session.cameraState.Yaw = math.Atan2(float64(session.cameraState.Forward.X), float64(session.cameraState.Forward.Z))
				session.cameraState.Pitch = math.Asin(float64(session.cameraState.Forward.Y))
			}
		} else if session.cameraState.Mode == ui.CameraModeTracking {
			// Tracking mode: reset zoom to 24% auto-zoom distance
			if session.cameraState.TrackTargetIndex >= 0 && session.cameraState.TrackTargetIndex < len(state.Objects) {
				targetObj := state.Objects[session.cameraState.TrackTargetIndex]
				session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.24)
			}
		}
	}

	// nav.child_next / nav.parent: Drill into child or move to parent in hierarchy
	if session.cameraState.Mode == ui.CameraModeTracking && !mainWindowInputSuspended {
		if km.IsPressed(input.ActionNavChildNext) {
			if session.cameraState.TrackTargetIndex >= 0 && session.cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[session.cameraState.TrackTargetIndex]

				// F: Drill down to closest child
				if a.cfg.Debug {
					fmt.Printf("[DEBUG] F key: Drilling down from %s\n", currentObj.Meta.Name)
				}
				// Find all visible children (objects whose ParentName == current object's Name)
				// Special case: if current is a star, also include objects with empty ParentName that are orbiting
				type ChildInfo struct {
					index    int
					distance float32
				}
				children := []ChildInfo{}

				isStar := currentObj.Meta.Category == engine.CategoryStar

				for i, obj := range state.Objects {
					isChild := false
					if obj.Visible && obj.Meta.ParentName == currentObj.Meta.Name {
						isChild = true
					} else if isStar && obj.Visible && obj.Meta.ParentName == "" && (obj.Meta.SemiMajorAxis > 0 || obj.Meta.OrbitRadius > 0) {
						// Object orbits the star but has no explicit parent
						isChild = true
					}

					if isChild {
						// Skip rings - they're not navigation targets
						if obj.Meta.Category == engine.CategoryRing {
							continue
						}
						// Use SemiMajorAxis or OrbitRadius as distance
						distance := obj.Meta.SemiMajorAxis
						if distance == 0 {
							distance = obj.Meta.OrbitRadius
						}
						children = append(children, ChildInfo{index: i, distance: distance})
					}
				}

				if a.cfg.Debug {
					fmt.Printf("[DEBUG] Found %d children\n", len(children))
				}
				// Sort children by distance (closest first)
				if len(children) > 0 {
					sort.Slice(children, func(i, j int) bool {
						return children[i].distance < children[j].distance
					})

					// Track the closest child
					closestChild := state.Objects[children[0].index]
					if a.cfg.Debug {
						fmt.Printf("[DEBUG] Tracking closest child: %s\n", closestChild.Meta.Name)
					}
					session.cameraState.StartTracking(children[0].index)
					session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(closestChild.Meta.PhysicalRadius, 0.24)
				}
			}
		}

		if km.IsPressed(input.ActionNavParent) {
			if session.cameraState.TrackTargetIndex >= 0 && session.cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[session.cameraState.TrackTargetIndex]

				// B: Move up to parent
				if a.cfg.Debug {
					fmt.Printf("[DEBUG] B key: Moving to parent of %s\n", currentObj.Meta.Name)
				}
				if currentObj.Meta.ParentName != "" {
					// Find parent object by name
					for i, obj := range state.Objects {
						if obj.Meta.Name == currentObj.Meta.ParentName {
							if a.cfg.Debug {
								fmt.Printf("[DEBUG] Found parent: %s\n", obj.Meta.Name)
							}
							session.cameraState.StartTracking(i)
							session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
							break
						}
					}
				} else if currentObj.Meta.SemiMajorAxis > 0 || currentObj.Meta.OrbitRadius > 0 {
					// No explicit parent but is orbiting - find the star
					if a.cfg.Debug {
						fmt.Printf("[DEBUG] No explicit parent, looking for central star\n")
					}
					for i, obj := range state.Objects {
						if obj.Meta.Category == engine.CategoryStar {
							if a.cfg.Debug {
								fmt.Printf("[DEBUG] Found central star: %s\n", obj.Meta.Name)
							}
							session.cameraState.StartTracking(i)
							session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
							break
						}
					}
				} else {
					if a.cfg.Debug {
						fmt.Printf("[DEBUG] No parent for %s (already at star)\n", currentObj.Meta.Name)
					}
				}
			}
		}
	}

	// nav.sibling_next / nav.sibling_prev: Cycle through siblings when tracking
	if session.cameraState.Mode == ui.CameraModeTracking && !mainWindowInputSuspended {
		siblingForward := km.IsPressed(input.ActionNavSiblingNext)
		siblingBack := km.IsPressed(input.ActionNavSiblingPrev)

		if siblingForward || siblingBack {
			if a.cfg.Debug {
				fmt.Printf("[DEBUG] sibling nav: forward=%v back=%v\n", siblingForward, siblingBack)
			}

			if session.cameraState.TrackTargetIndex >= 0 && session.cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[session.cameraState.TrackTargetIndex]

				// TAB: Cycle through siblings (same parent, same category)
				siblings := []int{}
				for i, obj := range state.Objects {
					if obj.Visible &&
						obj.Meta.Category == currentObj.Meta.Category &&
						obj.Meta.ParentName == currentObj.Meta.ParentName {
						siblings = append(siblings, i)
					}
				}

				// Only cycle if there are multiple siblings
				if len(siblings) > 1 {
					// Find current object in siblings list
					currentPos := -1
					for i, idx := range siblings {
						if idx == session.cameraState.TrackTargetIndex {
							currentPos = i
							break
						}
					}

					if currentPos >= 0 {
						var nextPos int
						if siblingBack {
							// Shift+TAB: go backwards
							nextPos = currentPos - 1
							if nextPos < 0 {
								nextPos = len(siblings) - 1 // Wrap to end
							}
						} else {
							// TAB: go forwards
							nextPos = currentPos + 1
							if nextPos >= len(siblings) {
								nextPos = 0 // Wrap to beginning
							}
						}

						nextIndex := siblings[nextPos]
						// Start tracking the next sibling with auto-zoom
						nextObj := state.Objects[nextIndex]
						session.cameraState.StartTracking(nextIndex)
						session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(nextObj.Meta.PhysicalRadius, 0.24)
					}
				}
			}
		}
	}

	// ESC / sim.track_stop: Cancel selection, exit tracking, or exit mouse mode (priority order).
	// Also handles F11 (ui.fullscreen) and F1 (ui.settings) as keymap-driven actions.
	if km.IsPressed(input.ActionUIFullscreen) {
		a.toggleFullscreen()
	}
	if km.IsPressed(input.ActionUISettings) {
		a.runtime.SettingsVisible = !a.runtime.SettingsVisible
		if a.runtime.SettingsVisible && a.runtime.MouseModeEnabled {
			a.runtime.MouseModeEnabled = false
			rl.EnableCursor()
		}
	}
	if km.IsPressed(input.ActionSimTrackStop) {
		if a.runtime.SettingsVisible {
			// Exit sub-states (file picker → capture → save-as editing) before closing dialog.
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
		} else if session.inputState.SelectionActive {
			session.inputState.CancelSelection()
			return false
		} else if session.cameraState.Mode == ui.CameraModeTracking {
			session.cameraState.StopTracking()
			return false
		} else if a.runtime.MouseModeEnabled {
			// Exit mouse mode, enable cursor
			a.runtime.MouseModeEnabled = false
			rl.EnableCursor()
			return false
		}
	}

	// Settings dialog keyboard navigation (only when dialog is open, takes priority over main window)
	if a.runtime.SettingsVisible {
		// Capture mode: intercept the next key press for rebinding an action.
		if a.runtime.Settings.KeybindCapture >= 0 {
			action := input.InputAction(a.runtime.Settings.KeybindCapture)
			pendingMod := a.runtime.Settings.KeybindCapturePendingMod

			if pendingMod != 0 {
				// A modifier was pressed; waiting for a second key or mod release.
				key, pressed := km.AnyPressed()
				if pressed && key == int32(rl.KeyEscape) {
					// Cancel
					a.runtime.Settings.KeybindCapture = -1
					a.runtime.Settings.KeybindConflict = ""
					a.runtime.Settings.KeybindCapturePendingMod = 0
				} else if pressed && !isModifierKey(key) {
					// Second (non-modifier) key: bind with live mod set.
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
					// All modifiers released without a second key: bind to the modifier key itself.
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
				return false // all input suspended during capture
			}

			// No pending modifier: wait for first key press.
			key, pressed := km.AnyPressed()
			if pressed {
				if key == int32(rl.KeyEscape) {
					a.runtime.Settings.KeybindCapture = -1
					a.runtime.Settings.KeybindConflict = ""
				} else if isModifierKey(key) {
					// Modifier pressed first: enter pending-mod state.
					a.runtime.Settings.KeybindCapturePendingMod = key
				} else {
					// Direct non-modifier key.
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
			return false // all other input suspended during capture
		}

		// Save-As inline path editor: captures text input for the destination path.
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
			return false // all input suspended during path editing
		}

		// File picker overlay: UP/DOWN/L/R navigates, ENTER loads, ESC dismisses.
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

		// Tab / Shift+Tab: switch active tab
		if rl.IsKeyPressed(rl.KeyTab) {
			if shiftHeld {
				a.runtime.Settings.ActiveTab = (a.runtime.Settings.ActiveTab + 3) % 4
			} else {
				a.runtime.Settings.ActiveTab = (a.runtime.Settings.ActiveTab + 1) % 4
			}
			a.runtime.Settings.SelectedRow = 0
		}
		// Up/Down: navigate rows within current tab
		// Tab 2 (Performance): 7 rows (0-6); Tab 3 (Controls): Load + SaveAs + 48 actions (0-49)
		tabRowMax := [4]int{0, 2, 6, 49}
		if a.runtime.Settings.ActiveTab > 0 { // System tab has no row selection
			if rl.IsKeyPressed(rl.KeyUp) && a.runtime.Settings.SelectedRow > 0 {
				a.runtime.Settings.SelectedRow--
			}
			if rl.IsKeyPressed(rl.KeyDown) && a.runtime.Settings.SelectedRow < tabRowMax[a.runtime.Settings.ActiveTab] {
				a.runtime.Settings.SelectedRow++
			}
		}
		// Space/Enter: toggle selected option
		if rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyEnter) {
			switch a.runtime.Settings.ActiveTab {
			case 1: // Display tab — HUD flags
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
				}
			case 2: // Performance tab — 5 toggles + ImportanceThreshold (row 5) + UseInPlaceSwap (row 6)
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
				case 6: // UseInPlaceSwap (row 5 = ImportanceThreshold, no Enter action there)
					a.runtime.Settings.Perf.UseInPlaceSwap = !a.runtime.Settings.Perf.UseInPlaceSwap
					a.runtime.PerfConfig.UseInPlaceSwap = a.runtime.Settings.Perf.UseInPlaceSwap
					session.inputState.PerfOptions.UseInPlaceSwap = a.runtime.Settings.Perf.UseInPlaceSwap
					if a.runtime.Settings.Perf.UseInPlaceSwap {
						session.sim.GetState().EnableInPlaceSwap()
					} else {
						session.sim.GetState().DisableInPlaceSwap()
					}
				}
			case 3: // Controls tab — keybinding editor
				switch a.runtime.Settings.SelectedRow {
				case 0: // Load: scan configs/ and open file picker
					files, _ := input.ScanKeybindingsDir("configs")
					if len(files) > 0 {
						a.runtime.Settings.AvailableFiles = files
						a.runtime.Settings.FilePickerIdx = 0
					}
				case 1: // Save As: begin inline path editing
					a.runtime.Settings.SaveAsPathPrev = a.runtime.Settings.SaveAsPath
					a.runtime.Settings.SaveAsEditing = true
				default: // Rows 2..49 = keybinding rows for actions 0..47 in OrderedActions()
					actionIdx := a.runtime.Settings.SelectedRow - 2
					allActions := input.OrderedActions()
					if actionIdx >= 0 && actionIdx < len(allActions) {
						a.runtime.Settings.KeybindCapture = int(allActions[actionIdx])
						a.runtime.Settings.KeybindConflict = ""
					}
				}
			}
		}
		// Left/Right: adjust ImportanceThreshold (Performance tab, row 5)
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

	// Object selection mode
	if session.inputState.SelectionActive {
		if session.inputState.SelectionMode == ui.SelectionModeSystemSelector {
			maxIndex := len(session.inputState.SystemOptions) - 1
			pageSize := 10

			if rl.IsKeyPressed(rl.KeyUp) {
				session.inputState.SelectPrevious()
				if session.inputState.SelectedIndex < session.inputState.ScrollOffset {
					session.inputState.ScrollOffset = session.inputState.SelectedIndex
				}
			}
			if rl.IsKeyPressed(rl.KeyDown) {
				session.inputState.SelectNext(maxIndex)
				if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+pageSize {
					session.inputState.ScrollOffset = session.inputState.SelectedIndex - pageSize + 1
				}
			}
			if rl.IsKeyPressed(rl.KeyPageUp) {
				session.inputState.SelectedIndex -= pageSize
				if session.inputState.SelectedIndex < 0 {
					session.inputState.SelectedIndex = 0
				}
				session.inputState.ScrollOffset = session.inputState.SelectedIndex
			}
			if rl.IsKeyPressed(rl.KeyPageDown) {
				session.inputState.SelectedIndex += pageSize
				if session.inputState.SelectedIndex > maxIndex {
					session.inputState.SelectedIndex = maxIndex
				}
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
				session.inputState.SelectedIndex = maxIndex
				session.inputState.ScrollOffset = maxIndex - pageSize + 1
				if session.inputState.ScrollOffset < 0 {
					session.inputState.ScrollOffset = 0
				}
			}
			if rl.IsKeyPressed(rl.KeyEnter) {
				session.inputState.ConfirmSystemSelection()
			}

			return false
		}

		// Text input for filtering
		{
			// Capture character input
			char := rl.GetCharPressed()
			for char > 0 {
				// Add printable characters to filter text
				if char >= 32 && char <= 126 {
					session.inputState.FilterText += string(rune(char))
					// Rebuild filtered list with new filter
					session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
					session.inputState.SelectedIndex = 0
					session.inputState.ScrollOffset = 0 // Reset scroll when filtering
				}
				char = rl.GetCharPressed()
			}

			// Backspace to delete characters
			if rl.IsKeyPressed(rl.KeyBackspace) && len(session.inputState.FilterText) > 0 {
				session.inputState.FilterText = session.inputState.FilterText[:len(session.inputState.FilterText)-1]
				// Rebuild filtered list
				session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
				if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
					session.inputState.SelectedIndex = 0
				}
				session.inputState.ScrollOffset = 0 // Reset scroll when filtering
			}
		}

		// Left/Right arrow keys for category cycling
		if rl.IsKeyPressed(rl.KeyLeft) {
			session.inputState.CycleCategoryBack(session.navigationOrder)
			session.inputState.FilterText = ""  // Clear filter when changing category
			session.inputState.ScrollOffset = 0 // Reset scroll position
			// Rebuild filtered list
			session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
			if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
				session.inputState.SelectedIndex = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			session.inputState.CycleCategory(session.navigationOrder)
			session.inputState.FilterText = ""  // Clear filter when changing category
			session.inputState.ScrollOffset = 0 // Reset scroll position
			// Rebuild filtered list
			session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
			if session.inputState.SelectedIndex >= len(session.inputState.FilteredIndices) {
				session.inputState.SelectedIndex = 0
			}
		}
		// Up/Down arrow keys for selection within category
		if rl.IsKeyPressed(rl.KeyUp) {
			session.inputState.SelectPrevious()
			// Auto-scroll to keep selection visible
			if session.inputState.SelectedIndex < session.inputState.ScrollOffset {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex
			}
		}
		if rl.IsKeyPressed(rl.KeyDown) {
			session.inputState.SelectNext(len(session.inputState.FilteredIndices) - 1)
			// Auto-scroll to keep selection visible.
			// Replicates drawSelectionUI panel calculation: 40% of screen width, clamped [400,700], square.
			// Fixed header height is 155px (startY=145 + bottom-pad=10); lineHeight=30.
			visibleItems := selectionDialogVisibleItems()
			if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+visibleItems {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex - visibleItems + 1
			}
		}
		// Page Up/Down for faster navigation
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
			// Auto-scroll to keep selection visible
			if session.inputState.SelectedIndex >= session.inputState.ScrollOffset+visibleItems {
				session.inputState.ScrollOffset = session.inputState.SelectedIndex - visibleItems + 1
			}
		}
		// Home/End for jumping to start/end
		if rl.IsKeyPressed(rl.KeyHome) {
			session.inputState.SelectedIndex = 0
			session.inputState.ScrollOffset = 0
		}
		if rl.IsKeyPressed(rl.KeyEnd) {
			maxIndex := len(session.inputState.FilteredIndices) - 1
			session.inputState.SelectedIndex = maxIndex
			// Scroll to show last item
			visibleItems := selectionDialogVisibleItems()
			session.inputState.ScrollOffset = maxIndex - visibleItems + 1
			if session.inputState.ScrollOffset < 0 {
				session.inputState.ScrollOffset = 0
			}
		}
		// Enter to confirm
		if rl.IsKeyPressed(rl.KeyEnter) {
			selectedIndex, mode := session.inputState.ConfirmSelection()
			// Map from filtered index to actual object index
			if selectedIndex >= 0 && selectedIndex < len(session.inputState.FilteredIndices) {
				actualIndex := session.inputState.FilteredIndices[selectedIndex]

				// Handle virtual belt indices - select random object from belt
				if actualIndex == -1 {
					// Asteroid Belt - find a random asteroid
					var asteroidIndices []int
					for i, obj := range state.Objects {
						if strings.HasPrefix(obj.Meta.Name, "Asteroid-") && obj.Visible {
							asteroidIndices = append(asteroidIndices, i)
						}
					}
					if len(asteroidIndices) > 0 {
						actualIndex = asteroidIndices[rl.GetRandomValue(0, int32(len(asteroidIndices)-1))]
					} else {
						// No visible asteroids, cancel
						return false
					}
				} else if actualIndex == -2 {
					// Kuiper Belt - find a random KBO
					var kboIndices []int
					for i, obj := range state.Objects {
						if strings.HasPrefix(obj.Meta.Name, "KBO-") && obj.Visible {
							kboIndices = append(kboIndices, i)
						}
					}
					if len(kboIndices) > 0 {
						actualIndex = kboIndices[rl.GetRandomValue(0, int32(len(kboIndices)-1))]
					} else {
						// No visible KBOs, cancel
						return false
					}
				}

				targetObj := state.Objects[actualIndex]
				if mode == ui.SelectionModeJump {
					// Jump to object with good viewing distance (5x radius)
					session.cameraState.StartJumpTo(actualIndex, targetObj.Anim.Position, float64(targetObj.Meta.PhysicalRadius)*5.0)
				} else if mode == ui.SelectionModeTrack {
					session.cameraState.StartTracking(actualIndex)
					session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.24)
				} else if mode == ui.SelectionModeTrackEquatorial {
					// Start tracking from surface view - closer zoom (40% of screen height)
					session.cameraState.StartTrackingEquatorial(actualIndex)
					session.cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.40)
				}
			}
		}
		return false // Don't process other keys during selection
	}

	// nav.jump: Jump to object (free-fly mode only)
	if !mainWindowInputSuspended && km.IsPressed(input.ActionNavJump) && session.cameraState.Mode == ui.CameraModeFree {
		session.inputState.StartSelection(ui.SelectionModeJump)
		session.inputState.FilterText = ""
		session.inputState.ScrollOffset = 0
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
	}

	// T / sim.track_next: Open the tracking dialog with the default tracking mode.
	if !mainWindowInputSuspended && km.IsPressed(input.ActionSimTrackNext) && (session.cameraState.Mode == ui.CameraModeFree || session.cameraState.Mode == ui.CameraModeTracking) {
		session.inputState.StartSelection(ui.SelectionModeTrack)
		session.inputState.FilterText = ""
		session.inputState.ScrollOffset = 0
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
	}

	return false
}

// updateCameraState updates camera position and orientation based on mode
func (a *App) updateCameraState(session *runtimeSession, state *engine.SimulationState, dt float32) float32 {
	km := a.keyMap.Load()
	mainWindowInputSuspended := session.inputState.MainWindowInputSuspended() || a.runtime.SettingsVisible

	// Mouse look (only active when mouse mode is enabled)
	var mouseDelta rl.Vector2
	if a.runtime.MouseModeEnabled && !mainWindowInputSuspended {
		mouseDelta = rl.GetMouseDelta()
	}

	// Mouse wheel for zoom in all modes
	wheelMove := float32(0.0)
	if !mainWindowInputSuspended {
		wheelMove = rl.GetMouseWheelMove()
	}
	zoomSpeed := float32(0.0)

	if wheelMove != 0 {
		// Two-finger scroll zoom: move camera forward/backward along view direction
		zoomSpeed = wheelMove * a.runtime.CameraSpeed * 0.5 // Reduced from 2.0 to 0.5 (1/4 speed)

		switch session.cameraState.Mode {
		case ui.CameraModeTracking:
			// In tracking mode, adjust distance from target
			session.cameraState.TrackDistance -= float64(zoomSpeed * 10.0)
			// Clamp to reasonable values
			if session.cameraState.TrackDistance < engine.CameraTrackDistMin {
				session.cameraState.TrackDistance = engine.CameraTrackDistMin
			}
			if session.cameraState.TrackDistance > engine.CameraTrackDistMax {
				session.cameraState.TrackDistance = engine.CameraTrackDistMax
			}

		case ui.CameraModeFree, ui.CameraModeJumping:
			// In free/jumping mode, move camera along forward direction
			session.cameraState.Position = session.cameraState.Position.Add(session.cameraState.Forward.Scale(zoomSpeed * 10.0))
		}
	}

	// Arrow keys for movement in the system plane (active in all modes)
	arrowSpeed := a.runtime.CameraSpeed * dt // Same base speed as WASD
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		arrowSpeed *= 2.0 // Consistent 2x speed boost
	}

	// Update based on camera mode
	switch session.cameraState.Mode {
	case ui.CameraModeJumping:
		session.cameraState.UpdateJump(float64(dt))

		// Jump just arrived — center on the target body (track it) so it
		// is immediately visible on screen at the correct zoom distance.
		// Running UpdateTracking in the same frame avoids a one-frame snap.
		if session.cameraState.Mode == ui.CameraModeFree {
			session.cameraState.StartTracking(session.cameraState.JumpTargetIndex)
			session.cameraState.TrackDistance = session.cameraState.JumpTargetViewDist
			session.cameraState.TrackOffset = engine.Vector3{}
			session.cameraState.UpdateTracking(state) // center now, not next frame
			// Apply any orbit that was queued while the jump was in flight.
			if session.cameraState.PendingOrbitSpeed != 0 {
				session.cameraState.OrbitSpeed = session.cameraState.PendingOrbitSpeed
				session.cameraState.OrbitRadiansRemaining = session.cameraState.PendingOrbitRadians
				session.cameraState.PendingOrbitSpeed = 0
				session.cameraState.PendingOrbitRadians = 0
			}
			session.cameraState.JumpDwellRemaining = session.cameraState.JumpCurrentDwell
			// No dwell: immediately pop the next hop if queued.
			if session.cameraState.JumpDwellRemaining <= 0 && len(session.cameraState.JumpQueue) > 0 {
				next := session.cameraState.JumpQueue[0]
				session.cameraState.JumpQueue = session.cameraState.JumpQueue[1:]
				session.cameraState.JumpCurrentDwell = next.DwellSeconds
				session.cameraState.StartJumpTo(next.TargetIndex, next.TargetPos, next.ViewDist)
			}
		}

		if !mainWindowInputSuspended {
			// Mouse changes camera facing in jumping mode
			session.cameraState.Yaw -= float64(mouseDelta.X * a.runtime.MouseSensitivity)
			session.cameraState.Pitch -= float64(mouseDelta.Y * a.runtime.MouseSensitivity)

			// Clamp pitch
			if session.cameraState.Pitch > 1.5 {
				session.cameraState.Pitch = 1.5
			}
			if session.cameraState.Pitch < -1.5 {
				session.cameraState.Pitch = -1.5
			}

			session.cameraState.UpdateForwardFromAngles()
		}

		if !mainWindowInputSuspended {
			// Arrow keys move camera position in jumping mode
			if km.IsDown(input.ActionCameraPitchUp) {
				session.cameraState.Position.Y += arrowSpeed
			}
			if km.IsDown(input.ActionCameraPitchDown) {
				session.cameraState.Position.Y -= arrowSpeed
			}
			if km.IsDown(input.ActionCameraYawLeft) {
				session.cameraState.Position.X -= arrowSpeed
			}
			if km.IsDown(input.ActionCameraYawRight) {
				session.cameraState.Position.X += arrowSpeed
			}
		}

	case ui.CameraModeTracking:
		// Tick dwell countdown for multi-hop jump sequences.
		if session.cameraState.JumpDwellRemaining > 0 {
			session.cameraState.JumpDwellRemaining -= float64(dt)
			if session.cameraState.JumpDwellRemaining <= 0 && len(session.cameraState.JumpQueue) > 0 {
				next := session.cameraState.JumpQueue[0]
				session.cameraState.JumpQueue = session.cameraState.JumpQueue[1:]
				session.cameraState.JumpCurrentDwell = next.DwellSeconds
				session.cameraState.StartJumpTo(next.TargetIndex, next.TargetPos, next.ViewDist)
			}
		}

		// Tick orbit animation.
		if session.cameraState.OrbitSpeed != 0 && session.cameraState.OrbitRadiansRemaining > 0 {
			delta := session.cameraState.OrbitSpeed * float64(dt)
			session.cameraState.TrackYaw += delta
			session.cameraState.OrbitRadiansRemaining -= math.Abs(delta)
			if session.cameraState.OrbitRadiansRemaining <= 0 {
				session.cameraState.OrbitSpeed = 0
			}
		}

		// Keep automatic tracking updates active, but suspend user input while a dialog is open.
		if !mainWindowInputSuspended && (mouseDelta.X != 0 || mouseDelta.Y != 0) {
			session.cameraState.AdjustTrackAngles(
				-float64(mouseDelta.X*a.runtime.MouseSensitivity*0.5),
				float64(-mouseDelta.Y*a.runtime.MouseSensitivity*0.5),
			)
		}

		session.cameraState.UpdateTracking(state)

		// WASD controls for camera offset in tracking mode
		moveSpeed := a.runtime.CameraSpeed * dt // Same base speed as free-fly mode
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			moveSpeed *= 2.0 // Consistent 2x speed boost
		}

		// Get camera-relative directions
		right := session.cameraState.GetRight()

		if !mainWindowInputSuspended {
			if km.IsDown(input.ActionThrustForward) {
				// Move forward (closer to target)
				session.cameraState.TrackOffset = session.cameraState.TrackOffset.Add(session.cameraState.Forward.Scale(moveSpeed))
			}
			if km.IsDown(input.ActionThrustBackward) {
				// Move backward (away from target)
				session.cameraState.TrackOffset = session.cameraState.TrackOffset.Sub(session.cameraState.Forward.Scale(moveSpeed))
			}
			if km.IsDown(input.ActionThrustLeft) {
				// Pan left
				session.cameraState.TrackOffset = session.cameraState.TrackOffset.Sub(right.Scale(moveSpeed))
			}
			if km.IsDown(input.ActionThrustRight) {
				// Pan right
				session.cameraState.TrackOffset = session.cameraState.TrackOffset.Add(right.Scale(moveSpeed))
			}

			// Space for up (camera-relative) - DISABLED FOR TESTING
			// if rl.IsKeyDown(rl.KeySpace) {
			// 	session.cameraState.TrackOffset = session.cameraState.TrackOffset.Add(session.cameraState.Up.Scale(moveSpeed))
			// }

			// Arrow keys modify offset in tracking mode
			if km.IsDown(input.ActionCameraPitchUp) {
				session.cameraState.TrackOffset.Y += arrowSpeed
			}
			if km.IsDown(input.ActionCameraPitchDown) {
				session.cameraState.TrackOffset.Y -= arrowSpeed
			}
			if km.IsDown(input.ActionCameraYawLeft) {
				session.cameraState.TrackOffset.X -= arrowSpeed
			}
			if km.IsDown(input.ActionCameraYawRight) {
				session.cameraState.TrackOffset.X += arrowSpeed
			}

			// camera.reset: reset offset
			if km.IsPressed(input.ActionCameraReset) {
				session.cameraState.TrackOffset = engine.Vector3{X: 0, Y: 0, Z: 0}
			}
		}

	case ui.CameraModeFree:
		if !mainWindowInputSuspended {
			// Mouse look
			session.cameraState.Yaw -= float64(mouseDelta.X * a.runtime.MouseSensitivity)
			session.cameraState.Pitch -= float64(mouseDelta.Y * a.runtime.MouseSensitivity)

			// Clamp pitch
			if session.cameraState.Pitch > 1.5 {
				session.cameraState.Pitch = 1.5
			}
			if session.cameraState.Pitch < -1.5 {
				session.cameraState.Pitch = -1.5
			}

			// Update forward vector
			session.cameraState.UpdateForwardFromAngles()
		}

		// Mirror camera facing into ShipInstance.FacingVector every frame
		// (happens regardless of input-suspension state; spec §9.4).
		if session.ship != nil {
			fwd := session.cameraState.Forward
			session.ship.FacingVector = [3]float32{fwd.X, fwd.Y, fwd.Z}
		}

		right := session.cameraState.GetRight()

		if !mainWindowInputSuspended {
			if session.ship != nil {
				// --- Ship kinematics (F-022 §9, Phase 2 using F-033 ShipInstance) ---
				// Acceleration is in m/s²; convert to sim_units/s² for integration.
				accelSimUnits := session.ship.EffectiveAccelMaxMS2() / engine.MetersPerSimUnit

				// Build thrust intent from held keys.
				var ax, ay, az float64
				if km.IsDown(input.ActionThrustForward) {
					ax += float64(session.cameraState.Forward.X) * accelSimUnits
					ay += float64(session.cameraState.Forward.Y) * accelSimUnits
					az += float64(session.cameraState.Forward.Z) * accelSimUnits
				}
				if km.IsDown(input.ActionThrustBackward) {
					ax -= float64(session.cameraState.Forward.X) * accelSimUnits
					ay -= float64(session.cameraState.Forward.Y) * accelSimUnits
					az -= float64(session.cameraState.Forward.Z) * accelSimUnits
				}
				if km.IsDown(input.ActionThrustLeft) {
					ax -= float64(right.X) * accelSimUnits
					ay -= float64(right.Y) * accelSimUnits
					az -= float64(right.Z) * accelSimUnits
				}
				if km.IsDown(input.ActionThrustRight) {
					ax += float64(right.X) * accelSimUnits
					ay += float64(right.Y) * accelSimUnits
					az += float64(right.Z) * accelSimUnits
				}
				if km.IsDown(input.ActionThrustUp) {
					ax += float64(session.cameraState.Up.X) * accelSimUnits
					ay += float64(session.cameraState.Up.Y) * accelSimUnits
					az += float64(session.cameraState.Up.Z) * accelSimUnits
				}
				if km.IsDown(input.ActionThrustDown) {
					ax -= float64(session.cameraState.Up.X) * accelSimUnits
					ay -= float64(session.cameraState.Up.Y) * accelSimUnits
					az -= float64(session.cameraState.Up.Z) * accelSimUnits
				}

				// velocity += accel * dt
				dtF := float64(dt)
				session.ship.Velocity[0] += ax * dtF
				session.ship.Velocity[1] += ay * dtF
				session.ship.Velocity[2] += az * dtF

				// Clamp to max speed (hard cap from ShipDefinition).
				if maxSpeed := session.ship.Definition.MaxSpeedSimUnitsPerS; maxSpeed > 0 {
					speed := math.Sqrt(
						session.ship.Velocity[0]*session.ship.Velocity[0]+
							session.ship.Velocity[1]*session.ship.Velocity[1]+
							session.ship.Velocity[2]*session.ship.Velocity[2],
					)
					if speed > maxSpeed {
						scale := maxSpeed / speed
						session.ship.Velocity[0] *= scale
						session.ship.Velocity[1] *= scale
						session.ship.Velocity[2] *= scale
					}
				}

				// position += velocity * dt
				session.ship.Position[0] += session.ship.Velocity[0] * dtF
				session.ship.Position[1] += session.ship.Velocity[1] * dtF
				session.ship.Position[2] += session.ship.Velocity[2] * dtF

				// Arrow keys move ship position directly (no inertia; useful for
				// spectator repositioning without changing ship velocity state).
				if km.IsDown(input.ActionCameraPitchUp) {
					session.ship.Position[1] += float64(arrowSpeed)
				}
				if km.IsDown(input.ActionCameraPitchDown) {
					session.ship.Position[1] -= float64(arrowSpeed)
				}
				if km.IsDown(input.ActionCameraYawLeft) {
					session.ship.Position[0] -= float64(arrowSpeed)
				}
				if km.IsDown(input.ActionCameraYawRight) {
					session.ship.Position[0] += float64(arrowSpeed)
				}

				// Mirror ship world position to camera (camera IS the ship cockpit).
				session.cameraState.Position = engine.Vector3{
					X: float32(session.ship.Position[0]),
					Y: float32(session.ship.Position[1]),
					Z: float32(session.ship.Position[2]),
				}
			} else {
				// --- Legacy direct-position camera (no ship) ---
				moveSpeed := a.runtime.CameraSpeed * dt
				if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
					moveSpeed *= 2.0
				}
				if km.IsDown(input.ActionThrustForward) {
					session.cameraState.Position = session.cameraState.Position.Add(session.cameraState.Forward.Scale(moveSpeed))
				}
				if km.IsDown(input.ActionThrustBackward) {
					session.cameraState.Position = session.cameraState.Position.Sub(session.cameraState.Forward.Scale(moveSpeed))
				}
				if km.IsDown(input.ActionThrustLeft) {
					session.cameraState.Position = session.cameraState.Position.Sub(right.Scale(moveSpeed))
				}
				if km.IsDown(input.ActionThrustRight) {
					session.cameraState.Position = session.cameraState.Position.Add(right.Scale(moveSpeed))
				}
				if km.IsDown(input.ActionCameraPitchUp) {
					session.cameraState.Position.Y += arrowSpeed
				}
				if km.IsDown(input.ActionCameraPitchDown) {
					session.cameraState.Position.Y -= arrowSpeed
				}
				if km.IsDown(input.ActionCameraYawLeft) {
					session.cameraState.Position.X -= arrowSpeed
				}
				if km.IsDown(input.ActionCameraYawRight) {
					session.cameraState.Position.X += arrowSpeed
				}
			}
		}

		// Apply persistent velocity drift (set via gRPC NavigationService).
		// Only applies when there is no ShipInstance; ship kinematics own position otherwise.
		if session.ship == nil {
			if session.cameraState.Velocity.X != 0 || session.cameraState.Velocity.Y != 0 || session.cameraState.Velocity.Z != 0 {
				session.cameraState.Position = session.cameraState.Position.Add(session.cameraState.Velocity.Scale(dt))
			}
		}
	}

	return wheelMove // Return zoom indicator value
}

// isModifierKey reports whether key is a shift, ctrl, or alt key code.
func isModifierKey(key int32) bool {
	return key == int32(rl.KeyLeftShift) || key == int32(rl.KeyRightShift) ||
		key == int32(rl.KeyLeftControl) || key == int32(rl.KeyRightControl) ||
		key == int32(rl.KeyLeftAlt) || key == int32(rl.KeyRightAlt)
}

// liveModSet returns the currently-held modifier keys as an input.ModSet.
func liveModSet() input.ModSet {
	var mods input.ModSet
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		mods |= input.ModShift
	}
	if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
		mods |= input.ModCtrl
	}
	if rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt) {
		mods |= input.ModAlt
	}
	return mods
}

// anyModifierDown reports whether any shift/ctrl/alt key is currently held.
func anyModifierDown() bool {
	return rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) ||
		rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) ||
		rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
}

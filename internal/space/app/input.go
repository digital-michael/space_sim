package app

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/digital-michael/space_sim/internal/space"
	engine "github.com/digital-michael/space_sim/internal/space/engine"
	"github.com/digital-michael/space_sim/internal/space/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInput processes keyboard input for camera modes and object selection
func handleInput(app *App, sim *space.Simulation, cameraState *ui.CameraState, inputState *ui.InputState, state *engine.SimulationState, navigationOrder []engine.ObjectCategory, gridVisible bool, asteroidDataset engine.AsteroidDataset, hudVisible bool, helpVisible bool, mouseModeEnabled bool, labelsVisible bool, debugEnabled bool) (bool, bool, engine.AsteroidDataset, bool, bool, bool, bool) {
	mainWindowInputSuspended := inputState.MainWindowInputSuspended()

	// G: Toggle grid
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyG) {
		gridVisible = !gridVisible
	}

	// H: Toggle HUD
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyH) {
		hudVisible = !hudVisible
	}

	// L: Toggle object labels
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyL) {
		labelsVisible = !labelsVisible
	}

	// ?: Toggle help screen
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeySlash) && (rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)) {
		helpVisible = !helpVisible
	}

	// M key: Toggle mouse mode (camera control vs UI cursor)
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyM) {
		mouseModeEnabled = !mouseModeEnabled
		if mouseModeEnabled {
			rl.DisableCursor()
		} else {
			rl.EnableCursor()
		}
	}

	// Cmd+F key: Toggle fullscreen with proper display resolution handling
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyF) && (rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)) {
		app.toggleFullscreen()
	}

	// Cmd+Q key: Quit application
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyQ) && (rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)) {
		return true, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
	}

	// , and . keys: Decrease/increase time scale (simulation seconds per real second)
	// Guard: skip when Shift is held — SHIFT+, is the Anim Speed control (</>)
	shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyComma) && !shiftHeld {
		back := sim.GetState().GetBack()
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
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyPeriod) && !shiftHeld {
		back := sim.GetState().GetBack()
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

	// +/- keys: Increase/decrease asteroid dataset
	if !mainWindowInputSuspended && (rl.IsKeyPressed(rl.KeyEqual) || rl.IsKeyPressed(rl.KeyKpAdd)) { // + key (Shift+= or numpad +)
		if asteroidDataset < 3 {
			asteroidDataset++
			sim.SetAsteroidDataset(asteroidDataset)
		}
	}
	if !mainWindowInputSuspended && (rl.IsKeyPressed(rl.KeyMinus) || rl.IsKeyPressed(rl.KeyKpSubtract)) { // - key (or numpad -)
		if asteroidDataset > 0 {
			asteroidDataset--
			sim.SetAsteroidDataset(asteroidDataset)
		}
	}

	// < and > keys: Decrease/increase animation speed (Shift + , and Shift + .)
	// Controls the physics tick rate — how many sim ticks fire per real second (0%–100%)
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyComma) && shiftHeld {
		// Decrease anim speed: < key
		currentSpeed := sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := len(speedSteps) - 1; i >= 0; i-- {
			if currentSpeed > speedSteps[i] {
				sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyPeriod) && shiftHeld {
		// Increase anim speed: > key
		currentSpeed := sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := 0; i < len(speedSteps); i++ {
			if currentSpeed < speedSteps[i] {
				sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}

	// P: Performance options
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyP) {
		inputState.StartSelection(ui.SelectionModePerformance)
	}

	// C: Center view - behavior depends on mode
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyC) {
		if cameraState.Mode == ui.CameraModeFree {
			// Free-fly mode: center camera view on origin (sun)
			toOrigin := engine.Vector3{X: 0, Y: 0, Z: 0}.Sub(cameraState.Position)
			if toOrigin.Length() > 0.1 {
				cameraState.Forward = toOrigin.Normalize()
				// Update yaw and pitch from forward vector
				cameraState.Yaw = math.Atan2(float64(cameraState.Forward.X), float64(cameraState.Forward.Z))
				cameraState.Pitch = math.Asin(float64(cameraState.Forward.Y))
			}
		} else if cameraState.Mode == ui.CameraModeTracking {
			// Tracking mode: reset zoom to 24% auto-zoom distance
			if cameraState.TrackTargetIndex >= 0 && cameraState.TrackTargetIndex < len(state.Objects) {
				targetObj := state.Objects[cameraState.TrackTargetIndex]
				cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.24)
			}
		}
	}

	// F key: Drill down to closest child (Forward in hierarchy)
	// B key: Move up to parent (Back in hierarchy)
	if cameraState.Mode == ui.CameraModeTracking && !mainWindowInputSuspended {
		if rl.IsKeyPressed(rl.KeyF) {
			if cameraState.TrackTargetIndex >= 0 && cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[cameraState.TrackTargetIndex]

				// F: Drill down to closest child
				if debugEnabled {
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

				if debugEnabled {
					fmt.Printf("[DEBUG] Found %d children\n", len(children))
				}
				// Sort children by distance (closest first)
				if len(children) > 0 {
					sort.Slice(children, func(i, j int) bool {
						return children[i].distance < children[j].distance
					})

					// Track the closest child
					closestChild := state.Objects[children[0].index]
					if debugEnabled {
						fmt.Printf("[DEBUG] Tracking closest child: %s\n", closestChild.Meta.Name)
					}
					cameraState.StartTracking(children[0].index)
					cameraState.TrackDistance = ui.CalculateAutoZoomDistance(closestChild.Meta.PhysicalRadius, 0.24)
				}
			}
		}

		if rl.IsKeyPressed(rl.KeyB) {
			if cameraState.TrackTargetIndex >= 0 && cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[cameraState.TrackTargetIndex]

				// B: Move up to parent
				if debugEnabled {
					fmt.Printf("[DEBUG] B key: Moving to parent of %s\n", currentObj.Meta.Name)
				}
				if currentObj.Meta.ParentName != "" {
					// Find parent object by name
					for i, obj := range state.Objects {
						if obj.Meta.Name == currentObj.Meta.ParentName {
							if debugEnabled {
								fmt.Printf("[DEBUG] Found parent: %s\n", obj.Meta.Name)
							}
							cameraState.StartTracking(i)
							cameraState.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
							break
						}
					}
				} else if currentObj.Meta.SemiMajorAxis > 0 || currentObj.Meta.OrbitRadius > 0 {
					// No explicit parent but is orbiting - find the star
					if debugEnabled {
						fmt.Printf("[DEBUG] No explicit parent, looking for central star\n")
					}
					for i, obj := range state.Objects {
						if obj.Meta.Category == engine.CategoryStar {
							if debugEnabled {
								fmt.Printf("[DEBUG] Found central star: %s\n", obj.Meta.Name)
							}
							cameraState.StartTracking(i)
							cameraState.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
							break
						}
					}
				} else {
					if debugEnabled {
						fmt.Printf("[DEBUG] No parent for %s (already at star)\n", currentObj.Meta.Name)
					}
				}
			}
		}
	}

	// TAB / Shift+TAB: Cycle through siblings when tracking (objects with same parent and category)
	if cameraState.Mode == ui.CameraModeTracking && !mainWindowInputSuspended {
		isShiftPressed := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		if rl.IsKeyPressed(rl.KeyTab) {
			if debugEnabled {
				fmt.Printf("[DEBUG] TAB pressed - Shift: %v\n", isShiftPressed)
			}

			if cameraState.TrackTargetIndex >= 0 && cameraState.TrackTargetIndex < len(state.Objects) {
				currentObj := state.Objects[cameraState.TrackTargetIndex]

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
						if idx == cameraState.TrackTargetIndex {
							currentPos = i
							break
						}
					}

					if currentPos >= 0 {
						var nextPos int
						if isShiftPressed {
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
						cameraState.StartTracking(nextIndex)
						cameraState.TrackDistance = ui.CalculateAutoZoomDistance(nextObj.Meta.PhysicalRadius, 0.24)
					}
				}
			}
		}
	}

	// ESC: Cancel selection, exit tracking, or exit mouse mode (priority order)
	if rl.IsKeyPressed(rl.KeyEscape) {
		if inputState.SelectionActive {
			inputState.CancelSelection()
			return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
		} else if cameraState.Mode == ui.CameraModeTracking {
			cameraState.StopTracking()
			return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
		} else if mouseModeEnabled {
			// Exit mouse mode, enable cursor
			mouseModeEnabled = false
			rl.EnableCursor()
			return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
		}
	}

	// Object selection mode
	if inputState.SelectionActive {
		// Performance options mode
		if inputState.SelectionMode == ui.SelectionModePerformance {
			// Left/Right to switch tabs (except when adjusting Importance Threshold)
			isAdjustingThreshold := inputState.PerformanceTab == 1 && inputState.SelectedIndex == 0
			if rl.IsKeyPressed(rl.KeyLeft) && !isAdjustingThreshold {
				if inputState.PerformanceTab == 1 {
					inputState.PerformanceTab = 0
					inputState.SelectedIndex = 0
				}
			}
			if rl.IsKeyPressed(rl.KeyRight) && !isAdjustingThreshold {
				if inputState.PerformanceTab == 0 {
					inputState.PerformanceTab = 1
					inputState.SelectedIndex = 0
				}
			}

			// Up/Down to navigate options
			if rl.IsKeyPressed(rl.KeyUp) {
				if inputState.SelectedIndex > 0 {
					inputState.SelectedIndex--
				}
			}
			if rl.IsKeyPressed(rl.KeyDown) {
				maxIndex := 4 // Performance tab has 5 options (0-4)
				if inputState.PerformanceTab == 1 {
					maxIndex = 1 // Configuration tab has 2 options (0-1)
				}
				if inputState.SelectedIndex < maxIndex {
					inputState.SelectedIndex++
				}
			}

			// Space to toggle selected option
			if rl.IsKeyPressed(rl.KeySpace) {
				if inputState.PerformanceTab == 0 {
					// Performance tab toggles
					switch inputState.SelectedIndex {
					case 0:
						inputState.PerfOptions.FrustumCulling = !inputState.PerfOptions.FrustumCulling
					case 1:
						inputState.PerfOptions.LODEnabled = !inputState.PerfOptions.LODEnabled
					case 2:
						inputState.PerfOptions.InstancedRendering = !inputState.PerfOptions.InstancedRendering
					case 3:
						inputState.PerfOptions.SpatialPartition = !inputState.PerfOptions.SpatialPartition
					case 4:
						inputState.PerfOptions.PointRendering = !inputState.PerfOptions.PointRendering
					}
				} else {
					// Configuration tab toggles
					switch inputState.SelectedIndex {
					case 1:
						inputState.PerfOptions.UseInPlaceSwap = !inputState.PerfOptions.UseInPlaceSwap
						// Apply the change to simulation
						if inputState.PerfOptions.UseInPlaceSwap {
							sim.GetState().EnableInPlaceSwap()
							fmt.Println("✓ Enabled in-place swap (zero-allocation mode)")
						} else {
							sim.GetState().DisableInPlaceSwap()
							fmt.Println("✓ Disabled in-place swap (dynamic allocation mode)")
						}
					}
				}
			}

			// Left/Right to adjust importance threshold (only on Configuration tab, option 0)
			if inputState.PerformanceTab == 1 && inputState.SelectedIndex == 0 {
				if rl.IsKeyPressed(rl.KeyLeft) {
					// Cycle through: 0, 5, 10, 15, 30, 40, 50, 60, 70, 80, 90
					thresholds := []int{0, 5, 10, 15, 30, 40, 50, 60, 70, 80, 90}
					current := inputState.PerfOptions.ImportanceThreshold
					for i := len(thresholds) - 1; i >= 0; i-- {
						if current > thresholds[i] {
							inputState.PerfOptions.ImportanceThreshold = thresholds[i]
							break
						}
					}
					if current <= thresholds[0] {
						inputState.PerfOptions.ImportanceThreshold = thresholds[len(thresholds)-1]
					}
				}
				if rl.IsKeyPressed(rl.KeyRight) {
					// Cycle through: 0, 5, 10, 15, 30, 40, 50, 60, 70, 80, 90
					thresholds := []int{0, 5, 10, 15, 30, 40, 50, 60, 70, 80, 90}
					current := inputState.PerfOptions.ImportanceThreshold
					for i := 0; i < len(thresholds); i++ {
						if current < thresholds[i] {
							inputState.PerfOptions.ImportanceThreshold = thresholds[i]
							break
						}
					}
					if current >= thresholds[len(thresholds)-1] {
						inputState.PerfOptions.ImportanceThreshold = thresholds[0]
					}
				}
			}
			return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
		}

		// Text input for filtering (for Jump/Track modes only, not Performance)
		if inputState.SelectionMode != ui.SelectionModePerformance {
			// Capture character input
			char := rl.GetCharPressed()
			for char > 0 {
				// Add printable characters to filter text
				if char >= 32 && char <= 126 {
					inputState.FilterText += string(rune(char))
					// Rebuild filtered list with new filter
					inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
					inputState.SelectedIndex = 0
					inputState.ScrollOffset = 0 // Reset scroll when filtering
				}
				char = rl.GetCharPressed()
			}

			// Backspace to delete characters
			if rl.IsKeyPressed(rl.KeyBackspace) && len(inputState.FilterText) > 0 {
				inputState.FilterText = inputState.FilterText[:len(inputState.FilterText)-1]
				// Rebuild filtered list
				inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
				if inputState.SelectedIndex >= len(inputState.FilteredIndices) {
					inputState.SelectedIndex = 0
				}
				inputState.ScrollOffset = 0 // Reset scroll when filtering
			}
		}

		// Left/Right arrow keys for category cycling
		if rl.IsKeyPressed(rl.KeyLeft) {
			inputState.CycleCategoryBack(navigationOrder)
			inputState.FilterText = ""  // Clear filter when changing category
			inputState.ScrollOffset = 0 // Reset scroll position
			// Rebuild filtered list
			inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
			if inputState.SelectedIndex >= len(inputState.FilteredIndices) {
				inputState.SelectedIndex = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			inputState.CycleCategory(navigationOrder)
			inputState.FilterText = ""  // Clear filter when changing category
			inputState.ScrollOffset = 0 // Reset scroll position
			// Rebuild filtered list
			inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
			if inputState.SelectedIndex >= len(inputState.FilteredIndices) {
				inputState.SelectedIndex = 0
			}
		}
		// Up/Down arrow keys for selection within category
		if rl.IsKeyPressed(rl.KeyUp) {
			inputState.SelectPrevious()
			// Auto-scroll to keep selection visible
			if inputState.SelectedIndex < inputState.ScrollOffset {
				inputState.ScrollOffset = inputState.SelectedIndex
			}
		}
		if rl.IsKeyPressed(rl.KeyDown) {
			inputState.SelectNext(len(inputState.FilteredIndices) - 1)
			// Auto-scroll to keep selection visible
			// Calculate visible items (must match drawSelectionUI calculation)
			bgHeight := int32(500)
			tabHeight := int32(30)
			filterBoxHeight := int32(30) // 25 + 5 padding
			startY := int32(75) + filterBoxHeight + tabHeight + 10
			lineHeight := int32(30)
			listAreaHeight := bgHeight - startY - 10
			visibleItems := int(listAreaHeight / lineHeight)
			if inputState.SelectedIndex >= inputState.ScrollOffset+visibleItems {
				inputState.ScrollOffset = inputState.SelectedIndex - visibleItems + 1
			}
		}
		// Page Up/Down for faster navigation
		if rl.IsKeyPressed(rl.KeyPageUp) {
			bgHeight := int32(500)
			tabHeight := int32(30)
			filterBoxHeight := int32(30)
			startY := int32(75) + filterBoxHeight + tabHeight + 10
			lineHeight := int32(30)
			listAreaHeight := bgHeight - startY - 10
			visibleItems := int(listAreaHeight / lineHeight)

			inputState.SelectedIndex -= visibleItems
			if inputState.SelectedIndex < 0 {
				inputState.SelectedIndex = 0
			}
			inputState.ScrollOffset = inputState.SelectedIndex
		}
		if rl.IsKeyPressed(rl.KeyPageDown) {
			bgHeight := int32(500)
			tabHeight := int32(30)
			filterBoxHeight := int32(30)
			startY := int32(75) + filterBoxHeight + tabHeight + 10
			lineHeight := int32(30)
			listAreaHeight := bgHeight - startY - 10
			visibleItems := int(listAreaHeight / lineHeight)
			maxIndex := len(inputState.FilteredIndices) - 1

			inputState.SelectedIndex += visibleItems
			if inputState.SelectedIndex > maxIndex {
				inputState.SelectedIndex = maxIndex
			}
			// Auto-scroll to keep selection visible
			if inputState.SelectedIndex >= inputState.ScrollOffset+visibleItems {
				inputState.ScrollOffset = inputState.SelectedIndex - visibleItems + 1
			}
		}
		// Home/End for jumping to start/end
		if rl.IsKeyPressed(rl.KeyHome) {
			inputState.SelectedIndex = 0
			inputState.ScrollOffset = 0
		}
		if rl.IsKeyPressed(rl.KeyEnd) {
			maxIndex := len(inputState.FilteredIndices) - 1
			inputState.SelectedIndex = maxIndex
			// Scroll to show last item
			bgHeight := int32(500)
			tabHeight := int32(30)
			filterBoxHeight := int32(30)
			startY := int32(75) + filterBoxHeight + tabHeight + 10
			lineHeight := int32(30)
			listAreaHeight := bgHeight - startY - 10
			visibleItems := int(listAreaHeight / lineHeight)
			inputState.ScrollOffset = maxIndex - visibleItems + 1
			if inputState.ScrollOffset < 0 {
				inputState.ScrollOffset = 0
			}
		}
		// Enter to confirm
		if rl.IsKeyPressed(rl.KeyEnter) {
			selectedIndex, mode := inputState.ConfirmSelection()
			// Map from filtered index to actual object index
			if selectedIndex >= 0 && selectedIndex < len(inputState.FilteredIndices) {
				actualIndex := inputState.FilteredIndices[selectedIndex]

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
						return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
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
						return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
					}
				}

				targetObj := state.Objects[actualIndex]
				if mode == ui.SelectionModeJump {
					// Jump to object with good viewing distance (5x radius)
					cameraState.StartJumpTo(actualIndex, targetObj.Anim.Position, float64(targetObj.Meta.PhysicalRadius)*5.0)
				} else if mode == ui.SelectionModeTrack {
					// Start tracking object with auto-zoom (24% of screen height)
					cameraState.StartTracking(actualIndex)
					cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.24)
				} else if mode == ui.SelectionModeTrackEquatorial {
					// Start tracking from surface view - closer zoom (40% of screen height)
					cameraState.StartTrackingEquatorial(actualIndex)
					cameraState.TrackDistance = ui.CalculateAutoZoomDistance(targetObj.Meta.PhysicalRadius, 0.40)
				}
			}
		}
		return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible // Don't process other keys during selection
	}

	// J: Jump to object (free-fly mode only)
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyJ) && cameraState.Mode == ui.CameraModeFree {
		inputState.StartSelection(ui.SelectionModeJump)
		inputState.FilterText = ""
		inputState.ScrollOffset = 0
		inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
	}

	// T/t: Track object (free-fly mode or tracking mode to switch target)
	// Shift+T: Track from above with comfortable distance
	// t (lowercase): Track from equatorial plane
	if !mainWindowInputSuspended && rl.IsKeyPressed(rl.KeyT) && (cameraState.Mode == ui.CameraModeFree || cameraState.Mode == ui.CameraModeTracking) {
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			// Uppercase T - track from above
			inputState.StartSelection(ui.SelectionModeTrack)
			inputState.FilterText = ""
			inputState.ScrollOffset = 0
			inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
		} else {
			// Lowercase t - track from equator
			inputState.StartSelection(ui.SelectionModeTrackEquatorial)
			inputState.FilterText = ""
			inputState.ScrollOffset = 0
			inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, inputState.SelectedCategory, inputState.FilterText)
		}
	}

	return false, gridVisible, asteroidDataset, hudVisible, helpVisible, mouseModeEnabled, labelsVisible
}

// updateCameraState updates camera position and orientation based on mode
func updateCameraState(cameraState *ui.CameraState, inputState *ui.InputState, state *engine.SimulationState, dt, speed, sensitivity float32, mouseModeEnabled bool) float32 {
	mainWindowInputSuspended := inputState.MainWindowInputSuspended()

	// Mouse look (only active when mouse mode is enabled)
	var mouseDelta rl.Vector2
	if mouseModeEnabled && !mainWindowInputSuspended {
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
		zoomSpeed = wheelMove * speed * 0.5 // Reduced from 2.0 to 0.5 (1/4 speed)

		switch cameraState.Mode {
		case ui.CameraModeTracking:
			// In tracking mode, adjust distance from target
			cameraState.TrackDistance -= float64(zoomSpeed * 10.0)
			// Clamp to reasonable values
			if cameraState.TrackDistance < engine.CameraTrackDistMin {
				cameraState.TrackDistance = engine.CameraTrackDistMin
			}
			if cameraState.TrackDistance > engine.CameraTrackDistMax {
				cameraState.TrackDistance = engine.CameraTrackDistMax
			}

		case ui.CameraModeFree, ui.CameraModeJumping:
			// In free/jumping mode, move camera along forward direction
			cameraState.Position = cameraState.Position.Add(cameraState.Forward.Scale(zoomSpeed * 10.0))
		}
	}

	// Arrow keys for movement in the system plane (active in all modes)
	arrowSpeed := speed * dt // Same base speed as WASD
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		arrowSpeed *= 2.0 // Consistent 2x speed boost
	}

	// Update based on camera mode
	switch cameraState.Mode {
	case ui.CameraModeJumping:
		cameraState.UpdateJump(float64(dt))

		if !mainWindowInputSuspended {
			// Mouse changes camera facing in jumping mode
			cameraState.Yaw -= float64(mouseDelta.X * sensitivity)
			cameraState.Pitch -= float64(mouseDelta.Y * sensitivity)

			// Clamp pitch
			if cameraState.Pitch > 1.5 {
				cameraState.Pitch = 1.5
			}
			if cameraState.Pitch < -1.5 {
				cameraState.Pitch = -1.5
			}

			cameraState.UpdateForwardFromAngles()
		}

		if !mainWindowInputSuspended {
			// Arrow keys move camera position in jumping mode
			if rl.IsKeyDown(rl.KeyUp) {
				cameraState.Position.Y += arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyDown) {
				cameraState.Position.Y -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyLeft) {
				cameraState.Position.X -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyRight) {
				cameraState.Position.X += arrowSpeed
			}
		}

	case ui.CameraModeTracking:
		// Keep automatic tracking updates active, but suspend user input while a dialog is open.
		if !mainWindowInputSuspended && (mouseDelta.X != 0 || mouseDelta.Y != 0) {
			cameraState.AdjustTrackAngles(
				-float64(mouseDelta.X*sensitivity*0.5),
				float64(-mouseDelta.Y*sensitivity*0.5),
			)
		}

		cameraState.UpdateTracking(state)

		// WASD controls for camera offset in tracking mode
		moveSpeed := speed * dt // Same base speed as free-fly mode
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			moveSpeed *= 2.0 // Consistent 2x speed boost
		}

		// Get camera-relative directions
		right := cameraState.GetRight()

		if !mainWindowInputSuspended {
			if rl.IsKeyDown(rl.KeyW) {
				// Move forward (closer to target)
				cameraState.TrackOffset = cameraState.TrackOffset.Add(cameraState.Forward.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyS) {
				// Move backward (away from target)
				cameraState.TrackOffset = cameraState.TrackOffset.Sub(cameraState.Forward.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyA) {
				// Pan left
				cameraState.TrackOffset = cameraState.TrackOffset.Sub(right.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyD) {
				// Pan right
				cameraState.TrackOffset = cameraState.TrackOffset.Add(right.Scale(moveSpeed))
			}

			// Space for up (camera-relative) - DISABLED FOR TESTING
			// if rl.IsKeyDown(rl.KeySpace) {
			// 	cameraState.TrackOffset = cameraState.TrackOffset.Add(cameraState.Up.Scale(moveSpeed))
			// }

			// Arrow keys modify offset in tracking mode
			if rl.IsKeyDown(rl.KeyUp) {
				cameraState.TrackOffset.Y += arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyDown) {
				cameraState.TrackOffset.Y -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyLeft) {
				cameraState.TrackOffset.X -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyRight) {
				cameraState.TrackOffset.X += arrowSpeed
			}

			// R key to reset offset
			if rl.IsKeyPressed(rl.KeyR) {
				cameraState.TrackOffset = engine.Vector3{X: 0, Y: 0, Z: 0}
			}
		}

	case ui.CameraModeFree:
		if !mainWindowInputSuspended {
			// Mouse look
			cameraState.Yaw -= float64(mouseDelta.X * sensitivity)
			cameraState.Pitch -= float64(mouseDelta.Y * sensitivity)

			// Clamp pitch
			if cameraState.Pitch > 1.5 {
				cameraState.Pitch = 1.5
			}
			if cameraState.Pitch < -1.5 {
				cameraState.Pitch = -1.5
			}

			// Update forward vector
			cameraState.UpdateForwardFromAngles()
		}

		// WASD movement with Shift for 2x speed
		moveSpeed := speed * dt
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			moveSpeed *= 2.0 // Consistent 2x speed boost
		}
		right := cameraState.GetRight()

		if !mainWindowInputSuspended {
			if rl.IsKeyDown(rl.KeyW) {
				cameraState.Position = cameraState.Position.Add(cameraState.Forward.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyS) {
				cameraState.Position = cameraState.Position.Sub(cameraState.Forward.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyA) {
				cameraState.Position = cameraState.Position.Sub(right.Scale(moveSpeed))
			}
			if rl.IsKeyDown(rl.KeyD) {
				cameraState.Position = cameraState.Position.Add(right.Scale(moveSpeed))
			}

			// Space for up, Ctrl for down (Shift used for speed) - DISABLED FOR TESTING
			// if rl.IsKeyDown(rl.KeySpace) {
			// 	cameraState.Position.Y += moveSpeed
			// }
			// if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
			// 	cameraState.Position.Y -= moveSpeed
			// }

			// Arrow keys move camera position in free-fly mode
			if rl.IsKeyDown(rl.KeyUp) {
				cameraState.Position.Y += arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyDown) {
				cameraState.Position.Y -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyLeft) {
				cameraState.Position.X -= arrowSpeed
			}
			if rl.IsKeyDown(rl.KeyRight) {
				cameraState.Position.X += arrowSpeed
			}
		}
	}

	return wheelMove // Return zoom indicator value
}

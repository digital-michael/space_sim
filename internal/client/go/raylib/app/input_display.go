package app

import (
	"fmt"
	"math"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	render "github.com/digital-michael/space_sim/internal/client/go/raylib/ui/render"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInputDisplay processes display/HUD and sim-control keybindings.
// Returns true if the application should quit.
func (a *App) handleInputDisplay(session *runtimeSession, km *input.KeyMap, suspended bool) bool {
	if !suspended && km.IsPressed(input.ActionUISystemSelector) {
		a.openSystemSelector(session.inputState)
		return false
	}
	if !suspended && km.IsPressed(input.ActionUIScriptBrowser) {
		a.openScriptSelector(session.inputState)
		return false
	}
	if !suspended && km.IsPressed(input.ActionUILabelCycle) {
		switch a.runtime.LabelMode {
		case ui.LabelModeOff:
			a.runtime.LabelMode = ui.LabelModeOn
		case ui.LabelModeOn:
			a.runtime.LabelMode = ui.LabelModeNearest
		default:
			a.runtime.LabelMode = ui.LabelModeOff
		}
	}
	if !suspended && km.IsPressed(input.ActionUIInfraCycle) {
		a.runtime.InfraMode = (a.runtime.InfraMode + 1) % 3
	}
	if !suspended && km.IsPressed(input.ActionUIDebugLabels) {
		a.runtime.PendingLabelDebugDump = true
	}
	if !suspended && km.IsPressed(input.ActionUIMouseModeToggle) {
		a.runtime.MouseModeEnabled = !a.runtime.MouseModeEnabled
		if a.runtime.MouseModeEnabled {
			disableCursor()
		} else {
			rl.EnableCursor()
		}
	}
	if !suspended && km.IsPressed(input.ActionUIQuit) {
		return true
	}

	// sim.timescale_decrease: step time rate down.
	if !suspended && km.IsPressed(input.ActionSimTimescaleDecrease) && session.sim != nil {
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
		if currentIdx > 0 {
			back.SecondsPerSecond = timeRates[currentIdx-1]
			fmt.Printf("Time rate: %s\n", timeLabels[currentIdx-1])
		} else if currentIdx == -1 && back.SecondsPerSecond > 0 {
			for i := len(timeRates) - 1; i >= 0; i-- {
				if timeRates[i] < back.SecondsPerSecond {
					back.SecondsPerSecond = timeRates[i]
					fmt.Printf("Time rate: %s\n", timeLabels[i])
					break
				}
			}
		}
	}
	// sim.timescale_increase: step time rate up.
	if !suspended && km.IsPressed(input.ActionSimTimescaleIncrease) && session.sim != nil {
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
			for i := 0; i < len(timeRates); i++ {
				if timeRates[i] > back.SecondsPerSecond {
					back.SecondsPerSecond = timeRates[i]
					fmt.Printf("Time rate: %s\n", timeLabels[i])
					break
				}
			}
		}
	}

	// sim.dataset_increase / sim.dataset_decrease: asteroid dataset size.
	if !suspended && km.IsPressed(input.ActionSimDatasetIncrease) && session.sim != nil {
		if a.runtime.AsteroidDataset < 3 {
			a.runtime.AsteroidDataset++
			session.sim.SetAsteroidDataset(a.runtime.AsteroidDataset)
		}
	}
	if !suspended && km.IsPressed(input.ActionSimDatasetDecrease) && session.sim != nil {
		if a.runtime.AsteroidDataset > 0 {
			a.runtime.AsteroidDataset--
			session.sim.SetAsteroidDataset(a.runtime.AsteroidDataset)
		}
	}

	// Bare - / = (no modifiers): step UI scale.
	if !suspended {
		noMods := !rl.IsKeyDown(rl.KeyLeftAlt) && !rl.IsKeyDown(rl.KeyRightAlt) &&
			!rl.IsKeyDown(rl.KeyLeftControl) && !rl.IsKeyDown(rl.KeyRightControl) &&
			!rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) &&
			!rl.IsKeyDown(rl.KeyLeftSuper) && !rl.IsKeyDown(rl.KeyRightSuper)
		if noMods && (rl.IsKeyPressed(rl.KeyMinus) || rl.IsKeyPressed(rl.KeyEqual)) {
			scales := []float32{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}
			cur := a.runtime.UIScale
			if cur <= 0 {
				cur = 1.0
			}
			idx := 2
			for i, s := range scales {
				if s >= cur {
					idx = i
					break
				}
			}
			if rl.IsKeyPressed(rl.KeyMinus) && idx > 0 {
				idx--
			} else if rl.IsKeyPressed(rl.KeyEqual) && idx < len(scales)-1 {
				idx++
			}
			a.runtime.UIScale = scales[idx]
			a.runtime.Settings.UIScale = scales[idx]
			render.SetUIScale(scales[idx])
		}
	}

	// sim.tick_speed_decrease / sim.tick_speed_increase: physics tick rate.
	if !suspended && km.IsPressed(input.ActionSimTickSpeedDecrease) && session.sim != nil {
		currentSpeed := session.sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := len(speedSteps) - 1; i >= 0; i-- {
			if currentSpeed > speedSteps[i] {
				session.sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}
	if !suspended && km.IsPressed(input.ActionSimTickSpeedIncrease) && session.sim != nil {
		currentSpeed := session.sim.GetSpeed()
		speedSteps := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 1.0}
		for i := 0; i < len(speedSteps); i++ {
			if currentSpeed < speedSteps[i] {
				session.sim.SetSpeed(speedSteps[i])
				break
			}
		}
	}

	// ui.record_toggle / ui.record_pause: recording controls.
	if !suspended && km.IsPressed(input.ActionUIRecordToggle) {
		if a.runtime.RecordingActive {
			a.stopRecording()
		} else {
			a.startRecording("")
		}
	}
	if !suspended && km.IsPressed(input.ActionUIRecordPause) {
		if a.runtime.RecordingActive {
			a.runtime.RecordingPaused = !a.runtime.RecordingPaused
			if a.runtime.RecordingPaused {
				fmt.Println("[REC] Paused")
			} else {
				fmt.Println("[REC] Resumed")
			}
		}
	}

	// hud.toggle / hud.client_list / hud.info: HUD visibility.
	if !suspended && km.IsPressed(input.ActionHUDToggle) {
		a.runtime.HUDVisible = !a.runtime.HUDVisible
	}
	if !suspended && km.IsPressed(input.ActionHUDClientList) {
		a.runtime.HUD.Player = !a.runtime.HUD.Player
	}
	if !suspended && km.IsPressed(input.ActionHUDInfo) {
		a.runtime.HUD.Info = !a.runtime.HUD.Info
	}

	// sim.pause_toggle: toggle simulation time flow.
	if !suspended && km.IsPressed(input.ActionSimPauseToggle) && session.sim != nil {
		back := session.sim.GetState().GetBack()
		if back.SecondsPerSecond == 0 {
			if a.runtime.PauseRestoreRate == 0 {
				a.runtime.PauseRestoreRate = 1.0
			}
			back.SecondsPerSecond = a.runtime.PauseRestoreRate
			fmt.Println("Simulation resumed")
		} else {
			a.runtime.PauseRestoreRate = back.SecondsPerSecond
			back.SecondsPerSecond = 0
			fmt.Println("Simulation paused")
		}
	}

	return false
}

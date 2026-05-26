package app

import (
	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInput processes keyboard input for the current frame and returns true
// if the application should quit.
func (a *App) handleInput(session *runtimeSession, state *engine.SimulationState) bool {
	km := a.keyMap.Load()
	km.DrainQueue()

	altFPressed := (rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)) && rl.IsKeyPressed(rl.KeyF)
	if km.IsPressed(input.ActionUIFullscreen) || altFPressed {
		a.toggleFullscreen()
	}
	if km.IsPressed(input.ActionUISettings) {
		a.runtime.SettingsVisible = !a.runtime.SettingsVisible
		if a.runtime.SettingsVisible && a.runtime.MouseModeEnabled {
			a.runtime.MouseModeEnabled = false
			rl.EnableCursor()
		}
	}

	mainWindowInputSuspended := session.inputState.MainWindowInputSuspended() || a.runtime.SettingsVisible

	if a.runtime.SettingsVisible {
		shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
		return a.handleInputSettings(session, km, shiftHeld)
	}
	if session.inputState.SelectionActive {
		return a.handleInputSelection(session, state, km)
	}
	if km.IsPressed(input.ActionSimTrackStop) {
		return a.handleInputTrackStop(session)
	}
	if a.handleInputDisplay(session, km, mainWindowInputSuspended) {
		return true
	}
	a.handleInputNav(session, state, km, mainWindowInputSuspended)
	a.handleInputCamera(session, state, km, mainWindowInputSuspended)
	return false
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

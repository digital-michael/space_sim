// Package input provides a hardware-profile-aware key-binding system for the
// Space Sim Raylib client. Actions are named string constants that map to
// physical keys; the mapping is loaded from stock profiles (data/profiles/)
// and overridden by the optional user config (configs/keybindings.json).
package input

// InputAction identifies a user-bindable gameplay action.
// Values are stable — never reorder or reuse a retired constant.
type InputAction uint16

// Action constants. The numeric values are part of the stability contract.
const (
	ActionNone InputAction = 0

	// Camera control
	ActionCameraPitchUp       InputAction = 1
	ActionCameraPitchDown     InputAction = 2
	ActionCameraYawLeft       InputAction = 3
	ActionCameraYawRight      InputAction = 4
	ActionCameraRollLeft      InputAction = 5
	ActionCameraRollRight     InputAction = 6
	ActionCameraZoomIn        InputAction = 7
	ActionCameraZoomOut       InputAction = 8
	ActionCameraReset         InputAction = 9
	ActionCameraToggleFreeFly InputAction = 10

	// Movement / locomotion (F-022)
	ActionThrustForward  InputAction = 11
	ActionThrustBackward InputAction = 12
	ActionThrustLeft     InputAction = 13
	ActionThrustRight    InputAction = 14
	ActionThrustUp       InputAction = 15
	ActionThrustDown     InputAction = 16
	ActionBrake          InputAction = 17
	ActionDriftToggle    InputAction = 18

	// Simulation control
	ActionSimSpeedIncrease InputAction = 19
	ActionSimSpeedDecrease InputAction = 20
	ActionSimPauseToggle   InputAction = 21
	ActionSimTrackNext     InputAction = 22
	ActionSimTrackStop     InputAction = 23

	// UI and HUD
	ActionHUDToggle     InputAction = 24
	ActionHUDClientList InputAction = 25
	ActionUISettings    InputAction = 26
	ActionUIFullscreen  InputAction = 27

	// REPL overlay (world-context open/close; history nav is REPL-context-only)
	ActionReplOpen        InputAction = 28
	ActionReplClose       InputAction = 29
	ActionReplHistoryPrev InputAction = 30
	ActionReplHistoryNext InputAction = 31

	// sentinel — must equal last constant + 1
	actionCount InputAction = 32
)

// actionNames maps each action to its dot-notation vocabulary name.
var actionNames = map[InputAction]string{
	ActionCameraPitchUp:       "camera.pitch_up",
	ActionCameraPitchDown:     "camera.pitch_down",
	ActionCameraYawLeft:       "camera.yaw_left",
	ActionCameraYawRight:      "camera.yaw_right",
	ActionCameraRollLeft:      "camera.roll_left",
	ActionCameraRollRight:     "camera.roll_right",
	ActionCameraZoomIn:        "camera.zoom_in",
	ActionCameraZoomOut:       "camera.zoom_out",
	ActionCameraReset:         "camera.reset",
	ActionCameraToggleFreeFly: "camera.toggle_free_fly",
	ActionThrustForward:       "move.thrust_forward",
	ActionThrustBackward:      "move.thrust_backward",
	ActionThrustLeft:          "move.thrust_left",
	ActionThrustRight:         "move.thrust_right",
	ActionThrustUp:            "move.thrust_up",
	ActionThrustDown:          "move.thrust_down",
	ActionBrake:               "move.brake",
	ActionDriftToggle:         "move.drift_toggle",
	ActionSimSpeedIncrease:    "sim.speed_increase",
	ActionSimSpeedDecrease:    "sim.speed_decrease",
	ActionSimPauseToggle:      "sim.pause_toggle",
	ActionSimTrackNext:        "sim.track_next",
	ActionSimTrackStop:        "sim.track_stop",
	ActionHUDToggle:           "hud.toggle",
	ActionHUDClientList:       "hud.client_list",
	ActionUISettings:          "ui.settings",
	ActionUIFullscreen:        "ui.fullscreen",
	ActionReplOpen:            "repl.open",
	ActionReplClose:           "repl.close",
	ActionReplHistoryPrev:     "repl.history_prev",
	ActionReplHistoryNext:     "repl.history_next",
}

// nameActions is the reverse of actionNames, built at init.
var nameActions map[string]InputAction

func init() {
	nameActions = make(map[string]InputAction, len(actionNames))
	for action, name := range actionNames {
		nameActions[name] = action
	}
}

// String returns the dot-notation vocabulary name for the action.
func (a InputAction) String() string {
	if s, ok := actionNames[a]; ok {
		return s
	}
	return "unknown"
}

// ParseAction resolves a dot-notation name to an InputAction.
// Returns (ActionNone, false) if the name is not in the vocabulary.
func ParseAction(name string) (InputAction, bool) {
	a, ok := nameActions[name]
	return a, ok
}

// AllActions returns every defined action in declaration order (excluding ActionNone).
func AllActions() []InputAction {
	actions := make([]InputAction, 0, int(actionCount)-1)
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		actions = append(actions, a)
	}
	return actions
}

// OrderedActions returns all defined actions in a logical display order for the
// keybinding editor UI. Every action appears exactly once; the order groups
// related actions so the list reads naturally from top to bottom.
func OrderedActions() []InputAction {
	return []InputAction{
		// Application
		ActionUISettings, ActionUIFullscreen,
		// Simulation
		ActionSimSpeedIncrease, ActionSimSpeedDecrease, ActionSimPauseToggle,
		ActionSimTrackNext, ActionSimTrackStop,
		// HUD
		ActionHUDToggle, ActionHUDClientList,
		// REPL overlay
		ActionReplOpen, ActionReplClose, ActionReplHistoryPrev, ActionReplHistoryNext,
		// Camera
		ActionCameraPitchUp, ActionCameraPitchDown,
		ActionCameraYawLeft, ActionCameraYawRight,
		ActionCameraRollLeft, ActionCameraRollRight,
		ActionCameraZoomIn, ActionCameraZoomOut,
		ActionCameraReset, ActionCameraToggleFreeFly,
		// Movement / locomotion
		ActionThrustForward, ActionThrustBackward,
		ActionThrustLeft, ActionThrustRight,
		ActionThrustUp, ActionThrustDown,
		ActionBrake, ActionDriftToggle,
	}
}

// isReplContext reports whether an action belongs to the REPL input context.
// REPL-context actions are only evaluated when the REPL overlay is open;
// they are never in conflict with world-context actions even if the same key
// is used (e.g. ESCAPE closes both the REPL and stops tracking).
func isReplContext(a InputAction) bool {
	switch a {
	case ActionReplOpen, ActionReplClose, ActionReplHistoryPrev, ActionReplHistoryNext:
		return true
	}
	return false
}

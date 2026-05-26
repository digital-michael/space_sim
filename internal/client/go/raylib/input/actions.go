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
	ActionSimTimescaleIncrease InputAction = 19
	ActionSimTimescaleDecrease InputAction = 20
	ActionSimPauseToggle       InputAction = 21
	ActionSimTrackNext         InputAction = 22
	ActionSimTrackStop         InputAction = 23

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

	// Simulation tick speed (physics tick rate; separate from time scale)
	ActionSimTickSpeedIncrease InputAction = 32
	ActionSimTickSpeedDecrease InputAction = 33

	// Simulation dataset size
	ActionSimDatasetIncrease InputAction = 34
	ActionSimDatasetDecrease InputAction = 35

	// UI actions
	ActionUISystemSelector  InputAction = 36
	ActionUILabelCycle      InputAction = 37
	ActionUIInfraCycle      InputAction = 38
	ActionUIMouseModeToggle InputAction = 39
	ActionUIQuit            InputAction = 40
	ActionUIRecordToggle    InputAction = 41
	ActionUIRecordPause     InputAction = 42

	// Camera positional actions
	ActionCameraCenter InputAction = 43

	// Navigation (hierarchy traversal; only active in tracking mode)
	ActionNavChildNext   InputAction = 44
	ActionNavParent      InputAction = 45
	ActionNavSiblingNext InputAction = 46
	ActionNavSiblingPrev InputAction = 47
	ActionNavJump        InputAction = 48

	// Debug
	ActionUIDebugLabels InputAction = 49

	// sentinel — must equal last constant + 1
	actionCount InputAction = 50
)

// actionNames maps each action to its dot-notation vocabulary name.
var actionNames = map[InputAction]string{
	ActionCameraPitchUp:        "camera.pitch_up",
	ActionCameraPitchDown:      "camera.pitch_down",
	ActionCameraYawLeft:        "camera.yaw_left",
	ActionCameraYawRight:       "camera.yaw_right",
	ActionCameraRollLeft:       "camera.roll_left",
	ActionCameraRollRight:      "camera.roll_right",
	ActionCameraZoomIn:         "camera.zoom_in",
	ActionCameraZoomOut:        "camera.zoom_out",
	ActionCameraReset:          "camera.reset",
	ActionCameraToggleFreeFly:  "camera.toggle_free_fly",
	ActionThrustForward:        "move.thrust_forward",
	ActionThrustBackward:       "move.thrust_backward",
	ActionThrustLeft:           "move.thrust_left",
	ActionThrustRight:          "move.thrust_right",
	ActionThrustUp:             "move.thrust_up",
	ActionThrustDown:           "move.thrust_down",
	ActionBrake:                "move.brake",
	ActionDriftToggle:          "move.drift_toggle",
	ActionSimTimescaleIncrease: "sim.timescale_increase",
	ActionSimTimescaleDecrease: "sim.timescale_decrease",
	ActionSimPauseToggle:       "sim.pause_toggle",
	ActionSimTrackNext:         "sim.track_next",
	ActionSimTrackStop:         "sim.track_stop",
	ActionHUDToggle:            "hud.toggle",
	ActionHUDClientList:        "hud.client_list",
	ActionUISettings:           "ui.settings",
	ActionUIFullscreen:         "ui.fullscreen",
	ActionReplOpen:             "repl.open",
	ActionReplClose:            "repl.close",
	ActionReplHistoryPrev:      "repl.history_prev",
	ActionReplHistoryNext:      "repl.history_next",

	ActionSimTickSpeedIncrease: "sim.tick_speed_increase",
	ActionSimTickSpeedDecrease: "sim.tick_speed_decrease",
	ActionSimDatasetIncrease:   "sim.dataset_increase",
	ActionSimDatasetDecrease:   "sim.dataset_decrease",

	ActionUISystemSelector:  "ui.system_selector",
	ActionUILabelCycle:      "ui.label_cycle",
	ActionUIInfraCycle:      "ui.infra_cycle",
	ActionUIMouseModeToggle: "ui.mouse_mode_toggle",
	ActionUIQuit:            "ui.quit",
	ActionUIRecordToggle:    "ui.record_toggle",
	ActionUIRecordPause:     "ui.record_pause",

	ActionCameraCenter: "camera.center",

	ActionNavChildNext:   "nav.child_next",
	ActionNavParent:      "nav.parent",
	ActionNavSiblingNext: "nav.sibling_next",
	ActionNavSiblingPrev: "nav.sibling_prev",
	ActionNavJump:        "nav.jump",

	ActionUIDebugLabels: "ui.debug_labels",
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
		ActionUISystemSelector, ActionUILabelCycle, ActionUIInfraCycle,
		ActionUIMouseModeToggle, ActionUIQuit,
		ActionUIRecordToggle, ActionUIRecordPause,
		ActionUIDebugLabels,
		// Simulation
		ActionSimTimescaleIncrease, ActionSimTimescaleDecrease,
		ActionSimTickSpeedIncrease, ActionSimTickSpeedDecrease,
		ActionSimDatasetIncrease, ActionSimDatasetDecrease,
		ActionSimPauseToggle, ActionSimTrackNext, ActionSimTrackStop,
		// HUD
		ActionHUDToggle, ActionHUDClientList,
		// REPL overlay
		ActionReplOpen, ActionReplClose, ActionReplHistoryPrev, ActionReplHistoryNext,
		// Camera
		ActionCameraPitchUp, ActionCameraPitchDown,
		ActionCameraYawLeft, ActionCameraYawRight,
		ActionCameraRollLeft, ActionCameraRollRight,
		ActionCameraZoomIn, ActionCameraZoomOut,
		ActionCameraReset, ActionCameraToggleFreeFly, ActionCameraCenter,
		// Navigation (tracking-mode hierarchy)
		ActionNavChildNext, ActionNavParent,
		ActionNavSiblingNext, ActionNavSiblingPrev,
		ActionNavJump,
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

// isMoveContext reports whether an action belongs to the ship-movement context.
// Move actions are only evaluated when the simulation is in ship-physics mode;
// they can share keys with camera actions (evaluated only in camera modes)
// without functional conflict.
func isMoveContext(a InputAction) bool {
	switch a {
	case ActionThrustForward, ActionThrustBackward,
		ActionThrustLeft, ActionThrustRight,
		ActionThrustUp, ActionThrustDown,
		ActionBrake, ActionDriftToggle:
		return true
	}
	return false
}

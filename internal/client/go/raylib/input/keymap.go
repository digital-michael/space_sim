package input

import rl "github.com/gen2brain/raylib-go/raylib"

// ModSet is a bitmask of required modifier keys.
type ModSet uint8

const (
	ModShift ModSet = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
)

// binding associates a Raylib key code and optional modifier set with an action.
type binding struct {
	key  int32
	mods ModSet
}

// KeyMap holds the active per-frame binding state. It is safe to read from
// multiple goroutines, but DrainQueue and the IsPressed/IsDown methods must
// all be called from the same OS thread (the Raylib render thread).
type KeyMap struct {
	bindings    [actionCount]binding
	pressed     map[int32]struct{} // key codes drained from rl.GetKeyPressed this frame
	pressedMods map[int32]ModSet   // modifier state captured when each key entered the queue
}

// newKeyMap allocates an empty KeyMap. Use the loader to populate bindings.
func newKeyMap() *KeyMap {
	return &KeyMap{
		pressed:     make(map[int32]struct{}, 16),
		pressedMods: make(map[int32]ModSet, 16),
	}
}

// DrainQueue captures all keys pressed since the last call by draining the
// Raylib key-press queue. Call exactly once per frame at the top of the input
// handler, before any IsPressed checks.
//
// Using DrainQueue + IsPressed instead of rl.IsKeyPressed prevents missed
// short-tap events when the sim loop runs slower than 60 Hz.
//
// Modifier state is sampled once before draining and associated with every key
// in this batch. This avoids false-negative mod checks when the user releases
// a modifier key before the next frame's input handler runs.
func (km *KeyMap) DrainQueue() {
	for k := range km.pressed {
		delete(km.pressed, k)
	}
	for k := range km.pressedMods {
		delete(km.pressedMods, k)
	}
	// Sample modifier state once, before draining the queue, so every key
	// pressed this frame inherits the modifiers that were held when the batch
	// was captured rather than when IsPressed is eventually called.
	currentMods := km.sampleMods()
	for {
		key := rl.GetKeyPressed()
		if key == 0 {
			break
		}
		km.pressed[int32(key)] = struct{}{}
		km.pressedMods[int32(key)] = currentMods
	}
}

// sampleMods reads the current modifier key state and returns it as a ModSet.
func (km *KeyMap) sampleMods() ModSet {
	var ms ModSet
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		ms |= ModShift
	}
	if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
		ms |= ModCtrl
	}
	if rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt) {
		ms |= ModAlt
	}
	if rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper) {
		ms |= ModSuper
	}
	return ms
}

// IsPressed reports whether the action's bound key was pressed this frame
// (captured by DrainQueue) with the required modifiers held.
//
// Modifier state is checked against the snapshot recorded at DrainQueue time,
// so modifiers released before the next frame are still correctly detected.
//
// Call DrainQueue once per frame before calling IsPressed.
func (km *KeyMap) IsPressed(action InputAction) bool {
	if action == ActionNone || action >= actionCount {
		return false
	}
	b := km.bindings[action]
	if b.key == 0 {
		return false
	}
	storedMods, ok := km.pressedMods[b.key]
	if !ok {
		return false
	}
	// All required modifiers must have been held when the key was pressed.
	return storedMods&b.mods == b.mods
}

// IsDown reports whether the action's bound key is currently held down with
// the required modifiers. Delegates directly to rl.IsKeyDown.
func (km *KeyMap) IsDown(action InputAction) bool {
	if action == ActionNone || action >= actionCount {
		return false
	}
	b := km.bindings[action]
	if b.key == 0 {
		return false
	}
	if !rl.IsKeyDown(b.key) {
		return false
	}
	return km.modsHeld(b.mods)
}

// modsHeld returns true when all required modifiers in mods are currently
// pressed. A zero ModSet always returns true.
func (km *KeyMap) modsHeld(mods ModSet) bool {
	if mods&ModShift != 0 {
		if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
			return false
		}
	}
	if mods&ModCtrl != 0 {
		if !rl.IsKeyDown(rl.KeyLeftControl) && !rl.IsKeyDown(rl.KeyRightControl) {
			return false
		}
	}
	if mods&ModAlt != 0 {
		if !rl.IsKeyDown(rl.KeyLeftAlt) && !rl.IsKeyDown(rl.KeyRightAlt) {
			return false
		}
	}
	if mods&ModSuper != 0 {
		if !rl.IsKeyDown(rl.KeyLeftSuper) && !rl.IsKeyDown(rl.KeyRightSuper) {
			return false
		}
	}
	return true
}

// BoundKey returns the Raylib key code bound to action, or 0 if unbound.
func (km *KeyMap) BoundKey(action InputAction) int32 {
	if action == ActionNone || action >= actionCount {
		return 0
	}
	return km.bindings[action].key
}

// BoundMods returns the modifier set for action.
func (km *KeyMap) BoundMods(action InputAction) ModSet {
	if action == ActionNone || action >= actionCount {
		return 0
	}
	return km.bindings[action].mods
}

// SetBinding replaces the binding for action without conflict checking.
// Call ConflictFor first to detect conflicts before committing.
func (km *KeyMap) SetBinding(action InputAction, key int32, mods ModSet) {
	if action == ActionNone || action >= actionCount {
		return
	}
	km.bindings[action] = binding{key: key, mods: mods}
}

// ConflictFor returns the first world-context action already bound to
// (key, mods), excluding except. Returns (ActionNone, false) when no
// conflict exists. REPL-context and move-context actions are excluded from
// conflict checks because they operate in separate input contexts.
func (km *KeyMap) ConflictFor(key int32, mods ModSet, except InputAction) (InputAction, bool) {
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		if a == except {
			continue
		}
		if isReplContext(a) {
			continue
		}
		if isMoveContext(a) {
			continue
		}
		b := km.bindings[a]
		if b.key == key && b.mods == mods {
			return a, true
		}
	}
	return ActionNone, false
}

// AnyPressed returns the first key code captured by the most recent
// DrainQueue call, and true. Returns (0, false) if no key was pressed.
func (km *KeyMap) AnyPressed() (int32, bool) {
	for k := range km.pressed {
		return k, true
	}
	return 0, false
}

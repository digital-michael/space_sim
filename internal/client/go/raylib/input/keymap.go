package input

import rl "github.com/gen2brain/raylib-go/raylib"

// ModSet is a bitmask of required modifier keys.
type ModSet uint8

const (
	ModShift ModSet = 1 << iota
	ModCtrl
	ModAlt
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
	bindings [actionCount]binding
	pressed  map[int32]struct{} // key codes drained from rl.GetKeyPressed this frame
}

// newKeyMap allocates an empty KeyMap. Use the loader to populate bindings.
func newKeyMap() *KeyMap {
	return &KeyMap{
		pressed: make(map[int32]struct{}, 16),
	}
}

// DrainQueue captures all keys pressed since the last call by draining the
// Raylib key-press queue. Call exactly once per frame at the top of the input
// handler, before any IsPressed checks.
//
// Using DrainQueue + IsPressed instead of rl.IsKeyPressed prevents missed
// short-tap events when the sim loop runs slower than 60 Hz.
func (km *KeyMap) DrainQueue() {
	for k := range km.pressed {
		delete(km.pressed, k)
	}
	for {
		key := rl.GetKeyPressed()
		if key == 0 {
			break
		}
		km.pressed[int32(key)] = struct{}{}
	}
}

// IsPressed reports whether the action's bound key was pressed this frame
// (captured by DrainQueue) with the required modifiers held.
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
	if _, ok := km.pressed[b.key]; !ok {
		return false
	}
	return km.modsHeld(b.mods)
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

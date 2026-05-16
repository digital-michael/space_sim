package input

import rl "github.com/gen2brain/raylib-go/raylib"

// keyNames maps the key name strings used in profile JSON files to
// Raylib key codes. Names follow the Raylib KEY_* constant names with the
// "KEY_" prefix stripped and case-normalised to upper-case.
//
// Example: "W" → rl.KeyW, "LEFT_CONTROL" → rl.KeyLeftControl.
var keyNames = map[string]int32{
	// Alphabetic
	"A": rl.KeyA, "B": rl.KeyB, "C": rl.KeyC, "D": rl.KeyD,
	"E": rl.KeyE, "F": rl.KeyF, "G": rl.KeyG, "H": rl.KeyH,
	"I": rl.KeyI, "J": rl.KeyJ, "K": rl.KeyK, "L": rl.KeyL,
	"M": rl.KeyM, "N": rl.KeyN, "O": rl.KeyO, "P": rl.KeyP,
	"Q": rl.KeyQ, "R": rl.KeyR, "S": rl.KeyS, "T": rl.KeyT,
	"U": rl.KeyU, "V": rl.KeyV, "W": rl.KeyW, "X": rl.KeyX,
	"Y": rl.KeyY, "Z": rl.KeyZ,

	// Digits
	"ZERO": rl.KeyZero, "ONE": rl.KeyOne, "TWO": rl.KeyTwo,
	"THREE": rl.KeyThree, "FOUR": rl.KeyFour, "FIVE": rl.KeyFive,
	"SIX": rl.KeySix, "SEVEN": rl.KeySeven, "EIGHT": rl.KeyEight,
	"NINE": rl.KeyNine,

	// Navigation / control
	"SPACE":       rl.KeySpace,
	"ENTER":       rl.KeyEnter,
	"ESCAPE":      rl.KeyEscape,
	"TAB":         rl.KeyTab,
	"BACKSPACE":   rl.KeyBackspace,
	"INSERT":      rl.KeyInsert,
	"DELETE":      rl.KeyDelete,
	"RIGHT":       rl.KeyRight,
	"LEFT":        rl.KeyLeft,
	"DOWN":        rl.KeyDown,
	"UP":          rl.KeyUp,
	"PAGE_UP":     rl.KeyPageUp,
	"PAGE_DOWN":   rl.KeyPageDown,
	"HOME":        rl.KeyHome,
	"END":         rl.KeyEnd,
	"CAPS_LOCK":   rl.KeyCapsLock,
	"SCROLL_LOCK": rl.KeyScrollLock,
	"NUM_LOCK":    rl.KeyNumLock,
	"PAUSE":       rl.KeyPause,

	// Function keys
	"F1": rl.KeyF1, "F2": rl.KeyF2, "F3": rl.KeyF3, "F4": rl.KeyF4,
	"F5": rl.KeyF5, "F6": rl.KeyF6, "F7": rl.KeyF7, "F8": rl.KeyF8,
	"F9": rl.KeyF9, "F10": rl.KeyF10, "F11": rl.KeyF11, "F12": rl.KeyF12,

	// Modifier keys
	"LEFT_SHIFT":    rl.KeyLeftShift,
	"LEFT_CONTROL":  rl.KeyLeftControl,
	"LEFT_ALT":      rl.KeyLeftAlt,
	"LEFT_SUPER":    rl.KeyLeftSuper,
	"RIGHT_SHIFT":   rl.KeyRightShift,
	"RIGHT_CONTROL": rl.KeyRightControl,
	"RIGHT_ALT":     rl.KeyRightAlt,
	"RIGHT_SUPER":   rl.KeyRightSuper,

	// Punctuation / symbols
	"SEMICOLON":     rl.KeySemicolon,
	"SLASH":         rl.KeySlash,
	"GRAVE":         rl.KeyGrave,
	"LEFT_BRACKET":  rl.KeyLeftBracket,
	"BACKSLASH":     rl.KeyBackSlash,
	"RIGHT_BRACKET": rl.KeyRightBracket,
	"APOSTROPHE":    rl.KeyApostrophe,
	"EQUAL":         rl.KeyEqual,
	"MINUS":         rl.KeyMinus,
	"COMMA":         rl.KeyComma,
	"PERIOD":        rl.KeyPeriod,

	// Numpad
	"KP_0":        rl.KeyKp0,
	"KP_1":        rl.KeyKp1,
	"KP_2":        rl.KeyKp2,
	"KP_3":        rl.KeyKp3,
	"KP_4":        rl.KeyKp4,
	"KP_5":        rl.KeyKp5,
	"KP_6":        rl.KeyKp6,
	"KP_7":        rl.KeyKp7,
	"KP_8":        rl.KeyKp8,
	"KP_9":        rl.KeyKp9,
	"KP_DECIMAL":  rl.KeyKpDecimal,
	"KP_DIVIDE":   rl.KeyKpDivide,
	"KP_MULTIPLY": rl.KeyKpMultiply,
	"KP_SUBTRACT": rl.KeyKpSubtract,
	"KP_ADD":      rl.KeyKpAdd,
	"KP_ENTER":    rl.KeyKpEnter,
	"KP_EQUAL":    rl.KeyKpEqual,
}

// ParseKeyName resolves a key name string (e.g. "W", "LEFT_CONTROL", "KP_8")
// to its Raylib key code. Returns (0, false) when the name is unrecognised.
func ParseKeyName(name string) (int32, bool) {
	code, ok := keyNames[name]
	return code, ok
}

// keyCodeToName is the reverse of keyNames, built at init.
// When multiple names map to the same code, the first one encountered wins.
var keyCodeToName map[int32]string

func init() {
	keyCodeToName = make(map[int32]string, len(keyNames))
	for name, code := range keyNames {
		if _, exists := keyCodeToName[code]; !exists {
			keyCodeToName[code] = name
		}
	}
}

// KeyName returns the canonical name string for a Raylib key code,
// or "UNKNOWN" if the code is not in the vocabulary.
func KeyName(code int32) string {
	if name, ok := keyCodeToName[code]; ok {
		return name
	}
	return "UNKNOWN"
}

// KeyNameOf returns the canonical string name for a Raylib key code, or an
// empty string when the code is not in the table.
func KeyNameOf(code int32) string {
	for name, c := range keyNames {
		if c == code {
			return name
		}
	}
	return ""
}

// AllKeyNames returns all known key name strings in no guaranteed order.
func AllKeyNames() []string {
	names := make([]string, 0, len(keyNames))
	for name := range keyNames {
		names = append(names, name)
	}
	return names
}

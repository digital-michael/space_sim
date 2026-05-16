package input

import (
	"fmt"
	"strings"
)

// ConflictError is returned when two world-context actions share the same
// (key, modifier) combination in the active binding set.
type ConflictError struct {
	Key     string
	Mods    []string
	Action1 string
	Action2 string
}

func (e *ConflictError) Error() string {
	keyStr := e.Key
	if len(e.Mods) > 0 {
		keyStr = strings.Join(e.Mods, "+") + "+" + keyStr
	}
	return fmt.Sprintf(
		"keybinding conflict: %q is bound to both %q and %q",
		keyStr, e.Action1, e.Action2,
	)
}

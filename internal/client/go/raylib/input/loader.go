package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// profileJSON is the on-disk shape of a stock profile (e.g. data/profiles/laptop.json).
type profileJSON struct {
	Version  int                `json:"version"`
	Name     string             `json:"name"`
	Bindings []bindingEntryJSON `json:"bindings"`
}

// keybindingsJSON is the on-disk shape of configs/keybindings.json.
type keybindingsJSON struct {
	Version     int                `json:"version"`
	BaseProfile string             `json:"base_profile"`
	Overrides   []bindingEntryJSON `json:"overrides"`
}

// bindingEntryJSON is one entry in either a profile or override list.
type bindingEntryJSON struct {
	Action string   `json:"action"`
	Key    string   `json:"key"`
	Mods   []string `json:"mods"`
}

// LoadKeyMap builds a KeyMap from a stock profile and an optional user
// override file.
//
//   - profilesDir is the directory containing stock profile JSON files
//     (e.g. "data/profiles").
//   - configPath is the optional user override file
//     (e.g. "configs/keybindings.json"); if the file does not exist the
//     laptop profile is used with no overrides.
//
// Returns a non-nil error (and exits) when:
//   - The override file exists but contains an unknown action or key name.
//   - Two world-context actions share the same (key, mods) combination.
func LoadKeyMap(profilesDir, configPath string) (*KeyMap, error) {
	baseProfile := "laptop"
	var overrides []bindingEntryJSON

	data, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}
	if err == nil {
		var cfg keybindingsJSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", configPath, err)
		}
		if cfg.Version != 1 {
			return nil, fmt.Errorf("%s: unsupported version %d (want 1)", configPath, cfg.Version)
		}
		if cfg.BaseProfile != "" {
			baseProfile = cfg.BaseProfile
		}
		overrides = cfg.Overrides
	}

	profilePath := filepath.Join(profilesDir, baseProfile+".json")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("reading profile %q: %w", profilePath, err)
	}
	var profile profileJSON
	if err := json.Unmarshal(profileData, &profile); err != nil {
		return nil, fmt.Errorf("parsing profile %q: %w", profilePath, err)
	}

	// Build binding slice: profile entries first, then override entries replace any
	// entries for the same action.
	merged, err := mergeBindings(profile.Bindings, overrides, configPath)
	if err != nil {
		return nil, err
	}

	if err := validateBindings(merged); err != nil {
		return nil, err
	}

	return buildKeyMap(merged)
}

// mergeBindings applies override entries on top of base entries.
// Override entries for the same action replace the base entry.
// Returns an error if any entry references an unknown action or key name.
func mergeBindings(base, overrides []bindingEntryJSON, overrideSource string) ([]bindingEntryJSON, error) {
	// Validate all entries first so errors are caught before any state is built.
	for _, e := range base {
		if _, ok := ParseAction(e.Action); !ok {
			return nil, fmt.Errorf("unknown action %q in profile", e.Action)
		}
		if _, ok := ParseKeyName(strings.ToUpper(e.Key)); !ok {
			return nil, fmt.Errorf("unknown key name %q in profile (action %s)", e.Key, e.Action)
		}
	}
	for _, e := range overrides {
		if _, ok := ParseAction(e.Action); !ok {
			return nil, fmt.Errorf("%s: unknown action %q", overrideSource, e.Action)
		}
		if _, ok := ParseKeyName(strings.ToUpper(e.Key)); !ok {
			return nil, fmt.Errorf("%s: unknown key name %q (action %s)", overrideSource, e.Key, e.Action)
		}
	}

	// Build indexed map from base, then apply overrides.
	indexed := make(map[string]bindingEntryJSON, len(base))
	for _, e := range base {
		indexed[e.Action] = e
	}
	for _, e := range overrides {
		indexed[e.Action] = e
	}

	// Reconstruct a flat slice preserving declaration order for conflict messages.
	result := make([]bindingEntryJSON, 0, len(indexed))
	seen := make(map[string]bool, len(indexed))
	for _, e := range base {
		if !seen[e.Action] {
			seen[e.Action] = true
			result = append(result, indexed[e.Action])
		}
	}
	// Append any override-only actions (not in base).
	for _, e := range overrides {
		if !seen[e.Action] {
			seen[e.Action] = true
			result = append(result, indexed[e.Action])
		}
	}
	return result, nil
}

// validateBindings checks for hard conflicts: two world-context actions bound
// to the same (key, mods) combination.
func validateBindings(entries []bindingEntryJSON) error {
	type keyMods struct {
		key  int32
		mods ModSet
	}
	seen := make(map[keyMods]string) // keyMods → action name
	for _, e := range entries {
		action, _ := ParseAction(e.Action)
		if isReplContext(action) {
			continue // REPL-context bindings are exempt from world-context conflict checks.
		}
		key, _ := ParseKeyName(strings.ToUpper(e.Key))
		mods, err := parseMods(e.Mods)
		if err != nil {
			return fmt.Errorf("action %s: %w", e.Action, err)
		}
		km := keyMods{key: key, mods: mods}
		if prev, conflict := seen[km]; conflict {
			keyStr := strings.ToUpper(e.Key)
			return &ConflictError{
				Key:     keyStr,
				Mods:    e.Mods,
				Action1: prev,
				Action2: e.Action,
			}
		}
		seen[km] = e.Action
	}
	return nil
}

// buildKeyMap constructs a KeyMap from validated binding entries.
func buildKeyMap(entries []bindingEntryJSON) (*KeyMap, error) {
	km := newKeyMap()
	for _, e := range entries {
		action, _ := ParseAction(e.Action)
		key, _ := ParseKeyName(strings.ToUpper(e.Key))
		mods, err := parseMods(e.Mods)
		if err != nil {
			return nil, fmt.Errorf("action %s: %w", e.Action, err)
		}
		km.bindings[action] = binding{key: key, mods: mods}
	}
	return km, nil
}

// parseMods converts a slice of modifier name strings to a ModSet.
func parseMods(mods []string) (ModSet, error) {
	var ms ModSet
	for _, m := range mods {
		switch strings.ToUpper(m) {
		case "SHIFT":
			ms |= ModShift
		case "CTRL":
			ms |= ModCtrl
		case "ALT":
			ms |= ModAlt
		case "SUPER":
			ms |= ModSuper
		default:
			return 0, fmt.Errorf("unknown modifier %q (valid: SHIFT, CTRL, ALT, SUPER)", m)
		}
	}
	return ms, nil
}

// modsToStrings converts a ModSet to the string slice representation used in
// JSON files (e.g. ["SHIFT", "CTRL"]).
// ModsToStrings converts a ModSet bitmask into a human-readable slice of
// modifier name strings ("SHIFT", "CTRL", "ALT", "SUPER").
// An empty ModSet returns an empty slice (never nil).
func ModsToStrings(mods ModSet) []string {
	return modsToStrings(mods)
}

func modsToStrings(mods ModSet) []string {
	if mods == 0 {
		return []string{}
	}
	var result []string
	if mods&ModShift != 0 {
		result = append(result, "SHIFT")
	}
	if mods&ModCtrl != 0 {
		result = append(result, "CTRL")
	}
	if mods&ModAlt != 0 {
		result = append(result, "ALT")
	}
	if mods&ModSuper != 0 {
		result = append(result, "SUPER")
	}
	return result
}

// WriteKeybindingsFile serialises the current live KeyMap as a keybindings
// override file at path, using baseProfile as the base profile name.
// Parent directories are created as needed. The write is atomic on platforms
// that support os.Rename (POSIX, Windows 10+).
func WriteKeybindingsFile(path string, km *KeyMap, baseProfile string) error {
	entries := make([]bindingEntryJSON, 0, int(actionCount)-1)
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		key := km.bindings[a].key
		if key == 0 {
			continue // unbound — skip
		}
		entries = append(entries, bindingEntryJSON{
			Action: a.String(),
			Key:    KeyName(key),
			Mods:   modsToStrings(km.bindings[a].mods),
		})
	}
	out := keybindingsJSON{
		Version:     1,
		BaseProfile: baseProfile,
		Overrides:   entries,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling keybindings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating keybindings directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing keybindings temp file: %w", err)
	}
	return os.Rename(tmp, path)
}

// ScanKeybindingsDir returns the paths of all *.json files in dir.
// Returns nil (not an error) when the directory does not exist.
func ScanKeybindingsDir(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scanning keybindings dir %q: %w", dir, err)
	}
	return matches, nil
}

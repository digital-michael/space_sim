package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── Key name parsing ─────────────────────────────────────────────────────────

func TestParseKeyNameKnown(t *testing.T) {
	cases := []struct {
		name string
		want int32
	}{
		{"W", 87},       // KeyW = ASCII 'W'
		{"ESCAPE", 256}, // rl.KeyEscape
		{"SPACE", 32},   // rl.KeySpace
		{"UP", 265},     // rl.KeyUp
		{"KP_8", 328},   // rl.KeyKp8
	}
	for _, tc := range cases {
		code, ok := ParseKeyName(tc.name)
		if !ok {
			t.Errorf("ParseKeyName(%q) returned false", tc.name)
			continue
		}
		if code != tc.want {
			t.Errorf("ParseKeyName(%q) = %d, want %d", tc.name, code, tc.want)
		}
	}
}

func TestParseKeyNameUnknown(t *testing.T) {
	_, ok := ParseKeyName("NOT_A_KEY")
	if ok {
		t.Error("ParseKeyName(unknown) returned true")
	}
}

// ─── Mod parsing ──────────────────────────────────────────────────────────────

func TestParseModsEmpty(t *testing.T) {
	ms, err := parseMods(nil)
	if err != nil {
		t.Fatalf("parseMods(nil) error: %v", err)
	}
	if ms != 0 {
		t.Errorf("parseMods(nil) = %d, want 0", ms)
	}
}

func TestParseModsAll(t *testing.T) {
	ms, err := parseMods([]string{"SHIFT", "CTRL", "ALT"})
	if err != nil {
		t.Fatalf("parseMods error: %v", err)
	}
	if ms != ModShift|ModCtrl|ModAlt {
		t.Errorf("parseMods all = %d, want %d", ms, ModShift|ModCtrl|ModAlt)
	}
}

func TestParseModsUnknown(t *testing.T) {
	_, err := parseMods([]string{"SUPER"})
	if err != nil {
		t.Errorf("parseMods(SUPER) expected no error, got: %v", err)
	}
	_, err = parseMods([]string{"INVALID"})
	if err == nil {
		t.Error("parseMods(INVALID) expected error, got nil")
	}
}

// ─── Conflict detection ───────────────────────────────────────────────────────

func TestConflictDetected(t *testing.T) {
	entries := []bindingEntryJSON{
		{Action: "move.thrust_forward", Key: "W", Mods: []string{}},
		{Action: "camera.reset", Key: "W", Mods: []string{}}, // same key → conflict
	}
	err := validateBindings(entries)
	if err == nil {
		t.Fatal("validateBindings: expected ConflictError, got nil")
	}
	var ce *ConflictError
	if !isConflictError(err, &ce) {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if ce.Key != "W" {
		t.Errorf("ConflictError.Key = %q, want W", ce.Key)
	}
}

func TestNoConflictOnReplContextSameKey(t *testing.T) {
	// sim.track_stop and repl.close both use ESCAPE — allowed because different contexts.
	entries := []bindingEntryJSON{
		{Action: "sim.track_stop", Key: "ESCAPE", Mods: []string{}},
		{Action: "repl.close", Key: "ESCAPE", Mods: []string{}},
	}
	if err := validateBindings(entries); err != nil {
		t.Errorf("validateBindings: unexpected error for cross-context ESCAPE: %v", err)
	}
}

func TestNoConflictWithDifferentMods(t *testing.T) {
	entries := []bindingEntryJSON{
		{Action: "move.thrust_forward", Key: "W", Mods: []string{}},
		{Action: "move.thrust_backward", Key: "W", Mods: []string{"SHIFT"}},
	}
	if err := validateBindings(entries); err != nil {
		t.Errorf("validateBindings: unexpected error for different mods: %v", err)
	}
}

// ─── Override merge ───────────────────────────────────────────────────────────

func TestMergeOverrideReplacesBaseBinding(t *testing.T) {
	base := []bindingEntryJSON{
		{Action: "move.thrust_forward", Key: "W"},
	}
	overrides := []bindingEntryJSON{
		{Action: "move.thrust_forward", Key: "I"},
	}
	merged, err := mergeBindings(base, overrides, "test")
	if err != nil {
		t.Fatalf("mergeBindings error: %v", err)
	}
	if len(merged) != 1 {
		t.Fatalf("len(merged) = %d, want 1", len(merged))
	}
	if merged[0].Key != "I" {
		t.Errorf("merged key = %q, want I", merged[0].Key)
	}
}

func TestMergeOverrideUnknownActionErrors(t *testing.T) {
	base := []bindingEntryJSON{}
	overrides := []bindingEntryJSON{
		{Action: "not.a.real.action", Key: "W"},
	}
	_, err := mergeBindings(base, overrides, "test.json")
	if err == nil {
		t.Fatal("expected error for unknown override action, got nil")
	}
}

func TestMergeOverrideUnknownKeyErrors(t *testing.T) {
	base := []bindingEntryJSON{}
	overrides := []bindingEntryJSON{
		{Action: "move.thrust_forward", Key: "BOGUS_KEY_NAME"},
	}
	_, err := mergeBindings(base, overrides, "test.json")
	if err == nil {
		t.Fatal("expected error for unknown key name, got nil")
	}
}

// ─── Full LoadKeyMap integration (no Raylib window needed) ───────────────────

func TestLoadKeyMapMissingConfigFallsBackToLaptop(t *testing.T) {
	dir := laptopProfileDir(t)
	km, err := LoadKeyMap(dir, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadKeyMap error: %v", err)
	}
	// Laptop profile maps W → thrust_forward.
	key := km.BoundKey(ActionThrustForward)
	wKey, _ := ParseKeyName("W")
	if key != wKey {
		t.Errorf("BoundKey(ThrustForward) = %d, want %d (W)", key, wKey)
	}
}

func TestLoadKeyMapInvalidKeyNameExits(t *testing.T) {
	dir := laptopProfileDir(t)
	cfg := writeKeybindings(t, `{"version":1,"base_profile":"laptop","overrides":[{"action":"move.thrust_forward","key":"BOGUS"}]}`)
	_, err := LoadKeyMap(dir, cfg)
	if err == nil {
		t.Fatal("expected error for invalid key name, got nil")
	}
	if !strings.Contains(err.Error(), "BOGUS") {
		t.Errorf("error %q should mention the bad key name", err.Error())
	}
}

func TestLoadKeyMapConflictExits(t *testing.T) {
	dir := laptopProfileDir(t)
	// Override thrust_forward and camera.reset both to the same key (W).
	// Laptop profile already has camera.reset = R; override it to W → conflict.
	cfg := writeKeybindings(t, `{"version":1,"base_profile":"laptop","overrides":[{"action":"camera.reset","key":"W","mods":[]}]}`)
	_, err := LoadKeyMap(dir, cfg)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// laptopProfileDir returns the real data/profiles directory relative to the
// workspace root. Tests that load stock profiles call this helper.
func laptopProfileDir(t *testing.T) string {
	t.Helper()
	// Walk up from the test file location until we find data/profiles/.
	dir := "."
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "data", "profiles")
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
		dir = filepath.Join(dir, "..")
	}
	t.Skip("data/profiles not found — skipping integration test")
	return ""
}

func writeKeybindings(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeKeybindings: %v", err)
	}
	return path
}

// isConflictError unwraps err into *ConflictError.
func isConflictError(err error, out **ConflictError) bool {
	if ce, ok := err.(*ConflictError); ok {
		*out = ce
		return true
	}
	return false
}

// ─── KeyName (reverse lookup) ─────────────────────────────────────────────────

func TestKeyNameRoundTrip(t *testing.T) {
	for name, want := range keyNames {
		code, ok := ParseKeyName(name)
		if !ok {
			t.Errorf("ParseKeyName(%q) = false", name)
			continue
		}
		got := KeyName(code)
		// The reverse map keeps the first name per code; just check code→name→code.
		parsed, ok2 := ParseKeyName(got)
		if !ok2 {
			t.Errorf("KeyName(%d)=%q: ParseKeyName returned false", code, got)
			continue
		}
		if parsed != want {
			t.Errorf("KeyName round-trip for %q: code %d → %q → %d, want %d", name, code, got, parsed, want)
		}
	}
}

func TestKeyNameUnknownCode(t *testing.T) {
	got := KeyName(0)
	if got != "UNKNOWN" {
		t.Errorf("KeyName(0) = %q, want UNKNOWN", got)
	}
}

// ─── SetBinding / ConflictFor ─────────────────────────────────────────────────

func TestSetBindingAndBoundKey(t *testing.T) {
	km := newKeyMap()
	key, ok := ParseKeyName("G")
	if !ok {
		t.Fatal("ParseKeyName(G) failed")
	}
	km.SetBinding(ActionCameraReset, key, 0)
	if got := km.BoundKey(ActionCameraReset); got != key {
		t.Errorf("BoundKey after SetBinding = %d, want %d", got, key)
	}
	if got := km.BoundMods(ActionCameraReset); got != 0 {
		t.Errorf("BoundMods after SetBinding = %d, want 0", got)
	}
}

func TestSetBindingIgnoresActionNone(t *testing.T) {
	km := newKeyMap()
	key, _ := ParseKeyName("G")
	km.SetBinding(ActionNone, key, 0) // should be a no-op
	if got := km.BoundKey(ActionNone); got != 0 {
		t.Errorf("BoundKey(ActionNone) after SetBinding = %d, want 0", got)
	}
}

func TestConflictFor_DetectsConflict(t *testing.T) {
	km := newKeyMap()
	key, _ := ParseKeyName("G")
	km.SetBinding(ActionCameraReset, key, 0)

	conflict, ok := km.ConflictFor(key, 0, ActionCameraYawLeft)
	if !ok {
		t.Fatal("ConflictFor: expected conflict, got none")
	}
	if conflict != ActionCameraReset {
		t.Errorf("ConflictFor = %v, want ActionCameraReset", conflict)
	}
}

func TestConflictFor_ExcludesExcept(t *testing.T) {
	km := newKeyMap()
	key, _ := ParseKeyName("G")
	km.SetBinding(ActionCameraReset, key, 0)

	// Rebinding ActionCameraReset to the same key — must not self-conflict.
	_, ok := km.ConflictFor(key, 0, ActionCameraReset)
	if ok {
		t.Error("ConflictFor should not report self-conflict when except matches")
	}
}

func TestConflictFor_NoConflictOnModsDifference(t *testing.T) {
	km := newKeyMap()
	key, _ := ParseKeyName("W")
	km.SetBinding(ActionThrustForward, key, 0)

	// Same key, different mods — no conflict.
	_, ok := km.ConflictFor(key, ModShift, ActionCameraYawLeft)
	if ok {
		t.Error("ConflictFor: different mods should not be a conflict")
	}
}

// ─── WriteKeybindingsFile round-trip ─────────────────────────────────────────

func TestWriteKeybindingsFile_RoundTrip(t *testing.T) {
	dir := laptopProfileDir(t)
	out := filepath.Join(t.TempDir(), "test_keybindings.json")

	// Load the default keymap.
	km, err := LoadKeyMap(dir, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadKeyMap: %v", err)
	}

	// Write it.
	if err := WriteKeybindingsFile(out, km, "laptop"); err != nil {
		t.Fatalf("WriteKeybindingsFile: %v", err)
	}

	// Load it back.
	km2, err := LoadKeyMap(dir, out)
	if err != nil {
		t.Fatalf("LoadKeyMap (round-trip): %v", err)
	}

	// All bindings must match.
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		if km.BoundKey(a) != km2.BoundKey(a) {
			t.Errorf("action %v key mismatch: original=%d, roundtrip=%d", a, km.BoundKey(a), km2.BoundKey(a))
		}
		if km.BoundMods(a) != km2.BoundMods(a) {
			t.Errorf("action %v mods mismatch: original=%d, roundtrip=%d", a, km.BoundMods(a), km2.BoundMods(a))
		}
	}
}

func TestWriteKeybindingsFile_ModifiedBinding(t *testing.T) {
	dir := laptopProfileDir(t)
	out := filepath.Join(t.TempDir(), "modified.json")

	km, err := LoadKeyMap(dir, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadKeyMap: %v", err)
	}

	// Rebind ActionCameraReset from R to G.
	newKey, _ := ParseKeyName("G")
	km.SetBinding(ActionCameraReset, newKey, 0)

	if err := WriteKeybindingsFile(out, km, "laptop"); err != nil {
		t.Fatalf("WriteKeybindingsFile: %v", err)
	}

	km2, err := LoadKeyMap(dir, out)
	if err != nil {
		t.Fatalf("LoadKeyMap (modified): %v", err)
	}

	if got := km2.BoundKey(ActionCameraReset); got != newKey {
		t.Errorf("after round-trip: ActionCameraReset = %d, want %d (G)", got, newKey)
	}
}

// ─── ScanKeybindingsDir ───────────────────────────────────────────────────────

func TestScanKeybindingsDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanKeybindingsDir(dir)
	if err != nil {
		t.Fatalf("ScanKeybindingsDir error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestScanKeybindingsDir_FindsJsonFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(""), 0644)

	files, err := ScanKeybindingsDir(dir)
	if err != nil {
		t.Fatalf("ScanKeybindingsDir error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 .json files, got %d: %v", len(files), files)
	}
}

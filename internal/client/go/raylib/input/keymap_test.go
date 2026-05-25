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
	// Two camera-context (world-context) actions on the same key — real conflict.
	entries := []bindingEntryJSON{
		{Action: "camera.pitch_up", Key: "W", Mods: []string{}},
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

func TestNoConflictOnMoveContextSameKey(t *testing.T) {
	// move.thrust_down and camera.pitch_down both use DOWN — allowed because
	// move actions are only active in ship mode, camera actions in camera mode.
	entries := []bindingEntryJSON{
		{Action: "camera.pitch_down", Key: "DOWN", Mods: []string{}},
		{Action: "move.thrust_down", Key: "DOWN", Mods: []string{}},
	}
	if err := validateBindings(entries); err != nil {
		t.Errorf("validateBindings: unexpected error for move/camera cross-context DOWN: %v", err)
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
	// Override camera.reset to UP — laptop profile already has camera.pitch_up = UP → conflict.
	cfg := writeKeybindings(t, `{"version":1,"base_profile":"laptop","overrides":[{"action":"camera.reset","key":"UP","mods":[]}]}`)
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

// ─── Phase A: OrderedActions ─────────────────────────────────────────────────

func TestOrderedActionsCount(t *testing.T) {
	// actionCount = 49, so we expect 48 actions (ActionNone excluded).
	want := int(actionCount) - 1
	got := OrderedActions()
	if len(got) != want {
		t.Errorf("OrderedActions len = %d, want %d", len(got), want)
	}
}

func TestOrderedActionsNoDuplicates(t *testing.T) {
	seen := make(map[InputAction]bool)
	for _, a := range OrderedActions() {
		if seen[a] {
			t.Errorf("action %v appears more than once in OrderedActions", a)
		}
		seen[a] = true
	}
}

func TestOrderedActionsNoActionNone(t *testing.T) {
	for _, a := range OrderedActions() {
		if a == ActionNone {
			t.Error("OrderedActions must not contain ActionNone")
		}
	}
}

func TestOrderedActionsContainsAllActions(t *testing.T) {
	ordered := make(map[InputAction]bool)
	for _, a := range OrderedActions() {
		ordered[a] = true
	}
	for a := ActionCameraPitchUp; a < actionCount; a++ {
		if !ordered[a] {
			t.Errorf("action %v (val %d) missing from OrderedActions", a, a)
		}
	}
}

// ─── Phase A: renamed and new action constants ────────────────────────────────

func TestTimescaleActionsExistWithCorrectNames(t *testing.T) {
	cases := []struct {
		action InputAction
		name   string
		val    InputAction
	}{
		{ActionSimTimescaleIncrease, "sim.timescale_increase", 19},
		{ActionSimTimescaleDecrease, "sim.timescale_decrease", 20},
	}
	for _, tc := range cases {
		if tc.action != tc.val {
			t.Errorf("%s value = %d, want %d", tc.name, tc.action, tc.val)
		}
		if got := tc.action.String(); got != tc.name {
			t.Errorf("String() = %q, want %q", got, tc.name)
		}
		parsed, ok := ParseAction(tc.name)
		if !ok {
			t.Errorf("ParseAction(%q) returned false", tc.name)
		} else if parsed != tc.action {
			t.Errorf("ParseAction(%q) = %d, want %d", tc.name, parsed, tc.action)
		}
	}
}

func TestOldSimSpeedNamesDoNotExist(t *testing.T) {
	for _, name := range []string{"sim.speed_increase", "sim.speed_decrease"} {
		if _, ok := ParseAction(name); ok {
			t.Errorf("ParseAction(%q): old name must not exist after rename", name)
		}
	}
}

func TestPhaseANewActionConstants(t *testing.T) {
	cases := []struct {
		action InputAction
		name   string
		val    InputAction
	}{
		{ActionSimTickSpeedIncrease, "sim.tick_speed_increase", 32},
		{ActionSimTickSpeedDecrease, "sim.tick_speed_decrease", 33},
		{ActionSimDatasetIncrease, "sim.dataset_increase", 34},
		{ActionSimDatasetDecrease, "sim.dataset_decrease", 35},
		{ActionUISystemSelector, "ui.system_selector", 36},
		{ActionUILabelCycle, "ui.label_cycle", 37},
		{ActionUIInfraCycle, "ui.infra_cycle", 38},
		{ActionUIMouseModeToggle, "ui.mouse_mode_toggle", 39},
		{ActionUIQuit, "ui.quit", 40},
		{ActionUIRecordToggle, "ui.record_toggle", 41},
		{ActionUIRecordPause, "ui.record_pause", 42},
		{ActionCameraCenter, "camera.center", 43},
		{ActionNavChildNext, "nav.child_next", 44},
		{ActionNavParent, "nav.parent", 45},
		{ActionNavSiblingNext, "nav.sibling_next", 46},
		{ActionNavSiblingPrev, "nav.sibling_prev", 47},
		{ActionNavJump, "nav.jump", 48},
	}
	for _, tc := range cases {
		if tc.action != tc.val {
			t.Errorf("constant value: %s = %d, want %d", tc.name, tc.action, tc.val)
		}
		if got := tc.action.String(); got != tc.name {
			t.Errorf("String(): %s got %q", tc.name, got)
		}
		parsed, ok := ParseAction(tc.name)
		if !ok {
			t.Errorf("ParseAction(%q) returned false", tc.name)
		} else if parsed != tc.action {
			t.Errorf("ParseAction(%q) = %d, want %d", tc.name, parsed, tc.action)
		}
	}
}

func TestActionStringUnknown(t *testing.T) {
	if got := InputAction(999).String(); got != "unknown" {
		t.Errorf("InputAction(999).String() = %q, want \"unknown\"", got)
	}
}

// ─── Phase A: ModSuper ────────────────────────────────────────────────────────

func TestParseModsWithSuper(t *testing.T) {
	ms, err := parseMods([]string{"SUPER", "SHIFT"})
	if err != nil {
		t.Fatalf("parseMods(SUPER, SHIFT) error: %v", err)
	}
	if ms != ModSuper|ModShift {
		t.Errorf("parseMods(SUPER,SHIFT) = %d, want %d", ms, ModSuper|ModShift)
	}
}

func TestParseModsSuperAllFour(t *testing.T) {
	ms, err := parseMods([]string{"SHIFT", "CTRL", "ALT", "SUPER"})
	if err != nil {
		t.Fatalf("parseMods all four error: %v", err)
	}
	if ms != ModShift|ModCtrl|ModAlt|ModSuper {
		t.Errorf("parseMods all four = %d, want %d", ms, ModShift|ModCtrl|ModAlt|ModSuper)
	}
}

func TestModsToStringsSuperRoundTrip(t *testing.T) {
	cases := []ModSet{
		ModSuper,
		ModSuper | ModShift,
		ModSuper | ModAlt,
		ModShift | ModCtrl | ModAlt | ModSuper,
	}
	for _, want := range cases {
		strs := modsToStrings(want)
		got, err := parseMods(strs)
		if err != nil {
			t.Errorf("parseMods(%v) error: %v", strs, err)
			continue
		}
		if got != want {
			t.Errorf("modsToStrings→parseMods round-trip: input=%d, got=%d", want, got)
		}
	}
}

func TestModsToStringsContainsSuper(t *testing.T) {
	strs := modsToStrings(ModSuper)
	if len(strs) != 1 || strs[0] != "SUPER" {
		t.Errorf("modsToStrings(ModSuper) = %v, want [SUPER]", strs)
	}
}

// ─── Phase A: BACKSLASH key ───────────────────────────────────────────────────

func TestParseKeyNameBackslash(t *testing.T) {
	code, ok := ParseKeyName("BACKSLASH")
	if !ok {
		t.Fatal("ParseKeyName(BACKSLASH) returned false — key used in stock profiles")
	}
	if code == 0 {
		t.Error("ParseKeyName(BACKSLASH) returned code 0")
	}
	// Round-trip: the code must map back to a known name.
	if KeyName(code) == "UNKNOWN" {
		t.Errorf("KeyName(%d) = UNKNOWN — reverse lookup broken for BACKSLASH", code)
	}
}

// ─── Phase A: BoundMods with ModSuper ────────────────────────────────────────

func TestBoundModsWithSuper(t *testing.T) {
	km := newKeyMap()
	key, _ := ParseKeyName("S")
	km.SetBinding(ActionUISystemSelector, key, ModSuper)
	if got := km.BoundMods(ActionUISystemSelector); got != ModSuper {
		t.Errorf("BoundMods(UISystemSelector) = %d, want ModSuper (%d)", got, ModSuper)
	}
}

func TestBoundModsOutOfRange(t *testing.T) {
	km := newKeyMap()
	if got := km.BoundMods(ActionNone); got != 0 {
		t.Errorf("BoundMods(ActionNone) = %d, want 0", got)
	}
	if got := km.BoundMods(InputAction(999)); got != 0 {
		t.Errorf("BoundMods(999) = %d, want 0", got)
	}
}

// ─── Phase A: ConflictError.Error ────────────────────────────────────────────

func TestConflictErrorMessageNoMods(t *testing.T) {
	e := &ConflictError{Key: "W", Mods: nil, Action1: "a.one", Action2: "a.two"}
	msg := e.Error()
	if !strings.Contains(msg, "W") || !strings.Contains(msg, "a.one") || !strings.Contains(msg, "a.two") {
		t.Errorf("ConflictError.Error() missing expected content: %q", msg)
	}
}

func TestConflictErrorMessageWithMods(t *testing.T) {
	e := &ConflictError{Key: "S", Mods: []string{"SUPER"}, Action1: "ui.system_selector", Action2: "other"}
	msg := e.Error()
	if !strings.Contains(msg, "SUPER") || !strings.Contains(msg, "S") {
		t.Errorf("ConflictError.Error() with mods = %q, expected SUPER+S", msg)
	}
}

// ─── Phase A: AllKeyNames / KeyNameOf ────────────────────────────────────────

func TestAllKeyNamesContainsBackslash(t *testing.T) {
	names := AllKeyNames()
	if len(names) == 0 {
		t.Fatal("AllKeyNames() returned empty slice")
	}
	found := false
	for _, n := range names {
		if n == "BACKSLASH" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllKeyNames() does not contain BACKSLASH")
	}
}

func TestAllKeyNamesMatchesParseKeyName(t *testing.T) {
	for _, name := range AllKeyNames() {
		if _, ok := ParseKeyName(name); !ok {
			t.Errorf("AllKeyNames() returned %q but ParseKeyName rejected it", name)
		}
	}
}

func TestKeyNameOfBackslash(t *testing.T) {
	code, ok := ParseKeyName("BACKSLASH")
	if !ok {
		t.Fatal("ParseKeyName(BACKSLASH) false")
	}
	got := KeyNameOf(code)
	if got == "" {
		t.Errorf("KeyNameOf(%d) = empty, want non-empty", code)
	}
}

// ─── Phase A: LoadKeyMap branches ────────────────────────────────────────────

func TestLoadKeyMapVersionMismatch(t *testing.T) {
	dir := laptopProfileDir(t)
	cfg := writeKeybindings(t, `{"version":2,"base_profile":"laptop","overrides":[]}`)
	_, err := LoadKeyMap(dir, cfg)
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error %q should mention version", err.Error())
	}
}

func TestLoadKeyMapSuperBindingIntegration(t *testing.T) {
	// The stock laptop profile binds ui.system_selector to S+SUPER.
	// Verify that after loading, BoundMods returns ModSuper.
	dir := laptopProfileDir(t)
	km, err := LoadKeyMap(dir, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadKeyMap error: %v", err)
	}
	mods := km.BoundMods(ActionUISystemSelector)
	if mods&ModSuper == 0 {
		t.Errorf("BoundMods(UISystemSelector) = %d, expected ModSuper bit set", mods)
	}
	key := km.BoundKey(ActionUISystemSelector)
	sKey, _ := ParseKeyName("S")
	if key != sKey {
		t.Errorf("BoundKey(UISystemSelector) = %d, want S (%d)", key, sKey)
	}
}

func TestLoadKeyMapSuperBindingRoundTrip(t *testing.T) {
	dir := laptopProfileDir(t)
	out := filepath.Join(t.TempDir(), "super_roundtrip.json")

	km, err := LoadKeyMap(dir, filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadKeyMap: %v", err)
	}
	if err := WriteKeybindingsFile(out, km, "laptop"); err != nil {
		t.Fatalf("WriteKeybindingsFile: %v", err)
	}
	km2, err := LoadKeyMap(dir, out)
	if err != nil {
		t.Fatalf("LoadKeyMap (round-trip): %v", err)
	}
	if km.BoundMods(ActionUISystemSelector) != km2.BoundMods(ActionUISystemSelector) {
		t.Errorf("ModSuper round-trip failed: original=%d roundtrip=%d",
			km.BoundMods(ActionUISystemSelector), km2.BoundMods(ActionUISystemSelector))
	}
}

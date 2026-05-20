package ship

import (
	"strings"
	"testing"
)

// ─── NewInstance ─────────────────────────────────────────────────────────────

func TestNewInstance_TransponderFormat(t *testing.T) {
	def := minimalDef()
	inst := NewInstance(def, "a3f7c2deadbeef", "Test Pilot")

	// Expect "SC-A3F7C2" (prefix + upper first 6 of session ID).
	want := "SC-A3F7C2"
	if inst.TransponderID != want {
		t.Errorf("TransponderID = %q, want %q", inst.TransponderID, want)
	}
}

func TestNewInstance_TransponderShortSessionID(t *testing.T) {
	def := minimalDef()
	inst := NewInstance(def, "abc", "Pilot")
	if !strings.HasPrefix(inst.TransponderID, "SC-") {
		t.Errorf("TransponderID %q should start with SC-", inst.TransponderID)
	}
	if !strings.Contains(inst.TransponderID, "ABC") {
		t.Errorf("TransponderID %q should contain ABC (upper of abc)", inst.TransponderID)
	}
}

func TestNewInstance_StartsInStage1(t *testing.T) {
	def := minimalDef()
	inst := NewInstance(def, "sess1", "P")
	if inst.ActiveStage != 1 {
		t.Errorf("ActiveStage = %d, want 1", inst.ActiveStage)
	}
}

func TestNewInstance_StartsUndamaged(t *testing.T) {
	def := minimalDef()
	inst := NewInstance(def, "sess1", "P")
	if inst.HullIntegrity != 1.0 {
		t.Errorf("HullIntegrity = %f, want 1.0", inst.HullIntegrity)
	}
	if inst.EngineIntegrity != 1.0 {
		t.Errorf("EngineIntegrity = %f, want 1.0", inst.EngineIntegrity)
	}
	if inst.PowerIntegrity != 1.0 {
		t.Errorf("PowerIntegrity = %f, want 1.0", inst.PowerIntegrity)
	}
}

func TestNewInstance_IdentityFields(t *testing.T) {
	def := minimalDef()
	inst := NewInstance(def, "sess99", "Red Fox")
	if inst.InstanceName != "Red Fox" {
		t.Errorf("InstanceName = %q, want Red Fox", inst.InstanceName)
	}
	if inst.UUID != "sess99" {
		t.Errorf("UUID = %q, want sess99", inst.UUID)
	}
	if inst.DefinitionID != "scout_mk1" {
		t.Errorf("DefinitionID = %q, want scout_mk1", inst.DefinitionID)
	}
	if inst.Definition != def {
		t.Error("Definition pointer should equal the original def")
	}
}

// ─── CurrentStage ─────────────────────────────────────────────────────────────

func TestCurrentStage_DefaultIsStage1(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	stage := inst.CurrentStage()
	if stage.Stage != 1 {
		t.Errorf("CurrentStage().Stage = %d, want 1", stage.Stage)
	}
	if stage.Label != "Maneuvering" {
		t.Errorf("CurrentStage().Label = %q, want Maneuvering", stage.Label)
	}
}

func TestCurrentStage_Stage2(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	inst.ActiveStage = 2
	stage := inst.CurrentStage()
	if stage.Stage != 2 {
		t.Errorf("CurrentStage().Stage = %d, want 2", stage.Stage)
	}
}

func TestCurrentStage_OutOfRangeFallsBackToFirst(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	inst.ActiveStage = 99
	stage := inst.CurrentStage()
	if stage.Stage != 1 {
		t.Errorf("out-of-range ActiveStage should fall back to stage 1, got %d", stage.Stage)
	}
}

// ─── EffectiveAccelMaxMS2 ─────────────────────────────────────────────────────

func TestEffectiveAccelMaxMS2_FullIntegrity(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	want := 1e5
	if got := inst.EffectiveAccelMaxMS2(); got != want {
		t.Errorf("EffectiveAccelMaxMS2 = %g, want %g", got, want)
	}
}

func TestEffectiveAccelMaxMS2_ReducedByDamage(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	inst.EngineIntegrity = 0.5
	want := 1e5 * 0.5
	if got := inst.EffectiveAccelMaxMS2(); got != want {
		t.Errorf("EffectiveAccelMaxMS2 at 50%% = %g, want %g", got, want)
	}
}

// ─── EffectiveTurnRateDegsPerS ────────────────────────────────────────────────

func TestEffectiveTurnRate_FullPower(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	want := 90.0
	if got := inst.EffectiveTurnRateDegsPerS(); got != want {
		t.Errorf("EffectiveTurnRateDegsPerS = %g, want %g", got, want)
	}
}

func TestEffectiveTurnRate_ReducedByPowerDamage(t *testing.T) {
	inst := NewInstance(minimalDef(), "s", "p")
	inst.PowerIntegrity = 0.5
	want := 90.0 * 0.5
	if got := inst.EffectiveTurnRateDegsPerS(); got != want {
		t.Errorf("EffectiveTurnRateDegsPerS at 50%% power = %g, want %g", got, want)
	}
}

// ─── FreePowerW ───────────────────────────────────────────────────────────────

func TestFreePowerW_Stage1(t *testing.T) {
	// available=5e9, baseline=2e8, engine_stage1=5e8, turning=1e8
	// free = 5e9 - 2e8 - 5e8 - 1e8 = 4.2e9
	inst := NewInstance(minimalDef(), "s", "p")
	want := 5e9 - 2e8 - 5e8 - 1e8
	if got := inst.FreePowerW(); got != want {
		t.Errorf("FreePowerW stage1 = %g, want %g", got, want)
	}
}

func TestFreePowerW_Stage2(t *testing.T) {
	// available=5e9, baseline=2e8, engine_stage2=4e9, turning=1e8
	// free = 5e9 - 2e8 - 4e9 - 1e8 = 7e8
	inst := NewInstance(minimalDef(), "s", "p")
	inst.ActiveStage = 2
	want := 5e9 - 2e8 - 4e9 - 1e8
	if got := inst.FreePowerW(); got != want {
		t.Errorf("FreePowerW stage2 = %g, want %g", got, want)
	}
}

func TestFreePowerW_DamagedPower(t *testing.T) {
	// available = 5e9 * 0.5 = 2.5e9; same draws → negative (overload)
	inst := NewInstance(minimalDef(), "s", "p")
	inst.PowerIntegrity = 0.5
	available := 5e9 * 0.5
	want := available - 2e8 - 5e8 - 1e8
	if got := inst.FreePowerW(); got != want {
		t.Errorf("FreePowerW damaged power = %g, want %g", got, want)
	}
}

// ─── buildTransponder ────────────────────────────────────────────────────────

func TestBuildTransponder_Length6(t *testing.T) {
	got := buildTransponder("EX", "abc123xyz")
	if got != "EX-ABC123" {
		t.Errorf("buildTransponder = %q, want EX-ABC123", got)
	}
}

func TestBuildTransponder_ShortID(t *testing.T) {
	got := buildTransponder("FT", "ab")
	if got != "FT-AB" {
		t.Errorf("buildTransponder short = %q, want FT-AB", got)
	}
}

func TestBuildTransponder_ExactlyLength6(t *testing.T) {
	got := buildTransponder("SC", "abcdef")
	if got != "SC-ABCDEF" {
		t.Errorf("buildTransponder exact6 = %q, want SC-ABCDEF", got)
	}
}

package ship

import (
	"fmt"
	"strings"
)

// ShipInstance is the mutable runtime state of one session's vessel.
// It wraps an immutable *ShipDefinition with live fields that change
// during gameplay: velocity, damage, active engine stage, etc.
//
// Phase 1: attached to runtimeSession (direct binary).
// Phase 2+: migrates to ClientSession in the session registry (F-020).
type ShipInstance struct {
	// --- Identity ---
	InstanceName  string // Player-chosen label for this copy of the ship.
	TransponderID string // Server-assigned: "<prefix>-<sessionID[:6].upper>"
	UUID          string // Copied from the owning session ID.

	// --- Definition ---
	DefinitionID string          // Key into ShipCatalog.
	Definition   *ShipDefinition // Immutable after assignment.

	// --- Engine ---
	ActiveStage int // 1-based index into Definition.EngineStages.

	// --- Kinematics (sim units / s) ---
	Velocity       [3]float64 // Current velocity vector.
	MovementVector [3]float64 // Thrust intent this frame (unit vector × throttle).
	FacingVector   [3]float32 // Unit forward vector (ship nose direction).

	// --- Power ---
	CurrentPowerW float64 // Remaining free power this tick.

	// --- Damage (0.0 = destroyed, 1.0 = undamaged) ---
	HullIntegrity    float32
	EngineIntegrity  float32
	PowerIntegrity   float32
	ShieldIntegrity  float32 // Reserved for future shield system.
}

// NewInstance creates a ShipInstance for a session. sessionID is used to
// derive the TransponderID; instanceName is the player-chosen label.
// The instance starts undamaged in engine stage 1.
func NewInstance(def *ShipDefinition, sessionID, instanceName string) *ShipInstance {
	return &ShipInstance{
		InstanceName:  instanceName,
		TransponderID: buildTransponder(def.Identification.TransponderPrefix, sessionID),
		UUID:          sessionID,

		DefinitionID: def.ID,
		Definition:   def,

		ActiveStage: 1,

		CurrentPowerW: def.Power.AvailableW - def.Power.SystemDrawBaselineW,

		HullIntegrity:   1.0,
		EngineIntegrity: 1.0,
		PowerIntegrity:  1.0,
		ShieldIntegrity: 1.0,
	}
}

// CurrentStage returns the active EngineStage for this instance.
// Returns the first stage if ActiveStage is out of range (defensive).
func (s *ShipInstance) CurrentStage() EngineStage {
	idx := s.ActiveStage - 1
	if idx < 0 || idx >= len(s.Definition.EngineStages) {
		return s.Definition.EngineStages[0]
	}
	return s.Definition.EngineStages[idx]
}

// EffectiveAccelMaxMS2 returns the engine's maximum acceleration this tick,
// scaled by engine integrity.
func (s *ShipInstance) EffectiveAccelMaxMS2() float64 {
	return s.CurrentStage().AccelMaxMS2 * float64(s.EngineIntegrity)
}

// EffectiveTurnRateDegsPerS returns the turning rate this tick, scaled by
// power integrity (reduced power reduces turn rate proportionally).
func (s *ShipInstance) EffectiveTurnRateDegsPerS() float64 {
	return s.Definition.Turning.RateDegreesPerS * float64(s.PowerIntegrity)
}

// FreePowerW returns the free (unallocated) power for this tick based on the
// current engine stage draw and turning draw at current power integrity.
func (s *ShipInstance) FreePowerW() float64 {
	available := s.Definition.Power.AvailableW * float64(s.PowerIntegrity)
	baseline := s.Definition.Power.SystemDrawBaselineW
	engineDraw := s.CurrentStage().PowerDrawW
	turningDraw := s.Definition.Turning.PowerDrawW
	return available - baseline - engineDraw - turningDraw
}

// buildTransponder produces "<prefix>-<upper(sessionID[:6])>".
// If sessionID is shorter than 6 characters it is used in full.
func buildTransponder(prefix, sessionID string) string {
	short := sessionID
	if len(short) > 6 {
		short = short[:6]
	}
	return fmt.Sprintf("%s-%s", prefix, strings.ToUpper(short))
}

// Package ship provides the ship definition catalog and per-session ship
// instance for Space Sim. Definitions are loaded from data/ships/*.json at
// startup; instances are created at session registration and carry live
// runtime state such as velocity, active engine stage, and damage levels.
package ship

// ShipClass is the size/role classification of a ship type.
type ShipClass string

const (
	ClassLight   ShipClass = "light"
	ClassMedium  ShipClass = "medium"
	ClassHeavy   ShipClass = "heavy"
	ClassCapital ShipClass = "capital"
)

// OverloadPolicy governs what happens when power demand exceeds supply.
type OverloadPolicy string

const (
	OverloadDegrade  OverloadPolicy = "degrade_non_critical"
	OverloadHardCut  OverloadPolicy = "hard_cut"
)

// ShipDefinition is the immutable, file-loaded description of a ship type.
// All sessions that choose this ship share the same *ShipDefinition value.
type ShipDefinition struct {
	Version     int    `json:"version"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Class       ShipClass `json:"class"`

	Model          ModelRef        `json:"model"`
	Identification IdentificationSpec `json:"identification"`
	EngineStages   []EngineStage   `json:"engine_stages"`
	Turning        TurningSpec     `json:"turning"`
	Power          PowerSpec       `json:"power"`

	MassKg                  float64 `json:"mass_kg"`
	MaxSpeedSimUnitsPerS     float64 `json:"max_speed_sim_units_per_s"`
	SuperluminalAllowed      bool    `json:"superluminal_allowed"`
}

// ModelRef points to the 3D model asset for this ship type.
type ModelRef struct {
	Path    string  `json:"path"`
	Texture string  `json:"texture,omitempty"`
	Scale   float64 `json:"scale"`
}

// IdentificationSpec holds the transponder prefix and registry authority.
type IdentificationSpec struct {
	TransponderPrefix string `json:"transponder_prefix"`
	Registry          string `json:"registry,omitempty"`
}

// EngineStage describes one thrust envelope. Stage 1 is always the
// lowest-power maneuvering mode; higher stages provide greater thrust.
type EngineStage struct {
	Stage       int     `json:"stage"`
	Label       string  `json:"label"`
	AccelMinMS2 float64 `json:"accel_min_ms2"`
	AccelMaxMS2 float64 `json:"accel_max_ms2"`
	PowerDrawW  float64 `json:"power_draw_w"`
}

// TurningSpec describes the angular velocity capability of the ship.
type TurningSpec struct {
	RateDegreesPerS float64 `json:"rate_deg_per_s"`
	PowerDrawW      float64 `json:"power_draw_w"`
}

// PowerSpec describes the ship's power plant and budget policy.
type PowerSpec struct {
	AvailableW          float64        `json:"available_w"`
	SystemDrawBaselineW float64        `json:"system_draw_baseline_w"`
	OverloadPolicy      OverloadPolicy `json:"overload_policy"`
}

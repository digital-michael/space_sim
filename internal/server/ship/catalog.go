package ship

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ShipCatalog is an in-memory index of all loaded ship definitions, keyed by ID.
type ShipCatalog struct {
	ships     map[string]*ShipDefinition
	defaultID string
}

// LoadCatalog scans dir for *.json files, parses each as a ShipDefinition, and
// returns a populated ShipCatalog. Invalid files are logged and skipped; they do
// not prevent startup. defaultID is the ship assigned when a session does not
// specify one. If defaultID is empty or not present in the catalog, the first
// loaded ship is used as the fallback.
func LoadCatalog(dir, defaultID string) (*ShipCatalog, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scanning ship catalog dir %q: %w", dir, err)
	}

	cat := &ShipCatalog{
		ships:     make(map[string]*ShipDefinition, len(matches)),
		defaultID: defaultID,
	}

	for _, path := range matches {
		def, err := loadDefinition(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ship catalog: skipping %s: %v\n", filepath.Base(path), err)
			continue
		}
		cat.ships[def.ID] = def
	}

	// Ensure defaultID is sane.
	if _, ok := cat.ships[cat.defaultID]; !ok {
		// Fall back to first loaded ship by alphabetical match order.
		for _, path := range matches {
			base := strings.TrimSuffix(filepath.Base(path), ".json")
			if _, ok := cat.ships[base]; ok {
				cat.defaultID = base
				break
			}
		}
	}

	return cat, nil
}

// Get returns the ShipDefinition for id, or nil if not found.
func (c *ShipCatalog) Get(id string) *ShipDefinition {
	return c.ships[id]
}

// Default returns the default ShipDefinition, or nil if the catalog is empty.
func (c *ShipCatalog) Default() *ShipDefinition {
	return c.ships[c.defaultID]
}

// DefaultID returns the id of the default ship.
func (c *ShipCatalog) DefaultID() string {
	return c.defaultID
}

// All returns a slice of every loaded ShipDefinition in no guaranteed order.
func (c *ShipCatalog) All() []*ShipDefinition {
	out := make([]*ShipDefinition, 0, len(c.ships))
	for _, def := range c.ships {
		out = append(out, def)
	}
	return out
}

// Len returns the number of ships in the catalog.
func (c *ShipCatalog) Len() int {
	return len(c.ships)
}

// loadDefinition reads and validates a single ship JSON file.
func loadDefinition(path string) (*ShipDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var def ShipDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if err := validateDefinition(path, &def); err != nil {
		return nil, err
	}

	return &def, nil
}

// validateDefinition enforces structural requirements on a loaded definition.
func validateDefinition(path string, def *ShipDefinition) error {
	base := strings.TrimSuffix(filepath.Base(path), ".json")

	if def.Version != 1 {
		return fmt.Errorf("unsupported version %d (want 1)", def.Version)
	}
	if def.ID == "" {
		return fmt.Errorf("missing id")
	}
	if def.ID != base {
		return fmt.Errorf("id %q does not match filename %q", def.ID, base+".json")
	}
	if def.Name == "" {
		return fmt.Errorf("missing name")
	}
	switch def.Class {
	case ClassLight, ClassMedium, ClassHeavy, ClassCapital:
		// valid
	default:
		return fmt.Errorf("unknown class %q", def.Class)
	}
	if len(def.EngineStages) == 0 {
		return fmt.Errorf("engine_stages must not be empty")
	}
	for i, s := range def.EngineStages {
		if s.Stage != i+1 {
			return fmt.Errorf("engine_stages[%d].stage = %d, want %d (must be contiguous from 1)", i, s.Stage, i+1)
		}
		if s.AccelMaxMS2 <= 0 {
			return fmt.Errorf("engine_stages[%d].accel_max_ms2 must be > 0", i)
		}
	}
	if def.Turning.RateDegreesPerS <= 0 {
		return fmt.Errorf("turning.rate_deg_per_s must be > 0")
	}
	if def.Power.AvailableW <= 0 {
		return fmt.Errorf("power.available_w must be > 0")
	}
	switch def.Power.OverloadPolicy {
	case OverloadDegrade, OverloadHardCut:
		// valid
	default:
		return fmt.Errorf("unknown overload_policy %q", def.Power.OverloadPolicy)
	}
	if def.MassKg <= 0 {
		return fmt.Errorf("mass_kg must be > 0")
	}
	if def.Model.Path == "" {
		return fmt.Errorf("model.path must not be empty")
	}
	if def.Model.Scale <= 0 {
		return fmt.Errorf("model.scale must be > 0")
	}
	if def.Identification.TransponderPrefix == "" {
		return fmt.Errorf("identification.transponder_prefix must not be empty")
	}
	return nil
}

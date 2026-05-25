package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
)

const defaultSystemConfigPath = "data/systems/solar_system"

type systemConfigSummary struct {
	Name string `json:"name"`
}

func normalizeSystemConfigPath(path string) string {
	if path == "" {
		path = defaultSystemConfigPath
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(path)
}

func readSystemDisplayName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var summary systemConfigSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return ""
	}

	return strings.TrimSpace(summary.Name)
}

func discoverSystemOptionsFromDir(dir string) ([]ui.SystemOption, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	options := make([]ui.SystemOption, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			// Directory-format system: must contain system.json.
			// Store the directory path so NewWorld calls LoadSystemFromDir.
			manifestPath := filepath.Join(dir, entry.Name(), "system.json")
			if _, err := os.Stat(manifestPath); err != nil {
				continue
			}
			label := entry.Name()
			displayName := readSystemDisplayName(manifestPath)
			if displayName == "" {
				displayName = label
			}
			options = append(options, ui.SystemOption{
				Label:       label,
				DisplayName: displayName,
				Path:        normalizeSystemConfigPath(filepath.Join(dir, entry.Name())),
			})
			continue
		}

		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		configPath := normalizeSystemConfigPath(filepath.Join(dir, entry.Name()))
		displayName := readSystemDisplayName(configPath)
		if displayName == "" {
			displayName = entry.Name()
		}
		options = append(options, ui.SystemOption{
			Label:       entry.Name(),
			DisplayName: displayName,
			Path:        configPath,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		left := strings.ToLower(options[i].DisplayName)
		right := strings.ToLower(options[j].DisplayName)
		if left == right {
			return options[i].Label < options[j].Label
		}
		return left < right
	})

	return options, nil
}

func discoverRuntimeSystemOptions() ([]ui.SystemOption, error) {
	return discoverSystemOptionsFromDir("data/systems")
}

func (a *App) openSystemSelector(inputState *ui.InputState) {
	activePath := normalizeSystemConfigPath(a.cfg.SystemConfig)
	options, err := discoverRuntimeSystemOptions()
	inputState.OpenSystemSelector(options, activePath)

	if err != nil {
		inputState.SetSystemSelectorStatus(fmt.Sprintf("Failed to list system configs: %v", err))
		return
	}
	if len(options) == 0 {
		inputState.SetSystemSelectorStatus("No system JSON files found in data/systems.")
	}
}

// DiscoverSystems returns all discoverable system JSON files from data/systems/.
// Safe to call from any goroutine; performs only filesystem I/O.
func DiscoverSystems() ([]ui.SystemOption, error) {
	return discoverRuntimeSystemOptions()
}

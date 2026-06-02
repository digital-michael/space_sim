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

	var regular, test []ui.SystemOption
	for _, opt := range options {
		if isTestSystem(opt.Label) {
			test = append(test, opt)
		} else {
			regular = append(regular, opt)
		}
	}

	byDisplayName := func(slice []ui.SystemOption) {
		sort.Slice(slice, func(i, j int) bool {
			l := strings.ToLower(slice[i].DisplayName)
			r := strings.ToLower(slice[j].DisplayName)
			if l == r {
				return slice[i].Label < slice[j].Label
			}
			return l < r
		})
	}
	byDisplayName(regular)
	byDisplayName(test)

	if len(regular) > 0 && len(test) > 0 {
		out := make([]ui.SystemOption, 0, len(regular)+1+len(test))
		out = append(out, regular...)
		out = append(out, ui.SystemOption{IsSeparator: true})
		out = append(out, test...)
		return out, nil
	}
	return append(regular, test...), nil
}

// isTestSystem reports whether a system directory or file name is a test system.
// Matches names whose base (extension stripped) ends with "_test".
func isTestSystem(label string) bool {
	base := strings.TrimSuffix(label, filepath.Ext(label))
	return strings.HasSuffix(base, "_test")
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

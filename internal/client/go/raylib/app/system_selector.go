package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
)

const defaultSystemConfigPath = "data/systems/solar_system.json"

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

func discoverSystemOptionsFromDir(dir string) ([]ui.SystemOption, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	options := make([]ui.SystemOption, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}

		configPath := normalizeSystemConfigPath(filepath.Join(dir, entry.Name()))
		options = append(options, ui.SystemOption{
			Label: entry.Name(),
			Path:  configPath,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
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

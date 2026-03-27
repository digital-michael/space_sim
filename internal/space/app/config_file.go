package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// LoadAppConfig loads persisted app/window config. Missing files fall back to defaults.
func LoadAppConfig(path string) (AppConfig, error) {
	cfg := AppConfig{
		Window: WindowConfig{
			Width:      defaultScreenWidth,
			Height:     defaultScreenHeight,
			Fullscreen: false,
			Resizable:  true,
		},
		Render: RenderConfig{
			Mode: RenderModeNative,
		},
	}

	if path == "" {
		path = DefaultAppConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Window.Width <= 0 {
		cfg.Window.Width = defaultScreenWidth
	}
	if cfg.Window.Height <= 0 {
		cfg.Window.Height = defaultScreenHeight
	}
	if !cfg.Window.Resizable {
		cfg.Window.Resizable = true
	}
	if cfg.Render.Mode == "" {
		cfg.Render.Mode = RenderModeNative
	}
	if cfg.Render.Mode == RenderModeFixed {
		if cfg.Render.Width <= 0 {
			cfg.Render.Width = defaultScreenWidth
		}
		if cfg.Render.Height <= 0 {
			cfg.Render.Height = defaultScreenHeight
		}
	}

	return cfg, nil
}

// SaveAppConfig persists the current app/window config to disk atomically.
func SaveAppConfig(path string, cfg AppConfig) error {
	if path == "" {
		path = DefaultAppConfigPath
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

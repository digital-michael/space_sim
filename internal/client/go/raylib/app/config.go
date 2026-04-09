package app

import (
	"fmt"
	"strings"
)

const (
	DefaultAppConfigPath = "configs/app.json"
	defaultScreenWidth   = 1280
	defaultScreenHeight  = 720
	defaultTargetFPS     = 60
	defaultSimHz         = 60.0
)

type RenderMode string

const (
	RenderModeNative RenderMode = "native"
	RenderModeFixed  RenderMode = "fixed"
)

// WindowConfig holds startup window preferences.
type WindowConfig struct {
	Width      int32 `json:"width"`
	Height     int32 `json:"height"`
	Fullscreen bool  `json:"fullscreen"`
	Resizable  bool  `json:"resizable"`
}

// RenderConfig holds the internal render resolution preferences.
type RenderConfig struct {
	Mode   RenderMode `json:"mode"`
	Width  int32      `json:"width"`
	Height int32      `json:"height"`
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Window WindowConfig `json:"window"`
	Render RenderConfig `json:"render"`
}

// Config holds bootstrap options for the Space Sim application.
type Config struct {
	PerformanceMode bool
	Profile         string
	Threads         int
	NoLocking       bool
	SystemConfig    string
	Debug           bool
	AppConfigPath   string
	AppConfig       AppConfig

	// RenderScale, when > 0, multiplies the display size at launch and forces
	// fixed render mode. Mutually exclusive with RenderSize.
	// Recommended: 1.0 (display resolution), 2.0 (2× super-sample).
	RenderScale float64

	// RenderSize, when non-empty, sets an explicit WxH render resolution and
	// forces fixed render mode. Format: "WIDTHxHEIGHT" e.g. "3840x2160".
	// Mutually exclusive with RenderScale.
	RenderSize string
}

// ParseRenderSize parses a "WIDTHxHEIGHT" string into (w, h, error).
func ParseRenderSize(s string) (int32, int32, error) {
	parts := strings.SplitN(strings.ToLower(s), "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("render-size must be WIDTHxHEIGHT (got %q)", s)
	}
	var w, h int32
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("render-size: invalid width %q", parts[0])
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("render-size: invalid height %q", parts[1])
	}
	return w, h, nil
}

// WithDefaults returns cfg with default values filled in.
func (cfg Config) WithDefaults() Config {
	if cfg.AppConfig.Window.Width <= 0 {
		cfg.AppConfig.Window.Width = defaultScreenWidth
	}
	if cfg.AppConfig.Window.Height <= 0 {
		cfg.AppConfig.Window.Height = defaultScreenHeight
	}
	if !cfg.AppConfig.Window.Resizable {
		cfg.AppConfig.Window.Resizable = true
	}

	// CLI render overrides take precedence over app.json render config.
	// RenderScale multiplies the configured window size; RenderSize is explicit.
	// Both force fixed mode.
	switch {
	case cfg.RenderScale > 0:
		w := int32(float64(cfg.AppConfig.Window.Width) * cfg.RenderScale)
		h := int32(float64(cfg.AppConfig.Window.Height) * cfg.RenderScale)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		cfg.AppConfig.Render.Mode = RenderModeFixed
		cfg.AppConfig.Render.Width = w
		cfg.AppConfig.Render.Height = h
	case cfg.RenderSize != "":
		if w, h, err := ParseRenderSize(cfg.RenderSize); err == nil {
			cfg.AppConfig.Render.Mode = RenderModeFixed
			cfg.AppConfig.Render.Width = w
			cfg.AppConfig.Render.Height = h
		}
	}

	if cfg.AppConfig.Render.Mode == "" {
		cfg.AppConfig.Render.Mode = RenderModeNative
	}
	if cfg.AppConfig.Render.Mode == RenderModeFixed {
		if cfg.AppConfig.Render.Width <= 0 {
			cfg.AppConfig.Render.Width = defaultScreenWidth
		}
		if cfg.AppConfig.Render.Height <= 0 {
			cfg.AppConfig.Render.Height = defaultScreenHeight
		}
	}
	if cfg.AppConfigPath == "" {
		cfg.AppConfigPath = DefaultAppConfigPath
	}
	if cfg.PerformanceMode {
		if cfg.Profile == "" {
			cfg.Profile = "worst"
		}
		if cfg.Threads == 0 {
			cfg.Threads = 4
		}
	}
	return cfg
}

// Validate checks for invalid flag/config combinations.
func (cfg Config) Validate() error {
	if !cfg.PerformanceMode {
		if cfg.Profile != "" {
			return fmt.Errorf("--profile can only be used with --performance")
		}
		if cfg.Threads != 0 {
			return fmt.Errorf("--threads can only be used with --performance")
		}
		if cfg.NoLocking {
			return fmt.Errorf("--no-locking can only be used with --performance")
		}
	}
	if cfg.PerformanceMode && (cfg.Threads < 1 || cfg.Threads > 25) {
		return fmt.Errorf("--threads must be between 1 and 25 (got %d)", cfg.Threads)
	}
	if cfg.PerformanceMode && cfg.Profile != "worst" && cfg.Profile != "better" {
		return fmt.Errorf("--profile must be 'worst' or 'better' (got %q)", cfg.Profile)
	}
	if cfg.RenderScale > 0 && cfg.RenderSize != "" {
		return fmt.Errorf("--render-scale and --render-size are mutually exclusive")
	}
	if cfg.RenderScale < 0 {
		return fmt.Errorf("--render-scale must be > 0 (got %g)", cfg.RenderScale)
	}
	if cfg.RenderSize != "" {
		if _, _, err := ParseRenderSize(cfg.RenderSize); err != nil {
			return err
		}
	}
	if cfg.AppConfig.Render.Mode != RenderModeNative && cfg.AppConfig.Render.Mode != RenderModeFixed {
		return fmt.Errorf("render.mode must be %q or %q (got %q)", RenderModeNative, RenderModeFixed, cfg.AppConfig.Render.Mode)
	}
	return nil
}

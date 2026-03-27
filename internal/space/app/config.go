package app

import "fmt"

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
	if cfg.AppConfig.Render.Mode != RenderModeNative && cfg.AppConfig.Render.Mode != RenderModeFixed {
		return fmt.Errorf("render.mode must be %q or %q (got %q)", RenderModeNative, RenderModeFixed, cfg.AppConfig.Render.Mode)
	}
	return nil
}

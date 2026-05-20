package app

import (
	"time"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// RuntimeContext holds mutable runtime UI/window state for the running app.
type RuntimeContext struct {
	ScreenWidth       int32
	ScreenHeight      int32
	WindowedWidth     int32
	WindowedHeight    int32
	RenderWidth       int32
	RenderHeight      int32
	RenderMode        RenderMode
	Fullscreen        bool
	Resizable         bool
	AsteroidDataset   engine.AsteroidDataset
	HUDVisible        bool             // master switch — false hides all HUD categories
	HUD               ui.HUDState      // per-category visibility (only consulted when HUDVisible is true)
	SettingsVisible   bool             // F1 toggle for the unified settings dialog
	Settings          ui.SettingsState // unified settings dialog UI state
	RecordingActive   bool             // true while recording to video
	RecordingPaused   bool             // true while recording is paused (freeze-frame)
	MouseModeEnabled  bool
	LabelMode         ui.LabelMode
	CameraSpeed       float32
	MouseSensitivity  float32
	WelcomeBannerText string
	WelcomeBannerAt   time.Time
	// InfraMode is the infrastructure ambient-light mode.
	// 0 = off (default), 1 = spotlight (FOV-centre ambient boost), 2 = reserved.
	InfraMode int

	// PerfConfig mirrors the active PerformanceOptions so they can be persisted
	// at shutdown via AppConfigSnapshot and restored at startup.
	PerfConfig PerformanceConfig

	// KeybindingsPath is the path of the currently loaded keybindings file.
	// Updated when the user loads a different file in the Configuration tab.
	// Persisted to ~/space-sim.json via AppConfigSnapshot.
	KeybindingsPath string

	// PauseRestoreRate is the SecondsPerSecond value to restore when unpausing
	// via sim.pause_toggle. Zero means "use 1.0 (real-time)".
	PauseRestoreRate float32

	// protectWindowedSize is set for one frame when exiting fullscreen on
	// macOS/X11 to prevent syncWindowState from clobbering WindowedWidth/Height
	// before SetWindowSize's resize event is reflected in GetRenderWidth.
	protectWindowedSize bool
}

// NewRuntimeContext creates the initial runtime state from app config.
func NewRuntimeContext(cfg AppConfig) *RuntimeContext {
	width := cfg.Window.Width
	height := cfg.Window.Height
	if width <= 0 {
		width = defaultScreenWidth
	}
	if height <= 0 {
		height = defaultScreenHeight
	}

	return &RuntimeContext{
		ScreenWidth:     width,
		ScreenHeight:    height,
		WindowedWidth:   width,
		WindowedHeight:  height,
		RenderWidth:     cfg.Render.Width,
		RenderHeight:    cfg.Render.Height,
		RenderMode:      cfg.Render.Mode,
		Fullscreen:      cfg.Window.Fullscreen,
		Resizable:       cfg.Window.Resizable,
		AsteroidDataset: engine.AsteroidDatasetSmall,
		HUDVisible:      true,
		HUD:             ui.AllOnHUD(),
		SettingsVisible: false,
		Settings: ui.SettingsState{
			ActiveTab:   0,
			SelectedRow: 0,
			HUD:         ui.AllOnHUD(),
			Perf: ui.PerformanceOptions{
				FrustumCulling:      cfg.Performance.FrustumCulling,
				LODEnabled:          cfg.Performance.LODEnabled,
				InstancedRendering:  cfg.Performance.InstancedRendering,
				SpatialPartition:    cfg.Performance.SpatialPartition,
				PointRendering:      cfg.Performance.PointRendering,
				ImportanceThreshold: cfg.Performance.ImportanceThreshold,
				UseInPlaceSwap:      cfg.Performance.UseInPlaceSwap,
			},
			KeybindCapture: -1,
		},
		MouseModeEnabled: true,
		LabelMode:        ui.LabelModeOff,
		CameraSpeed:      10.0,
		MouseSensitivity: 0.003,
		PerfConfig:       cfg.Performance,
	}
}

// UpdateScreenSize records the current live screen size.
func (ctx *RuntimeContext) UpdateScreenSize(width, height int32) {
	ctx.ScreenWidth = width
	ctx.ScreenHeight = height
}

// SetRenderSize records the current internal render resolution.
func (ctx *RuntimeContext) SetRenderSize(width, height int32) {
	ctx.RenderWidth = width
	ctx.RenderHeight = height
}

// SyncWindowState records live window dimensions and fullscreen state.
func (ctx *RuntimeContext) SyncWindowState(width, height int32, fullscreen bool) {
	ctx.ScreenWidth = width
	ctx.ScreenHeight = height
	ctx.Fullscreen = fullscreen
	if !fullscreen {
		if ctx.protectWindowedSize {
			// Consume the one-frame guard set during fullscreen exit.
			// GetRenderWidth may still reflect the monitor size while the
			// OS processes the SetWindowSize call asynchronously.
			ctx.protectWindowedSize = false
		} else {
			ctx.WindowedWidth = width
			ctx.WindowedHeight = height
		}
	}
}

// EffectiveScreenSize returns the live dimensions tracked by the app.
func (ctx *RuntimeContext) EffectiveScreenSize() (int32, int32) {
	return ctx.ScreenWidth, ctx.ScreenHeight
}

// AppConfigSnapshot captures the current runtime window state in config form.
func (ctx *RuntimeContext) AppConfigSnapshot() AppConfig {
	width := ctx.WindowedWidth
	height := ctx.WindowedHeight
	if width <= 0 {
		width = ctx.ScreenWidth
	}
	if height <= 0 {
		height = ctx.ScreenHeight
	}

	return AppConfig{
		Window: WindowConfig{
			Width:      width,
			Height:     height,
			Fullscreen: ctx.Fullscreen,
			Resizable:  ctx.Resizable,
		},
		Render: RenderConfig{
			Mode:   ctx.RenderMode,
			Width:  ctx.RenderWidth,
			Height: ctx.RenderHeight,
		},
		Performance:     ctx.PerfConfig,
		KeybindingsPath: ctx.KeybindingsPath,
	}
}

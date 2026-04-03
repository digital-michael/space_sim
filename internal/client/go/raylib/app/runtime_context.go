package app

import "github.com/digital-michael/space_sim/internal/sim/engine"

// RuntimeContext holds mutable runtime UI/window state for the running app.
type RuntimeContext struct {
	ScreenWidth      int32
	ScreenHeight     int32
	WindowedWidth    int32
	WindowedHeight   int32
	RenderWidth      int32
	RenderHeight     int32
	RenderMode       RenderMode
	Fullscreen       bool
	Resizable        bool
	GridVisible      bool
	AsteroidDataset  engine.AsteroidDataset
	HUDVisible       bool
	HelpVisible      bool
	MouseModeEnabled bool
	LabelsVisible    bool
	CameraSpeed      float32
	MouseSensitivity float32
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
		ScreenWidth:      width,
		ScreenHeight:     height,
		WindowedWidth:    width,
		WindowedHeight:   height,
		RenderWidth:      cfg.Render.Width,
		RenderHeight:     cfg.Render.Height,
		RenderMode:       cfg.Render.Mode,
		Fullscreen:       cfg.Window.Fullscreen,
		Resizable:        cfg.Window.Resizable,
		GridVisible:      false,
		AsteroidDataset:  engine.AsteroidDatasetSmall,
		HUDVisible:       true,
		HelpVisible:      false,
		MouseModeEnabled: true,
		LabelsVisible:    false,
		CameraSpeed:      10.0,
		MouseSensitivity: 0.003,
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
		ctx.WindowedWidth = width
		ctx.WindowedHeight = height
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
	}
}

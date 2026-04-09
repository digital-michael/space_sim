package app

import (
	"log"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const appWindowTitle = "Space Sim Smoke Test - Solar System"

func (a *App) initWindow() {
	flags := uint32(rl.FlagWindowHighdpi)
	if a.runtime.Resizable {
		flags |= rl.FlagWindowResizable
	}
	if !a.cfg.NoMSAA {
		flags |= rl.FlagMsaa4xHint
	}
	rl.SetConfigFlags(flags)

	rl.InitWindow(a.runtime.ScreenWidth, a.runtime.ScreenHeight, appWindowTitle)
	if a.runtime.Fullscreen {
		a.toggleFullscreen()
	}

	rl.SetTargetFPS(defaultTargetFPS)
	rl.SetExitKey(0)
	if a.runtime.MouseModeEnabled {
		rl.DisableCursor()
	}

	a.syncWindowState()
}

// toggleFullscreen switches between windowed and fullscreen modes using the
// current monitor resolution for fullscreen and restores previous windowed size.
func (a *App) toggleFullscreen() {
	if !rl.IsWindowFullscreen() {
		monitor := rl.GetCurrentMonitor()
		monitorWidth := int32(rl.GetMonitorWidth(monitor))
		monitorHeight := int32(rl.GetMonitorHeight(monitor))
		if monitorWidth > 0 && monitorHeight > 0 {
			rl.SetWindowSize(int(monitorWidth), int(monitorHeight))
		}
		rl.ToggleFullscreen()
	} else {
		rl.ToggleFullscreen()
		windowedWidth := a.runtime.WindowedWidth
		windowedHeight := a.runtime.WindowedHeight
		if windowedWidth <= 0 {
			windowedWidth = defaultScreenWidth
		}
		if windowedHeight <= 0 {
			windowedHeight = defaultScreenHeight
		}
		rl.SetWindowSize(int(windowedWidth), int(windowedHeight))

		monitor := rl.GetCurrentMonitor()
		monitorWidth := rl.GetMonitorWidth(monitor)
		monitorHeight := rl.GetMonitorHeight(monitor)
		if monitorWidth > 0 && monitorHeight > 0 {
			posX := (monitorWidth - int(windowedWidth)) / 2
			posY := (monitorHeight - int(windowedHeight)) / 2
			rl.SetWindowPosition(posX, posY)
		}
	}
	a.syncWindowState()
	a.syncRenderState()
}

func (a *App) closeWindow() {
	a.renderer.Close()
	a.persistWindowConfig()
	rl.CloseWindow()
}

func (a *App) syncWindowState() {
	a.runtime.SyncWindowState(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.IsWindowFullscreen())
}

func (a *App) persistWindowConfig() {
	a.syncWindowState()
	snap := a.runtime.AppConfigSnapshot()
	// Always persist the render config as originally loaded from app.json,
	// not the runtime state. CLI flags (--render-scale, --render-size) and
	// transient recording mode switches must not soil the saved config.
	snap.Render = a.savedRenderConfig
	if err := SaveAppConfig(a.cfg.AppConfigPath, snap); err != nil {
		log.Printf("Warning: could not save app config: %v", err)
	}
}

package app

import (
	"errors"
	"log"
	"math"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const appWindowTitle = "Space Sim Smoke Test - Solar System"

func (a *App) initWindow() error {
	flags := uint32(rl.FlagWindowHighdpi)
	if a.runtime.Resizable {
		flags |= rl.FlagWindowResizable
	}
	if !a.cfg.NoMSAA {
		// EGL (used by GLFW's Wayland backend) commonly fails to allocate MSAA
		// framebuffers on some compositors. When both X11 and Wayland are available
		// and GLFW_PLATFORM is not already set, prefer X11/GLX so MSAA works reliably.
		// If GLFW_PLATFORM is already set to a non-x11 value (e.g. "wayland"), we are
		// committed to EGL and requesting MSAA would crash; skip the hint in that case.
		_, hasWayland := os.LookupEnv("WAYLAND_DISPLAY")
		_, hasX11 := os.LookupEnv("DISPLAY")
		useMSAA := true
		if hasWayland {
			glfwPlatform := os.Getenv("GLFW_PLATFORM")
			switch {
			case glfwPlatform == "" && hasX11:
				os.Setenv("GLFW_PLATFORM", "x11")
				log.Println("initWindow: preferring X11/GLX over Wayland/EGL for MSAA compatibility")
			case glfwPlatform == "x11":
				// user already forced X11 — MSAA is safe
			default:
				// Wayland/EGL backend is active; MSAA allocation commonly fails
				log.Println("initWindow: MSAA disabled on Wayland/EGL (set GLFW_PLATFORM=x11 or pass --no-msaa to suppress)")
				useMSAA = false
			}
		}
		if useMSAA {
			flags |= rl.FlagMsaa4xHint
		}
	}
	rl.SetConfigFlags(flags)

	rl.InitWindow(a.runtime.ScreenWidth, a.runtime.ScreenHeight, appWindowTitle)
	if !rl.IsWindowReady() {
		return errors.New("window initialization failed: graphics context unavailable (check display or GPU driver)")
	}
	if a.runtime.Fullscreen {
		a.toggleFullscreen()
	}

	rl.SetTargetFPS(defaultTargetFPS)
	rl.SetExitKey(0)
	if a.runtime.MouseModeEnabled {
		rl.DisableCursor()
	}

	a.syncWindowState()
	return nil
}

// onWaylandCompositor reports whether the process is running under a Wayland
// compositor — either the native GLFW Wayland backend or XWayland (X11 GLFW
// backend running inside a Wayland session). Both cases share the same
// behaviour requirements:
//   - Do NOT pre-size the window before ToggleFullscreen: the compositor owns
//     fullscreen geometry and overrides any client-side hint anyway, but the
//     hint actively disrupts input delivery in fullscreen under XWayland.
//   - Do NOT call SetWindowPosition: unsupported on both backends under Wayland.
func onWaylandCompositor() bool {
	_, hasWayland := os.LookupEnv("WAYLAND_DISPLAY")
	return hasWayland
}

// toggleFullscreen switches between windowed and fullscreen modes.
// syncWindowState and syncRenderState are NOT called here: the init path calls
// them explicitly after this function returns, and the interactive loop calls
// them at the top of every frame. Calling them mid-frame (after input but
// before rendering) would corrupt the projection matrix for the current frame.
func (a *App) toggleFullscreen() {
	wayland := onWaylandCompositor()
	if !rl.IsWindowFullscreen() {
		if !wayland {
			// X11/GLX and macOS: pre-size to monitor resolution before toggling.
			// On Wayland (native or XWayland) the compositor owns fullscreen
			// geometry; SetWindowSize here conflicts with the compositor and
			// disrupts input delivery in fullscreen.
			monitor := rl.GetCurrentMonitor()
			monitorWidth := int32(rl.GetMonitorWidth(monitor))
			monitorHeight := int32(rl.GetMonitorHeight(monitor))
			if monitorWidth > 0 && monitorHeight > 0 {
				rl.SetWindowSize(int(monitorWidth), int(monitorHeight))
			}
		}
		// Guard against syncWindowState clobbering WindowedWidth/Height on the
		// first frame after ToggleFullscreen, when IsWindowFullscreen may still
		// report false while GetRenderWidth already reflects the monitor size.
		a.runtime.protectWindowedSize = true
		rl.ToggleFullscreen()
		// On Wayland, xdg_toplevel_set_fullscreen is sent asynchronously.
		// The compositor responds with xdg_toplevel_configure + xdg_surface_configure
		// which must be ack'd and followed by a correctly-sized EGL buffer before
		// the compositor will present any fullscreen frame.
		// PollInputEvents normally calls glfwPollEvents (poll timeout=0, non-blocking).
		// If the configure event hasn't arrived on the socket yet, poll returns
		// immediately with nothing processed — EGL surface stays at windowed size,
		// ack_configure is never sent, and ALL fullscreen frames are silently dropped.
		// EnableEventWaiting switches PollInputEvents to glfwWaitEvents (poll timeout=-1),
		// which blocks until the compositor writes the configure event to the socket.
		// This guarantees ack_configure is sent and the EGL surface is resized to the
		// monitor resolution before the next BeginDrawing commits a buffer.
		if wayland {
			rl.EnableEventWaiting()
			rl.PollInputEvents()
			rl.DisableEventWaiting()
		}
	} else {
		rl.ToggleFullscreen()
		if wayland {
			// Same for exit: wait for the windowed-restore configure event so the
			// EGL surface is back at windowed size before the next frame commits.
			rl.EnableEventWaiting()
			rl.PollInputEvents()
			rl.DisableEventWaiting()
		}
		if !wayland {
			// On Wayland, the compositor restores the pre-fullscreen window
			// geometry asynchronously after the toggle. Calling SetWindowSize
			// here races the compositor and produces wrong aspect ratios on
			// the frames immediately following the exit.
			// On X11 and macOS, explicit resize is required to restore size.
			windowedWidth := a.runtime.WindowedWidth
			windowedHeight := a.runtime.WindowedHeight
			if windowedWidth <= 0 {
				windowedWidth = defaultScreenWidth
			}
			if windowedHeight <= 0 {
				windowedHeight = defaultScreenHeight
			}
			// Guard against syncWindowState clobbering WindowedWidth/Height on
			// the next frame while GetRenderWidth may still reflect the monitor
			// size before SetWindowSize's resize event is committed by the OS.
			a.runtime.protectWindowedSize = true
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
	}
}

func (a *App) closeWindow() {
	a.renderer.Close()
	a.persistWindowConfig()
	rl.CloseWindow()
}

func (a *App) syncWindowState() {
	// GetRenderWidth/Height (CORE.Window.render.width/height) is used instead of
	// GetScreenWidth/Height (CORE.Window.screen.width/height) because raylib's
	// WindowSizeCallback skips updating screen.width/height when
	// IsWindowFullscreen() is true — so GetScreenWidth returns the pre-fullscreen
	// windowed size for the entire fullscreen session.  render.width/height IS
	// correctly updated via SetupViewport on every resize, including fullscreen.
	w := int32(rl.GetRenderWidth())
	h := int32(rl.GetRenderHeight())
	// Divide by the window content scale to convert physical (HiDPI) framebuffer
	// pixels to logical pixels. GetRenderWidth returns physical pixels on every
	// platform with a HiDPI window (macOS Retina, Linux HiDPI, Windows DPI-aware).
	// SetWindowSize and all config values use logical pixels; without this
	// conversion WindowedWidth doubles on each fullscreen round-trip on Retina.
	dpi := rl.GetWindowScaleDPI()
	w, h = scaledDimensions(w, h, float64(dpi.X), float64(dpi.Y))
	a.runtime.SyncWindowState(w, h, rl.IsWindowFullscreen())
}

// scaledDimensions converts physical (HiDPI) pixel dimensions to logical pixels
// by dividing by the window content scale. scaleX and scaleY come from
// rl.GetWindowScaleDPI(). On standard 1× displays the scale is 1.0 and the
// values pass through unchanged.
func scaledDimensions(w, h int32, scaleX, scaleY float64) (int32, int32) {
	if scaleX > 1 {
		w = int32(math.Round(float64(w) / scaleX))
	}
	if scaleY > 1 {
		h = int32(math.Round(float64(h) / scaleY))
	}
	return w, h
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

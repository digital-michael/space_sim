package app

import rl "github.com/gen2brain/raylib-go/raylib"

func (a *App) syncRenderState() {
	desiredWidth, desiredHeight := a.desiredRenderSize()
	if desiredWidth <= 0 || desiredHeight <= 0 {
		desiredWidth = a.runtime.ScreenWidth
		desiredHeight = a.runtime.ScreenHeight
	}

	if a.runtime.RenderMode == RenderModeNative {
		a.runtime.SetRenderSize(desiredWidth, desiredHeight)
		a.renderer.DisableRenderTarget()
		return
	}

	if desiredWidth == a.runtime.RenderWidth && desiredHeight == a.runtime.RenderHeight && a.renderer.HasRenderTarget() {
		return
	}

	a.runtime.SetRenderSize(desiredWidth, desiredHeight)
	a.renderer.ConfigureRenderTarget(desiredWidth, desiredHeight)
}

func (a *App) desiredRenderSize() (int32, int32) {
	if a.runtime.RenderMode == RenderModeFixed {
		if a.runtime.RenderWidth > 0 && a.runtime.RenderHeight > 0 {
			return a.runtime.RenderWidth, a.runtime.RenderHeight
		}
		return defaultScreenWidth, defaultScreenHeight
	}

	width := int32(rl.GetScreenWidth())
	height := int32(rl.GetScreenHeight())
	if width > 0 && height > 0 {
		return width, height
	}

	return a.runtime.ScreenWidth, a.runtime.ScreenHeight
}

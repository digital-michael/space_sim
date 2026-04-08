// Package render contains the Raylib-specific drawing implementation for Space
// Sim.
//
// It owns frame setup, render targets, object and label drawing, HUD and panel
// rendering, and other presentation concerns that depend directly on Raylib.
// Higher-level application logic should flow into this package through the app,
// engine, and ui abstractions rather than embedding draw logic elsewhere.
//
// # Render-Target Contract
//
// All drawing must occur between [Renderer.BeginFrame] and [Renderer.EndFrame].
// BeginFrame enters a Raylib render texture mode (BeginTextureMode); EndFrame
// exits it and blits the texture to the screen.
//
// Any draw call made outside that window — directly to the default framebuffer
// rather than to the render texture — will be visible on screen but WILL NOT
// be captured by [Renderer.CaptureRenderTexture] and therefore will be missing
// from video recordings and any future screenshot/export features.
//
// When render mode is "native" (no RenderTexture2D), BeginFrame falls back to
// a plain BeginDrawing and the capture path is unavailable. The app layer
// switches to "fixed" mode automatically when recording is started.
package render
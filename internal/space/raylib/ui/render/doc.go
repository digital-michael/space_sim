// Package render contains the Raylib-specific drawing implementation for Space
// Sim.
//
// It owns frame setup, render targets, object and label drawing, HUD and panel
// rendering, and other presentation concerns that depend directly on Raylib.
// Higher-level application logic should flow into this package through the app,
// engine, and ui abstractions rather than embedding draw logic elsewhere.
package render
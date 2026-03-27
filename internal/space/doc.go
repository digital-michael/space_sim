// Package space provides the Space Sim application layer built on top of the
// renderer-agnostic engine package.
//
// It owns the JSON configuration model, system and template loading pipeline,
// SOL-specific object construction, belt generation, and the thin simulation
// wrapper used by the application runtime. This package contains no Raylib
// types and is intended to remain testable without the rendering layer.
package space

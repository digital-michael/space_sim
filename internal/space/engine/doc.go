// Package engine implements the renderer-agnostic simulation kernel for Space
// Sim.
//
// The package contains the core object model, math primitives, feature schema,
// simulation state and double-buffering infrastructure, and the concurrent
// orbital physics loop. It is designed to depend only on the standard library
// so higher layers can reuse it without pulling in Raylib-specific concerns.
package engine
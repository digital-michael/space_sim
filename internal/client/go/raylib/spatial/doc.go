// Package spatial provides Raylib-backed culling and spatial partitioning
// helpers for Space Sim rendering.
//
// It bridges engine object data and ui camera state to frustum checks and a
// simple spatial hash grid so the renderer can reduce the number of objects it
// considers each frame.
package spatial
// Package engine provides the generic simulation engine for Space Sim.
// It has no knowledge of specific datasets (e.g. solar system) and no Raylib
// dependency — it is independently testable and reusable.
package engine

import "math"

// Vector3 represents a 3D position or direction.
type Vector3 struct {
	X, Y, Z float32
}

// Add returns the sum of two vectors.
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Sub returns the difference of two vectors.
func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Scale multiplies the vector by a scalar.
func (v Vector3) Scale(s float32) Vector3 {
	return Vector3{v.X * s, v.Y * s, v.Z * s}
}

// Length returns the magnitude of the vector.
func (v Vector3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

// Normalize returns a unit vector in the same direction.
func (v Vector3) Normalize() Vector3 {
	l := v.Length()
	if l == 0 {
		return Vector3{}
	}
	return v.Scale(1.0 / l)
}

// Dot returns the dot product of two vectors.
func (v Vector3) Dot(other Vector3) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross returns the cross product of two vectors.
func (v Vector3) Cross(other Vector3) Vector3 {
	return Vector3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Color represents an RGBA color (0-255 range, Raylib-compatible).
type Color struct {
	R, G, B, A uint8
}

// Predefined colors.
var (
	ColorYellow = Color{255, 255, 0, 255}
	ColorBlue   = Color{0, 100, 255, 255}
	ColorRed    = Color{200, 50, 50, 255}
	ColorOrange = Color{255, 140, 0, 255}
	ColorGray   = Color{128, 128, 128, 255}
	ColorWhite  = Color{255, 255, 255, 255}
	ColorCyan   = Color{0, 200, 200, 255}
)

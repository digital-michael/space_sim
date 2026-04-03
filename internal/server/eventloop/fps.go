package eventloop

import (
"sync"
"time"
)

// FPSController manages frame timing and FPS configuration.
type FPSController struct {
mu              sync.RWMutex
targetFPS       float64
frameDuration   time.Duration
deltaTime       float64
lastUpdateTime  time.Time
changed         bool
}

// NewFPSController creates an FPS controller with the target FPS.
func NewFPSController(targetFPS float64) *FPSController {
if targetFPS <= 0 {
targetFPS = 60.0
}
if targetFPS > 1000 {
targetFPS = 1000
}

fc := &FPSController{
targetFPS:      targetFPS,
lastUpdateTime: time.Now(),
}
fc.updateFrameDuration()
return fc
}

// updateFrameDuration recalculates frame duration from target FPS.
func (fc *FPSController) updateFrameDuration() {
fc.frameDuration = time.Duration(float64(time.Second) / fc.targetFPS)
fc.deltaTime = 1.0 / fc.targetFPS
}

// SetFPS changes the target FPS at runtime.
func (fc *FPSController) SetFPS(fps float64) {
fc.mu.Lock()
defer fc.mu.Unlock()

if fps <= 0 || fps > 1000 {
return // Ignore invalid FPS
}

if fps != fc.targetFPS {
fc.targetFPS = fps
fc.updateFrameDuration()
fc.changed = true
}
}

// FrameDuration returns the time.Duration per frame.
func (fc *FPSController) FrameDuration() time.Duration {
fc.mu.RLock()
defer fc.mu.RUnlock()
return fc.frameDuration
}

// DeltaTime returns the delta time per frame in seconds.
func (fc *FPSController) DeltaTime() float64 {
fc.mu.RLock()
defer fc.mu.RUnlock()
return fc.deltaTime
}

// GetTargetFPS returns the target FPS.
func (fc *FPSController) GetTargetFPS() float64 {
fc.mu.RLock()
defer fc.mu.RUnlock()
return fc.targetFPS
}

// HasChanged checks if the FPS has changed since last call.
func (fc *FPSController) HasChanged() bool {
fc.mu.Lock()
defer fc.mu.Unlock()

if fc.changed {
fc.changed = false
return true
}
return false
}

package eventloop

import (
"sync"
"time"
)

// WorkerMetrics tracks performance metrics for a single worker.
type WorkerMetrics struct {
mu              sync.RWMutex
workerID        int
eventsProcessed uint64
totalProcessTime time.Duration
errors          uint64
lastEventTime   time.Time
}

// NewWorkerMetrics creates a new worker metrics tracker.
func NewWorkerMetrics(workerID int) *WorkerMetrics {
return &WorkerMetrics{
workerID: workerID,
}
}

// RecordEvent records processing time for a single event.
func (m *WorkerMetrics) RecordEvent(duration time.Duration) {
m.mu.Lock()
defer m.mu.Unlock()
m.eventsProcessed++
m.totalProcessTime += duration
m.lastEventTime = time.Now()
}

// RecordError records an error during event processing.
func (m *WorkerMetrics) RecordError(err error) {
m.mu.Lock()
defer m.mu.Unlock()
m.errors++
}

// GetStats returns a snapshot of worker metrics.
func (m *WorkerMetrics) GetStats() WorkerStats {
m.mu.RLock()
defer m.mu.RUnlock()

avgTime := time.Duration(0)
if m.eventsProcessed > 0 {
avgTime = time.Duration(int64(m.totalProcessTime) / int64(m.eventsProcessed))
}

return WorkerStats{
WorkerID:        m.workerID,
EventsProcessed: m.eventsProcessed,
AverageTimePerEvent: avgTime,
Errors:          m.errors,
LastEventTime:   m.lastEventTime,
}
}

// WorkerStats represents a snapshot of worker statistics.
type WorkerStats struct {
WorkerID           int
EventsProcessed    uint64
AverageTimePerEvent time.Duration
Errors             uint64
LastEventTime      time.Time
}

// LoopMetrics tracks performance metrics for the event loop.
type LoopMetrics struct {
mu                sync.RWMutex
framesProcessed   uint64
frameErrors       uint64
totalFrameTime    time.Duration
minFrameTime      time.Duration
maxFrameTime      time.Duration
lastFrameTime     time.Duration
startTime         time.Time
}

// NewLoopMetrics creates a new event loop metrics tracker.
func NewLoopMetrics() *LoopMetrics {
return &LoopMetrics{
startTime:   time.Now(),
minFrameTime: time.Hour, // Initialize to large value
}
}

// RecordFrameSuccess records successful frame processing.
func (m *LoopMetrics) RecordFrameSuccess() {
m.mu.Lock()
defer m.mu.Unlock()
m.framesProcessed++
}

// RecordFrameError records a frame processing error.
func (m *LoopMetrics) RecordFrameError() {
m.mu.Lock()
defer m.mu.Unlock()
m.frameErrors++
}

// RecordFrameTime records the duration of a single frame.
func (m *LoopMetrics) RecordFrameTime(duration time.Duration) {
m.mu.Lock()
defer m.mu.Unlock()

m.lastFrameTime = duration
m.totalFrameTime += duration

if duration < m.minFrameTime {
m.minFrameTime = duration
}
if duration > m.maxFrameTime {
m.maxFrameTime = duration
}
}

// GetActualFPS calculates the actual FPS based on recorded frame times.
func (m *LoopMetrics) GetActualFPS() float64 {
m.mu.RLock()
defer m.mu.RUnlock()

if m.framesProcessed == 0 {
return 0
}

avgFrameTime := time.Duration(int64(m.totalFrameTime) / int64(m.framesProcessed))
if avgFrameTime == 0 {
return 0
}

return float64(time.Second) / float64(avgFrameTime)
}

// GetStats returns a snapshot of loop metrics.
func (m *LoopMetrics) GetStats() LoopStats {
m.mu.RLock()
defer m.mu.RUnlock()

uptime := time.Since(m.startTime)
avgFrameTime := time.Duration(0)
if m.framesProcessed > 0 {
avgFrameTime = time.Duration(int64(m.totalFrameTime) / int64(m.framesProcessed))
}

return LoopStats{
FramesProcessed: m.framesProcessed,
FrameErrors:     m.frameErrors,
AverageFrameTime: avgFrameTime,
MinFrameTime:    m.minFrameTime,
MaxFrameTime:    m.maxFrameTime,
LastFrameTime:   m.lastFrameTime,
Uptime:          uptime,
}
}

// LoopStats represents a snapshot of event loop statistics.
type LoopStats struct {
FramesProcessed  uint64
FrameErrors      uint64
AverageFrameTime time.Duration
MinFrameTime     time.Duration
MaxFrameTime     time.Duration
LastFrameTime    time.Duration
Uptime           time.Duration
}

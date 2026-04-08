package app

import (
	"bytes"
	"io"
	"testing"
)

// nopCloser wraps a bytes.Buffer so it satisfies io.WriteCloser without needing
// a real file or process pipe.
type nopCloser struct {
	*bytes.Buffer
	closed bool
}

func (n *nopCloser) Close() error {
	n.closed = true
	return nil
}

func newTestRecorder(w, h int) (*videoRecorder, *nopCloser) {
	buf := &nopCloser{Buffer: &bytes.Buffer{}}
	return newVideoRecorder(buf, w, h, "/tmp/test.mp4"), buf
}

// TestWriteFrameStoresLastFrame verifies that the first non-nil frame is
// written to the pipe and stored as lastFrame for freeze use.
func TestWriteFrameStoresLastFrame(t *testing.T) {
	rec, buf := newTestRecorder(2, 2)

	frame := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16} // 2×2 RGBA
	if err := rec.WriteFrame(frame); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}

	if !bytes.Equal(buf.Bytes(), frame) {
		t.Errorf("pipe bytes = %v, want %v", buf.Bytes(), frame)
	}
	if !bytes.Equal(rec.lastFrame, frame) {
		t.Errorf("lastFrame = %v, want %v", rec.lastFrame, frame)
	}
}

// TestWriteFrameNilFreezes verifies that passing nil re-sends the last frame
// (freeze-frame pause semantics).
func TestWriteFrameNilFreezes(t *testing.T) {
	rec, buf := newTestRecorder(2, 2)

	frame := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	_ = rec.WriteFrame(frame)
	buf.Reset() // clear the first write

	// nil → should repeat lastFrame
	if err := rec.WriteFrame(nil); err != nil {
		t.Fatalf("WriteFrame(nil) error = %v", err)
	}
	if !bytes.Equal(buf.Bytes(), frame) {
		t.Errorf("freeze: pipe bytes = %v, want %v", buf.Bytes(), frame)
	}
	// lastFrame must not change
	if !bytes.Equal(rec.lastFrame, frame) {
		t.Errorf("freeze: lastFrame changed unexpectedly")
	}
}

// TestWriteFrameNilBeforeAnyFrame verifies that nil before any frame is a no-op
// (nothing is written and no panic occurs).
func TestWriteFrameNilBeforeAnyFrame(t *testing.T) {
	rec, buf := newTestRecorder(2, 2)

	if err := rec.WriteFrame(nil); err != nil {
		t.Fatalf("WriteFrame(nil) on empty recorder error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no bytes written, got %d", buf.Len())
	}
}

// TestWriteFrameUpdatesLastFrame verifies that each non-nil call replaces
// lastFrame so the most recent frame is what gets frozen on pause.
func TestWriteFrameUpdatesLastFrame(t *testing.T) {
	rec, _ := newTestRecorder(1, 1)

	first := []byte{1, 2, 3, 4}
	second := []byte{5, 6, 7, 8}
	_ = rec.WriteFrame(first)
	_ = rec.WriteFrame(second)

	if !bytes.Equal(rec.lastFrame, second) {
		t.Errorf("lastFrame = %v, want %v (second frame)", rec.lastFrame, second)
	}
}

// TestStopClosesThePipe verifies that Stop() closes the underlying writer.
func TestStopClosesThePipe(t *testing.T) {
	rec, buf := newTestRecorder(4, 4)
	_ = rec.WriteFrame(make([]byte, 64))

	rec.Stop()

	if !buf.closed {
		t.Error("Stop() did not close the pipe")
	}
	if rec.pipe != nil {
		t.Error("Stop() did not nil rec.pipe")
	}
}

// TestStopIdempotent verifies that calling Stop() twice does not panic.
func TestStopIdempotent(t *testing.T) {
	rec, _ := newTestRecorder(4, 4)
	rec.Stop()
	rec.Stop() // must not panic
}

// TestNewVideoRecorderFields verifies field initialisation.
func TestNewVideoRecorderFields(t *testing.T) {
	pipe := &nopCloser{Buffer: &bytes.Buffer{}}
	rec := newVideoRecorder(pipe, 1920, 1080, "/out/video.mp4")

	if rec.width != 1920 {
		t.Errorf("width = %d, want 1920", rec.width)
	}
	if rec.height != 1080 {
		t.Errorf("height = %d, want 1080", rec.height)
	}
	if rec.outputPath != "/out/video.mp4" {
		t.Errorf("outputPath = %q, want /out/video.mp4", rec.outputPath)
	}
	if rec.cmd != nil {
		t.Error("cmd should be nil for manually constructed recorder")
	}
	if rec.lastFrame != nil {
		t.Error("lastFrame should be nil on construction")
	}
}

// TestWriteFramePropagatesError verifies that a write error from the underlying
// pipe is surfaced to the caller.
func TestWriteFramePropagatesError(t *testing.T) {
	rec := newVideoRecorder(errorWriter{}, 1, 1, "")
	err := rec.WriteFrame([]byte{1, 2, 3, 4})
	if err == nil {
		t.Error("expected error from broken pipe, got nil")
	}
}

// errorWriter is an io.WriteCloser that always returns an error on Write.
type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (errorWriter) Close() error              { return nil }

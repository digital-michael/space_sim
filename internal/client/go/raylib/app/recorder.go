package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// videoRecorder streams raw RGBA frames to an ffmpeg subprocess and produces
// a H.264 MP4 file. Pause is implemented as freeze-frame: the last captured
// frame is re-sent each tick so the video clock keeps advancing.
type videoRecorder struct {
	cmd        *exec.Cmd
	pipe       io.WriteCloser
	lastFrame  []byte // held during pause; re-written each tick
	width      int
	height     int
	outputPath string
}

// newVideoRecorder constructs a recorder with an already-open writer. Used by
// tests and by startRecording (which provides the real ffmpeg pipe).
func newVideoRecorder(pipe io.WriteCloser, width, height int, outputPath string) *videoRecorder {
	return &videoRecorder{
		pipe:       pipe,
		width:      width,
		height:     height,
		outputPath: outputPath,
	}
}

// startRecording forks an ffmpeg process and returns a ready recorder.
// If outputPath is empty a timestamped path on the Desktop is auto-generated.
// Returns an error if ffmpeg is not found or fails to start.
func startRecording(width, height int, outputPath string) (*videoRecorder, error) {
	if outputPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("recorder: home dir: %w", err)
		}
		ts := time.Now().Format("2006-01-02_15-04-05")
		outputPath = filepath.Join(home, "Desktop", fmt.Sprintf("space-sim-%s.mp4", ts))
	}

	size := fmt.Sprintf("%dx%d", width, height)
	cmd := exec.Command("ffmpeg",
		"-y", // overwrite if exists
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", size,
		"-r", "60", // input frame rate
		"-i", "pipe:0", // read from stdin
		"-vf", "vflip,scale=trunc(iw/2)*2:trunc(ih/2)*2", // flip Y; round to even dims required by libx264
		"-vcodec", "libx264",
		"-preset", "ultrafast", // minimise encode CPU overhead
		"-pix_fmt", "yuv420p", // broadest H.264 compatibility
		"-crf", "18", // near-lossless quality; raise to 23 for smaller files
		outputPath,
	)
	cmd.Stderr = os.Stderr // surface ffmpeg warnings without spamming stdout

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("recorder: stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		pipe.Close()
		return nil, fmt.Errorf("recorder: ffmpeg start (is ffmpeg installed?): %w", err)
	}

	fmt.Printf("[REC] Started: %s\n", outputPath)
	rec := newVideoRecorder(pipe, width, height, outputPath)
	rec.cmd = cmd
	return rec, nil
}

// WriteFrame sends one RGBA frame to ffmpeg. Pass nil to freeze on the last
// frame (pause mode). Returns a non-nil error only if the pipe is broken.
func (rec *videoRecorder) WriteFrame(frame []byte) error {
	if frame != nil {
		rec.lastFrame = frame
	}
	if rec.lastFrame == nil {
		return nil // nothing captured yet
	}
	_, err := rec.pipe.Write(rec.lastFrame)
	return err
}

// Stop closes the ffmpeg stdin pipe and waits for the process to finish.
func (rec *videoRecorder) Stop() {
	if rec.pipe != nil {
		rec.pipe.Close()
		rec.pipe = nil
	}
	if rec.cmd != nil {
		rec.cmd.Wait()
		rec.cmd = nil
	}
	fmt.Printf("[REC] Saved: %s\n", rec.outputPath)
}

// startRecording starts a new recording session on the app.
// outputPath is passed directly to the package-level startRecording; an empty
// string auto-generates a timestamped Desktop path.
func (a *App) startRecording(outputPath string) {
	w := int(a.runtime.RenderWidth)
	h := int(a.runtime.RenderHeight)
	if w <= 0 || h <= 0 {
		fmt.Println("[REC] No render target dimensions; cannot record.")
		return
	}
	rec, err := startRecording(w, h, outputPath)
	if err != nil {
		fmt.Printf("[REC] Error: %v\n", err)
		return
	}
	a.recorder = rec
	a.runtime.RecordingActive = true
	a.runtime.RecordingPaused = false
}

// stopRecording stops the active recording session.
func (a *App) stopRecording() {
	if a.recorder != nil {
		a.recorder.Stop()
		a.recorder = nil
	}
	a.runtime.RecordingActive = false
	a.runtime.RecordingPaused = false
}

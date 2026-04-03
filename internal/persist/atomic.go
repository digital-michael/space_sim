package persist

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWrite writes data to a temp sibling of path then renames it into
// place, so callers never observe a partial write even if the process crashes
// mid-write (provided temp and destination share the same filesystem).
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".persist-*.tmp")
	if err != nil {
		return fmt.Errorf("persist: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup; no-op after a successful Rename.
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("persist: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("persist: sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("persist: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("persist: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

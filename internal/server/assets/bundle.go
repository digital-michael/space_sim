// Package assets builds the server-authoritative content bundle that remote
// clients fetch in place of a local data/ directory.
//
// A bundle is a deterministic zip archive addressed by the SHA-256 of its
// bytes. Determinism matters for two reasons: identical content must always
// yield the same hash so a client's cache stays valid across server restarts,
// and the client must be able to verify the bytes it received before
// unpacking them.
package assets

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

// entryPrefix is prepended to every archive entry so that a client's bundle
// root joined with an existing repo-relative path — "data/assets/textures/x.jpg"
// as stored in ObjectMetadata.TexturePath — resolves without rewriting paths.
const entryPrefix = "data"

// includedDirs are the shipped-content subtrees of the data directory.
//
// "profiles" IS included, correcting an earlier reading of it as user
// configuration. They are shipped hardware keybinding profiles — app content —
// and input.LoadKeyMap treats the profile file as MANDATORY while treating the
// user's own overrides file as optional. A client with no profile cannot start.
// Genuine user customization lives in the keybindings config file, which is
// never bundled. See f040-remote-client-assets-spec.md D5.
var includedDirs = []string{"assets", "bodies", "profiles", "ships", "systems"}

// fixedModTime is stamped on every entry so identical content produces
// identical archive bytes. It is the MS-DOS epoch, the earliest instant the
// zip format can represent without clamping.
var fixedModTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// fixedMode is stamped on every entry so archive bytes do not vary with the
// source files' permission bits across checkouts and platforms.
const fixedMode fs.FileMode = 0o644

// Bundle is an immutable archive of simulation content, addressed by content hash.
type Bundle struct {
	// Hash is the hex-encoded SHA-256 of Data. It is the bundle's identity:
	// clients cache by it and verify against it.
	Hash string

	// Data is the complete zip archive. Bundles are small enough (~5 MB) that
	// holding one in memory is cheaper than managing a temp file.
	Data []byte

	// EntryCount is the number of files in the archive.
	EntryCount int
}

// Size returns the archive length in bytes.
func (b *Bundle) Size() int64 { return int64(len(b.Data)) }

// Build constructs a bundle from the simulation-content subtrees of dataDir.
//
// Subtrees that do not exist are skipped rather than treated as errors, so a
// deployment without a ship catalog still produces a valid bundle. Symlinks are
// skipped rather than followed or stored.
func Build(dataDir string) (*Bundle, error) {
	files, err := collect(dataDir)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		hdr := &zip.FileHeader{
			Name:     f.entry,
			Method:   zip.Deflate,
			Modified: fixedModTime,
		}
		hdr.SetMode(fixedMode)

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return nil, fmt.Errorf("create bundle entry %s: %w", f.entry, err)
		}
		if err := copyInto(w, f.abs); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize bundle: %w", err)
	}

	sum := sha256.Sum256(buf.Bytes())
	return &Bundle{
		Hash:       hex.EncodeToString(sum[:]),
		Data:       buf.Bytes(),
		EntryCount: len(files),
	}, nil
}

func copyInto(w io.Writer, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	return nil
}

// entryFile pairs an archive entry path with its source file on disk.
type entryFile struct {
	entry string // slash-separated, entryPrefix-rooted
	abs   string
}

// collect gathers every regular file under the included subtrees, sorted by
// entry path so archive ordering is stable.
func collect(dataDir string) ([]entryFile, error) {
	var out []entryFile
	for _, sub := range includedDirs {
		root := filepath.Join(dataDir, sub)
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", root, err)
		}
		if !info.IsDir() {
			continue
		}

		walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(dataDir, p)
			if err != nil {
				return fmt.Errorf("relativize %s: %w", p, err)
			}
			out = append(out, entryFile{
				entry: path.Join(entryPrefix, filepath.ToSlash(rel)),
				abs:   p,
			})
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("walk %s: %w", root, walkErr)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].entry < out[j].entry })
	return out, nil
}

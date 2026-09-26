// Package assets fetches, verifies, and unpacks the server-authoritative
// content bundle so a remote client can render without a local data/ directory.
//
// The cache is content-addressed: a bundle lives at <cache>/bundles/<hash>/ and
// its identity is the SHA-256 of the archive it came from. A cached bundle can
// therefore never be stale, and no invalidation protocol is needed.
//
// Unpacking an archive received over a network is the hazardous step here. Every
// entry is validated to resolve strictly inside the destination before any write,
// symlink entries are refused outright, and both per-entry and total uncompressed
// size are bounded. See docs/wip/f040-remote-client-assets-spec.md §4.
package assets

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
)

// partialSuffix marks an unpack in progress. Completion is the atomic rename
// onto the final path, so a crashed unpack leaves only a discardable *.partial
// directory and never a half-populated cache entry.
const partialSuffix = ".partial"

// Size bounds against decompression-bomb archives. Variables rather than
// constants so tests can lower them and exercise the guards without generating
// hundreds of megabytes.
var (
	// maxTotalUncompressed bounds the whole archive. Real content is ~5 MB; this
	// leaves headroom for higher-resolution texture packs while still stopping a
	// bomb long before it fills a disk.
	maxTotalUncompressed int64 = 256 << 20 // 256 MiB

	// maxEntryUncompressed bounds any single file in the archive.
	maxEntryUncompressed int64 = 64 << 20 // 64 MiB
)

// Root is a prepared bundle on local disk.
type Root struct {
	// Path is the directory the bundle was unpacked into. Bundle entries keep
	// their data/-prefixed names, so joining Path with an existing
	// repo-relative path such as "data/assets/textures/earthmap1k.jpg" resolves.
	Path string

	// Hash is the bundle's content hash, and the cache directory's name.
	Hash string

	// ActiveSystemPath is the bundle-relative path of the system the server has
	// loaded, e.g. "data/systems/solar_system".
	ActiveSystemPath string
}

// Resolve returns an absolute path for a bundle-relative path.
func (r *Root) Resolve(rel string) string {
	return filepath.Join(r.Path, filepath.FromSlash(rel))
}

// DefaultCacheDir returns the per-user cache location for bundles.
func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache dir: %w", err)
	}
	return filepath.Join(base, "space-sim"), nil
}

// Ensure makes the server's current bundle available locally and returns its
// root. On a cache hit nothing is transferred.
func Ensure(ctx context.Context, client spacesimv1connect.AssetServiceClient, cacheDir string) (*Root, error) {
	info, err := client.GetBundleInfo(ctx, connect.NewRequest(&v1.GetBundleInfoRequest{Version: 1}))
	if err != nil {
		return nil, fmt.Errorf("get bundle info: %w", err)
	}
	hash := info.Msg.GetHash()
	if hash == "" {
		return nil, errors.New("server reported an empty bundle hash")
	}

	bundlesDir := filepath.Join(cacheDir, "bundles")
	if err := os.MkdirAll(bundlesDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir %s: %w", bundlesDir, err)
	}
	// A previous run may have died mid-unpack.
	discardPartials(bundlesDir)

	root := &Root{
		Path:             filepath.Join(bundlesDir, hash),
		Hash:             hash,
		ActiveSystemPath: info.Msg.GetActiveSystemPath(),
	}

	if isDir(root.Path) {
		return root, nil
	}

	archivePath, err := download(ctx, client, hash, info.Msg.GetSizeBytes(), cacheDir)
	if err != nil {
		return nil, err
	}
	defer os.Remove(archivePath)

	partial := root.Path + partialSuffix
	if err := os.RemoveAll(partial); err != nil {
		return nil, fmt.Errorf("clear %s: %w", partial, err)
	}
	if err := Unpack(archivePath, partial); err != nil {
		os.RemoveAll(partial)
		return nil, err
	}
	if err := os.Rename(partial, root.Path); err != nil {
		os.RemoveAll(partial)
		// A concurrent client may have won the race and populated it already.
		if isDir(root.Path) {
			return root, nil
		}
		return nil, fmt.Errorf("publish bundle cache entry: %w", err)
	}
	return root, nil
}

// download streams the bundle to a temp file, hashing as it goes, and fails if
// the assembled bytes do not match the advertised hash. Verification happens
// before the archive is ever opened, so a corrupt or substituted payload is
// never unpacked.
func download(
	ctx context.Context,
	client spacesimv1connect.AssetServiceClient,
	wantHash string,
	declaredSize int64,
	cacheDir string,
) (string, error) {
	stream, err := client.FetchBundle(ctx, connect.NewRequest(&v1.FetchBundleRequest{
		Version:      1,
		ExpectedHash: wantHash,
	}))
	if err != nil {
		return "", fmt.Errorf("fetch bundle: %w", err)
	}
	defer stream.Close()

	tmp, err := os.CreateTemp(cacheDir, "bundle-*.zip")
	if err != nil {
		return "", fmt.Errorf("create temp archive: %w", err)
	}
	tmpPath := tmp.Name()

	// Bound the stream so a server that never stops sending cannot fill the
	// disk. Allow slack over the declared size rather than trusting it exactly.
	limit := maxTotalUncompressed
	if declaredSize > 0 && declaredSize < limit {
		limit = declaredSize
	}

	hasher := sha256.New()
	var written int64
	for stream.Receive() {
		chunk := stream.Msg().GetChunk()
		written += int64(len(chunk))
		if written > limit {
			tmp.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("bundle stream exceeded expected size %d bytes", limit)
		}
		if _, err := tmp.Write(chunk); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("write temp archive: %w", err)
		}
		hasher.Write(chunk)
	}
	if err := stream.Err(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("bundle stream: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("close temp archive: %w", err)
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if got != wantHash {
		os.Remove(tmpPath)
		return "", fmt.Errorf("bundle hash mismatch: received %s, expected %s", got, wantHash)
	}
	return tmpPath, nil
}

// Unpack extracts archivePath into dest, refusing any entry that could write
// outside it. Exported so the validation can be tested directly against
// deliberately malicious archives.
func Unpack(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer zr.Close()

	// Validate every entry before writing anything, so a hostile archive is
	// rejected without leaving partial output behind. Callers that unpack into a
	// staging directory get that for free, but Unpack must be safe on its own.
	type plannedEntry struct {
		file   *zip.File
		target string
	}
	planned := make([]plannedEntry, 0, len(zr.File))
	var declaredTotal uint64

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Refuse links rather than recreating them: a link is the simplest way
		// for an entry to point outside the destination after extraction.
		if f.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a symlink", f.Name)
		}
		if !f.Mode().IsRegular() {
			return fmt.Errorf("archive entry %q is not a regular file", f.Name)
		}
		// Declared sizes are attacker-controlled, so these are cheap early
		// rejects only. The authoritative bounds are enforced on bytes actually
		// written, below.
		if f.UncompressedSize64 > uint64(maxEntryUncompressed) {
			return fmt.Errorf("archive entry %q declares %d bytes, over the %d limit",
				f.Name, f.UncompressedSize64, maxEntryUncompressed)
		}
		declaredTotal += f.UncompressedSize64
		if declaredTotal > uint64(maxTotalUncompressed) {
			return fmt.Errorf("archive declares %d bytes, over the %d byte total limit",
				declaredTotal, maxTotalUncompressed)
		}

		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		planned = append(planned, plannedEntry{file: f, target: target})
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}

	var total uint64
	for _, e := range planned {
		written, err := extractFile(e.file, e.target)
		if err != nil {
			return err
		}
		total += written
		if total > uint64(maxTotalUncompressed) {
			return fmt.Errorf("archive exceeds the %d byte total limit", maxTotalUncompressed)
		}
	}
	return nil
}

func extractFile(f *zip.File, target string) (uint64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, fmt.Errorf("create parent of %s: %w", target, err)
	}
	rc, err := f.Open()
	if err != nil {
		return 0, fmt.Errorf("open archive entry %q: %w", f.Name, err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", target, err)
	}
	defer out.Close()

	// LimitReader is the authoritative bound — it holds even when the entry's
	// declared size lies.
	written, err := io.Copy(out, io.LimitReader(rc, maxEntryUncompressed+1))
	if err != nil {
		return 0, fmt.Errorf("extract %q: %w", f.Name, err)
	}
	if written > maxEntryUncompressed {
		return 0, fmt.Errorf("archive entry %q exceeds the %d byte limit", f.Name, maxEntryUncompressed)
	}
	return uint64(written), nil
}

// safeJoin resolves a relative archive entry name inside root, rejecting
// anything that escapes it.
func safeJoin(root, name string) (string, error) {
	if name == "" {
		return "", errors.New("archive contains an entry with an empty name")
	}
	// Reject both separator conventions: an archive is attacker-supplied and
	// need not follow the zip spec's forward-slash rule.
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) || filepath.IsAbs(name) {
		return "", fmt.Errorf("archive entry %q is an absolute path", name)
	}
	if strings.Contains(name, "\x00") {
		return "", fmt.Errorf("archive entry %q contains a null byte", name)
	}

	cleaned := filepath.Clean(filepath.FromSlash(name))
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination", name)
	}

	target := filepath.Join(root, cleaned)

	// Belt-and-braces containment check on the resolved absolute paths, so a
	// case this function's string rules missed still cannot escape.
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve destination %s: %w", root, err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve target %s: %w", target, err)
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q resolves outside the destination", name)
	}
	return target, nil
}

// discardPartials removes leftover in-progress unpack directories. Failures are
// ignored: a stale directory wastes space but does not affect correctness, and
// the caller has more useful work to do than fail over it.
func discardPartials(bundlesDir string) {
	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), partialSuffix) {
			os.RemoveAll(filepath.Join(bundlesDir, e.Name()))
		}
	}
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

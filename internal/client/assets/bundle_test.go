package assets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
)

// fakeAssetService serves a fixed archive. It is defined here rather than reusing
// the real server handler because internal/client/* must never import
// internal/server/* — a boundary the project treats as a hard constraint, tests
// included.
type fakeAssetService struct {
	archive          []byte
	hash             string // advertised hash; may deliberately disagree with archive
	activeSystemPath string
	fetches          atomic.Int32
}

func (f *fakeAssetService) GetBundleInfo(
	context.Context, *connect.Request[v1.GetBundleInfoRequest],
) (*connect.Response[v1.GetBundleInfoResponse], error) {
	return connect.NewResponse(&v1.GetBundleInfoResponse{
		Version:          1,
		Hash:             f.hash,
		SizeBytes:        int64(len(f.archive)),
		EntryCount:       1,
		ActiveSystemPath: f.activeSystemPath,
	}), nil
}

func (f *fakeAssetService) FetchBundle(
	_ context.Context,
	_ *connect.Request[v1.FetchBundleRequest],
	stream *connect.ServerStream[v1.FetchBundleResponse],
) error {
	f.fetches.Add(1)
	const chunk = 32 * 1024
	for off := 0; off < len(f.archive); off += chunk {
		end := off + chunk
		if end > len(f.archive) {
			end = len(f.archive)
		}
		if err := stream.Send(&v1.FetchBundleResponse{Version: 1, Chunk: f.archive[off:end]}); err != nil {
			return err
		}
	}
	return nil
}

func serveFake(t *testing.T, svc *fakeAssetService) spacesimv1connect.AssetServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	path, h := spacesimv1connect.NewAssetServiceHandler(svc)
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return spacesimv1connect.NewAssetServiceClient(srv.Client(), srv.URL)
}

// zipEntry describes one archive member, including deliberately hostile ones.
type zipEntry struct {
	name    string
	body    string
	symlink bool
}

func makeZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.symlink {
			hdr.SetMode(fs.ModeSymlink | 0o777)
		} else {
			hdr.SetMode(0o644)
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatalf("create entry %q: %v", e.name, err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatalf("write entry %q: %v", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func writeTempZip(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write temp zip: %v", err)
	}
	return p
}

func hashOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func goodArchive(t *testing.T) []byte {
	return makeZip(t, []zipEntry{
		{name: "data/systems/solar_system/system.json", body: `{"schema_version":2}`},
		{name: "data/assets/textures/earthmap1k.jpg", body: "not-really-a-jpeg"},
	})
}

// ── safeJoin ────────────────────────────────────────────────────────────────

func TestSafeJoinRejectsEscapes(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"plain nested path", "data/systems/system.json", false},
		{"single file", "file.txt", false},
		{"dot-prefixed but contained", "data/./systems/x.json", false},
		{"parent traversal", "../escape.txt", true},
		{"deep traversal", "data/../../escape.txt", true},
		{"bare parent", "..", true},
		{"absolute unix path", "/etc/passwd", true},
		{"backslash absolute", `\windows\system32`, true},
		{"empty name", "", true},
		{"null byte", "data/x\x00.json", true},
		{"traversal via many segments", "a/b/../../../../out.txt", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := safeJoin(root, tc.entry)
			if tc.wantErr {
				if err == nil {
					t.Errorf("safeJoin(%q) = %q, want error", tc.entry, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("safeJoin(%q) unexpected error: %v", tc.entry, err)
			}
			if !strings.HasPrefix(got, root+string(filepath.Separator)) {
				t.Errorf("safeJoin(%q) = %q, which is not inside %q", tc.entry, got, root)
			}
		})
	}
}

// ── Unpack: hostile archives ────────────────────────────────────────────────

func TestUnpackRejectsTraversalEntry(t *testing.T) {
	archive := writeTempZip(t, makeZip(t, []zipEntry{
		{name: "data/ok.json", body: "{}"},
		{name: "../escaped.txt", body: "pwned"},
	}))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err == nil {
		t.Fatal("expected Unpack to reject a traversal entry")
	} else if !strings.Contains(err.Error(), "escape") {
		t.Errorf("error did not describe an escape: %v", err)
	}

	// Nothing may have been written beside the destination.
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "escaped.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("traversal entry was written outside the destination")
	}

	// Unpack validates every entry before writing any, so the benign entry that
	// precedes the hostile one must not have landed either.
	if _, err := os.Stat(filepath.Join(dest, "data", "ok.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a hostile archive produced partial output; validation must precede all writes")
	}
}

func TestUnpackRejectsSymlinkEntry(t *testing.T) {
	archive := writeTempZip(t, makeZip(t, []zipEntry{
		{name: "data/link", body: "/etc/passwd", symlink: true},
	}))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err == nil {
		t.Fatal("expected Unpack to reject a symlink entry")
	} else if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("error did not describe a symlink: %v", err)
	}
}

func TestUnpackRejectsAbsolutePathEntry(t *testing.T) {
	archive := writeTempZip(t, makeZip(t, []zipEntry{
		{name: "/tmp/absolute.txt", body: "pwned"},
	}))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err == nil {
		t.Fatal("expected Unpack to reject an absolute-path entry")
	}
}

func TestUnpackEnforcesPerEntryLimit(t *testing.T) {
	origEntry := maxEntryUncompressed
	t.Cleanup(func() { maxEntryUncompressed = origEntry })
	maxEntryUncompressed = 64

	archive := writeTempZip(t, makeZip(t, []zipEntry{
		{name: "data/big.bin", body: strings.Repeat("A", 500)},
	}))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err == nil {
		t.Fatal("expected Unpack to reject an oversized entry")
	}
}

// A highly compressible payload declares a small compressed size but expands
// far beyond it — the decompression-bomb shape. The bound must hold on bytes
// actually written, not on the declared size.
func TestUnpackEnforcesTotalLimit(t *testing.T) {
	origTotal := maxTotalUncompressed
	origEntry := maxEntryUncompressed
	t.Cleanup(func() {
		maxTotalUncompressed = origTotal
		maxEntryUncompressed = origEntry
	})
	maxTotalUncompressed = 1000
	maxEntryUncompressed = 900

	archive := writeTempZip(t, makeZip(t, []zipEntry{
		{name: "data/a.bin", body: strings.Repeat("A", 800)},
		{name: "data/b.bin", body: strings.Repeat("B", 800)},
	}))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err == nil {
		t.Fatal("expected Unpack to reject an archive over the total limit")
	} else if !strings.Contains(err.Error(), "total limit") {
		t.Errorf("error did not describe the total limit: %v", err)
	}
}

func TestUnpackExtractsValidArchive(t *testing.T) {
	archive := writeTempZip(t, goodArchive(t))
	dest := filepath.Join(t.TempDir(), "out")

	if err := Unpack(archive, dest); err != nil {
		t.Fatalf("Unpack: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "data", "systems", "solar_system", "system.json"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(got) != `{"schema_version":2}` {
		t.Errorf("extracted content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "data", "assets", "textures", "earthmap1k.jpg")); err != nil {
		t.Errorf("second entry missing: %v", err)
	}
}

// ── Ensure ──────────────────────────────────────────────────────────────────

func TestEnsureFetchesThenCaches(t *testing.T) {
	archive := goodArchive(t)
	svc := &fakeAssetService{archive: archive, hash: hashOf(archive), activeSystemPath: "data/systems/solar_system"}
	client := serveFake(t, svc)
	cache := t.TempDir()

	root, err := Ensure(context.Background(), client, cache)
	if err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	if root.Hash != svc.hash {
		t.Errorf("hash = %q, want %q", root.Hash, svc.hash)
	}
	if root.ActiveSystemPath != "data/systems/solar_system" {
		t.Errorf("activeSystemPath = %q", root.ActiveSystemPath)
	}
	if _, err := os.Stat(root.Resolve("data/systems/solar_system/system.json")); err != nil {
		t.Errorf("bundle content not resolvable through Root.Resolve: %v", err)
	}
	if n := svc.fetches.Load(); n != 1 {
		t.Errorf("fetches after first Ensure = %d, want 1", n)
	}

	// Second call must be satisfied entirely from cache.
	root2, err := Ensure(context.Background(), client, cache)
	if err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if root2.Path != root.Path {
		t.Errorf("cache path changed: %q then %q", root.Path, root2.Path)
	}
	if n := svc.fetches.Load(); n != 1 {
		t.Errorf("fetches after cached Ensure = %d, want 1 — cache was not used", n)
	}
}

// The advertised hash disagreeing with the delivered bytes must fail before any
// archive is opened, so substituted content is never unpacked.
func TestEnsureRejectsHashMismatchBeforeUnpacking(t *testing.T) {
	archive := goodArchive(t)
	svc := &fakeAssetService{
		archive: archive,
		hash:    strings.Repeat("a", 64), // advertised hash the bytes will not match
	}
	client := serveFake(t, svc)
	cache := t.TempDir()

	_, err := Ensure(context.Background(), client, cache)
	if err == nil {
		t.Fatal("expected Ensure to reject a hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error did not describe a hash mismatch: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(cache, "bundles"))
	for _, e := range entries {
		t.Errorf("nothing should remain in the cache, found %q", e.Name())
	}
}

func TestEnsureRejectsEmptyHash(t *testing.T) {
	archive := goodArchive(t)
	client := serveFake(t, &fakeAssetService{archive: archive, hash: ""})

	if _, err := Ensure(context.Background(), client, t.TempDir()); err == nil {
		t.Fatal("expected Ensure to reject an empty advertised hash")
	}
}

func TestEnsureDiscardsStalePartialDirs(t *testing.T) {
	archive := goodArchive(t)
	svc := &fakeAssetService{archive: archive, hash: hashOf(archive)}
	client := serveFake(t, svc)
	cache := t.TempDir()

	// Simulate a previous run that died mid-unpack.
	stale := filepath.Join(cache, "bundles", "deadbeef"+partialSuffix)
	if err := os.MkdirAll(filepath.Join(stale, "data"), 0o755); err != nil {
		t.Fatalf("seed stale partial: %v", err)
	}

	if _, err := Ensure(context.Background(), client, cache); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Error("stale .partial directory was not discarded")
	}
}

func TestEnsureLeavesNoPartialOnUnpackFailure(t *testing.T) {
	// Valid hash, but the archive carries a traversal entry so unpack fails.
	archive := makeZip(t, []zipEntry{{name: "../escaped.txt", body: "pwned"}})
	svc := &fakeAssetService{archive: archive, hash: hashOf(archive)}
	client := serveFake(t, svc)
	cache := t.TempDir()

	if _, err := Ensure(context.Background(), client, cache); err == nil {
		t.Fatal("expected Ensure to fail on a hostile archive")
	}

	entries, _ := os.ReadDir(filepath.Join(cache, "bundles"))
	for _, e := range entries {
		t.Errorf("failed unpack left %q behind", e.Name())
	}
	if _, err := os.Stat(filepath.Join(cache, "bundles", "escaped.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("traversal entry escaped into the cache dir")
	}
}

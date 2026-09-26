package grpcserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/digital-michael/space_sim/internal/server/assets"
)

// newAssetTestServer stands up a real HTTP server carrying only AssetService and
// returns a client for it. Streaming RPCs need an actual transport, so these are
// integration tests rather than direct handler calls.
func newAssetTestServer(t *testing.T, h *AssetHandler) spacesimv1connect.AssetServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := spacesimv1connect.NewAssetServiceHandler(h)
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return spacesimv1connect.NewAssetServiceClient(srv.Client(), srv.URL)
}

// buildTestBundle produces a bundle from a synthetic content tree. extraBytes,
// when positive, adds a filler asset of that size so the archive exceeds a
// single chunk.
func buildTestBundle(t *testing.T, extraBytes int) *assets.Bundle {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"systems/solar_system/system.json": `{"schema_version":2,"name":"Sol"}`,
		"systems/solar_system/stars.json":  `{"bodies":[{"name":"Sol"}]}`,
		"bodies/earth.json":                `{"name":"Earth"}`,
	}
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	if extraBytes > 0 {
		// Filler must be genuinely incompressible or deflate reduces it to
		// almost nothing and the archive never exceeds one chunk. A seeded PRNG
		// gives incompressible bytes while keeping the test deterministic.
		filler := make([]byte, extraBytes)
		rng := rand.New(rand.NewSource(1))
		if _, err := rng.Read(filler); err != nil {
			t.Fatalf("generate filler: %v", err)
		}
		full := filepath.Join(root, "assets", "textures", "filler.bin")
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir assets: %v", err)
		}
		if err := os.WriteFile(full, filler, 0o644); err != nil {
			t.Fatalf("write filler: %v", err)
		}
	}

	b, err := assets.Build(root)
	if err != nil {
		t.Fatalf("build test bundle: %v", err)
	}
	return b
}

// drainBundle streams the whole bundle and returns the reassembled bytes plus
// the number of chunks received.
func drainBundle(t *testing.T, c spacesimv1connect.AssetServiceClient, expectedHash string) ([]byte, int, error) {
	t.Helper()
	stream, err := c.FetchBundle(context.Background(), connect.NewRequest(&v1.FetchBundleRequest{
		Version:      1,
		ExpectedHash: expectedHash,
	}))
	if err != nil {
		return nil, 0, err
	}
	defer stream.Close()

	var buf bytes.Buffer
	chunks := 0
	for stream.Receive() {
		buf.Write(stream.Msg().GetChunk())
		chunks++
	}
	return buf.Bytes(), chunks, stream.Err()
}

func TestAssetService_GetBundleInfo_RoundTrip(t *testing.T) {
	b := buildTestBundle(t, 0)
	c := newAssetTestServer(t, NewAssetHandler(b, "data/systems/solar_system"))

	resp, err := c.GetBundleInfo(context.Background(), connect.NewRequest(&v1.GetBundleInfoRequest{Version: 1}))
	if err != nil {
		t.Fatalf("GetBundleInfo: %v", err)
	}

	if got := resp.Msg.GetHash(); got != b.Hash {
		t.Errorf("hash = %q, want %q", got, b.Hash)
	}
	if got := resp.Msg.GetSizeBytes(); got != b.Size() {
		t.Errorf("size = %d, want %d", got, b.Size())
	}
	if got := resp.Msg.GetEntryCount(); got != uint32(b.EntryCount) {
		t.Errorf("entryCount = %d, want %d", got, b.EntryCount)
	}
	if got := resp.Msg.GetActiveSystemPath(); got != "data/systems/solar_system" {
		t.Errorf("activeSystemPath = %q, want %q", got, "data/systems/solar_system")
	}
}

// The reassembled stream must hash to the advertised value, or a client could
// never safely verify before unpacking.
func TestAssetService_FetchBundle_ReassemblesToAdvertisedHash(t *testing.T) {
	b := buildTestBundle(t, 0)
	c := newAssetTestServer(t, NewAssetHandler(b, "data/systems/solar_system"))

	got, chunks, err := drainBundle(t, c, b.Hash)
	if err != nil {
		t.Fatalf("FetchBundle: %v", err)
	}
	if chunks == 0 {
		t.Fatal("received no chunks")
	}
	if !bytes.Equal(got, b.Data) {
		t.Errorf("reassembled %d bytes, want %d", len(got), len(b.Data))
	}
	sum := sha256.Sum256(got)
	if hex.EncodeToString(sum[:]) != b.Hash {
		t.Error("reassembled bytes do not hash to the advertised bundle hash")
	}
}

func TestAssetService_FetchBundle_SpansMultipleChunks(t *testing.T) {
	// 300 KiB of filler against a 64 KiB chunk size.
	b := buildTestBundle(t, 300*1024)
	if b.Size() <= chunkSize {
		t.Fatalf("test bundle is %d bytes, needs to exceed chunkSize %d to exercise chunking", b.Size(), chunkSize)
	}
	c := newAssetTestServer(t, NewAssetHandler(b, "data/systems/solar_system"))

	got, chunks, err := drainBundle(t, c, b.Hash)
	if err != nil {
		t.Fatalf("FetchBundle: %v", err)
	}
	if chunks < 2 {
		t.Errorf("got %d chunks for a %d-byte bundle, want at least 2", chunks, b.Size())
	}
	if !bytes.Equal(got, b.Data) {
		t.Error("multi-chunk reassembly did not reproduce the archive exactly")
	}
}

// A client asking for a specific bundle must not silently receive a different
// one, or it would fail local verification with no indication why.
func TestAssetService_FetchBundle_HashMismatchRejected(t *testing.T) {
	b := buildTestBundle(t, 0)
	c := newAssetTestServer(t, NewAssetHandler(b, "data/systems/solar_system"))

	_, _, err := drainBundle(t, c, strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("expected FetchBundle to reject a mismatched expected_hash")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want %v (err: %v)", got, connect.CodeFailedPrecondition, err)
	}
}

// An empty expected_hash means "no preference"; the client still verifies the
// assembled bytes locally before unpacking.
func TestAssetService_FetchBundle_EmptyExpectedHashAllowed(t *testing.T) {
	b := buildTestBundle(t, 0)
	c := newAssetTestServer(t, NewAssetHandler(b, "data/systems/solar_system"))

	got, _, err := drainBundle(t, c, "")
	if err != nil {
		t.Fatalf("FetchBundle with empty expected_hash: %v", err)
	}
	if !bytes.Equal(got, b.Data) {
		t.Error("reassembly mismatch")
	}
}

// A server without a bundle must report Unavailable rather than panic on a nil
// dereference, so a misconfigured deployment still runs the simulation.
func TestAssetService_NilBundle_ReportsUnavailable(t *testing.T) {
	c := newAssetTestServer(t, NewAssetHandler(nil, ""))

	if _, err := c.GetBundleInfo(context.Background(), connect.NewRequest(&v1.GetBundleInfoRequest{Version: 1})); err == nil {
		t.Error("GetBundleInfo with nil bundle should fail")
	} else if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("GetBundleInfo code = %v, want %v", got, connect.CodeUnavailable)
	}

	if _, _, err := drainBundle(t, c, ""); err == nil {
		t.Error("FetchBundle with nil bundle should fail")
	} else if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("FetchBundle code = %v, want %v", got, connect.CodeUnavailable)
	}
}

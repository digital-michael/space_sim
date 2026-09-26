package grpcserver

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/server/assets"
)

// chunkSize is the payload carried per FetchBundle message. 64 KiB sits far
// below gRPC's default 4 MB message limit and behaves well under stream flow
// control. A ~4 MB bundle is ~64 messages, so per-message overhead is
// negligible and larger chunks would buy nothing.
const chunkSize = 64 * 1024

var errNoBundle = errors.New("server has no asset bundle")

// AssetHandler implements spacesimv1connect.AssetServiceHandler.
//
// It serves a single immutable bundle built once at startup. Because the bundle
// never changes for the lifetime of the process, the handler needs no locking.
// An operator who edits data files must restart the server for clients to see
// the change — which is also what makes the content hash a safe cache key.
type AssetHandler struct {
	bundle *assets.Bundle

	// activeSystemPath is the bundle-relative path of the loaded system, e.g.
	// "data/systems/solar_system". Reported alongside bundle info because a
	// headless server cannot serve SystemService.GetActiveSystem — that handler
	// depends on the Raylib app's command channel.
	activeSystemPath string
}

// NewAssetHandler constructs an AssetHandler around an already-built bundle and
// the bundle-relative path of the currently loaded system.
//
// A nil bundle is permitted; both RPCs then report Unavailable rather than
// panicking, so a server misconfigured for asset serving still runs.
func NewAssetHandler(b *assets.Bundle, activeSystemPath string) *AssetHandler {
	return &AssetHandler{bundle: b, activeSystemPath: activeSystemPath}
}

// GetBundleInfo reports the current bundle's identity and size.
func (h *AssetHandler) GetBundleInfo(
	_ context.Context,
	_ *connect.Request[v1.GetBundleInfoRequest],
) (*connect.Response[v1.GetBundleInfoResponse], error) {
	if h.bundle == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoBundle)
	}
	return connect.NewResponse(&v1.GetBundleInfoResponse{
		Version:          1,
		Hash:             h.bundle.Hash,
		SizeBytes:        h.bundle.Size(),
		EntryCount:       uint32(h.bundle.EntryCount),
		ActiveSystemPath: h.activeSystemPath,
	}), nil
}

// FetchBundle streams the archive in ordered chunks.
//
// An expected_hash that does not match the served bundle is rejected rather
// than silently satisfied: a client asking for a specific bundle must not
// receive a different one, or it would fail local verification with no
// indication of why.
func (h *AssetHandler) FetchBundle(
	ctx context.Context,
	req *connect.Request[v1.FetchBundleRequest],
	stream *connect.ServerStream[v1.FetchBundleResponse],
) error {
	if h.bundle == nil {
		return connect.NewError(connect.CodeUnavailable, errNoBundle)
	}

	// An empty expected_hash means the client has no preference — it will still
	// verify the assembled bytes locally before unpacking.
	if want := req.Msg.GetExpectedHash(); want != "" && want != h.bundle.Hash {
		return connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("bundle hash is %s, client expected %s", h.bundle.Hash, want))
	}

	data := h.bundle.Data
	for off := 0; off < len(data); off += chunkSize {
		if err := ctx.Err(); err != nil {
			return connect.NewError(connect.CodeCanceled, err)
		}
		end := off + chunkSize
		if end > len(data) {
			end = len(data)
		}
		if err := stream.Send(&v1.FetchBundleResponse{Version: 1, Chunk: data[off:end]}); err != nil {
			return fmt.Errorf("send bundle chunk at offset %d: %w", off, err)
		}
	}
	return nil
}

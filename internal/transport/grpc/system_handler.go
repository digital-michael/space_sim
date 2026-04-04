package grpcserver

import (
	"context"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/google/uuid"
)

// SystemHandler implements spacesimv1connect.SystemServiceHandler.
// ListSystems reads the filesystem directly (safe from any goroutine).
// GetActiveSystem and LoadSystem go through the AppCmd channel to the main thread.
type SystemHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewSystemHandler constructs a SystemHandler.
// sendCmd is typically app.SendCmd — it must be safe to call from any goroutine.
func NewSystemHandler(sendCmd func(rayapp.AppCmd) bool) *SystemHandler {
	return &SystemHandler{sendCmd: sendCmd}
}

func (h *SystemHandler) ListSystems(_ context.Context, _ *connect.Request[v1.ListSystemsRequest]) (*connect.Response[v1.ListSystemsResponse], error) {
	options, err := rayapp.DiscoverSystems()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	systems := make([]*v1.SystemInfo, 0, len(options))
	for _, opt := range options {
		systems = append(systems, &v1.SystemInfo{Label: opt.Label, Path: opt.Path})
	}
	return connect.NewResponse(&v1.ListSystemsResponse{Version: 1, Systems: systems}), nil
}

func (h *SystemHandler) GetActiveSystem(ctx context.Context, _ *connect.Request[v1.GetActiveSystemRequest]) (*connect.Response[v1.GetActiveSystemResponse], error) {
	respCh := make(chan string, 1)
	if !h.sendCmd(rayapp.GetActiveSystemCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case path := <-respCh:
		label := ui.SystemOption{Label: filepath.Base(path), Path: path}
		return connect.NewResponse(&v1.GetActiveSystemResponse{
			Version: 1,
			Active:  &v1.SystemInfo{Label: label.Label, Path: label.Path},
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}

func (h *SystemHandler) LoadSystem(_ context.Context, req *connect.Request[v1.LoadSystemRequest]) (*connect.Response[v1.LoadSystemResponse], error) {
	path := req.Msg.Path
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEmptyPath)
	}
	// Validate: must be under data/systems/
	if !isAllowedSystemPath(path) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errPathTraversal)
	}
	if !h.sendCmd(rayapp.LoadSystemCmd{Path: path}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.LoadSystemResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

// isAllowedSystemPath returns true when path is a simple filename or a path
// that begins with data/systems/ (after cleaning). This prevents directory
// traversal attacks.
func isAllowedSystemPath(path string) bool {
	// Reject traversal attempts in the raw path before Clean can resolve them.
	if strings.Contains(path, "..") {
		return false
	}
	// Reject absolute paths.
	if filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(path)
	// Accept bare filename or explicit data/systems/ prefix.
	if filepath.Dir(clean) == "." {
		return true
	}
	return strings.HasPrefix(filepath.ToSlash(clean), "data/systems/")
}

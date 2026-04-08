package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
)

// RecordingHandler implements spacesimv1connect.RecordingServiceHandler.
type RecordingHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewRecordingHandler constructs a RecordingHandler.
func NewRecordingHandler(sendCmd func(rayapp.AppCmd) bool) *RecordingHandler {
	return &RecordingHandler{sendCmd: sendCmd}
}

func (h *RecordingHandler) StartRecording(_ context.Context, req *connect.Request[v1.StartRecordingRequest]) (*connect.Response[v1.StartRecordingResponse], error) {
	if !h.sendCmd(rayapp.RecordStartCmd{Path: req.Msg.OutputPath}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.StartRecordingResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *RecordingHandler) PauseRecording(_ context.Context, _ *connect.Request[v1.PauseRecordingRequest]) (*connect.Response[v1.PauseRecordingResponse], error) {
	if !h.sendCmd(rayapp.RecordPauseCmd{}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.PauseRecordingResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *RecordingHandler) StopRecording(_ context.Context, _ *connect.Request[v1.StopRecordingRequest]) (*connect.Response[v1.StopRecordingResponse], error) {
	if !h.sendCmd(rayapp.RecordStopCmd{}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.StopRecordingResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

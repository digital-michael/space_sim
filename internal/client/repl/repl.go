// Package repl implements an interactive read-eval-print loop that connects
// to a running space-sim-grpc server and exposes SimulationService and
// WorldService operations as simple text commands.
//
// Commands are parsed by [commands.Parse]; each maps to one ConnectRPC call.
// Errors from the server are printed and the loop continues — only 'quit'
// or EOF exit cleanly.
package repl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

// REPL holds the live ConnectRPC client connections and executes parsed
// commands against them.
type REPL struct {
	simClient   spacesimv1connect.SimulationServiceClient
	worldClient spacesimv1connect.WorldServiceClient
	sysClient   spacesimv1connect.SystemServiceClient
	winClient   spacesimv1connect.WindowServiceClient
	camClient   spacesimv1connect.CameraServiceClient
	navClient   spacesimv1connect.NavigationServiceClient
	perfClient  spacesimv1connect.PerformanceServiceClient
	sdClient    spacesimv1connect.ShutdownServiceClient
	out         io.Writer
	lastSpeed   float32  // restored by resume; updated by setspeed / pause
	bodyNames   []string // cached body names for TAB completion; nil = not yet fetched
}

// New creates a REPL connected to addr (e.g. "http://localhost:9090").
// The HTTP client used is net/http's default — suitable for local Connect
// protocol. For gRPC wire format over HTTP/2 cleartext pass
// connect.WithGRPC() + an h2c transport as opts.
func New(addr string, opts ...connect.ClientOption) *REPL {
	return &REPL{
		simClient:   spacesimv1connect.NewSimulationServiceClient(http.DefaultClient, addr, opts...),
		worldClient: spacesimv1connect.NewWorldServiceClient(http.DefaultClient, addr, opts...),
		sysClient:   spacesimv1connect.NewSystemServiceClient(http.DefaultClient, addr, opts...),
		winClient:   spacesimv1connect.NewWindowServiceClient(http.DefaultClient, addr, opts...),
		camClient:   spacesimv1connect.NewCameraServiceClient(http.DefaultClient, addr, opts...),
		navClient:   spacesimv1connect.NewNavigationServiceClient(http.DefaultClient, addr, opts...),
		perfClient:  spacesimv1connect.NewPerformanceServiceClient(http.DefaultClient, addr, opts...),
		sdClient:    spacesimv1connect.NewShutdownServiceClient(http.DefaultClient, addr, opts...),
		out:         os.Stdout,
		lastSpeed:   1.0,
	}
}

// Run starts the interactive loop. It reads from in (typically os.Stdin)
// until EOF, 'quit', or ctx is cancelled.
//
// When in is a TTY, raw mode is enabled and the following keybindings are
// active:
//   - Up / Down arrows — navigate command history
//   - ESC              — clear the current line
//   - Ctrl-C / Ctrl-D  — exit the REPL
func (r *REPL) Run(ctx context.Context, in io.Reader) error {
	lr := newLineReader(in)
	lr.TabCompleter = func(partial string) []string {
		if needsBodyNames(partial) && r.bodyNames == nil {
			r.refreshBodyNames(ctx)
		}
		return replComplete(partial, r.bodyNames)
	}
	r.printf("Space Sim REPL — type 'help' for commands\n")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := lr.readLine("> ")
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		cmd, err := commands.Parse(line)
		if err != nil {
			r.printf("error: %v\n", err)
			continue
		}
		if cmd == nil {
			continue
		}

		done, execErr := r.exec(ctx, cmd)
		if execErr != nil {
			r.printf("error: %v\n", execErr)
		}
		if done {
			return nil
		}
	}
}

// exec dispatches cmd and returns (done=true) when the REPL should exit.
func (r *REPL) exec(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.SetSpeed:
		resp, err := r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{
			SecondsPerSecond: c.SecondsPerSecond,
		}))
		if err != nil {
			return false, err
		}
		if c.SecondsPerSecond > 0 {
			r.lastSpeed = c.SecondsPerSecond
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetSpeed:
		resp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("speed = %.4g s/s\n", resp.Msg.SecondsPerSecond)

	case commands.SetDataset:
		level := levelToProto(c.Level)
		resp, err := r.simClient.SetDataset(ctx, connect.NewRequest(&v1.SetDatasetRequest{Level: level}))
		if err != nil {
			return false, err
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetDataset:
		resp, err := r.simClient.GetDataset(ctx, connect.NewRequest(&v1.GetDatasetRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("dataset = %s\n", strings.ToLower(strings.TrimPrefix(resp.Msg.Level.String(), "DATASET_LEVEL_")))

	case commands.GetTime:
		resp, err := r.simClient.GetSimulationTime(ctx, connect.NewRequest(&v1.GetSimulationTimeRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("simulation_time = %.2f s (J2000)\n", resp.Msg.SecondsSinceJ2000)

	case commands.Stream:
		return false, r.runStream(ctx)

	case commands.Pause:
		// Snapshot the live speed before pausing so resume can restore it.
		speedResp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
		if err != nil {
			return false, fmt.Errorf("pause: %w", err)
		}
		if cur := speedResp.Msg.SecondsPerSecond; cur > 0 {
			r.lastSpeed = cur
		}
		_, err = r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{SecondsPerSecond: 0}))
		if err != nil {
			return false, err
		}
		r.printf("paused  (resume will restore %.4g sec/sec)\n", r.lastSpeed)

	case commands.Resume:
		speed := r.lastSpeed
		if speed <= 0 {
			speed = 1.0
		}
		resp, err := r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{SecondsPerSecond: speed}))
		if err != nil {
			return false, err
		}
		r.lastSpeed = speed
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s  (%.4g sec/sec)\n", ack.GetEventId(), ack.GetStatus(), speed)

	case commands.Bodies:
		return false, r.runBodies(ctx, c.Category)

	case commands.Inspect:
		return false, r.runInspect(ctx, c.Name)

	case commands.Status:
		return false, r.runStatus(ctx)

	case commands.Clear:
		r.printf("\x1b[2J\x1b[H")

	case commands.Help:
		r.printHelp()

	case commands.Quit:
		r.printf("bye\n")
		return true, nil

	// ── System ────────────────────────────────────────────────────────────────

	case commands.SystemList:
		resp, err := r.sysClient.ListSystems(ctx, connect.NewRequest(&v1.ListSystemsRequest{}))
		if err != nil {
			return false, err
		}
		for _, s := range resp.Msg.Systems {
			r.printf("  %s\n", s.Label)
		}

	case commands.SystemGet:
		resp, err := r.sysClient.GetActiveSystem(ctx, connect.NewRequest(&v1.GetActiveSystemRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("active system: %s\n", resp.Msg.GetActive().GetLabel())

	case commands.SystemLoad:
		resp, err := r.sysClient.LoadSystem(ctx, connect.NewRequest(&v1.LoadSystemRequest{Path: c.Label}))
		if err != nil {
			return false, err
		}
		r.bodyNames = nil // new system — invalidate body name cache
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Window ────────────────────────────────────────────────────────────────

	case commands.WindowGet:
		resp, err := r.winClient.GetWindow(ctx, connect.NewRequest(&v1.GetWindowRequest{}))
		if err != nil {
			return false, err
		}
		w := resp.Msg.GetWindow()
		r.printf("size: %dx%d  maximized: %v  fullscreen: %v\n",
			w.GetWidth(), w.GetHeight(), w.GetMaximized(), w.GetFullscreen())

	case commands.WindowSize:
		resp, err := r.winClient.SetWindowSize(ctx, connect.NewRequest(&v1.SetWindowSizeRequest{
			Width: c.Width, Height: c.Height,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.WindowMaximize:
		resp, err := r.winClient.SetWindowMaximize(ctx, connect.NewRequest(&v1.SetWindowMaximizeRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.WindowRestore:
		resp, err := r.winClient.SetWindowRestore(ctx, connect.NewRequest(&v1.SetWindowRestoreRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Camera ────────────────────────────────────────────────────────────────

	case commands.CameraCenter:
		camResp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
		if err != nil {
			return false, err
		}
		target := camResp.Msg.GetCamera().GetTrackTarget()
		if target != "" {
			// Tracking a body — animate jump back to it.
			resp, err := r.navClient.JumpTo(ctx, connect.NewRequest(&v1.JumpToRequest{Names: []string{target}}))
			if err != nil {
				return false, err
			}
			r.printf("centering on %s  event_id=%s  status=%s\n", target, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		} else {
			// Free-fly — teleport to solar system origin.
			resp, err := r.camClient.SetCameraPosition(ctx, connect.NewRequest(&v1.SetCameraPositionRequest{
				PosX: 0, PosY: 0, PosZ: 0,
			}))
			if err != nil {
				return false, err
			}
			r.printf("centering on system origin  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		}

	case commands.CameraGet:
		resp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
		if err != nil {
			return false, err
		}
		cam := resp.Msg.GetCamera()
		r.printf("position: x=%.4f  y=%.4f  z=%.4f AU\n", cam.GetPosX(), cam.GetPosY(), cam.GetPosZ())
		r.printf("orient:   yaw=%.2f°  pitch=%.2f°\n", cam.GetYawDeg(), cam.GetPitchDeg())
		r.printf("mode:     %s  tracking: %q\n", cam.GetMode(), cam.GetTrackTarget())

	case commands.CameraOrient:
		resp, err := r.camClient.SetCameraOrient(ctx, connect.NewRequest(&v1.SetCameraOrientRequest{
			YawDeg: c.YawDeg, PitchDeg: c.PitchDeg,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.CameraPosition:
		resp, err := r.camClient.SetCameraPosition(ctx, connect.NewRequest(&v1.SetCameraPositionRequest{
			PosX: c.X, PosY: c.Y, PosZ: c.Z,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.CameraTrack:
		resp, err := r.camClient.SetCameraTrack(ctx, connect.NewRequest(&v1.SetCameraTrackRequest{
			Name: c.Name,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Navigation ────────────────────────────────────────────────────────────

	case commands.NavStop:
		resp, err := r.navClient.SetVelocity(ctx, connect.NewRequest(&v1.SetVelocityRequest{
			Velocity: &v1.Velocity3{},
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.NavVelocity:
		resp, err := r.navClient.GetVelocity(ctx, connect.NewRequest(&v1.GetVelocityRequest{}))
		if err != nil {
			return false, err
		}
		vel := resp.Msg.GetVelocity()
		r.printf("velocity: vx=%.4f  vy=%.4f  vz=%.4f AU/s\n", vel.GetX(), vel.GetY(), vel.GetZ())

	case commands.NavMove:
		var vx, vy, vz float32
		switch c.Dir {
		case "forward":
			vz = -c.Velocity
		case "back":
			vz = c.Velocity
		case "left":
			vx = -c.Velocity
		case "right":
			vx = c.Velocity
		case "up":
			vy = c.Velocity
		case "down":
			vy = -c.Velocity
		}
		resp, err := r.navClient.SetVelocity(ctx, connect.NewRequest(&v1.SetVelocityRequest{
			Velocity: &v1.Velocity3{X: vx, Y: vy, Z: vz},
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.NavJump:
		resp, err := r.navClient.JumpTo(ctx, connect.NewRequest(&v1.JumpToRequest{Names: c.Names}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Performance ───────────────────────────────────────────────────────────

	case commands.PerfGet:
		resp, err := r.perfClient.GetPerformance(ctx, connect.NewRequest(&v1.GetPerformanceRequest{}))
		if err != nil {
			return false, err
		}
		p := resp.Msg.GetState()
		r.printf("frustum_culling:      %v\n", p.GetFrustumCulling())
		r.printf("lod_enabled:          %v\n", p.GetLodEnabled())
		r.printf("instanced_rendering:  %v\n", p.GetInstancedRendering())
		r.printf("spatial_partition:    %v\n", p.GetSpatialPartition())
		r.printf("point_rendering:      %v\n", p.GetPointRendering())
		r.printf("importance_threshold: %d\n", p.GetImportanceThreshold())
		r.printf("use_in_place_swap:    %v\n", p.GetUseInPlaceSwap())
		r.printf("camera_speed:         %.4g\n", p.GetCameraSpeed())
		r.printf("workers:              %d\n", p.GetNumWorkers())

	case commands.PerfSet:
		req := perfSetField(c.Field, c.Value)
		if req == nil {
			return false, fmt.Errorf("perf set: unknown field %q", c.Field)
		}
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(req))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Shutdown ──────────────────────────────────────────────────────────────

	case commands.Shutdown:
		_, err := r.sdClient.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("server shutting down\n")
		return true, nil

	// ── Orbit ─────────────────────────────────────────────────────────────────

	case commands.Orbit:
		resp, err := r.camClient.StartOrbit(ctx, connect.NewRequest(&v1.StartOrbitRequest{
			Name:           c.Name,
			SpeedDegPerSec: float32(c.SpeedDegPerSec),
			Orbits:         float32(c.Orbits),
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  orbiting %s at %.2f°/s × %.4g orbits  event_id=%s  status=%s\n",
			c.Name, c.SpeedDegPerSec, c.Orbits, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	// ── Sleep ─────────────────────────────────────────────────────────────────

	case commands.Sleep:
		time.Sleep(time.Duration(c.Seconds * float64(time.Second)))

	// ── HUD ───────────────────────────────────────────────────────────────────

	case commands.HUD:
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(&v1.SetPerformanceRequest{
			State:         &v1.PerformanceState{HudVisible: c.Visible},
			SetHudVisible: true,
		}))
		if err != nil {
			return false, err
		}
		onOff := "off"
		if c.Visible {
			onOff = "on"
		}
		r.printf("ok  hud %s  event_id=%s  status=%s\n", onOff, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
	}
	return false, nil
}

func (r *REPL) runStream(ctx context.Context) error {
	stream, err := r.worldClient.StreamSnapshot(ctx, connect.NewRequest(&v1.StreamSnapshotRequest{}))
	if err != nil {
		return err
	}
	r.printf("streaming — press Ctrl-C to stop\n")
	for stream.Receive() {
		msg := stream.Msg()
		r.printf("t=%.0f  speed=%.4g  bodies=%d\n",
			msg.SimulationTime, msg.Speed, len(msg.Bodies))
	}
	return stream.Err()
}

// oneSnapshot opens a StreamSnapshot RPC, receives the first message, and
// immediately cancels. Returns an empty response (not nil) when no snapshot
// arrives before the stream closes.
func (r *REPL) oneSnapshot(ctx context.Context) (*v1.StreamSnapshotResponse, error) {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := r.worldClient.StreamSnapshot(ctx2, connect.NewRequest(&v1.StreamSnapshotRequest{}))
	if err != nil {
		return nil, err
	}
	if stream.Receive() {
		return stream.Msg(), nil
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &v1.StreamSnapshotResponse{}, nil
}

// refreshBodyNames fetches one snapshot and populates r.bodyNames for TAB
// completion. Sets r.bodyNames to an empty slice (not nil) on error so that
// the caller's nil-sentinel check won't retry on every keypress after failure.
func (r *REPL) refreshBodyNames(ctx context.Context) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snap, err := r.oneSnapshot(ctx2)
	if err != nil || snap == nil {
		r.bodyNames = []string{}
		return
	}
	seen := make(map[string]struct{}, len(snap.Bodies))
	for _, b := range snap.Bodies {
		if b.Name != "" {
			seen[b.Name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	r.bodyNames = names
}

func (r *REPL) runBodies(ctx context.Context, filter string) error {
	snap, err := r.oneSnapshot(ctx)
	if err != nil {
		return err
	}
	if len(snap.Bodies) == 0 {
		r.printf("no bodies in snapshot (server may not be ready)\n")
		return nil
	}

	categoryOrder := []string{"star", "planet", "dwarf_planet", "moon", "asteroid", "ring", "belt"}
	groups := make(map[string][]string)
	for _, b := range snap.Bodies {
		cat := b.Category
		if filter != "" && !strings.EqualFold(cat, filter) {
			continue
		}
		groups[cat] = append(groups[cat], b.Name)
	}

	// Emit known categories in order, then any unknown ones alphabetically.
	seen := make(map[string]bool)
	var extraCats []string
	for cat := range groups {
		seen[cat] = false
	}
	for _, cat := range categoryOrder {
		if _, ok := groups[cat]; ok {
			seen[cat] = true
		}
	}
	for cat := range seen {
		if !seen[cat] {
			extraCats = append(extraCats, cat)
		}
	}
	sort.Strings(extraCats)
	orderedCats := append(categoryOrder, extraCats...)

	const maxShown = 12
	for _, cat := range orderedCats {
		names, ok := groups[cat]
		if !ok {
			continue
		}
		r.printf("%-12s (%d):\n", cat, len(names))
		shown := names
		if len(shown) > maxShown {
			shown = names[:maxShown]
		}
		for _, name := range shown {
			r.printf("  %s\n", name)
		}
		if len(names) > maxShown {
			r.printf("  … and %d more\n", len(names)-maxShown)
		}
	}
	return nil
}

func (r *REPL) runInspect(ctx context.Context, name string) error {
	snap, err := r.oneSnapshot(ctx)
	if err != nil {
		return err
	}
	for _, b := range snap.Bodies {
		if strings.EqualFold(b.Name, name) {
			parent := b.ParentName
			if parent == "" {
				parent = "(none)"
			}
			r.printf("name:      %s\n", b.Name)
			r.printf("category:  %s\n", b.Category)
			r.printf("parent:    %s\n", parent)
			r.printf("position:  x=%.4f  y=%.4f  z=%.4f AU\n", b.PosX, b.PosY, b.PosZ)
			r.printf("radius:    %.6f AU\n", b.PhysicalRadius)
			r.printf("visible:   %v\n", b.Visible)
			return nil
		}
	}
	return fmt.Errorf("body %q not found in snapshot (check spelling or capitalisation)", name)
}

func (r *REPL) runStatus(ctx context.Context) error {
	speedResp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	datasetResp, err := r.simClient.GetDataset(ctx, connect.NewRequest(&v1.GetDatasetRequest{}))
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	timeResp, err := r.simClient.GetSimulationTime(ctx, connect.NewRequest(&v1.GetSimulationTimeRequest{}))
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	sps := speedResp.Msg.SecondsPerSecond
	var speedLabel string
	if sps == 0 {
		speedLabel = "PAUSED"
	} else {
		speedLabel = fmt.Sprintf("%.4g sec/sec", sps)
	}
	dataset := strings.ToLower(strings.TrimPrefix(datasetResp.Msg.Level.String(), "DATASET_LEVEL_"))
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	simDate := j2000.Add(time.Duration(timeResp.Msg.SecondsSinceJ2000 * float64(time.Second)))
	r.printf("speed:    %s\n", speedLabel)
	r.printf("dataset:  %s\n", dataset)
	r.printf("simtime:  %.0f s (J2000) — %s\n", timeResp.Msg.SecondsSinceJ2000, simDate.Format("2006-01-02"))
	if camResp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{})); err == nil {
		target := camResp.Msg.GetCamera().GetTrackTarget()
		if target == "" {
			target = "—"
		}
		r.printf("tracking: %s\n", target)
	}
	return nil
}

func (r *REPL) printHelp() {
	r.printf(`Commands (by category):

Simulation
  setspeed <n>              time multiplier (0 = pause, n >= 0)
  getspeed                  current time multiplier
  pause                     pause the simulation, save current speed
  resume                    resume at last saved speed (default 1.0)

Dataset
  setdataset <level>        asteroid LOD: small | medium | large | huge
  getdataset                active asteroid dataset

World
  gettime                   simulation time (seconds since J2000)
  bodies [<category>]       list visible bodies; optional filter:
                              star | planet | dwarf_planet | moon
                              asteroid | ring | belt
  inspect <name>            position and metadata for a named body
  status                    speed + dataset + sim-date in one view

Streaming
  stream                    live snapshot line per physics frame (Ctrl-C to stop)

System
  system list               list discoverable solar-system files
  system get                show the active system label
  system load <label>       reload the simulation with named system file

Window
  window get                show current window size and state
  window size <W>x<H>       resize the window  e.g. 1920x1080
  window maximize           maximise the window
  window restore            restore from maximised state

Camera
  camera get                show position, orientation, tracking target
  camera center             jump to tracked body, or system origin if free-fly
  camera orient <yaw> <pitch>  set yaw/pitch in degrees
  camera position <x> <y> <z> teleport camera to AU coordinate
  camera track <name>       lock onto a named body (empty = free-fly)

Navigation
  nav stop                  zero all persistent velocity
  nav velocity              show current persistent velocity
  nav forward|back|left|right|up|down <v>  set velocity on that axis (AU/s)
  nav jump <name> [<secs>] [<name> [<secs>] …]  animated multi-hop jump; optional dwell in seconds at each stop

Performance
  perf get                  show all nine performance knobs
  perf set <field> <value>  update one knob
    fields: frustum_culling lod instanced_rendering spatial_partition
            point_rendering importance_threshold use_in_place_swap
            camera_speed workers

Meta
  shutdown                  ask the server to shut down gracefully
  clear                     clear the terminal display
  help                      this message
  quit / exit               exit the REPL

Camera Animation
  orbit <target> <deg/s> <n>  orbit a body at given angular speed for n circuits
                                e.g. orbit Earth 10 2

Timing
  sleep <seconds>           pause script execution  e.g. sleep 2.5

Display
  hud on | hud off          show or hide the heads-up display overlay
`)
}

func (r *REPL) printf(format string, args ...interface{}) {
	fmt.Fprintf(r.out, format, args...)
}

// perfSetField builds a SetPerformanceRequest that updates exactly one field.
// Returns nil when the field name is unrecognised.
func perfSetField(field, value string) *v1.SetPerformanceRequest {
	parseBool := func(v string) bool { return v == "true" || v == "1" || v == "yes" }
	parseInt32 := func(v string) int32 {
		n, _ := strconv.ParseInt(v, 10, 32)
		return int32(n)
	}
	parseFloat32 := func(v string) float32 {
		f, _ := strconv.ParseFloat(v, 32)
		return float32(f)
	}
	st := &v1.PerformanceState{}
	req := &v1.SetPerformanceRequest{State: st}
	switch field {
	case "frustum_culling":
		st.FrustumCulling = parseBool(value)
		req.SetFrustumCulling = true
	case "lod_enabled":
		st.LodEnabled = parseBool(value)
		req.SetLodEnabled = true
	case "instanced_rendering":
		st.InstancedRendering = parseBool(value)
		req.SetInstancedRendering = true
	case "spatial_partition":
		st.SpatialPartition = parseBool(value)
		req.SetSpatialPartition = true
	case "point_rendering":
		st.PointRendering = parseBool(value)
		req.SetPointRendering = true
	case "importance_threshold":
		st.ImportanceThreshold = parseInt32(value)
		req.SetImportanceThreshold = true
	case "use_in_place_swap":
		st.UseInPlaceSwap = parseBool(value)
		req.SetUseInPlaceSwap = true
	case "camera_speed":
		st.CameraSpeed = parseFloat32(value)
		req.SetCameraSpeed = true
	case "workers":
		st.NumWorkers = parseInt32(value)
		req.SetNumWorkers = true
	default:
		return nil
	}
	return req
}

// levelToProto converts a validated lower-case level string to the proto enum.
func levelToProto(level string) v1.DatasetLevel {
	switch level {
	case "small":
		return v1.DatasetLevel_DATASET_LEVEL_SMALL
	case "medium":
		return v1.DatasetLevel_DATASET_LEVEL_MEDIUM
	case "large":
		return v1.DatasetLevel_DATASET_LEVEL_LARGE
	case "huge":
		return v1.DatasetLevel_DATASET_LEVEL_HUGE
	default:
		return v1.DatasetLevel_DATASET_LEVEL_UNSPECIFIED
	}
}

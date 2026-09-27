// Command space-sim is the Space Sim application.
//
// Without --server it runs the simulation in-process (standalone mode).
// With --server it connects to a running space-sim-server as a pure renderer,
// streaming WorldSnapshots over gRPC and sending control commands back via
// SimulationService.
//
// Usage:
//
//	space-sim [flags]
//
// Flags:
//
//	--server host:port   address of a space-sim-server to connect to (enables remote-renderer mode)
//	--sim-speed n        simulated seconds per real second (standalone mode only, default 3600)
//	--system-config path path to system JSON (standalone mode only)
//	--no-textures        disable diffuse texture rendering
//	--no-lighting        disable Phong star lighting shader
//	--no-msaa            disable 4× MSAA anti-aliasing
//	--keybindings path   path to custom keybindings JSON
//	--name label         display name shown to other clients (default: hostname)
//	--performance        run automated performance testing (standalone mode only)
//	--profile name       camera profile for performance testing: 'worst' or 'better'
//	--threads n          number of physics worker threads (standalone mode only)
//	--no-locking         disable double-buffer locking (standalone mode only, unsafe)
//	--debug              enable verbose debug logging (standalone mode only)
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"golang.org/x/net/http2"

	"github.com/google/uuid"

	clientassets "github.com/digital-michael/space_sim/internal/client/assets"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim"
	"github.com/digital-michael/space_sim/internal/sim/engine"

	log "log"
)

func main() {
	serverAddr := flag.String("server", "", "address of space-sim-server (enables remote-renderer mode, e.g. localhost:9090)")
	performanceMode := flag.Bool("performance", false, "run automated performance testing (standalone only)")
	profileFlag := flag.String("profile", "", "camera profile for performance testing: 'worst' or 'better'")
	threadsFlag := flag.Int("threads", 0, "number of physics worker threads (standalone only)")
	noLockingFlag := flag.Bool("no-locking", false, "disable double-buffer locking (standalone only, unsafe)")
	systemConfigFlag := flag.String("system-config", "", "path to system JSON (standalone only)")
	debugFlag := flag.Bool("debug", false, "enable verbose debug logging (standalone only)")
	noTexturesFlag := flag.Bool("no-textures", false, "disable diffuse texture rendering")
	noLightingFlag := flag.Bool("no-lighting", false, "disable Phong star lighting shader")
	noMSAAFlag := flag.Bool("no-msaa", false, "disable 4× MSAA anti-aliasing")
	simSpeedFlag := flag.Float64("sim-speed", 3600, "simulated seconds per real second (standalone only)")
	keybindingsFlag := flag.String("keybindings", "", "path to custom keybindings JSON")
	nameFlag := flag.String("name", "", "display name for this client in multiplayer (default: hostname)")
	flag.Parse()

	profileProvided, threadsProvided, noLockingProvided := false, false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "profile":
			profileProvided = true
		case "threads":
			threadsProvided = true
		case "no-locking":
			noLockingProvided = true
		}
	})

	if (profileProvided || threadsProvided || noLockingProvided) && !*performanceMode {
		fmt.Fprintln(os.Stderr, "error: --profile, --threads, and --no-locking require --performance")
		os.Exit(1)
	}

	const appName = "space-sim"
	appConfigPath := app.DefaultAppConfigPathFor(appName)
	appConfig, err := app.LoadAppConfig(appConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading app config %s: %v\n", appConfigPath, err)
		os.Exit(1)
	}

	cfg := app.Config{
		PerformanceMode: *performanceMode,
		Profile:         *profileFlag,
		Threads:         *threadsFlag,
		NoLocking:       *noLockingFlag,
		SystemConfig:    *systemConfigFlag,
		Debug:           *debugFlag,
		NoTextures:      *noTexturesFlag,
		NoLighting:      *noLightingFlag,
		NoMSAA:          *noMSAAFlag,
		SimTimeScale:    *simSpeedFlag,
		AppConfigPath:   appConfigPath,
		AppConfig:       appConfig,
		KeybindingsPath: *keybindingsFlag,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Standalone mode ───────────────────────────────────────────────────────
	// Content is read from the working directory; no bundle is involved.
	if *serverAddr == "" {
		application, err := app.New(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := application.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// ── Remote-renderer mode ──────────────────────────────────────────────────
	// The simulation runs in space-sim-server. We stream snapshots and render
	// them locally. Simulation-control commands (setspeed, system load, etc.)
	// must still be dispatched over SimulationService gRPC.
	// TODO(F-020): wire sim-control commands back to SimulationService.
	snapSrc := &grpcSnapshotSource{rebuild: make(chan struct{}, 1)}
	snapSrc.setMetadata(metadataIndex{})

	httpClient := h2cClient()

	assetClient := spacesimv1connect.NewAssetServiceClient(
		httpClient, "http://"+*serverAddr, connect.WithGRPC())

	if err := bootstrapContent(ctx, assetClient, snapSrc, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Register a session so this client has an identity other clients can see:
	// a label, a server-assigned palette colour, and a spawn position. A failure
	// here is not fatal — the client can still render the world, it just will not
	// appear to peers — but it must be loud, because silent anonymity is exactly
	// the bug this wiring fixes.
	sessionClient := spacesimv1connect.NewSessionServiceClient(
		httpClient, "http://"+*serverAddr, connect.WithGRPC())

	sessionID := registerSession(ctx, sessionClient, *nameFlag)
	cfg.SessionID = sessionID

	// The renderer needs the asset root, so the app is constructed only after
	// the bundle is in place.
	application, err := app.New(cfg)
	if err != nil {
		unregisterSession(sessionClient, sessionID)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	go streamSnapshots(ctx, *serverAddr, snapSrc, sessionID)
	go publishPresence(ctx, sessionClient, application, snapSrc, sessionID)
	go watchForContentChange(ctx, assetClient, snapSrc)

	runErr := application.RunWithSnapshot(ctx, snapSrc)

	// Release the session before exiting so the server frees the colour slot
	// immediately rather than waiting for a liveness timeout.
	unregisterSession(sessionClient, sessionID)

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", runErr)
		os.Exit(1)
	}
	return
}

// h2cClient returns an HTTP client that speaks cleartext HTTP/2 with prior
// knowledge, which is what gRPC requires.
//
// A default http.Client negotiates HTTP/1.1 against an http:// URL. Unary and
// server-streaming calls tolerate that, but a bidirectional stream cannot be
// carried over a half-duplex connection — SessionStream failed immediately until
// both ends spoke HTTP/2.
func h2cClient() *http.Client {
	return &http.Client{
		Timeout: 0, // streaming calls must not be cut short by a client deadline
		Transport: &http2.Transport{
			AllowHTTP: true,
			// Called for https:// URLs by name, but with AllowHTTP set it is also
			// how a cleartext connection is made — so dial plainly, no TLS.
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}

// ── session identity ──────────────────────────────────────────────────────────

// registerSession claims an identity on the server and returns the assigned
// session id, or "" if registration failed.
func registerSession(ctx context.Context, c spacesimv1connect.SessionServiceClient, nameFlag string) string {
	label := resolveLabel(nameFlag)
	resp, err := c.RegisterClient(ctx, connect.NewRequest(&v1.RegisterClientRequest{
		Version:    1,
		ClientUuid: clientUUID(),
		Label:      label,
		Role:       v1.ClientRole_CLIENT_ROLE_PLAYER,
	}))
	if err != nil {
		log.Printf("WARNING: could not register a session: %v", err)
		log.Printf("WARNING: this client will be invisible to other clients")
		return ""
	}
	rgb := resp.Msg.GetColorRgb()
	if len(rgb) == 3 {
		log.Printf("registered as %q (session %s, colour #%02x%02x%02x)",
			label, short(resp.Msg.GetSessionId()), rgb[0], rgb[1], rgb[2])
	} else {
		log.Printf("registered as %q (session %s)", label, short(resp.Msg.GetSessionId()))
	}
	return resp.Msg.GetSessionId()
}

func unregisterSession(c spacesimv1connect.SessionServiceClient, sessionID string) {
	if sessionID == "" {
		return
	}
	// A fresh context: the run context is typically already cancelled by the time
	// we get here, and this request must still be sent.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.UnregisterClient(ctx, connect.NewRequest(&v1.UnregisterClientRequest{
		Version:   1,
		SessionId: sessionID,
	})); err != nil {
		log.Printf("could not release session %s: %v", short(sessionID), err)
		return
	}
	log.Printf("released session %s", short(sessionID))
}

// resolveLabel picks this client's display name.
//
// Priority is the --name flag, then the hostname. NOTE the hostname fallback
// gives two clients on one machine IDENTICAL labels, which is precisely when a
// multiplayer label is most needed — so a short session suffix is appended by the
// server-side display when labels collide. Pass --name to be unambiguous.
func resolveLabel(nameFlag string) string {
	if nameFlag != "" {
		return truncateLabel(nameFlag)
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return truncateLabel(h)
	}
	return "Player"
}

// truncateLabel enforces the 32-character limit the server also applies, so the
// name logged locally matches the name peers see.
func truncateLabel(s string) string {
	r := []rune(s)
	if len(r) > 32 {
		return string(r[:32])
	}
	return string(r)
}

// clientUUID returns a stable per-user identifier, creating it on first run.
//
// It is persisted so a reconnecting client can be recognised as the same client.
// The server does not yet tie session identity to it — that is part of the
// stable-session-id work — but persisting now means the value exists when it
// does, rather than changing identity on every launch.
func clientUUID() string {
	path := ""
	if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, "space-sim-client-uuid")
		if b, err := os.ReadFile(path); err == nil {
			if id := strings.TrimSpace(string(b)); id != "" {
				return id
			}
		}
	}
	id := uuid.NewString()
	if path != "" {
		if err := os.WriteFile(path, []byte(id+"\n"), 0o644); err != nil {
			log.Printf("could not persist client uuid to %s: %v", path, err)
		}
	}
	return id
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// bootstrapContent fetches the server's content bundle, loads the active system
// out of it, and points both the renderer and the snapshot converter at the
// result. Without this a remote client has geometry but no presentation data and
// renders flat untextured spheres.
func bootstrapContent(
	ctx context.Context,
	client spacesimv1connect.AssetServiceClient,
	snapSrc *grpcSnapshotSource,
	cfg *app.Config,
) error {
	cacheDir, err := clientassets.DefaultCacheDir()
	if err != nil {
		return err
	}
	started := time.Now()
	root, err := clientassets.Ensure(ctx, client, cacheDir)
	if err != nil {
		return fmt.Errorf("obtain content bundle: %w", err)
	}
	log.Printf("content bundle %s ready in %s at %s", root.Hash[:12], time.Since(started).Round(time.Millisecond), root.Path)

	idx, err := buildMetadataIndex(root)
	if err != nil {
		return err
	}
	log.Printf("loaded presentation metadata for %d bodies from %s", len(idx), root.ActiveSystemPath)

	snapSrc.setMetadata(idx)
	snapSrc.setRoot(root)
	cfg.AssetRoot = root.Path
	return nil
}

// ── gRPC snapshot source ──────────────────────────────────────────────────────

type grpcSnapshotSource struct {
	store atomic.Pointer[protocol.WorldSnapshot]

	// meta holds presentation metadata keyed by body name, loaded from the
	// bundled system definition. Swapped atomically when the server switches
	// system, so the render loop never reads a half-updated map.
	meta atomic.Pointer[metadataIndex]

	// root is the bundle the metadata came from, retained so a change check can
	// tell whether anything actually differs.
	root atomic.Pointer[clientassets.Root]

	// rebuild carries a nudge from the stream goroutine when a body arrives that
	// the metadata does not describe. Buffered depth 1: one pending nudge is as
	// informative as ten.
	rebuild chan struct{}
}

func (g *grpcSnapshotSource) LatestSnapshot() protocol.WorldSnapshot {
	if v := g.store.Load(); v != nil {
		return *v
	}
	return protocol.WorldSnapshot{}
}

func (g *grpcSnapshotSource) setMetadata(idx metadataIndex) { g.meta.Store(&idx) }

func (g *grpcSnapshotSource) metadata() metadataIndex {
	if v := g.meta.Load(); v != nil {
		return *v
	}
	return nil
}

func (g *grpcSnapshotSource) setRoot(r *clientassets.Root) { g.root.Store(r) }
func (g *grpcSnapshotSource) currentRoot() *clientassets.Root {
	return g.root.Load()
}

// nudgeRebuild signals that metadata may be stale, without blocking the stream.
func (g *grpcSnapshotSource) nudgeRebuild() {
	select {
	case g.rebuild <- struct{}{}:
	default:
	}
}

// ── presentation metadata ─────────────────────────────────────────────────────

// metadataIndex maps a body name to its presentation metadata: textures,
// atmosphere, luminosity, axial tilt, rotation. None of it is derivable from the
// per-frame wire format, which carries only geometry.
type metadataIndex map[string]engine.ObjectMetadata

// buildMetadataIndex loads the active system definition out of the bundle and
// indexes it by body name.
//
// Since F-034 a system is a directory of per-category files, so this uses
// LoadSystemFromDir; LoadSystemFromFile is the legacy single-document path.
func buildMetadataIndex(root *clientassets.Root) (metadataIndex, error) {
	dir := root.Resolve(root.ActiveSystemPath)
	state, err := sim.LoadSystemFromDir(dir)
	if err != nil {
		return nil, fmt.Errorf("load bundled system %s: %w", dir, err)
	}

	idx := make(metadataIndex, len(state.Objects))
	for _, obj := range state.Objects {
		m := obj.Meta
		// The renderer loads textures by path relative to the working directory,
		// so rewrite bundle-relative paths to absolute ones inside the cache.
		if m.TexturePath != "" {
			m.TexturePath = root.Resolve(m.TexturePath)
		}
		if m.NightTexturePath != "" {
			m.NightTexturePath = root.Resolve(m.NightTexturePath)
		}
		idx[m.Name] = m
	}
	return idx, nil
}

// watchForContentChange rebuilds the metadata index when the server's content or
// active system actually changes. It acts only on a nudge from the stream, so
// there is no polling while the world is stable.
func watchForContentChange(
	ctx context.Context,
	client spacesimv1connect.AssetServiceClient,
	g *grpcSnapshotSource,
) {
	// Procedurally generated bodies are never in the system definition, so
	// nudges can arrive continuously. Rate-limit hard; a system switch is an
	// operator action and a few seconds of stale presentation is invisible.
	const minInterval = 5 * time.Second
	var last time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.rebuild:
		}
		if time.Since(last) < minInterval {
			continue
		}
		last = time.Now()

		info, err := client.GetBundleInfo(ctx, connect.NewRequest(&v1.GetBundleInfoRequest{Version: 1}))
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("content change check failed: %v", err)
			}
			continue
		}

		// Distinguish a real change from unmatched procedural bodies.
		if cur := g.currentRoot(); cur != nil &&
			info.Msg.GetHash() == cur.Hash &&
			info.Msg.GetActiveSystemPath() == cur.ActiveSystemPath {
			continue
		}

		cacheDir, err := clientassets.DefaultCacheDir()
		if err != nil {
			log.Printf("locate cache dir: %v", err)
			continue
		}
		root, err := clientassets.Ensure(ctx, client, cacheDir)
		if err != nil {
			log.Printf("refresh content bundle: %v", err)
			continue
		}
		idx, err := buildMetadataIndex(root)
		if err != nil {
			log.Printf("rebuild metadata: %v", err)
			continue
		}
		g.setMetadata(idx)
		g.setRoot(root)
		log.Printf("content changed — reloaded metadata for %d bodies from %s", len(idx), root.ActiveSystemPath)
	}
}

// describedByDefinition reports whether a category is expected to appear in a
// system definition. Asteroids and belt members are generated at runtime from a
// seed, so their absence from the metadata index is normal and must not be read
// as a signal that the index is stale.
func describedByDefinition(cat engine.ObjectCategory) bool {
	switch cat {
	case engine.CategoryAsteroid, engine.CategoryBelt:
		return false
	default:
		return true
	}
}

func streamSnapshots(ctx context.Context, serverAddr string, g *grpcSnapshotSource, ownSessionID string) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := doStream(ctx, serverAddr, g, ownSessionID); err != nil && ctx.Err() == nil {
			log.Printf("stream error: %v — reconnecting in %s", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			return
		}
	}
}

func doStream(ctx context.Context, serverAddr string, g *grpcSnapshotSource, ownSessionID string) error {
	baseURL := "http://" + serverAddr
	client := spacesimv1connect.NewWorldServiceClient(
		h2cClient(),
		baseURL,
		connect.WithGRPC(),
	)
	stream, err := client.StreamSnapshot(ctx, connect.NewRequest(&v1.StreamSnapshotRequest{Version: 1}))
	if err != nil {
		return fmt.Errorf("StreamSnapshot: %w", err)
	}
	defer stream.Close()
	log.Printf("connected to %s", serverAddr)
	lastPeers := ""
	for stream.Receive() {
		snap, undescribed := protoToSnapshot(stream.Msg(), g.metadata())
		g.store.Store(&snap)
		if undescribed {
			g.nudgeRebuild()
		}
		// Log the peer set whenever it changes. Without this, "I cannot see the
		// other client" is indistinguishable between not-sent, not-received and
		// not-drawn — which cost a full diagnostic cycle to tell apart.
		if sig := peerSignature(snap.ClientSessions, ownSessionID); sig != lastPeers {
			lastPeers = sig
			log.Printf("peers: %s", sig)
		}
	}
	return stream.Err()
}

// ── proto conversion (mirrors space-sim-client) ───────────────────────────────

// protoToSnapshot merges the per-frame wire state with presentation metadata from
// the content bundle. The wire is authoritative for everything it carries —
// identity, position, radius, visibility — and the bundle supplies only what the
// wire cannot: textures, atmosphere, luminosity, tilt, rotation.
//
// The second return value reports whether any body that *should* have been
// described by the system definition was missing from it, which means the
// metadata is stale and worth rechecking.
func protoToSnapshot(resp *v1.StreamSnapshotResponse, idx metadataIndex) (protocol.WorldSnapshot, bool) {
	state := engine.NewSimulationState()
	state.Time = resp.SimulationTime
	state.SecondsPerSecond = resp.Speed

	undescribed := false
	for _, body := range resp.Bodies {
		cat := parseCategory(body.Category)

		meta, known := idx[body.Name]
		if !known {
			// Procedural belt members are legitimately absent; anything else
			// means the definition and the running world disagree.
			if describedByDefinition(cat) {
				undescribed = true
			}
			meta = engine.ObjectMetadata{
				Color:    categoryDefaultColor(cat),
				Material: categoryDefaultMaterial(cat),
			}
		}

		// Wire values overwrite whatever the definition claimed, so a server-side
		// override can never be contradicted by stale local content.
		meta.Name = body.Name
		meta.Category = cat
		meta.ParentName = body.ParentName
		meta.PhysicalRadius = body.PhysicalRadius

		obj := &engine.Object{
			Meta: meta,
			Anim: engine.AnimationState{
				Position: engine.Vector3{
					X: float32(body.PosX),
					Y: float32(body.PosY),
					Z: float32(body.PosZ),
				},
			},
			Visible: body.Visible,
		}
		state.Objects = append(state.Objects, obj)
		state.ObjectMap[obj.Meta.Name] = obj
	}
	state.NavigationOrder = deriveOrder(state.Objects)

	return protocol.WorldSnapshot{
		State:          state,
		Speed:          float64(resp.Speed),
		ClientSessions: protoToSessions(resp.GetClientSessions()),
	}, undescribed
}

// protoToSessions carries peer identities through to the renderer, which already
// knows how to draw their markers and labels — it was simply never given any.
func protoToSessions(in []*v1.ClientSessionInfo) []protocol.ClientSessionSnapshot {
	if len(in) == 0 {
		return nil
	}
	out := make([]protocol.ClientSessionSnapshot, 0, len(in))
	for _, s := range in {
		cs := protocol.ClientSessionSnapshot{
			SessionID: s.GetSessionId(),
			Label:     s.GetLabel(),
			Position:  [3]float64{s.GetPosX(), s.GetPosY(), s.GetPosZ()},
			POV:       [3]float32{s.GetPovX(), s.GetPovY(), s.GetPovZ()},
		}
		if rgb := s.GetColorRgb(); len(rgb) == 3 {
			cs.Color = [3]uint8{rgb[0], rgb[1], rgb[2]}
		} else {
			// A peer with no colour would render black against space; grey is at
			// least visible while making the anomaly obvious.
			cs.Color = [3]uint8{160, 160, 160}
		}
		out = append(out, cs)
	}
	return out
}

// categoryDefaultColor returns a visible fallback color for a body category.
//
// Since F-040 this is the fallback path only: named bodies take their colour from
// the bundled system definition. It still applies to procedurally generated belt
// members, which have no definition entry and no texture.
func categoryDefaultColor(cat engine.ObjectCategory) engine.Color {
	switch cat {
	case engine.CategoryStarPreMain, engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved, engine.CategorySubstellar:
		return engine.Color{R: 255, G: 220, B: 150, A: 255} // warm yellow-white
	case engine.CategoryStellarRemnant:
		return engine.Color{R: 180, G: 220, B: 255, A: 255} // blue-white neutron/remnant
	case engine.CategoryPlanet:
		return engine.Color{R: 70, G: 130, B: 180, A: 255} // steel blue
	case engine.CategoryDwarfPlanet:
		return engine.Color{R: 180, G: 160, B: 130, A: 255} // tan
	case engine.CategoryMoon:
		return engine.Color{R: 160, G: 160, B: 160, A: 255} // gray
	case engine.CategoryAsteroid:
		return engine.Color{R: 110, G: 100, B: 90, A: 255} // dark gray-brown
	case engine.CategoryRing:
		return engine.Color{R: 200, G: 185, B: 155, A: 140} // semi-transparent tan
	default:
		return engine.Color{R: 200, G: 200, B: 200, A: 255} // light gray
	}
}

func categoryDefaultMaterial(cat engine.ObjectCategory) engine.MaterialType {
	switch cat {
	case engine.CategoryStarPreMain, engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved, engine.CategorySubstellar:
		return engine.MaterialEmissive
	case engine.CategoryStellarRemnant:
		return engine.MaterialNeutronStar
	default:
		return engine.MaterialDiffuse
	}
}

func parseCategory(s string) engine.ObjectCategory {
	switch s {
	case "star", "star_main_sequence":
		return engine.CategoryStarMainSequence
	case "star_pre_main":
		return engine.CategoryStarPreMain
	case "star_evolved":
		return engine.CategoryStarEvolved
	case "substellar":
		return engine.CategorySubstellar
	case "stellar_remnant", "blackhole":
		return engine.CategoryStellarRemnant
	case "planet":
		return engine.CategoryPlanet
	case "dwarf_planet":
		return engine.CategoryDwarfPlanet
	case "moon":
		return engine.CategoryMoon
	case "asteroid":
		return engine.CategoryAsteroid
	case "ring":
		return engine.CategoryRing
	case "belt":
		return engine.CategoryBelt
	default:
		return engine.CategoryPlanet
	}
}

func deriveOrder(objects []*engine.Object) []engine.ObjectCategory {
	canonical := []engine.ObjectCategory{
		engine.CategoryStarPreMain, engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved, engine.CategorySubstellar,
		engine.CategoryStellarRemnant, engine.CategoryPlanet,
		engine.CategoryDwarfPlanet, engine.CategoryMoon,
		engine.CategoryAsteroid, engine.CategoryBelt,
		engine.CategoryRogue, engine.CategoryArtifact,
	}
	present := make(map[engine.ObjectCategory]bool, len(objects))
	for _, obj := range objects {
		present[obj.Meta.Category] = true
	}
	order := make([]engine.ObjectCategory, 0, len(canonical))
	for _, cat := range canonical {
		if present[cat] {
			order = append(order, cat)
		}
	}
	return order
}

// peerSignature renders the session set for logging, marking our own entry.
func peerSignature(sessions []protocol.ClientSessionSnapshot, ownSessionID string) string {
	if len(sessions) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(sessions))
	for _, s := range sessions {
		mark := ""
		if ownSessionID != "" && s.SessionID == ownSessionID {
			mark = " (you)"
		}
		parts = append(parts, fmt.Sprintf("%s[%s #%02x%02x%02x]%s",
			s.Label, short(s.SessionID), s.Color[0], s.Color[1], s.Color[2], mark))
	}
	return strings.Join(parts, ", ")
}

// ── presence publishing ───────────────────────────────────────────────────────

const (
	// presenceMaxHz matches the physics tick: publishing faster than the
	// simulation advances cannot convey new information.
	presenceMaxHz = 60.0

	// presenceMinHz is the floor. It doubles as a liveness heartbeat, which is
	// what distinguishes a client that is far away from one that has died.
	presenceMinHz = 1.0

	// presenceFullRateSU is the distance at or below which a peer is close enough
	// that motion must look smooth, so we publish at full rate.
	presenceFullRateSU = 5.0

	// presenceLookaheadS biases the rate by where a mover will be shortly, not
	// where it is. Without it two clients closing quickly are both throttled, so
	// both see stale positions and both react late.
	presenceLookaheadS = 1.0
)

// presenceRateHz returns how often this client should publish its pose, given the
// distance to its nearest peer and its own speed.
//
// The curve is inverse-square, which is how apparent angular size falls off — so
// the rate rises exactly when a peer is large enough on screen for motion to be
// perceptible, and collapses when it is a distant dot. dist may be +Inf, meaning
// no peers, which yields the floor.
func presenceRateHz(dist, speed float64) float64 {
	d := dist - speed*presenceLookaheadS
	if d <= presenceFullRateSU {
		return presenceMaxHz
	}
	rate := presenceMaxHz * (presenceFullRateSU * presenceFullRateSU) / (d * d)
	if rate < presenceMinHz {
		return presenceMinHz
	}
	if rate > presenceMaxHz {
		return presenceMaxHz
	}
	return rate
}

// nearestPeerDistance returns the distance to the closest other session, or +Inf
// when this client is alone.
func nearestPeerDistance(sessions []protocol.ClientSessionSnapshot, ownSessionID string, pos [3]float64) float64 {
	best := math.Inf(1)
	for _, s := range sessions {
		if s.SessionID == ownSessionID {
			continue
		}
		dx := s.Position[0] - pos[0]
		dy := s.Position[1] - pos[1]
		dz := s.Position[2] - pos[2]
		if d := math.Sqrt(dx*dx + dy*dy + dz*dz); d < best {
			best = d
		}
	}
	return best
}

// publishPresence streams this client's pose to the server at a distance-adaptive
// rate, so an isolated or distant client costs almost nothing while a close
// encounter is smooth.
func publishPresence(
	ctx context.Context,
	c spacesimv1connect.SessionServiceClient,
	ap *app.App,
	g *grpcSnapshotSource,
	sessionID string,
) {
	if sessionID == "" {
		return // unregistered: nothing to attribute a pose to
	}

	stream := c.SessionStream(ctx)
	defer func() { _ = stream.CloseRequest() }()

	// The server pushes session deltas on this stream. We take peer state from the
	// world snapshot instead, but the receive side must still be drained or the
	// server's sends would eventually block.
	go func() {
		for {
			if _, err := stream.Receive(); err != nil {
				return
			}
		}
	}()

	// Tick at the ceiling and gate each send on the adaptive interval; the ticker
	// itself is far cheaper than the RPC it guards.
	ticker := time.NewTicker(time.Second / presenceMaxHz)
	defer ticker.Stop()

	var lastSent time.Time
	var lastHz float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		pose, ok := ap.CurrentPose()
		if !ok {
			continue
		}

		dist := nearestPeerDistance(g.LatestSnapshot().ClientSessions, sessionID, pose.Position)
		hz := presenceRateHz(dist, pose.Speed)
		if time.Since(lastSent) < time.Duration(float64(time.Second)/hz) {
			continue
		}
		lastSent = time.Now()

		if err := stream.Send(&v1.ClientUpdate{
			Version:   1,
			SessionId: sessionID,
			PosX:      pose.Position[0],
			PosY:      pose.Position[1],
			PosZ:      pose.Position[2],
			PovX:      pose.Forward[0],
			PovY:      pose.Forward[1],
			PovZ:      pose.Forward[2],
		}); err != nil {
			if ctx.Err() == nil {
				log.Printf("presence stream ended: %v", err)
			}
			return
		}

		// Log only on a material rate change, so the adaptive behaviour is
		// observable without flooding at 60 Hz.
		if lastHz == 0 || hz/lastHz > 2 || lastHz/hz > 2 {
			log.Printf("presence rate %.1f Hz (nearest peer %.1f su)", hz, dist)
			lastHz = hz
		}
	}
}

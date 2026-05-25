package render

import (
	"math"
	"runtime"
	"time"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// modAlt is the OS-appropriate label for the Alt/Option modifier key.
// modSuper is the OS-appropriate label for the Super/Command modifier key.
var (
	modAlt   string
	modSuper string
)

func init() {
	switch runtime.GOOS {
	case "darwin":
		modAlt = "Opt"
		modSuper = "Cmd"
	case "windows":
		modAlt = "Alt"
		modSuper = "Win"
	default: // linux and others
		modAlt = "Alt"
		modSuper = "Super"
	}
}

var layoutWidth int32
var layoutHeight int32

const hideDateAtOrAboveSecondsPerSecond = 86400.0

// Renderer owns Raylib-specific drawing behavior for the Space Sim application.
type Renderer struct {
	renderWidth  int32
	renderHeight int32
	target       rl.RenderTexture2D
	targetLoaded bool

	// Texture and model resources (lazily loaded after GL context is ready).
	// noTextures disables all diffuse texture rendering.
	noTextures     bool
	textureCache   map[string]rl.Texture2D // path → GPU texture
	modelCache     map[modelKey]rl.Model   // (path,rings,slices) → textured sphere model
	ringImageCache map[string]*rl.Image    // path → CPU-side ring colour strip image

	// noLighting disables the Phong star lighting shader; bodies render with
	// the default Raylib shader (flat diffuse texture, no shadowing).
	noLighting bool
	lighting   lightingState

	// atmosphere holds the rim-glow + day/night shader; atmoSphere is a
	// lazily-loaded unit-sphere model reused for all atmosphere draw calls.
	atmosphere       atmosphereState
	atmoSphere       rl.Model
	atmoSphereLoaded bool

	// infraMode is the current infra ambient-light mode:
	//   0 = off, 1 = spotlight (FOV-centre ambient boost), 2 = reserved.
	// cameraForward is the normalised camera look direction, updated each frame
	// via SetInfraState so the spotlight tracks the viewpoint.
	infraMode     int
	cameraForward engine.Vector3

	// sky holds the Milky Way background skysphere. The sphere is centred at the
	// floating-origin camera (always (0,0,0)), so it rotates with camera orientation
	// but never translates — producing correct parallax-free sky behaviour.
	skyTex         rl.Texture2D
	skyModel       rl.Model
	skyLoaded      bool // true once load was attempted (even if it failed)
	skyModelLoaded bool // true if the model is ready to draw
}

// modelKey indexes the model cache.
type modelKey struct {
	path          string
	rings, slices int32
}

// New creates a Raylib renderer.
func New(noTextures, noLighting bool) *Renderer {
	setLayoutSize(0, 0)
	return &Renderer{
		noTextures:     noTextures,
		noLighting:     noLighting,
		textureCache:   make(map[string]rl.Texture2D),
		modelCache:     make(map[modelKey]rl.Model),
		ringImageCache: make(map[string]*rl.Image),
	}
}


func setLayoutSize(width, height int32) {
	layoutWidth = width
	layoutHeight = height
}

func currentScreenWidth() int {
	if layoutWidth > 0 {
		return int(layoutWidth)
	}
	return rl.GetScreenWidth()
}

func currentScreenHeight() int {
	if layoutHeight > 0 {
		return int(layoutHeight)
	}
	return rl.GetScreenHeight()
}



// activeUIScale is the current UI scaling factor. Updated via SetUIScale.
var activeUIScale float32 = 1.0

// SetUIScale sets the UI scaling factor applied by scaledInt32 throughout the
// settings dialog and UI chrome. Call at startup from the loaded AppConfig and
// whenever the user changes the value in the Display settings tab.
func SetUIScale(f float32) {
	if f <= 0 {
		f = 1.0
	}
	activeUIScale = f
}

func uiScale() float32 {
	return activeUIScale
}

func scaledInt32(base int32) int32 {
	scaled := int32(math.Round(float64(float32(base) * uiScale())))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func (r *Renderer) HasRenderTarget() bool {
	return r.targetLoaded
}

func (r *Renderer) DisableRenderTarget() {
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
		r.targetLoaded = false
	}
	setLayoutSize(0, 0)
}

func (r *Renderer) ConfigureRenderTarget(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}
	if r.targetLoaded && r.renderWidth == width && r.renderHeight == height {
		setLayoutSize(width, height)
		return
	}
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
	}
	r.target = rl.LoadRenderTexture(width, height)
	r.targetLoaded = true
	r.renderWidth = width
	r.renderHeight = height
	rl.SetTextureFilter(r.target.Texture, rl.FilterBilinear)
	setLayoutSize(width, height)
}


func (r *Renderer) BeginFrame() {
	if r.targetLoaded {
		setLayoutSize(r.renderWidth, r.renderHeight)
		rl.BeginTextureMode(r.target)
	} else {
		setLayoutSize(int32(rl.GetRenderWidth()), int32(rl.GetRenderHeight()))
		rl.BeginDrawing()
	}
	rl.ClearBackground(rl.Black)
}

func (r *Renderer) EndFrame(windowWidth, windowHeight int32) {
	if !r.targetLoaded {
		rl.EndDrawing()
		return
	}

	rl.EndTextureMode()
	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	scaleX := float32(windowWidth) / float32(r.renderWidth)
	scaleY := float32(windowHeight) / float32(r.renderHeight)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	destWidth := float32(r.renderWidth) * scale
	destHeight := float32(r.renderHeight) * scale
	destX := (float32(windowWidth) - destWidth) * 0.5
	destY := (float32(windowHeight) - destHeight) * 0.5

	source := rl.Rectangle{X: 0, Y: 0, Width: float32(r.renderWidth), Height: -float32(r.renderHeight)}
	dest := rl.Rectangle{X: destX, Y: destY, Width: destWidth, Height: destHeight}
	rl.DrawTexturePro(r.target.Texture, source, dest, rl.Vector2{}, 0, rl.White)
	rl.EndDrawing()
}

// CaptureRenderTexture reads pixels from the render texture by creating a
// temporary read-only OpenGL framebuffer attached to the texture. This works
// reliably on Apple Silicon (OpenGL via Metal) regardless of FBO binding state.
// The result is bottom-up (OpenGL origin); feed ffmpeg with -vf vflip.
// Can be called before or after EndTextureMode. Must be called on the GL thread.
//
// Only pixels drawn inside BeginFrame/EndFrame are captured. Anything drawn
// directly to the default framebuffer (outside that window) is not present in
// the render texture and will be absent from the returned bytes.
// Returns nil when no render texture is active (native mode or not yet loaded).
func (r *Renderer) CaptureRenderTexture() []byte {
	if !r.targetLoaded {
		return nil
	}
	return readTexturePixels(r.target.Texture.ID, int(r.renderWidth), int(r.renderHeight))
}

func (r *Renderer) DrawObjectsInstanced(objects []*engine.Object, cameraPos engine.Vector3, simTime float64, pointRenderingEnabled bool, lodEnabled bool, importanceThreshold int) int {
	return drawObjectsInstanced(r, objects, cameraPos, simTime, pointRenderingEnabled, lodEnabled, importanceThreshold)
}

func (r *Renderer) DrawObject(obj *engine.Object, cameraPos engine.Vector3, simTime float64, pointRenderingEnabled bool, lodEnabled bool) {
	drawObject(r, obj, cameraPos, simTime, pointRenderingEnabled, lodEnabled)
}

// UpdateLights uploads the current star light data to the Phong shader so all
// subsequent DrawObjectsInstanced / DrawObject calls this frame are correctly
// lit. Pass the full state.Objects slice (not frustum-culled) so that stars
// off-screen still illuminate the scene. No-op when noLighting is true.
func (r *Renderer) UpdateLights(objects []*engine.Object, cameraPos engine.Vector3) {
	if r.noLighting {
		return
	}
	r.lighting.load()
	r.lighting.setLights(objects, cameraPos)
}

// SetInfraState records the current infra mode and camera forward direction.
// Must be called each frame before any draw calls when infraMode may change.
// cameraForward must be normalised.
func (r *Renderer) SetInfraState(mode int, cameraForward engine.Vector3) {
	r.infraMode = mode
	r.cameraForward = cameraForward
}

func (r *Renderer) DrawGroundPlane() {
	drawGroundPlane()
}

func (r *Renderer) DrawObjectLabels(state *engine.SimulationState, cameraState *ui.CameraState, camera rl.Camera3D, objectsToRender []*engine.Object, mode ui.LabelMode, screenW, screenH int32) {
	drawObjectLabels(state, cameraState, camera, objectsToRender, mode, screenW, screenH)
}

func (r *Renderer) DrawHUD(state *engine.SimulationState, cameraState *ui.CameraState, inputState *ui.InputState, asteroidDataset engine.AsteroidDataset, mouseModeEnabled bool, speed float64, inViewCount int, eligibleInViewCount int, renderedCount int, showDebug bool, showInfo bool, showHelp bool, labelMode ui.LabelMode) {
	if showDebug {
		drawHUDDebug(state, cameraState, asteroidDataset, speed, inViewCount, eligibleInViewCount, renderedCount, labelMode)
	}
	if showInfo {
		drawHUDInfo(state, cameraState, inputState)
	}
	if showHelp {
		drawHUDHelp()
	}
}

func (r *Renderer) DrawZoomIndicator(zoomValue float32) {
	drawZoomIndicator(zoomValue)
}

func (r *Renderer) DrawWelcomeBanner(text string, elapsed time.Duration) {
	drawWelcomeBanner(text, elapsed)
}


// DrawSettingsDialog draws the unified 4-tab settings dialog.
func (r *Renderer) DrawSettingsDialog(state *ui.SettingsState, km *input.KeyMap) {
	drawSettingsDialog(state, km)
}


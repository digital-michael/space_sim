package render

// Phong lighting for space sim — inverse-square falloff from self-luminous stars.
//
// Shader uniforms set per-frame:
//
//	lightCount         int        number of active lights (0..maxLights)
//	lightPos[i]        vec3       camera-relative world position of light i
//	lightColor[i]      vec3       normalised RGB emission colour of light i
//	lightIntensity[i]  float      SolarLuminosity of light i
//	lightScale         float      global intensity multiplier (tuning knob)
//	ambient            float      minimum surface brightness

import (
	"fmt"
	"unsafe"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const maxLights = 4

// defaultLightScale converts SolarLuminosity / dist² into a [0,1] fragment
// contribution. Earth (dist≈100) from Sol (luminosity=1.0):
//
//	1.0 * 9000 / (100 * 100) = 0.9  → ~90% lit-side brightness, strong day/night contrast.
const defaultLightScale = float32(9000)

// defaultAmbient is the minimum surface brightness on the night side.
const defaultAmbient = float32(0.02)

// phongVS is the GLSL 330 vertex shader. Passes world-space position and
// corrected normal to the fragment stage. Raylib auto-populates mvp, matModel,
// and matNormal (transpose-inverse of model matrix) from the draw call.
const phongVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;
out vec3 fragPos;
out vec2 fragTexCoord;
out vec3 fragNormal;
void main() {
    fragPos       = vec3(matModel * vec4(vertexPosition, 1.0));
    fragTexCoord  = vertexTexCoord;
    fragNormal    = normalize(vec3(matNormal * vec4(vertexNormal, 0.0)));
    gl_Position   = mvp * vec4(vertexPosition, 1.0);
}
`

// phongFS is the GLSL 330 fragment shader. Sums inverse-square Lambert
// contributions from each star light and adds a small ambient floor.
// When hasNightTexture == 1, texture1 (city lights) is blended on the dark side.
const phongFS = `#version 330
in vec3 fragPos;
in vec2 fragTexCoord;
in vec3 fragNormal;
uniform sampler2D texture0;    // diffuse (day) texture
uniform sampler2D texture1;    // night-side emission (city lights); only sampled when hasNightTexture==1
uniform int       hasNightTexture;
uniform vec4 colDiffuse;
#define MAX_LIGHTS 4
uniform int   lightCount;
uniform vec3  lightPos[MAX_LIGHTS];
uniform vec3  lightColor[MAX_LIGHTS];
uniform float lightIntensity[MAX_LIGHTS];
uniform float lightScale;
uniform float ambient;
out vec4 finalColor;
void main() {
    vec4 texSample    = texture(texture0, fragTexCoord);
    vec3 surfaceColor = texSample.rgb * colDiffuse.rgb;
    vec3 norm         = normalize(fragNormal);
    vec3 diffuse      = vec3(0.0);
    float maxDiff     = 0.0;
    for (int i = 0; i < lightCount; i++) {
        vec3  toLight = lightPos[i] - fragPos;
        float dist2   = max(dot(toLight, toLight), 0.0001);
        float diff    = max(dot(norm, normalize(toLight)), 0.0);
        diffuse += lightColor[i] * diff * (lightIntensity[i] * lightScale / dist2);
        maxDiff = max(maxDiff, diff);
    }
    vec3 result = clamp((ambient + diffuse) * surfaceColor, 0.0, 1.0);
    // City lights: blend night texture on the unlit side, fading at the terminator.
    if (hasNightTexture == 1) {
        vec3 nightSample = texture(texture1, fragTexCoord).rgb;
        float nightBlend = 1.0 - smoothstep(0.0, 0.15, maxDiff);
        result += nightSample * nightBlend * 0.8;
    }
    finalColor = vec4(clamp(result, 0.0, 1.0), texSample.a * colDiffuse.a);
}
`

// atmoVS is the vertex shader for the atmosphere rim-glow effect.
// It passes camera-relative world position and corrected normal to the fragment stage.
const atmoVS = `#version 330
in vec3 vertexPosition;
in vec3 vertexNormal;
uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;
out vec3 fragPos;
out vec3 fragNormal;
void main() {
    fragPos    = vec3(matModel * vec4(vertexPosition, 1.0));
    fragNormal = normalize(vec3(matNormal * vec4(vertexNormal, 0.0)));
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

// atmoFS is the fragment shader for the atmosphere rim-glow effect.
// Fresnel rim term: transparent face-on, bright at the limb.
// Lambert sun-facing term: day side lit, night side dim (8% ambient scatter floor).
// Self-luminous bodies (stars) bypass the Lambert term — their corona glows uniformly.
// Output is used with BlendAddColors — alpha channel is ignored by the blend equation.
const atmoFS = `#version 330
in vec3 fragPos;
in vec3 fragNormal;
uniform vec3  lightPos;      // primary star, camera-relative world space
uniform vec4  glowColor;     // RGB tint; alpha is an intensity weight
uniform float glowEdge;      // Fresnel exponent (higher = narrower limb halo)
uniform int   selfLuminous;  // 1 if this body emits its own light (star); 0 otherwise
out vec4 finalColor;
void main() {
    vec3 viewDir = normalize(-fragPos);   // camera is always at origin in cam-relative space
    vec3 norm    = normalize(fragNormal);
    // Fresnel rim: 0 when facing camera, 1 at 90-degree grazing angle
    float rim = pow(max(1.0 - abs(dot(norm, viewDir)), 0.0), glowEdge);
    // Lambert sun-facing: bright on day side, near-zero on night side.
    // Self-luminous bodies skip this — the star illuminates its own corona from
    // within, so the glow is at full intensity all the way around the limb.
    float litFactor;
    if (selfLuminous == 1) {
        litFactor = 1.0;
    } else {
        vec3 toLight = normalize(lightPos - fragPos);
        litFactor = mix(0.08, 1.0, max(dot(norm, toLight), 0.0));
    }
    finalColor = vec4(glowColor.rgb * glowColor.a * rim * litFactor, 1.0);
}
`

// bhFS is the fragment shader for the black hole visual effect.
// Renders on a large sphere (~7× event horizon radius) centred at the BH.
// The camera sits at the origin (floating-origin coordinate system), so
// viewDir = normalize(-fragPos).
//
// sinTheta (screen-space normalised radius) drives all ring computations:
//   0 = centre of BH disk  →  1 = limb of the glow sphere.
//
// innerFrac = (2.6 × R_bh) / R_outer  — the critical-impact-parameter radius
// beyond which captured photons appear; the shadow edge / photon ring lives here.
// intensity scales brightness with BH mass (passed per-object from Go).
const bhFS = `#version 330
in vec3 fragPos;
in vec3 fragNormal;
uniform float innerFrac;
uniform float intensity;
out vec4 finalColor;

vec3 accrColor(float t) {
    vec3 hot  = vec3(1.00, 0.96, 0.78);
    vec3 mid  = vec3(1.00, 0.42, 0.06);
    vec3 cool = vec3(0.52, 0.08, 0.02);
    if (t < 0.5) return mix(hot, mid, t * 2.0);
    return mix(mid, cool, (t - 0.5) * 2.0);
}

void main() {
    vec3 viewDir = normalize(-fragPos);
    vec3 n       = normalize(fragNormal);
    float cosT   = dot(n, viewDir);
    if (cosT < 0.0) { finalColor = vec4(0.0); return; }
    float r = sqrt(max(0.0, 1.0 - cosT * cosT));

    // Shadow interior: output nothing; solid BH sphere and starfield show through.
    if (r < innerFrac * 0.91) { finalColor = vec4(0.0); return; }

    // Photon ring: narrow bright band at the shadow edge.
    float ringW = innerFrac * 0.22;
    float ringD = abs(r - innerFrac);
    float ringT = pow(max(0.0, 1.0 - ringD / ringW), 1.25);
    vec3 ringCol = mix(vec3(1.0, 0.97, 0.82), vec3(1.0, 0.62, 0.20), ringD / (ringW + 0.001) * 0.6);
    ringCol *= ringT * intensity * 5.8;

    // Accretion disk: temperature gradient from shadow edge outward.
    float aEnd   = 0.70;
    float aT     = clamp((r - innerFrac) / max(aEnd - innerFrac, 0.001), 0.0, 1.0);
    float aMask  = step(innerFrac, r) * (1.0 - step(aEnd, r));
    float aFade  = pow(1.0 - aT, 1.35);
    vec3  aCol   = accrColor(aT) * aFade * aMask * intensity * 2.4;

    // Relativistic corona: faint blue-purple halo beyond the disk.
    float hFrac  = max(0.0, r - innerFrac) / max(0.95 - innerFrac, 0.001);
    float haze   = max(0.0, 1.0 - hFrac * 1.6) * step(innerFrac, r) * intensity * 0.32;
    vec3  hCol   = vec3(0.16, 0.26, 0.78) * haze;

    vec3  col = ringCol + aCol + hCol;
    float a   = clamp(length(col) * 0.72, 0.0, 1.0);
    finalColor = vec4(col, a);
}
`

// bhShaderState owns the black hole visual shader.
type bhShaderState struct {
	shader       rl.Shader
	loaded       bool
	locInnerFrac int32
	locIntensity int32
}

func (bh *bhShaderState) load() {
	if bh.loaded {
		return
	}
	bh.shader = rl.LoadShaderFromMemory(atmoVS, bhFS)
	bh.locInnerFrac = rl.GetShaderLocation(bh.shader, "innerFrac")
	bh.locIntensity = rl.GetShaderLocation(bh.shader, "intensity")
	bh.loaded = true
}

func (bh *bhShaderState) unload() {
	if bh.loaded {
		rl.UnloadShader(bh.shader)
		bh.loaded = false
	}
}

// atmosphereState owns the rim-glow + day/night atmosphere shader.
type atmosphereState struct {
	shader          rl.Shader
	loaded          bool
	locLightPos     int32
	locGlowColor    int32
	locGlowEdge     int32
	locSelfLuminous int32
}

// load compiles the atmosphere shader and caches uniform locations. Idempotent.
func (as *atmosphereState) load() {
	if as.loaded {
		return
	}
	as.shader = rl.LoadShaderFromMemory(atmoVS, atmoFS)
	as.locLightPos = rl.GetShaderLocation(as.shader, "lightPos")
	as.locGlowColor = rl.GetShaderLocation(as.shader, "glowColor")
	as.locGlowEdge = rl.GetShaderLocation(as.shader, "glowEdge")
	as.locSelfLuminous = rl.GetShaderLocation(as.shader, "selfLuminous")
	as.loaded = true
}

// unload releases the GPU shader. Safe to call when not loaded.
func (as *atmosphereState) unload() {
	if as.loaded {
		rl.UnloadShader(as.shader)
		as.loaded = false
	}
}

// lightingState owns the Phong shader and its cached uniform locations.
type lightingState struct {
	shader rl.Shader
	loaded bool

	locCount           int32
	locPos             [maxLights]int32
	locColor           [maxLights]int32
	locIntensity       [maxLights]int32
	locScale           int32
	locAmbient         int32
	locHasNightTexture int32

	// PrimaryLightPos is the camera-relative position of the brightest star,
	// captured each frame by setLights for use by the atmosphere shader.
	PrimaryLightPos [3]float32
}

// load compiles the Phong shader and caches uniform locations. Idempotent.
// Must be called after the OpenGL context is initialised (i.e. during a frame).
func (ls *lightingState) load() {
	if ls.loaded {
		return
	}
	ls.shader = rl.LoadShaderFromMemory(phongVS, phongFS)
	ls.locCount = rl.GetShaderLocation(ls.shader, "lightCount")
	for i := 0; i < maxLights; i++ {
		ls.locPos[i] = rl.GetShaderLocation(ls.shader, fmt.Sprintf("lightPos[%d]", i))
		ls.locColor[i] = rl.GetShaderLocation(ls.shader, fmt.Sprintf("lightColor[%d]", i))
		ls.locIntensity[i] = rl.GetShaderLocation(ls.shader, fmt.Sprintf("lightIntensity[%d]", i))
	}
	ls.locScale = rl.GetShaderLocation(ls.shader, "lightScale")
	ls.locAmbient = rl.GetShaderLocation(ls.shader, "ambient")
	ls.locHasNightTexture = rl.GetShaderLocation(ls.shader, "hasNightTexture")

	// Set static defaults once; they persist until explicitly changed.
	rl.SetShaderValue(ls.shader, ls.locScale, []float32{defaultLightScale}, rl.ShaderUniformFloat)
	rl.SetShaderValue(ls.shader, ls.locAmbient, []float32{defaultAmbient}, rl.ShaderUniformFloat)

	ls.loaded = true
}

// unload releases the GPU shader. Safe to call when not loaded.
func (ls *lightingState) unload() {
	if ls.loaded {
		rl.UnloadShader(ls.shader)
		ls.loaded = false
	}
}

// setAmbient updates the ambient floor uniform on the live shader.
// Used by the infra spotlight to boost ambient per-object.
func (ls *lightingState) setAmbient(v float32) {
	if !ls.loaded {
		return
	}
	rl.SetShaderValue(ls.shader, ls.locAmbient, []float32{v}, rl.ShaderUniformFloat)
}

// setLights uploads the current star light data to the shader uniforms.
// objects is the full scene list (all objects, not just frustum-visible ones)
// so that star lighting remains correct even when the star is off-screen.
// cameraPos must match the camera offset used in DrawModel position arguments
// (all drawn positions are obj.Anim.Position - cameraPos).
func (ls *lightingState) setLights(objects []*engine.Object, cameraPos engine.Vector3) {
	count := int32(0)
	for _, obj := range objects {
		if !obj.Meta.SelfLuminous || count >= maxLights {
			continue
		}

		pos := []float32{
			obj.Anim.Position.X - cameraPos.X,
			obj.Anim.Position.Y - cameraPos.Y,
			obj.Anim.Position.Z - cameraPos.Z,
		}

		// Normalise emission colour; default to warm white when unset.
		r := float32(obj.Meta.EmissionColor.R) / 255.0
		g := float32(obj.Meta.EmissionColor.G) / 255.0
		b := float32(obj.Meta.EmissionColor.B) / 255.0
		if r == 0 && g == 0 && b == 0 {
			r, g, b = 1.0, 1.0, 0.95
		}

		// SolarLuminosity=0 means unset; default to 1 (solar equivalent).
		intensity := obj.Meta.SolarLuminosity
		if intensity <= 0 {
			intensity = 1.0
		}

		rl.SetShaderValue(ls.shader, ls.locPos[count], pos, rl.ShaderUniformVec3)
		rl.SetShaderValue(ls.shader, ls.locColor[count], []float32{r, g, b}, rl.ShaderUniformVec3)
		rl.SetShaderValue(ls.shader, ls.locIntensity[count], []float32{intensity}, rl.ShaderUniformFloat)
		// Capture the primary (first) light position for the atmosphere shader.
		if count == 0 {
			ls.PrimaryLightPos = [3]float32{pos[0], pos[1], pos[2]}
		}
		count++
	}

	// SetShaderValue takes []float32 for all types; reinterpret int32 bits.
	countF := *(*float32)(unsafe.Pointer(&count))
	rl.SetShaderValue(ls.shader, ls.locCount, []float32{countF}, rl.ShaderUniformInt)
}

// applyToModel sets the lighting shader on the first material of model and
// optionally binds a night-side emission texture (city lights). Pass a zero
// Texture2D (ID == 0) when the body has no night texture.
// Self-luminous (star) bodies skip this so they always render at full
// brightness regardless of scene lighting.
func (ls *lightingState) applyToModel(model rl.Model, selfLuminous bool, nightTex rl.Texture2D) {
	if !ls.loaded || selfLuminous {
		return
	}
	mats := model.GetMaterials()
	if len(mats) == 0 {
		return
	}
	mats[0].Shader = ls.shader
	// Bind night texture to MAP_SPECULAR (slot 1 → texture1 in GLSL).
	// Raylib binds material.maps[1] to texture unit 1 when drawing with a custom shader.
	mats[0].GetMap(rl.MapSpecular).Texture = nightTex
	// Upload hasNightTexture flag (int uniform via bit-reinterpretation).
	hasNight := int32(0)
	if nightTex.ID > 0 {
		hasNight = 1
	}
	hasNightF := *(*float32)(unsafe.Pointer(&hasNight))
	rl.SetShaderValue(ls.shader, ls.locHasNightTexture, []float32{hasNightF}, rl.ShaderUniformInt)
}

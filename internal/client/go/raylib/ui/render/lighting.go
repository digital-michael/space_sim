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

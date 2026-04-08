package render

// readCurrentFBOPixels reads RGBA pixels from the currently-bound OpenGL
// framebuffer using glReadPixels. This is reliable on all platforms including
// Apple Silicon where glGetTexImage (used by rl.LoadImageFromTexture) silently
// returns empty data for render textures.
//
// The result is bottom-up (OpenGL origin). Pass to ffmpeg with -vf vflip.
// Must be called on the GL/main thread while the FBO is still bound
// (i.e. before EndTextureMode / EndFrame).

/*
#include <stdlib.h>

#if defined(__APPLE__)
#define GL_SILENCE_DEPRECATION
#include <OpenGL/gl3.h>
#elif defined(_WIN32)
#include <windows.h>
#include <GL/gl.h>
#else
#define GL_GLEXT_PROTOTYPES
#include <GL/gl.h>
#include <GL/glext.h>
#endif

static unsigned char* captureCurrentFBO(int width, int height) {
    unsigned char* pixels = (unsigned char*)malloc((size_t)width * (size_t)height * 4);
    if (!pixels) return NULL;
    glReadPixels(0, 0, width, height, GL_RGBA, GL_UNSIGNED_BYTE, pixels);
    return pixels;
}
*/
import "C"

import "unsafe"

func readCurrentFBOPixels(width, height int) []byte {
	if width <= 0 || height <= 0 {
		return nil
	}
	ptr := C.captureCurrentFBO(C.int(width), C.int(height))
	if ptr == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(ptr))
	sz := width * height * 4
	out := make([]byte, sz)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(ptr)), sz))
	return out
}

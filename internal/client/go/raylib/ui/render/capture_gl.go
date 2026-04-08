package render

// captureTextureViaFBO reads RGBA pixels from a Raylib render texture by
// creating a temporary read-only OpenGL framebuffer and attaching the texture.
//
// This is the only reliable technique on Apple Silicon (OpenGL via Metal) where
// glReadPixels from a currently-bound Raylib FBO silently returns empty data,
// and glGetTexImage (used by rl.LoadImageFromTexture) is broken entirely.
//
// Result is bottom-up (OpenGL origin) — feed to ffmpeg with -vf vflip.
// No Raylib FBO-binding state required; can be called any time on the GL thread.

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

static unsigned char* captureTextureViaFBO(unsigned int textureId, int width, int height) {
    // Flush any pending GL errors before we start.
    while (glGetError() != GL_NO_ERROR) {}

    size_t sz = (size_t)width * (size_t)height * 4;
    unsigned char* pixels = (unsigned char*)malloc(sz);
    if (!pixels) return NULL;

    // Save previous framebuffer binding so we can restore it.
    GLint prevFBO = 0;
    glGetIntegerv(GL_FRAMEBUFFER_BINDING, &prevFBO);

    // Create a temporary FBO and attach the render texture as colour read.
    GLuint fbo;
    glGenFramebuffers(1, &fbo);
    glBindFramebuffer(GL_FRAMEBUFFER, fbo);
    glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                           GL_TEXTURE_2D, (GLuint)textureId, 0);

    GLenum status = glCheckFramebufferStatus(GL_FRAMEBUFFER);
    if (status == GL_FRAMEBUFFER_COMPLETE) {
        glReadPixels(0, 0, (GLsizei)width, (GLsizei)height,
                     GL_RGBA, GL_UNSIGNED_BYTE, pixels);
        if (glGetError() != GL_NO_ERROR) {
            free(pixels);
            pixels = NULL;
        }
    } else {
        free(pixels);
        pixels = NULL;
    }

    // Restore previous FBO and clean up.
    glBindFramebuffer(GL_FRAMEBUFFER, (GLuint)prevFBO);
    glDeleteFramebuffers(1, &fbo);

    return pixels;
}
*/
import "C"

import "unsafe"

func readTexturePixels(textureID uint32, width, height int) []byte {
	if width <= 0 || height <= 0 {
		return nil
	}
	ptr := C.captureTextureViaFBO(C.uint(textureID), C.int(width), C.int(height))
	if ptr == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(ptr))
	sz := width * height * 4
	out := make([]byte, sz)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(ptr)), sz))
	return out
}

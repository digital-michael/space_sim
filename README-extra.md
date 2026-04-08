# Space Sim — Optional Features & Platform Notes

Optional dependencies and platform-specific setup for features that are not
required to build or run the simulator.

---

## Video Recording

Space Sim uses **ffmpeg** to encode recorded frames into H.264 MP4 files.
ffmpeg is a standalone command-line program — not a library linked into the
binary. The simulator spawns it as a subprocess at the moment you start
recording and pipes raw RGBA frame data to it in the background.

**ffmpeg is not required to build or run the simulator.** If it is missing when
you press Opt+R, you will see a `[REC] Error:` message in the console and
recording will simply not start. Everything else continues to work normally.

### Keys

| Key | Action |
|-----|--------|
| Opt+R | Start recording / Stop recording |
| Opt+Shift+R | Pause recording (freeze-frame) / Resume recording |

While recording is active a **`* REC`** badge appears in the top-right corner
of the window. While paused it shows **`\|\| REC`** in yellow.

Output is saved to `~/Desktop/space-sim-<YYYY-MM-DD_HH-MM-SS>.mp4`.

---

### macOS (this machine)

ffmpeg is a Homebrew formula — one command, no compilation, no admin rights
needed beyond what you already have for Homebrew.

```bash
brew install ffmpeg
```

Verify the install:

```bash
ffmpeg -version   # should print "ffmpeg version 7.x ..."
```

**Apple Silicon performance tip:** The default recording uses `libx264`
(software encoder, ultrafast preset). On M-series Macs you can switch to the
hardware VideoToolbox encoder for near-zero CPU overhead. In
`internal/client/go/raylib/app/recorder.go`, replace:

```go
"-vcodec", "libx264",
"-preset", "ultrafast",
"-pix_fmt", "yuv420p",
"-crf", "18",
```

with:

```go
"-vcodec", "h264_videotoolbox",
"-b:v", "8000k",   // target bitrate; increase for higher quality
```

VideoToolbox does not accept `libx264` presets or CRF — use `-b:v` (bitrate)
instead. 8 Mbps is a good starting point for 1080p demo content.

---

### Linux

Use your distribution's package manager:

```bash
# Debian / Ubuntu
sudo apt install ffmpeg

# Fedora / RHEL 9+
sudo dnf install ffmpeg

# Arch
sudo pacman -S ffmpeg
```

The default `libx264` codepath works on all Linux distributions. For
NVIDIA hardware encoding substitute:

```go
"-vcodec", "h264_nvenc",
"-preset", "p1",   // fastest
"-b:v", "8000k",
```

For AMD/Intel GPU encoding use `h264_amf` or `h264_qsv` respectively.

---

### Windows

**Option A — Winget (Windows 11 / up-to-date Windows 10):**

```powershell
winget install --id Gyan.FFmpeg -e
```

**Option B — Manual install:**

1. Download a build from <https://www.gyan.dev/ffmpeg/builds/> (the
   *release essentials* zip is sufficient).
2. Extract to `C:\ffmpeg\` (or any path without spaces).
3. Add `C:\ffmpeg\bin` to your `PATH` environment variable and restart the
   terminal.

Verify:

```powershell
ffmpeg -version
```

The default `libx264` codepath works on Windows. For NVIDIA hardware encoding:

```go
"-vcodec", "h264_nvenc",
"-preset", "p1",
"-b:v", "8000k",
```

---

## Changing Output Location or Format

All recording parameters are in a single function in
`internal/client/go/raylib/app/recorder.go`. The relevant block:

```go
cmd := exec.Command("ffmpeg",
    "-y",
    "-f", "rawvideo",
    "-pix_fmt", "rgba",
    "-s", size,
    "-r", "60",
    "-i", "pipe:0",
    "-vf", "vflip",        // required: OpenGL textures are bottom-up
    "-vcodec", "libx264",
    "-preset", "ultrafast",
    "-pix_fmt", "yuv420p",
    "-crf", "18",          // quality: lower = better; 18 is near-lossless
    outPath,
)
```

Common adjustments:

| Goal | Change |
|------|--------|
| Higher quality | Lower `-crf` (e.g. `"15"`) |
| Smaller file | Raise `-crf` (e.g. `"23"`) |
| Different frame rate | Change `"-r", "60"` |
| GIF output | Replace everything after `-vf vflip` with `-vf vflip,fps=15,scale=960:-1:flags=lanczos -gifflags +transdiff` and change the file extension |
| ProRes (editing) | `-vcodec prores_ks -profile:v 2` (macOS/Linux only) |

Output path is constructed from `os.UserHomeDir()` + `Desktop/`. Change the
`outPath` line to redirect to a different directory.

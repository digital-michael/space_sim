package render

import (
	"fmt"
	"math"
	"time"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func formatSimulationDateText(simSeconds float64, secondsPerSecond float32) string {
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	currentTime := j2000.Add(time.Duration(simSeconds * float64(time.Second)))
	localTime := currentTime.Local()

	year := localTime.Year()
	month := int(localTime.Month())
	day := localTime.Day()

	if secondsPerSecond >= hideDateAtOrAboveSecondsPerSecond {
		return fmt.Sprintf("Date: %04d/%02d/%02d", year, month, day)
	}

	hour := localTime.Hour()
	minute := localTime.Minute()
	second := localTime.Second()
	millisecond := localTime.Nanosecond() / 1000000

	return fmt.Sprintf("Date: %04d/%02d/%02d %02d:%02d:%02d.%03d",
		year, month, day, hour, minute, second, millisecond)
}
// DrawRecordingIndicator draws a ● REC (active) or ⏸ REC (paused) badge in
// the top-right corner of the render surface when recording is active.
func (r *Renderer) DrawRecordingIndicator(paused bool) {
	fontSize := scaledInt32(25)
	pad := scaledInt32(8)

	var label string
	var dotColor rl.Color
	if paused {
		label = "|| REC"
		dotColor = rl.Yellow
	} else {
		label = "* REC"
		dotColor = rl.Red
	}

	textW := rl.MeasureText(label, fontSize)
	x := int32(currentScreenWidth()) - textW - pad
	y := pad

	// Semi-transparent background
	rl.DrawRectangle(x-pad/2, y-pad/2, textW+pad, fontSize+pad, rl.Color{R: 0, G: 0, B: 0, A: 160})
	rl.DrawText(label, x, y, fontSize, dotColor)
}
func drawHUDDebug(state *engine.SimulationState, cameraState *ui.CameraState, asteroidDataset engine.AsteroidDataset, speed float64, inViewCount int, eligibleInViewCount int, renderedCount int, labelMode ui.LabelMode) {
	leftPad := scaledInt32(10)
	fontLarge := scaledInt32(20)
	fontMedium := scaledInt32(18)
	line1Y := scaledInt32(10)
	line2Y := scaledInt32(35)
	line3Y := scaledInt32(60)
	line4Y := scaledInt32(85)
	line5Y := scaledInt32(108)
	line6Y := scaledInt32(133)
	line7Y := scaledInt32(158)
	line8Y := scaledInt32(183)
	line9Y := scaledInt32(208)

	fps := rl.GetFPS()
	rl.DrawText(fmt.Sprintf("FPS: %3d / %d threads", fps, state.NumWorkers), leftPad, line1Y, fontLarge, rl.Green)

	totalObjects := len(state.Objects)
	visibleObjects := 0
	for _, obj := range state.Objects {
		if obj.Visible {
			visibleObjects++
		}
	}
	datasetName := asteroidDataset.Name()
	rl.DrawText(fmt.Sprintf("Objects: %d total / %d visible (Dataset: %s)", totalObjects, visibleObjects, datasetName), leftPad, line2Y, fontLarge, rl.White)

	dateText := formatSimulationDateText(state.Time, state.SecondsPerSecond)
	rl.DrawText(dateText, leftPad, line3Y, fontLarge, rl.White)

	var timeRateText string
	timeRateColor := rl.Gray
	sps := state.SecondsPerSecond
	if sps == 0.0 {
		timeRateText = "Time Rate: PAUSED"
		timeRateColor = rl.Red
	} else if sps == 1.0 {
		timeRateText = "Time Rate: real-time"
		timeRateColor = rl.White
	} else if sps == 3600.0 {
		timeRateText = "Time Rate: 1 hour/sec"
		timeRateColor = rl.Green
	} else if sps == 86400.0 {
		timeRateText = "Time Rate: 1 day/sec"
		timeRateColor = rl.Green
	} else if sps == 604800.0 {
		timeRateText = "Time Rate: 1 week/sec"
		timeRateColor = rl.Green
	} else if sps == 2628000.0 {
		timeRateText = "Time Rate: 1 month/sec"
		timeRateColor = rl.Yellow
	} else if sps == 31557600.0 {
		timeRateText = "Time Rate: 1 year/sec"
		timeRateColor = rl.Yellow
	} else {
		timeRateText = fmt.Sprintf("Time Rate: %.4g sec/sec", sps)
		timeRateColor = rl.Gray
	}
	rl.DrawText(timeRateText, leftPad, line4Y, fontMedium, timeRateColor)

	animSpeed := speed
	var animSpeedText string
	var animSpeedColor rl.Color
	if animSpeed == 0.0 {
		animSpeedText = "Anim Speed: PAUSED"
		animSpeedColor = rl.Red
	} else if animSpeed > 1.0 {
		animSpeedText = fmt.Sprintf("Anim Speed: %.4g×", animSpeed)
		animSpeedColor = rl.White
	} else if animSpeed == 1.0 {
		animSpeedText = "Anim Speed: 100%"
		animSpeedColor = rl.White
	} else {
		animSpeedText = fmt.Sprintf("Anim Speed: %d%%", int(animSpeed*100))
		animSpeedColor = rl.Color{R: 255, G: 165, B: 0, A: 255}
	}
	rl.DrawText(animSpeedText, leftPad, line5Y, fontMedium, animSpeedColor)

	var modeText string
	switch cameraState.Mode {
	case ui.CameraModeFree:
		modeText = "FREE-FLY"
	case ui.CameraModeJumping:
		modeText = "JUMPING"
	case ui.CameraModeTracking:
		modeText = fmt.Sprintf("TRACKING: %s", state.Objects[cameraState.Tracking.TargetIndex].Meta.Name)
	}
	rl.DrawText(fmt.Sprintf("Mode: %s", modeText), leftPad, line6Y, fontLarge, rl.Yellow)

	posText := fmt.Sprintf("Camera Position: X:%.1f Y:%.1f Z:%.1f", cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z)
	rl.DrawText(posText, leftPad, line7Y, fontMedium, rl.Color{R: 0, G: 255, B: 255, A: 255})

	// Label mode indicator
	var labelModeText string
	var labelModeColor rl.Color
	switch labelMode {
	case ui.LabelModeOff:
		labelModeText = "Labels: OFF"
		labelModeColor = rl.Color{R: 150, G: 150, B: 150, A: 255}
	case ui.LabelModeOn:
		labelModeText = "Labels: ON (all)"
		labelModeColor = rl.Color{R: 100, G: 255, B: 100, A: 255}
	case ui.LabelModeNearest:
		labelModeText = "Labels: NEAREST"
		labelModeColor = rl.Color{R: 255, G: 220, B: 80, A: 255}
	}
	rl.DrawText(labelModeText, leftPad, line8Y, fontMedium, labelModeColor)

	// POV-inside-object indicator: scan all rendered objects for camera containment.
	// Camera position and object positions are both in sim-space (su), so the
	// comparison is direct: inside when dist < PhysicalRadius.
	type povInsideEntry struct {
		name   string
		dist   float32
		radius float32
	}
	var povInside []povInsideEntry
	for _, o := range state.Objects {
		if o.Meta.PhysicalRadius <= 0 {
			continue
		}
		d := cameraState.Position.Sub(o.Anim.Position).Length()
		if d < o.Meta.PhysicalRadius {
			povInside = append(povInside, povInsideEntry{name: o.Meta.Name, dist: d, radius: o.Meta.PhysicalRadius})
		}
	}
	if len(povInside) > 0 {
		y := line9Y
		for _, e := range povInside {
			povText := fmt.Sprintf("⚠ INSIDE: %s  [%.1f su inside / radius %.1f su]", e.name, e.radius-e.dist, e.radius)
			rl.DrawText(povText, leftPad, y, fontMedium, rl.Color{R: 255, G: 80, B: 80, A: 255})
			y += scaledInt32(25)
		}
	} else {
		rl.DrawText("POV: outside all objects", leftPad, line9Y, scaledInt32(14), rl.Color{R: 80, G: 200, B: 80, A: 160})
	}

	// Lower-left: screen/monitor/render diagnostics
	screenW := rl.GetScreenWidth()
	screenH := rl.GetScreenHeight()
	monitorW := rl.GetMonitorWidth(0)
	monitorH := rl.GetMonitorHeight(0)
	isFullscreen := rl.IsWindowFullscreen()

	debugLine1Y := int32(currentScreenHeight()) - scaledInt32(85)
	debugLine2Y := int32(currentScreenHeight()) - scaledInt32(60)
	debugFontSize := scaledInt32(14)
	debugColor := rl.Color{R: 200, G: 200, B: 200, A: 200}

	fsText := "windowed"
	if isFullscreen {
		fsText = "fullscreen"
	}
	rl.DrawText(fmt.Sprintf("Screen: %dx%d | Monitor: %dx%d | %s", screenW, screenH, monitorW, monitorH, fsText), leftPad, debugLine1Y, debugFontSize, debugColor)

	var visiblePct float32
	if totalObjects > 0 {
		visiblePct = float32(visibleObjects) / float32(totalObjects) * 100.0
	}
	var renderPct float32
	if eligibleInViewCount > 0 {
		renderPct = float32(renderedCount) / float32(eligibleInViewCount) * 100.0
	}
	processingColor := rl.Gray
	if eligibleInViewCount > 0 {
		processingColor = rl.Red
		if renderPct >= 60.0 {
			processingColor = rl.Yellow
		}
		if renderPct >= 90.0 {
			processingColor = rl.Green
		}
	}
	rl.DrawText(fmt.Sprintf("Render: %d/%d eligible (%.1f%%) | In-view: %d/%d visible | Visible: %.1f%% of %d total",
		renderedCount, eligibleInViewCount, renderPct, inViewCount, visibleObjects, visiblePct, totalObjects),
		leftPad, debugLine2Y, debugFontSize, processingColor)
}

// drawHUDInfo draws the lower-right tracking info panel and the selection UI
// overlay when an object is selected.
func drawHUDInfo(state *engine.SimulationState, cameraState *ui.CameraState, inputState *ui.InputState) {
	if inputState.SelectionActive {
		drawSelectionUI(state, inputState)
	}
	if cameraState.Mode == ui.CameraModeTracking {
		drawTrackingInfo(state, cameraState)
	}
}

// drawHUDHelp draws the bottom-left hint bar ("Ctrl+/ for help …").
func drawHUDHelp() {
	leftPad := scaledInt32(10)
	fontLarge := scaledInt32(20)
	helpY := int32(currentScreenHeight()) - scaledInt32(30)
	rl.DrawText("? for help | "+modSuper+"+S systems | "+modAlt+"+H HUD settings | "+modAlt+"+Q to quit", leftPad, helpY, fontLarge, rl.Gray)
}

// drawZoomIndicator draws a visual indicator when zooming
func drawZoomIndicator(zoomValue float32) {
	centerX := int32(currentScreenWidth() / 2)
	centerY := int32(currentScreenHeight() / 2)

	// Determine zoom direction and text
	var text string
	var color rl.Color
	var barLength int32

	if zoomValue > 0 {
		text = "ZOOM IN"
		color = rl.Color{R: 100, G: 200, B: 255, A: 220}
		barLength = int32(zoomValue * 50)
	} else {
		text = "ZOOM OUT"
		color = rl.Color{R: 255, G: 150, B: 100, A: 220}
		barLength = int32(-zoomValue * 50)
	}

	if barLength > 200 {
		barLength = 200
	}

	// Draw semi-transparent background
	bgWidth := int32(260)
	bgHeight := int32(60)
	rl.DrawRectangle(centerX-bgWidth/2, centerY+150, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 150})

	// Draw text
	textWidth := rl.MeasureText(text, 20)
	rl.DrawText(text, centerX-textWidth/2, centerY+160, 20, color)

	// Draw zoom bar
	barY := centerY + 185
	rl.DrawRectangle(centerX-100, barY, 200, 8, rl.Color{R: 50, G: 50, B: 50, A: 200})

	if zoomValue > 0 {
		// Zoom in - bar grows from center to right
		rl.DrawRectangle(centerX, barY, barLength, 8, color)
	} else {
		// Zoom out - bar grows from center to left
		rl.DrawRectangle(centerX-barLength, barY, barLength, 8, color)
	}
}

func drawWelcomeBanner(text string, elapsed time.Duration) {
	if text == "" {
		return
	}

	alpha := uint8(255)
	if elapsed > time.Second {
		fadeDuration := time.Second
		fadeElapsed := elapsed - time.Second
		if fadeElapsed >= fadeDuration {
			return
		}
		fadeRatio := 1.0 - float64(fadeElapsed)/float64(fadeDuration)
		alpha = uint8(math.Round(255.0 * fadeRatio))
	}

	centerX := int32(currentScreenWidth() / 2)
	centerY := int32(currentScreenHeight() / 2)
	fontSize := scaledInt32(28)
	padX := scaledInt32(24)
	padY := scaledInt32(16)
	textWidth := rl.MeasureText(text, fontSize)
	bgWidth := int32(textWidth) + padX*2
	bgHeight := fontSize + padY*2
	x := centerX - bgWidth/2
	y := centerY - bgHeight/2

	bgAlpha := uint8(math.Round(float64(alpha) * 0.7))
	borderAlpha := uint8(math.Round(float64(alpha) * 0.9))
	textColor := rl.Color{R: 255, G: 255, B: 255, A: alpha}

	rl.DrawRectangle(x, y, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: bgAlpha})
	rl.DrawRectangleLinesEx(rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(bgWidth), Height: float32(bgHeight)}, 2, rl.Color{R: 150, G: 210, B: 255, A: borderAlpha})
	rl.DrawText(text, centerX-int32(textWidth)/2, y+padY, fontSize, textColor)
}

// drawTrackingInfo displays comprehensive information about the tracked object in the lower right
func drawTrackingInfo(state *engine.SimulationState, cameraState *ui.CameraState) {
	if cameraState.Tracking.TargetIndex < 0 || cameraState.Tracking.TargetIndex >= len(state.Objects) {
		return
	}

	obj := state.Objects[cameraState.Tracking.TargetIndex]

	// Calculate display metrics
	const auToSimUnits = 100.0 // 100 units = 1 AU

	// Distance from system primary (highest-importance no-parent star or black hole)
	systemCenterName := "Sol"
	systemCenterPos := engine.Vector3{}
	var bestImportance int
	for _, o := range state.Objects {
		if o.Meta.ParentName != "" {
			continue
		}
		if !isStarLike(o.Meta.Category) {
			continue
		}
		if o.Meta.Importance > bestImportance || systemCenterName == "Sol" {
			bestImportance = o.Meta.Importance
			systemCenterName = o.Meta.Name
			systemCenterPos = o.Anim.Position
		}
	}
	distFromSol := obj.Anim.Position.Sub(systemCenterPos).Length() / auToSimUnits

	// Camera distance to object
	cameraDistUnits := cameraState.Position.Sub(obj.Anim.Position).Length()
	cameraDistAU := cameraDistUnits / auToSimUnits
	cameraDistKm := cameraDistAU * 149597870.7 // 1 AU in km

	// Determine parent and count siblings
	parentName := ""
	for _, o := range state.Objects {
		if isStarLike(o.Meta.Category) {
			parentName = o.Meta.Name
			break
		}
	}
	if parentName == "" {
		parentName = "Star"
	}
	siblingCount := 0
	siblingIndex := 0

	if obj.Meta.ParentName != "" {
		parentName = obj.Meta.ParentName
		// Count objects with same parent
		for i, otherObj := range state.Objects {
			if otherObj.Meta.ParentName == obj.Meta.ParentName && otherObj.Meta.Category == obj.Meta.Category {
				siblingCount++
				if i <= cameraState.Tracking.TargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryPlanet {
		// Count planets with no parent (orbit the star)
		for i, otherObj := range state.Objects {
			if otherObj.Meta.Category == engine.CategoryPlanet && otherObj.Meta.ParentName == "" {
				siblingCount++
				if i <= cameraState.Tracking.TargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryDwarfPlanet {
		// Count dwarf planets
		for i, otherObj := range state.Objects {
			if otherObj.Meta.Category == engine.CategoryDwarfPlanet {
				siblingCount++
				if i <= cameraState.Tracking.TargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryAsteroid {
		// For asteroids, show belt membership
		if len(obj.Meta.Name) >= 9 && obj.Meta.Name[0:9] == "Asteroid-" {
			parentName = "Asteroid Belt"
		} else {
			parentName = "Kuiper Belt"
		}
		// Count visible asteroids in same belt
		for i, otherObj := range state.Objects {
			if otherObj.Visible && otherObj.Meta.Category == engine.CategoryAsteroid {
				namePrefix := ""
				if len(otherObj.Meta.Name) >= 9 {
					namePrefix = otherObj.Meta.Name[0:9]
				}
				objPrefix := ""
				if len(obj.Meta.Name) >= 9 {
					objPrefix = obj.Meta.Name[0:9]
				}
				if namePrefix == objPrefix {
					siblingCount++
					if i <= cameraState.Tracking.TargetIndex && otherObj.Meta.Name == obj.Meta.Name {
						siblingIndex = siblingCount
					}
				}
			}
		}
	}

	// Format category name
	categoryName := ""
	switch obj.Meta.Category {
	case engine.CategoryPlanet:
		categoryName = "Planet"
	case engine.CategoryDwarfPlanet:
		categoryName = "Dwarf Planet"
	case engine.CategoryMoon:
		categoryName = "Moon"
	case engine.CategoryAsteroid:
		categoryName = "Asteroid"
	case engine.CategoryStarPreMain:
		categoryName = "Pre-Main Sequence Star"
	case engine.CategoryStarMainSequence:
		categoryName = "Star"
	case engine.CategoryStarEvolved:
		categoryName = "Evolved Star"
	case engine.CategorySubstellar:
		categoryName = "Substellar Object"
	case engine.CategoryStellarRemnant:
		if obj.Meta.StellarVariant != "" {
			categoryName = formatVariantLabel(obj.Meta.StellarVariant)
		} else {
			categoryName = "Stellar Remnant"
		}
	case engine.CategoryRogue:
		if obj.Meta.StellarVariant != "" {
			categoryName = formatVariantLabel(obj.Meta.StellarVariant)
		} else {
			categoryName = "Rogue Body"
		}
	case engine.CategoryArtifact:
		if obj.Meta.StellarVariant != "" {
			categoryName = formatVariantLabel(obj.Meta.StellarVariant)
		} else {
			categoryName = "Artifact"
		}
	}

	// Format mass (scientific notation)
	massStr := fmt.Sprintf("%.2e kg", obj.Meta.Mass)

	// Format physical radius.
	// Prefer PhysicalRadiusKm (real-world value) when populated; fall back to the
	// sim-space rendered radius converted to AU.
	var radiusStr string
	if obj.Meta.PhysicalRadiusKm > 0 {
		rKm := float64(obj.Meta.PhysicalRadiusKm)
		if rKm >= 149597.87 { // ≥ 0.001 AU — large enough to express as AU
			radiusStr = fmt.Sprintf("%.4f AU (%.0f km)", rKm/149597870.7, rKm)
		} else if rKm >= 1000 {
			radiusStr = fmt.Sprintf("%.0f km", rKm)
		} else {
			radiusStr = fmt.Sprintf("%.1f km", rKm)
		}
	} else if obj.Meta.PhysicalRadius > 0 {
		rAU := float64(obj.Meta.PhysicalRadius) / float64(auToSimUnits)
		if rAU >= 0.01 {
			radiusStr = fmt.Sprintf("%.3f AU", rAU)
		} else if rAU >= 0.0001 {
			radiusStr = fmt.Sprintf("%.5f AU", rAU)
		} else {
			radiusStr = fmt.Sprintf("%.0f km", rAU*149597870.7)
		}
	}

	// Format rotation period
	rotationStr := "Unknown"
	if obj.Meta.RotationPeriod > 0 {
		if obj.Meta.RotationPeriod < 48 {
			rotationStr = fmt.Sprintf("%.1f hours", obj.Meta.RotationPeriod)
		} else {
			days := obj.Meta.RotationPeriod / 24.0
			rotationStr = fmt.Sprintf("%.1f days", days)
		}

		// Check for tidal locking (rotation ~= orbital period)
		if obj.Meta.ParentName != "" && obj.Meta.OrbitalPeriod > 0 {
			orbitalDays := obj.Meta.OrbitalPeriod / 86400.0 // Convert seconds to days
			rotationalDays := obj.Meta.RotationPeriod / 24.0
			if math.Abs(float64(orbitalDays-rotationalDays)) < 0.5 {
				rotationStr += " (tidally locked)"
			}
		}
	}

	// Format orbital period
	orbitalStr := "N/A"
	if obj.Meta.OrbitalPeriod > 0 {
		days := obj.Meta.OrbitalPeriod / 86400.0 // Convert seconds to Earth days
		if days < 1 {
			hours := obj.Meta.OrbitalPeriod / 3600.0
			orbitalStr = fmt.Sprintf("%.2f hours", hours)
		} else if days < 730 {
			orbitalStr = fmt.Sprintf("%.1f days", days)
		} else {
			years := days / 365.256
			orbitalStr = fmt.Sprintf("%.1f years", years)
		}
	}

	// Format camera distance (auto-scale)
	cameraDistStr := ""
	if cameraDistKm < 1000 {
		cameraDistStr = fmt.Sprintf("%.1f km", cameraDistKm)
	} else if cameraDistAU < 0.1 {
		cameraDistStr = fmt.Sprintf("%.0f km", cameraDistKm)
	} else if cameraDistAU < 10 {
		cameraDistStr = fmt.Sprintf("%.3f AU", cameraDistAU)
	} else {
		cameraDistStr = fmt.Sprintf("%.1f AU", cameraDistAU)
	}

	// Format eccentricity
	eccentricityStr := fmt.Sprintf("%.3f", obj.Meta.Eccentricity)
	if obj.Meta.Eccentricity < 0.01 {
		eccentricityStr += " (circular)"
	} else if obj.Meta.Eccentricity < 0.1 {
		eccentricityStr += " (low)"
	} else if obj.Meta.Eccentricity < 0.5 {
		eccentricityStr += " (moderate)"
	} else {
		eccentricityStr += " (high)"
	}

	// Calculate orbital phase (percentage through orbit)
	orbitalPhase := 0.0
	if obj.Meta.OrbitalPeriod > 0 {
		// Mean anomaly is in radians, convert to percentage
		phase := float64(obj.Anim.MeanAnomaly) / (2.0 * math.Pi)
		// Normalize to 0-100%
		for phase < 0 {
			phase += 1.0
		}
		for phase > 1 {
			phase -= 1.0
		}
		orbitalPhase = phase * 100.0
	}

	// Build info lines with labels and values separated for color formatting
	type InfoLine struct {
		label string
		value string
	}

	infoLines := []InfoLine{
		{"Object:", obj.Meta.Name},
		{"Type:", categoryName},
		{"Distance from " + systemCenterName + ":", fmt.Sprintf("%.2f AU", distFromSol)},
	}

	// Add orbit info if applicable
	if siblingCount > 1 {
		// Only show count if more than one sibling
		infoLines = append(infoLines, InfoLine{"Orbits:", fmt.Sprintf("%s (%d of %d)", parentName, siblingIndex, siblingCount)})
	} else {
		infoLines = append(infoLines, InfoLine{"Orbits:", parentName})
	}

	infoLines = append(infoLines,
		InfoLine{"Mass:", massStr},
	)
	if radiusStr != "" {
		infoLines = append(infoLines, InfoLine{"Radius:", radiusStr})
	}
	infoLines = append(infoLines,
		InfoLine{"Rotation:", rotationStr},
	)

	// Add axial tilt if present
	if obj.Meta.AxialTilt != 0 {
		infoLines = append(infoLines, InfoLine{"Axial Tilt:", fmt.Sprintf("%.1f°", obj.Meta.AxialTilt)})
	}

	infoLines = append(infoLines,
		InfoLine{"Orbital Period:", orbitalStr},
	)

	// Add orbital metrics if object orbits something
	if obj.Meta.OrbitalPeriod > 0 {
		infoLines = append(infoLines,
			InfoLine{"Eccentricity:", eccentricityStr},
			InfoLine{"Orbital Phase:", fmt.Sprintf("%.1f%%", orbitalPhase)},
		)
	}

	// Calculate and display orbital velocity
	velocityMagnitude := obj.Anim.Velocity.Length()
	velocityKmPerSec := velocityMagnitude * 1495978.707 // Convert sim units to km/s (1 AU = 149597870.7 km, 1 sim unit = 0.01 AU)
	if velocityKmPerSec > 0.01 {
		infoLines = append(infoLines, InfoLine{"Orbital Velocity:", fmt.Sprintf("%8.2f km/s", velocityKmPerSec)})
	}

	// Calculate and display rotational velocity
	if obj.Meta.RotationPeriod > 0 && obj.Meta.PhysicalRadius > 0 {
		// Rotational velocity at equator: v = 2πr / T
		radiusKm := obj.Meta.PhysicalRadius / 1000.0              // Convert meters to km
		rotationPeriodSeconds := obj.Meta.RotationPeriod * 3600.0 // Convert hours to seconds
		rotationalVelocityKmPerSec := (2.0 * math.Pi * radiusKm) / rotationPeriodSeconds
		infoLines = append(infoLines, InfoLine{"Rotational Velocity:", fmt.Sprintf("%8.2f km/s (at equator)", rotationalVelocityKmPerSec)})
	}

	infoLines = append(infoLines, InfoLine{"Camera Distance:", cameraDistStr})

	// Calculate dimensions
	fontSize := scaledInt32(16)
	lineHeight := scaledInt32(22)
	padding := scaledInt32(15)
	panelMargin := scaledInt32(20)

	maxWidth := int32(0)
	for _, info := range infoLines {
		fullLine := info.label + " " + info.value
		width := rl.MeasureText(fullLine, fontSize)
		if width > maxWidth {
			maxWidth = width
		}
	}

	boxWidth := maxWidth + padding*2
	boxHeight := int32(len(infoLines))*lineHeight + padding*2

	// Position in lower right corner
	boxX := int32(currentScreenWidth()) - boxWidth - panelMargin
	boxY := int32(currentScreenHeight()) - boxHeight - panelMargin

	// Draw semi-transparent background
	rl.DrawRectangle(boxX, boxY, boxWidth, boxHeight, rl.Color{R: 0, G: 0, B: 0, A: 180})

	// Draw border
	rl.DrawRectangleLines(boxX, boxY, boxWidth, boxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})

	// Draw lines with colored labels and values
	labelColor := rl.Color{R: 150, G: 200, B: 255, A: 255} // Light blue for labels
	valueColor := rl.Color{R: 255, G: 255, B: 255, A: 255} // White for values

	textY := boxY + padding
	for _, info := range infoLines {
		textX := boxX + padding
		// Draw label in light blue
		rl.DrawText(info.label, textX, textY, fontSize, labelColor)
		labelWidth := rl.MeasureText(info.label, fontSize)

		// Draw value in white, offset by label width plus a space
		rl.DrawText(info.value, textX+labelWidth+rl.MeasureText(" ", fontSize), textY, fontSize, valueColor)
		textY += lineHeight
	}
}

// drawSelectionUI draws the object selection menu with category tabs

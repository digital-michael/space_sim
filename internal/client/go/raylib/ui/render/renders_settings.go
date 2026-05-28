package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// renderMousePosition converts the OS/window mouse position into render-texture
// coordinates. When the render texture is letterboxed or pillarboxed inside the
// window, rl.GetMousePosition() returns screen coords; this helper maps them
// back to the fixed layout space used by all UI drawing code.
func renderMousePosition() rl.Vector2 {
	m := rl.GetMousePosition()
	if layoutWidth <= 0 || layoutHeight <= 0 {
		return m
	}
	windowW := float32(rl.GetRenderWidth())
	windowH := float32(rl.GetRenderHeight())
	scaleX := windowW / float32(layoutWidth)
	scaleY := windowH / float32(layoutHeight)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	destW := float32(layoutWidth) * scale
	destH := float32(layoutHeight) * scale
	destX := (windowW - destW) * 0.5
	destY := (windowH - destH) * 0.5
	return rl.Vector2{
		X: (m.X - destX) / scale,
		Y: (m.Y - destY) / scale,
	}
}


// drawSettingsDialog draws a wide modal panel with four tabs:
// System, Display, Performance, Controls.
func drawSettingsDialog(state *ui.SettingsState, km *input.KeyMap) {
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())

	// Panel geometry — 70% width like the old help screen
	bgWidth := sw * 70 / 100
	if bgWidth < 800 {
		bgWidth = 800
	}
	if bgWidth > 1400 {
		bgWidth = 1400
	}
	bgHeight := sh * 78 / 100
	if bgHeight < 600 {
		bgHeight = 600
	}
	if bgHeight > 1000 {
		bgHeight = 1000
	}
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2

	// Background
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 230})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	titleFont := scaledInt32(31)
	tabFont := scaledInt32(22)
	hintFont := scaledInt32(20)

	// Title
	titleText := "SETTINGS"
	titleX := bgX + (bgWidth-rl.MeasureText(titleText, titleFont))/2
	rl.DrawText(titleText, titleX, bgY+scaledInt32(10), titleFont, rl.White)

	// Tab bar
	tabNames := [4]string{"System", "Display", "Performance", "Controls"}
	tabCount := int32(4)
	tabPad := scaledInt32(10)
	tabHeight := scaledInt32(32)
	tabWidth := (bgWidth - tabPad*2) / tabCount
	tabY := bgY + scaledInt32(44)
	for i := int32(0); i < tabCount; i++ {
		tx := bgX + tabPad + i*tabWidth
		tabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
		textColor := rl.LightGray
		if int(i) == state.ActiveTab {
			tabColor = rl.Color{R: 60, G: 100, B: 160, A: 255}
			textColor = rl.White
		}
		rl.DrawRectangle(tx, tabY, tabWidth-scaledInt32(4), tabHeight, tabColor)
		rl.DrawRectangleLines(tx, tabY, tabWidth-scaledInt32(4), tabHeight, rl.Gray)
		label := tabNames[i]
		lw := rl.MeasureText(label, tabFont)
		rl.DrawText(label, tx+(tabWidth-scaledInt32(4)-int32(lw))/2, tabY+scaledInt32(8), tabFont, textColor)
	}

	// Content area starts below tabs
	contentY := tabY + tabHeight + scaledInt32(10)
	contentH := bgHeight - (contentY - bgY) - scaledInt32(36) // leave room for hint bar

	switch state.ActiveTab {
	case 0:
		drawSettingsSystemTab(bgX, contentY, bgWidth, contentH)
	case 1:
		drawSettingsDisplayTab(state, bgX, contentY, bgWidth, contentH)
	case 2:
		drawSettingsPerfTab(state, bgX, contentY, bgWidth, contentH)
	case 3:
		drawSettingsConfigTab(state, km, bgX, contentY, bgWidth, contentH)
	}

	// Hint bar at bottom
	hintY := bgY + bgHeight - scaledInt32(28)
	hint := "TAB/SHIFT+TAB: switch tab  UP/DOWN: select  SPACE: toggle  LEFT/RIGHT: adjust  ESC: close"
	hintX := bgX + (bgWidth-rl.MeasureText(hint, hintFont))/2
	rl.DrawText(hint, hintX, hintY, hintFont, rl.Gray)
}

// drawSettingsSystemTab renders navigation, system shortcuts, dialog controls,
// and camera mode reference (tab 0).
func drawSettingsSystemTab(bgX, startY, bgWidth, _ int32) {
	margin := scaledInt32(20)
	valueGap := scaledInt32(180)
	lineHeight := scaledInt32(26)
	headerSize := scaledInt32(20)
	bodySize := scaledInt32(16)

	leftCol := bgX + margin
	rightCol := bgX + bgWidth/2 + margin
	y := startY

	// Left column: Navigation + Camera modes
	rl.DrawText("NAVIGATION", leftCol, y, headerSize, rl.Yellow)
	y += lineHeight + 4
	navEntries := [][2]string{
		{"C", "Center view on Sun / Reset zoom"},
		{"J", "Open jump dialog"},
		{"T", "Open track dialog"},
		{"TAB", "Next sibling (when tracking)"},
		{"Shift+TAB", "Previous sibling (tracking)"},
		{"F", "Forward to child (tracking)"},
		{"B", "Back to parent (tracking)"},
		{"R", "Reset camera offset"},
		{"ESC", "Exit tracking/mouse mode"},
		{modAlt + "+Q", "Quit application"},
	}
	for _, e := range navEntries {
		rl.DrawText(e[0], leftCol, y, bodySize, rl.White)
		rl.DrawText(e[1], leftCol+valueGap, y, bodySize, rl.LightGray)
		y += lineHeight
	}
	y += 8
	rl.DrawText("CAMERA MODES", leftCol, y, headerSize, rl.Yellow)
	y += lineHeight + 4
	modeEntries := [][2]string{
		{"Free-Fly", "Full manual control"},
		{"Jumping", "Animated fly-to"},
		{"Tracking", "Follow target object"},
	}
	for _, e := range modeEntries {
		rl.DrawText(e[0], leftCol, y, bodySize, rl.Lime)
		rl.DrawText(e[1], leftCol+valueGap, y, bodySize, rl.LightGray)
		y += lineHeight
	}

	// Right column: System & Display + Dialog Controls
	y = startY
	rl.DrawText("SYSTEM & DISPLAY", rightCol, y, headerSize, rl.Yellow)
	y += lineHeight + 4
	sysEntries := [][2]string{
		{"F1", "Open/close settings"},
		{modAlt + "+L", "Toggle object labels"},
		{modAlt + "+M", "Toggle mouse mode"},
		{modAlt + "+F", "Toggle fullscreen"},
		{"Ctrl+-  /  Ctrl+=", "Zoom in/out"},
		{modAlt + "+-  /  " + modAlt + "+=", "Asteroids (200->24K)"},
		{"-  /  =", "UI scale (0.5x\u20132.0x)"},
		{modSuper + "+<  /  " + modSuper + "+>", "Time scale (PAUSED->1yr/sec)"},
		{modAlt + "+,  /  " + modAlt + "+.", "Anim speed (0%->100% of 60Hz)"},
		{modAlt + "+R", "Start/stop recording"},
		{modAlt + "+Shift+R", "Pause/resume recording"},
	}
	for _, e := range sysEntries {
		rl.DrawText(e[0], rightCol, y, bodySize, rl.White)
		rl.DrawText(e[1], rightCol+valueGap, y, bodySize, rl.LightGray)
		y += lineHeight
	}
	y += 8
	rl.DrawText("DIALOG CONTROLS", rightCol, y, headerSize, rl.Yellow)
	y += lineHeight + 4
	dlgEntries := [][2]string{
		{"Up / Down", "Navigate list / settings rows"},
		{"Left / Right", "Change category / adjust value"},
		{"Enter", "Confirm selection"},
		{"Space", "Toggle option"},
		{"ESC", "Close active dialog"},
		{"Modal", "Dialogs suspend main controls"},
	}
	for _, e := range dlgEntries {
		rl.DrawText(e[0], rightCol, y, bodySize, rl.White)
		rl.DrawText(e[1], rightCol+valueGap, y, bodySize, rl.LightGray)
		y += lineHeight
	}
}

// drawSettingsDisplayTab renders HUD visibility checkboxes (tab 1).
func drawSettingsDisplayTab(state *ui.SettingsState, bgX, startY, bgWidth, _ int32) {
	pad := scaledInt32(20)
	rowFont := scaledInt32(18)
	titleFont := scaledInt32(20)
	boxSize := scaledInt32(18)
	rowSpacing := scaledInt32(40)
	arrowFont := scaledInt32(20)

	rl.DrawText("HUD DISPLAYS", bgX+pad, startY, titleFont, rl.White)

	type hudRow struct {
		label string
		val   *bool
	}
	rows := []hudRow{
		{"Debug  (stats, screen info)", &state.HUD.Debug},
		{"Info   (tracking, selection)", &state.HUD.Info},
		{"Help   (hint bar)", &state.HUD.Help},
		{"Advanced Info  (spectral class, stellar variants)", &state.AdvancedInfo},
	}

	mouse := renderMousePosition()
	clicked := rl.IsMouseButtonPressed(rl.MouseButtonLeft)

	for i, row := range rows {
		rowY := startY + scaledInt32(36) + int32(i)*rowSpacing
		boxX := bgX + pad
		labelColor := rl.White
		boxColor := rl.Color{R: 60, G: 60, B: 60, A: 255}

		if i == state.SelectedRow {
			rl.DrawText(">", boxX-scaledInt32(16), rowY, arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, rowY-2, bgWidth-4, boxSize+4, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}

		if *row.val {
			boxColor = rl.Green
		}
		// Mouse hit test
		hitX := float32(boxX) - 2
		hitY := float32(rowY) - 2
		hitW := float32(boxSize+4) + float32(scaledInt32(200))
		hitH := float32(boxSize + 4)
		if clicked && mouse.X >= hitX && mouse.X < hitX+hitW && mouse.Y >= hitY && mouse.Y < hitY+hitH {
			*row.val = !*row.val
		}

		rl.DrawRectangle(boxX, rowY, boxSize, boxSize, boxColor)
		rl.DrawRectangleLines(boxX, rowY, boxSize, boxSize, rl.White)
		if *row.val {
			rl.DrawText("X", boxX+scaledInt32(2), rowY, scaledInt32(16), rl.Green)
		}
		rl.DrawText(row.label, boxX+boxSize+scaledInt32(10), rowY, rowFont, labelColor)
	}

	// UI Scale row (row 3) — LEFT/RIGHT adjusts, steps of 0.25 in [0.5, 2.0]
	scaleRowY := startY + scaledInt32(36) + int32(len(rows))*rowSpacing + scaledInt32(16)
	rl.DrawText("RENDERING", bgX+pad, scaleRowY, titleFont, rl.White)
	scaleRowY += scaledInt32(30)

	if state.SelectedRow == 4 {
		rl.DrawText(">", bgX+pad-scaledInt32(16), scaleRowY, arrowFont, rl.Yellow)
		rl.DrawRectangle(bgX+2, scaleRowY-2, bgWidth-4, scaledInt32(22), rl.Color{R: 50, G: 100, B: 150, A: 80})
	}
	rl.DrawText("UI Scale", bgX+pad, scaleRowY, rowFont, rl.White)
	scaleVal := fmt.Sprintf("<  %.2f  >  (LEFT / RIGHT to adjust)", state.UIScale)
	rl.DrawText(scaleVal, bgX+pad+scaledInt32(120), scaleRowY, rowFont, rl.LightGray)
}

// drawSettingsPerfTab renders performance toggle checkboxes (tab 3).
func drawSettingsPerfTab(state *ui.SettingsState, bgX, startY, bgWidth, _ int32) {
	pad := scaledInt32(20)
	optionFont := scaledInt32(18)
	descFont := scaledInt32(14)
	checkSize := scaledInt32(22)
	lineHeight := scaledInt32(58)
	arrowFont := scaledInt32(20)

	type perfOpt struct {
		name string
		desc string
		val  *bool
	}
	opts := []perfOpt{
		{"Frustum Culling", "Cull objects outside camera view", &state.Perf.FrustumCulling},
		{"Level of Detail", "Use simpler models for distant objects", &state.Perf.LODEnabled},
		{"Instanced Rendering", "Batch objects by rendering properties", &state.Perf.InstancedRendering},
		{"Spatial Partitioning", "Use grid-based spatial culling", &state.Perf.SpatialPartition},
		{"Point Rendering", "Render distant objects as points", &state.Perf.PointRendering},
	}

	for i, opt := range opts {
		y := startY + int32(i)*lineHeight
		checkX := bgX + pad + scaledInt32(16)
		checkY := y + scaledInt32(6)

		if i == state.SelectedRow {
			rl.DrawText(">", bgX+pad, y+scaledInt32(6), arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, y-2, bgWidth-4, lineHeight-4, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}

		boxColor := rl.Color{R: 40, G: 40, B: 40, A: 255}
		rl.DrawRectangle(checkX, checkY, checkSize, checkSize, boxColor)
		rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)
		if *opt.val {
			rl.DrawText("X", checkX+scaledInt32(4), checkY+scaledInt32(1), scaledInt32(20), rl.Green)
		}
		rl.DrawText(opt.name, checkX+checkSize+scaledInt32(10), checkY, optionFont, rl.White)
		rl.DrawText(opt.desc, checkX+checkSize+scaledInt32(10), checkY+scaledInt32(22), descFont, rl.Gray)
	}

	// Row 5: Importance Threshold (L/R adjust)
	{
		row := 5
		y := startY + int32(row)*lineHeight
		checkX := bgX + pad + scaledInt32(16)
		checkY := y + scaledInt32(6)
		if state.SelectedRow == row {
			rl.DrawText(">", bgX+pad, y+scaledInt32(6), arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, y-2, bgWidth-4, lineHeight-4, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}
		rl.DrawText("Importance Threshold", checkX, checkY, optionFont, rl.White)
		thresholdDesc := importanceThresholdLabel(state.Perf.ImportanceThreshold)
		rl.DrawText(fmt.Sprintf("Value: %s", thresholdDesc), checkX, checkY+scaledInt32(22), descFont, rl.Gray)
		rl.DrawText("LEFT/RIGHT: adjust", checkX+scaledInt32(300), checkY+scaledInt32(22), descFont, rl.LightGray)
	}

	// Row 6: UseInPlaceSwap (checkbox)
	{
		row := 6
		y := startY + int32(row)*lineHeight
		checkX := bgX + pad + scaledInt32(16)
		checkY := y + scaledInt32(6)
		if state.SelectedRow == row {
			rl.DrawText(">", bgX+pad, y+scaledInt32(6), arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, y-2, bgWidth-4, lineHeight-4, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}
		boxColor := rl.Color{R: 40, G: 40, B: 40, A: 255}
		rl.DrawRectangle(checkX, checkY, checkSize, checkSize, boxColor)
		rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)
		if state.Perf.UseInPlaceSwap {
			rl.DrawText("X", checkX+scaledInt32(4), checkY+scaledInt32(1), scaledInt32(20), rl.Green)
		}
		rl.DrawText("Zero-Allocation In-Place Swap", checkX+checkSize+scaledInt32(10), checkY, optionFont, rl.White)
		rl.DrawText("Eliminates buffer allocations (disables dynamic adds)", checkX+checkSize+scaledInt32(10), checkY+scaledInt32(22), descFont, rl.Gray)
	}
}

// drawSettingsConfigTab renders the keybinding editor (tab 3, "Controls").
// Layout:
//
//	Row 0: [Load]   — active file path, ENTER opens file picker
//	Row 1: [Save As] — destination path, ENTER saves
//	Status line: conflict / unsaved / capture hint
//	Two-column action list: left = first half, right = second half
func drawSettingsConfigTab(state *ui.SettingsState, km *input.KeyMap, bgX, startY, bgWidth, contentH int32) {
	pad := scaledInt32(18)
	optionFont := scaledInt32(17)
	descFont := scaledInt32(13)
	groupHeaderH := scaledInt32(22)
	headerLineH := scaledInt32(30)
	statusH := scaledInt32(18)
	actionLineH := scaledInt32(24)
	arrowFont := scaledInt32(16)
	keyFont := scaledInt32(15)
	nameFont := scaledInt32(14)

	leftX := bgX + pad + scaledInt32(14)

	// "Keyboard" group header
	rl.DrawText("-- Keyboard --", leftX, startY+scaledInt32(2), descFont, rl.SkyBlue)
	rl.DrawLine(bgX+2, startY+scaledInt32(17), bgX+bgWidth-4, startY+scaledInt32(17), rl.Color{R: 50, G: 80, B: 120, A: 180})

	// ── Row 0: Load ──────────────────────────────────────────────────────
	{
		y := startY + groupHeaderH
		if state.SelectedRow == 0 {
			rl.DrawText(">", bgX+pad, y+scaledInt32(3), arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, y-2, bgWidth-4, headerLineH-2, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}
		rl.DrawText("[Load]", leftX, y+scaledInt32(3), optionFont, rl.SkyBlue)
		filePath := state.KeybindingsPath
		if filePath == "" {
			filePath = "configs/keybindings.json"
		}
		rl.DrawText(filePath, leftX+scaledInt32(76), y+scaledInt32(3), descFont, rl.LightGray)
	}

	// ── Row 1: Save As ───────────────────────────────────────────────────
	{
		y := startY + groupHeaderH + headerLineH
		if state.SelectedRow == 1 {
			rl.DrawText(">", bgX+pad, y+scaledInt32(3), arrowFont, rl.Yellow)
			rl.DrawRectangle(bgX+2, y-2, bgWidth-4, headerLineH-2, rl.Color{R: 50, G: 100, B: 150, A: 80})
		}
		saveLabel := "[Save As]"
		if state.KeybindsDirty {
			saveLabel = "[Save As] *"
		}
		rl.DrawText(saveLabel, leftX, y+scaledInt32(3), optionFont, rl.Green)
		savePath := state.SaveAsPath
		if savePath == "" {
			savePath = "configs/keybindings.json"
		}
		if state.SaveAsEditing {
			rl.DrawRectangle(leftX+scaledInt32(106)-2, y, bgWidth-scaledInt32(120), headerLineH-4, rl.Color{R: 20, G: 40, B: 20, A: 180})
			rl.DrawText(savePath+"|", leftX+scaledInt32(106), y+scaledInt32(3), descFont, rl.Yellow)
		} else {
			rl.DrawText(savePath, leftX+scaledInt32(106), y+scaledInt32(3), descFont, rl.LightGray)
		}
	}

	// ── Status line ──────────────────────────────────────────────────────
	statusY := startY + groupHeaderH + headerLineH*2
	if state.SaveAsEditing {
		rl.DrawText("Type path then ENTER to save -- ESC to cancel", leftX, statusY, descFont, rl.Yellow)
	} else if state.KeybindConflict != "" {
		rl.DrawText(state.KeybindConflict, leftX, statusY, descFont, rl.Red)
	} else if state.KeybindCapturePendingMod != 0 {
		modName := input.KeyName(state.KeybindCapturePendingMod)
		rl.DrawText(modName+" held — press another key, or release to bind "+modName+" alone", leftX, statusY, descFont, rl.Yellow)
	} else if state.KeybindCapture >= 0 {
		rl.DrawText("Press a key to bind... (ESC to cancel)", leftX, statusY, descFont, rl.Yellow)
	} else if state.KeybindsDirty {
		rl.DrawText("Unsaved changes — press ENTER on [Save As] to save", leftX, statusY, descFont, rl.Orange)
	}
	_ = statusH // used for layout reference

	// ── Two-column action list ───────────────────────────────────────────
	headerH := groupHeaderH + headerLineH*2 + statusH
	listStartY := startY + headerH

	allActions := input.OrderedActions()
	midpoint := (len(allActions) + 1) / 2 // left column count; right gets the rest

	// Column geometry
	colW := bgWidth / 2
	col1ArrowX := bgX + scaledInt32(4)
	col1NameX := col1ArrowX + scaledInt32(14)
	col2ArrowX := bgX + colW + scaledInt32(4)
	col2NameX := col2ArrowX + scaledInt32(14)
	keyOffset := scaledInt32(190) // x offset from name start to key label

	for actionIdx, action := range allActions {
		rowIdx := actionIdx + 2 // UI SelectedRow value for this action

		var visualRow int
		var nameX, arrowX, colBgX int32
		var colBgW int32
		if actionIdx < midpoint {
			visualRow = actionIdx
			nameX = col1NameX
			arrowX = col1ArrowX
			colBgX = bgX + 2
			colBgW = colW - 4
		} else {
			visualRow = actionIdx - midpoint
			nameX = col2NameX
			arrowX = col2ArrowX
			colBgX = bgX + colW + 2
			colBgW = colW - 4
		}
		y := listStartY + int32(visualRow)*actionLineH

		// Stop drawing if below content area.
		if y+actionLineH > startY+contentH {
			break
		}

		isSelected := state.SelectedRow == rowIdx
		isCapture := state.KeybindCapture == int(action)

		if isSelected {
			rl.DrawText(">", arrowX, y+scaledInt32(3), arrowFont, rl.Yellow)
			bgAlpha := uint8(80)
			if isCapture {
				bgAlpha = 120
			}
			rl.DrawRectangle(colBgX, y-1, colBgW, actionLineH-1, rl.Color{R: 50, G: 100, B: 150, A: bgAlpha})
		}

		// Action name
		nameColor := rl.White
		if isCapture {
			nameColor = rl.Yellow
		}
		rl.DrawText(action.String(), nameX, y+scaledInt32(3), nameFont, nameColor)

		// Bound key / capture hint
		kx := nameX + keyOffset
		if isCapture && state.KeybindCapturePendingMod != 0 {
			modName := input.KeyName(state.KeybindCapturePendingMod)
			rl.DrawText(modName+"+?", kx, y+scaledInt32(3), keyFont, rl.Yellow)
		} else if isCapture {
			rl.DrawText("→ key...", kx, y+scaledInt32(3), keyFont, rl.Yellow)
		} else {
			keyLabel := "[unbound]"
			if km != nil {
				if code := km.BoundKey(action); code != 0 {
					keyLabel = input.KeyName(code)
					if mods := km.BoundMods(action); mods != 0 {
						var parts []string
						if mods&input.ModShift != 0 {
							parts = append(parts, "SHF")
						}
						if mods&input.ModCtrl != 0 {
							parts = append(parts, "CTL")
						}
						if mods&input.ModAlt != 0 {
							parts = append(parts, "ALT")
						}
						keyLabel = strings.Join(parts, "+") + "+" + keyLabel
					}
				}
			}
			keyColor := rl.LightGray
			if keyLabel == "[unbound]" {
				keyColor = rl.Color{R: 90, G: 90, B: 90, A: 255}
			}
			rl.DrawText(keyLabel, kx, y+scaledInt32(3), keyFont, keyColor)
		}
	}

	// "Mouse" group header (placeholder — not yet configurable)
	mouseY := listStartY + int32(midpoint)*actionLineH + scaledInt32(12)
	if mouseY < startY+contentH-scaledInt32(36) {
		rl.DrawText("-- Mouse --", leftX, mouseY+scaledInt32(2), descFont, rl.SkyBlue)
		rl.DrawLine(bgX+2, mouseY+scaledInt32(17), bgX+bgWidth-4, mouseY+scaledInt32(17), rl.Color{R: 50, G: 80, B: 120, A: 180})
		rl.DrawText("(not yet configurable)", leftX, mouseY+scaledInt32(24), descFont, rl.Color{R: 90, G: 90, B: 90, A: 255})
	}
	if len(state.AvailableFiles) > 0 {
		overlayX := bgX + scaledInt32(40)
		overlayY := startY + scaledInt32(20)
		overlayW := bgWidth - scaledInt32(80)
		overlayH := scaledInt32(30) + int32(len(state.AvailableFiles))*scaledInt32(28) + scaledInt32(20)

		rl.DrawRectangle(overlayX, overlayY, overlayW, overlayH, rl.Color{R: 20, G: 30, B: 50, A: 240})
		rl.DrawRectangleLines(overlayX, overlayY, overlayW, overlayH, rl.SkyBlue)
		rl.DrawText("Select keybindings file  (UP/DOWN, ENTER to load, ESC cancel)",
			overlayX+scaledInt32(10), overlayY+scaledInt32(6), descFont, rl.SkyBlue)

		for i, f := range state.AvailableFiles {
			fy := overlayY + scaledInt32(30) + int32(i)*scaledInt32(28)
			rowColor := rl.LightGray
			if i == state.FilePickerIdx {
				rl.DrawRectangle(overlayX+2, fy-2, overlayW-4, scaledInt32(26), rl.Color{R: 50, G: 120, B: 180, A: 180})
				rowColor = rl.White
				rl.DrawText(">", overlayX+scaledInt32(8), fy+scaledInt32(4), keyFont, rl.Yellow)
			}
			rl.DrawText(filepath.Base(f), overlayX+scaledInt32(28), fy+scaledInt32(4), keyFont, rowColor)
		}
	}
}

// importanceThresholdLabel returns a human-readable label for a threshold value.
func importanceThresholdLabel(v int) string {
	switch v {
	case 0:
		return "0 (All objects)"
	case 5:
		return "5 (Hide tiny asteroids)"
	case 10:
		return "10 (Hide medium asteroids)"
	case 15:
		return "15 (Hide all asteroids)"
	case 30:
		return "30 (Hide rings)"
	case 40:
		return "40 (Hide small moons)"
	case 50:
		return "50 (Hide dwarf planets)"
	case 60:
		return "60 (Hide large moons)"
	case 70:
		return "70 (Hide ice giants)"
	case 80:
		return "80 (Hide rocky planets)"
	case 90:
		return "90 (Hide gas giants)"
	default:
		return fmt.Sprintf("%d", v)
	}
}


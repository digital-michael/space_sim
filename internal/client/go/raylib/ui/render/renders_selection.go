package render

import (
	"fmt"
	"path/filepath"
	"strings"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func drawSelectionUI(state *engine.SimulationState, inputState *ui.InputState) {
	if inputState.SelectionMode == ui.SelectionModeSystemSelector {
		drawSystemSelectorUI(inputState)
		return
	}

	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	hintFont := scaledInt32(12)
	filterFont := scaledInt32(16)
	filterHintFont := scaledInt32(14)
	tabFont := scaledInt32(18)
	itemFont := scaledInt32(20)
	itemSubFont := scaledInt32(16)
	catLabelFont := scaledInt32(13)
	arrowFont := scaledInt32(20)

	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(400) {
		bgWidth = scaledInt32(400)
	}
	if bgWidth > scaledInt32(700) {
		bgWidth = scaledInt32(700)
	}
	bgHeight := bgWidth
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 200})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Left sidebar for vertical category tabs
	tabSidebarW := scaledInt32(155)
	contentX := bgX + tabSidebarW
	contentW := bgWidth - tabSidebarW

	// Precompute list startY: bgY + radio(30) + hints(30) + filter(30) + gap(5) + gap(5) + 10 padding = bgY+110
	// Used for both tab alignment and list rendering.
	listStartY := bgY + scaledInt32(110)

	// Vertical divider
	rl.DrawLine(contentX, bgY+1, contentX, bgY+bgHeight-1, rl.Color{R: 70, G: 70, B: 70, A: 255})

	// Category tabs — all categories always shown, active ones highlighted
	allCategories := []engine.ObjectCategory{
		engine.CategoryBlackHole,
		engine.CategoryStar,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryRing,
		engine.CategoryBelt,
		engine.CategoryRogue,
		engine.CategoryArtifact,
	}
	present := make(map[engine.ObjectCategory]bool, len(state.NavigationOrder))
	for _, cat := range state.NavigationOrder {
		present[cat] = true
	}
	tabH := scaledInt32(28)
	tabGap := scaledInt32(3)
	for i, cat := range allCategories {
		ty := listStartY + int32(i)*(tabH+tabGap)
		tx := bgX + scaledInt32(3)
		tw := tabSidebarW - scaledInt32(6)
		hasData := present[cat]
		tabBg := rl.Color{R: 35, G: 35, B: 35, A: 255}
		textColor := rl.Color{R: 155, G: 155, B: 155, A: 255}
		if !hasData {
			tabBg = rl.Color{R: 22, G: 22, B: 22, A: 255}
			textColor = rl.Color{R: 70, G: 70, B: 70, A: 255}
		}
		if cat == inputState.SelectedCategory {
			tabBg = rl.Color{R: 60, G: 100, B: 145, A: 255}
			textColor = rl.White
		}
		rl.DrawRectangle(tx, ty, tw, tabH, tabBg)
		rl.DrawRectangleLines(tx, ty, tw, tabH, rl.Color{R: 60, G: 60, B: 60, A: 255})
		name := categoryDisplayName(cat)
		rl.DrawText(name, tx+scaledInt32(5), ty+(tabH-tabFont)/2, tabFont, textColor)
	}

	// Radio buttons: Face / Jump / Track — TAB/SHIFT+TAB cycles.
	type radioOption struct {
		label string
		mode  ui.SelectionMode
	}
	radioOptions := []radioOption{
		{"Face", ui.SelectionModeFace},
		{"Jump", ui.SelectionModeJump},
		{"Track", ui.SelectionModeTrack},
	}
	radioFont := scaledInt32(18)
	radioY := bgY + scaledInt32(12)
	radioSpacing := contentW / int32(len(radioOptions)+1)
	for i, opt := range radioOptions {
		selected := inputState.SelectionMode == opt.mode
		cx := contentX + radioSpacing*int32(i+1)
		cy := radioY + scaledInt32(9)
		r := float32(scaledInt32(6))
		col := rl.Gray
		if selected {
			col = rl.White
		}
		if selected {
			rl.DrawCircle(cx, cy, r, col)
		} else {
			rl.DrawCircleLines(cx, cy, r, col)
		}
		rl.DrawText(opt.label, cx+scaledInt32(10), radioY, radioFont, col)
	}
	rl.DrawText("UP/DOWN: select  LEFT/RIGHT: tab  ENTER: confirm  ESC: cancel", contentX+scaledInt32(5), bgY+scaledInt32(40), hintFont, rl.LightGray)
	rl.DrawText("TAB/SHIFT+TAB: mode    PgUp/PgDn: page    HOME/END: top/end", contentX+scaledInt32(5), bgY+scaledInt32(55), hintFont, rl.Gray)

	// Filter text box
	filterY := bgY + scaledInt32(75)
	filterBoxHeight := scaledInt32(25)
	if inputState.FilterText != "" {
		rl.DrawRectangle(contentX+scaledInt32(5), filterY, contentW-scaledInt32(10), filterBoxHeight, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(contentX+scaledInt32(5), filterY, contentW-scaledInt32(10), filterBoxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawText("Filter: "+inputState.FilterText+"_", contentX+scaledInt32(10), filterY+scaledInt32(5), filterFont, rl.Green)
		filterY += filterBoxHeight + scaledInt32(5)
	} else {
		rl.DrawText("Type to filter...", contentX+scaledInt32(10), filterY+scaledInt32(5), filterHintFont, rl.Gray)
		filterY += filterBoxHeight + scaledInt32(5)
	}

	// Filter objects by category
	if len(inputState.FilteredIndices) == 0 {
		inputState.FilteredIndices = filterObjectsByCategory(state.Objects, inputState.SelectedCategory)
	}

	startY := listStartY
	lineHeight := scaledInt32(30)
	listAreaHeight := bgHeight - (startY - bgY) - scaledInt32(10)
	visibleItems := int(listAreaHeight / lineHeight)
	totalItems := len(inputState.FilteredIndices)

	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if inputState.ScrollOffset > maxScroll {
		inputState.ScrollOffset = maxScroll
	}
	if inputState.ScrollOffset < 0 {
		inputState.ScrollOffset = 0
	}

	// Update distance cache every 5 seconds
	currentTime := rl.GetTime()
	if currentTime-inputState.LastDistanceUpdate > 5.0 {
		inputState.DistanceCache = make(map[int]string)
		for _, idx := range inputState.FilteredIndices {
			if idx < 0 {
				continue
			}
			obj := state.Objects[idx]
			dist := obj.Anim.Position.Sub(engine.Vector3{}).Length()
			inputState.DistanceCache[idx] = fmt.Sprintf("%.0f units", dist)
		}
		inputState.LastDistanceUpdate = currentTime
	}

	filterActive := inputState.FilterText != ""

	// Render visible items
	for i := inputState.ScrollOffset; i < inputState.ScrollOffset+visibleItems && i < totalItems; i++ {
		actualIndex := inputState.FilteredIndices[i]
		y := startY + int32(i-inputState.ScrollOffset)*lineHeight

		if i == inputState.SelectedIndex {
			rl.DrawRectangle(contentX+scaledInt32(3), y-scaledInt32(2), contentW-scaledInt32(6), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", contentX+scaledInt32(15), y+scaledInt32(5), arrowFont, rl.Yellow)
		}

		// Right edge for dist text: just left of the color swatch (swatch at contentW-28, gap 5)
		swatchX := contentX + contentW - scaledInt32(28)
		swatchRect := func(col rl.Color) {
			rl.DrawRectangleRec(rl.Rectangle{X: float32(swatchX), Y: float32(y + scaledInt32(7)), Width: float32(scaledInt32(16)), Height: float32(scaledInt32(16))}, col)
		}
		distRight := swatchX - scaledInt32(5)

		if actualIndex == -1 {
			rl.DrawText("Asteroid Belt", contentX+scaledInt32(35), y+scaledInt32(5), itemFont, rl.White)
			d := "195-240 AU"
			rl.DrawText(d, distRight-rl.MeasureText(d, itemSubFont), y+scaledInt32(8), itemSubFont, rl.LightGray)
			swatchRect(rl.Color{R: 150, G: 150, B: 150, A: 255})
		} else if actualIndex == -2 {
			rl.DrawText("Kuiper Belt", contentX+scaledInt32(35), y+scaledInt32(5), itemFont, rl.White)
			d := "3000-5000 AU"
			rl.DrawText(d, distRight-rl.MeasureText(d, itemSubFont), y+scaledInt32(8), itemSubFont, rl.LightGray)
			swatchRect(rl.Color{R: 200, G: 150, B: 130, A: 255})
		} else {
			obj := state.Objects[actualIndex]
			nameText := obj.Meta.Name
			objColor := rl.Color{R: obj.Meta.Color.R, G: obj.Meta.Color.G, B: obj.Meta.Color.B, A: 255}
			if filterActive {
				// Cross-category: leading color swatch + name + right-aligned category label
				rl.DrawRectangleRec(rl.Rectangle{X: float32(contentX + scaledInt32(22)), Y: float32(y + scaledInt32(8)), Width: float32(scaledInt32(14)), Height: float32(scaledInt32(14))}, objColor)
				rl.DrawText(nameText, contentX+scaledInt32(42), y+scaledInt32(5), itemFont, rl.White)
				catLabel := categoryShortLabel(obj.Meta.Category)
				catLabelW := rl.MeasureText(catLabel, catLabelFont)
				rl.DrawText(catLabel, contentX+contentW-catLabelW-scaledInt32(10), y+scaledInt32(9), catLabelFont, rl.Color{R: 120, G: 190, B: 120, A: 255})
			} else {
				// Normal: name + right-aligned dist + color swatch
				distText := inputState.DistanceCache[actualIndex]
				if distText == "" {
					distText = "--- units"
				}
				rl.DrawText(nameText, contentX+scaledInt32(35), y+scaledInt32(5), itemFont, rl.White)
				rl.DrawText(distText, distRight-rl.MeasureText(distText, itemSubFont), y+scaledInt32(8), itemSubFont, rl.LightGray)
				swatchRect(objColor)
			}
		}
	}

	// Scrollbar
	if totalItems > visibleItems {
		scrollBarX := contentX + contentW - scaledInt32(15)
		scrollBarY := startY
		scrollBarHeight := listAreaHeight
		scrollBarWidth := scaledInt32(10)
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})
		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < scaledInt32(20) {
			thumbHeight = scaledInt32(20)
		}
		thumbY := scrollBarY
		if maxScroll > 0 {
			thumbY += int32(float32(inputState.ScrollOffset) / float32(maxScroll) * float32(scrollBarHeight-thumbHeight))
		}
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
	}
}

func categoryDisplayName(cat engine.ObjectCategory) string {
	switch cat {
	case engine.CategoryPlanet:
		return "Planets"
	case engine.CategoryDwarfPlanet:
		return "Dwarf Planets"
	case engine.CategoryMoon:
		return "Moons"
	case engine.CategoryAsteroid:
		return "Asteroids"
	case engine.CategoryRing:
		return "Ring Systems"
	case engine.CategoryStar:
		return "Stars"
	case engine.CategoryBelt:
		return "Belts"
	case engine.CategoryRogue:
		return "Rogues"
	case engine.CategoryArtifact:
		return "Artifacts"
	case engine.CategoryBlackHole:
		return "Black Holes"
	default:
		return "Objects"
	}
}

func drawSystemSelectorUI(inputState *ui.InputState) {
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	titleFont := scaledInt32(20)
	hintFont := scaledInt32(12)
	itemFont := scaledInt32(18)
	statusFont := scaledInt32(14)
	arrowFont := scaledInt32(20)

	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(420) {
		bgWidth = scaledInt32(420)
	}
	if bgWidth > scaledInt32(760) {
		bgWidth = scaledInt32(760)
	}
	bgHeight := bgWidth
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2

	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 210})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	titleText := "SELECT RUNTIME SYSTEM"
	rl.DrawText(titleText, bgX+scaledInt32(18), bgY+scaledInt32(12), titleFont, rl.White)
	rl.DrawText("UP/DOWN: select, ENTER: load, ESC: cancel", bgX+scaledInt32(18), bgY+scaledInt32(42), hintFont, rl.LightGray)
	rl.DrawText(modSuper+"+S opens this selector", bgX+scaledInt32(18), bgY+scaledInt32(58), hintFont, rl.Gray)

	activeLabel := ""
	for _, option := range inputState.SystemOptions {
		if option.Path == inputState.ActiveSystemPath {
			activeLabel = option.DisplayName
			if activeLabel == "" {
				activeLabel = option.Label
			}
			break
		}
	}
	if activeLabel == "" {
		activeLabel = filepath.Base(inputState.ActiveSystemPath)
		if activeLabel == "." || activeLabel == string(filepath.Separator) || activeLabel == "" {
			activeLabel = inputState.ActiveSystemPath
		}
	}
	rl.DrawText("Current: "+activeLabel, bgX+scaledInt32(18), bgY+scaledInt32(86), statusFont, rl.Color{R: 140, G: 210, B: 255, A: 255})

	listStartY := bgY + scaledInt32(116)
	lineHeight := scaledInt32(32)
	listHeight := bgHeight - scaledInt32(176)
	visibleItems := int(listHeight / lineHeight)
	if visibleItems < 1 {
		visibleItems = 1
	}

	totalItems := len(inputState.SystemOptions)
	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if inputState.ScrollOffset > maxScroll {
		inputState.ScrollOffset = maxScroll
	}
	if inputState.ScrollOffset < 0 {
		inputState.ScrollOffset = 0
	}

	if totalItems == 0 {
		rl.DrawText("No runtime systems available.", bgX+scaledInt32(18), listStartY, itemFont, rl.LightGray)
	} else {
		for idx := inputState.ScrollOffset; idx < inputState.ScrollOffset+visibleItems && idx < totalItems; idx++ {
			option := inputState.SystemOptions[idx]
			y := listStartY + int32(idx-inputState.ScrollOffset)*lineHeight

			if idx == inputState.SelectedIndex {
				rl.DrawRectangle(bgX+scaledInt32(10), y-scaledInt32(2), bgWidth-scaledInt32(20), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
				rl.DrawText(">", bgX+scaledInt32(18), y+scaledInt32(4), arrowFont, rl.Yellow)
			}

			labelColor := rl.White
			suffix := ""
			labelText := option.DisplayName
			if labelText == "" {
				labelText = option.Label
			}
			if option.Path == inputState.ActiveSystemPath {
				labelColor = rl.Color{R: 140, G: 210, B: 255, A: 255}
				suffix = " [current]"
			}

			rl.DrawText(labelText+suffix, bgX+scaledInt32(44), y+scaledInt32(4), itemFont, labelColor)
		}
	}

	if totalItems > visibleItems {
		scrollBarX := bgX + bgWidth - scaledInt32(15)
		scrollBarY := listStartY
		scrollBarWidth := scaledInt32(10)
		scrollBarHeight := listHeight
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})

		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < scaledInt32(20) {
			thumbHeight = scaledInt32(20)
		}
		thumbY := scrollBarY
		if maxScroll > 0 {
			thumbY += int32(float32(inputState.ScrollOffset) / float32(maxScroll) * float32(scrollBarHeight-thumbHeight))
		}
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
	}

	statusText := inputState.SystemStatusMessage
	if statusText == "" && inputState.SelectedIndex >= 0 && inputState.SelectedIndex < len(inputState.SystemOptions) {
		selected := inputState.SystemOptions[inputState.SelectedIndex]
		if selected.Path == inputState.ActiveSystemPath {
			statusText = "Press ENTER to close without reloading the current system."
		} else {
			statusText = "Press ENTER to load the highlighted system."
		}
	}
	if statusText != "" {
		rl.DrawText(statusText, bgX+scaledInt32(18), bgY+bgHeight-scaledInt32(34), statusFont, rl.LightGray)
	}
}

// filterObjectsByCategory returns indices of objects matching the given category
func filterObjectsByCategory(objects []*engine.Object, category engine.ObjectCategory) []int {
	var indices []int

	// Special handling for belt category - return virtual entries for belts
	if category == engine.CategoryBelt {
		// Check if there are any asteroids (Asteroid Belt)
		hasAsteroids := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "Asteroid-") {
				hasAsteroids = true
				break
			}
		}

		// Check if there are any Kuiper Belt objects
		hasKuiper := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "KBO-") {
				hasKuiper = true
				break
			}
		}

		// Return virtual indices: -1 for Asteroid Belt, -2 for Kuiper Belt
		if hasAsteroids {
			indices = append(indices, -1)
		}
		if hasKuiper {
			indices = append(indices, -2)
		}
		return indices
	}

	// Normal category filtering
	for i, obj := range objects {
		if obj.Meta.Category == category {
			indices = append(indices, i)
		}
	}
	return indices
}

// filterObjectsByCategoryAndText filters objects by category and optional text search
func filterObjectsByCategoryAndText(objects []*engine.Object, category engine.ObjectCategory, filterText string) []int {
	var indices []int
	lowerFilter := strings.ToLower(filterText)

	// Special handling for belt category
	if category == engine.CategoryBelt {
		// Check if there are any asteroids (Asteroid Belt)
		hasAsteroids := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "Asteroid-") {
				hasAsteroids = true
				break
			}
		}

		// Check if there are any Kuiper Belt objects
		hasKuiper := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "KBO-") {
				hasKuiper = true
				break
			}
		}

		// Filter by text if provided
		if filterText == "" {
			if hasAsteroids {
				indices = append(indices, -1)
			}
			if hasKuiper {
				indices = append(indices, -2)
			}
		} else {
			if hasAsteroids && strings.Contains("asteroid belt", lowerFilter) {
				indices = append(indices, -1)
			}
			if hasKuiper && strings.Contains("kuiper belt", lowerFilter) {
				indices = append(indices, -2)
			}
		}
		return indices
	}

	// Normal category filtering
	for i, obj := range objects {
		// First check category match
		if obj.Meta.Category != category {
			continue
		}

		// If no filter text, include all objects in category
		if filterText == "" {
			indices = append(indices, i)
			continue
		}

		// Check if object name contains the filter text (case-insensitive)
		lowerName := strings.ToLower(obj.Meta.Name)
		if strings.Contains(lowerName, lowerFilter) {
			indices = append(indices, i)
		}
	}
	return indices
}

func categoryShortLabel(cat engine.ObjectCategory) string {
	switch cat {
	case engine.CategoryPlanet:
		return "planet"
	case engine.CategoryDwarfPlanet:
		return "dwarf"
	case engine.CategoryMoon:
		return "moon"
	case engine.CategoryAsteroid:
		return "asteroid"
	case engine.CategoryRing:
		return "ring"
	case engine.CategoryStar:
		return "star"
	case engine.CategoryBelt:
		return "belt"
	case engine.CategoryRogue:
		return "rogue"
	case engine.CategoryArtifact:
		return "artifact"
	case engine.CategoryBlackHole:
		return "blackhole"
	default:
		return "object"
	}
}

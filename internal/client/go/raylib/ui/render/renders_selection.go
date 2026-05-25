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

	// Semi-transparent background - responsive to screen size
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	titleFont := scaledInt32(20)
	hintFont := scaledInt32(12)
	filterFont := scaledInt32(16)
	filterHintFont := scaledInt32(14)
	tabFont := scaledInt32(14)
	itemFont := scaledInt32(20)
	itemSubFont := scaledInt32(16)
	arrowFont := scaledInt32(20)
	// Panel is 40% of screen width, clamped to reasonable bounds (400-700)
	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(400) {
		bgWidth = scaledInt32(400)
	}
	if bgWidth > scaledInt32(700) {
		bgWidth = scaledInt32(700)
	}
	// Panel height matches width for square aspect
	bgHeight := bgWidth
	// Center on screen
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 200})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Title - show different text based on mode
	titleText := "SELECT OBJECT"
	if inputState.SelectionMode == ui.SelectionModeJump {
		titleText = "SELECT OBJECT TO JUMP TO"
	} else if inputState.SelectionMode == ui.SelectionModeTrack {
		titleText = "SELECT OBJECT TO TRACK"
	} else if inputState.SelectionMode == ui.SelectionModeTrackEquatorial {
		titleText = "SELECT OBJECT (EQUATORIAL)"
	}
	rl.DrawText(titleText, bgX+scaledInt32(50), bgY+scaledInt32(10), titleFont, rl.White)
	rl.DrawText("UP/DOWN: select, LEFT/RIGHT: category, ENTER: confirm, ESC: cancel", bgX+scaledInt32(10), bgY+scaledInt32(40), hintFont, rl.LightGray)
	rl.DrawText("PgUp/PgDn: page, HOME/END: jump to start/end", bgX+scaledInt32(10), bgY+scaledInt32(55), hintFont, rl.Gray)

	// Filter text box
	filterY := bgY + scaledInt32(75)
	filterBoxHeight := scaledInt32(25)
	if inputState.FilterText != "" {
		rl.DrawRectangle(bgX+scaledInt32(10), filterY, bgWidth-scaledInt32(20), filterBoxHeight, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(bgX+scaledInt32(10), filterY, bgWidth-scaledInt32(20), filterBoxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		filterDisplay := "Filter: " + inputState.FilterText + "_"
		rl.DrawText(filterDisplay, bgX+scaledInt32(15), filterY+scaledInt32(5), filterFont, rl.Green)
		filterY += filterBoxHeight + scaledInt32(5)
	} else {
		rl.DrawText("Type to filter...", bgX+scaledInt32(15), filterY+scaledInt32(5), filterHintFont, rl.Gray)
		filterY += filterBoxHeight + scaledInt32(5)
	}

	// Category tabs - map display order to ObjectCategory enum values
	type categoryTab struct {
		name     string
		category engine.ObjectCategory
	}
	categories := []categoryTab{
		{"Black Holes", engine.CategoryBlackHole},
		{"Stars", engine.CategoryStar}, // First tab
		{"Planets", engine.CategoryPlanet},
		{"Dwarf Planets", engine.CategoryDwarfPlanet},
		{"Moons", engine.CategoryMoon},
		{"Ring Systems", engine.CategoryRing},
		{"Belts", engine.CategoryBelt}, // Asteroid Belt and Kuiper Belt
	}
	// Calculate tab width based on panel width and number of tabs
	numTabs := int32(len(categories))
	tabWidth := (bgWidth - scaledInt32(20)) / numTabs // 20 pixels margin
	if tabWidth < scaledInt32(60) {
		tabWidth = scaledInt32(60)
	}
	tabHeight := scaledInt32(30)
	tabY := filterY

	for i, cat := range categories {
		tabX := bgX + scaledInt32(10) + int32(i)*tabWidth
		tabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
		textColor := rl.LightGray

		// Highlight active category
		if cat.category == inputState.SelectedCategory {
			tabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
			textColor = rl.White
		}

		rl.DrawRectangle(tabX, tabY, tabWidth-scaledInt32(5), tabHeight, tabColor)
		rl.DrawRectangleLines(tabX, tabY, tabWidth-scaledInt32(5), tabHeight, rl.White)
		rl.DrawText(cat.name, tabX+scaledInt32(5), tabY+scaledInt32(8), tabFont, textColor)
	}

	// Filter objects by category
	if len(inputState.FilteredIndices) == 0 {
		inputState.FilteredIndices = filterObjectsByCategory(state.Objects, inputState.SelectedCategory)
	}

	// Object list (filtered by category) - start below the tabs
	startY := tabY + tabHeight + scaledInt32(10)
	lineHeight := scaledInt32(30)
	listAreaHeight := bgHeight - (startY - bgY) - scaledInt32(10) // Available height for list
	visibleItems := int(listAreaHeight / lineHeight)
	totalItems := len(inputState.FilteredIndices)

	// Calculate scroll bounds
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
			// Skip virtual belt indices
			if idx < 0 {
				continue
			}
			obj := state.Objects[idx]
			dist := obj.Anim.Position.Sub(engine.Vector3{}).Length()
			inputState.DistanceCache[idx] = fmt.Sprintf("%.0f units", dist)
		}
		inputState.LastDistanceUpdate = currentTime
	}

	// Render only visible items
	for i := inputState.ScrollOffset; i < inputState.ScrollOffset+visibleItems && i < totalItems; i++ {
		actualIndex := inputState.FilteredIndices[i]
		y := startY + int32(i-inputState.ScrollOffset)*lineHeight

		// Highlight selected
		if i == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+scaledInt32(5), y-scaledInt32(2), bgWidth-scaledInt32(10), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+scaledInt32(15), y+scaledInt32(5), arrowFont, rl.Yellow)
		}

		// Handle virtual belt indices
		if actualIndex == -1 {
			// Asteroid Belt
			rl.DrawText("Asteroid Belt", bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText("195-240 AU", bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}, rl.Color{R: 150, G: 150, B: 150, A: 255})
		} else if actualIndex == -2 {
			// Kuiper Belt
			rl.DrawText("Kuiper Belt", bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText("3000-5000 AU", bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}, rl.Color{R: 200, G: 150, B: 130, A: 255})
		} else {
			// Normal object
			obj := state.Objects[actualIndex]

			// Object name and info
			nameText := fmt.Sprintf("%s", obj.Meta.Name)
			distText := inputState.DistanceCache[actualIndex]
			if distText == "" {
				distText = "--- units" // Placeholder until first update
			}

			rl.DrawText(nameText, bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText(distText, bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)

			// Color indicator
			colorBox := rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}
			rl.DrawRectangleRec(colorBox, rl.Color{R: obj.Meta.Color.R, G: obj.Meta.Color.G, B: obj.Meta.Color.B, A: 255})
		}
	}

	// Draw scroll bar if needed
	if totalItems > visibleItems {
		scrollBarX := bgX + bgWidth - scaledInt32(15)
		scrollBarY := startY
		scrollBarHeight := listAreaHeight
		scrollBarWidth := scaledInt32(10)

		// Scroll bar background
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})

		// Scroll thumb
		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < scaledInt32(20) {
			thumbHeight = scaledInt32(20) // Minimum thumb size
		}
		thumbY := scrollBarY + int32(float32(inputState.ScrollOffset)/float32(maxScroll)*float32(scrollBarHeight-thumbHeight))
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
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

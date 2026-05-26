package app

import (
	"fmt"
	"sort"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

// handleInputNav processes object navigation keybindings:
// child/parent drill (G/B), sibling cycling (TAB), and the combined selector (/).
func (a *App) handleInputNav(session *runtimeSession, state *engine.SimulationState, km *input.KeyMap, suspended bool) {
	if suspended {
		return
	}

	// nav.child_next / nav.parent: drill down to child or up to parent (tracking mode only).
	if session.cameraState.Mode == ui.CameraModeTracking {
		if km.IsPressed(input.ActionNavChildNext) {
			idx := session.cameraState.Tracking.TargetIndex
			if idx >= 0 && idx < len(state.Objects) {
				currentObj := state.Objects[idx]
				if a.cfg.Debug {
					fmt.Printf("[DEBUG] F key: Drilling down from %s\n", currentObj.Meta.Name)
				}
				type ChildInfo struct {
					index    int
					distance float32
				}
				var children []ChildInfo
				isStar := currentObj.Meta.Category == engine.CategoryStar
				for i, obj := range state.Objects {
					isChild := obj.Visible && obj.Meta.ParentName == currentObj.Meta.Name
					if !isChild && isStar && obj.Visible && obj.Meta.ParentName == "" &&
						(obj.Meta.SemiMajorAxis > 0 || obj.Meta.OrbitRadius > 0) {
						isChild = true
					}
					if isChild && obj.Meta.Category != engine.CategoryRing {
						dist := obj.Meta.SemiMajorAxis
						if dist == 0 {
							dist = obj.Meta.OrbitRadius
						}
						children = append(children, ChildInfo{index: i, distance: dist})
					}
				}
				if a.cfg.Debug {
					fmt.Printf("[DEBUG] Found %d children\n", len(children))
				}
				if len(children) > 0 {
					sort.Slice(children, func(i, j int) bool {
						return children[i].distance < children[j].distance
					})
					child := state.Objects[children[0].index]
					if a.cfg.Debug {
						fmt.Printf("[DEBUG] Tracking closest child: %s\n", child.Meta.Name)
					}
					session.cameraState.StartTracking(children[0].index)
					session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(child.Meta.PhysicalRadius, 0.40)
				}
			}
		}

		if km.IsPressed(input.ActionNavParent) {
			idx := session.cameraState.Tracking.TargetIndex
			if idx >= 0 && idx < len(state.Objects) {
				currentObj := state.Objects[idx]
				if a.cfg.Debug {
					fmt.Printf("[DEBUG] B key: Moving to parent of %s\n", currentObj.Meta.Name)
				}
				if currentObj.Meta.ParentName != "" {
					for i, obj := range state.Objects {
						if obj.Meta.Name == currentObj.Meta.ParentName {
							if a.cfg.Debug {
								fmt.Printf("[DEBUG] Found parent: %s\n", obj.Meta.Name)
							}
							session.cameraState.StartTracking(i)
							session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.40)
							break
						}
					}
				} else if currentObj.Meta.SemiMajorAxis > 0 || currentObj.Meta.OrbitRadius > 0 {
					if a.cfg.Debug {
						fmt.Printf("[DEBUG] No explicit parent, looking for central star\n")
					}
					for i, obj := range state.Objects {
						if obj.Meta.Category == engine.CategoryStar {
							if a.cfg.Debug {
								fmt.Printf("[DEBUG] Found central star: %s\n", obj.Meta.Name)
							}
							session.cameraState.StartTracking(i)
							session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.40)
							break
						}
					}
				} else if a.cfg.Debug {
					fmt.Printf("[DEBUG] No parent for %s (already at star)\n", currentObj.Meta.Name)
				}
			}
		}
	}

	// nav.sibling_next / nav.sibling_prev: cycle siblings (tracking mode only).
	if session.cameraState.Mode == ui.CameraModeTracking {
		sibFwd := km.IsPressed(input.ActionNavSiblingNext)
		sibBack := km.IsPressed(input.ActionNavSiblingPrev)
		if sibFwd || sibBack {
			if a.cfg.Debug {
				fmt.Printf("[DEBUG] sibling nav: forward=%v back=%v\n", sibFwd, sibBack)
			}
			idx := session.cameraState.Tracking.TargetIndex
			if idx >= 0 && idx < len(state.Objects) {
				currentObj := state.Objects[idx]
				var siblings []int
				for i, obj := range state.Objects {
					if obj.Visible &&
						obj.Meta.Category == currentObj.Meta.Category &&
						obj.Meta.ParentName == currentObj.Meta.ParentName {
						siblings = append(siblings, i)
					}
				}
				if len(siblings) > 1 {
					currentPos := -1
					for i, si := range siblings {
						if si == idx {
							currentPos = i
							break
						}
					}
					if currentPos >= 0 {
						var nextPos int
						if sibBack {
							nextPos = currentPos - 1
							if nextPos < 0 {
								nextPos = len(siblings) - 1
							}
						} else {
							nextPos = currentPos + 1
							if nextPos >= len(siblings) {
								nextPos = 0
							}
						}
						nextObj := state.Objects[siblings[nextPos]]
						session.cameraState.StartTracking(siblings[nextPos])
						session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(nextObj.Meta.PhysicalRadius, 0.40)
					}
				}
			}
		}
	}

	// nav.select: open the combined Face/Jump/Track selector dialog.
	if km.IsPressed(input.ActionNavSelect) {
		session.inputState.StartSelection(ui.SelectionModeJump)
		session.inputState.FilterText = ""
		session.inputState.ScrollOffset = 0
		session.inputState.FilteredIndices = filterObjectsByCategoryAndText(state.Objects, session.inputState.SelectedCategory, session.inputState.FilterText)
	}
}

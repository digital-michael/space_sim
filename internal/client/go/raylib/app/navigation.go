package app

import (
	"strings"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

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

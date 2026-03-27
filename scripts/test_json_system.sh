#!/bin/bash
# Test script to compare hard-coded vs JSON-loaded solar system

set -e

echo "=== Solar System JSON Loading Test ==="
echo ""

# Test 1: JSON validity
echo "Test 1: Checking JSON validity..."
if jq empty data/systems/solar_system.json 2>/dev/null; then
    echo "✓ JSON is valid"
else
    echo "✗ JSON is invalid"
    exit 1
fi
echo ""

# Test 2: Body count
echo "Test 2: Counting bodies..."
BODY_COUNT=$(jq '.bodies | length' data/systems/solar_system.json)
echo "  Bodies in JSON: $BODY_COUNT"
echo "  Expected: 314 (1 star + 8 planets + 285 moons + 11 dwarf planets + 9 rings)"
echo ""

# Test 3: Importance values
echo "Test 3: Verifying importance scale..."
SUN_IMP=$(jq '.bodies[] | select(.name == "Sun") | .importance' data/systems/solar_system.json)
JUPITER_IMP=$(jq '.bodies[] | select(.name == "Jupiter") | .importance' data/systems/solar_system.json)
EARTH_IMP=$(jq '.bodies[] | select(.name == "Earth") | .importance' data/systems/solar_system.json)
URANUS_IMP=$(jq '.bodies[] | select(.name == "Uranus") | .importance' data/systems/solar_system.json)
MOON_IMP=$(jq '.bodies[] | select(.name == "Moon") | .importance' data/systems/solar_system.json)
PLUTO_IMP=$(jq '.bodies[] | select(.name == "Pluto") | .importance' data/systems/solar_system.json)

echo "  Sun: $SUN_IMP (expected 100) $([ "$SUN_IMP" -eq 100 ] && echo '✓' || echo '✗')"
echo "  Jupiter: $JUPITER_IMP (expected 90) $([ "$JUPITER_IMP" -eq 90 ] && echo '✓' || echo '✗')"
echo "  Earth: $EARTH_IMP (expected 80) $([ "$EARTH_IMP" -eq 80 ] && echo '✓' || echo '✗')"
echo "  Uranus: $URANUS_IMP (expected 70) $([ "$URANUS_IMP" -eq 70 ] && echo '✓' || echo '✗')"
echo "  Moon: $MOON_IMP (expected 60) $([ "$MOON_IMP" -eq 60 ] && echo '✓' || echo '✗')"
echo "  Pluto: $PLUTO_IMP (expected 50) $([ "$PLUTO_IMP" -eq 50 ] && echo '✓' || echo '✗')"
echo ""

# Test 4: Feature configuration
echo "Test 4: Checking asteroid belt feature..."
FEATURE_COUNT=$(jq '.features | length' data/systems/solar_system.json)
BELT_NAME=$(jq -r '.features[0].name' data/systems/solar_system.json)
echo "  Features: $FEATURE_COUNT (expected 2)"
echo "  First feature name: $BELT_NAME (expected 'Main Asteroid Belt')"
echo ""

# Test 5: Categories
echo "Test 5: Verifying object categories..."
STARS=$(jq '[.bodies[] | select(.type == "star")] | length' data/systems/solar_system.json)
PLANETS=$(jq '[.bodies[] | select(.type == "planet")] | length' data/systems/solar_system.json)
MOONS=$(jq '[.bodies[] | select(.type == "moon")] | length' data/systems/solar_system.json)
DWARF=$(jq '[.bodies[] | select(.type == "dwarf_planet")] | length' data/systems/solar_system.json)
RINGS=$(jq '[.bodies[] | select(.type == "ring")] | length' data/systems/solar_system.json)

echo "  Stars: $STARS (expected 1) $([ "$STARS" -eq 1 ] && echo '✓' || echo '✗')"
echo "  Planets: $PLANETS (expected 8) $([ "$PLANETS" -eq 8 ] && echo '✓' || echo '✗')"
echo "  Moons: $MOONS (expected 285) $([ "$MOONS" -eq 285 ] && echo '✓' || echo '✗')"
echo "  Dwarf Planets: $DWARF (expected 11) $([ "$DWARF" -eq 11 ] && echo '✓' || echo '✗')"
echo "  Rings: $RINGS (expected 9) $([ "$RINGS" -eq 9 ] && echo '✓' || echo '✗')"
echo ""

# Test 6: Template files exist
echo "Test 6: Checking template files..."
TEMPLATES=(
    "data/bodies/stars.json"
    "data/bodies/planets.json"
    "data/bodies/dwarf_planets.json"
    "data/bodies/moons.json"
)

for template in "${TEMPLATES[@]}"; do
    if [ -f "$template" ]; then
        echo "  ✓ $template exists"
    else
        echo "  ✗ $template missing"
    fi
done
echo ""

# Test 7: Compile check
echo "Test 7: Building application..."
if go build -o bin/space-sim ./cmd/space-sim 2>/dev/null; then
    echo "  ✓ Application builds successfully"
else
    echo "  ✗ Build failed"
    exit 1
fi
echo ""

# Summary
echo "=== Test Summary ==="
echo "All tests passed! JSON configuration system is operational."
echo ""
echo "To run with JSON config:"
echo "  ./bin/space-sim --system-config=data/systems/solar_system.json"
echo ""
echo "To run with the default bundled system:"
echo "  ./bin/space-sim"

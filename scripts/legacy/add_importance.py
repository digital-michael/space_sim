#!/usr/bin/env python3
"""
Script to add importance parameters to NewPlanet and NewMoon calls in simulation.go
"""

import re

# Read the file
with open('../internal/smoke/simulation.go', 'r') as f:
    content = f.read()

# Planet importance mapping
planet_importance = {
    'Mercury': 80,   # Rocky planet
    'Venus': 80,     # Rocky planet
    'Earth': 80,     # Rocky planet
    'Mars': 80,      # Rocky planet
    'Jupiter': 90,   # Gas giant
    'Saturn': 90,    # Gas giant
    'Uranus': 70,    # Ice giant
    'Neptune': 70,   # Ice giant
}

# Replace NewPlanet calls
for planet, importance in planet_importance.items():
    # Pattern: NewPlanet("PlanetName", ... , PlanetColors.PlanetName)
    pattern = rf'(NewPlanet\("{planet}",\s+[\d.]+,\s+[\d.]+,\s+[\d.]+,\s+[\d.e+]+,\s+PlanetColors\.{planet})\)'
    replacement = rf'\1, {importance})'
    content = re.sub(pattern, replacement, content)

# Moon importance: Large moons (>0.1 radius) = 60, Small moons (<= 0.1 radius) = 40
# Large moons: Moon (0.136), Io (0.143), Europa (0.122), Ganymede (0.206), Callisto (0.189), Titan (0.201)
large_moons = ['Moon', 'Io', 'Europa', 'Ganymede', 'Callisto', 'Titan', 'Triton', 'Titania', 'Oberon', 'Rhea', 'Iapetus', 'Dione', 'Tethys', 'Enceladus', 'Miranda', 'Ariel', 'Umbriel', 'Charon']

# Helper to add importance to addMoon calls
def replace_add_moon(match):
    full_match = match.group(0)
    moon_name = match.group(1)
    importance = 60 if moon_name in large_moons else 40
    # Add importance before the closing parenthesis
    return full_match.replace(')', f', {importance})')

# Pattern for addMoon("Name", "Parent", ... , Color{...})
pattern = r'addMoon\("([^"]+)",\s+"[^"]+",\s+[\d.]+,\s+[\d.]+,\s+[\d.]+,\s+[\d.]+,\s+[\d.e+]+,\s+Color\{[^}]+\}\)'
content = re.sub(pattern, replace_add_moon, content)

# Write back
with open('../internal/smoke/simulation.go', 'w') as f:
    f.write(content)

print("✓ Updated NewPlanet and addMoon calls with importance values")

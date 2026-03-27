#!/usr/bin/env python3
"""
Fix orbital periods in solar_system.json to use real astronomical values.
Replaces arbitrary orbital_speed with accurate orbital_period in days.
"""

import json
import math

# Real orbital periods in Earth days (from NASA/JPL data)
ORBITAL_PERIODS = {
    # Planets (in days)
    "Mercury": 87.969,
    "Venus": 224.701,
    "Earth": 365.256,
    "Mars": 686.980,
    "Jupiter": 4332.59,
    "Saturn": 10759.22,
    "Uranus": 30688.5,
    "Neptune": 60182.0,
    
    # Dwarf Planets
    "Pluto": 90560.0,
    "Ceres": 1681.0,
    "Eris": 203830.0,
    "Makemake": 112897.0,
    "Haumea": 103410.0,
    
    # Earth's Moon
    "Moon": 27.322,
    
    # Mars Moons
    "Phobos": 0.31891,
    "Deimos": 1.26244,
    
    # Jupiter Moons (Galilean + major)
    "Io": 1.769,
    "Europa": 3.551,
    "Ganymede": 7.155,
    "Callisto": 16.689,
    "Amalthea": 0.498,
    "Thebe": 0.675,
    "Adrastea": 0.298,
    "Metis": 0.295,
    
    # Saturn Moons
    "Titan": 15.945,
    "Rhea": 4.518,
    "Iapetus": 79.330,
    "Dione": 2.737,
    "Tethys": 1.888,
    "Enceladus": 1.370,
    "Mimas": 0.942,
    "Hyperion": 21.277,
    "Phoebe": 550.48,
    "Janus": 0.695,
    "Epimetheus": 0.694,
    "Prometheus": 0.613,
    "Pandora": 0.629,
    "Atlas": 0.602,
    "Pan": 0.575,
    
    # Uranus Moons
    "Titania": 8.706,
    "Oberon": 13.463,
    "Umbriel": 4.144,
    "Ariel": 2.520,
    "Miranda": 1.413,
    
    # Neptune Moons
    "Triton": 5.877,
    "Proteus": 1.122,
    "Nereid": 360.14,
}

def fix_json_file(input_path, output_path):
    """Load JSON, fix orbital periods, save result."""
    print(f"Loading {input_path}...")
    with open(input_path, 'r') as f:
        data = json.load(f)
    
    bodies = data.get("bodies", [])
    fixed_count = 0
    missing = []
    
    for body in bodies:
        name = body.get("name", "")
        orbit = body.get("orbit", {})
        
        # Skip the Sun (no orbit)
        if name == "Sun" or name.startswith("Asteroid-"):
            continue
        
        if name in ORBITAL_PERIODS:
            # Set the correct orbital period
            period_days = ORBITAL_PERIODS[name]
            orbit["orbital_period"] = period_days
            
            # Remove orbital_speed (we'll calculate from period)
            if "orbital_speed" in orbit:
                del orbit["orbital_speed"]
            
            fixed_count += 1
            print(f"✓ {name:20s} → {period_days:10.3f} days")
        else:
            if name and not name.startswith("Asteroid-"):
                missing.append(name)
    
    print(f"\nFixed {fixed_count} bodies")
    
    if missing:
        print(f"\nWarning: No data for these bodies (keeping original values):")
        for name in missing:
            print(f"  - {name}")
    
    # Save the result
    print(f"\nSaving to {output_path}...")
    with open(output_path, 'w') as f:
        json.dump(data, f, indent=2)
    
    print("✓ Done!")

if __name__ == "__main__":
    import sys
    
    input_file = "data/systems/solar_system.json"
    output_file = "data/systems/solar_system.json"
    
    if len(sys.argv) > 1:
        input_file = sys.argv[1]
    if len(sys.argv) > 2:
        output_file = sys.argv[2]
    
    fix_json_file(input_file, output_file)

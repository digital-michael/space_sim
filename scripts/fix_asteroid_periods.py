#!/usr/bin/env python3
"""
Fix asteroid belt orbital periods using Kepler's third law.
Asteroids orbit between 2.0 and 3.3 AU with periods of ~3-6 years.
"""

import json
import math
import random

def calculate_period_from_distance(distance_au):
    """
    Calculate orbital period using Kepler's third law: T² ∝ a³
    For solar orbits: T (years) = a^1.5 (AU)
    
    Returns period in Earth days.
    """
    T_years = distance_au ** 1.5
    T_days = T_years * 365.256
    return T_days

def fix_asteroid_periods(input_path, output_path):
    """Load JSON, fix asteroid orbital periods, save result."""
    print(f"Loading {input_path}...")
    with open(input_path, 'r') as f:
        data = json.load(f)
    
    bodies = data.get("bodies", [])
    asteroid_count = 0
    
    print("\nFixing asteroid orbital periods...")
    for body in bodies:
        name = body.get("name", "")
        orbit = body.get("orbit", {})
        
        # Only process asteroids
        if not name.startswith("Asteroid-"):
            continue
        
        # Get semi-major axis (distance from sun)
        semi_major = orbit.get("semi_major_axis", 0)
        if semi_major > 0:
            # Calculate realistic period
            period_days = calculate_period_from_distance(semi_major)
            orbit["orbital_period"] = period_days
            
            # Remove orbital_speed if present
            if "orbital_speed" in orbit:
                del orbit["orbital_speed"]
            
            asteroid_count += 1
            if asteroid_count <= 5 or asteroid_count % 1000 == 0:
                print(f"  {name:20s} @ {semi_major:5.2f} AU → {period_days:8.1f} days ({period_days/365.256:5.2f} years)")
    
    print(f"\n{'='*60}")
    print(f"Fixed {asteroid_count} asteroids")
    
    # Calculate some statistics
    if asteroid_count > 0:
        print(f"\nAsteroid belt orbital periods:")
        print(f"  2.0 AU → {calculate_period_from_distance(2.0)/365.256:.2f} years")
        print(f"  2.7 AU → {calculate_period_from_distance(2.7)/365.256:.2f} years (mid-belt)")
        print(f"  3.3 AU → {calculate_period_from_distance(3.3)/365.256:.2f} years")
    
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
    
    fix_asteroid_periods(input_file, output_file)

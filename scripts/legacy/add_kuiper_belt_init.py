#!/usr/bin/env python3
"""
Convenience script to add Kuiper Belt initialization to main.go.
This adds the CreateKuiperBelt call alongside asteroid creation.
"""

import re

def add_kuiper_belt_init(main_go_path):
    """Add Kuiper Belt initialization after asteroid loading."""
    
    with open(main_go_path, 'r') as f:
        content = f.read()
    
    # Check if already added
    if 'CreateKuiperBelt' in content:
        print("✓ Kuiper Belt initialization already present in main.go")
        return
    
    # Find the asteroid initialization pattern and add Kuiper Belt after it
    # Look for where CreateAsteroids is called
    pattern = r'(// Create initial asteroid belt[\s\S]*?smoke\.CreateAsteroids\([^)]+\))'
    
    replacement = r'''\1
	
	// Create initial Kuiper Belt (30-50 AU, 164-353 year periods)
	fmt.Println("Generating Kuiper Belt objects...")
	smoke.CreateKuiperBelt(initialState, rng, smoke.KuiperBeltDataset(asteroidDataset))
	fmt.Printf("✓ Generated Kuiper Belt for dataset %d\n", asteroidDataset)'''
    
    new_content = re.sub(pattern, replacement, content)
    
    if new_content == content:
        print("⚠ Could not find asteroid initialization pattern")
        print("Please manually add CreateKuiperBelt call after CreateAsteroids in main.go")
        return
    
    with open(main_go_path, 'w') as f:
        f.write(new_content)
    
    print("✓ Added Kuiper Belt initialization to main.go")

if __name__ == "__main__":
    main_go = "cmd/raylib-smoke/main.go"
    add_kuiper_belt_init(main_go)

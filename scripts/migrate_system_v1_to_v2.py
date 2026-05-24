#!/usr/bin/env python3
"""
migrate_systems.py — Split v1 monolithic system JSON files into v2 per-type
directory layout (F-034). Run from the space_sim project root.

Each data/systems/<name>.json → data/systems/<name>/
  system.json           (manifest, schema_version "2.0")
  stars.json            (if stars present)
  planets.json          (if planets present)
  dwarf_planets.json    (if dwarf planets present)
  moons.json            (if moons present)
  belts.json            (if any features present)

Original .json files are NOT removed.
"""

import json
import os
import sys

SYSTEMS_DIR = "data/systems"
SCHEMA_VERSION = "2.0"

TYPE_TO_FILE = {
    "star": "stars",
    "planet": "planets",
    "dwarf_planet": "dwarf_planets",
    "moon": "moons",
}

FEATURE_TYPES = {"asteroid_belt", "kuiper_belt", "ring_system"}


def migrate_system(src_path: str) -> str:
    """Migrate one monolithic JSON → directory. Returns directory path."""
    with open(src_path, "r") as f:
        cfg = json.load(f)

    # Derive directory name from file stem.
    stem = os.path.splitext(os.path.basename(src_path))[0]
    dest_dir = os.path.join(SYSTEMS_DIR, stem)
    os.makedirs(dest_dir, exist_ok=True)

    bodies_by_type: dict[str, list] = {}
    features = []

    for body in cfg.get("bodies", []):
        t = body.get("type", "planet").lower()
        key = TYPE_TO_FILE.get(t, "planets")
        bodies_by_type.setdefault(key, []).append(body)

    for feat in cfg.get("features", []):
        features.append(feat)

    files_manifest = {}

    # Write per-type body files.
    for key, bodies in bodies_by_type.items():
        filename = f"{key}.json"
        files_manifest[key] = filename
        out = {"schema_version": SCHEMA_VERSION, "bodies": bodies}
        with open(os.path.join(dest_dir, filename), "w") as f:
            json.dump(out, f, indent=2)
            f.write("\n")

    # Write belts.json if any features present.
    if features:
        filename = "belts.json"
        files_manifest["belts"] = filename
        out = {"schema_version": SCHEMA_VERSION, "belts": features}
        with open(os.path.join(dest_dir, filename), "w") as f:
            json.dump(out, f, indent=2)
            f.write("\n")

    # Build system.json manifest.
    manifest = {
        "name": cfg.get("name", stem.replace("_", " ").title()),
        "system_version": cfg.get("version", "1.0"),
        "schema_version": SCHEMA_VERSION,
        "scale_factor": cfg.get("scale_factor", 50),
        "simulation": {
            "default_time_scale": cfg.get("time_scale", cfg.get("seconds_per_second", 1)),
            "nbody_mode": "keplerian",
        },
        "files": files_manifest,
    }

    with open(os.path.join(dest_dir, "system.json"), "w") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")

    types_found = list(files_manifest.keys())
    print(f"  {stem}/ → {', '.join(types_found)}")
    return dest_dir


def main():
    src_files = sorted(
        os.path.join(SYSTEMS_DIR, f)
        for f in os.listdir(SYSTEMS_DIR)
        if f.endswith(".json") and os.path.isfile(os.path.join(SYSTEMS_DIR, f))
    )

    print(f"Migrating {len(src_files)} system(s):")
    for src in src_files:
        migrate_system(src)

    print(f"\nDone. Original .json files preserved.")


if __name__ == "__main__":
    main()

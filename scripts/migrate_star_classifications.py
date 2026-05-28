#!/usr/bin/env python3
"""Phase C migration: update star JSON bodies with new type strings,
spectral_class, and stellar_variant fields. Removes old subtype field
from stellar bodies (rogue/artifact bodies keep their subtype)."""

import json
import sys
from pathlib import Path

ROOT = Path(__file__).parent.parent

# Per-body spectral class overrides (name → spectral_class).
SPECTRAL = {
    "Sol":                    "G",
    "G-Type Star":            "G",
    "Alpha Centauri A":       "G",
    "Alpha Centauri B":       "K",
    "Proxima Centauri":       "M",
    "Barnard's Star":         "M",
    "M-Dwarf Companion":      "M",
    "Demo Red Dwarf":         "M",
    "Demo G-Type Star":       "G",
    "Epsilon Eridani":        "K",
    "Epsilon Indi A":         "K",
    "GJ 1061":                "M",
    "Luyten's Star":          "M",
    "Teegarden's Star":       "M",
    "Wolf 359":               "M",
    "O-Star Companion":       "O",
    "HDE 226868 Analog":      "O",   # O/B supergiant → star_evolved
    "Sirius A":               "A",
    "Test Star":              "G",
    # S-stars orbiting Sgr A* or quasar SMBH
    "S2":      "O", "S4714": "B", "S4711": "B", "S301": "B",
    "S38":     "O", "S1":    "O", "S62":   "O", "S8":   "B",
    "D9a":     "O", "D9b":   "O",
    "S-Star Q1": "O", "S-Star Q2": "O",
}

# Bodies that are evolved stars despite having type="star".
EVOLVED = {
    "HDE 226868 Analog",   # O-supergiant companion to Cygnus X-1
    "Demo Orange Giant",
    "Demo Red Giant",
    "Demo Blue Supergiant",
}

EVOLVED_SPECTRAL = {
    "HDE 226868 Analog":    "O",
    "Demo Orange Giant":    "K",
    "Demo Red Giant":       "M",
    "Demo Blue Supergiant": "B",
}

# Bodies that are T-type brown dwarfs (substellar, T-class).
T_DWARFS = {"Epsilon Indi Ba", "Epsilon Indi Bb", "Luhman 16 B"}
L_DWARFS = {"Luhman 16 A", "Luhman 16A Analog"}


def classify(body: dict) -> dict:
    t = body.get("type", "")
    st = body.get("subtype", "")
    name = body.get("name", "")

    new_type = t
    spectral_class = ""
    stellar_variant = ""

    if t == "blackhole":
        new_type = "stellar_remnant"
        nl = name.lower()
        if "quasar" in nl or "3c 273" in nl:
            stellar_variant = "Quasar"
        elif "sagittarius a" in nl or "sgr a" in nl:
            stellar_variant = "Supermassive Black Hole"
        else:
            stellar_variant = "Black Hole"

    elif t == "star":
        if st == "brown_dwarf" or name in T_DWARFS or name in L_DWARFS:
            new_type = "substellar"
            stellar_variant = "Brown Dwarf"
            if name in T_DWARFS:
                spectral_class = "T"
            else:
                spectral_class = "L"
        elif st == "white_dwarf" or name in {"Sirius B", "White Dwarf"}:
            new_type = "stellar_remnant"
            stellar_variant = "White Dwarf"
        elif st == "neutron_star":
            new_type = "stellar_remnant"
            stellar_variant = "Neutron Star"
        elif st == "pulsar":
            new_type = "stellar_remnant"
            stellar_variant = "Pulsar"
        elif st == "magnetar":
            new_type = "stellar_remnant"
            stellar_variant = "Magnetar"
        elif name in EVOLVED:
            new_type = "star_evolved"
            spectral_class = EVOLVED_SPECTRAL.get(name, "")
        else:
            new_type = "star_main_sequence"
            spectral_class = SPECTRAL.get(name, "")

    # Apply updates
    body["type"] = new_type
    if spectral_class:
        body["spectral_class"] = spectral_class
    if stellar_variant:
        body["stellar_variant"] = stellar_variant
    # Remove old subtype from stellar bodies; keep for rogue/artifact
    if "subtype" in body and t in ("star", "blackhole"):
        del body["subtype"]

    return body


def migrate_file(path: Path) -> bool:
    with open(path) as f:
        data = json.load(f)

    changed = False

    # New format: {"schema_version": "2.0", "bodies": [...]}
    if "bodies" in data:
        for i, body in enumerate(data["bodies"]):
            original = json.dumps(body)
            classify(body)
            if json.dumps(body) != original:
                changed = True
    # Old template format: {"stars": {"key": {...}}}
    elif "stars" in data:
        for key, body in data["stars"].items():
            original = json.dumps(body)
            classify(body)
            if json.dumps(body) != original:
                changed = True

    if changed:
        with open(path, "w") as f:
            json.dump(data, f, indent=2)
            f.write("\n")
        print(f"  updated: {path.relative_to(ROOT)}")
    return changed


def main():
    files = sorted(ROOT.glob("data/**/stars.json"))
    updated = 0
    for f in files:
        if migrate_file(f):
            updated += 1
    print(f"\n{updated}/{len(files)} files updated.")


if __name__ == "__main__":
    main()

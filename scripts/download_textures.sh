#!/bin/bash
# Download NASA/JPL texture maps for solar system bodies
# Textures sourced from Solar System Scope (solarsystemscope.com)
# and NASA's CGI Moon Kit

set -e

TEXTURE_DIR="data/assets/textures"
mkdir -p "$TEXTURE_DIR"

echo "Downloading solar system textures..."
echo "This will download approximately 500MB of texture data."
echo ""

# Solar System Scope textures (Creative Commons)
BASE_URL="https://www.solarsystemscope.com/textures/download"

# Sun (8K)
echo "Downloading Sun texture (8K)..."
curl -L -o "$TEXTURE_DIR/sun_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_sun.jpg" || echo "Sun texture failed"

# Mercury (4K)
echo "Downloading Mercury texture (4K)..."
curl -L -o "$TEXTURE_DIR/mercury_4k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_mercury.jpg" || echo "Mercury texture failed"

# Venus (4K surface + atmosphere)
echo "Downloading Venus textures (4K)..."
curl -L -o "$TEXTURE_DIR/venus_surface_4k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_venus_surface.jpg" || echo "Venus surface failed"
curl -L -o "$TEXTURE_DIR/venus_atmosphere_4k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_venus_atmosphere.jpg" || echo "Venus atmosphere failed"

# Earth (8K color + normal + specular)
echo "Downloading Earth textures (8K)..."
curl -L -o "$TEXTURE_DIR/earth_daymap_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_earth_daymap.jpg" || echo "Earth daymap failed"
curl -L -o "$TEXTURE_DIR/earth_nightmap_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_earth_nightmap.jpg" || echo "Earth nightmap failed"
curl -L -o "$TEXTURE_DIR/earth_normal_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_earth_normal_map.jpg" || echo "Earth normal failed"
curl -L -o "$TEXTURE_DIR/earth_specular_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_earth_specular_map.jpg" || echo "Earth specular failed"
curl -L -o "$TEXTURE_DIR/earth_clouds_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_earth_clouds.jpg" || echo "Earth clouds failed"

# Moon (8K color + normal)
echo "Downloading Moon textures (8K)..."
curl -L -o "$TEXTURE_DIR/moon_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_moon.jpg" || echo "Moon texture failed"

# Mars (8K)
echo "Downloading Mars texture (8K)..."
curl -L -o "$TEXTURE_DIR/mars_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_mars.jpg" || echo "Mars texture failed"

# Jupiter (4K)
echo "Downloading Jupiter texture (4K)..."
curl -L -o "$TEXTURE_DIR/jupiter_4k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_jupiter.jpg" || echo "Jupiter texture failed"

# Saturn (4K + rings)
echo "Downloading Saturn textures (4K)..."
curl -L -o "$TEXTURE_DIR/saturn_4k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_saturn.jpg" || echo "Saturn texture failed"
curl -L -o "$TEXTURE_DIR/saturn_ring_alpha.png" \
  "https://www.solarsystemscope.com/textures/download/2k_saturn_ring_alpha.png" || echo "Saturn ring failed"

# Uranus (2K)
echo "Downloading Uranus texture (2K)..."
curl -L -o "$TEXTURE_DIR/uranus_2k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_uranus.jpg" || echo "Uranus texture failed"

# Neptune (2K)
echo "Downloading Neptune texture (2K)..."
curl -L -o "$TEXTURE_DIR/neptune_2k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_neptune.jpg" || echo "Neptune texture failed"

# Pluto (doesn't have high-res on Solar System Scope, use gray placeholder)
echo "Downloading Pluto texture (1K)..."
curl -L -o "$TEXTURE_DIR/pluto_1k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_ceres_fictional.jpg" || echo "Pluto texture failed"

# Major moons - use generic rocky/icy textures
echo "Downloading generic moon textures..."
curl -L -o "$TEXTURE_DIR/generic_moon_2k.jpg" \
  "https://www.solarsystemscope.com/textures/download/2k_moon.jpg" || echo "Generic moon failed"

# Background stars
echo "Downloading starfield background (8K)..."
curl -L -o "$TEXTURE_DIR/starfield_8k.jpg" \
  "https://www.solarsystemscope.com/textures/download/8k_stars_milky_way.jpg" || echo "Starfield failed"

echo ""
echo "✓ Texture download complete!"
echo "Textures saved to: $TEXTURE_DIR"
echo ""
echo "Note: These textures are from Solar System Scope (CC BY 4.0 license)"
echo "For production use, verify licensing and consider NASA's official texture archive:"
echo "https://nasa3d.arc.nasa.gov/images"

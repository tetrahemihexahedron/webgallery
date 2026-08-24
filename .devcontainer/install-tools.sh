#!/usr/bin/env bash
set -euo pipefail

# Use Debian packages instead of downloading and building upstream releases.
# libvips-tools provides vipsthumbnail; libheif-plugin-aomenc is needed for
# AVIF output because --no-install-recommends skips libheif's recommended
# encoder plugins.
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  just \
  libimage-exiftool-perl \
  libvips-tools \
  libheif-plugin-aomenc

npm install -g --ignore-scripts @earendil-works/pi-coding-agent

command -v exiftool >/dev/null
command -v vips >/dev/null
command -v vipsthumbnail >/dev/null

# --------
# Smoke test vipsthumbnail
CHECK_DIR="$(mktemp -d)"
trap 'rm -rf "$CHECK_DIR"' EXIT

# Create a 1px test image
printf 'P3\n1 1\n255\n0 0 0\n' > "$CHECK_DIR/check.ppm"

vipsthumbnail "$CHECK_DIR/check.ppm" \
  --size "1x>" \
  --output "$CHECK_DIR/check.jpg[Q=75,keep=none]" \
  >/dev/null
vipsthumbnail "$CHECK_DIR/check.ppm" \
  --size "1x>" \
  --output "$CHECK_DIR/check.avif[Q=75,effort=6,keep=none]" \
  >/dev/null

test -s "$CHECK_DIR/check.jpg"
test -s "$CHECK_DIR/check.avif"

# --------
echo "Installed exiftool $(exiftool -ver)"
echo "Installed $(vips --version)"

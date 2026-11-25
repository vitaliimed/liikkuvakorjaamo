#!/bin/sh
set -eu
out=static/img/landing-v2
mkdir -p "$out"
for width in 480 640 960 1536; do
  vips thumbnail static/img/mobile-mechanic-v2.png "$out/hero-$width.webp[Q=72,keep=none]" "$width" --size down
done

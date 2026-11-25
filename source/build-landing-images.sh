#!/bin/sh
set -eu
out=static/img/landing-v2
mkdir -p "$out"
for width in 480 640 960 1536; do
  vips thumbnail static/img/mobile-mechanic-v2.png "$out/hero-$width.webp[Q=72,keep=none]" "$width" --size down
done
# Keep the original cascade order and original files for existing callers.
cat static/css/pro.css static/css/responsive.css static/css/brand.css \
    static/css/hero-form-ux.css static/css/footer-form.css static/css/landing-v2.css \
    > static/css/landing-release.css

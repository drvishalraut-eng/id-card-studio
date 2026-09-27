#!/usr/bin/env bash
# Fetches the pinned third-party libraries and fonts into web/vendor/,
# verifying each download's sha256 before it's kept. Run this after cloning
# if web/vendor/ is ever wiped, or to deliberately bump a pinned version
# (update both the URL and the checksum below together).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vendor_dir="$root/web/vendor"
fonts_dir="$vendor_dir/fonts"
mkdir -p "$vendor_dir" "$fonts_dir"

fetch() {
  local url="$1" dest="$2" want_sha="$3"
  echo "Fetching $(basename "$dest")..."
  curl -fsSL -o "$dest" "$url"
  local got_sha
  got_sha="$(sha256sum "$dest" | cut -d' ' -f1)"
  if [ "$got_sha" != "$want_sha" ]; then
    echo "Checksum mismatch for $dest" >&2
    echo "  expected: $want_sha" >&2
    echo "  got:      $got_sha" >&2
    rm -f "$dest"
    exit 1
  fi
}

# html-to-image 1.11.13 (MIT)
fetch "https://cdn.jsdelivr.net/npm/html-to-image@1.11.13/dist/html-to-image.min.js" \
  "$vendor_dir/html-to-image.min.js" \
  "40b86f338648f62b30d659dd20ab8eae5bd8326d1e988766ed6649ae0751c47a"

# jsPDF 4.2.1 (MIT)
fetch "https://cdn.jsdelivr.net/npm/jspdf@4.2.1/dist/jspdf.umd.min.js" \
  "$vendor_dir/jspdf.umd.min.js" \
  "e6551fcdc32f09d6853b2c5126d18d01d9447e0da618a41a11ebeee0f6c20d54"

# JSZip 3.10.2 (MIT)
fetch "https://cdn.jsdelivr.net/npm/jszip@3.10.2/dist/jszip.min.js" \
  "$vendor_dir/jszip.min.js" \
  "7f839b2d4688b845c105ebf5d2f9803075f91ea0fe72bdaac176c3a04dd3d2c1"

# SheetJS xlsx 0.18.5 (Apache-2.0)
fetch "https://cdn.jsdelivr.net/npm/xlsx@0.18.5/dist/xlsx.full.min.js" \
  "$vendor_dir/xlsx.full.min.js" \
  "c9506197caf809a075b6dee1da0d36fb19da7158ffe8a88e7b0c96c5d8623c99"

# Montserrat (OFL-1.1), variable font, Google Fonts v31, latin subset
fetch "https://fonts.gstatic.com/s/montserrat/v31/JTUSjIg1_i6t8kCHKm459Wlhyw.woff2" \
  "$fonts_dir/montserrat.woff2" \
  "06b16db7a969135d48d38c49183be7fb88d4452e2a3011957c7851941f4e4879"

# Josefin Sans (OFL-1.1), variable font, Google Fonts v34, latin subset
fetch "https://fonts.gstatic.com/s/josefinsans/v34/Qw3aZQNVED7rKGKxtqIqX5EUDXx4.woff2" \
  "$fonts_dir/josefin-sans.woff2" \
  "21efe1559a026f9ad7f880772917faa3a621186688e8e79e0568e4308c0bafa9"

echo "Vendored files are up to date in $vendor_dir"

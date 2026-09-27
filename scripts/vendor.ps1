# Fetches the pinned third-party libraries and fonts into web/vendor/,
# verifying each download's sha256 before it's kept. Run this after cloning
# if web/vendor/ is ever wiped, or to deliberately bump a pinned version
# (update both the URL and the checksum below together).
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$vendorDir = Join-Path $root "web\vendor"
$fontsDir = Join-Path $vendorDir "fonts"
New-Item -ItemType Directory -Force -Path $vendorDir, $fontsDir | Out-Null

function Fetch($Url, $Dest, $WantSha) {
    Write-Host "Fetching $(Split-Path -Leaf $Dest)..."
    Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing
    $gotSha = (Get-FileHash -Path $Dest -Algorithm SHA256).Hash.ToLower()
    if ($gotSha -ne $WantSha) {
        Write-Error "Checksum mismatch for $Dest`n  expected: $WantSha`n  got:      $gotSha"
        Remove-Item -Force $Dest
        exit 1
    }
}

# html-to-image 1.11.13 (MIT)
Fetch "https://cdn.jsdelivr.net/npm/html-to-image@1.11.13/dist/html-to-image.min.js" `
    (Join-Path $vendorDir "html-to-image.min.js") `
    "40b86f338648f62b30d659dd20ab8eae5bd8326d1e988766ed6649ae0751c47a"

# jsPDF 4.2.1 (MIT)
Fetch "https://cdn.jsdelivr.net/npm/jspdf@4.2.1/dist/jspdf.umd.min.js" `
    (Join-Path $vendorDir "jspdf.umd.min.js") `
    "e6551fcdc32f09d6853b2c5126d18d01d9447e0da618a41a11ebeee0f6c20d54"

# JSZip 3.10.2 (MIT)
Fetch "https://cdn.jsdelivr.net/npm/jszip@3.10.2/dist/jszip.min.js" `
    (Join-Path $vendorDir "jszip.min.js") `
    "7f839b2d4688b845c105ebf5d2f9803075f91ea0fe72bdaac176c3a04dd3d2c1"

# SheetJS xlsx 0.18.5 (Apache-2.0)
Fetch "https://cdn.jsdelivr.net/npm/xlsx@0.18.5/dist/xlsx.full.min.js" `
    (Join-Path $vendorDir "xlsx.full.min.js") `
    "c9506197caf809a075b6dee1da0d36fb19da7158ffe8a88e7b0c96c5d8623c99"

# Montserrat (OFL-1.1), variable font, Google Fonts v31, latin subset
Fetch "https://fonts.gstatic.com/s/montserrat/v31/JTUSjIg1_i6t8kCHKm459Wlhyw.woff2" `
    (Join-Path $fontsDir "montserrat.woff2") `
    "06b16db7a969135d48d38c49183be7fb88d4452e2a3011957c7851941f4e4879"

# Josefin Sans (OFL-1.1), variable font, Google Fonts v34, latin subset
Fetch "https://fonts.gstatic.com/s/josefinsans/v34/Qw3aZQNVED7rKGKxtqIqX5EUDXx4.woff2" `
    (Join-Path $fontsDir "josefin-sans.woff2") `
    "21efe1559a026f9ad7f880772917faa3a621186688e8e79e0568e4308c0bafa9"

Write-Host "Vendored files are up to date in $vendorDir"

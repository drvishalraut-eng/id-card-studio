# Runs the test suite, then cross-compiles idcard for every supported
# platform into dist/<os>-<arch>/idcard(.exe) - one self-contained,
# ready-to-copy folder per platform.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "Running go vet..."
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Running go test..."
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$dist = Join-Path $root "dist"
if (Test-Path $dist) { Remove-Item -Recurse -Force $dist }

function Build($GoOS, $GoArch, $Out) {
    Write-Host "Building $GoOS/$GoArch..."
    $dir = Join-Path $dist "$GoOS-$GoArch"
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $env:GOOS = $GoOS
    $env:GOARCH = $GoArch
    $env:CGO_ENABLED = "0"
    go build -o (Join-Path $dir $Out) .
    if ($LASTEXITCODE -ne 0) { throw "build failed for $GoOS/$GoArch" }
}

Build "windows" "amd64" "idcard.exe"
Build "darwin"  "arm64" "idcard"
Build "darwin"  "amd64" "idcard"
Build "linux"   "amd64" "idcard"

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Built binaries:"
Get-ChildItem -Recurse -File $dist | ForEach-Object {
    Write-Host ("{0,10}  {1}" -f $_.Length, $_.FullName)
}

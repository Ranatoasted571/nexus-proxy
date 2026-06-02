# NEXUS installer for Windows (PowerShell).
#
#   irm https://raw.githubusercontent.com/lynuxis2026-pixel/nexus-proxy/main/install.ps1 | iex
#
# Override the source repo or version:
#   $env:NEXUS_REPO="you/nexus"; $env:NEXUS_VERSION="v0.2.0"; .\install.ps1
$ErrorActionPreference = "Stop"

$Repo    = if ($env:NEXUS_REPO)    { $env:NEXUS_REPO }    else { "lynuxis2026-pixel/nexus-proxy" }
$Version = if ($env:NEXUS_VERSION) { $env:NEXUS_VERSION } else { "latest" }

$arch  = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$asset = "nexus_windows_$arch.zip"
$url   = if ($Version -eq "latest") {
  "https://github.com/$Repo/releases/latest/download/$asset"
} else {
  "https://github.com/$Repo/releases/download/$Version/$asset"
}

Write-Host "-> Downloading $asset ($Repo @ $Version)"
$tmp = Join-Path $env:TEMP ("nexus-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force $tmp | Out-Null
$zip = Join-Path $tmp $asset
Invoke-WebRequest -Uri $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $tmp -Force

$dest = Join-Path $env:LOCALAPPDATA "Programs\nexus"
New-Item -ItemType Directory -Force $dest | Out-Null
Copy-Item (Join-Path $tmp "nexus.exe") (Join-Path $dest "nexus.exe") -Force
Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue

# Add to the user PATH if not already present.
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dest*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$dest", "User")
  Write-Host "Added $dest to your user PATH (restart your terminal to pick it up)."
}

Write-Host "Installed nexus to $dest\nexus.exe"
Write-Host ""
Write-Host "Get started:"
Write-Host "  nexus start"
Write-Host '  $env:ANTHROPIC_BASE_URL="http://localhost:3000"'
Write-Host '  $env:ANTHROPIC_API_KEY="nexus-local"'
Write-Host "  claude"

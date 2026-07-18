<#
Solo Agency — Local Collector setup (Windows / PowerShell).

PowerShell counterpart of setup_collector.sh for machines that cannot run .sh
(Windows without Git Bash/WSL). Same behavior: download the bridge bundle + SHA256SUMS,
verify the checksum (matched by BASENAME, tolerant of bare / *name / full-path formats),
extract the binary for THIS machine, and print the launch command. It never starts the
bridge and is safe to re-run (idempotent).

  Windows:  powershell -ExecutionPolicy Bypass -File setup_collector.ps1
  PS7:      pwsh -File setup_collector.ps1

Run it from your agency root (the folder that contains daily-content-pipeline/).
#>

$ErrorActionPreference = 'Stop'

$Version = if ($env:SOLO_AGENCY_COLLECTOR_VERSION) { $env:SOLO_AGENCY_COLLECTOR_VERSION } else { '0.1.0' }
$BaseUrl = if ($env:SOLO_AGENCY_DIST_BASE) { $env:SOLO_AGENCY_DIST_BASE } else { 'https://raw.githubusercontent.com/soloagency/solo-agency/dist' }
$Bundle  = "collector-bridge-binaries-$Version.zip"
$Sums    = 'SHA256SUMS'

function Say  ($m) { Write-Host "`n$m" -ForegroundColor White }
function Info ($m) { Write-Host "  $m" }
function Ok   ($m) { Write-Host "  [OK] $m" -ForegroundColor Green }
function Warn ($m) { Write-Host "  [!] $m"  -ForegroundColor Yellow }
function Fail ($m, $h) { Write-Host "`n[X] $m" -ForegroundColor Red; if ($h) { Write-Host "  -> $h" }; exit 1 }

# --- runtime folders ---------------------------------------------------------
$Root    = (Get-Location).Path
$Runtime = Join-Path $Root 'solo-agency-local-collector'
$DL      = Join-Path $Runtime 'downloads'
$Bin     = Join-Path $Runtime 'bin'
New-Item -ItemType Directory -Force -Path $DL, $Bin | Out-Null

# --- platform detection ------------------------------------------------------
# $IsWindows/$IsMacOS/$IsLinux exist only in PowerShell 6+. Windows PowerShell 5.1
# (default on Windows) has no $IsWindows, so treat "< v6" as Windows.
$onWindows = ($PSVersionTable.PSVersion.Major -lt 6) -or ($IsWindows -eq $true)
if     ($onWindows)          { $O = 'windows'; $Ext = '.exe' }
elseif ($IsMacOS -eq $true)  { $O = 'darwin';  $Ext = '' }
elseif ($IsLinux -eq $true)  { $O = 'linux';   $Ext = '' }
else   { Fail "Could not determine your OS." "On macOS/Linux use setup_collector.sh instead." }

$archRaw = if ($env:PROCESSOR_ARCHITECTURE) { $env:PROCESSOR_ARCHITECTURE } elseif (Get-Command uname -ErrorAction SilentlyContinue) { (uname -m) } else { 'AMD64' }
switch -Regex ($archRaw) {
  'ARM64|aarch64' { $A = 'arm64' }
  'AMD64|x86_64'  { $A = 'amd64' }
  default         { $A = 'amd64' }
}

$TargetBin = "collector-bridge-$O-$A$Ext"
if ("$O-$A" -notin @('darwin-arm64', 'darwin-amd64', 'linux-amd64', 'windows-amd64')) {
  if ($O -eq 'windows') { $A = 'amd64'; $TargetBin = 'collector-bridge-windows-amd64.exe' }  # win-arm64 runs the amd64 build under emulation
  else { Fail "No prebuilt bridge for $O/$A." "Ask a maintainer to build one for your platform." }
}

# --- helpers -----------------------------------------------------------------
function Get-File ($name, $dest) {
  $url = "$BaseUrl/$name"
  for ($i = 1; $i -le 3; $i++) {
    try {
      Invoke-WebRequest -Uri $url -OutFile "$dest.part" -UseBasicParsing -TimeoutSec 60
      Move-Item -Force "$dest.part" $dest
      return
    } catch {
      Warn "Download failed for $name (attempt $i/3) - retrying in 2s..."
      Remove-Item -Force "$dest.part" -ErrorAction SilentlyContinue
      Start-Sleep -Seconds 2
    }
  }
  Fail "Could not download $name" "Check your internet connection, then run this script again. Nothing was changed."
}

function Get-ExpectedSum ($file) {
  # Match by BASENAME so it works whether SHA256SUMS lists a bare name, *name, or full path.
  $hit = Select-String -Path (Join-Path $DL $Sums) -SimpleMatch $file -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($hit) { (($hit.Line -split '\s+') | Where-Object { $_ })[0] } else { $null }
}
function Get-Sha256 ($path) { (Get-FileHash -Algorithm SHA256 -Path $path).Hash.ToLower() }

# --- run ---------------------------------------------------------------------
Say "Solo Agency Local Collector setup (v$Version)"
Info "Machine : $O/$A  ->  bridge binary: $TargetBin"
Info "Install : $Runtime"

Say "1/4  Fetching checksums"
Get-File $Sums (Join-Path $DL $Sums)
Ok "got $Sums"

Say "2/4  Fetching the bridge bundle"
$want = Get-ExpectedSum $Bundle
if (-not $want) { Fail "Checksum for $Bundle not found in $Sums." "The published checksum file looks out of date. Re-run in a minute; if it persists, tell your setup agent." }
$want = $want.ToLower()
$bundlePath = Join-Path $DL $Bundle
if ((Test-Path $bundlePath) -and ((Get-Sha256 $bundlePath) -eq $want)) {
  Ok "already downloaded and up to date (skipped the download)"
} else {
  Get-File $Bundle $bundlePath
  Ok "downloaded $Bundle"
}

Say "3/4  Verifying checksum"
if ((Get-Sha256 $bundlePath) -ne $want) {
  Remove-Item -Force $bundlePath
  Fail "Checksum MISMATCH for $Bundle (download corrupted or tampered)." "Deleted the bad file - run this script again to re-download. Do NOT use a file that fails this check."
}
Ok "checksum verified"

Say "4/4  Extracting your binary"
Expand-Archive -Path $bundlePath -DestinationPath $Bin -Force
$binPath = Join-Path $Bin $TargetBin
if (-not (Test-Path $binPath)) { Fail "The bundle did not contain $TargetBin." "It may be built for a different version. Tell your setup agent." }
Ok "installed: $binPath"

Say "Setup complete. Start the bridge yourself with (this script does NOT start it):"
Write-Host ""
Write-Host "  & `"$binPath`" --host 127.0.0.1 --port 17321 ``"
Write-Host "      --config-file daily-content-pipeline/collector/collector_config.json ``"
Write-Host "      --output-dir daily-content-pipeline/collector/inbox --persistent"
Write-Host ""
Info "One-time: install the Chrome extension via Developer Mode (see AGENT_RUNBOOK.md)."

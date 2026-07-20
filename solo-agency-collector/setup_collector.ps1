<#
Solo Agency — Local Collector setup (Windows / PowerShell).

PowerShell counterpart of setup_collector.sh for machines that cannot run .sh
(Windows without Git Bash/WSL). One command does the whole setup and is safe to re-run:
download the bundle + SHA256SUMS, verify the checksum (matched by BASENAME, tolerant of
bare / *name / full-path formats), extract the binary for THIS machine, STOP any bridge
already on the port (it never kills a non-collector process), and START the newest bridge
in the BACKGROUND so you can close the window. It never fails on "address already in use".

The bridge is registered as a logon Scheduled Task so it starts automatically after a
reboot (and restarts itself after a crash). Set SOLO_AGENCY_NO_AUTOSTART=1 to skip the
registration and start a plain background process instead.

  Windows:  powershell -ExecutionPolicy Bypass -File setup_collector.ps1
  PS7:      pwsh -File setup_collector.ps1

The agency root is resolved from the script's own location first (the install
this copy belongs to), so invoking it by absolute path always targets that
install; the terminal's current folder is only a fallback. Override with
SOLO_AGENCY_ROOT when needed.
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

# --- resolve agency root (script location beats the terminal's cwd) ----------
# Order: SOLO_AGENCY_ROOT env override -> the script's own location -> cwd.
# Invoking an install's script by absolute path must target THAT install even
# when the terminal is standing in another workspace with its own pipeline.
$Root = if ($env:SOLO_AGENCY_ROOT) {
  (Resolve-Path $env:SOLO_AGENCY_ROOT).Path
} elseif ($PSScriptRoot -and (Test-Path (Join-Path (Split-Path $PSScriptRoot -Parent) 'daily-content-pipeline'))) {
  Split-Path $PSScriptRoot -Parent
} elseif ($PSScriptRoot -and (Test-Path (Join-Path $PSScriptRoot 'daily-content-pipeline'))) {
  $PSScriptRoot
} else {
  (Get-Location).Path
}
$CwdPath = (Get-Location).Path
if ((Test-Path (Join-Path $CwdPath 'daily-content-pipeline')) -and ($Root -ne $CwdPath)) {
  Warn "Terminal is standing in a DIFFERENT workspace: $CwdPath"
  Warn "Using the install this script belongs to: $Root (set SOLO_AGENCY_ROOT to override)"
}

# --- runtime folders ---------------------------------------------------------
$Runtime = Join-Path $Root 'solo-agency-local-collector'
$DL      = Join-Path $Runtime 'downloads'
$Bin     = Join-Path $Runtime 'bin'
New-Item -ItemType Directory -Force -Path $DL, $Bin | Out-Null
$Port       = if ($env:SOLO_AGENCY_BRIDGE_PORT) { [int]$env:SOLO_AGENCY_BRIDGE_PORT } else { 17321 }
$ConfigFile = Join-Path $Root 'daily-content-pipeline/collector/collector_config.json'
$OutputDir  = Join-Path $Root 'daily-content-pipeline/collector/inbox'
$PidFile    = Join-Path $Runtime 'collector.pid'
$LogFile    = Join-Path $Runtime 'collector.log'
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Per-install autostart identity: two installs on one machine each get their own task.
$sha = [System.Security.Cryptography.SHA256]::Create()
$InstHash = ([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($Root))) -replace '-','').Substring(0,8).ToLower()
$TaskName = "SoloAgencyCollector-$InstHash"
$AutostartState = Join-Path $Runtime 'autostart.json'

# Canonical, workspace-readable evidence of the autostart outcome. Sandboxed
# agents cannot run Get-ScheduledTask, but they CAN read this file.
function Record-Autostart ($mode, $label, $reason) {
  try {
    @{ mode = $mode; label = $label; port = $Port; root = $Root
       registered_at = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
       reason = $reason } | ConvertTo-Json -Compress | Out-File -FilePath $AutostartState -Encoding ascii
  } catch { }
}

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

Say "1/6  Fetching checksums"
Get-File $Sums (Join-Path $DL $Sums)
Ok "got $Sums"

Say "2/6  Fetching the bridge bundle"
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

Say "3/6  Verifying checksum"
if ((Get-Sha256 $bundlePath) -ne $want) {
  Remove-Item -Force $bundlePath
  Fail "Checksum MISMATCH for $Bundle (download corrupted or tampered)." "Deleted the bad file - run this script again to re-download. Do NOT use a file that fails this check."
}
Ok "checksum verified"

Say "4/6  Extracting your binary"
Expand-Archive -Path $bundlePath -DestinationPath $Bin -Force
$binPath = Join-Path $Bin $TargetBin
if (-not (Test-Path $binPath)) { Fail "The bundle did not contain $TargetBin." "It may be built for a different version. Tell your setup agent." }
Ok "installed: $binPath"

if ($env:SOLO_AGENCY_SETUP_NO_START -eq '1') {
  Say "Install complete (SOLO_AGENCY_SETUP_NO_START=1 -> not stopping/starting the bridge)."
  Info "To run it: powershell -ExecutionPolicy Bypass -File setup_collector.ps1"
  exit 0
}

Say "5/6  Stopping any bridge already on port $Port"
# Detach the autostart task first so it cannot respawn the old binary mid-upgrade.
try { Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue } catch { }
try { Invoke-WebRequest -UseBasicParsing -Method Post -Uri "http://127.0.0.1:$Port/shutdown" -TimeoutSec 3 | Out-Null } catch { }
if (Test-Path $PidFile) {
  $oldPid = (Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
  if ($oldPid) { try { Stop-Process -Id ([int]$oldPid) -Force -ErrorAction SilentlyContinue } catch { } }
}
# Kill the port owner ONLY if it is a collector-bridge — never an unknown process.
if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
  foreach ($c in (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)) {
    $op = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
    if ($op -and ($op.ProcessName -like '*collector-bridge*')) { try { Stop-Process -Id $op.Id -Force -ErrorAction SilentlyContinue } catch { } }
    elseif ($op) { Fail "Port $Port is held by a NON-collector process (PID $($op.Id)): $($op.ProcessName)" "This setup will not kill an unknown process. Stop it yourself, then re-run." }
  }
}
Start-Sleep -Seconds 1
Ok "port $Port is free"

Say "6/6  Starting the newest bridge (background, persistent, autostart at logon)"
if (-not (Test-Path $ConfigFile)) { Warn "config not found at $ConfigFile - starting anyway; if the bridge exits, create the config and re-run." }

function Wait-Healthy {
  for ($i = 0; $i -lt 20; $i++) {
    try {
      Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/status" -TimeoutSec 2 | Out-Null
      return $true
    } catch { Start-Sleep -Seconds 1 }
  }
  return $false
}

function Start-PlainBackground {
  $argList = @('--host','127.0.0.1','--port',"$Port",'--config-file',$ConfigFile,'--output-dir',$OutputDir,'--persistent')
  $proc = Start-Process -FilePath $binPath -ArgumentList $argList -RedirectStandardOutput $LogFile -RedirectStandardError "$LogFile.err" -WindowStyle Hidden -PassThru
  $proc.Id | Out-File -FilePath $PidFile -Encoding ascii
  Start-Sleep -Seconds 2
  if ($proc.HasExited) {
    Fail "The bridge exited right after starting." ("Last lines of ${LogFile}:`n" + ((Get-Content $LogFile -Tail 15 -ErrorAction SilentlyContinue) -join "`n"))
  }
  Ok "bridge running (pid $($proc.Id))"
}

$Autostart = 'none'
if ($env:SOLO_AGENCY_NO_AUTOSTART -eq '1') {
  Info "SOLO_AGENCY_NO_AUTOSTART=1 - plain background start (no logon registration)."
  Record-Autostart 'none' '' 'opt_out_env'
  Start-PlainBackground
} elseif ($onWindows -and (Get-Command Register-ScheduledTask -ErrorAction SilentlyContinue)) {
  # Windows: a logon Scheduled Task. Starts at every logon, restarts on crash,
  # no time limit. Wrapped in cmd /c so the bridge's stdout/stderr still lands
  # in collector.log (Scheduled Tasks have no output redirection of their own).
  try {
    $inner = "`"$binPath`" --host 127.0.0.1 --port $Port --config-file `"$ConfigFile`" --output-dir `"$OutputDir`" --persistent >> `"$LogFile`" 2>&1"
    $action   = New-ScheduledTaskAction -Execute 'cmd.exe' -Argument "/c `"$inner`""
    $trigger  = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
      -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) `
      -MultipleInstances IgnoreNew
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
    if (Wait-Healthy) {
      $Autostart = "Scheduled Task '$TaskName'"
      # PID for the stop hint (best effort)
      if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
        $owner = (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1).OwningProcess
        if ($owner) { $owner | Out-File -FilePath $PidFile -Encoding ascii }
      }
      Record-Autostart 'scheduled_task' $TaskName 'registered'
      Ok "bridge running as a Scheduled Task - starts automatically at logon, restarts on crash"
    } else {
      Warn "Scheduled Task did not become healthy - falling back to a plain background start."
      try { Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue } catch { }
      Record-Autostart 'none' '' 'task_not_healthy'
      Start-PlainBackground
    }
  } catch {
    Warn "Scheduled Task registration failed ($($_.Exception.Message)) - falling back to a plain background start."
    Record-Autostart 'none' '' 'task_registration_failed'
    Start-PlainBackground
  }
} else {
  Record-Autostart 'none' '' 'no_supervisor_available'
  Start-PlainBackground
}

Say "Done - the collector is running in the background. You can close this window."
Info "Port      : 127.0.0.1:$Port"
Info "Status    : curl http://127.0.0.1:$Port/status"
Info "Logs      : $LogFile"
if ($Autostart -ne 'none') {
  Info "Autostart : $Autostart - survives reboots; re-run this script after updates."
  Info "Stop      : Stop-ScheduledTask -TaskName $TaskName   (disable: Unregister-ScheduledTask -TaskName $TaskName)"
} else {
  Info "Autostart : OFF - after a reboot re-run this script."
}
Info "One-time: install the Chrome extension via Developer Mode (see AGENT_RUNBOOK.md)."

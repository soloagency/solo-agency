<#
Solo Agency — Local Collector uninstall (Windows / PowerShell).

PowerShell counterpart of uninstall_collector.sh. Reverses setup_collector.ps1: stops the
Scheduled Task it registered, kills the collector-bridge process, releases the one-key-one-install
entitlement seat (best effort), and deletes the install's runtime state — leaving the machine as
if Solo Agency had never been installed, MODULO a short list of things this script cannot safely
touch on its own (an AI desktop app's own Scheduled panel, another agent runtime's automations,
the unpacked Chrome extension entry) — those are printed, never silently skipped.

  Usage:  powershell -ExecutionPolicy Bypass -File uninstall_collector.ps1 [-DryRun] [-Yes]
                                             [-Root PATH] [-KeepSource] [-KeepRootDir]
                                             [-Port N] [-OpenBrowser]
  PS7:    pwsh -File uninstall_collector.ps1 ...

Safe to run with no arguments: it discovers every Solo Agency install on this machine (script
location, Scheduled Task registrations matching SoloAgencyCollector-*, the process on the
collector port) and asks ONE y/N confirmation before touching anything. Pass -DryRun to see the
plan and change nothing, or -Yes to skip the prompt (this is what an agent should pass after the
human has explicitly confirmed).

It NEVER deletes anything outside a validated Solo Agency install root, and it NEVER kills a
process whose command does not contain "collector-bridge".
#>

param(
  [string[]]$Root = @(),
  [switch]$DryRun,
  [switch]$Yes,
  [switch]$KeepSource,
  [switch]$KeepRootDir,
  [int]$Port = $(if ($env:SOLO_AGENCY_BRIDGE_PORT) { [int]$env:SOLO_AGENCY_BRIDGE_PORT } else { 17321 }),
  [switch]$OpenBrowser,
  [switch]$Help
)

$ErrorActionPreference = 'Continue'

function Say  ($m) { Write-Host "`n$m" -ForegroundColor White }
function Info ($m) { Write-Host "  $m" }
function Ok   ($m) { Write-Host "  [OK] $m" -ForegroundColor Green }
function Warn ($m) { Write-Host "  [!] $m"  -ForegroundColor Yellow }
function Err  ($m) { Write-Host "  [X] $m"  -ForegroundColor Red }

if ($Help) {
  @"
Usage: powershell -ExecutionPolicy Bypass -File uninstall_collector.ps1 [options]

Options:
  -Root PATH         Uninstall this install root (repeatable: -Root A -Root B). Without -Root,
                      every install found on this machine is discovered and handled.
  -DryRun             Print the full plan for every root found and change NOTHING. Exit 0.
  -Yes                Do not ask for confirmation (the agent's automated path - use only after
                      the human has explicitly confirmed, in their own words, that they want
                      everything removed).
  -KeepSource         Do not delete <root>/solo-agency (the source checkout).
  -KeepRootDir        Do not remove <root> even if it ends up empty.
  -Port N             Collector bridge port to check for a stray listener (default 17321, or
                      `$env:SOLO_AGENCY_BRIDGE_PORT).
  -OpenBrowser        Also open chrome://extensions so the human can remove the extension
                      entries by hand.
  -Help               Show this help.

Exit codes: 0 = everything scripted succeeded (or -DryRun). 2 = at least one scripted step
failed, or a -Root was refused for safety - see the summary.
"@ | Write-Host
  exit 0
}

# --- MULTI_BRAIN_OPERATIONS.md install-root pointer-file signatures ----------
$AgentsSignature = 'This folder is a LIVE Solo Agency install, not a fresh setup.'
$ClaudeSignature = 'Read `AGENTS.md` in this same folder now, then `solo-agency/AGENTS.md` in full'

function Resolve-AbsPath ($p) {
  try { return (Resolve-Path -LiteralPath $p -ErrorAction Stop).Path } catch { return $p }
}

# True if $1 and $2 refer to the same directory. Compares the .NET-normalized
# full path (case-insensitively, matching Windows' case-insensitive-preserving
# filesystems) after both sides have gone through Resolve-AbsPath, so a
# reparse-point/junction alias or a differently-cased spelling of the same
# directory is treated as identical rather than as a distinct path.
function Test-SamePath ($a, $b) {
  if ([string]::IsNullOrWhiteSpace($a) -or [string]::IsNullOrWhiteSpace($b)) { return $false }
  $ra = (Resolve-AbsPath $a).TrimEnd('\', '/')
  $rb = (Resolve-AbsPath $b).TrimEnd('\', '/')
  return ($ra -ieq $rb)
}

# --- safety invariant: the only guard standing between this script and a wide
# Remove-Item -Recurse. A root is valid only if it is not the filesystem root, not
# $HOME/$env:USERPROFILE, and actually looks like a Solo Agency install (carries
# daily-content-pipeline/ or solo-agency-local-collector/). Every delete target
# below is built by joining a root that passed THIS check with a fixed, hardcoded
# subdirectory name - never a user-supplied glob or pattern.
function Test-RootSafe ($r) {
  if ([string]::IsNullOrWhiteSpace($r)) { return $false }
  $rp = $r.TrimEnd('\', '/')
  if ([string]::IsNullOrWhiteSpace($rp)) { return $false }  # "C:\" -> "" after trim = drive root
  $home1 = $env:USERPROFILE
  if ($home1) { $home1 = $home1.TrimEnd('\', '/') }
  if ($rp -ieq $home1) { return $false }
  # Belt-and-suspenders on top of the literal -ieq above: also compare through
  # Resolve-AbsPath, which is what catches a reparse-point/junction alias to
  # USERPROFILE (the literal-string compare alone would not).
  if ($home1 -and (Test-Path -LiteralPath $rp) -and (Test-SamePath $rp $home1)) { return $false }
  # Reject bare drive roots like "C:" or "C:\"
  if ($rp -match '^[A-Za-z]:$') { return $false }
  if ((Test-Path (Join-Path $rp 'daily-content-pipeline')) -or (Test-Path (Join-Path $rp 'solo-agency-local-collector'))) {
    return $true
  }
  return $false
}

function Get-Sha8 ($s) {
  $sha = [System.Security.Cryptography.SHA256]::Create()
  ([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($s))) -replace '-', '').Substring(0, 8).ToLower()
}

# --- root discovery ------------------------------------------------------------
$FoundRoots = New-Object System.Collections.Generic.List[string]
function Add-Root ($r) {
  if (-not $r) { return }
  $rp = Resolve-AbsPath $r
  if (-not ($FoundRoots -contains $rp)) { $FoundRoots.Add($rp) }
}

function Get-RootFromBinPath ($binPath) {
  if (-not $binPath) { return $null }
  if ($binPath -notmatch '[\\/]solo-agency-local-collector[\\/]bin[\\/]collector-bridge-') { return $null }
  $d1 = Split-Path $binPath -Parent      # ...\solo-agency-local-collector\bin
  $d2 = Split-Path $d1 -Parent           # ...\solo-agency-local-collector
  $d3 = Split-Path $d2 -Parent           # <root>
  return $d3
}

function Discover-FromScriptLocation {
  if (-not $PSScriptRoot) { return }
  # <root>/solo-agency/solo-agency-collector/uninstall_collector.ps1
  $twoUp = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
  if ($twoUp -and ((Test-Path (Join-Path $twoUp 'daily-content-pipeline')) -or (Test-Path (Join-Path $twoUp 'solo-agency-local-collector')))) {
    Add-Root $twoUp
  }
  # <root>/solo-agency-local-collector/uninstall_collector.ps1 (copied here by setup_collector.ps1)
  $oneUp = Split-Path $PSScriptRoot -Parent
  if ($oneUp -and (Test-Path (Join-Path $oneUp 'daily-content-pipeline'))) {
    Add-Root $oneUp
  }
}

function Discover-FromScheduledTasks {
  if (-not (Get-Command Get-ScheduledTask -ErrorAction SilentlyContinue)) { return }
  try {
    $tasks = Get-ScheduledTask -TaskName 'SoloAgencyCollector-*' -ErrorAction SilentlyContinue
  } catch { $tasks = @() }
  foreach ($t in $tasks) {
    foreach ($a in $t.Actions) {
      # Prefer the structured Execute property directly — it is the literal
      # path as registered, never word/whitespace-split — over regex-matching
      # a rebuilt "$Execute $Arguments" string, which would silently truncate
      # at the first space for a root under a path containing one (e.g. a
      # synced-folder mount named with spaces).
      if ($a.Execute -and ($a.Execute -match 'collector-bridge')) {
        $r = Get-RootFromBinPath $a.Execute
        if ($r) { Add-Root $r }
      } else {
        $line = "$($a.Execute) $($a.Arguments)"
        if ($line -match '([A-Za-z]:\\.*?collector-bridge-[^\s"]*|/.*?collector-bridge-[^\s"]*)') {
          $r = Get-RootFromBinPath $Matches[1]
          if ($r) { Add-Root $r }
        }
      }
    }
  }
}

function Discover-FromPortAndProcesses {
  $pids = @()
  if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
    try {
      $pids += (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess)
    } catch { }
  }
  try {
    $pids += (Get-CimInstance Win32_Process -Filter "Name LIKE '%collector-bridge%'" -ErrorAction SilentlyContinue | Select-Object -ExpandProperty ProcessId)
  } catch { }
  foreach ($procId in ($pids | Select-Object -Unique)) {
    if (-not $procId) { continue }
    try {
      $cim = Get-CimInstance Win32_Process -Filter "ProcessId = $procId" -ErrorAction SilentlyContinue
    } catch { $cim = $null }
    if (-not $cim) { continue }
    # Prefer the structured ExecutablePath field (never whitespace-split) over
    # regex-matching the rebuilt "$ExecutablePath $CommandLine" string, which
    # would silently truncate at the first space in a root path that contains
    # one.
    if ($cim.ExecutablePath -and ($cim.ExecutablePath -match 'collector-bridge')) {
      $r = Get-RootFromBinPath $cim.ExecutablePath
      if ($r) { Add-Root $r }
      continue
    }
    $cmd = "$($cim.ExecutablePath) $($cim.CommandLine)"
    if ($cmd -notmatch 'collector-bridge') { continue }
    if ($cmd -match '([A-Za-z]:\\.*?collector-bridge-[^\s"]*)') {
      $r = Get-RootFromBinPath $Matches[1]
      if ($r) { Add-Root $r }
    }
  }
}

if ($Root.Count -gt 0) {
  foreach ($r in $Root) { Add-Root $r }
} else {
  Discover-FromScriptLocation
  Discover-FromScheduledTasks
  Discover-FromPortAndProcesses
}

if ($FoundRoots.Count -eq 0) {
  Say 'No Solo Agency install found on this machine.'
  Info "(searched: this script's own location, Scheduled Task registrations, and port $Port)"
  exit 0
}

# --- validate every root before touching anything ------------------------------
# A root that simply does not exist (a previous uninstall already removed it, or
# -Root was given a typo of a path nothing ever occupied) is a no-op, not a safety
# refusal - that is what makes running this twice in a row idempotent. An EXISTING
# path that fails the marker check (including the drive root and $env:USERPROFILE,
# which always exist) is refused: this script will never guess that unrecognized,
# still-present content is safe to delete.
$ValidRoots = New-Object System.Collections.Generic.List[string]
$AnyRefused = $false
foreach ($r in $FoundRoots) {
  if (-not (Test-Path -LiteralPath $r)) {
    Info "Nothing to uninstall - path does not exist (already removed): $r"
  } elseif (Test-RootSafe $r) {
    $ValidRoots.Add($r)
  } else {
    Err "Refusing root (fails safety check - not a drive root, not `$env:USERPROFILE, and must contain daily-content-pipeline/ or solo-agency-local-collector/): $r"
    $AnyRefused = $true
  }
}

if ($ValidRoots.Count -eq 0) {
  Say 'Nothing to uninstall - no valid Solo Agency install root.'
  exit $(if ($AnyRefused) { 2 } else { 0 })
}

Say 'Solo Agency Local Collector uninstall - plan'
Info "Install root(s) found: $($ValidRoots.Count)"
foreach ($r in $ValidRoots) { Info "  - $r" }
if ($KeepSource)  { Info '-KeepSource: source checkout (solo-agency/) will be kept' }
if ($KeepRootDir) { Info '-KeepRootDir: root directory will be kept even if empty' }

# run_with_timeout: a fake/hung binary can never stall the uninstall.
function Invoke-WithTimeout ($Seconds, [scriptblock]$Script) {
  $job = Start-Job -ScriptBlock $Script
  $done = Wait-Job $job -Timeout $Seconds
  if (-not $done) {
    Stop-Job $job -ErrorAction SilentlyContinue | Out-Null
    Remove-Job $job -Force -ErrorAction SilentlyContinue | Out-Null
    return @{ Success = $false; Output = '(timed out)' }
  }
  $out = Receive-Job $job -ErrorAction SilentlyContinue
  $ok = ($job.State -eq 'Completed')
  Remove-Job $job -Force -ErrorAction SilentlyContinue | Out-Null
  return @{ Success = $ok; Output = ($out -join ' ') }
}

$Removed = New-Object System.Collections.Generic.List[string]
$Skipped = New-Object System.Collections.Generic.List[string]
$Manual  = New-Object System.Collections.Generic.List[string]
$Failed  = New-Object System.Collections.Generic.List[string]

function Remove-Target ($Target, $Desc, $RootLabel, $Mode) {
  if (-not (Test-Path -LiteralPath $Target)) {
    Info "$(if ($Mode -eq 'plan') { '[would]' } else { '[did] ' }) ${Desc}: nothing at $Target"
    return
  }
  if ($Mode -eq 'plan') {
    Info "[would] Remove-Item -Recurse -Force `"$Target`""
  } else {
    try {
      Remove-Item -LiteralPath $Target -Recurse -Force -ErrorAction Stop
      Ok "removed $Desc"
      $Removed.Add("${RootLabel}: $Desc ($Target)")
    } catch {
      Warn "failed to remove $Target ($($_.Exception.Message))"
      $Failed.Add("${RootLabel}: Remove-Item $Target")
    }
  }
}

function Invoke-ProcessRoot ($r, $Mode) {
  $prefix = if ($Mode -eq 'plan') { '[would]' } else { '[did] ' }
  Say "Root: $r  ($(if ($Mode -eq 'plan') { 'PLAN - nothing changes' } else { 'EXECUTING' }))"

  $instHash = Get-Sha8 $r
  $taskName = "SoloAgencyCollector-$instHash"

  # setup_collector.ps1 hashes ROOT as resolved at setup time, which may not be
  # byte-identical to $r if $r reached this script through a differently-cased
  # or reparse-point path (Resolve-AbsPath normalizes most of that, but is not
  # guaranteed to correct case on every PowerShell/filesystem combination). If
  # the hash-derived task name isn't registered, fall back to scanning every
  # SoloAgencyCollector-* task's own recorded binary path and matching it to
  # $r by resolved-path identity, so this can still find a task that setup
  # actually created for this root.
  $existingTaskProbe = $null
  try { $existingTaskProbe = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue } catch { }
  if (-not $existingTaskProbe -and (Get-Command Get-ScheduledTask -ErrorAction SilentlyContinue)) {
    try { $allTasks = Get-ScheduledTask -TaskName 'SoloAgencyCollector-*' -ErrorAction SilentlyContinue } catch { $allTasks = @() }
    foreach ($t in $allTasks) {
      foreach ($a in $t.Actions) {
        $bp = $a.Execute
        if ($bp -notmatch 'collector-bridge') { continue }
        $tr = Get-RootFromBinPath $bp
        if ($tr -and (Test-SamePath $tr $r)) { $taskName = $t.TaskName; break }
      }
    }
  }

  # (a) entitlement seat release - best effort, never fails the uninstall.
  $binPath = $null
  $binDir = Join-Path $r 'solo-agency-local-collector\bin'
  if (Test-Path $binDir) {
    $f = Get-ChildItem -Path $binDir -Filter 'collector-bridge-*' -File -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($f) { $binPath = $f.FullName }
  }
  $hasKey = $false
  if ($binPath) {
    $clientsDir = Join-Path $r 'daily-content-pipeline\clients'
    if (Test-Path $clientsDir) {
      $hit = Get-ChildItem -Path $clientsDir -Filter 'provider_config.local.json' -Recurse -Depth 6 -ErrorAction SilentlyContinue | Select-Object -First 1
      if ($hit) { $hasKey = $true }
    }
    Get-ChildItem Env: | Where-Object { $_.Name -match '^(SOLO|OUTREACHCRM)[A-Z0-9_]*KEY$' } | ForEach-Object { $hasKey = $true }
  }
  if ($binPath -and $hasKey) {
    $pipelineDir = Join-Path $r 'daily-content-pipeline'
    if ($Mode -eq 'plan') {
      Info "$prefix release entitlement seat: `"$binPath`" tool entitlement release --pipeline `"$pipelineDir`""
    } else {
      Info 'Releasing entitlement seat (best effort, 20s timeout)...'
      $res = Invoke-WithTimeout 20 ([scriptblock]::Create("& '$binPath' tool entitlement release --pipeline '$pipelineDir' 2>&1"))
      if ($res.Success) { Ok 'entitlement seat released' } else { Warn "entitlement release did not succeed (non-fatal): $($res.Output)" }
    }
  } else {
    Info "$prefix skip entitlement release (no bridge binary or no provider key found)"
  }

  # (b) stop the Scheduled Task.
  $existingTask = $null
  try { $existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue } catch { }
  if ($existingTask) {
    if ($Mode -eq 'plan') {
      Info "$prefix Stop-ScheduledTask + Unregister-ScheduledTask -TaskName $taskName"
    } else {
      try { Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue } catch { }
      try {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction Stop
        Ok "removed Scheduled Task $taskName"
        $Removed.Add("${r}: Scheduled Task $taskName")
      } catch {
        Warn "could not remove Scheduled Task $taskName ($($_.Exception.Message))"
        $Failed.Add("${r}: Unregister-ScheduledTask $taskName")
      }
    }
  } else {
    Info "$prefix no Scheduled Task registered for this root ($taskName) - nothing to stop"
  }

  # (c) kill the bridge process - PID file, then anything still on the port - and
  # ONLY when its command line contains "collector-bridge".
  $pidFile = Join-Path $r 'solo-agency-local-collector\collector.pid'
  if (Test-Path $pidFile) {
    $procId = (Get-Content $pidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
    if ($procId) {
      try { $cim = Get-CimInstance Win32_Process -Filter "ProcessId = $procId" -ErrorAction SilentlyContinue } catch { $cim = $null }
      if ($cim) {
        $cmd = "$($cim.ExecutablePath) $($cim.CommandLine)"
        if ($cmd -match 'collector-bridge') {
          if ($Mode -eq 'plan') {
            Info "$prefix Stop-Process -Id $procId ($cmd)"
          } else {
            try { Stop-Process -Id ([int]$procId) -Force -ErrorAction Stop; Ok "stopped collector-bridge PID $procId" } catch { Warn "could not stop PID $procId" }
          }
        } else {
          Info "$prefix collector.pid PID $procId is not a collector-bridge process - leaving it alone"
        }
      }
    }
  }
  if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
    $conns = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    foreach ($c in $conns) {
      try { $cim = Get-CimInstance Win32_Process -Filter "ProcessId = $($c.OwningProcess)" -ErrorAction SilentlyContinue } catch { $cim = $null }
      if (-not $cim) { continue }
      $cmd = "$($cim.ExecutablePath) $($cim.CommandLine)"
      if ($cmd -match 'collector-bridge') {
        # The collector port (17321 by default) is a single shared value across
        # every install on this machine - "the listener's command contains
        # collector-bridge" is NOT proof it belongs to THIS root. Resolve its
        # actual binary path back to a root and require that root be this one
        # (by resolved-path identity) before killing it; a collector-bridge for
        # a different, still-installed root is left running.
        $listenerBin = if ($cim.ExecutablePath -and ($cim.ExecutablePath -match 'collector-bridge')) { $cim.ExecutablePath } else { $null }
        $listenerRoot = if ($listenerBin) { Get-RootFromBinPath $listenerBin } else { $null }
        if ($listenerRoot -and -not (Test-SamePath $listenerRoot $r)) {
          Info "port $Port is held by another install's collector-bridge (PID $($c.OwningProcess), root $listenerRoot) - left running"
          if ($Mode -eq 'execute') { $Manual.Add("${r}: port $Port held by a DIFFERENT Solo Agency install's collector-bridge (PID $($c.OwningProcess), root $listenerRoot) - not touched; stop it yourself if you actually mean to remove that install too") }
          continue
        }
        if ($Mode -eq 'plan') {
          Info "$prefix Stop-Process -Id $($c.OwningProcess) (listener on port $Port: $cmd)"
        } else {
          try { Stop-Process -Id $c.OwningProcess -Force -ErrorAction Stop; Ok "stopped listener on port $Port (PID $($c.OwningProcess))" } catch { Warn "could not stop PID $($c.OwningProcess)" }
        }
      } else {
        Info "port $Port is held by a NON-collector process (PID $($c.OwningProcess)): $cmd - left running"
        if ($Mode -eq 'execute') { $Manual.Add("${r}: port $Port held by non-collector PID $($c.OwningProcess) - stop it yourself if needed") }
      }
    }
  }

  # (d) delete runtime state - every path below is <root> (already safety-checked)
  # joined with a fixed subdirectory name. No wildcards, no user input in the path.
  Remove-Target (Join-Path $r 'solo-agency-local-collector') 'local collector runtime' $r $Mode
  Remove-Target (Join-Path $r 'extensions') 'per-client Chrome extension copies' $r $Mode
  Remove-Target (Join-Path $r 'daily-content-pipeline') 'pipeline state (CRM, content library, config, secrets)' $r $Mode

  # <root>\outreach: no repo-defined marker distinguishes a live install's outreach
  # STATE directory from an accidental source checkout at that path. Be conservative:
  # never touch it.
  $outreachPath = Join-Path $r 'outreach'
  if (Test-Path $outreachPath) {
    Info "left in place: not a confirmed Solo Agency state dir: $outreachPath (no repo-defined marker - verify and remove by hand if it is agency state)"
    if ($Mode -eq 'execute') { $Skipped.Add("${r}: outreach (no marker, left in place)") }
  }
  $outreachCrmPath = Join-Path $r 'outreachcrm'
  if (Test-Path $outreachCrmPath) {
    Remove-Target $outreachCrmPath 'outreach CRM state' $r $Mode
  }

  foreach ($pf in @('AGENTS.md', 'CLAUDE.md')) {
    $p = Join-Path $r $pf
    $sig = if ($pf -eq 'AGENTS.md') { $AgentsSignature } else { $ClaudeSignature }
    if (Test-Path -LiteralPath $p -PathType Leaf) {
      $content = Get-Content -LiteralPath $p -Raw -ErrorAction SilentlyContinue
      if ($content -and $content.Contains($sig)) {
        if ($Mode -eq 'plan') {
          Info "$prefix Remove-Item `"$p`" (matches MULTI_BRAIN_OPERATIONS.md pointer-file signature)"
        } else {
          try { Remove-Item -LiteralPath $p -Force -ErrorAction Stop; Ok "removed pointer file $pf"; $Removed.Add("${r}: $pf") }
          catch { Warn "failed to remove $p"; $Failed.Add("${r}: Remove-Item $p") }
        }
      } else {
        Info "left in place: not a Solo Agency pointer file: $p"
        if ($Mode -eq 'execute') { $Skipped.Add("${r}: $pf (no signature match, left in place)") }
      }
    }
  }

  $sourcePath = Join-Path $r 'solo-agency'
  if ($KeepSource) {
    Info "$prefix keep $sourcePath (-KeepSource)"
    if ((Test-Path $sourcePath) -and ($Mode -eq 'execute')) { $Skipped.Add("${r}: solo-agency (kept: -KeepSource)") }
  } else {
    Remove-Target $sourcePath 'source checkout' $r $Mode
  }

  if ($KeepRootDir) {
    Info "$prefix keep root directory (-KeepRootDir)"
  } else {
    if ($Mode -eq 'plan') {
      Info "$prefix Remove-Item `"$r`" (only if empty afterward)"
    } else {
      $remaining = @(Get-ChildItem -LiteralPath $r -Force -ErrorAction SilentlyContinue)
      if ($remaining.Count -eq 0) {
        try { Remove-Item -LiteralPath $r -Force -ErrorAction Stop; Ok 'removed empty root directory'; $Removed.Add("${r}: root directory") }
        catch { Info "root directory left in place - could not remove: $($_.Exception.Message)" }
      } else {
        Info "root directory left in place - not empty: $(($remaining | Select-Object -ExpandProperty Name) -join ' ')"
      }
    }
  }

  # (e) CLI-era Claude scheduled-task folders. .claude\scheduled-tasks is a
  # SINGLE machine-wide directory shared by every Solo Agency install (and every
  # other Claude project) on this box - it carries no root/insthash in its own
  # name, so a "*solo-agency*" filter here cannot tell "this root's task" apart
  # from a different, still-installed root's task, or another client's. Never
  # auto-delete it: list matches for the human to review and remove by hand,
  # the same treatment already given to the Claude desktop Scheduled panel and
  # Codex automations below.
  $claudeDir = if ($env:SOLO_AGENCY_CLAUDE_SCHEDULED_TASKS_DIR) { $env:SOLO_AGENCY_CLAUDE_SCHEDULED_TASKS_DIR } else { Join-Path $env:USERPROFILE '.claude\scheduled-tasks' }
  if (Test-Path $claudeDir) {
    Get-ChildItem -Path $claudeDir -Filter '*solo-agency*' -Directory -ErrorAction SilentlyContinue | ForEach-Object {
      Info "$prefix NOT deleted (machine-wide, not root-scoped - review by hand): $($_.FullName)"
      if ($Mode -eq 'execute') { $Manual.Add("(machine-wide, review before deleting - may belong to a different install) $($_.FullName)") }
    }
  }

  if ($Mode -eq 'execute') {
    $Manual.Add("${r}: Claude desktop app - Scheduled panel entries for this install (e.g. '<client>-solo-agency-daily-run', 'solo-agency-github-update-watch') - remove them there; this script cannot reach that app's own registry.")
    $Manual.Add("${r}: Codex automations for this install (if any) - remove them in Codex's own automations UI.")
    $Manual.Add("${r}: chrome://extensions - the unpacked Solo Agency extension entries loaded from $r\extensions\*\ (one per Chrome profile) - Load unpacked folders are gone from disk now, but Chrome keeps the entry until you remove it there.")
    if ($OpenBrowser) {
      try { Start-Process 'chrome.exe' 'chrome://extensions/' -ErrorAction Stop } catch { try { Start-Process 'chrome://extensions/' } catch { } }
    }
  } else {
    Info "$prefix note manual leftovers (Claude desktop Scheduled panel, Codex automations, chrome://extensions entries)"
  }
}

foreach ($r in $ValidRoots) { Invoke-ProcessRoot $r 'plan' }

if ($DryRun) {
  Say 'Dry run - nothing was changed.'
  exit $(if ($AnyRefused) { 2 } else { 0 })
}

if (-not $Yes) {
  Say "This will permanently delete the above for $($ValidRoots.Count) install(s)."
  $ans = Read-Host 'Proceed? [y/N]'
  if ($ans -notmatch '^(y|yes)$') {
    Say 'Aborted - nothing changed.'
    exit 0
  }
}

Say 'Uninstalling...'
foreach ($r in $ValidRoots) { Invoke-ProcessRoot $r 'execute' }

Say 'Summary'
Info "Removed ($($Removed.Count)):"
foreach ($x in $Removed) { Info "  - $x" }
Info "Skipped/left in place ($($Skipped.Count)):"
foreach ($x in $Skipped) { Info "  - $x" }
Info "Manual leftovers to remove yourself ($($Manual.Count)):"
foreach ($x in $Manual) { Info "  - $x" }
$AnyFailed = $false
if ($Failed.Count -gt 0) {
  Warn "Failed steps ($($Failed.Count)):"
  foreach ($x in $Failed) { Warn "  - $x" }
  $AnyFailed = $true
}
Info 'Untouched by design: ~/.claude/projects transcripts and this AI runtime''s own CLAUDE.md - they are the AI runtime''s own data, not Solo Agency''s.'

if ($AnyFailed -or $AnyRefused) {
  Say 'Uninstall finished WITH ERRORS - see Failed steps above.'
  exit 2
}
Say 'Uninstall complete.'
exit 0

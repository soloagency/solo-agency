#!/usr/bin/env bash
# playbooks/tests/wait_for_run_test.sh — regression guard for tools/wait_for_run (S10).
# Runs each case in its own temp dir with a fake daily-content-pipeline/automation/, using
# --interval 1 --timeout 3 and a background writer (sleep 1; echo line >> file) so the
# helper has to actually poll. Prints PASS/FAIL per case and exits non-zero on any failure.
set -u

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
wfr="$repo_root/tools/wait_for_run"
since="2026-09-12T08:00:00Z"
after="2026-09-12T08:05:00Z"

fail_count=0

report() {
  local case_name="$1" ok="$2" detail="$3"
  if [ "$ok" -eq 0 ]; then
    echo "PASS: $case_name"
  else
    echo "FAIL: $case_name — $detail"
    fail_count=$((fail_count + 1))
  fi
}

new_pipeline_dir() {
  local d
  d="$(mktemp -d)"
  mkdir -p "$d/daily-content-pipeline/automation"
  printf '%s' "$d"
}

# (a) --watch progress exits 0 with "progress " prefix on a new run_progress line.
case_a() {
  local d progress out rc
  d="$(new_pipeline_dir)"
  progress="$d/daily-content-pipeline/automation/run_progress.jsonl"
  ( sleep 1; printf '{"ts":"%s","client_slug":"acme","stage":"find_people"}\n' "$after" >> "$progress" ) &
  out="$("$wfr" acme "$since" --pipeline "$d/daily-content-pipeline" --interval 1 --timeout 3 --watch progress)"
  rc=$?
  wait
  rm -rf "$d"
  if [ $rc -ne 0 ]; then report "a: --watch progress on new run_progress line" 1 "exit $rc, want 0"; return; fi
  case "$out" in
    "progress "*) report "a: --watch progress on new run_progress line" 0 "" ;;
    *) report "a: --watch progress on new run_progress line" 1 "output was: $out" ;;
  esac
}

# (b) --watch progress exits 0 with "standup " prefix when only a standup line lands.
case_b() {
  local d standup out rc
  d="$(new_pipeline_dir)"
  standup="$d/daily-content-pipeline/automation/standup.jsonl"
  ( sleep 1; printf '{"ts":"%s","client_slug":"acme","status":"done"}\n' "$after" >> "$standup" ) &
  out="$("$wfr" acme "$since" --pipeline "$d/daily-content-pipeline" --interval 1 --timeout 3 --watch progress)"
  rc=$?
  wait
  rm -rf "$d"
  if [ $rc -ne 0 ]; then report "b: --watch progress on standup-only line" 1 "exit $rc, want 0"; return; fi
  case "$out" in
    "standup "*) report "b: --watch progress on standup-only line" 0 "" ;;
    *) report "b: --watch progress on standup-only line" 1 "output was: $out" ;;
  esac
}

# (c) exit 3 on timeout (nothing ever written).
case_c() {
  local d out rc
  d="$(new_pipeline_dir)"
  out="$("$wfr" acme "$since" --pipeline "$d/daily-content-pipeline" --interval 1 --timeout 3 --watch progress)"
  rc=$?
  rm -rf "$d"
  report "c: exit 3 on timeout" $([ $rc -eq 3 ] && echo 0 || echo 1) "exit $rc, want 3"
}

# (d) exit 2 on bad args (missing required <since_iso8601>).
case_d() {
  local out rc
  out="$("$wfr" acme 2>/dev/null)"
  rc=$?
  report "d: exit 2 on bad args" $([ $rc -eq 2 ] && echo 0 || echo 1) "exit $rc, want 2"
}

# (e) legacy call without --watch still exits 0 on the standup line only, unprefixed.
case_e() {
  local d standup progress out rc
  d="$(new_pipeline_dir)"
  standup="$d/daily-content-pipeline/automation/standup.jsonl"
  progress="$d/daily-content-pipeline/automation/run_progress.jsonl"
  ( sleep 1
    printf '{"ts":"%s","client_slug":"acme","stage":"find_people"}\n' "$after" >> "$progress"
    printf '{"ts":"%s","client_slug":"acme","status":"done"}\n' "$after" >> "$standup"
  ) &
  out="$("$wfr" acme "$since" --pipeline "$d/daily-content-pipeline" --interval 1 --timeout 3)"
  rc=$?
  wait
  rm -rf "$d"
  if [ $rc -ne 0 ]; then report "e: legacy call (no --watch) on standup line" 1 "exit $rc, want 0"; return; fi
  case "$out" in
    "progress "*|"standup "*) report "e: legacy call (no --watch) on standup line" 1 "output carried a watch prefix: $out" ;;
    *"client_slug"*"acme"*) report "e: legacy call (no --watch) on standup line" 0 "" ;;
    *) report "e: legacy call (no --watch) on standup line" 1 "output was: $out" ;;
  esac
}

case_a
case_b
case_c
case_d
case_e

if [ "$fail_count" -gt 0 ]; then
  echo "$fail_count case(s) failed"
  exit 1
fi
echo "all cases passed"
exit 0

# Solo Agency Collector Development Instructions

These rules supplement `/Users/binhnguyen/Downloads/soloagency/AGENTS.md`; they do not replace it. Read and follow the parent file first. The parent remains authoritative for Solo Agency product, setup, playbook, licensing, entitlement, privacy, source-access, and operating rules. Newer direct user instructions take precedence. If instructions conflict materially, stop and ask the user before editing.

## Change review and approval

- Before implementing a bug fix, investigate and explain the root cause with evidence.
- Before editing, analyze possible negative effects and regressions across all affected paths. For this collector, include the Go bridge, Chrome extension, scripts, provider commands, stored client data and file bus, macOS/Windows/Linux behavior, install/update/uninstall flows, concurrency, caches, authentication, entitlement and licensing, privacy and security, backward compatibility, and existing live installs where applicable.
- Present the proposed solution to the user before applying a bug fix. State the exact scope, affected and unaffected paths, alternatives considered, trade-offs, known risks, rollback approach, and local verification plan. Wait for explicit approval before editing.
- For features, refactors, migrations, dependency changes, or operational changes with material risk or trade-offs, use the same review and approval process.
- Read-only investigation and a minimal local reproduction may be performed before approval. Do not mutate production or client data while investigating.
- If testing or new evidence materially changes the approved solution, stop and consult the user again before expanding or changing the implementation.

## Backups

- Before editing any existing source, configuration, test, script, or documentation file, create a timestamped backup under `/Users/binhnguyen/Downloads/soloagency/solo-agency-collector/.backups`.
- Preserve the source file's relative directory inside `.backups` when needed to avoid collisions between files with the same name.
- Use a timestamped name such as `{base_filename}_{YYYYMMDD}_{HHMMSS}.{extension}`. For extensionless files, use `{filename}_{YYYYMMDD}_{HHMMSS}.backup`.
- Never overwrite an existing backup. If a name already exists, choose a new timestamp.
- Do not leave new backup files beside source files.
- A newly created file does not need a backup, but confirm that it does not already exist immediately before creating it.

## Local testing and verification

- Test logic locally before handing work back to the user. Do not describe a change as complete or safe without relevant verification.
- For a bug fix, reproduce the failure locally when practical, then run the same reproduction after the edit and add focused regression checks for nearby behavior.
- Start with the smallest relevant tests, then run proportional integration or regression tests for every affected layer and contract. A large repository is not a reason to skip targeted tests.
- For changes crossing the Go bridge and Chrome extension, verify both sides of the message or API contract, including missing, stale, malformed, legacy, and partially upgraded data where relevant.
- For persistence, migration, queue, cache, concurrency, authentication, entitlement, or install/update changes, verify failure and rollback behavior as well as the successful path.
- Put temporary tests, harnesses, stubs, fixtures, screenshots, logs, and other disposable verification artifacts only in `/Users/binhnguyen/Downloads/soloagency/solo-agency-collector/localtest`.
- Delete temporary artifacts after verification. Resolve and verify every exact target before deleting it. Never use a broad glob or recursive deletion that could remove source code, permanent tests, or user data.
- Keep permanent tests in the existing `tests` or component-specific test directories only when the user explicitly requests permanent coverage or it is part of the approved solution.
- If a required test cannot run, report the exact command, blocker, unverified behavior, and resulting risk. Do not silently treat an environmental failure as a pass.

## Generated and third-party files

- Do not edit minified, bundled, generated, compiled, vendored, binary, or distribution artifacts directly, including files under `dist`, unless the user explicitly requests it and the approved plan identifies the authoritative source and regeneration process.
- Prefer changing the authoritative source and using the documented build or packaging process.
- Do not regenerate build or version artifacts solely because a source file changed unless the approved task requires it.

## Repository and data safety

- Preserve unrelated user changes and work carefully in a dirty working tree. Never discard or revert unrelated edits.
- Do not use destructive commands, broad path globs, `git reset --hard`, or `git checkout --` unless the user clearly authorizes the exact operation.
- Never test against real client data when synthetic or isolated fixtures are sufficient. Do not expose keys, tokens, personal data, cookies, messages, or private source content in commands, logs, screenshots, fixtures, or handoff notes.
- Treat changes to license, entitlement, authentication, privacy, data access, collection boundaries, or provider behavior as high risk. The parent `AGENTS.md` rules remain mandatory and must not be bypassed.
- Prefer narrow, reversible changes with explicit fallbacks. Preserve legacy behavior for data or clients outside the approved scope.

## Subagent model floor

- Never use mini, spark, or any model tier below the floors defined here, including for explorer or codebase-scouting work.
- For Codex/OpenAI subagents, use Luna or Terra at minimum. If the preferred model is unavailable, move upward to a stronger supported model; never downgrade to mini.
- For Claude subagents, use Haiku or Sonnet at minimum. If the preferred model is unavailable, move upward to a stronger supported model.
- Do not use a role that forces a model below these floors. Use a regular subagent at an allowed model tier for read-only scouting instead.

## Handoff requirements

- Summarize the root cause and the final solution.
- List every changed file and its backup path.
- Report the exact local tests run and their results.
- State the affected and intentionally unaffected paths, remaining risks, and rollback path.
- State any required build, packaging, migration, restart, deployment, extension reload, or automation-sync step. Never imply that unperformed operational steps have already happened.

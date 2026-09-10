# Solo Agency Local Collector

This is the Chrome MV3 extension for the Solo Agency Local Collector.

The user installs it once in the Chrome profile that is already logged in to Facebook, LinkedIn, Reddit, Instagram, TikTok, or other private data sources they want monitored.

## What It Does

- Polls `http://127.0.0.1:17321/status`.
- Polls every 5 seconds by default while enabled.
- Uses Chrome alarms as a wake-up fallback because Manifest V3 service workers can sleep.
- Fetches the current local job from the bridge.
- Opens configured source URLs in inactive tabs using the user's existing Chrome session.
- Closes collector-created tabs after scanning when configured.
- Waits 5 seconds between scroll/read actions by default.
- Uses 5 scrolls per private data source by default, configurable up to 10.
- Reads visible text, URLs, page title, engagement hints, profile URL candidates, and post/current URL candidates.
- Detects related recommended private groups/pages/communities and sends them as `new_private_source` records for human review.
- Sends structured data back to the local bridge.
- When the user changes collector settings in the panel, it tries to save the shared config to the local bridge through `/config`.

## What It Does Not Do

- It does not ask for passwords.
- It does not read cookies directly.
- It does not upload data to cloud services.
- It does not post, comment, react, message, follow, or change account state.
- It does not bypass platform access controls.
- It does not collect if Chrome is closed, the browser profile is not running, or the extension is disabled/removed.
- It does not guarantee a permanently awake background worker in all browser/OS power states.
- Hidden/background tabs may be throttled by Chrome; build `0.1.10-filtering-capture` and newer reduces this risk, gives 5-10 scroll social captures enough time to finish, clears stale active-run locks after updates, and keeps the full `filtering.js` capture pipeline. If tabs open/close but no data reaches disk, audit bridge/extension identity, write token, POST responses, and output folder routing before changing the capture pipeline.

## Source Layout

Platform-specific capture code lives under `platforms/<name>/` — today `facebook/`
(`gql_intercept.js`, `gql_extract.js`, `gql_actions.js`, `fb_normalize.js`), `instagram/`
(`ig_intercept.js`, `ig_extract.js`, `ig_normalize.js`; the second platform module, see
`INSTAGRAM_CAPABILITIES.md`)
and `zillow/` (`zillow_extract.js`, `zillow_normalize.js`). Each platform is registered as data in
`core/platform_registry.js` (which files `background.js` injects, which entry point it calls,
per-capability metadata); `core/schema.js` defines the one canonical record shape every
platform's normalizer maps into. Shared, platform-neutral files stay at this top level:
`collector_helpers.js`, `contact_extract.js`, `filtering.js`, `infinity_loops.js`,
`readability.js`, `offscreen.js`, `solo_entitlement.js`, `popup.*`, `audit.*`. See
`../GRAPHQL_MAINTENANCE.md` §2 for the full file map.

## Runtime Install

This folder is the source/developer copy of the Chrome extension. As of 2026-09-10 its own
`manifest.json` says so: the extension `name` and toolbar `default_title` are literally
**"Solo Agency Collector (source, do not load)"** — that way even someone who loads this folder
by mistake sees a name that says not to, right in `chrome://extensions` and on hover, instead of
a name indistinguishable from a real client's.

For a normal Solo Agency runtime setup, do not load this folder in Chrome. Every client gets its
own generated, unpacked copy instead, prepared by `solo-agency-collector/scripts/prepare_client_extension.sh`:

```text
extensions/{client_slug}_extension/
```

The folder name changed from `extensions/{client_slug}/` (2026-09-10) specifically because the
old name had no word "extension" in it and sat next to this source folder looking equally
legitimate — a low-tech operator had no reliable way to tell them apart. `prepare_client_extension.sh`
also drops a one-line `THIS_IS_THE_CLIENT_COPY.txt` inside every client folder it generates (which
client, when generated), so Finder shows at a glance that a given folder is a real client copy.
Backward compatibility: an older install whose folder is still named `extensions/{client_slug}/`
keeps working from that same folder — the script reuses it in place and never creates a second,
differently-named copy for the same client.

There is no Chrome Web Store submission today — one unpacked folder per client is the model, so install is a two-gesture flow off the client's dashboard page rather than a store install.

**One button (recommended):** open `http://127.0.0.1:17321/ui/{client_slug}/extension` and click the button. It reveals the `extensions/{client_slug}_extension/` folder in Finder/Explorer AND opens Chrome at `chrome://extensions` in the same click (the bridge runs on the human's own machine, so it can do this directly). Turn on **Developer mode**, then drag that folder onto the page — Chrome accepts a dropped folder as `Load unpacked`. The page turns green connected on its own when the extension checks in. A local-runtime agent (its own shell running on the human's machine) may trigger the same button itself via `POST /api/ui/{client_slug}/install-extension`, then poll `GET /status` until `extension_health.status` is recent (75-second grace window).

**Manual fallback:**

1. Open Chrome — the profile the human already has open and logged in for the first client; a second Chrome profile is only needed once a second client needs a different Facebook account.
2. Go to `chrome://extensions`.
3. Enable Developer Mode.
4. Click `Load unpacked`.
5. Select the absolute `extensions/{client_slug}_extension/` folder for that client — never this `solo-agency-collector/chrome-extension/` source folder.

Maintainers who are actively developing the extension may load this source folder (`solo-agency-collector/chrome-extension/`) in a separate development Chrome profile, but a normal agency setup must not use this folder — only the generated per-client copy. Loading the source folder still works for inspecting code and reloading after edits; the guard below just keeps it from quietly acting like a real client install while you do.

## Guard: Loaded The Wrong Folder?

`background.js` reads `client_binding.json` on every poll. `prepare_client_extension.sh` writes
that file into every client folder it generates, without exception — so if the file is missing
or unreadable, the loaded folder is either this source template (which ships no binding on
purpose) or a client copy that never finished generating. In that state:

- `background.js` **never polls the local bridge** — no `/status` call, no job fetch, nothing
  sent anywhere. It refuses before making any network call, not after.
- The popup's status box turns red and shows one plain-English sentence instead of the usual
  status lines.
- Once a real client copy (with its own `client_binding.json`) is loaded instead, everything
  behaves exactly as before — the guard only ever affects the no-binding case.

This is intentionally minimal: no new permissions, no change to any code path that already has a
working binding.

**The exact sentence an agent should relay to the human** when this guard state is hit (identical
to what the popup shows, `NO_CLIENT_BINDING_MESSAGE` in `background.js`):

> This is the SOURCE folder, not a client copy. Open the dashboard → Extension → click "Install extension" to install the correct {client_slug}_extension folder.

(`{client_slug}` is a placeholder — say the actual client's slug, e.g. `leadup_extension`.)

Covered by `solo-agency-collector/tests/test_client_binding_guard.js` (binding present → normal
poll reaches the bridge; binding missing or unreadable → guard status, zero bridge calls, message
present in both `background.js` state and the popup's rendered status box).

## Expected Flow

1. On a local runtime, the agent installs and starts the bridge itself (one plain-language safety line, one consent ask); on a remote/hosted runtime it hands the human the one-line command instead. Either way, once running, the bridge stays up via an OS-level autostart supervisor (LaunchAgent/systemd/Scheduled Task).
2. Extension detects the bridge while Chrome is open.
3. If the bridge reports an active collection window, the extension collects the job automatically.
4. Extension posts results to the bridge.
5. Bridge writes local monthly folders and marks the scheduled run complete.
6. AI agent reads the local files and continues the playbook workflow.

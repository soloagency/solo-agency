# Solo Agency Local Collector

This is the Chrome MV3 extension for the Solo Agency Local Collector.

The user installs it once in the Chrome profile that is already logged in to Facebook, LinkedIn, Reddit, Instagram, TikTok, or other private sources they want monitored.

## What It Does

- Polls `http://127.0.0.1:17321/status`.
- Polls every 5 seconds by default while enabled.
- Uses Chrome alarms as a wake-up fallback because Manifest V3 service workers can sleep.
- Fetches the current local job from the bridge.
- Opens configured source URLs in inactive tabs using the user's existing Chrome session.
- Closes collector-created tabs after scanning when configured.
- Waits 5 seconds between scroll/read actions by default.
- Uses 5 scrolls per private source by default, configurable up to 10.
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

## Install For Development

1. Open Chrome.
2. Go to `chrome://extensions`.
3. Enable Developer Mode.
4. Click `Load unpacked`.
5. Select `solo-agency-collector/chrome-extension`.

For public release, publish the extension through Chrome Web Store or provide a signed/internal extension package.

## Expected Flow

1. AI agent or OS startup starts the local bridge.
2. Extension detects the bridge while Chrome is open.
3. If the bridge reports an active collection window, the extension collects the job automatically.
4. Extension posts results to the bridge.
5. Bridge writes local monthly folders and marks the scheduled run complete.
6. AI agent reads the local files and continues the playbook workflow.

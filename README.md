<p align="center">
  <img src="VampYTDExtension/icons/logo.png" alt="VampYTD Logo" width="420">
</p>

<p align="center">
  <b>Windows-first, lightning-fast video downloader with pure Native Messaging browser integration.</b>
</p>

---

VampYTD is a Windows-first video downloader with a companion browser extension for Chromium and Firefox. It uses `yt-dlp` for extraction, FFmpeg for merging and trimming, and an optional FZF interface for choosing streams.

## Features at a glance

- ⚡ **Pure Native Messaging**: Launches the bridge process on demand with zero permanent background services, zero open network ports, and zero host permissions.
- 🎬 **In-Page YouTube Integration**: Embedded download buttons on both regular YouTube watch pages and vertical **YouTube Shorts** (`#actions` rail).
- 📌 **Active Video Detection Card**: Extension popup automatically detects the video playing in your active tab (YouTube / JioHotstar) with one-click downloading.
- 🔔 **Instant Visual Feedback**: Toolbar badge status indicators (`✓` / `ERR`) and non-intrusive floating in-page toast alerts on context-menu downloads.
- 🎛 **Format & Codec Preferences**: Quick-download presets (Interactive, Quick Max, 1080p, 4K) and codec selection (Auto, AV1, VP9, HEVC/H.265, H.264).
- 🍪 **Seamless Cookie Support**: Toggle site-specific authentication cookies (`cookies-yt.txt`, `cookies-jhs.txt`) directly from the popup for members-only or age-restricted content.
- ♿ **WCAG 2.1 AA Accessible UI**: Full keyboard arrow navigation, focus indicators, and ARIA roles with a sleek zero-footer dark design.

## Simple Windows installation

1. Download the Windows ZIP from [GitHub Releases](https://github.com/vampdued/vamp-ytd-extension/releases).
2. Extract the complete ZIP archive.
3. Double-click `Install-VampYTD.cmd`.
4. Activate the extension in your browser:
   - **Chrome / Edge / Brave / Vivaldi**: Open `chrome://extensions` (or `edge://extensions`), enable **Developer mode**, click **Load unpacked**, and paste/select the extension folder (path auto-copied to clipboard).
   - **Firefox**:
     - *Method 1 (All versions - Developer / Temporary)*: Open `about:debugging#/runtime/this-firefox`, click **Load Temporary Add-on...**, and select `manifest.json` inside `%LOCALAPPDATA%\VampYTD\VampYTDExtension-Firefox`.
     - *Method 2 (Dev Edition / ESR / Unbranded)*: Open `about:addons`, click the gear icon, click **Install Add-on From File...**, and select `%LOCALAPPDATA%\VampYTD\VampYTD-Extension.xpi` (requires `xpinstall.signatures.required=false` in `about:config`).

The double-click installer:

- installs missing `yt-dlp`, FFmpeg, Node.js, and FZF packages automatically via Windows Package Manager (`winget`);
- installs VampYTD under `%LOCALAPPDATA%\VampYTD`;
- packages `VampYTD-Extension.xpi` and `VampYTD-Extension.zip` for Firefox;
- adds the `ytd` command-line downloader to your user `PATH`;
- registers native messaging for Chrome, Edge, Chromium, Brave, Vivaldi, and Firefox;
- copies the unpacked extension folder path directly to your clipboard.

Administrator access is not required. Run `Install-VampYTD.cmd` again at any time to repair or update the installation.

To uninstall VampYTD, remove the unpacked browser extension and double-click `Uninstall-VampYTD.cmd`.

> Windows releases are currently unsigned. Windows may display a SmartScreen security warning on first launch.

## Requirements

The installer sets up dependencies automatically via Windows Package Manager (`winget`).

| Tool | Required | Auto-Installed | Purpose |
| --- | --- | --- | --- |
| `yt-dlp` | Yes | Yes (`yt-dlp.yt-dlp`) | Video extraction and download |
| FFmpeg | Yes | Yes (`Gyan.FFmpeg`) | Merging, thumbnails, and trimming |
| Node.js | Yes | Yes (`OpenJS.NodeJS.LTS`) | JavaScript runtime used by `yt-dlp` |
| FZF | Optional | Yes (`junegunn.fzf`) | Enhanced interactive format picker (falls back to numbered lists if missing) |

Keep `yt-dlp` current (`ytd -U`, `yt-dlp -U`, or `winget upgrade yt-dlp.yt-dlp`) as video site extractors update frequently. VampYTD also checks for updates periodically and displays a notice if `yt-dlp` is older than 30 days.

## Browser extension

The extension features:

- a right-click context menu to send videos or links directly to VampYTD with instant toolbar badge and floating toast confirmation;
- embedded **Download with VampYTD** buttons on YouTube video watch pages and YouTube Shorts;
- an **Active Video Card** in the popup to download the currently playing video with one click;
- customizable download mode (interactive vs quick presets), codec preferences (AV1, VP9, HEVC, H.264), and cookie toggles;
- a diagnostic dashboard verifying native messaging host connectivity, tool dependencies, and the target download folder.

Native messaging launches the bridge process on demand with zero permanent background tasks, startup shortcuts, or listening network ports.

The unpacked extension ID is pinned to `jjacbochmpbgpfpbfclmileocddkncgd` (Chromium) and `vampytd@vampdued.github.io` (Firefox).

## Command-line usage

```powershell
# Interactive format selection (uses FZF if available, or numbered fallback)
ytd "https://www.youtube.com/watch?v=..."

# Best available quality (quick mode)
ytd -q "URL"

# Resolution and codec constraints
ytd -q 1080p "URL"
ytd -q 4k,av1 "URL"
ytd -q 4k,hevc "URL"

# Lossless section trim
ytd -t 01:20 02:45 "URL"

# Use site-specific cookie file
ytd -c "URL"

# Update yt-dlp to the latest release
ytd -U
```

Downloads are saved under `%USERPROFILE%\Downloads\VampYTD`. Optional cookie files are stored under `%APPDATA%\vampytd`:

- `cookies-yt.txt`
- `cookies-jhs.txt`

## Using cookies

Cookies allow VampYTD to access videos that require your existing browser session, such as age-restricted, members-only, or authenticated content.

1. Sign in to the video site in your browser.
2. Export the site's cookies in Netscape `cookies.txt` format. A valid export commonly begins with `# Netscape HTTP Cookie File`.
3. Press `Win + R`, enter `%APPDATA%\vampytd`, and create the folder if it does not exist.
4. Save the exported file using the appropriate name:
   - YouTube: `cookies-yt.txt`
   - JioHotstar/Hotstar: `cookies-jhs.txt`
5. In the VampYTD extension popup, enable **Load cookies**, then start the download. For command-line downloads, add `-c`:

```powershell
ytd -c "https://www.youtube.com/watch?v=..."
```

VampYTD selects `cookies-jhs.txt` for JioHotstar/Hotstar URLs and `cookies-yt.txt` for YouTube and other sites. If authentication stops working, export a fresh cookie file because browser cookies can expire or be replaced.

> Cookie files contain sensitive session credentials. Do not share them or commit them to source control.

## Manual and source installation

To run setup manually from PowerShell:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
./deploy.ps1 -InstallDependencies
```

To specifically install FZF via winget during manual setup:

```powershell
./deploy.ps1 -InstallFZF
```

Building from source requires Go 1.22 or newer:

```powershell
go test ./...
go build -o ytd.exe ./cmd/ytd
go build -o bridge.exe ./cmd/bridge
```

## Project layout

```text
cmd/ytd/            downloader command-line application
cmd/bridge/         native messaging bridge for browser integration
VampYTDExtension/   Chromium & Firefox Manifest V3 browser extension
Install-VampYTD.cmd double-click Windows installer wrapper
Uninstall-VampYTD.cmd double-click Windows uninstaller wrapper
deploy.ps1          PowerShell installation and dependency logic
uninstall.ps1       PowerShell removal logic
.github/workflows/  CI and release pipelines
spec.md             downloader technical specification
```

## Release process

Windows AMD64 and ARM64 ZIP archives are generated automatically upon pushing a semantic version tag (e.g. `v1.4.0`).

VampYTD is released under the [MIT License](LICENSE).

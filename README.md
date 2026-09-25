<p align="center">
  <img src="VampYTDExtension/icons/logo.png" alt="VampYTD Logo" width="420">
</p>

<p align="center">
  <b>Windows-first, lightning-fast video downloader with pure Native Messaging browser integration.</b>
</p>

---

VampYTD is a Windows-first video downloader with a companion browser extension for Chromium and Firefox. It uses `yt-dlp` for extraction, FFmpeg for merging and trimming, and an optional FZF interface for choosing streams.

## Features at a glance

- ⚡ **Pure Native Messaging**: Launches the bridge process on demand with zero permanent background services, zero open network ports, and zero host permissions (`host_permissions: []` completely eliminated).
- 🎨 **Adaptive YouTube Theme Dashboard**: Minimal borderless UI with auto-switching light and dark themes matching YouTube's active page, WCAG 2.2 AA contrast compliance, and full keyboard accessibility across **Status**, **Settings**, and **YT Tools** tabs.
- 📌 **Hero Media Card**: Neutral elevated media card with a crisp crimson left signpost and one-click download CTA for detected YouTube and Hotstar streams.
- 🎛 **Segmented Codec & Mode Grid**: Balanced `Auto` hero pill + 2x2 codec matrix (`AV1`, `VP9`, `HEVC`, `H.264`), and left circular radio buttons (`○` / `◉`) for mode selection with zero text-indent jumping.
- 📋 **Structured Destination Folder**: Dedicated container with a docked one-click clipboard copy button and live feedback.
- 🎦 **Cinematic 2.39:1 Video Crop**: One-click in-player control bar button and `Alt+C` hotkey to crop black bars on widescreen videos with dynamic resize and fullscreen scaling.
- 💡 **Ambient Glow & Annotation Blocker**: Injected at `document_start` to hide distracting background glow effects and legacy annotations, cutting GPU/CPU rendering overhead.
- 📺 **Channel 'Videos' Tab First**: Automatically rewrites channel links and redirects channel featured pages straight to the `/videos` tab.
- 🎬 **In-Page YouTube Integration**: Embedded download buttons on both regular YouTube watch pages and vertical **YouTube Shorts** (`#actions` rail).
- 🔔 **Instant Visual Feedback**: Toolbar badge status indicators (`✓` / `ERR`) and non-intrusive floating in-page toast alerts on context-menu downloads.
- 🍪 **Seamless Cookie Support**: Toggle site-specific authentication cookies (`cookies-yt.txt`, `cookies-jhs.txt`) directly from the popup for members-only or age-restricted content.

## Simple Windows installation

1. Download the Windows ZIP from [GitHub Releases](https://github.com/vampdued/vamp-ytd-extension/releases).
2. Extract the complete ZIP archive.
3. Double-click `Install-VampYTD.cmd`.
4. Activate the extension in your browser:
   - **Chrome / Edge / Brave / Vivaldi**: Open `chrome://extensions` (or `edge://extensions` / `brave://extensions`), enable **Developer mode**, click **Load unpacked**, and select the extension folder (path auto-copied to clipboard).

The double-click installer:

- installs missing `yt-dlp`, FFmpeg, Node.js, and FZF packages automatically via Windows Package Manager (`winget`);
- installs pre-built VampYTD binaries under `%LOCALAPPDATA%\VampYTD`;
- adds the `ytd` command-line downloader to your user `PATH`;
- registers native messaging for Chrome, Edge, Chromium, Brave, and Vivaldi;
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

## Development workflow

To keep active local development cleanly separated from your normal installed version of VampYTD:

- **Switch to Dev Mode**:
  Double-click `dev.cmd` (or run `./setup-dev.ps1` in PowerShell).
  - Automatically compiles `ytd.exe` and `bridge.exe` directly in your Git workspace.
  - Registers the native messaging host to point directly to your workspace repository.
  - Copies the workspace extension path (`...\vamp-ytd-extension\VampYTDExtension`) to your clipboard.
  - You can now edit code, test changes live, and simply click **Reload 🔄** in `chrome://extensions`.

- **Switch back to Normal / Production**:
  Double-click `Install-VampYTD.cmd` (or run `./deploy.ps1`).
  - Restores your system to the clean production build under `%LOCALAPPDATA%\VampYTD`.
  - Re-registers native messaging back to the installed production binary.

### Git branching model

- **`main`**: Protected branch for stable production releases. Commits tagged with a version (e.g. `v1.4.0`) automatically build and publish release binaries.
- **`dev`**: Active development branch where ongoing features, experiments, and fixes are developed and tested before merging into `main`.

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

GitHub Releases are completely automated via GitHub Actions:
- Merging code into **`main`** with an updated version in `VampYTDExtension/manifest.json` automatically creates the release tag, builds Windows AMD64 and ARM64 ZIP archives with GoReleaser, generates checksums, and publishes the release.
- Releases can also be triggered manually using the **Run workflow** button in the GitHub Actions `Release` tab.

VampYTD is released under the [MIT License](LICENSE).

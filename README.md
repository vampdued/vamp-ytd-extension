# VampYTD

VampYTD is a Windows-first video downloader with a Chromium browser extension. It uses `yt-dlp` for extraction, FFmpeg for merging and trimming, and an optional FZF interface for choosing streams.

## Simple Windows installation

1. Download the Windows ZIP from [GitHub Releases](https://github.com/vampdued/vamp-ytd-extension/releases).
2. Extract the complete ZIP.
3. Double-click `Install-VampYTD.cmd`.
4. On the browser extensions page, enable **Developer mode**, choose **Load unpacked**, and select the folder opened by setup.

The installer:

- installs missing `yt-dlp`, FFmpeg, and Node.js packages through Windows Package Manager;
- installs VampYTD under `%LOCALAPPDATA%\VampYTD`;
- adds the command-line downloader to the user `PATH`;
- registers native messaging for Chrome, Edge, Chromium, Brave, and Vivaldi;
- opens the installed extension folder and copies its path to the clipboard.

Administrator access is not normally required. Run the installer again at any time to repair or update the installation.

To remove VampYTD, first remove the unpacked browser extension and then double-click `Uninstall-VampYTD.cmd`.

> Windows releases are currently unsigned. Windows may display a security warning until code signing is configured.

## Requirements

The double-click installer handles the required tools automatically when Windows Package Manager is available.

| Tool | Required | Purpose |
| --- | --- | --- |
| `yt-dlp` | Yes | Video extraction and download |
| FFmpeg | Yes | Merging, thumbnails, and trimming |
| Node.js | Yes | JavaScript runtime used by `yt-dlp` |
| FZF | No | Enhanced interactive format picker |

Keep `yt-dlp` current because video-site extractors change frequently.

## Browser extension

The extension offers:

- a right-click download menu;
- a VampYTD button on YouTube video pages;
- download mode, codec, and cookie settings;
- a diagnostic dashboard for native messaging, the optional localhost fallback, dependencies, and download location.

Chromium native messaging starts the bridge only when required. New installations do not need a permanent background process, startup shortcut, scheduled task, or listening network port. A secured localhost mode remains available only as a compatibility fallback for older installations.

The unpacked extension ID is pinned to `jjacbochmpbgpfpbfclmileocddkncgd`, and native-host manifests authorize that exact origin.

## Command-line usage

```powershell
# Interactive format selection
ytd "https://www.youtube.com/watch?v=..."

# Best available quality
ytd -q "URL"

# Resolution and codec constraints
ytd -q 1080p "URL"
ytd -q 4k,av1 "URL"

# Lossless section trim
ytd -t 01:20 02:45 "URL"

# Use a site-specific cookie file
ytd -c "URL"
```

Downloads are saved under the current user's `Downloads\VampYTD` directory. Optional cookie files are read from the operating system's user configuration directory under `vampytd`:

- `cookies-yt.txt`
- `cookies-jhs.txt`

## Manual and source installation

To install without automatic dependency setup:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
./deploy.ps1
```

Use `-InstallDependencies` to request dependency installation and `-NoLaunch` to avoid opening the browser and extension folder.

Go 1.26.2 or newer is required for source builds:

```powershell
go test ./...
go build -o ytd.exe ./cmd/ytd
go build -o bridge.exe ./cmd/bridge
```

The Go commands remain portable, and CI continues to test Windows, Linux, and macOS. User-friendly Linux and macOS packaging is intentionally deferred; the current public release is Windows-only.

## Project layout

```text
cmd/ytd/            downloader command
cmd/bridge/         native host and compatibility bridge
VampYTDExtension/   Chromium extension
deploy.ps1          Windows install and repair logic
uninstall.ps1       Windows removal logic
.github/workflows/  continuous integration and releases
spec.md             downloader behavior specification
```

## Release process

Windows AMD64 and ARM64 ZIP archives are created from semantic-version tags such as `v1.3.0`. The tag must match the extension version. Releases include SHA-256 checksums and GitHub artifact attestations.

VampYTD is available under the [MIT License](LICENSE). Windows code signing is recommended but is not yet configured.

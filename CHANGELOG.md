# Changelog

## 1.5.0 - Standalone Setup Executable & Streamlined Distribution

- **Standalone Windows Installer (`VampYTD-Setup.exe`)**:
  - Self-contained executable embedding `ytd.exe`, `bridge.exe`, and packed Chromium extension files with zero external dependencies.
  - Colored step-by-step progress, automatic process termination, registry configuration, user PATH update, and dependency checking.
  - Automatically copies extension path to clipboard, offers to open `chrome://extensions`, and opens the extension folder in Explorer.
  - Built-in `--uninstall` mode and automatic creation of `%LOCALAPPDATA%\VampYTD\uninstall.exe`.
- **1-Liner PowerShell Web Installer (`install.ps1`)**:
  - Direct execution via `irm https://raw.githubusercontent.com/vampdued/vamp-ytd-extension/main/install.ps1 | iex`.
- **Developer vs. Release Separation**:
  - Added dedicated `build.ps1` and `build.cmd` scripts for dev mode.
  - Release archives stripped of developer-only tools and scripts.
- **Automated GitHub Release Pipeline**:
  - Automatic version tag detection, build, and release asset generation on push to `main`.

## 1.4.0 - Pure Native Overhaul & YouTube Power Tools

- **Branding & Visual Identity**:
  - Official new high-definition brand logo and icons across browser extensions and documentation.
- **Pure Native Messaging Architecture**:
  - Removed localhost HTTP server fallback and port 8080 entirely.
  - Removed all browser `host_permissions` (`host_permissions: []`), ensuring zero unnecessary permissions.
  - Pure stdio-based native host communication for Chromium and Firefox.
- **Extension UI/UX Overhaul**:
  - Redesigned popup with adaptive light/dark theme matching YouTube's active page (`data-theme="dark"`).
  - Elevated neutral Hero Media Card with crimson left signpost and one-click download CTA.
  - Balanced Preferred Codec layout: `Auto` full-width hero pill on top + 2x2 grid (`AV1`, `VP9`, `HEVC`, `H264`) with centered labels.
  - Native circular radio indicators (`○` / `◉`) in Download Mode for perfect column alignment.
  - Structured Destination Folder input box with docked clipboard copy button.
  - Balanced 2-column dependencies grid with full-width FZF status row.
  - 44px full-row touch targets on toggle rows and YouTube native neutral switches.
  - Full WCAG 2.2 AA contrast compliance and keyboard navigation.
- **Integrated YouTube Power Tools**:
  - Added 2.39:1 Cinematic Crop engine with in-player control bar button and `Alt+C` keyboard shortcut.
  - Added Ambient Glow & Annotation Blocker running at `document_start` to save GPU/CPU cycles.
  - Added automatic channel `/videos` tab redirection.
  - Added independent toggle switches in the new "YT Tools" popup tab.
- **In-Page YouTube & Shorts Integration**:
  - Embedded download buttons on watch pages and YouTube Shorts action rail.
- **Bug Fixes**:
  - Fixed DOMException on in-player crop button insertion when fullscreen button is nested in sub-wrappers (`content.js`).
  - Fixed duplicate button creation during SPA navigation (`yt-navigate-finish`).

## 1.3.0 - Windows release candidate

- Added a double-click Windows installer with dependency setup and repair support.
- Replaced the permanent background bridge with on-demand native messaging.
- Added a popup diagnostics dashboard and clearer page-button feedback.
- Fixed Windows URL argument forwarding and YouTube page-button clicks.
- Added automated Windows install/uninstall testing and cross-platform Go checks.
- Added Windows AMD64 and ARM64 release archives, checksums, and attestations.
- Licensed the project under the MIT License.

## 1.0.0

- Initial command-line downloader and browser integration.

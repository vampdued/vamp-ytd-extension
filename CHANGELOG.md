# Changelog

## [Unreleased]

### Changed
- Build scripts consolidated: `build.ps1 -Dev` replaces `setup-dev.ps1` (deleted); `dev.cmd` now calls `build.ps1 -Dev`.
- `deploy.ps1 -InstallDependencies` no longer installs the optional FZF (use `-InstallFZF` explicitly); Node.js is only suggested when no JS runtime exists.
- `ytd` accepts `--dir`/`-d` for the download directory (`VAMPYTD_DIR` env also works); cookie file paths overridable via `VAMPYTD_COOKIES_YT`/`VAMPYTD_COOKIES_JHS`; `ytd-error.log` now lands in the download dir instead of the CWD.
- `ytd` arg parsing refactored into testable `parseArgsFrom` + `parseQuickOptions` with table-driven tests; quick-mode resolution parsing fixed (`TrimSuffix` instead of `TrimRight`).
- Removed dead `VampYTDBridge` scheduled-task cleanup from `deploy.ps1`/`uninstall.ps1`.

### Added
- CI now runs `golangci-lint`; README notes tests are Windows-only by nature.
- README documents the code-signing path (removes SmartScreen warning) and the `ExtensionInstallForcelist` true-1-click path for managed/enterprise installs.

## 1.5.2 — 2026-10-06

### Fixed
- Web installer (`install.ps1`) now downloads `checksums.txt` and verifies the SHA-256 of `VampYTD-Setup.exe` before running it.
- Bridge native-host registration no longer aborts on the first browser failure — it registers every browser it can and fails only if none succeeded.
- Installer PATH update now handles usernames with apostrophes, and only terminates `bridge.exe` processes from our own install directory.
- Popup diagnostics accept any JS runtime (Deno, Bun, or Node.js); the bridge reports the detected `jsRuntime` and the popup shows its name.
- `deploy.ps1` no longer forces a Node.js install — Node is suggested only when no JS runtime exists at all.
- Release workflow no longer silently re-publishes assets on merges without a version bump.
- Removed dead `.goreleaser.yaml`; corrected README release/Go-version claims.

### Removed
- All Firefox/Mozilla support (Chromium-only now): `manifest.firefox.json`, gecko settings, Mozilla native-host manifest and registry keys, and related tests/scripts/docs.

### Changed
- Installer no longer reimplements native-host registration — it runs the extracted `bridge.exe --install-native`, so the manifest + registry logic lives in exactly one place.
- New shared `internal/pathutil` package used by both `bridge` and `ytd` (was copy-pasted in each); PATH re-scan is cached for 60s so popup diagnostics no longer walk the filesystem on every open.

## 1.5.1 — 2026-09-29 — JS Runtime Auto-Detection

- **Auto-detect JavaScript runtime for yt-dlp**:
  - Removed hard-coded dependency on Node.js (`--js-runtime node`).
  - At startup, `ytd` now probes `deno → bun → node` in PATH and passes the first found runtime to yt-dlp via `--js-runtime`.
  - If none of the three runtimes are found, the flag is omitted and yt-dlp surfaces its own diagnostic error.
  - Eliminates the `"Node.js is required… not found in PATH"` crash for users with Deno or Bun installed.

## 1.5.0 — 2026-09-25 — Standalone Setup Executable & Streamlined Distribution

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

## 1.4.0 — 2026-09-25 — Pure Native Overhaul & YouTube Power Tools

- **Branding & Visual Identity**:
  - Official new high-definition brand logo and icons across browser extensions and documentation.
- **Pure Native Messaging Architecture**:
  - Removed localhost HTTP server fallback and port 8080 entirely.
  - Removed all browser `host_permissions` (`host_permissions: []`), ensuring zero unnecessary permissions.
  - Pure stdio-based native host communication for Chromium browsers.
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

## 1.3.0 — 2026-07-20 — Windows release candidate

- Added a double-click Windows installer with dependency setup and repair support.
- Replaced the permanent background bridge with on-demand native messaging.
- Added a popup diagnostics dashboard and clearer page-button feedback.
- Fixed Windows URL argument forwarding and YouTube page-button clicks.
- Added automated Windows install/uninstall testing and cross-platform Go checks.
- Added Windows AMD64 and ARM64 release archives, checksums, and attestations.
- Licensed the project under the MIT License.

## 1.0.0 — 2026-06-08

- Initial command-line downloader and browser integration.

# Changelog

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

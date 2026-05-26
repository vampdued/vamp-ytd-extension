# 🦇 VampYTD

A high-performance, developer-centric CLI video downloader and browser bridge for YouTube and hotstar/jiohotstar. Built in Go, VampYTD seamlessly wraps `yt-dlp`, `ffmpeg`, and `fzf` to provide a keyboard-driven, lightning-fast downloading experience with intelligent automatic format selection, media merging, and metadata embedding.

> [!NOTE]
> VampYTD has been completely updated to be fully portable across systems with dynamic directory resolution, eliminating any hardcoded user home path references.

---

## 🚀 Key Features

* **🧠 Smart Format Sorting & Quality Optimization**: Parses metadata and ranks formats based on **Resolution** → **Codec Efficiency** (AV1 > VP9 > HEVC > AVC) → **Audio Quality** → **Largest Bitrate/Filesize**. In quick modes, it dynamically compares resolution and bitrate to select the absolute highest quality track.
* **⌨️ Interactive Keyboard Picker**: Leverages `fzf` for smooth, lightning-fast multi-select stream picker directly in your terminal.
* **📦 Automatic Merging & Tagging**: Uses `ffmpeg` to merge high-quality video and audio into clean `.mkv` files, fully embedding chapters, thumbnails (converted to `.png`), and complete video metadata.
* **⚡ Quick Mode & Codec Mapping**: Instant, hands-off downloads for best quality, specific resolutions, or preferred video codecs (AV1/VP9/H264) using fuzzy mappings (e.g. `av1` -> `av01.*`, `hevc`/`h265` -> `h265`, `h264` -> `avc1.*`).
* **✂️ Trim Mode**: High-speed lossless section trimming via stream copying (e.g., `ytd -t 01:20 02:45 "URL"`).
* **🌐 Double-Layer SPA Extension**: Modern manifest-v3 chrome extension for one-click downloading featuring a slate-minimal popup panel and robust YouTube Single Page Application injection (MutationObserver + `yt-navigate-finish` triggers).
* **🛡️ Self-Generating Systemd Service**: Complete daemon automation via `./bridge --install` flag which auto-generates, registers, enables, and launches a systemd user service.

---

## 🛠️ Prerequisites

Ensure you have the following CLI tools installed and available in your system `PATH`:

| Tool | Recommended Install Command (macOS/Linux) | Recommended Install Command (Windows/Scoop) |
| :--- | :--- | :--- |
| **[yt-dlp](https://github.com/yt-dlp/yt-dlp)** | `brew install yt-dlp` or `sudo apt install yt-dlp` | `scoop install yt-dlp` |
| **[ffmpeg](https://ffmpeg.org/)** | `brew install ffmpeg` or `sudo apt install ffmpeg` | `scoop install ffmpeg` |
| **[fzf](https://github.com/junegunn/fzf)** | `brew install fzf` or `sudo apt install fzf` | `scoop install fzf` |

---

## 📦 Building & Installation

VampYTD is written in clean, modern Go. You can build it locally from the source files.

### 1. Build the Binaries

From the repository root, build the optimized Go executables:

```bash
# Build the CLI downloader
go build -o ytd main.go

# Build the browser companion bridge
go build -o bridge bridge.go
```

### 2. Linux/macOS Installation & systemd Service Setup

Make the compiled binaries executable and symlink them to your local user binary directory:

```bash
chmod +x ytd bridge
ln -sf $(pwd)/ytd ~/.local/bin/ytd
ln -sf $(pwd)/bridge ~/.local/bin/bridge
```

To configure the browser companion bridge to start automatically on user login and run silently in the background:

```bash
# Run the automated systemd installer built directly into the bridge executable
./bridge --install   # or ./bridge -i
```

This auto-generates your systemd user configuration (`~/.config/systemd/user/vampytd-bridge.service`), reloads the user systemd daemon, enables it on boot, and starts it instantly!

---

## 📖 CLI Usage Guides

### `ytd` — The Core Downloader

```bash
# 1. Interactive Selection Mode (launches fzf menu with custom styled streams)
ytd "https://www.youtube.com/watch?v=..."

# 2. Quick Mode (instantly download best resolution without manual selection)
ytd -q "URL"

# 3. Quick Mode with Resolution Constraints (limit maximum height)
ytd -q 1080p "URL"
ytd -q 4k,av1 "URL"

# 4. Trim Mode (lossless fast stream copy for specific duration)
ytd -t 01:20 02:45 "URL"

# 5. Enable Cookies (automatically loaded from config based on URL match)
ytd -c "URL"
```

### `bridge` — Browser Extension Background Server

Starts a lightweight background HTTP server listening on port `8080` to bridge browser events to your terminal downloader.

```bash
# 1. Register and enable the systemd daemon (recommended)
./bridge --install    # or ./bridge -i

# 2. Start the bridge manually in the current terminal window
./bridge
```

---

## ⚙️ Configuration & Paths

VampYTD uses dynamic home-directory resolution to determine the following local paths:

* **Downloads Target**: `~/Downloads/VampYTD/`
* **YouTube Cookie Storage**: `~/.config/vampytd/cookies-yt.txt`
* **Hotstar Cookie Storage**: `~/.config/vampytd/cookies-jhs.txt`

> [!TIP]
> Place your browser-exported cookies in the `~/.config/vampytd/` folder. When running with the `-c` flag, VampYTD will automatically load the appropriate cookie file for YouTube or Hotstar/JioHotstar!

---

## 🧩 Browser Extension Setup

VampYTD comes with a modern manifest-v3 chrome extension for one-click downloading.

1. Open your Chromium-based browser (Chrome, Edge, Brave, etc.) and navigate to the Extensions page (`chrome://extensions` or `edge://extensions`).
2. Toggle **Developer mode** in the upper right.
3. Click **Load unpacked** in the top-left corner.
4. Select the `VampYTDExtension` directory within this repository.
5. Click the extension toolbar icon to view the dynamic slate-minimal console. You can toggle **Download Mode**, **Preferred Codec (AV1/VP9/H264)**, **Cookies**, and verify the live connection indicator dot (green when bridge is online, amber when bridge is offline).
6. Click the watch page download button on YouTube to immediately trigger high-speed optimized CLI downloads!

---

## 🔍 Technical Specification

For the full architectural details, API specifications, output formatting contracts, and program constraints, see [spec.md](./spec.md).

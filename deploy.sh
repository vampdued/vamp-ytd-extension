#!/usr/bin/env bash

# VampYTD Automation Deployment Script
# Compiles CLI and bridge code, syncs files to active runtime path, restarts daemon, and verifies health.

set -euo pipefail

# Define operational paths
WORKSPACE_DIR="/home/vampdued/apps/vamp-ytd-extension"
RUN_DIR="/home/vampdued/apps/vamp-ytd-extension/run"

# Define operational host and port from environment (with defaults)
PORT="${VAMPYTD_PORT:-${PORT:-8080}}"
HOST="${VAMPYTD_HOST:-${HOST:-localhost}}"

echo "🚀 [VampYTD] Starting automated build and deployment..."

# 1. Compile the Golang Binaries
echo "📦 [1/5] Compiling Go binaries (ytd and bridge)..."
if go build -C "${WORKSPACE_DIR}" -o ytd main.go && go build -C "${WORKSPACE_DIR}" -o bridge bridge.go; then
    echo "✅ [1/5] Compilation succeeded!"
else
    echo "❌ [1/5] Compilation failed! Check Go environment."
    exit 1
fi

# 2. Sync Binaries to Active Runtime Folder & Create Symlinks
echo "🔄 [2/5] Syncing compiled binaries to operational path..."
mkdir -p "${RUN_DIR}"
cp -f "${WORKSPACE_DIR}/ytd" "${RUN_DIR}/ytd"
cp -f "${WORKSPACE_DIR}/bridge" "${RUN_DIR}/bridge"
chmod +x "${RUN_DIR}/ytd" "${RUN_DIR}/bridge"

# Symlink to local bin for terminal availability
mkdir -p "${HOME}/.local/bin"
ln -sf "${RUN_DIR}/ytd" "${HOME}/.local/bin/ytd"
ln -sf "${RUN_DIR}/bridge" "${HOME}/.local/bin/bridge"
echo "✅ [2/5] Binaries synchronized and symlinked to ~/.local/bin."

# 3. Sync Chrome Extension Assets
echo "🎨 [3/5] Syncing extension assets..."
mkdir -p "${RUN_DIR}/VampYTDExtension"
cp -rf "${WORKSPACE_DIR}/VampYTDExtension/"* "${RUN_DIR}/VampYTDExtension/"
echo "✅ [3/5] Extension assets synchronized."

# 4. Restart or Install the systemd user service
echo "⚡ [4/5] Checking vampytd-bridge.service status..."
SERVICE_FILE="${HOME}/.config/systemd/user/vampytd-bridge.service"

if [ ! -f "${SERVICE_FILE}" ]; then
    echo "⚠️ [VampYTD] vampytd-bridge.service was not found at ${SERVICE_FILE}!"
    
    # Prompt the user for confirmation (redirect stdin from /dev/tty for interactive shell environments)
    if read -rp "❓ Would you like to automatically install VampYTD Bridge as a systemd user service? [y/N]: " response < /dev/tty; then
        case "$response" in
            [yY]|[yY][eE][sS])
                echo "⚙️ Creating systemd user configuration directory..."
                mkdir -p "$(dirname "${SERVICE_FILE}")"
                
                echo "⚙️ Writing service file to ${SERVICE_FILE}..."
                
                # Construct environment lines to pass GUI display vars to the systemd service
                ENV_LINES=""
                if [ -n "${DISPLAY:-}" ]; then
                    ENV_LINES="${ENV_LINES}Environment=DISPLAY=${DISPLAY}\n"
                fi
                if [ -n "${WAYLAND_DISPLAY:-}" ]; then
                    ENV_LINES="${ENV_LINES}Environment=WAYLAND_DISPLAY=${WAYLAND_DISPLAY}\n"
                fi
                if [ -n "${XDG_RUNTIME_DIR:-}" ]; then
                    ENV_LINES="${ENV_LINES}Environment=XDG_RUNTIME_DIR=${XDG_RUNTIME_DIR}\n"
                fi
                if [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
                    ENV_LINES="${ENV_LINES}Environment=DBUS_SESSION_BUS_ADDRESS=${DBUS_SESSION_BUS_ADDRESS}\n"
                fi

                cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=VampYTD Bridge Server
After=network.target

[Service]
Type=simple
ExecStart=${RUN_DIR}/bridge
$(printf "%b" "${ENV_LINES}")Restart=always
RestartSec=3
WorkingDirectory=${RUN_DIR}

[Install]
WantedBy=default.target
EOF
                
                echo "⚙️ Reloading systemd user daemon and enabling service..."
                systemctl --user daemon-reload
                systemctl --user enable vampytd-bridge.service
                echo "✅ systemd user service successfully installed and enabled!"
                ;;
            *)
                echo "⚠️ Skipping systemd service installation."
                echo "👉 To run the bridge manually, execute: ${RUN_DIR}/bridge"
                ;;
        esac
    else
        echo "⚠️ Non-interactive terminal detected. Skipping systemd service installation."
    fi
fi

# Now check if the service is active/enabled to restart it
if [ -f "${SERVICE_FILE}" ]; then
    echo "🔄 Restarting vampytd-bridge.service daemon..."
    if systemctl --user restart vampytd-bridge.service; then
        echo "✅ [4/5] systemd service restarted successfully!"
    else
        echo "❌ [4/5] Failed to restart systemd user service."
        exit 1
    fi
else
    echo "⚠️ [4/5] systemd service is not installed. Skipping daemon restart."
    echo "👉 You must run the bridge binary manually: ${RUN_DIR}/bridge"
fi

# 5. Live Endpoint Health Check Verification
echo "🔍 [5/5] Performing live endpoint health check..."
HEALTH_HOST="127.0.0.1"
if [ "${HOST}" != "0.0.0.0" ] && [ "${HOST}" != "localhost" ]; then
    HEALTH_HOST="${HOST}"
fi
HEALTH_URL="http://${HEALTH_HOST}:${PORT}/"
MAX_ATTEMPTS=5
SUCCESS=0

# Give the server a small moment to bind, then check
sleep 0.5

for ((i=1; i<=MAX_ATTEMPTS; i++)); do
    echo "   - Attempt $i/$MAX_ATTEMPTS: Pinging $HEALTH_URL..."
    if curl -s -f -o /dev/null "${HEALTH_URL}"; then
        SUCCESS=1
        break
    fi
    sleep 1
done

if [ "$SUCCESS" -eq 1 ]; then
    echo "✅ [5/5] App is alive and responding!"
    if [ "${HOST}" = "0.0.0.0" ]; then
        echo "🎉 [VampYTD] Deployment complete! Bridge is available at http://localhost:${PORT} and on your local network."
    else
        echo "🎉 [VampYTD] Deployment complete! Bridge is available at http://${HOST}:${PORT}"
    fi
else
    echo "❌ [5/5] Health check failed! App is not responding on ${HEALTH_HOST}:${PORT}."
    if [ -f "${SERVICE_FILE}" ]; then
        echo "⚠️ Fetching systemd service status and logs:"
        systemctl --user status vampytd-bridge.service || true
        echo "⚠️ Recent logs from journalctl:"
        journalctl --user -u vampytd-bridge.service -n 25 --no-pager || true
    else
        echo "👉 Please verify the bridge binary is running manually."
    fi
    exit 1
fi

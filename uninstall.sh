#!/usr/bin/env bash

# VampYTD Systemd Service and Binaries Uninstaller
# Stops, disables, and deletes the systemd user service and deployed binaries cleanly.

set -euo pipefail

SERVICE_FILE="${HOME}/.config/systemd/user/vampytd-bridge.service"
RUN_DIR="/home/vampdued/apps/vamp-ytd-extension/run"

echo "🗑️ [VampYTD] Starting uninstallation..."

# 1. Stop and disable systemd service
if [ -f "${SERVICE_FILE}" ]; then
    echo "🛑 Stopping vampytd-bridge.service..."
    systemctl --user stop vampytd-bridge.service || true
    
    echo "🚫 Disabling vampytd-bridge.service..."
    systemctl --user disable vampytd-bridge.service || true
    
    echo "🔥 Removing service configuration file..."
    rm -f "${SERVICE_FILE}"
    
    echo "⚙️ Reloading systemd user daemon..."
    systemctl --user daemon-reload
    
    echo "✅ systemd user service has been successfully uninstalled!"
else
    echo "ℹ️ No systemd user service was found at ${SERVICE_FILE}."
fi

# 2. Remove symlinks from ~/.local/bin
echo "🔗 Removing symlinks from local bin..."
rm -f "${HOME}/.local/bin/ytd"
rm -f "${HOME}/.local/bin/bridge"

# 3. Clean up active runtime folder
if [ -d "${RUN_DIR}" ]; then
    echo "🧹 Removing active runtime folder at ${RUN_DIR}..."
    rm -rf "${RUN_DIR}"
fi

echo "🎉 [VampYTD] Uninstallation complete!"

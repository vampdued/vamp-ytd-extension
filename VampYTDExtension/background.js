const NATIVE_HOST = 'com.vampytd.bridge';
const NATIVE_TIMEOUT_MS = 10_000;

// Wraps chrome.runtime.sendNativeMessage with a hard timeout so the service
// worker never silently hangs waiting for a response that will never arrive.
function sendNative(payload) {
    return new Promise((resolve, reject) => {
        let settled = false;

        const timer = setTimeout(() => {
            if (settled) return;
            settled = true;
            reject(new Error('Native host timed out'));
        }, NATIVE_TIMEOUT_MS);

        chrome.runtime.sendNativeMessage(NATIVE_HOST, payload, (response) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);

            if (chrome.runtime.lastError) {
                reject(new Error(chrome.runtime.lastError.message));
                return;
            }
            if (!response || !response.ok) {
                reject(new Error(response?.error || 'Native host returned no response'));
                return;
            }
            resolve(response);
        });
    });
}

// Temporary badge feedback on toolbar icon for context menu or quick actions.
function showBadgeFeedback(ok) {
    if (!chrome.action?.setBadgeText) return;
    const text = ok ? '✓' : 'ERR';
    const color = ok ? '#10b981' : '#ef4444';
    chrome.action.setBadgeText({ text });
    chrome.action.setBadgeBackgroundColor({ color });
    setTimeout(() => {
        chrome.action.setBadgeText({ text: '' });
    }, 3000);
}

// Reads stored settings then dispatches the download payload to the native host.
function sendToBridge(url, isPlaylist = false, items = '') {
    return new Promise((resolve) => {
        chrome.storage.local.get({
            downloadMode: 'interactive',
            preferredCodec: 'auto',
            enableCookies: false
        }, (stored) => {
            const payload = {
                url,
                mode: stored.downloadMode,
                codec: stored.preferredCodec,
                cookies: stored.enableCookies
            };
            if (isPlaylist) {
                payload.playlist = true;
                if (items) {
                    payload.items = items;
                }
            }

            sendNative(payload)
                .then(() => {
                    console.log('[VampYTD] Download dispatched via native messaging:', payload);
                    resolve({ ok: true, transport: 'native' });
                })
                .catch((err) => {
                    console.error('[VampYTD] Native messaging failed:', err.message);
                    resolve({ ok: false, error: err.message });
                });
        });
    });
}

// Pings the native host and returns a unified diagnostics object.
async function getDiagnostics() {
    try {
        const result = await sendNative({ action: 'ping' });
        return {
            ok: true,
            native: { ok: true },
            details: result
        };
    } catch (err) {
        return {
            ok: false,
            native: { ok: false, error: err.message },
            details: null
        };
    }
}

// --- Background Context Menu Setup ---
chrome.runtime.onInstalled.addListener(() => {
    // Right-click on a video link
    chrome.contextMenus.create({
        id: 'vampytd-download-link',
        title: 'Download Link with VampYTD',
        contexts: ['link'],
        targetUrlPatterns: [
            '*://*.youtube.com/watch*',
            '*://*.youtube.com/v/*',
            '*://*.youtube.com/shorts/*',
            '*://youtu.be/*',
            '*://*.hotstar.com/*',
            '*://*.jiohotstar.com/*'
        ]
    });

    // Right-click on a playlist link
    chrome.contextMenus.create({
        id: 'vampytd-download-playlist-link',
        title: 'Download Playlist with VampYTD',
        contexts: ['link'],
        targetUrlPatterns: [
            '*://*.youtube.com/playlist*',
            '*://*.youtube.com/watch*list=*'
        ]
    });

    // Right-click on the active page itself (video)
    chrome.contextMenus.create({
        id: 'vampytd-download-page',
        title: 'Download Video on Page',
        contexts: ['page'],
        documentUrlPatterns: [
            '*://*.youtube.com/watch*',
            '*://*.youtube.com/shorts*',
            '*://*.hotstar.com/*',
            '*://*.jiohotstar.com/*'
        ]
    });

    // Right-click on the active page itself (playlist page)
    chrome.contextMenus.create({
        id: 'vampytd-download-playlist-page',
        title: 'Download Playlist on Page',
        contexts: ['page'],
        documentUrlPatterns: [
            '*://*.youtube.com/playlist*',
            '*://*.youtube.com/watch*list=*'
        ]
    });
});

// --- Context Menu Click Handler ---
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
    const isPlaylistMenu = info.menuItemId === 'vampytd-download-playlist-link' || info.menuItemId === 'vampytd-download-playlist-page';
    const isLinkMenu = info.menuItemId === 'vampytd-download-link' || info.menuItemId === 'vampytd-download-playlist-link';

    const targetUrl = isLinkMenu ? info.linkUrl : (info.pageUrl || tab?.url);

    if (!targetUrl) return;

    const res = await sendToBridge(targetUrl, isPlaylistMenu);
    showBadgeFeedback(res.ok);

    if (tab?.id) {
        chrome.tabs.sendMessage(tab.id, {
            action: 'toast',
            ok: res.ok,
            message: res.ok ? (isPlaylistMenu ? 'Playlist sent to VampYTD' : 'Sent to VampYTD') : (res.error || 'Bridge offline')
        }).catch(() => {});
    }
});

// --- Message Listener (from content script & popup) ---
chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (message.action === 'diagnostics') {
        getDiagnostics().then(sendResponse);
        return true; // keep port open for async response
    }
    if (message.action === 'ping') {
        getDiagnostics().then((result) =>
            sendResponse({ ok: result.ok, transport: result.native.ok ? 'native' : undefined })
        );
        return true;
    }
    if (message.action === 'download' && message.url) {
        sendToBridge(message.url, Boolean(message.playlist), message.items || '').then(sendResponse);
        return true;
    }
});

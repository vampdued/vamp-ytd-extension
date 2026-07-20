const NATIVE_HOST = 'com.vampytd.bridge';

function sendNative(payload) {
    return new Promise((resolve, reject) => {
        chrome.runtime.sendNativeMessage(NATIVE_HOST, payload, (response) => {
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

async function sendHttpFallback(payload) {
    const response = await fetch('http://localhost:8080/download', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
    });
    if (!response.ok) {
        throw new Error((await response.text()) || `Bridge returned status ${response.status}`);
    }
    return { ok: true, transport: 'http' };
}

async function getHttpDiagnostics() {
    const response = await fetch('http://localhost:8080/diagnostics');
    if (!response.ok) {
        throw new Error(`Bridge returned status ${response.status}`);
    }
    return response.json();
}

async function getDiagnostics() {
    const [nativeResult, httpResult] = await Promise.allSettled([
        sendNative({ action: 'ping' }),
        getHttpDiagnostics()
    ]);

    const nativeOK = nativeResult.status === 'fulfilled';
    const httpOK = httpResult.status === 'fulfilled' && Boolean(httpResult.value?.ok);
    const details = nativeOK ? nativeResult.value : (httpOK ? httpResult.value : null);

    return {
        ok: nativeOK || httpOK,
        native: nativeOK
            ? { ok: true }
            : { ok: false, error: nativeResult.reason?.message || 'Unavailable' },
        http: httpOK
            ? { ok: true }
            : { ok: false, error: httpResult.reason?.message || 'Unavailable' },
        details
    };
}

// Forward URL and stored settings to the native host. HTTP remains as a
// migration fallback for existing installations.
function sendToBridge(url) {
    return new Promise((resolve) => {
        chrome.storage.local.get({
            downloadMode: 'interactive',
            preferredCodec: 'auto',
            enableCookies: false
        }, (items) => {
            const payload = {
                url: url,
                mode: items.downloadMode,
                codec: items.preferredCodec,
                cookies: items.enableCookies
            };

            sendNative(payload)
                .then(() => {
                    console.log('Download sent through native messaging:', payload);
                    resolve({ ok: true, transport: 'native' });
                })
                .catch(nativeError => {
                    console.warn('Native messaging unavailable; trying localhost bridge:', nativeError);
                    sendHttpFallback(payload)
                        .then(resolve)
                        .catch(httpError => {
                            console.error('Both VampYTD transports failed:', httpError);
                            resolve({
                                ok: false,
                                error: `Native: ${nativeError.message}; HTTP: ${httpError.message}`
                            });
                        });
                });
        });
    });
}

// --- Background Context Menu Setup ---
chrome.runtime.onInstalled.addListener(() => {
    // 1. Right-click on a video link
    chrome.contextMenus.create({
        id: "vampytd-download-link",
        title: "Download Link with VampYTD",
        contexts: ["link"],
        targetUrlPatterns: [
            "*://*.youtube.com/watch*",
            "*://*.youtube.com/v/*",
            "*://*.youtube.com/shorts/*",
            "*://youtu.be/*",
            "*://*.hotstar.com/*",
            "*://*.jiohotstar.com/*"
        ]
    });

    // 2. Right-click on the active page itself
    chrome.contextMenus.create({
        id: "vampytd-download-page",
        title: "Download Video on Page",
        contexts: ["page"],
        documentUrlPatterns: [
            "*://*.youtube.com/watch*",
            "*://*.youtube.com/shorts*",
            "*://*.hotstar.com/*",
            "*://*.jiohotstar.com/*"
        ]
    });
});

// --- Context Menu Interactions Listener ---
chrome.contextMenus.onClicked.addListener((info, tab) => {
    if (info.menuItemId === "vampytd-download-link" && info.linkUrl) {
        sendToBridge(info.linkUrl);
    } else if (info.menuItemId === "vampytd-download-page") {
        sendToBridge(info.pageUrl || tab.url);
    }
});

// --- Injected Content Action Message Listener ---
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'diagnostics') {
        getDiagnostics().then(sendResponse);
        return true;
    }
    if (message.action === 'ping') {
        getDiagnostics().then(result => sendResponse({
            ok: result.ok,
            transport: result.native.ok ? 'native' : (result.http.ok ? 'http' : undefined)
        }));
        return true;
    }
    if (message.action === 'download' && message.url) {
        sendToBridge(message.url).then(sendResponse);
        return true;
    }
});

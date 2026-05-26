// --- Utility: Forward URL & Stored Settings to Bridge Daemon ---
function sendToBridge(url) {
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

        fetch('http://localhost:8080/download', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload)
        })
        .then(response => {
            if (response.ok) {
                console.log("Successfully sent download command to bridge daemon:", payload);
            } else {
                console.error("Bridge daemon returned error status:", response.status);
            }
        })
        .catch(error => {
            console.error("Connection failed. Ensure VampYTD Bridge is running:", error);
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
    if (message.action === 'download' && message.url) {
        sendToBridge(message.url);
    }
});

function sendToBridge(url) {
    fetch('http://localhost:8080/download', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: url })
    })
    .then(response => console.log("Sent to VampYTD Bridge"))
    .catch(error => console.error("Error communicating with bridge:", error));
}

// Handle clicks on the extension icon itself
chrome.action.onClicked.addListener((tab) => {
    if (tab.url.includes("youtube.com") || tab.url.includes("youtu.be") || tab.url.includes("hotstar")) {
        sendToBridge(tab.url);
    } else {
        console.log("Not a supported video site.");
    }
});

// Handle messages from the content script (injected button)
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'download' && message.url) {
        sendToBridge(message.url);
    }
});

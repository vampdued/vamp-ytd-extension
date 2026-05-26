// --- Professional Material Icons (SVGs) ---
const downloadIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24" fill="currentColor"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>`;

const checkIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24" fill="currentColor"><path d="M382-240 154-468l57-57 171 171 367-367 57 57-424 424Z"/></svg>`;

// --- Utility: Trigger Download & Animate Button ---
function triggerDownload(url, buttonElement, originalContent, isThumbnail) {
    chrome.runtime.sendMessage({ action: 'download', url: url });
    
    // Provide premium UI feedback
    if (isThumbnail) {
        buttonElement.innerHTML = checkIconSvg;
        buttonElement.style.backgroundColor = '#4CAF50'; // Success green
        buttonElement.style.transform = 'scale(1.1)';
    } else {
        buttonElement.innerHTML = `${checkIconSvg} <span style="margin-left: 8px;">Sent</span>`;
        // Make it green on success
        buttonElement.style.color = '#fff';
        buttonElement.style.backgroundColor = '#4CAF50'; 
    }

    setTimeout(() => { 
        buttonElement.innerHTML = originalContent;
        // Reset styles to let CSS hover states take over again
        buttonElement.style.backgroundColor = '';
        buttonElement.style.color = '';
        if (isThumbnail) buttonElement.style.transform = '';
    }, 2000);
}

// --- 1. Video Watch Page Button ---
function injectWatchPageButton() {
    // Locate the right-side actions container (Share, Thanks, etc.)
    const menuContainer = document.querySelector('ytd-watch-metadata #top-level-buttons-computed') || 
                          document.querySelector('ytd-menu-renderer #top-level-buttons-computed');

    if (!menuContainer) return;

    // Fix for YouTube SPA navigation: check if the button is already inside this ACTIVE container
    if (menuContainer.querySelector('#vampytd-watch-btn')) return;

    // Clean up any orphaned watch buttons that may be detached from previous pages
    document.querySelectorAll('#vampytd-watch-btn').forEach(orphan => {
        if (!menuContainer.contains(orphan)) {
            orphan.remove();
        }
    });

    const btn = document.createElement('button');
    btn.id = 'vampytd-watch-btn';
    
    const originalContent = `${downloadIconSvg} <span style="margin-left: 8px; font-weight: 500;">VampYTD</span>`;
    btn.innerHTML = originalContent;
    
    // Use YouTube's native CSS variables and styling
    btn.style.cssText = `
        background-color: var(--yt-spec-badge-chip-background, rgba(0, 0, 0, 0.05));
        color: var(--yt-spec-text-primary);
        border: none;
        padding: 0 16px;
        height: 36px;
        border-radius: 18px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        margin-right: 8px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-family: "Roboto", "Arial", sans-serif;
        transition: background-color 0.2s cubic-bezier(0.05, 0, 0, 1);
        box-sizing: border-box;
        white-space: nowrap;
    `;

    btn.onmouseover = () => {
        btn.style.backgroundColor = 'var(--yt-spec-button-chip-background-hover, rgba(0, 0, 0, 0.1))';
    };
    btn.onmouseout = () => {
        btn.style.backgroundColor = 'var(--yt-spec-badge-chip-background, rgba(0, 0, 0, 0.05))';
    };
    btn.onmousedown = () => { btn.style.transform = 'scale(0.95)'; };
    btn.onmouseup = () => { btn.style.transform = 'scale(1)'; };

    btn.addEventListener('click', () => {
        triggerDownload(window.location.href, btn, originalContent, false);
    });

    // Insert exactly at the start of the action buttons (before Like)
    menuContainer.insertBefore(btn, menuContainer.firstChild);
}

// --- Dynamic Page Observer ---
// YouTube dynamically loads content without full page reloads.
const observer = new MutationObserver(() => {
    // Inject into the watch page if we are viewing a video
    if (window.location.pathname === '/watch') {
        injectWatchPageButton();
    }
});

// Start the observer
observer.observe(document.body, { childList: true, subtree: true });

// Listen directly to YouTube's SPA navigation complete event
window.addEventListener('yt-navigate-finish', () => {
    if (window.location.pathname === '/watch') {
        injectWatchPageButton();
    }
});

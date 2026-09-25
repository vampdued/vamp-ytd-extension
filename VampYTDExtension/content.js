// --- YouTube action-button icons (SVGs) ---
const downloadIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" focusable="false" aria-hidden="true"><path d="M12 2a1 1 0 0 0-1 1v11.586l-4.293-4.293a1 1 0 1 0-1.414 1.414L12 18.414l6.707-6.707a1 1 0 1 0-1.414-1.414L13 14.586V3a1 1 0 0 0-1-1Zm7 18H5a1 1 0 0 0 0 2h14a1 1 0 0 0 0-2Z"/></svg>`;

const checkIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24" fill="currentColor"><path d="M382-240 154-468l57-57 171 171 367-367 57 57-424 424Z"/></svg>`;

// --- Utility: Trigger Download & Animate Button with Watchdog ---
function triggerDownload(url, buttonElement, originalContent, isShorts) {
    if (buttonElement.dataset.vampytdBusy === 'true') return;
    buttonElement.dataset.vampytdBusy = 'true';

    let settled = false;

    // 10s Watchdog: Ensure the button state resets even if the extension context disconnects.
    const watchdogTimer = setTimeout(() => {
        if (settled) return;
        settled = true;
        showResult(false, 'Timed out');
    }, 10_000);

    const showResult = (succeeded, label) => {
        if (isShorts) {
            buttonElement.innerHTML = succeeded ? checkIconSvg : downloadIconSvg;
            buttonElement.style.backgroundColor = succeeded ? '#10b981' : '#ef4444';
            buttonElement.style.transform = 'scale(1.1)';
        } else {
            buttonElement.innerHTML = createWatchButtonContent(succeeded ? checkIconSvg : downloadIconSvg, label);
            buttonElement.style.color = '#fff';
            buttonElement.style.backgroundColor = succeeded ? '#10b981' : '#ef4444';
        }

        setTimeout(() => {
            buttonElement.innerHTML = originalContent;
            buttonElement.style.backgroundColor = '';
            buttonElement.style.color = '';
            if (isShorts) buttonElement.style.transform = '';
            delete buttonElement.dataset.vampytdBusy;
        }, 2000);
    };

    try {
        chrome.runtime.sendMessage({ action: 'download', url }, (response) => {
            if (settled) return;
            settled = true;
            clearTimeout(watchdogTimer);

            const runtimeError = chrome.runtime.lastError;
            if (runtimeError) {
                const needsReload = runtimeError.message?.toLowerCase().includes('context invalidated');
                showResult(false, needsReload ? 'Reload page' : 'Extension error');
                return;
            }

            const succeeded = Boolean(response && response.ok);
            showResult(succeeded, succeeded ? 'Sent' : 'Bridge offline');
        });
    } catch (error) {
        if (settled) return;
        settled = true;
        clearTimeout(watchdogTimer);

        const needsReload = error instanceof Error && error.message.toLowerCase().includes('context invalidated');
        showResult(false, needsReload ? 'Reload page' : 'Extension error');
    }
}

// --- 1. Video Watch Page Button ---
function injectWatchPageButton() {
    const menuContainer =
        document.querySelector('ytd-watch-metadata #top-level-buttons-computed') ||
        document.querySelector('ytd-menu-renderer #top-level-buttons-computed');

    if (!menuContainer) return;

    // Avoid duplicate injection into the active container.
    if (menuContainer.querySelector('#vampytd-watch-btn')) return;

    // Remove any orphaned buttons from previous navigations.
    document.querySelectorAll('#vampytd-watch-btn').forEach((orphan) => {
        if (!menuContainer.contains(orphan)) orphan.remove();
    });

    const buttonModel = document.createElement('yt-button-view-model');
    buttonModel.id = 'vampytd-watch-btn';
    buttonModel.className = 'ytd-menu-renderer';
    // Use logical marginInlineEnd instead of marginRight for RTL compatibility.
    buttonModel.style.marginInlineEnd = '8px';
    buttonModel.innerHTML = `
        <button-view-model class="ytSpecButtonViewModelHost style-scope ytd-menu-renderer">
        <button type="button" class="ytSpecButtonShapeNextHost ytSpecButtonShapeNextTonal ytSpecButtonShapeNextMono ytSpecButtonShapeNextSizeM ytSpecButtonShapeNextIconLeading ytSpecButtonShapeNextEnableBackdropFilterExperiment" title="Download with VampYTD" aria-label="Download with VampYTD" aria-disabled="false">
            ${createWatchButtonContent(downloadIconSvg, 'VampYTD')}
        </button>
        </button-view-model>`;

    const btn = buttonModel.querySelector('button');
    const originalContent = createWatchButtonContent(downloadIconSvg, 'VampYTD');

    btn.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
        triggerDownload(window.location.href, btn, originalContent, false);
    }, true);

    menuContainer.insertBefore(buttonModel, menuContainer.firstChild);
}

function createWatchButtonContent(icon, label) {
    return `
        <div aria-hidden="true" class="ytSpecButtonShapeNextIcon ytSpecButtonShapeNextElevatedContent"><span class="ytIconWrapperHost" style="width: 24px; height: 24px;"><span class="yt-icon-shape ytSpecIconShapeHost"><div style="width: 100%; height: 100%; display: block; fill: currentcolor;">${icon}</div></span></span></div>
        <div class="ytSpecButtonShapeNextButtonTextContent ytSpecButtonShapeNextElevatedContent">${label}</div>
        <yt-touch-feedback-shape aria-hidden="true" class="ytSpecTouchFeedbackShapeHost ytSpecTouchFeedbackShapeTouchResponse"><div class="ytSpecTouchFeedbackShapeStroke"></div><div class="ytSpecTouchFeedbackShapeFill"></div></yt-touch-feedback-shape>
        <yt-light-shape aria-hidden="true" class="contribYtLightShapeHost contribYtLightShapeStaticRimLight contribYtLightShapeStaticRimLightTonal"><div class="contribYtLightShapeStaticWashLight contribYtLightShapeStaticWashLightTonal"></div></yt-light-shape>`;
}

// --- 2. YouTube Shorts In-Page Button ---
function injectShortsButton() {
    const actionPanel =
        document.querySelector('ytd-reel-video-renderer[is-active] #actions') ||
        document.querySelector('ytd-shorts #actions');

    if (!actionPanel) return;
    if (actionPanel.querySelector('#vampytd-shorts-btn')) return;

    const btn = document.createElement('button');
    btn.id = 'vampytd-shorts-btn';
    btn.type = 'button';
    btn.setAttribute('aria-label', 'Download Short with VampYTD');
    btn.title = 'Download Short with VampYTD';
    btn.innerHTML = downloadIconSvg;

    Object.assign(btn.style, {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '48px',
        height: '48px',
        borderRadius: '50%',
        border: 'none',
        background: 'rgba(255, 255, 255, 0.1)',
        color: '#ffffff',
        cursor: 'pointer',
        margin: '8px 0',
        backdropFilter: 'blur(8px)',
        transition: 'background 0.2s, transform 0.15s'
    });

    btn.addEventListener('mouseenter', () => { btn.style.background = 'rgba(255, 255, 255, 0.22)'; });
    btn.addEventListener('mouseleave', () => { btn.style.background = 'rgba(255, 255, 255, 0.1)'; });

    const originalContent = downloadIconSvg;

    btn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        triggerDownload(window.location.href, btn, originalContent, true);
    });

    actionPanel.appendChild(btn);
}

// --- 3. Floating In-Page Toast for Background Actions ---
function showFloatingToast(ok, text) {
    const existing = document.getElementById('vampytd-floating-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.id = 'vampytd-floating-toast';
    toast.style.cssText = `
        position: fixed;
        bottom: 24px;
        right: 24px;
        z-index: 999999;
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        border-radius: 999px;
        background: ${ok ? '#10b981' : '#ef4444'};
        color: #ffffff;
        font: 600 13px/1.2 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        box-shadow: 0 10px 25px rgba(0,0,0,0.45);
        opacity: 0;
        transform: translateY(8px);
        transition: opacity 0.2s ease, transform 0.2s ease;
        pointer-events: none;
    `;
    toast.textContent = text;
    document.body.appendChild(toast);

    requestAnimationFrame(() => {
        toast.style.opacity = '1';
        toast.style.transform = 'translateY(0)';
    });

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(8px)';
        setTimeout(() => toast.remove(), 250);
    }, 3000);
}

chrome.runtime.onMessage.addListener((msg) => {
    if (msg.action === 'toast') {
        showFloatingToast(msg.ok, msg.message);
    }
});

// --- Routing & Lifecycle ---
function injectForCurrentPage() {
    const path = window.location.pathname;
    if (path === '/watch') {
        injectWatchPageButton();
    } else if (path.startsWith('/shorts/')) {
        injectShortsButton();
    }
}

// Debounced observer to prevent main-thread saturation during YouTube dynamic renders
let debounceTimer = null;

const observer = new MutationObserver(() => {
    if (debounceTimer !== null) return;
    debounceTimer = setTimeout(() => {
        debounceTimer = null;
        injectForCurrentPage();
    }, 150);
});

observer.observe(document.body, { childList: true, subtree: true });

window.addEventListener('yt-navigate-finish', () => {
    if (debounceTimer !== null) {
        clearTimeout(debounceTimer);
        debounceTimer = null;
    }
    injectForCurrentPage();
});

// Initial injection
injectForCurrentPage();

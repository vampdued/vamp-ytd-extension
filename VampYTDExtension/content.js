// --- YouTube action-button icons (SVGs) ---
const downloadIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" focusable="false" aria-hidden="true"><path d="M12 2a1 1 0 0 0-1 1v11.586l-4.293-4.293a1 1 0 1 0-1.414 1.414L12 18.414l6.707-6.707a1 1 0 1 0-1.414-1.414L13 14.586V3a1 1 0 0 0-1-1Zm7 18H5a1 1 0 0 0 0 2h14a1 1 0 0 0 0-2Z"/></svg>`;

const checkIconSvg = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24" fill="currentColor"><path d="M382-240 154-468l57-57 171 171 367-367 57 57-424 424Z"/></svg>`;

// --- Utility: Trigger Download & Animate Button ---
function triggerDownload(url, buttonElement, originalContent, isThumbnail) {
    if (buttonElement.dataset.vampytdBusy === 'true') return;
    buttonElement.dataset.vampytdBusy = 'true';

    const showResult = (succeeded, label) => {
        if (isThumbnail) {
            buttonElement.innerHTML = succeeded ? checkIconSvg : downloadIconSvg;
            buttonElement.style.backgroundColor = succeeded ? '#4CAF50' : '#ef4444';
            buttonElement.style.transform = 'scale(1.1)';
        } else {
            buttonElement.innerHTML = createWatchButtonContent(succeeded ? checkIconSvg : downloadIconSvg, label);
            buttonElement.style.color = '#fff';
            buttonElement.style.backgroundColor = succeeded ? '#4CAF50' : '#ef4444';
        }

        setTimeout(() => {
            buttonElement.innerHTML = originalContent;
            buttonElement.style.backgroundColor = '';
            buttonElement.style.color = '';
            if (isThumbnail) buttonElement.style.transform = '';
            delete buttonElement.dataset.vampytdBusy;
        }, 2000);
    };

    try {
        chrome.runtime.sendMessage({ action: 'download', url }, (response) => {
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

    // Avoid duplicate injection into the same live container.
    if (menuContainer.querySelector('#vampytd-watch-btn')) return;

    // Remove any orphaned buttons from previous SPA navigations.
    document.querySelectorAll('#vampytd-watch-btn').forEach((orphan) => {
        if (!menuContainer.contains(orphan)) orphan.remove();
    });

    const buttonModel = document.createElement('yt-button-view-model');
    buttonModel.id = 'vampytd-watch-btn';
    buttonModel.className = 'ytd-menu-renderer';
    buttonModel.style.marginRight = '8px';
    buttonModel.innerHTML = `
        <button-view-model class="ytSpecButtonViewModelHost style-scope ytd-menu-renderer">
        <button type="button" class="ytSpecButtonShapeNextHost ytSpecButtonShapeNextTonal ytSpecButtonShapeNextMono ytSpecButtonShapeNextSizeM ytSpecButtonShapeNextIconLeading ytSpecButtonShapeNextEnableBackdropFilterExperiment" title="" aria-label="Download with VampYTD" aria-disabled="false">
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

// --- 2. YouTube Shorts Button ---
function injectShortsButton() {
    // The Shorts action panel lives inside ytd-reel-video-renderer (active slide).
    const actionPanel =
        document.querySelector('ytd-reel-video-renderer[is-active] #actions') ||
        document.querySelector('ytd-shorts #actions');

    if (!actionPanel) return;
    if (actionPanel.querySelector('#vampytd-shorts-btn')) return;

    const btn = document.createElement('button');
    btn.id = 'vampytd-shorts-btn';
    btn.type = 'button';
    btn.setAttribute('aria-label', 'Download Short with VampYTD');
    btn.title = 'Download with VampYTD';
    btn.innerHTML = downloadIconSvg;

    // Mirror the visual style of other Shorts action buttons.
    Object.assign(btn.style, {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '48px',
        height: '48px',
        borderRadius: '50%',
        border: 'none',
        background: 'rgba(255,255,255,0.1)',
        color: '#fff',
        cursor: 'pointer',
        margin: '8px 0',
        backdropFilter: 'blur(4px)',
        transition: 'background 0.15s'
    });

    btn.addEventListener('mouseenter', () => { btn.style.background = 'rgba(255,255,255,0.2)'; });
    btn.addEventListener('mouseleave', () => { btn.style.background = 'rgba(255,255,255,0.1)'; });

    const originalContent = downloadIconSvg;

    btn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        triggerDownload(window.location.href, btn, originalContent, true);
    });

    // Append after the last existing action button.
    actionPanel.appendChild(btn);
}

// --- Route injection based on current page ---
function injectForCurrentPage() {
    const path = window.location.pathname;
    if (path === '/watch') {
        injectWatchPageButton();
    } else if (path.startsWith('/shorts/')) {
        injectShortsButton();
    }
}

// --- Debounced MutationObserver ---
// Batching DOM mutations with a short debounce avoids saturating the main
// thread during heavy YouTube SPA renders (e.g. homepage scroll, feed loads).
let debounceTimer = null;

const observer = new MutationObserver(() => {
    if (debounceTimer !== null) return; // already scheduled
    debounceTimer = setTimeout(() => {
        debounceTimer = null;
        injectForCurrentPage();
    }, 150);
});

observer.observe(document.body, { childList: true, subtree: true });

// Listen to YouTube's own SPA navigation-complete event for fast, reliable
// injection without relying solely on DOM mutations after a page change.
window.addEventListener('yt-navigate-finish', () => {
    // Cancel any pending debounce and inject immediately after navigation.
    if (debounceTimer !== null) {
        clearTimeout(debounceTimer);
        debounceTimer = null;
    }
    injectForCurrentPage();
});

// Initial injection on script load (handles hard page loads / reloads).
injectForCurrentPage();

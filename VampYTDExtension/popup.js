document.addEventListener('DOMContentLoaded', () => {
  const DEFAULTS = {
    downloadMode: 'interactive',
    preferredCodec: 'auto',
    enableCookies: false,
    enableCinematicCrop: true,
    enableBlockAmbient: true,
    enableChannelVideos: true
  };

  const overallStatus = document.getElementById('overall-status');
  const overallText   = document.getElementById('overall-text');
  const testButton    = document.getElementById('test-button');
  const checkedTime   = document.getElementById('checked-time');
  const modeSelector  = document.getElementById('mode-selector');
  const codecSelector = document.getElementById('codec-selector');
  const cookiesToggle = document.getElementById('cookies-toggle');

  const activeVideoCard   = document.getElementById('active-video-card');
  const activeVideoTitle  = document.getElementById('active-video-title');
  const activeVideoBadge  = document.getElementById('active-video-badge');
  const activeDownloadBtn = document.getElementById('active-download-btn');

  const toolsStatus  = document.getElementById('tools-status');
  const toolsSummary = document.getElementById('tools-summary');
  const toolsDetails = document.getElementById('tools-details');

  const copyFolderBtn = document.getElementById('btn-copy-folder');
  const copyLabel     = document.getElementById('copy-label');
  const folderEl      = document.getElementById('download-folder');

  // YT Enhancements switches
  const toggleCinematicCrop = document.getElementById('toggle-cinematic-crop');
  const toggleBlockAmbient   = document.getElementById('toggle-block-ambient');
  const toggleChannelVideos  = document.getElementById('toggle-channel-videos');

  let currentTabUrl = '';

  document.getElementById('version-text').textContent =
    `v${chrome.runtime.getManifest().version}`;

  function setValue(id, text, state = '') {
    const el = document.getElementById(id);
    if (!el) return;
    el.textContent = text;
    el.className = `tool-val${state ? ` ${state}` : ''}`;
  }

  function renderTool(id, available, optional = false) {
    if (available === undefined || available === null) {
      setValue(id, 'Unknown');
      return null;
    } else if (available) {
      setValue(id, 'Ready', 'good');
      return true;
    } else if (optional) {
      setValue(id, 'Optional', 'warn');
      return true;
    } else {
      setValue(id, 'Missing', 'bad');
      return false;
    }
  }

  function renderDiagnostics(result) {
    const nativeOK = Boolean(result?.native?.ok);
    const details  = result?.details;

    const nativeStatusEl = document.getElementById('native-status');

    if (nativeOK) {
      overallStatus.className = 'overall-status good';
      overallText.textContent = 'Ready';
      if (nativeStatusEl) {
        nativeStatusEl.textContent = 'Connected';
        nativeStatusEl.className = 'status-pill good';
      }
    } else {
      overallStatus.className = 'overall-status bad';
      overallText.textContent = 'Needs repair';
      if (nativeStatusEl) {
        nativeStatusEl.textContent = result?.native?.error || 'Offline';
        nativeStatusEl.className = 'status-pill bad';
      }
    }

    const tDownloader = renderTool('downloader-status', details?.downloader);
    const tYtdlp      = renderTool('ytdlp-status',      details?.ytDlp);
    const tFfmpeg     = renderTool('ffmpeg-status',     details?.ffmpeg);
    const tNode       = renderTool('node-status',       details?.node);
    renderTool('fzf-status', details?.fzf, true);

    const allRequiredReady = (tDownloader === true && tYtdlp === true && tFfmpeg === true && tNode === true);

    if (toolsStatus && toolsSummary) {
      if (!details) {
        toolsStatus.textContent = 'Checking';
        toolsStatus.className = 'status-pill';
        toolsSummary.textContent = 'Detecting dependencies…';
      } else if (allRequiredReady) {
        toolsStatus.textContent = 'All Ready';
        toolsStatus.className = 'status-pill good';
        toolsSummary.textContent = 'yt-dlp, FFmpeg, Node.js, FZF ready';
      } else {
        toolsStatus.textContent = 'Attention';
        toolsStatus.className = 'status-pill warn';
        toolsSummary.textContent = 'One or more tools require setup';
        if (toolsDetails) toolsDetails.open = true; // Auto-expand when attention is needed
      }
    }

    if (folderEl) {
      folderEl.textContent = details?.downloadDir || 'Unavailable until bridge connects';
    }

    checkedTime.textContent =
      `Checked ${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`;
  }

  function runDiagnostics() {
    testButton.disabled = true;
    testButton.textContent = 'Testing…';
    overallStatus.className = 'overall-status';
    overallText.textContent = 'Checking';

    chrome.runtime.sendMessage({ action: 'diagnostics' }, (response) => {
      const runtimeError = chrome.runtime.lastError;
      if (runtimeError) {
        renderDiagnostics(null);
        checkedTime.textContent = 'Reload extension and try again';
      } else {
        renderDiagnostics(response);
      }
      testButton.disabled = false;
      testButton.textContent = 'Test connection';
    });
  }

  // --- Copy Folder Path to Clipboard ---
  function copyFolderPath() {
    const rawPath = folderEl?.textContent;
    if (!rawPath || rawPath.startsWith('Unavailable') || rawPath.startsWith('Checking')) return;

    navigator.clipboard.writeText(rawPath).then(() => {
      if (copyLabel) copyLabel.textContent = 'Copied!';
      copyFolderBtn?.classList.add('copied');
      setTimeout(() => {
        if (copyLabel) copyLabel.textContent = 'Copy';
        copyFolderBtn?.classList.remove('copied');
      }, 2000);
    }).catch(() => {});
  }

  if (copyFolderBtn) copyFolderBtn.addEventListener('click', copyFolderPath);
  if (folderEl) folderEl.addEventListener('click', copyFolderPath);

  // --- Active Tab Video & Theme Detection ---
  function inspectActiveTab() {
    if (!chrome.tabs?.query) return;

    chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
      const activeTab = tabs?.[0];
      if (!activeTab || !activeTab.url) return;

      currentTabUrl = activeTab.url;
      let isSupported = false;
      let siteName = 'Video';

      try {
        const u = new URL(activeTab.url);
        const host = u.hostname.toLowerCase();

        if (host.includes('youtube.com') || host.includes('youtu.be')) {
          if (u.pathname === '/watch' || u.pathname.startsWith('/shorts/') || host === 'youtu.be') {
            isSupported = true;
            siteName = u.pathname.startsWith('/shorts/') ? 'Shorts' : 'YouTube';
          }

          // Check if YouTube is currently in Dark Mode to match automatically
          chrome.tabs.sendMessage(activeTab.id, { action: 'getTheme' }, (res) => {
            if (!chrome.runtime.lastError && res?.isDark) {
              document.documentElement.dataset.theme = 'dark';
            }
          });
        } else if (host.includes('hotstar.com') || host.includes('jiohotstar.com')) {
          isSupported = true;
          siteName = 'Hotstar';
        }
      } catch (_) {}

      if (isSupported && activeVideoCard) {
        activeVideoCard.hidden = false;
        let cleanTitle = activeTab.title || 'Active video';
        cleanTitle = cleanTitle.replace(/\s*-\s*YouTube$/, '').replace(/\s*\|\s*Hotstar$/, '');
        activeVideoTitle.textContent = cleanTitle;
        activeVideoBadge.textContent = siteName;
        if (siteName === 'Hotstar') {
          activeVideoBadge.classList.add('hotstar');
        } else {
          activeVideoBadge.classList.remove('hotstar');
        }
      }
    });
  }

  if (activeDownloadBtn) {
    activeDownloadBtn.addEventListener('click', () => {
      if (!currentTabUrl) return;

      activeDownloadBtn.disabled = true;
      activeDownloadBtn.innerHTML = `<span>Sending…</span>`;

      chrome.runtime.sendMessage({ action: 'download', url: currentTabUrl }, (response) => {
        const err = chrome.runtime.lastError;
        const ok = !err && Boolean(response?.ok);

        if (ok) {
          activeDownloadBtn.innerHTML = `<span>✓ Sent to VampYTD</span>`;
          activeDownloadBtn.style.background = '#059669';
        } else {
          activeDownloadBtn.innerHTML = `<span>⚠ Bridge offline</span>`;
          activeDownloadBtn.style.background = '#dc2626';
        }

        setTimeout(() => {
          activeDownloadBtn.disabled = false;
          activeDownloadBtn.innerHTML = `
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2a1 1 0 0 0-1 1v11.586l-4.293-4.293a1 1 0 1 0-1.414 1.414L12 18.414l6.707-6.707a1 1 0 1 0-1.414-1.414L13 14.586V3a1 1 0 0 0-1-1Zm7 18H5a1 1 0 0 0 0 2h14a1 1 0 0 0 0-2Z"/></svg>
            <span>Download Video</span>
          `;
          activeDownloadBtn.style.background = '';
        }, 2200);
      });
    });
  }

  // --- Broadcast setting updates to active tab ---
  function broadcastSettingsUpdate(patch) {
    chrome.tabs?.query({ active: true, currentWindow: true }, (tabs) => {
      const activeTab = tabs?.[0];
      if (activeTab?.id) {
        chrome.tabs.sendMessage(activeTab.id, {
          action: 'settingsUpdated',
          settings: patch
        }).catch(() => {});
      }
    });
  }

  // --- Tab switching with Keyboard Navigation ---
  const tabs = Array.from(document.querySelectorAll('.tab'));

  function activateTab(tab) {
    tabs.forEach((t) => {
      const isActive = t === tab;
      t.classList.toggle('active', isActive);
      t.setAttribute('aria-selected', isActive ? 'true' : 'false');
      t.tabIndex = isActive ? 0 : -1;
    });

    document.querySelectorAll('.panel').forEach((panel) => {
      panel.hidden = panel.id !== tab.dataset.panel;
    });
  }

  tabs.forEach((tab) => {
    tab.addEventListener('click', () => activateTab(tab));

    tab.addEventListener('keydown', (e) => {
      let targetTab = null;
      const idx = tabs.indexOf(tab);

      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
        targetTab = tabs[(idx + 1) % tabs.length];
      } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        targetTab = tabs[(idx - 1 + tabs.length) % tabs.length];
      }

      if (targetTab) {
        e.preventDefault();
        targetTab.focus();
        activateTab(targetTab);
      }
    });
  });

  // --- Radio Groups with ARIA + Keyboard Navigation ---
  function setupRadioGroup(container, itemSelector, settingKey, attrKey) {
    const options = Array.from(container.querySelectorAll(itemSelector));

    function selectOption(option) {
      options.forEach((o) => {
        const isMatch = o === option;
        o.classList.toggle('active', isMatch);
        o.setAttribute('aria-checked', isMatch ? 'true' : 'false');
        o.tabIndex = isMatch ? 0 : -1;
      });
      chrome.storage.local.set({ [settingKey]: option.dataset[attrKey] });
    }

    container.addEventListener('click', (event) => {
      const option = event.target.closest(itemSelector);
      if (option) selectOption(option);
    });

    container.addEventListener('keydown', (e) => {
      const current = options.find((o) => o.classList.contains('active')) || options[0];
      const idx = options.indexOf(current);
      let nextOption = null;

      if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
        nextOption = options[(idx + 1) % options.length];
      } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
        nextOption = options[(idx - 1 + options.length) % options.length];
      } else if (e.key === ' ' || e.key === 'Enter') {
        const focused = document.activeElement.closest(itemSelector);
        if (focused) nextOption = focused;
      }

      if (nextOption) {
        e.preventDefault();
        nextOption.focus();
        selectOption(nextOption);
      }
    });

    return selectOption;
  }

  const setModeOption  = setupRadioGroup(modeSelector, '.option-chip', 'downloadMode', 'mode');
  const setCodecOption = setupRadioGroup(codecSelector, '.codec-chip', 'preferredCodec', 'codec');

  // --- Restore Saved Settings ---
  chrome.storage.local.get(DEFAULTS, (items) => {
    cookiesToggle.checked = items.enableCookies;

    const savedModeEl = modeSelector.querySelector(`[data-mode="${items.downloadMode}"]`);
    if (savedModeEl) setModeOption(savedModeEl);

    const savedCodecEl = codecSelector.querySelector(`[data-codec="${items.preferredCodec}"]`);
    if (savedCodecEl) setCodecOption(savedCodecEl);

    if (toggleCinematicCrop) toggleCinematicCrop.checked = Boolean(items.enableCinematicCrop);
    if (toggleBlockAmbient)   toggleBlockAmbient.checked   = Boolean(items.enableBlockAmbient);
    if (toggleChannelVideos)  toggleChannelVideos.checked  = Boolean(items.enableChannelVideos);
  });

  // --- Settings Listeners ---
  cookiesToggle.addEventListener('change', () => {
    chrome.storage.local.set({ enableCookies: cookiesToggle.checked });
  });

  if (toggleCinematicCrop) {
    toggleCinematicCrop.addEventListener('change', () => {
      const val = toggleCinematicCrop.checked;
      chrome.storage.local.set({ enableCinematicCrop: val });
      broadcastSettingsUpdate({ enableCinematicCrop: val });
    });
  }

  if (toggleBlockAmbient) {
    toggleBlockAmbient.addEventListener('change', () => {
      const val = toggleBlockAmbient.checked;
      chrome.storage.local.set({ enableBlockAmbient: val });
      broadcastSettingsUpdate({ enableBlockAmbient: val });
    });
  }

  if (toggleChannelVideos) {
    toggleChannelVideos.addEventListener('change', () => {
      const val = toggleChannelVideos.checked;
      chrome.storage.local.set({ enableChannelVideos: val });
      broadcastSettingsUpdate({ enableChannelVideos: val });
    });
  }

  testButton.addEventListener('click', runDiagnostics);

  // Initialize
  inspectActiveTab();
  runDiagnostics();
});

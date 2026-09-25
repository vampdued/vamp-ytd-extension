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
    el.className = `health-value${state ? ` ${state}` : ''}`;
  }

  function renderTool(id, available, optional = false) {
    if (available === undefined || available === null) {
      setValue(id, 'Unknown');
    } else if (available) {
      setValue(id, 'Ready', 'good');
    } else if (optional) {
      setValue(id, 'Not installed (optional)', 'warn');
    } else {
      setValue(id, 'Missing', 'bad');
    }
  }

  function renderDiagnostics(result) {
    const nativeOK = Boolean(result?.native?.ok);
    const details  = result?.details;

    if (nativeOK) {
      overallStatus.className = 'overall-status good';
      overallText.textContent = 'Ready';
      setValue('native-status', 'Connected', 'good');
    } else {
      overallStatus.className = 'overall-status bad';
      overallText.textContent = 'Needs repair';
      setValue('native-status', result?.native?.error || 'Not connected', 'bad');
    }

    renderTool('downloader-status', details?.downloader);
    renderTool('ytdlp-status',      details?.ytDlp);
    renderTool('ffmpeg-status',     details?.ffmpeg);
    renderTool('node-status',       details?.node);
    renderTool('fzf-status',        details?.fzf, true);

    const folderEl = document.getElementById('download-folder');
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
        checkedTime.textContent = 'Reload the extension and try again';
      } else {
        renderDiagnostics(response);
      }
      testButton.disabled = false;
      testButton.textContent = 'Test connection';
    });
  }

  // --- Active Tab Video Detection ---
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
      activeDownloadBtn.textContent = 'Sending…';

      chrome.runtime.sendMessage({ action: 'download', url: currentTabUrl }, (response) => {
        const err = chrome.runtime.lastError;
        const ok = !err && Boolean(response?.ok);

        if (ok) {
          activeDownloadBtn.textContent = '✓ Sent to VampYTD';
          activeDownloadBtn.style.background = '#059669';
        } else {
          activeDownloadBtn.textContent = '⚠ Bridge offline';
          activeDownloadBtn.style.background = '#dc2626';
        }

        setTimeout(() => {
          activeDownloadBtn.disabled = false;
          activeDownloadBtn.textContent = 'Download with VampYTD';
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
  function setupRadioGroup(container, settingKey, attrKey) {
    const options = Array.from(container.querySelectorAll('.option'));

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
      const option = event.target.closest('.option');
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
        const focused = document.activeElement.closest('.option');
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

  const setModeOption  = setupRadioGroup(modeSelector, 'downloadMode', 'mode');
  const setCodecOption = setupRadioGroup(codecSelector, 'preferredCodec', 'codec');

  // --- Restore Saved Settings ---
  chrome.storage.local.get(DEFAULTS, (items) => {
    cookiesToggle.checked = items.enableCookies;

    const savedModeEl = modeSelector.querySelector(`[data-mode="${items.downloadMode}"]`);
    if (savedModeEl) setModeOption(savedModeEl);

    const savedCodecEl = codecSelector.querySelector(`[data-codec="${items.preferredCodec}"]`);
    if (savedCodecEl) setCodecOption(savedCodecEl);

    // YT Enhancements toggles
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

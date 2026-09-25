document.addEventListener('DOMContentLoaded', () => {
  const DEFAULTS = {
    downloadMode: 'interactive',
    preferredCodec: 'auto',
    enableCookies: false
  };

  const overallStatus = document.getElementById('overall-status');
  const overallText   = document.getElementById('overall-text');
  const testButton    = document.getElementById('test-button');
  const checkedTime   = document.getElementById('checked-time');
  const modeSelector  = document.getElementById('mode-selector');
  const codecSelector = document.getElementById('codec-selector');
  const cookiesToggle = document.getElementById('cookies-toggle');

  document.getElementById('version-text').textContent =
    `Extension v${chrome.runtime.getManifest().version}`;

  function setValue(id, text, state = '') {
    const el = document.getElementById(id);
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
    document.getElementById('download-folder').textContent =
      details?.downloadDir || 'Unavailable until bridge connects';
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

  // --- Tab switching ---
  document.querySelectorAll('.tab').forEach((tab) => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.tab').forEach((t) =>
        t.classList.toggle('active', t === tab)
      );
      document.querySelectorAll('.panel').forEach((panel) => {
        panel.hidden = panel.id !== tab.dataset.panel;
      });
    });
  });

  // --- Restore saved settings ---
  chrome.storage.local.get(DEFAULTS, (items) => {
    cookiesToggle.checked = items.enableCookies;
    modeSelector.querySelector(`[data-mode="${items.downloadMode}"]`)?.classList.add('active');
    codecSelector.querySelector(`[data-codec="${items.preferredCodec}"]`)?.classList.add('active');
  });

  // --- Persist setting changes ---
  modeSelector.addEventListener('click', (event) => {
    const option = event.target.closest('.option');
    if (!option) return;
    modeSelector.querySelectorAll('.option').forEach((o) =>
      o.classList.toggle('active', o === option)
    );
    chrome.storage.local.set({ downloadMode: option.dataset.mode });
  });

  codecSelector.addEventListener('click', (event) => {
    const option = event.target.closest('.option');
    if (!option) return;
    codecSelector.querySelectorAll('.option').forEach((o) =>
      o.classList.toggle('active', o === option)
    );
    chrome.storage.local.set({ preferredCodec: option.dataset.codec });
  });

  cookiesToggle.addEventListener('change', () => {
    chrome.storage.local.set({ enableCookies: cookiesToggle.checked });
  });

  testButton.addEventListener('click', runDiagnostics);
  runDiagnostics();
});

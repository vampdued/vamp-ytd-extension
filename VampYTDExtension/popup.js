document.addEventListener('DOMContentLoaded', () => {
  const statusDot = document.getElementById('status-dot');
  const statusText = document.getElementById('status-text');
  const cookiesToggle = document.getElementById('cookies-toggle');
  const modeSelector = document.getElementById('mode-selector');

  // Default values
  const DEFAULTS = {
    downloadMode: 'interactive',
    enableCookies: false
  };

  // Ping the local Daemon bridge to verify online/offline status
  function checkDaemonStatus() {
    fetch('http://localhost:8080/')
      .then(response => {
        if (response.ok) {
          statusDot.className = 'status-dot online';
          statusText.textContent = 'Online';
        } else {
          statusDot.className = 'status-dot offline';
          statusText.textContent = 'Issue';
        }
      })
      .catch(() => {
        statusDot.className = 'status-dot offline';
        statusText.textContent = 'Offline';
      });
  }

  // Load and apply saved configuration values
  chrome.storage.local.get(DEFAULTS, (items) => {
    // 1. Setup Cookies state
    cookiesToggle.checked = items.enableCookies;

    // 2. Setup Segment Selector state
    const activeMode = items.downloadMode || 'interactive';
    const activeOption = modeSelector.querySelector(`[data-mode="${activeMode}"]`);
    if (activeOption) {
      activeOption.classList.add('active');
    }
  });

  // Handle segment mode clicks
  modeSelector.addEventListener('click', (event) => {
    const targetOption = event.target.closest('.segment-option');
    if (!targetOption) return;

    // Clear active states and activate clicked item
    modeSelector.querySelectorAll('.segment-option').forEach(el => {
      el.classList.remove('active');
    });
    targetOption.classList.add('active');

    const selectedMode = targetOption.getAttribute('data-mode');
    chrome.storage.local.set({ downloadMode: selectedMode });
  });

  // Handle cookies toggle change
  cookiesToggle.addEventListener('change', () => {
    chrome.storage.local.set({ enableCookies: cookiesToggle.checked });
  });

  // Initial and periodic daemon health check
  checkDaemonStatus();
  setInterval(checkDaemonStatus, 5000);
});

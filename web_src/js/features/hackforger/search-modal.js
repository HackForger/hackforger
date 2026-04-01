export function initSearchModal() {
  const modal = document.getElementById('hf-search-modal');
  const input = document.getElementById('hf-search-input');
  const trigger = document.getElementById('hf-search-trigger');
  const overlay = modal?.querySelector('.hf-search-overlay');
  const resultsContainer = document.getElementById('hf-search-results');
  const assistantPanel = document.getElementById('hf-assistant-panel');
  const assistantMessage = document.getElementById('hf-assistant-message');
  const assistantDisclaimer = document.getElementById('hf-assistant-disclaimer');
  const tabs = modal?.querySelectorAll('.hf-search-tab');

  if (!modal || !input) return;

  let debounceTimer = null;
  let currentScope = 'all';
  const emptyText = resultsContainer?.dataset.emptyText || 'No results found';

  function openModal() {
    modal.style.display = 'flex';
    input.focus();
    input.value = '';
    resultsContainer.innerHTML = `<div class="hf-search-empty">${escapeHtml(emptyText)}</div>`;
    assistantPanel.style.display = 'none';
  }

  function closeModal() {
    modal.style.display = 'none';
    input.value = '';
  }

  function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function getResultUrl(r) {
    switch (r.type) {
      case 'hackathon': return `${window.config?.appSubUrl || ''}/hackathons/${r.slug || r.id}`;
      case 'bounty': return `${window.config?.appSubUrl || ''}/explore/bounties`;
      case 'grant': return `${window.config?.appSubUrl || ''}/grants/${r.slug || r.id}`;
      default: return '#';
    }
  }

  function getTypeIcon(type) {
    switch (type) {
      case 'hackathon': return 'octicon-rocket';
      case 'bounty': return 'octicon-gift';
      case 'grant': return 'octicon-heart';
      default: return 'octicon-search';
    }
  }

  function renderResults(results) {
    if (!results || results.length === 0) {
      resultsContainer.innerHTML = `<div class="hf-search-empty">${escapeHtml(emptyText)}</div>`;
      return;
    }
    resultsContainer.innerHTML = results.map((r) => `
      <a href="${getResultUrl(r)}" class="hf-search-result-item">
        <svg class="svg octicon-16"><use xlink:href="#${getTypeIcon(r.type)}"></use></svg>
        <span class="hf-search-result-title">${escapeHtml(r.title)}</span>
        <span class="hf-search-result-badge hf-badge">${escapeHtml(r.status || '')}</span>
      </a>
    `).join('');
  }

  async function doSearch(query) {
    if (!query.trim()) {
      resultsContainer.innerHTML = `<div class="hf-search-empty">${escapeHtml(emptyText)}</div>`;
      assistantPanel.style.display = 'none';
      return;
    }

    const baseUrl = window.config?.appSubUrl || '';

    const [searchResp, assistantResp] = await Promise.all([
      fetch(`${baseUrl}/hackforger/search?q=${encodeURIComponent(query)}&scope=${currentScope}`),
      fetch(`${baseUrl}/hackforger/assistant/chat`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({query}),
      }).catch(() => null),
    ]);

    if (searchResp.ok) {
      const data = await searchResp.json();
      renderResults(data.results);
    }

    if (assistantResp?.ok) {
      const assistant = await assistantResp.json();
      if (assistantMessage) assistantMessage.textContent = assistant.message;
      if (assistantDisclaimer) assistantDisclaimer.textContent = assistant.disclaimer;
      assistantPanel.style.display = 'block';
    }
  }

  trigger?.addEventListener('click', openModal);
  overlay?.addEventListener('click', closeModal);

  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      modal.style.display === 'none' || !modal.style.display ? openModal() : closeModal();
    }
    if (e.key === 'Escape' && modal.style.display !== 'none') {
      closeModal();
    }
  });

  input.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => doSearch(input.value), 300);
  });

  tabs?.forEach((tab) => {
    tab.addEventListener('click', () => {
      tabs.forEach((t) => t.classList.remove('active'));
      tab.classList.add('active');
      currentScope = tab.dataset.scope;
      if (input.value.trim()) doSearch(input.value);
    });
  });
}

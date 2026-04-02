export function initSearchModal() {
  const modal = document.getElementById('hf-search-modal');
  const input = document.getElementById('hf-search-input');
  const trigger = document.getElementById('hf-search-trigger');
  const overlay = modal?.querySelector('.hf-search-overlay');
  const resultsContainer = document.getElementById('hf-search-results');
  const assistantPanel = document.getElementById('hf-assistant-panel');
  const assistantMessage = document.getElementById('hf-assistant-message');
  const assistantDisclaimer = document.getElementById('hf-assistant-disclaimer');

  if (!modal || !input) return;

  let debounceTimer = null;
  const emptyText = resultsContainer?.dataset.emptyText || 'No results found';
  const viewAllText = resultsContainer?.dataset.viewAll || 'View all';

  // Group display order and metadata
  const groupConfig = {
    hackathons: {title: resultsContainer?.dataset.groupHackathons || 'Hackathons', viewAllUrl: '/explore/hackathons'},
    bounties: {title: resultsContainer?.dataset.groupBounties || 'Bounties', viewAllUrl: '/explore/bounties'},
    grants: {title: resultsContainer?.dataset.groupGrants || 'Grants', viewAllUrl: '/explore/grants'},
    submissions: {title: resultsContainer?.dataset.groupSubmissions || 'Submissions', viewAllUrl: '/explore/submissions'},
    repos: {title: resultsContainer?.dataset.groupRepos || 'Repositories', viewAllUrl: '/explore/repos'},
    users: {title: resultsContainer?.dataset.groupUsers || 'Users', viewAllUrl: '/explore/users'},
    issues: {title: resultsContainer?.dataset.groupIssues || 'Issues', viewAllUrl: '/issues'},
  };

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

  function renderGroupedResults(data) {
    if (!data.groups || data.groups.length === 0) {
      resultsContainer.innerHTML = `<div class="hf-search-empty">${escapeHtml(emptyText)}</div>`;
      return;
    }

    const baseUrl = window.config?.appSubUrl || '';
    let html = '';

    for (const group of data.groups) {
      const config = groupConfig[group.key];
      if (!config || !group.items || group.items.length === 0) continue;

      html += `<div class="hf-search-group">`;
      html += `<div class="hf-search-group-header">`;
      html += `<span class="hf-search-group-title">${escapeHtml(config.title)}</span>`;
      html += `<a href="${baseUrl}${config.viewAllUrl}" class="hf-search-view-all">${escapeHtml(viewAllText)} &rarr;</a>`;
      html += `</div>`;

      for (const item of group.items) {
        const url = item.url.startsWith('/') ? `${baseUrl}${item.url}` : item.url;
        html += `<a href="${url}" class="hf-search-result-item">`;
        html += `<div class="hf-search-result-content">`;
        html += `<span class="hf-search-result-title">${escapeHtml(item.title)}</span>`;
        if (item.desc) {
          html += `<span class="hf-search-result-desc">${escapeHtml(item.desc)}</span>`;
        }
        html += `</div>`;
        if (item.status) {
          html += `<span class="hf-search-result-badge hf-badge">${escapeHtml(item.status)}</span>`;
        }
        html += `</a>`;
      }

      html += `</div>`;
    }

    resultsContainer.innerHTML = html;
  }

  async function doSearch(query) {
    if (!query.trim()) {
      resultsContainer.innerHTML = `<div class="hf-search-empty">${escapeHtml(emptyText)}</div>`;
      assistantPanel.style.display = 'none';
      return;
    }

    const baseUrl = window.config?.appSubUrl || '';

    const searchResp = await fetch(`${baseUrl}/hackforger/search?q=${encodeURIComponent(query)}`);

    if (searchResp.ok) {
      const data = await searchResp.json();
      renderGroupedResults(data);
    }

    // TODO(v0.2): AI assistant integration — uncomment when backend is ready
    // const assistantResp = await fetch(`${baseUrl}/hackforger/assistant/chat`, {
    //   method: 'POST',
    //   headers: {'Content-Type': 'application/json'},
    //   body: JSON.stringify({query}),
    // }).catch(() => null);
    // if (assistantResp?.ok) {
    //   const assistant = await assistantResp.json();
    //   if (assistantMessage) assistantMessage.textContent = assistant.message;
    //   if (assistantDisclaimer) assistantDisclaimer.textContent = assistant.disclaimer;
    //   assistantPanel.style.display = 'block';
    // }
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
}

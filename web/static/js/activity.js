import { api } from './api.js';
import { escapeHtml } from './util.js';

const PAGE_SIZE = 25;

// Renders the Activity page: user/date-range/text filters, the log table
// (newest first) and pagination.
export async function renderActivity(view) {
  view.innerHTML = `
    <div class="page">
      <h2>Activity</h2>
      <form id="activity-filters" class="filters" autocomplete="off">
        <div class="field"><label for="af-search">Search</label>
          <input id="af-search" name="search" placeholder="Search actions, names, IDs…"></div>
        <div class="field"><label for="af-username">User</label>
          <input id="af-username" name="username" placeholder="Username"></div>
        <div class="field"><label for="af-from">From</label><input id="af-from" name="from" type="date"></div>
        <div class="field"><label for="af-to">To</label><input id="af-to" name="to" type="date"></div>
        <button type="submit" class="btn ghost">Filter</button>
      </form>
      <p class="error-text" id="activity-error"></p>
      <table class="data-table" id="activity-table">
        <thead><tr><th>Time</th><th>User</th><th>PC and IP</th><th>Action</th></tr></thead>
        <tbody><tr><td colspan="4">Loading…</td></tr></tbody>
      </table>
      <div class="pager">
        <button type="button" class="btn ghost" id="activity-prev">‹ Prev</button>
        <span id="activity-page-label"></span>
        <button type="button" class="btn ghost" id="activity-next">Next ›</button>
      </div>
    </div>
  `;

  const form = view.querySelector('#activity-filters');
  const prevBtn = view.querySelector('#activity-prev');
  const nextBtn = view.querySelector('#activity-next');
  let page = 1;

  async function load() {
    const tbody = view.querySelector('#activity-table tbody');
    const errorEl = view.querySelector('#activity-error');
    errorEl.textContent = '';

    const params = new URLSearchParams();
    if (form.search.value.trim()) params.set('search', form.search.value.trim());
    if (form.username.value.trim()) params.set('username', form.username.value.trim());
    if (form.from.value) params.set('from', form.from.value);
    if (form.to.value) params.set('to', form.to.value);
    params.set('page', String(page));
    params.set('page_size', String(PAGE_SIZE));

    try {
      const result = await api.get(`/activity?${params.toString()}`);
      const entries = result.entries || [];
      tbody.innerHTML = entries.length
        ? entries.map(activityRow).join('')
        : '<tr><td colspan="4">No matching activity.</td></tr>';

      const totalPages = Math.max(1, Math.ceil(result.total / PAGE_SIZE));
      view.querySelector('#activity-page-label').textContent = `Page ${page} of ${totalPages}`;
      prevBtn.disabled = page <= 1;
      nextBtn.disabled = page >= totalPages;
    } catch (err) {
      tbody.innerHTML = '';
      errorEl.textContent = err.message;
    }
  }

  form.addEventListener('submit', (e) => {
    e.preventDefault();
    page = 1;
    load();
  });
  prevBtn.addEventListener('click', () => {
    if (page > 1) {
      page--;
      load();
    }
  });
  nextBtn.addEventListener('click', () => {
    page++;
    load();
  });

  await load();
}

function activityRow(e) {
  const when = e.ts ? new Date(e.ts).toLocaleString() : '';
  const pcAndIp = [e.host, e.ip].filter(Boolean).join(' · ');
  const action = e.target ? `${e.action} · ${e.target}` : e.action;
  return `<tr>
    <td>${escapeHtml(when)}</td>
    <td>${escapeHtml(e.username)}</td>
    <td>${escapeHtml(pcAndIp)}</td>
    <td>${escapeHtml(action)}</td>
  </tr>`;
}

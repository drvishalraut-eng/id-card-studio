// The collapsible "Under the hood" panel at the bottom of the wizard's
// left panel: connected users (from presence) and server details.
import { api } from './api.js';
import { escapeHtml } from './util.js';

const POLL_MS = 15000;

function timeAgo(iso) {
  const seconds = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000));
  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;
  return `${Math.round(seconds / 60)}m ago`;
}

// createUnderTheHood mounts the panel into container, polling
// GET /api/server-info every 15s (matching the presence heartbeat
// cadence) while mounted. currentUsername marks that row "(you)".
export function createUnderTheHood(container, currentUsername) {
  container.innerHTML = `
    <details class="under-the-hood">
      <summary>Under the hood</summary>
      <div class="uth-body">
        <p class="uth-count">Loading…</p>
        <ul class="uth-users"></ul>
        <dl class="uth-server"></dl>
      </div>
    </details>
  `;

  const details = container.querySelector('.under-the-hood');
  const countEl = container.querySelector('.uth-count');
  const usersEl = container.querySelector('.uth-users');
  const serverEl = container.querySelector('.uth-server');

  async function refresh() {
    let info;
    try {
      info = await api.get('/server-info');
    } catch {
      return; // best-effort; keep showing the last successful render
    }

    const users = info.users || [];
    countEl.textContent = `${users.length} user${users.length === 1 ? '' : 's'} connected`;
    usersEl.innerHTML = users
      .map(
        (u) => `
        <li>
          <strong>${escapeHtml(u.username)}${u.username === currentUsername ? ' (you)' : ''}</strong>
          <span>${escapeHtml(u.host)} · ${escapeHtml(u.ip)}</span>
          <span>${escapeHtml(u.step)}</span>
          <span>${escapeHtml(timeAgo(u.last_seen))}</span>
        </li>`,
      )
      .join('');

    serverEl.innerHTML = `
      <div><dt>URL</dt><dd>${(info.urls || []).map(escapeHtml).join('<br>')}</dd></div>
      <div><dt>Uptime</dt><dd>${escapeHtml(info.uptime)}</dd></div>
      <div><dt>Data folder</dt><dd>${escapeHtml(info.data_dir)}</dd></div>
      <div><dt>Export folder</dt><dd>${escapeHtml(info.export_dir)}</dd></div>
    `;
  }

  refresh();
  const timer = setInterval(refresh, POLL_MS);
  // Refresh immediately on open, so expanding the panel never shows a
  // stale snapshot from up to POLL_MS ago.
  details.addEventListener('toggle', () => {
    if (details.open) refresh();
  });

  return {
    destroy() {
      clearInterval(timer);
    },
  };
}

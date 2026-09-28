import { api } from './api.js';

// Renders the Admin-only Data page: export everything as one zip, restore
// from one, or wipe it all back to a fresh install. Import and Clear both
// invalidate every session on success (the server signs everyone out), so
// both reload the page afterwards to send the browser back to sign-in.
export async function renderData(view) {
  view.innerHTML = `
    <div class="page">
      <h2>Data</h2>

      <h3>Export backup</h3>
      <p>Downloads every client, employee, user, photo and client logo as one zip file. Port and export-folder settings are not included.</p>
      <p class="error-text" id="export-error"></p>
      <button type="button" class="btn" id="export-btn">Export backup</button>

      <h3>Import backup</h3>
      <p class="warning-text">This replaces every client, employee, user, photo and logo with the backup's. Nothing currently here is kept, and everyone — including you — is signed out once it finishes.</p>
      <form id="import-form" autocomplete="off">
        <div class="field"><label for="import-file">Backup zip file</label>
          <input id="import-file" type="file" accept=".zip" required></div>
        <label class="checkbox-field">
          <input type="checkbox" id="import-confirm">
          I understand this replaces all current data
        </label>
        <p class="error-text" id="import-error"></p>
        <p class="success-text" id="import-success"></p>
        <button type="submit" class="btn" id="import-btn" disabled>Import backup</button>
      </form>

      <h3>Clear all data</h3>
      <p class="warning-text">Permanently deletes every client, employee, user, photo and logo, and signs everyone out. Helios is reseeded afterwards, same as a fresh install.</p>
      <form id="clear-form" autocomplete="off">
        <div class="field"><label for="clear-confirm-text">Type CLEAR to confirm</label>
          <input id="clear-confirm-text" autocomplete="off"></div>
        <p class="error-text" id="clear-error"></p>
        <p class="success-text" id="clear-success"></p>
        <button type="submit" class="btn danger" id="clear-btn" disabled>Clear all data</button>
      </form>
    </div>
  `;

  wireExport(view);
  wireImport(view);
  wireClear(view);
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function wireExport(view) {
  const btn = view.querySelector('#export-btn');
  const errorEl = view.querySelector('#export-error');

  btn.addEventListener('click', async () => {
    errorEl.textContent = '';
    btn.disabled = true;
    try {
      // A binary download needs the raw response, not api.js's JSON parsing.
      const res = await fetch('/api/backup/export', {
        headers: { 'X-Requested-With': 'idcard' },
        credentials: 'same-origin',
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Export failed (${res.status}).`);
      }
      const blob = await res.blob();
      const match = /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') || '');
      downloadBlob(blob, match ? match[1] : 'idcard-backup.zip');
    } catch (err) {
      errorEl.textContent = err.message;
    } finally {
      btn.disabled = false;
    }
  });
}

function wireImport(view) {
  const form = view.querySelector('#import-form');
  const fileInput = view.querySelector('#import-file');
  const confirmBox = view.querySelector('#import-confirm');
  const btn = view.querySelector('#import-btn');
  const errorEl = view.querySelector('#import-error');
  const successEl = view.querySelector('#import-success');

  const refresh = () => {
    btn.disabled = !(fileInput.files.length && confirmBox.checked);
  };
  fileInput.addEventListener('change', refresh);
  confirmBox.addEventListener('change', refresh);

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    const file = fileInput.files[0];
    if (!file) return;
    btn.disabled = true;
    try {
      const body = new FormData();
      body.append('backup', file);
      await api.post('/backup/import', body);
      successEl.textContent = 'Backup imported. Signing you out…';
      setTimeout(() => location.reload(), 1200);
    } catch (err) {
      errorEl.textContent = err.message;
      btn.disabled = false;
    }
  });
}

function wireClear(view) {
  const form = view.querySelector('#clear-form');
  const textInput = view.querySelector('#clear-confirm-text');
  const btn = view.querySelector('#clear-btn');
  const errorEl = view.querySelector('#clear-error');
  const successEl = view.querySelector('#clear-success');

  textInput.addEventListener('input', () => {
    btn.disabled = textInput.value.trim() !== 'CLEAR';
  });

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    btn.disabled = true;
    try {
      await api.post('/backup/clear');
      successEl.textContent = 'All data cleared. Signing you out…';
      setTimeout(() => location.reload(), 1200);
    } catch (err) {
      errorEl.textContent = err.message;
      btn.disabled = false;
    }
  });
}

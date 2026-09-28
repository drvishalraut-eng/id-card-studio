import { api } from './api.js';
import { escapeHtml } from './util.js';

// Renders the Admin-only Users page: the users table (Reset PIN,
// Disable/Enable) and the add-user form.
export async function renderUsers(view) {
  view.innerHTML = `
    <div class="page">
      <h2>Users</h2>
      <p class="error-text" id="users-error"></p>
      <table class="data-table" id="users-table">
        <thead><tr><th>Name</th><th>Username</th><th>Role</th><th>Status</th><th>Last login</th><th>Actions</th></tr></thead>
        <tbody><tr><td colspan="6">Loading…</td></tr></tbody>
      </table>

      <h3>Add user</h3>
      <form id="add-user-form" autocomplete="off" novalidate>
        <div class="field"><label for="au-name">Name</label><input id="au-name" name="name" required></div>
        <div class="field"><label for="au-username">Username</label><input id="au-username" name="username" required></div>
        <div class="field"><label for="au-role">Role</label>
          <select id="au-role" name="role">
            <option value="operator">Operator</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <div class="field"><label for="au-pin">PIN (4-6 digits)</label>
          <input id="au-pin" name="pin" type="password" inputmode="numeric" pattern="[0-9]{4,6}" required></div>
        <div class="field"><label for="au-confirm">Confirm PIN</label>
          <input id="au-confirm" name="confirm" type="password" inputmode="numeric" pattern="[0-9]{4,6}" required></div>
        <p class="error-text" id="add-user-error"></p>
        <button type="submit" class="btn">Add user</button>
      </form>
    </div>
  `;

  const table = view.querySelector('#users-table');
  table.addEventListener('click', (e) => onTableClick(e, table, view));
  wireAddUserForm(view, table);

  await loadUsers(table, view);
}

async function loadUsers(table, view) {
  const tbody = table.querySelector('tbody');
  const errorEl = view.querySelector('#users-error');
  errorEl.textContent = '';
  try {
    const users = await api.get('/users');
    tbody.innerHTML = users.length ? users.map(userRow).join('') : '<tr><td colspan="6">No users yet.</td></tr>';
  } catch (err) {
    tbody.innerHTML = '';
    errorEl.textContent = err.message;
  }
}

function userRow(u) {
  const lastLogin = u.last_login ? new Date(u.last_login).toLocaleString() : 'Never';
  const toggleLabel = u.status === 'disabled' ? 'Enable' : 'Disable';
  return `
    <tr data-username="${escapeHtml(u.username)}">
      <td>${escapeHtml(u.name)}</td>
      <td>${escapeHtml(u.username)}</td>
      <td>${escapeHtml(u.role)}</td>
      <td>${escapeHtml(u.status)}</td>
      <td>${escapeHtml(lastLogin)}</td>
      <td>
        <button type="button" class="btn ghost reset-pin-btn">Reset PIN</button>
        <button type="button" class="btn ghost toggle-status-btn">${toggleLabel}</button>
      </td>
    </tr>
  `;
}

async function onTableClick(e, table, view) {
  const row = e.target.closest('tr[data-username]');
  if (!row) return;
  const username = row.dataset.username;

  if (e.target.classList.contains('toggle-status-btn')) {
    const disabling = e.target.textContent.trim() === 'Disable';
    const action = disabling ? 'disable' : 'enable';
    e.target.disabled = true;
    try {
      await api.post(`/users/${encodeURIComponent(username)}/${action}`);
      await loadUsers(table, view);
    } catch (err) {
      view.querySelector('#users-error').textContent = err.message;
      e.target.disabled = false;
    }
    return;
  }

  if (e.target.classList.contains('reset-pin-btn')) {
    showResetPinForm(row, username, table, view);
  }
}

let resetPinFormSeq = 0;

function showResetPinForm(row, username, table, view) {
  const cell = row.querySelector('td:last-child');
  const seq = ++resetPinFormSeq;
  const pinID = `rp-pin-${seq}`;
  const confirmID = `rp-confirm-${seq}`;
  cell.innerHTML = `
    <form class="inline-form reset-pin-form">
      <label class="sr-only" for="${pinID}">New PIN</label>
      <input id="${pinID}" class="rp-pin" type="password" inputmode="numeric" pattern="[0-9]{4,6}" placeholder="New PIN" required>
      <label class="sr-only" for="${confirmID}">Confirm PIN</label>
      <input id="${confirmID}" class="rp-confirm" type="password" inputmode="numeric" pattern="[0-9]{4,6}" placeholder="Confirm PIN" required>
      <button type="submit" class="btn">Save</button>
      <button type="button" class="btn ghost cancel-btn">Cancel</button>
    </form>
    <p class="error-text"></p>
  `;

  const form = cell.querySelector('form');
  const errorEl = cell.querySelector('.error-text');
  const pinInput = cell.querySelector('.rp-pin');
  const confirmInput = cell.querySelector('.rp-confirm');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const pin = pinInput.value;
    const confirmPin = confirmInput.value;
    if (pin !== confirmPin) {
      errorEl.textContent = 'PIN and confirm PIN must match. Re-enter them and try again.';
      return;
    }
    try {
      await api.post(`/users/${encodeURIComponent(username)}/reset-pin`, { pin, confirm_pin: confirmPin });
      await loadUsers(table, view);
    } catch (err) {
      errorEl.textContent = err.message;
    }
  });
  cell.querySelector('.cancel-btn').addEventListener('click', () => loadUsers(table, view));
}

function wireAddUserForm(view, table) {
  const form = view.querySelector('#add-user-form');
  const errorEl = view.querySelector('#add-user-error');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    const pin = form.pin.value;
    const confirm = form.confirm.value;
    if (pin !== confirm) {
      errorEl.textContent = 'PIN and confirm PIN must match. Re-enter them and try again.';
      return;
    }
    try {
      await api.post('/users', {
        name: form.name.value.trim(),
        username: form.username.value.trim(),
        role: form.role.value,
        pin,
        confirm_pin: confirm,
      });
      form.reset();
      await loadUsers(table, view);
    } catch (err) {
      errorEl.textContent = err.message;
    }
  });
}

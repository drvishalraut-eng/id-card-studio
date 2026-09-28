import { api } from './api.js';
import { escapeHtml } from './util.js';
import { resizeImageToJpeg } from './photo.js';

// normalizeHeader turns a spreadsheet header cell into one of our field
// names: trimmed, lowercased, non-alphanumerics collapsed to "_".
function normalizeHeader(cell) {
  return String(cell ?? '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
}

const IMPORT_FIELDS = ['employee_id', 'name', 'role', 'client', 'join_date', 'photo_note'];

// parseSheet reads an uploaded .xlsx/.csv ArrayBuffer (SheetJS auto-detects
// the format) and returns rows shaped for POST /api/employees/import,
// skipping fully-empty lines. join_date cells that are real Excel date
// serials come through as numbers; a plain date string (as every CSV cell
// is) is left as a string. The read()-level raw:true is required for
// this: without it, SheetJS's own CSV parser "helpfully" auto-detects
// date-like strings such as "2026-03-01" and silently converts them to a
// serial itself — computed via a local-timezone Date parse, so it lands
// on a fractional (not whole) day number, which truncates to the wrong
// calendar day depending on the browser's timezone. Leaving the string
// alone and letting the server parse "YYYY-MM-DD"/"DD-MM-YYYY" directly
// avoids that timezone round-trip entirely.
function parseSheet(arrayBuffer) {
  const workbook = window.XLSX.read(arrayBuffer, { type: 'array', raw: true });
  const sheetName = workbook.SheetNames.includes('Employees') ? 'Employees' : workbook.SheetNames[0];
  const sheet = workbook.Sheets[sheetName];
  const grid = window.XLSX.utils.sheet_to_json(sheet, { header: 1, raw: true, defval: '' });
  if (!grid.length) return [];

  const headerIndex = {};
  grid[0].forEach((cell, i) => {
    const key = normalizeHeader(cell);
    if (IMPORT_FIELDS.includes(key)) headerIndex[key] = i;
  });

  const rows = [];
  for (let r = 1; r < grid.length; r++) {
    const line = grid[r];
    if (!line || line.every((c) => c === '' || c == null)) continue;
    const row = {};
    for (const field of IMPORT_FIELDS) {
      const i = headerIndex[field];
      row[field] = i === undefined ? '' : line[i];
    }
    rows.push(row);
  }
  return rows;
}

export function renderStep2(container, deps) {
  const { user, preview } = deps;
  let clients = deps.clients;
  let employees = deps.employees;
  let selectedId = null;
  let search = '';
  let clientFilter = '';
  let editingId = null; // set while the Add/Edit form is editing an existing employee

  container.innerHTML = `
    <div class="step2">
      <div class="btns">
        <label class="btn ghost">Import CSV or Excel<input type="file" id="import-file" accept=".xlsx,.csv" hidden></label>
        <a class="btn ghost" href="/assets/employees_template.xlsx" download>Download template</a>
      </div>
      <p class="error-text" id="import-error"></p>
      <div id="import-result"></div>

      <div id="needs-attention" hidden>
        <h3>Needs attention</h3>
        <ul id="needs-attention-list"></ul>
      </div>

      <h3>Employees</h3>
      <div class="row-inline">
        <div class="field" style="flex:1;margin-top:0">
          <label for="emp-search">Search by name or ID</label>
          <input id="emp-search" placeholder="Search…">
        </div>
        <div class="field" style="flex:1;margin-top:0">
          <label for="emp-client-filter">Client</label>
          <select id="emp-client-filter"><option value="">All clients</option></select>
        </div>
      </div>
      <table class="data-table" id="employees-table">
        <thead><tr><th>ID</th><th>Name</th><th>Client</th><th>Actions</th></tr></thead>
        <tbody></tbody>
      </table>

      <h3 id="add-employee-heading">Add employee</h3>
      <form id="add-employee-form" autocomplete="off" novalidate>
        <div class="field"><label for="ae-id">Employee ID</label><input id="ae-id" name="employee_id" required></div>
        <p class="hint-text" id="ae-id-hint" hidden>Employee ID can't be changed here — delete this employee and add a new one to fix a wrong ID.</p>
        <div class="field"><label for="ae-name">Name</label><input id="ae-name" name="name" required></div>
        <div class="field"><label for="ae-role">Role</label><input id="ae-role" name="role"></div>
        <div class="field"><label for="ae-client">Client</label>
          <select id="ae-client" name="client"><option value="">Select a client…</option></select></div>
        <div class="field"><label for="ae-join-date">Join date</label><input id="ae-join-date" name="join_date" type="date" required></div>
        <div class="field"><label for="ae-photo" id="ae-photo-label">Photo (optional)</label><input id="ae-photo" type="file" accept="image/*"></div>
        <p class="error-text" id="add-employee-error"></p>
        <div class="btns">
          <button type="submit" class="btn" id="add-employee-submit">Add employee</button>
          <button type="button" class="btn ghost" id="add-employee-cancel" hidden>Cancel</button>
        </div>
      </form>
    </div>
  `;

  const importInput = container.querySelector('#import-file');
  const importError = container.querySelector('#import-error');
  const importResult = container.querySelector('#import-result');
  const needsAttentionBox = container.querySelector('#needs-attention');
  const needsAttentionList = container.querySelector('#needs-attention-list');
  const searchInput = container.querySelector('#emp-search');
  const clientFilterSelect = container.querySelector('#emp-client-filter');
  const table = container.querySelector('#employees-table');
  const addForm = container.querySelector('#add-employee-form');
  const addError = container.querySelector('#add-employee-error');
  const addClientSelect = container.querySelector('#ae-client');
  const formHeading = container.querySelector('#add-employee-heading');
  const idInput = container.querySelector('#ae-id');
  const idHint = container.querySelector('#ae-id-hint');
  const photoLabel = container.querySelector('#ae-photo-label');
  const submitBtn = container.querySelector('#add-employee-submit');
  const cancelBtn = container.querySelector('#add-employee-cancel');

  function clientOptionsHtml(includeBlank) {
    const blank = includeBlank ? '<option value="">Select a client…</option>' : '';
    return blank + clients.map((c) => `<option value="${escapeHtml(c.code)}">${escapeHtml(c.name)}</option>`).join('');
  }

  function populateClientControls() {
    clientFilterSelect.innerHTML = '<option value="">All clients</option>' + clientOptionsHtml(false);
    addClientSelect.innerHTML = clientOptionsHtml(true);
  }
  populateClientControls();

  function knownClient(code) {
    const needle = String(code || '').toLowerCase();
    return clients.find((c) => String(c.code).toLowerCase() === needle) || null;
  }

  function renderNeedsAttention() {
    const unresolved = employees.filter((e) => !knownClient(e.client_code));
    needsAttentionBox.hidden = unresolved.length === 0;
    needsAttentionList.innerHTML = unresolved
      .map(
        (e) => `
        <li data-id="${escapeHtml(e.employee_id)}">
          <span>${escapeHtml(e.name)} (${escapeHtml(e.employee_id)}) — unknown client "${escapeHtml(e.client_code || '')}"</span>
          <select class="na-select">
            <option value="">Map to…</option>
            ${clientOptionsHtml(false)}
          </select>
        </li>`,
      )
      .join('');
  }

  needsAttentionList.addEventListener('change', async (e) => {
    const select = e.target.closest('.na-select');
    if (!select || !select.value) return;
    const li = select.closest('li[data-id]');
    const id = li.dataset.id;
    select.disabled = true;
    try {
      await api.post(`/employees/${encodeURIComponent(id)}/client`, { client: select.value });
      await reloadEmployees();
    } catch (err) {
      importError.textContent = err.message;
      select.disabled = false;
    }
  });

  function filteredEmployees() {
    const q = search.trim().toLowerCase();
    return employees.filter((e) => {
      if (clientFilter && String(e.client_code || '').toLowerCase() !== clientFilter.toLowerCase()) return false;
      if (!q) return true;
      return e.name.toLowerCase().includes(q) || e.employee_id.toLowerCase().includes(q);
    });
  }

  function actionsCellHtml() {
    return `<td class="row-actions">
      <button type="button" class="btn ghost edit-btn">Edit</button>
      <button type="button" class="btn ghost delete-btn">Delete</button>
    </td>`;
  }

  function renderTable() {
    const tbody = table.querySelector('tbody');
    const rows = filteredEmployees();
    tbody.innerHTML = rows.length
      ? rows
          .map(
            (e) => `
        <tr data-id="${escapeHtml(e.employee_id)}" class="${e.employee_id === selectedId ? 'selected' : ''}">
          <td>${escapeHtml(e.employee_id)}</td>
          <td>${escapeHtml(e.name)}</td>
          <td>${escapeHtml(e.client_code)}</td>
          ${actionsCellHtml()}
        </tr>`,
          )
          .join('')
      : '<tr><td colspan="4">No employees match.</td></tr>';
  }

  // currentDraftEmployee builds an Employee-shaped object from the form's
  // current (possibly unsaved) values, so the preview can show it live —
  // preserving photo/crop from the employee being edited, since this form
  // never touches those directly.
  function currentDraftEmployee() {
    const base = editingId ? employees.find((emp) => emp.employee_id === editingId) : null;
    return {
      employee_id: idInput.value.trim(),
      name: addForm.name.value.trim(),
      role: addForm.role.value.trim(),
      client_code: addForm.client.value,
      join_date: addForm.join_date.value,
      photo: base ? base.photo : '',
      crop: base ? base.crop : { zoom: 1, x: 50, y: 50 },
    };
  }

  function updatePreviewDraft() {
    const draft = currentDraftEmployee();
    preview.setOverride(knownClient(draft.client_code), draft);
  }

  addForm.addEventListener('input', updatePreviewDraft);
  addForm.client.addEventListener('change', updatePreviewDraft);

  function startEdit(employee) {
    editingId = employee.employee_id;
    addError.textContent = '';
    formHeading.textContent = `Edit employee — ${employee.employee_id}`;
    idInput.value = employee.employee_id;
    idInput.readOnly = true;
    idHint.hidden = false;
    addForm.name.value = employee.name || '';
    addForm.role.value = employee.role || '';
    addForm.client.value = employee.client_code || '';
    addForm.join_date.value = employee.join_date || '';
    addForm.querySelector('#ae-photo').value = '';
    photoLabel.textContent = employee.photo ? 'Replace photo (optional)' : 'Add photo (optional)';
    submitBtn.textContent = 'Save changes';
    cancelBtn.hidden = false;
    addForm.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    updatePreviewDraft();
  }

  function stopEdit() {
    editingId = null;
    addError.textContent = '';
    formHeading.textContent = 'Add employee';
    idInput.readOnly = false;
    idHint.hidden = true;
    photoLabel.textContent = 'Photo (optional)';
    submitBtn.textContent = 'Add employee';
    cancelBtn.hidden = true;
    addForm.reset();
    preview.clearOverride();
  }

  cancelBtn.addEventListener('click', stopEdit);

  function showDeleteConfirm(row, id) {
    const cell = row.querySelector('.row-actions');
    cell.innerHTML = `
      <span class="delete-confirm-text">Delete this employee?</span>
      <button type="button" class="btn danger confirm-delete-btn">Delete</button>
      <button type="button" class="btn ghost cancel-delete-btn">Cancel</button>
    `;
    cell.querySelector('.cancel-delete-btn').addEventListener('click', renderTable);
    cell.querySelector('.confirm-delete-btn').addEventListener('click', async (e) => {
      e.target.disabled = true;
      try {
        await api.delete(`/employees/${encodeURIComponent(id)}`);
        if (editingId === id) stopEdit();
        if (selectedId === id) selectedId = null;
        await reloadEmployees();
      } catch (err) {
        importError.textContent = err.message;
        renderTable();
      }
    });
  }

  table.addEventListener('click', (e) => {
    const row = e.target.closest('tr[data-id]');
    if (!row) return;
    const id = row.dataset.id;

    if (e.target.closest('.edit-btn')) {
      const employee = employees.find((emp) => emp.employee_id === id);
      if (employee) startEdit(employee);
      return;
    }
    if (e.target.closest('.delete-btn')) {
      showDeleteConfirm(row, id);
      return;
    }

    selectedId = id;
    const index = employees.findIndex((emp) => emp.employee_id === selectedId);
    if (index >= 0) {
      preview.clearOverride();
      preview.setIndex(index);
    }
    renderTable();
  });

  searchInput.addEventListener('input', () => {
    search = searchInput.value;
    renderTable();
  });
  clientFilterSelect.addEventListener('change', () => {
    clientFilter = clientFilterSelect.value;
    renderTable();
  });

  async function reloadEmployees() {
    employees = await api.get('/employees');
    renderNeedsAttention();
    renderTable();
    deps.onEmployeesChanged(employees);
  }

  importInput.addEventListener('change', async () => {
    const file = importInput.files[0];
    importInput.value = '';
    if (!file) return;
    importError.textContent = '';
    importResult.textContent = '';

    let rows;
    try {
      const buffer = await file.arrayBuffer();
      rows = parseSheet(buffer);
    } catch (err) {
      importError.textContent = `Could not read ${file.name}: ${err.message}`;
      return;
    }
    if (!rows.length) {
      importError.textContent = `${file.name} has no data rows.`;
      return;
    }

    try {
      const result = await api.post('/employees/import', { rows });
      importResult.innerHTML =
        `<p>${result.imported} imported, ${result.skipped.length} skipped.</p>` +
        (result.skipped.length
          ? `<ul>${result.skipped.map((s) => `<li>Row ${s.row}: ${escapeHtml(s.reason)}</li>`).join('')}</ul>`
          : '');
      await reloadEmployees();
    } catch (err) {
      importError.textContent = err.message;
    }
  });

  addForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    addError.textContent = '';
    const photoFile = addForm.querySelector('#ae-photo').files[0];

    try {
      const saved = await api.post('/employees', {
        employee_id: addForm.employee_id.value.trim(),
        name: addForm.name.value.trim(),
        role: addForm.role.value.trim(),
        client: addForm.client.value,
        join_date: addForm.join_date.value,
      });

      if (photoFile) {
        const { blob } = await resizeImageToJpeg(photoFile);
        const form = new FormData();
        form.append('photo', blob, 'photo.jpg');
        form.append('zoom', '1');
        form.append('x', '50');
        form.append('y', '50');
        await api.post(`/employees/${encodeURIComponent(saved.employee_id)}/photo`, form);
      }

      stopEdit();
      await reloadEmployees();
      const savedIndex = employees.findIndex((emp) => emp.employee_id === saved.employee_id);
      if (savedIndex >= 0) preview.setIndex(savedIndex);
    } catch (err) {
      addError.textContent = err.message;
    }
  });

  renderNeedsAttention();
  renderTable();
}

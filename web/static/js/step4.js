import { api } from './api.js';
import { escapeHtml } from './util.js';
import { formatJoinDate, formatISODate, FIXED_MONTHS } from './card.js';
import { exportOne, exportFilename } from './cardexport.js';

const ROW_HEIGHT = 40; // px; virtualization math only, doesn't need to be exact
const HEADER_HEIGHT = 32;
const BUFFER_PX = 400;

function monthKey(joinDate) {
  const m = /^(\d{4})-(\d{2})-\d{2}$/.exec(joinDate || '');
  return m ? `${m[1]}-${m[2]}` : '';
}
function monthLabel(key) {
  const [y, mo] = key.split('-');
  return `${FIXED_MONTHS[Number(mo) - 1]} ${y}`;
}

export function renderStep4(container, deps) {
  let clients = deps.clients;
  let employees = deps.employees;

  const selected = new Set();
  let lastClickedId = null;
  let search = '';
  let clientFilter = '';
  let monthFilter = '';
  let statusFilter = 'not_exported'; // 'not_exported' | 'exported' | 'any'

  container.innerHTML = `
    <div class="step4">
      <div class="row-inline">
        <div class="field" style="flex:1;margin-top:0">
          <label for="ex-search">Search by name or ID</label>
          <input id="ex-search" placeholder="Search…">
        </div>
        <div class="field" style="flex:1;margin-top:0">
          <label for="ex-client-filter">Client</label>
          <select id="ex-client-filter"><option value="">All clients</option></select>
        </div>
      </div>
      <div class="row-inline">
        <div class="field" style="flex:1;margin-top:0">
          <label for="ex-month-filter">Join month</label>
          <select id="ex-month-filter"><option value="">All months</option></select>
        </div>
        <div class="field" style="flex:1;margin-top:0">
          <label for="ex-status-filter">Export status</label>
          <select id="ex-status-filter">
            <option value="not_exported">Not exported</option>
            <option value="exported">Exported</option>
            <option value="any">Any</option>
          </select>
        </div>
      </div>

      <div class="btns">
        <button type="button" class="btn ghost" id="select-shown-btn">Select shown</button>
        <button type="button" class="btn ghost" id="clear-selection-btn">Clear</button>
      </div>
      <p class="selection-summary" id="selection-summary"></p>
      <p class="warning-text" id="no-photo-warning" hidden></p>

      <div class="export-list" id="export-list">
        <div class="export-list-spacer-top"></div>
        <div class="export-list-rows"></div>
        <div class="export-list-spacer-bottom"></div>
      </div>

      <p class="error-text" id="export-error"></p>
      <div id="export-progress" hidden>
        <p id="export-progress-label"></p>
        <div class="progress-bar"><div class="progress-fill" id="export-progress-fill"></div></div>
      </div>
      <button type="button" class="btn" id="export-btn">Export PDF</button>
    </div>
  `;

  const searchInput = container.querySelector('#ex-search');
  const clientFilterSelect = container.querySelector('#ex-client-filter');
  const monthFilterSelect = container.querySelector('#ex-month-filter');
  const statusFilterSelect = container.querySelector('#ex-status-filter');
  const listEl = container.querySelector('#export-list');
  const rowsEl = container.querySelector('.export-list-rows');
  const topSpacer = container.querySelector('.export-list-spacer-top');
  const bottomSpacer = container.querySelector('.export-list-spacer-bottom');
  const summaryEl = container.querySelector('#selection-summary');
  const noPhotoWarning = container.querySelector('#no-photo-warning');
  const exportBtn = container.querySelector('#export-btn');
  const exportError = container.querySelector('#export-error');
  const progressBox = container.querySelector('#export-progress');
  const progressLabel = container.querySelector('#export-progress-label');
  const progressFill = container.querySelector('#export-progress-fill');

  clientFilterSelect.innerHTML =
    '<option value="">All clients</option>' +
    clients.map((c) => `<option value="${escapeHtml(c.code)}">${escapeHtml(c.name)}</option>`).join('');

  const months = [...new Set(employees.map((e) => monthKey(e.join_date)).filter(Boolean))].sort().reverse();
  monthFilterSelect.innerHTML =
    '<option value="">All months</option>' +
    months.map((m) => `<option value="${m}">${monthLabel(m)}</option>`).join('');

  function clientFor(code) {
    const needle = String(code || '').toLowerCase();
    return clients.find((c) => String(c.code).toLowerCase() === needle) || null;
  }

  function matchesFilters(e) {
    if (clientFilter && String(e.client_code || '').toLowerCase() !== clientFilter.toLowerCase()) return false;
    if (monthFilter && monthKey(e.join_date) !== monthFilter) return false;
    if (statusFilter === 'not_exported' && e.exported_at) return false;
    if (statusFilter === 'exported' && !e.exported_at) return false;
    const q = search.trim().toLowerCase();
    if (q && !e.name.toLowerCase().includes(q) && !e.employee_id.toLowerCase().includes(q)) return false;
    return true;
  }

  // flatRows: the filtered employees grouped by client, flattened into
  // {type:'header', ...} / {type:'row', employee} entries in display
  // order — the order shift-click ranges and virtualization both use.
  let flatRows = [];
  let shownIds = [];

  function rebuildRows() {
    const filtered = employees.filter(matchesFilters);
    const groups = new Map();
    for (const e of filtered) {
      const key = String(e.client_code || '').toLowerCase();
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(e);
    }
    flatRows = [];
    for (const [key, rows] of groups) {
      const client = clientFor(key);
      flatRows.push({ type: 'header', key, label: client ? client.name : rows[0].client_code || '(no client)', ids: rows.map((r) => r.employee_id) });
      for (const e of rows) flatRows.push({ type: 'row', employee: e });
    }
    shownIds = filtered.map((e) => e.employee_id);
  }

  function rowOffsetY(i) {
    let y = 0;
    for (let j = 0; j < i; j++) y += flatRows[j].type === 'header' ? HEADER_HEIGHT : ROW_HEIGHT;
    return y;
  }
  function totalHeight() {
    return flatRows.reduce((sum, r) => sum + (r.type === 'header' ? HEADER_HEIGHT : ROW_HEIGHT), 0);
  }

  function rowHtml(entry) {
    if (entry.type === 'header') {
      const groupSelected = entry.ids.filter((id) => selected.has(id));
      const allSelected = entry.ids.length > 0 && groupSelected.length === entry.ids.length;
      return `
        <div class="export-group-header" data-group="${escapeHtml(entry.key)}" style="height:${HEADER_HEIGHT}px">
          <label>
            <input type="checkbox" class="group-checkbox" data-group="${escapeHtml(entry.key)}" ${allSelected ? 'checked' : ''}>
            ${escapeHtml(entry.label)}
          </label>
        </div>`;
    }
    const e = entry.employee;
    const filename = exportFilename(e.employee_id, e.name);
    const joined = `Joined ${formatJoinDate(e.join_date)}`;
    const exported = e.exported_at
      ? ` · exported ${formatISODate(e.exported_at)} by ${escapeHtml(e.exported_by)}`
      : ' · not exported';
    return `
      <div class="export-row" data-id="${escapeHtml(e.employee_id)}" style="height:${ROW_HEIGHT}px">
        <label class="export-row-check">
          <input type="checkbox" class="row-checkbox" data-id="${escapeHtml(e.employee_id)}" ${selected.has(e.employee_id) ? 'checked' : ''}>
        </label>
        <span class="export-row-name" title="${escapeHtml(filename)}">${escapeHtml(filename)}</span>
        <span class="export-row-meta">${joined}${exported}</span>
        <span class="badge">${e.photo ? 'Photo' : 'No photo'}</span>
      </div>`;
  }

  function renderVirtual() {
    const scrollTop = listEl.scrollTop;
    const viewHeight = listEl.clientHeight || 400;
    const total = totalHeight();

    let y = 0;
    let startIdx = 0;
    while (startIdx < flatRows.length && y + (flatRows[startIdx].type === 'header' ? HEADER_HEIGHT : ROW_HEIGHT) < scrollTop - BUFFER_PX) {
      y += flatRows[startIdx].type === 'header' ? HEADER_HEIGHT : ROW_HEIGHT;
      startIdx++;
    }
    const topHeight = y;
    let endIdx = startIdx;
    while (endIdx < flatRows.length && y < scrollTop + viewHeight + BUFFER_PX) {
      y += flatRows[endIdx].type === 'header' ? HEADER_HEIGHT : ROW_HEIGHT;
      endIdx++;
    }

    topSpacer.style.height = `${topHeight}px`;
    bottomSpacer.style.height = `${Math.max(0, total - y)}px`;
    rowsEl.innerHTML = flatRows.slice(startIdx, endIdx).map(rowHtml).join('');
  }

  function renderSummary() {
    summaryEl.textContent = `${selected.size} selected · ${shownIds.length} shown`;
    const anySelectedMissingPhoto = [...selected].some((id) => {
      const e = employees.find((emp) => emp.employee_id === id);
      return e && !e.photo;
    });
    noPhotoWarning.hidden = !anySelectedMissingPhoto;
    noPhotoWarning.textContent = 'Some selected cards have no photo; they will use the silhouette placeholder.';
    exportBtn.textContent = selected.size > 1 ? `Export ${selected.size} PDFs (ZIP)` : 'Export PDF';
    exportBtn.disabled = selected.size === 0;
  }

  function refresh() {
    rebuildRows();
    renderVirtual();
    renderSummary();
  }

  listEl.addEventListener('scroll', () => renderVirtual());

  rowsEl.addEventListener('click', (e) => {
    const rowCheckbox = e.target.closest('.row-checkbox');
    const groupCheckbox = e.target.closest('.group-checkbox');

    if (rowCheckbox) {
      const id = rowCheckbox.dataset.id;
      if (e.shiftKey && lastClickedId) {
        const ids = flatRows.filter((r) => r.type === 'row').map((r) => r.employee.employee_id);
        const from = ids.indexOf(lastClickedId);
        const to = ids.indexOf(id);
        if (from >= 0 && to >= 0) {
          const [lo, hi] = from < to ? [from, to] : [to, from];
          const shouldSelect = rowCheckbox.checked;
          for (let i = lo; i <= hi; i++) {
            if (shouldSelect) selected.add(ids[i]);
            else selected.delete(ids[i]);
          }
        }
      } else if (rowCheckbox.checked) {
        selected.add(id);
      } else {
        selected.delete(id);
      }
      lastClickedId = id;
      renderVirtual();
      renderSummary();
      return;
    }

    if (groupCheckbox) {
      const entry = flatRows.find((r) => r.type === 'header' && r.key === groupCheckbox.dataset.group);
      if (!entry) return;
      const shownInGroup = entry.ids.filter((id) => shownIds.includes(id));
      if (groupCheckbox.checked) shownInGroup.forEach((id) => selected.add(id));
      else shownInGroup.forEach((id) => selected.delete(id));
      renderVirtual();
      renderSummary();
    }
  });

  container.querySelector('#select-shown-btn').addEventListener('click', () => {
    shownIds.forEach((id) => selected.add(id));
    renderVirtual();
    renderSummary();
  });
  container.querySelector('#clear-selection-btn').addEventListener('click', () => {
    selected.clear();
    renderVirtual();
    renderSummary();
  });

  searchInput.addEventListener('input', () => {
    search = searchInput.value;
    refresh();
  });
  clientFilterSelect.addEventListener('change', () => {
    clientFilter = clientFilterSelect.value;
    refresh();
  });
  monthFilterSelect.addEventListener('change', () => {
    monthFilter = monthFilterSelect.value;
    refresh();
  });
  statusFilterSelect.addEventListener('change', () => {
    statusFilter = statusFilterSelect.value;
    refresh();
  });

  exportBtn.addEventListener('click', async () => {
    const ids = [...selected];
    if (!ids.length) return;
    exportError.textContent = '';
    exportBtn.disabled = true;
    progressBox.hidden = false;

    const blobs = [];
    try {
      for (let i = 0; i < ids.length; i++) {
        const employee = employees.find((e) => e.employee_id === ids[i]);
        progressLabel.textContent = `Rendering ${i + 1} / ${ids.length}…`;
        progressFill.style.width = `${(i / ids.length) * 100}%`;
        const client = clientFor(employee.client_code);
        const blob = await exportOne(employee, client);
        blobs.push({ blob, filename: exportFilename(employee.employee_id, employee.name) });
      }
      progressFill.style.width = '100%';
      await downloadResults(blobs);

      employees = await api.get('/employees');
      deps.onEmployeesChanged(employees);
      refresh();
    } catch (err) {
      exportError.textContent = err.message;
    } finally {
      progressBox.hidden = true;
      exportBtn.disabled = selected.size === 0;
    }
  });

  async function downloadResults(blobs) {
    let blob;
    let filename;
    if (blobs.length === 1) {
      blob = blobs[0].blob;
      filename = blobs[0].filename;
    } else {
      const zip = new window.JSZip();
      for (const b of blobs) zip.file(b.filename, b.blob);
      blob = await zip.generateAsync({ type: 'blob' });
      filename = 'id-cards.zip';
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  refresh();
}

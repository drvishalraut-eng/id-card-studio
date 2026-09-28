import { api } from './api.js';
import { escapeHtml } from './util.js';
import { resizeImageToJpeg } from './photo.js';

const SHORT_SIDE_WARNING_PX = 600;

function clamp(n, min, max) {
  return Math.min(max, Math.max(min, n));
}

export function renderStep3(container, deps) {
  const { preview } = deps;
  let clients = deps.clients;
  let employees = deps.employees;
  let selectedId = null;
  let search = '';
  let clientFilter = '';

  // Editing-session state, live only while a photo is staged but not yet
  // saved.
  let editing = null; // { blob, url, crop, warn }

  container.innerHTML = `
    <div class="step3">
      <p class="progress-label" id="photo-progress-label"></p>
      <div class="progress-bar"><div class="progress-fill" id="photo-progress-fill"></div></div>

      <div class="row-inline">
        <div class="field" style="flex:1;margin-top:0">
          <label for="ph-search">Search by name or ID</label>
          <input id="ph-search" placeholder="Search…">
        </div>
        <div class="field" style="flex:1;margin-top:0">
          <label for="ph-client-filter">Client</label>
          <select id="ph-client-filter"><option value="">All clients</option></select>
        </div>
      </div>

      <div id="current-photo-box"></div>

      <ul class="photo-queue" id="photo-queue"></ul>
    </div>
  `;

  const progressLabel = container.querySelector('#photo-progress-label');
  const progressFill = container.querySelector('#photo-progress-fill');
  const searchInput = container.querySelector('#ph-search');
  const clientFilterSelect = container.querySelector('#ph-client-filter');
  const box = container.querySelector('#current-photo-box');
  const queueList = container.querySelector('#photo-queue');

  clientFilterSelect.innerHTML =
    '<option value="">All clients</option>' +
    clients.map((c) => `<option value="${escapeHtml(c.code)}">${escapeHtml(c.name)}</option>`).join('');

  function stopEditing() {
    if (editing) URL.revokeObjectURL(editing.url);
    editing = null;
    preview.setDraftPhoto(null);
    preview.setPhotoDragHandler(null);
  }

  function filteredEmployees() {
    const q = search.trim().toLowerCase();
    return employees.filter((e) => {
      if (clientFilter && String(e.client_code || '').toLowerCase() !== clientFilter.toLowerCase()) return false;
      if (!q) return true;
      return e.name.toLowerCase().includes(q) || e.employee_id.toLowerCase().includes(q);
    });
  }

  function queue() {
    const filtered = filteredEmployees();
    const missing = filtered.filter((e) => !e.photo);
    const done = filtered.filter((e) => e.photo);
    return { missing, done, all: [...missing, ...done] };
  }

  function renderProgress() {
    const done = employees.filter((e) => e.photo).length;
    const total = employees.length;
    progressLabel.textContent = `${done} / ${total} photos done`;
    progressFill.style.width = total ? `${(done / total) * 100}%` : '0%';
  }

  function renderQueue() {
    const { missing, done } = queue();
    const row = (e, isDone) => `
      <li data-id="${escapeHtml(e.employee_id)}" class="${e.employee_id === selectedId ? 'selected' : ''}">
        <span>${escapeHtml(e.name)} (${escapeHtml(e.employee_id)})</span>
        <span class="badge">${isDone ? 'Done' : 'Missing'}</span>
      </li>`;
    queueList.innerHTML = missing.map((e) => row(e, false)).join('') + done.map((e) => row(e, true)).join('');
  }

  function selectedEmployee() {
    return employees.find((e) => e.employee_id === selectedId) || null;
  }

  function select(id) {
    stopEditing();
    selectedId = id;
    const index = employees.findIndex((e) => e.employee_id === id);
    if (index >= 0) preview.setIndex(index);
    renderQueue();
    renderBox();
  }

  // Auto-selects the first missing employee in the current filtered view
  // (falling back to the first employee at all) unless the current
  // selection is still visible.
  function ensureSelection() {
    const { all } = queue();
    if (selectedId && all.some((e) => e.employee_id === selectedId)) return;
    const missing = all.find((e) => !e.photo);
    select(missing ? missing.employee_id : all[0] ? all[0].employee_id : null);
  }

  function renderBox() {
    const e = selectedEmployee();
    if (!e) {
      box.innerHTML = '<p class="placeholder">No employees match.</p>';
      return;
    }

    if (!editing) {
      box.innerHTML = `
        <div class="current-employee">
          <p><strong>${escapeHtml(e.name)}</strong> · ${escapeHtml(e.employee_id)}</p>
          <p class="hint">${e.photo_note ? `Note: ${escapeHtml(e.photo_note)}` : ''}</p>
          <div class="btns">
            <button type="button" class="btn" id="upload-btn">${e.photo ? 'Replace photo' : 'Upload photo'}</button>
          </div>
          <input type="file" id="photo-file-input" accept="image/*" hidden>
        </div>
      `;
      box.querySelector('#upload-btn').addEventListener('click', () => {
        box.querySelector('#photo-file-input').click();
      });
      box.querySelector('#photo-file-input').addEventListener('change', onFileChosen);
      return;
    }

    box.innerHTML = `
      <div class="current-employee editing" tabindex="-1">
        <p><strong>${escapeHtml(e.name)}</strong> · ${escapeHtml(e.employee_id)}</p>
        ${editing.warn ? '<p class="warning-text">This photo is smaller than 600 px on its short side; it may look soft on the printed card.</p>' : ''}
        <div class="field"><label for="crop-zoom">Zoom</label>
          <input id="crop-zoom" type="range" min="1" max="2.5" step="0.05" value="${editing.crop.zoom}"></div>
        <div class="field"><label for="crop-x">Horizontal position</label>
          <input id="crop-x" type="range" min="0" max="100" value="${editing.crop.x}"></div>
        <div class="field"><label for="crop-y">Vertical position</label>
          <input id="crop-y" type="range" min="0" max="100" value="${editing.crop.y}"></div>
        <p class="error-text" id="save-photo-error"></p>
        <div class="btns">
          <button type="button" class="btn" id="save-next-btn">Save &amp; next</button>
          <button type="button" class="btn ghost" id="cancel-photo-btn">Cancel</button>
        </div>
      </div>
    `;

    const zoomInput = box.querySelector('#crop-zoom');
    const xInput = box.querySelector('#crop-x');
    const yInput = box.querySelector('#crop-y');
    const applyCrop = () => {
      editing.crop = { zoom: Number(zoomInput.value), x: Number(xInput.value), y: Number(yInput.value) };
      preview.setDraftPhoto(editing.url, editing.crop);
    };
    zoomInput.addEventListener('input', applyCrop);
    xInput.addEventListener('input', applyCrop);
    yInput.addEventListener('input', applyCrop);

    preview.setPhotoDragHandler((dxPercent, dyPercent) => {
      editing.crop.x = clamp(editing.crop.x - dxPercent, 0, 100);
      editing.crop.y = clamp(editing.crop.y - dyPercent, 0, 100);
      xInput.value = String(Math.round(editing.crop.x));
      yInput.value = String(Math.round(editing.crop.y));
      preview.setDraftPhoto(editing.url, editing.crop);
    });

    const editorBox = box.querySelector('.current-employee.editing');
    editorBox.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        saveAndNext();
      }
    });
    box.querySelector('#save-next-btn').addEventListener('click', saveAndNext);
    box.querySelector('#cancel-photo-btn').addEventListener('click', () => {
      stopEditing();
      renderBox();
    });
  }

  async function onFileChosen(ev) {
    const file = ev.target.files[0];
    ev.target.value = '';
    if (!file) return;

    let result;
    try {
      result = await resizeImageToJpeg(file);
    } catch (err) {
      box.querySelector('#photo-file-input').insertAdjacentHTML(
        'afterend',
        `<p class="error-text">${escapeHtml(err.message)}</p>`,
      );
      return;
    }

    editing = {
      blob: result.blob,
      url: URL.createObjectURL(result.blob),
      crop: { zoom: 1, x: 50, y: 50 },
      warn: Math.min(result.originalWidth, result.originalHeight) < SHORT_SIDE_WARNING_PX,
    };
    preview.setDraftPhoto(editing.url, editing.crop);
    renderBox();
  }

  async function saveAndNext() {
    const e = selectedEmployee();
    if (!e || !editing) return;
    const errorEl = box.querySelector('#save-photo-error');
    const form = new FormData();
    form.append('photo', editing.blob, 'photo.jpg');
    form.append('zoom', String(editing.crop.zoom));
    form.append('x', String(editing.crop.x));
    form.append('y', String(editing.crop.y));

    try {
      await api.post(`/employees/${encodeURIComponent(e.employee_id)}/photo`, form);
    } catch (err) {
      if (errorEl) errorEl.textContent = err.message;
      return;
    }

    stopEditing();
    employees = await api.get('/employees');
    deps.onEmployeesChanged(employees);
    renderProgress();
    selectedId = null; // force ensureSelection to pick the next missing one
    ensureSelection();
  }

  queueList.addEventListener('click', (e) => {
    const li = e.target.closest('li[data-id]');
    if (li) select(li.dataset.id);
  });

  searchInput.addEventListener('input', () => {
    search = searchInput.value;
    renderQueue();
    ensureSelection();
  });
  clientFilterSelect.addEventListener('change', () => {
    clientFilter = clientFilterSelect.value;
    renderQueue();
    ensureSelection();
  });

  renderProgress();
  renderQueue();
  ensureSelection();
}

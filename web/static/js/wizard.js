// The Wizard view: the 4-step left panel (520px, #F4F4F2) and the live
// preview on the right (#2E3238). Step content itself is built out in
// later tasks; this module owns the shared shell, data loading and the
// preview/navigation.
import { api } from './api.js';
import { createPreview } from './preview.js';
import { escapeHtml } from './util.js';

const STEPS = ['Client', 'Employees', 'Photos', 'Export'];

// currentPreview tracks the one live preview instance so it can be
// destroyed (see preview.js) before a new one replaces it.
let currentPreview = null;

export async function renderWizard(view) {
  if (currentPreview) {
    currentPreview.destroy();
    currentPreview = null;
  }

  view.innerHTML = `
    <aside class="wizard-panel">
      <ol class="step-bar">
        ${STEPS.map((label, i) => `<li class="step${i === 0 ? ' active' : ''}">${i + 1}. ${label}</li>`).join('')}
      </ol>
      <div class="step-content">
        <p class="placeholder">Step content coming soon.</p>
      </div>
    </aside>
    <section class="wizard-preview"></section>
  `;

  currentPreview = createPreview(view.querySelector('.wizard-preview'));

  try {
    const [clients, employeeList] = await Promise.all([api.get('/clients'), api.get('/employees')]);
    const clientByCode = new Map(clients.map((c) => [String(c.code).toLowerCase(), c]));
    const items = employeeList.map((employee) => ({
      employee,
      client: clientByCode.get(String(employee.client_code || '').toLowerCase()) || null,
    }));
    currentPreview.setItems(items);
  } catch (err) {
    view.querySelector('.wizard-preview').insertAdjacentHTML(
      'beforeend',
      `<p class="error-text">${escapeHtml(err.message)}</p>`,
    );
  }
}

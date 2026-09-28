// The Wizard view: the 4-step left panel (520px, #F4F4F2) and the live
// preview on the right (#2E3238). Steps 2-4 are still placeholders,
// built out in later tasks; this module owns the shared shell, data
// loading and the preview/navigation that every step shares.
import { api } from './api.js';
import { createPreview } from './preview.js';
import { escapeHtml } from './util.js';
import { renderStep1 } from './step1.js';

const STEPS = ['Client', 'Employees', 'Photos', 'Export'];

// currentPreview tracks the one live preview instance so it can be
// destroyed (see preview.js) before a new one replaces it.
let currentPreview = null;

export async function renderWizard(view, user) {
  if (currentPreview) {
    currentPreview.destroy();
    currentPreview = null;
  }

  view.innerHTML = `
    <aside class="wizard-panel">
      <ol class="step-bar">
        ${STEPS.map((label, i) => `<li class="step${i === 0 ? ' active' : ''}">${i + 1}. ${label}</li>`).join('')}
      </ol>
      <div class="step-content"></div>
    </aside>
    <section class="wizard-preview"></section>
  `;

  const stepContent = view.querySelector('.step-content');
  currentPreview = createPreview(view.querySelector('.wizard-preview'));

  let employeeList = [];
  let clients = [];
  try {
    [clients, employeeList] = await Promise.all([api.get('/clients'), api.get('/employees')]);
  } catch (err) {
    stepContent.innerHTML = `<p class="error-text">${escapeHtml(err.message)}</p>`;
    return;
  }

  function refreshPreviewItems() {
    const clientByCode = new Map(clients.map((c) => [String(c.code).toLowerCase(), c]));
    const items = employeeList.map((employee) => ({
      employee,
      client: clientByCode.get(String(employee.client_code || '').toLowerCase()) || null,
    }));
    currentPreview.setItems(items, true);
  }

  refreshPreviewItems();

  renderStep1(stepContent, {
    user,
    clients,
    preview: currentPreview,
    onClientsChanged(updatedClients) {
      clients = updatedClients;
      refreshPreviewItems();
    },
  });
}

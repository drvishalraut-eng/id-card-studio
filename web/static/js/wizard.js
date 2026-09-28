// The Wizard view: the 4-step left panel (520px, #F4F4F2) and the live
// preview on the right (#2E3238). This module owns the shared shell, step
// navigation, data loading and the preview that every step shares.
import { api } from './api.js';
import { createPreview } from './preview.js';
import { escapeHtml } from './util.js';
import { renderStep1 } from './step1.js';
import { renderStep2 } from './step2.js';
import { renderStep3 } from './step3.js';
import { renderStep4 } from './step4.js';

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
        ${STEPS.map((label, i) => `<li><button type="button" class="step-btn" data-step="${i}">${i + 1}. ${label}</button></li>`).join('')}
      </ol>
      <div class="step-content"></div>
    </aside>
    <section class="wizard-preview"></section>
  `;

  const stepBar = view.querySelector('.step-bar');
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

  let activeStep = 0;

  function renderActiveStep() {
    for (const btn of stepBar.querySelectorAll('.step-btn')) {
      btn.classList.toggle('active', Number(btn.dataset.step) === activeStep);
    }

    // Leaving Step 3 mid-edit (without going through its own Save/Cancel)
    // must not leave the preview stuck showing an unsaved draft photo or
    // still intercepting drags on every other step.
    if (activeStep !== 2) {
      currentPreview.setDraftPhoto(null);
      currentPreview.setPhotoDragHandler(null);
    }

    if (activeStep === 0) {
      renderStep1(stepContent, {
        user,
        clients,
        preview: currentPreview,
        onClientsChanged(updated) {
          clients = updated;
          refreshPreviewItems();
        },
      });
    } else if (activeStep === 1) {
      renderStep2(stepContent, {
        user,
        clients,
        employees: employeeList,
        preview: currentPreview,
        onEmployeesChanged(updated) {
          employeeList = updated;
          refreshPreviewItems();
        },
      });
    } else if (activeStep === 2) {
      renderStep3(stepContent, {
        clients,
        employees: employeeList,
        preview: currentPreview,
        onEmployeesChanged(updated) {
          employeeList = updated;
          refreshPreviewItems();
        },
      });
    } else {
      renderStep4(stepContent, {
        clients,
        employees: employeeList,
        onEmployeesChanged(updated) {
          employeeList = updated;
          refreshPreviewItems();
        },
      });
    }
  }

  stepBar.addEventListener('click', (e) => {
    const btn = e.target.closest('.step-btn');
    if (!btn) return;
    activeStep = Number(btn.dataset.step);
    renderActiveStep();
  });

  renderActiveStep();
}

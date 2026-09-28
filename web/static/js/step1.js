import { api } from './api.js';
import { escapeHtml } from './util.js';
import { validateSvg } from './svgvalidate.js';

// The exact prompt shown in the Add client form, per spec — copied
// verbatim so any change to it happens in one obviously-intentional place.
const CLAUDE_PROMPT = `I'm preparing a client logo for an ID card printing system. The attached image is the client's logo, and it may be a sheet showing several versions.
Create ONE SVG file for the card front. The card background is dark charcoal (#17202A).
Requirements:
1. Choose the lockup that has both the symbol and the company name and fits a 38 × 17 mm area (aspect ratio 1:1 to 4:1, stacked or horizontal).
2. The colours must read clearly on #17202A. Keep the brand colours if they have enough contrast; otherwise use white (#FFFFFF).
3. Convert all text to paths. Use no <text> elements and no font references. Trace the letterforms closely.
4. Include a viewBox, crop tightly, and leave out width and height attributes.
5. Use a transparent background, with no background rectangle.
6. Use no <image>, embedded bitmaps, external links, scripts, filters, masks or <style> blocks. Use hex fill attributes only.
7. Keep the file lean: merge paths where possible and round coordinates to 1 decimal.
Output the complete SVG in one code block, named <client-name>-logo-dark.svg. Then render a preview on a #17202A background.`;

// renderStep1 mounts the client dropdown, Add/Edit client form and Claude
// prompt into container. deps: { user, clients, preview, onClientsChanged }.
export function renderStep1(container, deps) {
  const { user, preview } = deps;
  let clients = deps.clients;
  let editingClient = null; // the client object being edited, or null in create mode
  let logoSvg = '';

  function clientOptionsHtml() {
    const options = clients
      .map((c) => `<option value="${escapeHtml(c.id)}">${escapeHtml(c.name)}</option>`)
      .join('');
    return `<option value="">Select a client…</option>${options}`;
  }

  container.innerHTML = `
    <div class="step1">
      <div class="row-inline">
        <div class="field" style="flex:1;margin-top:0">
          <label for="client-select">Client</label>
          <select id="client-select">${clientOptionsHtml()}</select>
        </div>
        ${user.role === 'admin' ? '<button type="button" class="btn ghost" id="add-client-btn">+ Add client</button>' : ''}
      </div>
      <div id="client-form-wrap" hidden></div>
    </div>
  `;

  const select = container.querySelector('#client-select');
  const formWrap = container.querySelector('#client-form-wrap');
  const addBtn = container.querySelector('#add-client-btn');

  select.addEventListener('change', () => {
    const id = select.value;
    if (!id) {
      preview.clearOverride();
      closeForm();
      return;
    }
    const client = clients.find((c) => c.id === id);
    preview.setOverride(client);
    if (user.role === 'admin') openForm(client);
  });

  if (addBtn) {
    addBtn.addEventListener('click', () => {
      select.value = '';
      preview.clearOverride();
      openForm(null);
    });
  }

  function closeForm() {
    formWrap.hidden = true;
    formWrap.innerHTML = '';
  }

  function openForm(client) {
    editingClient = client;
    logoSvg = client ? client.logo : '';
    formWrap.hidden = false;
    formWrap.innerHTML = formHtml(client);
    wireForm(client);
  }

  function formHtml(client) {
    const tagline = (client && client.tagline) || [];
    return `
      <h3>Prepare the logo with Claude</h3>
      <div class="field" style="margin-top:0">
        <label for="claude-prompt">Prompt for Claude (read-only)</label>
        <textarea id="claude-prompt" rows="10" readonly>${escapeHtml(CLAUDE_PROMPT)}</textarea>
      </div>
      <div class="btns"><button type="button" class="btn ghost" id="copy-prompt-btn">Copy prompt</button></div>

      <h3>Client details</h3>
      <form id="client-form" autocomplete="off" novalidate>
        <div class="field"><label for="cf-name">Client name</label>
          <input id="cf-name" name="name" value="${escapeHtml(client ? client.name : '')}" required></div>
        <div class="field"><label for="cf-code">Excel client code</label>
          <input id="cf-code" name="code" value="${escapeHtml(client ? client.code : '')}" required></div>
        <div class="field"><label for="cf-logo-file">Logo: upload an SVG file</label>
          <input id="cf-logo-file" type="file" accept=".svg,image/svg+xml"></div>
        <div class="field"><label for="cf-logo-paste">…or paste SVG code</label>
          <textarea id="cf-logo-paste" rows="4">${client ? escapeHtml(client.logo) : ''}</textarea></div>
        <p class="error-text" id="logo-error"></p>
        <div class="field"><label for="cf-tag1">Tagline line 1</label>
          <input id="cf-tag1" name="tag1" value="${escapeHtml(tagline[0] || '')}" required></div>
        <div class="field"><label for="cf-tag2">Tagline line 2</label>
          <input id="cf-tag2" name="tag2" value="${escapeHtml(tagline[1] || '')}" required></div>
        <div class="field"><label for="cf-tag3">Tagline line 3</label>
          <input id="cf-tag3" name="tag3" value="${escapeHtml(tagline[2] || '')}" required></div>
        <p class="error-text" id="client-form-error"></p>
        <div class="btns">
          <button type="submit" class="btn">Save client</button>
          <button type="button" class="btn ghost" id="cancel-client-btn">Cancel</button>
        </div>
      </form>
    `;
  }

  function draftClient(form) {
    const tagline = [form.tag1.value, form.tag2.value, form.tag3.value]
      .map((s) => s.trim().toUpperCase())
      .filter(Boolean);
    while (tagline.length < 3) tagline.push(`LINE ${tagline.length + 1}`);
    return {
      code: form.code.value.trim(),
      name: form.name.value.trim(),
      tagline,
      logo: logoSvg,
    };
  }

  function wireForm(client) {
    const form = formWrap.querySelector('#client-form');
    const logoError = formWrap.querySelector('#logo-error');
    const formError = formWrap.querySelector('#client-form-error');
    const fileInput = formWrap.querySelector('#cf-logo-file');
    const pasteArea = formWrap.querySelector('#cf-logo-paste');
    const copyBtn = formWrap.querySelector('#copy-prompt-btn');

    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(CLAUDE_PROMPT);
        copyBtn.textContent = 'Copied';
        setTimeout(() => {
          copyBtn.textContent = 'Copy prompt';
        }, 1500);
      } catch {
        formError.textContent = 'Could not copy automatically. Select the prompt text above and copy it manually.';
      }
    });

    function applyLogoText(text) {
      const err = validateSvg(text);
      logoError.textContent = err || '';
      if (err) return;
      logoSvg = text;
      preview.setOverride(draftClient(form));
    }

    fileInput.addEventListener('change', () => {
      const file = fileInput.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        const text = String(reader.result);
        pasteArea.value = text;
        applyLogoText(text);
      };
      reader.readAsText(file);
    });

    pasteArea.addEventListener('input', () => {
      if (pasteArea.value.trim()) applyLogoText(pasteArea.value);
    });

    form.addEventListener('input', (e) => {
      if (e.target === pasteArea) return; // handled by applyLogoText above
      preview.setOverride(draftClient(form));
    });

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      formError.textContent = '';

      if (!logoSvg) {
        formError.textContent = 'Add a logo (upload a file or paste SVG code) before saving.';
        return;
      }
      const svgErr = validateSvg(logoSvg);
      if (svgErr) {
        formError.textContent = svgErr;
        return;
      }

      const payload = {
        name: form.name.value.trim(),
        code: form.code.value.trim(),
        tagline: [form.tag1.value, form.tag2.value, form.tag3.value],
        logo: logoSvg,
      };

      try {
        const saved = editingClient
          ? await api.put(`/clients/${encodeURIComponent(editingClient.id)}`, payload)
          : await api.post('/clients', payload);
        clients = await api.get('/clients');
        select.innerHTML = clientOptionsHtml();
        select.value = saved.id;
        preview.setOverride(saved);
        closeForm();
        deps.onClientsChanged(clients);
      } catch (err) {
        formError.textContent = err.message;
      }
    });

    formWrap.querySelector('#cancel-client-btn').addEventListener('click', () => {
      closeForm();
      if (client) {
        select.value = client.id;
        preview.setOverride(client);
      } else {
        select.value = '';
        preview.clearOverride();
      }
    });
  }
}

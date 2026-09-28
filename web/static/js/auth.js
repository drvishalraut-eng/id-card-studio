import { api } from './api.js';

const BRAND_LOGO = '/assets/brand/wcf-logo-color.svg';

function renderAuthShell(bodyHtml) {
  return `
    <div class="auth-screen">
      <div class="auth-card">
        <img class="brand-logo" src="${BRAND_LOGO}" alt="White Coat Foundry">
        <h1>ID Card Studio</h1>
        ${bodyHtml}
      </div>
    </div>
  `;
}

// Renders the first-run "create the first admin" screen into root, calling
// onDone(user) once the account is created and signed in.
export function renderSetup(root, onDone) {
  root.innerHTML = renderAuthShell(`
    <form id="setup-form" autocomplete="off" novalidate>
      <p>Create the first admin account to get started.</p>
      <div class="field"><label for="su-name">Name</label><input id="su-name" name="name" required></div>
      <div class="field"><label for="su-username">Username</label><input id="su-username" name="username" required></div>
      <div class="field"><label for="su-pin">PIN (4-6 digits)</label>
        <input id="su-pin" name="pin" type="password" inputmode="numeric" pattern="[0-9]{4,6}" minlength="4" maxlength="6" required></div>
      <div class="field"><label for="su-confirm">Confirm PIN</label>
        <input id="su-confirm" name="confirm" type="password" inputmode="numeric" pattern="[0-9]{4,6}" minlength="4" maxlength="6" required></div>
      <p class="error-text" id="setup-error"></p>
      <button type="submit" class="btn">Create admin account</button>
    </form>
  `);

  const form = root.querySelector('#setup-form');
  const errorEl = root.querySelector('#setup-error');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    const name = form.name.value.trim();
    const username = form.username.value.trim();
    const pin = form.pin.value;
    const confirm = form.confirm.value;
    if (pin !== confirm) {
      errorEl.textContent = 'PIN and confirm PIN must match. Re-enter them and try again.';
      return;
    }
    try {
      const user = await api.post('/setup', { name, username, pin });
      onDone(user);
    } catch (err) {
      errorEl.textContent = err.message;
    }
  });
}

// Renders the sign-in screen into root, calling onDone(user) on success.
export function renderLogin(root, onDone) {
  root.innerHTML = renderAuthShell(`
    <form id="login-form" autocomplete="off" novalidate>
      <div class="field"><label for="li-username">Username</label><input id="li-username" name="username" required></div>
      <div class="field"><label for="li-pin">PIN</label>
        <input id="li-pin" name="pin" type="password" inputmode="numeric" required></div>
      <p class="error-text" id="login-error"></p>
      <button type="submit" class="btn">Sign in</button>
    </form>
  `);

  const form = root.querySelector('#login-form');
  const errorEl = root.querySelector('#login-error');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    try {
      const user = await api.post('/login', { username: form.username.value.trim(), pin: form.pin.value });
      onDone(user);
    } catch (err) {
      errorEl.textContent = err.message;
    }
  });
}

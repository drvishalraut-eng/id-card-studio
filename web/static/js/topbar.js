import { escapeHtml } from './util.js';

// Renders the charcoal top bar: brand mark, Wizard | Users | Activity tabs
// (Users only for an admin), the signed-in user's name and role, and Sign
// out.
export function renderTopbar(user, activePath) {
  const tabs = [{ path: '/wizard', label: 'Wizard' }];
  if (user.role === 'admin') tabs.push({ path: '/users', label: 'Users' });
  tabs.push({ path: '/activity', label: 'Activity' });

  const navHtml = tabs
    .map((t) => `<a href="#${t.path}" class="${t.path === activePath ? 'active' : ''}">${t.label}</a>`)
    .join('');

  return `
    <header class="topbar">
      <div class="brand">
        <img src="/assets/brand/wcf-logo-subtle-dark.svg" alt="">
        <span>ID Card Studio</span>
      </div>
      <nav>${navHtml}</nav>
      <div class="who">
        <div>${escapeHtml(user.name)}</div>
        <div class="role">${escapeHtml(user.role)}</div>
      </div>
      <button type="button" class="btn ghost" id="sign-out-btn">Sign out</button>
    </header>
  `;
}

// Wires the Sign out button inside root to call onSignOut.
export function wireTopbar(root, onSignOut) {
  const btn = root.querySelector('#sign-out-btn');
  if (btn) btn.addEventListener('click', onSignOut);
}

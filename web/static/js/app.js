import { api } from './api.js';
import { renderSetup, renderLogin } from './auth.js';
import { renderTopbar, wireTopbar } from './topbar.js';
import { startPresence, stopPresence, setStep } from './presence.js';
import { renderUsers } from './users.js';
import { renderActivity } from './activity.js';

const root = document.getElementById('app');
let currentUser = null;

function currentPath() {
  return location.hash.slice(1) || '/wizard';
}

function showAuth(render) {
  stopPresence();
  currentUser = null;
  render(root, (user) => {
    currentUser = user;
    startPresence();
    renderShell();
  });
}

async function signOut() {
  try {
    await api.post('/logout');
  } catch {
    // sign out locally regardless of whether the request reached the server
  }
  showAuth(renderLogin);
}

function renderShell() {
  if (!currentUser) return;
  const path = currentPath();

  if (path === '/users' && currentUser.role !== 'admin') {
    location.hash = '/wizard';
    return;
  }
  setStep(path.replace(/^\//, '') || 'wizard');

  root.innerHTML = `${renderTopbar(currentUser, path)}<div class="main" id="main-view"></div>`;
  wireTopbar(root, signOut);
  renderView(path, root.querySelector('#main-view'));
}

function renderView(path, view) {
  if (path === '/users') {
    renderUsers(view);
  } else if (path === '/activity') {
    renderActivity(view);
  } else {
    view.innerHTML = '<p class="placeholder">Wizard coming soon.</p>';
  }
}

window.addEventListener('hashchange', renderShell);

async function boot() {
  let status;
  try {
    status = await api.get('/setup');
  } catch (err) {
    root.innerHTML = `<p class="placeholder">${err.message}</p>`;
    return;
  }

  if (status.needed) {
    showAuth(renderSetup);
    return;
  }

  try {
    currentUser = await api.get('/me');
    startPresence();
    renderShell();
  } catch {
    showAuth(renderLogin);
  }
}

boot();

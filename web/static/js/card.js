// Renders the card front/back HTML, matching reference/index.html exactly.
//
// Every render call is fully self-contained (no shared <symbol>/<use> or
// document-wide <defs>, unlike the reference's own DOM technique): the
// export flow (task 18) renders many employees' cards off-screen at once,
// where a shared element id would collide across instances. Each call
// takes an instanceId (the employee_id is perfect: already unique and
// already restricted to CSS-id-safe characters) and namespaces every id it
// introduces with it.
import { escapeHtml } from './util.js';

export const CARD_WIDTH = 540;
export const CARD_HEIGHT = 856;
export const EXPORT_PIXEL_RATIO = 1276 / 540;

const FIXED_MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

// formatJoinDate renders "YYYY-MM-DD" as "DD Mon YYYY" using the fixed
// English month list, so the result is identical on every computer.
export function formatJoinDate(isoDate) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(isoDate || '');
  if (!m) return '';
  const [, y, mo, d] = m;
  const month = FIXED_MONTHS[Number(mo) - 1];
  if (!month) return isoDate;
  return `${d} ${month} ${y}`;
}

// namespaceIds rewrites every id="...", url(#...) and href="#..." in svg to
// end with -suffix, so multiple instances of the same markup never collide
// in one document. Safe for our own generated markup and for client logos,
// which are validated server-side to have only local #fragment hrefs.
function namespaceIds(svg, suffix) {
  return svg
    .replace(/\bid="([^"]+)"/g, (_, id) => `id="${id}-${suffix}"`)
    .replace(/url\(#([^)"]+)\)/g, (_, id) => `url(#${id}-${suffix})`)
    .replace(/href="#([^"]+)"/g, (_, id) => `href="#${id}-${suffix}"`);
}

const SILHOUETTE_SVG =
  '<svg viewBox="0 0 250 270"><rect width="250" height="270" fill="#d9dcdf"/>' +
  '<circle cx="125" cy="105" r="52" fill="#b9bec4"/>' +
  '<path d="M30 270c0-62 42-100 95-100s95 38 95 100z" fill="#b9bec4"/></svg>';

function isHelios(client) {
  return !!client && String(client.code || '').toLowerCase() === 'helios';
}

function brandHtml(client, instanceId) {
  if (!client) {
    return '<p class="dashed-logo">Client logo</p>';
  }
  const logo = client.logo ? namespaceIds(client.logo, instanceId) : '';
  if (isHelios(client)) {
    return `<div class="helios">${logo}<b>HELIOS MATERIAL</b><span>HANDLING</span></div>`;
  }
  return logo;
}

function taglineLines(client) {
  if (client && Array.isArray(client.tagline) && client.tagline.length === 3) {
    return client.tagline;
  }
  return ['LINE ONE', 'LINE TWO', 'LINE THREE'];
}

function photoHtml(employee) {
  if (employee && employee.photo) {
    const crop = employee.crop || { zoom: 1, x: 50, y: 50 };
    const src = `/api/employees/${encodeURIComponent(employee.employee_id)}/photo`;
    const style =
      `object-position:${crop.x}% ${crop.y}%;` +
      `transform:scale(${crop.zoom});transform-origin:${crop.x}% ${crop.y}%`;
    return `<img src="${src}" alt="" style="${style}">`;
  }
  return SILHOUETTE_SVG;
}

// renderFront returns the card-front HTML for employee (using client for
// the logo/tagline), or a blank placeholder card when employee is null —
// the state shown before any employee/client is selected.
export function renderFront(employee, client, instanceId) {
  const resolvedClient = employee ? client : null;
  const name = employee ? employee.name : '';
  const role = employee ? employee.role : '';
  const id = employee ? employee.employee_id : '';
  const joinDate = employee ? formatJoinDate(employee.join_date) : '';
  const tagline = taglineLines(resolvedClient);

  return `<div class="card front">
    <div class="slot"></div>
    <div class="brand">${brandHtml(resolvedClient, instanceId)}</div>
    <div class="photo">${photoHtml(employee)}</div>
    <h1>${escapeHtml(name)}</h1>
    <p class="role">${escapeHtml(role)}</p>
    <dl class="meta">
      <div><dt>EMPLOYEE ID</dt><dd>${escapeHtml(id)}</dd></div>
      <div><dt>JOIN DATE</dt><dd>${escapeHtml(joinDate)}</dd></div>
    </dl>
    <div class="stripe"></div>
    <div class="foot"><p>${tagline.map(escapeHtml).join('<br>')}</p></div>
  </div>`;
}

const PIN_ICON =
  '<svg viewBox="0 0 24 24"><path d="M12 2a7 7 0 0 0-7 7c0 5 7 13 7 13s7-8 7-13a7 7 0 0 0-7-7zm0 9.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/></svg>';
const PHONE_ICON =
  '<svg viewBox="0 0 24 24"><path d="M6.6 10.8a15 15 0 0 0 6.6 6.6l2.2-2.2a1 1 0 0 1 1-.25 11 11 0 0 0 3.6.6 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1 11 11 0 0 0 .6 3.6 1 1 0 0 1-.25 1z"/></svg>';
const MAIL_ICON =
  '<svg viewBox="0 0 24 24"><path d="M2 5h20v14H2zm2 2v.4l8 5.6 8-5.6V7zm0 2.8V17h16V9.8l-8 5.6z" fill-rule="evenodd"/></svg>';

const BACK_CONTENT = {
  tagline: 'PEOPLE&nbsp;&nbsp;|&nbsp;&nbsp;PLACEMENT&nbsp;&nbsp;|&nbsp;&nbsp;PROGRESS',
  address:
    'Krishna Complex, Plot No. 2,<br>Property No. 545/1,<br>Near Chambharli Naka, At Rees<br>' +
    '(Navin Vasahat), Post Mohopada,<br>Taluka Khalapur, District Raigad,<br>Maharashtra – 410222',
  phone: '+91&nbsp;7483831212',
  email: 'Dynamis@rautgroup.com',
  legal: [
    'This card remains the property of Dynamis and must be returned upon cessation of employment.',
    'If found, please return to the above address or contact +91&nbsp;7483831212.',
  ],
};

function dynamisMarkSvg(gradId) {
  return `<svg class="mark" viewBox="0 0 112 100">
    <path style="fill:var(--d)" d="M0 0H62A50 50 0 0 1 62 100H30L43 82H62A32 32 0 0 0 62 18H18Z"/>
    <path fill="url(#${gradId})" d="M14 28H36L60 52L26 100H4L38 52Z"/>
  </svg>`;
}

function dynamisASvg(gradId) {
  return `<svg viewBox="0 0 40 36">
    <path style="fill:var(--d)" d="M16 0H24L40 36H33L20 7L7 36H0Z"/>
    <path fill="url(#${gradId})" d="M20 22L26 36H14Z"/>
  </svg>`;
}

function gradientDefs(gradId) {
  return `<svg width="0" height="0" style="position:absolute" aria-hidden="true">
    <defs><linearGradient id="${gradId}" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#E9B23E"/><stop offset="1" stop-color="#B57A1C"/>
    </linearGradient></defs>
  </svg>`;
}

// renderBack returns the fixed Dynamis card-back HTML — identical for
// every employee/client.
export function renderBack(instanceId) {
  const gradId = `dg-${instanceId}`;
  return `<div class="card back">
    <div class="slot"></div>
    <div class="gold"></div>
    ${gradientDefs(gradId)}
    <div class="dynamis">
      ${dynamisMarkSvg(gradId)}
      <div class="word">DYN${dynamisASvg(gradId)}MIS</div>
    </div>
    <p class="tag">${BACK_CONTENT.tagline}</p>
    <div class="contact">
      ${PIN_ICON}<p class="addr">${BACK_CONTENT.address}</p>
      ${PHONE_ICON}<p>${BACK_CONTENT.phone}</p>
      ${MAIL_ICON}<p>${BACK_CONTENT.email}</p>
    </div>
    <hr>
    ${BACK_CONTENT.legal.map((t) => `<p class="legal">${t}</p>`).join('')}
  </div>`;
}

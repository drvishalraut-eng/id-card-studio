// Renders a card to a print-ready, 2-page PDF (front, back) using the
// vendored html-to-image + jsPDF, and uploads it to the server.
import { renderFront, renderBack, CARD_WIDTH, CARD_HEIGHT, EXPORT_PIXEL_RATIO } from './card.js';
import { api } from './api.js';

const PAGE_WIDTH_MM = 54;
const PAGE_HEIGHT_MM = 85.6;

// exportFilename builds "<employee_id>_<Name>.pdf", replacing every
// character outside [A-Za-z0-9_-] in Name with "_" — matches the server's
// own filename rule exactly (internal/export.Filename).
export function exportFilename(employeeId, name) {
  const safeName = String(name).replace(/[^A-Za-z0-9_-]/g, '_');
  return `${employeeId}_${safeName}.pdf`;
}

// renderSideCanvas renders one side's HTML off-screen at export
// resolution (no slot guide, square corners — the .exporting class) and
// returns a canvas. Waits for webfonts to be ready first, per spec.
async function renderSideCanvas(html) {
  const holder = document.createElement('div');
  holder.style.position = 'fixed';
  holder.style.left = '-10000px';
  holder.style.top = '0';
  holder.style.width = `${CARD_WIDTH}px`;
  holder.style.height = `${CARD_HEIGHT}px`;
  holder.innerHTML = html;
  const cardEl = holder.querySelector('.card');
  cardEl.classList.add('exporting');
  document.body.appendChild(holder);

  try {
    await document.fonts.ready;
    return await window.htmlToImage.toCanvas(cardEl, {
      width: CARD_WIDTH,
      height: CARD_HEIGHT,
      pixelRatio: EXPORT_PIXEL_RATIO,
      cacheBust: true,
      skipFonts: true,
    });
  } finally {
    document.body.removeChild(holder);
  }
}

// renderCardPdf renders employee/client to a jsPDF document: two pages,
// each exactly 54 x 85.6mm, the card image filling the page (no bleed).
export async function renderCardPdf(employee, client) {
  const frontCanvas = await renderSideCanvas(renderFront(employee, client, employee.employee_id));
  const backCanvas = await renderSideCanvas(renderBack(employee.employee_id));

  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ unit: 'mm', format: [PAGE_WIDTH_MM, PAGE_HEIGHT_MM], orientation: 'portrait' });
  doc.addImage(frontCanvas.toDataURL('image/jpeg', 0.92), 'JPEG', 0, 0, PAGE_WIDTH_MM, PAGE_HEIGHT_MM);
  doc.addPage([PAGE_WIDTH_MM, PAGE_HEIGHT_MM], 'portrait');
  doc.addImage(backCanvas.toDataURL('image/jpeg', 0.92), 'JPEG', 0, 0, PAGE_WIDTH_MM, PAGE_HEIGHT_MM);
  return doc;
}

// exportOne renders employee's card, uploads it to the server (recording
// the export), and returns the PDF as a Blob for the caller to also offer
// as a direct download or bundle into a ZIP.
export async function exportOne(employee, client) {
  const doc = await renderCardPdf(employee, client);
  const blob = doc.output('blob');

  const form = new FormData();
  form.append('employee_id', employee.employee_id);
  form.append('pdf', blob, exportFilename(employee.employee_id, employee.name));
  await api.post('/export', form);

  return blob;
}

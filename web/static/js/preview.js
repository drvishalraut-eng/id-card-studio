// The right-side live preview: front + back cards side by side, scaled to
// fit the panel, with ‹ › navigation and an "n / N · Name" counter.
import { renderFront, renderBack, CARD_WIDTH, CARD_HEIGHT } from './card.js';
import { escapeHtml } from './util.js';

const GAP = 24;
const PAIR_WIDTH = CARD_WIDTH * 2 + GAP;

// createPreview mounts into container and returns a small API for the
// owning view to push data into.
export function createPreview(container) {
  let items = []; // [{employee, client}]
  let index = 0;

  container.innerHTML = `
    <div class="preview-scale-wrap">
      <div class="preview-pair"></div>
    </div>
    <div class="preview-nav">
      <button type="button" class="btn ghost" id="preview-prev" aria-label="Previous card">‹</button>
      <span class="preview-counter" id="preview-counter"></span>
      <button type="button" class="btn ghost" id="preview-next" aria-label="Next card">›</button>
    </div>
  `;

  const wrap = container.querySelector('.preview-scale-wrap');
  const pair = container.querySelector('.preview-pair');
  const counter = container.querySelector('#preview-counter');
  const prevBtn = container.querySelector('#preview-prev');
  const nextBtn = container.querySelector('#preview-next');

  function current() {
    return items.length ? items[index] : null;
  }

  function fitScale() {
    const available = container.clientWidth - 48; // panel padding
    if (available <= 0) return 1;
    return Math.min(1, available / PAIR_WIDTH);
  }

  function render() {
    const cur = current();
    const employee = cur ? cur.employee : null;
    const client = cur ? cur.client : null;
    const instanceId = employee ? employee.employee_id : 'preview';

    pair.innerHTML = renderFront(employee, client, instanceId) + renderBack(instanceId);

    const scale = fitScale();
    wrap.style.width = `${PAIR_WIDTH * scale}px`;
    wrap.style.height = `${CARD_HEIGHT * scale}px`;
    pair.style.transform = `scale(${scale})`;

    counter.textContent = employee
      ? `${index + 1} / ${items.length} · ${escapeHtml(employee.name)}`
      : '0 / 0';
    const canNav = items.length > 1;
    prevBtn.disabled = !canNav;
    nextBtn.disabled = !canNav;
  }

  prevBtn.addEventListener('click', () => {
    if (!items.length) return;
    index = (index - 1 + items.length) % items.length;
    render();
  });
  nextBtn.addEventListener('click', () => {
    if (!items.length) return;
    index = (index + 1) % items.length;
    render();
  });
  const onResize = () => render();
  window.addEventListener('resize', onResize);

  render();

  return {
    // setItems replaces the full list; index defaults to the first item
    // (or is preserved, clamped, if keepIndex is true).
    setItems(newItems, keepIndex) {
      items = newItems;
      if (!keepIndex || index >= items.length) index = 0;
      render();
    },
    setIndex(i) {
      if (!items.length) return;
      index = ((i % items.length) + items.length) % items.length;
      render();
    },
    currentIndex: () => index,
    // destroy removes the window resize listener; callers must invoke this
    // before discarding a preview instance (e.g. navigating away and back),
    // or it keeps running render() against detached DOM forever.
    destroy() {
      window.removeEventListener('resize', onResize);
    },
  };
}

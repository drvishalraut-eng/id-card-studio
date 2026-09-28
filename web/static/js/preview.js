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
  let override = null; // { client } — Step 1's "preview this client's branding" mode
  let draftPhoto = null; // { src, crop } — Step 3's "photo being edited, not yet saved" for the CURRENT item
  let photoDragHandler = null; // (dxPercent, dyPercent) => void, set while Step 3 is editing a crop

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
  const nav = container.querySelector('.preview-nav');
  const counter = container.querySelector('#preview-counter');
  const prevBtn = container.querySelector('#preview-prev');
  const nextBtn = container.querySelector('#preview-next');

  function current() {
    return items.length ? items[index] : null;
  }

  // fitScale "contains" the pair within container: constrained by BOTH
  // width and height, not just width. CARD_HEIGHT (856) is nearly double
  // CARD_WIDTH (540), so on a typical landscape window the container's
  // height — not its width — is usually the binding constraint; a
  // width-only fit (the original bug here) makes the card taller than the
  // panel, pushing its lower portion and the nav row below the visible
  // area and out of sync with where clicks actually land.
  function fitScale() {
    const style = getComputedStyle(container);
    const paddingX = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
    const paddingY = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
    const gap = parseFloat(style.gap) || 0;
    const navHeight = nav.getBoundingClientRect().height || 44;

    const availableWidth = container.clientWidth - paddingX;
    const availableHeight = container.clientHeight - paddingY - gap - navHeight;
    if (availableWidth <= 0 || availableHeight <= 0) return 1;

    return Math.min(1, availableWidth / PAIR_WIDTH, availableHeight / CARD_HEIGHT);
  }

  function render() {
    let employee = null;
    let client = null;
    let instanceId = 'preview';
    let counterText = '0 / 0';
    let canNav = false;
    let photoSrcOverride = null;

    if (override) {
      client = override.client;
      counterText = client ? escapeHtml(client.name) : '0 / 0';
    } else {
      const cur = current();
      employee = cur ? cur.employee : null;
      client = cur ? cur.client : null;
      instanceId = employee ? employee.employee_id : 'preview';
      counterText = employee ? `${index + 1} / ${items.length} · ${escapeHtml(employee.name)}` : '0 / 0';
      canNav = items.length > 1;

      if (employee && draftPhoto) {
        employee = { ...employee, crop: draftPhoto.crop };
        photoSrcOverride = draftPhoto.src;
      }
    }

    pair.innerHTML = renderFront(employee, client, instanceId, photoSrcOverride) + renderBack(instanceId);

    const scale = fitScale();
    wrap.style.width = `${PAIR_WIDTH * scale}px`;
    wrap.style.height = `${CARD_HEIGHT * scale}px`;
    pair.style.transform = `scale(${scale})`;

    counter.textContent = counterText;
    prevBtn.disabled = !canNav;
    nextBtn.disabled = !canNav;
  }

  prevBtn.addEventListener('click', () => {
    if (override || !items.length) return;
    index = (index - 1 + items.length) % items.length;
    render();
  });
  nextBtn.addEventListener('click', () => {
    if (override || !items.length) return;
    index = (index + 1) % items.length;
    render();
  });
  const onResize = () => render();
  window.addEventListener('resize', onResize);

  // Dragging inside the preview's photo box adjusts the crop (Step 3).
  // Delegated on `pair` (which persists across renders, unlike its
  // innerHTML-replaced children) so this needs wiring only once; the
  // window-level listeners are also attached once and clean up in
  // destroy(), rather than being re-attached every render.
  let dragState = null;
  pair.addEventListener('mousedown', (e) => {
    if (!photoDragHandler) return;
    const photoEl = e.target.closest('.photo');
    if (!photoEl) return;
    const rect = photoEl.getBoundingClientRect();
    dragState = { lastX: e.clientX, lastY: e.clientY, rect };
    e.preventDefault();
  });
  const onMouseMove = (e) => {
    if (!dragState || !photoDragHandler) return;
    const dxPercent = ((e.clientX - dragState.lastX) / dragState.rect.width) * 100;
    const dyPercent = ((e.clientY - dragState.lastY) / dragState.rect.height) * 100;
    dragState.lastX = e.clientX;
    dragState.lastY = e.clientY;
    photoDragHandler(dxPercent, dyPercent);
  };
  const onMouseUp = () => {
    dragState = null;
  };
  window.addEventListener('mousemove', onMouseMove);
  window.addEventListener('mouseup', onMouseUp);

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
    // setOverride shows client's branding on a blank (no employee) card,
    // suspending normal navigation — Step 1's "preview while editing" mode.
    // Pass null (or call clearOverride) to return to the employee list.
    setOverride(client) {
      override = { client };
      render();
    },
    clearOverride() {
      override = null;
      render();
    },
    // setDraftPhoto shows src (an object URL for a not-yet-uploaded blob)
    // and crop on the CURRENT item only, without changing which employee
    // is selected — Step 3's "editing this photo" mode. Pass null to
    // revert to the employee's actual saved photo.
    setDraftPhoto(src, crop) {
      draftPhoto = src ? { src, crop } : null;
      render();
    },
    // setPhotoDragHandler registers fn(dxPercent, dyPercent) to be called
    // while the user drags inside the preview's photo box (Step 3's crop
    // editor); pass null to disable dragging again.
    setPhotoDragHandler(fn) {
      photoDragHandler = fn;
    },
    // destroy removes this preview's window-level listeners; callers must
    // invoke this before discarding a preview instance (e.g. navigating
    // away and back), or they keep running against detached DOM forever.
    destroy() {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    },
  };
}

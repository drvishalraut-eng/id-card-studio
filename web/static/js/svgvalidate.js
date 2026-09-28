// Client-side mirror of internal/clients/svg.go's ValidateSVG, so a bad
// logo is caught before upload rather than only after a round trip to the
// server (which re-validates regardless — this is a UX check, not the
// security boundary).
const DISALLOWED_ELEMENTS = new Set(['text', 'image', 'script', 'foreignobject']);

// validateSvg returns an error message if text is not an acceptable logo
// SVG, or null if it is.
export function validateSvg(text) {
  const upper = text.toUpperCase();
  if (upper.includes('<!DOCTYPE') || upper.includes('<!ENTITY')) {
    return 'SVG must not declare a DOCTYPE or ENTITY.';
  }

  let doc;
  try {
    doc = new DOMParser().parseFromString(text, 'image/svg+xml');
  } catch {
    return 'File is not valid SVG.';
  }
  if (doc.querySelector('parsererror')) {
    return 'File is not valid SVG.';
  }

  const root = doc.documentElement;
  if (!root || root.localName.toLowerCase() !== 'svg') {
    return 'File is not an SVG (root element is not <svg>).';
  }
  if (!root.hasAttribute('viewBox')) {
    return 'SVG must have a viewBox attribute.';
  }

  for (const el of doc.querySelectorAll('*')) {
    const name = el.localName.toLowerCase();
    if (DISALLOWED_ELEMENTS.has(name)) {
      return `SVG must not contain <${el.localName}>.`;
    }
    for (const attr of el.attributes) {
      const attrName = attr.name.toLowerCase();
      if (attrName.startsWith('on')) {
        return `SVG must not contain event-handler attributes (found "${attr.name}").`;
      }
      if ((attrName === 'href' || attrName.endsWith(':href')) && attr.value && !attr.value.startsWith('#')) {
        return 'SVG must not reference external files (found an href that is not a local #fragment).';
      }
    }
  }

  return null;
}

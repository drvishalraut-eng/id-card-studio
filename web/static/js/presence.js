// Sends the "Under the hood" panel's presence heartbeat every 15 seconds,
// per the spec. A missed beat is not worth surfacing to the user — the
// panel simply won't show this tab as connected until the next one lands.
import { api } from './api.js';

const INTERVAL_MS = 15000;

let timer = null;
let step = 'wizard';

export function setStep(nextStep) {
  step = nextStep;
}

async function beat() {
  try {
    await api.post('/presence', { step });
  } catch {
    // best-effort; see module comment
  }
}

export function startPresence() {
  if (timer) return;
  beat();
  timer = setInterval(beat, INTERVAL_MS);
}

export function stopPresence() {
  if (timer) clearInterval(timer);
  timer = null;
}

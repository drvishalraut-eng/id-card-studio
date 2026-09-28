// Thin wrapper around fetch() for the /api/* endpoints: always sends the
// required X-Requested-With header and the session cookie, always parses
// JSON, and throws an Error whose message is the server's own error text
// (so callers can show it directly to the user).

class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

async function request(method, path, body) {
  const headers = { 'X-Requested-With': 'idcard' };
  let payload = body;
  if (body !== undefined && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }

  let res;
  try {
    res = await fetch('/api' + path, {
      method,
      headers,
      body: payload,
      credentials: 'same-origin',
    });
  } catch (err) {
    throw new ApiError('Could not reach the server. Check your connection and try again.', 0);
  }

  const isJSON = (res.headers.get('Content-Type') || '').includes('application/json');
  const data = isJSON ? await res.json().catch(() => ({})) : {};

  if (!res.ok) {
    throw new ApiError(data.error || `Request failed (${res.status}).`, res.status);
  }
  return data;
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  delete: (path) => request('DELETE', path),
  ApiError,
};
